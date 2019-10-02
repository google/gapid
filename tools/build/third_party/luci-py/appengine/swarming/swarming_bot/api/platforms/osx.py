# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""OSX specific utility functions."""

import cgi
import ctypes
import logging
import os
import platform
import plistlib
import re
import struct
import subprocess
import time

from utils import tools

import common
import gpu

try:
  import Quartz
except ImportError:
  Quartz = None


## Private stuff.


iokit = ctypes.CDLL(
    '/System/Library/Frameworks/IOKit.framework/Versions/A/IOKit')
# https://developer.apple.com/documentation/iokit/1514274-ioconnectcallstructmethod
iokit.IOConnectCallStructMethod.argtypes = [
    ctypes.c_uint, ctypes.c_uint, ctypes.c_void_p, ctypes.c_ulonglong,
    ctypes.c_void_p, ctypes.POINTER(ctypes.c_ulonglong),
]
iokit.IOConnectCallStructMethod.restype = ctypes.c_int

# https://developer.apple.com/documentation/iokit/1514515-ioserviceopen
iokit.IOServiceOpen.argtypes = [
    ctypes.c_uint, ctypes.c_uint, ctypes.c_uint, ctypes.POINTER(ctypes.c_uint),
]
iokit.IOServiceOpen.restype = ctypes.c_int

# https://developer.apple.com/documentation/iokit/1514687-ioservicematching
iokit.IOServiceMatching.restype = ctypes.c_void_p

# https://developer.apple.com/documentation/iokit/1514494-ioservicegetmatchingservices
iokit.IOServiceGetMatchingServices.argtypes = [
    ctypes.c_uint, ctypes.c_void_p, ctypes.POINTER(ctypes.c_uint),
]
iokit.IOServiceGetMatchingServices.restype = ctypes.c_int

# https://developer.apple.com/documentation/iokit/1514741-ioiteratornext
iokit.IOIteratorNext.argtypes = [ctypes.c_uint]
iokit.IOIteratorNext.restype = ctypes.c_uint

# https://developer.apple.com/documentation/iokit/1514627-ioobjectrelease
iokit.IOObjectRelease.argtypes = [ctypes.c_uint]
iokit.IOObjectRelease.restype = ctypes.c_int


libkern = ctypes.CDLL('/usr/lib/system/libsystem_kernel.dylib')
libkern.mach_task_self.restype = ctypes.c_uint


class _SMC_KeyDataVersion(ctypes.Structure):
  _fields_ = [
      ('major', ctypes.c_uint8),
      ('minor', ctypes.c_uint8),
      ('build', ctypes.c_uint8),
      ('reserved', ctypes.c_uint8),
      ('release', ctypes.c_uint16),
  ]


class _SMC_KeyDataLimits(ctypes.Structure):
  _fields_ = [
      ('version', ctypes.c_uint16),
      ('length', ctypes.c_uint16),
      ('cpu', ctypes.c_uint32),
      ('gpu', ctypes.c_uint32),
      ('mem', ctypes.c_uint32),
  ]


class _SMC_KeyDataInfo(ctypes.Structure):
  _fields_ = [
      ('size', ctypes.c_uint32),
      ('type', ctypes.c_uint32),
      ('attributes', ctypes.c_uint8),
  ]


class _SMC_KeyData(ctypes.Structure):
  _fields_ = [
      ('key', ctypes.c_uint32),
      ('version', _SMC_KeyDataVersion),
      ('pLimitData', _SMC_KeyDataLimits),
      ('keyInfo', _SMC_KeyDataInfo),
      ('result', ctypes.c_uint8),
      ('status', ctypes.c_uint8),
      ('data8', ctypes.c_uint8),
      ('data32', ctypes.c_uint32),
      ('bytes', ctypes.c_ubyte * 32),
  ]


class _SMC_Value(ctypes.Structure):
  _fields_ = [
      ('key', (ctypes.c_ubyte * 5)),
      ('size', ctypes.c_uint32),
      ('type', (ctypes.c_ubyte * 5)),
      ('bytes', ctypes.c_ubyte * 32),
  ]


