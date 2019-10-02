#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import logging
import os
import sys
import tempfile
import time
import unittest

import test_env_bot_code
test_env_bot_code.setup_test_env()

from depot_tools import auto_stub
from depot_tools import fix_encoding
from third_party import requests
from utils import file_path
from utils import auth_server

import bot_auth
import remote_client


def global_test_setup():
  # Terminate HTTP server in tests 50x faster. Impacts performance though, so
  # do it only in tests.
  auth_server._HTTPServer.poll_interval = 0.01


def call_rpc(ctx, account_id, scopes):
  r = requests.post(
      url='http://127.0.0.1:%d/rpc/LuciLocalAuthService.GetOAuthToken' %
          ctx['rpc_port'],
      data=json.dumps({
        'account_id': account_id,
        'scopes': scopes,
        'secret': ctx['secret'],
      }),
      headers={'Content-Type': 'application/json'})
  if r.status_code == 200:
    return 200, r.json()
  return r.status_code, r.content


class AuthSystemTest(auto_stub.TestCase):
  def setUp(self):
    super(AuthSystemTest, self).setUp()
    self.tmp_dir = tempfile.mkdtemp(prefix='bot_main')
    self.auth_sys = None

  def tearDown(self):
    try:
      if self.auth_sys:
        self.auth_sys.stop()
    finally:
      file_path.rmtree(self.tmp_dir)
      super(AuthSystemTest, self).tearDown()

  def init_auth_system(self, auth_params):
    self.assertIsNone(self.auth_sys)  # allowed to be called only once per test
    params_path = os.path.join(self.tmp_dir, 'auth_params.json')
    with open(params_path, 'w') as f:
      json.dump(auth_params._asdict(), f)
    self.auth_sys = bot_auth.AuthSystem(params_path)
    return self.auth_sys.start()

  def test_get_bot_headers(self):
    # 'get_bot_headers' returns swarming_http_headers.
    exp = int(time.time() + 3600)
    self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=exp,
        bot_service_account='none',
        system_service_account='none',
        task_service_account='none'))
    self.assertEqual(
        ({'Authorization': 'Bearer bot-own-token'}, exp),
        self.auth_sys.get_bot_headers())

  def test_no_auth(self):
    # Not using service accounts at all -> no LUCI_CONTEXT['local_auth'].
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=0,
        bot_service_account='none',
        system_service_account='none',
        task_service_account='none'))
    self.assertIsNone(local_auth_ctx)

  def test_task_as_bot(self):
    exp = int(time.time() + 3600)

    # An auth system is configured to use only task account, set to bot's own
    # credentials.
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=exp,
        bot_service_account='bot-account@example.com',
        system_service_account='none',
        task_service_account='bot'))
    # Note: default_account_id is omitted when it is None.
    self.assertEqual(['accounts', 'rpc_port', 'secret'], sorted(local_auth_ctx))

    # Only 'task' account is defined (no 'system'). And there's NO default.
    self.assertEqual(
        [{'id': 'task', 'email': 'bot-account@example.com'}],
        local_auth_ctx['accounts'])
    self.assertFalse(local_auth_ctx.get('default_account_id'))

    # Try to use the local RPC service to grab a 'task' token. Should return
    # the token specified by 'swarming_http_headers'.
    code, resp = call_rpc(local_auth_ctx, 'task', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual([u'access_token', u'expiry'], sorted(resp))
    self.assertEqual(u'bot-own-token', resp['access_token'])
    self.assertEqual(exp, resp['expiry'])

    # No 'system' token at all.
    code, _ = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(404, code)

  def test_system_as_bot(self):
    exp = int(time.time() + 3600)

    # An auth system is configured to use only system account, set to bot's own
    # credentials.
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=exp,
        bot_service_account='bot-account@example.com',
        system_service_account='bot',
        task_service_account='none'))
    self.assertEqual(
        ['accounts', 'default_account_id', 'rpc_port', 'secret'],
        sorted(local_auth_ctx))

    # Only 'system' account is defined (no 'task'), and it is default.
    self.assertEqual(
        [{'id': 'system', 'email': 'bot-account@example.com'}],
        local_auth_ctx['accounts'])
    self.assertEqual('system', local_auth_ctx['default_account_id'])

    # Try to use the local RPC service to grab a 'system' token. Should return
    # the token specified by 'swarming_http_headers'.
    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual([u'access_token', u'expiry'], sorted(resp))
    self.assertEqual(u'bot-own-token', resp['access_token'])
    self.assertEqual(exp, resp['expiry'])

    # No 'task' token at all.
    code, _ = call_rpc(local_auth_ctx, 'task', ['A', 'B', 'C'])
    self.assertEqual(404, code)

  def test_system_and_task_as_bot(self):
    exp = int(time.time() + 3600)

    # An auth system configured to use both system and task accounts, both set
    # to bot's own credentials.
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=exp,
        bot_service_account='bot-account@example.com',
        system_service_account='bot',
        task_service_account='bot'))
    self.assertEqual(
        ['accounts', 'default_account_id', 'rpc_port', 'secret'],
        sorted(local_auth_ctx))

    # Both are defined, 'system' is default.
    self.assertEqual(
        [
            {'id': 'system', 'email': 'bot-account@example.com'},
            {'id': 'task', 'email': 'bot-account@example.com'},
        ],
        local_auth_ctx['accounts'])
    self.assertEqual('system', local_auth_ctx.get('default_account_id'))

    # Both 'system' and 'task' tokens work.
    for account_id in ('system', 'task'):
      code, resp = call_rpc(local_auth_ctx, account_id, ['A', 'B', 'C'])
      self.assertEqual(200, code)
      self.assertEqual([u'access_token', u'expiry'], sorted(resp))
      self.assertEqual(u'bot-own-token', resp['access_token'])
      self.assertEqual(exp, resp['expiry'])

  def test_using_bot_without_known_email(self):
    # An auth system configured to use both system and task accounts, both set
    # to bot's own credentials, with email not known.
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={},
        swarming_http_headers_exp=None,
        bot_service_account='none',
        system_service_account='bot',
        task_service_account='bot'))

    # Email is not available, as indicated by '-'.
    self.assertEqual(
        [
            {'id': 'system', 'email': '-'},
            {'id': 'task', 'email': '-'},
        ],
        local_auth_ctx['accounts'])

  @staticmethod
  def mocked_rpc_client(reply):
    class MockedClient(object):
      def __init__(self):
        self.calls = []
      def mint_oauth_token(self, **kwargs):
        self.calls.append(kwargs)
        if isinstance(reply, Exception):
          raise reply
        return reply
    return MockedClient()

  def test_minting_via_rpc_ok(self):
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=int(time.time() + 3600),
        bot_service_account='none',
        system_service_account='abc@example.com',
        task_service_account='none'))

    # Email is set.
    self.assertEqual(
        [{'id': 'system', 'email': 'abc@example.com'}],
        local_auth_ctx['accounts'])

    expiry = int(time.time() + 3600)
    rpc_client = self.mocked_rpc_client({
        'service_account': 'abc@example.com',
        'access_token': 'blah',
        'expiry': expiry,
    })
    self.auth_sys.set_remote_client(rpc_client)

    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual({u'access_token': u'blah', u'expiry': expiry}, resp)
    self.assertEqual([{
        'account_id': 'system',
        'bot_id': 'bot_1',
        'scopes': ('A', 'B', 'C'),
        'task_id': 'task_1',
    }], rpc_client.calls)
    del rpc_client.calls[:]

    # The token is cached.
    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual({u'access_token': u'blah', u'expiry': expiry}, resp)
    self.assertFalse(rpc_client.calls)

  def test_minting_via_rpc_internal_error(self):
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=int(time.time() + 3600),
        bot_service_account='none',
        system_service_account='abc@example.com',
        task_service_account='none'))
    rpc_client = self.mocked_rpc_client(remote_client.InternalError('msg'))
    self.auth_sys.set_remote_client(rpc_client)

    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(500, code)
    self.assertEqual('msg\n', resp)
    self.assertTrue(rpc_client.calls)
    del rpc_client.calls[:]

    # The error is NOT cached, another RPC is made.
    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(500, code)
    self.assertTrue(rpc_client.calls)

  def test_minting_via_rpc_fatal_error(self):
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=int(time.time() + 3600),
        bot_service_account='none',
        system_service_account='abc@example.com',
        task_service_account='none'))
    rpc_client = self.mocked_rpc_client(
        remote_client.MintOAuthTokenError('msg'))
    self.auth_sys.set_remote_client(rpc_client)

    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual({u'error_message': u'msg', u'error_code': 4}, resp)
    self.assertTrue(rpc_client.calls)
    del rpc_client.calls[:]

    # The error is cached, no RPCs are made.
    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual({u'error_message': u'msg', u'error_code': 4}, resp)
    self.assertFalse(rpc_client.calls)

  def test_minting_via_rpc_switching_to_none(self):
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=int(time.time() + 3600),
        bot_service_account='none',
        system_service_account='abc@example.com',
        task_service_account='none'))
    rpc_client = self.mocked_rpc_client({'service_account': 'none'})
    self.auth_sys.set_remote_client(rpc_client)

    # Refused.
    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual({
        u'error_code': 1,
        u'error_message': u"The task has no 'system' account associated with it"
    }, resp)

  def test_minting_via_rpc_switching_to_bot(self):
    expiry = int(time.time() + 3600)
    local_auth_ctx = self.init_auth_system(bot_auth.AuthParams(
        bot_id='bot_1',
        task_id='task_1',
        swarming_http_headers={'Authorization': 'Bearer bot-own-token'},
        swarming_http_headers_exp=expiry,
        bot_service_account='none',
        system_service_account='abc@example.com',
        task_service_account='none'))
    rpc_client = self.mocked_rpc_client({'service_account': 'bot'})
    self.auth_sys.set_remote_client(rpc_client)

    # Got bot token instead.
    code, resp = call_rpc(local_auth_ctx, 'system', ['A', 'B', 'C'])
    self.assertEqual(200, code)
    self.assertEqual(
        {u'access_token': u'bot-own-token', u'expiry': expiry}, resp)


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  global_test_setup()
  unittest.main()
