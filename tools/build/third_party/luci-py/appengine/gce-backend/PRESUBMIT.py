# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Presubmit for GCE Backend.

See http://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into git cl.
"""

def CommonChecks(input_api, output_api):
  output = input_api.canned_checks.RunPylint(
      input_api,
      output_api,
      black_list=[r'.*_pb2\.py$'],
      # TODO(smut): Fix cyclic import (config.py <-> metrics.py).
      disabled_warnings=['cyclic-import'],
  )
  tests = input_api.canned_checks.GetUnitTestsInDirectory(
      input_api,
      output_api,
      input_api.PresubmitLocalPath(),
      whitelist=[r'.+_test\.py$'],
  )
  output.extend(input_api.RunTests(tests, parallel=True))
  return output


# pylint: disable=unused-argument
def CheckChangeOnUpload(input_api, output_api):
  return []


def CheckChangeOnCommit(input_api, output_api):
  return CommonChecks(input_api, output_api)