# http://bxr.su/OpenBSD/sys/dev/isa/asmc.c is a great list of sensors.
#
# The following other sensor were found on a MBP 2012 via brute forcing but
# their signification is unknown:
#   TC0E, TC0F, TH0A, TH0B, TH0V, TP0P, TS0D, TS0P
_sensor_names = {
  'TA0P': u'ambient', # u'hdd bay 1',
  #'TA0S': u'pci slot 1 pos 1',
  #'TA1S': u'pci slot 1 pos 2',
  #'TA3S': u'pci slot 2 pos 2',
  'TB0T': u'enclosure bottom',
  #'TB2T': u'enclosure bottom 3',
  #'TC0D': u'cpu0 die core',
  'TC0P': u'cpu', # u'cpu0 proximity'
  #'TC1D': u'cpu1',
  #'TCAH': u'cpu0',
  #'TCDH': u'cpu3',
  'TG0D': u'gpu', # u'gpu0 diode',
  #'TG0P': u'gpu0 proximity',
  #'TG1H': u'gpu heatsink 2',
  #'TH0P': u'hdd bay 1',
  #'TH2P': u'hdd bay 3',
  #'TL0P': u'lcd proximity',
  #'TM0P': u'mem bank a1',
  #'TM1P': u'mem bank a2',
  #'TM2P': u'mem bank a3',
  #'TM3P': u'mem bank a4',
  #'TM4P': u'mem bank a5',
  #'TM5P': u'mem bank a6',
  #'TM6P': u'mem bank a7',
  #'TM7P': u'mem bank a8',
  #'TM8P': u'mem bank b1',
  #'TM9P': u'mem bank b2',
  #'TMA1': u'ram a1',
  #'TMA3': u'ram a3',
  #'TMAP': u'mem bank b3',
  #'TMB1': u'ram b1',
  #'TMB3': u'ram b3',
  #'TMBP': u'mem bank b4',
  #'TMCP': u'mem bank b5',
  #'TMDP': u'mem bank b6',
  #'TMEP': u'mem bank b7',
  #'TMFP': u'mem bank b8',
  #'TN0D': u'northbridge die core',
  #'TN0P': u'northbridge proximity',
  #'TO0P': u'optical drive',
  #'TW0P': u'wireless airport card',
  #'Th0H': u'main heatsink a',
  #'Th2H': u'main heatsink c',
  #'Tm0P': u'memory controller',
  'Tp0C': u'power supply',
  #'Tp1C': u'power supply 2',
  #'Tp2P': u'power supply 3',
  #'Tp4P': u'power supply 5',
  #'TA1P': u'ambient 2',
  #'TA2S': u'pci slot 2 pos 1',
  #'TB1T': u'enclosure bottom 2',
  #'TB3T': u'enclosure bottom 4',
  #'TC0H': u'cpu0 heatsink',
  #'TC2D': u'cpu2',
  #'TC3D': u'cpu3',
  #'TCBH': u'cpu1',
  #'TCCH': u'cpu2',
  #'TG0H': u'gpu0 heatsink',
  #'TH1P': u'hdd bay 2',
  #'TH3P': u'hdd bay 4',
  #'TM0S': u'mem module a1',
  #'TM1S': u'mem module a2',
  #'TM2S': u'mem module a3',
  #'TM3S': u'mem module a4',
  #'TM4S': u'mem module a5',
  #'TM5S': u'mem module a6',
  #'TM6S': u'mem module a7',
  #'TM7S': u'mem module a8',
  #'TM8S': u'mem module b1',
  #'TM9S': u'mem module b2',
  #'TMA2': u'ram a2',
  #'TMA4': u'ram a4',
  #'TMAS': u'mem module b3',
  #'TMB2': u'ram b2',
  #'TMB4': u'ram b4',
  #'TMBS': u'mem module b4',
  #'TMCS': u'mem module b5',
  #'TMDS': u'mem module b6',
  #'TMES': u'mem module b7',
  #'TMFS': u'mem module b8',
  #'TN0H': u'northbridge',
  #'TN1P': u'northbridge 2',
  #'TS0C': u'expansion slots',
  #'Th1H': u'main heatsink b',
  #'Tp0P': u'power supply 1',
  #'Tp1P': u'power supply 2',
  #'Tp3P': u'power supply 4',
  #'Tp5P': u'power supply 6',
}

# _sensor_found_cache is set on the first call to _SMC_get_values.
_sensor_found_cache = None


