# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""NDB model classes used to model AuthDB relations.

Overview
--------

Models defined here are used by central authentication service (that stores all
groups and secrets) and by services that implement some concrete functionality
protected with ACLs (like isolate and swarming services).

Applications that use auth component may work in 3 modes:
  1. Standalone. Application is self contained and manages its own groups.
     Useful when developing a new service or for simple installations.
  2. Replica. Application uses a central authentication service. An application
     can be dynamically switched from Standalone to Replica mode.
  3. Primary. Application IS a central authentication service. Only 'auth'
     service is running in this mode. 'configure_as_primary' call during startup
     switches application to that mode.

Central authentication service (Primary) holds authoritative copy of all auth
related information (groups, secrets, etc.) and acts as a single source of truth
for it. All other services (Replicas) hold copies of a relevant subset of
this information (that they use to perform authorization checks).

Primary service is responsible for updating replicas' configuration via
service-to-service push based replication protocol.

AuthDB holds a list of groups. Each group has a unique name and is defined
as union of 3 sets:
  1) Explicit enumeration of particular Identities e.g. 'user:alice@example.com'
  2) Set of glob-like identity patterns e.g. 'user:*@example.com'
  3) Set of nested Groups.

Identity defines an actor making an action (it can be a real person, a bot,
an AppEngine application or special 'anonymous' identity).

In addition to that, AuthDB stores small amount of authentication related
configuration data, such as OAuth2 client_id and client_secret and various
secret keys.

Audit trail
-----------

Each change to AuthDB has an associated revision number (that monotonically
increases with each change). All entities modified by a change are copied to
append-only log under an entity key associated with the revision (see
historical_revision_key below). Removals are marked by special auth_db_deleted
flag in entites in the log. This is enough to recover a snapshot of all groups
at some specific moment in time, or to produce a diff between two revisions.

Note that entities in the historical log are not used by online queries. At any
moment in time most recent version of an AuthDB entity exists in two copies:
  1) Main copy used for online queries. It is mutated in-place with each change.
  2) Most recent record in the historical log. Read only.

To reduce a possibility of misuse of historical copies in online transactions,
history log entity classes are suffixied with 'History' suffix. They also have
all indexes stripped.

