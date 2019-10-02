# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Provides info about projects (service tenants)."""

import logging

from google.appengine.api import memcache
from google.appengine.ext import ndb
from google.appengine.ext.ndb import msgprop
from protorpc import messages

from components.config.proto import project_config_pb2
from components.config.proto import service_config_pb2

import common
import storage


DEFAULT_REF_CFG = project_config_pb2.RefsCfg(
    refs=[project_config_pb2.RefsCfg.Ref(name='refs/heads/master')])


class RepositoryType(messages.Enum):
  GITILES = 1


class ProjectImportInfo(ndb.Model):
  """Contains info how a project was imported.

  Entity key:
    Id is project id from the project registry. Has no parent.
  """
  created_ts = ndb.DateTimeProperty(auto_now_add=True)
  repo_type = msgprop.EnumProperty(RepositoryType, required=True)
  repo_url = ndb.StringProperty(required=True)


@ndb.transactional
def update_import_info(project_id, repo_type, repo_url):
  """Updates ProjectImportInfo if needed."""
  info = ProjectImportInfo.get_by_id(project_id)
  if info and info.repo_type == repo_type and info.repo_url == repo_url:
    return
  if info:
    values = (
      ('repo_url', repo_url, info.repo_url),
      ('repo_type', repo_type, info.repo_type),
    )
    logging.warning('Changing project %s repo info:\n%s',
        project_id,
        '\n'.join([
          '%s: %s -> %s' % (attr, old_value, new_value)
          for attr, old_value, new_value in values
          if old_value != new_value
        ]))
  ProjectImportInfo(id=project_id, repo_type=repo_type, repo_url=repo_url).put()


def get_projects():
  """Returns a list of projects stored in services/luci-config:projects.cfg.

  Never returns None. Cached.
  """
  cfg = storage.get_self_config_async(
      common.PROJECT_REGISTRY_FILENAME,
      service_config_pb2.ProjectsCfg).get_result()
  return cfg.projects or []


def get_project(id):
  """Returns a project by id."""
  for p in get_projects():
    if p.id == id:
      return p
  return None


@ndb.tasklet
def get_repos_async(project_ids):
  """Returns a mapping {project_id: (repo_type, repo_url)}.

  All projects must exist.
  """
  assert isinstance(project_ids, list)
  infos = yield ndb.get_multi_async(
      ndb.Key(ProjectImportInfo, pid) for pid in project_ids)
  raise ndb.Return({
    pid: (info.repo_type, info.repo_url) if info else (None, None)
    for pid, info in zip(project_ids, infos)
  })


@ndb.tasklet
def get_metadata_async(project_ids):
  """Returns a mapping {project_id: metadata}.

  If a project does not exist, the metadata is None.

  The project metadata stored in project.cfg files in each project.
  """
  PROJECT_DOES_NOT_EXIST_SENTINEL = (0,)
  cache_ns = 'projects.get_metadata'
  ctx = ndb.get_context()
  # ctx.memcache_get is auto-batching. Internally it makes get_multi RPC.
  cache_futs = {
    pid: ctx.memcache_get(pid, namespace=cache_ns)
    for pid in project_ids
  }
  yield cache_futs.values()
  result = {}
  missing = []
  for pid in project_ids:
    binary = cache_futs[pid].get_result()
    if binary is not None:
      # cache hit
      if binary == PROJECT_DOES_NOT_EXIST_SENTINEL:
        result[pid] = None
      else:
        cfg = project_config_pb2.ProjectCfg()
        cfg.ParseFromString(binary)
        result[pid] = cfg
    else:
      # cache miss
      missing.append(pid)

  if missing:
    fetched = yield _get_project_configs_async(
        missing, common.PROJECT_METADATA_FILENAME,
        project_config_pb2.ProjectCfg)
    result.update(fetched)  # at this point result must have all project ids
    # Cache metadata for 10 min. In practice, it never changes.
    # ctx.memcache_set is auto-batching. Internally it makes set_multi RPC.
    yield [
      ctx.memcache_set(
          pid,
          cfg.SerializeToString() if cfg else PROJECT_DOES_NOT_EXIST_SENTINEL,
          namespace=cache_ns,
          time=60 * 10)
      for pid, cfg in fetched.iteritems()
    ]

  raise ndb.Return(result)


def get_refs(project_ids):
  """Returns a mapping {project_id: list of refs}

  The ref list is None if a project does not exist.

  The list of refs stored in refs.cfg of a project.
  """
  cfgs = _get_project_configs_async(
      project_ids, common.REFS_FILENAME, project_config_pb2.RefsCfg
  ).get_result()
  return {
    pid: None if cfg is None else cfg.refs or DEFAULT_REF_CFG.refs
    for pid, cfg in cfgs.iteritems()
  }


def _get_project_configs_async(project_ids, path, message_factory):
  """Returns a mapping {project_id: message}.

  If a project does not exist, the message is None.
  """
  assert isinstance(project_ids, list)
  if not project_ids:
    empty = ndb.Future()
    empty.set_result({})
    return empty

  @ndb.tasklet
  def get_async():
    prefix = 'projects/'
    messages = yield storage.get_latest_messages_async(
        [prefix + pid for pid in _filter_existing(project_ids)],
        path, message_factory)
    raise ndb.Return({
      # messages may not have a key because we filter project ids by existence
      pid: messages.get(prefix + pid)
      for pid in project_ids
    })

  return get_async()


def _filter_existing(project_ids):
  # TODO(nodir): optimize
  assert isinstance(project_ids, list)
  if not project_ids:
    return project_ids
  assert all(pid for pid in project_ids)
  all_project_ids = set(p.id for p in get_projects())
  return [
    pid for pid in project_ids
    if pid in all_project_ids
  ]
