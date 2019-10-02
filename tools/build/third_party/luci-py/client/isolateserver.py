#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Archives a set of files or directories to an Isolate Server."""

__version__ = '0.9.0'

import collections
import errno
import functools
import logging
import optparse
import os
import Queue
import re
import signal
import stat
import sys
import tarfile
import tempfile
import threading
import time
import zlib

from third_party import colorama
from third_party.depot_tools import fix_encoding
from third_party.depot_tools import subcommand

from utils import file_path
from utils import fs
from utils import logging_utils
from utils import net
from utils import on_error
from utils import subprocess42
from utils import threading_utils
from utils import tools

import auth
import isolated_format
import isolate_storage
import local_caching


# Version of isolate protocol passed to the server in /handshake request.
ISOLATE_PROTOCOL_VERSION = '1.0'


# Maximum expected delay (in seconds) between successive file fetches or uploads
# in Storage. If it takes longer than that, a deadlock might be happening
# and all stack frames for all threads are dumped to log.
DEADLOCK_TIMEOUT = 5 * 60


# The number of files to check the isolate server per /pre-upload query.
# All files are sorted by likelihood of a change in the file content
# (currently file size is used to estimate this: larger the file -> larger the
# possibility it has changed). Then first ITEMS_PER_CONTAINS_QUERIES[0] files
# are taken and send to '/pre-upload', then next ITEMS_PER_CONTAINS_QUERIES[1],
# and so on. Numbers here is a trade-off; the more per request, the lower the
# effect of HTTP round trip latency and TCP-level chattiness. On the other hand,
# larger values cause longer lookups, increasing the initial latency to start
# uploading, which is especially an issue for large files. This value is
# optimized for the "few thousands files to look up with minimal number of large
# files missing" case.
ITEMS_PER_CONTAINS_QUERIES = (20, 20, 50, 50, 50, 100)


# A list of already compressed extension types that should not receive any
# compression before being uploaded.
ALREADY_COMPRESSED_TYPES = [
    '7z', 'avi', 'cur', 'gif', 'h264', 'jar', 'jpeg', 'jpg', 'mp4', 'pdf',
    'png', 'wav', 'zip',
]


# The delay (in seconds) to wait between logging statements when retrieving
# the required files. This is intended to let the user (or buildbot) know that
# the program is still running.
DELAY_BETWEEN_UPDATES_IN_SECS = 30


DEFAULT_BLACKLIST = (
  # Temporary vim or python files.
  r'^.+\.(?:pyc|swp)$',
  # .git or .svn directory.
  r'^(?:.+' + re.escape(os.path.sep) + r'|)\.(?:git|svn)$',
)


class Error(Exception):
  """Generic runtime error."""
  pass


class Aborted(Error):
  """Operation aborted."""
  pass


class AlreadyExists(Error):
  """File already exists."""


def file_read(path, chunk_size=isolated_format.DISK_FILE_CHUNK, offset=0):
  """Yields file content in chunks of |chunk_size| starting from |offset|."""
  with fs.open(path, 'rb') as f:
    if offset:
      f.seek(offset)
    while True:
      data = f.read(chunk_size)
      if not data:
        break
      yield data


def fileobj_path(fileobj):
  """Return file system path for file like object or None.

  The returned path is guaranteed to exist and can be passed to file system
  operations like copy.
  """
  name = getattr(fileobj, 'name', None)
  if name is None:
    return

  # If the file like object was created using something like open("test.txt")
  # name will end up being a str (such as a function outside our control, like
  # the standard library). We want all our paths to be unicode objects, so we
  # decode it.
  if not isinstance(name, unicode):
    # We incorrectly assume that UTF-8 is used everywhere.
    name = name.decode('utf-8')

  # fs.exists requires an absolute path, otherwise it will fail with an
  # assertion error.
  if not os.path.isabs(name):
      return

  if fs.exists(name):
    return name


# TODO(tansell): Replace fileobj_copy with shutil.copyfileobj once proper file
# wrappers have been created.
def fileobj_copy(
    dstfileobj, srcfileobj, size=-1,
    chunk_size=isolated_format.DISK_FILE_CHUNK):
  """Copy data from srcfileobj to dstfileobj.

  Providing size means exactly that amount of data will be copied (if there
  isn't enough data, an IOError exception is thrown). Otherwise all data until
  the EOF marker will be copied.
  """
  if size == -1 and hasattr(srcfileobj, 'tell'):
    if srcfileobj.tell() != 0:
      raise IOError('partial file but not using size')

  written = 0
  while written != size:
    readsize = chunk_size
    if size > 0:
      readsize = min(readsize, size-written)
    data = srcfileobj.read(readsize)
    if not data:
      if size == -1:
        break
      raise IOError('partial file, got %s, wanted %s' % (written, size))
    dstfileobj.write(data)
    written += len(data)


def putfile(srcfileobj, dstpath, file_mode=None, size=-1, use_symlink=False):
  """Put srcfileobj at the given dstpath with given mode.

  The function aims to do this as efficiently as possible while still allowing
  any possible file like object be given.

  Creating a tree of hardlinks has a few drawbacks:
  - tmpfs cannot be used for the scratch space. The tree has to be on the same
    partition as the cache.
  - involves a write to the inode, which advances ctime, cause a metadata
    writeback (causing disk seeking).
  - cache ctime cannot be used to detect modifications / corruption.
  - Some file systems (NTFS) have a 64k limit on the number of hardlink per
    partition. This is why the function automatically fallbacks to copying the
    file content.
  - /proc/sys/fs/protected_hardlinks causes an additional check to ensure the
    same owner is for all hardlinks.
  - Anecdotal report that ext2 is known to be potentially faulty on high rate
    of hardlink creation.

  Creating a tree of symlinks has a few drawbacks:
  - Tasks running the equivalent of os.path.realpath() will get the naked path
    and may fail.
  - Windows:
    - Symlinks are reparse points:
      https://msdn.microsoft.com/library/windows/desktop/aa365460.aspx
      https://msdn.microsoft.com/library/windows/desktop/aa363940.aspx
    - Symbolic links are Win32 paths, not NT paths.
      https://googleprojectzero.blogspot.com/2016/02/the-definitive-guide-on-win32-to-nt.html
    - Symbolic links are supported on Windows 7 and later only.
    - SeCreateSymbolicLinkPrivilege is needed, which is not present by
      default.
    - SeCreateSymbolicLinkPrivilege is *stripped off* by UAC when a restricted
      RID is present in the token;
      https://msdn.microsoft.com/en-us/library/bb530410.aspx
  """
  srcpath = fileobj_path(srcfileobj)
  if srcpath and size == -1:
    readonly = file_mode is None or (
        file_mode & (stat.S_IWUSR | stat.S_IWGRP | stat.S_IWOTH))

    if readonly:
      # If the file is read only we can link the file
      if use_symlink:
        link_mode = file_path.SYMLINK_WITH_FALLBACK
      else:
        link_mode = file_path.HARDLINK_WITH_FALLBACK
    else:
      # If not read only, we must copy the file
      link_mode = file_path.COPY

    file_path.link_file(dstpath, srcpath, link_mode)
  else:
    # Need to write out the file
    with fs.open(dstpath, 'wb') as dstfileobj:
      fileobj_copy(dstfileobj, srcfileobj, size)

  assert fs.exists(dstpath)

  # file_mode of 0 is actually valid, so need explicit check.
  if file_mode is not None:
    fs.chmod(dstpath, file_mode)


