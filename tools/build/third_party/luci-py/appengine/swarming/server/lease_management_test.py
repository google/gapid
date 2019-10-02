#!/usr/bin/python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for lease_management.py."""

import datetime
import json
import logging
import sys
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb
from protorpc.remote import protojson
import webtest

from components import machine_provider
from components import utils
from test_support import test_case

import bot_management
import lease_management
from proto.config import bots_pb2


def rpc_to_json(rpc_message):
  """Converts the given RPC message to a POSTable JSON dict.

  Args:
    rpc_message: A protorpc.message.Message instance.

  Returns:
    A string representing a JSON dict.
  """
  return json.loads(protojson.encode_message(rpc_message))


class TestCase(test_case.TestCase):
  def setUp(self):
    super(TestCase, self).setUp()
    self.mock_machine_types({})

  def mock_machine_types(self, cfg):
    self.mock(
        lease_management.bot_groups_config,
        'fetch_machine_types',
        lambda: cfg,
    )


class AssociateBotIdTest(TestCase):
  """Tests for lease_management._associate_bot_id."""

  def test_hostname_unset(self):
    key = lease_management.MachineLease().put()
    lease_management._associate_bot_id(key, 'id')
    self.assertFalse(key.get().bot_id)
    self.assertFalse(key.get().hostname)

  def test_hostname_mismatch(self):
    key = lease_management.MachineLease(hostname='id1').put()
    lease_management._associate_bot_id(key, 'id2')
    self.assertFalse(key.get().bot_id)
    self.assertEqual(key.get().hostname, 'id1')

  def test_bot_id_mismatch(self):
    key = lease_management.MachineLease(bot_id='id1', hostname='id1').put()
    lease_management._associate_bot_id(key, 'id2')
    self.assertEqual(key.get().bot_id, 'id1')
    self.assertEqual(key.get().hostname, 'id1')

  def test_hostname_set(self):
    key = lease_management.MachineLease(hostname='id1').put()
    lease_management._associate_bot_id(key, 'id1')
    self.assertEqual(key.get().bot_id, 'id1')
    self.assertEqual(key.get().hostname, 'id1')

  def test_bot_id_match(self):
    key = lease_management.MachineLease(bot_id='id1', hostname='id1').put()
    lease_management._associate_bot_id(key, 'id1')
    self.assertEqual(key.get().bot_id, 'id1')
    self.assertEqual(key.get().hostname, 'id1')


class CheckForConnectionTest(TestCase):
  """Tests for lease_management._check_for_connection."""

  def test_not_connected(self):
    machine_lease = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='req-id',
        hostname='bot-id',
        instruction_ts=utils.utcnow(),
        machine_type=ndb.Key(lease_management.MachineType, 'mt'),
    )
    machine_lease.put()
    bot_management.bot_event(
        event_type='bot_leased',
        bot_id=machine_lease.hostname,
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
    )

    lease_management._check_for_connection(machine_lease)
    self.failUnless(bot_management.get_info_key(machine_lease.bot_id).get())
    self.failUnless(machine_lease.key.get().client_request_id)
    self.failIf(machine_lease.key.get().connection_ts)

  def test_connected(self):
    machine_lease = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='req-id',
        hostname='bot-id',
        instruction_ts=utils.utcnow(),
        machine_type=ndb.Key(lease_management.MachineType, 'mt'),
    )
    machine_lease.put()
    bot_management.bot_event(
        event_type='bot_leased',
        bot_id=machine_lease.hostname,
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
    )
    bot_management.bot_event(
        event_type='bot_connected',
        bot_id=machine_lease.hostname,
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
    )

    lease_management._check_for_connection(machine_lease)
    self.failUnless(bot_management.get_info_key(machine_lease.bot_id).get())
    self.failUnless(machine_lease.key.get().client_request_id)
    self.failUnless(machine_lease.key.get().connection_ts)

  def test_connected_earlier_than_instructed(self):
    bot_management.bot_event(
        event_type='bot_connected',
        bot_id='bot-id',
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
    )
    machine_lease = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='req-id',
        hostname='bot-id',
        instruction_ts=utils.utcnow(),
        machine_type=ndb.Key(lease_management.MachineType, 'mt'),
    )
    machine_lease.put()
    bot_management.bot_event(
        event_type='bot_leased',
        bot_id=machine_lease.hostname,
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
    )

    lease_management._check_for_connection(machine_lease)
    self.failUnless(bot_management.get_info_key(machine_lease.bot_id).get())
    self.failUnless(machine_lease.key.get().client_request_id)
    self.failIf(machine_lease.key.get().connection_ts)

  def test_missing(self):
    self.mock(lease_management, 'release', lambda *args, **kwargs: True)

    machine_lease = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='req-id',
        hostname='bot-id',
        instruction_ts=utils.utcnow(),
        machine_type=ndb.Key(lease_management.MachineType, 'mt'),
    )
    machine_lease.put()

    lease_management._check_for_connection(machine_lease)
    self.failIf(bot_management.get_info_key(machine_lease.bot_id).get())
    self.failIf(machine_lease.key.get().client_request_id)
    self.failIf(machine_lease.key.get().connection_ts)

  def test_dead(self):
    def is_dead(_self, _now):
      return True
    self.mock(bot_management.BotInfo, 'is_dead', is_dead)
    self.mock(lease_management, 'release', lambda *args, **kwargs: True)

    machine_lease = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='req-id',
        hostname='bot-id',
        instruction_ts=utils.utcnow(),
        machine_type=ndb.Key(lease_management.MachineType, 'mt'),
    )
    machine_lease.put()
    bot_management.bot_event(
        event_type='bot_leased',
        bot_id=machine_lease.hostname,
        external_ip=None,
        authenticated_as=None,
        dimensions=None,
        state=None,
        version=None,
        quarantined=False,
        maintenance_msg=None,
        task_id='',
        task_name=None,
    )

    lease_management._check_for_connection(machine_lease)
    self.failIf(bot_management.get_info_key(machine_lease.bot_id).get())
    self.failIf(machine_lease.key.get().client_request_id)
    self.failIf(machine_lease.key.get().connection_ts)


