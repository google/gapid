# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Provides info about registered luci services."""

import logging

from google.appengine.ext import ndb

from components import config
from components import net
from components import utils
from components.config.proto import service_config_pb2

import common
import storage
import validation


class DynamicMetadataError(Exception):
  """Raised when a service metadata endpoint response is bad."""


class ServiceNotFoundError(Exception):
  """Raised when a service is not found."""


@ndb.tasklet
def get_services_async():
  """Returns a list of registered luci services.

  The list is stored in services/luci-config:services.cfg. Never returns None.
  Cached.

  Returns:
    A list of service_config_pb2.Service.
  """
  cfg = yield storage.get_self_config_async(
      common.SERVICES_REGISTRY_FILENAME, service_config_pb2.ServicesCfg)
  raise ndb.Return(cfg.services or [])


def _dict_to_dynamic_metadata(data):
  validation.validate_service_dynamic_metadata_blob(
      data,
      config.validation.Context.raise_on_error(exc_type=DynamicMetadataError))

  metadata = service_config_pb2.ServiceDynamicMetadata()
  validation_meta = data.get('validation')
  if validation_meta:
    metadata.validation.url = validation_meta['url']
    for p in validation_meta.get('patterns', []):
      pattern = metadata.validation.patterns.add()
      pattern.config_set = p['config_set']
      pattern.path = p['path']
  return metadata


@ndb.tasklet
def get_metadata_async(service_id):
  """Returns service dynamic metadata.

  Raises:
    ServiceNotFoundError if service |service_id| is not found.
    DynamicMetadataError if metadata endpoint response is bad.
  """
  entity = yield storage.ServiceDynamicMetadata.get_by_id_async(service_id)
  if entity:
    msg = service_config_pb2.ServiceDynamicMetadata()
    if entity.metadata:
      msg.ParseFromString(entity.metadata)
    raise ndb.Return(msg)

  #TODO(myjang): delete the rest of the function once entities in production
  services = yield get_services_async()
  service = None
  for s in services:
    if s.id == service_id:
      service = s
  if service is None:
    raise ServiceNotFoundError('Service "%s" not found', service_id)

  if not service.metadata_url:
    raise ndb.Return(service_config_pb2.ServiceDynamicMetadata())
  try:
    res = yield net.json_request_async(
        service.metadata_url, scopes=net.EMAIL_SCOPE)
  except net.Error as ex:
    raise DynamicMetadataError('Net error: %s' % ex.message)
  msg = _dict_to_dynamic_metadata(res)
  raise ndb.Return(msg)


@ndb.tasklet
def _update_service_metadata_async(service):
  entity = storage.ServiceDynamicMetadata(id=service.id)
  if service.metadata_url:
    try:
      res = yield net.json_request_async(
          service.metadata_url, scopes=net.EMAIL_SCOPE)
    except net.Error as ex:
      raise DynamicMetadataError('Net error: %s' % ex.message)
    entity.metadata = _dict_to_dynamic_metadata(res).SerializeToString()

  prev_entity = yield storage.ServiceDynamicMetadata.get_by_id_async(service.id)
  if not prev_entity or prev_entity.metadata != entity.metadata:
    yield entity.put_async()
    logging.info('Updated service metadata for %s', service.id)


def cron_request_metadata():
  services = get_services_async().get_result()
  futs = [_update_service_metadata_async(s) for s in services]
  ndb.Future.wait_all(futs)
  for s, fut in zip(services, futs):
    try:
      fut.check_success()
    except DynamicMetadataError:
      logging.exception('Could not load dynamic metadata for %s', s.id)
