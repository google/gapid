# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Integration with webapp2."""

# Disable 'Method could be a function.'
# pylint: disable=R0201

import functools
import json
import logging
import uuid
import webapp2

from google.appengine.api import users

from . import api
from . import check
from . import config
from . import delegation
from . import ipaddr
from . import model
from . import tokens

# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'ApiHandler',
  'AuthenticatingHandler',
  'gae_cookie_authentication',
  'get_authenticated_routes',
  'oauth_authentication',
  'require_xsrf_token_request',
  'service_to_service_authentication',
]


def require_xsrf_token_request(f):
  """Use for handshaking APIs."""
  @functools.wraps(f)
  def hook(self, *args, **kwargs):
    if not self.request.headers.get('X-XSRF-Token-Request'):
      raise api.AuthorizationError('Missing required XSRF request header')
    return f(self, *args, **kwargs)
  return hook


class XSRFToken(tokens.TokenKind):
  """XSRF token parameters."""
  expiration_sec = 4 * 3600
  secret_key = api.SecretKey('xsrf_token')
  version = 1


class AuthenticatingHandlerMetaclass(type):
  """Ensures that 'get', 'post', etc. are marked with @require or @public."""

  def __new__(mcs, name, bases, attributes):
    for method in webapp2.WSGIApplication.allowed_methods:
      func = attributes.get(method.lower())
      if func and not api.is_decorated(func):
        raise TypeError(
            'Method \'%s\' of \'%s\' is not protected by @require or @public '
            'decorator' % (method.lower(), name))
    return type.__new__(mcs, name, bases, attributes)


