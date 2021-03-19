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

# This script is the swarming task harness. This is the entry point for the
# Swarming bot.

import argparse
import glob
import json
import os
import subprocess
import sys
import time

# Load our own botutil
sys.path.append(os.path.join(os.path.dirname(os.path.abspath(__file__)), 'bot-scripts'))
import botutil


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('timeout', type=int, help='Timeout (duration limit for this test), in seconds')
    parser.add_argument('test_dir', help='Path to test directory, e.g. tests/foobar')
    parser.add_argument('out_dir', help='Path to output directory')
    args = parser.parse_args()

    #### Early checks and sanitization
    assert os.path.isdir(args.test_dir)
    test_dir = os.path.abspath(args.test_dir)
    assert os.path.isdir(args.out_dir)
    out_dir = os.path.abspath(args.out_dir)
    # bot-scripts/ contains test scripts
    assert os.path.isdir('bot-scripts')
    # agi/ contains the AGI build
    assert os.path.isdir('agi')
    agi_dir = os.path.abspath('agi')

    #### Print AGI build properties (AGI version, build commit SHA)
    cmd = ['cat', os.path.join(agi_dir, 'build.properties')]
    botutil.runcmd(cmd)

    #### Check test parameters
    test_params = {}
    params_file = os.path.join(test_dir, 'params.json')
    assert os.path.isfile(params_file)
    with open(params_file, 'r') as f:
        test_params = json.load(f)
    assert 'script' in test_params.keys()
    test_script = os.path.abspath(os.path.join('bot-scripts', test_params['script']))
    assert os.path.isfile(test_script)

    #### Timeout: make room for pre-script checks and post-script cleanup.
    # All durations are in seconds.
    cleanup_timeout = 15
    if args.timeout < cleanup_timeout:
        print('Error: timeout must be higher than the time for cleanup duration ({} sec)'.format(cleanup_timeout))
        return 1
    test_timeout = args.timeout - cleanup_timeout

    #### Check Android device access
    # This first adb command may take a while if the adb deamon has to launch
    p = botutil.adb(['shell', 'true'], timeout=10)
    if p.returncode != 0:
        print('Error: zero or more than one device connected')
        return 1
    # Print device fingerprint
    p = botutil.adb(['shell', 'getprop', 'ro.build.fingerprint'])
    print('Device fingerprint: ' + p.stdout)

    #### Prepare device
    # Wake up (224) and unlock (82) screen, sleep to pass any kind of animation
    # The screen wakeup (224) call sometimes takes more than a second to return,
    # hence the extended timeout.
    botutil.adb(['shell', 'input', 'keyevent', '224'], timeout=2)
    time.sleep(2)
    # TODO(b/157444640): Temporary workaround: touch the screen before unlocking it to bypass a possible "Android preview" notification
    botutil.adb(['shell', 'input', 'touchscreen', 'tap', '100', '100'])
    time.sleep(1)
    botutil.adb(['shell', 'input', 'keyevent', '82'])
    time.sleep(1)
    # Turn brightness to a minimum, to prevent device to get too hot
    botutil.adb(['shell', 'settings', 'put', 'system', 'screen_brightness', '0'])
    # Make sure to have the screen "stay awake" during the test, we turn off the screen ourselves at the end
    botutil.adb(['shell', 'settings', 'put', 'global', 'stay_on_while_plugged_in', '7'])
    # Avoid "Viewing full screen" notifications that makes app loose focus
    botutil.adb(['shell', 'settings', 'put', 'secure', 'immersive_mode_confirmations', 'confirmed'])
    # Remove any implicit vulkan layers
    botutil.adb(['shell', 'settings', 'delete', 'global', 'enable_gpu_debug_layers'])
    botutil.adb(['shell', 'settings', 'delete', 'global', 'gpu_debug_app'])
    botutil.adb(['shell', 'settings', 'delete', 'global', 'gpu_debug_layers'])
    botutil.adb(['shell', 'settings', 'delete', 'global', 'gpu_debug_layer_app'])
    # Clean up logcat, can take a few seconds
    botutil.adb(['logcat', '-c'], timeout=5)

    #### Launch test script
    print('Start test script "{}" with timeout of {} seconds'.format(test_script, test_timeout))
    cmd = [test_script, agi_dir, out_dir]
    test_returncode = None
    stdout_filename = os.path.abspath(os.path.join(out_dir, 'stdout.txt'))
    stderr_filename = os.path.abspath(os.path.join(out_dir, 'stderr.txt'))
    with open(stdout_filename, 'w') as stdout_file:
        with open(stderr_filename, 'w') as stderr_file:
            try:
                p = subprocess.run(cmd, timeout=test_timeout, cwd=test_dir, stdout=stdout_file, stderr=stderr_file)
                test_returncode = p.returncode
            except subprocess.TimeoutExpired as err:
                # Mirror returncode from unix 'timeout' command
                test_returncode = 124

    #### Dump the logcat
    logcat_file = os.path.join(out_dir, 'logcat.txt')
    with open(logcat_file, 'w') as f:
        cmd = ['adb', 'logcat', '-d']
        p = subprocess.run(cmd, timeout=5, check=True, stdout=f)

    #### Dump test outputs
    with open(stdout_filename, 'r') as f:
        print('#### Test stdout:')
        print(f.read())
    with open(stderr_filename, 'r') as f:
        print('#### Test stderr:')
        print(f.read())
    print('#### Test returncode:')
    print(test_returncode)

    #### Turn off the device screen
    # Key "power" (26) toggle between screen off and on, so first make sure to
    # have the screen on with key "wake up" (224), then press "power" (26).
    # The screen wakeup (224) call sometimes takes more than a second to return,
    # hence the extended timeout.
    botutil.adb(['shell', 'input', 'keyevent', '224'], timeout=2)
    # Wait a bit to let any kind of device wake up animation terminate
    time.sleep(2)
    botutil.adb(['shell', 'input', 'keyevent', '26'])

    #### Force-stop AGI and test app
    for abi in ['armeabiv7a', 'arm64v8a']:
        botutil.adb(['shell', 'am', 'force-stop', 'com.google.android.gapid.' + abi])
    if 'package' in test_params.keys():
        botutil.adb(['shell', 'am', 'force-stop', test_params['package']])

    #### Test may fail halfway through, salvage any gfxtrace
    gfxtraces = glob.glob(os.path.join(test_dir, '*.gfxtrace'))
    # Do not salvage a gfxtrace that is listed as a test input
    if ('gfxtrace' in test_params.keys()):
        g = os.path.join(test_dir, test_params['gfxtrace'])
        if g in gfxtraces:
            gfxtraces.remove(g)
    if len(gfxtraces) != 0:
        salvage_dir = os.path.join(out_dir, 'harness-salvage')
        os.makedirs(salvage_dir, exist_ok=True)
        for gfx in gfxtraces:
            dest = os.path.join(salvage_dir, os.path.basename(gfx))
            os.rename(gfx, dest)

    #### Analyze the return code
    print('#### Test status:')
    if test_returncode == 124:
        print('TIMEOUT')
        print('Sleep a bit more to trigger a Swarming-level timeout, to disambiguate a timeout from a crash')
        time.sleep(cleanup_timeout)
    elif test_returncode != 0:
        print('FAIL')
    else:
        print('PASS')
    return test_returncode


if __name__ == '__main__':
    sys.exit(main())
