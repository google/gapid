#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Disable 'Access to a protected member ...'. NDB uses '_' for other purposes.
# pylint: disable=W0212

import datetime
import sys
import threading
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import utils
from test_support import test_case


class Rambling(ndb.Model):
  """Fake statistics."""
  a = ndb.IntegerProperty()
  b = ndb.FloatProperty()
  c = ndb.DateTimeProperty()
  d = ndb.DateProperty()

  def to_dict(self):
    out = super(Rambling, self).to_dict()
    out['e'] = datetime.timedelta(seconds=1.1)
    out['f'] = '\xc4\xa9'
    return out


class UtilsTest(test_case.TestCase):
  def test_json(self):
    r = Rambling(
        a=2,
        b=0.2,
        c=datetime.datetime(2012, 1, 2, 3, 4, 5, 6),
        d=datetime.date(2012, 1, 2))
    actual = utils.to_json_encodable([r])
    # Confirm that default is tight encoding and sorted keys.
    expected = [
      {
        'a': 2,
        'b': 0.2,
        'c': u'2012-01-02 03:04:05',
        'd': u'2012-01-02',
        'e': 1,
        'f': u'\u0129',
      },
    ]
    self.assertEqual(expected, actual)

    self.assertEqual([0, 1], utils.to_json_encodable(range(2)))
    self.assertEqual([0, 1], utils.to_json_encodable(i for i in (0, 1)))
    self.assertEqual([0, 1], utils.to_json_encodable(xrange(2)))

  def test_validate_root_service_url_dev_server(self):
    self.mock(utils, 'is_local_dev_server', lambda: True)
    utils.validate_root_service_url('https://blah')
    utils.validate_root_service_url('http://localhost:8080')

  def test_validate_root_service_url_gae(self):
    self.mock(utils, 'is_local_dev_server', lambda: False)
    utils.validate_root_service_url('https://blah')
    with self.assertRaises(ValueError):
      utils.validate_root_service_url('http://localhost:8080')

  def test_validate_root_service_bad(self):
    with self.assertRaises(ValueError):
      utils.validate_root_service_url('')
    with self.assertRaises(ValueError):
      utils.validate_root_service_url('blah://blah')
    with self.assertRaises(ValueError):
      utils.validate_root_service_url('https://')
    with self.assertRaises(ValueError):
      utils.validate_root_service_url('https://blah/')
    with self.assertRaises(ValueError):
      utils.validate_root_service_url('https://blah?asdad')

  def test_parse_rfc3339_datetime(self):
    # Sanity round-trip test with current time.
    now = utils.utcnow()
    parsed = utils.parse_rfc3339_datetime(now.isoformat() + 'Z')
    self.assertEqual(now, parsed)

    ok_cases = [
      ('2017-08-17T04:21:32.722952943Z', (2017, 8, 17, 4, 21, 32, 722953)),
      ('2017-08-17T04:21:32Z',           (2017, 8, 17, 4, 21, 32, 0)),
      ('1972-01-01T10:00:20.021-05:00',  (1972, 1, 1, 15, 0, 20, 21000)),
      ('1972-01-01T10:00:20.021+05:00',  (1972, 1, 1, 5, 0, 20, 21000)),
      ('1985-04-12T23:20:50.52Z',        (1985, 4,  12, 23, 20, 50, 520000)),
      ('1996-12-19T16:39:57-08:00',      (1996, 12, 20,  0, 39, 57,  0)),
    ]
    for val, expected in ok_cases:
      parsed = utils.parse_rfc3339_datetime(val)
      self.assertIsNone(parsed.tzinfo)
      self.assertEqual(datetime.datetime(*expected), parsed)

    bad_cases = [
      '',
      '1985-04-12T23:20:50.52',            # no timezone
      '2017:08:17T04:21:32Z',              # bad base format
      '2017-08-17T04:21:32.7229529431Z' ,  # more than nano second precision
      '2017-08-17T04:21:32Zblah',          # trailing data
      '1972-01-01T10:00:20.021-0500',      # bad timezone format
    ]
    for val in bad_cases:
      with self.assertRaises(ValueError):
        utils.parse_rfc3339_datetime(val)

  def test_datetime_to_rfc2822(self):
    self.assertEqual(
      'Mon, 02 Jan 2012 03:04:05 -0000',
      utils.datetime_to_rfc2822(datetime.datetime(2012, 1, 2, 3, 4, 5)))

  def test_milliseconds_since_epoch(self):
    self.mock_now(datetime.datetime(1970, 1, 2, 3, 4, 5, 6789))
    delta = utils.milliseconds_since_epoch(None)
    self.assertEqual(97445007, delta)

  def test_cache(self):
    calls = []

    @utils.cache
    def get_me():
      calls.append(1)
      return len(calls)

    self.assertEqual(1, get_me())
    self.assertEqual(1, get_me())
    self.assertEqual(1, len(calls))

  def test_cache_with_tasklets(self):
    @utils.cache
    def f():
      ndb.sleep(0).wait()  # Yield thread.
      return 1

    @ndb.tasklet
    def g():
      yield ()  # Make g a generator.
      raise ndb.Return(f())

    def test():
      ndb.Future.wait_all([(g()), (g())])

    t = threading.Thread(target=test)
    t.daemon = True
    t.start()
    t.join(1)
    if t.is_alive():
      self.fail('deadlock')

  def test_cache_with_expiration(self):
    ran = []
    self.mock(utils, 'time_time', lambda: 1000)

    @utils.cache_with_expiration(30)
    def do_work():
      ran.append(1)
      return len(ran)

    self.assertEqual(30, do_work.__parent_cache__.expiration_sec)
    self.assertEqual(None, do_work.__parent_cache__.expires)

    self.assertEqual(1, do_work())
    self.assertEqual(1, do_work())
    self.assertEqual(1, len(ran))
    self.assertEqual(1030, do_work.__parent_cache__.expires)

    utils.clear_cache(do_work)
    self.assertEqual(2, do_work())
    self.assertEqual(2, len(ran))
    self.assertEqual(1030, do_work.__parent_cache__.expires)

    self.mock(utils, 'time_time', lambda: 1029)
    self.assertEqual(2, do_work())
    self.assertEqual(2, do_work())
    self.assertEqual(2, len(ran))

    self.mock(utils, 'time_time', lambda: 1030)
    self.assertEqual(3, do_work())
    self.assertEqual(3, do_work())
    self.assertEqual(3, len(ran))
    self.assertEqual(1060, do_work.__parent_cache__.expires)

  def test_clear_cache(self):
    calls = []

    @utils.cache
    def get_me():
      calls.append(1)
      return len(calls)

    self.assertEqual(1, get_me())
    utils.clear_cache(get_me)
    self.assertEqual(2, get_me())
    self.assertEqual(2, len(calls))


