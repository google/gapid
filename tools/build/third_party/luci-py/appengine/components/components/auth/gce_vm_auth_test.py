#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import webapp2

from components.auth import api
from components.auth import gce_vm_auth
from components.auth import signature
from components.auth import tokens
from test_support import test_case


class GCEAuthTest(test_case.TestCase):
  CERTS = object()
  TOKEN = 'mocked-token-body'

  def setUp(self):
    super(GCEAuthTest, self).setUp()
    aud_re = gce_vm_auth._audience_re('example.com')
    self.mock(gce_vm_auth, '_allowed_audience_re', lambda: aud_re)
    self.mock(signature, 'get_google_oauth2_certs', lambda: self.CERTS)

  def call(self, payload):
    def mocked_verify_jwt(token, certs):
      self.assertEqual(token, self.TOKEN)
      self.assertIs(certs, self.CERTS)
      if isinstance(payload, Exception):
        raise payload
      return None, payload
    self.mock(tokens, 'verify_jwt', mocked_verify_jwt)

    req = webapp2.Request({})
    if payload is not None:
      req.headers[gce_vm_auth.GCE_VM_TOKEN_HEADER] = self.TOKEN
    return gce_vm_auth.gce_vm_authentication(req)

  def should_fail(self, payload):
    with self.assertRaises(gce_vm_auth.BadTokenError) as err:
      self.call(payload)
    return err.exception.message

  def test_not_applicable(self):
    self.assertEqual(self.call(None), (None, None))

  def test_happy_path(self):
    ident, details = self.call({
      'aud': 'https://example.com',
      'google': {
        'compute_engine': {
          'project_id': 'proj',
          'instance_name': 'inst',
        },
      },
    })
    self.assertEqual(ident.to_bytes(), 'bot:inst@gce.proj')
    self.assertEqual(details, api.new_auth_details(
        gce_instance='inst', gce_project='proj'))

  def test_custom_realm_and_app_version(self):
    ident, details = self.call({
      'aud': 'https://123-dot-example.com',
      'google': {
        'compute_engine': {
          'project_id': 'domain.com:proj',
          'instance_name': 'inst',
        },
      },
    })
    self.assertEqual(ident.to_bytes(), 'bot:inst@gce.proj.domain.com')
    self.assertEqual(details, api.new_auth_details(
        gce_instance='inst', gce_project='domain.com:proj'))

  def test_broken_signature(self):
    self.assertIn(
        'Invalid GCE VM token: boo',
        self.should_fail(tokens.InvalidTokenError('boo')))

  def test_bad_audience(self):
    self.assertIn(
        'Bad audience in GCE VM token',
        self.should_fail({'aud': 'https://not-example.com'}))

  def test_not_full_token(self):
    self.assertIn(
        'No google.compute_engine in the GCE VM token',
        self.should_fail({'aud': 'https://example.com'}))


class AudienceCheckTest(test_case.TestCase):
  def test_works(self):
    def call(swarming_host, aud):
      return bool(gce_vm_auth._audience_re(swarming_host).match(aud))
    self.assertTrue(call('app.example.com', 'https://app.example.com'))
    self.assertFalse(call('app.example.com', 'https://app.example.com/'))
    self.assertFalse(call('app.example.com', 'zzhttps://app.example.com'))
    self.assertFalse(call('app.example.com', 'http://app.example.com'))
    self.assertFalse(call('app.example.com', 'https://app_example.com'))
    self.assertTrue(call('app.example.com', 'https://1-2-dot-app.example.com'))
    self.assertFalse(call('app.example.com', 'https://1\n-dot-app.example.com'))
    self.assertFalse(call('app.example.com', 'https://zzz-app.example.com'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
