#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for instances.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import net
from components import utils
from test_support import test_case

import instance_group_managers
import instances
import models


class AddLeaseExpirationTsTest(test_case.TestCase):
  """Tests for instances.add_lease_expiration_ts."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.Instance, 'fake-instance')

    instances.add_lease_expiration_ts(key, utils.utcnow())

    self.failIf(key.get())

  def test_lease_expiration_ts_added(self):
    now = utils.utcnow()
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
    ).put()

    instances.add_lease_expiration_ts(key, now)

    self.assertEqual(key.get().lease_expiration_ts, now)
    self.failUnless(key.get().leased)

  def test_lease_expiration_ts_matches(self):
    now = utils.utcnow()
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        lease_expiration_ts=now,
    ).put()

    instances.add_lease_expiration_ts(key, now)

    self.assertEqual(key.get().lease_expiration_ts, now)
    self.failUnless(key.get().leased)

  def test_lease_expiration_ts_updated(self):
    now = utils.utcnow()
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        lease_expiration_ts=utils.utcnow(),
    ).put()

    instances.add_lease_expiration_ts(key, now)

    self.assertEqual(key.get().lease_expiration_ts, now)
    self.failUnless(key.get().leased)


class DeletePendingTest(test_case.TestCase):
  """Tests for instances.delete_pending."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.Instance, 'fake-instance')

    instances.delete_pending(key)

    self.failIf(key.get())

  def test_not_pending_deletion(self):
    """Ensures nothing happens when the instance isn't pending deletion."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        pending_deletion=False,
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)

  def test_url_unspecified(self):
    """Ensures nothing happens when the URL doesn't exist."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)

  def test_parent_unspecified(self):
    """Ensures nothing happens when the parent doesn't exist."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)

  def test_grandparent_unspecified(self):
    """Ensures nothing happens when the grandparent doesn't exist."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()

    instances.delete_pending(key)

  def test_project_unspecified(self):
    """Ensures nothing happens when the project is unspecified."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
    ).put()

    instances.delete_pending(key)

  def test_deleted(self):
    """Ensures instance is deleted."""
    def json_request(*_args, **_kwargs):
      return {'status': 'DONE'}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)

  def test_deletion_ts(self):
    """Ensures deletion_ts is not overwritten, but deletion call is repeated."""
    def json_request(*_args, **_kwargs):
      return {'status': 'DONE'}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    now = utils.utcnow()
    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        deletion_ts=now,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        pending_deletion=True,
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)
    self.assertEqual(key.get().deletion_ts, now)

  def test_deleted_not_done(self):
    """Ensures nothing happens when instance deletion status is not DONE."""
    def json_request(*_args, **_kwargs):
      return {'status': 'error'}
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)

  def test_already_deleted(self):
    """Ensures errors are ignored when the instance is already deleted."""
    def json_request(*_args, **_kwargs):
      raise net.Error('400', 400, '400')
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    instances.delete_pending(key)

  def test_error_surfaced(self):
    """Ensures errors are surfaced."""
    def json_request(*_args, **_kwargs):
      raise net.Error('403', 403, '403')
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.net, 'json_request', json_request)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

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
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        project='project',
    ).put()

    self.assertRaises(net.Error, instances.delete_pending, key)


class GetInstanceGroupManagerKeyTest(test_case.TestCase):
  """Tests for instances.get_instance_group_manager_key."""

  def test_equal(self):
    expected = instance_group_managers.get_instance_group_manager_key(
        'base-name',
        'revision',
        'zone',
    )

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )

    self.assertEqual(instances.get_instance_group_manager_key(key), expected)


class EnsureEntityExistsTest(test_case.TestCase):
  """Tests for instances.ensure_entity_exists."""

  def test_creates(self):
    """Ensures entity is created when it doesn't exist."""
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    expected_url = 'url'

    future = instances.ensure_entity_exists(
        key,
        expected_url,
        instances.get_instance_group_manager_key(key),
    )
    future.wait()

    self.assertEqual(key.get().url, expected_url)

  def test_entity_exists(self):
    """Ensures nothing happens when the entity already exists."""
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
    ).put()

    future = instances.ensure_entity_exists(
        key, 'url', instances.get_instance_group_manager_key(key))
    future.wait()

    self.failIf(key.get().url)

  def test_entity_not_put(self):
    """Ensures nothing happens when the entity wasn't put."""
    @ndb.tasklet
    def _ensure_entity_exists(*_args, **_kwargs):
      raise ndb.Return(False)
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances, '_ensure_entity_exists', _ensure_entity_exists)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )

    future = instances.ensure_entity_exists(
        key, 'url', instances.get_instance_group_manager_key(key))
    future.wait()

    self.failIf(key.get())


class EnsureEntitiesExistTest(test_case.TestCase):
  """Tests for instances.ensure_entities_exist."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')

    instances.ensure_entities_exist(key)
    self.failIf(key.get())

  def test_url_unspecified(self):
    """Ensures nothing happens when URL is unspecified."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()

    instances.ensure_entities_exist(key)
    self.failIf(key.get().instances)

  def test_no_instances(self):
    """Ensures nothing happens when there are no instances."""
    def fetch(*_args, **_kwargs):
      return []
    self.mock(instances, 'fetch', fetch)

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()

    instances.ensure_entities_exist(key)
    self.failIf(key.get().instances)

  def test_already_exists(self):
    """Ensures nothing happens when the entity already exists."""
    def fetch(*_args, **_kwargs):
      return ['url/name']
    self.mock(instances, 'fetch', fetch)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'name',
    )
    models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    expected_instances = [
        key,
    ]

    instances.ensure_entities_exist(
        instances.get_instance_group_manager_key(key))
    self.failIf(key.get().url)
    self.assertItemsEqual(
        instances.get_instance_group_manager_key(key).get().instances,
        expected_instances,
    )

  def test_creates(self):
    """Ensures entity gets created."""
    def fetch(*_args, **_kwargs):
      return ['url/name']
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances, 'fetch', fetch)
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'name',
    )
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    expected_instances = [
        key,
    ]
    expected_url = 'url/name'

    instances.ensure_entities_exist(
        instances.get_instance_group_manager_key(key))
    self.assertItemsEqual(
        instances.get_instance_group_manager_key(key).get().instances,
        expected_instances,
    )
    self.assertEqual(key.get().url, expected_url)


