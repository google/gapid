# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging

from google.appengine.api import memcache
from google.appengine.ext import ndb
from protorpc import messages
from protorpc import message_types
from protorpc import remote
import endpoints

from components import auth
from components import utils
from components.config import endpoint as cfg_endpoint
from components.config import validation as cfg_validation
from components.config import common

import acl
import gitiles_import
import projects
import storage
import validation


# This is used by endpoints indirectly.
package = 'luci-config'


class Project(messages.Message):
  # Unique luci project id from services/luci-config:projects.cfg
  id = messages.StringField(1, required=True)
  # Project name from projects/<project_id>:project.cfg
  name = messages.StringField(2)
  repo_type = messages.EnumField(projects.RepositoryType, 3, required=True)
  repo_url = messages.StringField(4, required=True)


class Revision(messages.Message):
  id = messages.StringField(1)
  url = messages.StringField(2)
  timestamp = messages.IntegerField(3)
  committer_email = messages.StringField(4)

class File(messages.Message):
  """Describes a file."""
  path = messages.StringField(1)

class ConfigSet(messages.Message):
  """Describes a config set."""

  class ImportAttempt(messages.Message):
    timestamp = messages.IntegerField(1)
    revision = messages.MessageField(Revision, 2)
    success = messages.BooleanField(3)
    message = messages.StringField(4)
    validation_messages = messages.MessageField(
        cfg_endpoint.ValidationMessage, 5, repeated=True)

  config_set = messages.StringField(1, required=True)
  location = messages.StringField(2)
  revision = messages.MessageField(Revision, 3)
  last_import_attempt = messages.MessageField(ImportAttempt, 4)
  files = messages.MessageField(File, 5, repeated=True)

def attempt_to_msg(entity):
  if entity is None:
    return None
  return ConfigSet.ImportAttempt(
    timestamp=utils.datetime_to_timestamp(entity.time),
    revision=Revision(
        id=entity.revision.id,
        url=entity.revision.url,
        timestamp=utils.datetime_to_timestamp(entity.revision.time),
        committer_email=entity.revision.committer_email,
    ) if entity.revision else None,
    success=entity.success,
    message=entity.message,
    validation_messages=[
      cfg_endpoint.ValidationMessage(
          path=m.path, severity=m.severity, text=m.text)
      for m in entity.validation_messages
    ],
  )


GET_CONFIG_MULTI_REQUEST_RESOURCE_CONTAINER = endpoints.ResourceContainer(
    message_types.VoidMessage,
    path=messages.StringField(1, required=True),
    # If True, response.content will be None.
    hashes_only=messages.BooleanField(2, default=False),
)


class GetConfigMultiResponseMessage(messages.Message):
  class ConfigEntry(messages.Message):
    config_set = messages.StringField(1, required=True)
    revision = messages.StringField(2, required=True)
    content_hash = messages.StringField(3, required=True)
    # None if request.hash_only is True
    content = messages.BytesField(4)
    url = messages.StringField(5)
  configs = messages.MessageField(ConfigEntry, 1, repeated=True)


