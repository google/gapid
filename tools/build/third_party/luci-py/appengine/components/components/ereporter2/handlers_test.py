#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import json
import logging
import re
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.api import logservice

import webapp2
import webtest

from components import auth
from components import template
from components.ereporter2 import acl
from components.ereporter2 import handlers
from components.ereporter2 import logscraper
from components.ereporter2 import models
from components.ereporter2 import on_error
from components.ereporter2 import ui
from test_support import test_case


# Access to a protected member XXX of a client class - pylint: disable=W0212


def ErrorRecord(**kwargs):
  """Returns an ErrorRecord filled with default dummy values."""
  vals = {
      'request_id': '123',
      'start_time': None,
      'exception_time': None,
      'latency': 0,
      'mcycles': 0,
      'ip': '0.0.1.0',
      'nickname': None,
      'referrer': None,
      'user_agent': 'Comodore64',
      'host': 'localhost',
      'resource': '/foo',
      'method': 'GET',
      'task_queue_name': None,
      'was_loading_request': False,
      'version': 'v1',
      'module': 'default',
      'handler_module': 'main.app',
      'gae_version': '1.9.0',
      'instance': '123',
      'status': 200,
      'message': (
          u'Traceback (most recent call last):\n'
          '  File "handlers_frontend.py", line 461, in post\n'
          '    for entry_info, exists in self.check_entry_infos('
              'entries, namespace):\n'
          '  File "handlers_frontend.py", line 343, in check_entry_infos\n'
          '    future = ndb.Future.wait_any(futures)\n'
          '  File "appengine/ext/ndb/tasklets.py", line 338, in wait_any\n'
          '    ev.run1()\n'
          '  File "appengine/ext/ndb/eventloop.py", line 235, in run1\n'
          '    delay = self.run0()\n'
          '  File "appengine/ext/ndb/eventloop.py", line 197, in run0\n'
          '    callback(*args, **kwds)\n'
          '  File "appengine/ext/ndb/tasklets.py", line 474, in '
              '_on_future_completion\n'
          '    self._help_tasklet_along(ns, ds_conn, gen, val)\n'
          '  File "appengine/ext/ndb/tasklets.py", line 371, in '
              '_help_tasklet_along\n'
          '    value = gen.send(val)\n'
          '  File "appengine/ext/ndb/context.py", line 751, in get\n'
          '    pbs = entity._to_pb(set_key=False).SerializePartialToString()\n'
          '  File "appengine/ext/ndb/model.py", line 3069, in _to_pb\n'
          '    prop._serialize(self, pb, projection=self._projection)\n'
          '  File "appengine/ext/ndb/model.py", line 1374, in _serialize\n'
          '    self._db_set_value(v, p, val)\n'
          '  File "appengine/ext/ndb/model.py", line 2042, in _db_set_value\n'
          '    p.set_meaning(entity_pb.Property.GD_WHEN)\n'
          'DeadlineExceededError\n')
  }
  vals.update(kwargs)
  signature, exception_type = logscraper._signature_from_message(
      vals['message'])
  return logscraper._ErrorRecord(
      signature=signature, exception_type=exception_type, **vals)


class Base(test_case.TestCase):
  def setUp(self):
    super(Base, self).setUp()
    self.testbed.init_user_stub()
    self._now = datetime.datetime(2014, 6, 24, 20, 19, 42, 653775)
    self.mock_now(self._now, 0)
    ui.configure()

  def tearDown(self):
    template.reset()
    super(Base, self).tearDown()


