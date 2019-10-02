# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Functions to generate and validate tokens signed with MAC tags or RSA."""

import base64
import hashlib
import hmac
import json
import time

from components import utils

from . import api

# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'InvalidTokenError',
  'InvalidSignatureError',
  'TokenKind',
  'verify_jwt',
]


# Name -> (hash algorithm to use, how many bytes of digest to use).
MAC_ALGOS = {
  'hmac-sha256': (hashlib.sha256, 32),
}


# How much clock drift between machines we can tolerate, in seconds.
ALLOWED_CLOCK_DRIFT_SEC = 30


class InvalidTokenError(ValueError):
  """Token validation failed."""


class InvalidSignatureError(InvalidTokenError):
  """A token looks structurally sound, but its signature is invalid."""


class TokenKind(object):
  """Base class for a kind of token.

  Usage:
    1) Define token class, configuring its expiration, name of a secret:
      class MyToken(auth.TokenKind):
        expiration_sec = 3600
        secret_key = auth.SecretKey('my_secret')

    2) Use it to generate a token that embeds and\or signs some data:
      token = MyToken.generate('message to authenticate', {'data': 'public'})

    3) Validate token passed from elsewhere and extract embedded data:
      data = MyToken.validate(token, 'message to authenticate')
      assert data['data'] == 'public'

  Generated token expires after |expiration_sec| or when secret key expires.
  """
  # What algo to use for MAC tag, one of MAC_ALGOS.
  algo = 'hmac-sha256'
  # Defines how long token can live, in seconds. Should be set in subclasses.
  # May be overridden on per-token basis, see |expiration_sec| in 'generate'.
  expiration_sec = None
  # Name of the secret key to use when generating and validating the token.
  # Must be an instance of SecretKey. Should be set in subclasses.
  secret_key = None
  # Format version number that will be embedded into the token.
  version = 1

  @classmethod
  def generate(cls, message=None, embedded=None, expiration_sec=None):
    """Generates a token that contains MAC tag for |message|.

    Args:
      message: single string or list of strings to tag with MAC. It should be
          the same as one used to validate the token. It's not embedded into the
          token. See also 'validate' below.
      embedded: dict with additional data to add to token. It is embedded
          directly into the token and can be easily extracted from it by anyone
          who has the token. Should be used only for publicly visible data.
          It is tagged by token's MAC, so 'validate' function can detect
          any modifications (and reject tokens tampered with).
      expiration_sec: how long token lives before considered expired, overrides
          default TokenKind.expiration_sec if present.

    Returns:
      URL safe base64 encoded token.
    """
    if not cls.is_configured():
      raise ValueError('Token parameters are invalid ')

    # Convert all 'unicode' strings to 'str' in appropriate encoding.
    message = normalize_message(message) if message is not None else []
    embedded = normalize_embedded(embedded) if embedded else {}

    # Fetch an array of last values of secret key.
    secret = api.get_secret(cls.secret_key)
    assert secret

    # Append 'issued' timestamp (in milliseconds) and expiration time.
    embedded['_i'] = str(int(utils.time_time() * 1000))
    if expiration_sec is not None:
      assert expiration_sec > 0, expiration_sec
      embedded['_x'] = str(int(expiration_sec * 1000))

    # Encode token using most recent secret key value.
    return encode_token(cls.algo, cls.version, secret[0], message, embedded)

  @classmethod
  def validate(cls, token, message=None):
    """Checks token MAC and expiration, decodes data embedded into it.

    The following holds:
      token = TokenKind.generate(some_message, token_data)
      assert TokenKind.validate(token, some_message) == token_data

    Args:
      token: a token produced by 'generate' call.
      message: single string or list of strings that should be the same as one
          used to generate the token. If it's different, the token is considered
          invalid. It usually contains some implicitly passed state that should
          be the same when token is generated and validated. For example, it may
          be an account ID of current caller. Then if such token is used by
          another account, it is considered invalid.

    Returns:
      A dict with public data embedded into the token.

    Raises:
      InvalidTokenError if token is broken, tampered with or expired.
    """
    if not cls.is_configured():
      raise ValueError('Token parameters are invalid ')

    # Convert all 'unicode' strings to 'str' in appropriate encoding.
    token = to_encoding(token, 'ascii')
    message = normalize_message(message) if message is not None else []

    # Fetch an array of last values of secret key.
    secret = api.get_secret(cls.secret_key)
    assert secret

    # Decode token, use any recent value of secret to validate MAC.
    version, embedded = decode_token(cls.algo, token, secret, message)

    # Versions should match.
    if version != cls.version:
      raise InvalidTokenError(
          'Bad token format version - expected %r, got %r' %
          (cls.version, version))

    # Grab a timestamp (in milliseconds) when token was issued.
    issued_ts = embedded.pop('_i', None)
    if issued_ts is None:
      raise InvalidTokenError('Bad token: missing issued timestamp')
    issued_ts = int(issued_ts)

    # Discard tokens from the future. Someone is messing with the clock.
    now = utils.time_time() * 1000
    if issued_ts > now + ALLOWED_CLOCK_DRIFT_SEC * 1000:
      raise InvalidTokenError('Bad token: issued timestamp is in the future')

    # Grab expiration time embedded into the token, if any.
    expiration_msec = embedded.pop('_x', None)
    if expiration_msec is None:
      expiration_msec = cls.expiration_sec * 1000
    else:
      expiration_msec = int(expiration_msec)
      assert expiration_msec > 0, expiration_msec

    # Check token expiration.
    if now > issued_ts + expiration_msec:
      raise InvalidTokenError('Bad token: expired')

    return embedded

  @classmethod
  def is_configured(cls):
    """Returns True if token class parameters are correct."""
    return (
        cls.algo in MAC_ALGOS and
        isinstance(cls.expiration_sec, (int, float)) and
        cls.expiration_sec > 0 and
        isinstance(cls.secret_key, api.SecretKey) and
        0 <= cls.version <= 255)


