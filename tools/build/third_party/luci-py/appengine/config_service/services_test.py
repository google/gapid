#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from test_env import future
import test_env
test_env.setup_test_env()

import mock
import logging

from components import net
from components.config.proto import service_config_pb2
from test_support import test_case

import common
import services
import storage


class ProjectsTestCase(test_case.TestCase):
  def setUp(self):
    super(ProjectsTestCase, self).setUp()
    self.mock(storage, 'get_self_config_async', mock.Mock())
    storage.get_self_config_async.return_value = future(
        service_config_pb2.ServicesCfg(
            services=[
              service_config_pb2.Service(
                  id='foo', metadata_url='https://foo.com/metadata'),
              service_config_pb2.Service(id='metadataless'),
            ]
        ))

  def mock_metadata_entity(self):
    dict = {
      'version': '1.0',
      'validation': {
        'url': 'https://a.com/validate',
        'patterns': [
          {'config_set': 'projects/foo', 'path': 'bar.cfg'},
          {'config_set': 'regex:services/.+', 'path': 'regex:.+'},
        ]
      }
    }
    mck_meta = (services._dict_to_dynamic_metadata(dict).SerializeToString())
    storage.ServiceDynamicMetadata(
        id='deadbeef',
        metadata=mck_meta,
    ).put()

  def test_dict_to_dynamic_metadata(self):
    with self.assertRaises(services.DynamicMetadataError):
      services._dict_to_dynamic_metadata([])

    self.assertEqual(
      services._dict_to_dynamic_metadata({
        'version': '1.0',
        'validation': {
          'url': 'https://a.com/validate',
          'patterns': [
            {'config_set': 'projects/foo', 'path': 'bar.cfg'},
            {'config_set': 'regex:services/.+', 'path': 'regex:.+'},
          ]
        }
      }),
      service_config_pb2.ServiceDynamicMetadata(
          validation=service_config_pb2.Validator(
              url='https://a.com/validate',
              patterns=[
                service_config_pb2.ConfigPattern(
                    config_set='projects/foo', path='bar.cfg'),
                service_config_pb2.ConfigPattern(
                    config_set='regex:services/.+', path='regex:.+'),
              ]
          )
      )
    )

  def test_get_metadata_async(self):
    self.mock(storage, 'get_self_config_async', mock.Mock())
    storage.get_self_config_async.return_value = future(
        service_config_pb2.ServicesCfg(
            services=[
              service_config_pb2.Service(
                  id='foo', metadata_url='https://foo.com/metadata')
            ]
        ))

    self.mock(net, 'json_request_async', mock.Mock())
    net.json_request_async.return_value = future({
        'version': '1.0',
        'validation': {
          'url': 'https://a.com/validate',
          'patterns': [
            {'config_set': 'projects/foo', 'path': 'bar.cfg'},
            {'config_set': 'regex:services/.+', 'path': 'regex:.+'},
          ]
        }
    })

    metadata = services.get_metadata_async('foo').get_result()
    self.assertEqual(
      metadata,
      service_config_pb2.ServiceDynamicMetadata(
          validation=service_config_pb2.Validator(
              url='https://a.com/validate',
              patterns=[
                service_config_pb2.ConfigPattern(
                    config_set='projects/foo', path='bar.cfg'),
                service_config_pb2.ConfigPattern(
                    config_set='regex:services/.+', path='regex:.+'),
              ]
          )
      )
    )

    net.json_request_async.assert_called_once_with(
        'https://foo.com/metadata', scopes=net.EMAIL_SCOPE)

    storage.get_self_config_async.assert_called_once_with(
        common.SERVICES_REGISTRY_FILENAME, service_config_pb2.ServicesCfg)

  def test_get_metadata_async_not_found(self):
    with self.assertRaises(services.ServiceNotFoundError):
      services.get_metadata_async('non-existent').get_result()

  def test_get_metadata_async_no_metadata(self):
    metadata = services.get_metadata_async('metadataless').get_result()
    self.assertIsNotNone(metadata)
    self.assertFalse(metadata.validation.patterns)

  def test_update_service_metadata_async_same(self):
    self.mock_metadata_entity()
    self.mock(net, 'json_request_async', mock.Mock())
    dict = {
      'version': '1.0',
      'validation': {
        'url': 'https://a.com/validate',
        'patterns': [
          {'config_set': 'projects/foo', 'path': 'bar.cfg'},
          {'config_set': 'regex:services/.+', 'path': 'regex:.+'},
        ]
      }
    }

    net.json_request_async.return_value = future(dict)
    self.mock(logging, 'info', mock.Mock())
    mock_service = mock.Mock()
    mock_service.id = 'deadbeef'
    mock_service.metadata_url = 'https://a.com/validate'
    services._update_service_metadata_async(mock_service).get_result()
    self.assertFalse(logging.info.called)

  def test_update_service_metadata_async_different(self):
    self.mock_metadata_entity()
    self.mock(net, 'json_request_async', mock.Mock())
    dict = {
      'version': '1.0',
      'validation': {
        'url': 'https://a.com/different_validate',
        'patterns': [
          {'config_set': 'projects/bar', 'path': 'foo.cfg'},
          {'config_set': 'regex:services/.+', 'path': 'regex:.+'},
        ]
      }
    }

    net.json_request_async.return_value = future(dict)
    self.mock(logging, 'info', mock.Mock())
    mock_service = mock.Mock()
    mock_service.id = 'deadbeef'
    mock_service.metadata_url = 'https://a.com/validate'
    services._update_service_metadata_async(mock_service).get_result()
    self.assertTrue(logging.info.called)

  def test_update_service_metadata_no_service_url(self):
    self.mock_metadata_entity()
    mock_service = mock.Mock()
    mock_service.id = 'deadbeef'
    mock_service.metadata_url = ''
    self.mock(logging, 'info', mock.Mock())
    services._update_service_metadata_async(mock_service).get_result()
    self.assertTrue(logging.info.called)


if __name__ == '__main__':
  test_env.main()
