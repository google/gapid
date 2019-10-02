# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Monotonic addition of entities."""

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb
from google.appengine.runtime import apiproxy_errors

from components import utils
from . import txn


__all__ = [
  'HIGH_KEY_ID',
  'Root',
  'get_versioned_most_recent',
  'get_versioned_most_recent_async',
  'get_versioned_most_recent_with_root',
  'get_versioned_most_recent_with_root_async',
  'get_versioned_root_model',
  'insert',
  'insert_async',
  'store_new_version',
  'store_new_version_async',
]


# 2^53 is the largest that can be represented with a float. It's a bit large
# though so save a bit and start at 2^48-1.
HIGH_KEY_ID = (1 << 47) - 1


### Private stuff.


### Public API.


class Root(ndb.Model):
  """Root entity used for store_new_version() and get_versioned_most_recent().

  Either inherit from this class or use get_versioned_root_model().
  """
  # Key id of the most recent child entity in the DB. It is monotonically
  # decreasing starting at HIGH_KEY_ID. It is None if no child is present.
  current = ndb.IntegerProperty(indexed=False)


@ndb.tasklet
def insert_async(entity, new_key_callback=None, extra=None):
  """Inserts an entity in the DB and guarantees creation.

  Similar in principle to ndb.Model.get_or_insert() except that it only succeeds
  when the entity was not already present. As such, this always requires a
  transaction.

  Optionally retries with a new key if |new_key_callback| is provided.

  Arguments:
    entity: entity to save, it should have its .key already set accordingly. The
        .key property will be mutated, even if the function fails. It is highly
        preferable to have a root entity so the transaction can be done safely.
    new_key_callback: function to generates a new key if the previous key was
        already taken. If this function returns None, the execution is aborted.
        If this parameter is None, insertion is only tried once.
        May return a future.
    extra: additional entities to store simultaneously. For example a bookeeping
        entity that must be updated simultaneously along with |entity|. All the
        entities must be inside the same entity group. This function is not safe
        w.r.t. `extra`, entities in this list will overwrite entities already in
        the DB or have their key updated when new_key_callback() is called.

  Returns:
    ndb.Key of the newly saved entity or None if the entity was already present
    in the db.
  """
  assert not ndb.in_transaction()
  assert entity.key.id(), entity.key
  entities = [entity]
  if extra:
    entities.extend(extra)
    root = entity.key.pairs()[0]
    assert all(i.key and i.key.pairs()[0] == root for i in extra), extra

  def new_key_callback_async():
    key = None
    if new_key_callback:
      key = new_key_callback()
    if isinstance(key, ndb.Future):
       return key
    future = ndb.Future()
    future.set_result(key)
    return future

  @ndb.tasklet
  def run():
    if (yield entities[0].key.get_async()):
      # The entity exists, abort.
      raise ndb.Return(False)
    yield ndb.put_multi_async(entities)
    raise ndb.Return(True)

  # TODO(maruel): Run a severe load test and count the number of retries.
  while True:
    # First iterate outside the transaction in case the first entity key number
    # selected is already used.
    while entity.key and entity.key.id() and (yield entity.key.get_async()):
      entity.key = yield new_key_callback_async()

    if not entity.key or not entity.key.id():
      break

    try:
      if (yield txn.transaction_async(run, retries=0)):
        break
    except txn.CommitError:
      # Retry with the same key.
      pass
    else:
      # Entity existed. Get the next key.
      entity.key = yield new_key_callback_async()
  raise ndb.Return(entity.key)


insert = utils.sync_of(insert_async)


def get_versioned_root_model(model_name):
  """Returns a root model that can be used for versioned entities.

  Using this entity for get_versioned_most_recent(),
  get_versioned_most_recent_with_root() and store_new_version() is optional. Any
  entity with cls.current as an ndb.IntegerProperty will do.
  """
  assert isinstance(model_name, str), model_name
  class _Root(Root):
    @classmethod
    def _get_kind(cls):
      return model_name

  return _Root


@ndb.tasklet
def get_versioned_most_recent_async(cls, root_key):
  """Returns the most recent entity of cls child of root_key."""
  _, entity = yield get_versioned_most_recent_with_root_async(cls, root_key)
  raise ndb.Return(entity)


get_versioned_most_recent = utils.sync_of(get_versioned_most_recent_async)

@ndb.tasklet
def get_versioned_most_recent_with_root_async(cls, root_key):
  """Returns the most recent instance of a versioned entity and the root entity.

  Getting the root entity is needed to get the current index.
  """
  # Using a cls.query(ancestor=root_key).get() would work too but is less
  # efficient since it can't be cached by ndb's cache.
  assert not ndb.in_transaction()
  assert issubclass(cls, ndb.Model), cls
  assert root_key is None or isinstance(root_key, ndb.Key), root_key

  root = root_key.get()
  if not root or not root.current:
    raise ndb.Return(None, None)
  entity = yield ndb.Key(cls, root.current, parent=root_key).get_async()
  raise ndb.Return(root, entity)


get_versioned_most_recent_with_root = utils.sync_of(
  get_versioned_most_recent_with_root_async
)


@ndb.tasklet
def store_new_version_async(entity, root_cls, extra=None):
  """Stores a new version of the instance.

  entity.key is updated to the key used to store the entity. Only the parent key
  needs to be set. E.g. Entity(parent=ndb.Key(ParentCls, ParentId), ...) or
  entity.key = ndb.Key(Entry, None, ParentCls, ParentId).

  If there was no root entity in the DB, one is created by calling root_cls().

  Fetch for root entity is not done in a transaction, so this function is unsafe
  w.r.t. root content.

  Arguments:
    entity: ndb.Model entity to append in the DB.
    root_cls: class returned by get_versioned_root_model().
    extra: extraneous entities to put in the transaction. They must all be in
        the same entity group.

  Returns:
    tuple(root, entity) with the two entities that were PUT in the db.
  """
  assert not ndb.in_transaction()
  assert isinstance(entity, ndb.Model), entity
  assert entity.key and entity.key.parent(), 'entity.key.parent() must be set.'
  # Access to a protected member _XX of a client class - pylint: disable=W0212
  assert root_cls._properties.keys() == ['current'], (
      'This function is unsafe for root entity, use store_new_version_safe '
      'which is not yet implemented')
  root_key = entity.key.parent()
  root = (yield root_key.get_async()) or root_cls(key=root_key)
  root.current = root.current or HIGH_KEY_ID
  flat = list(entity.key.flat())
  flat[-1] = root.current
  entity.key = ndb.Key(flat=flat)

  def _new_key_minus_one_current():
    flat[-1] -= 1
    root.current = flat[-1]
    return ndb.Key(flat=flat)

  extra = (extra or [])[:]
  extra.append(root)
  result = yield insert_async(entity, _new_key_minus_one_current, extra=extra)
  raise ndb.Return(result)


store_new_version = utils.sync_of(store_new_version_async)
