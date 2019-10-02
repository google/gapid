# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Swarming bot main process.

This is the program that communicates with the Swarming server, ensures the code
is always up to date and executes a child process to run tasks and upload
results back.

It manages self-update and rebooting the host in case of problems.

Sections are:
  - Globals
  - Monitoring
  - bot_config handler
  - Public functions used by __main__.py
  - Sub process management
  - Bot lifetime management
"""

import argparse
import contextlib
import fnmatch
import json
import logging
import os
import shutil
import sys
import tempfile
import threading
import time
import traceback
import types
import zipfile

# Import _strptime before threaded code. datetime.datetime.strptime is
# threadsafe except for the initial import of the _strptime module.
# See https://bugs.python.org/issue7980.
import _strptime  # pylint: disable=unused-import

import bot_auth
import common
import file_refresher
import remote_client
import remote_client_errors
import singleton
from api import bot
from api import os_utilities
from api import platforms
from infra_libs import ts_mon
from utils import file_path
from utils import fs
from utils import net
from utils import on_error
from utils import subprocess42
from utils import tools
from utils import zip_package


### Globals


# Used to opportunistically set the error handler to notify the server when the
# process exits due to an exception.
_ERROR_HANDLER_WAS_REGISTERED = False


# Set to the zip's name containing this file. This is set to the absolute path
# to swarming_bot.zip when run as part of swarming_bot.zip. This value is
# overriden in unit tests.
#
# Note: this more or less requires the bot to be in a path without non-ASCII
# characters.
THIS_FILE = unicode(os.path.abspath(zip_package.get_main_script_path()))
THIS_DIR = os.path.dirname(THIS_FILE)


# The singleton, initially unset.
SINGLETON = singleton.Singleton(THIS_DIR)


# Whitelist of files that can be present in the bot's directory. Anything else
# will be forcibly deleted on startup! Note that 'w' (work) is not in this list,
# as we want it to be deleted on startup.
# See
# https://chromium.googlesource.com/infra/luci/luci-py.git/+/master/appengine/swarming/doc/Bot.md
# for more details.
PASSLIST = (
  '*-cacert.pem',
  'c',
  'cipd_cache',
  'isolated_cache',
  'logs',
  'README',
  'README.md',
  'swarming.lck',
  'swarming_bot.1.zip',
  'swarming_bot.2.zip',
  'swarming_bot.zip',
)


# These settings are documented in ../config/bot_config.py.
# Keep in sync with ../config/bot_config.py. This is enforced by a unit test.
DEFAULT_SETTINGS = {
  'free_partition': {
    'root': {
      'size': 1 * 1024*1024*1024,
      'max_percent': 10.,
      'min_percent': 6.,
    },
    'bot': {
      'size': 4 * 1024*1024*1024,
      'max_percent': 15.,
      'min_percent': 7.,
      'wiggle': 250 * 1024*1024,
    },
  },
  'caches': {
    'isolated': {
      'size': 50 * 1024*1024*1024,
      'items': 50*1024,
    },
  },
}

# Keep in sync with ../../ts_mon_metrics.py
_IGNORED_DIMENSIONS = (
    'android_devices', 'caches', 'id', 'server_version', 'temp_band')


### Monitoring


_bucketer = ts_mon.GeometricBucketer(growth_factor=10**0.07,
                                     num_finite_buckets=100)

_hooks_durations = ts_mon.CumulativeDistributionMetric(
    'swarming/bots/hooks/durations',
    'Duration of bot hook calls in ms', [
        ts_mon.StringField('hookname'),
        ts_mon.StringField('pool'),
    ],
    bucketer=_bucketer,
    units=ts_mon.MetricsDataUnits.MILLISECONDS)


def _pool_from_dimensions(dimensions):
  """Return a canonical string of flattened dimensions."""
  # Keep in sync with ../../ts_mon_metrics.py
  pairs = []
  for key, values in dimensions.iteritems():
    if key in _IGNORED_DIMENSIONS:
      continue
    # Strip all the prefixes of other values. values is already sorted.
    for i, value in enumerate(values):
      if not any(v.startswith(value) for v in values[i+1:]):
        pairs.append(u'%s:%s' % (key, value))
  return u'|'.join(sorted(pairs))


def _monitor_call(func):
  """Decorates a functions and reports the runtime to ts_mon."""
  def hook(chained, botobj, name, *args, **kwargs):
    start = time.time()
    try:
      return func(chained, botobj, name, *args, **kwargs)
    finally:
      duration = max(0, (time.time() - start) * 1000)
      if botobj and botobj.dimensions:
        flat_dims = _pool_from_dimensions(botobj.dimensions)
        if flat_dims:
          logging.info('ts_mon hook_name=%r pool=%r', name, flat_dims)
          _hooks_durations.add(
              duration, fields={u'hookname': name, u'pool': flat_dims})
      logging.info('%s(): %gs', name, round(duration/1000., 3))
  return hook


def _init_ts_mon():
  """Initializes ts_mon."""
  parser = argparse.ArgumentParser(description=sys.modules[__name__].__doc__)
  ts_mon.add_argparse_options(parser)
  parser.set_defaults(
      ts_mon_target_type='task',
      ts_mon_task_service_name='swarming-bot',
      ts_mon_task_job_name='default',
      ts_mon_flush='auto',
      ts_mon_ca_certs=tools.get_cacerts_bundle(),
  )
  args = parser.parse_args([])
  ts_mon.process_argparse_options(args)


### bot_config handler


# Reference to the config/bot_config.py module inside the swarming_bot.zip file.
# This variable is initialized inside _get_bot_config().
_BOT_CONFIG = None
# Reference to the second bot_config.py module injected by the server. This
# variable is initialized inside _do_handshake().
_EXTRA_BOT_CONFIG = None
# Super Sticky quarantine string. This variable is initialized inside
# _set_quarantined() and be set at various places when a hook throws an
# exception. Restarting the bot will clear the quarantine, which includes
# updated the bot due to new bot_config or new bot code.
_QUARANTINED = None


def _set_quarantined(reason):
  """Sets the Super Sticky Quarantine string."""
  logging.error('_set_quarantined(%s)', reason)
  global _QUARANTINED
  _QUARANTINED = _QUARANTINED or reason


def _get_bot_config():
  """Returns the bot_config.py module. Imports it only once.

  This file is called implicitly by _call_hook() and _call_hook_safe().
  """
  global _BOT_CONFIG
  if not _BOT_CONFIG:
    from config import bot_config as _BOT_CONFIG
  return _BOT_CONFIG


def _register_extra_bot_config(content):
  """Registers the server injected extra injected.py bot_config.

  This file is called implicitly by _call_hook() and _call_hook_safe().
  """
  global _EXTRA_BOT_CONFIG
  if isinstance(content, unicode):
    # compile will throw if there's a '# coding: utf-8' line and the string is
    # in unicode. <3 python.
    content = content.encode('utf-8')
  try:
    compiled = compile(content, 'injected.py', 'exec')
    _EXTRA_BOT_CONFIG = types.ModuleType('injected')
    exec(compiled, _EXTRA_BOT_CONFIG.__dict__)
  except (SyntaxError, TypeError) as e:
    _set_quarantined(
        'handshake returned invalid injected bot_config.py: %s' % e)


@_monitor_call
def _call_hook(chained, botobj, name, *args, **kwargs):
  """Calls a hook function named `name` in bot_config.py.

  If `chained` is True, calls the general bot_config.py then the injected
  version.

  If `chained` is False, the injected bot_config version is called first, and
  only if not present the general bot_config version is called.
  """
  try:
    if not chained:
      # Injected version has higher priority.
      hook = getattr(_EXTRA_BOT_CONFIG, name, None)
      if hook:
        return hook(botobj, *args, **kwargs)
      hook = getattr(_get_bot_config(), name, None)
      if hook:
        return hook(botobj, *args, **kwargs)
      # The hook is not defined.
      return None

    # In the case of chained=True, call both hooks. Call the generic one first,
    # then the specialized.
    ret = None
    hook = getattr(_get_bot_config(), name, None)
    if hook:
      ret = hook(botobj, *args, **kwargs)
    hook = getattr(_EXTRA_BOT_CONFIG, name, None)
    if hook:
      # Ignores the previous return value.
      ret = hook(botobj, *args, **kwargs)
    return ret
  finally:
    # TODO(maruel): Handle host_reboot() request the same way.
    if botobj:
      msg = botobj.bot_restart_msg()
      if msg:
        # The hook requested a bot restart. Do it right after the hook call.
        _bot_restart(botobj, msg)


def _call_hook_safe(chained, botobj, name, *args):
  """Calls a hook function in bot_config.py.

  Like _call_hook() but traps most exceptions.
  """
  try:
    return _call_hook(chained, botobj, name, *args)
  except Exception as e:
    logging.exception('%s() threw', name)
    msg = '%s\n%s' % (e, traceback.format_exc()[-2048:])
    if botobj:
      botobj.post_error('Failed to call hook %s(): %s' % (name, msg))
    # TODO(maruel): Disabled because of https://crbug.com/694327
    #_set_quarantined(msg)


def _get_dimensions(botobj):
  """Returns bot_config.py's get_dimensions() dict."""
  # Importing this administrator provided script could have side-effects on
  # startup. That is why it is imported late.
  out = _call_hook_safe(False, botobj, 'get_dimensions')
  if isinstance(out, dict):
    out = out.copy()
    out[u'server_version'] = [_get_server_version_safe()]
    return out
  try:
    _set_quarantined('get_dimensions(): expected a dict, got %r' % out)
    out = os_utilities.get_dimensions()
    out[u'quarantined'] = [u'1']
    out[u'server_version'] = [_get_server_version_safe()]
    return out
  except Exception as e:
    logging.exception('os.utilities.get_dimensions() failed')
    return {
      u'error': [u'%s\n%s' % (e, traceback.format_exc()[-2048:])],
      u'id': [_get_botid_safe()],
      u'quarantined': [u'1'],
      u'server_version': [_get_server_version_safe()],
    }


