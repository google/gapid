#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import hashlib
import json
import logging
import sys
import unittest

# pylint: disable=wrong-import-position
import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import utils
from server import bot_management
from server import named_caches
from server import pools_config
from test_support import test_case


def _bot_event(bot_id, pool, caches, oses):
  """Calls bot_management.bot_event with default arguments."""
  dimensions = {
    u'id': [bot_id],
    u'os': oses or [u'Linux', u'Ubuntu', u'Ubuntu-16.04'],
    u'pool': [pool],
  }
  # Format is named_caches: {name: [['shortname', size], timestamp]}.
  state = {
    'named_caches': {
      name: [['a', size], 10] for name, size in caches.iteritems()
    }
  }
  bot_management.bot_event(
      event_type='bot_connected',
      bot_id=bot_id,
      external_ip='8.8.4.4',
      authenticated_as=u'bot:%s.domain' % bot_id,
      dimensions=dimensions,
      state=state or {'ram': 65},
      version=unicode(hashlib.sha256().hexdigest()),
      quarantined=False,
      maintenance_msg=None,
      task_id=None,
      task_name=None)


class NamedCachesTest(test_case.TestCase):
  APP_DIR = test_env.APP_DIR

  def setUp(self):
    super(NamedCachesTest, self).setUp()
    self.mock(utils, 'enqueue_task', self._enqueue_task)
    self.mock(pools_config, 'known', lambda: ['first', 'second'])

  @ndb.non_transactional
  def _enqueue_task(self, url, queue_name, payload):
    if queue_name == 'named-cache-task':
      params = json.loads(payload)
      self.assertEqual(True, named_caches.task_update_pool(params['pool']))
      return True
    self.fail(url)
    return False

  def test_simple(self):
    _bot_event('first1', 'first', {'git': 1}, None)
    _bot_event('first2', 'first', {'build': 100000, 'git': 1000}, None)
    self.assertEqual(2, named_caches.cron_update_named_caches())

    self.assertEqual(2, named_caches.NamedCache.query().count())
    oses = ['Linux']
    hints = named_caches.get_hints('first', oses, ['git', 'build', 'new'])
    self.assertEqual([1000, 100000, -1], hints)

  def test_p95(self):
    # Create 45 bots with cache 'foo' size between 1 and 45.
    for i in xrange(45):
      _bot_event('second%d' % i, 'second', {'foo': i+1}, None)
    self.assertEqual(2, named_caches.cron_update_named_caches())

    self.assertEqual(1, named_caches.NamedCache.query().count())
    oses = ['Linux']
    hints = named_caches.get_hints('second', oses, ['foo'])
    # Roughly p95.
    self.assertEqual([43], hints)

  def test_fuzzy_other_os(self):
    # Use the hint from 'Mac' (the larger one) even if requesting for Linux.
    _bot_event('first1', 'first', {'build': 50000}, ['Android'])
    _bot_event('first2', 'first', {'build': 100000}, ['Mac'])
    self.assertEqual(2, named_caches.cron_update_named_caches())

    self.assertEqual(2, named_caches.NamedCache.query().count())
    oses = ['Linux']
    hints = named_caches.get_hints('first', oses, ['build'])
    self.assertEqual([100000], hints)

  def test_expired(self):
    now = datetime.datetime(2015, 1, 1, 1, 1, 1)
    self.mock_now(now)
    _bot_event('first1', 'first', {'build': 100000}, ['Mac'])
    self.assertEqual(2, named_caches.cron_update_named_caches())
    self.assertEqual([now], [e.ts for e in named_caches.NamedCache.query()])

    self.mock_now(now, 8*24*60*60)
    self.assertEqual(2, named_caches.cron_update_named_caches())
    self.assertEqual(1, named_caches.NamedCache.query().count())
    self.assertEqual([now], [e.ts for e in named_caches.NamedCache.query()])

    self.mock_now(now, 8*24*60*60+1)
    self.assertEqual(2, named_caches.cron_update_named_caches())
    self.assertEqual(0, named_caches.NamedCache.query().count())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
