# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Json serializable properties."""

from google.appengine.ext import ndb
from google.appengine.api import datastore_errors

from components import utils


__all__ = [
  'BytesSerializable',
  'BytesSerializableProperty',
  'JsonSerializable',
  'JsonSerializableProperty',
  'READABLE',
  'SerializableModelMixin',
  'WRITABLE',
  'register_converter',
]

# Access to a protected member _XX of a client class - pylint: disable=W0212
# Method could be a function - pylint: disable=R0201


# Use field when converting entity to a serializable dict.
READABLE = 1 << 0
# Use field when converting entity from a serializable dict.
WRITABLE = 1 << 1


### Private stuff.


_rich_to_simple_converters = []
_simple_to_rich_converters = []

# Properties with values that look exactly the same in to_dict() and
# in to_serializable_dict() representations.
_SIMPLE_PROPERTIES = (
  ndb.BlobProperty,
  ndb.BooleanProperty,
  ndb.FloatProperty,
  ndb.IntegerProperty,
  ndb.JsonProperty,
  ndb.PickleProperty,
  ndb.StringProperty,
  ndb.TextProperty,
)


def _register_simple_converters():
  noop = lambda _prop, x: x
  for simple_prop_cls in _SIMPLE_PROPERTIES:
    register_converter(
        property_cls=simple_prop_cls,
        include_subclasses=False,
        rich_to_simple=noop,
        simple_to_rich=noop)


class _ModelDictConverter(object):
  """Uses |property_converters| to recursively convert dictionary values.

  Works by simultaneously walking over dict and and entity's structure. Dict
  is used for actual values, entity is used for typing information.

  Used for conversion in both directions: rich-typed dict to serializable dict
  and vice versa. The difference is in |property_converters| used.

  For example when converting in 'rich-typed to serializable dict' direction,
  |property_converters| contains functions that take rich types (e.g. datetime)
  and produce simple types (e.g int with timestamp). For reverse direction,
  |property_converters| contain functions that perform reverse conversion
  (e.g. int timestamp -> datetime).
  """

  def __init__(self, property_converters, field_mode_predicate):
    """Args:
      property_converters: sequence of tuples that define how to handle various
        NDB property classes.
      field_mode_predicate: callable that will be used to decide what properties
        to use during conversion. It is called with single integer argument
        |mode| which is a value from entity.serializable_properties dictionary
        that correspond to property being considered. If |field_mode_predicate|
        returns True, then property will be used, otherwise it will be silently
        ignored during conversion (i.e. resulting dict will not have it even
        if it was present in incoming dict).

    Each property converter tuple has 3 components:
      * ndb.Property subclass this converter applies to.
      * Boolean: True to apply converter to all subclasses or False only to
        this specific class.
      * Actual converter: function(property instance, from type) -> to type.

    For instance when converting rich-typed dict to serializable dict, converter
    for DateTimeProperty will be defined as:
      (
        ndb.DateTimeProperty,
        False,
        lambda(ndb.DateTimeProperty instance, datetime) -> integer
      )
    """
    self.property_converters = property_converters
    self.field_mode_predicate = field_mode_predicate

  def convert_dict(self, model_cls, model_dict):
    """Returns new dictionary with values converted using |property_converters|.

    Args:
      model_cls: ndb.Model subclass that acts as a schema with type information,
        its model_cls._properties will be used as a source of typing information
        for corresponding keys in |model_dict|.
      model_dict: dictionary that has same structure as entity defined by
        model_cls. Its values will be passed through appropriate property
        converters to get values in returned dict.

    Returns:
      New dictionary that structurally is a subset of |model_dict|, but with
      values of different type (defined by |property_converters|).
    """
    if not isinstance(model_dict, dict):
      raise ValueError(
          'Expecting a dict, got \'%s\' instead' % type(model_dict).__name__)
    allowed_properties = self.get_allowed_properties(model_cls)
    result = {}
    for key, value in model_dict.iteritems():
      if allowed_properties is None or key in allowed_properties:
        result[key] = self.convert_property(model_cls._properties[key], value)
    return result

  def convert_property(self, prop, value):
    """Converts value of a single key.

    Args:
      prop: instance of ndb.Property subclass that defines typing information.
      values: incoming property value to transform.

    Returns:
      Transformed property value that should be used in resulting dictionary.
      Uses |prop| and |property_converters| to figure out how to perform the
      conversion.
    """
    if prop._repeated:
      # Do not allow None here. NDB doesn't accept None as a valid value for
      # repeated property in populate(...) or entity constructor.
      if not isinstance(value, (list, tuple)):
        raise ValueError(
            'Expecting a list or tuple for \'%s\', got \'%s\' instead' % (
                prop._name, type(value).__name__))
      converter = self.get_property_converter(prop)
      return [converter(prop, x) for x in value]

    # For singular properties pass None as is.
    if value is None:
      return None
    converter = self.get_property_converter(prop)
    return converter(prop, value)

  def get_allowed_properties(self, model_cls):
    """Returns a set of property names to consider when converting a dictionary.

    When working with StructuredProperty based on regular ndb.Model, export all
    fields. Otherwise use model_cls.serializable_properties and
    self.field_mode_predicate to figure out set of properties to use.

    Return value of None means all defined properties should be used.
    """
    assert issubclass(model_cls, ndb.Model)
    assert not issubclass(model_cls, ndb.Expando), 'Expando is not supported'
    if not issubclass(model_cls, SerializableModelMixin):
      return None
    if model_cls.serializable_properties is None:
      return None
    return set(
        field for field, mode in model_cls.serializable_properties.iteritems()
        if self.field_mode_predicate(mode))

  def get_property_converter(self, prop):
    """Returns callable that can convert values corresponding to ndb property.

    Args:
      prop: instance of ndb.Property subclass that defines typing information.

    Returns:
      Callable (property instance, incoming value) -> converter values.
    """
    # For structured properties, recursively call convert_dict.
    if isinstance(prop, (ndb.StructuredProperty, ndb.LocalStructuredProperty)):
      return lambda prop, x: self.convert_dict(prop._modelclass, x)
    # For other properties consult the registry of converters.
    for prop_cls, include_subclasses, conv in self.property_converters:
      # pylint: disable=unidiomatic-typecheck
      if (include_subclasses and isinstance(prop, prop_cls) or
          not include_subclasses and type(prop) == prop_cls):
        return conv
    # Give up.
    raise TypeError('Don\'t know how to work with %s' % type(prop).__name__)


