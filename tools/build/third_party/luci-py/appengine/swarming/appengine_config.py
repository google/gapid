# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Configures includes (components.auth).

https://developers.google.com/appengine/docs/python/tools/appengineconfig
"""

import os

from server import config


components_auth_UI_APP_NAME = 'Swarming'

# Used to make components.auth trust the client ID used by the web UI. This is
# a lazily-called callback (called once per minute, when initializing AuthDB).
components_auth_OAUTH_CLIENT_IDS_PROVIDER = lambda: [config.get_ui_client_id()]


def hack_windows():
  """Adds support for symlink-as-file on Windows.

  Manually resolves symlinks in path for directory and add them to sys.path.
  """
  import sys
  # Disable AppEngine's sandbox.
  from google.appengine.tools.devappserver2.python import stubs
  stubs.FakeFile.is_file_accessible = staticmethod(lambda *_: True)
  sys.meta_path = []
  for i in os.listdir('.'):
    if '.' in i or not os.path.isfile(i):
      continue
    # Found a file instead of a symlink to a directory. Adjust sys.path
    # accordingly to where the symlink points.
    with open(i) as f:
      link = f.read()
    if '\n' not in link:
      # This is not exactly right but close enough.
      sys.path.insert(
          0, os.path.dirname(os.path.normpath(os.path.abspath(link))))
  sys.path.insert(
      0,
      os.path.normpath(os.path.abspath(
          os.path.join('..', 'components', 'components', 'third_party'))))


if os.__file__[0] != '/':
  # Hack for smoke tests to pass on Windows. dev_appserver hacks the python
  # sys.platform value by setting it to 'linux3' on all platforms.
  hack_windows()
