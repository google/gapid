#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from protorpc import message_types
from protorpc import messages
from protorpc import remote
import endpoints

from test_support import test_case
import discovery


class Enum(messages.Enum):
  """An enum to test with."""
  UNKNOWN = 0
  KNOWN = 1


class Message(messages.Message):
  """A message to test with."""
  boolean = messages.BooleanField(1)
  integer = messages.IntegerField(2, default=2)
  string = messages.StringField(3)
  repeated = messages.BooleanField(4, repeated=True)
  required = messages.BooleanField(5, required=True)


class Child(messages.Message):
  """A child message to test recursion with."""
  enum = messages.EnumField(Enum, 1, default=Enum.UNKNOWN)


class Parent(messages.Message):
  """A parent message to test recursion with."""
  child = messages.MessageField(Child, 1)
  datetime = message_types.DateTimeField(2)


@endpoints.api('service', 'v1', title='Titled Service', documentation='link')
class Service(remote.Service):
  """A service to test with."""

  @endpoints.method(message_types.VoidMessage, Message, http_method='GET')
  def get_method(self, _):
    """An HTTP GET method."""
    return Message()

  @endpoints.method(Message, Message)
  def post_method(self, _):
    """An HTTP POST method.

    Has a multi-line description.
    """
    return Message()

  @endpoints.method(
      endpoints.ResourceContainer(
          message_types.VoidMessage,
          string=messages.StringField(1, repeated=True)),
      Message, http_method='GET')
  def query_string_method(self, _):
    """An HTTP GET method supporting query strings."""
    return Message()

  @endpoints.method(
      endpoints.ResourceContainer(Message, path=messages.StringField(1)),
      Message, path='{path}/method')
  def path_parameter_method(self, _):
    """An HTTP POST method supporting path parameters."""
    return Message()

  @endpoints.method(Message, Message, name='resource.a')
  def resource_method_a(self, _):
    """A method belonging to a resource."""
    return Message()

  @endpoints.method(Message, Message, name='resource.subresource.b')
  def resource_method_b(self, _):
    """A method belonging to a sub-resource."""
    return Message()

  @endpoints.method(Message, Message, name='resource.subresource.c')
  def resource_method_c(self, _):
    """A method belonging to a sub-resource."""
    return Message()


SplitService = endpoints.api(
    'split', 'v1', description='A split service to test with.')


@SplitService.api_class(resource_name='sa', path='a')
class ServiceA(remote.Service):
  """Part A of a split service to test with."""

  @endpoints.method(message_types.VoidMessage, Message)
  def post_method(self, _):
    """An HTTP POST method."""
    return Message()


@SplitService.api_class(resource_name='sb', path='b')
class ServiceB(remote.Service):
  """Part B of a split service to test with."""

  @endpoints.method(endpoints.ResourceContainer(), Message)
  def post_method(self, _):
    """An HTTP POST method."""
    return Message()