def to_encoding(string, encoding):
  """Unicode or str -> str in given |encoding|.

  If |string| is str already, just returns it as is. If |string| is unicode,
  it gets encoded into |encoding| (where encoding is 'ascii', or 'utf-8', etc.).

  Raises TypeError if |string| is not a string.
  """
  if isinstance(string, str):
    return string
  elif isinstance(string, unicode):
    return string.encode(encoding)
  raise TypeError('Expecting str or unicode')


def normalize_message(message):
  """One string or sequence of strings -> list of 'str' objects in UTF-8.

  Encodes 'unicode' in UTF-8, passes 'str' unchanged.
  """
  if not isinstance(message, (list, tuple)):
    message = [message]
  return [to_encoding(chunk, 'utf-8') for chunk in message]


def normalize_embedded(embedded):
  """Dict of strings -> dict with ASCII keys and values."""
  if not all(
      isinstance(k, basestring) and isinstance(v, basestring)
      for k, v in embedded.iteritems()):
    raise TypeError('Only string keys and values are allowed')
  if any(k.startswith('_') for k in embedded):
    raise ValueError('Keys starting with \'_\' are reserved for internal use')
  return {
    to_encoding(k, 'ascii'): to_encoding(v, 'ascii')
    for k, v in embedded.iteritems()
  }


def base64_encode(data):
  """Bytes str -> URL safe base64 with stripped '='."""
  # Borrowed from ndb's key.py. This is 3-4x faster than urlsafe_b64encode().
  if not isinstance(data, str):
    raise TypeError('Expecting str with binary data')
  urlsafe = base64.b64encode(data)
  return urlsafe.rstrip('=').replace('+', '-').replace('/', '_')


def base64_decode(data):
  """URL safe base64 with stripped '=' -> bytes str."""
  # Borrowed from ndb's key.py, _DecodeUrlSafe.
  if not isinstance(data, str):
    raise TypeError('Expecting str with base64 data')
  mod = len(data) % 4
  if mod:
    data += '=' * (4 - mod)
  # This is 3-4x faster than urlsafe_b64decode()
  return base64.b64decode(data.replace('-', '+').replace('_', '/'))


