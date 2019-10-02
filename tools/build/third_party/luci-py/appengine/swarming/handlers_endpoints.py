# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines Swarming Server endpoints handlers."""

import datetime
import json
import logging
import os

from google.appengine.api import datastore_errors
from google.appengine.api import memcache
from google.appengine.ext import ndb

import endpoints
import gae_ts_mon
from protorpc import messages
from protorpc import message_types
from protorpc import protojson
from protorpc import remote

from components import auth
from components import datastore_utils
from components import endpoints_webapp2
from components import machine_provider
from components import utils

import message_conversion
import swarming_rpcs
from server import acl
from server import bot_code
from server import bot_management
from server import config
from server import pools_config
from server import lease_management
from server import service_accounts
from server import task_pack
from server import task_queues
from server import task_request
from server import task_result
from server import task_scheduler


### Helper Methods


# Used by _get_request_and_result(), clearer than using True/False and important
# as this is part of the security boundary.
_EDIT = object()
_VIEW = object()


# Add support for BooleanField in protorpc in endpoints GET requests.
_old_decode_field = protojson.ProtoJson.decode_field
def _decode_field(self, field, value):
  if (isinstance(field, messages.BooleanField) and
      isinstance(value, basestring)):
    return value.lower() == 'true'
  return _old_decode_field(self, field, value)
protojson.ProtoJson.decode_field = _decode_field


def _to_keys(task_id):
  """Returns request and result keys, handling failure."""
  try:
    return task_pack.get_request_and_result_keys(task_id)
  except ValueError:
    raise endpoints.BadRequestException('%s is an invalid key.' % task_id)


@ndb.tasklet
def _get_task_request_async(task_id, request_key, viewing):
  """Returns the TaskRequest corresponding to a task ID.

  Enforces the ACL for users. Allows bots all access for the moment.

  Returns:
    TaskRequest instance.
  """
  request = yield request_key.get_async()
  if not request:
    raise endpoints.NotFoundException('%s not found.' % task_id)
  if viewing == _VIEW:
    if not acl.can_view_task(request):
      raise endpoints.ForbiddenException('%s is not accessible.' % task_id)
  elif viewing == _EDIT:
    if not acl.can_edit_task(request):
      raise endpoints.ForbiddenException('%s is not accessible.' % task_id)
  else:
    raise endpoints.InternalServerErrorException('_get_task_request_async()')
  raise ndb.Return(request)


def _get_request_and_result(task_id, viewing, trust_memcache):
  """Returns the TaskRequest and task result corresponding to a task ID.

  For the task result, first do an explict lookup of the caches, and then decide
  if it is necessary to fetch from the DB.

  Arguments:
    task_id: task ID as provided by the user.
    viewing: one of _EDIT or _VIEW
    trust_memcache: bool to state if memcache should be trusted for running
        task. If False, when a task is still pending/running, do a DB fetch.

  Returns:
    tuple(TaskRequest, result): result can be either for a TaskRunResult or a
                                TaskResultSummay.
  """
  request_key, result_key = _to_keys(task_id)
  # The task result has a very high odd of taking much more time to fetch than
  # the TaskRequest, albeit it is the TaskRequest that enforces ACL. Do the task
  # result fetch first, the worst that will happen is unnecessarily fetching the
  # task result.
  result_future = result_key.get_async(
      use_cache=True, use_memcache=True, use_datastore=False)

  # The TaskRequest has P(99.9%) chance of being fetched from memcache since it
  # is immutable.
  request_future = _get_task_request_async(task_id, request_key, viewing)

  result = result_future.get_result()
  if (not result or
      (result.state in task_result.State.STATES_RUNNING and not
        trust_memcache)):
    # Either the entity is not in cache, or we don't trust memcache for a
    # running task result. Do the DB fetch, which is slow.
    result = result_key.get(
        use_cache=False, use_memcache=False, use_datastore=True)

  request = request_future.get_result()

  if not result:
    raise endpoints.NotFoundException('%s not found.' % task_id)
  return request, result


def get_or_raise(key):
  """Returns an entity or raises an endpoints exception if it does not exist."""
  result = key.get()
  if not result:
    raise endpoints.NotFoundException('%s not found.' % key.id())
  return result


