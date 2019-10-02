# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Cloud Endpoints API for configs.

* Reads/writes config service location.
* Validates configs.
* Provides service metadata.
"""

import logging

from protorpc import messages
from protorpc import message_types
from protorpc import remote
import endpoints

from components import auth

from . import common
from . import validation


METADATA_FORMAT_VERSION = "1.0"


class ConfigSettingsMessage(messages.Message):
  """Configuration service location. Resembles common.ConfigSettings"""
  # Example: 'luci-config.appspot.com'
  service_hostname = messages.StringField(1)
  # Example: 'user:luci-config@appspot.gserviceaccount.com'
  trusted_config_account = messages.StringField(2)


class ValidateRequestMessage(messages.Message):
  config_set = messages.StringField(1, required=True)
  path = messages.StringField(2, required=True)
  content = messages.BytesField(3, required=True)


class ValidationMessage(messages.Message):
  path = messages.StringField(1)
  severity = messages.EnumField(common.Severity, 2, required=True)
  text = messages.StringField(3, required=True)


class ValidateResponseMessage(messages.Message):
  messages = messages.MessageField(ValidationMessage, 1, repeated=True)


def is_trusted_requester():
  """Returns True if the requester can see the service metadata.

  Used in metadata endpoint.

  Returns:
    True if the current identity is an admin or the config service.
  """
  if auth.is_superuser() or auth.is_admin():
    return True

  settings = common.ConfigSettings.cached()
  if settings and settings.trusted_config_account:
    identity = auth.get_current_identity()
    if identity == settings.trusted_config_account:
      return True

  return False


class ConfigPattern(messages.Message):
  """A pattern for one config file. See ServiceDynamicMetadata."""
  config_set = messages.StringField(1, required=True)
  path = messages.StringField(2, required=True)


class ServiceDynamicMetadata(messages.Message):
  """Equivalent of config_service's ServiceDynamicMetadata proto message.

  Keep this class in sync with:
    * ServiceDynamicMetadata message in
      appengine/config_service/proto/service_config.proto
    * validation.validate_service_dynamic_metadata_blob()
    * services._dict_to_dynamic_metadata()
  """

  class Validator(messages.Message):
    patterns = messages.MessageField(ConfigPattern, 1, repeated=True)
    url = messages.StringField(2, required=True)

  version = messages.StringField(1, required=True)
  validation = messages.MessageField(Validator, 2)


@auth.endpoints_api(name='config', version='v1', title='Configuration service')
class ConfigApi(remote.Service):
  """Configuration service."""

  @auth.endpoints_method(
      ConfigSettingsMessage, ConfigSettingsMessage,
      http_method='POST')
  @auth.require(lambda: auth.is_superuser() or auth.is_admin())
  def settings(self, request):
    """Reads/writes config service location. Accessible only by admins."""
    settings = common.ConfigSettings.fetch() or common.ConfigSettings()
    delta = {}
    if request.service_hostname is not None:
      delta['service_hostname'] = request.service_hostname
    if request.trusted_config_account is not None:
      try:
        delta['trusted_config_account'] = auth.Identity.from_bytes(
            request.trusted_config_account)
      except ValueError as ex:
        raise endpoints.BadRequestException(
            'Invalid trusted_config_account %s: %s' % (
              request.trusted_config_account,
              ex.message))
    changed = settings.modify(
        updated_by=auth.get_current_identity().to_bytes(),
        **delta)
    if changed:
      logging.warning('Updated config settings')
    settings = common.ConfigSettings.fetch() or settings
    return ConfigSettingsMessage(
        service_hostname=settings.service_hostname,
        trusted_config_account=(
            settings.trusted_config_account.to_bytes()
            if settings.trusted_config_account else None)
    )

  @auth.endpoints_method(
      ValidateRequestMessage, ValidateResponseMessage, http_method='POST')
  @auth.require(is_trusted_requester)
  def validate(self, request):
    """Validates a config.

    Compatible with validation protocol described in ValidationCfg message of
    /appengine/config_service/proto/service_config.proto.
    """
    ctx = validation.Context()
    validation.validate(request.config_set, request.path, request.content, ctx)

    res = ValidateResponseMessage()
    for m in ctx.result().messages:
      text = m.text
      if isinstance(m.text, str):
        text = text.decode('ascii', errors='replace')
      res.messages.append(ValidationMessage(
          path=request.path,
          severity=common.Severity.lookup_by_number(m.severity),
          text=text,
      ))
    return res

  @auth.endpoints_method(
      message_types.VoidMessage, ServiceDynamicMetadata,
      http_method='GET', path='metadata')
  @auth.require(is_trusted_requester)
  def get_metadata(self, _request):
    """Describes a service. Used by config service to discover other services.
    """
    meta = ServiceDynamicMetadata(version=METADATA_FORMAT_VERSION)
    http_headers = dict(self.request_state.headers)
    assert 'host' in http_headers or 'Host' in http_headers, http_headers
    meta.validation = meta.Validator(
        url='https://{hostname}/_ah/api/{name}/{version}/{path}validate'.format(
            hostname=http_headers.get('host') or http_headers['Host'],
            name=self.api_info.name,
            version=self.api_info.version,
            path=self.api_info.path or '',
        )
    )
    for p in sorted(validation.DEFAULT_RULE_SET.patterns()):
      meta.validation.patterns.append(ConfigPattern(**p._asdict()))
    return meta
