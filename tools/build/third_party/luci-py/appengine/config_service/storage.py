# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Storage of config files."""

import hashlib

from google.appengine.api import app_identity
from google.appengine.ext import ndb
from google.appengine.ext.ndb import msgprop
from google.protobuf import text_format

from components import config
from components import utils


class ServiceDynamicMetadata(ndb.Model):
  """Contains service dynamic metadata.

  Entity key:
    Id is the service id.
  """
  # contains ServiceDynamicMetadata message in binary format.
  # see components/config/proto/service_config.proto file.
  metadata = ndb.BlobProperty()


class Blob(ndb.Model):
  """Content-addressed blob. Immutable.

  Entity key:
    Id is content hash that has format "v1:<sha>"
    where sha is hex-encoded Git-compliant SHA-1 of
    'blob {content len}\0{content}'. Computed by compute_hash function.
    Blob has no parent.
  """
  created_ts = ndb.DateTimeProperty(auto_now_add=True)
  content = ndb.BlobProperty(required=True)


class ConfigSet(ndb.Model):
  """Versioned collection of config files.

  Entity key:
    Id is a config set name. Examples: services/luci-config, projects/chromium.

  gitiles_import.py relies on the fact that this class has only one attribute.
  """
  CUR_VERSION = 2

  # last imported revision of the config set. See also Revision and File.
  latest_revision = ndb.StringProperty()
  latest_revision_url = ndb.StringProperty(indexed=False)
  latest_revision_time = ndb.DateTimeProperty(indexed=False)
  latest_revision_committer_email = ndb.StringProperty(indexed=False)

  location = ndb.StringProperty(required=True)

  version = ndb.IntegerProperty(default=0)


class RevisionInfo(ndb.Model):
  """Contains revision metadata.

  Used with StructuredProperty.
  """
  id = ndb.StringProperty(required=True, indexed=False)
  url = ndb.StringProperty(indexed=False)
  time = ndb.DateTimeProperty(indexed=False)
  committer_email = ndb.StringProperty(indexed=False)


class ImportAttempt(ndb.Model):
  """Describes what happened last time we tried to import a config set.

  Entity key:
    Parent is ConfigSet (does not have to exist).
    ID is "last".
  """
  time = ndb.DateTimeProperty(auto_now_add=True, required=True, indexed=False)
  revision = ndb.StructuredProperty(RevisionInfo, indexed=False)
  success = ndb.BooleanProperty(required=True, indexed=False)
  message = ndb.StringProperty(required=True, indexed=False)

  class ValidationMessage(ndb.Model):
    path = ndb.StringProperty(indexed=False)
    severity = msgprop.EnumProperty(config.Severity, indexed=False)
    text = ndb.StringProperty(indexed=False)

  validation_messages = ndb.StructuredProperty(ValidationMessage, repeated=True)


class Revision(ndb.Model):
  """A single revision of a config set. Immutable.

  Parent of File entities. Revision entity does not have to exist.

  Entity key:
    Id is a revision name. If imported from Git, it is a commit hash.
    Parent is ConfigSet.
  """


class File(ndb.Model):
  """A single file in a revision. Immutable.

  Entity key:
    Id is a filename without a leading slash. Parent is Revision.
  """
  created_ts = ndb.DateTimeProperty(auto_now_add=True)
  # hash of the file content, computed by compute_hash().
  # A Blob entity with this key must exist.
  content_hash = ndb.StringProperty(indexed=False, required=True)
  # A pinned, fully resolved URL to this file.
  url = ndb.StringProperty(indexed=False)

  def _pre_put_hook(self):
    assert isinstance(self.key.id(), str)
    assert not self.key.id().startswith('/')


def last_import_attempt_key(config_set):
  return ndb.Key(ConfigSet, config_set, ImportAttempt, 'last')


def get_file_keys(config_set, revision):
  return File.query(
      default_options=ndb.QueryOptions(keys_only=True),
      ancestor=ndb.Key(ConfigSet, config_set, Revision, revision)).fetch()


@ndb.tasklet
def get_config_sets_async(config_set=None):
  if config_set:
    existing = yield ConfigSet.get_by_id_async(config_set)
    config_sets = [existing or ConfigSet(id=config_set)]
  else:
    config_sets = yield ConfigSet.query().fetch_async()
  raise ndb.Return(config_sets)


@ndb.tasklet
def get_latest_revisions_async(config_sets):
  """Returns a mapping {config_set: latest_revision}.

  A returned latest_revision may be None.
  """
  assert isinstance(config_sets, list)
  entities = yield ndb.get_multi_async(
      ndb.Key(ConfigSet, cs) for cs in config_sets
  )
  latest_revisions = {e.key.id(): e.latest_revision for e in entities if e}

  raise ndb.Return({cs: latest_revisions.get(cs) for cs in config_sets})


