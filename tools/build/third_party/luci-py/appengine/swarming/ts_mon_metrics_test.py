#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import json
import sys
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

import gae_ts_mon
from test_support import test_case

import ts_mon_metrics
from server import bot_management
from server import task_queues
from server import task_result


def _gen_task_result_summary(now, key_id, properties=None, **kwargs):
  """Creates a TaskRequest."""
  props = {
    'command': [u'command1'],
    'dimensions': {u'pool': u'default'},
    'env': {},
    'execution_timeout_secs': 24*60*60,
    'io_timeout_secs': None,
  }
  props.update(properties or {})
  args = {
    'created_ts': now,
    'modified_ts': now,
    'name': 'Request name',
    'tags': [u'tag:1'],
    'user': 'Jesus',
    'key': ndb.Key('TaskResultSummary', key_id),
  }
  args.update(kwargs)
  return task_result.TaskResultSummary(**args)


def _gen_bot_info(key_id, last_seen_ts, **kwargs):
  args = {
    'key': ndb.Key('BotRoot', key_id, 'BotInfo', 'info'),
    'last_seen_ts': last_seen_ts,
    'dimensions': {
        'os': ['Linux', 'Ubuntu'],
        'bot_id': [key_id],
    },
    'state': {},
  }
  args.update(**kwargs)
  args['dimensions_flat'] = task_queues.dimensions_to_flat(
      args.pop('dimensions'))
  return bot_management.BotInfo(**args)


