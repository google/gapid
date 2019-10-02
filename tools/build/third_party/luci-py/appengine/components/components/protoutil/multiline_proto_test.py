#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
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

import multiline_proto
import test_proto_pb2


class MultilineProtoTest(unittest.TestCase):

  def test_pass_through(self):
    content = """num: 42
    str: "Churp churp churp."
    """
    self.assertEqual(
        content, multiline_proto.parse_multiline(content))

  def test_multiline(self):
    poem = """\
              `Twas brillig, and the slithy toves
              Did gyre and gimble in the wabe:
              All mimsy were the borogoves,
              And the mome raths outgrabe."""
    content = textwrap.dedent("""
      str: <<EOP
      %s
      EOP
      num: 42
    """) % poem
    msg = test_proto_pb2.Msg()
    parsed_content = multiline_proto.parse_multiline(content)
    protobuf.text_format.Merge(parsed_content, msg)
    self.assertEqual(textwrap.dedent(poem), msg.str)

  def test_missing_terminator(self):
    content = """
      str: <<EONEVER
        blah
        yoda
      """
    with self.assertRaises(multiline_proto.MultilineParseError):
      multiline_proto.parse_multiline(content)

  def test_escapes_and_unprintable(self):
    poem = u"""\
               `Twas brillig, and the slithy toves\n
               Did gyre and gimble in the wabe:\t
               All mimsy were the borogoves,\r
               And the mome raths outgrabe.\
               \U0001F4A9"""
    content = textwrap.dedent(u"""
      str: <<EOP
      %s
      EOP
      num: 42
    """) % poem
    msg = test_proto_pb2.Msg()
    parsed_content = multiline_proto.parse_multiline(content)
    protobuf.text_format.Merge(parsed_content, msg)
    self.assertEqual(textwrap.dedent(poem), msg.str)

  def test_go_compatibility(self):
    # pylint: disable=line-too-long
    """Replicate tests from luci-go to ensure identical results.

    See https://github.com/luci/luci-go/blob/master/common/proto/multiline_test.go
    """
    # Each test is a tuple ('name', 'expected', 'data').
    tests = [
        ('basic',
         r'something: "this\000 \t is\n\na \"basic\" test\\example"',
         '\n\t\t  '.join([
             'something: << EOF',
             'this\x00 \t is',
             '',
             r'a "basic" test\example',
             'EOF'])),
        ('contained', r'something: "A << B"', r'something: "A << B"'),
        ('indent',
         r'something: "this\n  is indented\n\nwith empty line"',
         '\n'.join([
             'something: << EOF',
             '\t\t\tthis',
             '\t\t\t  is indented',
             '   ',
             '\t\t\twith empty line',
             '\t\tEOF'])),
        ('col 0 align',
         r'something: "this\n  is indented\n\nwith empty line"',
         '\n'.join([
             'something: << EOF',
             'this',
             '  is indented',
             '   ',
             'with empty line',
             'EOF'])),
        ('nested', r'something: "<< nerp\nOther\nnerp\nfoo"',
         '\n\t\t'.join([
             'something: << EOF',
             '<< nerp',
             'Other',
             'nerp',
             'foo',
             'EOF'])),
        ('multi',
         '\n'.join(['something: "this is something"',
                    'else: "this is else"']),
         '\n'.join([
             'something: << EOF',
             'this is something',
             'EOF',
             'else: << ELSE',
             'this is else',
             'ELSE'])),
        ('indented first line',
         r'something: "  this line\nis indented\n  this too"',
         '\n\t '.join([
             'something: <<DERP',
             '  this line',
             'is indented',
             '  this too',
             'DERP'])),
         ('mixed indents are not indents',
         r'something: "\ttab\n  spaces"',
         '\n'.join([
             'something: <<DERP',
             '\ttab',
             '  spaces',
             'DERP'])),
    ]
    for test in tests:
      parsed = multiline_proto.parse_multiline(test[2])
      self.assertEqual(
          parsed, test[1], 'Test "%s" failed. Expected:\n%s\n\nActual:\n%s' % (
              test[0], test[1], parsed))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
