#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

# google.protobuf package requires some python package magic hacking.
from components import utils
utils.fix_protobuf_package()

from google.protobuf import field_mask_pb2

import field_masks
import test_proto_pb2

# Shortcuts.
TestMsg = test_proto_pb2.Msg
TEST_DESC = TestMsg.DESCRIPTOR
STAR = field_masks.STAR
Mask = field_masks.Mask


class ParsePathTests(unittest.TestCase):
  def parse(self, path, **kwargs):
    return field_masks._parse_path(path, TEST_DESC, **kwargs)

  def test_str(self):
    actual = self.parse('str')
    expected = ('str',)
    self.assertEqual(actual, expected)

  def test_str_str(self):
    err_pattern = r'scalar field "str" cannot have subfields'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse('str.str')

  def test_str_invalid_char(self):
    err_pattern = r'unexpected token "@"; expected a period'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse('str@')

  def test_str_repeated(self):
    actual = self.parse('strs')
    expected = ('strs',)
    self.assertEqual(actual, expected)

  def test_str_repeated_trailing_star(self):
    actual = self.parse('strs.*')
    expected = ('strs', STAR)
    self.assertEqual(actual, expected)

  def test_str_repeated_index(self):
    err_pattern = r'unexpected token "1", expected a star'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse('strs.1')

  def test_map_num_key(self):
    actual = self.parse('map_num_str.1')
    expected = ('map_num_str', 1)
    self.assertEqual(actual, expected)

  def test_map_num_key_negative(self):
    actual = self.parse('map_num_str.-1')
    expected = ('map_num_str', -1)
    self.assertEqual(actual, expected)

  def test_map_num_key_invalid(self):
    with self.assertRaisesRegexp(ValueError, r'expected an integer'):
      self.parse('map_num_str.a')

  def test_map_num_key_invalid_with_correct_prefix(self):
    err_pattern = r'unexpected token "a"; expected a period'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse('map_num_str.1a')

  def test_map_str_key_unquoted(self):
    actual = self.parse('map_str_num.a')
    expected = ('map_str_num', 'a')
    self.assertEqual(actual, expected)

  def test_map_str_key_unquoted_longer(self):
    actual = self.parse('map_str_num.ab')
    expected = ('map_str_num', 'ab')
    self.assertEqual(actual, expected)

  def test_map_str_key_quoted(self):
    actual = self.parse('map_str_num.`a`')
    expected = ('map_str_num', 'a')
    self.assertEqual(actual, expected)

  def test_map_str_key_quoted_with_period(self):
    actual = self.parse('map_str_num.`a.b`')
    expected = ('map_str_num', 'a.b')
    self.assertEqual(actual, expected)

  def test_map_str_key_star(self):
    actual = self.parse('map_str_num.*')
    expected = ('map_str_num', STAR)
    self.assertEqual(actual, expected)

  def test_map_str_key_int(self):
    with self.assertRaisesRegexp(ValueError, r'expected a string'):
      self.parse('map_str_num.1')

  def test_map_bool_key_true(self):
    actual = self.parse('map_bool_str.true')
    expected = ('map_bool_str', True)
    self.assertEqual(actual, expected)

  def test_map_bool_key_false(self):
    actual = self.parse('map_bool_str.false')
    expected = ('map_bool_str', False)
    self.assertEqual(actual, expected)

  def test_map_bool_key_invalid(self):
    with self.assertRaisesRegexp(ValueError, r'expected true or false'):
      self.parse('map_bool_str.not-a-bool')

  def test_map_bool_key_star(self):
    actual = self.parse('map_bool_str.*')
    expected = ('map_bool_str', STAR)
    self.assertEqual(actual, expected)

  def test_msg_str(self):
    actual = self.parse('msg.str')
    expected = ('msg', 'str')
    self.assertEqual(actual, expected)

  def test_msg_star(self):
    actual = self.parse('msg.*')
    expected = ('msg', STAR)
    self.assertEqual(actual, expected)

  def test_msg_star_str(self):
    with self.assertRaisesRegexp(ValueError, r'expected end of string'):
      self.parse('msg.*.str')

  def test_msg_unexpected_field(self):
    err_pattern = r'field "x" does not exist in message protoutil.Msg'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse('msg.x')

  def test_msg_unexpected_subfield(self):
    err_pattern = r'field "x" does not exist in message protoutil.Msg'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse('msg.msg.x')

  def test_msg_msgs_str(self):
    actual = self.parse('msgs.*.str')
    expected = ('msgs', STAR, 'str')
    self.assertEqual(actual, expected)

  def test_msg_map_num_str(self):
    actual = self.parse('msg.map_num_str.1')
    expected = ('msg', 'map_num_str', 1)
    self.assertEqual(actual, expected)

  def test_map_str_map_num(self):
    actual = self.parse('map_str_msg.num')
    expected = ('map_str_msg', 'num')
    self.assertEqual(actual, expected)

  def test_map_str_map_num_star(self):
    actual = self.parse('map_str_msg.*')
    expected = ('map_str_msg', STAR)
    self.assertEqual(actual, expected)

  def test_map_str_map_num_star_str(self):
    actual = self.parse('map_str_msg.*.str')
    expected = ('map_str_msg', STAR, 'str')
    self.assertEqual(actual, expected)

  def test_trailing_period(self):
    with self.assertRaisesRegexp(ValueError, r'unexpected end'):
      self.parse('str.')

  def test_trailing_period_period(self):
    with self.assertRaisesRegexp(ValueError, r'cannot start with a period'):
      self.parse('str..')

  def test_repeated_valid(self):
    actual = self.parse('*.str', repeated=True)
    expected = (STAR, 'str')
    self.assertEqual(actual, expected)

  def test_repeated_invalid(self):
    with self.assertRaisesRegexp(ValueError, r'expected a star'):
      self.parse('1.str', repeated=True)

  def test_json_name(self):
    actual = self.parse('mapStrNum.aB', json_names=True)
    expected = ('map_str_num', 'aB')
    self.assertEqual(actual, expected)

  def test_json_name_option(self):
    actual = self.parse('another_json_name', json_names=True)
    expected = ('json_name_option',)
    self.assertEqual(actual, expected)


