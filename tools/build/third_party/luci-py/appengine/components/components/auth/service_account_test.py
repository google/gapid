#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import collections
import datetime
import json
import logging
import os
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.api import urlfetch
from google.appengine.ext import ndb

from components import utils
from components.auth import service_account
from test_support import test_case


EMPTY_SECRET_KEY = service_account.ServiceAccountKey(None, None, None)
FAKE_SECRET_KEY = service_account.ServiceAccountKey('email', 'pkey', 'pkey_id')


MockedResponse = collections.namedtuple('MockedResponse', 'status_code content')


class FakeSigner(object):
  def __init__(self):
    self.claimsets = []

  @property
  def email(self):
    return 'fake@example.com'

  @ndb.tasklet
  def sign_claimset_async(self, claimset):
    self.claimsets.append(claimset)
    raise ndb.Return('fake_jwt')


class GetAccessTokenTest(test_case.TestCase):
  def setUp(self):
    super(GetAccessTokenTest, self).setUp()
    self.log_lines = []
    def mocked_log(msg, *args):
      self.log_lines.append(msg % args)
    self.mock(logging, 'info', mocked_log)
    self.mock(logging, 'error', mocked_log)
    self.mock(logging, 'warning', mocked_log)

  def mock_methods(self):
    calls = []

    @ndb.tasklet
    def via_jwt(*args):
      calls.append(('jwt_based', args))
      raise ndb.Return({'access_token':'token', 'exp_ts': 0})
    self.mock(service_account, '_mint_jwt_based_token_async', via_jwt)

    def via_gae_api(*args):
      calls.append(('gae_api', args))
      return 'token', 0
    self.mock(service_account.app_identity, 'get_access_token', via_gae_api)

    return calls

  def test_memcache_key(self):
    cache_key = service_account._memcache_key(
        method='pkey',
        email='blah@example.com',
        scopes=['1', '2'],
        key_id='abc')
    self.assertEqual(
        '1f7c4e587cd8f6abf972bbcc6437eeba52be7bf69a69974d9a5ef131268a8a4c',
        cache_key)

  ## Verify what token generation method are used based on arguments.

  def test_no_key_uses_gae_api(self):
    """Uses GAE api if secret key is not used."""
    calls = self.mock_methods()
    self.assertEqual(('token', 0), service_account.get_access_token('scope'))
    self.assertEqual([('gae_api', (['scope'],))], calls)

  def test_empty_key_fails(self):
    """If empty key is passed on GAE, dies with error."""
    calls = self.mock_methods()
    with self.assertRaises(service_account.AccessTokenError):
      service_account.get_access_token('scope', EMPTY_SECRET_KEY)
    self.assertFalse(calls)

  def test_good_key_is_used(self):
    """If good key is passed on GAE, invokes JWT based fetch."""
    calls = self.mock_methods()
    self.mock(service_account, '_memcache_key', lambda **_kwargs: 'cache_key')


    self.assertEqual(
        ('token', 0),
        service_account.get_access_token('scope', FAKE_SECRET_KEY))
    self.assertEqual(1, len(calls))

    jwt_based, args = calls[0]
    scopes, signer = args
    self.assertEqual('jwt_based', jwt_based)
    self.assertEqual(['scope'], scopes)
    self.assertTrue(isinstance(signer, service_account._LocalSigner))
    self.assertEqual(FAKE_SECRET_KEY.client_email, signer.email)

  ## Tests for individual token generation methods.

  def test_get_jwt_based_token_memcache(self):
    now = datetime.datetime(2015, 1, 2, 3)


    def randint_mock(start, end):
      return (start + end) / 2
    self.mock(service_account, '_randint', randint_mock)

    # Fake memcache, dev server's one doesn't know about mocked time.
    memcache = {}

    @ndb.tasklet
    def fake_get(key, namespace=None):
      self.assertEqual(service_account._MEMCACHE_NS, namespace)
      if key not in memcache or memcache[key][1] < utils.time_time():
        raise ndb.Return(None)
      raise ndb.Return(memcache[key][0])
    self.mock(service_account, '_memcache_get', fake_get)

    @ndb.tasklet
    def fake_set(key, value, exp, namespace=None):
      self.assertEqual(service_account._MEMCACHE_NS, namespace)
      memcache[key] = (value, exp)
    self.mock(service_account, '_memcache_set', fake_set)

    # Stub calls to real minting method.
    calls = []
    @ndb.tasklet
    def fake_mint_token(*args):
      calls.append(args)
      raise ndb.Return({
        'access_token': 'token@%d' % utils.time_time(),
        'exp_ts': utils.time_time() + 3600,
      })
    self.mock(service_account, '_mint_jwt_based_token_async', fake_mint_token)

    fake_signer = FakeSigner()

    # Cold cache -> mint a new token, put in cache.
    self.mock_now(now, 0)
    self.assertEqual(
        ('token@1420167600', 1420171200.0),
        service_account._get_or_mint_token_async(
            'cache_key',
            300,
            lambda: service_account._mint_jwt_based_token_async(
                ['http://scope'],
                fake_signer))
          .get_result())
    self.assertEqual([(['http://scope'], fake_signer)], calls)
    self.assertEqual(['cache_key'], memcache.keys())
    del calls[:]

    # Uses cached copy while it is valid.
    self.mock_now(now, 3000)
    self.assertEqual(
        ('token@1420167600', 1420171200.0),
        service_account._get_or_mint_token_async(
            'cache_key',
            300,
            lambda: service_account._mint_jwt_based_token_async(
                ['http://scope'],
                fake_signer))
          .get_result())
    self.assertFalse(calls)

    # 5 min before expiration it is considered unusable, and new one is minted.
    self.mock_now(now, 3600 - 5 * 60 + 1)
    self.assertEqual(
        ('token@1420170901', 1420174501.0),
        service_account._get_or_mint_token_async(
            'cache_key',
            300,
            lambda: service_account._mint_jwt_based_token_async(
                ['http://scope'],
                fake_signer))
          .get_result())
    self.assertEqual([(['http://scope'], fake_signer)], calls)

  def test_mint_jwt_based_token(self):
    self.mock_now(datetime.datetime(2015, 1, 2, 3))
    self.mock(os, 'urandom', lambda x: '1'*x)

    calls = []
    @ndb.tasklet
    def mocked_call(**kwargs):
      calls.append(kwargs)
      raise ndb.Return({'access_token': 'token', 'expires_in': 3600})
    self.mock(service_account, '_call_async', mocked_call)

    signer = FakeSigner()
    token = service_account._mint_jwt_based_token_async(
        ['scope1', 'scope2'], signer).get_result()
    self.assertEqual({'access_token':'token', 'exp_ts':1420171200.0}, token)

    self.assertEqual([{
      'aud': 'https://www.googleapis.com/oauth2/v4/token',
      'exp': 1420171195,
      'iat': 1420167595,
      'iss': 'fake@example.com',
      'jti': 'MTExMTExMTExMTExMTExMQ',
      'scope': 'scope1 scope2',
    }], signer.claimsets)

    self.assertEqual([
      {
        'url': 'https://www.googleapis.com/oauth2/v4/token',
        'method': 'POST',
        'headers': {
          'Accept': 'application/json',
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        'payload':
            'grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Ajwt-bearer&'
            'assertion=fake_jwt',
      }], calls)

  def mock_urlfetch(self, calls):
    calls = calls[:]

    @ndb.tasklet
    def urlfetch_mock(
        url, payload, method, headers,
        follow_redirects, deadline, validate_certificate):
      self.assertFalse(follow_redirects)
      self.assertEqual(deadline, 5)
      self.assertTrue(validate_certificate)
      if not calls:
        self.fail('Unexpected call to %s' % url)
      call = calls.pop(0).copy()
      response = call.pop('response')
      self.assertEqual({
        'url': url,
        'payload': payload,
        'method': method,
        'headers': headers,
      }, call)
      if isinstance(response, Exception):
        raise response
      if isinstance(response, MockedResponse):
        raise ndb.Return(response)
      raise ndb.Return(MockedResponse(response[0], json.dumps(response[1])))

    self.mock(service_account, '_urlfetch', urlfetch_mock)
    return calls

  def test_call_async_success(self):
    calls = self.mock_urlfetch([
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': (200, {'abc': 'def'}),
      },
    ])
    response = service_account._call_async(
        url='http://example.com',
        payload='blah',
        method='POST',
        headers={'A': 'a'}).get_result()
    self.assertEqual({'abc': 'def'}, response)
    self.assertFalse(calls)

  def test_call_async_transient_error(self):
    calls = self.mock_urlfetch([
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': (500, {'error': 'zzz'}),
      },
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': urlfetch.Error('blah'),
      },
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': (200, {'abc': 'def'}),
      },
    ])
    response = service_account._call_async(
        url='http://example.com',
        payload='blah',
        method='POST',
        headers={'A': 'a'}).get_result()
    self.assertEqual({'abc': 'def'}, response)
    self.assertFalse(calls)

  def test_call_async_gives_up(self):
    calls = self.mock_urlfetch([
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': (500, {'error': 'zzz'}),
      } for _ in xrange(0, 4)
    ])
    with self.assertRaises(service_account.AccessTokenError) as err:
      service_account._call_async(
        url='http://example.com',
        payload='blah',
        method='POST',
        headers={'A': 'a'}).get_result()
    self.assertTrue(err.exception.transient)
    self.assertFalse(calls)

  def test_call_async_fatal_error(self):
    calls = self.mock_urlfetch([
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': (403, {'error': 'zzz'}),
      },
    ])
    with self.assertRaises(service_account.AccessTokenError) as err:
      service_account._call_async(
        url='http://example.com',
        payload='blah',
        method='POST',
        headers={'A': 'a'}).get_result()
    self.assertFalse(err.exception.transient)
    self.assertFalse(calls)

  def test_call_async_not_json(self):
    calls = self.mock_urlfetch([
      {
        'url': 'http://example.com',
        'payload': 'blah',
        'method': 'POST',
        'headers': {'A': 'a'},
        'response': MockedResponse(200, 'not a json'),
      },
    ])
    with self.assertRaises(service_account.AccessTokenError) as err:
      service_account._call_async(
        url='http://example.com',
        payload='blah',
        method='POST',
        headers={'A': 'a'}).get_result()
    self.assertFalse(err.exception.transient)
    self.assertFalse(calls)

  def test_local_signer(self):
    signer = service_account._LocalSigner(FAKE_SECRET_KEY)

    # We have to fake RSA, since PyCrypto is not available in unit tests.
    rsa_sign_calls = []
    def mocked_rsa_sign(*args):
      rsa_sign_calls.append(args)
      return '\x00signature\x00'
    self.mock(signer, '_rsa_sign', mocked_rsa_sign)

    claimset = {
      'aud': 'https://www.googleapis.com/oauth2/v4/token',
      'exp': 1420171185,
      'iat': 1420167585,
      'iss': 'fake@example.com',
      'scope': 'scope1 scope2',
    }

    jwt = signer.sign_claimset_async(claimset).get_result()

    # Deconstruct JWT.
    chunks = jwt.split('.')
    self.assertEqual(3, len(chunks))
    self.assertEqual({
      u'alg': u'RS256',
      u'kid': u'pkey_id',
      u'typ': u'JWT',
    }, json.loads(service_account._b64_decode(chunks[0])))
    self.assertEqual(
        claimset, json.loads(service_account._b64_decode(chunks[1])))
    self.assertEqual(
        '\x00signature\x00', service_account._b64_decode(chunks[2]))

    # Logged the token, the signature is truncated.
    self.assertEqual(
        'signed_jwt: by=email method=local '
        'hdr={"alg":"RS256","kid":"pkey_id","typ":"JWT"} '
        'claims={"aud":"https://www.googleapis.com/oauth2/v4/token",'
        '"exp":1420171185,"iat":1420167585,"iss":"fake@example.com",'
        '"scope":"scope1 scope2"} '
        'sig_prefix=AHNpZ25hdHVy fp=969934c161c64846dbfcbe9776191a73',
        self.log_lines[0])

  def test_get_access_token_async(self):
    orig_get_access_token_async = service_account.get_access_token_async

    expire_time = '2014-10-02T15:01:23.045123456Z'
    @ndb.tasklet
    def urlfetch_mock(**kwargs):
      class Response(dict):
        def __init__(self, *args, **kwargs):
          super(Response, self).__init__(*args, **kwargs)
          self.status_code = 200
          self.content = json.dumps(self)

      mock_dict = {
        "https://iamcredentials.googleapis.com/v1/projects/-/serviceAccounts/":
          Response({
            "accessToken": 'foobartoken',
            "expireTime": expire_time,
        })
      }
      for url_prefix, response in mock_dict.iteritems():
        if kwargs['url'].find(url_prefix) == 0:
          raise ndb.Return(response)
      raise Exception('url not found in mock: %s' % kwargs['url'])
    self.mock(service_account, '_urlfetch', urlfetch_mock)

    @ndb.tasklet
    def get_access_token_async_mock(
        scopes, service_account_key=None,
        act_as=None, min_lifetime_sec=5*60):
      if service_account_key:
        raise ndb.Return("FAKETOKENFAKETOKEN")
      result = yield orig_get_access_token_async(
          scopes,
          service_account_key,
          act_as,
          min_lifetime_sec
      )
      raise ndb.Return(result)

    # Wrap get_access_token to mock out local signing
    self.mock(service_account,
              'get_access_token_async',
              get_access_token_async_mock)

    # Quick self check on mock of local-signer based flow
    self.assertEqual("FAKETOKENFAKETOKEN",
                     service_account.get_access_token_async(
                         ["a", "b"],
                         service_account_key=FAKE_SECRET_KEY
                     ).get_result())

    res = service_account.get_access_token_async(
        ["c"],
        service_account_key=None,
        act_as="foo@example.com").get_result()
    self.assertEqual(
        (
          'foobartoken',
          int(
              utils.datetime_to_timestamp(
                  utils.parse_rfc3339_datetime(expire_time)) / 1e6)
        ),
        res)

if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
