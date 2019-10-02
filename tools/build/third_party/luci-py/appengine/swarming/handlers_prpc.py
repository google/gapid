# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines Swarming Server frontend pRPC handlers."""

import logging

from components import prpc
from components.prpc.codes import StatusCode

from components import datastore_utils
from proto.api import swarming_prpc_pb2  # pylint: disable=no-name-in-module
from proto.api import swarming_pb2  # pylint: disable=no-name-in-module
from server import bot_management


class BotAPIService(object):
  """Service implements the pRPC service in swarming.proto."""

  DESCRIPTION = swarming_prpc_pb2.BotAPIServiceDescription

  # TODO(maruel): Add implementation. https://crbug.com/913953

  def Events(self, request, context):
    logging.debug('%s', request)
    try:
      if not request.bot_id:
        # TODO(maruel): Allows not specifying one. Or specifying a pool.
        raise ValueError('specify bot_id')

      # Transparently limit to 1000, default to 200.
      page_size = request.page_size or 200
      if page_size > 1000:
        page_size = 1000
      if page_size < 0:
        raise ValueError('page_size must be positive')

      start = None
      end = None
      if request.HasField('start_time'):
        start = request.start_time.ToDatetime()
      if request.HasField('end_time'):
        end = request.end_time.ToDatetime()
      if (start and end) and start >= end:
        raise ValueError('start_time must be before end_time')

      # The BotEvent key is already in the right chronological order, but
      # querying per BotEvent.ts *requires* ordering per BotEvent.ts.
      order = not (start or end)
      q = bot_management.get_events_query(request.bot_id, order)
      if not order:
        q = q.order(-bot_management.BotEvent.ts, bot_management.BotEvent.key)
      if start:
        q = q.filter(bot_management.BotEvent.ts >= start)
      if end:
        q = q.filter(bot_management.BotEvent.ts < end)

      items, cursor = datastore_utils.fetch_page(
          q, page_size, request.page_token)
      if not items:
        # Check if the bot exists, if not, return a 404. We check BotRoot, not
        # BotInfo, so that even deleted bots can be queried. See bot_management
        # for more information.
        if not bot_management.get_root_key(request.bot_id).get():
          context.set_code(StatusCode.NOT_FOUND)
          context.set_details('Bot does not exist')
          return None
    except ValueError as e:
      context.set_code(StatusCode.INVALID_ARGUMENT)
      context.set_details(str(e))
      return None
    logging.info('Returning %d events', len(items))
    out = swarming_pb2.BotEventsResponse(next_page_token=cursor)
    for r in items:
      i = out.events.add()
      r.to_proto(i)
    return out


def get_routes():
  s = prpc.Server()
  s.add_service(BotAPIService())
  return s.get_routes()
