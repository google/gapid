# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module facilitates conversion from dictionaries to ProtoRPC messages.

Given a dictionary whose keys' names and values' types comport with the
fields defined for a protorpc.messages.Message subclass, this module tries to
generate a Message instance that corresponds to the provided dictionary. The
"normal" use case is for ndb.Models which need to be represented as a
ProtoRPC.
"""

import datetime
import json

import swarming_rpcs

from components import utils
from server import task_pack
from server import task_request
from server import task_result


### Private API.


def _string_pairs_from_dict(dictionary):
  # For key: value items like env.
  return [
    swarming_rpcs.StringPair(key=k, value=v)
    for k, v in sorted((dictionary or {}).iteritems())
  ]


def _duplicate_string_pairs_from_dict(dictionary):
  # For compatibility due to legacy swarming_rpcs.TaskProperties.dimensions.
  out = []
  for k, values in (dictionary or {}).iteritems():
    assert isinstance(values, (list, tuple)), dictionary
    for v in values:
      out.append(swarming_rpcs.StringPair(key=k, value=v))
  return out


def _string_list_pairs_from_dict(dictionary):
  # For key: values items like bot dimensions.
  return [
    swarming_rpcs.StringListPair(key=k, value=v)
    for k, v in sorted((dictionary or {}).iteritems())
  ]


def _ndb_to_rpc(cls, entity, **overrides):
  members = (f.name for f in cls.all_fields())
  kwargs = {m: getattr(entity, m) for m in members if not m in overrides}
  kwargs.update(overrides)
  return cls(**{k: v for k, v in kwargs.iteritems() if v is not None})


def _rpc_to_ndb(cls, entity, **overrides):
  kwargs = {
    m: getattr(entity, m) for m in cls._properties if not m in overrides
  }
  kwargs.update(overrides)
  return cls(**{k: v for k, v in kwargs.iteritems() if v is not None})


def _taskproperties_from_rpc(props):
  """Converts a swarming_rpcs.TaskProperties to a task_request.TaskProperties.
  """
  cipd_input = None
  if props.cipd_input:
    client_package = None
    if props.cipd_input.client_package:
      client_package = _rpc_to_ndb(
          task_request.CipdPackage, props.cipd_input.client_package)
    cipd_input = _rpc_to_ndb(
        task_request.CipdInput,
        props.cipd_input,
        client_package=client_package,
        packages=[
          _rpc_to_ndb(task_request.CipdPackage, p)
          for p in props.cipd_input.packages
        ])

  inputs_ref = None
  if props.inputs_ref:
    inputs_ref = _rpc_to_ndb(task_request.FilesRef, props.inputs_ref)

  secret_bytes = None
  if props.secret_bytes:
    secret_bytes = task_request.SecretBytes(secret_bytes=props.secret_bytes)

  if len(set(i.key for i in props.env)) != len(props.env):
    raise ValueError('same environment variable key cannot be specified twice')
  if len(set(i.key for i in props.env_prefixes)) != len(props.env_prefixes):
    raise ValueError('same environment prefix key cannot be specified twice')
  dims = {}
  for i in props.dimensions:
    dims.setdefault(i.key, []).append(i.value)
  out = _rpc_to_ndb(
      task_request.TaskProperties,
      props,
      caches=[_rpc_to_ndb(task_request.CacheEntry, c) for c in props.caches],
      cipd_input=cipd_input,
      # Passing command=None is supported at API level but not at NDB level.
      command=props.command or [],
      has_secret_bytes=secret_bytes is not None,
      secret_bytes=None, # ignore this, it's handled out of band
      dimensions=None, # it's named dimensions_data
      dimensions_data=dims,
      env={i.key: i.value for i in props.env},
      env_prefixes={i.key: i.value for i in props.env_prefixes},
      inputs_ref=inputs_ref)
  return out, secret_bytes


def _taskproperties_to_rpc(props):
  """Converts a task_request.TaskProperties to a swarming_rpcs.TaskProperties.
  """
  cipd_input = None
  if props.cipd_input:
    client_package = None
    if props.cipd_input.client_package:
      client_package = _ndb_to_rpc(
          swarming_rpcs.CipdPackage,
          props.cipd_input.client_package)
    cipd_input = _ndb_to_rpc(
        swarming_rpcs.CipdInput,
        props.cipd_input,
        client_package=client_package,
        packages=[
          _ndb_to_rpc(swarming_rpcs.CipdPackage, p)
          for p in props.cipd_input.packages
        ])

  inputs_ref = None
  if props.inputs_ref:
    inputs_ref = _ndb_to_rpc(swarming_rpcs.FilesRef, props.inputs_ref)

  return _ndb_to_rpc(
      swarming_rpcs.TaskProperties,
      props,
      caches=[_ndb_to_rpc(swarming_rpcs.CacheEntry, c) for c in props.caches],
      cipd_input=cipd_input,
      secret_bytes='<REDACTED>' if props.has_secret_bytes else None,
      dimensions=_duplicate_string_pairs_from_dict(props.dimensions),
      env=_string_pairs_from_dict(props.env),
      env_prefixes=_string_list_pairs_from_dict(props.env_prefixes or {}),
      inputs_ref=inputs_ref)


def _taskslice_from_rpc(msg):
  """Converts a swarming_rpcs.TaskSlice to a task_request.TaskSlice."""
  props, secret_bytes = _taskproperties_from_rpc(msg.properties)
  out = _rpc_to_ndb(task_request.TaskSlice, msg, properties=props)
  return out, secret_bytes


### Public API.


def epoch_to_datetime(value):
  """Converts a messages.FloatField that represents a timestamp since epoch in
  seconds to a datetime.datetime.

  Returns None when input is 0 or None.
  """
  if not value:
    return None
  try:
    return utils.timestamp_to_datetime(value*1000000.)
  except OverflowError as e:
    raise ValueError(e)


def bot_info_to_rpc(entity, deleted=False):
  """"Returns a swarming_rpcs.BotInfo from a bot.BotInfo."""
  return _ndb_to_rpc(
      swarming_rpcs.BotInfo,
      entity,
      bot_id=entity.id,
      deleted=deleted,
      dimensions=_string_list_pairs_from_dict(entity.dimensions),
      is_dead=entity.is_dead,
      machine_type=entity.machine_type,
      state=json.dumps(entity.state, sort_keys=True, separators=(',', ':')))


