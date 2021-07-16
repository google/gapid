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

# This Swarming bot test script uses gapit to perform various tests on a given
# workload.

import argparse
import botutil
import json
import os
import subprocess
import sys

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('adb_path', help='Path to adb command')
    parser.add_argument('agi_dir', help='Path to AGI build')
    parser.add_argument('out_dir', help='Path to output directory')
    args = parser.parse_args()

    #### Early checks and sanitization
    assert os.path.isfile(args.adb_path)
    adb_path = os.path.abspath(args.adb_path)
    assert os.path.isdir(args.agi_dir)
    agi_dir = os.path.abspath(args.agi_dir)
    assert os.path.isdir(args.out_dir)
    out_dir = os.path.abspath(args.out_dir)
    gapit_path = os.path.join(agi_dir, 'gapit')

    #### Create BotUtil with relevant adb and gapit paths
    bu = botutil.BotUtil(adb_path)
    bu.set_gapit_path(gapit_path)

    #### Test parameters
    test_params = {
        'startframe': '100',
        'numframes': '5',
        'observe_frames': '1',
        'api': 'vulkan',
    }
    required_keys = ['apk', 'package', 'activity']
    botutil.load_params(test_params, required_keys=required_keys)

    #### Install APK
    bu.install_apk(test_params)

    #### Trace the app
    gfxtrace = os.path.join(out_dir, test_params['package'] + '.gfxtrace')
    args = [
        '-api', test_params['api'],
        '-start-at-frame', test_params['startframe'],
        '-capture-frames', test_params['numframes'],
        '-observe-frames', test_params['observe_frames'],
        '-out', gfxtrace
    ]

    if 'additionalargs' in test_params.keys():
        args += ['-additionalargs', test_params['additionalargs']]

    args += [test_params['package'] + '/' + test_params['activity']]

    p = bu.gapit('trace', args)
    if p.returncode != 0:
        return 1

    #### Stop the app asap for device cool-down
    bu.adb(['shell', 'am', 'force-stop', test_params['package']])

    #### Replay
    # Use the 'sxs-frames' mode that generates a series of PNGs rather
    # than an mp4 video. This makes inspection easier, and removes the
    # dependency on ffmpeg on the running hosts.
    videooutfile = os.path.join(out_dir, test_params['package'] + '.frame.png')
    gapit_args = [
        '-gapir-nofallback',
        '-type', 'sxs-frames',
        '-frames-minimum', test_params['numframes'],
        '-out', videooutfile,
        gfxtrace
    ]
    p = bu.gapit('video', gapit_args)
    if p.returncode != 0:
        return p.returncode

    #### Screenshot test to retrieve mid-frame resources
    # This is meant to test the command buffer splitter, which is invoked to be
    # able to retrieve the framebuffer in the middle of a render pass. We ask
    # for the framebuffer at the 5th draw call, this number was choosen because:
    # it is low enough to be present in most frames (i.e. we expect frames to
    # have at least 5 draw calls), and it hopefully falls in the middle of a
    # renderpass. Also, we don't want to have a random number here, as we want
    # to keep the tests as reproducible as feasible.
    screenshotfile = os.path.join(out_dir, test_params['package'] + '.png')
    gapit_args = [
        '-executeddraws', '5',
        '-out', screenshotfile,
        gfxtrace
    ]
    p = bu.gapit('screenshot', gapit_args)
    if p.returncode != 0:
        return p.returncode

    #### Frame profiler
    # Check that frame profiling generates valid JSON
    profile_json = os.path.join(out_dir, test_params['package'] + '.profiling.json')
    gapit_args = [
        '-json',
        '-out', profile_json,
        gfxtrace
    ]
    p = bu.gapit('profile', gapit_args)
    if p.returncode != 0:
        return p.returncode
    assert botutil.is_valid_json(profile_json)

    #### Frame graph
    # Check that framegraph generates valid JSON
    framegraph_json = os.path.join(out_dir, test_params['package'] + '.framegraph.json')
    gapit_args = [
        '-json', framegraph_json,
        gfxtrace
    ]
    p = bu.gapit('framegraph', gapit_args)
    if p.returncode != 0:
        return p.returncode
    assert botutil.is_valid_json(framegraph_json)

    #### All tests have passed, return success
    return 0

if __name__ == '__main__':
    sys.exit(main())
