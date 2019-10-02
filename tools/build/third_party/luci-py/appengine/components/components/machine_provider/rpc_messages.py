# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Messages for the Machine Provider API."""

# pylint: disable=unused-wildcard-import, wildcard-import

from protorpc import messages

from components.machine_provider.dimensions import *
from components.machine_provider.instructions import *
from components.machine_provider.policies import *


class CatalogMachineRetrievalRequest(messages.Message):
  """Represents a request to retrieve a machine from the catalog."""
  # Hostname of the machine to retrieve.
  hostname = messages.StringField(1, required=True)
  # Backend which added the machine.
  backend = messages.EnumField(Backend, 2)


class CatalogMachineRetrievalResponse(messages.Message):
  """Represents a response to a catalog machine retrieval request."""
  # Dimensions instance specifying what sort of machine this is.
  dimensions = messages.MessageField(Dimensions, 1)
  # Policies governing this machine.
  policies = messages.MessageField(Policies, 2)
  # State of the CatalogMachineEntry.
  state = messages.StringField(3)
  # Cloud Pub/Sub subscription the machine must listen to for instructions.
  pubsub_subscription = messages.StringField(4)
  # Project the Cloud Pub/Sub subscription exists in.
  pubsub_subscription_project = messages.StringField(5)
  # Cloud Pub/Sub topic the machine must be subscribed to.
  pubsub_topic = messages.StringField(6)
  # Project the Cloud Pub/Sub topic exists in.
  pubsub_topic_project = messages.StringField(7)
  # Timestamp indicating lease expiration seconds from epoch in UTC.
  lease_expiration_ts = messages.IntegerField(8)
  # Whether or not this machine is leased indefinitely.
  # If true, disregard lease_expiration_ts. This lease will not expire.
  leased_indefinitely = messages.BooleanField(9)


class CatalogMachineAdditionRequest(messages.Message):
  """Represents a request to add a machine to the catalog.

  dimensions.backend must be specified.
  dimensions.hostname must be unique per backend.
  """
  # Dimensions instance specifying what sort of machine this is.
  dimensions = messages.MessageField(Dimensions, 1, required=True)
  # Policies instance specifying machine-specific configuration.
  policies = messages.MessageField(Policies, 2, required=True)


class CatalogMachineBatchAdditionRequest(messages.Message):
  """Represents a batched set of CatalogMachineAdditionRequests.

  dimensions.backend must be specified in each CatalogMachineAdditionRequest.
  dimensions.hostname must be unique per backend.
  """
  # CatalogMachineAdditionRequest instances to batch together.
  requests = messages.MessageField(
      CatalogMachineAdditionRequest, 1, repeated=True)


class CatalogMachineDeletionRequest(messages.Message):
  """Represents a request to delete a machine in the catalog."""
  # Dimensions instance specifying what sort of machine this is.
  dimensions = messages.MessageField(Dimensions, 1, required=True)


class CatalogManipulationRequestError(messages.Enum):
  """Represents an error in a catalog manipulation request."""
  # Per backend, hostnames must be unique in the catalog.
  HOSTNAME_REUSE = 1
  # Tried to lookup an entry that didn't exist.
  ENTRY_NOT_FOUND = 2
  # Didn't specify a backend.
  UNSPECIFIED_BACKEND = 3
  # Specified backend didn't match the backend originating the request.
  MISMATCHED_BACKEND = 4
  # Didn't specify a hostname.
  UNSPECIFIED_HOSTNAME = 5
  # Proposed Cloud Pub/Sub topic was invalid.
  INVALID_TOPIC = 6
  # Proposed Cloud Pub/Sub project was invalid.
  INVALID_PROJECT = 7
  # Didn't specify a Cloud Pub/Sub topic.
  UNSPECIFIED_TOPIC = 8
  # Attempted to delete a leased machine.
  LEASED = 9