def zip_compress(content_generator, level=7):
  """Reads chunks from |content_generator| and yields zip compressed chunks."""
  compressor = zlib.compressobj(level)
  for chunk in content_generator:
    compressed = compressor.compress(chunk)
    if compressed:
      yield compressed
  tail = compressor.flush(zlib.Z_FINISH)
  if tail:
    yield tail


def zip_decompress(
    content_generator, chunk_size=isolated_format.DISK_FILE_CHUNK):
  """Reads zipped data from |content_generator| and yields decompressed data.

  Decompresses data in small chunks (no larger than |chunk_size|) so that
  zip bomb file doesn't cause zlib to preallocate huge amount of memory.

  Raises IOError if data is corrupted or incomplete.
  """
  decompressor = zlib.decompressobj()
  compressed_size = 0
  try:
    for chunk in content_generator:
      compressed_size += len(chunk)
      data = decompressor.decompress(chunk, chunk_size)
      if data:
        yield data
      while decompressor.unconsumed_tail:
        data = decompressor.decompress(decompressor.unconsumed_tail, chunk_size)
        if data:
          yield data
    tail = decompressor.flush()
    if tail:
      yield tail
  except zlib.error as e:
    raise IOError(
        'Corrupted zip stream (read %d bytes) - %s' % (compressed_size, e))
  # Ensure all data was read and decompressed.
  if decompressor.unused_data or decompressor.unconsumed_tail:
    raise IOError('Not all data was decompressed')


def _get_zip_compression_level(filename):
  """Given a filename calculates the ideal zip compression level to use."""
  file_ext = os.path.splitext(filename)[1].lower()
  # TODO(csharp): Profile to find what compression level works best.
  return 0 if file_ext in ALREADY_COMPRESSED_TYPES else 7


def create_directories(base_directory, files):
  """Creates the directory structure needed by the given list of files."""
  logging.debug('create_directories(%s, %d)', base_directory, len(files))
  # Creates the tree of directories to create.
  directories = set(os.path.dirname(f) for f in files)
  for item in list(directories):
    while item:
      directories.add(item)
      item = os.path.dirname(item)
  for d in sorted(directories):
    if d:
      abs_d = os.path.join(base_directory, d)
      if not fs.isdir(abs_d):
        fs.mkdir(abs_d)


def _create_symlinks(base_directory, files):
  """Creates any symlinks needed by the given set of files."""
  for filepath, properties in files:
    if 'l' not in properties:
      continue
    if sys.platform == 'win32':
      # TODO(maruel): Create symlink via the win32 api.
      logging.warning('Ignoring symlink %s', filepath)
      continue
    outfile = os.path.join(base_directory, filepath)
    try:
      os.symlink(properties['l'], outfile)  # pylint: disable=E1101
    except OSError as e:
      if e.errno == errno.EEXIST:
        raise AlreadyExists('File %s already exists.' % outfile)
      raise


class _ThreadFile(object):
  """Multithreaded fake file. Used by TarBundle."""
  def __init__(self):
    self._data = threading_utils.TaskChannel()
    self._offset = 0

  def __iter__(self):
    return self._data

  def tell(self):
    return self._offset

  def write(self, b):
    self._data.send_result(b)
    self._offset += len(b)

  def close(self):
    self._data.send_done()


class FileItem(isolate_storage.Item):
  """A file to push to Storage.

  Its digest and size may be provided in advance, if known. Otherwise they will
  be derived from the file content.
  """

  def __init__(self, path, algo, digest=None, size=None, high_priority=False):
    super(FileItem, self).__init__(
        digest,
        size if size is not None else fs.stat(path).st_size,
        high_priority,
        compression_level=_get_zip_compression_level(path))
    self._path = path
    self._algo = algo
    self._meta = None

  @property
  def path(self):
    return self._path

  @property
  def digest(self):
    if not self._digest:
      self._digest = isolated_format.hash_file(self._path, self._algo)
    return self._digest

  @property
  def meta(self):
    if not self._meta:
      # TODO(maruel): Inline.
      self._meta = isolated_format.file_to_metadata(self.path, 0, False)
      # We need to hash right away.
      self._meta['h'] = self.digest
    return self._meta

  def content(self):
    return file_read(self.path)


class TarBundle(isolate_storage.Item):
  """Tarfile to push to Storage.

  Its digest is the digest of all the files it contains. It is generated on the
  fly.
  """

  def __init__(self, root, algo):
    # 2 trailing 512 bytes headers.
    super(TarBundle, self).__init__(size=1024)
    self._items = []
    self._meta = None
    self._algo = algo
    self._root_len = len(root) + 1
    # Same value as for Go.
    # https://chromium.googlesource.com/infra/luci/luci-go.git/+/master/client/archiver/tar_archiver.go
    # https://chromium.googlesource.com/infra/luci/luci-go.git/+/master/client/archiver/upload_tracker.go
    self._archive_max_size = int(10e6)

  @property
  def digest(self):
    if not self._digest:
      self._prepare()
    return self._digest

  @property
  def size(self):
    if self._size is None:
      self._prepare()
    return self._size

  def try_add(self, item):
    """Try to add this file to the bundle.

    It is extremely naive but this should be just enough for
    https://crbug.com/825418.

    Future improvements should be in the Go code, and the Swarming bot should be
    migrated to use the Go code instead.
    """
    if not item.size:
      return False
    # pylint: disable=unreachable
    rounded = (item.size + 512) & ~511
    if rounded + self._size > self._archive_max_size:
      return False
    # https://crbug.com/825418
    return False
    self._size += rounded
    self._items.append(item)
    return True

  def yield_item_path_meta(self):
    """Returns a tuple(Item, filepath, meta_dict).

    If the bundle contains less than 5 items, the items are yielded.
    """
    if len(self._items) < 5:
      # The tarball is too small, yield individual items, if any.
      for item in self._items:
        yield item, item.path[self._root_len:], item.meta
    else:
      # This ensures self._meta is set.
      p = self.digest + '.tar'
      # Yield itself as a tarball.
      yield self, p, self._meta

  def content(self):
    """Generates the tarfile content on the fly."""
    obj = _ThreadFile()
    def _tar_thread():
      try:
        t = tarfile.open(
            fileobj=obj, mode='w', format=tarfile.PAX_FORMAT, encoding='utf-8')
        for item in self._items:
          logging.info(' tarring %s', item.path)
          t.add(item.path)
        t.close()
      except Exception:
        logging.exception('Internal failure')
      finally:
        obj.close()

    t = threading.Thread(target=_tar_thread)
    t.start()
    try:
      for data in obj:
        yield data
    finally:
      t.join()

  def _prepare(self):
    h = self._algo()
    total = 0
    for chunk in self.content():
      h.update(chunk)
      total += len(chunk)
    # pylint: disable=attribute-defined-outside-init
    # This is not true, they are defined in Item.__init__().
    self._digest = h.hexdigest()
    self._size = total
    self._meta = {
      'h': self.digest,
      's': self.size,
      't': u'tar',
    }


