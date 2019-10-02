# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import copy
import datetime
import functools
import logging
import os
import sys
import time
import threading

# Not all apps enable endpoints. If the import fails, the app will not
# use @instrument_endpoint() decorator, so it is safe to ignore it.
try:
  import endpoints
except ImportError: # pragma: no cover
  pass

import webapp2

from google.appengine.api import modules
from google.appengine.api.app_identity import app_identity
from google.appengine.api import runtime
from google.appengine.ext import ndb

from infra_libs.ts_mon import handlers
from infra_libs.ts_mon import shared
from infra_libs.ts_mon.common import http_metrics
from infra_libs.ts_mon.common import interface
from infra_libs.ts_mon.common import metric_store
from infra_libs.ts_mon.common import monitors
from infra_libs.ts_mon.common import standard_metrics
from infra_libs.ts_mon.common import targets


def _reset_cumulative_metrics():
  """Clear the state when an instance loses its task_num assignment."""
  logging.warning('Instance %s got purged from Datastore, but is still alive. '
                  'Clearing cumulative metrics.', shared.instance_key_id())
  for _, metric, _, _, _ in interface.state.store.get_all():
    if metric.is_cumulative():
      metric.reset()


_flush_metrics_lock = threading.Lock()


def need_to_flush_metrics(time_now):
  """Check if metrics need flushing, and update the timestamp of last flush.

  Even though the caller of this function may not successfully flush the
  metrics, we still update the last_flushed timestamp to prevent too much work
  being done in user requests.

  Also, this check-and-update has to happen atomically, to ensure only one
  thread can flush metrics at a time.
  """
  if not interface.state.flush_enabled_fn():
    return False
  datetime_now = datetime.datetime.utcfromtimestamp(time_now)
  minute_ago = datetime_now - datetime.timedelta(seconds=60)
  with _flush_metrics_lock:
    if interface.state.last_flushed > minute_ago:
      return False
    interface.state.last_flushed = datetime_now
  return True


def flush_metrics_if_needed(time_now):
  if not need_to_flush_metrics(time_now):
    return False
  return _flush_metrics(time_now)


def _flush_metrics(time_now):
  """Return True if metrics were actually sent."""
  if interface.state.target is None:
    # ts_mon is not configured.
    return False

  datetime_now = datetime.datetime.utcfromtimestamp(time_now)
  entity = shared.get_instance_entity()
  if entity.task_num < 0:
    if interface.state.target.task_num >= 0:
      _reset_cumulative_metrics()
    interface.state.target.task_num = -1
    interface.state.last_flushed = entity.last_updated
    updated_sec_ago = (datetime_now - entity.last_updated).total_seconds()
    if updated_sec_ago > shared.INSTANCE_EXPECTED_TO_HAVE_TASK_NUM_SEC:
      logging.warning('Instance %s is %d seconds old with no task_num.',
                      shared.instance_key_id(), updated_sec_ago)
    return False
  interface.state.target.task_num = entity.task_num

  entity.last_updated = datetime_now
  entity_deferred = entity.put_async()

  interface.flush()

  for metric in interface.state.global_metrics.itervalues():
    metric.reset()

  entity_deferred.get_result()
  return True


def _shutdown_hook(time_fn=time.time):
  shared.shutdown_counter.increment()
  if flush_metrics_if_needed(time_fn()):
    logging.info('Shutdown hook: deleting %s, metrics were flushed.',
                 shared.instance_key_id())
  else:
    logging.warning('Shutdown hook: deleting %s, metrics were NOT flushed.',
                    shared.instance_key_id())
  with shared.instance_namespace_context():
    ndb.Key(shared.Instance._get_kind(), shared.instance_key_id()).delete()


def _internal_callback():
  for module_name in modules.get_modules():
    target_fields = {
        'task_num': 0,
        'hostname': '',
        'job_name': module_name,
    }
    shared.appengine_default_version.set(
        modules.get_default_version(module_name), target_fields=target_fields)


