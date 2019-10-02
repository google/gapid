#!/usr/bin/env python
# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import random
import sys
import unittest

import test_env
test_env.setup_test_env()

from components import auth
from components import config
from components import utils
from components.config import validation
from test_support import test_case

from proto.config import pools_pb2
from server import pools_config
from server import task_request

from google.protobuf import text_format


TEST_CONFIG = pools_pb2.PoolsCfg(
    pool=[
      pools_pb2.Pool(
        name=['pool_name', 'another_name'],
        schedulers=pools_pb2.Schedulers(
          user=['user:a@example.com', 'b@example.com'],
          group=['group1', 'group2'],
          trusted_delegation=[
            pools_pb2.TrustedDelegation(
              peer_id='delegatee@example.com',
              require_any_of=pools_pb2.TrustedDelegation.TagList(
                tag=['k:tag1', 'k:tag2'],
              ),
            ),
          ],
        ),
        allowed_service_account=[
          'a1@example.com',
          'a2@example.com',
        ],
        allowed_service_account_group=[
          'accounts_group1',
          'accounts_group2',
        ],
        bot_monitoring='bots',
        external_schedulers=[pools_pb2.ExternalSchedulerConfig(
          address='externalscheduler.google.com',
          id='ext1',
          dimensions=['key1:value1', 'key2:value2'],
          enabled=True,
          fallback_when_empty=True,
        )],
      ),
    ],
    default_external_services=pools_pb2.ExternalServices(
      isolate=pools_pb2.ExternalServices.Isolate(
        server='https://isolate.server.example.com',
        namespace='default-gzip',
      ),
      cipd=pools_pb2.ExternalServices.CIPD(
        server='https://cipd.server.example.com',
        client_version='latest',
      )
    ),
    bot_monitoring=[
      pools_pb2.BotMonitoring(name='bots', dimension_key=['os', 'bool']),
    ],
)


