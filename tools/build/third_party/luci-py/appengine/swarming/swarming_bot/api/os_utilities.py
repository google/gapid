# coding: utf-8
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""OS specific utility functions.

Includes code:
- to declare the current system this code is running under.
- to run a command on user login.
- to reboot the host.

This file serves as an API to bot_config.py. bot_config.py can be replaced on
the server to allow additional server-specific functionality.
"""

import getpass
import glob
import json
import locale
import logging
import multiprocessing
import os
import pipes
import platform
import re
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib
import urllib2

from api import parallel
from api import platforms
from utils import file_path
from utils import fs
from utils import tools
from third_party import httplib2
from third_party.oauth2client import client


# For compatibility with older bot_config.py files.
cached = tools.cached


# https://cloud.google.com/compute/pricing#machinetype
GCE_MACHINE_COST_HOUR_US = {
  u'n1-standard-1': 0.050,
  u'n1-standard-2': 0.100,
  u'n1-standard-4': 0.200,
  u'n1-standard-8': 0.400,
  u'n1-standard-16': 0.800,
  u'n1-standard-32': 1.600,
  u'f1-micro': 0.008,
  u'g1-small': 0.027,
  u'n1-highmem-2': 0.126,
  u'n1-highmem-4': 0.252,
  u'n1-highmem-8': 0.504,
  u'n1-highmem-16': 1.008,
  u'n1-highmem-32': 2.016,
  u'n1-highcpu-2': 0.076,
  u'n1-highcpu-4': 0.152,
  u'n1-highcpu-8': 0.304,
  u'n1-highcpu-16': 0.608,
  u'n1-highcpu-32': 1.216,
}


# https://cloud.google.com/compute/pricing#machinetype
GCE_MACHINE_COST_HOUR_EUROPE_ASIA = {
  u'n1-standard-1': 0.055,
  u'n1-standard-2': 0.110,
  u'n1-standard-4': 0.220,
  u'n1-standard-8': 0.440,
  u'n1-standard-16': 0.880,
  u'n1-standard-32': 1.760,
  u'f1-micro': 0.009,
  u'g1-small': 0.030,
  u'n1-highmem-2': 0.139,
  u'n1-highmem-4': 0.278,
  u'n1-highmem-8': 0.556,
  u'n1-highmem-16': 1.112,
  u'n1-highmem-32': 2.224,
  u'n1-highcpu-2': 0.084,
  u'n1-highcpu-4': 0.168,
  u'n1-highcpu-8': 0.336,
  u'n1-highcpu-16': 0.672,
  u'n1-highcpu-32': 1.344,
}


GCE_RAM_GB_PER_CORE_RATIOS = {
  0.9: u'n1-highcpu-',
  3.75: u'n1-standard-',
  6.5: u'n1-highmem-',
}


# https://cloud.google.com/compute/pricing#disk
GCE_HDD_GB_COST_MONTH = 0.04
GCE_SSD_GB_COST_MONTH = 0.17


# https://cloud.google.com/compute/pricing#premiumoperatingsystems
GCE_WINDOWS_COST_CORE_HOUR = 0.04


MONITORING_ENDPOINT = 'https://www.googleapis.com/cloudmonitoring/v2beta2'
MONITORING_SCOPES = ['https://www.googleapis.com/auth/monitoring']


### Private stuff.


# Used to calculated Swarming bot uptime.
_STARTED_TS = time.time()


def _write(filepath, content):
  """Writes out a file and returns True on success."""
  logging.info('Writing in %s:\n%s', filepath, content)
  try:
    with open(filepath, mode='wb') as f:
      f.write(content)
    return True
  except IOError as e:
    logging.error('Failed to write %s: %s', filepath, e)
    return False


def _safe_read(filepath):
  """Returns the content of the file if possible, None otherwise."""
  try:
    with open(filepath, 'rb') as f:
      return f.read()
  except (IOError, OSError):
    return None


### Public API.


def get_os_values():
  """Returns the values to use for 'os' dimension as a list.

  Note that we don't apply @tools.cached decorator since all heavy calls made
  by this function are already cached. By omitting the decorator we are building
  a new list object each time, so callers can safely modify it.
  """
  os_name = get_os_name()
  out = [os_name]
  if sys.platform == 'win32':
    # On Windows, do not use the version numbers, use the marketing name
    # instead.
    names = platforms.win.get_os_version_names()
    out.extend(u'%s-%s' % (os_name, n) for n in names)
  elif sys.platform == 'cygwin':
    # ... except on cygwin.
    out.append(u'%s-%s' % (os_name, platforms.win.get_os_version_number()))
  elif sys.platform == 'darwin':
    # Expects '10.a.b'. Add both '10.a' and '10.a.b'.
    number = platforms.osx.get_os_version_number()
    out.append(u'%s-%s' % (os_name, number.rsplit('.', 1)[0]))
    out.append(u'%s-%s' % (os_name, number))
  else:
    # TODO(maruel): Get rid of this, Linux is not an OS, it's a kernel.
    out.append(u'Linux')
    number = platforms.linux.get_os_version_number()
    out.append(u'%s-%s' % (os_name, number))
  out.sort()
  return out


@tools.cached
def get_os_name():
  """Returns standardized OS name.

  Defaults to sys.platform for OS not normalized.

  Returns:
    Windows, Mac, Ubuntu, Raspbian, etc.
  """
  value = {
    'cygwin': u'Windows',
    # TODO(maruel): 'Mac' is an historical accident, it should be named 'OSX'.
    'darwin': u'Mac',
    'win32': u'Windows',
  }.get(sys.platform)
  if value:
    return value

  if sys.platform == 'linux2':
    # Try to figure out the distro. Supported distros are Debian, Ubuntu,
    # Raspbian.
    # Add support for other OSes as relevant.
    content = _safe_read('/etc/os-release')
    if content:
      os_release = dict(l.split('=', 1) for l in content.splitlines() if l)
      os_id = os_release.get('ID').strip('"')
      # Uppercase the first letter for consistency with the other platforms.
      return unicode(os_id[0].upper() + os_id[1:])

  return unicode(sys.platform)


@tools.cached
def get_cpu_type():
  """Returns the type of processor: armv6l, armv7l, arm64 or x86."""
  machine = platform.machine().lower()
  if machine in ('amd64', 'x86_64', 'i386'):
    return u'x86'
  if machine == 'aarch64':
    return u'arm64'
  if machine == 'mips64':
    return u'mips'
  return unicode(machine)


@tools.cached
def get_cpu_bitness():
  """Returns the number of bits in the CPU architecture as a str: 32 or 64.

  Unless someone ported python to PDP-10 or 286.

  OSX or Windows, we don't care about userland, we report the bitness of the
  kernel.
  On other platforms (Linux), we explicitly want to report 32 bits userland,
  independent of the kernel bitness.
  """
  if sys.platform in ('darwin', 'win32') and platform.machine().endswith('64'):
    return u'64'
  return u'64' if sys.maxsize > 2**32 else u'32'


def _parse_intel_model(name):
  """Tries to extract the CPU model name from the display name.

  The actual format varies a bit across products but is consistent across OSes.
  """
  # List of regexp to parse Intel model name. It is simpler to have multiple
  # regexp than try to make one that matches them all.
  # They shall be in decreasing order of precision.
  regexps = [
    ur' ([a-zA-Z]\d-\d{4}[A-Z]{0,2} [vV]\d) ',
    ur' ([a-zA-Z]\d-\d{4}[A-Z]{0,2}) ',
    ur' ([A-Z]\d{4}[A-Z]{0,2}) ',
    # As generated by platforms.gce.get_cpuinfo():
    ur' ((:?[A-Z][a-z]+ )+GCE)',
  ]
  for r in regexps:
    m = re.search(r, name)
    if m:
      return m.group(1)


@tools.cached
def get_cpu_dimensions():
  """Returns the values that should be used as 'cpu' dimensions."""
  cpu_type = get_cpu_type()
  bitness = get_cpu_bitness()
  info = get_cpuinfo()
  out = [
    cpu_type,
    u'%s-%s' % (cpu_type, bitness)
  ]
  if 'avx2' in info.get(u'flags', []):
    out.append(u'%s-%s-%s' % (cpu_type, bitness, 'avx2'))

  vendor = info.get(u'vendor') or u''
  name = info.get(u'name') or u''
  if u'GenuineIntel' == vendor:
    model = _parse_intel_model(name)
    if model:
      out.append(u'%s-%s-%s' % (cpu_type, bitness, model.replace(' ', '_')))
  elif cpu_type.startswith(u'arm'):
    if name:
      out.append(u'%s-%s-%s' % (cpu_type, bitness, name.replace(' ', '_')))
    if cpu_type != u'arm':
      out.append(u'arm')
      out.append(u'arm-' + bitness)
  elif cpu_type.startswith(u'mips'):
    if name:
      out.append(u'%s-%s-%s' % (cpu_type, bitness, name.replace(' ', '_')))
  # else AMD like "AMD PRO A6-8500B R5, 6 Compute Cores 2C+4G     "

  out.sort()
  return out


@tools.cached
def get_cpuinfo():
  """Returns the flags of the processor."""
  if sys.platform == 'darwin':
    info = platforms.osx.get_cpuinfo()
  elif sys.platform == 'win32':
    info = platforms.win.get_cpuinfo()
  elif sys.platform == 'linux2':
     info = platforms.linux.get_cpuinfo()
  else:
    info = {}
  if platforms.is_gce():
    # On GCE, the OS reports a generic CPU. Replace with GCE-specific details,
    # keeping the CPU flags as reported by the OS.
    info.update(platforms.gce.get_cpuinfo() or {})
  return info


def get_ip():
  """Returns the IP that is the most likely to be used for TCP connections."""
  # Tries for ~0.5s then give up.
  max_tries = 10
  for i in xrange(10):
    # It's guesswork and could return the wrong IP. In particular a host can
    # have multiple IPs.
    s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    # This doesn't actually connect to the Google DNS server but this forces the
    # network system to figure out an IP interface to use.
    try:
      s.connect(('8.8.8.8', 80))
      return s.getsockname()[0].decode('utf-8')
    except socket.error:
      # Can raise "error: [Errno 10051] A socket operation was attempted to an
      # unreachable network" if the network is still booting up. We don't want
      # this function to crash.
      if i == max_tries - 1:
        # Can't determine the IP.
        return u'0.0.0.0'
      time.sleep(0.05)
    finally:
      s.close()


@tools.cached
def get_hostname():
  """Returns the machine's hostname."""
  if platforms.is_gce() and not os.path.isfile('/.dockerenv'):
    # When running on GCE, always use the hostname as defined by GCE. It's
    # possible the VM hadn't learned about it yet. We ignore GCE hostname when
    # running inside a Docker container and instead use its own hostname.
    meta = platforms.gce.get_metadata() or {}
    hostname = meta.get('instance', {}).get('hostname')
    if hostname:
      return unicode(hostname)

  # Windows enjoys putting random case in there. Enforces lower case for sanity.
  hostname = socket.getfqdn().lower()
  if hostname.endswith('.in-addr.arpa'):
    # When OSX fails to get the FDQN, it returns as the base name the IPv4
    # address reversed, which is not useful. Get the base hostname as defined by
    # the host itself instead of the FQDN since the returned FQDN is useless.
    hostname = socket.gethostname()
  return unicode(hostname)


