#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for parse.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import datastore_utils
from test_support import test_case

import instance_templates
import models
import parse
from proto import config_pb2


class ComputeTemplateChecksumTest(test_case.TestCase):
  """Tests for parse.compute_template_checksum."""

  def test_empty_template(self):
    """Ensures empty template checksum is computable."""
    template = config_pb2.InstanceTemplateConfig.InstanceTemplate()
    self.failUnless(parse.compute_template_checksum(template))

  def test_checksum_is_order_independent(self):
    """Ensures checksum is independent of the order of repeated field values."""
    template1 = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        dimensions=[
            'key1:value1',
            'key2:value2',
        ],
        disk_size_gb=300,
        disk_type='pd-ssd',
        machine_type='n1-standard-8',
        metadata=[
            'key1:value1',
            'key2:value2',
        ],
        tags=[
            'tag1',
            'tag2',
        ],
    )
    template2 = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        dimensions=[
            'key2:value2',
            'key1:value1',
        ],
        disk_size_gb=300,
        disk_type='pd-ssd',
        machine_type='n1-standard-8',
        metadata=[
            'key2:value2',
            'key1:value1',
        ],
        tags=[
            'tag2',
            'tag1',
        ],
    )

    self.assertEqual(
        parse.compute_template_checksum(template1),
        parse.compute_template_checksum(template2),
    )

  def test_checksum_is_first_service_account_dependent(self):
    """Ensures checksum is dependent on the first service account."""
    template1 = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        service_accounts=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-1',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-2',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-3',
            ),
        ],
    )
    template2 = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        service_accounts=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-3',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-2',
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-1',
            ),
        ],
    )

    self.assertNotEqual(
        parse.compute_template_checksum(template1),
        parse.compute_template_checksum(template2),
    )

  def test_checksum_is_only_first_service_account_dependent(self):
    """Ensures checksum is only dependent on the first service account."""
    template1 = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        service_accounts=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-1',
                scopes=[
                    'scope1',
                    'scope2',
                ],
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-2',
                scopes=[
                    'scope1',
                    'scope2',
                ],
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-3',
                scopes=[
                    'scope1',
                    'scope2',
                ],
            ),
        ],
    )
    template2 = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        service_accounts=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-1',
                scopes=[
                    'scope2',
                    'scope1',
                ],
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-3',
                scopes=[
                    'scope2',
                    'scope1',
                ],
            ),
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account-2',
                scopes=[
                    'scope2',
                    'scope1',
                ],
            ),
        ],
    )

    self.assertEqual(
        parse.compute_template_checksum(template1),
        parse.compute_template_checksum(template2),
    )


