# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Contains functions to access configs.

Uses remote.Provider or fs.Provider to load configs, depending on whether config
service hostname is configured in common.ConfigSettings.
See _get_config_provider_async().

Provider do not do type conversion, api.py does.
"""

import collections
import logging
import sys

from google.appengine.ext import ndb

from components import auth

from . import common
from . import fs
from . import remote
from .proto import project_config_pb2


Project = collections.namedtuple('Project', [
  'id',  # Unique project id, defined in projects.cfg
  'repo_type',   # e.g. 'GITILES'
  'repo_url',   # e.g. 'https://chromium.googlesource.com/chromium/src'
  'name',  # e.g. 'Chromium browser'
])


@ndb.tasklet
def _get_config_provider_async():  # pragma: no cover
  """Returns a config provider to load configs.

  There are two config provider implementations: remote.Provider and
  fs.Provider. Both implement get_async(), get_project_configs_async(),
  get_ref_configs_async() with same signatures.
  """
  raise ndb.Return((yield remote.get_provider_async()) or fs.get_provider())


@ndb.tasklet
def get_async(
    config_set, path, dest_type=None, revision=None, store_last_good=False):
  """Reads a revision and contents of a config.

  If |store_last_good| is True (default is False), does not make remote calls,
  but consults datastore only, so the call is faster, less likely to fail and
  resilient to config service outages. A request for a certain config with
  store_last_good==True, instructs Cron job to start checking it periodically.
  If a config was not requested for a week, it is deleted from the datastore and
  not updated anymore. If the Cron job receives an invalid config, it is
  ignored, so get_async with store_last_good=True is guaranteed to always return
  valid configs as long as validation code is not changed in a non-backward
  compatible way. If a config was requested with store_last_good=True for the
  first time, (None, None) is returned.

  Args:
    config_set (str): config set to read a config from.
    path (str): path to the config file within the config set.
    dest_type (type): if specified, config content will be converted to
      |dest_type|. Only protobuf messages are supported.
    revision (str): a revision of the config set. Defaults to the latest
      revision.
    store_last_good (bool): if True, store configs in the datastore. Detaults
      to True if latest revision of self config is requested, otherwise False.
      See above for more details.

  Returns:
    Tuple (revision, config), where config is converted to |dest_type|.
    If not found, returns (None, None).

  Raises:
    NotImplementedError if |dest_type| is not supported.
    ValueError on invalid parameter.
    ConfigFormatError if config could not be converted to |dest_type|.
  """
  assert config_set
  assert path
  common._validate_dest_type(dest_type)

  if store_last_good:
    if revision:  # pragma: no cover
      raise ValueError(
          'store_last_good parameter cannot be set to True if revision is '
          'specified')

  provider = yield _get_config_provider_async()
  result = yield provider.get_async(
      config_set, path, revision=revision, dest_type=dest_type,
      store_last_good=store_last_good)
  raise ndb.Return(result)


def get(*args, **kwargs):
  """Blocking version of get_async."""
  return get_async(*args, **kwargs).get_result()


def get_self_config_async(*args, **kwargs):
  """A shorthand for get_async with config set for the current appid."""
  return get_async(common.self_config_set(), *args, **kwargs)


def get_self_config(*args, **kwargs):
  """Blocking version of get_self_config_async."""
  return get_self_config_async(*args, **kwargs).get_result()


def get_project_config_async(project_id, *args, **kwargs):
  """A shorthand for get_async for a project config set."""
  return get_async('projects/%s' % project_id, *args, **kwargs)


def get_project_config(*args, **kwargs):
  """Blocking version of get_project_config_async."""
  return get_project_config_async(*args, **kwargs).get_result()


def get_ref_config_async(project_id, ref, *args, **kwargs):
  """A shorthand for get_async for a project ref config set."""
  assert ref and ref.startswith('refs/'), ref
  return get_async('projects/%s/%s' % (project_id, ref), *args, **kwargs)


def get_ref_config(*args, **kwargs):
  """Blocking version of get_ref_config_async."""
  return get_ref_config_async(*args, **kwargs).get_result()


@ndb.tasklet
def get_projects_async():
  """Returns a list of registered projects (type Project)."""
  provider = yield _get_config_provider_async()
  project_dicts = yield provider.get_projects_async()
  empty = Project('', '', '', '')
  raise ndb.Return([empty._replace(**p) for p in project_dicts])


def get_projects():
  """Blocking version of get_projects_async."""
  return get_projects_async().get_result()


@ndb.tasklet
def get_project_configs_async(path, dest_type=None):
  """Returns configs at |path| in all projects.

  Args:
    path (str): path to configuration files. Files at this path will be
      retrieved from all projects.
    dest_type (type): if specified, config contents will be converted to
      |dest_type|. Only protobuf messages are supported. If a config could not
      be converted, the exception will be logged, but not raised.
      If |dest_type| is not specified, returned config is bytes.

  Returns:
    {project_id -> (revision, config, exception)} map.
    In file system mode, revision is None.
    If config is invalid, config is None and exception is not.
  """
  assert path
  common._validate_dest_type(dest_type)

  provider = yield _get_config_provider_async()
  configs = yield provider.get_project_configs_async(path)
  result = {}
  for config_set, (revision, content) in configs.iteritems():
    assert config_set and config_set.startswith('projects/'), config_set
    project_id = config_set[len('projects/'):]
    assert project_id
    try:
      config = common._convert_config(content, dest_type)
    except common.ConfigFormatError as ex:
      logging.exception(
          'Could not parse config at %s in config set %s: %r',
          path, config_set, content)
      result[project_id] = (revision, None, ex)
    else:
      result[project_id] = (revision, config, None)
  raise ndb.Return(result)


def get_project_configs(path, dest_type=None):
  """Blocking version of get_project_configs_async."""
  return get_project_configs_async(path, dest_type).get_result()


@ndb.tasklet
def get_ref_configs_async(path, dest_type=None):
  """Returns config at |path| in all refs of all projects.

  Args:
    path (str): path to configuration files. Files at this path will be
      retrieved from all refs of all projects.
    dest_type (type): if specified, config contents will be converted to
      |dest_type|. Only protobuf messages are supported. If a config could not
      be converted, the exception will be logged, but not raised.

  Returns:
    A map {project -> {ref -> (revision, config, exception)}}.
    Here ref is a str that always starts with 'ref/'.
    In file system mode, revision is None.
    If config is invalid, config is None and exception is not.
  """
  assert path
  common._validate_dest_type(dest_type)
  provider = yield _get_config_provider_async()
  configs = yield provider.get_ref_configs_async(path)
  result = {}
  for config_set, (revision, content) in configs.iteritems():
    assert config_set and config_set.startswith('projects/'), config_set
    project_id, ref = config_set.split('/', 2)[1:]
    assert project_id
    assert ref
    try:
      config = common._convert_config(content, dest_type)
    except common.ConfigFormatError as ex:
      logging.exception(
          'Could not parse config at %s in config set %s: %r',
          path, config_set, content)
      ref_value = (revision, None, ex)
    else:
      ref_value = (revision, config, None)
    result.setdefault(project_id, {})[ref] = ref_value
  raise ndb.Return(result)


def get_ref_configs(path, dest_type=None):
  """Blocking version of get_ref_configs_async."""
  return get_ref_configs_async(path, dest_type).get_result()


@ndb.tasklet
def get_config_set_location_async(config_set):  # pragma: no cover
  """Returns URL of where configs for a given config set are stored.

  It is a heavy call that always makes an RPC to the config service. Cache
  results appropriately.

  Args:
    config_set: name of the config set, e.g 'service/<id>' or 'projects/<id>'.

  Returns:
    URL or None if no such config set. In file system mode always None.
  """
  provider = yield _get_config_provider_async()
  location = yield provider.get_config_set_location_async(config_set)
  raise ndb.Return(location)


def get_config_set_location(config_set):  # pragma: no cover
  """Blocking version of get_config_set_location_async."""
  return get_config_set_location_async(config_set).get_result()


def _has_access(access, identity=None):
  identity = identity or auth.get_current_identity()
  if access.startswith('group:'):
    group = access.split(':', 2)[1]
    return auth.is_group_member(group, identity)

  ac_identity_str = access
  if ':' not in ac_identity_str:
    ac_identity_str = 'user:%s' % ac_identity_str
  return identity.to_bytes() == ac_identity_str


@ndb.tasklet
def has_project_access_async(project_id, identity=None):
  """Returns True if |identity| has access to project |project_id|.

  The ACL is defined in project.cfg in the project repo, using "access" field.
  See Project message in proto/service_config.proto for more details.

  This function does not do RPC to config service.
  """
  cfg = yield get_project_config_async(
      project_id, 'project.cfg', project_config_pb2.ProjectCfg,
      store_last_good=True)
  raise ndb.Return(cfg and any(_has_access(a, identity) for a in cfg.access))


def has_project_access(*args, **kwargs):
  """Blocking version of has_project_access_async."""
  return has_project_access_async(*args, **kwargs).get_result()
