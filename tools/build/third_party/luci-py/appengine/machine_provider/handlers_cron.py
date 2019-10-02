# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Cron jobs for processing lease requests."""

import datetime
import logging
import time

from google.appengine.api import taskqueue
from google.appengine.ext import ndb
from protorpc import messages
from protorpc.remote import protojson
import webapp2

from components import decorators
from components import utils
from components.machine_provider import rpc_messages

import metrics
import models


class Error(Exception):
  pass


class TaskEnqueuingError(Error):
  def __init__(self, queue_name):
    super(TaskEnqueuingError, self).__init__()
    self.queue_name = queue_name


def can_fulfill(entry, request):
  """Determines if the given CatalogEntry can fulfill the given LeaseRequest.

  Args:
    entry: A models.CatalogEntry instance.
    request: An rpc_messages.LeaseRequest instance.

  Returns:
    True if the given CatalogEntry can be used to fulfill the given
    LeaseRequest, otherwise False.
  """
  # For each dimension, check if the entry meets or exceeds the request.
  # For now, "exceeds" is defined as when the request leaves the dimension
  # unspecified/None, but the request has a non-None value. In the future,
  # "exceeds" may be defined per-dimension. E.g. in the future, an entry
  # with 16GB RAM may fulfill a request for 8GB RAM.
  for dimension in rpc_messages.Dimensions.all_fields():
    entry_value = entry.dimensions.get_assigned_value(dimension.name)
    request_value = request.dimensions.get_assigned_value(dimension.name)
    if request_value is not None:
      if isinstance(request_value, messages.FieldList):
        if request_value:
          # For a non-empty list, ensure every specified value matches.
          if set(request_value) - set(entry_value):
            return False
      elif entry_value != request_value:
        # There is a mismatched dimension, and the requested dimension was
        # not None, which means the entry does not fulfill the request.
        return False
  return True


def get_dimension_filters(request):
  """Returns filters to match the requested dimensions in a CatalogEntry query.

  Args:
    request: An rpc_messages.LeaseRequest instance.

  Returns:
    A list of filters for a CatalogEntry query.
  """
  # See can_fulfill for more information.
  filters = []
  for dimension in rpc_messages.Dimensions.all_fields():
    entry_value = getattr(models.CatalogEntry.dimensions, dimension.name)
    request_value = request.dimensions.get_assigned_value(dimension.name)
    # Ignore unspecified values.
    if request_value is not None:
      if isinstance(request_value, messages.FieldList):
       if request_value:
        # Match all specified list values.
        filters.append(ndb.AND(*[(entry_value == rv) for rv in request_value]))
      else:
        # Match specified value exactly.
        filters.append(entry_value == request_value)
  return filters


