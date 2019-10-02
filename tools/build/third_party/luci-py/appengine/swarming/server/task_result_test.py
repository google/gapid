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

import test_env
test_env.setup_test_env()

from google.protobuf import duration_pb2

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

import webtest

from components import auth_testing
from components import utils
from test_support import test_case

from proto.api import swarming_pb2  # pylint: disable=no-name-in-module
from server import bq_state
from server import large
from server import task_pack
from server import task_request
from server import task_result
from server import task_to_run


# pylint: disable=W0212


def _gen_properties(**kwargs):
  """Creates a TaskProperties."""
  args = {
    'command': [u'command1'],
    'dimensions': {u'pool': [u'default']},
    'env': {},
    'execution_timeout_secs': 24*60*60,
    'io_timeout_secs': None,
  }
  args.update(kwargs or {})
  args['dimensions_data'] = args.pop('dimensions')
  return task_request.TaskProperties(**args)


def _gen_request_slice(**kwargs):
  """Creates a TaskRequest."""
  now = utils.utcnow()
  args = {
    'created_ts': now,
    'manual_tags': [u'tag:1'],
    'name': 'Request name',
    'priority': 50,
    'task_slices': [
      task_request.TaskSlice(expiration_secs=60, properties=_gen_properties()),
    ],
    'user': 'Jesus',
  }
  args.update(kwargs)
  ret = task_request.TaskRequest(**args)
  task_request.init_new_request(ret, True, task_request.TEMPLATE_AUTO)
  ret.key = task_request.new_request_key()
  ret.put()
  return ret


def _gen_request(properties=None, **kwargs):
  """Creates a TaskRequest."""
  return _gen_request_slice(
      task_slices=[
        task_request.TaskSlice(
            expiration_secs=60,
            properties=properties or _gen_properties()),
      ],
      **kwargs)


def _gen_summary_result(**kwargs):
  """Creates a TaskRunResult."""
  request = _gen_request(**kwargs)
  result_summary = task_result.new_result_summary(request)
  result_summary.modified_ts = utils.utcnow()
  ndb.transaction(result_summary.put)
  return result_summary.key.get()


def _gen_run_result(**kwargs):
  """Creates a TaskRunResult."""
  result_summary = _gen_summary_result(**kwargs)
  request = result_summary.request_key.get()
  to_run = task_to_run.new_task_to_run(request, 1, 0)
  run_result = task_result.new_run_result(
      request, to_run, 'localhost', 'abc', {})
  run_result.started_ts = result_summary.modified_ts
  run_result.modified_ts = utils.utcnow()
  ndb.transaction(
      lambda: result_summary.set_from_run_result(run_result, request))
  ndb.transaction(lambda: ndb.put_multi((result_summary, run_result)))
  return run_result.key.get()


def _safe_cmp(a, b):
  # cmp(datetime.datetime.utcnow(), None) throws TypeError. Workaround.
  return cmp(utils.encode_to_json(a), utils.encode_to_json(b))


def get_entities(entity_model):
  return sorted(
      (i.to_dict() for i in entity_model.query().fetch()), cmp=_safe_cmp)


class TestCase(test_case.TestCase):
  APP_DIR = test_env.APP_DIR

  def setUp(self):
    super(TestCase, self).setUp()
    auth_testing.mock_get_current_identity(self)