class EnsureInstanceGroupManagerMatches(test_case.TestCase):
  """Tests for parse.ensure_instance_group_manager_matches."""

  def test_already_matches(self):
    """Ensures that nothing changes when instance group manager matches."""
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=3,
        minimum_size=2,
        template_base_name='base-name',
        zone='zone',
    )
    instance_group_manager = models.InstanceGroupManager(
        maximum_size=manager_cfg.maximum_size,
        minimum_size=manager_cfg.minimum_size,
    )

    self.failIf(parse.ensure_instance_group_manager_matches(
        manager_cfg, instance_group_manager))
    self.assertEqual(
        instance_group_manager.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(
        instance_group_manager.minimum_size, manager_cfg.minimum_size)

  def test_max_matches(self):
    """Ensures that maximum_size is made to match."""
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=3,
        minimum_size=2,
        template_base_name='base-name',
        zone='zone',
    )
    instance_group_manager = models.InstanceGroupManager(
        maximum_size=manager_cfg.maximum_size + 1,
        minimum_size=manager_cfg.minimum_size,
    )

    self.failUnless(parse.ensure_instance_group_manager_matches(
        manager_cfg, instance_group_manager))
    self.assertEqual(
        instance_group_manager.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(
        instance_group_manager.minimum_size, manager_cfg.minimum_size)

  def test_min_matches(self):
    """Ensures that minimum_size is made to match."""
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=3,
        minimum_size=2,
        template_base_name='base-name',
        zone='zone',
    )
    instance_group_manager = models.InstanceGroupManager(
        maximum_size=manager_cfg.maximum_size,
        minimum_size=manager_cfg.minimum_size - 1,
    )

    self.failUnless(parse.ensure_instance_group_manager_matches(
        manager_cfg, instance_group_manager))
    self.assertEqual(
        instance_group_manager.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(
        instance_group_manager.minimum_size, manager_cfg.minimum_size)

  def test_matches(self):
    """Ensures that maximum_size and minimum_size are both made to match."""
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=3,
        minimum_size=2,
        template_base_name='base-name',
        zone='zone',
    )
    instance_group_manager = models.InstanceGroupManager(
        maximum_size=manager_cfg.maximum_size + 1,
        minimum_size=manager_cfg.minimum_size - 1,
    )

    self.failUnless(parse.ensure_instance_group_manager_matches(
        manager_cfg, instance_group_manager))
    self.assertEqual(
        instance_group_manager.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(
        instance_group_manager.minimum_size, manager_cfg.minimum_size)


class EnsureInstanceGroupManagersActiveTest(test_case.TestCase):
  """Tests for parse.ensure_group_managers_revision_active."""

  def test_activates(self):
    """Ensures that the instance group managers are activated."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfgs = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone1',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone2',
            ),
        ],
    ).managers
    expected_active_keys = [
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[0]),
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[1]),
    ]
    instance_template_revision = models.InstanceTemplateRevision(
        active=[
            parse.get_instance_group_manager_key(template_cfg, manager_cfgs[1]),
        ],
    )

    self.failUnless(parse.ensure_instance_group_managers_active(
        template_cfg, manager_cfgs, instance_template_revision))
    self.assertItemsEqual(
        instance_template_revision.active, expected_active_keys)

  def test_drains_and_activates(self):
    """Ensures that the active instance group managers are drained."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfgs = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone1',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone2',
            ),
        ],
    ).managers
    expected_active_keys = [
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[0]),
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[1]),
    ]
    expected_drained_keys = [
        ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
        ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
    ]
    instance_template_revision = models.InstanceTemplateRevision(
        active=[
            ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
        ],
        drained=[
            ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
        ],
    )

    self.failUnless(parse.ensure_instance_group_managers_active(
        template_cfg, manager_cfgs, instance_template_revision))
    self.assertItemsEqual(
        instance_template_revision.active, expected_active_keys)
    self.assertItemsEqual(
        instance_template_revision.drained, expected_drained_keys)

  def test_reactivates(self):
    """Ensures that the drained instance group managers are reactivated."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfgs = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone1',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone2',
            ),
        ],
    ).managers
    expected_active_keys = [
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[0]),
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[1]),
    ]
    instance_template_revision = models.InstanceTemplateRevision(
        drained=expected_active_keys,
    )

    self.failUnless(parse.ensure_instance_group_managers_active(
        template_cfg, manager_cfgs, instance_template_revision))
    self.assertItemsEqual(
        instance_template_revision.active, expected_active_keys)
    self.failIf(instance_template_revision.drained)

  def test_drains_and_reactivates(self):
    """Ensures that the active are drained and the drained are reactivated."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfgs = config_pb2.InstanceGroupManagerConfig(
        managers=[
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone1',
            ),
            config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
                template_base_name='base-name',
                zone='zone2',
            ),
        ],
    ).managers
    expected_active_keys = [
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[0]),
        parse.get_instance_group_manager_key(template_cfg, manager_cfgs[1]),
    ]
    instance_template_revision = models.InstanceTemplateRevision(
        active=[
            ndb.Key(models.InstanceGroupManager, 'fake-key'),
        ],
        drained=expected_active_keys,
    )

    self.failUnless(parse.ensure_instance_group_managers_active(
        template_cfg, manager_cfgs, instance_template_revision))
    self.assertItemsEqual(
        instance_template_revision.active, expected_active_keys)
    self.assertEqual(instance_template_revision.drained[0].id(), 'fake-key')


