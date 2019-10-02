#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for cleanup.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import net
from test_support import test_case

import cleanup
import instance_group_managers
import instance_templates
import instances
import models


class CheckDeletedInstanceTest(test_case.TestCase):
  """Tests for cleanup.check_deleted_instance."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    key = ndb.Key(models.Instance, 'fake-key')

    cleanup.check_deleted_instance(key)

    self.failIf(key.get())

  def test_not_pending_deletion(self):
    """Ensures nothing happens when the entity is not pending deletion."""
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        url='url',
    ).put()

    cleanup.check_deleted_instance(key)

    self.failIf(key.get().deleted)

  def test_no_url(self):
    """Ensures nothing happens when the entity has no URL."""
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        pending_deletion=True,
    ).put()

    cleanup.check_deleted_instance(key)

    self.failIf(key.get().deleted)

  def test_exists(self):
    """Ensures nothing happens when the instance still exists."""
    def json_request(*_args, **_kwargs):
      return {}
    self.mock(cleanup.net, 'json_request', json_request)

    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        pending_deletion=True,
        url='url',
    ).put()

    cleanup.check_deleted_instance(key)

    self.failIf(key.get().deleted)

  def test_deleted(self):
    """Ensures the entity is marked deleted when the instance doesn't exists."""
    def json_request(*_args, **_kwargs):
      raise net.NotFoundError('404', 404, '404')
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(cleanup.net, 'json_request', json_request)
    self.mock(cleanup.metrics, 'send_machine_event', send_machine_event)

    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        pending_deletion=True,
        url='url',
    ).put()

    cleanup.check_deleted_instance(key)

    self.failUnless(key.get().deleted)


