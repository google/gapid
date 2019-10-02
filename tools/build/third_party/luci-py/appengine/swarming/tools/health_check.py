#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Check the health of a Swarming version."""

import argparse
import collections
import functools
import json
import os
import subprocess
import sys
import time

HERE = os.path.dirname(__file__)
SWARMING_TOOL = os.path.join(HERE, '..', '..', '..', 'client', 'swarming.py')


def retry_exception(exc_type, max_attempts, delay):
  """Decorator to retry a function on failure with linear backoff.

  Args:
    exc_type: The type of exception raised by the function to retry.
    max_attempts: Maximum number of times to call the function before reraising
                  the exception.
    delay: Time to sleep between attempts, in seconds.

  Returns:
    A decorator to be applied to the function.
  """
  def deco(fn):
    @functools.wraps(fn)
    def wrapper(*args, **kwargs):
      for _ in range(max_attempts - 1):
        try:
          return fn(*args, **kwargs)
        except exc_type:
          time.sleep(delay)
      return fn(*args, **kwargs)
    return wrapper
  return deco


@retry_exception(ValueError, 12, 10)
def pick_best_pool(url, server_version):
  """Pick the best pool to run the health check task on.

  Asks the specified swarming server for a list of all bots, filters those
  running the specified server version, and returns the pool with the most bots
  in it.

  Args:
    url: The swarming server to query.
    server_version: Which server version to filter bots by.

  Returns:
    A string indicating the best pool to run the health check task on.
  """
  output = subprocess.check_output([
      SWARMING_TOOL, 'query',
      '-S', url,
      '--limit', '0',
      'bots/list?dimensions=server_version:%s' % server_version,
  ])
  data = json.loads(output)
  bots = data.get('items', [])

  pool_counts = collections.Counter()
  for bot in bots:
    for dimension in bot.get('dimensions', []):
      if dimension['key'] == 'pool':
        pool_counts.update(dimension['value'])

  if not pool_counts:
    raise ValueError('No bots are running server_version=%s' % server_version)

  return pool_counts.most_common(1)[0][0]


def main():
  parser = argparse.ArgumentParser()
  parser.add_argument(
      '--pool',
      help='Pool to schedule a task on. If unspecified, this is autodetected.')
  parser.add_argument('appid')
  parser.add_argument('server_version')
  args = parser.parse_args()

  url = 'https://{server_version}-dot-{appid}.appspot.com'.format(
      appid=args.appid,
      server_version=args.server_version)
  print 'Swarming server:', url

  pool = args.pool
  if not pool:
    print 'Finding best pool to use'
    pool = pick_best_pool(url, args.server_version)

  print 'Scheduling no-op task on pool %r' % pool
  rv = subprocess.call([
      SWARMING_TOOL, 'run',
      '-S', url,
      '--expiration', '120',
      '--hard-timeout', '120',
      '-d', 'pool', pool,
      '-d', 'server_version', args.server_version,
      '--raw-cmd', '--', 'python', '-c', 'pass'])
  if rv != 0:
    print>>sys.stderr, 'Failed to run no-op task'
    return 2
  return 0


if __name__ == '__main__':
  sys.exit(main())