class TaskResultApiTest(TestCase):
  def setUp(self):
    super(TaskResultApiTest, self).setUp()
    self.now = datetime.datetime(2014, 1, 2, 3, 4, 5, 6)
    self.mock_now(self.now)
    self.mock(random, 'getrandbits', lambda _: 0x88)

  def assertEntities(self, expected, entity_model):
    self.assertEqual(expected, get_entities(entity_model))

  def _gen_summary(self, **kwargs):
    """Returns TaskResultSummary.to_dict()."""
    out = {
      'abandoned_ts': None,
      'bot_dimensions': None,
      'bot_id': None,
      'bot_version': None,
      'cipd_pins': None,
      'children_task_ids': [],
      'completed_ts': None,
      'costs_usd': [],
      'cost_saved_usd': None,
      'created_ts': self.now,
      'current_task_slice': 0,
      'deduped_from': None,
      'duration': None,
      'exit_code': None,
      'failure': False,
      # Constant due to the mock of both utils.utcnow() and
      # random.getrandbits().
      'id': '1d69b9f088008810',
      'internal_failure': False,
      'modified_ts': None,
      'name': u'Request name',
      'outputs_ref': None,
      'server_versions': [u'v1a'],
      'started_ts': None,
      'state': task_result.State.PENDING,
      'tags': [
        u'pool:default',
        u'priority:50',
        u'service_account:none',
        u'swarming.pool.template:no_config',
        u'tag:1',
        u'user:Jesus',
      ],
      'try_number': None,
      'user': u'Jesus',
    }
    out.update(kwargs)
    return out

  def _gen_result(self, **kwargs):
    """Returns TaskRunResult.to_dict()."""
    out = {
      'abandoned_ts': None,
      'bot_dimensions': {u'id': [u'localhost'], u'foo': [u'bar', u'biz']},
      'bot_id': u'localhost',
      'bot_version': u'abc',
      'children_task_ids': [],
      'cipd_pins': None,
      'completed_ts': None,
      'cost_usd': 0.,
      'current_task_slice': 0,
      'duration': None,
      'exit_code': None,
      'failure': False,
      # Constant due to the mock of both utils.utcnow() and
      # random.getrandbits().
      'id': '1d69b9f088008811',
      'internal_failure': False,
      'killing': None,
      'modified_ts': None,
      'outputs_ref': None,
      'server_versions': [u'v1a'],
      'started_ts': None,
      'state': task_result.State.RUNNING,
      'try_number': 1,
    }
    out.update(kwargs)
    return out

  def test_all_apis_are_tested(self):
    # Ensures there's a test for each public API.
    module = task_result
    expected = set(
        i for i in dir(module)
        if i[0] != '_' and hasattr(getattr(module, i), 'func_name'))
    missing = expected - set(i[5:] for i in dir(self) if i.startswith('test_'))
    self.assertFalse(missing)

  def test_State(self):
    for i in task_result.State.STATES:
      self.assertTrue(task_result.State.to_string(i))
    with self.assertRaises(ValueError):
      task_result.State.to_string(0)

    self.assertEqual(
        set(task_result.State._NAMES), set(task_result.State.STATES))
    items = (
      task_result.State.STATES_RUNNING + task_result.State.STATES_DONE +
      task_result.State.STATES_ABANDONED)
    self.assertEqual(set(items), set(task_result.State.STATES))
    self.assertEqual(len(items), len(set(items)))

  def test_state_to_string(self):
    # Same code as State.to_string() except that it works for
    # TaskResultSummary too.
    class Foo(ndb.Model):
      deduped_from = None
      state = task_result.StateProperty()
      failure = ndb.BooleanProperty(default=False)
      internal_failure = ndb.BooleanProperty(default=False)

    for i in task_result.State.STATES:
      self.assertTrue(task_result.State.to_string(i))
    for i in task_result.State.STATES:
      self.assertTrue(task_result.state_to_string(Foo(state=i)))
    f = Foo(state=task_result.State.COMPLETED)
    f.deduped_from = '123'
    self.assertEqual('Deduped', task_result.state_to_string(f))

  def test_new_result_summary(self):
    request = _gen_request()
    actual = task_result.new_result_summary(request)
    actual.modified_ts = self.now
    # Trigger _pre_put_hook().
    actual.put()
    expected = self._gen_summary(modified_ts=self.now)
    self.assertEqual(expected, actual.to_dict())
    self.assertEqual(50, actual.request.priority)
    self.assertEqual(True, actual.can_be_canceled)
    actual.state = task_result.State.RUNNING
    self.assertEqual(True, actual.can_be_canceled)
    actual.state = task_result.State.TIMED_OUT
    actual.duration = 0.1
    actual.completed_ts = self.now
    self.assertEqual(False, actual.can_be_canceled)

    actual.children_task_ids = [
      '1d69ba3ea8008810', '3d69ba3ea8008810', '2d69ba3ea8008810',
    ]
    actual.modified_ts = utils.utcnow()
    ndb.transaction(actual.put)
    expected = [u'1d69ba3ea8008810', u'2d69ba3ea8008810', u'3d69ba3ea8008810']
    self.assertEqual(expected, actual.key.get().children_task_ids)

  def test_new_run_result(self):
    request = _gen_request()
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    actual = task_result.new_run_result(
        request, to_run, u'localhost', u'abc',
        {u'id': [u'localhost'], u'foo': [u'bar', u'biz']})
    actual.modified_ts = self.now
    actual.started_ts = self.now
    # Trigger _pre_put_hook().
    actual.put()
    expected = self._gen_result(modified_ts=self.now, started_ts=self.now)
    self.assertEqual(expected, actual.to_dict())
    self.assertEqual(50, actual.request.priority)
    self.assertEqual(True, actual.can_be_canceled)
    self.assertEqual(0, actual.current_task_slice)

  def test_new_run_result_duration_no_exit_code(self):
    request = _gen_request()
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    actual = task_result.new_run_result(
        request, to_run, u'localhost', u'abc',
        {u'id': [u'localhost'], u'foo': [u'bar', u'biz']})
    actual.completed_ts = self.now
    actual.modified_ts = self.now
    actual.started_ts = self.now
    actual.duration = 1.
    actual.state = task_result.State.COMPLETED
    # Trigger _pre_put_hook().
    with self.assertRaises(datastore_errors.BadValueError):
      actual.put()
    actual.state = task_result.State.TIMED_OUT
    actual.put()
    expected = self._gen_result(
        completed_ts=self.now, duration=1., modified_ts=self.now, failure=True,
        started_ts=self.now, state=task_result.State.TIMED_OUT)
    self.assertEqual(expected, actual.to_dict())

  def test_integration(self):
    # Creates a TaskRequest, along its TaskResultSummary and TaskToRun. Have a
    # bot reap the task, and complete the task. Ensure the resulting
    # TaskResultSummary and TaskRunResult are properly updated.
    request = _gen_request()
    result_summary = task_result.new_result_summary(request)
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    result_summary.modified_ts = utils.utcnow()
    ndb.transaction(lambda: ndb.put_multi([result_summary, to_run]))
    expected = self._gen_summary(modified_ts=self.now)
    self.assertEqual(expected, result_summary.to_dict())

    # Nothing changed 2 secs later except latency.
    self.mock_now(self.now, 2)
    self.assertEqual(expected, result_summary.to_dict())

    # Task is reaped after 2 seconds (4 secs total).
    reap_ts = self.now + datetime.timedelta(seconds=4)
    self.mock_now(reap_ts)
    to_run.queue_number = None
    to_run.put()
    run_result = task_result.new_run_result(
        request, to_run, u'localhost', u'abc', {})
    run_result.started_ts = utils.utcnow()
    run_result.modified_ts = run_result.started_ts
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result)))
    expected = self._gen_summary(
        bot_dimensions={},
        bot_version=u'abc',
        bot_id=u'localhost',
        costs_usd=[0.],
        modified_ts=reap_ts,
        state=task_result.State.RUNNING,
        started_ts=reap_ts,
        try_number=1)
    self.assertEqual(expected, result_summary.key.get().to_dict())

    # Task completed after 2 seconds (6 secs total), the task has been running
    # for 2 seconds.
    complete_ts = self.now + datetime.timedelta(seconds=6)
    self.mock_now(complete_ts)
    run_result.completed_ts = complete_ts
    run_result.duration = 0.1
    run_result.exit_code = 0
    run_result.state = task_result.State.COMPLETED
    run_result.modified_ts = utils.utcnow()
    task_result.PerformanceStats(
        key=task_pack.run_result_key_to_performance_stats_key(run_result.key),
        bot_overhead=0.1,
        isolated_download=task_result.OperationStats(
            duration=0.05, initial_number_items=10, initial_size=10000,
            items_cold=large.pack([1, 2]),
            items_hot=large.pack([3, 4, 5])),
        isolated_upload=task_result.OperationStats(
            duration=0.01,
            items_cold=large.pack([10]))).put()
    ndb.transaction(lambda: ndb.put_multi(run_result.append_output('foo', 0)))
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result)))
    expected = self._gen_summary(
        bot_dimensions={},
        bot_version=u'abc',
        bot_id=u'localhost',
        completed_ts=complete_ts,
        costs_usd=[0.],
        duration=0.1,
        exit_code=0,
        modified_ts=complete_ts,
        state=task_result.State.COMPLETED,
        started_ts=reap_ts,
        try_number=1)
    self.assertEqual(expected, result_summary.key.get().to_dict())
    expected = {
      'bot_overhead': 0.1,
      'isolated_download': {
        'duration': 0.05,
        'initial_number_items': 10,
        'initial_size': 10000,
        'items_cold': large.pack([1, 2]),
        'items_hot': large.pack([3, 4, 5]),
        'num_items_cold': 2,
        'total_bytes_items_cold': 3,
        'num_items_hot': 3,
        'total_bytes_items_hot': 12,
      },
      'isolated_upload': {
        'duration': 0.01,
        'initial_number_items': None,
        'initial_size': None,
        'items_cold': large.pack([10]),
        'items_hot': None,
        'num_items_cold': 1,
        'total_bytes_items_cold': 10,
        'num_items_hot': None,
        'total_bytes_items_hot': None,
      },
      'package_installation': {
        'duration': None,
        'initial_number_items': None,
        'initial_size': None,
        'items_cold': None,
        'items_hot': None,
        'num_items_cold': None,
        'total_bytes_items_cold': None,
        'num_items_hot': None,
        'total_bytes_items_hot': None,
      },
    }
    self.assertEqual(expected, result_summary.performance_stats.to_dict())
    self.assertEqual('foo', result_summary.get_output())
    self.assertEqual(
        datetime.timedelta(seconds=2),
        result_summary.duration_as_seen_by_server)
    self.assertEqual(
        datetime.timedelta(seconds=0.1),
        result_summary.duration_now(utils.utcnow()))
    self.assertEqual(
        datetime.timedelta(seconds=4), result_summary.pending)
    self.assertEqual(
        datetime.timedelta(seconds=4),
        result_summary.pending_now(utils.utcnow()))

    self.assertEqual(
        task_pack.pack_result_summary_key(result_summary.key),
        result_summary.task_id)
    self.assertEqual(complete_ts, result_summary.ended_ts)
    self.assertEqual(
        task_pack.pack_run_result_key(run_result.key),
        run_result.task_id)
    self.assertEqual(complete_ts, run_result.ended_ts)

  def test_yield_run_result_keys_with_dead_bot(self):
    request = _gen_request()
    result_summary = task_result.new_result_summary(request)
    result_summary.modified_ts = utils.utcnow()
    ndb.transaction(result_summary.put)
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    run_result = task_result.new_run_result(
        request, to_run, 'localhost', 'abc', {})
    run_result.started_ts = utils.utcnow()
    run_result.completed_ts = run_result.started_ts
    run_result.modified_ts = run_result.started_ts
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi((run_result, result_summary)))

    self.mock_now(self.now + task_result.BOT_PING_TOLERANCE)
    self.assertEqual(
        [], list(task_result.yield_run_result_keys_with_dead_bot()))

    self.mock_now(self.now + task_result.BOT_PING_TOLERANCE, 1)
    self.assertEqual(
        [run_result.key],
        list(task_result.yield_run_result_keys_with_dead_bot()))

  def test_set_from_run_result(self):
    request = _gen_request()
    result_summary = task_result.new_result_summary(request)
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    run_result = task_result.new_run_result(
        request, to_run, 'localhost', 'abc', {})
    run_result.started_ts = utils.utcnow()
    self.assertTrue(result_summary.need_update_from_run_result(run_result))
    result_summary.modified_ts = utils.utcnow()
    run_result.modified_ts = utils.utcnow()
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result)))

    self.assertTrue(result_summary.need_update_from_run_result(run_result))
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi([result_summary]))

    self.assertFalse(result_summary.need_update_from_run_result(run_result))

  def test_set_from_run_result_two_server_versions(self):
    request = _gen_request()
    result_summary = task_result.new_result_summary(request)
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    run_result = task_result.new_run_result(
        request, to_run, 'localhost', 'abc', {})
    run_result.started_ts = utils.utcnow()
    self.assertTrue(result_summary.need_update_from_run_result(run_result))
    result_summary.modified_ts = utils.utcnow()
    run_result.modified_ts = utils.utcnow()
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result)))

    self.assertTrue(result_summary.need_update_from_run_result(run_result))
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi([result_summary]))

    run_result.signal_server_version('new-version')
    run_result.modified_ts = utils.utcnow()
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result)))
    self.assertEqual(
        ['v1a', 'new-version'], run_result.key.get().server_versions)
    self.assertEqual(
        ['v1a', 'new-version'], result_summary.key.get().server_versions)

  def test_set_from_run_result_two_tries(self):
    request = _gen_request()
    result_summary = task_result.new_result_summary(request)
    to_run_1 = task_to_run.new_task_to_run(request, 1, 0)
    run_result_1 = task_result.new_run_result(
        request, to_run_1, 'localhost', 'abc', {})
    run_result_1.started_ts = utils.utcnow()
    to_run_2 = task_to_run.new_task_to_run(request, 2, 0)
    run_result_2 = task_result.new_run_result(
        request, to_run_2, 'localhost', 'abc', {})
    run_result_2.started_ts = utils.utcnow()
    self.assertTrue(result_summary.need_update_from_run_result(run_result_1))
    run_result_2.modified_ts = utils.utcnow()
    result_summary.modified_ts = utils.utcnow()
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result_2)))

    self.assertTrue(result_summary.need_update_from_run_result(run_result_1))
    run_result_1.modified_ts = utils.utcnow()
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result_1, request))
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result_1)))

    result_summary = result_summary.key.get()
    self.assertFalse(result_summary.need_update_from_run_result(run_result_1))

    self.assertTrue(result_summary.need_update_from_run_result(run_result_2))
    run_result_2.modified_ts = utils.utcnow()
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result_2, request))
    ndb.transaction(lambda: ndb.put_multi((result_summary, run_result_2)))
    result_summary = result_summary.key.get()

    self.assertEqual(2, result_summary.try_number)
    self.assertFalse(result_summary.need_update_from_run_result(run_result_1))

  def test_run_result_duration(self):
    run_result = task_result.TaskRunResult(
        started_ts=datetime.datetime(2010, 1, 1, 0, 0, 0),
        completed_ts=datetime.datetime(2010, 1, 1, 0, 2, 0))
    self.assertEqual(
        datetime.timedelta(seconds=120), run_result.duration_as_seen_by_server)
    self.assertEqual(
        datetime.timedelta(seconds=120),
        run_result.duration_now(utils.utcnow()))

    run_result = task_result.TaskRunResult(
        started_ts=datetime.datetime(2010, 1, 1, 0, 0, 0),
        abandoned_ts=datetime.datetime(2010, 1, 1, 0, 1, 0))
    self.assertEqual(None, run_result.duration_as_seen_by_server)
    self.assertEqual(None, run_result.duration_now(utils.utcnow()))

  def test_run_result_timeout(self):
    request = _gen_request()
    result_summary = task_result.new_result_summary(request)
    result_summary.modified_ts = utils.utcnow()
    ndb.transaction(result_summary.put)
    to_run = task_to_run.new_task_to_run(request, 1, 0)
    run_result = task_result.new_run_result(
        request, to_run, 'localhost', 'abc', {})
    run_result.state = task_result.State.TIMED_OUT
    run_result.duration = 0.1
    run_result.exit_code = -1
    run_result.started_ts = utils.utcnow()
    run_result.completed_ts = run_result.started_ts
    run_result.modified_ts = run_result.started_ts
    ndb.transaction(
        lambda: result_summary.set_from_run_result(run_result, request))
    ndb.transaction(lambda: ndb.put_multi((run_result, result_summary)))
    run_result = run_result.key.get()
    result_summary = result_summary.key.get()
    self.assertEqual(True, run_result.failure)
    self.assertEqual(True, result_summary.failure)

  def test_result_task_state(self):
    def check(expected, **kwargs):
      self.assertEqual(
          expected, task_result.TaskResultSummary(**kwargs).task_state)

    # That's an incorrect state:
    check(swarming_pb2.TASK_STATE_INVALID, state=task_result.State.BOT_DIED)
    check(swarming_pb2.PENDING, state=task_result.State.PENDING)
    # https://crbug.com/915342: PENDING_DEDUPING
    check(swarming_pb2.RUNNING, state=task_result.State.RUNNING)
    # https://crbug.com/796757: RUNNING_OVERHEAD_SETUP
    # https://crbug.com/813412: RUNNING_OVERHEAD_TEARDOWN
    # https://crbug.com/916560: TERMINATING
    check(
        swarming_pb2.RAN_INTERNAL_FAILURE,
        internal_failure=True, state=task_result.State.BOT_DIED)
    # https://crbug.com/902807: DUT_FAILURE
    # https://crbug.com/916553: BOT_DISAPPEARED
    # https://crbug.com/916559: PREEMPTED
    check(swarming_pb2.COMPLETED, state=task_result.State.COMPLETED)
    check(swarming_pb2.TIMED_OUT, state=task_result.State.TIMED_OUT)
    # https://crbug.com/916556: TIMED_OUT_SILENCE
    check(swarming_pb2.KILLED, state=task_result.State.KILLED)
    # https://crbug.com/916553: MISSING_INPUTS
    check(
        swarming_pb2.DEDUPED,
        state=task_result.State.COMPLETED, deduped_from=u'123')
    check(swarming_pb2.EXPIRED, state=task_result.State.EXPIRED)
    check(swarming_pb2.CANCELED, state=task_result.State.CANCELED)
    check(swarming_pb2.NO_RESOURCE, state=task_result.State.NO_RESOURCE)
    # https://crbug.com/916562: LOAD_SHED
    # https://crbug.com/916557: RESOURCE_EXHAUSTED

  def test_to_proto(self):
    cipd_client_pkg = task_request.CipdPackage(
        package_name=u'infra/tools/cipd/${platform}',
        version=u'git_revision:deadbeef')
    run_result = _gen_run_result(
        properties=_gen_properties(
            cipd_input={
              u'client_package': cipd_client_pkg,
              u'packages': [
                task_request.CipdPackage(
                    package_name=u'rm',
                    path=u'bin',
                    version=u'latest'),
              ],
              u'server': u'http://localhost:2'
            },
        ),
    )
    run_result.started_ts = self.now + datetime.timedelta(seconds=20)
    run_result.abandoned_ts = self.now + datetime.timedelta(seconds=30)
    run_result.completed_ts = self.now + datetime.timedelta(seconds=40)
    run_result.modified_ts = self.now + datetime.timedelta(seconds=50)
    run_result.duration = 1.
    run_result.current_task_slice = 2
    run_result.exit_code = 1
    run_result.children_task_ids = [u'12310']
    run_result.outputs_ref = task_request.FilesRef(
        isolated=u'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
        isolatedserver=u'http://localhost:1',
        namespace=u'default-gzip')
    run_result.cipd_pins = task_result.CipdPins(
        client_package=cipd_client_pkg,
        packages=[
          task_request.CipdPackage(
              package_name=u'rm', path=u'bin', version=u'stable'),
        ])
    task_result.PerformanceStats(
        key=task_pack.run_result_key_to_performance_stats_key(run_result.key),
        bot_overhead=0.1,
        isolated_download=task_result.OperationStats(
            duration=0.05, initial_number_items=10, initial_size=10000,
            items_cold=large.pack([1, 2]),
            items_hot=large.pack([3, 4, 5])),
        isolated_upload=task_result.OperationStats(
            duration=0.01,
            items_cold=large.pack([10]))).put()

    # Note: It cannot be both TIMED_OUT and have run_result.deduped_from set.
    run_result.state = task_result.State.TIMED_OUT
    run_result.bot_dimensions = {u'id': [u'bot1'], u'pool': [u'default']}
    run_result.put()

    expected = swarming_pb2.TaskResult(
        request=swarming_pb2.TaskRequest(
            task_slices=[
              swarming_pb2.TaskSlice(
                  properties=swarming_pb2.TaskProperties(
                    cipd_inputs=[
                      swarming_pb2.CIPDPackage(
                          package_name=u'rm',
                          version=u'latest',
                          dest_path=u'bin',
                      ),
                    ],
                    command=[u'command1'],
                    dimensions=[
                      swarming_pb2.StringListPair(
                          key=u'pool', values=[u'default']),
                    ],
                    execution_timeout=duration_pb2.Duration(seconds=86400),
                    grace_period=duration_pb2.Duration(seconds=30),
                  ),
                  expiration=duration_pb2.Duration(seconds=60),
                  properties_hash=
                      '17934af0f7d694d0fb1720ff970709a5fd150d4e532083173b9e38ec'
                      'f027e563',
              ),
            ],
            priority=50,
            service_account=u'none',
            name=u'Request name',
            tags=[
              u'pool:default',
              u'priority:50',
              u'service_account:none',
              u'swarming.pool.template:no_config',
              u'tag:1',
              u'user:Jesus',
            ],
            user=u'Jesus',
            task_id=u'1d69b9f088008810',
        ),
        duration=duration_pb2.Duration(seconds=1),
        state=swarming_pb2.TIMED_OUT,
        state_category=swarming_pb2.CATEGORY_EXECUTION_DONE,
        try_number=1,
        current_task_slice=2,
        bot=swarming_pb2.Bot(
            bot_id=u'bot1',
            pools=[u'default'],
            dimensions=[
              swarming_pb2.StringListPair(key=u'id', values=[u'bot1']),
              swarming_pb2.StringListPair(key=u'pool', values=[u'default']),
            ],
        ),
        server_versions=[u'v1a'],
        children_task_ids=[u'12310'],
        #deduped_from=u'123410',
        task_id=u'1d69b9f088008810',
        run_id=u'1d69b9f088008811',
        cipd_pins=swarming_pb2.CIPDPins(
            server=u'http://localhost:2',
            client_package=swarming_pb2.CIPDPackage(
                package_name=u'infra/tools/cipd/${platform}',
                version=u'git_revision:deadbeef',
            ),
            packages=[
              swarming_pb2.CIPDPackage(
                  package_name=u'rm',
                  version=u'stable',
                  dest_path=u'bin',
              ),
            ],
        ),
        performance=swarming_pb2.TaskPerformance(
            other_overhead=duration_pb2.Duration(nanos=100000000),
            setup=swarming_pb2.TaskOverheadStats(
              duration=duration_pb2.Duration(nanos=50000000),
            ),
            teardown=swarming_pb2.TaskOverheadStats(
              duration=duration_pb2.Duration(nanos=10000000),
            ),
        ),
        exit_code=1,
        outputs=swarming_pb2.CASTree(
            digest=u'deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
            server=u'http://localhost:1',
            namespace=u'default-gzip')
    )
    expected.request.create_time.FromDatetime(self.now)
    expected.create_time.FromDatetime(self.now)
    expected.start_time.FromDatetime(
        self.now + datetime.timedelta(seconds=20))
    expected.abandon_time.FromDatetime(
        self.now + datetime.timedelta(seconds=30))
    expected.end_time.FromDatetime(self.now + datetime.timedelta(seconds=40))

    actual = swarming_pb2.TaskResult()
    run_result.to_proto(actual)
    self.assertEqual(unicode(expected), unicode(actual))

  def test_TaskResultSummary_to_proto_empty(self):
    # Assert that it doesn't throw on empty entity.
    actual = swarming_pb2.TaskResult()
    # It's unreasonable to expect the entity key to be unset, which complicates
    # this test a bit.
    req = task_request.TaskRequest(id=1230)
    res = task_result.TaskResultSummary(parent=req.key)
    res._request_cache = req
    res.to_proto(actual)
    expected = swarming_pb2.TaskResult(
        request=swarming_pb2.TaskRequest(task_id='7ffffffffffffb310'),
        state=swarming_pb2.PENDING,
        state_category=swarming_pb2.CATEGORY_PENDING,
        task_id='7ffffffffffffb310')
    self.assertEqual(expected, actual)

  def test_TaskRunResult_to_proto_empty(self):
    # Assert that it doesn't throw on empty entity.
    actual = swarming_pb2.TaskResult()
    # It's unreasonable to expect the entity key to be unset, which complicates
    # this test a bit.
    req = task_request.TaskRequest(id=1230)
    res_sum = task_result.TaskResultSummary(parent=req.key, id=1)
    res_sum._request_cache = req
    res = task_result.TaskRunResult(parent=res_sum.key, id=1)
    res._request_cache = req
    res.to_proto(actual)
    expected = swarming_pb2.TaskResult(
        request=swarming_pb2.TaskRequest(task_id='7ffffffffffffb310'),
        state=swarming_pb2.RUNNING,
        state_category=swarming_pb2.CATEGORY_RUNNING,
        try_number=1,
        task_id='7ffffffffffffb310',
        run_id='7ffffffffffffb311')
    self.assertEqual(expected, actual)

  def test_performance_stats_pre_put_hook(self):
    with self.assertRaises(datastore_errors.BadValueError):
      task_result.PerformanceStats().put()

  def test_cron_update_tags(self):
    # TODO(maruel): https://crbug.com/912154
    self.assertEqual(0, task_result.cron_update_tags())

  def _mock_send_to_bq(self, expected_table_name):
    payloads = []
    def send_to_bq(table_name, rows):
      self.assertEqual(expected_table_name, table_name)
      if rows:
        # When rows is empty, send_to_bq() can exit early.
        payloads.append(rows)
      return 0
    self.mock(bq_state, 'send_to_bq', send_to_bq)
    return payloads

  def test_task_bq_run_empty(self):
    # Empty, nothing is done.
    start = utils.utcnow()
    end = start+datetime.timedelta(seconds=60)
    self.assertEqual((0, 0), task_result.task_bq_run(start, end))

  def test_task_bq_run(self):
    payloads = self._mock_send_to_bq('task_results_run')

    # Generate 4 tasks results to test boundaries.
    self.mock_now(self.now, 10)
    run_result_1 = _gen_run_result()
    run_result_1.abandoned_ts = utils.utcnow()
    run_result_1.completed_ts = utils.utcnow()
    run_result_1.modified_ts = utils.utcnow()
    run_result_1.put()
    start = self.mock_now(self.now, 20)
    run_result_2 = _gen_run_result()
    run_result_2.completed_ts = utils.utcnow()
    run_result_2.modified_ts = utils.utcnow()
    run_result_2.put()
    end = self.mock_now(self.now, 30)
    run_result_3 = _gen_run_result()
    run_result_3.completed_ts = utils.utcnow()
    run_result_3.modified_ts = utils.utcnow()
    run_result_3.put()
    self.mock_now(self.now, 40)
    run_result_4 = _gen_run_result()
    run_result_4.completed_ts = utils.utcnow()
    run_result_4.modified_ts = utils.utcnow()
    run_result_4.put()

    self.assertEqual((2, 0), task_result.task_bq_run(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(2, len(actual_rows))
    expected = [
      run_result_2.task_id,
      run_result_3.task_id,
    ]
    self.assertEqual(expected, [r[0] for r in actual_rows])

  def test_task_bq_run_running(self):
    payloads = self._mock_send_to_bq('task_results_run')
    self.now = datetime.datetime(2019, 1, 1)
    start = self.mock_now(self.now, 0)
    run_result = _gen_run_result()
    run_result.started_ts = utils.utcnow()
    run_result.modified_ts = utils.utcnow()
    run_result.put()
    end = self.mock_now(self.now, 60)

    self.assertEqual((1, 0), task_result.task_bq_run(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(1, len(actual_rows))
    self.assertEqual([run_result.task_id], [r[0] for r in actual_rows])

  def test_task_bq_run_old_abandoned_ts(self):
    # Confirm that an old entity without completed_ts set is still found.
    payloads = self._mock_send_to_bq('task_results_run')
    self.now = task_result._COMPLETED_TS_CUTOFF - datetime.timedelta(days=1)
    start = self.mock_now(self.now, 0)
    run_result = _gen_run_result()
    run_result.abandoned_ts = utils.utcnow()
    run_result.modified_ts = utils.utcnow()
    run_result.put()
    self.assertIsNone(run_result.key.get().completed_ts)
    end = self.mock_now(self.now, 60)

    self.assertEqual((1, 0), task_result.task_bq_run(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(1, len(actual_rows))
    self.assertEqual([run_result.task_id], [r[0] for r in actual_rows])

  def test_task_bq_run_recent_abandoned_ts(self):
    # Confirm that a recent entity without completed_ts set is not found.
    payloads = self._mock_send_to_bq('task_results_run')
    self.now = task_result._COMPLETED_TS_CUTOFF + datetime.timedelta(days=1)
    start = self.mock_now(self.now, 0)
    run_result = _gen_run_result()
    # Make sure started_ts is not caught.
    run_result.started_ts = datetime.datetime(2010, 1, 1)
    run_result.abandoned_ts = utils.utcnow()
    run_result.modified_ts = utils.utcnow()
    run_result.put()
    self.assertIsNone(run_result.key.get().completed_ts)
    end = self.mock_now(self.now, 60)

    self.assertEqual((0, 0), task_result.task_bq_run(start, end))
    self.assertEqual(0, len(payloads), payloads)

  def test_task_bq_summary_empty(self):
    # Empty, nothing is done.
    start = utils.utcnow()
    end = start+datetime.timedelta(seconds=60)
    self.assertEqual((0, 0), task_result.task_bq_summary(start, end))

  def test_task_bq_summary(self):
    payloads = self._mock_send_to_bq('task_results_summary')

    # Generate 4 tasks results to test boundaries.
    self.mock_now(self.now, 10)
    result_1 = _gen_summary_result()
    result_1.abandoned_ts = utils.utcnow()
    result_1.completed_ts = utils.utcnow()
    result_1.modified_ts = utils.utcnow()
    result_1.put()
    start = self.mock_now(self.now, 20)
    result_2 = _gen_summary_result()
    result_2.completed_ts = utils.utcnow()
    result_2.modified_ts = utils.utcnow()
    result_2.put()
    end = self.mock_now(self.now, 30)
    result_3 = _gen_summary_result()
    result_3.completed_ts = utils.utcnow()
    result_3.modified_ts = utils.utcnow()
    result_3.put()
    self.mock_now(self.now, 40)
    result_4 = _gen_summary_result()
    result_4.completed_ts = utils.utcnow()
    result_4.modified_ts = utils.utcnow()
    result_4.put()

    self.assertEqual((2, 0), task_result.task_bq_summary(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(2, len(actual_rows))
    expected = [
      result_2.task_id,
      result_3.task_id,
    ]
    self.assertEqual(expected, [r[0] for r in actual_rows])

  def test_task_bq_summary_pending(self):
    payloads = self._mock_send_to_bq('task_results_summary')
    self.now = datetime.datetime(2019, 2, 28)
    start = self.mock_now(self.now, 0)
    result = _gen_summary_result()
    result.created_ts = utils.utcnow()
    result.modified_ts = utils.utcnow()
    result.put()
    end = self.mock_now(self.now, 60)

    self.assertEqual((1, 0), task_result.task_bq_summary(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(1, len(actual_rows))
    self.assertEqual([result.task_id], [r[0] for r in actual_rows])

  def test_task_bq_summary_running(self):
    payloads = self._mock_send_to_bq('task_results_summary')
    self.now = datetime.datetime(2019, 2, 28)
    start = self.mock_now(self.now, 0)
    result = _gen_summary_result()
    result.started_ts = utils.utcnow()
    result.modified_ts = utils.utcnow()
    result.put()
    end = self.mock_now(self.now, 60)

    self.assertEqual((1, 0), task_result.task_bq_summary(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(1, len(actual_rows))
    self.assertEqual([result.task_id], [r[0] for r in actual_rows])

  def test_task_bq_summary_old_abandoned_ts(self):
    # Confirm that an old entity without completed_ts set is still found.
    payloads = self._mock_send_to_bq('task_results_summary')
    self.now = task_result._COMPLETED_TS_CUTOFF - datetime.timedelta(days=1)
    start = self.mock_now(self.now, 0)
    result = _gen_summary_result()
    result.abandoned_ts = utils.utcnow()
    result.modified_ts = utils.utcnow()
    result.put()
    self.assertIsNone(result.key.get().completed_ts)
    end = self.mock_now(self.now, 60)

    self.assertEqual((1, 0), task_result.task_bq_summary(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(1, len(actual_rows))
    self.assertEqual([result.task_id], [r[0] for r in actual_rows])

  def test_task_bq_summary_recent_abandoned_ts(self):
    # Confirm that a recent entity without completed_ts set is not found.
    payloads = self._mock_send_to_bq('task_results_summary')
    self.now = task_result._COMPLETED_TS_CUTOFF + datetime.timedelta(days=1)
    start = self.mock_now(self.now, 0)
    result = _gen_summary_result()
    # Make sure neither created_ts and started_ts is caught.
    result.created_ts = datetime.datetime(2010, 1, 1)
    result.started_ts = datetime.datetime(2010, 1, 1)
    result.abandoned_ts = utils.utcnow()
    result.modified_ts = utils.utcnow()
    result.put()
    self.assertIsNone(result.key.get().completed_ts)
    end = self.mock_now(self.now, 60)

    self.assertEqual((0, 0), task_result.task_bq_summary(start, end))
    self.assertEqual(0, len(payloads), payloads)

  def test_get_result_summaries_query(self):
    # Indirectly tested by API.
    pass

  def test_get_run_results_query(self):
    # Indirectly tested by API.
    pass


class TestOutput(TestCase):
  def assertTaskOutputChunk(self, expected):
    q = task_result.TaskOutputChunk.query().order(
        task_result.TaskOutputChunk.key)
    self.assertEqual(expected, [t.to_dict() for t in q.fetch()])

  def test_append_output(self):
    run_result = _gen_run_result()
    # Test that one can stream output and it is returned fine.
    def run(*args):
      entities = run_result.append_output(*args)
      self.assertEqual(1, len(entities))
      ndb.put_multi(entities)
    run('Part1\n', 0)
    run('Part2\n', len('Part1\n'))
    run('Part3\n', len('Part1P\n'))
    self.assertEqual('Part1\nPPart3\n', run_result.get_output())

  def test_append_output_large(self):
    run_result = _gen_run_result()
    self.mock(logging, 'error', lambda *_: None)
    one_mb = '<3Google' * (1024*1024/8)

    def run(*args):
      entities = run_result.append_output(*args)
      # Asserts at least one entity was created.
      self.assertTrue(entities)
      ndb.put_multi(entities)

    for i in xrange(16):
      run(one_mb, i*len(one_mb))

    self.assertEqual(
        task_result.TaskOutput.FETCH_MAX_CONTENT, len(run_result.get_output()))

  def test_append_output_max_chunk(self):
    # This test case is very slow (1m25s locally) if running with the default
    # values, so scale it down a bit which results in ~2.5s.
    run_result = _gen_run_result()
    self.mock(
        task_result.TaskOutput, 'PUT_MAX_CONTENT',
        task_result.TaskOutput.PUT_MAX_CONTENT / 8)
    self.mock(
        task_result.TaskOutput, 'PUT_MAX_CHUNKS',
        task_result.TaskOutput.PUT_MAX_CHUNKS / 8)
    self.assertFalse(
        task_result.TaskOutput.PUT_MAX_CONTENT %
            task_result.TaskOutput.CHUNK_SIZE)

    calls = []
    self.mock(logging, 'warning', lambda *args: calls.append(args))
    max_chunk = 'x' * task_result.TaskOutput.PUT_MAX_CONTENT
    entities = run_result.append_output(max_chunk, 0)
    self.assertEqual(task_result.TaskOutput.PUT_MAX_CHUNKS, len(entities))
    ndb.put_multi(entities)
    self.assertEqual([], calls)

    # Try with PUT_MAX_CONTENT + 1 bytes, so the last byte is discarded.
    entities = run_result.append_output(max_chunk + 'x', 0)
    self.assertEqual(task_result.TaskOutput.PUT_MAX_CHUNKS, len(entities))
    ndb.put_multi(entities)
    self.assertEqual(1, len(calls))
    self.assertTrue(calls[0][0].startswith('Dropping '), calls[0][0])
    self.assertEqual(1, calls[0][1])

  def test_append_output_partial(self):
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output('Foo', 10))
    expected_output = '\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00Foo'
    self.assertEqual(expected_output, run_result.get_output())
    self.assertTaskOutputChunk([{'chunk': expected_output, 'gaps': [0, 10]}])

  def test_append_output_partial_hole(self):
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output('Foo', 0))
    ndb.put_multi(run_result.append_output('Bar', 10))
    expected_output = 'Foo\x00\x00\x00\x00\x00\x00\x00Bar'
    self.assertEqual(expected_output, run_result.get_output())
    self.assertTaskOutputChunk([{'chunk': expected_output, 'gaps': [3, 10]}])

  def test_append_output_partial_far(self):
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output(
      'Foo', 10 + task_result.TaskOutput.CHUNK_SIZE))
    self.assertEqual(
        '\x00' * (task_result.TaskOutput.CHUNK_SIZE + 10) + 'Foo',
        run_result.get_output())
    expected = [
      {'chunk': '\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00Foo', 'gaps': [0, 10]},
    ]
    self.assertTaskOutputChunk(expected)

  def test_append_output_partial_far_split(self):
    # Missing, writing happens on two different TaskOutputChunk entities.
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output(
        'FooBar', 2 * task_result.TaskOutput.CHUNK_SIZE - 3))
    self.assertEqual(
        '\x00' * (task_result.TaskOutput.CHUNK_SIZE * 2 - 3) + 'FooBar',
        run_result.get_output())
    expected = [
      {
        'chunk': '\x00' * (task_result.TaskOutput.CHUNK_SIZE - 3) + 'Foo',
        'gaps': [0, 102397],
      },
      {'chunk': 'Bar', 'gaps': []},
    ]
    self.assertTaskOutputChunk(expected)

  def test_append_output_overwrite(self):
    # Overwrite previously written data.
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output('FooBar', 0))
    ndb.put_multi(run_result.append_output('X', 3))
    self.assertEqual('FooXar', run_result.get_output())
    self.assertTaskOutputChunk([{'chunk': 'FooXar', 'gaps': []}])

  def test_append_output_reverse_order(self):
    # Write the data in reverse order in multiple calls.
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output('Wow', 11))
    ndb.put_multi(run_result.append_output('Foo', 8))
    ndb.put_multi(run_result.append_output('Baz', 0))
    ndb.put_multi(run_result.append_output('Bar', 4))
    expected_output = 'Baz\x00Bar\x00FooWow'
    self.assertEqual(expected_output, run_result.get_output())
    self.assertTaskOutputChunk(
        [{'chunk': expected_output, 'gaps': [3, 4, 7, 8]}])

  def test_append_output_reverse_order_second_chunk(self):
    # Write the data in reverse order in multiple calls.
    run_result = _gen_run_result()
    ndb.put_multi(run_result.append_output(
        'Wow', task_result.TaskOutput.CHUNK_SIZE + 11))
    ndb.put_multi(run_result.append_output(
        'Foo', task_result.TaskOutput.CHUNK_SIZE + 8))
    ndb.put_multi(run_result.append_output(
        'Baz', task_result.TaskOutput.CHUNK_SIZE + 0))
    ndb.put_multi(run_result.append_output(
        'Bar', task_result.TaskOutput.CHUNK_SIZE + 4))
    self.assertEqual(
        task_result.TaskOutput.CHUNK_SIZE * '\x00' + 'Baz\x00Bar\x00FooWow',
        run_result.get_output())
    self.assertTaskOutputChunk(
        [{'chunk': 'Baz\x00Bar\x00FooWow', 'gaps': [3, 4, 7, 8]}])


if __name__ == '__main__':
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
