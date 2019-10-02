# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Manages user reports.

These are DB stored reports, vs logservice based reports. This is for events
like client failure or non exceptional server failure where more details is
desired.
"""

import datetime
import logging
import platform
import traceback

from google.appengine.api import app_identity
from google.appengine.api import datastore_errors
from google.appengine.api.logservice import logsutil

from components import auth
from components import utils

from . import formatter
from . import models


# Amount of time to keep error logs around.
ERROR_TIME_TO_LIVE = datetime.timedelta(days=30)


# Keys that can be specified by the client.
# pylint: disable=W0212
VALID_ERROR_KEYS = frozenset(models.Error._properties) - frozenset(
    ['created_ts', 'identity'])


def log(**kwargs):
  """Adds an error. This will indirectly notify the admins.

  Returns the entity id for the report.
  """
  identity = None
  if not auth.get_current_identity().is_anonymous:
    identity = auth.get_current_identity().to_bytes()
  try:
    # Trim all the messages to 4kb to reduce spam.
    LIMIT = 4096
    for key, value in kwargs.items():
      if key not in VALID_ERROR_KEYS:
        logging.error('Dropping unknown detail %s: %s', key, value)
        kwargs.pop(key)
      elif isinstance(value, basestring) and len(value) > LIMIT:
        value = value[:LIMIT-1] + u'\u2026'
        kwargs[key] = value

    if kwargs.get('source') == 'server':
      # Automatically use the version of the server code.
      kwargs.setdefault('version', utils.get_app_version())
      kwargs.setdefault('python_version', platform.python_version())

    error = models.Error(identity=identity, **kwargs)
    error.put()
    key_id = error.key.integer_id()
    # The format of the message is important here. The first line is used to
    # generate a signature, so it must be unique for each category of errors.
    logging.error(
        '%s\n\nSource: %s\nhttps://%s/restricted/ereporter2/errors/%s',
        error.message,
        error.source,
        app_identity.get_default_version_hostname(),
        key_id)
    return key_id
  except (datastore_errors.BadValueError, TypeError) as e:
    stack = formatter._reformat_stack(traceback.format_exc())
    # That's the error about the error.
    error = models.Error(
        source='server',
        category='exception',
        message='log(%s) caused: %s' % (kwargs, str(e)),
        exception_type=str(type(e)),
        stack=stack)
    error.put()
    key_id = error.key.integer_id()
    logging.error(
        'Failed to log a %s error\n%s\n%s', error.source, key_id, error.message)
    return key_id


def log_request(request, add_params=True, **kwargs):
  """Adds an error. This should be used normally."""
  kwargs['endpoint'] = request.path
  kwargs['method'] = request.method
  kwargs['request_id'] = logsutil.RequestID()
  kwargs['source_ip'] = request.remote_addr
  if add_params:
    kwargs['params'] = request.params.mixed()
    try:
      as_json = request.json
      if isinstance(as_json, dict):
        kwargs['params'].update(as_json)
    except (LookupError, TypeError, ValueError):
      pass
  return log(**kwargs)
