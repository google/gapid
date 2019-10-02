#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import hashlib
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components.datastore_utils import sharding
from test_support import test_case


class EntityX(ndb.Model):
  a = ndb.IntegerProperty()


class ShardingTest(test_case.TestCase):
  def test_shard_key(self):
    actual = sharding.shard_key('1234', 2, 'Root')
    expected = "Key('Root', '12')"
    self.assertEqual(expected, str(actual))

  def test_hashed_shard_key(self):
    actual = sharding.hashed_shard_key('1234', 2, 'Root')
    expected = "Key('Root', '%s')" % hashlib.md5('1234').hexdigest()[:2]
    self.assertEqual(expected, str(actual))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
