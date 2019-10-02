# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Models and functions to build and query Auth DB change log."""

import datetime
import logging
import re
import webapp2

from google.appengine.api import datastore_errors
from google.appengine.api import modules
from google.appengine.api import taskqueue
from google.appengine.ext import ndb
from google.appengine.ext.ndb import polymodel

from google.protobuf import json_format

from components import datastore_utils
from components import decorators
from components import utils

from . import config
from . import model
from .proto import security_config_pb2


def process_change(auth_db_rev):
  """Called via task queue to convert AuthDB commit into a set of changes."""
  # We need an initial snapshot of all groups to be able to reconstruct any
  # historical snapshot later. It's important only for applications that existed
  # before change log functionality was added.
  ensure_initial_snapshot(auth_db_rev)
  # Diff all entities modified in auth_db_rev revision against previous
  # versions to produce change log for this revision. It will also kick off
  # a chain of tasks to process all previous revisions (if not processed yet).
  generate_changes(auth_db_rev)


### Code to generate a change log for AuthDB commits.


# Regexp for valid values of AuthDBChange.target property.
TARGET_RE = re.compile(
    r'^[0-9a-zA-Z_]{1,40}\$' +                # entity kind
    r'[0-9a-zA-Z_\-\./ @]{1,300}' +           # entity ID (group, IP whitelist)
    r'(\$[0-9a-zA-Z_@\-\./\:\* ]{1,200})?$')  # optional subentity ID


class AuthDBLogRev(ndb.Model):
  """Presence of this entity marks that given revision was processed already.

  Entity key is change_log_revision_key(auth_db_rev).
  """
  # When the change was processed.
  when = ndb.DateTimeProperty()
  # Application version that processed the change.
  app_version = ndb.StringProperty()


