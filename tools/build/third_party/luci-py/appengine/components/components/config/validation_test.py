#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import re
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import mock
import yaml

from components.config import test_config_pb2
from components.config import validation
from test_support import test_case


class ValidationTestCase(test_case.TestCase):
  def setUp(self):
    super(ValidationTestCase, self).setUp()
    self.rule_set = validation.RuleSet()

  def test_rule(self):
    validating_func = mock.Mock()
    rule = validation.rule(
        'projects/foo', 'bar.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)
    rule(validating_func)

    self.rule_set.validate('projects/foo', 'bar.cfg', 'param: "valid config"')
    with self.assertRaises(ValueError):
      self.rule_set.validate('projects/foo', 'bar.cfg', 'wrong123')
    self.assertEqual(validating_func.call_count, 1)

    validating_func.side_effect = lambda cfg, ctx: ctx.error('bad')
    with self.assertRaises(ValueError):
      self.rule_set.validate('projects/foo', 'bar.cfg', 'param: "valid config"')

    # Different config_set/path
    self.rule_set.validate('projects/foo', 'baz.cfg', 'wrong')
    self.rule_set.validate('projects/baz', 'bar.cfg', 'wrong')

  def test_patterns(self):
    validation.rule(
        'projects/foo', 'bar.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)
    validation.rule(
        'services/foo', 'foo.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)
    validation.rule(
        'services/foo', 'foo.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)
    self.assertEqual(
      self.rule_set.patterns(),
      {
        validation.ConfigPattern('projects/foo', 'bar.cfg'),
        validation.ConfigPattern('services/foo', 'foo.cfg'),
        validation.ConfigPattern('services/foo', 'foo.cfg'),
      }
    )

  def test_context_metadata(self):
    ctx = validation.Context()

    ctx.config_set = 'services/foo'
    self.assertEqual(ctx.service_id, 'foo')
    self.assertEqual(ctx.project_id, None)
    self.assertEqual(ctx.ref, None)

    ctx.config_set = 'projects/foo'
    self.assertEqual(ctx.service_id, None)
    self.assertEqual(ctx.project_id, 'foo')
    self.assertEqual(ctx.ref, None)

    ctx.config_set = 'projects/foo/refs/a'
    self.assertEqual(ctx.service_id, None)
    self.assertEqual(ctx.project_id, 'foo')
    self.assertEqual(ctx.ref, 'refs/a')

  def test_regex_pattern_and_no_dest_type(self):
    rule = validation.rule(
        config_set='regex:projects/f[^/]+',
        path='regex:.+.yaml',
        rule_set=self.rule_set)
    def validate_yaml(cfg, ctx):
      try:
        yaml.safe_load(cfg)
      except Exception as ex:
        ctx.error('%s', ex)
    rule(validate_yaml)

    self.rule_set.validate('projects/foo', 'bar.cfg', '}{')
    self.rule_set.validate('projects/bar', 'bar.yaml', '}{')
    self.rule_set.validate('projects/foo', 'bar.yaml', '{}')

    with self.assertRaises(ValueError):
      self.rule_set.validate('projects/foo', 'bar.yaml', '}{')

  def test_project_config_rule(self):
    validation.project_config_rule(
        'bar.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)

    self.assertTrue(self.rule_set.is_defined_for('projects/foo', 'bar.cfg'))
    self.assertTrue(self.rule_set.is_defined_for('projects/baz', 'bar.cfg'))

    self.assertFalse(self.rule_set.is_defined_for('projects/x/huh', 'bar.cfg'))
    self.assertFalse(self.rule_set.is_defined_for('services/baz', 'bar.cfg'))
    self.assertFalse(self.rule_set.is_defined_for('projects/baz', 'notbar.cfg'))

  def test_ref_config_rule(self):
    validation.ref_config_rule(
        'bar.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)

    self.assertTrue(
        self.rule_set.is_defined_for(
            'projects/baz/refs/heads/master', 'bar.cfg'))

    self.assertFalse(
        self.rule_set.is_defined_for(
            'projects/baz/refs/heads/master', 'nonbar.cfg'))
    self.assertFalse(self.rule_set.is_defined_for('services/foo', 'bar.cfg'))
    self.assertFalse(self.rule_set.is_defined_for('projects/baz', 'bar.cfg'))

  def test_remove_rule(self):
    rule = validation.rule(
        'projects/foo', 'bar.cfg', test_config_pb2.Config,
        rule_set=self.rule_set)

    with self.assertRaises(ValueError):
      self.rule_set.validate('projects/foo', 'bar.cfg', 'invalid config')

    rule.remove()
    self.rule_set.validate('projects/foo', 'bar.cfg', 'invalid config')

  def test_compile_pattern(self):
    self.assertTrue(validation.compile_pattern('abc')('abc'))
    self.assertTrue(validation.compile_pattern('text:abc')('abc'))
    self.assertFalse(validation.compile_pattern('text:abc')('abcd'))

    self.assertTrue(validation.compile_pattern('regex:abc')('abc'))
    self.assertTrue(validation.compile_pattern('regex:\w+')('abc'))
    self.assertTrue(validation.compile_pattern('regex:^(\w+)c$')('abc'))
    self.assertFalse(validation.compile_pattern('regex:\d+')('a123b'))

  def test_is_valid_secure_url(self):
    true = [
      'http://localhost',
      'http://localhost/',
      'http://localhost/yo',
      'http://localhost:1',
      'http://localhost:1/yo',
      'https://localhost',
      'https://localhost/',
      'https://localhost/yo',
      'https://localhost:1',
      'https://localhost:1/yo',
      'https://user@bar.com',
      'https://user:pass@bar.com',
    ]
    for i in true:
      self.assertTrue(validation.is_valid_secure_url(i), i)
    false = [
      'http://',
      'http://#yo',
      'http://evil.com',
      'http://localhost:pwd@evil.com',
      'https://',
    ]
    for i in false:
      self.assertFalse(validation.is_valid_secure_url(i), i)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
