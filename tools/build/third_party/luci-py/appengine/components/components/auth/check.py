# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines high level authorization function called for all incoming requests.

Lives in its own module to avoid introducing module dependency cycle between
api.py and delegation.py.
"""

from . import api
from . import delegation
from . import model


# A group with service accounts of LUCI microservices belonging to the current
# LUCI deployment (and only them!).
#
# Accounts in this group are allowed to use 'X-Luci-Project' header to specify
# that RPCs are done in a context of some particular project. For such requests
# get_current_identity() == Identity('project:<X-Luci-Project value>').
#
# This group should contain only **fully trusted** services, deployed and
# managed by the LUCI deployment administrators. Adding "random" services here
# is a security risk, since they will be able to impersonate any LUCI project.
LUCI_SERVICES_GROUP = 'auth-luci-services'

# A header with a value of 'project_header' for check_request.
X_LUCI_PROJECT = 'X-Luci-Project'


def check_request(
    ctx,
    peer_identity,
    peer_ip,
    auth_details,
    delegation_token,
    project_header,
    use_project_identites,
    use_bots_ip_whitelist):
  """Prepares the request context, checking IP whitelist and delegation token.

  This is intended to be called by request processing middlewares right after
  they have authenticated the peer, and before they dispatch the request to the
  actual handler.

  It checks IP whitelist, and delegation token, and updates the request auth
  context accordingly, populating peer_identity, peer_ip, current_identity and
  auth_details fields.

  Args:
    ctx: instance of api.RequestCache to update.
    peer_identity: caller's identity.
    peer_ip: instance of ipaddr.IP.
    auth_details: api.AuthDetails tuple (or None) with additional auth info.
    delegation_token: the token from X-Delegation-Token-V1 header.
    project_header: a value of X-Luci-Project header (or None).
    use_project_identites: True to allow authenticating requests as coming
      from 'project:...' identities (based on X-Luci-Project header).
    use_bots_ip_whitelist: [DEPRECATED] if true, treat anonymous access from
      IPs in "<appid>-bots" whitelist as coming from "bot:whitelisted-ip"
      identity.

  Raises:
    api.AuthenticationError if the request use incompatible headers.
    api.AuthorizationError if identity has an IP whitelist assigned and given IP
    address doesn't belong to it.
    delegation.TransientError if there was a transient error checking the token.
  """
  auth_db = ctx.auth_db

  # The peer can either use a project identity, or delegate someone else's
  # identity or don't do any of that at all. But never both, it is ambiguous.
  if delegation_token and project_header:
    raise api.AuthenticationError(
        'Delegation tokens and %s cannot be used together' % X_LUCI_PROJECT)

  # Hack to allow pure IP-whitelist based authentication for bots, until they
  # are switched to use something better.
  #
  # TODO(vadimsh): Get rid of this. Blocked on killing IP whitelisted access
  # from Chrome Buildbot machines.
  if (use_bots_ip_whitelist and peer_identity.is_anonymous and
      auth_db.is_in_ip_whitelist(model.bots_ip_whitelist(), peer_ip, False)):
      peer_identity = model.IP_WHITELISTED_BOT_ID

  # Note: populating fields early is useful, since exception handlers may use
  # them for logging.
  ctx.peer_ip = peer_ip
  ctx.peer_identity = peer_identity

  # Verify the caller is allowed to make calls from the given IP. It raises
  # AuthorizationError if IP is not allowed.
  auth_db.verify_ip_whitelisted(peer_identity, peer_ip)

  if delegation_token:
    # Parse the delegation token to deduce end-user identity. We clear
    # auth_details if the delegation is used, since it no longer applies to
    # the delegated identity.
    try:
      ident, unwrapped_tok = delegation.check_bearer_delegation_token(
          delegation_token, peer_identity, auth_db)
      ctx.current_identity = ident
      ctx.delegation_token = unwrapped_tok
    except delegation.BadTokenError as exc:
      raise api.AuthorizationError('Bad delegation token: %s' % exc)
  elif use_project_identites and project_header:
    # X-Luci-Project header can be used only by LUCI services (which we
    # completely trust). Other callers must not provide it (most likely this
    # indicates a misconfiguration).
    if not auth_db.is_group_member(LUCI_SERVICES_GROUP, peer_identity):
      raise api.AuthenticationError(
          'Usage of %s is not allowed for %s: not a member of %s group' %
          (X_LUCI_PROJECT, peer_identity.to_bytes(), LUCI_SERVICES_GROUP))
    ctx.current_identity = _project_identity(project_header)
  else:
    ctx.current_identity = ctx.peer_identity
    ctx.auth_details = auth_details


def _project_identity(proj):
  """Returns model.Identity representing the given project.

  Raises:
    api.AuthenticationError if the project name is malformed.
  """
  try:
    return model.Identity(model.IDENTITY_PROJECT, proj)
  except ValueError as exc:
    raise api.AuthenticationError('Bad %s: %s' % (X_LUCI_PROJECT, exc))