class FakeNdbContext(object):
  def __init__(self):
    self.get_calls = []
    self.set_calls = []
    self.cached_value = None

  @ndb.tasklet
  def memcache_get(self, key):
    self.get_calls.append(key)
    raise ndb.Return(self.cached_value)

  # pylint: disable=redefined-outer-name
  @ndb.tasklet
  def memcache_set(self, key, value, time=None):
    self.cached_value = value
    self.set_calls.append((key, value, time))


class MemcacheTest(test_case.TestCase):

  def setUp(self):
    super(MemcacheTest, self).setUp()

    self.f_calls = []
    self.f_value = 'value'
    self.ctx = FakeNdbContext()
    self.mock(ndb, 'get_context', lambda: self.ctx)

  @utils.memcache('f', ['a', 'b', 'c', 'd'], time=54)
  def f(self, a, b, c=3, d=4, e=5):
    self.f_calls.append((a, b, c, d, e))
    return self.f_value

  @utils.memcache_async('f', ['a', 'b', 'c', 'd'], time=54)
  @ndb.tasklet
  def f_async(self, a, b, c=3, d=4, e=5):
    self.f_calls.append((a, b, c, d, e))
    raise ndb.Return(self.f_value)

  def test_async(self):
    self.f_async(1, 2, 3, 4, 5).get_result()
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [(1, 2, 3, 4, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 3, 4]', ('value',), 54)])

  def test_call(self):
    self.f(1, 2, 3, 4, 5)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [(1, 2, 3, 4, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 3, 4]', ('value',), 54)])

    self.ctx.get_calls = []
    self.f_calls = []
    self.ctx.set_calls = []
    self.f(1, 2, 3, 4)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [])
    self.assertEqual(self.ctx.set_calls, [])

  def test_none(self):
    self.f_value = None
    self.assertEqual(self.f(1, 2, 3, 4), None)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [(1, 2, 3, 4, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 3, 4]', (None,), 54)])

    self.ctx.get_calls = []
    self.f_calls = []
    self.ctx.set_calls = []
    self.assertEqual(self.f(1, 2, 3, 4), None)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [])
    self.assertEqual(self.ctx.set_calls, [])


  def test_call_without_optional_arg(self):
    self.f(1, 2)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [(1, 2, 3, 4, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 3, 4]', ('value',), 54)])

  def test_call_kwargs(self):
    self.f(1, 2, c=30, d=40)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 30, 40]'])
    self.assertEqual(self.f_calls, [(1, 2, 30, 40, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 30, 40]', ('value',), 54)])

  def test_call_all_kwargs(self):
    self.f(a=1, b=2, c=30, d=40)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 30, 40]'])
    self.assertEqual(self.f_calls, [(1, 2, 30, 40, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 30, 40]', ('value',), 54)])

  def test_call_packed_args(self):
    self.f(*[1, 2])
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 3, 4]'])
    self.assertEqual(self.f_calls, [(1, 2, 3, 4, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 3, 4]', ('value',), 54)])

  def test_call_packed_kwargs(self):
    self.f(1, 2, **{'c':30, 'd': 40})
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 30, 40]'])
    self.assertEqual(self.f_calls, [(1, 2, 30, 40, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 30, 40]', ('value',), 54)])

  def test_call_packed_both(self):
    self.f(*[1, 2], **{'c':30, 'd': 40})
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[1, 2, 30, 40]'])
    self.assertEqual(self.f_calls, [(1, 2, 30, 40, 5)])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[1, 2, 30, 40]', ('value',), 54)])

  def test_empty_key_arg(self):
    @utils.memcache('f')
    def f(a):
      # pylint: disable=unused-argument
      return 1

    f(1)
    self.assertEqual(self.ctx.get_calls, ['utils.memcache/v1a/f[]'])
    self.assertEqual(
        self.ctx.set_calls,
        [('utils.memcache/v1a/f[]', (1,), None)])

  def test_nonexisting_arg(self):
    with self.assertRaises(KeyError):
      # pylint: disable=unused-variable
      @utils.memcache('f', ['b'])
      def f(a):
        # pylint: disable=unused-argument
        pass

  def test_invalid_args(self):
    with self.assertRaises(TypeError):
      # pylint: disable=no-value-for-parameter
      self.f()

    with self.assertRaises(TypeError):
      # pylint: disable=no-value-for-parameter
      self.f(b=3)

    with self.assertRaises(TypeError):
      # pylint: disable=unexpected-keyword-arg
      self.f(1, 2, x=3)

  def test_args_prohibited(self):
    with self.assertRaises(NotImplementedError):
      # pylint: disable=unused-variable
      @utils.memcache('f', [])
      def f(a, *args):
        # pylint: disable=unused-argument
        pass

  def test_kwargs_prohibited(self):
    with self.assertRaises(NotImplementedError):
      # pylint: disable=unused-variable
      @utils.memcache('f', [])
      def f(**kwargs):
        # pylint: disable=unused-argument
        pass


class FingerprintTest(test_case.TestCase):
  def test_get_token_fingerprint(self):
    self.assertEqual(
        '8b7df143d91c716ecfa5fc1730022f6b',
        utils.get_token_fingerprint(u'blah'))


class AsyncApplyTest(test_case.TestCase):
  def test_ordered(self):
    items = range(3)

    @ndb.tasklet
    def fn_async(x):
      raise ndb.Return(x + 10)

    expected = [(0, 10), (1, 11), (2, 12)]
    actual = utils.async_apply(items, fn_async)
    self.assertFalse(isinstance(actual, list))
    self.assertEqual(expected, list(actual))

  def test_ordered_concurrent_jobs(self):
    items = range(4)

    log = []

    @ndb.tasklet
    def fn_async(x):
      log.append('%d started' % x)
      yield ndb.sleep(0.01)
      log.append('%d finishing' % x)
      raise ndb.Return(x + 10)

    expected = [(i, i + 10) for i in items]
    actual = utils.async_apply(items, fn_async, concurrent_jobs=2)
    self.assertFalse(isinstance(actual, list))
    self.assertEqual(expected, list(actual))
    self.assertEqual(log, [
        '0 started',
        '1 started',
        '0 finishing',
        '2 started',
        '1 finishing',
        '3 started',
        '2 finishing',
        '3 finishing',
    ])

  def test_unordered(self):
    items = [10, 5, 0]

    @ndb.tasklet
    def fn_async(x):
      yield ndb.sleep(float(x) / 1000)
      raise ndb.Return(x)

    expected = [(0, 0), (5, 5), (10, 10)]
    actual = utils.async_apply(items, fn_async, unordered=True)
    self.assertFalse(isinstance(actual, list))
    self.assertEqual(expected, list(actual))

  def test_unordered_first_concurrent_jobs(self):
    items = [10, 5, 0]

    log = []

    @ndb.tasklet
    def fn_async(x):
      log.append('%d started' % x)
      yield ndb.sleep(float(x) / 1000)
      log.append('%d finishing' % x)
      raise ndb.Return(x)

    expected = [(5, 5), (0, 0), (10, 10)]
    actual = utils.async_apply(
        items, fn_async, concurrent_jobs=2, unordered=True)
    self.assertFalse(isinstance(actual, list))
    self.assertEqual(expected, list(actual))
    self.assertEqual(log, [
        '10 started',
        '5 started',
        '5 finishing',
        '0 started',
        '0 finishing',
        '10 finishing',
    ])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
