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
import inspect
import logging
import Queue
import struct
import threading
import time

from adb import usb_exceptions


_LOG = logging.getLogger('adb.low')
_LOG.setLevel(logging.ERROR)


class InvalidResponseError(IOError):
  """Got an invalid command over USB."""

  def __init__(self, message, header):
    super(InvalidResponseError, self).__init__('%s: %s' % (message, header))
    self.header = header


def ID2Wire(name):
  assert len(name) == 4 and isinstance(name, str), name
  assert all('A' <= c <= 'Z' for c in name), name
  return sum(ord(c) << (i * 8) for i, c in enumerate(name))


def Wire2ID(encoded):
  assert isinstance(encoded, int), encoded
  name = (
      chr(encoded & 0xff) +
      chr((encoded >> 8) & 0xff) +
      chr((encoded >> 16) & 0xff) +
      chr(encoded >> 24))
  if not all('A' <= c <= 'Z' for c in name):
    return 'XXXX'
  return name


def _CalculateChecksum(data):
  """The checksum is just a sum of all the bytes. I swear."""
  return sum(ord(d) for d in data) & 0xFFFFFFFF


class AuthSigner(object):
  """Signer for use with authenticated ADB, introduced in 4.4.x/KitKat."""

  def Sign(self, data):
    """Signs given data using a private key."""
    raise NotImplementedError()

  def GetPublicKey(self):
    """Returns the public key in PEM format without headers or newlines."""
    raise NotImplementedError()


class _AdbMessageHeader(collections.namedtuple(
    '_AdbMessageHeader',
    ['command', 'arg0', 'arg1', 'data_length', 'data_checksum'])):
  """The raw wire format for the header only.

  Protocol Notes

  local_id/remote_id:
    Turns out the documentation is host/device ambidextrous, so local_id is the
    id for 'the sender' and remote_id is for 'the recipient'. So since we're
    only on the host, we'll re-document with host_id and device_id:

    OPEN(host_id, 0, 'shell:XXX')
    READY/OKAY(device_id, host_id, '')
    WRITE(0, host_id, 'data')
    CLOSE(device_id, host_id, '')
  """
  _VALID_IDS = ('AUTH', 'CLSE', 'CNXN', 'FAIL', 'OKAY', 'OPEN', 'SYNC', 'WRTE')

  # CNXN constants for arg0.
  # If the client initializes a connection to a P+ device with the
  # VERSION_NO_CHECKSUM version, all checksum verifications are skipped and the
  # checksum field in the header is replaced with 0. Since adbd on the device
  # is (hopefully) backwards compatible, use the older version regardless of
  # device OS and continue the old checksum protocol.
  DEFAULT_VERSION = 0x01000000
  VERSION_NO_CHECKSUM = 0x01000001
  SUPPORTED_VERSIONS = (DEFAULT_VERSION, VERSION_NO_CHECKSUM)

  # AUTH constants for arg0.
  AUTH_TOKEN = 1
  AUTH_SIGNATURE = 2
  AUTH_RSAPUBLICKEY = 3

  @classmethod
  def Make(cls, command_name, arg0, arg1, data):
    assert command_name in cls._VALID_IDS
    assert isinstance(arg0, int), arg0
    assert isinstance(arg1, int), arg1
    assert isinstance(data, str), repr(data)
    return cls(
        ID2Wire(command_name), arg0, arg1, len(data), _CalculateChecksum(data))

  @classmethod
  def Unpack(cls, message):
    try:
      command, arg0, arg1, data_length, data_checksum, magic = struct.unpack(
          '<6I', message)
    except struct.error:
      raise InvalidResponseError('Unable to unpack ADB message', message)
    hdr = cls(command, arg0, arg1, data_length, data_checksum)
    expected_magic = command ^ 0xFFFFFFFF
    if magic != expected_magic:
      raise InvalidResponseError(
          'Invalid magic %r != %r' % (magic, expected_magic), hdr)
    if hdr.command_name == 'XXXX':
      raise InvalidResponseError('Unknown command', hdr)
    if hdr.data_length < 0:
      raise InvalidResponseError('Invalid data length', hdr)
    return hdr

  @property
  def Packed(self):
    """Returns this message in an over-the-wire format."""
    magic = self.command ^ 0xFFFFFFFF
    # An ADB message is 6 words in little-endian.
    return struct.pack(
        '<6I', self.command, self.arg0, self.arg1, self.data_length,
        self.data_checksum, magic)

  @property
  def command_name(self):
    return Wire2ID(self.command)

  def str_partial(self):
    command_name = self.command_name
    arg0 = self.arg0
    arg1 = self.arg1
    if command_name == 'AUTH':
      if arg0 == self.AUTH_TOKEN:
        arg0 = 'TOKEN'
      elif arg0 == self.AUTH_SIGNATURE:
        arg0 = 'SIGNATURE'
      elif arg0 == self.AUTH_RSAPUBLICKEY:
        arg0 = 'RSAPUBLICKEY'
      if arg1 != 0:
        raise InvalidResponseError(
            'Unexpected arg1 value (0x%x) on AUTH packet' % arg1, self)
      return '%s, %s' % (command_name, arg0)
    elif command_name == 'CNXN':
      if arg0 == self.DEFAULT_VERSION:
        arg0 = 'v1'
      elif arg0 == self.VERSION_NO_CHECKSUM:
        arg0 = 'v2'
      arg1 = 'pktsize:%d' % arg1
    return '%s, %s, %s' % (command_name, arg0, arg1)

  def __str__(self):
    return '%s, %d' % (self.str_partial(), self.data_length)