def compute_mac(algo, secret, chunks):
  """Secret + list of arbitrary strings -> MAC tag.

  Basically transforms a list of chunks into single big string, preserving
  boundaries between separate chunks, and then calculates MAC for it.

  Args:
    algo: MAC algorithm to use, one of MAC_ALGOS.
    secret: string with a secret to use for MAC.
    chunks: list of chunks as byte strings, empty chunks and arbitrary binary
        chunks are allowed. Unicode strings are not allowed, unicode should be
        property encoded into binary beforehand.

  Returns:
    MAC tag as a binary string. Its length depends on |algo| used.
  """
  assert isinstance(secret, str) and secret
  assert isinstance(chunks, list)
  assert algo in MAC_ALGOS, algo
  hash_algo, digest_size = MAC_ALGOS[algo]
  mac = hmac.new(secret, digestmod=hash_algo)
  for chunk in chunks:
    assert isinstance(chunk, str)
    # Separator '\n' is necessary to guarantee that two different messages
    # map to different MACs. Consider two list of chunks:
    #   1) ['0', '', '', '', '', '', '', '', '', '', '']
    #   2) ['0000000000'].
    # Without '\n' they both map to same hmac('100000000000'). With a separator
    # between a chunk length and body there's an algorithm that can take final
    # combined string as input and return original list of chunks. Which means
    # that a list of chunks reversibly (i.e. uniquely) maps to a combined
    # string.
    mac.update('%d\n' % len(chunk))
    mac.update(chunk)
  return mac.digest()[:digest_size]


def encode_token(algo, version, secret, message, embedded):
  """Wraps embedded data, tags it with MAC and encodes in base64.

  Args:
    algo: MAC algorithm to use, one of MAC_ALGOS.
    version: int in range [0, 255], defines version of a token format. Can be
      later accessed without decoding an entire token, so |algo| can potentially
      depend on this version.
    secret: string with a secret to use for MAC.
    message: list of string to tag with MAC.
    embedded: dict to embed into token, it is also tagged by MAC.

  Anatomy of an encoded token:
    base64(version + public + mac(version + public + message))
  Where:
    version - one byte that defines version of a token format.
    public - compactly serialized json dict with |embedded|.
    message - any additional data to tag, it is NOT embedded into the token.

  Returns:
    URL safe base64 encoded token.
  """
  assert isinstance(message, list)
  assert isinstance(embedded, dict)
  public = json.dumps(
      embedded, sort_keys=True, separators=(',', ':'), encoding='ascii')
  mac = compute_mac(algo, secret, [chr(version), public] + message)
  return base64_encode(''.join([chr(version), public, mac]))


def decode_token(algo, token, possible_secrets, message):
  """Decodes a token, checks MAC tag and unwraps embedded data.

  Args:
    algo: MAC algorithm to use, one of MAC_ALGOS. Should be same as used to
        encode this token.
    token: actual token value in base64 encoded form.
    possible_secrets: list of secret keys to try to use to validate MAC tag.
    message: list of string tagged by MAC in this token, should be same list as
        used to create it.

  Returns:
    Tuple (version, embedded data dict).
  """
  assert algo in MAC_ALGOS, algo
  assert isinstance(token, str)
  assert possible_secrets
  assert isinstance(message, list)

  # Unwrap version, embedded data and MAC.
  _, digest_size = MAC_ALGOS[algo]
  try:
    # One byte for version, at least one byte for public embedded dict portion,
    # the rest is MAC digest.
    binary = base64_decode(token)
    if len(binary) < digest_size + 2:
      raise ValueError()
    version = ord(binary[0])
    public = binary[1:-digest_size]
    token_mac = binary[-digest_size:]
  except (ValueError, TypeError):
    raise InvalidTokenError('Bad token format: %r' % token)

  # Validate MAC tag. Run in constant time to prevent timing attacks.
  for secret in possible_secrets:
    good_mac = compute_mac(algo, secret, [chr(version), public] + message)
    assert len(good_mac) == len(token_mac)
    accum = 0
    for x, y in zip(token_mac, good_mac):
      accum |= ord(x) ^ ord(y)
    # Match! Return version and embedded token data. It somewhat breaks constant
    # time promise, but at that point token is verified to be valid anyway. For
    # invalid tokens all cycles of the loop are executed.
    if not accum:
      # The public part is a JSON encoded dict with ASCII key-value pairs,
      # as generated by normalize_embedded. Convert the result to ASCII too.
      public = {
        k.encode('ascii'): v.encode('ascii')
        for k, v in json.loads(public).iteritems()
      }
      return version, public

  try:
    public = {
      k.encode('ascii', 'replace'): v.encode('ascii', 'replace')
      for k, v in json.loads(public).iteritems()
    }
  except (AttributeError, ValueError):
    pass
  # At least one secret key should match.
  raise InvalidTokenError(
      'Bad token MAC; now=%d; data=%s' % (time.time(), public))


