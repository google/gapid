#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""A low-level blob storage/retrieval interface to the Isolate server"""

import base64
import collections
import hashlib
import logging
import os
import re
import sys
import threading
import time
import types
import uuid

from utils import file_path
from utils import net

import isolated_format

try:
  import grpc # for error codes
  from utils import grpc_proxy
  from proto import bytestream_pb2
  # If not present, grpc crashes later.
  import pyasn1_modules
except ImportError as err:
  grpc = None
  grpc_proxy = None
  bytestream_pb2 = None


# Chunk size to use when reading from network stream.
NET_IO_FILE_CHUNK = 16 * 1024


# Read timeout in seconds for downloads from isolate storage. If there's no
# response from the server within this timeout whole download will be aborted.
DOWNLOAD_READ_TIMEOUT = 60


# Stores the gRPC proxy address. Must be set if the storage API class is
# IsolateServerGrpc (call 'set_grpc_proxy').
_grpc_proxy = None


class ServerRef(object):
  """ServerRef is a reference to the remote cache.

  This is expected to be an isolate server.
  """
  def __init__(self, url, namespace):
    """
    Args:
      url: URL of isolate service to use shared cloud based storage.
      namespace: isolate namespace to operate in, also defines hashing and
        compression scheme used, e.g. namespace names that end with '-gzip'
        store compressed data.
    """
    assert file_path.is_url(url) or not url, url
    self._url = url.rstrip('/')
    self._namespace = namespace
    self._hash_algo = hashlib.sha1
    self._hash_algo_name = 'sha-1'
    if self.namespace.startswith('sha256-'):
      self._hash_algo = hashlib.sha256
      self._hash_algo_name = 'sha-256'
    if self.namespace.startswith('sha512-'):
      self._hash_algo = hashlib.sha512
      self._hash_algo_name = 'sha-512'
    self._is_with_compression = self.namespace.endswith(('-gzip', '-deflate'))

  @property
  def url(self):
    return self._url

  @property
  def namespace(self):
    return self._namespace

  @property
  def hash_algo(self):
    """Hashing algorithm class to use when uploading to given |namespace|."""
    return self._hash_algo

  @property
  def hash_algo_name(self):
    return self._hash_algo_name

  @property
  def is_with_compression(self):
    """True if given |namespace| stores compressed objects.

    This means that this is the responsibility of the client to compress the
    data *before* uploading and decompress *after* downloading.
    """
    return self._is_with_compression


class Item(object):
  """An item to push to Storage.

  Its digest and size may be provided in advance, if known. Otherwise they will
  be derived from content(). If digest is provided, it MUST correspond to
  hash algorithm used by Storage.

  When used with Storage, Item starts its life in a main thread, travels
  to 'contains' thread, then to 'push' thread and then finally back to
  the main thread. It is never used concurrently from multiple threads.
  """

  def __init__(
      self, digest=None, size=None, high_priority=False,
      compression_level=6):
    self._digest = digest
    self._size = size
    self._high_priority = high_priority
    self._compression_level = compression_level

  @property
  def digest(self):
    assert self._digest
    return self._digest

  @property
  def size(self):
    return self._size

  @property
  def high_priority(self):
    return self._high_priority

  @property
  def compression_level(self):
    return self._compression_level

  def content(self):
    """Iterable with content of this item as byte string (str) chunks."""
    raise NotImplementedError()


