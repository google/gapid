# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Cloud endpoints for the Machine Provider API."""

import hashlib
import json
import logging

import endpoints
import webapp2

from google.appengine import runtime
from google.appengine.api import app_identity
from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from protorpc import message_types
from protorpc import protobuf
from protorpc import remote

import gae_ts_mon

from components import auth
from components import endpoints_webapp2
from components import pubsub
from components import utils
from components.machine_provider import rpc_messages

import acl
import config
import metrics
import models


PUBSUB_DEFAULT_PROJECT = app_identity.get_application_id()


@auth.endpoints_api(name='catalog', version='v1')
class CatalogEndpoints(remote.Service):
  """Implements cloud endpoints for the Machine Provider Catalog."""

  @staticmethod
  def check_backend(request):
    """Checks that the given catalog manipulation request specifies a backend.

    Returns:
      rpc_messages.CatalogManipulationRequestError.UNSPECIFIED_BACKEND if the
      backend is unspecified and can't be inferred, otherwise None.
    """
    if acl.is_catalog_admin():
      # Catalog administrators may update any CatalogEntry, but the backend must
      # be specified because hostname uniqueness is enforced per-backend.
      if not request.dimensions.backend:
        logging.warning('Backend unspecified by administrator')
        return rpc_messages.CatalogManipulationRequestError.UNSPECIFIED_BACKEND
    elif acl.is_backend_service():
      # Backends may only update their own machines.
      current_backend = acl.get_current_backend()
      if request.dimensions.backend is None:
        request.dimensions.backend = current_backend
      if request.dimensions.backend != current_backend:
        logging.warning('Mismatched backend')
        return rpc_messages.CatalogManipulationRequestError.MISMATCHED_BACKEND

  @staticmethod
  def check_hostname(request):
    """Checks that the given catalog manipulation request specifies a hostname.

    Returns:
      rpc_messages.CatalogManipulationRequestError.UNSPECIFIED_HOSTNAME if the
      hostname is unspecified, otherwise None.
    """
    if not request.dimensions.hostname:
      logging.warning('Hostname unspecified')
      return rpc_messages.CatalogManipulationRequestError.UNSPECIFIED_HOSTNAME

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.CatalogMachineRetrievalRequest,
      rpc_messages.CatalogMachineRetrievalResponse,
  )
  @auth.require(acl.is_backend_service_or_catalog_admin)
  def get(self, request):
    """Handles an incoming CatalogMachineRetrievalRequest."""
    user = auth.get_current_identity().to_bytes()
    logging.info(
        'Received CatalogMachineRetrievalRequest:\nUser: %s\n%s',
        user,
        request,
    )
    if acl.is_catalog_admin():
      if not request.backend:
        raise endpoints.BadRequestException(
            'Backend unspecified by administrator')
    elif acl.is_backend_service():
      current_backend = acl.get_current_backend()
      if request.backend is None:
        request.backend = current_backend
      if request.backend != current_backend:
        raise endpoints.ForbiddenException('Mismatched backend')

    entry = models.CatalogMachineEntry.get(request.backend, request.hostname)
    if not entry:
      raise endpoints.NotFoundException('CatalogMachineEntry not found')

    response = rpc_messages.CatalogMachineRetrievalResponse(
        dimensions=entry.dimensions,
        policies=entry.policies,
        state=entry.state,
    )
    if entry.leased_indefinitely:
      response.leased_indefinitely = True
    elif entry.lease_expiration_ts:
      # datetime_to_timestamp returns microseconds, convert to seconds.
      response.lease_expiration_ts = utils.datetime_to_timestamp(
          entry.lease_expiration_ts) / 1000 / 1000
    return response

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.CatalogMachineAdditionRequest,
      rpc_messages.CatalogManipulationResponse,
  )
  @auth.require(acl.is_backend_service_or_catalog_admin)
  def add_machine(self, request):
    """Handles an incoming CatalogMachineAdditionRequest."""
    user = auth.get_current_identity().to_bytes()
    logging.info(
        'Received CatalogMachineAdditionRequest:\nUser: %s\n%s',
        user,
        request,
    )
    error = self.check_backend(request) or self.check_hostname(request)
    if error:
      return rpc_messages.CatalogManipulationResponse(
          error=error,
          machine_addition_request=request,
      )

    return self._add_machine(request)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.CatalogMachineBatchAdditionRequest,
      rpc_messages.CatalogBatchManipulationResponse,
  )
  @auth.require(acl.is_backend_service_or_catalog_admin)
  def add_machines(self, request):
    """Handles an incoming CatalogMachineBatchAdditionRequest.

    Batches are intended to save on RPCs only. The batched requests will not
    execute transactionally.
    """
    user = auth.get_current_identity().to_bytes()
    logging.info(
        'Received CatalogMachineBatchAdditionRequest:\nUser: %s\n%s',
        user,
        request,
    )
    responses = []
    for request in request.requests:
      logging.info(
          'Processing CatalogMachineAdditionRequest:\n%s',
          request,
      )
      error = self.check_backend(request) or self.check_hostname(request)
      if error:
        responses.append(rpc_messages.CatalogManipulationResponse(
            error=error,
            machine_addition_request=request,
        ))
      else:
        responses.append(self._add_machine(request))
    return rpc_messages.CatalogBatchManipulationResponse(responses=responses)

  @ndb.transactional
  def _add_machine(self, request):
    """Handles datastore operations for CatalogMachineAdditionRequests."""
    entry = models.CatalogMachineEntry.generate_key(request.dimensions).get()
    if entry:
      # Enforces per-backend hostname uniqueness.
      logging.warning('Hostname reuse:\nOriginally used for: \n%s', entry)
      return rpc_messages.CatalogManipulationResponse(
          error=rpc_messages.CatalogManipulationRequestError.HOSTNAME_REUSE,
          machine_addition_request=request,
      )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(request.dimensions),
        dimensions=request.dimensions,
        policies=request.policies,
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()
    return rpc_messages.CatalogManipulationResponse(
        machine_addition_request=request,
    )

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.CatalogMachineDeletionRequest,
      rpc_messages.CatalogManipulationResponse,
  )
  @auth.require(acl.is_backend_service_or_catalog_admin)
  def delete_machine(self, request):
    """Handles an incoming CatalogMachineDeletionRequest."""
    user = auth.get_current_identity().to_bytes()
    logging.info(
        'Received CatalogMachineDeletionRequest:\nUser: %s\n%s',
        user,
        request,
    )
    error = self.check_backend(request) or self.check_hostname(request)
    if error:
      return rpc_messages.CatalogManipulationResponse(
          error=error,
          machine_deletion_request=request,
      )
    return self._delete_machine(request)

  @ndb.transactional
  def _delete_machine(self, request):
    """Handles datastore operations for CatalogMachineDeletionRequests."""
    entry = models.CatalogMachineEntry.generate_key(request.dimensions).get()
    if not entry:
      logging.info('Catalog entry not found')
      return rpc_messages.CatalogManipulationResponse(
          error=rpc_messages.CatalogManipulationRequestError.ENTRY_NOT_FOUND,
          machine_deletion_request=request,
      )
    if entry.lease_id:
      logging.warning('Attempting to delete leased machine: %s', entry)
      return rpc_messages.CatalogManipulationResponse(
          error=rpc_messages.CatalogManipulationRequestError.LEASED,
          machine_deletion_request=request,
      )
    entry.key.delete()
    return rpc_messages.CatalogManipulationResponse(
        machine_deletion_request=request,
    )


