#!/usr/bin/env python
# coding=utf-8
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Integration test for the Swarming server, Swarming bot and Swarming client.

It starts a Swarming server, Isolate server and a Swarming bot, then triggers
tasks with the Swarming client to ensure the system works end to end.
"""

import base64
import contextlib
import json
import logging
import os
import re
import signal
import socket
import sys
import tempfile
import textwrap
import time
import unittest
import urllib

APP_DIR = os.path.dirname(os.path.abspath(__file__))
BOT_DIR = os.path.join(APP_DIR, 'swarming_bot')
CLIENT_DIR = os.path.join(APP_DIR, '..', '..', 'client')

from tools import start_bot
from tools import start_servers

sys.path.insert(0, CLIENT_DIR)
from third_party.depot_tools import fix_encoding
from utils import fs
from utils import file_path
from utils import large
from utils import subprocess42
sys.path.pop(0)

sys.path.insert(0, BOT_DIR)
import test_env_bot
test_env_bot.setup_test_env()

from api import os_utilities


# Use a variable because it is corrupting my text editor from the 80s.
# One important thing to note is that this character U+1F310 is not in the BMP
# so it tests more accurately emojis like 💣 (U+1F4A3) and 💩(U+1F4A9).
HELLO_WORLD = u'hello_🌐'


# Signal as seen by the process.
SIGNAL_TERM = -1073741510 if sys.platform == 'win32' else -signal.SIGTERM


# Timeout to wait for the operations, so that the smoke test doesn't hang
# indefinitely.
TIMEOUT_SECS = 20


# The default isolated command is to map and run HELLO_WORLD.
DEFAULT_ISOLATE_HELLO = """{
  "variables": {
    "command": ["python", "-u", "%(name)s.py"],
    "files": ["%(name)s.py"],
  },
}""" % {'name': HELLO_WORLD.encode('utf-8')}


def _script(content):
  """Deindents and encode to utf-8."""
  return textwrap.dedent(content.encode('utf-8'))


class SwarmingClient(object):
  def __init__(self, swarming_server, isolate_server, namespace, tmpdir):
    self._swarming_server = swarming_server
    self._isolate_server = isolate_server
    self._namespace = namespace
    self._tmpdir = tmpdir
    self._index = 0

  def isolate(self, isolate_path, isolated_path):
    """Archives a .isolate file into the isolate server and returns the isolated
    hash.
    """
    args = [
      '--namespace', self._namespace,
      '-i', isolate_path,
      '-s', isolated_path,
    ]
    isolated_hash = self._capture_isolate('archive', args, '').split()[0]
    logging.debug('%s = %s', isolated_path, isolated_hash)
    return isolated_hash

  def retrieve_file(self, isolated_hash, dst):
    """Retrieves a single isolated content."""
    args = [
      '--namespace', self._namespace,
      '-f', isolated_hash, dst,
      '--cache', os.path.join(os.path.dirname(dst), 'cache'),
    ]
    return self._run_isolateserver('download', args)

  def task_trigger_raw(self, args):
    """Triggers a task and return the task id."""
    h, tmp = tempfile.mkstemp(
        dir=self._tmpdir, prefix='trigger_raw', suffix='.json')
    os.close(h)
    cmd = [
      '--user', 'joe@localhost',
      '-d', 'pool', 'default',
      '--dump-json', tmp,
      '--raw-cmd',
    ]
    cmd.extend(args)
    assert not self._run_swarming('trigger', cmd), args
    with fs.open(tmp, 'rb') as f:
      data = json.load(f)
      task_id = data['tasks'].popitem()[1]['task_id']
      logging.debug('task_id = %s', task_id)
      return task_id

  def task_trigger_isolated(self, isolated_hash, name, extra):
    """Triggers a task and return the task id."""
    h, tmp = tempfile.mkstemp(
        dir=self._tmpdir, prefix='trigger_isolated', suffix='.json')
    os.close(h)
    cmd = [
      '--user', 'joe@localhost',
      '-d', 'pool', 'default',
      '--dump-json', tmp,
      '--task-name', name,
      '-I',  self._isolate_server,
      '--namespace', self._namespace,
      '-s', isolated_hash,
    ]
    if extra:
      cmd.extend(extra)
    assert not self._run_swarming('trigger', cmd)
    with fs.open(tmp, 'rb') as f:
      data = json.load(f)
      task_id = data['tasks'].popitem()[1]['task_id']
      logging.debug('task_id = %s', task_id)
      return task_id

  def task_trigger_post(self, request):
    """Triggers a task with the very raw RPC and returns the task id.

    It does an HTTP POST at tasks/new with the provided data in 'request'.
    """
    out = self._capture_swarming('post', ['tasks/new'], request)
    try:
      data = json.loads(out)
    except ValueError:
      logging.info('Data could not be decoded: %r', out)
      return None
    task_id = data['task_id']
    logging.debug('task_id = %s', task_id)
    return task_id

  def task_collect(self, task_id, timeout=TIMEOUT_SECS):
    """Collects the results for a task.

    Returns:
      - Result summary as saved by the tool
      - output files as a dict
    """
    tmp = os.path.join(self._tmpdir, task_id + '.json')
    tmpdir = unicode(os.path.join(self._tmpdir, task_id))
    if os.path.isdir(tmpdir):
      for i in xrange(100000):
        t = '%s_%d' % (tmpdir, i)
        if not os.path.isdir(t):
          tmpdir = t
          break
    os.mkdir(tmpdir)
    # swarming.py collect will return the exit code of the task.
    args = [
      '--task-summary-json', tmp, task_id, '--task-output-dir', tmpdir,
      '--timeout', str(timeout), '--perf',
    ]
    self._run_swarming('collect', args)
    with fs.open(tmp, 'rb') as f:
      data = f.read()
    try:
      summary = json.loads(data)
    except ValueError:
      print >> sys.stderr, 'Bad json:\n%s' % data
      raise
    file_outputs = {}
    for root, _, files in fs.walk(tmpdir):
      for i in files:
        p = os.path.join(root, i)
        name = p[len(tmpdir)+1:]
        with fs.open(p, 'rb') as f:
          file_outputs[name] = f.read()
    return summary, file_outputs

  def task_cancel(self, task_id, args):
    """Cancels a task."""
    return self._capture_swarming(
        'cancel', list(args) + [str(task_id)], '') == ''

  def task_result(self, task_id):
    """Queries a task result without waiting for it to complete."""
    # collect --timeout 0 now works the same.
    return json.loads(
        self._capture_swarming('query', ['task/%s/result' % task_id], ''))

  def task_stdout(self, task_id):
    """Returns current task stdout without waiting for it to complete."""
    raw = self._capture_swarming('query', ['task/%s/stdout' % task_id], '')
    return json.loads(raw).get('output')

  def terminate(self, bot_id):
    task_id = self._capture_swarming('terminate', [bot_id], '').strip()
    logging.info('swarming.py terminate returned %r', task_id)
    if not task_id:
      return 1
    return self._run_swarming(
        'collect', ['--timeout', str(TIMEOUT_SECS), task_id])

  def query_bot(self):
    """Returns the bot's properties."""
    raw = self._capture_swarming('query', ['bots/list', '--limit', '10'], '')
    data = json.loads(raw)
    if not data.get('items'):
      return None
    assert len(data['items']) == 1
    return data['items'][0]

  def dump_log(self):
    print >> sys.stderr, '-' * 60
    print >> sys.stderr, 'Client calls'
    print >> sys.stderr, '-' * 60
    for i in xrange(self._index):
      with fs.open(os.path.join(self._tmpdir, 'client_%d.log' % i), 'rb') as f:
        log = f.read().strip('\n')
      for l in log.splitlines():
        sys.stderr.write('  %s\n' % l)

  def _run_swarming(self, command, args):
    """Runs swarming.py and capture the stdout to a log file.

    The log file will be printed by the test framework in case of failure or
    verbose mode.

    Returns:
      The process exit code.
    """
    name = os.path.join(self._tmpdir, u'client_%d.log' % self._index)
    self._index += 1
    cmd = [
      sys.executable, 'swarming.py', command, '-S', self._swarming_server,
      '--verbose',
    ] + args
    with fs.open(name, 'wb') as f:
      f.write('\nRunning: %s\n' % ' '.join(cmd))
      f.flush()
      p = subprocess42.Popen(
          cmd, stdout=f, stderr=subprocess42.STDOUT, cwd=CLIENT_DIR)
      p.communicate()
      return p.returncode

  def _run_isolateserver(self, command, args):
    """Runs isolateserver.py and capture the stdout to a log file.

    The log file will be printed by the test framework in case of failure or
    verbose mode.

    Returns:
      The process exit code.
    """
    name = os.path.join(self._tmpdir, u'client_%d.log' % self._index)
    self._index += 1
    cmd = [
      sys.executable, 'isolateserver.py', command, '-I', self._isolate_server,
      '--verbose',
    ] + args
    with fs.open(name, 'wb') as f:
      f.write('\nRunning: %s\n' % ' '.join(cmd))
      f.flush()
      p = subprocess42.Popen(
          cmd, stdout=f, stderr=subprocess42.STDOUT, cwd=CLIENT_DIR)
      p.communicate()
      return p.returncode

  def _capture_swarming(self, command, args, stdin):
    name = os.path.join(self._tmpdir, u'client_%d.log' % self._index)
    self._index += 1
    cmd = [
      sys.executable, 'swarming.py', command, '-S', self._swarming_server,
      '--log-file', name
    ] + args
    with fs.open(name, 'wb') as f:
      f.write('\nRunning: %s\n' % ' '.join(cmd))
    p = subprocess42.Popen(
        cmd, stdin=subprocess42.PIPE, stdout=subprocess42.PIPE, cwd=CLIENT_DIR)
    return p.communicate(stdin)[0]

  def _capture_isolate(self, command, args, stdin):
    name = os.path.join(self._tmpdir, u'client_%d.log' % self._index)
    self._index += 1
    cmd = [
      sys.executable, 'isolate.py', command, '-I', self._isolate_server,
      '--log-file', name
    ] + args
    with fs.open(name, 'wb') as f:
      f.write('\nRunning: %s\n' % ' '.join(cmd))
    p = subprocess42.Popen(
        cmd, stdin=subprocess42.PIPE, stdout=subprocess42.PIPE, cwd=CLIENT_DIR)
    return p.communicate(stdin)[0]