class _AdbMessage(object):
  """ADB message class including the data."""
  def __init__(self, header, data=''):
    self.header = header
    self.data = data

  def Write(self, usb, timeout_ms=None):
    """Send this message over USB."""
    # We can't merge these 2 writes, the device wouldn't be able to read the
    # packet.
    try:
      usb.BulkWrite(self.header.Packed, timeout_ms)
      # For data-less headers (eg: OKAY packets) don't send empty data. This has
      # been shown to cause protocol faults on P+ devices. (How did this ever
      # work...?)
      if self.header.data_length:
        usb.BulkWrite(self.data, timeout_ms)
    finally:
      self._log_msg(usb)

  @classmethod
  def Read(cls, usb, timeout_ms=None):
    """Reads one _AdbMessage.

    Returns None if it failed to read the header with a ReadFailedError.
    """
    packet = usb.BulkRead(24, timeout_ms)
    hdr = _AdbMessageHeader.Unpack(packet)
    if hdr.data_length:
      data = usb.BulkRead(hdr.data_length, timeout_ms)
      assert len(data) == hdr.data_length, (len(data), hdr.data_length)
      actual_checksum = _CalculateChecksum(data)
      if actual_checksum != hdr.data_checksum:
        raise InvalidResponseError(
            'Received checksum %s != %s' % (actual_checksum, hdr.data_checksum),
            hdr)
    else:
      data = ''
    msg = cls(hdr, data)
    msg._log_msg(usb)
    return msg

  @classmethod
  def Make(cls, command_name, arg0, arg1, data):
    return cls(_AdbMessageHeader.Make(command_name, arg0, arg1, data), data)

  def _log_msg(self, usb):
    _LOG.debug(
        '%s.%s(%s)',
        '/'.join(str(i) for i in usb.port_path), inspect.stack()[1][3], self)

  def __str__(self):
    if self.data:
      data = repr(self.data)
      if len(data) > 128:
        data = data[:128] + u'\u2026\''
      return '%s, %s' % (self.header.str_partial(), data)
    return self.header.str_partial()


