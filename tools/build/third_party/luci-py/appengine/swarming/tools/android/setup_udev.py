#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Generates the file /etc/udev/rules.d/android_swarming_bot.rules to enable
automatic Swarming bot to be fired up when an Android device with USB debugging
is connected.
"""

__version__ = '0.1'

import getpass
import optparse
import os
import string
import subprocess
import sys
import tempfile

THIS_FILE = os.path.abspath(__file__)
ROOT_DIR = os.path.dirname(THIS_FILE)

HEADER = '# This file was AUTOMATICALLY GENERATED with %s\n' % THIS_FILE

RULE_FILE = '/etc/udev/rules.d/android_swarming_bot.rules'

LETTERS_AND_DIGITS = frozenset(string.ascii_letters + string.digits)


def gen_udev_rule(user, dev_filters):
  """Generates the content of the udev .rules file."""
  # The command executed must exit immediately.
  script = os.path.join(ROOT_DIR, 'udev_start_bot_deferred.sh')
  items = [
    'ACTION=="add"',
    'SUBSYSTEM=="usb"',
  ]
  items.extend(dev_filters)
  # - sudo -u <user> is important otherwise a user writeable script would be run
  #   as root.
  # - -H makes it easier to find the user's local files.
  # - -E is important, otherwise the necessary udev environment variables won't
  #   be set. Also we don't want to run the script as root.
  items.append('RUN+="/usr/bin/sudo -H -E -u %s %s"' % (user, script))
  line = ', '.join(items)
  # https://code.google.com/p/swarming/issues/detail?id=127
  # TODO(maruel): Create rule for ACTION=="remove" which would send a signal to
  # the currently running process.
  # TODO(maruel): The add rule should try to find a currently running bot first.
  return HEADER + line + '\n'


def write_udev_rule(filepath):
  """Writes the udev rules file in /etc/udev/rules.d when run as root."""
  with open(filepath, 'rb') as f:
    content = f.read()
  if os.path.isfile(RULE_FILE):
    print('Overwritting existing file')
  with open(RULE_FILE, 'w+b') as f:
    f.write(content)
  print('Wrote %d bytes successfully to %s' % (len(content), RULE_FILE))


def work(user, dev_filters):
  """The guts of this script."""
  content = gen_udev_rule(user, dev_filters)
  print('WARNING: About to write in %s:' % RULE_FILE)
  print('***')
  sys.stdout.write(content)
  print('***')
  raw_input('Press enter to continue or Ctrl-C to cancel.')

  handle, filepath = tempfile.mkstemp(
      prefix='swarming_bot_udev', suffix='.rules')
  os.close(handle)
  try:
    with open(filepath, 'w+') as f:
      f.write(content)
      command = ['sudo', sys.executable, THIS_FILE, '--file', filepath]
      print('Running: %s' % ' '.join(command))
    return subprocess.call(command)
  finally:
    os.remove(filepath)


def test_device_rule(device):
  # To find your device:
  # unbuffer udevadm monitor --environment --udev --subsystem-match=usb
  #   | grep DEVNAME
  # udevadm info -a -n <value from DEVNAME>
  #
  # sudo udevadm control --log-priority=debug
  # udevadm info --query all --export-db | less
  cmd = ['sudo', 'udevadm', 'test', '--action=add', device]
  print('Running: %s' % ' '.join(cmd))
  return subprocess.call(cmd)


def main():
  if sys.platform != 'linux2':
    print('Only tested on linux')
    return 1

  parser = optparse.OptionParser(
      description=sys.modules[__name__].__doc__,
      version=__version__)
  parser.add_option('--file', help=optparse.SUPPRESS_HELP)
  parser.add_option(
      '-d', '--dev_filters', default=[], action='append',
      help='udev filters to use; get device id with "lsusb" then udev details '
           'with "udevadm info -a -n /dev/bus/usb/002/001"')
  parser.add_option(
      '--user', default=getpass.getuser(),
      help='User account to start the bot with')
  parser.add_option(
      '--test', help='Tests the rule for a device')
  options, args = parser.parse_args()
  if args:
    parser.error('Unsupported arguments %s' % args)

  if options.test:
    return test_device_rule(options.test)

  if options.file:
    if options.user != 'root':
      parser.error('When --file is used, expected to be run as root')
  else:
    if options.user == 'root':
      parser.error('Run as the user that will be used to run the bot')

  if not LETTERS_AND_DIGITS.issuperset(options.user):
    parser.error('User must be [a-zA-Z0-9]+')

  os.chdir(ROOT_DIR)
  if not os.path.isfile(os.path.join(ROOT_DIR, 'swarming_bot.zip')):
    print('First download swarming_bot.zip aside this script')
    return 1

  if options.file:
    write_udev_rule(options.file)
    return 0

  # 18d1==Google Inc. but we'd likely want to filter more broadly.
  options.dev_filters = options.dev_filters or ['ATTR{idVendor}=="18d1"']
  work(options.user, options.dev_filters)
  return 0


if __name__ == '__main__':
  sys.exit(main())