class BufferItem(isolate_storage.Item):
  """A byte buffer to push to Storage."""

  def __init__(self, buf, algo, high_priority=False):
    super(BufferItem, self).__init__(
        digest=algo(buf).hexdigest(),
        size=len(buf),
        high_priority=high_priority)
    self._buffer = buf

  def content(self):
    return [self._buffer]


class Storage(object):
  """Efficiently downloads or uploads large set of files via StorageApi.

  Implements compression support, parallel 'contains' checks, parallel uploads
  and more.

  Works only within single namespace (and thus hashing algorithm and compression
  scheme are fixed).

  Spawns multiple internal threads. Thread safe, but not fork safe. Modifies
  signal handlers table to handle Ctrl+C.
  """

  def __init__(self, storage_api):
    self._storage_api = storage_api
    self._cpu_thread_pool = None
    self._net_thread_pool = None
    self._aborted = False
    self._prev_sig_handlers = {}

  @property
  def server_ref(self):
    """Shortcut to get the server_ref from storage_api.

    This can be used to get the underlying hash_algo.
    """
    return self._storage_api.server_ref

  @property
  def cpu_thread_pool(self):
    """ThreadPool for CPU-bound tasks like zipping."""
    if self._cpu_thread_pool is None:
      threads = max(threading_utils.num_processors(), 2)
      if sys.maxsize <= 2L**32:
        # On 32 bits userland, do not try to use more than 16 threads.
        threads = min(threads, 16)
      self._cpu_thread_pool = threading_utils.ThreadPool(2, threads, 0, 'zip')
    return self._cpu_thread_pool

  @property
  def net_thread_pool(self):
    """AutoRetryThreadPool for IO-bound tasks, retries IOError."""
    if self._net_thread_pool is None:
      self._net_thread_pool = threading_utils.IOAutoRetryThreadPool()
    return self._net_thread_pool

  def close(self):
    """Waits for all pending tasks to finish."""
    logging.info('Waiting for all threads to die...')
    if self._cpu_thread_pool:
      self._cpu_thread_pool.join()
      self._cpu_thread_pool.close()
      self._cpu_thread_pool = None
    if self._net_thread_pool:
      self._net_thread_pool.join()
      self._net_thread_pool.close()
      self._net_thread_pool = None
    logging.info('Done.')

  def abort(self):
    """Cancels any pending or future operations."""
    # This is not strictly theadsafe, but in the worst case the logging message
    # will be printed twice. Not a big deal. In other places it is assumed that
    # unprotected reads and writes to _aborted are serializable (it is true
    # for python) and thus no locking is used.
    if not self._aborted:
      logging.warning('Aborting... It can take a while.')
      self._aborted = True

  def __enter__(self):
    """Context manager interface."""
    assert not self._prev_sig_handlers, self._prev_sig_handlers
    for s in (signal.SIGINT, signal.SIGTERM):
      self._prev_sig_handlers[s] = signal.signal(s, lambda *_args: self.abort())
    return self

  def __exit__(self, _exc_type, _exc_value, _traceback):
    """Context manager interface."""
    self.close()
    while self._prev_sig_handlers:
      s, h = self._prev_sig_handlers.popitem()
      signal.signal(s, h)
    return False

  def upload_items(self, items):
    """Uploads a generator of Item to the isolate server.

    It figures out what items are missing from the server and uploads only them.

    It uses 3 threads internally:
    - One to create batches based on a timeout
    - One to dispatch the /contains RPC and field the missing entries
    - One to field the /push RPC

    The main threads enumerates 'items' and pushes to the first thread. Then it
    join() all the threads, waiting for them to complete.

        (enumerate items of Item, this can be slow as disk is traversed)
              |
              v
    _create_items_batches_thread       Thread #1
        (generates list(Item), every 3s or 20~100 items)
              |
              v
      _do_lookups_thread               Thread #2
         |         |
         v         v
     (missing)  (was on server)
         |
         v
    _handle_missing_thread             Thread #3
          |
          v
      (upload Item, append to uploaded)

    Arguments:
      items: list of isolate_storage.Item instances that represents data to
             upload.

    Returns:
      List of items that were uploaded. All other items are already there.
    """
    incoming = Queue.Queue()
    batches_to_lookup = Queue.Queue()
    missing = Queue.Queue()
    uploaded = []

    def _create_items_batches_thread():
      """Creates batches for /contains RPC lookup from individual items.

      Input: incoming
      Output: batches_to_lookup
      """
      try:
        batch_size_index = 0
        batch_size = ITEMS_PER_CONTAINS_QUERIES[batch_size_index]
        batch = []
        while not self._aborted:
          try:
            item = incoming.get(True, timeout=3)
            if item:
              batch.append(item)
          except Queue.Empty:
            item = False
          if len(batch) == batch_size or (not item and batch):
            if len(batch) == batch_size:
              batch_size_index += 1
              batch_size = ITEMS_PER_CONTAINS_QUERIES[
                  min(batch_size_index, len(ITEMS_PER_CONTAINS_QUERIES)-1)]
            batches_to_lookup.put(batch)
            batch = []
          if item is None:
            break
      finally:
        # Unblock the next pipeline.
        batches_to_lookup.put(None)

    def _do_lookups_thread():
      """Enqueues all the /contains RPCs and emits the missing items.

      Input: batches_to_lookup
      Output: missing, to_upload
      """
      try:
        channel = threading_utils.TaskChannel()
        def _contains(b):
          if self._aborted:
            raise Aborted()
          return self._storage_api.contains(b)

        pending_contains = 0
        while not self._aborted:
          batch = batches_to_lookup.get()
          if batch is None:
            break
          self.net_thread_pool.add_task_with_channel(
              channel, threading_utils.PRIORITY_HIGH, _contains, batch)
          pending_contains += 1
          while pending_contains and not self._aborted:
            try:
              v = channel.next(timeout=0)
            except threading_utils.TaskChannel.Timeout:
              break
            pending_contains -= 1
            for missing_item, push_state in v.iteritems():
              missing.put((missing_item, push_state))
        while pending_contains and not self._aborted:
          for missing_item, push_state in channel.next().iteritems():
            missing.put((missing_item, push_state))
          pending_contains -= 1
      finally:
        # Unblock the next pipeline.
        missing.put((None, None))

    def _handle_missing_thread():
      """Sends the missing items to the uploader.

      Input: missing
      Output: uploaded
      """
      with threading_utils.DeadlockDetector(DEADLOCK_TIMEOUT) as detector:
        channel = threading_utils.TaskChannel()
        pending_upload = 0
        while not self._aborted:
          try:
            missing_item, push_state = missing.get(True, timeout=5)
            if missing_item is None:
              break
            self._async_push(channel, missing_item, push_state)
            pending_upload += 1
          except Queue.Empty:
            pass
          detector.ping()
          while not self._aborted and pending_upload:
            try:
              item = channel.next(timeout=0)
            except threading_utils.TaskChannel.Timeout:
              break
            uploaded.append(item)
            pending_upload -= 1
            logging.debug(
                'Uploaded %d; %d pending: %s (%d)',
                len(uploaded), pending_upload, item.digest, item.size)
        while not self._aborted and pending_upload:
          item = channel.next()
          uploaded.append(item)
          pending_upload -= 1
          logging.debug(
              'Uploaded %d; %d pending: %s (%d)',
              len(uploaded), pending_upload, item.digest, item.size)

    threads = [
        threading.Thread(target=_create_items_batches_thread),
        threading.Thread(target=_do_lookups_thread),
        threading.Thread(target=_handle_missing_thread),
    ]
    for t in threads:
      t.start()

    try:
      # For each digest keep only first isolate_storage.Item that matches it.
      # All other items are just indistinguishable copies from the point of view
      # of isolate server (it doesn't care about paths at all, only content and
      # digests).
      seen = {}
      try:
        # TODO(maruel): Reorder the items as a priority queue, with larger items
        # being processed first. This is, before hashing the data.
        # This must be done in the primary thread since items can be a
        # generator.
        for item in items:
          if seen.setdefault(item.digest, item) is item:
            incoming.put(item)
      finally:
        incoming.put(None)
    finally:
      for t in threads:
        t.join()

    logging.info('All %s files are uploaded', len(uploaded))
    if seen:
      _print_upload_stats(seen.values(), uploaded)
    return uploaded

  def _async_push(self, channel, item, push_state):
    """Starts asynchronous push to the server in a parallel thread.

    Can be used only after |item| was checked for presence on a server with a
    /contains RPC.

    Arguments:
      channel: TaskChannel that receives back |item| when upload ends.
      item: item to upload as instance of isolate_storage.Item class.
      push_state: push state returned by storage_api.contains(). It contains
          storage specific information describing how to upload the item (for
          example in case of cloud storage, it is signed upload URLs).

    Returns:
      None, but |channel| later receives back |item| when upload ends.
    """
    # Thread pool task priority.
    priority = (
        threading_utils.PRIORITY_HIGH if item.high_priority
        else threading_utils.PRIORITY_MED)

    def _push(content):
      """Pushes an isolate_storage.Item and returns it to |channel|."""
      if self._aborted:
        raise Aborted()
      self._storage_api.push(item, push_state, content)
      return item

    # If zipping is not required, just start a push task. Don't pass 'content'
    # so that it can create a new generator when it retries on failures.
    if not self.server_ref.is_with_compression:
      self.net_thread_pool.add_task_with_channel(channel, priority, _push, None)
      return

    # If zipping is enabled, zip in a separate thread.
    def zip_and_push():
      # TODO(vadimsh): Implement streaming uploads. Before it's done, assemble
      # content right here. It will block until all file is zipped.
      try:
        if self._aborted:
          raise Aborted()
        stream = zip_compress(item.content(), item.compression_level)
        data = ''.join(stream)
      except Exception as exc:
        logging.error('Failed to zip \'%s\': %s', item, exc)
        channel.send_exception()
        return
      # Pass '[data]' explicitly because the compressed data is not same as the
      # one provided by 'item'. Since '[data]' is a list, it can safely be
      # reused during retries.
      self.net_thread_pool.add_task_with_channel(
          channel, priority, _push, [data])
    self.cpu_thread_pool.add_task(priority, zip_and_push)

  def push(self, item, push_state):
    """Synchronously pushes a single item to the server.

    If you need to push many items at once, consider using 'upload_items' or
    '_async_push' with instance of TaskChannel.

    Arguments:
      item: item to upload as instance of isolate_storage.Item class.
      push_state: push state returned by storage_api.contains(). It contains
          storage specific information describing how to upload the item (for
          example in case of cloud storage, it is signed upload URLs).

    Returns:
      Pushed item (same object as |item|).
    """
    channel = threading_utils.TaskChannel()
    with threading_utils.DeadlockDetector(DEADLOCK_TIMEOUT):
      self._async_push(channel, item, push_state)
      pushed = channel.next()
      assert pushed is item
    return item

  def async_fetch(self, channel, priority, digest, size, sink):
    """Starts asynchronous fetch from the server in a parallel thread.

    Arguments:
      channel: TaskChannel that receives back |digest| when download ends.
      priority: thread pool task priority for the fetch.
      digest: hex digest of an item to download.
      size: expected size of the item (after decompression).
      sink: function that will be called as sink(generator).
    """
    def fetch():
      try:
        # Prepare reading pipeline.
        stream = self._storage_api.fetch(digest, size, 0)
        if self.server_ref.is_with_compression:
          stream = zip_decompress(stream, isolated_format.DISK_FILE_CHUNK)
        # Run |stream| through verifier that will assert its size.
        verifier = FetchStreamVerifier(
            stream, self.server_ref.hash_algo, digest, size)
        # Verified stream goes to |sink|.
        sink(verifier.run())
      except Exception as err:
        logging.error('Failed to fetch %s: %s', digest, err)
        raise
      return digest

    # Don't bother with zip_thread_pool for decompression. Decompression is
    # really fast and most probably IO bound anyway.
    self.net_thread_pool.add_task_with_channel(channel, priority, fetch)


