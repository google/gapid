# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Generates statistics out of logs. Contains the backend code.

The first 100mb of logs read is free. It's important to keep logs concise also
for general performance concerns. Each http handler should strive to do only one
log entry at info level per request.
"""

import datetime
import logging

from google.appengine.api import app_identity
from google.appengine.api import memcache
from google.appengine.ext import ndb

from components import stats_framework
from components.stats_framework import stats_logs
from components import net
from components import utils

import bqh

from proto import isolated_pb2


### Models


class _Snapshot(ndb.Model):
  """A snapshot of statistics, to be embedded in another entity."""
  # Number of individual uploads and total amount of bytes. Same for downloads.
  uploads = ndb.IntegerProperty(default=0, indexed=False)
  uploads_bytes = ndb.IntegerProperty(default=0, indexed=False)
  downloads = ndb.IntegerProperty(default=0, indexed=False)
  downloads_bytes = ndb.IntegerProperty(default=0, indexed=False)

  # Number of /contains requests and total number of items looked up.
  contains_requests = ndb.IntegerProperty(default=0, indexed=False)
  contains_lookups = ndb.IntegerProperty(default=0, indexed=False)

  # Total number of requests to calculate QPS
  requests = ndb.IntegerProperty(default=0, indexed=False)
  # Number of non-200 requests.
  failures = ndb.IntegerProperty(default=0, indexed=False)

  def accumulate(self, rhs):
    return stats_framework.accumulate(self, rhs, [])


class BqStateStats(ndb.Model):
  """Stores the last BigQuery successful writes.

  Key id: 1

  By storing the successful writes, this enables not having to read from BQ. Not
  having to sync state *from* BQ means one less RPC that could fail randomly.
  """
  # Last time this entity was updated.
  ts = ndb.DateTimeProperty(indexed=False)
  # Timestamp of the last STATS_HANDLER.stats_minute_cls uploaded.
  last = ndb.DateTimeProperty(indexed=False)
  # Timestamp of the STATS_HANDLER.stats_minute_cls previously uploaded that had
  # failed and should be retried.
  failed = ndb.DateTimeProperty(repeated=True, indexed=False)


### Utility


# Text to store for the corresponding actions.
_ACTION_NAMES = ['store', 'return', 'lookup', 'dupe']


def _parse_line(line, values):
  """Updates a _Snapshot instance with a processed statistics line if relevant.
  """
  if line.count(';') < 2:
    return False
  action_id, measurement, _rest = line.split('; ', 2)
  action = _ACTION_NAMES.index(action_id)
  measurement = int(measurement)

  if action == STORE:
    values.uploads += 1
    values.uploads_bytes += measurement
    return True
  elif action == RETURN:
    values.downloads += 1
    values.downloads_bytes += measurement
    return True
  elif action == LOOKUP:
    values.contains_requests += 1
    values.contains_lookups += measurement
    return True
  elif action == DUPE:
    return True
  else:
    return False


def _extract_snapshot_from_logs(start_time, end_time):
  """Returns a _Snapshot from the processed logs for the specified interval.

  The data is retrieved from logservice via stats_framework.
  """
  values = _Snapshot()
  total_lines = 0
  parse_errors = 0
  for entry in stats_logs.yield_entries(start_time, end_time):
    values.requests += 1
    if entry.request.status >= 400:
      values.failures += 1
    for l in entry.entries:
      if _parse_line(l, values):
        total_lines += 1
      else:
        parse_errors += 1
  logging.debug(
      '_extract_snapshot_from_logs(%s, %s): %d lines, %d errors',
      start_time, end_time, total_lines, parse_errors)
  return values


def _to_proto(s):
  """Shorthand to create a proto."""
  out = isolated_pb2.StatsSnapshot()
  snapshot_to_proto(s, out)
  return out


def _send_to_bq(snapshots):
  """Sends the snapshots to BigQuery.

  Returns:
    Timestamps, encoded as strings, of snapshots that failed to be sent
  """
  # See doc/Monitoring.md.
  dataset = 'isolated'
  table_name = 'stats'

  # BigQuery API doc:
  # https://cloud.google.com/bigquery/docs/reference/rest/v2/tabledata/insertAll
  url = (
      'https://www.googleapis.com/bigquery/v2/projects/%s/datasets/%s/tables/'
      '%s/insertAll') % (app_identity.get_application_id(), dataset, table_name)
  payload = {
    'kind': 'bigquery#tableDataInsertAllRequest',
    # Do not fail entire request because of one bad snapshot.
    # We handle invalid rows below.
    'skipInvalidRows': True,
    'ignoreUnknownValues': False,
    'rows': [
      {
        'insertId': s.timestamp_str,
        'json': bqh.message_to_dict(_to_proto(s)),
      } for s in snapshots
    ],
  }
  res = net.json_request(
      url=url, method='POST', payload=payload, scopes=bqh.INSERT_ROWS_SCOPE,
      deadline=600)

  failed = []
  for err in res.get('insertErrors', []):
    t = snapshots[err['index']].timestamp_str
    if not failed:
      # Log the error for the first entry, useful to diagnose schema failure.
      logging.error('Failed to insert row %s: %r', t, err['errors'])
    failed.append(t)
  return failed


### Public API


STATS_HANDLER = stats_framework.StatisticsFramework(
    'global_stats', _Snapshot, _extract_snapshot_from_logs)


# Action to log.
STORE, RETURN, LOOKUP, DUPE = range(4)


def add_entry(action, number, where):
  """Formatted statistics log entry so it can be processed for daily stats.

  The format is simple enough that it doesn't require a regexp for faster
  processing.
  """
  stats_logs.add_entry('%s; %d; %s' % (_ACTION_NAMES[action], number, where))


def snapshot_to_proto(s, out):
  """Converts a stats._Snapshot to isolated_pb2.Snapshot."""
  out.start_time.FromDatetime(s.timestamp)
  v = s.values
  out.uploads = v.uploads
  out.uploads_bytes = v.uploads_bytes
  out.downloads = v.downloads
  out.downloads_bytes = v.downloads_bytes
  out.contains_requests = v.contains_requests
  out.contains_lookups = v.contains_lookups
  out.requests = v.requests
  out.failures = v.failures


def cron_generate_stats():
  """Returns the number of minutes processed."""
  return STATS_HANDLER.process_next_chunk(stats_framework.TOO_RECENT)


def cron_send_to_bq():
  """Sends the statistics generated by cron_generate_stats() to BigQuery.

  It is intentionally a separate cron job because if the cron job would fail in
  the middle, it's possible that some items wouldn't be sent to BQ, causing
  holes.

  To ensure no items is missing, we query the last item in the table, then look
  the last item in the DB, and stream these.

  Logs insert errors and returns a list of timestamps of snapshots that could
  not be inserted.

  This cron job is surprisingly fast, it processes a year of backlog within a
  few hours.

  Returns:
    total number of statistics snapshot sent to BQ.
  """
  total = 0
  start = utils.utcnow()
  state = BqStateStats.get_by_id(1)
  if not state:
    # No saved state found. Find the oldest entity to send.
    oldest = STATS_HANDLER.stats_minute_cls.query(
        ancestor=STATS_HANDLER.root_key).order(
            STATS_HANDLER.stats_minute_cls.key).get()
    if not oldest:
      logging.info('No Stats found!')
      return total

    state = BqStateStats(id=1, ts=start, last=oldest.timestamp)
    state.put()

  if not memcache.add('running', 'yep', time=400, namespace='stats'):
    logging.info('Other cron already running')
    return total
  # At worst if it dies, the cron job will run for a while.
  # At worst if memcache is cleared, two cron job will run concurrently. It's
  # inefficient but it's not going to break.

  try:
    should_stop = start + datetime.timedelta(seconds=300)
    while utils.utcnow() < should_stop:
      if not memcache.get('running', namespace='stats'):
        logging.info('memcache was cleared')
        return total

      # Now find the most recent entity.
      root = STATS_HANDLER.root_key.get()
      if not root:
        logging.error('Internal failure: couldn\'t find root entity')
        return total
      recent = root.timestamp
      if not recent:
        logging.error('Internal failure: root stats entity has no timestamp')
        return total

      # It is guaranteed to be rounded to the minute. Round explicitly because
      # floating point. :/
      size = int(round((recent - state.last).total_seconds() / 60)) + 1
      if size <= 0 and not state.failed:
        return total

      # Send at most 500 items at a time to reduce the risks of failure.
      max_batch = 500
      if size + len(state.failed) > max_batch:
        # There cannot be more than 500 failed pending send.
        size = max_batch - len(state.failed)

      logging.info(
          'Fetching %d entities starting from %s and %d failed backlog',
          size, state.last, len(state.failed))

      keys = [
        STATS_HANDLER.minute_key(state.last + datetime.timedelta(seconds=60*i))
        for i in xrange(size)
      ]
      # Do them last in case they fail again.
      keys.extend(STATS_HANDLER.minute_key(t) for t in state.failed)

      entities = [e for e in ndb.get_multi(keys) if e]
      if not entities:
        logging.error('Entities are missing')
        return total
      logging.info('Sending %d rows', len(entities))
      failed = _send_to_bq(entities)
      if failed:
        logging.error('Failed to insert %s rows', len(failed))
      total += len(entities) - len(failed)

      # The next cron job round will retry the ones that failed.
      state = BqStateStats(
          id=1,
          ts=utils.utcnow(),
          last=state.last + datetime.timedelta(seconds=60*size),
          failed=failed)
      state.put()
  finally:
    memcache.delete('running', namespace='stats')
