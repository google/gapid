# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Primary side of Primary <-> Replica protocol."""

import base64
import hashlib
import logging
import zlib

from google.appengine.api import app_identity
from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components import datastore_utils
from components import utils

from components.auth import model
from components.auth import replication
from components.auth import signature
from components.auth import version
from components.auth.proto import replication_pb2

import pubsub


# Possible values of push_status field of AuthReplicaState.
PUSH_STATUS_SUCCESS = 0
PUSH_STATUS_TRANSIENT_ERROR = 1
PUSH_STATUS_FATAL_ERROR = 2


class ReplicationTriggerError(Exception):
  """Failed to trigger a replication task."""


class ReplicaUpdateError(Exception):
  """Failed to update a replica."""


class TransientReplicaUpdateError(ReplicaUpdateError):
  """Failed to update a replica, update should be retried."""


class FatalReplicaUpdateError(ReplicaUpdateError):
  """Failed to update a replica, update must not be retried."""


class AuthReplicaState(ndb.Model, datastore_utils.SerializableModelMixin):
  """Last known state of a Replica as known by Primary.

  Parent key is replicas_root_key(). Key id is GAE application ID of a replica.
  """
  # How to convert this entity to serializable dict.
  serializable_properties = {
    'replica_url': datastore_utils.READABLE,
    'auth_db_rev': datastore_utils.READABLE,
    'rev_modified_ts': datastore_utils.READABLE,
    'auth_code_version': datastore_utils.READABLE,
    'push_started_ts': datastore_utils.READABLE,
    'push_finished_ts': datastore_utils.READABLE,
    'push_status': datastore_utils.READABLE,
    'push_error': datastore_utils.READABLE,
  }

  # URL of a host to push AuthDB updates to, especially useful on dev_appserver.
  replica_url = ndb.StringProperty(indexed=False)
  # Revision of auth DB replica is synced to.
  auth_db_rev = ndb.IntegerProperty(default=0, indexed=False)
  # Time when auth_db_rev was created (by primary clock).
  rev_modified_ts = ndb.DateTimeProperty(indexed=False)
  # Value of components.auth.version.__version__ used by replica.
  auth_code_version = ndb.StringProperty(indexed=False)

  # Timestamp of when last push attempt started.
  push_started_ts = ndb.DateTimeProperty(indexed=False)
  # Timestamp of when last push attempt finished (successfully or not).
  push_finished_ts = ndb.DateTimeProperty(indexed=False)
  # Status of last push attempt. See PUSH_STATUS_* enumeration.
  push_status = ndb.IntegerProperty(indexed=False)
  # Error message of last push attempt, or empty string if it was successful.
  push_error = ndb.StringProperty(indexed=False)


class AuthDBSnapshot(ndb.Model):
  """Contains deflated serialized AuthDB proto message for some revision.

  Root entity. ID is corresponding revision number (as integer). Immutable.
  """
  # Deflated serialized AuthDB proto message.
  auth_db_deflated = ndb.BlobProperty()
  # SHA256 hex digest of auth_db (before compression).
  auth_db_sha256 = ndb.StringProperty(indexed=False)
  # When this revision was created.
  created_ts = ndb.DateTimeProperty(indexed=True)


class AuthDBSnapshotLatest(ndb.Model):
  """Pointer to latest stored AuthDBSnapshot entity.

  Exists in single instance with key ('AuthDBSnapshotLatest', 'latest').
  """
  # Revision number of latest stored AuthDBSnaphost. Monotonically increases.
  auth_db_rev = ndb.IntegerProperty(indexed=False)
  # When latest stored AuthDBSnaphost was created (and this entity updated).
  modified_ts = ndb.DateTimeProperty(indexed=False)
  # SHA256 hex digest of latest AuthDBSnapshot's auth_db (before compression).
  auth_db_sha256 = ndb.StringProperty(indexed=False)