@tools.cached
def get_hostname_short():
  """Returns the base host name."""
  return get_hostname().split(u'.', 1)[0]


@tools.cached
def get_num_processors():
  """Returns the number of processors.

  Python on OSX 10.6 raises a NotImplementedError exception.
  """
  try:
    if sys.platform == 'linux2':
      return platforms.linux.get_num_processors()
    # Multiprocessing
    return multiprocessing.cpu_count()
  except:  # pylint: disable=W0702
    try:
      # Mac OS 10.6
      return int(os.sysconf('SC_NPROCESSORS_ONLN'))  # pylint: disable=E1101
    except:
      # Returns non-zero, otherwise it could generate a divide by zero later
      # when doing calculations, leading to a crash. Saw it happens on Win2K8R2
      # on python 2.7.5 on cygwin 1.7.28.
      logging.error('get_num_processors() failed to query number of cores')
      # Return an improbable number to make it easier to catch.
      return 5


@tools.cached
def get_physical_ram():
  """Returns the amount of installed RAM in Mb, rounded to the nearest number.
  """
  if sys.platform == 'win32':
    return platforms.win.get_physical_ram()
  if sys.platform == 'darwin':
    return platforms.osx.get_physical_ram()
  if os.path.isfile('/proc/meminfo'):
    # linux.
    meminfo = _safe_read('/proc/meminfo') or ''
    matched = re.search(r'MemTotal:\s+(\d+) kB', meminfo)
    if matched:
      mb = int(matched.groups()[0]) / 1024.
      if 0. < mb < 1.:
        return 1
      return int(round(mb))

  logging.error('get_physical_ram() failed to query amount of physical RAM')
  return 0


