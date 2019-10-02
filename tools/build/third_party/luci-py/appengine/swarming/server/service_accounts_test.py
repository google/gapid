#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import random
import sys
import unittest

import test_env
test_env.setup_test_env()

from test_support import test_case

from components import auth
from components import net
from components import utils

from server import service_accounts
from server import task_pack
from server import task_request


class ServiceAccountRegexpTest(test_case.TestCase):
  def test_is_service_account(self):
    self.assertTrue(
        service_accounts.is_service_account('a@proj.iam.gserviceaccount.com'))
    self.assertFalse(
      service_accounts.is_service_account('bot:something'))
    self.assertFalse(
      service_accounts.is_service_account('user:something@something'))
    self.assertFalse(service_accounts.is_service_account(''))


class MockedAuthDB(object):
  token_server_url = 'https://tokens.example.com'


class TestBase(test_case.TestCase):
  def setUp(self):
    super(TestBase, self).setUp()
    self.mock(random, 'randint', lambda _a, _b: 333)
    self.mock(random, 'getrandbits', lambda _: 0x88)
    self.mock(auth, 'get_request_auth_db', MockedAuthDB)

  def mock_json_request(self, expected_url, expected_payload, response):
    calls = []
    def mocked(url, method, payload, headers, scopes):
      calls.append(url)
      self.assertEqual(expected_url, url)
      self.assertEqual('POST', method)
      if expected_payload:
        self.assertEqual(expected_payload, payload)
      self.assertEqual({'Accept': 'application/json; charset=utf-8'},  headers)
      self.assertEqual([net.EMAIL_SCOPE], scopes)
      if isinstance(response, Exception):
        raise response
      return response
    self.mock(net, 'json_request', mocked)
    return calls


class OAuthTokenGrantTest(TestBase):
  def setUp(self):
    super(OAuthTokenGrantTest, self).setUp()
    self.mock(
        auth, 'get_current_identity',
        lambda: auth.Identity.from_bytes('user:end-user@example.com'))

  def test_happy_path(self):
    now = datetime.datetime(2010, 1, 2, 3, 4, 5)
    self.mock_now(now)

    expiry = now + datetime.timedelta(seconds=7200)
    calls = self.mock_json_request(
        expected_url='https://tokens.example.com/prpc/'
            'tokenserver.minter.TokenMinter/MintOAuthTokenGrant',
        expected_payload={
          'auditTags': [
            'swarming:gae_request_id:7357B3D7091D',
            'swarming:service_version:sample-app/v1a',
          ],
          'endUser': 'user:end-user@example.com',
          'serviceAccount': 'service-account@example.com',
          'validityDuration': 7200,
        },
        response={
          'grantToken': 'totally_real_token',
          'serviceVersion': 'token-server-id/ver',
          'expiry': expiry.isoformat() + 'Z',
        })

    # Minting new one.
    tok = service_accounts.get_oauth_token_grant(
        'service-account@example.com', datetime.timedelta(seconds=3600))
    self.assertEqual('totally_real_token', tok)
    self.assertEqual(1, len(calls))

    # Using cached one.
    tok = service_accounts.get_oauth_token_grant(
        'service-account@example.com', datetime.timedelta(seconds=3600))
    self.assertEqual('totally_real_token', tok)
    self.assertEqual(1, len(calls))  # no new calls

    # Minting another one when the cache expires.
    now += datetime.timedelta(seconds=7200)
    self.mock_now(now)

    expiry = now + datetime.timedelta(seconds=7200)
    calls = self.mock_json_request(
        expected_url='https://tokens.example.com/prpc/'
            'tokenserver.minter.TokenMinter/MintOAuthTokenGrant',
        expected_payload={
          'auditTags': [
            'swarming:gae_request_id:7357B3D7091D',
            'swarming:service_version:sample-app/v1a',
          ],
          'endUser': 'user:end-user@example.com',
          'serviceAccount': 'service-account@example.com',
          'validityDuration': 7200,
        },
        response={
          'grantToken': 'another_totally_real_token',
          'serviceVersion': 'token-server-id/ver',
          'expiry': expiry.isoformat() + 'Z',
        })

    tok = service_accounts.get_oauth_token_grant(
        'service-account@example.com', datetime.timedelta(seconds=3600))
    self.assertEqual('another_totally_real_token', tok)
    self.assertEqual(1, len(calls))

  def test_not_allowed_account(self):
    now = datetime.datetime(2010, 1, 2, 3, 4, 5)
    self.mock_now(now)

    self.mock_json_request(
        expected_url='https://tokens.example.com/prpc/'
            'tokenserver.minter.TokenMinter/MintOAuthTokenGrant',
        expected_payload=None,
      response=net.Error('bad', 403, 'Token server error message'))

    with self.assertRaises(service_accounts.PermissionError) as err:
      service_accounts.get_oauth_token_grant(
          'service-account@example.com', datetime.timedelta(seconds=3600))
    self.assertIn('Token server error message', str(err.exception))


