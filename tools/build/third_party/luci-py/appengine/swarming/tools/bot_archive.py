#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Generates the swarming_bot.zip archive for local testing."""

import argparse
import json
import os
import sys

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def read_config():
  config_path = os.path.join(ROOT_DIR, 'swarming_bot', 'config', 'config.json')
  with open(config_path, 'rb') as f:
    return json.load(f) or {}


def get_swarming_bot_zip():
  config = read_config()
  bot_config_path = os.path.join(
      ROOT_DIR, 'swarming_bot', 'config', 'bot_config.py')
  with open(bot_config_path, 'rb') as f:
    additionals = {'config/bot_config.py': f.read()}
  from server import bot_archive
  return bot_archive.get_swarming_bot_zip(
      os.path.join(ROOT_DIR, 'swarming_bot'), config['server'], '1',
      additionals, config['enable_ts_monitoring'])


def main():
  parser = argparse.ArgumentParser(description=__doc__)
  parser.add_argument(
      '--output',
      help='Where to drop the bot archive (default: ./swarming_bot.zip).',
      default='swarming_bot.zip')
  args = parser.parse_args()

  sys.path.insert(0, ROOT_DIR)
  content, _ = get_swarming_bot_zip()
  with open(args.output, 'wb') as f:
    f.write(content)
  return 0


if __name__ == '__main__':
  sys.exit(main())
