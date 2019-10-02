#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import os

from test_env import future
import test_env
test_env.setup_test_env()

from test_support import test_case
import mock

from components.config.proto import service_config_pb2

import storage


class StorageTestCase(test_case.TestCase):

  def put_file(self, config_set, revision, path, content):
    revision_url = 'https://x.com/+/%s' % revision
    confg_set_key = storage.ConfigSet(
        id=config_set,
        location='https://x.com',

        latest_revision=revision,
        latest_revision_url=revision_url,
        latest_revision_time=datetime.datetime(2016, 1, 1),
        latest_revision_committer_email='john@example.com',
    ).put()
    rev_key = storage.Revision(id=revision, parent=confg_set_key).put()

    content_hash = storage.compute_hash(content)
    storage.File(
        id=path,
        parent=rev_key,
        content_hash=content_hash,
        url=os.path.join(revision_url, path),
    ).put()
    storage.Blob(id=content_hash, content=content).put()

  def test_get_config(self):
    self.put_file('foo', 'deadbeef', 'config.cfg', 'content')
    self.put_file('bar', 'badcoffee', 'config.cfg', 'content2')
    expected = {
      'foo': (
          'deadbeef',
          'https://x.com/+/deadbeef/config.cfg',
          'v1:6b584e8ece562ebffc15d38808cd6b98fc3d97ea',
      ),
      'bar': (
          'badcoffee',
          'https://x.com/+/badcoffee/config.cfg',
          'v1:db00fd65b218578127ea51f3dffac701f12f486a',
      ),
    }
    actual = storage.get_config_hashes_async(
      {'foo': 'deadbeef', 'bar': 'badcoffee'}, 'config.cfg').get_result()
    self.assertEqual(expected, actual)

  def test_get_non_existing_config(self):
    self.put_file('foo', 'deadbeef', 'config.cfg', 'content')
    expected = {
      'foo': (
          'deadbeef',
          'https://x.com/+/deadbeef/config.cfg',
          'v1:6b584e8ece562ebffc15d38808cd6b98fc3d97ea',
       ),
      'bar': (None, None, None),
    }
    actual = storage.get_config_hashes_async(
      {'foo': 'deadbeef', 'bar': 'badcoffee'}, 'config.cfg').get_result()
    self.assertEqual(expected, actual)

  def test_get_latest_config(self):
    self.put_file('foo', 'deadbeef', 'config.cfg', 'content')
    self.put_file('bar', 'badcoffee', 'config.cfg', 'content2')
    expected = {
      'foo': (
          'deadbeef',
          'https://x.com/+/deadbeef/config.cfg',
          'v1:6b584e8ece562ebffc15d38808cd6b98fc3d97ea'
      ),
      'bar': (
          'badcoffee',
          'https://x.com/+/badcoffee/config.cfg',
          'v1:db00fd65b218578127ea51f3dffac701f12f486a'
      ),
    }
    actual = storage.get_config_hashes_async(
        {'foo': None, 'bar': None}, 'config.cfg').get_result()
    self.assertEqual(expected, actual)

  def test_get_latest_non_existing_config_set(self):
    self.put_file('foo', 'deadbeef', 'config.cfg', 'content')
    expected = {
      'foo': (
          'deadbeef',
          'https://x.com/+/deadbeef/config.cfg',
          'v1:6b584e8ece562ebffc15d38808cd6b98fc3d97ea'
      ),
      'bar': (None, None, None),
    }
    actual = storage.get_config_hashes_async(
      {'foo': None, 'bar': None}, 'config.cfg').get_result()
    self.assertEqual(expected, actual)

  def test_get_config_by_hash(self):
    expected = {'deadbeef': None}
    actual = storage.get_configs_by_hashes_async(['deadbeef']).get_result()
    self.assertEqual(expected, actual)

    storage.Blob(id='deadbeef', content='content').put()
    expected = {'deadbeef': 'content'}
    actual = storage.get_configs_by_hashes_async(['deadbeef']).get_result()
    self.assertEqual(expected, actual)

  def test_compute_hash(self):
    content = 'some content\n'
    # echo some content | git hash-object --stdin
    expected = 'v1:2ef267e25bd6c6a300bb473e604b092b6a48523b'
    self.assertEqual(expected, storage.compute_hash(content))

  def test_import_blob(self):
    content = 'some content'
    storage.import_blob(content)
    storage.import_blob(content)  # Coverage.
    blob = storage.Blob.get_by_id_async(
        storage.compute_hash(content)).get_result()
    self.assertIsNotNone(blob)
    self.assertEqual(blob.content, content)

  def test_message_field_merge(self):
    default_msg = service_config_pb2.ImportCfg(
        gitiles=service_config_pb2.ImportCfg.Gitiles(fetch_log_deadline=42))
    self.mock(storage, 'get_latest_configs_async', mock.Mock())
    storage.get_latest_configs_async.return_value = future({
      storage.get_self_config_set(): (
          'rev', 'file://config', 'hash',
          'gitiles { fetch_archive_deadline: 10 }'),
    })
    msg = storage.get_self_config_async(
        'import.cfg', lambda: default_msg).get_result()
    self.assertEqual(msg.gitiles.fetch_log_deadline, 42)

  def test_get_self_config(self):
    expected = service_config_pb2.AclCfg(project_access_group='group')

    self.mock(storage, 'get_config_hashes_async', mock.Mock())
    self.mock(storage, 'get_configs_by_hashes_async', mock.Mock())
    storage.get_config_hashes_async.return_value = future({
        storage.get_self_config_set(): (
            'deadbeef', 'file://config', 'beefdead'),
    })
    storage.get_configs_by_hashes_async.return_value = future({
      'beefdead': 'project_access_group: "group"',
    })

    actual = storage.get_self_config_async(
        'acl.cfg', service_config_pb2.AclCfg).get_result()
    self.assertEqual(expected, actual)

    storage.get_config_hashes_async.assert_called_once_with(
        {storage.get_self_config_set(): None}, 'acl.cfg')
    storage.get_configs_by_hashes_async.assert_called_once_with(['beefdead'])

    # memcached:
    storage.get_config_hashes_async.reset_mock()
    storage.get_configs_by_hashes_async.reset_mock()
    actual = storage.get_self_config_async(
        'acl.cfg', service_config_pb2.AclCfg).get_result()
    self.assertEqual(expected, actual)
    self.assertFalse(storage.get_config_hashes_async.called)
    self.assertFalse(storage.get_configs_by_hashes_async.called)


if __name__ == '__main__':
  test_env.main()