class Ereporter2FrontendTest(Base):
  def setUp(self):
    super(Ereporter2FrontendTest, self).setUp()
    self.app = webtest.TestApp(
        webapp2.WSGIApplication(handlers.get_frontend_routes(), debug=True),
        extra_environ={'REMOTE_ADDR': '127.0.0.1'})

  def mock_as_admin(self):
    def is_group_member_mock(group, identity=None):
      return group == auth.model.ADMIN_GROUP or original(group, identity)
    original = self.mock(auth.api, 'is_group_member', is_group_member_mock)
    class admin(object):
      def email(self):
        return 'admin@example.com'
    self.mock(auth.AuthenticatingHandler, 'get_current_user', lambda _s: admin)

  def test_frontend_general(self):
    self.mock_as_admin()
    exception = (
      '/ereporter2/api/v1/on_error',
      r'/restricted/ereporter2/errors/<error_id:\d+>',
      '/restricted/ereporter2/request/<request_id:[0-9a-fA-F]+>',
    )
    for route in handlers.get_frontend_routes():
      if not route.template in exception:
        self.app.get(route.template, status=200)

    def gen_request(request_id):
      # TODO(maruel): Fill up with fake data if found necessary to test edge
      # cases.
      self.assertEqual('123', request_id)
      return logservice.RequestLog()
    self.mock(logscraper, '_log_request_id', gen_request)
    self.app.get('/restricted/ereporter2/request/123', status=200)

  def test_on_error_handler(self):
    self.mock(logging, 'error', lambda *_a, **_k: None)
    data = {
      'foo': 'bar',
    }
    for key in on_error.VALID_ERROR_KEYS:
      data[key] = 'bar %s' % key
    data['category'] = 'auth'
    data['duration'] = 2.3
    data['source'] = 'run_isolated'
    params = {
      'r': data,
      'v': '1',
    }
    response = self.app.post(
        '/ereporter2/api/v1/on_error', json.dumps(params), status=200,
        content_type='application/json; charset=utf-8').json

    self.assertEqual(1, models.Error.query().count())
    error_id = models.Error.query().get().key.integer_id()
    expected = {
      'id': error_id,
      'url': u'http://localhost/restricted/ereporter2/errors/%d' % error_id,
    }
    self.assertEqual(expected, response)

    self.mock_as_admin()
    self.app.get('/restricted/ereporter2/errors')
    self.app.get('/restricted/ereporter2/errors/%d' % error_id)

  def test_on_error_handler_denied(self):
    self.app.get('/ereporter2/api/v1/on_error', status=405)

  def test_on_error_handler_bad_type(self):
    self.mock(logging, 'error', lambda *_a, **_k: None)
    params = {
      # 'args' should be a list.
      'r': {'args': 'bar'},
      'v': '1',
    }
    response = self.app.post(
        '/ereporter2/api/v1/on_error', json.dumps(params), status=200,
        content_type='application/json; charset=utf-8').json
    # There's still a response but it will be an error about the error.
    self.assertEqual(1, models.Error.query().count())
    error_id = models.Error.query().get().key.integer_id()
    self.assertEqual(response.get('id'), error_id)

  def test_report_silence(self):
    # Log an error, ensure it's returned, silence it, ensure it's silenced.
    self.mock_as_admin()
    exceptions = [ErrorRecord()]
    self.mock(
        logscraper, '_extract_exceptions_from_logs', lambda *_: exceptions[:])

    resp = self.app.get('/restricted/ereporter2/report')
    # Grep the form. This is crude parsing with assumption of the form layout.
    # mechanize could be used if more complex parsing is needed.
    forms = re.findall(r'(\<form .+?\<\/form\>)', resp.body, re.DOTALL)
    self.assertEqual(1, len(forms))
    form = forms[0]
    silence_url = re.search(r'action\=\"(.+?)\"', form).group(1)
    self.assertEqual('/restricted/ereporter2/silence', silence_url)

    expected_inputs = {
      'exception_type': 'DeadlineExceededError',
      'signature': 'DeadlineExceededError@check_entry_infos',
      'mute_type': 'exception_type',
      'silenced': None,
      'silenced_until': 'T',
      'threshold': '10',
    }
    actual_inputs = {}
    for i in re.findall(r'(\<input .+?\<\/input\>)', form, re.DOTALL):
      input_type = re.search(r'type\=\"(.+?)\"', i).group(1)
      name_match = re.search(r'name\=\"(.+?)\"', i)
      if input_type == 'submit':
        self.assertEqual(None, name_match)
        continue
      self.assertTrue(name_match, i)
      name = name_match.group(1)
      # That's cheezy, as silenced used 'checked', not value.
      value_match = re.search(r'value\=\"(.+)\"', i)
      if name == 'xsrf_token':
        expected_inputs[name] = value_match.group(1)
      actual_inputs[name] = value_match.group(1) if value_match else None
    self.assertEqual(expected_inputs, actual_inputs)

    def gen_request(request_id):
      for i in exceptions:
        if i.request_id == request_id:
          return logservice.RequestLog()
      self.fail()
    self.mock(logscraper, '_log_request_id', gen_request)
    self.app.get('/restricted/ereporter2/request/123', status=200)

    params = {k: (v or '') for k, v in actual_inputs.iteritems()}
    # Silence it.
    params['silenced'] = '1'
    resp = self.app.post(silence_url, params=params)

    silenced = models.ErrorReportingMonitoring().query().fetch()
    self.assertEqual(1, len(silenced))

    # Ensures silencing worked.
    resp = self.app.get('/restricted/ereporter2/report')
    self.assertIn('Found 0 occurrences of 0 errors across', resp.body)
    self.assertIn('Ignored 1 occurrences of 1 errors across', resp.body)

  def test_report_silence_autologin(self):
    resp = self.app.get('/restricted/ereporter2/report')
    self.assertEqual(302, resp.status_code)


