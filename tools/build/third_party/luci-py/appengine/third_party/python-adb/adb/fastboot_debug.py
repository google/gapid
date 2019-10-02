#!/usr/bin/env python
# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Fastboot debugging binary.

Call it similar to how you call android's fastboot. Call it similar to how you
call android's fastboot, but this only accepts usb paths and no serials.
"""
import sys

import gflags
import progressbar

import common_cli
import fastboot

gflags.ADOPT_module_key_flags(common_cli)

FLAGS = gflags.FLAGS


def KwargHandler(kwargs, argspec):

  if 'info_cb' in argspec.args:
    # Use an unbuffered version of stdout.
    def InfoCb(message):
      if not message.message:
        return
      sys.stdout.write('%s: %s\n' % (message.header, message.message))
      sys.stdout.flush()
    kwargs['info_cb'] = InfoCb
  if 'progress_callback' in argspec.args:
    bar = progressbar.ProgessBar(
        widgets=[progressbar.Bar(), progressbar.Percentage()])
    bar.start()
    def SetProgress(current, total):
      bar.update(current / total * 100.0)
      if current == total:
        bar.finish()
    kwargs['progress_callback'] = SetProgress


def main(argv):
  common_cli.StartCli(
      argv, fastboot.FastbootCommands.ConnectDevice,
      list_callback=fastboot.FastbootCommands.Devices,
      kwarg_callback=KwargHandler)


if __name__ == '__main__':
  main(FLAGS(sys.argv))
