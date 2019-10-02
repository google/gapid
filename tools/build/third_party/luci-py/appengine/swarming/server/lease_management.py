# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Lease management for machines leased from the Machine Provider.

Keeps a list of machine types which should be leased from the Machine Provider
and the list of machines of each type currently leased.

Swarming integration with Machine Provider
==========================================

handlers_backend.py contains a cron job which looks at each MachineType and
ensures there are at least as many MachineLeases in the datastore which refer
to that MachineType as the target_size in MachineType specifies by numbering
them 0 through target_size - 1. If there are MachineType entities numbered
target_size or greater which refer to that MachineType, those MachineLeases
are marked as drained.

Each MachineLease manages itself. A cron job in handlers_backend.py will trigger
self-management jobs for each entity. If there is no associated lease and the
MachineLease is not drained, issue a request to the Machine Provider for a
matching machine. If there is an associated request, check the status of that
request. If it is fulfilled, ensure the existence of a BotInfo entity (see
server/bot_management.py) corresponding to the machine provided for the lease.
Include the lease ID and lease_expiration_ts as fields in the BotInfo. If it
is expired, clear the associated lease. If there is no associated lease and
the MachineLease is drained, delete the MachineLease entity.

Each self-management job performs only one idempotent operation then returns,
meaning each iteration makes incremental progress over the last iteration and
the self-management jobs must be called continuously via cron.
"""

import base64
import collections
import datetime
import json
import logging
import math

from google.appengine.api import app_identity
from google.appengine.api import datastore_errors
from google.appengine.ext import ndb
from google.appengine.ext.ndb import msgprop
from protorpc.remote import protojson

import ts_mon_metrics

from components import datastore_utils
from components import machine_provider
from components import pubsub
from components import utils
from server import bot_groups_config
from server import bot_management
from server import task_queues
from server import task_request
from server import task_result
from server import task_pack
from server import task_scheduler


# Name of the topic the Machine Provider is authorized to publish
# lease information to.
PUBSUB_TOPIC = 'machine-provider'

# Name of the pull subscription to the Machine Provider topic.
PUBSUB_SUBSCRIPTION = 'machine-provider'


class MachineLease(ndb.Model):
  """A lease request for a machine from the Machine Provider.

  Key:
    id: A string in the form <machine type id>-<number>.
    kind: MachineLease. Is a root entity.
  """
  # Bot ID for the BotInfo created for this machine.
  bot_id = ndb.StringProperty(indexed=False)
  # Request ID used to generate this request.
  client_request_id = ndb.StringProperty(indexed=True)
  # DateTime indicating when the bot first connected to the server.
  connection_ts = ndb.DateTimeProperty()
  # Whether or not this MachineLease should issue lease requests.
  drained = ndb.BooleanProperty(indexed=True)
  # Number of seconds ahead of lease_expiration_ts to release leases.
  # Not specified for indefinite leases.
  early_release_secs = ndb.IntegerProperty(indexed=False)
  # Hostname of the machine currently allocated for this request.
  hostname = ndb.StringProperty()
  # DateTime indicating when the instruction to join the server was sent.
  instruction_ts = ndb.DateTimeProperty()
  # Duration to lease for, as specified in the config.
  # Only one of lease_duration_secs and lease_indefinitely must be specified.
  lease_duration_secs = ndb.IntegerProperty(indexed=False)
  # DateTime indicating lease expiration time, as specified by Machine Provider.
  # Only one of lease_expiration_ts and leased_indefinitely must be specified.
  lease_expiration_ts = ndb.DateTimeProperty()
  # Lease ID assigned by Machine Provider.
  lease_id = ndb.StringProperty(indexed=False)
  # Lease indefinitely, as specified in the config.
  # Only one of lease_duration_secs and lease_indefinitely must be specified.
  lease_indefinitely = ndb.BooleanProperty()
  # Leased indefinitely, as specified by Machine Provider.
  # Only one of lease_expiration_ts and leased_indefinitely must be specified.
  leased_indefinitely = ndb.BooleanProperty()
  # ndb.Key for the MachineType this MachineLease is created for.
  machine_type = ndb.KeyProperty()
  # machine_provider.Dimensions describing the machine.
  mp_dimensions = msgprop.MessageProperty(
      machine_provider.Dimensions, indexed=False)
  # Last request number used.
  request_count = ndb.IntegerProperty(default=0, required=True)
  # Base string to use as the request ID.
  request_id_base = ndb.StringProperty(indexed=False)
  # Task ID for the termination task scheduled for this machine.
  termination_task = ndb.StringProperty(indexed=False)

  def _pre_put_hook(self):
    super(MachineLease, self)._pre_put_hook()
    if self.lease_duration_secs and self.lease_indefinitely:
      raise datastore_errors.BadValueError(
        'lease_duration_secs and lease_indefinitely both set:\n%s' % self)
    if self.early_release_secs and self.lease_indefinitely:
      raise datastore_errors.BadValueError(
        'early_release_secs and lease_indefinitely both set:\n%s' % self)
    if self.lease_expiration_ts and self.leased_indefinitely:
      raise datastore_errors.BadValueError(
        'lease_expiration_ts and leased_indefinitely both set:\n%s' % self)


class MachineType(ndb.Model):
  """A type of machine which should be leased from the Machine Provider.

  Key:
    id: A human-readable name for this machine type.
    kind: MachineType. Is a root entity.
  """
  # Description of this machine type for humans.
  description = ndb.StringProperty(indexed=False)
  # Number of seconds ahead of lease_expiration_ts to release leases.
  early_release_secs = ndb.IntegerProperty(indexed=False)
  # Whether or not to attempt to lease machines of this type.
  enabled = ndb.BooleanProperty(default=True)
  # Duration to lease each machine for.
  # Only one of lease_duration_secs and lease_indefinitely must be specified.
  lease_duration_secs = ndb.IntegerProperty(indexed=False)
  # Lease indefinitely.
  # Only one of lease_duration_secs and lease_indefinitely must be specified.
  lease_indefinitely = ndb.BooleanProperty(indexed=False)
  # machine_provider.Dimensions describing the machine.
  mp_dimensions = msgprop.MessageProperty(
      machine_provider.Dimensions, indexed=False)
  # Target number of machines of this type to have leased at once.
  target_size = ndb.IntegerProperty(indexed=False, required=True)

  def _pre_put_hook(self):
    super(MachineType, self)._pre_put_hook()
    if self.lease_duration_secs and self.lease_indefinitely:
      raise datastore_errors.BadValueError(
        'lease_duration_secs and lease_indefinitely both set:\n%s' % self)


class MachineTypeUtilization(ndb.Model):
  """Utilization numbers for a MachineType.

  Key:
    id: Name of the MachineType these utilization numbers are associated with.
    kind: MachineTypeUtilization. Is a root entity.
  """
  # Number of busy bots created from this machine type.
  busy = ndb.IntegerProperty(indexed=False)
  # Number of idle bots created from this machine type.
  idle = ndb.IntegerProperty(indexed=False)
  # DateTime indicating when busy/idle numbers were last computed.
  last_updated_ts = ndb.DateTimeProperty()


## Private stuff.


@ndb.transactional_tasklet
def _create_machine_lease(machine_lease_key, machine_type):
  """Creates a MachineLease from the given MachineType and MachineLease key.

  Args:
    machine_lease_key: ndb.Key for a MachineLease entity.
    machine_type: MachineType entity.
  """
  machine_lease = yield machine_lease_key.get_async()
  if machine_lease:
    return

  yield MachineLease(
      key=machine_lease_key,
      lease_duration_secs=machine_type.lease_duration_secs,
      lease_indefinitely=machine_type.lease_indefinitely,
      early_release_secs=machine_type.early_release_secs,
      machine_type=machine_type.key,
      mp_dimensions=machine_type.mp_dimensions,
      # Deleting and recreating the MachineLease needs a unique base request ID,
      # otherwise it will hit old requests.
      request_id_base='%s-%s' % (machine_lease_key.id(), utils.time_time()),
  ).put_async()


@ndb.transactional_tasklet
def _update_machine_lease(machine_lease_key, machine_type):
  """Updates the given MachineLease from the given MachineType.

  Args:
    machine_lease_key: ndb.Key for a MachineLease entity.
    machine_type: MachineType entity.
  """
  machine_lease = yield machine_lease_key.get_async()
  if not machine_lease:
    logging.error('MachineLease not found:\nKey: %s', machine_lease_key)
    return

  # When updating an indefinitely leased machine, drain it, otherwise the
  # new lease parameters, which only take effect on the next lease, will
  # not take effect until the machine is manually deleted.
  put = False
  drain = False

  # See _ensure_entity_exists below for why we only update leased machines.
  if not machine_lease.lease_expiration_ts:
    if not machine_lease.leased_indefinitely:
      return
    drain = True

  if machine_lease.early_release_secs != machine_type.early_release_secs:
    machine_lease.early_release_secs = machine_type.early_release_secs
    put = True

  if machine_lease.lease_duration_secs != machine_type.lease_duration_secs:
    machine_lease.lease_duration_secs = machine_type.lease_duration_secs
    put = True

  if machine_lease.lease_indefinitely != machine_type.lease_indefinitely:
    machine_lease.lease_indefinitely = machine_type.lease_indefinitely
    put = True

  if machine_lease.mp_dimensions != machine_type.mp_dimensions:
    machine_lease.mp_dimensions = machine_type.mp_dimensions
    put = True

  if not put:
    return

  if drain:
    logging.info('Draining MachineLease:\nKey: %s\nHostname: %s',
                 machine_lease_key, machine_lease.hostname)
    machine_lease.drained = True

  yield machine_lease.put_async()


@ndb.tasklet
def _ensure_entity_exists(machine_type, n):
  """Ensures the nth MachineLease for the given MachineType exists.

  Args:
    machine_type: MachineType entity.
    n: The MachineLease index.
  """
  machine_lease_key = ndb.Key(
      MachineLease, '%s-%s' % (machine_type.key.id(), n))
  machine_lease = yield machine_lease_key.get_async()

  if not machine_lease:
    yield _create_machine_lease(machine_lease_key, machine_type)
    return

  # If there is a MachineLease, we may need to update it if the MachineType's
  # lease properties have changed. It's only safe to update it if the current
  # lease is fulfilled (indicated by the presence of lease_expiration_ts or
  # leased_indefinitely) so the changes only go into effect for the next lease
  # request. This is because leasing parameters are immutable in Machine
  # Provider.
  if machine_lease.lease_expiration_ts or machine_lease.leased_indefinitely:
    if (machine_lease.early_release_secs != machine_type.early_release_secs
        or machine_lease.lease_duration_secs != machine_type.lease_duration_secs
        or machine_lease.lease_indefinitely != machine_type.lease_indefinitely
        or machine_lease.mp_dimensions != machine_type.mp_dimensions
    ):
      yield _update_machine_lease(machine_lease_key, machine_type)


def _machine_type_pb2_to_entity(pb2):
  """Creates a MachineType entity from the given bots_pb2.MachineType.

  Args:
    pb2: A proto.bots_pb2.MachineType proto.

  Returns:
    A MachineType entity.
  """
  # Put dimensions into k: [v0, v1, v2, ...] form. protojson.decode_message
  # can handle non-repeated dimensions in a list. At this point, it's verified
  # that only dimensions allowed to be repeated are. See bot_groups_config.py.
  dims = {}
  for dim in pb2.mp_dimensions:
    k, v = dim.split(':', 1)
    dims.setdefault(k, []).append(v)
  return MachineType(
      id=pb2.name,
      description=pb2.description,
      early_release_secs=pb2.early_release_secs,
      enabled=True,
      lease_duration_secs=pb2.lease_duration_secs,
      lease_indefinitely=pb2.lease_indefinitely,
      mp_dimensions=protojson.decode_message(
          machine_provider.Dimensions,
          json.dumps(dims),
      ),
      target_size=pb2.target_size,
  )


def _get_target_size(schedule, machine_type, current, default, now=None):
  """Returns the current target size for the MachineType.

  Args:
    schedule: A proto.bots_pb2.Schedule proto.
    machine_type: ID of the key for the MachineType to get a target size for.
    current: The current target_size. Used to ensure load-based target size
      recommendations don't drop too quickly.
    default: A default to return if now is not within any of config's intervals
      or the last-known utilization is not set.
    now: datetime.datetime to use as the time to check what the MachineType's
      target size currently is. Defaults to use the current time if unspecified.

  Returns:
    Target size.
  """
  now = now or utils.utcnow()

  # The validator ensures the given time will fall in at most one interval,
  # because intervals are not allowed to intersect. So just search linearly
  # for a matching interval.
  # TODO(smut): Improve linear search if we end up with many intervals.
  for i in schedule.daily:
    # If the days of the week given by this interval do not include the current
    # day, move on to the next interval. If no days of the week are given by
    # this interval at all, then the interval applies every day.
    if i.days_of_the_week and now.weekday() not in i.days_of_the_week:
      continue

    # Get the start and end times of this interval relative to the current day.
    h, m = map(int, i.start.split(':'))
    start = datetime.datetime(now.year, now.month, now.day, h, m)
    h, m = map(int, i.end.split(':'))
    end = datetime.datetime(now.year, now.month, now.day, h, m)

    if start <= now <= end:
      return i.target_size

  # Fall back on load-based scheduling. This allows combining scheduled changes
  # with load-based changes occurring outside any explicitly given intervals.
  # Only one load-based schedule is supported.
  if schedule.load_based:
    utilization = ndb.Key(MachineTypeUtilization, machine_type).get()
    if not utilization:
      return default
    logging.info(
        'Last known utilization for MachineType %s: %s/%s (computed at %s)',
        machine_type,
        utilization.busy,
        utilization.busy + utilization.idle,
        utilization.last_updated_ts,
    )
    # Target 10% more than the number of busy bots, but not more than the
    # configured maximum and not less than the configured minimum. In order
    # to prevent drastic drops, do not allow the target size to fall below 99%
    # of current capacity. Note that this dampens scale downs as a function of
    # the frequency with which this function runs, which is currently every
    # minute controlled by cron job. Tweak these numbers if the cron frequency
    # changes.
    # TODO(smut): Tune this algorithm.
    # TODO(smut): Move algorithm parameters to luci-config.
    target = int(math.ceil(utilization.busy * 1.5))
    if target >= schedule.load_based[0].maximum_size:
      return schedule.load_based[0].maximum_size
    if target < int(0.99 * current):
      target = int(0.99 * current)
    if target < schedule.load_based[0].minimum_size:
      target = schedule.load_based[0].minimum_size
    return target

  return default


def _ensure_entities_exist(max_concurrent=50):
  """Ensures MachineType entities are correct, and MachineLease entities exist.

  Updates MachineType entities based on the config and creates corresponding
  MachineLease entities.

  Args:
    max_concurrent: Maximum number of concurrent asynchronous requests.
  """
  now = utils.utcnow()
  # Seconds and microseconds are too granular for determining scheduling.
  now = datetime.datetime(now.year, now.month, now.day, now.hour, now.minute)

  # Generate a few asynchronous requests at a time in order to prevent having
  # too many in flight at a time.
  futures = []
  machine_types = bot_groups_config.fetch_machine_types().copy()
  total = len(machine_types)

  for machine_type in MachineType.query():
    # Check the MachineType in the datastore against its config.
    # If it no longer exists, just disable it here. If it exists but
    # doesn't match, update it.
    machine_type_cfg = machine_types.pop(machine_type.key.id(), None)

    # If there is no config, disable the MachineType.
    if not machine_type_cfg:
      if machine_type.enabled:
        machine_type.enabled = False
        futures.append(machine_type.put_async())
        logging.info('Disabling deleted MachineType: %s', machine_type)
      continue

    put = False

    # Re-enable disabled MachineTypes.
    if not machine_type.enabled:
      logging.info('Enabling MachineType: %s', machine_type)
      machine_type.enabled = True
      put = True

    # Handle scheduled config changes.
    if machine_type_cfg.schedule:
      target_size = _get_target_size(
          machine_type_cfg.schedule,
          machine_type.key.id(),
          machine_type.target_size,
          machine_type_cfg.target_size,
          now=now,
      )
      if machine_type.target_size != target_size:
        logging.info(
            'Adjusting target_size (%s -> %s) for MachineType: %s',
            machine_type.target_size,
            target_size,
            machine_type,
        )
        machine_type.target_size = target_size
        put = True

    # If the MachineType does not match the config, update it. Copy the values
    # of certain fields so we can compare the MachineType to the config to check
    # for differences in all other fields.
    ent = _machine_type_pb2_to_entity(machine_type_cfg)
    ent.target_size = machine_type.target_size
    if machine_type != ent:
      logging.info('Updating MachineType: %s', ent)
      machine_type = ent
      put = True

    # If there's anything to update, update it once here.
    if put:
      futures.append(machine_type.put_async())

    # If the MachineType isn't enabled, don't create MachineLease entities.
    if not machine_type.enabled:
      continue

    # Ensure the existence of MachineLease entities.
    cursor = 0
    while cursor < machine_type.target_size:
      while len(futures) < max_concurrent and cursor < machine_type.target_size:
        futures.append(_ensure_entity_exists(machine_type, cursor))
        cursor += 1
      ndb.Future.wait_any(futures)
      # We don't bother checking success or failure. If a transient error
      # like TransactionFailed or DeadlineExceeded is raised and an entity
      # is not created, we will just create it the next time this is called,
      # converging to the desired state eventually.
      futures = [future for future in futures if not future.done()]

  # Create MachineTypes that never existed before.
  # The next iteration of this cron job will create their MachineLeases.
  machine_types_values = machine_types.values()

  while machine_types_values:
    num_futures = len(futures)
    if num_futures < max_concurrent:
      futures.extend(
          _machine_type_pb2_to_entity(m).put_async()
          for m in machine_types_values[:max_concurrent - num_futures]
      )
      machine_types_values = machine_types_values[max_concurrent - num_futures:]
    ndb.Future.wait_any(futures)
    futures = [future for future in futures if not future.done()]

  if futures:
    ndb.Future.wait_all(futures)
  return total


@ndb.transactional_tasklet
def _drain_entity(key):
  """Drains the given MachineLease.

  Args:
    key: ndb.Key for a MachineLease entity.
  """
  machine_lease = yield key.get_async()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if machine_lease.drained:
    return

  logging.info(
      'Draining MachineLease:\nKey: %s\nHostname: %s',
      key,
      machine_lease.hostname,
  )
  machine_lease.drained = True
  yield machine_lease.put_async()


@ndb.tasklet
def _ensure_entity_drained(machine_lease):
  """Ensures the given MachineLease is drained.

  Args:
    machine_lease: MachineLease entity.
  """
  if machine_lease.drained:
    return

  yield _drain_entity(machine_lease.key)


def _drain_excess(max_concurrent=50):
  """Marks MachineLeases beyond what is needed by their MachineType as drained.

  Args:
    max_concurrent: Maximum number of concurrent asynchronous requests.
  """
  futures = []

  for machine_type in MachineType.query():
    for machine_lease in MachineLease.query(
        MachineLease.machine_type == machine_type.key,
    ):
      try:
        index = int(machine_lease.key.id().rsplit('-', 1)[-1])
      except ValueError:
        logging.error(
            'MachineLease index could not be deciphered\n Key: %s',
            machine_lease.key,
        )
        continue
      # Drain MachineLeases where the MachineType is not enabled or the index
      # exceeds the target_size given by the MachineType. Since MachineLeases
      # are created in contiguous blocks, only indices 0 through target_size - 1
      # should exist.
      if not machine_type.enabled or index >= machine_type.target_size:
        if len(futures) == max_concurrent:
          ndb.Future.wait_any(futures)
          futures = [future for future in futures if not future.done()]
        futures.append(_ensure_entity_drained(machine_lease))

  if futures:
    ndb.Future.wait_all(futures)


@ndb.transactional
def _clear_lease_request(key, request_id):
  """Clears information about given lease request.

  Args:
    key: ndb.Key for a MachineLease entity.
    request_id: ID of the request to clear.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if not machine_lease.client_request_id:
    return

  if request_id != machine_lease.client_request_id:
    # Already cleared and incremented?
    logging.warning(
        'Request ID mismatch for MachineLease: %s\nExpected: %s\nActual: %s',
        key,
        request_id,
        machine_lease.client_request_id,
    )
    return

  machine_lease.bot_id = None
  machine_lease.client_request_id = None
  machine_lease.connection_ts = None
  machine_lease.hostname = None
  machine_lease.instruction_ts = None
  machine_lease.lease_expiration_ts = None
  machine_lease.lease_id = None
  machine_lease.leased_indefinitely = None
  machine_lease.termination_task = None
  machine_lease.put()


