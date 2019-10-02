# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import json
import logging
import posixpath
import urlparse

from google.appengine.ext import ndb

from components import auth
from components import config
from components import gitiles
from components import net
from components.config import validation
from components.config.proto import project_config_pb2
from components.config.proto import service_config_pb2

import common
import services


def validate_config_set(config_set, ctx=None):
  ctx = ctx or validation.Context.raise_on_error()
  if not any(r.match(config_set) for r in config.ALL_CONFIG_SET_RGX):
    ctx.error('invalid config set: %s', config_set)


def validate_path(path, ctx=None):
  ctx = ctx or validation.Context.raise_on_error(prefix='Invalid path: ')
  if not path:
    ctx.error('not specified')
    return
  if posixpath.isabs(path):
    ctx.error('must not be absolute: %s', path)
  if any(p in ('.', '..') for p in path.split(posixpath.sep)):
    ctx.error('must not contain ".." or "." components: %s', path)


def validate_url(url, ctx):
  if not url:
    ctx.error('not specified')
    return
  parsed = urlparse.urlparse(url)
  if not parsed.netloc:
    ctx.error('hostname not specified')
  if parsed.scheme != 'https':
    ctx.error('scheme must be "https"')


def validate_pattern(pattern, literal_validator, ctx):
  try:
    config.validation.compile_pattern(pattern)
  except ValueError as ex:
    ctx.error('%s', ex.message)
    return

  if ':' not in pattern:
    literal_validator(pattern, ctx)
  elif pattern.startswith('text:'):
    literal_validator(pattern.split(':', 2)[1], ctx)


def check_id_sorted(iterable, list_name, ctx):
  """Emits a warning if the iterable is not sorted by id."""
  prev = None
  for item in iterable:
    if not item.id:
      continue
    if prev is not None and item.id < prev:
      ctx.warning(
          '%s are not sorted by id. First offending id: %s', list_name, item.id)
      return
    prev = item.id


def validate_id(id, rgx, known_ids, ctx):
  if not id:
    ctx.error('id is not specified')
    return
  if not rgx.match(id):
    ctx.error('id "%s" does not match %s regex', id, rgx.pattern)
    return
  if id in known_ids:
    ctx.error('id is not unique')
  else:
    known_ids.add(id)


def validate_config_set_location(loc, ctx, allow_relative_url=False):
  if not loc:
    ctx.error('not specified')
    return
  if is_url_relative(loc.url):
    if not allow_relative_url:
      ctx.error('url is relative')
    elif loc.storage_type != service_config_pb2.ConfigSetLocation.UNSET:
      ctx.error('storage_type must not be set if relative url is used')
  elif loc.storage_type == service_config_pb2.ConfigSetLocation.UNSET:
    ctx.error('storage_type is not set')
  else:
    assert loc.storage_type == service_config_pb2.ConfigSetLocation.GITILES
    try:
      parsed_loc = gitiles.Location.parse(loc.url)
    except ValueError as ex:
      ctx.error('%s', ex.message)
    else:
      if parsed_loc.treeish == 'HEAD':
        ctx.error('ref/commit is not specified')


@validation.self_rule(
    common.PROJECT_REGISTRY_FILENAME, service_config_pb2.ProjectsCfg)
def validate_project_registry(cfg, ctx):
  project_ids = set()
  for i, project in enumerate(cfg.projects):
    with ctx.prefix('Project %s: ', project.id or ('#%d' % (i + 1))):
      validate_id(project.id, config.common.PROJECT_ID_RGX, project_ids, ctx)
      with ctx.prefix('config_location: '):
        validate_config_set_location(project.config_location, ctx)
  check_id_sorted(cfg.projects, 'Projects', ctx)



def validate_identity(identity, ctx):
  try:
    auth.Identity.from_bytes(identity)
  except ValueError as ex:
    ctx.error('%s', ex.message)


def validate_email(email, ctx):
  try:
    auth.Identity('user', email)
  except ValueError:
    ctx.error('invalid email: "%s"', email)


def validate_group(group, ctx):
  if not auth.is_valid_group_name(group):
    ctx.error('invalid group: %s', group)