class AuthenticatingHandler(webapp2.RequestHandler):
  """Base class for webapp2 request handlers that use Auth system.

  Knows how to extract Identity from request data and how to initialize auth
  request context, so that get_current_identity() and is_group_member() work.

  All request handling methods (like 'get', 'post', etc) should be marked by
  either @require or @public decorators.
  """

  # Checks that all 'get', 'post', etc. are marked with @require or @public.
  __metaclass__ = AuthenticatingHandlerMetaclass

  # List of HTTP methods that trigger XSRF token validation.
  xsrf_token_enforce_on = ('DELETE', 'POST', 'PUT')
  # If not None, the header to search for XSRF token.
  xsrf_token_header = 'X-XSRF-Token'
  # If not None, the request parameter (GET or POST) to search for XSRF token.
  xsrf_token_request_param = 'xsrf_token'
  # Embedded data extracted from XSRF token of current request.
  xsrf_token_data = None

  # If not None, sets X-Frame-Options on all replies.
  frame_options = 'DENY'

  # If true, will add 'nonce-*' to script-src CSP. Forbids inline scripts.
  csp_use_script_nonce = False
  # If true, will add 'nonce-*' to style-src CSP. Forbids inline styles.
  csp_use_style_nonce = False

  # See csp_nonce attribute.
  _csp_nonce = None

  # A method used to authenticate this request, see get_auth_methods().
  auth_method = None
  # If True, allow to use '<appid>-bots' IP whitelist to auth anonymous calls.
  use_bots_ip_whitelist = True

  def dispatch(self):
    """Extracts and verifies Identity, sets up request auth context."""
    # Ensure auth component is configured before executing any code.
    conf = config.ensure_configured()
    ctx = api.reinitialize_request_cache()

    # Set CSP header, if necessary. Subclasses may extend it or disable it.
    policy = self.get_content_security_policy()
    if policy:
      self.response.headers['Content-Security-Policy'] = '; '.join(
        str('%s %s' % (directive, ' '.join(sources)))
        for directive, sources in sorted(policy.iteritems())
      )
    # Enforce HTTPS by adding the HSTS header; 365*24*60*60s.
    # https://www.owasp.org/index.php/HTTP_Strict_Transport_Security
    self.response.headers['Strict-Transport-Security'] = (
        'max-age=31536000; includeSubDomains; preload')
    # Disable frame support wholesale.
    # https://www.owasp.org/index.php/Clickjacking_Defense_Cheat_Sheet
    if self.frame_options:
      self.response.headers['X-Frame-Options'] = self.frame_options

    peer_identity = None
    auth_details = None
    for method_func in self.get_auth_methods(conf):
      try:
        peer_identity, auth_details = method_func(self.request)
        if peer_identity:
          break
      except api.AuthenticationError as err:
        self.authentication_error(err)
        return
      except api.AuthorizationError as err:
        self.authorization_error(err)
        return
    else:
      method_func = None
    self.auth_method = method_func

    # If no authentication method is applicable, default to anonymous identity.
    if not peer_identity:
      peer_identity = model.Anonymous
      auth_details = None

    try:
      # Verify the caller is allowed to make calls from the given IP and use the
      # delegation token (if any). It raises AuthorizationError if something is
      # not allowed. Populates auth context fields.
      check.check_request(
          ctx=ctx,
          peer_identity=peer_identity,
          peer_ip=ipaddr.ip_from_string(self.request.remote_addr),
          auth_details=auth_details,
          delegation_token=self.request.headers.get(delegation.HTTP_HEADER),
          project_header=self.request.headers.get(check.X_LUCI_PROJECT),
          use_project_identites=conf.USE_PROJECT_IDENTITIES,
          use_bots_ip_whitelist=self.use_bots_ip_whitelist)

      # XSRF token is required only if using Cookie based or IP whitelist auth.
      # A browser doesn't send Authorization: 'Bearer ...' or any other headers
      # by itself. So XSRF check is not required if header based authentication
      # is used.
      using_headers_auth = method_func in (
          oauth_authentication, service_to_service_authentication)

      # Fail if XSRF token is required, but not provided.
      need_xsrf_token = (
          not using_headers_auth and
          self.request.method in self.xsrf_token_enforce_on)
      if need_xsrf_token and self.xsrf_token is None:
        raise api.AuthorizationError('XSRF token is missing')

      # If XSRF token is present, verify it is valid and extract its payload.
      # Do it even if XSRF token is not strictly required, since some handlers
      # use it to store session state (it is similar to a signed cookie).
      self.xsrf_token_data = {}
      if self.xsrf_token is not None:
        # This raises AuthorizationError if token is invalid.
        try:
          self.xsrf_token_data = self.verify_xsrf_token()
        except api.AuthorizationError as exc:
          if not need_xsrf_token:
            logging.warning('XSRF token is broken, ignoring - %s', exc)
          else:
            raise

      # All other ACL checks will be performed by corresponding handlers
      # manually or via '@required' decorator. Failed ACL check raises
      # AuthorizationError.
      super(AuthenticatingHandler, self).dispatch()
    except api.AuthenticationError as err:
      self.authentication_error(err)
    except api.AuthorizationError as err:
      self.authorization_error(err)

  @classmethod
  def get_auth_methods(cls, conf):  # pylint: disable=unused-argument
    """Returns an enumerable of functions to use to authenticate request.

    The handler will try to apply auth methods sequentially one by one by until
    it finds one that works.

    Each auth method is a function that accepts webapp2.Request and can finish
    with 3 outcomes:

    * Return (None, ...): authentication method is not applicable to that
      request and next method should be tried (for example cookie-based
      authentication is not applicable when there's no cookies).

    * Returns (Identity, AuthDetails). It means the authentication method
      is applicable and the caller is authenticated as 'Identity'. All
      additional information extracted from the credentials (like if the caller
      is a GAE-level admin) is returned through AuthDetails tuple. It can be
      None if there are no extra information.

    * Raises AuthenticationError: authentication method is applicable, but
      request contains bad credentials or invalid token, etc. For example,
      OAuth2 token is given, but it is revoked.

    A chosen auth method function will be stored in request's auth_method field.

    Args:
      conf: components.auth GAE config, see config.py.
    """
    return (
        oauth_authentication,
        gae_cookie_authentication,
        service_to_service_authentication)

  def generate_xsrf_token(self, xsrf_token_data=None):
    """Returns new XSRF token that embeds |xsrf_token_data|.

    The token is bound to current identity and is valid only when used by same
    identity.
    """
    return XSRFToken.generate(
        [api.get_current_identity().to_bytes()], xsrf_token_data)

  @property
  def xsrf_token(self):
    """Returns XSRF token passed with the request or None if missing.

    Doesn't do any validation. Use verify_xsrf_token() instead.
    """
    token = None
    if self.xsrf_token_header:
      token = self.request.headers.get(self.xsrf_token_header)
    if not token and self.xsrf_token_request_param:
      param = self.request.get_all(self.xsrf_token_request_param)
      token = param[0] if param else None
    return token

  def verify_xsrf_token(self):
    """Grabs a token from the request, validates it and extracts embedded data.

    Current identity must be the same as one used to generate the token.

    Returns:
      Whatever was passed as |xsrf_token_data| in 'generate_xsrf_token'
      method call used to generate the token.

    Raises:
      AuthorizationError if token is missing, invalid or expired.
    """
    token = self.xsrf_token
    if not token:
      raise api.AuthorizationError('XSRF token is missing')
    # Check that it was generated for the same identity.
    try:
      return XSRFToken.validate(token, [api.get_current_identity().to_bytes()])
    except tokens.InvalidTokenError as err:
      raise api.AuthorizationError(str(err))

  @property
  def csp_nonce(self):
    """Returns random nonce used in Content-Security-Policies.

    It should be used as nonce=... attribute of all inline Javascript code
    (<script> tags) if self.csp_use_script_nonce is True, and <style> tags
    if self.csp_use_style_nonce is True.

    It makes harder to inject unintended Javascript or CSS code into the page
    (one now has to extract the nonce somehow first), thus improving XSS
    protection.

    For more details see https://www.w3.org/TR/CSP2/#script-src-nonce-usage.

    Usage:
      class MyHandler(AuthenticatingHandler):
        csp_use_script_nonce = True

        def get(self):
          ...
          self.response.write(
            template.render(..., {'csp_nonce': self.csp_nonce}))

    In the template:
      <script nonce="{{csp_nonce}}">
        ...
      </script>
    """
    if not self._csp_nonce:
      self._csp_nonce = tokens.base64_encode(uuid.uuid4().bytes)
    return self._csp_nonce

  def get_content_security_policy(self):
    """Returns a dict {CSP directive (e.g. 'script-src') => list of sources}.

    The returned policy (unless empty or None) will be formated and put in
    Content-Security-Policy header. Default implementation returns a policy
    suitable for apps that depend on Google (and only Google) services.

    Called once at the start of request handler. Always returns a copy, caller
    can safely modify it.
    """
    # See:
    #   https://developers.google.com/web/fundamentals/security/csp/
    #   https://www.w3.org/TR/CSP2/
    #   https://www.owasp.org/index.php/Content_Security_Policy
    #
    # TODO(maruel): Remove 'unsafe-inline' once all inline style="foo:bar" in
    # all HTML tags were removed. Warning if seeing this post 2016, it could
    # take a while.
    csp = {
      'default-src': ["'self'"],

      'script-src': [
        "'self'",
        "'unsafe-inline'",  # fallback if the browser doesn't support nonces
        "'unsafe-eval'",    # required by Polymer and Handlebars templates

        'https://www.google-analytics.com',
        'https://www.google.com/jsapi',
        'https://apis.google.com',
        'https://www.gstatic.com', # Google charts loader
      ],

      'style-src': [
        "'self'",
        "'unsafe-inline'",  # fallback if the browser doesn't support nonces
        "https://fonts.googleapis.com",
        "https://www.gstatic.com", # Google charts styling
      ],

      'frame-src': [
        'https://accounts.google.com',  # Google OAuth2 library opens iframes
      ],

      'img-src': [
        "'self'",
        'https://www.google-analytics.com',
        'https://*.googleusercontent.com',  # Google user avatars
      ],

      'font-src': [
        "'self'",
        "https://fonts.gstatic.com",  # Google-hosted fonts
      ],

      'object-src': ["'none'"],  # we don't generally use Flash or Java
    }

    # When 'unsafe-inline' and 'nonce-*' are both specified, newer browsers
    # prefer nonces. Older browsers (that don't support nonces), fall back to
    # 'unsafe-inline' and just ignore nonces.
    if self.csp_use_script_nonce:
      csp['script-src'].append("'nonce-%s'" % self.csp_nonce)
    if self.csp_use_style_nonce:
      csp['style-src'].append("'nonce-%s'" % self.csp_nonce)

    return csp

  def authentication_error(self, error):
    """Called when authentication fails to report the error to requester.

    Authentication error means that some credentials are provided but they are
    invalid. If no credentials are provided at all, no authentication is
    attempted and current identity is just set to 'anonymous:anonymous'.

    Default behavior is to abort the request with HTTP 401 error (and human
    readable HTML body).

    Args:
      error: instance of AuthenticationError subclass.
    """
    logging.warning('Authentication error.\n%s', error)
    self.abort(401, detail=str(error))

  def authorization_error(self, error):
    """Called when authentication succeeds, but access to a resource is denied.

    Called whenever request handler raises AuthorizationError exception.
    In particular this exception is raised by method decorated with @require if
    current identity doesn't have required permission.

    Default behavior is to abort the request with HTTP 403 error (and human
    readable HTML body).

    Args:
      error: instance of AuthorizationError subclass.
    """
    logging.warning(
        'Authorization error.\n%s\nPeer: %s\nIP: %s\nOrigin: %s',
        error, api.get_peer_identity().to_bytes(), self.request.remote_addr,
        self.request.headers.get('Origin'))
    self.abort(403, detail=str(error))

  ### Wrappers around Users API or its equivalent.

  def get_current_user(self):
    """When cookie auth is used returns instance of CurrentUser or None."""
    return self._get_users_api().get_current_user(self.request)

  def create_login_url(self, dest_url):
    """When cookie auth is used returns URL to redirect user to login."""
    return self._get_users_api().create_login_url(self.request, dest_url)

  def create_logout_url(self, dest_url):
    """When cookie auth is used returns URL to redirect user to logout."""
    return self._get_users_api().create_logout_url(self.request, dest_url)

  def _get_users_api(self):
    """Returns a Users API implementation or raises NotImplementedError.

    Chooses based on what auth_method was used of what methods are available.
    """
    method = self.auth_method
    if not method:
      # Anonymous request -> pick first method that supports API.
      for method in self.get_auth_methods(config.ensure_configured()):
        if method in _METHOD_TO_USERS_API:
          break
      else:
        raise NotImplementedError('No methods support UsersAPI')
    elif method not in _METHOD_TO_USERS_API:
      raise NotImplementedError(
          '%s doesn\'t support UsersAPI' % method.__name__)
    return _METHOD_TO_USERS_API[method]


