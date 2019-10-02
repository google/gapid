# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This modules is imported by AppEngine and defines the frontend 'app' object.

It is a separate file so that application bootstrapping code like ereporter2,
that shouldn't be done in unit tests, can be done safely. This file must be
tested via a smoke test.
"""

import os
import sys

import webapp2

APP_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(APP_DIR, 'components', 'third_party'))

from google.appengine.ext import ndb

from components import endpoints_webapp2
from components import ereporter2
from components import utils

import gae_ts_mon

import event_mon_metrics
import handlers_endpoints
import handlers_frontend
import handlers_prpc
import template
import ts_mon_metrics
from server import acl
from server import config
from server import pools_config


# pylint: disable=redefined-outer-name
def create_application():
  ereporter2.register_formatter()
  # Task queues must be sent to the backend.
  utils.set_task_queue_module('backend')
  template.bootstrap()

  # Zap out the ndb in-process cache by default.
  # This cache causes excessive memory usage in in handler where a lot of
  # entities are fetched in one query. When coupled with high concurrency
  # as specified via max_concurrent_requests in app.yaml, this may cause out of
  # memory errors.
  ndb.Context.default_cache_policy = staticmethod(lambda _key: False)
  ndb.Context._cache_policy = staticmethod(lambda _key: False)

  # If running on a local dev server, allow bots to connect without prior
  # groups configuration. Useful when running smoke test.
  if utils.is_local_dev_server():
    acl.bootstrap_dev_server_acls()
    pools_config.bootstrap_dev_server_acls()

  def is_enabled_callback():
    return config.settings().enable_ts_monitoring

  # App that serves HTML pages and old API.
  frontend_app = handlers_frontend.create_application(False)
  gae_ts_mon.initialize(frontend_app, is_enabled_fn=is_enabled_callback)

  endpoints_api = endpoints_webapp2.api_server([
    handlers_endpoints.SwarmingServerService,
    handlers_endpoints.SwarmingTaskService,
    handlers_endpoints.SwarmingTasksService,
    handlers_endpoints.SwarmingQueuesService,
    handlers_endpoints.SwarmingBotService,
    handlers_endpoints.SwarmingBotsService,
    # components.config endpoints for validation and configuring of luci-config
    # service URL.
    config.ConfigApi,
  ])

  prpc_api = webapp2.WSGIApplication(handlers_prpc.get_routes())

  # Local import, because it instantiates the mapreduce app.
  # This is for the Web UI.
  from mapreduce import main
  gae_ts_mon.initialize(main.APP, is_enabled_fn=is_enabled_callback)

  event_mon_metrics.initialize()
  ts_mon_metrics.initialize()
  utils.report_memory(frontend_app)
  utils.report_memory(endpoints_api)
  utils.report_memory(prpc_api)
  return frontend_app, endpoints_api, prpc_api, main.APP


app, endpoints_app, prpc_app, mapreduce_app = create_application()