### Public API.


class SerializableModelMixin(object):
  """Mixing for entity that can convert itself to/from serializable dictionary.

  A serializable dictionary trivially can be converted to/from JSON, XML, YAML,
  etc. via standard serializers (e.g json.dump and json.load).

  A serializable dictionary is a dictionary with string keys and values that are
    * Scalar types: int, long, float.
    * String types: str, unicode.
    * Sequences: list, tuple.
    * Another serializable dictionaries.
  """

  # Dictionary: property name -> bit mask with READABLE and\or WRITABLE flag.
  # It defines what properties to use when convert an entity to or from
  # serializable dict. See doc strings for 'to_serializable_dict' and
  # 'convert_serializable_dict' for more details.
  # Default is None, which means that all defined properties are readable
  # and writable.
  serializable_properties = None

  def to_serializable_dict(self, with_id_as=None, exclude=None):
    """Converts this entity to a serializable dictionary.

    Operates only on properties that have READABLE flag set in
    |serializable_properties|. All other entity properties are effectively
    invisible.

    Args:
      with_id_as: name of the optional dict key to put entity's string_id() to.
      exclude: list of fields to exclude from the dict.
    """
    # TODO(vadimsh): Add 'include' and 'exclude' support when needed.
    conv = _ModelDictConverter(
        property_converters=_rich_to_simple_converters,
        field_mode_predicate=lambda mode: bool(mode & READABLE))
    serializable_dict = conv.convert_dict(
        self.__class__, self.to_dict(exclude=exclude))
    if with_id_as:
      assert isinstance(with_id_as, basestring)
      serializable_dict[with_id_as] = self.key.string_id()
    return serializable_dict

  @classmethod
  def from_serializable_dict(cls, serializable_dict, **props):
    """Makes an entity with properties from |serializable_dict| and |props|.

    Properties from |serializable_dict| are converted from simple types to
    rich types first (e.g. int -> DateTimeProperty). See doc string for
    'convert_serializable_dict' method for more details.

    Properties from |props| are passed to entity constructor as is. Values in
    |props| override values from |serializable_dict|.

    Raises ValueError if types or structure of |serializable_dict| doesn't match
    entity schema.
    """
    try:
      all_props = cls.convert_serializable_dict(serializable_dict)
      all_props.update(props)
      return cls(**all_props)
    except datastore_errors.BadValueError as e:
      raise ValueError(e)

  @classmethod
  def convert_serializable_dict(cls, serializable_dict):
    """Converts a serializable dictionary to dictionary with rich-typed values.

    It can then be used in entity constructor or in 'populate' method. This
    method works as reverse of to_serializable_dict, in particular if all
    fields are readable and writable the following holds:

    ent = Entity(...)
    assert ent == Entity(
        **Entity.convert_serializable_dict(ent.to_serializable_dict()))

    Operates only on properties that have WRITABLE flag set in
    |serializable_properties|. All other keys from |serializable_dict|
    (i.e. ones that don't match any entity properties at all or ones that match
    properties not explicitly marked as WRITABLE) are silently ignored.
    """
    conv = _ModelDictConverter(
        property_converters=_simple_to_rich_converters,
        field_mode_predicate=lambda mode: bool(mode & WRITABLE))
    return conv.convert_dict(cls, serializable_dict)