class CleanupDeletedInstanceTest(test_case.TestCase):
  """Tests for cleanup.cleanup_deleted_instance."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    key = ndb.Key(models.Instance, 'fake-key')

    cleanup.cleanup_deleted_instance(key)

    self.failIf(key.get())

  def test_not_deleted(self):
    """Ensures nothing happens when the instance is not deleted."""
    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        deleted=False,
    ).put()

    cleanup.cleanup_deleted_instance(key)

    self.failUnless(key.get())

  def test_deletes(self):
    """Ensures the entity is deleted."""
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(cleanup.metrics, 'send_machine_event', send_machine_event)

    key = models.Instance(
        key=instances.get_instance_key(
            'base-name',
            'revision',
            'zone',
            'instance-name',
        ),
        deleted=True,
    ).put()

    cleanup.cleanup_deleted_instance(key)

    self.failIf(key.get())


class CleanupDrainedInstanceTest(test_case.TestCase):
  """Tests for cleanup.cleanup_drained_instance."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    self.mock(cleanup.net, 'json_request', json_request)

    key = ndb.Key(models.Instance, 'fake-key')

    cleanup.cleanup_drained_instance(key)

    self.failIf(key.get())

  def test_url_unspecified(self):
    """Ensures nothing happens when the entity has no URL."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    self.mock(cleanup.net, 'json_request', json_request)

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
    models.InstanceTemplate(
        key=instances.get_instance_group_manager_key(key).parent().parent(),
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failIf(key.get().deleted)

  def test_parent_unspecified(self):
    """Ensures nothing happens when the parent doesn't exist."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    self.mock(cleanup.net, 'json_request', json_request)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
    ).put()
    models.InstanceTemplate(
        key=instances.get_instance_group_manager_key(key).parent().parent(),
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failIf(key.get().deleted)

  def test_grandparent_unspecified(self):
    """Ensures nothing happens when the grandparent doesn't exist."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    self.mock(cleanup.net, 'json_request', json_request)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplate(
        key=instances.get_instance_group_manager_key(key).parent().parent(),
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failIf(key.get().deleted)

  def test_root_unspecified(self):
    """Ensures nothing happens when the parent doesn't exist."""
    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failIf(key.get().deleted)

  def test_not_drained(self):
    """Ensures nothing happens when the parent is not drained."""
    def json_request(*_args, **_kwargs):
      self.fail('json_request called')
    self.mock(cleanup.net, 'json_request', json_request)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
    ).put()
    models.InstanceTemplate(
        key=instances.get_instance_group_manager_key(key).parent().parent(),
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failIf(key.get().deleted)

  def test_drained(self):
    """Ensures the entity is marked deleted when the parent is drained."""
    def json_request(*_args, **_kwargs):
      raise net.NotFoundError('404', 404, '404')
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(cleanup.net, 'json_request', json_request)
    self.mock(cleanup.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
        drained=[
            instances.get_instance_group_manager_key(key),
        ],
    ).put()
    models.InstanceTemplate(
        key=instances.get_instance_group_manager_key(key).parent().parent(),
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failUnless(key.get().deleted)

  def test_implicitly_drained(self):
    """Ensures the entity is marked deleted when the grandparent is drained."""
    def json_request(*_args, **_kwargs):
      raise net.NotFoundError('404', 404, '404')
    def send_machine_event(*_args, **_kwargs):
      pass
    self.mock(cleanup.net, 'json_request', json_request)
    self.mock(cleanup.metrics, 'send_machine_event', send_machine_event)

    key = instances.get_instance_key(
        'base-name',
        'revision',
        'zone',
        'instance-name',
    )
    key = models.Instance(
        key=key,
        instance_group_manager=instances.get_instance_group_manager_key(key),
        url='url',
    ).put()
    models.InstanceGroupManager(
        key=instances.get_instance_group_manager_key(key),
    ).put()
    models.InstanceTemplateRevision(
        key=instances.get_instance_group_manager_key(key).parent(),
    ).put()
    models.InstanceTemplate(
        key=instances.get_instance_group_manager_key(key).parent().parent(),
        drained=[
            instances.get_instance_group_manager_key(key).parent(),
        ],
    ).put()

    cleanup.cleanup_drained_instance(key)

    self.failUnless(key.get().deleted)


class CleanupInstanceGroupManagersTest(test_case.TestCase):
  """Tests for cleanup.cleanup_instance_group_managers."""

  def test_no_entities(self):
    def get_drained_instance_group_managers(*_args, **_kwargs):
      return []
    @ndb.tasklet
    def delete_instance_group_manager(*_args, **_kwargs):
      self.fail('delete_instance_group_manager called')
    self.mock(
        cleanup.instance_group_managers,
        'get_drained_instance_group_managers',
        get_drained_instance_group_managers,
    )
    self.mock(
        cleanup, 'delete_instance_group_manager', delete_instance_group_manager)

    cleanup.cleanup_instance_group_managers()
    self.failIf(models.InstanceGroupManager.query().count())

  def test_deletes(self):
    def get_drained_instance_group_managers(*_args, **_kwargs):
      return [
          ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
          ndb.Key(models.InstanceGroupManager, 'fake-key-3'),
          ndb.Key(models.InstanceGroupManager, 'fake-key-4'),
      ]
    @ndb.tasklet
    def delete_instance_group_manager(key):
      yield key.delete_async()
    self.mock(
        cleanup.instance_group_managers,
        'get_drained_instance_group_managers',
        get_drained_instance_group_managers,
    )
    self.mock(
        cleanup, 'delete_instance_group_manager', delete_instance_group_manager)
    models.InstanceGroupManager(
        key=ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
    ).put()
    models.InstanceGroupManager(
        key=ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
    ).put()
    models.InstanceGroupManager(
        key=ndb.Key(models.InstanceGroupManager, 'fake-key-3'),
    ).put()
    models.InstanceGroupManager(
        key=ndb.Key(models.InstanceGroupManager, 'fake-key-4'),
    ).put()

    cleanup.cleanup_instance_group_managers(max_concurrent=2)
    self.failIf(ndb.Key(models.InstanceGroupManager, 'fake-key-1').get())
    self.failUnless(ndb.Key(models.InstanceGroupManager, 'fake-key-2').get())
    self.failIf(ndb.Key(models.InstanceGroupManager, 'fake-key-3').get())
    self.failIf(ndb.Key(models.InstanceGroupManager, 'fake-key-4').get())


class CleanupInstanceTemplateRevisionsTest(test_case.TestCase):
  """Tests for cleanup.cleanup_instance_template_revisions."""

  def test_no_entities(self):
    def get_drained_instance_template_revisions(*_args, **_kwargs):
      return []
    @ndb.tasklet
    def delete_instance_template_revision(*_args, **_kwargs):
      self.fail('delete_instance_template_revision called')
    self.mock(
        cleanup.instance_templates,
        'get_drained_instance_template_revisions',
        get_drained_instance_template_revisions,
    )
    self.mock(
        cleanup,
        'delete_instance_template_revision',
        delete_instance_template_revision,
    )

    cleanup.cleanup_instance_template_revisions()
    self.failIf(models.InstanceTemplateRevision.query().count())

  def test_deletes(self):
    def get_drained_instance_template_revisions(*_args, **_kwargs):
      return [
          ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
          ndb.Key(models.InstanceTemplateRevision, 'fake-key-3'),
          ndb.Key(models.InstanceTemplateRevision, 'fake-key-4'),
      ]
    @ndb.tasklet
    def delete_instance_template_revision(key):
      yield key.delete_async()
    self.mock(
        cleanup.instance_templates,
        'get_drained_instance_template_revisions',
        get_drained_instance_template_revisions,
    )
    self.mock(
        cleanup,
        'delete_instance_template_revision',
        delete_instance_template_revision,
    )
    models.InstanceTemplateRevision(
        key=ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
    ).put()
    models.InstanceTemplateRevision(
        key=ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
    ).put()
    models.InstanceTemplateRevision(
        key=ndb.Key(models.InstanceTemplateRevision, 'fake-key-3'),
    ).put()
    models.InstanceTemplateRevision(
        key=ndb.Key(models.InstanceTemplateRevision, 'fake-key-4'),
    ).put()

    cleanup.cleanup_instance_template_revisions(max_concurrent=2)
    self.failIf(ndb.Key(models.InstanceTemplateRevision, 'fake-key-1').get())
    self.failUnless(
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-2').get())
    self.failIf(ndb.Key(models.InstanceTemplateRevision, 'fake-key-3').get())
    self.failIf(ndb.Key(models.InstanceTemplateRevision, 'fake-key-4').get())


class CleanupInstanceTemplatesTest(test_case.TestCase):
  """Tests for cleanup.cleanup_instance_templates."""

  def test_no_entities(self):
    @ndb.tasklet
    def delete_instance_template(*_args, **_kwargs):
      self.fail('delete_instance_template called')
    self.mock(
        cleanup,
        'delete_instance_template',
        delete_instance_template,
    )

    cleanup.cleanup_instance_templates()
    self.failIf(models.InstanceTemplate.query().count())

  def test_deletes(self):
    @ndb.tasklet
    def delete_instance_template(key):
      yield key.delete_async()
    self.mock(
        cleanup,
        'delete_instance_template',
        delete_instance_template,
    )
    models.InstanceTemplate(
        key=ndb.Key(models.InstanceTemplate, 'fake-key-1'),
    ).put()
    models.InstanceTemplate(
        key=ndb.Key(models.InstanceTemplate, 'fake-key-2'),
    ).put()
    models.InstanceTemplate(
        key=ndb.Key(models.InstanceTemplate, 'fake-key-3'),
    ).put()

    cleanup.cleanup_instance_templates(max_concurrent=2)
    self.failIf(ndb.Key(models.InstanceTemplate, 'fake-key-1').get())
    self.failIf(ndb.Key(models.InstanceTemplate, 'fake-key-2').get())
    self.failIf(ndb.Key(models.InstanceTemplate, 'fake-key-3').get())


class DeleteInstanceGroupManagerTest(test_case.TestCase):
  """Tests for cleanup.delete_instance_group_manager."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    key = ndb.Key(models.InstanceGroupManager, 'fake-key')

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failIf(key.get())

  def test_url_specified(self):
    """Ensures nothing happens when the entity still has a URL."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        url='url',
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failUnless(key.get())

  def test_active_instances(self):
    """Ensures nothing happens when there are active Instances."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
        instances=[
            ndb.Key(models.Instance, 'fake-key'),
        ],
    ).put()
    models.InstanceTemplateRevision(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failUnless(key.get())

  def test_parent_doesnt_exist(self):
    """Ensures nothing happens when the parent doesn't exist."""
    key = models.InstanceGroupManager(
        key=instance_group_managers.get_instance_group_manager_key(
            'base-name',
            'revision',
            'zone',
        ),
    ).put()
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failUnless(key.get())

  def test_root_doesnt_exist(self):
    """Ensures nothing happens when the root doesn't exist."""
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

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failUnless(key.get())

  def test_active(self):
    """Ensures nothing happens when the entity is active."""
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
        active=key.parent(),
    ).put()

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failUnless(key.get())

  def test_deletes_drained(self):
    """Ensures a drained entity is deleted."""
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
            ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
            key,
            ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent().parent(),
        active=key.parent(),
    ).put()
    expected_drained = [
        ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
        ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
    ]

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failIf(key.get())
    self.assertItemsEqual(key.parent().get().drained, expected_drained)

  def test_deletes_implicitly_drained(self):
    """Ensures an implicitly drained entity is deleted."""
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
            ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
            key,
            ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent().parent(),
        drained=[
            key.parent(),
        ],
    ).put()
    expected_active = [
        ndb.Key(models.InstanceGroupManager, 'fake-key-1'),
        ndb.Key(models.InstanceGroupManager, 'fake-key-2'),
    ]

    future = cleanup.delete_instance_group_manager(key)
    future.wait()

    self.failIf(key.get())
    self.assertItemsEqual(key.parent().get().active, expected_active)


