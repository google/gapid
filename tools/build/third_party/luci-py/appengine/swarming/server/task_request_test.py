#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import random
import string
import sys
import unittest

import test_env
test_env.setup_test_env()

from google.protobuf import duration_pb2
from google.protobuf import timestamp_pb2

from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components import auth_testing
from components import utils
from test_support import test_case

from proto.api import swarming_pb2
from server import bq_state
from server import config
from server import pools_config
from server import task_pack
from server import task_request


# pylint: disable=W0212


def _gen_cipd_input(**kwargs):
  """Creates a CipdInput."""
  args = {
    u'client_package': task_request.CipdPackage(
        package_name=u'infra/tools/cipd/${platform}',
        version=u'git_revision:deadbeef'),
    u'packages': [
      task_request.CipdPackage(
          package_name=u'rm',
          path=u'bin',
          version=u'git_revision:deadbeef'),
    ],
    u'server': u'https://chrome-infra-packages.appspot.com'
  }
  args.update(kwargs)
  return task_request.CipdInput(**args)


def _gen_properties(**kwargs):
  """Creates a TaskProperties."""
  args = {
    u'cipd_input': _gen_cipd_input(),
    u'command': [u'command1', u'arg1'],
    u'dimensions': {
      u'OS': [u'Windows-3.1.1'],
      u'hostname': [u'localhost'],
      u'pool': [u'default'],
    },
    u'env': {u'foo': u'bar', u'joe': u'2'},
    u'env_prefixes': {u'PATH': [u'local/path']},
    u'execution_timeout_secs': 30,
    u'grace_period_secs': 30,
    u'idempotent': False,
    u'inputs_ref': task_request.FilesRef(
        isolatedserver=u'https://isolateserver.appspot.com',
        namespace=u'default-gzip'),
    u'io_timeout_secs': None,
    u'has_secret_bytes': False,
  }
  args.update(kwargs)
  args[u'dimensions_data'] = args.pop(u'dimensions')
  return task_request.TaskProperties(**args)


def _gen_request_slices(**kwargs):
  """Creates a TaskRequest."""
  template_apply = kwargs.pop('_template_apply', task_request.TEMPLATE_AUTO)
  now = utils.utcnow()
  args = {
    u'created_ts': now,
    u'manual_tags': [u'tag:1'],
    u'name': u'Request name',
    u'priority': 50,
    u'task_slices': [
      task_request.TaskSlice(expiration_secs=30, properties=_gen_properties()),
    ],
    u'user': u'Jesus',
  }
  args.update(kwargs)
  # Note that ndb model constructor accepts dicts for structured properties.
  req = task_request.TaskRequest(**args)
  task_request.init_new_request(req, True, template_apply)
  return req


def _gen_request(properties=None, **kwargs):
  """Creates a TaskRequest with a single TaskSlice."""
  return _gen_request_slices(
      task_slices=[
        task_request.TaskSlice(
            expiration_secs=30,
            properties=properties or _gen_properties()),
      ],
      **kwargs)


def _gen_secret(req, secret_bytes):
  assert req.key
  sb = task_request.SecretBytes(secret_bytes=secret_bytes)
  sb.key = req.secret_bytes_key
  return sb


def _gen_task_template(cache=None, cipd_package=None, env=None):
  """Builds an unverified pools_config.TaskTemplate for use with
  _set_pool_config_with_templates.

  Args:
    cache (None|dict{name: path}) - cache entries to set.
    cipd_package (None|dict{(path, pkg): version}) - cipd packages to set.
    env (None|dict{var: value|(value, prefix)|(value, prefix, soft)}) -
        envvars to set. The key is always the envvar to set, and the value may
        be:
          * the envvar value as a string (prefix=() and soft=False)
          * A (value, prefix) tuple (soft=False)
          * A (value, prefix, soft) tuple

  Returns constructed pools_config.TaskTemplate.
  """
  def env_value(var, combo_value):
    prefix, soft = (), False
    if isinstance(combo_value, tuple):
      assert len(combo_value) in (2, 3), (
          'unexpected tuple length: %r' % combo_value)
      if len(combo_value) == 2:
        value, prefix = combo_value
      else:
        value, prefix, soft = combo_value
    else:
      value = unicode(combo_value)

    return pools_config.Env(var, value, tuple(map(unicode, prefix)), soft)

  return pools_config.TaskTemplate(
    cache=sorted(
      pools_config.CacheEntry(unicode(name), unicode(path))
      for name, path in (cache or {}).iteritems()
    ),
    cipd_package=sorted(
      pools_config.CipdPackage(unicode(path), unicode(pkg), unicode(version))
      for (path, pkg), version in (cipd_package or {}).iteritems()
    ),
    env=sorted(
      env_value(unicode(var), value)
      for var, value in (env or {}).iteritems()
    ),
    inclusions=(),
  )


class Prop(object):
  _name = 'foo'


class TestCase(test_case.TestCase):
  def setUp(self):
    super(TestCase, self).setUp()
    auth_testing.mock_get_current_identity(self)


