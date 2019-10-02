#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import sys
import threading
import time
import unittest

import test_env_bot_code
test_env_bot_code.setup_test_env()

from depot_tools import auto_stub

import remote_client


class TestRemoteClient(auto_stub.TestCase):
  def setUp(self):
    super(TestRemoteClient, self).setUp()
    self.slept = 0
    def sleep_mock(t):
      self.slept += t
    self.mock(time, 'sleep', sleep_mock)

  def test_initialize_success(self):
    headers = {'A': 'a'}
    exp_ts = time.time() + 3600
    c = remote_client.RemoteClientNative(
        'http://localhost:1', lambda: (headers, exp_ts), 'localhost', '/')
    c.initialize(threading.Event())
    self.assertEqual(0, self.slept)
    self.assertTrue(c.uses_auth)
    self.assertEqual(headers, c.get_authentication_headers())

  def test_initialize_retries(self):
    headers = {'A': 'a'}
    exp_ts = time.time() + 3600
    attempt = [0]
    def callback():
      attempt[0] += 1
      if attempt[0] == 10:
        return headers, exp_ts
      raise Exception('fail')
    c = remote_client.RemoteClientNative('http://localhost:1', callback,
                                         'localhost', '/')
    c.initialize(threading.Event())
    self.assertEqual(9*2, self.slept)
    self.assertTrue(c.uses_auth)
    self.assertEqual(headers, c.get_authentication_headers())

  def test_initialize_gives_up(self):
    def callback():
      raise Exception('fail')
    c = remote_client.RemoteClientNative('http://localhost:1', callback,
                                         'localhost', '/')
    with self.assertRaises(remote_client.InitializationError):
      c.initialize(threading.Event())
    self.assertEqual(29*2, self.slept)
    self.assertFalse(c.uses_auth)
    self.assertEqual({}, c.get_authentication_headers())

  def test_get_headers(self):
    today = datetime.datetime(2018, 2, 16, 1, 19, 45, 130574)
    self.mock(remote_client, 'utcnow', lambda: today)

    auth_headers = {'A': 'a'}
    auth_exp_ts = time.time() + 3600

    c = remote_client.RemoteClientNative(
        'http://localhost:1',
        lambda: (auth_headers, auth_exp_ts),
        'localhost',
        '/')
    self.assertTrue(c.uses_auth)

    self.assertEqual({'Cookie': 'GOOGAPPUID=899'}, c.get_headers())
    self.assertEqual({'A': 'a', 'Cookie': 'GOOGAPPUID=899'},
                     c.get_headers(include_auth=True))

  def test_get_authentication_headers(self):
    self.mock(time, 'time', lambda: 100000)
    c = remote_client.RemoteClientNative(
        'http://localhost:1',
        lambda: ({'Now': str(time.time())}, time.time() + 3600),
        'localhost',
        '/')

    # Grab initial headers.
    self.assertEqual({'Now': '100000'}, c.get_authentication_headers())

    # A bit later still using same cached headers.
    self.mock(time, 'time', lambda: 102000)
    self.assertEqual({'Now': '100000'}, c.get_authentication_headers())

    # Close to expiration => refreshed.
    self.mock(time, 'time', lambda: 103500)
    self.assertEqual({'Now': '103500'}, c.get_authentication_headers())

  def test_mint_oauth_token_ok(self):
    fake_resp = {
        'service_account': 'blah@example.com',
        'access_token': 'abc',
        'expiry': 12345,
    }

    c = remote_client.RemoteClientNative('http://localhost:1', None,
                                         'localhost', '/')
    def mocked_call(url_path, data):
      self.assertEqual('/swarming/api/v1/bot/oauth_token', url_path)
      self.assertEqual({
          'account_id': 'account_id',
          'id': 'bot_id',
          'scopes': ['a', 'b'],
          'task_id': 'task_id',
      }, data)
      return fake_resp
    self.mock(c, '_url_read_json', mocked_call)

    resp = c.mint_oauth_token('task_id', 'bot_id', 'account_id', ['a', 'b'])
    self.assertEqual(fake_resp, resp)

  def test_mint_oauth_token_transient_err(self):
    c = remote_client.RemoteClientNative('http://localhost:1', None,
                                         'localhost', '/')
    def mocked_call(*_args, **_kwargs):
      return None  # that's how net.url_read_json indicates HTTP 500 :-/
    self.mock(c, '_url_read_json', mocked_call)
    with self.assertRaises(remote_client.InternalError):
      c.mint_oauth_token('task_id', 'bot_id', 'account_id', ['a', 'b'])

  def test_mint_oauth_token_fatal_err(self):
    c = remote_client.RemoteClientNative('http://localhost:1', None,
                                         'localhost', '/')
    def mocked_call(*_args, **_kwargs):
      return {'error': 'blah'}
    self.mock(c, '_url_read_json', mocked_call)
    with self.assertRaises(remote_client.MintOAuthTokenError):
      c.mint_oauth_token('task_id', 'bot_id', 'account_id', ['a', 'b'])


if __name__ == '__main__':
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