def _get_server_version_safe():
  return get_config().get(u'server_version', u'N/A')


@tools.cached
def _get_botid_safe():
  """Paranoid version of get_hostname_short()."""
  try:
    return os_utilities.get_hostname_short()
  except Exception as e:
    logging.exception('os.utilities.get_hostname_short() failed')
    return 'error_%s' % str(e)


def _get_settings(botobj):
  """Returns settings for this bot.

  The way used to make it work safely is to take the default settings, then
  merge the custom settings. This way, a user can only specify a subset of the
  desired settings.

  The function won't alert on unknown settings. This is so bot_config.py can be
  updated in advance before pushing new bot_main.py. The main drawback is that
  it will make typos silently fail. CHECK FOR TYPOS in get_settings() in your
  bot_config.py.
  """
  settings = _call_hook_safe(False, botobj, 'get_settings')
  try:
    if isinstance(settings, dict):
      return _dict_deep_merge(DEFAULT_SETTINGS, settings)
  except (KeyError, TypeError, ValueError):
    logging.exception('get_settings() failed')
  return DEFAULT_SETTINGS


def _get_state(botobj, sleep_streak):
  """Returns dict with a state of the bot reported to the server with each poll.
  """
  state = _call_hook_safe(False, botobj, 'get_state')
  if not isinstance(state, dict):
    _set_quarantined('get_state(): expected a dict, got %r' % state)
    state = {u'broken': state}

  if not state.get(u'quarantined'):
    if not _is_base_dir_ok(botobj):
      # Use super hammer in case of dangerous environment.
      _set_quarantined('Can\'t run from blacklisted directory')
    if _QUARANTINED:
      state[u'quarantined'] = _QUARANTINED

  state[u'sleep_streak'] = sleep_streak
  if not state.get(u'quarantined') and botobj:
    # Reuse the data from 'state/disks'
    disks = state.get(u'disks', {})
    err = _get_disks_quarantine(botobj, disks)
    if err:
      state[u'quarantined'] = err
  return state


