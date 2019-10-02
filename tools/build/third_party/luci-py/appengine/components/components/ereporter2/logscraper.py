# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Backend functions to gather error reports."""

import collections
import hashlib
import logging
import os
import re

import webob

from google.appengine.api import logservice

from components import utils

from . import formatter
from . import models


# Silence this error message specifically. There's no action item here.
SOFT_MEMORY_EXCEEDED = u'Exceeded soft memory limit'

# These are stronger statements, so do not silence them, but still group them.
MEMORY_EXCEEDED_PREFIXES = (
    u'Exceeded medium memory limit',
    u'Exceeded hard memory limit',
)

MEMORY_EXCEEDED = u'Exceeded memory limit'


### Private constants.


# Markers to read back a stack trace.
_STACK_TRACE_MARKER = u'Traceback (most recent call last):'


# Number of first error records to show in the category error list.
_ERROR_LIST_HEAD_SIZE = 10
# Number of last error records to show in the category error list.
_ERROR_LIST_TAIL_SIZE = 10


### Private suff.


class _CappedList(object):
  """List of objects with only several first ones and several last ones
  actually stored.

  Basically a structure for:
  0 1 2 3 ..... N-3 N-2 N-1 N
  """

  def __init__(self, head_size, tail_size, items=None):
    assert head_size > 0, head_size
    assert tail_size > 0, tail_size
    self.head_size = head_size
    self.tail_size = tail_size
    self.head = []
    self.tail = collections.deque()
    self.total_count = 0
    for item in (items or []):
      self.append(item)

  def append(self, item):
    """Adds item to the list.

    If list is small enough, will add it to the head, else to the tail
    (evicting oldest stored tail item).
    """
    # Keep count of all elements ever added (even though they may not be
    # actually stored).
    self.total_count += 1
    # List is still short, grow head.
    if len(self.head) < self.head_size:
      self.head.append(item)
    else:
      # List is long enough to start using tail. Grow tail, but keep only
      # end of it.
      if len(self.tail) == self.tail_size:
        self.tail.popleft()
      self.tail.append(item)

  def __iter__(self):
    for i in self.head:
      yield i
    for i in self.tail:
      yield i

  def __len__(self):
    return self.total_count

  def __neg__(self):
    return not bool(self.total_count)

  @property
  def has_gap(self):
    """True if this list contains skipped middle section."""
    return len(self.head) + len(self.tail) < self.total_count


class _ErrorCategory(object):
  """Describes a 'class' of error messages' according to an unique signature."""
  def __init__(self, signature):
    assert isinstance(signature, unicode), signature
    # Remove the version embedded in the signature.
    self.signature = signature
    self.events = _CappedList(_ERROR_LIST_HEAD_SIZE, _ERROR_LIST_TAIL_SIZE)
    self._exception_type = None

  def append_error(self, error):
    if not self.events:
      assert self._exception_type is None
      self._exception_type = error.exception_type
    self.events.append(error)

  @property
  def exception_type(self):
    return self._exception_type

  @property
  def messages(self):
    return sorted(set(e.message for e in self.events))

  @property
  def modules(self):
    return sorted(set(e.module for e in self.events))

  @property
  def resources(self):
    return sorted(set(e.resource for e in self.events))

  @property
  def versions(self):
    return sorted(set(e.version for e in self.events))


class _ErrorRecord(object):
  """Describes the context in which an error was logged."""

  # Use slots to reduce memory footprint of _ErrorRecord object.
  __slots__ = (
      'request_id', 'start_time', 'exception_time', 'latency', 'mcycles', 'ip',
      'nickname', 'referrer', 'user_agent', 'host', 'resource', 'method',
      'task_queue_name', 'was_loading_request', 'version', 'module',
      'handler_module', 'gae_version', 'instance', 'status', 'message',
      'exception_type', 'signature')

  def __init__(
      self, request_id, start_time, exception_time, latency, mcycles,
      ip, nickname, referrer, user_agent,
      host, resource, method, task_queue_name,
      was_loading_request, version, module, handler_module, gae_version,
      instance,
      status, message, signature, exception_type):
    assert isinstance(message, unicode), repr(message)
    # Unique identifier.
    self.request_id = request_id
    # Initial time the request was handled.
    self.start_time = start_time
    # Time of the exception.
    self.exception_time = exception_time
    # Wall-clock time duration of the request.
    self.latency = latency
    # CPU usage in mega-cyclers.
    self.mcycles = mcycles

    # Who called.
    self.ip = ip
    self.nickname = nickname
    self.referrer = referrer
    self.user_agent = user_agent

    # What was called.
    self.host = host
    self.resource = resource
    self.method = method
    self.task_queue_name = task_queue_name

    # What handled the call.
    self.was_loading_request = was_loading_request
    self.version = version
    self.module = module
    self.handler_module = handler_module
    self.gae_version = gae_version
    self.instance = instance

    # What happened.
    self.status = status
    self.message = message
    self.signature = signature
    self.exception_type = exception_type
    if not self.signature:
      # Default the signature to the exception type if None.
      self.signature = self.exception_type
    assert isinstance(self.signature, unicode), repr(self.signature)