class Ereporter2BackendTest(Base):
  def setUp(self):
    super(Ereporter2BackendTest, self).setUp()
    self.app = webtest.TestApp(
        webapp2.WSGIApplication(handlers.get_backend_routes(), debug=True),
        extra_environ={'REMOTE_ADDR': '127.0.0.1'})

  def test_cron_ereporter2_mail_not_cron(self):
    self.mock(logging, 'error', lambda *_a, **_k: None)
    response = self.app.get(
        '/internal/cron/ereporter2/mail', expect_errors=True)
    self.assertEqual(response.status_int, 403)
    self.assertEqual(response.content_type, 'text/plain')
    # Verify no email was sent.
    self.assertEqual([], self.mail_stub.get_sent_messages())

  def test_cron_ereporter2_mail(self):
    data = [ErrorRecord()]
    self.mock(logscraper, '_extract_exceptions_from_logs', lambda *_: data)
    self.mock(acl, 'get_ereporter2_recipients', lambda: ['joe@localhost'])
    headers = {'X-AppEngine-Cron': 'true'}
    response = self.app.get(
        '/internal/cron/ereporter2/mail', headers=headers)
    self.assertEqual(response.status_int, 200)
    self.assertEqual(response.normal_body, 'Success.')
    self.assertEqual(response.content_type, 'text/plain')
    # Verify the email was sent.
    messages = self.mail_stub.get_sent_messages()
    self.assertEqual(1, len(messages))
    message = messages[0]
    self.assertTrue(hasattr(message, 'to'), message.html)
    escaped = data[0].message.replace('"', '&#34;')
    expected_text = (
      '1 occurrences of 1 errors across 1 versions.\n\n'
      'DeadlineExceededError@check_entry_infos\n'
      'Handler: main.app\n'
      'Modules: default\n'
      'Versions: v1\n'
      'GET localhost/foo (HTTP 200)\n') + escaped + (
      '\n'
      '1 occurrences: Entry \n\n')
    self.assertEqual(expected_text, message.body.payload)

  def test_cron_old_errors(self):
    self.mock(logging, 'error', lambda *_a, **_k: None)
    kwargs = dict((k, k) for k in on_error.VALID_ERROR_KEYS)
    kwargs['category'] = 'exception'
    kwargs['duration'] = 2.3
    kwargs['source'] = 'bot'
    kwargs['source_ip'] = '0.0.0.0'
    on_error.log(**kwargs)

    # First call shouldn't delete the error since its not stale yet.
    headers = {'X-AppEngine-Cron': 'true'}
    response = self.app.get(
        '/internal/cron/ereporter2/cleanup', headers=headers)
    self.assertEqual('0', response.body)
    self.assertEqual(1, models.Error.query().count())

    # Set the current time to the future, but not too much.
    now = self._now + on_error.ERROR_TIME_TO_LIVE
    self.mock_now(now, -60)

    headers = {'X-AppEngine-Cron': 'true'}
    response = self.app.get(
        '/internal/cron/ereporter2/cleanup', headers=headers)
    self.assertEqual('0', response.body)
    self.assertEqual(1, models.Error.query().count())

    # Set the current time to the future.
    now = self._now + on_error.ERROR_TIME_TO_LIVE
    self.mock_now(now, 60)

    # Second call should remove the now stale error.
    headers = {'X-AppEngine-Cron': 'true'}
    response = self.app.get(
        '/internal/cron/ereporter2/cleanup', headers=headers)
    self.assertEqual('1', response.body)
    self.assertEqual(0, models.Error.query().count())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
