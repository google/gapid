# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Actual WSGI app instantiations used from app.yaml.

Function 'main.initialize' must be called from a separate module
not imported in tests.
"""

import main

html, endpoints, backend = main.initialize()
