#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Disable 'Unused variable', 'Unused argument' and 'Method could be a function'.
# pylint: disable=W0612,W0613,R0201

import json
import os
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import webapp2
import webtest

from components import utils
from components.auth import api
from components.auth import check
from components.auth import delegation
from components.auth import handler
from components.auth import ipaddr
from components.auth import model
from components.auth import testing
from components.auth import tokens
from components.auth.proto import delegation_pb2
from test_support import test_case


class AuthenticatingHandlerMetaclassTest(test_case.TestCase):
  """Tests for AuthenticatingHandlerMetaclass."""

  def test_good(self):
    # No request handling methods defined at all.
    class TestHandler1(handler.AuthenticatingHandler):
      def some_other_method(self):
        pass

    # @public is used.
    class TestHandler2(handler.AuthenticatingHandler):
      @api.public
      def get(self):
        pass

    # @require is used.
    class TestHandler3(handler.AuthenticatingHandler):
      @api.require(lambda: True)
      def get(self):
        pass

  def test_bad(self):
    # @public or @require is missing.
    with self.assertRaises(TypeError):
      class TestHandler1(handler.AuthenticatingHandler):
        def get(self):
          pass


class AuthenticatingHandlerTest(testing.TestCase):
  """Tests for AuthenticatingHandler class."""

  # pylint: disable=unused-argument

  def make_test_app(self, path, request_handler):
    """Returns webtest.TestApp with single route."""
    return webtest.TestApp(
        webapp2.WSGIApplication([(path, request_handler)], debug=True),
        extra_environ={'REMOTE_ADDR': '127.0.0.1'})

  def make_test_app_with_peer(self, peer_ident):
    """Returns a callback that calls a handler through webtest.TestApp.

    Calls are authenticated as if coming from 'peer_ident'.

    Returns:
      call(headers) callback.
    """
    class Handler(handler.AuthenticatingHandler):
      @classmethod
      def get_auth_methods(cls, conf):
        return [lambda _request: (model.Identity.from_bytes(peer_ident), None)]
      @api.public
      def get(self):
        self.response.write(json.dumps({
          'peer_id': api.get_peer_identity().to_bytes(),
          'cur_id': api.get_current_identity().to_bytes(),
        }))

    app = self.make_test_app('/request', Handler)
    def call(headers=None):
      resp = app.get('/request', headers=headers, expect_errors=True)
      return {
        'status': resp.status_int,
        'body': json.loads(resp.body) if resp.status_int == 200 else resp.body,
      }
    return call

  def test_anonymous(self):
    """If all auth methods are not applicable, identity is set to Anonymous."""
    test = self

    class Handler(handler.AuthenticatingHandler):
      @classmethod
      def get_auth_methods(cls, conf):
        non_applicable = lambda _request: (None, None)
        return [non_applicable, non_applicable]

      @api.public
      def get(self):
        test.assertEqual(model.Anonymous, api.get_current_identity())
        self.response.write('OK')

    app = self.make_test_app('/request', Handler)
    self.assertEqual('OK', app.get('/request').body)

  def test_ip_whitelist_bot(self):
    """Requests from client in bots IP whitelist are authenticated as bot."""
    model.bootstrap_ip_whitelist(
        model.bots_ip_whitelist(), ['192.168.1.100/32'])

    class Handler(handler.AuthenticatingHandler):
      @api.public
      def get(self):
        self.response.write(api.get_current_identity().to_bytes())

    app = self.make_test_app('/request', Handler)
    def call(ip):
      api.reset_local_state()
      return app.get('/request', extra_environ={'REMOTE_ADDR': ip}).body

    self.assertEqual('bot:whitelisted-ip', call('192.168.1.100'))
    self.assertEqual('anonymous:anonymous', call('127.0.0.1'))

  def test_ip_whitelist_bot_disabled(self):
    """Same as test_ip_whitelist_bot, but IP whitelist auth is disabled."""
    model.bootstrap_ip_whitelist(
        model.bots_ip_whitelist(), ['192.168.1.100/32'])

    class Handler(handler.AuthenticatingHandler):
      use_bots_ip_whitelist = False
      @api.public
      def get(self):
        self.response.write(api.get_current_identity().to_bytes())

    app = self.make_test_app('/request', Handler)
    def call(ip):
      api.reset_local_state()
      return app.get('/request', extra_environ={'REMOTE_ADDR': ip}).body

    self.assertEqual('anonymous:anonymous', call('192.168.1.100'))

  def test_ip_whitelist(self):
    """Per-account IP whitelist works."""
    ident1 = model.Identity(model.IDENTITY_USER, 'a@example.com')
    ident2 = model.Identity(model.IDENTITY_USER, 'b@example.com')

    model.bootstrap_ip_whitelist('whitelist', ['192.168.1.100/32'])
    model.bootstrap_ip_whitelist_assignment(ident1, 'whitelist')

    mocked_ident = [None]

    class Handler(handler.AuthenticatingHandler):
      @classmethod
      def get_auth_methods(cls, conf):
        return [lambda _req: (mocked_ident[0], None)]

      @api.public
      def get(self):
        self.response.write('OK')

    app = self.make_test_app('/request', Handler)
    def call(ident, ip):
      api.reset_local_state()
      mocked_ident[0] = ident
      response = app.get(
          '/request', extra_environ={'REMOTE_ADDR': ip}, expect_errors=True)
      return response.status_int

    # IP is whitelisted.
    self.assertEqual(200, call(ident1, '192.168.1.100'))
    # IP is NOT whitelisted.
    self.assertEqual(403, call(ident1, '127.0.0.1'))
    # Whitelist is not used.
    self.assertEqual(200, call(ident2, '127.0.0.1'))

  def test_auth_method_order(self):
    """Registered auth methods are tested in order."""
    test = self
    calls = []
    ident = model.Identity(model.IDENTITY_USER, 'joe@example.com')
    auth_details = api.new_auth_details()

    def not_applicable(request):
      self.assertEqual('/request', request.path)
      calls.append('not_applicable')
      return None, None

    def applicable(request):
      self.assertEqual('/request', request.path)
      calls.append('applicable')
      return ident, auth_details

    class Handler(handler.AuthenticatingHandler):
      @classmethod
      def get_auth_methods(cls, conf):
        return [not_applicable, applicable]

      @api.public
      def get(self):
        test.assertEqual(ident, api.get_current_identity())
        test.assertIs(auth_details, api.get_auth_details())
        self.response.write('OK')

    app = self.make_test_app('/request', Handler)
    self.assertEqual('OK', app.get('/request').body)

    # Both methods should be tried.
    expected_calls = [
      'not_applicable',
      'applicable',
    ]
    self.assertEqual(expected_calls, calls)

  def test_authentication_error(self):
    """AuthenticationError in auth method stops request processing."""
    test = self
    calls = []

    def failing(request):
      raise api.AuthenticationError('Too bad')

    def skipped(request):
      self.fail('authenticate should not be called')

    class Handler(handler.AuthenticatingHandler):
      @classmethod
      def get_auth_methods(cls, conf):
        return [failing, skipped]

      @api.public
      def get(self):
        test.fail('Handler code should not be called')

      def authentication_error(self, err):
        test.assertEqual('Too bad', err.message)
        calls.append('authentication_error')
        # pylint: disable=bad-super-call
        super(Handler, self).authentication_error(err)

    app = self.make_test_app('/request', Handler)
    response = app.get('/request', expect_errors=True)

    # Custom error handler is called and returned HTTP 401.
    self.assertEqual(['authentication_error'], calls)
    self.assertEqual(401, response.status_int)

    # Authentication error is logged.
    self.assertEqual(1, len(self.logged_warnings))

  def test_authorization_error(self):
    """AuthorizationError in auth method is handled."""
    test = self
    calls = []

    class Handler(handler.AuthenticatingHandler):
      @api.require(lambda: False)
      def get(self):
        test.fail('Handler code should not be called')

      def authorization_error(self, err):
        calls.append('authorization_error')
        # pylint: disable=bad-super-call
        super(Handler, self).authorization_error(err)

    app = self.make_test_app('/request', Handler)
    response = app.get('/request', expect_errors=True)

    # Custom error handler is called and returned HTTP 403.
    self.assertEqual(['authorization_error'], calls)
    self.assertEqual(403, response.status_int)

  def make_xsrf_handling_app(
      self,
      xsrf_token_enforce_on=None,
      xsrf_token_header=None,
      xsrf_token_request_param=None):
    """Returns webtest app with single XSRF-aware handler.

    If generates XSRF tokens on GET and validates them on POST, PUT, DELETE.
    """
    calls = []

    def record(request_handler, method):
      is_valid = request_handler.xsrf_token_data == {'some': 'data'}
      calls.append((method, is_valid))

    class Handler(handler.AuthenticatingHandler):
      @api.public
      def get(self):
        self.response.write(self.generate_xsrf_token({'some': 'data'}))
      @api.public
      def post(self):
        record(self, 'POST')
      @api.public
      def put(self):
        record(self, 'PUT')
      @api.public
      def delete(self):
        record(self, 'DELETE')

    if xsrf_token_enforce_on is not None:
      Handler.xsrf_token_enforce_on = xsrf_token_enforce_on
    if xsrf_token_header is not None:
      Handler.xsrf_token_header = xsrf_token_header
    if xsrf_token_request_param is not None:
      Handler.xsrf_token_request_param = xsrf_token_request_param

    app = self.make_test_app('/request', Handler)
    return app, calls

  def mock_get_current_identity(self, ident):
    """Mocks api.get_current_identity() to return |ident|."""
    self.mock(handler.api, 'get_current_identity', lambda: ident)

  def test_xsrf_token_get_param(self):
    """XSRF token works if put in GET parameters."""
    app, calls = self.make_xsrf_handling_app()
    token = app.get('/request').body
    app.post('/request?xsrf_token=%s' % token)
    self.assertEqual([('POST', True)], calls)

  def test_xsrf_token_post_param(self):
    """XSRF token works if put in POST parameters."""
    app, calls = self.make_xsrf_handling_app()
    token = app.get('/request').body
    app.post('/request', {'xsrf_token': token})
    self.assertEqual([('POST', True)], calls)

  def test_xsrf_token_header(self):
    """XSRF token works if put in the headers."""
    app, calls = self.make_xsrf_handling_app()
    token = app.get('/request').body
    app.post('/request', headers={'X-XSRF-Token': token})
    self.assertEqual([('POST', True)], calls)

  def test_xsrf_token_missing(self):
    """XSRF token is not given but handler requires it."""
    app, calls = self.make_xsrf_handling_app()
    response = app.post('/request', expect_errors=True)
    self.assertEqual(403, response.status_int)
    self.assertFalse(calls)

  def test_xsrf_token_uses_enforce_on(self):
    """Only methods set in |xsrf_token_enforce_on| require token validation."""
    # Validate tokens only on PUT (not on POST).
    app, calls = self.make_xsrf_handling_app(xsrf_token_enforce_on=('PUT',))
    token = app.get('/request').body
    # Both POST and PUT work when token provided, verifying it.
    app.post('/request', {'xsrf_token': token})
    app.put('/request', {'xsrf_token': token})
    self.assertEqual([('POST', True), ('PUT', True)], calls)
    # POST works without a token, put PUT doesn't.
    self.assertEqual(200, app.post('/request').status_int)
    self.assertEqual(403, app.put('/request', expect_errors=True).status_int)
    # Only the one that requires the token fails if wrong token is provided.
    bad_token = {'xsrf_token': 'boo'}
    self.assertEqual(200, app.post('/request', bad_token).status_int)
    self.assertEqual(
        403, app.put('/request', bad_token, expect_errors=True).status_int)

  def test_xsrf_token_uses_xsrf_token_header(self):
    """Name of the header used for XSRF can be changed."""
    app, calls = self.make_xsrf_handling_app(xsrf_token_header='X-Some')
    token = app.get('/request').body
    app.post('/request', headers={'X-Some': token})
    self.assertEqual([('POST', True)], calls)

  def test_xsrf_token_uses_xsrf_token_request_param(self):
    """Name of the request param used for XSRF can be changed."""
    app, calls = self.make_xsrf_handling_app(xsrf_token_request_param='tok')
    token = app.get('/request').body
    app.post('/request', {'tok': token})
    self.assertEqual([('POST', True)], calls)

  def test_xsrf_token_identity_matters(self):
    app, calls = self.make_xsrf_handling_app()
    # Generate token for identity A.
    self.mock_get_current_identity(
        model.Identity(model.IDENTITY_USER, 'a@example.com'))
    token = app.get('/request').body
    # Try to use it by identity B.
    self.mock_get_current_identity(
        model.Identity(model.IDENTITY_USER, 'b@example.com'))
    response = app.post('/request', {'tok': token}, expect_errors=True)
    self.assertEqual(403, response.status_int)
    self.assertFalse(calls)

  def test_get_authenticated_routes(self):
    class Authenticated(handler.AuthenticatingHandler):
      pass

    class NotAuthenticated(webapp2.RequestHandler):
      pass

    app = webapp2.WSGIApplication([
      webapp2.Route('/authenticated', Authenticated),
      webapp2.Route('/not-authenticated', NotAuthenticated),
    ])
    routes = handler.get_authenticated_routes(app)
    self.assertEqual(1, len(routes))
    self.assertEqual(Authenticated, routes[0].handler)

  def test_get_peer_ip(self):
    class Handler(handler.AuthenticatingHandler):
      @api.public
      def get(self):
        self.response.write(ipaddr.ip_to_string(api.get_peer_ip()))

    app = self.make_test_app('/request', Handler)
    response = app.get('/request', extra_environ={'REMOTE_ADDR': '192.1.2.3'})
    self.assertEqual('192.1.2.3', response.body)

  def test_delegation_token(self):
    call = self.make_test_app_with_peer('user:peer@a.com')

    # No delegation.
    self.assertEqual({
      'status': 200,
      'body': {
        u'cur_id': u'user:peer@a.com',
        u'peer_id': u'user:peer@a.com',
      },
    }, call())

    # Grab a fake-signed delegation token.
    subtoken = delegation_pb2.Subtoken(
        delegated_identity='user:delegated@a.com',
        kind=delegation_pb2.Subtoken.BEARER_DELEGATION_TOKEN,
        audience=['*'],
        services=['*'],
        creation_time=int(utils.time_time()),
        validity_duration=3600)
    tok_pb = delegation_pb2.DelegationToken(
        serialized_subtoken=subtoken.SerializeToString(),
        signer_id='user:token-server@example.com',
        signing_key_id='signing-key',
        pkcs1_sha256_sig='fake-signature')
    tok = tokens.base64_encode(tok_pb.SerializeToString())

    # With valid delegation token.
    self.assertEqual({
      'status': 200,
      'body': {
        u'cur_id': u'user:delegated@a.com',
        u'peer_id': u'user:peer@a.com',
      },
    }, call({'X-Delegation-Token-V1': tok}))

    # With invalid delegation token.
    resp = call({'X-Delegation-Token-V1': tok+'blah'})
    self.assertEqual(403, resp['status'])
    self.assertIn('Bad delegation token', resp['body'])

    # Transient error.
    def mocked_check(*_args):
      raise delegation.TransientError('Blah')
    self.mock(delegation, 'check_bearer_delegation_token', mocked_check)
    resp = call({'X-Delegation-Token-V1': tok})
    self.assertEqual(500, resp['status'])
    self.assertIn('Blah', resp['body'])

  def test_x_luci_project_works(self):
    self.mock_group(check.LUCI_SERVICES_GROUP, ['user:peer@a.com'])
    call = self.make_test_app_with_peer('user:peer@a.com')

    # No header -> authenticated as is.
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    self.assertEqual({
      'status': 200,
      'body': {
        u'cur_id': u'user:peer@a.com',
        u'peer_id': u'user:peer@a.com',
      },
    }, call({}))

    # With header, but X-Luci-Project auth is off -> authenticated as is.
    self.mock_config(USE_PROJECT_IDENTITIES=False)
    self.assertEqual({
      'status': 200,
      'body': {
        u'cur_id': u'user:peer@a.com',
        u'peer_id': u'user:peer@a.com',
      },
    }, call({check.X_LUCI_PROJECT: 'proj-name'}))

    # With header and X-Luci-Project auth is on -> authenticated as project.
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    self.assertEqual({
      'status': 200,
      'body': {
        u'cur_id': u'project:proj-name',
        u'peer_id': u'user:peer@a.com',
      },
    }, call({check.X_LUCI_PROJECT: 'proj-name'}))

  def test_x_luci_project_from_unrecognized_service(self):
    self.mock_config(USE_PROJECT_IDENTITIES=True)
    call = self.make_test_app_with_peer('user:peer@a.com')
    resp = call({check.X_LUCI_PROJECT: 'proj-name'})
    self.assertEqual(401, resp['status'])
    self.assertIn(
        'Usage of X-Luci-Project is not allowed for user:peer@a.com: not a '
        'member of auth-luci-services group', resp['body'])

  def test_x_luci_project_with_delegation_token(self):
    call = self.make_test_app_with_peer('user:peer@a.com')
    resp = call({
      check.X_LUCI_PROJECT: 'proj-name',
      'X-Delegation-Token-V1': 'tok',
    })
    self.assertEqual(401, resp['status'])
    self.assertIn(
        'Delegation tokens and X-Luci-Project cannot be used together',
        resp['body'])


