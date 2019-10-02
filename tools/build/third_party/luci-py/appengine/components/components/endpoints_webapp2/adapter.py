# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import collections
import endpoints
import httplib
import json
import logging
import os
import posixpath
import urlparse

from endpoints import protojson
from google.appengine.api import memcache
from google.appengine.api import modules
from protorpc import message_types
from protorpc import messages
from protorpc import remote
import webapp2

from components import net
from components import template
from components import utils

import discovery
import partial


PROTOCOL = protojson.EndpointsProtoJson()


THIS_DIR = os.path.dirname(os.path.abspath(__file__))


def decode_field(field, value):
  """Like PROTOCOL.decode_field, but also supports booleans."""
  if isinstance(field, messages.BooleanField):
    value = value.lower()
    if value == 'true':
      return True
    elif value == 'false':
      return False
    else:
      raise ValueError('boolean field must be either "true" or "false"')
  return PROTOCOL.decode_field(field, value)


def decode_message(remote_method_info, request):
  """Decodes a protorpc message from an webapp2 request.

  If method accepts a resource container, parses field values from URL too.
  """
  req_msg = endpoints.ResourceContainer.get_request_message(remote_method_info)
  if isinstance(req_msg, endpoints.ResourceContainer):
    res_container = req_msg
    body_type = req_msg.body_message_class
  else:
    res_container = None
    body_type = remote_method_info.request_type

  body = PROTOCOL.decode_message(body_type, request.body)
  if res_container:
    result = res_container.combined_message_class()
    for f in body.all_fields():
      setattr(result, f.name, getattr(body, f.name))
  else:
    result = body

  # Read field values from query string parameters or URL path.
  if res_container or request.method == 'GET':
    if request.method == 'GET':
      # In addition to standard ResourceContainer request type, we also support
      # GET request handlers that use Message instead of ResourceContainer,
      # because it is non-ambiguous (because GET requests cannot have body).
      param_fields = result.all_fields()
    else:
      param_fields = res_container.parameters_message_class.all_fields()
    for f in param_fields:
      if f.name in request.route_kwargs:
        values = [request.route_kwargs[f.name]]
      else:
        values = request.get_all(f.name)
      if values:
        values = [decode_field(f, v) for v in values]
        if f.repeated:
          getattr(result, f.name).extend(values)
        else:
          setattr(result, f.name, values[0])
  return result


def add_cors_headers(headers):
  headers['Access-Control-Allow-Origin'] = '*'
  headers['Access-Control-Allow-Headers'] = (
    'Origin, Authorization, Content-Type, Accept, User-Agent')
  headers['Access-Control-Allow-Methods'] = (
    'DELETE, GET, OPTIONS, POST, PUT')


class CorsHandler(webapp2.RequestHandler):
  def options(self, *_args, **_kwargs):
    add_cors_headers(self.response.headers)


def path_handler(api_class, api_method, service_path):
  """Returns a webapp2.RequestHandler subclass for the API methods."""
  # Why return a class? Because webapp2 explicitly checks if handler that we
  # passed to Route is a class.

  class Handler(webapp2.RequestHandler):
    def dispatch(self):
      add_cors_headers(self.response.headers)

      api = api_class()
      api.initialize_request_state(remote.HttpRequestState(
          remote_host=None,
          remote_address=self.request.remote_addr,
          server_host=self.request.host,
          server_port=self.request.server_port,
          http_method=self.request.method,
          service_path=service_path,
          headers=self.request.headers.items()))

      try:
        req = decode_message(api_method.remote, self.request)
      except (messages.DecodeError, messages.ValidationError, ValueError) as ex:
        response_body = json.dumps({'error': {'message': ex.message}})
        self.response.set_status(httplib.BAD_REQUEST)
      else:
        try:
          res = api_method(api, req)
        except endpoints.ServiceException as ex:
          response_body = json.dumps({'error': {'message': ex.message}})
          self.response.set_status(ex.http_status)
        else:
          if isinstance(res, message_types.VoidMessage):
            self.response.set_status(204)
            response_body = None
          else:
            response_body = PROTOCOL.encode_message(res)
            if self.request.get('fields'):
              try:
                # PROTOCOL.encode_message checks that the message is initialized
                # before dumping it directly to JSON string. Therefore we can't
                # mask the protocol buffer (if masking removes a required field
                # then encode_message will fail). Instead, call encode_message
                # first, then load the JSON string into a dict, mask the dict,
                # and dump it back to JSON.
                response_body = json.dumps(partial.mask(
                    json.loads(response_body), self.request.get('fields')))
              except (partial.ParsingError, ValueError) as e:
                # Log the error but return the full response.
                logging.warning('Ignoring erroneous field mask %r: %s',
                                self.request.get('fields'), e)

      if self.response.status_int != 204:
        self.response.content_type = 'application/json; charset=utf-8'
        self.response.out.write(response_body)
      else:
        # webob sets content_type to text/html by default.
        self.response.content_type = ''

  return Handler


