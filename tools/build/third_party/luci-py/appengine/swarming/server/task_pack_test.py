#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import os
import sys
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from test_support import test_case

from server import task_pack


# pylint: disable=W0212


class TaskPackApiTest(test_case.TestCase):
  def test_all_apis_are_tested(self):
    # Ensures there's a test for each public API.
    module = task_pack
    expected = frozenset(
        i for i in dir(module)
        if i[0] != '_' and hasattr(getattr(module, i), 'func_name'))
    missing = expected - frozenset(
        i[5:] for i in dir(self) if i.startswith('test_'))
    self.assertFalse(missing)

  def test_pack_request_key(self):
    self.assertEqual(
        '11',
       task_pack.pack_request_key(ndb.Key('TaskRequest', 0x7fffffffffffffee)))

  def test_unpack_request_key(self):
    self.assertEqual(
        ndb.Key('TaskRequest', 0x7fffffffffffffee),
        task_pack.unpack_request_key('11'))
    with self.assertRaises(ValueError):
      task_pack.unpack_request_key('2')

  def test_request_key_to_result_summary_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_key = task_pack.request_key_to_result_summary_key(request_key)
    expected = ndb.Key(
        'TaskRequest', 0x7fffffffffffffee, 'TaskResultSummary', 1)
    self.assertEqual(expected, result_key)

  def test_request_key_to_secret_bytes_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_key = task_pack.request_key_to_secret_bytes_key(request_key)
    expected = ndb.Key(
        'TaskRequest', 0x7fffffffffffffee, 'SecretBytes', 1)
    self.assertEqual(expected, result_key)

  def test_result_summary_key_to_request_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_summary_key = task_pack.request_key_to_result_summary_key(
        request_key)
    actual = task_pack.result_summary_key_to_request_key(result_summary_key)
    self.assertEqual(request_key, actual)

  def test_result_summary_key_to_run_result_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_summary_key = task_pack.request_key_to_result_summary_key(
        request_key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        result_summary_key, 1)
    expected = ndb.Key(
        'TaskRequest', 0x7fffffffffffffee, 'TaskResultSummary', 1,
        'TaskRunResult', 1)
    self.assertEqual(expected, run_result_key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        result_summary_key, 2)
    expected = ndb.Key(
        'TaskRequest', 0x7fffffffffffffee, 'TaskResultSummary', 1,
        'TaskRunResult', 2)
    self.assertEqual(expected, run_result_key)

    with self.assertRaises(ValueError):
      task_pack.result_summary_key_to_run_result_key(result_summary_key, 0)
    with self.assertRaises(ValueError):
      task_pack.result_summary_key_to_run_result_key(result_summary_key, 3)

  def test_run_result_key_to_performance_stats_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_summary_key = task_pack.request_key_to_result_summary_key(
        request_key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        result_summary_key, 1)
    perf_stats_key = task_pack.run_result_key_to_performance_stats_key(
        run_result_key)
    self.assertEqual('PerformanceStats',perf_stats_key.kind())

  def test_run_result_key_to_result_summary_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_summary_key = task_pack.request_key_to_result_summary_key(
        request_key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        result_summary_key, 1)
    self.assertEqual(
        result_summary_key,
        task_pack.run_result_key_to_result_summary_key(run_result_key))

  def test_pack_result_summary_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_summary_key = task_pack.request_key_to_result_summary_key(
        request_key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        result_summary_key, 1)

    actual = task_pack.pack_result_summary_key(result_summary_key)
    self.assertEqual('110', actual)

    with self.assertRaises(AssertionError):
      task_pack.pack_result_summary_key(run_result_key)

  def test_pack_run_result_key(self):
    request_key = task_pack.unpack_request_key('11')
    result_summary_key = task_pack.request_key_to_result_summary_key(
        request_key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        result_summary_key, 1)
    self.assertEqual('111', task_pack.pack_run_result_key(run_result_key))

    with self.assertRaises(AssertionError):
      task_pack.pack_run_result_key(result_summary_key)

  def test_unpack_result_summary_key(self):
    actual = task_pack.unpack_result_summary_key('bb80210')
    expected = ndb.Key(
        'TaskRequest', 0x7fffffffff447fde, 'TaskResultSummary', 1)
    self.assertEqual(expected, actual)

    with self.assertRaises(ValueError):
      task_pack.unpack_result_summary_key('0')
    with self.assertRaises(ValueError):
      task_pack.unpack_result_summary_key('g')
    with self.assertRaises(ValueError):
      task_pack.unpack_result_summary_key('bb80201')

  def test_unpack_run_result_key(self):
    for i in ('1', '2'):
      actual = task_pack.unpack_run_result_key('bb8021' + i)
      expected = ndb.Key(
          'TaskRequest', 0x7fffffffff447fde,
          'TaskResultSummary', 1, 'TaskRunResult', int(i))
      self.assertEqual(expected, actual)

    with self.assertRaises(ValueError):
      task_pack.unpack_run_result_key('1')
    with self.assertRaises(ValueError):
      task_pack.unpack_run_result_key('g')
    with self.assertRaises(ValueError):
      task_pack.unpack_run_result_key('bb80200')
    with self.assertRaises(ValueError):
      task_pack.unpack_run_result_key('bb80203')

  def test_get_request_and_result_keys(self):
    # Moved here, but tested in handlers_endpoints_test.py
    pass


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
