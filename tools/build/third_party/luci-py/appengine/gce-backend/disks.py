# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for operating on persistent disks."""

import logging

from google.appengine.ext import ndb

from components import gce
from components import net

import instances
import models
import utilities


@ndb.transactional
def set_attached(key, disk):
  """Sets the disk as attached to the given Instance.

  Args:
    key: ndb.Key for a models.Instance entity.
    disk: Name of the disk.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.disk == disk:
    return

  logging.info('Updating disk (%s -> %s)', instance.disk, disk)
  instance.disk = disk
  instance.put()


def create(key):
  """Creates and attaches a disk for the given Instance.

  Args:
    key: ndb.Key for a models.Instance entity.
  """
  instance = key.get()
  if not instance:
    logging.warning('Instance does not exist: %s', key)
    return

  if instance.disk:
    return

  igm_key = instances.get_instance_group_manager_key(key)
  itr_key = igm_key.parent()
  itr = itr_key.get()
  if not itr:
    logging.warning('InstanceTemplateRevision does not exist: %s', itr_key)
    return

  if not itr.project:
    logging.warning('InstanceTemplateRevision project unspecified: %s', itr_key)
    return

  if not itr.snapshot_url:
    logging.warning(
        'InstanceTemplateRevision snapshot unspecified: %s', itr_key)
    return

  name = instances.get_name(key)
  disk = '%s-disk' % name
  api = gce.Project(itr.project)

  # Create the disk from the snapshot.
  # Returns 200 if the operation has started. Returns 409 if the disk has
  # already been created.
  try:
    api.create_disk(disk, itr.snapshot_url, igm_key.id())
  except net.Error as e:
    if e.status_code != 409:
      # 409 means the disk already exists.
      raise

  # Attach the disk to the instance.
  # Returns 200 if the operation has started, an operation already exists, or
  # the disk has already been attached. Returns 404 if the disk doesn't exist
  # and 400 if the disk is still being created.
  try:
    api.attach_disk(name, disk, igm_key.id())
  except net.Error as e:
    if e.status_code in (400, 404):
      # 400 or 404 means this was called too soon after creating the disk.
      return
    raise

  # Check if the disk is attached.
  result = api.get_instance(igm_key.id(), name, fields=['disks/deviceName'])
  for d in result.get('disks', []):
    if d.get('deviceName') == disk:
      set_attached(key, disk)
      return


def _schedule_creation(keys):
  """Enqueues tasks to create disks.

  Args:
    keys: ndb.Key for models.InstanceGroupManager entities.
  """
  for igm_key in keys:
    igm = igm_key.get()
    if igm:
      for key in igm.instances:
        instance = key.get()
        if instance and not instance.disk:
          utilities.enqueue_task('create-disk', key)


def schedule_creation():
  """Enqueues tasks to create disks."""
  for itr in models.InstanceTemplateRevision.query():
    if itr.snapshot_url:
      _schedule_creation(itr.active)
      _schedule_creation(itr.drained)
