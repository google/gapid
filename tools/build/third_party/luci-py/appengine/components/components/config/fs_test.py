#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components.config import fs
from test_support import test_case

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
CONFIG_ROOT = os.path.join(THIS_DIR, 'test_data', 'configs')


class FsTestCase(test_case.TestCase):
  def setUp(self):
    super(FsTestCase, self).setUp()
    self.provider = fs.Provider(CONFIG_ROOT)
    self.empty_provider = fs.Provider('nonexistent')

  def test_get(self):
    rev, content = self.provider.get_async(
        'projects/chromium', 'foo.cfg', revision='will be ignored').get_result()
    self.assertIsNone(rev)
    self.assertEqual(content, 'projects/chromium:foo.cfg\n')

  def test_get_projects_async(self):
    projects = self.provider.get_projects_async().get_result()
    self.assertEqual(projects, [
      {'id': 'chromium'},
      {'id': 'empty_project'},
      {'id': 'v8'},
    ])

  def test_get_project_configs(self):
    expected = {
      'projects/chromium': (None, 'projects/chromium:foo.cfg\n'),
      'projects/v8': (None, 'projects/v8:foo.cfg\n'),
    }
    actual = self.provider.get_project_configs_async('foo.cfg').get_result()
    self.assertEqual(expected, actual)

    actual = self.empty_provider.get_project_configs_async(
        'foo.cfg').get_result()
    self.assertEqual({}, actual)

  def test_get_ref_configs(self):
    expected = {
      'projects/chromium/refs/heads/master': (
          None, 'projects/chromium/refs/heads/master:foo.cfg\n'),
      'projects/chromium/refs/non-branch': (
          None, 'projects/chromium/refs/non-branch:foo.cfg\n'),
      'projects/v8/refs/heads/master': (
          None, 'projects/v8/refs/heads/master:foo.cfg\n'),
    }
    actual = self.provider.get_ref_configs_async('foo.cfg').get_result()
    self.assertEqual(expected, actual)

    actual = self.empty_provider.get_ref_configs_async('foo.cfg').get_result()
    self.assertEqual({}, actual)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
