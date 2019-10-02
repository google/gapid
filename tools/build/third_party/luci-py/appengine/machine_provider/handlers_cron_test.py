#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for handlers_cron.py."""

import datetime
import json
import unittest

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from protorpc.remote import protojson
import webtest

from components import auth_testing
from components import utils
from components.machine_provider import rpc_messages
from test_support import test_case

import handlers_cron
import models


class CanFulfillTest(test_case.TestCase):
  """Tests for handlers_cron.can_fulfill."""

  def test_exact_match(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            disk_gb=100,
            disk_type=rpc_messages.DiskTypes.SSD,
            num_cpus=2,
            os_family=rpc_messages.OSFamily.LINUX,
            snapshot='snapshot',
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            disk_gb=100,
            disk_type=rpc_messages.DiskTypes.SSD,
            num_cpus=2,
            os_family=rpc_messages.OSFamily.LINUX,
            snapshot='snapshot',
        ),
    )

    self.assertTrue(handlers_cron.can_fulfill(entry, request))

  def test_subset_match(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            disk_gb=100,
            num_cpus=2,
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            disk_gb=100,
            disk_type=rpc_messages.DiskTypes.HDD,
            hostname='fake-host',
            memory_gb=8.0,
            num_cpus=2,
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    )

    self.assertTrue(handlers_cron.can_fulfill(entry, request))

  def test_mismatch(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            disk_gb=100,
            num_cpus=4,
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            disk_gb=100,
            hostname='fake-host',
            memory_gb=8.0,
            num_cpus=2,
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    )

    self.assertFalse(handlers_cron.can_fulfill(entry, request))

  def test_multi_value_empty_subset_match(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=[],
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2'],
        ),
    )

    self.assertTrue(handlers_cron.can_fulfill(entry, request))

  def test_multi_value_subset_match(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1'],
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2'],
        ),
    )

    self.assertTrue(handlers_cron.can_fulfill(entry, request))

  def test_multi_value_exact_match(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2'],
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2'],
        ),
    )

    self.assertTrue(handlers_cron.can_fulfill(entry, request))

  def test_multi_value_mismatch(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label3'],
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2'],
        ),
    )

    self.assertFalse(handlers_cron.can_fulfill(entry, request))

  def test_multi_value_superset_mismatch(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2', 'label3'],
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2'],
        ),
    )

    self.assertFalse(handlers_cron.can_fulfill(entry, request))

  def test_multi_value_superset_empty_mismatch(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1', 'label2', 'label3'],
        ),
    )
    entry = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=[],
        ),
    )

    self.assertFalse(handlers_cron.can_fulfill(entry, request))


