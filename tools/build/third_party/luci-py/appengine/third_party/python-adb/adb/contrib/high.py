# coding=utf-8
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

"""High level functionality.

This module defines high level functions and class to communicate through ADB
protocol.
"""

import collections
import logging
import os
import pipes
import posixpath
import random
import re
import string
import threading
import time


from adb import common
from adb import sign_pythonrsa
from adb.contrib import adb_commands_safe
from adb.contrib import parallel


### Private stuff.


_LOG = logging.getLogger('adb.high')
_LOG.setLevel(logging.ERROR)


_ADB_KEYS_LOCK = threading.Lock()
# Both following are set when ADB is initialized.
# _ADB_KEYS is set to a list of adb_protocol.AuthSigner instances. It contains
# one or multiple key used to authenticate to Android debug protocol (adb).
_ADB_KEYS = None
# _ADB_KEYS_PUB is the set of public keys for corresponding initialized private
# keys.
_ADB_KEYS_PUB = set()


class _PerDeviceCache(object):
  """Caches data per device, thread-safe."""

  def __init__(self):
    self._lock = threading.Lock()
    # Keys is usb path, value is a _Cache.Device.
    self._per_device = {}

  def get(self, device):
    with self._lock:
      return self._per_device.get(device.port_path)

  def set(self, device, cache):
    with self._lock:
      self._per_device[device.port_path] = cache

  def trim(self, devices):
    """Removes any stale cache for any device that is not found anymore.

    So if a device is disconnected, reflashed then reconnected, the cache isn't
    invalid.
    """
    device_keys = {d.port_path: d for d in devices}
    with self._lock:
      for port_path in self._per_device.keys():
        dev = device_keys.get(port_path)
        if not dev or not dev.is_valid:
          del self._per_device[port_path]


# Global cache of per device cache.
_PER_DEVICE_CACHE = _PerDeviceCache()


def _ParcelToList(lines):
  """Parses 'service call' output."""
  out = []
  for line in lines:
    match = re.match(
        '  0x[0-9a-f]{8}\\: ([0-9a-f ]{8}) ([0-9a-f ]{8}) ([0-9a-f ]{8}) '
        '([0-9a-f ]{8}) \'.{16}\'\\)?', line)
    if not match:
      break
    for i in xrange(1, 5):
      group = match.group(i)
      char = group[4:8]
      if char != '    ':
        out.append(char)
      char = group[0:4]
      if char != '    ':
        out.append(char)
  return out


def _InitCache(device):
  """Primes data known to be fetched soon right away that is static for the
  lifetime of the device.

  The data is cached in _PER_DEVICE_CACHE() as long as the device is connected
  and responsive.
  """
  cache = _PER_DEVICE_CACHE.get(device)
  if not cache:
    # TODO(maruel): This doesn't seem super useful since the following symlinks
    # already exist: /sdcard/, /mnt/sdcard, /storage/sdcard0.
    external_storage_path, exitcode = device.Shell('echo -n $EXTERNAL_STORAGE')
    if exitcode:
      external_storage_path = None

    properties = {}
    out = device.PullContent('/system/build.prop')
    if out:
      for line in out.splitlines():
        if line.startswith(u'#') or not line:
          continue
        if line.startswith('import '):
          # That's a new-style import line. For now just ignore it as we do not
          # need it for now.
          continue
        key, value = line.split(u'=', 1)
        properties[key] = value
    else:
      # /system/build.prop isn't world-readable in Oreo, so populate the cache
      # with getprop directly.
      out, exit_code = device.Shell('getprop | grep -E "ro.(build|product)"')
      if exit_code != 0:
        _LOG.warning('Unable to fetch build properties: %s', out)
      else:
        for line in out.splitlines():
          match = re.match(r'\[(\S+)\]\: \[(\S+)\]', line)
          if match:
            properties[match.group(1)] = match.group(2)

    mode, _, _ = device.Stat('/system/xbin/su')
    has_su = bool(mode)
    # Stat'ing /system/xbin/su directly from adbd outside of a shell can fail
    # due to SELinux permission errors. Try again by executing su via shell.
    if not has_su:
      out, exit_code = device.Shell('/system/xbin/su root whoami')
      if not exit_code and out and out.strip() == 'root':
        has_su = True

    available_governors = KNOWN_CPU_SCALING_GOVERNOR_VALUES
    out = device.PullContent(
        '/sys/devices/system/cpu/cpu0/cpufreq/scaling_available_governors')
    if out:
      available_governors = sorted(i for i in out.split())
      assert set(available_governors).intersection(
          KNOWN_CPU_SCALING_GOVERNOR_VALUES), available_governors

    available_frequencies = device.PullContent(
        '/sys/devices/system/cpu/cpu0/cpufreq/scaling_available_frequencies')
    if available_frequencies:
      available_frequencies = sorted(
          int(i) for i in available_frequencies.strip().split())
    else:
      # It's possibly an older kernel. In that case, query the min/max instead.
      scaling_min_freq = device.PullContent(
          '/sys/devices/system/cpu/cpu0/cpufreq/scaling_min_freq')
      scaling_max_freq = device.PullContent(
          '/sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq')
      if scaling_min_freq and scaling_max_freq:
        # In practice there's more CPU speeds than this but there's no way (?)
        # to query this information.
        available_frequencies = [int(scaling_min_freq), int(scaling_max_freq)]

    cache = DeviceCache(
        properties, external_storage_path, has_su,
        available_frequencies, available_governors)
    # Only save the cache if all the calls above worked.
    if all(i is not None for i in cache._asdict().itervalues()):
      _PER_DEVICE_CACHE.set(device, cache)
  return cache


