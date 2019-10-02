# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import httplib
import logging

import webapp2

from components import decorators
from components import gitiles
from components.config.proto import service_config_pb2
from google.appengine.ext.webapp import template

import common
import gitiles_import
import notifications
import os
import storage
import services


OPENSEARCH_XML = '''<?xml version="1.0" encoding="UTF-8"?>
<OpenSearchDescription xmlns="http://a9.com/-/spec/opensearch/1.1/">
  <ShortName>luci-config</ShortName>
  <Description>
    A configuration service for LUCI
  </Description>
  <Url type="text/html" template="https://%s/#/q/{searchTerms}" />
</OpenSearchDescription>'''


class CronGitilesImport(webapp2.RequestHandler):
  """Imports configs from Gitiles."""
  @decorators.require_cronjob
  def get(self):
    gitiles_import.cron_run_import()


class CronServicesMetadataRequest(webapp2.RequestHandler):
  """Updates stored service metadata."""
  @decorators.require_cronjob
  def get(self):
    services.cron_request_metadata()


class MainPageHandler(webapp2.RequestHandler):
  """ Serves the UI with the proper client ID. """

  def get(self):
    # TODO(nodir): put the client_id to a config file so that it's not hardcoded
    template_values = {
      'client_id':
          ('247108661754-svmo17vmk1j5hlt388gb45qblgvg2h98.apps.'
           'googleusercontent.com'),
    }
    path = os.path.join(os.path.dirname(__file__), 'ui/static/index.html')
    self.response.out.write(template.render(path, template_values))


class SchemasHandler(webapp2.RequestHandler):
  """Redirects to a known schema definition."""

  def get(self, name):
    cfg = storage.get_self_config_async(
        common.SCHEMAS_FILENAME, service_config_pb2.SchemasCfg).get_result()
    # Assume cfg was validated by validation.py
    if cfg:
      for schema in cfg.schemas:
        if schema.name == name:
          # Convert from unicode.
          assert schema.url
          self.redirect(str(schema.url))
          return

    self.response.write('Schema %s not found\n' % name)
    self.response.set_status(httplib.NOT_FOUND)


class OpensearchHandler(webapp2.RequestHandler):
  """Returns opensearch.xml with the correct headers."""

  def get(self):
    self.response.content_type = 'application/opensearchdescription+xml'
    self.response.write(OPENSEARCH_XML % self.request.host)


class TaskGitilesImportConfigSet(webapp2.RequestHandler):
  """Imports a config set from gitiles."""

  def post(self, config_set):
    try:
      gitiles_import.import_config_set(config_set)
    except gitiles_import.NotFoundError as ex:
      logging.warning(ex.message)


def get_frontend_routes():  # pragma: no cover
  return [
    webapp2.Route(r'/', MainPageHandler),
    webapp2.Route(r'/opensearch.xml', OpensearchHandler),
    webapp2.Route(r'/schemas/<name:.+>', SchemasHandler),
    webapp2.Route(r'/_ah/bounce', notifications.BounceHandler),
  ]


def get_backend_routes():  # pragma: no cover
  return [
      webapp2.Route(
          r'/internal/cron/luci-config/gitiles_import',
          CronGitilesImport),
      webapp2.Route(
          r'/internal/cron/luci-config/update_services_metadata',
          CronServicesMetadataRequest),
      webapp2.Route(
          r'/internal/task/luci-config/gitiles_import/<config_set:.+>',
          TaskGitilesImportConfigSet),
  ]