def bot_event_to_rpc(entity):
  """"Returns a swarming_rpcs.BotEvent from a bot.BotEvent."""
  return _ndb_to_rpc(
      swarming_rpcs.BotEvent,
      entity,
      dimensions=_string_list_pairs_from_dict(entity.dimensions),
      state=json.dumps(entity.state, sort_keys=True, separators=(',', ':')),
      task_id=entity.task_id if entity.task_id else None)


def task_request_to_rpc(entity):
  """"Returns a swarming_rpcs.TaskRequest from a task_request.TaskRequest."""
  assert entity.__class__ is task_request.TaskRequest
  slices = []
  for i in xrange(entity.num_task_slices):
    t = entity.task_slice(i)
    slices.append(
        _ndb_to_rpc(
            swarming_rpcs.TaskSlice,
            t,
            properties=_taskproperties_to_rpc(t.properties)))

  return _ndb_to_rpc(
      swarming_rpcs.TaskRequest,
      entity,
      authenticated=entity.authenticated.to_bytes(),
      # For some amount of time, the properties will be copied into the
      # task_slices and vice-versa, to give time to the clients to update.
      properties=slices[0].properties,
      task_slices=slices)


def new_task_request_from_rpc(msg, now):
  """"Returns a (task_request.TaskRequest, task_request.SecretBytes,
  task_request.TemplateApplyEnum) from a swarming_rpcs.NewTaskRequest.

  If secret_bytes were not included in the rpc, the SecretBytes entity will be
  None.
  """
  assert msg.__class__ is swarming_rpcs.NewTaskRequest
  if msg.task_slices and msg.properties:
    raise ValueError('Specify one of properties or task_slices, not both')

  if msg.properties:
    if not msg.expiration_secs:
      raise ValueError('missing expiration_secs')
    props, secret_bytes = _taskproperties_from_rpc(msg.properties)
    slices = [
      task_request.TaskSlice(
          properties=props, expiration_secs=msg.expiration_secs),
    ]
  elif msg.task_slices:
    if msg.expiration_secs:
      raise ValueError(
          'When using task_slices, do not specify a global expiration_secs')
    secret_bytes = None
    slices = []
    for t in (msg.task_slices or []):
      sl, se = _taskslice_from_rpc(t)
      slices.append(sl)
      if se:
        if secret_bytes and se != secret_bytes:
          raise ValueError(
              'When using secret_bytes multiple times, all values must match')
        secret_bytes = se
  else:
    raise ValueError('Specify one of properties or task_slices')

  pttf = swarming_rpcs.PoolTaskTemplateField
  template_apply = {
    pttf.AUTO: task_request.TEMPLATE_AUTO,
    pttf.CANARY_NEVER: task_request.TEMPLATE_CANARY_NEVER,
    pttf.CANARY_PREFER: task_request.TEMPLATE_CANARY_PREFER,
    pttf.SKIP: task_request.TEMPLATE_SKIP,
  }[msg.pool_task_template]

  req = _rpc_to_ndb(
      task_request.TaskRequest,
      msg,
      created_ts=now,
      expiration_ts=None,
      # It is set in task_request.init_new_request().
      authenticated=None,
      properties=None,
      task_slices=slices,
      # 'tags' is now generated from manual_tags plus automatic tags.
      tags=None,
      manual_tags=msg.tags,
      # This is internal field not settable via RPC.
      service_account_token=None,
      pool_task_template=None) # handled out of band

  return req, secret_bytes, template_apply