def validate_identity_predicate(access, ctx):
  """Ensures |access| is "group:<group>", an identity or an email."""
  if not access:
    ctx.error('not specified')
    return
  elif access.startswith('group:'):
    group = access.split(':', 2)[1]
    validate_group(group, ctx)
  elif ':' in access:
    validate_identity(access, ctx)
  else:
    validate_email(access, ctx)


def validate_access_list(access_list, ctx):
  for i, ac in enumerate(access_list):
    with ctx.prefix('access #%d: ', i + 1):
      validate_identity_predicate(ac, ctx)


@validation.self_rule(
    common.SERVICES_REGISTRY_FILENAME, service_config_pb2.ServicesCfg)
def validate_services_cfg(cfg, ctx):
  service_ids = set()
  for i, service in enumerate(cfg.services):
    with ctx.prefix('Service %s: ', service.id or ('#%d' % (i + 1))):
      validate_id(service.id, config.common.SERVICE_ID_RGX, service_ids, ctx)
      if service.config_location and service.config_location.url:
        with ctx.prefix('config_location: '):
          validate_config_set_location(
              service.config_location, ctx, allow_relative_url=True)
      for owner in service.owners:
        validate_email(owner, ctx)
      if service.metadata_url:
        with ctx.prefix('metadata_url: '):
          validate_url(service.metadata_url, ctx)
      validate_access_list(service.access, ctx)

  check_id_sorted(cfg.services, 'Services', ctx)


def validate_service_dynamic_metadata_blob(metadata, ctx):
  """Validates JSON-encoded ServiceDynamicMetadata"""
  if not isinstance(metadata, dict):
    ctx.error('Service dynamic metadata must be an object')
    return

  if metadata.get('version') != '1.0':
    ctx.error(
        'Expected format version 1.0, but found "%s"', metadata.get('version'))

  validation = metadata.get('validation')
  if validation is None:
    return

  with ctx.prefix('validation: '):
    if not isinstance(validation, dict):
      ctx.error('must be an object')
      return
    with ctx.prefix('url: '):
      validate_url(validation.get('url'), ctx)
    patterns = validation.get('patterns', [])
    if not isinstance(patterns, list):
      ctx.error('patterns must be a list')
      return
    for i, p in enumerate(patterns):
      with ctx.prefix('pattern #%d: ', i + 1):
        if not isinstance(p, dict):
          ctx.error('must be an object')
          continue
        with ctx.prefix('config_set: '):
          validate_pattern(p.get('config_set'), validate_config_set, ctx)
        with ctx.prefix('path: '):
          validate_pattern(p.get('path'), validate_path, ctx)


@validation.self_rule(common.ACL_FILENAME, service_config_pb2.AclCfg)
def validate_acl_cfg(cfg, ctx):
  if cfg.project_access_group:
    validate_group(cfg.project_access_group, ctx)


@validation.self_rule(common.IMPORT_FILENAME, service_config_pb2.ImportCfg)
def validate_import_cfg(_cfg, _ctx):
  # A valid protobuf message is enough.
  pass


@validation.self_rule(common.SCHEMAS_FILENAME, service_config_pb2.SchemasCfg)
def validate_schemas(cfg, ctx):
  names = set()
  for i, schema in enumerate(cfg.schemas):
    with ctx.prefix('Schema %s: ', schema.name or '#%d' % (i + 1)):
      if not schema.name:
        ctx.error('name is not specified')
      elif ':' not in schema.name:
        ctx.error('name must contain ":"')
      else:
        if schema.name in names:
          ctx.error('duplicate schema name')
        else:
          names.add(schema.name)

        config_set, path = schema.name.split(':', 2)
        if (not config.SERVICE_CONFIG_SET_RGX.match(config_set) and
            config_set not in ('projects', 'projects/refs')):
          ctx.error(
              'left side of ":" must be a service config set, "projects" or '
              '"projects/refs"')
        validate_path(path, ctx)
      with ctx.prefix('url: '):
        validate_url(schema.url, ctx)


@validation.project_config_rule(
    common.PROJECT_METADATA_FILENAME, project_config_pb2.ProjectCfg)
def validate_project_metadata(cfg, ctx):
  if not cfg.name:
    ctx.error('name is not specified')
  validate_access_list(cfg.access, ctx)


