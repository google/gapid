#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# Disable 'Method could be a function.'
# pylint: disable=R0201

import json
import logging
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import webapp2
import webtest

from google.appengine.ext import ndb

from components import utils
from components.auth import api
from components.auth import handler
from components.auth import model
from components.auth import version
from components.auth.ui import acl
from components.auth.ui import rest_api
from components.auth.ui import ui
from test_support import test_case


def call_get(request_handler, uri=None, **kwargs):
  """Calls request_handler's 'get' in a context of webtest app."""
  uri = uri or '/dummy_path'
  assert uri.startswith('/')
  path = uri.rsplit('?', 1)[0]
  app = webtest.TestApp(
      webapp2.WSGIApplication([(path, request_handler)], debug=True),
      extra_environ={'REMOTE_ADDR': '127.0.0.1'})
  return app.get(uri, **kwargs)


def call_post(request_handler, body, uri=None, **kwargs):
  """Calls request_handler's 'post' in a context of webtest app."""
  uri = uri or '/dummy_path'
  assert uri.startswith('/')
  path = uri.rsplit('?', 1)[0]
  app = webtest.TestApp(
      webapp2.WSGIApplication([(path, request_handler)], debug=True),
      extra_environ={'REMOTE_ADDR': '127.0.0.1'})
  return app.post(uri, body, **kwargs)


def make_xsrf_token(identity=model.Anonymous, xsrf_token_data=None):
  """Returns XSRF token that can be used in tests."""
  # See handler.AuthenticatingHandler.generate
  return handler.XSRFToken.generate([identity.to_bytes()], xsrf_token_data)


def get_auth_db_rev():
  """Returns current version of AuthReplicationState.auth_db_rev."""
  ent = model.replication_state_key().get()
  return 0 if not ent else ent.auth_db_rev


def mock_replication_state(primary_url):
  """Modifies AuthReplicationState to represent Replica or Standalone modes."""
  if not primary_url:
    # Convert to standalone by nuking AuthReplicationState.
    model.replication_state_key().delete()
    assert model.is_standalone()
  else:
    # Convert to replica by writing AuthReplicationState with primary_id set.
    model.AuthReplicationState(
        key=model.replication_state_key(),
        primary_id='mocked-primary',
        primary_url=primary_url).put()
    assert model.is_replica()


def make_group(name, **kwargs):
  group = model.AuthGroup(
      key=model.group_key(name),
      created_ts=utils.utcnow(),
      modified_ts=utils.utcnow(),
      **kwargs)
  group.put()
  return group


def make_ip_whitelist(name, **kwargs):
  ip_whitelist = model.AuthIPWhitelist(
      key=model.ip_whitelist_key(name),
      created_ts=utils.utcnow(),
      modified_ts=utils.utcnow(),
      **kwargs)
  ip_whitelist.put()
  return ip_whitelist


class ApiHandlerClassTest(test_case.TestCase):
  """Tests for ApiHandler base class itself."""

  def setUp(self):
    super(ApiHandlerClassTest, self).setUp()
    self.errors = []
    self.mock(handler.logging, 'error',
        lambda *args, **kwargs: self.errors.append((args, kwargs)))

  def test_authentication_error(self):
    """AuthenticationErrors are returned as JSON with status 401."""
    test = self

    def failing_auth(_request):
      raise api.AuthenticationError('Boom!')

    class Handler(handler.ApiHandler):
      @classmethod
      def get_auth_methods(cls, conf):
        return [failing_auth]

      @api.public
      def get(self):
        test.fail('Should not be called')

    response = call_get(Handler, status=401)
    self.assertEqual(
        'application/json; charset=utf-8', response.headers.get('Content-Type'))
    self.assertEqual({'text': 'Boom!'}, json.loads(response.body))

  def test_authorization_error(self):
    """AuthorizationErrors are returned as JSON with status 403."""
    class Handler(handler.ApiHandler):
      @api.public
      def get(self):
        raise api.AuthorizationError('Boom!')

    response = call_get(Handler, status=403)
    self.assertEqual(
        'application/json; charset=utf-8', response.headers.get('Content-Type'))
    self.assertEqual({'text': 'Boom!'}, json.loads(response.body))

  def test_send_response_simple(self):
    class Handler(handler.ApiHandler):
      @api.public
      def get(self):
        self.send_response({'some': 'response'})

    response = call_get(Handler, status=200)
    self.assertEqual(
        'application/json; charset=utf-8', response.headers.get('Content-Type'))
    self.assertEqual({'some': 'response'}, json.loads(response.body))

  def test_send_response_custom_status_code(self):
    """Non 200 status codes in 'send_response' work."""
    class Handler(handler.ApiHandler):
      @api.public
      def get(self):
        self.send_response({'some': 'response'}, http_code=302)

    response = call_get(Handler, status=302)
    self.assertEqual(
        'application/json; charset=utf-8', response.headers.get('Content-Type'))
    self.assertEqual({'some': 'response'}, json.loads(response.body))

  def test_send_response_custom_header(self):
    """Response headers in 'send_response' work."""
    class Handler(handler.ApiHandler):
      @api.public
      def get(self):
        self.send_response({'some': 'response'}, headers={'Some-Header': '123'})

    response = call_get(Handler, status=200)
    self.assertEqual(
        'application/json; charset=utf-8', response.headers.get('Content-Type'))
    self.assertEqual(
        '123', response.headers.get('Some-Header'))
    self.assertEqual({'some': 'response'}, json.loads(response.body))

  def test_abort_with_error(self):
    """'abort_with_error' aborts execution and returns error as JSON."""
    test = self

    class Handler(handler.ApiHandler):
      @api.public
      def get(self):
        self.abort_with_error(http_code=404, text='abc', stuff=123)
        test.fail('Should not be called')

    response = call_get(Handler, status=404)
    self.assertEqual(
        'application/json; charset=utf-8', response.headers.get('Content-Type'))
    self.assertEqual({'text': 'abc', 'stuff': 123}, json.loads(response.body))

  def test_parse_body_success(self):
    """'parse_body' successfully decodes json-encoded dict in the body."""
    test = self

    class Handler(handler.ApiHandler):
      xsrf_token_enforce_on = ()
      @api.public
      def post(self):
        test.assertEqual({'abc': 123}, self.parse_body())

    call_post(
        Handler,
        json.dumps({'abc': 123}),
        content_type='application/json; charset=utf-8',
        status=200)

  def test_parse_body_bad_content_type(self):
    """'parse_body' checks Content-Type header."""
    test = self

    class Handler(handler.ApiHandler):
      xsrf_token_enforce_on = ()
      @api.public
      def post(self):
        self.parse_body()
        test.fail('Request should have been aborted')

    call_post(
        Handler,
        json.dumps({'abc': 123}),
        content_type='application/xml; charset=utf-8',
        status=400)

  def test_parse_body_bad_json(self):
    """'parse_body' returns HTTP 400 if body is not a valid json."""
    test = self

    class Handler(handler.ApiHandler):
      xsrf_token_enforce_on = ()
      @api.public
      def post(self):
        self.parse_body()
        test.fail('Request should have been aborted')

    call_post(
        Handler,
        'not-json',
        content_type='application/json; charset=utf-8',
        status=400)

  def test_parse_body_not_dict(self):
    """'parse_body' returns HTTP 400 if body is not a json dict."""
    test = self

    class Handler(handler.ApiHandler):
      xsrf_token_enforce_on = ()
      @api.public
      def post(self):
        self.parse_body()
        test.fail('Request should have been aborted')

    call_post(
        Handler,
        '[]',
        content_type='application/json; charset=utf-8',
        status=400)

  def test_parse_body_bad_encoding(self):
    test = self

    class Handler(handler.ApiHandler):
      xsrf_token_enforce_on = ()
      @api.public
      def post(self):
        self.parse_body()
        test.fail('Request should have been aborted')

    call_post(
        Handler,
        '[]',
        content_type='application/json; charset=ascii',
        status=400)


# To be used from inside mocks.
_original_is_group_member = api.is_group_member
_original_XSRFToken_validate = handler.XSRFToken.validate


