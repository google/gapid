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
"""ADB protocol implementation.

Implements the ADB protocol as seen in android's adb/adbd binaries, but only the
host side.
"""

import collections
import stat
import struct
import time

import libusb1

from adb import adb_protocol
from adb import usb_exceptions


class PushFailedError(usb_exceptions.AdbCommandFailureException):
  """Pushing a file failed for some reason."""


DeviceFile = collections.namedtuple('DeviceFile', [
    'filename', 'mode', 'size', 'mtime'])


class FilesyncProtocol(object):
  """Implements the FileSync protocol as described in ../filesync_protocol.txt.

  TODO(maruel): Make these functions async.
  """
  # Maximum size of a filesync DATA packet; file_sync_service.h
  SYNC_DATA_MAX = 64*1024
  # Default mode for pushed files.
  DEFAULT_PUSH_MODE = stat.S_IFREG | stat.S_IRWXU | stat.S_IRWXG

  @staticmethod
  def Stat(connection, filename):
    if isinstance(filename, unicode):
      filename = filename.encode('utf-8')
    cnxn = FileSyncConnection(connection, '<4I')
    cnxn.Send('STAT', filename)
    command, (mode, size, mtime) = cnxn.ReadNoData(('STAT',))

    if command != 'STAT':
      raise adb_protocol.InvalidResponseError(
          'Expected STAT response to STAT, got %s' % command)
    return mode, size, mtime

  @classmethod
  def List(cls, connection, path):
    if isinstance(path, unicode):
      path = path.encode('utf-8')
    cnxn = FileSyncConnection(connection, '<5I')
    cnxn.Send('LIST', path)
    files = []
    for cmd_id, header, filename in cnxn.ReadUntil(('DENT',), 'DONE'):
      if cmd_id == 'DONE':
        break
      mode, size, mtime = header
      files.append(DeviceFile(filename, mode, size, mtime))
    return files

  @classmethod
  def Pull(cls, connection, filename, dest_file):
    """Pull a file from the device into the file-like dest_file."""
    if isinstance(filename, unicode):
      filename = filename.encode('utf-8')
    cnxn = FileSyncConnection(connection, '<2I')
    cnxn.Send('RECV', filename)
    for cmd_id, _, data in cnxn.ReadUntil(('DATA',), 'DONE'):
      if cmd_id == 'DONE':
        break
      dest_file.write(data)

  @classmethod
  def Push(cls, connection, datafile, filename,
           st_mode=DEFAULT_PUSH_MODE, mtime=0):
    """Push a file-like object to the device.

    Args:
      connection: ADB connection
      datafile: File-like object for reading from
      filename: Filename to push to
      st_mode: stat mode for filename
      mtime: modification time

    Raises:
      PushFailedError: Raised on push failure.
    """
    if isinstance(filename, unicode):
      filename = filename.encode('utf-8')
    fileinfo = '%s,%s' % (filename, st_mode)
    assert len(filename) <= 1024, 'Name too long: %s' % filename

    cnxn = FileSyncConnection(connection, '<2I')
    cnxn.Send('SEND', fileinfo)

    while True:
      data = datafile.read(cls.SYNC_DATA_MAX)
      if not data:
        break
      cnxn.Send('DATA', data)

    if mtime == 0:
      mtime = int(time.time())
    # DONE doesn't send data, but it hides the last bit of data in the size
    # field. #youhadonejob
    cnxn.Send('DONE', size=mtime)
    for cmd_id, _, data in cnxn.ReadUntil((), 'OKAY', 'DATA', 'FAIL'):
      if cmd_id == 'OKAY':
        return
      if cmd_id == 'DATA':
        # file_sync_client.cpp CopyDone ignores the cmd_id in this case.
        raise PushFailedError(data)
      if cmd_id == 'FAIL':
        raise PushFailedError(data)
      raise PushFailedError('Unexpected message %s: %s' % (cmd_id, data))