class DiscoveryWebapp2TestCase(test_case.TestCase):
  """Tests for discovery.py"""

  def test_normalize_name(self):
    """Tests discovery._normalize_name."""
    self.assertEqual(
        discovery._normalize_name('some_package.subpackage.a__method'),
        'SomePackageSubpackageAMethod')

  def test_get_methods(self):
    """Tests discovery._get_methods."""
    expected = (
      # methods
      {
        'get_method': {
          'description': 'An HTTP GET method.',
          'httpMethod': 'GET',
          'id': 'service.get_method',
          'path': 'get_method',
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
        'path_parameter_method': {
          'description': 'An HTTP POST method supporting path parameters.',
          'httpMethod': 'POST',
          'id': 'service.path_parameter_method',
          'parameterOrder': [
            'path',
          ],
          'parameters': {
            'path': {
              'location': 'path',
              'type': 'string',
            },
          },
          'path': '{path}/method',
          'request': {
            '$ref': 'DiscoveryTestMessage',
            'parameterName': 'resource',
          },
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
        'post_method': {
          'description': 'An HTTP POST method. Has a multi-line description.',
          'httpMethod': 'POST',
          'id': 'service.post_method',
          'path': 'post_method',
          'request': {
            '$ref': 'DiscoveryTestMessage',
            'parameterName': 'resource',
          },
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
        'query_string_method': {
          'description': 'An HTTP GET method supporting query strings.',
          'httpMethod': 'GET',
          'id': 'service.query_string_method',
          'parameters': {
            'string': {
              'location': 'query',
              'repeated': True,
              'type': 'string',
            },
          },
          'path': 'query_string_method',
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
      },
      # resources
      {
        'resource': {
          'methods': {
            'a': {
              'description': 'A method belonging to a resource.',
              'httpMethod': 'POST',
              'id': 'service.resource.a',
              'path': 'resource_method_a',
              'request': {
                '$ref': 'DiscoveryTestMessage',
                'parameterName': 'resource',
              },
              'response': {
                '$ref': 'DiscoveryTestMessage',
              },
              'scopes': [
                'https://www.googleapis.com/auth/userinfo.email',
              ],
            },
          },
          'resources': {
            'subresource': {
              'methods': {
                'b': {
                  'description': 'A method belonging to a sub-resource.',
                  'httpMethod': 'POST',
                  'id': 'service.resource.subresource.b',
                  'path': 'resource_method_b',
                  'request': {
                    '$ref': 'DiscoveryTestMessage',
                    'parameterName': 'resource',
                  },
                  'response': {
                    '$ref': 'DiscoveryTestMessage',
                  },
                  'scopes': [
                    'https://www.googleapis.com/auth/userinfo.email',
                  ],
                    },
                'c': {
                  'description': 'A method belonging to a sub-resource.',
                  'httpMethod': 'POST',
                  'id': 'service.resource.subresource.c',
                  'path': 'resource_method_c',
                  'request': {
                    '$ref': 'DiscoveryTestMessage',
                    'parameterName': 'resource',
                  },
                  'response': {
                    '$ref': 'DiscoveryTestMessage',
                  },
                  'scopes': [
                    'https://www.googleapis.com/auth/userinfo.email',
                  ],
                },
              },
            },
          },
        },
      },
      # schemas
      {
        'DiscoveryTestMessage': {
          'description': 'A message to test with.',
          'id': 'DiscoveryTestMessage',
          'properties': {
            'boolean': {
              'type': 'boolean',
            },
            'integer': {
              'default': '2',
              'format': 'int64',
              'type': 'string',
            },
            'repeated': {
              'items': {
                'type': 'boolean',
              },
              'type': 'array',
            },
            'required': {
              'required': True,
              'type': 'boolean',
            },
            'string': {
              'type': 'string',
            },
          },
          'type': 'object',
        },
      },
    )
    self.assertEqual(discovery._get_methods(Service), expected)

  def test_get_parameters(self):
    """Tests for discovery._get_parameters."""
    expected = {
      'parameterOrder': [
        'boolean',
        'string',
        'required',
      ],
      'parameters': {
        'boolean': {
          'location': 'path',
          'type': 'boolean',
        },
        'integer': {
          'default': '2',
          'format': 'int64',
          'location': 'query',
          'type': 'string',
        },
        'repeated': {
          'location': 'query',
          'repeated': True,
          'type': 'boolean',
        },
        'required': {
          'location': 'query',
          'required': True,
          'type': 'boolean',
        },
        'string': {
          'location': 'path',
          'type': 'string',
        },
      },
    }
    self.assertEqual(discovery._get_parameters(
        Message, 'path/{boolean}/with/{string}/parameters'), expected)

  def test_get_schemas(self):
    """Tests for discovery._get_schemas."""
    expected = {
      'DiscoveryTestChild': {
        'description': 'A child message to test recursion with.',
        'id': 'DiscoveryTestChild',
        'properties': {
          'enum': {
            'default': 'UNKNOWN',
            'enum': [
              'KNOWN',
              'UNKNOWN',
            ],
            'enumDescriptions': [
              '',
              '',
            ],
            'type': 'string',
          },
        },
        'type': 'object',
      },
      'DiscoveryTestParent': {
        'description': 'A parent message to test recursion with.',
        'id': 'DiscoveryTestParent',
        'properties': {
          'child': {
            '$ref': 'DiscoveryTestChild',
            'description': 'A child message to test recursion with.',
          },
          'datetime': {
            'format': 'date-time',
            'type': 'string',
          },
        },
        'type': 'object',
      },
    }
    self.assertEqual(discovery._get_schemas([Parent]), expected)

  def test_generate(self):
    """Tests for discovery.generate."""
    with self.assertRaises(AssertionError):
      discovery.generate([], 'localhost:8080', '/api')
    expected = {
      'auth': {
        'oauth2': {
          'scopes': {
            'https://www.googleapis.com/auth/userinfo.email': {
              'description': 'https://www.googleapis.com/auth/userinfo.email',
            },
          },
        },
      },
      'basePath': '/api/service/v1',
      'baseUrl': 'http://localhost:8080/api/service/v1',
      'batchPath': 'batch',
      'description': 'A service to test with.',
      'discoveryVersion': 'v1',
      'documentationLink': 'link',
      'icons': {
        'x16': 'https://www.google.com/images/icons/product/search-16.gif',
        'x32': 'https://www.google.com/images/icons/product/search-32.gif',
      },
      'id': 'service:v1',
      'kind': 'discovery#restDescription',
      'methods': {
        'get_method': {
          'description': 'An HTTP GET method.',
          'httpMethod': 'GET',
          'id': 'service.get_method',
          'path': 'get_method',
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
        'path_parameter_method': {
          'description': 'An HTTP POST method supporting path parameters.',
          'httpMethod': 'POST',
          'id': 'service.path_parameter_method',
          'parameterOrder': [
            'path',
          ],
          'parameters': {
            'path': {
              'location': 'path',
              'type': 'string',
            },
          },
          'path': '{path}/method',
          'request': {
            '$ref': 'DiscoveryTestMessage',
            'parameterName': 'resource',
          },
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
        'post_method': {
          'description': 'An HTTP POST method. Has a multi-line description.',
          'httpMethod': 'POST',
          'id': 'service.post_method',
          'path': 'post_method',
          'request': {
            '$ref': 'DiscoveryTestMessage',
            'parameterName': 'resource',
          },
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
        'query_string_method': {
          'description': 'An HTTP GET method supporting query strings.',
          'httpMethod': 'GET',
          'id': 'service.query_string_method',
          'parameters': {
            'string': {
              'location': 'query',
              'repeated': True,
              'type': 'string',
            },
          },
          'path': 'query_string_method',
          'response': {
            '$ref': 'DiscoveryTestMessage',
          },
          'scopes': [
            'https://www.googleapis.com/auth/userinfo.email',
          ],
        },
      },
      'name': 'service',
      'parameters': {
        'alt': {
          'default': 'json',
          'description': 'Data format for the response.',
          'enum': ['json'],
          'enumDescriptions': [
            'Responses with Content-Type of application/json',
          ],
          'location': 'query',
          'type': 'string',
        },
        'fields': {
          'description': (
              'Selector specifying which fields to include in a partial'
              ' response.'),
          'location': 'query',
          'type': 'string',
        },
        'key': {
          'description': (
              'API key. Your API key identifies your project and provides you'
              ' with API access, quota, and reports. Required unless you'
              ' provide an OAuth 2.0 token.'),
          'location': 'query',
          'type': 'string',
        },
        'oauth_token': {
          'description': 'OAuth 2.0 token for the current user.',
          'location': 'query',
          'type': 'string',
        },
        'prettyPrint': {
          'default': 'true',
          'description': 'Returns response with indentations and line breaks.',
          'location': 'query',
          'type': 'boolean',
        },
        'quotaUser': {
          'description': (
              'Available to use for quota purposes for server-side'
              ' applications. Can be any arbitrary string assigned to a user,'
              ' but should not exceed 40 characters. Overrides userIp if both'
              ' are provided.'),
          'location': 'query',
          'type': 'string',
        },
        'userIp': {
          'description': (
              'IP address of the site where the request originates. Use this if'
              ' you want to enforce per-user limits.'),
          'location': 'query',
          'type': 'string',
        },
      },
      'protocol': 'rest',
      'resources': {
        'resource': {
          'methods': {
            'a': {
              'description': 'A method belonging to a resource.',
              'httpMethod': 'POST',
              'id': 'service.resource.a',
              'path': 'resource_method_a',
              'request': {
                '$ref': 'DiscoveryTestMessage',
                'parameterName': 'resource',
              },
              'response': {
                '$ref': 'DiscoveryTestMessage',
              },
              'scopes': [
                'https://www.googleapis.com/auth/userinfo.email',
              ],
            },
          },
          'resources': {
            'subresource': {
              'methods': {
                'b': {
                  'description': 'A method belonging to a sub-resource.',
                  'httpMethod': 'POST',
                  'id': 'service.resource.subresource.b',
                  'path': 'resource_method_b',
                  'request': {
                    '$ref': 'DiscoveryTestMessage',
                    'parameterName': 'resource',
                  },
                  'response': {
                    '$ref': 'DiscoveryTestMessage',
                  },
                  'scopes': [
                    'https://www.googleapis.com/auth/userinfo.email',
                  ],
                },
                'c': {
                  'description': 'A method belonging to a sub-resource.',
                  'httpMethod': 'POST',
                  'id': 'service.resource.subresource.c',
                  'path': 'resource_method_c',
                  'request': {
                    '$ref': 'DiscoveryTestMessage',
                    'parameterName': 'resource',
                  },
                  'response': {
                    '$ref': 'DiscoveryTestMessage',
                  },
                  'scopes': [
                    'https://www.googleapis.com/auth/userinfo.email',
                  ],
                },
              },
            },
          },
        },
      },
      'rootUrl': 'http://localhost:8080/api/',
      'schemas': {
        'DiscoveryTestMessage': {
          'description': 'A message to test with.',
          'id': 'DiscoveryTestMessage',
          'properties': {
            'boolean': {
              'type': 'boolean',
            },
            'integer': {
              'default': '2',
              'format': 'int64',
              'type': 'string',
            },
            'repeated': {
              'items': {
                'type': 'boolean',
              },
              'type': 'array',
            },
            'required': {
              'required': True,
              'type': 'boolean',
            },
            'string': {
              'type': 'string',
            },
          },
          'type': 'object',
        },
      },
      'servicePath': 'service/v1/',
      'title': 'Titled Service',
      'version': 'v1',
    }
    self.assertEqual(
        discovery.generate([Service], 'localhost:8080', '/api'), expected)
    expected = {
      'auth': {
        'oauth2': {
          'scopes': {
            'https://www.googleapis.com/auth/userinfo.email': {
              'description': 'https://www.googleapis.com/auth/userinfo.email',
            },
          },
        },
      },
      'basePath': '/api/split/v1',
      'baseUrl': 'http://localhost:8080/api/split/v1',
      'batchPath': 'batch',
      'description': 'A split service to test with.',
      'discoveryVersion': 'v1',
      'icons': {
        'x16': 'https://www.google.com/images/icons/product/search-16.gif',
        'x32': 'https://www.google.com/images/icons/product/search-32.gif',
      },
      'id': 'split:v1',
      'kind': 'discovery#restDescription',
      'name': 'split',
      'parameters': {
        'alt': {
          'default': 'json',
          'description': 'Data format for the response.',
          'enum': ['json'],
          'enumDescriptions': [
            'Responses with Content-Type of application/json',
          ],
          'location': 'query',
          'type': 'string',
        },
        'fields': {
          'description': (
              'Selector specifying which fields to include in a partial'
              ' response.'),
          'location': 'query',
          'type': 'string',
        },
        'key': {
          'description': (
              'API key. Your API key identifies your project and provides you'
              ' with API access, quota, and reports. Required unless you'
              ' provide an OAuth 2.0 token.'),
          'location': 'query',
          'type': 'string',
        },
        'oauth_token': {
          'description': 'OAuth 2.0 token for the current user.',
          'location': 'query',
          'type': 'string',
        },
        'prettyPrint': {
          'default': 'true',
          'description': 'Returns response with indentations and line breaks.',
          'location': 'query',
          'type': 'boolean',
        },
        'quotaUser': {
          'description': (
              'Available to use for quota purposes for server-side'
              ' applications. Can be any arbitrary string assigned to a user,'
              ' but should not exceed 40 characters. Overrides userIp if both'
              ' are provided.'),
          'location': 'query',
          'type': 'string',
        },
        'userIp': {
          'description': (
              'IP address of the site where the request originates. Use this if'
              ' you want to enforce per-user limits.'),
          'location': 'query',
          'type': 'string',
        },
      },
      'protocol': 'rest',
      'resources': {
        'sa': {
          'methods': {
            'post_method': {
              'description': 'An HTTP POST method.',
              'httpMethod': 'POST',
              'id': 'split.sa.post_method',
              'path': 'a/post_method',
              'response': {
                '$ref': 'DiscoveryTestMessage',
              },
              'scopes': [
                'https://www.googleapis.com/auth/userinfo.email',
              ],
            },
          },
        },
        'sb': {
          'methods': {
            'post_method': {
              'description':
                  'An HTTP POST method.',
              'httpMethod': 'POST',
              'id': 'split.sb.post_method',
              'path': 'b/post_method',
              'response': {
                '$ref': 'DiscoveryTestMessage',
              },
              'scopes': [
                'https://www.googleapis.com/auth/userinfo.email',
              ],
            },
          },
        },
      },
      'rootUrl': 'http://localhost:8080/api/',
      'schemas': {
        'DiscoveryTestMessage': {
          'description': 'A message to test with.',
          'id': 'DiscoveryTestMessage',
          'properties': {
            'boolean': {
              'type': 'boolean',
            },
            'integer': {
              'default': '2',
              'format': 'int64',
              'type': 'string',
            },
            'repeated': {
              'items': {
                'type': 'boolean',
              },
              'type': 'array',
            },
            'required': {
              'required': True,
              'type': 'boolean',
            },
            'string': {
              'type': 'string',
            },
          },
          'type': 'object',
        },
      },
      'servicePath': 'split/v1/',
      'version': 'v1',
    }
    self.assertEqual(
        discovery.generate([ServiceA, ServiceB], 'localhost:8080', '/api'),
        expected)

  def test_directory(self):
    """Tests for discovery.directory."""
    expected = {
      'discoveryVersion': 'v1',
      'items': [
        {
          'description': 'A service to test with.',
          'discoveryLink': './apis/service/v1/rest',
          'discoveryRestUrl':
              'http://localhost:8080/api/discovery/v1/apis/service/v1/rest',
          'documentationLink': 'link',
          'icons': {
            'x16': 'https://www.google.com/images/icons/product/search-16.gif',
            'x32': 'https://www.google.com/images/icons/product/search-32.gif',
          },
          'id': 'service:v1',
          'kind': 'discovery#directoryItem',
          'name': 'service',
          'preferred': True,
          'version': 'v1',
        },
        {
          'description': 'A split service to test with.',
          'discoveryLink': './apis/split/v1/rest',
          'discoveryRestUrl':
              'http://localhost:8080/api/discovery/v1/apis/split/v1/rest',
          'icons': {
            'x16': 'https://www.google.com/images/icons/product/search-16.gif',
            'x32': 'https://www.google.com/images/icons/product/search-32.gif',
          },
          'id': 'split:v1',
          'kind': 'discovery#directoryItem',
          'name': 'split',
          'preferred': True,
          'version': 'v1',
        },
      ],
      'kind': 'discovery#directoryList',
    }
    self.assertEqual(
        discovery.directory(
            [Service, ServiceA, ServiceB], 'localhost:8080', '/api'), expected)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