def get_disks_info():
  """Returns a dict of dict of free and total disk space."""
  if sys.platform == 'win32':
    return platforms.win.get_disks_info()
  else:
    return platforms.posix.get_disks_info()


@tools.cached
def get_disk_size(path):
  """Returns the partition size that is referenced by this path in Mb."""
  # Find the disk for the path.
  path = os.path.realpath(path)
  paths = sorted(
      ((p, k[u'size_mb']) for p, k in get_disks_info().iteritems()),
      key=lambda x: -len(x[0]))
  # It'd be nice if it were possible to know on a per-path basis, e.g. you can
  # have both case sensitive and insensitive partitions mounted on OSX.
  case_insensitive = sys.platform in ('darwin', 'win32')
  if case_insensitive:
    path = path.lower()
  for base, size_mb in paths:
    if path.startswith(base.lower() if case_insensitive else base):
      return size_mb
  # We have no idea.
  return 0.


@tools.cached
def get_audio():
  """Returns the active audio card(s)."""
  # There's a risk that an audio card may "appear", which may be especially true
  # on OSX when an audio cable is plugged in.
  if sys.platform == 'darwin':
    return platforms.osx.get_audio()
  elif sys.platform == 'linux2':
    return platforms.linux.get_audio()
  elif sys.platform == 'win32':
    return platforms.win.get_audio()
  return None


def get_gpu():
  """Returns the installed video card(s) name.

  Not cached as the GPU driver may change underneat.

  Returns:
    All the video cards detected.
    tuple(list(dimensions), list(state)).
  """
  if sys.platform == 'darwin':
    dimensions, state = platforms.osx.get_gpu()
  elif sys.platform == 'linux2':
    dimensions, state = platforms.linux.get_gpu()
  elif sys.platform == 'win32':
    dimensions, state = platforms.win.get_gpu()
  else:
    dimensions, state = None, None

  # 15ad is VMWare. It's akin not having a GPU card so replace it with the
  # string 'none'.
  if not dimensions or '15ad' in dimensions:
    dimensions = [u'none']
  dimensions.sort()
  return dimensions, state


@tools.cached
def get_monitor_hidpi():
  """Returns True if there is an hidpi monitor detected."""
  if sys.platform == 'darwin':
    return [platforms.osx.get_monitor_hidpi()]
  return None


@tools.cached
def get_cost_hour():
  """Returns the cost in $USD/h as a floating point value if applicable."""
  # Machine.
  machine_type = get_machine_type()
  if platforms.is_gce():
    if platforms.gce.get_zone().startswith('us-'):
      machine_cost = GCE_MACHINE_COST_HOUR_US.get(machine_type, 0.)
    else:
      machine_cost = GCE_MACHINE_COST_HOUR_EUROPE_ASIA.get(machine_type, 0.)
  else:
    # Guess an equivalent machine_type.
    machine_cost = GCE_MACHINE_COST_HOUR_US.get(machine_type, 0.)

  # OS.
  os_cost = 0.
  if sys.platform == 'darwin':
    # Apple tax. It's 50% better, right?
    os_cost = GCE_WINDOWS_COST_CORE_HOUR * 1.5 * get_num_processors()
  elif sys.platform == 'win32':
    # MS tax.
    if machine_type in ('f1-micro', 'g1-small'):
      os_cost = 0.02
    else:
      os_cost = GCE_WINDOWS_COST_CORE_HOUR * get_num_processors()

  # Disk.
  # TODO(maruel): Figure out the disk type. The metadata is not useful AFAIK.
  # Assume HDD for now, it's the cheapest. That's not true, we do have SSDs.
  disk_gb_cost = 0.
  for disk in get_disks_info().itervalues():
    disk_gb_cost += disk[u'free_mb'] / 1024. * (
        GCE_HDD_GB_COST_MONTH / 30. / 24.)

  # TODO(maruel): Network. It's not a constant cost, it's per task.
  # See https://cloud.google.com/monitoring/api/metrics
  # compute.googleapis.com/instance/network/sent_bytes_count
  return machine_cost + os_cost + disk_gb_cost


