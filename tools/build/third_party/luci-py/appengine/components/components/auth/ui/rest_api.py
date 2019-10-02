# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Auth management REST API."""

import base64
import functools
import logging
import textwrap
import urllib
import webapp2

from google.appengine.api import app_identity
from google.appengine.api import datastore_errors
from google.appengine.api import memcache
from google.appengine.datastore import datastore_query
from google.appengine.ext import ndb

from components import utils

from . import acl
from .. import api
from .. import change_log
from .. import handler
from .. import ipaddr
from .. import model
from .. import replication
from .. import signature
from .. import version
from ..proto import replication_pb2


# Set by set_config_locked.
_is_config_locked_cb = None


def get_rest_api_routes():
  """Return a list of webapp2 routes with auth REST API handlers."""
  assert model.GROUP_NAME_RE.pattern[0] == '^'
  group_re = model.GROUP_NAME_RE.pattern[1:]
  assert model.IP_WHITELIST_NAME_RE.pattern[0] == '^'
  ip_whitelist_re = model.IP_WHITELIST_NAME_RE.pattern[1:]
  return [
    webapp2.Route('/auth/api/v1/accounts/self', SelfHandler),
    webapp2.Route('/auth/api/v1/accounts/self/xsrf_token', XSRFHandler),
    webapp2.Route('/auth/api/v1/change_log', ChangeLogHandler),
    webapp2.Route('/auth/api/v1/groups', GroupsHandler),
    webapp2.Route('/auth/api/v1/groups/<name:%s>' % group_re, GroupHandler),
    webapp2.Route('/auth/api/v1/internal/replication', ReplicationHandler),
    webapp2.Route('/auth/api/v1/ip_whitelists', IPWhitelistsHandler),
    webapp2.Route(
        '/auth/api/v1/ip_whitelists/<name:%s>' % ip_whitelist_re,
        IPWhitelistHandler),
    webapp2.Route(
        '/auth/api/v1/listing/groups/<name:%s>' % group_re,
        GroupListingHandler),
    webapp2.Route('/auth/api/v1/memberships/list', MembershipsListHandler),
    webapp2.Route('/auth/api/v1/memberships/check', MembershipsCheckHandler),
    webapp2.Route('/auth/api/v1/subgraph/<principal:.*$>', SubgraphHandler),
    webapp2.Route('/auth/api/v1/suggest/groups', GroupsSuggestHandler),
    webapp2.Route('/auth/api/v1/server/certificates', CertificatesHandler),
    webapp2.Route('/auth/api/v1/server/info', ServerInfoHandler),
    webapp2.Route('/auth/api/v1/server/oauth_config', OAuthConfigHandler),
    webapp2.Route('/auth/api/v1/server/state', ServerStateHandler),
  ]


def forbid_api_on_replica(method):
  """Decorator for methods that are not allowed to be called on Replica.

  If such method is called on a service in Replica mode, it would return
  HTTP 405 "Method Not Allowed".
  """
  @functools.wraps(method)
  def wrapper(self, *args, **kwargs):
    assert isinstance(self, webapp2.RequestHandler)
    if model.is_replica():
      self.abort(
          405,
          json={
            'primary_url': model.get_replication_state().primary_url,
            'text': 'Use Primary service for API requests',
          },
          headers={
            'Content-Type': 'application/json; charset=utf-8',
          })
    return method(self, *args, **kwargs)
  return wrapper


def is_config_locked():
  """Returns True to forbid configuration changing API calls.

  If is_config_locked returns True API requests that change configuration will
  return HTTP 409 error.

  A configuration is subset of AuthDB that changes infrequently:
  * OAuth client_id whitelist
  * IP whitelist

  Used by auth_service that utilizes config_service for config management.
  """
  return _is_config_locked_cb() if _is_config_locked_cb else False


def set_config_locked(locked_callback):
  """Sets a function that returns True if configuration is locked."""
  global _is_config_locked_cb
  _is_config_locked_cb = locked_callback


def _is_no_cache(request):
  """Returns True if the request should skip the cache."""
  cache_control = request.headers.get('Cache-Control') or ''
  return 'no-cache' in cache_control or 'max-age=0' in cache_control


def _get_maybe_cached_auth_db(request):
  """Returns cached auth DB unless the request has no cache header."""
  if _is_no_cache(request):
    return api.get_latest_auth_db()
  return api.get_request_auth_db()


class EntityOperationError(Exception):
  """Raised by do_* methods in EntityHandlerBase to indicate a conflict."""
  def __init__(self, message, details=None):
    super(EntityOperationError, self).__init__(message)
    self.message = message
    self.details = details