class AuthDBChange(polymodel.PolyModel):
  """Base class for change log entries.

  Has a change type and a bunch of common change properties (like who and when
  made the change). Change type order is important, it is used in UI when
  sorting changes introduces by single AuthDB commit.

  Change types represent minimal indivisible changes. Large AuthDB change is
  usually represented by (unordered) set of indivisible changes. For example,
  an act of creation of a new AuthDB group produces following changes:
    CHANGE_GROUP_CREATED
    CHANGE_GROUP_MEMBERS_ADDED
    CHANGE_GROUP_GLOBS_ADDED
    CHANGE_GROUP_NESTED_ADDED

  They are unordered, but UI sorts them based on change type integer. Thus
  CHANGE_GROUP_CREATED appears on top: it's represented by the smallest integer.

  Entity id has following format:
    <original_entity_kind>$<original_id>[$<subentity_id>]!<change_type>
  where:
    original_entity_kind: a kind of modified AuthDB entity (e.g 'AuthGroup')
    original_id: ID of modified AuthDB entity (e.g. 'Group name')
    subentity_id: optional identified of modified part of the entity, used for
        IP whitelist assignments entity (since it's just one big singleton).
    change_type: integer CHANGE_GROUP_* (see below), e.g. '1100'.

  Such key structure makes 'diff_entity_by_key' operation idempotent. A hash of
  entity body could have been used too, but having readable (and sortable) keys
  are nice.

  Parent entity is change_log_revision_key(auth_db_rev).

  Note: '$' and '!' are not likely to appear in entity names since they are
  forbidden in AuthDB names (see GROUP_NAME_RE and IP_WHITELIST_NAME_RE in
  model.py). Code here also asserts this.
  """
  # AuthDBGroupChange change types.
  CHANGE_GROUP_CREATED             = 1000
  CHANGE_GROUP_DESCRIPTION_CHANGED = 1100
  CHANGE_GROUP_OWNERS_CHANGED      = 1150
  CHANGE_GROUP_MEMBERS_ADDED       = 1200
  CHANGE_GROUP_MEMBERS_REMOVED     = 1300
  CHANGE_GROUP_GLOBS_ADDED         = 1400
  CHANGE_GROUP_GLOBS_REMOVED       = 1500
  CHANGE_GROUP_NESTED_ADDED        = 1600
  CHANGE_GROUP_NESTED_REMOVED      = 1700
  CHANGE_GROUP_DELETED             = 1800

  # AuthDBIPWhitelistChange change types.
  CHANGE_IPWL_CREATED             = 3000
  CHANGE_IPWL_DESCRIPTION_CHANGED = 3100
  CHANGE_IPWL_SUBNETS_ADDED       = 3200
  CHANGE_IPWL_SUBNETS_REMOVED     = 3300
  CHANGE_IPWL_DELETED             = 3400

  # AuthDBIPWhitelistAssignmentChange change types.
  CHANGE_IPWLASSIGN_SET   = 5000
  CHANGE_IPWLASSIGN_UNSET = 5100

  # AuthDBConfigChange change types.
  CHANGE_CONF_OAUTH_CLIENT_CHANGED     = 7000
  CHANGE_CONF_CLIENT_IDS_ADDED         = 7100
  CHANGE_CONF_CLIENT_IDS_REMOVED       = 7200
  CHANGE_CONF_TOKEN_SERVER_URL_CHANGED = 7300
  CHANGE_CONF_SECURITY_CONFIG_CHANGED  = 7400

  # What kind of a change this is (see CHANGE_*). Defines what subclass to use.
  change_type = ndb.IntegerProperty()
  # Entity (or subentity) that was changed: kind$id[$subid] (subid is optional).
  target = ndb.StringProperty()
  # AuthDB revision at which the change was made.
  auth_db_rev = ndb.IntegerProperty()
  # Who made the change.
  who = model.IdentityProperty()
  # When the change was made.
  when = ndb.DateTimeProperty()
  # Comment passed to record_revision or record_deletion.
  comment = ndb.StringProperty()
  # GAE application version at which the change was made.
  app_version = ndb.StringProperty()

  def to_jsonish(self):
    """Returns JSON-serializable dict with entity properties for REST API."""
    def simplify(v):
      if isinstance(v, list):
        return [simplify(i) for i in v]
      elif isinstance(v, datastore_utils.BytesSerializable):
        return v.to_bytes()
      elif isinstance(v, datastore_utils.JsonSerializable):
        return v.to_jsonish()
      elif isinstance(v, datetime.datetime):
        return utils.datetime_to_timestamp(v)
      return v
    as_dict = self.to_dict(exclude=['class_'])
    for k, v in as_dict.iteritems():
      if k.startswith('security_config_') and v:
        as_dict[k] = json_format.MessageToDict(
            security_config_pb2.SecurityConfig.FromString(v),
            preserving_proto_field_name=True)
      else:
        as_dict[k] = simplify(v)
    as_dict['change_type'] = _CHANGE_TYPE_TO_STRING[self.change_type]
    return as_dict


# Integer CHANGE_* => string for UI.
_CHANGE_TYPE_TO_STRING = {
  v: k[len('CHANGE_'):] for k, v in AuthDBChange.__dict__.iteritems()
  if k.startswith('CHANGE_')
}


def change_log_root_key():
  """Root key of an entity group with change log."""
  # Bump ID to rebuild the change log from *History entities.
  return ndb.Key('AuthDBLog', 'v1')


def change_log_revision_key(auth_db_rev):
  """A key of entity subgroup that keeps AuthDB change log for a revision."""
  return ndb.Key(AuthDBLogRev, auth_db_rev, parent=change_log_root_key())


def make_change_key(change):
  """Returns ndb.Key for AuthDBChange entity based on its properties."""
  # Note: in datastore all AuthDBChange subclasses have 'AuthDBChange' kind,
  # it's how PolyModel works.
  assert change.target and '$' in change.target, change.target
  assert '!' not in change.target, change.target
  return ndb.Key(
      change.__class__,
      '%s!%d' % (change.target, change.change_type),
      parent=change_log_revision_key(change.auth_db_rev))