class BytesSerializable(object):
  """Interface that defines to_bytes() and from_bytes() methods.

  Objects that implement this interface know how to serialize/deserialize
  themselves to/from bytes array (represented by 'str').
  """

  def to_bytes(self):
    """Serialize this object to byte array."""
    raise NotImplementedError()

  @classmethod
  def from_bytes(cls, byte_buf):
    """Deserialize byte array into new instance of the class."""
    raise NotImplementedError()


class JsonSerializable(object):
  """Interface that defines to_jsonish() and from_jsonish() methods.

  Value is 'jsonish' if it can be converted to JSON with standard json.dump.

  Objects that implement this interface know how to convert themselves to/from
  jsonish values (usually dicts but not necessarily).
  """

  def to_jsonish(self):
    """Convert this object to jsonish value."""
    raise NotImplementedError()

  @classmethod
  def from_jsonish(cls, obj):
    """Given jsonish value convert it to new instance of the class."""
    raise NotImplementedError()


class BytesSerializableProperty(ndb.BlobProperty):
  """BlobProperty that uses values's to_bytes/from_bytes methods.

  Property will use to_bytes() to serialize an object before storing it in
  DB and from_bytes() when fetching it from DB and validating.

  Usage:
    class MyValue(BytesSerializable):
      ...

    class MyValueProperty(BytesSerializableProperty):
      _value_type = MyValue

    class Model(ndb.Model):
      my_value = MyValueProperty()
  """

  # Should be set in subclasses to some BytesSerializable subclass that this
  # property class will represent.
  _value_type = None

  def _validate(self, value):
    if not isinstance(value, self._value_type):
      raise TypeError(
          'Expecting %s, got %r' % (self._value_type.__name__, value))

  def _to_base_type(self, value):
    result = value.to_bytes()
    assert isinstance(result, str)
    return result

  def _from_base_type(self, value):
    assert isinstance(value, str)
    result = self._value_type.from_bytes(value)
    assert isinstance(result, self._value_type)
    return result


class JsonSerializableProperty(ndb.JsonProperty):
  """JsonProperty that uses values's to_jsonish/from_jsonish methods.

  Property will use to_jsonish() to convert an object to simple JSONish value
  before storing it in DB as JSON and from_jsonish() when fetching it from
  DB and validating.

  Usage:
    class MyValue(JsonSerializable):
      ...

    class MyValueProperty(JsonSerializableProperty):
      _value_type = MyValue

    class Model(ndb.Model):
      my_value = MyValueProperty()
  """

  # Should be set in subclasses to some JsonSerializable subclass that this
  # property class will represent.
  _value_type = None

  def _validate(self, value):
    if not isinstance(value, self._value_type):
      raise TypeError(
          'Expecting %s, got %r' % (self._value_type.__name__, value))

  def _to_base_type(self, value):
    return value.to_jsonish()

  def _from_base_type(self, value):
    return self._value_type.from_jsonish(value)


def register_converter(
    property_cls, include_subclasses, rich_to_simple, simple_to_rich):
  """Register a pair of functions that can convert some ndb.Property subclass.

  Used by ndb.Model, utils.SerializableModelMixin to convert entities to
  serializable dicts and vice versa.

  Args:
    property_cls: ndb.Property subclass.
    include_subclasses: True to apply this converter to all subclasses as well.
    rich_to_simple: function that converts property's value type to some simple
        type: rich_to_simple(property_instance, property_value) -> simple_value.
    simple_to_rich: function that converts some simple type to property's value
        type: simple_to_rich(property_instance, simple_value) -> property_value.
  """
  assert issubclass(property_cls, ndb.Property)
  _rich_to_simple_converters.append(
      (property_cls, include_subclasses, rich_to_simple))
  _simple_to_rich_converters.append(
      (property_cls, include_subclasses, simple_to_rich))


### Function calls.


_register_simple_converters()


# TODO(vadimsh): Add ndb.DateProperty if needed.
register_converter(
    property_cls=ndb.DateTimeProperty,
    include_subclasses=False,
    rich_to_simple=lambda _prop, x: utils.datetime_to_timestamp(x),
    simple_to_rich=lambda _prop, x: utils.timestamp_to_datetime(x))


# Handles all property classes inherited from JsonSerializableProperty.
register_converter(
    property_cls=JsonSerializableProperty,
    include_subclasses=True,
    rich_to_simple=lambda prop, value: value.to_jsonish(),
    simple_to_rich=lambda prop, value: prop._value_type.from_jsonish(value))


# Handles all property classes inherited from BytesSerializableProperty.
register_converter(
    property_cls=BytesSerializableProperty,
    include_subclasses=True,
    rich_to_simple=lambda prop, value: value.to_bytes(),
    simple_to_rich=lambda prop, value: prop._value_type.from_bytes(value))
