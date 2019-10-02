# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Swarming bot named cache management, i.e. list of known named cache and their
state on each bot.

    +--------------+
    |NamedCacheRoot|
    |id=<pool>     |
    +--------------+
        |
        +----------------+
        |                |
        |                v
    +----------+     +----------+
    |NamedCache|     |NamedCache|
    |id=<name> | ... |id=<name> |
    +----------+     +----------+


The cached size hint for each named cache is the 95th percentile for the caches
found on the fleet. It is updated every 24 hours, so that if a large cache is
not re-observed for 24h, it will be lowered. When an higher size hint is
observed that is more than 10% of the previous one, the value is immediately
updated.

Caches for named cache that haven't been updated for 8 days are deleted.

The caches will only be precalculated for the pools defined in pools.cfg.
"""

import datetime
import json
import logging

from google.appengine.ext import ndb

from components import utils
from server import bot_management
from server import pools_config


### Models.


class NamedCacheRoot(ndb.Model):
  """There is one NamedCacheRoot entity per pool."""


class NamedCache(ndb.Model):
  """Represents the state of a single named cache.

  Parent is NamedCacheRoot for the pool.

  This entity is not stored in a transaction, as we want it to be hot in
  memcache, even if the hint is not exactly right.
  """
  ts = ndb.DateTimeProperty()
  os = ndb.StringProperty()
  name = ndb.StringProperty()
  hint = ndb.IntegerProperty(indexed=False, default=0)


### Private APIs.


def _named_cache_key(pool, os, name):
  """Returns the ndb.Key to a NamedCache."""
  assert isinstance(pool, basestring), repr(pool)
  assert isinstance(os, basestring), repr(os)
  assert isinstance(name, basestring), repr(name)
  return ndb.Key(NamedCacheRoot, pool, NamedCache, os + ':' + name)


def _update_named_cache(pool, os, name, hint):
  """Opportunistically update a named cache if the hint was off by 10% or more.

  Arguments:
  - pool: pool name
  - os: reduced 'os' value
  - name: named cache name
  - hint: observed size hint to use on the fleet

  It will be updated when:
  - The NamedCache is older than 24 hours, where the new size is used and the
    old maximum ignored.
  - The new maximum is at least 10% higher than the previous one.
  """
  assert isinstance(pool, basestring), repr(pool)
  assert isinstance(os, basestring), repr(os)
  assert isinstance(name, basestring), repr(name)
  assert isinstance(hint, (int, long)), repr(hint)
  key = _named_cache_key(pool, os, name)
  now = utils.utcnow().replace(microsecond=0)
  e = key.get()
  exp = now - datetime.timedelta(hours=24)
  if not e or e.hint <= hint*0.9 or e.ts < exp:
    e = NamedCache(key=key, ts=now, os=os, name=name, hint=hint)
    e.put()
    return True
  return False


def _reduce_oses(oses):
  """Returns a single OS key."""
  assert isinstance(oses, list), repr(oses)
  if not oses:
    return 'unknown'

  # TODO(maruel): It's a bit adhoc. Revisit if it's not good enough.
  if 'Windows' in oses:
    return 'Windows'
  if 'Mac' in oses:
    return 'Mac'
  if 'Android' in oses:
    return 'Android'
  if 'Linux' in oses:
    return 'Linux'

  return min(oses)


### Public APIs.


def get_hints(pool, oses, names):
  """Returns the hints for each named caches.

  Returns:
    list of hints in bytes for each named cache, or -1 when there's no hint
    available.
  """
  assert isinstance(oses, list), repr(oses)
  assert isinstance(names, list), repr(names)
  os = _reduce_oses(oses)
  keys = [_named_cache_key(pool, os, name) for name in names]
  entities = ndb.get_multi(keys)
  hints = [e.hint if e else -1 for e in entities]
  ancestor = ndb.Key(NamedCacheRoot, pool)
  for i, hint in enumerate(hints):
    if hint > -1:
      continue
    # Look for named cache in other OSes in the same pool.
    q = NamedCache.query(NamedCache.name == names[i], ancestor=ancestor)
    other_oses = q.fetch()
    if not other_oses:
      # TODO(maruel): We could define default hints in the pool.
      continue
    # Found something! Take the largest value.
    hints[i] = max(e.hint for e in other_oses)

  return hints


def task_update_pool(pool):
  """Updates the NamedCache for a pool.

  This needs to be able to scale for several thousands bots and hundreds of
  different caches.

  - Query all the bots in a pool.
  - Calculate the named caches for the bots in this pool.
  - Update the entities.
  """
  q = bot_management.BotInfo.query()
  found = {}
  bots = 0
  exp = utils.utcnow().replace(microsecond=0) - datetime.timedelta(hours=4)
  for bot in bot_management.filter_dimensions(q, [u'pool:'+pool]):
    if bot.last_seen_ts < exp:
      # Very dead bot; it hasn't pinged for 4 hours.
      continue
    bots += 1
    # Some bots do not have 'os' correctly specified. They fall into the
    # 'unknown' OS bucket.
    os = _reduce_oses(bot.dimensions.get('os') or [])
    state = bot.state
    if not state or not isinstance(state, dict):
      logging.debug('%s has no state', bot.id)
      continue
    # TODO(maruel): Use structured data instead of adhoc json.
    c = state.get('named_caches')
    if not isinstance(c, dict):
      continue
    d = found.setdefault(os, {})
    for key, value in sorted(c.iteritems()):
      if not value or not isinstance(value, list) or len(value) != 2:
        logging.error('%s has bad cache (A)', bot.id)
        continue
      if not value[0] or not isinstance(value[0], list) or len(value[0]) != 2:
        logging.error('%s has bad cache (B)', bot.id)
        continue
      s = value[0][1]
      d.setdefault(key, []).append(s)
  logging.info(
      'Found %d bots, %d caches in %d distinct OSes in pool %r',
      bots, sum(len(f) for f in found.itervalues()), len(found), pool)

  # TODO(maruel): Parallelise.
  for os, d in sorted(found.iteritems()):
    for name, sizes in sorted(d.iteritems()):
      # Adhoc calculation to take the ~95th percentile.
      sizes.sort()
      hint = sizes[int(float(len(sizes)) * 0.95)]
      if _update_named_cache(pool, os, name, hint):
        logging.debug('Pool %r  OS %r  Cache %r  hint=%d', pool, os, name, hint)

  # Delete the old ones.
  exp = utils.utcnow().replace(microsecond=0) - datetime.timedelta(days=8)
  ancestor = ndb.Key(NamedCacheRoot, pool)
  logging.info('Exp: %s', exp)
  q = NamedCache.query(NamedCache.ts < exp, ancestor=ancestor)
  keys = q.fetch(keys_only=True)
  if keys:
    logging.info('Deleting %d stale entities', len(keys))
    ndb.delete_multi(keys)
  return True


def cron_update_named_caches():
  """Trigger one task queue per pool to update NamedCache entites."""
  total = 0
  for pool in pools_config.known():
    if not utils.enqueue_task(
        '/internal/taskqueue/important/named_cache/update-pool',
        'named-cache-task',
        payload=json.dumps({'pool': pool}),
    ):
      logging.error('Failed to enqueue task for pool %s', pool)
    else:
      logging.debug('Enqueued task for pool %s', pool)
      total += 1
  return total
