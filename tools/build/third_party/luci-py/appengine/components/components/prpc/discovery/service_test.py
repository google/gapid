#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.protobuf import descriptor_pb2
from google.protobuf import text_format

from components.prpc.discovery import service
from components.prpc.discovery.test import test_pb2  # register messages
from components.prpc.discovery.test import test_prpc_pb2


class TestService(object):
  DESCRIPTION = test_prpc_pb2.TestServiceServiceDescription


EXPECTED_DESCRIPTION = '''
file {
  name: "test.proto"
  package: "discovery_test"
  dependency: "imported1_test.proto"
  message_type {
    name: "HelloRequest"
  }
  message_type {
    name: "HelloResponse"
    field {
      name: "imported"
      number: 1
      label: LABEL_OPTIONAL
      type: TYPE_MESSAGE
      type_name: ".discovery_test.Imported1"
      json_name: "imported"
    }
  }
  service {
    name: "TestService"
    method {
      name: "Hello"
      input_type: ".discovery_test.HelloRequest"
      output_type: ".discovery_test.HelloResponse"
      options {
      }
    }
  }
  syntax: "proto3"
}
file {
  name: "imported1_test.proto"
  package: "discovery_test"
  dependency: "imported2_test.proto"
  message_type {
    name: "Imported1"
    field {
      name: "imported"
      number: 1
      label: LABEL_OPTIONAL
      type: TYPE_MESSAGE
      type_name: ".discovery_test.Imported2"
      json_name: "imported"
    }
  }
  syntax: "proto3"
}
file {
  name: "imported2_test.proto"
  package: "discovery_test"
  message_type {
    name: "Imported2"
    field {
      name: "x"
      number: 1
      label: LABEL_OPTIONAL
      type: TYPE_INT32
      json_name: "x"
    }
  }
  syntax: "proto3"
}
'''


class DiscoveryServiceTests(unittest.TestCase):
  def test(self):
    serv = service.Discovery()
    serv.add_service(TestService.DESCRIPTION)
    res = serv.Describe(None, None)
    self.assertEquals(res.services, ['discovery_test.TestService'])

    for f in res.description.file:
      self.assertTrue(f.HasField('source_code_info'))
      f.ClearField('source_code_info')

    expected_description = descriptor_pb2.FileDescriptorSet()
    text_format.Merge(EXPECTED_DESCRIPTION, expected_description)
    self.assertEqual(expected_description, res.description)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