@tools.cached
def get_machine_type():
  """Returns a GCE-equivalent machine type.

  If running on GCE, returns the right machine type. Otherwise tries to find the
  'closest' one.
  """
  if platforms.is_gce():
    return platforms.gce.get_machine_type()

  ram_gb = get_physical_ram() / 1024.
  cores = get_num_processors()
  ram_gb_per_core = ram_gb / cores
  logging.info('RAM GB/core = %.3f', ram_gb_per_core)
  best_fit = None
  for ratio, prefix in GCE_RAM_GB_PER_CORE_RATIOS.iteritems():
    delta = (ram_gb_per_core-ratio)**2
    if best_fit is None or delta < best_fit[0]:
      best_fit = (delta, prefix)
  prefix = best_fit[1]
  machine_type = prefix + unicode(cores)
  if machine_type not in GCE_MACHINE_COST_HOUR_US:
    # Try a best fit.
    logging.info('Failed to find a good machine_type match: %s', machine_type)
    for i in (16, 8, 4, 2):
      if cores > i:
        machine_type = prefix + unicode(i)
        break
    else:
      if cores == 1:
        # There's no n1-highcpu-1 nor n1-highmem-1.
        if ram_gb < 1.7:
          machine_type = u'f1-micro'
        elif ram_gb < 3.75:
          machine_type = u'g1-small'
        else:
          machine_type = u'n1-standard-1'
      else:
        logging.info('Failed to find a fit: %s', machine_type)

  if machine_type not in GCE_MACHINE_COST_HOUR_US:
    return None
  return machine_type


@tools.cached
def get_locale():
  """Returns the OS's UI active locale."""
  locales = locale.getdefaultlocale()
  if locales[0]:
    return u'.'.join(locales)


def get_uptime():
  """Returns uptime as a float in seconds.

  May or may not include sleep time.
  """
  if sys.platform == 'darwin':
    return platforms.osx.get_uptime()
  if sys.platform == 'win32':
    return platforms.win.get_uptime()
  if sys.platform == 'cygwin':
    # Not important.
    return 0.
  return platforms.linux.get_uptime()


def get_reboot_required():
  """Returns True if the system should be rebooted to apply updates.

  This is not guaranteed to notice all conditions that could require reboot.
  """
  if sys.platform == 'darwin':
    # There doesn't seem to be a good way to do this for OSX.
    return False
  if sys.platform == 'win32' or sys.platform == 'cygwin':
    return platforms.win.get_reboot_required()
  return platforms.linux.get_reboot_required()


def get_ssd():
  """Returns a list of SSD disks."""
  if sys.platform == 'darwin':
    return platforms.osx.get_ssd()
  if sys.platform == 'linux2':
    return platforms.linux.get_ssd()
  return ()


def get_cipd_cache_info():
  """Returns the items in cipd cache."""
  # Strictly speaking, this is a layering violation. This data is managed by
  # cipd.py but this is valuable to expose this as a Swarming bot state so
  # ¯\_(ツ)_/¯
  #
  # Assumptions:
  # - ../__main__.py calls os.chdir(__file__)
  # - ../client/cipd.py behavior
  # - cache entries are in cipd_cache/cache/instances
  try:
    items = 0
    total = 0
    root = os.path.join(u'cipd_cache', u'cache', u'instances')
    for i in os.listdir(root):
      if i != u'state.db':
        items += 1
        total += os.stat(os.path.join(root, i)).st_size
    return {u'items': items, u'size': total}
  except (IOError, OSError):
    return {}
  return 0


def get_isolated_cache_info():
  """Returns the items in state.json describing isolated caches."""
  # Strictly speaking, this is a layering violation. This data is managed by
  # run_isolated.py but this is valuable to expose this as a Swarming bot
  # state so ¯\_(ツ)_/¯
  #
  # Assumptions:
  # - ../__main__.py calls os.chdir(__file__)
  # - ../bot_code/bot_main.py specifies
  #   --cache os.path.join(botobj.base_dir, 'isolated_cache') to run_isolated.
  # - state.json is lru.LRUDict format.
  # - ../client/isolateserver.py behavior
  try:
    with open(os.path.join(u'isolated_cache', u'state.json'), 'rb') as f:
      return dict(json.load(f)['items'])
  except (IOError, KeyError, OSError, TypeError, ValueError):
    return {}


def get_named_caches_info():
  """"Returns the items in state.json describing named caches."""
  # Strictly speaking, this is a layering violation. This data is managed by
  # run_isolated.py but this is valuable to expose this as a Swarming bot
  # dimension and state so ¯\_(ツ)_/¯
  #
  # Assumptions:
  # - ../__main__.py calls os.chdir(__file__)
  # - ../bot_code/bot_main.py specifies
  #   --named-cache-root os.path.join(botobj.base_dir, 'c') to run_isolated.
  # - state.json is lru.LRUDict format.
  # - ../client/named_cache.py behavior
  #
  # A better implementation would require:
  # - Access to bot.Bot instance to query bot.base_dir
  # - Access to named_cache.py to load state.json
  # - Access to --named-cache-root hardcoded in bot_main.py
  #
  # but hey, the following code is 5 lines...
  try:
    with open(os.path.join(u'c', u'state.json'), 'rb') as f:
      return dict(json.load(f)['items'])
  except (IOError, KeyError, OSError, TypeError, ValueError):
    return {}


@tools.cached
def get_python_packages():
  """Returns the list of third party python packages."""
  try:
    # --disable-pip-version-check is only supported in v6.0 and we still have
    # bots running very old versions. Use the environment variable instead.
    env = os.environ.copy()
    env['PIP_DISABLE_PIP_VERSION_CHECK'] = '1'
    cmd = ['pip', 'freeze']
    return unicode(subprocess.check_output(cmd, env=env)).splitlines()
  except (subprocess.CalledProcessError, OSError):
    return None


class AuthenticatedHttpRequestFailure(Exception):
  pass


