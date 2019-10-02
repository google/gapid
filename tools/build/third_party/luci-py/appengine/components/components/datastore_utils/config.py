# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Versioned singleton entity with global application configuration.

Example usage:
  from components.datastore_utils import config

  class MyConfig(config.GlobalConfig):
    param1 = ndb.StringProperty()
    param2 = ndb.StringProperty()
    ...

  ...

  def do_stuff():
    param1 = MyConfig.cached().param1
    param2 = MyConfig.cached().param2
    ...

  def modify():
    conf = MyConfig.fetch()
    conf.modify(updated_by='user:abc@example.com', param1='123')


Advantages over regular ndb.Entity with predefined key:
  1. All changes are logged (see datastore_utils.store_new_version).
  2. In-memory process-wide cache.
"""

# Pylint fails to recognize that ndb.Model.key is defined in ndb.Model.
# pylint: disable=attribute-defined-outside-init

import datetime
import threading

from google.appengine.ext import ndb

from components import datastore_utils
from components import utils


class GlobalConfig(ndb.Model):
  """Singleton entity with the global configuration of the service.

  All changes are stored in the revision log.
  """

  # When this revision of configuration was created.
  updated_ts = ndb.DateTimeProperty(indexed=False, auto_now_add=True)
  # Who created this revision of configuration (as identity string).
  updated_by = ndb.StringProperty(indexed=False, default='')

  @classmethod
  def cached_async(cls):
    """Fetches config entry from local cache or datastore.

    Bootstraps it if missing. May return slightly stale data but in most cases
    doesn't do any RPCs. Should be used for read-only access to config.
    """
    # Build new class-specific fetcher function with cache on the fly on
    # the first attempt (it's not a big deal if it happens concurrently in MT
    # environment, last one wins). Same can be achieved with metaclasses, but no
    # one likes metaclasses.
    if not cls._config_fetcher_async:
      @ndb.tasklet
      def fetcher():
        with fetcher.cache_lock:
          expiry = fetcher.cache_expiry
          if expiry is not None and utils.utcnow() < expiry:
            raise ndb.Return(fetcher.cache_value)

        # Do not lock while yielding, it would cause deadlock.
        # Also do not cache a future, it might cross ndb context boundary.
        # If there is no cached value, multiple concurrent requests will make
        # multiple RPCs, but as soon as one of them updates cache, subsequent
        # requests will use the cached value, for a minute.
        conf = yield cls.fetch_async()
        if not conf:
          conf = cls()
          conf.set_defaults()
          yield conf.store_async(updated_by='')

        with fetcher.cache_lock:
          fetcher.cache_expiry = utils.utcnow() + datetime.timedelta(minutes=1)
          fetcher.cache_value = conf
        raise ndb.Return(conf)

      fetcher.cache_lock = threading.Lock()
      fetcher.cache_expiry = None
      fetcher.cache_value = None

      cls._config_fetcher_async = staticmethod(fetcher)
    return cls._config_fetcher_async()

  cached = utils.sync_of(cached_async)

  @classmethod
  def clear_cache(cls):
    """Clears the cache of .cached().

    So the next call to .cached() returns the fresh instance from ndb.
    """
    if cls._config_fetcher_async:
      cls._config_fetcher_async.cache_expiry = None

  @classmethod
  def fetch_async(cls):
    """Returns the current up-to-date version of the config entity.

    Always fetches it from datastore. May return None if missing.
    """
    return datastore_utils.get_versioned_most_recent_async(
        cls, cls._get_root_key())

  fetch = utils.sync_of(fetch_async)

  def store_async(self, updated_by):
    """Stores a new version of the config entity."""
    # Create an incomplete key, to be completed by 'store_new_version'.
    self.key = ndb.Key(self.__class__, None, parent=self._get_root_key())
    self.updated_by = updated_by
    self.updated_ts = utils.utcnow()
    return datastore_utils.store_new_version_async(self, self._get_root_model())

  store = utils.sync_of(store_async)

  def modify(self, updated_by, **kwargs):
    """Applies |kwargs| dict to the entity and stores the entity if changed."""
    dirty = False
    for k, v in kwargs.iteritems():
      assert k in self._properties, k
      if getattr(self, k) != v:
        setattr(self, k, v)
        dirty = True
    if dirty:
      self.store(updated_by)
    return dirty

  def set_defaults(self):
    """Fills in default values for empty config. Implemented by subclasses."""

  ### Private stuff.

  _config_fetcher_async = None

  @classmethod
  def _get_root_model(cls):
    return datastore_utils.get_versioned_root_model('%sRoot' % cls.__name__)

  @classmethod
  def _get_root_key(cls):
    return ndb.Key(cls._get_root_model(), 1)
