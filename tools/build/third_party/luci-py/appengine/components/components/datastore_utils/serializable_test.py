#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components.datastore_utils import serializable
from test_support import test_case


# Access to a protected member _XX of a client class - pylint: disable=W0212


class SerializableModelTest(test_case.TestCase):
  """Tests for datastore_utils.serializable.SerializableModelMixin and related
  property converters.
  """

  def test_simple_properties(self):
    """Simple properties are unmodified in to_serializable_dict()."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      blob_prop = ndb.BlobProperty()
      bool_prop = ndb.BooleanProperty()
      float_prop = ndb.FloatProperty()
      int_prop = ndb.IntegerProperty()
      json_prop = ndb.JsonProperty()
      pickle_prop = ndb.PickleProperty()
      str_prop = ndb.StringProperty()
      text_prop = ndb.TextProperty()

    # Test data in simple dict form.
    as_serializable_dict = {
      'blob_prop': 'blob',
      'bool_prop': True,
      'float_prop': 3.14,
      'int_prop': 42,
      'json_prop': ['a list', 'why', 'not?'],
      'pickle_prop': {'some': 'dict'},
      'str_prop': 'blah-blah',
      'text_prop': 'longer blah-blah',
    }

    # Same data but in entity form. Constructing entity directly from
    # |as_serializable_dict| works only if it contains only simple properties
    # (that look the same in serializable dict and entity form).
    as_entity = Entity(**as_serializable_dict)

    # Ensure all simple properties (from _SIMPLE_PROPERTIES) are covered.
    self.assertEqual(
        set(serializable._SIMPLE_PROPERTIES),
        set(prop.__class__ for prop in Entity._properties.itervalues()))

    # Check entity -> serializable dict conversion.
    self.assertEqual(
        as_serializable_dict,
        as_entity.to_serializable_dict())

    # Check serializable dict -> Entity conversion.
    self.assertEqual(
        as_entity,
        Entity.from_serializable_dict(as_serializable_dict))

  def test_serializable_properties(self):
    """Check that |serializable_properties| works as expected."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      serializable_properties = {
        'prop_rw': serializable.READABLE | serializable.WRITABLE,
        'prop_r': serializable.READABLE,
        'prop_w': serializable.WRITABLE,
        'prop_hidden_1': 0,
      }
      prop_r = ndb.IntegerProperty(default=0)
      prop_w = ndb.IntegerProperty(default=0)
      prop_rw = ndb.IntegerProperty(default=0)
      prop_hidden_1 = ndb.IntegerProperty(default=0)
      prop_hidden_2 = ndb.IntegerProperty(default=0)

    entity = Entity()

    # Only fields with READABLE flag set show up in to_serializable_dict().
    self.assertEqual(
        {'prop_r': 0, 'prop_rw': 0},
        entity.to_serializable_dict())

    # Fields with WRITABLE flag can be used in convert_serializable_dict.
    self.assertEqual(
        {'prop_rw': 1, 'prop_w': 2},
        Entity.convert_serializable_dict({'prop_rw': 1, 'prop_w': 2}))

    # Writable fields are optional.
    self.assertEqual(
        {'prop_rw': 1},
        Entity.convert_serializable_dict({'prop_rw': 1}))

    # convert_serializable_dict ignores read only, hidden and unrecognized keys.
    all_props = {
      'prop_r': 0,
      'prop_rw': 1,
      'prop_w': 2,
      'prop_hidden_1': 3,
      'prop_hidden_2': 4,
      'unknown_prop': 5,
    }
    self.assertEqual(
        {'prop_rw': 1, 'prop_w': 2},
        Entity.convert_serializable_dict(all_props))

  def test_entity_id(self):
    """Test that 'with_id_as' argument in to_serializable_dict is respected."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      pass
    self.assertEqual(
        {'my_id': 'abc'},
        Entity(id='abc').to_serializable_dict(with_id_as='my_id'))

  def test_datetime_properties(self):
    """Test handling of DateTimeProperty."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      dt = ndb.DateTimeProperty()

    # Same point in time as datetime and as timestamp.
    dt = datetime.datetime(2012, 1, 2, 3, 4, 5)
    ts = 1325473445000000

    # Datetime is serialized to a number of milliseconds since epoch.
    self.assertEqual({'dt': ts}, Entity(dt=dt).to_serializable_dict())
    # Reverse operation also works.
    self.assertEqual({'dt': dt}, Entity.convert_serializable_dict({'dt': ts}))

  def test_repeated_properties(self):
    """Test that properties with repeated=True are handled."""
    class IntsEntity(ndb.Model, serializable.SerializableModelMixin):
      ints = ndb.IntegerProperty(repeated=True)
    class DatesEntity(ndb.Model, serializable.SerializableModelMixin):
      dates = ndb.DateTimeProperty(repeated=True)

    # Same point in time as datetime and as timestamp.
    dt = datetime.datetime(2012, 1, 2, 3, 4, 5)
    ts = 1325473445000000

    # Repeated properties that are not set are converted to empty lists.
    self.assertEqual({'ints': []}, IntsEntity().to_serializable_dict())
    self.assertEqual({'dates': []}, DatesEntity().to_serializable_dict())

    # List of ints works (as an example of simple repeated property).
    self.assertEqual(
        {'ints': [1, 2]},
        IntsEntity(ints=[1, 2]).to_serializable_dict())
    self.assertEqual(
        {'ints': [1, 2]},
        IntsEntity.convert_serializable_dict({'ints': [1, 2]}))

    # List of datetimes works (as an example of not-so-simple property).
    self.assertEqual(
        {'dates': [ts, ts]},
        DatesEntity(dates=[dt, dt]).to_serializable_dict())
    self.assertEqual(
        {'dates': [dt, dt]},
        DatesEntity.convert_serializable_dict({'dates': [ts, ts]}))

  def _test_structured_properties_class(self, structured_cls):
    """Common testing for StructuredProperty and LocalStructuredProperty."""
    # Plain ndb.Model.
    class InnerSimple(ndb.Model):
      a = ndb.IntegerProperty()

    # With SerializableModelMixin.
    class InnerSmart(ndb.Model, serializable.SerializableModelMixin):
      serializable_properties = {
        'a': serializable.READABLE | serializable.WRITABLE,
      }
      a = ndb.IntegerProperty()
      b = ndb.IntegerProperty()

    class Outter(ndb.Model, serializable.SerializableModelMixin):
      simple = structured_cls(InnerSimple)
      smart = structured_cls(InnerSmart)

    # InnerSimple gets serialized entirely, while only readable fields
    # on InnerSmart are serialized.
    entity = Outter()
    entity.simple = InnerSimple(a=1)
    entity.smart = InnerSmart(a=2, b=3)
    self.assertEqual(
        {'simple': {'a': 1}, 'smart': {'a': 2}},
        entity.to_serializable_dict())

    # Works backwards as well. Note that 'convert_serializable_dict' returns
    # a dictionary that can be fed to entity's 'populate' or constructor. Entity
    # by itself is smart enough to transform subdicts into structured
    # properties.
    self.assertEqual(
        Outter(simple=InnerSimple(a=1), smart=InnerSmart(a=2)),
        Outter.from_serializable_dict({'simple': {'a': 1}, 'smart': {'a': 2}}))

  def _test_repeated_structured_properties_class(self, structured_cls):
    """Common testing for StructuredProperty and LocalStructuredProperty."""
    class Inner(ndb.Model):
      a = ndb.IntegerProperty()

    class Outter(ndb.Model, serializable.SerializableModelMixin):
      inner = structured_cls(Inner, repeated=True)

    # Repeated structured property -> list of dicts.
    entity = Outter()
    entity.inner.extend([Inner(a=1), Inner(a=2)])
    self.assertEqual(
        {'inner': [{'a': 1}, {'a': 2}]},
        entity.to_serializable_dict())

    # Reverse also works.
    self.assertEqual(
        entity,
        Outter.from_serializable_dict({'inner': [{'a': 1}, {'a': 2}]}))

  def test_structured_properties(self):
    """Test handling of StructuredProperty."""
    self._test_structured_properties_class(ndb.StructuredProperty)

  def test_local_structured_properties(self):
    """Test handling of LocalStructuredProperty."""
    self._test_structured_properties_class(ndb.LocalStructuredProperty)

  def test_repeated_structured_properties(self):
    """Test handling of StructuredProperty(repeated=True)."""
    self._test_repeated_structured_properties_class(ndb.StructuredProperty)

  def test_repeated_local_structured_properties(self):
    """Test handling of LocalStructuredProperty(repeated=True)."""
    self._test_repeated_structured_properties_class(ndb.LocalStructuredProperty)

  def test_exclude_works(self):
    """|exclude| argument of to_serializable_dict() is respected."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      prop1 = ndb.IntegerProperty()
      prop2 = ndb.IntegerProperty()
      prop3 = ndb.IntegerProperty()

    entity = Entity(prop1=1, prop2=2, prop3=3)
    self.assertEqual(
        {'prop1': 1, 'prop3': 3},
        entity.to_serializable_dict(exclude=['prop2']))

  def test_from_serializable_dict_kwargs_work(self):
    """Keyword arguments in from_serializable_dict are passed to constructor."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      prop = ndb.IntegerProperty()

    # Pass entity key via keyword parameters.
    entity = Entity.from_serializable_dict(
        {'prop': 123}, id='my id', parent=ndb.Key('Fake', 'parent'))
    self.assertEqual(123, entity.prop)
    self.assertEqual(ndb.Key('Fake', 'parent', 'Entity', 'my id'), entity.key)

  def test_from_serializable_dict_kwargs_precedence(self):
    """Keyword arguments in from_serializable_dict take precedence."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      prop = ndb.IntegerProperty()

    # Pass |prop| via serialized dict and as a keyword arg.
    entity = Entity.from_serializable_dict({'prop': 123}, prop=456)
    # Keyword arg wins.
    self.assertEqual(456, entity.prop)

  def test_bad_type_in_from_serializable_dict(self):
    """from_serializable_dict raises ValueError when seeing unexpected type."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      pass

    # Pass a list instead of dict.
    with self.assertRaises(ValueError):
      Entity.from_serializable_dict([])

  def test_bad_type_for_repeated_property(self):
    """Trying to deserialize repeated property not from a list -> ValueError."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      prop = ndb.IntegerProperty(repeated=True)

    # A list, tuple or nothing should work.
    Entity.from_serializable_dict({'prop': [1]})
    Entity.from_serializable_dict({'prop': (1,)})
    Entity.from_serializable_dict({})

    # A single item shouldn't.
    with self.assertRaises(ValueError):
      Entity.from_serializable_dict({'prop': 1})
    # 'None' shouldn't.
    with self.assertRaises(ValueError):
      Entity.from_serializable_dict({'prop': None})
    # Dict shouldn't.
    with self.assertRaises(ValueError):
      Entity.from_serializable_dict({'prop': {}})

  def test_bad_type_for_simple_property(self):
    """Trying to deserialize non-number into IntegerProperty -> ValueError."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      prop = ndb.IntegerProperty()

    # Works.
    Entity.from_serializable_dict({'prop': 123})
    # Doesn't.
    with self.assertRaises(ValueError):
      Entity.from_serializable_dict({'prop': 'abc'})

  def test_bad_type_for_datetime_property(self):
    """Trying to deserialize non-number into DateTimeProperty -> ValueError."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      prop = ndb.DateTimeProperty()

    # Works.
    Entity.from_serializable_dict({'prop': 123})
    # Doesn't.
    with self.assertRaises(ValueError):
      Entity.from_serializable_dict({'prop': 'abc'})


class BytesSerializableObject(serializable.BytesSerializable):
  def __init__(self, payload):  # pylint: disable=W0231
    self.payload = payload

  def to_bytes(self):
    return 'prefix:' + self.payload

  @classmethod
  def from_bytes(cls, byte_buf):
    return cls(byte_buf[len('prefix:'):])


class BytesSerializableObjectProperty(serializable.BytesSerializableProperty):
  _value_type = BytesSerializableObject


class BytesSerializableTest(test_case.TestCase):
  """Test BytesSerializable and its integration with SerializableModel."""

  def test_bytes_serializable(self):
    """Test to_serializable_dict and convert_serializable_dict."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      bytes_prop = BytesSerializableObjectProperty()

    # Ensure to_serializable_dict uses to_bytes.
    self.assertEqual(
        {'bytes_prop': 'prefix:hi'},
        Entity(bytes_prop=BytesSerializableObject('hi')).to_serializable_dict())

    # Ensure convert_serializable_dict uses from_bytes.
    entity = Entity.from_serializable_dict({'bytes_prop': 'prefix:hi'})
    self.assertEqual('hi', entity.bytes_prop.payload)


class JsonSerializableObject(serializable.JsonSerializable):
  def __init__(self, payload):  # pylint: disable=W0231
    self.payload = payload

  def to_jsonish(self):
    return {'payload': self.payload}

  @classmethod
  def from_jsonish(cls, obj):
    return cls(obj['payload'])


class JsonSerializableObjectProperty(serializable.JsonSerializableProperty):
  _value_type = JsonSerializableObject


class JsonSerializableTest(test_case.TestCase):
  """Test JsonSerializable and its integration with SerializableModel."""

  def test_json_serializable(self):
    """Test to_serializable_dict and convert_serializable_dict."""
    class Entity(ndb.Model, serializable.SerializableModelMixin):
      json_prop = JsonSerializableObjectProperty()

    # Ensure to_serializable_dict uses to_jsonish.
    self.assertEqual(
        {'json_prop': {'payload': [1, 2]}},
        Entity(json_prop=JsonSerializableObject([1, 2])).to_serializable_dict())

    # Ensure convert_serializable_dict uses from_jsonish.
    entity = Entity.from_serializable_dict(
        {'json_prop': {'payload': [1, 2]}})
    self.assertEqual([1, 2], entity.json_prop.payload)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
