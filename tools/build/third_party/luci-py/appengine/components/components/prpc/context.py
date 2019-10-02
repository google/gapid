# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import time

from components.prpc import codes


class ServicerContext(object):
  """A context object passed to method implementations."""

  # Note: guts of this object are used by Server, but RPC handlers MUST use only
  # public API.

  def __init__(self):
    self._start_time = time.time()
    self._timeout = None
    self._active = True
    self._code = codes.StatusCode.OK
    self._details = None
    self._invocation_metadata = []
    self._peer = None
    self._request_encoding = None
    self._response_encoding = None

  def invocation_metadata(self):
    """Accesses the metadata from the sent by the client.

    Returns:
      The invocation metadata as a list of (k, v) pairs, the key is lowercase.
    """
    return self._invocation_metadata

  def peer(self):
    """Identifies the peer that invoked the RPC being serviced.

    Returns:
      A string - either "ipv4:xxx.xxx.xxx.xxx" or "ipv6:[....]".
    """
    return self._peer

  def is_active(self):
    """Describes whether the RPC is active or has terminated.

    Returns:
      True if RPC is active, False otherwise.
    """
    return self._active

  def time_remaining(self):
    """Describes the length of allowed time remaining for the RPC.

    Returns:
      A nonnegative float indicating the length of allowed time in seconds
      remaining for the RPC to complete before it is considered to have timed
      out, or None if no deadline was specified for the RPC.
    """
    if self._timeout is None:
      return None
    now = time.time()
    return max(0, self._start_time + self._timeout - now)

  def cancel(self):
    """Cancels the RPC.

    Idempotent and has no effect if the RPC has already terminated.
    """
    self._active = False

  @property
  def code(self):
    """Returns current gRPC status code."""
    return self._code

  def set_code(self, code):
    """Accepts the status code of the RPC.

    This method need not be called by method implementations if they wish the
    gRPC runtime to determine the status code of the RPC.

    Args:
      code: One of StatusCode.* tuples that contains a status code of the RPC to
        be transmitted to the invocation side of the RPC.
    """
    assert code in codes.ALL_CODES, '%r is not StatusCode.*' % (code,)
    self._code = code

  @property
  def details(self):
    """Returns details set by set_details."""
    return self._details

  def set_details(self, details):
    """Accepts the service-side details of the RPC.

    This method need not be called by method implementations if they have no
    details to transmit.

    Args:
      details: The details string of the RPC to be transmitted to
        the invocation side of the RPC.
    """
    assert isinstance(details, basestring), '%r is not string' % (details,)
    self._details = details

  @property
  def request_encoding(self):
    """Returns prpc.Encoding of the request."""
    return self._request_encoding

  @property
  def response_encoding(self):
    """Returns prpc.Encoding of the response."""
    return self._response_encoding

  def clone(self):
    """Returns a shallow copy of self.

    This may be useful for running parallel handlers that each may return
    different results and may have different timeouts.
    """
    ret = ServicerContext()
    # pylint: disable=attribute-defined-outside-init
    ret.__dict__ = self.__dict__.copy()
    return ret
