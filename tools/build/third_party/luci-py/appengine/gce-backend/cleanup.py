# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for cleaning up GCE Backend."""

import datetime
import logging

from google.appengine.ext import ndb

from components import gce
from components import net
from components import utils

import instance_group_managers
import instance_templates
import instances
import metrics
import models
import utilities


def exists(instance_url):
  """Returns whether the given instance exists or not.

  Args:
    instance_url: URL of the instance.

  Returns:
    True if the instance exists, False otherwise.

  Raises:
    net.Error: If GCE responds with an error.
  """
  try:
    net.json_request(instance_url, method='GET', scopes=gce.AUTH_SCOPES)
    return True
  except net.Error as e:
    if e.status_code == 404:
      return False
    raise


@ndb.transactional
def set_instance_deleted(key, drained):
  """Attempts to set the given Instance as deleted.

  Args:
    key: ndb.Key for a models.Instance entity.
    drained: Whether or not the Instance is being set as deleted
      because it is drained.
  """
  instance = key.get()
  if not instance:
    logging.info('Instance does not exist: %s', key)
    return

  if not drained and not instance.pending_deletion:
    logging.warning('Instance not drained or pending deletion: %s', key)
    return

  if not instance.deleted:
    logging.info('Setting Instance as deleted: %s', key)
    instance.deleted = True
    instance.put()


@ndb.transactional_tasklet
def delete_instance_group_manager(key):
  """Attempts to delete the given InstanceGroupManager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.
  """
  instance_group_manager = yield key.get_async()
  if not instance_group_manager:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return

  if instance_group_manager.url or instance_group_manager.instances:
    return

  instance_template_revision = yield key.parent().get_async()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key.parent())
    return

  instance_template = yield instance_template_revision.key.parent().get_async()
  if not instance_template:
    logging.warning(
        'InstanceTemplate does not exist: %s',
        instance_template_revision.key.parent(),
    )
    return

  # If the InstanceGroupManager is drained, we can delete it now.
  for i, drained_key in enumerate(instance_template_revision.drained):
    if key.id() == drained_key.id():
      instance_template_revision.drained.pop(i)
      yield instance_template_revision.put_async()
      yield key.delete_async()
      return

  # If the InstanceGroupManager is implicitly drained, we can still delete it.
  if instance_template_revision.key in instance_template.drained:
    for i, drained_key in enumerate(instance_template_revision.active):
      if key.id() == drained_key.id():
        instance_template_revision.active.pop(i)
        yield instance_template_revision.put_async()
        yield key.delete_async()


@ndb.transactional_tasklet
def delete_instance_template_revision(key):
  """Attempts to delete the given InstanceTemplateRevision.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.
  """
  instance_template_revision = yield key.get_async()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return

  if instance_template_revision.active or instance_template_revision.drained:
    # All instance group managers, even drained ones, must be deleted first.
    return

  if instance_template_revision.url:
    # GCE instance template must be deleted first.
    return

  instance_template = yield key.parent().get_async()
  if not instance_template:
    logging.warning('InstanceTemplate does not exist: %s', key.parent())
    return

  for i, drained_key in enumerate(instance_template.drained):
    if key.id() == drained_key.id():
      instance_template.drained.pop(i)
      yield instance_template.put_async()
      yield key.delete_async()


@ndb.transactional_tasklet
def delete_instance_template(key):
  """Attempts to delete the given InstanceTemplate.

  Args:
    key: ndb.Key for a models.InstanceTemplate entity.
  """
  instance_template = yield key.get_async()
  if not instance_template:
    logging.warning('InstanceTemplate does not exist: %s', key)
    return

  if instance_template.active or instance_template.drained:
    # All instance template revisions, even drained ones, must be deleted first.
    return

  yield key.delete_async()


def cleanup_instance_group_managers(max_concurrent=50):
  """Deletes drained InstanceGroupManagers.

  Args:
    max_concurrent: Maximum number to delete concurrently.
  """
  utilities.batch_process_async(
      instance_group_managers.get_drained_instance_group_managers(),
      delete_instance_group_manager,
      max_concurrent=max_concurrent,
  )


def cleanup_instance_template_revisions(max_concurrent=50):
  """Deletes drained InstanceTemplateRevisions.

  Args:
    max_concurrent: Maximum number to delete concurrently.
  """
  utilities.batch_process_async(
      instance_templates.get_drained_instance_template_revisions(),
      delete_instance_template_revision,
      max_concurrent=max_concurrent,
  )


