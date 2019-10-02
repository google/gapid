# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import hashlib

from google.appengine.ext import ndb


### Models.


class ErrorReportingInfo(ndb.Model):
  """Notes the last timestamp to be used to resume collecting errors."""
  KEY_ID = 'root'

  timestamp = ndb.FloatProperty()

  @classmethod
  def primary_key(cls):
    return ndb.Key(cls, cls.KEY_ID)


class ErrorReportingMonitoring(ndb.Model):
  """Represents an error that should be limited in its verbosity.

  Key name is the hash of the error name.
  """
  created_ts = ndb.DateTimeProperty(auto_now_add=True)

  # The error string. It can be either the exception type or a single line.
  error = ndb.StringProperty(indexed=False)

  # If True, this error is silenced and never reported.
  silenced = ndb.BooleanProperty(default=False, indexed=False)

  # Silence an error for a certain amount of time. This is useful to silence a
  # error for an hour or a day when a known error condition occurs. Only one of
  # |silenced| or |silenced_until| should be set.
  silenced_until = ndb.DateTimeProperty(indexed=False)

  # Minimum number of errors that must occurs before the error is reported.
  threshold = ndb.IntegerProperty(default=0, indexed=False)

  @staticmethod
  def error_to_key_id(error):
    """Returns the key id for an error signature."""
    assert isinstance(error, unicode), repr(error)
    return hashlib.sha1(error.encode('utf-8')).hexdigest()

  @classmethod
  def error_to_key(cls, error):
    """Returns the ndb.Key for an error signature."""
    return ndb.Key(cls, cls.error_to_key_id(error))


class Error(ndb.Model):
  """Represents an error logged either by the server itself or by a client of
  the service.

  The entity is immutable once created.
  """
  created_ts = ndb.DateTimeProperty(auto_now_add=True)

  # Examples includes 'bot', 'client', 'run_isolated', 'server'.
  source = ndb.StringProperty(default='unknown')

  # Examples includes 'auth', 'exception', 'task_failure'.
  category = ndb.StringProperty()

  # Identity as seen by auth module.
  identity = ndb.StringProperty()

  # Free form message for 'auth' and 'task_failure'. In case of an exception, it
  # is the exception's text.
  message = ndb.TextProperty()

  # Set if the log entry was generated via an except clause.
  exception_type = ndb.StringProperty()
  # Will be trimmed to 4kb.
  stack = ndb.TextProperty()

  # Can be the client code version or the server version.
  version = ndb.StringProperty()

  # Can be the client code version or the server version.
  python_version = ndb.StringProperty()

  # Remote client details and endpoint accessed.
  source_ip = ndb.StringProperty()
  # The resource accessed.
  endpoint = ndb.StringProperty()
  method = ndb.StringProperty()
  params = ndb.JsonProperty(indexed=False, json_type=dict)
  # To be able to find the log back via logservice.
  request_id = ndb.StringProperty()

  # Only applicable for client-side reports.
  args = ndb.StringProperty(repeated=True)
  cwd = ndb.StringProperty()
  duration = ndb.FloatProperty()
  env = ndb.JsonProperty(indexed=False, json_type=dict)
  hostname = ndb.StringProperty()
  os = ndb.StringProperty()
  # The local user, orthogonal to authentication in self.identity.
  user = ndb.StringProperty()

