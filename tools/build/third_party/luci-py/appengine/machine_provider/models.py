# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Datastore models for the Machine Provider messages."""

import hashlib

from google.appengine.ext import ndb
from google.appengine.ext.ndb import msgprop

from protorpc import messages
from protorpc import protobuf
from protorpc.remote import protojson

from components import auth
from components.machine_provider import rpc_messages


class Enum(frozenset):
  def __getattr__(self, attr):
    if attr in self:
      return attr
    raise AttributeError(attr)


class LeaseRequest(ndb.Model):
  """Datastore representation of a LeaseRequest.

  Key:
    id: Hash of the client + client-generated request ID which issued the
      original rpc_messages.LeaseRequest instance. Used for easy deduplication.
    kind: LeaseRequest. This root entity does not reference any parents.
  """
  # DateTime indicating original datastore write time.
  created_ts = ndb.DateTimeProperty(auto_now_add=True)
  # Checksum of the rpc_messages.LeaseRequest instance. Used to compare incoming
  # LeaseRequets for deduplication.
  deduplication_checksum = ndb.StringProperty(required=True, indexed=False)
  # DateTime indicating the last modification time.
  last_modified_ts = ndb.DateTimeProperty(auto_now=True)
  # ID of the CatalogMachineEntry provided for this lease.
  machine_id = ndb.StringProperty()
  # auth.model.Identity of the issuer of the original request.
  owner = auth.IdentityProperty(required=True)
  # Whether this lease request has been released voluntarily by the owner.
  released = ndb.BooleanProperty()
  # rpc_messages.LeaseRequest instance representing the original request.
  request = msgprop.MessageProperty(rpc_messages.LeaseRequest, required=True)
  # rpc_messages.LeaseResponse instance representing the current response.
  # This field will be updated as the request is being processed.
  response = msgprop.MessageProperty(
      rpc_messages.LeaseResponse, indexed_fields=['state'])

  @classmethod
  def compute_deduplication_checksum(cls, request):
    """Computes the deduplication checksum for the given request.

    Args:
      request: The rpc_messages.LeaseRequest instance to deduplicate.

    Returns:
      The deduplication checksum.
    """
    return hashlib.sha1(protobuf.encode_message(request)).hexdigest()

  @classmethod
  def generate_key(cls, user, request):
    """Generates the key for the given request initiated by the given user.

    Args:
      user: An auth.model.Identity instance representing the requester.
      request: The rpc_messages.LeaseRequest sent by the user.

    Returns:
      An ndb.Key instance.
    """
    # Enforces per-user request ID uniqueness
    return ndb.Key(
        cls,
        hashlib.sha1('%s\0%s' % (user, request.request_id)).hexdigest(),
    )

  @classmethod
  def query_untriaged(cls):
    """Queries for untriaged LeaseRequests.

    Yields:
      Untriaged LeaseRequests in no guaranteed order.
    """
    for request in cls.query(
        cls.response.state == rpc_messages.LeaseRequestState.UNTRIAGED
    ):
      yield request


InstructionStates = Enum(['PENDING', 'RECEIVED', 'EXECUTED'])


class Instruction(ndb.Model):
  """Datastore representation of an instruction for a machine.

  Standalone instances should not be present in the datastore.
  """
  # Instruction to execute.
  instruction = msgprop.MessageProperty(rpc_messages.Instruction)
  # State of the instruction.
  state = ndb.StringProperty(choices=InstructionStates)


class CatalogEntry(ndb.Model):
  """Datastore representation of an entry in the catalog."""
  # rpc_messages.Dimensions describing this machine.
  dimensions = msgprop.MessageProperty(
      rpc_messages.Dimensions,
      indexed_fields=[
          field.name for field in rpc_messages.Dimensions.all_fields()
      ],
  )
  # DateTime indicating the last modified time.
  last_modified_ts = ndb.DateTimeProperty(auto_now=True)


CatalogMachineEntryStates = Enum(['AVAILABLE', 'LEASED', 'NEW'])


class CatalogMachineEntry(CatalogEntry):
  """Datastore representation of a machine in the catalog.

  Key:
    id: Hash of the backend + hostname dimensions. Used to enforce per-backend
      hostname uniqueness.
    kind: CatalogMachineEntry. This root entity does not reference any parents.
  """
  # Instruction for this machine.
  instruction = ndb.LocalStructuredProperty(Instruction)
  # ID of the LeaseRequest this machine is provided for.
  lease_id = ndb.StringProperty()
  # DateTime indicating lease expiration time.
  lease_expiration_ts = ndb.DateTimeProperty()
  # Indicates whether this machine is leased indefinitely.
  # Supersedes lease_expiration_ts.
  leased_indefinitely = ndb.BooleanProperty()
  # rpc_messages.Policies governing this machine.
  policies = msgprop.MessageProperty(rpc_messages.Policies)
  # Determines sorted order relative to other CatalogMachineEntries.
  sort_ordering = ndb.ComputedProperty(lambda self: '%s:%s' % (
      self.dimensions.backend, self.dimensions.hostname))
  # Element of CatalogMachineEntryStates giving the state of this entry.
  state = ndb.StringProperty(
      choices=CatalogMachineEntryStates,
      default=CatalogMachineEntryStates.AVAILABLE,
      indexed=True,
      required=True,
  )

  @classmethod
  def generate_key(cls, dimensions):
    """Generates the key for a CatalogEntry with the given dimensions.

    Args:
      dimensions: rpc_messages.Dimensions describing this machine.

    Returns:
      An ndb.Key instance.
    """
    # Enforces per-backend hostname uniqueness.
    assert dimensions.backend is not None
    assert dimensions.hostname is not None
    return cls._generate_key(dimensions.backend, dimensions.hostname)

  @classmethod
  def _generate_key(cls, backend, hostname):
    """Generates the key for a CatalogEntry with the given backend and hostname.

    Args:
      backend: rpc_messages.Backend.
      hostname: Hostname of the machine.

    Returns:
      An ndb.Key instance.
    """
    return ndb.Key(
        cls,
        hashlib.sha1('%s\0%s' % (backend, hostname)).hexdigest(),
    )

  @classmethod
  def get(cls, backend, hostname):
    """Gets the CatalogEntry with by backend and hostname.

    Args:
      backend: rpc_messages.Backend.
      hostname: Hostname of the machine.

    Returns:
      An ndb.Key instance.
    """
    return cls._generate_key(backend, hostname).get()

  @classmethod
  def query_available(cls, *filters):
    """Queries for available machines.

    Args:
      *filters: Any additional filters to include in the query.

    Yields:
      CatalogMachineEntry keys in no guaranteed order.
    """
    available = cls.state == CatalogMachineEntryStates.AVAILABLE
    for machine in cls.query(available, *filters).fetch(keys_only=True):
      yield machine