@ndb.transactional
def _clear_termination_task(key, task_id):
  """Clears the termination task associated with the given lease request.

  Args:
    key: ndb.Key for a MachineLease entity.
    task_id: ID for a termination task.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if not machine_lease.termination_task:
    return

  if task_id != machine_lease.termination_task:
    logging.error(
        'Task ID mismatch\nKey: %s\nExpected: %s\nActual: %s',
        key,
        task_id,
        machine_lease.task_id,
    )
    return

  machine_lease.termination_task = None
  machine_lease.put()


@ndb.transactional
def _associate_termination_task(key, hostname, task_id):
  """Associates a termination task with the given lease request.

  Args:
    key: ndb.Key for a MachineLease entity.
    hostname: Hostname of the machine the termination task is for.
    task_id: ID for a termination task.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if hostname != machine_lease.hostname:
    logging.error(
        'Hostname mismatch\nKey: %s\nExpected: %s\nActual: %s',
        key,
        hostname,
        machine_lease.hostname,
    )
    return

  if machine_lease.termination_task:
    return

  logging.info(
      'Associating termination task\nKey: %s\nHostname: %s\nTask ID: %s',
      key,
      machine_lease.hostname,
      task_id,
  )
  machine_lease.termination_task = task_id
  machine_lease.put()