class FetchTest(test_case.TestCase):
  """Tests for instances.fetch."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')
    urls = instances.fetch(key)
    self.failIf(urls)

  def test_url_unspecified(self):
    """Ensures nothing happens when URL is unspecified."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplateRevision(key=key.parent(), project='project').put()

    urls = instances.fetch(key)
    self.failIf(urls)

  def test_parent_doesnt_exist(self):
    """Ensures nothing happens when the parent doesn't exist."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()

    urls = instances.fetch(key)
    self.failIf(urls)

  def test_parent_project_unspecified(self):
    """Ensures nothing happens when parent doesn't specify a project."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    models.InstanceTemplateRevision(key=key.parent()).put()

    urls = instances.fetch(key)
    self.failIf(urls)

  def test_no_instances(self):
    """Ensures nothing happens when there are no instances."""
    def get_instances_in_instance_group(*_args, **_kwargs):
      return {}
    self.mock(
        instances.gce.Project,
        'get_instances_in_instance_group',
        get_instances_in_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    models.InstanceTemplateRevision(key=key.parent(), project='project').put()

    urls = instances.fetch(key)
    self.failIf(urls)

  def test_instances(self):
    """Ensures instances are returned."""
    def get_instances_in_instance_group(*_args, **_kwargs):
      return {
          'instanceGroup': 'instance-group-url',
          'items': [
              {'instance': 'url/instance'},
          ],
      }
    self.mock(
        instances.gce.Project,
        'get_instances_in_instance_group',
        get_instances_in_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    models.InstanceTemplateRevision(key=key.parent(), project='project').put()
    expected_urls = ['url/instance']

    urls = instances.fetch(key)
    self.assertItemsEqual(urls, expected_urls)

  def test_instances_with_page_token(self):
    """Ensures all instances are returned."""
    def get_instances_in_instance_group(*_args, **kwargs):
      if kwargs.get('page_token'):
        return {
            'items': [
                {'instance': 'url/instance-2'},
            ],
        }
      return {
          'items': [
              {'instance': 'url/instance-1'},
          ],
          'nextPageToken': 'page-token',
      }
    self.mock(
        instances.gce.Project,
        'get_instances_in_instance_group',
        get_instances_in_instance_group,
    )

    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    models.InstanceTemplateRevision(key=key.parent(), project='project').put()
    expected_urls = ['url/instance-1', 'url/instance-2']

    urls = instances.fetch(key)
    self.assertItemsEqual(urls, expected_urls)


class MarkForDeletionTest(test_case.TestCase):
  """Tests for instances.mark_for_deletion."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity doesn't exist."""
    def send_machine_event(*_args, **_kwargs):
      self.fail('send_machine_event called')
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    key = ndb.Key(models.Instance, 'fake-instance')

    instances.mark_for_deletion(key)

    self.failIf(key.get())

  def test_marked(self):
    """Ensures the entity can be marked for deletion."""
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(instances.metrics, 'send_machine_event', send_machine_event)

    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        lease_expiration_ts=utils.utcnow(),
    ).put()

    instances.mark_for_deletion(key)

    self.failUnless(key.get().pending_deletion)
    self.failIf(key.get().lease_expiration_ts)
    self.failIf(key.get().leased)


class SetDeletionTime(test_case.TestCase):
  """Tests for instances.set_deletion_time."""

  def test_not_found(self):
    """Ensures nothing happens when the Instance doesn't exist."""
    now = utils.utcnow()
    key = ndb.Key(models.Instance, 'fake-instance')

    instances.set_deletion_time(key, now)
    self.failIf(key.get())

  def test_already_set(self):
    """Ensures nothing happens when the deletion_ts is already set."""
    now = utils.utcnow()
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        deletion_ts=utils.utcnow(),
    ).put()

    instances.set_deletion_time(key, now)
    self.failUnless(key.get().deletion_ts)
    self.assertNotEqual(key.get().deletion_ts, now)

  def test_sets(self):
    """Ensures the deletion_ts can be set."""
    now = utils.utcnow()
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
    ).put()

    instances.set_deletion_time(key, now)
    self.assertEqual(key.get().deletion_ts, now)


class SetLeasedIndefinitelyTest(test_case.TestCase):
  """Tests for instances.set_leased_indefinitely."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.Instance, 'fake-instance')

    instances.set_leased_indefinitely(key)

    self.failIf(key.get())

  def test_leased_indefinitely_set(self):
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
    ).put()

    instances.set_leased_indefinitely(key)

    self.failUnless(key.get().leased_indefinitely)
    self.failUnless(key.get().leased)

  def test_leased_indefinitely_matches(self):
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        leased_indefinitely=True,
    ).put()

    instances.set_leased_indefinitely(key)

    self.failUnless(key.get().leased_indefinitely)
    self.failUnless(key.get().leased)

  def test_leased_indefinitely_updated(self):
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        leased_indefinitely=False,
    ).put()

    instances.set_leased_indefinitely(key)

    self.failUnless(key.get().leased_indefinitely)
    self.failUnless(key.get().leased)


if __name__ == '__main__':
  unittest.main()