def generate_changes(auth_db_rev):
  """Generates change log for entities modified in given revision.

  Starts a chain of tasks to process all previous revisions too (if not yet
  processed).
  """
  # Already done?
  rev = change_log_revision_key(auth_db_rev).get()
  if rev:
    logging.info(
        'Rev %d was already processed at %s by app ver %s',
        auth_db_rev, utils.datetime_to_rfc2822(rev.when), rev.app_version)
    return

  # Use kindless query to grab _all_ changed entities regardless of their kind.
  # Do keys only query to workaround ndb's inability to work with dynamically
  # generated classes (e.g. *History).
  q = ndb.Query(ancestor=model.historical_revision_key(auth_db_rev))
  changes = []
  for change_list in q.map(diff_entity_by_key, keys_only=True):
    changes.extend(change_list or [])
  logging.info('Changes found: %d', len(changes))

  # Commit changes, start processing previous version if not yet done. Need to
  # write AuthDBLogRev marker even if there were no significant changes.
  @ndb.transactional
  def commit():
    if change_log_revision_key(auth_db_rev).get():
      logging.warning('Rev %d was already processed concurrently', auth_db_rev)
      return
    rev = AuthDBLogRev(
        key=change_log_revision_key(auth_db_rev),
        when=utils.utcnow(),
        app_version=utils.get_app_version())
    ndb.put_multi(changes + [rev])
    # Enqueue a task to process previous version if not yet done.
    if auth_db_rev > 1:
      prev_rev = auth_db_rev - 1
      if not change_log_revision_key(prev_rev).get():
        logging.info('Enqueuing task to process rev %d', prev_rev)
        enqueue_process_change_task(prev_rev)
  commit()


@ndb.tasklet
def diff_entity_by_key(cur_key):
  """Given a key of historical entity, returns a bunch of AuthDBChange entities.

  Fetches the entity and its previous version, diffs the two to produce a set of
  AuthDBChange entities, returns them.
  """
  # Ancestor query with ancestor == historical_revision_key(...) may return
  # grandchildren of historical_revision_key(...). There should be no such
  # entities in the datastore. Assert this by checking cur_key has
  # historical_revision_key(...) as a parent. historical_revision_key(...) has
  # 'Rev' as a kind name.
  assert cur_key.parent().kind() == 'Rev', cur_key
  assert '$' not in cur_key.id(), cur_key # '$' is used as delimiter
  assert '!' not in cur_key.id(), cur_key # '!' is used as delimiter
  kind = cur_key.kind()
  if kind not in KNOWN_HISTORICAL_ENTITIES:
    logging.error('Unexpected entity kind in historical log: %r', kind)
    return
  # Original 'key' is not usable as is, since ndb can't find model class
  # (specified as a string), since *History classes are not in the module scope.
  # Construct a new key with the class object (*History) already provided.
  orig_kind, diff_callback = KNOWN_HISTORICAL_ENTITIES[kind]
  cur_key = ndb.Key(
      orig_kind.get_historical_copy_class(), cur_key.id(),
      parent=cur_key.parent())
  cur_ver = yield cur_key.get_async()
  if cur_ver is None:
    logging.error('Historical entity is unexpectedly gone: %s', cur_key)
    return
  # Grab a historical copy of a previous version (if any).
  prev_key = cur_ver.get_previous_historical_copy_key()
  if prev_key:
    prev_ver = yield prev_key.get_async()
  else:
    prev_ver = None
  # E.g. 'AuthGroup$group name'.
  target = '%s$%s' % (orig_kind.__name__, cur_key.id())
  # Materialize generator into list.
  changes = list(diff_callback(target, prev_ver, cur_ver))
  for ch in changes:
    assert ch.change_type
    assert ch.target and TARGET_RE.match(ch.target), ch.target
    ch.auth_db_rev = cur_ver.auth_db_rev
    ch.who = cur_ver.modified_by
    ch.when = cur_ver.modified_ts
    ch.comment = cur_ver.auth_db_change_comment
    ch.app_version = cur_ver.auth_db_app_version
    ch.key = make_change_key(ch)
  raise ndb.Return(changes)


## AuthGroup changes.


