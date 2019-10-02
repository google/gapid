# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import contextlib
import datetime
import json
import logging
import time
import urllib

import webtest

from google.appengine.datastore import datastore_stub_util
from google.appengine.ext import ndb
from google.appengine.ext import testbed

from components import endpoints_webapp2
from components import utils
from depot_tools import auto_stub

# W0212: Access to a protected member XXX of a client class
# pylint: disable=W0212


def mock_now(test, now, seconds):
  """Mocks utcnow() and ndb properties.

  In particular handles when auto_now and auto_now_add are used.
  """
  now = now + datetime.timedelta(seconds=seconds)
  test.mock(utils, 'utcnow', lambda: now)
  test.mock(ndb.DateTimeProperty, '_now', lambda _: now)
  test.mock(ndb.DateProperty, '_now', lambda _: now.date())
  return now


class TestCase(auto_stub.TestCase):
  """Support class to enable more unit testing in GAE.

  Adds support for:
    - google.appengine.api.mail.send_mail_to_admins().
    - Running task queues.
  """
  # See APP_DIR to the root directory containing index.yaml and queue.yaml. It
  # will be used to assert the indexes and task queues are properly defined. It
  # can be left to None if no index or task queue is used for the test case.
  APP_DIR = None

  # A test can explicitly acknowledge it depends on composite indexes that may
  # not be defined in index.yaml by setting this to True. It is valid only for
  # components unit tests that are running outside of a context of some app
  # (APP_DIR is None in this case). If APP_DIR is provided, GAE testbed silently
  # overwrite index.yaml, and it's not what we want.
  SKIP_INDEX_YAML_CHECK = False

  # If taskqueues are enqueued during the unit test, self.app must be set to a
  # webtest.Test instance. It will be used to do the HTTP post when executing
  # the enqueued tasks via the taskqueue module.
  app = None

  def setUp(self):
    """Initializes the commonly used stubs.

    Using init_all_stubs() costs ~10ms more to run all the tests so only enable
    the ones known to be required. Test cases requiring more stubs can enable
    them in their setUp() function.
    """
    super(TestCase, self).setUp()
    self.testbed = testbed.Testbed()
    self.testbed.activate()

    # If you have a NeedIndexError, here is the switch you need to flip to make
    # the new required indexes to be automatically added. Change
    # train_index_yaml to True to have index.yaml automatically updated, then
    # run your test case. Do not forget to put it back to False.
    train_index_yaml = False

    if self.SKIP_INDEX_YAML_CHECK:
      # See comment for skip_index_yaml_check above.
      self.assertIsNone(self.APP_DIR)

    self.testbed.init_app_identity_stub()
    self.testbed.init_datastore_v3_stub(
        require_indexes=not train_index_yaml and not self.SKIP_INDEX_YAML_CHECK,
        root_path=self.APP_DIR,
        consistency_policy=datastore_stub_util.PseudoRandomHRConsistencyPolicy(
            probability=1))
    self.testbed.init_logservice_stub()
    self.testbed.init_memcache_stub()
    self.testbed.init_modules_stub()

    # Use mocked time in memcache.
    memcache = self.testbed.get_stub(testbed.MEMCACHE_SERVICE_NAME)
    memcache._gettime = lambda: int(utils.time_time())

    # Email support.
    self.testbed.init_mail_stub()
    self.mail_stub = self.testbed.get_stub(testbed.MAIL_SERVICE_NAME)
    self.old_send_to_admins = self.mock(
        self.mail_stub, '_Dynamic_SendToAdmins', self._SendToAdmins)

    self.testbed.init_taskqueue_stub()
    self._taskqueue_stub = self.testbed.get_stub(testbed.TASKQUEUE_SERVICE_NAME)
    self._taskqueue_stub._root_path = self.APP_DIR

    self.testbed.init_user_stub()

  def tearDown(self):
    try:
      if not self.has_failed():
        remaining = self.execute_tasks()
        self.assertEqual(0, remaining,
            'Passing tests must leave behind no pending tasks, found %d.'
            % remaining)
      self.testbed.deactivate()
    finally:
      super(TestCase, self).tearDown()

  def mock_now(self, now, seconds=0):
    return mock_now(self, now, seconds)

  def _SendToAdmins(self, request, *args, **kwargs):
    """Make sure the request is logged.

    See google_appengine/google/appengine/api/mail_stub.py around line 299,
    MailServiceStub._SendToAdmins().
    """
    self.mail_stub._CacheMessage(request)
    return self.old_send_to_admins(request, *args, **kwargs)

  def execute_tasks(self, **kwargs):
    """Executes enqueued tasks that are ready to run and return the number run.

    A task may trigger another task.

    Sadly, taskqueue_stub implementation does not provide a nice way to run
    them so run the pending tasks manually.
    """
    self.assertEqual([None], self._taskqueue_stub._queues.keys())
    ran_total = 0
    while True:
      # Do multiple loops until no task was run.
      ran = 0
      for queue in self._taskqueue_stub.GetQueues():
        for task in self._taskqueue_stub.GetTasks(queue['name']):
          # Remove 2 seconds for jitter.
          eta = task['eta_usec'] / 1e6 - 2
          if eta >= time.time():
            continue
          self.assertEqual('POST', task['method'])
          logging.info('Task: %s', task['url'])

          # Not 100% sure why the Content-Length hack is needed, nor why the
          # stub returns unicode values that break webtest's assertions.
          body = base64.b64decode(task['body'])
          headers = {k: str(v) for k, v in task['headers']}
          headers['Content-Length'] = str(len(body))
          try:
            self.app.post(task['url'], body, headers=headers, **kwargs)
          except:
            logging.error(task)
            raise
          self._taskqueue_stub.DeleteTask(queue['name'], task['name'])
          ran += 1
      if not ran:
        return ran_total
      ran_total += ran