class PoolsConfigTest(test_case.TestCase):
  def validator_test(self, cfg, messages):
    ctx = validation.Context()
    pools_config._validate_pools_cfg(cfg, ctx)
    self.assertEquals(ctx.result().messages, [
      validation.Message(severity=logging.ERROR, text=m)
      for m in messages
    ])

  def mock_config(self, cfg):
    def get_self_config_mock(path, cls=None, **kwargs):
      self.assertEqual({'store_last_good': True}, kwargs)
      self.assertEqual('pools.cfg', path)
      self.assertEqual(cls, pools_pb2.PoolsCfg)
      return 'rev', cfg
    self.mock(config, 'get_self_config', get_self_config_mock)
    utils.clear_cache(pools_config._fetch_pools_config)

  def test_get_pool_config(self):
    self.mock_config(TEST_CONFIG)
    self.assertEqual(None, pools_config.get_pool_config('unknown'))

    expected1 = pools_config.PoolConfig(
        name=u'pool_name',
        rev='rev',
        scheduling_users=frozenset([
          auth.Identity('user', 'b@example.com'),
          auth.Identity('user', 'a@example.com'),
        ]),
        scheduling_groups=frozenset([u'group2', u'group1']),
        trusted_delegatees={
          auth.Identity('user', 'delegatee@example.com'):
            pools_config.TrustedDelegatee(
              peer_id=auth.Identity('user', 'delegatee@example.com'),
              required_delegation_tags=frozenset([u'k:tag1', u'k:tag2']),
            ),
        },
        service_accounts=frozenset([u'a2@example.com', u'a1@example.com']),
        service_accounts_groups=(u'accounts_group1', u'accounts_group2'),
        task_template_deployment=None,
        bot_monitoring=None,
        default_isolate=pools_config.IsolateServer(
          server='https://isolate.server.example.com',
          namespace='default-gzip',
        ),
        default_cipd=pools_config.CipdServer(
          server='https://cipd.server.example.com',
          client_version='latest',
        ),
        external_schedulers=(
          pools_config.ExternalSchedulerConfig(
            address=u'externalscheduler.google.com',
            id=u'ext1',
            dimensions=frozenset(['key2:value2', 'key1:value1']),
            all_dimensions=frozenset(),
            any_dimensions=frozenset(),
            enabled=True,
            fallback_when_empty=True,
          ),
        ),
    )
    expected2 = expected1._replace(name='another_name')

    self.assertEqual(expected1, pools_config.get_pool_config('pool_name'))
    self.assertEqual(expected2, pools_config.get_pool_config('another_name'))
    self.assertEqual(['another_name', 'pool_name'], pools_config.known())


  def test_validate_external_services_isolate(self):
    def msg(**kwargs):
      return pools_pb2.PoolsCfg(
        default_external_services=pools_pb2.ExternalServices(
          isolate=pools_pb2.ExternalServices.Isolate(**kwargs),
          cipd=pools_pb2.ExternalServices.CIPD(
            server="https://example.com",
            client_version="test",
          ),
        ))

    self.validator_test(
        msg(server='https://isolateserver.appspot.com'),
        ['isolate namespace is not set'])

    self.validator_test(
        msg(
            server='isolateserver.appspot.com',
            namespace='abc',
        ),
        [
          'isolate server must start with "https://" or "http://localhost"',
        ])

    self.validator_test(
        msg(
          server='https://isolateserver.appspot.com',
          namespace='bad namespace',
        ),
        [
          'isolate namespace is invalid "bad namespace"'
        ])

    self.validator_test(
        msg(
          server='https://isolateserver.appspot.com',
          namespace='abc',
        ),
        [])

    self.validator_test(
        msg(),
        [
          'isolate server is not set',
          'isolate namespace is not set',
        ])

  def test_validate_external_services_cipd(self):
    def msg(**kwargs):
      return pools_pb2.PoolsCfg(
        default_external_services=pools_pb2.ExternalServices(
          isolate=pools_pb2.ExternalServices.Isolate(
            server="https://example.com",
            namespace="test",
          ),
          cipd=pools_pb2.ExternalServices.CIPD(**kwargs),
        ))

    self.validator_test(
        msg(),
        [
          'cipd server is not set',
          'cipd client_version is invalid ""',
        ])

    self.validator_test(
        msg(
            server='chrome-infra-packages.appspot.com',
            client_version='git_revision:deadbeef',
        ),
        [
          'cipd server must start with "https://" or "http://localhost"',
        ])

    self.validator_test(
        msg(
            server='https://chrome-infra-packages.appspot.com',
            client_version='git_revision:deadbeef',
        ),
        [])


  def test_empty_config_is_valid(self):
    self.validator_test(pools_pb2.PoolsCfg(), [])

  def test_good_config_is_valid(self):
    self.validator_test(TEST_CONFIG, [])

  def test_missing_pool_name(self):
    cfg = pools_pb2.PoolsCfg(pool=[pools_pb2.Pool()])
    self.validator_test(cfg, [
      'pool #0 (unnamed): at least one pool name must be given',
    ])

  def test_bad_pool_name(self):
    n = 'x'*300
    cfg = pools_pb2.PoolsCfg(pool=[pools_pb2.Pool(name=[n])])
    self.validator_test(cfg, [
      'pool #0 (%s): bad pool name "%s", not a valid dimension value' % (n, n),
    ])

  def test_duplicate_pool_name(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc']),
      pools_pb2.Pool(name=['abc']),
    ])
    self.validator_test(cfg, [
      'pool #1 (abc): pool "abc" was already declared',
    ])

  def test_bad_scheduling_user(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc'], schedulers=pools_pb2.Schedulers(
        user=['not valid email'],
      )),
    ])
    self.validator_test(cfg, [
      'pool #0 (abc): bad user value "not valid email" - '
      'Identity has invalid format: not valid email',
    ])

  def test_bad_scheduling_group(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc'], schedulers=pools_pb2.Schedulers(
        group=['!!!'],
      )),
    ])
    self.validator_test(cfg, [
      'pool #0 (abc): bad group name "!!!"',
    ])

  def test_no_delegatee_peer_id(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc'], schedulers=pools_pb2.Schedulers(
        trusted_delegation=[pools_pb2.TrustedDelegation()],
      )),
    ])
    self.validator_test(cfg, [
      'pool #0 (abc): trusted_delegation #0 (): "peer_id" is required',
    ])

  def test_bad_delegatee_peer_id(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc'], schedulers=pools_pb2.Schedulers(
        trusted_delegation=[pools_pb2.TrustedDelegation(
          peer_id='not valid email',
        )],
      )),
    ])
    self.validator_test(cfg, [
      'pool #0 (abc): trusted_delegation #0 (not valid email): bad peer_id '
      'value "not valid email" - Identity has invalid format: not valid email',
    ])

  def test_duplicate_delegatee_peer_id(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc'], schedulers=pools_pb2.Schedulers(
        trusted_delegation=[
          pools_pb2.TrustedDelegation(peer_id='a@example.com'),
          pools_pb2.TrustedDelegation(peer_id='a@example.com'),
        ],
      )),
    ])
    self.validator_test(cfg, [
      'pool #0 (abc): trusted_delegation #0 (a@example.com): peer '
      '"a@example.com" was specified twice',
    ])

  def test_bad_delegation_tag(self):
    cfg = pools_pb2.PoolsCfg(pool=[
      pools_pb2.Pool(name=['abc'], schedulers=pools_pb2.Schedulers(
        trusted_delegation=[pools_pb2.TrustedDelegation(
          peer_id='a@example.com',
          require_any_of=pools_pb2.TrustedDelegation.TagList(
            tag=['not kv'],
          ),
        )],
      )),
    ])
    self.validator_test(cfg, [
      'pool #0 (abc): trusted_delegation #0 (a@example.com): bad tag #0 '
      '"not kv" - must be <key>:<value>',
    ])

  def test_bad_service_account(self):
    cfg = pools_pb2.PoolsCfg(pool=[pools_pb2.Pool(
      name=['abc'],
      allowed_service_account=['not an email'],
    )])
    self.validator_test(cfg, [
      'pool #0 (abc): bad allowed_service_account #0 "not an email"',
    ])

  def test_bad_service_account_group(self):
    cfg = pools_pb2.PoolsCfg(pool=[pools_pb2.Pool(
      name=['abc'],
      allowed_service_account_group=['!!!'],
    )])
    self.validator_test(cfg, [
      'pool #0 (abc): bad allowed_service_account_group #0 "!!!"',
    ])

  def test_missing_bot_monitoring(self):
    cfg = pools_pb2.PoolsCfg(pool=[pools_pb2.Pool(
      name=['abc'],
      bot_monitoring='missing',
    )])
    self.validator_test(cfg, [
      'pool #0 (abc): refer to missing bot_monitoring u\'missing\'',
    ])

  def test_good_bot_monitoring(self):
    cfg = pools_pb2.PoolsCfg(
        pool=[pools_pb2.Pool(name=['abc'], bot_monitoring='mon')],
        bot_monitoring=[
          pools_pb2.BotMonitoring(name='mon', dimension_key='a'),
        ])
    self.validator_test(cfg, [])

  def test_unreferenced_bot_monitoring(self):
    cfg = pools_pb2.PoolsCfg(
        pool=[pools_pb2.Pool(name=['abc'])],
        bot_monitoring=[
          pools_pb2.BotMonitoring(name='mon', dimension_key='a'),
        ])
    self.validator_test(cfg, [
      'bot_monitoring not referred to: mon',
    ])


