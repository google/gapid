#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import json
import logging
import sys
import time
import unittest

import test_env_platforms
test_env_platforms.setup_test_env()

from depot_tools import auto_stub

# Disable caching before importing gce.
from utils import tools
tools.cached = lambda func: func

import gce


class TestGCE(auto_stub.TestCase):
  def test_get_zones(self):
    self.mock(gce, 'get_zone', lambda: 'us-central2-a')
    self.assertEqual(
        ['us', 'us-central', 'us-central2', 'us-central2-a'], gce.get_zones())
    self.mock(gce, 'get_zone', lambda: 'europe-west1-b')
    self.assertEqual(
        ['europe', 'europe-west', 'europe-west1', 'europe-west1-b'],
        gce.get_zones())


class TestSignedMetadataToken(auto_stub.TestCase):
  def setUp(self):
    super(TestSignedMetadataToken, self).setUp()
    self.now = 1541465089.0
    self.mock(time, 'time', lambda: self.now)
    self.mock(gce, 'is_gce', lambda: True)

  def test_works(self):
    # JWTs are '<header>.<payload>.<signature>'. We care only about payload.
    jwt = 'unimportant.%s.unimportant' % base64.urlsafe_b64encode(json.dumps({
      'iat': self.now - 600,
      'exp': self.now + 3000,  # 1h after 'iat'
    }))

    metadata_calls = []
    def mocked_raw_metadata_request(path):
      metadata_calls.append(path)
      return jwt
    self.mock(gce, '_raw_metadata_request', mocked_raw_metadata_request)

    tok, exp = gce.signed_metadata_token('https://example.com')
    self.assertEqual(tok, jwt)
    self.assertEqual(exp, self.now + 3600)
    self.assertEqual(metadata_calls, [
      '/computeMetadata/v1/instance/service-accounts/default/'
      'identity?audience=https%3A%2F%2Fexample.com&format=full',
    ])

    # Hitting the cache now.
    tok, exp = gce.signed_metadata_token('https://example.com')
    self.assertEqual(tok, jwt)
    self.assertEqual(exp, self.now + 3600)
    self.assertEqual(len(metadata_calls), 1)  # still same 1 call

    # 1h later cache has expired.
    self.now += 3600
    tok, exp = gce.signed_metadata_token('https://example.com')
    self.assertEqual(tok, jwt)
    self.assertEqual(exp, self.now + 3600)
    self.assertEqual(len(metadata_calls), 2)  # have made a new call


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
