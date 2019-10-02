#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""High level test for Primary <-> Replica replication logic.

It launches two local services (Primary and Replica) via dev_appserver and sets
up auth db replication between them.
"""

import base64
import hashlib
import logging
import os
import shutil
import sys
import tempfile
import time
import unittest
import zlib

from tool_support import gae_sdk_utils
from tool_support import local_app

# /appengine/auth_service/.
APP_DIR = os.path.dirname(os.path.abspath(__file__))
# /appengine/auth_service/test_replica_app/.
REPLICA_APP_DIR = os.path.join(APP_DIR, 'test_replica_app')


class ReplicationTest(unittest.TestCase):
  def setUp(self):
    super(ReplicationTest, self).setUp()
    self.root = tempfile.mkdtemp(prefix='replication_smoke_test')
    self.auth_service = local_app.LocalApplication(
        APP_DIR, 9500, False, self.root)
    self.replica = local_app.LocalApplication(
        REPLICA_APP_DIR, 9600, False, self.root)
    # Launch both first, only then wait for them to come online.
    apps = [self.auth_service, self.replica]
    for app in apps:
      app.start()
    for app in apps:
      app.ensure_serving()
      app.client.login_as_admin('test@example.com')

  def tearDown(self):
    try:
      self.auth_service.stop()
      self.replica.stop()
      shutil.rmtree(self.root)
      if self.has_failed() or self.maxDiff is None:
        self.auth_service.dump_log()
        self.replica.dump_log()
    finally:
      super(ReplicationTest, self).tearDown()

  def has_failed(self):
    # pylint: disable=E1101
    return not self._resultForDoCleanups.wasSuccessful()

  def test_replication_workflow(self):
    """Tests full Replica <-> Primary flow (linking and replication)."""
    self.link_replica_to_primary()
    self.check_oauth_config_replication()
    self.check_group_replication()
    self.check_ip_whitelist_replication()
    self.check_snapshot_endpoint()

  def link_replica_to_primary(self):
    """Links replica to primary."""
    logging.info('Linking replica to primary')

    # Verify initial state: no linked services on primary.
    linked_services = self.auth_service.client.json_request(
        '/auth_service/api/v1/services').body
    self.assertEqual([], linked_services['services'])

    # Step 1. Generate a link to associate |replica| to |auth_service|.
    app_id = '%s@localhost:%d' % (self.replica.app_id, self.replica.port)
    response = self.auth_service.client.json_request(
        resource='/auth_service/api/v1/services/%s/linking_url' % app_id,
        body={},
        headers={'X-XSRF-Token': self.auth_service.client.xsrf_token})
    self.assertEqual(201, response.http_code)

    # URL points to HTML page on the replica.
    linking_url = response.body['url']
    self.assertTrue(
        linking_url.startswith('%s/auth/link?t=' % self.replica.url))

    # Step 2. "Click" this link. It should associates Replica with Primary via
    # behind-the-scenes service <-> service URLFetch call.
    response = self.replica.client.request(
        resource=linking_url,
        body='',
        headers={'X-XSRF-Token': self.replica.client.xsrf_token})
    self.assertEqual(200, response.http_code)
    self.assertIn('Success!', response.body)

    # Verify primary knows about new replica now.
    linked_services = self.auth_service.client.json_request(
        '/auth_service/api/v1/services').body
    self.assertEqual(1, len(linked_services['services']))
    service = linked_services['services'][0]
    self.assertEqual(self.replica.app_id, service['app_id'])
    self.assertEqual(self.replica.url, service['replica_url'])

    # Verify replica knows about the primary now.
    replica_state = self.replica.client.json_request(
        '/auth/api/v1/server/state').body
    self.assertEqual('replica', replica_state['mode'])
    self.assertEqual(
        self.auth_service.app_id,
        replica_state['replication_state']['primary_id'])
    self.assertEqual(
        self.auth_service.url,
        replica_state['replication_state']['primary_url'])

  def wait_for_sync(self, timeout=4):
    """Waits for replica to catch up to primary."""
    logging.info('Waiting for replica to catch up to primary')
    primary_rev = self.auth_service.client.json_request(
        '/auth/api/v1/server/state').body['replication_state']['auth_db_rev']
    deadline = time.time() + timeout
    while time.time() < deadline:
      replica_rev = self.replica.client.json_request(
        '/auth/api/v1/server/state').body['replication_state']['auth_db_rev']
      if replica_rev == primary_rev:
        return
      time.sleep(0.1)
    self.fail('Replica couldn\'t synchronize to primary fast enough')

  def check_oauth_config_replication(self):
    """Verifies changes to OAuth config propagate to replica."""
    oauth_config = {
      u'additional_client_ids': [u'a', u'b'],
      u'client_id': u'some-id',
      u'client_not_so_secret': u'secret',
      u'primary_url': u'http://localhost:9500',
      u'token_server_url': u'https://example.com',
    }
    response = self.auth_service.client.json_request(
        resource='/auth/api/v1/server/oauth_config',
        body=oauth_config,
        headers={'X-XSRF-Token': self.auth_service.client.xsrf_token})
    self.assertEqual(200, response.http_code)

    # Ensure replica got the update.
    self.wait_for_sync()
    response = self.replica.client.json_request(
        '/auth/api/v1/server/oauth_config')
    self.assertEqual(200, response.http_code)
    self.assertEqual(oauth_config, response.body)

  def check_group_replication(self):
    """Verifies changes to groups propagate to replica."""
    logging.info('Creating group')
    group = {
      'name': 'some-group',
      'members': ['user:jekyll@example.com', 'user:hyde@example.com'],
      'globs': ['user:*@google.com'],
      'nested': [],
      'description': 'Blah',
    }
    response = self.auth_service.client.json_request(
        resource='/auth/api/v1/groups/some-group',
        body=group,
        headers={'X-XSRF-Token': self.auth_service.client.xsrf_token})
    self.assertEqual(201, response.http_code)

    # Read it back from primary to grab created_ts and modified_ts.
    response = self.auth_service.client.json_request(
        '/auth/api/v1/groups/some-group',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(200, response.http_code)
    group = response.body

    # Ensure replica got the update.
    self.wait_for_sync()
    response = self.replica.client.json_request(
        '/auth/api/v1/groups/some-group',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(200, response.http_code)
    self.assertEqual(group, response.body)

    logging.info('Modifying group')
    group = {
      'name': 'some-group',
      'members': ['user:hyde@example.com'],
      'globs': ['user:*@google.com'],
      'nested': [],
      'description': 'Some other blah',
    }
    response = self.auth_service.client.json_request(
        resource='/auth/api/v1/groups/some-group',
        body=group,
        headers={'X-XSRF-Token': self.auth_service.client.xsrf_token},
        method='PUT')
    self.assertEqual(200, response.http_code)

    # Read it back from primary to grab created_ts and modified_ts.
    response = self.auth_service.client.json_request(
        '/auth/api/v1/groups/some-group',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(200, response.http_code)
    group = response.body

    # Ensure replica got the update.
    self.wait_for_sync()
    response = self.replica.client.json_request(
        '/auth/api/v1/groups/some-group',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(200, response.http_code)
    self.assertEqual(group, response.body)

    logging.info('Deleting group')
    response = self.auth_service.client.json_request(
        resource='/auth/api/v1/groups/some-group',
        headers={'X-XSRF-Token': self.auth_service.client.xsrf_token},
        method='DELETE')
    self.assertEqual(200, response.http_code)

    # Ensure replica got the update.
    self.wait_for_sync()
    response = self.replica.client.json_request(
        '/auth/api/v1/groups/some-group',
        headers={'Cache-Control': 'no-cache'})
    self.assertEqual(404, response.http_code)

  def check_ip_whitelist_replication(self):
    """Verifies changes to IP whitelist propagate to replica."""
    # TODO(vadimsh): Implement once IP whitelist is accessible via API.

  def check_snapshot_endpoint(self):
    """Verifies /auth_service/api/v1/authdb/revisions/ works."""
    response = self.auth_service.client.json_request(
        '/auth_service/api/v1/authdb/revisions/latest')
    self.assertEqual(200, response.http_code)
    latest = response.body['snapshot']

    response = self.auth_service.client.json_request(
        '/auth_service/api/v1/authdb/revisions/%d' % latest['auth_db_rev'])
    self.assertEqual(200, response.http_code)
    at_rev = response.body['snapshot']

    self.assertEqual(latest, at_rev)
    deflated = base64.b64decode(latest['deflated_body'])
    self.assertEqual(
        latest['sha256'], hashlib.sha256(zlib.decompress(deflated)).hexdigest())


if __name__ == '__main__':
  gae_sdk_utils.setup_gae_env()
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