def apply_server_property_defaults(properties):
  """Fills ndb task properties with default values read from server settings."""
  settings = config.settings()
  # TODO(iannucci): This was an artifact of the existing test harnesses;
  # get_pool_config raises on None, but the way it's mocked in
  # ./test_env_handlers.py allows `get_pool_config` to return None in this case.
  # This try/except will be cleaned up in a subsequent CL, once I remove these
  # default services from `config`.
  try:
    pool_cfg = pools_config.get_pool_config(properties.pool)
  except ValueError:
    pool_cfg = None
  if not settings and not pool_cfg:
    return

  iso_server = settings.isolate.default_server
  iso_ns = settings.isolate.default_namespace
  if pool_cfg and pool_cfg.default_isolate:
    iso_server = pool_cfg.default_isolate.server
    iso_ns = pool_cfg.default_isolate.namespace

  if iso_server and iso_ns:
    properties.inputs_ref = properties.inputs_ref or task_request.FilesRef()
    properties.inputs_ref.isolatedserver = (
        properties.inputs_ref.isolatedserver or iso_server)
    properties.inputs_ref.namespace = (
        properties.inputs_ref.namespace or iso_ns)

  cipd_server = settings.cipd.default_server
  cipd_vers = settings.cipd.default_client_package.version
  if pool_cfg and pool_cfg.default_cipd:
    cipd_server = pool_cfg.default_cipd.server
    cipd_vers = pool_cfg.default_cipd.client_version

  if cipd_server and properties.cipd_input:
    properties.cipd_input.server = (
        properties.cipd_input.server or cipd_server)
    properties.cipd_input.client_package = (
        properties.cipd_input.client_package or task_request.CipdPackage())
    # TODO(iannucci) - finish removing 'client_package' as a task-configurable
    # setting.
    properties.cipd_input.client_package.package_name = (
      'infra/tools/cipd/${platform}')
    properties.cipd_input.client_package.version = (
        properties.cipd_input.client_package.version or cipd_vers)


### API


swarming_api = auth.endpoints_api(
    name='swarming',
    version='v1',
    description=
        'API to interact with the Swarming service. Permits to create, '
        'view and cancel tasks, query tasks and bots')


@swarming_api.api_class(resource_name='server', path='server')
class SwarmingServerService(remote.Service):
  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.ServerDetails,
      http_method='GET')
  @auth.require(acl.can_access)
  def details(self, _request):
    """Returns information about the server."""
    host = 'https://' + os.environ['HTTP_HOST']

    cfg = config.settings()
    isolate, _cipd = pools_config.get_default_external_services()

    default_isolate_server = cfg.isolate.default_server
    default_isolate_namespace = cfg.isolate.default_namespace
    if isolate:
      default_isolate_server = isolate.server
      default_isolate_namespace = isolate.namespace

    mpp = ''
    if cfg.mp and cfg.mp.server:
      mpp = cfg.mp.server
    # as a fallback, try pulling from datastore
    if not mpp:
      mpp = machine_provider.MachineProviderConfiguration.get_instance_url()
    if mpp:
      mpp = mpp + '/leases/%s'

    return swarming_rpcs.ServerDetails(
        bot_version=bot_code.get_bot_version(host)[0],
        server_version=utils.get_app_version(),
        machine_provider_template=mpp,
        display_server_url_template=cfg.display_server_url_template,
        luci_config=config.config.config_service_hostname(),
        default_isolate_server=default_isolate_server,
        default_isolate_namespace=default_isolate_namespace)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.BootstrapToken)
  @auth.require(acl.can_create_bot)
  def token(self, _request):
    """Returns a token to bootstrap a new bot.

    This may seem strange to be a POST and not a GET, but it's very
    important to make sure GET requests are idempotent and safe
    to be pre-fetched; generating a token is neither of those things.
    """
    return swarming_rpcs.BootstrapToken(
        bootstrap_token = bot_code.generate_bootstrap_token(),
      )

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.ClientPermissions,
      http_method='GET')
  @auth.public
  def permissions(self, _request):
    """Returns the caller's permissions."""
    return swarming_rpcs.ClientPermissions(
        delete_bot=acl.can_delete_bot(),
        terminate_bot=acl.can_edit_bot(),
        get_configs=acl.can_view_config(),
        put_configs=acl.can_edit_config(),
        cancel_task=acl._is_user() or acl.is_ip_whitelisted_machine(),
        cancel_tasks=acl.can_edit_all_tasks(),
        get_bootstrap_token=acl.can_create_bot())

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.FileContent,
      http_method='GET')
  @auth.require(acl.can_view_config)
  def get_bootstrap(self, _request):
    """Retrieves the current version of bootstrap.py."""
    obj = bot_code.get_bootstrap('', '')
    return swarming_rpcs.FileContent(
        content=obj.content.decode('utf-8'),
        who=obj.who,
        when=obj.when,
        version=obj.version)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.FileContent,
      http_method='GET')
  @auth.require(acl.can_view_config)
  def get_bot_config(self, _request):
    """Retrieves the current version of bot_config.py."""
    obj = bot_code.get_bot_config()
    return swarming_rpcs.FileContent(
        content=obj.content.decode('utf-8'),
        who=obj.who,
        when=obj.when,
        version=obj.version)


TaskId = endpoints.ResourceContainer(
    message_types.VoidMessage,
    task_id=messages.StringField(1, required=True))


TaskIdWithPerf = endpoints.ResourceContainer(
    message_types.VoidMessage,
    task_id=messages.StringField(1, required=True),
    include_performance_stats=messages.BooleanField(2, default=False))


TaskCancel = endpoints.ResourceContainer(
    swarming_rpcs.TaskCancelRequest,
    task_id=messages.StringField(1, required=True))