### Public API.


# List of known CPU scaling governor values. Each devices has a subset of these.
# There are other scaling governors not listed there that can be seen on
# non-standard non-Nexus builds.
KNOWN_CPU_SCALING_GOVERNOR_VALUES = (
    # conservative tries to keep the CPU at its lowest frequency as much as
    # possible. It slowly ramp up CPU frequency by going through all the
    # intermediate frequencies then quickly drop to the lowest frequency as fast
    # as possible.
    'conservative',
    # interactive quickly scales up frequency on load then slowly scales back
    # when there isn't much load.
    'interactive',
    # ondemand quickly scales up frequency on load then quickly scales back when
    # there isn't much load.
    'ondemand',
    # performance locks the CPU frequency at its maximum supported frequency; it
    # is the equivalent of userspace at scaling_max_freq. On a ARM based device
    # without active cooling, this means that eventually the CPU will hit
    # temperature based throttling.
    'performance',
    # powersave locks the CPU frequency to its lowest supported frequency; it is
    # the equivalent of userspace at scaling_min_freq.
    'powersave',
    # userspace overrides the scaling governor with an user defined constant
    # frequency.
    'userspace',
    # hotplug is alike ondemand but enables turning off unused CPU cores during
    # periods of low utilization.
    'hotplug',
    # schedutil uses CPU utilization data available from the CPU scheduler
    # https://www.kernel.org/doc/html/v4.14/admin-guide/pm/cpufreq.html#schedutil
    'schedutil',
)


# DeviceCache is static information about a device that it preemptively
# initialized and that cannot change without formatting the device.
DeviceCache = collections.namedtuple(
    'DeviceCache',
    [
        # Cache of /system/build.prop on the Android device.
        'build_props',
        # Cache of $EXTERNAL_STORAGE_PATH.
        'external_storage_path',
        # /system/xbin/su exists.
        'has_su',
        # Valid CPU frequencies.
        'available_frequencies',
        # Valid CPU scaling governors.
        'available_governors',
    ])


def Initialize(pub_key, priv_key):
  """Initialize Android support through adb.

  You can steal pub_key, priv_key pair from ~/.android/adbkey and
  ~/.android/adbkey.pub.
  """
  with _ADB_KEYS_LOCK:
    global _ADB_KEYS
    if _ADB_KEYS is not None:
      assert False, 'initialize() was called repeatedly: ignoring keys'
    assert bool(pub_key) == bool(priv_key)
    pub_key = pub_key.strip() if pub_key else pub_key
    priv_key = priv_key.strip() if priv_key else priv_key

    _ADB_KEYS = []
    if pub_key:
      _ADB_KEYS.append(sign_pythonrsa.PythonRSASigner(pub_key, priv_key))
      _ADB_KEYS_PUB.add(pub_key)

    # Try to add local adb keys if available.
    path = os.path.expanduser('~/.android/adbkey')
    if os.path.isfile(path) and os.path.isfile(path + '.pub'):
      with open(path + '.pub', 'rb') as f:
        pub_key = f.read().strip()
      with open(path, 'rb') as f:
        priv_key = f.read().strip()
      _ADB_KEYS.append(sign_pythonrsa.PythonRSASigner(pub_key, priv_key))
      _ADB_KEYS_PUB.add(pub_key)

    return _ADB_KEYS[:]


