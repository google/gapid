# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Helper functions for working with Cloud Pub/Sub."""

import base64
import contextlib
import copy
import logging
import re

from google.appengine.ext import ndb
import webapp2

from components import net


# From Pub/Sub docs: `{subscription}` must start with a letter, and contain only
# letters (`[A-Za-z]`), numbers (`[0-9]`), dashes (`-`), underscores (`_`),
# periods (`.`), tildes (`~`), plus (`+`) or percent signs (`%`). It must be
# between 3 and 255 characters in length, and it must not start with `"goog"`.
#
# The last check is done in _validate_name to simplify the regexp.
#
# Same rules apply to topic names too.
_NAME_RE = re.compile(r'^[A-Za-z][A-Za-z0-9\-_\.~\+%]{2,254}$')


class Error(Exception):
  """Raised on fatal errors."""
  def __init__(self, inner):
    super(Error, self).__init__(str(inner))
    self.inner = inner


class TransientError(Exception):
  """Raised on errors that can go away with retry. Results in 500 HTTP code.

  Specifically not a subclass of Error, so that "except Error" block passes
  transient errors through.
  """
  def __init__(self, inner):
    super(TransientError, self).__init__(str(inner))
    self.inner = inner


def validate_project(project):
  """Ensures the given project is valid for Cloud Pub/Sub."""
  # Technically, there are more restrictions for project names than we check
  # here, but the API will reject anything that doesn't match. We only check /
  # in case the user is trying to manipulate the topic into posting somewhere
  # else (e.g. by setting the project as ../../<some other project>.
  return project and '/' not in project


def validate_topic(topic):
  """Ensures the given topic is valid for Cloud Pub/Sub."""
  return _validate_name(topic)


def validate_subscription(subscription):
  """Ensures the given subscription is valid for Cloud Pub/Sub."""
  return _validate_name(subscription)


def full_topic_name(project, topic):
  """Returns full topic name in given project."""
  assert validate_project(project), project
  assert validate_topic(topic), topic
  return 'projects/%s/topics/%s' % (project, topic)


def full_subscription_name(project, subscription):
  """Returns full subscription name in given project."""
  assert validate_project(project), project
  assert validate_subscription(subscription), subscription
  return 'projects/%s/subscriptions/%s' % (project, subscription)


def validate_full_name(name, kind):
  """Returns True if name has form "projects/<project-id>/<kind>/<id>."""
  chunks = name.split('/')
  return (
      len(chunks) == 4 and
      chunks[0] == 'projects' and
      validate_project(chunks[1]) and
      chunks[2] == kind and
      _validate_name(chunks[3]))


def _validate_name(name):
  """Returns True if the name matches rules for topic and subcription names."""
  return (
      not name.startswith('goog') and
      bool(_NAME_RE.match(name)))


def _call_async(method, endpoint, payload=None):
  """Makes HTTP request to Pub/Sub service.

  Args:
    method: HTTP verb, such as 'GET' or 'PUT'.
    endpoint: URL of the endpoint, relative to pubsub.googleapis.com/v1/.
    payload: Body of the request to send as JSON.
  """
  return net.json_request_async(
      url='https://pubsub.googleapis.com/v1/' + endpoint,
      method=method,
      payload=payload,
      scopes=['https://www.googleapis.com/auth/pubsub'])


def _call(method, endpoint, payload=None, accepted_http_statuses=None):
  """Makes HTTP request to Pub/Sub service.

  Args:
    method: HTTP verb, such as 'GET' or 'PUT'.
    endpoint: URL of the endpoint, relative to pubsub.googleapis.com/v1/.
    payload: Body of the request to send as JSON.
    accepted_http_statuses: List of additional status codes to treat as success.

  Raises:
    Error or TransientError.
  """
  try:
    return _call_async(method, endpoint, payload=payload).get_result()
  except net.Error as e:
    if accepted_http_statuses and e.status_code in accepted_http_statuses:
      return None
    if e.status_code >= 500:
      raise TransientError(e)
    raise Error(e)