def authenticated_http_request(service_account, *args, **kwargs):
  """Sends an OAuth2-authenticated HTTP request.

  Args:
    service_account: Service account to use. For GCE, the name of the service
      account, otherwise the path to the service account JSON file.

  Raises:
    AuthenticatedHttpRequestFailure
  """
  scopes = kwargs.pop('scopes', [])
  kwargs['headers'] = kwargs.get('headers', {}).copy()
  http = httplib2.Http(ca_certs=tools.get_cacerts_bundle())

  # Authorize the request. In general, we need to produce an OAuth2 bearer token
  # using a service account JSON file. However on GCE there is a shortcut: it
  # can fetch the current bearer token right from the instance metadata without
  # the need for the oauth2client.client library.
  if platforms.is_gce():
    try:
      gce_bearer_token, _ = platforms.gce.oauth2_access_token_with_expiration(
          account=service_account)
    except (IOError, urllib2.HTTPError) as e:
      raise AuthenticatedHttpRequestFailure(e)
    kwargs['headers']['Authorization'] = 'Bearer %s' % gce_bearer_token
  else:
    try:
      oauth2client = get_oauth2_client(service_account, scopes)
    except (IOError, OSError, ValueError) as e:
      raise AuthenticatedHttpRequestFailure(e)
    http = oauth2client.authorize(http)

  try:
    return http.request(*args, **kwargs)
  except client.Error as e:
    raise AuthenticatedHttpRequestFailure(e)


class GetTimeseriesDataFailure(Exception):
  pass


def get_timeseries_data(name, project, service_account, **kwargs):
  """Gets timeseries data from Cloud Monitoring.

  Args:
    name: Name of the custom metric. Must already exist.
    project: Project the metric exists in. Must already exist.
    service_account: Service account to use. For GCE, the name of the
      service account, otherwise the path to the service account JSON file.

  **kwargs:
    count: If specified, the maximum number of data points per page.
    labels: If specified, a list of labels to filter the returned timeseries
      data. In general, each label should be of the form "key<operator>value",
      where <operator> is one of ==, !=, =~, !~ meaning (respectively) that the
      key matches, does not match, regex matches, and does not regex match the
      value.
    oldest: If specified, the RFC 3339 timestamp string of the earliest time to
      exclude timeseries data for. No timeseries data timestamped with this
      timestamp or earlier will be returned. If unspecified, adheres to the
      Cloud Monitoring default.
    page_token: If specified, the token used to access a specific page of
      results.
    youngest: If specified, the RFC 3339 timestamp string of the latest time to
      include timeseries data for. No timeseries data timestamped after this
      time will be returned. If unspecified, defaults to the current time.

  Returns:
    A 2-tuple (list of matching timeseries data dicts, next page token).

  Raises:
    GetTimeseriesDataFailure
  """
  params = {
      'youngest': kwargs.get('youngest') or time.strftime(
          '%Y-%m-%dT%H:%M:%SZ', time.gmtime())
  }
  if kwargs.get('count'):
    params['count'] = kwargs['count']
  if kwargs.get('labels'):
    params['labels'] = kwargs['labels']
  if kwargs.get('oldest'):
    params['oldest'] = kwargs['oldest']
  if kwargs.get('page_token'):
    params['pageToken'] = kwargs['page_token']

  metric = urllib.quote('custom.cloudmonitoring.googleapis.com/%s' % name, '')
  params = urllib.urlencode(params, True)
  url = '%s/projects/%s/timeseries/%s?%s' % (
      MONITORING_ENDPOINT, project, metric, params)
  logging.info('Attempting to get timeseries data: %s', url)
  try:
    response, content = authenticated_http_request(
        service_account, url, method='GET', scopes=MONITORING_SCOPES)
  except (AuthenticatedHttpRequestFailure, IOError) as e:
    raise GetTimeseriesDataFailure(e)

  if response['status'] != '200':
    try:
      content = json.loads(content)
    except ValueError:
      pass
    raise GetTimeseriesDataFailure(content)

  content = json.loads(content)
  return content.get('timeseries', []), content.get('nextPageToken')


class SendMetricsFailure(Exception):
  pass


def send_metric(name, value, labels, project, service_account):
  """Send a custom metric value to Cloud Monitoring.

  Args:
    name: Name of the custom metric. Must already exist.
    value: Value to send. Must be int or float.
    labels: Labels to include with the metric. Must be a dict.
    project: Project the metric exists in. Must already exist.
    service_account: Service account to use. For GCE, the name of the
      service account, otherwise the path to the service account JSON file.

  Raises:
    SendMetricsFailure
  """
  headers = {
      'Content-Type': 'application/json',
  }

  now = time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
  body = {
      'commonLabels': labels,
      'timeseries': [{
          'point': {
              'end': now,
              'start': now,
          },
          'timeseriesDesc': {
              'metric': 'custom.cloudmonitoring.googleapis.com/%s' % name,
              'project': project,
          },
      }],
  }
  # Cloud Monitoring only supports int64 and double for custom metrics.
  if isinstance(value, int):
    body['timeseries'][0]['point']['int64Value'] = value
  elif isinstance(value, float):
    body['timeseries'][0]['point']['doubleValue'] = value
  else:
    raise SendMetricsFailure('Invalid value type: %s' % type(value))

  try:
    response, content = authenticated_http_request(
        service_account,
        '%s/projects/%s/timeseries:write' % (MONITORING_ENDPOINT, project),
        method='POST', body=json.dumps(body), headers=headers,
        scopes=MONITORING_SCOPES)
  except (AuthenticatedHttpRequestFailure, IOError) as e:
    raise SendMetricsFailure(e)

  if response['status'] != '200':
    try:
      content = json.loads(content)
    except ValueError:
      pass
    raise SendMetricsFailure(content)


def get_oauth2_client(service_account_json_file, scopes=None):
  if not scopes:
    scopes = []
  # Ensure scopes is a hashable type for caching.
  return _get_oauth2_client(service_account_json_file, tuple(sorted(scopes)))


