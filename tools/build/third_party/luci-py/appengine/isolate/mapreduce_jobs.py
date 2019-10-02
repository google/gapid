# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines the mapreduces, which are used to do one-off mass updates on entities
and other manually triggered maintenance tasks.

Automatically triggered maintenance tasks should use a task queue on the backend
instead.
"""

import logging

from google.appengine.api import app_identity

from mapreduce import mapreduce_pipeline

import config
import gcs


# Task queue name to run all map reduce jobs on.
MAPREDUCE_TASK_QUEUE = 'mapreduce-jobs'


# Registered mapreduce jobs, displayed on admin page.
MAPREDUCE_JOBS = {
  'find_missing_gs_files': {
    'job_name': 'Report missing GS files',
    'mapper_spec': 'mapreduce_jobs.find_missing_gs_files',
    'mapper_params': {
      'entity_kind': 'model.ContentEntry',
    },
  },
  'delete_broken_entries': {
    'job_name': 'Delete entries that do not have corresponding GS files',
    'mapper_spec': 'mapreduce_jobs.delete_broken_entries',
    'mapper_params': {
      'entity_kind': 'model.ContentEntry',
    },
  },
}


def launch_job(job_id):
  """Launches a job given its key from MAPREDUCE_JOBS dict."""
  assert job_id in MAPREDUCE_JOBS, 'Unknown mapreduce job id %s' % job_id
  job_def = MAPREDUCE_JOBS[job_id].copy()
  job_def.setdefault('shards', 64)
  job_def.setdefault(
      'input_reader_spec', 'mapreduce.input_readers.DatastoreInputReader')
  job_def['mapper_params'] = job_def['mapper_params'].copy()
  job_def['mapper_params'].setdefault(
      'bucket_name', app_identity.get_default_gcs_bucket_name())

  if 'reducer_spec' in job_def:
    logging.info('Starting mapreduce job')
    pipeline = mapreduce_pipeline.MapreducePipeline(**job_def)
  else:
    logging.info('Starting mapper-only pipeline')
    job_def['params'] = job_def.pop('mapper_params')
    pipeline = mapreduce_pipeline.MapPipeline(**job_def)

  pipeline.start(queue_name=MAPREDUCE_TASK_QUEUE)
  logging.info('Pipeline ID: %s', pipeline.pipeline_id)
  return pipeline.pipeline_id


def is_good_content_entry(entry):
  """True if ContentEntry is not broken.

  ContentEntry is broken if it is in old format (before content namespace
  were sharded) or corresponding Google Storage file doesn't exist.
  """
  # New entries use GS file path as ids. File path is always <namespace>/<hash>.
  entry_id = entry.key.id()
  if '/' not in entry_id:
    return False
  # Content is inline, entity doesn't have GS file attached -> it is fine.
  if entry.content is not None:
    return True
  # Ensure GS file exists.
  return bool(gcs.get_file_info(config.settings().gs_bucket, entry_id))


### Actual mappers


def find_missing_gs_files(entry):
  """Mapper that takes ContentEntry and logs to output if GS file is missing."""
  if not is_good_content_entry(entry):
    logging.error('MR: found bad entry\n%s', entry.key.id())


def delete_broken_entries(entry):
  """Mapper that deletes ContentEntry entities that are broken."""
  if not is_good_content_entry(entry):
    # MR framework disables memcache on a context level. Explicitly request
    # to cleanup memcache, otherwise the rest of the isolate service will still
    # think that entity exists.
    entry.key.delete(use_memcache=True)
    logging.error('MR: deleted bad entry\n%s', entry.key.id())
