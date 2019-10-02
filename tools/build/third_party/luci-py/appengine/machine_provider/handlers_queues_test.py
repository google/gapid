#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for handlers_queues.py."""

import datetime
import json
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from protorpc.remote import protojson
import webtest

from components import auth_testing
from components.machine_provider import rpc_messages
from test_support import test_case

import handlers_queues
import models


class MaybeNotifyBackendTest(test_case.TestCase):
  """Tests for handlers_queues.maybe_notify_backend."""

  def test_notify(self):
    def publish(topic, message, attributes):
      self.assertEqual(topic, 'projects/project/topics/topic')
      self.assertEqual(message, 'asdf')
      self.assertEqual(attributes, {'hostname': 'fake-host', 'k': 'v'})
    self.mock(handlers_queues.pubsub, 'publish', publish)

    policies = {
        'backend_attributes': [{'key': 'k', 'value': 'v'}],
        'backend_project': 'project',
        'backend_topic': 'topic',
    }

    handlers_queues.maybe_notify_backend('asdf', 'fake-host', policies)


class MaybeNotifyLesseeTest(test_case.TestCase):
  """Tests for handlers_queues.maybe_notify_lessee."""

  def test_notify(self):
    def publish(topic, message, attributes):
      self.assertEqual(topic, 'projects/project/topics/topic')
      self.assertEqual(
          json.loads(message),
          {'hostname': 'fake-host', 'lease_expiration_ts': 0},
      )
      self.assertFalse(attributes)
    self.mock(handlers_queues.pubsub, 'publish', publish)

    request = {
        'pubsub_project': 'project',
        'pubsub_topic': 'topic',
    }
    response = {
        'hostname': 'fake-host',
        'lease_expiration_ts': 0,
    }

    handlers_queues.maybe_notify_lessee(request, response)


class LeaseRequestFulfillerTest(test_case.TestCase):
  """Tests for handlers_queues.LeaseRequestFulfiller."""

  def setUp(self):
    super(LeaseRequestFulfillerTest, self).setUp()
    app = handlers_queues.create_queues_app()
    self.app = webtest.TestApp(app)

  def test_fulfill(self):
    def publish_multi(topic, messages):
      self.assertEqual(topic, 'projects/project/topics/topic')
      self.assertEqual(
          messages,
          {
              'LEASED': {'lease_expiration_ts': '1'},
              'CONNECT': {'swarming_server': 'server'},
          },
      )
    self.mock(handlers_queues.pubsub, 'publish_multi', publish_multi)

    self.app.post(
        '/internal/queues/fulfill-lease-request',
        headers={'X-AppEngine-QueueName': 'fulfill-lease-request'},
        params={
            'machine_project': 'project',
            'machine_topic': 'topic',
            'policies': json.dumps({}),
            'request_json': json.dumps({
                'on_lease': {'swarming_server': 'server'},
             }),
            'response_json': json.dumps({
                'hostname': 'fake-host',
                'lease_expiration_ts': 1,
            }),
        },
    )


class ReclaimTest(test_case.TestCase):
  """Tests for handlers_queues.reclaim."""

  def test_not_found(self):
    key = ndb.Key(models.CatalogMachineEntry, 'fake-key')
    handlers_queues.reclaim(key)
    self.assertFalse(key.get())

  def test_reclaimed(self):
    lease_key = models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        machine_id='fake-host',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            hostname='fake-host',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(),
        lease_id=lease_key.id(),
    ).put()

    handlers_queues.reclaim(machine_key)
    self.assertFalse(lease_key.get().machine_id)
    self.assertFalse(lease_key.get().response.hostname)
    self.assertFalse(machine_key.get())


if __name__ == '__main__':
  unittest.main()
