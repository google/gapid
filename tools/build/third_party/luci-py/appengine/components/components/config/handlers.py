# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import webapp2

from components import decorators

from . import remote


class CronUpdateConfigs(webapp2.RequestHandler):
  """Imports configs from Gitiles."""
  @decorators.require_cronjob
  def get(self):
    remote.cron_update_last_good_configs()


def get_backend_routes():
  # This requires a cron job to this URL.
  return [
    webapp2.Route(
        r'/internal/cron/config/update', CronUpdateConfigs),
  ]
