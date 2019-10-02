#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components.config import test_config_pb2
from test_support import test_case

import common


class CommonCase(test_case.TestCase):
  def test_convert_none(self):
    self.assertIsNone(common._convert_config(None, test_config_pb2.Config))

  def test_convert_empty(self):
    self.assertIsNotNone(common._convert_config('', test_config_pb2.Config))

  def test_convert_unicode(self):
    text = u'param: "\U0001f604"'.encode('utf-8')
    self.assertEqual(
        common._convert_config(text, test_config_pb2.Config),
        test_config_pb2.Config(param=u'\U0001f604'),
    )

  def test_trim_app_id(self):
    trimmed_app_id = 'gce-backend'
    app_id_external = trimmed_app_id
    app_id_internal = 'google.com:%s' % trimmed_app_id
    app_id_empty = ''
    self.assertEqual(common._trim_app_id(app_id_external), trimmed_app_id)
    self.assertEqual(common._trim_app_id(app_id_internal), trimmed_app_id)
    self.assertEqual(common._trim_app_id(app_id_empty), app_id_empty)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  else:
    logging.basicConfig(level=logging.CRITICAL)
  unittest.main()