class DeleteInstanceTemplateTest(test_case.TestCase):
  """Tests for cleanup.delete_instance_template."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    key = ndb.Key(models.InstanceTemplate, 'fake-key')

    future = cleanup.delete_instance_template(key)
    future.wait()

    self.failIf(key.get())

  def test_active_instance_template_revisions(self):
    """Ensures nothing happens when an instance template revision is active."""
    key = models.InstanceTemplate(
        active=ndb.Key(models.InstanceTemplateRevision, 'fake-key'),
    ).put()

    future = cleanup.delete_instance_template(key)
    future.wait()

    self.failUnless(key.get())

  def test_drained_instance_template_revisions(self):
    """Ensures nothing happens when instance template revisions are drained."""
    key = models.InstanceTemplate(
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key'),
        ],
    ).put()

    future = cleanup.delete_instance_template(key)
    future.wait()

    self.failUnless(key.get())

  def test_deletes(self):
    """Ensures the entity is deleted."""
    key = models.InstanceTemplate(
    ).put()

    future = cleanup.delete_instance_template(key)
    future.wait()

    self.failIf(key.get())


class DeleteInstanceTemplateRevisionTest(test_case.TestCase):
  """Tests for cleanup.delete_instance_template_revision."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    key = ndb.Key(models.InstanceTemplateRevision, 'fake-key')

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failIf(key.get())

  def test_url_specified(self):
    """Ensures nothing happens when the entity still has a URL."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        url='url',
    ).put()
    models.InstanceTemplate(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failUnless(key.get())

  def test_active_instance_group_managers(self):
    """Ensures nothing happens when there are active InstanceGroupManagers."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        active=[
            ndb.Key(models.InstanceGroupManager, 'fake-key'),
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failUnless(key.get())

  def test_drained_instance_group_managers(self):
    """Ensures nothing happens when there are drained InstanceGroupManagers."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        drained=[
            ndb.Key(models.InstanceGroupManager, 'fake-key'),
        ],
    ).put()
    models.InstanceTemplate(
        key=key.parent(),
        drained=[
            key,
        ],
    ).put()

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failUnless(key.get())

  def test_parent_doesnt_exist(self):
    """Ensures nothing happens when the parent doesn't exist."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
    ).put()

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failUnless(key.get())

  def test_active(self):
    """Ensures nothing happens when the entity is active."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
    ).put()
    models.InstanceTemplate(
        key=key.parent(),
        active=key,
    ).put()

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failUnless(key.get())

  def test_deletes(self):
    """Ensures the entity is deleted."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
    ).put()
    models.InstanceTemplate(
        key=key.parent(),
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
            key,
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
        ],
    ).put()

    expected_drained = [
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
    ]

    future = cleanup.delete_instance_template_revision(key)
    future.wait()

    self.failIf(key.get())
    self.assertItemsEqual(key.parent().get().drained, expected_drained)


