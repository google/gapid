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

# This Swarming bot test script calls the gapit benchmark command.

import argparse
import json
import os
import subprocess
import sys


def runcmd(cmd):
    p = subprocess.run(cmd, stdout=sys.stdout, stderr=sys.stderr)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('agi_dir', help='Path to AGI build')
    parser.add_argument('out_dir', help='Path to output directory')
    parser.add_argument('--force-install', help='Force installation of APK')
    args = parser.parse_args()

    #### Early checks and sanitization
    assert os.path.isdir(args.agi_dir)
    agi_dir = os.path.normpath(args.agi_dir)
    assert os.path.isdir(args.out_dir)
    out_dir = os.path.normpath(args.out_dir)

    #### Check test parameters
    test_params = {}
    params_file = 'params.json'
    assert os.path.isfile(params_file)
    with open(params_file, 'r') as f:
        test_params = json.load(f)
    for k in ['apk', 'package', 'activity', 'startframe', 'numframes']:
        assert k in test_params.keys()

    #### Install APK
    runcmd(['adb', 'install', '-g', '-t', test_params['apk']])

    #### Call benchmark command
    gapit = os.path.join(agi_dir, 'gapit')
    cmd = [
        gapit, 'benchmark',
        '-startframe', test_params['startframe'],
        '-numframes', test_params['numframes'],
        test_params['package'] + '/' + test_params['activity']
    ]
    runcmd(cmd)

    #### Save gfxtrace
    gfxtrace = 'benchmark.gfxtrace'
    if os.path.isfile(gfxtrace):
        dest = os.path.join(out_dir, gfxtrace)
        os.rename(gfxtrace, dest)


if __name__ == '__main__':
    sys.exit(main())
