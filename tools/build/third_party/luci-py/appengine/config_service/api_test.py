#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import datetime
import httplib

import test_env
from test_env import future

test_env.setup_test_env()

from components import config
from components.config.proto import project_config_pb2
from components.config.proto import service_config_pb2
from test_support import test_case
import mock

import acl
import api
import gitiles_import
import projects
import storage
import validation


class ApiTest(test_case.EndpointsTestCase):
  api_service_cls = api.ConfigApi
  api_service_regex = '.+'

  def setUp(self):
    super(ApiTest, self).setUp()
    self.mock(acl, 'has_projects_access', mock.Mock())
    acl.has_projects_access.side_effect = (
        lambda pids: {pid: pid != 'secret' for pid in pids}
    )

    self.mock(
        acl, 'has_services_access', lambda sids: {sid: True for sid in sids})
    self.mock(projects, 'get_projects', mock.Mock())
    projects.get_projects.return_value = [
      service_config_pb2.Project(id='chromium'),
      service_config_pb2.Project(id='v8'),
    ]
    self.mock(projects, 'get_metadata_async', mock.Mock(return_value=future({
      'chromium': project_config_pb2.ProjectCfg(),
      'v8': project_config_pb2.ProjectCfg(),
    })))
    self.mock(projects, 'get_repos_async', mock.Mock(return_value=future({
      'chromium': (
          projects.RepositoryType.GITILES, 'https://chromium.example.com'),
      'v8': (
          projects.RepositoryType.GITILES, 'https://v8.example.com'),
    })))

  def mock_config(self, mock_content=True):
    self.mock(storage, 'get_config_hashes_async', mock.Mock())
    storage.get_config_hashes_async.return_value = future({
      'services/luci-config': ('deadbeef', 'https://x.com/+/deadbeef', 'abc0123'),
    })

    if mock_content:
      self.mock(storage, 'get_configs_by_hashes_async', mock.Mock())
      storage.get_configs_by_hashes_async.return_value = future({
        'abc0123': 'config text',
      })

  def mock_refs(self):
    self.mock(projects, 'get_refs', mock.Mock())
    projects.get_refs.return_value = {
      'chromium': [
        project_config_pb2.RefsCfg.Ref(name='refs/heads/master'),
        project_config_pb2.RefsCfg.Ref(name='refs/heads/release42'),
      ],
      'v8': [
        project_config_pb2.RefsCfg.Ref(name='refs/heads/master'),
      ],
    }

  ##############################################################################
  # get_mapping

  def test_get_mapping_one(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(id='services/x', location='https://x'),
    ])

    req = {
      'config_set': 'services/x',
    }
    resp = self.call_api('get_mapping', req).json_body

    storage.get_config_sets_async.assert_called_once_with(
        config_set='services/x')

    self.assertEqual(resp, {
      'mappings': [
        {
          'config_set': 'services/x',
          'location': 'https://x',
        },
      ],
    })

  def test_get_mapping_one_forbidden(self):
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/x': False,
    }))
    with self.call_should_fail(httplib.FORBIDDEN):
      req = {
        'config_set': 'services/x',
      }
      self.call_api('get_mapping', req)

  def test_get_mapping_all(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(id='services/x', location='https://x'),
      storage.ConfigSet(id='services/y', location='https://y'),
    ])
    resp = self.call_api('get_mapping', {}).json_body

    self.assertEqual(resp, {
      'mappings': [
        {
          'config_set': 'services/x',
          'location': 'https://x',
        },
        {
          'config_set': 'services/y',
          'location': 'https://y',
        },
      ],
    })

  def test_get_mapping_all_partially_forbidden(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(id='services/x', location='https://x'),
      storage.ConfigSet(id='services/y', location='https://y'),
    ])
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/x': True,
      'services/y': False,
    }))

    resp = self.call_api('get_mapping', {}).json_body

    self.assertEqual(resp, {
      'mappings': [
        {
          'config_set': 'services/x',
          'location': 'https://x',
        },
      ],
    })

  ##############################################################################
  # get_config_sets

  def test_get_config_one(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(
          id='services/x',
          location='https://x.googlesource.com/x',
          latest_revision='deadbeef',
          latest_revision_url='https://x.googlesource.com/x/+/deadbeef',
          latest_revision_time=datetime.datetime(2016, 1, 1),
          latest_revision_committer_email='john@doe.com',
      ),
    ])

    req = {
      'config_set': 'services/x',
    }
    resp = self.call_api('get_config_sets', req).json_body

    storage.get_config_sets_async.assert_called_once_with(
        config_set='services/x')

    self.assertEqual(resp, {
      'config_sets': [
        {
          'config_set': 'services/x',
          'location': 'https://x.googlesource.com/x',
          'revision': {
            'id': 'deadbeef',
            'url': 'https://x.googlesource.com/x/+/deadbeef',
            'timestamp': '1451606400000000',
            'committer_email': 'john@doe.com',
          },
        },
      ],
    })

  @mock.patch('storage.get_file_keys', autospec=True)
  @mock.patch('storage.get_config_sets_async', autospec=True)
  def test_get_config_with_include_files(
      self, mock_get_config_sets_async, mock_get_file_keys):
    mock_get_config_sets_async.return_value = future([
      storage.ConfigSet(
          id='services/x',
          location='https://x.googlesource.com/x',
          latest_revision='deadbeef',
          latest_revision_url='https://x.googlesource.com/x/+/deadbeef',
          latest_revision_time=datetime.datetime(2016, 1, 1),
          latest_revision_committer_email='john@doe.com',
      ),
    ])

    class Key:
      def __init__(self, name):
        self.name = name

      def id(self):
        return self.name

    mock_get_file_keys.return_value = [
      Key('README.md'),
      Key('rick.morty'),
      Key('pied.piper')
    ]

    req = {
      'config_set': 'services/x',
      'include_files': True,
    }
    resp = self.call_api('get_config_sets', req).json_body

    self.assertEqual(resp, {
      'config_sets': [
        {
          'config_set': 'services/x',
          'files': [
            {
              'path': 'README.md'
            },
            {
              'path': 'rick.morty'
            },
            {
              'path': 'pied.piper'
            }
          ],
          'location': 'https://x.googlesource.com/x',
          'revision': {
            'committer_email': 'john@doe.com',
            'id': 'deadbeef',
            'timestamp': '1451606400000000',
            'url': 'https://x.googlesource.com/x/+/deadbeef'
          }
        }
      ]
    })

  def test_get_config_with_inconsistent_request(self):
    with self.call_should_fail(httplib.BAD_REQUEST):
      req = {
        'include_files': True,
      }
      self.call_api('get_config_sets', req)

  def test_get_config_one_with_last_attempt(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(
          id='services/x',
          location='https://x.googlesource.com/x',
          latest_revision='deadbeef',
          latest_revision_url='https://x.googlesource.com/x/+/deadbeef',
          latest_revision_time=datetime.datetime(2016, 1, 1),
          latest_revision_committer_email='john@doe.com',
      ),
    ])

    storage.ImportAttempt(
        key=storage.last_import_attempt_key('services/x'),
        time=datetime.datetime(2016, 1, 2),
        revision=storage.RevisionInfo(
          id='badcoffee',
          url='https://x.googlesource.com/x/+/badcoffee',
          time=datetime.datetime(2016, 1, 1),
          committer_email='john@doe.com',
        ),
        success=False,
        message='Validation errors',
        validation_messages=[
          storage.ImportAttempt.ValidationMessage(
              path='foo.cfg',
              severity=config.Severity.ERROR,
              text='error!',
          ),
          storage.ImportAttempt.ValidationMessage(
              path='bar.cfg',
              severity=config.Severity.WARNING,
              text='warning!',
          ),
        ],
    ).put()

    req = {
      'config_set': 'services/x',
    }
    resp = self.call_api('get_config_sets', req).json_body

    storage.get_config_sets_async.assert_called_once_with(
        config_set='services/x')

    expected_resp = {
      'config_sets': [
        {
          'config_set': 'services/x',
          'location': 'https://x.googlesource.com/x',
          'revision': {
            'id': 'deadbeef',
            'url': 'https://x.googlesource.com/x/+/deadbeef',
            'timestamp': '1451606400000000',
            'committer_email': 'john@doe.com',
          },
        },
      ],
    }
    self.assertEqual(resp, expected_resp)

    req['include_last_import_attempt'] = True
    resp = self.call_api('get_config_sets', req).json_body
    expected_resp['config_sets'][0]['last_import_attempt'] = {
      'timestamp': '1451692800000000',
      'revision': {
        'id': 'badcoffee',
        'url': 'https://x.googlesource.com/x/+/badcoffee',
        'timestamp': '1451606400000000',
        'committer_email': 'john@doe.com',
      },
      'success': False,
      'message': 'Validation errors',
      'validation_messages': [
        {
          'path': 'foo.cfg',
          'severity': 'ERROR',
          'text': 'error!',
        },
        {
          'path': 'bar.cfg',
          'severity': 'WARNING',
          'text': 'warning!',
        },
      ]
    }
    self.assertEqual(resp, expected_resp)

  def test_get_config_one_forbidden(self):
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/x': False,
    }))
    with self.call_should_fail(httplib.FORBIDDEN):
      req = {
        'config_set': 'services/x',
      }
      self.call_api('get_config_sets', req)

  def test_get_config_all(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(
          id='services/x',
          location='https://x.googlesource.com/x',
          latest_revision='deadbeef',
          latest_revision_url='https://x.googlesource.com/x/+/deadbeef',
          latest_revision_time=datetime.datetime(2016, 1, 1),
          latest_revision_committer_email='john@doe.com',
      ),
      storage.ConfigSet(
          id='projects/y',
          location='https://y.googlesource.com/y',
          latest_revision='badcoffee',
          latest_revision_url='https://y.googlesource.com/y/+/badcoffee',
          latest_revision_time=datetime.datetime(2016, 1, 2),
          latest_revision_committer_email='john@doe.com',
      ),
    ])

    resp = self.call_api('get_config_sets', {}).json_body

    storage.get_config_sets_async.assert_called_once_with(config_set=None)

    self.assertEqual(resp, {
      'config_sets': [
        {
          'config_set': 'services/x',
          'location': 'https://x.googlesource.com/x',
          'revision': {
            'id': 'deadbeef',
            'url': 'https://x.googlesource.com/x/+/deadbeef',
            'timestamp': '1451606400000000',
            'committer_email': 'john@doe.com',
          },
        },
        {
          'config_set': 'projects/y',
          'location': 'https://y.googlesource.com/y',
          'revision': {
            'id': 'badcoffee',
            'url': 'https://y.googlesource.com/y/+/badcoffee',
            'timestamp': '1451692800000000',
            'committer_email': 'john@doe.com',
          },
        },
      ],
    })

  def test_get_config_all_partially_forbidden(self):
    self.mock(storage, 'get_config_sets_async', mock.Mock())
    storage.get_config_sets_async.return_value = future([
      storage.ConfigSet(
          id='services/x',
          location='https://x.googlesource.com/x',
          latest_revision='deadbeef',
      ),
      storage.ConfigSet(
          id='projects/y',
          location='https://y.googlesource.com/y',
          latest_revision='badcoffee',
      ),
    ])
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/x': True,
      'projects/y': False,
    }))

    resp = self.call_api('get_config_sets', {}).json_body

    self.assertEqual(resp, {
      'config_sets': [
        {
          'config_set': 'services/x',
          'location': 'https://x.googlesource.com/x',
          'revision': {
            'id': 'deadbeef',
          }
        },
      ],
    })

  ##############################################################################
  # get_config

  def test_get_config(self):
    self.mock_config()

    req = {
      'config_set': 'services/luci-config',
      'path': 'my.cfg',
      'revision': 'deadbeef',
    }
    resp = self.call_api('get_config', req).json_body

    self.assertEqual(resp, {
      'content': base64.b64encode('config text'),
      'content_hash': 'abc0123',
      'revision': 'deadbeef',
      'url': 'https://x.com/+/deadbeef',
    })
    storage.get_config_hashes_async.assert_called_once_with(
        {'services/luci-config': 'deadbeef'}, 'my.cfg')
    storage.get_configs_by_hashes_async.assert_called_once_with(['abc0123'])

  def test_get_config_hash_only(self):
    self.mock_config()

    req = {
      'config_set': 'services/luci-config',
      'hash_only': True,
      'path': 'my.cfg',
      'revision': 'deadbeef',
    }
    resp = self.call_api('get_config', req).json_body

    self.assertEqual(resp, {
      'content_hash': 'abc0123',
      'revision': 'deadbeef',
      'url': 'https://x.com/+/deadbeef',
    })
    self.assertFalse(storage.get_configs_by_hashes_async.called)

  def test_get_config_blob_not_found(self):
    self.mock_config(mock_content=False)
    req = {
      'config_set': 'services/luci-config',
      'path': 'my.cfg',
    }
    with self.call_should_fail(httplib.NOT_FOUND):
      self.call_api('get_config', req)

  def test_get_config_not_found(self):
    def get_config_hashes_async(revs, path):
      return future({cs: (None, None, None) for cs in revs})

    self.mock(storage, 'get_config_hashes_async', get_config_hashes_async)

    req = {
      'config_set': 'services/x',
      'path': 'a.cfg',
    }
    with self.call_should_fail(httplib.NOT_FOUND):
      self.call_api('get_config', req)

  def test_get_wrong_config_set(self):
    self.mock(acl, 'can_read_config_sets', mock.Mock(side_effect=ValueError))

    req = {
      'config_set': 'xxx',
      'path': 'my.cfg',
      'revision': 'deadbeef',
    }
    with self.call_should_fail(httplib.BAD_REQUEST):
      self.call_api('get_config', req)

  def test_get_config_without_permissions(self):
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/luci-config': False,
    }))
    self.mock(storage, 'get_config_hashes_async', mock.Mock())

    req = {
      'config_set': 'services/luci-config',
      'path': 'projects.cfg',
    }
    with self.call_should_fail(404):
      self.call_api('get_config', req)
    self.assertFalse(storage.get_config_hashes_async.called)

  ##############################################################################
  # get_config_by_hash

  def test_get_config_by_hash(self):
    self.mock(storage, 'get_configs_by_hashes_async', mock.Mock())
    storage.get_configs_by_hashes_async.return_value = future({
      'deadbeef': 'some content',
    })

    req = {'content_hash': 'deadbeef'}
    resp = self.call_api('get_config_by_hash', req).json_body

    self.assertEqual(resp, {
      'content': base64.b64encode('some content'),
    })

    storage.get_configs_by_hashes_async.return_value = future({
      'deadbeef': None,
    })
    with self.call_should_fail(httplib.NOT_FOUND):
      self.call_api('get_config_by_hash', req)

  ##############################################################################
  # get_projects

  def test_get_projects(self):
    projects.get_projects.return_value = [
      service_config_pb2.Project(id='chromium'),
      service_config_pb2.Project(id='v8'),
      service_config_pb2.Project(id='inconsistent'),
      service_config_pb2.Project(id='secret'),
    ]
    projects.get_metadata_async.return_value = future({
      'chromium': project_config_pb2.ProjectCfg(
          name='Chromium, the best browser', access='all'),
      'v8': project_config_pb2.ProjectCfg(access='all'),
      'inconsistent': project_config_pb2.ProjectCfg(access='all'),
      'secret': project_config_pb2.ProjectCfg(access='administrators'),
    })
    projects.get_repos_async.return_value = future({
      'chromium': (
          projects.RepositoryType.GITILES, 'https://chromium.example.com'),
      'v8': (projects.RepositoryType.GITILES, 'https://v8.example.com'),
      'inconsistent': (None, None),
      'secret': (projects.RepositoryType.GITILES, 'https://localhost/secret'),
    })

    resp = self.call_api('get_projects', {}).json_body

    self.assertEqual(resp, {
      'projects': [
        {
          'id': 'chromium',
          'name': 'Chromium, the best browser',
          'repo_type': 'GITILES',
          'repo_url': 'https://chromium.example.com',
        },
        {
          'id': 'v8',
          'repo_type': 'GITILES',
          'repo_url': 'https://v8.example.com',
        },
      ],
    })

  def mock_no_project_access(self):
    acl.has_projects_access.side_effect = (
        lambda pids: {pid: False for pid in pids})

  def test_get_projects_without_permissions(self):
    self.mock_no_project_access()
    self.call_api('get_projects', {})

  ##############################################################################
  # get_refs

  def test_get_refs(self):
    self.mock_refs()

    req = {'project_id': 'chromium'}
    resp = self.call_api('get_refs', req).json_body

    self.assertEqual(resp, {
      'refs': [
        {'name': 'refs/heads/master'},
        {'name': 'refs/heads/release42'},
      ],
    })

  def test_get_refs_without_permissions(self):
    self.mock_refs()
    self.mock_no_project_access()

    req = {'project_id': 'chromium'}
    with self.call_should_fail(httplib.NOT_FOUND):
      self.call_api('get_refs', req)
    self.assertFalse(projects.get_refs.called)

  def test_get_refs_of_non_existent_project(self):
    self.mock(projects, 'get_refs', mock.Mock())
    projects.get_refs.return_value = {'non-existent': None}
    req = {'project_id': 'non-existent'}
    with self.call_should_fail(httplib.NOT_FOUND):
      self.call_api('get_refs', req)

  ##############################################################################
  # get_project_configs

  def test_get_config_multi(self):
    projects.get_projects.return_value.extend([
      service_config_pb2.Project(id='inconsistent'),
      service_config_pb2.Project(id='secret'),
    ])
    projects.get_metadata_async.return_value.get_result().update({
      'inconsistent': project_config_pb2.ProjectCfg(access='all'),
      'secret': project_config_pb2.ProjectCfg(access='administrators'),
    })
    projects.get_repos_async.return_value.get_result().update({
      'inconsistent': (None, None),
      'secret': (projects.RepositoryType.GITILES, 'https://localhost/secret'),
    })

    self.mock(storage, 'get_latest_configs_async', mock.Mock())
    storage.get_latest_configs_async.return_value = future({
      'projects/chromium': (
          'deadbeef',
          'https://x.com/+/deadbeef',
          'abc0123',
          'config text'
      ),
      'projects/v8': (
          'beefdead',
          'https://x.com/+/beefdead',
          'ccc123',
          None  # no content
      ),
      'projects/secret': (
          'badcoffee',
          'https://x.com/+/badcoffee',
          'abcabc',
          'abcsdjksl'
      ),
    })

    req = {'path': 'cq.cfg'}
    resp = self.call_api('get_project_configs', req).json_body

    self.assertEqual(resp, {
      'configs': [{
        'config_set': 'projects/chromium',
        'revision': 'deadbeef',
        'content_hash': 'abc0123',
        'content': base64.b64encode('config text'),
        'url': 'https://x.com/+/deadbeef',
      }],
    })

  def test_get_config_multi_hashes_only(self):
    projects.get_projects.return_value.extend([
      service_config_pb2.Project(id='inconsistent'),
      service_config_pb2.Project(id='secret'),
    ])
    projects.get_metadata_async.return_value.get_result().update({
      'inconsistent': project_config_pb2.ProjectCfg(access='all'),
      'secret': project_config_pb2.ProjectCfg(access='administrators'),
    })
    projects.get_repos_async.return_value.get_result().update({
      'inconsistent': (None, None),
      'secret': (projects.RepositoryType.GITILES, 'https://localhost/secret'),
    })

    self.mock(storage, 'get_latest_configs_async', mock.Mock())
    storage.get_latest_configs_async.return_value = future({
      'projects/chromium': (
          'deadbeef',
          'https://x.com/+/deadbeef',
          'abc0123',
          None
      ),
      'projects/v8': (
          'beefdead',
          'https://x.com/+/beefdead',
          None,
          None  # no content hash
      ),
      'projects/secret': (
          'badcoffee',
          'https://x.com/+/badcoffee',
          'abcabc',
          None
      ),
    })

    req = {'path': 'cq.cfg', 'hashes_only': True}
    resp = self.call_api('get_project_configs', req).json_body

    self.assertEqual(resp, {
      'configs': [{
        'config_set': 'projects/chromium',
        'revision': 'deadbeef',
        'content_hash': 'abc0123',
        'url': 'https://x.com/+/deadbeef',
      }],
    })

  ##############################################################################
  # get_ref_configs

  def test_get_ref_configs_without_permission(self):
    self.mock_refs()
    self.mock_no_project_access()

    req = {'path': 'cq.cfg'}
    resp = self.call_api('get_ref_configs', req).json_body
    self.assertEqual(resp, {})

  ##############################################################################
  # reimport

  def test_reimport_without_permissions(self):
    req = {'config_set': 'services/x'}
    with self.call_should_fail(403):
      self.call_api('reimport', req)

  def test_reimport(self):
    self.mock(acl, 'is_admin', mock.Mock(return_value=True))
    self.mock(gitiles_import, 'import_config_set', mock.Mock())
    req = {'config_set': 'services/x'}
    self.call_api('reimport', req)
    gitiles_import.import_config_set.assert_called_once_with('services/x')

  def test_reimport_not_found(self):
    self.mock(acl, 'is_admin', mock.Mock(return_value=True))
    self.mock(gitiles_import, 'import_config_set', mock.Mock())
    gitiles_import.import_config_set.side_effect = gitiles_import.NotFoundError
    req = {'config_set': 'services/x'}
    with self.call_should_fail(404):
      self.call_api('reimport', req)

  def test_reimport_bad_request(self):
    self.mock(acl, 'is_admin', mock.Mock(return_value=True))
    self.mock(gitiles_import, 'import_config_set', mock.Mock())
    gitiles_import.import_config_set.side_effect = gitiles_import.Error
    req = {'config_set': 'services/x'}
    with self.call_should_fail(500):
      self.call_api('reimport', req)

  ##############################################################################
  # validate_config

  def test_validate_config(self):

    def validate_mock(_config_set, _path, _content, ctx=None):
      ctx.warning('potential problem')
      ctx.error('real problem')
      return future(ctx.result())

    self.mock(validation, 'validate_config_async', mock.Mock())
    validation.validate_config_async.side_effect = validate_mock

    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/luci-config': True,
    }))
    self.mock(acl, 'has_validation_access', mock.Mock(return_value=True))

    req = {
      'config_set': 'services/luci-config',
      'files': [{'path': 'myproj.cfg', 'content': 'mock_content'}]
    }
    resp = self.call_api('validate_config', req).json_body
    self.assertEqual(resp, {
      'messages': [
        {'path': 'myproj.cfg',
         'severity': 'WARNING',
         'text': 'myproj.cfg: potential problem'},
        {'path': 'myproj.cfg',
         'severity': 'ERROR',
         'text': 'myproj.cfg: real problem'},
      ]
    })

  def test_validate_config_no_files(self):
    req = {
      'config_set': 'services/luci-config',
      'files': []
    }
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/luci-config': True,
    }))
    self.mock(acl, 'has_validation_access', mock.Mock(return_value=True))
    with self.call_should_fail(400):
      self.call_api('validate_config', req)

  def test_validate_config_no_path(self):
    req = {
      'config_set': 'services/luci-config',
      'files': [{'path': '', 'content': 'mock_content'}]
    }
    self.mock(acl, 'can_read_config_sets', mock.Mock(return_value={
      'services/luci-config': True,
    }))
    self.mock(acl, 'has_validation_access', mock.Mock(return_value=True))
    with self.call_should_fail(400):
      self.call_api('validate_config', req)


if __name__ == '__main__':
  test_env.main()
