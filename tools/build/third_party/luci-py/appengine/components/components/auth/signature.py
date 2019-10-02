# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Functions to produce and verify RSA+SHA256 signatures.

Based on app_identity.sign_blob() and app_identity.get_public_certificates()
functions, and thus private keys are managed by GAE.
"""

import base64
import json
import logging
import random
import threading
import urllib

from google.appengine.api import app_identity
from google.appengine.api import memcache
from google.appengine.api import urlfetch
from google.appengine.runtime import apiproxy_errors

from components import utils


# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'CertificateBundle',
  'CertificateError',
  'get_google_oauth2_certs',
  'get_own_public_certificates',
  'get_service_account_certificates',
  'get_service_public_certificates',
  'sign_blob',
]


# Base URL to fetch Google service account certs from.
_GOOGLE_ROBOT_CERTS_URL = 'https://www.googleapis.com/robot/v1/metadata/x509/'
# URL to fetch Google root OAuth2 certs from.
_GOOGLE_OAUTH2_CERTS_URL = 'https://www.googleapis.com/oauth2/v1/certs'

# For how long to cache the certificates. Due to the way how local memory cache
# and memcache are used, the actual expiration time can be up to 2 times larger.
# This is fine, since certs lifetime are usually 12h or more.
_CERTS_CACHE_EXP_SEC = 3600

_certs_cache = {} # cache key => (CertificateBundle, cache expiration time)
_certs_cache_lock = threading.Lock()


class CertificateError(Exception):
  """Errors when working with a certificate."""

  def __init__(self, msg, transient=False):
    super(CertificateError, self).__init__(msg)
    self.transient = transient


class CertificateBundle(object):
  """A bunch of certificates of some service or service account.

  This is NOT a certificate chain.

  It is just a list of currently non-expired certificates of some service.
  Usually one of them contains a public key currently used for singing, while
  the rest contain "old" public keys that are being rotated out.

  It is identical to this structure:
  {
    'certificates': [
      {
        'key_name': '...',
        'x509_certificate_pem': '...'
      },
      ...
    ],
    'timestamp': 123354545
  }

  This object caches parsed public keys internally.
  """

  def __init__(self, jsonish):
    self._jsonish = jsonish
    self._lock = threading.Lock()
    self._verifiers = {} # key_id => PKCS1_v1_5 verifier

  @property
  def service_account_name(self):
    """Returns the service account that owns the keys.

    It matches Subject Name in the certificates.

    May return None for CertificateBundle objects fetched from old services
    that do not provide this information. The support was added in v1.2.9.
    """
    return self._jsonish.get('service_account_name')

  @property
  def app_id(self):
    """If the service that owns the keys is on GAE, returns its app ID.

    May return None for CertificateBundle objects fetched from old services
    that do not provide this information. The support was added in v1.2.11.
    """
    return self._jsonish.get('app_id')

  def to_jsonish(self):
    """Returns JSON-serializable representation of this bundle.

    Caller must not modify the returned object.

    {
      'app_id': '<GAE app id or None if not fetched from GAE>',
      'service_account_name': '<email>',
      'certificates': [
        {
          'key_name': '...',
          'x509_certificate_pem': '...'
        },
        ...
      ],
      'timestamp': 123354545 # when it was fetched
    }
    """
    return self._jsonish

  def check_signature(self, blob, key_name, signature):
    """Verifies signature produced by 'sign_blob' function.

    Args:
      blob: binary buffer to check the signature for.
      key_name: identifier of a private key used to sign the blob.
      signature: the signature, as returned by sign_blob function.

    Returns:
      True if signature is correct.

    Raises:
      CertificateError if no such key or the certificate is invalid.
    """
    # Lazy import Crypto, since not all service that use 'auth' may need it.
    from Crypto.Hash import SHA256
    from Crypto.PublicKey import RSA
    from Crypto.Signature import PKCS1_v1_5
    from Crypto.Util import asn1

    verifier = None
    with self._lock:
      verifier = self._verifiers.get(key_name)
      if not verifier:
        # Grab PEM-encoded x509 cert.
        x509_cert = None
        for cert in self._jsonish['certificates']:
          if cert['key_name'] == key_name:
            x509_cert = cert['x509_certificate_pem']
            break
        else:
          raise CertificateError('The key %r was not found' % key_name)

        # See https://stackoverflow.com/a/12921889.

        # Convert PEM to DER. There's a function for this in 'ssl' module
        # (ssl.PEM_cert_to_DER_cert), but 'ssl' is not importable in GAE sandbox
        # on dev server (C extension is not whitelisted).
        lines = x509_cert.strip().split('\n')
        if (len(lines) < 3 or
            lines[0] != '-----BEGIN CERTIFICATE-----' or
            lines[-1] != '-----END CERTIFICATE-----'):
          raise CertificateError('Invalid certificate format')
        der = base64.b64decode(''.join(lines[1:-1]))

        # Extract subjectPublicKeyInfo field from X.509 certificate
        # (see RFC3280).
        cert = asn1.DerSequence()
        cert.decode(der)
        tbsCertificate = asn1.DerSequence()
        tbsCertificate.decode(cert[0])
        subjectPublicKeyInfo = tbsCertificate[6]

        # TODO(vadimsh): Extract certificate subject name and verify that it
        # matches self.service_account_name. Unfortunately, PyCrypto's asn1
        # library is to dumb for this task. It doesn't support ASN1 SET OF
        # elements.

        verifier = PKCS1_v1_5.new(RSA.importKey(subjectPublicKeyInfo))
        self._verifiers[key_name] = verifier

    return verifier.verify(SHA256.new(blob), signature)


def sign_blob(blob, deadline=None):
  """Signs a blob using current service's private key.

  Just an alias for GAE app_identity.sign_blob function for symmetry with
  'check_signature'. Note that |blob| can be at most 8KB.

  Returns:
    Tuple (name of a key used, RSA+SHA256 signature).
  """
  # app_identity.sign_blob is producing RSA+SHA256 signature. Sadly, it isn't
  # documented anywhere. But it should be relatively stable since this API is
  # used by OAuth2 libraries (and so changing signature method may break a lot
  # of stuff).
  return app_identity.sign_blob(blob, deadline)


def get_google_oauth2_certs():
  """Returns CertificateBundle with Google's public OAuth2 certificates."""
  return _use_cached_or_fetch(
      'v1:google_auth2_certs',
      lambda: _fetch_certs_from_json(_GOOGLE_OAUTH2_CERTS_URL))


