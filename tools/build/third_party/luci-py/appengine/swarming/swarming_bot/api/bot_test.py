#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import os
import struct
import sys
import unittest

import test_env_api
test_env_api.setup_test_env()

import bot


def make_bot(remote=None):
  return bot.Bot(
      remote,
      {'dimensions': {'id': ['bot1'], 'pool': ['private']}},
      'https://localhost:1',
      '1234-1a2b3c4-tainted-joe',
      'base_dir',
      None)


class TestBot(unittest.TestCase):
  def test_get_pseudo_rand(self):
    # This test assumes little endian.
    # The following confirms the equivalent code in Bot.get_pseudo_rand():
    self.assertEqual(-1., round(struct.unpack('h', '\x00\x80')[0] / 32768., 4))
    self.assertEqual(1., round(struct.unpack('h', '\xff\x7f')[0] / 32768., 4))
    b = make_bot()
    self.assertEqual(-0.7782, b.get_pseudo_rand(1.))
    self.assertEqual(-0.0778, b.get_pseudo_rand(.1))

  def test_post_error(self):
    # Not looking at the actual stack since the file name is call dependent and
    # the line number will change as the code is modified.
    prefix = (
        'US has failed us\n'
        'Calling stack:\n'
        '  0  ')
    calls = []
    class FakeRemote(object):
      # pylint: disable=no-self-argument
      def post_bot_event(self2, event_type, message, attributes):
        try:
          self.assertEqual('bot_error', event_type)
          expected = {'dimensions': {'id': ['bot1'], 'pool': ['private']}}
          self.assertEqual(expected, attributes)
          self.assertTrue(message.startswith(prefix), repr(message))
          calls.append(event_type)
        except Exception as e:
          calls.append(str(e))

    remote = FakeRemote()
    make_bot(remote).post_error('US has failed us')
    self.assertEqual(['bot_error'], calls)

  def test_frame(self):
    stack = bot._make_stack()
    # Not looking at the actual stack since the file name is call dependent and
    # the line number will change as the code is modified.
    self.assertTrue(stack.startswith('  0  '), repr(stack))

  def test_bot(self):
    obj = make_bot()
    self.assertEqual({'id': ['bot1'], 'pool': ['private']}, obj.dimensions)
    self.assertEqual(
        os.path.join(obj.base_dir, 'swarming_bot.zip'), obj.swarming_bot_zip)
    self.assertEqual('1234-1a2b3c4-tainted-joe', obj.server_version)
    self.assertEqual('base_dir', obj.base_dir)

  def test_attribute_updates(self):
    obj = make_bot()
    obj._update_bot_group_cfg('cfg_ver', {'dimensions': {'pool': ['A']}})
    self.assertEqual({'id': ['bot1'], 'pool': ['A']}, obj.dimensions)
    self.assertEqual({'bot_group_cfg_version': 'cfg_ver'}, obj.state)

    # Dimension in bot_group_cfg ('A') wins over custom one ('B').
    obj._update_dimensions({'foo': ['baz'], 'pool': ['B']})
    self.assertEqual({'foo': ['baz'], 'pool': ['A']}, obj.dimensions)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