@tools.cached
def _SMC_open():
  """Opens the default SMC driver and returns the first device.

  It leaves the device handle open for the duration of the process.
  """
  # There should be only one.
  itr = ctypes.c_uint()
  result = iokit.IOServiceGetMatchingServices(
      0, iokit.IOServiceMatching('AppleSMC'), ctypes.byref(itr))
  if result:
    logging.error('failed to get AppleSMC (%d)', result)
    return None
  dev = iokit.IOIteratorNext(itr)
  iokit.IOObjectRelease(itr)
  if not dev:
    logging.error('no SMC found')
    return None
  conn = ctypes.c_uint()
  if iokit.IOServiceOpen(dev, libkern.mach_task_self(), 0, ctypes.byref(conn)):
    logging.error('failed to open AppleSMC (%d)', result)
    return None
  return conn


def _SMC_call(conn, index, indata, outdata):
  """Executes a call to the SMC subsystem."""
  return iokit.IOConnectCallStructMethod(
      conn,
      ctypes.c_uint(index),
      ctypes.cast(ctypes.pointer(indata), ctypes.c_void_p),
      ctypes.c_ulonglong(ctypes.sizeof(_SMC_KeyData)),
      ctypes.cast(ctypes.pointer(outdata), ctypes.c_void_p),
      ctypes.pointer(ctypes.c_ulonglong(ctypes.sizeof(_SMC_KeyData))))


def _SMC_read_key(conn, key):
  """Retrieves an unprocessed key value."""
  KERNEL_INDEX_SMC = 2

  # Call with SMC_CMD_READ_KEYINFO.
  # TODO(maruel): Keep cache of result size.
  indata = _SMC_KeyData(key=struct.unpack('>i', key)[0], data8=9)
  outdata = _SMC_KeyData()
  if _SMC_call(conn, KERNEL_INDEX_SMC, indata, outdata):
    logging.error('SMC call to get key info failed')
    return None

  # Call with SMC_CMD_READ_BYTES.
  val = _SMC_Value(size=outdata.keyInfo.size)
  for i, x in enumerate(struct.pack('>i', outdata.keyInfo.type)):
    val.type[i] = ord(x)
  # pylint: disable=attribute-defined-outside-init
  indata.data8 = 5
  indata.keyInfo.size = val.size
  if _SMC_call(conn, KERNEL_INDEX_SMC, indata, outdata):
    logging.error('SMC call to get data info failed')
    return None
  val.bytes = outdata.bytes
  return val


def _SMC_get_value(conn, key):
  """Returns a processed measurement via AppleSMC for a specified key.

  Returns None on failure.
  """
  val = _SMC_read_key(conn, key)
  if not val or not val.size:
    return None
  t = ''.join(map(chr, val.type))
  if t == 'sp78\0' and val.size == 2:
    # Format is first byte signed int8, second byte uint8 fractional.
    return round(
        float(ctypes.c_int8(val.bytes[0]).value) + (val.bytes[1] / 256.),
        2)
  if t == 'fpe2\0' and val.size == 2:
    # Format is unsigned 14 bits big endian, 2 bits fractional.
    return round(
        (float((val.bytes[0] << 6) + (val.bytes[1] >> 2)) +
        (val.bytes[1] & 3) / 4.),
        2)
  # TODO(maruel): Handler other formats like 64 bits long. This is used for fan
  # speed.
  logging.error(
      '_SMC_get_value(%s) got unknown format: %s of %d bytes', key, t, val.size)
  return None


def _get_system_profiler(data_type):
  """Returns an XML about the system display properties."""
  sp = subprocess.check_output(
      ['system_profiler', data_type, '-xml'])
  return plistlib.readPlistFromString(sp)[0].get('_items', [])


@tools.cached
def _get_libc():
  ctypes.cdll.LoadLibrary('/usr/lib/libc.dylib')
  return ctypes.CDLL('/usr/lib/libc.dylib')


def _sysctl(ctl, item, result):
  """Calls sysctl. Ignores return value."""
  # https://developer.apple.com/library/mac/documentation/Darwin/Reference/ManPages/man3/sysctl.3.html
  # http://opensource.apple.com/source/xnu/xnu-1699.24.8/bsd/sys/sysctl.h
  arr = (ctypes.c_int * 2)(ctl, item)
  size = ctypes.c_size_t(ctypes.sizeof(result))
  _get_libc().sysctl(
      arr, len(arr), ctypes.byref(result), ctypes.byref(size),
      ctypes.c_void_p(), ctypes.c_size_t(0))


class _timeval(ctypes.Structure):
  _fields_ = [('tv_sec', ctypes.c_long), ('tv_usec', ctypes.c_int)]


