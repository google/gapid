# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import re

from google.appengine.api import app_identity
from google.appengine.api import lib_config
from google.appengine.ext import ndb
from protorpc import messages

# Config component is using google.protobuf package, it requires some python
# package magic hacking.
from components import utils
utils.fix_protobuf_package()

from google import protobuf

from components import auth
from components import protoutil
from components import utils
from components.datastore_utils import config


class Severity(messages.Enum):
  DEBUG = logging.DEBUG
  INFO = logging.INFO
  WARNING = logging.WARNING
  ERROR = logging.ERROR
  CRITICAL = logging.CRITICAL


################################################################################
# Patterns


SERVICE_ID_PATTERN = r'[a-z0-9\-_]+'
SERVICE_ID_RGX = re.compile(r'^%s$' % SERVICE_ID_PATTERN)
SERVICE_CONFIG_SET_RGX = re.compile(r'^services/(%s)$' % SERVICE_ID_PATTERN)

PROJECT_ID_PATTERN = SERVICE_ID_PATTERN
PROJECT_ID_RGX = re.compile(r'^%s$' % PROJECT_ID_PATTERN)
PROJECT_CONFIG_SET_RGX = re.compile(r'^projects/(%s)$' % PROJECT_ID_PATTERN)

REF_NAME_PATTERN = r'refs/.+'
REF_NAME_RGX = re.compile(r'^%s$' % REF_NAME_PATTERN)
REF_CONFIG_SET_RGX = re.compile(
    r'^projects/(%s)/(%s)$' % (PROJECT_ID_PATTERN, REF_NAME_PATTERN))

ALL_CONFIG_SET_RGX = [
  SERVICE_CONFIG_SET_RGX,
  PROJECT_CONFIG_SET_RGX,
  REF_CONFIG_SET_RGX,
]


################################################################################
# Settings


class ConstantConfig(object):
  # In filesystem mode, the directory where configs are read from.
  CONFIG_DIR = 'configs'


CONSTANTS = lib_config.register('components_config', ConstantConfig.__dict__)


class ConfigSettings(config.GlobalConfig):
  # Hostname of the config service.
  service_hostname = ndb.StringProperty(indexed=False)
  # Identity account used by config service.
  trusted_config_account = auth.IdentityProperty(indexed=False)


################################################################################
# Config parsing


class ConfigFormatError(Exception):
  """A config file is malformed."""


def _validate_dest_type(dest_type):
  if dest_type is None:
    return
  if not issubclass(dest_type, protobuf.message.Message):
    raise NotImplementedError('%s type is not supported' % dest_type.__name__)


def _convert_config(content, dest_type):
  _validate_dest_type(dest_type)
  if dest_type is None or isinstance(content, dest_type):
    return content
  if content is None:
    return None
  msg = dest_type()
  try:
    protobuf.text_format.Merge(
        protoutil.parse_multiline(content.decode('utf-8')), msg)
  except (protoutil.MultilineParseError, protobuf.text_format.ParseError,
          UnicodeDecodeError) as ex:
    raise ConfigFormatError(ex.message)
  return msg


################################################################################
# Rest


def _trim_app_id(app_id):
  """Returns the App ID with the domain prefix removed, if present."""
  return app_id.split(':')[-1]


@utils.cache
def self_config_set():
  return 'services/%s' % _trim_app_id(app_identity.get_application_id())


def config_service_hostname():
  """Returns hostname of the config service, or None."""
  settings = ConfigSettings.cached()
  return settings and settings.service_hostname or None
