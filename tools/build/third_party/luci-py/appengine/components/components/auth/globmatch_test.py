#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components.auth import globmatch


class GlobMatchTest(unittest.TestCase):
  def test_translate(self):
    self.assertEqual('^$', globmatch._translate(''))
    self.assertEqual('^.*$', globmatch._translate('*'))
    self.assertEqual('^abc$', globmatch._translate('abc'))
    self.assertEqual('^a\\.bc$', globmatch._translate('a.bc'))

  def test_match(self):
    self.assertTrue(globmatch.match('', ''))
    self.assertTrue(globmatch.match('abc', '*'))
    self.assertTrue(globmatch.match('abc', 'abc'))
    self.assertFalse(globmatch.match('abcd', 'abc'))
    self.assertFalse(globmatch.match('zabc', 'abc'))
    self.assertFalse(globmatch.match('ABC', 'abc'))

    self.assertTrue(globmatch.match('abc@domain.com', '*@domain.com'))
    self.assertTrue(globmatch.match('@domain.com', '*@domain.com'))
    self.assertFalse(globmatch.match('abc@notdomain.com', '*@domain.com'))

    self.assertTrue(globmatch.match('p-abc@domain.com', 'p-*@domain.com'))
    self.assertFalse(globmatch.match('p-@anotherdomain.com', 'p-*@domain.com'))

    self.assertTrue(globmatch.match('p-abc', 'p-*'))
    self.assertFalse(globmatch.match('not-p-abc', 'p-*'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