class NormalizePathsTests(unittest.TestCase):

  def test_empty(self):
    actual = field_masks._normalize_paths([])
    expected = set()
    self.assertEqual(actual, expected)

  def test_normal(self):
    actual = field_masks._normalize_paths([
        ('a',),
        ('b',),
    ])
    expected = {('a',), ('b',)}
    self.assertEqual(actual, expected)

  def test_redundancy_one_level(self):
    actual = field_masks._normalize_paths([
        ('a',),
        ('a', 'b'),
    ])
    expected = {('a',)}
    self.assertEqual(actual, expected)

  def test_redundancy_second_level(self):
    actual = field_masks._normalize_paths([
        ('a',),
        ('a', 'b', 'c'),
    ])
    expected = {('a',)}
    self.assertEqual(actual, expected)


class FromFieldMaskTests(unittest.TestCase):
  def parse(self, paths, **kwargs):
    fm = field_mask_pb2.FieldMask(paths=paths)
    return Mask.from_field_mask(fm, TEST_DESC, **kwargs)

  def test_empty(self):
    actual = self.parse([])
    expected = Mask(TEST_DESC)
    self.assertEqual(actual, expected)

  def test_str(self):
    actual = self.parse(['str'])
    expected = Mask(TEST_DESC, children={
        'str': Mask(),
    })
    self.assertEqual(actual, expected)

  def test_str_num(self):
    actual = self.parse(['str', 'num'])
    expected = Mask(TEST_DESC, children={
        'str': Mask(),
        'num': Mask(),
    })
    self.assertEqual(actual, expected)

  def test_str_msg_num(self):
    actual = self.parse(['str', 'msg.num'])
    expected = Mask(TEST_DESC, children={
        'str': Mask(),
        'msg': Mask(TEST_DESC, children={
            'num': Mask(),
        }),
    })
    self.assertEqual(actual, expected)

  def test_redunant(self):
    actual = self.parse(['msg', 'msg.num'])
    expected = Mask(TEST_DESC, children={
        'msg': Mask(TEST_DESC),
    })
    self.assertEqual(actual, expected)

  def test_redunant_star(self):
    actual = self.parse(['msg.*', 'msg.msg.num'])
    expected = Mask(TEST_DESC, children={
        'msg': Mask(TEST_DESC),
    })
    self.assertEqual(actual, expected)

  def test_json_names(self):
    actual = self.parse(['mapStrNum.aB', 'str'], json_names=True)
    map_str_num_desc = TEST_DESC.fields_by_name['map_str_num'].message_type
    expected = Mask(TEST_DESC, children={
        'str': Mask(),
        'map_str_num': Mask(map_str_num_desc, repeated=True, children={
            'aB': Mask(),
        }),
    })
    self.assertEqual(actual, expected)

  def test_update_mask(self):
    actual = self.parse(['msgs'], update_mask=True)
    expected = Mask(TEST_DESC, children={
        'msgs': Mask(TEST_DESC, repeated=True),
    })
    self.assertEqual(actual, expected)

  def test_update_mask_with_intermediate_repeated_field(self):
    err_pattern = r'field "msgs" in "msgs\.\*\.str" is not last'
    with self.assertRaisesRegexp(ValueError, err_pattern):
      self.parse(['msgs.*.str'], update_mask=True)