class TaskRequestPrivateTest(TestCase):
  def test_validate_task_run_id(self):
    self.assertEqual(
        '1d69b9f088008811',
        task_request._validate_task_run_id(Prop(), '1d69b9f088008811'))
    self.assertEqual(None, task_request._validate_task_run_id(Prop(), ''))
    with self.assertRaises(ValueError):
      task_request._validate_task_run_id(Prop(), '1')

  def test_apply_template_simple(self):
    tt = _gen_task_template(
      cache={'cache': 'c'},
      cipd_package={('cipd', 'some/pkg'): 'latest'},
      env={'ENV': ('1', ['a'])},
    )
    p = task_request.TaskProperties()
    task_request._apply_task_template(tt, p)
    self.assertEqual(p, task_request.TaskProperties(
      env={u'ENV': u'1'},
      env_prefixes={u'ENV': [u'a']},
      caches=[task_request.CacheEntry(name=u'cache', path=u'c')],
      cipd_input=task_request.CipdInput(
          packages=[task_request.CipdPackage(
              package_name=u'some/pkg', path=u'cipd', version=u'latest')])
    ))

  def test_apply_template_env_set_error(self):
    tt = _gen_task_template(env={'ENV': ('1', ['a'])})
    p = task_request.TaskProperties(env={u'ENV': u'10'})
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "request.env[u'ENV'] conflicts with pool's template")

  def test_apply_template_env_prefix_set_error(self):
    tt = _gen_task_template(env={'ENV': ('1', ['a'])})
    p = task_request.TaskProperties(env_prefixes={u'ENV': [u'b']})
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "request.env_prefixes[u'ENV'] conflicts with pool's template")

  def test_apply_template_env_override_soft(self):
    tt = _gen_task_template(env={'ENV': ('1', ['a'], True)})
    p = task_request.TaskProperties(env={u'ENV': u'2'})
    task_request._apply_task_template(tt, p)
    self.assertEqual(p, task_request.TaskProperties(
        env={u'ENV': u'2'},
        env_prefixes={u'ENV': [u'a']},
    ))

  def test_apply_template_env_prefixes_append_soft(self):
    tt = _gen_task_template(env={'ENV': ('1', ['a'], True)})
    p = task_request.TaskProperties(env_prefixes={u'ENV': [u'b']})
    task_request._apply_task_template(tt, p)
    self.assertEqual(p, task_request.TaskProperties(
      env={u'ENV': u'1'},
      env_prefixes={u'ENV': [u'a', u'b']},
    ))

  def test_apply_template_conflicting_cache(self):
    tt = _gen_task_template(cache={'c': 'C'})
    p = task_request.TaskProperties(
      caches=[task_request.CacheEntry(name='c', path='B')])
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "request.cache['c'] conflicts with pool's template")

  def test_apply_template_conflicting_cache_path(self):
    tt = _gen_task_template(cache={'c': 'C'})
    p = task_request.TaskProperties(
      caches=[task_request.CacheEntry(name='other', path='C')])
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "u'C': directory has conflicting owners: task cache 'other' "
        "and task template cache u'c'")

  def test_apply_template_conflicting_cache_cipd_path(self):
    tt = _gen_task_template(cache={'c': 'C'})
    p = task_request.TaskProperties(
        cipd_input=task_request.CipdInput(
            packages=[
              task_request.CipdPackage(
                path='C', package_name='pkg', version='latest')]))
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "u'C': directory has conflicting owners: task cipd['pkg:latest'] "
        "and task template cache u'c'")

  def test_apply_template_conflicting_cipd_package(self):
    tt = _gen_task_template(cipd_package={('C', 'pkg'): 'latest'})
    p = task_request.TaskProperties(
        cipd_input=task_request.CipdInput(
            packages=[
              task_request.CipdPackage(
                path='C', package_name='other', version='latest')]))
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "u'C': directory has conflicting owners: task cipd['other:latest'] "
        "and task template cipd[u'pkg:latest']")

  def test_apply_template_conflicting_cipd_cache_path(self):
    tt = _gen_task_template(cipd_package={('C', 'pkg'): 'latest'})
    p = task_request.TaskProperties(
      caches=[task_request.CacheEntry(name='other', path='C')])
    with self.assertRaises(ValueError) as ex:
      task_request._apply_task_template(tt, p)
    self.assertEqual(
        ex.exception.message,
        "u'C': directory has conflicting owners: task cache 'other' "
        "and task template cipd[u'pkg:latest']")


