# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Policies for machines in the Machine Provider Catalog."""

from protorpc import messages


class KeyValuePair(messages.Message):
  """Represents a key-value pair."""
  key = messages.StringField(1, required=True)
  value = messages.StringField(2)


class MachineReclamationPolicy(messages.Enum):
  """Lists valid machine reclamation policies."""
  # Make the machine available for lease again immediately.
  MAKE_AVAILABLE = 1
  # Keep the machine in the Catalog, but prevent it from being leased out.
  RECLAIM = 2
  # Delete the machine from the Catalog.
  DELETE = 3


class Policies(messages.Message):
  """Represents the policies for a machine.

  There are two Pub/Sub channels of communication for each machine.
  One is the backend-level channel which the Machine Provider will use
  to tell the backend that the machine has been leased, or that the machine
  needs to be reclaimed. The other is the channel between the Machine Provider
  and the machine itself. The machine should listen for instructions from the
  Machine Provider on this channel.

  Since the machine itself is what's being leased out to untrusted users,
  we will assign this Cloud Pub/Sub topic and give it restricted permissions
  which only allow it to subscribe to the one topic. On the other hand, the
  backend is trusted so we allow it to choose its own topic.

  When a backend adds a machine to the Catalog, it should provide the Pub/Sub
  topic and project to communicate on regarding the machine, as well as the
  service account on the machine itself which will be used to authenticate
  pull requests on the subscription created by the Machine Provider for the
  machine.
  """
  # Cloud Pub/Sub topic name to communicate on with the backend regarding
  # this machine.
  backend_topic = messages.StringField(1)
  # Cloud Pub/Sub project to communicate on with the backend regarding
  # this machine.
  backend_project = messages.StringField(2)
  # Cloud Pub/Sub service account which the Machine Provider should authorize
  # to consume messages on the machine pull subscription.
  machine_service_account = messages.StringField(3)
  # Action the Machine Provider should take when reclaiming a machine
  # from a lessee.
  on_reclamation = messages.EnumField(
      MachineReclamationPolicy,
      4,
      default=MachineReclamationPolicy.MAKE_AVAILABLE,
  )
  # Cloud Pub/Sub message attributes to include when communicating with the
  # backend regarding this machine.
  backend_attributes = messages.MessageField(KeyValuePair, 5, repeated=True)