@swarming_api.api_class(resource_name='task', path='task')
class SwarmingTaskService(remote.Service):
  """Swarming's task-related API."""
  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TaskIdWithPerf, swarming_rpcs.TaskResult,
      name='result',
      path='{task_id}/result',
      http_method='GET')
  @auth.require(acl.can_access)
  def result(self, request):
    """Reports the result of the task corresponding to a task ID.

    It can be a 'run' ID specifying a specific retry or a 'summary' ID hidding
    the fact that a task may have been retried transparently, when a bot reports
    BOT_DIED.

    A summary ID ends with '0', a run ID ends with '1' or '2'.
    """
    logging.debug('%s', request)
    # Workaround a bug in ndb where if a memcache set fails, a stale copy can be
    # kept in memcache indefinitely. In the case of task result, if this happens
    # on the very last store where the task is saved to NDB to be marked as
    # completed, and that the DB store succeeds *but* the memcache update fails,
    # this API will *always* return the stale version.
    try:
      _, result = _get_request_and_result(request.task_id, _VIEW, False)
    except ValueError:
      raise endpoints.BadRequestException('Invalid task ID')
    return message_conversion.task_result_to_rpc(
        result, request.include_performance_stats)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TaskId, swarming_rpcs.TaskRequest,
      name='request',
      path='{task_id}/request',
      http_method='GET')
  @auth.require(acl.can_access)
  def request(self, request):
    """Returns the task request corresponding to a task ID."""
    logging.debug('%s', request)
    request_key, _ = _to_keys(request.task_id)
    request_obj = _get_task_request_async(
        request.task_id, request_key, _VIEW).get_result()
    return message_conversion.task_request_to_rpc(request_obj)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TaskCancel, swarming_rpcs.CancelResponse,
      name='cancel',
      path='{task_id}/cancel')
  @auth.require(acl.can_access)
  def cancel(self, request):
    """Cancels a task.

    If a bot was running the task, the bot will forcibly cancel the task.
    """
    logging.debug('%s', request)
    request_key, result_key = _to_keys(request.task_id)
    request_obj = _get_task_request_async(
        request.task_id, request_key, _EDIT).get_result()
    ok, was_running = task_scheduler.cancel_task(
        request_obj, result_key, request.kill_running or False, None)
    return swarming_rpcs.CancelResponse(ok=ok, was_running=was_running)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TaskId, swarming_rpcs.TaskOutput,
      name='stdout',
      path='{task_id}/stdout',
      http_method='GET')
  @auth.require(acl.can_access)
  def stdout(self, request):
    """Returns the output of the task corresponding to a task ID."""
    # TODO(maruel): Add streaming. Real streaming is not supported by AppEngine
    # v1.
    # TODO(maruel): Send as raw content instead of encoded. This is not
    # supported by cloud endpoints.
    logging.debug('%s', request)
    # The result must be fetched to know the right run_result_key to use.
    _, result = _get_request_and_result(request.task_id, _VIEW, True)
    output = result.get_output()
    if output:
      output = output.decode('utf-8', 'replace')
    return swarming_rpcs.TaskOutput(output=output)


TasksRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    limit=messages.IntegerField(1, default=200),
    cursor=messages.StringField(2),
    # These should be DateTimeField but endpoints + protorpc have trouble
    # encoding this message in a GET request, this is due to DateTimeField's
    # special encoding in protorpc-1.0/protorpc/message_types.py that is
    # bypassed when using endpoints-1.0/endpoints/protojson.py to add GET query
    # parameter support.
    end=messages.FloatField(3),
    start=messages.FloatField(4),
    state=messages.EnumField(swarming_rpcs.TaskStateQuery, 5, default='ALL'),
    tags=messages.StringField(6, repeated=True),
    sort=messages.EnumField(swarming_rpcs.TaskSort, 7, default='CREATED_TS'),
    include_performance_stats=messages.BooleanField(8, default=False))


TaskStatesRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    task_id=messages.StringField(1, repeated=True))


TasksCountRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    end=messages.FloatField(3),
    start=messages.FloatField(4),
    state=messages.EnumField(swarming_rpcs.TaskStateQuery, 5, default='ALL'),
    tags=messages.StringField(6, repeated=True))


