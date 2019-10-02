#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import os
import unittest

# Import before test_env, to confirm it doesn't depend on GAE.
import bot_archive

import test_env
test_env.setup_test_env()

from proto.config import config_pb2


ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


_EXPECTED_CONFIG_KEYS = {
  'enable_ts_monitoring', 'isolate_grpc_proxy', 'server', 'server_version',
  'swarming_grpc_proxy',
}


def _read_config():
  config_path = os.path.join(ROOT_DIR, 'swarming_bot', 'config', 'config.json')
  with open(config_path, 'rb') as f:
    return json.load(f) or {}


class Test(unittest.TestCase):
  def test_file(self):
    self.assertEqual(_EXPECTED_CONFIG_KEYS, set(_read_config()))

  def test_make(self):
    settings = config_pb2.SettingsCfg()
    config = json.loads(
        bot_archive._make_config_json('host', 'host_version', settings))
    self.assertEqual(_EXPECTED_CONFIG_KEYS, set(config))


if __name__ == '__main__':
  unittest.main()