def ensure_topic_exists(topic):
  """Ensures the given Cloud Pub/Sub topic exists.

  Args:
    topic: Full topic name, as returned by `full_topic_name`.

  Raises:
    Error or TransientError.
  """
  assert validate_full_name(topic, 'topics'), topic
  # 409 is the status code when the topic already exists.
  _call('PUT', topic, accepted_http_statuses=[409])


def _with_existing_topic(topic, callback):
  """Calls callback, catches 404, creates the topic, calls callback again.

  Used by 'publish' and 'ensure_subscription_exists'.

  Raises:
    Error or TransientError.
  """
  try:
    return callback()
  except Error as e:
    if e.inner.status_code != 404:
      raise e
    ensure_topic_exists(topic)
    return callback()


def publish_multi(topic, messages):
  """Publish messages to Cloud Pub/Sub. Creates the topic if it doesn't exist.

  Args:
    topic: Full name of the topic to publish to.
    messages: Content of the message to publish mapped to any attributes to
      send with the message.

  Raises:
    Error or TransientError.
  """
  assert validate_full_name(topic, 'topics'), topic
  messages = [
      {'attributes': attributes or {}, 'data': base64.b64encode(message)}
      for message, attributes in messages.iteritems()
  ]

  def call_publish():
    _call('POST', '%s:publish' % topic, payload={'messages': messages})

  _with_existing_topic(topic, call_publish)


def publish(topic, message, attributes):
  """Publish a message to Cloud Pub/Sub. Creates the topic if it doesn't exist.

  Args:
    topic: Full name of the topic to publish to.
    message: Content of the message to publish.
    attributes: Any attributes to send with the message.

  Raises:
    Error or TransientError.
  """
  publish_multi(topic, {message: attributes})


def modify_ack_deadline_async(subscription, deadline, *ack_ids):
  """Modifies acknowledgement deadline of messages.

  Args:
    subcription: Full name of the subscription.
    deadline: New deadline (in seconds).
    *ack_ids: List of IDs of messages to extend the ack deadline of.
  """
  return _call_async(
      'POST',
      '%s:modifyAckDeadline' % subscription,
      payload={
          'ackDeadlineSeconds': deadline,
          'ackIds': ack_ids,
      },
  )


def ack_async(subscription, *ack_ids):
  """Acknowledges receipt of messages.

  Args:
    subscription: Full name of the subscription.
    *ack_ids: List of IDs of messages to ack.
  """
  return _call_async(
      'POST', '%s:acknowledge' % subscription, payload={'ackIds': ack_ids})


def ack(*args, **kwargs):
  return ack_async(*args, **kwargs).get_result()


def pull(subscription, max_messages=100):
  """Polls a pull subscription for messages.

  Args:
    subscription: Full name of the subscription.
    max_messages: Maximum number of messages to return.
  """
  return _call(
      'POST',
      '%s:pull' % subscription,
      payload={
          'maxMessages': max_messages,
          'returnImmediately': True,
      },
  )


def get_subscription(subscription):
  """Returns subscription dict or None if no such subscription.

  See https://cloud.google.com/pubsub/reference/rest/v1/projects.subscriptions.

  Args:
    subscription: Full name of the subscription.

  Raises:
    Error or TransientError.
  """
  return _call('GET', subscription, accepted_http_statuses=[404])


