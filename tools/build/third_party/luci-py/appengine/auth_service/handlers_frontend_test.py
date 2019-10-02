#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

import webtest

from components import auth
from components import auth_testing
from components import template
from test_support import test_case

import handlers_frontend
import importer
import replication


GOOD_IMPORTER_CONFIG = """
# Comment.
tarball {
  domain: "example.com"
  systems: "ldap"
  url: "http://example.com/stuff.tar.gz"
}
plainlist {
  group: "chromium-committers"
  url: "http://chromium-committers.appspot.com/chromium"
}
"""

BAD_IMPORTER_CONFIG = """
# Missing 'url'.
tarball {
  domain: "example.com"
  systems: "ldap"
}
"""


class FrontendHandlersTest(test_case.TestCase):
  """Tests the frontend handlers."""

  def setUp(self):
    super(FrontendHandlersTest, self).setUp()
    self.mock(replication, 'trigger_replication', lambda *_args, **_kws: None)
    self.app = webtest.TestApp(
        handlers_frontend.create_application(debug=True),
        extra_environ={'REMOTE_ADDR': '127.0.0.1'})

  def tearDown(self):
    try:
      template.reset()
    finally:
      super(FrontendHandlersTest, self).tearDown()

  def mock_oauth_authentication(self):
    def mocked(request):
      hdr = request.headers.get('Authorization')
      if not hdr:
        return None, False
      if not hdr.startswith('Bearer '):
        raise auth.AuthenticationError()
      email = hdr[len('Bearer '):]
      return auth.Identity(auth.IDENTITY_USER, email), None
    self.mock(auth, 'oauth_authentication', mocked)

  def mock_ingest_tarball(self, raise_error=False):
    calls = []
    def mocked(name, content):
      if raise_error:
        raise importer.BundleImportError('Import error')
      calls.append((auth.get_current_identity(), name, content))
      return ['a', 'b', 'c'], 123
    self.mock(handlers_frontend.importer, 'ingest_tarball', mocked)
    return calls

  def mock_admin(self):
    auth_testing.mock_is_admin(self, True)
    auth_testing.mock_get_current_identity(self)

  def test_warmup(self):
    response = self.app.get('/_ah/warmup')
    self.assertEqual(200, response.status_code)
    self.assertEqual('ok', response.body)

  def test_importer_config_get_default(self):
    self.mock_admin()
    response = self.app.get('/auth_service/api/v1/importer/config', status=200)
    self.assertEqual({'config': ''}, response.json)

  def test_importer_config_get(self):
    self.mock_admin()
    importer.write_config(GOOD_IMPORTER_CONFIG)
    response = self.app.get('/auth_service/api/v1/importer/config', status=200)
    self.assertEqual({'config': GOOD_IMPORTER_CONFIG}, response.json)

  def test_importer_config_post_ok(self):
    self.mock_admin()
    response = self.app.post_json(
        '/auth_service/api/v1/importer/config',
        {'config': GOOD_IMPORTER_CONFIG},
        headers={'X-XSRF-Token': auth_testing.generate_xsrf_token_for_test()},
        status=200)
    self.assertEqual({'ok': True}, response.json)
    self.assertEqual(GOOD_IMPORTER_CONFIG, importer.read_config())

  def test_importer_config_post_bad(self):
    self.mock_admin()
    response = self.app.post_json(
        '/auth_service/api/v1/importer/config',
        {'config': BAD_IMPORTER_CONFIG},
        headers={'X-XSRF-Token': auth_testing.generate_xsrf_token_for_test()},
        status=400)
    self.assertEqual(
        {'text': '"url" field is required in TarballEntry'}, response.json)
    self.assertEqual('', importer.read_config())

  def test_importer_config_post_locked(self):
    self.mock_admin()
    self.mock(handlers_frontend.config, 'is_remote_configured', lambda: True)
    response = self.app.post_json(
        '/auth_service/api/v1/importer/config',
        {'config': GOOD_IMPORTER_CONFIG},
        headers={'X-XSRF-Token': auth_testing.generate_xsrf_token_for_test()},
        status=409)
    self.assertEqual(
        {'text': 'The configuration is managed elsewhere'}, response.json)

  def test_importer_ingest_tarball_ok(self):
    self.mock_oauth_authentication()
    calls = self.mock_ingest_tarball()
    response = self.app.put(
        '/auth_service/api/v1/importer/ingest_tarball/zzz',
        'tar body',
        headers={'Authorization': 'Bearer xxx@example.com'})
    self.assertEqual(
        {u'auth_db_rev': 123, u'groups': [u'a', u'b', u'c']}, response.json)
    self.assertEqual(
      [(auth.Identity(kind='user', name='xxx@example.com'), 'zzz', 'tar body')],
      calls)

  def test_importer_ingest_tarball_error(self):
    self.mock_oauth_authentication()
    self.mock_ingest_tarball(raise_error=True)
    response = self.app.put(
        '/auth_service/api/v1/importer/ingest_tarball/zzz',
        'tar body',
        headers={'Authorization': 'Bearer xxx@example.com'},
        status=400)
    self.assertEqual({'error': 'Import error'}, response.json)

  def test_importer_ingest_tarball_no_creds(self):
    self.mock_oauth_authentication()
    calls = self.mock_ingest_tarball()
    self.app.put(
        '/auth_service/api/v1/importer/ingest_tarball/zzz',
        'tar body',
        status=403)
    self.assertEqual([], calls)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
