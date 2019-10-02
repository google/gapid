#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from components import gce
from components import net
from test_support import test_case


def instance(name, status='RUNNING'):
  return {'name': name, 'status': status}


class GceTest(test_case.TestCase):
  def mock_requests(self, requests):
    def mocked_request(**kwargs):
      if not requests:  # pragma: no cover
        self.fail('Unexpected request: %r' % (kwargs,))
      expected, response = requests.pop(0)
      defaults = {
        'deadline': 30,
        'method': 'GET',
        'params': None,
        'payload': None,
        'scopes': ['https://www.googleapis.com/auth/compute'],
        'service_account_key': None,
      }
      defaults.update(expected)
      self.assertEqual(defaults, kwargs)
      if isinstance(response, net.Error):
        raise response
      return response
    self.mock(net, 'json_request', mocked_request)
    return requests

  def test_machine_type_to_num_cpus(self):
    self.assertEqual(8, gce.machine_type_to_num_cpus('n1-standard-8'))
    self.assertEqual(1, gce.machine_type_to_num_cpus('custom-1-2048'))
    with self.assertRaises(AssertionError):
      gce.machine_type_to_num_cpus('incorrect-machine-type')

  def test_machine_type_to_memory(self):
    self.assertEqual(30, gce.machine_type_to_memory('n1-standard-8'))
    self.assertEqual(2, gce.machine_type_to_memory('custom-1-2048'))
    with self.assertRaises(AssertionError):
      gce.machine_type_to_memory('incorrect-machine-type')

  def test_project_id(self):
    self.assertEqual('123', gce.Project('123').project_id)

  def test_get_instance(self):
    self.mock_requests([
      (
        {
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/zone-id/instances/inst_id',
        },
        instance('inst_id'),
      ),
    ])
    self.assertEqual(
        instance('inst_id'),
        gce.Project('123').get_instance('zone-id', 'inst_id'))

  def test_get_instance_with_fields(self):
    self.mock_requests([
      (
        {
          'params': {'fields': 'metadata,name'},
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/zone-id/instances/inst_id',
        },
        instance('inst_id'),
      ),
    ])
    self.assertEqual(
        instance('inst_id'),
        gce.Project('123').get_instance(
            'zone-id', 'inst_id', ['metadata', 'name']))

  def test_yield_instances(self):
    self.mock_requests([
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/aggregated/instances',
        },
        {
          'items': {
            'zone1': {'instances': [instance('a')]},
            'zone2': {'instances': [instance('b')]},
          },
          'nextPageToken': 'page-token',
        },
      ),
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
            'pageToken': 'page-token',
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/aggregated/instances',
        },
        {
          'items': {
            'zone1': {'instances': [instance('c')]},
          },
        },
      ),
    ])
    result = list(gce.Project('123').yield_instances())
    self.assertEqual(
        [instance('a'), instance('b'), instance('c')], result)

  def test_yield_instances_with_filter(self):
    self.mock_requests([
      (
        {
          'deadline': 120,
          'params': {
            'filter': 'name eq "inst-filter"',
            'maxResults': 250,
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/aggregated/instances',
        },
        {
          'items': {
            'zone1': {'instances': [instance('a')]},
            'zone2': {'instances': [instance('b', status='STOPPED')]},
          },
        },
      ),
    ])
    result = list(gce.Project('123').yield_instances('inst-filter'))
    self.assertEqual([instance('a'), instance('b', status='STOPPED')], result)

  def test_yield_instances_bad_filter(self):
    with self.assertRaises(ValueError):
      list(gce.Project('123').yield_instances('"'))

  def test_yield_instances_in_zones(self):
    self.mock_requests([
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/z1/instances',
        },
        {
          'items': [instance('a')],
          'nextPageToken': 'page-token',
        },
      ),
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
            'pageToken': 'page-token',
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/z1/instances',
        },
        {
          'items': [instance('b')],
        },
      ),
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/z2/instances',
        },
        {
          'items': [instance('c')],
        },
      ),
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/z3/instances',
        },
        # Missing zone is ignored.
        net.Error('Bad request', 400, json.dumps({
          'error': {
            'errors': [{
              'domain': 'global',
              'reason': 'invalid',
              'message':
                'Invalid value for field \'zone\': \'z3\'. Unknown zone.',
            }],
            'code': 400,
            'message':
              'Invalid value for field \'zone\': \'z3\'. Unknown zone.',
          },
        })),
      ),
    ])
    result = gce.Project('123').yield_instances_in_zones(['z1', 'z2', 'z3'])
    self.assertEqual(
        [instance('a'), instance('b'), instance('c')], list(result))

  def test_set_metadata(self):
    self.mock_requests([
      (
        {
          'method': 'POST',
          'payload': {
            'fingerprint': 'fingerprint',
            'items': [{'key': 'k', 'value': 'v'}],
            'kind': 'compute#metadata'
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/zone-id/instances/inst_id/setMetadata',
        },
        {
          'name': 'operation',
          'status': 'DONE',
        },
      ),
    ])
    op = gce.Project('123').set_metadata(
        'zone-id', 'inst_id', 'fingerprint', [{'key': 'k', 'value': 'v'}])
    self.assertTrue(op.done)

  def test_add_access_config(self):
    self.mock_requests([
      (
        {
          'method': 'POST',
          'params': {'networkInterface': 'nic0'},
          'payload': {
            'kind': 'compute#accessConfig',
            'name': 'External NAT',
            'natIP': '1.2.3.4',
            'type': 'ONE_TO_ONE_NAT',
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/zone-id/instances/inst_id/addAccessConfig',
        },
        {
          'name': 'operation',
          'status': 'DONE',
        },
      ),
    ])
    op = gce.Project('123').add_access_config(
        'zone-id', 'inst_id', 'nic0', '1.2.3.4')
    self.assertTrue(op.done)

  def test_list_addresses(self):
    self.mock_requests([
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/regions/region-id/addresses',
        },
        {
          'items': [{'name': 'a'}, {'name': 'b'}],
          'nextPageToken': 'page-token',
        },
      ),
      (
        {
          'deadline': 120,
          'params': {
            'maxResults': 250,
            'pageToken': 'page-token',
          },
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/regions/region-id/addresses',
        },
        {
          'items': [{'name': 'c'}],
        },
      ),
    ])
    result = list(gce.Project('123').list_addresses('region-id'))
    self.assertEqual(
        [{'name': 'a'}, {'name': 'b'}, {'name': 'c'}], result)

  def test_zone_operation_poll(self):
    self.mock_requests([
      (
        {
          'url':
              'https://www.googleapis.com/compute/v1/projects/123'
              '/zones/zone-id/operations/op',
        },
        {
          'name': 'op',
          'status': 'DONE',
        },
      ),
    ])
    op = gce.ZoneOperation(
        gce.Project('123'), 'zone-id', {'name': 'op', 'status': 'PENDING'})
    self.assertFalse(op.done)
    self.assertFalse(op.error)
    self.assertTrue(op.poll())
    # Second 'poll' is skipped if the operation is already done.
    self.assertTrue(op.poll())

  def test_zone_operation_error(self):
    op = gce.ZoneOperation(
        gce.Project('123'),
        'zone-id',
        {
          'name': 'op',
          'status': 'DONE',
          'error': {
            'errors': [
              {'message': 'A', 'code': 'ERROR_CODE'},
              {'message': 'B'},
            ],
          },
        })
    self.assertTrue(op.has_error_code('ERROR_CODE'))
    self.assertFalse(op.has_error_code('NOT_ERROR_CODE'))
    self.assertEqual('A B', op.error)

  def test_is_valid_project_id(self):
    self.assertTrue(gce.is_valid_project_id('project'))
    self.assertTrue(gce.is_valid_project_id('1234567'))
    self.assertFalse(gce.is_valid_project_id(''))
    self.assertFalse(gce.is_valid_project_id('12345/67'))

  def test_is_valid_region(self):
    self.assertTrue(gce.is_valid_region('us-central1'))
    self.assertFalse(gce.is_valid_region('us-central1:a'))

  def test_is_valid_zone(self):
    self.assertTrue(gce.is_valid_zone('us-central1-a'))
    self.assertFalse(gce.is_valid_zone('us-central1/a'))

  def test_is_valid_instance(self):
    self.assertTrue(gce.is_valid_instance('slave123-c4'))
    self.assertFalse(gce.is_valid_instance('slave123/c4'))

  def test_get_network_interfaces(self):
    expected_interfaces_ext  = [{'network': 'global/networks/default',
                                 'accessConfigs': [{'type': 'ONE_TO_ONE_NAT'}]}]
    expected_interfaces_int  = [{'network': 'global/networks/default'}]
    self.assertEqual(expected_interfaces_int,
                     gce.get_network_interfaces('project',
                                                'global/networks/default',
                                                False))
    self.assertEqual(expected_interfaces_ext,
                     gce.get_network_interfaces('project',
                                                'global/networks/default',
                                                True))
    expected_interfaces_int  = [{'network':
                                 gce.get_network_url('project', 'default')}]
    self.assertEqual(expected_interfaces_int,
                     gce.get_network_interfaces('project', '', False))

  def test_get_zone_url(self):
    self.assertEqual(
        'https://www.googleapis.com/compute/v1/projects'
        '/123/zones/us-central1-a', gce.get_zone_url('123', 'us-central1-a'))

  def test_extract_zone(self):
    self.assertEqual(
        'us-central1-a',
        gce.extract_zone(
            'https://www.googleapis.com/compute/v1/projects'
            '/123/zones/us-central1-a'))

  def test_get_region_url(self):
    self.assertEqual(
        'https://www.googleapis.com/compute/v1/projects'
        '/123/regions/us-central1', gce.get_region_url('123', 'us-central1'))

  def test_extract_region(self):
    self.assertEqual(
        'us-central1',
        gce.extract_region(
            'https://www.googleapis.com/compute/v1/projects'
            '/123/regions/us-central1'))

  def test_region_from_zone(self):
    self.assertEqual('us-central1', gce.region_from_zone('us-central1-a'))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
