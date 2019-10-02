# coding: utf-8
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Base class for handlers_*_test.py"""

import base64
import json
import os

import test_env
test_env.setup_test_env()

from protorpc.remote import protojson
import webtest

import event_mon_metrics
import handlers_endpoints
import swarming_rpcs
from components import auth
from components import auth_testing
from components import utils
import gae_ts_mon
from test_support import test_case

from proto.config import config_pb2
from server import config
from server import large
from server import pools_config
from server import service_accounts


class AppTestBase(test_case.TestCase):
  APP_DIR = test_env.APP_DIR

  def setUp(self):
    super(AppTestBase, self).setUp()
    self.bot_version = None
    self.source_ip = '192.168.2.2'
    self.testbed.init_user_stub()

    gae_ts_mon.reset_for_unittest(disable=True)
    event_mon_metrics.initialize()

    # By default requests in tests are coming from bot with fake IP.
    # WSGI app that implements auth REST API.
    self.auth_app = webtest.TestApp(
        auth.create_wsgi_application(debug=True),
        extra_environ={
          'REMOTE_ADDR': self.source_ip,
          'SERVER_SOFTWARE': os.environ['SERVER_SOFTWARE'],
        })

    admins_group = 'test_admins_group'
    priv_users_group = 'test_priv_users_group'
    users_group = 'test_users_group'

    cfg = config_pb2.SettingsCfg(auth=config_pb2.AuthSettings(
        admins_group=admins_group,
        privileged_users_group=priv_users_group,
        users_group=users_group,
    ))
    self.mock(config, '_get_settings', lambda: ('test_rev', cfg))
    utils.clear_cache(config.settings)

    # Note that auth.ADMIN_GROUP != admins_group.
    auth.bootstrap_group(
        auth.ADMIN_GROUP,
        [auth.Identity(auth.IDENTITY_USER, 'super-admin@example.com')])
    auth.bootstrap_group(
        admins_group,
        [auth.Identity(auth.IDENTITY_USER, 'admin@example.com')])
    auth.bootstrap_group(
        priv_users_group,
        [auth.Identity(auth.IDENTITY_USER, 'priv@example.com')])
    auth.bootstrap_group(
        users_group,
        [auth.Identity(auth.IDENTITY_USER, 'user@example.com')])

  def set_as_anonymous(self):
    """Removes all IPs from the whitelist."""
    self.testbed.setup_env(USER_EMAIL='', overwrite=True)
    auth.ip_whitelist_key(auth.bots_ip_whitelist()).delete()
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(self, auth.Anonymous)

  def set_as_super_admin(self):
    self.set_as_anonymous()
    self.testbed.setup_env(USER_EMAIL='super-admin@example.com', overwrite=True)
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(
        self, auth.Identity.from_bytes('user:' + os.environ['USER_EMAIL']))

  def set_as_admin(self):
    self.set_as_anonymous()
    self.testbed.setup_env(USER_EMAIL='admin@example.com', overwrite=True)
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(
        self, auth.Identity.from_bytes('user:' + os.environ['USER_EMAIL']))

  def set_as_privileged_user(self):
    self.set_as_anonymous()
    self.testbed.setup_env(USER_EMAIL='priv@example.com', overwrite=True)
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(
        self, auth.Identity.from_bytes('user:' + os.environ['USER_EMAIL']))

  def set_as_user(self):
    self.set_as_anonymous()
    self.testbed.setup_env(USER_EMAIL='user@example.com', overwrite=True)
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(
        self, auth.Identity.from_bytes('user:' + os.environ['USER_EMAIL']))

  def set_as_bot(self):
    self.set_as_anonymous()
    auth.bootstrap_ip_whitelist(auth.bots_ip_whitelist(), [self.source_ip])
    auth_testing.reset_local_state()
    auth_testing.mock_get_current_identity(self, auth.IP_WHITELISTED_BOT_ID)

  # Web or generic

  def get_xsrf_token(self):
    """Gets the generic XSRF token for web clients."""
    resp = self.auth_app.post(
        '/auth/api/v1/accounts/self/xsrf_token',
        headers={'X-XSRF-Token-Request': '1'}).json
    return resp['xsrf_token'].encode('ascii')

  def post_json(self, url, params, **kwargs):
    """Does an HTTP POST with a JSON API and return JSON response."""
    return self.app.post_json(url, params=params, **kwargs).json

  def mock_task_service_accounts(self, exc=None):
    """Mocks support for task-associated service accounts."""
    self.mock(service_accounts, 'has_token_server', lambda: True)
    calls = []
    def mocked(service_account, validity_duration):
      calls.append((service_account, validity_duration))
      if exc:
        raise exc  # pylint: disable=raising-bad-type
      return 'token-grant-%s-%d' % (
          str(service_account), validity_duration.total_seconds())
    self.mock(service_accounts, 'get_oauth_token_grant', mocked)
    return calls

  # pylint: disable=redefined-outer-name
  def mock_default_pool_acl(self, service_accounts):
    """Mocks ACLs of 'default' pool to allow usage of given service accounts."""
    assert isinstance(service_accounts, (list, tuple)), service_accounts
    def mocked_fetch_pools_config():
      default_isolate = pools_config.IsolateServer(
        server='https://pool.config.isolate.example.com',
        namespace='default-gzip',
      )
      default_cipd = pools_config.CipdServer(
        server='https://pool.config.cipd.example.com',
        client_version='from_pool_config',
      )
      return pools_config._PoolsCfg(
          {
            "template": pools_config.PoolConfig(
              name='template',
              rev='pools_cfg_rev',
              scheduling_users=frozenset([
                # See setUp above. We just duplicate the first ACL layer here
                auth.Identity(auth.IDENTITY_USER, 'super-admin@example.com'),
                auth.Identity(auth.IDENTITY_USER, 'admin@example.com'),
                auth.Identity(auth.IDENTITY_USER, 'priv@example.com'),
                auth.Identity(auth.IDENTITY_USER, 'user@example.com'),
              ]),
              scheduling_groups=frozenset(),
              trusted_delegatees={},
              service_accounts=frozenset(service_accounts),
              service_accounts_groups=(),
              task_template_deployment=pools_config.TaskTemplateDeployment(
                  prod=pools_config.TaskTemplate(
                    cache=(),
                    cipd_package=(),
                    env=(pools_config.Env('VAR', 'prod', (), False),),
                    inclusions=()),
                  canary=pools_config.TaskTemplate(
                    cache=(),
                    cipd_package=(),
                    env=(pools_config.Env('VAR', 'canary', (), False),),
                    inclusions=()),
                  canary_chance=0.5,
              ),
              default_isolate=default_isolate,
              default_cipd=default_cipd,
              bot_monitoring=None,
              external_schedulers=None,
            ),

            "default": pools_config.PoolConfig(
              name='default',
              rev='pools_cfg_rev',
              scheduling_users=frozenset([
                # See setUp above. We just duplicate the first ACL layer here
                auth.Identity(auth.IDENTITY_USER, 'super-admin@example.com'),
                auth.Identity(auth.IDENTITY_USER, 'admin@example.com'),
                auth.Identity(auth.IDENTITY_USER, 'priv@example.com'),
                auth.Identity(auth.IDENTITY_USER, 'user@example.com'),
              ]),
              scheduling_groups=frozenset(),
              trusted_delegatees={},
              service_accounts=frozenset(service_accounts),
              service_accounts_groups=(),
              task_template_deployment=None,
              bot_monitoring=None,
              default_isolate=default_isolate,
              default_cipd=default_cipd,
              external_schedulers=None,
            ),
          },
        (default_isolate, default_cipd)
    )

    self.mock(pools_config, '_fetch_pools_config', mocked_fetch_pools_config)

  # Bot

  def do_handshake(self, bot='bot1'):
    """Performs bot handshake, returns data to be sent to bot handlers.

    Also populates self.bot_version.
    """
    params = {
      'dimensions': {
        'id': [bot],
        'os': ['Amiga'],
        'pool': ['default'],
      },
      'state': {
        'running_time': 1234.0,
        'sleep_streak': 0,
        'started_ts': 1410990411.111,
      },
      'version': '123',
    }
    response = self.app.post_json(
        '/swarming/api/v1/bot/handshake',
        params=params).json
    self.bot_version = response['bot_version']
    params['version'] = self.bot_version
    params['state']['bot_group_cfg_version'] = response['bot_group_cfg_version']
    # A bit hackish but fine for unit testing purpose.
    if response.get('bot_config'):
      params['bot_config'] = response['bot_config']
    return params

  def bot_poll(self, bot='bot1', params=None):
    """Simulates a bot that polls for task."""
    if not params:
      params = self.do_handshake(bot)
    return self.post_json('/swarming/api/v1/bot/poll', params)

  def bot_complete_task(self, **kwargs):
    # Emulate an isolated task.
    params = {
      'cost_usd': 0.1,
      'duration': 0.1,
      'bot_overhead': 0.1,
      'exit_code': 0,
      'id': 'bot1',
      'isolated_stats': {
        'download': {
          'duration': 1.,
          'initial_number_items': 10,
          'initial_size': 100000,
          'items_cold': [20],
          'items_hot': [30, 40],
        },
        'upload': {
          'duration': 2.,
          'items_cold': [1, 2, 40],
          'items_hot': [1, 2, 3, 50],
        },
      },
      'output': base64.b64encode(u'rÉsult string'.encode('utf-8')),
      'output_chunk_start': 0,
      'task_id': None,
    }
    for k in ('download', 'upload'):
      for j in ('items_cold', 'items_hot'):
        params['isolated_stats'][k][j] = base64.b64encode(
            large.pack(params['isolated_stats'][k][j]))
    params.update(kwargs)
    return self.post_json('/swarming/api/v1/bot/task_update', params)

  def bot_run_task(self):
    res = self.bot_poll()
    task_id = res['manifest']['task_id']
    response = self.bot_complete_task(task_id=task_id)
    self.assertEqual({u'must_stop': False, u'ok': True}, response)
    return task_id

  # Client

  def endpoint_call(self, service, name, args):
    srv = test_case.Endpoints(service, source_ip=self.source_ip)
    if not isinstance(args, dict):
      args = json.loads(protojson.encode_message(args))
    return srv.call_api(name, body=args).json

  @staticmethod
  def create_props(**kwargs):
    """Returns a serialized swarming_rpcs.TaskProperties."""
    out = {
      u'cipd_input': {
        u'client_package': {
          u'package_name': u'infra/tools/cipd/${platform}',
          u'version': u'git_revision:deadbeef',
        },
        u'packages': [{
          u'package_name': u'rm',
          u'path': u'bin',
          u'version': u'git_revision:deadbeef',
        }],
        u'server': u'https://pool.config.cipd.example.com',
      },
      u'dimensions': [
        {u'key': u'os', u'value': u'Amiga'},
        {u'key': u'pool', u'value': u'default'},
      ],
      u'env': [],
      u'execution_timeout_secs': 3600,
      u'io_timeout_secs': 1200,
      u'outputs': [u'foo', u'path/to/foobar']
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    return out

  def create_new_request(self, **kwargs):
    """Returns an initialized swarming_rpcs.TaskNewRequest.

    Useful to use a swarming_rpcs.TaskSlice.
    """
    out = {
      'expiration_secs': 24*60*60,
      'name': 'job1',
      'priority': 20,
      'tags': [u'a:tag'],
      'user': 'joe@localhost',
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    # Note that protorpc message constructor accepts dicts for submessages.
    return swarming_rpcs.NewTaskRequest(**out)

  def client_cancel_task(self, task_id, kill_running):
    return self.endpoint_call(
        handlers_endpoints.SwarmingTaskService,
        'cancel', {'task_id': task_id, 'kill_running': kill_running})

  def client_create_task(self, **kwargs):
    """Creates a minimal task request via the Cloud Endpoints API."""
    request = self.create_new_request(**kwargs)
    response = self.endpoint_call(
        handlers_endpoints.SwarmingTasksService, 'new', request)
    return response, response['task_id']

  def client_create_task_isolated(self, properties=None, **kwargs):
    """Creates a TaskRequest using an isolated tree via the Cloud Endpoints API.
    """
    properties = (properties or {}).copy()
    properties['inputs_ref'] = {
      'isolated': '0123456789012345678901234567890123456789',
      'isolatedserver': 'http://localhost:1',
      'namespace': 'default-gzip',
    }
    return self.client_create_task(
        properties=self.create_props(**properties), **kwargs)

  def client_create_task_raw(self, properties=None, **kwargs):
    """Creates a raw command TaskRequest via the Cloud Endpoints API."""
    properties = (properties or {}).copy()
    properties['command'] = ['python', 'run_test.py']
    return self.client_create_task(
        properties=self.create_props(**properties), **kwargs)

  def client_get_results(self, task_id):
    return self.endpoint_call(
        handlers_endpoints.SwarmingTaskService, 'result', {'task_id': task_id})

  @staticmethod
  def gen_props(**kwargs):
    """Returns a serialized swarming_rpcs.TaskProperties.

    To be used for expectations.
    """
    out = {
      u'cipd_input': {
        u'client_package': {
          u'package_name': u'infra/tools/cipd/${platform}',
          u'version': u'git_revision:deadbeef',
        },
        u'packages': [{
          u'package_name': u'rm',
          u'path': u'bin',
          u'version': u'git_revision:deadbeef',
        }],
        u'server': u'https://pool.config.cipd.example.com',
      },
      u'dimensions': [
        {u'key': u'os', u'value': u'Amiga'},
        {u'key': u'pool', u'value': u'default'},
      ],
      u'execution_timeout_secs': u'3600',
      u'grace_period_secs': u'30',
      u'idempotent': False,
      u'inputs_ref': {
        'isolatedserver': 'https://pool.config.isolate.example.com',
        'namespace': 'default-gzip',
      },
      u'io_timeout_secs': u'1200',
      u'outputs': [u'foo', u'path/to/foobar']
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    return out

  @staticmethod
  def gen_request(**kwargs):
    """Returns a serialized swarming_rpcs.TaskRequest.

    To be used for expectations.
    """
    out = {
      u'authenticated': u'user:user@example.com',
      u'expiration_secs': u'86400',
      u'name': u'job1',
      u'priority': u'20',
      u'service_account': u'none',
      u'tags': [
        u'a:tag',
        u'os:Amiga',
        u'pool:default',
        u'priority:20',
        u'service_account:none',
        u'swarming.pool.template:none',
        u'swarming.pool.version:pools_cfg_rev',
        u'user:joe@localhost',
      ],
      u'user': u'joe@localhost',
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    return out

  @staticmethod
  def gen_perf_stats(**kwargs):
    """Returns a serialized swarming_rpcs.PerformanceStats.

    To be used for expectations.
    """
    out = {
      u'bot_overhead': 0.1,
      u'isolated_download': {
        u'duration': 1.0,
        u'initial_number_items': u'10',
        u'initial_size': u'100000',
        u'items_cold': [20],
        u'items_hot': [30, 40],
        u'num_items_cold': u'1',
        u'total_bytes_items_cold': u'20',
        u'num_items_hot': u'2',
        u'total_bytes_items_hot': u'70',
      },
      u'isolated_upload': {
        u'duration': 2.0,
        u'items_cold': [1, 2, 40],
        u'items_hot': [1, 2, 3, 50],
        u'num_items_cold': u'3',
        u'total_bytes_items_cold': u'43',
        u'num_items_hot': u'4',
        u'total_bytes_items_hot': u'56',
      },
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    return out

  def gen_result_summary(self, **kwargs):
    """Returns a serialized swarming_rpcs.TaskResult initialized from a
    TaskResultSummary.

    To be used for expectations.
    """
    out = {
      u'bot_dimensions': [
        {u'key': u'id', u'value': [u'bot1']},
        {u'key': u'os', u'value': [u'Amiga']},
        {u'key': u'pool', u'value': [u'default']},
      ],
      u'bot_id': u'bot1',
      u'bot_version': self.bot_version,
      u'current_task_slice': u'0',
      u'failure': False,
      u'internal_failure': False,
      u'name': u'job1',
      u'run_id': u'5cee488008811',
      u'server_versions': [u'v1a'],
      u'state': u'COMPLETED',
      u'tags': [
        u'a:tag',
        u'os:Amiga',
        u'pool:default',
        u'priority:20',
        u'service_account:none',
        u'swarming.pool.template:no_config',
        u'user:joe@localhost',
      ],
      u'task_id': u'5cee488008810',
      u'try_number': u'0',
      u'user': u'joe@localhost',
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    return out

  def gen_run_result(self, **kwargs):
    """Returns a serialized swarming_rpcs.TaskResult initialized from a
    TaskRunResult.

    To be used for expectations.
    """
    out = {
      u'bot_dimensions': [
        {u'key': u'id', u'value': [u'bot1']},
        {u'key': u'os', u'value': [u'Amiga']},
        {u'key': u'pool', u'value': [u'default']},
      ],
      u'bot_id': u'bot1',
      u'bot_version': self.bot_version,
      u'costs_usd': [0.0],
      u'current_task_slice': u'0',
      u'failure': False,
      u'internal_failure': False,
      u'name': u'job1',
      u'run_id': u'5cee488008811',
      u'server_versions': [u'v1a'],
      u'state': u'RUNNING',
      u'task_id': u'5cee488008811',
      u'try_number': u'1',
    }
    out.update((unicode(k), v) for k, v in kwargs.iteritems())
    return out
