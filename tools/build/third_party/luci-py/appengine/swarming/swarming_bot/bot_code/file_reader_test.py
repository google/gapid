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

import file_reader


class TestFileReaderThread(auto_stub.TestCase):
  def setUp(self):
    super(TestFileReaderThread, self).setUp()
    self.root_dir = tempfile.mkdtemp(prefix='file_reader')
    self.path = os.path.join(self.root_dir, 'target_file')

  def tearDown(self):
    file_path.rmtree(self.root_dir)
    super(TestFileReaderThread, self).tearDown()

  def test_works(self):
    with open(self.path, 'wb') as f:
      json.dump({'A': 'a'}, f)

    r = file_reader.FileReaderThread(self.path, 0.1)
    r.start()
    self.assertEqual(r.last_value, {'A': 'a'})

    # Change the file. Expect possible conflict on Windows.
    attempt = 0
    while True:
      try:
        file_path.atomic_replace(self.path, json.dumps({'B': 'b'}))
        break
      except OSError:
        attempt += 1
        if attempt == 20:
          self.fail('Cannot replace the file, giving up')
        time.sleep(0.05)

    # Give some reasonable time for the reader thread to pick up the change.
    # This test will flake if for whatever reason OS thread scheduler is lagging
    # for more than 2 seconds.
    time.sleep(2)

    self.assertEqual(r.last_value, {'B': 'b'})

  def test_start_throws_on_error(self):
    r = file_reader.FileReaderThread(
        self.path + "_no_such_file", max_attempts=2)
    with self.assertRaises(file_reader.FatalReadError):
      r.start()


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
