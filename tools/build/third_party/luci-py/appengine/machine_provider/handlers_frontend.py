# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Front-end UI."""

import logging
import os

import webapp2

from components import auth
from components import datastore_utils
from components import template
from components import utils

import acl
import handlers_endpoints
import models


THIS_DIR = os.path.dirname(os.path.abspath(__file__))


class CatalogHandler(auth.AuthenticatingHandler):
  """Catalog handler."""

  @auth.autologin
  @auth.require(acl.can_view_catalog)
  def get(self, machine_id=None):
    params = {
        'machines': [],
        'next_page_token': None,
    }
    if machine_id:
      machine = models.CatalogMachineEntry.get_by_id(machine_id)
      if not machine:
        self.abort(404)
      params['machines'] = [machine]
    else:
      query = models.CatalogMachineEntry.query().order(
          models.CatalogMachineEntry.dimensions.hostname)
      page_token = self.request.get('page_token') or ''
      params['machines'], params['next_page_token'] = (
          datastore_utils.fetch_page(query, 50, page_token))

    self.response.write(
        template.render('templates/catalog.html', params=params))


class LeaseRequestHandler(auth.AuthenticatingHandler):
  """Lease request handler."""

  # TODO(smut): Update list of lease request viewers.
  @auth.autologin
  @auth.require(auth.is_admin)
  def get(self, lease_id=None):
    params = {
        'lease_requests': [],
        'next_page_token': None,
        'now_ts': utils.time_time(),
    }
    if lease_id:
      lease_request = models.LeaseRequest.get_by_id(lease_id)
      if not lease_request:
        self.abort(404)
      params['lease_requests'] = [lease_request]
    else:
      query = models.LeaseRequest.query().order(
          -models.LeaseRequest.last_modified_ts)
      page_token = self.request.get('page_token') or ''
      params['lease_requests'], params['next_page_token'] = (
          datastore_utils.fetch_page(query, 50, page_token))

    self.response.write(template.render('templates/leases.html', params=params))


class RootHandler(auth.AuthenticatingHandler):
  """Root handler."""

  @auth.public
  def get(self):
    self.response.write(template.render('templates/root.html'))


def get_routes():
  return [
      webapp2.Route('/', handler=RootHandler),
      webapp2.Route('/catalog', handler=CatalogHandler),
      webapp2.Route('/catalog/<machine_id>', handler=CatalogHandler),
      webapp2.Route('/leases', handler=LeaseRequestHandler),
      webapp2.Route('/leases/<lease_id>', handler=LeaseRequestHandler),
  ]


def create_frontend_app():
  template.bootstrap({
      'templates': os.path.join(THIS_DIR, 'templates'),
  })
  routes = []
  if not utils.should_disable_ui_routes():
    routes.extend(get_routes())
  return webapp2.WSGIApplication(routes)
