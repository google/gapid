# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for parsing GCE Backend configuration."""

import collections
import hashlib
import json
import logging

from google.appengine.ext import ndb
from protorpc.remote import protojson

from components.machine_provider import dimensions

import models
import utilities


def get_instance_template_key(template_cfg):
  """Returns a key for an InstanceTemplate for the given config.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.

  Returns:
    ndb.Key for a models.InstanceTemplate entity.
  """
  return ndb.Key(models.InstanceTemplate, template_cfg.base_name)


def get_instance_template_revision_key(template_cfg):
  """Returns a key for an InstanceTemplateRevision for the given config.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.

  Returns:
    ndb.Key for a models.InstanceTemplateRevision entity.
  """
  return ndb.Key(
      models.InstanceTemplateRevision,
      compute_template_checksum(template_cfg),
      parent=get_instance_template_key(template_cfg),
  )


def get_instance_group_manager_key(template_cfg, manager_cfg):
  """Returns a key for an InstanceGroupManager for the given config.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.
    manager_cfg:
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManager which has
      template_base_name equal to template_cfg.base_name.

  Returns:
    ndb.Key for a models.InstanceGroupManager entity.
  """
  assert manager_cfg.template_base_name == template_cfg.base_name
  return ndb.Key(
      models.InstanceGroupManager,
      manager_cfg.zone,
      parent=get_instance_template_revision_key(template_cfg),
  )


def _load_dict(pairs):
  """Loads from the given dict-like property.

  Args:
    pairs: A list of "<key>:<value>" strings.

  Returns:
    A dict.
  """
  return dict(pair.split(':', 1) for pair in pairs)


def _load_machine_provider_dimensions(pairs):
  """Loads from the given dict-like property.

  Args:
    pairs: A list of "<key>:<value>" strings.

  Returns:
    dimensions.Dimensions instance.
  """
  return protojson.decode_message(
      dimensions.Dimensions, json.dumps(_load_dict(pairs)))


def _load_service_accounts(service_accounts):
  """Loads from the given service accounts.

  Args:
    service_accounts:
      proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount.

  Returns:
    models.ServiceAccount instance.
  """
  return [models.ServiceAccount(name=sa.name, scopes=list(sa.scopes))
          for sa in service_accounts]


def compute_template_checksum(template_cfg):
  """Computes a checksum from the given config.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.

  Returns:
    The checksum string.
  """
  identifying_properties = {
      'auto-assign-external-ip': template_cfg.auto_assign_external_ip,
      'dimensions': _load_dict(template_cfg.dimensions),
      'disk-size-gb': template_cfg.disk_size_gb,
      'disk-type': template_cfg.disk_type,
      'image-name': template_cfg.image_name,
      'image-project': template_cfg.image_project,
      'machine-type': template_cfg.machine_type,
      'metadata': _load_dict(template_cfg.metadata),
      'min-cpu-platform': template_cfg.min_cpu_platform,
      'network_url': template_cfg.network_url,
      'project': template_cfg.project,
      'service-accounts': [],
      'snapshot_labels': sorted(template_cfg.snapshot_labels),
      'snapshot_name': template_cfg.snapshot_name,
      'tags': sorted(template_cfg.tags),
  }
  if template_cfg.service_accounts:
    # Changing the first service account has special meaning because the
    # first service account is the one we communicate to Machine Provider.
    identifying_properties['service-accounts'].append({
        'name': template_cfg.service_accounts[0].name,
        'scopes': sorted(template_cfg.service_accounts[0].scopes),
    })
    # The rest of the service accounts have no special meaning, so changing
    # their order shouldn't affect the checksum.
    identifying_properties['service-accounts'].extend([
        {
          'name': i.name,
          'scopes': sorted(i.scopes),
        }
        for i in sorted(template_cfg.service_accounts[1:], key=lambda i: i.name)
    ])
  return utilities.compute_checksum(identifying_properties)


def ensure_instance_group_manager_matches(manager_cfg, instance_group_manager):
  """Ensures the InstanceGroupManager matches the config.

  Args:
    manager_cfg:
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManagers.
    instance_group_manager: models.InstanceGroupManager.

  Returns:
    Whether or not the given InstanceGroupManager was modified.
  """
  modified = False

  if instance_group_manager.minimum_size != manager_cfg.minimum_size:
    logging.info(
        'Updating minimum size (%s -> %s): %s',
        instance_group_manager.minimum_size,
        manager_cfg.minimum_size,
        instance_group_manager.key,
    )
    instance_group_manager.minimum_size = manager_cfg.minimum_size
    modified = True

  if instance_group_manager.maximum_size != manager_cfg.maximum_size:
    logging.info(
        'Updating maximum size (%s -> %s): %s',
        instance_group_manager.maximum_size,
        manager_cfg.maximum_size,
        instance_group_manager.key,
    )
    instance_group_manager.maximum_size = manager_cfg.maximum_size
    modified = True

  return modified