@ndb.transactional(xg=True)
def lease_machine(machine_key, lease):
  """Attempts to lease the given machine.

  Args:
    machine_key: ndb.Key for a model.CatalogMachineEntry instance.
    lease: model.LeaseRequest instance.

  Returns:
    True if the machine was leased, otherwise False.
  """
  machine = machine_key.get()
  lease = lease.key.get()
  logging.info('Attempting to lease matching CatalogMachineEntry:\n%s', machine)

  if not can_fulfill(machine, lease.request):
    logging.warning('CatalogMachineEntry no longer matches:\n%s', machine)
    return False
  if machine.state != models.CatalogMachineEntryStates.AVAILABLE:
    logging.warning('CatalogMachineEntry no longer available:\n%s', machine)
    return False
  if lease.response.state != rpc_messages.LeaseRequestState.UNTRIAGED:
    logging.warning('LeaseRequest no longer untriaged:\n%s', lease)
    return False

  logging.info('Leasing CatalogMachineEntry:\n%s', machine)
  lease.leased_ts = utils.utcnow()
  indefinite = False
  if lease.request.lease_expiration_ts:
    lease_expiration_ts = datetime.datetime.utcfromtimestamp(
        lease.request.lease_expiration_ts)
  elif lease.request.duration:
    lease_expiration_ts = lease.leased_ts + datetime.timedelta(
        seconds=lease.request.duration,
    )
  else:
    # Indefinite lease. Set dummy expiration date. This prevents having to
    # create a compound index which allows querying by lease_expiration_ts
    # and indefinite == False. To make the lease truly indefinite, when
    # processing a lease expiration, ensure it isn't indefinite.
    lease_expiration_ts = lease.leased_ts + datetime.timedelta(days=10000)
    indefinite = True
  lease.machine_id = machine.key.id()
  lease.response.hostname = machine.dimensions.hostname
  # datetime_to_timestamp returns microseconds, which are too fine grain.
  lease.response.lease_expiration_ts = utils.datetime_to_timestamp(
      lease_expiration_ts) / 1000 / 1000
  lease.response.leased_indefinitely = indefinite
  lease.response.state = rpc_messages.LeaseRequestState.FULFILLED
  machine.lease_id = lease.key.id()
  machine.lease_expiration_ts = lease_expiration_ts
  machine.leased_indefinitely = indefinite
  machine.state = models.CatalogMachineEntryStates.LEASED
  ndb.put_multi([lease, machine])
  params = {
      'policies': protojson.encode_message(machine.policies),
      'request_json': protojson.encode_message(lease.request),
      'response_json': protojson.encode_message(lease.response),
  }
  if not utils.enqueue_task(
      '/internal/queues/fulfill-lease-request',
      'fulfill-lease-request',
      params=params,
      transactional=True,
  ):
    raise TaskEnqueuingError('fulfill-lease-request')
  return True


@ndb.transactional
def deny_lease(lease_key, cutoff_ts):
  """Denies the given lease request if it's had no activity since the cutoff.

  Args:
    lease_key: ndb.Key for a models.LeaseRequest instance.
    cutoff_ts: datetime.datetime instance indicating the cutoff time.
  """
  lease = lease_key.get()
  if lease.response.state != rpc_messages.LeaseRequestState.UNTRIAGED:
    logging.warning('LeaseRequest no longer untriaged:\n%s', lease)
    return
  if lease.last_modified_ts >= cutoff_ts:
    logging.warning('LeaseRequest modified after cutoff:\n%s', lease)
    return
  logging.info('Denying lease request with no activity:\n%s', lease)
  lease.response.state = rpc_messages.LeaseRequestState.DENIED
  lease.put()


class LeaseRequestProcessor(webapp2.RequestHandler):
  """Worker for processing lease requests."""

  @decorators.require_cronjob
  def get(self):
    # The cutoff should be small enough to provide a quick answer to whether or
    # not a lease request can be fulfilled, but not so small that the request
    # gets denied while machines which could have fulfilled the request were
    # still being provisioned by one of the backend services. Adjust if needed.
    now = utils.utcnow()
    cutoff = now - datetime.timedelta(hours=4)

    for lease in models.LeaseRequest.query_untriaged():
      filters = get_dimension_filters(lease.request)
      for machine_key in models.CatalogMachineEntry.query_available(*filters):
        if lease_machine(machine_key, lease):
          metrics.lease_requests_fulfilled.increment()
          metrics.lease_requests_fulfilled_time.add(
              (now - lease.created_ts).total_seconds())
          break
      else:
        if lease.last_modified_ts < cutoff:
          deny_lease(lease.key, cutoff)


