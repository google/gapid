#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import logging
import os
import sys
import tempfile
import time
import unittest

import test_env_bot_code
test_env_bot_code.setup_test_env()

from depot_tools import auto_stub
from depot_tools import fix_encoding
from utils import file_path

import file_refresher


class TestFileRefresherThread(auto_stub.TestCase):
  def setUp(self):
    super(TestFileRefresherThread, self).setUp()
    self.root_dir = tempfile.mkdtemp(prefix='file_refresher')
    self.path = os.path.join(self.root_dir, 'target_file')

  def tearDown(self):
    file_path.rmtree(self.root_dir)
    super(TestFileRefresherThread, self).tearDown()

  def test_works(self):
    counter = [0]
    def callback():
      counter[0] += 1
      return counter[0]
    r = file_refresher.FileRefresherThread(self.path, callback, 0.1)
    r.start()
    time.sleep(1)
    r.stop()
    self.assertTrue(0 < counter[0] < 15) # was called reasonable number of times
    with open(self.path, 'rb') as f:
      body = json.load(f)
    self.assertEqual(counter[0], body) # actually updated the file


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