class AuthDBGroupChange(AuthDBChange):
  # Valid for CHANGE_GROUP_CREATED and CHANGE_GROUP_DESCRIPTION_CHANGED.
  description = ndb.TextProperty()
  # Valid for CHANGE_GROUP_DESCRIPTION_CHANGED, CHANGE_GROUP_DELETED.
  old_description = ndb.TextProperty()
  # Valid for CHANGE_GROUP_CREATED and CHANGE_GROUP_OWNERS_CHANGED.
  owners = ndb.StringProperty()
  # Valid for CHANGE_GROUP_OWNERS_CHANGES and CHANGE_GROUP_DELETED.
  old_owners = ndb.StringProperty()
  # Valid for CHANGE_GROUP_MEMBERS_ADDED and CHANGE_GROUP_MEMBERS_REMOVED.
  members = model.IdentityProperty(repeated=True)
  # Valid for CHANGE_GROUP_GLOBS_ADDED and CHANGE_GROUP_GLOBS_REMOVED.
  globs = model.IdentityGlobProperty(repeated=True)
  # Valid for CHANGE_GROUP_NESTED_ADDED and CHANGE_GROUP_NESTED_REMOVED.
  nested = ndb.StringProperty(repeated=True)


def diff_groups(target, old, new):
  # Helper to reduce amount of typing.
  change = lambda tp, **kwargs: AuthDBGroupChange(
      change_type=getattr(AuthDBChange, 'CHANGE_GROUP_%s' % tp),
      target=target,
      **kwargs)

  # A group was removed. Don't trust 'old' since it may be nil for old apps that
  # did not keep history. Use "last known state" snapshot in 'new' instead.
  if new.auth_db_deleted:
    if new.members:
      yield change('MEMBERS_REMOVED', members=new.members)
    if new.globs:
      yield change('GLOBS_REMOVED', globs=new.globs)
    if new.nested:
      yield change('NESTED_REMOVED', nested=new.nested)
    yield change(
        'DELETED',
        old_description=new.description,
        old_owners=new.owners or model.ADMIN_GROUP)
    return

  # A group was just added (or at least it's first its appearance in the log).
  if old is None:
    yield change('CREATED', description=new.description, owners=new.owners)
    if new.members:
      yield change('MEMBERS_ADDED', members=new.members)
    if new.globs:
      yield change('GLOBS_ADDED', globs=new.globs)
    if new.nested:
      yield change('NESTED_ADDED', nested=new.nested)
    return

  if old.description != new.description:
    yield change(
        'DESCRIPTION_CHANGED',
        description=new.description,
        old_description=old.description)

  # Old entities have no 'owners' field, they are implicitly owned by admins.
  old_owners = old.owners or model.ADMIN_GROUP
  if old_owners != new.owners:
    yield change('OWNERS_CHANGED', owners=new.owners, old_owners=old_owners)

  added, removed = diff_lists(old.members, new.members)
  if added:
    yield change('MEMBERS_ADDED', members=added)
  if removed:
    yield change('MEMBERS_REMOVED', members=removed)

  added, removed = diff_lists(old.globs, new.globs)
  if added:
    yield change('GLOBS_ADDED', globs=added)
  if removed:
    yield change('GLOBS_REMOVED', globs=removed)

  added, removed = diff_lists(old.nested, new.nested)
  if added:
    yield change('NESTED_ADDED', nested=added)
  if removed:
    yield change('NESTED_REMOVED', nested=removed)


## AuthIPWhitelist changes.


class AuthDBIPWhitelistChange(AuthDBChange):
  # Valid for CHANGE_IPWL_CREATED and CHANGE_IPWL_DESCRIPTION_CHANGED.
  description = ndb.TextProperty()
  # Valid for CHANGE_IPWL_DESCRIPTION_CHANGED, CHANGE_IPWL_DELETED.
  old_description = ndb.TextProperty()
  # Valid for CHANGE_IPWL_SUBNETS_ADDED and CHANGE_IPWL_SUBNETS_REMOVED.
  subnets = ndb.StringProperty(repeated=True)


