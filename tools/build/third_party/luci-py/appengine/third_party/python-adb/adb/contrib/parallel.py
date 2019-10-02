# Copyright 2015 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Implements parallel map (pmap)."""

import logging
import Queue
import sys
import threading


# Current thread pools. This is effectively a leak.
_POOL = []
_POOL_LOCK = threading.Lock()
_QUEUE_IN = Queue.Queue()
_QUEUE_OUT = Queue.Queue()


def pmap(fn, items):
  """Runs map() in parallel.

  Rethrows any exception caught.

  Returns:
  - list(fn() result's) in order.
  """
  if not items:
    return []
  if len(items) == 1:
    return [fn(items[0])]

  # Try to reuse the common pool. Can only be used by one mapper at a time.
  locked = _POOL_LOCK.acquire(False)
  if locked:
    # _QUEUE_OUT may not be empty if the previous run threw.
    while not _QUEUE_OUT.empty():
      _QUEUE_OUT.get()
    try:
      return _pmap(_POOL, _QUEUE_IN, _QUEUE_OUT, fn, items)
    finally:
      _POOL_LOCK.release()

  # A pmap() is currently running, create a temporary pool.
  return _pmap([], Queue.Queue(), Queue.Queue(), fn, items)


def _pmap(pool, queue_in, queue_out, fn, items):
  while len(pool) < len(items) and len(pool) < 64:
    t = threading.Thread(
        target=_run, name='parallel%d' % len(pool), args=(queue_in, queue_out))
    t.daemon = True
    t.start()
    pool.append(t)

  for index, item in enumerate(items):
    queue_in.put((index, fn, item))
  out = [None] * len(items)
  e = None
  for _ in xrange(len(items)):
    index, result = queue_out.get()
    if index < 0:
      # This is an exception.
      if not e:
        e = result
      else:
        logging.debug(
            'pmap(): discarding exception for item %d: %s',
            -index - 1, result[1])
    else:
      out[index] = result
  if e:
    raise e[0], e[1], e[2]
  return out


def _run(queue_in, queue_out):
  while True:
    index, fn, item = queue_in.get()
    try:
      result = fn(item)
    except:  # pylint: disable=bare-except
      # Starts at -1 otherwise -0 == 0.
      index = -index - 1
      result = sys.exc_info()
    finally:
      queue_out.put((index, result))
