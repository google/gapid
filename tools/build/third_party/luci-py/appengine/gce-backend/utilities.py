# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for GCE Backend."""

import hashlib
import json

from google.appengine.ext import ndb

from components import utils


def batch_process_async(items, f, max_concurrent=50):
  """Processes asynchronous calls in parallel, but batched.

  Args:
    items: List of items to process.
    f: f(item) -> ndb.Future. Asynchronous function to apply to each item.
    max_concurrent: Maximum number of futures to have pending concurrently.
  """
  futures = []
  while items:
    num_futures = len(futures)
    if num_futures < max_concurrent:
      futures.extend([f(item) for item in items[:max_concurrent - num_futures]])
      items = items[max_concurrent - num_futures:]
    ndb.Future.wait_any(futures)
    futures = [future for future in futures if not future.done()]
  if futures:
    ndb.Future.wait_all(futures)


def compute_checksum(json_encodable):
  """Computes a checksum from a JSON-encodable dict or list."""
  return hashlib.sha1(json.dumps(json_encodable, sort_keys=True)).hexdigest()


def enqueue_task(taskqueue, key):
  """Enqueues a task for the specified task queue to process the given key.

  Args:
    taskqueue: Name of the task queue.
    key: ndb.Key to pass as a parameter to the task queue.
  """
  utils.enqueue_task(
      '/internal/queues/%s' % taskqueue,
      taskqueue,
      params={
          'key': key.urlsafe(),
      },
  )
