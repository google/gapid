# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Helper functions for working with ACLs."""

import logging

from components import auth
from components.machine_provider import rpc_messages


def is_logged_in():
  """Returns whether the current used is logged in."""
  if auth.get_current_identity().is_anonymous:
    logging.info('User is not logged in')
    return False
  return True


def is_catalog_admin():
  """Returns whether the current user is a catalog administrator."""
  if auth.is_group_member('machine-provider-catalog-administrators'):
    logging.info('User is Catalog administrator')
    return True
  return False


def get_current_backend():
  """Returns the backend associated with the current user.

  Returns:
    An rpc_messages.Backend instance representing the current user, or None if
    the current user is not a recognized backend.
  """
  for backend in rpc_messages.Backend:
    backend_group = 'machine-provider-%s-backend' % backend.name.lower()
    if auth.is_group_member(backend_group):
      logging.info('User is %s backend', backend.name)
      return backend
  logging.info('User is not a recognized backend service')


def is_backend_service():
  """Returns whether the current user is a recognized backend."""
  return get_current_backend() is not None


def is_backend_service_or_catalog_admin():
  return is_backend_service() or is_catalog_admin()


def can_issue_lease_requests():
  """Returns whether the current user may issue lease requests."""
  return auth.is_group_member('machine-provider-users')


def can_view_catalog():
  """Returns whether the current user can view the catalog."""
  if is_catalog_admin():
    return True
  return auth.is_group_member('machine-provider-catalog-viewers')
