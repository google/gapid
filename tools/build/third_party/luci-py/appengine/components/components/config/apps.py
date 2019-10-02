# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""WSGI app with a cron job endpoint.

Used when config component is included via app.yaml includes. If app has
a backend module, it must be included there.
"""

import webapp2

from components import utils

from . import handlers


def create_backend_application():
  app = webapp2.WSGIApplication(
      handlers.get_backend_routes(),
      debug=utils.is_local_dev_server())
  utils.report_memory(app)
  return app


backend = create_backend_application()
