# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines Isolate Server frontend pRPC handlers."""

import datetime
import logging

from components import prpc
from components.prpc.codes import StatusCode
from components import stats_framework
from components import utils

from proto import isolated_pb2  # pylint: disable=no-name-in-module
from proto import isolated_prpc_pb2 # pylint: disable=no-name-in-module

import stats


class IsolatedService(object):
  """Service implements the pRPC service in isolated.proto."""

  DESCRIPTION = isolated_prpc_pb2.IsolatedServiceDescription
  _RESOLUTION_MAP = {
    isolated_pb2.MINUTE: 'minutes',
    isolated_pb2.HOUR: 'hours',
    isolated_pb2.DAY: 'days',
  }

  def Stats(self, request, context):
    res = self._RESOLUTION_MAP.get(request.resolution)
    if not res:
      context.set_code(StatusCode.INVALID_ARGUMENT)
      context.set_details('Invalid resolution')
      return None

    if not 1 <= request.page_size <= 1000:
      context.set_code(StatusCode.INVALID_ARGUMENT)
      context.set_details('Invalid page_size; must be between 1 and 1000')
      return None

    if request.latest_time.seconds:
      now = request.latest_time.ToDatetime()
    else:
      now = utils.utcnow()
    # Round time to the minute.
    now = datetime.datetime(*now.timetuple()[:5], tzinfo=now.tzinfo)
    entities = stats_framework.get_stats(
        stats.STATS_HANDLER, res, now, request.page_size, False)
    out = isolated_pb2.StatsResponse()
    for s in entities:
      stats.snapshot_to_proto(s, out.measurements.add())
    logging.info('Found %d entities', len(entities))
    return out


def get_routes():
  s = prpc.Server()
  s.add_service(IsolatedService())
  return s.get_routes()