@ndb.transactional
def _log_lease_fulfillment(
    key, request_id, hostname, lease_expiration_ts, leased_indefinitely,
    lease_id):
  """Logs lease fulfillment.

  Args:
    key: ndb.Key for a MachineLease entity.
    request_id: ID of the request being fulfilled.
    hostname: Hostname of the machine fulfilling the request.
    lease_expiration_ts: UTC seconds since epoch when the lease expires.
    leased_indefinitely: Whether this lease is indefinite or not. Supersedes
      lease_expiration_ts.
    lease_id: ID of the lease assigned by Machine Provider.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if request_id != machine_lease.client_request_id:
    logging.error(
        'Request ID mismatch\nKey: %s\nExpected: %s\nActual: %s',
        key,
        request_id,
        machine_lease.client_request_id,
    )
    return

  # If we've already logged this lease fulfillment, there's nothing to do.
  if hostname == machine_lease.hostname and lease_id == machine_lease.lease_id:
    if lease_expiration_ts == machine_lease.lease_expiration_ts:
      return
    if leased_indefinitely and machine_lease.indefinite:
      return

  machine_lease.hostname = hostname
  if leased_indefinitely:
    machine_lease.leased_indefinitely = True
  else:
    machine_lease.lease_expiration_ts = datetime.datetime.utcfromtimestamp(
        lease_expiration_ts)
  machine_lease.lease_id = lease_id
  machine_lease.put()


@ndb.transactional
def _update_client_request_id(key):
  """Sets the client request ID used to lease a machine.

  Args:
    key: ndb.Key for a MachineLease entity.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if machine_lease.drained:
    logging.info('MachineLease is drained\nKey: %s', key)
    return

  if machine_lease.client_request_id:
    return

  machine_lease.request_count += 1
  machine_lease.client_request_id = '%s-%s' % (
      machine_lease.request_id_base, machine_lease.request_count)
  machine_lease.put()