class ComputeUtilizationTest(TestCase):
  """Tests for lease_management.cron_compute_utilization."""
  APP_DIR = test_env.APP_DIR

  def test_no_machine_provider_bots(self):
    bots = [
    ]
    def fetch_page(*_args, **_kwargs):
      return bots, None
    self.mock(lease_management.datastore_utils, 'fetch_page', fetch_page)

    lease_management.MachineType(
        id='machine-type',
        target_size=1,
    ).put()
    key = ndb.Key(lease_management.MachineTypeUtilization, 'machine-type')

    self.assertEqual(0, lease_management.cron_compute_utilization())

    self.failIf(key.get())

  def test_machine_provider_bots(self):
    ndb.get_context().set_cache_policy(lambda _: None)
    now = utils.utcnow()
    bots = [
        bot_management.BotInfo(
            key=bot_management.get_info_key('bot1'),
            machine_type='machine-type-1',
            last_seen_ts=now,
        ),
        bot_management.BotInfo(
            key=bot_management.get_info_key('bot2'),
            machine_type='machine-type-1',
            last_seen_ts=now,
        ),
        bot_management.BotInfo(
            key=bot_management.get_info_key('bot3'),
            machine_type='machine-type-2',
            last_seen_ts=now,
            task_id='task',
        ),
        bot_management.BotInfo(
            key=bot_management.get_info_key('bot4'),
            machine_type='machine-type-3',
            last_seen_ts=now,
            task_id='task',
        ),
        bot_management.BotInfo(
            key=bot_management.get_info_key('bot5'),
            machine_type='machine-type-3',
            last_seen_ts=now,
        ),
        bot_management.BotInfo(
            key=bot_management.get_info_key('bot6'),
            machine_type='machine-type-3',
            last_seen_ts=now,
            task_id='task',
        ),
    ]
    ndb.put_multi(bots)

    obj1 = lease_management.MachineType(id='machine-type-1', target_size=2)
    obj1.put()
    obj2 = lease_management.MachineType(id='machine-type-2', target_size=1)
    obj2.put()
    obj3 = lease_management.MachineType(id='machine-type-3', target_size=1)
    obj3.put()

    self.assertEqual(3, lease_management.cron_compute_utilization())

    u1 = ndb.Key(lease_management.MachineTypeUtilization,
        obj1.key.string_id()).get()
    self.assertEqual(u1.busy, 0)
    self.assertEqual(u1.idle, 2)
    self.failUnless(u1.last_updated_ts)

    u2 = ndb.Key(lease_management.MachineTypeUtilization,
        obj2.key.string_id()).get()
    self.assertEqual(u2.busy, 1)
    self.assertEqual(u2.idle, 0)
    self.failUnless(u2.last_updated_ts)

    u3 = ndb.Key(lease_management.MachineTypeUtilization,
        obj3.key.string_id()).get()
    self.assertEqual(u3.busy, 2)
    self.assertEqual(u3.idle, 1)
    self.failUnless(u3.last_updated_ts)