class StorageApi(object):
  """Interface for classes that implement low-level storage operations.

  StorageApi is oblivious of compression and hashing scheme used. This details
  are handled in higher level Storage class.

  Clients should generally not use StorageApi directly. Storage class is
  preferred since it implements compression and upload optimizations.
  """

  @property
  def server_ref(self):
    """ServerRef instance."""
    raise NotImplementedError()

  @property
  def internal_compression(self):
    """True if this class doesn't require external compression.

    If true, callers should not compress items, even if the namespace indicates
    otherwise. Compression will be performed by the StorageApi class.
    """
    return False

  def fetch(self, digest, size, offset):
    """Fetches an object and yields its content.

    Arguments:
      digest: hash digest of item to download.
      size: size of the item to download if known, or None otherwise.
      offset: offset (in bytes) from the start of the file to resume fetch from.

    Yields:
      Chunks of downloaded item (as str objects).
    """
    raise NotImplementedError()

  def push(self, item, push_state, content=None):
    """Uploads an |item| with content generated by |content| generator.

    |item| MUST go through 'contains' call to get |push_state| before it can
    be pushed to the storage.

    To be clear, here is one possible usage:
      all_items = [... all items to push as Item subclasses ...]
      for missing_item, push_state in storage_api.contains(all_items).items():
        storage_api.push(missing_item, push_state)

    When pushing to a namespace with compression, data that should be pushed
    and data provided by the item is not the same. In that case |content| is
    not None and it yields chunks of compressed data (using item.content() as
    a source of original uncompressed data). This is implemented by Storage
    class.

    Arguments:
      item: Item object that holds information about an item being pushed.
      push_state: push state object as returned by 'contains' call.
      content: a generator that yields chunks to push, item.content() if None.

    Returns:
      None.
    """
    raise NotImplementedError()

  def contains(self, items):
    """Checks for |items| on the server, prepares missing ones for upload.

    Arguments:
      items: list of Item objects to check for presence.

    Returns:
      A dict missing Item -> opaque push state object to be passed to 'push'.
      See doc string for 'push'.
    """
    raise NotImplementedError()


class _IsolateServerPushState(object):
  """Per-item state passed from IsolateServer.contains to IsolateServer.push.

  Note this needs to be a global class to support pickling.
  """

  def __init__(self, preupload_status, size):
    self.preupload_status = preupload_status
    gs_upload_url = preupload_status.get('gs_upload_url') or None
    if gs_upload_url:
      self.upload_url = gs_upload_url
      self.finalize_url = '_ah/api/isolateservice/v1/finalize_gs_upload'
    else:
      self.upload_url = '_ah/api/isolateservice/v1/store_inline'
      self.finalize_url = None
    self.uploaded = False
    self.finalized = False
    self.size = size


def guard_memory_use(server, content, size):
  """Guards a server against using excessive memory while uploading.

  The server needs to contain a _memory_use int and a _lock mutex
  (both IsolateServer and IsolateServerGrpc qualify); this function
  then uses those values to track memory usage in a thread-safe way.

  If a request would cause the memory usage to exceed a safe maximum,
  this function sleeps in 0.1s increments until memory usage falls
  below the maximum.
  """
  if isinstance(content, (basestring, list)):
    # Memory is already used, too late.
    with server._lock:
      server._memory_use += size
  else:
    # TODO(vadimsh): Do not read from |content| generator when retrying push.
    # If |content| is indeed a generator, it can not be re-winded back to the
    # beginning of the stream. A retry will find it exhausted. A possible
    # solution is to wrap |content| generator with some sort of caching
    # restartable generator. It should be done alongside streaming support
    # implementation.
    #
    # In theory, we should keep the generator, so that it is not serialized in
    # memory. Sadly net.HttpService.request() requires the body to be
    # serialized.
    assert isinstance(content, types.GeneratorType), repr(content)
    slept = False
    # HACK HACK HACK. Please forgive me for my sins but OMG, it works!
    # One byte less than 512mb. This is to cope with incompressible content.
    max_size = int(sys.maxsize * 0.25)
    while True:
      with server._lock:
        # This is due to 32 bits python when uploading very large files. The
        # problem is that it's comparing uncompressed sizes, while we care
        # about compressed sizes since it's what is serialized in memory.
        # The first check assumes large files are compressible and that by
        # throttling one upload at once, we can survive. Otherwise, kaboom.
        memory_use = server._memory_use
        if ((size >= max_size and not memory_use) or
            (memory_use + size <= max_size)):
          server._memory_use += size
          memory_use = server._memory_use
          break
      time.sleep(0.1)
      slept = True
    if slept:
      logging.info('Unblocked: %d %d', memory_use, size)