@ndb.transactional
def _delete_machine_lease(key):
  """Deletes the given MachineLease if it is drained and has no active lease.

  Args:
    key: ndb.Key for a MachineLease entity.
  """
  machine_lease = key.get()
  if not machine_lease:
    return

  if not machine_lease.drained:
    logging.warning('MachineLease not drained: %s', key)
    return

  if machine_lease.client_request_id:
    return

  key.delete()


@ndb.transactional
def _associate_bot_id(key, bot_id):
  """Associates a bot with the given machine lease.

  Args:
    key: ndb.Key for a MachineLease entity.
    bot_id: ID for a bot.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if machine_lease.bot_id and bot_id != machine_lease.bot_id:
    logging.error('MachineLease already replaced:\n%s', machine_lease)
    return

  if bot_id != machine_lease.hostname:
    logging.error('MachineLease already released:\n%s', machine_lease)
    return

  if machine_lease.bot_id == machine_lease.hostname:
    return

  assert not machine_lease.bot_id, machine_lease
  machine_lease.bot_id = bot_id
  machine_lease.put()


def _ensure_bot_info_exists(machine_lease):
  """Ensures a BotInfo entity exists and has Machine Provider-related fields.

  Args:
    machine_lease: MachineLease instance.
  """
  if machine_lease.bot_id == machine_lease.hostname:
    return
  bot_info = bot_management.get_info_key(machine_lease.hostname).get()
  if not (
      bot_info
      and bot_info.lease_id
      and (bot_info.lease_expiration_ts or bot_info.leased_indefinitely)
      and bot_info.machine_type
  ):
    logging.info(
        'Creating BotEvent\nKey: %s\nHostname: %s\nBotInfo: %s',
        machine_lease.key,
        machine_lease.hostname,
        bot_info,
    )
    bot_management.bot_event(
        event_type='bot_leased',
        bot_id=machine_lease.hostname,
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
        lease_id=machine_lease.lease_id,
        lease_expiration_ts=machine_lease.lease_expiration_ts,
        leased_indefinitely=machine_lease.leased_indefinitely,
        machine_type=machine_lease.machine_type.id(),
        machine_lease=machine_lease.key.id(),
    )
    # Occasionally bot_management.bot_event fails to store the BotInfo so
    # verify presence of Machine Provider fields. See https://crbug.com/681224.
    bot_info = bot_management.get_info_key(machine_lease.hostname).get()
    if not (
        bot_info
        and bot_info.lease_id
        and (bot_info.lease_expiration_ts or bot_info.leased_indefinitely)
        and bot_info.machine_type
        and bot_info.machine_lease
    ):
      # If _associate_bot_id isn't called, cron will try again later.
      logging.error(
          'Failed to put BotInfo\nKey: %s\nHostname: %s\nBotInfo: %s',
          machine_lease.key,
          machine_lease.hostname,
          bot_info,
      )
      return
    logging.info(
        'Put BotInfo\nKey: %s\nHostname: %s\nBotInfo: %s',
        machine_lease.key,
        machine_lease.hostname,
        bot_info,
    )
  else:
    logging.info(
        'Associating BotInfo\nKey: %s\nHostname: %s\nBotInfo: %s',
        machine_lease.key,
        machine_lease.hostname,
        bot_info,
    )
  _associate_bot_id(machine_lease.key, machine_lease.hostname)


@ndb.transactional
def _associate_instruction_ts(key, instruction_ts):
  """Associates an instruction time with the given machine lease.

  Args:
    key: ndb.Key for a MachineLease entity.
    instruction_ts: DateTime indicating when the leased machine was instructed.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if not machine_lease.bot_id:
    logging.error('MachineLease already released:\n%s', machine_lease)
    return

  if machine_lease.instruction_ts:
    return

  machine_lease.instruction_ts = instruction_ts
  machine_lease.put()