class DrainExcessTest(TestCase):
  """Tests for lease_management._drain_excess."""

  def test_no_machine_types(self):
    lease_management._drain_excess()

    self.failIf(lease_management.MachineLease.query().count())

  def test_nothing_to_drain(self):
    key = lease_management.MachineType(
        target_size=1,
    ).put()
    key = lease_management.MachineLease(
        id='%s-0' % key.id(),
        machine_type=key,
    ).put()

    lease_management._drain_excess()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.failIf(key.get().drained)

  def test_drain_one(self):
    key = lease_management.MachineType(
        target_size=0,
    ).put()
    key = lease_management.MachineLease(
        id='%s-0' % key.id(),
        machine_type=key,
    ).put()

    lease_management._drain_excess()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.assertTrue(key.get().drained)

  def test_drain_all(self):
    key = lease_management.MachineType(
        enabled=False,
        target_size=3,
    ).put()
    lease_management.MachineLease(
        id='%s-0' % key.id(),
        machine_type=key,
    ).put()
    lease_management.MachineLease(
        id='%s-1' % key.id(),
        machine_type=key,
    ).put()
    lease_management.MachineLease(
        id='%s-2' % key.id(),
        machine_type=key,
    ).put()

    lease_management._drain_excess()

    self.assertEqual(lease_management.MachineLease.query().count(), 3)
    for machine_lease in lease_management.MachineLease.query():
      self.assertTrue(machine_lease.drained)

  def test_drain_batched(self):
    key = lease_management.MachineType(
        enabled=False,
        target_size=2,
    ).put()
    lease_management.MachineLease(
        id='%s-0' % key.id(),
        machine_type=key,
    ).put()
    lease_management.MachineLease(
        id='%s-1' % key.id(),
        machine_type=key,
    ).put()
    key = lease_management.MachineType(
        enabled=False,
        target_size=2,
    ).put()
    lease_management.MachineLease(
        id='%s-0' % key.id(),
        machine_type=key,
    ).put()
    lease_management.MachineLease(
        id='%s-1' % key.id(),
        machine_type=key,
    ).put()
    key = lease_management.MachineType(
        target_size=0,
    ).put()
    lease_management.MachineLease(
        id='%s-0' % key.id(),
        machine_type=key,
    ).put()

    # Choice of 2, 2, 1 above and 3 here ensures at least one batch contains
    # MachineLease entities created for two different MachineTypes.
    lease_management._drain_excess(max_concurrent=3)

    self.assertEqual(lease_management.MachineLease.query().count(), 5)
    for machine_lease in lease_management.MachineLease.query():
      self.assertTrue(machine_lease.drained)


class EnsureBotInfoExistsTest(TestCase):
  """Tests for lease_management._ensure_bot_info_exists."""

  def test_creates(self):
    key = lease_management.MachineLease(
        id='machine-type-1',
        hostname='hostname',
        lease_id='lease-id',
        lease_expiration_ts=utils.utcnow(),
        machine_type=ndb.Key(lease_management.MachineType, 'machine-type'),
    ).put()

    lease_management._ensure_bot_info_exists(key.get())

    machine_lease = key.get()
    bot_info = bot_management.get_info_key(machine_lease.bot_id).get()
    self.assertEqual(machine_lease.bot_id, machine_lease.hostname)
    self.assertEqual(bot_info.lease_id, machine_lease.lease_id)
    self.assertEqual(
        bot_info.lease_expiration_ts, machine_lease.lease_expiration_ts)
    self.assertTrue(bot_info.lease_expiration_ts)
    self.assertEqual(
        bot_info.leased_indefinitely, machine_lease.leased_indefinitely)
    self.assertFalse(bot_info.leased_indefinitely)
    self.assertEqual(bot_info.machine_type, machine_lease.machine_type.id())
    self.assertEqual(bot_info.machine_lease, machine_lease.key.id())

  def test_creates_indefinite(self):
    key = lease_management.MachineLease(
        id='machine-type-1',
        hostname='hostname',
        lease_id='lease-id',
        leased_indefinitely=True,
        machine_type=ndb.Key(lease_management.MachineType, 'machine-type'),
    ).put()

    lease_management._ensure_bot_info_exists(key.get())

    machine_lease = key.get()
    bot_info = bot_management.get_info_key(machine_lease.bot_id).get()
    self.assertEqual(machine_lease.bot_id, machine_lease.hostname)
    self.assertEqual(bot_info.lease_id, machine_lease.lease_id)
    self.assertEqual(
        bot_info.lease_expiration_ts, machine_lease.lease_expiration_ts)
    self.assertFalse(bot_info.lease_expiration_ts)
    self.assertEqual(
        bot_info.leased_indefinitely, machine_lease.leased_indefinitely)
    self.assertTrue(bot_info.leased_indefinitely)
    self.assertEqual(bot_info.machine_type, machine_lease.machine_type.id())
    self.assertEqual(bot_info.machine_lease, machine_lease.key.id())