class EnsureInstanceTemplateRevisionActiveTest(test_case.TestCase):
  """Tests for parse.ensure_instance_template_revision_active."""

  def test_activates(self):
    """Ensures that the instance template revision is activated."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    expected_active_key = parse.get_instance_template_revision_key(template_cfg)
    instance_template = models.InstanceTemplate()

    self.failUnless(parse.ensure_instance_template_revision_active(
        template_cfg, instance_template))
    self.assertEqual(instance_template.active, expected_active_key)

  def test_drains_and_activates(self):
    """Ensures that the active instance template revision is drained."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    expected_active_key = parse.get_instance_template_revision_key(template_cfg)
    instance_template = models.InstanceTemplate(
        active=ndb.Key(models.InstanceTemplateRevision, 'fake-key'),
    )

    self.failUnless(parse.ensure_instance_template_revision_active(
        template_cfg, instance_template))
    self.assertEqual(instance_template.active, expected_active_key)
    self.assertEqual(instance_template.drained[0].id(), 'fake-key')

  def test_reactivates(self):
    """Ensures that the drained instance template revision is reactivated."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    expected_active_key = parse.get_instance_template_revision_key(template_cfg)
    instance_template = models.InstanceTemplate(
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
            parse.get_instance_template_revision_key(template_cfg),
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
        ],
    )

    self.failUnless(parse.ensure_instance_template_revision_active(
        template_cfg, instance_template))
    self.assertEqual(instance_template.active, expected_active_key)
    self.assertEqual(len(instance_template.drained), 2)
    self.assertEqual(instance_template.drained[0].id(), 'fake-key-1')
    self.assertEqual(instance_template.drained[1].id(), 'fake-key-2')

  def test_drains_and_reactivates(self):
    """Ensures that the active is drained and the drained is reactivated."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    expected_active_key = parse.get_instance_template_revision_key(template_cfg)
    instance_template = models.InstanceTemplate(
        active=ndb.Key(models.InstanceTemplateRevision, 'fake-key'),
        drained=[
            parse.get_instance_template_revision_key(template_cfg),
        ],
    )

    self.failUnless(parse.ensure_instance_template_revision_active(
        template_cfg, instance_template))
    self.assertEqual(instance_template.active, expected_active_key)
    self.assertEqual(len(instance_template.drained), 1)
    self.assertEqual(instance_template.drained[0].id(), 'fake-key')

  def test_already_active(self):
    """Ensures that the active instance template revision remains active."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    expected_active_key = parse.get_instance_template_revision_key(template_cfg)
    instance_template = models.InstanceTemplate(
        active=parse.get_instance_template_revision_key(template_cfg),
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key'),
        ],
    )

    self.failIf(parse.ensure_instance_template_revision_active(
        template_cfg, instance_template))
    self.assertEqual(instance_template.active, expected_active_key)
    self.assertEqual(len(instance_template.drained), 1)
    self.assertEqual(instance_template.drained[0].id(), 'fake-key')


class EnsureInstanceTemplateRevisionDrainedTest(test_case.TestCase):
  """Tests for parse.ensure_instance_template_revision_drained."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the InstanceTemplate doesn't exist."""
    key = ndb.Key(models.InstanceTemplate, 'fake-key')
    parse.ensure_instance_template_revision_drained(key).wait()
    self.failIf(key.get())

  def test_nothing_active(self):
    """Ensures nothing happens when nothing is active."""
    key = models.InstanceTemplate(
        key=instance_templates.get_instance_template_key('base-name'),
    ).put()

    parse.ensure_instance_template_revision_drained(key).wait()
    self.failIf(key.get().active)
    self.failIf(key.get().drained)

  def test_already_drained(self):
    """Ensures nothing happens when the InstanceTemplateRevision is drained."""
    key = instance_templates.get_instance_template_revision_key(
        'base-name',
        'revision',
    )
    models.InstanceTemplate(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()
    expected_drained = [
        key,
    ]

    parse.ensure_instance_template_revision_drained(key.parent()).wait()
    self.failIf(key.parent().get().active)
    self.assertEqual(key.parent().get().drained, expected_drained)

  def test_drains(self):
    """Ensures active InstanceTemplateRevision is drained."""
    key = instance_templates.get_instance_template_revision_key(
        'base-name',
        'revision',
    )
    models.InstanceTemplate(
        key=key.parent(),
        active=key,
    ).put()
    expected_drained = [
        key,
    ]

    parse.ensure_instance_template_revision_drained(key.parent()).wait()
    self.failIf(key.parent().get().active)
    self.assertEqual(key.parent().get().drained, expected_drained)


class EnsureInstanceGroupManagerExistsTest(test_case.TestCase):
  """Tests for parse.ensure_instance_group_manager_exists."""

  def test_creates_new_entity(self):
    """Ensures that a new entity is created when one doesn't exist."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=2,
        minimum_size=1,
        template_base_name='base-name',
        zone='zone',
    )
    expected_key = parse.get_instance_group_manager_key(
        template_cfg, manager_cfg)

    key = parse.ensure_instance_group_manager_exists(
        template_cfg, manager_cfg).get_result()
    entity = key.get()

    self.assertEqual(key, expected_key)
    self.assertEqual(entity.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(entity.minimum_size, manager_cfg.minimum_size)

  def test_returns_existing_entity(self):
    """Ensures that an entity is returned when it already exists."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=2,
        minimum_size=1,
        template_base_name='base-name',
        zone='zone',
    )
    expected_key = parse.get_instance_group_manager_key(
        template_cfg, manager_cfg)
    models.InstanceGroupManager(
        key=expected_key,
        maximum_size=2,
        minimum_size=1,
    ).put()

    key = parse.ensure_instance_group_manager_exists(
        template_cfg, manager_cfg).get_result()
    entity = key.get()

    self.assertEqual(key, expected_key)
    self.assertEqual(entity.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(entity.minimum_size, manager_cfg.minimum_size)

  def test_matches_existing_entity(self):
    """Ensures that an entity matches when it already exists."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
    )
    manager_cfg = config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
        maximum_size=3,
        minimum_size=2,
        template_base_name='base-name',
        zone='zone',
    )
    expected_key = parse.get_instance_group_manager_key(
        template_cfg, manager_cfg)
    models.InstanceGroupManager(
        key=expected_key,
        maximum_size=2,
        minimum_size=1,
    ).put()

    key = parse.ensure_instance_group_manager_exists(
        template_cfg, manager_cfg).get_result()
    entity = key.get()

    self.assertEqual(key, expected_key)
    self.assertEqual(entity.maximum_size, manager_cfg.maximum_size)
    self.assertEqual(entity.minimum_size, manager_cfg.minimum_size)


class EnsureEntityExists(test_case.TestCase):
  """Tests for parse.ensure_instance_group_manager_exists."""

  def test_creates_new_entity(self):
    """Ensures that a new entity is created when one doesn't exist."""
    template_cfg = config_pb2.InstanceTemplateConfig.InstanceTemplate(
        base_name='base-name',
        dimensions=[
            'os_family:LINUX',
        ],
        disk_size_gb=100,
        disk_type="pd-ssd",
        machine_type='n1-standard-8',
        metadata=[
            'key:value',
        ],
        service_accounts=[
            config_pb2.InstanceTemplateConfig.InstanceTemplate.ServiceAccount(
                name='service-account',
                scopes=[
                  'scope',
                ],
            ),
        ],
        snapshot_labels=[
            'key:value',
        ],
        tags=[
          'tag',
        ],
    )
    manager_cfgs = [
        config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
            maximum_size=2,
            minimum_size=1,
            template_base_name='base-name',
            zone='us-central1-a',
        ),
        config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
            maximum_size=3,
            minimum_size=2,
            template_base_name='base-name',
            zone='us-central1-b',
        ),
        config_pb2.InstanceGroupManagerConfig.InstanceGroupManager(
            maximum_size=4,
            minimum_size=3,
            template_base_name='base-name',
            zone='us-central1-c',
        ),
    ]
    expected_instance_template_key = parse.get_instance_template_key(
        template_cfg)
    expected_instance_template_revision_key = (
        parse.get_instance_template_revision_key(template_cfg))
    expected_dimensions = parse._load_machine_provider_dimensions(
        template_cfg.dimensions)
    expected_metadata = parse._load_dict(template_cfg.metadata)
    expected_service_accounts = [
        models.ServiceAccount(
            name=template_cfg.service_accounts[0].name,
            scopes=list(template_cfg.service_accounts[0].scopes),
        ),
    ]
    expected_active_keys = [
        parse.get_instance_group_manager_key(template_cfg, manager_cfg)
        for manager_cfg in manager_cfgs
    ]

    future = parse.ensure_entities_exist(
        template_cfg, manager_cfgs)
    future.wait()
    instance_template_key = future.get_result()
    instance_template = instance_template_key.get()
    instance_template_revision = instance_template.active.get()
    instance_group_managers = sorted(
        [
            instance_group_manager.get()
            for instance_group_manager in instance_template_revision.active
        ],
        key=lambda instance_group_manager: instance_group_manager.key.id(),
    )

    self.assertEqual(instance_template_key, expected_instance_template_key)
    self.assertEqual(
        instance_template.active, expected_instance_template_revision_key)
    self.assertEqual(instance_template_revision.dimensions, expected_dimensions)
    self.assertEqual(
        instance_template_revision.disk_size_gb, template_cfg.disk_size_gb)
    self.assertEqual(
        instance_template_revision.disk_type, template_cfg.disk_type)
    self.assertEqual(
        instance_template_revision.machine_type, template_cfg.machine_type)
    self.assertEqual(instance_template_revision.metadata, expected_metadata)
    self.assertItemsEqual(
        instance_template_revision.service_accounts, expected_service_accounts)
    self.assertItemsEqual(instance_template_revision.snapshot_labels,
                          template_cfg.snapshot_labels)
    self.assertItemsEqual(instance_template_revision.tags, template_cfg.tags)
    self.assertItemsEqual(
        instance_template_revision.active, expected_active_keys)
    self.assertEqual(
        instance_group_managers[0].maximum_size, manager_cfgs[0].maximum_size)
    self.assertEqual(
        instance_group_managers[0].minimum_size, manager_cfgs[0].minimum_size)
    self.assertEqual(
        instance_group_managers[1].maximum_size, manager_cfgs[1].maximum_size)
    self.assertEqual(
        instance_group_managers[1].minimum_size, manager_cfgs[1].minimum_size)
    self.assertEqual(
        instance_group_managers[2].maximum_size, manager_cfgs[2].maximum_size)
    self.assertEqual(
        instance_group_managers[2].minimum_size, manager_cfgs[2].minimum_size)


if __name__ == '__main__':
  unittest.main()