@swarming_api.api_class(resource_name='tasks', path='tasks')
class SwarmingTasksService(remote.Service):
  """Swarming's tasks-related API."""
  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      swarming_rpcs.NewTaskRequest, swarming_rpcs.TaskRequestMetadata)
  @auth.require(acl.can_create_task, 'User cannot create tasks.')
  def new(self, request):
    """Creates a new task.

    The task will be enqueued in the tasks list and will be executed at the
    earliest opportunity by a bot that has at least the dimensions as described
    in the task request.
    """
    sb = (request.properties.secret_bytes
          if request.properties is not None else None)
    if sb is not None:
      request.properties.secret_bytes = "HIDDEN"
    logging.debug('%s', request)
    if sb is not None:
      request.properties.secret_bytes = sb

    try:
      request_obj, secret_bytes, template_apply = (
          message_conversion.new_task_request_from_rpc(
              request, utils.utcnow()))
      for index in xrange(request_obj.num_task_slices):
        apply_server_property_defaults(request_obj.task_slice(index).properties)
      task_request.init_new_request(
          request_obj, acl.can_schedule_high_priority_tasks(),
          template_apply)
      # We need to call the ndb.Model pre-put check earlier because the
      # following checks assume that the request itself is valid and could crash
      # otherwise.
      request_obj._pre_put_hook()
    except (datastore_errors.BadValueError, TypeError, ValueError) as e:
      logging.warning('Incorrect new task request', exc_info=True)
      raise endpoints.BadRequestException(e.message)

    # Make sure the caller is actually allowed to schedule the task before
    # asking the token server for a service account token.
    task_scheduler.check_schedule_request_acl(request_obj)

    # If request_obj.service_account is an email, contact the token server to
    # generate "OAuth token grant" (or grab a cached one). By doing this we
    # check that the given service account usage is allowed by the token server
    # rules at the time the task is posted. This check is also performed later
    # (when running the task), when we get the actual OAuth access token.
    if service_accounts.is_service_account(request_obj.service_account):
      if not service_accounts.has_token_server():
        raise endpoints.BadRequestException(
            'This Swarming server doesn\'t support task service accounts '
            'because Token Server URL is not configured')
      max_lifetime_secs = request_obj.max_lifetime_secs
      try:
        duration = datetime.timedelta(seconds=max_lifetime_secs)
        request_obj.service_account_token = (
            service_accounts.get_oauth_token_grant(
                service_account=request_obj.service_account,
                validity_duration=duration))
      except service_accounts.PermissionError as exc:
        raise auth.AuthorizationError(exc.message)
      except service_accounts.MisconfigurationError as exc:
        raise endpoints.BadRequestException(exc.message)
      except service_accounts.InternalError as exc:
        raise endpoints.InternalServerErrorException(exc.message)

    # If the user only wanted to evaluate scheduling the task, but not actually
    # schedule it, return early without a task_id.
    if request.evaluate_only:
      request_obj._pre_put_hook()
      return swarming_rpcs.TaskRequestMetadata(
          request=message_conversion.task_request_to_rpc(request_obj))

    try:
      result_summary = task_scheduler.schedule_request(
          request_obj, secret_bytes)
    except (datastore_errors.BadValueError, TypeError, ValueError) as e:
      raise endpoints.BadRequestException(e.message)

    returned_result = None
    if result_summary.state != task_result.State.PENDING:
      # It likely is COMPLETED (because of deduping) or NO_RESOURCE.
      returned_result = message_conversion.task_result_to_rpc(
          result_summary, False)

    return swarming_rpcs.TaskRequestMetadata(
        request=message_conversion.task_request_to_rpc(request_obj),
        task_id=task_pack.pack_result_summary_key(result_summary.key),
        task_result=returned_result)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TasksRequest, swarming_rpcs.TaskList,
      http_method='GET')
  @auth.require(acl.can_view_all_tasks)
  def list(self, request):
    """Returns full task results based on the filters.

    This endpoint is significantly slower than 'count'. Use 'count' when
    possible. If you just want the state of tasks, use 'get_states'.
    """
    # TODO(maruel): Rename 'list' to 'results'.
    # TODO(maruel): Rename 'TaskList' to 'TaskResults'.
    logging.debug('%s', request)
    now = utils.utcnow()
    try:
      items, cursor = datastore_utils.fetch_page(
          self._query_from_request(request), request.limit, request.cursor)
    except ValueError as e:
      raise endpoints.BadRequestException(
          'Inappropriate filter for tasks/list: %s' % e)
    except datastore_errors.NeedIndexError as e:
      logging.error('%s', e)
      raise endpoints.BadRequestException(
          'Requires new index, ask admin to create one.')
    except datastore_errors.BadArgumentError as e:
      logging.error('%s', e)
      raise endpoints.BadRequestException(
          'This combination is unsupported, sorry.')
    return swarming_rpcs.TaskList(
        cursor=cursor,
        items=[
          message_conversion.task_result_to_rpc(
              i, request.include_performance_stats)
          for i in items
        ],
        now=now)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TaskStatesRequest, swarming_rpcs.TaskStates,
      http_method='GET')
  # TODO(martiniss): users should be able to view their state. This requires
  # looking up each TaskRequest.
  @auth.require(acl.can_view_all_tasks)
  def get_states(self, request):
    """Returns task state for a specific set of tasks.
    """
    logging.debug('%s', request)
    result_keys = [_to_keys(task_id)[1] for task_id in request.task_id]

    # Hot path. Fetch everything we can from memcache.
    entities = ndb.get_multi(
        result_keys, use_cache=True, use_memcache=True, use_datastore=False)
    states = [t.state if t else task_result.State.PENDING for t in entities]
    # Now fetch both the ones in non-stable state or not in memcache.
    missing_keys = [
      result_keys[i] for i, state in enumerate(states)
      if state in task_result.State.STATES_RUNNING
    ]
    if missing_keys:
      more = ndb.get_multi(
          missing_keys, use_cache=False, use_memcache=False, use_datastore=True)
      # This relies on missing_keys being in the same order as states (for
      # common elements).
      for i, s in enumerate(states):
        if s in task_result.State.STATES_RUNNING:
          states[i] = more.pop(0).state

    return swarming_rpcs.TaskStates(
        states=[swarming_rpcs.TaskState(state) for state in states])

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TasksRequest, swarming_rpcs.TaskRequests,
      http_method='GET')
  @auth.require(acl.can_view_all_tasks)
  def requests(self, request):
    """Returns tasks requests based on the filters.

    This endpoint is slightly slower than 'list'. Use 'list' or 'count' when
    possible.
    """
    logging.debug('%s', request)
    if request.include_performance_stats:
      raise endpoints.BadRequestException(
          'Can\'t set include_performance_stats for tasks/list')
    now = utils.utcnow()
    try:
      # Get the TaskResultSummary keys, then fetch the corresponding
      # TaskRequest entities.
      keys, cursor = datastore_utils.fetch_page(
          self._query_from_request(request),
          request.limit, request.cursor, keys_only=True)
      items = ndb.get_multi(
          task_pack.result_summary_key_to_request_key(k) for k in keys)
    except ValueError as e:
      raise endpoints.BadRequestException(
          'Inappropriate filter for tasks/requests: %s' % e)
    except datastore_errors.NeedIndexError as e:
      logging.error('%s', e)
      raise endpoints.BadRequestException(
          'Requires new index, ask admin to create one.')
    except datastore_errors.BadArgumentError as e:
      logging.error('%s', e)
      raise endpoints.BadRequestException(
          'This combination is unsupported, sorry.')
    return swarming_rpcs.TaskRequests(
        cursor=cursor,
        items=[message_conversion.task_request_to_rpc(i) for i in items],
        now=now)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      swarming_rpcs.TasksCancelRequest, swarming_rpcs.TasksCancelResponse,
      http_method='POST')
  @auth.require(acl.can_edit_all_tasks)
  def cancel(self, request):
    """Cancel a subset of pending tasks based on the tags.

    Cancellation happens asynchronously, so when this call returns,
    cancellations will not have completed yet.
    """
    logging.debug('%s', request)
    if not request.tags:
      # Prevent accidental cancellation of everything.
      raise endpoints.BadRequestException(
          'You must specify tags when cancelling multiple tasks.')

    now = utils.utcnow()
    cond = task_result.TaskResultSummary.state == task_result.State.PENDING
    if request.kill_running:
      cond = ndb.OR(
          cond,
          task_result.TaskResultSummary.state == task_result.State.RUNNING)
    q = task_result.TaskResultSummary.query(cond).order(
        task_result.TaskResultSummary.key)
    for tag in request.tags:
      q = q.filter(task_result.TaskResultSummary.tags == tag)

    tasks, cursor = datastore_utils.fetch_page(q, request.limit, request.cursor)

    if tasks:
      payload = json.dumps(
          {
            'tasks': [t.task_id for t in tasks],
            'kill_running': request.kill_running or False,
          })
      ok = utils.enqueue_task(
          '/internal/taskqueue/important/tasks/cancel', 'cancel-tasks',
          payload=payload)
      if not ok:
        raise endpoints.InternalServerErrorException(
            'Could not enqueue cancel request, try again later')
    else:
      logging.info('No tasks to cancel.')

    return swarming_rpcs.TasksCancelResponse(
        cursor=cursor,
        matched=len(tasks),
        now=now)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TasksCountRequest, swarming_rpcs.TasksCount,
      http_method='GET')
  @auth.require(acl.can_view_all_tasks)
  def count(self, request):
    """Counts number of tasks in a given state."""
    logging.debug('%s', request)
    if not request.start:
      raise endpoints.BadRequestException('start (as epoch) is required')
    now = utils.utcnow()
    mem_key = self._memcache_key(request, now)
    count = memcache.get(mem_key, namespace='tasks_count')
    if count is not None:
      return swarming_rpcs.TasksCount(count=count, now=now)

    try:
      count = self._query_from_request(request, 'created_ts').count()
      memcache.add(mem_key, count, 24*60*60, namespace='tasks_count')
    except ValueError as e:
      raise endpoints.BadRequestException(
          'Inappropriate filter for tasks/count: %s' % e)
    return swarming_rpcs.TasksCount(count=count, now=now)

  def _memcache_key(self, request, now):
    # Floor now to minute to account for empty "end"
    end = request.end or now.replace(second=0, microsecond=0)
    request.tags.sort()
    return '%s|%s|%s|%s' % (request.tags, request.state, request.start, end)

  def _query_from_request(self, request, sort=None):
    """Returns a TaskResultSummary query."""
    start = message_conversion.epoch_to_datetime(request.start)
    end = message_conversion.epoch_to_datetime(request.end)
    return task_result.get_result_summaries_query(
        start, end,
        sort or request.sort.name.lower(),
        request.state.name.lower(),
        request.tags)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.TasksTags,
      http_method='GET')
  @auth.require(acl.can_view_all_tasks)
  def tags(self, _request):
    """Returns the cached set of tags currently seen in the fleet."""
    tags = task_result.TagAggregation.KEY.get()
    ft = [
      swarming_rpcs.StringListPair(key=t.tag, value=t.values)
      for t in tags.tags
    ]
    return swarming_rpcs.TasksTags(tasks_tags=ft, ts=tags.ts)


TaskQueuesRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    # Note that it's possible that the RPC returns a tad more or less items than
    # requested limit.
    limit=messages.IntegerField(1, default=200),
    cursor=messages.StringField(2))


@swarming_api.api_class(resource_name='queues', path='queues')
class SwarmingQueuesService(remote.Service):
  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      TaskQueuesRequest, swarming_rpcs.TaskQueueList,
      http_method='GET')
  @auth.require(acl.can_view_all_tasks)
  def list(self, request):
    logging.debug('%s', request)
    now = utils.utcnow()
    # Disable the in-process local cache. This is important, as there can be up
    # to a thousand entities loaded in memory, and this is a pure memory leak,
    # as there's no chance this specific instance will need these again,
    # therefore this leads to 'Exceeded soft memory limit' AppEngine errors.
    q = task_queues.TaskDimensions.query(
        default_options=ndb.QueryOptions(use_cache=False))
    cursor = request.cursor
    out = []
    count = 0
    # As there can be a lot of terminate tasks, try to loop a few times (max 5)
    # to get more items.
    while len(out) < request.limit and (cursor or not count) and count < 5:
      items, cursor = datastore_utils.fetch_page(
          q, request.limit - len(out), cursor)
      for i in items:
        for s in i.sets:
          # Ignore the tasks that are only id specific, since they are
          # termination tasks. This happens in particular with Machine Provider
          # managed bots, since they are recycled on a continuous basis. There
          # may be one per bot, and it is not really useful for the user, the
          # user may just query the list of bots.
          if (len(s.dimensions_flat) == 1 and
              s.dimensions_flat[0].startswith('id:')):
            # A terminate task.
            continue
          out.append(swarming_rpcs.TaskQueue(
              dimensions=s.dimensions_flat, valid_until_ts=s.valid_until_ts))
      count += 1
    return swarming_rpcs.TaskQueueList(cursor=cursor, items=out, now=now)


