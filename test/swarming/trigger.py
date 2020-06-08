#!/usr/bin/env python3

# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script triggers a Swarming task for a given test.

import argparse
import hashlib
import json
import os
import subprocess
import sys

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('test_dir', help='Path to a test directory')
    parser.add_argument('--prefix', help='Prefix for Swarming task name')
    args = parser.parse_args()

    #### Early checks and sanitization
    assert 'LUCI_CLIENT_ROOT' in os.environ.keys()
    assert os.path.isdir(args.test_dir)
    test_dir = os.path.normpath(args.test_dir)
    prefix = ''
    if args.prefix:
        prefix = args.prefix + '_'

    #### Set-up default Swarming parameters
    # Must all be strings
    swarming_params = {
        'test_dir': test_dir,
        'test_name': os.path.basename(test_dir),
        'task_prefix': prefix,
        'devices': ['flame'],
        'priority': '100',
        'timeout': '300',
        'expiration': '600',
    }

    #### Pick up specific parameters from test_dir/params.json
    params_file = os.path.join(test_dir, 'params.json')
    if os.path.isfile(params_file):
        with open(params_file, 'r') as f:
            params = json.load(f)
        for key in params.keys():
            swarming_params[key] = params[key]
    # Setup task name
    swarming_params['task_name'] = swarming_params['task_prefix'] + swarming_params['test_name']

    #### Generate Isolate config file
    isolate_file = swarming_params['test_name'] + '.isolate'
    # The trailing '/' in variables -> files -> $test_dir/ is necessary to
    # indicate all the directory content must be uploaded.
    isolate_body = '''{
  'variables': {
    'files': [
      'agi/',
      'bot-harness.py',
      'bot-scripts/',
''' + "'" + swarming_params['test_dir'] + "/'" + ''',
    ]
  },
}
'''
    with open(isolate_file, 'w') as f:
        f.write(isolate_body)

    #### Upload to isolate
    isolated_file = swarming_params['test_name'] + '.isolated'
    # Make sure isolated file does not exist
    if os.path.exists(isolated_file):
        os.remove(isolated_file)
    cmd = [
        os.path.join(os.environ['LUCI_CLIENT_ROOT'], 'isolate.py'),
        'archive',
        '--isolate-server=https://chrome-isolated.appspot.com',
        '--isolate', isolate_file,
        '--isolated', isolated_file
    ]
    if ('SWARMING_AUTH_FLAG' in os.environ.keys()) and (os.environ['SWARMING_AUTH_FLAG'] != ''):
        cmd += [ os.environ['SWARMING_AUTH_FLAG'] ]
    # We expect this command to always succeed
    subprocess.run(cmd, check=True)
    # The isolated file must be produced
    assert os.path.isfile(isolated_file)

    #### Get the isolated SHA
    isolated_sha = ''
    with open(isolated_file, 'rb') as f:
        isolated_sha = hashlib.sha1(f.read()).hexdigest()
    assert isolated_sha != ''

    #### Trigger the Swarming task
    for device in swarming_params['devices']:
        triggered_dir = os.path.join('triggered', device)
        os.makedirs(triggered_dir, exist_ok=True)
        task_json = os.path.join(triggered_dir, swarming_params['test_name'] + '.json')
        # Make sure task JSON does not exist
        if os.path.exists(task_json):
            os.remove(task_json)
        cmd = [
            os.path.join(os.environ['LUCI_CLIENT_ROOT'], 'swarming.py'),
            'trigger',
            '--swarming=https://chrome-swarming.appspot.com',
            '--isolate-server=https://chrome-isolated.appspot.com',
            '--isolated', isolated_sha,
            '--task-name', swarming_params['task_name'],
            '--dump-json', task_json,
            '--dimension', 'pool', 'SkiaInternal',
            '--dimension', 'device_type', device,
            '--priority', swarming_params['priority'],
            '--expiration', swarming_params['expiration'],
            '--hard-timeout', swarming_params['timeout'],
        ]
        if ('SWARMING_AUTH_FLAG' in os.environ.keys()) and (os.environ['SWARMING_AUTH_FLAG'] != ''):
            cmd += [ os.environ['SWARMING_AUTH_FLAG'] ]

        # Since June 2020, using the 'command' field in the 'isolate' file is
        # deprecated on Swarming infra, so we MUST use the --raw-cmd flag.
        cmd += [
            '--raw-cmd', '--',
            'python3',
            './bot-harness.py',
            swarming_params['timeout'],
            swarming_params['test_dir'],
            # '${ISOLATED_OUTDIR}' is a special string that must appear as-is, it
            # is replaced by Swarming to point to the directory where test
            # outputs can be saved
            '${ISOLATED_OUTDIR}',
        ]

        # We expect this command to always succeed
        subprocess.run(cmd, check=True)
        # The task JSON file must be produced
        assert os.path.isfile(task_json)


if __name__ == '__main__':
    sys.exit(main())
