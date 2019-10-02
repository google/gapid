#!/usr/bin/python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for instance_templates.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import datastore_utils
from components import gce
from components import net
from test_support import test_case

import instance_group_managers
import instance_templates
import models


class CreateTest(test_case.TestCase):
  """Tests for instance_templates.create."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceTemplateRevision, 'fake-key')

    instance_templates.create(key)
    self.failIf(key.get())

  def test_project_unspecified(self):
    """Ensures nothing happens when project is unspecified."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
    ).put()

    instance_templates.create(key)
    self.failIf(key.get().url)

  def test_url_specified(self):
    """Ensures nothing happens when URL is already specified."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        project='project',
        url='url',
    ).put()
    expected_url = 'url'

    instance_templates.create(key)
    self.assertEqual(key.get().url, expected_url)

  def test_creates(self):
    """Ensures an instance template is created."""
    def create_instance_template(*_args, **_kwargs):
      return {'targetLink': 'url'}
    self.mock(
        instance_templates.gce.Project,
        'create_instance_template',
        create_instance_template,
    )

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        image_name='image',
        image_project='project',
        metadata={
            'key': 'value',
        },
        project='project',
        service_accounts=[
            models.ServiceAccount(
                name='service-account',
                scopes=[
                    'scope',
                ],
            ),
        ],
    ).put()
    expected_url = 'url'

    instance_templates.create(key)
    self.assertEqual(key.get().url, expected_url)

  def test_updates_when_already_created(self):
    """Ensures an instance template is updated when already created."""
    def create_instance_template(*_args, **_kwargs):
      raise net.Error('', 409, '')
    def get_instance_template(*_args, **_kwargs):
      return {'selfLink': 'url'}
    self.mock(
        instance_templates.gce.Project,
        'create_instance_template',
        create_instance_template,
    )
    self.mock(
        instance_templates.gce.Project,
        'get_instance_template',
        get_instance_template,
    )

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        image_name='image',
        project='project',
    ).put()
    expected_url = 'url'

    instance_templates.create(key)
    self.assertEqual(key.get().url, expected_url)

  def test_doesnt_update_when_creation_fails(self):
    """Ensures an instance template is not updated when creation fails."""
    def create_instance_template(*_args, **_kwargs):
      raise net.Error('', 403, '')
    self.mock(
        instance_templates.gce.Project,
        'create_instance_template',
        create_instance_template,
    )

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        image_name='image',
        project='project',
    ).put()

    self.assertRaises(net.Error, instance_templates.create, key)
    self.failIf(key.get().url)


class DeleteTest(test_case.TestCase):
  """Tests for instance_templates.delete."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceTemplateRevision, 'fake-key')

    instance_templates.delete(key)
    self.failIf(key.get())

  def test_deletes(self):
    """Ensures an instance template is deleted."""
    def json_request(url, *_args, **_kwargs):
      return {'targetLink': url}
    self.mock(instance_templates.net, 'json_request', json_request)

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        project='project',
        url='url',
    ).put()

    instance_templates.delete(key)
    self.failIf(key.get().url)

  def test_url_mismatch(self):
    """Ensures nothing happens when the targetLink doesn't match."""
    def json_request(*_args, **_kwargs):
      return {'targetLink': 'mismatch'}
    self.mock(instance_templates.net, 'json_request', json_request)

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        project='project',
        url='url',
    ).put()

    instance_templates.delete(key)
    self.assertEqual(key.get().url, 'url')

  def test_url_unspecified(self):
    """Ensures nothing happens when URL is unspecified."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        project='project',
    ).put()

    instance_templates.delete(key)
    self.failIf(key.get().url)

  def test_url_not_found(self):
    """Ensures URL is updated when the instance template is not found."""
    def json_request(*_args, **_kwargs):
      raise net.Error('', 404, '')
    self.mock(instance_templates.net, 'json_request', json_request)

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        project='project',
        url='url',
    ).put()

    instance_templates.delete(key)
    self.failIf(key.get().url)

  def test_deletion_fails(self):
    """Ensures nothing happens when instance template deletion fails."""
    def json_request(*_args, **_kwargs):
      raise net.Error('', 400, '')
    self.mock(instance_templates.net, 'json_request', json_request)

    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        project='project',
        url='url',
    ).put()
    expected_url = 'url'

    self.assertRaises(net.Error, instance_templates.delete, key)
    self.assertEqual(key.get().url, expected_url)


class GetInstanceTemplateToDeleteTest(test_case.TestCase):
  """Tests for instance_templates.get_instance_template_to_delete."""

  def test_entity_doesnt_exist(self):
    """Ensures no URL when the entity doesn't exist."""
    key = ndb.Key(models.InstanceTemplateRevision, 'fake-key')
    self.failIf(instance_templates.get_instance_template_to_delete(key))

  def test_active_instance_group_managers(self):
    """Ensures no URL when there are active instance group managers."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        active=[
            ndb.Key(models.InstanceGroupManager, 'fake-key'),
        ],
        url='url',
    ).put()

    self.failIf(instance_templates.get_instance_template_to_delete(key))

  def test_drained_instance_group_managers(self):
    """Ensures no URL when there are drained instance group managers."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        drained=[
            ndb.Key(models.InstanceGroupManager, 'fake-key'),
        ],
        url='url',
    ).put()

    self.failIf(instance_templates.get_instance_template_to_delete(key))

  def test_url_unspecified(self):
    """Ensures no URL when URL is unspecified."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
    ).put()

    self.failIf(instance_templates.get_instance_template_to_delete(key))

  def test_returns_url(self):
    """Ensures URL is returned."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        url='url',
    ).put()
    expected_url = 'url'

    self.assertEqual(
        instance_templates.get_instance_template_to_delete(key), expected_url)


