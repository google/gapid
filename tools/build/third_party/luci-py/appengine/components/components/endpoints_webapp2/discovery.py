# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Discovery document generator for an Endpoints v1 over webapp2 service."""

import re

import endpoints

from protorpc import message_types
from protorpc import messages

from components import utils


def _normalize_name(n):
  """Splits words by non-alphanumeric characters and PascalCases the rest.

  Args:
    n: The string to normalize.

  Returns:
    A normalized version of the given string.
  """
  words = []
  for word in re.split(r'[^0-9a-zA-Z]', n):
    words.append('%s%s' % (word[:1].upper(), word[1:]))
  return ''.join(words)


def _normalize_whitespace(s):
  """Replaces consecutive whitespace characters with a single space.

  Args:
    s: The string to normalize, or None to return an empty string.

  Returns:
    A normalized version of the given string.
  """
  return ' '.join((s or '').split())


def _get_type_format(field):
  """Returns the schema type and format for the given message type.

  Args:
    field: The protorpc.messages.Field to get schema type and format for.

  Returns:
    (type, format) for use in the "schemas" section of a discovery document.
  """
  if isinstance(field, messages.BooleanField):
    return ('boolean', None)

  if isinstance(field, messages.BytesField):
    return ('string', 'byte')

  if isinstance(field, message_types.DateTimeField):
    return ('string', 'date-time')

  if isinstance(field, messages.EnumField):
    return ('string', None)

  if isinstance(field, messages.FloatField):
    if field.variant == messages.Variant.DOUBLE:
      return ('number', 'double')
    return ('number', 'float')

  if isinstance(field, messages.IntegerField):
    if field.variant in (messages.Variant.INT32, messages.Variant.SINT32):
      return ('integer', 'int32')

    if field.variant in (messages.Variant.INT64, messages.Variant.SINT64):
      # If the type requires int64 or uint64, specify string or JavaScript will
      # convert them to 32-bit.
      return ('string', 'int64')

    if field.variant == messages.Variant.UINT32:
      return ('integer', 'uint32')

    if field.variant == messages.Variant.UINT64:
      return ('string', 'uint64')

    # Despite the warning about JavaScript, Endpoints v2's discovery document
    # generator uses integer, int64 as the default here. Follow their choice.
    return ('integer', 'int64')

  if isinstance(field, messages.StringField):
    return ('string', None)

  return (None, None)


def _get_schemas(types):
  """Returns a schemas document for the given types.

  Args:
    types: The set of protorpc.messages.Messages subclasses to describe.

  Returns:
    A dict which can be written as JSON describing the types.
  """
  schemas = {}
  seen = set(types)
  types = list(types)
  # Messages may reference other messages whose schemas we need to add.
  # Keep a set of types we've already seen (but not necessarily processed) to
  # avoid repeatedly processing or queuing to process the same type.
  # Desired invariant: seen contains types which have ever been in types.
  # This invariant allows us to extend types mid-loop to add more types to
  # process without unnecessarily processing the same type twice. We achieve
  # this invariant by initializing seen to types and adding to seen every time
  # the loop adds to types.
  for message_type in types:
    name = _normalize_name(message_type.definition_name())

    schemas[name] = {
      'id': name,
      'type': 'object',
    }

    desc = _normalize_whitespace(message_type.__doc__)
    if desc:
      schemas[name]['description'] = desc

    properties = {}
    for field in message_type.all_fields():
      items = {}
      field_properties = {}
      schema_type = None

      # For non-message fields, add the field information to the schema
      # directly. For message fields, add a $ref to elsewhere in the schema
      # and ensure the type is queued to have its schema added. DateTimeField
      # is a message field but is treated as a non-message field.
      if (isinstance(field, messages.MessageField)
          and not isinstance(field, message_types.DateTimeField)):
        field_type = field.type().__class__
        desc = _normalize_whitespace(field_type.__doc__)
        if desc:
          field_properties['description'] = desc
        # Queue new types to have their schema added in a future iteration.
        if field_type not in seen:
          types.append(field_type)
          # Maintain loop invariant.
          seen.add(field_type)
        items['$ref'] = _normalize_name(field_type.definition_name())
      else:
        schema_type, schema_format = _get_type_format(field)
        items['type'] = schema_type
        if schema_format:
          items['format'] = schema_format

      if isinstance(field, messages.EnumField):
        # Endpoints v1 sorts these alphabetically while v2 does not.
        items['enum'] = sorted([enum.name for enum in field.type])
        # Endpoints v1 includes empty descriptions for each enum.
        items['enumDescriptions'] = ['' for _ in field.type]

      if field.default:
        field_properties['default'] = field.default
        # Defaults for types that aren't strings in the code but are strings
        # in the schema must be converted here. For example, EnumField
        # would otherwise have a default that isn't even valid JSON.
        if schema_type == 'string':
          field_properties['default'] = str(field.default)

      if field.required:
        field_properties['required'] = True

      # For repeated fields, most of the field information gets added to items.
      # For non-repeated fields, the field information is added directly to the
      # field properties. However, for parameters, even repeated fields have
      # their field information added directly to the field properties. See
      # _get_parameters below.
      if field.repeated:
        field_properties['items'] = items
        field_properties['type'] = 'array'
      else:
        field_properties.update(items)

      properties[field.name] = field_properties

    if properties:
      schemas[name]['properties'] = properties

  return schemas