@tools.cached
def _get_xcode_version(xcode_app):
  """Returns the version of Xcode installed at the given path.

  Args:
    xcode_app: Absolute path to the Xcode app directory, e.g
               /Applications/Xcode.app.

  Returns:
    A tuple of (Xcode version, Xcode build version), or None if
    this is not an Xcode installation.
  """
  xcodebuild = os.path.join(
      xcode_app, 'Contents', 'Developer', 'usr', 'bin', 'xcodebuild')
  if os.path.exists(xcodebuild):
    try:
      out = subprocess.check_output([xcodebuild, '-version']).splitlines()
    except subprocess.CalledProcessError:
      return None
    return out[0].split()[-1], out[1].split()[-1]


## Public API.


def get_xcode_state():
  """Returns the state of Xcode installations on this machine."""
  state = {}
  applications_dir = os.path.join('/Applications')
  for app in os.listdir(applications_dir):
    name, ext = os.path.splitext(app)
    if name.startswith('Xcode') and ext == '.app':
      xcode_app = os.path.join(applications_dir, app)
      version = _get_xcode_version(xcode_app)
      if version:
        state[xcode_app] = {
          'version': version[0],
          'build version': version[1],
        }
        device_support_dir = os.path.join(
            xcode_app, 'Contents', 'Developer', 'Platforms',
            'iPhoneOS.platform', 'DeviceSupport')
        if os.path.exists(device_support_dir):
          state[xcode_app]['device support'] = os.listdir(device_support_dir)
  return state


def get_xcode_versions():
  """Returns a list of Xcode versions installed on this machine."""
  return sorted(
      set(xcode['version'] for xcode in get_xcode_state().itervalues()))


def get_current_xcode_version():
  """Returns the active version of Xcode."""
  try:
    out = subprocess.check_output(['xcodebuild', '-version']).splitlines()
  except (OSError, subprocess.CalledProcessError):
    return None
  return out[0].split()[-1], out[1].split()[-1]


def get_ios_device_ids():
  """Returns a list of UDIDs of attached iOS devices.

  Requires idevice_id in $PATH. idevice_id is part of libimobiledevice.
  See http://libimobiledevice.org.
  """
  try:
    return subprocess.check_output(['idevice_id', '--list']).splitlines()
  except (OSError, subprocess.CalledProcessError):
    return []


def get_ios_version(udid):
  """Returns the OS version of the specified iOS device.

  Requires ideviceinfo in $PATH. ideviceinfo is part of libimobiledevice.
  See http://libimobiledevice.org.

  Args:
    udid: UDID string as returned by get_ios_device_ids.
  """
  try:
    out = subprocess.check_output(
        ['ideviceinfo', '-k', 'ProductVersion', '-u', udid]).splitlines()
    if len(out) == 1:
      return out[0]
  except (OSError, subprocess.CalledProcessError):
    pass


@tools.cached
def get_ios_device_type(udid):
  """Returns the type of the specified iOS device.

  Requires ideviceinfo in $PATH. ideviceinfo is part of libimobiledevice.
  See http://libimobiledevice.org.

  Args:
    udid: UDID string as returned by get_ios_device_ids.
  """
  try:
    out = subprocess.check_output(
        ['ideviceinfo', '-k', 'ProductType', '-u', udid]).splitlines()
    if len(out) == 1:
      return out[0]
  except (OSError, subprocess.CalledProcessError):
    pass


@tools.cached
def get_hardware_model_string():
  """Returns the Mac model string.

  Returns:
    A string like Macmini5,3 or MacPro6,1.
  """
  try:
    return unicode(
        subprocess.check_output(['sysctl', '-n', 'hw.model']).rstrip())
  except (OSError, subprocess.CalledProcessError):
    return None


@tools.cached
def get_os_version_number():
  """Returns the normalized OS version number as a string.

  Returns:
    Version as a string like '10.12.4'
  """
  version_parts = platform.mac_ver()[0].split('.')
  assert len(version_parts) >= 2, 'Unable to determine Mac version'
  return u'.'.join(version_parts)


@tools.cached
def get_audio():
  """Returns the audio cards that are "connected"."""
  return [
    card['_name'] for card in _get_system_profiler('SPAudioDataType')
    if card.get('coreaudio_default_audio_output_device') == 'spaudio_yes'
  ]