class EnsureEntitiesExistTest(TestCase):
  """Tests for lease_management._ensure_entities_exist."""

  def test_no_machine_types(self):
    lease_management._ensure_entities_exist()

    self.failIf(lease_management.MachineLease.query().count())

  def test_no_enabled_machine_types(self):
    lease_management.MachineType(
        enabled=False,
        target_size=3,
    ).put()

    lease_management._ensure_entities_exist()

    self.failIf(lease_management.MachineLease.query().count())

  def test_one_enabled_machine_type(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=[
                  'disk_gb:100',
                  'snapshot_labels:label1',
                  'snapshot_labels:label2',
              ],
              name='machine-type',
              target_size=1,
          ),
      })

    key = lease_management.MachineType(
        id='machine-type',
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(key.get().early_release_secs, 0)
    self.assertEqual(key.get().lease_duration_secs, 1)
    self.assertEqual(key.get().mp_dimensions.disk_gb, 100)
    self.assertEqual(key.get().mp_dimensions.snapshot_labels[0], 'label1')
    self.assertEqual(key.get().mp_dimensions.snapshot_labels[1], 'label2')
    self.assertEqual(key.get().target_size, 1)
    self.assertEqual(lease_management.MachineLease.query().count(), 1)

  def test_two_enabled_machine_types(self):
    self.mock_machine_types(
      {
          'machine-type-a': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type-a',
              target_size=1,
          ),
          'machine-type-b': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type-b',
              target_size=1,
          ),
      })

    lease_management.MachineType(
        id='machine-type-a',
        target_size=1,
    ).put()
    lease_management.MachineType(
        id='machine-type-b',
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 2)
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-a-0'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-b-0'))

  def test_one_machine_type_multiple_batches(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=5,
          ),
      })

    lease_management.MachineType(
        id='machine-type',
        target_size=5,
    ).put()

    # Choice of 3 here and 5 above ensures MachineLeases are created in two
    # batches of differing sizes.
    lease_management._ensure_entities_exist(max_concurrent=3)

    self.assertEqual(lease_management.MachineLease.query().count(), 5)
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-0'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-1'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-2'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-3'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-4'))

  def test_three_machine_types_multiple_batches(self):
    self.mock_machine_types(
      {
          'machine-type-a': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type-a',
              target_size=2,
          ),
          'machine-type-b': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type-b',
              target_size=2,
          ),
          'machine-type-c': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type-c',
              target_size=1,
          ),
      })

    lease_management.MachineType(
        id='machine-type-a',
        target_size=2,
    ).put()
    lease_management.MachineType(
        id='machine-type-b',
        target_size=2,
    ).put()
    lease_management.MachineType(
        id='machine-type-c',
        target_size=1,
    ).put()

    # Choice of 2, 2, 1 above and 3 here ensures at least one batch contains
    # MachineLease entities created for two different MachineTypes.
    lease_management._ensure_entities_exist(max_concurrent=3)

    self.assertEqual(lease_management.MachineLease.query().count(), 5)
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-a-0'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-a-1'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-b-0'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-b-1'))
    self.failUnless(lease_management.MachineLease.get_by_id('machine-type-c-0'))

  def test_enable_machine_type(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
          ),
      })
    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        enabled=False,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.failUnless(key.get().enabled)

  def test_update_machine_type(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=2,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
          ),
      })
    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        enabled=True,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(key.get().lease_duration_secs, 2)

  def test_enable_and_update_machine_type(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=2,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
          ),
      })
    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        enabled=False,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.failUnless(key.get().enabled)
    self.assertEqual(key.get().lease_duration_secs, 2)

  def test_disable_machine_type(self):
    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        enabled=True,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.failIf(key.get().enabled)

  def test_machine_lease_exists_mismatched_not_updated(self):
    key = lease_management.MachineType(
        early_release_secs=0,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()
    key = lease_management.MachineLease(
        id='%s-0' % key.id(),
        early_release_secs=1,
        lease_duration_secs=2,
        machine_type=key,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=200,
        ),
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.assertEqual(key.get().early_release_secs, 1)
    self.assertEqual(key.get().lease_duration_secs, 2)
    self.assertEqual(key.get().mp_dimensions.disk_gb, 200)

  def test_machine_lease_exists_mismatched_updated(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
          ),
      })

    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()
    key = lease_management.MachineLease(
        id='%s-0' % key.id(),
        early_release_secs=1,
        lease_duration_secs=2,
        lease_expiration_ts=utils.utcnow(),
        machine_type=key,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=200,
        ),
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.assertEqual(key.get().early_release_secs, 0)
    self.assertEqual(key.get().lease_duration_secs, 1)
    self.assertEqual(key.get().mp_dimensions.disk_gb, 100)

  def test_machine_lease_exists_mismatched_updated_to_indefinite(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              lease_indefinitely=True,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
          ),
      })

    key = lease_management.MachineType(
        id='machine-type',
        lease_indefinitely=True,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()
    key = lease_management.MachineLease(
        id='%s-0' % key.id(),
        early_release_secs=1,
        lease_duration_secs=2,
        lease_expiration_ts=utils.utcnow(),
        machine_type=key,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.assertFalse(key.get().early_release_secs)
    self.assertFalse(key.get().lease_duration_secs)
    self.assertTrue(key.get().lease_indefinitely)
    self.assertFalse(key.get().drained)

  def test_machine_lease_exists_mismatched_updated_to_finite(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
          ),
      })

    key = lease_management.MachineType(
        id='machine-type',
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()
    key = lease_management.MachineLease(
        id='%s-0' % key.id(),
        lease_indefinitely=True,
        leased_indefinitely=True,
        machine_type=key,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.assertEqual(key.get().lease_duration_secs, 1)
    self.assertFalse(key.get().lease_indefinitely)
    self.assertTrue(key.get().drained)

  def test_daily_schedule_resize(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
              schedule=bots_pb2.Schedule(
                  daily=[bots_pb2.DailySchedule(
                      start='0:00',
                      end='1:00',
                      days_of_the_week=xrange(7),
                      target_size=3,
                  )],
              ),
          ),
      })
    self.mock_now(datetime.datetime(1969, 1, 1, 0, 30))

    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 3)
    self.assertEqual(key.get().target_size, 3)

  def test_daily_schedule_resize_to_default(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
              schedule=bots_pb2.Schedule(
                  daily=[bots_pb2.DailySchedule(
                      start='0:00',
                      end='1:00',
                      days_of_the_week=xrange(7),
                      target_size=3,
                  )],
              ),
          ),
      })
    self.mock_now(datetime.datetime(1969, 1, 1, 2))

    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.assertEqual(lease_management.MachineLease.query().count(), 1)
    self.assertEqual(key.get().target_size, 1)

  def test_daily_schedule_resize_to_zero(self):
    self.mock_machine_types(
      {
          'machine-type': bots_pb2.MachineType(
              early_release_secs=0,
              lease_duration_secs=1,
              mp_dimensions=['disk_gb:100'],
              name='machine-type',
              target_size=1,
              schedule=bots_pb2.Schedule(
                  daily=[bots_pb2.DailySchedule(
                      start='0:00',
                      end='1:00',
                      days_of_the_week=xrange(7),
                      target_size=0,
                  )],
              ),
          ),
      })
    self.mock_now(datetime.datetime(1969, 1, 1, 0, 30))

    key = lease_management.MachineType(
        id='machine-type',
        early_release_secs=0,
        lease_duration_secs=1,
        mp_dimensions=machine_provider.Dimensions(
            disk_gb=100,
        ),
        target_size=1,
    ).put()

    lease_management._ensure_entities_exist()

    self.failIf(lease_management.MachineLease.query().count())
    self.failIf(key.get().target_size)