class EntityHandlerBase(handler.ApiHandler):
  """Handler for creating, reading, updating and deleting an entity in AuthDB.

  Implements optimistic concurrency control based on Last-Modified header.

  Subclasses must override class methods (see below). Entities being manipulated
  should implement datastore_utils.SerializableModelMixin and
  model.AuthVersionedEntityMixin and have following properties:
    created_by
    created_ts
    modified_by
    modified_ts

  GET handler is available in Standalone, Primary and Replica modes.

  Everything else is available only in Standalone and Primary modes.
  """

  # Root URL of this handler, e.g. '/auth/api/v1/groups/'.
  entity_url_prefix = None
  # Entity class being operated on, e.g. model.AuthGroup.
  entity_kind = None
  # Will show up as a key in response dicts, e.g. {<name>: ...}.
  entity_kind_name = None
  # Will show up in error messages, e.g. 'Failed to delete <title>'.
  entity_kind_title = None

  def check_preconditions(self):
    """Called after initial has_access checks, but before actual handling."""

  @classmethod
  def get_entity_key(cls, name):
    """Returns ndb.Key corresponding to entity with given name."""
    raise NotImplementedError()

  @classmethod
  def is_entity_writable(cls, _name):
    """Returns True if entity can be modified (e.g. not an external group)."""
    return True

  @classmethod
  def entity_to_dict(cls, entity):
    """Converts an entity to a serializable dictionary."""
    return entity.to_serializable_dict(with_id_as='name')

  @classmethod
  def do_get(cls, name, request):  # pylint: disable=unused-argument
    """Returns an entity given its name or None if no such entity.

    Args:
      name: name of the entity to fetch (use get_entity_key to convert to key).
      request: webapp2.Request object.
    """
    return cls.get_entity_key(name).get()

  @classmethod
  def can_create(cls):
    """True if caller is allowed to create a new entity."""
    return acl.is_admin()

  @classmethod
  def do_create(cls, entity):
    """Called in transaction to validate and put a new entity.

    Raises:
      EntityOperationError in case of a conflict.
    """
    raise NotImplementedError()

  @classmethod
  def can_update(cls, entity):  # pylint: disable=unused-argument
    """True if caller is allowed to update a given entity."""
    return acl.is_admin()

  @classmethod
  def do_update(cls, entity, params):
    """Called in transaction to update existing entity.

    Raises:
      EntityOperationError in case of a conflict.
    """
    raise NotImplementedError()

  @classmethod
  def can_delete(cls, entity):  # pylint: disable=unused-argument
    """True if caller is allowed to delete a given entity."""
    return acl.is_admin()

  @classmethod
  def do_delete(cls, entity):
    """Called in transaction to delete existing entity.

    Raises:
      EntityOperationError in case of a conflict.
    """
    raise NotImplementedError()

  # Actual handlers implemented in terms of do_* calls.

  @api.require(acl.has_access)
  def get(self, name):
    """Fetches entity give its name."""
    self.check_preconditions()
    obj = self.do_get(name, self.request)
    if not obj:
      self.abort_with_error(404, text='No such %s' % self.entity_kind_title)
    self.send_response(
        response={self.entity_kind_name: self.entity_to_dict(obj)},
        headers={'Last-Modified': utils.datetime_to_rfc2822(obj.modified_ts)})

  @forbid_api_on_replica
  @api.require(acl.has_access)
  def post(self, name):
    """Creates a new entity, ensuring it's indeed new (no overwrites)."""
    self.check_preconditions()
    try:
      body = self.parse_body()
      name_in_body = body.pop('name', None)
      if not name_in_body or name_in_body != name:
        raise ValueError('Missing or mismatching name in request body')
      if not self.is_entity_writable(name):
        raise ValueError('This %s is not writable' % self.entity_kind_title)
      entity = self.entity_kind.from_serializable_dict(
          serializable_dict=body,
          key=self.get_entity_key(name),
          created_ts=utils.utcnow(),
          created_by=api.get_current_identity())
    except (TypeError, ValueError) as e:
      self.abort_with_error(400, text=str(e))

    # No need to enter a transaction (like in do_update) to check this.
    if not self.can_create():
      raise api.AuthorizationError(
          '"%s" has no permission to create a %s' %
          (api.get_current_identity().to_bytes(), self.entity_kind_title))

    @ndb.transactional
    def create(entity):
      if entity.key.get():
        return False, {
          'http_code': 409,
          'text': 'Such %s already exists' % self.entity_kind_title,
        }
      entity.record_revision(
          modified_by=api.get_current_identity(),
          modified_ts=utils.utcnow(),
          comment='REST API')
      try:
        self.do_create(entity)
      except EntityOperationError as exc:
        return False, {
          'http_code': 409,
          'text': exc.message,
          'details': exc.details,
        }
      except ValueError as exc:
        return False, {
          'http_code': 400,
          'text': str(exc),
        }
      model.replicate_auth_db()
      return True, None

    success, error_details = create(entity)
    if not success:
      self.abort_with_error(**error_details)
    self.send_response(
        response={'ok': True},
        http_code=201,
        headers={
          'Last-Modified': utils.datetime_to_rfc2822(entity.modified_ts),
          'Location':
              '%s%s' % (self.entity_url_prefix, urllib.quote(entity.key.id())),
        }
    )

  @forbid_api_on_replica
  @api.require(acl.has_access)
  def put(self, name):
    """Updates an existing entity."""
    self.check_preconditions()
    try:
      body = self.parse_body()
      name_in_body = body.pop('name', None)
      if not name_in_body or name_in_body != name:
        raise ValueError('Missing or mismatching name in request body')
      if not self.is_entity_writable(name):
        raise ValueError('This %s is not writable' % self.entity_kind_title)
      entity_params = self.entity_kind.convert_serializable_dict(body)
    except (TypeError, ValueError) as e:
      self.abort_with_error(400, text=str(e))

    @ndb.transactional
    def update(params, expected_ts):
      entity = self.get_entity_key(name).get()
      if not entity:
        return None, None, {
          'http_code': 404,
          'text': 'No such %s' % self.entity_kind_title,
        }
      if (expected_ts and
          utils.datetime_to_rfc2822(entity.modified_ts) != expected_ts):
        return None, None, {
          'http_code': 412,
          'text':
              '%s was modified by someone else' %
              self.entity_kind_title.capitalize(),
        }
      if not self.can_update(entity):
        # Raising from inside a transaction produces ugly logs. Just return the
        # exception to be raised outside.
        ident = api.get_current_identity()
        exc = api.AuthorizationError(
            '"%s" has no permission to update %s "%s"' %
            (ident.to_bytes(), self.entity_kind_title, name))
        return None, exc, None
      entity.record_revision(
          modified_by=api.get_current_identity(),
          modified_ts=utils.utcnow(),
          comment='REST API')
      try:
        self.do_update(entity, params)
      except EntityOperationError as exc:
        return None, None, {
          'http_code': 409,
          'text': exc.message,
          'details': exc.details,
        }
      except ValueError as exc:
        return None, None, {
          'http_code': 400,
          'text': str(exc),
        }
      model.replicate_auth_db()
      return entity, None, None

    entity, exc, error_details = update(
        entity_params, self.request.headers.get('If-Unmodified-Since'))
    if exc:
      raise exc  # pylint: disable=raising-bad-type
    if not entity:
      self.abort_with_error(**error_details)
    self.send_response(
        response={'ok': True},
        http_code=200,
        headers={
          'Last-Modified': utils.datetime_to_rfc2822(entity.modified_ts),
        }
    )

  @forbid_api_on_replica
  @api.require(acl.has_access)
  def delete(self, name):
    """Deletes an entity."""
    self.check_preconditions()
    if not self.is_entity_writable(name):
      self.abort_with_error(
          400, text='This %s is not writable' % self.entity_kind_title)

    @ndb.transactional
    def delete(expected_ts):
      entity = self.get_entity_key(name).get()
      if not entity:
        if expected_ts:
          return None, {
            'http_code': 412,
            'text':
                '%s was deleted by someone else' %
                self.entity_kind_title.capitalize(),
          }
        else:
          # Unconditionally deleting it, and it's already gone -> success.
          return None, None
      if (expected_ts and
          utils.datetime_to_rfc2822(entity.modified_ts) != expected_ts):
        return None, {
          'http_code': 412,
          'text':
              '%s was modified by someone else' %
              self.entity_kind_title.capitalize(),
        }
      if not self.can_delete(entity):
        # Raising from inside a transaction produces ugly logs. Just return the
        # exception to be raised outside.
        ident = api.get_current_identity()
        exc = api.AuthorizationError(
            '"%s" has no permission to delete %s "%s"' %
            (ident.to_bytes(), self.entity_kind_title, name))
        return exc, None
      entity.record_deletion(
          modified_by=api.get_current_identity(),
          modified_ts=utils.utcnow(),
          comment='REST API')
      try:
        self.do_delete(entity)
      except EntityOperationError as exc:
        return None, {
          'http_code': 409,
          'text': exc.message,
          'details': exc.details,
        }
      model.replicate_auth_db()
      return None, None

    exc, error_details = delete(self.request.headers.get('If-Unmodified-Since'))
    if exc:
      raise exc  # pylint: disable=raising-bad-type
    if error_details:
      self.abort_with_error(**error_details)
    self.send_response({'ok': True})


