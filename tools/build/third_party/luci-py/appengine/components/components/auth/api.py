# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines main bulk of public API of auth component.

Functions defined here can be safely called often (multiple times per request),
since they are using in-memory read only cache of Auth DB entities.

Functions that operate on most current state of DB are in model.py. And they are
generally should not be used outside of Auth components implementation.
"""

# Pylint doesn't like ndb.transactional(...).
# pylint: disable=E1120

import collections
import functools
import json
import logging
import os
import threading
import time
import urllib

from google.appengine.api import oauth
from google.appengine.api import urlfetch
from google.appengine.ext import ndb
from google.appengine.runtime import apiproxy_errors

from components.datastore_utils import config as ds_config
from components import utils

from . import config
from . import ipaddr
from . import model
from .proto import delegation_pb2

# Part of public API of 'auth' component, exposed by this module.
__all__ = [
  'AuthDetails',
  'AuthenticationError',
  'AuthorizationError',
  'autologin',
  'disable_process_cache',
  'Error',
  'get_auth_details',
  'get_current_identity',
  'get_delegation_token',
  'get_peer_identity',
  'get_peer_ip',
  'get_process_cache_expiration_sec',
  'get_request_auth_db',
  'get_secret',
  'get_web_client_id',
  'GroupListing',
  'is_admin',
  'is_group_member',
  'is_in_ip_whitelist',
  'is_superuser',
  'list_group',
  'new_auth_details',
  'public',
  'require',
  'SecretKey',
  'verify_ip_whitelisted',
  'warmup',
]


# A callback (configured through appengine_config.py mechanism) that returns
# a list of additional OAuth client IDs we trust in this GAE application.
_additional_client_ids_cb = None

# How soon process-global AuthDB cache expires (may be 0), sec.
_process_cache_expiration_sec = 30
# True if fetch_auth_db was called at least once and created all root entities.
_lazy_bootstrap_ran = False

# Protects _auth_db* globals below.
_auth_db_lock = threading.Lock()
# Currently cached instance of AuthDB.
_auth_db = None
# When current value of _auth_db should be refetched.
_auth_db_expiration = None
# Holds id of a thread that is currently fetching AuthDB (or None).
_auth_db_fetching_thread = None

# Limits concurrent fetches of AuthDB.
#
# We don't want multiple threads fetching heavy AuthDB objects concurrently,
# since they may not all fit in the memory.
#
# If both _auth_db_lock and _auth_db_fetch_lock need to be locked,
# _auth_db_fetch_lock should be locked first.
_auth_db_fetch_lock = threading.Lock()

# Thread local storage for RequestCache (see 'get_request_cache').
_thread_local = threading.local()


# The endpoint used to validate an access token on dev server.
TOKEN_INFO_ENDPOINT = 'https://www.googleapis.com/oauth2/v1/tokeninfo'

# OAuth2 client_id of the "API Explorer" web app.
API_EXPLORER_CLIENT_ID = '292824132082.apps.googleusercontent.com'


################################################################################
## Exception classes.


class Error(Exception):
  """Base class for exceptions raised by auth component."""
  def __init__(self, message=None):
    super(Error, self).__init__(message or self.__doc__)


class AuthenticationError(Error):
  """Provided credentials are invalid."""


class AuthorizationError(Error):
  """Access is denied."""


################################################################################
## AuthDB.


# Name of a secret. Used by 'get_secret' function.
SecretKey = collections.namedtuple('SecretKey', ['name'])


# The representation of AuthGroup used by AuthDB, preprocessed for faster
# membership checks. We keep it in AuthDB in place of AuthGroup to reduce RAM
# usage.
CachedGroup = collections.namedtuple('CachedGroup', [
  'members',  # == set(m.to_bytes() for m in auth_group.members)
  'globs',
  'nested',
  'description',
  'owners',
  'created_ts',
  'created_by',
  'modified_ts',
  'modified_by',
])


# GroupListing is returned by list_group.
GroupListing = collections.namedtuple('GroupListing', [
  'members',  # list of Identity in no particular order
  'globs',    # list of IdentityGlob in no particular order
  'nested',   # list of strings with nested group names in no particular order
])


class AuthDB(object):
  """A read only in-memory database of auth configuration of a service.

  Holds user groups, all secret keys and OAuth2 configuration.

  Each instance process holds AuthDB object in memory and shares it between all
  requests, occasionally refetching it from Datastore.
  """

  def __init__(
      self,
      replication_state=None,
      global_config=None,
      groups=None,
      secrets=None,
      ip_whitelist_assignments=None,
      ip_whitelists=None,
      additional_client_ids=None):
    """
    Args:
      replication_state: instance of AuthReplicationState entity.
      global_config: instance of AuthGlobalConfig entity.
      groups: list of AuthGroup entities.
      secrets: list of AuthSecret entities.
      ip_whitelist_assignments: AuthIPWhitelistAssignments entity.
      ip_whitelists: list of AuthIPWhitelist entities.
      additional_client_ids: an additional list of OAuth2 client IDs to trust.
    """
    self.replication_state = replication_state or model.AuthReplicationState()
    self.global_config = global_config or model.AuthGlobalConfig()
    self.secrets = {}
    self.ip_whitelists = {e.key.string_id(): e for e in (ip_whitelists or [])}
    self.ip_whitelist_assignments = (
        ip_whitelist_assignments or model.AuthIPWhitelistAssignments())

    for secret in (secrets or []):
      assert secret.key.string_id() not in self.secrets, secret.key
      self.secrets[secret.key.string_id()] = secret

    # Preprocess groups for faster membership checks. Throw away original
    # entities to reduce memory usage.
    self.groups = {}
    for entity in (groups or []):
      self.groups[entity.key.string_id()] = CachedGroup(
          members=frozenset(m.to_bytes() for m in entity.members),
          globs=entity.globs or (),
          nested=entity.nested or (),
          description=entity.description,
          owners=entity.owners,
          created_ts=entity.created_ts,
          created_by=entity.created_by,
          modified_ts=entity.modified_ts,
          modified_by=entity.modified_by)

    # A set of all allowed client IDs (as provided via config and the callback).
    client_ids = []
    if self.global_config.oauth_client_id:
      client_ids.append(self.global_config.oauth_client_id)
    if self.global_config.oauth_additional_client_ids:
      client_ids.extend(self.global_config.oauth_additional_client_ids)
    client_ids.append(API_EXPLORER_CLIENT_ID)
    if additional_client_ids:
      client_ids.extend(additional_client_ids)
    self.allowed_client_ids = set(c for c in client_ids if c)

    # Lazy-initialized indexes structures. See _indexes().
    self._lock = threading.Lock()
    self._members_idx = None
    self._globs_idx = None
    self._nested_idx = None
    self._owned_idx = None

  def _indexes(self):
    """Lazily builds and returns various indexes used by get_relevant_subgraph.

    Members index is a map from serialized Identity to a list of groups that
    directly include it (NOT via glob or a nested subgroup).

    Globs index is a map from IdentityGlob to a list of groups that directly
    include it. We store it as OrderedDict to make 'get_relevant_subgraph'
    output deterministic (it linearly traverses through globs index keys at some
    point).

    Nested groups index is a map from a group name to a list of groups that
    directly include it.

    Ownership index is a map from a group name to a list of groups directly
    owned by it.

    Returns:
      (
        Members index as dict(Identity.to_bytes() str => [str with group name]),
        Globs index as OrderedDict(IndentityGlob => [str with group name],
        Nested groups index as dict(group name => [str with group name]),
        Ownership index as dict(group name => [str with group name]),
      )
    """
    with self._lock:
      if self._members_idx is not None:
        assert self._globs_idx is not None
        assert self._nested_idx is not None
        assert self._owned_idx is not None
        return (
            self._members_idx, self._globs_idx,
            self._nested_idx, self._owned_idx)

      logging.info('Building in-memory indexes...')

      members_idx = collections.defaultdict(list)
      globs_idx = collections.defaultdict(list)
      nested_idx = collections.defaultdict(list)
      owned_idx = collections.defaultdict(list)
      for name, group in sorted(self.groups.iteritems()):
        for member in group.members:
          members_idx[member].append(name)
        for glob in group.globs:
          globs_idx[glob].append(name)
        for nested in group.nested:
          nested_idx[nested].append(name)
        owned_idx[group.owners].append(name)

      logging.info('Finished building in-memory indexes')

      self._members_idx = members_idx
      self._globs_idx = collections.OrderedDict(sorted(globs_idx.items()))
      self._nested_idx = nested_idx
      self._owned_idx = owned_idx
      return members_idx, globs_idx, nested_idx, owned_idx

  @property
  def auth_db_rev(self):
    """Returns the revision number of groups database."""
    return self.replication_state.auth_db_rev

  @property
  def primary_id(self):
    """For services in Replica mode, GAE application ID of Primary."""
    return self.replication_state.primary_id

  @property
  def primary_url(self):
    """For services in Replica mode, root URL of Primary, i.e https://<host>."""
    return self.replication_state.primary_url

  @property
  def token_server_url(self):
    """URL of a token server to use to generate tokens, provided by Primary."""
    return self.global_config.token_server_url

  def is_group_member(self, group_name, identity):
    """Returns True if |identity| belongs to group |group_name|.

    Unknown groups are considered empty.
    """
    # Will be used when checking self.group_members_set sets.
    ident_as_bytes = identity.to_bytes()

    # While the code to add groups refuses to add cycle, this code ensures that
    # it doesn't go in a cycle by keeping track of the groups currently being
    # visited via |current| stack.
    current = []

    # Used to avoid revisiting same groups multiple times in case of
    # diamond-like graphs, e.g. A->B, A->C, B->D, C->D.
    visited = set()

    def is_member(group_name):
      # Wildcard group that matches all identities (including anonymous!).
      if group_name == model.GROUP_ALL:
        return True

      # An unknown group is empty.
      group_obj = self.groups.get(group_name)
      if not group_obj:
        logging.warning(
            'Querying unknown group: %s via %s', group_name, current)
        return False

      # In a group DAG a group can not reference any of its ancestors, since it
      # creates a cycle.
      if group_name in current:
        logging.warning(
            'Cycle in a group graph: %s via %s', group_name, current)
        return False

      # Explored this group already (and didn't find |identity| there) while
      # visiting some sibling branch? Can happen in diamond-like graphs.
      if group_name in visited:
        return False

      current.append(group_name)
      try:
        # Note that we don't include nested groups in GroupEssense.members sets
        # because it blows up memory usage pretty bad. We don't have very deep
        # nesting graphs, so checking nested groups separately is OK.
        if ident_as_bytes in group_obj.members:
          return True

        if any(glob.match(identity) for glob in group_obj.globs):
          return True

        return any(is_member(nested) for nested in group_obj.nested)
      finally:
        current.pop()
        visited.add(group_name)

    return is_member(group_name)

  def get_group(self, group_name):
    """Returns AuthGroup entity reconstructing it from the cache.

    It slightly differs from the original entity:
      - 'members' list is always sorted.
      - 'auth_db_rev' and 'auth_db_prev_rev' are not set.

    Returns:
      AuthGroup object or None if no such group.
    """
    g = self.groups.get(group_name)
    if not g:
      return None
    return model.AuthGroup(
        key=model.group_key(group_name),
        members=[model.Identity.from_bytes(m) for m in sorted(g.members)],
        globs=list(g.globs),
        nested=list(g.nested),
        description=g.description,
        owners=g.owners,
        created_ts=g.created_ts,
        created_by=g.created_by,
        modified_ts=g.modified_ts,
        modified_by=g.modified_by)

  def list_group(self, group_name, recursive=True):
    """Returns all members, all globs and all nested groups in a group.

    The returned lists are unordered.

    Args:
      group_name: name of a group to list.
      recursive: True to include nested group.

    Returns:
      GroupListing object.
    """
    members = set()  # set of strings (not Identity!), see CachedGroup
    globs = set()    # set of IdentityGlob
    nested = set()   # set of strings

    def accumulate(group_obj):
      members.update(group_obj.members)
      globs.update(group_obj.globs)
      nested.update(group_obj.nested)

    def finalize_listing():
      return GroupListing(
          members=[model.Identity.from_bytes(m) for m in members],
          globs=list(globs),
          nested=list(nested))

    if not recursive:
      group_obj = self.groups.get(group_name)
      if group_obj:
        accumulate(group_obj)
      return finalize_listing()

    # Set of groups already added to the listing.
    visited = set()

    def visit_group(name):
      # An unknown group is empty.
      group_obj = self.groups.get(name)
      if not group_obj or name in visited:
        return
      visited.add(name)
      accumulate(group_obj)
      for nested in group_obj.nested:
        visit_group(nested)

    visit_group(group_name)
    return finalize_listing()

  def fetch_groups_with_member(self, ident):
    """Returns a set of group names that have given Identity as a member.

    This is expensive call, don't use it unless really necessary.
    """
    # TODO(vadimsh): This is currently very dumb and can probably be optimized.
    return {g for g in self.groups if self.is_group_member(g, ident)}

  def get_group_names_with_prefix(self, prefix):
    """Returns a sorted list of group names that start with the given prefix."""
    return sorted(g for g in self.groups if g.startswith(prefix))

  def get_relevant_subgraph(self, principal):
    """Returns groups that include the principal and owned by principal.

    Returns it in a graph form where edges represent relations "subset of" and
    "owned by".

    Args:
      principal: Identity, IdentityGlob or a group name string.

    Returns:
      Graph instance.
    """
    # members_idx: {identity str => list of group names that have it}
    # globs_idx: {IdentityGlob tuple => list of group names that have it}
    # nested_idx: {group name => list of group names that include it}
    # owned_idx: {group name => list of group names owned by it}
    members_idx, globs_idx, nested_idx, owned_idx = self._indexes()

    # Note: when we say 'add_edge(A, IN, B)' we mean 'A' is a direct subset of
    # 'B' in the full group graph, i.e 'B' includes 'A' directly.
    graph = Graph()
    add_node = graph.add_node
    add_edge = graph.add_edge

    # Adds the given group and all groups that include it and owned by it (
    # perhaps indirectly) to 'graph'. Traverses group graph from leafs (most
    # nested groups) to roots (least nested groups that include other groups).
    def traverse(group):
      group_id, added = add_node(group)
      if added:
        for supergroup in nested_idx.get(group, ()):
          add_edge(group_id, Graph.IN, traverse(supergroup))
        for owned in owned_idx.get(group, ()):
          add_edge(group_id, Graph.OWNS, traverse(owned))
      return group_id

    # Find the leafs of the graph. It's the only part that depends on the exact
    # kind of the principal. Once we get to leaf groups, everything is uniform
    # after that: we just travel through the graph via 'traverse'.
    if isinstance(principal, model.Identity):
      graph.root_id, _ = add_node(principal)

      # Find all globs that match the identity. The identity will belong to
      # all groups the globs belong to. Note that 'globs_idx' is OrderedDict.
      for glob, groups_that_have_glob in globs_idx.iteritems():
        if glob.match(principal):
          glob_id, _ = add_node(glob)
          add_edge(graph.root_id, Graph.IN, glob_id)
          for group in groups_that_have_glob:
            add_edge(glob_id, Graph.IN, traverse(group))

      # Find all groups that directly mention the identity.
      for group in members_idx.get(principal.to_bytes(), ()):
        add_edge(graph.root_id, Graph.IN, traverse(group))

    elif isinstance(principal, model.IdentityGlob):
      graph.root_id, _ = add_node(principal)

      # Find all groups that directly mention the glob.
      for group in globs_idx.get(principal, ()):
        add_edge(graph.root_id, Graph.IN, traverse(group))

    elif isinstance(principal, basestring):
      graph.root_id = traverse(principal)
    else:
      raise TypeError('Wrong "principal" type %s' % type(principal))

    return graph

  def get_secret(self, key):
    """Returns list of strings with last known values of a secret.

    If secret doesn't exist yet, it will be created.

    Args:
      secret_key: instance of SecretKey with name of a secret.
    """
    # There's a race condition here: multiple requests, that share same AuthDB
    # object, fetch same missing secret key. It's rare (since key bootstrap
    # process is rare) and not harmful (since AuthSecret.bootstrap is
    # implemented with transaction inside). We ignore it.
    if key.name not in self.secrets:
      self.secrets[key.name] = model.AuthSecret.bootstrap(key.name)
    entity = self.secrets[key.name]
    return list(entity.values)

  def is_in_ip_whitelist(self, whitelist_name, ip, warn_if_missing=True):
    """Returns True if the given IP belongs to the given IP whitelist.

    Missing IP whitelists are considered empty.

    Args:
      whitelist_name: name of the IP whitelist (e.g. 'bots').
      ip: instance of ipaddr.IP.
      warn_if_missing: if True and IP whitelist is missing, logs a warning.
    """
    whitelist = self.ip_whitelists.get(whitelist_name)
    if not whitelist:
      if warn_if_missing:
        logging.error('Unknown IP whitelist: %s', whitelist_name)
      return False
    return whitelist.is_ip_whitelisted(ip)

  def verify_ip_whitelisted(self, identity, ip):
    """Verifies IP is in a whitelist assigned to the Identity.

    This check is used to restrict some callers to particular IP subnets as
    additional security measure.

    Args:
      identity: caller's identity.
      ip: instance of ipaddr.IP.

    Raises:
      AuthorizationError if identity has an IP whitelist assigned and given IP
      address doesn't belong to it.
    """
    assert isinstance(identity, model.Identity), identity

    for assignment in self.ip_whitelist_assignments.assignments:
      if assignment.identity == identity:
        whitelist_name = assignment.ip_whitelist
        break
    else:
      return

    if not self.is_in_ip_whitelist(whitelist_name, ip):
      ip_as_str = ipaddr.ip_to_string(ip)
      logging.error(
          'IP is not whitelisted.\nIdentity: %s\nIP: %s\nWhitelist: %s',
          identity.to_bytes(), ip_as_str, whitelist_name)
      raise AuthorizationError('IP %s is not whitelisted' % ip_as_str)

  def is_allowed_oauth_client_id(self, client_id):
    """True if given OAuth2 client_id can be used to authenticate the user."""
    return client_id in self.allowed_client_ids

  def get_oauth_config(self):
    """Returns a tuple with OAuth2 config.

    Format of the tuple: (client_id, client_secret, additional client ids list).
    """
    if not self.global_config:
      return None, None, None
    return (
        self.global_config.oauth_client_id,
        self.global_config.oauth_client_secret,
        self.global_config.oauth_additional_client_ids)


################################################################################
## OAuth client configuration for the web UI.


class AuthWebUIConfig(ds_config.GlobalConfig):
  """Configuration of web UI (updated through /auth/bootstrap/oauth).

  See BootstrapOAuthHandler in ui/ui.py for where this config is updated.
  """
  web_client_id = ndb.StringProperty(indexed=False, default='')


@utils.cache_with_expiration(300)
def get_web_client_id():
  """Returns OAuth2 client ID for the web UI (if configured) or '' (if not).

  Can be used by components.auth API users to inject a web client ID into pages.
  """
  return get_web_client_id_uncached()


def get_web_client_id_uncached():
  """Fetches web client ID from the datastore (slow, use get_web_client_id)."""
  cfg = AuthWebUIConfig.fetch()
  return cfg.web_client_id if cfg else ''


def set_web_client_id(web_client_id):
  """Changes the configured OAuth2 client ID for the web UI."""
  cfg = AuthWebUIConfig.fetch() or AuthWebUIConfig()
  cfg.modify(
      updated_by=get_current_identity().to_bytes(),
      web_client_id=web_client_id)


################################################################################
## OAuth token check.


def configure_client_ids_provider(cb):
  """Sets a callback that returns a list of additional client IDs to trust.

  This list is used in additional to a global list of trusted client IDs,
  distributed by the auth service.

  This list usually includes "local" client ID, used only by the UI of the
  current service.

  Args:
    cb: argumentless function returning an iterable of client_ids.
  """
  global _additional_client_ids_cb
  _additional_client_ids_cb = cb


def attempt_oauth_initialization(scope):
  """Attempts to perform GetOAuthUser RPC retrying deadlines.

  The result it cached in appengine.api.oauth guts. Never raises exceptions,
  just gives up letting subsequent oauth.* calls fail in a proper way.
  """
  # 4 attempts: ~20 sec (default RPC deadline is 5 sec).
  attempt = 0
  while attempt < 4:
    attempt += 1
    try:
      oauth.get_client_id(scope)
      return
    except apiproxy_errors.DeadlineExceededError as e:
      logging.warning('DeadlineExceededError: %s', e)
      continue
    except oauth.OAuthServiceFailureError as e:
      logging.warning(
          'oauth.OAuthServiceFailureError (%s): %s', e.__class__.__name__, e)
      # oauth library "caches" the error code in os.environ and retrying
      # oauth.get_client_id doesn't do anything. Clear this cache first, see
      # oauth_api.py, _maybe_call_get_oauth_user in GAE SDK.
      os.environ.pop('OAUTH_ERROR_CODE', None)
      continue
    except oauth.Error as e:
      # Next call to oauth.get_client_id() will trigger same error and it will
      # be handled for real.
      logging.warning('oauth.Error (%s): %s', e.__class__.__name__, e)
      return


def extract_oauth_caller_identity():
  """Extracts and validates Identity of a caller for the current request.

  Implemented on top of GAE OAuth2 API.

  Uses client_id whitelist fetched from the datastore to validate OAuth client
  used to build access_token. Also recognizes various types of service accounts
  and verifies that their client_id is what it should be. Service account's
  client_id doesn't have to be in client_id whitelist.

  Returns:
    (Identity, AuthDetails).

  Raises:
    AuthenticationError in case access_token is missing or invalid.
    AuthorizationError in case client_id is forbidden.
  """
  # OAuth2 scope a token should have.
  oauth_scope = 'https://www.googleapis.com/auth/userinfo.email'

  # Fetch OAuth request state with retries. oauth.* calls use it internally.
  attempt_oauth_initialization(oauth_scope)

  # Extract client_id and email from access token. That also validates the token
  # and raises OAuthRequestError if token is revoked or otherwise not valid.
  try:
    client_id = oauth.get_client_id(oauth_scope)
  except oauth.OAuthRequestError:
    raise AuthenticationError('Invalid OAuth token')

  # This call just reads data cached by oauth.get_client_id, and thus should
  # never fail.
  email = oauth.get_current_user(oauth_scope).email()

  # Is client_id in the explicit whitelist? Used with three legged OAuth. Detect
  # Google service accounts. No need to whitelist client_ids for each of them,
  # since email address uniquely identifies credentials used.
  good = (
      email.endswith('.gserviceaccount.com') or
      get_request_auth_db().is_allowed_oauth_client_id(client_id))

  if not good:
    raise AuthorizationError(
        'Unrecognized combination of email (%s) and client_id (%s). '
        'Is client_id whitelisted? Is it unrecognized service account?' %
        (email, client_id))
  try:
    ident = model.Identity(model.IDENTITY_USER, email)
  except ValueError:
    raise AuthenticationError('Unsupported user email: %s' % email)
  return ident, new_auth_details(
      is_superuser=oauth.is_current_user_admin(oauth_scope))


def check_oauth_access_token(header):
  """Verifies the access token of the current request.

  This function uses slightly different strategies for prod, dev and local
  environments:
    * In prod it always require real OAuth2 tokens, validated by GAE OAuth2 API.
    * On local devserver it uses URL Fetch and prod token info endpoint.
    * On '-dev' instances or on dev server it can also fallback to a custom
      token info endpoint, defined in AuthDevConfig datastore entity. This is
      useful to "stub" authentication when running integration or load tests.

  In addition to checking the correctness of OAuth token, this function also
  verifies that the client_id associated with the token is whitelisted in the
  auth config.

  The client_id check is skipped on the local devserver or when using custom
  token info endpoint (e.g. on '-dev' instances).

  Args:
    header: a value of Authorization header (as is in the request).

  Returns:
    Tuple (ident, AuthDetails), where ident is an identity of the caller in
    case the request was successfully validated (always 'user:...', never
    anonymous), and AuthDetails.is_superuser is true if the caller is GAE-level
    admin.

  Raises:
    AuthenticationError in case the access token is invalid.
    AuthorizationError in case the access token is not allowed.
  """
  if not header:
    raise AuthenticationError('No "Authorization" header')

  # Non-development instances always use real OAuth API.
  if not utils.is_local_dev_server() and not utils.is_dev():
    return extract_oauth_caller_identity()

  # OAuth2 library is mocked on dev server to return some nonsense. Use (slow,
  # but real) OAuth2 API endpoint instead to validate access_token. It is also
  # what Cloud Endpoints do on a local server.
  if utils.is_local_dev_server():
    # auth_call returns tuple (Identity, AuthDetails). There are no additional
    # details if not using native GAE OAuth API.
    auth_call = lambda: (
        dev_oauth_authentication(header, TOKEN_INFO_ENDPOINT), None)
  else:
    auth_call = extract_oauth_caller_identity

  # Do not fallback to custom endpoint if not configured. This call also has a
  # side effect of initializing AuthDevConfig entity in the datastore, to make
  # it editable in Datastore UI.
  cfg = model.get_dev_config()
  if not cfg.token_info_endpoint:
    return auth_call()

  # Try the real call first, then fallback to the custom validation endpoint.
  try:
    return auth_call()
  except AuthenticationError:
    ident = dev_oauth_authentication(header, cfg.token_info_endpoint, '.dev')
    logging.warning('Authenticated as dev account: %s', ident.to_bytes())
    return ident, None


def dev_oauth_authentication(header, token_info_endpoint, suffix=''):
  """OAuth2 based authentication via URL Fetch to the token info endpoint.

  This is slow and ignores client_id whitelist. Must be used only in
  a development environment.

  Returns:
    Identity of the caller in case the request was successfully validated.

  Raises:
    AuthenticationError in case access token is missing or invalid.
    AuthorizationError in case the token is not trusted.
  """
  assert utils.is_local_dev_server() or utils.is_dev()

  header = header.split(' ', 1)
  if len(header) != 2 or header[0] not in ('OAuth', 'Bearer'):
    raise AuthenticationError('Invalid authorization header')

  # Adapted from endpoints/users_id_tokens.py, _set_bearer_user_vars_local.
  logging.info('Using dev token info endpoint %s', token_info_endpoint)
  result = urlfetch.fetch(
      url='%s?%s' % (
          token_info_endpoint,
          urllib.urlencode({'access_token': header[1]})),
      follow_redirects=False,
      validate_certificate=True)
  if result.status_code != 200:
    try:
      error = json.loads(result.content)['error_description']
    except (KeyError, ValueError):
      error = repr(result.content)
    raise AuthenticationError('Failed to validate the token: %s' % error)

  token_info = json.loads(result.content)
  if 'email' not in token_info:
    raise AuthorizationError('Token doesn\'t include an email address')
  if not token_info.get('verified_email'):
    raise AuthorizationError('Token email isn\'t verified')

  email = token_info['email'] + suffix
  try:
    return model.Identity(model.IDENTITY_USER, email)
  except ValueError:
    raise AuthorizationError('Unsupported user email: %s' % email)


################################################################################
## RequestCache.


# Additional information extracted from the credentials by an auth method.
#
# Lives in the request authentication context (aka RequestCache). Cleared in
# a presence of a delegation token.
AuthDetails = collections.namedtuple('AuthDetails', [
  'is_superuser',  # True if the caller is GAE-level administrator

  # Populated when using 'gce_vm_authentication' method.
  'gce_instance',  # name of a GCE VM that is making the call
  'gce_project',   # name of a GCE project that owns a VM making the call
])


# pylint: disable=redefined-outer-name
def new_auth_details(is_superuser=False, gce_instance=None, gce_project=None):
  """Constructs AuthDetails, filling in defaults."""
  return AuthDetails(
      is_superuser=is_superuser,
      gce_instance=gce_instance,
      gce_project=gce_project)


class RequestCache(object):
  """Holds authentication related information for the current request.

  Current request is a request being processed by currently running thread.
  A thread can handle at most one request at a time (as assumed by WSGI model).
  But same thread can be reused for another request later. In that case second
  request gets a new copy of RequestCache.

  All members can be set only once, since they are not supposed to be changing
  during lifetime of a request.

  See also:
    * reinitialize_request_cache - to forcibly setup new RequestCache.
    * get_request_cache - to grab current thread-local RequestCache.
  """

  def __init__(self):
    self._auth_db = None
    self._current_identity = None
    self._delegation_token = None
    self._peer_identity = None
    self._peer_ip = None
    self._auth_details = None

  @property
  def auth_db(self):
    """Returns request-local copy of AuthDB, fetching it if necessary."""
    if self._auth_db is None:
      self._auth_db = get_process_auth_db()
    return self._auth_db

  @property
  def auth_details(self):
    return self._auth_details or new_auth_details()

  @auth_details.setter
  def auth_details(self, value):
    assert self._auth_details is None # haven't been set yet
    assert value is None or isinstance(value, AuthDetails), value
    self._auth_details = value or new_auth_details()

  @property
  def current_identity(self):
    return self._current_identity or model.Anonymous

  @current_identity.setter
  def current_identity(self, current_identity):
    """Records identity to use for auth decisions.

    It may be delegated identity conveyed through delegation token.
    If delegation is not used, it is equal to peer identity.
    """
    assert isinstance(current_identity, model.Identity), current_identity
    assert not self._current_identity
    self._current_identity = current_identity

  @property
  def delegation_token(self):
    return self._delegation_token

  @delegation_token.setter
  def delegation_token(self, token):
    """Records unwrapped verified delegation token used by this request."""
    assert isinstance(token, delegation_pb2.Subtoken), token
    assert not self._delegation_token
    self._delegation_token = token

  @property
  def peer_identity(self):
    return self._peer_identity or model.Anonymous

  @peer_identity.setter
  def peer_identity(self, peer_identity):
    """Records identity of whoever is making the request.

    It's an identity directly extracted from user credentials (ignoring
    delegation tokens).
    """
    assert isinstance(peer_identity, model.Identity), peer_identity
    assert not self._peer_identity
    self._peer_identity = peer_identity

  @property
  def peer_ip(self):
    return self._peer_ip

  @peer_ip.setter
  def peer_ip(self, peer_ip):
    assert isinstance(peer_ip, ipaddr.IP)
    assert not self._peer_ip
    self._peer_ip = peer_ip

  def close(self):
    """Helps GC to collect garbage faster."""
    self._auth_db = None
    self._current_identity = None
    self._delegation_token = None
    self._peer_identity = None
    self._peer_ip = None
    self._auth_details = None


def disable_process_cache():
  """Disables in-process cache of AuthDB.

  Useful in tests. Once disabled, it can't be enabled again.
  """
  global _process_cache_expiration_sec
  _process_cache_expiration_sec = 0


def get_process_cache_expiration_sec():
  """How long auth db is cached in process memory."""
  return _process_cache_expiration_sec


def reinitialize_request_cache():
  """Creates new RequestCache instance and puts it into thread local store.

  RequestCached used by the thread before this call (if any) is forcibly closed.
  """
  prev = getattr(_thread_local, 'request_cache', None)
  if prev:
    prev.close()
  request_cache = RequestCache()
  _thread_local.request_cache = request_cache
  return request_cache


def get_request_cache():
  """Returns instance of RequestCache associated with the current request.

  Creates a new empty one if necessary.
  """
  cache = getattr(_thread_local, 'request_cache', None)
  return cache or reinitialize_request_cache()


def fetch_auth_db(known_auth_db=None):
  """Returns instance of AuthDB.

  If |known_auth_db| is None, this function always returns a new instance.

  If |known_auth_db| is not None, this function will compare it to the latest
  version in the datastore. It they match, function will return known_auth_db
  unaltered (meaning that there's no need to refetch AuthDB), otherwise it will
  fetch a fresh copy of AuthDB and return it.

  Runs in transaction to guarantee consistency of fetched data. Effectively it
  fetches momentary snapshot of subset of root_key() entity group.
  """
  # Entity group root. To reduce amount of typing.
  root_key = model.root_key()

  additional_client_ids = []

  @ndb.non_transactional
  def prepare():
    """Returns True to proceed with the fetch, False to abort."""
    # Assumption that root entities always exist make code simpler by removing
    # 'is not None' checks. So make sure they do, by running bootstrap code
    # at most once per lifetime of an instance. We do it lazily here (instead of
    # module scope) to ensure NDB calls are happening in a context of HTTP
    # request. Presumably it reduces probability of instance to stuck during
    # initial loading.
    global _lazy_bootstrap_ran
    if not _lazy_bootstrap_ran:
      config.ensure_configured()
      model.AuthGlobalConfig.get_or_insert(root_key.string_id())
      _lazy_bootstrap_ran = True
    # Call the user-supplied callbacks in non-transactional context.
    if _additional_client_ids_cb:
      additional_client_ids.extend(_additional_client_ids_cb())
    web_id = get_web_client_id()
    if web_id:
      additional_client_ids.append(web_id)
    # Fetch the latest known revision before opening the transaction. If it
    # matches |known_auth_db| we don't need to do the transaction at all.
    if known_auth_db is not None:
      state = model.get_replication_state()
      return (
          not state or
          state.primary_id != known_auth_db.primary_id or
          state.auth_db_rev != known_auth_db.auth_db_rev)
    return True

  @ndb.transactional(propagation=ndb.TransactionOptions.INDEPENDENT)
  def fetch():
    # TODO(vadimsh): Add memcache keyed at |auth_db_rev| so only one frontend
    # instance has to pay the cost of fetching AuthDB from Datastore via
    # multiple RPCs. All other instances will fetch it via single memcache
    # 'get'.

    # Fetch all stuff in parallel. Fetch ALL groups and ALL secrets.
    replication_state_future = model.replication_state_key().get_async()
    global_config_future = root_key.get_async()
    groups_future = model.AuthGroup.query(ancestor=root_key).fetch_async()
    secrets_future = model.AuthSecret.query(ancestor=root_key).fetch_async()

    # It's fine to block here as long as it's the last fetch.
    ip_whitelist_assignments, ip_whitelists = model.fetch_ip_whitelists()

    # Do not invoke AuthDB constructor while we still hold the transaction,
    # since it does some heavy computations. Instead just return all kwargs for
    # it, so AuthDB can be built outside.
    return {
      'replication_state': replication_state_future.get_result(),
      'global_config': global_config_future.get_result(),
      'groups': groups_future.get_result(),
      'secrets': secrets_future.get_result(),
      'ip_whitelist_assignments': ip_whitelist_assignments,
      'ip_whitelists': ip_whitelists,
      'additional_client_ids': additional_client_ids,
    }

  if prepare():  # non-transactional work
    return AuthDB(**fetch())
  return known_auth_db


def reset_local_state():
  """Resets all local caches to an initial state. Only for testing."""
  global _auth_db
  global _auth_db_expiration
  global _auth_db_fetching_thread
  global _lazy_bootstrap_ran
  _auth_db = None
  _auth_db_expiration = None
  _auth_db_fetching_thread = None
  _lazy_bootstrap_ran = False
  _thread_local.request_cache = None


def get_process_auth_db():
  """Returns instance of AuthDB from process-global cache.

  Will refetch it if necessary. Two subsequent calls may return different
  instances if cache expires between the calls.
  """
  global _auth_db_fetching_thread

  known_auth_db = None

  with _auth_db_lock:
    # Not using cache at all (usually in tests) => always fetch.
    if not _process_cache_expiration_sec:
      return fetch_auth_db()

    # Cached copy is still fresh?
    if _auth_db and time.time() < _auth_db_expiration:
      return _auth_db

    # Fetching AuthDB for the first time ever? Do it under the lock because
    # there's nothing to return yet. All threads would have to wait for this
    # initial fetch to complete.
    if _auth_db is None:
      return _initialize_auth_db_cache()

    # We have a cached copy and it has expired. Maybe some thread is already
    # fetching it? Don't block an entire process on this, return a little bit
    # stale copy instead right away.
    if _auth_db_fetching_thread is not None:
      logging.info(
          'Using stale copy of AuthDB while another thread is fetching '
          'a fresh one. Cached copy expired %.1f sec ago.',
          time.time() - _auth_db_expiration)
      return _auth_db

    # No one is fetching AuthDB yet. Start the operation, release the lock so
    # other threads can figure this out and use stale copies instead of blocking
    # on the lock.
    _auth_db_fetching_thread = threading.current_thread()
    known_auth_db = _auth_db
    logging.debug('Refetching AuthDB')

  # Do the actual fetch outside the lock. Be careful to handle any unexpected
  # exception by 'fixing' the global state before leaving this function.
  try:
    # Note: if process doesn't use 'get_latest_auth_db' this lock is noop, since
    # the dance we do with _auth_db_fetching_thread already guarantees there's
    # only one thread that is doing the fetch. This lock is useful only in
    # conjunction with concurrent 'get_latest_auth_db' calls.
    with _auth_db_fetch_lock:
      fetched = fetch_auth_db(known_auth_db=known_auth_db)
  except Exception:
    # Be sure to allow other threads to try the fetch. Meanwhile log the
    # exception and return a stale copy of AuthDB. Better than nothing.
    logging.exception('Failed to refetch AuthDB, returning stale cached copy')
    with _auth_db_lock:
      assert _auth_db_fetching_thread == threading.current_thread()
      _auth_db_fetching_thread = None
      return _auth_db

  # Fetch has completed successfully. Update the process cache now.
  with _auth_db_lock:
    assert _auth_db_fetching_thread == threading.current_thread()
    _auth_db_fetching_thread = None
    return _roll_auth_db_cache(fetched)


def get_latest_auth_db():
  """Returns the most recent AuthDB instance, fetching it if necessary.

  Very heavy call. If the absolute consistency is not required, prefer to use
  get_process_auth_db instead. The later is much faster by relying on in-process
  cache (as a downside it may lag behind the most recent state).
  """
  # We just "rush" the update of the internal cache. That way get_latest_auth_db
  # blocks for long only if something in AuthDB has changed, i.e our cached copy
  # becomes stale. By reusing _auth_db (instead of keeping a separate cache or
  # something like that), we keep the memory footprint smaller.
  #
  # Also, to avoid fetching heavy AuthDB objects concurrently (and thus causing
  # OOM), we do the entire transaction under the lock. We can't reuse
  # _auth_db_lock, since it must not be locked for a long time (it would break
  # performance guarantees of 'get_process_auth_db'). We guard everything with
  # _auth_db_fetch_lock (instead of just 'fetch_auth_db') to make sure that once
  # it gets unlocked, waiting threads quickly discover that '_auth_db' is
  # already fresh.
  with _auth_db_fetch_lock:
    # Not using cache at all (usually in tests) => always fetch.
    if not _process_cache_expiration_sec:
      return fetch_auth_db()

    cached = None
    with _auth_db_lock:
      if _auth_db is None:
        return _initialize_auth_db_cache()
      cached = _auth_db

    fetched = fetch_auth_db(known_auth_db=cached)

    with _auth_db_lock:
      return _roll_auth_db_cache(fetched)


def warmup():
  """Can be called from /_ah/warmup handler to precache authentication DB."""
  get_process_auth_db()


################################################################################
## AuthDB cache internal guts.


def _initialize_auth_db_cache():
  """Initializes auth runtime and _auth_db in particular.

  Must be called under _auth_db_lock.
  """
  global _auth_db
  global _auth_db_expiration

  assert _auth_db is None
  logging.info('Initial fetch of AuthDB')
  _auth_db = fetch_auth_db()
  _auth_db_expiration = time.time() + _process_cache_expiration_sec
  logging.info('Fetched AuthDB at rev %d', _auth_db.auth_db_rev)

  return _auth_db


def _roll_auth_db_cache(candidate):
  """Updates _auth_db if the given candidate AuthDB is fresher.

  Must be called under _auth_db_lock.
  """
  global _auth_db
  global _auth_db_expiration

  # This may happen after 'reset_local_state' call.
  if _auth_db is None:
    logging.info('Fetched AuthDB at rev %d', candidate.auth_db_rev)
    _auth_db = candidate
    _auth_db_expiration = time.time() + _process_cache_expiration_sec
    return _auth_db

  # This may happen when we switch the primary server the replica is linked to.
  # AuthDB revisions are not directly comparable in this case, so assume
  # 'candidate' is newer.
  if _auth_db.primary_id != candidate.primary_id:
    logging.info(
        'AuthDB primary changed %s (rev %d) -> %s (rev %d)',
        _auth_db.primary_id, _auth_db.auth_db_rev,
        candidate.primary_id, candidate.auth_db_rev)
    _auth_db = candidate
    _auth_db_expiration = time.time() + _process_cache_expiration_sec
    return _auth_db

  # Completely skip the update if the fetched version is older than what we
  # already have.
  if candidate.auth_db_rev < _auth_db.auth_db_rev:
    logging.info(
        'Someone else updated the cached AuthDB already '
        '(cached rev %d > fetched rev %d)',
        _auth_db.auth_db_rev, candidate.auth_db_rev)
    return _auth_db

  # Prefer to reuse the known copy if it matches the fetched one, it may have
  # some internal caches we want to keep. So update _auth_db only if candidate
  # is strictly fresher.
  if candidate.auth_db_rev > _auth_db.auth_db_rev:
    _auth_db = candidate
    logging.info(
        'Updated cached AuthDB: rev %d->%d',
        _auth_db.auth_db_rev, candidate.auth_db_rev)

  # Bump the expiration time even if the candidate's version is same as the
  # current cached one. We've just confirmed it is still fresh, we can keep
  # it cached for longer.
  _auth_db_expiration = time.time() + _process_cache_expiration_sec
  return _auth_db


################################################################################
## Group graph used by 'get_relevant_subgraph'.


class Graph(object):
  """Graph is directed multigraph with labeled edges and a designated root node.

  Nodes are assigned integer IDs and edges are stored as a map
  {node_from_id => label => node_to_id}. It simplifies serializing such graphs.

  Nodes must be comparable and hashable, since we use them as a dictionary keys.
  """

  # Note: exact values of labels end up in JSON API output, so change carefully.
  IN   = 'IN'    # edge A->B labeled 'IN' means 'A is subset of B'
  OWNS = 'OWNS'  # edge A->B labeled 'OWNS' means 'A owns B'

  def __init__(self):
    self._nodes = []        # list of all added nodes
    self._nodes_to_id = {}  # node object -> index of the node in _nodes
    self._root_id = None
    self._edges = collections.defaultdict(lambda: {
      self.IN: set(),
      self.OWNS: set(),
    })

  @property
  def root_id(self):
    return self._root_id

  @root_id.setter
  def root_id(self, node_id):
    assert node_id >= 0 and node_id < len(self._nodes)
    self._root_id = node_id

  def add_node(self, value):
    """Adds the given node (if not there).

    Returns:
      (Integer ID of the node, True if was added or False if existed before).
    """
    node_id = self._nodes_to_id.get(value)
    if node_id is not None:
      return node_id, False
    self._nodes_to_id[value] = node_id = len(self._nodes)
    self._nodes.append(value)
    return node_id, True

  def add_edge(self, from_node_id, relation, to_node_id):
    """Adds an edge (labeled by 'relation') between nodes given by their IDs."""
    assert from_node_id >= 0 and from_node_id < len(self._nodes)
    assert to_node_id >= 0 and to_node_id < len(self._nodes)
    assert relation in (self.IN, self.OWNS), relation
    self._edges[from_node_id][relation].add(to_node_id)

  def describe(self):
    """Yields pairs (node, edges from it) in order of node IDs.

    Nodes IDs are sequential, starting from 0. Edges are represented by a map
    {label -> set([to_node_id])}.
    """
    for i, node in enumerate(self._nodes):
      yield node, self._edges.get(i, {})


################################################################################
## Identity retrieval, @public and @require decorators.


def get_request_auth_db():
  """Returns instance of AuthDB from request-local cache.

  In a context of a single request this function always returns same
  instance of AuthDB. So as long as request runs, auth config stay consistent
  and don't change beneath your feet.

  Effectively request handler uses a snapshot of AuthDB at the moment request
  starts. If it somehow makes a call that initiates another request that uses
  AuthDB (via task queue, or UrlFetch) that another request may see a different
  copy of AuthDB.
  """
  return get_request_cache().auth_db


def get_current_identity():
  """Returns Identity associated with the current request.

  Takes into account delegation tokens, e.g. it can return end-user identity
  delegated to caller via delegation token. Use get_peer_identity() to get
  ID of a real caller, disregarding delegation.

  Always returns instance of Identity (that can be Anonymous, but never None).

  Returns non-Anonymous only if authentication context is properly initialized:
    * For webapp2, handlers must inherit from handlers.AuthenticatingHandler.
    * For Cloud Endpoints see endpoints_support.py.
  """
  return _get_current_identity()


def _get_current_identity():
  """Actual implementation of get_current_identity().

  Exists to be mocked, since original get_current_identity symbol is copied by
  value to 'auth' package scope, and mocking the identity would require mocking
  both 'auth.get_current_identity' and 'auth.api.get_current_identity'. It's
  simpler to move implementation to a private mockable function.
  """
  return get_request_cache().current_identity


def get_delegation_token():
  """Returns unwrapped validated delegation token used by this request.

  Services that accept the token may use them for additional authorization
  decisions. Please use extremely carefully, only when you control both sides
  of the delegation link and can guarantee that services involved understand
  the additional authorization limitations.
  """
  return get_request_cache().delegation_token


def get_peer_identity():
  """Returns Identity of whoever made the request (disregarding delegation).

  Always returns instance of Identity (that can be Anonymous, but never None).

  Returns non-Anonymous only if authentication context is properly initialized:
    * For webapp2 handlers must inherit from handlers.AuthenticatingHandler.
    * For Cloud Endpoints see endpoints_support.py.
  """
  return get_request_cache().peer_identity


def get_peer_ip():
  """Returns ipaddr.IP address of a peer that sent current request."""
  return get_request_cache().peer_ip


def get_auth_details():
  """Returns AuthDetails with extra information extracted from credentials."""
  return get_request_cache().auth_details


def is_group_member(group_name, identity=None):
  """Returns True if |identity| (or current identity if None) is in the group.

  Unknown groups are considered empty.
  """
  return get_request_cache().auth_db.is_group_member(
      group_name, identity or get_current_identity())


def is_admin(identity=None):
  """Returns True if |identity| (or current identity if None) is an admin.

  Admins are identities belonging to 'administrators' group
  (see model.ADMIN_GROUP). They have no relation to GAE notion of 'admin'.

  See 'is_superuser' for asserting GAE-level admin access.
  """
  return is_group_member(model.ADMIN_GROUP, identity)


def is_superuser():
  """Returns True if the current caller is GAE-level administrator.

  This works only for requests authenticated via GAE Users API or OAuth APIs.
  """
  return get_request_cache().auth_details.is_superuser


def list_group(group_name, recursive=True):
  """Returns all members, all globs and all nested groups in a group.

  The returned lists are unordered.

  Returns:
    GroupListing object.
  """
  return get_request_cache().auth_db.list_group(group_name, recursive)


def get_secret(secret_key):
  """Given an instance of SecretKey returns several last values of the secret.

  First item in the list is the current value of a secret (that can be used to
  validate and generate tokens), the rest are previous values (that can be used
  to validate older tokens, but shouldn't be used to create new ones).

  Creates a new secret if necessary.
  """
  return get_request_cache().auth_db.get_secret(secret_key)


def is_in_ip_whitelist(whitelist_name, ip, warn_if_missing=True):
  """Returns True if the given IP belongs to the given IP whitelist.

  Missing IP whitelists are considered empty.

  Args:
    whitelist_name: name of the IP whitelist (e.g. 'bots').
    ip: instance of ipaddr.IP.
    warn_if_missing: if True and IP whitelist is missing, logs a warning.
  """
  return get_request_cache().auth_db.is_in_ip_whitelist(
      whitelist_name, ip, warn_if_missing)


def verify_ip_whitelisted(identity, ip):
  """Verifies IP is in a whitelist assigned to the Identity.

  This check is used to restrict some callers to particular IP subnets as
  additional security measure.

  Args:
    identity: caller's identity.
    ip: instance of ipaddr.IP.

  Raises:
    AuthorizationError if identity has an IP whitelist assigned and given IP
    address doesn't belong to it.
  """
  get_request_cache().auth_db.verify_ip_whitelisted(identity, ip)


def public(func):
  """Decorator that marks a function as available for anonymous access.

  Useful only in a context of AuthenticatingHandler subclass to mark method as
  explicitly open for anonymous access. Without it AuthenticatingHandler will
  complain:

  class MyHandler(auth.AuthenticatingHandler):
    @auth.public
    def get(self):
      ....
  """
  # @require decorator sets __auth_require attribute.
  if hasattr(func, '__auth_require'):
    raise TypeError('Can\'t use @public and @require on a same function')
  func.__auth_public = True
  return func


def require(callback, error_msg=None):
  """Decorator that checks current identity's permissions.

  Args:
    callback: callback that is called without arguments and returns True
        to grant access to current identity (by calling decorated function) or
        False to forbid it (by raising AuthorizationError). It can
        use get_current_identity() (and other request state) to figure this out.
    error_msg: string that is included as the message in the AuthorizationError
        raised if callback returns False.

  Multiple @require decorators can be safely nested on top of each other to
  check multiple permissions. In that case a current identity needs to have all
  specified permissions to pass the check, i.e. permissions checks are combined
  using logical AND operation.

  It's safe to mix @require with NDB decorators such as @ndb.transactional.

  Usage example:

  class MyHandler(auth.AuthenticatingHandler):
    @auth.require(auth.is_admin)
    def get(self):
      ....
  """
  def decorator(func):
    # @public decorator sets __auth_public attribute.
    if hasattr(func, '__auth_public'):
      raise TypeError('Can\'t use @public and @require on same function')

    # When nesting multiple decorators the information (argspec, name) about
    # original function gets lost. __wrapped__ is used by NDB decorators
    # to preserve reference to original function. Use it too.
    original = getattr(func, '__wrapped__', func)

    @functools.wraps(func)
    def wrapper(*args, **kwargs):
      if not callback():
        raise AuthorizationError(error_msg)
      return func(*args, **kwargs)

    # Propagate reference to original function, mark function as decorated.
    wrapper.__wrapped__ = original
    wrapper.__auth_require = True

    return wrapper

  return decorator


def autologin(func):
  """Decorator that autologin anonymous users via the web UI.

  This is meant to to used on handlers that require a non-anonymous user via
  @require(), so that the user is not served a 403 simply because they didn't
  have the cookie set yet. Do not use this decorator on APIs or anything other
  than handlers that serve HTML.

  Usage example:

  class MyHandler(auth.AuthenticatingHandler):
    @auth.autologin
    @auth.require(auth.is_admin)
    def get(self):
      ....
  """
  # @public decorator sets __auth_public attribute.
  if hasattr(func, '__auth_public'):
    raise TypeError('Can\'t use @public and @autolgin on same function')
  # When nesting multiple decorators the information (argspec, name) about
  # original function gets lost. __wrapped__ is used by NDB decorators
  # to preserve reference to original function. Use it too.
  original = getattr(func, '__wrapped__', func)
  if original.__name__ != 'get':
    raise TypeError('Only get() can be set as autologin')

  @functools.wraps(func)
  def wrapper(self, *args, **kwargs):
    if not self.get_current_user():
      self.redirect(self.create_login_url(self.request.url))
      return
    try:
      return func(self, *args, **kwargs)
    except AuthorizationError:
      # Redirect to auth bootstrap page only if called by GAE-level admin
      # (only they are capable of running bootstrap), not already bootstrapped
      # (as approximated by is_admin returning False), and not on replica
      # (bootstrap works only on standalone or on primary).
      if not is_superuser() or is_admin() or model.is_replica():
        raise
      self.redirect(
          '/auth/bootstrap?r=%s' % urllib.quote_plus(self.request.path_qs))

  # Propagate reference to original function, mark function as decorated.
  wrapper.__wrapped__ = original
  wrapper.__auth_require = True

  return wrapper


def is_decorated(func):
  """Return True if |func| is decorated by @public or @require decorators."""
  return hasattr(func, '__auth_public') or hasattr(func, '__auth_require')
