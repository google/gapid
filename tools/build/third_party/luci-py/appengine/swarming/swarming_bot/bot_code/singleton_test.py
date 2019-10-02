#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import os
import subprocess
import sys
import threading
import unittest

import singleton


THIS_DIR = os.path.dirname(os.path.abspath(__file__))


CMD_ACQUIRE = [
  sys.executable, '-u', '-c',
  'import singleton; print singleton.Singleton(%r).acquire()' % THIS_DIR,
]


class Test(unittest.TestCase):
  def test_singleton_with(self):
    with singleton.singleton(THIS_DIR) as s:
      self.assertEqual(True, s)

  def test_singleton_recursive(self):
    with singleton.singleton(THIS_DIR) as s:
      self.assertEqual(True, s)
      with singleton.singleton(THIS_DIR) as s2:
        self.assertEqual(False, s2)
      with singleton.singleton(THIS_DIR) as s3:
        self.assertEqual(False, s3)

  def test_singleton_acquire(self):
    f = singleton.Singleton(THIS_DIR)
    try:
      f.acquire()
    finally:
      f.release()

  def test_singleton_child(self):
    logging.info('using command:\n%s', ' '.join(CMD_ACQUIRE))
    with singleton.singleton(THIS_DIR):
      pass
    self.assertEqual('True\n', subprocess.check_output(CMD_ACQUIRE))
    with singleton.singleton(THIS_DIR):
      self.assertEqual('False\n', subprocess.check_output(CMD_ACQUIRE))
    self.assertEqual('True\n', subprocess.check_output(CMD_ACQUIRE))


if __name__ == '__main__':
  os.chdir(THIS_DIR)
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