class FetchQueue(object):
  """Fetches items from Storage and places them into ContentAddressedCache.

  It manages multiple concurrent fetch operations. Acts as a bridge between
  Storage and ContentAddressedCache so that Storage and ContentAddressedCache
  don't depend on each other at all.
  """

  def __init__(self, storage, cache):
    self.storage = storage
    self.cache = cache
    self._channel = threading_utils.TaskChannel()
    self._pending = set()
    self._accessed = set()
    self._fetched = set(cache)
    # Pending digests that the caller waits for, see wait_on()/wait().
    self._waiting_on = set()
    # Already fetched digests the caller waits for which are not yet returned by
    # wait().
    self._waiting_on_ready = set()

  def add(
      self,
      digest,
      size=local_caching.UNKNOWN_FILE_SIZE,
      priority=threading_utils.PRIORITY_MED):
    """Starts asynchronous fetch of item |digest|."""
    # Fetching it now?
    if digest in self._pending:
      return

    # Mark this file as in use, verify_all_cached will later ensure it is still
    # in cache.
    self._accessed.add(digest)

    # Already fetched? Notify cache to update item's LRU position.
    if digest in self._fetched:
      # 'touch' returns True if item is in cache and not corrupted.
      if self.cache.touch(digest, size):
        return
      logging.error('%s is corrupted', digest)
      self._fetched.remove(digest)

    # TODO(maruel): It should look at the free disk space, the current cache
    # size and the size of the new item on every new item:
    # - Trim the cache as more entries are listed when free disk space is low,
    #   otherwise if the amount of data downloaded during the run > free disk
    #   space, it'll crash.
    # - Make sure there's enough free disk space to fit all dependencies of
    #   this run! If not, abort early.

    # Start fetching.
    self._pending.add(digest)
    self.storage.async_fetch(
        self._channel, priority, digest, size,
        functools.partial(self.cache.write, digest))

  def wait_on(self, digest):
    """Updates digests to be waited on by 'wait'."""
    # Calculate once the already fetched items. These will be retrieved first.
    if digest in self._fetched:
      self._waiting_on_ready.add(digest)
    else:
      self._waiting_on.add(digest)

  def wait(self):
    """Waits until any of waited-on items is retrieved.

    Once this happens, it is remove from the waited-on set and returned.

    This function is called in two waves. The first wave it is done for HIGH
    priority items, the isolated files themselves. The second wave it is called
    for all the files.

    If the waited-on set is empty, raises RuntimeError.
    """
    # Flush any already fetched items.
    if self._waiting_on_ready:
      return self._waiting_on_ready.pop()

    assert self._waiting_on, 'Needs items to wait on'

    # Wait for one waited-on item to be fetched.
    while self._pending:
      digest = self._channel.next()
      self._pending.remove(digest)
      self._fetched.add(digest)
      if digest in self._waiting_on:
        self._waiting_on.remove(digest)
        return digest

    # Should never reach this point due to assert above.
    raise RuntimeError('Impossible state')

  @property
  def wait_queue_empty(self):
    """Returns True if there is no digest left for wait() to return."""
    return not self._waiting_on and not self._waiting_on_ready

  def inject_local_file(self, path, algo):
    """Adds local file to the cache as if it was fetched from storage."""
    with fs.open(path, 'rb') as f:
      data = f.read()
    digest = algo(data).hexdigest()
    self.cache.write(digest, [data])
    self._fetched.add(digest)
    return digest

  @property
  def pending_count(self):
    """Returns number of items to be fetched."""
    return len(self._pending)

  def verify_all_cached(self):
    """True if all accessed items are in cache."""
    # Not thread safe, but called after all work is done.
    return self._accessed.issubset(self.cache)