class CatalogManipulationResponse(messages.Message):
  """Represents a response to a catalog manipulation request."""
  # CatalogManipulationRequestError instance indicating an error with the
  # request, or None if there is no error.
  error = messages.EnumField(CatalogManipulationRequestError, 1)
  # CatalogMachineAdditionRequest this response is in reference to.
  machine_addition_request = messages.MessageField(
      CatalogMachineAdditionRequest, 2)
  # CatalogMachineDeletionRequest this response is in reference to.
  machine_deletion_request = messages.MessageField(
      CatalogMachineDeletionRequest, 3)


class CatalogBatchManipulationResponse(messages.Message):
  """Represents a response to a batched catalog manipulation request."""
  responses = messages.MessageField(
      CatalogManipulationResponse, 1, repeated=True)


class LeaseRequest(messages.Message):
  """Represents a request for a lease on a machine."""
  # Per-user unique ID used to deduplicate requests.
  request_id = messages.StringField(1, required=True)
  # Dimensions instance specifying what sort of machine to lease.
  dimensions = messages.MessageField(Dimensions, 2, required=True)
  # Desired length of the lease in seconds.
  # Specify one of duration, lease_expiration_ts, or indefinite.
  duration = messages.IntegerField(3)
  # Cloud Pub/Sub topic name to communicate on regarding this request.
  pubsub_topic = messages.StringField(4)
  # Cloud Pub/Sub project name to communicate on regarding this request.
  pubsub_project = messages.StringField(5)
  # Instructions to give the machine once it's been leased.
  on_lease = messages.MessageField(Instruction, 6)
  # UTC seconds from epoch when lease should expire.
  # Specify one of duration, lease_expiration_ts, or indefinite.
  lease_expiration_ts = messages.IntegerField(7)
  # Whether or not this lease is indefinite.
  # Specify one of duration, lease_expiration_ts, or indefinite.
  indefinite = messages.BooleanField(8)


class BatchedLeaseRequest(messages.Message):
  """Represents a batched set of LeaseRequests."""
  # LeaseRequest instances to batch together.
  requests = messages.MessageField(LeaseRequest, 1, repeated=True)


class LeaseRequestError(messages.Enum):
  """Represents an error in a LeaseRequest."""
  # Request IDs are intended to be unique.
  # Reusing a request ID in a different request is an error.
  REQUEST_ID_REUSE = 1
  # Proposed Cloud Pub/Sub topic was invalid.
  INVALID_TOPIC = 2
  # Proposed Cloud Pub/Sub project was invalid.
  INVALID_PROJECT = 3
  # Didn't specify a Cloud Pub/Sub topic.
  UNSPECIFIED_TOPIC = 4
  # Request couldn't be processed in time.
  DEADLINE_EXCEEDED = 5
  # Miscellaneous transient error.
  TRANSIENT_ERROR = 6
  # Mutually exclusive duration and lease_expiration_ts both specified.
  MUTUAL_EXCLUSION_ERROR = 7
  # Proposed duration was zero or negative.
  NONPOSITIVE_DEADLINE = 8
  # Proposed expiration time is not in the future.
  LEASE_EXPIRATION_TS_ERROR = 9
  # Neither duration nor lease_expiration_ts were specified.
  LEASE_LENGTH_UNSPECIFIED = 10
  # Requested lease duration is too long.
  LEASE_TOO_LONG = 11


class LeaseRequestState(messages.Enum):
  """Represents the state of a LeaseRequest."""
  # LeaseRequest has been received, but not processed yet.
  UNTRIAGED = 0
  # LeaseRequest is pending provisioning of additional capacity.
  PENDING = 1
  # LeaseRequest has been fulfilled.
  FULFILLED = 2
  # LeaseRequest has been denied.
  DENIED = 3


class LeaseResponse(messages.Message):
  """Represents a response to a LeaseRequest."""
  # SHA-1 identifying the LeaseRequest this response refers to.
  request_hash = messages.StringField(1)
  # LeaseRequestError instance indicating an error with the request, or None
  # if there is no error.
  error = messages.EnumField(LeaseRequestError, 2)
  # Request ID used by the client to generate the LeaseRequest.
  client_request_id = messages.StringField(3, required=True)
  # State of the LeaseRequest.
  state = messages.EnumField(LeaseRequestState, 4)
  # Hostname of the machine available for this request.
  hostname = messages.StringField(5)
  # Timestamp indicating lease expiration seconds from epoch in UTC.
  lease_expiration_ts = messages.IntegerField(6)
  # Whether or not this machine is leased indefinitely.
  # If true, disregard lease_expiration_ts. This lease will not expire.
  leased_indefinitely = messages.BooleanField(7)