def diff_ip_whitelists(target, old, new):
  # Helper to reduce amount of typing.
  change = lambda tp, **kwargs: AuthDBIPWhitelistChange(
      change_type=getattr(AuthDBChange, 'CHANGE_IPWL_%s' % tp),
      target=target,
      **kwargs)

  # An IP whitelist was removed. Don't trust 'old' since it may be nil for old
  # apps that did not keep history. Use "last known state" snapshot in 'new'.
  if new.auth_db_deleted:
    if new.subnets:
      yield change('SUBNETS_REMOVED', subnets=new.subnets)
    yield change('DELETED', old_description=new.description)
    return

  # An IP whitelist was just added (or it's first its appearance in the log).
  if old is None:
    yield change('CREATED', description=new.description)
    if new.subnets:
      yield change('SUBNETS_ADDED', subnets=new.subnets)
    return

  if old.description != new.description:
    yield change(
        'DESCRIPTION_CHANGED',
        description=new.description,
        old_description=old.description)

  added, removed = diff_lists(old.subnets, new.subnets)
  if added:
    yield change('SUBNETS_ADDED', subnets=added)
  if removed:
    yield change('SUBNETS_REMOVED', subnets=removed)


## AuthIPWhitelistAssignments changes.


class AuthDBIPWhitelistAssignmentChange(AuthDBChange):
  # Valid for ..._SET and ..._UNSET.
  identity = model.IdentityProperty()
  # Valid for ..._SET and ..._UNSET.
  ip_whitelist = ndb.StringProperty()


def diff_ip_whitelist_assignments(target, old, new):
  # Helper to reduce amount of typing.
  def change(tp, identity, ip_whitelist, **kwargs):
    # Whitelist assignments are special: individual assignments are defined
    # as LocalStructuredProperties of a single singleton entity. We want changes
    # to refer to individual assignments (keyed by identity name), and so
    # construct change target manually to reflect that.
    return AuthDBIPWhitelistAssignmentChange(
        change_type=getattr(AuthDBChange, 'CHANGE_IPWLASSIGN_%s' % tp),
        target='%s$%s' % (target, identity.to_bytes()),
        identity=identity,
        ip_whitelist=ip_whitelist,
        **kwargs)

  # All assignment were removed. Don't trust 'old' since it may be nil for old
  # apps that did not keep history. Use "last known state" snapshot in 'new'.
  if new.auth_db_deleted:
    for a in new.assignments:
      yield change('UNSET', a.identity, a.ip_whitelist)
    return

  # All assignments were added.
  if old is None:
    for a in new.assignments:
      yield change('SET', a.identity, a.ip_whitelist)
    return

  # Diff two lists of assignment.
  old_by_ident = {a.identity: a for a in old.assignments}
  new_by_ident = {a.identity: a for a in new.assignments}

  # Delete old ones.
  for ident, a in old_by_ident.iteritems():
    if ident not in new_by_ident:
      yield change('UNSET', a.identity, a.ip_whitelist)

  # Add new ones, update existing ones.
  for ident, a in new_by_ident.iteritems():
    old_a = old_by_ident.get(ident)
    if not old_a or a.ip_whitelist != old_a.ip_whitelist:
      yield change('SET', a.identity, a.ip_whitelist)


## AuthGlobalConfig changes.


class AuthDBConfigChange(AuthDBChange):
  # Valid for CHANGE_CONF_OAUTH_CLIENT_CHANGED.
  oauth_client_id = ndb.StringProperty()
  # Valid for CHANGE_CONF_OAUTH_CLIENT_CHANGED.
  oauth_client_secret = ndb.StringProperty()
  # Valid for CHANGE_CONF_CLIENT_IDS_ADDED and CHANGE_CONF_CLIENT_IDS_REMOVED.
  oauth_additional_client_ids = ndb.StringProperty(repeated=True)
  # Valid for CHANGE_CONF_TOKEN_SERVER_URL_CHANGED.
  token_server_url_old = ndb.StringProperty()
  # Valid for CHANGE_CONF_TOKEN_SERVER_URL_CHANGED.
  token_server_url_new = ndb.StringProperty()
  # Valid for CHANGE_CONF_SECURITY_CONFIG_CHANGED.
  security_config_old = ndb.BlobProperty()
  # Valid for CHANGE_CONF_SECURITY_CONFIG_CHANGED.
  security_config_new = ndb.BlobProperty()


