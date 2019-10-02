#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import hashlib
import json
import logging
import os
import sys
import unittest
import zlib

import test_env
test_env.setup_test_env()

import cloudstorage
from google.appengine.ext import ndb

import webtest

from components import auth
from components import auth_testing
from components import template
from components import utils
from test_support import test_case

import config
import gcs
import handlers_frontend
import model

# Access to a protected member _XXX of a client class
# pylint: disable=W0212


class MainTest(test_case.TestCase):
  """Tests the handlers."""
  APP_DIR = test_env.APP_DIR

  def setUp(self):
    """Creates a new app instance for every test case."""
    super(MainTest, self).setUp()
    self.testbed.init_user_stub()

    self.source_ip = '192.168.0.1'
    self.app = webtest.TestApp(
        handlers_frontend.create_application(debug=True),
        extra_environ={'REMOTE_ADDR': self.source_ip})

    self.auth_app = webtest.TestApp(
        auth.create_wsgi_application(debug=True),
        extra_environ={
          'REMOTE_ADDR': self.source_ip,
          'SERVER_SOFTWARE': os.environ['SERVER_SOFTWARE'],
        })

    full_access_group = config.settings().auth.full_access_group
    readonly_access_group = config.settings().auth.readonly_access_group

    auth.bootstrap_group(
        auth.ADMIN_GROUP,
        [auth.Identity(auth.IDENTITY_USER, 'admin@example.com')])
    auth.bootstrap_group(
        readonly_access_group,
        [auth.Identity(auth.IDENTITY_USER, 'reader@example.com')])
    auth.bootstrap_group(
        full_access_group,
        [auth.Identity(auth.IDENTITY_USER, 'writer@example.com')])
    self.set_as_anonymous()

  def tearDown(self):
    template.reset()
    super(MainTest, self).tearDown()

  def set_as_anonymous(self):
    self.testbed.setup_env(USER_EMAIL='', overwrite=True)
    auth.ip_whitelist_key(auth.bots_ip_whitelist()).delete()
    auth_testing.reset_local_state()

  def set_as_admin(self):
    self.set_as_anonymous()
    self.testbed.setup_env(USER_EMAIL='admin@example.com', overwrite=True)

  def set_as_reader(self):
    self.set_as_anonymous()
    self.testbed.setup_env(USER_EMAIL='reader@example.com', overwrite=True)

  @staticmethod
  def gen_content_inline(namespace='default', content='Foo', is_isolated=False):
    hashhex = hashlib.sha1(content).hexdigest()
    key = model.get_entry_key(namespace, hashhex)
    model.new_content_entry(
        key,
        is_isolated=is_isolated,
        content=content,
        compressed_size=len(content),
        expanded_size=len(content),
        is_verified=True).put()
    return hashhex

  def get_xsrf_token(self):
    """Gets the generic XSRF token for web clients."""
    resp = self.auth_app.post(
        '/auth/api/v1/accounts/self/xsrf_token',
        headers={'X-XSRF-Token-Request': '1'}).json
    return resp['xsrf_token'].encode('ascii')

  def test_root(self):
    # Just asserts it doesn't crash.
    self.app.get('/')

  def test_browse(self):
    self.set_as_reader()
    hashhex = self.gen_content_inline()
    self.app.get('/browse?namespace=default&digest=%s' % hashhex)
    self.app.get('/browse?namespace=default&digest=%s&as=file1.txt' % hashhex)

  def test_browse_isolated(self):
    self.set_as_reader()
    content = json.dumps({'algo': 'sha1', 'includes': ['hash1']})
    hashhex = self.gen_content_inline(content=content, is_isolated=True)
    self.app.get('/browse?namespace=default&digest=%s' % hashhex)
    self.app.get('/browse?namespace=default&digest=%s&as=file1.txt' % hashhex)

  def test_browse_missing(self):
    self.set_as_reader()
    hashhex = '0123456780123456780123456789990123456789'
    self.app.get('/browse?namespace=default&digest=%s' % hashhex, status=404)

  def test_content(self):
    self.set_as_reader()
    hashhex = self.gen_content_inline(content='Foo')
    resp = self.app.get('/content?namespace=default&digest=%s' % hashhex)
    self.assertEqual('Foo', resp.body)
    resp = self.app.get(
        '/content?namespace=default&digest=%s&as=file1.txt' % hashhex)
    self.assertEqual('Foo', resp.body)

  def test_content_isolated(self):
    self.set_as_reader()
    content = json.dumps({'algo': 'sha1', 'includes': ['hash1']})
    hashhex = self.gen_content_inline(content=content, is_isolated=True)
    resp = self.app.get('/content?namespace=default&digest=%s' % hashhex)
    self.assertTrue(resp.body.startswith('<style>'), resp.body)

  def test_content_gcs(self):
    content = 'Foo'
    compressed = zlib.compress(content)
    namespace = 'default-gzip'
    hashhex = hashlib.sha1(content).hexdigest()

    def read_file(bucket, key):
      self.assertEqual(u'sample-app', bucket)
      self.assertEqual(namespace + '/' + hashhex, key)
      return [compressed]
    self.mock(gcs, 'read_file', read_file)

    key = model.get_entry_key(namespace, hashhex)
    model.new_content_entry(
        key,
        is_isolated=False,
        compressed_size=len(compressed),
        expanded_size=len(content),
        is_verified=True).put()

    self.set_as_reader()
    resp = self.app.get('/content?namespace=default-gzip&digest=%s' % hashhex)
    self.assertEqual(content, resp.body)
    resp = self.app.get(
        '/content?namespace=default-gzip&digest=%s&as=file1.txt' % hashhex)
    self.assertEqual(content, resp.body)
    self.assertNotEqual(None, key.get())

  def test_content_gcs_missing(self):
    content = 'Foo'
    compressed = zlib.compress(content)
    namespace = 'default-gzip'
    hashhex = hashlib.sha1(content).hexdigest()

    def read_file(bucket, key):
      self.assertEqual(u'sample-app', bucket)
      self.assertEqual(namespace + '/' + hashhex, key)
      raise cloudstorage.NotFoundError('Someone deleted the file from GCS')
    self.mock(gcs, 'read_file', read_file)

    key = model.get_entry_key(namespace, hashhex)
    model.new_content_entry(
        key,
        is_isolated=False,
        compressed_size=len(compressed),
        expanded_size=len(content),
        is_verified=True).put()

    self.set_as_reader()
    self.app.get(
        '/content?namespace=default-gzip&digest=%s' % hashhex, status=404)
    self.assertEqual(None, key.get())

  def test_config(self):
    self.set_as_admin()
    resp = self.app.get('/restricted/config')
    # TODO(maruel): Use beautifulsoup?
    priv_key = 'test private key'
    params = {
      'gs_private_key': priv_key,
      'keyid': str(config.settings_info()['cfg'].key.integer_id()),
      'xsrf_token': self.get_xsrf_token(),
    }
    self.assertEqual('', config.settings().gs_private_key)
    resp = self.app.post('/restricted/config', params)
    self.assertNotIn('Update conflict', resp)
    self.assertEqual(priv_key, config.settings().gs_private_key)

  def test_config_conflict(self):
    self.set_as_admin()
    resp = self.app.get('/restricted/config')
    # TODO(maruel): Use beautifulsoup?
    params = {
      'google_analytics': 'foobar',
      'keyid': str(config.settings().key.integer_id() - 1),
      'reusable_task_age_secs': 30,
      'xsrf_token': self.get_xsrf_token(),
    }
    self.assertEqual('', config.settings().google_analytics)
    resp = self.app.post('/restricted/config', params)
    self.assertIn('Update conflict', resp)
    self.assertEqual('', config.settings().google_analytics)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
