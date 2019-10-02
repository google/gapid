# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utility functions for Protocol Buffers."""

# Pylint doesn't like relative wildcard imports.
# pylint: disable=W0401,W0403

from .field_masks import *
from .multiline_proto import parse_multiline, MultilineParseError
from .protoutil import merge_dict