class RestAPITestCase(test_case.TestCase):
  """Test case for some concrete Auth REST API handler.

  Handler should be defined in get_rest_api_routes().
  """

  def setUp(self):
    super(RestAPITestCase, self).setUp()
    # Make webtest app that can execute REST API requests.
    self.app = webtest.TestApp(
        webapp2.WSGIApplication(rest_api.get_rest_api_routes(), debug=True),
        extra_environ={'REMOTE_ADDR': '127.0.0.1'})
    # Reset global config and cached state.
    api.reset_local_state()
    self.mocked_identity = model.Anonymous
    # Mock is_group_member checks.
    self.mocked_groups = {}
    def is_group_member_mock(group, identity=None):
      if group in self.mocked_groups:
        return self.mocked_groups[group]
      return _original_is_group_member(group, identity)
    self.mock(api, 'is_group_member', is_group_member_mock)
    # Catch errors in log.
    self.errors = []
    self.mock(handler.logging, 'error',
        lambda *args, **kwargs: self.errors.append((args, kwargs)))
    # Revision of AuthDB before the test. Used in tearDown().
    self._initial_auth_db_rev = get_auth_db_rev()
    self._auth_db_rev_inc = 0

  def tearDown(self):
    # Ensure auth_db_rev was changed expected number of times.
    try:
      self.assertEqual(
          self._initial_auth_db_rev + self._auth_db_rev_inc,
          get_auth_db_rev())
    finally:
      super(RestAPITestCase, self).tearDown()

  def expect_auth_db_rev_change(self, rev_inc=1):
    """Instruct tearDown to verify that auth_db_rev has changed."""
    self._auth_db_rev_inc += rev_inc

  def mock_current_identity(self, identity):
    """Makes api.get_current_identity() return predefined value."""
    self.mocked_identity = identity
    self.mock(api, 'get_current_identity', lambda: identity)

  def mock_is_group_member(self, group, value):
    """Mocks return value of is_group_member(group, _) call."""
    self.mocked_groups[group] = value

  def mock_is_admin(self, value):
    """Mocks value of is_admin check."""
    self.mock_is_group_member(model.ADMIN_GROUP, value)

  def mock_has_access(self, value):
    """Mocks value of has_access check."""
    self.mock_is_group_member(acl.ACCESS_GROUP_NAME, value)

  def make_request(
      self,
      method,
      path,
      params=None,
      headers=None,
      extra_environ=None,
      expect_errors=False,
      expect_xsrf_token_check=False):
    """Sends a request to the app via webtest framework.

    Args:
      method: HTTP method of a request (like 'GET' or 'POST').
      path: request path.
      params: for GET, it's a query string, for POST/PUT its a body, etc. See
          webtest docs.
      headers: a dict with request headers.
      expect_errors: True if call is expected to end with some HTTP error code.
      expect_xsrf_token_check: if True, will append X-XSRF-Token header to the
          request and will verify that request handler checked it.

    Returns:
      Tuple (status code, deserialized JSON response, response headers).
    """
    # Add XSRF header if required.
    headers = dict(headers or {})
    if expect_xsrf_token_check:
      assert 'X-XSRF-Token' not in headers
      headers['X-XSRF-Token'] = make_xsrf_token(self.mocked_identity)

    # Hook XSRF token check to verify it's called.
    xsrf_token_validate_calls = []
    def mocked_validate(*args, **kwargs):
      xsrf_token_validate_calls.append((args, kwargs))
      return _original_XSRFToken_validate(*args, **kwargs)
    self.mock(handler.XSRFToken, 'validate', staticmethod(mocked_validate))

    # Do the call. Pass |params| only if not None, otherwise use whatever
    # webtest method uses by default (for 'DELETE' it's something called
    # utils.NoDefault and it's treated differently from None).
    kwargs = {
      'expect_errors': expect_errors,
      'extra_environ': extra_environ,
      'headers': headers,
    }
    if params is not None:
      kwargs['params'] = params
    response = getattr(self.app, method.lower())(path, **kwargs)

    # Ensure XSRF token was checked.
    if expect_xsrf_token_check:
      self.assertEqual(1, len(xsrf_token_validate_calls))

    # All REST API responses should be in JSON. Even errors. If content type
    # is not application/json, then response was generated by webapp2 itself,
    # show it in error message (it's usually some sort of error page).
    self.assertEqual(
        response.headers['Content-Type'],
        'application/json; charset=utf-8',
        response)
    return response.status_int, json.loads(response.body), response.headers

  def get(self, path, **kwargs):
    """Sends GET request to REST API endpoint.

    Returns tuple (status code, deserialized JSON response, response headers).
    """
    return self.make_request('GET', path, **kwargs)

  def post(self, path, body=None, **kwargs):
    """Sends POST request to REST API endpoint.

    Returns tuple (status code, deserialized JSON response, response headers).
    """
    assert 'params' not in kwargs
    headers = dict(kwargs.pop('headers', None) or {})
    if body:
      headers['Content-Type'] = 'application/json; charset=utf-8'
    kwargs['headers'] = headers
    kwargs['params'] = json.dumps(body) if body is not None else ''
    return self.make_request('POST', path, **kwargs)

  def put(self, path, body=None, **kwargs):
    """Sends PUT request to REST API endpoint.

    Returns tuple (status code, deserialized JSON response, response headers).
    """
    assert 'params' not in kwargs
    headers = dict(kwargs.pop('headers', None) or {})
    if body:
      headers['Content-Type'] = 'application/json; charset=utf-8'
    kwargs['headers'] = headers
    kwargs['params'] = json.dumps(body) if body is not None else ''
    return self.make_request('PUT', path, **kwargs)

  def delete(self, path, **kwargs):
    """Sends DELETE request to REST API endpoint.

    Returns tuple (status code, deserialized JSON response, response headers).
    """
    return self.make_request('DELETE', path, **kwargs)


################################################################################
## Test cases for REST end points.


class SelfHandlerTest(RestAPITestCase):
  def test_anonymous(self):
    status, body, _ = self.get(
        '/auth/api/v1/accounts/self',
        extra_environ={'REMOTE_ADDR': '1.2.3.4'})
    self.assertEqual(200, status)
    self.assertEqual({
      'identity': 'anonymous:anonymous',
      'ip': '1.2.3.4',
    }, body)

  def test_non_anonymous(self):
    self.mock_current_identity(
        model.Identity(model.IDENTITY_USER, 'joe@example.com'))
    status, body, _ = self.get(
        '/auth/api/v1/accounts/self',
        extra_environ={'REMOTE_ADDR': '1.2.3.4'})
    self.assertEqual(200, status)
    self.assertEqual({
      'identity': 'user:joe@example.com',
      'ip': '1.2.3.4',
    }, body)