def ensure_subscription_exists(
    subscription, topic, push_config=None, ack_deadline_seconds=None):
  """Ensures given subscription exists.

  Will register new subscription or chage push config of an existing one
  if necessary, but won't change a topic or default ack_deadline_seconds if they
  don't match requested values (there's no API to do that).

  Will also try to create the topic if it's missing.

  Args:
    subscription: Full name of the subscription.
    topic: Full name of the topic to subscribe to.
    push_config: Dict with push configuration, or None to not touch it.
    ack_deadline_seconds: How long to wait for ack.

  Raises:
    Error or TransientError.
  """
  assert validate_full_name(subscription, 'subscriptions'), subscription
  assert validate_full_name(topic, 'topics'), topic

  # Need to use GET to ensure the subscription is actually subscripted to
  # the given topic and not to something else.
  existing = _call('GET', subscription, accepted_http_statuses=[404])
  if existing:
    if existing['topic'] != topic:
      raise Error('Can\'t change topic of an existing subscription')
    if (ack_deadline_seconds is not None and
        ack_deadline_seconds != existing['ackDeadlineSeconds']):
      raise Error('Can\'t change ack deadline of an existing subscription')
    if push_config is not None and push_config != existing['pushConfig']:
      _call(
          'POST', '%s:modifyPushConfig' % subscription,
          payload={'pushConfig': push_config})
    return

  def create_subscription():
    _call(
        'PUT', subscription,
        payload={
            'topic': topic,
            'pushConfig': push_config or {},
            'ackDeadlineSeconds': ack_deadline_seconds or 60,
        },
    )

  _with_existing_topic(topic, create_subscription)


def ensure_topic_deleted(topic):
  """Deletes a topic if it exists.

  Args:
    topic: Full name of the topic.

  Raises:
    Error or TransientError.
  """
  assert validate_full_name(topic, 'topics'), topic
  _call('DELETE', topic, accepted_http_statuses=[404])


def ensure_subscription_deleted(subscription):
  """Deletes a subscription if it exists.

  Args:
    subscription: Full name of the subscription.

  Raises:
    Error or TransientError.
  """
  assert validate_full_name(subscription, 'subscriptions'), subscription
  _call('DELETE', subscription, accepted_http_statuses=[404])


@contextlib.contextmanager
def iam_policy(object_name):
  """Changes IAM policy for an existing subscription or topic.

  Reads current policy, invokes the context manager body, writes modified policy
  back. Uses etags for concurrency control. Autocreates a topic if object_name
  is referencing a topic that doesn't exist.

  Usage example:

  with pubsub.iam_policy(...) as policy:
    policy.add_member('roles/viewer', 'user:mike@example.com')

  Args:
    object_name: Full subscription or topic name.

  Raises:
    Error or TransientError.
  """
  is_sub = validate_full_name(object_name, 'subscriptions')
  is_topic = validate_full_name(object_name, 'topics')
  assert is_sub or is_topic, object_name

  def get_policy():
    return _call('GET', '%s:getIamPolicy' % object_name)

  # We can create topics on the fly (but not subscriptions).
  if is_topic:
    policy = _with_existing_topic(object_name, get_policy)
  else:
    policy = get_policy()

  copied = IAMPolicy(copy.deepcopy(policy))
  yield copied

  if copied.policy != policy:
    _call(
        'POST', '%s:setIamPolicy' % object_name,
        payload={'policy': copied.policy})


class IAMPolicy(object):
  """Wrapper around IAM policy dict for simpler modification.

  See https://cloud.google.com/pubsub/reference/rest/Shared.Types/Policy.
  """

  def __init__(self, policy):
    assert isinstance(policy, dict), policy
    self.policy = policy

  def members(self, role):
    """Returns list of members with given role, or empty list if none.

    Args:
      role: Role name, such as 'roles/viewer'.
    """
    for b in self.policy.get('bindings', []):
      if b.get('role') == role:
        return list(b.get('members', []))
    return []

  def add_member(self, role, member):
    """Adds a member to some role.

    Args:
      role: Role name, such as 'roles/viewer'.
      member: ID of the member to add, such as 'user:mike@example.com'.

    Returns:
      True if added, False if was already there.
    """
    role_dict = None
    for b in self.policy.get('bindings', []):
      if b.get('role') == role:
        role_dict = b
        break

    if role_dict is None:
      self.policy.setdefault('bindings', []).append({
          'role': role,
          'members': [member],
      })
      return True

    if member in role_dict.get('members', []):
      return False

    role_dict.setdefault('members', []).append(member)
    return True

  def remove_member(self, role, member):
    """Removes a member from some role.

    Args:
      role: Role name, such as 'roles/viewer'.
      member: ID of the member to add, such as 'user:mike@example.com'.

    Returns:
      True if removed, False if wasn't there.
    """
    for b in self.policy.get('bindings', []):
      if b.get('role') == role:
        members = b.get('members')
        if not members or member not in members:
          return False
        members.remove(member)
        return True
    return False


