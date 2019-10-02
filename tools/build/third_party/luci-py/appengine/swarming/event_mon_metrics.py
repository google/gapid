# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import time

import gae_event_mon

from server import task_result


DIMENSIONS = (
    ('cores', int),
    ('cpu', unicode),
    ('device_os', unicode),
    ('device_type', unicode),
    ('gpu', unicode),
    ('hidpi', unicode),
    ('machine_type', unicode),
    ('os', unicode),
    ('pool', unicode),
    ('xcode_version', unicode),
    ('zone', unicode),
)


TAGS = {
    'build_id': ['buildnumber'],
    'buildername': ['buildername'],
    'master': ['master'],
    'name': ['name'],
    'patch_project': ['patch_project'],
    'project': ['project'],
    'purpose': ['purpose'],
    'slavename': ['slavename'],
    'spec_name': ['spec_name'],
    'stepname': ['stepname'],
}


def _to_timestamp(dt):
  return int(time.mktime(dt.timetuple()))


def _files_ref_to_proto(files_ref, proto):
  if files_ref.isolated:
    proto.isolated = files_ref.isolated
  if files_ref.isolatedserver:
    proto.isolatedserver = files_ref.isolatedserver
  if files_ref.namespace:
    proto.namespace = files_ref.namespace


def _cipd_package_to_proto(cipd_package, package_proto):
  if cipd_package.package_name:
    package_proto.package_name = cipd_package.package_name
  if cipd_package.version:
    package_proto.version = cipd_package.version
  if package_proto.path:
    package_proto.path = cipd_package.path


def _task_summary_to_proto(summary, event):
  event.proto.swarming_task_event.id = summary.task_id

  request_proto = event.proto.swarming_task_event.request
  if summary.request.parent_task_id:
    request_proto.parent_task_id = summary.request.parent_task_id
  request_proto.name = summary.request.name
  request_proto.created_ts = _to_timestamp(summary.request.created_ts)
  request_proto.expiration_ts = _to_timestamp(summary.request.expiration_ts)
  request_proto.priority = summary.request.priority
  if summary.request.pubsub_topic:
    request_proto.pubsub_topic = summary.request.pubsub_topic

  task_properties = summary.request.task_slice(
      summary.current_task_slice).properties
  properties_proto = request_proto.properties

  if task_properties.inputs_ref:
    _files_ref_to_proto(task_properties.inputs_ref, properties_proto.inputs_ref)

  if task_properties.cipd_input:
    cipd_proto = properties_proto.cipd_input
    cipd_proto.server = task_properties.cipd_input.server

    _cipd_package_to_proto(task_properties.cipd_input.client_package,
                           cipd_proto.client_package)
    for package in task_properties.cipd_input.packages:
      package_proto = cipd_proto.packages.add()
      _cipd_package_to_proto(package, package_proto)

  dimensions = task_properties.dimensions
  for d, t in DIMENSIONS:
    if d in dimensions:
      for v in dimensions[d]:
        getattr(properties_proto.dimensions, d).append(t(v))

  if task_properties.execution_timeout_secs:
    properties_proto.execution_timeout_s = \
        task_properties.execution_timeout_secs
  if task_properties.grace_period_secs:
    properties_proto.grace_period_s = task_properties.grace_period_secs
  if task_properties.io_timeout_secs:
    properties_proto.io_timeout_s = task_properties.io_timeout_secs
  properties_proto.idempotent = task_properties.idempotent

  state_enum = event.proto.swarming_task_event.State.DESCRIPTOR.values_by_name
  if summary.state == task_result.State.COMPLETED:
    event.proto.swarming_task_event.state = state_enum['COMPLETED'].number
  elif summary.state == task_result.State.CANCELED:
    event.proto.swarming_task_event.state = state_enum['CANCELED'].number
  elif summary.state == task_result.State.BOT_DIED:
    event.proto.swarming_task_event.state = state_enum['BOT_DIED'].number
  elif summary.state == task_result.State.TIMED_OUT:
    event.proto.swarming_task_event.state = state_enum['TIMED_OUT'].number
  elif summary.state == task_result.State.EXPIRED:
    event.proto.swarming_task_event.state = state_enum['EXPIRED'].number
  # TODO(maruel): Report KILLED tasks.
  # https://crbug.com/754390
  #elif summary.state == task_result.State.KILLED:
  #  event.proto.swarming_task_event.state = state_enum['KILLED'].number
  else:
    logging.error('Unhandled task state %r', summary.state)

  event.proto.swarming_task_event.bot_id = summary.bot_id
  event.proto.swarming_task_event.bot_version = summary.bot_version

  for d, t in DIMENSIONS:
    for v in summary.bot_dimensions.get(d, []):
      getattr(event.proto.swarming_task_event.bot_dimensions, d).append(t(v))

  for v in summary.server_versions:
    event.proto.swarming_task_event.server_versions.append(v)

  event.proto.swarming_task_event.internal_failure = summary.internal_failure
  if summary.exit_code is not None:
    event.proto.swarming_task_event.exit_code = summary.exit_code
  else:
    # Default value is 0, use -1 so it is not considered a success by accident.
    event.proto.swarming_task_event.exit_code = -1
  if summary.started_ts:
    event.proto.swarming_task_event.started_ts = _to_timestamp(
        summary.started_ts)
  if summary.completed_ts:
    event.proto.swarming_task_event.completed_ts = _to_timestamp(
        summary.completed_ts)
  if summary.abandoned_ts:
    event.proto.swarming_task_event.abandoned_ts = _to_timestamp(
        summary.abandoned_ts)

  for task_id in summary.children_task_ids:
    event.proto.swarming_task_event.children_task_ids.append(task_id)

  if summary.outputs_ref:
    _files_ref_to_proto(
        summary.outputs_ref, event.proto.swarming_task_event.outputs_ref)

  event.proto.swarming_task_event.cost_usd = summary.cost_usd
  if summary.cost_saved_usd:
    event.proto.swarming_task_event.cost_saved_usd = summary.cost_saved_usd
  if summary.deduped_from:
    event.proto.swarming_task_event.deduped_from = summary.deduped_from
  event.proto.swarming_task_event.try_number = summary.try_number

  for tag in summary.tags:
    if ':' not in tag:
      logging.error('Unexpected tag: %r', tag)
      continue
    name, value = tag.split(':', 1)
    for event_tag, task_tags in TAGS.iteritems():
      if name in task_tags:
        getattr(event.proto.swarming_task_event.tags, event_tag).append(value)


def initialize():
  gae_event_mon.initialize('swarming')


def send_task_event(summary):
  """Sends an event_mon event about a swarming task.

  Currently implemented as sending a HTTP request.

  Args:
    summary: TaskResultSummary object.
  """
  # Isolate rest of the app from monitoring pipeline issues. They should
  # not cause outage of swarming.
  try:
    event = gae_event_mon.Event('POINT')
    _task_summary_to_proto(summary, event)
    event.send()
  except Exception:
    logging.exception('Caught exception while sending event')
