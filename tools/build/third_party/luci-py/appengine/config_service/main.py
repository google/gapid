# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys

APP_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(APP_DIR, 'components', 'third_party'))

import webapp2

from components import endpoints_webapp2
from components import ereporter2
from components import template
from components import utils
from google.appengine.api import app_identity

import gae_ts_mon

import admin
import api
import handlers


def bootstrap_templates():
  TEMPLATES_DIR = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), 'templates')
  template.bootstrap(
      {'templates': TEMPLATES_DIR},
      global_env={
        'hostname': app_identity.get_default_version_hostname()
      })


def create_html_app():  # pragma: no cover
  """Returns WSGI app that serves HTML pages."""
  return webapp2.WSGIApplication(
      handlers.get_frontend_routes(), debug=utils.is_local_dev_server())


def create_endpoints_app():  # pragma: no cover
  """Returns WSGI app that serves cloud endpoints requests."""
  # The default regex doesn't allow / but config_sets and paths have / in them.
  return endpoints_webapp2.api_server(
      [api.ConfigApi, admin.AdminApi], regex='.+')


def create_backend_app():  # pragma: no cover
  """Returns WSGI app for backend."""
  bootstrap_templates()
  return webapp2.WSGIApplication(
      handlers.get_backend_routes(), debug=utils.is_local_dev_server())


def initialize():  # pragma: no cover
  """Bootstraps the global state and creates WSGI applications."""
  ereporter2.register_formatter()
  apps = (create_html_app(), create_endpoints_app(), create_backend_app())
  for app in apps:
    gae_ts_mon.initialize(app=app)
  return apps