class _AdbConnection(object):
  """One logical ADB connection to a service."""
  class _MessageQueue(object):
    def __init__(self, manager, timeout_ms=None):
      self._queue = Queue.Queue()
      self._manager = manager
      self._timeout_ms = timeout_ms

    def __iter__(self):
      return self

    def next(self):
      while True:
        try:
          i = self._queue.get_nowait()
        except Queue.Empty:
          # Will reentrantly call self._Add() via parent._OnRead()
          if not self._manager.ReadAndDispatch(timeout_ms=self._timeout_ms):
            # Failed to read from the device, the connection likely dropped.
            raise StopIteration()
          continue
        if isinstance(i, StopIteration):
          raise i
        return i

    def _Add(self, message):
      self._queue.put(message)

    def _Close(self):
      self._queue.put(StopIteration())

  def __init__(self, manager, local_id, service_name, timeout_ms=None):
    # ID as given by the remote device.
    self.remote_id = 0
    # Service requested on the remote device.
    self.service_name = service_name
    # Self assigned local ID.
    self._local_id = local_id
    self._yielder = self._MessageQueue(manager, timeout_ms=timeout_ms)
    self._manager = manager

  @property
  def local_id(self):
    """Local connection ID as sent to adbd."""
    return self._local_id

  def __iter__(self):
    # If self._yielder is None, it means it has already closed. Return a fake
    # iterator with nothing in it.
    return self._yielder or []

  def Make(self, command_name, data):
    return _AdbMessage.Make(command_name, self._local_id, self.remote_id, data)

  def _Write(self, command_name, data):
    assert len(data) <= self.max_packet_size, '%d > %d' % (
        len(data), self.max_packet_size)
    self.Make(command_name, data).Write(self._manager._usb)

  def Close(self):
    """User initiated stream close.

    It's rare that the user needs to do this.
    """
    try:
      self._Write('CLSE', '')
      for _ in self:
        pass
    except (usb_exceptions.ReadFailedError, usb_exceptions.WriteFailedError):
      # May get a LIBUSB_ERROR_TIMEOUT
      pass

  @property
  def max_packet_size(self):
    return self._manager.max_packet_size

  @property
  def port_path(self):
    return self._manager.port_path

  def _HasClosed(self):
    """Must be called with the manager lock held."""
    if self._yielder:
      self._yielder._Close()
      self._yielder = None
      self._manager._UnregisterLocked(self._local_id)

  def _OnRead(self, message):
    """Calls from within ReadAndDispatch(), so the manager lock is held."""
    # Can be CLSE, OKAY or WRTE. It's generally basically an ACK.
    cmd_name = message.header.command_name
    if message.header.arg0 != self.remote_id and cmd_name != 'CLSE':
      # We can't assert that for now. TODO(maruel): Investigate the one-off
      # cases.
      logging.warning(
          'Unexpected remote ID: expected %d: %s', self.remote_id, message)
    if message.header.arg1 != self._local_id:
      # As per adb protocol, "A CLOSE message containing a remote-id which
      # does not map to an open stream on the recipient's side is ignored."
      if cmd_name == 'CLSE':
        # It seems adbd on N devices sends duplicate CLSE packets.
        # TODO(bpastene): Find out why/how to detect it.
        return
      raise InvalidResponseError(
          'Unexpected local ID: expected %d' % self._local_id, message)
    if cmd_name == 'CLSE':
      self._HasClosed()
      return
    if cmd_name == 'OKAY':
      self._yielder._Add(message)
      return
    if cmd_name == 'WRTE':
      try:
        self._Write('OKAY', '')
      except usb_exceptions.WriteFailedError as e:
        _LOG.info('%s._OnRead(): Failed to reply OKAY: %s', self.port_path, e)
      self._yielder._Add(message)
      return
    if cmd_name == 'AUTH':
      self._manager._HandleAUTH(message)
      return
    if cmd_name == 'CNXN':
      self._manager.HandleCNXN(message)
      return
    # Unexpected message.
    assert False, message

  # Adaptors.

  def Write(self, data):
    self._Write('WRTE', data)

  def ReadUntil(self, finish_command='WRTE'):
    try:
      with self._manager._lock:
        yielder = self._yielder
      if yielder is None:
        raise InvalidResponseError('Never got \'%s\'' % finish_command, '<N/A>')
      while True:
        message = yielder.next()
        if message.header.command_name == finish_command:
          return message
    except StopIteration:
      raise InvalidResponseError('Never got \'%s\'' % finish_command, '<N/A>')


