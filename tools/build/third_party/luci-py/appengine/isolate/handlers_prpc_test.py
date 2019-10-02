#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

import webapp2
import webtest

from components.prpc import encoding
from test_support import test_case

from proto import isolated_pb2  # pylint: disable=no-name-in-module

import handlers_prpc
import stats


def _decode(raw, dst):
  # Skip escaping characters.
  assert raw[:5] == ')]}\'\n', raw[:5]
  return encoding.get_decoder(encoding.Encoding.JSON)(raw[5:], dst)


def _encode(d):
  # Skip escaping characters.
  raw = encoding.get_encoder(encoding.Encoding.JSON)(d)
  assert raw[:5] == ')]}\'\n', raw[:5]
  return raw[5:]


class PRPCTest(test_case.TestCase):
  """Tests the pRPC handlers."""
  APP_DIR = test_env.APP_DIR

  def setUp(self):
    super(PRPCTest, self).setUp()
    self.app = webtest.TestApp(
        webapp2.WSGIApplication(handlers_prpc.get_routes(), debug=True),
        extra_environ={'REMOTE_ADDR': '::ffff:127.0.0.1'},
    )
    self._headers = {
      'Content-Type': encoding.Encoding.JSON[1],
      'Accept': encoding.Encoding.JSON[1],
    }
    self.now = datetime.datetime(2010, 1, 2, 3, 4, 5, 6)

  def _gen_stats(self):
    """Generates data for the last 10 days, last 10 hours and last 10 minutes.
    """
    # TODO(maruel): Stop accessing the DB directly. Use stats_framework_mock to
    # generate it.
    self.mock_now(self.now, 0)
    handler = stats.STATS_HANDLER
    for i in xrange(10):
      s = stats._Snapshot(requests=100 + i)
      day = (self.now - datetime.timedelta(days=i)).date()
      handler.stats_day_cls(key=handler.day_key(day), values_compressed=s).put()

    for i in xrange(10):
      s = stats._Snapshot(requests=10 + i)
      timestamp = (self.now - datetime.timedelta(hours=i))
      handler.stats_hour_cls(
          key=handler.hour_key(timestamp), values_compressed=s).put()

    for i in xrange(10):
      s = stats._Snapshot(requests=1 + i)
      timestamp = (self.now - datetime.timedelta(minutes=i))
      handler.stats_minute_cls(
          key=handler.minute_key(timestamp), values_compressed=s).put()

  def _assert_stats(self, resolution, page_size, time, expected):
    self._gen_stats()
    msg = isolated_pb2.StatsRequest()
    if time:
      msg.latest_time.FromDatetime(time)
    msg.resolution = resolution
    msg.page_size = page_size
    raw_resp = self.app.post(
        '/prpc/isolated.v1.Isolated/Stats', _encode(msg), self._headers)
    resp = isolated_pb2.StatsResponse()
    _decode(raw_resp.body, resp)
    self.assertEqual(expected, unicode(resp))

  def test_stats_bad_request_resolution(self):
    msg = isolated_pb2.StatsRequest()
    msg.page_size = 1
    raw_resp = self.app.post(
        '/prpc/isolated.v1.Isolated/Stats', _encode(msg), self._headers,
        expect_errors=True)
    self.assertEqual(raw_resp.body, 'Invalid resolution')

  def test_stats_bad_request_page_size(self):
    msg = isolated_pb2.StatsRequest()
    msg.resolution = isolated_pb2.MINUTE
    raw_resp = self.app.post(
        '/prpc/isolated.v1.Isolated/Stats', _encode(msg), self._headers,
        expect_errors=True)
    self.assertEqual(
        raw_resp.body, 'Invalid page_size; must be between 1 and 1000')

  def test_stats_day(self):
    # Limit the number of entities created.
    expected = (
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262390400\n'
      u'  }\n'
      u'  requests: 100\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262304000\n'
      u'  }\n'
      u'  requests: 101\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262217600\n'
      u'  }\n'
      u'  requests: 102\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262131200\n'
      u'  }\n'
      u'  requests: 103\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262044800\n'
      u'  }\n'
      u'  requests: 104\n'
      u'}\n')
    self._assert_stats(isolated_pb2.DAY, 5, None, expected)

  def test_stats_hour(self):
    # There are fewer entries than the requested limit.
    expected = (
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262401200\n'
      u'  }\n'
      u'  requests: 10\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {'
      u'\n'
      u'    seconds: 1262397600\n'
      u'  }\n'
      u'  requests: 11\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262394000\n'
      u'  }\n'
      u'  requests: 12\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262390400\n'
      u'  }\n'
      u'  requests: 13\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262386800\n'
      u'  }\n'
      u'  requests: 14\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262383200\n'
      u'  }\n'
      u'  requests: 15\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262379600\n'
      u'  }\n'
      u'  requests: 16\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262376000\n'
      u'  }\n'
      u'  requests: 17\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262372400\n'
      u'  }\n'
      u'  requests: 18\n'
      u'}\n'
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262368800\n'
      u'  }\n'
      u'  requests: 19\n'
      u'}\n')
    self._assert_stats(isolated_pb2.HOUR, 20, None, expected)

  def test_stats_minute(self):
    # Intentionally take one from the middle.
    expected = (
      u'measurements {\n'
      u'  start_time {\n'
      u'    seconds: 1262401140\n'
      u'  }\n'
      u'  requests: 6\n'
      u'}\n')
    now = self.now - datetime.timedelta(minutes=5)
    self._assert_stats(isolated_pb2.MINUTE, 1, now, expected)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
