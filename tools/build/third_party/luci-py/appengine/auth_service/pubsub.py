# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Manages PubSub topic that receives notifications about AuthDB changes.

The topic is hosted in auth_service Cloud Project and auth_service manages its
IAM policies.

All service accounts listed in 'auth-trusted-services' group are entitled for
a subscription to AuthDB change notifications, so that they can pull AuthDB
snapshots as soon as they are available.

Members of 'auth-trusted-services' can create as many subscription as they like.
They have 'pubsub.topics.attachSubscription' permission on the topic and can
create subscriptions belong to Cloud Projects they own.
"""

import base64
import logging

from google.appengine.api import app_identity

from components import auth
from components import pubsub
from components import utils
from components.auth import signature
from components.auth.proto import replication_pb2

import acl


# Fatal errors raised by this module. Reuse pubset.Error to avoid catching and
# raising an exception again all the time.
Error = pubsub.Error


def topic_name():
  """Full name of PubSub topic that receives AuthDB change notifications."""
  return pubsub.full_topic_name(
      app_identity.get_application_id(), 'auth-db-changed')


def _email_to_iam_ident(email):
  """Given email returns 'user:...' or 'serviceAccount:...'."""
  if email.endswith('.gserviceaccount.com'):
    return 'serviceAccount:' + email
  return 'user:' + email


def _iam_ident_to_email(ident):
  """Given IAM identity returns email address or None."""
  for p in ('user:', 'serviceAccount:'):
    if ident.startswith(p):
      return ident[len(p):]
  return None


def is_authorized_subscriber(email):
  """True if given user can attach subscriptions to the topic."""
  with pubsub.iam_policy(topic_name()) as p:
    return _email_to_iam_ident(email) in p.members('roles/pubsub.subscriber')


def authorize_subscriber(email):
  """Allows given user to attach subscriptions to the topic."""
  with pubsub.iam_policy(topic_name()) as p:
    p.add_member('roles/pubsub.subscriber', _email_to_iam_ident(email))


def deauthorize_subscriber(email):
  """Revokes authorization to attach subscriptions to the topic."""
  with pubsub.iam_policy(topic_name()) as p:
    p.remove_member('roles/pubsub.subscriber', _email_to_iam_ident(email))


def revoke_stale_authorization():
  """Removes pubsub.subscriber role from accounts that no longer have access."""
  try:
    with pubsub.iam_policy(topic_name()) as p:
      for iam_ident in p.members('roles/pubsub.subscriber'):
        email = _iam_ident_to_email(iam_ident)
        if email:
          ident = auth.Identity.from_bytes('user:' + email)
          if not acl.is_trusted_service(ident):
            logging.warning('Removing "%s" from subscribers list', iam_ident)
            p.remove_member('roles/pubsub.subscriber', iam_ident)
  except Error as e:
    logging.warning('Failed to revoke stale users: %s', e)


def publish_authdb_change(state):
  """Publishes AuthDB change notification to the topic.

  Args:
    state: AuthReplicationState with version info.
  """
  if utils.is_local_dev_server():
    return

  msg = replication_pb2.ReplicationPushRequest()
  msg.revision.primary_id = app_identity.get_application_id()
  msg.revision.auth_db_rev = state.auth_db_rev
  msg.revision.modified_ts = utils.datetime_to_timestamp(state.modified_ts)

  blob = msg.SerializeToString()
  key_name, sig = signature.sign_blob(blob)

  pubsub.publish(topic_name(), blob, {
    'X-AuthDB-SigKey-v1': key_name,
    'X-AuthDB-SigVal-v1': base64.b64encode(sig),
  })
