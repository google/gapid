# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Google Compute Engine specific utility functions."""

import base64
import json
import logging
import re
import socket
import threading
import time
import urllib
import urllib2

from api import oauth
from utils import tools


### Private stuff.


# One-tuple that contains cached metadata, as returned by get_metadata.
_CACHED_METADATA = None


# Cache of GCE OAuth2 token.
_CACHED_OAUTH2_TOKEN = {}
_CACHED_OAUTH2_TOKEN_LOCK = threading.Lock()


# Cached signed GCE VM metadata (keyed by the audience URL).
_CACHED_METADATA_TOKEN = {}
_CACHED_METADATA_TOKEN_LOCK = threading.Lock()


def _padded_b64_decode(data):
  mod = len(data) % 4
  if mod:
    data += '=' * (4 - mod)
  return base64.urlsafe_b64decode(data)


def _raw_metadata_request(path):
  """Sends a request to metadata.google.internal, retrying on errors.

  Returns raw response as str or None if not running on GCE or if the metadata
  server is unreachable (after multiple attempts). Logs errors internally.

  Args:
    path: path on the metadata server to query ('/computeMetadata/...').
  """
  assert path.startswith('/'), path
  if not is_gce():
    logging.info('GCE metadata is not available: not on GCE')
    return None
  url = 'http://metadata.google.internal' + path
  headers = {'Metadata-Flavor': 'Google'}
  for i in xrange(0, 10):
    time.sleep(i*2)
    try:
      resp = urllib2.urlopen(urllib2.Request(url, headers=headers), timeout=10)
      return resp.read()
    except IOError as e:
      logging.warning(
          'Failed to grab GCE metadata from %s on attempt #%d: %s',
          path, i+1, e)
  return None


## Public API.


@tools.cached
def is_gce():
  """Returns True if running on Google Compute Engine."""
  # Attempting to hit the metadata server is unreliable since it sometimes
  # doesn't respond. Resolving DNS name is more stable, since it actually just
  # looks into /etc/hosts file (at least on Linux).
  #
  # See also https://github.com/GoogleCloudPlatform/gcloud-golang/issues/194.
  try:
    # 169.254.169.254 is the documented metadata server IP address.
    return socket.gethostbyname('metadata.google.internal') == '169.254.169.254'
  except socket.error:
    return False


def get_metadata_uncached():
  """Returns the GCE metadata as a dict.

  Returns None if not running on GCE or if metadata server is unreachable.

  Refs:
    https://cloud.google.com/compute/docs/metadata
    https://cloud.google.com/compute/docs/machine-types

  To get the at the command line from a GCE VM, use:
    curl --silent \
      http://metadata.google.internal/computeMetadata/v1/?recursive=true \
      -H "Metadata-Flavor: Google" | python -m json.tool | less
  """
  raw = _raw_metadata_request('/computeMetadata/v1/?recursive=true')
  return json.loads(raw) if raw else None


def wait_for_metadata(quit_bit):
  """Spins until GCE metadata server responds.

  Precaches value of 'get_metadata'.

  Args:
    quit_bit: its 'is_set' method is used to break from the loop earlier.
  """
  global _CACHED_METADATA
  if not is_gce():
    return
  while not quit_bit.is_set():
    meta = get_metadata_uncached()
    if meta is not None:
      _CACHED_METADATA = (meta,)
      break


def get_metadata():
  """Cached version of get_metadata_uncached()"""
  global _CACHED_METADATA
  if _CACHED_METADATA is not None:
    return _CACHED_METADATA[0]
  meta = get_metadata_uncached()
  # Don't cache 'None' reply on GCE. It is wrong. Better to try to grab the
  # right one next time.
  if meta is not None or not is_gce():
    _CACHED_METADATA = (meta,)
    return meta
  return None


def oauth2_access_token_with_expiration(account):
  """Returns tuple (oauth2 access token, expiration timestamp).

  Args:
    account: GCE service account to use (e.g "default").
  """
  # TODO(maruel): Move GCE VM authentication logic into client/utils/net.py.
  # As seen in google-api-python-client/oauth2client/gce.py
  with _CACHED_OAUTH2_TOKEN_LOCK:
    cached_tok = _CACHED_OAUTH2_TOKEN.get(account)
    # Cached and expires in more than 5 min from now.
    if cached_tok and cached_tok['expiresAt'] >= time.time() + 5*60:
      return cached_tok['accessToken'], cached_tok['expiresAt']
    # Grab the token.
    url = (
        'http://metadata.google.internal/computeMetadata/v1/instance'
        '/service-accounts/%s/token' % account)
    headers = {'Metadata-Flavor': 'Google'}
    access_token, expires_at = oauth.oauth2_access_token_from_url(url, headers)
    tok = {
      'accessToken': access_token,
      'expiresAt': expires_at,
    }
    _CACHED_OAUTH2_TOKEN[account] = tok
    return access_token, expires_at


def oauth2_access_token(account='default'):
  """Returns a value of oauth2 access token.

  Same as oauth2_access_token_with_expiration, but discards expiration and
  assumed 'default' account. Exists for backward compatibility with various
  hooks.
  """
  return oauth2_access_token_with_expiration(account)[0]


