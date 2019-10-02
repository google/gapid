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
"""A libusb1-based fastboot implementation."""

import binascii
import collections
import cStringIO
import logging
import os
import struct

from adb import common
from adb import usb_exceptions

# The size of packets to write to usb, this is set to 4 for legacy reasons.
# We've had success with 1MB DRASTICALLY decreasing flashing times.
FASTBOOT_WRITE_CHUNK_SIZE_KB = 1024

_LOG = logging.getLogger('fastboot')
_LOG.setLevel(logging.ERROR)

DEFAULT_MESSAGE_CALLBACK = lambda m: logging.info('Got %s from device', m)
FastbootMessage = collections.namedtuple(  # pylint: disable=invalid-name
    'FastbootMessage', ['message', 'header'])

# From fastboot.c
VENDORS = {0x18D1, 0x0451, 0x0502, 0x0FCE, 0x05C6, 0x22B8, 0x0955,
           0x413C, 0x2314, 0x0BB4, 0x8087}
CLASS = 0xFF
SUBCLASS = 0x42
PROTOCOL = 0x03
# pylint: disable=invalid-name
DeviceIsAvailable = common.InterfaceMatcher(CLASS, SUBCLASS, PROTOCOL)


# pylint doesn't understand cross-module exception baseclasses.
# pylint: disable=nonstandard-exception
class FastbootTransferError(usb_exceptions.FormatMessageWithArgumentsException):
  """Transfer error."""


class FastbootRemoteFailure(usb_exceptions.FormatMessageWithArgumentsException):
  """Remote error."""


class FastbootStateMismatch(usb_exceptions.FormatMessageWithArgumentsException):
  """Fastboot and uboot's state machines are arguing. You Lose."""


class FastbootInvalidResponse(
    usb_exceptions.FormatMessageWithArgumentsException):
  """Fastboot responded with a header we didn't expect."""


