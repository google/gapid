#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Integration test for the Swarming server."""

import Queue
import json
import logging
import optparse
import os
import subprocess
import sys
import tempfile
import threading
import time

APP_DIR = os.path.dirname(os.path.abspath(__file__))
CHECKOUT_DIR = os.path.dirname(os.path.dirname(APP_DIR))
CLIENT_DIR = os.path.join(CHECKOUT_DIR, 'client')
SWARMING_SCRIPT = os.path.join(CLIENT_DIR, 'swarming.py')

sys.path.insert(0, CLIENT_DIR)
from third_party.depot_tools import fix_encoding
from utils import file_path
sys.path.pop(0)


def gen_isolated(isolate, script, includes=None):
  """Archives a script to `isolate` server."""
  tmp = tempfile.mkdtemp(prefix='swarming_smoke')
  data = {
    'variables': {
      'command': ['python', '-u', 'script.py'],
      'files': ['script.py'],
    },
  }
  try:
    with open(os.path.join(tmp, 'script.py'), 'wb') as f:
      f.write(script)
    path = os.path.join(tmp, 'script.isolate')
    with open(path, 'wb') as f:
      # This file is actually python but it's #closeenough.
      json.dump(data, f, sort_keys=True, separators=(',', ':'))
    isolated = os.path.join(tmp, 'script.isolated')
    cmd = [
      os.path.join(CLIENT_DIR, 'isolate.py'), 'archive',
      '-I', isolate, '-i', path, '-s', isolated,
    ]
    out = subprocess.check_output(cmd)
    if includes:
      # Mangle the .isolated to include another one. A bit hacky but works well.
      # In practice, we'd need to add a --include flag to isolate.py archive or
      # something.
      with open(isolated, 'rb') as f:
        data = json.load(f)
      data['includes'] = includes
      with open(isolated, 'wb') as f:
        json.dump(data, f, sort_keys=True, separators=(',', ':'))
      cmd = [
        os.path.join(CLIENT_DIR, 'isolateserver.py'), 'archive',
        '-I', isolate, '--namespace', 'default-gzip', isolated,
      ]
      out = subprocess.check_output(cmd)
    return out.split(' ', 1)[0]
  finally:
    file_path.rmtree(tmp)


def capture(cmd, **kwargs):
  """Captures output and return exit code."""
  proc = subprocess.Popen(
      cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, **kwargs)
  out = proc.communicate()[0]
  return out, proc.returncode


def test_normal(swarming, isolate, extra_flags):
  """Runs a normal task that succeeds."""
  h = gen_isolated(isolate, 'print(\'SUCCESS\')')
  subprocess.check_output(
      [SWARMING_SCRIPT, 'run', '-S', swarming, '-I', isolate, h] + extra_flags)
  return 'SUCCESS'


def test_expiration(swarming, isolate, extra_flags):
  """Schedule a task that cannot be scheduled and expire."""
  h = gen_isolated(isolate, 'print(\'SUCCESS\')')
  start = time.time()
  out, exitcode = capture(
      [
        SWARMING_SCRIPT, 'run', '-S', swarming, '-I', isolate, h,
        '--expiration', '30', '-d', 'invalid', 'always',
      ] + extra_flags)
  duration = time.time() - start
  if exitcode != 1:
    return 'Unexpected exit code: %d' % exitcode
  # TODO(maruel): Shouldn't take more than a minute or so.
  if duration < 30 or duration > 120:
    return 'Unexpected expiration timeout: %d\n%s' % (duration, out)
  return 'SUCCESS'


def test_io_timeout(swarming, isolate, extra_flags):
  """Runs a task that triggers IO timeout."""
  h = gen_isolated(
      isolate,
      'import time\n'
      'print(\'SUCCESS\')\n'
      'time.sleep(40)\n'
      'print(\'FAILURE\')')
  start = time.time()
  out, exitcode = capture(
      [
        SWARMING_SCRIPT, 'run', '-S', swarming, '-I', isolate, h,
        '--io-timeout', '30',
      ] + extra_flags)
  duration = time.time() - start
  if exitcode != 1:
    return 'Unexpected exit code: %d\n%s' % (exitcode, out)
  if duration < 30:
    return 'Unexpected fast execution: %d' % duration
  return 'SUCCESS'


