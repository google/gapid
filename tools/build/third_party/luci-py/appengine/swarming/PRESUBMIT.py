# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Top-level presubmit script for swarming-server.

See http://dev.chromium.org/developers/how-tos/depottools/presubmit-scripts for
details on the presubmit API built into gcl.
"""


def FindAppEngineSDK(input_api):
  """Returns an absolute path to AppEngine SDK (or None if not found)."""
  import sys
  old_sys_path = sys.path
  try:
    sys.path = [input_api.PresubmitLocalPath()] + sys.path
    from tool_support import gae_sdk_utils
    return gae_sdk_utils.find_gae_sdk()
  finally:
    sys.path = old_sys_path


def CommonChecks(input_api, output_api):
  output = []
  def join(*args):
    return input_api.os_path.join(input_api.PresubmitLocalPath(), *args)

  gae_sdk_path = FindAppEngineSDK(input_api)
  if not gae_sdk_path:
    output.append(output_api.PresubmitError('Couldn\'t find AppEngine SDK.'))
  if not input_api.os_path.isfile(join('..', '..', 'client', 'swarming.py')):
    output.append(
        output_api.PresubmitError(
            'Couldn\'t find ../../client. Please run:\n'
            '  git submodule init\n'
            '  git submodule update'))
  if output:
    return output

  import sys
  old_sys_path = sys.path
  try:
    # Add GAE SDK modules to sys.path.
    sys.path = [gae_sdk_path] + sys.path
    import appcfg
    appcfg.fix_sys_path()
    # Add project specific paths to sys.path
    sys.path = [
      join('..', 'components'),
      join('..', 'third_party_local'),
      join('..', '..', 'client', 'tests'),
    ] + sys.path
    black_list = list(input_api.DEFAULT_BLACK_LIST) + [
      r'.*_pb2\.py$',
      r'.*_pb2_grpc\.py$',
    ]
    disabled_warnings = [
      'relative-import',
    ]
    output.extend(input_api.canned_checks.RunPylint(
        input_api, output_api,
        black_list=black_list,
        disabled_warnings=disabled_warnings))
  finally:
    sys.path = old_sys_path

  test_directories = [
    input_api.PresubmitLocalPath(),
    join('server'),
    join('swarming_bot'),
    join('swarming_bot', 'api'),
    join('swarming_bot', 'api', 'platforms'),
    join('swarming_bot', 'bot_code'),
  ]

  blacklist = [
    # Never run the remote_smoke_test automatically. Should instead be run after
    # uploading a server instance.
    r'^remote_smoke_test\.py$'
  ]
  tests = []
  for directory in test_directories:
    tests.extend(
        input_api.canned_checks.GetUnitTestsInDirectory(
            input_api, output_api,
            directory,
            whitelist=[r'.+_test\.py$'],
            blacklist=blacklist))
  output.extend(input_api.RunTests(tests, parallel=True))
  return output


# pylint: disable=unused-argument
def CheckChangeOnUpload(input_api, output_api):
  return []


def CheckChangeOnCommit(input_api, output_api):
  return CommonChecks(input_api, output_api)
