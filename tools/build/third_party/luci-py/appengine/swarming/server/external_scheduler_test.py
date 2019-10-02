#!/usr/bin/env python
# coding: utf-8
# Copyright 2019 The LUCI Authors. All rights reserved.
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
import handlers_backend

from test_support import test_case

from components import utils
from proto.api import plugin_pb2
from proto.api import swarming_pb2
from server import external_scheduler
from server import pools_config
from server import task_request
from server import task_scheduler


def _gen_properties(**kwargs):
  """Creates a TaskProperties."""
  args = {
    'command': [u'command1'],
    'dimensions': {u'os': [u'Windows-3.1.1'], u'pool': [u'default']},
    'env': {},
    'execution_timeout_secs': 24*60*60,
    'io_timeout_secs': None,
  }
  args.update(kwargs)
  args['dimensions_data'] = args.pop('dimensions')
  return task_request.TaskProperties(**args)


def _gen_request(**kwargs):
  """Returns an initialized task_request.TaskRequest."""
  now = utils.utcnow()
  args = {
    # Don't be confused, this is not part of the API. This code is constructing
    # a DB entity, not a swarming_rpcs.NewTaskRequest.
    u'created_ts': now,
    u'manual_tags': [u'tag:1'],
    u'name': u'yay',
    u'priority': 50,
    u'task_slices': [
      task_request.TaskSlice(
        expiration_secs=60,
        properties=_gen_properties(),
        wait_for_capacity=False),
    ],
    u'user': u'Jesus',
  }
  args.update(kwargs)
  ret = task_request.TaskRequest(**args)
  task_request.init_new_request(ret, True, task_request.TEMPLATE_AUTO)
  return ret


class FakeExternalScheduler(object):
  def __init__(self, test):
    self._test = test
    self.called_with_requests = []

  def AssignTasks(self, req, credentials): # pylint: disable=unused-argument
    self._test.assertIsInstance(req, plugin_pb2.AssignTasksRequest)
    self.called_with_requests.append(req)
    resp = plugin_pb2.AssignTasksResponse()
    item = resp.assignments.add()
    item.bot_id = req.idle_bots[0].bot_id
    item.task_id = 'task A'
    item.slice_number = 1
    return resp

  # pylint: disable=unused-argument
  def GetCancellations(self, req, credentials):
    self._test.assertIsInstance(req, plugin_pb2.GetCancellationsRequest)
    self.called_with_requests.append(req)
    resp = plugin_pb2.GetCancellationsResponse()
    item = resp.cancellations.add()
    item.bot_id = 'bot_id'
    item.task_id = 'task_id'
    return resp

  def NotifyTasks(self, req, credentials):  # pylint: disable=unused-argument
    self._test.assertIsInstance(req, plugin_pb2.NotifyTasksRequest)
    self.called_with_requests.append(req)
    return plugin_pb2.NotifyTasksResponse()

  def GetCallbacks(self, req, credentials): # pylint: disable=unused-argument
    self._test.assertIsInstance(req, plugin_pb2.GetCallbacksRequest)
    self.called_with_requests.append(req)
    resp = plugin_pb2.GetCallbacksResponse()
    resp.task_ids.append('task A')
    resp.task_ids.append('task B')
    return resp


class ExternalSchedulerApiTest(test_env_handlers.AppTestBase):

  def setUp(self):
    super(ExternalSchedulerApiTest, self).setUp()
    self.es_cfg = pools_config.ExternalSchedulerConfig(
        address=u'http://localhost:1',
        id=u'foo',
        dimensions=['key1:value1', 'key2:value2'],
        all_dimensions=None,
        any_dimensions=None,
        enabled=True,
        fallback_when_empty=True)

    # Make the values deterministic.
    self.mock_now(datetime.datetime(2014, 1, 2, 3, 4, 5, 6))
    self.mock(random, 'getrandbits', lambda _: 0x88)

    # Use the local fake client to external scheduler..
    self.mock(external_scheduler, '_get_client', self._get_client)
    self._client = None

    # Setup the backend to handle task queues.
    self.app = webtest.TestApp(
        handlers_backend.create_application(True),
        extra_environ={
          'REMOTE_ADDR': self.source_ip,
          'SERVER_SOFTWARE': os.environ['SERVER_SOFTWARE'],
        })
    self._enqueue_orig = self.mock(utils, 'enqueue_task', self._enqueue)

  def _enqueue(self, *args, **kwargs):
    return self._enqueue_orig(*args, use_dedicated_module=False, **kwargs)

  def _get_client(self, addr):
    self.assertEqual(u'http://localhost:1', addr)
    self.assertFalse(self._client)
    self._client = FakeExternalScheduler(self)
    return self._client

  def test_all_apis_are_tested(self):
    actual = frozenset(i[5:] for i in dir(self) if i.startswith('test_'))
    # Contains the list of all public APIs.
    expected = frozenset(
        i for i in dir(external_scheduler)
        if i[0] != '_' and hasattr(getattr(external_scheduler, i), 'func_name'))
    missing = expected - actual
    self.assertFalse(missing)

  def test_assign_task(self):
    task_id, slice_number = external_scheduler.assign_task(
        self.es_cfg, {u'id': 'bot_id'})
    self.assertEqual(task_id, 'task A')
    self.assertEqual(slice_number, 1)

  def test_config_for_bot(self):
    # TODO(akeshet): Add.
    pass

  def test_config_for_task(self):
    # TODO(akeshet): Add.
    pass

  def test_get_cancellations(self):
    c = external_scheduler.get_cancellations(self.es_cfg)
    self.assertEqual(len(c), 1)
    self.assertEqual(c[0].bot_id, 'bot_id')
    self.assertEqual(c[0].task_id, 'task_id')

  def test_notify_requests(self):
    request = _gen_request()
    result_summary = task_scheduler.schedule_request(request, None)
    external_scheduler.notify_requests(
        self.es_cfg, [(request, result_summary)], False, False)

    self.assertEqual(len(self._client.called_with_requests), 1)
    called_with = self._client.called_with_requests[0]
    self.assertEqual(len(called_with.notifications), 1)
    notification = called_with.notifications[0]

    self.assertEqual(request.created_ts,
                     notification.task.enqueued_time.ToDatetime())
    self.assertEqual(request.task_id, notification.task.id)
    self.assertEqual(request.num_task_slices, len(notification.task.slices))

    self.execute_tasks()

  def test_notify_request_with_tq(self):
    request = _gen_request()
    result_summary = task_scheduler.schedule_request(request, None)
    external_scheduler.notify_requests(
      self.es_cfg, [(request, result_summary)], True, False)

    # There should have been no call to _get_client yet.
    self.assertEqual(self._client, None)

    self.execute_tasks()

    # After taskqueue executes, there should be a call to the client.
    self.assertEqual(len(self._client.called_with_requests), 1)
    called_with = self._client.called_with_requests[0]
    self.assertEqual(len(called_with.notifications), 1)
    notification = called_with.notifications[0]

    self.assertEqual(request.created_ts,
                     notification.task.enqueued_time.ToDatetime())
    self.assertEqual(request.task_id, notification.task.id)
    self.assertEqual(request.num_task_slices, len(notification.task.slices))

  def test_notify_request_now(self):
    r = plugin_pb2.NotifyTasksRequest()
    res = external_scheduler.notify_request_now("http://localhost:1", r)
    self.assertEqual(plugin_pb2.NotifyTasksResponse(), res)

  def test_get_callbacks(self):
    tasks = external_scheduler.get_callbacks(self.es_cfg)
    self.assertEqual(tasks, ['task A', 'task B'])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