def _get_disks_quarantine(botobj, disks):
  """Returns a quarantine error message when there's not enough free space.

  It looks at both root partition and the current partition the bot is running
  in.
  """
  settings = _get_settings(botobj)['free_partition']
  # On Windows, drive letters are always lower case.
  root = 'c:\\' if sys.platform == 'win32' else '/'

  errors = []
  def _check_for_quarantine(r, i, key):
    min_free = _min_free_disk(i, settings[key])
    if int(i[u'free_mb']*1024*1024) < min_free:
      errors.append(
          u'Not enough free disk space on %s. %.1fmib < %.1fmib' %
          (r, i[u'free_mb'], round(min_free / 1024. / 1024., 1)))

  # root may be missing in the case of netbooted devices.
  if root in disks:
    _check_for_quarantine(root, disks[root], 'root')

  # Try again with the bot's base directory. It is frequent to run the bot
  # from a secondary partition, to reduce the risk of OS failure due to full
  # root partition.
  # This code is similar to os_utilities.get_disk_size().
  path = botobj.base_dir
  case_insensitive = sys.platform in ('darwin', 'win32')
  if case_insensitive:
    path = path.lower()
  for mount, infos in sorted(disks.iteritems(), key=lambda x: -len(x[0])):
    if path.startswith(mount.lower() if case_insensitive else mount):
      # Apply 'bot' check if bot is on its own partition, or it's on
      # root partition and there are no errors reported yet.
      if mount != root or not errors:
        _check_for_quarantine(mount, infos, 'bot')
      break
  if errors:
    return '\n'.join(errors)


def _get_authentication_headers(botobj):
  """Calls bot_config.get_authentication_headers() if it is defined.

  See remote_client.RemoteClient doc for the expected format of the return
  value.

  Doesn't catch exceptions. RemoteClient knows how to deal with them.
  """
  return _call_hook(False, botobj, 'get_authentication_headers') or (None, None)


def _on_shutdown_hook(b):
  """Called when the bot is restarting."""
  _call_hook_safe(True, b, 'on_bot_shutdown')
  # Aggressively set itself up so we ensure the auto-reboot configuration is
  # fine before restarting the host. This is important as some tasks delete the
  # autorestart script (!)
  setup_bot(True)


def _min_free_disk(infos, settings):
  """Returns the calculated minimum free disk space for this partition.

  See _get_settings() in ../config/bot_config.py for an explanation.
  """
  size = int(infos[u'size_mb']*1024*1024)
  x1 = settings['size'] or 0
  x2 = int(round(size * float(settings['max_percent'] or 0) * 0.01))
  # Select the lowest non-zero value.
  x = min(x1, x2) if (x1 and x2) else (x1 or x2)
  # Select the maximum value.
  return max(x, int(round(size * float(settings['min_percent'] or 0) * 0.01)))


def _dict_deep_merge(x, y):
  """Returns the union of x and y.

  y takes predescence.
  """
  if x is None:
    return y
  if y is None:
    return x
  if isinstance(x, dict):
    if isinstance(y, dict):
      return {k: _dict_deep_merge(x.get(k), y.get(k)) for k in set(x).union(y)}
    assert y is None, repr(y)
    return x
  if isinstance(y, dict):
    assert x is None, repr(x)
    return y
  # y is overriding x.
  return y


def _is_base_dir_ok(botobj):
  """Returns False if the bot must be quarantined at all cost."""
  if not botobj:
    # This can happen very early in the process lifetime.
    return THIS_DIR != os.path.expanduser('~')
  return botobj.base_dir != os.path.expanduser('~')


### Public functions used by __main__.py


def setup_bot(skip_reboot):
  """Calls bot_config.setup_bot() to have the bot self-configure itself.

  Reboots the host if bot_config.setup_bot() returns False, unless skip_reboot
  is also true.

  Does nothing if SWARMING_EXTERNAL_BOT_SETUP env var is set to 1. It is set in
  case bot's autostart configuration is managed elsewhere, and we don't want
  the bot itself to interfere.
  """
  if os.environ.get('SWARMING_EXTERNAL_BOT_SETUP') == '1':
    logging.info('Skipping setup_bot, SWARMING_EXTERNAL_BOT_SETUP is set')
    return

  botobj = get_bot(get_config())
  try:
    from config import bot_config
  except Exception as e:
    msg = '%s\n%s' % (e, traceback.format_exc()[-2048:])
    botobj.post_error('bot_config.py is bad: %s' % msg)
    return

  # TODO(maruel): Convert the should_continue return value to the hook calling
  # botobj.host_reboot() by itself.
  try:
    should_continue = bot_config.setup_bot(botobj)
  except Exception as e:
    msg = '%s\n%s' % (e, traceback.format_exc()[-2048:])
    botobj.post_error('bot_config.setup_bot() threw: %s' % msg)
    return

  if not should_continue and not skip_reboot:
    botobj.host_reboot('Starting new swarming bot: %s' % THIS_FILE)


@tools.cached
def generate_version():
  """Returns the bot's code version."""
  try:
    return zip_package.generate_version()
  except Exception as e:
    return 'Error: %s' % e


def get_attributes(botobj):
  """Returns the attributes sent to the server in /handshake.

  Each called function catches all exceptions so the bot doesn't die on startup,
  which is annoying to recover. In that case, we set a special property to catch
  these and help the admin fix the swarming_bot code more quickly.

  Arguments:
  - botobj: bot.Bot instance or None
  """
  return {
    u'dimensions': _get_dimensions(botobj),
    u'state': _get_state(botobj, 0),
    u'version': generate_version(),
  }


def get_bot(config):
  """Returns a valid Bot instance.

  Should only be called once in the process lifetime.

  It can be called by ../__main__.py, something to keep in mind.
  """
  # This variable is used to bootstrap the initial bot.Bot object, which then is
  # used to get the dimensions and state.
  attributes = {
    'dimensions': {u'id': ['none']},
    'state': {},
    'version': generate_version(),
  }
  hostname = _get_botid_safe()
  base_dir = THIS_DIR
  # Use temporary Bot object to call get_attributes. Attributes are needed to
  # construct the "real" bot.Bot.
  attributes = get_attributes(
    bot.Bot(
      remote_client.createRemoteClient(config['server'],
                                       None,
                                       hostname,
                                       base_dir,
                                       config.get('swarming_grpc_proxy')),
      attributes,
      config['server'],
      config['server_version'],
      base_dir,
      _on_shutdown_hook))

  # Make remote client callback use the returned bot object. We assume here
  # RemoteClient doesn't call its callback in the constructor (since 'botobj' is
  # undefined during the construction).
  botobj = bot.Bot(
      remote_client.createRemoteClient(
          config['server'],
          lambda: _get_authentication_headers(botobj),
          hostname,
          base_dir,
          config.get('swarming_grpc_proxy')),
      attributes,
      config['server'],
      config['server_version'],
      base_dir,
      _on_shutdown_hook)
  return botobj


