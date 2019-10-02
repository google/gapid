#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import calendar
import datetime
import sys
import time
import unittest

# pylint: disable=no-self-argument,relative-import,ungrouped-imports
# pylint: disable=wrong-import-position
from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

import webapp2
import webtest

from components import stats_framework
from components.stats_framework import stats_logs
from components import utils
from test_support import stats_framework_logs_mock
from test_support import test_case


# TODO(maruel): Split stats_logs specific test into a separate unit test.


class InnerSnapshot(ndb.Model):
  c = ndb.StringProperty(default='', indexed=False)

  def accumulate(self, rhs):
    return stats_framework.accumulate(self, rhs, [])


class Snapshot(ndb.Model):
  """Fake statistics."""
  requests = ndb.IntegerProperty(default=0, indexed=False)
  b = ndb.FloatProperty(default=0, indexed=False)
  inner = ndb.LocalStructuredProperty(InnerSnapshot)
  d = ndb.StringProperty(repeated=True)

  def __init__(self, **kwargs):
    # This is the recommended way to use ndb.LocalStructuredProperty inside a
    # snapshot.
    #
    # Warning: The only reason it works is because Snapshot is itself inside a
    # ndb.LocalStructuredProperty.
    kwargs.setdefault('inner', InnerSnapshot())
    super(Snapshot, self).__init__(**kwargs)

  def accumulate(self, rhs):
    # accumulate() specifically handles where default value is a class instance.
    return stats_framework.accumulate(self, rhs, ['d'])


def get_now():
  """Returns an hard coded 'utcnow'.

  It is important because the timestamp near the hour or day limit could induce
  the creation of additional 'hours' stats, which would make the test flaky.
  """
  return datetime.datetime(2010, 1, 2, 3, 4, 5, 6)


def strip_seconds(timestamp):
  """Returns timestamp with seconds and microseconds stripped."""
  return datetime.datetime(*timestamp.timetuple()[:5], second=0)


