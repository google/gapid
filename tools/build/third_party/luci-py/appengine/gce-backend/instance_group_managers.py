# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for operating on instance group managers."""

import collections
import json
import logging
import math

from google.appengine.ext import ndb

from components import gce
from components import net

import instance_templates
import models
import utilities


def get_instance_group_manager_key(base_name, revision, zone):
  """Returns a key for an InstanceTemplateGroupManager.

  Args:
    base_name: Base name for the models.InstanceTemplate.
    revision: Revision string for the models.InstanceTemplateRevision.
    zone: Zone for the models.InstanceGroupManager.

  Returns:
    ndb.Key for a models.InstanceTemplate entity.
  """
  return ndb.Key(
      models.InstanceGroupManager,
      zone,
      parent=instance_templates.get_instance_template_revision_key(
          base_name, revision),
  )


def get_name(instance_group_manager):
  """Returns the name to use when creating an instance group manager.

  Args:
    instance_group_manager: models.InstanceGroupManager.

  Returns:
    A string.
  """
  # <base-name>-<revision>
  return '%s-%s' % (
      instance_group_manager.key.parent().parent().id(),
      instance_group_manager.key.parent().id(),
  )


def get_base_name(instance_group_manager):
  """Returns the base name to use when creating an instance group manager.

  The base name is suffixed randomly by GCE when naming instances.

  Args:
    instance_group_manager: models.InstanceGroupManager.

  Returns:
    A string.
  """
  # <base-name>-<abbreviated-revision>-<zone>
  # TODO(smut): Ensure this name is < 59 characters because the final
  # instance name must be < 64 characters and the instance group manager
  # will add a 5 character random suffix when creating instances.
  return '%s-%s-%s' % (
      instance_group_manager.key.parent().parent().id(),
      instance_group_manager.key.parent().id()[:8],
      instance_group_manager.key.id(),
  )


@ndb.transactional
def set_instances(key, keys):
  """Associates the given Instances with the given InstanceGroupManager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.
    keys: List of ndb.Keys for models.Instance entities.
  """
  instance_group_manager = key.get()
  if not instance_group_manager:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return

  instances = sorted(keys)
  if sorted(instance_group_manager.instances) != instances:
    instance_group_manager.instances = instances
    instance_group_manager.put()


@ndb.transactional
def update_url(key, url):
  """Updates the given InstanceGroupManager with the instance group manager URL.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.
    url: URL string for the instance group manager.
  """
  instance_group_manager = key.get()
  if not instance_group_manager:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return

  if instance_group_manager.url == url:
    return

  logging.warning(
      'Updating URL for InstanceGroupManager: %s\nOld: %s\nNew: %s',
      key,
      instance_group_manager.url,
      url,
  )

  instance_group_manager.url = url
  instance_group_manager.put()


def create(key):
  """Creates an instance group manager from the given InstanceGroupManager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.

  Raises:
    net.Error: HTTP status code is not 200 (created) or 409 (already created).
  """
  instance_group_manager = key.get()
  if not instance_group_manager:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return

  if instance_group_manager.url:
    logging.warning(
        'Instance group manager for InstanceGroupManager already exists: %s',
        key,
    )
    return

  instance_template_revision = key.parent().get()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key.parent())
    return

  if not instance_template_revision.project:
    logging.warning(
        'InstanceTemplateRevision project unspecified: %s', key.parent())
    return

  if not instance_template_revision.url:
    logging.warning(
        'InstanceTemplateRevision URL unspecified: %s', key.parent())
    return

  api = gce.Project(instance_template_revision.project)
  try:
    # Create the instance group manager with 0 instances. The resize cron job
    # will adjust this later.
    result = api.create_instance_group_manager(
        get_name(instance_group_manager),
        instance_template_revision.url,
        0,
        instance_group_manager.key.id(),
        base_name=get_base_name(instance_group_manager),
    )
  except net.Error as e:
    if e.status_code == 409:
      # If the instance template already exists, just record the URL.
      result = api.get_instance_group_manager(
          get_name(instance_group_manager), instance_group_manager.key.id())
      update_url(instance_group_manager.key, result['selfLink'])
      return
    else:
      raise

  update_url(instance_group_manager.key, result['targetLink'])