@tools.cached
def get_config():
  """Returns the data from config.json."""
  global _ERROR_HANDLER_WAS_REGISTERED
  try:
    with contextlib.closing(zipfile.ZipFile(THIS_FILE, 'r')) as f:
      config = json.load(f.open('config/config.json', 'r'))
    if config['server'].endswith('/'):
      raise ValueError('Invalid server entry %r' % config['server'])
  except (zipfile.BadZipfile, IOError, OSError, TypeError, ValueError):
    logging.exception('Invalid config.json!')
    config = {
      u'server': u'',
      u'server_version': u'N/A',
    }
  if not _ERROR_HANDLER_WAS_REGISTERED and config['server']:
    on_error.report_on_exception_exit(config['server'])
    _ERROR_HANDLER_WAS_REGISTERED = True
  return config


### Sub process management


def _cleanup_bot_directory(botobj):
  """Delete anything not expected in the swarming bot directory.

  This helps with stale work directory or any unexpected junk that could cause
  this bot to self-quarantine. Do only this when running from the zip.
  """
  if not _is_base_dir_ok(botobj):
    # That's an important one-off check as cleaning the $HOME directory has
    # really bad effects on normal host.
    logging.error('Not cleaning root directory because of bad base directory')
    return
  for i in fs.listdir(botobj.base_dir):
    if any(fnmatch.fnmatch(i, w) for w in PASSLIST):
      continue
    try:
      p = unicode(os.path.join(botobj.base_dir, i))
      if fs.isdir(p):
        file_path.rmtree(p)
      else:
        file_path.remove(p)
    except (IOError, OSError) as e:
      botobj.post_error(
          'Failed to remove %s from bot\'s directory: %s' % (i, e))


def _run_isolated_flags(botobj):
  """Returns flags to pass to run_isolated.

  These are not meant to be processed by task_runner.py.
  """
  settings = _get_settings(botobj)
  partition = settings['free_partition']['bot']
  size = os_utilities.get_disk_size(THIS_FILE)
  min_free = (
      _min_free_disk({'size_mb': size}, partition) +
      partition['wiggle'])
  args = [
    '--cache', os.path.join(botobj.base_dir, 'isolated_cache'),
    '--min-free-space', str(min_free),
    '--named-cache-root', os.path.join(botobj.base_dir, 'c'),
    '--max-cache-size', str(settings['caches']['isolated']['size']),
    '--max-items', str(settings['caches']['isolated']['items']),
  ]

  # Get the gRPC proxy from the config, but allow an environment variable to
  # override.
  grpc_proxy = get_config().get('isolate_grpc_proxy')
  grpc_proxy = os.environ.get('ISOLATE_GRPC_PROXY', grpc_proxy)
  if grpc_proxy:
    logging.info('Isolate will use gRPC proxy %s', grpc_proxy)
    args.extend(['--grpc-proxy', grpc_proxy])
  return args


def _clean_cache(botobj):
  """Asks run_isolated to clean its cache.

  This may take a while but it ensures that in the case of a run_isolated run
  failed and it temporarily used more space than _min_free_disk, it can cleans
  up the mess properly.

  It will remove unexpected files, remove corrupted files, trim the cache size
  based on the policies and update state.json.
  """
  cmd = [
    sys.executable, THIS_FILE, 'run_isolated',
    '--clean',
    '--log-file', os.path.join(botobj.base_dir, 'logs', 'run_isolated.log'),
  ]
  cmd.extend(_run_isolated_flags(botobj))
  logging.info('Running: %s', cmd)
  try:
    # Intentionally do not use a timeout, it can take a while to hash 50gb but
    # better be safe than sorry.
    proc = subprocess42.Popen(
        cmd,
        stdin=subprocess42.PIPE,
        stdout=subprocess42.PIPE, stderr=subprocess42.STDOUT,
        cwd=botobj.base_dir,
        detached=True,
        close_fds=sys.platform != 'win32')
    output, _ = proc.communicate(None)
    logging.info('Result:\n%s', output)
    if proc.returncode:
      botobj.post_error(
          'swarming_bot.zip failure during run_isolated --clean:\n%s' % output)
  except OSError:
    botobj.post_error(
        'swarming_bot.zip internal failure during run_isolated --clean')


def _post_error_task(botobj, error, task_id):
  """Posts given error as failure cause for the task.

  This is used in case of internal code error, and this causes the task to
  become BOT_DIED.

  Arguments:
    botobj: A bot.Bot instance.
    error: String representing the problem.
    task_id: Task that had an internal error. When the Swarming server sends
        commands to a bot, even though they could be completely wrong, the
        server assumes the job as running. Thus this function acts as the
        exception handler for incoming commands from the Swarming server. If for
        any reason the local test runner script can not be run successfully,
        this function is invoked.
  """
  logging.error('Error: %s', error)
  return botobj.remote.post_task_error(task_id, botobj.id, error)