class SelfHandler(handler.ApiHandler):
  """Returns identity of a caller and authentication related request properties.

  Available in Standalone, Primary and Replica modes.
  """

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'doc':
        'Returns identity of a caller based on passed authentication tokens, '
        'as well as requester\'s IP address (as seen by AppEngine). Useful '
        'when debugging authentication issues.',
      'response_type': 'Self info',
    },
  ]

  @api.public
  def get(self):
    self.send_response({
      'identity': api.get_current_identity().to_bytes(),
      'ip': ipaddr.ip_to_string(api.get_peer_ip()),
    })


class XSRFHandler(handler.ApiHandler):
  """Generates XSRF token on demand.

  Should be used only by client scripts or Ajax calls. Requires header
  'X-XSRF-Token-Request' to be present (actual value doesn't matter).

  Available in Standalone, Primary and Replica modes.
  """

  # Don't enforce prior XSRF token, it might not be known yet.
  xsrf_token_enforce_on = ()

  @handler.require_xsrf_token_request
  @api.public
  def post(self):
    token = self.generate_xsrf_token()
    self.send_response(
        {
          'expiration_sec': handler.XSRFToken.expiration_sec,
          'xsrf_token': token,
        })


class ChangeLogHandler(handler.ApiHandler):
  """Returns AuthDBChange log entries matching some query.

  Supported query parameters (with example):
    target='AuthGroup$A group' - limit changes to given target only (if given).
    auth_db_rev=123 - limit changes to given revision only (if given).
    limit=50 - how many changes to return in a single page (default: 50).
    cursor=.... - urlsafe datastore cursor for pagination.
  """

  # The list of indexes here is synchronized with auth_service/index.yaml.
  NEED_INDEX_ERROR_MESSAGE = textwrap.dedent(r"""
  Your GAE app doesn't have indexes required for "Change log" functionality.

  If you need this feature, add following indexes to index.yaml. You can do it
  any time: changes are collected, they are just not queriable until indexed.

  - kind: AuthDBChange
    ancestor: yes
    properties:
    - name: target
    - name: __key__
      direction: desc

  - kind: AuthDBChange
    ancestor: yes
    properties:
    - name: __key__
      direction: desc
  """).strip()

  @forbid_api_on_replica
  @api.require(acl.has_access)
  def get(self):
    target = self.request.get('target')
    if target and not change_log.TARGET_RE.match(target):
      self.abort_with_error(400, text='Invalid \'target\' param')

    auth_db_rev = self.request.get('auth_db_rev')
    if auth_db_rev:
      try:
        auth_db_rev = int(auth_db_rev)
        if auth_db_rev <= 0:
          raise ValueError('Outside of allowed range')
      except ValueError as exc:
        self.abort_with_error(
            400, text='Invalid \'auth_db_rev\' param: %s' % exc)

    try:
      limit = int(self.request.get('limit', 50))
      if limit <= 0 or limit > 1000:
        raise ValueError('Outside of allowed range')
    except ValueError as exc:
      self.abort_with_error(400, text='Invalid \'limit\' param: %s' % exc)

    try:
      cursor = datastore_query.Cursor(urlsafe=self.request.get('cursor'))
    except (datastore_errors.BadValueError, ValueError) as exc:
      self.abort_with_error(400, text='Invalid \'cursor\' param: %s' % exc)

    q = change_log.make_change_log_query(target=target, auth_db_rev=auth_db_rev)
    try:
      changes, cursor, more = q.fetch_page(limit, start_cursor=cursor)
    except datastore_errors.NeedIndexError:
      # This is expected for users of components.auth that did not update
      # index.yaml. Return a friendlier message pointing them to instructions.
      self.abort_with_error(500, text=self.NEED_INDEX_ERROR_MESSAGE)

    self.send_response({
      'changes': [c.to_jsonish() for c in changes],
      'cursor': cursor.urlsafe() if cursor and more else None,
    })