def test_hard_timeout(swarming, isolate, extra_flags):
  """Runs a task that triggers hard timeout."""
  h = gen_isolated(
      isolate,
      'import time\n'
      'for i in xrange(6):'
      '  print(\'.\')\n'
      '  time.sleep(10)\n')
  start = time.time()
  out, exitcode = capture(
      [
        SWARMING_SCRIPT, 'run', '-S', swarming, '-I', isolate, h,
        '--hard-timeout', '30',
      ] + extra_flags)
  duration = time.time() - start
  if exitcode != 1:
    return 'Unexpected exit code: %d\n%s' % (exitcode, out)
  if duration < 30:
    return 'Unexpected fast execution: %d' % duration
  return 'SUCCESS'


def test_reentrant(swarming, isolate, extra_flags):
  """Runs a task that triggers a child task.

  To be able to do so, it archives all of ../../client/.

  Because the parent task blocks on the child task, it requires at least 2 bots
  alive.
  """
  # First isolate the whole client directory.
  cmd = [
    os.path.join(CLIENT_DIR, 'isolateserver.py'), 'archive',
    '-I', isolate, '--namespace', 'default-gzip', '--blacklist', 'tests',
    CLIENT_DIR,
  ]
  client_isolated = subprocess.check_output(cmd).split()[0]
  logging.info('- %s', client_isolated)

  script = '\n'.join((
      'import os',
      'import subprocess',
      'import sys',
      'print("Before\\n")',
      'print("SWARMING_TASK_ID=%s\\n" % os.environ["SWARMING_TASK_ID"])',
      'subprocess.check_call(',
      '  [sys.executable, "-u", "example/3_swarming_run_auto_upload.py",',
      '    "-S", "%s",' % swarming,
      '    "-I", "%s",' % isolate,
      '    "--verbose",',
      '  ])',
      'print("After\\n")'))
  h = gen_isolated(isolate, script, [client_isolated])
  subprocess.check_output(
      [SWARMING_SCRIPT, 'run', '-S', swarming, '-I', isolate, h] + extra_flags)
  return 'SUCCESS'


def get_all_tests():
  m = sys.modules[__name__]
  return {k[5:]: getattr(m, k) for k in dir(m) if k.startswith('test_')}


def run_test(results, swarming, isolate, extra_flags, name, test_case):
  start = time.time()
  try:
    result = test_case(swarming, isolate, extra_flags)
  except Exception as e:
    result = e
  results.put((name, result, time.time() - start))


def main():
  fix_encoding.fix_encoding()
  # It's necessary for relative paths in .isolate.
  os.chdir(APP_DIR)

  parser = optparse.OptionParser()
  parser.add_option('-S', '--swarming', help='Swarming server')
  parser.add_option('-I', '--isolate-server', help='Isolate server')
  parser.add_option('-d', '--dimensions', nargs=2, default=[], action='append')
  parser.add_option('-v', '--verbose', action='store_true', help='Logs more')
  options, args = parser.parse_args()

  if args:
    parser.error('Unsupported args: %s' % args)
  if not options.swarming:
    parser.error('--swarming required')
  if not options.isolate_server:
    parser.error('--isolate-server required')
  if not os.path.isfile(SWARMING_SCRIPT):
    parser.error('Invalid checkout, %s does not exist' % SWARMING_SCRIPT)

  logging.basicConfig(level=logging.DEBUG if options.verbose else logging.ERROR)
  extra_flags = ['--priority', '5', '--tags', 'smoke_test:1']
  for k, v in options.dimensions or [('os', 'Linux')]:
    extra_flags.extend(('-d', k, v))

  # Run all the tests in parallel.
  tests = get_all_tests()
  results = Queue.Queue(maxsize=len(tests))

  for name, fn in sorted(tests.iteritems()):
    logging.info('%s', name)
    t = threading.Thread(
        target=run_test, name=name,
        args=(results, options.swarming, options.isolate_server, extra_flags,
              name, fn))
    t.start()

  print('%d tests started' % len(tests))
  maxlen = max(len(name) for name in tests)
  for i in xrange(len(tests)):
    name, result, duration = results.get()
    print('[%d/%d] %-*s: %4.1fs: %s' %
        (i, len(tests), maxlen, name, duration, result))

  return 0


if __name__ == '__main__':
  sys.exit(main())
