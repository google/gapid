# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Mixed bag of utilities."""

# Disable 'Access to a protected member ...'. NDB uses '_' for other purposes.
# pylint: disable=W0212

# Disable: 'Method could be a function'. It can't: NDB expects a method.
# pylint: disable=R0201

import binascii
import datetime
import functools
import hashlib
import inspect
import json
import logging
import os
import re
import sys
import threading
import urlparse

from email import utils as email_utils

from google.appengine import runtime
from google.appengine.api import runtime as apiruntime
from google.appengine.api import app_identity
from google.appengine.api import memcache as gae_memcache
from google.appengine.api import modules
from google.appengine.api import taskqueue
from google.appengine.ext import ndb
from google.appengine.runtime import apiproxy_errors

from protorpc import messages
from protorpc.remote import protojson

THIS_DIR = os.path.dirname(os.path.abspath(__file__))

DATETIME_FORMAT = u'%Y-%m-%d %H:%M:%S'
DATE_FORMAT = u'%Y-%m-%d'
VALID_DATETIME_FORMATS = ('%Y-%m-%d', '%Y-%m-%d %H:%M', '%Y-%m-%d %H:%M:%S')


# UTC datetime corresponding to zero Unix timestamp.
EPOCH = datetime.datetime.utcfromtimestamp(0)

# Module to run task queue tasks on by default. Used by get_task_queue_host
# function. Can be changed by 'set_task_queue_module' function.
_task_queue_module = 'backend'


## GAE environment


def should_disable_ui_routes():
    return os.environ.get('LUCI_DISABLE_UI_ROUTES', '0') == '1'


def is_local_dev_server():
  """Returns True if running on local development server or in unit tests.

  This function is safe to run outside the scope of a HTTP request.
  """
  return os.environ.get('SERVER_SOFTWARE', '').startswith('Development')


def is_dev():
  """Returns True if the server is running a development/staging instance.

  We define a 'development instance' as an instance that has the suffix '-dev'
  in its instance name.

  This function is safe to run outside the scope of a HTTP request.
  """
  return os.environ['APPLICATION_ID'].endswith('-dev')


def is_unit_test():
  """Returns True if running in a unit test.

  Don't abuse it, use only if really desperate. For example, in a component that
  is included by many-many projects across many repos, when mocking some
  component behavior in all unit tests that indirectly invoke it is infeasible.
  """
  if not is_local_dev_server():
    return False
  # devappserver2 sets up some sort of a sandbox that is not activated for
  # unit tests. So differentiate based on that.
  return all(
      'google.appengine.tools.devappserver2' not in str(p)
      for p in sys.meta_path)


def _get_memory_usage():
  """Returns the amount of memory available as an float in MiB."""
  try:
    return apiruntime.runtime.memory_usage().current()
  except (AssertionError, apiproxy_errors.CancelledError):
    return None


## Handler


def get_request_as_int(request, key, default, min_value, max_value):
  """Returns a request value as int."""
  value = request.params.get(key, '')
  try:
    value = int(value)
  except ValueError:
    return default
  return min(max_value, max(min_value, value))


def report_memory(app):
  """Wraps an app so handlers log when memory usage increased by at least 0.5MB
  after the handler completed.
  """
  min_delta = 0.5
  old_dispatcher = app.router.dispatch
  def dispatch_and_report(*args, **kwargs):
    before = _get_memory_usage()
    try:
      return old_dispatcher(*args, **kwargs)
    finally:
      after = _get_memory_usage()
      if before and after and after >= before + min_delta:
        logging.debug(
            'Memory usage: %.1f -> %.1f MB; delta: %.1f MB',
            before, after, after-before)
  app.router.dispatch = dispatch_and_report


## Time


def utcnow():
  """Returns datetime.utcnow(), used for testing.

  Use this function so it can be mocked everywhere.
  """
  return datetime.datetime.utcnow()


def time_time():
  """Returns the equivalent of time.time() as mocked if applicable."""
  return (utcnow() - EPOCH).total_seconds()


def milliseconds_since_epoch(now):
  """Returns the number of milliseconds since unix epoch as an int."""
  now = now or utcnow()
  return int(round((now - EPOCH).total_seconds() * 1000.))