def replicas_root_key():
  """Root key for AuthReplicaState entities. Entity itself doesn't exist."""
  # It' intentionally not under model.root_key(). It has nothing to do with core
  # auth model.
  return ndb.Key('AuthReplicaStateRoot', 'root')


def replica_state_key(app_id):
  """Returns key of corresponding AuthReplicaState entity."""
  return ndb.Key(AuthReplicaState, app_id, parent=replicas_root_key())


def auth_db_snapshot_key(auth_db_rev):
  """Key for AuthDBSnapshot at given revision."""
  assert isinstance(auth_db_rev, (int, long)), auth_db_rev
  return ndb.Key(AuthDBSnapshot, int(auth_db_rev))


def auth_db_snapshot_latest_key():
  """Key of AuthDBSnapshotLatest singleton entity."""
  return ndb.Key(AuthDBSnapshotLatest, 'latest')


def get_latest_auth_db_snapshot(skip_body):
  """Returns latest known AuthDBSnapshot or None."""
  latest = auth_db_snapshot_latest_key().get()
  if not latest:
    return None
  if skip_body:
    return AuthDBSnapshot(
        key=auth_db_snapshot_key(latest.auth_db_rev),
        auth_db_sha256=latest.auth_db_sha256,
        created_ts=latest.modified_ts)
  return get_auth_db_snapshot(latest.auth_db_rev, skip_body)


def get_auth_db_snapshot(rev, skip_body):
  """Returns AuthDBSnapshot at given revision or None."""
  s = auth_db_snapshot_key(rev).get()
  if not s:
    return None
  if skip_body:
    s.auth_db_deflated = None
  return s


def configure_as_primary():
  """Switches current service to Primary mode.

  Called during loading of backend and frontend modules before any other calls.
  """
  def replication_callback(auth_state):
    assert ndb.in_transaction()
    trigger_replication(auth_state.auth_db_rev, transactional=True)
  model.configure_as_primary(replication_callback)


def is_replica(ident):
  """True if given identity corresponds to registered replica."""
  return ident.is_service and bool(replica_state_key(ident.name).get())


def register_replica(app_id, replica_url):
  """Creates a new AuthReplicaState or resets the state of existing one."""
  ent = AuthReplicaState(
      id=app_id,
      parent=replicas_root_key(),
      replica_url=replica_url)
  ent.put()
  trigger_replication()


def trigger_replication(auth_db_rev=None, transactional=False):
  """Enqueues a task to push auth db to replicas.

  Args:
    auth_db_rev: revision to push, if at the moment the task is executing
        current revision is different, the task will be skipped. By default uses
        a revision at the moment 'trigger_replication' is called.
    transactional: if True enqueue the task transactionally.

  Raises:
    ReplicationTriggerError on error.
  """
  if auth_db_rev is None:
    auth_db_rev = model.get_replication_state().auth_db_rev

  # Use explicit task queue call instead of 'deferred' module to route tasks
  # through WSGI app set up in backend/handlers.py. It has global state
  # correctly configured (ereporter config, etc). 'deferred' module uses its
  # own WSGI app. Task '/internal/taskqueue/replication/<rev>' translates
  # to a call to 'update_replicas_task(<rev>)'.
  if not utils.enqueue_task(
      url='/internal/taskqueue/replication/%d' % auth_db_rev,
      queue_name='replication',
      transactional=transactional):
    raise ReplicationTriggerError()


