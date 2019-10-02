#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components.datastore_utils import txn
from test_support import test_case


class EntityX(ndb.Model):
  a = ndb.IntegerProperty()


class Failure(Exception):
  pass


class TransactionTest(test_case.TestCase):
  def test_transaction(self):
    def run():
      EntityX(a=1).put()
      return 2
    self.assertEqual(2, txn.transaction(run))
    self.assertEqual(1, EntityX.query().count())

  def test_transaction_async(self):
    def run():
      EntityX(a=1).put()
      return 2
    future = txn.transaction_async(run)
    self.assertEqual(2, future.get_result())
    self.assertEqual(1, EntityX.query().count())

  def test_transaction_failure(self):
    def run():
      EntityX(a=1).put()
      raise Failure()
    with self.assertRaises(Failure):
      txn.transaction(run)
    self.assertEqual(0, EntityX.query().count())

  def test_transaction_async_failure(self):
    def run():
      EntityX(a=1).put()
      raise Failure()
    future = txn.transaction_async(run)
    with self.assertRaises(Failure):
      future.get_result()
    self.assertEqual(0, EntityX.query().count())

  def test_transaction_failed_failure(self):
    def run():
      EntityX(a=1).put()
      raise Failure()
    with self.assertRaises(Failure):
      txn.transaction(run)
    self.assertEqual(0, EntityX.query().count())

  def test_transaction_async_failed_failure(self):
    def run():
      EntityX(a=1).put()
      raise Failure()
    future = txn.transaction_async(run)
    with self.assertRaises(Failure):
      future.get_result()
    self.assertEqual(0, EntityX.query().count())

  def test_transactional(self):
    @txn.transactional
    def run():
      EntityX(a=1).put()
      return 2
    self.assertEqual(2, run())
    self.assertEqual(1, EntityX.query().count())

  def test_transactional_async(self):
    @txn.transactional_async
    def run():
      EntityX(a=1).put()
      return 2
    future = run()
    self.assertEqual(2, future.get_result())
    self.assertEqual(1, EntityX.query().count())

  def test_transactional_task(self):
    @txn.transactional_tasklet
    def run():
      yield EntityX(a=1).put_async()
      raise ndb.Return(2)
    future = run()
    self.assertEqual(2, future.get_result())
    self.assertEqual(1, EntityX.query().count())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