def api_routes(api_classes, base_path='/_ah/api', regex='[^/]+'):
  """Creates webapp2 routes for the given Endpoints v1 services.

  Args:
    api_classes: A list of protorpc.remote.Service classes to create routes for.
    base_path: The base path under which all service paths should exist. If
      unspecified, defaults to /_ah/api.
    regex: Regular expression to allow in path parameters.

  Returns:
    A list of webapp2.Routes.
  """
  routes = []

  # Add routes for each class.
  for api_class in api_classes:
    api_base_path = '%s/%s/%s' % (
        base_path, api_class.api_info.name, api_class.api_info.version)
    templates = set()

    # Add routes for each method of each class.
    for _, method in sorted(api_class.all_remote_methods().iteritems()):
      info = method.method_info
      method_path = info.get_path(api_class.api_info)
      method_path = method_path.replace('{', '<').replace('}', ':%s>' % regex)
      t = posixpath.join(api_base_path, method_path)
      http_method = info.http_method.upper() or 'POST'
      handler = path_handler(api_class, method, api_base_path)
      routes.append(webapp2.Route(t, handler, methods=[http_method]))
      templates.add(t)

    # Add routes for HTTP OPTIONS (to add CORS headers) for each method.
    for t in sorted(templates):
      routes.append(webapp2.Route(t, CorsHandler, methods=['OPTIONS']))

  # Add generic routes.
  routes.extend([
      directory_service_route(api_classes, base_path),
      discovery_service_route(api_classes, base_path),
      explorer_proxy_route(base_path),
      explorer_redirect_route(base_path),
  ])
  return routes


def api_server(api_classes, base_path='/_ah/api', regex='[^/]+'):
  """Creates a webapp2 application for the given Endpoints v1 services.

  A shortcut for webapp2.WSGIApplication(api_routes(...)), which exists to
  match endpoints.api_server's behavior of returning a webapp2.WSGIApplication.

  Args:
    api_classes: A list of protorpc.remote.Service classes to create routes for.
    base_path: The base path under which all service paths should exist. If
      unspecified, defaults to /_ah/api.
    regex: Regular expression to allow in path parameters.

  Returns:
    A webapp2.WSGIApplication.
  """
  return webapp2.WSGIApplication(api_routes(api_classes, base_path, regex))


def discovery_handler_factory(api_classes, base_path):
  """Returns a discovery request handler which knows about the given services.

  Args:
    api_classes: A list of protorpc.remote.Service classes the handler should
      know about.
    base_path: The base path under which all service paths exist.

  Returns:
    A webapp2.RequestHandler.
  """
  # Create a map of (name, version) => [services...].
  service_map = collections.defaultdict(list)
  for api_class in api_classes:
    service_map[(api_class.api_info.name, api_class.api_info.version)].append(
        api_class)
  class DiscoveryHandler(webapp2.RequestHandler):
    """Returns a discovery document for known services."""

    def get(self, name, version):
      host = self.request.headers['Host']
      services = service_map.get((name, version))
      if not services:
        self.abort(404, 'Not Found')

      self.response.headers['Content-Type'] = 'application/json'
      json.dump(
          discovery.generate(services, host, base_path),
          self.response, indent=2, sort_keys=True, separators=(',', ':'))

  return DiscoveryHandler


def discovery_service_route(api_classes, base_path):
  """Returns a route to a handler which serves discovery documents.

  Args:
    api_classes: a list of protorpc.remote.Service classes the handler should
      know about.
    base_path: The base path under which all service paths exist.

  Returns:
    A webapp2.Route.
  """
  return webapp2.Route(
      '%s/discovery/v1/apis/<name>/<version>/rest' % base_path,
      discovery_handler_factory(api_classes, base_path))


def directory_handler_factory(api_classes, base_path):
  """Returns a directory request handler which knows about the given services.

  Args:
    api_classes: A list of protorpc.remote.Service classes the handler should
      know about.
    base_path: The base path under which all service paths exist.

  Returns:
    A webapp2.RequestHandler.
  """
  class DirectoryHandler(webapp2.RequestHandler):
    """Returns a directory list for known services."""

    def get(self):
      host = self.request.headers['Host']
      self.response.headers['Content-Type'] = 'application/json'
      json.dump(
          discovery.directory(api_classes, host, base_path),
          self.response, indent=2, sort_keys=True, separators=(',', ':'))

  return DirectoryHandler


def directory_service_route(api_classes, base_path):
  """Returns a route to a handler which serves a directory list.

  Args:
    api_classes: A list of protorpc.remote.Service classes the handler should
      know about.
    base_path: The base path under which all service paths exist.

  Returns:
    A webapp2.Route.
  """
  return webapp2.Route(
      '%s/discovery/v1/apis' % base_path,
      directory_handler_factory(api_classes, base_path))


def explorer_proxy_route(base_path):
  """Returns a route to a handler which serves an API explorer proxy.

  Args:
    base_path: The base path under which all service paths exist.

  Returns:
    A webapp2.Route.
  """
  class ProxyHandler(webapp2.RequestHandler):
    """Returns a proxy capable of handling requests from API explorer."""

    def get(self):
      self.response.write(template.render(
          'adapter/proxy.html', params={'base_path': base_path}))

  template.bootstrap({
      'adapter': os.path.join(THIS_DIR, 'templates'),
  })
  return webapp2.Route('%s/static/proxy.html' % base_path, ProxyHandler)


def explorer_redirect_route(base_path):
  """Returns a route to a handler which redirects to the API explorer.

  Args:
    base_path: The base path under which all service paths exist.

  Returns:
    A webapp2.Route.
  """
  class RedirectHandler(webapp2.RequestHandler):
    """Returns a handler redirecting to the API explorer."""

    def get(self):
      host = self.request.headers['Host']
      self.redirect('https://apis-explorer.appspot.com/apis-explorer'
                    '/?base=https://%s%s' % (host, base_path))

  return webapp2.Route('%s/explorer' % base_path, RedirectHandler)