@auth.endpoints_api(name='config', version='v1', title='Configuration Service')
class ConfigApi(remote.Service):
  """API to access configurations."""

  ##############################################################################
  # endpoint: get_mapping

  class GetMappingResponseMessage(messages.Message):
    class Mapping(messages.Message):
      config_set = messages.StringField(1, required=True)
      location = messages.StringField(2)
    mappings = messages.MessageField(Mapping, 1, repeated=True)

  @auth.endpoints_method(
      endpoints.ResourceContainer(
          message_types.VoidMessage,
          config_set=messages.StringField(1),
      ),
      GetMappingResponseMessage,
      http_method='GET',
      path='mapping')
  @auth.public # ACL check inside
  def get_mapping(self, request):
    """DEPRECATED. Use get_config_sets."""
    if request.config_set and not can_read_config_set(request.config_set):
      raise endpoints.ForbiddenException()

    config_sets = storage.get_config_sets_async(
        config_set=request.config_set).get_result()
    can_read = can_read_config_sets([cs.key.id() for cs in config_sets])
    return self.GetMappingResponseMessage(
        mappings=[
          self.GetMappingResponseMessage.Mapping(
              config_set=cs.key.id(), location=cs.location)
          for cs in config_sets
          if can_read[cs.key.id()]
        ]
    )

  ##############################################################################
  # endpoint: validate_config

  class ValidateConfigResponseMessage(messages.Message):
    messages = messages.MessageField(
        cfg_endpoint.ValidationMessage, 1, repeated=True)

  class ValidateConfigRequestMessage(messages.Message):
    class File(messages.Message):
      path = messages.StringField(1)
      content = messages.BytesField(2)

    config_set = messages.StringField(1)
    files = messages.MessageField(File, 2, repeated=True)

  @auth.endpoints_method(
      ValidateConfigRequestMessage,
      ValidateConfigResponseMessage,
      http_method='POST',
      path='validate-config',
  )
  @auth.public # ACL check inside
  def validate_config(self, request):
    logging.debug(
        "requester: %s, config_set: %s, paths: %s",
        auth.get_current_identity().to_bytes(),
        request.config_set,
        [f.path for f in request.files])
    if not request.config_set:
      raise endpoints.BadRequestException('Must specify a config_set')
    if not request.files:
      raise endpoints.BadRequestException('Must specify files to validate')
    for f in request.files:
      if not f.path:
        raise endpoints.BadRequestException('Must specify the path of a file')
    if not acl.has_validation_access():
      logging.warning(
          '%s does not have validation access',
          auth.get_current_identity().to_bytes())
      raise endpoints.ForbiddenException()
    if not can_read_config_set(request.config_set):
      logging.warning(
          '%s does not have access to %s',
          auth.get_current_identity().to_bytes(),
          request.config_set)
      raise endpoints.ForbiddenException()

    futs = []
    for f in request.files:
      ctx = cfg_validation.Context()
      with ctx.prefix(f.path + ': '):
        futs.append(validation.validate_config_async(
            request.config_set, f.path, f.content, ctx=ctx))

    ndb.Future.wait_all(futs)
    # return the severities and the texts
    msgs = []
    for f, fut in zip(request.files, futs):
      for msg in fut.get_result().messages:
        msgs.append(cfg_endpoint.ValidationMessage(
          path=f.path,
          severity=common.Severity.lookup_by_number(msg.severity),
          text=msg.text))

    return self.ValidateConfigResponseMessage(messages=msgs)

  ##############################################################################
  # endpoint: get_config_sets

  class GetConfigSetsResponseMessage(messages.Message):
    config_sets = messages.MessageField(ConfigSet, 1, repeated=True)

  @auth.endpoints_method(
    endpoints.ResourceContainer(
        message_types.VoidMessage,
        config_set=messages.StringField(1),
        include_last_import_attempt=messages.BooleanField(2),
        include_files=messages.BooleanField(3),
    ),
    GetConfigSetsResponseMessage,
    http_method='GET',
    path='config-sets')
  @auth.public # ACL check inside
  def get_config_sets(self, request):
    """Returns config sets."""
    if request.config_set and not can_read_config_set(request.config_set):
      raise endpoints.ForbiddenException()
    if request.include_files and not request.config_set:
        raise endpoints.BadRequestException(
            'Must specify config_set to use include_files')

    config_sets = storage.get_config_sets_async(
        config_set=request.config_set).get_result()

    # The files property must always be a list of File objects (not None).
    files = []
    if request.include_files:
      # There must be a single config set because request.config_set is
      # specified.
      cs = config_sets[0]
      if cs.latest_revision:
        file_keys = storage.get_file_keys(
            request.config_set, cs.latest_revision)
        files = [File(path=key.id()) for key in file_keys]

    if request.include_last_import_attempt:
      attempts = ndb.get_multi([
        storage.last_import_attempt_key(cs.key.id()) for cs in config_sets
      ])
    else:
      attempts = [None] * len(config_sets)

    res = self.GetConfigSetsResponseMessage()
    can_read = can_read_config_sets([cs.key.id() for cs in config_sets])
    for cs, attempt in zip(config_sets, attempts):
      if not can_read[cs.key.id()]:
        continue

      if common.REF_CONFIG_SET_RGX.match(cs.key.id()):
        # Exclude ref configs from the listing for crbug.com/935667
        # TODO(crbug.com/924803): remove ref configs altogether.
        continue

      cs_msg = ConfigSet(
          config_set=cs.key.id(),
          location=cs.location,
          files=files,
          last_import_attempt=attempt_to_msg(attempt),
      )
      if cs.latest_revision:
        cs_msg.revision = Revision(
            id=cs.latest_revision,
            url=cs.latest_revision_url,
            committer_email=cs.latest_revision_committer_email,
        )
        if cs.latest_revision_time:
          cs_msg.revision.timestamp = utils.datetime_to_timestamp(
              cs.latest_revision_time)
      res.config_sets.append(cs_msg)
    return res

  ##############################################################################
  # endpoint: get_config

  class GetConfigResponseMessage(messages.Message):
    revision = messages.StringField(1, required=True)
    content_hash = messages.StringField(2, required=True)
    # If request.only_hash is not set to True, the contents of the
    # config file.
    content = messages.BytesField(3)
    # This field is only populated if the latest revision is requested.
    # TODO(jchinlee): populate in case of specific revision requested.
    url = messages.StringField(4)

  @auth.endpoints_method(
      endpoints.ResourceContainer(
          message_types.VoidMessage,
          config_set=messages.StringField(1, required=True),
          path=messages.StringField(2, required=True),
          revision=messages.StringField(3),
          hash_only=messages.BooleanField(4),
      ),
      GetConfigResponseMessage,
      http_method='GET',
      path='config_sets/{config_set}/config/{path}')
  @auth.public # ACL check inside
  def get_config(self, request):
    """Gets a config file."""
    try:
      validation.validate_config_set(request.config_set)
      validation.validate_path(request.path)
    except ValueError as ex:
      raise endpoints.BadRequestException(ex.message)
    res = self.GetConfigResponseMessage()

    if not can_read_config_set(request.config_set):
      logging.warning(
          '%s does not have access to %s',
          auth.get_current_identity().to_bytes(),
          request.config_set)
      raise_config_not_found()

    content_hashes = storage.get_config_hashes_async(
        {request.config_set: request.revision}, request.path).get_result()
    res.revision, res.url, res.content_hash = (
        content_hashes.get(request.config_set))
    if not res.content_hash:
      raise_config_not_found()

    if not request.hash_only:
      res.content = storage.get_configs_by_hashes_async(
          [res.content_hash]).get_result().get(res.content_hash)
      if not res.content:
        logging.warning(
            'Config hash is found, but the blob is not.\n'
            'File: "%s:%s:%s". Hash: %s', request.config_set,
            request.revision, request.path, res.content_hash)
        raise_config_not_found()

    return res

  ##############################################################################
  # endpoint: get_config_by_hash

  class GetConfigByHashResponseMessage(messages.Message):
    content = messages.BytesField(1, required=True)

  @auth.endpoints_method(
      endpoints.ResourceContainer(
          message_types.VoidMessage,
          content_hash=messages.StringField(1, required=True),
      ),
      GetConfigByHashResponseMessage,
      http_method='GET',
      path='config/{content_hash}')
  @auth.require(acl.can_get_by_hash)
  def get_config_by_hash(self, request):
    """Gets a config file by its hash."""
    res = self.GetConfigByHashResponseMessage(
        content=storage.get_configs_by_hashes_async(
            [request.content_hash]).get_result().get(request.content_hash)
    )
    if not res.content:
      raise_config_not_found()
    return res

  ##############################################################################
  # endpoint: get_projects

  class GetProjectsResponseMessage(messages.Message):
    projects = messages.MessageField(Project, 1, repeated=True)

  @auth.endpoints_method(
      message_types.VoidMessage,
      GetProjectsResponseMessage,
      http_method='GET',
      path='projects')
  @auth.public # ACL check inside
  def get_projects(self, request):  # pylint: disable=W0613
    """Gets list of registered projects.

    The project list is stored in services/luci-config:projects.cfg.
    """
    projs = get_projects()
    has_access = acl.has_projects_access([p.id for p in projs])
    return self.GetProjectsResponseMessage(
        projects=[p for p in projs if has_access[p.id]],
    )

  ##############################################################################
  # endpoint: get_refs

  class GetRefsResponseMessage(messages.Message):
    class Ref(messages.Message):
      name = messages.StringField(1)
    refs = messages.MessageField(Ref, 1, repeated=True)

  @auth.endpoints_method(
      endpoints.ResourceContainer(
          message_types.VoidMessage,
          project_id=messages.StringField(1, required=True),
      ),
      GetRefsResponseMessage,
      http_method='GET',
      path='projects/{project_id}/refs')
  @auth.public # ACL check inside
  def get_refs(self, request):
    """Gets list of refs of a project."""
    has_access = acl.has_projects_access(
        [request.project_id]).get(request.project_id)
    if not has_access:
      raise endpoints.NotFoundException()
    refs = projects.get_refs([request.project_id]).get(request.project_id)
    if refs is None:
      # Project not found
      raise endpoints.NotFoundException()
    res = self.GetRefsResponseMessage()
    res.refs = [res.Ref(name=ref.name) for ref in refs]
    return res

  ##############################################################################
  # endpoint: get_project_configs

  @auth.endpoints_method(
      GET_CONFIG_MULTI_REQUEST_RESOURCE_CONTAINER,
      GetConfigMultiResponseMessage,
      http_method='GET',
      path='configs/projects/{path}')
  @auth.public # ACL check inside
  def get_project_configs(self, request):
    """Gets configs in all project config sets."""
    try:
      validation.validate_path(request.path)
    except ValueError as ex:
      raise endpoints.BadRequestException(ex.message)

    return get_config_multi('projects', request.path, request.hashes_only)

  ##############################################################################
  # endpoint: get_ref_configs

  @auth.endpoints_method(
      GET_CONFIG_MULTI_REQUEST_RESOURCE_CONTAINER,
      GetConfigMultiResponseMessage,
      http_method='GET',
      path='configs/refs/{path}')
  @auth.public # ACL check inside
  def get_ref_configs(self, request):
    """Gets configs in all ref config sets."""
    try:
      validation.validate_path(request.path)
    except ValueError as ex:
      raise endpoints.BadRequestException(ex.message)

    return get_config_multi('refs', request.path, request.hashes_only)

  ##############################################################################
  # endpoint: reimport

  @auth.endpoints_method(
    endpoints.ResourceContainer(
        message_types.VoidMessage,
        config_set=messages.StringField(1, required=True)
    ),
    message_types.VoidMessage,
    http_method='POST',
    path='reimport')
  @auth.public # ACL check inside
  def reimport(self, request):
    """Reimports a config set."""
    try:
      validation.validate_config_set(request.config_set)
    except ValueError:
      raise endpoints.BadRequestException(
          'invalid config_set "%s"' % request.config_set)

    if not acl.can_reimport(request.config_set):
      raise endpoints.ForbiddenException(
          '%s is now allowed to reimport %r' % (
              auth.get_current_identity().to_bytes(), request.config_set))
    # Assume it is Gitiles.
    try:
      gitiles_import.import_config_set(request.config_set)
      return message_types.VoidMessage()
    except gitiles_import.NotFoundError as e:
      raise endpoints.NotFoundException(e.message)
    except ValueError as e:
      raise endpoints.BadRequestException(e.message)
    except gitiles_import.Error as e:
      raise endpoints.InternalServerErrorException(e.message)