class BatchedLeaseResponse(messages.Message):
  """Represents a response to a batched lease request."""
  responses = messages.MessageField(LeaseResponse, 1, repeated=True)


class LeaseReleaseRequest(messages.Message):
  """Represents a request to voluntarily cancel a LeaseRequest."""
  # Per-user unique ID used to identify the LeaseRequest.
  request_id = messages.StringField(1, required=True)


class BatchedLeaseReleaseRequest(messages.Message):
  """Represents a batched set of lease release requests."""
  requests = messages.MessageField(LeaseReleaseRequest, 1, repeated=True)


class LeaseReleaseRequestError(messages.Enum):
  """Represents an error in a LeaseReleaseRequest."""
  # Request ID referred to non-existent request for this user.
  NOT_FOUND = 1
  # Request ID referred to an unfulfilled request.
  NOT_FULFILLED = 2
  # Request ID referred to a fulfilled request whose machine was
  # already reclaimed.
  ALREADY_RECLAIMED = 3
  # Request couldn't be processed in time.
  DEADLINE_EXCEEDED = 4
  # Miscellaneous transient error.
  TRANSIENT_ERROR = 5


class LeaseReleaseResponse(messages.Message):
  """Represents a response to a LeaseReleaseRequest."""
  # SHA-1 identifying the LeaseRequest this response refers to.
  request_hash = messages.StringField(1)
  # LeaseReleaseRequestError indicating an error with the request, or None
  # if there is no error.
  error = messages.EnumField(LeaseReleaseRequestError, 2)
  # Request ID used by the client to generate the LeaseRequest
  # referred to by the LeaseReleaseRequest.
  client_request_id = messages.StringField(3, required=True)


class BatchedLeaseReleaseResponse(messages.Message):
  """Represents responses to a batched set of lease release requests."""
  responses = messages.MessageField(LeaseReleaseResponse, 1, repeated=True)


class MachineInstructionRequest(messages.Message):
  """Represents a request to send an instruction to a leased machine."""
  # Request ID for the fulfilled LeaseRequest whose machine should be
  # instructed.
  request_id = messages.StringField(1, required=True)
  # Instruction to send the leased machine.
  instruction = messages.MessageField(Instruction, 2)


class MachineInstructionError(messages.Enum):
  """Represents an error in a MachineInstructionRequest."""
  # Request ID referred to an unfulfilled request.
  NOT_FULFILLED = 1
  # Request ID referred to a fulfilled request whose machine was
  # already reclaimed.
  ALREADY_RECLAIMED = 2
  # Invalid instruction for the machine.
  INVALID_INSTRUCTION = 3


class MachineInstructionResponse(messages.Message):
  """Represents a response to a MachineInstructionRequest."""
  # Request ID used by the client to generate the LeaseRequest for the
  # machine being instructed.
  client_request_id = messages.StringField(1, required=True)
  # MachineInstructionError indicating an error with the request, or None
  # if there is no error.
  error = messages.EnumField(MachineInstructionError, 2)


class PollRequest(messages.Message):
  """Represents a request to poll for instructions given to a machine."""
  # Hostname of the machine whose instructions to retrieve.
  hostname = messages.StringField(1, required=True)
  # Backend the machine belongs to. Generally required.
  backend = messages.EnumField(Backend, 2)


class PollResponse(messages.Message):
  """Represents a response to a request for instructions given to a machine."""
  # Instruction given to the machine.
  instruction = messages.MessageField(Instruction, 1)
  # State of the instruction.
  state = messages.StringField(2)


class AckRequest(messages.Message):
  """Represents a request to ack an instruction received by a machine."""
  # Hostname of the machine whose instruction to ack.
  hostname = messages.StringField(1, required=True)
  # Backend the machine belongs to.
  backend = messages.EnumField(Backend, 2)
