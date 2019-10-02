#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import os
import random
import sys
import unittest

import test_env_handlers

import webapp2
import webtest

from google.appengine.ext import ndb

from google.protobuf import struct_pb2
from google.protobuf import timestamp_pb2

from components import utils
from components.prpc import encoding

from proto.api import swarming_pb2  # pylint: disable=no-name-in-module
from server import task_queues
import handlers_bot
import handlers_prpc


def _decode(raw, dst):
  # Skip escaping characters.
  assert raw[:5] == ')]}\'\n', raw[:5]
  return encoding.get_decoder(encoding.Encoding.JSON)(raw[5:], dst)


def _encode(d):
  # Skip escaping characters.
  raw = encoding.get_encoder(encoding.Encoding.JSON)(d)
  assert raw[:5] == ')]}\'\n', raw[:5]
  return raw[5:]


class PRPCTest(test_env_handlers.AppTestBase):
  """Tests the pRPC handlers."""
  def setUp(self):
    super(PRPCTest, self).setUp()
    # handlers_bot is necessary to run fake tasks.
    routes = handlers_prpc.get_routes() + handlers_bot.get_routes()
    self.app = webtest.TestApp(
        webapp2.WSGIApplication(routes, debug=True),
        extra_environ={
          'REMOTE_ADDR': self.source_ip,
          'SERVER_SOFTWARE': os.environ['SERVER_SOFTWARE'],
        },
    )
    self._headers = {
      'Content-Type': encoding.Encoding.JSON[1],
      'Accept': encoding.Encoding.JSON[1],
    }
    self._enqueue_task_orig = self.mock(
        utils, 'enqueue_task', self._enqueue_task)
    self.now = datetime.datetime(2010, 1, 2, 3, 4, 5)
    self.mock_now(self.now)
    self.mock_default_pool_acl([])

  @ndb.non_transactional
  def _enqueue_task(self, url, queue_name, **kwargs):
    if queue_name == 'rebuild-task-cache':
      # Call directly into it.
      self.assertEqual(True, task_queues.rebuild_task_cache(kwargs['payload']))
      return True
    if queue_name == 'pubsub':
      return True
    self.fail(url)
    return False

  def _test_bot_events_simple(self, request):
    self.set_as_bot()
    self.do_handshake()
    self.set_as_user()
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(request), self._headers)
    expected = swarming_pb2.BotEventsResponse(
      events=[
        swarming_pb2.BotEvent(
          event_time=timestamp_pb2.Timestamp(seconds=1262401445),
          bot=swarming_pb2.Bot(
            bot_id='bot1',
            pools=[u'default'],
            info=swarming_pb2.BotInfo(
              supplemental=struct_pb2.Struct(
                fields={
                  'running_time': struct_pb2.Value(number_value=1234.0),
                  'sleep_streak': struct_pb2.Value(number_value=0),
                  'started_ts': struct_pb2.Value(number_value=1410990411.11),
                }),
              external_ip='192.168.2.2',
              authenticated_as='bot:whitelisted-ip',
              version='123',
              ),
            dimensions=[
              swarming_pb2.StringListPair(key='id', values=['bot1']),
              swarming_pb2.StringListPair(key='os', values=['Amiga']),
              swarming_pb2.StringListPair(key='pool', values=['default']),
            ]),
          event=swarming_pb2.BOT_NEW_SESSION,
        ),
      ])
    resp = swarming_pb2.BotEventsResponse()
    _decode(raw_resp.body, resp)
    self.assertEqual(unicode(expected), unicode(resp))

  def test_botevents_empty(self):
    # Minimum request, all optional fields left out.
    self._test_bot_events_simple(swarming_pb2.BotEventsRequest(bot_id=u'bot1'))

  def test_botevents_empty_time(self):
    msg = swarming_pb2.BotEventsRequest(bot_id=u'bot1')
    msg.start_time.FromDatetime(self.now)
    msg.end_time.FromDatetime(self.now + datetime.timedelta(seconds=1))
    self._test_bot_events_simple(msg)

  def test_botevents_missing(self):
    # No such bot.
    msg = swarming_pb2.BotEventsRequest(bot_id=u'unknown')
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(msg), self._headers,
        expect_errors=True)
    self.assertEqual(raw_resp.status, '404 Not Found')
    self.assertEqual(raw_resp.body, 'Bot does not exist')

  def test_botevents_invalid_page_size(self):
    msg = swarming_pb2.BotEventsRequest(bot_id=u'bot1', page_size=-1)
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(msg), self._headers,
        expect_errors=True)
    self.assertEqual(raw_resp.status, '400 Bad Request')
    self.assertEqual(raw_resp.body, 'page_size must be positive')

  def test_botevents_invalid_bot_id(self):
    # Missing bot_id
    msg = swarming_pb2.BotEventsRequest()
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(msg), self._headers,
        expect_errors=True)
    self.assertEqual(raw_resp.status, '400 Bad Request')
    self.assertEqual(raw_resp.body, 'specify bot_id')

  def test_botevents_start_end(self):
    msg = swarming_pb2.BotEventsRequest(bot_id=u'bot1')
    msg.start_time.FromDatetime(self.now)
    msg.end_time.FromDatetime(self.now)
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(msg), self._headers,
        expect_errors=True)
    self.assertEqual(raw_resp.status, '400 Bad Request')
    self.assertEqual(raw_resp.body, 'start_time must be before end_time')

  def test_botevents(self):
    # Run one task.
    self.mock(random, 'getrandbits', lambda _: 0x88)

    self.set_as_bot()
    self.mock_now(self.now, 0)
    params = self.do_handshake()
    self.set_as_user()
    now_60 = self.mock_now(self.now, 60)
    self.client_create_task_raw()
    self.set_as_bot()
    self.mock_now(self.now, 120)
    res = self.bot_poll(params=params)
    now_180 = self.mock_now(self.now, 180)
    response = self.bot_complete_task(task_id=res['manifest']['task_id'])
    self.assertEqual({u'must_stop': False, u'ok': True}, response)
    self.mock_now(self.now, 240)
    params['event'] = 'bot_rebooting'
    params['message'] = 'for the best'
    # TODO(maruel): https://crbug.com/913953
    response = self.post_json('/swarming/api/v1/bot/event', params)
    self.assertEqual({}, response)

    # Do not filter by time.
    self.set_as_privileged_user()
    msg = swarming_pb2.BotEventsRequest(bot_id=u'bot1', page_size=1001)
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(msg), self._headers)
    resp = swarming_pb2.BotEventsResponse()
    _decode(raw_resp.body, resp)

    dimensions = [
      swarming_pb2.StringListPair(key='id', values=['bot1']),
      swarming_pb2.StringListPair(key='os', values=['Amiga']),
      swarming_pb2.StringListPair(key='pool', values=['default']),
    ]
    common_info = swarming_pb2.BotInfo(
        supplemental=struct_pb2.Struct(
            fields={
              'bot_group_cfg_version':struct_pb2.Value(string_value='default'),
              'running_time': struct_pb2.Value(number_value=1234.0),
              'sleep_streak': struct_pb2.Value(number_value=0),
              'started_ts': struct_pb2.Value(number_value=1410990411.11),
            }),
        external_ip='192.168.2.2',
        authenticated_as='bot:whitelisted-ip',
        version=self.bot_version,
    )
    events = [
      swarming_pb2.BotEvent(
          event_time=timestamp_pb2.Timestamp(seconds=1262401685),
          bot=swarming_pb2.Bot(
              bot_id='bot1',
              pools=[u'default'],
              status=swarming_pb2.BOT_STATUS_UNSPECIFIED,
              info=common_info,
              dimensions=dimensions),
          event=swarming_pb2.BOT_REBOOTING_HOST,
          event_msg='for the best',
      ),
      swarming_pb2.BotEvent(
          event_time=timestamp_pb2.Timestamp(seconds=1262401625),
          bot=swarming_pb2.Bot(
              bot_id='bot1',
              pools=[u'default'],
              status=swarming_pb2.BUSY,
              current_task_id='5cfcee8008811',
              info=common_info,
              dimensions=dimensions),
          event=swarming_pb2.TASK_COMPLETED,
      ),
      swarming_pb2.BotEvent(
          event_time=timestamp_pb2.Timestamp(seconds=1262401565),
          bot=swarming_pb2.Bot(
              bot_id='bot1',
              pools=[u'default'],
              current_task_id='5cfcee8008811',
              status=swarming_pb2.BUSY,
              info=common_info,
              dimensions=dimensions),
          event=swarming_pb2.INSTRUCT_START_TASK,
      ),
      swarming_pb2.BotEvent(
          event_time=timestamp_pb2.Timestamp(seconds=1262401445),
          bot=swarming_pb2.Bot(
              bot_id='bot1',
              pools=[u'default'],
              status=swarming_pb2.BOT_STATUS_UNSPECIFIED,
              info=swarming_pb2.BotInfo(
                  supplemental=struct_pb2.Struct(
                      fields={
                        'running_time': struct_pb2.Value(number_value=1234.0),
                        'sleep_streak': struct_pb2.Value(number_value=0),
                        'started_ts': struct_pb2.Value(
                            number_value=1410990411.11),
                      }),
                  external_ip='192.168.2.2',
                  authenticated_as='bot:whitelisted-ip',
                  version='123',
              ),
              dimensions=dimensions),
          event=swarming_pb2.BOT_NEW_SESSION,
      ),
    ]
    self.assertEqual(
        unicode(swarming_pb2.BotEventsResponse(events=events)), unicode(resp))

    # Now test with a subset. It will retrieve events 1 and 2.
    msg = swarming_pb2.BotEventsRequest(bot_id=u'bot1')
    msg.start_time.FromDatetime(now_60)
    msg.end_time.FromDatetime(now_180 + datetime.timedelta(seconds=1))
    raw_resp = self.app.post(
        '/prpc/swarming.v1.BotAPI/Events', _encode(msg), self._headers)
    resp = swarming_pb2.BotEventsResponse()
    _decode(raw_resp.body, resp)
    self.assertEqual(
        unicode(swarming_pb2.BotEventsResponse(events=events[1:3])),
        unicode(resp))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