class FetchStreamVerifier(object):
  """Verifies that fetched file is valid before passing it to the
  ContentAddressedCache.
  """

  def __init__(self, stream, hasher, expected_digest, expected_size):
    """Initializes the verifier.

    Arguments:
    * stream: an iterable yielding chunks of content
    * hasher: an object from hashlib that supports update() and hexdigest()
      (eg, hashlib.sha1).
    * expected_digest: if the entire stream is piped through hasher and then
      summarized via hexdigest(), this should be the result. That is, it
      should be a hex string like 'abc123'.
    * expected_size: either the expected size of the stream, or
      local_caching.UNKNOWN_FILE_SIZE.
    """
    assert stream is not None
    self.stream = stream
    self.expected_digest = expected_digest
    self.expected_size = expected_size
    self.current_size = 0
    self.rolling_hash = hasher()

  def run(self):
    """Generator that yields same items as |stream|.

    Verifies |stream| is complete before yielding a last chunk to consumer.

    Also wraps IOError produced by consumer into MappingError exceptions since
    otherwise Storage will retry fetch on unrelated local cache errors.
    """
    # Read one chunk ahead, keep it in |stored|.
    # That way a complete stream can be verified before pushing last chunk
    # to consumer.
    stored = None
    for chunk in self.stream:
      assert chunk is not None
      if stored is not None:
        self._inspect_chunk(stored, is_last=False)
        try:
          yield stored
        except IOError as exc:
          raise isolated_format.MappingError(
              'Failed to store an item in cache: %s' % exc)
      stored = chunk
    if stored is not None:
      self._inspect_chunk(stored, is_last=True)
      try:
        yield stored
      except IOError as exc:
        raise isolated_format.MappingError(
            'Failed to store an item in cache: %s' % exc)

  def _inspect_chunk(self, chunk, is_last):
    """Called for each fetched chunk before passing it to consumer."""
    self.current_size += len(chunk)
    self.rolling_hash.update(chunk)
    if not is_last:
      return

    if ((self.expected_size != local_caching.UNKNOWN_FILE_SIZE) and
        (self.expected_size != self.current_size)):
      msg = 'Incorrect file size: want %d, got %d' % (
          self.expected_size, self.current_size)
      raise IOError(msg)

    actual_digest = self.rolling_hash.hexdigest()
    if self.expected_digest != actual_digest:
      msg = 'Incorrect digest: want %s, got %s' % (
          self.expected_digest, actual_digest)
      raise IOError(msg)


class IsolatedBundle(object):
  """Fetched and parsed .isolated file with all dependencies."""

  def __init__(self, filter_cb):
    """
    filter_cb: callback function to filter downloaded content.
               When filter_cb is not None, Isolated file is downloaded iff
               filter_cb(filepath) returns True.
    """

    self.command = []
    self.files = {}
    self.read_only = None
    self.relative_cwd = None
    # The main .isolated file, a IsolatedFile instance.
    self.root = None

    self._filter_cb = filter_cb

  def fetch(self, fetch_queue, root_isolated_hash, algo):
    """Fetches the .isolated and all the included .isolated.

    It enables support for "included" .isolated files. They are processed in
    strict order but fetched asynchronously from the cache. This is important so
    that a file in an included .isolated file that is overridden by an embedding
    .isolated file is not fetched needlessly. The includes are fetched in one
    pass and the files are fetched as soon as all the ones on the left-side
    of the tree were fetched.

    The prioritization is very important here for nested .isolated files.
    'includes' have the highest priority and the algorithm is optimized for both
    deep and wide trees. A deep one is a long link of .isolated files referenced
    one at a time by one item in 'includes'. A wide one has a large number of
    'includes' in a single .isolated file. 'left' is defined as an included
    .isolated file earlier in the 'includes' list. So the order of the elements
    in 'includes' is important.

    As a side effect this method starts asynchronous fetch of all data files
    by adding them to |fetch_queue|. It doesn't wait for data files to finish
    fetching though.
    """
    self.root = isolated_format.IsolatedFile(root_isolated_hash, algo)

    # Isolated files being retrieved now: hash -> IsolatedFile instance.
    pending = {}
    # Set of hashes of already retrieved items to refuse recursive includes.
    seen = set()
    # Set of IsolatedFile's whose data files have already being fetched.
    processed = set()

    def retrieve_async(isolated_file):
      """Retrieves an isolated file included by the root bundle."""
      h = isolated_file.obj_hash
      if h in seen:
        raise isolated_format.IsolatedError(
            'IsolatedFile %s is retrieved recursively' % h)
      assert h not in pending
      seen.add(h)
      pending[h] = isolated_file
      # This isolated item is being added dynamically, notify FetchQueue.
      fetch_queue.wait_on(h)
      fetch_queue.add(h, priority=threading_utils.PRIORITY_HIGH)

    # Start fetching root *.isolated file (single file, not the whole bundle).
    retrieve_async(self.root)

    while pending:
      # Wait until some *.isolated file is fetched, parse it.
      item_hash = fetch_queue.wait()
      item = pending.pop(item_hash)
      with fetch_queue.cache.getfileobj(item_hash) as f:
        item.load(f.read())

      # Start fetching included *.isolated files.
      for new_child in item.children:
        retrieve_async(new_child)

      # Always fetch *.isolated files in traversal order, waiting if necessary
      # until next to-be-processed node loads. "Waiting" is done by yielding
      # back to the outer loop, that waits until some *.isolated is loaded.
      for node in isolated_format.walk_includes(self.root):
        if node not in processed:
          # Not visited, and not yet loaded -> wait for it to load.
          if not node.is_loaded:
            break
          # Not visited and loaded -> process it and continue the traversal.
          self._start_fetching_files(node, fetch_queue)
          processed.add(node)

    # All *.isolated files should be processed by now and only them.
    all_isolateds = set(isolated_format.walk_includes(self.root))
    assert all_isolateds == processed, (all_isolateds, processed)
    assert fetch_queue.wait_queue_empty, 'FetchQueue should have been emptied'

    # Extract 'command' and other bundle properties.
    for node in isolated_format.walk_includes(self.root):
      self._update_self(node)
    self.relative_cwd = self.relative_cwd or ''

  def _start_fetching_files(self, isolated, fetch_queue):
    """Starts fetching files from |isolated| that are not yet being fetched.

    Modifies self.files.
    """
    files = isolated.data.get('files', {})
    logging.debug('fetch_files(%s, %d)', isolated.obj_hash, len(files))
    for filepath, properties in files.iteritems():
      if self._filter_cb and not self._filter_cb(filepath):
        continue

      # Root isolated has priority on the files being mapped. In particular,
      # overridden files must not be fetched.
      if filepath not in self.files:
        self.files[filepath] = properties

        # Make sure if the isolated is read only, the mode doesn't have write
        # bits.
        if 'm' in properties and self.read_only:
          properties['m'] &= ~(stat.S_IWUSR | stat.S_IWGRP | stat.S_IWOTH)

        # Preemptively request hashed files.
        if 'h' in properties:
          fetch_queue.add(
              properties['h'], properties['s'], threading_utils.PRIORITY_MED)

  def _update_self(self, node):
    """Extracts bundle global parameters from loaded *.isolated file.

    Will be called with each loaded *.isolated file in order of traversal of
    isolated include graph (see isolated_format.walk_includes).
    """
    # Grabs properties.
    if not self.command and node.data.get('command'):
      # Ensure paths are correctly separated on windows.
      self.command = node.data['command']
      if self.command:
        self.command[0] = self.command[0].replace('/', os.path.sep)
    if self.read_only is None and node.data.get('read_only') is not None:
      self.read_only = node.data['read_only']
    if (self.relative_cwd is None and
        node.data.get('relative_cwd') is not None):
      self.relative_cwd = node.data['relative_cwd']