def update_replicas_task(auth_db_rev):
  """Packs AuthDB and pushes it to all out-of-date Replicas.

  Called via /internal/taskqueue/replication/<auth_db_rev> task (see
  backend/handlers.py) enqueued by 'trigger_replication'.

  Will check that AuthReplicationState.auth_db_rev is still equal to
  |auth_db_rev| before doing anything.

  Returns:
    True if all replicas are up-to-date now, False if task should be retried.
  """
  # Check that the task is not stale before doing any heavy lifting.
  replication_state = model.get_replication_state()
  if replication_state.auth_db_rev != auth_db_rev:
    logging.info(
        'Skipping stale task, current rev is %d, task was enqueued for rev %d)',
        replication_state.auth_db_rev, auth_db_rev)
    return True

  # Pack an entire AuthDB into a blob to be to stored in the datastore and
  # pushed to Replicas.
  replication_state, auth_db_blob = pack_auth_db()

  # Put the blob into datastore. Also updates pointer to the latest stored blob.
  store_auth_db_snapshot(replication_state, auth_db_blob)

  # Notify PubSub subscribers that new snapshot is available.
  pubsub.publish_authdb_change(replication_state)

  # Grab last known replicas state and push only to replicas that are behind.
  stale_replicas = [
    entity for entity in AuthReplicaState.query(ancestor=replicas_root_key())
    if entity.auth_db_rev is None or entity.auth_db_rev < auth_db_rev
  ]
  if not stale_replicas:
    logging.info('All replicas are up-to-date.')
    return True

  # Sign the blob, replicas check the signature.
  key_name, sig = sign_auth_db_blob(auth_db_blob)

  # Push the blob to all out-of-date replicas, in parallel.
  push_started_ts = utils.utcnow()
  futures = {
    push_to_replica(
        replica.replica_url, auth_db_blob, key_name, sig): replica
    for replica in stale_replicas
  }

  # Wait for all attempts to complete.
  retry = []
  while futures:
    completed = ndb.Future.wait_any(futures)
    replica = futures.pop(completed)

    exception = completed.get_exception()
    success = exception is None

    current_revision = None
    auth_code_version = None
    if success:
      current_revision, auth_code_version = completed.get_result()

    if not success:
      logging.error(
          'Error when pushing update to replica: %s (%s).\nReplica id is %s.',
          exception.__class__.__name__, exception, replica.key.id())
      # Give up only on explicit fatal error, retry on any other exception.
      if not isinstance(exception, FatalReplicaUpdateError):
        retry.append(replica)

    # Eagerly update known replica state in local DB as soon as response is
    # received. That way if 'update_replicas_task' is killed midway, at least
    # the state of some replicas will be updated. Note that this transaction is
    # modifying a single entity group (replicas_root_key()) and thus can't be
    # called very often (due to 1 QPS limit on entity group updates).
    # If contention here becomes an issue, adding simple time.sleep(X) before
    # the transaction is totally fine (since 'update_replicas_task' is executed
    # on background task queue).
    try:
      if success:
        stored_rev = _update_state_on_success(
            key=replica.key,
            started_ts=push_started_ts,
            finished_ts=utils.utcnow(),
            current_revision=current_revision,
            auth_code_version=auth_code_version)
        logging.info(
            'Replica %s is updated to rev %d', replica.key.id(), stored_rev)
      else:
        stored_rev = _update_state_on_fail(
            key=replica.key,
            started_ts=push_started_ts,
            finished_ts=utils.utcnow(),
            old_auth_db_rev=replica.auth_db_rev,
            exc=exception)
        # If current push failed, but some other concurrent push (if any)
        # succeeded (and so replica is up-to-date), do not retry current push.
        if stored_rev is None or stored_rev > auth_db_rev:
          if replica in retry:
            retry.remove(replica)
    except (
        datastore_errors.InternalError,
        datastore_errors.Timeout,
        datastore_errors.TransactionFailedError) as exc:
      logging.exception(
          'Datastore error when updating replica state: %s.\n'
          'Replica id is %s.', exc.__class__.__name__, replica.key.id())
      # Should retry the task because of this.
      retry.add(replica)

  # Retry the task if at least one replica reported a retryable error.
  return not retry