class LeaseMachineTest(test_case.TestCase):
  """Tests for handlers_cron.lease_machine."""

  def test_leased(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    machine_key = models.CatalogMachineEntry(
        id='machine-id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            machine_service_account='service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='request-id',
    )
    lease_request_key = models.LeaseRequest(
        id='lease-id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='client-request-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()

    handlers_cron.lease_machine(machine_key, lease_request_key.get())
    self.assertEqual(lease_request_key.get().machine_id, machine_key.id())
    self.assertEqual(
        lease_request_key.get().response.state,
        rpc_messages.LeaseRequestState.FULFILLED,
    )
    self.assertEqual(machine_key.get().lease_id, lease_request_key.id())
    self.assertEqual(
        machine_key.get().state,
        models.CatalogMachineEntryStates.LEASED,
    )

  def test_cant_fulfill(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    machine_key = models.CatalogMachineEntry(
        id='machine-id',
        dimensions=rpc_messages.Dimensions(
            snapshot_labels=['label1'],
        ),
        policies=rpc_messages.Policies(
            machine_service_account='service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='request-id',
    )
    lease_request_key = models.LeaseRequest(
        id='lease-id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='client-request-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()

    handlers_cron.lease_machine(machine_key, lease_request_key.get())
    self.assertFalse(lease_request_key.get().machine_id)
    self.assertEqual(
        lease_request_key.get().response.state,
        rpc_messages.LeaseRequestState.UNTRIAGED,
    )
    self.assertFalse(machine_key.get().lease_id)
    self.assertEqual(
        machine_key.get().state,
        models.CatalogMachineEntryStates.AVAILABLE,
    )

  def test_machine_unavailable(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    machine_key = models.CatalogMachineEntry(
        id='machine-id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            machine_service_account='service-account',
        ),
        state=models.CatalogMachineEntryStates.LEASED,
    ).put()
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='request-id',
    )
    lease_request_key = models.LeaseRequest(
        id='lease-id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='client-request-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()

    handlers_cron.lease_machine(machine_key, lease_request_key.get())
    self.assertFalse(lease_request_key.get().machine_id)
    self.assertEqual(
        lease_request_key.get().response.state,
        rpc_messages.LeaseRequestState.UNTRIAGED,
    )
    self.assertFalse(machine_key.get().lease_id)
    self.assertEqual(
        machine_key.get().state,
        models.CatalogMachineEntryStates.LEASED,
    )

  def test_request_triaged(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    machine_key = models.CatalogMachineEntry(
        id='machine-id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            machine_service_account='service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='request-id',
    )
    lease_request_key = models.LeaseRequest(
        id='lease-id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='client-request-id',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()

    handlers_cron.lease_machine(machine_key, lease_request_key.get())
    self.assertFalse(lease_request_key.get().machine_id)
    self.assertEqual(
        lease_request_key.get().response.state,
        rpc_messages.LeaseRequestState.FULFILLED,
    )
    self.assertFalse(machine_key.get().lease_id)
    self.assertEqual(
        machine_key.get().state,
        models.CatalogMachineEntryStates.AVAILABLE,
    )

  def test_leased_task_failed(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: False)

    machine_key = models.CatalogMachineEntry(
        id='machine-id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            machine_service_account='service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='request-id',
    )
    lease_request_key = models.LeaseRequest(
        id='lease-id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='client-request-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()

    with self.assertRaises(handlers_cron.TaskEnqueuingError):
      handlers_cron.lease_machine(machine_key, lease_request_key.get())
    self.assertFalse(lease_request_key.get().machine_id)
    self.assertEqual(
        lease_request_key.get().response.state,
        rpc_messages.LeaseRequestState.UNTRIAGED,
    )
    self.assertFalse(machine_key.get().lease_id)
    self.assertEqual(
        machine_key.get().state,
        models.CatalogMachineEntryStates.AVAILABLE,
    )


class LeaseRequestProcessorTest(test_case.TestCase):
  """Tests for handlers_cron.LeaseRequestProcessor."""

  def setUp(self):
    super(LeaseRequestProcessorTest, self).setUp()
    app = handlers_cron.create_cron_app()
    self.app = webtest.TestApp(app)
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

  def test_one_request_one_matching_machine_entry_duration(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
            snapshot_labels=['label1', 'label2'],
        ),
        duration=1,
        request_id='fake-id',
    )
    key = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
        os_family=rpc_messages.OSFamily.LINUX,
        snapshot_labels=['label1', 'label2', 'label3', 'label4'],
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()

    self.app.get(
        '/internal/cron/process-lease-requests',
        headers={'X-AppEngine-Cron': 'true'},
    )
    self.assertTrue(key.get().response.lease_expiration_ts)

  def test_one_request_one_matching_machine_entry_lease_expiration_ts(self):
    ts = int(utils.time_time())
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=ts,
        request_id='fake-id',
    )
    key = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
        os_family=rpc_messages.OSFamily.LINUX,
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()

    self.app.get(
        '/internal/cron/process-lease-requests',
        headers={'X-AppEngine-Cron': 'true'},
    )
    self.assertEqual(key.get().response.lease_expiration_ts, ts)

  def test_one_request_one_matching_machine_entry_leased_indefinitely(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        indefinite=True,
        request_id='fake-id',
    )
    key = models.LeaseRequest(
        deduplication_checksum=
          models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
        os_family=rpc_messages.OSFamily.LINUX,
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
        state=models.CatalogMachineEntryStates.AVAILABLE,
    ).put()

    self.app.get(
        '/internal/cron/process-lease-requests',
        headers={'X-AppEngine-Cron': 'true'},
    )
    self.assertTrue(key.get().response.leased_indefinitely)

  def test_one_request_inactive(self):
    now = utils.utcnow()
    def utcnow():
      return now + datetime.timedelta(days=10)
    self.mock(handlers_cron.utils, 'utcnow', utcnow)
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        indefinite=True,
        request_id='fake-id',
    )
    key = models.LeaseRequest(
        deduplication_checksum=
          models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()

    self.app.get(
        '/internal/cron/process-lease-requests',
        headers={'X-AppEngine-Cron': 'true'},
    )
    self.assertEqual(
        key.get().response.state, rpc_messages.LeaseRequestState.DENIED)


class MachineReclamationProcessorTest(test_case.TestCase):
  """Tests for handlers_cron.MachineReclamationProcessor."""

  def setUp(self):
    super(MachineReclamationProcessorTest, self).setUp()
    app = handlers_cron.create_cron_app()
    self.app = webtest.TestApp(app)
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

  def test_reclaim_indefinite(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        indefinite=True,
        request_id='fake-id',
    )
    lease = models.LeaseRequest(
        deduplication_checksum=
          models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    )
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
        os_family=rpc_messages.OSFamily.LINUX,
    )
    machine = models.CatalogMachineEntry(
        dimensions=dimensions,
        key=models.CatalogMachineEntry.generate_key(dimensions),
        lease_id=lease.key.id(),
        lease_expiration_ts=datetime.datetime.utcfromtimestamp(1),
        leased_indefinitely=True,
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
        state=models.CatalogMachineEntryStates.LEASED,
    ).put()
    lease.machine_id = machine.id()
    lease.put()

    self.app.get(
        '/internal/cron/process-machine-reclamations',
        headers={'X-AppEngine-Cron': 'true'},
    )
    # Assert extended.
    self.assertEqual(lease.key.get().response.hostname, 'fake-host')
    self.assertEqual(
        machine.get().lease_expiration_ts,
        datetime.datetime.utcfromtimestamp(1) + datetime.timedelta(days=10000),
    )
    self.assertTrue(machine.get().leased_indefinitely)

  def test_reclaim_immediately(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=0,
        request_id='fake-id',
    )
    lease = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    )
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
        os_family=rpc_messages.OSFamily.LINUX,
    )
    machine = models.CatalogMachineEntry(
        dimensions=dimensions,
        key=models.CatalogMachineEntry.generate_key(dimensions),
        lease_id=lease.key.id(),
        lease_expiration_ts=datetime.datetime.utcfromtimestamp(1),
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
        state=models.CatalogMachineEntryStates.LEASED,
    ).put()
    lease.machine_id = machine.id()
    lease.put()

    self.app.get(
        '/internal/cron/process-machine-reclamations',
        headers={'X-AppEngine-Cron': 'true'},
    )
    # Assert reclaimed.
    self.assertFalse(lease.key.get().response.hostname)
    self.assertEqual(
        machine.get().lease_expiration_ts,
        datetime.datetime.utcfromtimestamp(1),
    )
    self.assertFalse(machine.get().leased_indefinitely)


class ReclaimMachineTest(test_case.TestCase):
  """Tests for handlers_cron.reclaim_machine."""

  def test_reclaimed(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_request_key = models.LeaseRequest(
        id='id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=utils.utcnow(),
        lease_id=lease_request_key.id(),
        policies=rpc_messages.Policies(
            backend_attributes=[
                rpc_messages.KeyValuePair(
                    key='key',
                    value='value',
                ),
            ],
            machine_service_account='fake-service-account',
        ),
    ).put()

    handlers_cron.reclaim_machine(machine_key, utils.utcnow())
    self.assertFalse(lease_request_key.get().response.hostname)

  def test_not_found(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_request_key = models.LeaseRequest(
        id='id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    ).put()
    machine_key = ndb.Key(models.CatalogMachineEntry, 'fake-key')

    handlers_cron.reclaim_machine(machine_key, utils.utcnow())
    self.assertTrue(lease_request_key.get().response.hostname)

  def test_no_expiration_ts(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_request_key = models.LeaseRequest(
        id='id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_id=lease_request_key.id(),
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
    ).put()

    handlers_cron.reclaim_machine(machine_key, utils.utcnow())
    self.assertTrue(lease_request_key.get().response.hostname)

  def test_not_expired(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: True)

    ts = utils.utcnow()
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_request_key = models.LeaseRequest(
        id='id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=utils.utcnow(),
        lease_id=lease_request_key.id(),
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
    ).put()

    handlers_cron.reclaim_machine(machine_key, ts)
    self.assertTrue(lease_request_key.get().response.hostname)

  def test_enqueue_failed(self):
    self.mock(utils, 'enqueue_task', lambda *args, **kwargs: False)

    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_request_key = models.LeaseRequest(
        id='id',
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            hostname='fake-host',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=utils.utcnow(),
        lease_id=lease_request_key.id(),
        policies=rpc_messages.Policies(
            machine_service_account='fake-service-account',
        ),
    ).put()

    with self.assertRaises(handlers_cron.TaskEnqueuingError):
      handlers_cron.reclaim_machine(machine_key, utils.utcnow())
    self.assertTrue(lease_request_key.get().response.hostname)


class ReleaseLeaseTest(test_case.TestCase):
  """Tests for handlers_cron.release_lease."""

  def test_released(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_key = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        machine_id='id',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        released=True,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    ).put()

    handlers_cron.release_lease(lease_key)
    self.assertFalse(lease_key.get().released)
    self.assertEqual(
        lease_key.get().response.lease_expiration_ts,
        utils.datetime_to_timestamp(
            machine_key.get().lease_expiration_ts) / 1000 / 1000,
    )

  def test_lease_not_found(self):
    machine_key = models.CatalogMachineEntry(
        id='id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    ).put()

    handlers_cron.release_lease(ndb.Key(models.LeaseRequest, 'fake-request'))
    self.assertFalse(machine_key.get().lease_expiration_ts)

  def test_not_released(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_key = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        machine_id='id',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        released=False,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    ).put()

    handlers_cron.release_lease(lease_key)
    self.assertFalse(lease_key.get().response.lease_expiration_ts)
    self.assertFalse(lease_key.get().released)
    self.assertFalse(machine_key.get().lease_expiration_ts)

  def test_no_machine_id(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_key = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        released=True,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='id',
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    ).put()

    handlers_cron.release_lease(lease_key)
    self.assertFalse(lease_key.get().response.lease_expiration_ts)
    self.assertFalse(lease_key.get().released)
    self.assertFalse(machine_key.get().lease_expiration_ts)

  def test_machine_doesnt_exist(self):
    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    lease_key = models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        machine_id='id',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        released=True,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
        ),
    ).put()

    handlers_cron.release_lease(lease_key)
    self.assertFalse(lease_key.get().response.lease_expiration_ts)
    self.assertFalse(lease_key.get().released)


class LeaseReleaseProcessorTest(test_case.TestCase):
  """Tests for handlers_cron.LeaseReleaseProcessor."""

  def setUp(self):
    super(LeaseReleaseProcessorTest, self).setUp()
    app = handlers_cron.create_cron_app()
    self.app = webtest.TestApp(app)

  def test_releases(self):
    self.mock(handlers_cron, 'release_lease', lambda *args, **kwargs: True)

    request = rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='fake-id',
    )
    models.LeaseRequest(
        deduplication_checksum=
            models.LeaseRequest.compute_deduplication_checksum(request),
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        released=True,
        request=request,
        response=rpc_messages.LeaseResponse(
            client_request_id='fake-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()

    self.app.get(
        '/internal/cron/process-lease-releases',
        headers={'X-AppEngine-Cron': 'true'},
    )


if __name__ == '__main__':
  unittest.main()
