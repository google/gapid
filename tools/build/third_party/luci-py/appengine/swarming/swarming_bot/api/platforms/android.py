# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Android specific utility functions.

This file serves as an API to bot_config.py. bot_config.py can be replaced on
the server to allow additional server-specific functionality.
"""

import collections
import logging
import os
import time

try:
  from adb import adb_protocol
  from adb import common
  from adb.contrib import adb_commands_safe
  from adb.contrib import high
  from api import parallel
  from api.platforms import gce

  # Master switch that can easily be temporarily increased to INFO or even DEBUG
  # when needed by simply pushing a new tainted swarming server version. This
  # helps quickly debugging issues. On the other hand, even INFO level is quite
  # verbose so keep it at WARNING by default.
  LEVEL = logging.WARNING
  adb_commands_safe._LOG.setLevel(LEVEL)
  adb_protocol._LOG.setLevel(LEVEL)
  common._LOG.setLevel(LEVEL)
  high._LOG.setLevel(LEVEL)
except OSError:
  # This can fail on macOS if libusb-1.0.dylib is not installed.
  pass


# This list of third party apps embedded in the base OS image varies from
# version to version.
KNOWN_APPS = frozenset([
    'android',
    'android.autoinstalls.config.google.nexus',
    'com.frogmind.badland',
    'com.hp.android.printservice',
    'com.huawei.callstatisticsutils',
    'com.huawei.entitlement',
    'com.huawei.mmitest',
    'com.huawei.sarcontrolservice',
    'com.lge.HiddenMenu',
    'com.lge.SprintHiddenMenu',
    'com.lge.entitlement',
    'com.lge.lifetimer',
    'com.lge.update',
    'com.mediatek.fmradio',
    'com.mediatek.lbs.em2.ui',
    'com.motorola.android.buacontactadapter',
    'com.motorola.appdirectedsmsproxy',
    'com.motorola.entitlement',
    'com.motorola.motocit',
    'com.motorola.motosignature.app',
    'com.motorola.triggerenroll',
    'com.motorola.triggertrainingservice',
    'com.nuance.xt9.input',
    'com.nvidia.NvCPLSvc',
    'com.nvidia.NvCPLUpdater',
    'com.nvidia.benchmarkblocker',
    'com.nvidia.blakepairing',
    'com.nvidia.feedback',
    'com.nvidia.nvcecservice',
    'com.nvidia.nvgamecast',
    'com.nvidia.osc',
    'com.nvidia.ota',
    'com.nvidia.shield.nvcustomize',
    'com.nvidia.shield.welcome',
    'com.nvidia.shieldservice',
    'com.nvidia.stats',
    'com.nvidia.tegraprofiler.security',
    'com.nvidia.tegrazone3',
    'com.plexapp.android',
    'com.qti.qualcomm.datastatusnotification',
    'com.qualcomm.atfwd',
    'com.qualcomm.cabl',
    'com.qualcomm.embms',
    'com.qualcomm.qcrilmsgtunnel',
    'com.qualcomm.qti.rcsbootstraputil',
    'com.qualcomm.qti.rcsimsbootstraputil',
    'com.qualcomm.shutdownlistner',
    'com.qualcomm.timeservice',
    'com.quicinc.cne.CNEService',
    'com.quickoffice.android',
    'com.redbend.vdmc',
    'com.verizon.omadm',
    'com.vzw.apnservice',
    'com.yodo1.crossyroad',
    'jp.co.omronsoft.iwnnime.ml',
    'jp.co.omronsoft.iwnnime.ml.kbd.white',
    'org.codeaurora.ims',
    'org.simalliance.openmobileapi.service',
])


def get_unknown_apps(device):
  return [
      p for p in device.GetPackages() or []
      if (not p.startswith(('com.android.', 'com.google.')) and
          p not in KNOWN_APPS)
  ]


def initialize(pub_key, priv_key):
  return high.Initialize(pub_key, priv_key)


# TODO(bpastene): Remove bot arg when call site has been updated.
def get_devices(bot=None, endpoints=None, enable_resets=False):
  # pylint: disable=unused-argument
  devices = []
  if not gce.is_gce():
    devices += high.GetLocalDevices(
      'swarming', 10000, 10000, as_root=False, enable_resets=enable_resets)

  if endpoints:
    devices += high.GetRemoteDevices(
        'swarming', endpoints, 10000, 10000, as_root=False)

  return devices


def close_devices(devices):
  return high.CloseDevices(devices)


def kill_adb():
  return adb_commands_safe.KillADB()


def get_dimensions(devices):
  """Returns the default dimensions for an host with multiple android devices.
  """
  dimensions = {}
  start = time.time()
  # Each key in the following dict is a dimension and its value is the list of
  # all possible device properties that can define that dimension.
  # TODO(bpastene) Make sure all the devices use the same board and OS.
  # product.device should be read (and listed) first, that is, before
  # build.product because the latter is deprecated.
  # https://android.googlesource.com/platform/build/+/master/tools/buildinfo.sh
  dimension_properties = {
    u'device_os': ['build.id'],
    u'device_os_flavor': ['product.brand'],
    u'device_type': ['product.device', 'build.product', 'product.board'],
  }
  for dim in dimension_properties:
    dimensions[dim] = set()

  dimensions[u'android'] = []
  for device in devices:
    properties = device.cache.build_props
    if properties:
      for dim, props in dimension_properties.iteritems():
        for prop in props:
          real_prop = u'ro.' + prop
          if real_prop in properties:
            p = properties[real_prop].strip()
            if p and p not in dimensions[dim]:
              dimensions[dim].add(p)
              # In the past, we would break here to have only one device_type.
              # This can be re-introduced if needed, but for some devices
              # (e.g. 2017 NVidia Shield, aka darcy) there are different values
              # for product.device and build.product (e.g. darcy and foster
              # [the device_type of the 2015 Shield]). Rather than knowing
              # which of the three keys is "correct", just report all of them.
      # Only advertize devices that can be used.
      dimensions[u'android'].append(device.serial)

  # Add the first character of each device_os to the dimension.
  android_vers = {
    os[0] for os in dimensions.get(u'device_os', []) if os and os[0].isupper()
  }
  dimensions[u'device_os'] = dimensions[u'device_os'].union(android_vers)
  dimensions[u'android'].sort()
  for dim in dimension_properties:
    if not dimensions[dim]:
      del dimensions[dim]
    else:
      dimensions[dim] = sorted(dimensions[dim])

  # Tweaks the 'product.brand' prop to be a little more readable.
  if dimensions.get(u'device_os_flavor'):
    def _fix_flavor(flavor):
      flavor = flavor.lower()
      # Non-aosp stock android is reported as 'google'. Other OEMs that ship
      # their own images are free to report what they want. (eg: Nvidia Shield
      # is reported as 'NVIDIA'.
      return 'aosp' if flavor == 'android' else flavor

    dimensions[u'device_os_flavor'] = list(
        map(_fix_flavor, dimensions[u'device_os_flavor']))

  nb_android = len(dimensions[u'android'])
  dimensions[u'android_devices'] = map(
      str, range(nb_android, max(0, nb_android-4), -1))

  # TODO(maruel): Add back once dimensions limit is figured out and there's a
  # need.
  del dimensions[u'android']

  # Trim 'os' to reduce the number of dimensions and not run tests by accident
  # on it.
  dimensions[u'os'] = ['Android']

  logging.info(
      'get_dimensions() (device part) took %gs' %
      round(time.time() - start, 1))

  def _get_package_versions(package):
    versions = set()
    for device in devices:
      version = device.GetPackageVersion(package)
      if version:
        versions.add(version)
    return sorted(versions)

  # Add gms core and Playstore versions
  dimensions[u'device_gms_core_version'] = (
      _get_package_versions('com.google.android.gms') or ['unknown'])
  dimensions[u'device_playstore_version'] = (
      _get_package_versions('com.android.vending') or ['unknown'])

  return dimensions


def get_state(devices):
  """Returns state information about all the devices connected to the host.
  """
  keys = (
    u'board.platform',
    u'build.product',
    u'build.fingerprint',
    u'build.id',
    u'build.version.sdk',
    u'product.board',
    u'product.cpu.abi',
    u'product.device')

  def fn(device):
    if not device.is_valid or device.failure:
      return {u'state': device.failure or 'unavailable'}
    properties = device.cache.build_props
    if not properties:
      return {u'state': 'unavailable'}
    no_sd_card = properties.get(u'ro.product.model', '') in ['Chromecast']
    return {
      u'battery': device.GetBattery(),
      u'build': {key: properties.get(u'ro.'+key, '<missing>') for key in keys},
      u'cpu': device.GetCPUScale(),
      u'disk': device.GetDisk(),
      u'imei': device.GetIMEI(),
      u'ip': device.GetIPs(),
      u'max_uid': device.GetLastUID(),
      u'mem': device.GetMemInfo(),
      u'other_packages': get_unknown_apps(device),
      u'port_path': device.port_path,
      u'processes': device.GetProcessCount(),
      u'state': (u'available' if
          no_sd_card or device.IsFullyBooted()[0] else u'booting'),
      u'temp': device.GetTemperatures(),
      u'uptime': device.GetUptime(),
    }

  start = time.time()
  state = {
      u'devices': {
          device.serial: out
          for device, out in zip(devices, parallel.pmap(fn, devices))
      }
  }
  logging.info(
      'get_state() (device part) took %gs' %
      round(time.time() - start, 1))
  return state