class GetTargetSize(TestCase):
  """Tests for lease_management._get_target_size."""

  def test_no_schedules(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule())

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 2), 2)

  def test_wrong_day(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        daily=[bots_pb2.DailySchedule(
            start='1:00',
            end='2:00',
            days_of_the_week=xrange(5),
            target_size=3,
        )],
    ))
    now = datetime.datetime(2012, 1, 1, 1, 2)

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 2, now), 2)

  def test_right_day(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        daily=[bots_pb2.DailySchedule(
            start='1:00',
            end='2:00',
            days_of_the_week=xrange(7),
            target_size=3,
        )],
    ))
    now = datetime.datetime(2012, 1, 1, 1, 2)

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 2, now), 3)

  def test_no_utilization(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        load_based=[bots_pb2.LoadBased(
            maximum_size=5,
            minimum_size=3,
        )],
    ))

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 4), 4)

  def test_utilization(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        load_based=[bots_pb2.LoadBased(
            maximum_size=6,
            minimum_size=2,
        )],
    ))
    lease_management.MachineTypeUtilization(
        id='mt',
        busy=4,
        idle=0,
    ).put()

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 3), 6)

  def test_load_based_fallback(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        daily=[bots_pb2.DailySchedule(
            start='1:00',
            end='2:00',
            days_of_the_week=xrange(5),
            target_size=3,
        )],
        load_based=[bots_pb2.LoadBased(
            maximum_size=6,
            minimum_size=2,
        )],
    ))
    lease_management.MachineTypeUtilization(
        id='mt',
        busy=4,
        idle=0,
    ).put()
    now = datetime.datetime(2012, 1, 1, 1, 2)

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 3, now), 6)

  def test_upper_bound(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        load_based=[bots_pb2.LoadBased(
            maximum_size=4,
            minimum_size=2,
        )],
    ))
    lease_management.MachineTypeUtilization(
        id='mt',
        busy=4,
        idle=0,
    ).put()

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 3), 4)

  def test_drop_dampening(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        load_based=[bots_pb2.LoadBased(
            maximum_size=100,
            minimum_size=1,
        )],
    ))
    lease_management.MachineTypeUtilization(
        id='mt',
        busy=60,
        idle=20,
    ).put()

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 100, 50), 99)

  def test_lower_bound(self):
    config = bots_pb2.MachineType(schedule=bots_pb2.Schedule(
        load_based=[bots_pb2.LoadBased(
            maximum_size=4,
            minimum_size=2,
        )],
    ))
    lease_management.MachineTypeUtilization(
        id='mt',
        busy=0,
        idle=4,
    ).put()

    self.assertEqual(
        lease_management._get_target_size(config.schedule, 'mt', 1, 3), 2)