This mechanism is enabled only on services in Standalone or Primary mode.
Replicas do not keep track of AuthDB revisions and do not keep any historical
log.
"""

import collections
import logging
import os
import re

from google.appengine.api import app_identity
from google.appengine.ext import ndb

from components import datastore_utils
from components import utils

from . import globmatch
from . import ipaddr

# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'ADMIN_GROUP',
  'Anonymous',
  'bootstrap_group',
  'bootstrap_ip_whitelist',
  'bootstrap_loopback_ips',
  'bots_ip_whitelist',
  'configure_as_primary',
  'find_group_dependency_cycle',
  'find_referencing_groups',
  'get_auth_db_revision',
  'get_missing_groups',
  'get_service_self_identity',
  'group_key',
  'Identity',
  'IDENTITY_ANONYMOUS',
  'IDENTITY_BOT',
  'IDENTITY_PROJECT',
  'IDENTITY_SERVICE',
  'IDENTITY_USER',
  'IdentityGlob',
  'IdentityProperty',
  'ip_whitelist_key',
  'IP_WHITELISTED_BOT_ID',
  'is_empty_group',
  'is_external_group_name',
  'is_primary',
  'is_replica',
  'is_standalone',
  'is_valid_group_name',
  'is_valid_ip_whitelist_name',
  'replicate_auth_db',
]


# Name of a group whose members have access to Group management UI. It's the
# only group needed to bootstrap everything else.
ADMIN_GROUP = 'administrators'

# No identity information is provided. Identity name is always 'anonymous'.
IDENTITY_ANONYMOUS = 'anonymous'
# Using bot credentials. Used primary by Swarming. Identity encodes bot's id.
IDENTITY_BOT = 'bot'
# Using X-Luci-Project header in an internal RPC. Identity name is project name.
IDENTITY_PROJECT = 'project'
# Using App Engine X-Appengine-Inbound-Appid header. Identity name is app name.
IDENTITY_SERVICE = 'service'
# Using user credentials (cookies or access tokens). Identity name is email.
IDENTITY_USER = 'user'


# All allowed identity kinds + regexps to validate identity name.
ALLOWED_IDENTITY_KINDS = {
  IDENTITY_ANONYMOUS: re.compile(r'^anonymous$'),
  IDENTITY_BOT: re.compile(r'^[0-9a-zA-Z_\-\.@]+$'),
  # See also PROJECT_ID_RGX in components/config/common.py.
  IDENTITY_PROJECT: re.compile(r'^[a-z0-9\-_]+$'),
  IDENTITY_SERVICE: re.compile(r'^[0-9a-zA-Z_\-\:\.]+$'),
  IDENTITY_USER: re.compile(r'^[0-9a-zA-Z_\-\.\+\%]+@[0-9a-zA-Z_\-\.]+$'),
}

# Regular expression that matches group names. ASCII only, no leading or
# trailing spaces allowed (spaces inside are fine).
GROUP_NAME_RE = re.compile(
    r'^([a-z\-]+/)?[0-9a-zA-Z_][0-9a-zA-Z_\-\.\ @]{1,80}[0-9a-zA-Z_\-\.]$')
# Special group name that means 'All possible users' (including anonymous!).
GROUP_ALL = '*'

# Regular expression for IP whitelist name.
IP_WHITELIST_NAME_RE = re.compile(r'^[0-9a-zA-Z_\-\+\.\ ]{2,200}$')


# Configuration of Primary service, set by 'configure_as_primary'.
_replication_callback = None


# Root ndb keys of various models. They can't be defined as a module level
# constants because ndb.Key implicitly includes current APPLICATION_ID. And in
# testing environment it is '_' during module loading time. Trying to use such
# key from within a testbed test case results in the following error:
# BadRequestError: app "testbed-test" cannot access app "_"'s data


def root_key():
  """Global root key of auth models entity group."""
  return ndb.Key('AuthGlobalConfig', 'root')


def replication_state_key():
  """Key of AuthReplicationState entity."""
  return ndb.Key('AuthReplicationState', 'self', parent=root_key())


def ip_whitelist_assignments_key():
  """Key of AuthIPWhitelistAssignments entity."""
  return ndb.Key('AuthIPWhitelistAssignments', 'default', parent=root_key())


def historical_revision_key(auth_db_rev):
  """Key for entity subgroup that holds changes done in a concrete revision."""
  return ndb.Key('Rev', auth_db_rev, parent=root_key())


################################################################################
## Identity & IdentityGlob.


class Identity(
    datastore_utils.BytesSerializable,
    collections.namedtuple('Identity', 'kind, name')):
  """Represents a caller that makes requests. Immutable.

  A tuple of (kind, name) where 'kind' is one of IDENTITY_* constants and
  meaning of 'name' depends on a kind (see comments for IDENTITY_*).
  It generalizes accounts of real people, bot accounts and service-to-service
  accounts.

  It's a pure identity information. Any additional information that may be
  related to an identity (e.g. registration date, last access time, etc.) should
  be stored elsewhere using Identity.to_bytes() as a key.
  """

  # Inheriting from tuple requires use of __new__ instead of __init__. __init__
  # is called with object already 'frozen', so it's not possible to modify its
  # attributes in __init__.
  # See http://docs.python.org/2/reference/datamodel.html#object.__new__
  def __new__(cls, kind, name):
    if isinstance(name, unicode):
      try:
        name = name.encode('ascii')
      except UnicodeEncodeError:
        raise ValueError('Identity has invalid format: only ASCII is allowed')
    if (kind not in ALLOWED_IDENTITY_KINDS or
        not ALLOWED_IDENTITY_KINDS[kind].match(name)):
      raise ValueError('Identity has invalid format: %s' % name)
    return super(Identity, cls).__new__(cls, str(kind), name)

  def to_bytes(self):
    """Serializes this identity to byte buffer."""
    return '%s:%s' % (self.kind, self.name)

  @classmethod
  def from_bytes(cls, byte_buf):
    """Given a byte buffer returns corresponding Identity object."""
    kind, sep, name = byte_buf.partition(':')
    if not sep:
      raise ValueError('Missing \':\' separator in Identity string')
    return cls(kind, name)

  @property
  def is_anonymous(self):
    """True if this object represents anonymous identity."""
    return self.kind == IDENTITY_ANONYMOUS

  @property
  def is_bot(self):
    """True if this object represents bot account."""
    return self.kind == IDENTITY_BOT

  @property
  def is_project(self):
    """True if this object represents a LUCI project."""
    return self.kind == IDENTITY_PROJECT

  @property
  def is_service(self):
    """True if this object represents an appengine app."""
    return self.kind == IDENTITY_SERVICE

  @property
  def is_user(self):
    """True if this object represents user account."""
    return self.kind == IDENTITY_USER


# Predefined Anonymous identity.
Anonymous = Identity(IDENTITY_ANONYMOUS, 'anonymous')


# Identity assigned to callers that make unauthenticated calls from IPs
# belonging to '<appid>-bots' IP whitelist. Note that same bot may appear to use
# different IP addresses (happens with some NATs), thus we can't put IP
# address into the bot identity string and instead hardcode some arbitrary
# name (defined here).
#
# TODO(vadimsh): Get rid of this. Blocked on Swarming and Isolate switching
# to service accounts.
IP_WHITELISTED_BOT_ID = Identity(IDENTITY_BOT, 'whitelisted-ip')


class IdentityProperty(datastore_utils.BytesSerializableProperty):
  """NDB model property for Identity values.

  Identities are stored as indexed short blobs internally.
  """
  _value_type = Identity
  _indexed = True


class IdentityGlob(
    datastore_utils.BytesSerializable,
    collections.namedtuple('IdentityGlob', 'kind, pattern')):
  """Glob-like pattern that matches subset of identities. Immutable.

  Tuple (kind, glob) where 'kind' is is one of IDENTITY_* constants and 'glob'
  defines pattern that identity names' should match. For example, IdentityGlob
  that matches all bots is (IDENTITY_BOT, '*') which is also can be written
  as 'bot:*'.

  The pattern language only supports '*' currently.
  """

  # See comment for Identity.__new__ regarding use of __new__ here.
  def __new__(cls, kind, pattern):
    if isinstance(pattern, unicode):
      try:
        pattern = pattern.encode('ascii')
      except UnicodeEncodeError:
        raise ValueError('Invalid IdentityGlob pattern: only ASCII is allowed')
    if not pattern:
      raise ValueError('No pattern is given')
    if '\n' in pattern:
      raise ValueError('Multi-line patterns are not allowed')
    if kind not in ALLOWED_IDENTITY_KINDS:
      raise ValueError('Invalid Identity kind: %s' % kind)
    return super(IdentityGlob, cls).__new__(cls, str(kind), pattern)

  def to_bytes(self):
    """Serializes this identity glob to byte buffer."""
    return '%s:%s' % (self.kind, self.pattern)

  @classmethod
  def from_bytes(cls, byte_buf):
    """Given a byte buffer returns corresponding IdentityGlob object."""
    kind, sep, pattern = byte_buf.partition(':')
    if not sep:
      raise ValueError('Missing \':\' separator in IdentityGlob string')
    return cls(kind, pattern)

  def match(self, identity):
    """Return True if |identity| matches this pattern."""
    if identity.kind != self.kind:
      return False
    return globmatch.match(identity.name, self.pattern)


class IdentityGlobProperty(datastore_utils.BytesSerializableProperty):
  """NDB model property for IdentityGlob values.

  IdentityGlobs are stored as short indexed blobs internally.
  """
  _value_type = IdentityGlob
  _indexed = True


################################################################################
## Singleton entities and replication related models.


def configure_as_primary(replication_callback):
  """Registers a callback to be called when AuthDB changes.

  Should be called during Primary application startup. The callback will be
  called as 'replication_callback(AuthReplicationState)' from inside transaction
  on root_key() entity group whenever replicate_auth_db() is called (i.e. on
  every change to auth db that should be replication to replicas).
  """
  global _replication_callback
  _replication_callback = replication_callback


def is_primary():
  """Returns True if current application was configured as Primary."""
  return bool(_replication_callback)


def is_replica():
  """Returns True if application is in Replica mode."""
  return not is_primary() and not is_standalone()


def is_standalone():
  """Returns True if application is in Standalone mode."""
  ent = get_replication_state()
  return not ent or not ent.primary_id


def get_replication_state():
  """Returns AuthReplicationState singleton entity if it exists."""
  return replication_state_key().get()


def get_auth_db_revision():
  """Returns current revision of AuthDB, it increases with each change."""
  state = get_replication_state()
  return state.auth_db_rev if state else 0


def get_service_self_identity():
  """Returns Identity that correspond to the current GAE app itself."""
  return Identity(IDENTITY_SERVICE, app_identity.get_application_id())


class AuthVersionedEntityMixin(object):
  """Mixin class for entities that keep track of when they change.

  Entities that have this mixin are supposed to be updated in get()\put() or
  get()\delete() transactions. Caller must call record_revision(...) sometime
  during the transaction (but before put()). Similarly a call to
  record_deletion(...) is expected sometime before delete().

  replicate_auth_db will store a copy of the entity in the revision log when
  committing a transaction.

  A pair of properties auth_db_rev and auth_db_prev_rev are used to implement
  a linked list of versions of this entity (e.g. one can take most recent entity
  version and go back in time by following auth_db_prev_rev links).
  """
  # When the entity was modified last time. Do not use 'auto_now' property since
  # such property overrides any explicitly set value with now() during put. It's
  # undesired when storing a copy of entity received from Primary (Replica
  # should have modified_ts to be same as on Primary).
  modified_ts = ndb.DateTimeProperty()
  # Who modified the entity last time.
  modified_by = IdentityProperty()

  # Revision of Auth DB at which this entity was updated last time.
  auth_db_rev = ndb.IntegerProperty()
  # Revision of Auth DB of previous version of this entity or None.
  auth_db_prev_rev = ndb.IntegerProperty()

  def record_revision(self, modified_by, modified_ts=None, comment=None):
    """Updates the entity to record Auth DB revision of the current transaction.

    Stages the entity to be copied to historical log.

    Must be called sometime before 'put' (not necessary right before it). Note
    that NDB hooks are not used because they are buggy. See docstring for
    replicate_auth_db for more info.

    Args:
      modified_by: Identity that made the change.
      modified_ts: datetime when the change was made (or None for current time).
      comment: optional comment to put in the revision log.
    """
    _get_pending_auth_db_transaction().record_change(
        entity=self,
        deletion=False,
        modified_by=modified_by,
        modified_ts=modified_ts or utils.utcnow(),
        comment=comment)

  def record_deletion(self, modified_by, modified_ts=None, comment=None):
    """Marks entity as being deleted in the current transaction.

    Stages the entity to be copied to historical log (with 'auth_db_deleted'
    flag set). The entity must not be mutated between 'get' and AuthDB commit.

    Must be called sometime before 'delete' (not necessary right before it).
    Note that NDB hooks are not used because they are buggy. See docstring for
    replicate_auth_db for more info.

    Args:
      modified_by: Identity that made the change.
      modified_ts: datetime when the change was made (or None for current time).
      comment: optional comment to put in the revision log.
    """
    _get_pending_auth_db_transaction().record_change(
        entity=self,
        deletion=True,
        modified_by=modified_by,
        modified_ts=modified_ts or utils.utcnow(),
        comment=comment)

  ## Internal interface. Do not use directly unless you know what you are doing.

  @classmethod
  def get_historical_copy_class(cls):
    """Returns entity class for historical copies of original entity.

    Has all the same properties, but unindexed (not needed), unvalidated
    (original entity is already validated) and not cached.

    The name of the new entity class is "<original name>History" (to make sure
    it doesn't show up in indexes for original entity class).
    """
    existing = getattr(cls, '_auth_db_historical_copy_cls', None)
    if existing:
      return existing
    props = {}
    for name, prop in cls._properties.iteritems():
      # Whitelist supported property classes. Better to fail loudly when
      # encountering something new, rather than silently produce (possibly)
      # incorrect result. Note that all AuthDB classes are instantiated in
      # unit tests, so there should be no unexpected asserts in production.
      assert prop.__class__ in (
        IdentityGlobProperty,
        IdentityProperty,
        ndb.BlobProperty,
        ndb.BooleanProperty,
        ndb.DateTimeProperty,
        ndb.IntegerProperty,
        ndb.LocalStructuredProperty,
        ndb.StringProperty,
        ndb.TextProperty,
      ), prop.__class__
      kwargs = {
        'name': prop._name,
        'indexed': False,
        'required': False,
        'repeated': prop._repeated,
      }
      if prop.__class__ == ndb.LocalStructuredProperty:
        kwargs['modelclass'] = prop._modelclass
      props[name] = prop.__class__(**kwargs)
    new_cls = type(
        '%sHistory' % cls.__name__, (_AuthDBHistoricalEntity,), props)
    cls._auth_db_historical_copy_cls = new_cls
    return new_cls

  def make_historical_copy(self, deleted, comment):
    """Returns an entity to put in the historical log.

    It's a copy of the original entity, but stored under another key and with
    indexes removed. It also has a bunch of additional properties (defined
    in _AuthDBHistoricalEntity). See 'get_historical_copy_class'.

    The key is derived from auth_db_rev and class and ID of the original entity.
    For example, AuthGroup "admins" modified at rev 123 will be copied to
    the history as ('AuthGlobalConfig', 'root', 'Rev', 123, 'AuthGroupHistory',
    'admins'), where the key prefix (first two pairs) is obtained with
    historical_revision_key(...).
    """
    assert self.key.parent() == root_key() or self.key == root_key(), self.key
    cls = self.get_historical_copy_class()
    entity = cls(
        id=self.key.id(),
        parent=historical_revision_key(self.auth_db_rev))
    for prop in self._properties:
      setattr(entity, prop, getattr(self, prop))
    entity.auth_db_deleted = deleted
    entity.auth_db_change_comment = comment
    entity.auth_db_app_version = utils.get_app_version()
    return entity


class AuthGlobalConfig(ndb.Model, AuthVersionedEntityMixin):
  """Acts as a root entity for auth models.

  There should be only one instance of this model in Datastore, with a key set
  to root_key(). A change to an entity group rooted at this key is a signal that
  AuthDB has to be refetched (see 'fetch_auth_db' in api.py).

  Entities that change often or associated with particular bot or user
  MUST NOT be in this entity group.

  Content of this particular entity is replicated from Primary service to all
  Replicas.

  Entities that belong to this entity group are:
   * AuthGroup
   * AuthIPWhitelist
   * AuthIPWhitelistAssignments
   * AuthReplicationState
   * AuthSecret
  """
  # Disable useless in-process per-request cache.
  _use_cache = False

  # OAuth2 client_id to use to mint new OAuth2 tokens.
  oauth_client_id = ndb.StringProperty(indexed=False, default='')
  # OAuth2 client secret. Not so secret really, since it's passed to clients.
  oauth_client_secret = ndb.StringProperty(indexed=False, default='')
  # Additional OAuth2 client_ids allowed to access the services.
  oauth_additional_client_ids = ndb.StringProperty(repeated=True, indexed=False)

  # URL of a token server to use to generate delegation tokens.
  token_server_url = ndb.StringProperty(indexed=False, default='')
  # Serialized security_config_pb2.SecurityConfig, see security_config.proto.
  security_config = ndb.BlobProperty()


class AuthReplicationState(ndb.Model, datastore_utils.SerializableModelMixin):
  """Contains state used to control Primary -> Replica replication.

  It's a singleton entity with key replication_state_key() (in same entity
  groups as root_key()). This entity should be small since it is updated
  (auth_db_rev is incremented) whenever AuthDB changes.

  Exists in any AuthDB (on Primary and Replicas). Primary updates it whenever
  changes to AuthDB are made, Replica updates it whenever it receives a push
  from Primary.
  """
  # Disable useless in-process per-request cache.
  _use_cache = False

  # How to convert this entity to or from serializable dict.
  serializable_properties = {
    'primary_id': datastore_utils.READABLE,
    'primary_url': datastore_utils.READABLE,
    'auth_db_rev': datastore_utils.READABLE,
    'modified_ts': datastore_utils.READABLE,
  }

  # For services in Standalone mode it is None.
  # For services in Primary mode: own GAE application ID.
  # For services in Replica mode it is a GAE application ID of Primary.
  primary_id = ndb.StringProperty(indexed=False)

  # For services in Replica mode, root URL of Primary, i.e https://<host>.
  primary_url = ndb.StringProperty(indexed=False)

  # Revision of auth DB. Increased by 1 with every change that should be
  # propagate to replicas. Only services in Standalone or Primary mode
  # update this property by themselves. Replicas receive it from Primary.
  auth_db_rev = ndb.IntegerProperty(default=0, indexed=False)

  # Time when auth_db_rev was created (by Primary clock). For informational
  # purposes only. See comment at AuthGroup.modified_ts for explanation why
  # auto_now is not used.
  modified_ts = ndb.DateTimeProperty(auto_now_add=True, indexed=False)


def replicate_auth_db():
  """Increments auth_db_rev, updates historical log, triggers replication.

  Must be called once from inside a transaction (right before exiting it).

  Should only be called for services in Standalone or Primary modes. Will raise
  ValueError if called on Replica. When called for service in Standalone mode,
  will update auth_db_rev but won't kick any replication. For services in
  Primary mode will also initiate replication by calling callback set in
  'configure_as_primary'. The callback usually transactionally enqueues a task
  (to gracefully handle transaction rollbacks).

  WARNING: This function relies on a valid transaction context. NDB hooks and
  asynchronous operations are known to be buggy in this regard: NDB hook for
  an async operation in a transaction may be called with a wrong context
  (main event loop context instead of transaction context). One way to work
  around that is to monkey patch NDB (as done here: https://goo.gl/1yASjL).
  Another is to not use hooks at all. There's no way to differentiate between
  sync and async modes of an NDB operation from inside a hook. And without a
  strict assert it's very easy to forget about "Do not use put_async" warning.
  For that reason _post_put_hook is NOT used and replicate_auth_db() should be
  called explicitly whenever relevant part of root_key() entity group is
  updated.

  Returns:
    New AuthDB revision number.
  """
  assert ndb.in_transaction()
  txn = _get_pending_auth_db_transaction()
  txn.commit()
  if is_primary():
    _replication_callback(txn.replication_state)
  return txn.replication_state.auth_db_rev


################################################################################
## Auth DB transaction details (used for historical log of changes).


_commit_callbacks = []


def commit_callback(cb):
  """Adds a callback that's called before AuthDB transaction is committed.

  Can be used as decorator. Adding a callback second time is noop.

  Args:
    cb: function that takes single auth_db_rev argument as input.
  """
  if cb not in _commit_callbacks:
    _commit_callbacks.append(cb)
  return cb


def _get_pending_auth_db_transaction():
  """Used internally to keep track of changes done in the transaction.

  Returns:
    Instance of _AuthDBTransaction (stored in the transaction context).
  """
  # Use transaction context to store the object. Note that each transaction
  # retry gets its own new transaction context which is what we need,
  # see ndb/context.py, 'transaction' tasklet, around line 982 (for SDK 1.9.6).
  assert ndb.in_transaction()
  ctx = ndb.get_context()
  txn = getattr(ctx, '_auth_db_transaction', None)
  if txn:
    return txn

  # Prepare next AuthReplicationState (auth_db_rev +1).
  state = replication_state_key().get()
  if not state:
    primary_id = app_identity.get_application_id() if is_primary() else None
    state = AuthReplicationState(
        key=replication_state_key(),
        primary_id=primary_id,
        auth_db_rev=0)
  # Assert Primary or Standalone. Replicas can't increment auth db revision.
  if not is_primary() and state.primary_id:
    raise ValueError('Can\'t modify Auth DB on Replica')
  state.auth_db_rev += 1
  state.modified_ts = utils.utcnow()

  # Store the state in the transaction context. Used in replicate_auth_db(...)
  # later.
  txn = _AuthDBTransaction(state)
  ctx._auth_db_transaction = txn
  return txn


class _AuthDBTransaction(object):
  """Keeps track of entities updated or removed in current transaction."""

  _Change = collections.namedtuple('_Change', 'entity deletion comment')

  def __init__(self, replication_state):
    self.replication_state = replication_state
    self.changes = [] # list of _Change tuples
    self.committed = False

  def record_change(self, entity, deletion, modified_by, modified_ts, comment):
    assert not self.committed
    assert isinstance(entity, AuthVersionedEntityMixin)
    assert all(entity.key != c.entity.key for c in self.changes)

    # Mutate the main entity (the one used to serve online requests).
    entity.modified_by = modified_by
    entity.modified_ts = modified_ts
    entity.auth_db_prev_rev = entity.auth_db_rev # can be None for new entities
    entity.auth_db_rev = self.replication_state.auth_db_rev

    # Keep a historical copy. Delay make_historical_copy call until the commit.
    # Here (in 'record_change') entity may not have all the fields updated yet.
    self.changes.append(self._Change(entity, deletion, comment))

  def commit(self):
    assert not self.committed
    puts = [
      c.entity.make_historical_copy(c.deletion, c.comment)
      for c in self.changes
    ]
    ndb.put_multi(puts + [self.replication_state])
    for cb in _commit_callbacks:
      cb(self.replication_state.auth_db_rev)
    self.committed = True


class _AuthDBHistoricalEntity(ndb.Model):
  """Base class for *History magic class in AuthVersionedEntityMixin.

  In addition to properties defined here the child classes (*History) also
  always inherit (for some definition of "inherit") properties from
  AuthVersionedEntityMixin.

  See get_historical_copy_class().
  """
  # Historical entities are not intended to be read often, and updating the
  # cache will make AuthDB transactions only slower.
  _use_cache = False
  _use_memcache = False

  # True if entity was deleted in the given revision.
  auth_db_deleted = ndb.BooleanProperty(indexed=False)
  # Comment string passed to record_revision or record_deletion.
  auth_db_change_comment = ndb.StringProperty(indexed=False)
  # A GAE module version that committed the change.
  auth_db_app_version = ndb.StringProperty(indexed=False)

  def get_previous_historical_copy_key(self):
    """Returns ndb.Key of *History entity matching auth_db_prev_rev revision."""
    if self.auth_db_prev_rev is None:
      return None
    return ndb.Key(
        self.__class__, self.key.id(),
        parent=historical_revision_key(self.auth_db_prev_rev))


################################################################################
## Groups.


class AuthGroup(
    ndb.Model,
    AuthVersionedEntityMixin,
    datastore_utils.SerializableModelMixin):
  """A group of identities, entity id is a group name.

  Parent is AuthGlobalConfig entity keyed at root_key().

  Primary service holds authoritative list of Groups, that gets replicated to
  all Replicas.
  """
  # Disable useless in-process per-request cache.
  _use_cache = False

  # How to convert this entity to or from serializable dict.
  serializable_properties = {
    'members': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'globs': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'nested': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'description': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'owners': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'created_ts': datastore_utils.READABLE,
    'created_by': datastore_utils.READABLE,
    'modified_ts': datastore_utils.READABLE,
    'modified_by': datastore_utils.READABLE,
  }

  # List of members that are explicitly in this group. Indexed.
  members = IdentityProperty(repeated=True)
  # List of identity-glob expressions (like 'user:*@example.com'). Indexed.
  globs = IdentityGlobProperty(repeated=True)
  # List of nested group names. Indexed.
  nested = ndb.StringProperty(repeated=True)

  # Human readable description.
  description = ndb.TextProperty(default='')
  # A name of the group that can modify or delete this group.
  owners = ndb.StringProperty(default=ADMIN_GROUP)

  # When the group was created.
  created_ts = ndb.DateTimeProperty()
  # Who created the group.
  created_by = IdentityProperty()


def group_key(group):
  """Returns ndb.Key for AuthGroup entity."""
  return ndb.Key(AuthGroup, group, parent=root_key())


def is_empty_group(group):
  """Returns True if group is missing or completely empty."""
  group = group_key(group).get()
  return not group or not(group.members or group.globs or group.nested)


def is_valid_group_name(name):
  """True if string looks like a valid group name."""
  return bool(GROUP_NAME_RE.match(name))


def is_external_group_name(name):
  """True if group is imported from outside and is not writable."""
  return is_valid_group_name(name) and '/' in name


@ndb.transactional
def bootstrap_group(group, identities, description=''):
  """Makes a group (if not yet exists) and adds |identities| to it as members.

  Returns True if modified the group, False if identities are already there.
  """
  key = group_key(group)
  entity = key.get()
  if entity and all(i in entity.members for i in identities):
    return False
  now = utils.utcnow()
  if not entity:
    entity = AuthGroup(
        key=key,
        description=description,
        created_ts=now,
        created_by=get_service_self_identity())
  for i in identities:
    if i not in entity.members:
      entity.members.append(i)
  entity.record_revision(
      modified_by=get_service_self_identity(),
      modified_ts=now,
      comment='Bootstrap')
  entity.put()
  replicate_auth_db()
  return True


def find_referencing_groups(group):
  """Finds groups that reference the specified group as nested group or owner.

  Used to verify that |group| is safe to delete, i.e. no other group is
  depending on it.

  Returns:
    Set of names of referencing groups.
  """
  nesting_groups = AuthGroup.query(
      AuthGroup.nested == group,
      ancestor=root_key()).fetch_async(keys_only=True)
  owned_groups = AuthGroup.query(
      AuthGroup.owners == group,
      ancestor=root_key()).fetch_async(keys_only=True)
  refs = set()
  refs.update(key.id() for key in nesting_groups.get_result())
  refs.update(key.id() for key in owned_groups.get_result())
  return refs


def get_missing_groups(groups):
  """Given a list of group names, returns a list of groups that do not exist."""
  # We need to iterate over |groups| twice. It won't work if |groups|
  # is a generator. So convert to list first.
  groups = list(groups)
  entities = ndb.get_multi(group_key(name) for name in groups)
  return [name for name, ent in zip(groups, entities) if not ent]


def find_group_dependency_cycle(group):
  """Searches for dependency cycle between nested groups.

  Traverses the dependency graph starting from |group|, fetching all necessary
  groups from datastore along the way.

  Args:
    group: instance of AuthGroup to start traversing from. It doesn't have to be
        committed to Datastore itself (but all its nested groups should be
        there already).

  Returns:
    List of names of groups that form a cycle or empty list if no cycles.
  """
  # It is a depth-first search on a directed graph with back edge detection.
  # See http://www.cs.nyu.edu/courses/summer04/G22.1170-001/6a-Graphs-More.pdf

  # Cache of already fetched groups.
  groups = {group.key.id(): group}

  # List of groups that are completely explored (all subtree is traversed).
  visited = []
  # Stack of groups that are being explored now. In case cycle is detected
  # it would contain that cycle.
  visiting = []

  def visit(group):
    """Recursively explores |group| subtree, returns True if finds a cycle."""
    assert group not in visiting
    assert group not in visited

    # Load bodies of nested groups not seen so far into |groups|.
    entities = ndb.get_multi(
        group_key(name) for name in group.nested if name not in groups)
    groups.update({entity.key.id(): entity for entity in entities if entity})

    visiting.append(group)
    for nested in group.nested:
      obj = groups.get(nested)
      # Do not crash if non-existent group is referenced somehow.
      if not obj:
        continue
      # Cross edge. Can happen in diamond-like graph, not a cycle.
      if obj in visited:
        continue
      # Back edge: |group| references its own ancestor -> cycle.
      if obj in visiting:
        return True
      # Explore subtree.
      if visit(obj):
        return True
    visiting.pop()

    visited.append(group)
    return False

  visit(group)
  return [group.key.id() for group in visiting]


################################################################################
## Secrets store.


# TODO(vadimsh): Move secrets outside of AuthGlobalConfig entity group and
# encrypt them.


class AuthSecretScope(ndb.Model):
  """Entity to act as parent entity for AuthSecret.

  Parent is AuthGlobalConfig entity keyed at root_key().

  Id of this entity defines scope of secret keys that have this entity as
  a parent. Always 'local' currently.
  """


class AuthSecret(ndb.Model):
  """Some service-wide named secret blob.

  Parent entity is always Key(AuthSecretScope, 'local', parent=root_key()) now.

  There should be only very limited number of AuthSecret entities around. AuthDB
  fetches them all at once. Do not use this entity for per-user secrets.

  Holds most recent value of a secret as well as several previous values. Most
  recent value is used to generate new tokens, previous values may be used to
  validate existing tokens. That way secret can be rotated without invalidating
  any existing outstanding tokens.
  """
  # Disable useless in-process per-request cache.
  _use_cache = False

  # Last several values of a secret, with current value in front.
  values = ndb.BlobProperty(repeated=True, indexed=False)

  # When secret was modified last time.
  modified_ts = ndb.DateTimeProperty(auto_now_add=True)
  # Who modified the secret last time.
  modified_by = IdentityProperty()

  @classmethod
  def bootstrap(cls, name, length=32):
    """Creates a secret if it doesn't exist yet.

    Args:
      name: name of the secret.
      length: length of the secret to generate if secret doesn't exist yet.

    Returns:
      Instance of AuthSecret (creating it if necessary) with random secret set.
    """
    # Note that 'get_or_insert' is a bad fit here. With 'get_or_insert' we'd
    # have to call os.urandom every time we want to get a key. It's a waste of
    # time and entropy.
    key = ndb.Key(
        cls, name,
        parent=ndb.Key(AuthSecretScope, 'local', parent=root_key()))
    entity = key.get()
    if entity is not None:
      return entity
    @ndb.transactional
    def create():
      entity = key.get()
      if entity is not None:
        return entity
      logging.info('Creating new secret key %s', name)
      entity = cls(
          key=key,
          values=[os.urandom(length)],
          modified_by=get_service_self_identity())
      entity.put()
      return entity
    return create()


################################################################################
## IP whitelist.


class AuthIPWhitelistAssignments(ndb.Model, AuthVersionedEntityMixin):
  """A singleton entity with "identity -> AuthIPWhitelist to use" mapping.

  Entity key is ip_whitelist_assignments_key(). Parent entity is root_key().

  See AuthIPWhitelist for more info about IP whitelists.
  """
  # Disable useless in-process per-request cache.
  _use_cache = False

  class Assignment(ndb.Model):
    # Identity name to limit by IP whitelist. Unique key in 'assignments' list.
    identity = IdentityProperty()
    # Name of IP whitelist to use (see AuthIPWhitelist).
    ip_whitelist = ndb.StringProperty()
    # Why the assignment was created.
    comment = ndb.StringProperty()
    # When the assignment was created.
    created_ts = ndb.DateTimeProperty()
    # Who created the assignment.
    created_by = IdentityProperty()

  # Holds all the assignments.
  assignments = ndb.LocalStructuredProperty(Assignment, repeated=True)


class AuthIPWhitelist(
    ndb.Model,
    AuthVersionedEntityMixin,
    datastore_utils.SerializableModelMixin):
  """A named set of whitelisted IPv4 and IPv6 subnets.

  Can be assigned to individual user accounts to forcibly limit them only to
  particular IP addresses, e.g. it can be used to enforce that specific service
  account is used only from some known IP range. The mapping between accounts
  and IP whitelists is stored in AuthIPWhitelistAssignments.

  Entity id is a name of the whitelist. Parent entity is root_key().
  """
  # Disable useless in-process per-request cache.
  _use_cache = False

  # How to convert this entity to or from serializable dict.
  serializable_properties = {
    'subnets': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'description': datastore_utils.READABLE | datastore_utils.WRITABLE,
    'created_ts': datastore_utils.READABLE,
    'created_by': datastore_utils.READABLE,
    'modified_ts': datastore_utils.READABLE,
    'modified_by': datastore_utils.READABLE,
  }

  # The list of subnets. The validator is used only as a last measure. JSON API
  # handler should do validation too.
  subnets = ndb.StringProperty(
      repeated=True, validator=lambda _, val: ipaddr.normalize_subnet(val))

  # Human readable description.
  description = ndb.TextProperty(default='')

  # When the list was created.
  created_ts = ndb.DateTimeProperty()
  # Who created the list.
  created_by = IdentityProperty()

  def is_ip_whitelisted(self, ip):
    """Returns True if ipaddr.IP is in the whitelist."""
    # TODO(vadimsh): If number of subnets to check grows it makes sense to add
    # an internal cache to 'subnet_from_string' (sort of like in re.compile).
    return any(
        ipaddr.is_in_subnet(ip, ipaddr.subnet_from_string(net))
        for net in self.subnets)


def ip_whitelist_key(name):
  """Returns ndb.Key for AuthIPWhitelist entity given its name."""
  return ndb.Key(AuthIPWhitelist, name, parent=root_key())


def is_valid_ip_whitelist_name(name):
  """True if string looks like a valid IP whitelist name."""
  return bool(IP_WHITELIST_NAME_RE.match(name))


def bots_ip_whitelist():
  """Returns a name of a special IP whitelist that controls IP-based auth.

  Requests without authentication headers coming from IPs in this whitelist
  are authenticated as coming from IP_WHITELISTED_BOT_ID ('bot:whitelisted-ip').

  DEPRECATED.
  """
  return '%s-bots' % app_identity.get_application_id()


@ndb.transactional
def bootstrap_ip_whitelist(name, subnets, description=''):
  """Adds subnets to an IP whitelist if not there yet.

  Can be used on local dev appserver to add 127.0.0.1 to IP whitelist during
  startup. Should not be used from request handlers.

  Args:
    name: IP whitelist name to add a subnet to.
    subnets: IP subnet to add (as a list of strings).
    description: description of IP whitelist (if new entity is created).

  Returns:
    True if entry was added, False if it is already there or subnet is invalid.
  """
  assert isinstance(subnets, (list, tuple))
  try:
    subnets = [ipaddr.normalize_subnet(s) for s in subnets]
  except ValueError:
    return False
  key = ip_whitelist_key(name)
  entity = key.get()
  if entity and all(s in entity.subnets for s in subnets):
    return False
  now = utils.utcnow()
  if not entity:
    entity = AuthIPWhitelist(
        key=key,
        description=description,
        created_ts=now,
        created_by=get_service_self_identity())
  for s in subnets:
    if s not in entity.subnets:
      entity.subnets.append(s)
  entity.record_revision(
      modified_by=get_service_self_identity(),
      modified_ts=now,
      comment='Bootstrap')
  entity.put()
  replicate_auth_db()
  return True


def bootstrap_loopback_ips():
  """Adds 127.0.0.1 and ::1 to '<appid>-bots' IP whitelist.

  Useful on local dev server and in tests. Must not be used in production.

  Returns list of corresponding bot Identities.
  """
  # See api.py, AuthDB.verify_ip_whitelisted for IP -> Identity conversion.
  assert utils.is_local_dev_server()
  bootstrap_ip_whitelist(
      bots_ip_whitelist(), ['127.0.0.1', '::1'], 'Local bots')
  return [IP_WHITELISTED_BOT_ID]


@ndb.transactional
def bootstrap_ip_whitelist_assignment(identity, ip_whitelist, comment=''):
  """Sets a mapping "identity -> IP whitelist to use" for some account.

  Replaces existing assignment. Can be used on local dev appserver to configure
  IP whitelist assignments during startup or in tests. Should not be used from
  request handlers.

  Args:
    identity: Identity to modify.
    ip_whitelist: name of AuthIPWhitelist to assign.
    comment: comment to set.

  Returns:
    True if IP whitelist assignment was modified, False if it was already set.
  """
  entity = (
      ip_whitelist_assignments_key().get() or
      AuthIPWhitelistAssignments(key=ip_whitelist_assignments_key()))

  found = False
  for assignment in entity.assignments:
    if assignment.identity == identity:
      if assignment.ip_whitelist == ip_whitelist:
        return False
      assignment.ip_whitelist = ip_whitelist
      assignment.comment = comment
      found = True
      break

  now = utils.utcnow()
  if not found:
    entity.assignments.append(
        AuthIPWhitelistAssignments.Assignment(
            identity=identity,
            ip_whitelist=ip_whitelist,
            comment=comment,
            created_ts=now,
            created_by=get_service_self_identity()))

  entity.record_revision(
      modified_by=get_service_self_identity(),
      modified_ts=now,
      comment='Bootstrap')
  entity.put()
  replicate_auth_db()
  return True


def fetch_ip_whitelists():
  """Fetches AuthIPWhitelistAssignments and all AuthIPWhitelist entities.

  Returns:
    (AuthIPWhitelistAssignments, list of AuthIPWhitelist).
  """
  assign_fut = ip_whitelist_assignments_key().get_async()
  whitelists_fut = AuthIPWhitelist.query(ancestor=root_key()).fetch_async()

  assignments = (
      assign_fut.get_result() or
      AuthIPWhitelistAssignments(key=ip_whitelist_assignments_key()))

  whitelists = sorted(whitelists_fut.get_result(), key=lambda x: x.key.id())
  return assignments, whitelists


################################################################################
## Dev config. Used only on dev server or '-dev' instances.


class AuthDevConfig(ndb.Model):
  """Authentication related configuration for development or tests.

  Meant to be updated via Cloud Console Datastore UI.

  ID is 'dev_config'.
  """

  # Disable memcache to simplify editing of this entity through datastore UI.
  _use_cache = False
  _use_memcache = False

  # A custom endpoint to validate OAuth tokens to use as a fallback.
  #
  # E.g. "https://www.googleapis.com/oauth2/v1/tokeninfo".
  token_info_endpoint = ndb.StringProperty(indexed=False, default='')


@utils.cache_with_expiration(60)
def get_dev_config():
  """Returns an instance of AuthDevConfig (possibly uninitialized).

  Asserts that it is used only on dev instance.
  """
  assert utils.is_local_dev_server() or utils.is_dev()
  k = ndb.Key('AuthDevConfig', 'dev_config')
  e = k.get()
  if not e:
    logging.warning('Initializing AuthDevConfig entity')
    e = AuthDevConfig(key=k)
    e.put()  # there's a race condition here, but we don't care
  return e
