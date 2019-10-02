#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

# pylint: disable=unused-argument

import contextlib
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

import mock

from google.appengine.ext import ndb
from google.protobuf import empty_pb2

from test_support import test_case

from components import net
from components.prpc import client as prpc_client
from components.prpc import codes
from components.prpc.test import test_pb2
from components.prpc.test import test_prpc_pb2


class PRPCClientTestCase(test_case.TestCase):

  def make_test_client(self):
    return prpc_client.Client(
        'example.com', test_prpc_pb2.TestServiceDescription)

  def test_generated_methods(self):
    expected_methods = {
      'GiveAsync',
      'Give',
      'TakeAsync',
      'Take',
      'EchoAsync',
      'Echo',
    }

    members = dir(self.make_test_client())
    for m in expected_methods:
      self.assertIn(m, members)


  @contextlib.contextmanager
  def mocked_request_async(self, res=None):
    res = res or empty_pb2.Empty()
    with mock.patch('components.net.request_async', autospec=True) as m:
      m.return_value = ndb.Future()
      m.return_value.set_result(res.SerializeToString())
      yield


  def test_request(self):
    with self.mocked_request_async():
      req = test_pb2.GiveRequest(m=1)
      self.make_test_client().GiveAsync(req).get_result()

      net.request_async.assert_called_with(
          url='https://example.com/prpc/test.Test/Give',
          method='POST',
          payload=req.SerializeToString(),
          headers={
            'Content-Type': 'application/prpc; encoding=binary',
            'Accept': 'application/prpc; encoding=binary',
            'X-Prpc-Timeout': '10S',
          },
          scopes=None,
          service_account_key=None,
          delegation_token=None,
          deadline=10,
          max_attempts=4,
      )

  def give_creds(self, creds):
    self.make_test_client().Give(test_pb2.GiveRequest(), credentials=creds)

  def test_request_credentials_service_account(self):
    with self.mocked_request_async():
      self.give_creds(prpc_client.service_account_credentials())
      _, kwargs = net.request_async.call_args
      self.assertEqual(kwargs['scopes'], [net.EMAIL_SCOPE])

  def test_request_credentials_service_account_key(self):
    with self.mocked_request_async():
      self.give_creds(prpc_client.service_account_credentials('key'))
      _, kwargs = net.request_async.call_args
      self.assertEqual(kwargs['scopes'], [net.EMAIL_SCOPE])
      self.assertEqual(kwargs['service_account_key'], 'key')

  def test_request_credentials_delegation(self):
    with self.mocked_request_async():
      self.give_creds(prpc_client.delegation_credentials('token'))
      _, kwargs = net.request_async.call_args
      self.assertEqual(kwargs['delegation_token'], 'token')

  def test_request_credentials_composite(self):
    with self.mocked_request_async():
      self.give_creds(prpc_client.composite_call_credentials(
          prpc_client.service_account_credentials(),
          prpc_client.delegation_credentials('token'),
      ))
      _, kwargs = net.request_async.call_args
      self.assertEqual(kwargs['scopes'], [net.EMAIL_SCOPE])
      self.assertEqual(kwargs['delegation_token'], 'token')

  def test_request_timeout(self):
    with self.mocked_request_async():
      self.make_test_client().Give(test_pb2.GiveRequest(), timeout=20)
      _, kwargs = net.request_async.call_args
      self.assertEqual(kwargs['deadline'], 20)
      self.assertEqual(kwargs['headers']['X-Prpc-Timeout'], '20S')

  def test_response_ok(self):
    expected = test_pb2.TakeResponse(k=1)
    with self.mocked_request_async(res=expected):
      actual = self.make_test_client().Take(empty_pb2.Empty())
      self.assertEqual(actual, expected)

  @mock.patch('components.net.request_async', autospec=True)
  def test_response_protocol_error(self, request_async):
    request_async.side_effect = net.NotFoundError(
        msg='not found',
        status_code=404,
        response='not found',
        headers={
            # no X-Prpc-Grpc-Code header.
        },
    )
    with self.assertRaises(prpc_client.ProtocolError):
      self.make_test_client().Take(empty_pb2.Empty())

  @mock.patch('components.net.request_async', autospec=True)
  def test_response_rpc_error(self, request_async):
    request_async.side_effect = net.NotFoundError(
        msg='not found',
        status_code=404,
        response='not found',
        headers={
            'X-Prpc-Grpc-Code': str(codes.StatusCode.NOT_FOUND[0]),
        },
    )
    with self.assertRaises(prpc_client.RpcError) as cm:
      self.make_test_client().Take(empty_pb2.Empty())
    self.assertEqual(cm.exception.status_code, codes.StatusCode.NOT_FOUND)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
