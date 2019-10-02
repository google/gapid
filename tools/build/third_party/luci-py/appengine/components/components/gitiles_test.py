#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import datetime
import json
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from test_support import test_case
import mock

from google.appengine.ext import ndb

from components import auth
from components import gerrit
from components import gitiles


HOSTNAME = 'chromium.googlesource.com'
PROJECT = 'project'
REVISION = '404d1697dca23824bc1130061a5bd2be4e073922'
PATH = '/dir'


class GitilesTestCase(test_case.TestCase):
  def setUp(self):
    super(GitilesTestCase, self).setUp()
    self.mock(auth, 'get_access_token', mock.Mock(return_value=('token', 0.0)))

  def mock_fetch(self, result):
    self.mock(gerrit, 'fetch_async', mock.Mock(return_value=ndb.Future()))
    gerrit.fetch_async.return_value.set_result(result)

  def mock_fetch_json(self, result):
    self.mock(gerrit, 'fetch_json_async', mock.Mock(return_value=ndb.Future()))
    gerrit.fetch_json_async.return_value.set_result(result)

  def test_parse_time(self):
    time_str = 'Fri Nov 07 17:09:03 2014'
    expected = datetime.datetime(2014, 11, 07, 17, 9, 3)
    actual = gitiles.parse_time(time_str)
    self.assertEqual(expected, actual)

  def test_parse_time_with_positive_timezone(self):
    time_str = 'Fri Nov 07 17:09:03 2014 +01:00'
    expected = datetime.datetime(2014, 11, 07, 16, 9, 3)
    actual = gitiles.parse_time(time_str)
    self.assertEqual(expected, actual)

  def test_parse_time_with_negative_timezone(self):
    time_str = 'Fri Nov 07 17:09:03 2014 -01:00'
    expected = datetime.datetime(2014, 11, 07, 18, 9, 3)
    actual = gitiles.parse_time(time_str)
    self.assertEqual(expected, actual)

  def test_get_commit(self):
    req_path = 'project/+/%s' % REVISION
    self.mock_fetch_json({
      'commit': REVISION,
      'tree': '3cfb41e1c6c37e61c3eccfab2395752298a5743c',
      'parents': [
        '4087678c002d57e1148f21da5e00867df9a7d973',
      ],
      'author': {
        'name': 'John Doe',
        'email': 'john.doe@chromium.org',
        'time': 'Tue Apr 29 00:00:00 2014',
      },
      'committer': {
        'name': 'John Doe',
        'email': 'john.doe@chromium.org',
        'time': 'Tue Apr 29 00:00:00 2014',
      },
      'message': 'Subject\\n\\nBody',
      'tree_diff': [
        {
          'type': 'modify',
          'old_id': 'f23dec7da271f7e9d8c55a35f32f6971b7ce486d',
          'old_mode': 33188,
          'old_path': 'codereview.settings',
          'new_id': '0bdbda926c49aa3cc4b7248bc22cc261abff5f94',
          'new_mode': 33188,
          'new_path': 'codereview.settings',
        },
        {
          'type': 'add',
          'old_id': '0000000000000000000000000000000000000000',
          'old_mode': 0,
          'old_path': '/dev/null',
          'new_id': 'e69de29bb2d1d6434b8b29ae775ad8c2e48c5391',
          'new_mode': 33188,
          'new_path': 'x',
        }
      ],
    })

    commit = gitiles.get_commit(HOSTNAME, PROJECT, REVISION)
    gerrit.fetch_json_async.assert_called_once_with(HOSTNAME, req_path)
    self.assertIsNotNone(commit)
    self.assertEqual(commit.sha, REVISION)
    self.assertEqual(commit.committer.name, 'John Doe')
    self.assertEqual(commit.committer.email, 'john.doe@chromium.org')
    self.assertEqual(commit.author.name, 'John Doe')
    self.assertEqual(commit.author.email, 'john.doe@chromium.org')

  def test_get_tree(self):
    req_path = 'project/+/deadbeef/dir'
    self.mock_fetch_json({
        'id': 'c244aa92a18cd719c55205f99e04333840330012',
        'entries': [
          {
            'id': '0244aa92a18cd719c55205f99e04333840330012',
            'name': 'a',
            'type': 'blob',
            'mode': 33188,
          },
          {
            'id': '9c247a8aa968a3e2641addf1f4bd4acfc24e7915',
            'name': 'b',
            'type': 'blob',
            'mode': 33188,
          },
        ],
    })

    tree = gitiles.get_tree(HOSTNAME, 'project', 'deadbeef', '/dir')
    gerrit.fetch_json_async.assert_called_once_with(HOSTNAME, req_path)
    self.assertIsNotNone(tree)
    self.assertEqual(tree.id, 'c244aa92a18cd719c55205f99e04333840330012')
    self.assertEqual(
        tree.entries[0].id, '0244aa92a18cd719c55205f99e04333840330012')
    self.assertEqual(tree.entries[0].name, 'a')

  def test_get_log(self):
    req_path = 'project/+log/master/'
    self.mock_fetch_json({
      'log': [
        {
          'commit': REVISION,
          'tree': '3cfb41e1c6c37e61c3eccfab2395752298a5743c',
          'parents': [
            '4087678c002d57e1148f21da5e00867df9a7d973',
          ],
          'author': {
            'name': 'John Doe',
            'email': 'john.doe@chromium.org',
            'time': 'Tue Apr 29 00:00:00 2014',
          },
          'committer': {
            'name': 'John Doe',
            'email': 'john.doe@chromium.org',
            'time': 'Tue Apr 29 00:00:00 2014',
          },
          'message': 'Subject\\n\\nBody',
          'tree_diff': [],
        },
        {
          'commit': '4087678c002d57e1148f21da5e00867df9a7d973',
          'tree': '3cfb41asdc37e61c3eccfab2395752298a5743c',
          'parents': [
            '1237678c002d57e1148f21da5e00867df9a7d973',
          ],
          'author': {
            'name': 'John Doe',
            'email': 'john.doe@chromium.org',
            'time': 'Tue Apr 29 00:00:00 2014',
          },
          'committer': {
            'name': 'John Doe',
            'email': 'john.doe@chromium.org',
            'time': 'Tue Apr 29 00:00:00 2014',
          },
          'message': 'Subject2\\n\\nBody2',
          'tree_diff': [],
        },
      ],
    })

    log = gitiles.get_log(HOSTNAME, 'project', 'master', limit=2)
    gerrit.fetch_json_async.assert_called_once_with(
        HOSTNAME, req_path, params={'n': 2})

    john = gitiles.Contribution(
        name='John Doe', email='john.doe@chromium.org',
        time=datetime.datetime(2014, 4, 29))
    self.assertEqual(
        log,
        gitiles.Log(
            commits=[
              gitiles.Commit(
                  sha=REVISION,
                  tree='3cfb41e1c6c37e61c3eccfab2395752298a5743c',
                  parents=[
                    '4087678c002d57e1148f21da5e00867df9a7d973',
                  ],
                  message='Subject\\n\\nBody',
                  author=john,
                  committer=john,
                  tree_diff=[],
              ),
              gitiles.Commit(
                  sha='4087678c002d57e1148f21da5e00867df9a7d973',
                  tree='3cfb41asdc37e61c3eccfab2395752298a5743c',
                  parents=[
                    '1237678c002d57e1148f21da5e00867df9a7d973',
                  ],
                  message='Subject2\\n\\nBody2',
                  author=john,
                  committer=john,
                  tree_diff=[],
              ),
            ],
            next_cursor=None,
        )
    )

  def test_get_log_with_slash(self):
    req_path = 'project/+log/master/'
    self.mock_fetch_json(None)

    gitiles.get_log(HOSTNAME, 'project', 'master', path='/', limit=2)
    gerrit.fetch_json_async.assert_called_once_with(
        HOSTNAME, req_path, params={'n': 2})

  def test_get_log_with_path(self):
    req_path = 'project/+log/master/x'
    self.mock_fetch_json(None)

    gitiles.get_log(HOSTNAME, 'project', 'master', path='x', limit=2)
    gerrit.fetch_json_async.assert_called_once_with(
        HOSTNAME, req_path, params={'n': 2})

  def test_get_log_with_path_with_space(self):
    req_path = 'project/+log/master/x%20y'
    self.mock_fetch_json(None)

    gitiles.get_log(HOSTNAME, 'project', 'master', path='x y')
    gerrit.fetch_json_async.assert_called_once_with(
        HOSTNAME, req_path, params={})

  def test_get_file_content(self):
    req_path = 'project/+/master/a.txt'
    self.mock_fetch(base64.b64encode('content'))

    content = gitiles.get_file_content(HOSTNAME, 'project', 'master', '/a.txt')
    gerrit.fetch_async.assert_called_once_with(
        HOSTNAME, req_path, headers={'Accept': 'text/plain'})
    self.assertEqual(content, 'content')

  def test_get_archive(self):
    req_path = 'project/+archive/master.tar.gz'
    self.mock_fetch('tar gz bytes')

    content = gitiles.get_archive(HOSTNAME, 'project', 'master')
    gerrit.fetch_async.assert_called_once_with(HOSTNAME, req_path)
    self.assertEqual('tar gz bytes', content)

  def test_get_archive_with_dirpath(self):
    req_path = 'project/+archive/master/dir.tar.gz'
    self.mock_fetch('tar gz bytes')

    content = gitiles.get_archive(HOSTNAME, 'project', 'master', '/dir')
    gerrit.fetch_async.assert_called_once_with(HOSTNAME, req_path)
    self.assertEqual('tar gz bytes', content)

  def test_get_diff(self):
    req_path = 'project/+/deadbeef..master/'
    self.mock_fetch(base64.b64encode('thepatch'))

    patch = gitiles.get_diff(HOSTNAME, 'project', 'deadbeef', 'master', '/')
    self.assertEqual(patch, 'thepatch')

    gerrit.fetch_async.assert_called_once_with(
        HOSTNAME,
        req_path,
        headers={'Accept': 'text/plain'})

  def test_to_from_dict(self):
    loc = gitiles.Location.parse(
        'http://localhost/project/+/treeish/path/to/something')
    self.assertEqual(loc, gitiles.Location.from_dict(loc.to_dict()))

  def test_parse_location(self):
    url = 'http://localhost/project/+/treeish/path/to/something'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.hostname, 'localhost')
    self.assertEqual(loc.project, 'project')
    self.assertEqual(loc.treeish, 'refs/heads/treeish')
    self.assertEqual(loc.path, '/path/to/something')

  def test_parse_location_no_treeish(self):
    url = 'http://localhost/project'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.hostname, 'localhost')
    self.assertEqual(loc.project, 'project')
    self.assertEqual(loc.treeish, 'HEAD')
    self.assertEqual(loc.path, '/')

  def test_parse_refs_heads_master(self):
    url = 'http://localhost/project/+/refs/heads/master/path/to/something'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.hostname, 'localhost')
    self.assertEqual(loc.project, 'project')
    self.assertEqual(loc.treeish, 'refs/heads/master')
    self.assertEqual(loc.path, '/path/to/something')

  def test_parse_authenticated_url(self):
    url = 'http://localhost/a/project/+/treeish/path'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.hostname, 'localhost')
    self.assertEqual(loc.project, 'project')

  def test_parse_location_with_dot(self):
    url = 'http://localhost/project/+/treeish/path'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.treeish, 'refs/heads/treeish')
    self.assertEqual(loc.path, '/path')

  def test_parse_location_with_git_ending(self):
    url = 'http://localhost/project.git/+/treeish/path'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.project, 'project')

  def test_parse_location_hash_treeish(self):
    url = ('http://localhost/project.git/+/'
           'f9af6214956f071d0e541d05e65285b3600079a0/path')
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.treeish, 'f9af6214956f071d0e541d05e65285b3600079a0')

  def test_parse_location_fake_hash_treeish(self):
    url = ('http://localhost/project.git/+/'
           'f9af6214956f071d0e541d05e65285b3600079a0d/path')
    loc = gitiles.Location.parse(url)
    self.assertEqual(
        loc.treeish, 'refs/heads/f9af6214956f071d0e541d05e65285b3600079a0d')

  def test_parse_location_HEAD_treeish(self):
    url = 'http://localhost/project/+/HEAD/path'
    loc = gitiles.Location.parse(url)
    self.assertEqual(loc.treeish, 'HEAD')

  def test_parse_resolve(self):
    self.mock(gitiles, 'get_refs', mock.Mock())
    gitiles.get_refs.return_value = {
      'refs/heads/master': {},
      'refs/heads/a/b': {},
      'refs/tags/c/d': {},
    }
    loc = gitiles.Location.parse_resolve('http://h/p/+/a/b/c')
    self.assertEqual(loc.treeish, 'refs/heads/a/b')
    self.assertEqual(loc.path, '/c')

    loc = gitiles.Location.parse_resolve('http://h/p/+/c/d/e')
    self.assertEqual(loc.treeish, 'refs/heads/c/d')
    self.assertEqual(loc.path, '/e')

    with self.assertRaises(gitiles.TreeishResolutionError):
      gitiles.Location.parse_resolve('http://h/p/+/a/c/b')

  def test_parse_resolve_head(self):
    self.mock(gitiles, 'get_refs', mock.Mock())
    gitiles.get_refs.return_value = {
      'refs/heads/a/b': {},
    }
    loc = gitiles.Location.parse_resolve('http://h/p/+/refs/heads/a/b/c')
    self.assertEqual(loc.treeish, 'refs/heads/a/b')
    self.assertEqual(loc.path, '/c')
    gitiles.get_refs.assert_called_with('h', 'p', 'refs/heads/a/')

    gitiles.get_refs.return_value = {}
    with self.assertRaises(gitiles.TreeishResolutionError):
      gitiles.Location.parse_resolve('http://h/p/+/refs/heads/x/y')
    gitiles.get_refs.assert_called_with('h', 'p', 'refs/heads/x/')

  def test_location_neq(self):
    loc1 = gitiles.Location(
        hostname='localhost', project='project',
        treeish='treeish', path='/path')
    loc2 = gitiles.Location(
        hostname='localhost', project='project',
        treeish='treeish', path='/path')
    self.assertFalse(loc1.__ne__(loc2))

  def test_location_str(self):
    loc = gitiles.Location(
        hostname='localhost', project='project',
        treeish='treeish', path='/path')
    self.assertEqual(loc, 'https://localhost/project/+/treeish/path')

  def test_location_str_with_slash_path(self):
    loc = gitiles.Location(
        hostname='localhost', project='project',
        treeish='treeish', path='/')
    self.assertEqual(loc, 'https://localhost/project/+/treeish')

  def test_location_str_defaults_to_head(self):
    loc = gitiles.Location(
        hostname='localhost', project='project', treeish=None, path='/path')
    self.assertEqual(loc, 'https://localhost/project/+/HEAD/path')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
