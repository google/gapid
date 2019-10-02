# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Implements Cloud Endpoints v1 over webapp2 routes."""

# Pylint doesn't like relative wildcard imports.
# pylint: disable=relative-import, wildcard-import

from adapter import *
from discovery import *
