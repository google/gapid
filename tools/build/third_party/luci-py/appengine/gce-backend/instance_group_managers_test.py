#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for instance_group_managers.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import datastore_utils
from components import net
from test_support import test_case

import instance_group_managers
import models


class CountInstancesTest(test_case.TestCase):
  """Tests for instance_group_managers.count_instances."""

  def test_no_instance_group_managers(self):
    self.failIf(instance_group_managers.count_instances())

  def test_one_instance_group_manager_no_instances(self):
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    expected = {'base-name': [0, 0]}

    actual = instance_group_managers.count_instances()
    self.assertEqual(actual, expected)

  def test_one_instance_group_manager(self):
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key-1'),
            ndb.Key(models.Instance, 'fake-key-2'),
        ],
    ).put()
    expected = {'base-name': [2, 0]}

    actual = instance_group_managers.count_instances()
    self.assertEqual(actual, expected)

  def test_one_instance_group_manager_drained(self):
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key-1'),
            ndb.Key(models.Instance, 'fake-key-2'),
        ],
    ).put()
    def get_drained_instance_group_managers(*_args, **_kwargs):
      return [key]
    self.mock(
        instance_group_managers,
        'get_drained_instance_group_managers',
        get_drained_instance_group_managers,
    )
    expected = {'base-name': [0, 2]}

    actual = instance_group_managers.count_instances()
    self.assertEqual(actual, expected)

  def test_several_instance_group_managers(self):
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name-1',
            'revision',
            'zone-a',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key-1'),
            ndb.Key(models.Instance, 'fake-key-2'),
        ],
    ).put()
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name-1',
            'revision',
            'zone-b',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key-3'),
        ],
    ).put()
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name-2',
            'revision',
            'zone',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key-4'),
        ],
    ).put()
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name-3',
            'revision',
            'zone',
        ),
    ).put()
    def get_drained_instance_group_managers(*_args, **_kwargs):
      return [key]
    self.mock(
        instance_group_managers,
        'get_drained_instance_group_managers',
        get_drained_instance_group_managers,
    )
    expected = {
        'base-name-1': [2, 1], 'base-name-2': [1, 0], 'base-name-3': [0, 0]}

    actual = instance_group_managers.count_instances()
    self.assertEqual(actual, expected)


