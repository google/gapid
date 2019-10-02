# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Sharding Entity group utility function to improve performance.

This enforces artificial root entity grouping, which can be actually useful in
some specific circumstances.
"""

import hashlib
import string

from google.appengine.ext import ndb

__all__ = [
  'shard_key',
  'hashed_shard_key',
]


### Private stuff.


_HEX = frozenset(string.hexdigits.lower())


### Public API.


def shard_key(key, number_of_letters, root_entity_type):
  """Returns an ndb.Key to a virtual entity of type |root_entity_type|.

  This key is to be used as an entity group for database sharding. Transactions
  can be done over this group. Note that this sharding root entity doesn't have
  to ever exist in the database.

  Arguments:
    key: full key to take a subset of. It must be '[0-9a-f]+'. It is assumed
        that this key is well distributed, if not, use hashed_shard_key()
        instead. This means the available number of buckets is done in
        increments of 4 bits, e.g. 16, 256, 4096, 65536.
    number_of_letters: number of letters to use from |key|. key length must be
        encoded through an out-of-band mean and be constant.
    root_entity_type: root entity type. It can be either a reference to a
        ndb.Model class or just a string.
  """
  assert _HEX.issuperset(key), key
  assert isinstance(key, str) and len(key) >= number_of_letters, repr(key)
  # number_of_letters==10 means 1099511627776 shards, which is unreasonable.
  assert 1 <= number_of_letters < 10, number_of_letters
  assert isinstance(root_entity_type, (ndb.Model, str)) and root_entity_type, (
      root_entity_type)
  return ndb.Key(root_entity_type, key[:number_of_letters])


def hashed_shard_key(key, number_of_letters, root_entity_type):
  """Returns a ndb.Key to a virtual entity of type |root_entity_type|.

  The main difference with shard_key() is that it doesn't assume the key is well
  distributed so it first hashes the value via MD5 to make it more distributed.
  """
  return shard_key(
      hashlib.md5(key).hexdigest(), number_of_letters, root_entity_type)