class TaskTemplateBaseTest(unittest.TestCase):
  def setUp(self):
    super(TaskTemplateBaseTest, self).setUp()
    self._canary_dice_roll = 5000  # 50%
    self._randint_normal = random.randint
    random.randint = lambda *_: self._canary_dice_roll
    self.ctx = validation.Context()

  def tearDown(self):
    super(TaskTemplateBaseTest, self).tearDown()
    random.randint = self._randint_normal

  PTT = pools_pb2.TaskTemplate
  PCE = pools_pb2.TaskTemplate.CacheEntry
  PCP = pools_pb2.TaskTemplate.CipdPackage
  PE = pools_pb2.TaskTemplate.Env

  def tt(self, **kwargs):
    """Builds a pools_config.TaskTemplate.

    Intercepts 'inclusions' from kwargs, and passes the rest to
    `pools_pb2.TaskTemplate`. Then builds a `pools_config.TaskTemplate` the same
    way that pools_config.py does, adds in the inclusions, then returns its
    finalized form.

    Errors during the construction of the TaskTemplate are accumulated in
    self.ctx.
    """
    inclusions = kwargs.pop('inclusions', ())
    ret = pools_config.TaskTemplate._Intermediate(self.ctx, self.PTT(**kwargs))
    ret.inclusions.update(inclusions)
    return ret.finalize(self.ctx)


