#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components.datastore_utils import properties
from components.protoutil  import test_proto_pb2
from test_support import test_case


class BP(ndb.Model):
  prop = properties.BytesComputedProperty(lambda _: '\x00')


class DJP(ndb.Model):
  prop = properties.DeterministicJsonProperty(json_type=dict)


class PP(ndb.Model):
  prop = properties.ProtobufProperty(test_proto_pb2.Msg)
  prop_limited = properties.ProtobufProperty(test_proto_pb2.Msg, max_length=10)


class PropertiesTest(test_case.TestCase):
  def test_DeterministicJsonProperty(self):
    self.assertEqual({'a': 1}, DJP(prop={'a': 1}).prop)

    DJP(prop={'a': 1}).put()
    self.assertEqual({'a': 1}, DJP.query().get().prop)

    with self.assertRaises(TypeError):
      DJP(prop=[])

  def test_BytesComputedProperty(self):
    self.assertEqual('\x00', BP().prop)
    BP().put()
    self.assertEqual('\x00', BP.query().get().prop)

  def test_ProtobufProperty(self):
    m = test_proto_pb2.Msg(num=1)
    entity = PP(prop=m)
    self.assertEqual(m, entity.prop)

    entity.put()
    self.assertEqual(m, PP.query().get().prop)

    with self.assertRaises(TypeError):
      PP(prop='not a protobuf message')
    with self.assertRaises(datastore_errors.BadValueError):
      PP(prop_limited=test_proto_pb2.Msg(str='a very long string'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