def diff_global_config(target, old, new):
  # Helper to reduce amount of typing.
  change = lambda tp, **kwargs: AuthDBConfigChange(
      change_type=getattr(AuthDBChange, 'CHANGE_CONF_%s' % tp),
      target=target,
      **kwargs)

  # AuthGlobalConfig can't be created or deleted. 'old' still can be None for
  # old apps that existed before history related code was added.
  assert not new.auth_db_deleted
  prev_client_id = old.oauth_client_id if old else ''
  prev_client_secret = old.oauth_client_secret if old else ''
  prev_client_ids = old.oauth_additional_client_ids if old else []
  prev_token_server_url = old.token_server_url if old else ''
  prev_security_config = old.security_config if old else None

  if (prev_client_id != new.oauth_client_id or
      prev_client_secret != new.oauth_client_secret):
    yield change(
        'OAUTH_CLIENT_CHANGED',
        oauth_client_id=new.oauth_client_id,
        oauth_client_secret=new.oauth_client_secret)

  added, removed = diff_lists(prev_client_ids, new.oauth_additional_client_ids)
  if added:
    yield change('CLIENT_IDS_ADDED', oauth_additional_client_ids=added)
  if removed:
    yield change('CLIENT_IDS_REMOVED', oauth_additional_client_ids=removed)

  if prev_token_server_url != new.token_server_url:
    yield change(
        'TOKEN_SERVER_URL_CHANGED',
        token_server_url_old=prev_token_server_url,
        token_server_url_new=new.token_server_url)

  if prev_security_config != new.security_config:
    yield change(
        'SECURITY_CONFIG_CHANGED',
        security_config_old=prev_security_config,
        security_config_new=new.security_config)


###


def diff_lists(old, new):
  """Returns sorted lists of added and removed items."""
  old = set(old or [])
  new = set(new or [])
  return sorted(new - old), sorted(old - new)


# Name of *History entity class name => (original class, diffing function).
KNOWN_HISTORICAL_ENTITIES = {
  'AuthGroupHistory': (model.AuthGroup, diff_groups),
  'AuthIPWhitelistHistory': (model.AuthIPWhitelist, diff_ip_whitelists),
  'AuthIPWhitelistAssignmentsHistory': (
      model.AuthIPWhitelistAssignments, diff_ip_whitelist_assignments),
  'AuthGlobalConfigHistory': (model.AuthGlobalConfig, diff_global_config),
}


### Code to query change log.


@utils.cache_with_expiration(expiration_sec=300)
def is_changle_log_indexed():
  """True if required Datastore composite indexes exist.

  May be False for GAE apps that use components.auth, but did not update
  index.yaml. It is fine, mostly, since apps in Standalone mode are discouraged
  and central Auth service has necessary indexes.

  UI of apps without indexes hide "Change log" tab.

  Note: this function spams log with warning "suspended generator
  has_next_async(query.py:1760) raised NeedIndexError(....)". Unfortunately
  these lines are generated in NDB guts and the only way to hide them is to
  modify global NDB state:

    ndb.add_flow_exception(datastore_errors.NeedIndexError)

  Each individual app should decide whether it wants to do it or not.
  """
  try:
    q = make_change_log_query(target='bogus$bogus')
    q.fetch_page(1)
    return True
  except datastore_errors.NeedIndexError:
    return False


def make_change_log_query(target=None, auth_db_rev=None):
  """Returns ndb.Query over AuthDBChange entities."""
  if auth_db_rev:
    ancestor = change_log_revision_key(auth_db_rev)
  else:
    ancestor = change_log_root_key()
  q = AuthDBChange.query(ancestor=ancestor)
  if target:
    if not TARGET_RE.match(target):
      raise ValueError('Invalid target: %r' % target)
    q = q.filter(AuthDBChange.target == target)
  q = q.order(-AuthDBChange.key)
  return q


### Code to snapshot initial state of AuthDB into *History.


class _AuthDBSnapshotMarker(ndb.Model):
  # AuthDB rev of the snapshot.
  auth_db_rev = ndb.IntegerProperty(indexed=False)

  @staticmethod
  def marker_key():
    """Returns ndb.Key of entity that exists only if initial snapshot was done.

    Bump key ID to redo the snapshot.
    """
    return ndb.Key(_AuthDBSnapshotMarker, 1, parent=model.root_key())


