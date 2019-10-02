#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Starts a local bot to connect to a local server."""

import argparse
import glob
import logging
import os
import shutil
import signal
import socket
import sys
import tempfile
import urllib


THIS_DIR = os.path.dirname(os.path.abspath(__file__))
CLIENT_DIR = os.path.join(THIS_DIR, '..', '..', '..', 'client')
sys.path.insert(0, CLIENT_DIR)
from third_party.depot_tools import fix_encoding
from utils import file_path
from utils import subprocess42
sys.path.pop(0)


def _safe_rm(path):
  if os.path.exists(path):
    try:
      file_path.rmtree(path)
    except OSError as e:
      logging.error('Failed to delete %s: %s', path, e)


class LocalBot(object):
  """A local running Swarming bot.

  It creates its own temporary directory to download the zip and run tasks
  locally.
  """
  def __init__(self, swarming_server_url, redirect, botdir):
    self._botdir = botdir
    self._swarming_server_url = swarming_server_url
    self._proc = None
    self._logs = {}
    self._redirect = redirect

  def wipe_cache(self, restart):
    """Blows away this bot's cache and restart it.

    There's just too much risk of the bot failing over so it's not worth not
    restarting it.
    """
    if restart:
      logging.info('wipe_cache(): Restarting the bot')
      self.stop()
      # Deletion needs to happen while the bot is not running to ensure no side
      # effect.
      # These values are from ./swarming_bot/bot_code/bot_main.py.
      _safe_rm(os.path.join(self._botdir, 'c'))
      _safe_rm(os.path.join(self._botdir, 'isolated_cache'))
      self.start()
    else:
      logging.info('wipe_cache(): wiping cache without telling the bot')
      _safe_rm(os.path.join(self._botdir, 'c'))
      _safe_rm(os.path.join(self._botdir, 'isolated_cache'))

  @property
  def bot_id(self):
    # TODO(maruel): Big assumption.
    return socket.getfqdn().split('.')[0]

  @property
  def log(self):
    """Returns the log output. Only set after calling stop()."""
    return '\n'.join(self._logs.itervalues()) if self._logs else None

  def start(self):
    """Starts the local Swarming bot."""
    assert not self._proc
    bot_zip = os.path.join(self._botdir, 'swarming_bot.zip')
    urllib.urlretrieve(self._swarming_server_url + '/bot_code', bot_zip)
    cmd = [sys.executable, bot_zip, 'start_slave']
    if self._redirect:
      logs = os.path.join(self._botdir, 'logs')
      if not os.path.isdir(logs):
        os.mkdir(logs)
      with open(os.path.join(logs, 'bot_stdout.log'), 'wb') as f:
        self._proc = subprocess42.Popen(
            cmd, cwd=self._botdir, stdout=f, stderr=f, detached=True)
    else:
      self._proc = subprocess42.Popen(cmd, cwd=self._botdir, detached=True)

  def stop(self):
    """Stops the local Swarming bot. Returns the process exit code."""
    if not self._proc:
      return None
    if self._proc.poll() is None:
      try:
        self._proc.send_signal(signal.SIGTERM)
        # TODO(maruel): SIGKILL after N seconds.
        self._proc.wait()
      except OSError:
        pass
    exit_code = self._proc.returncode
    for i in sorted(glob.glob(os.path.join(self._botdir, 'logs', '*.log'))):
      self._read_log(i)
    self._proc = None
    return exit_code

  def poll(self):
    """Polls the process to know if it exited."""
    if self._proc:
      self._proc.poll()

  def wait(self, timeout=None):
    """Waits for the process to normally exit."""
    if self._proc:
      return self._proc.wait(timeout)

  def kill(self):
    """Kills the child forcibly."""
    if self._proc:
      self._proc.kill()

  def dump_log(self):
    """Prints dev_appserver log to stderr, works only if app is stopped."""
    print >> sys.stderr, '-' * 60
    print >> sys.stderr, 'swarming_bot log'
    print >> sys.stderr, '-' * 60
    if not self._logs:
      print >> sys.stderr, '<N/A>'
    else:
      for name, content in sorted(self._logs.iteritems()):
        sys.stderr.write(name + ':\n')
        for l in content.strip('\n').splitlines():
          sys.stderr.write('  %s\n' % l)
    print >> sys.stderr, '-' * 60

  def _read_log(self, path):
    try:
      with open(path, 'rb') as f:
        self._logs[os.path.basename(path)] = f.read()
    except (IOError, OSError):
      pass


def main():
  parser = argparse.ArgumentParser(description=sys.modules[__name__].__doc__)
  parser.add_argument('server', help='Swarming server to connect bot to.')
  args = parser.parse_args()
  fix_encoding.fix_encoding()
  botdir = tempfile.mkdtemp(prefix='start_bot')
  try:
    bot = LocalBot(args.server, False, botdir)
    try:
      bot.start()
      bot.wait()
      bot.dump_log()
    except KeyboardInterrupt:
      print >> sys.stderr, '<Ctrl-C> received; stopping bot'
    finally:
      exit_code = bot.stop()
  finally:
    shutil.rmtree(botdir)
  return exit_code


if __name__ == '__main__':
  sys.exit(main())
