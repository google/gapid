# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for reading GCE Backend configuration."""

import collections
import logging

from google import protobuf
from google.appengine.ext import ndb

import metrics

from components import config
from components import datastore_utils
from components import gce
from components import machine_provider
from components import utils
from components.config import validation

from proto import config_pb2


TEMPLATES_CFG_FILENAME = 'templates.cfg'
MANAGERS_CFG_FILENAME = 'managers.cfg'
SETTINGS_CFG_FILENAME = 'settings.cfg'


class Configuration(datastore_utils.config.GlobalConfig):
  """Configuration for this service."""
  # Text-formatted proto.config_pb2.InstanceTemplateConfig.
  template_config = ndb.TextProperty(default='')
  # Text-formatted proto.config_pb2.InstanceGroupManagerConfig.
  manager_config = ndb.TextProperty(default='')
  # Revision of the configs.
  revision = ndb.StringProperty(default='')

  @classmethod
  def load(cls):
    """Loads text-formatted template and manager configs into message.Messages.

    Returns:
      A 2-tuple of (InstanceTemplateConfig, InstanceGroupManagerConfig).
    """
    def _adds_cfg_to_message(name, text_cfg, proto_cfg):
      try:
        protobuf.text_format.Merge(text_cfg, proto_cfg)
      except protobuf.text_format.ParseError as ex:
        logging.error('Invalid %s: %s', name, ex)
        raise ValueError(ex)
      return proto_cfg

    def _desugar_template(proto_cfg):
      for template in proto_cfg.templates:
        for value in template.metadata_from_file:
          # Validated below. See validate_template_config.
          part = value.split(':', 1)
          _, content = config.get_self_config(part[1], None,
                  store_last_good=True)
          template.metadata.append('%s:%s' % (part[0], content))
        del template.metadata_from_file[:]
      return proto_cfg

    configuration = cls.cached()
    template_cfg = _adds_cfg_to_message(
        'template.cfg', configuration.template_config,
        config_pb2.InstanceTemplateConfig()
    )
    template_cfg = _desugar_template(template_cfg)
    manager_cfg = _adds_cfg_to_message(
        'manager.cfg', configuration.manager_config,
        config_pb2.InstanceGroupManagerConfig()
    )
    return template_cfg, manager_cfg


def count_instances():
  """Counts the numbers of instances configured by each instance template.

  Returns:
    A dict mapping instance template name to a list of (min, max).
  """
  # Aggregate the numbers of instances configured in each instance group manager
  # created for each instance template.
  totals = collections.defaultdict(lambda: [0, 0])
  _, manager_cfg = Configuration.load()
  for manager in manager_cfg.managers:
    totals[manager.template_base_name][0] += manager.minimum_size
    totals[manager.template_base_name][1] += manager.maximum_size
  return totals


def update_template_configs():
  """Updates the local template configuration from the config service.

  Ensures that all config files are at the same path revision.
  """
  template_revision, template_config = config.get_self_config(
      TEMPLATES_CFG_FILENAME, config_pb2.InstanceTemplateConfig,
      store_last_good=True)
  manager_revision, manager_config = config.get_self_config(
      MANAGERS_CFG_FILENAME, config_pb2.InstanceGroupManagerConfig,
      store_last_good=True)

  if template_revision != manager_revision:
    logging.error('Not updating configuration due to revision mismatch.')
    return

  stored_config = Configuration.cached()

  if stored_config.revision == template_revision:
    # Config is up-to-date so just report validity.
    # The stored_config will always be valid.
    metrics.config_valid.set(True, fields={'config': TEMPLATES_CFG_FILENAME})
    metrics.config_valid.set(True, fields={'config': MANAGERS_CFG_FILENAME})
    return

  errors = False

  context = config.validation_context.Context.logging()
  if template_config:
    validate_template_config(template_config, context)
  if context.result().has_errors:
    logging.warning(
        'Not updating configuration due to errors in templates.cfg')
    errors = True

  context = config.validation_context.Context.logging()
  if manager_config:
    validate_manager_config(manager_config, context)
  if context.result().has_errors:
    logging.error('Not updating configuration due to errors in managers.cfg')
    errors = True

  if errors:
    return

  logging.info('Updating configuration to %s', template_revision)
  stored_config.modify(
      updated_by='',  # this is called from cron, there's no user here
      manager_config=protobuf.text_format.MessageToString(manager_config),
      revision=template_revision,
      template_config=protobuf.text_format.MessageToString(template_config),
  )