def _send_connection_instruction(machine_lease):
  """Sends an instruction to the given machine to connect to the server.

  Args:
    machine_lease: MachineLease instance.
  """
  assert machine_lease.bot_id, machine_lease
  now = utils.utcnow()
  response = machine_provider.instruct_machine(
      machine_lease.client_request_id,
      'https://%s' % app_identity.get_default_version_hostname(),
  )
  if not response:
    logging.error(
        'MachineLease instruction got empty response:\nKey: %s\nHostname: %s',
        machine_lease.key,
        machine_lease.hostname,
    )
  elif not response.get('error'):
    logging.info(
        'MachineLease instruction sent:\nKey: %s\nHostname: %s',
        machine_lease.key,
        machine_lease.hostname,
    )
    _associate_instruction_ts(machine_lease.key, now)
  elif response['error'] == 'ALREADY_RECLAIMED':
    # Can happen if lease duration is very short or there is a significant delay
    # in creating the BotInfo or instructing the machine. Consider it an error.
    logging.error(
        'MachineLease expired before machine connected:\nKey: %s\nHostname: %s',
        machine_lease.key,
        machine_lease.hostname,
    )
    _clear_lease_request(machine_lease.key, machine_lease.client_request_id)
  else:
    logging.warning(
        'MachineLease instruction error:\nKey: %s\nHostname: %s\nError: %s',
        machine_lease.key,
        machine_lease.hostname,
        response['error'],
    )