class TaskRequestApiTest(TestCase):
  def setUp(self):
    super(TaskRequestApiTest, self).setUp()
    # pool_configs is a mapping of pool name -> pools_config.PoolConfig. Tests
    # can modify this to have pools_config.get_pool_config return the
    # appropriate data.
    self._pool_configs = {}
    self.mock(pools_config, 'get_pool_config', self._pool_configs.get)
    self._enqueue_calls = []
    self._enqueue_orig = self.mock(utils, 'enqueue_task', self._enqueue)

  def tearDown(self):
    try:
      self.assertFalse(self._enqueue_calls)
    finally:
      super(TaskRequestApiTest, self).tearDown()

  def _enqueue(self, *args, **kwargs):
    self._enqueue_calls.append((args, kwargs))
    return self._enqueue_orig(*args, use_dedicated_module=False, **kwargs)

  def test_all_apis_are_tested(self):
    # Ensures there's a test for each public API.
    module = task_request
    expected = frozenset(
        i for i in dir(module)
        if i[0] != '_' and hasattr(getattr(module, i), 'func_name'))
    missing = expected - frozenset(
        i[5:] for i in dir(self) if i.startswith('test_'))
    self.assertFalse(missing)

  def test_get_automatic_tags(self):
    req = _gen_request()
    expected = set((
        u'hostname:localhost',
        u'OS:Windows-3.1.1',
        u'pool:default',
        u'priority:50',
        u'service_account:none',
        u'user:Jesus'))
    self.assertEqual(expected, task_request.get_automatic_tags(req, 0))
    with self.assertRaises(IndexError):
      task_request.get_automatic_tags(req, 1)

  def test_get_automatic_tags_slices(self):
    # Repeated TaskSlice.
    slices = [
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(
              dimensions={u'gpu': [u'1234:5678'], u'pool': [u'GPU']})),
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(
              dimensions={u'gpu': [u'none'], u'pool': [u'GPU']})),
    ]
    req = _gen_request_slices(task_slices=slices)
    expected = set((
        u'gpu:1234:5678',
        u'pool:GPU',
        u'priority:50',
        u'service_account:none',
        u'user:Jesus'))
    self.assertEqual(expected, task_request.get_automatic_tags(req, 0))
    expected = set((
        u'gpu:none',
        u'pool:GPU',
        u'priority:50',
        u'service_account:none',
        u'user:Jesus'))
    self.assertEqual(expected, task_request.get_automatic_tags(req, 1))
    with self.assertRaises(IndexError):
      task_request.get_automatic_tags(req, 2)

  def test_create_termination_task(self):
    request = task_request.create_termination_task(u'some-bot',
        wait_for_capacity=True)
    self.assertTrue(request.task_slice(0).properties.is_terminate)

  def test_new_request_key(self):
    for _ in xrange(3):
      delta = utils.utcnow() - task_request._BEGINING_OF_THE_WORLD
      now = int(round(delta.total_seconds() * 1000.))
      key = task_request.new_request_key()
      # Remove the XOR.
      key_id = key.integer_id() ^ task_pack.TASK_REQUEST_KEY_ID_MASK
      timestamp = key_id >> 20
      randomness = (key_id >> 4) & 0xFFFF
      version = key_id & 0xF
      self.assertLess(abs(timestamp - now), 1000)
      self.assertEqual(1, version)
      if randomness:
        break
    else:
      self.fail('Failed to find randomness')

  def test_new_request_key_zero(self):
    def getrandbits(i):
      self.assertEqual(i, 16)
      return 0x7766
    self.mock(random, 'getrandbits', getrandbits)
    self.mock_now(task_request._BEGINING_OF_THE_WORLD)
    key = task_request.new_request_key()
    # Remove the XOR.
    key_id = key.integer_id() ^ task_pack.TASK_REQUEST_KEY_ID_MASK
    #   00000000000 7766 1
    #     ^          ^   ^
    #     |          |   |
    #  since 2010    | schema version
    #                |
    #               rand
    self.assertEqual('0x0000000000077661', '0x%016x' % key_id)

  def test_new_request_key_end(self):
    def getrandbits(i):
      self.assertEqual(i, 16)
      return 0x7766
    self.mock(random, 'getrandbits', getrandbits)
    days_until_end_of_the_world = 2**43 / 24. / 60. / 60. / 1000.
    num_days = int(days_until_end_of_the_world)
    # Remove 1ms to not overflow.
    num_seconds = (
        (days_until_end_of_the_world - num_days) * 24. * 60. * 60. - 0.001)
    self.assertEqual(101806, num_days)
    self.assertEqual(278, int(num_days / 365.3))
    now = (task_request._BEGINING_OF_THE_WORLD +
        datetime.timedelta(days=num_days, seconds=num_seconds))
    self.mock_now(now)
    key = task_request.new_request_key()
    # Remove the XOR.
    key_id = key.integer_id() ^ task_pack.TASK_REQUEST_KEY_ID_MASK
    #   7ffffffffff 7766 1
    #     ^          ^   ^
    #     |          |   |
    #  since 2010    | schema version
    #                |
    #               rand
    self.assertEqual('0x7ffffffffff77661', '0x%016x' % key_id)

  def test_validate_request_key(self):
    task_request.validate_request_key(task_pack.unpack_request_key('11'))
    with self.assertRaises(ValueError):
      task_request.validate_request_key(ndb.Key('TaskRequest', 1))

  def test_init_new_request(self):
    parent = _gen_request()
    # Parent entity must have a valid key id and be stored.
    parent.key = task_request.new_request_key()
    parent.put()
    # The reference is to the TaskRunResult.
    parent_id = task_pack.pack_request_key(parent.key) + u'1'
    req = _gen_request(
        properties=_gen_properties(
            idempotent=True,
            relative_cwd=u'deeep',
            has_secret_bytes=True),
        parent_task_id=parent_id)
    # TaskRequest with secret must have a valid key.
    req.key = task_request.new_request_key()
    # Needed for the get() call below.
    req.put()
    sb = _gen_secret(req, 'I am a banana')
    # Needed for properties_hash() call.
    sb.put()
    expected_properties = {
      'caches': [],
      'cipd_input': {
        'client_package': {
          'package_name': u'infra/tools/cipd/${platform}',
          'path': None,
          'version': u'git_revision:deadbeef',
        },
        'packages': [{
          'package_name': u'rm',
          'path': u'bin',
          'version': u'git_revision:deadbeef',
        }],
        'server': u'https://chrome-infra-packages.appspot.com'
      },
      'command': [u'command1', u'arg1'],
      'relative_cwd': u'deeep',
      'dimensions': {
        u'OS': [u'Windows-3.1.1'],
        u'hostname': [u'localhost'],
        u'pool': [u'default'],
      },
      'env': {u'foo': u'bar', u'joe': u'2'},
      'env_prefixes': {u'PATH': [u'local/path']},
      'extra_args': [],
      'execution_timeout_secs': 30,
      'grace_period_secs': 30,
      'has_secret_bytes': True,
      'idempotent': True,
      'inputs_ref': {
        'isolated': None,
        'isolatedserver': u'https://isolateserver.appspot.com',
        'namespace': u'default-gzip',
      },
      'io_timeout_secs': None,
      'outputs': [],
    }
    expected_request = {
      'authenticated': auth_testing.DEFAULT_MOCKED_IDENTITY,
      'name': u'Request name',
      'parent_task_id': unicode(parent_id),
      'priority': 50,
      'pubsub_topic': None,
      'pubsub_userdata': None,
      'service_account': u'none',
      'tags': [
        u'OS:Windows-3.1.1',
        u'hostname:localhost',
        u'pool:default',
        u'priority:50',
        u'service_account:none',
        u'swarming.pool.template:no_config',
        u'tag:1',
        u'user:Jesus',
      ],
      'task_slices': [
        {
          'expiration_secs': 30,
          'properties': expected_properties,
          'wait_for_capacity': False,
        },
      ],
      'user': u'Jesus',
    }
    actual = req.to_dict()
    actual.pop('created_ts')
    actual.pop('expiration_ts')
    self.assertEqual(expected_request, actual)
    self.assertEqual(30, req.expiration_secs)
    # Intentionally hard code the hash value since it has to be deterministic.
    # Other unit tests should use the calculated value.
    self.assertEqual(
        'aa33c679b3ee30e37b9724d79a9d20bc767475c00e7f659b6191508f6b16f1ab',
        req.task_slice(0).properties_hash(req).encode('hex'))

  def test_init_new_request_isolated(self):
    parent = _gen_request(
        properties=_gen_properties(
            command=[],
            inputs_ref={
              'isolated': '0123456789012345678901234567890123456789',
              'isolatedserver': 'http://localhost:1',
              'namespace': 'default-gzip',
            }))
    # Parent entity must have a valid key id and be stored.
    parent.key = task_request.new_request_key()
    parent.put()
    # The reference is to the TaskRunResult.
    parent_id = task_pack.pack_request_key(parent.key) + u'1'
    req = _gen_request(
        properties=_gen_properties(idempotent=True, has_secret_bytes=True),
        parent_task_id=parent_id)
    # TaskRequest with secret must have a valid key.
    req.key = task_request.new_request_key()
    # Needed for the get() call below.
    req.put()
    sb = _gen_secret(req, 'I am not a banana')
    # Needed for properties_hash() call.
    sb.put()
    expected_properties = {
      'caches': [],
      'cipd_input': {
        'client_package': {
          'package_name': u'infra/tools/cipd/${platform}',
          'path': None,
          'version': u'git_revision:deadbeef',
        },
        'packages': [{
          'package_name': u'rm',
          'path': u'bin',
          'version': u'git_revision:deadbeef',
        }],
        'server': u'https://chrome-infra-packages.appspot.com'
      },
      'command': [u'command1', u'arg1'],
      'relative_cwd': None,
      'dimensions': {
        u'OS': [u'Windows-3.1.1'],
        u'hostname': [u'localhost'],
        u'pool': [u'default'],
      },
      'env': {u'foo': u'bar', u'joe': u'2'},
      'env_prefixes': {u'PATH': [u'local/path']},
      'extra_args': [],
      'execution_timeout_secs': 30,
      'grace_period_secs': 30,
      'idempotent': True,
      'inputs_ref': {
        'isolated': None,
        'isolatedserver': u'https://isolateserver.appspot.com',
        'namespace': u'default-gzip',
      },
      'io_timeout_secs': None,
      'outputs': [],
      'has_secret_bytes': True,
    }
    expected_request = {
      'authenticated': auth_testing.DEFAULT_MOCKED_IDENTITY,
      'name': u'Request name',
      'parent_task_id': unicode(parent_id),
      'priority': 50,
      'pubsub_topic': None,
      'pubsub_userdata': None,
      'service_account': u'none',
      'tags': [
        u'OS:Windows-3.1.1',
        u'hostname:localhost',
        u'pool:default',
        u'priority:50',
        u'service_account:none',
        u'swarming.pool.template:no_config',
        u'tag:1',
        u'user:Jesus',
      ],
      'task_slices': [
        {
          'expiration_secs': 30,
          'properties': expected_properties,
          'wait_for_capacity': False,
        },
      ],
      'user': u'Jesus',
    }
    actual = req.to_dict()
    # expiration_ts - created_ts == scheduling_expiration_secs.
    actual.pop('created_ts')
    actual.pop('expiration_ts')
    self.assertEqual(expected_request, actual)
    self.assertEqual(30, req.expiration_secs)
    # Intentionally hard code the hash value since it has to be deterministic.
    # Other unit tests should use the calculated value.
    self.assertEqual(
        '121c6bd6216a4cc9c4302a52da6292e5a240807ef13ace6f7f36a0c83aec6f55',
        req.task_slice(0).properties_hash(req).encode('hex'))

  def test_init_new_request_parent(self):
    parent = _gen_request()
    # Parent entity must have a valid key id and be stored.
    parent.key = task_request.new_request_key()
    parent.put()
    # The reference is to the TaskRunResult.
    parent_id = task_pack.pack_request_key(parent.key) + '1'
    child = _gen_request(parent_task_id=parent_id)
    self.assertEqual(parent_id, child.parent_task_id)

  def test_init_new_request_invalid_parent_id(self):
    # Must ends with '1' or '2', not '0'
    with self.assertRaises(ValueError):
      _gen_request(parent_task_id='1d69b9f088008810')

  def test_init_new_request_idempotent(self):
    request = _gen_request(properties=_gen_properties(idempotent=True))
    as_dict = request.to_dict()
    self.assertEqual(
        True, as_dict['task_slices'][0]['properties']['idempotent'])
    # Intentionally hard code the hash value since it has to be deterministic.
    # Other unit tests should use the calculated value.
    # Ensure the algorithm is deterministic.
    self.assertEqual(
        '58b6b8966199b901406b82ed15b23b7070cbf6ea8cba237838911939b387b4c6',
        request.task_slice(0).properties_hash(request).encode('hex'))

  def test_init_new_request_bot_service_account(self):
    request = _gen_request(service_account='bot')
    request.put()
    as_dict = request.to_dict()
    self.assertEqual('bot', as_dict['service_account'])
    self.assertIn(u'service_account:bot', as_dict['tags'])

  def test_init_new_request_RBE_CAS(self):
    request = _gen_request(
        properties=_gen_properties(
            inputs_ref=task_request.FilesRef(
                isolated='dead' * (64/4),
                isolatedserver='astuce-service',
                namespace='sha256-GCP')))
    request.put()
    as_dict = request.to_dict()
    expected = {
      'isolated':
          u'deaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead',
      'isolatedserver': u'astuce-service',
      'namespace': u'sha256-GCP',
    }
    self.assertEqual(
        expected, as_dict['task_slices'][0]['properties']['inputs_ref'])

  def _set_pool_config_with_templates(
      self, prod=None, canary=None, canary_chance=None, pool_name=u'default'):
    """Builds a new pools_config.PoolConfig populated with the given
    pools_config.TaskTemplate objects and assigns it into the mocked
    `pools_confi.get_pool_config()` method.

    If prod is None, this omits the TaskTemplateDeployment entirely.

    canary_chance may be supplied as >9999 (normally illegal) in order to force
    the selection of canary."""
    deployment = None
    if prod is not None:
      deployment = pools_config.TaskTemplateDeployment(
            prod=prod, canary=canary, canary_chance=canary_chance)

    self._pool_configs[pool_name] = pools_config.PoolConfig(
        name=pool_name,
        rev=u'testVersion1',
        scheduling_users=(),
        scheduling_groups=(),
        trusted_delegatees={},
        service_accounts=(),
        service_accounts_groups=(),
        task_template_deployment=deployment,
        bot_monitoring=None,
        default_isolate=None,
        default_cipd=None,
        external_schedulers=None,)

  def test_init_new_request_skip_template(self):
    self._set_pool_config_with_templates(_gen_task_template(env={'hi': 'prod'}))

    request = _gen_request(_template_apply=task_request.TEMPLATE_SKIP)
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:skip', as_dict['tags'])

  def test_init_new_request_missing_template(self):
    self._set_pool_config_with_templates()

    request = _gen_request()
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:none', as_dict['tags'])

  def test_init_new_request_prod_template(self):
    self._set_pool_config_with_templates(
      _gen_task_template(env={'hi': 'prod'}),
      canary=None,
      canary_chance=0,  # always prefer prod serverside
    )

    request = _gen_request()
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:prod', as_dict['tags'])
    self.assertEqual(as_dict['task_slices'][0]['properties']['env']['hi'],
                     'prod')

  def test_init_new_request_canary_template(self):
    self._set_pool_config_with_templates(
      _gen_task_template(env={'hi': 'prod'}),
      _gen_task_template(env={'hi': 'canary'}),
      canary_chance=10000,  # always prefer canary serverside
    )

    request = _gen_request()
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:canary', as_dict['tags'])
    self.assertEqual(as_dict['task_slices'][0]['properties']['env']['hi'],
                     'canary')

  def test_init_new_request_canary_never_template(self):
    self._set_pool_config_with_templates(
      _gen_task_template(env={'hi': 'prod'}),
      _gen_task_template(env={'hi': 'canary'}),
      canary_chance=10000,  # always prefer canary serverside
    )

    request = _gen_request(_template_apply=task_request.TEMPLATE_CANARY_NEVER)
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:prod', as_dict['tags'])
    self.assertEqual(as_dict['task_slices'][0]['properties']['env']['hi'],
                     'prod')

  def test_init_new_request_canary_prefer_template(self):
    self._set_pool_config_with_templates(
      _gen_task_template(env={'hi': 'prod'}),
      _gen_task_template(env={'hi': 'canary'}),
      canary_chance=0,  # always prefer prod serverside
    )

    request = _gen_request(_template_apply=task_request.TEMPLATE_CANARY_PREFER)
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:canary', as_dict['tags'])
    self.assertEqual(as_dict['task_slices'][0]['properties']['env']['hi'],
                     'canary')

  def test_init_new_request_canary_prefer_prod_template(self):
    self._set_pool_config_with_templates(
      _gen_task_template(env={'hi': 'prod'}),
      # No canary defined, even though caller would prefer it, if available.
    )

    request = _gen_request(_template_apply=task_request.TEMPLATE_CANARY_PREFER)
    as_dict = request.to_dict()
    self.assertIn(u'swarming.pool.version:testVersion1', as_dict['tags'])
    self.assertIn(u'swarming.pool.template:prod', as_dict['tags'])
    self.assertEqual(as_dict['task_slices'][0]['properties']['env']['hi'],
                     'prod')

  def test_duped(self):
    # Two TestRequest with the same properties.
    request_1 = _gen_request(properties=_gen_properties(idempotent=True))
    now = utils.utcnow()
    request_2 = _gen_request_slices(
        name='Other',
        user='Other',
        priority=201,
        created_ts=now,
        manual_tags=['tag:2'],
        task_slices=[
          task_request.TaskSlice(
              expiration_secs=129,
              properties=_gen_properties(idempotent=True)),
        ])
    self.assertEqual(
        request_1.task_slice(0).properties_hash(request_1),
        request_2.task_slice(0).properties_hash(request_2))
    self.assertTrue(request_1.task_slice(0).properties_hash(request_1))

  def test_different(self):
    # Two TestRequest with different properties.
    request_1 = _gen_request(
        properties=_gen_properties(execution_timeout_secs=30, idempotent=True))
    request_2 = _gen_request(
        properties=_gen_properties(
            execution_timeout_secs=129, idempotent=True))
    self.assertNotEqual(
        request_1.task_slice(0).properties_hash(request_1),
        request_2.task_slice(0).properties_hash(request_2))

  def test_to_proto(self):
    # Try to set as much things as possible to exercise most code paths.
    def getrandbits(i):
      self.assertEqual(i, 16)
      return 0x7766
    self.mock(random, 'getrandbits', getrandbits)
    self.mock_now(task_request._BEGINING_OF_THE_WORLD)

    # Parent entity must have a valid key id and be stored.
    # This task uses user:Jesus, which will be inherited automatically.
    parent = _gen_request()
    parent.key = task_request.new_request_key()
    parent.put()
    # The reference is to the TaskRunResult.
    parent_id = task_pack.pack_request_key(parent.key) + u'0'
    parent_run_id = task_pack.pack_request_key(parent.key) + u'1'

    request_props = _gen_properties(
        inputs_ref={
          'isolated': '0123456789012345678901234567890123456789',
          'isolatedserver': 'http://localhost:1',
          'namespace': 'default-gzip',
        },
        relative_cwd=u'subdir',
        caches=[
          task_request.CacheEntry(name=u'git_chromium', path=u'git_cache'),
        ],
        cipd_input=_gen_cipd_input(
            packages=[
              task_request.CipdPackage(
                  package_name=u'foo', path=u'tool', version=u'git:12345'),
            ],
        ),
        idempotent=True,
        outputs=[u'foo'],
        has_secret_bytes=True,
    )
    request = _gen_request_slices(
        task_slices=[
          task_request.TaskSlice(
              expiration_secs=30,
              properties=request_props,
              wait_for_capacity=True,
          ),
        ],
        # The user is ignored; the value is overridden by the parent task's
        # user.
        user=u'Joe',
        parent_task_id=parent_run_id,
        service_account=u'foo@gserviceaccount.com',
        pubsub_topic=u'projects/a/topics/abc',
        pubsub_auth_token=u'sekret',
        pubsub_userdata=u'obscure_reference',
    )
    # Necessary to have a valid task_id:
    request.key = task_request.new_request_key()
    # Necessary to attach a secret to the request:
    request.put()
    _gen_secret(request, 'I am a banana').put()

    expected_props = swarming_pb2.TaskProperties(
        cas_inputs=swarming_pb2.CASTree(
            digest=u'0123456789012345678901234567890123456789',
            server=u'http://localhost:1',
            namespace=u'default-gzip',
        ),
        cipd_inputs=[
          swarming_pb2.CIPDPackage(
              package_name=u'foo', version=u'git:12345', dest_path=u'tool'),
        ],
        named_caches=[
            swarming_pb2.NamedCacheEntry(
                name=u'git_chromium', dest_path=u'git_cache'),
        ],
        command=[u'command1', u'arg1'],
        relative_cwd=u'subdir',
        # extra_args cannot be specified with command.
        # secret_bytes cannot be retrieved, but is included in properties_hash.
        has_secret_bytes=True,
        dimensions=[
          swarming_pb2.StringListPair(key=u'OS', values=[u'Windows-3.1.1']),
          swarming_pb2.StringListPair(key=u'hostname', values=[u'localhost']),
          swarming_pb2.StringListPair(key=u'pool', values=[u'default']),
        ],
        env=[
          swarming_pb2.StringPair(key=u'foo', value=u'bar'),
          swarming_pb2.StringPair(key=u'joe', value=u'2'),
        ],
        env_paths=[
          swarming_pb2.StringListPair(key=u'PATH', values=[u'local/path']),
        ],
        execution_timeout=duration_pb2.Duration(seconds=30),
        grace_period=duration_pb2.Duration(seconds=30),
        idempotent=True,
        outputs=[u'foo'],
    )
    expected = swarming_pb2.TaskRequest(
        # Scheduling.
        task_slices=[
          swarming_pb2.TaskSlice(
              properties=expected_props,
              expiration=duration_pb2.Duration(seconds=30),
              wait_for_capacity=True,
              properties_hash=
                  '575b14dc0f59f68d54e2264f51d3c88c41f1465852f99fc9f66a8aa770c2'
                  '33f8',
          ),
        ],
        priority=50,
        service_account=u'foo@gserviceaccount.com',
        # Information.
        create_time=timestamp_pb2.Timestamp(seconds=1262304000),
        name=u'Request name',
        tags=[
          u'OS:Windows-3.1.1',
          u'hostname:localhost',
          u'pool:default',
          u'priority:50',
          u'service_account:foo@gserviceaccount.com',
          u'swarming.pool.template:no_config',
          u'tag:1',
          u'user:Jesus',
        ],
        user=u'Jesus',
        # Hierarchy.
        task_id=u'776610',
        parent_task_id=parent_id,
        parent_run_id=parent_run_id,
        # Notification. auth_token cannot be retrieved.
        pubsub_notification=swarming_pb2.PubSub(
            topic=u'projects/a/topics/abc', userdata=u'obscure_reference'),
    )

    actual = swarming_pb2.TaskRequest()
    request.to_proto(actual)
    self.assertEqual(unicode(expected), unicode(actual))

  def test_TaskRequest_to_proto_empty(self):
    # Assert that it doesn't throw on empty entity.
    actual = swarming_pb2.TaskRequest()
    task_request.TaskRequest().to_proto(actual)
    self.assertEqual(swarming_pb2.TaskRequest(), actual)

  def test_TaskSlice_to_proto_empty(self):
    # Assert that it doesn't throw on empty entity.
    request = task_request.TaskRequest()
    actual = swarming_pb2.TaskSlice()
    task_request.TaskSlice().to_proto(actual, request)
    self.assertEqual(swarming_pb2.TaskSlice(), actual)

  def test_TaskProperties_to_proto_empty(self):
    # Assert that it doesn't throw on empty entity.
    actual = swarming_pb2.TaskProperties()
    task_request.TaskProperties().to_proto(actual)
    expected = swarming_pb2.TaskProperties()
    expected.grace_period.seconds = 30
    self.assertEqual(expected, actual)

  def test_request_bad_values(self):
    with self.assertRaises(AttributeError):
      _gen_request(properties=_gen_properties(foo='bar'))

  def test_request_bad_values_stale_style(self):
    # Old TaskRequest.properties style.
    # Hack a bit the call to force the incorrect call.
    req = _gen_request_slices(
        task_slices=[],
        expiration_ts=utils.utcnow() + datetime.timedelta(hours=1),
        properties_old=_gen_properties())
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()

  def test_request_bad_values_task_slices(self):
    with self.assertRaises(ValueError):
      # No TaskSlice
      _gen_request_slices(task_slices=[])
    def _gen_slice(**props):
      return task_request.TaskSlice(
          expiration_secs=60, properties=_gen_properties(**props))

    slices = [_gen_slice(dimensions={u'pool': [u'GPU']})]
    _gen_request_slices(task_slices=slices).put()

    # Limit on the maximum number of TaskSlice in a TaskRequest.
    slices = [
      _gen_slice(dimensions={u'pool': [u'GPU'], u'v': [unicode(i)]})
      for i in xrange(8)
    ]
    _gen_request_slices(task_slices=slices).put()
    slices = [
      _gen_slice(dimensions={u'pool': [u'GPU'], u'v': [unicode(i)]})
      for i in xrange(9)
    ]
    req = _gen_request_slices(task_slices=slices)
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    # Different pools.
    slices = [
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(dimensions={u'pool': [u'GPU']})),
      task_request.TaskSlice(
          expiration_secs=60,
          properties=_gen_properties(dimensions={u'pool': [u'other']})),
    ]
    req = _gen_request_slices(task_slices=slices)
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()

  def test_request_bad_command(self):
    req = _gen_request(
        properties=_gen_properties(command=[], inputs_ref=None))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(properties=_gen_properties(command={'a': 'b'}))
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(properties=_gen_properties(command='python'))
    _gen_request(properties=_gen_properties(command=['python'])).put()
    _gen_request(properties=_gen_properties(command=[u'python'])).put()
    _gen_request(properties=_gen_properties(command=[u'python']*128)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(properties=_gen_properties(command=[u'python']*129)).put()

  def test_request_bad_extra_args(self):
    _gen_request(
        properties=_gen_properties(
            command=[],
            extra_args=[u'python']*128,
            inputs_ref=task_request.FilesRef(
                isolated='deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
                isolatedserver='http://localhost:1',
                namespace='default-gzip'))).put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(
              command=[],
              extra_args=[u'python']*129,
              inputs_ref=task_request.FilesRef(
                  isolated='deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
                  isolatedserver='http://localhost:1',
                  namespace='default-gzip'))).put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(
              command=[u'python'],
              extra_args=[u'python'],
              inputs_ref=task_request.FilesRef(
                  isolated='deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
                  isolatedserver='http://localhost:1',
                  namespace='default-gzip'))).put()

  def test_request_bad_cipd_input(self):
    def mkcipdreq(idempotent=False, **cipd_input):
      return _gen_request(
          properties=_gen_properties(
              idempotent=idempotent,
              cipd_input=_gen_cipd_input(**cipd_input)))

    req = mkcipdreq(packages=[{}])
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(
          packages=[
            task_request.CipdPackage(
                package_name='infra|rm', path='.', version='latest'),
          ])
    req = mkcipdreq(
        packages=[task_request.CipdPackage(package_name='rm', path='.')])
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    req = mkcipdreq(
        packages=[
          task_request.CipdPackage(package_name='rm', version='latest'),
        ])
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(
          packages=[
            task_request.CipdPackage(
                package_name='rm', path='/', version='latest'),
          ])
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(
          packages=[
            task_request.CipdPackage(
                package_name='rm', path='/a', version='latest'),
          ])
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(
          packages=[
            task_request.CipdPackage(
                package_name='rm', path='a/..', version='latest'),
          ])
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(
          packages=[
            task_request.CipdPackage(
                package_name='rm', path='a/./b', version='latest'),
          ])
    req = mkcipdreq(
        packages=[
          task_request.CipdPackage(
              package_name='rm', path='.', version='latest'),
          task_request.CipdPackage(
              package_name='rm', path='.', version='canary'),
        ])
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    req = mkcipdreq(
        idempotent=True,
        packages=[
          task_request.CipdPackage(
              package_name='rm', path='.', version='latest'),
        ])
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(server='abc')
    with self.assertRaises(datastore_errors.BadValueError):
      mkcipdreq(
          client_package=task_request.CipdPackage(
              package_name='--bad package--'))
    mkcipdreq().put()
    mkcipdreq(
        packages=[
          task_request.CipdPackage(
              package_name='rm', path='.', version='latest'),
        ]).put()
    mkcipdreq(
        client_package=task_request.CipdPackage(
            package_name='infra/tools/cipd/${platform}',
            version='git_revision:daedbeef'),
        packages=[
          task_request.CipdPackage(
              package_name='rm', path='.', version='latest'),
          ],
        server='https://chrome-infra-packages.appspot.com').put()

  def test_request_bad_named_cache(self):
    mkcachereq = lambda *c: _gen_request(
        properties=_gen_properties(caches=c)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(task_request.CacheEntry(name='', path='git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(task_request.CacheEntry(name='git_chromium', path=''))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='git_cache'),
          task_request.CacheEntry(name='git_v8', path='git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='git_cache'),
          task_request.CacheEntry(name='git_chromium', path='git_cache2'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='/git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='../git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(
              name='git_chromium', path='git_cache/../../a'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='../git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='git_cache//a'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(
          task_request.CacheEntry(name='git_chromium', path='a/./git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(task_request.CacheEntry(name='has space', path='git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(task_request.CacheEntry(name='CAPITAL', path='git_cache'))
    mkcachereq()
    mkcachereq(task_request.CacheEntry(name='git_chromium', path='git_cache'))
    mkcachereq(
        task_request.CacheEntry(name='git_chromium', path='git_cache'),
        task_request.CacheEntry(name='build_chromium', path='out'))
    mkcachereq(task_request.CacheEntry(name=u'g'*128, path=u'git_cache'))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(task_request.CacheEntry(name=u'g'*129, path=u'git_cache'))
    mkcachereq(task_request.CacheEntry(name=u'g', path=u'p'*256))
    with self.assertRaises(datastore_errors.BadValueError):
      mkcachereq(task_request.CacheEntry(name=u'g', path=u'p'*257))
    # Too many.
    c = [
      task_request.CacheEntry(name=unicode(i), path=unicode(i))
      for i in xrange(32)
    ]
    _gen_request(properties=_gen_properties(caches=c)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      c = [
        task_request.CacheEntry(name=unicode(i), path=unicode(i))
        for i in xrange(33)
      ]
      _gen_request(properties=_gen_properties(caches=c)).put()

  def test_request_bad_named_cache_and_cipd_input(self):
    # A CIPD package and named caches cannot be mapped to the same path.
    req = _gen_request(
        properties=_gen_properties(
            caches=[
              task_request.CacheEntry(name='git_chromium', path='git_cache'),
            ],
            cipd_input=_gen_cipd_input(
                packages=[
                  task_request.CipdPackage(
                      package_name='foo', path='git_cache', version='latest'),
                ])))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    req = _gen_request(
        properties=_gen_properties(
            caches=[
              task_request.CacheEntry(name='git_chromium', path='git_cache1'),
            ],
            cipd_input=_gen_cipd_input(
                packages=[
                  task_request.CipdPackage(
                      package_name='foo', path='git_cache2', version='latest'),
                ]))).put()

  def test_request_bad_dimensions(self):
    # Type error.
    with self.assertRaises(TypeError):
      _gen_request(properties=_gen_properties(dimensions=[]))
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(properties=_gen_properties(dimensions={}))
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(dimensions={u'id': u'b', u'a:': u'b'}))
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(dimensions={u'id': u'b', u'a.': u'b'}))
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(dimensions={u'id': u'b', u'a': [u'b']}))
    # >1 value for id.
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(dimensions={u'id': [u'a', u'b']}))
    # >1 value for pool.
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(dimensions={u'pool': [u'b', u'b']}))
    _gen_request(
        properties=_gen_properties(
            dimensions={u'id': [u'b'], u'pool': [u'b']})).put()
    _gen_request(
        properties=_gen_properties(
            dimensions={u'id': [u'b'], u'pool': [u'b'], u'a.': [u'c']})).put()
    _gen_request(
        properties=_gen_properties(
            dimensions={u'pool': [u'b'], u'a.': [u'b', u'c']})).put()

  def test_request_bad_dimensions_key(self):
    # Max # keys.
    d = {u'a%s' % string.ascii_letters[i]: [unicode(i)] for i in xrange(31)}
    d[u'pool'] = [u'a']
    _gen_request(properties=_gen_properties(dimensions=d)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      d = {u'a%s' % string.ascii_letters[i]: [unicode(i)] for i in xrange(32)}
      d[u'pool'] = [u'a']
      _gen_request(properties=_gen_properties(dimensions=d)).put()

    with self.assertRaises(datastore_errors.BadValueError):
      # Key regexp.
      d = {u'pool': [u'default'], u'1': [u'value']}
      _gen_request(properties=_gen_properties(dimensions=d)).put()
    # Key length.
    d = {
      u'pool': [u'default'],
      u'v'*config.DIMENSION_KEY_LENGTH: [u'v'],
    }
    _gen_request(properties=_gen_properties(dimensions=d)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      d = {
        u'pool': [u'default'],
        u'v'*(config.DIMENSION_KEY_LENGTH+1): [u'value'],
      }
      _gen_request(properties=_gen_properties(dimensions=d)).put()

  def test_request_bad_dimensions_value(self):
    # Max # values.
    d = {u'pool': [u'b'], u'a.': [unicode(i) for i in xrange(16)]}
    _gen_request(properties=_gen_properties(dimensions=d)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      d = {u'pool': [u'b'], u'a.': [unicode(i) for i in xrange(17)]}
      _gen_request(properties=_gen_properties(dimensions=d)).put()
    # Value length.
    d = {
      u'pool': [u'default'],
      u'v': [u'v'*config.DIMENSION_VALUE_LENGTH],
    }
    _gen_request(properties=_gen_properties(dimensions=d)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      d = {
        u'pool': [u'default'],
        u'v': [u'v'*(config.DIMENSION_VALUE_LENGTH+1)],
      }
      _gen_request(properties=_gen_properties(dimensions=d)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      # Value with space.
      d = {u'pool': [u'default'], u'v': [u'v ']}
      _gen_request(properties=_gen_properties(dimensions=d)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      # Duplicate value.
      d = {u'pool': [u'default'], u'v': [u'v', u'v']}
      _gen_request(properties=_gen_properties(dimensions=d)).put()

  def test_request_bad_env(self):
    # Type error.
    with self.assertRaises(TypeError):
      _gen_request(properties=_gen_properties(env=[]))
    with self.assertRaises(TypeError):
      _gen_request(properties=_gen_properties(env={u'a': 1}))
    _gen_request(properties=_gen_properties(env={})).put()
    e = {u'k': u'v'}
    _gen_request(properties=_gen_properties(env=e)).put()
    # Key length.
    e = {u'k'*64: u'v'}
    _gen_request(properties=_gen_properties(env=e)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      e = {u'k'*65: u'v'}
      _gen_request(properties=_gen_properties(env=e)).put()
    # # keys.
    e = {u'k%s' % i: u'v' for i in xrange(64)}
    _gen_request(properties=_gen_properties(env=e)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      e = {u'k%s' % i: u'v' for i in xrange(65)}
      _gen_request(properties=_gen_properties(env=e)).put()
    # Value length.
    e = {u'k': u'v'*1024}
    _gen_request(properties=_gen_properties(env=e)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      e = {u'k': u'v'*1025}
      _gen_request(properties=_gen_properties(env=e)).put()

  def test_request_bad_env_prefixes(self):
    # Type error.
    with self.assertRaises(TypeError):
      _gen_request(properties=_gen_properties(env_prefixes=[]))
    with self.assertRaises(TypeError):
      _gen_request(properties=_gen_properties(env_prefixes={u'a': 1}))
    _gen_request(properties=_gen_properties(env_prefixes={})).put()
    e = {u'k': [u'v']}
    _gen_request(properties=_gen_properties(env_prefixes=e)).put()
    # Key length.
    e = {u'k'*64: [u'v']}
    _gen_request(properties=_gen_properties(env_prefixes=e)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      e = {u'k'*65: [u'v']}
      _gen_request(properties=_gen_properties(env_prefixes=e)).put()
    # # keys.
    e = {u'k%s' % i: [u'v'] for i in xrange(64)}
    _gen_request(properties=_gen_properties(env_prefixes=e)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      e = {u'k%s' % i: [u'v'] for i in xrange(65)}
      _gen_request(properties=_gen_properties(env_prefixes=e)).put()
    # Value length.
    e = {u'k': [u'v'*1024]}
    _gen_request(properties=_gen_properties(env_prefixes=e)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      e = {u'k': [u'v'*1025]}
      _gen_request(properties=_gen_properties(env_prefixes=e)).put()

  def test_request_bad_priority(self):
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(priority=task_request.MAXIMUM_PRIORITY+1)
    _gen_request(priority=task_request.MAXIMUM_PRIORITY).put()

  def test_request_bad_execution_timeout(self):
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(
          properties=_gen_properties(
              execution_timeout_secs=task_request.MAX_TIMEOUT_SECS+1))
    _gen_request(
        properties=_gen_properties(
            execution_timeout_secs=task_request.MAX_TIMEOUT_SECS)).put()

  def test_request_bad_expiration(self):
    now = utils.utcnow()
    with self.assertRaises(ValueError):
      _gen_request_slices(
          created_ts=now,
          task_slices=[
            task_request.TaskSlice(
                expiration_secs=None,
                properties=_gen_properties()),
          ])
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request_slices(
          created_ts=now,
          task_slices=[
            task_request.TaskSlice(
                expiration_secs=task_request._MIN_TIMEOUT_SECS-1,
                properties=_gen_properties()),
          ])
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request_slices(
          created_ts=now,
          task_slices=[
            task_request.TaskSlice(
                expiration_secs=task_request.MAX_EXPIRATION_SECS+1,
                properties=_gen_properties()),
          ])
    _gen_request_slices(
        created_ts=now,
        task_slices=[
          task_request.TaskSlice(
              expiration_secs=task_request._MIN_TIMEOUT_SECS,
              properties=_gen_properties()),
        ]).put()
    _gen_request_slices(
        created_ts=now,
        task_slices=[
          task_request.TaskSlice(
              expiration_secs=task_request.MAX_EXPIRATION_SECS,
              properties=_gen_properties()),
        ]).put()

  def test_request_bad_inputs_ref(self):
    # Both command and inputs_ref.isolated.
    _gen_request(
        properties=_gen_properties(
            command=['python'],
            inputs_ref=task_request.FilesRef(
                isolated='deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
                isolatedserver='http://localhost:1',
                namespace='default-gzip'))).put()
    # Bad digest.
    req = _gen_request(properties=_gen_properties(
        command=['see', 'spot', 'run'],
        inputs_ref=task_request.FilesRef(
            isolated='deadbeef',
            isolatedserver='http://localhost:1',
            namespace='default-gzip')))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    # inputs_ref without server/namespace.
    req = _gen_request(
        properties=_gen_properties(inputs_ref=task_request.FilesRef()))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    # Without digest nor command.
    req = _gen_request(properties=_gen_properties(
        command=[],
        inputs_ref=task_request.FilesRef(
            isolatedserver='https://isolateserver.appspot.com',
            namespace='default-gzip^^^')))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    # For 'sha256-GCP', the length must be 64.
    req = _gen_request(properties=_gen_properties(
        command=[],
        inputs_ref=task_request.FilesRef(
            isolated='deadbeefdeadbeefdeadbeefdeadbeefdeadbeef',
            isolatedserver='foo-bar',
            namespace='sha256-GCP')))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()
    # For 'sha256-GCP', the isolatedserver value must not contain '://'.
    req = _gen_request(properties=_gen_properties(
        command=[],
        inputs_ref=task_request.FilesRef(
            isolated='dead' * (64/4),
            isolatedserver='foo://bar',
            namespace='sha256-GCP')))
    with self.assertRaises(datastore_errors.BadValueError):
      req.put()

  def test_request_bad_pubsub(self):
    _gen_request(pubsub_topic=u'projects/a/topics/abc').put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(pubsub_topic=u'a')
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(pubsub_topic=u'projects/a/topics/ab').put()
    _gen_request(pubsub_topic=u'projects/' + u'a'*1004 + u'/topics/abc').put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(pubsub_topic=u'projects/' + u'a'*1005 + u'/topics/abc').put()

  def test_request_bad_service_account(self):
    _gen_request(service_account=u'none').put()
    _gen_request(service_account=u'bot').put()
    _gen_request(service_account=u'joe@localhost').put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(service_account=u'joe').put()
    _gen_request(service_account=u'joe@'+u'l'*124).put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(service_account=u'joe@'+u'l'*125).put()

  def test_request_bad_tags(self):
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(manual_tags=['a']).put()

  def test_request_bad_tags_too_many(self):
    _gen_request(manual_tags=['a:b']*256).put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(manual_tags=['a:b']*257).put()

  def test_request_bad_tags_too_long(self):
    # Minus 2 for the 'a:' prefix.
    l = task_request._TAG_LENGTH - 2
    _gen_request(manual_tags=['a:' + 'b'*l]).put()
    with self.assertRaises(datastore_errors.BadValueError):
      _gen_request(manual_tags=['a:' + 'a'*(l+1)]).put()

  def test_validate_priority(self):
    with self.assertRaises(TypeError):
      task_request.validate_priority('1')
    with self.assertRaises(datastore_errors.BadValueError):
      task_request.validate_priority(-1)
    with self.assertRaises(datastore_errors.BadValueError):
      task_request.validate_priority(task_request.MAXIMUM_PRIORITY+1)
    task_request.validate_priority(0)
    task_request.validate_priority(1)
    task_request.validate_priority(task_request.MAXIMUM_PRIORITY)

  def test_datetime_to_request_base_id(self):
    now = datetime.datetime(2012, 1, 2, 3, 4, 5, 123456)
    self.assertEqual(
        0xeb5313d0300000, task_request.datetime_to_request_base_id(now))

  def test_convert_to_request_key(self):
    """Indirectly tested by API."""
    now = datetime.datetime(2012, 1, 2, 3, 4, 5, 123456)
    key = task_request.convert_to_request_key(now)
    self.assertEqual(9157134072765480958, key.id())

  def test_request_key_to_datetime(self):
    key = ndb.Key(task_request.TaskRequest, 0x7f14acec2fcfffff)
    # Resolution is only kept at millisecond level compared to
    # datetime_to_request_base_id() by design.
    self.assertEqual(
        datetime.datetime(2012, 1, 2, 3, 4, 5, 123000),
        task_request.request_key_to_datetime(key))

  def test_request_id_to_key(self):
    # Simple XOR.
    self.assertEqual(
        ndb.Key(task_request.TaskRequest, 0x7f14acec2fcfffff),
        task_request.request_id_to_key(0xeb5313d0300000))

  def test_secret_bytes(self):
    task_request.SecretBytes(secret_bytes='a'*(20*1024)).put()
    with self.assertRaises(datastore_errors.BadValueError):
      task_request.SecretBytes(secret_bytes='a'*(20*1024+1)).put()

  def test_cron_delete_old_task_requests(self):
    # Creating 1000 tasks would make this test significantly slower.
    self.mock(task_request, '_TASKS_DELETE_CHUNK_SIZE', 5)

    now = utils.utcnow()
    task_ids = []
    for i in xrange(14):
      self.mock_now(now, i)
      request = _gen_request_slices()
      request.key = task_request.new_request_key()
      request.put()
      task_ids.append(task_pack.pack_request_key(request.key))

    # Use 11 seconds offset, so that entities 12, 13 are not deleted. Yet create
    # 3 GAE tasks to delete the chunks limited at 5 items.
    self.mock_now(now + task_request._OLD_TASK_REQUEST_CUT_OFF, 11)
    self.assertEqual(12, task_request.cron_delete_old_task_requests())
    expected = [
      (
        ('/internal/taskqueue/cleanup/tasks/delete', 'delete-tasks'),
        {'payload': utils.encode_to_json({u'task_ids': task_ids[0:5]})},
      ),
      (
        ('/internal/taskqueue/cleanup/tasks/delete', 'delete-tasks'),
        {'payload': utils.encode_to_json({u'task_ids': task_ids[5:10]})},
      ),
      (
        ('/internal/taskqueue/cleanup/tasks/delete', 'delete-tasks'),
        {'payload': utils.encode_to_json({u'task_ids': task_ids[10:12]})},
      ),
    ]
    # task_ids[12:14] are not touched.
    self.assertEqual(expected, self._enqueue_calls)
    self._enqueue_calls = []

  def test_task_delete_tasks(self):
    # The data here should be the same as what is passed to the task queue in
    # test_cron_delete_old_task_requests.
    class Foo(ndb.Model):
      pass
    task_ids = []
    for _ in xrange(5):
      request = _gen_request_slices()
      request.key = task_request.new_request_key()
      request.put()
      # Create a dummy child entity to ensure it's deleted too.
      Foo(parent=request.key, id=1).put()
      task_ids.append(task_pack.pack_request_key(request.key))

    self.assertEqual(5, task_request.task_delete_tasks(task_ids))
    self.assertEqual(0, task_request.TaskRequest.query().count())
    self.assertEqual(0, Foo.query().count())

  def test_task_bq_empty(self):
    # Empty, nothing is done.
    start = utils.utcnow()
    end = start+datetime.timedelta(seconds=60)
    self.assertEqual((0, 0), task_request.task_bq(start, end))

  def test_task_bq(self):
    def getrandbits(i):
      self.assertEqual(i, 16)
      return 0x7766
    self.mock(random, 'getrandbits', getrandbits)
    payloads = []
    def send_to_bq(table_name, rows):
      self.assertEqual('task_requests', table_name)
      payloads.append(rows)
      return 0
    self.mock(bq_state, 'send_to_bq', send_to_bq)

    # Generate two tasks requests.
    now = datetime.datetime(2014, 1, 2, 3, 4, 5, 6)
    start = self.mock_now(now, 10)
    request_1 = _gen_request()
    request_1.key = task_request.new_request_key()
    request_1.put()
    self.mock_now(now, 20)
    request_2 = _gen_request()
    request_2.key = task_request.new_request_key()
    request_2.put()
    end = self.mock_now(now, 30)

    self.assertEqual((2, 0), task_request.task_bq(start, end))
    self.assertEqual(1, len(payloads), payloads)
    actual_rows = payloads[0]
    self.assertEqual(2, len(actual_rows))
    expected = [
      request_1.task_id,
      request_2.task_id,
    ]
    self.assertEqual(expected, [r[0] for r in actual_rows])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
