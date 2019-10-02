#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

from components.config import validation

from server import directory_occlusion


class TestDirectoryOcclusionChecker(unittest.TestCase):
  def setUp(self):
    self.ctx = validation.Context()
    self.doc = directory_occlusion.Checker()

  def test_no_conflicts(self):
    self.doc.add('a', 'bobbie', 'for justice')
    self.doc.add('b', 'charlie', 'for peace')
    self.doc.add('c/d', 'ashley', 'for science')
    self.doc.add('c/e', 'logan', 'for victory')

    self.assertFalse(self.doc.conflicts(self.ctx))
    self.assertEqual(self.ctx.result().messages, [])

  def test_conflicting_directory(self):
    self.doc.add('some/path', 'bobbie', 'for justice')
    self.doc.add('some/path', 'charlie', 'for peace')

    self.assertTrue(self.doc.conflicts(self.ctx))
    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [
          ('\'some/path\': directory has conflicting owners: '
           'bobbie[\'for justice\'] and charlie[\'for peace\']')
        ])

  def test_conflicting_subdir(self):
    self.doc.add('some/path', 'bobbie', 'for justice')
    self.doc.add('some/path/other', 'charlie', 'for peace')

    self.assertTrue(self.doc.conflicts(self.ctx))
    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [
          ('charlie[\'for peace\'] uses \'some/path/other\', which conflicts '
           'with bobbie[\'for justice\'] using \'some/path\'')
        ])

  def test_conflicting_deep_subdir(self):
    self.doc.add('some/path', 'bobbie', 'for justice')
    self.doc.add('some/path/rather/deep/other', 'charlie', 'for peace')

    self.assertTrue(self.doc.conflicts(self.ctx))
    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [
          ('charlie[\'for peace\'] uses \'some/path/rather/deep/other\', '
           'which conflicts with bobbie[\'for justice\'] using \'some/path\'')
        ])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