class FastbootProtocol(object):
  """Encapsulates the fastboot protocol."""
  FINAL_HEADERS = {'OKAY', 'DATA'}

  def __init__(self, usb):
    """Constructs a FastbootProtocol instance.

    Arguments:
      usb: UsbHandle instance.
    """
    self.usb = usb

  @property
  def usb_handle(self):
    return self.usb

  def SendCommand(self, command, arg=None):
    """Sends a command to the device.

    Args:
      command: The command to send.
      arg: Optional argument to the command.
    """
    if arg is not None:
      command = '%s:%s' % (command, arg)
    self._Write(cStringIO.StringIO(command), len(command))

  def HandleSimpleResponses(
      self, timeout_ms=None, info_cb=DEFAULT_MESSAGE_CALLBACK):
    """Accepts normal responses from the device.

    Args:
      timeout_ms: Timeout in milliseconds to wait for each response.
      info_cb: Optional callback for text sent from the bootloader.

    Returns:
      OKAY packet's message.
    """
    return self._AcceptResponses('OKAY', info_cb, timeout_ms=timeout_ms)

  def HandleDataSending(self, source_file, source_len,
                        info_cb=DEFAULT_MESSAGE_CALLBACK,
                        progress_callback=None, timeout_ms=None):
    """Handles the protocol for sending data to the device.

    Arguments:
      source_file: File-object to read from for the device.
      source_len: Amount of data, in bytes, to send to the device.
      info_cb: Optional callback for text sent from the bootloader.
      progress_callback: Callback that takes the current and the total progress
        of the current file.
      timeout_ms: Timeout in milliseconds to wait for each response.

    Raises:
      FastbootTransferError: When fastboot can't handle this amount of data.
      FastbootStateMismatch: Fastboot responded with the wrong packet type.
      FastbootRemoteFailure: Fastboot reported failure.
      FastbootInvalidResponse: Fastboot responded with an unknown packet type.

    Returns:
      OKAY packet's message.
    """
    accepted_size = self._AcceptResponses(
        'DATA', info_cb, timeout_ms=timeout_ms)

    accepted_size = binascii.unhexlify(accepted_size[:8])
    accepted_size, = struct.unpack('>I', accepted_size)
    if accepted_size != source_len:
      raise FastbootTransferError(
          'Device refused to download %s bytes of data (accepts %s bytes)',
          source_len, accepted_size)
    self._Write(source_file, accepted_size, progress_callback)
    return self._AcceptResponses('OKAY', info_cb, timeout_ms=timeout_ms)

  def _AcceptResponses(self, expected_header, info_cb, timeout_ms=None):
    """Accepts responses until the expected header or a FAIL.

    Arguments:
      expected_header: OKAY or DATA
      info_cb: Optional callback for text sent from the bootloader.
      timeout_ms: Timeout in milliseconds to wait for each response.

    Raises:
      FastbootStateMismatch: Fastboot responded with the wrong packet type.
      FastbootRemoteFailure: Fastboot reported failure.
      FastbootInvalidResponse: Fastboot responded with an unknown packet type.

    Returns:
      OKAY packet's message.
    """
    while True:
      response = self.usb.BulkRead(64, timeout_ms=timeout_ms)
      header = response[:4]
      remaining = response[4:]

      if header == 'INFO':
        info_cb(FastbootMessage(remaining, header))
      elif header in self.FINAL_HEADERS:
        if header != expected_header:
          raise FastbootStateMismatch(
              'Expected %s, got %s', expected_header, header)
        if header == 'OKAY':
          info_cb(FastbootMessage(remaining, header))
        return remaining
      elif header == 'FAIL':
        info_cb(FastbootMessage(remaining, header))
        raise FastbootRemoteFailure('FAIL: %s', remaining)
      else:
        raise FastbootInvalidResponse(
            'Got unknown header %s and response %s', header, remaining)

  def _HandleProgress(self, total, progress_callback):
    """Calls the callback with the current progress and total ."""
    current = 0
    while True:
      current += yield
      try:
        progress_callback(current, total)
      except Exception:  # pylint: disable=broad-except
        _LOG.exception('Progress callback raised an exception. %s',
                       progress_callback)
        continue

  def _Write(self, data, length, progress_callback=None):
    """Sends the data to the device, tracking progress with the callback."""
    if progress_callback:
      progress = self._HandleProgress(length, progress_callback)
      progress.next()
    while length:
      tmp = data.read(FASTBOOT_WRITE_CHUNK_SIZE_KB * 1024)
      length -= len(tmp)
      self.usb.BulkWrite(tmp)

      if progress_callback:
        progress.send(len(tmp))


