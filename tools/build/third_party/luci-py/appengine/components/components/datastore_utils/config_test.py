#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components.datastore_utils import config
from test_support import test_case


class ConfigTest(test_case.TestCase):
  def setUp(self):
    super(ConfigTest, self).setUp()
    # Disable in-memory NDB cache, it messes with cache related test cases.
    ndb.get_context().set_cache_policy(lambda _: False)

  def test_bootstrap(self):
    class Config(config.GlobalConfig):
      param = ndb.StringProperty()
      def set_defaults(self):
        self.param = 'abc'
    conf = Config.cached()
    self.assertEqual('abc', conf.param)
    self.assertEqual(conf.to_dict(), conf.fetch().to_dict())

  def test_fetch_store(self):
    class Config(config.GlobalConfig):
      param = ndb.StringProperty()
    conf = Config.fetch()
    self.assertIsNone(conf)
    conf = Config.cached()
    self.assertIsNotNone(conf)
    conf.param = '1234'
    now = self.mock_now(datetime.datetime(2010, 1, 1))
    conf.store(updated_by='someone')
    self.mock_now(datetime.datetime(2010, 1, 1), 100)
    conf = Config.fetch()
    self.assertEqual('1234', conf.param)
    self.assertEqual(now, conf.updated_ts)

  def test_expiration(self):
    self.mock_now(datetime.datetime(2014, 1, 2, 3, 4, 5, 6))

    class Config(config.GlobalConfig):
      param = ndb.StringProperty(default='default')

    # Bootstrap the config.
    Config.cached()

    # fetch-update cycle, necessary to avoid modifying cached copy in-place.
    conf = Config.fetch()
    conf.param = 'new-value'
    conf.store(updated_by='someone')

    # Right before expiration.
    self.mock_now(datetime.datetime(2014, 1, 2, 3, 4, 5, 6), 59)
    self.assertEqual('default', Config.cached().param)

    # After expiration.
    self.mock_now(datetime.datetime(2014, 1, 2, 3, 4, 5, 6), 61)
    self.assertEqual('new-value', Config.cached().param)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
