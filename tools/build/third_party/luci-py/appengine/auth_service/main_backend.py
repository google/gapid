# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This modules is imported by AppEngine and defines the 'app' object.

It is a separate file so that application bootstrapping code like ereporter2,
that shouldn't be done in unit tests, can be done safely. This file must be
tested via a smoke test.
"""

import os
import sys

APP_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(APP_DIR, 'components', 'third_party'))

from components import ereporter2

import handlers_backend
import monitoring


def create_application():
  ereporter2.register_formatter()
  return monitoring.wrap_webapp2_app(handlers_backend.create_application(False))


app = create_application()
ts_mon_app = monitoring.get_tsmon_app()