class TestTaskTemplates(TaskTemplateBaseTest):
  @staticmethod
  def parse(textpb):
    return text_format.Merge(textpb, pools_pb2.TaskTemplate())

  def test_task_template_update_cache(self):
    tti = pools_config.TaskTemplate._Intermediate(
        self.ctx, pools_pb2.TaskTemplate())
    tti.update(
        self.ctx, self.tt(cache=[self.PCE(name='hi', path='there')]), None)

    self.assertEqual(
        self.tt(cache=[self.PCE(name='hi', path='there')]),
        tti.finalize(self.ctx))

    # override existing
    tti.update(
        self.ctx, self.tt(cache=[self.PCE(name='hi', path='nerd')]), None)

    self.assertEqual(
        self.tt(cache=[self.PCE(name='hi', path='nerd')]),
        tti.finalize(self.ctx))

    # add new
    tti.update(
        self.ctx, self.tt(cache=[self.PCE(name='other', path='yep')]), None)

    self.assertEqual(
        self.tt(cache=[
          self.PCE(name='hi', path='nerd'),
          self.PCE(name='other', path='yep'),
        ]),
        tti.finalize(self.ctx))

  def test_task_template_update_cipd_package(self):
    tti = pools_config.TaskTemplate._Intermediate(
        self.ctx, pools_pb2.TaskTemplate())
    tti.update(self.ctx, self.tt(
        cipd_package=[self.PCP(path='path', pkg='some/pkg', version='latest')]
    ), None)

    self.assertEqual(
      self.tt(cipd_package=[
          self.PCP(path='path', pkg='some/pkg', version='latest')]),
      tti.finalize(self.ctx),
    )

    # override existing
    tti.update(self.ctx, self.tt(cipd_package=[
        self.PCP(path='path', pkg='some/pkg', version='oldest')]), None)

    self.assertEqual(
      self.tt(cipd_package=[
          self.PCP(path='path', pkg='some/pkg', version='oldest')]),
      tti.finalize(self.ctx),
    )

    # add new
    tti.update(self.ctx, self.tt(cipd_package=[
        self.PCP(path='other_path', pkg='some/pkg', version='1'),
    ]), None)

    self.assertEqual(
      self.tt(cipd_package=[
        self.PCP(path='other_path', pkg='some/pkg', version='1'),
        self.PCP(path='path', pkg='some/pkg', version='oldest'),
      ]),
      tti.finalize(self.ctx),
    )

  def test_task_template_update_env(self):
    tti = pools_config.TaskTemplate._Intermediate(
        self.ctx, pools_pb2.TaskTemplate())
    tti.update(self.ctx, self.tt(
        env=[self.PE(var='VAR', value='1', soft=True)]), None)

    self.assertEqual(
        self.tt(env=[self.PE(var='VAR', value='1', soft=True)]),
        tti.finalize(self.ctx))

    # override existing
    tti.update(self.ctx, self.tt(env=[self.PE(var='VAR', value='2')]), None)

    self.assertEqual(
        self.tt(env=[self.PE(var='VAR', value='2')]),
        tti.finalize(self.ctx))

    # add new
    tti.update(
        self.ctx, self.tt(env=[self.PE(var='OTHER', value='thing')]), None)

    self.assertEqual(
        self.tt(env=[
          self.PE(var='OTHER', value='thing'),
          self.PE(var='VAR', value='2'),
        ]),
        tti.finalize(self.ctx),
    )

  def test_task_template_update_env_prefix(self):
    tti = pools_config.TaskTemplate._Intermediate(
        self.ctx, pools_pb2.TaskTemplate())
    tti.update(self.ctx, self.tt(env=[
        self.PE(var='PATH', prefix=['1'], soft=True)]), None)

    self.assertEqual(
        self.tt(env=[self.PE(var='PATH', prefix=['1'], soft=True)]),
        tti.finalize(self.ctx))

    # append existing
    tti.update(self.ctx, self.tt(env=[self.PE(var='PATH', prefix=['2'])]), None)

    self.assertEqual(
        self.tt(env=[self.PE(var='PATH', prefix=['1', '2'])]),
        tti.finalize(self.ctx))

    # existing, add new
    tti.update(
        self.ctx, self.tt(env=[self.PE(var='OTHER', prefix=['thing'])]), None)

    self.assertEqual(
        self.tt(env=[
          self.PE(var='OTHER', prefix=['thing']),
          self.PE(var='PATH', prefix=['1', '2']),
        ]),
        tti.finalize(self.ctx))

  def test_finalize_overlapping_paths(self):
    # adds stuff to self.ctx
    self.tt(
      cache=[
        self.PCE(name='other_name', path='cache_cipd/path'),

        # Cannot overlap caches
        self.PCE(name='some_name', path='good/path'),
        self.PCE(name='whatnow', path='good/path/b'),
      ],
      cipd_package=[
        self.PCP(path='good/other', pkg='some/pkg', version='latest'),
        self.PCP(path='cache_cipd', pkg='other/pkg', version='latest'),

        # multiple cipd in same dir is OK
        self.PCP(path='cache_cipd', pkg='other/pkg2', version='latest'),
      ])

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [
          ('cache u\'other_name\' uses u\'cache_cipd/path\', which conflicts '
           'with cipd[u\'other/pkg2:latest\', u\'other/pkg:latest\'] using'
           ' u\'cache_cipd\''),
          ('cache u\'whatnow\' uses u\'good/path/b\', which conflicts with '
           'cache u\'some_name\' using u\'good/path\''),
        ])

  def test_finalize_empty_values(self):
    self.tt(
        cache=[
          self.PCE(path='path'),
          self.PCE(name='cool_name'),
        ],
        cipd_package=[
          self.PCP(path='good/other', pkg='some/pkg'),
          self.PCP(pkg='some/pkg', version='latest'),
          self.PCP(path='good/other', version='latest'),
        ],
        env=[
          self.PE(value='1', prefix=['path']),
          self.PE(var='VAR', value='1', prefix=['']),
          self.PE(var='VARR'),
        ])

    self.assertEqual(
      [x.text for x in self.ctx.result().messages],
      [
        'cache[0]: empty name',
        'cache[u\'cool_name\']: empty path',
        'cipd_package[(u\'good/other\', u\'some/pkg\')]: empty version',
        'cipd_package[2]: empty pkg',
        'env[0]: empty var',
        'env[u\'VARR\']: empty value AND prefix',
        (
            'u\'\': directory has conflicting owners: cache u\'cool_name\' and'
            ' cipd[u\'some/pkg:latest\']'
        ),
      ])

  def test_simple_pb(self):
    tt = self.parse("""
    cache: { name: "hi"  path: "cache/hi" }
    cache: { name: "there"  path: "cache/there" }
    cipd_package: { path: "bin" pkg: "foo/bar" version: "latest" }
    env: {var: "VAR" value: "1"}
    env: {var: "PATH" prefix: "1" prefix: "2" soft: true}
    """)

    self.assertEqual(
        pools_config.TaskTemplate.from_pb(self.ctx, tt),
        pools_config.TaskTemplate(
            cache=(
              pools_config.CacheEntry('hi', 'cache/hi'),
              pools_config.CacheEntry('there', 'cache/there'),
            ),
            cipd_package=(
              pools_config.CipdPackage('bin', 'foo/bar', 'latest'),
            ),
            env=(
              pools_config.Env('PATH', '', ('1', '2'), True),
              pools_config.Env('VAR', '1', (), False),
            ),
            inclusions=frozenset()))

  def test_simple_include(self):
    base = pools_config.TaskTemplate.from_pb(self.ctx, self.parse("""
    cache: { name: "hi"  path: "cache/hi" }
    cipd_package: { path: "bin" pkg: "foo/bar" version: "latest" }
    env: {var: "VAR" value: "1"}
    env: {var: "PATH" prefix: "1" prefix: "2" soft: true}
    """))

    tt = self.parse("""
    include: "base"
    cache: { name: "there"  path: "cache/there" }
    cipd_package: { path: "bin" pkg: "foo/nerps" version: "yes" }
    env: {var: "VAR" value: "2"}
    env: {var: "PATH" prefix: "3" soft: true}
    """)

    self.assertEqual(
        pools_config.TaskTemplate.from_pb(self.ctx, tt, {'base': base}.get),
        pools_config.TaskTemplate(
            cache=(
              pools_config.CacheEntry('hi', 'cache/hi'),
              pools_config.CacheEntry('there', 'cache/there'),
            ),
            cipd_package=(
              pools_config.CipdPackage('bin', 'foo/bar', 'latest'),
              pools_config.CipdPackage('bin', 'foo/nerps', 'yes'),
            ),
            env=(
              pools_config.Env('PATH', '', ('1', '2', '3'), True),
              pools_config.Env('VAR', '2', (), False),
            ),
            inclusions=frozenset({'base'}),
          ))