class FileSyncConnection(object):
  """Encapsulate a FileSync service connection."""

  _VALID_IDS = [
      'STAT', 'LIST', 'SEND', 'RECV', 'DENT', 'DONE', 'DATA', 'OKAY',
      'FAIL', 'QUIT',
  ]

  def __init__(self, adb_connection, recv_header_format):
    self.adb = adb_connection

    # Sending
    self.send_buffer = ''
    self.send_header_len = struct.calcsize('<2I')

    # Receiving
    self.recv_buffer = ''
    self.recv_header_format = recv_header_format
    self.recv_header_len = struct.calcsize(recv_header_format)

  def Send(self, command_id, data='', size=0):
    """Send/buffer FileSync packets.

    Packets are buffered and only flushed when this connection is read from. All
    messages have a response from the device, so this will always get flushed.

    Args:
      command_id: Command to send.
      data: Optional data to send, must set data or size.
      size: Optionally override size from len(data).
    """
    if data:
      size = len(data)
    header = struct.pack('<2I', adb_protocol.ID2Wire(command_id), size)
    self.send_buffer += header + data

  def Read(self, expected_ids):
    """Read ADB messages and return FileSync packets."""
    self._Flush()

    # Read one filesync packet off the recv buffer.
    header_data = self._ReadBuffered(self.recv_header_len)
    header = struct.unpack(self.recv_header_format, header_data)

    # Header is (ID, ..., size).
    size = header[-1]
    data = self._ReadBuffered(size)
    command_id = self._VerifyReplyCommand(header, expected_ids)
    return command_id, header[1:-1], data

  def ReadNoData(self, expected_ids):
    """Read ADB messages and return FileSync packets.

    This is for special packets that do not return data.
    """
    self._Flush()

    # Read one filesync packet off the recv buffer.
    header_data = self._ReadBuffered(self.recv_header_len)
    header = struct.unpack(self.recv_header_format, header_data)
    command_id = self._VerifyReplyCommand(header, expected_ids)
    return command_id, header[1:]

  def ReadUntil(self, expected_ids, *finish_ids):
    """Useful wrapper around Read."""
    while True:
      cmd_id, header, data = self.Read(expected_ids + finish_ids)
      yield cmd_id, header, data
      if cmd_id in finish_ids:
        break

  def _Flush(self):
    while self.send_buffer:
      chunk = self.send_buffer[:self.adb.max_packet_size]
      try:
        self.adb.Write(chunk)
        # Wait for ack from device, ignoring these for too long causes things
        # to explode.
        self.adb.ReadUntil('OKAY')
      except (libusb1.USBError, adb_protocol.InvalidResponseError) as e:
        self.send_buffer = ''
        raise usb_exceptions.WriteFailedError('Could not write %r' % chunk, e)
      self.send_buffer = self.send_buffer[self.adb.max_packet_size:]

  def _ReadBuffered(self, size):
    # Ensure recv buffer has enough data.
    while len(self.recv_buffer) < size:
      try:
        msg = self.adb.ReadUntil('WRTE')
      except adb_protocol.InvalidResponseError as e:
        raise usb_exceptions.AdbCommandFailureException(
          'Command failed: %s' % e)
      self.recv_buffer += msg.data

    result = self.recv_buffer[:size]
    self.recv_buffer = self.recv_buffer[size:]
    return result

  @classmethod
  def _VerifyReplyCommand(cls, header, expected_ids):
    # Header is (ID, ...).
    command_id = adb_protocol.Wire2ID(header[0])
    if command_id not in cls._VALID_IDS:
      raise usb_exceptions.AdbCommandFailureException(
          'Command failed; incorrect header: %s' % header)
    if command_id not in expected_ids:
      if command_id == 'FAIL':
        raise usb_exceptions.AdbCommandFailureException('Command failed.')
      raise adb_protocol.InvalidResponseError(
          'Expected one of %s, got %s' % (expected_ids, command_id))
    return command_id