@utils.memcache('projects_with_details', time=60)  # 1 min.
def get_projects():
  """Returns list of projects with metadata and repo info.

  Does not return projects that have no repo information. It might happen due
  to eventual consistency.

  Does not check access.

  Caches results in memcache for 1 min.
  """
  result = []
  projs = projects.get_projects()
  project_ids = [p.id for p in projs]
  repos_fut = projects.get_repos_async(project_ids)
  metadata_fut = projects.get_metadata_async(project_ids)
  ndb.Future.wait_all([repos_fut, metadata_fut])
  repos, metadata = repos_fut.get_result(), metadata_fut.get_result()
  for p in projs:
    repo_type, repo_url = repos.get(p.id, (None, None))
    if repo_type is None:
      # Not yet consistent.
      continue
    name = None
    if metadata.get(p.id) and metadata[p.id].name:
      name = metadata[p.id].name
    result.append(Project(
        id=p.id,
        name=name,
        repo_type=repo_type,
        repo_url=repo_url,
    ))
  return result


def get_config_sets_from_scope(scope):
  """Yields config sets from 'projects' or 'refs'."""
  assert scope in ('projects', 'refs'), scope
  projs = get_projects()
  refs = None
  if scope == 'refs':
    refs = projects.get_refs([p.id for p in projs])
  for p in projs:
    if scope == 'projects':
      yield 'projects/%s' % p.id
    else:
      for ref in refs[p.id] or ():
        yield 'projects/%s/%s' % (p.id, ref.name)


