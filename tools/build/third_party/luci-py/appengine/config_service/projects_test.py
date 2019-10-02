#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from test_env import future
import test_env
test_env.setup_test_env()

from components.config.proto import service_config_pb2
from components.config.proto import project_config_pb2
from test_support import test_case
import mock

import projects
import storage


class ProjectsTestCase(test_case.TestCase):
  def setUp(self):
    super(ProjectsTestCase, self).setUp()
    self.mock(projects, '_filter_existing', lambda pids: pids)

  def mock_latest_config(self, config_set, contents):
    self.mock(storage, 'get_latest_configs_async', mock.Mock())
    storage.get_latest_configs_async.return_value = future({
      config_set: (
          'rev', 'file://config', storage.compute_hash(contents), contents),
    })

  def test_get_projects(self):
    self.mock_latest_config(storage.get_self_config_set(), '''
      projects {
        id: "chromium"
        config_location {
          storage_type: GITILES
          url: "http://localhost"
        }
      }
    ''')
    expected = service_config_pb2.ProjectsCfg(
        projects=[
          service_config_pb2.Project(
              id='chromium',
              config_location=service_config_pb2.ConfigSetLocation(
                storage_type=service_config_pb2.ConfigSetLocation.GITILES,
                url='http://localhost')
              ),
        ],
    )
    self.assertEqual(projects.get_projects(), expected.projects)

  def test_get_refs(self):
    self.mock_latest_config('projects/chromium', '''
      refs {
        name: "refs/heads/master"
      }
      refs {
        name: "refs/heads/release42"
        config_path: "other"
      }
    ''')
    expected = project_config_pb2.RefsCfg(
        refs=[
          project_config_pb2.RefsCfg.Ref(
              name='refs/heads/master'),
          project_config_pb2.RefsCfg.Ref(
              name='refs/heads/release42', config_path='other'),
        ],
    )
    self.assertEqual(
        projects.get_refs(['chromium']), {'chromium': expected.refs})

  def test_get_refs_of_non_existent_project(self):
    self.mock(projects, '_filter_existing', lambda pids: [])
    self.assertEqual(projects.get_refs(['chromium']), {'chromium': None})

  def test_repo_info(self):
    self.assertEqual(
        projects.get_repos_async(['x']).get_result(),
        {'x': (None, None)})
    projects.update_import_info(
        'x', projects.RepositoryType.GITILES, 'http://localhost/x')
    # Second time for coverage.
    projects.update_import_info(
        'x', projects.RepositoryType.GITILES, 'http://localhost/x')
    self.assertEqual(
        projects.get_repos_async(['x']).get_result(),
        {'x': (projects.RepositoryType.GITILES, 'http://localhost/x')})

    # Change it
    projects.update_import_info(
        'x', projects.RepositoryType.GITILES, 'http://localhost/y')


if __name__ == '__main__':
  test_env.main()