def _run_manifest(botobj, manifest, start):
  """Defers to task_runner.py.

  Return True if the task succeeded.
  """
  # Ensure the manifest is valid. This can throw a json decoding error. Also
  # raise if it is empty.
  if not manifest:
    raise ValueError('Empty manifest')

  # Necessary to signal an internal_failure. This occurs when task_runner fails
  # to execute the command. It is important to note that this data is extracted
  # before any I/O is done, like writting the manifest to disk.
  task_id = manifest['task_id']
  last_ditch_timeout = manifest['hard_timeout'] or None
  # The grace period is the time between SIGTERM and SIGKILL.
  grace_period = max(manifest['grace_period'] or 0, 30)
  if last_ditch_timeout:
    # One for the child process, one for run_isolated, one for task_runner.
    last_ditch_timeout += 3 * grace_period
    # CIPD, isolated download time, plus named cache cleanup is not counted for
    # hard timeout so add more time; hard_timeout is handled by run_isolated.
    last_ditch_timeout += max(manifest['io_timeout'] or 0, 1200)

  # Get the server info to pass to the task runner so it can provide updates.
  url = botobj.remote.server
  if not botobj.remote.is_grpc and 'host' in manifest:
    # The URL in the manifest includes the version - eg not https://chromium-
    # swarm-dev.appspot.com, but https://<some-version>-dot-chromiium-swarm-
    # dev.appspot.com. That way, if a new server version becomes the default,
    # old bots will continue to work with a server version that can manipulate
    # the old data (the new server will only ever have to read it, which is
    # much simpler) while new bots won't accidentally contact an old server
    # which the GAE engine hasn't gotten around to updating yet.
    #
    # With a gRPC proxy, we could theoretically run into the same problem
    # if we change the meaning of some data without changing the protos.
    # However, if we *do* change the protos, we already need to make the
    # change in a few steps:
    #    1. Modify the Swarming server to accept the new data
    #    2. Modify the protos and the proxy to accept the new data
    #       in gRPC calls and translate it to "native" Swarming calls.
    #    3. Update the bots to transmit the new protos.
    # Throughout all this, the proto format itself irons out minor differences
    # and additions. But because we deploy in three steps, the odds of a
    # newer bot contacting an older server is very low.
    #
    # None of this applies if we don't actually update the protos but just
    # change the semantics. If this becomes a significant problem, we could
    # start transmitting the expected server version using gRPC metadata.
    #    - aludwin, Nov 2016
    url = manifest['host']

  task_dimensions = manifest['dimensions']
  task_result = {}

  failure = False
  internal_failure = False
  msg = None
  auth_params_dumper = None
  must_reboot = False
  # Use 'w' instead of 'work' because path length is precious on Windows.
  work_dir = os.path.join(botobj.base_dir, u'w')
  try:
    try:
      if fs.isdir(work_dir):
        file_path.rmtree(work_dir)
    except OSError:
      # If a previous task created an undeleteable file/directory inside 'w',
      # make sure that following tasks are not affected. This is done by working
      # around the undeleteable directory by creating a temporary directory
      # instead. This is not normal behavior. The bot will report a failure on
      # start.
      work_dir = tempfile.mkdtemp(dir=botobj.base_dir, prefix=u'w')
    else:
      try:
        fs.makedirs(work_dir)
      except OSError:
        # Sometimes it's a race condition, so do a last ditch attempt.
        work_dir = tempfile.mkdtemp(dir=botobj.base_dir, prefix=u'w')

    env = os.environ.copy()
    # Windows in particular does not tolerate unicode strings in environment
    # variables.
    env['SWARMING_TASK_ID'] = task_id.encode('ascii')
    env['SWARMING_SERVER'] = botobj.server.encode('ascii')

    task_in_file = os.path.join(work_dir, 'task_runner_in.json')
    with fs.open(task_in_file, 'wb') as f:
      f.write(json.dumps(manifest))
    handle, bot_file = tempfile.mkstemp(
        prefix='bot_file', suffix='.json', dir=work_dir)
    os.close(handle)
    task_result_file = os.path.join(work_dir, 'task_runner_out.json')
    if fs.exists(task_result_file):
      fs.remove(task_result_file)

    # Start a thread that periodically puts authentication headers and other
    # authentication related information to a file on disk. task_runner reads it
    # from there before making authenticated HTTP calls.
    #
    # TODO(vadimsh): Switch to pipes or local sockets if the latency tokens
    # propagation here becomes an issue.
    auth_params_file = os.path.join(work_dir, 'bot_auth_params.json')
    auth_params_dumper = file_refresher.FileRefresherThread(
        auth_params_file,
        lambda: bot_auth.prepare_auth_params_json(botobj, manifest))
    auth_params_dumper.start()

    command = [
      sys.executable, THIS_FILE, 'task_runner',
      '--swarming-server', url,
      '--in-file', task_in_file,
      '--out-file', task_result_file,
      '--cost-usd-hour', str(botobj.state.get('cost_usd_hour') or 0.),
      # Include the time taken to poll the task in the cost.
      '--start', str(start),
      '--bot-file', bot_file,
      '--auth-params-file', auth_params_file,
    ]
    if botobj.remote.is_grpc:
      command.append('--is-grpc')
    # Flags for run_isolated.py are passed through by task_runner.py as-is
    # without interpretation.
    command.append('--')
    command.extend(_run_isolated_flags(botobj))
    _call_hook_safe(True, botobj, 'on_before_task', bot_file, command, env)
    logging.debug('Running command: %s', command)

    base_log = os.path.join(botobj.base_dir, u'logs')
    if not fs.isdir(base_log):
      # It was observed that this directory may be unexpectedly deleted.
      # Recreate as needed, otherwise it may throw at the open() call below.
      fs.mkdir(base_log)
    log_path = os.path.join(base_log, 'task_runner_stdout.log')
    os_utilities.roll_log(log_path)
    os_utilities.trim_rolled_log(log_path)
    with fs.open(log_path, 'a+b') as f:
      proc = subprocess42.Popen(
          command,
          detached=True,
          cwd=botobj.base_dir,
          env=env,
          stdin=subprocess42.PIPE,
          stdout=f,
          stderr=subprocess42.STDOUT,
          close_fds=sys.platform != 'win32')
      try:
        proc.wait(last_ditch_timeout)
      except subprocess42.TimeoutExpired:
        # That's the last ditch effort; as task_runner should have completed a
        # while ago and had enforced the io_timeout or run_isolated for
        # hard_timeout.
        logging.error('Sending SIGTERM to task_runner')
        proc.terminate()
        internal_failure = True
        msg = 'task_runner hung'
        try:
          proc.wait(2*grace_period)
        except subprocess42.TimeoutExpired:
          logging.error('Sending SIGKILL to task_runner')
          proc.kill()
        proc.wait()
        return False

    logging.info('task_runner exit: %d', proc.returncode)
    if fs.exists(task_result_file):
      with fs.open(task_result_file, 'rb') as fd:
        task_result = json.load(fd)

    if proc.returncode:
      # STATUS_DLL_INIT_FAILED generally means that something bad happened, and
      # a reboot magically clears things out. :(
      must_reboot = sys.platform == 'win32' and proc.returncode == -1073741502
      msg = 'Execution failed: internal error (%d).' % proc.returncode
      internal_failure = True
    elif not task_result:
      logging.warning('task_runner failed to write metadata')
      msg = 'Execution failed: internal error (no metadata).'
      internal_failure = True
    elif task_result[u'must_signal_internal_failure']:
      msg = (
        'Execution failed: %s' % task_result[u'must_signal_internal_failure'])
      internal_failure = True

    failure = bool(task_result.get('exit_code')) if task_result else False
    return not internal_failure and not failure
  except Exception as e:
    # Failures include IOError when writing if the disk is full, OSError if
    # swarming_bot.zip doesn't exist anymore, etc.
    logging.exception('_run_manifest failed')
    msg = 'Internal exception occured: %s\n%s' % (
        e, traceback.format_exc()[-2048:])
    internal_failure = True
  finally:
    if auth_params_dumper:
      auth_params_dumper.stop()
    if internal_failure:
      _post_error_task(botobj, msg, task_id)
    _call_hook_safe(
        True, botobj, 'on_after_task', failure, internal_failure,
        task_dimensions, task_result)
    if fs.isdir(work_dir):
      try:
        file_path.rmtree(work_dir)
      except Exception as e:
        botobj.post_error(
            'Failed to delete work directory %s: %s' % (work_dir, e))
    if must_reboot:
      botobj.host_reboot('Working around STATUS_DLL_INIT_FAILED by task_runner')