def _ConnectFromHandles(handles, as_root=False, **kwargs):
  """Connects to the devices provided by handles."""
  def fn(handle):
    device = HighDevice.Connect(handle, **kwargs)
    if as_root and device.cache.has_su and not device.IsRoot():
      # This updates the port path of the device thus clears its cache.
      device.Root()
    return device

  devices = parallel.pmap(fn, handles)
  _PER_DEVICE_CACHE.trim(devices)
  return devices


def GetLocalDevices(
    banner, default_timeout_ms, auth_timeout_ms, on_error=None, as_root=False,
    enable_resets=False):
  """Returns the list of devices available.

  Caller MUST call CloseDevices(devices) on the return value or call .Close() on
  each element to close the USB handles.

  Arguments:
  - banner: authentication banner associated with the RSA keys. It's better to
        use a constant.
  - default_timeout_ms: default I/O operation timeout.
  - auth_timeout_ms: timeout for the user to accept the public key.
  - on_error: callback when an internal failure occurs.
  - as_root: if True, restarts adbd as root if possible.

  Returns one of:
    - list of HighDevice instances.
    - None if adb is unavailable.
  """
  with _ADB_KEYS_LOCK:
    if not _ADB_KEYS:
      return []

  # Skip devices that don't expose a serial number.
  device_matcher = lambda device: device.serial_number is not None

  # Create unopened handles for all usb devices.
  handles = list(
      common.UsbHandle.FindDevicesSafe(
          adb_commands_safe.DeviceIsAvailable, timeout_ms=default_timeout_ms,
          device_matcher=device_matcher))

  return _ConnectFromHandles(handles, banner=banner,
                             default_timeout_ms=default_timeout_ms,
                             auth_timeout_ms=auth_timeout_ms, on_error=on_error,
                             as_root=as_root, enable_resets=enable_resets)


def GetRemoteDevices(banner, endpoints, default_timeout_ms, auth_timeout_ms,
                     on_error=None, as_root=False):
  """Returns the list of devices available.

  Caller MUST call CloseDevices(devices) on the return value or call .Close() on
  each element to close the TCP handles.

  Arguments:
  - banner: authentication banner associated with the RSA keys. It's better to
        use a constant.
  - endpoints: list of ip[:port] endpoints of devices to connect to via TCP.
  - default_timeout_ms: default I/O operation timeout.
  - auth_timeout_ms: timeout for the user to accept the public key.
  - on_error: callback when an internal failure occurs.
  - as_root: if True, restarts adbd as root if possible.

  Returns one of:
    - list of HighDevice instances.
    - None if adb is unavailable.
  """
  with _ADB_KEYS_LOCK:
    if not _ADB_KEYS:
      return []
  # Create unopened handles for all remote devices.
  handles = [common.TcpHandle(endpoint) for endpoint in endpoints]

  return _ConnectFromHandles(handles, banner=banner,
                             default_timeout_ms=default_timeout_ms,
                             auth_timeout_ms=auth_timeout_ms, on_error=on_error,
                             as_root=as_root)


def CloseDevices(devices):
  """Closes all devices opened by GetDevices()."""
  for device in devices or []:
    device.Close()