def caller_can_modify(group_dict):
  """True if given group (presented as dict) is modifiable by a caller."""
  if model.is_external_group_name(group_dict['name']):
    return False
  return api.is_admin() or api.is_group_member(group_dict['owners'])


class GroupsHandler(handler.ApiHandler):
  """Lists all registered groups.

  Returns a list of groups, sorted by name. Each entry in a list is a dict with
  all details about the group except the actual list of members
  (which may be large).

  Available in Standalone, Primary and Replica modes.
  """

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'doc': 'Lists names and descriptions of all known groups.',
      'response_type': 'Groups',
    },
  ]

  @staticmethod
  def cache_key(auth_db_rev):
    return 'api:v1:GroupsHandler/%d' % auth_db_rev

  @staticmethod
  def adjust_response_for_user(response):
    """Modifies response (in place) based on user ACLs."""
    for g in response['groups']:
      g['caller_can_modify'] = caller_can_modify(g)

  @api.require(acl.has_access)
  def get(self):
    # Try to find a cached response for the current revision.
    auth_db_rev = model.get_auth_db_revision()
    cached_response = memcache.get(self.cache_key(auth_db_rev))
    if cached_response is not None:
      self.adjust_response_for_user(cached_response)
      self.send_response(cached_response)
      return

    # Grab a list of groups and corresponding revision for cache key.
    def run():
      fut = model.AuthGroup.query(ancestor=model.root_key()).fetch_async()
      return model.get_auth_db_revision(), fut.get_result()
    auth_db_rev, group_list = ndb.transaction(run)

    # Currently AuthGroup entity contains a list of group members in the entity
    # body. It's an implementation detail that should not be relied upon.
    # Generally speaking, fetching a list of group members can be an expensive
    # operation, and group listing call shouldn't do it all the time. So throw
    # away all fields that enumerate group members.
    response = {
      'groups': [
        g.to_serializable_dict(
            with_id_as='name',
            exclude=('globs', 'members', 'nested'))
        for g in sorted(group_list, key=lambda x: x.key.string_id())
      ],
    }
    memcache.set(self.cache_key(auth_db_rev), response, time=24*3600)
    self.adjust_response_for_user(response)
    self.send_response(response)


