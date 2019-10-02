# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Helper functions for working with the Machine Provider."""

import logging

from google.appengine.ext import ndb

from components import net
from components import utils
from components.datastore_utils import config


MACHINE_PROVIDER_SCOPES = (
    'https://www.googleapis.com/auth/userinfo.email',
)


class MachineProviderConfiguration(config.GlobalConfig):
  """Configuration for talking to the Machine Provider."""
  # URL of the Machine Provider instance to use.
  instance_url = ndb.StringProperty(required=True)

  @classmethod
  def get_instance_url(cls):
    """Returns the URL of the Machine Provider instance."""
    return cls.cached().instance_url

  def set_defaults(self):
    """Sets default values used to initialize the config."""
    self.instance_url = 'https://machine-provider.appspot.com'


def add_machine(dimensions, policies):
  """Add a machine to the Machine Provider's Catalog.

  Args:
    dimensions: Dimensions for this machine.
    policies: Policies governing this machine.
  """
  logging.info('Sending add_machine request')
  return net.json_request(
      '%s/_ah/api/catalog/v1/add_machine' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({
          'dimensions': dimensions,
          'policies': policies,
      }),
      scopes=MACHINE_PROVIDER_SCOPES,
  )


def add_machines(requests):
  """Add machines to the Machine Provider's Catalog.

  Args:
    requests: A list of rpc_messages.CatalogMachineAdditionRequest instances.
  """
  logging.info('Sending batched add_machines request')
  return net.json_request(
      '%s/_ah/api/catalog/v1/add_machines' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({'requests': requests}),
      scopes=MACHINE_PROVIDER_SCOPES,
 )


def delete_machine(dimensions):
  """Deletes a machine from the Machine Provider's Catalog.

  Args:
    dimensions: Dimensions for the machine.
  """
  logging.info('Sending delete_machine request')
  return net.json_request(
      '%s/_ah/api/catalog/v1/delete_machine' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({
          'dimensions': dimensions,
      }),
      scopes=MACHINE_PROVIDER_SCOPES,
  )


def instruct_machine(request_id, swarming_server):
  """Instruct a leased machine to connect to a Swarming server.

  Args:
    request_id: Request ID for the fulfilled lease whose machine to send
      the instruction to.
    swarming_server: URL of the Swarming server to connect to.
  """
  return net.json_request(
      '%s/_ah/api/machine_provider/v1/instruct' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({
          'instruction': {
              'swarming_server': swarming_server,
          },
          'request_id': request_id,
      }),
      scopes=MACHINE_PROVIDER_SCOPES,
  )


def lease_machine(request):
  """Lease a machine from the Machine Provider.

  Args:
    request: An rpc_messages.LeaseRequest instance.
  """
  return net.json_request(
      '%s/_ah/api/machine_provider/v1/lease' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable(request),
      scopes=MACHINE_PROVIDER_SCOPES,
  )


def lease_machines(requests):
  """Lease machines from the Machine Provider.

  Args:
    requests: A list of rpc_messages.LeaseRequest instances.
  """
  logging.info('Sending batched lease_machines request')
  return net.json_request(
      '%s/_ah/api/machine_provider/v1/batched_lease' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({'requests': requests}),
      scopes=MACHINE_PROVIDER_SCOPES,
  )


def release_machine(client_request_id):
  """Voluntarily releases a leased machine back to Machine Provider.

  Args:
    client_request_id: Request ID originally used by the client when creating
      the lease request.
  """
  return net.json_request(
      '%s/_ah/api/machine_provider/v1/release' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({'request_id': client_request_id}),
      scopes=MACHINE_PROVIDER_SCOPES,
  )


def retrieve_machine(hostname, backend=None):
  """Requests information about a machine from the Machine Provider's Catalog.

  Args:
    hostname: Hostname of the machine to request information about.
    backend: Backend the machine belongs to.
  """
  return net.json_request(
      '%s/_ah/api/catalog/v1/get' %
          MachineProviderConfiguration.get_instance_url(),
      method='POST',
      payload=utils.to_json_encodable({
          'backend': backend,
          'hostname': hostname,
      }),
      scopes=MACHINE_PROVIDER_SCOPES,
  )
