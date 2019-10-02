#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import textwrap
import unittest

from test_support import test_env
test_env.setup_test_env()

# google.protobuf package requires some python package magic hacking.
from components import utils
utils.fix_protobuf_package()

from google import protobuf

import protoutil
import test_proto_pb2

class MergeDictTests(unittest.TestCase):

  def test_works(self):
    msg = test_proto_pb2.Msg(
        str='s',
        strs=['s1', 's2'],
        num=1,
        nums=[1, 2],
        msg=test_proto_pb2.Msg(num=3),
        msgs=[test_proto_pb2.Msg(num=4), test_proto_pb2.Msg(num=5)]
    )
    data = {
        'str': 'a',
        'strs': ['a', 'b'],
        'num': 2,
        'nums': [3, 4],
        'msg': {
          'num': 5,
        },
        'msgs': [{
          'num': 6,
        }],
    }
    protoutil.merge_dict(data, msg)
    self.assertEqual(msg, test_proto_pb2.Msg(
        str='a',
        strs=['s1', 's2', 'a', 'b'],
        num=2,
        nums=[1, 2, 3, 4],
        msg=test_proto_pb2.Msg(num=5),
        msgs=[
            test_proto_pb2.Msg(num=4),
            test_proto_pb2.Msg(num=5),
            test_proto_pb2.Msg(num=6),
        ]
    ))

  def test_invalid_field_name(self):
    with self.assertRaises(TypeError):
      protoutil.merge_dict({'no_such_field': 0}, test_proto_pb2.Msg())

  def test_invalid_field_value_type(self):
    with self.assertRaises(TypeError):
      protoutil.merge_dict({'str': 0}, test_proto_pb2.Msg())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