class ManageLeasedMachineTest(TestCase):
  """Tests for lease_management._manage_leased_machine."""

  def test_creates_bot_id_and_sends_connection_instruction(self):
    def _send_connection_instruction(machine_lease):
      self.assertTrue(machine_lease)
    self.mock(lease_management, '_send_connection_instruction',
              _send_connection_instruction)
    key = lease_management.MachineLease(
        id='machine-lease',
        client_request_id='request-id',
        hostname='hostname',
        lease_id='lease-id',
        leased_indefinitely=True,
        machine_type=lease_management.MachineType(
            id='machine-type',
            target_size=1,
        ).put(),
    ).put()
    lease_management._manage_leased_machine(key.get())
    self.assertTrue(key.get().bot_id)
    self.assertEquals(key.get().bot_id, key.get().hostname)

  def test_checks_for_connection(self):
    def _check_for_connection(machine_lease):
      self.assertTrue(machine_lease)
    def cleanup_bot(*_args, **_kwargs):
      self.fail('cleanup_bot called')
    self.mock(lease_management, '_check_for_connection', _check_for_connection)
    self.mock(lease_management, 'cleanup_bot', cleanup_bot)
    key = lease_management.MachineLease(
        id='machine-lease',
        bot_id='hostname',
        client_request_id='request-id',
        hostname='hostname',
        instruction_ts=utils.utcnow(),
        lease_id='lease-id',
        leased_indefinitely=True,
        machine_type=lease_management.MachineType(
            id='machine-type',
            target_size=1,
        ).put(),
    ).put()
    lease_management._manage_leased_machine(key.get())
    self.assertTrue(key.get().client_request_id)

  def test_cleans_up_bot(self):
    key = lease_management.MachineLease(
        id='machine-lease',
        bot_id='hostname',
        client_request_id='request-id',
        connection_ts=utils.utcnow(),
        hostname='hostname',
        instruction_ts=utils.utcnow(),
        lease_expiration_ts=utils.utcnow(),
        lease_id='lease-id',
        machine_type=lease_management.MachineType(
            id='machine-type',
            target_size=1,
        ).put(),
    ).put()
    lease_management._manage_leased_machine(key.get())
    self.assertFalse(key.get().client_request_id)

  def test_releases(self):
    key = lease_management.MachineLease(
        id='machine-lease',
        bot_id='hostname',
        client_request_id='request-id',
        connection_ts=utils.utcnow(),
        early_release_secs=86400,
        hostname='hostname',
        instruction_ts=utils.utcnow(),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(days=1),
        lease_id='lease-id',
        machine_type=lease_management.MachineType(
            id='machine-type',
            target_size=1,
        ).put(),
    ).put()
    lease_management._manage_leased_machine(key.get())
    self.assertTrue(key.get().termination_task)

  def test_releases_drained_bot(self):
    key = lease_management.MachineLease(
        id='machine-lease',
        bot_id='hostname',
        client_request_id='request-id',
        connection_ts=utils.utcnow(),
        drained=True,
        hostname='hostname',
        instruction_ts=utils.utcnow(),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(days=1),
        lease_id='lease-id',
        machine_type=lease_management.MachineType(
            id='machine-type',
            target_size=1,
        ).put(),
    ).put()
    lease_management._manage_leased_machine(key.get())
    self.assertTrue(key.get().termination_task)

  def test_releases_drained_indefinite_bot(self):
    key = lease_management.MachineLease(
        id='machine-lease',
        bot_id='hostname',
        client_request_id='request-id',
        connection_ts=utils.utcnow(),
        drained=True,
        hostname='hostname',
        instruction_ts=utils.utcnow(),
        leased_indefinitely=True,
        lease_id='lease-id',
        machine_type=lease_management.MachineType(
            id='machine-type',
            target_size=1,
        ).put(),
    ).put()
    lease_management._manage_leased_machine(key.get())
    self.assertTrue(key.get().termination_task)