def get_storage(server_ref):
  """Returns Storage class that can upload and download from |namespace|.

  Arguments:
    server_ref: isolate_storage.ServerRef instance.

  Returns:
    Instance of Storage.
  """
  assert isinstance(server_ref, isolate_storage.ServerRef), repr(server_ref)
  return Storage(isolate_storage.get_storage_api(server_ref))


def fetch_isolated(isolated_hash, storage, cache, outdir, use_symlinks,
                   filter_cb=None):
  """Aggressively downloads the .isolated file(s), then download all the files.

  Arguments:
    isolated_hash: hash of the root *.isolated file.
    storage: Storage class that communicates with isolate storage.
    cache: ContentAddressedCache class that knows how to store and map files
           locally.
    outdir: Output directory to map file tree to.
    use_symlinks: Use symlinks instead of hardlinks when True.
    filter_cb: filter that works as whitelist for downloaded files.

  Returns:
    IsolatedBundle object that holds details about loaded *.isolated file.
  """
  logging.debug(
      'fetch_isolated(%s, %s, %s, %s, %s)',
      isolated_hash, storage, cache, outdir, use_symlinks)
  # Hash algorithm to use, defined by namespace |storage| is using.
  algo = storage.server_ref.hash_algo
  fetch_queue = FetchQueue(storage, cache)
  bundle = IsolatedBundle(filter_cb)

  with tools.Profiler('GetIsolateds'):
    # Optionally support local files by manually adding them to cache.
    if not isolated_format.is_valid_hash(isolated_hash, algo):
      logging.debug('%s is not a valid hash, assuming a file '
                    '(algo was %s, hash size was %d)',
                    isolated_hash, algo(), algo().digest_size)
      path = unicode(os.path.abspath(isolated_hash))
      try:
        isolated_hash = fetch_queue.inject_local_file(path, algo)
      except IOError as e:
        raise isolated_format.MappingError(
            '%s doesn\'t seem to be a valid file. Did you intent to pass a '
            'valid hash (error: %s)?' % (isolated_hash, e))

    # Load all *.isolated and start loading rest of the files.
    bundle.fetch(fetch_queue, isolated_hash, algo)

  with tools.Profiler('GetRest'):
    # Create file system hierarchy.
    file_path.ensure_tree(outdir)
    create_directories(outdir, bundle.files)
    _create_symlinks(outdir, bundle.files.iteritems())

    # Ensure working directory exists.
    cwd = os.path.normpath(os.path.join(outdir, bundle.relative_cwd))
    file_path.ensure_tree(cwd)

    # Multimap: digest -> list of pairs (path, props).
    remaining = {}
    for filepath, props in bundle.files.iteritems():
      if 'h' in props:
        remaining.setdefault(props['h'], []).append((filepath, props))
        fetch_queue.wait_on(props['h'])

    # Now block on the remaining files to be downloaded and mapped.
    logging.info('Retrieving remaining files (%d of them)...',
        fetch_queue.pending_count)
    last_update = time.time()
    with threading_utils.DeadlockDetector(DEADLOCK_TIMEOUT) as detector:
      while remaining:
        detector.ping()

        # Wait for any item to finish fetching to cache.
        digest = fetch_queue.wait()

        # Create the files in the destination using item in cache as the
        # source.
        for filepath, props in remaining.pop(digest):
          fullpath = os.path.join(outdir, filepath)

          with cache.getfileobj(digest) as srcfileobj:
            filetype = props.get('t', 'basic')

            if filetype == 'basic':
              # Ignore all bits apart from the user.
              file_mode = (props.get('m') or 0500) & 0700
              if bundle.read_only:
                # Enforce read-only if the root bundle does.
                file_mode &= 0500
              putfile(
                  srcfileobj, fullpath, file_mode,
                  use_symlink=use_symlinks)

            elif filetype == 'tar':
              basedir = os.path.dirname(fullpath)
              with tarfile.TarFile(fileobj=srcfileobj, encoding='utf-8') as t:
                for ti in t:
                  if not ti.isfile():
                    logging.warning(
                        'Path(%r) is nonfile (%s), skipped',
                        ti.name, ti.type)
                    continue
                  # Handle files created on Windows fetched on POSIX and the
                  # reverse.
                  other_sep = '/' if os.path.sep == '\\' else '\\'
                  name = ti.name.replace(other_sep, os.path.sep)
                  fp = os.path.normpath(os.path.join(basedir, name))
                  if not fp.startswith(basedir):
                    logging.error(
                        'Path(%r) is outside root directory',
                        fp)
                  ifd = t.extractfile(ti)
                  file_path.ensure_tree(os.path.dirname(fp))
                  file_mode = ti.mode & 0700
                  if bundle.read_only:
                    # Enforce read-only if the root bundle does.
                    file_mode &= 0500
                  putfile(ifd, fp, file_mode, ti.size)

            else:
              raise isolated_format.IsolatedError(
                    'Unknown file type %r', filetype)

        # Report progress.
        duration = time.time() - last_update
        if duration > DELAY_BETWEEN_UPDATES_IN_SECS:
          msg = '%d files remaining...' % len(remaining)
          sys.stdout.write(msg + '\n')
          sys.stdout.flush()
          logging.info(msg)
          last_update = time.time()
    assert fetch_queue.wait_queue_empty, 'FetchQueue should have been emptied'

  # Save the cache right away to not loose the state of the new objects.
  cache.save()
  # Cache could evict some items we just tried to fetch, it's a fatal error.
  if not fetch_queue.verify_all_cached():
    free_disk = file_path.get_free_space(cache.cache_dir)
    msg = (
        'Cache is too small to hold all requested files.\n'
        '  %s\n  cache=%dbytes, %d items; %sb free_space') % (
          cache.policies, cache.total_size, len(cache), free_disk)
    raise isolated_format.MappingError(msg)
  return bundle


