#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components.datastore_utils import mapping
from test_support import test_case


class EntityX(ndb.Model):
  a = ndb.IntegerProperty()


def int_ceil_div(value, divisor):
  """Returns the ceil() value of a integer based division."""
  return (value + divisor - 1) / divisor


class MappingTest(test_case.TestCase):
  def test_pop_future(self):
    items = [ndb.Future() for _ in xrange(5)]
    items[1].set_result(None)
    items[3].set_result('foo')
    inputs = items[:]
    mapping.pop_future_done(inputs)
    self.assertEqual([items[0], items[2], items[4]], inputs)

  def test_page_queries(self):
    for i in range(40):
      EntityX(id=i, a=i%4).put()
    queries = [
      EntityX.query(),
      EntityX.query(EntityX.a == 1),
      EntityX.query(EntityX.a == 2),
    ]
    actual = list(mapping.page_queries(queries))

    # The order won't be deterministic. The only important this is that exactly
    # all the items are returned as chunks.
    expected = [
      [EntityX(id=i, a=1) for i in xrange(1, 40, 4)],
      [EntityX(id=i, a=2) for i in xrange(2, 42, 4)],
      [EntityX(id=i, a=i%4) for i in xrange(1, 21)],
      [EntityX(id=i, a=i%4) for i in xrange(21, 40)],
    ]
    self.assertEqual(len(expected), len(actual))
    for line in actual:
      # Items may be returned out of order.
      try:
        i = expected.index(line)
      except ValueError:
        self.fail('%s not found in %s' % (line, actual))
      self.assertEqual(expected.pop(i), line)

  def test_incremental_map(self):
    for i in range(40):
      EntityX(id=i, a=i%4).put()
    queries = [
      EntityX.query(),
      EntityX.query(EntityX.a == 1),
      EntityX.query(EntityX.a == 2),
    ]
    actual = []
    # Use as much default arguments as possible.
    mapping.incremental_map(queries, actual.append)

    # The order won't be deterministic. The only important this is that exactly
    # all the items are returned as chunks and there is 3 chunks.
    expected = sorted(
        [EntityX(id=i, a=1) for i in xrange(1, 40, 4)] +
        [EntityX(id=i, a=2) for i in xrange(2, 42, 4)] +
        [EntityX(id=i, a=i%4) for i in xrange(1, 21)] +
        [EntityX(id=i, a=i%4) for i in xrange(21, 40)],
        key=lambda x: (x.key.id, x.to_dict()))
    map_page_size = 20
    self.assertEqual(int_ceil_div(len(expected), map_page_size), len(actual))
    actual = sorted(sum(actual, []), key=lambda x: (x.key.id, x.to_dict()))
    self.assertEqual(expected, actual)

  def test_incremental_map_throttling(self):
    for i in range(40):
      EntityX(id=i, a=i%4).put()
    queries = [
      EntityX.query(),
      EntityX.query(EntityX.a == 1),
      EntityX.query(EntityX.a == 2),
    ]
    actual = []
    def map_fn(items):
      actual.extend(items)
      # Note that it is returning more Future than what is called. It's fine.
      for _ in xrange(len(items) * 5):
        n = ndb.Future('yo dawg')
        # TODO(maruel): It'd be nice to not set them completed right away to
        # have better code coverage but I'm not sure how to do this.
        n.set_result('yo')
        yield n

    def filter_fn(item):
      return item.a == 2

    mapping.incremental_map(
        queries=queries,
        map_fn=map_fn,
        filter_fn=filter_fn,
        max_inflight=1,
        map_page_size=2,
        fetch_page_size=3)

    # The order won't be deterministic so sort it.
    expected = sorted(
        [EntityX(id=i, a=2) for i in xrange(2, 42, 4)] * 2,
        key=lambda x: (x.key.id, x.to_dict()))
    actual.sort(key=lambda x: (x.key.id, x.to_dict()))
    self.assertEqual(expected, actual)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