class GroupHandler(EntityHandlerBase):
  """Creating, reading, updating and deleting a single group.

  GET is available in Standalone, Primary and Replica modes.
  Everything else is available only in Standalone and Primary modes.
  """
  entity_url_prefix = '/auth/api/v1/groups/'
  entity_kind = model.AuthGroup
  entity_kind_name = 'group'
  entity_kind_title = 'group'

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'doc': 'Returns a group given its name. Doesn\'t expand nested groups.',
      'response_type': 'Group',
    },
    {
      'verb': 'POST',
      'doc': 'Creates a new group (ensuring it is indeed new).',
      'request_type': 'Group',
      'response_type': 'Status',
    },
    {
      'verb': 'PUT',
      'doc':
        'Updates an existing group. Use If-Unmodified-Since header to '
        'avoid unintentional overwrites.',
      'request_type': 'Group',
      'response_type': 'Status',
    },
    {
      'verb': 'DELETE',
      'doc':
        'Deletes an existing group. Use If-Unmodified-Since header to '
        'avoid unintentional removals.',
      'response_type': 'Status',
    },
  ]

  @classmethod
  def get_entity_key(cls, name):
    assert model.is_valid_group_name(name), name
    return model.group_key(name)

  @classmethod
  def is_entity_writable(cls, name):
    return not model.is_external_group_name(name)

  @classmethod
  def entity_to_dict(cls, entity):
    g = super(GroupHandler, cls).entity_to_dict(entity)
    g['caller_can_modify'] = caller_can_modify(g)
    return g

  @classmethod
  def do_get(cls, name, request):
    # Use in-memory cache by default. It can be slightly stale, but serving from
    # it is extra fast (0 RPCs). Use datastore if explicitly asked to bypass
    # the cache. Direct datastore reads are used mostly by admin UI.
    if _is_no_cache(request):
      return super(GroupHandler, cls).do_get(name, request)
    return api.get_request_auth_db().get_group(name)

  # Same as in the base class, repeated here just for clarity.
  @classmethod
  def can_create(cls):
    return acl.is_admin()

  @classmethod
  def do_create(cls, entity):
    # Check that all references group (owning group, nested groups) exist. It is
    # ok for a new group to have itself as an owner.
    entity.owners = entity.owners or model.ADMIN_GROUP
    to_check = list(entity.nested)
    if entity.owners != entity.key.id() and entity.owners not in to_check:
      to_check.append(entity.owners)
    missing = model.get_missing_groups(to_check)
    if missing:
      raise EntityOperationError(
          message=
              'Some referenced groups don\'t exist: %s.' % ', '.join(missing),
          details={'missing': missing})
    entity.put()

  @classmethod
  def can_update(cls, entity):
    return acl.is_admin() or api.is_group_member(entity.owners)

  @classmethod
  def do_update(cls, entity, params):
    # If changing an owner, ensure new owner exists. No need to do it if
    # the group owns itself (we know it exists).
    new_owners = params.get('owners', entity.owners)
    if new_owners != entity.owners and new_owners != entity.key.id():
      ent = model.group_key(new_owners).get()
      if not ent:
        raise EntityOperationError(
            message='Owners groups (%s) doesn\'t exist.' % new_owners,
            details={'missing': [new_owners]})
    # Admin group must be owned by itself.
    if entity.key.id() == model.ADMIN_GROUP and new_owners != model.ADMIN_GROUP:
      raise EntityOperationError(
          message='Can\'t change owner of \'%s\' group.' % model.ADMIN_GROUP)
    # If adding new nested groups, need to ensure they exist.
    added_nested_groups = None
    if 'nested' in params:
      added_nested_groups = set(params['nested']) - set(entity.nested)
      if added_nested_groups:
        missing = model.get_missing_groups(added_nested_groups)
        if missing:
          raise EntityOperationError(
              message=
                  'Some referenced groups don\'t exist: %s.'
                  % ', '.join(missing),
              details={'missing': missing})
    # Now make sure updated group is not a part of new group dependency cycle.
    entity.populate(**params)
    if added_nested_groups:
      cycle = model.find_group_dependency_cycle(entity)
      if cycle:
        # Make it clear that cycle starts from the group being modified.
        cycle = [entity.key.id()] + cycle
        as_str = ' -> '.join(cycle)
        raise EntityOperationError(
            message='Groups can not have cyclic dependencies: %s.' % as_str,
            details={'cycle': cycle})
    # TODO(vadimsh): Temporary forbid using 'project:...' identities in groups.
    # They are not safe to be added to groups until all services that consume
    # AuthDB understand them. Many services do an overzealous validation of
    # AuthDB pushes and totally reject them if AuthDB contains some unrecognized
    # identity kinds.
    if any(m.is_project for m in entity.members):
      raise EntityOperationError(
          message='"project:..." identities aren\'t allowed in groups yet')
    # Good enough.
    entity.put()

  @classmethod
  def can_delete(cls, entity):
    return acl.is_admin() or api.is_group_member(entity.owners)

  @classmethod
  def do_delete(cls, entity):
    # Admin group is special, deleting it would be bad.
    if entity.key.id() == model.ADMIN_GROUP:
      raise EntityOperationError(
          message='Can\'t delete \'%s\' group.' % model.ADMIN_GROUP)
    # A group can be its own owner (but it can not "nest" itself, as checked by
    # find_group_dependency_cycle). It is OK to delete a self-owning group.
    referencing_groups = model.find_referencing_groups(entity.key.id())
    referencing_groups.discard(entity.key.id())
    if referencing_groups:
      grs = sorted(referencing_groups)
      raise EntityOperationError(
          message=(
              'This group is being referenced by other groups: %s.' %
                  ', '.join(grs)),
          details={'groups': grs})
    entity.key.delete()


class ReplicationHandler(handler.AuthenticatingHandler):
  """Accepts AuthDB push from Primary."""

  # Handler uses X-Appengine-Inbound-Appid header protected by GAE.
  xsrf_token_enforce_on = ()

  def send_response(self, response):
    """Sends serialized ReplicationPushResponse as a response."""
    assert isinstance(response, replication_pb2.ReplicationPushResponse)
    self.response.headers['Content-Type'] = 'application/octet-stream'
    self.response.write(response.SerializeToString())

  def send_error(self, error_code):
    """Sends ReplicationPushResponse with fatal error as a response."""
    response = replication_pb2.ReplicationPushResponse()
    response.status = replication_pb2.ReplicationPushResponse.FATAL_ERROR
    response.error_code = error_code
    response.auth_code_version = version.__version__
    self.send_response(response)

  # Check that request came from some GAE app. More thorough check is inside.
  @api.require(lambda: api.get_current_identity().is_service)
  def post(self):
    # Check that current service is a Replica.
    if not model.is_replica():
      self.send_error(replication_pb2.ReplicationPushResponse.NOT_A_REPLICA)
      return

    # Check that request came from expected Primary service.
    expected_ident = model.Identity(
        model.IDENTITY_SERVICE, model.get_replication_state().primary_id)
    if api.get_current_identity() != expected_ident:
      self.send_error(replication_pb2.ReplicationPushResponse.FORBIDDEN)
      return

    # Check the signature headers are present.
    key_name = self.request.headers.get('X-AuthDB-SigKey-v1')
    sign = self.request.headers.get('X-AuthDB-SigVal-v1')
    if not key_name or not sign:
      self.send_error(replication_pb2.ReplicationPushResponse.MISSING_SIGNATURE)
      return

    # Verify the signature.
    body = self.request.body
    sign = base64.b64decode(sign)
    if not replication.is_signed_by_primary(body, key_name, sign):
      self.send_error(replication_pb2.ReplicationPushResponse.BAD_SIGNATURE)
      return

    # Deserialize the request, check it is valid.
    request = replication_pb2.ReplicationPushRequest.FromString(body)
    if not request.revision or not request.HasField('auth_db'):
      self.send_error(replication_pb2.ReplicationPushResponse.BAD_REQUEST)
      return

    # Handle it.
    logging.info('Received AuthDB push: rev %d', request.revision.auth_db_rev)
    if request.auth_code_version:
      logging.info(
          'Primary\'s auth component version: %s', request.auth_code_version)
    applied, state = replication.push_auth_db(request.revision, request.auth_db)
    logging.info(
        'AuthDB push %s: rev is %d',
        'applied' if applied else 'skipped', state.auth_db_rev)

    # Send the response.
    response = replication_pb2.ReplicationPushResponse()
    if applied:
      response.status = replication_pb2.ReplicationPushResponse.APPLIED
    else:
      response.status = replication_pb2.ReplicationPushResponse.SKIPPED
    response.current_revision.primary_id = state.primary_id
    response.current_revision.auth_db_rev = state.auth_db_rev
    response.current_revision.modified_ts = utils.datetime_to_timestamp(
        state.modified_ts)
    response.auth_code_version = version.__version__
    self.send_response(response)