def datetime_to_rfc2822(dt):
  """datetime -> string value for Last-Modified header as defined by RFC2822."""
  if not isinstance(dt, datetime.datetime):
    raise TypeError(
        'Expecting datetime object, got %s instead' % type(dt).__name__)
  assert dt.tzinfo is None, 'Expecting UTC timestamp: %s' % dt
  return email_utils.formatdate(datetime_to_timestamp(dt) / 1000000.0)


def datetime_to_timestamp(value):
  """Converts UTC datetime to integer timestamp in microseconds since epoch."""
  if not isinstance(value, datetime.datetime):
    raise ValueError(
        'Expecting datetime object, got %s instead' % type(value).__name__)
  if value.tzinfo is not None:
    raise ValueError('Only UTC datetime is supported')
  dt = value - EPOCH
  return dt.microseconds + 1000 * 1000 * (dt.seconds + 24 * 3600 * dt.days)


def timestamp_to_datetime(value):
  """Converts integer timestamp in microseconds since epoch to UTC datetime."""
  if not isinstance(value, (int, long, float)):
    raise ValueError(
        'Expecting a number, got %s instead' % type(value).__name__)
  return EPOCH + datetime.timedelta(microseconds=value)


def parse_datetime(text):
  """Converts text to datetime.datetime instance or None."""
  for f in VALID_DATETIME_FORMATS:
    try:
      return datetime.datetime.strptime(text, f)
    except ValueError:
      continue
  return None


def parse_rfc3339_datetime(value):
  """Parses RFC 3339 datetime string (as used in Timestamp proto JSON encoding).

  Keeps only microsecond precision (dropping nanoseconds).

  Examples of the input:
    2017-08-17T04:21:32.722952943Z
    1972-01-01T10:00:20.021-05:00

  Returns:
    datetime.datetime in UTC (regardless of timezone of the original string).

  Raises:
    ValueError on errors.
  """
  # Adapted from protobuf/internal/well_known_types.py Timestamp.FromJsonString.
  # We can't use the original, since it's marked as internal. Also instantiating
  # proto messages here to parse a string would been odd.
  timezone_offset = value.find('Z')
  if timezone_offset == -1:
    timezone_offset = value.find('+')
  if timezone_offset == -1:
    timezone_offset = value.rfind('-')
  if timezone_offset == -1:
    raise ValueError('Failed to parse timestamp: missing valid timezone offset')
  time_value = value[0:timezone_offset]
  # Parse datetime and nanos.
  point_position = time_value.find('.')
  if point_position == -1:
    second_value = time_value
    nano_value = ''
  else:
    second_value = time_value[:point_position]
    nano_value = time_value[point_position + 1:]
  date_object = datetime.datetime.strptime(second_value, '%Y-%m-%dT%H:%M:%S')
  td = date_object - EPOCH
  seconds = td.seconds + td.days * 86400
  if len(nano_value) > 9:
    raise ValueError(
        'Failed to parse timestamp: nanos %r more than 9 fractional digits'
        % nano_value)
  if nano_value:
    nanos = round(float('0.' + nano_value) * 1e9)
  else:
    nanos = 0
  # Parse timezone offsets.
  if value[timezone_offset] == 'Z':
    if len(value) != timezone_offset + 1:
      raise ValueError(
          'Failed to parse timestamp: invalid trailing data %r' % value)
  else:
    timezone = value[timezone_offset:]
    pos = timezone.find(':')
    if pos == -1:
      raise ValueError('Invalid timezone offset value: %r' % timezone)
    if timezone[0] == '+':
      seconds -= (int(timezone[1:pos])*60+int(timezone[pos+1:]))*60
    else:
      seconds += (int(timezone[1:pos])*60+int(timezone[pos+1:]))*60
  return timestamp_to_datetime(int(seconds)*1e6 + int(nanos)/1e3)


def constant_time_equals(a, b):
  """Compares two strings in constant time regardless of theirs content."""
  if len(a) != len(b):
    return False
  result = 0
  for x, y in zip(a, b):
    result |= ord(x) ^ ord(y)
  return result == 0


## Cache


