#!/usr/bin/env python
# coding: utf-8
# Copyright 2019 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import logging
import os
import sys
import unittest

# Sets up environment.
import test_env_handlers

import webtest

import handlers_frontend
import template
from server import bot_code


class FrontendTest(test_env_handlers.AppTestBase):
  def setUp(self):
    super(FrontendTest, self).setUp()
    template.bootstrap()
    # By default requests in tests are coming from bot with fake IP.
    self.app = webtest.TestApp(
        handlers_frontend.create_application(True),
        extra_environ={
          'REMOTE_ADDR': self.source_ip,
          'SERVER_SOFTWARE': os.environ['SERVER_SOFTWARE'],
        })

  def tearDown(self):
    try:
      template.reset()
    finally:
      super(FrontendTest, self).tearDown()

  def test_root(self):
    response = self.app.get('/', status=200)
    self.assertGreater(len(response.body), 600)

  def test_all_swarming_handlers_secured(self):
    # Test that all handlers are accessible only to authenticated user or
    # bots. Assumes all routes are defined with plain paths (i.e.
    # '/some/handler/path' and not regexps).

    # URL prefixes that correspond to routes that are not protected by swarming
    # app code. It may be routes that do not require login or routes protected
    # by GAE itself via 'login: admin' in app.yaml.
    using_app_login_prefixes = (
      '/auth/',
    )

    public_urls = frozenset([
      '/',
      '/oldui',
      '/_ah/warmup',
      '/api/config/v1/validate',
      '/auth',
      '/ereporter2/api/v1/on_error',
      '/api/discovery/v1/apis',
      '/api/static/proxy.html',
      '/api/swarming/v1/server/permissions',
      '/swarming/api/v1/client/list',
      '/swarming/api/v1/bot/server_ping',
      '/user/tasks',
      '/restricted/bots',
    ])

    # Grab the set of all routes.
    app = self.app.app
    routes = set(app.router.match_routes)
    routes.update(app.router.build_routes.itervalues())

    # Get all routes that are not protected by GAE auth mechanism.
    routes_to_check = [
      route for route in routes
      if (route.template not in public_urls and
          not route.template.startswith(using_app_login_prefixes))
    ]

    # Produces a body to POST to a /swarming/api/v1/bot/* request.
    def fake_body_for_bot_request(path):
      body = {'id': 'bot-id', 'task_id': 'task_id'}
      if path == '/swarming/api/v1/bot/oauth_token':
        body.update({'account_id': 'system', 'scopes': ['a', 'b']})
      return body

    # Helper function that executes GET or POST handler for corresponding route
    # and asserts it returns 403 or 405.
    def check_protected(route, method):
      assert method in ('GET', 'POST')
      # Get back original path from regexp.
      path = route.template
      if path[0] == '^':
        path = path[1:]
      if path[-1] == '$':
        path = path[:-1]

      headers = {}
      body = ''
      if method == 'POST' and path.startswith('/swarming/api/v1/bot/'):
        headers = {'Content-Type': 'application/json'}
        body = json.dumps(fake_body_for_bot_request(path))

      response = getattr(self.app, method.lower())(
          path, body, expect_errors=True, headers=headers)
      message = ('%s handler is not protected: %s, '
                 'returned %s' % (method, path, response))
      self.assertIn(response.status_int, (302, 403, 405), msg=message)
      if response.status_int == 302:
        # There's two reasons, either login or redirect to api-explorer.
        options = (
          # See user_service_stub.py, _DEFAULT_LOGIN_URL.
          'https://www.google.com/accounts/Login?continue=',
          'https://apis-explorer.appspot.com/apis-explorer',
        )
        self.assertTrue(
            response.headers['Location'].startswith(options), route)

    self.set_as_anonymous()
    # Try to execute 'get' and 'post' and verify they fail with 403 or 405.
    for route in routes_to_check:
      if '<' in route.template:
        # Sadly, the url cannot be used as-is. Figure out a way to test them
        # easily.
        continue
      check_protected(route, 'GET')
      check_protected(route, 'POST')

  def test_task_redirect(self):
    self.set_as_anonymous()
    self.app.get('/user/tasks', status=302)
    self.app.get('/user/task/123', status=302)

  def test_bot_redirect(self):
    self.set_as_anonymous()
    self.app.get('/restricted/bots', status=302)
    self.app.get('/restricted/bot/bot321', status=302)

  # Admin-specific management pages.
  def test_bootstrap_default(self):
    self.set_as_bot()
    self.mock(bot_code, 'generate_bootstrap_token', lambda: 'bootstrap-token')
    actual = self.app.get('/bootstrap').body
    path = os.path.join(self.APP_DIR, 'swarming_bot', 'config', 'bootstrap.py')
    with open(path, 'rb') as f:
      expected = f.read()
    header = (
        u'#!/usr/bin/env python\n'
        '# coding: utf-8\n'
        'host_url = \'http://localhost\'\n'
        'bootstrap_token = \'bootstrap-token\'\n')
    self.assertEqual(header + expected, actual)

  def test_config(self):
    self.set_as_admin()
    self.app.get('/restricted/config')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL,
      format='%(levelname)-7s %(filename)s:%(lineno)3d %(message)s')
  unittest.main()