def get_gpu():
  """Returns video device as listed by 'system_profiler'.

  Not cached as the GPU driver may change underneat.
  """
  dimensions = set()
  state = set()
  for card in _get_system_profiler('SPDisplaysDataType'):
    # Warning: the value provided depends on the driver manufacturer.
    # Other interesting values: spdisplays_vram, spdisplays_revision-id
    ven_id = None
    if 'spdisplays_vendor-id' in card:
      # NVidia
      ven_id = card['spdisplays_vendor-id'][2:]
    elif 'spdisplays_vendor' in card:
      # Intel and ATI
      match = re.search(r'\(0x([0-9a-f]{4})\)', card['spdisplays_vendor'])
      if match:
        ven_id = match.group(1)
    dev_id = card['spdisplays_device-id'][2:]

    # Looks like: u'4.0.20 [3.2.8]'
    version = unicode(card.get('spdisplays_gmux-version', u''))

    # VMWare doesn't set it.
    dev_name = unicode(card.get('sppci_model', u''))
    ven_name = ''
    if dev_name:
      # The first word is pretty much always the company name on OSX.
      split_name = dev_name.split(' ', 1)
      ven_name = split_name[0]
      dev_name = split_name[1]

    # macOS 10.13 stopped including the vendor ID in the spdisplays_vendor
    # string. Infer it from the vendor name instead.
    if not ven_id:
      ven_id = gpu.vendor_name_to_id(ven_name)
    if not ven_id:
      ven_id = u'UNKNOWN'
    ven_name, dev_name = gpu.ids_to_names(ven_id, ven_name, dev_id, dev_name)

    dimensions.add(unicode(ven_id))
    dimensions.add(u'%s:%s' % (ven_id, dev_id))
    if version:
      match = re.search(r'([0-9.]+) \[([0-9.]+)\]', version)
      if match:
        dimensions.add(u'%s:%s-%s-%s' % (
            ven_id, dev_id, match.group(1), match.group(2)))
      state.add(u'%s %s %s' % (ven_name, dev_name, version))
    else:
      state.add(u'%s %s' % (ven_name, dev_name))
  return sorted(dimensions), sorted(state)


@tools.cached
def get_cpuinfo():
  """Returns CPU information."""
  values = common._safe_parse(
      subprocess.check_output(['sysctl', 'machdep.cpu']))
  # http://unix.stackexchange.com/questions/43539/what-do-the-flags-in-proc-cpuinfo-mean
  return {
    u'flags': sorted(
        i.lower() for i in values[u'machdep.cpu.features'].split()),
    u'model': [
      int(values['machdep.cpu.family']), int(values['machdep.cpu.model']),
      int(values['machdep.cpu.stepping']),
      int(values['machdep.cpu.microcode_version']),
    ],
    u'name': values[u'machdep.cpu.brand_string'],
    u'vendor': values[u'machdep.cpu.vendor'],
  }


def get_temperatures():
  """Returns the temperatures in Celsius."""
  global _sensor_found_cache
  conn = _SMC_open()
  if not conn:
    return None

  out = {}
  if _sensor_found_cache is None:
    _sensor_found_cache = set()
    # Populate the cache of the sensors found on this system, so that the next
    # call can only get the actual sensors.
    # Note: It is relatively fast to brute force all the possible names.
    for key, name in _sensor_names.iteritems():
      value = _SMC_get_value(conn, key)
      if value is not None:
        _sensor_found_cache.add(key)
        out[name] = value
    return out

  for key in _sensor_found_cache:
    value = _SMC_get_value(conn, key)
    if value is not None:
      out[_sensor_names[key]] = value
  return out


@tools.cached
def get_monitor_hidpi():
  """Returns True if the monitor is hidpi.

  On 10.12.3 and earlier, the following could be used to detect an hidpi
  display:
    <key>spdisplays_retina</key>
    <string>spdisplays_yes</string>

  On 10.12.4 and later, the key above doesn't exist anymore. Fall back to search
  for:
    <key>spdisplays_display_type</key>
    <string>spdisplays_built-in_retinaLCD</string>
  """
  def is_hidpi(displays):
    return any(
        d.get('spdisplays_retina') == 'spdisplays_yes' or
        'retina' in d.get('spdisplays_display_type', '').lower()
        for d in displays)

  hidpi = any(
    is_hidpi(card['spdisplays_ndrvs'])
    for card in _get_system_profiler('SPDisplaysDataType')
    if 'spdisplays_ndrvs' in card)
  return unicode(int(hidpi))


def get_xcode_versions():
  """Returns all Xcode versions installed."""
  return sorted(
      unicode(xcode['version']) for xcode in get_xcode_state().itervalues())