class ApiHandler(AuthenticatingHandler):
  """Parses JSON request body to a dict, serializes response to JSON."""
  CONTENT_TYPE_BASE = 'application/json'
  CONTENT_TYPE_FULL = 'application/json; charset=utf-8'
  _json_body = None

  # Clickjacking not applicable to APIs.
  frame_options = None

  # CSP is not applicable to APIs.
  def get_content_security_policy(self):
    return None

  def authentication_error(self, error):
    logging.warning('Authentication error.\n%s', error)
    self.abort_with_error(401, text=str(error))

  def authorization_error(self, error):
    logging.warning(
        'Authorization error.\n%s\nPeer: %s\nIP: %s\nOrigin: %s',
        error, api.get_peer_identity().to_bytes(), self.request.remote_addr,
        self.request.headers.get('Origin'))
    self.abort_with_error(403, text=str(error))

  def send_response(self, response, http_code=200, headers=None):
    """Sends successful reply and continues execution."""
    self.response.set_status(http_code)
    self.response.headers.update(headers or {})
    self.response.headers['Content-Type'] = self.CONTENT_TYPE_FULL
    self.response.write(json.dumps(response))

  def abort_with_error(self, http_code, **kwargs):
    """Sends error reply and stops execution."""
    logging.warning('abort_with_error(%s)', kwargs)
    self.abort(
        http_code,
        json=kwargs,
        headers={'Content-Type': self.CONTENT_TYPE_FULL})

  def parse_body(self):
    """Parses JSON body and verifies it's a dict.

    webob.Request doesn't cache the decoded json body, this function does.
    """
    if self._json_body is None:
      if (self.CONTENT_TYPE_BASE and
          self.request.content_type != self.CONTENT_TYPE_BASE):
        msg = (
            'Expecting JSON body with content type \'%s\'' %
            self.CONTENT_TYPE_BASE)
        self.abort_with_error(400, text=msg)
      try:
        self._json_body = self.request.json
        if not isinstance(self._json_body, dict):
          raise ValueError()
      except (LookupError, ValueError):
        self.abort_with_error(400, text='Not a valid json dict body')
    return self._json_body.copy()