def ensure_instance_group_managers_active(
    template_cfg, manager_cfgs, instance_template_revision):
  """Ensures the configured InstanceGroupManagers are active.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.
    manager_cfgs: List of
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManagers which
      all have template_base_name equal to template_cfg.base_name.
    instance_template_revision: models.InstanceTemplateRevision.

  Returns:
    Whether or not the given InstanceTemplateRevision was modified.
  """
  active = []
  drained = []
  modified = False
  managers = {manager_cfg.zone: manager_cfg for manager_cfg in manager_cfgs}

  for instance_group_manager_key in instance_template_revision.active:
    if instance_group_manager_key.id() not in managers:
      logging.info(
          'Draining InstanceGroupManager: %s', instance_group_manager_key)
      drained.append(instance_group_manager_key)
      modified = True
    else:
      active.append(instance_group_manager_key)
      managers.pop(instance_group_manager_key.id())

  for instance_group_manager_key in instance_template_revision.drained:
    if instance_group_manager_key.id() in managers:
      logging.info(
          'Reactivating InstanceGroupManager: %s', instance_group_manager_key)
      active.append(instance_group_manager_key)
      managers.pop(instance_group_manager_key.id())
      modified = True
    else:
      drained.append(instance_group_manager_key)

  for zone in managers:
    instance_group_manager_key = get_instance_group_manager_key(
        template_cfg, managers[zone])
    logging.info(
        'Activating InstanceGroupManager: %s', instance_group_manager_key)
    active.append(instance_group_manager_key)
    modified = True

  instance_template_revision.active = active
  instance_template_revision.drained = drained
  return modified


def ensure_instance_template_revision_active(template_cfg, instance_template):
  """Ensures the configured InstanceTemplateRevision is active.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.
    instance_template: models.InstanceTemplate.

  Returns:
    Whether or not the given InstanceTemplate was modified.
  """
  checksum = compute_template_checksum(template_cfg)
  if instance_template.active:
    if instance_template.active.id() == checksum:
      # Correct InstanceTemplateRevision is already active.
      return False

    logging.info(
        'Draining InstanceTemplateRevision: %s', instance_template.active)
    instance_template.drained.append(instance_template.active)
    instance_template.active = None

  for i, instance_template_revision_key in enumerate(instance_template.drained):
    if instance_template_revision_key.id() == checksum:
      logging.info(
          'Reactivating InstanceTemplateRevision: %s',
          instance_template_revision_key,
      )
      instance_template.active = instance_template.drained.pop(i)
      return True

  instance_template_revision_key = get_instance_template_revision_key(
      template_cfg)
  logging.info(
      'Activating InstanceTemplateRevision: %s', instance_template_revision_key)
  instance_template.active = instance_template_revision_key
  return True


@ndb.transactional_tasklet
def ensure_instance_template_revision_drained(instance_template_key):
  """Ensures any active InstanceTemplateRevision is drained.

  Args:
    instance_template_key: ndb.Key for a models.InstanceTemplateRevision.
  """
  instance_template = instance_template_key.get()
  if not instance_template:
    logging.warning(
        'InstanceTemplate does not exist: %s', instance_template_key)
    return

  if not instance_template.active:
    return

  logging.info(
      'Draining InstanceTemplateRevision: %s', instance_template.active)
  instance_template.drained.append(instance_template.active)
  instance_template.active = None
  instance_template.put()


@ndb.transactional_tasklet
def ensure_instance_group_manager_exists(template_cfg, manager_cfg):
  """Ensures an InstanceGroupManager exists for the given config.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.
    manager_cfg:
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManager which
      has template_base_name equal to template_cfg.base_name.

  Returns:
    ndb.Key for a models.InstanceGroupManager entity.
  """
  instance_group_manager_key = get_instance_group_manager_key(
      template_cfg, manager_cfg)
  instance_group_manager = yield instance_group_manager_key.get_async()

  if not instance_group_manager:
    logging.info(
        'Creating InstanceGroupManager: %s', instance_group_manager_key)
    instance_group_manager = models.InstanceGroupManager(
        key=instance_group_manager_key)

  if ensure_instance_group_manager_matches(manager_cfg, instance_group_manager):
    instance_group_manager_key = yield instance_group_manager.put_async()

  raise ndb.Return(instance_group_manager_key)