class ScheduleLeaseManagementTest(TestCase):
  """Tests for lease_management.cron_schedule_lease_management."""

  def test_none(self):
    def enqueue_task(*_args, **_kwargs):
      self.fail('enqueue_task called')
    self.mock(utils, 'enqueue_task', enqueue_task)

    self.assertEqual(0, lease_management.cron_schedule_lease_management())

  def test_manageable(self):
    def enqueue_task(*_args, **kwargs):
      self.assertTrue(kwargs.get('params', {}).get('key'))
    self.mock(utils, 'enqueue_task', enqueue_task)

    lease_management.MachineLease().put()
    self.assertEqual(1, lease_management.cron_schedule_lease_management())

  def test_pending_connection(self):
    def enqueue_task(*_args, **kwargs):
      self.assertTrue(kwargs.get('params', {}).get('key'))
    self.mock(utils, 'enqueue_task', enqueue_task)

    key = lease_management.MachineLease(
        client_request_id='request-id',
    ).put()
    lease_management._log_lease_fulfillment(
        key, 'request-id', 'hostname', 0, True, 'lease-id')
    self.assertEqual(1, lease_management.cron_schedule_lease_management())

  def test_leased(self):
    def enqueue_task(*_args, **_kwargs):
      self.fail('enqueue_task called')
    self.mock(utils, 'enqueue_task', enqueue_task)

    key = lease_management.MachineLease(
        client_request_id='request-id',
    ).put()
    lease_expiration_ts = utils.datetime_to_timestamp(
        utils.utcnow()) / 1000 / 1000 + 3600
    lease_management._log_lease_fulfillment(
        key, 'request-id', 'hostname', lease_expiration_ts, False, 'lease-id')
    lease_management._associate_bot_id(key, 'hostname')
    lease_management._associate_connection_ts(key, utils.utcnow())
    self.assertEqual(0, lease_management.cron_schedule_lease_management())

  def test_expired(self):
    def enqueue_task(*_args, **kwargs):
      self.assertTrue(kwargs.get('params', {}).get('key'))
    self.mock(utils, 'enqueue_task', enqueue_task)

    key = lease_management.MachineLease(
        client_request_id='request-id',
        early_release_secs=3600,
    ).put()
    lease_expiration_ts = utils.datetime_to_timestamp(
        utils.utcnow()) / 1000 / 1000
    lease_management._log_lease_fulfillment(
        key, 'request-id', 'hostname', lease_expiration_ts, False, 'lease-id')
    lease_management._associate_connection_ts(key, utils.utcnow())
    self.assertEqual(1, lease_management.cron_schedule_lease_management())

  def test_leased_indefinitely(self):
    def enqueue_task(*_args, **_kwargs):
      self.fail('enqueue_task called')
    self.mock(utils, 'enqueue_task', enqueue_task)

    key = lease_management.MachineLease(
        client_request_id='request-id',
    ).put()
    lease_management._log_lease_fulfillment(
        key, 'request-id', 'hostname', 0, True, 'lease-id')
    lease_management._associate_bot_id(key, 'hostname')
    lease_management._associate_connection_ts(key, utils.utcnow())
    self.assertEqual(0, lease_management.cron_schedule_lease_management())

  def test_drained(self):
    def enqueue_task(*_args, **kwargs):
      self.assertTrue(kwargs.get('params', {}).get('key'))
    self.mock(utils, 'enqueue_task', enqueue_task)

    key = lease_management.MachineLease(
        client_request_id='request-id',
    ).put()
    lease_management._log_lease_fulfillment(
        key, 'request-id', 'hostname', 0, True, 'lease-id')
    lease_management._associate_connection_ts(key, utils.utcnow())
    lease_management._drain_entity(key)
    self.assertEqual(1, lease_management.cron_schedule_lease_management())


