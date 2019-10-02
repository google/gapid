# coding=utf-8
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines Isolate Server model(s)."""

import datetime
import hashlib
import logging
import random
import zlib

from google.appengine.api import memcache
from google.appengine.ext import ndb

import config
import gcs
from components import datastore_utils
from components import utils


# The maximum number of entries that can be queried in a single request.
MAX_KEYS_PER_DB_OPS = 1000


# Maximum size of file stored in GS to be saved in memcache. The value must be
# small enough so that the whole content can safely fit in memory.
MAX_MEMCACHE_ISOLATED = 500*1024


# Valid namespace key.
NAMESPACE_RE = r'[a-z0-9A-Z\-._]+'


#### Models


class ContentEntry(ndb.Model):
  """Represents the content, keyed by its SHA-1 hash.

  Parent is a ContentShard.

  Key is '<namespace>-<hash>'.

  Eventually, the table name could have a prefix to determine the hashing
  algorithm, like 'sha1-'.

  There's usually only one table name:
    - default:    The default CAD.
    - temporary*: This family of namespace is a discardable namespace for
                  testing purpose only.

  The table name can have suffix:
    - -deflate: The namespace contains the content in deflated format. The
                content key is the hash of the uncompressed data, not the
                compressed one. That is why it is in a separate namespace.
  """
  # Cache the file size for statistics purposes.
  compressed_size = ndb.IntegerProperty(indexed=False)

  # The value is the Cache the expanded file size for statistics purposes. Its
  # value is different from size *only* in compressed namespaces. It may be -1
  # if yet unknown.
  expanded_size = ndb.IntegerProperty(indexed=False)

  # Set to True once the entry's content has been verified to match the hash.
  is_verified = ndb.BooleanProperty()

  # The content stored inline. This is only valid if the content was smaller
  # than MIN_SIZE_FOR_GS.
  content = ndb.BlobProperty()

  # Moment when this item expires and should be cleared. This is the only
  # property that has to be indexed.
  expiration_ts = ndb.DateTimeProperty()

  # Moment when this item should have its expiration time updatd.
  next_tag_ts = ndb.DateTimeProperty()

  # Moment when this item was created.
  creation_ts = ndb.DateTimeProperty(indexed=False, auto_now=True)

  # It is an important item, normally .isolated file.
  is_isolated = ndb.BooleanProperty(default=False)

  @property
  def is_compressed(self):
    """Is it the raw data or was it modified in any form, e.g. compressed, so
    that the SHA-1 doesn't match.
    """
    return self.key.parent().id().endswith(('-bzip2', '-deflate', '-gzip'))


### Private stuff.


# Valid hash keys.
_HASH_LETTERS = frozenset('0123456789abcdef')


### Public API.


def is_valid_hex(hex_string):
  """Returns True if the string consists of hexadecimal characters."""
  return _HASH_LETTERS.issuperset(hex_string)


def check_hash(hash_key, length):
  """Checks the validity of an hash_key. Doesn't use a regexp for speed.

  Raises in case of non-validity.
  """
  # It is faster than running a regexp.
  if len(hash_key) != length or not is_valid_hex(hash_key):
    raise ValueError('Invalid \'%s\' as ContentEntry key' % hash_key)


def get_hash(namespace):
  """Returns an initialized hashlib object that corresponds to the namespace."""
  if namespace.startswith('sha256-'):
    return hashlib.sha256()
  if namespace.startswith('sha512-'):
    return hashlib.sha512()
  return hashlib.sha1()


def get_entry_key(namespace, hash_key):
  """Returns a valid ndb.Key for a ContentEntry."""
  if isinstance(namespace, unicode):
    namespace = namespace.encode('utf-8')
  if isinstance(hash_key, unicode):
    hash_key = hash_key.encode('utf-8')
  check_hash(hash_key, get_hash(namespace).digest_size * 2)
  return entry_key_from_id('%s/%s' % (namespace, hash_key))


def entry_key_from_id(key_id):
  """Returns the ndb.Key for the key_id."""
  hash_key = key_id.rsplit('/', 1)[1]
  N = config.settings().sharding_letters
  return ndb.Key(
      ContentEntry, key_id,
      parent=datastore_utils.shard_key(hash_key, N, 'ContentShard'))


def get_content(namespace, hash_key):
  """Returns the content from either memcache or datastore, when stored inline.

  This does NOT return data from GCS, it is up to the client to do that.

  Returns:
    tuple(content, ContentEntry)
    At most only one of the two is set.

  Raises LookupError if the content cannot be found.
  Raises ValueError if the hash_key is invalid.
  """
  memcache_entry = memcache.get(hash_key, namespace='table_%s' % namespace)
  if memcache_entry is not None:
    return (memcache_entry, None)
  else:
    # Raises ValueError
    key = get_entry_key(namespace, hash_key)
    entity = key.get()
    if entity is None:
      raise LookupError("namespace %s, key %s does not refer to anything" %
        (namespace, hash_key))
    return (entity.content, entity)


def expiration_jitter(now, expiration):
  """Returns expiration/next_tag pair to set in a ContentEntry."""
  jittered = random.uniform(1, 1.2) * expiration
  expiration = now + datetime.timedelta(seconds=jittered)
  next_tag = now + datetime.timedelta(seconds=jittered*0.1)
  return expiration, next_tag


def expand_content(namespace, source):
  """Yields expanded data from source."""
  # Note: '-gzip' since it's a misnomer.
  if namespace.endswith(('-deflate', '-gzip')):
    zlib_state = zlib.decompressobj()
    for i in source:
      data = zlib_state.decompress(i, gcs.CHUNK_SIZE)
      yield data
      del data
      while zlib_state.unconsumed_tail:
        data = zlib_state.decompress(
            zlib_state.unconsumed_tail, gcs.CHUNK_SIZE)
        yield data
        del data
      del i
    data = zlib_state.flush()
    yield data
    del data
    # Forcibly delete the state.
    del zlib_state
  else:
    # Returns the source as-is.
    for i in source:
      yield i
      del i


def save_in_memcache(namespace, hash_key, content, async=False):
  namespace_key = 'table_%s' % namespace
  if async:
    return ndb.get_context().memcache_set(
        hash_key, content, namespace=namespace_key)
  try:
    if not memcache.set(hash_key, content, namespace=namespace_key):
      msg = 'Failed to save content to memcache.\n%s\\%s %d bytes' % (
          namespace_key, hash_key, len(content))
      if len(content) < 100*1024:
        logging.error(msg)
      else:
        logging.warning(msg)
  except ValueError as e:
    logging.error(e)


def new_content_entry(key, **kwargs):
  """Generates a new ContentEntry for the request.

  Doesn't store it. Just creates a new ContentEntry instance.
  """
  expiration, next_tag = expiration_jitter(
      utils.utcnow(), config.settings().default_expiration)
  return ContentEntry(
      key=key, expiration_ts=expiration, next_tag_ts=next_tag, **kwargs)


@ndb.tasklet
def delete_entry_and_gs_entry_async(key):
  """Deletes synchronously a ContentEntry and its GS file.

  It deletes the ContentEntry first, then the file in GS. The worst case is that
  the GS file is left behind and will be reaped by a lost GS task queue. The
  reverse is much worse, having a ContentEntry pointing to a deleted GS entry
  will lead to lookup failures.
  """
  bucket = config.settings().gs_bucket
  # Note that some content entries may NOT have corresponding GS files. That
  # happens for small entry stored inline in the datastore. Since this function
  # operates only on keys, it can't distinguish "large" entries stored in GS
  # from "small" ones stored inline. So instead it always tries to delete the
  # corresponding GS files, silently skipping ones that are not there.
  # Always delete ContentEntry first.
  name = key.string_id()
  yield key.delete_async()
  # This is synchronous.
  yield gcs.delete_file_async(bucket, name, ignore_missing=True)
  raise ndb.Return(None)