BotId = endpoints.ResourceContainer(
    message_types.VoidMessage,
    bot_id=messages.StringField(1, required=True))


BotEventsRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    bot_id=messages.StringField(1, required=True),
    limit=messages.IntegerField(2, default=200),
    cursor=messages.StringField(3),
    # end, start are seconds since epoch.
    end=messages.FloatField(4),
    start=messages.FloatField(5))


BotTasksRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    bot_id=messages.StringField(1, required=True),
    limit=messages.IntegerField(2, default=200),
    cursor=messages.StringField(3),
    # end, start are seconds since epoch.
    end=messages.FloatField(4),
    start=messages.FloatField(5),
    state=messages.EnumField(swarming_rpcs.TaskStateQuery, 6, default='ALL'),
    sort=messages.EnumField(swarming_rpcs.TaskSort, 7, default='CREATED_TS'),
    include_performance_stats=messages.BooleanField(8, default=False))


@swarming_api.api_class(resource_name='bot', path='bot')
class SwarmingBotService(remote.Service):
  """Bot-related API. Permits querying information about the bot's properties"""
  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotId, swarming_rpcs.BotInfo,
      name='get',
      path='{bot_id}/get',
      http_method='GET')
  @auth.require(acl.can_view_bot)
  def get(self, request):
    """Returns information about a known bot.

    This includes its state and dimensions, and if it is currently running a
    task.
    """
    logging.debug('%s', request)
    bot_id = request.bot_id
    bot = bot_management.get_info_key(bot_id).get()
    deleted = False
    if not bot:
      # If there is not BotInfo, look if there are BotEvent child of this
      # entity. If this is the case, it means the bot was deleted but it's
      # useful to show information about it to the user even if the bot was
      # deleted. For example, it could be an machine-provider bot.
      events = bot_management.get_events_query(bot_id, True).fetch(1)
      if not events:
        raise endpoints.NotFoundException('%s not found.' % bot_id)
      bot = bot_management.BotInfo(
          key=bot_management.get_info_key(bot_id),
          dimensions_flat=task_queues.dimensions_to_flat(
              events[0].dimensions),
          state=events[0].state,
          external_ip=events[0].external_ip,
          authenticated_as=events[0].authenticated_as,
          version=events[0].version,
          quarantined=events[0].quarantined,
          maintenance_msg=events[0].maintenance_msg,
          task_id=events[0].task_id,
          last_seen_ts=events[0].ts,
          lease_id=events[0].lease_id,
          lease_expiration_ts=events[0].lease_expiration_ts,
          leased_indefinitely=events[0].leased_indefinitely,
          machine_type=events[0].machine_type,
          machine_lease=events[0].machine_lease)
      deleted = True

    return message_conversion.bot_info_to_rpc(bot, deleted=deleted)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotId, swarming_rpcs.DeletedResponse,
      name='delete',
      path='{bot_id}/delete')
  @auth.require(acl.can_delete_bot)
  def delete(self, request):
    """Deletes the bot corresponding to a provided bot_id.

    At that point, the bot will not appears in the list of bots but it is still
    possible to get information about the bot with its bot id is known, as
    historical data is not deleted.

    It is meant to remove from the DB the presence of a bot that was retired,
    e.g. the VM was shut down already. Use 'terminate' instead of the bot is
    still alive.
    """
    logging.debug('%s', request)
    bot_info_key = bot_management.get_info_key(request.bot_id)
    bot_info = get_or_raise(bot_info_key)  # raises 404 if there is no such bot
    if bot_info.machine_lease:
      ml = lease_management.MachineLease.get_by_id(bot_info.machine_lease)
      # MachineLease entities persist, ensure it's still referring to this bot.
      if ml and ml.hostname == bot_info.id:
        if lease_management.release(ml):
          # This calls task_queues.cleanup_after_bot and bot_info_key.delete.
          lease_management.cleanup_bot(ml)
          return swarming_rpcs.DeletedResponse(deleted=True)
        # MP bot failed to be released. Don't delete or clean up after it.
        return swarming_rpcs.DeletedResponse(deleted=False)
      else:
        # MachineLease moved on to the next bot.
        # Not an error. See server/lease_management.py.
        logging.info(
            'MachineLease doesn\'t refer to this BotInfo, skipping release')
    # Non-MP bot. Attempt to delete it normally.
    # BotRoot is parent to BotInfo. It is important to note that the bot is
    # not there anymore, so it is not a member of any task queue.
    task_queues.cleanup_after_bot(bot_info_key.parent())
    bot_info_key.delete()
    return swarming_rpcs.DeletedResponse(deleted=True)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotEventsRequest, swarming_rpcs.BotEvents,
      name='events',
      path='{bot_id}/events',
      http_method='GET')
  @auth.require(acl.can_view_bot)
  def events(self, request):
    """Returns events that happened on a bot."""
    logging.debug('%s', request)
    try:
      now = utils.utcnow()
      start = message_conversion.epoch_to_datetime(request.start)
      end = message_conversion.epoch_to_datetime(request.end)
      order = not (start or end)
      q = bot_management.get_events_query(request.bot_id, order)
      if not order:
        q = q.order(-bot_management.BotEvent.ts, bot_management.BotEvent.key)
      if start:
        q = q.filter(bot_management.BotEvent.ts >= start)
      if end:
        q = q.filter(bot_management.BotEvent.ts < end)
      items, cursor = datastore_utils.fetch_page(
          q, request.limit, request.cursor)
    except ValueError as e:
      raise endpoints.BadRequestException(
          'Inappropriate filter for bot.events: %s' % e)
    return swarming_rpcs.BotEvents(
        cursor=cursor,
        items=[message_conversion.bot_event_to_rpc(r) for r in items],
        now=now)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotId, swarming_rpcs.TerminateResponse,
      name='terminate',
      path='{bot_id}/terminate')
  @auth.require(acl.can_edit_bot)
  def terminate(self, request):
    """Asks a bot to terminate itself gracefully.

    The bot will stay in the DB, use 'delete' to remove it from the DB
    afterward. This request returns a pseudo-taskid that can be waited for to
    wait for the bot to turn down.

    This command is particularly useful when a privileged user needs to safely
    debug a machine specific issue. The user can trigger a terminate for one of
    the bot exhibiting the issue, wait for the pseudo-task to run then access
    the machine with the guarantee that the bot is not running anymore.
    """
    # TODO(maruel): Disallow a terminate task when there's one currently
    # pending or if the bot is considered 'dead', e.g. no contact since 10
    # minutes.
    logging.debug('%s', request)
    bot_id = unicode(request.bot_id)
    bot_key = bot_management.get_info_key(bot_id)
    get_or_raise(bot_key)  # raises 404 if there is no such bot
    try:
      # Craft a special priority 0 task to tell the bot to shutdown.
      request = task_request.create_termination_task(
          bot_id, wait_for_capacity=True)
    except (datastore_errors.BadValueError, TypeError, ValueError) as e:
      raise endpoints.BadRequestException(e.message)

    result_summary = task_scheduler.schedule_request(
        request, secret_bytes=None)
    return swarming_rpcs.TerminateResponse(
        task_id=task_pack.pack_result_summary_key(result_summary.key))

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotTasksRequest, swarming_rpcs.BotTasks,
      name='tasks',
      path='{bot_id}/tasks',
      http_method='GET')
  @auth.require(acl.can_view_all_tasks)
  def tasks(self, request):
    """Lists a given bot's tasks within the specified date range.

    In this case, the tasks are effectively TaskRunResult since it's individual
    task tries sent to this specific bot.

    It is impossible to search by both tags and bot id. If there's a need,
    TaskRunResult.tags will be added (via a copy from TaskRequest.tags).
    """
    logging.debug('%s', request)
    try:
      start = message_conversion.epoch_to_datetime(request.start)
      end = message_conversion.epoch_to_datetime(request.end)
      now = utils.utcnow()
      q = task_result.get_run_results_query(
          start, end,
          request.sort.name.lower(),
          request.state.name.lower(),
          request.bot_id)
      items, cursor = datastore_utils.fetch_page(
          q, request.limit, request.cursor)
    except ValueError as e:
      raise endpoints.BadRequestException(
          'Inappropriate filter for bot.tasks: %s' % e)
    return swarming_rpcs.BotTasks(
        cursor=cursor,
        items=[
          message_conversion.task_result_to_rpc(
              r, request.include_performance_stats)
          for r in items
        ],
        now=now)


BotsRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    limit=messages.IntegerField(1, default=200),
    cursor=messages.StringField(2),
    # Must be a list of 'key:value' strings to filter the returned list of bots
    # on.
    dimensions=messages.StringField(3, repeated=True),
    quarantined=messages.EnumField(
        swarming_rpcs.ThreeStateBool, 4, default='NONE'),
    in_maintenance=messages.EnumField(
        swarming_rpcs.ThreeStateBool, 8, default='NONE'),
    is_dead=messages.EnumField(swarming_rpcs.ThreeStateBool, 5, default='NONE'),
    is_busy=messages.EnumField(swarming_rpcs.ThreeStateBool, 6, default='NONE'),
    is_mp=messages.EnumField(swarming_rpcs.ThreeStateBool, 7, default='NONE'))


BotsCountRequest = endpoints.ResourceContainer(
    message_types.VoidMessage,
    dimensions=messages.StringField(1, repeated=True))


@swarming_api.api_class(resource_name='bots', path='bots')
class SwarmingBotsService(remote.Service):
  """Bots-related API."""

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotsRequest, swarming_rpcs.BotList,
      http_method='GET')
  @auth.require(acl.can_view_bot)
  def list(self, request):
    """Provides list of known bots.

    Deleted bots will not be listed.
    """
    logging.debug('%s', request)
    now = utils.utcnow()
    # Disable the in-process local cache. This is important, as there can be up
    # to a thousand entities loaded in memory, and this is a pure memory leak,
    # as there's no chance this specific instance will need these again,
    # therefore this leads to 'Exceeded soft memory limit' AppEngine errors.
    q = bot_management.BotInfo.query(
        default_options=ndb.QueryOptions(use_cache=False))
    try:
      q = bot_management.filter_dimensions(q, request.dimensions)
      q = bot_management.filter_availability(
          q, swarming_rpcs.to_bool(request.quarantined),
          swarming_rpcs.to_bool(request.in_maintenance),
          swarming_rpcs.to_bool(request.is_dead),
          swarming_rpcs.to_bool(request.is_busy),
          swarming_rpcs.to_bool(request.is_mp))
    except ValueError as e:
      raise endpoints.BadRequestException(str(e))

    bots, cursor = datastore_utils.fetch_page(q, request.limit, request.cursor)
    return swarming_rpcs.BotList(
        cursor=cursor,
        death_timeout=config.settings().bot_death_timeout_secs,
        items=[message_conversion.bot_info_to_rpc(bot) for bot in bots],
        now=now)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      BotsCountRequest, swarming_rpcs.BotsCount,
      http_method='GET')
  @auth.require(acl.can_view_bot)
  def count(self, request):
    """Counts number of bots with given dimensions."""
    logging.debug('%s', request)
    now = utils.utcnow()
    q = bot_management.BotInfo.query()
    try:
      q = bot_management.filter_dimensions(q, request.dimensions)
    except ValueError as e:
      raise endpoints.BadRequestException(str(e))

    f_count = q.count_async()
    f_dead = bot_management.filter_availability(
        q, None, None, True, None, None).count_async()
    f_quarantined = bot_management.filter_availability(
        q, True, None, None, None, None).count_async()
    f_maintenance = bot_management.filter_availability(
        q, None, True, None, None, None).count_async()
    f_busy = bot_management.filter_availability(
        q, None, None, None, True, None).count_async()
    return swarming_rpcs.BotsCount(
        count=f_count.get_result(),
        quarantined=f_quarantined.get_result(),
        maintenance=f_maintenance.get_result(),
        dead=f_dead.get_result(),
        busy=f_busy.get_result(),
        now=now)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      message_types.VoidMessage, swarming_rpcs.BotsDimensions,
      http_method='GET')
  @auth.require(acl.can_view_bot)
  def dimensions(self, _request):
    """Returns the cached set of dimensions currently in use in the fleet."""
    dims = bot_management.DimensionAggregation.KEY.get()
    fd = [
      swarming_rpcs.StringListPair(key=d.dimension, value=d.values)
      for d in dims.dimensions
    ]
    return swarming_rpcs.BotsDimensions(bots_dimensions=fd, ts=dims.ts)


def get_routes():
  return endpoints_webapp2.api_routes([
    SwarmingServerService,
    SwarmingTaskService,
    SwarmingTasksService,
    SwarmingQueuesService,
    SwarmingBotService,
    SwarmingBotsService,
    # components.config endpoints for validation and configuring of luci-config
    # service URL.
    config.ConfigApi], base_path='/api')