def task_result_to_rpc(entity, send_stats):
  """"Returns a swarming_rpcs.TaskResult from a task_result.TaskResultSummary or
  task_result.TaskRunResult.
  """
  outputs_ref = (
      _ndb_to_rpc(swarming_rpcs.FilesRef, entity.outputs_ref)
      if entity.outputs_ref else None)
  cipd_pins = None
  if entity.cipd_pins:
    cipd_pins = swarming_rpcs.CipdPins(
      client_package=(
        _ndb_to_rpc(swarming_rpcs.CipdPackage,
                    entity.cipd_pins.client_package)
        if entity.cipd_pins.client_package else None
      ),
      packages=[
        _ndb_to_rpc(swarming_rpcs.CipdPackage, pkg)
        for pkg in entity.cipd_pins.packages
      ] if entity.cipd_pins.packages else None
    )
  performance_stats = None
  if send_stats and entity.performance_stats.is_valid:
      def op(entity):
        if entity:
          return _ndb_to_rpc(swarming_rpcs.OperationStats, entity)

      performance_stats = _ndb_to_rpc(
          swarming_rpcs.PerformanceStats,
          entity.performance_stats,
          isolated_download=op(entity.performance_stats.isolated_download),
          isolated_upload=op(entity.performance_stats.isolated_upload))
  kwargs = {
    'bot_dimensions': _string_list_pairs_from_dict(entity.bot_dimensions or {}),
    'cipd_pins': cipd_pins,
    'outputs_ref': outputs_ref,
    'performance_stats': performance_stats,
    'state': swarming_rpcs.TaskState(entity.state),
  }
  if entity.__class__ is task_result.TaskRunResult:
    kwargs['costs_usd'] = []
    if entity.cost_usd is not None:
      kwargs['costs_usd'].append(entity.cost_usd)
    kwargs['tags'] = []
    kwargs['user'] = None
    kwargs['run_id'] = entity.task_id
  else:
    assert entity.__class__ is task_result.TaskResultSummary, entity
    # This returns the right value for deduped tasks too.
    k = entity.run_result_key
    kwargs['run_id'] = task_pack.pack_run_result_key(k) if k else None
  return _ndb_to_rpc(
      swarming_rpcs.TaskResult,
      entity,
      **kwargs)
