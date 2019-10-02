#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import net
from components import pubsub
from test_support import test_case


class PubSubTest(test_case.TestCase):
  def mock_requests(self, requests):
    def mocked_request(url, method, payload, scopes):
      self.assertEqual(['https://www.googleapis.com/auth/pubsub'], scopes)
      request = {
        'method': method,
        'payload': payload,
        'url': url,
      }
      if not requests:  # pragma: no cover
        self.fail('Unexpected request:\n%r' % request)
      expected = requests.pop(0)
      response = expected.pop('response', None)
      self.assertEqual(expected, request)
      if isinstance(response, net.Error):
        raise response
      future = ndb.Future()
      future.set_result(response)
      return future
    self.mock(net, 'json_request_async', mocked_request)
    return requests

  def test_validate_name(self):
    self.assertTrue(pubsub._validate_name('blah1234-_.~+%'))
    self.assertFalse(pubsub._validate_name('1blah1234-_.~+%'))
    self.assertFalse(pubsub._validate_name('long'*200))
    self.assertFalse(pubsub._validate_name(''))
    self.assertFalse(pubsub._validate_name('googbutwhy'))

  def test_full_topic_name(self):
    self.assertEqual(
        'projects/abc/topics/def', pubsub.full_topic_name('abc', 'def'))

  def test_full_subscription_name(self):
    self.assertEqual(
        'projects/abc/subscriptions/def',
        pubsub.full_subscription_name('abc', 'def'))

  def test_validate_full_name(self):
    self.assertTrue(
        pubsub.validate_full_name('projects/abc/topics/def', 'topics'))
    self.assertFalse(
        pubsub.validate_full_name('projects/abc/topics', 'topics'))
    self.assertFalse(
        pubsub.validate_full_name('what/abc/topics/def', 'topics'))
    self.assertFalse(
        pubsub.validate_full_name('projects//topics/def', 'topics'))
    self.assertFalse(
        pubsub.validate_full_name('projects/abc/nottopics/def', 'topics'))
    self.assertFalse(
        pubsub.validate_full_name('projects/abc/topics/1topic', 'topics'))

  def test_publish_ok(self):
    self.mock_requests([
      # First attempt. Encounters 404 due to non-existing topic.
      {
        'url': 'https://pubsub.googleapis.com/v1/projects/a/topics/def:publish',
        'method': 'POST',
        'payload': {
          'messages': [
            {
              'attributes': {'a': 1, 'b': 2},
              'data': 'bXNn',
            },
          ],
        },
        'response': net.NotFoundError('topic not found', 404, ''),
      },
      # Creates the topic.
      {
        'url': 'https://pubsub.googleapis.com/v1/projects/a/topics/def',
        'method': 'PUT',
        'payload': None,
      },
      # Second attempt, succeeds.
      {
        'url': 'https://pubsub.googleapis.com/v1/projects/a/topics/def:publish',
        'method': 'POST',
        'payload': {
          'messages': [
            {
              'attributes': {'a': 1, 'b': 2},
              'data': 'bXNn',
            },
          ],
        },
      },
    ])
    pubsub.publish('projects/a/topics/def', 'msg', {'a': 1, 'b': 2})

  def test_publish_transient_error(self):
    self.mock_requests([
      {
        'url': 'https://pubsub.googleapis.com/v1/projects/a/topics/def:publish',
        'method': 'POST',
        'payload': {
          'messages': [
            {
              'attributes': {'a': 1, 'b': 2},
              'data': 'bXNn',
            },
          ],
        },
        'response': net.Error('transient error', 500, ''),
      }
    ])
    with self.assertRaises(pubsub.TransientError):
      pubsub.publish('projects/a/topics/def', 'msg', {'a': 1, 'b': 2})

  def test_publish_fatal_error(self):
    self.mock_requests([
      {
        'url': 'https://pubsub.googleapis.com/v1/projects/a/topics/def:publish',
        'method': 'POST',
        'payload': {
          'messages': [
            {
              'attributes': {'a': 1, 'b': 2},
              'data': 'bXNn',
            },
          ],
        },
        'response': net.Error('fatal error', 403, ''),
      }
    ])
    with self.assertRaises(pubsub.Error):
      pubsub.publish('projects/a/topics/def', 'msg', {'a': 1, 'b': 2})

  def test_iam_policy_works(self):
    self.mock_requests([
      # Returns empty policy.
      {
        'url':
          'https://pubsub.googleapis.com/v1/projects/a/topics/def:getIamPolicy',
        'method': 'GET',
        'payload': None,
        'response': {'etag': 'blah'},
      },
      # Changes policy. Same etag is passed.
      {
        'url':
          'https://pubsub.googleapis.com/v1/projects/a/topics/def:setIamPolicy',
        'method': 'POST',
        'payload': {
          'policy': {
            'bindings': [{'role': 'role', 'members': ['member']}],
            'etag': 'blah',
          },
        },
      },
    ])
    with pubsub.iam_policy('projects/a/topics/def') as p:
      p.add_member('role', 'member')

  def test_iam_policy_skips_put_if_no_change(self):
    self.mock_requests([
      {
        'url':
          'https://pubsub.googleapis.com/v1/projects/a/topics/def:getIamPolicy',
        'method': 'GET',
        'payload': None,
        'response': {'etag': 'blah'},
      },
    ])
    with pubsub.iam_policy('projects/a/topics/def'):
      pass


class IAMPolicyTest(unittest.TestCase):
  def test_add_member(self):
    p = pubsub.IAMPolicy({})

    # Add new role and member.
    p.add_member('role1', 'member1')
    self.assertEqual(
        {'bindings': [{'members': ['member1'], 'role': 'role1'}]}, p.policy)

    # Adding same member is noop.
    p.add_member('role1', 'member1')
    self.assertEqual(
        {'bindings': [{'members': ['member1'], 'role': 'role1'}]}, p.policy)

    # Add another member to same role.
    p.add_member('role1', 'member2')
    self.assertEqual(
        {'bindings': [{'members': ['member1', 'member2'], 'role': 'role1'}]},
        p.policy)

    # List all member.
    self.assertEqual(['member1', 'member2'], p.members('role1'))
    self.assertEqual([], p.members('unknown role'))

    # Removing some.
    self.assertTrue(p.remove_member('role1', 'member1'))
    self.assertFalse(p.remove_member('role1', 'member1'))
    self.assertFalse(p.remove_member('unknown role', 'member1'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