class HighDevice(object):
  """High level device representation.

  This class contains all the methods that are effectively composite calls to
  the low level functionality provided by AdbCommandsSafe. As such there's no
  direct access by methods of this class to self.cmd.

  Most importantly, there must be no automatic retry at this level; all
  automatic retries must be inside AdbCommandsSafe. The only exception is
  WaitForXXX() functions.
  """
  def __init__(self, device, cache):
    # device can be one of adb_commands_safe.AdbCommandsSafe,
    # adb_commands.AdbCommands or a fake.

    # Immutable.
    self._device = device
    self._cache = cache

  @classmethod
  def ConnectDevice(cls, port_path, **kwargs):
    """Opens the device and do the initial adb-connect."""
    kwargs['port_path'] = port_path
    return cls._Connect(
        adb_commands_safe.AdbCommandsSafe.ConnectDevice, **kwargs)

  @classmethod
  def Connect(cls, handle, **kwargs):
    """Opens the device and do the initial adb-connect."""
    kwargs['handle'] = handle
    return cls._Connect(adb_commands_safe.AdbCommandsSafe.Connect, **kwargs)

  # Proxy the embedded low level methods.

  @property
  def cache(self):
    """Returns an instance of DeviceCache."""
    return self._cache

  @property
  def max_packet_size(self):
    return self._device.max_packet_size

  @property
  def port_path(self):
    return self._device.port_path

  @property
  def sysfs_port_path(self):
    return self._device.sysfs_port_path

  @property
  def serial(self):
    return self._device.serial

  @property
  def is_valid(self):
    return self._device.is_valid

  @property
  def failure(self):
    return self._device.failure

  def Close(self):
    self._device.Close()

  def GetUptime(self):
    """Returns the device's uptime in second."""
    return self._device.GetUptime()

  def IsRoot(self):
    return self._device.IsRoot()

  def List(self, destdir):
    return self._device.List(destdir)

  def Pull(self, *args, **kwargs):
    return self._device.Pull(*args, **kwargs)

  def PullContent(self, *args, **kwargs):
    return self._device.PullContent(*args, **kwargs)

  def Push(self, *args, **kwargs):
    return self._device.Push(*args, **kwargs)

  def PushContent(self, *args, **kwargs):
    return self._device.PushContent(*args, **kwargs)

  def Reboot(self, wait=True, force=False):
    """Reboots the phone then Waits for the device to come back.

    adbd running on the phone will likely not be in Root(), so the caller should
    call Root() right afterward if desired.

    Arguments:
    - wait: If true, attempts to reconnect to the device after sending the
        reboot. Otherwise, exit early.
    - force: If true, attempts to force a reboot via the /proc/sysrq-trigger
        file if the initial command fails. See
        https://android.googlesource.com/kernel/common/+/android-3.10.y/Documentation/sysrq.txt
        for more info.
    """
    if not self._device.Reboot(wait=wait, force=force):
      return False
    return self.WaitUntilFullyBooted() if wait else True

  def Reset(self, new_endpoint=None):
    self._device.ResetHandle(new_endpoint=new_endpoint)

  def Remount(self):
    return self._device.Remount()

  def Root(self):
    return self._device.Root()

  def Shell(self, cmd, timeout_ms=None):
    """Automatically uses WrappedShell() when necessary."""
    if self._device.IsShellOk(cmd):
      return self._device.Shell(cmd, timeout_ms=timeout_ms)
    else:
      return self.WrappedShell([cmd], timeout_ms=timeout_ms)

  def ShellRaw(self, cmd):
    return self._device.ShellRaw(cmd)

  def StreamingShell(self, cmd):
    return self._device.StreamingShell(cmd)

  def Stat(self, dest):
    return self._device.Stat(dest)

  def Unroot(self):
    return self._device.Unroot()

  def __repr__(self):
    return repr(self._device)

  # High level methods.

  def GetCPUScale(self):
    """Returns the CPU scaling factor."""
    mapping = {
        'scaling_cur_freq': u'cur',
        'scaling_governor': u'governor',
    }
    out = {
        v: self.PullContent('/sys/devices/system/cpu/cpu0/cpufreq/' + k)
        for k, v in mapping.iteritems()
    }
    return {
        k: v.strip() if isinstance(v, str) else v for k, v in out.iteritems()
    }

  def SetCPUScalingGovernor(self, governor):
    """Sets the CPU scaling governor to the one specified.

    Returns:
      True on success.
    """
    assert governor in KNOWN_CPU_SCALING_GOVERNOR_VALUES, repr(governor)
    if not self.cache.available_governors:
      return False
    if governor not in self.cache.available_governors:
      if governor == 'powersave' and self.cache.available_frequencies:
        return self.SetCPUSpeed(self.cache.available_frequencies[0])
      if governor == 'ondemand':
        governor = 'interactive'
      elif governor == 'interactive':
        governor = 'ondemand'
      else:
        _LOG.warning(
            '%s.SetCPUScalingGovernor(): Can\'t switch to %s',
            self.port_path, governor)
        return False
    assert governor in KNOWN_CPU_SCALING_GOVERNOR_VALUES, governor

    path = '/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor'
    # First query the current state and only try to switch if it's different.
    prev = self.PullContent(path)
    if prev:
      prev = prev.strip()
      if prev == governor:
        return True
      if prev not in self.cache.available_governors:
        _LOG.warning(
            '%s.SetCPUScalingGovernor(): Read invalid scaling_governor: %s',
            self.port_path, prev)
    else:
      _LOG.warning(
          '%s.SetCPUScalingGovernor(): Failed to read %s', self.port_path, path)

    # This works on Nexus 10 but not on Nexus 5. Need to investigate more. In
    # the meantime, simply try one after the other.
    if not self.PushContent(governor + '\n', path):
      _LOG.info(
          '%s.SetCPUScalingGovernor(): Failed to push %s in %s',
          self.port_path, governor, path)
      # Fallback via shell.
      _, exit_code = self.Shell('echo "%s" > %s' % (governor, path))
      if exit_code != 0:
        _LOG.warning(
            '%s.SetCPUScalingGovernor(): Writing %s failed; was %s',
            self.port_path, governor, prev)
        return False
    # Get it back to confirm.
    newval = self.PullContent(path)
    if not (newval or '').strip() == governor:
      _LOG.warning(
          '%s.SetCPUScalingGovernor(): Wrote %s; was %s; got %s',
          self.port_path, governor, prev, newval)
      return False
    return True

  def SetCPUSpeed(self, speed):
    """Enforces strict CPU speed and disable the CPU scaling governor.

    Returns:
      True on success.
    """
    assert isinstance(speed, int), speed
    assert 10000 <= speed <= 10000000, speed
    if not self.cache.available_frequencies:
      return False
    assert speed in self.cache.available_frequencies, (
        speed, self.cache.available_frequencies)
    if not self.SetCPUScalingGovernor('userspace'):
      return False

    # This works on Nexus 10 but not on Nexus 5. Need to investigate more. In
    # the meantime, simply try one after the other.
    path = '/sys/devices/system/cpu/cpu0/cpufreq/scaling_setspeed'
    if self.PushContent('%d\n' % speed, path):
      # Get it back to confirm.
      val = self.PullContent(path)
      if (val or '').strip() == str(speed):
        return True

    _LOG.info(
        '%s.SetCPUSpeed(): Failed to push %d in %s',
        self.port_path, speed, path)
    # Fallback via shell.
    _, exit_code = self.Shell('echo "%d" > %s' % (speed, path))
    if exit_code != 0:
      _LOG.warning(
          '%s.SetCPUSpeed(): Writing %d failed',
          self.port_path, speed)
      return False

    # Get it back to confirm.
    val = self.PullContent(path)
    return (val or '').strip() == str(speed)

  def GetTemperatures(self):
    """Returns the device's temperatures if available as a dict."""
    # Not all devices export these files. On other devices, the only real way to
    # read it is via Java
    # developer.android.com/guide/topics/sensors/sensors_environment.html
    out = {}
    for sensor in self.List('/sys/class/thermal') or []:
      if sensor.filename in ('.', '..'):
        continue
      if not sensor.filename.startswith('thermal_zone'):
        continue
      path = '/sys/class/thermal/' + sensor.filename
      # Expected files:
      # - mode: enabled or disabled.
      # - temp: temperature as reported by the sensor, generally in C or mC.
      # - type: driver name.
      # - power/
      # - trip_point_0_temp
      # - trip_point_0_type
      # - trip_point_1_temp
      # - trip_point_1_type
      # - subsystem/ -> link back to ../../../../class/thermal
      # - policy
      # - uevent
      # - passive
      temp = self.PullContent(path + '/temp')
      if not temp:
        continue
      # Assumes it's in °C.
      value = float(temp)
      if value > 1000:
        # Then assumes it's in m°C.
        # TODO(maruel): Discern near cold temperature, e.g. 0.1°C.
        value = value / 1000.
      if value <= 0.:
        # TODO(maruel): Support cold temperatures below 0°C.
        continue
      sensor_type = self.PullContent(path + '/type')
      if sensor_type:
        out[sensor_type.strip()] = value
    # Filter out unnecessary stuff.
    return out

  def GetBattery(self):
    """Returns details about the battery's state."""
    props = {}
    out = self.Dumpsys('battery')
    if not out:
      return props

    for line in out.splitlines():
      if line.endswith(u':'):
        continue
      # On Android 4.1.2, it uses "voltage:123" instead of "voltage: 123".
      parts = line.split(u':', 2)
      if len(parts) == 2:
        key, value = parts
        props[key.lstrip()] = value.strip()
    out = {u'power': [], u'current': None}

    # This isn't in "dumpsys battery" for some reason, but is still useful.
    current = self.PullContent('/sys/class/power_supply/battery/current_now')
    if current:
      try:
        out[u'current'] = int(current.strip())
      except ValueError:
        pass

    if props.get(u'AC powered') == u'true':
      out[u'power'].append(u'AC')
    if props.get(u'USB powered') == u'true':
      out[u'power'].append(u'USB')
    if props.get(u'Wireless powered') == u'true':
      out[u'power'].append(u'Wireless')
    for key in (u'health', u'level', u'status', u'temperature', u'voltage'):
      out[key] = int(props[key]) if key in props else None
    return out

  def GetMemInfo(self):
    """Returns used, free, and total ram."""
    props = {}
    out = self.PullContent('/proc/meminfo')
    if not out:
      return props

    def parse_memline(key, res):
      if line.startswith('%s:' % key):
        parts = line.split()
        if len(parts) == 3 and parts[2] == 'kB':
          props[res] = int(parts[1])

    for line in out.splitlines():
      parse_memline('MemTotal', 'total')
      parse_memline('MemFree', 'free')
      parse_memline('Buffers', 'buffers')
      parse_memline('Cached', 'cached')

    if ('total' in props and
        'free' in props and
        'buffers' in props and
        'cached' in props):
      props['avail'] = props['free'] + props['cached'] + props['buffers']
      props['used'] = props['total'] - props['avail']

    return props

  def GetProcessCount(self):
    """Returns the number of total processes on the device."""
    out, exit_code = self.Shell('ps')
    if exit_code != 0:
      return None
    if not out.startswith('USER'):
      return None

    return out.count('\n') - 1

  def GetDisk(self):
    """Returns details about the device's disks."""
    props = {}
    out = self.Dumpsys('diskstats')
    if not out:
      return props
    for line in out.splitlines():
      if line.endswith(u':'):
        continue
      parts = line.split(u': ', 2)
      if len(parts) == 2:
        key, value = parts
        match = re.match(ur'^(\d+)K / (\d+)K.*', value)
        if match:
          props[key.lstrip()] = {
              'free_mb': round(float(match.group(1)) / 1024., 1),
              'size_mb': round(float(match.group(2)) / 1024., 1),
          }
    km = {
      u'cache': u'Cache-Free',
      u'data': u'Data-Free',
      u'system': u'System-Free',
    }
    return {k: props[v] for k, v in km.iteritems() if v in props}

  def GetPackageVersion(self, package):
    """Returns the installed version of the given package."""
    out = self.Dumpsys('package %s' % package)
    if out is None:
      return None
    for line in out.splitlines():
      if 'versionName=' in line:
        # Version format is wildly different for different packages, or even
        # sometimes within different builds for the same package. However, they
        # all seem to start it "versionName=" and have some sort of whitespace
        # after the actual version string.
        return line.split('=', 1)[1].split()[0]
    return None

  def GetIMEI(self):
    """Returns the phone's IMEI."""
    # Android <5.0.
    out = self.Dumpsys('iphonesubinfo')
    if out:
      match = re.search('  Device ID = (.+)$', out)
      if match:
        return match.group(1)

    # Android >= 5.0.
    out, _ = self.Shell('service call iphonesubinfo 1')
    if out:
      lines = out.splitlines()
      if len(lines) >= 4 and lines[0] == 'Result: Parcel(':
        # Process the UTF-16 string.
        chars = _ParcelToList(lines[1:])[4:-1]
        return u''.join(unichr(c) for c in (int(i, 16) for i in chars))
    return None

  def GetIPs(self):
    """Returns the current IP addresses of networks that are up."""
    # There's multiple ways to find parts of this information:
    # - dumpsys wifi
    # - getprop dhcp.wlan0.ipaddress
    # - ip -o addr show
    # - ip route
    # - netcfg

    # <NAME> <UP/DOWN> <IP/MASK> <UNKNOWN> <MAC>
    out, exit_code = self.Shell('netcfg')
    if exit_code or out is None:
      return []
    parts = (l.split() for l in out.splitlines())
    return {
        p[0]: p[2].split('/', 1)[0] for p in parts
        if p[0] != 'lo' and p[1] == 'UP' and p[2] != '0.0.0.0/0'
    }

  def GetLastUID(self):
    """Returns the highest UID on the device."""
    # pylint: disable=line-too-long
    # Applications are set in the 10000-19999 UID space. The UID are not reused.
    # So after installing 10000 apps, including stock apps, ignoring uninstalls,
    # then the device becomes unusable. Oops!
    # https://android.googlesource.com/platform/frameworks/base/+/master/api/current.txt
    # or:
    # curl -sSLJ \
    #   https://android.googlesource.com/platform/frameworks/base/+/master/api/current.txt?format=TEXT \
    #   | base64 --decode | grep APPLICATION_UID
    out = self.PullContent('/data/system/packages.list')
    if not out:
      return None
    return max(int(l.split(' ', 2)[1]) for l in out.splitlines() if len(l) > 2)

  def GetPackages(self):
    """Returns the list of packages installed."""
    # pm can be very slow at times. Use a longer timeout to prevent
    # confusing a long-running command with an interrupted connection.
    out, _ = self.Shell('pm list packages', timeout_ms=30000)
    if not out:
      return None
    return [l.split(':', 1)[1] for l in out.strip().splitlines() if ':' in l]

  def InstallAPK(self, destdir, apk):
    """Installs apk to destdir directory."""
    # TODO(maruel): Test.
    # '/data/local/tmp/'
    dest = posixpath.join(destdir, os.path.basename(apk))
    if not self.Push(apk, dest):
      return False
    cmd = 'pm install -r %s' % pipes.quote(dest)
    out, exit_code = self.Shell(cmd)
    if not exit_code:
      return True
    _LOG.info('%s: %s', cmd, out)
    return False

  def UninstallAPK(self, package, user=None):
    """Uninstalls the package."""
    cmd = 'pm uninstall'
    if user is not None:
      cmd += ' --user %s' % user
    cmd += ' %s' % pipes.quote(package)
    out, exit_code = self.Shell(cmd)
    # pm can exit with 0 even if it fails to uninstall, and who knows if the
    # output can be reliably parsed. So only return success if the package
    # is missing afterwards.
    if not exit_code:
      out, _ = self.Shell('pm path %s' % pipes.quote(package))
      # Ignore the exit code of "pm path" since it can't be trusted. (On K, it
      # always returns 0.)
      if out is not None and 'package:' not in out:
        return True
    _LOG.info('%s: %s', cmd, out)
    return False

  def GetApplicationPath(self, package):
    # TODO(maruel): Test.
    out, _ = self.Shell('pm path %s' % pipes.quote(package))
    return out.strip().split(':', 1)[1] if out else out

  def IsFullyBooted(self, skip_sd_card=False):
    """Checks whether the device is fully booted.

    Args:
      - skip_sd_card: Skips checking if the external storage is ready if True.
                      Some devices lack an internal sd card and permanently
                      fail this check.
    Returns:
      tuple(booted, status)
      - booted: True if device is fully booted, false otherwise.
      - status: If not booted, string describing why.
    """
    if not skip_sd_card:
      ext = self.cache.external_storage_path
      if not ext or not ext.startswith('/'):
        return False, 'external storage not ready'
      if self.Stat(ext)[0] is None:
        return False, 'external storage not ready'

    # Check if the boot animation has stopped.
    prop = self.GetProp('init.svc.bootanim')
    if prop is None:
      return False, 'could not get init.svc.bootanim property'
    elif prop != 'stopped':
      return False, 'boot animation still running'

    # pm can be very slow at times. Use a longer timeout to prevent
    # confusing a long-running command with an interrupted connection.
    out, exit_code = self.Shell('pm path', timeout_ms=30000)
    if out is None:
      logging.warning('pm path (%s): failed', exit_code)
      return False, 'pm not ready'
    if ('Error: no package specified' not in out and
        'Argument expected after "path"' not in out):
      # Accepts an empty string too, which has been observed only on Android 4.4
      # (Kitkat) but not on later versions.
      if out not in (
          'Error: Could not access the Package Manager.  Is the system '
          'running?\n',
          ''):
        logging.warning(
            'Unexpected reply from pm path (%d): %r', exit_code, out)
      return False, 'pm not ready'

    # All checks passed.
    return True, None

  def WaitUntilFullyBooted(self, timeout=300, **kwargs):
    """Waits for the device to be fully started up with network connectivity.

    Arguments:
    - timeout: minimum amount of time to wait for for the device to come up
          online. It may extend to up to lock_timeout_ms more.
    """
    start = time.time()
    while True:
      is_booted, status = self.IsFullyBooted(**kwargs)
      if is_booted:
        return True
      if (time.time() - start) > timeout:
        _LOG.warning(
            '%s.WaitUntilFullyBooted() timed out due to %s',
            self.port_path, status)
        return False
      time.sleep(1)

  def PushKeys(self):
    """Pushes all the keys on the file system to the device.

    This is necessary when the device just got wiped but still has
    authorization, as soon as it reboots it'd lose the authorization.

    It never removes a previously trusted key, only adds new ones. Saves writing
    to the device if unnecessary.
    """
    keys = set(self._device.public_keys)
    old_content = self.PullContent('/data/misc/adb/adb_keys')
    if old_content:
      old_keys = set(old_content.strip().splitlines())
      if keys.issubset(old_keys):
        return True
      keys = keys | old_keys
    assert all('\n' not in k for k in keys), keys
    if self.Shell('mkdir -p /data/misc/adb')[1] != 0:
      return False
    if self.Shell('restorecon /data/misc/adb')[1] != 0:
      return False
    if not self.PushContent(
        ''.join(k + '\n' for k in sorted(keys)), '/data/misc/adb/adb_keys'):
      return False
    if self.Shell('restorecon /data/misc/adb/adb_keys')[1] != 0:
      return False
    return True

  def Mkstemp(self, content, prefix='python-adb', suffix=''):
    """Make a new temporary files with content.

    The random part of the name is guaranteed to not require quote.

    Returns None in case of failure.
    """
    # There's a small race condition in there but it's assumed only this process
    # is doing something on the device at this point.
    choices = string.ascii_letters + string.digits
    for _ in xrange(5):
      name = '/data/local/tmp/' + prefix + ''.join(
          random.choice(choices) for _ in xrange(5)) + suffix
      mode, _, _ = self.Stat(name)
      if mode:
        continue
      if self.PushContent(content, name):
        return name

  def WrappedShell(self, commands, timeout_ms=None):
    """Creates a temporary shell script, runs it then return the data.

    This is needed when:
    - the expected command is more than ~500 characters
    - the expected output is more than 32k characters

    Returns:
      tuple(stdout and stderr merged, exit_code).
    """
    content = ''.join(l + '\n' for l in commands)
    script = self.Mkstemp(content, suffix='.sh')
    if not script:
      return False
    try:
      outfile = self.Mkstemp('', suffix='.txt')
      if not outfile:
        return False
      try:
        _, exit_code = self.Shell('sh %s &> %s' % (script, outfile),
                                  timeout_ms=timeout_ms)
        out = self.PullContent(outfile)
        return out, exit_code
      finally:
        self.Shell('rm %s' % outfile)
    finally:
      self.Shell('rm %s' % script)

  def GetProp(self, prop):
    out, exit_code = self.Shell('getprop %s' % pipes.quote(prop))
    if exit_code != 0:
      return None
    return out.rstrip()

  def Dumpsys(self, arg):
    """dumpsys is a native android tool that returns inconsistent semi
    structured data.

    It acts as a directory service but each service return their data without
    any real format, and will happily return failure.
    """
    _failure_msgs = [
        'Error dumping service info',
        'Can\'t find service: ',
    ]
    out, exit_code = self.Shell('dumpsys ' + arg)
    if exit_code != 0 or any(out.startswith(msg) for msg in _failure_msgs):
      return None
    return out

  @classmethod
  def _Connect(cls, constructor, **kwargs):
    """Called by either ConnectDevice or Connect."""
    if not kwargs.get('rsa_keys'):
      with _ADB_KEYS_LOCK:
        kwargs['rsa_keys'] = _ADB_KEYS[:]
    device = constructor(**kwargs)
    return HighDevice(device, _InitCache(device))
