# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Imports config files stored in Gitiles.

If services_config_location is set in admin.GlobalConfig root entity,
each directory in the location is imported as services/<directory_name>.

For each project defined in the project registry with
config_storage_type == Gitiles, projects/<project_id> config set is imported
from project.config_location.
"""

import contextlib
import json
import logging
import os
import random
import re
import StringIO
import tarfile

from google.appengine.api import memcache
from google.appengine.api import taskqueue
from google.appengine.api import urlfetch_errors
from google.appengine.ext import ndb
from google.protobuf import text_format

from components import config
from components import gitiles
from components import net
from components.config.proto import service_config_pb2

import admin
import common
import notifications
import projects
import storage
import validation


GITILES_STORAGE_TYPE = admin.ServiceConfigStorageType.GITILES
GITILES_LOCATION_TYPE = service_config_pb2.ConfigSetLocation.GITILES
DEFAULT_GITILES_IMPORT_CONFIG = service_config_pb2.ImportCfg.Gitiles(
    fetch_log_deadline=15,
    fetch_archive_deadline=15,
    ref_config_default_path='luci',
)


class Error(Exception):
  """A config set import-specific error."""


class NotFoundError(Error):
  """A service, project or ref is not found."""


class HistoryDisappeared(Error):
  """Gitiles history unexpectedly disappeared."""


def _commit_to_revision_info(commit, location):
  if commit is None:
    return None
  url = ''
  if location:
    url = str(location._replace(treeish=commit.sha))
  return storage.RevisionInfo(
      id=commit.sha,
      url=url,
      committer_email=commit.committer.email,
      time=commit.committer.time,
  )


def get_gitiles_config():
  cfg = service_config_pb2.ImportCfg(gitiles=DEFAULT_GITILES_IMPORT_CONFIG)
  try:
    cfg = storage.get_self_config_async(
        common.IMPORT_FILENAME, lambda: cfg).get_result()
  except text_format.ParseError as ex:
    # It is critical that get_gitiles_config() returns a valid config.
    # If import.cfg is broken, it should not break importing mechanism,
    # otherwise the system won't be able to heal itself by importing a fixed
    # config.
    logging.exception('import.cfg is broken')
  return cfg.gitiles


## Low level import functions


def _resolved_location(url):
  """Gitiles URL string -> gitiles.Location.parse_resolve(url).

  Does caching internally for X sec (X in [30m, 1h30m]) to avoid hitting Gitiles
  all the time for data that is almost certainly static.
  """
  cache_key = 'gitiles_location:v1:' + url
  as_dict = memcache.get(cache_key)
  if as_dict is not None:
    return gitiles.Location.from_dict(as_dict)
  logging.debug('Cache miss when resolving gitiles location %s', url)
  loc = gitiles.Location.parse_resolve(url)
  memcache.set(cache_key, loc.to_dict(), time=random.randint(1800, 5400))
  return loc


def _import_revision(config_set, base_location, commit, force_update):
  """Imports a referenced Gitiles revision into a config set.

  |base_location| will be used to set storage.ConfigSet.location.

  Updates last ImportAttempt for the config set.

  Puts ConfigSet initialized from arguments.
  """
  revision = commit.sha
  assert re.match('[0-9a-f]{40}', revision), (
      '"%s" is not a valid sha' % revision
  )
  rev_key = ndb.Key(
      storage.ConfigSet, config_set,
      storage.Revision, revision)

  location = base_location._replace(treeish=revision)
  attempt = storage.ImportAttempt(
      key=storage.last_import_attempt_key(config_set),
      revision=_commit_to_revision_info(commit, location))

  cs_entity = storage.ConfigSet(
      id=config_set,
      latest_revision=revision,
      latest_revision_url=str(location),
      latest_revision_committer_email=commit.committer.email,
      latest_revision_time=commit.committer.time,
      location=str(base_location),
      version=storage.ConfigSet.CUR_VERSION,
  )

  if not force_update and rev_key.get():
    attempt.success = True
    attempt.message = 'Up-to-date'
    ndb.put_multi([cs_entity, attempt])
    return

  rev_entities = [cs_entity, storage.Revision(key=rev_key)]

  # Fetch archive outside ConfigSet transaction.
  archive = location.get_archive(
      deadline=get_gitiles_config().fetch_archive_deadline)
  if not archive:
    logging.warning(
        'Configuration %s does not exist. Probably it was deleted', config_set)
    attempt.success = True
    attempt.message = 'Config directory not found. Imported as empty'
  else:
    # Extract files and save them to Blobs outside ConfigSet transaction.
    files, validation_result = _read_and_validate_archive(
        config_set, rev_key, archive, location)
    if validation_result.has_errors:
      logging.warning('Invalid revision %s@%s', config_set, revision)
      notifications.notify_gitiles_rejection(
          config_set, location, validation_result)

      attempt.success = False
      attempt.message = 'Validation errors'
      attempt.validation_messages = [
        storage.ImportAttempt.ValidationMessage(
            severity=config.Severity.lookup_by_number(m.severity),
            text=m.text,
        )
        for m in validation_result.messages
      ]
      attempt.put()
      return
    rev_entities += files
    attempt.success = True
    attempt.message = 'Imported'

  @ndb.transactional
  def txn():
    if force_update or not rev_key.get():
      ndb.put_multi(rev_entities)
    attempt.put()

  txn()
  logging.info('Imported revision %s/%s', config_set, location.treeish)


def _read_and_validate_archive(config_set, rev_key, archive, location):
  """Reads an archive, validates all files, imports blobs and returns files.

  If all files are valid, saves contents to Blob entities and returns
  files with their hashes.

  Return:
      (files, validation_result) tuple.
  """
  logging.info('%s archive size: %d bytes' % (config_set, len(archive)))

  stream = StringIO.StringIO(archive)
  blob_futures = []
  with tarfile.open(mode='r|gz', fileobj=stream) as tar:
    files = {}
    ctx = config.validation.Context()
    for item in tar:
      if not item.isreg():  # pragma: no cover
        continue
      logging.info('Found file "%s"', item.name)
      with contextlib.closing(tar.extractfile(item)) as extracted:
        content = extracted.read()
        files[item.name] = content
        with ctx.prefix(item.name + ': '):
          validation.validate_config(config_set, item.name, content, ctx=ctx)

  if ctx.result().has_errors:
    return [], ctx.result()

  entities = []
  for name, content in files.iteritems():
    content_hash = storage.compute_hash(content)
    blob_futures.append(storage.import_blob_async(
      content=content, content_hash=content_hash))
    entities.append(
      storage.File(
        id=name,
        parent=rev_key,
        content_hash=content_hash,
        url=str(location.join(name)))
    )
  # Wait for Blobs to be imported before proceeding.
  ndb.Future.wait_all(blob_futures)
  return entities, ctx.result()


def _import_config_set(config_set, location):
  """Imports the latest version of config set from a Gitiles location.

  Args:
    config_set (str): name of a config set to import.
    location (gitiles.Location): location of the config set.
  """
  assert config_set
  assert location

  commit = None
  def save_attempt(success, msg):
    storage.ImportAttempt(
      key=storage.last_import_attempt_key(config_set),
      revision=_commit_to_revision_info(commit, location),
      success=success,
      message=msg,
    ).put()

  try:
    logging.debug('Importing %s from %s', config_set, location)

    log = location.get_log(
        limit=1, deadline=get_gitiles_config().fetch_log_deadline)
    if not log or not log.commits:

      @ndb.transactional
      def txn():
        cs = storage.ConfigSet.get_by_id(config_set)
        if cs:
          # The config set existed once, but its git history disappeared.
          # Most probably, it is Gitiles bug https://crbug.com/819453#c21
          raise HistoryDisappeared()

        save_attempt(False, 'Could not load commit log')

      txn()

      raise NotFoundError('Could not load commit log for %s' % (location,))

    commit = log.commits[0]

    config_set_key = ndb.Key(storage.ConfigSet, config_set)
    config_set_entity = config_set_key.get()
    force_update = (config_set_entity and
                    config_set_entity.version < storage.ConfigSet.CUR_VERSION)
    if (config_set_entity and config_set_entity.latest_revision == commit.sha
        and not force_update):
      save_attempt(True, 'Up-to-date')
      logging.debug('Up-to-date')
      return

    logging.info(
        'Rolling %s => %s',
        config_set_entity and config_set_entity.latest_revision, commit.sha)
    _import_revision(config_set, location, commit, force_update)
  except urlfetch_errors.DeadlineExceededError:
    save_attempt(False, 'Could not import: deadline exceeded')
    raise Error(
        'Could not import config set %s from %s: urlfetch deadline exceeded' %
            (config_set, location))
  except net.AuthError:
    save_attempt(False, 'Could not import: permission denied')
    raise Error(
        'Could not import config set %s from %s: permission denied' % (
            config_set, location))


## Import individual config set


def import_service(service_id, conf=None):
  if not config.validation.is_valid_service_id(service_id):
    raise ValueError('Invalid service id: %s' % service_id)
  # TODO(nodir): import services from location specified in services.cfg
  conf = conf or admin.GlobalConfig.fetch()
  if not conf:
    raise Exception('not configured')
  if conf.services_config_storage_type != GITILES_STORAGE_TYPE:
    raise Error('services are not stored on Gitiles')
  if not conf.services_config_location:
    raise Error('services config location is not set')
  location_root = _resolved_location(conf.services_config_location)
  service_location = location_root._replace(
      path=os.path.join(location_root.path, service_id))
  _import_config_set('services/%s' % service_id, service_location)


def import_project(project_id):
  if not config.validation.is_valid_project_id(project_id):
    raise ValueError('Invalid project id: %s' % project_id)

  config_set = 'projects/%s' % project_id

  project = projects.get_project(project_id)
  if project is None:
    raise NotFoundError('project %s not found' % project_id)
  if project.config_location.storage_type != GITILES_LOCATION_TYPE:
    raise Error('project %s is not a Gitiles project' % project_id)

  try:
    loc = _resolved_location(project.config_location.url)
  except gitiles.TreeishResolutionError:

    @ndb.transactional
    def txn():
      key = ndb.Key(storage.ConfigSet, config_set)
      if key.get():
        logging.warning(
            'treeish was not resolved in URL "%s" => delete project',
            project.config_location.url)
        key.delete()

    txn()
    return

  # Update project repo info.
  repo_url = str(loc._replace(treeish=None, path=None))
  projects.update_import_info(
      project_id, projects.RepositoryType.GITILES, repo_url)

  _import_config_set(config_set, loc)


def import_ref(project_id, ref_name):
  if not config.validation.is_valid_project_id(project_id):
    raise ValueError('Invalid project id "%s"' % project_id)
  if not config.validation.is_valid_ref_name(ref_name):
    raise ValueError('Invalid ref name "%s"' % ref_name)

  project = projects.get_project(project_id)
  if project is None:
    raise NotFoundError('project %s not found' % project_id)
  if project.config_location.storage_type != GITILES_LOCATION_TYPE:
    raise Error('project %s is not a Gitiles project' % project_id)

  # We don't call _resolved_location here because we are replacing treeish and
  # path below anyway.
  loc = gitiles.Location.parse(project.config_location.url)

  ref = None
  for r in projects.get_refs([project_id])[project_id] or ():
    if r.name == ref_name:
      ref = r

  if ref is None:
    raise NotFoundError(
        ('ref "%s" is not found in project %s. '
         'Possibly it is not declared in projects/%s:refs.cfg') %
        (ref_name, project_id, project_id))
  cfg = get_gitiles_config()
  loc = loc._replace(
      treeish=ref_name,
      path=ref.config_path or cfg.ref_config_default_path,
  )
  _import_config_set('projects/%s/%s' % (project_id, ref_name), loc)


def import_config_set(config_set):
  """Imports a config set."""
  service_match = config.SERVICE_CONFIG_SET_RGX.match(config_set)
  if service_match:
    service_id = service_match.group(1)
    return import_service(service_id)

  project_match = config.PROJECT_CONFIG_SET_RGX.match(config_set)
  if project_match:
    project_id = project_match.group(1)
    return import_project(project_id)

  ref_match = config.REF_CONFIG_SET_RGX.match(config_set)
  if ref_match:
    project_id = ref_match.group(1)
    ref_name = ref_match.group(2)
    return import_ref(project_id, ref_name)

  raise ValueError('Invalid config set "%s' % config_set)


## A cron job that schedules an import push task for each config set


def _service_config_sets(location_root):
  """Returns a list of all service config sets stored in Gitiles."""
  # TODO(nodir): import services from location specified in services.cfg
  assert location_root
  tree = location_root.get_tree()

  ret = []
  for service_entry in tree.entries:
    service_id = service_entry.name
    if service_entry.type != 'tree':
      continue
    if not config.validation.is_valid_service_id(service_id):
      logging.error('Invalid service id: %s', service_id)
      continue
    ret.append('services/%s' % service_id)
  return ret


def _project_and_ref_config_sets():
  """Returns a list of project and ref config sets stored in Gitiles."""
  projs = projects.get_projects()
  refs = projects.get_refs([p.id for p in projs])
  ret = []

  for project in projs:
    ret.append('projects/%s' % project.id)

    # Import refs of the project
    for ref in refs[project.id] or []:
      assert ref.name
      assert ref.name.startswith('refs/'), ref.name
      ret.append('projects/%s/%s' % (project.id, ref.name))
  return ret


def cron_run_import():  # pragma: no cover
  """Schedules a push task for each config set imported from Gitiles."""
  conf = admin.GlobalConfig.fetch()

  # Collect the list of config sets to import.
  config_sets = []
  if (conf and conf.services_config_storage_type == GITILES_STORAGE_TYPE and
      conf.services_config_location):
    loc = _resolved_location(conf.services_config_location)
    config_sets += _service_config_sets(loc)
  config_sets += _project_and_ref_config_sets()

  # For each config set, schedule a push task.
  # This assumes that tasks are processed faster than we add them.
  tasks = [
    taskqueue.Task(url='/internal/task/luci-config/gitiles_import/%s' % cs)
    for cs in config_sets
  ]

  # Task Queues try to preserve FIFO semantics. But if something is partially
  # failing (e.g. LUCI Config hitting gitiles quota midway through update), we'd
  # want to make a slow progress across all config sets. Shuffle tasks, so we
  # don't give accidental priority to lexicographically first ones.
  random.shuffle(tasks)

  q = taskqueue.Queue('gitiles-import')
  pending = tasks
  while pending:
    batch = pending[:100]
    pending = pending[len(batch):]
    q.add(batch)

  logging.info('scheduled %d tasks', len(tasks))