@ndb.transactional
def _associate_connection_ts(key, connection_ts):
  """Associates a connection time with the given machine lease.

  Args:
    key: ndb.Key for a MachineLease entity.
    connection_ts: DateTime indicating when the bot first connected.
  """
  machine_lease = key.get()
  if not machine_lease:
    logging.error('MachineLease does not exist\nKey: %s', key)
    return

  if machine_lease.connection_ts:
    return

  if not machine_lease.bot_id:
    # Can happen if two task queue tasks to manage the same lease are in flight
    # at once, and one releases the lease on a bot which didn't connect in time
    # while the bot connects right after, and the other sees the new connection.
    logging.warning('MachineLease already released:\n%s', machine_lease)
    return

  machine_lease.connection_ts = connection_ts
  machine_lease.put()


def _check_for_connection(machine_lease):
  """Checks for a bot_connected event.

  Args:
    machine_lease: MachineLease instance.
  """
  assert machine_lease.bot_id, machine_lease
  assert machine_lease.instruction_ts, machine_lease

  # Technically this query is wrong because it looks at events in reverse
  # chronological order. The connection time we find here is actually the
  # most recent connection when we want the earliest. However, this function
  # is only called for new bots and stops being called once the connection
  # time is recorded, so the connection time we record should end up being the
  # first connection anyways. Iterating in the correct order would require
  # building a new, large index.
  for event in bot_management.get_events_query(machine_lease.bot_id, True):
    # We don't want to find a bot_connected event from before we sent the
    # connection instruction (e.g. in the event of hostname reuse), so do not
    # look at events from before the connection instruction was sent.
    if event.ts < machine_lease.instruction_ts:
      break
    # TODO(smut): Create bot_released event, then abandon if we see it.
    # This will avoid the situation where we process the connection of a bot
    # which connects just after it was released for taking too long to connect.
    if event.event_type == 'bot_connected':
      logging.info(
          'Bot connected:\nKey: %s\nHostname: %s\nTime: %s',
          machine_lease.key,
          machine_lease.hostname,
          event.ts,
      )
      _associate_connection_ts(machine_lease.key, event.ts)
      ts_mon_metrics.on_machine_connected_time(
          (event.ts - machine_lease.instruction_ts).total_seconds(),
          fields={
              'machine_type': machine_lease.machine_type.id(),
          },
      )
      return

  # The bot hasn't connected yet. If it's dead or missing, release the lease.
  # At this point we have sent the connection instruction so the bot could still
  # connect after we release the lease but before Machine Provider actually
  # deletes the bot. Therefore we also schedule a termination task if releasing
  # the bot. That way, if the bot connects, it will just shut itself down.
  bot_info = bot_management.get_info_key(machine_lease.hostname).get()
  if not bot_info:
    logging.error(
        'BotInfo missing:\nKey: %s\nHostname: %s',
        machine_lease.key,
        machine_lease.hostname,
    )
    task = task_request.create_termination_task(
        machine_lease.hostname, wait_for_capacity=True)
    task_scheduler.schedule_request(task, secret_bytes=None)
    if release(machine_lease):
      _clear_lease_request(machine_lease.key, machine_lease.client_request_id)
    return
  if bot_info.is_dead:
    logging.warning(
        'Bot failed to connect in time:\nKey: %s\nHostname: %s',
        machine_lease.key,
        machine_lease.hostname,
    )
    task = task_request.create_termination_task(
        machine_lease.hostname, wait_for_capacity=True)
    task_scheduler.schedule_request(task, secret_bytes=None)
    if release(machine_lease):
      cleanup_bot(machine_lease)


def _handle_termination_task(machine_lease):
  """Checks the state of the termination task, releasing the lease if completed.

  Args:
    machine_lease: MachineLease instance.
  """
  assert machine_lease.termination_task

  task_result_summary = task_pack.unpack_result_summary_key(
      machine_lease.termination_task).get()
  if task_result_summary.state in task_result.State.STATES_EXCEPTIONAL:
    logging.info(
        'Termination failed:\nKey: %s\nHostname: %s\nTask ID: %s\nState: %s',
        machine_lease.key,
        machine_lease.hostname,
        machine_lease.termination_task,
        task_result.State.to_string(task_result_summary.state),
    )
    _clear_termination_task(machine_lease.key, machine_lease.termination_task)
    return

  if task_result_summary.state == task_result.State.COMPLETED:
    # There is a race condition where the bot reports the termination task as
    # completed but hasn't exited yet. The last thing it does before exiting
    # is post a bot_shutdown event. Check for the presence of a bot_shutdown
    # event which occurred after the termination task was completed.
    shutdown_ts = last_shutdown_ts(machine_lease.bot_id)
    if not shutdown_ts or shutdown_ts < task_result_summary.completed_ts:
      logging.info(
          'Machine terminated but not yet shut down:\nKey: %s\nHostname: %s',
          machine_lease.key,
          machine_lease.hostname,
      )
      return

    if release(machine_lease):
      cleanup_bot(machine_lease)


def _handle_early_release(machine_lease):
  """Handles the early release of a leased machine.

  Early release can be due to configured early release time, or being drained.

  Args:
    machine_lease: MachineLease instance.
  """
  assert not machine_lease.termination_task, machine_lease.termination_task

  # Machines leased indefinitely can only be released if they are drained.
  # All other machines can be released if they are drained or their early
  # release time has arrived.
  if not machine_lease.drained:
    early_release_ts = machine_lease.lease_expiration_ts - datetime.timedelta(
        seconds=machine_lease.early_release_secs)
    if early_release_ts > utils.utcnow():
      return

  logging.info(
      'MachineLease ready to be released:\nKey: %s\nHostname: %s',
      machine_lease.key,
      machine_lease.hostname,
  )
  task = task_request.create_termination_task(
      machine_lease.hostname, wait_for_capacity=True)
  task_result_summary = task_scheduler.schedule_request(
      task, secret_bytes=None)
  _associate_termination_task(
      machine_lease.key, machine_lease.hostname, task_result_summary.task_id)


