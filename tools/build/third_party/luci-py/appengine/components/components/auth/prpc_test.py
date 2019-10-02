#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import collections
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.protobuf import empty_pb2

from components import prpc as prpclib
from components import utils
from components.auth import api
from components.auth import check
from components.auth import ipaddr
from components.auth import model
from components.auth import prpc
from components.auth import testing
from components.auth import tokens
from components.auth.proto import delegation_pb2
from test_support import test_case


CapturedState = collections.namedtuple('CapturedState', [
  'current_identity',  # value of get_current_identity().to_bytes()
  'is_superuser',      # value of is_superuser()
  'peer_identity',     # value of get_peer_identity().to_bytes()
  'peer_ip',           # value of get_peer_ip()
  'delegation_token',  # value of get_delegation_token()
])


class MockContext(object):
  def __init__(self, peer, metadata):
    self.code = prpclib.StatusCode.OK
    self.details = None
    self._peer = peer
    self._metdata = metadata

  def invocation_metadata(self):
    return self._metdata

  def peer(self):
    return self._peer

  def set_code(self, code):
    self.code = code

  def set_details(self, details):
    self.details = details


class PrpcAuthTest(testing.TestCase):
  # pylint: disable=unused-argument

  def call(self, peer_id, email, headers=None):
    """Mocks pRPC environment and calls the interceptor.

    Returns:
      CapturedState with info about auth context inside the handler.
      MockContext as it is after the request finishes.
    """
    api.reset_local_state()

    headers = (headers or {}).copy()
    if email:
      headers['Authorization'] = 'Bearer %s' % email
    metadata = [(k.lower(), v) for k, v in sorted(headers.items())]

    if email != 'BROKEN':
      ident = model.Anonymous
      if email:
        ident = model.Identity(model.IDENTITY_USER, email)
      self.mock(api, 'check_oauth_access_token', lambda _: (ident, None))
    else:
      def raise_exc(_):
        raise api.AuthenticationError('OMG, bad token')
      self.mock(api, 'check_oauth_access_token', raise_exc)

    ctx = MockContext(peer_id, metadata)
    call_details = prpclib.HandlerCallDetails('service.Method', metadata)

    state = []
    def continuation(request, context, call_details):
      state.append(CapturedState(
          current_identity=api.get_current_identity().to_bytes(),
          is_superuser=api.is_superuser(),
          peer_identity=api.get_peer_identity().to_bytes(),
          peer_ip=api.get_peer_ip(),
          delegation_token=api.get_delegation_token(),
      ))
      return empty_pb2.Empty()
    prpc.prpc_interceptor(empty_pb2.Empty(), ctx, call_details, continuation)

    self.assertTrue(len(state) <= 1)
    return state[0] if state else None, ctx

  def test_anonymous_ipv4(self):
    state, _ = self.call('ipv4:192.168.1.100', None)
    self.assertEqual(state, CapturedState(
        current_identity='anonymous:anonymous',
        is_superuser=False,
        peer_identity='anonymous:anonymous',
        peer_ip=ipaddr.ip_from_string('192.168.1.100'),
        delegation_token=None,
    ))

  def test_anonymous_ipv6(self):
    state, _ = self.call('ipv6:[::1]', None)
    self.assertEqual(state, CapturedState(
        current_identity='anonymous:anonymous',
        is_superuser=False,
        peer_identity='anonymous:anonymous',
        peer_ip=ipaddr.ip_from_string('::1'),
        delegation_token=None,
    ))

  def test_anonymous_bad_peer_id(self):
    state, ctx = self.call('zzz:zzz', None)
    self.assertIsNone(state)
    self.assertEqual(ctx.code, prpclib.StatusCode.INTERNAL)
    self.assertEqual(
        ctx.details,
        'Could not parse peer IP "zzz:zzz": unrecognized RPC peer ID scheme')

  def test_good_access_token(self):
    state, _ = self.call('ipv4:127.0.0.1', 'a@example.com')
    self.assertEqual(state, CapturedState(
        current_identity='user:a@example.com',
        is_superuser=False,
        peer_identity='user:a@example.com',
        peer_ip=ipaddr.ip_from_string('127.0.0.1'),
        delegation_token=None,
    ))

  def test_bad_acess_token(self):
    state, ctx = self.call('ipv4:127.0.0.1', 'BROKEN')
    self.assertIsNone(state)
    self.assertEqual(ctx.code, prpclib.StatusCode.UNAUTHENTICATED)
    self.assertEqual(ctx.details, 'OMG, bad token')

  def test_ip_whitelisted_bot(self):
    model.bootstrap_ip_whitelist(
        model.bots_ip_whitelist(), ['192.168.1.100/32'])

    state, _ = self.call('ipv4:192.168.1.100', None)
    self.assertEqual(state, CapturedState(
        current_identity='bot:whitelisted-ip',
        is_superuser=False,
        peer_identity='bot:whitelisted-ip',
        peer_ip=ipaddr.ip_from_string('192.168.1.100'),
        delegation_token=None,
    ))

    state, _ = self.call('ipv4:127.0.0.1', None)
    self.assertEqual(state, CapturedState(
        current_identity='anonymous:anonymous',
        is_superuser=False,
        peer_identity='anonymous:anonymous',
        peer_ip=ipaddr.ip_from_string('127.0.0.1'),
        delegation_token=None,
    ))

  def test_ip_whitelist_whitelisted(self):
    model.bootstrap_ip_whitelist('whitelist', ['192.168.1.100/32'])
    model.bootstrap_ip_whitelist_assignment(
        model.Identity(model.IDENTITY_USER, 'a@example.com'), 'whitelist')

    state, _ = self.call('ipv4:192.168.1.100', 'a@example.com')
    self.assertEqual(state, CapturedState(
        current_identity='user:a@example.com',
        is_superuser=False,
        peer_identity='user:a@example.com',
        peer_ip=ipaddr.ip_from_string('192.168.1.100'),
        delegation_token=None,
    ))

  def test_ip_whitelist_not_whitelisted(self):
    model.bootstrap_ip_whitelist('whitelist', ['192.168.1.100/32'])
    model.bootstrap_ip_whitelist_assignment(
        model.Identity(model.IDENTITY_USER, 'a@example.com'), 'whitelist')

    state, ctx = self.call('ipv4:127.0.0.1', 'a@example.com')
    self.assertIsNone(state)
    self.assertEqual(ctx.code, prpclib.StatusCode.PERMISSION_DENIED)
    self.assertEqual(ctx.details, 'IP 127.0.0.1 is not whitelisted')

  def test_delegation_token(self):
    # Grab a fake-signed delegation token.
    subtoken = delegation_pb2.Subtoken(
        delegated_identity='user:delegated@a.com',
        kind=delegation_pb2.Subtoken.BEARER_DELEGATION_TOKEN,
        audience=['*'],
        services=['*'],
        creation_time=int(utils.time_time()),
        validity_duration=3600)
    tok_pb = delegation_pb2.DelegationToken(
      serialized_subtoken=subtoken.SerializeToString(),
      signer_id='user:token-server@example.com',
      signing_key_id='signing-key',
      pkcs1_sha256_sig='fake-signature')
    tok = tokens.base64_encode(tok_pb.SerializeToString())

    # Valid delegation token.
    state, ctx = self.call(
        'ipv4:127.0.0.1', 'peer@a.com', {'X-Delegation-Token-V1': tok})
    self.assertEqual(state, CapturedState(
        current_identity='user:delegated@a.com',
        is_superuser=False,
        peer_identity='user:peer@a.com',
        peer_ip=ipaddr.ip_from_string('127.0.0.1'),
        delegation_token=subtoken,
    ))

    # Invalid delegation token.
    state, ctx = self.call(
        'ipv4:127.0.0.1', 'peer@a.com', {'X-Delegation-Token-V1': tok + 'blah'})
    self.assertIsNone(state)
    self.assertEqual(ctx.code, prpclib.StatusCode.PERMISSION_DENIED)
    self.assertEqual(
        ctx.details, 'Bad delegation token: Bad proto: Truncated message.')

  def test_x_luci_project_works(self):
    self.mock_group(check.LUCI_SERVICES_GROUP, ['user:peer@a.com'])

    # No header -> authenticated as is.
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    state, ctx = self.call('ipv4:127.0.0.1', 'peer@a.com', {})
    self.assertEqual(ctx.code, prpclib.StatusCode.OK)
    self.assertEqual(state.current_identity, 'user:peer@a.com')
    self.assertEqual(state.peer_identity, 'user:peer@a.com')

    # With header, but X-Luci-Project auth is off -> authenticated as is.
    self.mock_config(USE_PROJECT_IDENTITIES=False)
    state, ctx = self.call(
        'ipv4:127.0.0.1', 'peer@a.com', {check.X_LUCI_PROJECT: 'proj-name'})
    self.assertEqual(ctx.code, prpclib.StatusCode.OK)
    self.assertEqual(state.current_identity, 'user:peer@a.com')
    self.assertEqual(state.peer_identity, 'user:peer@a.com')

    # With header and X-Luci-Project auth is on -> authenticated as project.
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    state, ctx = self.call(
        'ipv4:127.0.0.1', 'peer@a.com', {check.X_LUCI_PROJECT: 'proj-name'})
    self.assertEqual(ctx.code, prpclib.StatusCode.OK)
    self.assertEqual(state.current_identity, 'project:proj-name')
    self.assertEqual(state.peer_identity, 'user:peer@a.com')

  def test_x_luci_project_from_unrecognized_service(self):
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    _, ctx = self.call(
        'ipv4:127.0.0.1', 'peer@a.com', {check.X_LUCI_PROJECT: 'proj-name'})
    self.assertEqual(ctx.code, prpclib.StatusCode.UNAUTHENTICATED)
    self.assertEqual(
        ctx.details,
        'Usage of X-Luci-Project is not allowed for user:peer@a.com: not a '
        'member of auth-luci-services group')

  def test_x_luci_project_with_delegation_token(self):
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    _, ctx = self.call(
        'ipv4:127.0.0.1', 'peer@a.com',
        {check.X_LUCI_PROJECT: 'proj-name', 'X-Delegation-Token-V1': 'tok'})
    self.assertEqual(ctx.code, prpclib.StatusCode.UNAUTHENTICATED)
    self.assertEqual(
        ctx.details,
        'Delegation tokens and X-Luci-Project cannot be used together')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
