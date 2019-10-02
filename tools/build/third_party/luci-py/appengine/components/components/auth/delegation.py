# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Delegation token implementation.

See delegation.proto for general idea behind it.
"""

import collections
import datetime
import hashlib
import json
import logging
import urllib

from google.appengine.api import urlfetch
from google.appengine.ext import ndb
from google.appengine.runtime import apiproxy_errors
from google.protobuf import message

from components import utils

from . import api
from . import model
from . import service_account
from . import signature
from . import tokens
from .proto import delegation_pb2


__all__ = [
  'delegate',
  'delegate_async',
  'DelegationToken',
  'DelegationTokenCreationError',
]


# TODO(vadimsh): Add a simple encryption layer, so that token's guts are not
# visible in plain form to anyone who sees the token.


# Tokens that are larger than this (before base64 encoding) are rejected.
MAX_TOKEN_SIZE = 8 * 1024

# How much clock drift between machines we can tolerate, in seconds.
ALLOWED_CLOCK_DRIFT_SEC = 30

# Name of the HTTP header to look for delegation token.
HTTP_HEADER = 'X-Delegation-Token-V1'


class BadTokenError(Exception):
  """Raised on fatal errors (like bad signature). Results in 403 HTTP code."""


class TransientError(Exception):
  """Raised on errors that can go away with retry. Results in 500 HTTP code."""


class DelegationTokenCreationError(Exception):
  """Raised on delegation token creation errors."""


class DelegationAuthorizationError(DelegationTokenCreationError):
  """Raised on authorization error during delegation token creation."""


# A minted delegation token returned by delegate_async and delegate.
DelegationToken = collections.namedtuple('DelegationToken', [
  'token',  # urlsafe base64 encoded blob with delegation token.
  'expiry',  # datetime.datetime of expiration.
])


@utils.cache_with_expiration(expiration_sec=900)
def get_trusted_signers():
  """Returns dict {signer_id => CertificateBundle} with all signers we trust.

  Keys are identity strings (e.g. 'service:<app-id>' and 'user:<email>'), values
  are CertificateBundle with certificates to use when verifying signatures.
  """
  auth_db = api.get_request_auth_db()

  # We currently trust only the token server, as provided by the primary.
  if not auth_db.primary_url:
    logging.warning(
        'Delegation is not supported, not linked to an auth service')
    return {}
  if not auth_db.token_server_url:
    logging.warning(
        'Delegation is not supported, the token server URL is not set by %s',
        auth_db.primary_url)
    return {}

  certs = signature.get_service_public_certificates(auth_db.token_server_url)
  if certs.service_account_name:
    # Construct Identity to ensure service_account_name is a valid email.
    tok_server_id = model.Identity('user', certs.service_account_name)
    return {tok_server_id.to_bytes(): certs}

  # This can happen if the token server is too old (pre v1.2.9). Just skip
  # it then.
  logging.warning(
      'The token server at %r didn\'t provide its service account name. '
      'Old version? Ignoring it.', auth_db.token_server_url)
  return {}


## Low level API for components.auth and services that know what they are doing.


def deserialize_token(blob):
  """Coverts urlsafe base64 text to delegation_pb2.DelegationToken.

  Raises:
    BadTokenError if blob doesn't look like a valid DelegationToken.
  """
  if isinstance(blob, unicode):
    blob = blob.encode('ascii', 'ignore')
  try:
    as_bytes = tokens.base64_decode(blob)
  except (TypeError, ValueError) as exc:
    raise BadTokenError('Not base64: %s' % exc)
  if len(as_bytes) > MAX_TOKEN_SIZE:
    raise BadTokenError('Unexpectedly huge token (%d bytes)' % len(as_bytes))
  try:
    return delegation_pb2.DelegationToken.FromString(as_bytes)
  except message.DecodeError as exc:
    raise BadTokenError('Bad proto: %s' % exc)


def unseal_token(tok):
  """Checks the signature of DelegationToken and deserializes the subtoken.

  Does not check the subtoken itself.

  Args:
    tok: delegation_pb2.DelegationToken message.

  Returns:
    delegation_pb2.Subtoken message.

  Raises:
    BadTokenError:
        On non-transient fatal errors: if token is not structurally valid,
        signature is bad or signer is untrusted.
    TransientError:
        On transient errors, that may go away on the next call: for example if
        the signer public key can't be fetched.
  """
  # Check all required fields are set.
  assert isinstance(tok, delegation_pb2.DelegationToken)

  if not tok.serialized_subtoken:
    raise BadTokenError('serialized_subtoken is missing')
  if not tok.signer_id:
    raise BadTokenError('signer_id is missing')
  if not tok.signing_key_id:
    raise BadTokenError('signing_key_id is missing')
  if not tok.pkcs1_sha256_sig:
    raise BadTokenError('pkcs1_sha256_sig is missing')

  # Make sure signer_id looks like model.Identity.
  try:
    model.Identity.from_bytes(tok.signer_id)
  except ValueError as exc:
    raise BadTokenError('signer_id is not a valid identity: %s' % exc)

  # Validate the signature.
  certs = get_trusted_signers().get(tok.signer_id)
  if not certs:
    raise BadTokenError('Not a trusted signer: %r' % tok.signer_id)
  try:
    is_valid_sig = certs.check_signature(
        blob=tok.serialized_subtoken,
        key_name=tok.signing_key_id,
        signature=tok.pkcs1_sha256_sig)
  except signature.CertificateError as exc:
    if exc.transient:
      raise TransientError(str(exc))
    raise BadTokenError(
        'Bad certificate (signer_id == %s, signing_key_id == %s): %s' % (
        tok.signer_id, tok.signing_key_id, exc))
  if not is_valid_sig:
    raise BadTokenError(
        'Invalid signature (signer_id == %s, signing_key_id == %s)' % (
        tok.signer_id, tok.signing_key_id))

  # The signature is correct, deserialize the subtoken.
  try:
    return delegation_pb2.Subtoken.FromString(tok.serialized_subtoken)
  except message.DecodeError as exc:
    raise BadTokenError('Bad serialized_subtoken: %s' % exc)


## Token creation.


def _urlfetch_async(**kwargs):
  """To be mocked in tests."""
  return ndb.get_context().urlfetch(**kwargs)


@ndb.tasklet
def _authenticated_request_async(url, method='GET', payload=None, params=None):
  """Sends an authenticated JSON API request, returns deserialized response.

  Raises:
    DelegationTokenCreationError if request failed or response is malformed.
    DelegationAuthorizationError on HTTP 401 or 403 response from auth service.
  """
  scope = 'https://www.googleapis.com/auth/userinfo.email'
  access_token = service_account.get_access_token(scope)[0]
  headers = {
    'Accept': 'application/json; charset=utf-8',
    'Authorization': 'Bearer %s' % access_token,
  }

  if payload is not None:
    assert method in ('CREATE', 'POST', 'PUT'), method
    headers['Content-Type'] = 'application/json; charset=utf-8'
    payload = utils.encode_to_json(payload)

  if utils.is_local_dev_server():
    protocols = ('http://', 'https://')
  else:
    protocols = ('https://',)
  assert url.startswith(protocols) and '?' not in url, url
  if params:
    url += '?' + urllib.urlencode(params)

  try:
    res = yield _urlfetch_async(
        url=url,
        payload=payload,
        method=method,
        headers=headers,
        follow_redirects=False,
        deadline=10,
        validate_certificate=True)
  except (apiproxy_errors.DeadlineExceededError, urlfetch.Error) as e:
    raise DelegationTokenCreationError(str(e))

  if res.status_code in (401, 403):
    logging.error('Token server HTTP %d: %s', res.status_code, res.content)
    raise DelegationAuthorizationError(
        'HTTP %d: %s' % (res.status_code, res.content))

  if res.status_code >= 300:
    logging.error('Token server HTTP %d: %s', res.status_code, res.content)
    raise DelegationTokenCreationError(
        'HTTP %d: %s' % (res.status_code, res.content))

  try:
    content = res.content
    if content.startswith(")]}'\n"):
      content = content[5:]
    json_res = json.loads(content)
  except ValueError as e:
    raise DelegationTokenCreationError('Bad JSON response: %s' % e)
  raise ndb.Return(json_res)


@ndb.tasklet
def delegate_async(
    audience,
    services,
    min_validity_duration_sec=5*60,
    max_validity_duration_sec=60*60*3,
    impersonate=None,
    tags=None,
    token_server_url=None):
  """Creates a delegation token by contacting the token server.

  Memcaches the token.

  Args:
    audience (list of (str or Identity)): to WHOM caller's identity is
      delegated; a list of identities or groups, a string "REQUESTOR" (to
      indicate the current service) or symbol '*' (which means ANY).
      Example: ['user:def@example.com', 'group:abcdef', 'REQUESTOR'].
    services (list of (str or Identity)): WHERE token is accepted.
      Each list element must be an identity of 'service' kind, a root URL of a
      service (e.g. 'https://....'), or symbol '*'.
      Example: ['service:gae-app1', 'https://gae-app2.appspot.com']
    min_validity_duration_sec (int): minimally acceptable lifetime of the token.
      If there's existing token cached locally that have TTL
      min_validity_duration_sec or more, it will be returned right away.
      Default is 5 min.
    max_validity_duration_sec (int): defines lifetime of a new token.
      It will bet set as tokens' TTL if there's no existing cached tokens with
      sufficiently long lifetime. Default is 3 hours.
    impersonate (str or Identity): a caller can mint a delegation token on
      someone else's behalf (effectively impersonating them). Only a privileged
      set of callers can do that. If impersonation is allowed, token's
      delegated_identity field will contain whatever is in 'impersonate' field.
      Example: 'user:abc@example.com'
    tags (list of str): optional list of key:value pairs to embed into the
      token. Services that accept the token may use them for additional
      authorization decisions.
    token_server_url (str): the URL for the token service that will mint the
      token. Defaults to the URL provided by the primary auth service.

  Returns:
    DelegationToken as ndb.Future.

  Raises:
    ValueError if args are invalid.
    DelegationTokenCreationError if could not create a token.
    DelegationAuthorizationError on HTTP 403 response from auth service.
  """
  assert isinstance(audience, list), audience
  assert isinstance(services, list), services

  id_to_str = lambda i: i.to_bytes() if isinstance(i, model.Identity) else i

  # Validate audience.
  if '*' in audience:
    audience = ['*']
  else:
    if not audience:
      raise ValueError('audience can\'t be empty')
    for a in audience:
      if isinstance(a, model.Identity):
        continue # identities are already validated
      if not isinstance(a, basestring):
        raise ValueError('expecting a string or Identity')
      if a == 'REQUESTOR' or a.startswith('group:'):
        continue
      # The only remaining option is a string that represents an identity.
      # Validate it. from_bytes may raise ValueError.
      model.Identity.from_bytes(a)
    audience = sorted(map(id_to_str, audience))

  # Validate services.
  if '*' in services:
    services = ['*']
  else:
    if not services:
      raise ValueError('services can\'t be empty')
    for s in services:
      if isinstance(s, basestring):
        if s.startswith('https://'):
          continue  # an URL, the token server knows how to handle it
        s = model.Identity.from_bytes(s)
      assert isinstance(s, model.Identity), s
      assert s.kind == model.IDENTITY_SERVICE, s
    services = sorted(map(id_to_str, services))

  # Validate validity durations.
  assert isinstance(min_validity_duration_sec, int), min_validity_duration_sec
  assert isinstance(max_validity_duration_sec, int), max_validity_duration_sec
  assert min_validity_duration_sec >= 5
  assert max_validity_duration_sec >= 5
  assert min_validity_duration_sec <= max_validity_duration_sec

  # Validate impersonate.
  if impersonate is not None:
    assert isinstance(impersonate, (basestring, model.Identity)), impersonate
    impersonate = id_to_str(impersonate)

  # Validate tags.
  tags = sorted(tags or [])
  for tag in tags:
    parts = tag.split(':', 1)
    if len(parts) != 2 or parts[0] == '' or parts[1] == '':
      raise ValueError('Bad delegation token tag: %r' % tag)

  # Grab the token service URL.
  if not token_server_url:
    token_server_url = api.get_request_auth_db().token_server_url
    if not token_server_url:
      raise DelegationTokenCreationError('Token server URL is not configured')

  # End of validation.

  # See MintDelegationTokenRequest in
  # https://github.com/luci/luci-go/blob/master/tokenserver/api/minter/v1/token_minter.proto.
  req = {
    'delegatedIdentity': impersonate or 'REQUESTOR',
    'validityDuration': max_validity_duration_sec,
    'audience': audience,
    'services': services,
    'tags': tags,
  }

  # Get from cache.
  cache_key_hash = hashlib.sha256(
      token_server_url + '\n' + json.dumps(req, sort_keys=True)).hexdigest()
  cache_key = 'delegation_token/v2/%s' % cache_key_hash
  ctx = ndb.get_context()
  token = yield ctx.memcache_get(cache_key)
  min_validity_duration = datetime.timedelta(seconds=min_validity_duration_sec)
  now = utils.utcnow()
  if token and token.expiry - min_validity_duration > now:
    logging.info(
        'Fetched cached delegation token: fingerprint=%s',
        utils.get_token_fingerprint(token.token))
    raise ndb.Return(token)

  # Request a new one.
  logging.info(
      'Minting a delegation token for %r',
      {k: v for k, v in req.iteritems() if v},
  )
  res = yield _authenticated_request_async(
      '%s/prpc/tokenserver.minter.TokenMinter/MintDelegationToken' %
          token_server_url,
      method='POST',
      payload=req)

  signed_token = res.get('token')
  if not signed_token or not isinstance(signed_token, basestring):
    logging.error('Bad MintDelegationToken response: %s', res)
    raise DelegationTokenCreationError('Bad response, no token')

  token_struct = res.get('delegationSubtoken')
  if not token_struct or not isinstance(token_struct, dict):
    logging.error('Bad MintDelegationToken response: %s', res)
    raise DelegationTokenCreationError('Bad response, no delegationSubtoken')

  if token_struct.get('kind') != 'BEARER_DELEGATION_TOKEN':
    logging.error('Bad MintDelegationToken response: %s', res)
    raise DelegationTokenCreationError(
        'Bad response, not BEARER_DELEGATION_TOKEN')

  actual_validity_duration_sec = token_struct.get('validityDuration')
  if not isinstance(actual_validity_duration_sec, (int, float)):
    logging.error('Bad MintDelegationToken response: %s', res)
    raise DelegationTokenCreationError(
        'Unexpected response, validityDuration is absent or not a number')

  token = DelegationToken(
      token=str(signed_token),
      expiry=now + datetime.timedelta(seconds=actual_validity_duration_sec),
  )

  logging.info(
      'Token server "%s" generated token (subtoken_id=%s, fingerprint=%s):\n%s',
      res.get('serviceVersion'),
      token_struct.get('subtokenId'),
      utils.get_token_fingerprint(token.token),
      json.dumps(
          res.get('delegationSubtoken'),
          sort_keys=True, indent=2, separators=(',', ': ')))

  # Put to cache. Refresh the token 10 sec in advance.
  if actual_validity_duration_sec > 10:
    yield ctx.memcache_add(
        cache_key, token, time=actual_validity_duration_sec - 10)

  raise ndb.Return(token)


def delegate(**kwargs):
  """Blocking version of delegate_async."""
  return delegate_async(**kwargs).get_result()


## Token validation.


def check_subtoken(subtoken, peer_identity, auth_db):
  """Validates the delegation subtoken, extracts delegated_identity.

  Args:
    subtoken: instance of delegation_pb2.Subtoken.
    peer_identity: identity of whoever tries to use this token.
    auth_db: instance of AuthDB with groups.

  Returns:
    Delegated Identity extracted from the token (if it is valid).

  Raises:
    BadTokenError if the token is invalid or not usable by peer_identity.
  """
  assert isinstance(subtoken, delegation_pb2.Subtoken)

  # Do fast failing checks before heavy ones.
  service_id = model.get_service_self_identity()
  check_subtoken_expiration(subtoken, int(utils.time_time()))
  check_subtoken_services(subtoken, service_id.to_bytes())

  # Verify caller can use the token, figure out a delegated identity.
  check_subtoken_audience(subtoken, peer_identity, auth_db)
  try:
    return model.Identity.from_bytes(subtoken.delegated_identity)
  except ValueError as exc:
    raise BadTokenError('Invalid delegated_identity: %s' % exc)


def check_subtoken_expiration(subtoken, now):
  """Checks 'creation_time' and 'validity_duration' fields.

  Args:
    subtoken: instance of delegation_pb2.Subtoken.
    now: current time (number of seconds since epoch).

  Raises:
    BadTokenError if token has expired or not valid yet.
  """
  if not subtoken.creation_time:
    raise BadTokenError('Missing "creation_time" field')
  if subtoken.validity_duration <= 0:
    raise BadTokenError(
        'Invalid validity_duration: %d' % subtoken.validity_duration)
  if subtoken.creation_time >= now + ALLOWED_CLOCK_DRIFT_SEC:
    raise BadTokenError(
        'Token is not active yet (%d < %d)' %
        (subtoken.creation_time, now + ALLOWED_CLOCK_DRIFT_SEC))
  if subtoken.creation_time + subtoken.validity_duration < now:
    exp = now - (subtoken.creation_time + subtoken.validity_duration)
    raise BadTokenError('Token has expired %d sec ago' % exp)


def check_subtoken_services(subtoken, service_id):
  """Checks 'services' field of the subtoken.

  Args:
    subtoken: instance of delegation_pb2.Subtoken.
    service_id: 'service:<id>' string to look for in 'services' field.

  Raises:
    BadTokenError if token is not intended for the current service.
  """
  if not subtoken.services:
    raise BadTokenError('The token\'s services list is empty')
  if '*' not in subtoken.services and service_id not in subtoken.services:
    raise BadTokenError('The token is not intended for %s' % service_id)


def check_subtoken_audience(subtoken, current_identity, auth_db):
  """Checks 'audience' field of the subtoken.

  Args:
    subtoken: instance of delegation_pb2.Subtoken.
    current_identity: Identity to look for in 'audience' field.
    auth_db: instance of AuthDB with groups.

  Raises:
    BadTokenError if token is not allowed to be used by current_identity.
  """
  # No audience at all -> forbid.
  if not subtoken.audience:
    raise BadTokenError('The token\'s audience field is empty')
  # '*' in audience -> allow all.
  if '*' in subtoken.audience:
    return
  # Try to find a direct hit first, to avoid calling expensive is_group_member.
  ident_as_bytes = current_identity.to_bytes()
  if ident_as_bytes in subtoken.audience:
    return
  # Search through groups now.
  for aud in subtoken.audience:
    if not aud.startswith('group:'):
      continue
    group = aud[len('group:'):]
    if auth_db.is_group_member(group, current_identity):
      return
  raise BadTokenError('%s is not allowed to use the token' % ident_as_bytes)


## High level API to parse, validate and traverse delegation token.


def check_bearer_delegation_token(token, peer_identity, auth_db=None):
  """Decodes the token, checks its validity, extracts delegated Identity.

  Logs details about the token.

  Args:
    token: blob with base64 encoded delegation token.
    peer_identity: Identity of whoever tries to wield the token.
    auth_db: AuthDB instance with groups, defaults to get_request_auth_db().

  Returns:
    (Delegated Identity, validated delegation_pb2.Subtoken proto).

  Raises:
    BadTokenError if token is invalid.
    TransientError if token can't be verified due to transient errors.
  """
  logging.info(
      'Checking delegation token: fingerprint=%s',
      utils.get_token_fingerprint(token))
  subtoken = unseal_token(deserialize_token(token))
  if subtoken.kind != delegation_pb2.Subtoken.BEARER_DELEGATION_TOKEN:
    raise BadTokenError('Not a valid delegation token kind: %s' % subtoken.kind)
  ident = check_subtoken(
      subtoken, peer_identity, auth_db or api.get_request_auth_db())
  logging.info(
      'Using delegation token: subtoken_id=%s, delegated_identity=%s',
      subtoken.subtoken_id, ident.to_bytes())
  return ident, subtoken
