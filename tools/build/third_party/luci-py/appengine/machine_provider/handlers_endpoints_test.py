#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Unit tests for handlers_endpoints.py."""

import datetime
import json
import unittest

import test_env
test_env.setup_test_env()

import endpoints

from google.appengine import runtime
from google.appengine.ext import ndb

from protorpc.remote import protojson
import webtest

from components import auth_testing
from components import utils
from components.machine_provider import rpc_messages
from test_support import test_case

import acl
import handlers_endpoints
import models


def rpc_to_json(rpc_message):
  """Converts the given RPC message to a POSTable JSON dict.

  Args:
    rpc_message: A protorpc.message.Message instance.

  Returns:
    A string representing a JSON dict.
  """
  return json.loads(protojson.encode_message(rpc_message))


def jsonish_dict_to_rpc(dictionary, rpc_message_type):
  """Converts the given dict to the specified RPC message type.

  Args:
    dictionary: A dict instance containing only values which can be
      encoded as JSON.
    rpc_message_type: A type inheriting from protorpc.message.Message.

  Returns:
    An object of type rpc_message_type.
  """
  return protojson.decode_message(rpc_message_type, json.dumps(dictionary))


class CatalogTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.CatalogEndpoints."""
  api_service_cls = handlers_endpoints.CatalogEndpoints

  def setUp(self):
    super(CatalogTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def mock_get_current_backend(self, backend=rpc_messages.Backend.DUMMY):
    self.mock(acl, 'get_current_backend', lambda *args, **kwargs: backend)

  def test_get(self):
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry._generate_key('DUMMY', 'fake-host'),
        dimensions=rpc_messages.Dimensions(hostname='fake-host'),
        lease_expiration_ts=utils.utcnow(),
    ).put()
    request = rpc_to_json(rpc_messages.CatalogMachineRetrievalRequest(
        hostname='fake-host',
    ))

    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('get', request).json,
        rpc_messages.CatalogMachineRetrievalResponse,
    )
    self.assertEqual(response.dimensions.hostname, 'fake-host')
    self.assertTrue(response.lease_expiration_ts)

  def test_get_mismatched_backend(self):
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry._generate_key('DUMMY', 'fake-host'),
        dimensions=rpc_messages.Dimensions(hostname='fake-host'),
    ).put()
    request = rpc_to_json(rpc_messages.CatalogMachineRetrievalRequest(
        backend=rpc_messages.Backend.GCE,
        hostname='fake-host',
    ))

    self.mock_get_current_backend()

    jsonish_dict_to_rpc(
        self.call_api('get', request, status=403).json,
        rpc_messages.CatalogMachineRetrievalResponse,
    )

  def test_get_backend_unspecified_by_admin(self):
    self.mock(acl, 'is_catalog_admin', lambda *args, **kwargs: True)

    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry._generate_key('DUMMY', 'fake-host'),
        dimensions=rpc_messages.Dimensions(hostname='fake-host'),
    ).put()
    request = rpc_to_json(rpc_messages.CatalogMachineRetrievalRequest(
        hostname='fake-host',
    ))

    jsonish_dict_to_rpc(
        self.call_api('get', request, status=400).json,
        rpc_messages.CatalogMachineRetrievalResponse,
    )

  def test_get_not_found(self):
    request = rpc_to_json(rpc_messages.CatalogMachineRetrievalRequest(
        hostname='fake-host',
    ))

    self.mock_get_current_backend()

    jsonish_dict_to_rpc(
        self.call_api('get', request, status=404).json,
        rpc_messages.CatalogMachineRetrievalResponse,
    )

  def test_add(self):
    request = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('add_machine', request).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertFalse(response.error)

  def test_mismatched_backend(self):
    request = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.GCE,
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('add_machine', request).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertEqual(
        response.error,
        rpc_messages.CatalogManipulationRequestError.MISMATCHED_BACKEND,
    )

  def test_add_backend_unspecified_by_admin(self):
    self.mock(acl, 'is_catalog_admin', lambda *args, **kwargs: True)

    request = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_topic='fake-topic',
        ),
    ))

    response = jsonish_dict_to_rpc(
        self.call_api('add_machine', request).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertEqual(
        response.error,
        rpc_messages.CatalogManipulationRequestError.UNSPECIFIED_BACKEND,
    )

  def test_add_no_hostname(self):
    request = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('add_machine', request).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertEqual(
      response.error,
      rpc_messages.CatalogManipulationRequestError.UNSPECIFIED_HOSTNAME,
    )

  def test_add_duplicate(self):
    request_1 = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    request_2 = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response_1 = jsonish_dict_to_rpc(
        self.call_api('add_machine', request_1).json,
        rpc_messages.CatalogManipulationResponse,
    )
    response_2 = jsonish_dict_to_rpc(
        self.call_api('add_machine', request_2).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertFalse(response_1.error)
    self.assertEqual(
        response_2.error,
        rpc_messages.CatalogManipulationRequestError.HOSTNAME_REUSE,
    )

  def test_add_batch_empty(self):
    request = rpc_to_json(rpc_messages.CatalogMachineBatchAdditionRequest())
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('add_machines', request).json,
        rpc_messages.CatalogBatchManipulationResponse,
    )
    self.assertFalse(response.responses)

  def test_add_batch(self):
    request = rpc_to_json(rpc_messages.CatalogMachineBatchAdditionRequest(
        requests=[
            rpc_messages.CatalogMachineAdditionRequest(
                dimensions=rpc_messages.Dimensions(
                    hostname='fake-host-1',
                    os_family=rpc_messages.OSFamily.LINUX,
                ),
                policies=rpc_messages.Policies(
                    backend_project='fake-project',
                    backend_topic='fake-topic',
                ),
            ),
            rpc_messages.CatalogMachineAdditionRequest(
                dimensions=rpc_messages.Dimensions(
                    hostname='fake-host-2',
                    os_family=rpc_messages.OSFamily.WINDOWS,
                ),
                policies=rpc_messages.Policies(
                    backend_project='fake-project',
                    backend_topic='fake-topic',
                ),
            ),
            rpc_messages.CatalogMachineAdditionRequest(
                dimensions=rpc_messages.Dimensions(
                    hostname='fake-host-1',
                    os_family=rpc_messages.OSFamily.OSX,
                ),
                policies=rpc_messages.Policies(
                    backend_project='fake-project',
                    backend_topic='fake-topic',
                ),
            ),
        ],
    ))
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('add_machines', request).json,
        rpc_messages.CatalogBatchManipulationResponse,
    )
    self.assertEqual(len(response.responses), 3)
    self.assertFalse(response.responses[0].error)
    self.assertFalse(response.responses[1].error)
    self.assertEqual(
        response.responses[2].error,
        rpc_messages.CatalogManipulationRequestError.HOSTNAME_REUSE,
    )

  def test_add_batch_error(self):
    request = rpc_to_json(rpc_messages.CatalogMachineBatchAdditionRequest(
        requests=[
            rpc_messages.CatalogMachineAdditionRequest(
                dimensions=rpc_messages.Dimensions(
                    backend=rpc_messages.Backend.GCE,
                    hostname='fake-host-1',
                    os_family=rpc_messages.OSFamily.LINUX,
                ),
                policies=rpc_messages.Policies(
                    backend_project='fake-project',
                    backend_topic='fake-topic',
                ),
            ),
        ],
    ))
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('add_machines', request).json,
        rpc_messages.CatalogBatchManipulationResponse,
    )
    self.assertEqual(len(response.responses), 1)
    self.assertEqual(
        response.responses[0].error,
        rpc_messages.CatalogManipulationRequestError.MISMATCHED_BACKEND,
    )

  def test_delete(self):
    request_1 = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    request_2 = rpc_to_json(rpc_messages.CatalogMachineDeletionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    ))
    request_3 = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.WINDOWS,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response_1 = jsonish_dict_to_rpc(
        self.call_api('add_machine', request_1).json,
        rpc_messages.CatalogManipulationResponse,
    )
    response_2 = jsonish_dict_to_rpc(
        self.call_api('delete_machine', request_2).json,
        rpc_messages.CatalogManipulationResponse,
    )
    response_3 = jsonish_dict_to_rpc(
        self.call_api('add_machine', request_3).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertFalse(response_1.error)
    self.assertFalse(response_2.error)
    self.assertFalse(response_3.error)

  def test_delete_error(self):
    request = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.GCE,
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('delete_machine', request).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertEqual(
        response.error,
        rpc_messages.CatalogManipulationRequestError.MISMATCHED_BACKEND,
    )

  def test_delete_leased(self):
    request = rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
            hostname='fake-host',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    )
    key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(request.dimensions),
        dimensions=request.dimensions,
        lease_id='lease-id',
    ).put()
    request = rpc_to_json(request)
    self.mock_get_current_backend()

    response = jsonish_dict_to_rpc(
        self.call_api('delete_machine', request).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertEqual(
        response.error,
        rpc_messages.CatalogManipulationRequestError.LEASED,
    )
    self.assertTrue(key.get())

  def test_delete_invalid(self):
    request_1 = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host-1',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    request_2 = rpc_to_json(rpc_messages.CatalogMachineDeletionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host-2',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
    ))
    request_3 = rpc_to_json(rpc_messages.CatalogMachineAdditionRequest(
        dimensions=rpc_messages.Dimensions(
            hostname='fake-host-1',
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        policies=rpc_messages.Policies(
            backend_project='fake-project',
            backend_topic='fake-topic',
        ),
    ))
    self.mock_get_current_backend()

    response_1 = jsonish_dict_to_rpc(
        self.call_api('add_machine', request_1).json,
        rpc_messages.CatalogManipulationResponse,
    )
    response_2 = jsonish_dict_to_rpc(
        self.call_api('delete_machine', request_2).json,
        rpc_messages.CatalogManipulationResponse,
    )
    response_3 = jsonish_dict_to_rpc(
        self.call_api('add_machine', request_3).json,
        rpc_messages.CatalogManipulationResponse,
    )
    self.assertFalse(response_1.error)
    self.assertEqual(
        response_2.error,
        rpc_messages.CatalogManipulationRequestError.ENTRY_NOT_FOUND,
    )
    self.assertEqual(
        response_3.error,
        rpc_messages.CatalogManipulationRequestError.HOSTNAME_REUSE,
    )


class MachineTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.MachineEndpoints."""
  api_service_cls = handlers_endpoints.MachineEndpoints

  def setUp(self):
    super(MachineTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def test_update_instruction_state_not_found(self):
    machine_key = ndb.Key(models.CatalogMachineEntry, 'fake-machine')

    with self.assertRaises(endpoints.NotFoundException):
      handlers_endpoints.MachineEndpoints._update_instruction_state(
          machine_key, models.InstructionStates.EXECUTED)

    self.failIf(machine_key.get())

  def test_update_instruction_state_no_instruction(self):
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
    ).put()

    handlers_endpoints.MachineEndpoints._update_instruction_state(
        machine_key, models.InstructionStates.EXECUTED)

    self.failIf(machine_key.get().instruction)

  def test_update_instruction_state_already_updated(self):
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        instruction=models.Instruction(
            state=models.InstructionStates.EXECUTED
        ),
    ).put()

    handlers_endpoints.MachineEndpoints._update_instruction_state(
        machine_key, models.InstructionStates.EXECUTED)

    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.EXECUTED)

  def test_update_instruction_state_invalid_new_state(self):
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        instruction=models.Instruction(
            state=models.InstructionStates.EXECUTED
        ),
    ).put()

    handlers_endpoints.MachineEndpoints._update_instruction_state(
        machine_key, models.InstructionStates.PENDING)

    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.EXECUTED)

  def test_update_instruction_state_invalid_transition(self):
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        instruction=models.Instruction(
            state=models.InstructionStates.EXECUTED
        ),
    ).put()

    handlers_endpoints.MachineEndpoints._update_instruction_state(
        machine_key, models.InstructionStates.RECEIVED)

    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.EXECUTED)

  def test_update_instruction_state(self):
    machine_key = models.CatalogMachineEntry(
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        instruction=models.Instruction(
            state=models.InstructionStates.PENDING
        ),
    ).put()

    handlers_endpoints.MachineEndpoints._update_instruction_state(
        machine_key, models.InstructionStates.RECEIVED)

    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.RECEIVED)

  def test_poll_anonymous(self):
    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    with self.assertRaises(webtest.app.AppError):
      jsonish_dict_to_rpc(
          self.call_api('poll', request).json,
          rpc_messages.PollResponse,
      )

  def test_poll_backend_omitted(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    with self.assertRaises(webtest.app.AppError):
      jsonish_dict_to_rpc(
          self.call_api('poll', request).json,
          rpc_messages.PollResponse,
      )

  def test_poll_entry_not_found(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    with self.assertRaises(webtest.app.AppError):
      jsonish_dict_to_rpc(
          self.call_api('poll', request).json,
          rpc_messages.PollResponse,
      )

  def test_poll_unauthorized(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    with self.assertRaises(webtest.app.AppError):
      jsonish_dict_to_rpc(
          self.call_api('poll', request).json,
          rpc_messages.PollResponse,
      )

  def test_poll_not_leased(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.failIf(response.instruction)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.PENDING)

  def test_poll_no_instruction(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.failIf(response.instruction)
    self.failIf(machine_key.get().instruction)

  def test_poll_expired(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow(),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.failIf(response.instruction)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.PENDING)

  def test_poll_no_lease(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.failIf(response.instruction)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.PENDING)

  def test_poll_no_lease_released(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        released=True,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.failIf(response.instruction)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.PENDING)

  def test_poll_implied_backend(self):
    def is_group_member(group):
      return group == 'machine-provider-dummy-backend'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.assertEqual(response.instruction.swarming_server, 'example.com')
    self.assertEqual(response.state, models.InstructionStates.PENDING)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.PENDING)

  def test_poll(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.PollRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.PENDING,
        ),
        lease_expiration_ts=utils.utcnow() + datetime.timedelta(hours=24),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()
    models.LeaseRequest(
        id='fake-id',
        deduplication_checksum='checksum',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
    ).put()

    response = jsonish_dict_to_rpc(
        self.call_api('poll', request).json,
        rpc_messages.PollResponse,
    )
    self.assertEqual(response.instruction.swarming_server, 'example.com')
    self.assertEqual(response.state, models.InstructionStates.PENDING)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.RECEIVED)

  def test_ack_entry_not_found(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.AckRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))

    with self.assertRaises(webtest.app.AppError):
      self.call_api('ack', request)

  def test_ack_unauthorized(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.AckRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.RECEIVED,
        ),
        lease_id='fake-id',
        policies=rpc_messages.Policies(),
    ).put()

    with self.assertRaises(webtest.app.AppError):
      self.call_api('ack', request)

  def test_ack_no_instruction(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.AckRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()

    with self.assertRaises(webtest.app.AppError):
      self.call_api('ack', request)

  def test_ack(self):
    auth_testing.mock_get_current_identity(self)

    request = rpc_to_json(rpc_messages.AckRequest(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    ))
    dimensions = rpc_messages.Dimensions(
        backend=rpc_messages.Backend.DUMMY,
        hostname='fake-host',
    )
    machine_key = models.CatalogMachineEntry(
        key=models.CatalogMachineEntry.generate_key(dimensions),
        dimensions=dimensions,
        instruction=models.Instruction(
            instruction=rpc_messages.Instruction(swarming_server='example.com'),
            state=models.InstructionStates.RECEIVED,
        ),
        lease_id='fake-id',
        policies=rpc_messages.Policies(
            machine_service_account=auth_testing.DEFAULT_MOCKED_IDENTITY.name,
        ),
    ).put()

    self.call_api('ack', request)
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.EXECUTED)


class MachineProviderReleaseTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.MachineProviderEndpoints.release."""
  api_service_cls = handlers_endpoints.MachineProviderEndpoints

  def setUp(self):
    super(MachineProviderReleaseTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def test_release(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    self.mock(
        handlers_endpoints.MachineProviderEndpoints,
        '_release',
        lambda *args, **kwargs: None,
    )

    request = rpc_to_json(rpc_messages.LeaseReleaseRequest(
        request_id='request-id',
    ))

    response = jsonish_dict_to_rpc(
        self.call_api('release', request).json,
        rpc_messages.LeaseReleaseResponse,
    )
    self.assertEqual(response.client_request_id, 'request-id')
    self.assertFalse(response.error)


class MachineProviderBatchedReleaseTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.MachineProviderEndpoints.batched_release."""
  api_service_cls = handlers_endpoints.MachineProviderEndpoints

  def setUp(self):
    super(MachineProviderBatchedReleaseTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def test_batch(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    ts = utils.utcnow()
    self.mock(utils, 'utcnow', lambda *args, **kwargs: ts)

    release_requests = rpc_to_json(rpc_messages.BatchedLeaseReleaseRequest(
        requests=[
            rpc_messages.LeaseReleaseRequest(
                request_id='request-id',
            ),
        ],
    ))

    release_responses = jsonish_dict_to_rpc(
        self.call_api('batched_release', release_requests).json,
        rpc_messages.BatchedLeaseReleaseResponse,
    )
    self.assertEqual(len(release_responses.responses), 1)
    self.assertEqual(
        release_responses.responses[0].client_request_id, 'request-id')
    self.assertEqual(
        release_responses.responses[0].error,
        rpc_messages.LeaseReleaseRequestError.NOT_FOUND,
    )

  def test_deadline_exceeded(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    class utcnow(object):
      def __init__(self, init_ts):
        self.last_ts = init_ts
      def __call__(self, *args, **kwargs):
        self.last_ts = self.last_ts + datetime.timedelta(seconds=60)
        return self.last_ts
    self.mock(utils, 'utcnow', utcnow(utils.utcnow()))

    release_requests = rpc_to_json(rpc_messages.BatchedLeaseReleaseRequest(
        requests=[
            rpc_messages.LeaseReleaseRequest(
                request_id='request-id',
            ),
        ],
    ))

    release_responses = jsonish_dict_to_rpc(
        self.call_api('batched_release', release_requests).json,
        rpc_messages.BatchedLeaseReleaseResponse,
    )
    self.assertEqual(len(release_responses.responses), 1)
    self.assertEqual(
        release_responses.responses[0].client_request_id, 'request-id')
    self.assertEqual(
        release_responses.responses[0].error,
        rpc_messages.LeaseReleaseRequestError.DEADLINE_EXCEEDED,
    )

  def test_exception(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    ts = utils.utcnow()
    self.mock(utils, 'utcnow', lambda *args, **kwargs: ts)

    def _release(*_args, **_kwargs):
      raise runtime.apiproxy_errors.CancelledError
    self.mock(handlers_endpoints.MachineProviderEndpoints, '_release', _release)

    release_requests = rpc_to_json(rpc_messages.BatchedLeaseReleaseRequest(
        requests=[
            rpc_messages.LeaseReleaseRequest(
                request_id='request-id',
            ),
        ],
    ))

    release_responses = jsonish_dict_to_rpc(
        self.call_api('batched_release', release_requests).json,
        rpc_messages.BatchedLeaseReleaseResponse,
    )
    self.assertEqual(len(release_responses.responses), 1)
    self.assertEqual(
        release_responses.responses[0].client_request_id, 'request-id')
    self.assertEqual(
        release_responses.responses[0].error,
        rpc_messages.LeaseReleaseRequestError.TRANSIENT_ERROR,
    )


class MachineProviderBatchedLeaseTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.MachineProviderEndpoints.batched_lease."""
  api_service_cls = handlers_endpoints.MachineProviderEndpoints

  def setUp(self):
    super(MachineProviderBatchedLeaseTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def test_batch(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    ts = utils.utcnow()
    self.mock(utils, 'utcnow', lambda *args, **kwargs: ts)

    lease_requests = rpc_to_json(rpc_messages.BatchedLeaseRequest(requests=[
        rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(
                os_family=rpc_messages.OSFamily.LINUX,
            ),
            duration=1,
            request_id='request-id',
        ),
    ]))

    lease_responses = jsonish_dict_to_rpc(
        self.call_api('batched_lease', lease_requests).json,
        rpc_messages.BatchedLeaseResponse,
    )
    self.assertEqual(len(lease_responses.responses), 1)
    self.assertEqual(
        lease_responses.responses[0].client_request_id, 'request-id')
    self.assertFalse(lease_responses.responses[0].error)

  def test_deadline_exceeded(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    class utcnow(object):
      def __init__(self, init_ts):
        self.last_ts = init_ts
      def __call__(self, *args, **kwargs):
        self.last_ts = self.last_ts + datetime.timedelta(seconds=60)
        return self.last_ts
    self.mock(utils, 'utcnow', utcnow(utils.utcnow()))

    lease_requests = rpc_to_json(rpc_messages.BatchedLeaseRequest(requests=[
        rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(
                os_family=rpc_messages.OSFamily.LINUX,
            ),
            duration=1,
            request_id='request-id',
        ),
    ]))

    lease_responses = jsonish_dict_to_rpc(
        self.call_api('batched_lease', lease_requests).json,
        rpc_messages.BatchedLeaseResponse,
    )
    self.assertEqual(len(lease_responses.responses), 1)
    self.assertEqual(
        lease_responses.responses[0].client_request_id, 'request-id')
    self.assertEqual(
        lease_responses.responses[0].error,
        rpc_messages.LeaseRequestError.DEADLINE_EXCEEDED,
    )

  def test_exception(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    ts = utils.utcnow()
    self.mock(utils, 'utcnow', lambda *args, **kwargs: ts)

    def _lease(*_args, **_kwargs):
      raise runtime.apiproxy_errors.CancelledError
    self.mock(handlers_endpoints.MachineProviderEndpoints, '_lease', _lease)

    lease_requests = rpc_to_json(rpc_messages.BatchedLeaseRequest(requests=[
        rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(
                os_family=rpc_messages.OSFamily.LINUX,
            ),
            duration=1,
            request_id='request-id',
        ),
    ]))

    lease_responses = jsonish_dict_to_rpc(
        self.call_api('batched_lease', lease_requests).json,
        rpc_messages.BatchedLeaseResponse,
    )
    self.assertEqual(len(lease_responses.responses), 1)
    self.assertEqual(
        lease_responses.responses[0].client_request_id, 'request-id')
    self.assertEqual(
        lease_responses.responses[0].error,
        rpc_messages.LeaseRequestError.TRANSIENT_ERROR,
    )


class MachineProviderLeaseTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.MachineProviderEndpoints.lease."""
  api_service_cls = handlers_endpoints.MachineProviderEndpoints

  def setUp(self):
    super(MachineProviderLeaseTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def test_lease_duration(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertFalse(lease_response.error)

  def test_lease_duration_zero(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=0,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.LEASE_LENGTH_UNSPECIFIED,
    )

  def test_lease_duration_negative(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=-1,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.NONPOSITIVE_DEADLINE,
    )

  def test_lease_duration_too_long(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=9999999999,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.LEASE_TOO_LONG,
    )

  def test_lease_duration_and_lease_expiration_ts(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        lease_expiration_ts=int(utils.time_time()) + 3600,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.MUTUAL_EXCLUSION_ERROR,
    )

  def test_lease_duration_and_indefinite(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        duration=1,
        indefinite=True,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.MUTUAL_EXCLUSION_ERROR,
    )

  def test_lease_timestamp(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=int(utils.time_time()) + 3600,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertFalse(lease_response.error)

  def test_lease_timestamp_passed(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=1,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.LEASE_EXPIRATION_TS_ERROR,
    )

  def test_lease_timestamp_too_far(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        lease_expiration_ts=9999999999,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.LEASE_TOO_LONG,
    )

  def test_lease_timestamp_and_indefinite(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        indefinite=True,
        lease_expiration_ts=int(utils.time_time()) + 3600,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.MUTUAL_EXCLUSION_ERROR,
    )

  def test_lease_indefinite(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        indefinite=True,
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertFalse(lease_response.error)

  def test_lease_length_unspecified(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.LINUX,
        ),
        request_id='abc',
    ))

    lease_response = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertEqual(
        lease_response.error,
        rpc_messages.LeaseRequestError.LEASE_LENGTH_UNSPECIFIED,
    )

  def test_duplicate(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.OSX,
        ),
        duration=3,
        request_id='asdf',
    ))

    lease_response_1 = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    lease_response_2 = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request).json,
        rpc_messages.LeaseResponse,
    )
    self.assertFalse(lease_response_1.error)
    self.assertFalse(lease_response_2.error)
    self.assertEqual(
        lease_response_1.request_hash,
        lease_response_2.request_hash,
    )

  def test_request_id_reuse(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    self.mock(acl.auth, 'is_group_member', is_group_member)
    lease_request_1 = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.WINDOWS,
        ),
        duration=7,
        request_id='qwerty',
    ))
    lease_request_2 = rpc_to_json(rpc_messages.LeaseRequest(
        dimensions=rpc_messages.Dimensions(
            os_family=rpc_messages.OSFamily.WINDOWS,
        ),
        duration=189,
        request_id='qwerty',
    ))

    lease_response_1 = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request_1).json,
        rpc_messages.LeaseResponse,
    )
    lease_response_2 = jsonish_dict_to_rpc(
        self.call_api('lease', lease_request_2).json,
        rpc_messages.LeaseResponse,
    )
    self.assertFalse(lease_response_1.error)
    self.assertEqual(
        lease_response_2.error,
        rpc_messages.LeaseRequestError.REQUEST_ID_REUSE,
    )
    self.assertNotEqual(
        lease_response_1.request_hash,
        lease_response_2.request_hash,
    )