@validation.self_rule(TEMPLATES_CFG_FILENAME, config_pb2.InstanceTemplateConfig)
def validate_template_config(cfg, context):
  """Validates an InstanceTemplateConfig instance."""
  # We don't do any GCE-specific validation here. Just require globally
  # unique base name because base name is used as the key in the datastore.
  base_names = set()
  valid = True
  for template in cfg.templates:
    if template.base_name in base_names:
      context.error('base_name %s is not globally unique.', template.base_name)
      valid = False
    else:
      base_names.add(template.base_name)
    if template.disk_type and template.disk_type not in gce.DISK_TYPES:
      context.error('disk_type %s is invalid.', template.disk_type)
      valid = False
    for metadata in template.metadata:
      parts = metadata.split(':', 1)
      if len(parts) != 2:
        context.error('metadata %s is not in key:value form.', metadata)
        valid = False
      elif not parts[0]:
        context.error('metadata %s has empty key.', metadata)
        valid = False
    for metadata in template.metadata_from_file:
      parts = metadata.split(':', 1)
      if len(parts) != 2:
        context.error(
            'metadata_from_file %s is not in key:value form.', metadata)
        valid = False
      else:
        if not parts[0]:
          context.error('metadata_from_file %s has empty key.', metadata)
          valid = False
        if not parts[1]:
          context.error('metadata_from_file %s has empty value.', metadata)
          valid = False
    for label in template.snapshot_labels:
      parts = label.split(':', 1)
      if len(parts) != 2:
        context.error('snapshot_label %s is not in key:value form.', label)
        valid = False
      elif not parts[0]:
        context.error('snapshot_label %s has empty key.', label)
        valid = False
  metrics.config_valid.set(valid, fields={'config': TEMPLATES_CFG_FILENAME})


@validation.self_rule(
    MANAGERS_CFG_FILENAME, config_pb2.InstanceGroupManagerConfig)
def validate_manager_config(cfg, context):
  """Validates an InstanceGroupManagerConfig instance."""
  # We don't do any GCE-specific validation here. Just require per-template
  # unique zone because template+zone is used as a key in the datastore.
  zones = collections.defaultdict(set)
  valid = True
  for manager in cfg.managers:
    if manager.zone in zones[manager.template_base_name]:
      context.error(
          'zone %s is not unique in template %s.',
          manager.zone,
          manager.template_base_name,
      )
      valid = False
    else:
      zones[manager.template_base_name].add(manager.zone)
    if manager.minimum_size > manager.maximum_size:
      context.error(
          'minimum_size > maximum_size for zone %s in template %s.',
          manager.zone,
          manager.template_base_name,
      )
      valid = False
    if manager.maximum_size > 1000:
      # A GCE InstanceGroup is limited to 1000 instances.
      context.error(
          'maximum_size > 1000 for zone %s in template %s.',
          manager.zone,
          manager.template_base_name,
      )
      valid = False
  metrics.config_valid.set(valid, fields={'config': MANAGERS_CFG_FILENAME})


@validation.self_rule(SETTINGS_CFG_FILENAME, config_pb2.SettingsCfg)
def validate_settings_config(cfg, context):
  if cfg.HasField('mp_server'):
    if not validation.is_valid_secure_url(cfg.mp_server):
      context.error(
          'mp_server must start with "https://" or "http://localhost"')


def _get_settings():
  """Returns (rev, cfg) where cfg is a parsed SettingsCfg message.

  The config is cached in the datastore.
  """
  rev, cfg = config.get_self_config(
      SETTINGS_CFG_FILENAME, config_pb2.SettingsCfg, store_last_good=True)
  cfg = cfg or config_pb2.SettingsCfg()
  if cfg.mp_server:
    current_config = machine_provider.MachineProviderConfiguration.cached()
    if cfg.mp_server != current_config.instance_url:
      logging.info('Updating Machine Provider server to %s', cfg.mp_server)
      current_config.modify(updated_by='', instance_url=cfg.mp_server)
  return rev, cfg


@utils.cache_with_expiration(60)
def settings():
  """Loads settings from an NDB-based cache or a default one if not present."""
  return _get_settings()[1]
