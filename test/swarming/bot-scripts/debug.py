#!/usr/bin/env python3

# Copyright 2021 Google LLC
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

# This Swarming bot test script is a basis to run short debug tasks.

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
    test_params = {}
    botutil.load_params(test_params)

    #### Here add your debug experiments
    # For instance, list packages installed on the device:
    botutil.runcmd(['adb', 'shell', 'pm', 'list', 'packages'])


if __name__ == '__main__':
    sys.exit(main())