@validation.project_config_rule(
    common.REFS_FILENAME, project_config_pb2.RefsCfg)
def validate_refs_cfg(cfg, ctx):
  refs = set()
  for i, ref in enumerate(cfg.refs):
    with ctx.prefix('Ref #%d: ', i + 1):
      if not ref.name:
        ctx.error('name is not specified')
      elif not ref.name.startswith('refs/'):
        ctx.error('name does not start with "refs/": %s', ref.name)
      elif ref.name in refs:
        ctx.error('duplicate ref: %s', ref.name)
      else:
        refs.add(ref.name)
      if ref.config_path:
        validate_path(ref.config_path, ctx)


@ndb.tasklet
def _validate_by_service_async(service, config_set, path, content, ctx):
  """Validates a config with an external service.

  Validation results will be stored in the validation context.

  Args:
    service (service_config_pb2.Service): service to be validated against.
    config_set (str): config set being validated.
    path (str): path of the config file being validated.
    content (str): byte-form of the content of the file being validated.
    ctx (validation.Context): context in which validation messages
      will be stored.
  """
  try:
    metadata = yield services.get_metadata_async(service.id)
  except services.DynamicMetadataError as ex:
    logging.error('Could not load dynamic metadata for %s: %s', service.id, ex)
    return

  assert metadata and metadata.validation
  url = metadata.validation.url
  if not url:
    return

  match = False
  for p in metadata.validation.patterns:
    # TODO(nodir): optimize if necessary.
    if (validation.compile_pattern(p.config_set)(config_set) and
        validation.compile_pattern(p.path)(path)):
      match = True
      break
  if not match:
    return

  res = None

  def report_error(text):
    text = (
        'Error during external validation: %s\n'
        'url: %s\n'
        'config_set: %s\n'
        'path: %s\n'
        'response: %r') % (text, url, config_set, path, res)
    logging.error(text)
    ctx.critical('%s', text)

  try:
    req = {
      'config_set': config_set,
      'path': path,
      'content': base64.b64encode(content),
    }
    res = yield net.json_request_async(
        url, method='POST', payload=req, scopes=net.EMAIL_SCOPE)
  except net.Error as ex:
    report_error('Net error: %s' % ex)
    return

  try:
    for msg in res.get('messages', []):
      if not isinstance(msg, dict):
        report_error('invalid response: message is not a dict: %r' % msg)
        continue
      severity = msg.get('severity') or 'INFO'
      # validation library for Go services sends severity as an integer
      # corresponding to Python's logging severity level.
      if severity in (logging.DEBUG, logging.INFO, logging.WARNING,
                      logging.ERROR, logging.CRITICAL):
        severity = logging.getLevelName(severity)
      if (severity not in
          service_config_pb2.ValidationResponseMessage.Severity.keys()):
        report_error(
            'invalid response: unexpected message severity: %r' % severity)
        continue
      # It is safe because we've validated |severity|.
      func = getattr(ctx, severity.lower())
      func('%s', msg.get('text') or '')
  except Exception as ex:
    report_error(ex)


@ndb.tasklet
def validate_config_async(config_set, path, content, ctx=None):
  """Validates a config against built-in and external validators.

  External validators are defined in validation.cfg,
  see proto/service_config.proto.

  Returns:
    components.config.validation_context.Result.
  """
  ctx = ctx or validation.Context()

  # Check the config against built-in validators,
  # defined using validation.self_rule.
  validation.validate(config_set, path, content, ctx=ctx)

  all_services = yield services.get_services_async()
  futures = []
  for service in all_services:
    futures.append(
        _validate_by_service_async(service, config_set, path, content, ctx))
  yield futures
  raise ndb.Return(ctx.result())


def validate_config(*args, **kwargs):
  """Blocking version of validate_async."""
  return validate_config_async(*args, **kwargs).get_result()


def is_url_relative(url):
  parsed = urlparse.urlparse(url)
  return bool(not parsed.scheme and not parsed.netloc and parsed.path)


@validation.rule('regex:.+', 'regex:.+\.json')
def validate_json_files(cfg, ctx):
  try:
    json.loads(cfg)
  except ValueError as ex:
    ctx.error('Invalid JSON file: %s', ex)