class CreateTest(test_case.TestCase):
  """Tests for instance_group_managers.create."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')

    instance_group_managers.create(key)

    self.failIf(key.get())

  def test_url_specified(self):
    """Ensures nothing happens when URL is already specified."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    expected_url = 'url'

    instance_group_managers.create(key)
    self.assertEqual(key.get().url, expected_url)

  def test_parent_doesnt_exist(self):
    """Ensures nothing happens when the parent doesn't exist."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()

    instance_group_managers.create(key)
    self.failIf(key.get().url)

  def test_parent_project_unspecified(self):
    """Ensures nothing happens when parent doesn't specify project."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplateRevision(key=key.parent(), url='url').put()

    instance_group_managers.create(key)
    self.failIf(key.get().url)

  def test_parent_url_unspecified(self):
    """Ensures nothing happens when parent doesn't specify URL."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplateRevision(key=key.parent(), project='project').put()

    instance_group_managers.create(key)
    self.failIf(key.get().url)

  def test_creates(self):
    """Ensures an instance group manager is created."""
    def create_instance_group_manager(*_args, **_kwargs):
      return {'targetLink': 'url'}
    self.mock(
        instance_group_managers.gce.Project,
        'create_instance_group_manager',
        create_instance_group_manager,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=2,
        maximum_size=2,
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='project',
        url='instance-template-url',
    ).put()
    expected_url = 'url'

    instance_group_managers.create(key)
    self.assertEqual(key.get().url, expected_url)

  def test_updates_when_already_created(self):
    """Ensures an instance group manager is updated when already created."""
    def create_instance_group_manager(*_args, **_kwargs):
      raise net.Error('', 409, '')
    def get_instance_group_manager(*_args, **_kwargs):
      return {'selfLink': 'url'}
    self.mock(
        instance_group_managers.gce.Project,
        'create_instance_group_manager',
        create_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=2,
        maximum_size=2,
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='project',
        url='instance-template-url',
    ).put()
    expected_url = 'url'

    instance_group_managers.create(key)
    self.assertEqual(key.get().url, expected_url)

  def test_doesnt_update_when_creation_fails(self):
    """Ensures an instance group manager is not updated when creation fails."""
    def create_instance_group_manager(*_args, **_kwargs):
      raise net.Error('', 400, '')
    self.mock(
        instance_group_managers.gce.Project,
        'create_instance_group_manager',
        create_instance_group_manager,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=2,
        maximum_size=2,
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='project',
        url='instance-template-url',
    ).put()

    self.assertRaises(net.Error, instance_group_managers.create, key)
    self.failIf(key.get().url)


class DeleteTest(test_case.TestCase):
  """Tests for instance_group_managers.delete."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')

    instance_group_managers.delete(key)
    self.failIf(key.get())

  def test_deletes(self):
    """Ensures an instance group manager is deleted."""
    def json_request(url, *_args, **_kwargs):
      return {'targetLink': url}
    self.mock(instance_group_managers.net, 'json_request', json_request)

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()

    instance_group_managers.delete(key)
    self.failIf(key.get().url)

  def test_target_link_mismatch(self):
    """Ensures nothing happens when targetLink doesn't match."""
    def json_request(*_args, **_kwargs):
      return {'targetLink': 'mismatch'}
    self.mock(instance_group_managers.net, 'json_request', json_request)

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()

    instance_group_managers.delete(key)
    self.assertEqual(key.get().url, 'url')

  def test_url_unspecified(self):
    """Ensures nothing happens when URL is unspecified."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()

    instance_group_managers.delete(key)
    self.failIf(key.get().url)

  def test_url_not_found(self):
    """Ensures URL is updated when the instance group manager is not found."""
    def json_request(*_args, **_kwargs):
      raise net.Error('', 404, '')
    self.mock(instance_group_managers.net, 'json_request', json_request)

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()

    instance_group_managers.delete(key)
    self.failIf(key.get().url)

  def test_deletion_fails(self):
    """Ensures nothing happens when instance group manager deletion fails."""
    def json_request(*_args, **_kwargs):
      raise net.Error('', 400, '')
    self.mock(instance_group_managers.net, 'json_request', json_request)

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    expected_url = 'url'

    self.assertRaises(net.Error, instance_group_managers.delete, key)
    self.assertEqual(key.get().url, expected_url)


class GetDrainedInstanceGroupManagersTest(test_case.TestCase):
  """Tests for instance_group_managers.get_drained_instance_group_managers."""

  def test_no_entities(self):
    """Ensures nothing is returned when there are no entities."""
    self.failIf(instance_group_managers.get_drained_instance_group_managers())

  def test_nothing_active_or_drained(self):
    """Ensures nothing is returned when there are no active/drained entities."""
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()

    self.failIf(instance_group_managers.get_drained_instance_group_managers())

  def test_active_only(self):
    """Ensures nothing is returned when there are only active entities."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
    ).put()

    self.failIf(instance_group_managers.get_drained_instance_group_managers())

  def test_drained(self):
    """Ensures drained entities are returned."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()
    expected_keys = [key]

    self.assertItemsEqual(
        instance_group_managers.get_drained_instance_group_managers(),
        expected_keys,
    )

  def test_implicitly_drained(self):
    """Ensures implicitly drained entities are returned."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent().parent(),
        drained=[
            key.parent(),
        ],
    ).put()
    expected_keys = [
        key,
    ]

    self.assertItemsEqual(
        instance_group_managers.get_drained_instance_group_managers(),
        expected_keys,
    )


