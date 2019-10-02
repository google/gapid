# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""pRPC client.

Supports components.auth-based authentication, including delegation tokens.
Retries requests on transient errors.
"""

import collections

from google.appengine.ext import ndb
from google.protobuf import symbol_database

from components import net
from components.prpc import codes
from components.prpc import encoding

_BINARY_MEDIA_TYPE = encoding.Encoding.media_type(encoding.Encoding.BINARY)


# A low-level pRPC request to be sent using components.net module.
# Most clients should use Client class instead.
# Use new_request to create a new request.
Request = collections.namedtuple('Request', [
    # hostname of the pRPC server, e.g. "app.example.com".
    # Must not contain a scheme.
    'hostname',
    # True if the client must use HTTP, as opposed to HTTPS.
    # Useful for local servers.
    'insecure',
    # Full name of the service, including the package name,
    # e.g. "mypackage.MyService".
    'service_name',
    # Name of the RPC method.
    'method_name',
    # The request message.
    'request_message',
    # Target response message.
    'response_message',
    # A dict of call metadata. Will be available to the server.
    'metadata',
    # RPC timeout in seconds.
    'timeout',
    # OAuth2 scopes for the access token (ok skip auth if None).
    'scopes',
    # components.auth.ServiceAccountKey with credentials.
    'service_account_key',
    # delegation token returned by components.auth.delegate.
    'delegation_token',
    # how many times to retry on errors (4 times by default).
    'max_attempts',
])


def new_request(
    hostname, service_name, method_name, request_message, response_message,
    **kwargs):
  """Creates a Request object. Provides defaults for optional fields."""
  ret = Request(
      hostname=hostname,
      insecure=False,
      service_name=service_name,
      method_name=method_name,
      request_message=request_message,
      response_message=response_message,
      metadata=None,
      timeout=None,
      scopes=None,
      service_account_key=None,
      delegation_token=None,
      max_attempts=None,
  )
  return ret._replace(**kwargs)


class Error(Exception):
  """A base class for pRPC client-side errors."""


class RpcError(Error):
  """Raised when an RPC terminated with non-OK status.

  Use attribute status_code of type codes.StatusCode to decide
  how to handle the error.
  """

  def __init__(self, message, status_code, metadata):
    super(RpcError, self).__init__(message)
    self.status_code = status_code
    self.metadata = metadata


class ProtocolError(Error):
  """Server returned a malformed pRPC response."""


@ndb.tasklet
def rpc_async(req):
  """Sends an asynchronous pRPC request.

  This API is low level. Most users should use Client class instead.

  Args:
    req (Request): a pRPC request.

  Returns the response message if the RPC status code is OK.
  Otherwise raises an Error.
  """

  # The protocol is documented in
  # https://godoc.org/go.chromium.org/luci/grpc/prpc#hdr-Protocol

  # Ensure timeout is set, such that we use same values for deadline
  # parameter in net.request_async and X-Prpc-Timeout value are same.
  # Default to 10s, which is the default used in net.request_async.
  timeout = req.timeout or 10

  headers = (req.metadata or {}).copy()
  headers['Content-Type'] = _BINARY_MEDIA_TYPE
  headers['Accept'] = _BINARY_MEDIA_TYPE
  headers['X-Prpc-Timeout'] = '%dS' % timeout

  try:
    res_bytes = yield net.request_async(
        url='http%s://%s/prpc/%s/%s' % (
            '' if req.insecure else 's',
            req.hostname,
            req.service_name,
            req.method_name,
        ),
        method='POST',
        payload=req.request_message.SerializeToString(),
        headers=headers,
        scopes=req.scopes,
        service_account_key=req.service_account_key,
        delegation_token=req.delegation_token,
        deadline=timeout,
        max_attempts=req.max_attempts or 4,
    )
    # Unfortunately, net module does not expose headers of HTTP 200
    # responses.
    # Assume (HTTP OK => pRPC OK).
  except net.Error as ex:
    # net.Error means HTTP status code was not 200.
    try:
      code = codes.INT_TO_CODE[int(ex.headers['X-Prpc-Grpc-Code'])]
    except (ValueError, KeyError, TypeError):
      raise ProtocolError(
          'response does not contain a valid X-Prpc-Grpc-Code header')
    msg = ex.response.decode('utf-8', 'ignore')
    raise RpcError(msg, code, ex.headers)

  # Status code is OK.
  # Parse the response and return it.
  res = req.response_message
  res.ParseFromString(res_bytes)
  raise ndb.Return(res)


def service_account_credentials(service_account_key=None):
  """Returns credentials that use GAE service account.

  The returned value can be used as "credentials" argument in RPC method calls.
  """
  return lambda req: req._replace(
      scopes=[net.EMAIL_SCOPE],
      service_account_key=service_account_key,
  )


def delegation_credentials(delegation_token):
  """Returns credentials that use a LUCI delegation token.

  The returned value can be used as "credentials" argument in RPC method calls.
  """
  return lambda req: req._replace(delegation_token=delegation_token)


def composite_call_credentials(*call_credentials):
  """Combines multiple credentials.

  The returned value can be used as "credentials" argument in RPC method calls.
  """
  def fn(req):
    for mut in call_credentials:
      req = mut(req)
    return req
  return fn


class Client(object):
  """A client of a pRPC service.

  For each RPC method, a client has two instance methods,
  one sync method with the same name, and one async method with "Async"
  suffix.
  The async method returns an ndb.Future of the response.
  For example, for an RPC "Ping", the class will have methods Ping and
  PingAsync.

  Both async and async methods accept arguments:
    request: the request message.
    timeout: optional RPC timeout in seconds. Defaults to 10s.
    metadata: optional dict of call metadata. Will be available to the server.
    credentials: optional call credentials. Create them using credential
      functions in this module.
  On OK status code, they return the response message.
  Otherwise, they raise an Error.
  """

  def __init__(
      self, hostname, service_description, insecure=False):
    """Initializes a new pRPC Client.

    Args:
      hostname: hostname of the pRPC server, e.g. "app.example.com".
        Must not contain a schema.
      service_description: a service description object from a generated
        _prpc_pb2.py file.
      insecure: True if the client must use HTTP, as opposed to HTTPS.
        Useful for local servers.
    """
    self._hostname = hostname
    self._insecure = insecure

    desc = service_description['service_descriptor']
    self._full_service_name = desc.name
    pkg = service_description['file_descriptor'].package
    if pkg:
      self._full_service_name = '%s.%s' % (pkg, desc.name)

    for method_desc in desc.method:
      self._generate_rpc_method(method_desc)

  def _generate_rpc_method(self, method_desc):
    sym_db = symbol_database.Default()
    response_py_type = sym_db.GetSymbol(method_desc.output_type[1:])
    assert response_py_type, 'response type for %s.%s not found' % (
        self._full_service_name, method_desc.name)

    def method_async(  # pylint: disable=redefined-outer-name
        request, timeout=None, metadata=None, credentials=None):
      # The signature of this function must match
      # https://grpc.io/grpc/python/grpc.html#grpc.UnaryUnaryMultiCallable.__call__

      prpc_req = new_request(
          hostname=self._hostname,
          insecure=self._insecure,
          service_name=self._full_service_name,
          method_name=method_desc.name,
          request_message=request,
          response_message=response_py_type(),
          metadata=metadata,
          timeout=timeout,
      )

      if credentials:
        assert hasattr(credentials, '__call__'), (
          'credentials must be created using credentials functions in '
          'components.prpc.client module')
        prpc_req = credentials(prpc_req)

      return rpc_async(prpc_req)

    def method(*args, **kwargs):
      return method_async(*args, **kwargs).get_result()

    # Expose two instance methods for each RPC method: one async and one sync.
    method.__name__ = str(method_desc.name)
    method_async.__name__ = method.__name__ + 'Async'
    setattr(self, method.__name__, method)
    setattr(self, method_async.__name__, method_async)
