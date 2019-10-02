# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# This is a reimplementation of RemoteClientNative but it uses gRPC to
# communicate with a server instead of REST.
#
# TODO(aludwin): remove extremely verbose logging throughout when this is
# actually working.

import copy
import json
import logging
import time

from google.protobuf import json_format
from proto_bot import bots_pb2
from proto_bot import bytestream_pb2
from proto_bot import code_pb2
from proto_bot import command_pb2
from proto_bot import tasks_pb2
from remote_client_errors import InternalError
from remote_client_errors import MintOAuthTokenError
from remote_client_errors import PollError
from remote_client_errors import BotCodeError
from utils import net
from utils import grpc_proxy

try:
  from proto_bot import bots_pb2_grpc
  from proto_bot import tasks_pb2_grpc
  from proto_bot import bytestream_pb2_grpc
except ImportError:
  # This happens legitimately during unit tests
  bots_pb2_grpc = None
  tasks_pb2_grpc = None
  bytestream_pb2_grpc = None


class RemoteClientGrpc(object):
  """RemoteClientGrpc knows how to make calls via gRPC.

  Any non-scalar value that is returned that references values from the proto
  messages should be deepcopy'd since protos make use of weak references and
  might be garbage-collected before the values are used.
  """

  def __init__(self, server, fake_proxy=None):
    logging.info('Communicating with host %s via gRPC', server)
    if fake_proxy:
      self._proxy_bots = fake_proxy
      self._proxy_tasks = fake_proxy
      self._proxy_bytestream = fake_proxy
    else:
      if not bots_pb2_grpc:
        raise ImportError('can\'t import stubs - is gRPC installed?')
      self._proxy_bots = grpc_proxy.Proxy(server, bots_pb2_grpc.BotsStub)
      self._proxy_tasks = grpc_proxy.Proxy(server, tasks_pb2_grpc.TasksStub)
      self._proxy_bs = grpc_proxy.Proxy(server,
                                        bytestream_pb2_grpc.ByteStreamStub)
    self._server = server
    self._log_is_asleep = False
    self._session = None
    self._stdout_offset = 0
    self._stdout_resource = None

  @property
  def server(self):
    return self._server

  @property
  def is_grpc(self):
    return True

  def initialize(self, quit_bit=None):
    pass

  @property
  def uses_auth(self):
    return False

  def get_authentication_headers(self):
    return {}

  def ping(self):
    pass

  def do_handshake(self, attributes):
    logging.info('do_handshake(%s)', attributes)
    # Initialize the session
    self._session = bots_pb2.BotSession()
    self._session.status = bots_pb2.OK
    self._attributes_to_session(attributes)

    # Call the server and overwrite our copy of the session
    request = bots_pb2.CreateBotSessionRequest()
    request.parent = self._proxy_bots.prefix
    request.bot_session.CopyFrom(self._session)
    self._session = self._proxy_bots.call_unary('CreateBotSession', request)
    resp = {
        'bot_version': self._session.version,
        'bot_group_cfg': {
            'dimensions': _worker_to_bot_group_cfg(self._session.worker),
        },
        # server_version: unknown, doesn't seem to matter
        # bot_group_cfg_version: only really matters that it's non-None
        'bot_group_cfg_version': 1,
    }

    logging.info('Completed handshake: %s', resp)
    return copy.deepcopy(resp)

  def poll(self, attributes):
    logging.info('poll(%s)', attributes)
    if self._session:
      if len(self._session.leases) == 1:
        # Wouldn't have gotten here unless the prior task finished
        self._session.leases[0].state = bots_pb2.COMPLETED
      elif self._session.leases:
        # Not a good assumption but works for now. When this fails, need to do
        # more realistic state management.
        logging.error('multiple leases are not supported yet')
    else:
      # This shouldn't happen since the session was created in do_handshake, but
      # if it does, we want the poll to continue so we can download a new
      # version of the bot code.
      logging.error('session is unset; creating a blank one')
      self._session = bots_pb2.BotSession()

    # Update and send the session
    self._attributes_to_session(attributes)
    req = bots_pb2.UpdateBotSessionRequest()
    req.bot_session.CopyFrom(self._session)
    req.name = req.bot_session.name
    try:
      # TODO(aludwin): pass in quit_bit? Pressing ctrl-c doesn't actually
      # interrupt either the long poll or the retry process, so we need to wait
      # a while to exit cleanly.
      self._session = self._proxy_bots.call_unary('UpdateBotSession', req)
    except grpc_proxy.grpc.RpcError as e:
      # It's fine for the deadline to expire; this simply means that no tasks
      # were received while long-polling. Any other exception cannot be
      # recovered here.
      if e.code() is not grpc_proxy.grpc.StatusCode.DEADLINE_EXCEEDED:
        raise PollError(str(e))
    except Exception as e:
      raise PollError(str(e))

    # Check for new leases
    new_lease = None
    for lease in self._session.leases:
      if lease.state == bots_pb2.PENDING:
        if new_lease:
          raise PollError('Multiple new leases are not supported')
        new_lease = lease
        # This will be picked up the next time we call UpdateBotSession (in
        # post_bot_event).
        new_lease.state = bots_pb2.ACTIVE
      elif lease.state == bots_pb2.ACTIVE:
        raise PollError('Server thinks lease is already running')
      elif lease.state == bots_pb2.COMPLETED:
        pass
      elif lease.state == bots_pb2.CANCELLED:
        pass

    if not new_lease:
      # Nothing to do at the moment. Sleep briefly and then call again because
      # the server supports long-polls. If the server is overwhelmed, it should
      # reply with UNAVAILABLE.
      # TODO(aludwin): handle UNAVAILABLE.
      if not self._log_is_asleep:
        logging.info('Going to sleep')
        self._log_is_asleep = True
      return 'sleep', 1

    self._log_is_asleep = False
    if not new_lease.inline_assignment:
      raise PollError('task must be included in lease')

    return self._process_lease(new_lease)

  def post_bot_event(self, event_type, message, _attributes):
    """Logs bot-specific info to the server"""
    logging.info('post_bot_event(%s, %s)', event_type, message)

    req_type, status = {
        'bot_error': (bots_pb2.PostBotEventTempRequest.ERROR, None),
        'bot_rebooting': (bots_pb2.PostBotEventTempRequest.INFO,
                          bots_pb2.HOST_REBOOTING),
        'bot_shutdown': (bots_pb2.PostBotEventTempRequest.INFO,
                         bots_pb2.BOT_TERMINATING),
    }.get(event_type, (None, None))

    if req_type is None:
      logging.error('Unsupported event type %s: %s', event_type, message)
      return

    if self._session is None:
      logging.error('post_bot_event called before do_handshake: %s %s',
                    event_type, message)
      return

    # The proxy already knows when a bot is shutting down or rebooting, so the
    # proxy API doesn't allow the bot to send state transition messages - it
    # generates the state transition messages itself."
    if status is not None:
      self._session.status = status

    req = bots_pb2.PostBotEventTempRequest()
    req.name = self._session.name
    req.bot_session_temp.CopyFrom(self._session)
    req.type = req_type
    req.msg = message
    try:
      self._proxy_bots.call_unary('PostBotEventTemp', req)
    except grpc_proxy.grpc.RpcError as e:
      logging.error('gRPC error posting bot event: %s', e)

  def post_task_update(self, task_id, bot_id, params,
                       stdout_and_chunk=None, exit_code=None):
    # In gRPC mode, task_id is a full resource name, as returned in
    # lease.assignment_name.
    #
    # Note that exit_code may be "0" which is falsey, so specifically check for
    # None.
    finished = (exit_code is not None)

    # If there is stdout, send an update.
    if stdout_and_chunk or finished:
      self._send_stdout_chunk(bot_id, task_id, stdout_and_chunk, finished)

    # If this is the last update, call UpdateTaskResult.
    if finished:
      self._complete_task(task_id, bot_id, params, exit_code)

    # TODO(aludwin): send normal BotSessionUpdate and verify that this lease is
    # not CANCELLED.
    should_continue = True
    return should_continue

  def post_task_error(self, task_id, bot_id, message):
    # Send final stdout chunk before aborting the task. This is to satisfy the
    # Remote Worker API semantics.
    self._send_stdout_chunk(bot_id, task_id, None, True)

    req = tasks_pb2.UpdateTaskResultRequest()
    req.name = task_id + '/result'
    req.source = bot_id
    req.result.name = req.name
    req.result.complete = True
    req.result.status.code = code_pb2.ABORTED
    req.result.status.message = message
    self._proxy_tasks.call_unary('UpdateTaskResult', req)

  def get_bot_code(self, new_zip_fn, bot_version, _bot_id):
    with open(new_zip_fn, 'wb') as zf:
      req = bytestream_pb2.ReadRequest()
      req.resource_name = bot_version
      try:
        for resp in self._proxy_bs.call_no_retries('Read', req):
          zf.write(resp.data)
      except grpc_proxy.grpc.RpcError as e:
        logging.error('gRPC error fetching %s: %s', req.resource_name, e)
        raise BotCodeError(new_zip_fn, req.resource_name, bot_version)

  def mint_oauth_token(self, task_id, bot_id, account_id, scopes):
    # pylint: disable=unused-argument
    raise MintOAuthTokenError(
        'mint_oauth_token is not supported in grpc protocol')

  def _attributes_to_session(self, attributes):
    """Creates a proto Worker message from bot attributes."""
    self._session.bot_id = attributes['dimensions']['id'][0]
    _dimensions_to_workers(attributes['dimensions'], self._session.worker)
    self._session.version = attributes['version']

  def _process_lease(self, lease):
    """Process the lease and return its command and payload."""
    pf = 'type.googleapis.com/google.devtools.remoteworkers.v1test2.'
    t = lease.inline_assignment.type_url
    if t == pf + 'AdminTemp':
      return self._process_admin_lease(lease)
    if t == pf + 'Task':
      return self._process_task_lease(lease)
    raise PollError('unknown assignment type %s' % t)

  def _process_task_lease(self, lease):
    """Processes the task lease."""
    task = tasks_pb2.Task()
    lease.inline_assignment.Unpack(task)
    command = command_pb2.CommandTask()
    task.description.Unpack(command)

    # Save the log handle
    self._stdout_offset = 0
    self._stdout_resource = task.logs['stdout']
    expected_stdout = self._stdout_resource_name_from_ids(
        self._session.bot_id, lease.assignment)
    # We don't currently pass _stdout_resource to task_runner.py so verify now
    # that it's what we can reconstruct given the information that we *do* have.
    # TODO(aludwin): pass this information to task_runner.py.
    if self._stdout_resource != expected_stdout:
      raise PollError('expected stdout resouce %s but got %s' % (
          expected_stdout, self._stdout_resource))

    # TODO(aludwin): Pass the namespace through the proxy. Using
    # proxy hardcoded values for now.
    inferred_namespace = 'default-gzip'
    if len(command.inputs.files[0].hash) == 64:
      inferred_namespace = 'sha-256-flat'

    outputs = command.expected_outputs.files[:]
    outputs.extend(command.expected_outputs.directories)

    manifest = {
      'bot_id': self._session.bot_id,
      'dimensions' : {
        # TODO(aludwin): handle standard keys
        prop.key: prop.value for prop in lease.requirements.properties
      },
      'env': {
        env.name: env.value for env in command.inputs.environment_variables
      },
      # proto duration uses long integers; clients expect ints
      'grace_period': int(command.timeouts.shutdown.seconds),
      'hard_timeout': int(command.timeouts.execution.seconds),
      'io_timeout': int(command.timeouts.idle.seconds),
      'isolated': {
        'namespace': inferred_namespace,
        'input' : command.inputs.files[0].hash,
        'server': self._server,
      },
      'outputs': outputs,
      'task_id': lease.assignment,
      # These keys are only needed by raw commands. While this method
      # only supports isolated commands, the keys need to exist to avoid
      # missing key errors.
      'command': None,
      'extra_args': None,
    }
    logging.info('returning manifest: %s', manifest)
    return ('run', manifest)

  def _process_admin_lease(self, lease):
    """Process the admin lease."""
    action = bots_pb2.AdminTemp()
    lease.inline_assignment.Unpack(action)
    cmd = None
    if action.command == bots_pb2.AdminTemp.BOT_UPDATE:
      cmd = 'update'
    elif action.command == bots_pb2.AdminTemp.BOT_RESTART:
      cmd = 'bot_restart'
    elif action.command == bots_pb2.AdminTemp.BOT_TERMINATE:
      cmd = 'terminate'
    elif action.command == bots_pb2.AdminTemp.HOST_RESTART:
      cmd = 'host_reboot'

    if not cmd:
      raise PollError('Unknown command: %s(%s)' % (action.command, action.arg))

    logging.info('Performing admin action: %s(%s)', cmd, action.arg)
    return (cmd, action.arg)

  def _send_stdout_chunk(self, bot_id, task_id, stdout_and_chunk, finished):
    """Sends a stdout chunk to Bytestream.

    If stdout_and_chunk is not None, it must be a two element array, with the
    first element being the data to send and the second being the offset (which
    must match the size of all previously sent chunks). If it is None, then
    `finished` must be true and we will tell the server the log is complete.
    """
    if not stdout_and_chunk and not finished:
      raise InternalError('missing stdout, but finished is False')

    req = bytestream_pb2.WriteRequest()
    req.resource_name = self._stdout_resource_name_from_ids(bot_id, task_id)
    req.write_offset = self._stdout_offset
    req.finish_write = finished
    if stdout_and_chunk:
      req.data = stdout_and_chunk[0]
      req.write_offset = stdout_and_chunk[1]
      self._stdout_offset += len(req.data)

    res = None
    try:
      def stream():
        logging.info('Writing to ByteStream:\n%s', req)
        yield req
      res = self._proxy_bs.call_unary('Write', stream())
    except grpc_proxy.grpc.RpcError as r:
      logging.error('gRPC error during stdout update: %s' % r)
      raise r

    if res is not None and res.committed_size != self._stdout_offset:
      raise InternalError('%s: incorrect size written (got %d, want %d)' % (
          req.resource_name, res.committed_size, self._stdout_offset))

    if finished:
      self._stdout_offset = 0
      self._stdout_resource = None

  def _stdout_resource_name_from_ids(self, bot_id, task_id):
    # TODO(aludwin): use self._stdout_resource, but it's not set in
    # task_runner.py yet. Until then, take the task_id (currently in the form
    # projects/project/tasks/taskid) and extract the taskid from it.
    project_id = self._proxy_bs.prefix
    real_task_id = task_id[task_id.rfind('/')+1:]
    return '%s/logs/%s/%s/stdout' % (project_id, bot_id, real_task_id)

  def _complete_task(self, task_id, bot_id, params, exit_code):
    # Create command result
    res = command_pb2.CommandOutputs()
    res.exit_code = exit_code
    if 'outputs_ref' in params:
      res.outputs.hash = params['outputs_ref']['isolated']
      res.outputs.size_bytes = -1

    # Create command overhead
    ovh = command_pb2.CommandOverhead()
    _time_to_duration(params.get('duration'), ovh.duration)
    # bot_overhead is not set for terminate task.
    if params.get('bot_overhead'):
      _time_to_duration(params.get('bot_overhead'), ovh.overhead)

    # Create task result and pack in command result/overhead
    req = tasks_pb2.UpdateTaskResultRequest()
    req.name = task_id + '/result'
    req.source = bot_id
    req.result.name = req.name
    req.result.complete = True
    if params.get('io_timeout'):
      req.result.status.code = code_pb2.UNAVAILABLE
    elif params.get('hard_timeout'):
      req.result.status.code = code_pb2.DEADLINE_EXCEEDED
    req.result.output.Pack(res)
    req.result.meta.Pack(ovh)

    # Send update
    self._proxy_tasks.call_unary('UpdateTaskResult', req)


def _dimensions_to_workers(dims, worker):
  """Converts botattribute dims to a Worker."""
  if not worker.devices:
    worker.devices.add()
  del worker.properties[:]
  del worker.devices[0].properties[:]
  for k, values in sorted(dims.iteritems()):
    if k == 'id':
      # Proxy treats ID as worker-level, not device-level. But use this for the
      # device name.
      worker.devices[0].handle = values[0]
      continue
    for v in sorted(values):
      prop = None
      if k == 'pool':
        prop = worker.properties.add()
      else:
        prop = worker.devices[0].properties.add()
      prop.key = k
      prop.value = v


def _worker_to_bot_group_cfg(worker):
  """Returns global properties since those are only settable by a config."""
  dims = {}
  for prop in worker.properties:
    k = prop.key
    dims[k] = dims.get(k, [])
    dims[k].append(prop.value)
  return dims

def _time_to_duration(time_f, duration):
    duration.seconds = int(time_f)
    duration.nanos = int(1e9 * (
        time_f - int(time_f)))