class TestMetrics(test_case.TestCase):

  def setUp(self):
    super(TestMetrics, self).setUp()
    gae_ts_mon.reset_for_unittest()
    gae_ts_mon.initialize()
    self.now = datetime.datetime(2016, 4, 7)
    self.mock_now(self.now)

  def test_pool_from_dimensions(self):
    dimensions = {
        u'os': [u'Linux', u'Ubuntu', u'Ubuntu-14.04'],
        u'cpu': [u'x86', u'x86-64'],
    }
    dimensions.update(
        {k: 'ignored' for k in ts_mon_metrics._IGNORED_DIMENSIONS})
    expected = u'cpu:x86-64|os:Linux|os:Ubuntu-14.04'
    self.assertEqual(expected, ts_mon_metrics._pool_from_dimensions(dimensions))

  def test_shard_params(self):
    payload = {
        'cursor': None,
        'task_start': '2016-04-07 12:13:14',
        'task_count': 2,
        'count': 42,
    }
    params = ts_mon_metrics._ShardParams(json.dumps(payload))
    self.assertEqual(json.loads(params.json()), payload)

  def test_shard_params_fail(self):
    with self.assertRaises(ValueError):
      ts_mon_metrics._ShardParams('invalid}')

  def test_on_task_completed(self):
    tags = [
        'project:test_project',
        'subproject:test_subproject',
        'pool:test_pool',
        'master:test_master',
        'buildername:test_builder',
        'name:some_tests',
    ]
    fields = {
        'project_id': 'test_project',
        'subproject_id': 'test_subproject',
        'pool': 'test_pool',
        'spec_name': 'test_master:test_builder',
    }
    summary = _gen_task_result_summary(self.now, 1, tags=tags)
    summary.exit_code = 0 # sets failure = False.
    summary.internal_failure = False
    summary.duration = 42

    fields['result'] = 'success'
    self.assertIsNone(ts_mon_metrics._jobs_completed.get(fields=fields))
    ts_mon_metrics.on_task_completed(summary)
    self.assertEqual(1, ts_mon_metrics._jobs_completed.get(fields=fields))

    summary.exit_code = 1 # sets failure = True.
    fields['result'] = 'failure'
    self.assertIsNone(ts_mon_metrics._jobs_completed.get(fields=fields))
    ts_mon_metrics.on_task_completed(summary)
    self.assertEqual(1, ts_mon_metrics._jobs_completed.get(fields=fields))

    summary.internal_failure = True
    fields['result'] = 'infra-failure'
    self.assertIsNone(ts_mon_metrics._jobs_completed.get(fields=fields))
    ts_mon_metrics.on_task_completed(summary)
    self.assertEqual(1, ts_mon_metrics._jobs_completed.get(fields=fields))

  def test_on_task_requested(self):
    tags = [
        'project:test_project',
        'subproject:test_subproject',
        'pool:test_pool',
        'master:test_master',
        'buildername:test_builder',
        'name:some_tests',
        'spec_name:my:custom:test:spec:name',
    ]
    fields = {
        'project_id': 'test_project',
        'subproject_id': 'test_subproject',
        'pool': 'test_pool',
        'spec_name': 'my:custom:test:spec:name',
    }
    summary = _gen_task_result_summary(self.now, 1, tags=tags)

    fields['deduped'] = True
    self.assertIsNone(ts_mon_metrics._jobs_requested.get(fields=fields))
    ts_mon_metrics.on_task_requested(summary, deduped=True)
    self.assertEqual(1, ts_mon_metrics._jobs_requested.get(fields=fields))

    fields['deduped'] = False
    self.assertIsNone(ts_mon_metrics._jobs_requested.get(fields=fields))
    ts_mon_metrics.on_task_requested(summary, deduped=False)
    self.assertEqual(1, ts_mon_metrics._jobs_requested.get(fields=fields))

  def test_on_task_requested_experimental(self):
    tags = [
        'project:test_project',
        'subproject:test_subproject',
        'pool:test_pool',
        'master:test_master',
        'buildername:test_builder',
        'name:some_tests',
        'build_is_experimental:true',
    ]
    fields = {
        'project_id': 'test_project',
        'subproject_id': 'test_subproject',
        'pool': 'test_pool',
        'spec_name': 'test_master:test_builder:experimental',
    }
    summary = _gen_task_result_summary(self.now, 1, tags=tags)

    fields['deduped'] = False
    self.assertIsNone(ts_mon_metrics._jobs_requested.get(fields=fields))
    ts_mon_metrics.on_task_requested(summary, deduped=False)
    self.assertEqual(1, ts_mon_metrics._jobs_requested.get(fields=fields))

  def test_initialize(self):
    # Smoke test for syntax errors.
    ts_mon_metrics.initialize()

  def test_set_global_metrics(self):
    tags = [
        'project:test_project',
        'subproject:test_subproject',
        'pool:test_pool',
        'master:test_master',
        'buildername:test_builder',
        'name:some_tests',
    ]
    summary_running = _gen_task_result_summary(self.now, 1, tags=tags)
    summary_running.state = task_result.State.RUNNING
    summary_running.modified_ts = self.now
    summary_running.started_ts = self.now
    summary_running.bot_id = 'test_bot1'
    summary_running.put()

    summary_pending = _gen_task_result_summary(
        self.now - datetime.timedelta(minutes=5), 2, tags=tags)
    summary_pending.state = task_result.State.PENDING
    summary_pending.modified_ts = self.now
    summary_pending.bot_id = 'test_bot2'
    summary_pending.put()

    summary_pending = _gen_task_result_summary(
        self.now - datetime.timedelta(minutes=10), 3, tags=tags)
    summary_pending.state = task_result.State.PENDING
    summary_pending.modified_ts = self.now
    summary_pending.bot_id = ''
    summary_pending.put()

    _gen_bot_info('bot_ready', self.now).put()
    _gen_bot_info('bot_running', self.now, task_id='deadbeef').put()
    _gen_bot_info('bot_quarantined', self.now, quarantined=True).put()
    _gen_bot_info('bot_dead', self.now - datetime.timedelta(days=365)).put()
    _gen_bot_info(
        'bot_maintenance', self.now, state={'maintenance': True}).put()
    bots_expected = {
        'bot_ready': 'ready',
        'bot_running': 'running',
        'bot_quarantined': 'quarantined',
        'bot_dead': 'dead',
        'bot_maintenance': 'maintenance'
    }

    ts_mon_metrics.set_global_metrics('jobs')
    ts_mon_metrics.set_global_metrics('executors')

    jobs_fields = {
        'project_id': 'test_project',
        'subproject_id': 'test_subproject',
        'pool': 'test_pool',
        'spec_name': 'test_master:test_builder',
    }
    jobs_target_fields = dict(ts_mon_metrics._TARGET_FIELDS)
    jobs_target_fields['hostname'] = 'autogen:test_bot1'

    self.assertTrue(ts_mon_metrics._jobs_running.get(
        fields=jobs_fields, target_fields=jobs_target_fields))
    jobs_target_fields['hostname'] = 'autogen:test_bot2'
    self.assertFalse(ts_mon_metrics._jobs_running.get(
        fields=jobs_fields, target_fields=jobs_target_fields))
    jobs_fields['status'] = 'running'
    self.assertEqual(1, ts_mon_metrics._jobs_active.get(
        fields=jobs_fields, target_fields=ts_mon_metrics._TARGET_FIELDS))
    jobs_fields['status'] = 'pending'
    self.assertEqual(2, ts_mon_metrics._jobs_active.get(
        fields=jobs_fields, target_fields=ts_mon_metrics._TARGET_FIELDS))

    self.assertEqual(900, ts_mon_metrics._jobs_pending_durations.get(
        fields=jobs_fields, target_fields=ts_mon_metrics._TARGET_FIELDS).sum)
    self.assertEqual(600, ts_mon_metrics._jobs_max_pending_duration.get(
        fields=jobs_fields, target_fields=ts_mon_metrics._TARGET_FIELDS))

    for bot_id, status in bots_expected.iteritems():
      target_fields = dict(ts_mon_metrics._TARGET_FIELDS)
      target_fields['hostname'] = 'autogen:' + bot_id
      self.assertEqual(status, ts_mon_metrics._executors_status.get(
          target_fields=target_fields))

      self.assertEqual('bot_id:%s|os:Linux|os:Ubuntu' % bot_id,
                       ts_mon_metrics._executors_pool.get(
                           target_fields=target_fields))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