class StatsFrameworkTest(test_case.TestCase):
  def test_empty(self):
    handler = stats_framework.StatisticsFramework(
        'test_framework', Snapshot, self.fail)

    self.assertEqual(0, stats_framework.StatsRoot.query().count())
    self.assertEqual(0, handler.stats_day_cls.query().count())
    self.assertEqual(0, handler.stats_hour_cls.query().count())
    self.assertEqual(0, handler.stats_minute_cls.query().count())

  def test_too_recent(self):
    # Other tests assumes TOO_RECENT == 1. Update accordingly if needed.
    self.assertEquals(1, stats_framework.TOO_RECENT)

  def test_framework_empty(self):
    handler = stats_framework.StatisticsFramework(
        'test_framework', Snapshot, self.fail)
    now = get_now()
    self.mock_now(now, 0)
    handler._set_last_processed_time(strip_seconds(now))
    i = handler.process_next_chunk(0)
    self.assertEqual(0, i)
    self.assertEqual(1, stats_framework.StatsRoot.query().count())
    self.assertEqual(0, handler.stats_day_cls.query().count())
    self.assertEqual(0, handler.stats_hour_cls.query().count())
    self.assertEqual(0, handler.stats_minute_cls.query().count())
    root = handler.root_key.get()
    self.assertEqual(strip_seconds(now), root.timestamp)

  def test_framework_fresh(self):
    # Ensures the processing will run for 120 minutes starting
    # StatisticsFramework.MAX_BACKTRACK days ago.
    called = []

    def gen_data(start, end):
      """Returns fake statistics."""
      self.assertEqual(start + 60, end)
      called.append(start)
      return Snapshot(
          requests=1, b=1, inner=InnerSnapshot(c='%d,' % len(called)))

    handler = stats_framework.StatisticsFramework(
        'test_framework', Snapshot, gen_data)

    now = get_now()
    self.mock_now(now, 0)
    start_date = now - datetime.timedelta(
        days=handler._max_backtrack_days)
    limit = handler._max_minutes_per_process

    i = handler.process_next_chunk(5)
    self.assertEqual(limit, i)

    # Fresh new stats gathering always starts at midnight.
    midnight = datetime.datetime(*start_date.date().timetuple()[:3])
    expected_calls = [
      calendar.timegm((midnight + datetime.timedelta(minutes=i)).timetuple())
      for i in range(limit)
    ]
    self.assertEqual(expected_calls, called)

    # Verify root.
    root = stats_framework.StatsRoot.query().fetch()
    self.assertEqual(1, len(root))
    # When timestamp is not set, it starts at the begining of the day,
    # MAX_BACKTRACK days ago, then process MAX_MINUTES_PER_PROCESS.
    timestamp = midnight + datetime.timedelta(seconds=(limit - 1)*60)
    expected = {
      'created': now,
      'timestamp': timestamp,
    }
    self.assertEqual(expected, root[0].to_dict())

    # Verify days.
    expected = [
      {
        'key': midnight.date(),
        'requests': limit,
        'b': float(limit),
        'd': [],
        'inner': {
          'c': u''.join('%d,' % i for i in xrange(1, limit + 1)),
        },
      }
    ]
    days = handler.stats_day_cls.query().fetch()
    self.assertEqual(expected, [d.to_dict() for d in days])
    # These are left out from .to_dict().
    self.assertEqual(now, days[0].created)
    self.assertEqual(now, days[0].modified)
    self.assertEqual(3, days[0].hours_bitmap)

    # Verify hours.
    expected = [
      {
        'key': (midnight + datetime.timedelta(seconds=i*60*60)),
        'requests': 60,
        'b': 60.,
        'd': [],
        'inner': {
          'c': u''.join(
              '%d,' % i for i in xrange(60 * i + 1, 60 * (i + 1) + 1)),
        },
      }
      for i in range(limit / 60)
    ]
    hours = handler.stats_hour_cls.query().fetch()
    self.assertEqual(expected, [d.to_dict() for d in hours])
    for h in hours:
      # These are left out from .to_dict().
      self.assertEqual(now, h.created)
      self.assertEqual((1<<60)-1, h.minutes_bitmap)

    # Verify minutes.
    expected = [
      {
        'key': (midnight + datetime.timedelta(seconds=i*60)),
        'requests': 1,
        'b': 1.,
        'd': [],
        'inner': {
          'c': u'%d,' % (i + 1),
        },
      } for i in range(limit)
    ]
    minutes = handler.stats_minute_cls.query().fetch()
    self.assertEqual(expected, [d.to_dict() for d in minutes])
    for m in minutes:
      # These are left out from .to_dict().
      self.assertEqual(now, m.created)

  def test_framework_last_few(self):
    called = []

    def gen_data(start, end):
      """Returns fake statistics."""
      self.assertEqual(start + 60, end)
      called.append(start)
      return Snapshot(
          requests=1, b=1, inner=InnerSnapshot(c='%d,' % len(called)))

    handler = stats_framework.StatisticsFramework(
        'test_framework', Snapshot, gen_data)

    now = get_now()
    self.mock_now(now, 0)
    handler._set_last_processed_time(
        strip_seconds(now) - datetime.timedelta(seconds=3*60))
    i = handler.process_next_chunk(1)
    self.assertEqual(2, i)
    self.assertEqual(1, stats_framework.StatsRoot.query().count())
    self.assertEqual(1, handler.stats_day_cls.query().count())
    self.assertEqual(1, handler.stats_hour_cls.query().count())
    self.assertEqual(2, handler.stats_minute_cls.query().count())
    root = handler.root_key.get()
    self.assertEqual(
        strip_seconds(now) - datetime.timedelta(seconds=60), root.timestamp)

    # Trying to process more won't do anything.
    i = handler.process_next_chunk(1)
    self.assertEqual(0, i)
    root = handler.root_key.get()
    self.assertEqual(
        strip_seconds(now) - datetime.timedelta(seconds=60), root.timestamp)

    expected = [
      {
        'key': now.date(),
        'requests': 0,
        'b': 0.0,
        'd': [],
        'inner': {'c': u''},
      },
    ]
    self.assertEqual(
        expected, stats_framework.get_stats(handler, 'days', now, 100, True))

    expected = [
      {
        'key': datetime.datetime(*now.timetuple()[:4]),
        'requests': 2,
        'b': 2.0,
        'd': [],
        'inner': {'c': u'1,2,'},
      },
    ]
    self.assertEqual(
        expected, stats_framework.get_stats(handler, 'hours', now, 100, True))

    expected = [
      {
        'key': datetime.datetime(
            *(now - datetime.timedelta(seconds=60)).timetuple()[:5]),
        'requests': 1,
        'b': 1.0,
        'd': [],
        'inner': {'c': u'2,'},
      },
      {
        'key': datetime.datetime(
            *(now - datetime.timedelta(seconds=120)).timetuple()[:5]),
        'requests': 1,
        'b': 1.0,
        'd': [],
        'inner': {'c': u'1,'},
      },
    ]
    self.assertEqual(
        expected, stats_framework.get_stats(handler, 'minutes', now, 100, True))

  def test_keys(self):
    handler = stats_framework.StatisticsFramework(
        'test_framework', Snapshot, self.fail)
    date = datetime.datetime(2010, 1, 2)
    self.assertEqual(
        ndb.Key('StatsRoot', 'test_framework', 'StatsDay', '2010-01-02'),
        handler.day_key(date.date()))

    self.assertEqual(
        ndb.Key(
          'StatsRoot', 'test_framework',
          'StatsDay', '2010-01-02',
          'StatsHour', '00'),
        handler.hour_key(date))
    self.assertEqual(
        ndb.Key(
          'StatsRoot', 'test_framework',
          'StatsDay', '2010-01-02',
          'StatsHour', '00',
          'StatsMinute', '00'),
        handler.minute_key(date))

  def test_yield_empty(self):
    self.testbed.init_modules_stub()
    self.assertEqual(0, len(list(stats_logs.yield_entries(None, None))))


