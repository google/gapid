#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import copy
import datetime
import json
import logging
import os
import sys
import tempfile
import textwrap
import threading
import time
import unittest
import zipfile

import test_env_bot_code
test_env_bot_code.setup_test_env()

# Creates a server mock for functions in net.py.
import net_utils

import bot_main
import remote_client
from api import bot
from api import os_utilities
from api.platforms import gce
from depot_tools import fix_encoding
from utils import file_path
from utils import logging_utils
from utils import net
from utils import subprocess42
from utils import zip_package


# pylint: disable=no-self-argument


class FakeThreadingEvent(object):
  def is_set(self):
    return False

  def wait(self, timeout=None):
    pass


class TestBotBase(net_utils.TestCase):
  def setUp(self):
    super(TestBotBase, self).setUp()
    # Throw away all swarming environ if running the test on Swarming. It may
    # interfere with the test.
    for k in os.environ.keys():
      if k.startswith('SWARMING_'):
        os.environ.pop(k)
    self.root_dir = tempfile.mkdtemp(prefix='bot_main')
    self.old_cwd = os.getcwd()
    os.chdir(self.root_dir)
    self.url = 'https://localhost:1'
    self.attributes = {
      'dimensions': {
        'foo': ['bar'],
        'id': ['localhost'],
        'pool': ['default'],
      },
      'state': {
        'bot_group_cfg_version': None,
        'cost_usd_hour': 3600.,
        'sleep_streak': 0,
      },
      'version': '123',
    }
    self.bot = self.make_bot()

  def tearDown(self):
    os.chdir(self.old_cwd)
    file_path.rmtree(self.root_dir)
    super(TestBotBase, self).tearDown()

  def make_bot(self, auth_headers_cb=None):
    return bot.Bot(
        remote_client.createRemoteClient('https://localhost:1',
                                         auth_headers_cb,
                                         'localhost',
                                         self.root_dir,
                                         False),
        copy.deepcopy(self.attributes), 'https://localhost:1', 'version1',
        unicode(self.root_dir), self.fail)