class IPWhitelistsHandler(handler.ApiHandler):
  """Lists all IP whitelists.

  Available in Standalone, Primary and Replica modes. Replicas only have IP
  whitelists referenced in "account -> IP whitelist" mapping.
  """

  @api.require(acl.has_access)
  def get(self):
    entities = model.AuthIPWhitelist.query(ancestor=model.root_key())
    self.send_response({
      'ip_whitelists': [
        e.to_serializable_dict(with_id_as='name')
        for e in sorted(entities, key=lambda x: x.key.id())
      ],
    })


class IPWhitelistHandler(EntityHandlerBase):
  """Creating, reading, updating and deleting a single IP whitelist.

  GET is available in Standalone, Primary and Replica modes.
  Everything else is available only in Standalone and Primary modes.
  """
  entity_url_prefix = '/auth/api/v1/ip_whitelists/'
  entity_kind = model.AuthIPWhitelist
  entity_kind_name = 'ip_whitelist'
  entity_kind_title = 'ip whitelist'

  def check_preconditions(self):
    if self.request.method != 'GET' and is_config_locked():
      self.abort_with_error(409, text='The configuration is managed elsewhere')

  @classmethod
  def get_entity_key(cls, name):
    assert model.is_valid_ip_whitelist_name(name), name
    return model.ip_whitelist_key(name)

  @classmethod
  def do_create(cls, entity):
    entity.put()

  @classmethod
  def do_update(cls, entity, params):
    entity.populate(**params)
    entity.put()

  @classmethod
  def do_delete(cls, entity):
    # TODO(vadimsh): Verify it isn't being referenced by whitelist assigments.
    entity.key.delete()


class PerIdentityBatchHandler(handler.ApiHandler):
  """A class with the POST handler being the batch version of the GET handler.

  GET handler accepts 'identity=...' query parameter.
  POST handler accepts {'per_identity': {<ident>: <parameters>}} dict.

  Subclasses must override 'collect_get_params', 'validate_params' and
  'execute_batch'.
  """

  # POST here is not state-modifying, no need for XSRF token.
  xsrf_token_enforce_on = ()

  def collect_get_params(self):
    """Examines GET query and returns a dict with them.

    The format of the dict must match what is supposed to be passed as a value
    in 'per_identity' dict in POST body. So POST body is essentially
    a collection of GET queries to be executed as a batch.
    """
    raise NotImplementedError()

  def validate_params(self, params):
    """Takes a dict with some single query parameters and validates it.

    Raises ValueError if parameters are invalid.
    """
    raise NotImplementedError()

  def execute_batch(self, queries):
    """Takes a dict {Identity => params dict} and returns {Identity => results}.

    Parameters are already validated at this point.
    """
    raise NotImplementedError()

  @api.require(acl.has_access)
  def post(self):
    self.send_response(self._handle_batch(self.parse_body()))

  @api.require(acl.has_access)
  def get(self):
    ident = self.request.get('identity')
    if not ident:
      self.abort_with_error(400, text='"identity" query parameter is required')
    try:
      model.Identity.from_bytes(ident)
    except ValueError as e:
      self.abort_with_error(400, text='Invalid "identity" - %s' % e)

    # Make the "batch" call with the single request.
    resp = self._handle_batch({
      'per_identity': {
        ident: self.collect_get_params(),
      },
    })

    # Extract back singular response.
    per_ident = resp.get('per_identity')
    assert ident in per_ident, resp
    self.send_response(per_ident[ident])

  def _handle_batch(self, body):
    # 'per_identity' is a dict {identity => request parameters},
    if not isinstance(body, dict):
      self.abort_with_error(400, text='The body must be a dict')
    per_identity = body.get('per_identity', None)
    if not isinstance(per_identity, dict) or not per_identity:
      self.abort_with_error(400, text='"per_identity" must be a non-empty dict')

    # Validate individual queries.
    queries = {}
    for ident_str, params in per_identity.iteritems():
      try:
        ident = model.Identity.from_bytes(ident_str)
      except ValueError as e:
        self.abort_with_error(
            400, text='Not a valid identity %r - %s' % (ident_str, e))
      # Make sure 'ident' serializes back to 'ident_str'. This is important,
      # since we use it as key in the response, and callers most likely will be
      # searching for exact same value they pass in the query.
      assert ident.to_bytes() == ident_str, (ident, ident_str)
      if params is None:
        params = {}
      try:
        if not isinstance(params, dict):
          raise ValueError('parameters must be specified as a dict')
        self.validate_params(params)
      except ValueError as e:
        self.abort_with_error(400, text='When querying %s: %s' % (ident_str, e))
      queries[ident] = params

    return {
      'per_identity': {
        ident.to_bytes(): res
        for ident, res in self.execute_batch(queries).iteritems()
      },
    }