@tools.cached
def _get_oauth2_client(service_account_json_file, scopes):
  with open(service_account_json_file) as f:
    service_account_json = json.load(f)
  return client.SignedJwtAssertionCredentials(
      service_account_json['client_email'], service_account_json['private_key'],
      scopes)


### Android.


def get_dimensions_all_devices_android(devices):
  """Returns the default dimensions for an host with multiple android devices.
  """
  dimensions = get_dimensions()
  if not devices:
    return dimensions

  # Pop a few dimensions otherwise there will be too many dimensions.
  del dimensions[u'cpu']
  del dimensions[u'cores']
  del dimensions[u'gpu']
  dimensions.pop(u'kvm', None)
  dimensions.pop(u'machine_type')
  dimensions.pop(u'ssd', None)

  dimensions.update(platforms.android.get_dimensions(devices))
  return dimensions


def get_state_all_devices_android(devices):
  """Returns state information about all the devices connected to the host.
  """
  state = get_state()
  if not devices:
    return state

  # Add a few values that were popped from dimensions.
  state[u'host_dimensions'] = get_dimensions()
  # The default value is irrelevant.
  state[u'host_dimensions'].pop(u'pool', None)
  state.update(platforms.android.get_state(devices))
  return state


###


def get_dimensions():
  """Returns the default dimensions."""
  dimensions = {
    u'cores': [unicode(get_num_processors())],
    u'cpu': get_cpu_dimensions(),
    u'gpu': get_gpu()[0],
    u'id': [get_hostname_short()],
    u'os': get_os_values(),
    # This value is frequently overridden by bots.cfg via luci-config.
    u'pool': [u'default'],
    u'python': [unicode(sys.version).split()[0]],
  }

  # Conditional dimensions:
  id_override = os.environ.get('SWARMING_BOT_ID')
  if id_override:
    dimensions[u'id'] = [unicode(id_override)]

  caches = get_named_caches_info()
  if caches:
    dimensions[u'caches'] = sorted(caches)

  if u'none' not in dimensions[u'gpu']:
    hidpi = get_monitor_hidpi()
    if hidpi:
      dimensions[u'hidpi'] = hidpi

  machine_type = get_machine_type()
  if machine_type:
    dimensions[u'machine_type'] = [machine_type]

  if platforms.is_gce():
    dimensions[u'gce'] = [u'1']
    image = platforms.gce.get_image()
    if image:
      dimensions[u'image'] = [image]
    dimensions[u'zone'] = platforms.gce.get_zones()
  else:
    dimensions[u'gce'] = [u'0']

  loc = get_locale()
  if loc:
    dimensions[u'locale'] = [loc]

  ssd = get_ssd()
  if ssd:
    dimensions[u'ssd'] = [u'1']

  if sys.platform == 'linux2':
    inside_docker = platforms.linux.get_inside_docker()
    if not inside_docker:
      dimensions[u'inside_docker'] = [u'0']
    else:
      dimensions[u'inside_docker'] = [u'1', inside_docker]

    dimensions[u'kvm'] = [unicode(int(platforms.linux.get_kvm()))]

    comp = platforms.linux.get_device_tree_compatible()
    if comp:
      dimensions[u'device_tree_compatible'] = comp
    # Just check CPU #0. In practice different CPU core could have different CPU
    # governor.
    gov = platforms.linux.get_cpu_scaling_governor(0)
    if gov:
      dimensions[u'cpu_governor'] = gov

  if sys.platform == 'darwin':
    model = platforms.osx.get_hardware_model_string()
    if model:
      dimensions[u'mac_model'] = [model]
    xcode_versions = platforms.osx.get_xcode_versions()
    if xcode_versions:
      dimensions[u'xcode_version'] = xcode_versions

    # iOS devices
    udids = platforms.osx.get_ios_device_ids()
    device_types = set()
    for udid in udids:
      version = platforms.osx.get_ios_version(udid)
      if version:
        dimensions[u'os'].append('iOS-%s' % version)
        dimensions[u'os'].sort()
      device_type = platforms.osx.get_ios_device_type(udid)
      if device_type:
        device_types.add(device_type)
    if device_types:
      dimensions[u'device'] = sorted(device_types)

  if sys.platform == 'win32':
    integrity = platforms.win.get_integrity_level()
    if integrity is not None:
      dimensions[u'integrity'] = [integrity]

  return dimensions


def get_state():
  """Returns dict with a state of the bot reported to the server with each poll.

  Supposed to be use only for dynamic state that changes while bot is running.

  The server can not use this state for immediate scheduling purposes (use
  'dimensions' for that), but it can use it for maintenance and bookkeeping
  tasks.
  """
  tmpdir = tempfile.gettempdir()
  try:
    nb_files_in_temp = len(os.listdir(tmpdir))
  except OSError:
    nb_files_in_temp = 'N/A'
  state = {
    u'audio': get_audio(),
    u'cpu_name': get_cpuinfo().get(u'name'),
    u'cost_usd_hour': get_cost_hour(),
    u'cwd': file_path.get_native_path_case(os.getcwd().decode('utf-8')),
    u'disks': get_disks_info(),
    # Only including a subset of the environment variable, as state is not
    # designed to sustain large load at the moment.
    u'env': {
      u'PATH': os.environ[u'PATH'],
    },
    u'gpu': get_gpu()[1],
    u'hostname': get_hostname(),
    u'ip': get_ip(),
    u'nb_files_in_temp': nb_files_in_temp,
    u'pid': os.getpid(),
    u'python': {
      u'executable': unicode(sys.executable),
      u'packages': get_python_packages(),
      u'version': unicode(sys.version),
    },
    u'ram': get_physical_ram(),
    u'running_time': int(round(time.time() - _STARTED_TS)),
    u'ssd': list(get_ssd()),
    u'started_ts': int(round(_STARTED_TS)),
    u'uptime': int(round(get_uptime())),
    u'user': getpass.getuser().decode('utf-8'),
  }
  if get_reboot_required():
    state[u'reboot_required'] = True
  cache = get_named_caches_info()
  if cache:
    state[u'named_caches'] = cache
  if sys.platform in ('cygwin', 'win32'):
    state[u'cygwin'] = [sys.platform == 'cygwin']
  if sys.platform == 'darwin':
    state[u'xcode'] = platforms.osx.get_xcode_state()
    temp = platforms.osx.get_temperatures()
    if temp is not None:
      state[u'temp'] = temp
  if sys.platform == 'linux2':
    temp = platforms.linux.get_temperatures()
    if temp:
      state[u'temp'] = temp

    docker_host_hostname = os.environ.get('DOCKER_HOST_HOSTNAME')
    if docker_host_hostname:
      state[u'docker_host_hostname'] = unicode(docker_host_hostname)

  # Put an arbitrary limit on the amount of junk that can stay in TEMP.
  if nb_files_in_temp == 'N/A':
    state[u'quarantined'] = 'Failed to access TEMP (%s)' % tmpdir
  elif nb_files_in_temp > 1024:
    state[u'quarantined'] = '> 1024 files in TEMP (%s)' % tmpdir
  return state


