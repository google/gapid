# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines validation rules for configuration files.

Configurations requested with store_last_good=True are automatically validated
against rules defined using this module. Otherwise, validation.is_defined_for
and validation.validate can be used to validate configs in your code.

Example:

  # Configs foo.cfg in all project config sets must be valid protobuf messages.
  validation.project_config_rule('foo.cfg', config_pb2.FooCfg)
  validation.validate('projects/bar', 'foo.cfg', 'invalid content')
  # Will raise ValueError

Example with custom validation:

  @validation.project_config_rule('foo.cfg', config_pb2.FooCfg)
  def validate_foo(cfg, ctx):
    if cfg.bar < 0:
      ctx.error('bar must be non-negative: %d', cfg.bar)

  validation.validate('projects/baz', 'foo.cfg', 'bar: -1')
  # Will raise ValueError
"""

import collections
import fnmatch
import functools
import re
import urlparse

# Pylint doesn't like relative wildcard imports.
# pylint: disable=W0401,W0403

from . import common
from . import validation_context

__all__ = [
  'Context',
  'Message',
  'RuleSet',
  'is_defined_for',
  'is_valid',
  'project_config_rule',
  'ref_config_rule',
  'rule',
  'self_rule',
  'validate',
]


Message = validation_context.Message


class Context(validation_context.Context):
  """A validation context with config metadata information."""

  config_set = None
  path = None

  @property
  def service_id(self):
    if self.config_set:
      m = common.SERVICE_CONFIG_SET_RGX.match(self.config_set)
      if m:
        return m.group(1)
    return None

  @property
  def project_id(self):
    if self.config_set:
      m = common.PROJECT_CONFIG_SET_RGX.match(self.config_set)
      if m:
        return m.group(1)
      m = common.REF_CONFIG_SET_RGX.match(self.config_set)
      if m:
        return m.group(1)
    return None

  @property
  def ref(self):
    if self.config_set:
      m = common.REF_CONFIG_SET_RGX.match(self.config_set)
      if m:
        return m.group(2)
    return None


def is_valid_service_id(service_id):
  return bool(common.SERVICE_ID_RGX.match(service_id))


def is_valid_project_id(service_id):
  return bool(common.PROJECT_ID_RGX.match(service_id))


def is_valid_ref_name(ref):
  return bool(common.REF_NAME_RGX.match(ref))


def is_valid_secure_url(url):
  """Returns True if the URL is valid and secure, except for localhost."""
  parsed = urlparse.urlparse(url)
  if not parsed.netloc:
    return False
  if parsed.hostname in ('localhost', '127.0.0.1', '::1'):
    return parsed.scheme in ('http', 'https')
  return parsed.scheme == 'https'


ConfigPattern = collections.namedtuple(
    'ConfigPattern',
    [
      'config_set',  # config_set pattern, see compile_pattern().
      'path',  # path pattern, see compile_pattern().
    ])


def rule(config_set, path, dest_type=None, rule_set=None):
  """Creates a validation rule, that can act as a decorator.

  Just calling the function defines a validation rule that verifies that a
  config file is convertible to |dest_type|. Usage:

      validation.rule('projects/chromium', 'foo.cfg', myconfig_pb2.FooCfg)

  If the rule is used as decorator, the function being decorated is used for
  further validation. It should have the following parameters:
    * cfg: the converted config to be validated.
    * ctx (Context): used to report validation errors. It may have
      "config_set" and/or "path" attributes set.

  Usage:

      @validation.rule('projects/chromium', 'foo.cfg', myconfig_pb2.FooCfg)
      def validate_foo(cfg, ctx):
        if cfg.bar < 0:
          ctx.error('bar cannot be negative: %d', cfg.bar)

  |config_set| and |path| are patterns that determine if a rule is applicable
  to a config. Both |config_set| and |path| patterns must match. See
  compile_pattern's docstring for the definition of "pattern".

  Args:
    config_set (str): pattern for config set, see compile_pattern.
    path (str): pattern for path, see compile_pattern.
    dest_type (type): if specified, config contents will be converted to
      |dest_type| before calling the decorated function.
      Currently only protobuf messages are supported. If a config cannot be
      converted, it is considered invalid.
    rule_set (RuleSet): target rule set, defaults to the global
      DEFAULT_RULE_SET.

  Returns:
    A rule. Calling rule.remove() will remove the rule from the rule set.
  """
  rule_set = rule_set or DEFAULT_RULE_SET
  new_rule = Rule(config_set, path, dest_type)
  rule_set.add(new_rule)
  return new_rule


def project_config_rule(*args, **kwargs):
  """Shortcut for rule() for project configs."""
  return rule(
      'regex:%s' % common.PROJECT_CONFIG_SET_RGX.pattern,
      *args, **kwargs)


def ref_config_rule(*args, **kwargs):
  """Shortcut for rule() for ref configs."""
  return rule(
      'regex:%s' % common.REF_CONFIG_SET_RGX.pattern,
      *args, **kwargs)


def self_rule(*args, **kwargs):
  """Shortcut for rule() for current appid."""
  return rule(common.self_config_set(), *args, **kwargs)


def is_defined_for(config_set, path):
  """Returns True if validation is defined for given config_set and path."""
  return DEFAULT_RULE_SET.is_defined_for(config_set, path)


def validate(config_set, path, content, ctx=None):
  """Validates a config.

  If ctx is not specified, raises a ValueError on a first validation error.
  """
  DEFAULT_RULE_SET.validate(config_set, path, content, ctx=ctx)


def is_valid(config_set, path, content):
  """Returns True if the config is valid."""
  try:
    validate(config_set, path, content)
    return True
  except ValueError:
    return False


class Rule(object):
  """Validates a config if config_set and path match a predicate.

  See rule()'s docstring for more info.
  """

  rule_set = None

  def __init__(self, config_set, path, dest_type=None):
    self.config_set = config_set
    self.path = path
    self.config_set_fn = compile_pattern(config_set)
    self.path_fn = compile_pattern(path)
    common._validate_dest_type(dest_type)
    self.dest_type = dest_type
    self.validator_funcs = []

  def match(self, config_set, path):
    """Returns True if this rule is applicable to |config_set| and |path|."""
    return self.config_set_fn(config_set) and self.path_fn(path)

  def validate(self, config_set, path, content, ctx):
    try:
      cfg = common._convert_config(content, self.dest_type)
    except common.ConfigFormatError as ex:
      ctx.error('%s', ex)
      return
    ctx.config_set = config_set
    ctx.path = path
    for f in self.validator_funcs:
      f(cfg, ctx)

  def __call__(self, func):
    """Adds |func| to the list of validation functions. Used as a decorator."""
    assert func
    assert hasattr(func, '__call__')
    self.validator_funcs.append(func)
    func.rule = self
    return func

  def remove(self):
    """Removes this rule from its ruleset."""
    if self.rule_set:
      if self in self.rule_set.rules:
        self.rule_set.rules.remove(self)
      self.rule_set = None


class RuleSet(object):
  def __init__(self):
    self.rules = []

  def add(self, new_rule):
    assert isinstance(new_rule, Rule)
    new_rule.rule_set = self
    self.rules.append(new_rule)

  def validate(self, config_set, path, content, ctx=None):
    ctx = ctx or Context.raise_on_error()
    assert config_set
    assert path
    assert content is not None
    assert isinstance(ctx, Context)
    for r in self.rules:
      if r.match(config_set, path):
        r.validate(config_set, path, content, ctx)

  def is_defined_for(self, config_set, path):
    return any(r.match(config_set, path) for r in self.rules)

  def patterns(self):
    """Returns a set of all config patterns that this rule_set can validate.

    Returns:
      A set of ConfigPattern objects.
    """
    return set(
      ConfigPattern(config_set=r.config_set, path=r.path)
      for r in self.rules
    )


def compile_pattern(pattern):
  """Compiles a pattern to a predicate function.

  A pattern is a "<kind>:<value>" pair, where kind can be "text" (default) or
  "regex" and value interpretation depends on the kind:
    regex: value must be a regular expression. If it does not start/end with
      ^/$, they are added automatically.
    text: exact string.
  If colon is not present in the pattern, it is treated as "text:<pattern>".

  Returns:
    func (s: string): bool

  Raises:
    ValueError if |pattern| is malformed.
  """
  if not isinstance(pattern, basestring):
    raise ValueError('Pattern must be a string')
  if ':' in pattern:
    kind, value = pattern.split(':', 2)
  else:
    kind = 'text'
    value = pattern

  if kind in ('text', 'exact'):
    return lambda s: s == value

  if kind == 'regex':
    if not value.startswith('^'):
      value = '^' + value
    if not value.endswith('$'):
      value = value + '$'
    try:
      regex = re.compile(value)
    except re.error as ex:
      raise ValueError(ex.message)
    return regex.match

  raise ValueError('Invalid pattern kind: %s' % kind)


DEFAULT_RULE_SET = RuleSet()
