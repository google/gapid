#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Smoke test for Cloud Endpoints support in auth component.

It launches app via dev_appserver and queries a bunch of cloud endpoints
methods.
"""

import os
import shutil
import sys
import tempfile
import unittest

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
TEST_APP_DIR = os.path.join(THIS_DIR, 'test_endpoints_app')
CLIENT_DIR = os.path.join(
    os.path.dirname(os.path.dirname(os.path.dirname(THIS_DIR))), 'client')
sys.path.insert(0, CLIENT_DIR)

from third_party.depot_tools import fix_encoding
from tool_support import gae_sdk_utils
from tool_support import local_app


class CloudEndpointsSmokeTest(unittest.TestCase):
  def setUp(self):
    super(CloudEndpointsSmokeTest, self).setUp()
    self.root = tempfile.mkdtemp(prefix='endpoints_smoke_test')
    self.app = local_app.LocalApplication(TEST_APP_DIR, 9700, False, self.root)
    self.app.start()
    self.app.ensure_serving()

  def tearDown(self):
    try:
      self.app.stop()
      shutil.rmtree(self.root)
      if self.has_failed():
        self.app.dump_log()
    finally:
      super(CloudEndpointsSmokeTest, self).tearDown()

  def has_failed(self):
    # pylint: disable=E1101
    return not self._resultForDoCleanups.wasSuccessful()

  def test_smoke(self):
    self.check_who_anonymous()
    self.check_who_authenticated()
    self.check_forbidden()

  def check_who_anonymous(self):
    response = self.app.client.json_request('/_ah/api/testing_service/v1/who')
    self.assertEqual(200, response.http_code)
    self.assertEqual('anonymous:anonymous', response.body.get('identity'))
    self.assertIn(response.body.get('ip'), ('127.0.0.1', '0:0:0:0:0:0:0:1'))

  def check_who_authenticated(self):
    # TODO(vadimsh): Testing this requires interacting with real OAuth2 service
    # to get OAuth2 token. It's doable, but the service account secrets had to
    # be hardcoded into the source code. I'm not sure it's a good idea.
    pass

  def check_forbidden(self):
    response = self.app.client.json_request(
        '/_ah/api/testing_service/v1/forbidden')
    self.assertEqual(403, response.http_code)


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  gae_sdk_utils.setup_gae_env()
  unittest.main()