class _Cache(object):
  """Holds state of a cache for cache_with_expiration and cache decorators.

  May call func more than once.
  Thread- and NDB tasklet-safe.
  """

  def __init__(self, func, expiration_sec):
    self.func = func
    self.expiration_sec = expiration_sec
    self.lock = threading.Lock()
    self.value = None
    self.value_is_set = False
    self.expires = None

  def get_value(self):
    """Returns a cached value refreshing it if it has expired."""
    with self.lock:
      if self.value_is_set and (not self.expires or time_time() < self.expires):
        return self.value

    new_value = self.func()

    with self.lock:
      self.value = new_value
      self.value_is_set = True
      if self.expiration_sec:
        self.expires = time_time() + self.expiration_sec

    return self.value

  def clear(self):
    """Clears stored cached value."""
    with self.lock:
      self.value = None
      self.value_is_set = False
      self.expires = None

  def get_wrapper(self):
    """Returns a callable object that can be used in place of |func|.

    It's basically self.get_value, updated by functools.wraps to look more like
    original function.
    """
    # functools.wraps doesn't like 'instancemethod', use lambda as a proxy.
    # pylint: disable=W0108
    wrapper = functools.wraps(self.func)(lambda: self.get_value())
    wrapper.__parent_cache__ = self
    return wrapper


def cache(func):
  """Decorator that implements permanent cache of a zero-parameter function."""
  return _Cache(func, None).get_wrapper()


def cache_with_expiration(expiration_sec):
  """Decorator that implements in-memory cache for a zero-parameter function."""
  def decorator(func):
    return _Cache(func, expiration_sec).get_wrapper()
  return decorator


def clear_cache(func):
  """Given a function decorated with @cache, resets cached value."""
  func.__parent_cache__.clear()


# ignore time parameter warning | pylint: disable=redefined-outer-name
def memcache_async(key, key_args=None, time=None):
  """Decorator that implements memcache-based cache for a function.

  The generated cache key contains current application version and values of
  |key_args| arguments converted to string using `repr`.

  Args:
    key (str): unique string that will be used as a part of cache key.
    key_args (list of str): list of function argument names to include
      in the generated cache key.
    time (int): optional expiration time.

  Example:
    @memcache('f', ['a', 'b'])
    def f(a, b=2, not_used_in_cache_key=6):
      # Heavy computation
      return 42

  Decorator raises:
    NotImplementedError if function uses varargs or kwargs.
  """
  assert isinstance(key, basestring), key
  key_args = key_args or []
  assert isinstance(key_args, list), key_args
  assert all(isinstance(a, basestring) for a in key_args), key_args
  assert all(key_args), key_args

  memcache_set_kwargs = {}
  if time is not None:
    memcache_set_kwargs['time'] = time

  def decorator(func):
    unwrapped = func
    while True:
      deeper = getattr(unwrapped, '__wrapped__', None)
      if not deeper:
        break
      unwrapped = deeper

    argspec = inspect.getargspec(unwrapped)
    if argspec.varargs:
      raise NotImplementedError(
          'varargs in memcached functions are not supported')
    if argspec.keywords:
      raise NotImplementedError(
          'kwargs in memcached functions are not supported')

    # List of arg names and indexes. Has same order as |key_args|.
    arg_indexes = []
    for name in key_args:
      try:
        i = argspec.args.index(name)
      except ValueError:
        raise KeyError(
            'key_format expects "%s" parameter, but it was not found among '
            'function parameters' % name)
      arg_indexes.append((name, i))

    @functools.wraps(func)
    @ndb.tasklet
    def decorated(*args, **kwargs):
      arg_values = []
      for name, i in arg_indexes:
        if i < len(args):
          arg_value = args[i]
        elif name in kwargs:
          arg_value = kwargs[name]
        else:
          # argspec.defaults contains _last_ default values, so we need to shift
          # |i| left.
          default_value_index = i - (len(argspec.args) - len(argspec.defaults))
          if default_value_index < 0:
            # Parameter not provided. Call function to cause TypeError
            func(*args, **kwargs)
            assert False, 'Function call did not fail'
          arg_value = argspec.defaults[default_value_index]
        arg_values.append(arg_value)

      # Instead of putting a raw value to memcache, put tuple (value,)
      # so we can distinguish a cached None value and absence of the value.

      cache_key = 'utils.memcache/%s/%s%s' % (
          get_app_version(), key, repr(arg_values))

      ctx = ndb.get_context()
      result = yield ctx.memcache_get(cache_key)
      if isinstance(result, tuple) and len(result) == 1:
        raise ndb.Return(result[0])

      result = func(*args, **kwargs)
      if isinstance(result, ndb.Future):
        result = yield result
      yield ctx.memcache_set(cache_key, (result,), **memcache_set_kwargs)
      raise ndb.Return(result)

    return decorated
  return decorator


