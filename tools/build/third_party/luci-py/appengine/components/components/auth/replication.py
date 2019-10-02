# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Replica side of Primary <-> Replica protocol.

Also includes common code used by both Replica and Primary.
"""

import collections
import hashlib

from google.appengine.api import app_identity
from google.appengine.api import urlfetch
from google.appengine.ext import ndb

from components import utils

from . import model
from . import signature
from . import tokens
from .proto import replication_pb2


# Messages for error codes in ServiceLinkResponse.
LINKING_ERRORS = {
  replication_pb2.ServiceLinkResponse.TRANSPORT_ERROR: 'Transport error,',
  replication_pb2.ServiceLinkResponse.BAD_TICKET: 'The link has expired.',
  replication_pb2.ServiceLinkResponse.AUTH_ERROR: 'Authentication error.',
}


# Returned by new_auth_db_snapshot.
AuthDBSnapshot = collections.namedtuple(
    'AuthDBSnapshot',
    'global_config, groups, ip_whitelists, ip_whitelist_assignments')


class ProtocolError(Exception):
  """Raised when request to primary fails."""
  def __init__(self, status_code, msg):
    super(ProtocolError, self).__init__(msg)
    self.status_code = status_code


def decode_link_ticket(encoded):
  """Returns replication_pb2.ServiceLinkTicket given base64 encoded blob."""
  return replication_pb2.ServiceLinkTicket.FromString(
      tokens.base64_decode(encoded))


def become_replica(ticket, initiated_by):
  """Converts current service to a replica of a primary specified in a ticket.

  Args:
    ticket: replication_pb2.ServiceLinkTicket passed from a primary.
    initiated_by: Identity of a user that accepted linking request, for logging.

  Raises:
    ProtocolError in case the request to primary fails.
  """
  assert model.is_standalone()

  # On dev appserver emulate X-Appengine-Inbound-Appid header.
  headers = {'Content-Type': 'application/octet-stream'}
  protocol = 'https'
  if utils.is_local_dev_server():
    headers['X-Appengine-Inbound-Appid'] = app_identity.get_application_id()
    protocol = 'http'
  headers['X-URLFetch-Service-Id'] = utils.get_urlfetch_service_id()

  # Pass back the ticket for primary to verify it, tell the primary to use
  # default version hostname to talk to us.
  link_request = replication_pb2.ServiceLinkRequest()
  link_request.ticket = ticket.ticket
  link_request.replica_url = (
      '%s://%s' % (protocol, app_identity.get_default_version_hostname()))
  link_request.initiated_by = initiated_by.to_bytes()

  # Primary will look at X-Appengine-Inbound-Appid and compare it to what's in
  # the ticket.
  try:
    result = urlfetch.fetch(
        url='%s/auth_service/api/v1/internal/link_replica' % ticket.primary_url,
        payload=link_request.SerializeToString(),
        method='POST',
        headers=headers,
        follow_redirects=False,
        deadline=30,
        validate_certificate=True)
  except urlfetch.Error as exc:
    raise ProtocolError(
        replication_pb2.ServiceLinkResponse.TRANSPORT_ERROR,
        'URLFetch error (%s): %s' % (exc.__class__.__name__, exc))

  # Protobuf based protocol is not using HTTP codes (handler always replies with
  # HTTP 200, providing error details if needed in protobuf serialized body).
  # So any other status code here means there was a transport level error.
  if result.status_code != 200:
    raise ProtocolError(
        replication_pb2.ServiceLinkResponse.TRANSPORT_ERROR,
        'Request to the primary failed with HTTP %d.' % result.status_code)

  link_response = replication_pb2.ServiceLinkResponse.FromString(result.content)
  if link_response.status != replication_pb2.ServiceLinkResponse.SUCCESS:
    message = LINKING_ERRORS.get(
        link_response.status,
        'Request to the primary failed with status %d.' % link_response.status)
    raise ProtocolError(link_response.status, message)

  # Become replica. Auth DB will be overwritten on a first push from Primary.
  state = model.AuthReplicationState(
      key=model.replication_state_key(),
      primary_id=ticket.primary_id,
      primary_url=ticket.primary_url)
  state.put()


@ndb.transactional
def new_auth_db_snapshot():
  """Makes a consistent snapshot of replicated subset of AuthDB entities.

  Returns:
    Tuple (AuthReplicationState, AuthDBSnapshot).
  """
  # Start fetching stuff in parallel.
  state_future = model.replication_state_key().get_async()
  config_future = model.root_key().get_async()
  groups_future = model.AuthGroup.query(ancestor=model.root_key()).fetch_async()

  # It's fine to block here as long as it's the last fetch.
  ip_whitelist_assignments, ip_whitelists = model.fetch_ip_whitelists()

  snapshot = AuthDBSnapshot(
      config_future.get_result() or model.AuthGlobalConfig(
          key=model.root_key()),
      groups_future.get_result(),
      ip_whitelists,
      ip_whitelist_assignments)
  return state_future.get_result(), snapshot


def auth_db_snapshot_to_proto(snapshot, auth_db_proto=None):
  """Writes AuthDBSnapshot into replication_pb2.AuthDB message.

  Args:
    snapshot: instance of AuthDBSnapshot with entities to convert to protobuf.
    auth_db_proto: optional instance of replication_pb2.AuthDB to update.

  Returns:
    Instance of replication_pb2.AuthDB (same as |auth_db_proto| if passed).
  """
  auth_db_proto = auth_db_proto or replication_pb2.AuthDB()

  # Many fields in auth_db_proto were once 'required' and many clients still
  # expect them to be populated. Unfortunately setting a proto3 string field
  # to '' doesn't mark it as "set" from proto2 perspective. So inject some
  # sentinel values instead.
  #
  # TODO(vadimsh): Remove this hack when all services (including Gerrit plugin)
  # are switched to use proto3.
  auth_db_proto.oauth_client_id = (
      snapshot.global_config.oauth_client_id or 'empty')
  auth_db_proto.oauth_client_secret = (
      snapshot.global_config.oauth_client_secret or 'empty')
  if snapshot.global_config.oauth_additional_client_ids:
    auth_db_proto.oauth_additional_client_ids.extend(
        snapshot.global_config.oauth_additional_client_ids)

  auth_db_proto.token_server_url = (
      snapshot.global_config.token_server_url or 'empty')
  auth_db_proto.security_config = (
      snapshot.global_config.security_config or 'empty')

  for ent in snapshot.groups:
    msg = auth_db_proto.groups.add()
    msg.name = ent.key.id()
    msg.members.extend(ident.to_bytes() for ident in ent.members)
    msg.globs.extend(glob.to_bytes() for glob in ent.globs)
    msg.nested.extend(ent.nested)
    msg.description = ent.description or 'empty'
    msg.created_ts = utils.datetime_to_timestamp(ent.created_ts)
    msg.created_by = ent.created_by.to_bytes()
    msg.modified_ts = utils.datetime_to_timestamp(ent.modified_ts)
    msg.modified_by = ent.modified_by.to_bytes()
    msg.owners = ent.owners

  for ent in snapshot.ip_whitelists:
    msg = auth_db_proto.ip_whitelists.add()
    msg.name = ent.key.id()
    msg.subnets.extend(ent.subnets)
    msg.description = ent.description or 'empty'
    msg.created_ts = utils.datetime_to_timestamp(ent.created_ts)
    msg.created_by = ent.created_by.to_bytes()
    msg.modified_ts = utils.datetime_to_timestamp(ent.modified_ts)
    msg.modified_by = ent.modified_by.to_bytes()

  for ent in snapshot.ip_whitelist_assignments.assignments:
    msg = auth_db_proto.ip_whitelist_assignments.add()
    msg.identity = ent.identity.to_bytes()
    msg.ip_whitelist = ent.ip_whitelist
    msg.comment = ent.comment or 'empty'
    msg.created_ts = utils.datetime_to_timestamp(ent.created_ts)
    msg.created_by = ent.created_by.to_bytes()

  return auth_db_proto


def proto_to_auth_db_snapshot(auth_db_proto):
  """Given replication_pb2.AuthDB message returns AuthDBSnapshot."""
  # Explicit conversion to 'list' is needed here since protobuf magic doesn't
  # stack with NDB magic.
  global_config = model.AuthGlobalConfig(
      key=model.root_key(),
      oauth_client_id=auth_db_proto.oauth_client_id,
      oauth_client_secret=auth_db_proto.oauth_client_secret,
      oauth_additional_client_ids=list(
          auth_db_proto.oauth_additional_client_ids),
      token_server_url=auth_db_proto.token_server_url,
      security_config=auth_db_proto.security_config)

  groups = [
    model.AuthGroup(
        key=model.group_key(msg.name),
        members=[model.Identity.from_bytes(x) for x in msg.members],
        globs=[model.IdentityGlob.from_bytes(x) for x in msg.globs],
        nested=list(msg.nested),
        description=msg.description,
        owners=msg.owners or model.ADMIN_GROUP,
        created_ts=utils.timestamp_to_datetime(msg.created_ts),
        created_by=model.Identity.from_bytes(msg.created_by),
        modified_ts=utils.timestamp_to_datetime(msg.modified_ts),
        modified_by=model.Identity.from_bytes(msg.modified_by))
    for msg in auth_db_proto.groups
  ]

  ip_whitelists = [
    model.AuthIPWhitelist(
        key=model.ip_whitelist_key(msg.name),
        subnets=list(msg.subnets),
        description=msg.description,
        created_ts=utils.timestamp_to_datetime(msg.created_ts),
        created_by=model.Identity.from_bytes(msg.created_by),
        modified_ts=utils.timestamp_to_datetime(msg.modified_ts),
        modified_by=model.Identity.from_bytes(msg.modified_by))
    for msg in auth_db_proto.ip_whitelists
  ]

  ip_whitelist_assignments = model.AuthIPWhitelistAssignments(
      key=model.ip_whitelist_assignments_key(),
      assignments=[
        model.AuthIPWhitelistAssignments.Assignment(
            identity=model.Identity.from_bytes(msg.identity),
            ip_whitelist=msg.ip_whitelist,
            comment=msg.comment,
            created_ts=utils.timestamp_to_datetime(msg.created_ts),
            created_by=model.Identity.from_bytes(msg.created_by))
        for msg in auth_db_proto.ip_whitelist_assignments
      ],
  )

  return AuthDBSnapshot(
      global_config, groups, ip_whitelists, ip_whitelist_assignments)


def get_changed_entities(new_entity_list, old_entity_list):
  """Returns subset of changed entites.

  Compares entites from |new_entity_list| with entities from |old_entity_list|
  with same key, returns all changed or added entities.
  """
  old_by_key = {x.key: x for x in old_entity_list}
  new_or_changed = []
  for new_entity in new_entity_list:
    old_entity = old_by_key.get(new_entity.key)
    if not old_entity or old_entity.to_dict() != new_entity.to_dict():
      new_or_changed.append(new_entity)
  return new_or_changed


def get_deleted_keys(new_entity_list, old_entity_list):
  """Returns list of keys of entities that were removed."""
  new_by_key = frozenset(x.key for x in new_entity_list)
  return [old.key for old in old_entity_list if old.key not in new_by_key]


def replace_auth_db(auth_db_rev, modified_ts, snapshot):
  """Replaces AuthDB in datastore if it's older than |auth_db_rev|.

  May return False in case of race conditions (i.e. if some other concurrent
  process happened to update AuthDB earlier). May be retried in that case.

  Args:
    auth_db_rev: revision number of |snapshot|.
    modified_ts: datetime timestamp of when |auth_db_rev| was created.
    snapshot: AuthDBSnapshot with entity to store.

  Returns:
    Tuple (True if update was applied, current AuthReplicationState value).
  """
  assert model.is_replica()

  # Quickly check current auth_db rev before doing heavy calls.
  current_state = model.get_replication_state()
  if current_state.auth_db_rev >= auth_db_rev:
    return False, current_state

  # Make a snapshot of existing state of AuthDB to figure out what to change.
  current_state, current = new_auth_db_snapshot()

  # Entities that needs to be updated or created.
  entites_to_put = []
  if snapshot.global_config.to_dict() != current.global_config.to_dict():
    entites_to_put.append(snapshot.global_config)
  entites_to_put.extend(get_changed_entities(snapshot.groups, current.groups))
  entites_to_put.extend(
      get_changed_entities(snapshot.ip_whitelists, current.ip_whitelists))
  new_ips = snapshot.ip_whitelist_assignments
  old_ips = current.ip_whitelist_assignments
  if new_ips.to_dict() != old_ips.to_dict():
    entites_to_put.append(new_ips)

  # Keys of entities that needs to be removed.
  keys_to_delete = []
  keys_to_delete.extend(get_deleted_keys(snapshot.groups, current.groups))
  keys_to_delete.extend(
      get_deleted_keys(snapshot.ip_whitelists, current.ip_whitelists))

  @ndb.transactional
  def update_auth_db():
    # AuthDB changed since 'new_auth_db_snapshot' transaction? Back off.
    state = model.get_replication_state()
    if state.auth_db_rev != current_state.auth_db_rev:
      return False, state

    # Update auth_db_rev in AuthReplicationState.
    state.auth_db_rev = auth_db_rev
    state.modified_ts = modified_ts

    # Apply changes.
    futures = []
    futures.extend(ndb.put_multi_async([state] + entites_to_put))
    futures.extend(ndb.delete_multi_async(keys_to_delete))

    # Wait for all pending futures to complete. Aborting the transaction with
    # outstanding futures is a bad idea (ndb complains in log about that).
    ndb.Future.wait_all(futures)

    # Raise an exception, if any.
    for future in futures:
      future.check_success()

    # Success.
    return True, state

  # Do the transactional update.
  return update_auth_db()


def is_signed_by_primary(blob, key_name, sig):
  """Verifies that |blob| was signed by Primary."""
  # Assert that running on Replica.
  state = model.get_replication_state()
  assert state and state.primary_url, state
  # Grab the cert from primary and verify the signature. We are signing SHA512
  # hashes, since AuthDB blob is too large.
  certs = signature.get_service_public_certificates(state.primary_url)
  digest = hashlib.sha512(blob).digest()
  return certs.check_signature(digest, key_name, sig)


def push_auth_db(revision, auth_db):
  """Accepts AuthDB push from Primary and applies it to replica.

  Args:
    revision: replication_pb2.AuthDBRevision describing revision of pushed DB.
    auth_db: replication_pb2.AuthDB with pushed DB.

  Returns:
    Tuple (True if update was applied, stored or updated AuthReplicationState).
  """
  # Already up-to-date? Check it first before doing heavy calls.
  state = model.get_replication_state()
  if (state.primary_id == revision.primary_id and
      state.auth_db_rev >= revision.auth_db_rev):
    return False, state

  # Try to apply it, retry until success (or until some other task applies
  # an even newer version of auth_db).
  snapshot = proto_to_auth_db_snapshot(auth_db)
  while True:
    applied, current_state = replace_auth_db(
        revision.auth_db_rev,
        utils.timestamp_to_datetime(revision.modified_ts),
        snapshot)

    # Update was successfully applied.
    if applied:
      return True, current_state

    # Some other task managed to apply the update already.
    if current_state.auth_db_rev >= revision.auth_db_rev:
      return False, current_state

    # Need to retry. Try until success or deadline.
    assert current_state.auth_db_rev < revision.auth_db_rev
