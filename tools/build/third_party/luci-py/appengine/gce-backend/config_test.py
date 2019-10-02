#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for config.py."""

import logging
import unittest

import test_env
test_env.setup_test_env()

from components import datastore_utils
from components import machine_provider
from components.config import validation
from test_support import test_case

import config
from proto import config_pb2


class UpdateConfigTest(test_case.TestCase):
  """Tests for config.update_template_configs."""

  def tearDown(self, *args, **kwargs):
    """Performs post-test case tear-down."""
    super(UpdateConfigTest, self).tearDown(*args, **kwargs)

    # Even though the datastore resets between test cases and
    # the underlying entity doesn't persist, the cache does.
    config.Configuration.clear_cache()

  def validator_test(self, validator, cfg, messages):
    ctx = validation.Context()
    validator(cfg, ctx)
    self.assertEquals(ctx.result().messages, [
      validation.Message(severity=logging.ERROR, text=m)
      for m in messages
    ])

  def install_mock(
      self,
      revision=None,
      template_config=None,
      manager_config=None,
      settings_config=None,
      metadata_file=None,
  ):
    """Installs a mock for config.config.get_self_config.

    Args:
      template_config: What to return when templates.cfg is requested. Defaults
        to an empty config_pb2.InstanceTemplateConfig instance.
      manager_config: What to return when managers.cfg is requested. Defaults
        to an empty config_pb2.InstanceGroupManagerConfig instance.
      settings_config: What to return when settings.cfg is requested. Defaults
        to an empty config_pb2.SettingsCfg instance.
      metadata_file: What to return when metadata_file is requested. Defaults
        to an empty string.
    """
    def get_self_config(path, *_args, **_kwargs):
      self.assertIn(path, ('templates.cfg', 'managers.cfg', 'settings.cfg',
          'metadata_file'))
      if path == 'templates.cfg':
        proto = template_config or config_pb2.InstanceTemplateConfig()
      elif path == 'managers.cfg':
        proto = manager_config or config_pb2.InstanceGroupManagerConfig()
      elif path == 'settings.cfg':
        proto = settings_config or config_pb2.SettingsCfg()
      elif path == 'metadata_file':
        proto = metadata_file or ''
      return revision or 'mock-revision', proto
    self.mock(config.config, 'get_self_config', get_self_config)

  def test_empty_configs(self):
    """Ensures empty configs are successfully stored."""
    self.install_mock()

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.assertEqual(config.Configuration.cached().revision, 'mock-revision')

  def test_repeated_base_names(self):
    """Ensures duplicate base names reject the entire config."""
    template_config = config_pb2.InstanceTemplateConfig(
        templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-1',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-2',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-3',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-4',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-2',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-3',
            ),
        ],
    )
    self.install_mock(template_config=template_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.failIf(config.Configuration.cached().revision)

  def test_invalid_disk_type(self):
    """Ensures invalid disk types reject the entire config."""
    template_config = config_pb2.InstanceTemplateConfig(
        templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name',
                disk_type='invalid-disk-type',
            ),
        ],
    )
    self.install_mock(template_config=template_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.failIf(config.Configuration.cached().revision)

  def test_repeated_zone_different_base_name(self):
    """Ensures repeated zones in different base names are valid."""
    manager_config = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name-1',
                zone='us-central1-a',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name-2',
                zone='us-central1-a',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name-3',
                zone='us-central1-a',
            ),
        ],
    )
    self.install_mock(manager_config=manager_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failUnless(config.Configuration.cached().manager_config)
    self.assertEqual(config.Configuration.cached().revision, 'mock-revision')

  def test_repeated_zone_same_base_name(self):
    """Ensures repeated zones in a base name reject the entire config."""
    manager_config = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name-1',
                zone='us-central1-a',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name-2',
                zone='us-central1-b',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name-1',
                zone='us-central1-a',
            ),
        ],
    )
    self.install_mock(manager_config=manager_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.failIf(config.Configuration.cached().revision)

  def test_minimum_size_exceeds_maximum_size(self):
    """Ensures repeated zones in a base name reject the entire config."""
    manager_config = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                maximum_size=1,
                minimum_size=2,
                template_base_name='base-name-1',
                zone='us-central1-a',
            ),
        ],
    )
    self.install_mock(manager_config=manager_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.failIf(config.Configuration.cached().revision)

  def test_maximum_size_exceeds_maximum_allowed(self):
    """Ensures repeated zones in a base name reject the entire config."""
    manager_config = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                maximum_size=9999,
                template_base_name='base-name-1',
                zone='us-central1-a',
            ),
        ],
    )
    self.install_mock(manager_config=manager_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.failIf(config.Configuration.cached().revision)

  def test_update_template_configs(self):
    """Ensures config is updated when revision changes."""
    manager_config = config_pb2.InstanceGroupManagerConfig(
        managers=[config_pb2.InstanceGroupManagerConfig.InstanceGroupManager()],
    )
    self.install_mock(revision='revision-1', manager_config=manager_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failUnless(config.Configuration.cached().manager_config)
    self.assertEqual(config.Configuration.cached().revision, 'revision-1')

    template_config = config_pb2.InstanceTemplateConfig(
        templates=[config_pb2.InstanceTemplateConfig.InstanceTemplate()],
    )
    self.install_mock(revision='revision-2', template_config=template_config)

    config.update_template_configs()
    self.failUnless(config.Configuration.cached().template_config)
    self.failIf(config.Configuration.cached().manager_config)
    self.assertEqual(config.Configuration.cached().revision, 'revision-2')

  def test_update_template_configs_same_revision(self):
    """Ensures config is not updated when revision doesn't change."""
    manager_config = config_pb2.InstanceGroupManagerConfig(
        managers=[config_pb2.InstanceGroupManagerConfig.InstanceGroupManager()],
    )
    self.install_mock(manager_config=manager_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failUnless(config.Configuration.cached().manager_config)
    self.assertEqual(config.Configuration.cached().revision, 'mock-revision')

    template_config = config_pb2.InstanceTemplateConfig(
        templates=[config_pb2.InstanceTemplateConfig.InstanceTemplate()],
    )
    self.install_mock(template_config=template_config)

    config.update_template_configs()
    self.failIf(config.Configuration.cached().template_config)
    self.failUnless(config.Configuration.cached().manager_config)
    self.assertEqual(config.Configuration.cached().revision, 'mock-revision')

  def test_settings_valid_mp_config(self):
    """Ensures base MP settings are correctly received if configured."""
    settings_config = config_pb2.SettingsCfg(mp_server='server')
    self.install_mock(settings_config=settings_config)

    self.assertEqual(config._get_settings()[1].mp_server, 'server')
    self.assertEqual(machine_provider.MachineProviderConfiguration.instance_url,
        'server')

  def test_settings_empty_mp_config(self):
    """Ensures base MP settings are correctly received if not configured."""
    settings_config = config_pb2.SettingsCfg()
    self.install_mock(settings_config=settings_config)

    mpdefault = machine_provider.MachineProviderConfiguration.get_instance_url()
    self.failIf(config._get_settings()[1].mp_server)
    self.assertEqual(machine_provider.MachineProviderConfiguration.instance_url,
        mpdefault)

  def test_validate_settings(self):
    self.validator_test(config.validate_settings_config,
        config_pb2.SettingsCfg(), [])
    self.validator_test(
        config.validate_settings_config,
        config_pb2.SettingsCfg(mp_server='http://url'),
        ['mp_server must start with "https://" or "http://localhost"'])
    self.validator_test(
        config.validate_settings_config,
        config_pb2.SettingsCfg(mp_server='url'),
        ['mp_server must start with "https://" or "http://localhost"'])

  def test_validate_metadata(self):
    self.validator_test(config.validate_template_config,
        config_pb2.InstanceTemplateConfig(), [])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='name',
            ),
        ]),
        [])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='name',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='name',
            ),
        ]),
        ['base_name name is not globally unique.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                metadata=['key'],
            ),
        ]),
        ['metadata key is not in key:value form.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                metadata=[':'],
            ),
        ]),
        ['metadata : has empty key.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                metadata_from_file=['key'],
            ),
        ]),
        ['metadata_from_file key is not in key:value form.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                metadata_from_file=['key:'],
            ),
        ]),
        ['metadata_from_file key: has empty value.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                metadata_from_file=[':value'],
            ),
        ]),
        ['metadata_from_file :value has empty key.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                snapshot_labels=['key'],
            ),
        ]),
        ['snapshot_label key is not in key:value form.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                snapshot_labels=[':'],
            ),
        ]),
        ['snapshot_label : has empty key.'])
    self.validator_test(
        config.validate_template_config,
        config_pb2.InstanceTemplateConfig(templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                metadata=['key:value', 'key:value:with:colons', 'key:'],
                metadata_from_file=['key:value', 'key:value:with:colons'],
                snapshot_labels=['key:value', 'key:value:with:colons', 'key:'],
            ),
        ]),
        [])

  def test_metadata_from_file(self):
    key = "startup-script"
    content = "./cmd.sh"
    template_config = config_pb2.InstanceTemplateConfig(
      templates=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate(
                base_name='base-name-1',
                metadata_from_file=[
                    '%s:metadata_file' % key,
                ],
            ),
      ],
    )
    self.install_mock(template_config=template_config, metadata_file=content)
    config.update_template_configs()
    self.failUnless(config.Configuration.cached().template_config)
    self.assertEqual(len(template_config.templates[0].metadata), 0)
    cfg, _ = config.Configuration.load()
    self.assertEqual(len(cfg.templates), 1)
    self.assertEqual(len(cfg.templates[0].metadata), 1)
    self.assertEqual(cfg.templates[0].metadata[0], '%s:%s' % (key, content))

if __name__ == '__main__':
  unittest.main()