def initialize(app=None, is_enabled_fn=None, cron_module='default',
               is_local_unittest=None):
  """Instruments webapp2 `app` with gae_ts_mon metrics.

  Instruments all the endpoints in `app` with basic metrics.

  Args:
    app (webapp2 app): the app to instrument.
    is_enabled_fn (function or None): a function returning bool if ts_mon should
      send the actual metrics. None (default) is equivalent to lambda: True.
      This allows apps to turn monitoring on or off dynamically, per app.
    cron_module (str): the name of the module handling the
      /internal/cron/ts_mon/send endpoint. This allows moving the cron job
      to any module the user wants.
    is_local_unittest (bool or None): whether we are running in a unittest.
  """
  if is_local_unittest is None:  # pragma: no cover
    # Since gae_ts_mon.initialize is called at module-scope by appengine apps,
    # AppengineTestCase.setUp() won't have run yet and none of the appengine
    # stubs will be initialized, so accessing Datastore or even getting the
    # application ID will fail.
    is_local_unittest = ('expect_tests' in sys.argv[0])

  if is_enabled_fn is not None:
    interface.state.flush_enabled_fn = is_enabled_fn

  if app is not None:
    instrument_wsgi_application(app)
    if is_local_unittest or modules.get_current_module_name() == cron_module:
      instrument_wsgi_application(handlers.app)

  # Use the application ID as the service name and the module name as the job
  # name.
  if is_local_unittest:  # pragma: no cover
    service_name = 'unittest'
    job_name = 'unittest'
    hostname = 'unittest'
  else:
    service_name = app_identity.get_application_id()
    job_name = modules.get_current_module_name()
    hostname = modules.get_current_version_name()
    runtime.set_shutdown_hook(_shutdown_hook)

  interface.state.target = targets.TaskTarget(
      service_name, job_name, shared.REGION, hostname, task_num=-1)
  interface.state.flush_mode = 'manual'
  interface.state.last_flushed = datetime.datetime.utcnow()

  # Don't send metrics when running on the dev appserver.
  if (is_local_unittest or
      os.environ.get('SERVER_SOFTWARE', '').startswith('Development')):
    logging.info('Using debug monitor')
    interface.state.global_monitor = monitors.DebugMonitor()
  else:
    logging.info('Using https monitor %s with %s', shared.PRODXMON_ENDPOINT,
                 shared.PRODXMON_SERVICE_ACCOUNT_EMAIL)
    interface.state.global_monitor = monitors.HttpsMonitor(
        shared.PRODXMON_ENDPOINT,
        monitors.DelegateServiceAccountCredentials(
            shared.PRODXMON_SERVICE_ACCOUNT_EMAIL,
            monitors.AppengineCredentials()))
    interface.state.use_new_proto = True

  interface.register_global_metrics([shared.appengine_default_version])
  interface.register_global_metrics_callback(
      shared.INTERNAL_CALLBACK_NAME, _internal_callback)

  # We invoke global callbacks once for the whole application in the cron
  # handler.  Leaving this set to True would invoke them once per task.
  interface.state.invoke_global_callbacks_on_flush = False

  standard_metrics.init()

  logging.info('Initialized ts_mon with service_name=%s, job_name=%s, '
               'hostname=%s', service_name, job_name, hostname)


def _instrumented_dispatcher(dispatcher, request, response, time_fn=time.time):
  start_time = time_fn()
  response_status = 0
  flush_thread = None
  time_now = time_fn()
  if need_to_flush_metrics(time_now):
    flush_thread = threading.Thread(target=_flush_metrics, args=(time_now,))
    flush_thread.start()
  try:
    ret = dispatcher(request, response)
  except webapp2.HTTPException as ex:
    response_status = ex.code
    raise
  except Exception:
    response_status = 500
    raise
  else:
    if isinstance(ret, webapp2.Response):
      response = ret
    response_status = response.status_int
  finally:
    if flush_thread:
      flush_thread.join()
    elapsed_ms = int((time_fn() - start_time) * 1000)

    # Use the route template regex, not the request path, to prevent an
    # explosion in possible field values.
    name = request.route.template if request.route is not None else ''

    http_metrics.update_http_server_metrics(
        name, response_status, elapsed_ms,
        request_size=request.content_length,
        response_size=response.content_length,
        user_agent=request.user_agent)

  return ret


