#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import os
import platform
import re
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import webapp2
import webtest

from components import auth
from components.ereporter2 import formatter
from components.ereporter2 import models
from components.ereporter2 import on_error
from test_support import test_case


# Access to a protected member XXX of a client class - pylint: disable=W0212


ON_ERROR_PATH = os.path.abspath(on_error.__file__)


class Ereporter2OnErrorTest(test_case.TestCase):
  def setUp(self):
    super(Ereporter2OnErrorTest, self).setUp()
    self.mock(logging, 'error', lambda *_, **_kwargs: None)
    self._now = datetime.datetime(2014, 6, 24, 20, 19, 42, 653775)
    self.mock_now(self._now, 0)

  def test_log(self):
    kwargs = dict((k, k) for k in on_error.VALID_ERROR_KEYS)
    kwargs['args'] = ['args']
    kwargs['category'] = 'exception'
    kwargs['duration'] = 2.3
    kwargs['env'] = {'foo': 'bar'}
    kwargs['params'] = {'foo': 'bar'}
    kwargs['source'] = 'bot'
    kwargs['source_ip'] = '0.0.0.0'
    on_error.log(**kwargs)
    self.assertEqual(1, models.Error.query().count())
    expected = {
      'args': [u'args'],
      'category': u'exception',
      'created_ts': self._now,
      'cwd': u'cwd',
      'duration': 2.3,
      'endpoint': u'endpoint',
      'env': {u'foo': u'bar'},
      'exception_type': u'exception_type',
      'hostname': u'hostname',
      'identity': None,
      'message': u'message',
      'method': u'method',
      'os': u'os',
      'params': {u'foo': u'bar'},
      'python_version': u'python_version',
      'request_id': u'request_id',
      'source': u'bot',
      'source_ip': u'0.0.0.0',
      'stack': u'stack',
      'user': u'user',
      'version': u'version',
    }
    self.assertEqual(expected, models.Error.query().get().to_dict())

  def test_log_server(self):
    # version is automatiaclly added.
    on_error.log(source='server')
    self.assertEqual(1, models.Error.query().count())
    expected = dict((k, None) for k in on_error.VALID_ERROR_KEYS)
    expected['args'] = []
    expected['created_ts'] = self._now
    expected['identity'] = None
    expected['python_version'] = unicode(platform.python_version())
    expected['source'] = u'server'
    expected['source_ip'] = None
    expected['version'] = u'v1a'
    self.assertEqual(expected, models.Error.query().get().to_dict())

  def test_ignored_flag(self):
    on_error.log(foo='bar')
    self.assertEqual(1, models.Error.query().count())
    expected = {
      'args': [],
      'category': None,
      'created_ts': self._now,
      'cwd': None,
      'duration': None,
      'endpoint': None,
      'env': None,
      'exception_type': None,
      'hostname': None,
      'identity': None,
      'message': None,
      'method': None,
      'os': None,
      'params': None,
      'python_version': None,
      'request_id': None,
      'source': u'unknown',
      'source_ip': None,
      'stack': None,
      'user': None,
      'version': None,
    }
    self.assertEqual(expected, models.Error.query().get().to_dict())

  def test_exception(self):
    on_error.log(env='str')
    self.assertEqual(1, models.Error.query().count())
    relpath_on_error = formatter._relative_path(ON_ERROR_PATH)
    expected = {
      'args': [],
      'category': u'exception',
      'created_ts': self._now,
      'cwd': None,
      'duration': None,
      'endpoint': None,
      'env': None,
      'exception_type': u'<type \'exceptions.TypeError\'>',
      'hostname': None,
      'identity': None,
      'message':
          u'log({\'env\': \'str\'}) caused: JSON property must be a '
          u'<type \'dict\'>',
      'method': None,
      'os': None,
      'params': None,
      'python_version': None,
      'request_id': None,
      'source': u'server',
      'source_ip': None,
      'stack':
          u'Traceback (most recent call last):\n'
          u'  File "%s", line 0, in log\n'
          u'    error = models.Error(identity=identity, **kwargs)\n'
          u'  File "appengine/ext/ndb/model.py", line 0, in __init__\n' %
            relpath_on_error.replace('.pyc', '.py'),
      'user': None,
      'version': None,
    }
    actual = models.Error.query().get().to_dict()
    # Zap out line numbers to 0, it's annoying otherwise to update the unit test
    # just for line move. Only keep the first 4 lines because json_dict
    # verification is a tad deep insode ndb/model.py.
    actual['stack'] = ''.join(
        re.sub(r' \d+', ' 0', actual['stack']).splitlines(True)[:4])
    # Also make no distinction between *.pyc and *.py files.
    actual['stack'] = actual['stack'].replace('.pyc', '.py')
    self.assertEqual(expected, actual)

  def test_log_request(self):
    # Create a small adhoc webapp2 instance and ensures logging works.
    def handle(request):
      on_error.log_request(request)
    app = webtest.TestApp(webapp2.WSGIApplication([('/', handle)], debug=True))
    app.get('/?foo=bar')
    app.post('/?foo=bar', {'foo': 'baz'})
    # Strip None values for clarity.
    actual = [
      {k: v for k, v in e.to_dict().iteritems() if v is not None}
      for e in models.Error.query()
    ]
    # It happens this value is hardcoded on time.
    request_id = u'7357B3D7091D'
    expected = [
      {
        'args': [],
        'created_ts': self._now,
        'endpoint': u'/',
        'method': u'GET',
        'params': {u'foo': u'bar'},
        'request_id': request_id,
        'source': u'unknown',
      },
      {
        'args': [],
        'created_ts': self._now,
        'endpoint': u'/',
        'method': u'POST',
        'params': {u'foo': [u'bar', u'baz']},
        'request_id': request_id,
        'source': u'unknown',
      },
    ]
    self.assertEqual(expected, actual)


class Ereporter2OnErrorTestNoAuth(test_case.TestCase):
  def setUp(self):
    super(Ereporter2OnErrorTestNoAuth, self).setUp()
    self._now = datetime.datetime(2014, 6, 24, 20, 19, 42, 653775)
    self.mock_now(self._now, 0)

  def test_log(self):
    # It must work even if auth is not initialized.
    self.mock(logging, 'error', lambda *_, **_kwargs: None)
    error_id = on_error.log(
        source='bot', category='task_failure', message='Dang')
    self.assertEqual(1, models.Error.query().count())
    self.assertEqual(error_id, models.Error.query().get().key.integer_id())
    expected = {
      'args': [],
      'category': u'task_failure',
      'created_ts': self._now,
      'cwd': None,
      'duration': None,
      'endpoint': None,
      'env': None,
      'exception_type': None,
      'hostname': None,
      'identity': None,
      'message': u'Dang',
      'method': None,
      'os': None,
      'params': None,
      'python_version': None,
      'request_id': None,
      'source': u'bot',
      'source_ip': None,
      'stack': None,
      'user': None,
      'version': None,
    }
    self.assertEqual(expected, models.Error.query().get().to_dict())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
