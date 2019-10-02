# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines the mapreduces, which are used to do one-off mass updates on entities
and other manually triggered maintenance tasks.

Automatically triggered maintenance tasks should use a task queue on the backend
instead.
"""

import datetime
import json
import logging

from google.appengine.api import app_identity
from google.appengine.ext import ndb

from mapreduce import mapreduce_pipeline
from mapreduce import operation

from components import utils

from server import task_pack
from server import task_result  # Needed for entity.get()


# Base path to the mapreduce pipeline.
MAPREDUCE_PIPELINE_BASE_PATH = '/internal/mapreduce/pipeline'
# Task queue name to run all map reduce jobs on.
MAPREDUCE_TASK_QUEUE = 'mapreduce-jobs'


# Registered mapreduce jobs, displayed on admin page.
MAPREDUCE_JOBS = {
  'backfill_tags': {
    'job_name': 'Backfill tags',
    'mapper_spec': 'mapreduce_jobs.backfill_tags',
    'mapper_params': {
      'entity_kind': 'server.task_result.TaskResultSummary',
    },
  },
  'fix_tags': {
    'job_name': 'fix_tags',
    'mapper_spec': 'mapreduce_jobs.fix_tags',
    'mapper_params': {
      'entity_kind': 'server.task_result.TaskResultSummary',
    },
  },
  'delete_old': {
    'job_name': 'delete_old',
    'mapper_spec': 'mapreduce_jobs.delete_old',
    'mapper_params': {
      'entity_kind': 'server.task_request.TaskRequest',
    },
  },
  'find_ios_infrastructure_failures': {
    'job_name': 'Find iOS Infrastructure Failures',
    'mapper_spec': 'mapreduce_jobs.map_ios_infrastructure_failures',
    'reducer_spec': 'mapreduce_jobs.reduce_ios_infrastructure_failures',
    'mapper_params': {
      'entity_kind': 'server.task_result.TaskResultSummary',
    },
  },
}


def launch_job(job_id):
  """Launches a job given its key from MAPREDUCE_JOBS dict."""
  assert job_id in MAPREDUCE_JOBS, 'Unknown mapreduce job id %s' % job_id
  job_def = MAPREDUCE_JOBS[job_id].copy()
  # 256 helps getting things done faster but it is very easy to burn thousands
  # of $ within a few hours. Don't forget to update queue.yaml accordingly.
  job_def.setdefault('shards', 128)
  job_def.setdefault(
      'input_reader_spec', 'mapreduce.input_readers.DatastoreInputReader')
  job_def['mapper_params'] = job_def['mapper_params'].copy()
  job_def['mapper_params'].setdefault(
      'bucket_name', app_identity.get_default_gcs_bucket_name())

  if 'reducer_spec' in job_def:
    logging.info('Starting mapreduce job')
    pipeline = mapreduce_pipeline.MapreducePipeline(**job_def)
  else:
    logging.info('Starting mapper-only job')
    job_def['params'] = job_def.pop('mapper_params')
    pipeline = mapreduce_pipeline.MapPipeline(**job_def)

  pipeline.start(
      base_path=MAPREDUCE_PIPELINE_BASE_PATH, queue_name=MAPREDUCE_TASK_QUEUE)
  logging.info('Pipeline ID: %s', pipeline.pipeline_id)
  return pipeline.pipeline_id


### Actual mappers


OLD_TASKS_CUTOFF = utils.utcnow() - datetime.timedelta(hours=12)


def backfill_tags(entity):
  # Already handled?
  if entity.tags:
    return

  # TaskRequest is immutable, can be fetched outside the transaction.
  task_request = entity.request_key.get(use_cache=False, use_memcache=False)
  if not task_request or not task_request.tags:
    return

  # Fast path for old entries: do not use transaction, assumes old entities are
  # not being concurrently modified outside of this job.
  if entity.created_ts and entity.created_ts < OLD_TASKS_CUTOFF:
    entity.tags = task_request.tags
    yield operation.db.Put(entity)
    return

  # For recent entries be careful and use transaction.
  def fix_task_result_summary():
    task_result_summary = entity.key.get()
    if task_result_summary and not task_result_summary.tags:
      task_result_summary.tags = task_request.tags
      task_result_summary.put()

  ndb.transaction(fix_task_result_summary, use_cache=False, use_memcache=False)


def fix_tags(entity):
  """Backfills missing tags and fix the ones with an invalid value."""
  request = entity.request_key.get(use_cache=False, use_memcache=False)
  # Compare the two lists of tags.
  if entity.tags != request.tags:
    entity.tags = request.tags
    logging.info('Fixed %s', entity.task_id)
    yield operation.db.Put(entity)


def delete_old(entity):
  key_to_delete = None
  if entity.key.parent():
    # It is a TaskRequestShard, it is very old.
    key_to_delete = entity.key.parent()
  elif not task_pack.request_key_to_result_summary_key(entity.key).get(
      use_cache=False, use_memcache=False):
    # There's a TaskRequest without TaskResultSummary, delete it.
    key_to_delete = entity.key

  if key_to_delete:
    logging.info('Deleting %s: %s', entity.task_id, key_to_delete)
    total = 1
    qo = ndb.QueryOptions(keys_only=True)
    for k in ndb.Query(default_options=qo, ancestor=key_to_delete):
      yield operation.db.Delete(k)
      total += 1
    yield operation.db.Delete(key_to_delete)
    logging.info('Deleted %d entities', total)


# iOS


def map_ios_infrastructure_failures(entity):
  """Finds tasks which were iOS tests with infrastructure failures.

  Args:
    entity: The task_result.TaskResult entity to look at.

  Yields:
    A 2-tuple of (test_name, labels_json). labels_json is a dict of
    various tags associated with the task encoded as a JSON string.
  """
  if entity.created_ts and entity.created_ts < OLD_TASKS_CUTOFF:
    return

  # iOS test runner signals an infrastructure failure with exit code 2.
  if entity.exit_code != 2:
    return

  tags = (tag.split(':', 1) for tag in entity.tags)
  tags_dict = {t[0]: t[1] for t in tags}

  if not tags_dict.get('test'):
    # Not every task is a test run.
    return

  labels = {
    'buildername': None,
    'buildnumber': None,
    'device_type': None,
    'ios_version': None,
    'master': None,
    'platform': None,
    'revision': None,
    'xcode_version': None,
  }

  for label in labels:
    labels[label] = tags_dict.get(label)
  labels['hostname'] = entity.bot_id
  labels['task_id'] = entity.task_id
  labels['test'] = tags_dict['test']

  logging.info('Mapping %s: %s', labels['test'], json.dumps(labels, indent=2))
  yield labels['test'], json.dumps(labels)


def reduce_ios_infrastructure_failures(test_name, labels_json_list):
  """Counts iOS test infrastructure failures.

  Args:
    test_name: Name of the test which had an infrastructure failure.
    labels_json_list: List of dicts of various tags associated with
      the task, each one individually encoded as a JSON string.
  """
  for labels_json in labels_json_list:
    try:
      labels = json.loads(labels_json)
      logging.info('Reducing %s: %s', test_name,
                   json.dumps(labels, indent=2, separators=(',', ': ')))
      # TODO(smut): Do something with this information. Store it somewhere.
    except (IOError, ValueError):
      logging.error('Invalid labels for %s: %s', test_name, labels_json)
