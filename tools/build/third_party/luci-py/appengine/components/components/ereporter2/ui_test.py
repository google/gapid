#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import os
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components import auth
from components import template
from components.ereporter2 import acl
from components.ereporter2 import logscraper
from components.ereporter2 import ui
from test_support import test_case


ABS_PATH = os.path.abspath(__file__)
ROOT_DIR = os.path.dirname(ABS_PATH)


# Access to a protected member XXX of a client class - pylint: disable=W0212
# Method could be a function - pylint: disable=R0201

def ErrorRecord(**kwargs):
  """Returns an ErrorRecord filled with default dummy values."""
  vals = {
      'request_id': 'a',
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
      'message': u'Failed',
  }
  vals.update(kwargs)
  signature, exception_type = logscraper._signature_from_message(
      vals['message'])
  return logscraper._ErrorRecord(
      signature=signature, exception_type=exception_type, **vals)


class Ereporter2Test(test_case.TestCase):
  def setUp(self):
    super(Ereporter2Test, self).setUp()
    self.testbed.init_user_stub()
    self.mock(acl, 'get_ereporter2_recipients', lambda: ['foo@localhost'])
    self.mock(ui, '_get_end_time_for_email', lambda: 1383000000)
    self._now = datetime.datetime(2014, 6, 24, 20, 19, 42, 653775)
    self.mock_now(self._now, 0)
    ui.configure()

  def tearDown(self):
    template.reset()
    super(Ereporter2Test, self).tearDown()

  def assertContent(self, message):
    self.assertEqual(
        u'no_reply@sample-app.appspotmail.com', message.sender)
    self.assertEqual(u'Exceptions on "sample-app"', message.subject)
    expected_html = (
        '<html><body><h3><a href="http://foo/report?start=0&end=1383000000">1 '
        'occurrences of 1 errors across 1 versions.</a></h3>\n'
        '\n'
        '<span style="font-size:130%">Failed</span><br>\n'
        'Handler: main.app<br>\n'
        'Modules: default<br>\n'
        'Versions: v1<br>\n'
        'GET localhost/foo (HTTP 200)<br>\n'
        '<pre>Failed</pre>\n'
        '1 occurrences: <a href="http://foo/request/a">Entry</a> <p>\n'
        '<br>\n'
        '</body></html>')
    self.assertEqual(
        expected_html.splitlines(), message.html.payload.splitlines())
    expected_text = (
        '1 occurrences of 1 errors across 1 versions.\n'
        '\n'
        'Failed\n'
        'Handler: main.app\n'
        'Modules: default\n'
        'Versions: v1\n'
        'GET localhost/foo (HTTP 200)\n'
        'Failed\n'
        '1 occurrences: Entry \n\n')
    self.assertEqual(expected_text, message.body.payload)

  def test_email_no_recipients(self):
    data = [
      ErrorRecord(),
    ]
    self.mock(logscraper, '_extract_exceptions_from_logs', lambda *_: data)
    result = ui._generate_and_email_report(
        module_versions=[],
        recipients=None,
        request_id_url='http://foo/request/',
        report_url='http://foo/report',
        extras={})
    self.assertEqual(True, result)

    # Verify the email that was sent.
    messages = self.mail_stub.get_sent_messages()
    message = messages[-1]
    self.assertFalse(hasattr(message, 'to'))
    self.assertContent(message)

  def test_email_recipients(self):
    data = [
      ErrorRecord(),
    ]
    self.mock(logscraper, '_extract_exceptions_from_logs', lambda *_: data)
    result = ui._generate_and_email_report(
        module_versions=[],
        recipients='joe@example.com',
        request_id_url='http://foo/request/',
        report_url='http://foo/report',
        extras={})
    self.assertEqual(True, result)

    # Verify the email that was sent.
    messages = self.mail_stub.get_sent_messages()
    self.assertEqual(1, len(messages))
    message = messages[0]
    self.assertEqual(u'joe@example.com', message.to)
    self.assertContent(message)

  def test_get_template_env(self):
    env = ui._get_template_env(10, 20, [('foo', 'bar')])
    expected = {
      'end': 20,
      'module_versions': [('foo', 'bar')],
      'start': 10,
    }
    self.assertEqual(expected, env)

  def test_records_to_params(self):
    msg = logscraper._STACK_TRACE_MARKER + u'\nDeadlineExceededError'
    data = [
      ErrorRecord(),
      ErrorRecord(message=msg),
      ErrorRecord(),
    ]
    self.mock(logscraper, '_extract_exceptions_from_logs', lambda *_: data)
    module_versions = [('foo', 'bar')]
    report, ignored, end_time = logscraper.scrape_logs_for_errors(
        10, 20, module_versions)
    self.assertEqual(20, end_time)
    out = ui._records_to_params(
        report, len(ignored), 'http://localhost:1/request_id',
        'http://localhost:2/report')
    expected = {
      'error_count': 2,
      'ignored_count': 0,
      'occurrence_count': 3,
      'report_url': 'http://localhost:2/report',
      'request_id_url': 'http://localhost:1/request_id',
      'version_count': 1,
    }
    out.pop('errors')
    self.assertEqual(expected, out)


class Ereporter2RecipientsTest(test_case.TestCase):
  def test_recipients_from_auth_group(self):
    fake_listing = auth.GroupListing([
      auth.Identity(auth.IDENTITY_USER, 'a@example.com'),
      auth.Identity(auth.IDENTITY_USER, 'b@example.com'),
      auth.Identity(auth.IDENTITY_SERVICE, 'blah-service'),
    ], [], [])
    self.mock(auth, 'list_group', lambda _: fake_listing)
    self.assertEqual(
        ['a@example.com', 'b@example.com'], acl.get_ereporter2_recipients())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
