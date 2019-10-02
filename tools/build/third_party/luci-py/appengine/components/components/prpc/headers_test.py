#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import collections
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from test_support import test_case

from components.prpc import encoding
from components.prpc import headers


class PRPCHeadersTestCase(test_case.TestCase):
  def setUp(self):
    super(PRPCHeadersTestCase, self).setUp()

  def parse_headers(self, h, expect_content_type=None, expect_accept=None):
    res = headers.parse_headers(collections.OrderedDict(h))
    if expect_content_type is not None:
      self.assertEqual(res.content_type, expect_content_type)
    if expect_accept is not None:
      self.assertEqual(res.accept, expect_accept)
    return res

  def test_no_header(self):
    self.parse_headers([])

  def test_header_bad_content_type(self):
    with self.assertRaises(ValueError):
      self.parse_headers([
        ('Content-Type', 'www/urlencoded'),
      ])

  def test_parse_headers_encodings(self):
    check_known_encodings = lambda e: self.parse_headers(
        [('Accept', e[1])],
        expect_accept=e,
    )
    check_known_encodings(encoding.Encoding.JSON)
    check_known_encodings(encoding.Encoding.TEXT)
    check_known_encodings(encoding.Encoding.BINARY)
    self.parse_headers(
        [('Accept', 'application/json')],
        expect_accept=encoding.Encoding.JSON,
    )
    self.parse_headers(
        [('Accept', '*/*')],
        expect_accept=encoding.Encoding.BINARY,
    )


  def test_parse_headers_timeout(self):
    check_timeout = lambda t, n: self.assertEqual(
        self.parse_headers(
            [
              ('Accept', encoding.Encoding.JSON[1]),
              ('X-Prpc-Timeout', t),
            ],
            expect_accept=encoding.Encoding.JSON,
        ).timeout,
        n,
    )
    check_timeout('10H', 10*60*60)
    check_timeout('1124M', 1124*60)
    check_timeout('15S', 15)
    check_timeout('36m', 36*0.001)
    check_timeout('92u', 92*1e-6)
    check_timeout('56n', 56*1e-9)
    with self.assertRaises(ValueError):
      self.parse_headers([
        ('Accept', encoding.Encoding.JSON[1]),
        ('X-Prpc-Timeout', '222222'),
      ])

  def test_parse_headers_content_type(self):
    self.parse_headers(
        [
          ('Accept', encoding.Encoding.TEXT[1]),
          ('Content-Type', 'application/json'),
        ],
        expect_accept=encoding.Encoding.TEXT,
        expect_content_type=encoding.Encoding.JSON,
    )
    self.parse_headers(
        [
          ('Accept', encoding.Encoding.JSON[1]),
          ('Content_Type', encoding.Encoding.BINARY[1]),
        ],
        expect_accept=encoding.Encoding.JSON,
        expect_content_type=encoding.Encoding.BINARY,
    )

  def test_parse_headers_metadata(self):
    ctx = self.parse_headers(
        [
          ('Accept', encoding.Encoding.JSON[1]),
          ('X-Prpc-Timeout', '1m'),    # skipped from metadata
          ('X-Prpc-Future-Option', 'zzz'),  # same
          ('What-Bin', base64.b64encode('haha')),
          ('What', 'dup is ok'),
          ('Uhhhhh-Bin', base64.b64encode('lol')),
        ],
        expect_accept=encoding.Encoding.JSON,
    )
    self.assertEqual(ctx.invocation_metadata, [
        ('accept', encoding.Encoding.JSON[1]),
        ('what', 'haha'),
        ('what', 'dup is ok'),
        ('uhhhhh', 'lol'),
    ])
    with self.assertRaises(ValueError):
      self.parse_headers([
        ('Accept', encoding.Encoding.JSON[1]),
        ('What', 'haha'),
        ('What-Bin', 'lol'),
      ])

    with self.assertRaises(ValueError):
      self.parse_headers([
        ('Accept', encoding.Encoding.JSON[1]),
        ('What-Bin', 'asdfs='),
      ])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
