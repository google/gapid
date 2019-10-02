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
"""A libusb1-based ADB reimplementation.

ADB was giving us trouble with its client/server architecture, which is great
for users and developers, but not so great for reliable scripting. This will
allow us to more easily catch errors as Python exceptions instead of checking
random exit codes, and all the other great benefits from not going through
subprocess and a network socket.

All timeouts are in milliseconds.
"""

import cStringIO
import os
import socket

from adb import adb_protocol
from adb import common
from adb import filesync_protocol

# From adb.h
CLASS = 0xFF
SUBCLASS = 0x42
PROTOCOL = 0x01
# pylint: disable=invalid-name
DeviceIsAvailable = common.InterfaceMatcher(CLASS, SUBCLASS, PROTOCOL)


try:
  # Imported locally to keep compatibility with previous code.
  from sign_m2crypto import M2CryptoSigner
except ImportError:
  # Ignore this error when M2Crypto is not installed, there are other options.
  pass


class AdbCommands(object):
  """Exposes adb-like methods for use.

  Some methods are more-pythonic and/or have more options.
  """
  @classmethod
  def ConnectDevice(
      cls, port_path=None, serial=None, default_timeout_ms=None, **kwargs):
    """Convenience function to get an adb device from usb path or serial.

    Args:
      port_path: The filename of usb port to use.
      serial: The serial number of the device to use.
      default_timeout_ms: The default timeout in milliseconds to use.

    If serial specifies a TCP address:port, then a TCP connection is
    used instead of a USB connection.
    """
    if serial and ':' in serial:
      handle = common.TcpHandle(serial)
    else:
      handle = common.UsbHandle.FindAndOpen(
          DeviceIsAvailable, port_path=port_path, serial=serial,
          timeout_ms=default_timeout_ms)
    return cls.Connect(handle, **kwargs)

  def __init__(self, conn):
    self.conn = conn

  @property
  def handle(self):
    return self.conn._usb

  def Close(self):
    self.conn.Close()

  @classmethod
  def Connect(cls, usb, banner, **kwargs):
    """Connect to the device."""
    kwargs['banner'] = banner or socket.gethostname()
    return cls(adb_protocol.AdbConnectionManager.Connect(usb, **kwargs))

  @classmethod
  def Devices(cls):
    """Get a generator of UsbHandle for devices available."""
    return common.UsbHandle.FindDevices(DeviceIsAvailable)

  def GetState(self):
    return self.conn.state

  def Install(self, apk_path, destination_dir=None, timeout_ms=None):
    """Install an apk to the device.

    Doesn't support verifier file, instead allows destination directory to be
    overridden.

    Arguments:
      apk_path: Local path to apk to install.
      destination_dir: Optional destination directory. Use /system/app/ for
        persistent applications.
      timeout_ms: Expected timeout for pushing and installing.

    Returns:
      The pm install output.
    """
    if not destination_dir:
      destination_dir = '/data/local/tmp/'
    basename = os.path.basename(apk_path)
    destination_path = destination_dir + basename
    self.Push(apk_path, destination_path, timeout_ms=timeout_ms)
    return self.Shell('pm install -r "%s"' % destination_path,
                      timeout_ms=timeout_ms)

  def Push(self, source_file, device_filename, mtime='0', timeout_ms=None):
    """Push a file or directory to the device.

    Arguments:
      source_file: Either a filename, a directory or file-like object to push to
                   the device.
      device_filename: The filename on the device to write to.
      mtime: Optional, modification time to set on the file.
      timeout_ms: Expected timeout for any part of the push.
    """
    connection = self.conn.Open(
        destination='sync:', timeout_ms=timeout_ms)
    if isinstance(source_file, basestring):
      source_file = open(source_file)
    filesync_protocol.FilesyncProtocol.Push(
        connection, source_file, device_filename, mtime=int(mtime))
    connection.Close()

  def Pull(self, device_filename, dest_file=None, timeout_ms=None):
    """Pull a file from the device.

    Arguments:
      device_filename: The filename on the device to pull.
      dest_file: If set, a filename or writable file-like object.
      timeout_ms: Expected timeout for any part of the pull.

    Returns:
      The file data if dest_file is not set.
    """
    if isinstance(dest_file, basestring):
      dest_file = open(dest_file, 'w')
    elif not dest_file:
      dest_file = cStringIO.StringIO()
    connection = self.conn.Open(
        destination='sync:', timeout_ms=timeout_ms)
    filesync_protocol.FilesyncProtocol.Pull(
        connection, device_filename, dest_file)
    connection.Close()
    # An empty call to cStringIO.StringIO returns an instance of
    # cStringIO.OutputType.
    if isinstance(dest_file, cStringIO.OutputType):
      return dest_file.getvalue()

  def Stat(self, device_filename):
    """Get a file's stat() information."""
    connection = self.conn.Open(destination='sync:')
    mode, size, mtime = filesync_protocol.FilesyncProtocol.Stat(
        connection, device_filename)
    connection.Close()
    return mode, size, mtime

  def List(self, device_path):
    """Return a directory listing of the given path.

    Returns:
      list of file_sync_protocol.DeviceFile.
    """
    connection = self.conn.Open(destination='sync:')
    listing = filesync_protocol.FilesyncProtocol.List(connection, device_path)
    connection.Close()
    return listing

  def Reboot(self, destination=''):
    """Reboot the device.

    Specify 'bootloader' for fastboot.
    """
    return self.conn.Command(service='reboot', command=destination)

  def RebootBootloader(self):
    """Reboot device into fastboot."""
    return self.Reboot('bootloader')

  def Remount(self):
    """Remount / as read-write."""
    return self.conn.Command(service='remount')

  def Root(self):
    """Restart adbd as root on device."""
    return self.conn.Command(service='root')

  def Unroot(self):
    """Restart adbd as user on device."""
    # adbd implementation of self.conn.Command(service='unroot') is defined in
    # the adb code but doesn't work on 4.4.
    # Until then, emulate Hardcoded strings in
    # platform_system_core/adb/services.cpp. #closeenough
    cmd = (
        'if [ "$(getprop service.adb.root)" == "0" ]; then '
          'echo "adbd not running as root"; '
        'else '
          'setprop service.adb.root 0 && echo "restarting adbd as non root" && '
          'setprop ctl.restart adbd; '
        'fi')
    # adb shell uses CRLF EOL. Only God Knows Why. To add excitment, this is not
    # the case when using a direct service like what Root() is doing, so do the
    # CRLF->LF conversion manually.
    return self.Shell(cmd).replace('\r\n', '\n')

  def Shell(self, command, timeout_ms=None):
    """Run command on the device, returning the output."""
    return self.conn.Command(
        service='shell', command=command, timeout_ms=timeout_ms)

  def StreamingShell(self, command, timeout_ms=None):
    """Run command on the device, yielding each line of output.

    Args:
      command: the command to run on the target.
      timeout_ms: Maximum time to allow the command to run.

    Yields:
      The responses from the shell command.
    """
    return self.conn.StreamingCommand(
        service='shell', command=command, timeout_ms=timeout_ms)

  def Logcat(self, options, timeout_ms=None):
    """Run 'shell logcat' and stream the output to stdout."""
    return self.conn.StreamingCommand(
        service='shell', command='logcat %s' % options, timeout_ms=timeout_ms)