def pack_auth_db():
  """Packs an entire AuthDB into a blob (serialized protobuf message).

  Returns:
    Tuple (AuthReplicationState, blob).
  """
  # Grab the snapshot.
  state, snapshot = replication.new_auth_db_snapshot()

  # Serialize to binary proto message.
  req = replication_pb2.ReplicationPushRequest()
  req.revision.primary_id = app_identity.get_application_id()
  req.revision.auth_db_rev = state.auth_db_rev
  req.revision.modified_ts = utils.datetime_to_timestamp(state.modified_ts)
  replication.auth_db_snapshot_to_proto(snapshot, req.auth_db)
  req.auth_code_version = version.__version__
  auth_db_blob = req.SerializeToString()

  logging.debug('AuthDB blob size is %d bytes', len(auth_db_blob))
  return state, auth_db_blob


def sign_auth_db_blob(auth_db_blob):
  """Signs AuthDB blob with app's private key.

  Returns:
    Tuple (name of a key used, base64 encoded signature).
  """
  # sign_blob is limited to 8KB only, so
  # hash the body first and sign the digest.
  key_name, sig = signature.sign_blob(hashlib.sha512(auth_db_blob).digest())
  return key_name, base64.b64encode(sig)


def store_auth_db_snapshot(replication_state, auth_db_blob):
  """Puts AuthDB blob (serialized proto) into datastore.

  Args:
    replication_state: AuthReplicationState that correspond to auth_db_blob.
    auth_db_blob: serialized AuthDB proto message.
  """
  deflated = zlib.compress(auth_db_blob)
  sha256 = hashlib.sha256(auth_db_blob).hexdigest()
  key = auth_db_snapshot_key(replication_state.auth_db_rev)
  latest_key = auth_db_snapshot_latest_key()

  logging.debug('AuthDB deflated blob size is %d bytes', len(deflated))

  @ndb.transactional
  def insert():
    if not key.get():
      e = AuthDBSnapshot(
        key=key,
        auth_db_deflated=deflated,
        auth_db_sha256=sha256,
        created_ts=replication_state.modified_ts)
      e.put()
  insert()

  @ndb.transactional
  def update_latest_pointer():
    latest = latest_key.get()
    if not latest:
      latest = AuthDBSnapshotLatest(key=latest_key)
    if latest.auth_db_rev < replication_state.auth_db_rev:
      latest.auth_db_rev = replication_state.auth_db_rev
      latest.modified_ts = replication_state.modified_ts
      latest.auth_db_sha256 = sha256
      latest.put()
  update_latest_pointer()


@ndb.tasklet
def push_to_replica(replica_url, auth_db_blob, key_name, sig):
  """Pushes |auth_db_blob| to a replica via URLFetch POST.

  Args:
    replica_url: root URL of a replica (i.e. https://<host>).
    auth_db_blob: binary blob with serialized Auth DB.
    key_name: name of a RSA key used to generate a signature.
    sig: base64 encoded signature of |auth_db_blob|.

  Returns:
    Tuple:
      AuthDB revision reporter by a replica (as replication_pb2.AuthDBRevision).
      Auth component version used by replica (see components.auth.version).

  Raises:
    FatalReplicaUpdateError if replica rejected the push.
    TransientReplicaUpdateError if push should be retried.
  """
  replica_url = replica_url.rstrip('/')
  logging.info('Updating replica %s', replica_url)
  protocol = 'http://' if utils.is_local_dev_server() else 'https://'
  assert replica_url.startswith(protocol)

  # Pass signature via the headers.
  headers = {
    'Content-Type': 'application/octet-stream',
    'X-URLFetch-Service-Id': utils.get_urlfetch_service_id(),
    'X-AuthDB-SigKey-v1': key_name,
    'X-AuthDB-SigVal-v1': sig,
  }

  # On dev appserver emulate X-Appengine-Inbound-Appid header.
  if utils.is_local_dev_server():
    headers['X-Appengine-Inbound-Appid'] = app_identity.get_application_id()

  # 'follow_redirects' set to False is required for 'X-Appengine-Inbound-Appid'
  # to work. 70 sec deadline correspond to 60 sec GAE foreground requests
  # deadline plus 10 seconds to account for URL fetch own lags.
  ctx = ndb.get_context()
  result = yield ctx.urlfetch(
      url=replica_url + '/auth/api/v1/internal/replication',
      payload=auth_db_blob,
      method='POST',
      headers=headers,
      follow_redirects=False,
      deadline=70,
      validate_certificate=True)

  # Any transport level error is transient.
  if result.status_code != 200:
    raise TransientReplicaUpdateError(
        'Push request failed with HTTP code %d' % result.status_code)

  # Deserialize the response.
  cls = replication_pb2.ReplicationPushResponse
  response = cls.FromString(result.content)

  # Convert errors to exceptions.
  if response.status == cls.TRANSIENT_ERROR:
    raise TransientReplicaUpdateError(
        'Transient error (error code %d).' % response.error_code)
  if response.status == cls.FATAL_ERROR:
    raise FatalReplicaUpdateError(
        'Fatal error (error code %d).' % response.error_code)
  if response.status not in (cls.APPLIED, cls.SKIPPED):
    raise FatalReplicaUpdateError(
        'Unexpected response status: %d' % response.status)

  # Replica applied the update, current_revision should be set.
  if not response.HasField('current_revision'):
    raise FatalReplicaUpdateError(
        'Incomplete response, current_revision is missing')

  raise ndb.Return((response.current_revision, response.auth_code_version))