class IncludeTests(unittest.TestCase):
  def mask(self, *paths):
    return Mask.from_field_mask(
        field_mask_pb2.FieldMask(paths=list(paths)), TEST_DESC)

  def test_all(self):
    actual = self.mask().includes('str')
    self.assertEqual(actual, field_masks.INCLUDE_ENTIRELY)

  def test_include_recursively(self):
    actual = self.mask('str').includes('str')
    self.assertEqual(actual, field_masks.INCLUDE_ENTIRELY)

  def test_include_recursively_second_level(self):
    actual = self.mask('msg.str').includes('msg.str')
    self.assertEqual(actual, field_masks.INCLUDE_ENTIRELY)

  def test_include_recursively_star(self):
    actual = self.mask('map_str_msg.*.str').includes('map_str_msg.x.str')
    self.assertEqual(actual, field_masks.INCLUDE_ENTIRELY)

  def test_include_partially(self):
    actual = self.mask('msg.str').includes('msg')
    self.assertEqual(actual, field_masks.INCLUDE_PARTIALLY)

  def test_include_partially_second_level(self):
    actual = self.mask('msg.msg.str').includes('msg.msg')
    self.assertEqual(actual, field_masks.INCLUDE_PARTIALLY)

  def test_include_partially_star(self):
    actual = self.mask('map_str_msg.*.str').includes('map_str_msg.x')
    self.assertEqual(actual, field_masks.INCLUDE_PARTIALLY)

  def test_exclude(self):
    actual = self.mask('str').includes('num')
    self.assertEqual(actual, field_masks.EXCLUDE)

  def test_exclude_second_level(self):
    actual = self.mask('msg.str').includes('msg.num')
    self.assertEqual(actual, field_masks.EXCLUDE)


class TrimTests(unittest.TestCase):

  def trim(self, msg, *mask_paths):
    mask = Mask.from_field_mask(
        field_mask_pb2.FieldMask(paths=mask_paths), msg.DESCRIPTOR)
    mask.trim(msg)

  def test_scalar_trim(self):
    msg = TestMsg(num=1)
    self.trim(msg, 'str')
    self.assertEqual(msg, TestMsg())

  def test_scalar_leave(self):
    msg = TestMsg(num=1)
    self.trim(msg, 'num')
    self.assertEqual(msg, TestMsg(num=1))

  def test_scalar_repeated_trim(self):
    msg = TestMsg(nums=[1, 2])
    self.trim(msg, 'str')
    self.assertEqual(msg, TestMsg())

  def test_scalar_repeated_leave(self):
    msg = TestMsg(nums=[1, 2])
    self.trim(msg, 'nums')
    self.assertEqual(msg, TestMsg(nums=[1, 2]))

  def test_submessage_trim(self):
    msg = TestMsg(msg=TestMsg(num=1))
    self.trim(msg, 'str')
    self.assertEqual(msg, TestMsg())

  def test_submessage_leave_entirely(self):
    msg = TestMsg(msg=TestMsg(num=1))
    self.trim(msg, 'msg')
    self.assertEqual(msg, TestMsg(msg=TestMsg(num=1)))

  def test_submessage_leave_partially(self):
    msg = TestMsg(msg=TestMsg(num=1, str='x'))
    self.trim(msg, 'msg.num')
    self.assertEqual(msg, TestMsg(msg=TestMsg(num=1)))

  def test_submessage_repeated_trim(self):
    msg = TestMsg(
        msgs=[TestMsg(num=1), TestMsg(num=2)])
    self.trim(msg, 'str')
    self.assertEqual(msg, TestMsg())

  def test_submessage_repeated_leave_entirely(self):
    msg = TestMsg(
        msgs=[TestMsg(num=1), TestMsg(num=2)])
    expected = TestMsg(
        msgs=[TestMsg(num=1), TestMsg(num=2)])
    self.trim(msg, 'msgs')
    self.assertEqual(msg, expected)

  def test_submessage_repeated_leave_entirely_trailing_star(self):
    msg = TestMsg(
        msgs=[TestMsg(num=1), TestMsg(num=2)])
    expected = TestMsg(
        msgs=[TestMsg(num=1), TestMsg(num=2)])
    self.trim(msg, 'msgs.*')
    self.assertEqual(msg, expected)

  def test_submessage_repeated_leave_partially(self):
    msg = TestMsg(msgs=[
        TestMsg(num=1, str='x'),
        TestMsg(num=2, str='y'),
    ])
    expected = TestMsg(msgs=[
        TestMsg(num=1),
        TestMsg(num=2),
    ])
    self.trim(msg, 'msgs.*.num')
    self.assertEqual(msg, expected)

  def test_map_str_num_trim(self):
    msg = TestMsg(map_str_num={'1': 1, '2': 2})
    self.trim(msg, 'str')
    self.assertEqual(msg, TestMsg())

  def test_map_str_num_leave_key(self):
    msg = TestMsg(map_str_num={'a': 1, 'b': 2})
    self.trim(msg, 'map_str_num.a')
    self.assertEqual(msg, TestMsg(map_str_num={'a': 1}))

  def test_map_str_num_leave_key_with_int_key(self):
    msg = TestMsg(map_str_num={'1': 1, '2': 2})
    self.trim(msg, 'map_str_num.`1`')
    self.assertEqual(msg, TestMsg(map_str_num={'1': 1}))

  def test_map_str_num_leave_key_with_int_key_invalid(self):
    msg = TestMsg(map_str_num={'1': 1, '2': 2})
    with self.assertRaisesRegexp(ValueError, 'expected a string'):
      self.trim(msg, 'map_str_num.1')

  def test_map_str_msg_trim(self):
    msg = TestMsg(map_str_msg={'a': TestMsg()})
    self.trim(msg, 'str')
    self.assertEqual(msg, TestMsg())

  def test_map_str_msg_leave_key_entirely(self):
    msg = TestMsg(
        map_str_msg={
            'a': TestMsg(num=1),
            'b': TestMsg(num=2),
        },
        num=1)
    self.trim(msg, 'map_str_msg.a')
    self.assertEqual(
        msg,
        TestMsg(map_str_msg={'a': TestMsg(num=1)}))

  def test_map_str_msg_leave_key_partially(self):
    msg = TestMsg(
        map_str_msg={
            'a': TestMsg(num=1, str='a'),
            'b': TestMsg(num=2, str='b'),
        },
        num=1)
    self.trim(msg, 'map_str_msg.a.num')
    self.assertEqual(
        msg,
        TestMsg(map_str_msg={'a': TestMsg(num=1)}))