@ndb.transactional(xg=True)
def reclaim_machine(machine_key, reclamation_ts):
  """Attempts to reclaim the given machine.

  Args:
    machine_key: ndb.Key for a model.CatalogMachineEntry instance.
    reclamation_ts: datetime.datetime instance indicating when the machine was
      reclaimed.

  Returns:
    True if the machine was reclaimed, else False.
  """
  machine = machine_key.get()
  if not machine:
    logging.warning('CatalogMachineEntry not found: %s', machine_key)
    return

  logging.info('Attempting to reclaim CatalogMachineEntry:\n%s', machine)

  if machine.leased_indefinitely:
    # Extend the lease so we don't have to process this reclamation again for
    # awhile. This avoids creating a compound datastore index.
    logging.warning('CatalogMachineEntry leased indefinitely:\n%s', machine)
    machine.lease_expiration_ts += datetime.timedelta(days=10000)
    machine.put()
    return False

  if machine.lease_expiration_ts is None:
    # This can reasonably happen if e.g. the lease was voluntarily given up.
    logging.warning('CatalogMachineEntry no longer leased:\n%s', machine)
    return False

  if reclamation_ts < machine.lease_expiration_ts:
    # This can reasonably happen if e.g. the lease duration was extended.
    logging.warning('CatalogMachineEntry no longer overdue:\n%s', machine)
    return False

  logging.info('Reclaiming CatalogMachineEntry:\n%s', machine)
  lease = models.LeaseRequest.get_by_id(machine.lease_id)
  hostname = lease.response.hostname
  lease.response.hostname = None

  params = {
      'hostname': hostname,
      'machine_key': machine.key.urlsafe(),
      'policies': protojson.encode_message(machine.policies),
      'request_json': protojson.encode_message(lease.request),
      'response_json': protojson.encode_message(lease.response),
  }
  backend_attributes = {}
  for attribute in machine.policies.backend_attributes:
    backend_attributes[attribute.key] = attribute.value
  params['backend_attributes'] = utils.encode_to_json(backend_attributes)
  if lease.request.pubsub_topic:
    params['lessee_project'] = lease.request.pubsub_project
    params['lessee_topic'] = lease.request.pubsub_topic
  if not utils.enqueue_task(
      '/internal/queues/reclaim-machine',
      'reclaim-machine',
      params=params,
      transactional=True,
  ):
    raise TaskEnqueuingError('reclaim-machine')
  return True


class MachineReclamationProcessor(webapp2.RequestHandler):
  """Worker for processing machine reclamation."""

  @decorators.require_cronjob
  def get(self):
    min_ts = utils.timestamp_to_datetime(0)
    now = utils.utcnow()

    for machine_key in models.CatalogMachineEntry.query(
        models.CatalogMachineEntry.lease_expiration_ts < now,
        # Also filter out unassigned machines, i.e. CatalogMachineEntries
        # where lease_expiration_ts is None. None sorts before min_ts.
        models.CatalogMachineEntry.lease_expiration_ts > min_ts,
    ).fetch(keys_only=True):
      reclaim_machine(machine_key, now)


@ndb.transactional(xg=True)
def release_lease(lease_key):
  """Releases a lease on a machine.

  Args:
    lease_key: ndb.Key for a models.LeaseRequest entity.
  """
  lease = lease_key.get()
  if not lease:
    logging.warning('LeaseRequest not found: %s', lease_key)
    return
  if not lease.released:
    logging.warning('LeaseRequest not released:\n%s', lease)
    return

  lease.released = False
  if not lease.machine_id:
    logging.warning('LeaseRequest has no associated machine:\n%s', lease)
    lease.put()
    return

  machine = ndb.Key(models.CatalogMachineEntry, lease.machine_id).get()
  if not machine:
    logging.error('LeaseRequest has non-existent machine leased:\n%s', lease)
    lease.put()
    return

  # Just expire the lease now and let MachineReclamationProcessor handle it.
  logging.info('Expiring LeaseRequest:\n%s', lease)
  now = utils.utcnow()
  lease.response.lease_expiration_ts = utils.datetime_to_timestamp(
      now) / 1000 / 1000
  machine.lease_expiration_ts = now
  machine.leased_indefinitely = False
  ndb.put_multi([lease, machine])


class LeaseReleaseProcessor(webapp2.RequestHandler):
  """Worker for processing voluntary lease releases."""

  @decorators.require_cronjob
  def get(self):
    for lease_key in models.LeaseRequest.query(
        models.LeaseRequest.released == True,
    ).fetch(keys_only=True):
      release_lease(lease_key)


def create_cron_app():
  return webapp2.WSGIApplication([
      ('/internal/cron/process-lease-requests', LeaseRequestProcessor),
      ('/internal/cron/process-machine-reclamations',
       MachineReclamationProcessor),
      ('/internal/cron/process-lease-releases', LeaseReleaseProcessor),
  ])
