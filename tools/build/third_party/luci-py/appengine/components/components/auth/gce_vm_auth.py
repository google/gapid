# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Implements authentication based on signed GCE VM metadata tokens.

See https://cloud.google.com/compute/docs/instances/verifying-instance-identity.

JWTs with signed metadata are read from X-Luci-Gce-Vm-Token header. The 'aud'
field in the tokens is expected to be https://(.*-dot-)?<app-id>.appspot.com.

On successful validation, the bot is authenticated as
  bot:<instance-name>@gce.<project>[.<realm>]

Additional details are then available via get_auth_details():
  gce_instance - an instance name extracted from the token, as is.
  gce_project  - a project name extracted from the token, as is.

Prefer to use get_auth_details() for authorization checks instead of parsing
"bot:..." identifier, since there's less chance of a mistake that way.
"""

import logging
import re

from . import api
from . import model
from . import signature
from . import tokens

from components import utils

from google.appengine.api import app_identity


# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'gce_vm_authentication',
  'optional_gce_vm_authentication',
]


# HTTP header that carries the GCE VM token.
GCE_VM_TOKEN_HEADER = 'X-Luci-Gce-Vm-Token'


class BadTokenError(api.AuthenticationError):
  """Raised if the supplied GCE VM token is not valid."""


def gce_vm_authentication(request):
  """Reads and validates X-Luci-Gce-Vm-Token header, if present.

  See components.auth.handler.AuthenticatingHandler.get_auth_methods for details
  of the expected interface.

  Args:
    request: webapp2.Request with the incoming request.

  Returns:
    (auth.Identity, AuthDetails) on success.
    (None, None) if there's no VM token header (which means this authentication
        method is not applicable).

  Raises:
    BadTokenError (which is api.AuthenticationError) if VM token header is
    present, but the token is invalid.
    CertificateError on transient errors when fetching google certs.
  """
  token = request.headers.get(GCE_VM_TOKEN_HEADER)
  if not token:
    return None, None

  # Fetch (most likely already cached) Google OAuth2 certs.
  certs = signature.get_google_oauth2_certs()

  # Make sure the JWT is signed by Google, and not yet expired.
  try:
    _, payload = tokens.verify_jwt(token, certs)
  except (signature.CertificateError, tokens.InvalidTokenError) as exc:
    raise BadTokenError('Invalid GCE VM token: %s' % exc)

  # The valid payload looks like this:
  # {
  #    "iss": "[TOKEN_ISSUER]",
  #    "iat": [ISSUED_TIME],
  #    "exp": [EXPIRED_TIME],
  #    "aud": "[AUDIENCE]",
  #    "sub": "[SUBJECT]",
  #    "azp": "[AUTHORIZED_PARTY]",
  #    "google": {
  #     "compute_engine": {
  #       "project_id": "[PROJECT_ID]",
  #       "project_number": [PROJECT_NUMBER],
  #       "zone": "[ZONE]",
  #       "instance_id": [INSTANCE_ID],
  #       "instance_name": "[INSTANCE_NAME]"
  #       "instance_creation_timestamp": [CREATION_TIMESTAMP]
  #     }
  #   }
  # }

  # Verify the token was intended for us.
  allowed = _allowed_audience_re()
  aud = str(payload.get('aud', ''))
  if not allowed.match(aud):
    raise BadTokenError(
        'Bad audience in GCE VM token: got %r, expecting %r' %
        (aud, allowed.pattern))

  # The token should have 'google.compute_engine' field, which happens only if
  # it was generated with format=full.
  gce = payload.get('google', {}).get('compute_engine')
  if not gce:
    raise BadTokenError(
        'No google.compute_engine in the GCE VM token, use "full" format')
  if not isinstance(gce, dict):
    raise BadTokenError('Wrong type for compute_engine: %r' % (gce,))

  instance_name = gce.get('instance_name')
  if not isinstance(instance_name, basestring):
    raise BadTokenError('Wrong type for instance_name: %r' % (instance_name,))
  project_id = gce.get('project_id')
  if not isinstance(project_id, basestring):
    raise BadTokenError('Wrong type for project_id: %r' % (project_id,))
  details = api.new_auth_details(
      gce_instance=str(instance_name),
      gce_project=str(project_id))

  # Convert '<realm>:<project>' to '<project>.<realm>' for bot:... string.
  domain = details.gce_project
  if ':' in domain:
    realm, proj = domain.split(':', 1)
    domain = '%s.%s' % (proj, realm)

  # The token is valid. Construct and validate bot identity.
  try:
    ident = model.Identity(
        model.IDENTITY_BOT, '%s@gce.%s' % (details.gce_instance, domain))
  except ValueError as exc:
    raise BadTokenError(str(exc))
  return ident, details


def optional_gce_vm_authentication(request):
  """It's like gce_vm_authentication except it ignores broken tokens.

  Usable during development and initial roll out when GCE VM tokens may not
  be working.
  """
  try:
    return gce_vm_authentication(request)
  except BadTokenError as exc:
    logging.error('Skipping GCE VM auth, it returned an error: %s', exc)
    return None, None


@utils.cache
def _allowed_audience_re():
  """Returns a regular expression for allowed 'aud' field."""
  return _audience_re(app_identity.get_default_version_hostname())


# Extracted into a separate function for simpler testing.
def _audience_re(hostname):
  return re.compile(r'^https\://([a-z0-9\-_]+-dot-)?'+re.escape(hostname)+'$')
