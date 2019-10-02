# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Instance specific settings."""

import logging

from components import config
from components import net
from components import utils
from components.config import validation

from proto import config_pb2

SETTINGS_CFG_FILENAME = 'settings.cfg'


ConfigApi = config.ConfigApi


@validation.self_rule(SETTINGS_CFG_FILENAME, config_pb2.SettingsCfg)
def validate_settings(_cfg, _ctx):
  """Validates settings.cfg file against proto message schema."""
  pass


def _get_settings():
  """Returns (rev, cfg) where cfg is a parsed SettingsCfg message.

  If config does not exists, returns (None, <cfg with defaults>).

  The config is cached in the datastore.
  """
  # store_last_good=True tells config component to update the config file
  # in a cron job. Here we just read from the datastore.
  rev, cfg = config.get_self_config(
      SETTINGS_CFG_FILENAME, config_pb2.SettingsCfg, store_last_good=True)
  cfg = cfg or config_pb2.SettingsCfg()
  return rev, cfg


@utils.cache_with_expiration(60)
def settings():
  """Loads settings from an NDB-based cache or a default one if not present."""
  return _get_settings()[1]
