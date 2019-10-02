# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import webapp2

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components import auth
from components import utils


class WarmupHandler(webapp2.RequestHandler):
  def get(self):
    auth.warmup()
    self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
    self.response.write('ok')


assert utils.is_local_dev_server()
auth.disable_process_cache()

# See components/auth/change_log.py, is_changle_log_indexed.
ndb.add_flow_exception(datastore_errors.NeedIndexError)

# Add a fake admin for local dev server.
if not auth.is_replica():
  auth.bootstrap_group(
      auth.ADMIN_GROUP,
      [auth.Identity(auth.IDENTITY_USER, 'test@example.com')],
      'Users that can manage groups')

# /_ah/warmup is used by the smoke test to detect that app is alive.
app = webapp2.WSGIApplication(
    [webapp2.Route(r'/_ah/warmup', WarmupHandler)], debug=True)
