#!/usr/bin/python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for disks.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import net
from test_support import test_case

import disks
import instances
import models


class CreateTest(test_case.TestCase):
  """Tests for disks.create."""

  @staticmethod
  def create_entities(project, snapshot_url, create_parents, disk):
    """Creates a models.Instance entity.

    Args:
      project: The project name to create disks in.
      snapshot_url: The snapshot url to create a disks from.
      create_parents: Whether or not to create parent entities.
      disk: The disk associated with the instance.

    Returns:
      An ndb.Key for a models.InstanceTemplateRevision instance.
    """
    key = models.Instance(id='template revision zone instance', disk=disk).put()
    if create_parents:
      igm_key = instances.get_instance_group_manager_key(key)
      models.InstanceGroupManager(key=igm_key).put()
      itr_key = igm_key.parent()
      models.InstanceTemplateRevision(
          key=itr_key,
          project=project,
          snapshot_url=snapshot_url,
      ).put()
    return key

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    def create_disk(*_args, **_kwargs):
      self.fail('create_disk called')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    key = ndb.Key(models.Instance, 'fake-key')
    disks.create(key)
    self.failIf(key.get())

  def test_disk_attached(self):
    """Ensures nothing happens when the disk is already attached."""
    def create_disk(*_args, **_kwargs):
      self.fail('create_disk called')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    key = self.create_entities('project', 'snapshot', True, 'disk')
    disks.create(key)
    self.assertEqual(key.get().disk, 'disk')

  def test_parent_doesnt_exist(self):
    """Ensures nothing happens when the entity's parent doesn't exist."""
    def create_disk(*_args, **_kwargs):
      self.fail('create_disk called')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    key = self.create_entities('project', 'snapshot', False, None)
    disks.create(key)
    self.failIf(key.get().disk)

  def test_no_project(self):
    """Ensures nothing happens when there is no project."""
    def create_disk(*_args, **_kwargs):
      self.fail('create_disk called')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    key = self.create_entities(None, 'snapshot', True, None)
    disks.create(key)
    self.failIf(key.get().disk)

  def test_no_snapshot(self):
    """Ensures nothing happens when there is no snapshot."""
    def create_disk(*_args, **_kwargs):
      self.fail('create_disk called')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    key = self.create_entities('project', None, True, None)
    disks.create(key)
    self.failIf(key.get().disk)

  def test_create_disk_error(self):
    """Ensures nothing happens when create_disk fails."""
    def create_disk(*_args, **_kwargs):
      raise net.Error('500', 500, '500')
    self.mock(disks.gce.Project, 'create_disk', create_disk)

    key = self.create_entities('project', 'snapshot', True, None)
    with self.assertRaises(net.Error):
      disks.create(key)
    self.failIf(key.get().disk)

  def test_attach_disk_error(self):
    """Ensures nothing happens when attach_disk fails."""
    def create_disk(*_args, **_kwargs):
      return
    def attach_disk(*_args, **_kwargs):
      raise net.Error('500', 500, '500')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)

    key = self.create_entities('project', 'snapshot', True, None)
    with self.assertRaises(net.Error):
      disks.create(key)
    self.failIf(key.get().disk)

  def test_get_instance_error(self):
    """Ensures nothing happens when get_instance fails."""
    def create_disk(*_args, **_kwargs):
      return
    def attach_disk(*_args, **_kwargs):
      return
    def get_instance(*_args, **_kwargs):
      raise net.Error('500', 500, '500')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)
    self.mock(disks.gce.Project, 'get_instance', get_instance)

    key = self.create_entities('project', 'snapshot', True, None)
    with self.assertRaises(net.Error):
      disks.create(key)
    self.failIf(key.get().disk)

  def test_disk_doesnt_exist(self):
    """Ensures nothing happens when the disk doesn't exist."""
    def create_disk(*_args, **_kwargs):
      return
    def attach_disk(*_args, **_kwargs):
      raise net.Error('404', 404, '404')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)

    key = self.create_entities('project', 'snapshot', True, None)
    disks.create(key)
    self.failIf(key.get().disk)

  def test_disk_not_ready(self):
    """Ensures nothing happens when the disk exists but isn't ready."""
    def create_disk(*_args, **_kwargs):
      return
    def attach_disk(*_args, **_kwargs):
      raise net.Error('400', 400, '400')
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)

    key = self.create_entities('project', 'snapshot', True, None)
    disks.create(key)
    self.failIf(key.get().disk)

  def test_disk_not_attached(self):
    """Ensures nothing happens when the disk is ready but not attached yet."""
    def create_disk(*_args, **_kwargs):
      return
    def attach_disk(*_args, **_kwargs):
      return
    def get_instance(*_args, **_kwargs):
      return {}
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)
    self.mock(disks.gce.Project, 'get_instance', get_instance)

    key = self.create_entities('project', 'snapshot', True, None)
    disks.create(key)
    self.failIf(key.get().disk)

  def test_disk_already_created(self):
    """Ensures disk is set when the disk is already created."""
    def create_disk(*_args, **_kwargs):
      raise net.Error('409', 409, '409')
    def attach_disk(*_args, **_kwargs):
      return
    def get_instance(*_args, **_kwargs):
      return {'disks': [{'deviceName': 'instance-disk'}]}
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)
    self.mock(disks.gce.Project, 'get_instance', get_instance)

    key = self.create_entities('project', 'snapshot', True, None)
    disks.create(key)
    self.failUnless(key.get().disk)

  def test_ok(self):
    """Ensures disk is set."""
    def create_disk(*_args, **_kwargs):
      return
    def attach_disk(*_args, **_kwargs):
      return
    def get_instance(*_args, **_kwargs):
      return {'disks': [{'deviceName': 'instance-disk'}]}
    self.mock(disks.gce.Project, 'create_disk', create_disk)
    self.mock(disks.gce.Project, 'attach_disk', attach_disk)
    self.mock(disks.gce.Project, 'get_instance', get_instance)

    key = self.create_entities('project', 'snapshot', True, None)
    disks.create(key)
    self.failUnless(key.get().disk)


if __name__ == '__main__':
  unittest.main()