@ndb.transactional
def _update_state_on_success(
    key, started_ts, finished_ts, current_revision, auth_code_version):
  """Updates AuthReplicaState after a successful push.

  Args:
    key: key of AuthReplicaState entity to update.
    started_ts: datetime timestamp of when push was initiated.
    finished_ts: datetime timestamp of when push was completed.
    current_revision: an instance of AuthDBRevision as reported by Replica.
    auth_code_version: components.auth.version.__version__ on replica.

  Returns:
    Auth DB revision of replica as it is stored in DB after the update. May be
    different from current_revision.auth_db_rev (in case some other task
    already managed to update the replica).
  """
  # Currently stored state. May be ahead of the state initially fetched in
  # 'update_replicas_task'. If missing, the replica was removed from
  # replication list (and shouldn't be added back).
  state = key.get()
  if not state:
    return None

  # The state was updated by some other task already?
  if state.auth_db_rev >= current_revision.auth_db_rev:
    return state.auth_db_rev

  # Update stored revision, mark last push as success.
  state.auth_db_rev = current_revision.auth_db_rev
  state.rev_modified_ts = utils.timestamp_to_datetime(
      current_revision.modified_ts)
  state.auth_code_version = auth_code_version
  state.push_started_ts = started_ts
  state.push_finished_ts = finished_ts
  state.push_status = PUSH_STATUS_SUCCESS
  state.push_error = ''
  state.put()

  return state.auth_db_rev


@ndb.transactional
def _update_state_on_fail(key, started_ts, finished_ts, old_auth_db_rev, exc):
  """Updates AuthReplicaState after a failed push (on transient or fatal error).

  Args:
    key: key of AuthReplicaState entity to update.
    started_ts: datetime timestamp of when push was initiated.
    finished_ts: datetime timestamp of when push was completed.
    old_auth_db_rev: a revision stored in AuthReplicaState at the moment
        the push was initiated.
    exc: an instance of exception raised by 'push_to_replica'.

  Returns:
    Auth DB revision of replica as it is stored in DB.
  """
  # Currently stored state. If missing, the replica was removed.
  state = key.get()
  if not state:
    return None

  # Some other task updated the state already, don't mess with it.
  if state.auth_db_rev > old_auth_db_rev:
    return state.auth_db_rev

  # Add the error message to the last known state, do not change the revision.
  state.push_started_ts = started_ts
  state.push_finished_ts = finished_ts
  if isinstance(exc, FatalReplicaUpdateError):
    state.push_status = PUSH_STATUS_FATAL_ERROR
  else:
    state.push_status = PUSH_STATUS_TRANSIENT_ERROR
  state.push_error = str(exc)
  state.put()

  return state.auth_db_rev
