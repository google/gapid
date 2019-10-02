# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Setups jinja2 environment to be reused by all components and services."""

import datetime
import os
import urllib

from components import utils
utils.import_jinja2()

import jinja2

from google.appengine.api import app_identity
from google.appengine.api import users

from components import natsort


NON_BREAKING_HYPHEN = u'\u2011'


### Private stuff.


def _basename(filepath):
  return os.path.basename(filepath)


def _datetimeformat(value, f='%Y-%m-%d %H:%M:%S'):
  if not value:
    return NON_BREAKING_HYPHEN + NON_BREAKING_HYPHEN
  return value.strftime(f)


def _succinctdatetimeformat(value, f='%H:%M:%S'):
  """Similar to datetimeformat but skips the dates when it's today."""
  assert all(item not in f for item in ('%Y', '%m', '%d')), f
  if not value:
    return NON_BREAKING_HYPHEN + NON_BREAKING_HYPHEN
  if utils.utcnow().date() != value.date():
    # Prefix with the day.
    f = '%Y-%m-%d ' + f
  return value.strftime(f)


def _epochformat(value, f='%Y-%m-%d %H:%M:%S'):
  """Formats a float representing epoch to datetime."""
  if not value:
    return NON_BREAKING_HYPHEN + NON_BREAKING_HYPHEN
  return _datetimeformat(utils.timestamp_to_datetime(value * 1000000), f)


def _natsorted(value):
  """Accepts None transparently."""
  return natsort.natsorted(value or [])


def _timedeltaformat(value, include_ms=False):
  """Formats a timedelta in a sane way.

  Ignores sub-second precision by default.
  """
  if not value:
    return NON_BREAKING_HYPHEN + NON_BREAKING_HYPHEN
  total_seconds = value.total_seconds()
  suffix = ''
  if include_ms:
    ms = int(round(total_seconds-int(total_seconds), 3) * 1000)
    if ms:
      suffix = '.%03d' % ms
  hours, remainder = divmod(int(round(total_seconds)), 3600)
  minutes, seconds = divmod(remainder, 60)
  if hours:
    return '%d:%02d:%02d%s' % (hours, minutes, seconds, suffix)
  # Always prefix minutes, even if 0, otherwise this looks weird. Revisit this
  # decision if bikeshedding is desired.
  return '%d:%02d%s' % (minutes, seconds, suffix)


def _utf8(s):
  """Decodes an str to unicode."""
  if s is None:
    return s
  return s.decode('utf-8')


# Filters available by default.
_DEFAULT_GLOBAL_FILTERS = {
  'basename': _basename,
  'datetimeformat': _datetimeformat,
  'encode_to_json': utils.encode_to_json,
  'epochformat': _epochformat,
  'natsort': _natsorted,
  'succinctdatetimeformat': _succinctdatetimeformat,
  'timedeltaformat': _timedeltaformat,
  'utf8': _utf8,
}


# All the templates paths: prefix -> path.
_TEMPLATE_PATHS = {}
# Registered global variables.
_GLOBAL_ENV = {}
# Registered filters.
_GLOBAL_FILTERS = _DEFAULT_GLOBAL_FILTERS.copy()


### Public API.


def bootstrap(paths, global_env=None, filters=None):
  """Resets cached Jinja2 env to pick up new template paths.

  This is purely additive and idempotent. So consecutive calls to this functions
  with different arguments is fine.

  Args:
    paths: dict {prefix -> template_dir}, templates under template_dir would be
        accessible as <prefix>/<path relative to template_dir>.
    global_env: dict with variables to add to global template environment.
    filters: dict with filters to add to global filter list.
  """
  assert isinstance(paths, dict), paths
  assert all(
      _TEMPLATE_PATHS.get(k, v) == v for k, v in paths.iteritems()), paths
  assert all(os.path.isabs(p) for p in paths.itervalues()), paths
  assert all(os.path.isdir(p) for p in paths.itervalues()), paths

  if global_env is not None:
    assert isinstance(global_env, dict), global_env
    assert all(isinstance(k, str) for k in global_env), global_env
    assert all(
        _GLOBAL_ENV.get(k, v) == v
        for k, v in global_env.iteritems()), global_env

  if filters is not None:
    assert isinstance(filters, dict), filters
    assert all(
        isinstance(k, str) and callable(v)
        for k, v in filters.iteritems()), filters
    assert all(
        _GLOBAL_FILTERS.get(k, v) == v
        for k, v in filters.iteritems()), filters

  _TEMPLATE_PATHS.update(paths)

  if global_env:
    _GLOBAL_ENV.update(global_env)
  # These are immutable.
  _GLOBAL_ENV.setdefault('app_id', app_identity.get_application_id())
  _GLOBAL_ENV.setdefault('app_version', utils.get_app_version())
  _GLOBAL_ENV.setdefault('app_revision_url', utils.get_app_revision_url())

  if filters:
    _GLOBAL_FILTERS.update(filters)
  utils.clear_cache(get_jinja_env)


def reset():
  """To be used in tests only."""
  global _TEMPLATE_PATHS
  global _GLOBAL_ENV
  global _GLOBAL_FILTERS
  _TEMPLATE_PATHS = {}
  _GLOBAL_ENV = {}
  _GLOBAL_FILTERS = _DEFAULT_GLOBAL_FILTERS.copy()
  utils.clear_cache(get_jinja_env)


@utils.cache
def get_jinja_env():
  """Returns jinja2.Environment object that knows how to render templates."""
  # TODO(maruel): Add lstrip_blocks=True when jinja2 2.7 becomes available in
  # the GAE SDK.
  env = jinja2.Environment(
      loader=jinja2.PrefixLoader({
        prefix: jinja2.FileSystemLoader(path)
        for prefix, path in _TEMPLATE_PATHS.iteritems()
      }),
      autoescape=True,
      extensions=['jinja2.ext.autoescape'],
      trim_blocks=True,
      undefined=jinja2.StrictUndefined)
  env.filters.update(_GLOBAL_FILTERS)
  env.globals.update(_GLOBAL_ENV)
  return env


def _get_defaults():
  """Returns parameters used by most templates/base.html."""
  account = users.get_current_user()
  return {
    'nickname': account.email() if account else 'Sign in',
    'now': utils.utcnow(),
    # Hack to enable multi-login.
    'signin_link': users.create_login_url('/').replace(
        '/accounts/ServiceLogin', '/a/SelectSession', 1),
  }


def render(name, params=None):
  """Shorthand to render a template.

  It includes default useful parameters used in most templates/base.html.

  TODO(maruel): Uniformize this.
  """
  data = _get_defaults()
  data.update(params or {})
  return get_jinja_env().get_template(name).render(data)