@utils.cache_with_expiration(3600)
def get_own_public_certificates():
  """Returns CertificateBundle with certificates of the current service."""
  attempt = 0
  while True:
    attempt += 1
    try:
      certs = app_identity.get_public_certificates(deadline=1.5)
      break
    except apiproxy_errors.DeadlineExceededError as e:
      logging.warning('%s', e)
      if attempt == 3:
        raise
  return CertificateBundle({
    'app_id': app_identity.get_application_id(),
    'service_account_name': utils.get_service_account_name(),
    'certificates': [
      {
        'key_name': cert.key_name,
        'x509_certificate_pem': cert.x509_certificate_pem,
      }
      for cert in certs
    ],
    'timestamp': utils.datetime_to_timestamp(utils.utcnow()),
  })


def get_service_public_certificates(service_url):
  """Returns CertificateBundle with certificates of a LUCI service.

  The LUCI service at |service_url| must have 'auth' component enabled (to serve
  the certificates).

  Raises CertificateError on errors.
  """
  return _use_cached_or_fetch(
      'v1:service_certs:%s' % service_url,
      lambda: _fetch_service_certs(service_url))


def get_service_account_certificates(service_account_name):
  """Returns CertificateBundle with certificates of a service account.

  Works only for Google Cloud Platform service accounts.

  Raises CertificateError on errors.
  """
  return _use_cached_or_fetch(
      'v1:service_account_certs:%s' % service_account_name,
      lambda: _fetch_certs_from_json(
          url=_GOOGLE_ROBOT_CERTS_URL+urllib.quote_plus(service_account_name),
          service_account_name=service_account_name,
      ))