class Endpoints(object):
  """Handles endpoints API calls."""
  def __init__(self, api_service_cls, regex=None, source_ip='127.0.0.1'):
    super(Endpoints, self).__init__()
    self._api_service_cls = api_service_cls
    kwargs = {}
    if regex:
      kwargs['regex'] = regex
    self._api_app = webtest.TestApp(
        endpoints_webapp2.api_server([self._api_service_cls], **kwargs),
        extra_environ={'REMOTE_ADDR': source_ip})

  def call_api(self, method, body=None, status=(200, 204)):
    """Calls endpoints API method identified by its name."""
    # Because body is a dict and not a ResourceContainer, there's no way to tell
    # which parameters belong in the URL and which belong in the body when the
    # HTTP method supports both. However there's no harm in supplying parameters
    # in both the URL and the body since ResourceContainers don't allow the same
    # parameter name to be used in both places. Supplying parameters in both
    # places produces no ambiguity and extraneous parameters are safely ignored.
    assert hasattr(self._api_service_cls, method), method
    info = getattr(self._api_service_cls, method).method_info
    path = info.get_path(self._api_service_cls.api_info)

    # Identify which arguments are path parameters and which are query strings.
    body = body or {}
    query_strings = []
    for key, value in sorted(body.iteritems()):
      if '{%s}' % key in path:
        path = path.replace('{%s}' % key, value)
      else:
        # We cannot tell if the parameter is a repeated field from a dict.
        # Allow all query strings to be multi-valued.
        if not isinstance(value, list):
          value = [value]
        for val in value:
          query_strings.append('%s=%s' % (key, val))
    if query_strings:
      path = '%s?%s' % (path, '&'.join(query_strings))

    path = '/_ah/api/%s/%s/%s' % (self._api_service_cls.api_info.name,
                                  self._api_service_cls.api_info.version,
                                  path)
    try:
      if info.http_method in ('GET', 'DELETE'):
        return self._api_app.get(path, status=status)
      return self._api_app.post_json(path, body, status=status)
    except Exception as e:
      # Useful for diagnosing issues in test cases.
      logging.info('%s failed: %s', path, e)
      raise


class EndpointsTestCase(TestCase):
  """Base class for a test case that tests Cloud Endpoint Service.

  Usage:
    class MyTestCase(test_case.EndpointsTestCase):
      api_service_cls = MyEndpointsService

      def test_stuff(self):
        response = self.call_api('my_method')
        self.assertEqual(...)

      def test_expected_fail(self):
        with self.call_should_fail(403):
          self.call_api('protected_method')
  """
  # Should be set in subclasses to a subclass of remote.Service.
  api_service_cls = None
  # Should be set in subclasses to a regular expression to match against path
  # parameters. See components.endpoints_webapp2.adapter.api_server.
  api_service_regex = None

  # See call_should_fail.
  expected_fail_status = None

  _endpoints = None

  def setUp(self):
    super(EndpointsTestCase, self).setUp()
    self._endpoints = Endpoints(
        self.api_service_cls, regex=self.api_service_regex)

  def call_api(self, method, body=None, status=(200, 204)):
    if self.expected_fail_status:
      status = self.expected_fail_status
    return self._endpoints.call_api(method, body, status)

  @contextlib.contextmanager
  def call_should_fail(self, status):
    """Asserts that Endpoints call inside the guarded region of code fails."""
    # TODO(vadimsh): Get rid of this function and just use
    # call_api(..., status=...). It existed as a workaround for bug that has
    # been fixed:
    # https://code.google.com/p/googleappengine/issues/detail?id=10544
    assert self.expected_fail_status is None, 'nested call_should_fail'
    assert status is not None
    self.expected_fail_status = int(status)
    try:
      yield
    except AssertionError:
      # Assertion can happen if tests are running on GAE < 1.9.31, where
      # endpoints bug still exists (and causes webapp guts to raise assertion).
      # It should be rare (since we are switching to GAE >= 1.9.31), so don't
      # bother to check that assertion was indeed raised. Just skip it if it
      # did.
      pass
    finally:
      self.expected_fail_status = None
