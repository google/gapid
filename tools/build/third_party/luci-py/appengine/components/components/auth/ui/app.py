# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Exports a function that creates WSGI app with Auth routes (API and UI)."""

import webapp2

from . import rest_api
from . import ui
from .. import change_log

# Part of public API of 'auth' component, exposed by this module.
__all__ = ['create_wsgi_application']


def create_wsgi_application(debug=False):
  routes = []
  routes.extend(rest_api.get_rest_api_routes())
  routes.extend(ui.get_ui_routes())
  routes.extend(change_log.get_backend_routes())
  return webapp2.WSGIApplication(routes, debug=debug)
