#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import mock

from components.config import validation_context
from test_support import test_case


class ValidationContextTestCase(test_case.TestCase):
  def test_logs_and_prefix(self):
    ctx = validation_context.Context()
    self.assertFalse(ctx.result().has_errors)

    ctx.error('hello %s', 'world')
    self.assertTrue(ctx.result().has_errors)

    with ctx.prefix('prefix %d ', 3):
      ctx.warning('warning %s', 2)

    with ctx.prefix('unicode %s', u'\xf0\x9f\x90\xb1 '):
      ctx.error('no cat')

    self.assertEqual(
      ctx.result(),
      validation_context.Result(
        messages=[
          validation_context.Message(
              severity=logging.ERROR, text='hello world'),
          validation_context.Message(
              severity=logging.WARNING, text='prefix 3 warning 2'),
          validation_context.Message(
              severity=logging.ERROR, text=u'unicode \xf0\x9f\x90\xb1 no cat'),
        ],
      ),
    )

  def test_raise_on_error(self):
    class Error(Exception):
      pass
    ctx = validation_context.Context.raise_on_error(exc_type=Error)
    with self.assertRaises(Error):
      ctx.error('1')

  def test_logging(self):
    logger = mock.Mock()
    ctx = validation_context.Context.logging(logger=logger)

    ctx.error('error')
    logger.log.assert_called_once_with(logging.ERROR, 'error')

    logger.log.reset_mock()
    ctx.warning('warning')
    logger.log.assert_called_once_with(logging.WARNING, 'warning')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
