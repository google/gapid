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

# This script collects the results of a Swarming task for a given test, and add
# it to a result file.

import argparse
import json
import os
import subprocess
import sys

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('build_timestamp', help='AGI build timestamp, prefer YYYYMMDD-HHMMSS format')
    parser.add_argument('commit', help='AGI commit sha')
    parser.add_argument('test_name', help='Test name to use when storing the test results.')
    parser.add_argument('task_json', help='Task json file produced by Swarming trigger. The name of the test is this obtained from this file name by removing the ".json" extension.')
    parser.add_argument('results_json', help='Results json file to store result')
    args = parser.parse_args()

    #### Check LUCI is installed
    assert 'LUCI_CLIENT_ROOT' in os.environ.keys()

    #### Collect swarming result
    summary = 'summary.json'
    # Make sure summary file does not exist
    if os.path.exists(summary):
        os.remove(summary)
    cmd = [
        os.path.join(os.environ['LUCI_CLIENT_ROOT'], 'swarming.py'),
        'collect',
        '--swarming=https://chrome-swarming.appspot.com',
        '--task-output-stdout=none',
        '--task-summary-json', summary,
        '--json', args.task_json,
    ]
    if ('SWARMING_AUTH_FLAG' in os.environ.keys()) and (os.environ['SWARMING_AUTH_FLAG'] != ''):
        cmd += [ os.environ['SWARMING_AUTH_FLAG'] ]
    # Do NOT check the return code, as collect returns the return code from the
    # task, so we must be robust to non-zero return code from this command.
    subprocess.call(cmd)
    # The commmand must have produced a summary file
    assert os.path.exists(summary)

    #### Extract test results
    with open(summary, 'r') as f:
        result = json.load(f)
    # We expect a single shard
    assert len(result['shards']) == 1
    task_result = result['shards'][0]
    device = ''
    task_id = task_result['task_id']
    # Find device name in the "tags" list, the entry looks like
    # "device_type:<device>". The device is also mentioned the bot_dimensions
    # list with a "device_type" key, but the value of this dimension is itself a
    # list for which we have no guarantee on the order. Hence we prefer to
    # retrieve the device name from the tags.
    for tag in task_result['tags']:
        if tag.startswith('device_type:'):
            device = tag[len('device_type:'):]
    assert device != ''
    # Set status: default to fail, and lower the risk of false-positive by being
    # pedantic to setup the 'pass' status. Warning: an expired task have less
    # fields, e.g. it doesn't have the 'exit_code' field.
    status = 'fail'
    if ('exit_code' in task_result.keys()) and (task_result['exit_code'] == '0') and (task_result['state'] == 'COMPLETED') and (task_result['failure'] == False) and (task_result['internal_failure'] == False):
        status = 'pass'
    elif task_result['state'] == 'TIMED_OUT':
        status = 'timeout'
    elif task_result['state'] == 'EXPIRED':
        status = 'expired'
    elif task_result['internal_failure']:
        status = 'internal_failure'

    #### Add result to result file
    # Results are stored in JSON with the following format:
    # {
    #   "20200324-032345": { # Build timestamp, YYYYMMDD-HHMMSS to be usable as a key to sort
    #     "commit": "a1b2c3e4f5",  # build commit sha
    #     "tests": {
    #       "flame": { # device name
    #         "foo-bar": { # test name
    #           "task_id": "abcd1234", # Swarming task ID
    #           "status": "pass"      # Swarming task status (pass/fail/expired...)
    #         },
    #         ...
    #       }
    #     }
    #   }
    # }

    if os.path.exists(args.results_json):
        with open(args.results_json, 'r') as f:
            results = json.load(f)
    else:
        print('Warning: results file "' + args.results_json + '" does not exist, it will be created')
        results = {}

    if args.build_timestamp in results.keys():
        assert results[args.build_timestamp]['commit'] == args.commit
    else:
        results[args.build_timestamp] = {
            'commit': args.commit,
            'tests': {}
        }

    if device not in results[args.build_timestamp]['tests'].keys():
        results[args.build_timestamp]['tests'][device] = {}
    assert args.test_name not in results[args.build_timestamp]['tests'][device].keys()
    results[args.build_timestamp]['tests'][device][args.test_name] = {
        'task_id': task_id,
        'status': status
    }

    #### Write back result file
    os.makedirs(os.path.dirname(args.results_json), exist_ok=True)
    with open(args.results_json, 'w') as f:
        f.write(json.dumps(results, sort_keys=True, indent=2))

    #### Exit code
    # Return non-zero if the task failed, but silence failures due to expiration
    # of an internal failure, because we do not want Swarming infrastructure
    # failures to be reported as build failures.
    if status == 'pass' or status == 'expired' or status == 'internal_failure':
        return 0
    else:
        return 1


if __name__ == '__main__':
    sys.exit(main())
