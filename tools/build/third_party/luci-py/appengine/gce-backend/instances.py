# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for operating on instances."""

import json
import logging

from google.appengine.ext import ndb

from components import gce
from components import net
from components import utils

import instance_group_managers
import metrics
import models
import utilities


def get_instance_key(base_name, revision, zone, instance_name):
  """Returns a key for an Instance.

  Args:
    base_name: Base name for the models.InstanceTemplate.
    revision: Revision string for the models.InstanceTemplateRevision.
    zone: Zone for the models.InstanceGroupManager.
    instance_name: Name of the models.Instance.

  Returns:
    ndb.Key for a models.Instance entity.
  """
  return ndb.Key(
    models.Instance, '%s %s %s %s' % (base_name, revision, zone, instance_name))


def get_instance_group_manager_key(key):
  """Returns a key for the InstanceGroupManager the given Instance belongs to.

  Args:
    key: ndb.Key for a models.Instance.

  Returns:
    ndb.Key for a models.InstanceGroupManager entity.
  """
  return instance_group_managers.get_instance_group_manager_key(
      *key.id().split()[:-1])


def get_name(key):
  """Returns the name of the given Instance.

  Args:
    key: ndb.Key for a models.Instance.

  Returns:
    A string.
  """
  return key.id().split()[-1]


@ndb.transactional
def mark_for_deletion(key):
  """Marks the given instance for deletion.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if not instance.pending_deletion:
    logging.info('Marking Instance for deletion: %s', key)
    instance.lease_expiration_ts = None
    instance.pending_deletion = True
    instance.put()
    metrics.send_machine_event('DELETION_PROPOSED', instance.hostname)


@ndb.transactional
def add_lease_expiration_ts(key, lease_expiration_ts):
  """Adds the lease expiration time to the instance.

  Args:
    key: ndb.Key for a models.Instance entity.
    lease_expiration_ts: datetime.datetime indicating when the lease on this
      machine expires.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.lease_expiration_ts == lease_expiration_ts:
    return

  logging.info(
      'Updating lease_expiration_ts (%s -> %s)',
      instance.lease_expiration_ts,
      lease_expiration_ts,
  )
  instance.lease_expiration_ts = lease_expiration_ts
  instance.put()