def memcache(*args, **kwargs):
  """Blocking version of memcache_async."""
  decorator_async = memcache_async(*args, **kwargs)
  def decorator(func):
    decorated_async = decorator_async(func)
    @functools.wraps(func)
    def decorated(*args, **kwargs):
      return decorated_async(*args, **kwargs).get_result()
    return decorated
  return decorator


## GAE identity


@cache
def get_app_version():
  """Returns currently running version (not necessary a default one)."""
  # Sadly, this causes an RPC and when called too frequently, throws quota
  # errors.
  return modules.get_current_version_name() or 'N/A'


@cache
def get_versioned_hosturl():
  """Returns the url hostname of this instance locked to the currently running
  version.

  This function hides the fact that app_identity.get_default_version_hostname()
  returns None on the dev server and modules.get_hostname() returns incorrectly
  qualified hostname for HTTPS usage on the prod server. <3
  """
  if is_local_dev_server():
    # TODO(maruel): It'd be nice if it were easier to use a ephemeral SSL
    # certificate here and not assume unsecured connection.
    return 'http://' + modules.get_hostname()

  return 'https://%s-dot-%s' % (
      get_app_version(), app_identity.get_default_version_hostname())


@cache
def get_urlfetch_service_id():
  """Returns a value for X-URLFetch-Service-Id header for GAE <-> GAE calls.

  Usually it can be omitted. It is required in certain environments.
  """
  if is_local_dev_server():
    return 'LOCAL'
  hostname = app_identity.get_default_version_hostname().split('.')
  return hostname[-2].upper() if len(hostname) >= 3 else 'APPSPOT'


@cache
def get_app_revision_url():
  """Returns URL of a git revision page for currently running app version.

  Works only for non-tainted versions uploaded with tools/update.py: app version
  should look like '162-efaec47'. Assumes all services that use 'components'
  live in a single repository.

  Returns None if a version is tainted or has unexpected name.
  """
  rev = re.match(r'\d+-([a-f0-9]+)$', get_app_version())
  template = 'https://chromium.googlesource.com/infra/luci/luci-py/+/%s'
  return template % rev.group(1) if rev else None


@cache
def get_service_account_name():
  """Same as app_identity.get_service_account_name(), but caches the result.

  app_identity.get_service_account_name() does an RPC on each call, yet the
  result is always the same.
  """
  return app_identity.get_service_account_name()


def get_module_version_list(module_list, tainted):
  """Returns a list of pairs (module name, version name) to fetch logs for.

  Arguments:
    module_list: list of modules to list, defaults to all modules.
    tainted: if False, excludes versions with '-tainted' in their name.
  """
  result = []
  if not module_list:
    # If the function it called too often, it'll raise a OverQuotaError. So
    # cache it for 10 minutes.
    module_list = gae_memcache.get('modules_list')
    if not module_list:
      module_list = modules.get_modules()
      gae_memcache.set('modules_list', module_list, time=10*60)

  for module in module_list:
    # If the function it called too often, it'll raise a OverQuotaError.
    # Versions is a bit more tricky since we'll loose data, since versions are
    # changed much more often than modules. So cache it for 1 minute.
    key = 'modules_list-' + module
    version_list = gae_memcache.get(key)
    if not version_list:
      version_list = modules.get_versions(module)
      gae_memcache.set(key, version_list, time=60)
    result.extend(
        (module, v) for v in version_list if tainted or '-tainted' not in v)
  return result


## Task queue


@cache
def get_task_queue_host():
  """Returns domain name of app engine instance to run a task queue task on.

  By default will use 'backend' module. Can be changed by calling
  set_task_queue_module during application startup.

  This domain name points to a matching version of appropriate app engine
  module - <version>.<module>.<app-id>.appspot.com where:
    version: version of the module that is calling this function.
    module: app engine module to execute task on.

  That way a task enqueued from version 'A' of default module would be executed
  on same version 'A' of backend module.
  """
  # modules.get_hostname sometimes fails with unknown internal error.
  # Cache its result in a memcache to avoid calling it too often.
  cache_key = 'task_queue_host:%s:%s' % (_task_queue_module, get_app_version())
  value = gae_memcache.get(cache_key)
  if not value:
    value = modules.get_hostname(module=_task_queue_module)
    gae_memcache.set(cache_key, value)
  return value


