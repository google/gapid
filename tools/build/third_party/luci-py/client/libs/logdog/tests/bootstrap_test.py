#!/usr/bin/env python
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys
import unittest

ROOT_DIR = os.path.dirname(os.path.abspath(os.path.join(
    __file__.decode(sys.getfilesystemencoding()),
    os.pardir, os.pardir, os.pardir)))
sys.path.insert(0, ROOT_DIR)

from libs.logdog import bootstrap, stream


# pylint: disable=protected-access

class BootstrapTestCase(unittest.TestCase):

  def setUp(self):
    self.env = {
        bootstrap.ButlerBootstrap._ENV_PROJECT: 'test-project',
        bootstrap.ButlerBootstrap._ENV_PREFIX: 'foo/bar',
        bootstrap.ButlerBootstrap._ENV_STREAM_SERVER_PATH: 'fake:path',
        bootstrap.ButlerBootstrap._ENV_COORDINATOR_HOST: 'example.appspot.com',
        bootstrap.ButlerBootstrap._ENV_NAMESPACE: 'something',
    }

  @classmethod
  def setUpClass(cls):
    class TestStreamClient(stream.StreamClient):
      @classmethod
      def _create(cls, _value, **kwargs):
        return cls(**kwargs)

      def _connect_raw(self):
        class fakeFile(object):
          def write(self, data):
            pass
        return fakeFile()
    cls.reg = stream.StreamProtocolRegistry()
    cls.reg.register_protocol('test', TestStreamClient)

  def testProbeSucceeds(self):
    bs = bootstrap.ButlerBootstrap.probe(self.env)
    self.assertEqual(bs, bootstrap.ButlerBootstrap(
      project='test-project',
      prefix='foo/bar',
      streamserver_uri='fake:path',
      coordinator_host='example.appspot.com',
      namespace='something'))

  def testProbeNoBootstrapRaisesError(self):
    self.assertRaises(bootstrap.NotBootstrappedError,
        bootstrap.ButlerBootstrap.probe, env={})

  def testNoNamespaceOK(self):
    del self.env[bootstrap.ButlerBootstrap._ENV_NAMESPACE]
    bootstrap.ButlerBootstrap.probe(self.env)

  def testProbeBadNamespaceRaisesError(self):
    self.env[bootstrap.ButlerBootstrap._ENV_NAMESPACE] = '!!! invalid'
    self.assertRaises(bootstrap.NotBootstrappedError,
        bootstrap.ButlerBootstrap.probe, env=self.env)

  def testProbeMissingProjectRaisesError(self):
    self.env.pop(bootstrap.ButlerBootstrap._ENV_PROJECT)
    self.assertRaises(bootstrap.NotBootstrappedError,
        bootstrap.ButlerBootstrap.probe, env=self.env)

  def testProbeMissingPrefixRaisesError(self):
    self.env.pop(bootstrap.ButlerBootstrap._ENV_PREFIX)
    self.assertRaises(bootstrap.NotBootstrappedError,
        bootstrap.ButlerBootstrap.probe, env=self.env)

  def testProbeInvalidPrefixRaisesError(self):
    self.env[bootstrap.ButlerBootstrap._ENV_PREFIX] = '!!! not valid !!!'
    self.assertRaises(bootstrap.NotBootstrappedError,
        bootstrap.ButlerBootstrap.probe, env=self.env)

  def testCreateStreamClient(self):
    bs = bootstrap.ButlerBootstrap(
      project='test-project',
      prefix='foo/bar',
      streamserver_uri='test:',
      coordinator_host='example.appspot.com',
      namespace='something/deep')
    sc = bs.stream_client(reg=self.reg)
    self.assertEqual(sc.prefix, 'foo/bar')
    self.assertEqual(sc.coordinator_host, 'example.appspot.com')
    self.assertEqual(sc.open_text('foobar').params.name,
                     'something/deep/foobar')

  def testCreateStreamClientNoNamespace(self):
    bs = bootstrap.ButlerBootstrap(
      project='test-project',
      prefix='foo/bar',
      streamserver_uri='test:',
      coordinator_host='example.appspot.com',
      namespace='')
    sc = bs.stream_client(reg=self.reg)
    self.assertEqual(sc.open_text('foobar').params.name, 'foobar')


if __name__ == '__main__':
  unittest.main()
