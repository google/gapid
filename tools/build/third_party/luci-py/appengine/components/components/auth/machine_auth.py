# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Implements authentication based on LUCI machine tokens.

LUCI machine tokens are short lived signed protobuf blobs that (among other
information) contain machines' FQDNs.

Each machine has a TLS certificate (and corresponding private key) it uses
to authenticate to LUCI token server when periodically refreshing machine
tokens. Other LUCI backends then simply verifies that the short lived machine
token was signed by the trusted LUCI token server key. That way all the
complexities of dealing with PKI (like checking for certificate revocation) are
implemented in a dedicated LUCI token server, not in the each individual
service.

See:
  * https://github.com/luci/luci-go/tree/master/appengine/cmd/tokenserver
  * https://github.com/luci/luci-go/tree/master/client/cmd/luci_machine_tokend
  * https://github.com/luci/luci-go/tree/master/server/auth/machine
"""

import base64
import logging

from google.protobuf import message

from components import utils

from . import api
from . import model
from . import signature
from .proto import machine_token_pb2


# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'BadTokenError',
  'TransientError',
  'machine_authentication',
  'optional_machine_authentication',
]


# HTTP header that carries the machine token.
MACHINE_TOKEN_HEADER = 'X-Luci-Machine-Token'

# Name of a group with trusted token servers. This group should contain service
# account emails of token servers we trust.
TOKEN_SERVERS_GROUP = 'auth-token-servers'

# How much clock difference we tolerate.
ALLOWED_CLOCK_DRIFT_SEC = 10

# For how long to cache the certificates of the token server in the memory.
CERTS_CACHE_EXP_SEC = 60 * 60


class BadTokenError(api.AuthenticationError):
  """Raised if the supplied machine token is not valid.

  See app logs for more details.
  """

  def __init__(self):
    super(BadTokenError, self).__init__('Bad machine token')


class TransientError(Exception):
  """Raised on transient errors.

  Supposed to trigger HTTP 500 response.
  """


def machine_authentication(request):
  """Implementation of the machine authentication.

  See components.auth.handler.AuthenticatingHandler.get_auth_methods for details
  of the expected interface.

  Args:
    request: webapp2.Request with the incoming request.

  Returns:
    (auth.Identity, None) with machine ID ("bot:<fqdn>") on success or
    (None, None) if there's no machine token header (which means this
    authentication method is not applicable).

  Raises:
    BadTokenError (which is api.AuthenticationError) if machine token header is
    present, but the token is invalid. We also log the error details, but return
    only generic error message to the user.
  """
  token = request.headers.get(MACHINE_TOKEN_HEADER)
  if not token:
    return None, None

  # Deserialize both envelope and the body.
  try:
    token = b64_decode(token)
  except TypeError as exc:
    log_error(request, None, exc, 'Failed to decode base64')
    raise BadTokenError()

  try:
    envelope = machine_token_pb2.MachineTokenEnvelope()
    envelope.MergeFromString(token)
    body = machine_token_pb2.MachineTokenBody()
    body.MergeFromString(envelope.token_body)
  except message.DecodeError as exc:
    log_error(request, None, exc, 'Failed to deserialize the token')
    raise BadTokenError()

  # Construct an identity of a token server that signed the token to check that
  # it belongs to "auth-token-servers" group.
  try:
    signer_service_account = model.Identity.from_bytes('user:' + body.issued_by)
  except ValueError as exc:
    log_error(request, body, exc, 'Bad issued_by field - %s', body.issued_by)
    raise BadTokenError()

  # Reject tokens from unknown token servers right away.
  if not api.is_group_member(TOKEN_SERVERS_GROUP, signer_service_account):
    log_error(request, body, None, 'Unknown token issuer - %s', body.issued_by)
    raise BadTokenError()

  # Check the expiration time before doing any heavier checks.
  now = utils.time_time()
  if now < body.issued_at - ALLOWED_CLOCK_DRIFT_SEC:
    log_error(request, body, None, 'The token is not yet valid')
    raise BadTokenError()
  if now > body.issued_at + body.lifetime + ALLOWED_CLOCK_DRIFT_SEC:
    log_error(request, body, None, 'The token has expired')
    raise BadTokenError()

  # Check the token was actually signed by the server.
  try:
    certs = signature.get_service_account_certificates(body.issued_by)
    is_valid_sig = certs.check_signature(
        blob=envelope.token_body,
        key_name=envelope.key_id,
        signature=envelope.rsa_sha256)
    if not is_valid_sig:
      log_error(request, body, None, 'Bad signature')
      raise BadTokenError()
  except signature.CertificateError as exc:
    if exc.transient:
      raise TransientError(str(exc))
    log_error(
        request, body, exc, 'Unexpected error when checking the signature')
    raise BadTokenError()

  # The token is valid. Construct the bot identity.
  try:
    ident = model.Identity.from_bytes('bot:' + body.machine_fqdn)
  except ValueError as exc:
    log_error(request, body, exc, 'Bad machine_fqdn - %s', body.machine_fqdn)
    raise BadTokenError()

  # Unfortunately 'bot:*' identity namespace is shared between token-based
  # identities and old IP-whitelist based identity. They shouldn't intersect,
  # but better to enforce this.
  if ident == model.IP_WHITELISTED_BOT_ID:
    log_error(request, body, None, 'Bot ID %s is forbidden', ident.to_bytes())
    raise BadTokenError()

  return ident, None


def optional_machine_authentication(request):
  """It's like machine_authentication except it ignores broken tokens.

  Usable during development and initial roll out when machine tokens may not
  be working all the time.
  """
  try:
    return machine_authentication(request)
  except BadTokenError:
    return None, None # error details are already logged


def b64_decode(data):
  """Decodes standard unpadded base64 encoded string."""
  mod = len(data) % 4
  if mod:
    data += '=' * (4 - mod)
  return base64.b64decode(data)


def log_error(request, token_body, exc, msg, *args):
  """Logs details about the request and the token, along with error message."""
  lines = [('machine_auth: ' + msg) % args]
  if exc:
    lines.append('  exception: %s (%s)' % (exc, exc.__class__.__name__))
  if request:
    lines.append('  remote_addr: %s' % request.remote_addr)
  if token_body:
    lines.extend([
      '  machine_fqdn: %s' % token_body.machine_fqdn,
      '  issued_by: %s' % token_body.issued_by,
      '  issued_at: %s' % token_body.issued_at,
      '  now: %s' % int(utils.time_time()), # for comparison with issued_at
      '  lifetime: %s' % token_body.lifetime,
      '  ca_id: %s' % token_body.ca_id,
      '  cert_sn: %s' % token_body.cert_sn,
    ])
  logging.warning('\n'.join(lines))
