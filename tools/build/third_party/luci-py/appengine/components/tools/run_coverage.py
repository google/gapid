#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Runs the coverage tool on unit tests in %s and prints out report.

The tests are run in parallel for maximum efficiency.
"""

import optparse
import os
import socket
import subprocess
import sys
import tempfile
import time

# When run directly, runs the components/ unit tests.
ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))


def _get_tests(root, blacklist):
  """Yields all the tests to run excluding smoke tests."""
  for root, dirs, files in os.walk(root):
    for d in dirs[:]:
      if d.startswith('.') or d in blacklist:
        dirs.remove(d)

    for f in files:
      if f.endswith('_test.py') and not f.endswith('_smoke_test.py'):
        yield os.path.join(root, f)


def _run_coverage(root, args):
  """Returns True on success, e.g. if exit code is 0."""
  subprocess.check_call(['coverage'] + args, cwd=root)


def _start_coverage(root, args):
  """Returns a Popen instance."""
  return subprocess.Popen(
      ['coverage'] + args, cwd=root,
      stderr=subprocess.STDOUT, stdout=subprocess.PIPE)


def main(root, blacklist, omit):
  parser = optparse.OptionParser(
      description=sys.modules[__name__].__doc__ % root)
  parser.add_option(
      '-H', '--html',
      help='Generates HTML report to this directory instead of printing it out')
  options, args = parser.parse_args()
  if args:
    parser.error('Unknown args: %s' % args)

  try:
    _run_coverage(root, ['erase'])
  except OSError:
    print >> sys.stderr, (
        'Please install coverage first.\n'
        'See http://nedbatchelder.com/code/coverage for more details.')
    return 2

  # http://nedbatchelder.com/code/coverage/config.html
  rcfile_content = (
    '[report]',
    'exclude_lines =',
    r'    pragma: no cover',
    r'    def __repr__\(',
    r'    raise NotImplementedError\(',
    r'    if False:',
    r'    unittest\.TestCase\.maxDiff = None',
    r'    (self|test)\.fail\(',
    'precision = 1',
  )

  h, rcfile = tempfile.mkstemp(prefix='coverage', suffix='.rc')
  os.close(h)
  try:
    with open(rcfile, 'wb') as f:
      f.write('\n'.join(rcfile_content) + '\n')
    flags = [
      'run',
      '--parallel-mode',
      '--omit=' + omit,
      '--rcfile=' + rcfile,
      '--source=' + root,
    ]
    start = time.time()
    processes = {
      _start_coverage(root, flags + [test]): None
      for test in _get_tests(root, blacklist)
    }
    if not processes:
      print('Failed to find any test in %s' % root)
      return 1

    # Poll for the processes to complete.
    while not all(v is not None for v in processes.itervalues()):
      for proc, v in processes.iteritems():
        if v is None:
          # TODO(maruel): The test may hang if the stdout pipe becomes full. Fix
          # if it becomes an issue in practice (it hasn't here) and the speed
          # improvement is definitely worth it.
          processes[proc] = proc.poll()
          if processes[proc] is not None:
            # Empty the pipe in any case.
            out = proc.communicate()[0]
            if processes[proc]:
              sys.stdout.write(out)
            else:
              sys.stdout.write('.')
            sys.stdout.flush()
      time.sleep(0.1)
    end = time.time()
    results = [not v for v in processes.itervalues()]
    print(
        '\n%d out of %d tests succeeded in %.2fs.\n' %
        (sum(results), len(results), end-start))
    _run_coverage(root, ['combine', '--rcfile=' + rcfile])

    if options.html:
      full_path = os.path.normpath(os.path.join(root, options.html))
      args = [
        'html',
        '--directory', os.path.relpath(full_path, root),
        '--rcfile=' + rcfile,
      ]
      _run_coverage(root, args)
      rel_path = os.path.relpath(full_path, os.getcwd())
      if rel_path.startswith('..'):
        print('First run "cd %s"' % root)
        rel_path = options.html
      print('Start a web browser with "python -m SimpleHTTPServer"')
      print(
          'Then point your web browser at http://%s:8000/%s' %
          (socket.getfqdn(), rel_path.replace(os.path.sep, '/')))
    else:
      args = ['report', '-m', '--rcfile=' + rcfile]
      _run_coverage(root, args)
      print('To generate HTML report, use "coverage html -d out"')

    # Pass out the number of tests that failed as exit code.
    return len(results) - sum(results)
  finally:
    os.remove(rcfile)


if __name__ == '__main__':
  sys.exit(main(
      ROOT_DIR,
      ('third_party', 'tools'),
      'PRESUBMIT.py,third_party/*,tools/*'))