def _shorten(l):
  assert isinstance(l, unicode), repr(l)
  if len(l) > 256:
    return u'hash:%s' % hashlib.sha1(l.encode('utf-8')).hexdigest()
  return l


def _signature_from_message(message):
  """Calculates a signature and extract the exception if any.

  Arguments:
    message: a complete log entry potentially containing a stack trace.

  Returns:
    tuple of a signature and the exception type, if any.
  """
  assert isinstance(message, unicode), repr(message)
  lines = message.splitlines()
  if not lines:
    return '', None

  if _STACK_TRACE_MARKER not in lines:
    # Not an exception. Use the first line as the 'signature'.

    # Look for special messages to reduce.
    if lines[0].startswith(SOFT_MEMORY_EXCEEDED):
      # Ignore soft memory, it's just noise.
      return '', None
    if lines[0].startswith(MEMORY_EXCEEDED_PREFIXES):
      # Consider (non-soft) memory exceeded an 'exception'.
      return MEMORY_EXCEEDED, MEMORY_EXCEEDED
    return _shorten(lines[0].strip()), None

  # It is a stack trace.
  stacktrace = []
  index = lines.index(_STACK_TRACE_MARKER) + 1
  while index < len(lines):
    if not re.match(formatter.RE_STACK_TRACE_FILE, lines[index]):
      break
    if (len(lines) > index + 1 and
        re.match(formatter.RE_STACK_TRACE_FILE, lines[index+1])):
      # It happens occasionally with jinja2 templates.
      stacktrace.append(lines[index])
      index += 1
    else:
      stacktrace.extend(lines[index:index+2])
      index += 2

  if index >= len(lines):
    # Failed at grabbing the exception.
    return _shorten(lines[0].strip()), None

  # SyntaxError produces this.
  if lines[index].strip() == '^':
    index += 1

  assert index > 0
  while True:
    ex_type = lines[index].split(':', 1)[0].strip()
    if ex_type:
      break
    if not index:
      logging.error('Failed to process message.\n%s', message)
      # Fall back to returning the first line.
      return _shorten(lines[0].strip()), None
    index -= 1

  function = None
  path = None
  line_no = -1
  for l in reversed(stacktrace):
    m = re.match(formatter.RE_STACK_TRACE_FILE, l)
    if m:
      if not path:
        path = os.path.basename(m.group('file'))
        line_no = int(m.group('line_no'))
      if m.group('file').startswith(('appengine', 'python2.7', 'third_party')):
        continue
      function = m.group('function')
      break
  if function:
    signature = '%s@%s' % (ex_type, function)
  else:
    signature = '%s@%s:%d' % (ex_type, path, line_no)
  return _shorten(signature), ex_type