def generate_snapshot(start_time, end_time):
  values = Snapshot()
  for entry in stats_logs.yield_entries(start_time, end_time):
    values.requests += 1
    for l in entry.entries:
      values.inner.c += l
  return values


class StatsFrameworkLogTest(test_case.TestCase):
  def setUp(self):
    super(StatsFrameworkLogTest, self).setUp()
    stats_framework_logs_mock.configure(self)
    self.h = stats_framework.StatisticsFramework(
        'test_framework', Snapshot, generate_snapshot)

    class GenerateHandler(webapp2.RequestHandler):
      def get(self2):
        stats_logs.add_entry('Hello')
        self2.response.write('Yay')

    class JsonHandler(webapp2.RequestHandler):
      def get(self2):
        self2.response.headers['Content-Type'] = (
            'application/json; charset=utf-8')
        duration = int(self2.request.get('duration', 120))
        now = self2.request.get('now')
        resolution = self2.request.get('resolution')
        data = stats_framework.get_stats(
            self.h, resolution, now, duration, True)
        self2.response.write(stats_framework.utils.encode_to_json(data))

    routes = [
        ('/generate', GenerateHandler),
        ('/json', JsonHandler),
    ]
    real_app = webapp2.WSGIApplication(routes, debug=True)
    self.app = webtest.TestApp(
        real_app, extra_environ={'REMOTE_ADDR': 'fake-ip'})
    self.now = datetime.datetime(2010, 1, 2, 3, 4, 5, 6)
    self.mock_now(self.now, 0)

  def test_yield_entries(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)

    self.assertEqual(0, len(list(stats_logs.yield_entries(None, None))))
    self.assertEqual(0, len(list(stats_logs.yield_entries(1, time.time()))))

    self.assertEqual('Yay', self.app.get('/generate').body)

    self.assertEqual(1, len(list(stats_logs.yield_entries(None, None))))
    self.assertEqual(1, len(list(stats_logs.yield_entries(1, time.time()))))
    self.assertEqual(
        0, len(list(stats_logs.yield_entries(None, utils.time_time()))))

  def test_json_empty_days(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.assertEqual([], self.app.get('/json?resolution=days&duration=9').json)

  def test_json_empty_hours(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.assertEqual([], self.app.get('/json?resolution=hours&duration=9').json)

  def test_json_empty_minutes(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.assertEqual(
        [], self.app.get('/json?resolution=minutes&duration=9').json)

  def test_json_empty_processed_days(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=days&duration=9').json)

  def test_json_empty_processed_hours(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02 03:00:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 02:00:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=hours&duration=9').json)

  def test_json_empty_processed_minutes(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02 03:04:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 03:03:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 03:02:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 03:01:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 03:00:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 02:59:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 02:58:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 02:57:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 02:56:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 02:55:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=minutes&duration=11').json)

  def test_json_empty_processed_minutes_limited(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02 03:04:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 03:03:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
      {
        u'key': u'2010-01-02 03:02:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=minutes&duration=3').json)

  def test_json_two_days(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.assertEqual('Yay', self.app.get('/generate').body)
    self.assertEqual('Yay', self.app.get('/generate').body)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u''},
        u'requests': 0,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=days&duration=9').json)

  def test_json_two_hours(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.assertEqual('Yay', self.app.get('/generate').body)
    self.assertEqual('Yay', self.app.get('/generate').body)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02 03:00:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u'HelloHello'},
        u'requests': 2,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=hours&duration=1').json)

  def test_json_two_minutes(self):
    stats_framework_logs_mock.reset_timestamp(self.h, self.now)
    self.assertEqual('Yay', self.app.get('/generate').body)
    self.assertEqual('Yay', self.app.get('/generate').body)
    self.h.process_next_chunk(0)
    expected = [
      {
        u'key': u'2010-01-02 03:04:00',
        u'b': 0.0,
        u'd': [],
        u'inner': {u'c': u'HelloHello'},
        u'requests': 2,
      },
    ]
    self.assertEqual(
        expected, self.app.get('/json?resolution=minutes&duration=1').json)

  def test_accumulate(self):
    a = Snapshot(
        requests=23,
        b=0.1,
        inner=InnerSnapshot(c='foo'),
        d=['a', 'b'])
    b = Snapshot(
        requests=None,
        b=None,
        inner=InnerSnapshot(c=None),
        d=['c', 'd'])
    a.accumulate(b)
    expected = {
      'b': 0.1,
      'd': ['a', 'b'],
      'inner': {'c': 'foo'},
      'requests': 23,
    }
    self.assertEqual(expected, a.to_dict())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