class GetDrainedInstanceTemplateRevisions(test_case.TestCase):
  """Tests for instance_templates.get_drained_instance_template_revisions."""

  def test_no_instance_templates(self):
    self.failIf(instance_templates.get_drained_instance_template_revisions())

  def test_one_instance_template_no_drained_revisions(self):
    models.InstanceTemplate().put()
    expected = []

    actual = instance_templates.get_drained_instance_template_revisions()
    self.assertItemsEqual(actual, expected)

  def test_one_instance_template_drained_revisions(self):
    models.InstanceTemplate(
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
        ],
    ).put()
    expected = [
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
    ]

    actual = instance_templates.get_drained_instance_template_revisions()
    self.assertItemsEqual(actual, expected)

  def test_multiple_instance_templates_drained_revisions(self):
    models.InstanceTemplate(
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
        ],
    ).put()
    models.InstanceTemplate(
        active=ndb.Key(models.InstanceTemplateRevision, 'fake-key-3'),
        drained=[
            ndb.Key(models.InstanceTemplateRevision, 'fake-key-4'),
        ],
    ).put()
    expected = [
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-1'),
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-2'),
        ndb.Key(models.InstanceTemplateRevision, 'fake-key-4'),
    ]

    actual = instance_templates.get_drained_instance_template_revisions()
    self.assertItemsEqual(actual, expected)


class ScheduleCreationTest(test_case.TestCase):
  """Tests for instance_templates.schedule_creation."""

  def setUp(self, *args, **kwargs):
    def enqueue_task(_taskqueue, key):
      entity = key.get()
      entity.url = key.urlsafe()
      entity.put()
      return True

    super(ScheduleCreationTest, self).setUp(*args, **kwargs)
    self.mock(instance_templates.utilities, 'enqueue_task', enqueue_task)

  def test_enqueues_task(self):
    """Ensures a task is enqueued."""
    key = instance_templates.get_instance_template_revision_key(
        'base-name', 'revision')
    models.InstanceTemplate(key=key.parent(), active=key).put()
    models.InstanceTemplateRevision(key=key).put()
    expected_url = key.urlsafe()

    instance_templates.schedule_creation()

    self.assertEqual(key.get().url, expected_url)

  def test_enqueues_tasks(self):
    """Ensures tasks are enqueued."""
    # Instance template should be created for key1.
    key1 = instance_templates.get_instance_template_revision_key(
        'base-name-1', 'revision')
    models.InstanceTemplate(key=key1.parent(), active=key1).put()
    models.InstanceTemplateRevision(key=key1).put()
    # key2 refers to an inactive instance template revision. No instance
    # template should be created.
    key2 = instance_templates.get_instance_template_revision_key(
        'base-name-2', 'revision')
    models.InstanceTemplate(key=key2.parent()).put()
    models.InstanceTemplateRevision(key=key2).put()
    # key3 refers to a drained instance template revision. No instance
    # template should be created.
    key3 = instance_templates.get_instance_template_revision_key(
        'base-name-3', 'revision')
    models.InstanceTemplate(key=key3.parent(), drained=[key3]).put()
    models.InstanceTemplateRevision(key=key3).put()
    # key4 refers to an active instance template revision that does not
    # exist. No instance template should be created.
    key4 = instance_templates.get_instance_template_revision_key(
        'base-name-4', 'revision')
    models.InstanceTemplate(key=key4.parent(), active=key4).put()
    # Instance template should be created for key5.
    key5 = instance_templates.get_instance_template_revision_key(
        'base-name-5', 'revision')
    models.InstanceTemplate(key=key5.parent(), active=key5).put()
    models.InstanceTemplateRevision(key=key5).put()

    instance_templates.schedule_creation()

    self.assertEqual(key1.get().url, key1.urlsafe())
    self.failIf(key2.get().url)
    self.failIf(key3.get().url)
    self.failIf(key4.get())
    self.assertEqual(key5.get().url, key5.urlsafe())


class UpdateURLTest(test_case.TestCase):
  """Tests for instance_templates.update_url."""

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceTemplateRevision, 'fake-key')
    instance_templates.update_url(key, 'url')
    self.failIf(key.get())

  def test_url_matches(self):
    """Ensures nothing happens when the URL already matches."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        url='url',
    ).put()

    instance_templates.update_url(key, 'url')

    self.assertEqual(key.get().url, 'url')

  def test_url_mismatch(self):
    """Ensures the URL is updated when it doesn't match."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
        url='old-url',
    ).put()

    instance_templates.update_url(key, 'new-url')

    self.assertEqual(key.get().url, 'new-url')

  def test_url_updated(self):
    """Ensures the URL is updated."""
    key = models.InstanceTemplateRevision(
        key=instance_templates.get_instance_template_revision_key(
            'base-name',
            'revision',
        ),
    ).put()

    instance_templates.update_url(key, 'url')

    self.assertEqual(key.get().url, 'url')

if __name__ == '__main__':
  unittest.main()