@ndb.transactional_tasklet
def ensure_entities_exist(template_cfg, manager_cfgs, max_concurrent=50):
  """Ensures entities exist for the given config.

  Ensures the existence of the root InstanceTemplate, the active
  InstanceTemplateRevision, and the active InstanceGroupManagers.

  Args:
    template_cfg: proto.config_pb2.InstanceTemplateConfig.InstanceTemplate.
    manager_cfgs: List of
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManagers which
      all have template_base_name equal to template_cfg.base_name.
    max_concurrent: Maximum number to create concurrently.

  Returns:
    ndb.Key for the root models.InstanceTemplate entity.
  """
  instance_template_key = get_instance_template_key(template_cfg)
  instance_template_revision_key = get_instance_template_revision_key(
      template_cfg)
  instance_template, instance_template_revision = yield ndb.get_multi_async([
      instance_template_key, instance_template_revision_key])

  modified = []

  # Ensure InstanceTemplate exists and has the correct active
  # InstanceTemplateRevision.
  put_it = False
  if not instance_template:
    logging.info('Creating InstanceTemplate: %s', instance_template_key)
    instance_template = models.InstanceTemplate(key=instance_template_key)
    put_it = True
  if ensure_instance_template_revision_active(template_cfg, instance_template):
    put_it = True
  if put_it:
    modified.append(instance_template)

  # Ensure InstanceTemplateRevision exists and has the correct active
  # InstanceGroupManagers.
  put_itr = False
  if not instance_template_revision:
    logging.info(
        'Creating InstanceTemplateRevision: %s', instance_template_revision_key)
    instance_template_revision = models.InstanceTemplateRevision(
        key=instance_template_revision_key,
        dimensions=_load_machine_provider_dimensions(template_cfg.dimensions),
        disk_size_gb=template_cfg.disk_size_gb,
        disk_type=template_cfg.disk_type,
        auto_assign_external_ip=template_cfg.auto_assign_external_ip,
        image_name=template_cfg.image_name,
        image_project=template_cfg.image_project,
        machine_type=template_cfg.machine_type,
        metadata=_load_dict(template_cfg.metadata),
        min_cpu_platform=template_cfg.min_cpu_platform,
        network_url=template_cfg.network_url,
        project=template_cfg.project,
        service_accounts=_load_service_accounts(template_cfg.service_accounts),
        snapshot_labels=list(template_cfg.snapshot_labels),
        snapshot_name=template_cfg.snapshot_name,
        tags=list(template_cfg.tags),
    )
    put_itr = True
  if ensure_instance_group_managers_active(
      template_cfg, manager_cfgs, instance_template_revision):
    put_itr = True
  if put_itr:
    modified.append(instance_template_revision)

  # Ensure InstanceGroupManagers exist and have correct minimum/maximum sizes.
  while manager_cfgs:
    yield [ensure_instance_group_manager_exists(template_cfg, manager_cfg)
           for manager_cfg in manager_cfgs[:max_concurrent]]
    manager_cfgs = manager_cfgs[max_concurrent:]

  if modified:
    yield ndb.put_multi_async(modified)
  raise ndb.Return(instance_template_key)


def parse(template_cfgs, manager_cfgs, max_concurrent=50, max_concurrent_igm=5):
  """Ensures entities exist for and match the given config.

  Ensures the existence of the root InstanceTemplate, the active
  InstanceTemplateRevision, and the active InstanceGroupManagers.

  Args:
    template_cfgs: List of
      proto.config_pb2.InstanceTemplateConfig.InstanceTemplates.
    manager_cfgs: List of
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManagers.
    max_concurrent: Maximum number to create concurrently.
    max_concurrent_igm: Maximum number of InstanceGroupManagers to create
      concurrently for each InstanceTemplate/InstanceTemplateRevision. The
      actual maximum number of concurrent entities being created will be
      max_concurrent * max_concurrent_igm.
  """
  manager_cfg_map = collections.defaultdict(list)
  for manager_cfg in manager_cfgs:
    manager_cfg_map[manager_cfg.template_base_name].append(manager_cfg)

  def f(template_cfg):
    return ensure_entities_exist(
        template_cfg,
        manager_cfg_map.get(template_cfg.base_name, []),
        max_concurrent=max_concurrent_igm,
    )

  utilities.batch_process_async(template_cfgs, f, max_concurrent=max_concurrent)

  # Now go over every InstanceTemplate not mentioned anymore in the config and
  # mark its active InstanceTemplateRevision as drained.
  template_names = set(template_cfg.base_name for template_cfg in template_cfgs)
  instance_template_keys = []
  for instance_template in models.InstanceTemplate.query().fetch():
    if instance_template.key.id() not in template_names:
      if instance_template.active:
        instance_template_keys.append(instance_template.key)

  utilities.batch_process_async(
      instance_template_keys,
      ensure_instance_template_revision_drained,
      max_concurrent=max_concurrent,
  )
