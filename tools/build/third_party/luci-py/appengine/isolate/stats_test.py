#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import sys
import unittest

# pylint: disable=wrong-import-position
import test_env
test_env.setup_test_env()

import webapp2
import webtest

from components import stats_framework
from components.stats_framework import stats_logs
from test_support import stats_framework_logs_mock
from test_support import test_case

from proto import isolated_pb2
import stats


class Store(webapp2.RequestHandler):
  def get(self):
    """Generates fake stats."""
    stats.add_entry(stats.STORE, 2048, 'GS; inline')
    self.response.write('Yay')


class Return(webapp2.RequestHandler):
  def get(self):
    """Generates fake stats."""
    stats.add_entry(stats.RETURN, 4096, 'memcache')
    self.response.write('Yay')


class Lookup(webapp2.RequestHandler):
  def get(self):
    """Generates fake stats."""
    stats.add_entry(stats.LOOKUP, 200, 103)
    self.response.write('Yay')


class Dupe(webapp2.RequestHandler):
  def get(self):
    """Generates fake stats."""
    stats.add_entry(stats.DUPE, 1024, 'inline')
    self.response.write('Yay')


def to_str(now, delta):
  """Converts a datetime to unicode."""
  now = now + datetime.timedelta(seconds=delta)
  return unicode(now.strftime(stats.utils.DATETIME_FORMAT))


class StatsTest(test_case.TestCase):
  def setUp(self):
    super(StatsTest, self).setUp()
    fake_routes = [
        ('/store', Store),
        ('/return', Return),
        ('/lookup', Lookup),
        ('/dupe', Dupe),
    ]
    self.app = webtest.TestApp(
        webapp2.WSGIApplication(fake_routes, debug=True),
        extra_environ={'REMOTE_ADDR': 'fake-ip'})
    stats_framework_logs_mock.configure(self)
    self.now = datetime.datetime(2010, 1, 2, 3, 4, 5, 6)
    self.mock_now(self.now, 0)

  def test_all_apis_are_tested(self):
    # Ensures there's a test for each public API.
    module = stats
    expected = frozenset(
        i for i in dir(module)
        if i[0] != '_' and hasattr(getattr(module, i), 'func_name'))
    missing = expected - frozenset(
        i[5:] for i in dir(self) if i.startswith('test_'))
    self.assertFalse(missing)

  def _test_handler(self, url, added_data):
    stats_framework_logs_mock.reset_timestamp(stats.STATS_HANDLER, self.now)

    self.assertEqual('Yay', self.app.get(url).body)
    self.assertEqual(1, len(list(stats_logs.yield_entries(None, None))))

    self.mock_now(self.now, 60)
    self.assertEqual(10, stats.cron_generate_stats())

    actual = stats_framework.get_stats(
        stats.STATS_HANDLER, 'minutes', self.now, 1, True)
    expected = [
      {
        'contains_lookups': 0,
        'contains_requests': 0,
        'downloads': 0,
        'downloads_bytes': 0,
        'failures': 0,
        'key': '2010-01-02T03:04',
        'requests': 1,
        'uploads': 0,
        'uploads_bytes': 0,
      },
    ]
    expected[0].update(added_data)
    self.assertEqual(expected, actual)

  def test_store(self):
    expected = {
      'uploads': 1,
      'uploads_bytes': 2048,
    }
    self._test_handler('/store', expected)

  def test_return(self):
    expected = {
      'downloads': 1,
      'downloads_bytes': 4096,
    }
    self._test_handler('/return', expected)

  def test_lookup(self):
    expected = {
      'contains_lookups': 200,
      'contains_requests': 1,
    }
    self._test_handler('/lookup', expected)

  def test_dupe(self):
    expected = {}
    self._test_handler('/dupe', expected)

  def test_add_entry(self):
    # Tested by other test cases.
    pass

  def test_snapshot_to_proto(self):
    s = stats.STATS_HANDLER.stats_minute_cls(
        key=stats.STATS_HANDLER.minute_key(self.now),
        created=self.now,
        values_compressed=stats._Snapshot(
          uploads=1,
          uploads_bytes=2,
          downloads=3,
          downloads_bytes=4,
          contains_requests=5,
          contains_lookups=6,
          requests=7,
          failures=8,
        ))
    p = isolated_pb2.StatsSnapshot()
    stats.snapshot_to_proto(s, p)
    expected = (
      u'start_time {\n'
      u'  seconds: 1262401440\n'
      u'}\n'
      u'uploads: 1\n'
      u'uploads_bytes: 2\n'
      u'downloads: 3\n'
      u'downloads_bytes: 4\n'
      u'contains_requests: 5\n'
      u'contains_lookups: 6\n'
      u'requests: 7\n'
      u'failures: 8\n')
    self.assertEqual(expected, unicode(p))

  def test_cron_generate_stats(self):
    # It generates empty stats.
    self.assertEqual(120, stats.cron_generate_stats())

  def test_cron_send_to_bq_empty(self):
    # Empty, nothing is done. No need to mock the HTTP client.
    self.assertEqual(0, stats.cron_send_to_bq())
    # State is not stored if nothing was found.
    self.assertEqual(None, stats.BqStateStats.get_by_id(1))

  def test_cron_send_to_bq(self):
    # Generate entities.
    self.assertEqual(120, stats.cron_generate_stats())

    payloads = []
    def json_request(url, method, payload, scopes, deadline):
      self.assertEqual(
          'https://www.googleapis.com/bigquery/v2/projects/sample-app/datasets/'
            'isolated/tables/stats/insertAll',
          url)
      payloads.append(payload)
      self.assertEqual('POST', method)
      self.assertEqual(stats.bqh.INSERT_ROWS_SCOPE, scopes)
      self.assertEqual(600, deadline)
      return {'insertErrors': []}
    self.mock(stats.net, 'json_request', json_request)

    self.assertEqual(120, stats.cron_send_to_bq())
    expected = {
      'failed': [],
      'last': datetime.datetime(2009, 12, 28, 2, 0),
      'ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
    }
    self.assertEqual(
        expected, stats.BqStateStats.get_by_id(1).to_dict())

    expected = [
      {
        'ignoreUnknownValues': False,
        'kind': 'bigquery#tableDataInsertAllRequest',
        'skipInvalidRows': True,
      },
    ]
    actual_rows = payloads[0].pop('rows')
    self.assertEqual(expected, payloads)
    self.assertEqual(120, len(actual_rows))

    # Next cron skips everything that was processed.
    self.assertEqual(0, stats.cron_send_to_bq())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