def cleanup_instance_templates(max_concurrent=50):
  """Deletes InstanceTemplates.

  Args:
    max_concurrent: Maximum number to delete concurrently.
  """
  utilities.batch_process_async(
      models.InstanceTemplate.query().fetch(keys_only=True),
      delete_instance_template,
      max_concurrent=max_concurrent,
  )


def check_deleted_instance(key):
  """Marks the given Instance as deleted if it refers to a deleted GCE instance.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    return

  if instance.deleted:
    return

  if not instance.pending_deletion:
    logging.warning('Instance not pending deletion: %s', key)
    return

  if not instance.url:
    logging.warning('Instance URL unspecified: %s', key)
    return

  now = utils.utcnow()
  if not exists(instance.url):
    # When the instance isn't found, assume it's deleted.
    if instance.deletion_ts:
      metrics.instance_deletion_time.add(
          (now - instance.deletion_ts).total_seconds(),
          fields={
              'zone': instance.instance_group_manager.id(),
          },
      )
    set_instance_deleted(key, False)
    metrics.send_machine_event('DELETION_SUCCEEDED', instance.hostname)


def schedule_deleted_instance_check():
  """Enqueues tasks to check for deleted instances."""
  for instance in models.Instance.query():
    if instance.pending_deletion and not instance.deleted:
      utilities.enqueue_task('check-deleted-instance', instance.key)


@ndb.transactional
def cleanup_deleted_instance(key):
  """Deletes the given Instance.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    return

  if not instance.deleted:
    logging.warning('Instance not deleted: %s', key)
    return

  logging.info('Deleting Instance entity: %s', key)
  key.delete()
  metrics.send_machine_event('DELETED', instance.hostname)


def schedule_deleted_instance_cleanup():
  """Enqueues tasks to clean up deleted instances."""
  # Only delete entities for instances which were marked as deleted >10 minutes
  # ago. This is because there can be a race condition with the task queue that
  # detects new instances. At the start of the queue it may detect an instance
  # which gets deleted before it finishes, and at the end of the queue it may
  # incorrectly create an entity for that deleted instance. Since task queues
  # can take at most 10 minutes, we can avoid the race condition by deleting
  # only those entities referring to instances which were detected as having
  # been deleted >10 minutes ago. Here we use 20 minutes for safety.
  THRESHOLD = 60 * 20
  now = utils.utcnow()

  for instance in models.Instance.query():
    if instance.deleted and (now - instance.last_updated).seconds > THRESHOLD:
      utilities.enqueue_task('cleanup-deleted-instance', instance.key)


def cleanup_drained_instance(key):
  """Deletes the given drained Instance.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    return

  if instance.deleted:
    return

  if not instance.url:
    logging.warning('Instance URL unspecified: %s', key)
    return

  instance_group_manager = instance.instance_group_manager.get()
  if not instance_group_manager:
    logging.warning(
        'InstanceGroupManager does not exist: %s',
        instance.instance_group_manager,
    )
    return

  instance_template_revision = instance_group_manager.key.parent().get()
  if not instance_template_revision:
    logging.warning(
        'InstanceTemplateRevision does not exist: %s',
        instance_group_manager.key.parent(),
    )
    return

  instance_template = instance_template_revision.key.parent().get()
  if not instance_template:
    logging.warning(
        'InstanceTemplate does not exist: %s',
        instance_template_revision.key.parent(),
    )
    return

  if instance_group_manager.key not in instance_template_revision.drained:
    if instance_template_revision.key not in instance_template.drained:
      logging.warning('Instance is not drained: %s', key)
      return

  now = utils.utcnow()
  if not exists(instance.url):
    # When the instance isn't found, assume it's deleted.
    if instance.deletion_ts:
      metrics.instance_deletion_time.add(
          (now - instance.deletion_ts).total_seconds(),
          fields={
              'zone': instance.instance_group_manager.id(),
          },
      )
    set_instance_deleted(key, True)
    metrics.send_machine_event('DELETION_SUCCEEDED', instance.hostname)


def schedule_drained_instance_cleanup():
  """Enqueues tasks to clean up drained instances."""
  for instance_group_manager_key in (
      instance_group_managers.get_drained_instance_group_managers()):
    instance_group_manager = instance_group_manager_key.get()
    if instance_group_manager:
      for instance_key in instance_group_manager.instances:
        instance = instance_key.get()
        if instance and not instance.cataloged:
          utilities.enqueue_task('cleanup-drained-instance', instance.key)
