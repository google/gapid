# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Auth component configuration hooks.

Application that use 'auth' component can override settings defined here by
adding the following lines to appengine_config.py:

  components_auth_UI_APP_NAME = 'My service name'

Code flow when this is used:
  * GAE app starts and loads a module with main WSGI app.
  * This module import 'components.auth'.
  * components.auth imports components.auth.config (thus executing code here).
  * lib_config.register below imports appengine_config.py.
  * Later when code path hits auth-related code, ensure_configured is called.
  * ensure_configured calls handler.configure and auth.ui.configure.
  * Fin.
"""

import threading

from google.appengine.api import lib_config

# Used in ensure_configured.
_config_lock = threading.Lock()
_config_called = False


# Read the configuration. It would be applied later in 'ensure_configured'.
_config = lib_config.register(
    'components_auth',
    {
      # Title of the service to show in UI.
      'UI_APP_NAME': 'Auth',
      # True if application is calling 'configure_ui' manually.
      'UI_CUSTOM_CONFIG': False,
      # Module name to use for task queue tasks.
      'BACKEND_MODULE': 'default',
      # Name of the task queue that processes AuthDB diffs (see change_log.py).
      'PROCESS_CHANGE_TASK_QUEUE': 'default',
      # A callback that returns a list of OAuth client IDs to accept.
      'OAUTH_CLIENT_IDS_PROVIDER': None,
      # True to enable authentication based on 'X-Luci-Project' headers.
      'USE_PROJECT_IDENTITIES': False,
    })


def ensure_configured():
  """Applies component configuration.

  Called lazily when auth component is used for a first time.
  """
  global _config_called

  # It is python: no need for memory barrier to do this kind of check. Having it
  # here avoid hitting a bunch of locks (in imp guts and _config_lock) in code
  # executed by _every_ request.
  if _config_called:
    return _config

  # Import lazily to avoid module reference cycle.
  from .ui import ui
  from . import api

  with _config_lock:
    if not _config_called:
      api.configure_client_ids_provider(_config.OAUTH_CLIENT_IDS_PROVIDER)
      # Customize auth UI to show where it's running.
      if not _config.UI_CUSTOM_CONFIG:
        ui.configure_ui(_config.UI_APP_NAME)
      # Mark as successfully completed.
      _config_called = True
  return _config
