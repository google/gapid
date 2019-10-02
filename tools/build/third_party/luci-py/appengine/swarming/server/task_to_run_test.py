#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import os
import random
import sys
import unittest

# Setups environment.
APP_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, APP_DIR)
import test_env_handlers

import webtest

from google.appengine.ext import ndb

import handlers_backend

from components import auth_testing
from components import utils
from test_support import test_case

from server import bot_management
from server import config
from server import task_queues
from server import task_request
from server import task_to_run


# pylint: disable=W0212
# Method could be a function - pylint: disable=R0201


def _gen_cipd(**kwargs):
  """Creates a CipdInputs."""
  args = {
    u'client_package': task_request.CipdPackage(
        package_name=u'infra/tools/cipd/${platform}',
        version=u'git_revision:deadbeef'),
    u'packages': [
      task_request.CipdPackage(
          package_name=u'rm',
          path=u'bin',
          version=u'git_revision:deadbeef'),
    ],
    u'server': u'https://chrome-infra-packages.appspot.com',
  }
  args.update(kwargs)
  return task_request.CipdInput(**args)


def _gen_properties(**kwargs):
  """Creates a TaskProperties."""
  args = {
    u'cipd_input': _gen_cipd(),
    u'command': [u'command1', u'arg1'],
    u'dimensions': {
      u'OS': [u'Windows-3.1.1'],
      u'hostname': [u'localhost'],
      u'pool': [u'default'],
    },
    u'env': {u'foo': u'bar', u'joe': u'2'},
    u'env_prefixes': {u'PATH': [u'local/path']},
    u'execution_timeout_secs': 24*60*60,
    u'grace_period_secs': 30,
    u'idempotent': False,
    u'inputs_ref': task_request.FilesRef(
        isolatedserver=u'https://isolateserver.appspot.com',
        namespace=u'default-gzip'),
    u'io_timeout_secs': None,
    u'has_secret_bytes': u'secret_bytes' in kwargs,
  }
  args.update(kwargs)
  args[u'dimensions_data'] = args.pop(u'dimensions')
  return task_request.TaskProperties(**args)


def _gen_request_slices(**kwargs):
  """Creates a TaskRequest."""
  now = utils.utcnow()
  args = {
    u'created_ts': now,
    u'manual_tags': [u'tag:1'],
    u'name': u'Request name',
    u'priority': 50,
    u'user': u'Jesus',
  }
  args.update(kwargs)
  req = task_request.TaskRequest(**args)
  task_request.init_new_request(req, True, task_request.TEMPLATE_AUTO)
  return req


def _gen_request(properties=None, **kwargs):
  """Creates a TaskRequest with one task slice."""
  return _gen_request_slices(
    task_slices=[
      task_request.TaskSlice(
          expiration_secs=60,
          properties=properties or _gen_properties()),
    ],
    **kwargs)


def _yield_next_available_task_to_dispatch(bot_dimensions, deadline):
  bot_id = bot_dimensions[u'id'][0]
  bot_management.bot_event(
      'bot_connected', bot_id, '1.2.3.4', 'joe@localhost',
      bot_dimensions, {'state': 'real'}, '1234', False, None, None, None)
  bot_root_key = bot_management.get_root_key(bot_id)
  task_queues.assert_bot_async(bot_root_key, bot_dimensions).get_result()
  return [
    to_run.to_dict()
    for _request, to_run in
        task_to_run.yield_next_available_task_to_dispatch(
            bot_dimensions, deadline)
  ]