def _get_parameters(message, path):
  """Returns a parameters document for the given parameters and path.

  Args:
    message: The protorpc.message.Message class describing the parameters.
    path: The path to the method.

  Returns:
    A dict which can be written as JSON describing the path parameters.
  """
  PARAMETER_REGEX = r'{([a-zA-Z_][a-zA-Z0-9_]*)}'
  # The order is the names of path parameters in the order in which they
  # appear in the path followed by the names of required query strings.
  order = re.findall(PARAMETER_REGEX, path)
  parameters = _get_schemas([message]).get(
      _normalize_name(message.definition_name()), {}).get('properties', {})
  for parameter, schema in parameters.iteritems():
    # As above, repeated fields for parameters do not have items.
    if schema['type'] == 'array':
      schema.update(schema.pop('items'))
      schema['repeated'] = True
    if parameter in order:
      schema['location'] = 'path'
    else:
      schema['location'] = 'query'
      if schema.get('required'):
        order.append(parameter)

  document = {}
  if order:
    document['parameterOrder'] = order
  if parameters:
    document['parameters'] = parameters
  return document


def _get_methods(service):
  """Returns methods, resources, and schemas documents for the given service.

  Args:
    service: The protorpc.remote.Service to describe.

  Returns:
    A tuple of three dicts which can be written as JSON describing the methods,
    resources, and types.
  """
  document = {
    'methods': {},
    'resources': {},
  }
  types = set()

  for _, method in service.all_remote_methods().iteritems():
    # Only describe methods decorated with @method.
    info = getattr(method, 'method_info', None)
    if info is None:
      continue
    # info.method_id returns <service name>.[<resource name>.]*<method name>.
    # There may be 0 or more resource names.
    method_id = info.method_id(service.api_info)
    parts = method_id.split('.')
    assert len(parts) > 1, method_id
    name = parts[-1]
    resource_parts = parts[1:-1]

    item = {
      'httpMethod': info.http_method,
      'id': method_id,
      'path': info.get_path(service.api_info),
      'scopes': [
        'https://www.googleapis.com/auth/userinfo.email',
      ],
    }

    desc = _normalize_whitespace(method.remote.method.__doc__)
    if desc:
      item['description'] = desc

    request = method.remote.request_type()
    rc = endpoints.ResourceContainer.get_request_message(method.remote)
    if not isinstance(rc, endpoints.ResourceContainer):
      if not isinstance(request, message_types.VoidMessage):
        if info.http_method not in ('GET', 'DELETE'):
          item['request'] = {
            # $refs refer to the "schemas" section of the discovery doc.
            '$ref': _normalize_name(request.__class__.definition_name()),
            'parameterName': 'resource',
          }
          types.add(request.__class__)
    else:
      # If the request type is a known ResourceContainer, create a schema
      # reference to the body if necessary. Path parameters are handled
      # differently.
      if rc.body_message_class != message_types.VoidMessage:
        if info.http_method not in ('GET', 'DELETE'):
          item['request'] = {
            '$ref': _normalize_name(rc.body_message_class.definition_name()),
            'parameterName': 'resource',
          }
          types.add(rc.body_message_class)
      item.update(_get_parameters(
          rc.parameters_message_class, info.get_path(service.api_info)))

    response = method.remote.response_type()
    if not isinstance(response, message_types.VoidMessage):
      item['response'] = {
        '$ref': _normalize_name(response.__class__.definition_name()),
      }
      types.add(response.__class__)

    pointer = document
    for part in resource_parts:
      pointer = pointer.setdefault('resources', {}).setdefault(part, {})
    pointer.setdefault('methods', {})[name] = item

  return document['methods'], document['resources'], _get_schemas(types)


