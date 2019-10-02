# Copyright 2015 Google Inc. All rights reserved.
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

"""Defines AdbCommandsSafe, an exception safe version of AdbCommands."""

import cStringIO
import inspect
import logging
import socket
import subprocess
import time
import uuid

from adb import adb_commands
from adb import adb_protocol
from adb import common
from adb import usb_exceptions


_LOG = logging.getLogger('adb.cmd')
_LOG.setLevel(logging.ERROR)


### Public API.


# Make adb_commands_safe a drop-in replacement for adb_commands.
from adb.adb_commands import DeviceIsAvailable


def KillADB():
  """Stops the adb daemon.

  It's possible that adb daemon could be alive on the same host where python-adb
  is used. The host keeps the USB devices open so it's not possible for other
  processes to open it. Gently stop adb so this process can access the USB
  devices.

  adb's stability is less than stellar. Kill it with fire.
  """
  _LOG.info('KillADB()')
  attempts = 10
  for _ in xrange(attempts):
    try:
      subprocess.check_output(['pgrep', 'adb'])
    except subprocess.CalledProcessError:
      return
    try:
      subprocess.call(
          ['adb', 'kill-server'],
          stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    except OSError:
      pass
    subprocess.call(
        ['killall', '--exact', 'adb'],
        stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    # Force thread scheduling to give a chance to the OS to clean out the
    # process.
    time.sleep(0.001)

  try:
    processes = subprocess.check_output(['ps', 'aux']).splitlines()
  except subprocess.CalledProcessError:
    _LOG.error('KillADB(): unable to scan process list.')
    processes = []

  culprits = '\n'.join(p for p in processes if 'adb' in p)
  _LOG.error(
      'KillADB() failed after %d attempts. Potential culprits: %s',
      attempts, culprits)


class AdbCommandsSafe(object):
  """Wraps an AdbCommands to make it exception safe.

  The fact that exceptions can be thrown any time makes the client code really
  hard to write safely. Convert USBError* to None return value.

  Only contains the low level commands. High level operations are built upon the
  low level functionality provided by this class.
  """
  # - IOError includes usb_exceptions.CommonUsbError, which means that device
  #   I/O failed, e.g.  a write or a read call returned an error. It also
  #   includes adb_protocol.InvalidResponseError which happens if the
  #   communication becomes desynchronized.
  # - USBError means that a bus I/O failed, e.g. the device path is not present
  #   anymore. It is sometimes thrown as USBErrorIO.
  _ERRORS = (
      IOError,
      common.usb1.USBError,
      common.usb1.USBErrorIO)

  _SHELL_SUFFIX = ' ;echo -e "\n$?"'

  def __init__(
      self, handle, banner, rsa_keys, on_error, port_path=None,
      default_timeout_ms=10000, auth_timeout_ms=10000, lost_timeout_ms=10000,
      enable_resets=False):
    """Constructs an AdbCommandsSafe.

    Arguments:
    - port_path: str addressing the device on the USB bus, e.g. '1/2'.
    - handle: common.UsbHandle or None.
    - banner: How the app present itself to the device. This affects
          authentication so it is better to use an hardcoded constant.
    - rsa_keys: list of AuthSigner.
    - on_error: callback to call in case of error.
    - default_timeout_ms: Timeout for adbd to reply to a command.
    - auth_timeout_ms: Timeout for the user to accept the dialog.
    - lost_timeout_ms: Duration to wait for the device to come back alive when
          either adbd or the phone is rebooting.
    """
    assert isinstance(auth_timeout_ms, int), auth_timeout_ms
    assert isinstance(default_timeout_ms, int), default_timeout_ms
    assert isinstance(lost_timeout_ms, int), lost_timeout_ms
    assert isinstance(banner, str), banner
    assert on_error is None or callable(on_error), on_error
    assert handle is None or isinstance(handle, common.Handle), handle
    assert all('\n' not in r.GetPublicKey() for r in rsa_keys), rsa_keys

    port_path = handle.port_path if handle else port_path

    _LOG.debug(
        'AdbCommandsSafe(%s, %s, %s, %s, %s, %s, %s, %s)',
        port_path, handle, banner, rsa_keys, on_error, default_timeout_ms,
        auth_timeout_ms, lost_timeout_ms)
    # Immutable.
    self._auth_timeout_ms = auth_timeout_ms
    self._default_timeout_ms = default_timeout_ms
    self._banner = banner or socket.gethostname()
    self._lost_timeout_ms = lost_timeout_ms
    self._on_error = on_error
    self._rsa_keys = rsa_keys
    self._sleep = 0.1
    self._tries = int(round((self._lost_timeout_ms / 1000. + 5) / self._sleep))
    self._should_reset = enable_resets

    # State.
    self._adb_cmd = None
    self._needs_su = False
    self._serial = None
    self._failure = None
    self._handle = handle
    self._port_path = '/'.join(str(p) for p in port_path) if port_path else None

  @classmethod
  def ConnectDevice(cls, port_path, **kwargs):
    """Return a AdbCommandsSafe for a USB device referenced by the port path.

    Arguments:
    - port_path: str in form '1/2' to refer to a connected but unopened USB
          device.
    - The rest are the same as __init__().
    """
    # pylint: disable=protected-access
    obj = cls(port_path=port_path, handle=None, **kwargs)
    if obj._WaitUntilFound(use_serial=False):
      if obj._OpenHandle():
        obj._Connect(use_serial=False)
    return obj

  @classmethod
  def Connect(cls, handle, **kwargs):
    """Return a AdbCommandsSafe for a USB device referenced by a handle.

    Arguments:
    - handle: an opened or unopened common.Handle.
    - The rest are the same as __init__().

    Returns:
      AdbCommandsSafe.
    """
    # pylint: disable=protected-access
    obj = cls(handle=handle, **kwargs)
    if not handle.is_open:
      obj._OpenHandle()
    if obj._handle:
      obj._Connect(use_serial=False)
    return obj

  @property
  def is_valid(self):
    return bool(self._adb_cmd)

  @property
  def failure(self):
    return self._failure

  @property
  def max_packet_size(self):
    if not self._adb_cmd:
      return None
    return self._adb_cmd.conn.max_packet_size

  @property
  def port_path(self):
    """Returns the USB port path as a str."""
    return self._port_path

  @property
  def sysfs_port_path(self):
    if self._handle and self._handle.is_open:
      return self._handle.sysfs_port_path

  @property
  def public_keys(self):
    """Returns the list of the public keys."""
    return [r.GetPublicKey() for r in self._rsa_keys]

  @property
  def serial(self):
    return self._serial

  def Close(self):
    if self._adb_cmd:
      self._adb_cmd.Close()
      self._adb_cmd = None
      self._handle = None
    elif self._handle:
      self._handle.Close()
      self._handle = None

  def GetUptime(self):
    """Returns the device's uptime in second."""
    # This is an high level functionality but is needed by self.Reboot().
    out, _ = self.Shell('cat /proc/uptime')
    if out:
      return float(out.split()[0])
    return None

  def List(self, destdir):
    """List a directory on the device.

    Returns:
      list of file_sync_protocol.DeviceFile.
    """
    assert destdir.startswith('/'), destdir
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          return self._adb_cmd.List(destdir)
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(%s): %s', destdir, e):
            break
    return None

  def Stat(self, dest):
    """Stats a file/dir on the device. It's likely faster than shell().

    Returns:
      tuple(mode, size, mtime)
    """
    assert dest.startswith('/'), dest
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          return self._adb_cmd.Stat(dest)
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(%s): %s', dest, e):
            break
    return None, None, None

  def Pull(self, remotefile, dest):
    """Retrieves a file from the device to dest on the host.

    Returns True on success.
    """
    assert remotefile.startswith('/'), remotefile
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          self._adb_cmd.Pull(remotefile, dest)
          return True
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(%s, %s): %s', remotefile, dest, e):
            break
    return False

  def PullContent(self, remotefile):
    """Reads a file from the device.

    Returns content on success as str, None on failure.
    """
    assert remotefile.startswith('/'), remotefile
    if self._adb_cmd:
      # TODO(maruel): Distinction between file is not present and I/O error.
      for _ in self._Loop():
        try:
          if not self._needs_su:
            return self._adb_cmd.Pull(remotefile, None)
          else:
            # If we need su to be root, we likely can't use adb's filesync. So
            # cat the file instead as a normal shell command and return the
            # output.
            out, exit_code = self.Shell('cat %s' % remotefile)
            if exit_code:
              return None
            return out
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(%s): %s', remotefile, e):
            break
    return None

  def Push(self, localfile, dest, mtime='0'):
    """Pushes a local file to dest on the device.

    Returns True on success.
    """
    assert dest.startswith('/'), dest
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          self._adb_cmd.Push(localfile, dest, mtime)
          return True
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(%s, %s): %s', localfile, dest, e):
            break
    return False

  def PushContent(self, content, dest, mtime='0'):
    """Writes content to dest on the device.

    Returns True on success.
    """
    assert dest.startswith('/'), dest
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          if dest.startswith('/data/local/tmp') or not self._needs_su:
            self._adb_cmd.Push(cStringIO.StringIO(content), dest, mtime)
          else:
            # If we need to use "su" to access root privileges, first push the
            # file to a world-writable dir, then use "su" to move the file
            # over.
            tmp_dest = '/data/local/tmp/%s' % uuid.uuid4()
            self._adb_cmd.Push(cStringIO.StringIO(content), tmp_dest, mtime)
            _, exit_code = self.Shell('mv %s %s' % (tmp_dest, dest))
            if exit_code:
              return False
          return True
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(%s, %s): %s', dest, content, e):
            break
    return False

  def Reboot(self, wait=True, force=False):
    """Reboots the device. Waits for it to be rebooted but not fully
    operational.

    This causes the USB device to disapear momentarily and get a new port_path.
    If the phone just booted up, this function will cause the caller to sleep.

    The caller will likely want to call self.Root() to switch adbd to root
    context if desired.

    Arguments:
    - wait: If true, attempts to reconnect to the device after sending the
        reboot. Otherwise, exit early.
    - force: If true, attempts to force a reboot via the /proc/sysrq-trigger
        file if the initial command fails. See
        https://android.googlesource.com/kernel/common/+/android-3.10.y/Documentation/sysrq.txt
        for more info.

    Returns True on success.
    """
    if self._adb_cmd:
      if not wait:
        return self._Reboot()

      # Get previous uptime to ensure the phone actually rebooted.
      previous_uptime = self.GetUptime()
      if not previous_uptime:
        return False
      if previous_uptime <= 30.:
        # Wait for uptime to be high enough. Otherwise we can't know if the
        # device rebooted or not.
        time.sleep(31. - previous_uptime)
        previous_uptime = self.GetUptime()
        if not previous_uptime:
          return False

      start = time.time()
      if not self._Reboot():
        return False

      # There's no need to loop initially too fast. Restarting the phone
      # always take several tens of seconds at best.
      time.sleep(5.)

      def _wait_to_bootup():
        for _ in self._Loop(timeout=60000):
          if not self._Reconnect(True, timeout=60000):
            continue
          uptime = self.GetUptime()
          if uptime and uptime < previous_uptime:
            return True
          time.sleep(0.1)
        return False

      if _wait_to_bootup():
        return True

      if force:
        if self.IsRoot() or self.Root():
          # Ignore exit code and stdout/stderr since this will theoretically
          # cause the device to immediately reboot.
          self.Shell('echo b > /proc/sysrq-trigger')
          if _wait_to_bootup():
            return True

      _LOG.error(
          '%s.Reboot(): Failed to reboot after %.2fs', self.port_path,
          time.time() - start)
    return False

  def Remount(self):
    """Remount / as read-write."""
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          out = self._adb_cmd.Remount()
          # TODO(maruel): Wait for the remount operation to be completed.
          _LOG.info('%s.Remount(): %s', self.port_path, out)
          return True
        except usb_exceptions.AdbCommandFailureException:
          break
        except self._ERRORS as e:
          if not self._Reset('(): %s', e):
            break
    return False

  def ResetHandle(self, new_endpoint=None):
    """Resets the handle used to communicate to the device.

    For USB connections, it resets the usb device. For TCP connections,
    it closes and reopens the TCP connection.

    Args:
      new_endpoint: If a TCP device, this will be used as the new endpoint in
                    the TCP connection.
    Returns:
      True on success.
    """
    if self._adb_cmd:
      self._adb_cmd.Close()
      self._adb_cmd = None
    if new_endpoint:
      self._serial = None
    self._handle.Reset(new_endpoint=new_endpoint)
    return self._Connect(False)

  def Shell(self, cmd, timeout_ms=None):
    """Runs a command on an Android device while swallowing exceptions.

    Traps all kinds of USB errors so callers do not have to handle this.

    Returns:
      tuple(stdout, exit_code)
      - stdout is as unicode if it ran, None if an USB error occurred.
      - exit_code is set if ran.
    """
    if self._needs_su:
      cmd = 'su root ' + cmd
    if self._adb_cmd:
      for _ in self._Loop():
        try:
          return self.ShellRaw(cmd, timeout_ms=timeout_ms)
        except self._ERRORS as e:
          if not self._Reset('(%s): %s', cmd, e):
            break
    return None, None

  def IsShellOk(self, cmd):
    """Returns True if the shell command can be sent."""
    if isinstance(cmd, unicode):
      cmd = cmd.encode('utf-8')
    assert isinstance(cmd, str), cmd
    if not self._adb_cmd:
      return True
    cmd_size = len(cmd + self._SHELL_SUFFIX)
    pkt_size = self.max_packet_size - len('shell:')
    # Has to keep one byte for trailing nul byte.
    return cmd_size < pkt_size

  def ShellRaw(self, cmd, timeout_ms=None):
    """Runs a command on an Android device.

    It is expected that the user quote cmd properly.

    It fails if cmd is too long.

    Returns:
      tuple(stdout, exit_code)
      - stdout is as unicode if it ran, None if an USB error occurred.
      - exit_code is set if ran.
    """
    if isinstance(cmd, unicode):
      cmd = cmd.encode('utf-8')
    assert isinstance(cmd, str), cmd
    if not self._adb_cmd:
      return None, None
    # The adb protocol doesn't return the exit code, so embed it inside the
    # command.
    assert self.IsShellOk(cmd), 'Command is too long: %r' % cmd
    if timeout_ms is None:
      timeout_ms = self._default_timeout_ms
    out = self._adb_cmd.Shell(
        cmd + self._SHELL_SUFFIX,
        timeout_ms=timeout_ms).decode('utf-8', 'replace')
    # Protect against & or other bash conditional execution that wouldn't make
    # the 'echo $?' command to run.
    if not out:
      return out, None
    # adb shell uses CRLF EOL. Only God Knows Why.
    out = out.replace('\r\n', '\n')
    exit_code = None
    if out[-1] != '\n':
      # The command was cut out. Assume return code was 1 but do not discard the
      # out.
      exit_code = 1
    else:
      # Strip the last line to extract the exit code.
      parts = out[:-1].rsplit('\n', 1)
      if len(parts) > 1:
        try:
          exit_code = int(parts[1])
        except (IndexError, ValueError):
          exit_code = 1
        else:
          out = parts[0]
    return out, exit_code

  def StreamingShell(self, cmd):
    """Streams the output from shell.

    Yields output as str. The exit code and exceptions are lost. If the device
    context is invalid, the command is silently dropped.
    """
    if isinstance(cmd, unicode):
      cmd = cmd.encode('utf-8')
    assert isinstance(cmd, str), cmd
    assert self.IsShellOk(cmd), 'Command is too long: %r' % cmd
    if self._adb_cmd:
      try:
        for out in self._adb_cmd.StreamingShell(cmd):
          yield out
      except self._ERRORS as e:
        # Do not try to reset the USB context, just exit.
        _LOG.info('%s.StreamingShell(): %s', self.port_path, e)

  def Root(self):
    """If adbd on the device is not root, ask it to restart as root.

    This causes the USB device to disapear momentarily, which causes a big mess,
    as we cannot communicate with it for a moment. So try to be clever and
    reenumerate the device until the device is back, then reinitialize the
    communication, all synchronously.
    """
    # Don't bother restarting adbd if we can already attain root status.
    if self.IsRoot():
      return True
    if self._adb_cmd and self._Root():
      # There's no need to loop initially too fast. Restarting the adbd always
      # take 'some' amount of time. In practice, this can take a good 1 second.
      time.sleep(0.1)
      i = 0
      for i in self._Loop():
        # We need to reconnect so we can assert the connection to adbd is to the
        # right process, not the old one but the new one.
        if not self._Reconnect(True):
          continue
        if self.IsRoot():
          return True
        elif self.IsSuRoot():
          self._needs_su = True
          return True
      _LOG.error('%s.Root(): Failed to id after %d tries', self.port_path, i+1)
    return False

  def Unroot(self):
    """If adbd on the device is root, ask it to restart as user."""
    if self._adb_cmd and self._Unroot():
      # There's no need to loop initially too fast. Restarting the adbd always
      # take 'some' amount of time. In practice, this can take a good 5 seconds.
      time.sleep(0.1)
      i = 0
      for i in self._Loop():
        # We need to reconnect so we can assert the connection to adbd is to the
        # right process, not the old one but the new one.
        if not self._Reconnect(True):
          continue
        if self.IsRoot() is False:
          self._needs_su = False
          return True
      _LOG.error(
          '%s.Unroot(): Failed to id after %d tries', self.port_path, i+1)
    return False

  def IsRoot(self):
    """Returns True if adbd is running as root.

    Returns None if it can't give a meaningful answer.

    Technically speaking this function is "high level" but is needed because
    reset_adbd_as_*() calls are asynchronous, so there is a race condition while
    adbd triggers the internal restart and its socket waiting for new
    connections; the previous (non-switched) server may accept connection while
    it is shutting down so it is important to repeatedly query until connections
    go to the new restarted adbd process.
    """
    out, exit_code = self.Shell('id')
    if exit_code != 0 or not out:
      return None
    return out.startswith('uid=0(root)')

  def IsSuRoot(self):
    """Returns True if the user can "su" as root.

    Like IsRoot(), but explicitly runs 'su root' to try to attain root status.
    Some devices explicitly need this, even if adbd has successfully been
    restarted as root.
    """
    out, exit_code = self.Shell('su root id')
    if exit_code != 0 or not out:
      return None
    return out.startswith('uid=0(root)')

  # Protected methods.

  def _Reboot(self):
    """Reboots the phone."""
    i = 0
    for i in self._Loop():
      try:
        out = self._adb_cmd.Reboot()
      except usb_exceptions.ReadFailedError:
        # It looks like it's possible that adbd restarts the device so fast that
        # it close the USB connection before adbd has the time to reply (yay for
        # race conditions). In that case there's no way to know if the command
        # worked too fast or something went wrong and the command didn't go
        # through.
        # Assume it worked, which is nasty.
        out = ''
      except self._ERRORS as e:
        if not self._Reset('(): %s', e, use_serial=True):
          break
        continue

      _LOG.info('%s._Reboot(): %s: %r', self.port_path, self.serial, out)
      if out == '':
        # reboot doesn't reply anything.
        return True
      assert False, repr(out)
    _LOG.error('%s._Reboot(): Failed after %d tries', self.port_path, i+1)
    return False

  def _Root(self):
    """Upgrades adbd from shell user context (uid 2000) to root."""
    i = 0
    for i in self._Loop():
      try:
        out = self._adb_cmd.Root()
      except usb_exceptions.ReadFailedError:
        # It looks like it's possible that adbd restarts itself so fast that it
        # close the USB connection before adbd has the time to reply (yay for
        # race conditions). In that case there's no way to know if the command
        # worked too fast or something went wrong and the command didn't go
        # through.
        # Assume it worked, which is nasty.
        out = 'restarting adbd as root\n'
      except adb_protocol.InvalidResponseError as e:
        # Same issue as mentioned above, but this error surfaces itself as an
        # InvalidResponseError exception when communicating over tcp.
        out = 'restarting adbd as root\n'
      except self._ERRORS as e:
        if not self._Reset('(): %s', e, use_serial=True):
          break
        continue

      _LOG.info('%s._Root(): %r', self.port_path, out)
      # Hardcoded strings in platform_system_core/adb/services.cpp
      if out == 'adbd is already running as root\n':
        return True
      if out == 'adbd cannot run as root in production builds\n':
        _LOG.error('%s._Root(): %r', self.port_path, out)
        return False
      # Sadly, it's possible that adbd restarts so fast that it doesn't even
      # wait for the output buffer to be flushed. In this case, out == ''.
      if out in ('', 'restarting adbd as root\n'):
        return True
      assert False, repr(out)
    _LOG.error('%s._Root(): Failed after %d tries', self.port_path, i+1)
    return False

  def _Unroot(self):
    """Reduces adbd from root to shell user context (uid 2000).

    Doing so has the effect of having the device switch USB port. As such, the
    device has to be found back by the serial number, not by self.port_path
    """
    assert self._serial
    i = 0
    for i in self._Loop():
      try:
        out = self._adb_cmd.Unroot()
      except usb_exceptions.ReadFailedError:
        # Unroot() is special (compared to Reboot and Root) as it's mostly
        # guaranteed that the output was sent before the USB connection is torn
        # down. But the exception still swallows the output (!)
        # Assume it worked, which is nasty.
        out = 'restarting adbd as non root\n'
      except adb_protocol.InvalidResponseError as e:
        # Same issue as mentioned above, but this error surfaces itself as an
        # InvalidResponseError exception when communicating over tcp.
        if self._handle.is_local:
          raise
        out = 'restarting adbd as non root\n'
      except self._ERRORS as e:
        if not self._Reset('(): %s', e, use_serial=True):
          break
        continue

      _LOG.info('%s.Unroot(): %r', self.port_path, out)
      # Hardcoded strings in platform_system_core/adb/services.cpp
      # Sadly, it's possible that adbd restarts so fast that it doesn't even
      # wait for the output buffer to be flushed. In this case, out == ''.
      # It was observed that the trailing \n could be missing so strip it.
      if out.rstrip('\n') in (
          '', 'adbd not running as root', 'restarting adbd as non root'):
        return True
      assert False, repr(out)
    _LOG.error('%s._Unroot(): Failed after %d tries', self.port_path, i+1)
    return False

  def _Find(self, use_serial):
    """Initializes self._handle from self.port_path.

    The handle is left unopened.
    """
    assert not self._handle
    assert not self._adb_cmd
    # TODO(maruel): Add support for TCP/IP communication.
    try:
      if self.port_path:
        previous_port_path = self._port_path
        if use_serial:
          assert self._serial
          self._handle = common.UsbHandle.Find(
              adb_commands.DeviceIsAvailable, serial=self._serial,
              timeout_ms=self._default_timeout_ms)
          # Update the new found port path.
          self._port_path = self._handle.port_path_str
        else:
          self._handle = common.UsbHandle.Find(
              adb_commands.DeviceIsAvailable, port_path=self.port_path,
              timeout_ms=self._default_timeout_ms)
        _LOG.info(
            '%s._Find(%s) %s = %s',
            previous_port_path, use_serial, self._serial,
            self.port_path if self._handle else 'None')
      else:
        self._handle = common.TcpHandle(self._serial)
    except (common.usb1.USBError, usb_exceptions.DeviceNotFoundError) as e:
      _LOG.debug(
          '%s._Find(%s) %s : %s', self.port_path, use_serial, self._serial, e)
    return bool(self._handle)

  def _WaitUntilFound(self, use_serial, timeout=None):
    """Loops until the device is found on the USB bus.

    The handle is left unopened.

    This function should normally be called when either adbd or the phone is
    rebooting.
    """
    assert not self._handle
    _LOG.debug(
        '%s._WaitUntilFound(%s)',
        self.port_path, self.serial if use_serial else use_serial)
    i = 0
    for i in self._Loop(timeout=timeout):
      if self._Find(use_serial=use_serial):
        return True
    # Enumerate the devices present to help.
    def fn(h):
      try:
        return '%s:%s' % (h.port_path, h.serial_number)
      except common.usb1.USBError:
        return '%s' % (h.port_path,)
    devices = '; '.join(
        fn(h) for h in
        common.UsbHandle.FindDevicesSafe(DeviceIsAvailable, timeout_ms=1000))
    _LOG.warning(
        '%s._WaitUntilFound(%s) gave up after %d tries; found %s.',
        self.port_path, self.serial if use_serial else use_serial, i+1, devices)

    return False

  def _OpenHandle(self):
    """Opens the unopened self._handle."""
    #_LOG.debug('%s._OpenHandle()', self.port_path)
    if self._handle:
      assert not self._handle.is_open
      i = 0
      for i in self._Loop():
        try:
          # If this succeeds, this initializes self._handle._handle, which is a
          # usb1.USBDeviceHandle.
          self._handle.Open()
          return True
        except common.usb1.USBErrorNoDevice as e:
          _LOG.warning(
              '%s._OpenHandle(): USBErrorNoDevice: %s', self.port_path, e)
          # Do not kill adb, it just means the USB host is likely resetting and
          # the device is temporarily unavailable. We can't use
          # handle.serial_number since this communicates with the device.
          # Might take a while for the device to come back. Exit early.
          break
        except common.usb1.USBErrorNotFound as e:
          _LOG.warning(
              '%s._OpenHandle(): USBErrorNotFound: %s', self.port_path, e)
          # Do not kill adb, it just means the USB host is likely resetting (?)
          # and the device is temporarily unavailable. We can't use
          # handle.serial_number since this communicates with the device.
          # Might take a while for the device to come back. Exit early.
          break
        except common.usb1.USBErrorBusy as e:
          _LOG.warning('%s._OpenHandle(): USBErrorBusy: %s', self.port_path, e)
          KillADB()
        except common.usb1.USBErrorAccess as e:
          # Do not try to use serial_number, since we can't even access the
          # port.
          _LOG.error(
              '%s._OpenHandle(): No access, maybe change udev rule or add '
              'yourself to plugdev: %s', self.port_path, e)
          # Not worth retrying, exit early.
          break
        except common.usb1.USBErrorIO as e:
          _LOG.warning('%s._OpenHandle(): USBErrorIO: %s', self.port_path, e)
      else:
        _LOG.error(
            '%s._OpenHandle(): Failed after %d tries', self.port_path, i+1)
      self.Close()
    return False

  def _Connect(self, use_serial):
    """Initializes self._adb_cmd from the opened self._handle.

    Returns True on success.
    """
    assert not self._adb_cmd
    _LOG.debug('%s._Connect(%s)', self.port_path, use_serial)
    if self._handle:
      assert self._handle.is_open
      # Ensure at least two loops if an I/O timeout occurs.
      for _ in self._Loop(timeout=2*self._lost_timeout_ms):
        # On the first access with an open handle, try to set self._serial to
        # the serial number of the device. This means communicating to the USB
        # device, so it may throw.
        if not self._handle:
          # It may happen on a retry.
          if self._WaitUntilFound(use_serial=use_serial):
            self._OpenHandle()
          if not self._handle:
            break

        if not self._serial or use_serial:
          try:
            # The serial number is attached to common.UsbHandle, no
            # adb_commands.AdbCommands.
            self._serial = self._handle.serial_number
          except self._ERRORS as e:
            self.Close()
            continue

        assert self._handle
        assert not self._adb_cmd
        try:
          # TODO(maruel): A better fix would be to change python-adb to continue
          # the authentication dance from where it stopped. This is left as a
          # follow up.
          self._adb_cmd = adb_commands.AdbCommands.Connect(
              self._handle, banner=self._banner, rsa_keys=self._rsa_keys,
              auth_timeout_ms=self._auth_timeout_ms)
          self._failure = None if self._adb_cmd else 'unknown'
          break
        except usb_exceptions.DeviceAuthError as e:
          self._failure = 'unauthorized'
          _LOG.warning('AUTH FAILURE: %s: %s', self.port_path, e)
        except usb_exceptions.LibusbWrappingError as e:
          self._failure = 'usb_failure'
          _LOG.warning('I/O FAILURE: %s: %s', self.port_path, e)
          if self._should_reset:
            self._handle.Reset()
        except adb_protocol.InvalidResponseError as e:
          self._failure = 'protocol_fault'
          _LOG.warning('SYNC FAILURE: %s: %s', self.port_path, e)
        finally:
          # Do not leak the USB handle when we can't talk to the device.
          if not self._adb_cmd:
            self.Close()

    if not self._adb_cmd and self._handle:
      _LOG.error('Unexpected close')
      self.Close()
    return bool(self._adb_cmd)

  def _Loop(self, timeout=None):
    """Yields a loop until it's too late."""
    timeout = timeout or self._lost_timeout_ms
    start = time.time()
    for i in xrange(self._tries):
      if ((time.time() - start) * 1000) >= timeout:
        return
      yield i
      if ((time.time() - start) * 1000) >= timeout:
        return
      time.sleep(self._sleep)

  def _Reset(self, fmt, *args, **kwargs):
    """When a self._ERRORS occurred, try to reset the device.

    Returns True on success.
    """
    items = [self.port_path, inspect.stack()[1][3]]
    items.extend(args)
    try:
      msg = (u'%s.%s' + fmt) % tuple(items)
    except UnicodeDecodeError:
      msg = u'%s.%s: failed to encode error message as unicode' % (
          self.port_path, inspect.stack()[1][3])
    _LOG.error(msg)
    if self._on_error:
      self._on_error(msg)

    # Reset the adbd and USB connections with a new connection.
    return self._Reconnect(kwargs.get('use_serial', False))

  def _Reconnect(self, use_serial, timeout=None):
    """Disconnects and reconnect.

    Arguments:
    - use_serial: If True, search the device by the serial number instead of the
        port number. This is necessary when downgrading adbd from root to user
        context.

    Returns True on success.
    """
    self.Close()
    if not self._WaitUntilFound(use_serial=use_serial, timeout=timeout):
      return False
    if not self._OpenHandle():
      return False
    return self._Connect(use_serial=use_serial)

  def __repr__(self):
    return '<Device %s %s>' % (
        self.port_path, self.serial if self.is_valid else '(broken)')
