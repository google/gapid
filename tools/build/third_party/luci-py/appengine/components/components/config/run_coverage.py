#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys

THIS_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, os.path.join(THIS_DIR, '..', '..'))

from tools import run_coverage


if __name__ == '__main__':
  sys.exit(run_coverage.main(
      THIS_DIR,
      [],
      'PRESUBMIT.py,components,*test*,tool*'))
