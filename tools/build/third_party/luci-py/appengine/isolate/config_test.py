#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

from components.config import validation
from components import net
from components import utils
from test_support import test_case
import components.config

from proto import config_pb2
import config


# pylint: disable=W0212,W0612


class ConfigTest(test_case.TestCase):
  def setUp(self):
    super(ConfigTest, self).setUp()

    utils.clear_cache(config._get_settings_cached)
    config.GlobalConfig.clear_cache()

  def validator_test(self, cfg, messages):
    ctx = validation.Context()
    self.assertIsInstance(cfg, config_pb2.SettingsCfg)
    config._validate_settings(cfg, ctx)
    self.assertEquals(ctx.result().messages, [
      validation.Message(severity=logging.ERROR, text=m)
      for m in messages
    ])

  def test_validate_defaults(self):
    self.validator_test(config.settings(fresh=True)._cfg, [])

  def test_validate_defaults_cached(self):
    self.validator_test(config.settings()._cfg, [])

  def test_validate_default_expiration(self):
    cfg = config.settings()._cfg
    cfg.default_expiration = -1
    self.validator_test(cfg, ['default_expiration cannot be negative'])

  def test_validate_sharding_letters(self):
    cfg = config.settings()._cfg
    cfg.sharding_letters = -1
    self.validator_test(cfg, ['sharding_letters must be within [0..5]'])
    cfg.sharding_letters = 6
    self.validator_test(cfg, ['sharding_letters must be within [0..5]'])

  def test_validate_gs_bucket(self):
    cfg = config.settings()._cfg
    cfg.gs_bucket = 'b@d_b1cket'
    self.validator_test(cfg, ['gs_bucket invalid value: b@d_b1cket'])

  def test_validate_gs_client_id_email(self):
    cfg = config.settings()._cfg
    cfg.gs_client_id_email = 'not.an.email'
    self.validator_test(cfg, ['gs_client_id_email invalid value: not.an.email'])

    cfg.gs_client_id_email = 'valid@email.net'
    self.validator_test(cfg, [])

  def test_settings_info(self):
    url = 'https://test-config.com'
    self.mock(components.config, 'get_config_set_location', lambda _: url)
    d = config.settings_info()
    self.assertEqual(d['rev_url'], url)

  def test_get_configs_url_exception(self):
    """Test that 'except' specifies exceptions correctly."""
    def mock_raise(_):
      raise net.Error('test', 404, None)
    self.mock(components.config, 'get_config_set_location', mock_raise)
    self.assertIsNone(config._get_configs_url())

  def test_gitiles_url(self):
    url = 'https://gitiles.net/repo/+/master/'
    self.assertEqual('https://gitiles.net/repo/+/testrev/settings.cfg',
                     config._gitiles_url(url, 'testrev', 'settings.cfg'))

  def test_gitiles_url_none(self):
    self.assertIsNone(config._gitiles_url(None, 'testrev', 'settings.cfg'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