def gen_expected(**kwargs):
  expected = {
    u'bot_dimensions': None,
    u'bot_id': unicode(socket.getfqdn().split('.', 1)[0]),
    u'current_task_slice': u'0',
    u'exit_code': u'0',
    u'failure': False,
    u'internal_failure': False,
    u'name': u'',
    u'output': u'hi\n',
    u'server_versions': [u'N/A'],
    u'state': u'COMPLETED',
    u'tags': [
      u'pool:default',
      u'priority:200',
      u'service_account:none',
      u'swarming.pool.template:none',
      u'swarming.pool.version:pools_cfg_rev',
      u'user:joe@localhost',
    ],
    u'try_number': u'1',
    u'user': u'joe@localhost',
  }
  expected.update({unicode(k): v for k, v in kwargs.iteritems()})
  return expected


class Test(unittest.TestCase):
  maxDiff = None
  client = None
  servers = None
  namespace = None
  bot = None
  leak = False

  def setUp(self):
    super(Test, self).setUp()
    self.dimensions = os_utilities.get_dimensions()
    # The bot forcibly adds server_version.
    self.dimensions[u'server_version'] = [u'N/A']
    # Reset the bot's isolated cache at the start of each task, so that the
    # cache reuse data becomes deterministic. Only restart the bot when it had a
    # named cache because it takes multiple seconds to to restart the bot. :(
    #
    # TODO(maruel): 'isolated_upload' is not deterministic because the isolate
    # server not cleared.
    start = time.time()
    while True:
      old = self.client.query_bot()
      if old:
        break
      if time.time() - start > TIMEOUT_SECS:
        self.fail('Bot took too long to start')

    started_ts = json.loads(old['state'])['started_ts']
    logging.info('setUp: started_ts was %s', started_ts)
    had_cache = any(u'caches' == i['key'] for i in old['dimensions'])
    self.bot.wipe_cache(had_cache)

    # The bot restarts due to wipe_cache(True) so wait for the bot to come back
    # online. It may takes a few loop.
    start = time.time()
    while True:
      if time.time() - start > TIMEOUT_SECS:
        self.fail('Bot took too long to start after wipe_cache()')
      state = self.client.query_bot()
      if not state:
        time.sleep(0.1)
        continue
      if not had_cache:
        break
      new_started_ts = json.loads(state['state'])['started_ts']
      logging.info('setUp: new_started_ts is %s', new_started_ts)
      # This assumes that starting the bot and running the previous test case
      # took more than 1s.
      if not started_ts or new_started_ts != started_ts:
        self.assertNotIn(u'caches', state['dimensions'])
        break

  def gen_expected(self, **kwargs):
    dims = [
      {u'key': k, u'value': v} for k, v in sorted(self.dimensions.iteritems())
    ]
    return gen_expected(bot_dimensions=dims, **kwargs)

  def test_raw_bytes_and_dupe_dimensions(self):
    # A string of a letter 'A', UTF-8 BOM then UTF-16 BOM then UTF-EDBCDIC then
    # invalid UTF-8 and the letter 'B'. It is double escaped so it can be passed
    # down the shell.
    #
    # Leverage the occasion to also specify the same dimension key twice with
    # different values, and ensure it works.
    invalid_bytes = 'A\\xEF\\xBB\\xBF\\xFE\\xFF\\xDD\\x73\\x66\\x73\\xc3\\x28B'
    args = [
      '-T', 'non_utf8',
      # It's pretty much guaranteed 'os' has at least two values.
      '-d', 'os', self.dimensions['os'][0],
      '-d', 'os', self.dimensions['os'][1],
      '--',
      'python', '-u', '-c', 'print(\'' + invalid_bytes + '\')',
    ]
    summary = self.gen_expected(
        name=u'non_utf8',
        # The string is mostly converted to 'Replacement Character'.
        output=u'A\ufeff\ufffd\ufffd\ufffdsfs\ufffd(B\n',
        tags=sorted([
          u'os:' + self.dimensions['os'][0],
          u'os:' + self.dimensions['os'][1],
          u'pool:default',
          u'priority:200',
          u'service_account:none',
          u'swarming.pool.template:none',
          u'swarming.pool.version:pools_cfg_rev',
          u'user:joe@localhost',
        ]))
    self.assertOneTask(args, summary, {})

  def test_invalid_command(self):
    args = ['-T', 'invalid', '--', 'unknown_invalid_command']
    summary = self.gen_expected(
        name=u'invalid',
        exit_code=u'1',
        failure=True,
        output=re.compile(
            u'^<The executable does not exist or a dependent library is '
            u'missing>'))
    self.assertOneTask(args, summary, {})

  def test_hard_timeout(self):
    args = [
      # Need to flush to ensure it will be sent to the server.
      '-T', 'hard_timeout', '--hard-timeout', '1', '--',
      'python', '-u', '-c',
      'import time,sys; sys.stdout.write(\'hi\\n\'); '
        'sys.stdout.flush(); time.sleep(120)',
    ]
    summary = self.gen_expected(
        name=u'hard_timeout',
        exit_code=unicode(SIGNAL_TERM),
        failure=True,
        state=u'TIMED_OUT')
    self.assertOneTask(args, summary, {})

  def test_io_timeout(self):
    args = [
      # Need to flush to ensure it will be sent to the server.
      '-T', 'io_timeout', '--io-timeout', '1', '--',
      'python', '-u', '-c',
      'import time,sys; sys.stdout.write(\'hi\\n\'); '
        'sys.stdout.flush(); time.sleep(120)',
    ]
    summary = self.gen_expected(
        name=u'io_timeout',
        exit_code=unicode(SIGNAL_TERM),
        failure=True,
        state=u'TIMED_OUT')
    self.assertOneTask(args, summary, {})

  def test_success_fails(self):
    def get_cmd(name, exit_code):
      return [
        '-T', name, '--',
        'python', '-u', '-c',
        'import sys; print(\'hi\'); sys.exit(%d)' % exit_code,
      ]
    # tuple(task_request, expectation)
    tasks = [
      (
        get_cmd('simple_success', 0),
        (self.gen_expected(name=u'simple_success'), {}),
      ),
      (
        get_cmd('simple_failure', 1),
        (
          self.gen_expected(
              name=u'simple_failure', exit_code=u'1', failure=True),
          {},
        ),
      ),
      (
        get_cmd('ending_simple_success', 0),
        (self.gen_expected(name=u'ending_simple_success'), {}),
      ),
    ]

    # tuple(task_id, expectation)
    running_tasks = [
      (self.client.task_trigger_raw(args), expected) for args, expected in tasks
    ]

    for task_id, (summary, files) in running_tasks:
      actual_summary, actual_files = self.client.task_collect(task_id)
      performance_stats = actual_summary['shards'][0].pop('performance_stats')
      self.assertPerformanceStatsEmpty(performance_stats)
      self.assertResults(summary, actual_summary)
      actual_files.pop('summary.json')
      self.assertEqual(files, actual_files)

  def test_isolated(self):
    # Make an isolated file, archive it.
    # Assert that the environment variable SWARMING_TASK_ID is set.
    content = {
      HELLO_WORLD + u'.py': _script(u"""
        # coding=utf-8
        import os
        import sys
        print('hi')
        assert os.environ.get("SWARMING_TASK_ID")
        with open(os.path.join(sys.argv[1], u'💣.txt'), 'wb') as f:
          f.write('test_isolated')
        """),
    }
    # The problem here is that we don't know the isolzed size yet, so we need to
    # do this first.
    name = 'isolated_task'
    isolated_hash, isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(name=u'isolated_task', output=u'hi\n')
    expected_files = {
      os.path.join(u'0', u'💣.txt'.encode('utf-8')): 'test_isolated',
    }
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name, ['--', '${ISOLATED_OUTDIR}'],
        expected_summary, expected_files, deduped=False)
    result_isolated_size = self.assertOutputsRef(outputs_ref)
    items_out = [len('test_isolated'), result_isolated_size]
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': sorted(items_out),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_out)),
        u'total_bytes_items_cold': unicode(sum(items_out)),
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

  def test_isolated_command(self):
    # Command is specified in Swarming task, still with isolated file.
    # Confirms that --relative-cwd, --env and --env-prefix work.
    content = {
      os.path.join(u'base', HELLO_WORLD + u'.py'): _script(u"""
        # coding=utf-8
        import os
        import sys
        print(sys.argv[1])
        assert "SWARMING_TASK_ID" not in os.environ
        # cwd is in base/
        cwd = os.path.realpath(os.getcwd())
        base = os.path.basename(cwd)
        assert base == 'base', base
        cwd = os.path.dirname(cwd)
        path = os.environ["PATH"].split(os.pathsep)
        print(os.path.realpath(path[0]).replace(cwd, "$CWD"))
        with open(os.path.join(sys.argv[2], 'FOO.txt'), 'wb') as f:
          f.write(os.environ["FOO"])
        with open(os.path.join(sys.argv[2], 'result.txt'), 'wb') as f:
          f.write('hey2')
        """),
    }
    isolate_content = '{"variables": {"files": ["%s"]}}' % os.path.join(
        u'base', HELLO_WORLD + u'.py').encode('utf-8')
    name = 'separate_cmd'
    isolated_hash, isolated_size = self._archive(name, content, isolate_content)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(
        name=u'separate_cmd',
        output=u'hi💩\n%s\n' % os.sep.join(['$CWD', 'local', 'path']))
    expected_files = {
      os.path.join('0', 'result.txt'): 'hey2',
      os.path.join('0', 'FOO.txt'): u'bar💩'.encode('utf-8'),
    }
    # Use a --raw-cmd instead of a command in the isolated file. This is the
    # future!
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name,
        ['--raw-cmd',
         '--relative-cwd', 'base',
         '--env', 'FOO', u'bar💩',
         '--env', 'SWARMING_TASK_ID', '',
         '--env-prefix', 'PATH', 'local/path',
         '--', 'python', HELLO_WORLD + u'.py', u'hi💩',
         '${ISOLATED_OUTDIR}'],
        expected_summary, expected_files, deduped=False)
    result_isolated_size = self.assertOutputsRef(outputs_ref)
    items_out = [
      len('hey2'), len(u'bar💩'.encode('utf-8')), result_isolated_size,
    ]
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': sorted(items_out),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_out)),
        u'total_bytes_items_cold': unicode(sum(items_out)),
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

  def test_isolated_hard_timeout(self):
    # Make an isolated file, archive it, have it time out. Similar to
    # test_hard_timeout. The script doesn't handle signal so it failed the grace
    # period.
    content = {
      HELLO_WORLD + u'.py': _script(u"""
        import os
        import sys
        import time
        sys.stdout.write('hi\\n')
        sys.stdout.flush()
        time.sleep(120)
        with open(os.path.join(sys.argv[1], 'result.txt'), 'wb') as f:
          f.write('test_isolated_hard_timeout')
        """),
    }
    name = 'isolated_hard_timeout'
    isolated_hash, isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(
        name=u'isolated_hard_timeout',
        exit_code=unicode(SIGNAL_TERM),
        failure=True,
        state=u'TIMED_OUT')
    # Hard timeout is enforced by run_isolated, I/O timeout by task_runner.
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name,
        ['--hard-timeout', '1', '--', '${ISOLATED_OUTDIR}'],
        expected_summary, {}, deduped=False)
    self.assertIsNone(outputs_ref)
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': [],
        u'items_hot': [],
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

  def test_isolated_hard_timeout_grace(self):
    # Make an isolated file, archive it, have it time out. Similar to
    # test_hard_timeout. The script handles signal so it send results back.
    content = {
      HELLO_WORLD + u'.py': _script(u"""
        import os
        import signal
        import sys
        import threading
        bit = threading.Event()
        def handler(signum, _):
          bit.set()
          sys.stdout.write('got signal %%d\\n' %% signum)
          sys.stdout.flush()
        signal.signal(signal.%s, handler)
        sys.stdout.write('hi\\n')
        sys.stdout.flush()
        while not bit.wait(0.01):
          pass
        with open(os.path.join(sys.argv[1], 'result.txt'), 'wb') as f:
          f.write('test_isolated_hard_timeout_grace')
        """ % ('SIGBREAK' if sys.platform == 'win32' else 'SIGTERM')
        ),
    }
    name = 'isolated_hard_timeout_grace'
    isolated_hash, isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(
        name=u'isolated_hard_timeout_grace',
        output=u'hi\ngot signal 15\n',
        failure=True,
        state=u'TIMED_OUT')
    expected_files = {
      os.path.join('0', 'result.txt'): 'test_isolated_hard_timeout_grace',
    }
    # Hard timeout is enforced by run_isolated, I/O timeout by task_runner.
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name,
        ['--hard-timeout', '1', '--', '${ISOLATED_OUTDIR}'],
        expected_summary, expected_files, deduped=False)
    result_isolated_size = self.assertOutputsRef(outputs_ref)
    items_out = [len('test_isolated_hard_timeout_grace'), result_isolated_size]
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': sorted(items_out),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_out)),
        u'total_bytes_items_cold': unicode(sum(items_out)),
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

  def test_idempotent_reuse(self):
    content = {HELLO_WORLD + u'.py': 'print "hi"\n'}
    name = 'idempotent_reuse'
    isolated_hash, isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(name=u'idempotent_reuse')
    task_id, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name, ['--idempotent'], expected_summary, {},
        deduped=False)
    self.assertIsNone(outputs_ref)
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': [],
        u'items_hot': [],
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

    # The task name changes, there's a bit less data but the rest is the same.
    expected_summary[u'name'] = u'idempotent_reuse2'
    expected_summary[u'cost_saved_usd'] = 0.02
    expected_summary[u'deduped_from'] = task_id[:-1] + u'1'
    expected_summary[u'try_number'] = u'0'
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, 'idempotent_reuse2', ['--idempotent'], expected_summary,
        {}, deduped=True)
    self.assertIsNone(outputs_ref)
    self.assertIsNone(performance_stats)

  def test_secret_bytes(self):
    content = {
      HELLO_WORLD + u'.py': _script(u"""
        import sys
        import os
        import json

        print "hi"

        with open(os.environ['LUCI_CONTEXT'], 'r') as f:
          data = json.load(f)

        with open(os.path.join(sys.argv[1], 'sekret'), 'w') as f:
          print >> f, data['swarming']['secret_bytes'].decode('base64')
      """),
    }
    name = 'secret_bytes'
    isolated_hash, isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(name=u'secret_bytes')
    tmp = os.path.join(self.tmpdir, 'test_secret_bytes')
    with fs.open(tmp, 'wb') as f:
      f.write('foobar')
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name,
        ['--secret-bytes-path', tmp, '--', '${ISOLATED_OUTDIR}'],
        expected_summary, {os.path.join('0', 'sekret'): 'foobar\n'},
        deduped=False)
    result_isolated_size = self.assertOutputsRef(outputs_ref)
    items_out = [len('foobar\n'), result_isolated_size]
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': sorted(items_out),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_out)),
        u'total_bytes_items_cold': unicode(sum(items_out)),
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

  def test_local_cache(self):
    # First task creates the cache, second copy the content to the output
    # directory. Each time it's the exact same script.
    dimensions = {
        i['key']: i['value'] for i in self.client.query_bot()['dimensions']}
    self.assertEqual(set(self.dimensions), set(dimensions))
    self.assertNotIn(u'cache', set(dimensions))
    content = {
      HELLO_WORLD + u'.py': _script(u"""
        import os, shutil, sys
        p = "p/b/a.txt"
        if not os.path.isfile(p):
          with open(p, "wb") as f:
            f.write("Yo!")
        else:
          shutil.copy(p, sys.argv[1])
        print "hi"
        """),
    }
    name = 'cache_first'
    isolated_hash, isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    items_in = [sum(len(c) for c in content.itervalues()), isolated_size]
    expected_summary = self.gen_expected(name=u'cache_first')
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, name,
        ['--named-cache', 'fuu', 'p/b', '--', '${ISOLATED_OUTDIR}/yo'],
        expected_summary, {}, deduped=False)
    self.assertIsNone(outputs_ref)
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': u'0',
        u'initial_size': u'0',
        u'items_cold': sorted(items_in),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_in)),
        u'total_bytes_items_cold': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': [],
        u'items_hot': [],
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

    # Second run with a cache available.
    expected_summary = self.gen_expected(name=u'cache_second')
    # The previous task caused the bot to have a named cache.
    # pylint: disable=not-an-iterable,unsubscriptable-object
    expected_summary['bot_dimensions'] = (
        expected_summary['bot_dimensions'][:])
    self.assertNotIn(
        'caches', [i['key'] for i in expected_summary['bot_dimensions']])
    expected_summary['bot_dimensions'] = [
      {u'key': u'caches', u'value': [u'fuu']}
    ] + expected_summary['bot_dimensions']
    _, outputs_ref, performance_stats = self._run_isolated(
        isolated_hash, 'cache_second',
        ['--named-cache', 'fuu', 'p/b', '--', '${ISOLATED_OUTDIR}/yo'],
        expected_summary,
        {'0/yo': 'Yo!'}, deduped=False)
    result_isolated_size = self.assertOutputsRef(outputs_ref)
    items_out = [3, result_isolated_size]
    expected_performance_stats = {
      u'isolated_download': {
        u'initial_number_items': unicode(len(items_in)),
        u'initial_size': unicode(sum(items_in)),
        u'items_cold': [],
        # Items are hot.
        u'items_hot': items_in,
        u'num_items_hot': unicode(len(items_in)),
        u'total_bytes_items_hot': unicode(sum(items_in)),
      },
      u'isolated_upload': {
        u'items_cold': sorted(items_out),
        u'items_hot': [],
        u'num_items_cold': unicode(len(items_out)),
        u'total_bytes_items_cold': unicode(sum(items_out)),
      },
    }
    self.assertPerformanceStats(expected_performance_stats, performance_stats)

    # Check that the bot now has a cache dimension by independently querying.
    expected = set(self.dimensions)
    expected.add(u'caches')
    dimensions = {
        i['key']: i['value'] for i in self.client.query_bot()['dimensions']}
    self.assertEqual(expected, set(dimensions))

  def test_priority(self):
    # Make a test that keeps the bot busy, while all the other tasks are being
    # created with priorities that are out of order. Then it unblocks the bot
    # and asserts the tasks are run in the expected order, based on priority and
    # created_ts.
    # List of tuple(task_name, priority, task_id).
    tasks = []
    tags = [
      u'pool:default',
      u'service_account:none',
      u'swarming.pool.template:none',
      u'swarming.pool.version:pools_cfg_rev',
      u'user:joe@localhost',
    ]
    with self._make_wait_task('test_priority'):
      # This is the order of the priorities used for each task triggered. In
      # particular, below it asserts that the priority 8 tasks are run in order
      # of created_ts.
      for i, priority in enumerate((9, 8, 6, 7, 8)):
        task_name = u'%d-p%d' % (i, priority)
        args = [
          '-T', task_name, '--priority', str(priority), '--',
          'python', '-u', '-c', 'print(\'%d\')' % priority,
        ]
        tasks.append((task_name, priority, self.client.task_trigger_raw(args)))
      # Ensure the tasks under test are pending.
      for task_name, priority, task_id in tasks:
        result = self.client.task_result(task_id)
      self.assertEqual(u'PENDING', result[u'state'], result)

    # The wait task is done. This will cause all the pending tasks to be run.
    # Now, will they be run in the expected order? That is the question!
    # List of tuple(task_name, priority, task_id, results).
    results = []
    # Collect every tasks.
    for task_name, priority, task_id in tasks:
      actual_summary, actual_files = self.client.task_collect(task_id)
      performance_stats = actual_summary['shards'][0].pop('performance_stats')
      self.assertPerformanceStatsEmpty(performance_stats)
      t = tags[:] + [u'priority:%d' % priority]
      expected_summary = self.gen_expected(
          name=task_name, tags=sorted(t), output=u'%d\n' % priority)
      self.assertResults(expected_summary, actual_summary)
      self.assertEqual(['summary.json'], actual_files.keys())
      results.append(
          (task_name, priority, task_id, actual_summary[u'shards'][0]))

    # Now assert that they ran in the expected order. started_ts encoding means
    # string sort is equivalent to timestamp sort.
    results.sort(key=lambda x: x[3][u'started_ts'])
    expected = ['2-p6', '3-p7', '1-p8', '4-p8', '0-p9']
    self.assertEqual(expected, [r[0] for r in results])

  def test_cancel_pending(self):
    # Cancel a pending task. Triggering a task for an unknown dimension will
    # result in state NO_RESOURCE, so instead a dummy task is created to make
    # sure the bot cannot reap the second one.
    args = [
      '-T', 'cancel_pending',
      '--', 'python', '-u', '-c', 'print(\'hi\')'
    ]
    with self._make_wait_task('test_cancel_pending'):
      task_id = self.client.task_trigger_raw(args)
      actual, actual_files = self.client.task_collect(task_id, timeout=-1)
      self.assertEqual(u'PENDING', actual[u'shards'][0][u'state'])
      self.assertTrue(self.client.task_cancel(task_id, []))
      self.assertEqual(['summary.json'], actual_files.keys())
    actual, actual_files = self.client.task_collect(task_id, timeout=-1)
    self.assertEqual(
        u'CANCELED', actual[u'shards'][0][u'state'], actual[u'shards'][0])
    self.assertEqual(['summary.json'], actual_files.keys())

  def test_kill_running(self):
    # Kill a running task. Make sure the target process handles the signal
    # well with a graceful termination via SIGTERM.
    content = {
      HELLO_WORLD + u'.py': _script(u"""
        import signal
        import sys
        import threading
        bit = threading.Event()
        def handler(signum, _):
          bit.set()
          sys.stdout.write('got signal %%d\\n' %% signum)
          sys.stdout.flush()
        signal.signal(signal.%s, handler)
        sys.stdout.write('hi\\n')
        sys.stdout.flush()
        while not bit.wait(0.01):
          pass
        sys.exit(23)
        """) % ('SIGBREAK' if sys.platform == 'win32' else 'SIGTERM'),
    }
    name = 'kill_running'
    isolated_hash, _isolated_size = self._archive(
        name, content, DEFAULT_ISOLATE_HELLO)
    # Do not use self._run_isolated() here since we want to kill it, not wait
    # for it to complete.
    task_id = self.client.task_trigger_isolated(
        isolated_hash, name, ['--', '${ISOLATED_OUTDIR}'])

    # Wait for the task to start on the bot.
    self._wait_for_state(task_id, u'PENDING', u'RUNNING')
    # Make sure the signal handler is set by looking that 'hi' was printed,
    # otherwise it could continue looping.
    start = time.time()
    out = None
    while time.time() - start < 45.:
      out = self.client.task_stdout(task_id)
      if out == 'hi\n':
        break
    self.assertEqual(out, 'hi\n')

    # Cancel it.
    self.assertTrue(self.client.task_cancel(task_id, ['--kill-running']))

    result = self._wait_for_state(task_id, u'RUNNING', u'KILLED')
    # Make sure the exit code is what the script returned, which means the
    # script in fact did handle the SIGTERM and returned properly.
    self.assertEqual(u'23', result[u'exit_code'])

  def test_task_slice(self):
    request = {
      'name': 'task_slice',
      'priority': 40,
      'task_slices': [
        {
          'expiration_secs': 120,
          'properties': {
            'command': ['python', '-c', 'print("first")'],
            'dimensions': [
              {
                'key': 'pool',
                'value': 'default',
              },
            ],
            'grace_period_secs': 30,
            'execution_timeout_secs': 30,
            'io_timeout_secs': 30,
          },
        },
        {
          'expiration_secs': 120,
          'properties': {
            'command': ['python', '-c', 'print("second")'],
            'dimensions': [
              {
                'key': 'pool',
                'value': 'default',
              },
            ],
            'grace_period_secs': 30,
            'execution_timeout_secs': 30,
            'io_timeout_secs': 30,
          },
        },
      ],
    }
    task_id = self.client.task_trigger_post(json.dumps(request))
    expected_summary = self.gen_expected(
        name=u'task_slice',
        output=u'first\n',
        tags=[
          u'pool:default',
          u'priority:40',
          u'service_account:none',
          u'swarming.pool.template:none',
          u'swarming.pool.version:pools_cfg_rev',
          u'user:None',
        ],
        user=u'')
    actual_summary, actual_files = self.client.task_collect(task_id)
    performance_stats = actual_summary['shards'][0].pop('performance_stats')
    self.assertPerformanceStatsEmpty(performance_stats)
    self.assertResults(expected_summary, actual_summary, deduped=False)
    self.assertEqual(['summary.json'], actual_files.keys())

  def test_task_slice_fallback(self):
    # The first one shall be skipped.
    request = {
      'name': 'task_slice_fallback',
      'priority': 40,
      'task_slices': [
        {
          # Really long expiration that would cause the smoke test to abort
          # under normal condition.
          'expiration_secs': 1200,
          'properties': {
            'command': ['python', '-c', 'print("first")'],
            'dimensions': [
              {
                'key': 'pool',
                'value': 'default',
              },
              {
                'key': 'invalidkey',
                'value': 'invalidvalue',
              },
            ],
            'grace_period_secs': 30,
            'execution_timeout_secs': 30,
            'io_timeout_secs': 30,
          },
        },
        {
          'expiration_secs': 120,
          'properties': {
            'command': ['python', '-c', 'print("second")'],
            'dimensions': [
              {
                'key': 'pool',
                'value': 'default',
              },
            ],
            'grace_period_secs': 30,
            'execution_timeout_secs': 30,
            'io_timeout_secs': 30,
          },
        },
      ],
    }
    task_id = self.client.task_trigger_post(json.dumps(request))
    expected_summary = self.gen_expected(
        name=u'task_slice_fallback',
        current_task_slice=u'1',
        output=u'second\n',
        tags=[
          # Bug!
          u'invalidkey:invalidvalue',
          u'pool:default',
          u'priority:40',
          u'service_account:none',
          u'swarming.pool.template:none',
          u'swarming.pool.version:pools_cfg_rev',
          u'user:None',
        ],
        user=u'')
    actual_summary, actual_files = self.client.task_collect(task_id)
    performance_stats = actual_summary['shards'][0].pop('performance_stats')
    self.assertPerformanceStatsEmpty(performance_stats)
    self.assertResults(expected_summary, actual_summary, deduped=False)
    self.assertEqual(['summary.json'], actual_files.keys())

  def test_no_resource(self):
    request = {
      'name': 'no_resource',
      'priority': 40,
      'task_slices': [
        {
          # Really long expiration that would cause the smoke test to abort
          # under normal conditions.
          'expiration_secs': 1200,
          'properties': {
            'command': ['python', '-c', 'print("hello")'],
            'dimensions': [
              {
                'key': 'pool',
                'value': 'default',
              },
              {
                'key': 'bad_key',
                'value': 'bad_value',
              },
            ],
            'grace_period_secs': 30,
            'execution_timeout_secs': 30,
            'io_timeout_secs': 30,
          },
        },
      ],
    }
    task_id = self.client.task_trigger_post(json.dumps(request))
    actual_summary, actual_files = self.client.task_collect(task_id)
    summary = actual_summary[u'shards'][0]
    # Immediately cancelled.
    self.assertEqual(u'NO_RESOURCE', summary[u'state'])
    self.assertTrue(summary[u'abandoned_ts'])
    self.assertEqual(['summary.json'], actual_files.keys())

  @contextlib.contextmanager
  def _make_wait_task(self, name):
    """Creates a dummy task that keeps the bot busy, while other things are
    being done.
    """
    signal_file = os.path.join(self.tmpdir, name)
    fs.open(signal_file, 'wb').close()
    args = [
      '-T', 'wait', '--priority', '20', '--',
      'python', '-u', '-c',
      # Cheezy wait.
      ('import os,time;'
        'print(\'hi\');'
        '[time.sleep(0.1) for _ in xrange(100000) if os.path.exists(\'%s\')];'
        'print(\'hi again\')') % signal_file,
    ]
    wait_task_id = self.client.task_trigger_raw(args)
    # Assert that the 'wait' task has started but not completed, otherwise
    # this defeats the purpose.
    self._wait_for_state(wait_task_id, u'PENDING', u'RUNNING')
    yield
    # Double check.
    result = self.client.task_result(wait_task_id)
    self.assertEqual(u'RUNNING', result[u'state'], result)
    # Unblock the wait_task_id on the bot.
    os.remove(signal_file)
    # Ensure the initial wait task is completed.
    actual_summary, actual_files = self.client.task_collect(wait_task_id)
    tags = [
      u'pool:default',
      u'priority:20',
      u'service_account:none',
      u'swarming.pool.template:none',
      u'swarming.pool.version:pools_cfg_rev',
      u'user:joe@localhost',
    ]
    performance_stats = actual_summary['shards'][0].pop('performance_stats')
    self.assertPerformanceStatsEmpty(performance_stats)
    self.assertResults(
        self.gen_expected(name=u'wait', tags=tags, output=u'hi\nhi again\n'),
        actual_summary)
    self.assertEqual(['summary.json'], actual_files.keys())

  def _run_isolated(
      self, isolated_hash, name, args, expected_summary, expected_files,
      deduped):
    """Triggers a Swarming task and asserts results.

    It runs a python script archived as an isolated file.

    Returns:
      tuple of:
        task_id
        outputs_ref value if any, or None
        performance_stats if any, or None
    """
    task_id = self.client.task_trigger_isolated(isolated_hash, name, args)
    actual_summary, actual_files = self.client.task_collect(task_id)
    outputs_ref = actual_summary[u'shards'][0].pop('outputs_ref', None)
    performance_stats = actual_summary[u'shards'][0].pop(
        'performance_stats', None)
    self.assertResults(expected_summary, actual_summary, deduped=deduped)
    actual_files.pop('summary.json')
    self.assertEqual(expected_files, actual_files)
    return task_id, outputs_ref, performance_stats

  def _archive(self, name, contents, isolate_content):
    """Archives data to the isolate server.

    Returns the isolated hash and its size.
    """
    # Shared code for all test_isolated_* test cases.
    root = os.path.join(self.tmpdir, name)
    # Refuse reusing the same task name twice, it makes the whole test suite
    # more manageable.
    self.assertFalse(os.path.isdir(root), root)
    fs.mkdir(root)
    isolate_path = os.path.join(root, 'i.isolate')
    isolated_path = os.path.join(root, 'i.isolated')
    with fs.open(isolate_path, 'wb') as f:
      f.write(isolate_content)
    for relpath, content in contents.iteritems():
      p = os.path.join(root, relpath)
      d = os.path.dirname(p)
      if not os.path.isdir(d):
        os.makedirs(d)
      with fs.open(p, 'wb') as f:
        f.write(content)
    isolated_hash = self.client.isolate(isolate_path, isolated_path)
    return isolated_hash, fs.stat(isolated_path).st_size

  def assertResults(self, expected, result, deduped=False):
    """Compares the outputs of a swarming task."""
    self.assertEqual([u'shards'], result.keys())
    self.assertEqual(1, len(result[u'shards']))
    self.assertTrue(result[u'shards'][0], result)
    result = result[u'shards'][0].copy()
    self.assertFalse(result.get(u'abandoned_ts'))
    bot_version = result.pop(u'bot_version')
    self.assertTrue(bot_version)
    if result.get(u'costs_usd') is not None:
      expected.pop(u'costs_usd', None)
      self.assertLess(0, result.pop(u'costs_usd'))
    if result.get(u'cost_saved_usd') is not None:
      expected.pop(u'cost_saved_usd', None)
      self.assertLess(0, result.pop(u'cost_saved_usd'))
    self.assertTrue(result.pop(u'created_ts'))
    self.assertTrue(result.pop(u'completed_ts'))
    self.assertLess(0, result.pop(u'duration'))
    task_id = result.pop(u'task_id')
    run_id = result.pop(u'run_id')
    self.assertTrue(task_id)
    self.assertTrue(task_id.endswith('0'), task_id)
    if not deduped:
      self.assertEqual(task_id[:-1] + '1', run_id)
    self.assertTrue(result.pop(u'modified_ts'))
    self.assertTrue(result.pop(u'started_ts'))

    if getattr(expected.get(u'output'), 'match', None):
      expected_output = expected.pop(u'output')
      output = result.pop('output')
      self.assertTrue(
          expected_output.match(output),
          '%s does not match %s' % (output, expected_output.pattern))

    self.assertEqual(expected, result)
    return bot_version

  def assertOneTask(self, args, expected_summary, expected_files):
    """Runs a single task at a time."""
    task_id = self.client.task_trigger_raw(args)
    actual_summary, actual_files = self.client.task_collect(task_id)
    performance_stats = actual_summary['shards'][0].pop('performance_stats')
    self.assertPerformanceStatsEmpty(performance_stats)
    bot_version = self.assertResults(expected_summary, actual_summary)
    actual_files.pop('summary.json')
    self.assertEqual(expected_files, actual_files)
    return bot_version

  def assertOutputsRef(self, actual):
    """Returns the size of the isolated file."""
    self.assertEqual(self.servers.isolate_server.url, actual['isolatedserver'])
    self.assertEqual(self.namespace, actual['namespace'])
    path = os.path.join(self.tmpdir, actual['isolated'])
    self.client.retrieve_file(actual['isolated'], path)
    return fs.stat(path).st_size

  def assertPerformanceStatsEmpty(self, actual):
    self.assertLess(0, actual.pop(u'bot_overhead'))
    self.assertEqual(
        {
          u'isolated_download': {
            u'initial_number_items': u'0',
            u'initial_size': u'0',
          },
          u'isolated_upload': {},
        },
        actual)

  def assertPerformanceStats(self, expected, actual):
    # These are not deterministic (or I'm too lazy to calculate the value).
    self.assertLess(0, actual.pop(u'bot_overhead'))
    self.assertLess(0, actual[u'isolated_download'].pop(u'duration'))
    self.assertLess(0, actual[u'isolated_upload'].pop(u'duration'))
    for k in (u'isolated_download', u'isolated_upload'):
      for j in (u'items_cold', u'items_hot'):
        actual[k][j] = large.unpack(base64.b64decode(actual[k].get(j, '')))
    self.assertEqual(expected, actual)

  def _wait_for_state(self, task_id, current, new):
    """Waits for the task to start on the bot."""
    state = result = None
    start = time.time()
    # 45 seconds is a long time.
    # TODO(maruel): Make task_runner use exponential backoff instead of
    # hardcoded 10s/30s, which makes these tests tremendously slower than
    # necessary.
    # https://crbug.com/825500
    while time.time() - start < 45.:
      result = self.client.task_result(task_id)
      state = result[u'state']
      if state == new:
        break
      self.assertEqual(current, state, result)
      time.sleep(0.01)
    self.assertEqual(new, state, result)
    return result