class XSRFHandlerTest(RestAPITestCase):
  def test_works(self):
    status, body, _ = self.post(
        path='/auth/api/v1/accounts/self/xsrf_token',
        headers={'X-XSRF-Token-Request': '1'})
    self.assertEqual(200, status)
    self.assertTrue(isinstance(body.get('xsrf_token'), basestring))

  def test_requires_header(self):
    status, body, _ = self.post(
        path='/auth/api/v1/accounts/self/xsrf_token',
        expect_errors=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Missing required XSRF request header'}, body)


class GroupsHandlerTest(RestAPITestCase):
  def test_requires_admin(self):
    status, body, _ = self.get('/auth/api/v1/groups', expect_errors=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_empty_list(self):
    self.mock_is_admin(True)
    status, body, _ = self.get('/auth/api/v1/groups')
    self.assertEqual(200, status)
    self.assertEqual({'groups': []}, body)

  def test_non_empty_list(self):
    # Freeze time in NDB's |auto_now| properties.
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    owner = model.Identity.from_bytes('user:owner@example.com')
    make_group(
        name='owners-check',
        members=[owner],
        owners='owners-check')
    make_group(name='z-external/group')

    # Create a bunch of groups with all kinds of members.
    for i in xrange(0, 5):
      make_group(
          name='Test group %d' % i,
          created_by=model.Identity.from_bytes('user:creator@example.com'),
          description='Group for testing, #%d' % i,
          modified_by=model.Identity.from_bytes('user:modifier@example.com'),
          members=[model.Identity.from_bytes('user:joe@example.com')],
          globs=[model.IdentityGlob.from_bytes('user:*@example.com')])

    # Group Listing should return all groups. Member lists should be omitted.
    # Order is alphabetical by name,
    self.mock_is_admin(True)
    status, body, _ = self.get('/auth/api/v1/groups')
    self.assertEqual(200, status)
    self.assertEqual(
      {
        u'groups': [
          {
            u'caller_can_modify': True,
            u'created_by': u'user:creator@example.com',
            u'created_ts': 1300000000000000,
            u'description': u'Group for testing, #%d' % i,
            u'modified_by': u'user:modifier@example.com',
            u'modified_ts': 1300000000000000,
            u'name': u'Test group %d' % i,
            u'owners': u'administrators',
          } for i in xrange(0, 5)
        ] + [
          {
            u'caller_can_modify': True,
            u'created_by': None,
            u'created_ts': 1300000000000000,
            u'description': u'',
            u'modified_by': None,
            u'modified_ts': 1300000000000000,
            u'name': u'owners-check',
            u'owners': u'owners-check',
          },
          {
            u'caller_can_modify': False,
            u'created_by': None,
            u'created_ts': 1300000000000000,
            u'description': u'',
            u'modified_by': None,
            u'modified_ts': 1300000000000000,
            u'name': u'z-external/group',
            u'owners': u'administrators',
          },
        ],
      }, body)

    # Check caller_can_modify for non-admin.
    self.mock_current_identity(owner)
    self.mock_is_admin(False)
    self.mock_has_access(True)
    status, body, _ = self.get('/auth/api/v1/groups')
    self.assertEqual(200, status)
    self.assertEqual(
        ['owners-check'],
        [g['name'] for g in body['groups'] if g['caller_can_modify']])


class GroupHandlerTest(RestAPITestCase):
  def setUp(self):
    super(GroupHandlerTest, self).setUp()
    # Admin group is referenced as owning group by default, and thus must exist.
    make_group(model.ADMIN_GROUP)
    self.mock_is_admin(True)

  def test_get_missing(self):
    status, body, _ = self.get(
        path='/auth/api/v1/groups/a%20group',
        expect_errors=True)
    self.assertEqual(404, status)
    self.assertEqual({'text': 'No such group'}, body)

  def test_get_existing(self):
    # Freeze time in NDB's |auto_now| properties.
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    # Create a group with all kinds of members.
    make_group(
        name='A Group',
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        description='Group for testing',
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
        members=[model.Identity.from_bytes('user:joe@example.com')],
        globs=[model.IdentityGlob.from_bytes('user:*@example.com')])

    # Fetch it via API call.
    status, body, headers = self.get(path='/auth/api/v1/groups/A%20Group')
    self.assertEqual(200, status)
    self.assertEqual(
      {
        'group': {
          'caller_can_modify': True,
          'created_by': 'user:creator@example.com',
          'created_ts': 1300000000000000,
          'description': 'Group for testing',
          'globs': ['user:*@example.com'],
          'members': ['user:joe@example.com'],
          'modified_by': 'user:modifier@example.com',
          'modified_ts': 1300000000000000,
          'name': 'A Group',
          'nested': [],
          'owners': 'administrators',
        },
      }, body)
    self.assertEqual(
        'Sun, 13 Mar 2011 07:06:40 -0000',
        headers['Last-Modified'])

  def test_get_is_using_cache(self):
    # Hit the cache first to warm it up.
    make_group('a group')
    status, _, _ = self.get(path='/auth/api/v1/groups/a%20group')
    self.assertEqual(200, status)

    # Modify the group.
    make_group(
        name='a group',
        members=[model.Identity.from_bytes('user:joe@example.com')])

    # Still serving the cached group.
    status, body, _ = self.get(path='/auth/api/v1/groups/a%20group')
    self.assertEqual(200, status)
    self.assertEqual(body['group']['members'], [])

    # Serving up-to-date group if asked to bypass the cache.
    status, body, _ = self.get(
        path='/auth/api/v1/groups/a%20group',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(200, status)
    self.assertEqual(body['group']['members'], ['user:joe@example.com'])

  def test_get_requires_admin_or_access(self):
    make_group('a group')

    # Not admin and no access => 403.
    self.mock_is_admin(False)
    status, body, _ = self.get(
        path='/auth/api/v1/groups/a%20group',
        expect_errors=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

    # Has access => 200.
    self.mock_has_access(True)
    status, _, _ = self.get(path='/auth/api/v1/groups/a%20group')
    self.assertEqual(200, status)

  def test_delete_existing(self):
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    group = make_group('A Group')

    # Delete it via API.
    self.expect_auth_db_rev_change()
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A%20Group',
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)

    # It is gone.
    self.assertFalse(model.group_key('A Group').get())

    # There's an entry in historical log.
    copy_in_history = ndb.Key(
        'AuthGroupHistory', 'A Group',
        parent=model.historical_revision_key(1))
    expected = group.to_dict()
    expected.update({
      'auth_db_prev_rev': None,
      'auth_db_rev': 1,
      'auth_db_app_version': u'v1a',
      'auth_db_deleted': True,
      'auth_db_change_comment': u'REST API',
      'modified_by': model.Identity(kind='anonymous', name='anonymous'),
      'modified_ts': utils.timestamp_to_datetime(1300000000000000),
    })
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_delete_existing_with_condition_ok(self):
    group = make_group('A Group')

    # Delete it via API using passing If-Unmodified-Since condition.
    self.expect_auth_db_rev_change()
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A%20Group',
        headers={
          'If-Unmodified-Since': utils.datetime_to_rfc2822(group.modified_ts),
        },
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)

    # It is gone.
    self.assertFalse(model.group_key('A Group').get())

  def test_delete_existing_with_condition_fail(self):
    make_group('A Group')

    # Try to delete it via API using failing If-Unmodified-Since condition.
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A%20Group',
        headers={
          'If-Unmodified-Since': 'Sun, 1 Mar 1990 00:00:00 -0000',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(412, status)
    self.assertEqual({'text': 'Group was modified by someone else'}, body)

    # It is still there.
    self.assertTrue(model.group_key('A Group').get())

  def test_delete_referenced_group(self):
    make_group('A Group')
    make_group('Another group', nested=['A Group'])

    # Try to delete it via API.
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A%20Group',
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {
          u'text':
              u'This group is being referenced by other groups: Another group.',
          u'details': {
            u'groups': [u'Another group'],
          },
        }, body)

    # It is still there.
    self.assertTrue(model.group_key('A Group').get())

  def test_delete_missing(self):
    # Unconditionally deleting a group that's not there is ok.
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A Group',
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)

  def test_delete_missing_with_condition(self):
    # Deleting missing group with condition is a error.
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A%20Group',
        headers={
          'If-Unmodified-Since': 'Sun, 1 Mar 1990 00:00:00 -0000',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(412, status)
    self.assertEqual({'text': 'Group was deleted by someone else'}, body)

  def test_delete_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A Group',
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_delete_by_owner(self):
    self.mock_current_identity(model.Identity.from_bytes('user:a@a.com'))
    self.mock_is_admin(False)
    self.mock_has_access(True)
    make_group('A Group', owners='owners')

    # Not an owner => error.
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A Group',
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual(
        body,
        {'text': u'"user:a@a.com" has no permission to delete group "A Group"'})

    # Owner => works.
    self.mock_is_group_member('owners', True)
    self.expect_auth_db_rev_change()
    status, _, _ = self.delete(
        path='/auth/api/v1/groups/A Group',
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)

  def test_delete_external_fails(self):
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/prefix/name',
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual({'text': 'This group is not writable'}, body)

  def test_delete_selfowned_group(self):
    # It is ok to delete self-owned group.
    make_group('A Group', owners='A Group')

    # Delete it via API.
    self.expect_auth_db_rev_change()
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/A%20Group',
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)

    # It is gone.
    self.assertFalse(model.group_key('A Group').get())

  def test_delete_admins_fails(self):
    status, body, _ = self.delete(
        path='/auth/api/v1/groups/' + model.ADMIN_GROUP,
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
      {
        u'text': u'Can\'t delete \'administrators\' group.',
        u'details': None,
      }, body)

  def test_post_success(self):
    frozen_time = utils.timestamp_to_datetime(1300000000000000)
    creator_identity = model.Identity.from_bytes('user:creator@example.com')

    # Freeze time in NDB's |auto_now| properties.
    self.mock_now(frozen_time)
    # get_current_identity is used for 'created_by' and 'modified_by'.
    self.mock_current_identity(creator_identity)

    make_group('Nested Group')
    make_group('Owning Group')

    # Create the group using REST API.
    self.expect_auth_db_rev_change()
    status, body, headers = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': ['user:*@example.com'],
          'members': ['bot:some-bot', 'user:some@example.com'],
          'name': 'A Group',
          'nested': ['Nested Group'],
          'owners': 'Owning Group',
        },
        expect_xsrf_token_check=True)
    self.assertEqual(201, status)
    self.assertEqual({'ok': True}, body)
    self.assertEqual(
        'Sun, 13 Mar 2011 07:06:40 -0000', headers['Last-Modified'])
    self.assertEqual(
        'http://localhost/auth/api/v1/groups/A%20Group', headers['Location'])

    # Ensure it's there and all fields are set.
    entity = model.group_key('A Group').get()
    self.assertTrue(entity)
    expected = {
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'created_by': model.Identity(kind='user', name='creator@example.com'),
      'created_ts': frozen_time,
      'description': 'Test group',
      'globs': [model.IdentityGlob(kind='user', pattern='*@example.com')],
      'members': [
        model.Identity(kind='bot', name='some-bot'),
        model.Identity(kind='user', name='some@example.com'),
      ],
      'modified_by': model.Identity(kind='user', name='creator@example.com'),
      'modified_ts': frozen_time,
      'nested': [u'Nested Group'],
      'owners': u'Owning Group',
    }
    self.assertEqual(expected, entity.to_dict())

    # Ensure it's in the revision log.
    copy_in_history = ndb.Key(
        'AuthGroupHistory', 'A Group', parent=model.historical_revision_key(1))
    expected = {
      'auth_db_app_version': u'v1a',
      'auth_db_change_comment': u'REST API',
      'auth_db_deleted': False,
    }
    expected.update(entity.to_dict())
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_post_minimal_body(self):
    # Posting just a name is enough to create an empty group.
    self.expect_auth_db_rev_change()
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group'},
        expect_xsrf_token_check=True)
    self.assertEqual(201, status)
    self.assertEqual({'ok': True}, body)

  def test_post_mismatching_name(self):
    # 'name' key and name in URL should match.
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'Another name here'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual(
        {'text': 'Missing or mismatching name in request body'}, body)

  def test_post_bad_body(self):
    # Posting invalid body ('members' should be a list, not a dict).
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group', 'members': {}},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual(
        {
          'text':
            'Expecting a list or tuple for \'members\', got \'dict\' instead',
        },
        body)

  def test_post_already_exists(self):
    make_group('A Group')

    # Now try to recreate it again via API. Should fail with HTTP 409.
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual({'text': 'Such group already exists'}, body)

  def test_post_missing_nested(self):
    # Try to create a group that references non-existing nested group.
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group', 'nested': ['Missing group']},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
      {
        u'text': u'Some referenced groups don\'t exist: Missing group.',
        u'details': {
          u'missing': [u'Missing group'],
        }
      }, body)

  def test_post_missing_owners(self):
    # Try to create a group that references non-existing owners group.
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group', 'owners': 'Missing group'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
      {
        u'text': u'Some referenced groups don\'t exist: Missing group.',
        u'details': {
          u'missing': [u'Missing group'],
        }
      }, body)

  def test_post_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_post_by_non_admin_with_access(self):
    self.mock_current_identity(model.Identity.from_bytes('user:a@a.com'))
    self.mock_is_admin(False)
    self.mock_has_access(True)
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual(
        body,
        {'text': u'"user:a@a.com" has no permission to create a group'})

  def test_post_external_fails(self):
    status, body, _ = self.post(
        path='/auth/api/v1/groups/prefix/name',
        body={'name': 'prefix/name'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual({'text': 'This group is not writable'}, body)

  def test_post_selfowned_group(self):
    # A group can be created to own itself.
    self.expect_auth_db_rev_change()
    status, body, _ = self.post(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group', 'owners': 'A Group'},
        expect_xsrf_token_check=True)
    self.assertEqual(201, status)
    self.assertEqual({'ok': True}, body)

  def test_put_success(self):
    frozen_time = utils.timestamp_to_datetime(1300000000000000)
    creator_identity = model.Identity.from_bytes('user:creator@example.com')

    # Freeze time in NDB's |auto_now| properties.
    self.mock_now(frozen_time)
    # get_current_identity is used for 'created_by' and 'modified_by'.
    self.mock_current_identity(creator_identity)

    make_group('Nested Group')
    make_group('A Group')
    make_group('Owning Group')

    # Update it via API.
    self.expect_auth_db_rev_change()
    status, body, headers = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': ['user:*@example.com'],
          'members': ['bot:some-bot', 'user:some@example.com'],
          'name': 'A Group',
          'nested': ['Nested Group'],
          'owners': 'Owning Group',
        },
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)
    self.assertEqual(
        'Sun, 13 Mar 2011 07:06:40 -0000', headers['Last-Modified'])

    # Ensure it is updated.
    entity = model.group_key('A Group').get()
    self.assertTrue(entity)
    expected = {
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'created_by': None,
      'created_ts': frozen_time,
      'description': u'Test group',
      'globs': [model.IdentityGlob(kind='user', pattern='*@example.com')],
      'members': [
        model.Identity(kind='bot', name='some-bot'),
        model.Identity(kind='user', name='some@example.com'),
      ],
      'modified_by': model.Identity(kind='user', name='creator@example.com'),
      'modified_ts': frozen_time,
      'nested': [u'Nested Group'],
      'owners': u'Owning Group',
    }
    self.assertEqual(expected, entity.to_dict())

    # Ensure it's in the revision log.
    copy_in_history = ndb.Key(
        'AuthGroupHistory', 'A Group', parent=model.historical_revision_key(1))
    expected = {
      'auth_db_app_version': u'v1a',
      'auth_db_change_comment': u'REST API',
      'auth_db_deleted': False,
    }
    expected.update(entity.to_dict())
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_put_mismatching_name(self):
    make_group('A Group')

    # Update it via API, pass bad name.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': 'Bad group name',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual(
        {'text': 'Missing or mismatching name in request body'}, body)

  def test_put_bad_body(self):
    make_group('A Group')

    # Update it via API, pass a bad body ('globs' should be a list, not a dict).
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': {},
          'members': [],
          'name': 'A Group',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual(
        {
          'text':
            'Expecting a list or tuple for \'globs\', got \'dict\' instead'
        }, body)

  def test_put_missing(self):
    # Try to update a group that doesn't exist.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': 'A Group',
          'nested': [],
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(404, status)
    self.assertEqual({'text': 'No such group'}, body)

  def test_put_bad_precondition(self):
    # Freeze time in NDB's |auto_now| properties.
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    make_group('A Group')

    # Try to update it. Pass incorrect If-Modified-Since header.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': 'A Group',
          'nested': [],
        },
        headers={
          'If-Unmodified-Since': 'Sun, 1 Mar 1990 00:00:00 -0000',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(412, status)
    self.assertEqual({'text': 'Group was modified by someone else'}, body)

  def test_put_missing_nested(self):
    make_group('A Group')

    # Try to update it. Pass missing group as a nested group.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': 'A Group',
          'nested': ['Missing group'],
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {
          u'text': u'Some referenced groups don\'t exist: Missing group.',
          u'details': {u'missing': [u'Missing group']},
        }, body)

  def test_put_missing_owners(self):
    make_group('A Group')

    # Try to update it. Pass missing group as an owning group.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': 'A Group',
          'owners': 'Missing group',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {
          u'text': u'Owners groups (Missing group) doesn\'t exist.',
          u'details': {u'missing': [u'Missing group']},
        }, body)

  def test_put_changing_admin_owners(self):
    make_group('Another group')
    status, body, _ = self.put(
        path='/auth/api/v1/groups/' + model.ADMIN_GROUP,
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': model.ADMIN_GROUP,
          'owners': 'Another group',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
      {
        u'text': u'Can\'t change owner of \'administrators\' group.',
        u'details': None
      }, body)

  def test_put_dependency_cycle(self):
    make_group('A Group')

    # Try to update it. Reference itself as a nested group.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={
          'description': 'Test group',
          'globs': [],
          'members': [],
          'name': 'A Group',
          'nested': ['A Group'],
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {
          u'text':
              u'Groups can not have cyclic dependencies: A Group -> A Group.',
          u'details': {u'cycle': [u'A Group', u'A Group']},
        }, body)

  def test_put_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_put_by_owner(self):
    self.mock_current_identity(model.Identity.from_bytes('user:a@a.com'))
    self.mock_is_admin(False)
    self.mock_has_access(True)
    make_group('A Group', owners='owners')

    # Not an owner => error.
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual(
        body,
        {'text': u'"user:a@a.com" has no permission to update group "A Group"'})

    # Owner => works.
    self.mock_is_group_member('owners', True)
    self.expect_auth_db_rev_change()
    status, _, _ = self.put(
        path='/auth/api/v1/groups/A%20Group',
        body={'name': 'A Group'},
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)

  def test_put_external_fails(self):
    status, body, _ = self.post(
        path='/auth/api/v1/groups/prefix/name',
        body={'name': 'prefix/name'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual({'text': 'This group is not writable'}, body)


class IPWhitelistsHandlerTest(RestAPITestCase):
  def setUp(self):
    super(IPWhitelistsHandlerTest, self).setUp()
    self.mock_is_admin(True)

  def test_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.get('/auth/api/v1/ip_whitelists', expect_errors=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_empty_list(self):
    status, body, _ = self.get('/auth/api/v1/ip_whitelists')
    self.assertEqual(200, status)
    self.assertEqual({'ip_whitelists': []}, body)

  def test_non_empty_list(self):
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    make_ip_whitelist(
        name='bots',
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        description='Bots whitelist',
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
        subnets=['127.0.0.1/32', '::1/128'])

    make_ip_whitelist(
        name='another whitelist',
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        description='Another whitelist',
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
        subnets=[])

    # Sorted by name. Subnets are normalized.
    status, body, _ = self.get('/auth/api/v1/ip_whitelists')
    self.assertEqual(200, status)
    self.assertEqual(
      {
        'ip_whitelists': [
          {
            'created_by': 'user:creator@example.com',
            'created_ts': 1300000000000000,
            'description': 'Another whitelist',
            'modified_by': 'user:modifier@example.com',
            'modified_ts': 1300000000000000,
            'name': 'another whitelist',
            'subnets': [],
          },
          {
            'created_by': 'user:creator@example.com',
            'created_ts': 1300000000000000,
            'description': 'Bots whitelist',
            'modified_by': 'user:modifier@example.com',
            'modified_ts': 1300000000000000,
            'name': 'bots',
            'subnets': ['127.0.0.1/32', '0:0:0:0:0:0:0:1/128'],
          },
        ],
      }, body)


class IPWhitelistHandlerTest(RestAPITestCase):
  # Test cases here are very similar to GroupHandlerTest. If something seems
  # cryptic, look up corresponding test in GroupHandlerTest, it is usually more
  # commented.

  def setUp(self):
    super(IPWhitelistHandlerTest, self).setUp()
    self.mock_is_admin(True)

  def test_get_missing(self):
    status, body, _ = self.get(
        path='/auth/api/v1/ip_whitelists/some_whitelist',
        expect_errors=True)
    self.assertEqual(404, status)
    self.assertEqual({'text': 'No such ip whitelist'}, body)

  def test_get_existing(self):
    # Works even if config modifications are forbidden.
    self.mock(rest_api, 'is_config_locked', lambda: True)
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    make_ip_whitelist(
        name='bots',
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        description='Bots whitelist',
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
        subnets=['127.0.0.1/32', '::1/128'])

    status, body, headers = self.get(path='/auth/api/v1/ip_whitelists/bots')
    self.assertEqual(200, status)
    self.assertEqual(
      {
        'ip_whitelist': {
          'created_by': 'user:creator@example.com',
          'created_ts': 1300000000000000,
          'description': 'Bots whitelist',
          'modified_by': 'user:modifier@example.com',
          'modified_ts': 1300000000000000,
          'name': 'bots',
          'subnets': ['127.0.0.1/32', '0:0:0:0:0:0:0:1/128'],
        },
      }, body)
    self.assertEqual(
        'Sun, 13 Mar 2011 07:06:40 -0000',
        headers['Last-Modified'])

  def test_get_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.get(
        path='/auth/api/v1/ip_whitelists/bots',
        expect_errors=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_delete_existing(self):
    frozen_time = utils.timestamp_to_datetime(1300000000000000)
    self.mock_now(frozen_time)
    ent = make_ip_whitelist('A whitelist')
    self.expect_auth_db_rev_change()
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)
    self.assertFalse(model.ip_whitelist_key('A whitelist').get())
    copy_in_history = ndb.Key(
        'AuthIPWhitelistHistory', 'A whitelist',
        parent=model.historical_revision_key(1))
    expected = ent.to_dict()
    expected.update({
      'auth_db_rev': 1,
      'auth_db_app_version': u'v1a',
      'auth_db_change_comment': u'REST API',
      'auth_db_deleted': True,
      'modified_by': model.Identity(kind='anonymous', name='anonymous'),
      'modified_ts': frozen_time,
    })
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_delete_existing_with_condition_ok(self):
    ent = make_ip_whitelist('A whitelist')
    self.expect_auth_db_rev_change()
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        headers={
          'If-Unmodified-Since': utils.datetime_to_rfc2822(ent.modified_ts),
        },
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)
    self.assertFalse(model.ip_whitelist_key('A whitelist').get())

  def test_delete_existing_with_condition_fail(self):
    make_ip_whitelist('A whitelist')
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        headers={
          'If-Unmodified-Since': 'Sun, 1 Mar 1990 00:00:00 -0000',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(412, status)
    self.assertEqual(
        {'text': 'Ip whitelist was modified by someone else'}, body)
    self.assertTrue(model.ip_whitelist_key('A whitelist').get())

  def test_delete_missing(self):
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)

  def test_delete_missing_with_condition(self):
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        headers={
          'If-Unmodified-Since': 'Sun, 1 Mar 1990 00:00:00 -0000',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(412, status)
    self.assertEqual({'text': 'Ip whitelist was deleted by someone else'}, body)

  def test_delete_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_delete_assigned_whitelist(self):
    # TODO(vadimsh): Add the test once implemented, see TODO in
    # IPWhitelistHandler.do_delete.
    pass

  def test_delete_when_config_locked(self):
    self.mock(rest_api, 'is_config_locked', lambda: True)
    status, body, _ = self.delete(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {'text': 'The configuration is managed elsewhere'}, body)

  def test_post_success(self):
    frozen_time = utils.timestamp_to_datetime(1300000000000000)
    self.mock_now(frozen_time)
    creator_identity = model.Identity.from_bytes('user:creator@example.com')
    self.mock_current_identity(creator_identity)

    self.expect_auth_db_rev_change()
    status, body, headers = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={
          'description': 'Test whitelist',
          'subnets': ['127.0.0.1/32'],
          'name': 'A whitelist',
        },
        expect_xsrf_token_check=True)
    self.assertEqual(201, status)
    self.assertEqual({'ok': True}, body)
    self.assertEqual(
        'Sun, 13 Mar 2011 07:06:40 -0000', headers['Last-Modified'])
    self.assertEqual(
        'http://localhost/auth/api/v1/ip_whitelists/A%20whitelist',
        headers['Location'])

    entity = model.ip_whitelist_key('A whitelist').get()
    self.assertTrue(entity)
    self.assertEqual({
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'created_by': model.Identity(kind='user', name='creator@example.com'),
      'created_ts': frozen_time,
      'description': 'Test whitelist',
      'modified_by': model.Identity(kind='user', name='creator@example.com'),
      'modified_ts': frozen_time,
      'subnets': ['127.0.0.1/32'],
    }, entity.to_dict())

    copy_in_history = ndb.Key(
        'AuthIPWhitelistHistory', 'A whitelist',
        parent=model.historical_revision_key(1))
    expected = {
      'auth_db_app_version': u'v1a',
      'auth_db_change_comment': u'REST API',
      'auth_db_deleted': False,
    }
    expected.update(entity.to_dict())
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_post_minimal_body(self):
    self.expect_auth_db_rev_change()
    status, body, _ = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={'name': 'A whitelist'},
        expect_xsrf_token_check=True)
    self.assertEqual(201, status)
    self.assertEqual({'ok': True}, body)

  def test_post_mismatching_name(self):
    # 'name' key and name in URL should match.
    status, body, _ = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={'name': 'Another name here'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual(
        {'text': 'Missing or mismatching name in request body'}, body)

  def test_post_bad_body(self):
    # Posting invalid body (bad subnet format).
    status, body, _ = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={'name': 'A whitelist', 'subnets': ['not a subnet']},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual({
        'text':
          'u\'not a subnet\' is not an IP address (not IPv4 or IPv6 address)',
        }, body)

  def test_post_already_exists(self):
    make_ip_whitelist('A whitelist')
    status, body, _ = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={'name': 'A whitelist'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual({'text': 'Such ip whitelist already exists'}, body)

  def test_post_requires_admin(self):
    self.mock_is_admin(False)
    status, body, _ = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={'name': 'A whitelist'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, body)

  def test_post_when_config_locked(self):
    self.mock(rest_api, 'is_config_locked', lambda: True)
    status, body, _ = self.post(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={'name': 'A whitelist'},
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {'text': 'The configuration is managed elsewhere'}, body)

  def test_put_success(self):
    frozen_time = utils.timestamp_to_datetime(1300000000000000)
    self.mock_now(frozen_time)
    creator_identity = model.Identity.from_bytes('user:creator@example.com')
    self.mock_current_identity(creator_identity)

    make_ip_whitelist('A whitelist')

    self.expect_auth_db_rev_change()
    status, body, headers = self.put(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={
          'description': 'Test whitelist',
          'name': 'A whitelist',
          'subnets': ['127.0.0.1/32'],
        },
        expect_xsrf_token_check=True)
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, body)
    self.assertEqual(
        'Sun, 13 Mar 2011 07:06:40 -0000', headers['Last-Modified'])

    entity = model.ip_whitelist_key('A whitelist').get()
    self.assertTrue(entity)
    self.assertEqual({
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'created_by': None,
      'created_ts': frozen_time,
      'description': 'Test whitelist',
      'modified_by': model.Identity(kind='user', name='creator@example.com'),
      'modified_ts': frozen_time,
      'subnets': ['127.0.0.1/32'],
    }, entity.to_dict())

    copy_in_history = ndb.Key(
        'AuthIPWhitelistHistory', 'A whitelist',
        parent=model.historical_revision_key(1))
    expected = {
      'auth_db_app_version': u'v1a',
      'auth_db_change_comment': u'REST API',
      'auth_db_deleted': False,
    }
    expected.update(entity.to_dict())
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_put_mismatching_name(self):
    make_ip_whitelist('A whitelist')
    status, body, _ = self.put(
        path='/auth/api/v1/groups/A%20whitelist',
        body={
          'description': 'Test group',
          'subnets': [],
          'name': 'Bad group name',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual(
        {'text': 'Missing or mismatching name in request body'}, body)

  def test_put_bad_body(self):
    make_ip_whitelist('A whitelist')
    status, body, _ = self.put(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={
          'name': 'A whitelist',
          'subnets': ['not a subnet'],
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(400, status)
    self.assertEqual({
        'text':
          'u\'not a subnet\' is not an IP address (not IPv4 or IPv6 address)'
        }, body)

  def test_put_missing(self):
    status, body, _ = self.put(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={
          'description': 'Test whitelist',
          'name': 'A whitelist',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(404, status)
    self.assertEqual({'text': 'No such ip whitelist'}, body)

  def test_put_bad_precondition(self):
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    make_ip_whitelist('A whitelist')
    status, body, _ = self.put(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={
          'description': 'Test whitelist',
          'name': 'A whitelist',
        },
        headers={
          'If-Unmodified-Since': 'Sun, 1 Mar 1990 00:00:00 -0000',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(412, status)
    self.assertEqual(
        {'text': 'Ip whitelist was modified by someone else'}, body)

  def test_put_when_config_locked(self):
    self.mock(rest_api, 'is_config_locked', lambda: True)
    status, body, _ = self.put(
        path='/auth/api/v1/ip_whitelists/A%20whitelist',
        body={
          'description': 'Test whitelist',
          'name': 'A whitelist',
        },
        expect_errors=True,
        expect_xsrf_token_check=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {'text': 'The configuration is managed elsewhere'}, body)


class MembershipsListHandlerTest(RestAPITestCase):
  def setUp(self):
    super(MembershipsListHandlerTest, self).setUp()
    self.mock_is_admin(True)
    make_group(model.ADMIN_GROUP)
    make_group('A', members=[
      model.Identity.from_bytes('user:a@example.com'),
      model.Identity.from_bytes('user:c@example.com'),
    ])
    make_group('B', members=[
      model.Identity.from_bytes('user:b@example.com'),
      model.Identity.from_bytes('user:c@example.com'),
    ])
    api.reset_local_state()  # invalidate request cache to reread new groups

  def test_get_ok(self):
    status, body, _ = self.get(
        path='/auth/api/v1/memberships/list?identity=user:c@example.com')
    self.assertEqual(200, status)
    self.assertEqual(
        {u'memberships': [{u'group': u'A'}, {u'group': u'B'}]}, body)

  def test_get_empty(self):
    status, body, _ = self.get(
        path='/auth/api/v1/memberships/list?identity=user:unknown@example.com')
    self.assertEqual(200, status)
    self.assertEqual({u'memberships': []}, body)

  def test_get_no_ident(self):
    status, _, _ = self.get(
        path='/auth/api/v1/memberships/list',
        expect_errors=True)
    self.assertEqual(400, status)

  def test_get_bad_ident(self):
    status, _, _ = self.get(
        path='/auth/api/v1/memberships/list?identity=unknown@example.com',
        expect_errors=True)
    self.assertEqual(400, status)

  def test_post_ok(self):
    status, body, _ = self.post(
        path='/auth/api/v1/memberships/list',
        body={
          'per_identity': {
            'user:a@example.com': None,
            'user:b@example.com': None,
            'user:c@example.com': None,
            'user:d@example.com': None,
          }
        })
    self.assertEqual(200, status)
    self.assertEqual({
      u'per_identity': {
        u'user:a@example.com': {u'memberships': [{u'group': u'A'}]},
        u'user:b@example.com': {u'memberships': [{u'group': u'B'}]},
        u'user:c@example.com': {
          u'memberships': [{u'group': u'A'}, {u'group': u'B'}],
        },
        u'user:d@example.com': {u'memberships': []},
      },
    }, body)

  def test_post_empty(self):
    status, _, _ = self.post(
        path='/auth/api/v1/memberships/list',
        body={'per_identity': {}},
        expect_errors=True)
    self.assertEqual(400, status)

  def test_post_bad_params(self):
    status, _, _ = self.post(
        path='/auth/api/v1/memberships/list',
        body={
          'per_identity': {
            'user:a@example.com': 'blah',
          }
        },
        expect_errors=True)
    self.assertEqual(400, status)


class MembershipsCheckHandlerTest(RestAPITestCase):
  def setUp(self):
    super(MembershipsCheckHandlerTest, self).setUp()
    self.mock_is_admin(True)
    make_group(model.ADMIN_GROUP)
    make_group('A', members=[
      model.Identity.from_bytes('user:a@example.com'),
      model.Identity.from_bytes('user:c@example.com'),
    ])
    make_group('B', members=[
      model.Identity.from_bytes('user:b@example.com'),
      model.Identity.from_bytes('user:c@example.com'),
    ])
    api.reset_local_state()  # invalidate request cache to reread new groups

  def test_get_ok(self):
    status, body, _ = self.get(
        path='/auth/api/v1/memberships/check?' +
             'identity=user:a@example.com&groups=XXX&groups=A')
    self.assertEqual(200, status)
    self.assertEqual({u'is_member': True}, body)

  def test_get_no_ident(self):
    status, _, _ = self.get(
        path='/auth/api/v1/memberships/check?groups=XXX&groups=A',
        expect_errors=True)
    self.assertEqual(400, status)

  def test_get_no_groups(self):
    status, _, _ = self.get(
        path='/auth/api/v1/memberships/check?identity=user:a@example.com',
        expect_errors=True)
    self.assertEqual(400, status)

  def test_get_bad_ident(self):
    status, _, _ = self.get(
        path='/auth/api/v1/memberships/check?identity=unknown@example.com',
        expect_errors=True)
    self.assertEqual(400, status)

  def test_post_ok(self):
    status, body, _ = self.post(
        path='/auth/api/v1/memberships/check',
        body={
          'per_identity': {
            'user:a@example.com': {'groups': ['A']},
            'user:b@example.com': {'groups': ['A']},
            'user:c@example.com': {'groups': ['A', 'B']},
            'user:d@example.com': {'groups': ['A', 'B', 'C']},
          }
        })
    self.assertEqual(200, status)
    self.assertEqual({
      u'per_identity': {
        u'user:a@example.com': {u'is_member': True},
        u'user:b@example.com': {u'is_member': False},
        u'user:c@example.com': {u'is_member': True},
        u'user:d@example.com': {u'is_member': False},
      },
    }, body)

  def test_post_empty(self):
    status, _, _ = self.post(
        path='/auth/api/v1/memberships/check',
        body={'per_identity': {}},
        expect_errors=True)
    self.assertEqual(400, status)

  def test_post_bad_params(self):
    status, _, _ = self.post(
        path='/auth/api/v1/memberships/check',
        body={
          'per_identity': {
            'user:a@example.com': {'groups': 'blah'},
          }
        },
        expect_errors=True)
    self.assertEqual(400, status)


class SubgraphHandlerTest(RestAPITestCase):
  def setUp(self):
    super(SubgraphHandlerTest, self).setUp()
    self.mock_is_admin(True)

  def test_no_arg(self):
    status, body, _ = self.get(
        path='/auth/api/v1/subgraph/', expect_errors=True)
    self.assertEqual(400, status)
    self.assertEqual({u'text': u'Bad principal - Not a valid group name'}, body)

  def test_invalid_arg(self):
    status, body, _ = self.get(
        path='/auth/api/v1/subgraph/???', expect_errors=True)
    self.assertEqual(400, status)
    self.assertEqual({u'text': u'Bad principal - Not a valid group name'}, body)

  def test_empty_reply_identity(self):
    status, body, _ = self.get(path='/auth/api/v1/subgraph/user:a@example.com')
    self.assertEqual(200, status)
    self.assertEqual({
      u'subgraph': {
        u'nodes': [
          {u'kind': u'IDENTITY', u'value': u'user:a@example.com'},
        ],
      },
    }, body)

  def test_empty_reply_glob(self):
    status, body, _ = self.get(path='/auth/api/v1/subgraph/user:*@example.com')
    self.assertEqual(200, status)
    self.assertEqual({
      u'subgraph': {
        u'nodes': [
          {u'kind': u'GLOB', u'value': u'user:*@example.com'},
        ],
      },
    }, body)

  def test_empty_reply_group(self):
    status, body, _ = self.get(path='/auth/api/v1/subgraph/group')
    self.assertEqual(200, status)
    self.assertEqual({
      u'subgraph': {
        u'nodes': [
          {u'kind': u'GROUP', u'value': u'group'},
        ],
      },
    }, body)

  def test_non_empty_reply(self):
    make_group('a-root', nested=['b-inner'])
    make_group('b-inner')
    make_group('c-owned-by-root', owners='a-root')
    make_group('d-inc-owned', nested=['c-owned-by-root'])
    make_group('e-owned-by-3', owners='d-inc-owned')
    api.reset_local_state()  # invalidate request cache to reread new groups

    status, body, _ = self.get(path='/auth/api/v1/subgraph/b-inner')
    self.assertEqual(200, status)
    self.assertEqual({u'subgraph': {u'nodes': [
      {u'edges': {u'IN': [1]}, u'kind': u'GROUP', u'value': u'b-inner'},
      {u'edges': {u'OWNS': [2]}, u'kind': u'GROUP', u'value': u'a-root'},
      {u'edges': {u'IN': [3]}, u'kind': u'GROUP', u'value': u'c-owned-by-root'},
      {u'edges': {u'OWNS': [4]}, u'kind': u'GROUP', u'value': u'd-inc-owned'},
      {u'kind': u'GROUP', u'value': u'e-owned-by-3'},
    ]}}, body)


class GroupsSuggestHandlerTest(RestAPITestCase):
  def setUp(self):
    super(GroupsSuggestHandlerTest, self).setUp()
    self.mock_is_admin(True)
    make_group(model.ADMIN_GROUP)
    make_group('Ade')
    make_group('Abc')
    make_group('Z')
    api.reset_local_state()  # invalidate request cache to reread new groups

  def test_get_some(self):
    status, body, _ = self.get(
        path='/auth/api/v1/suggest/groups?name=A')
    self.assertEqual(200, status)
    self.assertEqual({u'names': [u'Abc', u'Ade']}, body)

  def test_get_none(self):
    status, body, _ = self.get(
        path='/auth/api/v1/suggest/groups?name=ZZZ')
    self.assertEqual(200, status)
    self.assertEqual({u'names': []}, body)

  def test_get_all(self):
    status, body, _ = self.get(
        path='/auth/api/v1/suggest/groups')
    self.assertEqual(200, status)
    self.assertEqual(
        {u'names': [u'Abc', u'Ade', u'Z', u'administrators']}, body)


class CertificatesHandlerTest(RestAPITestCase):
  def test_works(self):
    # Test mostly for code coverage.
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))
    status, body, _ = self.get('/auth/api/v1/server/certificates')
    self.assertEqual(200, status)
    self.assertEqual(1300000000000000, body['timestamp'])
    self.assertTrue(body['certificates'])
    for cert in body['certificates']:
      self.assertTrue(isinstance(cert['key_name'], basestring))
      self.assertTrue(isinstance(cert['x509_certificate_pem'], basestring))


class OAuthConfigHandlerTest(RestAPITestCase):
  def setUp(self):
    super(OAuthConfigHandlerTest, self).setUp()
    self.mock_is_admin(True)

  def test_non_configured_works(self):
    expected = {
      'additional_client_ids': [],
      'client_id': '',
      'client_not_so_secret': '',
      'primary_url': None,
      'token_server_url': '',
    }
    status, body, _ = self.get('/auth/api/v1/server/oauth_config')
    self.assertEqual(200, status)
    self.assertEqual(expected, body)

  def test_primary_url_is_set_on_replica(self):
    mock_replication_state('https://primary-url')
    expected = {
      'additional_client_ids': [],
      'client_id': '',
      'client_not_so_secret': '',
      'primary_url': 'https://primary-url',
      'token_server_url': '',
    }
    status, body, _ = self.get('/auth/api/v1/server/oauth_config')
    self.assertEqual(200, status)
    self.assertEqual(expected, body)

  def test_configured_works(self):
    # Mock auth_db.get_oauth_config().
    fake_config = model.AuthGlobalConfig(
        oauth_client_id='some-client-id',
        oauth_client_secret='some-secret',
        oauth_additional_client_ids=['a', 'b', 'c'],
        token_server_url='https://token-server')
    self.mock(rest_api.api, 'get_request_auth_db',
        lambda: api.AuthDB(global_config=fake_config))
    # Call should return this data.
    expected = {
      'additional_client_ids': ['a', 'b', 'c'],
      'client_id': 'some-client-id',
      'client_not_so_secret': 'some-secret',
      'primary_url': None,
      'token_server_url': 'https://token-server',
    }
    status, body, _ = self.get('/auth/api/v1/server/oauth_config')
    self.assertEqual(200, status)
    self.assertEqual(expected, body)

  def test_no_cache_works(self):
    # Put something into DB.
    config_in_db = model.AuthGlobalConfig(
        key=model.root_key(),
        oauth_client_id='config-from-db',
        oauth_client_secret='some-secret-db',
        oauth_additional_client_ids=['a', 'b'],
        token_server_url='https://token-server-db')
    config_in_db.put()

    # Put another version into auth DB cache.
    config_in_cache = model.AuthGlobalConfig(
        oauth_client_id='config-from-cache',
        oauth_client_secret='some-secret-cache',
        oauth_additional_client_ids=['c', 'd'],
        token_server_url='https://token-server-cache')
    self.mock(rest_api.api, 'get_request_auth_db',
        lambda: api.AuthDB(global_config=config_in_cache))

    # Without cache control header a cached version is used.
    expected = {
      'additional_client_ids': ['c', 'd'],
      'client_id': 'config-from-cache',
      'client_not_so_secret': 'some-secret-cache',
      'primary_url': None,
      'token_server_url': 'https://token-server-cache',
    }
    status, body, _ = self.get('/auth/api/v1/server/oauth_config')
    self.assertEqual(200, status)
    self.assertEqual(expected, body)

    # With cache control header a version from DB is used.
    expected = {
      'additional_client_ids': ['a', 'b'],
      'client_id': 'config-from-db',
      'client_not_so_secret': 'some-secret-db',
      'primary_url': None,
      'token_server_url': 'https://token-server-db',
    }
    status, body, _ = self.get(
        path='/auth/api/v1/server/oauth_config',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(200, status)
    self.assertEqual(expected, body)

  def test_post_works(self):
    # Send POST.
    request_body = {
      'additional_client_ids': ['1', '2', '3'],
      'client_id': 'some-client-id',
      'client_not_so_secret': 'some-secret',
      'token_server_url': 'https://token-server',
    }
    self.expect_auth_db_rev_change()
    status, response, _ = self.post(
        path='/auth/api/v1/server/oauth_config',
        body=request_body,
        headers={'X-XSRF-Token': make_xsrf_token()})
    self.assertEqual(200, status)
    self.assertEqual({'ok': True}, response)

    # Ensure it modified the state in DB.
    config = model.root_key().get()
    self.assertEqual('some-client-id', config.oauth_client_id)
    self.assertEqual('some-secret', config.oauth_client_secret)
    self.assertEqual(['1', '2', '3'], config.oauth_additional_client_ids)
    self.assertEqual('https://token-server', config.token_server_url)

    # Created a copy in the historical log.
    copy_in_history = ndb.Key(
        'AuthGlobalConfigHistory', 'root',
        parent=model.historical_revision_key(1))
    expected = {
      'auth_db_app_version': u'v1a',
      'auth_db_change_comment': u'REST API',
      'auth_db_deleted': False,
    }
    expected.update(config.to_dict())
    self.assertEqual(expected, copy_in_history.get().to_dict())

  def test_post_requires_admin(self):
    self.mock_is_admin(False)
    status, response, _ = self.post(
        path='/auth/api/v1/server/oauth_config',
        body={},
        headers={'X-XSRF-Token': make_xsrf_token()},
        expect_errors=True)
    self.assertEqual(403, status)
    self.assertEqual({'text': 'Access is denied.'}, response)

  def test_post_when_config_locked(self):
    self.mock(rest_api, 'is_config_locked', lambda: True)
    request_body = {
      'additional_client_ids': ['1', '2', '3'],
      'client_id': 'some-client-id',
      'client_not_so_secret': 'some-secret',
      'token_server_url': 'https://token-server',
    }
    status, response, _ = self.post(
        path='/auth/api/v1/server/oauth_config',
        body=request_body,
        headers={'X-XSRF-Token': make_xsrf_token()},
        expect_errors=True)
    self.assertEqual(409, status)
    self.assertEqual(
        {'text': 'The configuration is managed elsewhere'}, response)


class ServerStateHandlerTest(RestAPITestCase):
  def test_works(self):
    self.mock_now(utils.timestamp_to_datetime(1300000000000000))

    # Configure as standalone.
    state = model.AuthReplicationState(key=model.replication_state_key())
    state.put()

    expected = {
      'auth_code_version': version.__version__,
      'mode': 'standalone',
      'replication_state': {
        'auth_db_rev': 0,
        'modified_ts': 1300000000000000,
        'primary_id': None,
        'primary_url': None,
      }
    }
    self.mock_is_admin(True)
    status, body, _ = self.get('/auth/api/v1/server/state')
    self.assertEqual(200, status)
    self.assertEqual(expected, body)


class ForbidApiOnReplicaTest(test_case.TestCase):
  """Tests for rest_api.forbid_api_on_replica decorator."""

  def test_allowed_on_non_replica(self):
    class Handler(webapp2.RequestHandler):
      @rest_api.forbid_api_on_replica
      def get(self):
        self.response.write('ok')

    mock_replication_state(None)
    self.assertEqual('ok', call_get(Handler).body)

  def test_forbidden_on_replica(self):
    calls = []

    class Handler(webapp2.RequestHandler):
      @rest_api.forbid_api_on_replica
      def get(self):
        calls.append(1)

    mock_replication_state('http://locahost:1234')
    response = call_get(Handler, status=405)

    self.assertEqual(0, len(calls))
    expected = {
      'primary_url': 'http://locahost:1234',
      'text': 'Use Primary service for API requests',
    }
    self.assertEqual(expected, json.loads(response.body))


class ForbidUiOnReplicaTest(test_case.TestCase):
  """Tests for ui.forbid_ui_on_replica decorator."""

  def test_allowed_on_non_replica(self):
    class Handler(webapp2.RequestHandler):
      @ui.forbid_ui_on_replica
      def get(self):
        self.response.write('ok')

    mock_replication_state(None)
    self.assertEqual('ok', call_get(Handler).body)

  def test_forbidden_on_replica(self):
    calls = []

    class Handler(webapp2.RequestHandler):
      @ui.forbid_ui_on_replica
      def get(self):
        calls.append(1)

    mock_replication_state('http://locahost:1234')
    response = call_get(Handler, status=405)

    self.assertEqual(0, len(calls))
    self.assertEqual(
        '405 Method Not Allowed\n\n'
        'The method GET is not allowed for this resource. \n\n '
        'Not allowed on a replica, see primary at http://locahost:1234',
        response.body)


class RedirectUiOnReplicaTest(test_case.TestCase):
  """Tests for ui.redirect_ui_on_replica decorator."""

  def test_allowed_on_non_replica(self):
    class Handler(webapp2.RequestHandler):
      @ui.redirect_ui_on_replica
      def get(self):
        self.response.write('ok')

    mock_replication_state(None)
    self.assertEqual('ok', call_get(Handler).body)

  def test_redirects_on_replica(self):
    calls = []

    class Handler(webapp2.RequestHandler):
      @ui.redirect_ui_on_replica
      def get(self):
        calls.append(1)

    mock_replication_state('http://locahost:1234')
    response = call_get(Handler, status=302, uri='/some/method?arg=1')

    self.assertEqual(0, len(calls))
    self.assertEqual(
        'http://locahost:1234/some/method?arg=1', response.headers['Location'])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
