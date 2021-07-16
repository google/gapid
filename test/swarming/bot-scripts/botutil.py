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

# This is a library of utilies for Swarming bot scripts

import json
import os
import tempfile
import subprocess
import sys
import time
from shutil import which

def log(msg):
    '''Log the message, making sure to force flushing to stdout'''
    print(msg, flush=True)


def runcmd(cmd):
    '''Log and run a command, redirecting output to the system stdout and stderr.'''
    print('Run command: ' + ' '.join(cmd), flush=True)
    return subprocess.run(cmd, stdout=sys.stdout, stderr=sys.stderr)


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


def is_valid_json(filename):
    '''Return true if filename contains valid JSON, false otherwise'''
    with open(filename, 'r') as f:
        try:
            j = json.load(f)
        except JSONDecodeError as err:
            log('Invalid JSON: {}'.format(err))
            return False
    return True


class BotUtil:
    '''Various utilities that rely on ADB. Since using different ADB commands
    can lead to loosing device connection, this class takes a path to ADB in its
    constructor, and makes sure to use this ADB across all commands.'''

    def __init__(self, adb_path):
        assert(os.path.isfile(adb_path))
        self.adb_path = adb_path
        self.gapit_path = ''

    def adb(self, args, timeout=1):
        '''Log and run an ADB command, returning a subprocess.CompletedProcess with output captured'''
        cmd = [self.adb_path] + args
        print('ADB command: ' + ' '.join(cmd), flush=True)
        return subprocess.run(cmd, timeout=timeout, check=True, capture_output=True, text=True)

    def set_gapit_path(self, gapit_path):
        '''Set path to gapit, must be called once before gapit() can be used.'''
        self.gapit_path = gapit_path

    def gapit(self, verb, args, stdout=sys.stdout, stderr=sys.stderr):
        '''Build and run gapit command. Requires gapit path to be set.'''
        assert(self.gapit_path != '')
        cmd = [self.gapit_path, verb]
        cmd += ['-gapis-args=-adb ' + self.adb_path]
        cmd += args
        print('GAPIT command: ' + ' '.join(cmd), flush=True)
        return subprocess.run(cmd, stdout=stdout, stderr=stderr)

    def is_package_installed(self, package):
        '''Check if package is installed on the device.'''
        line_to_match = 'package:' + package
        cmd = [self.adb_path, 'shell', 'pm', 'list', 'packages']
        with tempfile.TemporaryFile(mode='w+') as tmp:
            subprocess.run(cmd, timeout=2, check=True, stdout=tmp)
            tmp.seek(0)
            for line in tmp.readlines():
                line = line.rstrip()
                if line == line_to_match:
                    return True
        return False

    def install_apk(self, test_params):
        '''Install the test APK

        test_params is a dict where:
        {
        "apk": "foobar.apk", # APK file
        "package": "com.example.foobar", # Package name
        "force_install": true|false, # (Optional): force APK installation,
                                    # even if the package is already found
                                    # on the device
        "install_flags": ["-g", "-t"], # (Optional) list of flags to pass
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
        if force and self.is_package_installed(test_params['package']):
            cmd = [self.adb_path, 'uninstall', test_params['package']]
            log('Force install, start by uninstalling: ' + ' '.join(cmd))
            subprocess.run(cmd, timeout=20, check=True, stdout=sys.stdout, stderr=sys.stderr)
        if force or not self.is_package_installed(test_params['package']):
            cmd = [self.adb_path, 'install']
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
        # Set Android properties specific to this APK.
        if 'setprop' in test_params.keys():
            for prop in test_params['setprop']:
                self.adb(['shell',  'setprop', prop['name'], prop['value']])