class TestBotMain(TestBotBase):
  maxDiff = 2000

  def setUp(self):
    super(TestBotMain, self).setUp()
    # __main__ does it for us.
    os.mkdir('logs')
    self.mock(zip_package, 'generate_version', lambda: '123')
    self.mock(self.bot, 'post_error', self.fail)
    self.mock(os_utilities, 'host_reboot', self.fail)
    self.mock(subprocess42, 'call', self.fail)
    self.mock(time, 'time', lambda: 100.)
    self.mock(remote_client, 'make_appengine_id', lambda *a: 42)
    config_path = os.path.join(
        test_env_bot_code.BOT_DIR, 'config', 'config.json')
    with open(config_path, 'rb') as f:
      config = json.load(f)
    self.mock(bot_main, 'get_config', lambda: config)
    self.mock(bot_main, '_bot_restart', self.fail)
    self.mock(
        bot_main, 'THIS_FILE',
        unicode(os.path.join(test_env_bot_code.BOT_DIR, 'swarming_bot.zip')))
    # Need to disable this otherwise it'd kill the current checkout.
    self.mock(bot_main, '_cleanup_bot_directory', lambda _: None)
    # Test results shouldn't depend on where they run. And they should not use
    # real GCE tokens.
    self.mock(gce, 'is_gce', lambda: False)
    self.mock(
        gce, 'oauth2_access_token_with_expiration',
        lambda *_args, **_kwargs: ('fake-access-token', 0))
    # Ensures the global state is reset after each test case.
    self.mock(bot_main, '_BOT_CONFIG', None)
    self.mock(bot_main, '_EXTRA_BOT_CONFIG', None)
    self.mock(bot_main, '_QUARANTINED', None)
    self.mock(bot_main, 'SINGLETON', None)

  def print_err_and_fail(self, _bot, msg, _task_id):
    print msg
    self.fail('post_error_task was called')

  def test_hook_restart(self):
    from config import bot_config
    obj = self.make_bot()
    def get_dimensions(botobj):
      self.assertEqual(obj, botobj)
      obj.bot_restart('Yo')
      return {u'id': [u'foo'], u'pool': [u'bar']}
    self.mock(bot_config, 'get_dimensions', get_dimensions)
    restarts = []
    self.mock(bot_main, '_bot_restart', lambda *args: restarts.append(args))
    expected = {
      u'id': [u'foo'], u'pool': [u'bar'], u'server_version': [u'version1']}
    self.assertEqual(expected, bot_main._get_dimensions(obj))
    self.assertEqual('Yo', obj.bot_restart_msg())
    self.assertEqual([(obj, 'Yo')], restarts)

  def test_get_dimensions(self):
    from config import bot_config
    obj = self.make_bot()
    def get_dimensions(botobj):
      self.assertEqual(obj, botobj)
      return {u'yo': [u'dawh']}
    self.mock(bot_config, 'get_dimensions', get_dimensions)
    expected = {u'server_version': [u'version1'], u'yo': [u'dawh']}
    self.assertEqual(expected, bot_main._get_dimensions(obj))

  def test_get_dimensions_extra(self):
    from config import bot_config
    obj = self.make_bot()
    def get_dimensions(botobj):
      self.assertEqual(obj, botobj)
      return {u'yo': [u'dawh']}
    self.mock(bot_config, 'get_dimensions', get_dimensions)

    # The extra version takes priority.
    class extra(object):
      def get_dimensions(self2, botobj): # pylint: disable=no-self-argument
        self.assertEqual(obj, botobj)
        return {u'alternative': [u'truth']}
    self.mock(bot_main, '_EXTRA_BOT_CONFIG', extra())
    expected = {u'alternative': [u'truth'], u'server_version': [u'version1']}
    self.assertEqual(expected, bot_main._get_dimensions(obj))

  def test_generate_version(self):
    self.assertEqual('123', bot_main.generate_version())

  def test_get_state(self):
    from config import bot_config
    obj = self.make_bot()
    def get_state(botobj):
      self.assertEqual(obj, botobj)
      return {'yo': 'dawh'}
    self.mock(bot_config, 'get_state', get_state)
    expected = {'sleep_streak': 0.1, 'yo': 'dawh'}
    self.assertEqual(expected, bot_main._get_state(obj, 0.1))

  def test_get_state_quarantine(self):
    botobj = bot_main.get_bot(bot_main.get_config())
    root = u'c:\\' if sys.platform == 'win32' else u'/'
    def get_state(_):
      return {
        u'disks': {
          root: {
            u'free_mb': 0.1,
            u'size_mb': 1000,
          },
          botobj.base_dir: {
            u'free_mb': 0.1,
            u'size_mb': 1000,
          },
        },
      }
    # This uses the default get_settings() values. The threshold used is
    # dependent on these values. This affects the error message below.
    # 'size' == 4096Mb
    # 'max_percent' == 15% * 1000Mb = 150Mb
    # 'min_percent' == 5% of 1000Mb == 50Mb
    # 'max_percent' is chosen.
    from config import bot_config
    self.mock(bot_config, 'get_state', get_state)
    expected = {
      u'disks': {
        u'c:\\' if sys.platform == 'win32' else u'/': {
          u'free_mb': 0.1,
          u'size_mb': 1000,
        },
        botobj.base_dir: {
          u'free_mb': 0.1,
          u'size_mb': 1000,
        },
      },
      u'quarantined':
        (u'Not enough free disk space on %s. 0.1mib < 100.0mib\n'
        u'Not enough free disk space on %s. 0.1mib < 150.0mib') %
        (root, botobj.base_dir),
      u'sleep_streak': 1,
    }
    self.assertEqual(expected, bot_main._get_state(botobj, 1))

  def test_get_state_quarantine_sticky(self):
    # A crash in get_dimensions() causes sticky quarantine in get_state.
    from config import bot_config
    obj = self.make_bot()
    def get_dimensions(botobj):
      self.assertEqual(obj, botobj)
      return 'invalid'
    self.mock(bot_config, 'get_dimensions', get_dimensions)
    def get_dimensions_os():
      return {u'os': [u'safe']}
    self.mock(os_utilities, 'get_dimensions', get_dimensions_os)
    def get_state(botobj):
      self.assertEqual(obj, botobj)
      return {'yo': 'dawh'}
    self.mock(bot_config, 'get_state', get_state)

    expected = {
      u'os': [u'safe'],
      u'quarantined': [u'1'],
      u'server_version': [u'version1'],
    }
    self.assertEqual(expected, bot_main._get_dimensions(obj))
    expected = {
      'quarantined': "get_dimensions(): expected a dict, got 'invalid'",
      'sleep_streak': 0.1,
      'yo': 'dawh',
    }
    self.assertEqual(expected, bot_main._get_state(obj, 0.1))

  def test_get_disks_quarantine_empty(self):
    root = 'c:\\' if sys.platform == 'win32' else '/'
    disks = {
      self.bot.base_dir: {
        'free_mb': 0,
        'size_mb': 0,
      },
      root: {
        'free_mb': 0,
        'size_mb': 0,
      },
    }
    expected = (
      u'Not enough free disk space on %s. 0.0mib < 1024.0mib\n'
      u'Not enough free disk space on %s. 0.0mib < 4096.0mib') % (
          root, self.bot.base_dir)
    self.assertEqual(expected, bot_main._get_disks_quarantine(self.bot, disks))

  def test_get_disks_quarantine(self):
    root = 'c:\\' if sys.platform == 'win32' else '/'
    disks = {
      self.bot.base_dir: {
        'free_mb': 4096,
        'size_mb': 4096,
      },
      root: {
        'free_mb': 4096,
        'size_mb': 4096,
      },
    }
    expected = None
    self.assertEqual(expected, bot_main._get_disks_quarantine(self.bot, disks))

  def test_default_settings(self):
    # If this trigger, you either forgot to update bot_main.py or bot_config.py.
    from config import bot_config
    self.assertEqual(bot_main.DEFAULT_SETTINGS, bot_config.get_settings(None))

  def test_min_free_disk(self):
    # size_mb, size, min_percent, max_percent, expected
    data = [
      (0, 0, 0, 0, 0),
      # 1GB*10% = 100Mb
      (1000, 1000, 10., 20., 104857600),
      # size is between min_percent (104857600) and max_percent (209715200)
      (1000, 150000000, 10., 20., 150000000),
      # 1GB*20% = 200Mb
      (1000, 300000000, 10., 20., 209715200),
      # No max_percent, so use size
      (1000, 300000000, 10., 0, 300000000),
    ]
    for size_mb, size, minp, maxp, expected in data:
      infos = {'size_mb': size_mb}
      settings = {'size': size, 'min_percent': minp, 'max_percent': maxp}
      actual = bot_main._min_free_disk(infos, settings)
      self.assertEqual(expected, actual)

  def test_dict_deep_merge(self):
    a = {
      'a': {
        'a': 1,
        'b': 2,
      },
    }
    b = {
      'a': {
        'b': 3,
        'c': 4,
      },
    }
    expected = {
      'a': {
        'a': 1,
        'b': 3,
        'c': 4,
      },
    }
    self.assertEqual(expected, bot_main._dict_deep_merge(a, b))
    self.assertEqual(a, bot_main._dict_deep_merge(a, None))
    self.assertEqual(a, bot_main._dict_deep_merge(None, a))

  def test_setup_bot(self):
    setup_bots = []
    def setup_bot(_bot):
      setup_bots.append(1)
      return False
    from config import bot_config
    self.mock(bot_config, 'setup_bot', setup_bot)
    self.mock(bot, '_make_stack', lambda: 'fake stack')
    restarts = []
    post_event = []
    self.mock(
        os_utilities, 'host_reboot', lambda *a, **kw: restarts.append((a, kw)))
    self.mock(
        bot.Bot, 'post_event', lambda *a, **kw: post_event.append((a, kw)))
    self.expected_requests([])
    bot_main.setup_bot(False)
    expected = [
      (('Starting new swarming bot: %s' % bot_main.THIS_FILE,),
        {'timeout': 900}),
    ]
    self.assertEqual(expected, restarts)
    # It is called twice, one as part of setup_bot(False), another as part of
    # on_shutdown_hook().
    self.assertEqual([1, 1], setup_bots)
    expected = [
      'Starting new swarming bot: %s' % bot_main.THIS_FILE,
      ('Host is stuck rebooting for: Starting new swarming bot: %s\n'
       'Calling stack:\nfake stack') % bot_main.THIS_FILE,
    ]
    self.assertEqual(expected, [i[0][2] for i in post_event])

  def test_post_error_task(self):
    self.mock(time, 'time', lambda: 126.0)
    self.mock(logging, 'error', lambda *_, **_kw: None)
    self.mock(
        bot_main, 'get_config',
        lambda: {'server': self.url, 'server_version': '1'})
    expected_attribs = bot_main.get_attributes(None)
    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/task_error/23',
            {
              'data': {
                'id': expected_attribs['dimensions']['id'][0],
                'message': 'error',
                'task_id': 23,
              },
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {'resp': 1},
          ),
        ])
    botobj = bot_main.get_bot(bot_main.get_config())
    self.assertEqual(True, bot_main._post_error_task(botobj, 'error', 23))

  def test_do_handshake(self):
    # Ensures the injected code was called. Ensures the injected name is
    # 'injected', that it can imports the base one.
    quit_bit = threading.Event()
    obj = self.make_bot()

    # Hack into bot_config.
    bot_config = bot_main._get_bot_config()
    bot_config.base_func = lambda: 'yo'
    try:
      def do_handshake(attributes):
        return {
          'bot_version': attributes['version'],
          'bot_group_cfg_version': None,
          'bot_group_cfg': None,
          'bot_config': textwrap.dedent("""
              from config import bot_config
              def get_dimensions(_):
                return {
                  'alternative': __name__,
                  'bot_config': bot_config.__file__,
                  'called': bot_config.base_func(),
                }
              """),
        }
      self.mock(obj.remote, 'do_handshake', do_handshake)
      bot_main._do_handshake(obj, quit_bit)
      self.assertFalse(quit_bit.is_set())
      self.assertEqual(None, obj.bot_restart_msg())
      expected = {
        'alternative': 'injected',
        'bot_config': bot_config.__file__,
        'called': 'yo',
      }
      self.assertEqual(expected, bot_main._EXTRA_BOT_CONFIG.get_dimensions(obj))
    finally:
      del bot_config.base_func

  def test_call_hook_both(self):
    # Both hooks must be called.
    first = threading.Event()
    second = threading.Event()
    from config import bot_config
    obj = self.make_bot()
    def on_bot_shutdown_1(botobj):
      self.assertEqual(obj, botobj)
      first.set()
    self.mock(bot_config, 'on_bot_shutdown', on_bot_shutdown_1)

    class extra(object):
      def on_bot_shutdown(self2, botobj): # pylint: disable=no-self-argument
        self.assertEqual(obj, botobj)
        second.set()
    self.mock(bot_main, '_EXTRA_BOT_CONFIG', extra())
    bot_main._call_hook(True, obj, 'on_bot_shutdown')
    self.assertTrue(first.is_set())
    self.assertTrue(second.is_set())

  def test_run_bot(self):
    self.mock(threading, 'Event', FakeThreadingEvent)

    # Test the run_bot() loop. Does not use self.bot.
    self.mock(time, 'time', lambda: 126.0)
    class Foo(Exception):
      pass

    def poll_server(botobj, _quit_bit, _last_action):
      sleep_streak = botobj.state['sleep_streak']
      self.assertEqual(self.url, botobj.server)
      if sleep_streak == 5:
        raise Exception('Jumping out of the loop')
      return False
    self.mock(bot_main, '_poll_server', poll_server)

    def post_error(botobj, e):
      self.assertEqual(self.url, botobj.server)
      lines = e.splitlines()
      self.assertEqual('Jumping out of the loop', lines[0])
      self.assertEqual('Traceback (most recent call last):', lines[1])
      raise Foo('Necessary to get out of the loop')
    self.mock(bot.Bot, 'post_error', post_error)

    orig = bot_main.get_bot
    botobj = [None]
    def get_bot(config):
      botobj[0] = orig(config)
      return botobj[0]
    self.mock(bot_main, 'get_bot', get_bot)

    self.mock(
        bot_main, 'get_config',
        lambda: {
          'server': self.url,
          'server_version': '1',
        })
    self.mock(
        bot_main, '_get_dimensions', lambda _: self.attributes['dimensions'])
    self.mock(os_utilities, 'get_state', lambda *_: self.attributes['state'])

    # pylint: disable=unused-argument
    class Popen(object):
      def __init__(
          self2, cmd, detached, cwd, stdout, stderr, stdin, close_fds):
        self2.returncode = None
        expected = [sys.executable, bot_main.THIS_FILE, 'run_isolated']
        self.assertEqual(expected, cmd[:len(expected)])
        self.assertEqual(True, detached)
        self.assertEqual(subprocess42.PIPE, stdout)
        self.assertEqual(subprocess42.STDOUT, stderr)
        self.assertEqual(subprocess42.PIPE, stdin)
        self.assertEqual(sys.platform != 'win32', close_fds)

      def communicate(self2, i):
        self.assertEqual(None, i)
        self2.returncode = 0
        return '', None
    self.mock(subprocess42, 'Popen', Popen)

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/server_ping',
            {}, 'foo', None,
          ),
          (
            'https://localhost:1/swarming/api/v1/bot/handshake',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            None, # fails, gets retried
          ),
          (
            'https://localhost:1/swarming/api/v1/bot/handshake',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'bot_version': '123',
              'server': self.url,
              'server_version': 1,
              'bot_group_cfg_version': 'abc:def',
              'bot_group_cfg': {
                'dimensions': {'bot_side': ['A']},
              },
            },
          ),
        ])

    with self.assertRaises(Foo):
      bot_main._run_bot(None)
    self.assertEqual(
        self.attributes['dimensions']['id'][0], os.environ['SWARMING_BOT_ID'])

    self.assertEqual({
      'bot_side': ['A'],
      'foo': ['bar'],
      'id': ['localhost'],
      'pool': ['default'],
    }, botobj[0].dimensions)

  def test_poll_server_sleep(self):
    slept = []
    bit = threading.Event()
    self.mock(bit, 'wait', slept.append)
    self.mock(bot_main, '_run_manifest', self.fail)
    self.mock(bot_main, '_update_bot', self.fail)
    from config import bot_config
    called = []
    self.mock(bot_config, 'on_bot_idle', lambda _bot, _s: called.append(1))

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/poll',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'cmd': 'sleep',
              'duration': 1.24,
            },
          ),
        ])
    self.assertFalse(bot_main._poll_server(self.bot, bit, 2))
    self.assertEqual([1.24], slept)
    self.assertEqual([1], called)

  def test_poll_server_sleep_with_auth(self):
    slept = []
    bit = threading.Event()
    self.mock(bit, 'wait', slept.append)
    self.mock(bot_main, '_run_manifest', self.fail)
    self.mock(bot_main, '_update_bot', self.fail)

    self.bot = self.make_bot(lambda: ({'A': 'a'}, time.time() + 3600))

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/poll',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'A': 'a', 'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'cmd': 'sleep',
              'duration': 1.24,
            },
          ),
        ])
    self.assertFalse(bot_main._poll_server(self.bot, bit, 0))
    self.assertEqual([1.24], slept)

  def test_poll_server_run(self):
    manifest = []
    clean = []
    bit = threading.Event()
    self.mock(bit, 'wait', self.fail)
    self.mock(bot_main, '_run_manifest', lambda *args: manifest.append(args))
    self.mock(bot_main, '_clean_cache',
              lambda *args: clean.append(args))
    self.mock(bot_main, '_update_bot', self.fail)

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/poll',
            {
              'data': self.bot._attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'cmd': 'run',
              'manifest': {'foo': 'bar'},
            },
          ),
        ])
    self.assertTrue(bot_main._poll_server(self.bot, bit, 0))
    expected = [(self.bot, {'foo': 'bar'}, time.time())]
    self.assertEqual(expected, manifest)
    expected = [(self.bot,)]
    self.assertEqual(expected, clean)
    self.assertEqual(None, self.bot.bot_restart_msg())

  def test_poll_server_update(self):
    update = []
    bit = threading.Event()
    self.mock(bit, 'wait', self.fail)
    self.mock(bot_main, '_run_manifest', self.fail)
    self.mock(bot_main, '_update_bot', lambda *args: update.append(args))

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/poll',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'cmd': 'update',
              'version': '123',
            },
          ),
        ])
    self.assertTrue(bot_main._poll_server(self.bot, bit, 0))
    self.assertEqual([(self.bot, '123')], update)
    self.assertEqual(None, self.bot.bot_restart_msg())

  def test_poll_server_restart(self):
    restarts = []
    bit = threading.Event()
    self.mock(bit, 'wait', self.fail)
    self.mock(bot_main, '_run_manifest', self.fail)
    self.mock(bot_main, '_update_bot', self.fail)
    self.mock(self.bot, 'host_reboot', self.fail)
    self.mock(bot_main, '_bot_restart', lambda obj, x: restarts.append(x))

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/poll',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'cmd': 'bot_restart',
              'message': 'Please restart now',
            },
          ),
        ])
    self.assertTrue(bot_main._poll_server(self.bot, bit, 0))
    self.assertEqual(['Please restart now'], restarts)
    self.assertEqual(None, self.bot.bot_restart_msg())

  def test_poll_server_reboot(self):
    reboots = []
    bit = threading.Event()
    self.mock(bit, 'wait', self.fail)
    self.mock(bot_main, '_run_manifest', self.fail)
    self.mock(bot_main, '_update_bot', self.fail)
    self.mock(self.bot, 'host_reboot', lambda *args: reboots.append(args))

    self.expected_requests(
        [
          (
            'https://localhost:1/swarming/api/v1/bot/poll',
            {
              'data': self.attributes,
              'follow_redirects': False,
              'headers': {'Cookie': 'GOOGAPPUID=42'},
              'timeout': remote_client.NET_CONNECTION_TIMEOUT_SEC,
            },
            {
              'cmd': 'host_reboot',
              'message': 'Please die now',
            },
          ),
        ])
    self.assertTrue(bot_main._poll_server(self.bot, bit, 0))
    self.assertEqual([('Please die now',)], reboots)
    self.assertEqual(None, self.bot.bot_restart_msg())

  def _mock_popen(
      self, returncode=0, exit_code=0, url='https://localhost:1',
      expected_auth_params_json=None):
    result = {
      'exit_code': exit_code,
      'must_signal_internal_failure': None,
      'version': 3,
    }
    # Method should have "self" as first argument - pylint: disable=E0213
    class Popen(object):
      def __init__(
          self2, cmd, detached, cwd, env, stdout, stderr, stdin, close_fds):
        self2.returncode = None
        self2._out_file = os.path.join(
            self.root_dir, 'w', 'task_runner_out.json')
        cmd = cmd[:]
        expected = [
          sys.executable, bot_main.THIS_FILE, 'task_runner',
          '--swarming-server', url,
          '--in-file',
          os.path.join(self.root_dir, 'w', 'task_runner_in.json'),
          '--out-file', self2._out_file,
          '--cost-usd-hour', '3600.0', '--start', '100.0',
          '--bot-file',
        ]
        # After than there may be --bot-file and --auth-params-file. Then --
        # will be used to mark the separation of flags meant to be sent to
        # run_isolated.
        self.assertEqual(cmd[:len(expected)], expected)
        del cmd[:len(expected)]
        self.assertTrue(cmd.pop(0).endswith('.json'))
        if expected_auth_params_json:
          auth_params_file = os.path.join(
              self.root_dir, 'w', 'bot_auth_params.json')
          with open(auth_params_file, 'rb') as f:
            actual_auth_params = json.load(f)
          self.assertEqual(expected_auth_params_json, actual_auth_params)
          self.assertEqual(cmd[:2], ['--auth-params-file', auth_params_file])
        self.assertEqual(True, detached)
        self.assertEqual(self.bot.base_dir, cwd)
        self.assertEqual('24', env['SWARMING_TASK_ID'])
        self.assertTrue(stdout)
        self.assertEqual(subprocess42.STDOUT, stderr)
        self.assertEqual(subprocess42.PIPE, stdin)
        self.assertEqual(sys.platform != 'win32', close_fds)

      def wait(self2, timeout=None): # pylint: disable=unused-argument
        self2.returncode = returncode
        with open(self2._out_file, 'wb') as f:
          json.dump(result, f)
        return 0

    self.mock(subprocess42, 'Popen', Popen)
    return result

  def test_run_manifest(self):
    self.mock(bot_main, '_post_error_task', self.print_err_and_fail)
    def call_hook(botobj, name, *args):
      if name == 'on_after_task':
        failure, internal_failure, dimensions, summary = args
        self.assertEqual(self.attributes['dimensions'], botobj.dimensions)
        self.assertEqual(False, failure)
        self.assertEqual(False, internal_failure)
        self.assertEqual({'os': 'Amiga', 'pool': 'default'}, dimensions)
        self.assertEqual(result, summary)
    self.mock(bot_main, '_call_hook', call_hook)
    result = self._mock_popen(url='https://localhost:3')

    manifest = {
      'command': ['echo', 'hi'],
      'dimensions': {'os': 'Amiga', 'pool': 'default'},
      'grace_period': 30,
      'hard_timeout': 60,
      'io_timeout': None,
      'host': 'https://localhost:3',
      'task_id': '24',
    }
    self.assertEqual(self.root_dir, self.bot.base_dir)
    bot_main._run_manifest(self.bot, manifest, time.time())

  def test_run_manifest_with_auth_headers(self):
    self.bot = self.make_bot(
        auth_headers_cb=lambda: ({'A': 'a'}, time.time() + 3600))

    self.mock(bot_main, '_post_error_task', self.print_err_and_fail)
    def call_hook(botobj, name, *args):
      if name == 'on_after_task':
        failure, internal_failure, dimensions, summary = args
        self.assertEqual(self.attributes['dimensions'], botobj.dimensions)
        self.assertEqual(False, failure)
        self.assertEqual(False, internal_failure)
        self.assertEqual({'os': 'Amiga', 'pool': 'default'}, dimensions)
        self.assertEqual(result, summary)
    self.mock(bot_main, '_call_hook', call_hook)
    result = self._mock_popen(
        url='https://localhost:3',
        expected_auth_params_json={
          'bot_id': 'localhost',
          'task_id': '24',
          'swarming_http_headers': {'A': 'a'},
          'swarming_http_headers_exp': int(time.time() + 3600),
          'bot_service_account': 'none',
          'system_service_account': 'robot@example.com',  # as in task manifest
          'task_service_account': 'bot',
        })

    manifest = {
      'command': ['echo', 'hi'],
      'dimensions': {'os': 'Amiga', 'pool': 'default'},
      'grace_period': 30,
      'hard_timeout': 60,
      'io_timeout': None,
      'host': 'https://localhost:3',
      'service_accounts': {
        'system': {'service_account': 'robot@example.com'},
        'task': {'service_account': 'bot'},
      },
      'task_id': '24',
    }
    self.assertEqual(self.root_dir, self.bot.base_dir)
    bot_main._run_manifest(self.bot, manifest, time.time())

  def test_run_manifest_task_failure(self):
    self.mock(bot_main, '_post_error_task', self.print_err_and_fail)
    def call_hook(_botobj, name, *args):
      if name == 'on_after_task':
        failure, internal_failure, dimensions, summary = args
        self.assertEqual(True, failure)
        self.assertEqual(False, internal_failure)
        self.assertEqual({'pool': 'default'}, dimensions)
        self.assertEqual(result, summary)
    self.mock(bot_main, '_call_hook', call_hook)
    result = self._mock_popen(exit_code=1)

    manifest = {
      'command': ['echo', 'hi'],
      'dimensions': {'pool': 'default'},
      'grace_period': 30,
      'hard_timeout': 60,
      'io_timeout': 60,
      'task_id': '24',
    }
    bot_main._run_manifest(self.bot, manifest, time.time())

  def test_run_manifest_internal_failure(self):
    posted = []
    self.mock(bot_main, '_post_error_task', lambda *args: posted.append(args))
    def call_hook(_botobj, name, *args):
      if name == 'on_after_task':
        failure, internal_failure, dimensions, summary = args
        self.assertEqual(False, failure)
        self.assertEqual(True, internal_failure)
        self.assertEqual({'pool': 'default'}, dimensions)
        self.assertEqual(result, summary)
    self.mock(bot_main, '_call_hook', call_hook)
    result = self._mock_popen(returncode=1)

    manifest = {
      'command': ['echo', 'hi'],
      'dimensions': {'pool': 'default'},
      'grace_period': 30,
      'hard_timeout': 60,
      'io_timeout': 60,
      'task_id': '24',
    }
    bot_main._run_manifest(self.bot, manifest, time.time())
    expected = [(self.bot, 'Execution failed: internal error (1).', '24')]
    self.assertEqual(expected, posted)

  def test_run_manifest_exception(self):
    posted = []
    def post_error_task(botobj, msg, task_id):
      posted.append((botobj, msg.splitlines()[0], task_id))
    self.mock(bot_main, '_post_error_task', post_error_task)
    def call_hook(_botobj, name, *args):
      if name == 'on_after_task':
        failure, internal_failure, dimensions, summary = args
        self.assertEqual(False, failure)
        self.assertEqual(True, internal_failure)
        self.assertEqual({'pool': 'default'}, dimensions)
        self.assertEqual({}, summary)
    self.mock(bot_main, '_call_hook', call_hook)
    def raiseOSError(*_a, **_k):
      raise OSError('Dang')
    self.mock(subprocess42, 'Popen', raiseOSError)

    manifest = {
      'command': ['echo', 'hi'],
      'dimensions': {'pool': 'default'},
      'grace_period': 30,
      'hard_timeout': 60,
      'io_timeout': None,
      'task_id': '24',
    }
    bot_main._run_manifest(self.bot, manifest, time.time())
    expected = [(self.bot, 'Internal exception occured: Dang', '24')]
    self.assertEqual(expected, posted)

  def test_update_bot(self):
    restarts = []
    def bot_restart(_botobj, message, filepath):
      self.assertEqual('Updating to 123', message)
      self.assertEqual(new_zip, filepath)
      restarts.append(1)
    self.mock(bot_main, '_bot_restart', bot_restart)
    # Mock the file to download in the temporary directory.
    self.mock(
        bot_main, 'THIS_FILE',
        unicode(os.path.join(self.root_dir, 'swarming_bot.1.zip')))
    new_zip = os.path.join(self.root_dir, 'swarming_bot.2.zip')
    # This is necessary otherwise zipfile will crash.
    self.mock(time, 'time', lambda: 1400000000)
    def url_retrieve(f, url, headers=None, timeout=None):
      self.assertEqual(
          'https://localhost:1/swarming/api/v1/bot/bot_code'
          '/123?bot_id=localhost', url)
      self.assertEqual(new_zip, f)
      self.assertEqual({'Cookie': 'GOOGAPPUID=42'}, headers)
      self.assertEqual(remote_client.NET_CONNECTION_TIMEOUT_SEC, timeout)
      # Create a valid zip that runs properly.
      with zipfile.ZipFile(f, 'w') as z:
        z.writestr('__main__.py', 'print("hi")')
      return True
    self.mock(net, 'url_retrieve', url_retrieve)
    bot_main._update_bot(self.bot, '123')
    self.assertEqual([1], restarts)

  def test_main(self):
    def check(x):
      self.assertEqual(logging.WARNING, x)
    self.mock(logging_utils, 'set_console_level', check)

    def run_bot(error):
      self.assertEqual(None, error)
      return 0
    self.mock(bot_main, '_run_bot', run_bot)

    class Singleton(object):
      # pylint: disable=no-self-argument
      def acquire(self2):
        return True
      def release(self2):
        self.fail()
    self.mock(bot_main, 'SINGLETON', Singleton())

    self.assertEqual(0, bot_main.main([]))

  def test_update_lkgbc(self):
    # Create LKGBC with a timestamp from 1h ago.
    lkgbc = os.path.join(self.bot.base_dir, 'swarming_bot.zip')
    with open(lkgbc, 'wb') as f:
      f.write('a')
    past = time.time() - 60*60
    os.utime(lkgbc, (past, past))

    cur = os.path.join(self.bot.base_dir, 'swarming_bot.1.zip')
    with open(cur, 'wb') as f:
      f.write('ab')
    self.mock(bot_main, 'THIS_FILE', cur)

    self.assertEqual(True, bot_main._update_lkgbc(self.bot))
    with open(lkgbc, 'rb') as f:
      self.assertEqual('ab', f.read())

  def test_maybe_update_lkgbc(self):
    # Create LKGBC with a timestamp from 1h ago.
    lkgbc = os.path.join(self.bot.base_dir, 'swarming_bot.zip')
    with open(lkgbc, 'wb') as f:
      f.write('a')
    past = time.time() - 60*60
    os.utime(lkgbc, (past, past))

    cur = os.path.join(self.bot.base_dir, 'swarming_bot.1.zip')
    with open(cur, 'wb') as f:
      f.write('ab')
    self.mock(bot_main, 'THIS_FILE', cur)

    # No update even if they mismatch, LKGBC is not old enough.
    self.assertEqual(False, bot_main._maybe_update_lkgbc(self.bot))
    with open(lkgbc, 'rb') as f:
      self.assertEqual('a', f.read())

    # Fast forward a little more than 7 days.
    now = time.time()
    self.mock(time, 'time', lambda: now + 7*24*60*60+10)
    self.assertEqual(True, bot_main._maybe_update_lkgbc(self.bot))
    with open(lkgbc, 'rb') as f:
      self.assertEqual('ab', f.read())


class TestBotNotMocked(TestBotBase):
  def test_bot_restart(self):
    calls = []
    def exec_python(args):
      calls.append(args)
      return 23
    self.mock(bot_main.common, 'exec_python', exec_python)
    # pylint: disable=unused-argument
    class Popen(object):
      def __init__(self2, cmd, stdout, stderr):
        self2.returncode = None
        expected = [sys.executable, bot_main.THIS_FILE, 'is_fine']
        self.assertEqual(expected, cmd)
        self.assertEqual(subprocess42.PIPE, stdout)
        self.assertEqual(subprocess42.STDOUT, stderr)

      def communicate(self2):
        self2.returncode = 0
        return '', None
    self.mock(subprocess42, 'Popen', Popen)

    with self.assertRaises(SystemExit) as e:
      bot_main._bot_restart(self.bot, 'Yo', bot_main.THIS_FILE)
    self.assertEqual(23, e.exception.code)

    self.assertEqual([[bot_main.THIS_FILE, 'start_slave', '--survive']], calls)


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  if '-v' in sys.argv:
    TestBotMain.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
