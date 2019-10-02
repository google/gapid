# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Helpers for mocking auth stuff in unit tests.

Must not be used from main application code, only from unit tests.
"""

from components import utils

from components.auth import api
from components.auth import handler
from components.auth import model


# Will be set as current identity in mock_get_current_identity by default.
DEFAULT_MOCKED_IDENTITY = model.Identity.from_bytes('user:mocked@example.com')


def generate_xsrf_token_for_test(
    ident=DEFAULT_MOCKED_IDENTITY, xsrf_token_data=None):
  """Generates XSRF token to use when sending requests in unit tests."""
  assert utils.is_local_dev_server()
  # See also handler.AuthenticatingHandler.generate_xsrf_token.
  return handler.XSRFToken.generate([ident.to_bytes()], xsrf_token_data)


def mock_get_current_identity(test_case, ident=DEFAULT_MOCKED_IDENTITY):
  """Mocks get_current_identity() to return ident."""
  test_case.mock(api, '_get_current_identity', lambda: ident)


def mock_is_admin(test_case, value=True):
  """Mocks is_admin() to return |value|."""
  # Just mocking 'is_admin' is not enough since many @require() decorators
  # capture its value during class loading, e.g. when used like
  # @auth.require(auth.is_admin). So mock is_group_member(model.ADMIN_GROUP, *)
  # instead (it is called by is_admin).
  orig = api.is_group_member
  def mocked_is_group_member(group, ident):
    if group == model.ADMIN_GROUP:
      return value
    return orig(group, ident)
  test_case.mock(api, 'is_group_member', mocked_is_group_member)


# reset_local_state must be called only in tests, so expose it here.
reset_local_state = api.reset_local_state