def _directory_to_metadata(root, algo, blacklist):
  """Yields every file and/or symlink found.

  Yields:
    tuple(FileItem, relpath, metadata)
    For a symlink, FileItem is None.
  """
  # Current tar file bundle, if any.
  root = file_path.get_native_path_case(root)
  bundle = TarBundle(root, algo)
  for relpath, issymlink in isolated_format.expand_directory_and_symlink(
      root,
      u'.' + os.path.sep,
      blacklist,
      follow_symlinks=(sys.platform != 'win32')):

    filepath = os.path.join(root, relpath)
    if issymlink:
      # TODO(maruel): Do not call this.
      meta = isolated_format.file_to_metadata(filepath, 0, False)
      yield None, relpath, meta
      continue

    prio = relpath.endswith('.isolated')
    if bundle.try_add(FileItem(path=filepath, algo=algo, high_priority=prio)):
      # The file was added to the current pending tarball and won't be archived
      # individually.
      continue

    # Flush and reset the bundle.
    for i, p, m in bundle.yield_item_path_meta():
      yield i, p, m
    bundle = TarBundle(root, algo)

    # Yield the file individually.
    item = FileItem(path=filepath, algo=algo, size=None, high_priority=prio)
    yield item, relpath, item.meta

  for i, p, m in bundle.yield_item_path_meta():
    yield i, p, m


def _print_upload_stats(items, missing):
  """Prints upload stats."""
  total = len(items)
  total_size = sum(f.size for f in items)
  logging.info(
      'Total:      %6d, %9.1fkiB', total, total_size / 1024.)
  cache_hit = set(items).difference(missing)
  cache_hit_size = sum(f.size for f in cache_hit)
  logging.info(
      'cache hit:  %6d, %9.1fkiB, %6.2f%% files, %6.2f%% size',
      len(cache_hit),
      cache_hit_size / 1024.,
      len(cache_hit) * 100. / total,
      cache_hit_size * 100. / total_size if total_size else 0)
  cache_miss = missing
  cache_miss_size = sum(f.size for f in cache_miss)
  logging.info(
      'cache miss: %6d, %9.1fkiB, %6.2f%% files, %6.2f%% size',
      len(cache_miss),
      cache_miss_size / 1024.,
      len(cache_miss) * 100. / total,
      cache_miss_size * 100. / total_size if total_size else 0)


def _enqueue_dir(dirpath, blacklist, hash_algo, hash_algo_name):
  """Called by archive_files_to_storage for a directory.

  Create an .isolated file.

  Yields:
    FileItem for every file found, plus one for the .isolated file itself.
  """
  files = {}
  for item, relpath, meta in _directory_to_metadata(
      dirpath, hash_algo, blacklist):
    # item is None for a symlink.
    files[relpath] = meta
    if item:
      yield item

  # TODO(maruel): If there' not file, don't yield an .isolated file.
  data = {
    'algo': hash_algo_name,
    'files': files,
    'version': isolated_format.ISOLATED_FILE_VERSION,
  }
  # Keep the file in memory. This is fine because .isolated files are relatively
  # small.
  yield BufferItem(
      tools.format_json(data, True), algo=hash_algo, high_priority=True)


def archive_files_to_storage(storage, files, blacklist):
  """Stores every entry into remote storage and returns stats.

  Arguments:
    storage: a Storage object that communicates with the remote object store.
    files: iterable of files to upload. If a directory is specified (with a
          trailing slash), a .isolated file is created and its hash is returned.
          Duplicates are skipped.
    blacklist: function that returns True if a file should be omitted.

  Returns:
    tuple(OrderedDict(path: hash), list(FileItem cold), list(FileItem hot)).
    The first file in the first item is always the .isolated file.
  """
  # Dict of path to hash.
  results = collections.OrderedDict()
  hash_algo = storage.server_ref.hash_algo
  hash_algo_name = storage.server_ref.hash_algo_name
  # Generator of FileItem to pass to upload_items() concurrent operation.
  channel = threading_utils.TaskChannel()
  uploaded_digests = set()
  def _upload_items():
    results = storage.upload_items(channel)
    uploaded_digests.update(f.digest for f in results)
  t = threading.Thread(target=_upload_items)
  t.start()

  # Keep track locally of the items to determine cold and hot items.
  items_found = []
  try:
    for f in files:
      assert isinstance(f, unicode), repr(f)
      if f in results:
        # Duplicate
        continue
      try:
        filepath = os.path.abspath(f)
        if fs.isdir(filepath):
          # Uploading a whole directory.
          item = None
          for item in _enqueue_dir(
              filepath, blacklist, hash_algo, hash_algo_name):
            channel.send_result(item)
            items_found.append(item)
            # The very last item will be the .isolated file.
          if not item:
            # There was no file in the directory.
            continue
        elif fs.isfile(filepath):
          item = FileItem(
              path=filepath,
              algo=hash_algo,
              size=None,
              high_priority=f.endswith('.isolated'))
          channel.send_result(item)
          items_found.append(item)
        else:
          raise Error('%s is neither a file or directory.' % f)
        results[f] = item.digest
      except OSError:
        raise Error('Failed to process %s.' % f)
  finally:
    # Stops the generator, so _upload_items() can exit.
    channel.send_done()
  t.join()

  cold = []
  hot = []
  for i in items_found:
    # Note that multiple FileItem may have the same .digest.
    if i.digest in uploaded_digests:
      cold.append(i)
    else:
      hot.append(i)
  return results, cold, hot


@subcommand.usage('<file1..fileN> or - to read from stdin')
def CMDarchive(parser, args):
  """Archives data to the server.

  If a directory is specified, a .isolated file is created the whole directory
  is uploaded. Then this .isolated file can be included in another one to run
  commands.

  The commands output each file that was processed with its content hash. For
  directories, the .isolated generated for the directory is listed as the
  directory entry itself.
  """
  add_isolate_server_options(parser)
  add_archive_options(parser)
  options, files = parser.parse_args(args)
  process_isolate_server_options(parser, options, True, True)
  server_ref = isolate_storage.ServerRef(
      options.isolate_server, options.namespace)
  if files == ['-']:
    files = (l.rstrip('\n\r') for l in sys.stdin)
  if not files:
    parser.error('Nothing to upload')
  files = (f.decode('utf-8') for f in files)
  blacklist = tools.gen_blacklist(options.blacklist)
  try:
    with get_storage(server_ref) as storage:
      results, _cold, _hot = archive_files_to_storage(storage, files, blacklist)
  except (Error, local_caching.NoMoreSpace) as e:
    parser.error(e.args[0])
  print('\n'.join('%s %s' % (h, f) for f, h in results.iteritems()))
  return 0


