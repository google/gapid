#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

# from components.auth import api
from components import auth
from components import auth_testing
from components import utils
from server import config
from test_support import test_case

from proto.config import config_pb2

from server import acl
from server import task_request


# Default names of authorization groups.
ADMINS_GROUP = 'administrators'
PRIVILEGED_USERS_GROUP = ADMINS_GROUP
USERS_GROUP = ADMINS_GROUP
BOT_BOOTSTRAP_GROUP = ADMINS_GROUP


class AclTest(test_case.TestCase):
  def setUp(self):
    super(AclTest, self).setUp()
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(self)

    def settings():
      return config_pb2.SettingsCfg(
          auth=config_pb2.AuthSettings(
            admins_group='admins',
            bot_bootstrap_group='bot_bootstrap',
            privileged_users_group='privileged_users',
            users_group='users',
            view_all_bots_group='view_all_bots',
            view_all_tasks_group='view_all_tasks'))
    self.mock(config, 'settings', settings)
    self._task_owned = task_request.TaskRequest(
        authenticated=auth.get_current_identity())
    self._task_other = task_request.TaskRequest(
        authenticated=auth.Identity(auth.IDENTITY_USER, 'larry@localhost'))

  @staticmethod
  def _add_to_group(group):
    auth.bootstrap_group(group, [auth.get_current_identity()])
    auth_testing.reset_local_state()

  def test_nobody(self):
    auth_testing.mock_get_current_identity(self, auth.Anonymous)
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertFalse(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertFalse(acl.can_create_bot())
    self.assertFalse(acl.can_edit_bot())
    self.assertFalse(acl.can_delete_bot())
    self.assertFalse(acl.can_view_bot())
    self.assertFalse(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertFalse(acl.can_edit_task(self._task_owned))
    self.assertFalse(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertFalse(acl.can_view_task(self._task_owned))
    self.assertFalse(acl.can_view_task(self._task_other))
    self.assertFalse(acl.can_view_all_tasks())

  def test_instance_admin(self):
    auth_testing.mock_is_admin(self, True)
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertTrue(acl.can_view_config())
    self.assertTrue(acl.can_edit_config())
    self.assertTrue(acl.can_create_bot())
    self.assertTrue(acl.can_edit_bot())
    self.assertTrue(acl.can_delete_bot())
    self.assertTrue(acl.can_view_bot())
    self.assertTrue(acl.can_create_task())
    self.assertTrue(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertTrue(acl.can_edit_task(self._task_other))
    self.assertTrue(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertTrue(acl.can_view_task(self._task_other))
    self.assertTrue(acl.can_view_all_tasks())

  def test_ip_whitelisted(self):
    self.mock(auth, 'is_in_ip_whitelist', lambda _name, _ip, _warn: True)
    self.assertTrue(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertFalse(acl.can_create_bot())
    self.assertTrue(acl.can_edit_bot())
    self.assertTrue(acl.can_delete_bot())
    self.assertTrue(acl.can_view_bot())
    self.assertTrue(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertTrue(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertTrue(acl.can_view_task(self._task_other))
    self.assertFalse(acl.can_view_all_tasks())

  def test_admins(self):
    self._add_to_group('admins')
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertTrue(acl.can_view_config())
    self.assertTrue(acl.can_edit_config())
    self.assertTrue(acl.can_create_bot())
    self.assertTrue(acl.can_edit_bot())
    self.assertTrue(acl.can_delete_bot())
    self.assertTrue(acl.can_view_bot())
    self.assertTrue(acl.can_create_task())
    self.assertTrue(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertTrue(acl.can_edit_task(self._task_other))
    self.assertTrue(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertTrue(acl.can_view_task(self._task_other))
    self.assertTrue(acl.can_view_all_tasks())

  def test_bot_bootstrap(self):
    self._add_to_group('bot_bootstrap')
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertFalse(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertTrue(acl.can_create_bot())
    self.assertFalse(acl.can_edit_bot())
    self.assertFalse(acl.can_delete_bot())
    self.assertFalse(acl.can_view_bot())
    self.assertFalse(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertFalse(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertFalse(acl.can_view_task(self._task_other))
    self.assertFalse(acl.can_view_all_tasks())

  def test_privileged_users(self):
    self._add_to_group('privileged_users')
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertFalse(acl.can_create_bot())
    self.assertTrue(acl.can_edit_bot())
    self.assertFalse(acl.can_delete_bot())
    self.assertTrue(acl.can_view_bot())
    self.assertTrue(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertTrue(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertTrue(acl.can_view_task(self._task_other))
    self.assertTrue(acl.can_view_all_tasks())

  def test_users(self):
    self._add_to_group('users')
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertFalse(acl.can_create_bot())
    self.assertFalse(acl.can_edit_bot())
    self.assertFalse(acl.can_delete_bot())
    self.assertFalse(acl.can_view_bot())
    self.assertTrue(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertFalse(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertFalse(acl.can_view_task(self._task_other))
    self.assertFalse(acl.can_view_all_tasks())

  def test_view_all_bots(self):
    self._add_to_group('view_all_bots')
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertFalse(acl.can_create_bot())
    self.assertFalse(acl.can_edit_bot())
    self.assertFalse(acl.can_delete_bot())
    self.assertTrue(acl.can_view_bot())
    self.assertFalse(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertFalse(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertFalse(acl.can_view_task(self._task_other))
    self.assertFalse(acl.can_view_all_tasks())

  def test_view_all_tasks(self):
    self._add_to_group('view_all_tasks')
    self.assertFalse(acl.is_ip_whitelisted_machine())
    self.assertTrue(acl.can_access())
    self.assertFalse(acl.can_view_config())
    self.assertFalse(acl.can_edit_config())
    self.assertFalse(acl.can_create_bot())
    self.assertFalse(acl.can_edit_bot())
    self.assertFalse(acl.can_delete_bot())
    self.assertFalse(acl.can_view_bot())
    self.assertFalse(acl.can_create_task())
    self.assertFalse(acl.can_schedule_high_priority_tasks())
    self.assertTrue(acl.can_edit_task(self._task_owned))
    self.assertFalse(acl.can_edit_task(self._task_other))
    self.assertFalse(acl.can_edit_all_tasks())
    self.assertTrue(acl.can_view_task(self._task_owned))
    self.assertTrue(acl.can_view_task(self._task_other))
    self.assertTrue(acl.can_view_all_tasks())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()

