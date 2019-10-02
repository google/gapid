#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import datetime
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import mock

from google.appengine.ext import ndb

from components import auth
from components import net
from components.config import remote
from test_support import test_case

import test_config_pb2


class RemoteTestCase(test_case.TestCase):
  def setUp(self):
    super(RemoteTestCase, self).setUp()
    self.mock(net, 'json_request_async', mock.Mock())
    net.json_request_async.side_effect = self.json_request_async

    self.provider = remote.Provider('luci-config.appspot.com')
    provider_future = ndb.Future()
    provider_future.set_result(self.provider)
    self.mock(remote, 'get_provider_async', lambda: provider_future)

  @ndb.tasklet
  def json_request_async(self, url, **kwargs):
    assert kwargs['scopes']
    URL_PREFIX = 'https://luci-config.appspot.com/_ah/api/config/v1/'
    if url == URL_PREFIX + 'config_sets/services%2Ffoo/config/bar.cfg':
      assert kwargs['params']['hash_only']
      raise ndb.Return({
        'content_hash': 'deadbeef',
        'revision': 'aaaabbbb',
      })
    if url == URL_PREFIX + 'config_sets/services%2Ffoo/config/baz.cfg':
      assert kwargs['params']['hash_only']
      raise ndb.Return({
        'content_hash': 'badcoffee',
        'revision': 'aaaabbbb',
      })

    if url == URL_PREFIX + 'config/deadbeef':
      raise ndb.Return({
        'content':  base64.b64encode('a config'),
      })
    if url == URL_PREFIX + 'config/badcoffee':
      raise ndb.Return({
        'content':  base64.b64encode('param: "qux"'),
      })

    if url == URL_PREFIX + 'projects':
      raise ndb.Return({
        'projects':[
          {
           'id': 'chromium',
           'repo_type': 'GITILES',
           'repo_url': 'https://chromium.googlesource.com/chromium/src',
           'name': 'Chromium browser'
          },
          {
           'id': 'infra',
           'repo_type': 'GITILES',
           'repo_url': 'https://chromium.googlesource.com/infra/infra',
          },
        ]
      })
    self.fail('Unexpected url: %s' % url)

  def test_get_async(self):
    revision, content = self.provider.get_async(
        'services/foo', 'bar.cfg').get_result()
    self.assertEqual(revision, 'aaaabbbb')
    self.assertEqual(content, 'a config')

    # Memcache coverage
    net.json_request_async.reset_mock()
    revision, content = self.provider.get_async(
        'services/foo', 'bar.cfg').get_result()
    self.assertEqual(revision, 'aaaabbbb')
    self.assertEqual(content, 'a config')
    self.assertFalse(net.json_request_async.called)

  def test_get_async_with_revision(self):
    revision, content = self.provider.get_async(
        'services/foo', 'bar.cfg', revision='aaaabbbb').get_result()
    self.assertEqual(revision, 'aaaabbbb')
    self.assertEqual(content, 'a config')

    net.json_request_async.assert_any_call(
        'https://luci-config.appspot.com/_ah/api/config/v1/'
        'config_sets/services%2Ffoo/config/bar.cfg',
        params={'hash_only': True, 'revision': 'aaaabbbb'},
        scopes=net.EMAIL_SCOPE)

    # Memcache coverage
    net.json_request_async.reset_mock()
    revision, content = self.provider.get_async(
        'services/foo', 'bar.cfg', revision='aaaabbbb').get_result()
    self.assertEqual(revision, 'aaaabbbb')
    self.assertEqual(content, 'a config')
    self.assertFalse(net.json_request_async.called)

  def test_last_good(self):
    revision, content = self.provider.get_async(
        'services/foo', 'bar.cfg', store_last_good=True).get_result()
    self.assertIsNone(revision)
    self.assertIsNone(content)

    self.assertTrue(remote.LastGoodConfig.get_by_id('services/foo:bar.cfg'))
    remote.LastGoodConfig(
        id='services/foo:bar.cfg',
        content='a config',
        content_hash='deadbeef',
        revision='aaaaaaaa').put()

    revision, content = self.provider.get_async(
        'services/foo', 'bar.cfg', store_last_good=True).get_result()
    self.assertEqual(revision, 'aaaaaaaa')
    self.assertEqual(content, 'a config')

    self.assertFalse(net.json_request_async.called)

  def test_get_projects(self):
    projects = self.provider.get_projects_async().get_result()
    self.assertEqual(projects, [
      {
       'id': 'chromium',
       'repo_type': 'GITILES',
       'repo_url': 'https://chromium.googlesource.com/chromium/src',
       'name': 'Chromium browser'
      },
      {
       'id': 'infra',
       'repo_type': 'GITILES',
       'repo_url': 'https://chromium.googlesource.com/infra/infra',
      },
    ])

  def test_get_project_configs_async_receives_404(self):
    net.json_request_async.side_effect = net.NotFoundError(
        'Not found', 404, None)
    with self.assertRaises(net.NotFoundError):
      self.provider.get_project_configs_async('cfg').get_result()

  def test_get_project_configs_async(self):
    self.mock(net, 'json_request_async', mock.Mock())
    net.json_request_async.return_value = ndb.Future()
    net.json_request_async.return_value.set_result({
      'configs': [
        {
          'config_set': 'projects/chromium',
          'content_hash': 'deadbeef',
          'path': 'cfg',
          'revision': 'aaaaaaaa',
        }
      ]
    })
    self.mock(self.provider, 'get_config_by_hash_async', mock.Mock())
    self.provider.get_config_by_hash_async.return_value = ndb.Future()
    self.provider.get_config_by_hash_async.return_value.set_result('a config')

    configs = self.provider.get_project_configs_async('cfg').get_result()

    self.assertEqual(configs, {'projects/chromium': ('aaaaaaaa', 'a config')})

  def test_get_config_set_location_async(self):
    self.mock(net, 'json_request_async', mock.Mock())
    net.json_request_async.return_value = ndb.Future()
    net.json_request_async.return_value.set_result({
      'mappings': [
        {
          'config_set': 'services/abc',
          'location': 'http://example.com',
        },
      ],
    })
    r = self.provider.get_config_set_location_async('services/abc').get_result()
    self.assertEqual(r, 'http://example.com')
    net.json_request_async.assert_called_once_with(
        'https://luci-config.appspot.com/_ah/api/config/v1/mapping',
        scopes=net.EMAIL_SCOPE,
        params={'config_set': 'services/abc'})

  def test_cron_update_last_good_configs(self):
    self.provider.get_async(
        'services/foo', 'bar.cfg', store_last_good=True).get_result()
    self.provider.get_async(
        'services/foo', 'baz.cfg', dest_type=test_config_pb2.Config,
        store_last_good=True).get_result()

    # Will be removed.
    old_cfg = remote.LastGoodConfig(
        id='projects/old:foo.cfg',
        content_hash='aaaa',
        content='content',
        last_access_ts=datetime.datetime(2010, 1, 1))
    old_cfg.put()

    remote.cron_update_last_good_configs()

    revision, config = self.provider.get_async(
        'services/foo', 'bar.cfg', store_last_good=True).get_result()
    self.assertEqual(revision, 'aaaabbbb')
    self.assertEqual(config, 'a config')

    revision, config = self.provider.get_async(
        'services/foo', 'baz.cfg', dest_type=test_config_pb2.Config,
        store_last_good=True).get_result()
    self.assertEqual(revision, 'aaaabbbb')
    self.assertEqual(config.param, 'qux')

    baz_cfg = remote.LastGoodConfig.get_by_id(id='services/foo:baz.cfg')
    self.assertIsNotNone(baz_cfg)
    self.assertEquals(baz_cfg.content_binary, config.SerializeToString())

    self.assertIsNone(old_cfg.key.get())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