@ndb.tasklet
def get_config_hashes_async(revs, path):
  """Returns a mapping {config_set: (revision, file_url, content_hash)}.

  A returned revision, file_url, or content_hash may be None.

  Args:
    revs: a mapping {config_set: revision}.
      If revision is None, latest will be used.
    path (str): path to the file.
  """
  assert isinstance(revs, dict)
  for cs, rev in revs.iteritems():
    assert isinstance(cs, basestring)
    assert cs
    assert rev is None or isinstance(rev, basestring)
    assert rev is None or rev
  assert path
  assert not path.startswith('/')

  # Resolve latest revisions.
  revs = revs.copy()
  config_sets_without_rev = [cs for cs, rev in revs.iteritems() if not rev]
  if config_sets_without_rev:
    latest_revisions = yield get_latest_revisions_async(config_sets_without_rev)
    revs.update(latest_revisions)

  # Load content hashes
  file_entities = yield ndb.get_multi_async([
    ndb.Key(
        ConfigSet, cs,
        Revision, rev,
        File, path)
    for cs, rev in revs.iteritems()
    if rev
  ])
  content_url_and_hashes = {
    # map key is config set
    f.key.parent().parent().id(): (f.url, f.content_hash)
    for f in file_entities
    if f
  }
  raise ndb.Return({
    cs: (
        (rev, ) + content_url_and_hashes.get(cs)
        if content_url_and_hashes.get(cs) else (None, None, None))
    for cs, rev in revs.iteritems()
  })


@ndb.tasklet
def get_configs_by_hashes_async(content_hashes):
  """Returns a mapping {hash: content}."""
  assert isinstance(content_hashes, list)
  if not content_hashes:
    raise ndb.Return({})
  assert all(h for h in content_hashes)
  content_hashes = list(set(content_hashes))
  blobs = yield ndb.get_multi_async(ndb.Key(Blob, h) for h in content_hashes)
  raise ndb.Return({
    h: b.content if b else None
    for h, b in zip(content_hashes, blobs)
  })


@ndb.tasklet
def get_latest_configs_async(config_sets, path, hashes_only=False):
  """Returns a mapping {config_set: (revision, file_url, hash, content)}.

  If hash_only is True, returned content items are None.
  """
  assert isinstance(config_sets, list)
  # Resolve content hashes.
  revs_and_hashes = yield get_config_hashes_async(
      {cs: None for cs in config_sets}, path)

  if hashes_only:
    contents = {}
  else:
    hashes = [h for _, _, h in revs_and_hashes.itervalues() if h]
    contents = yield get_configs_by_hashes_async(hashes)

  raise ndb.Return({
    cs: (rev, file_url, content_hash, contents.get(content_hash))
    for cs, (rev, file_url, content_hash) in revs_and_hashes.iteritems()
  })


@ndb.tasklet
def get_latest_messages_async(config_sets, path, message_factory):
  """Reads latest config files as a text-formatted protobuf message.

  |message_factory| is a function that creates a message. Typically the message
  type itself. Values found in the retrieved config file are merged into the
  return value of the factory.

  Returns:
    A mapping {config_set: message}. A message is empty if the file does not
    exist.
  """
  configs = yield get_latest_configs_async(config_sets, path)

  def to_msg(text):
    msg = message_factory()
    if text:
      text_format.Merge(text, msg)
    return msg

  raise ndb.Return({
    cs: to_msg(text)
    for cs, (_, _, _, text) in configs.iteritems()
  })


@utils.cache
def get_self_config_set():
  return 'services/%s' % app_identity.get_application_id()


@ndb.tasklet
def get_self_config_async(path, message_factory):
  """Parses a config file in the app's config set into a protobuf message."""
  cache_key = 'get_self_config_async(%r)' % path
  ctx = ndb.get_context()
  cached = yield ctx.memcache_get(cache_key)
  if cached:
    msg = message_factory()
    msg.ParseFromString(cached)
    raise ndb.Return(msg)

  cs = get_self_config_set()
  messages = yield get_latest_messages_async([cs], path, message_factory)
  msg = messages[cs]
  yield ctx.memcache_set(cache_key, msg.SerializeToString(), time=60)
  raise ndb.Return(msg)


def compute_hash(content):
  """Computes Blob id by its content.

  See Blob docstring for Blob id format.
  """
  sha = hashlib.sha1()
  sha.update('blob %d\0' % len(content))
  sha.update(content)
  return 'v1:%s' % sha.hexdigest()


@ndb.tasklet
def import_blob_async(content, content_hash=None):
  """Saves |content| to a Blob entity.

  Returns:
    Content hash.
  """
  content_hash = content_hash or compute_hash(content)

  # pylint: disable=E1120
  if not Blob.get_by_id(content_hash):
    yield Blob(id=content_hash, content=content).put_async()
  raise ndb.Return(content_hash)


def import_blob(content, content_hash=None):
  return import_blob_async(content, content_hash=content_hash).get_result()
