# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Monitoring related helpers."""

import config
import gae_ts_mon


def is_ts_monitoring_enabled():
  """Returns True if time-series monitoring is enabled."""
  return config.get_settings().enable_ts_monitoring


def wrap_webapp2_app(app):
  """Instruments webapp2 application to track HTTP endpoints performance."""
  gae_ts_mon.initialize(
      app=app,
      is_enabled_fn=is_ts_monitoring_enabled,
      cron_module='backend')
  return app


def get_tsmon_app():
  """Returns the WSGI app with tsmon internal handlers."""
  return gae_ts_mon.app