class GaeCookieAuthenticationTest(test_case.TestCase):
  """Tests for gae_cookie_authentication function."""

  def test_non_applicable(self):
    self.assertEqual(
        (None, None),
        handler.gae_cookie_authentication(webapp2.Request({})))

  def test_applicable_non_admin(self):
    os.environ.update({
      'USER_EMAIL': 'joe@example.com',
      'USER_ID': '123',
      'USER_IS_ADMIN': '0',
    })
    # Actual request is not used by CookieAuthentication.
    self.assertEqual(
        (
          model.Identity(model.IDENTITY_USER, 'joe@example.com'),
          api.new_auth_details(is_superuser=False),
        ),
        handler.gae_cookie_authentication(webapp2.Request({})))

  def test_applicable_admin(self):
    os.environ.update({
      'USER_EMAIL': 'joe@example.com',
      'USER_ID': '123',
      'USER_IS_ADMIN': '1',
    })
    # Actual request is not used by CookieAuthentication.
    self.assertEqual(
        (
          model.Identity(model.IDENTITY_USER, 'joe@example.com'),
          api.new_auth_details(is_superuser=True),
        ),
        handler.gae_cookie_authentication(webapp2.Request({})))


class ServiceToServiceAuthenticationTest(test_case.TestCase):
  """Tests for service_to_service_authentication."""

  def test_non_applicable(self):
    request = webapp2.Request({})
    self.assertEqual(
        (None, None),
        handler.service_to_service_authentication(request))

  def test_applicable(self):
    request = webapp2.Request({
      'HTTP_X_APPENGINE_INBOUND_APPID': 'some-app',
    })
    self.assertEqual(
      (model.Identity(model.IDENTITY_SERVICE, 'some-app'), None),
      handler.service_to_service_authentication(request))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