def get_authenticated_routes(app):
  """Given WSGIApplication returns list of routes that use authentication.

  Intended to be used only for testing.
  """
  # This code is adapted from router's __repr__ method (that enumerate
  # all routes for pretty-printing).
  routes = list(app.router.match_routes)
  routes.extend(
      v for k, v in app.router.build_routes.iteritems()
      if v not in app.router.match_routes)
  return [r for r in routes if issubclass(r.handler, AuthenticatingHandler)]


################################################################################
## All supported implementations of authentication methods for webapp2 handlers.


def gae_cookie_authentication(_request):
  """AppEngine cookie based authentication via users.get_current_user()."""
  user = users.get_current_user()
  if not user:
    return None, None
  try:
    ident = model.Identity(model.IDENTITY_USER, user.email())
  except ValueError:
    raise api.AuthenticationError('Unsupported user email: %s' % user.email())
  return ident, api.new_auth_details(is_superuser=users.is_current_user_admin())


def oauth_authentication(request):
  """OAuth2 based authentication via access tokens."""
  auth_header = request.headers.get('Authorization')
  if not auth_header:
    return None, None
  return api.check_oauth_access_token(auth_header)


def service_to_service_authentication(request):
  """Used for AppEngine <-> AppEngine communication.

  Relies on X-Appengine-Inbound-Appid header set by AppEngine itself. It can't
  be set by external users (with exception of admins).
  """
  app_id = request.headers.get('X-Appengine-Inbound-Appid')
  try:
    ident = model.Identity(model.IDENTITY_SERVICE, app_id) if app_id else None
  except ValueError:
    raise api.AuthenticationError('Unsupported application ID: %s' % app_id)
  return ident, None


################################################################################
## API wrapper for generating login and logout URLs.


class CurrentUser(object):
  """Mimics subset of GAE users.User object for ease of transition.

  Also adds .picture().
  """

  def __init__(self, user_id, email, picture):
    self._user_id = user_id
    self._email = email
    self._picture = picture

  def nickname(self):
    return self._email

  def email(self):
    return self._email

  def user_id(self):
    return self._user_id

  def picture(self):
    return self._picture

  def __unicode__(self):
    return unicode(self.nickname())

  def __str__(self):
    return str(self.nickname())


class GAEUsersAPI(object):
  @staticmethod
  def get_current_user(request):  # pylint: disable=unused-argument
    user = users.get_current_user()
    return CurrentUser(user.user_id(), user.email(), None) if user else None

  @staticmethod
  def create_login_url(request, dest_url):  # pylint: disable=unused-argument
    return users.create_login_url(dest_url)

  @staticmethod
  def create_logout_url(request, dest_url):  # pylint: disable=unused-argument
    return users.create_logout_url(dest_url)


# See AuthenticatingHandler._get_users_api().
_METHOD_TO_USERS_API = {
  gae_cookie_authentication: GAEUsersAPI,
}