### Bot lifetime management


def _run_bot(arg_error):
  """Runs _run_bot_inner() with a signal handler."""
  # The quit_bit is to signal that the bot process must shutdown. It is
  # different from a request to restart the bot process or reboot the host.
  quit_bit = threading.Event()
  def handler(sig, _):
    # A signal terminates the bot process, it doesn't cause it to restart.
    logging.info('Got signal %s', sig)
    quit_bit.set()

  # TODO(maruel): Set quit_bit when stdin is closed on Windows.

  with subprocess42.set_signal_handler(subprocess42.STOP_SIGNALS, handler):
    return _run_bot_inner(arg_error, quit_bit)


def _run_bot_inner(arg_error, quit_bit):
  """Runs the bot until an event occurs.

  One of the three following even can occur:
  - host reboots
  - bot process restarts (this includes self-update)
  - bot process shuts down (this includes a signal is received)
  """
  config = get_config()
  if config.get('enable_ts_monitoring'):
    _init_ts_mon()
  try:
    # First thing is to get an arbitrary url. This also ensures the network is
    # up and running, which is necessary before trying to get the FQDN below.
    # There's no need to do error handling here - the "ping" is just to "wake
    # up" the network; if there's something seriously wrong, the handshake will
    # fail and we'll handle it there.
    hostname = _get_botid_safe()
    base_dir = os.path.dirname(THIS_FILE)
    remote = remote_client.createRemoteClient(config['server'], None,
                                              hostname, base_dir,
                                              config.get('swarming_grpc_proxy'))
    remote.ping()
  except Exception:
    # url_read() already traps pretty much every exceptions. This except
    # clause is kept there "just in case".
    logging.exception('server_ping threw')

  # If we are on GCE, we want to make sure GCE metadata server responds, since
  # we use the metadata to derive bot ID, dimensions and state.
  if platforms.is_gce():
    logging.info('Running on GCE, waiting for the metadata server')
    platforms.gce.wait_for_metadata(quit_bit)
    if quit_bit.is_set():
      logging.info('Early quit 1')
      return 0

  # Next we make sure the bot can make authenticated calls by grabbing the auth
  # headers, retrying on errors a bunch of times. We don't give up if it fails
  # though (maybe the bot will "fix itself" later).
  botobj = get_bot(config)
  try:
    botobj.remote.initialize(quit_bit)
  except remote_client.InitializationError as exc:
    botobj.post_error('failed to grab auth headers: %s' % exc.last_error)
    logging.error('Can\'t grab auth headers, continuing anyway...')

  if arg_error:
    botobj.post_error('Bootstrapping error: %s' % arg_error)

  if quit_bit.is_set():
    logging.info('Early quit 2')
    return 0

  _call_hook_safe(True, botobj, 'on_bot_startup')

  # Initial attributes passed to bot.Bot in get_bot above were constructed for
  # 'fake' bot ID ('none'). Refresh them to match the real bot ID, now that we
  # have fully initialize bot.Bot object. Note that 'get_dimensions' and
  # 'get_state' may depend on actions done by 'on_bot_startup' hook, that's why
  # we do it here and not in 'get_bot'.
  dims = _get_dimensions(botobj)
  states = _get_state(botobj, 0)
  with botobj._lock:
    botobj._update_dimensions(dims)
    botobj._update_state(states)

  if quit_bit.is_set():
    logging.info('Early quit 3')
    return 0

  _do_handshake(botobj, quit_bit)

  if quit_bit.is_set():
    logging.info('Early quit 4')
    return 0

  # Let the bot to finish the initialization, now that it knows its server
  # defined dimensions.
  _call_hook_safe(True, botobj, 'on_handshake')

  _cleanup_bot_directory(botobj)
  _clean_cache(botobj)

  if quit_bit.is_set():
    logging.info('Early quit 5')
    return 0

  # This environment variable is accessible to the tasks executed by this bot.
  os.environ['SWARMING_BOT_ID'] = botobj.id.encode('utf-8')

  consecutive_sleeps = 0
  last_action = time.time()
  while not quit_bit.is_set():
    try:
      _call_hook_safe(False, botobj, 'on_before_poll')
      dims = _get_dimensions(botobj)
      states = _get_state(botobj, consecutive_sleeps)
      with botobj._lock:
        botobj._update_dimensions(dims)
        botobj._update_state(states)
      did_something = _poll_server(botobj, quit_bit, last_action)
      if did_something:
        last_action = time.time()
        consecutive_sleeps = 0
      else:
        consecutive_sleeps += 1
    except Exception as e:
      logging.exception('_poll_server failed in a completely unexpected way')
      msg = '%s\n%s' % (e, traceback.format_exc()[-2048:])
      botobj.post_error(msg)
      consecutive_sleeps = 0
      # Sleep a bit as a precaution to avoid hammering the server.
      quit_bit.wait(10)
  # Tell the server we are going away.
  botobj.post_event('bot_shutdown', 'Signal was received')
  return 0


