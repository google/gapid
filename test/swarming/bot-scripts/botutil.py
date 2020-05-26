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

# This is a  library of utilies for Swarming bot scripts

import json
import os
import tempfile
import subprocess
import sys
import time

def log(msg):
    '''Log the message, making sure to force flushing to stdout'''
    print(msg, flush=True)


def runcmd(cmd):
    '''Run a command, redirecting output to the system stdout and stderr.'''
    return subprocess.run(cmd, stdout=sys.stdout, stderr=sys.stderr)


def adb(args, timeout=1):
    '''Log and run an ADB command, returning a subprocess.CompletedProcess with output captured'''
    cmd = ['adb'] + args
    print('ADB command: ' + ' '.join(cmd), flush=True)
    return subprocess.run(cmd, timeout=timeout, check=True, capture_output=True, text=True)


def load_params(test_params, params_file='params.json', required_keys=[]):
    '''Load the JSON params_file into test_params.

    This overrides the test_params with the values found in params_file. The
    optional required_keys is a list of keys that must be found in params_file.
    '''
    with open(params_file, 'r') as f:
        j = json.load(f)
    for k in required_keys:
        if not k in j.keys():
            raise UserWarning('Missing required key in params.json: {}'.format(k))
    for k in j.keys():
        test_params[k] = j[k]


def is_package_installed(package):
    '''Check if package is installed on the device.'''
    line_to_match = 'package:' + package
    cmd = ['adb', 'shell', 'pm', 'list', 'packages']
    with tempfile.TemporaryFile(mode='w+') as tmp:
        subprocess.run(cmd, timeout=2, check=True, stdout=tmp)
        tmp.seek(0)
        for line in tmp.readlines():
            line = line.rstrip()
            if line == line_to_match:
                return True
    return False


def install_apk(test_params):
    '''Install the test APK

    test_params is a dict where:
    {
      "apk": "foobar.apk", # APK file
      "package": "com.example.foobar", # Package name
      "force_install": true|false, # (Optional): force APK installation,
                                  # even if the package is already found
                                  # on the device
      "install_flags": ["-g", "-t"], # (Opriotnal) list of flags to pass
                                     # to adb install
      ...
    }
'''
    force = False
    if 'force_install' in test_params.keys():
        force = test_params['force_install']
    # -g: grant all needed permissions, -t: accept test APK
    install_flags = ['-g', '-t']
    if 'install_flags' in test_params.keys():
        install_flags = test_params['install_flags']
    if force or not is_package_installed(test_params['package']):
        cmd = ['adb', 'install']
        cmd += install_flags
        cmd += [test_params['apk']]
        log('Install APK with command: ' + ' '.join(cmd))
        # Installing big APKs can take more than a minute, but get also get
        # stuck, so give a big timeout to this command.
        subprocess.run(cmd, timeout=120, check=True, stdout=sys.stdout, stderr=sys.stderr)
        # Sleep a bit, as the app may not be listed right after install
        time.sleep(1)
    else:
        log('Skip install of {} because package {} is already installed.'.format(test_params['apk'], test_params['package']))
