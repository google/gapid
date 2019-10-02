# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Generation of OAuth2 token for a service account.

Supports three ways to generate OAuth2 tokens:
  * app_identity.get_access_token(...) to use native GAE service account.
  * OAuth flow with JWT token, for @*.iam.gserviceaccount.com service
    accounts (the one with a private key).
  * Acting as another service account (via signJwt IAM RPC).
"""

import base64
import collections
import hashlib
import json
import logging
import os
import random
import urllib

from google.appengine.api import app_identity
from google.appengine.api import urlfetch
from google.appengine.ext import ndb
from google.appengine.runtime import apiproxy_errors

from components import utils


# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'get_access_token',
  'get_access_token_async',
  'AccessTokenError',
  'ServiceAccountKey',
]

# Information about @*.iam.gserviceaccount.com. Field values can be extracted
# from corresponding fields in JSON file produced by "Generate new JSON key"
# button in "Credentials" section of any Cloud Console project.
ServiceAccountKey = collections.namedtuple('ServiceAccountKey', [
  # Service account email.
  'client_email',
  # Service account PEM encoded private key.
  'private_key',
  # Service account key fingerprint, an unique identifier of this key.
  'private_key_id',
])


class AccessTokenError(Exception):
  """Raised by get_access_token() on fatal or transient errors."""

  def __init__(self, msg, transient=False):
    super(AccessTokenError, self).__init__(msg)
    self.transient = transient


# Do not log AccessTokenError exception raised from a tasklet.
ndb.add_flow_exception(AccessTokenError)

@ndb.tasklet
def get_access_token_async(
    scopes, service_account_key=None, act_as=None, min_lifetime_sec=5*60):
  """Returns an OAuth2 access token for a service account.

  If 'service_account_key' is specified, will use it to generate access token
  for corresponding @*iam.gserviceaccount.com account. Otherwise will invoke
  app_identity.get_access_token(...) to use app's @appspot.gserviceaccount.com
  account.

  If 'act_as' is specified, will return an access token for this account with
  given scopes, generating it through a call to IAM:generateAccessToken, using
  IAM-scoped access token of a primary service account (an appspot one, or the
  one specified via 'service_account_key'). In this case the primary service
  account should have 'serviceAccountTokenCreator' role in the service account
  it acts as.

  See https://cloud.google.com/iam/docs/service-accounts.

  If using 'act_as' or 'service_account_key', the returned token will be valid
  for at least approximately 'min_lifetime_sec' (5 min by default), but possibly
  longer (up to 1h). If both 'act_as' and 'service_account_key' are None,
  'min_lifetime_sec' is ignored and the returned token should be assumed
  short-lived (<5 min).

  Args:
    scopes: the requested API scope string, or a list of strings.
    service_account_key: optional instance of ServiceAccountKey.
    act_as: email of an account to impersonate.
    min_lifetime_sec: desired minimal lifetime of the produced token.

  Returns:
    Tuple (access token, expiration time in seconds since the epoch).

  Raises:
    AccessTokenError on errors.
  """
  # Limit min_lifetime_sec, since requesting very long-lived tokens reduces
  # efficiency of the cache (we need to constantly update it to keep tokens
  # fresh).
  if min_lifetime_sec <= 0 or min_lifetime_sec > 30 * 60:
    raise ValueError(
        '"min_lifetime_sec" should be in range (0; 1800], actual: %d'
        % min_lifetime_sec)

  # Accept a single string to mimic app_identity.get_access_token behavior.
  if isinstance(scopes, basestring):
    scopes = [scopes]
  scopes = sorted(scopes)

  # When acting as account, grab an IAM-scoped token of a primary account first,
  # and use it to sign JWT when making a token for the target account.
  if act_as:
    # Cache key for the target token! Not the IAM-scoped one. The key ID is not
    # known in advance when using signJwt RPC.
    cache_key = _memcache_key(
        method='iam',
        email=act_as,
        scopes=scopes,
        key_id=None)
    # We need IAM-scoped token only on cache miss, so generate it lazily.
    iam_token_factory = (
      lambda: get_access_token_async(
        scopes=['https://www.googleapis.com/auth/iam'],
        service_account_key=service_account_key,
        act_as=None,
        min_lifetime_sec=5*60))
    token = yield _get_or_mint_token_async(
        cache_key,
        min_lifetime_sec,
        lambda: _mint_oauth_token_async(
            iam_token_factory,
            act_as,
            scopes,
            min_lifetime_sec)
    )
    raise ndb.Return(token)

  # Generate a token directly from the service account key.
  if service_account_key:
    # Empty private_key_id probably means that the app is not configured yet.
    if not service_account_key.private_key_id:
      raise AccessTokenError('Service account secret key is not initialized')
    cache_key = _memcache_key(
        method='pkey',
        email=service_account_key.client_email,
        scopes=scopes,
        key_id=service_account_key.private_key_id)
    token = yield _get_or_mint_token_async(
        cache_key,
        min_lifetime_sec,
        lambda: _mint_jwt_based_token_async(
            scopes,
            _LocalSigner(service_account_key))
    )
    raise ndb.Return(token)

  # TODO(vadimsh): Use app_identity.make_get_access_token_call to make it async.
  raise ndb.Return(app_identity.get_access_token(scopes))


def get_access_token(*args, **kwargs):
  """Blocking version of get_access_token_async."""
  return get_access_token_async(*args, **kwargs).get_result()


## Private stuff.


_MEMCACHE_NS = 'access_tokens'


def _memcache_key(method, email, scopes, key_id=None):
  """Returns a string to use as a memcache key for a token.

  Args:
    method: 'pkey' or 'iam'.
    email: service account email we are getting a token for.
    scopes: list of strings with scopes.
    key_id: private key ID used (if known).
  """
  blob = utils.encode_to_json({
    'method': method,
    'email': email,
    'scopes': scopes,
    'key_id': key_id,
  })
  return hashlib.sha256(blob).hexdigest()

@ndb.tasklet
def _get_or_mint_token_async(cache_key, min_lifetime_secs, minter):
  """Gets an accress token from the cache or triggers mint flow."""
  # Randomize refresh time to avoid thundering herd effect when token expires.
  # Also add 5 sec extra to make sure callers will get the token that lives for
  # at least min_lifetime_sec even taking into account possible delays in
  # propagating the token up the stack. We can't give any strict guarantees
  # here though (need to be able to stop time to do that).
  token_info = yield _memcache_get(cache_key, namespace=_MEMCACHE_NS)

  min_allowed_exp = (
    utils.time_time() +
    _randint(min_lifetime_secs + 5, min_lifetime_secs + 305))

  if not token_info or token_info['exp_ts'] < min_allowed_exp:
    token_info = yield minter()
    yield _memcache_set(cache_key, token_info,
                        token_info['exp_ts'], namespace=_MEMCACHE_NS)
  raise ndb.Return(token_info['access_token'], token_info['exp_ts'])

@ndb.tasklet
def _mint_jwt_based_token_async(scopes, signer):
  """Creates new access token given a JWT signer."""
  # For more info see:
  # * https://developers.google.com/accounts/docs/OAuth2ServiceAccount.

  # Prepare a claim set to be signed by the service account key. Note that
  # Google backends seem to ignore 'exp' field and always give one-hour long
  # tokens, so we just always request 1h long token too.
  #
  # Also revert time back a tiny bit, for the sake of machines whose time is not
  # perfectly in sync with global time. If client machine's time is in the
  # future according to Google server clock, the access token request will be
  # denied. It doesn't complain about slightly late clock though.
  logging.info(
    'Refreshing the access token for %s with scopes %s',
    signer.email, scopes)

  now = int(utils.time_time()) - 5
  jwt = yield signer.sign_claimset_async({
    'aud': 'https://www.googleapis.com/oauth2/v4/token',
    'exp': now + 3600,
    'iat': now,
    'iss': signer.email,
    'jti': _b64_encode(os.urandom(16)),
    'scope': ' '.join(scopes),
  })

  # URL encoded body of a token request.
  request_body = urllib.urlencode({
    'grant_type': 'urn:ietf:params:oauth:grant-type:jwt-bearer',
    'assertion': jwt,
  })

  # Exchange signed claimset for an access token.
  token = yield _call_async(
      url='https://www.googleapis.com/oauth2/v4/token',
      payload=request_body,
      method='POST',
      headers={
        'Accept': 'application/json',
        'Content-Type': 'application/x-www-form-urlencoded',
      })
  raise ndb.Return({
    'access_token': str(token['access_token']),
    'exp_ts': int(utils.time_time() + token['expires_in'])
  })

@ndb.tasklet
def _mint_oauth_token_async(token_factory, email, scopes,
    min_lifetime_secs=0, delegates=None):
  """Creates a new access token using IAM credentials API."""
  # Query IAM credentials generateAccessToken API to obtain an OAuth token for
  # a given service account. Maximum lifetime is 1 hour. And can be obtained
  # through a chain of delegates.
  logging.info(
      'Refreshing the access token for %s with scopes %s',
      email, scopes
  )

  request_body = {'scope': scopes}
  if delegates:
    request_body['delegates'] = delegates
  if min_lifetime_secs > 0:
    # Api accepts number of seconds with trailing 's'
    request_body['lifetime'] = '%ds' % min_lifetime_secs

  http_auth, _ = yield token_factory()
  response = yield _call_async(
      url='https://iamcredentials.googleapis.com/v1/projects/-/'
          'serviceAccounts/%s:generateAccessToken' % urllib.quote_plus(email),
      method='POST',
      headers={
        'Accept': 'application/json',
        'Authorization': 'Bearer %s' % http_auth,
        'Content-Type': 'application/json; charset=utf-8',
      },
      payload=utils.encode_to_json(request_body),
  )
  expired_at = int(utils.datetime_to_timestamp(
      utils.parse_rfc3339_datetime(response['expireTime'])) / 1e6)
  raise ndb.Return({
    'access_token': response['accessToken'],
    'exp_ts': expired_at,
  })


@ndb.tasklet
def _call_async(url, payload, method, headers):
  """Makes URL fetch call aggressively retrying on errors a bunch of times.

  On success returns deserialized JSON response body.
  On failure raises AccessTokenError.
  """
  attempt = 0
  while attempt < 4:
    if attempt:
      logging.info('Retrying...')
    attempt += 1
    logging.info('%s %s', method, url)
    try:
      response = yield _urlfetch(
          url=url,
          payload=payload,
          method=method,
          headers=headers,
          follow_redirects=False,
          deadline=5,  # all RPCs we do should be fast
          validate_certificate=True)
    except (apiproxy_errors.DeadlineExceededError, urlfetch.Error) as e:
      # Transient network error or URL fetch service RPC deadline.
      logging.warning('%s %s failed: %s', method, url, e)
      continue

    # Transient error on the other side.
    if response.status_code >= 500:
      logging.warning(
          '%s %s failed with HTTP %d: %r',
          method, url, response.status_code, response.content)
      continue

    # Non-transient error.
    if 300 <= response.status_code < 500:
      logging.warning(
          '%s %s failed with HTTP %d: %r',
          method, url, response.status_code, response.content)
      raise AccessTokenError(
          'Failed to call %s: HTTP %d' % (url, response.status_code))

    # Success.
    try:
      body = json.loads(response.content)
    except ValueError:
      logging.error('Non-JSON response from %s: %r', url, response.content)
      raise AccessTokenError('Non-JSON response from %s' % url)
    raise ndb.Return(body)

  # All our attempts failed with transient errors. Perhaps some later retry
  # can help, so set transient to True.
  raise AccessTokenError(
      'Failed to call %s after multiple attempts' % url, transient=True)


def _randint(*args, **kwargs):
  """To be mocked in tests."""
  return random.randint(*args, **kwargs)

def _urlfetch(**kwargs):
  """To be mocked in tests."""
  return ndb.get_context().urlfetch(**kwargs)


def _memcache_get(*args, **kwargs):
  """To be mocked in tests."""
  return ndb.get_context().memcache_get(*args, **kwargs)


def _memcache_set(*args, **kwargs):
  """To be mocked in tests."""
  return ndb.get_context().memcache_set(*args, **kwargs)


def _is_json_object(blob):
  """True if blob is valid JSON object, i.e '{...}'."""
  try:
    return isinstance(json.loads(blob), dict)
  except ValueError:
    return False


def _log_jwt(email, method, jwt):
  """Logs information about the signed JWT.

  Does some minimal validation which fails only if Google backends misbehave,
  which should not happen. Logs broken JWTs, assuming they are unusable.
  """
  parts = jwt.split('.')
  if len(parts) != 3:
    logging.error(
        'Got broken JWT (not <hdr>.<claims>.<sig>): by=%s method=%s jwt=%r',
        email, method, jwt)
    raise AccessTokenError('Got broken JWT, see logs')

  try:
    hdr = _b64_decode(parts[0])     # includes key ID
    claims = _b64_decode(parts[1])  # includes scopes and timestamp
    sig = parts[2][:12]             # only 9 bytes of the signature
  except (TypeError, ValueError):
    logging.error(
        'Got broken JWT (can\'t base64-decode): by=%s method=%s jwt=%r',
        email, method, jwt)
    raise AccessTokenError('Got broken JWT, see logs')

  if not _is_json_object(hdr):
    logging.error(
        'Got broken JWT (the header is not JSON dict): by=%s method=%s jwt=%r',
        email, method, jwt)
    raise AccessTokenError('Got broken JWT, see logs')
  if not _is_json_object(claims):
    logging.error(
        'Got broken JWT (claims are not JSON dict): by=%s method=%s jwt=%r',
        email, method, jwt)
    raise AccessTokenError('Got broken JWT, see logs')

  logging.info(
      'signed_jwt: by=%s method=%s hdr=%s claims=%s sig_prefix=%s fp=%s',
      email, method, hdr, claims, sig, utils.get_token_fingerprint(jwt))


def _b64_encode(data):
  return base64.urlsafe_b64encode(data).rstrip('=')


def _b64_decode(data):
  mod = len(data) % 4
  if mod:
    data += '=' * (4 - mod)
  return base64.urlsafe_b64decode(data)


## Signers implementation.


class _LocalSigner(object):
  """Knows how to sign JWTs with local private key."""

  def __init__(self, service_account_key):
    self._key = service_account_key

  @property
  def email(self):
    return self._key.client_email

  @ndb.tasklet
  def sign_claimset_async(self, claimset):
    # Prepare JWT header and claimset as base 64.
    header_b64 = _b64_encode(utils.encode_to_json({
      'alg': 'RS256',
      'kid': self._key.private_key_id,
      'typ': 'JWT',
    }))
    claimset_b64 = _b64_encode(utils.encode_to_json(claimset))
    # Sign <header>.<claimset> with account's private key.
    signature_b64 = _b64_encode(self._rsa_sign(
        '%s.%s' % (header_b64, claimset_b64), self._key.private_key))
    jwt = '%s.%s.%s' % (header_b64, claimset_b64, signature_b64)
    _log_jwt(self.email, 'local', jwt)
    raise ndb.Return(jwt)

  @staticmethod
  def _rsa_sign(blob, private_key_pem):
    """Byte blob + PEM key => RSA-SHA256 signature byte blob."""
    # Lazy import crypto. It is not available in unit tests outside of sandbox.
    from Crypto.Hash import SHA256
    from Crypto.PublicKey import RSA
    from Crypto.Signature import PKCS1_v1_5
    pkey = RSA.importKey(private_key_pem)
    return PKCS1_v1_5.new(pkey).sign(SHA256.new(blob))