def _should_have_exited_but_didnt(reason):
  """Something super sad happened, set the sticky quarantine bit before polling
  again and sleep a bit to prevent busy-loop/DDoS.
  """
  time.sleep(2)
  _set_quarantined(reason)


def _do_handshake(botobj, quit_bit):
  """Connects to /handshake and reads the bot_config if specified."""
  # This is the first authenticated request to the server. If the bot is
  # misconfigured, the request may fail with HTTP 401 or HTTP 403. Instead of
  # dying right away, spin in a loop, hoping the bot will "fix itself"
  # eventually. Authentication errors in /handshake are logged on the server and
  # generate error reports, so bots stuck in this state are discoverable.
  sleep_time = 5
  while not quit_bit.is_set():
    resp = botobj.remote.do_handshake(botobj._attributes)
    if resp:
      logging.info('Connected to %s', resp.get('server_version'))
      if resp.get('bot_version') != botobj._attributes['version']:
        logging.warning(
            'Found out we\'ll need to update: server said %s; we\'re %s',
            resp.get('bot_version'), botobj._attributes['version'])
      # Remember the server-provided per-bot configuration. '/handshake' is
      # the only place where the server returns it. The bot will be sending
      # the 'bot_group_cfg_version' back in each /poll (as part of 'state'),
      # so that the server can instruct the bot to restart itself when
      # config changes.
      cfg_version = resp.get('bot_group_cfg_version')
      if cfg_version:
        botobj._update_bot_group_cfg(cfg_version, resp.get('bot_group_cfg'))
      content = resp.get('bot_config')
      if content:
        _register_extra_bot_config(content)
      break
    logging.error(
        'Failed to contact for handshake, retrying in %d sec...', sleep_time)
    quit_bit.wait(sleep_time)
    sleep_time = min(300, sleep_time * 2)


def _poll_server(botobj, quit_bit, last_action):
  """Polls the server to run one loop.

  Returns True if executed some action, False if server asked the bot to sleep.
  """
  start = time.time()
  cmd = None
  try:
    cmd, value = botobj.remote.poll(botobj._attributes)
  except remote_client_errors.PollError as e:
    # Back off on failure.
    delay = max(1, min(60, botobj.state.get(u'sleep_streak', 10) * 2))
    logging.warning('Poll failed (%s), sleeping %.1f sec', e, delay)
    quit_bit.wait(delay)
    return False
  finally:
    _call_hook_safe(False, botobj, 'on_after_poll', cmd)

  logging.debug('Server response:\n%s: %s', cmd, value)

  if cmd == 'sleep':
    # Value is duration
    _call_hook_safe(
        True, botobj, 'on_bot_idle', max(0, time.time() - last_action))
    _maybe_update_lkgbc(botobj)
    try:
      # Sometimes throw with "[Errno 4] Interrupted function call", especially
      # on Windows upon system shutdown.
      quit_bit.wait(value)
    except IOError:
      # Act as it if were set as this likely mean a system shutdown.
      quit_bit.set()
    return False

  if cmd == 'terminate':
    # The value is the task ID to serve as the special termination command.
    quit_bit.set()
    try:
      # Duration must be set or server IEs. For that matter, we've never cared
      # if there's an error here before, so let's preserve that behaviour
      # (though anything that's not a remote_client.InternalError will make
      # it through, again preserving prior behaviour).
      botobj.remote.post_task_update(value, botobj.id, {'duration':0}, None, 0)
    except remote_client_errors.InternalError:
      pass
    return False

  if cmd == 'run':
    # Value is the manifest
    success = _run_manifest(botobj, value, start)
    # Unconditionally clean up cache after each task. This is done *after* the
    # task is terminated, so that:
    # - there's no task overhead
    # - if there's an exception while cleaning, it's not logged in the task
    _clean_cache(botobj)
    if success:
      # Completed a task successfully so update swarming_bot.zip if necessary.
      _update_lkgbc(botobj)
    # TODO(maruel): Handle the case where quit_bit.is_set() happens here. This
    # is concerning as this means a signal (often SIGTERM) was received while
    # running the task. Make sure the host is properly restarting.
  elif cmd == 'update':
    # Value is the version
    _update_bot(botobj, value)
    _should_have_exited_but_didnt('Failed to self-update the bot')
  elif cmd in ('host_reboot', 'restart'):
    # Value is the message to display while rebooting the host
    botobj.host_reboot(value)
    _should_have_exited_but_didnt('Failed to reboot the host')
  elif cmd == 'bot_restart':
    # Value is the message to display while restarting
    _bot_restart(botobj, value)
    _should_have_exited_but_didnt('Failed to restart the bot process')
  else:
    raise ValueError('Unexpected command: %s\n%s' % (cmd, value))

  return True


def _update_bot(botobj, version):
  """Downloads the new version of the bot code and then runs it.

  Use alternating files; first load swarming_bot.1.zip, then swarming_bot.2.zip,
  never touching swarming_bot.zip which was the originally bootstrapped file.

  LKGBC is handled by _update_lkgbc() and _maybe_update_lkgbc().

  Returns only in case of failure to get the new bot code.
  """
  # Alternate between .1.zip and .2.zip.
  new_zip = 'swarming_bot.1.zip'
  if os.path.basename(THIS_FILE) == new_zip:
    new_zip = 'swarming_bot.2.zip'
  new_zip = os.path.join(botobj.base_dir, new_zip)

  # Download as a new file.
  try:
    botobj.remote.get_bot_code(new_zip, version, botobj.id)
  except remote_client.BotCodeError as e:
    botobj.post_error(str(e))
  else:
    _bot_restart(botobj, 'Updating to %s' % version, filepath=new_zip)