@ndb.transactional
def set_leased_indefinitely(key):
  """Sets the instance as leased indefinitely.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.leased_indefinitely:
    return

  logging.info('Setting instance as leased indefinitely')
  instance.leased_indefinitely = True
  instance.put()


def fetch(key):
  """Gets instances created by the given instance group manager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.

  Returns:
    A list of instance URLs.
  """
  instance = key.get()
  if not instance:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return []

  if not instance.url:
    logging.warning('InstanceGroupManager URL unspecified: %s', key)
    return []

  instance_template_revision = key.parent().get()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key.parent())
    return []

  if not instance_template_revision.project:
    logging.warning(
        'InstanceTemplateRevision project unspecified: %s', key.parent())
    return []

  api = gce.Project(instance_template_revision.project)
  result = api.get_instances_in_instance_group(
      instance_group_managers.get_name(instance),
      instance.key.id(),
      max_results=500,
  )
  instance_urls = [i['instance'] for i in result.get('items', [])]
  while result.get('nextPageToken'):
    result = api.get_instances_in_instance_group(
        instance_group_managers.get_name(instance),
        instance.key.id(),
        max_results=500,
        page_token=result['nextPageToken'],
    )
    instance_urls.extend([instance['instance'] for instance in result['items']])

  return instance_urls


@ndb.transactional_tasklet
def _ensure_entity_exists(key, url, instance_group_manager):
  """Ensures an Instance entity exists.

  Args:
    key: ndb.Key for a models.Instance entity.
    url: URL for the instance.
    instance_group_manager: ndb.Key for the models.InstanceGroupManager the
      instance was created from.

  Returns:
    True if an entity was written to the datastore, False otherwise.
  """
  instance = yield key.get_async()
  if instance:
    logging.info('Instance entity already exists: %s', key)
    raise ndb.Return(False)

  logging.info('Creating Instance entity: %s', key)
  yield models.Instance(
      key=key,
      instance_group_manager=instance_group_manager,
      url=url,
  ).put_async()
  raise ndb.Return(True)


@ndb.tasklet
def ensure_entity_exists(key, url, instance_group_manager):
  """Ensures an Instance entity exists.

  Args:
    key: ndb.Key for a models.Instance entity.
    url: URL for the instance.
    instance_group_manager: ndb.Key for the models.InstanceGroupManager the
      instance was created from.
  """
  instance = yield key.get_async()
  if instance:
    return

  put = yield _ensure_entity_exists(key, url, instance_group_manager)
  if put:
    metrics.send_machine_event('CREATED', gce.extract_instance_name(url))


def ensure_entities_exist(key, max_concurrent=50):
  """Ensures Instance entities exist for the given instance group manager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.
    max_concurrent: Maximun number of entities to create concurrently.
  """
  urls = fetch(key)
  if not urls:
    instance_group_managers.set_instances(key, [])
    return

  base_name = key.parent().parent().id()
  revision = key.parent().id()
  zone = key.id()

  keys = {
      url: get_instance_key(
          base_name, revision, zone, gce.extract_instance_name(url))
      for url in urls
  }

  utilities.batch_process_async(
      urls,
      lambda url: ensure_entity_exists(keys[url], url, key),
      max_concurrent=max_concurrent,
  )

  instance_group_managers.set_instances(key, keys.values())


def schedule_fetch():
  """Enqueues tasks to fetch instances."""
  for instance_group_manager in models.InstanceGroupManager.query():
    if instance_group_manager.url:
      utilities.enqueue_task('fetch-instances', instance_group_manager.key)


@ndb.transactional
def set_deletion_time(key, ts):
  """Sets the time the deletion RPC was first sent for this Instance entity.

  Args:
    key: ndb.Key for a models.Instance entity.
    ts: datetime.datetime when the RPC was sent.
  """
  instance = key.get()
  if not instance:
    return

  if instance.deletion_ts:
    return

  instance.deletion_ts = ts
  instance.put()


def _delete(instance_template_revision, instance_group_manager, instance):
  """Deletes the given instance.

  Args:
    instance_template_revision: models.InstanceTemplateRevision.
    instance_group_manager: models.InstanceGroupManager.
    instance: models.Instance
  """
  # We don't check if there are any pending deletion calls because we don't
  # care. We just want the instance to be deleted, so we make repeated calls
  # until the instance is no longer detected. However, we do care how long
  # it takes, so we only write instance.deletion_ts once.

  api = gce.Project(instance_template_revision.project)
  try:
    now = utils.utcnow()
    result = api.delete_instances(
        instance_group_managers.get_name(instance_group_manager),
        instance_group_manager.key.id(),
        [instance.url],
    )
    if result['status'] != 'DONE':
      # This is not the status of the instance deletion, it's the status of
      # scheduling the instance deletions in the managed instance group. If
      # it's not DONE, the deletions won't even be attempted. If it is DONE,
      # the actual deletions may still fail.
      logging.warning(
          'Instance group manager operation failed: %s\n%s',
          instance_group_manager.key,
          json.dumps(result, indent=2),
      )
    else:
      if not instance.deletion_ts:
        set_deletion_time(instance.key, now)
        metrics.send_machine_event('DELETION_SCHEDULED', instance.hostname)
  except net.Error as e:
    if e.status_code == 400:
      if not instance.deletion_ts:
        set_deletion_time(instance.key, now)
    else:
      raise


def delete_pending(key):
  """Deletes the given instance pending deletion.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    return

  if not instance.pending_deletion:
    logging.warning('Instance not pending deletion: %s', key)
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

  if not instance_template_revision.project:
    logging.warning(
        'InstanceTemplateRevision project unspecified: %s',
        instance_template_revision.key,
    )
    return

  _delete(instance_template_revision, instance_group_manager, instance)


def schedule_pending_deletion():
  """Enqueues tasks to delete instances."""
  for instance in models.Instance.query():
    if instance.pending_deletion and not instance.deleted:
      utilities.enqueue_task('delete-instance-pending-deletion', instance.key)