def schedule_creation():
  """Enqueues tasks to create missing instance group managers."""
  # For each active InstanceGroupManager without a URL, schedule creation
  # of its instance group manager. Since we are outside a transaction the
  # InstanceGroupManager could be out of date and may already have a task
  # scheduled/completed. In either case it doesn't matter since we make
  # creating an instance group manager and updating the URL idempotent.
  for instance_template in models.InstanceTemplate.query():
    if instance_template.active:
      instance_template_revision = instance_template.active.get()
      if instance_template_revision and instance_template_revision.url:
        for instance_group_manager_key in instance_template_revision.active:
          instance_group_manager = instance_group_manager_key.get()
          if instance_group_manager and not instance_group_manager.url:
            utilities.enqueue_task(
                'create-instance-group-manager', instance_group_manager_key)


def get_instance_group_manager_to_delete(key):
  """Returns the URL of the instance group manager to delete.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.

  Returns:
    The URL of the instance group manager to delete, or None if there isn't one.
  """
  instance_group_manager = key.get()
  if not instance_group_manager:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return

  if instance_group_manager.instances:
    logging.warning('InstanceGroupManager has active Instances: %s', key)
    return

  if not instance_group_manager.url:
    logging.warning('InstanceGroupManager URL unspecified: %s', key)
    return

  return instance_group_manager.url


def delete(key):
  """Deletes the instance group manager for the given InstanceGroupManager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.

  Raises:
    net.Error: HTTP status code is not 200 (deleted) or 404 (already deleted).
  """
  url = get_instance_group_manager_to_delete(key)
  if not url:
    return

  try:
    result = net.json_request(url, method='DELETE', scopes=gce.AUTH_SCOPES)
    if result['targetLink'] != url:
      logging.warning(
          'InstanceGroupManager mismatch: %s\nExpected: %s\nFound: %s',
          key,
          url,
          result['targetLink'],
      )
      return
  except net.Error as e:
    if e.status_code != 404:
      # If the instance group manager isn't found, assume it's already deleted.
      raise

  update_url(key, None)


def get_drained_instance_group_managers():
  """Returns drained InstanceGroupManagers.

  Returns:
    A list of ndb.Keys for models.InstanceGroupManager entities.
  """
  keys = []

  for instance_template_revision in models.InstanceTemplateRevision.query():
    for key in instance_template_revision.drained:
      keys.append(key)

  # Also include implicitly drained InstanceGroupManagers, those that are active
  # but are members of drained InstanceTemplateRevisions.
  for instance_template in models.InstanceTemplate.query():
    for instance_template_revision_key in instance_template.drained:
      instance_template_revision = instance_template_revision_key.get()
      if instance_template_revision:
        for key in instance_template_revision.active:
          keys.append(key)

  return keys


def schedule_deletion():
  """Enqueues tasks to delete drained instance group managers."""
  for key in get_drained_instance_group_managers():
    instance_group_manager = key.get()
    if instance_group_manager:
      if instance_group_manager.url and not instance_group_manager.instances:
        utilities.enqueue_task('delete-instance-group-manager', key)


