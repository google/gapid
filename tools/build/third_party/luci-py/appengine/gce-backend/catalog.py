# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for interacting with the Machine Provider catalog."""

import datetime
import json
import logging

from google.appengine.ext import ndb
from protorpc.remote import protojson

from components import gce
from components import machine_provider
from components import net

import instances
import instance_group_managers
import metrics
import models
import utilities


def get_policies(key, service_account):
  """Returns Machine Provider policies governing the given instance.

  Args:
    key: ndb.Key for a models.Instance entity.
    service_account: Name of the service account the instance will use to
      talk to Machine Provider.
  """
  return {
      'backend_attributes': {
          'key': 'key',
          'value': key.urlsafe(),
      },
      'machine_service_account': service_account,
      'on_reclamation': 'DELETE',
  }


def extract_dimensions(instance, instance_template_revision):
  """Extracts Machine Provider dimensions.

  Args:
    instance: models.Instance entity.
    instance_template_revision: models.InstanceTemplateRevision entity.

  Returns:
    A dict of dimensions.
  """
  if instance_template_revision.dimensions:
    dimensions = json.loads(protojson.encode_message(
        instance_template_revision.dimensions))
  else:
    dimensions = {}

  dimensions['backend'] = 'GCE'

  if instance_template_revision.disk_size_gb:
    dimensions['disk_gb'] = instance_template_revision.disk_size_gb

  # GCE defaults to an HDD type.
  dimensions['disk_type'] = 'HDD'
  if instance_template_revision.disk_type:
    if gce.DISK_TYPES[instance_template_revision.disk_type]['ssd']:
      dimensions['disk_type'] = 'SSD'

  if instance_template_revision.machine_type:
    dimensions['memory_gb'] = gce.machine_type_to_memory(
        instance_template_revision.machine_type)
    dimensions['num_cpus'] = gce.machine_type_to_num_cpus(
        instance_template_revision.machine_type)

  dimensions['hostname'] = instance.hostname

  dimensions['project'] = instance_template_revision.project

  if instance_template_revision.snapshot_name:
    dimensions['snapshot'] = instance_template_revision.snapshot_name

  if instance_template_revision.snapshot_labels:
    dimensions['snapshot_labels'] = (
        instance_template_revision.snapshot_labels[:])

  return dimensions


@ndb.transactional
def set_cataloged(key):
  """Marks the given instance as cataloged.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.cataloged:
    return

  instance.cataloged = True
  instance.put()


def catalog(key):
  """Catalogs the given instance.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.cataloged:
    return

  if instance.pending_deletion:
    logging.warning('Instance pending deletion: %s', key)
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

  if not instance_template_revision.service_accounts:
    logging.warning(
        'InstanceTemplateRevision service account unspecified: %s',
        instance_template_revision.key,
    )
    return

  logging.info('Cataloging Instance: %s', key)
  response = machine_provider.add_machine(
      extract_dimensions(instance, instance_template_revision),
      get_policies(key, instance_template_revision.service_accounts[0].name),
  )

  if response.get('error') and response['error'] != 'HOSTNAME_REUSE':
    # Assume HOSTNAME_REUSE implies a duplicate request.
    logging.warning(
        'Error adding Instance to catalog: %s\nError: %s',
        key,
        response['error'],
    )
    return

  set_cataloged(key)
  metrics.send_machine_event('CATALOGED', instance.hostname)


def schedule_catalog():
  """Enqueues tasks to catalog instances."""
  # Only enqueue tasks for uncataloged instances not pending deletion which
  # are part of active instance group managers which are part of active
  # instance templates.
  for instance_template in models.InstanceTemplate.query():
    if instance_template.active:
      instance_template_revision = instance_template.active.get()
      if instance_template_revision:
        for instance_group_manager_key in instance_template_revision.active:
          instance_group_manager = instance_group_manager_key.get()
          if instance_group_manager:
            for instance_key in instance_group_manager.instances:
              instance = instance_key.get()
              if instance:
                if not instance.cataloged and not instance.pending_deletion:
                  utilities.enqueue_task('catalog-instance', instance.key)


def remove(key):
  """Removes the given instance from the catalog.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.pending_deletion:
    return

  response = machine_provider.delete_machine({'hostname': instance.hostname})
  if response.get('error') and response['error'] != 'ENTRY_NOT_FOUND':
    # Assume ENTRY_NOT_FOUND implies a duplicate request.
    logging.warning(
        'Error removing Instance from catalog: %s\nError: %s',
        key,
        response['error'],
    )
    return

  instances.mark_for_deletion(key)


def schedule_removal():
  """Enqueues tasks to remove drained instances from the catalog."""
  for instance_group_manager_key in (
      instance_group_managers.get_drained_instance_group_managers()):
    instance_group_manager = instance_group_manager_key.get()
    if instance_group_manager:
      for instance_key in instance_group_manager.instances:
        instance = instance_key.get()
        if instance and not instance.pending_deletion:
          utilities.enqueue_task('remove-cataloged-instance', instance.key)


def update_cataloged_instance(key):
  """Updates an Instance based on its state in Machine Provider's catalog.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if not instance.cataloged:
    return

  if instance.pending_deletion:
    return

  try:
    logging.info('Retrieving cataloged instance: %s', key)
    response = machine_provider.retrieve_machine(instance.hostname)
    if response.get('lease_expiration_ts'):
      lease_expiration_ts = datetime.datetime.utcfromtimestamp(
          int(response['lease_expiration_ts']))
      if instance.lease_expiration_ts != lease_expiration_ts:
        instances.add_lease_expiration_ts(key, lease_expiration_ts)
    elif response.get('leased_indefinitely'):
      if not instance.leased_indefinitely:
        instances.set_leased_indefinitely(key)
  except net.NotFoundError:
    logging.info('Instance not found in catalog: %s', key)
    instances.mark_for_deletion(key)


def schedule_cataloged_instance_update():
  """Enqueues tasks to update information about cataloged instances."""
  for instance in models.Instance.query():
    if instance.cataloged and not instance.pending_deletion:
      utilities.enqueue_task('update-cataloged-instance', instance.key)
