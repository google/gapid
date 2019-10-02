#!/usr/bin/env python
# coding: utf-8
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

# Setups environment.
import test_env_handlers

import cipd


class Test(unittest.TestCase):
  def test_is_valid_package_name(self):
    self.assertTrue(cipd.is_valid_package_name('foo'))
    self.assertTrue(cipd.is_valid_package_name('foo/.bar'))
    self.assertFalse(cipd.is_valid_package_name('foo{'))
    self.assertFalse(cipd.is_valid_package_name('foo/../bar'))

  def test_is_valid_package_name_template(self):
    for i in ('foo', 'foo${bar}', 'infra/tools/cipd/${platform}',
        'infra/git/${os=linux,mac}-${arch}'):
      self.assertTrue(cipd.is_valid_package_name_template(i), i)
    for i in ('foo{', 'foo{bar}', ''):
      self.assertFalse(cipd.is_valid_package_name_template(i), i)

  def test_is_valid_version(self):
    self.assertTrue(cipd.is_valid_version('foo'))
    self.assertFalse(cipd.is_valid_version('foo{'))

  def test_is_valid_tag(self):
    self.assertTrue(cipd.is_valid_tag('foo:1'))
    self.assertFalse(cipd.is_valid_tag('foo'))
    self.assertFalse(cipd.is_valid_tag('f'*401))

  def test_is_valid_instance_id(self):
    # Legacy SHA1s.
    self.assertTrue(cipd.is_valid_instance_id('1'*40))
    self.assertFalse(cipd.is_valid_instance_id('1'))
    # Newer base64-encoded hashes. No padding symbol is used or allowed.
    self.assertTrue(cipd.is_valid_instance_id(
        'b-AF8UbArxfy_4EXYaa8vAxTncdMtMIorleb_Wg303UC'))
    self.assertFalse(cipd.is_valid_instance_id(
        'bAF8UbArxfy_UEXYaa8vAxTncdMtMIorleb_Wg30UC=='))
    self.assertFalse(cipd.is_valid_instance_id(
        'b-AF8UbArxfy/4EXYaa8vAxTncdMtMIorleb_Wg303UC'))

  def test_is_pinned_version(self):
    self.assertTrue(cipd.is_pinned_version('1'*40))
    self.assertFalse(cipd.is_pinned_version('1'))
    self.assertTrue(cipd.is_pinned_version('foo:1'))
    self.assertFalse(cipd.is_pinned_version('ffff'))
    self.assertFalse(cipd.is_pinned_version('some/very/long/' + 'ref'*20))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL,
      format='%(levelname)-7s %(filename)s:%(lineno)3d %(message)s')
  unittest.main()
