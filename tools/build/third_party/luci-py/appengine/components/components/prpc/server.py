# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""A stripped-down implementation of the gRPC interface for Python on AppEngine.

The Server class itself also provides the bulk of the implementation which
actually runs on AppEngine (and therefore couldn't include Cython). It acts
as a webapp2.RequestHandler, and exposes a .get_routes() method for the host
application to call.

https://github.com/grpc/grpc/tree/master/src/python/grpcio/grpc
"""

import collections
import httplib
import logging
import webapp2

from google.protobuf import symbol_database

# Helpers are in separate modules so this one exposes only the public interface.
from components.prpc import discovery
from components.prpc import encoding
from components.prpc import headers
from components.prpc.codes import StatusCode
from components.prpc.context import ServicerContext


__all__ = [
  'HandlerCallDetails',
  'Server',
  'StatusCode',
]

# pylint: disable=pointless-string-statement

_PRPC_TO_HTTP_STATUS = {
  StatusCode.OK: httplib.OK,
  StatusCode.CANCELLED: httplib.NO_CONTENT,
  StatusCode.INVALID_ARGUMENT: httplib.BAD_REQUEST,
  StatusCode.DEADLINE_EXCEEDED: httplib.SERVICE_UNAVAILABLE,
  StatusCode.NOT_FOUND: httplib.NOT_FOUND,
  StatusCode.ALREADY_EXISTS: httplib.CONFLICT,
  StatusCode.PERMISSION_DENIED: httplib.FORBIDDEN,
  StatusCode.RESOURCE_EXHAUSTED: httplib.SERVICE_UNAVAILABLE,
  StatusCode.FAILED_PRECONDITION: httplib.PRECONDITION_FAILED,
  StatusCode.OUT_OF_RANGE: httplib.BAD_REQUEST,
  StatusCode.UNIMPLEMENTED: httplib.NOT_IMPLEMENTED,
  StatusCode.INTERNAL: httplib.INTERNAL_SERVER_ERROR,
  StatusCode.UNAVAILABLE: httplib.SERVICE_UNAVAILABLE,
  StatusCode.UNAUTHENTICATED: httplib.UNAUTHORIZED,
}


_Service = collections.namedtuple('_Service', ['servicer', 'methods'])


# Details about the RPC call passed to the interceptors.
HandlerCallDetails = collections.namedtuple('HandlerCallDetails', [
  'method',               # full method name <service>.<method>
  'invocation_metadata',  # (k, v) pairs list with metadata sent by the client
])


class Server(object):
  """Server represents a pRPC server that handles a set of services.

  Server is intended to vaguely mimic gRPC's Server as an abstraction, but
  provides a simpler interface via add_service and get_routes.
  """

  def __init__(self):
    self._services = {}
    self._interceptors = ()
    self._discovery_service = discovery.Discovery()
    self.add_service(self._discovery_service)

  def add_interceptor(self, interceptor):
    """Adds an interceptor to the interceptor chain.

    Interceptors can be used to examine or modify requests before they reach
    the real handlers. The API is vaguely similar to grpc.ServerInterceptor.

    An interceptor is a callback that accepts following arguments:
      request: deserialized request message.
      context: an instance of ServicerContext.
      call_details: an instance of HandlerCallDetails.
      continuation: a callback that resumes the processing, accepts
        (request, context, call_details).

    An interceptor may decide NOT to call the continuation if it handles the
    request itself.

    Args:
      interceptor: an interceptor callback to add to the chain.
    """
    self._interceptors = self._interceptors + (interceptor,)

  def add_service(self, servicer):
    """Registers a servicer for a service with the server.

    Args:
      servicer: A servicer which represents a service. It must have a
        DESCRIPTION of the service and implements handlers for each service
        method.

    Raises:
      ValueError: when trying to add another handler for the same service name.
    """
    sym_db = symbol_database.Default()
    pkg = servicer.DESCRIPTION['file_descriptor'].package
    desc = servicer.DESCRIPTION['service_descriptor']

    # Construct handler.
    methods = {
      method.name: (
        # Fully-qualified proto type names will always begin with a '.' which
        # GetSymbol doesn't strip out.
        sym_db.GetSymbol(method.input_type[1:]),
        sym_db.GetSymbol(method.output_type[1:]),
        getattr(servicer, method.name),
      )
      for method in desc.method if hasattr(servicer, method.name)
    }

    full_name = desc.name
    if pkg:
      full_name = '%s.%s' % (pkg, desc.name)

    # Register handler with internal server state.
    if desc.name in self._services:
      raise ValueError(
          'Tried to double-register handlers for service %s' % desc.name)
    self._services[full_name] = _Service(servicer, methods)

    self._discovery_service.add_service(servicer.DESCRIPTION)

  def get_routes(self):
    """Returns a list of webapp2.Route for all the routes the API handles."""
    return [webapp2.Route('/prpc/<service>/<method>',
                          handler=self._handler(),
                          methods=['POST', 'OPTIONS'])]

  def _handler(self):
    """Returns a RequestHandler class with access to this server's data."""

    # Alias self as server here for readability.
    server = self

    class Handler(webapp2.RequestHandler):

      def post(self, service, method):
        """Writes body and headers of webapp2.Response.

        Args:
          service: the service being targeted by this pRPC call.
          method: the method being invoked by this pRPC call.

        Returns:
          response: a webapp2.Response.
        """
        context = ServicerContext()
        content = self._handle(context, service, method)
        origin = self.request.headers.get('Origin')
        if origin:
          self.response.headers['Access-Control-Allow-Origin'] = origin
          self.response.headers['Vary'] = 'Origin'
          self.response.headers['Access-Control-Allow-Credentials'] = 'true'
        self.response.status = _PRPC_TO_HTTP_STATUS[context._code]
        self.response.headers['X-Prpc-Grpc-Code'] = str(context._code.value)
        self.response.headers['Access-Control-Expose-Headers'] = (
            'X-Prpc-Grpc-Code')
        self.response.headers['X-Content-Type-Options'] = 'nosniff'
        if content is not None:
          self.response.headers['Content-Type'] = encoding.Encoding.media_type(
              context._response_encoding)
          self.response.out.write(content)
        elif context._details is not None:
          # webapp2 will automatically encode strings as utf-8.
          # http://webapp2.readthedocs.io/en/latest/guide/response.html
          #
          # TODO(nodir,mknyszek): Come up with an actual test for this.
          self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
          self.response.out.write(context._details)
        return self.response

      def _handle(self, context, service, method):
        """Generates the response content and sets the context appropriately.

        Sets context._request_encoding and context._response_encoding.

        Args:
          context: a context.ServicerContext.
          service: the service being targeted by this pRPC call.
          method: the method being invoked by this pRPC call.

        Returns:
          content: the binary or textual content of the RPC response. Note
            that this may be None in the event that an error occurs.
        """
        try:
          parsed_headers = headers.parse_headers(self.request.headers)
          context._request_encoding = parsed_headers.content_type
          context._response_encoding = parsed_headers.accept
        except ValueError as e:
          logging.warning('Error parsing headers: %s', e)
          context.set_code(StatusCode.INVALID_ARGUMENT)
          context.set_details(e.message)
          return None

        if service not in server._services:
          context.set_code(StatusCode.UNIMPLEMENTED)
          context.set_details('Service %s does not exist' % service)
          return None
        rpc_handler = server._services[service].methods.get(method)
        if rpc_handler is None:
          context.set_code(StatusCode.UNIMPLEMENTED)
          context.set_details('Method %s does not exist' % method)
          return None
        request_message, response_message, handler = rpc_handler

        request = request_message()
        try:
          decoder = encoding.get_decoder(parsed_headers.content_type)
          decoder(self.request.body, request)
        except Exception as e:
          logging.warning('Failed to decode request: %s', e, exc_info=True)
          context.set_code(StatusCode.INVALID_ARGUMENT)
          context.set_details('Error parsing request: %s' % e.message)
          return None

        context._timeout = parsed_headers.timeout
        context._invocation_metadata = parsed_headers.invocation_metadata

        # Only ipv6 addresses have ':' in them. Assume everything else is ipv4.
        if ':' in self.request.remote_addr:
          context._peer = 'ipv6:[%s]' % self.request.remote_addr
        else:
          context._peer = 'ipv4:%s' % self.request.remote_addr

        call_details = HandlerCallDetails(
            method='%s.%s' % (service, method),
            invocation_metadata=context.invocation_metadata())

        try:
          # TODO(nodir,mknyszek): Poll for context to hit timeout or be
          # canceled.
          response = server._run_interceptors(
              request, context, call_details, handler, 0)
        except Exception:
          logging.exception('Service implementation threw an exception')
          context.set_code(StatusCode.INTERNAL)
          context.set_details('Service implementation threw an exception')
          return None

        if response is None:
          if context._code == StatusCode.OK:
            context.set_code(StatusCode.INTERNAL)
            context.set_details(
                'Service implementation didn\'t return a response')
          return None

        if not isinstance(response, response_message):
          logging.error('Service implementation response has incorrect type')
          context.set_code(StatusCode.INTERNAL)
          context.set_details('Service implementation returned invalid value')
          return None

        try:
          encoder = encoding.get_encoder(parsed_headers.accept)
          content = encoder(response)
        except Exception:
          logging.exception('Failed to encode response')
          context.set_code(StatusCode.INTERNAL)
          context.set_details('Error serializing response')
          return None

        return content

      # pylint: disable=unused-argument
      def options(self, service, method):
        """Sends an empty response with headers for CORS for all requests."""
        origin = self.request.headers.get('Origin')
        if origin:
          self.response.headers['Access-Control-Allow-Origin'] = origin
          self.response.headers['Vary'] = 'Origin'
          self.response.headers['Access-Control-Allow-Credentials'] = 'true'
          self.response.headers['Access-Control-Allow-Headers'] = \
              'Origin, Content-Type, Accept, Authorization'
          self.response.headers['Access-Control-Allow-Methods'] = \
              'OPTIONS, POST'
          self.response.headers['Access-Control-Max-Age'] = '600'

    return Handler

  def _run_interceptors(self, request, context, call_details, handler, idx):
    """Runs the request via interceptors chain starting from given index.

    Args:
      request: deserialized request proto.
      context: a context.ServicerContext.
      handler: a final handler, given as callback (request, context): response.
      idx: an index in the interceptor chain to start from.

    Returns:
      Response message.
    """
    if idx == len(self._interceptors):
      return handler(request, context)

    def continuation(request, context, call_details):
      return self._run_interceptors(
          request, context, call_details, handler, idx+1)

    interceptor = self._interceptors[idx]
    return interceptor(request, context, call_details, continuation)
