#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Prints a short log from HEAD (or [end]) to a pseudo revision number."""

__version__ = '1.0'

import optparse
import subprocess
import sys

import calculate_version  # pylint: disable=W0403


def get_logs(root, pseudo_revision, mergebase, start, end):
  start_ref = '%s~%d' % (mergebase, pseudo_revision - start)
  end_ref = mergebase
  if end is not None:
    end_ref += '~%d' % (pseudo_revision - end)
  refspec = '%s..%s' % (start_ref, end_ref)
  cmd = ['git', 'log', refspec, '--date=short', '--format=%ad %ae %s']
  nb_commits = (end or pseudo_revision) - start
  try:
    log = subprocess.check_output(cmd, cwd=root)
  except subprocess.CalledProcessError:
    print >> sys.stderr, (
        '\nFailed to retrieve the log of last %d commits.' % nb_commits)
    return 1
  maxlen = 0
  lines = []
  for l in log.rstrip().splitlines():
    parts = l.split(' ', 2)
    parts[1] = parts[1].split('@', 1)[0]
    maxlen = max(maxlen, len(parts[1]))
    lines.append(parts)
  out = '\n'.join(
    '%s %-*s %s' % (parts[0], maxlen, parts[1], parts[2])
    for parts in lines)
  return out, refspec


def main():
  root = calculate_version.checkout_root('.')
  pseudo_revision, mergebase = calculate_version.get_head_pseudo_revision(
      root, 'origin/master')
  is_pristine = calculate_version.is_pristine(root, mergebase)

  parser = optparse.OptionParser(
      usage='%prog [options] <start> [end]',
      version=__version__,
      description=sys.modules[__name__].__doc__)
  parser.add_option(
      '-f', '--force', action='store_true',
      help='Run even if not pristine checkout, e.g. HEAD != origin/master')
  parser.add_option(
      '-F', '--files', action='store_true', help='List all modified files')
  options, args = parser.parse_args()

  print >> sys.stderr, (
      'Current version: %s @ %s\n' % (pseudo_revision, mergebase))

  if not args:
    parser.error('Specify the pseudo-revision number of the last push.')
  start = int(args[0])
  end = None
  if len(args) == 2:
    end = int(args[1])
  if len(args) > 2:
    parser.error('Too many arguments.')

  if start >= pseudo_revision:
    parser.error(
        '%d >= %d, you specified \'start\' that was not committed yet?'
        % (start, pseudo_revision))
  if end is not None:
    if start >= end:
      parser.error('%d >= %d, did you reverse start and end?' % (start, end))
    if end > pseudo_revision:
      parser.error(
          '%d >= %d, you specified \'end\' that was not committed yet?'
          % (end, pseudo_revision))
  nb_commits = (end or pseudo_revision) - start

  if not is_pristine:
    if not options.force:
      parser.error(
          'Make sure to sync to what was committed and uploaded first.')
    print >> sys.stderr, (
        'Warning: --force was specified, continuing even if not pristine.\n')

  out, refspec = get_logs(root, pseudo_revision, mergebase, start, end)
  print(out)

  if options.files:
    print('')
    cmd = ['git', 'diff', refspec, '--stat', '-C', '-C']
    try:
      subprocess.check_call(cmd, cwd=root)
    except subprocess.CalledProcessError:
      print >> sys.stderr, (
          '\nFailed to list files of last %d commits.' % nb_commits)
      return 1

  return 0


if __name__ == '__main__':
  sys.exit(main())