class IsolateServer(StorageApi):
  """StorageApi implementation that downloads and uploads to Isolate Server.

  It uploads and downloads directly from Google Storage whenever appropriate.
  Works only within single namespace.
  """

  def __init__(self, server_ref):
    super(IsolateServer, self).__init__()
    assert isinstance(server_ref, ServerRef), repr(server_ref)
    self._server_ref = server_ref
    algo_name = server_ref.hash_algo_name
    self._namespace_dict = {
        'compression': 'flate' if server_ref.is_with_compression else '',
        'digest_hash': algo_name,
        'namespace': server_ref.namespace,
    }
    self._lock = threading.Lock()
    self._server_caps = None
    self._memory_use = 0

  @property
  def _server_capabilities(self):
    """Gets server details.

    Returns:
      Server capabilities dictionary as returned by /server_details endpoint.
    """
    # TODO(maruel): Make this request much earlier asynchronously while the
    # files are being enumerated.

    # TODO(vadimsh): Put |namespace| in the URL so that server can apply
    # namespace-level ACLs to this call.

    with self._lock:
      if self._server_caps is None:
        url = '%s/_ah/api/isolateservice/v1/server_details' % (
            self.server_ref.url)
        self._server_caps = net.url_read_json(url=url, data={})
      return self._server_caps

  @property
  def server_ref(self):
    """Returns the ServerRef instance that represents the remote server."""
    return self._server_ref

  def fetch(self, digest, _size, offset):
    assert offset >= 0
    source_url = '%s/_ah/api/isolateservice/v1/retrieve' % (
        self.server_ref.url)
    logging.debug('download_file(%s, %d)', source_url, offset)
    response = self._do_fetch(source_url, digest, offset)

    if not response:
      raise IOError(
          'Attempted to fetch from %s; no data exist: %s / %s.' % (
            source_url, self.server_ref.namespace, digest))

    # for DB uploads
    content = response.get('content')
    if content is not None:
      yield base64.b64decode(content)
      return

    if not response.get('url'):
      raise IOError(
          'Invalid response while fetching %s: %s' % (digest, response))

    # for GS entities
    connection = net.url_open(response['url'])
    if not connection:
      raise IOError(
          'Failed to download %s / %s' % (self.server_ref.namespace, digest))

    # If |offset|, verify server respects it by checking Content-Range.
    if offset:
      content_range = connection.get_header('Content-Range')
      if not content_range:
        raise IOError('Missing Content-Range header')

      # 'Content-Range' format is 'bytes <offset>-<last_byte_index>/<size>'.
      # According to a spec, <size> can be '*' meaning "Total size of the file
      # is not known in advance".
      try:
        match = re.match(r'bytes (\d+)-(\d+)/(\d+|\*)', content_range)
        if not match:
          raise ValueError()
        content_offset = int(match.group(1))
        last_byte_index = int(match.group(2))
        size = None if match.group(3) == '*' else int(match.group(3))
      except ValueError:
        raise IOError('Invalid Content-Range header: %s' % content_range)

      # Ensure returned offset equals requested one.
      if offset != content_offset:
        raise IOError('Expecting offset %d, got %d (Content-Range is %s)' % (
            offset, content_offset, content_range))

      # Ensure entire tail of the file is returned.
      if size is not None and last_byte_index + 1 != size:
        raise IOError('Incomplete response. Content-Range: %s' % content_range)

    for data in connection.iter_content(NET_IO_FILE_CHUNK):
      yield data

  def push(self, item, push_state, content=None):
    assert isinstance(item, Item)
    assert item.digest is not None
    assert item.size is not None
    assert isinstance(push_state, _IsolateServerPushState)
    assert not push_state.finalized

    # Default to item.content().
    content = item.content() if content is None else content
    guard_memory_use(self, content, push_state.size)

    try:
      # This push operation may be a retry after failed finalization call below,
      # no need to reupload contents in that case.
      if not push_state.uploaded:
        # PUT file to |upload_url|.
        success = self._do_push(push_state, content)
        if not success:
          raise IOError('Failed to upload file with hash %s to URL %s' % (
              item.digest, push_state.upload_url))
        push_state.uploaded = True
      else:
        logging.info(
            'A file %s already uploaded, retrying finalization only',
            item.digest)

      # Optionally notify the server that it's done.
      if push_state.finalize_url:
        # TODO(vadimsh): Calculate MD5 or CRC32C sum while uploading a file and
        # send it to isolated server. That way isolate server can verify that
        # the data safely reached Google Storage (GS provides MD5 and CRC32C of
        # stored files).
        # TODO(maruel): Fix the server to accept properly data={} so
        # url_read_json() can be used.
        response = net.url_read_json(
            url='%s/%s' % (self.server_ref.url, push_state.finalize_url),
            data={
                'upload_ticket': push_state.preupload_status['upload_ticket'],
            })
        if not response or not response.get('ok'):
          raise IOError(
              'Failed to finalize file with hash %s\n%r' %
              (item.digest, response))
      push_state.finalized = True
    finally:
      with self._lock:
        self._memory_use -= push_state.size

  def contains(self, items):
    # Ensure all items were initialized with 'prepare' call. Storage does that.
    assert all(i.digest is not None and i.size is not None for i in items)

    # Request body is a json encoded list of dicts.
    body = {
        'items': [
          {
            'digest': item.digest,
            'is_isolated': bool(item.high_priority),
            'size': item.size,
          } for item in items
        ],
        'namespace': self._namespace_dict,
    }

    query_url = '%s/_ah/api/isolateservice/v1/preupload' % self.server_ref.url

    # Response body is a list of push_urls (or null if file is already present).
    response = None
    try:
      response = net.url_read_json(url=query_url, data=body)
      if response is None:
        raise isolated_format.MappingError(
            'Failed to execute preupload query')
    except ValueError as err:
      raise isolated_format.MappingError(
          'Invalid response from server: %s, body is %s' % (err, response))

    # Pick Items that are missing, attach _PushState to them.
    missing_items = {}
    for preupload_status in response.get('items', []):
      assert 'upload_ticket' in preupload_status, (
          preupload_status, '/preupload did not generate an upload ticket')
      index = int(preupload_status['index'])
      missing_items[items[index]] = _IsolateServerPushState(
          preupload_status, items[index].size)
    logging.info('Queried %d files, %d cache hit',
        len(items), len(items) - len(missing_items))
    return missing_items

  def _do_fetch(self, url, digest, offset):
    """Fetches isolated data from the URL.

    Used only for fetching files, not for API calls. Can be overridden in
    subclasses.

    Args:
      url: URL to fetch the data from, can possibly return http redirect.
      offset: byte offset inside the file to start fetching from.

    Returns:
      net.HttpResponse compatible object, with 'read' and 'get_header' calls.
    """
    assert isinstance(offset, int)
    data = {
        'digest': digest.encode('utf-8'),
        'namespace': self._namespace_dict,
        'offset': offset,
    }
    # TODO(maruel): url + '?' + urllib.urlencode(data) once a HTTP GET endpoint
    # is added.
    return net.url_read_json(
        url=url,
        data=data,
        read_timeout=DOWNLOAD_READ_TIMEOUT)

  def _do_push(self, push_state, content):
    """Uploads isolated file to the URL.

    Used only for storing files, not for API calls. Can be overridden in
    subclasses.

    Args:
      url: URL to upload the data to.
      push_state: an _IsolateServicePushState instance
      item: the original Item to be uploaded
      content: an iterable that yields 'str' chunks.
    """
    # A cheezy way to avoid memcpy of (possibly huge) file, until streaming
    # upload support is implemented.
    if isinstance(content, list) and len(content) == 1:
      content = content[0]
    else:
      content = ''.join(content)

    # DB upload
    if not push_state.finalize_url:
      url = '%s/%s' % (self.server_ref.url, push_state.upload_url)
      content = base64.b64encode(content)
      data = {
          'upload_ticket': push_state.preupload_status['upload_ticket'],
          'content': content,
      }
      response = net.url_read_json(url=url, data=data)
      return response is not None and response['ok']

    # upload to GS
    url = push_state.upload_url
    response = net.url_read(
        content_type='application/octet-stream',
        data=content,
        method='PUT',
        headers={'Cache-Control': 'public, max-age=31536000'},
        url=url)
    return response is not None