## State mutating.


def rmtree(path):
  """Removes a directory the bold way."""
  file_path.rmtree(path)


def setup_auto_startup_win(command, cwd, batch_name):
  """Uses Startup folder in the Start Menu.

  This assumes the user is automatically logged in on OS startup.

  Works both inside cygwin's python or native python which makes this function a
  bit more tricky than necessary.

  Use the start up menu instead of registry for two reasons:
  - It's easy to remove in case of failure, for example in case of reboot loop.
  - It works well even with cygwin.

  TODO(maruel): This function assumes |command| is python script to be run.
  """
  logging.info('setup_auto_startup_win(%s, %s, %s)', command, cwd, batch_name)
  if not os.path.isabs(cwd):
    raise ValueError('Refusing relative path')
  assert batch_name.endswith('.bat'), batch_name
  batch_path = platforms.win.get_startup_dir() + batch_name

  # If we are running through cygwin, the path to write to must be changed to be
  # in the cywgin format, but we also need to change the commands to be in
  # non-cygwin format (since they will execute in a batch file).
  if sys.platform == 'cygwin':
    batch_path = platforms.win.to_cygwin_path(batch_path)
    assert batch_path
    cwd = platforms.win.from_cygwin_path(cwd)
    assert cwd

    # Convert all the cygwin paths in the command.
    for i in range(len(command)):
      if '/cygdrive/' in command[i]:
        command[i] = platforms.win.from_cygwin_path(command[i])

  # Don't forget the CRLF, otherwise cmd.exe won't process it.
  #
  # Do manual roll at each system startup because on Windows, cmd.exe opens
  # redirected file with no sharing permission at all (grrr) so log roll cannot
  # be done from within the swarming_bot process. This manual roll means we only
  # keep the last 10 boots instead of the last 10Mb of logs, the difference is
  # significant but 10 boots should be good enough in general.
  #
  # pipes.quote() shell escape is sadly escaping with single quotes instead of
  # double quotes, which isn't always great on Windows. Sadly shlex.quote() is
  # only available starting python 3.3 and it's tricky on Windows with '^'. So
  # skip this for now and hope for the best.
  content = (
      '@echo off\r\n'
      ':: This file was generated automatically by os_platforms.py.\r\n'
      'setlocal enableextensions enabledelayedexpansion\r\n'
      'cd /d %(root)s\r\n'
      '\r\n'
      'if not exist logs mkdir logs\r\n'
      'if exist logs\\bot_stdout.log.9 del logs\\bot_stdout.log.9\r\n'
      'for %%%%i in (8 7 6 5 4 3 2 1) do (\r\n'
      '  if exist logs\\bot_stdout.log.%%%%i (\r\n'
      '    set /a "j=%%%%i+1"\r\n'
      '    echo move logs\\bot_stdout.log.%%%%i logs\\bot_stdout.log.!j!\r\n'
      '    move logs\\bot_stdout.log.%%%%i logs\\bot_stdout.log.!j!\r\n'
      '    set j=\r\n'
      '  )\r\n'
      ')\r\n'
      'if exist logs\\bot_stdout.log (\r\n'
      '  echo move logs\\bot_stdout.log logs\\bot_stdout.log.1\r\n'
      '  move logs\\bot_stdout.log logs\\bot_stdout.log.1\r\n'
      ')\r\n'
      '\r\n'
      'echo Running: %(command)s\r\n'
      '%(command)s 1>> logs\\bot_stdout.log 2>&1\r\n') % {
        'root': cwd,
        'command': ' '.join(command)
      }
  success = _write(batch_path, content)
  if success and sys.platform == 'cygwin':
    # For some reason, cygwin tends to create the file with 0644.
    os.chmod(batch_path, 0755)
  return success


def setup_auto_startup_osx(command, cwd, plistname):
  """Uses launchd to start the command when the user logs in.

  This assumes the user is automatically logged in on OS startup.

  In case of failure like reboot loop, simply remove the file in
  ~/Library/LaunchAgents/.
  """
  logging.info('setup_auto_startup_osx(%s, %s, %s)', command, cwd, plistname)
  if not os.path.isabs(cwd):
    raise ValueError('Refusing relative path')
  assert plistname.endswith('.plist'), plistname
  launchd_dir = os.path.expanduser('~/Library/LaunchAgents')
  if not os.path.isdir(launchd_dir):
    # This directory doesn't exist by default.
    # Sometimes ~/Library gets deleted.
    os.makedirs(launchd_dir)
  filepath = os.path.join(launchd_dir, plistname)
  return _write(
      filepath, platforms.osx.generate_launchd_plist(command, cwd, plistname))