def CMDdownload(parser, args):
  """Download data from the server.

  It can either download individual files or a complete tree from a .isolated
  file.
  """
  add_isolate_server_options(parser)
  parser.add_option(
      '-s', '--isolated', metavar='HASH',
      help='hash of an isolated file, .isolated file content is discarded, use '
           '--file if you need it')
  parser.add_option(
      '-f', '--file', metavar='HASH DEST', default=[], action='append', nargs=2,
      help='hash and destination of a file, can be used multiple times')
  parser.add_option(
      '-t', '--target', metavar='DIR', default='download',
      help='destination directory')
  parser.add_option(
      '--use-symlinks', action='store_true',
      help='Use symlinks instead of hardlinks')
  add_cache_options(parser)
  options, args = parser.parse_args(args)
  if args:
    parser.error('Unsupported arguments: %s' % args)
  if not file_path.enable_symlink():
    logging.warning('Symlink support is not enabled')

  process_isolate_server_options(parser, options, True, True)
  if bool(options.isolated) == bool(options.file):
    parser.error('Use one of --isolated or --file, and only one.')
  if not options.cache and options.use_symlinks:
    parser.error('--use-symlinks require the use of a cache with --cache')

  cache = process_cache_options(options, trim=True)
  cache.cleanup()
  options.target = unicode(os.path.abspath(options.target))
  if options.isolated:
    if (fs.isfile(options.target) or
        (fs.isdir(options.target) and fs.listdir(options.target))):
      parser.error(
          '--target \'%s\' exists, please use another target' % options.target)
  server_ref = isolate_storage.ServerRef(
      options.isolate_server, options.namespace)
  with get_storage(server_ref) as storage:
    # Fetching individual files.
    if options.file:
      # TODO(maruel): Enable cache in this case too.
      channel = threading_utils.TaskChannel()
      pending = {}
      for digest, dest in options.file:
        dest = unicode(dest)
        pending[digest] = dest
        storage.async_fetch(
            channel,
            threading_utils.PRIORITY_MED,
            digest,
            local_caching.UNKNOWN_FILE_SIZE,
            functools.partial(
                local_caching.file_write, os.path.join(options.target, dest)))
      while pending:
        fetched = channel.next()
        dest = pending.pop(fetched)
        logging.info('%s: %s', fetched, dest)

    # Fetching whole isolated tree.
    if options.isolated:
      bundle = fetch_isolated(
          isolated_hash=options.isolated,
          storage=storage,
          cache=cache,
          outdir=options.target,
          use_symlinks=options.use_symlinks)
      cache.trim()
      if bundle.command:
        rel = os.path.join(options.target, bundle.relative_cwd)
        print('To run this test please run from the directory %s:' %
              os.path.join(options.target, rel))
        print('  ' + ' '.join(bundle.command))

  return 0


def add_archive_options(parser):
  parser.add_option(
      '--blacklist',
      action='append', default=list(DEFAULT_BLACKLIST),
      help='List of regexp to use as blacklist filter when uploading '
           'directories')


def add_isolate_server_options(parser):
  """Adds --isolate-server and --namespace options to parser."""
  parser.add_option(
      '-I', '--isolate-server',
      metavar='URL', default=os.environ.get('ISOLATE_SERVER', ''),
      help='URL of the Isolate Server to use. Defaults to the environment '
           'variable ISOLATE_SERVER if set. No need to specify https://, this '
           'is assumed.')
  parser.add_option(
      '--grpc-proxy', help='gRPC proxy by which to communicate to Isolate')
  parser.add_option(
      '--namespace', default='default-gzip',
      help='The namespace to use on the Isolate Server, default: %default')


def process_isolate_server_options(
    parser, options, set_exception_handler, required):
  """Processes the --isolate-server option.

  Returns the identity as determined by the server.
  """
  if not options.isolate_server:
    if required:
      parser.error('--isolate-server is required.')
    return

  if options.grpc_proxy:
    isolate_storage.set_grpc_proxy(options.grpc_proxy)
  else:
    try:
      options.isolate_server = net.fix_url(options.isolate_server)
    except ValueError as e:
      parser.error('--isolate-server %s' % e)
  if set_exception_handler:
    on_error.report_on_exception_exit(options.isolate_server)
  try:
    return auth.ensure_logged_in(options.isolate_server)
  except ValueError as e:
    parser.error(str(e))


def add_cache_options(parser):
  cache_group = optparse.OptionGroup(parser, 'Cache management')
  cache_group.add_option(
      '--cache', metavar='DIR', default='cache',
      help='Directory to keep a local cache of the files. Accelerates download '
           'by reusing already downloaded files. Default=%default')
  cache_group.add_option(
      '--max-cache-size',
      type='int',
      metavar='NNN',
      default=50*1024*1024*1024,
      help='Trim if the cache gets larger than this value, default=%default')
  cache_group.add_option(
      '--min-free-space',
      type='int',
      metavar='NNN',
      default=2*1024*1024*1024,
      help='Trim if disk free space becomes lower than this value, '
           'default=%default')
  cache_group.add_option(
      '--max-items',
      type='int',
      metavar='NNN',
      default=100000,
      help='Trim if more than this number of items are in the cache '
           'default=%default')
  parser.add_option_group(cache_group)


def process_cache_options(options, trim, **kwargs):
  if options.cache:
    policies = local_caching.CachePolicies(
        options.max_cache_size,
        options.min_free_space,
        options.max_items,
        # 3 weeks.
        max_age_secs=21*24*60*60)

    # |options.cache| path may not exist until DiskContentAddressedCache()
    # instance is created.
    return local_caching.DiskContentAddressedCache(
        unicode(os.path.abspath(options.cache)), policies, trim, **kwargs)
  else:
    return local_caching.MemoryContentAddressedCache()


class OptionParserIsolateServer(logging_utils.OptionParserWithLogging):
  def __init__(self, **kwargs):
    logging_utils.OptionParserWithLogging.__init__(
        self,
        version=__version__,
        prog=os.path.basename(sys.modules[__name__].__file__),
        **kwargs)
    auth.add_auth_options(self)

  def parse_args(self, *args, **kwargs):
    options, args = logging_utils.OptionParserWithLogging.parse_args(
        self, *args, **kwargs)
    auth.process_auth_options(self, options)
    return options, args


def main(args):
  dispatcher = subcommand.CommandDispatcher(__name__)
  return dispatcher.execute(OptionParserIsolateServer(), args)


if __name__ == '__main__':
  subprocess42.inhibit_os_error_reporting()
  fix_encoding.fix_encoding()
  tools.disable_buffering()
  colorama.init()
  sys.exit(main(sys.argv[1:]))