def set_task_queue_module(module):
  """Changes a module used by get_task_queue_host() function.

  Should be called during application initialization if default 'backend' module
  is not appropriate.
  """
  global _task_queue_module
  _task_queue_module = module
  clear_cache(get_task_queue_host)


@ndb.tasklet
def enqueue_task_async(
    url,
    queue_name,
    params=None,
    payload=None,
    name=None,
    countdown=None,
    use_dedicated_module=True,
    transactional=False):
  """Adds a task to a task queue.

  If |use_dedicated_module| is True (default) the task will be executed by
  a separate backend module instance that runs same version as currently
  executing instance. Otherwise it will run on a current version of default
  module.

  Returns True if the task was successfully added or a task with such name
  existed before (i.e. on TombstonedTaskError exception): deduplicated task is
  not a error.

  Logs an error and returns False if task queue is acting up.
  """
  try:
    headers = None
    if use_dedicated_module:
      headers = {'Host': get_task_queue_host()}
    # Note that just using 'target=module' here would redirect task request to
    # a default version of a module, not the curently executing one.
    task = taskqueue.Task(
        url=url,
        params=params,
        payload=payload,
        name=name,
        countdown=countdown,
        headers=headers)
    yield task.add_async(queue_name=queue_name, transactional=transactional)
    raise ndb.Return(True)
  except (taskqueue.TombstonedTaskError, taskqueue.TaskAlreadyExistsError):
    logging.info(
        'Task %r deduplicated (already exists in queue %r)',
        name, queue_name)
    raise ndb.Return(True)
  except (
      taskqueue.Error,
      runtime.DeadlineExceededError,
      runtime.apiproxy_errors.CancelledError,
      runtime.apiproxy_errors.DeadlineExceededError,
      runtime.apiproxy_errors.OverQuotaError) as e:
    logging.warning(
        'Problem adding task %r to task queue %r (%s): %s',
        url, queue_name, e.__class__.__name__, e)
    raise ndb.Return(False)


def enqueue_task(*args, **kwargs):
  """Adds a task to a task queue.

  Returns:
    True if the task was enqueued, False otherwise.
  """
  return enqueue_task_async(*args, **kwargs).get_result()


## JSON


def to_json_encodable(data):
  """Converts data into json-compatible data."""
  if isinstance(data, messages.Message):
    # protojson.encode_message returns a string that is already encoded json.
    # Load it back into a json-compatible representation of the data.
    return json.loads(protojson.encode_message(data))
  if isinstance(data, unicode) or data is None:
    return data
  if isinstance(data, str):
    return data.decode('utf-8')
  if isinstance(data, (int, float, long)):
    # Note: overflowing is an issue with int and long.
    return data
  if isinstance(data, (list, set, tuple)):
    return [to_json_encodable(i) for i in data]
  if isinstance(data, dict):
    assert all(isinstance(k, basestring) for k in data), data
    return {
      to_json_encodable(k): to_json_encodable(v) for k, v in data.iteritems()
    }

  if isinstance(data, datetime.datetime):
    # Convert datetime objects into a string, stripping off milliseconds. Only
    # accept naive objects.
    if data.tzinfo is not None:
      raise ValueError('Can only serialize naive datetime instance')
    return data.strftime(DATETIME_FORMAT)
  if isinstance(data, datetime.date):
    return data.strftime(DATE_FORMAT)
  if isinstance(data, datetime.timedelta):
    # Convert timedelta into seconds, stripping off milliseconds.
    return int(data.total_seconds())

  if hasattr(data, 'to_dict') and callable(data.to_dict):
    # This takes care of ndb.Model.
    return to_json_encodable(data.to_dict())

  if hasattr(data, 'urlsafe') and callable(data.urlsafe):
    # This takes care of ndb.Key.
    return to_json_encodable(data.urlsafe())

  if inspect.isgenerator(data) or isinstance(data, xrange):
    # Handle it like a list. Sadly, xrange is not a proper generator so it has
    # to be checked manually.
    return [to_json_encodable(i) for i in data]

  assert False, 'Don\'t know how to handle %r' % data


def encode_to_json(data):
  """Converts any data as a json string."""
  return json.dumps(
      to_json_encodable(data),
      sort_keys=True,
      separators=(',', ':'),
      encoding='utf-8')


## General