def resize(key):
  """Resizes the given instance group manager.

  Args:
    key: ndb.Key for a models.InstanceGroupManager entity.
  """
  # To avoid a massive resize, impose a limit on how much larger we can
  # resize the instance group. Repeated calls will eventually allow the
  # instance group to reach its target size. Cron timing together with
  # this limit controls the rate at which instances are created.
  RESIZE_LIMIT = 100
  # Ratio of total instances to leased instances.
  THRESHOLD = 1.1

  instance_group_manager = key.get()
  if not instance_group_manager:
    logging.warning('InstanceGroupManager does not exist: %s', key)
    return

  if not instance_group_manager.url:
    logging.warning('InstanceGroupManager URL unspecified: %s', key)
    return

  instance_template_revision = key.parent().get()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return

  if not instance_template_revision.project:
    logging.warning('InstanceTemplateRevision project unspecified: %s', key)
    return

  # Determine how many total instances exist for all other revisions of this
  # InstanceGroupManager. Different revisions will all have the same
  # ancestral InstanceTemplate.
  instance_template_key = instance_template_revision.key.parent()
  other_revision_total_size = 0
  for igm in models.InstanceGroupManager.query(ancestor=instance_template_key):
    # Find InstanceGroupManagers in the same zone, except the one being resized.
    if igm.key.id() == key.id() and igm.key != key:
        logging.info(
            'Found another revision of InstanceGroupManager: %s\nSize: %s',
            igm.key,
            igm.current_size,
        )
        other_revision_total_size += igm.current_size

  # Determine how many total instances for this revision of the
  # InstanceGroupManager have been leased out by the Machine Provider
  leased = 0
  for instance in models.Instance.query(
      models.Instance.instance_group_manager == key):
    if instance.leased:
      leased += 1

  api = gce.Project(instance_template_revision.project)
  response = api.get_instance_group_manager(
      get_name(instance_group_manager), key.id())

  # Find out how many instances are idle (i.e. not currently being created
  # or deleted). This helps avoid doing too many VM actions simultaneously.
  current_size = response.get('currentActions', {}).get('none')
  if current_size is None:
    logging.error('Unexpected response: %s', json.dumps(response, indent=2))
    return

  # Ensure there are at least as many instances as needed, but not more than
  # the total allowed at this time.
  new_target_size = int(min(
      # Minimum size to aim for. At least THRESHOLD times more than the number
      # of instances already leased out, but not less than the minimum
      # configured size. If the THRESHOLD suggests we need a fraction of an
      # instance, we need to provide at least one additional whole instance.
      max(instance_group_manager.minimum_size, math.ceil(leased * THRESHOLD)),
      # Total number of instances for this instance group allowed at this time.
      # Ensures that a config change waits for instances of the old revision to
      # be deleted before bringing up instances of the new revision.
      instance_group_manager.maximum_size - other_revision_total_size,
      # Maximum amount the size is allowed to be increased each iteration.
      current_size + RESIZE_LIMIT,
  ))
  logging.info(
      ('Key: %s\nSize: %s\nOld target: %s\nNew target: %s\nMin: %s\nMax: %s'
       '\nLeased: %s\nOther revisions: %s'),
      key,
      current_size,
      response['targetSize'],
      new_target_size,
      instance_group_manager.minimum_size,
      instance_group_manager.maximum_size,
      leased,
      other_revision_total_size,
  )
  if new_target_size <= min(current_size, response['targetSize']):
    return

  api.resize_managed_instance_group(response['name'], key.id(), new_target_size)


def schedule_resize():
  """Enqueues tasks to resize instance group managers."""
  for instance_template in models.InstanceTemplate.query():
    if instance_template.active:
      instance_template_revision = instance_template.active.get()
      if instance_template_revision:
        for instance_group_manager_key in instance_template_revision.active:
          utilities.enqueue_task(
              'resize-instance-group', instance_group_manager_key)


def count_instances():
  """Counts the number of instances owned by each instance template.

  Returns:
    A dict mapping instance template name to a two element list with counts of
    [active, drained] instances.
  """
  # Aggregate the number of instances owned by each instance group manager
  # created for each instance template.
  totals = collections.defaultdict(lambda: [0, 0])
  drained = set(get_drained_instance_group_managers())
  for igm in models.InstanceGroupManager.query():
    # Get the name of the instance template these instances belong to.
    name = igm.key.parent().parent().id()
    totals[name][int(igm.key in drained)] += len(igm.instances)
  return totals
