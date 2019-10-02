# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""ACL checks for endpoints exposed by auth_service."""

from components import auth
from components.auth.ui import acl


def has_access(identity=None):
  """Returns True if current caller can access groups and other auth data."""
  return acl.has_access(identity)


def is_trusted_service(identity=None):
  """Returns True if caller is in 'auth-trusted-services' group."""
  return auth.is_group_member('auth-trusted-services', identity)
