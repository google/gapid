# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import collections
import logging

from google.appengine.api import logservice

from components import utils


# Logs prefix.
#
# The idea is that a request handler logs with this prefix, and this is reaped
# back by reading the logs, which enables reconstructing the stats.
PREFIX = 'Stats: '


## Private code.


def _yield_logs(start_time, end_time):
  """Yields logservice.RequestLogs for the requested time interval.

  Meant to be mocked in tests.
  """
  # If module_versions is not specified, it will default to the current version
  # on current module, which is not what we want.
  # TODO(maruel): Keep request.offset and use it to resume the query by using it
  # instead of using start_time/end_time.
  module_versions = utils.get_module_version_list(None, True)
  for request in logservice.fetch(
      start_time=start_time - 1 if start_time else start_time,
      end_time=end_time + 1 if end_time else end_time,
      include_app_logs=True,
      module_versions=module_versions):
    yield request


## Public code.


# One handled HTTP request and the associated statistics if any.
StatsEntry = collections.namedtuple('StatsEntry', ('request', 'entries'))


def add_entry(message):
  """Adds an entry for the current request.

  Meant to be mocked in tests.
  """
  logging.debug(PREFIX + message)


def yield_entries(start_time, end_time):
  """Yields StatsEntry in this time interval.

  Look at requests that *ended* between [start_time, end_time[. Ignore the start
  time of the request. This is because the parameters start_time and end_time of
  logserver.fetch() filters on the completion time of the request.
  """
  offset = len(PREFIX)
  for request in _yield_logs(start_time, end_time):
    if not request.finished or not request.end_time:
      continue
    if start_time and request.end_time < start_time:
      continue
    if end_time and request.end_time >= end_time:
      continue

    # Gathers all the entries added via add_entry().
    entries = [
      l.message[offset:] for l in request.app_logs
      if l.level <= logservice.LOG_LEVEL_INFO and l.message.startswith(PREFIX)
    ]
    yield StatsEntry(request, entries)
