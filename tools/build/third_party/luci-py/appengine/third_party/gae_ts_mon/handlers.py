# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import datetime
import logging

import webapp2

from google.appengine.ext import ndb

from infra_libs.ts_mon import shared
from infra_libs.ts_mon.common import interface


def find_gaps(num_iter):
  """Generate integers not present in an iterable of integers.

  Caution: this is an infinite generator.
  """
  next_num = -1
  for n in num_iter:
    next_num += 1
    while next_num < n:
      yield next_num
      next_num += 1
  while True:
    next_num += 1
    yield next_num


def _assign_task_num(time_fn=datetime.datetime.utcnow):
  expired_keys = []
  unassigned = []
  used_task_nums = []
  time_now = time_fn()
  expired_time = time_now - datetime.timedelta(
      seconds=shared.INSTANCE_EXPIRE_SEC)
  for entity in shared.Instance.query():
    # Don't reassign expired task_num right away to avoid races.
    if entity.task_num >= 0:
      used_task_nums.append(entity.task_num)
    # At the same time, don't assign task_num to expired entities.
    if entity.last_updated < expired_time:
      expired_keys.append(entity.key)
      shared.expired_counter.increment()
      logging.debug(
          'Expiring %s task_num %d, inactive for %s',
          entity.key.id(), entity.task_num,
          time_now - entity.last_updated)
    elif entity.task_num < 0:
      shared.started_counter.increment()
      unassigned.append(entity)

  logging.debug('Found %d expired and %d unassigned instances',
                len(expired_keys), len(unassigned))

  used_task_nums = sorted(used_task_nums)
  for entity, task_num in zip(unassigned, find_gaps(used_task_nums)):
    entity.task_num = task_num
    logging.debug('Assigned %s task_num %d', entity.key.id(), task_num)
  futures_unassigned = ndb.put_multi_async(unassigned)
  futures_expired = ndb.delete_multi_async(expired_keys)
  ndb.Future.wait_all(futures_unassigned + futures_expired)
  logging.debug('Committed all changes')


class SendHandler(webapp2.RequestHandler):
  def get(self):
    if self.request.headers.get('X-Appengine-Cron') != 'true':
      self.abort(403)

    with shared.instance_namespace_context():
      _assign_task_num()

    interface.invoke_global_callbacks()


app = webapp2.WSGIApplication([
    (r'/internal/cron/ts_mon/send', SendHandler),
], debug=True)
