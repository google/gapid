# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Integration with Cloud Endpoints.

This module is used only when 'endpoints' is importable (see auth/__init__.py).
"""

import functools
import logging

import endpoints

from protorpc import message_types
from protorpc import util

from . import api
from . import check
from . import config
from . import delegation
from . import ipaddr
from . import model

from components import utils

# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'endpoints_api',
  'endpoints_method',
]


# TODO(vadimsh): id_token auth (used when talking to Cloud Endpoints from
# Android for example) is not supported yet, since this module talks to
# OAuth API directly to validate client_ids to simplify usage of Cloud Endpoints
# APIs by service accounts. Otherwise, each service account (or rather it's
# client_id) has to be hardcoded into the application source code.


# Cloud Endpoints auth library likes to spam logging.debug(...) messages: four
# messages per _every_ authenticated request. Monkey patch it.
from endpoints import users_id_token
users_id_token.logging = logging.getLogger('endpoints.users_id_token')
users_id_token.logging.setLevel(logging.INFO)


# Reduce the verbosity of messages dumped by _ah/spi/BackendService.logMessages.
# Otherwise ereporter2 catches them one by one, generating email for each
# individual error. See endpoints.api_backend_service.BackendServiceImpl.
def monkey_patch_endpoints_logger():
  logger = logging.getLogger('endpoints.api_backend_service')
  original = logger.handle
  def patched_handle(record):
    if record.levelno >= logging.ERROR:
      record.levelno = logging.WARNING
      record.levelname = logging.getLevelName(record.levelno)
    return original(record)
  logger.handle = patched_handle
monkey_patch_endpoints_logger()


@util.positional(2)
def endpoints_api(
    name, version,
    auth_level=None,
    allowed_client_ids=None,
    **kwargs):
  """Same as @endpoints.api but tweaks default auth related properties.

  By default API marked with this decorator will use same authentication scheme
  as non-endpoints request handlers (i.e. fetch a whitelist of OAuth client_id's
  from the datastore, recognize service accounts, etc.), disabling client_id
  checks performed by Cloud Endpoints frontend (and doing them on the backend,
  see 'initialize_auth' below).

  Using service accounts with vanilla Cloud Endpoints auth is somewhat painful:
  every service account should be whitelisted in the 'allowed_client_ids' list
  in the source code of the application (when calling @endpoints.api). By moving
  client_id checks to the backend we can support saner logic.
  """
  # 'audiences' is used with id_token auth, it's not supported yet.
  assert 'audiences' not in kwargs, 'Not supported'

  # On prod, make sure Cloud Endpoints frontend validates OAuth tokens for us.
  # On dev instances we will validate them ourselves to support custom token
  # validation endpoint.
  if auth_level is not None:
    if utils.is_local_dev_server() or utils.is_dev():
      # AUTH_LEVEL.NONE: Frontend authentication will be skipped. If
      # authentication is desired, it will need to be performed by the backend.
      auth_level = endpoints.AUTH_LEVEL.NONE
    else:
      # AUTH_LEVEL.OPTIONAL: Authentication is optional. If authentication
      # credentials are supplied they must be valid. Backend will be called if
      # the request contains valid authentication credentials or no
      # authentication credentials.
      auth_level = endpoints.AUTH_LEVEL.OPTIONAL

  # We love API Explorer.
  if allowed_client_ids is None:
    allowed_client_ids = endpoints.SKIP_CLIENT_ID_CHECK
  if allowed_client_ids != endpoints.SKIP_CLIENT_ID_CHECK:
    allowed_client_ids = sorted(
        set(allowed_client_ids) | set([endpoints.API_EXPLORER_CLIENT_ID]))

  # Someone was looking for job security here:
  # - api() returns _ApiDecorator class instance.
  # - One of the following is done:
  #   - _ApiDecorator.__call__() is called with the remote.Service class as
  #     argument.
  #   - api_class() is explicitly called which returns a function, which is then
  #     called with the  remote.Service class as argument.
  api_decorator = endpoints.api(
      name, version,
      auth_level=auth_level,
      allowed_client_ids=allowed_client_ids,
      **kwargs)

  def fn(cls):
    if not cls.all_remote_methods():
      raise TypeError(
          'Service %s must have at least one auth.endpoints_method method' %
          name)
    for method, func in cls.all_remote_methods().iteritems():
      if func and not api.is_decorated(func.remote._RemoteMethodInfo__method):
        raise TypeError(
            'Method \'%s\' of \'%s\' is not protected by @require or @public '
            'decorator' % (method, name))
    return cls

  # Monkey patch api_decorator to make 'api_class' to return wrapped decorator.
  orig = api_decorator.api_class
  def patched_api_class(*args, **kwargs):
    wrapper = orig(*args, **kwargs)
    return lambda cls: fn(wrapper(cls))
  api_decorator.api_class = patched_api_class

  return api_decorator


def endpoints_method(
    request_message=message_types.VoidMessage,
    response_message=message_types.VoidMessage,
    **kwargs):
  """Same as @endpoints.method but also adds auth state initialization code.

  Also forbids changing auth parameters on per-method basis, since it
  unnecessary complicates authentication code. All methods inherit properties
  set on the service level.
  """
  assert 'audiences' not in kwargs, 'Not supported'
  assert 'allowed_client_ids' not in kwargs, 'Not supported'

  # @endpoints.method wraps a method with a call that sets up state for
  # endpoints.get_current_user(). It does a bunch of checks and eventually calls
  # oauth.get_client_id(). oauth.get_client_id() times out a lot and we want to
  # retry RPC on deadline exceptions a bunch of times. To do so we rely on the
  # fact that oauth.get_client_id() (and other functions in oauth module)
  # essentially caches a result of RPC call to OAuth service in os.environ. So
  # we call it ourselves (with retries) to cache the state in os.environ before
  # giving up control to @endpoints.method. That's what @initialize_oauth
  # decorator does.

  def new_decorator(func):
    @initialize_oauth
    @endpoints.method(request_message, response_message, **kwargs)
    @functools.wraps(func)
    def wrapper(service, *args, **kwargs):
      try:
        initialize_request_auth(
            service.request_state.remote_address,
            service.request_state.headers)
        return func(service, *args, **kwargs)
      except endpoints.BadRequestException as e:
        # Useful to debug HTTP 400s.
        logging.warning('%s', e, exc_info=True)
        raise
      except api.AuthenticationError as ex:
        logging.warning(
            'Authentication error.\n%s\nPeer: %s\nIP: %s\nOrigin: %s',
            ex.message, api.get_peer_identity().to_bytes(),
            service.request_state.remote_address,
            service.request_state.headers.get('Origin'))
        raise endpoints.UnauthorizedException(ex.message)
      except api.AuthorizationError as ex:
        logging.warning(
            'Authorization error.\n%s\nPeer: %s\nIP: %s\nOrigin: %s',
            ex.message, api.get_peer_identity().to_bytes(),
            service.request_state.remote_address,
            service.request_state.headers.get('Origin'))
        raise endpoints.ForbiddenException(ex.message)
    return wrapper
  return new_decorator


def initialize_oauth(method):
  """Initializes OAuth2 state before calling wrapped @endpoints.method.

  Used to retry deadlines in GetOAuthUser RPCs before diving into Endpoints code
  that doesn't care about retries.

  TODO(vadimsh): This call is unnecessary if id_token is used instead of
  access_token. We do not use id_tokens currently.
  """
  @functools.wraps(method)
  def wrapper(service, *args, **kwargs):
    if service.request_state.headers.get('Authorization'):
      # See _maybe_set_current_user_vars in endpoints/users_id_token.py.
      scopes = (
          method.method_info.scopes
          if method.method_info.scopes is not None
          else service.api_info.scopes)
      # GAE OAuth module uses internal cache for OAuth RCP responses. The cache
      # key, unfortunately, is basically str(scopes), and a single scope passed
      # as a string (Endpoints lib does that) and a list with one scope only
      # have different cache keys (even though RCPs are identical). So do what
      # Endpoints lib does to warm the cache for it.
      scopes = scopes[0] if len(scopes) == 1 else scopes
      api.attempt_oauth_initialization(scopes)
    return method(service, *args, **kwargs)
  return wrapper


def initialize_request_auth(remote_address, headers):
  """Grabs caller identity and initializes request local authentication context.

  Called before executing a cloud endpoints method. May raise AuthorizationError
  or AuthenticationError exceptions.
  """
  conf = config.ensure_configured()
  ctx = api.reinitialize_request_cache()

  # Verify the validity of the token (including client_id check), if given.
  auth_details = None
  auth_header = headers.get('Authorization')
  if auth_header:
    peer_identity, auth_details = api.check_oauth_access_token(auth_header)
  else:
    # Cloud Endpoints support more authentication methods than we do. Make sure
    # to fail the request if one of such methods is used.
    if endpoints.get_current_user() is not None:
      raise api.AuthenticationError('Unsupported authentication method')
    peer_identity = model.Anonymous

  # Verify the caller is allowed to make calls from the given IP and use the
  # delegation token (if any). It raises AuthorizationError if something is
  # not allowed. Populates auth context fields.
  check.check_request(
      ctx=ctx,
      peer_identity=peer_identity,
      peer_ip=ipaddr.ip_from_string(remote_address),
      auth_details=auth_details,
      delegation_token=headers.get(delegation.HTTP_HEADER),
      project_header=headers.get(check.X_LUCI_PROJECT),
      use_project_identites=conf.USE_PROJECT_IDENTITIES,
      use_bots_ip_whitelist=True)