@auth.endpoints_api(name='machine', version='v1')
class MachineEndpoints(remote.Service):
  """Implements cloud endpoints for Machine Provider's machines."""

  # The endpoints only allow a subset of possible instruction state
  # transitions. poll only allows PENDING -> RECEIVED, and the (unimplemented)
  # ack only allows * -> EXECUTED.
  ALLOWED_TRANSITIONS = (
      (models.InstructionStates.PENDING, models.InstructionStates.RECEIVED),
      (models.InstructionStates.PENDING, models.InstructionStates.EXECUTED),
      (models.InstructionStates.RECEIVED, models.InstructionStates.EXECUTED),
  )

  @staticmethod
  @ndb.transactional
  def _update_instruction_state(machine_key, new_state):
    """Updates the state of the instruction for the given machine.

    The only updates allowed are:
      PENDING -> RECEIVED
      PENDING -> EXECUTED
      RECEIVED -> EXECUTED

    Args:
      machine_key: ndb.Key for a models.CatalogMachineEntry.
      new_state: One of models.InstructionStates, but not
        models.InstructionStates.PENDING.
    """
    machine = machine_key.get()
    if not machine:
      raise endpoints.NotFoundException('CatalogMachineEntry not found')

    if not machine.instruction or machine.instruction.state == new_state:
      return

    transition = (machine.instruction.state, new_state)
    if transition not in MachineEndpoints.ALLOWED_TRANSITIONS:
      logging.error(
          'Invalid instruction state transition (%s -> %s): %s',
          machine.instruction.state,
          new_state,
          machine_key,
      )
      return

    logging.info(
        'Updating instruction state (%s -> %s): %s',
        machine.instruction.state,
        new_state,
        machine_key,
    )
    machine.instruction.state = new_state
    machine.put()

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(rpc_messages.PollRequest, rpc_messages.PollResponse)
  @auth.require(acl.is_logged_in)
  def poll(self, request):
    """Handles an incoming PollRequest."""
    user = auth.get_current_identity()
    if not request.backend:
      if acl.is_backend_service():
        # Backends may omit this field to mean backend == self.
        request.backend = acl.get_current_backend()
      else:
        # Anyone else omitting the backend field is an error.
        raise endpoints.BadRequestException('Backend unspecified')

    entry = models.CatalogMachineEntry.get(request.backend, request.hostname)
    if not entry:
      raise endpoints.NotFoundException('CatalogMachineEntry not found')

    # Determine authorization. User must be the known service account for the
    # machine, the backend service that owns the machine, or an administrator.
    machine_service_account = None
    if entry.policies:
      machine_service_account = entry.policies.machine_service_account
    if user.name != machine_service_account:
      if entry.dimensions.backend != acl.get_current_backend():
        if not acl.is_catalog_admin():
          # It's found, but raise 404 so we don't reveal the existence of
          # machines to unauthorized users.
          logging.warning(
              'Unauthorized poll\nUser: %s\nMachine: %s',
              user.to_bytes(),
              entry,
          )
          raise endpoints.NotFoundException('CatalogMachineEntry not found')

    # Authorized request, return the current instruction for the machine.
    if not entry.lease_id or not entry.instruction:
      return rpc_messages.PollResponse()

    # The cron job which processes expired leases may not have run yet. Check
    # the lease expiration time to make sure this lease is still active.
    if entry.lease_expiration_ts <= utils.utcnow():
      return rpc_messages.PollResponse()

    # The cron job which processes early lease releases may not have run yet.
    # Check the LeaseRequest to make sure this lease is still active.
    lease = models.LeaseRequest.get_by_id(entry.lease_id)
    if not lease or lease.released:
      return rpc_messages.PollResponse()

    # Only the machine itself polling for its own instructions causes
    # the state of the instruction to be updated. Only update the state
    # if it's PENDING.
    if entry.instruction.state == models.InstructionStates.PENDING:
      if user.name == machine_service_account:
        self._update_instruction_state(
            entry.key, models.InstructionStates.RECEIVED)

    return rpc_messages.PollResponse(
        instruction=entry.instruction.instruction,
        state=entry.instruction.state,
    )

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(rpc_messages.AckRequest)
  @auth.require(acl.is_logged_in)
  def ack(self, request):
    """Handles an incoming AckRequest."""
    user = auth.get_current_identity()
    entry = models.CatalogMachineEntry.get(request.backend, request.hostname)
    if (not entry or not entry.policies
        or entry.policies.machine_service_account != user.name):
      raise endpoints.NotFoundException('CatalogMachineEntry not found')

    if not entry.lease_id or not entry.instruction:
      raise endpoints.BadRequestException('Machine has no instruction')

    if entry.instruction.state != models.InstructionStates.EXECUTED:
      self._update_instruction_state(
          entry.key, models.InstructionStates.EXECUTED)

    return message_types.VoidMessage()


