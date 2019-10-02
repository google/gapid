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
"""Common code for ADB and Fastboot CLI.

Usage introspects the given class for methods, args, and docs to show the user.

StartCli handles connecting to a device, calling the expected method, and
outputting the results.
"""

import cStringIO
import inspect
import re
import sys
import types

import gflags

import usb_exceptions

gflags.DEFINE_integer('timeout_ms', 10000, 'Timeout in milliseconds.')
gflags.DEFINE_list('port_path', [], 'USB port path integers (eg 1,2 or 2,1,1)')
gflags.DEFINE_string('serial', None, 'Device serial to look for (host:port or USB serial)', short_name='s')

gflags.DEFINE_bool('output_port_path', False,
                   'Affects the devices command only, outputs the port_path '
                   'alongside the serial if true.')

FLAGS = gflags.FLAGS

_BLACKLIST = {
    'Connect',
    'Close',
    'ConnectDevice',
    'DeviceIsAvailable',
}


def Uncamelcase(name):
  parts = re.split(r'([A-Z][a-z]+)', name)[1:-1:2]
  return ('-'.join(parts)).lower()


def Camelcase(name):
  return name.replace('-', ' ').title().replace(' ', '')


def Usage(adb_dev):
  methods = inspect.getmembers(adb_dev, inspect.ismethod)
  print 'Methods:'
  for name, method in methods:
    if name.startswith('_'):
      continue
    if not method.__doc__:
      continue
    if name in _BLACKLIST:
      continue

    argspec = inspect.getargspec(method)
    args = argspec.args[1:] or ''
    # Surround default'd arguments with []
    defaults = argspec.defaults or []
    if args:
      args = (args[:-len(defaults)] +
              ['[%s]' % arg for arg in args[-len(defaults):]])

      args = ' ' + ' '.join(args)

    print '  %s%s:' % (Uncamelcase(name), args)
    print '    %s' % method.__doc__


def StartCli(argv, device_callback, kwarg_callback=None, list_callback=None,
             **device_kwargs):
  """Starts a common CLI interface for this usb path and protocol."""
  argv = argv[1:]

  if len(argv) == 1 and argv[0] == 'devices' and list_callback is not None:
    # To mimic 'adb devices' output like:
    # ------------------------------
    # List of devices attached
    # 015DB7591102001A        device
    # Or with --output_port_path:
    # 015DB7591102001A        device        1,2
    # ------------------------------
    for device in list_callback():
      if FLAGS.output_port_path:
        print '%s\tdevice\t%s' % (
            device.serial_number,
            ','.join(str(port) for port in device.port_path))
      else:
        print '%s\tdevice' % device.serial_number
    return

  port_path = [int(part) for part in FLAGS.port_path]
  serial = FLAGS.serial

  device_kwargs.setdefault('default_timeout_ms', FLAGS.timeout_ms)
  try:
    dev = device_callback(
        port_path=port_path, serial=serial, banner='python-adb',
        **device_kwargs)
  except usb_exceptions.DeviceNotFoundError as e:
    print >> sys.stderr, 'No device found: %s' % e
    return
  except usb_exceptions.CommonUsbError as e:
    print >> sys.stderr, 'Could not connect to device: %s' % e
    raise

  if not argv:
    Usage(dev)
    return

  kwargs = {}

  # CamelCase method names, eg reboot-bootloader -> RebootBootloader
  method_name = Camelcase(argv[0])
  method = getattr(dev, method_name)
  argspec = inspect.getargspec(method)
  num_args = len(argspec.args) - 1  # self is the first one.
  # Handle putting the remaining command line args into the last normal arg.
  argv.pop(0)

  # Flags -> Keyword args
  if kwarg_callback:
    kwarg_callback(kwargs, argspec)

  try:
    if num_args == 1:
      # Only one argument, so join them all with spaces
      result = method(' '.join(argv), **kwargs)
    else:
      result = method(*argv, **kwargs)

    if result is not None:
      if isinstance(result, cStringIO.OutputType):
        sys.stdout.write(result.getvalue())
      elif isinstance(result, (list, types.GeneratorType)):
        for r in result:
          r = str(r)
          sys.stdout.write(r)
          if not r.endswith('\n'):
            sys.stdout.write('\n')
      else:
        sys.stdout.write(result)
    sys.stdout.write('\n')
  except Exception as e:  # pylint: disable=broad-except
    sys.stdout.write(str(e))
    return
  finally:
    dev.Close()