class GroupListingHandler(handler.ApiHandler):
  """Lists all members of a group, recursively."""

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'doc': 'Lists all members of a group, expanding subgroups.',
      'response_type': 'Group listing',
    },
  ]

  @api.require(acl.has_access)
  def get(self, name):
    if not model.is_valid_group_name(name):
      self.abort_with_error(400, text='Invalid group name')

    # By default use cached auth DB. Switch to latest one only if No-Cache
    # header is given.
    listing = _get_maybe_cached_auth_db(self.request).list_group(name)

    self.send_response({
      'listing': {
        'members': [{'principal': m.to_bytes()} for m in listing.members],
        'globs': [{'principal': g.to_bytes()} for g in listing.globs],
        'nested': [{'principal': n} for n in listing.nested],
      },
    })


class MembershipsListHandler(PerIdentityBatchHandler):
  """Lists all groups a user belongs to."""

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'params': 'identity=...',

      'doc':
        'Returns a list of groups an identity belongs to (including all '
        'transitive relations) as a list of memberships.',

      'response_type': {
        'name': 'Membership list',
        'doc': 'Represents a list of groups some identity is a member of.',
        'example': {
          'memberships': [
            {'group': 'Group name'},
            {'group': 'Another group name'},
          ],
        },
      },
    },

    {
      'verb': 'POST',

      'doc':
        'A batch version of the membership listing call. Executes multiple '
        'queries for multiple identities in parallel.',

      'request_type': {
        'name': 'Batch listing request',
        'doc':
          'A request to query a list of groups of multiple identities in '
          'parallel. Per-identity dict values are options for membership '
          'listing (there are currently none, so pass null or {}).',
        'example': {
          'per_identity': {
            'user:someone@example.com': None,
            'user:someone_else@example.com': None,
          },
        },
      },

      'response_type': {
        'name': 'Batch listing response',
        'doc':
          'For each identity specifies a list of groups it is a member of '
          '(in the same format as non-batched version).',
        'example': {
          'per_identity': {
            'user:someone@example.com': {
              'memberships': [
                {'group': 'Group name'},
                {'group': 'Another group name'},
              ],
            },
            'user:someone_else@example.com': {
              'memberships': [
                {'group': 'Group name'},
                {'group': 'Another group name'},
              ],
            },
          },
        },
      },
    },
  ]

  def collect_get_params(self):
    # No parameters for now.
    return {}

  def validate_params(self, params):
    # No parameters for now.
    pass

  def execute_batch(self, queries):
    # Since we currently have all groups data in memory, doing queries truly
    # in parallel will only hurt, since we have only one CPU and there's no IO.
    auth_db = api.get_request_cache().auth_db
    resp = {}
    for ident in queries:
      resp[ident] = {
        'memberships': [
          {'group': g} for g in sorted(auth_db.fetch_groups_with_member(ident))
        ],
      }
    return resp


class MembershipsCheckHandler(PerIdentityBatchHandler):
  """Checks whether an identity belongs to any of given groups."""

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'params': 'identity=...&groups=...',

      'doc':
        'Checks whether a user belongs to any of given groups (provided via '
        '"groups" query parameter that can be specified multiple times).',

      'response_type': {
        'name': 'Check response',
        'doc':
          'Indicates whether the identity is a member of any of the groups '
          'specified in the request.',
        'example': {
          'is_member': True,
        }
      },
    },

    {
      'verb': 'POST',

      'doc':
        'A batch version of the membership check call. Executes multiple '
        'checks for multiple identities in parallel.',

      'request_type': {
        'name': 'Batch check request',
        'doc':
          'Represents a request to check memberships of multiple identities in '
          'parallel.',
        'example': {
          'per_identity': {
            'user:someone@example.com': {
              'groups': ['Group A', 'Group B'],
            },
            'user:someone_else@example.com': {
              'groups': ['Group C', 'Group D'],
            },
          },
        },
      },

      'response_type': {
        'name': 'Batch check response',
        'doc':
          'For each queried identity specifies whether it is a member of any '
          'of the groups specified in the request for this identity.',
        'example': {
          'per_identity': {
            'user:someone@example.com': {
              'is_member': True,
            },
            'user:someone_else@example.com': {
              'is_member': False,
            },
          },
        },
      }
    },
  ]

  def collect_get_params(self):
    groups = self.request.GET.getall('groups')
    if not groups:
      self.abort_with_error(400, text='"groups" query parameter is required')
    return {'groups': groups}

  def validate_params(self, params):
    groups = params.get('groups')
    if not isinstance(groups, list) or not groups:
      raise ValueError('must specify a non-empty list of groups to check')
    for g in groups:
      if not isinstance(g, basestring):
        raise ValueError('not a group name %r' % (g,))

  def execute_batch(self, queries):
    # Since we currently have all groups data in memory, doing queries truly
    # in parallel will only hurt, since we have only one CPU and there's no IO.
    auth_db = api.get_request_cache().auth_db
    resp = {}
    for iden, p in queries.iteritems():
      assert isinstance(p['groups'], list)
      resp[iden] = {
        'is_member': any(auth_db.is_group_member(g, iden) for g in p['groups']),
      }
    return resp