class TestPoolCfgTaskTemplate(TaskTemplateBaseTest):
  @staticmethod
  def parse(textpb):
    return text_format.Merge(textpb, pools_pb2.PoolsCfg())

  def test_resolve_tree_inclusion(self):
    poolcfg = self.parse("""
      task_template: {
        name: "a"
        env: {var: "VAR" value: "1"}
      }
      task_template: {
        name: "b"
        env: {var: "VAR" prefix: "pfx"}
      }
      task_template: {
        name: "c"
        include: "a"
        include: "b"
      }
      task_template: {
        name: "d"
        include: "c"
      }
    """)

    template_map = pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)

    self.assertSetEqual(set('abcd'), set(template_map.keys()))

    self.assertEqual(template_map['d'], self.tt(
        env=[self.PE(var='VAR', value='1', prefix=['pfx'])],
        inclusions='abc',
    ))

  def test_resolve_repeated_inclusion(self):
    poolcfg = self.parse("""
      task_template: {name: "a"}
      task_template: {
        name: "b"
        include: "a"
        include: "a"
      }
    """)

    pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['template[u\'b\']: template u\'a\' included multiple times'])

  def test_resolve_diamond_inclusion(self):
    poolcfg = self.parse("""
      task_template: {name: "a"}
      task_template: {
        name: "b"
        include: "a"
      }
      task_template: {
        name: "c"
        include: "a"
      }
      task_template: {
        name: "d"
        include: "b" include: "c"
      }
    """)

    pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['template[u\'d\']: template u\'a\' included (transitively) multiple '
         'times'])

  def test_inclusion_cycle(self):
    poolcfg = self.parse("""
      task_template: {name: "a" include: "b"}
      task_template: {name: "b" include: "a"}
    """)

    template_map = pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)
    self.assertDictEqual(template_map, {
      'a': pools_config.TaskTemplate.CYCLE,
      'b': pools_config.TaskTemplate.CYCLE,
    })

    tail = ', which causes an import cycle'
    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [
            'template[u\'a\']: template[u\'b\']: depends on u\'a\'' + tail,
            'template[u\'a\']: depends on u\'b\'' + tail,
        ])

  def test_no_name(self):
    poolcfg = self.parse("""
      task_template: {}
    """)

    self.assertIsNone(pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template))

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['one or more templates has a blank name'])

  def test_dup_name(self):
    poolcfg = self.parse("""
      task_template: {name: "a"}
      task_template: {name: "a"}
    """)

    self.assertIsNone(pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template))

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['one or more templates has a duplicate name'])

  def test_bad_include(self):
    poolcfg = self.parse("""
      task_template: {name: "a" include: "nope"}
      task_template: {name: "b" include: "nope"}
    """)

    template_map = pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)
    self.assertDictEqual(template_map, {
      'a': None,
      'b': None,
    })

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [
          'template[u\'a\']: unknown include: u\'nope\'',
          'template[u\'b\']: unknown include: u\'nope\'',
        ])

  def test_bad_result(self):
    poolcfg = self.parse("""
      task_template: {
        name: "a"
        env: {var: "VAR" }
      }
    """)

    pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['template[u\'a\']: env[u\'VAR\']: empty value AND prefix'])


