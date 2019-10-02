# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for operating on instance templates."""

import logging

from google.appengine.ext import ndb

from components import gce
from components import net

import models
import utilities


def get_instance_template_key(base_name):
  """Returns a key for an InstanceTemplate.

  Args:
    base_name: Base name for the models.InstanceTemplate.

  Returns:
    ndb.Key for a models.InstanceTemplate entity.
  """
  return ndb.Key(models.InstanceTemplate, base_name)


def get_instance_template_revision_key(base_name, revision):
  """Returns a key for an InstanceTemplateRevision.

  Args:
    base_name: Base name for the models.InstanceTemplate.
    revision: Revision string for the models.InstanceTemplateRevision.

  Returns:
    ndb.Key for a models.InstanceTemplateRevision entity.
  """
  return ndb.Key(
      models.InstanceTemplateRevision,
      revision,
      parent=get_instance_template_key(base_name),
  )


def get_name(instance_template_revision):
  """Returns the name to use when creating an instance template.

  Args:
    instance_template_revision: models.InstanceTemplateRevision.

  Returns:
    A string.
  """
  # <base-name>-<revision>
  return '%s-%s' % (
      instance_template_revision.key.parent().id(),
      instance_template_revision.key.id(),
  )


@ndb.transactional
def update_url(key, url):
  """Updates the given InstanceTemplateRevision with the instance template URL.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.
    url: URL string for the instance template.
  """
  instance_template_revision = key.get()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return

  if instance_template_revision.url == url:
    return

  logging.warning(
      'Updating URL for InstanceTemplateRevision: %s\nOld: %s\nNew: %s',
      key,
      instance_template_revision.url,
      url,
  )

  instance_template_revision.url = url
  instance_template_revision.put()


def create(key):
  """Creates an instance template from the given InstanceTemplateRevision.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.

  Raises:
    net.Error: HTTP status code is not 200 (created) or 409 (already created).
  """
  instance_template_revision = key.get()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return

  if not instance_template_revision.project:
    logging.warning('InstanceTemplateRevision project unspecified: %s', key)
    return

  if instance_template_revision.url:
    logging.info(
        'Instance template for InstanceTemplateRevision already exists: %s',
        key,
    )
    return

  if instance_template_revision.metadata:
    metadata = [{'key': k, 'value': v}
                for k, v in instance_template_revision.metadata.iteritems()]
  else:
    metadata = []

  service_accounts = [
      {'email': service_account.name, 'scopes': service_account.scopes}
      for service_account in instance_template_revision.service_accounts
  ]

  api = gce.Project(instance_template_revision.project)
  try:
    image_project = api.project_id
    if instance_template_revision.image_project:
      image_project = instance_template_revision.image_project
    result = api.create_instance_template(
        get_name(instance_template_revision),
        instance_template_revision.disk_size_gb,
        gce.get_image_url(image_project, instance_template_revision.image_name),
        instance_template_revision.machine_type,
        auto_assign_external_ip=
            instance_template_revision.auto_assign_external_ip,
        disk_type=instance_template_revision.disk_type,
        metadata=metadata,
        min_cpu_platform=instance_template_revision.min_cpu_platform,
        network_url=instance_template_revision.network_url,
        service_accounts=service_accounts,
        tags=instance_template_revision.tags,
    )
  except net.Error as e:
    if e.status_code == 409:
      # If the instance template already exists, just record the URL.
      result = api.get_instance_template(get_name(instance_template_revision))
      update_url(instance_template_revision.key, result['selfLink'])
      return
    else:
      raise

  update_url(instance_template_revision.key, result['targetLink'])


def schedule_creation():
  """Enqueues tasks to create missing instance templates."""
  # For each active InstanceTemplateRevision without a URL, schedule
  # creation of its instance template. Since we are outside a transaction
  # the InstanceTemplateRevision could be out of date and may already have
  # a task scheduled/completed. In either case it doesn't matter since
  # we make creating an instance template and updating the URL idempotent.
  for instance_template in models.InstanceTemplate.query():
    if instance_template.active:
      instance_template_revision = instance_template.active.get()
      if instance_template_revision and not instance_template_revision.url:
        utilities.enqueue_task(
            'create-instance-template', instance_template.active)


def get_instance_template_to_delete(key):
  """Returns the URL of the instance template to delete.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.

  Returns:
    The URL of the instance template to delete, or None if there isn't one.
  """
  instance_template_revision = key.get()
  if not instance_template_revision:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return

  if instance_template_revision.active:
    logging.warning(
        'InstanceTemplateRevision has active InstanceGroupManagers: %s', key)
    return

  if instance_template_revision.drained:
    logging.warning(
        'InstanceTemplateRevision has drained InstanceGroupManagers: %s', key)
    return

  if not instance_template_revision.url:
    logging.warning('InstanceTemplateRevision URL unspecified: %s', key)
    return

  return instance_template_revision.url


def delete(key):
  """Deletes the instance template for the given InstanceTemplateRevision.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.

  Raises:
    net.Error: HTTP status code is not 200 (created) or 404 (already deleted).
  """
  url = get_instance_template_to_delete(key)
  if not url:
    return

  try:
    result = net.json_request(url, method='DELETE', scopes=gce.AUTH_SCOPES)
    if result['targetLink'] != url:
      logging.warning(
          'InstanceTemplateRevision mismatch: %s\nExpected: %s\nFound: %s',
          key,
          url,
          result['targetLink'],
      )
      return
  except net.Error as e:
    if e.status_code != 404:
      # If the instance template isn't found, assume it's already deleted.
      raise

  update_url(key, None)


def get_drained_instance_template_revisions():
  """Returns drained InstanceTemplateRevisions.

  Returns:
    A list of ndb.Keys for models.InstanceTemplateRevision entities.
  """
  keys = []
  for instance_template in models.InstanceTemplate.query():
    for key in instance_template.drained:
      keys.append(key)
  return keys


def schedule_deletion():
  """Enqueues tasks to delete drained instance templates."""
  for key in get_drained_instance_template_revisions():
    instance_template = key.get()
    if instance_template and instance_template.url:
      if not instance_template.active and not instance_template.drained:
        utilities.enqueue_task('delete-instance-template', key)
