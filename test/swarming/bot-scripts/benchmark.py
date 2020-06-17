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
import botutil
import json
import os
import sys


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('agi_dir', help='Path to AGI build')
    parser.add_argument('out_dir', help='Path to output directory')
    args = parser.parse_args()

    #### Early checks and sanitization
    assert os.path.isdir(args.agi_dir)
    agi_dir = os.path.normpath(args.agi_dir)
    assert os.path.isdir(args.out_dir)
    out_dir = os.path.normpath(args.out_dir)

    #### Check test parameters
    test_params = {}
    required_keys = ['apk', 'package', 'activity', 'startframe', 'numframes']
    botutil.load_params(test_params, required_keys=required_keys)

    #### Install APK
    botutil.install_apk(test_params)

    #### Call benchmark command
    gapit = os.path.join(agi_dir, 'gapit')
    dumptrace = os.path.join(out_dir, 'benchmark.systrace')
    cmd = [
        gapit, 'benchmark',
        '-startframe', test_params['startframe'],
        '-numframes', test_params['numframes'],
        '-dumptrace', dumptrace
    ]

    if 'additionalargs' in test_params.keys():
        cmd += ['-additionalargs', test_params['additionalargs']]

    cmd += [test_params['package'] + '/' + test_params['activity']]

    p = botutil.runcmd(cmd)

    #### Save gfxtrace
    gfxtrace = 'benchmark.gfxtrace'
    if os.path.isfile(gfxtrace):
        dest = os.path.join(out_dir, gfxtrace)
        os.rename(gfxtrace, dest)

    return p.returncode


if __name__ == '__main__':
    sys.exit(main())
