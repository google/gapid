#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for catalog.py."""

import collections
import datetime
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import datastore_utils
from components import machine_provider
from components import net
from components import utils
from test_support import test_case

import catalog
import instances
import models


class CatalogTest(test_case.TestCase):
  """Tests for catalog.catalog."""

  def test_not_found(self):
    """Ensures nothing happens when the instance doesn't exist."""
    def add_machine(*_args, **_kwargs):
      self.fail('add_machine called')
    self.mock(catalog.machine_provider, 'add_machine', add_machine)

    catalog.catalog(ndb.Key(models.Instance, 'fake-instance'))
    self.failIf(models.Instance.query().get())

  def test_already_cataloged(self):
    """Ensures nothing happens when the instance is already cataloged."""
    def add_machine(*_args, **_kwargs):
      self.fail('add_machine called')
    self.mock(catalog.machine_provider, 'add_machine', add_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=True,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.catalog(key)
    self.failUnless(key.get().cataloged)

  def test_pending_deletion(self):
    """Ensures nothing happens when the instance is pending deletion."""
    def add_machine(*_args, **_kwargs):
      self.fail('add_machine called')
    self.mock(catalog.machine_provider, 'add_machine', add_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        pending_deletion=True,
    ).put()

    catalog.catalog(key)
    self.failIf(key.get().cataloged)

  def test_instance_group_manager_not_found(self):
    """Ensures nothing happens when the instance group manager doesn't exist."""
    def add_machine(*_args, **_kwargs):
      self.fail('add_machine called')
    self.mock(catalog.machine_provider, 'add_machine', add_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.catalog(key)
    self.failIf(key.get().cataloged)

  def test_instance_template_revision_not_found(self):
    """Ensures nothing happens when instance template revision doesn't exist."""
    def add_machine(*_args, **_kwargs):
      self.fail('add_machine called')
    self.mock(catalog.machine_provider, 'add_machine', add_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.catalog(key)
    self.failIf(key.get().cataloged)

  def test_service_account_not_found(self):
    """Ensures nothing happens when a service account doesn't exist."""
    def add_machine(*_args, **_kwargs):
      self.fail('add_machine called')
    self.mock(catalog.machine_provider, 'add_machine', add_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
    ).put()

    catalog.catalog(key)
    self.failIf(key.get().cataloged)

  def test_cataloged(self):
    """Ensures an instance can be cataloged."""
    def add_machine(*_args, **_kwargs):
      return {}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(catalog.machine_provider, 'add_machine', add_machine)
    self.mock(catalog.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        service_accounts=[
            models.ServiceAccount(name='service-account'),
        ],
    ).put()

    catalog.catalog(key)
    self.failUnless(key.get().cataloged)

  def test_cataloging_error(self):
    """Ensures an instance isn't marked cataloged on error."""
    def add_machine(*_args, **_kwargs):
      return {'error': 'error'}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(catalog.machine_provider, 'add_machine', add_machine)
    self.mock(catalog.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        service_accounts=[
            models.ServiceAccount(name='service-account'),
        ],
    ).put()

    catalog.catalog(key)
    self.failIf(key.get().cataloged)

  def test_cataloging_error_hostname_reuse(self):
    """Ensures an instance is marked cataloged on HOSTNAME_REUSE."""
    def add_machine(*_args, **_kwargs):
      return {'error': 'HOSTNAME_REUSE'}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(catalog.machine_provider, 'add_machine', add_machine)
    self.mock(catalog.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        service_accounts=[
            models.ServiceAccount(name='service-account'),
        ],
    ).put()

    catalog.catalog(key)
    self.failUnless(key.get().cataloged)


class ExtractDimensionsTest(test_case.TestCase):
  """Tests for catalog.extract_dimensions."""

  def test_no_dimensions(self):
    """Ensures basic dimensions are returned when there are no others."""
    instance = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
    )
    instance_template_revision = models.InstanceTemplateRevision(
        project='project',
    )
    expected_dimensions = {
        'backend': 'GCE',
        'disk_type': 'HDD',
        'hostname': 'instance-name',
        'project': 'project',
    }

    self.assertEqual(
        catalog.extract_dimensions(instance, instance_template_revision),
        expected_dimensions,
    )

  def test_dimensions(self):
    """Ensures dimensions are returned."""
    instance = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
    )
    instance_template_revision = models.InstanceTemplateRevision(
        dimensions=machine_provider.Dimensions(
            os_family=machine_provider.OSFamily.LINUX,
        ),
        disk_size_gb=300,
        disk_type='pd-ssd',
        machine_type='n1-standard-8',
        project='project',
    )
    expected_dimensions = {
        'backend': 'GCE',
        'disk_gb': 300,
        'disk_type': 'SSD',
        'hostname': 'instance-name',
        'memory_gb': 30,
        'num_cpus': 8,
        'os_family': 'LINUX',
        'project': 'project',
    }

    self.assertEqual(
        catalog.extract_dimensions(instance, instance_template_revision),
        expected_dimensions,
    )


class RemoveTest(test_case.TestCase):
  """Tests for catalog.remove."""

  def test_not_found(self):
    """Ensures nothing happens when the instance doesn't exist."""
    def delete_machine(*_args, **_kwargs):
      self.fail('delete_machine called')
    self.mock(catalog.machine_provider, 'delete_machine', delete_machine)

    catalog.remove(ndb.Key(models.Instance, 'fake-instance'))
    self.failIf(models.Instance.query().get())

  def test_not_cataloged(self):
    """Ensures an instance is set for deletion when not cataloged."""
    def delete_machine(*_args, **_kwargs):
      return {'error': 'ENTRY_NOT_FOUND'}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(catalog.machine_provider, 'delete_machine', delete_machine)
    self.mock(catalog.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=False,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.remove(key)
    self.failIf(key.get().cataloged)
    self.failUnless(key.get().pending_deletion)

  def test_removed(self):
    """Ensures an instance can be removed."""
    def delete_machine(*_args, **_kwargs):
      return {}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(catalog.machine_provider, 'delete_machine', delete_machine)
    self.mock(catalog.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=True,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.remove(key)
    self.failUnless(key.get().cataloged)
    self.failUnless(key.get().pending_deletion)

  def test_removal_error(self):
    """Ensures an instance isn't set for deletion on error."""
    def delete_machine(*_args, **_kwargs):
      return {'error': 'error'}
    self.mock(catalog.machine_provider, 'delete_machine', delete_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=True,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.remove(key)
    self.failUnless(key.get().cataloged)
    self.failIf(key.get().pending_deletion)


class SetCatalogedTest(test_case.TestCase):
  """Tests for catalog.set_cataloged."""

  def test_not_found(self):
    """Ensures nothing happens when the instance doesn't exist."""
    catalog.set_cataloged(ndb.Key(models.Instance, 'fake-instance'))
    self.failIf(models.Instance.query().get())

  def test_cataloged(self):
    """Ensures an instance can be cataloged."""
    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=False,
    ).put()

    catalog.set_cataloged(key)
    self.failUnless(key.get().cataloged)


class UpdateCatalogedEntryTest(test_case.TestCase):
  """Tests for catalog.update_cataloged_entry."""

  def test_not_found(self):
    """Ensures nothing happens when the instance doesn't exist."""
    def retrieve_machine(*_args, **_kwargs):
      self.fail('retrieve_machine called')
    self.mock(catalog.machine_provider, 'retrieve_machine', retrieve_machine)

    catalog.update_cataloged_instance(ndb.Key(models.Instance, 'fake-instance'))
    self.failIf(models.Instance.query().get())

  def test_not_cataloged(self):
    """Ensures nothing happens when the instance is not cataloged."""
    def retrieve_machine(*_args, **_kwargs):
      self.fail('retrieve_machine called')
    self.mock(catalog.machine_provider, 'retrieve_machine', retrieve_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=False,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.update_cataloged_instance(key)
    self.failIf(key.get().cataloged)
    self.failIf(key.get().leased)
    self.failIf(key.get().pending_deletion)

  def test_updated_lease_expiration_ts(self):
    """Ensures an instance can be updated with a lease_expiration_ts."""
    now = int(utils.time_time())
    def retrieve_machine(*_args, **_kwargs):
      return {
          'lease_expiration_ts': str(now),
      }
    self.mock(catalog.machine_provider, 'retrieve_machine', retrieve_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=True,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    self.failIf(key.get().leased)
    catalog.update_cataloged_instance(key)
    self.failUnless(key.get().cataloged)
    self.assertEqual(
        key.get().lease_expiration_ts, datetime.datetime.utcfromtimestamp(now))
    self.failIf(key.get().leased_indefinitely)
    self.failUnless(key.get().leased)
    self.failIf(key.get().pending_deletion)

  def test_updated_leased_indefinitely(self):
    """Ensures an instance can be updated with leased_indefinitely."""
    def retrieve_machine(*_args, **_kwargs):
      return {
          'leased_indefinitely': True,
      }
    self.mock(catalog.machine_provider, 'retrieve_machine', retrieve_machine)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=True,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    self.failIf(key.get().leased)
    catalog.update_cataloged_instance(key)
    self.failUnless(key.get().cataloged)
    self.failUnless(key.get().leased_indefinitely)
    self.failIf(key.get().lease_expiration_ts)
    self.failUnless(key.get().leased)
    self.failIf(key.get().pending_deletion)

  def test_retrieval_error(self):
    """Ensures an instance is set for deletion when not found."""
    def retrieve_machine(*_args, **_kwargs):
      raise net.NotFoundError('404', 404, '404')
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(catalog.machine_provider, 'retrieve_machine', retrieve_machine)
    self.mock(catalog.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        cataloged=True,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()

    catalog.update_cataloged_instance(key)
    self.failUnless(key.get().cataloged)
    self.failIf(key.get().leased)
    self.failUnless(key.get().pending_deletion)


if __name__ == '__main__':
  unittest.main()
