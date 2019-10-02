# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Code to help test cases using stats_framework.

Implicitly depends on auto_stub.
"""

import datetime

import webtest

from components.stats_framework import stats_logs
from components import utils


## Private code.


class _RequestLog(object):
  """Simple mock of logservice.RequestLog."""
  def __init__(self):
    self.status = 200
    self.response_size = None
    self.end_time = None
    self.app_logs = []
    self.finished = True


## Public code.


def configure(test):
  """Mocks add_entry/_yield_logs until we figure out how to use
  init_logservice_stub() successfully.
  """
  _request_logs = []

  def _add_entry(message):
    _request_logs[-1].app_logs.append(
        stats_logs.logservice.AppLog(
            level=stats_logs.logservice.LOG_LEVEL_DEBUG,
            message=stats_logs.PREFIX + message))

  def _do_request(req, *args, **kwargs):
    entry = _RequestLog()
    _request_logs.append(entry)
    response = None
    try:
      response = _old_request(req, *args, **kwargs)
      return response
    finally:
      entry.status = response.status_code if response else 503
      entry.response_size = response.content_length if response else 0
      entry.end_time = utils.time_time()

  def _yield_logs(_start_time, _end_time):
    """Returns fake _RequestLog entities.

    Ignore start_time and end_time, it's assumed the caller will filter them
    again.
    """
    for request in _request_logs:
      yield request

  test.mock(stats_logs, 'add_entry', _add_entry)
  test.mock(stats_logs, '_yield_logs', _yield_logs)
  _old_request = test.mock(webtest.TestApp, 'do_request', _do_request)


def reset_timestamp(handler, timestamp):
  """Registers last timestamp to 10 minutes ago so it doesn't search earlier
  in time.

  Otherwise by default StatsFramework will backtrace to MAX_BACKTRACK days
  ago and will then limit processing to MAX_MINUTES_PER_PROCESS minutes.
  """
  timestamp = timestamp - datetime.timedelta(seconds=10*60)
  timestamp = datetime.datetime(*timestamp.timetuple()[:5], second=0)
  # pylint: disable=W0212
  handler._set_last_processed_time(timestamp)
