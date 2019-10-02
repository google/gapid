# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Useful custom properties."""

import json

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components import utils


__all__ = [
  'BytesComputedProperty',
  'DeterministicJsonProperty',
  'ProtobufProperty',
]

# Some methods below don't use self because they implement an interface of their
# base class.
# pylint: disable=no-self-use


### Other specialized properties.


class BytesComputedProperty(ndb.ComputedProperty):
  """Adds support to ComputedProperty for raw binary data.

  Use this class instead of ComputedProperty if the returned data is raw binary
  and not utf-8 compatible, as ComputedProperty assumes.
  """

  def _db_set_value(self, v, p, value):
    # From BlobProperty.
    p.set_meaning(ndb.google_imports.entity_pb.Property.BYTESTRING)
    v.set_stringvalue(value)


class DeterministicJsonProperty(ndb.BlobProperty):
  """Makes JsonProperty encoding deterministic where the same data results in
  the same blob all the time.

  For example, a dict is guaranteed to have its keys sorted, the whitespace
  separators are stripped, encoding is set to utf-8 so the output is constant.

  Sadly, we can't inherit from JsonProperty because it would result in
  duplicate encoding. So copy-paste the class from SDK v1.9.0 here.
  """
  _json_type = None

  @ndb.utils.positional(1 + ndb.BlobProperty._positional)
  def __init__(self, name=None, compressed=False, json_type=None, **kwds):
    super(DeterministicJsonProperty, self).__init__(
        name=name, compressed=compressed, **kwds)
    self._json_type = json_type

  def _validate(self, value):
    if self._json_type is not None and not isinstance(value, self._json_type):
      # Add the property name, otherwise it's annoying to try to figure out
      # which property is incorrect.
      raise TypeError(
          'Property %s must be a %s' % (self._name, self._json_type))

  def _to_base_type(self, value):
    """Makes it deterministic compared to ndb.JsonProperty._to_base_type()."""
    return utils.encode_to_json(value)

  def _from_base_type(self, value):
    return json.loads(value)


class ProtobufProperty(ndb.BlobProperty):
  """A property that stores a protobuf message in binary format.

  Supports length limiting and compression. Not indexable.
  """
  _message_class = None
  _max_length = None

  @ndb.utils.positional(2 + ndb.BlobProperty._positional)
  def __init__(
      self, message_class, name=None, compressed=False, max_length=None,
      **kwds):
    super(ProtobufProperty, self).__init__(
        name=name, compressed=compressed, **kwds)
    assert message_class, message_class
    self._message_class = message_class
    self._max_length = max_length

  def _validate(self, value):
    if not isinstance(value, self._message_class):
      # Add the property name, otherwise it's annoying to try to figure out
      # which property is incorrect.
      raise TypeError(
          'Property %s must be a %s' % (self._name, self._message_class))
    if self._max_length is not None and value.ByteSize() > self._max_length:
      raise datastore_errors.BadValueError(
          'Property %s is more than %d bytes' % (self._name, self._max_length))

  def _to_base_type(self, value):
    """Interprets value as a protobuf message and serialized to bytes."""
    return value.SerializeToString()

  def _from_base_type(self, value):
    """Interprets value as bytes and deserializes it to a protobuf message."""
    msg = self._message_class()
    msg.ParseFromString(value)
    return msg
