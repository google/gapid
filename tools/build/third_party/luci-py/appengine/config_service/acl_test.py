#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from test_env import future
import test_env
test_env.setup_test_env()

from test_support import test_case
import mock

from components import auth
from components.config.proto import project_config_pb2
from components.config.proto import service_config_pb2

import acl
import projects
import services
import storage


def can_read_config_set(config_set):
  return acl.can_read_config_sets([config_set])[config_set]


def has_project_access(project_id):
  return acl.has_projects_access([project_id])[project_id]


class AclTestCase(test_case.TestCase):
  def setUp(self):
    super(AclTestCase, self).setUp()
    self.mock(auth, 'get_current_identity', mock.Mock())
    auth.get_current_identity.return_value = auth.Anonymous
    self.mock(auth, 'is_group_member', mock.Mock(return_value=False))
    self.mock(
        services, 'get_services_async', mock.Mock(return_value=future([])))

    acl_cfg = service_config_pb2.AclCfg(
        project_access_group='project-admins',
        service_access_group='service-admins',
    )
    self.mock(projects, '_filter_existing', lambda pids: pids)
    self.mock(storage, 'get_self_config_async', lambda *_: future(acl_cfg))

  def test_admin_can_read_all(self):
    self.mock(acl, 'is_admin', mock.Mock(return_value=True))
    self.assertTrue(can_read_config_set('services/swarming'))
    self.assertTrue(can_read_config_set('projects/chromium'))
    self.assertTrue(has_project_access('chromium'))

  def test_has_service_access(self):
    self.assertFalse(can_read_config_set('services/swarming'))

    services.get_services_async.return_value = future([
      service_config_pb2.Service(
          id='swarming', access=['group:swarming-app']),
    ])
    auth.is_group_member.side_effect = lambda g, *_: g == 'swarming-app'

    self.assertTrue(can_read_config_set('services/swarming'))

  def test_service_access_group(self):
    self.assertFalse(can_read_config_set('services/swarming'))

    auth.is_group_member.side_effect = lambda name, *_: name == 'service-admins'
    self.assertTrue(can_read_config_set('services/swarming'))

  def test_has_service_access_no_access(self):
    self.assertFalse(can_read_config_set('services/swarming'))

  def test_has_project_access_group(self):
    self.mock(projects, 'get_metadata_async', mock.Mock(return_value=future({
      'secret': project_config_pb2.ProjectCfg(
          access=['group:googlers', 'a@a.com']),
    })))

    self.assertFalse(can_read_config_set('projects/secret'))

    auth.is_group_member.side_effect = lambda name, *_: name == 'googlers'
    self.assertTrue(can_read_config_set('projects/secret'))

    auth.is_group_member.side_effect = lambda name, *_: name == 'project-admins'
    self.assertTrue(can_read_config_set('projects/secret'))

  def test_has_project_access_identity(self):
    self.mock(projects, 'get_metadata_async', mock.Mock(return_value=future({
      'secret': project_config_pb2.ProjectCfg(
          access=['group:googlers', 'a@a.com']),
    })))

    self.assertFalse(can_read_config_set('projects/secret'))

    auth.get_current_identity.return_value = auth.Identity('user', 'a@a.com')
    self.assertTrue(can_read_config_set('projects/secret'))

  def test_can_read_project_config_no_access(self):
    self.assertFalse(has_project_access('projects/swarming'))
    self.assertFalse(can_read_config_set('projects/swarming/refs/heads/x'))

  def test_malformed_config_set(self):
    with self.assertRaises(ValueError):
      can_read_config_set('invalid config set')


if __name__ == '__main__':
  test_env.main()