def _fetch_service_certs(service_url):
  protocol = 'https://'
  if utils.is_local_dev_server():
    protocol = ('http://', 'https://')
  assert service_url.startswith(protocol), (service_url, protocol)
  url = '%s/auth/api/v1/server/certificates' % service_url

  # Retry code is adapted from components/net.py. net.py can't be used directly
  # since it depends on components.auth (and dependency cycles between
  # components are bad).
  attempt = 0
  result = None
  while attempt < 4:
    if attempt:
      logging.info('Retrying...')
    attempt += 1
    logging.info('GET %s', url)
    try:
      result = urlfetch.fetch(
          url=url,
          method='GET',
          headers={'X-URLFetch-Service-Id': utils.get_urlfetch_service_id()},
          follow_redirects=False,
          deadline=5,
          validate_certificate=True)
    except (apiproxy_errors.DeadlineExceededError, urlfetch.Error) as e:
      # Transient network error or URL fetch service RPC deadline.
      logging.warning('GET %s failed: %s', url, e)
      continue
    # It MUST return 200 on success, it can't return 403, 404 or >=500.
    if result.status_code != 200:
      logging.warning(
          'GET %s failed, HTTP %d: %r', url, result.status_code, result.content)
      continue
    return json.loads(result.content)

  # All attempts failed, give up.
  msg = 'Failed to grab public certs from %s (HTTP code %s)' % (
      service_url, result.status_code if result else '???')
  raise CertificateError(msg, transient=True)


def _fetch_certs_from_json(url, service_account_name=None):
  """Fetches certs from a JSON bundle in form {<key_id>: <pem encoded cert>}."""
  # Retry code is adapted from components/net.py. net.py can't be used directly
  # since it depends on components.auth (and dependency cycles between
  # components are bad).
  attempt = 0
  result = None
  while attempt < 4:
    if attempt:
      logging.info('Retrying...')
    attempt += 1
    logging.info('GET %s', url)
    try:
      result = urlfetch.fetch(
          url=url,
          method='GET',
          follow_redirects=False,
          deadline=5,
          validate_certificate=True)
    except (apiproxy_errors.DeadlineExceededError, urlfetch.Error) as e:
      # Transient network error or URL fetch service RPC deadline.
      logging.warning('GET %s failed: %s', url, e)
      continue
    # It MUST return 200 on success, it can't return 403, 404 or >=500.
    if result.status_code != 200:
      logging.warning(
          'GET %s failed, HTTP %d: %r', url, result.status_code, result.content)
      continue
    response = json.loads(result.content)
    return {
      'service_account_name': service_account_name,
      'certificates': [
        {
          'key_name': key_name,
          'x509_certificate_pem': pem,
        }
        for key_name, pem in sorted(response.iteritems())
      ],
      'timestamp': utils.datetime_to_timestamp(utils.utcnow()),
    }

  # All attempts failed, give up.
  msg = 'Failed to grab service account certs for %s (HTTP code %s)' % (
      service_account_name, result.status_code if result else '???')
  raise CertificateError(msg, transient=True)


def _use_cached_or_fetch(cache_key, fetch_cb):
  """Implements caching layer for the public certificates.

  Caches certificate in both memcache and local instance memory. Uses
  probabilistic early expiration to avoid hitting the backend from multiple
  request handlers simultaneously when cache expires.

  'fetch_cb' is expected to return a dict to be passed to CertificateBundle
  constructor.
  """
  # Try local memory first.
  now = utils.time_time()
  with _certs_cache_lock:
    if cache_key in _certs_cache:
      certs, exp = _certs_cache[cache_key]
      if exp > now + 0.1 * _CERTS_CACHE_EXP_SEC * random.random():
        return certs

  # Try memcache now. Use same trick with random early expiration.
  entry = memcache.get(cache_key)
  if entry:
    certs_dict, exp = entry
    if exp > now + 0.1 * _CERTS_CACHE_EXP_SEC * random.random():
      certs = CertificateBundle(certs_dict)
      with _certs_cache_lock:
        _certs_cache[cache_key] = (certs, now + _CERTS_CACHE_EXP_SEC)
      return certs

  # Multiple concurrent fetches are possible, but it's not a big deal. The last
  # one wins.
  certs_dict = fetch_cb()
  exp = now + _CERTS_CACHE_EXP_SEC
  memcache.set(cache_key, (certs_dict, exp), time=exp)
  certs = CertificateBundle(certs_dict)
  with _certs_cache_lock:
    _certs_cache[cache_key] = (certs, exp)
  return certs
