#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import datetime
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import webapp2

from components import utils
from components.auth import machine_auth
from components.auth import signature
from test_support import test_case

from components.auth.proto import machine_token_pb2


GOOD_CERTS = {
  'certificates': [
    {
      'key_name': 'signing_key',
      'x509_certificate_pem': 'cert',
    },
  ],
}


class MachineAuthTest(test_case.TestCase):
  def setUp(self):
    super(MachineAuthTest, self).setUp()
    self.mock_now(datetime.datetime(2016, 1, 2, 3, 4, 5))

    self.logs = []
    self.mock(
        machine_auth.logging, 'warning', lambda msg, *_: self.logs.append(msg))

    def is_group_member(group, ident):
      return (group == machine_auth.TOKEN_SERVERS_GROUP and
          ident.to_bytes() == 'user:good-issuer@example.com')
    self.mock(machine_auth.api, 'is_group_member', is_group_member)

    self.mock_check_signature(True)

  def mock_check_signature(self, is_valid=False, exc=None):
    bundle = signature.CertificateBundle(GOOD_CERTS)
    def mocked_check_sig(**_kwargs):
      if exc:
        raise exc  # pylint: disable=raising-bad-type
      return is_valid
    self.mock(bundle, 'check_signature', mocked_check_sig)
    def get(asked):
      self.assertEqual('good-issuer@example.com', asked)
      return bundle
    self.mock(signature, 'get_service_account_certificates', get)

  def has_log(self, msg):
    for m in self.logs:
      if msg in m:
        return True
    return False

  def call(self, body=None, raw_token=None):
    if body:
      env = machine_token_pb2.MachineTokenEnvelope()
      env.token_body = body.SerializeToString()
      env.key_id = 'signing_key'
      env.rsa_sha256 = 'signature'
      raw_token = base64.b64encode(env.SerializeToString())
    req = webapp2.Request({})
    if raw_token:
      req.headers['X-Luci-Machine-Token'] = raw_token
    return machine_auth.machine_authentication(req)

  def good_body(self):
    return machine_token_pb2.MachineTokenBody(
        machine_fqdn='some-machine.host',
        issued_by='good-issuer@example.com',
        issued_at=int(utils.time_time()),
        lifetime=3600,
        ca_id=1,
        cert_sn=3456)

  def test_good_token(self):
    try:
      ident = self.call(body=self.good_body())[0]
      self.assertEqual('bot:some-machine.host', ident.to_bytes())
    except machine_auth.BadTokenError:
      print self.logs
      raise

  def test_no_header(self):
    self.assertIsNone(self.call(raw_token=None)[0])

  def test_not_base64(self):
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(raw_token='not base 64')
    self.assertTrue(self.has_log('Failed to decode base64'))

  def test_bad_envelope(self):
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(raw_token='aaaaaaaa')
    self.assertTrue(self.has_log('Failed to deserialize the token'))

  def test_bad_body(self):
    env = machine_token_pb2.MachineTokenEnvelope()
    env.token_body = 'blah-blah-blah'
    env.key_id = 'signing_key'
    env.rsa_sha256 = 'signature'
    raw_token = base64.b64encode(env.SerializeToString())
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(raw_token=raw_token)
    self.assertTrue(self.has_log('Failed to deserialize the token'))

  def test_bad_issued_by_field(self):
    body = self.good_body()
    body.issued_by='not an email'
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(body)
    self.assertTrue(self.has_log('Bad issued_by field'))

  def test_unknown_issuer(self):
    body = self.good_body()
    body.issued_by='unknown-issuer@example.com'
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(body)
    self.assertTrue(self.has_log('Unknown token issuer'))

  def test_not_valid_yet(self):
    body = self.good_body()
    body.issued_at = int(utils.time_time()) + 600
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(body)
    self.assertTrue(self.has_log('The token is not yet valid'))

  def test_already_expired(self):
    body = self.good_body()
    body.issued_at = int(utils.time_time()) - 3600 - 600
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(body)
    self.assertTrue(self.has_log('The token has expired'))

  def test_bad_signature(self):
    self.mock_check_signature(False)
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(self.good_body())
    self.assertTrue(self.has_log('Bad signature'))

  def test_signature_check_err(self):
    self.mock_check_signature(exc=signature.CertificateError('err', False))
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(self.good_body())
    self.assertTrue(self.has_log('error when checking the signature'))

  def test_bad_bot_id(self):
    body = self.good_body()
    body.machine_fqdn = '#####not-valid#####'
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(body)
    self.assertTrue(self.has_log('Bad machine_fqdn'))

  def test_forbidden_bot_id(self):
    body = self.good_body()
    body.machine_fqdn = 'whitelisted-ip'
    with self.assertRaises(machine_auth.BadTokenError):
      self.call(body)
    self.assertTrue(self.has_log('is forbidden'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
