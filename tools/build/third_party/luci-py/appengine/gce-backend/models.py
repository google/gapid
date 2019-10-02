# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Datastore model for GCE Backend."""

from google.appengine.ext import ndb
from google.appengine.ext.ndb import msgprop

from components.machine_provider import rpc_messages

import utilities


class ServiceAccount(ndb.Model):
  """A service account.

  Standalone instances should not be present in the datastore.
  """
  # Name of the service account.
  name = ndb.StringProperty(indexed=False)
  # List of authorized OAuth2 scopes for this service account.
  scopes = ndb.StringProperty(indexed=False, repeated=True)


class Instance(ndb.Model):
  """A GCE instance.

  Key:
    id: Space-separated string:
      instance template base name, revision, zone, GCE instance name.
    parent: None (root).
  """
  # Whether or not this instance is cataloged in the Machine Provider.
  cataloged = ndb.BooleanProperty(indexed=True)
  # Whether or not this instance has been deleted.
  deleted = ndb.BooleanProperty(indexed=True)
  # Time that the deletion request for this instance was first initiated.
  deletion_ts = ndb.DateTimeProperty(indexed=False)
  # The disk attached to this instance.
  disk = ndb.StringProperty(indexed=False)
  # Name of this instance.
  hostname = ndb.ComputedProperty(
      lambda self: self.key.id().split()[-1], indexed=True)
  # ndb.Key for the InstanceGroupManager this Instance belongs to.
  instance_group_manager = ndb.KeyProperty(indexed=True)
  # Last modification to this entity.
  last_updated = ndb.DateTimeProperty(auto_now=True, indexed=True)
  # Whether or not this instance has been leased out by Machine Provider.
  leased = ndb.ComputedProperty(
      lambda self: bool(self.leased_indefinitely or self.lease_expiration_ts),
      indexed=True)
  # Indicates whether this instance is leased indefinitely.
  leased_indefinitely = ndb.BooleanProperty()
  # DateTime indicating when the lease on this instance expires.
  lease_expiration_ts = ndb.DateTimeProperty(indexed=False)
  # Whether or not this instance is pending deletion.
  pending_deletion = ndb.BooleanProperty(indexed=True)
  # URL of the instance.
  url = ndb.StringProperty(indexed=False)


class InstanceGroupManager(ndb.Model):
  """An instance group manager in the config.

  Key:
    id: zone of the
      proto.config_pb2.InstanceGroupManagerConfig.InstanceGroupManager this
      entity represents.
    parent: InstanceTemplateRevision.
  """
  # Current number of instances managed by the instance group manager created
  # from this entity.
  current_size = ndb.ComputedProperty(lambda self: len(self.instances))
  # ndb.Keys for the active Instances.
  instances = ndb.KeyProperty(kind=Instance, repeated=True)
  # Maximum number of instances the instance group manager created from this
  # entity can maintain. Must be at least equal to minimum_size. Leave
  # unspecified for unlimited.
  maximum_size = ndb.IntegerProperty(indexed=False)
  # Minimum number of instances the instance group manager created from this
  # entity should maintain. Must be positive. Also defines the initial size
  # when first creating the instance group manager.
  minimum_size = ndb.IntegerProperty(indexed=False)
  # URL of the instance group manager created from this entity.
  url = ndb.StringProperty(indexed=False)


class InstanceTemplateRevision(ndb.Model):
  """A specific revision of an instance template in the config.

  Key:
    id: Checksum of the instance template config.
    parent: InstanceTemplate.
  """
  # List of ndb.Keys for the InstanceGroupManagers.
  active = ndb.KeyProperty(kind=InstanceGroupManager, repeated=True)
  # Enable external network with automatic IP assignment.
  auto_assign_external_ip = ndb.BooleanProperty(indexed=False)
  # rpc_messages.Dimensions describing instances created from this template.
  dimensions = msgprop.MessageProperty(rpc_messages.Dimensions)
  # Disk size in GiB for instances created from this template.
  disk_size_gb = ndb.IntegerProperty(indexed=False)
  # Disk type for instances created from this template.
  disk_type = ndb.StringProperty(indexed=False)
  # List of ndb.Keys for drained InstanceGroupManagers.
  drained = ndb.KeyProperty(kind=InstanceGroupManager, repeated=True)
  # Name of the image for instances created from this template.
  image_name = ndb.StringProperty(indexed=False)
  # Project containing the image specified by image_name.
  image_project = ndb.StringProperty(indexed=False)
  # GCE machine type for instances created from this template.
  machine_type = ndb.StringProperty(indexed=False)
  # Initial metadata to apply when creating instances from this template.
  metadata = ndb.JsonProperty()
  # Minimum CPU platform for instances created from this template.
  min_cpu_platform = ndb.StringProperty(indexed=False)
  # Network URL for this template.
  network_url = ndb.StringProperty(indexed=False)
  # Project to create the instance template in.
  project = ndb.StringProperty(indexed=False)
  # List of service accounts available to instances created from this template.
  service_accounts = ndb.LocalStructuredProperty(ServiceAccount, repeated=True)
  # List of labels describing a snapshot to create a disk from for instances
  # created from this template.
  snapshot_labels = ndb.StringProperty(indexed=False, repeated=True)
  # Name of a snapshot to create a disk from for instances created from this
  # template.
  snapshot_name = ndb.StringProperty(indexed=False)
  # Snapshot URL derived from snapshot labels and name.
  snapshot_url = ndb.StringProperty(indexed=False)
  # Initial list of tags to apply when creating instances from this template.
  tags = ndb.StringProperty(indexed=False, repeated=True)
  # URL of the instance template created from this entity.
  url = ndb.StringProperty(indexed=False)


class InstanceTemplate(ndb.Model):
  """An instance template in the config.

  Key:
    id: base_name of the
      proto.config_pb2.InstanceTemplateConfig.InstanceTemplate this entity
      represents.
    parent: None (root entity).
  """
  # ndb.Key for the active InstanceTemplateRevision.
  active = ndb.KeyProperty(kind=InstanceTemplateRevision)
  # List of ndb.Keys for drained InstanceTemplateRevisions.
  drained = ndb.KeyProperty(kind=InstanceTemplateRevision, repeated=True)
