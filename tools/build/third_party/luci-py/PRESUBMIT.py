# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Top-level presubmit script for LUCI.

See http://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into gcl.
"""


def header(input_api):
  """Returns the expected license header regexp for this project."""
  current_year = int(input_api.time.strftime('%Y'))
  allowed_years = (str(s) for s in reversed(xrange(2011, current_year + 1)))
  years_re = '(' + '|'.join(allowed_years) + ')'
  license_header = (
    r'.*? Copyright %(year)s The LUCI Authors\. '
      r'All rights reserved\.\n'
    r'.*? Use of this source code is governed under the Apache License, '
      r'Version 2\.0\n'
    r'.*? that can be found in the LICENSE file\.(?: \*/)?\n'
  ) % {
    'year': years_re,
  }
  return license_header


def CommonChecks(input_api, output_api):
  excluded = [
    r'.+-build\.(js|html)$',
    r'.+/build/.+(js|html)$',
    r'.+/dist/.+(js|html|css)$',
    r'/test',
    r'.+_pb2\.py$',
    r'.+_pb2_grpc\.py$',
    r'.*third_party.*',
    # These are a symlink to third_party, so it shouldn't be checked.
    r'appengine/isolate/bqh\.py$',
    r'appengine/swarming/bqh\.py$',
  ]
  return input_api.canned_checks.PanProjectChecks(
      input_api, output_api,
      excluded_paths=excluded,
      license_header=header(input_api))


def CheckChangeOnUpload(input_api, output_api):
  return CommonChecks(input_api, output_api)


def CheckChangeOnCommit(input_api, output_api):
  return CommonChecks(input_api, output_api)
