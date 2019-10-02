# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities for operating on persistent disk snapshots."""

import logging

from google.appengine.ext import ndb

from components import gce

import models
import utilities


def fetch(key):
  """Gets snapshots referenced by the given instance template revision.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.

  Returns:
    A list of snapshot URLs.
  """
  itr = key.get()
  if not itr:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return []

  if not itr.project:
    logging.warning('InstanceTemplateRevision project unspecified: %s', key)
    return []

  # No snapshots configured.
  if not itr.snapshot_name and not itr.snapshot_labels:
    return []

  labels = {}
  for label in itr.snapshot_labels:
    # label is necessarily in this format per config.py.
    key, value = label.split(':', 1)
    labels[key] = value

  api = gce.Project(itr.project)
  result = api.get_snapshots(itr.snapshot_name, labels, max_results=500)
  snapshot_urls = [i['selfLink'] for i in result.get('items', [])]
  while result.get('nextPageToken'):
    result = api.get_snapshots(
        itr.snapshot_name, labels, max_results=500,
        page_token=result['nextPageToken'])
    snapshot_urls.extend([i['selfLink'] for i in result['items']])

  return snapshot_urls


@ndb.transactional
def set_snapshot(key, url):
  """Sets the snapshot for the given instance template revision.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.
    url: URL for the snapshot to use for instances created from this template.
  """
  itr = key.get()
  if not itr:
    logging.warning('InstanceTemplateRevision does not exist: %s', key)
    return

  if itr.snapshot_url == url:
    return

  logging.info('Updating snapshot (%s -> %s)', itr.snapshot_url, url)
  itr.snapshot_url = url
  itr.put()


def derive_snapshot(key):
  """Derives the current snapshot for the given InstanceTemplateRevision.

  Args:
    key: ndb.Key for a models.InstanceTemplateRevision entity.
  """
  urls = fetch(key)
  if not urls:
    return
  # TODO(smut): Break ties by creationTimestamp.
  set_snapshot(key, urls[0])


def schedule_fetch():
  """Enqueues tasks to fetch snapshots."""
  for itr in models.InstanceTemplateRevision.query():
    if itr.snapshot_name or itr.snapshot_labels:
      utilities.enqueue_task('fetch-snapshots', itr.key)