def verify_jwt(jwt, bundle):
  """Verifies and decodes a JWT, returning its JSON payload.

  Checks its header (must be using RS256 algo with one of the keys from the
  bundle), its RSA signature, and its issued and expiration times. Does not
  check the audience.

  Supports only RS256 algo. Tokens that use something else are rejected with
  InvalidTokenError exception.

  See https://tools.ietf.org/html/rfc7519 for more details on JWT. This function
  supports only limited subset of the spec needed to verify JWTs produced by
  Google backends.

  TODO(vadimsh): Consider pulling a third party JWT library as a dependency if
  we need something more advanced in the future.

  Args:
    jwt: JWT (as '<base64 hdr>.<base64 payload>.<base64 sig>' string).
    bundle: signature.CertificateBundle object with public keys.

  Returns:
    Tuple (header dict, verified and decoded payload as dict).

  Raises:
    InvalidTokenError if JWT is malformed or expired.
    InvalidSignatureError if JWT's signature is invalid.
    signature.CertificateError if the signing key is unknown or invalid.
  """
  jwt = jwt.encode('ascii') if isinstance(jwt, unicode) else jwt
  if jwt.count('.') != 2:
    raise InvalidTokenError('Bad JWT, should have 3 segments')
  segments = jwt.split('.')

  try:
    hdr, payload, sig = (base64_decode(b64) for b64 in segments)
  except (ValueError, TypeError) as exc:
    raise InvalidTokenError('Malformed JWT, not valid base64: %s' % exc)

  try:
    hdr_dict = json.loads(hdr)
    if not isinstance(hdr_dict, dict):
      raise ValueError('not a dict')
    typ = hdr_dict.get('typ', 'JWT')
    alg = hdr_dict.get('alg')
    kid = hdr_dict.get('kid')
  except ValueError as exc:
    raise InvalidTokenError('Malformed JWT header %r: %s' % (hdr, exc))

  if typ != 'JWT':
    raise InvalidTokenError('Only JWT tokens are supported, got %s' % (typ,))
  if alg != 'RS256':
    raise InvalidTokenError('Only RS256 tokens are supported, got %s' % (alg,))
  if not kid:
    raise InvalidTokenError('Key ID is not specified in the header')
  if not bundle.check_signature('%s.%s' % (segments[0], segments[1]), kid, sig):
    raise InvalidSignatureError('Bad JWT: invalid signature')

  # Here token's signature is valid, but the token may have expired already.
  try:
    payload = json.loads(payload)
    if not isinstance(payload, dict):
      raise ValueError('not a dict')
  except ValueError as exc:
    raise InvalidTokenError('Malformed JWT payload %r: %s' % (payload, exc))

  for key in ('iat', 'exp', 'nbf'):
    ts = payload.get(key)
    if not ts:
      if key == 'nbf':
        continue  # 'nbf' is optional
      raise InvalidTokenError('Bad JWT: has no %r field' % key)
    if not isinstance(ts, (int, long, float)):
      raise InvalidTokenError('Bad JWT: %r (%r) is not a number' % (key, ts))

  now = utils.time_time()
  nbf = payload.get('nbf') or payload['iat']
  exp = payload['exp']

  # Make sure the token is not from the future or already expired. Give some
  # wiggle room to account for a possible clock skew between the machine that
  # produced the token and us.
  if now < nbf - ALLOWED_CLOCK_DRIFT_SEC:
    raise InvalidTokenError('Bad JWT: too early (now %d < nbf %d)' % (now, nbf))
  if now > exp + ALLOWED_CLOCK_DRIFT_SEC:
    raise InvalidTokenError('Bad JWT: expired (now %d > exp %d)' % (now, exp))

  return hdr_dict, payload
