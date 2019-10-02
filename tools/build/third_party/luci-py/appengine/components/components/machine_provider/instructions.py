# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Instructions for machines."""

from protorpc import messages


class Instruction(messages.Message):
  """Represents instructions for a machine."""
  # Swarming server to connect to. e.g. https://chromium-swarm.appspot.com.
  swarming_server = messages.StringField(1)