def _extract_exceptions_from_logs(start_time, end_time, module_versions):
  """Yields _ErrorRecord objects from the logs.

  Arguments:
    start_time: epoch time to start searching. If 0 or None, defaults to
                1970-01-01.
    end_time: epoch time to stop searching. If 0 or None, defaults to
              time.time().
    module_versions: list of tuple of module-version to gather info about.
  """
  if start_time and end_time and start_time >= end_time:
    raise webob.exc.HTTPBadRequest(
        'Invalid range, start_time must be before end_time.')
  try:
    for entry in logservice.fetch(
        start_time=start_time or None,
        end_time=end_time or None,
        minimum_log_level=logservice.LOG_LEVEL_ERROR,
        include_incomplete=True,
        include_app_logs=True,
        module_versions=module_versions):
      # Merge all error messages. The main reason to do this is that sometimes
      # a single logging.error() 'Traceback' is split on each line as an
      # individual log_line entry.
      msgs = []
      log_time = None
      for log_line in entry.app_logs:
        # TODO(maruel): Specifically handle:
        # 'Request was aborted after waiting too long to attempt to service your
        # request.'
        # For an unknown reason, it is logged at level info (!?)
        if log_line.level < logservice.LOG_LEVEL_ERROR:
          continue
        msg = log_line.message.strip('\n')
        if not msg.strip():
          continue
        # The message here is assumed to be utf-8 encoded but that is not
        # guaranteed. The dashboard does prints out utf-8 log entries properly.
        try:
          msg = msg.decode('utf-8')
        except UnicodeDecodeError:
          msg = msg.decode('ascii', 'replace')
        msgs.append(msg)
        log_time = log_time or log_line.time

      message = '\n'.join(msgs)
      # Creates a unique signature string based on the message.
      signature, exception_type = _signature_from_message(message)
      if exception_type:
        yield _ErrorRecord(
            entry.request_id,
            entry.start_time, log_time, entry.latency, entry.mcycles,
            entry.ip, entry.nickname, entry.referrer, entry.user_agent,
            entry.host, entry.resource, entry.method, entry.task_queue_name,
            entry.was_loading_request, entry.version_id, entry.module_id,
            entry.url_map_entry, entry.app_engine_release, entry.instance_key,
            entry.status, message, signature, exception_type)
  except logservice.Error as e:
    # It's not worth generating an error log when logservice is temporarily
    # down. Retrying is not worth either.
    logging.warning('Failed to scrape log:\n%s', e)


def _should_ignore_error_category(monitoring, error_category):
  """Returns True if an _ErrorCategory should be ignored."""
  if not monitoring:
    return False
  if monitoring.silenced:
    return True
  if (monitoring.silenced_until and
      monitoring.silenced_until >= utils.utcnow()):
    return True
  if (monitoring.threshold and len(error_category.events) <
      monitoring.threshold):
    return True
  return False


def _log_request_id(request_id):
  """Returns a logservice.RequestLog for a request id or None if not found."""
  request = list(logservice.fetch(
      include_incomplete=True, include_app_logs=True, request_ids=[request_id]))
  if not request:
    logging.info('Dang, didn\'t find the request_id %s', request_id)
    return None
  assert len(request) == 1, request
  return request[0]


### Public API.


def scrape_logs_for_errors(start_time, end_time, module_versions):
  """Returns a list of _ErrorCategory to generate a report.

  Arguments:
    start_time: time to look for report, defaults to last email sent.
    end_time: time to end the search for error, defaults to now.
    module_versions: list of tuple of module-version to gather info about.

  Returns:
    tuple of 3 items:
      - list of _ErrorCategory that should be reported
      - list of _ErrorCategory that should be ignored
      - end_time of the last item processed if not all items were processed or
        |end_time|
  """
  # Scan for up to 9 minutes. This function is assumed to be run by a backend
  # (cron job or task queue) which has a 10 minutes deadline. This leaves ~1
  # minute to the caller to send an email and update the DB entity.
  start = utils.time_time()

  # In practice, we don't expect more than ~100 entities.
  filters = {
    e.key.string_id(): e for e in models.ErrorReportingMonitoring.query()
  }

  # Gather all the error categories.
  buckets = {}
  for error_record in _extract_exceptions_from_logs(
      start_time, end_time, module_versions):
    bucket = buckets.setdefault(
        error_record.signature, _ErrorCategory(error_record.signature))
    bucket.append_error(error_record)
    # Abort, there's too much logs.
    if (utils.time_time() - start) >= 9*60:
      end_time = error_record.start_time
      break

  # Filter them.
  categories = []
  ignored = []
  for category in buckets.itervalues():
    # Ignore either the exception or the signature. Signature takes precedence.
    f = filters.get(models.ErrorReportingMonitoring.error_to_key_id(
        category.signature))
    if not f and category.exception_type:
      f = filters.get(models.ErrorReportingMonitoring.error_to_key_id(
          category.exception_type))
    if _should_ignore_error_category(f, category):
      ignored.append(category)
    else:
      categories.append(category)

  return categories, ignored, end_time