class FastbootCommands(object):
  """Encapsulates the fastboot commands."""
  protocol_handler = FastbootProtocol

  def __init__(self, usb):
    """Constructs a FastbootCommands instance.

    Arguments:
      usb: UsbHandle instance.
    """
    self._usb = usb
    self._protocol = self.protocol_handler(usb)

  @property
  def usb_handle(self):
    return self._usb

  def Close(self):
    self._usb.Close()

  @classmethod
  def ConnectDevice(
      cls, port_path=None, serial=None, default_timeout_ms=None):
    """Convenience function to get an adb device from usb path or serial."""
    usb = common.UsbHandle.FindAndOpen(
        DeviceIsAvailable, port_path=port_path, serial=serial,
        timeout_ms=default_timeout_ms)
    return cls(usb)

  @classmethod
  def Devices(cls):
    """Get a generator of UsbHandle for devices available."""
    return common.UsbHandle.FindDevices(DeviceIsAvailable)

  def _SimpleCommand(self, command, arg=None, **kwargs):
    self._protocol.SendCommand(command, arg)
    return self._protocol.HandleSimpleResponses(**kwargs)

  def FlashFromFile(self, partition, source_file, source_len=0,
                    info_cb=DEFAULT_MESSAGE_CALLBACK, progress_callback=None):
    """Flashes a partition from the file on disk.

    Args:
      partition: Partition name to flash to.
      source_file: Filename to download to the device.
      source_len: Optional length of source_file, uses os.stat if not provided.
      info_cb: See Download.
      progress_callback: See Download.

    Returns:
      Download and flash responses, normally nothing.
    """
    if source_len == 0:
      # Fall back to stat.
      source_len = os.stat(source_file).st_size
    download_response = self.Download(
        source_file, source_len=source_len, info_cb=info_cb,
        progress_callback=progress_callback)
    flash_response = self.Flash(partition, info_cb=info_cb)
    return download_response + flash_response

  def Download(self, source_file, source_len=0,
               info_cb=DEFAULT_MESSAGE_CALLBACK, progress_callback=None):
    """Downloads a file to the device.

    Args:
      source_file: A filename or file-like object to download to the device.
      source_len: Optional length of source_file. If source_file is a file-like
          object and source_len is not provided, source_file is read into
          memory.
      info_cb: Optional callback accepting FastbootMessage for text sent from
          the bootloader.
      progress_callback: Optional callback called with the percent of the
          source_file downloaded. Note, this doesn't include progress of the
          actual flashing.

    Returns:
      Response to a download request, normally nothing.
    """
    if isinstance(source_file, basestring):
      source_len = os.stat(source_file).st_size
      source_file = open(source_file)

    if source_len == 0:
      # Fall back to storing it all in memory :(
      data = source_file.read()
      source_file = cStringIO.StringIO(data)
      source_len = len(data)

    self._protocol.SendCommand('download', '%08x' % source_len)
    return self._protocol.HandleDataSending(
        source_file, source_len, info_cb, progress_callback=progress_callback)

  def Flash(self, partition, timeout_ms=0, info_cb=DEFAULT_MESSAGE_CALLBACK):
    """Flashes the last downloaded file to the given partition.

    Args:
      partition: Partition to flash.
      timeout_ms: Optional timeout in milliseconds to wait for it to finish.
      info_cb: See Download. Usually no messages.

    Returns:
      Response to a download request, normally nothing.
    """
    return self._SimpleCommand('flash', arg=partition, info_cb=info_cb,
                               timeout_ms=timeout_ms)

  def Erase(self, partition, timeout_ms=None):
    """Erases the given partition."""
    self._SimpleCommand('erase', arg=partition, timeout_ms=timeout_ms)

  def Getvar(self, var, info_cb=DEFAULT_MESSAGE_CALLBACK):
    """Returns the given variable's definition.

    Args:
      var: A variable the bootloader tracks, such as version.
      info_cb: See Download. Usually no messages.
    Returns:
      Value of var according to the current bootloader.
    """
    return self._SimpleCommand('getvar', arg=var, info_cb=info_cb)

  def Oem(self, command, timeout_ms=None, info_cb=DEFAULT_MESSAGE_CALLBACK):
    """Executes an OEM command on the device.

    Args:
      command: The command to execute, such as 'poweroff' or 'bootconfig read'.
      timeout_ms: Optional timeout in milliseconds to wait for a response.
      info_cb: See Download. Messages vary based on command.
    Returns:
      The final response from the device.
    """
    return self._SimpleCommand(
        'oem %s' % command, timeout_ms=timeout_ms, info_cb=info_cb)

  def Continue(self):
    """Continues execution past fastboot into the system."""
    return self._SimpleCommand('continue')

  def Reboot(self, target_mode=None, timeout_ms=None):
    """Reboots the device.

    Args:
        target_mode: Normal reboot when unspecified (or None). Can specify
            other target modes, such as 'recovery' or 'bootloader'.
        timeout_ms: Optional timeout in milliseconds to wait for a response.
    Returns:
        Usually the empty string. Depends on the bootloader and the target_mode.
    """
    return self._SimpleCommand('reboot', arg=target_mode, timeout_ms=timeout_ms)

  def RebootBootloader(self, timeout_ms=None):
    """Reboots into the bootloader, usually equiv to Reboot('bootloader')."""
    return self._SimpleCommand('reboot-bootloader', timeout_ms=timeout_ms)