class TestPoolCfgTaskTemplateDeployments(TaskTemplateBaseTest):
  @staticmethod
  def parse(textpb):
    return text_format.Merge(textpb, pools_pb2.PoolsCfg())

  def test_resolve_deployments(self):
    poolcfg = self.parse("""
      task_template: {name: "prod" env: {var: "VAR" value: "prod"}}
      task_template: {name: "canary" env: {var: "VAR" value: "canary"}}

      task_template_deployment: {
        name: "standard"
        prod: {include: "prod"}
        canary: {include: "canary"}
        canary_chance: 5000
      }
    """)

    tmap = pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)
    dmap = pools_config._resolve_task_template_deployments(
        self.ctx, tmap, poolcfg.task_template_deployment)

    self.assertSetEqual({'standard'}, set(dmap.keys()))

    self.assertEqual(dmap['standard'], pools_config.TaskTemplateDeployment(
        prod=self.tt(env=[self.PE(var='VAR', value='prod'),],
                     inclusions={'prod'}),
        canary=self.tt(env=[self.PE(var='VAR', value='canary'),],
                       inclusions={'canary'}),
        canary_chance=5000))

  def test_resolve_noname_deployment(self):
    poolcfg = self.parse("""
      task_template_deployment: {}
    """)

    self.assertIsNone(pools_config._resolve_task_template_deployments(
        self.ctx, {}, poolcfg.task_template_deployment))

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['deployment[0]: has no name'])

  def test_resolve_bad_canary(self):
    poolcfg = self.parse("""
      task_template_deployment: {name: "a" canary_chance: 10000}
    """)

    pools_config._resolve_task_template_deployments(
        self.ctx, {}, poolcfg.task_template_deployment)

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        ['deployment[u\'a\']: '+
         'canary_chance out of range `[0,9999]`: 10000 -> %100.00'])

  def test_resolve_bad_canary_2(self):
    poolcfg = self.parse("""
      task_template_deployment: {name: "a" canary_chance: -1}
    """)

    pools_config._resolve_task_template_deployments(
        self.ctx, {}, poolcfg.task_template_deployment)

    self.assertEqual(
        [x.text for x in self.ctx.result().messages],
        [('deployment[u\'a\']: '
          'canary_chance out of range `[0,9999]`: -1 -> %-0.01')])

  def test_resolve_single_deployment(self):
    poolcfg = self.parse("""
      task_template: {name: "a" env: {var: "VAR" value: "1"} }
      task_template_deployment: {
        name: "std"
        prod: {include: "a"}
      }
      pool {
        task_template_deployment: "std"
      }
      pool {
        task_template_deployment_inline: {
          prod: {include: "a"}
          canary: {
            include: "a"
            env: {var: "WAT" value: "yes"}
          }
          canary_chance: 5000
        }
      }
    """)

    tmap = pools_config._resolve_task_template_inclusions(
        self.ctx, poolcfg.task_template)
    dmap = pools_config._resolve_task_template_deployments(
        self.ctx, tmap, poolcfg.task_template_deployment)

    self.assertEqual(pools_config.TaskTemplateDeployment(
      prod=self.tt(
        env=[self.PE(var='VAR', value='1')],
        inclusions='a'),
      canary=None, canary_chance=0,
    ), pools_config._resolve_deployment(self.ctx, poolcfg.pool[0], tmap, dmap))

    self.assertEqual(pools_config.TaskTemplateDeployment(
      prod=self.tt(
          env=[self.PE(var='VAR', value='1')],
          inclusions='a'),
      canary=self.tt(
          env=(
            self.PE(var='VAR', value='1'),
            self.PE(var='WAT', value='yes')),
          inclusions={'a'}),
      canary_chance=5000,
    ), pools_config._resolve_deployment(self.ctx, poolcfg.pool[1], tmap, dmap))