def generate(classes, host, base_path):
  """Returns a discovery document for the given service.

  Args:
    classes: The non-empty list of protorpc.remote.Service classes to describe.
      All classes must be part of the same service.
    host: The host this request was received by.
    base_path: The base path under which all service paths exist.

  Returns:
    A dict which can be written as JSON describing the service.
  """
  assert classes, classes
  scheme = 'http:' if utils.is_local_dev_server() else 'https:'
  document = {
    'discoveryVersion': 'v1',
    'auth': {
      'oauth2': {
        'scopes': {s: {'description': s} for s in classes[0].api_info.scopes},
      },
    },
    'basePath': '%s/%s/%s' % (
        base_path, classes[0].api_info.name, classes[0].api_info.version),
    'baseUrl': '%s//%s%s/%s/%s' % (
        scheme, host, base_path,
        classes[0].api_info.name, classes[0].api_info.version),
    'batchPath': 'batch',
    'icons': {
      'x16': 'https://www.google.com/images/icons/product/search-16.gif',
      'x32': 'https://www.google.com/images/icons/product/search-32.gif',
    },
    'id': '%s:%s' % (classes[0].api_info.name, classes[0].api_info.version),
    'kind': 'discovery#restDescription',
    'name': classes[0].api_info.name,
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
            ' with API access, quota, and reports. Required unless you provide'
            ' an OAuth 2.0 token.'),
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
            'Available to use for quota purposes for server-side applications.'
            ' Can be any arbitrary string assigned to a user, but should not'
            ' exceed 40 characters. Overrides userIp if both are provided.'),
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
    'rootUrl': '%s//%s%s/' % (scheme, host, base_path),
    'servicePath': '%s/%s/' % (
        classes[0].api_info.name, classes[0].api_info.version),
    'version': classes[0].api_info.version,
  }
  if classes[0].api_info.title:
    document['title'] = classes[0].api_info.title
  desc = _normalize_whitespace(
      classes[0].api_info.description or classes[0].__doc__)
  if desc:
    document['description'] = desc
  if classes[0].api_info.documentation:
    document['documentationLink'] = classes[0].api_info.documentation
  methods = {}
  resources = {}
  schemas = {}
  for service in classes:
    m, r, s = _get_methods(service)
    methods.update(m)
    resources.update(r)
    schemas.update(s)
  if methods:
    document['methods'] = methods
  if resources:
    document['resources'] = resources
  if schemas:
    document['schemas'] = schemas
  return document


def directory(classes, host, base_path):
  """Returns a directory list for the given services.

  Args:
    classes: The list of protorpc.remote.Service classes to describe.
    host: The host this request was received by.
    base_path: The base path under which all service paths exist.

  Returns:
    A dict which can be written as JSON describing the services.
  """
  scheme = 'http:' if utils.is_local_dev_server() else 'https:'
  document = {
    'discoveryVersion': 'v1',
    'kind': 'discovery#directoryList',
  }

  items = {}
  for service in classes:
    item = {
      'discoveryLink': './apis/%s/%s/rest' % (
          service.api_info.name, service.api_info.version),
      'discoveryRestUrl': '%s//%s%s/discovery/v1/apis/%s/%s/rest' % (
          scheme, host, base_path,
          service.api_info.name, service.api_info.version),
      'id': '%s:%s' % (service.api_info.name, service.api_info.version),
      'icons': {
        'x16': 'https://www.google.com/images/icons/product/search-16.gif',
        'x32': 'https://www.google.com/images/icons/product/search-32.gif',
      },
      'kind': 'discovery#directoryItem',
      'name': service.api_info.name,
      'preferred': True,
      'version': service.api_info.version,
    }
    desc = _normalize_whitespace(
        service.api_info.description or service.__doc__)
    if desc:
      item['description'] = desc
    if service.api_info.documentation:
      item['documentationLink'] = service.api_info.documentation
    items[item['id']] = item

  if items:
    document['items'] = sorted(items.values(), key=lambda i: i['id'])
  return document