def _bot_restart(botobj, message, filepath=None):
  """Restarts the bot process, optionally in a new file.

  The function will return if the new bot code is not valid.
  """
  filepath = filepath or THIS_FILE
  s = fs.stat(filepath)
  logging.info('Restarting to %s; %d bytes.', filepath, s.st_size)
  sys.stdout.flush()
  sys.stderr.flush()

  proc = subprocess42.Popen(
     [sys.executable, filepath, 'is_fine'],
     stdout=subprocess42.PIPE, stderr=subprocess42.STDOUT)
  output, _ = proc.communicate()
  if proc.returncode:
    botobj.post_error(
        'New bot code is bad: proc exit = %s. stdout:\n%s' %
        (proc.returncode, output))
    if sys.platform == 'win32' and proc.returncode == -1073741502:
      # STATUS_DLL_INIT_FAILED generally means that something bad happened, and
      # a reboot magically clears things out. :(
      botobj.host_reboot(
          'Working around STATUS_DLL_INIT_FAILED when restarting the bot')
    return

  botobj.post_event('bot_shutdown', 'About to restart: %s' % message)

  # Sleep a bit to make sure new bot process connects to a GAE instance with
  # the fresh bot group config cache (it gets refreshed each second). This makes
  # sure the bot doesn't accidentally pick up the old config after restarting
  # and connecting to an instance with a stale cache.
  if not botobj.remote.is_grpc:
    time.sleep(2)

  # Don't forget to release the singleton before restarting itself.
  SINGLETON.release()

  # Do not call on_bot_shutdown.
  # On OSX, launchd will be unhappy if we quit so the old code bot process has
  # to outlive the new code child process. Launchd really wants the main process
  # to survive, and it'll restart it if it disappears. os.exec*() replaces the
  # process so this is fine.
  ret = common.exec_python([filepath, 'start_slave', '--survive'])
  if ret in (1073807364, -1073741510):
    # 1073807364 is returned when the process is killed due to shutdown. No need
    # to alert anyone in that case.
    # -1073741510 is returned when rebooting too. This can happen when the
    # parent code was running the old version and gets confused and decided to
    # poll again.
    # In any case, zap out the error code.
    ret = 0
  elif ret:
    botobj.post_error('Bot failed to respawn after update: %s' % ret)
  sys.exit(ret)


def _update_lkgbc(botobj):
  """Updates the Last Known Good Bot Code if necessary.

  Returns True if LKGBC was updated.
  """
  try:
    if not fs.isfile(THIS_FILE):
      # TODO(maruel): Try to download the code again from the server.
      botobj.post_error('Missing file %s for LKGBC' % THIS_FILE)
      return False

    golden = os.path.join(botobj.base_dir, 'swarming_bot.zip')
    if fs.isfile(golden):
      org = fs.stat(golden)
      cur = fs.stat(THIS_FILE)
      if org.st_size == org.st_size and org.st_mtime >= cur.st_mtime:
        return False

    # Copy the current file back to LKGBC.
    shutil.copy(THIS_FILE, golden)
    return True
  except Exception as e:
    botobj.post_error('Failed to update LKGBC: %s' % e)
    return False


def _maybe_update_lkgbc(botobj):
  """Updates the Last Known Good Bot Code (LKGBC) when it is older than 1 one
  week.

  This either means:
  - The bot code is particularly hosed, for an extended period of time.
  - All tasks are failing, which may legitimitely happen sometimes when the devs
    don't care about a particular configuration.
  - The bot is completely idle, so it never gets the chance to update LKGBC.

  We decide that the first situation is rare enough that it's preferable to
  explcitly handle the later two. We've seen bots being idle for extended
  periods of time (well over a year), which introduces all sorts of
  'interesting' problems.

  Returns True if LKGBC was updated.
  """
  try:
    if not fs.isfile(THIS_FILE):
      # TODO(maruel): Try to download the code again from the server.
      return False
    golden = os.path.join(botobj.base_dir, u'swarming_bot.zip')
    if fs.isfile(golden):
      org = fs.stat(golden)
      cur = fs.stat(THIS_FILE)
      if org.st_size == org.st_size and org.st_mtime >= cur.st_mtime:
        return False
      if org.st_mtime >= time.time() - 7*24*60*60:
        return False

    # Copy the current file back to LKGBC.
    shutil.copy(THIS_FILE, golden)
    return True
  except Exception as e:
    botobj.post_error('Failed to update LKGBC while idle: %s' % e)
    return False


def main(argv):
  subprocess42.inhibit_os_error_reporting()

  # Disable magical auto-detection of OAuth config. bot_main.py prepares auth
  # headers on its own explicitly (via get_authentication_headers hook) when
  # using 'net' library through RemoteClientNative class and auto-configured
  # auth in net.py may interfere with this. We also disable auto-detection in
  # task_runner.py (since it also uses special mechanism for getting auth
  # headers from bot_main.py). We do _not_ disable auto-detection in
  # run_isolated.py, since at this layer we have an auth context (setup by
  # task_runner.py) and it is correctly getting recognized by the auto-detection
  # in net.py.
  net.disable_oauth_config()

  # Add SWARMING_HEADLESS into environ so subcommands know that they are running
  # in a headless (non-interactive) mode.
  os.environ['SWARMING_HEADLESS'] = '1'

  # The only reason this is kept is to enable the unit test to use --help to
  # quit the process.
  parser = argparse.ArgumentParser(description=sys.modules[__name__].__doc__)
  parser.add_argument('unsupported', nargs='*', help=argparse.SUPPRESS)
  args = parser.parse_args(argv)

  if sys.platform == 'win32':
    if not file_path.enable_privilege('SeShutdownPrivilege'):
      logging.error('Failed to enable SeShutdownPrivilege')

  # Enforces that only one process with a bot in this directory can be run on
  # this host at once.
  if not SINGLETON.acquire():
    if sys.platform == 'darwin':
      msg = (
          'Found a previous bot, %d rebooting as a workaround for '
          'https://crbug.com/569610.') % os.getpid()
      print >> sys.stderr, msg
      os_utilities.host_reboot(msg)
    else:
      print >> sys.stderr, 'Found a previous bot, %d exiting.' % os.getpid()
    return 1

  base_dir = os.path.dirname(THIS_FILE)
  for t in ('out', 'err'):
    log_path = os.path.join(base_dir, 'logs', 'bot_std%s.log' % t)
    os_utilities.roll_log(log_path)
    os_utilities.trim_rolled_log(log_path)

  error = None
  if len(args.unsupported) != 0:
    error = 'Unexpected arguments: %s' % args
  try:
    return _run_bot(error)
  finally:
    _call_hook_safe(
        True, bot.Bot(None, None, None, None, base_dir, None),
        'on_bot_shutdown')
    logging.info('main() returning')
