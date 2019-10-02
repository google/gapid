# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Adds logging formatException() filtering to reduce the file paths length in
error logged.
"""

import logging
import os
import re

import webapp2

from google.appengine import runtime


# Paths that can be stripped from the stack traces by _relative_path().
PATHS_TO_STRIP = (
  # On AppEngine, cwd is always the application's root directory.
  os.getcwd() + os.path.sep,
  os.path.dirname(os.path.dirname(os.path.dirname(runtime.__file__))) +
      os.path.sep,
  os.path.dirname(os.path.dirname(os.path.dirname(webapp2.__file__))) +
      os.path.sep,
  # Fallback to stripping at appid.
  os.path.dirname(os.getcwd()) + os.path.sep,
  # stdlib, will contain 'python2.7' as prefix.
  os.path.dirname(os.path.dirname(os.__file__)) + os.path.sep,
  '.' + os.path.sep,
)


### Private stuff.


RE_STACK_TRACE_FILE = (
    r'^(?P<prefix>  File \")(?P<file>[^\"]+)(?P<suffix>\"\, line )'
    r'(?P<line_no>\d+)(?:|(?P<rest>\, in )(?P<function>.+))$')


def _relative_path(path):
  """Strips the current working directory or common library prefix.

  Used by _Formatter.
  """
  for i in PATHS_TO_STRIP:
    if path.startswith(i):
      return path[len(i):]
  return path


def _reformat_stack(stack):
  """Post processes the stack trace through _relative_path()."""
  out = stack.splitlines(True)
  def replace(l):
    m = re.match(RE_STACK_TRACE_FILE, l, re.DOTALL)
    if m:
      groups = list(m.groups())
      groups[1] = _relative_path(groups[1])
      return ''.join(groups)
    return l
  return ''.join(map(replace, out))


class _Formatter(object):
  """Formats exceptions nicely.

  Is is very important that this class does not throw exceptions.
  """
  def __init__(self, original):
    self._original = original

  def formatTime(self, record, datefmt=None):
    return self._original.formatTime(record, datefmt)

  def format(self, record):
    """Overrides formatException()."""
    # Cache the traceback text to avoid having the wrong formatException() be
    # called, e.g. we don't want the one in self._original.formatException() to
    # be called.
    if record.exc_info and not record.exc_text:
      record.exc_text = self.formatException(record.exc_info)
    return self._original.format(record)

  def formatException(self, exc_info):
    return _reformat_stack(self._original.formatException(exc_info))


### Public API.


def register_formatter():
  """Registers a nicer logging formatter to all currently registered handlers.

  This is optional but helps reduce spam significantly so it is highly
  encouraged.
  """
  for handler in logging.getLogger().handlers:
    # pylint: disable=W0212
    formatter = handler.formatter or logging._defaultFormatter
    if not isinstance(formatter, _Formatter):
      # Prevent immediate recursion.
      handler.setFormatter(_Formatter(formatter))