class ExistsTest(test_case.TestCase):
  """Tests for cleanup.exists."""

  def test_exists(self):
    """Ensures an existing entity can be detected."""
    def json_request(*_args, **_kwargs):
      return {}
    self.mock(cleanup.net, 'json_request', json_request)

    self.failUnless(cleanup.exists('instance'))

  def test_not_found(self):
    """Ensures a non-existant entity can be detected."""
    def json_request(*_args, **_kwargs):
      raise net.NotFoundError('404', 404, '404')
    self.mock(cleanup.net, 'json_request', json_request)

    self.failIf(cleanup.exists('instance'))

  def test_error(self):
    """Ensures errors are surfaced."""
    def json_request(*_args, **_kwargs):
      raise net.AuthError('403', 403, '403')
    self.mock(cleanup.net, 'json_request', json_request)

    self.assertRaises(net.AuthError, cleanup.exists, 'instance')


class SetInstanceDeletedTest(test_case.TestCase):
  """Tests for cleanup.set_instance_deleted."""

  def test_entity_not_found(self):
    """Ensures nothing happens when the entity is not found."""
    key = ndb.Key(models.Instance, 'fake-key')

    cleanup.set_instance_deleted(key, False)

    self.failIf(key.get())

  def test_not_drained_or_pending_deletion(self):
    """Ensures nothing happens when the entity isn't drained or pending."""
    key = models.Instance(
      key=instances.get_instance_key(
          'base-name',
          'revision',
          'zone',
          'instance-name',
      ),
      pending_deletion=False,
    ).put()

    cleanup.set_instance_deleted(key, False)

    self.failIf(key.get().deleted)

  def test_drained(self):
    """Ensures the entity is marked as deleted when drained."""
    key = models.Instance(
      key=instances.get_instance_key(
          'base-name',
          'revision',
          'zone',
          'instance-name',
      ),
      pending_deletion=False,
    ).put()

    cleanup.set_instance_deleted(key, True)

    self.failUnless(key.get().deleted)

  def test_pending_deletion(self):
    """Ensures the entity is marked as deleted when pending deletion."""
    key = models.Instance(
      key=instances.get_instance_key(
          'base-name',
          'revision',
          'zone',
          'instance-name',
      ),
      pending_deletion=True,
    ).put()

    cleanup.set_instance_deleted(key, False)

    self.failUnless(key.get().deleted)


if __name__ == '__main__':
  unittest.main()
