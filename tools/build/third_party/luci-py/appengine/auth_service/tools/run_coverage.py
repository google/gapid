#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys

APP_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.join(APP_DIR, '..', 'components'))

from tools import run_coverage


if __name__ == '__main__':
  sys.exit(run_coverage.main(
      APP_DIR,
      ('tools',),
      'PRESUBMIT.py,components,*test*,tool*'))
