#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

from components.config import validation
from components import utils
from test_support import test_case

from proto.config import config_pb2
from server import config


# pylint: disable=W0212,W0612


class ConfigTest(test_case.TestCase):
  def setUp(self):
    super(ConfigTest, self).setUp()

    utils.clear_cache(config.settings)

  def validator_test(self, validator, cfg, messages):
    ctx = validation.Context()
    validator(cfg, ctx)
    self.assertEquals(ctx.result().messages, [
      validation.Message(severity=logging.ERROR, text=m)
      for m in messages
    ])

  def test_validate_flat_dimension(self):
    self.assertTrue(config.validate_flat_dimension(u'a:b'))
    # Broken flat dimensions.
    self.assertFalse(config.validate_flat_dimension('a:b'))
    self.assertFalse(config.validate_flat_dimension(u'a:'))
    self.assertFalse(config.validate_flat_dimension(u':b'))
    self.assertFalse(config.validate_flat_dimension(u'ab'))

  def test_validate_flat_dimension_key(self):
    l = config.DIMENSION_KEY_LENGTH
    self.assertTrue(config.validate_flat_dimension(u'a'*l + u':b'))
    self.assertFalse(config.validate_flat_dimension(u'a'*(l+1) + u':b'))

  def test_validate_flat_dimension_value(self):
    l = config.DIMENSION_VALUE_LENGTH
    self.assertTrue(config.validate_flat_dimension(u'a:' + u'b'*l))
    self.assertFalse(config.validate_flat_dimension(u'a:' + u'b'*(l+1)))

  def test_validate_dimension_key(self):
    self.assertTrue(config.validate_dimension_key(u'b'))
    self.assertTrue(config.validate_dimension_key(u'-'))
    self.assertFalse(config.validate_dimension_key(u''))
    self.assertFalse(config.validate_dimension_key(u'+'))

  def test_validate_dimension_key_length(self):
    l = config.DIMENSION_KEY_LENGTH
    self.assertTrue(config.validate_dimension_key(u'b'*l))
    self.assertFalse(config.validate_dimension_key(u'b'*(l+1)))

  def test_validate_dimension_value(self):
    self.assertTrue(config.validate_dimension_value(u'b'))
    self.assertFalse(config.validate_dimension_value(u''))
    self.assertFalse(config.validate_dimension_value(u' a'))

  def test_validate_dimension_value_length(self):
    l = config.DIMENSION_VALUE_LENGTH
    self.assertTrue(config.validate_dimension_value(u'b'*l))
    self.assertFalse(config.validate_dimension_value(u'b'*(l+1)))

  def test_validate_isolate_settings(self):
    self.validator_test(
        config._validate_isolate_settings,
        config_pb2.IsolateSettings(
            default_server='https://isolateserver.appspot.com'),
        [
          'either specify both default_server and default_namespace or '
            'none',
        ])

    self.validator_test(
        config._validate_isolate_settings,
        config_pb2.IsolateSettings(
            default_server='isolateserver.appspot.com',
            default_namespace='abc',
        ),
        [
          'default_server must start with "https://" or "http://localhost"',
        ])

    self.validator_test(
        config._validate_isolate_settings,
        config_pb2.IsolateSettings(
          default_server='https://isolateserver.appspot.com',
          default_namespace='abc',
        ),
        [])

    self.validator_test(
        config._validate_isolate_settings,
        config_pb2.IsolateSettings(),
        [])

  def test_validate_cipd_settings(self):
    self.validator_test(
        config._validate_cipd_settings,
        config_pb2.CipdSettings(),
        [
          'default_server is not set',
          'default_client_package: invalid package_name ""',
          'default_client_package: invalid version ""',
        ])

    self.validator_test(
        config._validate_cipd_settings,
        config_pb2.CipdSettings(
            default_server='chrome-infra-packages.appspot.com',
            default_client_package=config_pb2.CipdPackage(
                package_name='infra/tools/cipd/windows-i386',
                version='git_revision:deadbeef'),
            ),
        [
          'default_server must start with "https://" or "http://localhost"',
        ])

    self.validator_test(
        config._validate_cipd_settings,
        config_pb2.CipdSettings(
            default_server='https://chrome-infra-packages.appspot.com',
            default_client_package=config_pb2.CipdPackage(
                package_name='infra/tools/cipd/${platform}',
                version='git_revision:deadbeef'),
            ),
        [])

  def test_validate_settings(self):
    self.validator_test(
        config._validate_settings,
        config_pb2.SettingsCfg(
            bot_death_timeout_secs=-1,
            reusable_task_age_secs=-1),
      [
        'bot_death_timeout_secs cannot be negative',
        'reusable_task_age_secs cannot be negative',
      ])

    self.validator_test(
        config._validate_settings,
        config_pb2.SettingsCfg(
            bot_death_timeout_secs=config._SECONDS_IN_YEAR + 1,
            reusable_task_age_secs=config._SECONDS_IN_YEAR + 1),
      [
        'bot_death_timeout_secs cannot be more than a year',
        'reusable_task_age_secs cannot be more than a year',
      ])

    self.validator_test(
        config._validate_settings,
        config_pb2.SettingsCfg(
            display_server_url_template='http://foo/bar',
            extra_child_src_csp_url=['http://alpha/beta', 'https://']),
      [
        'display_server_url_template URL http://foo/bar must be https',
        'extra_child_src_csp_url URL http://alpha/beta must be https',
        'extra_child_src_csp_url URL https:// must be https',
      ])

    self.validator_test(
        config._validate_settings,
        config_pb2.SettingsCfg(
            display_server_url_template='https://foo/',
            extra_child_src_csp_url=['https://alpha/beta/']), [])

    self.validator_test(config._validate_settings, config_pb2.SettingsCfg(), [])

    self.validator_test(
        config._validate_settings,
        config_pb2.SettingsCfg(
            mp=config_pb2.MachineProviderSettings(server='http://url')),
      [
        'mp.server must start with "https://" or "http://localhost"',
      ])

    self.validator_test(
        config._validate_settings,
        config_pb2.SettingsCfg(
            mp=config_pb2.MachineProviderSettings(server='url')),
      [
        'mp.server must start with "https://" or "http://localhost"',
      ])

  def test_get_settings_with_defaults_from_none(self):
    """Make sure defaults are applied even if raw config is None."""
    self.mock(config, '_get_settings', lambda: (None, None))
    _, cfg = config._get_settings_with_defaults()
    self.assertEqual(cfg.reusable_task_age_secs, 7*24*60*60)
    self.assertEqual(cfg.bot_death_timeout_secs, 10*60)
    self.assertEqual(cfg.auth.admins_group, 'administrators')
    self.assertEqual(cfg.auth.bot_bootstrap_group, 'administrators')
    self.assertEqual(cfg.auth.privileged_users_group, 'administrators')
    self.assertEqual(cfg.auth.users_group, 'administrators')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