def ensure_initial_snapshot(auth_db_rev):
  """Makes sure all current AuthDB entities are represented in the history.

  It's important only for applications that existed before change log
  functionality was added.

  It generates a new AuthDB revision by "touching" all existing entities. That
  way we reuse logic of generating *History entities already present in
  model.py. Note that original entities will also be updated ('auth_db_rev'
  property is modified), so it's indeed a true new AuthDB revision.
  """
  # Already done?
  key = _AuthDBSnapshotMarker.marker_key()
  if key.get() is not None:
    return

  # Is it a fresh application that had change log from the very beginning?
  # No need to snapshot existing groups (they all end up in the history by usual
  # means).
  if auth_db_rev == 1:
    _AuthDBSnapshotMarker(key=key, auth_db_rev=1).put()
    return

  @ndb.transactional
  def touch_auth_db():
    # Recheck under transaction.
    if key.get() is not None:
      return
    to_process = []

    # Start slow queries in parallel.
    groups_future = model.AuthGroup.query(
        ancestor=model.root_key()).fetch_async()
    whitelists_future = model.AuthIPWhitelist.query(
        ancestor=model.root_key()).fetch_async()

    # Singleton entities.
    to_process.append(model.root_key().get())
    to_process.append(model.ip_whitelist_assignments_key().get())

    # Finish queries.
    to_process.extend(groups_future.get_result())
    to_process.extend(whitelists_future.get_result())

    # Update auth_db_rev properties, make *History entities. Keep modified_by
    # and modified_ts as they were.
    to_put = []
    for ent in to_process:
      if not ent:
        continue
      ent.record_revision(
          modified_by=ent.modified_by,
          modified_ts=ent.modified_ts,
          comment='Initial snapshot')
      to_put.append(ent)

    # Store changes, update the marker to make sure this won't run again.
    ndb.put_multi(to_put)
    auth_db_rev = model.replicate_auth_db()
    _AuthDBSnapshotMarker(key=key, auth_db_rev=auth_db_rev).put()

  logging.info('Snapshotting all existing AuthDB entities for history')
  touch_auth_db()


### Task queue plumbing.


@model.commit_callback
def on_auth_db_change(auth_db_rev):
  """Called in a transaction that updated AuthDB."""
  # Avoid adding task queues in unit tests, since there are many-many unit tests
  # (in multiple project and repos) that indirectly make AuthDB transactions
  # and mocking out 'enqueue_process_change_task' in all of them is stupid
  # unscalable work. So be evil and detect unit tests right here.
  if not utils.is_unit_test():
    enqueue_process_change_task(auth_db_rev)


def enqueue_process_change_task(auth_db_rev):
  """Transactionally adds a call to 'process_change' to the task queue.

  Pins the task to currently executing version of BACKEND_MODULE module
  (defined in config.py).

  Added as AuthDB commit callback in get_backend_routes() below.
  """
  assert ndb.in_transaction()
  conf = config.ensure_configured()
  try:
    # Pin the task to the module and version.
    taskqueue.add(
        url='/internal/auth/taskqueue/process-change/%d' % auth_db_rev,
        queue_name=conf.PROCESS_CHANGE_TASK_QUEUE,
        headers={'Host': modules.get_hostname(module=conf.BACKEND_MODULE)},
        transactional=True)
  except Exception as e:
    logging.error(
        'Problem adding "process-change" task to the task queue (%s): %s',
        e.__class__.__name__, e)
    raise


class InternalProcessChangeHandler(webapp2.RequestHandler):
  def post(self, auth_db_rev):
    # We don't know task queue name during module loading time, so delay
    # decorator application until the actual call.
    queue_name = config.ensure_configured().PROCESS_CHANGE_TASK_QUEUE
    @decorators.require_taskqueue(queue_name)
    def call_me(_self):
      process_change(int(auth_db_rev))
    call_me(self)


def get_backend_routes():
  """Returns a list of routes with task queue handlers.

  Used from ui/app.py (since it's where WSGI module is defined) and directly
  from auth_service backend module.
  """
  return [
    webapp2.Route(
        r'/internal/auth/taskqueue/process-change/<auth_db_rev:\d+>',
        InternalProcessChangeHandler),
  ]
