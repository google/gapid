# Copyright 2019 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for internal components.auth tests."""

import collections
import logging

from components.auth import api
from components.auth import config
from components.auth import delegation
from components.auth import model
from test_support import test_case


# Mocked subset of config tuple returned by config.ensure_configured().
_MockedConfig = collections.namedtuple('_MockedConfig', [
    'USE_PROJECT_IDENTITIES'
])


class TestCase(test_case.TestCase):
  """Test case with a separate auth context and captured logging."""

  # pylint: disable=unused-argument

  def setUp(self):
    super(TestCase, self).setUp()
    api.reset_local_state()

    self.logged_errors = []
    self.mock(
        logging, 'error',
        lambda *args, **kwargs: self.logged_errors.append((args, kwargs)))
    self.logged_warnings = []
    self.mock(
        logging, 'warning',
        lambda *args, **kwargs: self.logged_warnings.append((args, kwargs)))

    self.trusted_signers = {'user:token-server@example.com': self}
    self.mock(delegation, 'get_trusted_signers', lambda: self.trusted_signers)

  # Implements CertificateBundle interface, as used by get_trusted_signers.
  def check_signature(self, blob, key_name, signature):
    return True

  def mock_config(self, **kwargs):
    """Mocks result of config.ensure_configured() call."""
    self.mock(config, 'ensure_configured', lambda: _MockedConfig(**kwargs))

  @staticmethod
  def mock_group(group, members):
    """Creates new group entity in the datastore."""
    members = [
        model.Identity.from_bytes(m) if isinstance(m, basestring) else m
        for m in members
    ]
    model.AuthGroup(key=model.group_key(group), members=members).put()