class TaskToRunApiTest(test_env_handlers.AppTestBase):
  def setUp(self):
    super(TaskToRunApiTest, self).setUp()
    self.now = datetime.datetime(2014, 01, 02, 03, 04, 05, 06)
    self.mock_now(self.now)
    auth_testing.mock_get_current_identity(self)
    # Setup the backend to handle task queues.
    self.app = webtest.TestApp(
        handlers_backend.create_application(True),
        extra_environ={
          'REMOTE_ADDR': self.source_ip,
          'SERVER_SOFTWARE': os.environ['SERVER_SOFTWARE'],
        })
    self._enqueue_orig = self.mock(utils, 'enqueue_task', self._enqueue)
    cfg = config.settings()
    cfg.use_lifo = True
    self.mock(config, 'settings', lambda: cfg)

  def _enqueue(self, *args, **kwargs):
    return self._enqueue_orig(*args, use_dedicated_module=False, **kwargs)

  def mkreq(self, nb_task, req):
    """Stores a new initialized TaskRequest.

    nb_task is 1 or 0. It represents the number of GAE task queue
    rebuild-task-cache enqueued. It is 1 when the
    request.task_slice(0).properties.dimensions is new (unseen before) and a GAE
    task queue was enqueued to process it, 0 otherwise.
    """
    # It is important that the task queue to be asserted.
    task_queues.assert_task(req)
    self.assertEqual(nb_task, self.execute_tasks())
    req.key = task_request.new_request_key()
    req.put()
    return req

  def _gen_new_task_to_run(self, nb_task, **kwargs):
    """Returns TaskRequest, TaskToRun saved in the DB."""
    request = self.mkreq(nb_task, _gen_request(**kwargs))
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    to_run.put()
    return request, to_run

  def _gen_new_task_to_run_slices(self, nb_task, **kwargs):
    """Returns TaskRequest, TaskToRun saved in the DB."""
    request = self.mkreq(nb_task, _gen_request_slices(**kwargs))
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    to_run.put()
    return request, to_run

  def test_all_apis_are_tested(self):
    actual = frozenset(i[5:] for i in dir(self) if i.startswith('test_'))
    # Contains the list of all public APIs.
    expected = frozenset(
        i for i in dir(task_to_run)
        if i[0] != '_' and hasattr(getattr(task_to_run, i), 'func_name'))
    missing = expected - actual
    self.assertFalse(missing)

  def test_task_to_run_key_to_request_key(self):
    request = self.mkreq(1, _gen_request())
    task_key = task_to_run.request_to_task_to_run_key(request, 1, 0)
    actual = task_to_run.task_to_run_key_to_request_key(task_key)
    self.assertEqual(request.key, actual)

  def test_request_to_task_to_run_key(self):
    self.mock(random, 'getrandbits', lambda _: 0x88)
    request = self.mkreq(1, _gen_request())
    # Ensures that the hash value is constant for the same input.
    self.assertEqual(
        ndb.Key('TaskRequest', 0x7e296460f77ff77e, 'TaskToRun', 1),
        task_to_run.request_to_task_to_run_key(request, 1, 0))

  def test_gen_queue_number(self):
    # tuples of (input, expected).
    # 0x3fc00000 is the priority mask.
    # pylint: disable=bad-whitespace
    data = [
      # Priorities.
      ((1, '1970-01-01 00:00:00.000',   0), (0x92cc0300,  75)),
      ((1, '1970-01-01 00:00:00.000',   1), (0x930c0300,  76)),
      ((1, '1970-01-01 00:00:00.000',   2), (0x934c0300,  77)),
      ((1, '1970-01-01 00:00:00.000',   3), (0x938c0300,  78)),
      ((1, '1970-01-01 00:00:00.000', 255), (0xd28c0300, 330)),
      # Largest hash.
      ((0xffffffff, '1970-01-01 00:00:00.000', 255), (0x7fffffffd28c0300, 330)),
      # Time resolution.
      ((1, '1970-01-01 00:00:00.040',   0), (0x92cc0300,  75)),
      ((1, '1970-01-01 00:00:00.050',   0), (0x92cc0300,  75)),
      ((1, '1970-01-01 00:00:00.100',   0), (0x92cc02ff,  75)),
      ((1, '1970-01-01 00:00:00.900',   0), (0x92cc02f7,  75)),
      ((1, '1970-01-01 00:00:01.000',   0), (0x92cc02f6,  75)),  # 10
      ((1, '2010-01-02 03:04:05.060',   0), (0x92bd248d,  74)),
      ((1, '2010-01-02 03:04:05.060',   1), (0x92fd248d,  75)),
      ((1, '2010-12-31 23:59:59.999',   0), (0x80000000,   0)),
      ((1, '2010-12-31 23:59:59.999',   1), (0x80400000,   1)),
      ((1, '2010-12-31 23:59:59.999',   2), (0x80800000,   2)),
      ((1, '2010-12-31 23:59:59.999', 255), (0xbfc00000, 255)),
      # It's the end of the world as we know it...
      ((1, '9998-12-31 23:59:59.999',   0), (0x80000000,   0)),
      ((1, '9998-12-31 23:59:59.999',   1), (0x80400000,   1)),
      ((1, '9998-12-31 23:59:59.999', 255), (0xbfc00000, 255)),
    ]
    # pylint: enable=bad-whitespace
    for i, (
      (dimensions_hash, timestamp, priority),
      (expected_v, expected_p)) in enumerate(data):
      d = datetime.datetime.strptime(timestamp, '%Y-%m-%d %H:%M:%S.%f')
      actual = task_to_run._gen_queue_number(
          dimensions_hash, d, priority)
      self.assertEqual((i, '0x%016x' % expected_v), (i, '0x%016x' % actual))
      # Ensure we can extract the priority back. That said, it is corrupted by
      # time.
      v = task_to_run.TaskToRun(queue_number=actual)
      self.assertEqual(
          (i, expected_p), (i, task_to_run._queue_number_priority(v)))

  def test_new_task_to_run(self):
    self.mock(random, 'getrandbits', lambda _: 0x12)
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    data = _gen_request_slices(
        task_slices=[
          task_request.TaskSlice(
              expiration_secs=31,
              properties=_gen_properties(
                  command=[u'command1', u'arg1'],
                  dimensions=request_dimensions,
                  env={u'foo': u'bar'},
                  execution_timeout_secs=30)),
          task_request.TaskSlice(
              expiration_secs=30,
              properties=_gen_properties(
                  command=[u'command2'],
                  dimensions=request_dimensions,
                  execution_timeout_secs=30)),
        ],
        priority=20,
        created_ts=self.now)
    request = self.mkreq(2, data)
    # request.created_ts is used.
    self.mock_now(self.now, 1)
    expected = {
      'created_ts': self.now,
      'expiration_ts': self.now + datetime.timedelta(seconds=31),
      'queue_number': '0x1a3aa66317bd248e',
      'task_slice_index': 0,
      'try_number': 1,
    }
    actual = task_to_run.new_task_to_run(request, 1, 0).to_dict()
    self.assertEqual(expected, actual)
    # now is used when try_number != 1.
    expected['created_ts'] = self.now + datetime.timedelta(seconds=1)
    expected['try_number'] = 2
    actual = task_to_run.new_task_to_run(request, 2, 0).to_dict()
    self.assertEqual(expected, actual)
    # now is used when task_slice_index != 0.
    expected['task_slice_index'] = 1
    expected['try_number'] = 1
    expected['expiration_ts'] = self.now + datetime.timedelta(
        minutes=1, seconds=1)
    actual = task_to_run.new_task_to_run(request, 1, 1).to_dict()
    self.assertEqual(expected, actual)

  def test_new_task_to_run_limits(self):
    # Generate a TaskRequest with eight TaskSlice.
    slices = [
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(
              dimensions={u'pool': [u'default'], u'v': [unicode(i)]}))
      for i in xrange(8)
    ]
    request = self.mkreq(8, _gen_request_slices(task_slices=slices))
    with self.assertRaises(AssertionError):
      task_to_run.new_task_to_run(request, 0, 0)
    task_to_run.new_task_to_run(request, 1, 0)
    task_to_run.new_task_to_run(request, 2, 0)
    with self.assertRaises(AssertionError):
      task_to_run.new_task_to_run(request, 3, 0)
    task_to_run.new_task_to_run(request, 1, 7)
    with self.assertRaises(IndexError):
      task_to_run.new_task_to_run(request, 1, 8)

  def test_task_to_run_key_slice_index(self):
    slices = [
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(
              dimensions={u'pool': [u'default'], u'v': [unicode(i)]}))
      for i in xrange(8)
    ]
    request = self.mkreq(len(slices), _gen_request_slices(task_slices=slices))
    for i in xrange(len(slices)):
      to_run = task_to_run.new_task_to_run(request, 1, i)
      self.assertEqual(i, to_run.task_slice_index)
      self.assertEqual(i, task_to_run.task_to_run_key_slice_index(to_run.key))

  def test_task_to_run_key_try_number(self):
    slices = [
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(
              dimensions={u'pool': [u'default'], u'v': [unicode(i)]}))
      for i in xrange(8)
    ]
    request = self.mkreq(len(slices), _gen_request_slices(task_slices=slices))
    for i in (1, 2):
      to_run = task_to_run.new_task_to_run(request, i, 0)
      self.assertEqual(i, to_run.try_number)
      self.assertEqual(i, task_to_run.task_to_run_key_try_number(to_run.key))

  def test_new_task_to_run_list(self):
    self.mock(random, 'getrandbits', lambda _: 0x12)
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    data = _gen_request_slices(
        priority=20,
        created_ts=self.now,
        task_slices=[
          task_request.TaskSlice(
              expiration_secs=31,
              properties=_gen_properties(
                  command=[u'command1', u'arg1'],
                  dimensions=request_dimensions,
                  env={u'foo': u'bar'},
                  execution_timeout_secs=30)),
        ])
    request = self.mkreq(1, data)
    task_to_run.new_task_to_run(request, 1, 0).put()

    # Create a second with higher priority.
    self.mock(random, 'getrandbits', lambda _: 0x23)
    data = _gen_request_slices(
        priority=10,
        created_ts=self.now,
        task_slices=[
          task_request.TaskSlice(
            expiration_secs=31,
            properties=_gen_properties(
                command=[u'command1', u'arg1'],
                dimensions=request_dimensions,
                env={u'foo': u'bar'},
                execution_timeout_secs=30)),
        ])
    task_to_run.new_task_to_run(self.mkreq(0, data), 1, 0).put()

    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(seconds=31),
        'request_key': '0x7e296460f77ffdce',
        # Lower priority value means higher priority.
        'queue_number': '0x1a3aa663153d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(seconds=31),
        'request_key': '0x7e296460f77ffede',
        'queue_number': '0x1a3aa66317bd248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]

    def flatten(i):
      out = i.to_dict()
      out['request_key'] = '0x%016x' % i.request_key.integer_id()
      return out

    # Warning: Ordering by key doesn't work because of TaskToRunShard; e.g.
    # the entity key ordering DOES NOT correlate with .queue_number
    # Ensure they come out in expected order.
    q = task_to_run.TaskToRun.query().order(task_to_run.TaskToRun.queue_number)
    self.assertEqual(expected, map(flatten, q.fetch()))

  def test_match_dimensions(self):
    data_true = (
      ({}, {}),
      ({}, {'a': 'b'}),
      ({'a': ['b']}, {'a': ['b']}),
      ({'os': ['amiga']}, {'os': ['amiga', 'amiga-3.1']}),
      ( {'os': ['amiga'], 'foo': ['bar']},
        {'os': ['amiga', 'amiga-3.1'], 'a': 'b', 'foo': 'bar'}),
      ( {'os': ['amiga', 'amiga-3.1'], 'foo': ['bar']},
        {'os': ['amiga', 'amiga-3.1'], 'a': 'b', 'foo': 'bar'}),
    )

    for request_dimensions, bot_dimensions in data_true:
      self.assertEqual(
          True,
          task_to_run.match_dimensions(request_dimensions, bot_dimensions))

    data_false = (
      ({'os': ['amiga']}, {'os': ['Win', 'Win-3.1']}),
    )
    for request_dimensions, bot_dimensions in data_false:
      self.assertEqual(
          False,
          task_to_run.match_dimensions(request_dimensions, bot_dimensions))

  def test_yield_next_available_task_to_dispatch_none(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    # Bot declares no dimensions, so it will fail to match.
    bot_dimensions = {u'id': [u'bot1'], u'pool': [u'default']}
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    self.assertEqual([], actual)

  def test_yield_next_available_task_to_dispatch_none_mismatch(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    # Bot declares other dimensions, so it will fail to match.
    bot_dimensions = {
      u'id': [u'bot1'],
      u'os': [u'Windows-3.0'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    self.assertEqual([], actual)

  def test_yield_next_available_task_to_dispatch(self):
    request_dimensions = {
      u'foo': [u'bar'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    _request, _ = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    # Bot declares exactly same dimensions so it matches.
    bot_dimensions = request_dimensions.copy()
    bot_dimensions[u'id'] = [u'bot1']
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x613fbb331f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_dispatch_subset(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    _request, _ = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    # Bot declares more dimensions than needed, this is fine and it matches.
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa6631f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_shard(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    _request, _ = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    bot_dimensions = request_dimensions.copy()
    bot_dimensions[u'id'] = [u'bot1']
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa6631f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_dispatch_subset_multivalue(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    _request, _ = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    # Bot declares more dimensions than needed.
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows', u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa6631f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_dispatch_multi_normal(self):
    # Task added one after the other, normal case.
    request_dimensions_1 = {
      u'foo': [u'bar'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    _request1, _ = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions_1))

    # It's normally time ordered.
    self.mock_now(self.now, 1)
    request_dimensions_2 = {u'id': [u'localhost'], u'pool': [u'default']}
    _request2, _ = self._gen_new_task_to_run(
        0, properties=_gen_properties(dimensions=request_dimensions_2))

    bot_dimensions = {
      u'foo': [u'bar'],
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x613fbb331f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now + datetime.timedelta(seconds=1),
        'expiration_ts': self.now + datetime.timedelta(minutes=1, seconds=1),
        'queue_number': '0x5385bf749f3d2484',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    # There is a significant risk of non-determinism.
    self.assertEqual(sorted(expected), sorted(actual))

  def test_yield_next_available_task_to_dispatch_clock_skew(self):
    # Asserts that a TaskToRun added later in the DB (with a Key with an higher
    # value) but with a timestamp sooner (for example, time desynchronization
    # between machines) is still returned in the timestamp order, e.g. priority
    # is done based on timestamps and priority only.
    request_dimensions_1 = {
      u'foo': [u'bar'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    _request1, _ = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions_1))

    # The second shard is added before the first, potentially because of a
    # desynchronized clock. It'll have lower priority.
    self.mock_now(self.now, -1)
    request_dimensions_2 = {u'id': [u'localhost'], u'pool': [u'default']}
    _request2, _ = self._gen_new_task_to_run(
        0, properties=_gen_properties(dimensions=request_dimensions_2))

    bot_dimensions = {
      u'foo': [u'bar'],
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    expected = [
      {
        'created_ts': self.now + datetime.timedelta(seconds=-1),
        # Due to time being late on the second requester frontend.
        'expiration_ts': self.now + datetime.timedelta(minutes=1, seconds=-1),
        'queue_number': '0x5385bf749f3d2498',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x613fbb331f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    # There is a significant risk of non-determinism.
    self.assertEqual(expected, sorted(actual, key=lambda x: x['queue_number']))

  def test_yield_next_available_task_to_dispatch_priority(self):
    # Tasks added earlier but with higher priority are returned first.
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    self._gen_new_task_to_run(
        1,
        properties=_gen_properties(dimensions=request_dimensions),
        priority=10)

    # This one is later but has lower priority.
    self.mock_now(self.now, 60)
    request = self.mkreq(
        0,
        _gen_request(
            properties=_gen_properties(dimensions=request_dimensions),
            priority=50))
    task_to_run.new_task_to_run(request, 1, 0).put()

    # It should return them all, in the expected order: highest priority
    # (lowest priority value) first.
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa663153d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now + datetime.timedelta(minutes=1),
        'expiration_ts': self.now + datetime.timedelta(minutes=2),
        'queue_number': '0x1a3aa6631f3d2236',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_dispatch_fifo(self):
    cfg = config.settings()
    cfg.use_lifo = False
    self.mock(config, 'settings', lambda: cfg)

    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    self._gen_new_task_to_run(
        1,
        properties=_gen_properties(dimensions=request_dimensions),
        priority=50)

    self.mock_now(self.now, 60)
    request = self.mkreq(
        0,
        _gen_request(
            properties=_gen_properties(dimensions=request_dimensions),
            priority=50))
    task_to_run.new_task_to_run(request, 1, 0).put()

    # It should return them all, in the expected order: first in, first out.
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa6630c8ede72',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now + datetime.timedelta(minutes=1),
        'expiration_ts': self.now + datetime.timedelta(minutes=2),
        'queue_number': '0x1a3aa6630c8ee0ca',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_dispatch_lifo(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    self._gen_new_task_to_run(
        1,
        properties=_gen_properties(dimensions=request_dimensions),
        priority=50)

    self.mock_now(self.now, 60)
    request = self.mkreq(
        0,
        _gen_request(
            properties=_gen_properties(dimensions=request_dimensions),
            priority=50))
    task_to_run.new_task_to_run(request, 1, 0).put()

    # It should return them all, in the expected order: last in, first out.
    expected = [
      {
        'created_ts': self.now + datetime.timedelta(minutes=1),
        'expiration_ts': self.now + datetime.timedelta(minutes=2),
        'queue_number': '0x1a3aa6631f3d2236',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa6631f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_dispatch_multi_priority(self):
    # High priority tasks added earlier with other dimensions are returned
    # first.
    request_dimensions_1 = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    _request1, _ = self._gen_new_task_to_run(
        1,
        properties=_gen_properties(dimensions=request_dimensions_1),
        priority=10)

    # This one is later but has lower priority.
    self.mock_now(self.now, 60)
    request_dimensions_2 = {u'id': [u'localhost'], u'pool': [u'default']}
    request2 = self.mkreq(
        0,
        _gen_request(
            properties=_gen_properties(dimensions=request_dimensions_2),
            priority=50))
    task_to_run.new_task_to_run(request2, 1, 0).put()

    # It should return them all, in the expected order: highest priority
    # (lowest priority value) first.
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x1a3aa663153d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
      {
        'created_ts': self.now + datetime.timedelta(minutes=1),
        'expiration_ts': self.now + datetime.timedelta(minutes=2),
        'queue_number': '0x5385bf749f3d2236',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, None)
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_run_task_exceeds_deadline(self):
    request_dimensions = {
      u'foo': [u'bar'],
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    self._gen_new_task_to_run(
        0, properties=_gen_properties(dimensions=request_dimensions))
    # Bot declares exactly same dimensions so it matches.
    bot_dimensions = request_dimensions.copy()
    actual = _yield_next_available_task_to_dispatch(
        bot_dimensions, datetime.datetime(1969, 1, 1))
    self.failIf(actual)

  def test_yield_next_available_task_to_run_task_meets_deadline(self):
    request_dimensions = {
      u'foo': [u'bar'],
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    _request, _ = self._gen_new_task_to_run(
        properties=_gen_properties(dimensions=request_dimensions), nb_task=0)
    # Bot declares exactly same dimensions so it matches.
    bot_dimensions = request_dimensions.copy()
    actual = _yield_next_available_task_to_dispatch(
        bot_dimensions, datetime.datetime(3000, 1, 1))
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x3f6b0f051f3d248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    self.assertEqual(expected, actual)

  def test_yield_next_available_task_to_run_task_terminate(self):
    request_dimensions = {u'id': [u'fake-id']}
    _request, task = self._gen_new_task_to_run(
        0,
        priority=0,
        properties=_gen_properties(
            cipd_input=None,
            command=[],
            dimensions=request_dimensions,
            env=None,
            env_prefixes=None,
            inputs_ref=None,
            execution_timeout_secs=0,
            grace_period_secs=0))
    self.assertTrue(
        task.key.parent().get().task_slice(0).properties.is_terminate)
    # Bot declares exactly same dimensions so it matches.
    bot_dimensions = request_dimensions.copy()
    bot_dimensions[u'pool'] = [u'default']
    actual = _yield_next_available_task_to_dispatch(bot_dimensions, 0)
    expected = [
      {
        'created_ts': self.now,
        'expiration_ts': self.now + datetime.timedelta(minutes=1),
        'queue_number': '0x54795e3c92bd248e',
        'task_slice_index': 0,
        'try_number': 1,
      },
    ]
    self.assertEqual(expected, actual)

  def test_yield_expired_task_to_run(self):
    self._gen_new_task_to_run_slices(
        1,
        created_ts=self.now,
        task_slices=[
          {
            'expiration_secs': 60,
            'properties': _gen_properties(),
          },
        ])
    bot_dimensions = {u'id': [u'bot1'], u'pool': [u'default']}
    self.assertEqual(
        0,
        len(_yield_next_available_task_to_dispatch(bot_dimensions, None)))
    self.assertEqual(
        0, len(list(task_to_run.yield_expired_task_to_run())))

    # All tasks are now expired. Note that even if they still have .queue_number
    # set because the cron job wasn't run. They are still yielded by
    # yield_next_available_task_to_dispatch() because then task_scheduler can
    # expire them "inline" instead of waiting for a cron job.
    self.mock_now(self.now, 61)
    self.assertEqual(
        0, len(_yield_next_available_task_to_dispatch(bot_dimensions, None)))
    self.assertEqual(
        1, len(list(task_to_run.yield_expired_task_to_run())))

  def test_is_reapable(self):
    request_dimensions = {u'os': [u'Windows-3.1.1'], u'pool': [u'default']}
    _, to_run = self._gen_new_task_to_run(
        1, properties=_gen_properties(dimensions=request_dimensions))
    bot_dimensions = {
      u'id': [u'localhost'],
      u'os': [u'Windows-3.1.1'],
      u'pool': [u'default'],
    }
    self.assertEqual(
        1, len(_yield_next_available_task_to_dispatch(bot_dimensions, None)))

    self.assertEqual(True, to_run.is_reapable)
    to_run.queue_number = None
    to_run.put()
    self.assertEqual(False, to_run.is_reapable)

  def test_set_lookup_cache(self):
    # Create two TaskToRun on the same TaskRequest and assert that affecting one
    # negative cache entry doesn't affect the other.
    request = self.mkreq(1, _gen_request())
    to_run_1 = task_to_run.new_task_to_run(request, 1, 0)
    to_run_1.put()
    to_run_2 = task_to_run.new_task_to_run(request, 2, 0)
    to_run_2.put()
    lookup = lambda k: task_to_run._lookup_cache_is_taken_async(k).get_result()
    # By default, the negative cache is false, i.e. it is safe to reap the task.
    self.assertEqual(False, lookup(to_run_1.key))
    self.assertEqual(False, lookup(to_run_2.key))
    # Mark to_run_1 as safe to reap.
    task_to_run.set_lookup_cache(to_run_1.key, True)
    self.assertEqual(False, lookup(to_run_1.key))
    self.assertEqual(False, lookup(to_run_2.key))
    # Mark to_run_1 as unreapable, i.e. a bot is about to reap it.
    task_to_run.set_lookup_cache(to_run_1.key, False)
    self.assertEqual(True, lookup(to_run_1.key))
    self.assertEqual(False, lookup(to_run_2.key))
    task_to_run.set_lookup_cache(to_run_1.key, True)
    self.assertEqual(False, lookup(to_run_1.key))
    self.assertEqual(False, lookup(to_run_2.key))
    # Mark to_run_2 as unreapable, i.e. a bot is about to reap it.
    task_to_run.set_lookup_cache(to_run_2.key, False)
    self.assertEqual(False, lookup(to_run_1.key))
    self.assertEqual(True, lookup(to_run_2.key))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
