#!/usr/bin/env python
# Copyright 2019 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import datetime
import json
import logging
import sys
import time
import unittest
from Crypto.PublicKey import RSA

import test_env
test_env.setup_test_env()

import cloudstorage
from google.appengine.ext import ndb

from protorpc.remote import protojson
import webtest

from components import auth
from components import auth_testing
from components import template
from components import utils
from test_support import test_case

import config
import gcs
import handlers_backend
import handlers_endpoints_v1
import model


def make_private_key():
  new_key = RSA.generate(1024)
  pem_key = base64.b64encode(new_key.exportKey('PEM'))
  config.settings()._ds_cfg.gs_private_key = pem_key


def hash_content(namespace, content):
  """Hashes uncompressed content."""
  d = model.get_hash(namespace)
  d.update(content)
  return d.hexdigest()


def message_to_dict(message):
  """Returns a JSON-ish dictionary corresponding to the RPC message."""
  return json.loads(protojson.encode_message(message))


def generate_digest(namespace, content):
  """Create a Digest from content (in a given namespace) for preupload.

  Arguments:
    content: the content to be hashed

  Returns:
    a Digest corresponding to the namespace pair
  """
  return handlers_endpoints_v1.Digest(
      digest=hash_content(namespace, content), size=len(content))


def generate_collection(namespace, contents):
  return handlers_endpoints_v1.DigestCollection(
      namespace=handlers_endpoints_v1.Namespace(namespace=namespace),
      items=[generate_digest(namespace, content) for content in contents])


def preupload_status_to_request(preupload_status, content):
  """Create a Storage/FinalizeRequest corresponding to a PreuploadStatus."""
  ticket = preupload_status.get('upload_ticket')
  url = preupload_status.get('gs_upload_url', None)
  if url is not None:
    return handlers_endpoints_v1.FinalizeRequest(upload_ticket=ticket)
  return handlers_endpoints_v1.StorageRequest(
      upload_ticket=ticket, content=content)


validate = handlers_endpoints_v1.TokenSigner.validate


class MainTest(test_case.EndpointsTestCase):
  """Tests the backend handlers."""

  api_service_cls = handlers_endpoints_v1.IsolateService
  store_prefix = 'https://storage.googleapis.com/sample-app/'

  APP_DIR = test_env.APP_DIR

  def setUp(self):
    """Creates a new app instance for every test case."""
    super(MainTest, self).setUp()
    self.testbed.init_blobstore_stub()
    self.testbed.init_urlfetch_stub()
    admin = auth.Identity(auth.IDENTITY_USER, 'admin@example.com')
    full_access_group = config.settings().auth.full_access_group
    auth.bootstrap_group(full_access_group, [admin])
    auth_testing.mock_get_current_identity(self, admin)
    version = utils.get_app_version()
    self.mock(utils, 'get_task_queue_host', lambda: version)
    self.testbed.setup_env(current_version_id='testbed.version')
    self.source_ip = '127.0.0.1'
    self.app = webtest.TestApp(
        handlers_backend.create_application(debug=True),
        extra_environ={'REMOTE_ADDR': self.source_ip})
    # add a private key; signing depends on config.settings()
    make_private_key()
    # Remove the check for dev server in should_push_to_gs().
    self.mock(utils, 'is_local_dev_server', lambda: False)

  def tearDown(self):
    template.reset()
    super(MainTest, self).tearDown()

  def store_request(self, namespace, content):
    """Generate a Storage/FinalizeRequest via preupload status."""
    collection = generate_collection(namespace, [content])
    response = self.call_api('preupload', message_to_dict(collection), 200)
    message = response.json.get(u'items', [{}])[0]
    return preupload_status_to_request(message, content)

  def test_cron_cleanup_trigger_expired(self):
    # Asserts that old entities are deleted through a task queue.

    # Removes the jitter.
    def _expiration_jitter(now, expiration):
      out = now + datetime.timedelta(seconds=expiration)
      return out, out
    self.mock(model, 'expiration_jitter', _expiration_jitter)
    now = self.mock_now(datetime.datetime(2020, 1, 2, 3, 4, 5), 0)
    request = self.store_request('sha1-raw', 'Foo')
    self.call_api('store_inline', message_to_dict(request))
    self.assertEqual(1, model.ContentEntry.query().count())

    self.mock_now(now, config.settings().default_expiration)
    self.app.get(
        '/internal/cron/cleanup/trigger/expired',
        headers={'X-AppEngine-Cron': 'true'})
    self.assertEqual(1, model.ContentEntry.query().count())
    self.assertEqual(0, self.execute_tasks())

    # Try again, second later.
    self.mock_now(now, config.settings().default_expiration+1)
    self.app.get(
        '/internal/cron/cleanup/trigger/expired',
        headers={'X-AppEngine-Cron': 'true'})
    self.assertEqual(1, model.ContentEntry.query().count())

    self.assertEqual(1, self.execute_tasks())
    # Boom it's gone.
    self.assertEqual(0, model.ContentEntry.query().count())

  def test_cron_cleanup_trigger_orphan(self):
    now = 12345678.
    self.mock(time, 'time', lambda: now)
    # Asserts that lost GCS files are deleted through a task queue.
    def _list_files(bucket):
      self.assertEqual('sample-app', bucket)
      # Include two files, one recent, one old.
      recent = cloudstorage.GCSFileStat(
          filename=bucket + '/namespace/recent',
          st_size=10,
          etag='123',
          st_ctime=now - 24*60*60,
          content_type=None,
          metadata=None,
          is_dir=False)
      old = cloudstorage.GCSFileStat(
          filename=bucket + '/namespace/old',
          st_size=11,
          etag='123',
          st_ctime=now - 24*60*60-1,
          content_type=None,
          metadata=None,
          is_dir=False)
      return [('namespace/recent', recent), ('namespace/old', old)]
    self.mock(gcs, 'list_files', _list_files)

    called = []
    @ndb.tasklet
    def _delete_file_async(bucket, filename, ignore_missing):
      called.append(filename)
      self.assertEqual('sample-app', bucket)
      self.assertEqual(True, ignore_missing)
      raise ndb.Return(None)
    self.mock(gcs, 'delete_file_async', _delete_file_async)

    self.app.get(
        '/internal/cron/cleanup/trigger/orphan',
        headers={'X-AppEngine-Cron': 'true'})
    self.assertEqual(1, self.execute_tasks())
    self.assertEqual(['namespace/old'], called)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