def setup_auto_startup_initd_linux(command, cwd, user=None, name='swarming'):
  """Uses init.d to start the bot automatically."""
  if not user:
    user = getpass.getuser()
  logging.info(
      'setup_auto_startup_initd_linux(%s, %s, %s, %s)',
      command, cwd, user, name)
  if not os.path.isabs(cwd):
    raise ValueError('Refusing relative path')
  script = platforms.linux.generate_initd(command, cwd, user)
  filepath = pipes.quote(os.path.join('/etc/init.d', name))
  with tempfile.NamedTemporaryFile() as f:
    if not _write(f.name, script):
      return False

    # Need to do 3 things as sudo. Do it all at once to enable a single sudo
    # request.
    # TODO(maruel): Likely not the sanest thing, reevaluate.
    cmd = [
      'sudo', '/bin/sh', '-c',
      "cp %s %s && chmod 0755 %s && update-rc.d %s defaults" % (
        pipes.quote(f.name), filepath, filepath, name)
    ]
    subprocess.check_call(cmd)
    print('To remove, use:')
    print('  sudo update-rc.d -f %s remove' % name)
    print('  sudo rm %s' % filepath)
  return True


def setup_auto_startup_autostart_desktop_linux(command, name='swarming'):
  """Uses ~/.config/autostart to start automatically the bot on user login.

  http://standards.freedesktop.org/autostart-spec/autostart-spec-latest.html
  """
  basedir = os.path.expanduser('~/.config/autostart')
  if not os.path.isdir(basedir):
    os.makedirs(basedir)
  filepath = os.path.join(basedir, '%s.desktop' % name)
  return _write(
      filepath, platforms.linux.generate_autostart_desktop(command, name))


def host_reboot(message=None, timeout=None):
  """Reboots this machine.

  If it fails to reboot the host, it loops until timeout. This function does
  not return on successful host reboot, or returns False if machine wasn't
  restarted within |timeout| seconds.
  """
  # The shutdown process sends SIGTERM and waits for processes to exit. It's
  # important to not handle SIGTERM and die when needed.
  # TODO(maruel): We may want to die properly here.
  signal.signal(signal.SIGTERM, signal.SIG_DFL)

  deadline = time.time() + timeout if timeout else None
  while True:
    host_reboot_and_return(message)
    # Sleep for 300 seconds to ensure we don't try to do anymore work while the
    # OS is preparing to shutdown.
    loop = True
    while loop:
      duration = min(300, deadline - time.time()) if timeout else 300
      if duration <= 0:
        break
      logging.info('Sleeping for %s', duration)
      try:
        time.sleep(duration)
        loop = False
      except IOError as e:
        # Ignore "[Errno 4] Interrupted function call"; we do not want the
        # process to die, so let's sleep again until the OS forcibly kill the
        # process.
        logging.info('Interrupted sleep: %s', e)
    if timeout and time.time() >= deadline:
      logging.warning(
          'Waited for host to reboot for too long (%s); aborting', timeout)
      return False


def host_reboot_and_return(message=None):
  """Tries to reboot this host and immediately return to the caller.

  This is mostly useful when done via remote shell, like via ssh, where it is
  not worth waiting for the TCP connection to tear down.

  Returns:
    True if at least one command succeeded.
  """
  if sys.platform == 'win32':
    cmds = [
      ['shutdown', '-r', '-f', '-t', '1'],
    ]
  elif sys.platform == 'cygwin':
    # The one that will succeed depends if it is executed via a prompt or via
    # a ssh command. #itscomplicated.
    cmds = [
      ['shutdown', '-r', '-f', '-t', '1'],
      ['shutdown', '-r', '-f', '1'],
    ]
  elif sys.platform == 'linux2':
    # systemd removed support for -f. So Ubuntu 14.04 supports -f but 16.04
    # won't. This is also the case of Raspbian Jessie, which is on systemd. For
    # now, just try both. Once pre-systemd system are not supported anymore,
    # remove the call with -f.
    cmds = [
      ['sudo', '-n', '/sbin/shutdown', '-f', '-r', 'now'],
      ['sudo', '-n', '/sbin/shutdown', '-r', 'now'],
    ]
  elif sys.platform == 'darwin':
    # -f is supported on linux but not OSX.
    cmds = [['sudo', '-n', '/sbin/shutdown', '-r', 'now']]
  else:
    cmds = [['sudo', '-n', 'shutdown', '-r', 'now']]

  success = False
  for cmd in cmds:
    logging.info(
        'Restarting machine with command %s (%s)', ' '.join(cmd), message)
    try:
      subprocess.check_call(cmd)
      logging.info('Restart command exited successfully')
    except (OSError, subprocess.CalledProcessError) as e:
      logging.error('Failed to run %s: %s', ' '.join(cmd), e)
    else:
      success = True
  return success


def roll_log(name):
  """Rolls a log in 5Mb chunks and keep the last 10 files."""
  try:
    if not os.path.isfile(name) or os.stat(name).st_size < 5*1024*1024:
      return
    if os.path.isfile('%s.9' % name):
      os.remove('%s.9' % name)
    for i in xrange(8, 0, -1):
      item = '%s.%d' % (name, i)
      if os.path.isfile(item):
        os.rename(item, '%s.%d' % (name, i+1))
    if os.path.isfile(name):
      os.rename(name, '%s.1' % name)
  except Exception as e:
    logging.exception('roll_log(%s) failed: %s', name, e)


def trim_rolled_log(name):
  try:
    for item in glob.iglob('%s.??' % name):
      os.remove(item)
    for item in glob.iglob('%s.???' % name):
      os.remove(item)
  except Exception as e:
    logging.exception('trim_rolled_log(%s) failed: %s', name, e)