def instrument_wsgi_application(app, time_fn=time.time):
  # Don't instrument the same router twice.
  if hasattr(app.router, '__instrumented_by_ts_mon'):
    return

  old_dispatcher = app.router.dispatch

  def dispatch(router, request, response):
    return _instrumented_dispatcher(old_dispatcher, request, response,
                                    time_fn=time_fn)

  app.router.set_dispatcher(dispatch)
  app.router.__instrumented_by_ts_mon = True


def instrument_endpoint(time_fn=time.time):
  """Decorator to instrument Cloud Endpoint methods."""
  def decorator(fn):
    method_name = fn.__name__
    assert method_name
    @functools.wraps(fn)
    def decorated(service, *args, **kwargs):
      service_name = service.__class__.__name__
      endpoint_name = '/_ah/spi/%s.%s' % (service_name, method_name)
      start_time = time_fn()
      response_status = 0
      flush_thread = None
      time_now = time_fn()
      if need_to_flush_metrics(time_now):
        flush_thread = threading.Thread(target=_flush_metrics, args=(time_now,))
        flush_thread.start()
      try:
        ret = fn(service, *args, **kwargs)
        response_status = 200
        return ret
      except endpoints.ServiceException as e:
        response_status = e.http_status
        raise
      except Exception:
        response_status = 500
        raise
      finally:
        if flush_thread:
          flush_thread.join()
        elapsed_ms = int((time_fn() - start_time) * 1000)
        http_metrics.update_http_server_metrics(
            endpoint_name, response_status, elapsed_ms)
    return decorated
  return decorator


class DjangoMiddleware(object):
  STATE_ATTR = 'ts_mon_state'

  def __init__(self, time_fn=time.time):
    self._time_fn = time_fn

  def _callable_name(self, fn):
    if hasattr(fn, 'im_class') and hasattr(fn, 'im_func'):  # Bound method.
      return '.'.join([
          fn.im_class.__module__,
          fn.im_class.__name__,
          fn.im_func.func_name])
    if hasattr(fn, '__name__'):  # Function.
      return fn.__module__ + '.' + fn.__name__
    return '<unknown>'  # pragma: no cover

  def process_view(self, request, view_func, view_args, view_kwargs):
    time_now = self._time_fn()
    state = {
        'flush_thread': None,
        'name': self._callable_name(view_func),
        'start_time': time_now,
    }

    if need_to_flush_metrics(time_now):
      thread = threading.Thread(target=_flush_metrics, args=(time_now,))
      thread.start()
      state['flush_thread'] = thread

    setattr(request, self.STATE_ATTR, state)
    return None

  def process_response(self, request, response):
    try:
      state = getattr(request, self.STATE_ATTR)
    except AttributeError:
      return response

    if state['flush_thread'] is not None:
      state['flush_thread'].join()

    duration_secs = self._time_fn() - state['start_time']

    request_size = 0
    if hasattr(request, 'body'):
      request_size = len(request.body)

    response_size = 0
    if hasattr(response, 'content'):
      response_size = len(response.content)

    http_metrics.update_http_server_metrics(
        state['name'],
        response.status_code,
        duration_secs * 1000,
        request_size=request_size,
        response_size=response_size,
        user_agent=request.META.get('HTTP_USER_AGENT', None))
    return response


def reset_for_unittest(disable=False):
  interface.reset_for_unittest(disable=disable)