class GetBaseNameTest(test_case.TestCase):
  """Tests for instance_group_managers.get_base_name."""

  def test_get_base_name(self):
    """Ensures base instance name is generated correctly."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision-extra-long',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    self.assertEqual(
        instance_group_managers.get_base_name(key.get()),
        'base-name-revision-zone',
    )


class GetInstanceGroupManagerToDeleteTest(test_case.TestCase):
  """Tests for instance_group_managers.get_instance_group_manager_to_delete."""

  def test_entity_doesnt_exist(self):
    """Ensures no URL when the entity doesn't exist."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')
    self.failIf(
        instance_group_managers.get_instance_group_manager_to_delete(key))

  def test_instances(self):
    """Ensures no URL when there are active instances."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key'),
        ],
        url='url',
    ).put()

    self.failIf(
        instance_group_managers.get_instance_group_manager_to_delete(key))

  def test_url_unspecified(self):
    """Ensures no URL when URL is unspecified."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()

    self.failIf(
        instance_group_managers.get_instance_group_manager_to_delete(key))

  def test_returns_url(self):
    """Ensures URL is returned."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    expected_url = 'url'

    self.assertEqual(
        instance_group_managers.get_instance_group_manager_to_delete(key),
        expected_url,
    )


class ResizeTest(test_case.TestCase):
  """Tests for instance_group_managers.resize."""

  def test_entity_doesnt_exist(self):
    def get_instance_group_manager(*_args, **_kwargs):
      self.fail('get_instance_group_manager called')

    def resize_managed_instance_group(*_args, **_kwargs):
      self.fail('resize_managed_instance_group called')

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = ndb.Key(models.InstanceGroupManager, 'fake-key')
    instance_group_managers.resize(key)
    self.failIf(key.get())

  def test_no_url(self):
    def get_instance_group_manager(*_args, **_kwargs):
      self.fail('get_instance_group_manager called')

    def resize_managed_instance_group(*_args, **_kwargs):
      self.fail('resize_managed_instance_group called')

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_parent_doesnt_exist(self):
    def get_instance_group_manager(*_args, **_kwargs):
      self.fail('get_instance_group_manager called')

    def resize_managed_instance_group(*_args, **_kwargs):
      self.fail('resize_managed_instance_group called')

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
        url='https://example.com',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_no_project(self):
    def get_instance_group_manager(*_args, **_kwargs):
      self.fail('get_instance_group_manager called')

    def resize_managed_instance_group(*_args, **_kwargs):
      self.fail('resize_managed_instance_group called')

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_resize_no_actions(self):
    def get_instance_group_manager(*_args, **_kwargs):
      return {
          'name': 'name',
      }

    def resize_managed_instance_group(*_args, **_kwargs):
      self.fail('resize_managed_instance_group called')

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_resize_none_created(self):
    def get_instance_group_manager(*_args, **_kwargs):
      return {
          'currentActions': {
              'none': 0,
          },
          'name': 'name',
          'targetSize': 0,
      }

    def resize_managed_instance_group(_, name, zone, size):
      self.assertEqual(name, 'name')
      self.assertEqual(zone, 'zone')
      self.assertEqual(size, 10)

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_resize_some_created(self):
    def get_instance_group_manager(*_args, **_kwargs):
      return {
          'currentActions': {
              'none': 3,
          },
          'name': 'name',
          'targetSize': 3,
      }

    def resize_managed_instance_group(_, name, zone, size):
      self.assertEqual(name, 'name')
      self.assertEqual(zone, 'zone')
      self.assertEqual(size, 103)

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=1000,
        maximum_size=1000,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_resize_all_created(self):
    def get_instance_group_manager(*_args, **_kwargs):
      return {
          'currentActions': {
              'none': 10,
          },
          'name': 'name',
          'targetSize': 10,
      }

    def resize_managed_instance_group(*_args, **_kwargs):
      self.failIf(True)

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=10,
        maximum_size=10,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_resize_excess_created(self):
    def get_instance_group_manager(*_args, **_kwargs):
      return {
          'currentActions': {
              'none': 2,
          },
          'name': 'name',
          'targetSize': 2,
      }

    def resize_managed_instance_group(*_args, **_kwargs):
      self.failIf(True)

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        minimum_size=1,
        maximum_size=1,
        url='https://example.com',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)

  def test_resize_other_revisions_created(self):
    def get_instance_group_manager(*_args, **_kwargs):
      return {
          'currentActions': {
              'none': 0,
          },
          'name': 'name',
	  'targetSize': 0,
      }

    def resize_managed_instance_group(_, name, zone, size):
      self.assertEqual(name, 'name')
      self.assertEqual(zone, 'zone')
      self.assertEqual(size, 4)

    self.mock(
        instance_group_managers.gce.Project,
        'get_instance_group_manager',
        get_instance_group_manager,
    )
    self.mock(
        instance_group_managers.gce.Project,
        'resize_managed_instance_group',
        resize_managed_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision-1',
            'zone',
        ),
        minimum_size=7,
        maximum_size=7,
        url='https://example.com',
    ).put()
    models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision-2',
            'zone',
        ),
        instances=[
            ndb.Key(models.Instance, 'instance-name-1'),
            ndb.Key(models.Instance, 'instance-name-2'),
            ndb.Key(models.Instance, 'instance-name-3'),
        ],
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        project='fake-project',
    ).put()
    models.InstanceTemplate(key=key.parent().parent()).put()

    instance_group_managers.resize(key)


class ScheduleCreationTest(test_case.TestCase):
  """Tests for instance_group_managers.schedule_creation."""

  def setUp(self, *args, **kwargs):
    def enqueue_task(_taskqueue, key):
      entity = key.get()
      entity.url = key.urlsafe()
      entity.put()
      return True

    super(ScheduleCreationTest, self).setUp(*args, **kwargs)
    self.mock(instance_group_managers.utilities, 'enqueue_task', enqueue_task)

  def test_enqueues_task(self):
    """Ensures a task is enqueued."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
        url='url',
    ).put()
    models.InstanceGroupManager(key=key).put()
    expected_url = key.urlsafe()

    instance_group_managers.schedule_creation()

    self.assertEqual(key.get().url, expected_url)

  def test_instance_template_revision_inactive(self):
    """Ensures no task is enqueued for inactive instance template revisions."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
        url='url',
    ).put()
    models.InstanceGroupManager(key=key).put()

    instance_group_managers.schedule_creation()

    self.failIf(key.get().url)

  def test_instance_template_revision_missing(self):
    """Ensures no task is enqueued for missing instance template revisions."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceGroupManager(key=key).put()

    instance_group_managers.schedule_creation()

    self.failIf(key.get().url)

  def test_instance_template_revision_no_url(self):
    """Ensures no task is enqueued when instance template URL is missing."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
    ).put()
    models.InstanceGroupManager(key=key).put()

    instance_group_managers.schedule_creation()

    self.failIf(key.get().url)

  def test_instance_group_manager_inactive(self):
    """Ensures no task is enqueued for inactive instance group managers."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        url='url',
    ).put()
    models.InstanceGroupManager(key=key).put()

    instance_group_managers.schedule_creation()

    self.failIf(key.get().url)

  def test_instance_group_manager_drained(self):
    """Ensures no task is enqueued for drained instance group managers."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        drained=[
            key,
        ],
        url='url',
    ).put()
    models.InstanceGroupManager(key=key).put()

    instance_group_managers.schedule_creation()

    self.failIf(key.get().url)

  def test_instance_group_manager_missing(self):
    """Ensures no task is enqueued for missing instance group managers."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
        url='url',
    ).put()
    key = models.InstanceGroupManager().put()

    instance_group_managers.schedule_creation()

    self.failIf(key.get().url)

  def test_instance_group_manager_already_created(self):
    """Ensures no task is enqueued for existing instance group managers."""
    key = instance_group_managers.get_instance_group_manager_key(
        'base-name', 'revision', 'zone')
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        active=[
            key,
        ],
        url='instance-template-url',
    ).put()
    models.InstanceGroupManager(key=key, url='url').put()
    expected_url = 'url'

    instance_group_managers.schedule_creation()

    self.assertEqual(key.get().url, expected_url)


class UpdateURLTest(test_case.TestCase):
  """Tests for instance_group_managers.update_url."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')
    instance_group_managers.update_url(key, 'url')
    self.failIf(key.get())

  def test_url_matches(self):
    """Ensures nothing happens when the URL already matches."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()

    instance_group_managers.update_url(key, 'url')

    self.assertEqual(key.get().url, 'url')

  def test_url_mismatch(self):
    """Ensures the URL is updated when it doesn't match."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='old-url',
    ).put()

    instance_group_managers.update_url(key, 'new-url')

    self.assertEqual(key.get().url, 'new-url')

  def test_url_updated(self):
    """Ensures the URL is updated."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()

    instance_group_managers.update_url(key, 'url')

    self.assertEqual(key.get().url, 'url')


if __name__ == '__main__':
  unittest.main()