def get_config_multi(scope, path, hashes_only):
  """Returns configs at |path| in all config sets.

  scope can be 'projects' or 'refs'.

  Returns empty config list if requester does not have project access.
  """
  assert scope in ('projects', 'refs'), scope
  cache_key = (
    'v2/%s%s:%s' % (scope, ',hashes_only' if hashes_only else '', path))
  configs = memcache.get(cache_key)
  if configs is None:
    config_sets = list(get_config_sets_from_scope(scope))
    cfg_map = storage.get_latest_configs_async(
        config_sets, path, hashes_only=hashes_only).get_result()
    configs = []
    for cs in config_sets:
      rev, rev_url, content_hash, content = cfg_map.get(cs, (None, None, None))
      if not content_hash:
        continue
      configs.append({
        'config_set': cs,
        'revision': rev,
        'content_hash': content_hash,
        'content': content,
        'url': rev_url,
      })
      if not hashes_only and content is None:
        logging.error(
            'Blob %s referenced from %s:%s:%s was not found',
            content_hash, cs, rev, path)
    try:
      memcache.add(cache_key, configs, time=60)
    except ValueError:
      logging.exception('%s:%s configs are too big for memcache', scope, path)

  res = GetConfigMultiResponseMessage()
  can_read = can_read_config_sets([c['config_set'] for c in configs])
  for config in configs:
    if not can_read[config['config_set']]:
      continue
    if not hashes_only and config.get('content') is None:
      continue
    res.configs.append(res.ConfigEntry(
        config_set=config['config_set'],
        revision=config['revision'],
        content_hash=config['content_hash'],
        content=config.get('content'),
        url=config.get('url'),
    ))
  return res


def raise_config_not_found():
  raise endpoints.NotFoundException('The requested config is not found')


def can_read_config_sets(config_sets):
  try:
    return acl.can_read_config_sets(config_sets)
  except ValueError as ex:
    raise endpoints.BadRequestException(ex.message)


def can_read_config_set(config_set):
  return can_read_config_sets([config_set]).get(config_set)
