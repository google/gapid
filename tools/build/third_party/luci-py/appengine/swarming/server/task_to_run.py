# coding=utf-8
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Task entity that describe when a task is to be scheduled.

This module doesn't do the scheduling itself. It only describes the tasks ready
to be scheduled.

Graph of the schema:

    +--------Root-------+
    |TaskRequest        |                                        task_request.py
    |  +--------------+ |
    |  |TaskProperties| |
    |  |  +--------+  | |
    |  |  |FilesRef|  | |
    |  |  +--------+  | |
    |  +--------------+ |
    |id=<based on epoch>|
    +-------------------+
        |
        |
        v
    +--------------+     +--------------+
    |TaskToRun     | ... |TaskToRun     |
    |id=<composite>| ... |id=<composite>|
    +--------------+     +--------------+
"""

import collections
import datetime
import logging
import time

from google.appengine.runtime import apiproxy_errors
from google.appengine.api import memcache
from google.appengine.ext import ndb

from components import utils
from server import bot_management
from server import config
from server import task_pack
from server import task_queues
from server import task_request


### Models.


class TaskToRun(ndb.Model):
  """Defines a TaskRequest ready to be scheduled on a bot.

  This specific request for a specific task can be executed multiple times,
  each execution will create a new child task_result.TaskResult of
  task_result.TaskResultSummary.

  This entity must be kept small and contain the minimum data to enable the
  queries for two reasons:
  - it is updated inside a transaction for each scheduling event, e.g. when a
    bot gets assigned this task item to work on.
  - all the ones currently active are fetched at once in a cron job.

  The key id is:
  - lower 4 bits is the try number. The only supported values are 1 and 2.
  - next 5 bits are TaskResultSummary.current_task_slice (shifted by 4 bits).
  - rest is 0.
  """
  # This entity is used in transactions. It is not worth using either cache.
  # https://cloud.google.com/appengine/docs/standard/python/ndb/cache
  _use_cache = False
  _use_memcache = False

  # Used to know when retries are enqueued. The very first TaskToRun
  # (try_number=1) has the same value as TaskRequest.created_ts, but following
  # ones have created_ts set at the time the new entity is created: when the
  # task is reenqueued.
  created_ts = ndb.DateTimeProperty(indexed=False)

  # Moment by which this TaskSlice has to be requested by a bot.
  # expiration_ts is based on TaskSlice.expiration_ts. This is used to figure
  # out TaskSlice fallback and enable a cron job query to clean up stale tasks.
  expiration_ts = ndb.DateTimeProperty(required=True)

  # Everything above is immutable, everything below is mutable.

  # priority and request creation timestamp are mixed together to allow queries
  # to order the results by this field to allow sorting by priority first, and
  # then timestamp. See _gen_queue_number() for details. This value is only set
  # when the task is available to run, i.e.
  # ndb.TaskResult.query(ancestor=self.key).get().state==AVAILABLE.
  # If this task it not ready to be scheduled, it must be None.
  queue_number = ndb.IntegerProperty()

  @property
  def task_slice_index(self):
    """Returns the TaskRequest.task_slice() index this entity represents as
    pending.
    """
    return task_to_run_key_slice_index(self.key)

  @property
  def try_number(self):
    """Returns the try number, 1 or 2."""
    return task_to_run_key_try_number(self.key)

  @property
  def is_reapable(self):
    """Returns True if the task is ready to be scheduled."""
    return bool(self.queue_number)

  @property
  def request_key(self):
    """Returns the TaskRequest ndb.Key that is parent to the task to run."""
    return task_to_run_key_to_request_key(self.key)

  def to_dict(self):
    """Purely used for unit testing."""
    out = super(TaskToRun, self).to_dict()
    # Consistent formatting makes it easier to reason about.
    if out['queue_number']:
      out['queue_number'] = '0x%016x' % out['queue_number']
    out['try_number'] = self.try_number
    out['task_slice_index'] = self.task_slice_index
    return out


### Private functions.


def _gen_queue_number(dimensions_hash, timestamp, priority):
  """Generates a 63 bit packed value used for TaskToRun.queue_number.

  Arguments:
  - dimensions_hash: 32 bit integer to classify in a queue.
  - timestamp: datetime.datetime when the TaskRequest was filed in. This value
        is used for FIFO or LIFO ordering (depending on configuration) with a
        100ms granularity; the year is ignored.
  - priority: priority of the TaskRequest. It's a 8 bit integer. Lower is higher
        priority.

  Returns:
    queue_number is a 63 bit integer with dimension_hash, timestamp at 100ms
    resolution plus priority.
  """
  # dimensions_hash should be 32 bits but on AppEngine, which is using 32 bits
  # python, it is silently upgraded to long.
  assert isinstance(dimensions_hash, (int, long)), repr(dimensions_hash)
  assert dimensions_hash > 0 and dimensions_hash <= 0xFFFFFFFF, hex(
      dimensions_hash)
  assert isinstance(timestamp, datetime.datetime), repr(timestamp)
  task_request.validate_priority(priority)

  # Ignore the year.

  if config.settings().use_lifo:
    next_year = datetime.datetime(timestamp.year + 1, 1, 1)
    # It is guaranteed to fit 32 bits but upgrade to long right away to ensure
    # assert works.
    t = long(round((next_year - timestamp).total_seconds() * 10.))
  else:
    year_start = datetime.datetime(timestamp.year, 1, 1)
    # It is guaranteed to fit 32 bits but upgrade to long right away to ensure
    # assert works.
    t = long(round((timestamp - year_start).total_seconds() * 10.))

  assert t >= 0 and t <= 0x7FFFFFFF, (
      hex(t), dimensions_hash, timestamp, priority)
  # 31-22 == 9, leaving room for overflow with the addition.
  # 0x3fc00000 is the priority mask.
  # It is important that priority mixed with time is an addition, not a bitwise
  # or.
  low_part = (long(priority) << 22) + t
  assert low_part >= 0 and low_part <= 0xFFFFFFFF, '0x%X is out of band' % (
      low_part)
  # int may be 32 bits, upgrade to long for consistency, albeit this is
  # significantly slower.
  high_part = long(dimensions_hash) << 31
  return high_part | low_part


def _queue_number_order_priority(v):
  """Returns the number to be used as a comparison for priority.

  Lower values are more important. The queue priority is the lowest 31 bits,
  of which the top 9 bits are the task priority, and the rest is the timestamp
  which may overflow in the task priority.
  """
  return v.queue_number & 0x7FFFFFFF


def _queue_number_priority(v):
  """Returns the task's priority.

  There's an overflow of 1 bit, as part of the timestamp overflows on the laster
  part of the year, so the result is between 0 and 330. See _gen_queue_number()
  for the details.
  """
  return int(_queue_number_order_priority(v) >> 22)


def _memcache_to_run_key(to_run_key):
  """Encodes the key as a string to uniquely address the TaskToRun in the
  negative cache in memcache.

  See set_lookup_cache() for more explanation.
  """
  request_key = task_to_run_key_to_request_key(to_run_key)
  return '%x-%d-%d' % (
      request_key.integer_id(),
      task_to_run_key_try_number(to_run_key),
      task_to_run_key_slice_index(to_run_key))


@ndb.tasklet
def _lookup_cache_is_taken_async(to_run_key):
  """Queries the quick lookup cache to reduce DB operations."""
  key = _memcache_to_run_key(to_run_key)
  neg = yield ndb.get_context().memcache_get(key, namespace='task_to_run')
  raise ndb.Return(bool(neg))


class _QueryStats(object):
  """Statistics for a yield_next_available_task_to_dispatch() loop."""
  broken = 0
  cache_lookup = 0
  deadline = None
  expired = 0
  hash_mismatch = 0
  ignored = 0
  no_queue = 0
  real_mismatch = 0
  too_long = 0
  total = 0

  def __str__(self):
    return (
        '%d total, %d exp %d no_queue, %d hash mismatch, %d cache negative, '
        '%d dimensions mismatch, %d ignored, %d broken, '
        '%d not executable by deadline (UTC %s)') % (
        self.total,
        self.expired,
        self.no_queue,
        self.hash_mismatch,
        self.cache_lookup,
        self.real_mismatch,
        self.ignored,
        self.broken,
        self.too_long,
        self.deadline)


@ndb.tasklet
def _validate_task_async(bot_dimensions, deadline, stats, now, to_run):
  """Validates the TaskToRun and updates stats.

  Returns:
    None if the task cannot be reaped by this bot.
    TaskRequest if this is a good candidate to reap.
  """
  # TODO(maruel): Create one TaskToRun per TaskRunResult.
  packed = task_pack.pack_request_key(
      task_to_run_key_to_request_key(to_run.key)) + '0'
  stats.total += 1

  # Do this after the basic weeding out but before fetching TaskRequest.
  neg = yield _lookup_cache_is_taken_async(to_run.key)
  if neg:
    logging.debug('_validate_task_async(%s): negative cache', packed)
    stats.cache_lookup += 1
    raise ndb.Return((None, None))

  # Ok, it's now worth taking a real look at the entity.
  request = yield task_to_run_key_to_request_key(to_run.key).get_async()
  props = request.task_slice(to_run.task_slice_index).properties

  # The hash may have conflicts. Ensure the dimensions actually match by
  # verifying the TaskRequest.
  #
  # There's a probability of 2**-31 of conflicts, which is low enough for our
  # purpose.
  if not match_dimensions(props.dimensions, bot_dimensions):
    logging.debug('_validate_task_async(%s): dimensions mismatch', packed)
    stats.real_mismatch += 1
    raise ndb.Return((None, None))

  # If the bot has a deadline, don't allow it to reap the task unless it can be
  # completed before the deadline. We have to assume the task takes the
  # theoretical maximum amount of time possible, which is governed by
  # execution_timeout_secs. An isolated task's download phase is not subject to
  # this limit, so we need to add io_timeout_secs. When a task is signalled that
  # it's about to be killed, it receives a grace period as well.
  # grace_period_secs is given by run_isolated to the task execution process, by
  # task_runner to run_isolated, and by bot_main to the task_runner. Lastly, add
  # a few seconds to account for any overhead.
  #
  # Give an exemption to the special terminate task because it doesn't actually
  # run anything.
  if deadline is not None and not props.is_terminate:
    if not props.execution_timeout_secs:
      # Task never times out, so it cannot be accepted.
      logging.debug(
          '_validate_task_async(%s): deadline %s but no execution timeout',
          packed, deadline)
      stats.too_long += 1
      raise ndb.Return((None, None))
    hard = props.execution_timeout_secs
    grace = 3 * (props.grace_period_secs or 30)
    # Allowance buffer for overheads (scheduling and isolation)
    overhead = 300
    max_schedule = now + datetime.timedelta(seconds=hard + grace + overhead)
    if deadline <= max_schedule:
      logging.debug(
          '_validate_task_async(%s): deadline and too late %s > %s (%s + %d + '
          '%d + %d)',
          packed, deadline, max_schedule, now, hard, grace, overhead)
      stats.too_long += 1
      raise ndb.Return((None, None))

  # Expire as the bot polls by returning it, and task_scheduler will handle it.
  if to_run.expiration_ts < now:
    logging.debug(
        '_validate_task_async(%s): expired %s < %s',
        packed, to_run.expiration_ts, now)
    stats.expired += 1
  else:
    # It's a valid task! Note that in the meantime, another bot may have reaped
    # it. This is verified one last time in task_scheduler._reap_task() by
    # calling set_lookup_cache().
    logging.info('_validate_task_async(%s): ready to reap!', packed)
  raise ndb.Return((request, to_run))


def _yield_pages_async(q, size):
  """Given a ndb.Query, yields ndb.Future that returns pages of results
  asynchronously.
  """
  next_cursor = [None]
  should_continue = [True]
  def fire(page, result):
    results, cursor, more = page.get_result()
    result.set_result(results)
    next_cursor[0] = cursor
    should_continue[0] = more

  while should_continue[0]:
    page_future = q.fetch_page_async(size, start_cursor=next_cursor[0])
    result_future = ndb.Future()
    page_future.add_immediate_callback(fire, page_future, result_future)
    yield result_future
    result_future.get_result()


def _get_task_to_run_query(dimensions_hash):
  """Returns a ndb.Query of TaskToRun within this dimensions_hash queue."""
  # dimensions_hash should be 32 bits but on AppEngine, which is using 32 bits
  # python, it is silently upgraded to long.
  assert isinstance(dimensions_hash, (int, long)), repr(dimensions_hash)
  opts = ndb.QueryOptions(deadline=15)
  # See _gen_queue_number() as of why << 31. This query cannot use the key
  # because it is not a root entity.
  return TaskToRun.query(default_options=opts).order(
          TaskToRun.queue_number).filter(
              TaskToRun.queue_number >= (dimensions_hash << 31),
              TaskToRun.queue_number < ((dimensions_hash+1) << 31))


def _yield_potential_tasks(bot_id):
  """Queries all the known task queues in parallel and yields the task in order
  of priority.

  The ordering is opportunistic, not strict. There's a risk of not returning
  exactly in the priority order depending on index staleness and query execution
  latency. The number of queries is unbounded.

  Yields:
    TaskToRun entities, trying to yield the highest priority one first. To have
    finite execution time, starts yielding results once one of these conditions
    are met:
    - 1 second elapsed; in this case, continue iterating in the background
    - First page of every query returned
    - All queries exhausted
  """
  bot_root_key = bot_management.get_root_key(bot_id)
  potential_dimensions_hashes = task_queues.get_queues(bot_root_key)
  # Note that the default ndb.EVENTUAL_CONSISTENCY is used so stale items may be
  # returned. It's handled specifically by consumers of this function.
  start = time.time()
  queries = [_get_task_to_run_query(d) for d in potential_dimensions_hashes]
  yielders = [_yield_pages_async(q, 10) for q in queries]
  # We do care about the first page of each query so we cannot merge all the
  # results of every query insensibly.
  futures = []

  try:
    for y in yielders:
      futures.append(next(y, None))

    while (time.time() - start) < 1 and not all(f.done() for f in futures if f):
      r = ndb.eventloop.run0()
      if r is None:
        break
      time.sleep(r)
    logging.debug(
        '_yield_potential_tasks(%s): waited %.3fs for %d items from %d Futures',
        bot_id, time.time() - start,
        sum(len(f.get_result()) for f in futures if f.done()),
        len(futures))
    # items is a list of TaskToRun. The entities are needed because property
    # queue_number is used to sort according to each task's priority.
    items = []
    for i, f in enumerate(futures):
      if f and f.done():
        # The ndb.Future returns a list of up to 10 TaskToRun entities.
        r = f.get_result()
        if r:
          # The ndb.Query ask for a valid queue_number but under load, it
          # happens the value is not valid anymore.
          items.extend(i for i in r if i.queue_number)
          # Prime the next page, in case.
          futures[i] = next(yielders[i], None)

    # That's going to be our search space for now.
    items.sort(key=_queue_number_order_priority)

    # It is possible that there is no items yet, in case all futures are taking
    # more than 1 second.
    # It is possible that all futures are done if every queue has less than 10
    # task pending.
    while any(futures) or items:
      if items:
        yield items[0]
        items = items[1:]
      else:
        # Let activity happen.
        ndb.eventloop.run1()
      changed = False
      for i, f in enumerate(futures):
        if f and f.done():
          # See loop above for explanation.
          items.extend(i for i in f.get_result() if i.queue_number)
          futures[i] = next(yielders[i], None)
          changed = True
      if changed:
        items.sort(key=_queue_number_order_priority)
  except apiproxy_errors.DeadlineExceededError as e:
    # This is normally due to: "The API call datastore_v3.RunQuery() took too
    # long to respond and was cancelled."
    # At that point, the Cloud DB index is not able to sustain the load. So the
    # best thing to do is to back off a bit and not return any task to the bot
    # for this poll.
    logging.error(
        'Failed to yield a task due to an RPC timeout. Returning no '
        'task to the bot: %s', e)


### Public API.


def request_to_task_to_run_key(request, try_number, task_slice_index):
  """Returns the ndb.Key for a TaskToRun from a TaskRequest."""
  assert 1 <= try_number <= 2, try_number
  assert 0 <= task_slice_index < request.num_task_slices
  return ndb.Key(
      TaskToRun, try_number | (task_slice_index << 4), parent=request.key)


def task_to_run_key_to_request_key(to_run_key):
  """Returns the ndb.Key for a TaskToRun from a TaskRequest key."""
  assert to_run_key.kind() == 'TaskToRun', to_run_key
  return to_run_key.parent()


def task_to_run_key_slice_index(to_run_key):
  """Returns the TaskRequest.task_slice() index this TaskToRun entity represents
  as pending.
  """
  return to_run_key.integer_id() >> 4


def task_to_run_key_try_number(to_run_key):
  """Returns the try number, 1 or 2."""
  return to_run_key.integer_id() & 15


def new_task_to_run(request, try_number, task_slice_index):
  """Returns a fresh new TaskToRun for the task ready to be scheduled.

  Returns:
    Unsaved TaskToRun entity.
  """
  assert 1 <= try_number <= 2, try_number
  assert 0 <= task_slice_index < 64, task_slice_index
  created = request.created_ts
  if try_number != 1 or task_slice_index:
    # When retrying, use the current time.
    created = utils.utcnow()
  # TODO(maruel): expiration_ts is based on request.created_ts but it could be
  # enqueued sooner or later. crbug.com/781021
  offset = 0
  if task_slice_index:
    for i in xrange(task_slice_index):
      offset += request.task_slice(i).expiration_secs
  exp = request.created_ts + datetime.timedelta(
      seconds=request.task_slice(task_slice_index).expiration_secs+offset)
  h = request.task_slice(task_slice_index).properties.dimensions
  qn = _gen_queue_number(
      task_queues.hash_dimensions(h), request.created_ts, request.priority)
  return TaskToRun(
      key=request_to_task_to_run_key(request, try_number, task_slice_index),
      created_ts=created,
      queue_number=qn,
      expiration_ts=exp)


def match_dimensions(request_dimensions, bot_dimensions):
  """Returns True if the bot dimensions satisfies the request dimensions."""
  assert isinstance(request_dimensions, dict), request_dimensions
  assert isinstance(bot_dimensions, dict), bot_dimensions
  if frozenset(request_dimensions).difference(bot_dimensions):
    return False
  for key, values in request_dimensions.iteritems():
    if not all(v in bot_dimensions.get(key, []) for v in values):
      return False
  return True


def set_lookup_cache(to_run_key, is_available_to_schedule):
  """Updates the quick lookup cache to mark an item as available or not.

  This cache is a blacklist of items that are about to be reaped or are already
  reaped, so it is not worth trying to reap it with a DB transaction. This saves
  on DB contention when a high number (>1000) of concurrent bots with similar
  dimension are reaping tasks simultaneously. In this case, there is a high
  likelihood that multiple concurrent HTTP handlers are trying to reap the exact
  same task simultaneously. This blacklist helps reduce the contention by
  telling the other bots to back off.

  Another reason for this negative cache is that the DB index takes some seconds
  to be updated, which means it can return stale items (e.g. already reaped).

  It can be viewed as a lock, except that the 'lock' is never released, it is
  impliclity released after 15 seconds.

  Returns:
    True if the key was updated, False if was trying to reap and the entry was
    already set.
  """
  # Set the expiration time for items in the negative cache as 15 seconds. This
  # copes with significant index inconsistency but do not clog the memcache
  # server with unneeded keys.
  cache_lifetime = 15

  key = _memcache_to_run_key(to_run_key)
  if is_available_to_schedule:
    # The item is now available, so remove it from memcache.
    memcache.delete(key, namespace='task_to_run')
    return True

  # add() returns True if the entry was added, False otherwise. That's perfect.
  return memcache.add(key, True, time=cache_lifetime, namespace='task_to_run')


def yield_next_available_task_to_dispatch(bot_dimensions, deadline):
  """Yields next available (TaskRequest, TaskToRun) in decreasing order of
  priority.

  Once the caller determines the task is suitable to execute, it must use
  reap_task_to_run(task.key) to mark that it is not to be scheduled anymore.

  Performance is the top most priority here.

  Arguments:
  - bot_dimensions: dimensions (as a dict) defined by the bot that can be
      matched.
  - deadline: UTC timestamp (as an int) that the bot must be able to
      complete the task by. None if there is no such deadline.
  """
  assert len(bot_dimensions['id']) == 1, bot_dimensions
  # List of all the valid dimensions hashed.
  now = utils.utcnow()
  stats = _QueryStats()
  stats.deadline = deadline
  bot_id = bot_dimensions[u'id'][0]
  futures = collections.deque()
  try:
    for ttr in _yield_potential_tasks(bot_id):
      duration = (utils.utcnow() - now).total_seconds()
      if duration > 40.:
        # Stop searching after too long, since the odds of the request blowing
        # up right after succeeding in reaping a task is not worth the dangling
        # task request that will stay in limbo until the cron job reaps it and
        # retry it. The current handlers are given 60s to complete. By limiting
        # search to 40s, it gives 20s to complete the reaping and complete the
        # HTTP request.
        return
      futures.append(
          _validate_task_async(bot_dimensions, deadline, stats, now, ttr))
      while futures:
        # Keep a FIFO or LIFO queue ordering, depending on configuration.
        if futures[0].done():
          request, task = futures[0].get_result()
          if request and task:
            yield request, task
            # If the code is still executed, it means that the task reaping
            # wasn't successful. Note that this includes expired ones, which is
            # kinda weird but it's not a big deal.
            stats.ignored += 1
          futures.popleft()
        # Don't batch too much.
        if len(futures) < 50:
          break
        futures[0].wait()

    # No more tasks to yield. Empty the pending futures.
    while futures:
      request, task = futures[0].get_result()
      if request and task:
        yield request, task
        # If the code is still executed, it means that the task reaping
        # wasn't successful. Same as above about expired.
        stats.ignored += 1
      futures.popleft()
  finally:
    # Don't leave stray RPCs as much as possible, this can mess up following
    # HTTP handlers.
    ndb.Future.wait_all(futures)
    # stats output is a bit misleading here, as many _validate_task_async()
    # could be started yet never yielded.
    logging.debug(
        'yield_next_available_task_to_dispatch(%s) in %.3fs: %s',
        bot_id, (utils.utcnow() - now).total_seconds(), stats)


def yield_expired_task_to_run():
  """Yields all the expired TaskToRun still marked as available."""
  # The reason it is done this way as an iteration over all the pending entities
  # instead of using a composite index with 'queue_number' and 'expiration_ts'
  # is that TaskToRun entities are very hot and it is important to not require
  # composite indexes on it. It is expected that the number of pending task is
  # 'relatively low', in the orders of 100,000 entities.
  #
  # Use a large batch size since the entities are very small and to reduce RPC
  # overhead.
  opts = ndb.QueryOptions(batch_size=256)
  now = utils.utcnow()
  for task in TaskToRun.query(TaskToRun.queue_number > 0, default_options=opts):
    if task.expiration_ts < now:
      yield task