class SubscriptionHandler(webapp2.RequestHandler):
  """Base class for defining Pub/Sub subscription handlers."""
  # TODO(smut): Keep in datastore. See components/datastore_utils.
  ENDPOINT = None
  MAX_MESSAGES = None
  SUBSCRIPTION = None
  SUBSCRIPTION_PROJECT = None
  TOPIC = None
  TOPIC_PROJECT = None

  @classmethod
  def get_subscription_name(cls):
    """Returns full name of the subscription."""
    return full_subscription_name(cls.SUBSCRIPTION_PROJECT, cls.SUBSCRIPTION)

  @classmethod
  def get_topic_name(cls):
    """Returns full name of the topic."""
    return full_topic_name(cls.TOPIC_PROJECT, cls.TOPIC)

  @classmethod
  def unsubscribe(cls):
    """Unsubscribes from a Cloud Pub/Sub project."""
    ensure_subscription_deleted(cls.get_subscription_name())

  @classmethod
  def ensure_subscribed(cls, push=False):
    """Ensures a Cloud Pub/Sub subscription exists.

    Can also be used to change subscription type between push and pull.

    Args:
      push: Whether or not to create a push subscription. Defaults to pull.
    """
    ensure_subscription_exists(
        subscription=cls.get_subscription_name(),
        topic=cls.get_topic_name(),
        push_config={'pushEndpoint': cls.ENDPOINT} if push else {})

  @classmethod
  def is_subscribed(cls):
    """Returns whether or not a Cloud Pub/Sub subscription exists.

    Returns:
      True if the subscription exists, False otherwise.
    """
    return bool(get_subscription(cls.get_subscription_name()))

  def get(self):
    """Queries for Pub/Sub messages."""
    response = _call(
        'POST', '%s:pull' % self.get_subscription_name(),
        payload={
            'maxMessages': self.MAX_MESSAGES,
            'returnImmediately': True,
        },
    )
    message_ids = []
    for received_message in response.get('receivedMessages', []):
      attributes = received_message.get('message', {}).get('attributes', {})
      message = received_message.get('message', {}).get('data', '')
      logging.info(
          'Received Pub/Sub message:\n%s\nAttributes:\n%s', message, attributes)
      # TODO(smut): Process messages in parallel.
      self.process_message(message, attributes)
      message_ids.append(received_message['ackId'])
    if message_ids:
      _call(
          'POST', '%s:acknowledge' % self.get_subscription_name(),
          payload={'ackIds': message_ids})

  def post(self):
    """Handles a Pub/Sub push message."""
    # TODO(smut): Ensure message came from Cloud Pub/Sub.
    # Since anyone can post to this endpoint, we need to ensure the message
    # actually came from Cloud Pub/Sub. Unfortunately, there aren't any
    # useful headers set that can guarantee this.
    attributes = self.request.json.get('message', {}).get('attributes', {})
    message = self.request.json.get('message', {}).get('data', '')
    subscription = self.request.json.get('subscription')

    if subscription != self.get_subscription_name():
      self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
      logging.error('Ignoring unexpected subscription: %s', subscription)
      self.abort(403, 'Unexpected subscription: %s' % subscription)
      return

    logging.info(
        'Received Pub/Sub message:\n%s\nAttributes:\n%s', message, attributes)
    return self.process_message(message, attributes)

  def process_message(self, message, attributes):
    """Process a Pub/Sub message.

    Args:
      message: The message string.
      attributes: A dict of key/value pairs representing attributes associated
        with this message.

    Returns:
      A webapp2.Response instance, or None.
    """
    raise NotImplementedError()
