#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from test_support import test_case

from components import auth
from components.config import common
from components.config import endpoint
from components.config import validation


class EndpointTestCase(test_case.EndpointsTestCase):
  api_service_cls = endpoint.ConfigApi

  def test_metadata(self):
    rule_set = validation.RuleSet()
    self.mock(validation, 'DEFAULT_RULE_SET', rule_set)
    validation.rule('projects/foo', 'bar.cfg', rule_set=rule_set)
    validation.rule('services/foo', 'foo.cfg', rule_set=rule_set)

    self.mock(auth, 'is_admin', lambda: True)
    resp = self.call_api('get_metadata', {}).json_body
    self.assertEqual(
        resp,
        {
          'version': '1.0',
          'validation': {
            'url': 'https://localhost:80/_ah/api/config/v1/validate',
            'patterns': [
              {'config_set': 'projects/foo', 'path': 'bar.cfg'},
              {'config_set': 'services/foo', 'path': 'foo.cfg'},
            ],
          },
        }
    )

  def test_metadata_without_permissions(self):
    with self.call_should_fail(403):
      self.call_api('get_metadata', {})

  def test_config_service_is_trusted_requester(self):
    self.mock(auth, 'is_admin', lambda: False)
    config_identity = auth.Identity(
        'user', 'luci-config@appspot.gserviceaccount.com')
    self.mock(auth, 'get_current_identity', lambda: config_identity)
    self.assertFalse(endpoint.is_trusted_requester())

    common.ConfigSettings().modify(
        updated_by='unit-test',
        service_hostname='luci-config.appspot.com',
        trusted_config_account=auth.Identity(
            'user', 'luci-config@appspot.gserviceaccount.com'),
    )
    common.ConfigSettings.clear_cache()
    self.assertTrue(endpoint.is_trusted_requester())

  def test_bad_message_text_characters(self):
    rule_set = validation.RuleSet()
    self.mock(validation, 'DEFAULT_RULE_SET', rule_set)

    @validation.rule('projects/foo', 'bar.cfg', rule_set=rule_set)
    def validate(cfg, ctx):  # pylint: disable=unused-argument, unused-variable
      ctx.error('a\x80b')

    self.mock(auth, 'is_admin', lambda: True)

    req = {
      'config_set': 'projects/foo',
      'path': 'bar.cfg',
      'content': '',
    }
    resp = self.call_api('validate', req).json_body
    self.assertEqual(resp, {
      'messages': [
        {'path': 'bar.cfg', 'severity': 'ERROR', 'text': u'a\ufffdb'},
      ],
    })


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