def oauth2_available_scopes(account='default'):
  """Returns a list of OAuth2 scopes granted to GCE service account."""
  metadata = get_metadata()
  if not metadata:
    return []
  accounts = metadata['instance']['serviceAccounts']
  return accounts.get(account, {}).get('scopes') or []


def signed_metadata_token(audience):
  """Returns tuple (signed GCE metadata JWT, expiration timestamp).

  Returns (None, None) if not running on GCE or the metadata server is
  unreachable or misbehaves. More details are available in the log.

  The returned token has at least 5 min of lifetime left.

  Args:
    audience: audience URL to put into the token (usually Swarming server URL).
  """
  with _CACHED_METADATA_TOKEN_LOCK:
    cached_tok = _CACHED_METADATA_TOKEN.get(audience)
    if cached_tok and cached_tok['exp'] >= time.time() + 5*60:
      return cached_tok['jwt'], cached_tok['exp']

    jwt = _raw_metadata_request(
        '/computeMetadata/v1/instance/service-accounts/default/'
        'identity?audience=%s&format=full' % urllib.quote_plus(audience))
    if not jwt:
      return None, None

    # Extract lifetime of the token from JWT's payload (the second segment).
    # Use now+(expiry-issued_at) to account for possible clock skew between GCE
    # backend clock and our machine's clock (expiry and issued_at are expressed
    # in GCE backend clock time, but we want to compare the expiry to our
    # clock). We assume that issued_at ('iat') in the token matches current
    # timestamp.
    #
    # Note that we don't do any strict validation here, since we trust the
    # metadata server on the bot. Swarming backend does full validation though,
    # so if metadata server returns something crazy, we'll know on the server
    # side.
    try:
      payload = json.loads(_padded_b64_decode(jwt.split('.')[1]))
      exp = int(time.time()) + (payload['exp'] - payload['iat'])
    except (IndexError, KeyError, TypeError, ValueError) as exc:
      logging.error('Metadata server returned invalid JWT (%s):\n%s', exc, jwt)
      return None, None

    _CACHED_METADATA_TOKEN[audience] = {'jwt': jwt, 'exp': exp}
    return jwt, exp


@tools.cached
def get_image():
  """Returns the image used by the GCE VM."""
  # Format is projects/<id>/global/images/<image>
  metadata = get_metadata()
  return unicode(metadata['instance']['image'].rsplit('/', 1)[-1])


@tools.cached
def get_zone():
  """Returns the zone containing the GCE VM."""
  # Format is projects/<id>/zones/<zone>
  metadata = get_metadata()
  return unicode(metadata['instance']['zone'].rsplit('/', 1)[-1])


@tools.cached
def get_zones():
  """Returns the list of zone values for the GCE VM."""
  # Ref: https://cloud.google.com/compute/docs/regions-zones/
  # 'continent-regionN-L' where N is a number and L a letter.
  m = re.match(ur'([a-z]+)-([a-z]+)(\d)-([a-z])', get_zone())
  if not m:
    # Safety fallback in case a new form appears.
    return [get_zone()]
  return [
      m.group(1),
      m.group(1) + '-' + m.group(2),
      m.group(1) + '-' + m.group(2) + m.group(3),
      m.group(1) + '-' + m.group(2) + m.group(3) + '-' + m.group(4),
  ]


@tools.cached
def get_machine_type():
  """Returns the GCE machine type."""
  # Format is projects/<id>/machineTypes/<machine_type>
  metadata = get_metadata()
  return unicode(metadata['instance']['machineType'].rsplit('/', 1)[-1])


@tools.cached
def get_cpuinfo():
  """Returns the GCE CPU information as reported by GCE instance metadata."""
  metadata = get_metadata()
  if not metadata:
    return None
  cpu_platform = unicode(metadata['instance']['cpuPlatform'])
  if not cpu_platform:
    return None
  # Normalize according to the expected name as reported by the CPUID
  # instruction. Sadly what GCE metadata reports is a 'creative' name that bears
  # no relation with what CPUID reports.
  #
  # Fake it based on the expected names reported at
  # https://cloud.google.com/compute/docs/instances/specify-min-cpu-platform#availablezones
  #
  # TODO: Update once we get other platforms. We don't know in advance what
  # 'creative' names will be used so better to fail explicitly and fix it right
  # away.
  assert cpu_platform.startswith(u'Intel '), cpu_platform
  return {
    u'name': u'Intel(R) Xeon(R) CPU %s GCE' % cpu_platform[len(u'Intel '):],
    u'vendor': u'GenuineIntel',
  }


@tools.cached
def get_tags():
  """Returns a list of instance tags or empty list if not GCE VM."""
  return get_metadata()['instance']['tags']


@tools.cached
def can_send_metric(service_account):
  """True if 'send_metric' really does something."""
  # Scope to use Cloud Monitoring.
  scope = 'https://www.googleapis.com/auth/monitoring'
  return scope in oauth2_available_scopes(account=service_account)
