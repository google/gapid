#!/usr/bin/python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for snapshots.py."""

import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from test_support import test_case

import instance_templates
import models
import snapshots


class FetchTest(test_case.TestCase):
  """Tests for snapshots.fetch."""

  @staticmethod
  def create_entity(project, snapshot_name, snapshot_labels):
    """Creates a models.InstanceTemplateRevision.

    Args:
      project: The project name to fetch in.
      snapshot_name: The snapshot name to fetch.
      snapshot_labels: A list of snapshot labels to fetch.

    Returns:
      An ndb.Key for a models.InstanceTemplateRevision instance.
    """
    return models.InstanceTemplateRevision(
        project=project,
        snapshot_name=snapshot_name,
        snapshot_labels=snapshot_labels,
    ).put()

  def test_entity_doesnt_exist(self):
    """Ensures nothing happens when the entity doesn't exist."""
    key = ndb.Key(models.InstanceTemplateRevision, 'fake-key')
    urls = snapshots.fetch(key)
    self.failIf(urls)

  def test_project_unspecified(self):
    """Ensures nothing happens when the entity doesn't specify a project."""
    key = self.create_entity(None, 'name', ['key:value'])
    urls = snapshots.fetch(key)
    self.failIf(urls)

  def test_snapshot_unspecified(self):
    """Ensures nothing happens when the entity doesn't specify a snapshot."""
    key = self.create_entity('project', None, [])
    urls = snapshots.fetch(key)
    self.failIf(urls)

  def test_no_snapshots(self):
    """Ensures nothing happens when there are no snapshots."""
    def get_snapshots(*_args, **_kwargs):
      return {}
    self.mock(snapshots.gce.Project, 'get_snapshots', get_snapshots)

    key = self.create_entity('project', 'name', ['key:value'])
    urls = snapshots.fetch(key)
    self.failIf(urls)

  def test_snapshots(self):
    """Ensures snapshots are returned."""
    def get_snapshots(*_args, **_kwargs):
      return {
          'items': [
              {'selfLink': 'url/snapshot'},
          ],
      }
    self.mock(snapshots.gce.Project, 'get_snapshots', get_snapshots)

    key = self.create_entity('project', 'name', ['key:value'])
    expected_urls = ['url/snapshot']
    urls = snapshots.fetch(key)
    self.assertItemsEqual(urls, expected_urls)

  def test_snapshots_with_page_token(self):
    """Ensures all snapshots are returned."""
    def get_snapshots(*_args, **kwargs):
      if kwargs.get('page_token'):
        return {
            'items': [
                {'selfLink': 'url/snapshot-2'},
            ],
        }
      return {
          'items': [
              {'selfLink': 'url/snapshot-1'},
          ],
          'nextPageToken': 'page-token',
      }
    self.mock(snapshots.gce.Project, 'get_snapshots', get_snapshots)

    key = self.create_entity('project', 'name', ['key:value'])
    expected_urls = ['url/snapshot-1', 'url/snapshot-2']
    urls = snapshots.fetch(key)
    self.assertItemsEqual(urls, expected_urls)


if __name__ == '__main__':
  unittest.main()