def to_units(number):
  """Convert a string to numbers."""
  UNITS = ('', 'k', 'm', 'g', 't', 'p', 'e', 'z', 'y')
  unit = 0
  while number >= 1024.:
    unit += 1
    number = number / 1024.
    if unit == len(UNITS) - 1:
      break
  if unit:
    return '%.2f%s' % (number, UNITS[unit])
  return '%d' % number


def validate_root_service_url(url):
  """Raises ValueError if the URL doesn't look like https://<host>."""
  schemes = ('https', 'http') if is_local_dev_server() else ('https',)
  parsed = urlparse.urlparse(url)
  if parsed.scheme not in schemes:
    raise ValueError('unsupported protocol %r' % str(parsed.scheme))
  if not parsed.netloc:
    raise ValueError('missing hostname')
  stripped = urlparse.urlunparse((parsed[0], parsed[1], '', '', '', ''))
  if stripped != url:
    raise ValueError('expecting root host URL, e.g. %r)' % str(stripped))


def get_token_fingerprint(blob):
  """Given a blob with a token returns first 16 bytes of its SHA256 as hex.

  It can be used to identify this particular token in logs without revealing it.
  """
  assert isinstance(blob, basestring)
  if isinstance(blob, unicode):
    blob = blob.encode('ascii', 'ignore')
  return binascii.hexlify(hashlib.sha256(blob).digest()[:16])


## Hacks


def fix_protobuf_package():
  """Modifies 'google' package to include path to 'google.protobuf' package.

  Prefer our own proto package on the server. Note that this functions is not
  used on the Swarming bot nor any other client.
  """
  # google.__path__[0] will be google_appengine/google.
  import google
  if len(google.__path__) > 1:
    return

  # We do not mind what 'google' get used, inject protobuf in there.
  path = os.path.join(THIS_DIR, 'third_party', 'protobuf', 'google')
  google.__path__.append(path)

  # six is needed for oauth2client and webtest (local testing).
  six_path = os.path.join(THIS_DIR, 'third_party', 'six')
  if six_path not in sys.path:
    sys.path.insert(0, six_path)


def import_jinja2():
  """Remove any existing jinja2 package and add ours."""
  for i in sys.path[:]:
    if os.path.basename(i) == 'jinja2':
      sys.path.remove(i)
  sys.path.append(os.path.join(THIS_DIR, 'third_party'))


# NDB Futures


def async_apply(iterable, async_fn, unordered=False, concurrent_jobs=50):
  """Applies async_fn to each item and yields (item, result) tuples.

  Args:
    iterable: an iterable of items for which to call async_fn
    async_fn: (item) => ndb.Future. It is called for each item in iterable.
    unordered: False to return results in the same order as iterable.
      Otherwise, yield results as soon as futures finish.
    concurrent_jobs: maximum number of futures running concurrently.
  """
  if unordered:
    return _async_apply_unordered(iterable, async_fn, concurrent_jobs)
  return _async_apply_ordered(iterable, async_fn, concurrent_jobs)


def _async_apply_ordered(iterable, async_fn, concurrent_jobs):
  results = _async_apply_unordered(
      enumerate(iterable),
      lambda (i, item): async_fn(item),
      concurrent_jobs)
  for (_, item), result in sorted(results, key=lambda i: i[0][0]):
    yield item, result


def _async_apply_unordered(iterable, async_fn, concurrent_jobs):
  # maps a future to the original item(s). Items is a list because async_fn
  # is allowed to return the same future for different items.
  futs = {}
  iterator = iter(iterable)

  def launch():
    running_futs = sum(1 for f in futs if not f.done())
    while running_futs < concurrent_jobs:
      try:
        item = next(iterator)
      except StopIteration:
        break
      future = async_fn(item)
      if not future.done():
        running_futs += 1
      futs.setdefault(future, []).append(item)

  launch()
  while futs:
    future = ndb.Future.wait_any(futs)
    res = future.get_result()
    launch()  # launch more before yielding
    for item in futs.pop(future):
      yield item, res


def sync_of(async_fn):
  """Returns a synchronous version of an asynchronous function."""
  is_static_method = isinstance(async_fn, staticmethod)
  is_class_method = isinstance(async_fn, classmethod)
  if is_static_method or is_class_method:
    async_fn = async_fn.__func__

  @functools.wraps(async_fn)
  def sync(*args, **kwargs):
    return async_fn(*args, **kwargs).get_result()

  if is_static_method:
    sync = staticmethod(sync)
  elif is_class_method:
    sync = classmethod(sync)
  return sync