def _manage_leased_machine(machine_lease):
  """Manages a leased machine.

  Args:
    machine_lease: MachineLease instance with client_request_id, hostname,
      lease_expiration_ts set.
  """
  assert machine_lease.client_request_id, machine_lease.key
  assert machine_lease.hostname, machine_lease.key
  assert (machine_lease.lease_expiration_ts
          or machine_lease.leased_indefinitely), machine_lease.key

  # Handle a newly leased machine.
  if not machine_lease.bot_id:
    _ensure_bot_info_exists(machine_lease)
    return

  # Once BotInfo is created, send the instruction to join the server.
  if not machine_lease.instruction_ts:
    _send_connection_instruction(machine_lease)
    return

  # Once the instruction is sent, check for connection.
  if not machine_lease.connection_ts:
    _check_for_connection(machine_lease)
    return

  # Handle an expired lease.
  if not machine_lease.leased_indefinitely:
    if machine_lease.lease_expiration_ts <= utils.utcnow():
      logging.info(
          'MachineLease expired:\nKey: %s\nHostname: %s',
          machine_lease.key,
          machine_lease.hostname,
      )
      cleanup_bot(machine_lease)
      return

  # Handle an active lease with a termination task scheduled.
  # TODO(smut): Check if the bot got terminated by some other termination task.
  if machine_lease.termination_task:
    logging.info(
        'MachineLease pending termination:\nKey: %s\nHostname: %s\nTask ID: %s',
        machine_lease.key,
        machine_lease.hostname,
        machine_lease.termination_task,
    )
    _handle_termination_task(machine_lease)
    return

  # Handle a lease ready for early release.
  if machine_lease.early_release_secs or machine_lease.drained:
    _handle_early_release(machine_lease)
    return


def _handle_lease_request_error(machine_lease, response):
  """Handles an error in the lease request response from Machine Provider.

  Args:
    machine_lease: MachineLease instance.
    response: Response returned by components.machine_provider.lease_machine.
  """
  error = machine_provider.LeaseRequestError.lookup_by_name(response['error'])
  if error in (
      machine_provider.LeaseRequestError.DEADLINE_EXCEEDED,
      machine_provider.LeaseRequestError.TRANSIENT_ERROR,
  ):
    logging.warning(
        'Transient failure: %s\nRequest ID: %s\nError: %s',
        machine_lease.key,
        response['client_request_id'],
        response['error'],
    )
  else:
    logging.error(
        'Lease request failed\nKey: %s\nRequest ID: %s\nError: %s',
        machine_lease.key,
        response['client_request_id'],
        response['error'],
    )
    _clear_lease_request(machine_lease.key, machine_lease.client_request_id)


def _handle_lease_request_response(machine_lease, response):
  """Handles a successful lease request response from Machine Provider.

  Args:
    machine_lease: MachineLease instance.
    response: Response returned by components.machine_provider.lease_machine.
  """
  assert not response.get('error')
  state = machine_provider.LeaseRequestState.lookup_by_name(response['state'])
  if state == machine_provider.LeaseRequestState.FULFILLED:
    if not response.get('hostname'):
      # Lease has already expired. This shouldn't happen, but it indicates the
      # lease expired faster than we could tell it even got fulfilled.
      logging.error(
          'Request expired\nKey: %s\nRequest ID:%s\nExpired: %s',
          machine_lease.key,
          machine_lease.client_request_id,
          response['lease_expiration_ts'],
      )
      _clear_lease_request(machine_lease.key, machine_lease.client_request_id)
    else:
      expires = response['lease_expiration_ts']
      if response.get('leased_indefinitely'):
        expires = 'never'
      logging.info(
          'Request fulfilled: %s\nRequest ID: %s\nHostname: %s\nExpires: %s',
          machine_lease.key,
          machine_lease.client_request_id,
          response['hostname'],
          expires,
      )
      _log_lease_fulfillment(
          machine_lease.key,
          machine_lease.client_request_id,
          response['hostname'],
          int(response['lease_expiration_ts']),
          response.get('leased_indefinitely'),
          response['request_hash'],
      )
  elif state == machine_provider.LeaseRequestState.DENIED:
    logging.warning(
        'Request denied: %s\nRequest ID: %s',
        machine_lease.key,
        machine_lease.client_request_id,
    )
    _clear_lease_request(machine_lease.key, machine_lease.client_request_id)


def _manage_pending_lease_request(machine_lease):
  """Manages a pending lease request.

  Args:
    machine_lease: MachineLease instance with client_request_id set.
  """
  assert machine_lease.client_request_id, machine_lease.key

  logging.info(
      'Sending lease request: %s\nRequest ID: %s',
      machine_lease.key,
      machine_lease.client_request_id,
  )
  response = machine_provider.lease_machine(
      machine_provider.LeaseRequest(
          dimensions=machine_lease.mp_dimensions,
          # TODO(smut): Vary duration so machines don't expire all at once.
          duration=machine_lease.lease_duration_secs,
          indefinite=machine_lease.lease_indefinitely,
          request_id=machine_lease.client_request_id,
      ),
  )

  if response.get('error'):
    _handle_lease_request_error(machine_lease, response)
    return

  _handle_lease_request_response(machine_lease, response)


## Public API.


def cleanup_bot(machine_lease):
  """Cleans up entities after a bot is removed."""
  bot_root_key = bot_management.get_root_key(machine_lease.hostname)
  # The bot is being removed, remove it from the task queues.
  task_queues.cleanup_after_bot(bot_root_key)
  bot_management.get_info_key(machine_lease.hostname).delete()
  _clear_lease_request(machine_lease.key, machine_lease.client_request_id)
  logging.info('MachineLease cleared:\nKey: %s', machine_lease.key)


