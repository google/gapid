# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Functions to read configs from a configuration service.

Reads configuration files stored in a configuration service instance.
Memaches results, in particular caches config blobs by hash.
Optionally caches configs in the datastore for performance and reliability.

If config service hostname is not set, reads configs from the filesystem.

Example usage:

  // my_config.proto
  message MyCfg {
    optional string param1 = 1;
  }

  # my.cfg file in a repository mapped to services/my_service
  params1: "value"

  # my_config.py in my_service.appspot.com
  from components import config
  import my_config_pb2

  def get_param1():
    # Read latest version of my.cfg in services/my_service config set.
    rev, cfg = config.get_self_config('my.cfg', my_config_pb2.MyCfg)
    return cfg.param1

For more info, read docstrings in api.py.
"""

# Pylint doesn't like relative wildcard imports.
# pylint: disable=W0401,W0403

from .api import *
from .common import *
from .endpoint import ConfigApi