@auth.endpoints_api(name='machine_provider', version='v1')
class MachineProviderEndpoints(remote.Service):
  """Implements cloud endpoints for the Machine Provider."""

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.BatchedLeaseRequest,
      rpc_messages.BatchedLeaseResponse,
  )
  @auth.require(acl.can_issue_lease_requests)
  def batched_lease(self, request):
    """Handles an incoming BatchedLeaseRequest.

    Batches are intended to save on RPCs only. The batched requests will not
    execute transactionally.
    """
    # To avoid having large batches timed out by AppEngine after 60 seconds
    # when some requests have been processed and others haven't, enforce a
    # smaller deadline on ourselves to process the entire batch.
    DEADLINE_SECS = 30
    start_time = utils.utcnow()
    user = auth.get_current_identity().to_bytes()
    logging.info('Received BatchedLeaseRequest:\nUser: %s\n%s', user, request)
    responses = []
    for request in request.requests:
      request_hash = models.LeaseRequest.generate_key(user, request).id()
      logging.info(
          'Processing LeaseRequest:\nRequest hash: %s\n%s',
          request_hash,
          request,
      )
      if (utils.utcnow() - start_time).seconds > DEADLINE_SECS:
        logging.warning(
          'BatchedLeaseRequest exceeded enforced deadline: %s', DEADLINE_SECS)
        responses.append(rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.DEADLINE_EXCEEDED,
            request_hash=request_hash,
        ))
      else:
        try:
          responses.append(self._lease(request, request_hash))
        except (
            datastore_errors.Timeout,
            runtime.apiproxy_errors.CancelledError,
            runtime.apiproxy_errors.DeadlineExceededError,
            runtime.apiproxy_errors.OverQuotaError,
        ) as e:
          logging.warning('Exception processing LeaseRequest:\n%s', e)
          responses.append(rpc_messages.LeaseResponse(
              client_request_id=request.request_id,
              error=rpc_messages.LeaseRequestError.TRANSIENT_ERROR,
              request_hash=request_hash,
          ))
    return rpc_messages.BatchedLeaseResponse(responses=responses)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(rpc_messages.LeaseRequest, rpc_messages.LeaseResponse)
  @auth.require(acl.can_issue_lease_requests)
  def lease(self, request):
    """Handles an incoming LeaseRequest."""
    # Hash the combination of client + client-generated request ID in order to
    # deduplicate responses on a per-client basis.
    user = auth.get_current_identity().to_bytes()
    request_hash = models.LeaseRequest.generate_key(user, request).id()
    logging.info(
        'Received LeaseRequest:\nUser: %s\nRequest hash: %s\n%s',
        user,
        request_hash,
        request,
    )
    return self._lease(request, request_hash)

  def _lease(self, request, request_hash):
    """Handles an incoming LeaseRequest."""
    # Arbitrary limit of 3 weeks. Increase if necessary.
    MAX_LEASE_DURATION_SECS = 60 * 60 * 24 * 21
    now = utils.time_time()
    max_lease_expiration_ts = now + MAX_LEASE_DURATION_SECS

    metrics.lease_requests_received.increment()
    if request.duration:
      if request.lease_expiration_ts or request.indefinite:
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.MUTUAL_EXCLUSION_ERROR,
        )
      if request.duration < 1:
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.NONPOSITIVE_DEADLINE,
        )
      if request.duration > MAX_LEASE_DURATION_SECS:
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.LEASE_TOO_LONG,
        )
    elif request.lease_expiration_ts:
      if request.indefinite:
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.MUTUAL_EXCLUSION_ERROR,
        )
      if request.lease_expiration_ts <= now:
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.LEASE_EXPIRATION_TS_ERROR,
        )
      if request.lease_expiration_ts > max_lease_expiration_ts:
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.LEASE_TOO_LONG,
        )
    elif not request.indefinite:
      return rpc_messages.LeaseResponse(
          client_request_id=request.request_id,
          error=rpc_messages.LeaseRequestError.LEASE_LENGTH_UNSPECIFIED,
      )
    if request.pubsub_topic:
      if not pubsub.validate_topic(request.pubsub_topic):
        logging.warning(
            'Invalid topic for Cloud Pub/Sub: %s',
            request.pubsub_topic,
        )
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.INVALID_TOPIC,
        )
      if not request.pubsub_project:
        logging.info(
            'Cloud Pub/Sub project unspecified, using default: %s',
            PUBSUB_DEFAULT_PROJECT,
        )
        request.pubsub_project = PUBSUB_DEFAULT_PROJECT
    if request.pubsub_project:
      if not pubsub.validate_project(request.pubsub_project):
        logging.warning(
            'Invalid project for Cloud Pub/Sub: %s',
            request.pubsub_topic,
        )
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.INVALID_PROJECT,
        )
      elif not request.pubsub_topic:
        logging.warning(
            'Cloud Pub/Sub project specified without specifying topic: %s',
            request.pubsub_project,
        )
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.UNSPECIFIED_TOPIC,
        )
    duplicate = models.LeaseRequest.get_by_id(request_hash)
    deduplication_checksum = models.LeaseRequest.compute_deduplication_checksum(
        request,
    )
    if duplicate:
      # Found a duplicate request ID from the same user. Attempt deduplication.
      if deduplication_checksum == duplicate.deduplication_checksum:
        metrics.lease_requests_deduped.increment()
        # The LeaseRequest RPC we just received matches the original.
        # We're safe to dedupe.
        logging.info(
            'Dropped duplicate LeaseRequest:\n%s', duplicate.response,
        )
        return duplicate.response
      else:
        logging.warning(
            'Request ID reuse:\nOriginally used for:\n%s',
            duplicate.request
        )
        return rpc_messages.LeaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseRequestError.REQUEST_ID_REUSE,
        )
    else:
      logging.info('Storing LeaseRequest')
      response = rpc_messages.LeaseResponse(
          client_request_id=request.request_id,
          request_hash=request_hash,
          state=rpc_messages.LeaseRequestState.UNTRIAGED,
      )
      models.LeaseRequest(
          deduplication_checksum=deduplication_checksum,
          id=request_hash,
          owner=auth.get_current_identity(),
          request=request,
          response=response,
      ).put()
      logging.info('Sending LeaseResponse:\n%s', response)
      return response

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.BatchedLeaseReleaseRequest,
      rpc_messages.BatchedLeaseReleaseResponse,
  )
  @auth.require(acl.can_issue_lease_requests)
  def batched_release(self, request):
    """Handles an incoming BatchedLeaseReleaseRequest.

    Batches are intended to save on RPCs only. The batched requests will not
    execute transactionally.
    """
    # TODO(smut): Dedupe common logic in batched RPC handling.
    DEADLINE_SECS = 30
    start_time = utils.utcnow()
    user = auth.get_current_identity().to_bytes()
    logging.info(
        'Received BatchedLeaseReleaseRequest:\nUser: %s\n%s', user, request)
    responses = []
    for request in request.requests:
      request_hash = models.LeaseRequest.generate_key(user, request).id()
      logging.info(
          'Processing LeaseReleaseRequest:\nRequest hash: %s\n%s',
          request_hash,
          request,
      )
      if (utils.utcnow() - start_time).seconds > DEADLINE_SECS:
        logging.warning(
          'BatchedLeaseReleaseRequest exceeded enforced deadline: %s',
          DEADLINE_SECS,
        )
        responses.append(rpc_messages.LeaseReleaseResponse(
            client_request_id=request.request_id,
            error=rpc_messages.LeaseReleaseRequestError.DEADLINE_EXCEEDED,
            request_hash=request_hash,
        ))
      else:
        try:
          responses.append(rpc_messages.LeaseReleaseResponse(
              client_request_id=request.request_id,
              error=self._release(request_hash),
              request_hash=request_hash,
          ))
        except (
            datastore_errors.Timeout,
            runtime.apiproxy_errors.CancelledError,
            runtime.apiproxy_errors.DeadlineExceededError,
            runtime.apiproxy_errors.OverQuotaError,
        ) as e:
          logging.warning('Exception processing LeaseReleaseRequest:\n%s', e)
          responses.append(rpc_messages.LeaseReleaseResponse(
              client_request_id=request.request_id,
              error=rpc_messages.LeaseReleaseRequestError.TRANSIENT_ERROR,
              request_hash=request_hash,
          ))
    return rpc_messages.BatchedLeaseReleaseResponse(responses=responses)

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.LeaseReleaseRequest, rpc_messages.LeaseReleaseResponse)
  @auth.require(acl.can_issue_lease_requests)
  def release(self, request):
    """Handles an incoming LeaseReleaseRequest."""
    user = auth.get_current_identity().to_bytes()
    request_hash = models.LeaseRequest.generate_key(user, request).id()
    logging.info(
        'Received LeaseReleaseRequest:\nUser: %s\nLeaseRequest: %s\n%s',
        user,
        request_hash,
        request,
    )
    return rpc_messages.LeaseReleaseResponse(
        client_request_id=request.request_id,
        error=self._release(request_hash),
        request_hash=request_hash,
    )

  @staticmethod
  @ndb.transactional
  def _release(request_hash):
    """Releases a LeaseRequest.

    Args:
      request_hash: ID of a models.LeaseRequest entity in the datastore to
        release.

    Returns:
      rpc_messages.LeaseReleaseRequestError indicating an error that occurred,
      or None if there was no error and the lease was released successfully.
    """
    request = ndb.Key(models.LeaseRequest, request_hash).get()
    if not request:
      logging.warning(
          'LeaseReleaseRequest referred to non-existent LeaseRequest: %s',
          request_hash,
      )
      return rpc_messages.LeaseReleaseRequestError.NOT_FOUND
    if request.response.state != rpc_messages.LeaseRequestState.FULFILLED:
      logging.warning(
          'LeaseReleaseRequest referred to unfulfilled LeaseRequest: %s',
          request_hash,
      )
      return rpc_messages.LeaseReleaseRequestError.NOT_FULFILLED
      # TODO(smut): Cancel the request.
    if not request.machine_id:
      logging.warning(
          'LeaseReleaseRequest referred to already reclaimed LeaseRequest: %s',
          request_hash,
      )
      return rpc_messages.LeaseReleaseRequestError.ALREADY_RECLAIMED
    logging.info('Releasing LeaseRequest: %s', request_hash)
    request.released = True
    request.put()

  @gae_ts_mon.instrument_endpoint()
  @auth.endpoints_method(
      rpc_messages.MachineInstructionRequest,
      rpc_messages.MachineInstructionResponse,
  )
  @auth.require(acl.can_issue_lease_requests)
  def instruct(self, request):
    """Handles an incoming MachineInstructionRequest."""
    user = auth.get_current_identity().to_bytes()
    if not request.instruction or not request.instruction.swarming_server:
      # For now only one type of instruction is supported.
      return rpc_messages.MachineInstructionResponse(
          client_request_id=request.request_id,
          error=rpc_messages.MachineInstructionError.INVALID_INSTRUCTION,
      )
    request_hash = models.LeaseRequest.generate_key(user, request).id()

    lease = models.LeaseRequest.get_by_id(request_hash)
    if not lease:
      raise endpoints.NotFoundException('LeaseRequest not found')
    if lease.response.state != rpc_messages.LeaseRequestState.FULFILLED:
      return rpc_messages.MachineInstructionResponse(
          client_request_id=request.request_id,
          error=rpc_messages.MachineInstructionError.NOT_FULFILLED,
      )
    if not lease.response.hostname:
      return rpc_messages.MachineInstructionResponse(
          client_request_id=request.request_id,
          error=rpc_messages.MachineInstructionError.ALREADY_RECLAIMED,
      )

    machine = models.CatalogMachineEntry.get_by_id(lease.machine_id)
    if not machine:
      raise endpoints.NotFoundException('CatalogMachineEntry not found')
    if machine.lease_id != request_hash:
      return rpc_messages.MachineInstructionResponse(
          client_request_id=request.request_id,
          error=rpc_messages.MachineInstructionError.NOT_FULFILLED,
      )
    if machine.lease_expiration_ts <= utils.utcnow():
      return rpc_messages.MachineInstructionResponse(
          client_request_id=request.request_id,
          error=rpc_messages.MachineInstructionError.ALREADY_RECLAIMED,
      )

    self._instruct(machine.key, request.instruction)

    return rpc_messages.MachineInstructionResponse(
        client_request_id=request.request_id)

  @staticmethod
  @ndb.transactional
  def _instruct(machine_key, instruction):
    """Stores an instruction for a machine."""
    # Probably not worth double-checking that the machine is still leased
    # because the CatalogMachineEntry will be deleted at the end of the lease.
    # If the entity still exists then it's likely still leased.
    machine = machine_key.get()
    if not machine:
      raise endpoints.NotFoundException('CatalogMachineEntry not found')

    # Currently there is only one type of instruction, an instruction to connect
    # to a Swarming server. Just overwrite any previous instruction.
    machine.instruction = models.Instruction(
        instruction=instruction, state=models.InstructionStates.PENDING)
    machine.put()


def create_endpoints_app():
  return endpoints_webapp2.api_server([
      CatalogEndpoints,
      MachineEndpoints,
      MachineProviderEndpoints,
      config.ConfigApi,
  ])