class MachineProviderInstructTest(test_case.EndpointsTestCase):
  """Tests for handlers_endpoints.MachineProviderEndpoints.instruct."""
  api_service_cls = handlers_endpoints.MachineProviderEndpoints

  def setUp(self):
    super(MachineProviderInstructTest, self).setUp()
    app = handlers_endpoints.create_endpoints_app()
    self.app = webtest.TestApp(app)

  def test_lease_request_not_found(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(9999999999),
        lease_id=ndb.Key(models.LeaseRequest, 'id').id(),
    ).put()
    request = rpc_to_json(request)

    with self.assertRaises(webtest.app.AppError):
      jsonish_dict_to_rpc(
          self.call_api('instruct', request).json,
          rpc_messages.MachineInstructionResponse,
      )
    self.failIf(machine_key.get().instruction)

  def test_lease_request_not_fulfilled(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    lease_key = models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            state=rpc_messages.LeaseRequestState.UNTRIAGED,
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(9999999999),
        lease_id=lease_key.id(),
    ).put()
    request = rpc_to_json(request)

    response = jsonish_dict_to_rpc(
        self.call_api('instruct', request).json,
        rpc_messages.MachineInstructionResponse,
    )
    self.assertEqual(
        response.error, rpc_messages.MachineInstructionError.NOT_FULFILLED)
    self.failIf(machine_key.get().instruction)

  def test_lease_request_already_reclaimed(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    lease_key = models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(9999999999),
        lease_id=lease_key.id(),
    ).put()
    request = rpc_to_json(request)

    response = jsonish_dict_to_rpc(
        self.call_api('instruct', request).json,
        rpc_messages.MachineInstructionResponse,
    )
    self.assertEqual(
        response.error, rpc_messages.MachineInstructionError.ALREADY_RECLAIMED)
    self.failIf(machine_key.get().instruction)

  def test_machine_not_found(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            hostname='fake-host',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()
    request = rpc_to_json(request)

    with self.assertRaises(webtest.app.AppError):
      jsonish_dict_to_rpc(
          self.call_api('instruct', request).json,
          rpc_messages.MachineInstructionResponse,
      )

  def test_machine_not_fulfilled(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            hostname='fake-host',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(9999999999),
    ).put()
    request = rpc_to_json(request)

    response = jsonish_dict_to_rpc(
        self.call_api('instruct', request).json,
        rpc_messages.MachineInstructionResponse,
    )
    self.assertEqual(
        response.error, rpc_messages.MachineInstructionError.NOT_FULFILLED)
    self.failIf(machine_key.get().instruction)

  def test_machine_already_reclaimed(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    lease_key = models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            hostname='fake-host',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(1),
        lease_id=lease_key.id(),
    ).put()
    request = rpc_to_json(request)

    response = jsonish_dict_to_rpc(
        self.call_api('instruct', request).json,
        rpc_messages.MachineInstructionResponse,
    )
    self.assertEqual(
        response.error, rpc_messages.MachineInstructionError.ALREADY_RECLAIMED)
    self.failIf(machine_key.get().instruction)

  def test_invalid_instruction(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
        ),
    )
    lease_key = models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            hostname='fake-host',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(9999999999),
        lease_id=lease_key.id(),
    ).put()
    request = rpc_to_json(request)

    response = jsonish_dict_to_rpc(
        self.call_api('instruct', request).json,
        rpc_messages.MachineInstructionResponse,
    )
    self.assertEqual(
        response.error,
        rpc_messages.MachineInstructionError.INVALID_INSTRUCTION,
    )
    self.failIf(machine_key.get().instruction)

  def test_instructed(self):
    def is_group_member(group):
      return group == 'machine-provider-users'
    auth_testing.mock_get_current_identity(self)
    self.mock(acl.auth, 'is_group_member', is_group_member)

    request = rpc_messages.MachineInstructionRequest(
        request_id='request-id',
        instruction=rpc_messages.Instruction(
            swarming_server='example.com',
        ),
    )
    lease_key = models.LeaseRequest(
        key=models.LeaseRequest.generate_key(
            auth_testing.DEFAULT_MOCKED_IDENTITY.to_bytes(),
            request,
        ),
        deduplication_checksum='checksum',
        machine_id='machine',
        owner=auth_testing.DEFAULT_MOCKED_IDENTITY,
        request=rpc_messages.LeaseRequest(
            dimensions=rpc_messages.Dimensions(),
            request_id='request-id',
        ),
        response=rpc_messages.LeaseResponse(
            client_request_id='request-id',
            hostname='fake-host',
            state=rpc_messages.LeaseRequestState.FULFILLED,
        ),
    ).put()
    machine_key = models.CatalogMachineEntry(
        id='machine',
        dimensions=rpc_messages.Dimensions(
            backend=rpc_messages.Backend.DUMMY,
        ),
        lease_expiration_ts=datetime.datetime.fromtimestamp(9999999999),
        lease_id=lease_key.id(),
    ).put()
    request = rpc_to_json(request)

    response = jsonish_dict_to_rpc(
        self.call_api('instruct', request).json,
        rpc_messages.MachineInstructionResponse,
    )
    self.failIf(response.error)
    self.assertEqual(
        machine_key.get().instruction.instruction.swarming_server,
        'example.com',
    )
    self.assertEqual(
        machine_key.get().instruction.state, models.InstructionStates.PENDING)


if __name__ == '__main__':
  unittest.main()