class SubgraphHandler(handler.ApiHandler):
  """Returns groups that include this principal and are owned by it."""

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'doc': 'Returns groups that include this principal and are owned by it.',
      'response_type': 'Group subgraph',
    },
  ]

  @api.require(acl.has_access)
  def get(self, principal):
    # Guess the principal type. All globs necessarily have '*' and ':', and all
    # identities necessarily have ':' (but don't have '*'). Group names are
    # forbidden to have '*' or ':'.
    try:
      if '*' in principal:
        principal = model.IdentityGlob.from_bytes(principal)
      elif ':' in principal:
        principal = model.Identity.from_bytes(principal)
      elif not model.is_valid_group_name(principal):
        raise ValueError('Not a valid group name')
    except ValueError as exc:
      self.abort_with_error(400, text='Bad principal - %s' % exc)

    # By default use cached auth DB. Switch to latest one only if No-Cache
    # header is given.
    auth_db = _get_maybe_cached_auth_db(self.request)
    subgraph = auth_db.get_relevant_subgraph(principal)

    def as_dict(node, edges):
      if isinstance(node, model.Identity):
        kind = 'IDENTITY'
        value = node.to_bytes()
      elif isinstance(node, model.IdentityGlob):
        kind = 'GLOB'
        value = node.to_bytes()
      else:
        assert isinstance(node, basestring), node
        kind = 'GROUP'
        value = node

      sorted_edges = {}
      for label, node_id_set in edges.iteritems():
        if node_id_set:
          sorted_edges[label] = sorted(node_id_set)

      out = {'kind': kind, 'value': value}
      if sorted_edges:
        out['edges'] = sorted_edges
      return out

    # Per API contract the requested principal should have ID 0, verify this.
    assert subgraph.root_id == 0, subgraph.root_id
    self.send_response({
      'subgraph': {
        'nodes': [as_dict(node, edges) for node, edges in subgraph.describe()],
      },
    })


class GroupsSuggestHandler(handler.ApiHandler):
  """Auto-complete for group names."""

  # This is visible in the UI.
  api_doc = [
    {
      'verb': 'GET',
      'params': 'name=...',

      'doc': 'Suggests group names that match the given string.',

      'response_type': {
        'name': 'Names',
        'doc': 'A list of group names.',
        'example': {
          'names': ['Group A', 'Group B'],
        },
      },
    },
  ]

  @api.require(acl.has_access)
  def get(self):
    name = self.request.get('name') or ''
    auth_db = api.get_request_cache().auth_db
    self.send_response({'names': auth_db.get_group_names_with_prefix(name)})


class ServerInfoHandler(handler.ApiHandler):
  """Returns information about the service (app version, service account name).

  May be used by other services to know what account to add to ACLs.
  """

  # In 99% cases account name is guessable from appID anyway, so its fine to
  # have this public.
  @api.public
  def get(self):
    self.send_response({
      'app_id': app_identity.get_application_id(),
      'app_runtime': 'python27',
      'app_version': utils.get_app_version(),
      'service_account_name': utils.get_service_account_name(),
    })


class CertificatesHandler(handler.ApiHandler):
  """Public certificates that service uses to sign blobs.

  May be used by other services when validating a signature of this service.
  Used by signature.get_service_public_certificates() method.
  """

  # Available to anyone, there's no secrets here.
  @api.public
  def get(self):
    self.send_response(signature.get_own_public_certificates().to_jsonish())


class OAuthConfigHandler(handler.ApiHandler):
  """Returns client_id and client_secret to use for OAuth2 login on a client.

  GET is available in Standalone, Primary and Replica modes.
  POST is available only in Standalone and Primary modes.
  """

  @api.public
  def get(self):
    client_id = None
    client_secret = None
    additional_ids = None
    token_server_url = None

    # Use most up-to-date data in datastore if requested. Used by management UI.
    if _is_no_cache(self.request):
      global_config = model.root_key().get()
      client_id = global_config.oauth_client_id
      client_secret = global_config.oauth_client_secret
      additional_ids = global_config.oauth_additional_client_ids
      token_server_url = global_config.token_server_url
    else:
      # Faster call that uses cached config (that may be several minutes stale).
      # Used by all client side scripts that just want to authenticate.
      auth_db = api.get_request_auth_db()
      client_id, client_secret, additional_ids = auth_db.get_oauth_config()
      token_server_url = auth_db.token_server_url

    # Grab URL of a primary service if running as a replica.
    replication_state = model.get_replication_state()
    primary_url = replication_state.primary_url if replication_state else None

    self.send_response({
      'additional_client_ids': additional_ids,
      'client_id': client_id,
      'client_not_so_secret': client_secret,
      'primary_url': primary_url,
      'token_server_url': token_server_url,
    })

  @forbid_api_on_replica
  @api.require(acl.is_admin)
  def post(self):
    if is_config_locked():
      self.abort_with_error(409, text='The configuration is managed elsewhere')

    body = self.parse_body()
    try:
      client_id = body['client_id']
      client_secret = body['client_not_so_secret']
      additional_client_ids = filter(bool, body['additional_client_ids'])
      token_server_url = body['token_server_url']
    except KeyError as exc:
      self.abort_with_error(400, text='Missing key %s' % exc)

    if token_server_url:
      try:
        utils.validate_root_service_url(token_server_url)
      except ValueError as exc:
        self.abort_with_error(400, text='Invalid token server URL - %s' % exc)

    @ndb.transactional
    def update():
      config = model.root_key().get()
      config.populate(
          oauth_client_id=client_id,
          oauth_client_secret=client_secret,
          oauth_additional_client_ids=additional_client_ids,
          token_server_url=token_server_url)
      config.record_revision(
          modified_by=api.get_current_identity(),
          modified_ts=utils.utcnow(),
          comment='REST API')
      config.put()
      model.replicate_auth_db()

    update()
    self.send_response({'ok': True})


class ServerStateHandler(handler.ApiHandler):
  """Reports replication state of a service."""

  @api.require(acl.has_access)
  def get(self):
    if model.is_primary():
      mode = 'primary'
    elif model.is_replica():
      mode = 'replica'
    else:
      assert model.is_standalone()
      mode = 'standalone'
    state = model.get_replication_state() or model.AuthReplicationState()
    self.send_response({
      'auth_code_version': version.__version__,
      'mode': mode,
      'replication_state': state.to_serializable_dict(),
    })