@tools.cached
def get_physical_ram():
  """Returns the amount of installed RAM in Mb, rounded to the nearest number.
  """
  CTL_HW = 6
  HW_MEMSIZE = 24
  result = ctypes.c_uint64(0)
  _sysctl(CTL_HW, HW_MEMSIZE, result)
  return int(round(result.value / 1024. / 1024.))


def get_uptime():
  """Returns uptime in seconds since system startup.

  Includes sleep time.
  """
  CTL_KERN = 1
  KERN_BOOTTIME = 21
  result = _timeval()
  _sysctl(CTL_KERN, KERN_BOOTTIME, result)
  start = float(result.tv_sec) + float(result.tv_usec) / 1000000.
  return time.time() - start


def is_locked():
  """Returns whether the lock screen is currently displayed or not.

  Returns:
    None, False or True. It is None when the state of the GUI cannot be
    determined, and a bool otherwise returning the state of the GUI.
  """
  if Quartz is None:
    return None
  current_session = Quartz.CGSessionCopyCurrentDictionary()
  if not current_session:
    return None
  return bool(current_session.get('CGSSessionScreenIsLocked', False))


@tools.cached
def get_ssd():
  """Returns a list of SSD disks."""
  try:
    out = subprocess.check_output(['diskutil', 'list', '-plist', 'physical'])
    pl = plistlib.readPlistFromString(out)
    ssd = []
    for disk in pl['WholeDisks']:
      out = subprocess.check_output(['diskutil', 'info', '-plist', disk])
      pl = plistlib.readPlistFromString(out)
      if pl['SolidState']:
        ssd.append(disk.decode('utf-8'))
    return tuple(sorted(ssd))
  except (OSError, subprocess.CalledProcessError) as e:
    logging.error('Failed to read disk info: %s', e)
    return ()


## Mutating code.


def generate_launchd_plist(command, cwd, plistname):
  """Generates a plist content with the corresponding command for launchd."""
  # The documentation is available at:
  # https://developer.apple.com/library/mac/documentation/Darwin/Reference/ \
  #    ManPages/man5/launchd.plist.5.html
  escaped_plist = cgi.escape(plistname)
  entries = [
    '<key>Label</key><string>%s</string>' % escaped_plist,
    '<key>StandardOutPath</key><string>logs/bot_stdout.log</string>',
    '<key>StandardErrorPath</key><string>logs/bot_stderr.log</string>',
    '<key>LimitLoadToSessionType</key><array><string>Aqua</string></array>',
    '<key>RunAtLoad</key><true/>',
    '<key>Umask</key><integer>18</integer>',

    # https://developer.apple.com/library/mac/documentation/Darwin/Reference/ManPages/man5/launchd.plist.5.html
    # launchd expects the process to call vproc_transaction_begin(), which we
    # don't. Otherwise it sends SIGKILL instead of SIGTERM, which isn't nice.
    '<key>EnableTransactions</key>',
    '<false/>',

    '<key>EnvironmentVariables</key>',
    '<dict>',
    '  <key>PATH</key>',
    # TODO(maruel): Makes it configurable if necessary.
    '  <string>/opt/local/bin:/opt/local/sbin:/usr/local/sbin'
    ':/usr/local/git/bin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin</string>',
    '</dict>',

    '<key>SoftResourceLimits</key>',
    '<dict>',
    '  <key>NumberOfFiles</key>',
    '  <integer>8000</integer>',
    '</dict>',

    '<key>KeepAlive</key>',
    '<dict>',
    '  <key>SuccessfulExit</key>',
    '  <false/>',
    '</dict>',
  ]
  entries.append(
      '<key>Program</key><string>%s</string>' % cgi.escape(command[0]))
  entries.append('<key>ProgramArguments</key>')
  entries.append('<array>')
  # Command[0] must be passed as an argument.
  entries.extend('  <string>%s</string>' % cgi.escape(i) for i in command)
  entries.append('</array>')
  entries.append(
      '<key>WorkingDirectory</key><string>%s</string>' % cgi.escape(cwd))
  header = (
    '<?xml version="1.0" encoding="UTF-8"?>\n'
    '<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" '
    '"http://www.apple.com/DTDs/PropertyList-1.0.dtd">\n'
    '<plist version="1.0">\n'
    '  <dict>\n'
    + ''.join('    %s\n' % l for l in entries) +
    '  </dict>\n'
    '</plist>\n')
  return header