def last_shutdown_ts(hostname):
  """Returns the time the given bot posted a final bot_shutdown event.

  The bot_shutdown event is only considered if it is the last recorded event.

  Args:
    hostname: Hostname of the machine.

  Returns:
    datetime.datetime or None if the last recorded event is not bot_shutdown.
  """
  bot_event = bot_management.get_events_query(hostname, True).get()
  if bot_event and bot_event.event_type == 'bot_shutdown':
    return bot_event.ts
  return None


def release(machine_lease):
  """Releases the given lease.

  Args:
    machine_lease: MachineLease instance.

  Returns:
    True if the lease was released, False otherwise.
  """
  response = machine_provider.release_machine(machine_lease.client_request_id)
  if response.get('error'):
    error = machine_provider.LeaseReleaseRequestError.lookup_by_name(
        response['error'])
    if error not in (
        machine_provider.LeaseReleaseRequestError.ALREADY_RECLAIMED,
        machine_provider.LeaseReleaseRequestError.NOT_FOUND,
    ):
      logging.error(
          'Lease release failed\nKey: %s\nRequest ID: %s\nError: %s',
          machine_lease.key,
          response['client_request_id'],
          response['error'],
      )
      return False
  logging.info(
      'MachineLease released:\nKey: %s\nHostname: %s',
      machine_lease.key,
      machine_lease.hostname,
  )
  return True


def task_manage_lease(key):
  """Manages a MachineLease.

  Args:
    key: ndb.Key for a MachineLease entity.
  """
  machine_lease = key.get()
  if not machine_lease:
    return

  # Manage a leased machine.
  if machine_lease.lease_expiration_ts or machine_lease.leased_indefinitely:
    _manage_leased_machine(machine_lease)
    return

  # Lease expiration time is unknown, so there must be no leased machine.
  assert not machine_lease.hostname, key
  assert not machine_lease.termination_task, key

  # Manage a pending lease request.
  if machine_lease.client_request_id:
    _manage_pending_lease_request(machine_lease)
    return

  # Manage an uninitiated lease request.
  if not machine_lease.drained:
    _update_client_request_id(key)
    return

  # Manage an uninitiated, drained lease request.
  _delete_machine_lease(key)


def cron_compute_utilization():
  """Computes bot utilization per machine type."""
  # A query that requires multiple batches may produce duplicate results. To
  # ensure each bot is only counted once, map machine types to [busy, idle]
  # sets of bots.
  machine_types = collections.defaultdict(lambda: [set(), set()])

  def process(bot):
    bot_id = bot.key.parent().id()
    if bot.task_id:
      machine_types[bot.machine_type][0].add(bot_id)
      machine_types[bot.machine_type][1].discard(bot_id)
    else:
      machine_types[bot.machine_type][0].discard(bot_id)
      machine_types[bot.machine_type][1].add(bot_id)

  # Expectation is ~2000 entities, so batching is valuable but local caching is
  # not. Can't use a projection query because 'cannot use projection on a
  # property with an equality filter'.
  now = utils.utcnow()
  q = bot_management.BotInfo.query()
  q = bot_management.filter_availability(
      q, quarantined=None, in_maintenance=None, is_dead=False, is_busy=None,
      is_mp=True)
  q.map(process, batch_size=128, use_cache=False)

  # The number of machine types isn't very large, in the few tens, so no need to
  # rate limit parallelism yet.
  futures = []
  total = 0
  for machine_type, (busy, idle) in machine_types.iteritems():
    total += 1
    busy = len(busy)
    idle = len(idle)
    logging.info('Utilization for %s: %s/%s', machine_type, busy, busy + idle)
    # TODO(maruel): This should be a single entity.
    # TODO(maruel): Historical data would be useful.
    obj = MachineTypeUtilization(
        id=machine_type, busy=busy, idle=idle, last_updated_ts=now)
    futures.append(obj.put_async())
  for f in futures:
    f.get_result()
  return total


def cron_schedule_lease_management():
  """Schedules task queues to process each MachineLease."""
  now = utils.utcnow()
  total = 0
  for machine_lease in MachineLease.query():
    # If there's no connection_ts, we're waiting on a bot so schedule the
    # management job to check on it. If there is a connection_ts, then don't
    # schedule the management job until it's time to release the machine.
    if machine_lease.connection_ts:
      if not machine_lease.drained:
        # lease_expiration_ts is None if leased_indefinitely, check it first.
        if machine_lease.leased_indefinitely:
          continue
        assert machine_lease.lease_expiration_ts, machine_lease
        if machine_lease.lease_expiration_ts > now + datetime.timedelta(
            seconds=machine_lease.early_release_secs or 0):
          continue
    total += 1
    if not utils.enqueue_task(
        '/internal/taskqueue/important/machine-provider/manage',
        'machine-provider-manage',
        params={'key': machine_lease.key.urlsafe()},
    ):
      logging.warning(
          'Failed to enqueue task for MachineLease: %s', machine_lease.key)
  return total


def cron_sync_config(server):
  """Updates MP related configuration."""
  if server:
    current_config = machine_provider.MachineProviderConfiguration().cached()
    if server != current_config.instance_url:
      logging.info('Updating Machine Provider server to %s', server)
      current_config.modify(updated_by='', instance_url=server)
  else:
    logging.info('No MP server specified')
  _ensure_entities_exist()
  return _drain_excess()


def set_global_metrics():
  """Set global Machine Provider-related ts_mon metrics."""
  # Consider utilization metrics over 2 minutes old to be outdated.
  outdated = utils.utcnow() - datetime.timedelta(minutes=2)
  payload = {}
  for machine_type in MachineType.query():
    data = {
      'enabled': machine_type.enabled,
      'target_size': machine_type.target_size,
    }
    utilization = ndb.Key(MachineTypeUtilization, machine_type.key.id()).get()
    if utilization and utilization.last_updated_ts > outdated:
      data['busy'] = utilization.busy
      data['idle'] = utilization.idle
    payload[machine_type.key.id()] = data
  ts_mon_metrics.set_global_metrics('mp', payload)
