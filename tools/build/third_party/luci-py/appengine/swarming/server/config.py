# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Instance specific settings."""

import logging
import posixpath
import re

from components import auth
from components import config
from components import gitiles
from components import net
from components import utils
from components.config import validation

from proto.config import config_pb2

import cipd

NAMESPACE_RE = re.compile(r'^[a-z0-9A-Z\-._]+$')


ConfigApi = config.ConfigApi


### Public code.


# Maximum acceptable length for dimensions.
DIMENSION_KEY_LENGTH = 64
DIMENSION_VALUE_LENGTH = 256

# Regular expression for dimension key.
DIMENSION_KEY_RE = r'^[a-zA-Z\-\_\.]+$'


def settings_info():
  """Returns information about the settings file.

  Returns a dict with keys:
    'cfg': parsed SettingsCfg message
    'rev': revision of cfg
    'rev_url': URL of a human-consumable page that displays the config
    'config_service_url': URL of the config_service.
  """
  rev, cfg = _get_settings_with_defaults()
  rev_url = _gitiles_url(_get_configs_url(), rev, _SETTINGS_CFG_FILENAME)
  cfg_service_hostname = config.config_service_hostname()
  return {
    'cfg': cfg,
    'rev': rev,
    'rev_url': rev_url,
    'config_service_url': (
        'https://%s' % cfg_service_hostname if cfg_service_hostname else ''
    ),
  }


@utils.cache_with_expiration(60)
def settings():
  """Loads settings from an NDB-based cache or a default one if not present."""
  return _get_settings_with_defaults()[1]


def get_ui_client_id():
  """Returns OAuth client ID to use for web UI.

  Used as OAUTH_CLIENT_IDS_PROVIDER in appengien_config.py. Mocked in tests.
  """
  return settings().ui_client_id


def validate_flat_dimension(d):
  """Return strue if a 'key:value' dimension is valid."""
  key, _, val = d.partition(':')
  return validate_dimension_value(val) and validate_dimension_key(key)


def validate_dimension_key(key):
  """Returns True if the dimension key is valid."""
  return (
      isinstance(key, unicode) and
      key and
      len(key) <= DIMENSION_KEY_LENGTH and
      bool(re.match(DIMENSION_KEY_RE, key)))


def validate_dimension_value(value):
  """Returns True if the dimension key is valid."""
  return (
      bool(isinstance(value, unicode) and
      value and
      len(value) <= DIMENSION_VALUE_LENGTH and
      value.strip() == value))


### Private code.


_SETTINGS_CFG_FILENAME = 'settings.cfg'
_SECONDS_IN_YEAR = 60 * 60 * 24 * 365


def _validate_url(value, ctx):
  if not value:
    ctx.error('is not set')
  elif not validation.is_valid_secure_url(value):
    ctx.error('must start with "https://" or "http://localhost"')


def _validate_isolate_settings(cfg, ctx):
  if bool(cfg.default_server) != bool(cfg.default_namespace):
    ctx.error(
        'either specify both default_server and default_namespace or none')
  elif cfg.default_server:
    with ctx.prefix('default_server '):
      _validate_url(cfg.default_server, ctx)

    if not NAMESPACE_RE.match(cfg.default_namespace):
      ctx.error('invalid namespace "%s"', cfg.default_namespace)


def _validate_cipd_package(cfg, ctx):
  if not cipd.is_valid_package_name_template(cfg.package_name):
    ctx.error('invalid package_name "%s"', cfg.package_name)
  if not cipd.is_valid_version(cfg.version):
    ctx.error('invalid version "%s"', cfg.version)


def _validate_cipd_settings(cfg, ctx=None):
  """Validates CipdSettings message stored in settings.cfg."""
  ctx = ctx or validation.Context.raise_on_error()
  with ctx.prefix('default_server '):
    _validate_url(cfg.default_server, ctx)

  with ctx.prefix('default_client_package: '):
    _validate_cipd_package(cfg.default_client_package, ctx)


@validation.self_rule(_SETTINGS_CFG_FILENAME, config_pb2.SettingsCfg)
def _validate_settings(cfg, ctx):
  """Validates settings.cfg file against proto message schema."""
  def within_year(value):
    if value < 0:
      ctx.error('cannot be negative')
    elif value > _SECONDS_IN_YEAR:
      ctx.error('cannot be more than a year')

  with ctx.prefix('bot_death_timeout_secs '):
    within_year(cfg.bot_death_timeout_secs)
  with ctx.prefix('reusable_task_age_secs '):
    within_year(cfg.reusable_task_age_secs)

  if cfg.HasField('isolate'):
    with ctx.prefix('isolate: '):
      _validate_isolate_settings(cfg.isolate, ctx)

  if cfg.HasField('cipd'):
    with ctx.prefix('cipd: '):
      _validate_cipd_settings(cfg.cipd, ctx)

  if cfg.HasField('mp') and cfg.mp.server:
    with ctx.prefix('mp.server '):
      _validate_url(cfg.mp.server, ctx)

  with ctx.prefix('display_server_url_template '):
    url = cfg.display_server_url_template
    if url and not validation.is_valid_secure_url(url):
      ctx.error('URL %s must be https' % url)

  with ctx.prefix('extra_child_src_csp_url '):
    for url in cfg.extra_child_src_csp_url:
      if not validation.is_valid_secure_url(url):
        ctx.error('URL %s must be https' % url)


@utils.memcache('config:get_configs_url', time=60)
def _get_configs_url():
  """Returns URL where luci-config fetches configs from."""
  url = None
  try:
    url = config.get_config_set_location(config.self_config_set())
  except net.Error:
    logging.info(
        'Could not get configs URL. Possibly config directory for this '
        'instance of swarming does not exist')
  return url or 'about:blank'


def _gitiles_url(configs_url, rev, path):
  """URL to a directory in gitiles -> URL to a file at concrete revision."""
  try:
    loc = gitiles.Location.parse(configs_url)
    return str(loc._replace(
        treeish=rev or loc.treeish,
        path=posixpath.join(loc.path, path)))
  except ValueError:
    # Not a gitiles URL, return as is.
    return configs_url


def _get_settings():
  """Returns (rev, cfg) where cfg is a parsed SettingsCfg message.

  If config does not exists, returns (None, None).

  Mock this method in tests to inject changes to the defaults.
  """
  # store_last_good=True tells config component to update the config file
  # in a cron job. Here we just read from the datastore.
  return config.get_self_config(
      _SETTINGS_CFG_FILENAME, config_pb2.SettingsCfg, store_last_good=True)


def _get_settings_with_defaults():
  """Returns (rev, cfg) where cfg is a parsed SettingsCfg message.

  If config does not exists, returns (None, <cfg with defaults>).

  The config is cached in the datastore.
  """
  rev, cfg = _get_settings()
  cfg = cfg or config_pb2.SettingsCfg()
  cfg.reusable_task_age_secs = cfg.reusable_task_age_secs or 7*24*60*60
  cfg.bot_death_timeout_secs = cfg.bot_death_timeout_secs or 10*60

  cfg.auth.admins_group = cfg.auth.admins_group or 'administrators'
  cfg.auth.bot_bootstrap_group = cfg.auth.bot_bootstrap_group or \
     'administrators'
  cfg.auth.privileged_users_group = cfg.auth.privileged_users_group or \
     'administrators'
  cfg.auth.users_group = cfg.auth.users_group or 'administrators'

  return rev, cfg