class AdbConnectionManager(object):
  """Multiplexes the multiple connections."""
  # Maximum amount of data in an ADB packet. Value of MAX_PAYLOAD_V2 in adb.h.
  MAX_ADB_DATA = 256*1024

  def __init__(self, usb, banner, rsa_keys, auth_timeout_ms):
    # Constants.
    self._usb = usb
    self._host_banner = banner
    self._rsa_keys = rsa_keys
    self._auth_timeout_ms = auth_timeout_ms

    self._lock = threading.Lock()
    # As defined by the device.
    self.max_packet_size = 0
    # Banner replied in CNXN packet.
    self.state = None
    # Multiplexed stream handling.
    self._connections = {}
    self._next_local_id = 16

  @classmethod
  def Connect(cls, usb, banner, rsa_keys, auth_timeout_ms):
    """Establish a new connection to the device.

    Args:
      usb: A USBHandle with BulkRead and BulkWrite methods. Takes ownership of
          the handle, it will be closed by this instance.
      rsa_keys: List of AuthSigner subclass instances to be used for
          authentication. The device can either accept one of these via the Sign
          method, or we will send the result of GetPublicKey from the first one
          if the device doesn't accept any of them.
      banner: A string to send as a host identifier.
      auth_timeout_ms: Timeout to wait for when sending a new public key. This
          is only relevant when we send a new public key. The device shows a
          dialog and this timeout is how long to wait for that dialog. If used
          in automation, this should be low to catch such a case as a failure
          quickly; while in interactive settings it should be high to allow
          users to accept the dialog. We default to automation here, so it's low
          by default.
    Returns:
      An AdbConnection.
    """
    assert isinstance(rsa_keys, (list, tuple)), rsa_keys
    assert len(rsa_keys) <= 10, 'adb will sleep 1s after each key above 10'
    # pylint: disable=protected-access
    self = cls(usb, banner, rsa_keys, auth_timeout_ms)
    self._Connect()
    return self

  @property
  def port_path(self):
    return self._usb.port_path

  def Open(self, destination, timeout_ms=None):
    """Opens a new connection to the device via an OPEN message.

    Args:
      destination: The service:command string.

    Returns:
      The local connection object to use.

    Yields:
      The responses from the service if used as such.
    """
    with self._lock:
      next_id = self._next_local_id
      self._next_local_id += 1

    conn = _AdbConnection(self, next_id, destination, timeout_ms=timeout_ms)
    conn._Write('OPEN', destination + '\0')
    with self._lock:
      self._connections[conn.local_id] = conn
    # TODO(maruel): Timeout.
    # Reads until we got the proper remote id.
    while True:
      msg = _AdbMessage.Read(self._usb, timeout_ms)
      if msg.header.arg1 == conn.local_id:
        conn.remote_id = msg.header.arg0
      conn._OnRead(msg)
      if msg.header.arg1 == conn.local_id:
        return conn

  def Close(self):
    """Also closes the usb handle."""
    with self._lock:
      conns = self._connections.values()
    for conn in conns:
      conn._HasClosed()
    with self._lock:
      assert not self._connections, self._connections
    self._usb.Close()

  def StreamingCommand(self, service, command='', timeout_ms=None):
    """One complete set of USB packets for a single connection for a single
    command.

    Sends service:command in a new connection, reading the data for the
    response. All the data is held in memory, large responses will be slow and
    can fill up memory.

    Args:
      service: The service on the device to talk to.
      command: The command to send to the service.
      timeout_ms: Timeout for USB packets, in milliseconds.
    """
    return self.Open('%s:%s' % (service, command), timeout_ms).__iter__()

  def Command(self, service, command='', timeout_ms=None):
    return ''.join(msg.data for msg in self.StreamingCommand(service, command,
                                                             timeout_ms))

  def ReadAndDispatch(self, timeout_ms=None):
    """Receive a response from the device."""
    with self._lock:
      try:
        msg = _AdbMessage.Read(self._usb, timeout_ms)
      except usb_exceptions.ReadFailedError as e:
        # adbd could be rebooting, etc. Return None to signal that this kind of
        # failure is expected.
        _LOG.info(
            '%s.ReadAndDispatch(): Masking read error %s', self.port_path, e)
        return False
      conn = self._connections.get(msg.header.arg1)
      if not conn:
        # It's likely a tored down connection from a previous ADB instance,
        # e.g.  pkill adb.
        # TODO(maruel): It could be a spurious CNXN. In that case we're better
        # to cancel all the known _AdbConnection and start over.
        _LOG.error(
            '%s.ReadAndDispatch(): Got unexpected connection, dropping: %s',
            self.port_path, msg)
        return False
      conn._OnRead(msg)
      return True

  def _Connect(self):
    """Connect to the device."""
    with self._lock:
      reply = None
      start = time.time()
      nb = 0
      _LOG.debug('Emptying the connection first')
      while True:
        try:
          msg = _AdbMessage.Read(self._usb, 20)
        except usb_exceptions.ReadFailedError:
          break
        nb += 1
        if msg.header.command_name in ('AUTH', 'CNXN'):
          # Assert the message has the expected host.
          reply = msg
        else:
          conn = self._connections.get(msg.header.arg1)
          if conn:
            conn._OnRead(msg)
      _LOG.info(
          '%s._Connect(): Flushed %d messages in %.1fs',
          self.port_path, nb, time.time() - start)

      if not reply:
        # Initialize the connection using the older protocol version.
        msg = _AdbMessage.Make(
            'CNXN', _AdbMessageHeader.DEFAULT_VERSION, self.MAX_ADB_DATA,
            'host::%s\0' % self._host_banner)
        msg.Write(self._usb)
        reply = _AdbMessage.Read(self._usb)
      if reply.header.command_name == 'AUTH':
        self._HandleAUTH(reply)
      else:
        self._HandleCNXN(reply)

  def _HandleAUTH(self, reply):
    # self._lock must be held.
    if not self._rsa_keys:
      raise usb_exceptions.DeviceAuthError(
          'Device authentication required, no keys available.')

    # Loop through our keys, signing the last data which is the challenge.
    for rsa_key in self._rsa_keys:
      reply = self._HandleReplyChallenge(rsa_key, reply, self._auth_timeout_ms)
      if reply.header.command_name == 'CNXN':
        break

    if reply.header.command_name == 'AUTH':
      # None of the keys worked, so send a public key. This will prompt to the
      # user.
      msg = _AdbMessage.Make(
          'AUTH', _AdbMessageHeader.AUTH_RSAPUBLICKEY, 0,
          self._rsa_keys[0].GetPublicKey() + '\0')
      msg.Write(self._usb)
      try:
        reply = _AdbMessage.Read(self._usb, self._auth_timeout_ms)
      except usb_exceptions.ReadFailedError as e:
        if e.usb_error.value == -7:  # Timeout.
          raise usb_exceptions.DeviceAuthError(
              'Accept auth key on device, then retry.')
        raise
    self._HandleCNXN(reply)

  def _HandleCNXN(self, reply):
    # self._lock must be held.
    if reply.header.command_name != 'CNXN':
      raise usb_exceptions.DeviceAuthError(
          'Accept auth key on device, then retry.')
    if reply.header.arg0 not in _AdbMessageHeader.SUPPORTED_VERSIONS:
      raise InvalidResponseError(
          'Unsupported protocol version 0x%x in CNXN response' % (
              reply.header.arg0),
          reply)
    self.state = reply.data
    self.max_packet_size = reply.header.arg1
    _LOG.debug(
        '%s._HandleCNXN(): max packet size: %d',
        self.port_path, self.max_packet_size)
    for conn in self._connections.itervalues():
      conn._HasClosed()
    self._connections = {}

  def _HandleReplyChallenge(self, rsa_key, reply, auth_timeout_ms):
    # self._lock must be held.
    if (reply.header.arg0 != _AdbMessageHeader.AUTH_TOKEN or
        reply.header.arg1 != 0 or
        reply.header.data_length != 20 or
        len(reply.data) != 20):
      raise InvalidResponseError('Unknown AUTH response', reply)
    msg = _AdbMessage.Make(
        'AUTH', _AdbMessageHeader.AUTH_SIGNATURE, 0, rsa_key.Sign(reply.data))
    msg.Write(self._usb)
    return _AdbMessage.Read(self._usb, auth_timeout_ms)

  def _Unregister(self, conn_id):
    with self._lock:
      self._UnregisterLocked(conn_id)

  def _UnregisterLocked(self, conn_id):
    # self._lock must be held.
    self._connections.pop(conn_id, None)
