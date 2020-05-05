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

# This Swarming bot test script uses gapit to perform a capture-replay test,
# checking whether the replay output matches the app frames.

import argparse
import botutil
import json
import os
import subprocess
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

    #### Test parameters
    test_params = {
        'startframe': '100',
        'numframes': '5',
        'observe_frames': '1',
    }
    botutil.load_params(test_params)

    #### Install APK
    botutil.install_apk(test_params)

    #### Trace the app
    gapit = os.path.join(agi_dir, 'gapit')
    gfxtrace = os.path.join(out_dir, test_params['package'] + '.gfxtrace')
    cmd = [
        gapit, 'trace',
        '-api', 'vulkan',
        '-start-at-frame', test_params['startframe'],
        '-capture-frames', test_params['numframes'],
        '-observe-frames', test_params['observe_frames'],
        '-out', gfxtrace
    ]

    if 'additionalargs' in test_params.keys():
        cmd += ['-additionalargs', test_params['additionalargs']]

    cmd += [test_params['package'] + '/' + test_params['activity']]

    p = botutil.runcmd(cmd)
    if p.returncode != 0:
        return 1

    #### Replay
    videofile = os.path.join(out_dir, test_params['package'] + '.mp4')
    cmd = [
        gapit, 'video',
        '-gapir-nofallback',
        '-type', 'sxs',
        '-frames-minimum', test_params['numframes'],
        '-out', videofile,
        gfxtrace
    ]
    p = botutil.runcmd(cmd)
    return p.returncode


if __name__ == '__main__':
    sys.exit(main())