def cleanup(bot, client, servers, print_all):
  """Kills bot, kills server, print logs if failed, delete tmpdir."""
  try:
    if bot:
      bot.stop()
  finally:
    if servers:
      servers.stop()
  if print_all:
    if bot:
      bot.dump_log()
    if servers:
      servers.dump_log()
    if client:
      client.dump_log()
  if not Test.leak:
    file_path.rmtree(Test.tmpdir)


def main():
  fix_encoding.fix_encoding()
  verbose = '-v' in sys.argv
  Test.leak = bool('--leak' in sys.argv)
  Test.tmpdir = unicode(tempfile.mkdtemp(prefix='local_smoke_test'))
  if Test.leak:
    # Note that --leak will not guarantee that 'c' and 'isolated_cache' are
    # kept. Only the last test case will leak these two directories.
    sys.argv.remove('--leak')
  if verbose:
    logging.basicConfig(level=logging.INFO)
    Test.maxDiff = None
  else:
    logging.basicConfig(level=logging.ERROR)

  # Force language to be English, otherwise the error messages differ from
  # expectations.
  os.environ['LANG'] = 'en_US.UTF-8'
  os.environ['LANGUAGE'] = 'en_US.UTF-8'

  # So that we don't get goofed up when running this test on swarming :)
  os.environ.pop('SWARMING_TASK_ID', None)
  os.environ.pop('SWARMING_SERVER', None)
  os.environ.pop('ISOLATE_SERVER', None)

  # Ensure ambient identity doesn't leak through during local smoke test.
  os.environ.pop('LUCI_CONTEXT', None)

  bot = None
  client = None
  servers = None
  failed = True
  try:
    servers = start_servers.LocalServers(False, Test.tmpdir)
    servers.start()
    botdir = os.path.join(Test.tmpdir, 'bot')
    os.mkdir(botdir)
    bot = start_bot.LocalBot(servers.swarming_server.url, True, botdir)
    Test.bot = bot
    bot.start()
    namespace = 'sha256-deflate'
    client = SwarmingClient(
        servers.swarming_server.url, servers.isolate_server.url, namespace,
        Test.tmpdir)
    # Test cases only interract with the client; except for test_update_continue
    # which mutates the bot.
    Test.client = client
    Test.servers = servers
    Test.namespace = namespace
    failed = not unittest.main(exit=False).result.wasSuccessful()

    # Then try to terminate the bot sanely. After the terminate request
    # completed, the bot process should have terminated. Give it a few
    # seconds due to delay between sending the event that the process is
    # shutting down vs the process is shut down.
    if client.terminate(bot.bot_id) != 0:
      print >> sys.stderr, 'swarming.py terminate failed'
      failed = True
    try:
      bot.wait(10)
    except subprocess42.TimeoutExpired:
      print >> sys.stderr, 'Bot is still alive after swarming.py terminate'
      failed = True
  except KeyboardInterrupt:
    print >> sys.stderr, '<Ctrl-C>'
    failed = True
    if bot is not None and bot.poll() is None:
      bot.kill()
      bot.wait()
  finally:
    cleanup(bot, client, servers, failed or verbose)
  return int(failed)


if __name__ == '__main__':
  sys.exit(main())
