# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys


# swarming_bot/
BOT_DIR = os.path.dirname(
    os.path.dirname(os.path.realpath(os.path.abspath(__file__))))


def setup_test_env():
  """Sets up the environment for bot tests."""
  sys.path.insert(0, BOT_DIR)
  import test_env_bot
  test_env_bot.setup_test_env()