class TestBotMonitoring(TaskTemplateBaseTest):
  @staticmethod
  def parse(textpb):
    return text_format.Merge(textpb, pools_pb2.BotMonitoring())

  def validator_test(self, bm, messages):
    ctx = validation.Context()
    actual = pools_config._resolve_bot_monitoring(ctx, bm)
    self.assertEqual(ctx.result().messages, [
      validation.Message(severity=logging.ERROR, text=m)
      for m in messages
    ])
    return actual

  def test_valid_empty(self):
    bm = self.parse('name: "hi"')
    actual = self.validator_test([bm], [])
    self.assertEqual({u'hi': ['pool']}, actual)

  def test_valid_normal(self):
    bm = self.parse("""
    name: "hi"
    dimension_key: "a"
    dimension_key: "z"
    """)
    actual = self.validator_test([bm], [])
    self.assertEqual({u'hi': [u'a', 'pool', u'z']}, actual)

  def test_bad_name(self):
    bm = self.parse('name: "hi "')
    self.validator_test([bm], ['bot_monitoring u\'hi \': invalid name'])

  def test_name_missing(self):
    bm = self.parse('')
    self.validator_test([bm], ['bot_monitoring u\'\': invalid name'])

  def test_bad_dimension_key(self):
    bm = self.parse("""
    name: "hi"
    dimension_key: "first "
    """)
    self.validator_test(
        [bm], ['bot_monitoring u\'hi\': invalid dimension_key u\'first \''])

  def test_bad_repeated_dimension_key(self):
    bm = self.parse("""
    name: "hi"
    dimension_key: "same"
    dimension_key: "same"
    """)
    self.validator_test(
        [bm], ['bot_monitoring u\'hi\': duplicate dimension_key'])

  def test_bad_repeated_name(self):
    bm = [
      self.parse('name: "hi"'),
      self.parse('name: "hi"'),
    ]
    self.validator_test(bm, ['bot_monitoring u\'hi\': duplicate name'])


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.CRITICAL)
  unittest.main()