class MergeTests(unittest.TestCase):

  def mask(self, *paths):
    return Mask.from_field_mask(
        field_mask_pb2.FieldMask(paths=list(paths)),
        TEST_DESC,
        update_mask=True,
    )

  def test_scalar_field(self):
    src = TestMsg(num=1)
    dest = TestMsg(num=2)
    self.mask('num').merge(src, dest)
    self.assertEqual(dest.num, src.num)

  def test_repeated_scalar(self):
    src = TestMsg(nums=[1, 2])
    dest = TestMsg(nums=[3, 4])
    self.mask('nums').merge(src, dest)
    self.assertEqual(dest.nums, src.nums)

  def test_repeated_messages(self):
    src = TestMsg(msgs=[TestMsg(num=1), TestMsg(num=2)])
    dest = TestMsg(msgs=[TestMsg(num=3), TestMsg(num=4)])
    self.mask('msgs').merge(src, dest)
    self.assertEqual(dest.msgs, src.msgs)

  def test_entire_submessage(self):
    src = TestMsg(msg=TestMsg(num=1, str='a'))
    dest = TestMsg(msg=TestMsg(num=1, str='a'))
    self.mask('msg').merge(src, dest)
    self.assertEqual(dest.msg, src.msg)


  def test_unrelated_fields(self):
    src = TestMsg(num=1, str='a')
    dest = TestMsg(num=2, str='b')
    self.mask('num').merge(src, dest)
    self.assertEqual(dest.num, src.num)
    self.assertEqual(dest.str, 'b')

  def test_empty(self):
    src = TestMsg(num=1)
    dest = TestMsg(num=2)
    self.mask().merge(src, dest)
    self.assertEqual(dest.num, 2)

  def test_multiple(self):
    src = TestMsg(num=1, strs=['a', 'b'])
    dest = TestMsg(num=2, strs=['c', 'd'])
    self.mask('num', 'strs').merge(src, dest)
    self.assertEqual(dest.num, src.num)
    self.assertEqual(dest.strs, src.strs)


class SubmaskTests(unittest.TestCase):
  def mask(self, *paths):
    return Mask.from_field_mask(
        field_mask_pb2.FieldMask(paths=list(paths)), TEST_DESC)

  def test_one_level(self):
    actual = self.mask('msg.str', 'str').submask('msg')
    expected = self.mask('str')
    self.assertEqual(actual, expected)

  def test_two_levels(self):
    actual = self.mask('msg.msg.str', 'str').submask('msg.msg')
    expected = self.mask('str')
    self.assertEqual(actual, expected)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
