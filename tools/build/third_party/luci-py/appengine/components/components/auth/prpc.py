# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines pRPC server interceptor that initializes auth context."""

import logging

from components import prpc

from . import api
from . import check
from . import config
from . import delegation
from . import ipaddr
from . import model


# Part of public API of 'auth' component, exposed by this module.
__all__ = ['prpc_interceptor']


def prpc_interceptor(request, context, call_details, continuation):
  """Initializes the auth context and catches auth exceptions.

  Validates Authorization header, delegation tokens and checks IP whitelist.
  On success updates the auth context in the thread-local storage. This makes
  various components.auth functions work from inside pRPC handlers.

  Args:
    request: deserialized request message.
    context: an instance of prpc.ServicerContext.
    call_details: an instance of prpc.HandlerCallDetails.
    continuation: a callback that resumes the processing.
  """
  try:
    peer_ip = _parse_rpc_peer(context.peer())
  except ValueError as exc:
    context.set_code(prpc.StatusCode.INTERNAL)
    context.set_details(
        'Could not parse peer IP "%s": %s' % (context.peer(), exc))
    logging.error('Could not parse peer IP "%s": %s', context.peer(), exc)
    return

  metadata = call_details.invocation_metadata
  try:
    _prepare_auth_context(metadata, peer_ip)
    return continuation(request, context, call_details)
  except api.AuthenticationError as exc:
    _log_auth_error('Authentication error', exc, metadata, peer_ip)
    context.set_code(prpc.StatusCode.UNAUTHENTICATED)
    context.set_details(exc.message)
  except api.AuthorizationError as exc:
    _log_auth_error('Authorization error', exc, metadata, peer_ip)
    context.set_code(prpc.StatusCode.PERMISSION_DENIED)
    context.set_details(exc.message)


### Private stuff.


# Keys to look up in the metadata. Must be lowercase.
_AUTHORIZATION_METADATA_KEY = 'authorization'
_DELEGATION_METADATA_KEY = delegation.HTTP_HEADER.lower()
_X_LUCI_PROJECT_METADATA_KEY = check.X_LUCI_PROJECT.lower()


def _parse_rpc_peer(rpc_peer):
  """Parses RPC peer identifier into ipaddr.IP struct.

  Raises:
    ValueError if rpc_peer is malformed.
  """
  if rpc_peer.startswith('ipv4:'):
    ip_str = rpc_peer[len('ipv4:'):]
  elif rpc_peer.startswith('ipv6:'):
    ip_str = rpc_peer[len('ipv6:'):].strip('[]')
  else:
    raise ValueError('unrecognized RPC peer ID scheme')
  return ipaddr.ip_from_string(ip_str)


def _grab_metadata(metadata, key):
  """Searches for a metadata value given a key, first one wins."""
  for k, v in metadata:
    if k == key:
      return v
  return None


def _prepare_auth_context(metadata, peer_ip):
  """Initializes authentication context for the thread.

  Args:
    metadata: RPC invocation metadata, as a list of (k, v) pairs.
    peer_ip: ipaddr.IP with the peer IP address.

  Raises:
    api.AuthenticationError if the authentication token is malformed.
    api.AuthorizationError if the caller is not in the IP whitelist or not
      authorized to use the delegation token.
  """
  conf = config.ensure_configured()
  ctx = api.reinitialize_request_cache()

  # Verify the OAuth token (including client_id check), if given.
  auth_details = None
  auth_header = _grab_metadata(metadata, _AUTHORIZATION_METADATA_KEY)
  if auth_header:
    peer_identity, auth_details = api.check_oauth_access_token(auth_header)
  else:
    peer_identity = model.Anonymous

  # Verify the caller is allowed to make calls from the given IP and use the
  # delegation token (if any). It raises AuthorizationError if something is
  # not allowed. Populates auth context fields.
  check.check_request(
      ctx=ctx,
      peer_identity=peer_identity,
      peer_ip=peer_ip,
      auth_details=auth_details,
      delegation_token=_grab_metadata(metadata, _DELEGATION_METADATA_KEY),
      project_header=_grab_metadata(metadata, _X_LUCI_PROJECT_METADATA_KEY),
      use_project_identites=conf.USE_PROJECT_IDENTITIES,
      use_bots_ip_whitelist=True)


def _log_auth_error(title, exc, metadata, peer_ip):
  """Logs an authentication or authorization error to the log (as warning).

  Args:
    title: the title of the error.
    exc: the corresponding exception.
    metadata: RPC invocation metadata, as a list of (k, v) pairs.
    peer_ip: ipaddr.IP with the peer IP address.
  """
  logging.warning(
      '%s.\n%s\nPeer: %s\nIP: %s\nOrigin: %s',
      title, exc.message,
      api.get_peer_identity().to_bytes(),
      ipaddr.ip_to_string(peer_ip),
      _grab_metadata(metadata, 'origin') or '<unknown>')
