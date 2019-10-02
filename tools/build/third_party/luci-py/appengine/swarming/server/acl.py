# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines access groups.

    +------+
    |admins|
    +------+
       ^
       |
       +------------------------+
       |                        |
    +----------------+    +-------------+
    |privileged_users|    |bot_bootstrap|
    +----------------+    +-------------+
       ^
       |
    +-----+
    |users|
    +-----+


    +-------------+    +--------------+
    |view_all_bots|    |view_all_tasks|
    +-------------+    +--------------+


    +-------------------------+
    |is_ip_whitelisted_machine|
    +-------------------------+


Keep this file synchronized with the documentation at ../proto/config.proto.
"""

from components import auth
from components import utils
from server import config


def _is_admin():
  """Full administrative access."""
  group = config.settings().auth.admins_group
  return auth.is_group_member(group) or auth.is_admin()


def _is_privileged_user():
  """Can edit all bots and tasks."""
  group = config.settings().auth.privileged_users_group
  return auth.is_group_member(group) or _is_admin()


def _is_user():
  group = config.settings().auth.users_group
  return auth.is_group_member(group) or _is_privileged_user()


def _is_view_all_bots():
  group = config.settings().auth.view_all_bots_group
  return auth.is_group_member(group) or _is_privileged_user()


def _is_view_all_tasks():
  group = config.settings().auth.view_all_tasks_group
  return auth.is_group_member(group) or _is_privileged_user()


def _is_bootstrapper():
  """Returns True if current user have access to bot code (for bootstrap)."""
  bot_group = config.settings().auth.bot_bootstrap_group
  return auth.is_group_member(bot_group) or _is_admin()


def _is_project():
  """Returns True if the request is authenticated as coming from "project:...".

  This happens when the request is coming from a trusted LUCI service which
  acts in a context of some LUCI project. We trust such services to authorize
  access to Swarming however they like. Swarming may impose some additional
  checks in pool ACLs though (e.g. make sure a pool is used only by some
  specific projects).
  """
  return auth.get_current_identity().is_project


### Capabilities


def is_ip_whitelisted_machine():
  """Returns True if the call is made from IP whitelisted machine."""
  # TODO(vadimsh): Get rid of this. It's blocked on fixing /bot_code calls in
  # bootstrap code everywhere to use service accounts and switching all Swarming
  # Tasks API calls made from bots to use proper authentication.
  return auth.is_in_ip_whitelist(
      auth.bots_ip_whitelist(), auth.get_peer_ip(), False)


def can_access():
  """Minimally authenticated user."""
  return (
      is_ip_whitelisted_machine() or _is_user() or _is_project() or
      _is_view_all_bots() or _is_view_all_tasks())


#### Config


def can_view_config():
  """Can view the configuration data."""
  return _is_admin()


def can_edit_config():
  """Can edit the configuration data.

  Only super users can edit the configuration data.
  """
  return _is_admin()


#### Bot


def can_create_bot():
  """Can create (bootstrap) a bot."""
  return _is_bootstrapper()


def can_edit_bot():
  """Can terminate a bot.

  Bots can terminate other bots. This may change in the future.
  """
  return is_ip_whitelisted_machine() or _is_privileged_user()


def can_delete_bot():
  """Can delete the existence of a bot.

  Bots can delete other bots. This may change in the future.
  """
  return is_ip_whitelisted_machine() or _is_admin()


def can_view_bot():
  """Can view bot.

  Bots can view other bots. This may change in the future.
  """
  return is_ip_whitelisted_machine() or _is_view_all_bots()


#### Task


def can_create_task():
  """Can create a task.

  Swarming is reentrant, a bot can create a new task as part of a task. This may
  change in the future.
  """
  return is_ip_whitelisted_machine() or _is_user() or _is_project()


def can_schedule_high_priority_tasks():
  """Returns True if the current user can schedule high priority (<20) tasks."""
  return _is_admin()


def can_edit_task(task):
  """Can 'edit' tasks, like cancelling.

  Since bots can create tasks, they can also cancel them. This may change in the
  future.
  """
  return (
      is_ip_whitelisted_machine() or _is_privileged_user() or
      auth.get_current_identity() == task.authenticated)


def can_edit_all_tasks():
  """Can 'edit' a batch of tasks, like cancelling."""
  return _is_admin()


def can_view_task(task):
  """Can view a single task."""
  return (
      is_ip_whitelisted_machine() or _is_view_all_tasks() or
      auth.get_current_identity() == task.authenticated)


def can_view_all_tasks():
  """Can view all tasks."""
  return _is_view_all_tasks()


### Other


def bootstrap_dev_server_acls():
  """Adds localhost to IP whitelist and Swarming groups."""
  assert utils.is_local_dev_server()
  if auth.is_replica():
    return

  bots = auth.bootstrap_loopback_ips()

  auth_settings = config.settings().auth
  admins_group = auth_settings.admins_group
  users_group = auth_settings.users_group
  bot_bootstrap_group = auth_settings.bot_bootstrap_group

  auth.bootstrap_group(users_group, bots, 'Swarming users')
  auth.bootstrap_group(bot_bootstrap_group, bots, 'Bot bootstrap')

  # Add a swarming admin. smoke-test@example.com is used in
  # server_smoke_test.py
  admin = auth.Identity(auth.IDENTITY_USER, 'smoke-test@example.com')
  auth.bootstrap_group(admins_group, [admin], 'Swarming administrators')

  # Add an instance admin (for easier manual testing when running dev server).
  auth.bootstrap_group(
      auth.ADMIN_GROUP,
      [auth.Identity(auth.IDENTITY_USER, 'test@example.com')],
      'Users that can manage groups')