class TaskAccountTokenTest(TestBase):
  def make_task_request(
      self, service_account, service_account_token, try_number=1):
    now = utils.utcnow()
    args = {
      'created_ts': now,
      'manual_tags': [u'tag:1'],
      'name': 'Request with %s' % service_account,
      'priority': 50,
      'task_slices': [
        task_request.TaskSlice(
            expiration_secs=60,
            properties=task_request.TaskProperties(
                command=[u'command1'],
                dimensions_data={u'pool': [u'default']},
                execution_timeout_secs=24*60*60)),
      ],
      'user': 'Someone',
    }
    req = task_request.TaskRequest(**args)
    task_request.init_new_request(req, True, task_request.TEMPLATE_AUTO)
    req.key = task_request.new_request_key()
    req.service_account = service_account
    req.service_account_token = service_account_token
    req.put()

    summary_key = task_pack.request_key_to_result_summary_key(req.key)
    run_result_key = task_pack.result_summary_key_to_run_result_key(
        summary_key, try_number)
    return task_pack.pack_run_result_key(run_result_key)

  def test_happy_path(self):
    now = datetime.datetime(2010, 1, 2, 3, 4, 5)
    self.mock_now(now)

    # Initial attempt and a retry.
    for try_number in (1, 2):
      task_id = self.make_task_request(
          service_account='service-account@example.com',
          service_account_token='mocked-oauth-token-grant',
          try_number=try_number)

      expiry = now + datetime.timedelta(seconds=3600)
      self.mock_json_request(
          expected_url='https://tokens.example.com/prpc/'
              'tokenserver.minter.TokenMinter/MintOAuthTokenViaGrant',
          expected_payload={
            'grantToken': 'mocked-oauth-token-grant',
            'oauthScope': ['scope1', 'scope2'],
            'minValidityDuration': 300,
            'auditTags': [
              'swarming:gae_request_id:7357B3D7091D',
              'swarming:service_version:sample-app/v1a',
              'swarming:bot_id:bot-id',
              'swarming:task_id:' + task_id,
              'swarming:task_name:Request with service-account@example.com',
            ],
          },
          response={
            'accessToken': 'totally_real_token',
            'serviceVersion': 'token-server-id/ver',
            'expiry': expiry.isoformat() + 'Z',
          })

      tok = service_accounts.AccessToken(
          'totally_real_token', int(utils.time_time() + 3600))
      self.assertEqual(
          ('service-account@example.com', tok),
          service_accounts.get_task_account_token(
              task_id, 'bot-id', ['scope1', 'scope2']))

  def test_malformed_task_id(self):
    with self.assertRaises(service_accounts.MisconfigurationError):
      service_accounts.get_task_account_token(
          'bad-task-id', 'bot-id', ['scope1', 'scope2'])

  def test_missing_task_id(self):
    with self.assertRaises(service_accounts.MisconfigurationError):
      service_accounts.get_task_account_token(
          '382b353612985111', 'bot-id', ['scope1', 'scope2'])

  def test_task_account_is_bot(self):
    task_id = self.make_task_request(
        service_account='bot',
        service_account_token=None)
    account, tok = service_accounts.get_task_account_token(
        task_id, 'bot-id', ['scope1', 'scope2'])
    self.assertEqual('bot', account)
    self.assertIsNone(tok)

  def test_task_account_is_none(self):
    task_id = self.make_task_request(
        service_account='none',
        service_account_token=None)
    account, tok = service_accounts.get_task_account_token(
        task_id, 'bot-id', ['scope1', 'scope2'])
    self.assertEqual('none', account)
    self.assertIsNone(tok)


class SystemAccountTokenTest(test_case.TestCase):
  def setUp(self):
    super(SystemAccountTokenTest, self).setUp()
    self.mock_now(datetime.datetime(2010, 1, 2, 3, 4, 5))

  def test_none(self):
    self.assertEqual(
        ('none', None),
        service_accounts.get_system_account_token(None, ['scope']))

  def test_bot(self):
    self.assertEqual(
        ('bot', None),
        service_accounts.get_system_account_token('bot', ['scope']))

  def test_token(self):
    calls = []
    def mocked(**kwargs):
      calls.append(kwargs)
      return 'fake-token', utils.time_time() + 3600
    self.mock(auth, 'get_access_token', mocked)

    tok = service_accounts.AccessToken('fake-token', utils.time_time() + 3600)
    self.assertEqual(
        ('bot@example.com', tok),
        service_accounts.get_system_account_token('bot@example.com', ['scope']))

    self.assertEqual([{
        'act_as': 'bot@example.com',
        'min_lifetime_sec': service_accounts.MIN_TOKEN_LIFETIME_SEC,
        'scopes': ['scope'],
    }], calls)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
