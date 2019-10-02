# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Imports groups from some external tar.gz bundle or plain text list.

External URL should serve *.tar.gz file with the following file structure:
  <external group system name>/<group name>:
    userid
    userid
    ...

For example ldap.tar.gz may look like:
  ldap/trusted-users:
    jane
    joe
    ...
  ldap/all:
    jane
    joe
    ...

Each tarball may have groups from multiple external systems, but groups from
some external system must not be split between multiple tarballs. When importer
sees <external group system name>/* in a tarball, it modifies group list from
that system on the server to match group list in the tarball _exactly_,
including removal of groups that are on the server, but no longer present in
the tarball.

Plain list format should have one userid per line and can only describe a single
group in a single system. Such groups will be added to 'external/*' groups
namespace. Removing such group from importer config will remove it from
service too.

The service can also be configured to accept tarball uploads (instead of
fetching them). Fetched and uploaded tarballs are handled in the exact same way,
in particular all caveats related to external group system names apply.
"""

import contextlib
import logging
import StringIO
import tarfile

from google.appengine.ext import ndb

from google import protobuf

from components import auth
from components import net
from components import utils
from components.auth import model

from proto import config_pb2


class BundleImportError(Exception):
  """Base class for errors while fetching external bundle."""


class BundleFetchError(BundleImportError):
  """Failed to fetch the archive from remote URL."""

  def __init__(self, url, status_code, content):
    super(BundleFetchError, self).__init__()
    self.url = url
    self.status_code = status_code
    self.content = content

  def __str__(self):
    return 'Request to %s failed with code %d:\n%r' % (
        self.url, self.status_code, self.content)


class BundleUnpackError(BundleImportError):
  """Failed to untar the archive."""

  def __init__(self, inner_exc):
    super(BundleUnpackError, self).__init__()
    self.inner_exc = inner_exc

  def __str__(self):
    return 'Not a valid tar archive: %s' % self.inner_exc


class BundleBadFormatError(BundleImportError):
  """Group file in bundle has invalid format."""

  def __init__(self, inner_exc):
    super(BundleBadFormatError, self).__init__()
    self.inner_exc = inner_exc

  def __str__(self):
    return 'Bundle contains invalid group file: %s' % self.inner_exc


def config_key():
  """Key of GroupImporterConfig singleton entity."""
  return ndb.Key('GroupImporterConfig', 'config')


class GroupImporterConfig(ndb.Model):
  """Singleton entity with group importer configuration JSON."""
  config_proto = ndb.TextProperty()
  config_revision = ndb.JsonProperty() # see config.py, _update_imports_config
  modified_by = auth.IdentityProperty(indexed=False)
  modified_ts = ndb.DateTimeProperty(auto_now=True, indexed=False)


def validate_config(text):
  """Deserializes text to config_pb2.GroupImporterConfig and validates it.

  Raise:
    ValueError if config is not valid.
  """
  msg = config_pb2.GroupImporterConfig()
  try:
    protobuf.text_format.Merge(text, msg)
  except protobuf.text_format.ParseError as ex:
    raise ValueError('Config is badly formated: %s' % ex)
  validate_config_proto(msg)


def validate_config_proto(config):
  """Checks config_pb2.GroupImporterConfig for correctness.

  Raises:
    ValueError if config has invalid structure.
  """
  if not isinstance(config, config_pb2.GroupImporterConfig):
    raise ValueError('Not GroupImporterConfig proto message')

  # Validate fields common to Tarball and Plainlist.
  for entry in list(config.tarball) + list(config.plainlist):
    if not entry.url:
      raise ValueError(
          '"url" field is required in %s' % entry.__class__.__name__)

  # Check TarballUpload names are unique, validate authorized_uploader emails.
  tarball_upload_names = set()
  for entry in config.tarball_upload:
    if not entry.name:
      raise ValueError('Some tarball_upload entry does\'t have a name')
    if entry.name in tarball_upload_names:
      raise ValueError(
          'tarball_upload entry "%s" is specified twice' % entry.name)
    tarball_upload_names.add(entry.name)
    if not entry.authorized_uploader:
      raise ValueError(
          'authorized_uploader is required in tarball_upload entry "%s"' %
          entry.name)
    for email in entry.authorized_uploader:
      try:
        model.Identity(model.IDENTITY_USER, email)
      except ValueError:
        raise ValueError(
            'invalid email "%s" in tarball_upload entry "%s"' %
            (email, entry.name))

  # Validate tarball and tarball_upload fields.
  seen_systems = set(['external'])
  for tarball in list(config.tarball) + list(config.tarball_upload):
    title = ''
    if isinstance(tarball, config_pb2.GroupImporterConfig.TarballEntry):
      title = '"tarball" entry with URL "%s"' % tarball.url
    elif isinstance(tarball, config_pb2.GroupImporterConfig.TarballUploadEntry):
      title = '"tarball_upload" entry with name "%s"' % tarball.name
    if not tarball.systems:
      raise ValueError('%s needs "systems" field' % title)
    # There should be no overlap in systems between different bundles.
    twice = set(tarball.systems) & seen_systems
    if twice:
      raise ValueError(
          '%s is specifying a duplicate system(s): %s' % (title, sorted(twice)))
    seen_systems.update(tarball.systems)

  # Validate plainlist fields.
  seen_groups = set()
  for plainlist in config.plainlist:
    if not plainlist.group:
      raise ValueError(
          '"plainlist" entry "%s" needs "group" field' % plainlist.url)
    if plainlist.group in seen_groups:
      raise ValueError('The group "%s" imported twice' % plainlist.group)
    seen_groups.add(plainlist.group)


def read_config():
  """Returns importer config as a text blob (or '' if not set)."""
  e = config_key().get()
  return e.config_proto if e else ''


def write_config(text, config_revision=None, modified_by=None):
  """Validates config text blobs and puts it into the datastore.

  Raises:
    ValueError on invalid format.
  """
  validate_config(text)
  e = GroupImporterConfig(
      key=config_key(),
      config_proto=text,
      config_revision=config_revision,
      modified_by=modified_by or auth.get_service_self_identity())
  e.put()


def load_config():
  """Reads and parses the config, returns it as GroupImporterConfig or None.

  Raises BundleImportError if the config can't be parsed or doesn't pass
  the validation. Missing config is not an error (the function just returns
  None).
  """
  config_text = read_config()
  if not config_text:
    return None
  config = config_pb2.GroupImporterConfig()
  try:
    protobuf.text_format.Merge(config_text, config)
  except protobuf.text_format.ParseError as ex:
    raise BundleImportError('Bad config format: %s' % ex)
  try:
    validate_config_proto(config)
  except ValueError as ex:
    raise BundleImportError('Bad config structure: %s' % ex)
  return config


def ingest_tarball(name, content):
  """Handles upload of tarballs specified in 'tarball_upload' config entries.

  Expected to be called in an auth context of the upload PUT request.

  Args:
    name: name of the corresponding 'tarball_upload' entry (defines ACLs).
    content: raw byte buffer with *.tar.gz file data.

  Returns:
    (list of modified groups, new AuthDB revision number or 0 if no changes).

  Raises:
    auth.AuthorizationError if caller is not authorized to do this upload.
    BundleImportError if the tarball can't be imported (e.g. wrong format).
  """
  # Return generic HTTP 403 error unless we can verify the caller to avoid
  # leaking information about our config.
  config = load_config()
  if not config:
    logging.error('Group imports are not configured')
    raise auth.AuthorizationError()

  # Same here. We should not leak config entry names to untrusted callers.
  entry = None
  for entry in config.tarball_upload:
    if entry.name == name:
      break
  else:
    logging.error('No such tarball_upload entry in the config: "%s"', name)
    raise auth.AuthorizationError()

  # The caller must be specified in 'authorized_uploader' field.
  caller = auth.get_current_identity()
  for email in entry.authorized_uploader:
    if caller == model.Identity(model.IDENTITY_USER, email):
      break
  else:
    logging.error(
        'Caller %s is not authorized to upload tarball "%s"',
        caller.to_bytes(), entry.name)
    raise auth.AuthorizationError()

  # Authorization check passed. Now parse the tarball, converting it into
  # {system -> {group -> identities}} map (aka "bundles set") and import it into
  # the datastore.
  logging.info('Ingesting tarball "%s" uploaded by %s', name, caller.to_bytes())
  bundles = load_tarball(content, entry.systems, entry.groups, entry.domain)
  return import_bundles(
      bundles, caller, 'Uploaded as "%s" tarball' % entry.name)


def import_external_groups():
  """Refetches external groups specified via 'tarball' or 'plainlist' entries.

  Runs as a cron task. Raises BundleImportError in case of import errors.
  """
  config = load_config()
  if not config:
    logging.info('Not configured')
    return

  # Fetch files specified in the config in parallel.
  entries = list(config.tarball) + list(config.plainlist)
  files = utils.async_apply(
      entries, lambda e: fetch_file_async(e.url, e.oauth_scopes))

  # {system name -> group name -> list of identities}
  bundles = {}
  for e, contents in files:
    # Unpack tarball into {system name -> group name -> list of identities}.
    if isinstance(e, config_pb2.GroupImporterConfig.TarballEntry):
      fetched = load_tarball(contents, e.systems, e.groups, e.domain)
      assert not (
          set(fetched) & set(bundles)), (fetched.keys(), bundles.keys())
      bundles.update(fetched)
      continue

    # Add plainlist group to 'external/*' bundle.
    if isinstance(e, config_pb2.GroupImporterConfig.PlainlistEntry):
      group = load_group_file(contents, e.domain)
      name = 'external/%s' % e.group
      if 'external' not in bundles:
        bundles['external'] = {}
      assert name not in bundles['external'], name
      bundles['external'][name] = group
      continue

    assert False, 'Unreachable'

  import_bundles(
      bundles, model.get_service_self_identity(), 'External group import')


def import_bundles(bundles, provided_by, change_log_comment):
  """Imports given set of bundles all at once.

  A bundle is a dict with groups that is result of a processing of some tarball.
  A bundle specifies the _desired state_ of all groups under some system, e.g.
  import_bundles({'ldap': {}}, ...) will REMOVE all existing 'ldap/*' groups.

  Group names in the bundle are specified in their full prefixed form (with
  system name prefix). An example of expected 'bundles':
  {
    'ldap': {
      'ldap/group': [Identity(...), Identity(...)],
    },
  }

  Args:
    bundles: dict {system name -> {group name -> list of identities}}.
    provided_by: auth.Identity to put in 'modified_by' or 'created_by' fields.
    change_log_comment: a comment to put in the change log.

  Returns:
    (list of modified groups, new AuthDB revision number or 0 if no changes).
  """
  # Nothing to process?
  if not bundles:
    return [], 0

  @ndb.transactional
  def snapshot_groups():
    """Fetches all existing groups and AuthDB revision number."""
    groups = model.AuthGroup.query(ancestor=model.root_key()).fetch_async()
    return auth.get_auth_db_revision(), groups.get_result()

  @ndb.transactional
  def apply_import(revision, entities_to_put, entities_to_delete, ts):
    """Transactionally puts and deletes a bunch of entities."""
    # DB changed between transactions, retry.
    if auth.get_auth_db_revision() != revision:
      return False
    # Apply mutations, bump revision number.
    for e in entities_to_put:
      e.record_revision(
          modified_by=provided_by,
          modified_ts=ts,
          comment=change_log_comment)
    for e in entities_to_delete:
      e.record_deletion(
          modified_by=provided_by,
          modified_ts=ts,
          comment=change_log_comment)
    futures = []
    futures.extend(ndb.put_multi_async(entities_to_put))
    futures.extend(ndb.delete_multi_async(e.key for e in entities_to_delete))
    for f in futures:
      f.check_success()
    auth.replicate_auth_db()
    return True

  # Try to apply the change until success or deadline. Split transaction into
  # two (assuming AuthDB changes infrequently) to avoid reading and writing too
  # much stuff from within a single transaction (and to avoid keeping the
  # transaction open while calculating the diff).
  while True:
    # Use same timestamp everywhere to reflect that groups were imported
    # atomically within a single transaction.
    ts = utils.utcnow()
    entities_to_put = []
    entities_to_delete = []
    revision, existing_groups = snapshot_groups()
    for system, groups in bundles.iteritems():
      to_put, to_delete = prepare_import(
          system, existing_groups, groups, ts, provided_by)
      entities_to_put.extend(to_put)
      entities_to_delete.extend(to_delete)
    if not entities_to_put and not entities_to_delete:
      break
    if apply_import(revision, entities_to_put, entities_to_delete, ts):
      revision += 1
      break

  if not entities_to_put and not entities_to_delete:
    logging.info('No changes')
    return [], 0

  logging.info('Groups updated, new authDB rev is %d', revision)
  updated_groups = []
  for e in entities_to_put + entities_to_delete:
    logging.info('%s', e.key.id())
    updated_groups.append(e.key.id())
  return sorted(updated_groups), revision


def load_tarball(content, systems, groups, domain):
  """Unzips tarball with groups and deserializes them.

  Args:
    content: byte buffer with *.tar.gz data.
    systems: names of external group systems expected to be in the bundle.
    groups: list of group name to extract, or empty to extract all.
    domain: email domain to append to naked user ids.

  Returns:
    Dict {system name -> {group name -> list of identities}}.

  Raises:
    BundleImportError on errors.
  """
  bundles = {s: {} for s in systems}
  try:
    # Expected filenames are <external system name>/<group name>, skip
    # everything else.
    for filename, fileobj in extract_tar_archive(content):
      chunks = filename.split('/')
      if len(chunks) != 2 or not auth.is_valid_group_name(filename):
        logging.warning('Skipping file %s, not a valid name', filename)
        continue
      if groups and filename not in groups:
        continue
      system = chunks[0]
      if system not in systems:
        logging.warning('Skipping file %s, not allowed', filename)
        continue
      # Do not catch BundleBadFormatError here and in effect reject the whole
      # bundle if at least one group file is broken. That way all existing
      # groups will stay intact. Simply ignoring broken group here will cause
      # the importer to remove it completely.
      bundles[system][filename] = load_group_file(fileobj.read(), domain)
  except tarfile.TarError as exc:
    raise BundleUnpackError('Not a valid tar archive: %s' % exc)
  return bundles


def load_group_file(body, domain):
  """Given body of imported group file returns list of Identities.

  Raises BundleBadFormatError if group file is malformed.
  """
  members = set()
  for uid in body.strip().splitlines():
    email = '%s@%s' % (uid, domain) if domain else uid
    if email.endswith('@gtempaccount.com'):
      # See https://support.google.com/a/answer/185186?hl=en. These emails look
      # like 'name%domain@gtempaccount.com'. We convert them to 'name@domain'.
      email = email[:-len('@gtempaccount.com')].replace('%', '@')
    try:
      members.add(auth.Identity(auth.IDENTITY_USER, email))
    except ValueError as exc:
      raise BundleBadFormatError(exc)
  return sorted(members, key=lambda x: x.to_bytes())


@ndb.tasklet
def fetch_file_async(url, oauth_scopes):
  """Fetches a file optionally using OAuth2 for authentication.

  Args:
    url: url to a file to fetch.
    oauth_scopes: list of OAuth scopes to use when generating access_token for
        accessing |url|, if not set or empty - do not use OAuth.

  Returns:
    Byte buffer with file's body.

  Raises:
    BundleImportError on fetch errors.
  """
  try:
    data = yield net.request_async(url, scopes=oauth_scopes, deadline=60)
    raise ndb.Return(data)
  except net.Error as e:
    raise BundleFetchError(url, e.status_code, e.response)


def extract_tar_archive(content):
  """Given a body of tar.gz file yields pairs (file name, file obj)."""
  stream = StringIO.StringIO(content)
  with tarfile.open(mode='r|gz', fileobj=stream) as tar:
    for item in tar:
      if item.isreg():
        with contextlib.closing(tar.extractfile(item)) as extracted:
          yield item.name, extracted


def prepare_import(
    system_name, existing_groups, imported_groups, timestamp, provided_by):
  """Prepares lists of entities to put and delete to apply group import.

  Operates exclusively over '<system name>/*' groups.

  Args:
    system_name: name of external groups system being imported (e.g. 'ldap'),
      all existing groups belonging to that system will be replaced with
      |imported_groups|.
    existing_groups: ALL existing groups (not only '<system name>/*' ones).
    imported_groups: dict {imported group name -> list of identities}.
    timestamp: modification timestamp to set on all touched entities.
    provided_by: auth.Identity to put in 'modified_by' or 'created_by' fields.

  Returns:
    (List of entities to put, list of entities to delete).
  """
  # Return values of this function.
  to_put = []
  to_delete = []

  # Pick only groups that belong to |system_name|.
  system_groups = {
    g.key.id(): g for g in existing_groups
    if g.key.id().startswith('%s/' % system_name)
  }

  def clear_group(group_name):
    assert group_name.startswith('%s/' % system_name), group_name
    ent = system_groups[group_name]
    if ent.members:
      ent.members = []
      ent.modified_ts = timestamp
      ent.modified_by = provided_by
      to_put.append(ent)

  def delete_group(group_name):
    assert group_name.startswith('%s/' % system_name), group_name
    to_delete.append(system_groups[group_name])

  def create_group(group_name):
    assert group_name.startswith('%s/' % system_name), group_name
    ent = model.AuthGroup(
        key=model.group_key(group_name),
        members=imported_groups[group_name],
        created_ts=timestamp,
        created_by=provided_by,
        modified_ts=timestamp,
        modified_by=provided_by)
    to_put.append(ent)

  def update_group(group_name):
    assert group_name.startswith('%s/' % system_name), group_name
    existing = system_groups[group_name]
    imported = imported_groups[group_name]
    if existing.members != imported:
      existing.members = imported
      existing.modified_ts = timestamp
      existing.modified_by = provided_by
      to_put.append(existing)

  # Delete groups that are no longer present in the bundle. If group is
  # referenced somewhere, just clear its members list (to avoid creating
  # inconsistency in group inclusion graph).
  for group_name in (set(system_groups) - set(imported_groups)):
    if any(group_name in g.nested for g in existing_groups):
      clear_group(group_name)
    else:
      delete_group(group_name)

  # Create new groups.
  for group_name in (set(imported_groups) - set(system_groups)):
    create_group(group_name)

  # Update existing groups.
  for group_name in (set(imported_groups) & set(system_groups)):
    update_group(group_name)

  return to_put, to_delete
