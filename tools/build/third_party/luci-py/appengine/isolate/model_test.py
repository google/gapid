#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import hashlib
import logging
import os
import sys
import unittest

import test_env
test_env.setup_test_env()

from components import auth
from components import auth_testing
from components import datastore_utils
from test_support import test_case

import model

# Access to a protected member _XXX of a client class
# pylint: disable=W0212


def hash_item(namespace, content):
  h = model.get_hash(namespace)
  h.update(content)
  return h.hexdigest()


class MainTest(test_case.TestCase):
  """Tests the handlers."""
  APP_DIR = test_env.APP_DIR

  def setUp(self):
    """Creates a new app instance for every test case."""
    super(MainTest, self).setUp()
    auth_testing.mock_get_current_identity(
        self, auth.Identity(auth.IDENTITY_USER, 'reader@example.com'))

  def test_ancestor_assumption(self):
    prefix = '1234'
    suffix = 40 - len(prefix)
    c = model.new_content_entry(model.get_entry_key('n', prefix + '0' * suffix))
    self.assertEqual(0, len(list(model.ContentEntry.query())))
    c.put()
    self.assertEqual(1, len(list(model.ContentEntry.query())))

    c = model.new_content_entry(model.get_entry_key('n', prefix + '1' * suffix))
    self.assertEqual(1, len(list(model.ContentEntry.query())))
    c.put()
    self.assertEqual(2, len(list(model.ContentEntry.query())))

    actual_prefix = c.key.parent().id()
    k = datastore_utils.shard_key(
        actual_prefix, len(actual_prefix), 'ContentShard')
    self.assertEqual(2, len(list(model.ContentEntry.query(ancestor=k))))

  def test_get_hash(self):
    content = 'foo'
    self.assertEqual(
        '0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33',
        hash_item('default-gzip', content))
    self.assertEqual(
        '0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33',
        hash_item('sha1-gzip', content))
    self.assertEqual(
        '2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae',
        hash_item('sha256-gzip', content))
    self.assertEqual(
        'f7fbba6e0636f890e56fbbf3283e524c6fa3204ae298382d624741d0dc6638326e282'
          'c41be5e4254d8820772c5518a2c5a8c0c7f7eda19594a7eb539453e1ed7',
        hash_item('sha512-gzip', content))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
