# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from components import auth
from components import utils

import config


def isolate_writable():
  """Returns True if current user can write to isolate."""
  full_access = auth.is_group_member(config.settings().auth.full_access_group)
  return full_access or auth.is_admin()


def isolate_readable():
  """Returns True if current user can read from isolate."""
  read_only = config.settings().auth.readonly_access_group
  return auth.is_group_member(read_only) or isolate_writable()


def get_user_type():
  """Returns a string describing the current access control for the user."""
  if auth.is_admin():
    return 'admin'
  if isolate_readable():
    return 'user'


def bootstrap():
  """Adds 127.0.0.1 as a whitelisted IP when testing."""
  if not utils.is_local_dev_server() or auth.is_replica():
    return

  # Allow local bots full access.
  bots = auth.bootstrap_loopback_ips()
  full_access = config.settings().auth.full_access_group
  auth.bootstrap_group(full_access, bots, 'Can read and write from/to Isolate')

  # Add a fake admin for local dev server.
  auth.bootstrap_group(
      auth.ADMIN_GROUP,
      [auth.Identity(auth.IDENTITY_USER, 'test@example.com')],
      'Users that can manage groups')
