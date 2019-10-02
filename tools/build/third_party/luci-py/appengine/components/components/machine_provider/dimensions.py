# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Dimensions for the Machine Provider."""

from protorpc import messages


class Backend(messages.Enum):
  """Lists valid backends."""
  DUMMY = 0
  GCE = 1
  VSPHERE = 2


class OSFamily(messages.Enum):
  """Lists valid OS families."""
  LINUX = 1
  OSX = 2
  WINDOWS = 3


class LinuxFlavor(messages.Enum):
  """Lists valid flavors of Linux."""
  UBUNTU = 1
  DEBIAN = 2


class DiskTypes(messages.Enum):
  """Lists valid disk types."""
  HDD = 1
  SSD = 2


class Dimensions(messages.Message):
  """Represents the dimensions of a machine."""
  # The operating system family of this machine.
  os_family = messages.EnumField(OSFamily, 1)
  # The backend which should be used to spin up this machine. This should
  # generally be left unspecified so the Machine Provider selects the backend
  # on its own.
  backend = messages.EnumField(Backend, 2)
  # The hostname of this machine.
  hostname = messages.StringField(3)
  # The number of CPUs available to this machine.
  num_cpus = messages.IntegerField(4)
  # The amount of memory available to this machine.
  memory_gb = messages.FloatField(5)
  # The disk space available to this machine.
  disk_gb = messages.IntegerField(6)
  # The flavor of Linux of this machine.
  linux_flavor = messages.EnumField(LinuxFlavor, 7)
  # The operating system version of this machine.
  os_version = messages.StringField(8)
  # The project this machine was created in.
  project = messages.StringField(9)
  # The type of disk this machine has.
  disk_type = messages.EnumField(DiskTypes, 10)
  # The name of the snapshot used by this machine.
  snapshot = messages.StringField(11)
  # The labels describing the snapshot used by this machine.
  snapshot_labels = messages.StringField(12, repeated=True)
