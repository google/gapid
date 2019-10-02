#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import os
import re
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components.ereporter2 import formatter
from test_support import test_case


# Access to a protected member XXX of a client class - pylint: disable=W0212


class Ereporter2FormatterTest(test_case.TestCase):
  def test_re_stack_trace(self):
    data = [
      (
        '  File "appengine/ext/ndb/tasklets.py", line 336, in wait_any',
        (
          '  File "',
          'appengine/ext/ndb/tasklets.py',
          '", line ',
          '336',
          ', in ',
          'wait_any',
        ),
      ),
      (
        '  File "appengine/api/memcache/__init__.py", line 955, in '
        '_set_multi_async_with_policy',
        (
          '  File "',
          'appengine/api/memcache/__init__.py',
          '", line ',
          '955',
          ', in ',
          '_set_multi_async_with_policy',
        ),
      ),
      (
        '  File "appengine/ext/ndb/eventloop.py", line 197',
        (
          '  File "',
          'appengine/ext/ndb/eventloop.py',
          '", line ',
          '197',
          None,
          None,
        ),
      ),
      (
        '  File "templates/restricted_bot.html", line 86, in block_body',
        (
          '  File "',
          'templates/restricted_bot.html',
          '", line ',
          '86',
          ', in ',
          'block_body',
        ),
      ),
    ]
    for line, expected in data:
      match = re.match(formatter.RE_STACK_TRACE_FILE, line)
      self.assertEqual(expected, match.groups())

  def test_relative_path(self):
    data = [
      os.getcwd(),
      os.path.dirname(os.path.dirname(os.path.dirname(
          formatter.runtime.__file__))),
      os.path.dirname(os.path.dirname(os.path.dirname(
          formatter.webapp2.__file__))),
      os.path.dirname(os.getcwd()),
      '.',
    ]
    for value in data:
      i = os.path.join(value, 'foo')
      self.assertEqual('foo', formatter._relative_path(i))

    self.assertEqual('bar/foo', formatter._relative_path('bar/foo'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