class _IsolateServerGrpcPushState(object):
  """Empty class, just to present same interface as IsolateServer  """

  def __init__(self):
    pass


class IsolateServerGrpc(StorageApi):
  """StorageApi implementation that downloads and uploads to a gRPC service.

  Limitations: does not pass on namespace to the server (uses it only for hash
  algo and compression), and only allows zero offsets while fetching.
  """

  def __init__(self, server_ref, proxy):
    super(IsolateServerGrpc, self).__init__()
    logging.info(
        'Using gRPC for Isolate with server %s, proxy %s',
        server_ref.url, proxy)
    self._server_ref = server_ref
    self._lock = threading.Lock()
    self._memory_use = 0
    self._num_pushes = 0
    self._already_exists = 0
    self._proxy = grpc_proxy.Proxy(proxy, bytestream_pb2.ByteStreamStub)

  @property
  def internal_compression(self):
    # gRPC natively compresses all messages before transmission.
    return True

  @property
  def server_ref(self):
    """Returns the ServerRef instance that represents the remote server."""
    return self._server_ref

  def fetch(self, digest, size, offset):
    # The gRPC APIs only work with an offset of 0
    assert offset == 0
    request = bytestream_pb2.ReadRequest()
    if not size:
      size = -1
    request.resource_name = '%s/blobs/%s/%d' % (
        self._proxy.prefix, digest, size)
    try:
      for response in self._proxy.get_stream('Read', request):
        yield response.data
    except grpc.RpcError as g:
      logging.error('gRPC error during fetch: re-throwing as IOError (%s)' % g)
      raise IOError(g)

  def push(self, item, push_state, content=None):
    assert isinstance(item, Item)
    assert item.digest is not None
    assert item.size is not None
    assert isinstance(push_state, _IsolateServerGrpcPushState)

    # Default to item.content().
    content = item.content() if content is None else content
    guard_memory_use(self, content, item.size)
    self._num_pushes += 1

    try:
      def chunker():
        # Returns one bit of content at a time
        if (isinstance(content, str)
            or not isinstance(content, collections.Iterable)):
          yield content
        else:
          for chunk in content:
            yield chunk
      def slicer():
        # Ensures every bit of content is under the gRPC max size; yields
        # proto messages to send via gRPC.
        request = bytestream_pb2.WriteRequest()
        u = uuid.uuid4()
        request.resource_name = '%s/uploads/%s/blobs/%s/%d' % (
            self._proxy.prefix, u, item.digest, item.size)
        request.write_offset = 0
        for chunk in chunker():
          # Make sure we send at least one chunk for zero-length blobs
          has_sent_anything = False
          while chunk or not has_sent_anything:
            has_sent_anything = True
            slice_len = min(len(chunk), NET_IO_FILE_CHUNK)
            request.data = chunk[:slice_len]
            if request.write_offset + slice_len == item.size:
              request.finish_write = True
            yield request
            request.write_offset += slice_len
            chunk = chunk[slice_len:]

      response = None
      try:
        response = self._proxy.call_no_retries('Write', slicer())
      except grpc.RpcError as r:
        if r.code() == grpc.StatusCode.ALREADY_EXISTS:
          # This is legit - we didn't check before we pushed so no problem if
          # it's already there.
          self._already_exists += 1
          if self._already_exists % 100 == 0:
            logging.info('unnecessarily pushed %d/%d blobs (%.1f%%)' % (
                self._already_exists, self._num_pushes,
                100.0 * self._already_exists / self._num_pushes))
        else:
          logging.error('gRPC error during push: throwing as IOError (%s)' % r)
          raise IOError(r)
      except Exception as e:
        logging.error('error during push: throwing as IOError (%s)' % e)
        raise IOError(e)

      if response is not None and response.committed_size != item.size:
        raise IOError('%s/%d: incorrect size written (%d)' % (
            item.digest, item.size, response.committed_size))
      elif response is None and item.size > 0:
        # This happens when the content generator is exhausted and the gRPC call
        # simply returns None. Throw gRPC error as this is not recoverable.
        raise grpc.RpcError('None gRPC response on uploading %s' % item.digest)

    finally:
      with self._lock:
        self._memory_use -= item.size

  def contains(self, items):
    """Returns the set of all missing items."""
    # TODO(aludwin): this isn't supported directly in Bytestream, so for now
    # assume that nothing is present in the cache.
    # Ensure all items were initialized with 'prepare' call. Storage does that.
    assert all(i.digest is not None and i.size is not None for i in items)
    # Assume all Items are missing, and attach _PushState to them. The gRPC
    # implementation doesn't actually have a push state, we just attach empty
    # objects to satisfy the StorageApi interface.
    missing_items = {}
    for item in items:
      missing_items[item] = _IsolateServerGrpcPushState()
    return missing_items


def set_grpc_proxy(proxy):
  """Sets the StorageApi to use the specified proxy."""
  global _grpc_proxy
  assert _grpc_proxy is None
  _grpc_proxy = proxy


def get_storage_api(server_ref):
  """Returns an object that implements low-level StorageApi interface.

  It is used by Storage to work with single isolate |namespace|. It should
  rarely be used directly by clients, see 'get_storage' for
  a better alternative.

  Arguments:
    server_ref: ServerRef instance.

  Returns:
    Instance of StorageApi subclass.
  """
  assert isinstance(server_ref, ServerRef), repr(server_ref)
  if _grpc_proxy is not None:
    return IsolateServerGrpc(server_ref.url, _grpc_proxy)
  return IsolateServer(server_ref)
