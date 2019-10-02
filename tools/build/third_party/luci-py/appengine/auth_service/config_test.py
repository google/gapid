#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import config as config_component
from components import utils
from components.auth import model
from components.auth.proto import security_config_pb2
from components.config import validation
from test_support import test_case

from proto import config_pb2
import config


class ConfigTest(test_case.TestCase):
  def setUp(self):
    super(ConfigTest, self).setUp()
    self.mock_now(datetime.datetime(2014, 1, 2, 3, 4, 5))
    model.AuthGlobalConfig(
        key=model.root_key(),
        auth_db_rev=0,
    ).put()

  def test_refetch_config(self):
    initial_revs = {
      'a.cfg': config.Revision('old_a_rev', 'urla'),
      'b.cfg': config.Revision('old_b_rev', 'urlb'),
      'c.cfg': config.Revision('old_c_rev', 'urlc'),
    }

    revs = initial_revs.copy()
    bumps = []

    def bump_rev(pkg, rev, conf):
      revs[pkg] = rev
      bumps.append((pkg, rev, conf, ndb.in_transaction()))
      return True

    @ndb.tasklet
    def get_rev_async(pkg):
      raise ndb.Return(revs[pkg])

    self.mock(config, 'is_remote_configured', lambda: True)
    self.mock(config, '_CONFIG_SCHEMAS', {
      # Will be updated outside of auth db transaction.
      'a.cfg': {
        'proto_class': None,
        'revision_getter': lambda: get_rev_async('a.cfg'),
        'validator': lambda body: self.assertEqual(body, 'new a body'),
        'updater': lambda root, rev, conf: bump_rev('a.cfg', rev, conf),
        'use_authdb_transaction': False,
      },
      # Will not be changed.
      'b.cfg': {
        'proto_class': None,
        'revision_getter': lambda: get_rev_async('b.cfg'),
        'validator': lambda _body: True,
        'updater': lambda root, rev, conf: bump_rev('b.cfg', rev, conf),
        'use_authdb_transaction': False,
      },
      # Will be updated inside auth db transaction.
      'c.cfg': {
        'proto_class': None,
        'revision_getter': lambda: get_rev_async('c.cfg'),
        'validator': lambda body: self.assertEqual(body, 'new c body'),
        'updater': lambda root, rev, conf: bump_rev('c.cfg', rev, conf),
        'use_authdb_transaction': True,
      },
    })

    # _fetch_configs is called by config.refetch_config().
    configs_to_fetch = {
      'a.cfg': (config.Revision('new_a_rev', 'urla'), 'new a body'),
      'b.cfg': (config.Revision('old_b_rev', 'urlb'), 'old b body'),
      'c.cfg': (config.Revision('new_c_rev', 'urlc'), 'new c body'),
    }
    self.mock(config, '_fetch_configs', lambda _: configs_to_fetch)

    # Old revisions initially.
    self.assertEqual(initial_revs, config.get_revisions())

    # Initial update.
    config.refetch_config()
    self.assertEqual([
      ('a.cfg', config.Revision('new_a_rev', 'urla'), 'new a body', False),
      ('c.cfg', config.Revision('new_c_rev', 'urlc'), 'new c body', True),
    ], bumps)
    del bumps[:]

    # Updated revisions now.
    self.assertEqual(
        {k: v[0] for k, v in configs_to_fetch.iteritems()},
        config.get_revisions())

    # Refetch, nothing new.
    config.refetch_config()
    self.assertFalse(bumps)

  def test_update_imports_config(self):
    new_rev = config.Revision('rev', 'url')
    body = 'tarball{url:"a" systems:"b"}'
    self.assertTrue(config._update_imports_config(None, new_rev, body))
    self.assertEqual(
        new_rev, config._get_imports_config_revision_async().get_result())

  def test_validate_ip_whitelist_config_ok(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=['127.0.0.1/32', '0.0.0.0/0']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='bots',
              subnets=[],
              includes=['abc']),
        ],
        assignments=[
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:abc@example.com',
              ip_whitelist_name='abc'),
        ])
    config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_config_empty(self):
    config._validate_ip_whitelist_config(config_pb2.IPWhitelistConfig())

  def test_validate_ip_whitelist_config_bad_name(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(name='<bad name>'),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_config_duplicated_wl(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(name='abc'),
          config_pb2.IPWhitelistConfig.IPWhitelist(name='abc'),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_config_bad_subnet(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=['not a subnet']),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_config_bad_identity(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(name='abc')
        ],
        assignments=[
          config_pb2.IPWhitelistConfig.Assignment(
              identity='bad identity',
              ip_whitelist_name='abc'),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_config_unknown_whitelist(self):
    conf = config_pb2.IPWhitelistConfig(
        assignments=[
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:abc@example.com',
              ip_whitelist_name='missing'),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_config_identity_twice(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(name='abc'),
          config_pb2.IPWhitelistConfig.IPWhitelist(name='def'),
        ],
        assignments=[
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:abc@example.com',
              ip_whitelist_name='abc'),
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:abc@example.com',
              ip_whitelist_name='def'),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_unknown_include(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=[],
              includes=['unknown']),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_include_cycle_1(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=[],
              includes=['abc']),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_include_cycle_2(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=[],
              includes=['def']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='def',
              subnets=[],
              includes=['abc']),
        ])
    with self.assertRaises(ValueError):
      config._validate_ip_whitelist_config(conf)

  def test_validate_ip_whitelist_include_diamond(self):
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=[],
              includes=['middle1', 'middle2']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='middle1',
              subnets=[],
              includes=['inner']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='middle2',
              subnets=[],
              includes=['inner']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='inner',
              subnets=[]),
        ])
    config._validate_ip_whitelist_config(conf)

  def test_update_ip_whitelist_config(self):
    def run(conf):
      return config._update_authdb_configs({
        'ip_whitelist.cfg': (
          config.Revision('ip_whitelist_cfg_rev', 'http://url'), conf
        ),
      })
    # Pushing empty config to empty DB -> no changes.
    self.assertFalse(run(config_pb2.IPWhitelistConfig()))

    # Added a bunch of IP whitelists and assignments.
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=['0.0.0.1/32']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='bots',
              subnets=['0.0.0.2/32']),
          config_pb2.IPWhitelistConfig.IPWhitelist(name='empty'),
        ],
        assignments=[
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:abc@example.com',
              ip_whitelist_name='abc'),
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:def@example.com',
              ip_whitelist_name='bots'),
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:xyz@example.com',
              ip_whitelist_name='bots'),
        ])
    self.assertTrue(run(conf))

    # Verify everything is there.
    self.assertEqual({
      'assignments': [
        {
          'comment':
              u'Imported from ip_whitelist.cfg at rev ip_whitelist_cfg_rev',
          'created_by': model.Identity(kind='service', name='sample-app'),
          'created_ts': datetime.datetime(2014, 1, 2, 3, 4, 5),
          'identity': model.Identity(kind='user', name='abc@example.com'),
          'ip_whitelist': u'abc',
        },
        {
          'comment':
              u'Imported from ip_whitelist.cfg at rev ip_whitelist_cfg_rev',
          'created_by': model.Identity(kind='service', name='sample-app'),
          'created_ts': datetime.datetime(2014, 1, 2, 3, 4, 5),
          'identity': model.Identity(kind='user', name='def@example.com'),
          'ip_whitelist': u'bots',
        },
        {
          'comment':
              u'Imported from ip_whitelist.cfg at rev ip_whitelist_cfg_rev',
          'created_by': model.Identity(kind='service', name='sample-app'),
          'created_ts': datetime.datetime(2014, 1, 2, 3, 4, 5),
          'identity': model.Identity(kind='user', name='xyz@example.com'),
          'ip_whitelist': u'bots',
        },
      ],
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'modified_by': model.get_service_self_identity(),
      'modified_ts': datetime.datetime(2014, 1, 2, 3, 4, 5),
    }, model.ip_whitelist_assignments_key().get().to_dict())
    self.assertEqual(
        {
          'abc': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.1/32'],
          },
          'bots': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.2/32'],
          },
          'empty': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [],
          },
        },
        {
          x.key.id(): x.to_serializable_dict()
          for x in model.AuthIPWhitelist.query(ancestor=model.root_key())
        })

    # Exact same config a bit later -> no changes applied.
    self.mock_now(datetime.datetime(2014, 2, 2, 3, 4, 5))
    self.assertFalse(run(conf))

    # Modify whitelist, add new one, remove some. Same for assignments.
    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='abc',
              subnets=['0.0.0.3/32']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='bots',
              subnets=['0.0.0.2/32']),
          config_pb2.IPWhitelistConfig.IPWhitelist(name='another'),
        ],
        assignments=[
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:abc@example.com',
              ip_whitelist_name='abc'),
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:def@example.com',
              ip_whitelist_name='another'),
          config_pb2.IPWhitelistConfig.Assignment(
              identity='user:zzz@example.com',
              ip_whitelist_name='bots'),
        ])
    self.mock_now(datetime.datetime(2014, 3, 2, 3, 4, 5))
    self.assertTrue(run(conf))

    # Verify everything is there.
    self.assertEqual({
      'assignments': [
        {
          'comment':
              u'Imported from ip_whitelist.cfg at rev ip_whitelist_cfg_rev',
          'created_by': model.Identity(kind='service', name='sample-app'),
          'created_ts': datetime.datetime(2014, 1, 2, 3, 4, 5),
          'identity': model.Identity(kind='user', name='abc@example.com'),
          'ip_whitelist': u'abc',
        },
        {
          'comment':
              u'Imported from ip_whitelist.cfg at rev ip_whitelist_cfg_rev',
          'created_by': model.Identity(kind='service', name='sample-app'),
          'created_ts': datetime.datetime(2014, 3, 2, 3, 4, 5),
          'identity': model.Identity(kind='user', name='def@example.com'),
          'ip_whitelist': u'another',
        },
        {
          'comment':
              u'Imported from ip_whitelist.cfg at rev ip_whitelist_cfg_rev',
          'created_by': model.Identity(kind='service', name='sample-app'),
          'created_ts': datetime.datetime(2014, 3, 2, 3, 4, 5),
          'identity': model.Identity(kind='user', name='zzz@example.com'),
          'ip_whitelist': u'bots',
        },
      ],
      'auth_db_rev': 2,
      'auth_db_prev_rev': 1,
      'modified_by': model.get_service_self_identity(),
      'modified_ts': datetime.datetime(2014, 3, 2, 3, 4, 5),
    }, model.ip_whitelist_assignments_key().get().to_dict())
    self.assertEqual(
        {
          'abc': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1393729445000000,
            'subnets': [u'0.0.0.3/32'],
          },
          'bots': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.2/32'],
          },
          'another': {
            'created_by': 'service:sample-app',
            'created_ts': 1393729445000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1393729445000000,
            'subnets': [],
          },
        },
        {
          x.key.id(): x.to_serializable_dict()
          for x in model.AuthIPWhitelist.query(ancestor=model.root_key())
        })

  def test_update_ip_whitelist_config_with_includes(self):
    def run(conf):
      return config._update_authdb_configs({
        'ip_whitelist.cfg': (
          config.Revision('ip_whitelist_cfg_rev', 'http://url'), conf
        ),
      })

    conf = config_pb2.IPWhitelistConfig(
        ip_whitelists=[
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='a',
              subnets=['0.0.0.1/32']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='b',
              subnets=['0.0.0.1/32', '0.0.0.2/32'],
              includes=['a']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='c',
              subnets=['0.0.0.3/32'],
              includes=['a', 'b']),
          config_pb2.IPWhitelistConfig.IPWhitelist(
              name='d',
              includes=['c']),
        ])
    self.assertTrue(run(conf))

    # Verify everything is there.
    self.assertEqual(
        {
          'a': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.1/32'],
          },
          'b': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.1/32', u'0.0.0.2/32'],
          },
          'c': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.1/32', u'0.0.0.2/32', u'0.0.0.3/32'],
          },
          'd': {
            'created_by': 'service:sample-app',
            'created_ts': 1388631845000000,
            'description':
                u'Imported from ip_whitelist.cfg',
            'modified_by': 'service:sample-app',
            'modified_ts': 1388631845000000,
            'subnets': [u'0.0.0.1/32', u'0.0.0.2/32', u'0.0.0.3/32'],
          },
        },
        {
          x.key.id(): x.to_serializable_dict()
          for x in model.AuthIPWhitelist.query(ancestor=model.root_key())
        })

  def test_update_oauth_config(self):
    def run(conf):
      return config._update_authdb_configs({
        'oauth.cfg': (config.Revision('oauth_cfg_rev', 'http://url'), conf),
      })
    # Pushing empty config to empty state -> no changes.
    self.assertFalse(run(config_pb2.OAuthConfig()))
    # Updating config.
    self.assertTrue(run(config_pb2.OAuthConfig(
        primary_client_id='a',
        primary_client_secret='b',
        client_ids=['c', 'd'],
        token_server_url='https://token-server')))
    self.assertEqual({
      'auth_db_rev': 1,
      'auth_db_prev_rev': 0,
      'modified_by': model.get_service_self_identity(),
      'modified_ts': datetime.datetime(2014, 1, 2, 3, 4, 5),
      'oauth_additional_client_ids': [u'c', u'd'],
      'oauth_client_id': u'a',
      'oauth_client_secret': u'b',
      'security_config': None,
      'token_server_url': u'https://token-server',
    }, model.root_key().get().to_dict())
    # Same config again -> no changes.
    self.assertFalse(run(config_pb2.OAuthConfig(
        primary_client_id='a',
        primary_client_secret='b',
        client_ids=['c', 'd'],
        token_server_url='https://token-server')))

  def test_validate_oauth_config(self):
    with self.assertRaises(ValueError):
      config._validate_oauth_config(
          config_pb2.OAuthConfig(
            primary_client_id='a',
            primary_client_secret='b',
            client_ids=['c', 'd'],
            token_server_url='https://not-root-url/abc/def'))

  def test_fetch_configs_ok(self):
    fetches = {
      'imports.cfg': ('imports_cfg_rev', 'tarball{url:"a" systems:"b"}'),
      'ip_whitelist.cfg': (
          'ip_whitelist_cfg_rev', config_pb2.IPWhitelistConfig()),
      'oauth.cfg': (
          'oauth_cfg_rev', config_pb2.OAuthConfig(primary_client_id='a')),
      'settings.cfg': (None, None),  # emulate missing config
    }
    @ndb.tasklet
    def get_self_config_mock(path, *_args, **_kwargs):
      self.assertIn(path, fetches)
      raise ndb.Return(fetches.pop(path))
    self.mock(config_component, 'get_self_config_async', get_self_config_mock)
    self.mock(config, '_get_configs_url', lambda: 'http://url')
    result = config._fetch_configs(fetches.keys())
    self.assertFalse(fetches)
    self.assertEqual({
      'imports.cfg': (
          config.Revision('imports_cfg_rev', 'http://url'),
          'tarball{url:"a" systems:"b"}'),
      'ip_whitelist.cfg': (
          config.Revision('ip_whitelist_cfg_rev', 'http://url'),
          config_pb2.IPWhitelistConfig()),
      'oauth.cfg': (
          config.Revision('oauth_cfg_rev', 'http://url'),
          config_pb2.OAuthConfig(primary_client_id='a')),
      'settings.cfg': (config.Revision('0'*40, 'http://url'), ''),
    }, result)

  def test_fetch_configs_not_valid(self):
    @ndb.tasklet
    def get_self_config_mock(*_args, **_kwargs):
      raise ndb.Return(('imports_cfg_rev', 'bad config'))
    self.mock(config_component, 'get_self_config_async', get_self_config_mock)
    self.mock(config, '_get_configs_url', lambda: 'http://url')
    with self.assertRaises(config.CannotLoadConfigError):
      config._fetch_configs(['imports.cfg'])

  def test_gitiles_url(self):
    self.assertEqual(
        'https://host/repo/+/aaa/path/b/c.cfg',
        config._gitiles_url('https://host/repo/+/HEAD/path', 'aaa', 'b/c.cfg'))
    self.assertEqual(
        'https://not-gitiles',
        config._gitiles_url('https://not-gitiles', 'aaa', 'b/c.cfg'))

  def test_update_service_config(self):
    # Missing.
    self.assertIsNone(config._get_service_config('abc.cfg'))
    self.assertIsNone(
        config._get_service_config_rev_async('abc.cfg').get_result())
    # Updated.
    rev = config.Revision('rev', 'url')
    self.assertTrue(config._update_service_config('abc.cfg', rev, 'body'))
    self.assertEqual('body', config._get_service_config('abc.cfg'))
    self.assertEqual(
        rev, config._get_service_config_rev_async('abc.cfg').get_result())
    # Same body, returns False, though updates rev.
    rev2 = config.Revision('rev2', 'url')
    self.assertFalse(config._update_service_config('abc.cfg', rev2, 'body'))
    self.assertEqual(
        rev2, config._get_service_config_rev_async('abc.cfg').get_result())

  def test_settings_updates(self):
    # Fetch only settings.cfg in this test case.
    self.mock(config, 'is_remote_configured', lambda: True)
    self.mock(config, '_CONFIG_SCHEMAS', {
      'settings.cfg': config._CONFIG_SCHEMAS['settings.cfg'],
    })

    # Default settings.
    self.assertEqual(config_pb2.SettingsCfg(), config.get_settings())

    # Mock new settings value in luci-config.
    settings_cfg_text = 'enable_ts_monitoring: true'
    self.mock(config, '_fetch_configs', lambda _: {
      'settings.cfg': (config.Revision('rev', 'url'), settings_cfg_text),
    })

    # Fetch them.
    config.refetch_config()

    # Verify they are used now.
    utils.clear_cache(config.get_settings)
    self.assertEqual(
        config_pb2.SettingsCfg(enable_ts_monitoring=True),
        config.get_settings())

    # "Delete" them from luci-config.
    self.mock(config, '_fetch_configs', lambda _: {
      'settings.cfg': (config.Revision('0'*40, 'url'), ''),
    })

    # Fetch them.
    config.refetch_config()

    # Verify defaults are restored.
    utils.clear_cache(config.get_settings)
    self.assertEqual(config_pb2.SettingsCfg(), config.get_settings())

  def test_validate_security_config_ok(self):
    ctx = validation.Context()
    config.validate_security_config(security_config_pb2.SecurityConfig(), ctx)
    self.assertEqual(ctx.result().messages, [])

  def test_validate_security_config_bad_regexp(self):
    ctx = validation.Context()
    config.validate_security_config(security_config_pb2.SecurityConfig(
        internal_service_regexp=['???'],
    ), ctx)
    self.assertEqual(ctx.result().messages, [
      validation.Message(
          "internal_service_regexp: bad regexp '???' - nothing to repeat", 40),
    ])

  def test_update_security_config(self):
    def cfg(internal_service_regexp):
      return security_config_pb2.SecurityConfig(
          internal_service_regexp=internal_service_regexp)

    def run(conf):
      return config._update_authdb_configs({
        'security.cfg': (config.Revision('cfg_rev', 'http://url'), conf),
      })

    def extract():
      d = model.root_key().get()
      return {
        'auth_db_rev': d.auth_db_rev,
        'security_config': security_config_pb2.SecurityConfig.FromString(
            d.security_config),
      }

    # Pushing empty config -> no changes.
    self.assertFalse(run(cfg([])))

    # Updating the config.
    self.assertTrue(run(cfg([r'example\.com'])))
    self.assertEqual({
      'auth_db_rev': 1,
      'security_config': cfg([r'example\.com']),
    }, extract())

    # Pushing same config again. No changes.
    self.assertFalse(run(cfg([r'example\.com'])))

  def test_update_two_authdb_cfgs(self):
    """It is OK to update oauth.cfg and security.cfg at once."""
    def oauth_cfg(client_id):
      return config_pb2.OAuthConfig(primary_client_id=client_id)
    def sec_cfg(regexps):
      return security_config_pb2.SecurityConfig(internal_service_regexp=regexps)

    def run(oauth, sec):
      return config._update_authdb_configs({
        'oauth.cfg': (config.Revision('cfg_rev', 'http://url'), oauth),
        'security.cfg': (config.Revision('cfg_rev', 'http://url'), sec),
      })

    def extract():
      d = model.root_key().get()
      sec = security_config_pb2.SecurityConfig()
      if d.security_config:
        sec.MergeFromString(d.security_config)
      return {
        'auth_db_rev': d.auth_db_rev,
        'oauth_client_id': d.oauth_client_id,
        'security_config': sec,
      }

    # Both are empty when applied to empty state. No changes.
    self.assertFalse(run(oauth_cfg(''), sec_cfg([])))
    self.assertEqual({
      'auth_db_rev': 0,
      'oauth_client_id': u'',
      'security_config': sec_cfg([]),
    }, extract())

    # Both have changes. AuthDB revision is bumped only once. Both changes are
    # preserved.
    self.assertTrue(run(oauth_cfg('z'), sec_cfg(['z'])))
    self.assertEqual({
      'auth_db_rev': 1,
      'oauth_client_id': u'z',
      'security_config': sec_cfg(['z']),
    }, extract())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