class SendConnectionInstructionTest(TestCase):
  """Tests for lease_management._send_connection_instruction."""

  def test_empty(self):
    def instruct_machine(*_args, **_kwargs):
      return {}
    self.mock(machine_provider, 'instruct_machine', instruct_machine)

    key = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='request-id',
        hostname='bot-id',
    ).put()

    lease_management._send_connection_instruction(key.get())
    self.assertFalse(key.get().instruction_ts)

  def test_ok(self):
    def instruct_machine(*_args, **_kwargs):
      return {'client_request_id': 'request-id'}
    self.mock(machine_provider, 'instruct_machine', instruct_machine)

    key = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='request-id',
        hostname='bot-id',
    ).put()

    lease_management._send_connection_instruction(key.get())
    self.assertTrue(key.get().instruction_ts)

  def test_reclaimed(self):
    def instruct_machine(*_args, **_kwargs):
      return {'client_request_id': 'request-id', 'error': 'ALREADY_RECLAIMED'}
    self.mock(machine_provider, 'instruct_machine', instruct_machine)

    key = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='request-id',
        hostname='bot-id',
    ).put()

    lease_management._send_connection_instruction(key.get())
    self.assertFalse(key.get().bot_id)
    self.assertFalse(key.get().client_request_id)
    self.assertFalse(key.get().hostname)
    self.assertFalse(key.get().instruction_ts)

  def test_error(self):
    def instruct_machine(*_args, **_kwargs):
      return {'client_request_id': 'request-id', 'error': 'error'}
    self.mock(machine_provider, 'instruct_machine', instruct_machine)

    key = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='request-id',
        hostname='bot-id',
    ).put()

    lease_management._send_connection_instruction(key.get())
    self.assertTrue(key.get().bot_id)
    self.assertTrue(key.get().client_request_id)
    self.assertTrue(key.get().hostname)
    self.assertFalse(key.get().instruction_ts)

  def test_race(self):
    key = lease_management.MachineLease(
        bot_id='bot-id',
        client_request_id='request-id',
        hostname='bot-id',
    ).put()

    def instruct_machine(*_args, **_kwargs):
      # Mimic race condition by clearing the MachineLease.
      # In reality this would happen concurrently elsewhere.
      lease_management._clear_lease_request(key, key.get().client_request_id)
      return {'client_request_id': 'request-id'}
    self.mock(machine_provider, 'instruct_machine', instruct_machine)

    lease_management._send_connection_instruction(key.get())
    self.assertFalse(key.get().instruction_ts)


if __name__ == '__main__':
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  unittest.main()
