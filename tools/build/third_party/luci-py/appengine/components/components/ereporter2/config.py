# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Configuraiton of ereporter2.

To use, put the following lines in appengine_config.py:
  components_ereporter2_RECIPIENTS_AUTH_GROUP = 'myproject-ereporter-reports'
"""

from google.appengine.api import lib_config


class Config(object):
  # Name of a group that lists users that receive ereporter2 reports.
  RECIPIENTS_AUTH_GROUP = 'ereporter2-reports'

  # Group that can view all ereporter2 reports without being a general auth
  # admin. It can also silence reports.
  VIEWERS_AUTH_GROUP = 'ereporter2-viewers'


config = lib_config.register('components_ereporter2', Config.__dict__)
