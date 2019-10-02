# coding: utf-8
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Tasks definition.

Each user request creates a new TaskRequest. The TaskRequest instance saves the
metadata of the request, e.g. who requested it, when why, etc. It links to the
actual data of the request in a TaskProperties. The TaskProperties represents
everything needed to run the task.

This means if two users request an identical task, it can be deduped
accordingly and efficiently by the scheduler.

Note that the mere existence of a TaskRequest in the db doesn't mean it will be
scheduled, see task_scheduler.py for the actual scheduling. Registering tasks
and scheduling are kept separated to keep the immutable and mutable models in
separate files.


Overview of transactions:
- TaskRequest() are created inside a transaction.


Graph of the schema:

    +--------Root------------------------------------------------------+
    |TaskRequest                                                       |
    |  +--------------+      +----------------+     +----------------+ |
    |  |TaskProperties|      |TaskSlice       |     |TaskSlice       | |
    |  |  +--------+  |      |+--------------+| ... |+--------------+| |
    |  |  |FilesRef|  | *or* ||TaskProperties|| ... ||TaskProperties|| |
    |  |  +--------+  |      |+--------------+|     |+--------------+| |
    |  +--------------+      +----------------+     +----------------+ |
    |id=<based on epoch>                                               |
    +------------------------------------------------------------------+
        |          |
        v          |
    +-----------+  |
    |SecretBytes|  |
    |id=1       |  |
    +-----------+  |
                   |
                   v
    <See task_to_run.py and task_result.py>

TaskProperties is embedded in TaskRequest. TaskProperties is still declared as a
separate entity to clearly declare the boundary for task request deduplication.
"""

import datetime
import hashlib
import logging
import posixpath
import random
import re

from google.appengine import runtime
from google.appengine.api import datastore_errors
from google.appengine.ext import ndb

from components import auth
from components import datastore_utils
from components import pubsub
from components import utils
from components.config import validation

from proto.api import swarming_pb2
from server import bq_state
from server import config
from server import directory_occlusion
from server import pools_config
from server import service_accounts
from server import task_pack

import cipd


# Maximum acceptable priority value, which is effectively the lowest priority.
MAXIMUM_PRIORITY = 255


# Enum singletons controlling the application of the pool task templates in
# init_new_request. This is the counterpart of
# swarming_rpcs.PoolTaskTemplateField
class TemplateApplyEnum(str):
  pass
TEMPLATE_AUTO = TemplateApplyEnum('TEMPLATE_AUTO')
TEMPLATE_CANARY_PREFER = TemplateApplyEnum('TEMPLATE_CANARY_PREFER')
TEMPLATE_CANARY_NEVER = TemplateApplyEnum('TEMPLATE_CANARY_NEVER')
TEMPLATE_SKIP = TemplateApplyEnum('TEMPLATE_SKIP')


# Maximum allowed timeout for I/O and hard timeouts.
#
# Three days in seconds. Includes an additional 10s to account for small jitter.
MAX_TIMEOUT_SECS = 3*24*60*60 + 10


# Maximum allowed expiration for a pending task.
#
# Seven days in seconds. Includes an additional 10s to account for small jitter.
MAX_EXPIRATION_SECS = 7*24*60*60 + 10


# Minimum value for timeouts.
_MIN_TIMEOUT_SECS = 1 if utils.is_local_dev_server() else 30


# The world started on 2010-01-01 at 00:00:00 UTC. The rationale is that using
# EPOCH (1970) means that 40 years worth of keys are wasted.
#
# Note: This creates a 'naive' object instead of a formal UTC object. Note that
# datetime.datetime.utcnow() also return naive objects. That's python.
_BEGINING_OF_THE_WORLD = datetime.datetime(2010, 1, 1, 0, 0, 0, 0)


# 1 is for ':'.
_TAG_LENGTH = config.DIMENSION_KEY_LENGTH + config.DIMENSION_VALUE_LENGTH + 1


# Used for isolated files.
_HASH_CHARS = frozenset('0123456789abcdef')

# Keep synced with named_cache.py
_CACHE_NAME_RE = re.compile(ur'^[a-z0-9_]{1,4096}$')


# Early verification of environment variable key name.
_ENV_KEY_RE = re.compile(r'^[A-Za-z_][A-Za-z0-9_]*$')


# TaskRequest entity groups are deleted when they are older than the cutoff.
_OLD_TASK_REQUEST_CUT_OFF = datetime.timedelta(days=18*31)

# Number of TaskRequest entity groups to be deleted per GAE task. In practice
# we've observed it's possible to delete 1500 TaskRequest groups per 4.5
# minutes, so there's ~3x room.
#
# Defined here so it can be reduced in tests.
_TASKS_DELETE_CHUNK_SIZE = 1000


### Properties validators must come before the models.


def _validate_length(prop, value, maximum):
  if len(value) > maximum:
    raise datastore_errors.BadValueError(
        'too long %s: %d > %d' % (prop._name, len(value), maximum))


def _get_validate_length(maximum):
  return lambda prop, value: _validate_length(prop, value, maximum)


def _validate_url(prop, value):
  _validate_length(prop, value, 1024)
  if value and not validation.is_valid_secure_url(value):
    raise datastore_errors.BadValueError(
        '%s must be valid HTTPS URL, not %s' % (prop._name, value))


def _validate_dimensions(_prop, value):
  """Validates TaskProperties.dimensions."""
  maxkeys = 32
  maxvalues = 16
  if not value:
    raise datastore_errors.BadValueError(u'dimensions must be specified')
  if len(value) > maxkeys:
    raise datastore_errors.BadValueError(
        u'dimensions can have up to %d keys' % maxkeys)

  normalized = {}
  for k, values in value.iteritems():
    # Validate the key.
    if not config.validate_dimension_key(k):
      raise datastore_errors.BadValueError(
          u'dimension key must a string that fits %r: %r is invalid' %
          (config.DIMENSION_KEY_RE, k))

    # Validate the values.
    if not values:
      raise datastore_errors.BadValueError(
          u'dimensions must be a dict of strings or list of string, not %r' %
          value)
    if not isinstance(values, (list, tuple)):
      # Internal bug.
      raise datastore_errors.BadValueError(
          u'dimensions must be a dict of strings or list of string, not %r' %
          value)

    if len(values) > maxvalues:
      raise datastore_errors.BadValueError(
          u'dimension key %r has too many values; maximum is %d' %
          (k, maxvalues))
    if len(values) != len(set(values)):
      raise datastore_errors.BadValueError(
          u'dimension key %r has repeated values' % k)
    for v in values:
      if not config.validate_dimension_value(v):
        raise datastore_errors.BadValueError(
            u'dimension key %r has invalid value %r' % (k, v))

    # Key specific checks.
    if k == u'id' and len(values) != 1:
      raise datastore_errors.BadValueError(
          u'\'id\' cannot be specified more than once in dimensions')
    # Do not allow a task to be triggered in multiple pools, as this could
    # cross a security boundary.
    if k == u'pool' and len(values) != 1:
      raise datastore_errors.BadValueError(
          u'\'pool\' cannot be specified more than once in dimensions')

    # Always store the values sorted, that simplies the code.
    normalized[k] = sorted(values)

  return normalized


def _validate_env_key(prop, key):
  """Validates TaskProperties.env."""
  maxlen = 64
  if not isinstance(key, unicode):
    raise TypeError(
        '%s must have string key, not %r' % (prop._name, key))
  if not key:
    raise datastore_errors.BadValueError(
        'valid key are required in %s' % prop._name)
  if len(key) > maxlen:
    raise datastore_errors.BadValueError(
        'key in %s is too long: %d > %d' % (prop._name, len(key), maxlen))
  if not _ENV_KEY_RE.match(key):
    raise datastore_errors.BadValueError(
        'key in %s is invalid: %r' % (prop._name, key))


def _validate_env(prop, value):
  # pylint: disable=protected-access
  if not all(isinstance(v, unicode) for v in value.itervalues()):
    raise TypeError(
        '%s must be a dict of strings, not %r' % (prop._name, value))
  maxlen = 1024
  for k, v in value.iteritems():
    _validate_env_key(prop, k)
    if len(v) > maxlen:
      raise datastore_errors.BadValueError(
          '%s: key %r has too long value: %d > %d' %
          (prop._name, k, len(v), maxlen))
  if len(value) > 64:
    raise datastore_errors.BadValueError(
        '%s can have up to 64 keys' % prop._name)


def _validate_env_prefixes(prop, value):
  # pylint: disable=protected-access
  maxlen = 1024
  for k, values in value.iteritems():
    _validate_env_key(prop, k)
    if (not isinstance(values, list) or
        not all(isinstance(v, unicode) for v in values)):
      raise TypeError(
          '%s must have list unicode value for key %r, not %r' %
          (prop._name, k, values))
    for path in values:
      if len(path) > maxlen:
        raise datastore_errors.BadValueError(
            '%s: value for key %r is too long: %d > %d' %
            (prop._name, k, len(path), maxlen))
      _validate_rel_path('Env Prefix', path)

  if len(value) > 64:
    raise datastore_errors.BadValueError(
        '%s can have up to 64 keys' % prop._name)


def _check_expiration_secs(name, value):
  """Validates expiration_secs."""
  if not (_MIN_TIMEOUT_SECS <= value <= MAX_EXPIRATION_SECS):
    raise datastore_errors.BadValueError(
        '%s (%s) must be between %ds and 7 days' %
        (name, value, _MIN_TIMEOUT_SECS))


def _validate_expiration_ts(prop, value):
  """Validates TaskRequest.expiration_ts."""
  # pylint: disable=protected-access
  offset = int(round((value - utils.utcnow()).total_seconds()))
  _check_expiration_secs(prop._name, offset)


def _validate_expiration_secs(prop, value):
  """Validates TaskSlice.expiration_secs."""
  # pylint: disable=protected-access
  _check_expiration_secs(prop._name, value)


def _validate_grace(prop, value):
  """Validates grace_period_secs in TaskProperties."""
  # pylint: disable=protected-access
  if not (0 <= value <= 60*60):
    raise datastore_errors.BadValueError(
        '%s (%ds) must be between 0s and one hour' % (prop._name, value))


def _validate_priority(_prop, value):
  """Validates TaskRequest.priority."""
  validate_priority(value)
  return value


def _validate_task_run_id(_prop, value):
  """Validates a task_id looks valid without fetching the entity."""
  if not value:
    return None
  task_pack.unpack_run_result_key(value)
  return value


def _validate_timeout(prop, value):
  """Validates timeouts in seconds in TaskProperties."""
  # pylint: disable=protected-access
  if value and not (_MIN_TIMEOUT_SECS <= value <= MAX_TIMEOUT_SECS):
    raise datastore_errors.BadValueError(
        '%s (%ds) must be 0 or between %ds and three days' %
            (prop._name, value, _MIN_TIMEOUT_SECS))


def _validate_tags(prop, value):
  """Validates TaskRequest.tags."""
  # pylint: disable=protected-access
  _validate_length(prop, value, _TAG_LENGTH)
  if ':' not in value:
    raise datastore_errors.BadValueError(
        '%s must be key:value form, not %s' % (prop._name, value))


def _validate_pubsub_topic(prop, value):
  """Validates TaskRequest.pubsub_topic."""
  # pylint: disable=protected-access
  _validate_length(prop, value, 1024)
  if value and '/' not in value:
    raise datastore_errors.BadValueError(
        '%s must be a well formatted pubsub topic' % (prop._name))


def _validate_package_name_template(prop, value):
  """Validates a CIPD package name template."""
  _validate_length(prop, value, 1024)
  if not cipd.is_valid_package_name_template(value):
    raise datastore_errors.BadValueError(
        '%s must be a valid CIPD package name template "%s"' % (
              prop._name, value))


def _validate_package_version(prop, value):
  """Validates a CIPD package version."""
  _validate_length(prop, value, 1024)
  if not cipd.is_valid_version(value):
    raise datastore_errors.BadValueError(
        '%s must be a valid package version "%s"' % (prop._name, value))


def _validate_cache_name(prop, value):
  _validate_length(prop, value, 128)
  if not _CACHE_NAME_RE.match(value):
    raise datastore_errors.BadValueError(
        '%s %r does not match %s' % (prop._name, value, _CACHE_NAME_RE.pattern))


def _validate_cache_path(prop, value):
  _validate_length(prop, value, 256)
  _validate_rel_path('Cache path', value)


def _validate_package_path(prop, value):
  """Validates a CIPD installation path."""
  _validate_length(prop, value, 256)
  if not value:
    raise datastore_errors.BadValueError(
        'CIPD package path is required. Use "." to install to run dir.')
  _validate_rel_path('CIPD package path', value)


def _validate_output_path(prop, value):
  """Validates a path for an output file."""
  _validate_length(prop, value, 512)
  _validate_rel_path('output file', value)


def _validate_rel_path(value_name, path):
  """Validates a relative path to be valid.

  Length have to be validated first.
  """
  if not path:
    raise datastore_errors.BadValueError(
        'No argument provided for %s.' % value_name)
  if '\\' in path:
    raise datastore_errors.BadValueError(
        '%s cannot contain \\. On Windows forward-slashes '
        'will be replaced with back-slashes.' % value_name)
  if '..' in path.split('/'):
    raise datastore_errors.BadValueError(
        '%s cannot contain "..".' % value_name)
  normalized = posixpath.normpath(path)
  if path != normalized:
    raise datastore_errors.BadValueError(
        '%s is not normalized. Normalized is "%s".' % (value_name, normalized))
  if path.startswith('/'):
    raise datastore_errors.BadValueError(
        '%s cannot start with "/".' % value_name)


def _validate_service_account(prop, value):
  """Validates that 'service_account' field is 'bot', 'none' or email."""
  _validate_length(prop, value, 128)
  if not value:
    return None
  if value in ('bot', 'none') or service_accounts.is_service_account(value):
    return value
  raise datastore_errors.BadValueError(
      '%r must be an email, "bot" or "none" string, got %r' %
      (prop._name, value))


### Models.


class FilesRef(ndb.Model):
  """Defines a data tree reference for Swarming task inputs or outputs.

  It can either be:
    - a reference to an isolated file on an isolate server
    - a reference to an isolated file on a RBE CAS server

  In the RBE CAS case, the isolatedserver must be set to GCP name, and namespace
  must be set to "sha256-GCP". For the moment, RBE CAS requires SHA-256 and
  doesn't support precompressed data.
  """
  # The hash of an isolated archive.
  isolated = ndb.StringProperty(indexed=False)
  # The hostname of the isolated server to use or the Google Cloud Project name.
  isolatedserver = ndb.StringProperty(indexed=False)
  # Namespace on the isolate server or "sha256-GCP" for a RBE CAS.
  namespace = ndb.StringProperty(indexed=False)

  def to_proto(self, out):
    """Converts self to a swarming_pb2.CASTree."""
    if self.isolated:
      out.digest = self.isolated
    if self.isolatedserver:
      out.server = self.isolatedserver
    if self.namespace:
      out.namespace = self.namespace

  def _pre_put_hook(self):
    super(FilesRef, self)._pre_put_hook()
    if not self.isolatedserver or not self.namespace:
      raise datastore_errors.BadValueError(
          'isolate server and namespace are required')

    if self.namespace == 'sha256-GCP':
      # Minimally validate GCP project name. For now, just assert length and
      # that it doesn't contain '://'.
      if ((not 3 <= len(self.isolatedserver) <= 30) or
          '://' in self.isolatedserver):
        raise datastore_errors.BadValueError(
            'isolatedserver must be valid GCP project')
    else:
      _validate_url(self.__class__.isolatedserver, self.isolatedserver)
      _validate_length(self.__class__.namespace, self.namespace, 128)
      if not pools_config.NAMESPACE_RE.match(self.namespace):
        raise datastore_errors.BadValueError('malformed namespace')

    if self.isolated:
      if not _HASH_CHARS.issuperset(self.isolated):
        raise datastore_errors.BadValueError(
            'isolated must be lowercase hex')
      length = len(self.isolated)
      expected = 40
      if self.namespace.startswith('sha256-'):
        expected = 64
      if self.namespace.startswith('sha512-'):
        expected = 128
      if length != expected:
        raise datastore_errors.BadValueError(
            'isolated must be lowercase hex of length %d, but length is %d' %
            (expected, length))

class SecretBytes(ndb.Model):
  """Defines an optional secret byte string logically defined with the
  TaskProperties.

  Stored separately for size and data-leakage reasons.
  """
  _use_memcache = False
  secret_bytes = ndb.BlobProperty(
      validator=_get_validate_length(20*1024), indexed=False)


class CipdPackage(ndb.Model):
  """A CIPD package to install in the run dir before task execution.

  A part of TaskProperties.
  """
  # Package name template. May use cipd.ALL_PARAMS.
  # Most users will specify ${platform} parameter.
  package_name = ndb.StringProperty(
      indexed=False, validator=_validate_package_name_template)
  # Package version that is valid for all packages matched by package_name.
  # Most users will specify tags.
  version = ndb.StringProperty(
      indexed=False, validator=_validate_package_version)
  # Path to dir, relative to the run dir, where to install the package.
  # If empty, the package will be installed in the run dir.
  path = ndb.StringProperty(indexed=False, validator=_validate_package_path)

  def to_proto(self, out):
    """Converts self to a swarming_pb2.CIPDPackage."""
    if self.package_name:
      out.package_name = self.package_name
    if self.version:
      out.version = self.version
    if self.path:
      out.dest_path = self.path

  def __str__(self):
    return '%s:%s' % (self.package_name, self.version)

  def _pre_put_hook(self):
    super(CipdPackage, self)._pre_put_hook()
    if not self.package_name:
      raise datastore_errors.BadValueError('CIPD package name is required')
    if not self.version:
      raise datastore_errors.BadValueError('CIPD package version is required')


class CipdInput(ndb.Model):
  """Specifies which CIPD client and packages to install, from which server.

  A part of TaskProperties.
  """
  # URL of the CIPD server. Must start with "https://" or "http://".
  server = ndb.StringProperty(indexed=False, validator=_validate_url)

  # CIPD package of CIPD client to use.
  # client_package.version is required.
  # client_package.path must be None.
  client_package = ndb.LocalStructuredProperty(CipdPackage)

  # List of packages to install.
  packages = ndb.LocalStructuredProperty(CipdPackage, repeated=True)

  def to_proto(self, out):
    """Converts self to a swarming_pb2.CIPDInputs."""
    if self.server:
      out.server = self.server
    for c in self.packages:
      dst = out.packages.add()
      c.to_proto(dst)

  def _pre_put_hook(self):
    if not self.server:
      raise datastore_errors.BadValueError('cipd server is required')
    if not self.client_package:
      raise datastore_errors.BadValueError('client_package is required')
    if self.client_package.path:
      raise datastore_errors.BadValueError('client_package.path must be unset')
    # _pre_put_hook() doesn't recurse correctly into
    # ndb.LocalStructuredProperty. Call the function manually.
    self.client_package._pre_put_hook()

    if not self.packages:
      raise datastore_errors.BadValueError(
          'cipd_input cannot have an empty package list')
    if len(self.packages) > 64:
      raise datastore_errors.BadValueError(
          'Up to 64 CIPD packages can be listed for a task')

    # Make sure we don't install multiple versions of the same package at the
    # same path.
    package_path_names = set()
    for p in self.packages:
      # _pre_put_hook() doesn't recurse correctly into
      # ndb.LocalStructuredProperty. Call the function manually.
      p._pre_put_hook()
      if not p.path:
        raise datastore_errors.BadValueError(
            'package %s:%s: path is required' % (p.package_name, p.version))
      path_name = (p.path, p.package_name)
      if path_name in package_path_names:
        raise datastore_errors.BadValueError(
           'package %r is specified more than once in path %r'
          % (p.package_name, p.path))
      package_path_names.add(path_name)
    self.packages.sort(key=lambda p: (p.path, p.package_name))


class CacheEntry(ndb.Model):
  """Describes a named cache that should be present on the bot."""
  name = ndb.StringProperty(validator=_validate_cache_name)
  path = ndb.StringProperty(validator=_validate_cache_path)

  def to_proto(self, out):
    """Converts self to a swarming_pb2.NamedCacheEntry."""
    out.name = self.name
    out.dest_path = self.path

  def _pre_put_hook(self):
    if not self.name:
      raise datastore_errors.BadValueError('name is not specified')


class TaskProperties(ndb.Model):
  """Defines all the properties of a task to be run on the Swarming
  infrastructure.

  This entity is not saved in the DB as a standalone entity, instead it is
  embedded in a TaskSlice.

  This model is immutable.

  New-style TaskProperties supports invocation of run_isolated. When this
  behavior is desired, the member .inputs_ref with an .isolated field value must
  be supplied. .extra_args can be supplied to pass extraneous arguments.
  """
  # TODO(maruel): convert inputs_ref and _TaskResultCommon.outputs_ref as:
  # - input = String which is the isolated input, if any
  # - isolated_server = <server, metadata e.g. namespace> which is a
  #   simplified version of FilesRef
  # - _TaskResultCommon.output = String which is isolated output, if any.

  caches = ndb.LocalStructuredProperty(CacheEntry, repeated=True)

  # Command to run. This overrides the command in the isolated file if any.
  command = ndb.StringProperty(repeated=True, indexed=False)

  # Relative working directory to run 'command' in, defaults to one specified
  # in an isolated file, if any, else the root mapped directory.
  relative_cwd = ndb.StringProperty(indexed=False)

  # Isolate server, namespace and input isolate hash.
  #
  # Despite its name, contains isolate server URL and namespace for isolated
  # output too. See TODO at the top of this class.
  # May be non-None even if task input is not isolated.
  #
  # Only inputs_ref.isolated or command can be specified.
  inputs_ref = ndb.LocalStructuredProperty(FilesRef)

  # CIPD packages to install.
  cipd_input = ndb.LocalStructuredProperty(CipdInput)

  # Filter to use to determine the required properties on the bot to run on. For
  # example, Windows or hostname. Encoded as json. 'pool' dimension is required
  # for all tasks except terminate (see _pre_put_hook).
  dimensions_data = datastore_utils.DeterministicJsonProperty(
      validator=_validate_dimensions, json_type=dict, indexed=False,
      name='dimensions')

  # Environment variables. Encoded as json. Optional.
  env = datastore_utils.DeterministicJsonProperty(
      validator=_validate_env, json_type=dict, indexed=False)

  # Environment path prefix variables. Encoded as json. Optional.
  #
  # Env key -> [list, of, rel, paths, to, prepend]
  env_prefixes = datastore_utils.DeterministicJsonProperty(
      validator=_validate_env_prefixes, json_type=dict, indexed=False)

  # Maximum duration the bot can take to run this task. It's named hard_timeout
  # in the bot.
  execution_timeout_secs = ndb.IntegerProperty(
      validator=_validate_timeout, required=True, indexed=False)

  # Extra arguments to supply to the command `python run_isolated ...`. Can only
  # be set if inputs_ref.isolated is set.
  extra_args = ndb.StringProperty(repeated=True, indexed=False)

  # Grace period is the time between signaling the task it timed out and killing
  # the process. During this time the process should clean up itself as quickly
  # as possible, potentially uploading partial results back.
  grace_period_secs = ndb.IntegerProperty(
      validator=_validate_grace, default=30, indexed=False)

  # Bot controlled timeout for new bytes from the subprocess. If a subprocess
  # doesn't output new data to stdout for .io_timeout_secs, consider the command
  # timed out. Optional.
  io_timeout_secs = ndb.IntegerProperty(
      validator=_validate_timeout, indexed=False)

  # If True, the task can safely be served results from a previously succeeded
  # task.
  idempotent = ndb.BooleanProperty(default=False, indexed=False)

  # A list of outputs expected. If empty, all files written to
  # $(ISOLATED_OUTDIR) will be returned; otherwise, the files in this list
  # will be added to those in that directory.
  outputs = ndb.StringProperty(repeated=True, indexed=False,
      validator=_validate_output_path)

  # If True, the TaskRequest embedding these TaskProperties has an associated
  # SecretBytes entity.
  has_secret_bytes = ndb.BooleanProperty(default=False, indexed=False)

  @property
  def pool(self):
    """Returns the pool that this TaskProperties has in dimensions, or None if
    no pool dimension exists."""
    return self.dimensions.get('pool', [None])[0]

  @property
  def dimensions(self):
    """Returns dimensions as a dict(unicode, list(unicode)), even for older
    entities.
    """
    # Just look at the first one. The property is guaranteed to be internally
    # consistent.
    data = self.dimensions_data or {}
    for v in data.itervalues():
      if isinstance(v, (list, tuple)):
        return self.dimensions_data
      break
    # Compatibility code for old entities.
    return {k: [v] for k, v in data.iteritems()}

  @property
  def is_terminate(self):
    """If True, it is a terminate request."""
    # Check dimensions last because it's a bit slower.
    return (
        not self.caches and
        not self.command and
        not (self.inputs_ref and self.inputs_ref.isolated) and
        not self.cipd_input and
        not self.env and
        not self.env_prefixes and
        not self.execution_timeout_secs and
        not self.extra_args and
        not self.grace_period_secs and
        not self.io_timeout_secs and
        not self.idempotent and
        not self.outputs and
        not self.has_secret_bytes and
        self.dimensions_data.keys() == [u'id'])

  def to_dict(self):
    out = super(TaskProperties, self).to_dict(
        exclude=['dimensions_data'])
    # Use the data stored as-is, so properties_hash doesn't change.
    out['dimensions'] = self.dimensions_data
    return out

  def to_proto(self, out):
    """Converts self to a swarming_pb2.TaskProperties."""
    if self.inputs_ref:
      self.inputs_ref.to_proto(out.cas_inputs)
    if self.cipd_input:
      # It's possible for self.cipd_input to be None.
      for c in self.cipd_input.packages:
        dst = out.cipd_inputs.add()
        c.to_proto(dst)
    for c in self.caches:
      dst = out.named_caches.add()
      c.to_proto(dst)
    if self.command:
      out.command.extend(self.command)
    if self.relative_cwd:
      out.relative_cwd = self.relative_cwd
    if self.extra_args:
      out.extra_args.extend(self.extra_args)
    out.has_secret_bytes = self.has_secret_bytes
    for key, values in sorted(self.dimensions.iteritems()):
      v = out.dimensions.add()
      v.key = key
      v.values.extend(sorted(values))
    for key, value in sorted((self.env or {}).iteritems()):
      v = out.env.add()
      v.key = key
      v.value = value
    for key, values in sorted((self.env_prefixes or {}).iteritems()):
      v = out.env_paths.add()
      v.key = key
      v.values.extend(sorted(values))
    # TODO(maruel): Define containment; https://crbug.com/808836
    if self.execution_timeout_secs:
      out.execution_timeout.seconds = self.execution_timeout_secs
    if self.io_timeout_secs:
      out.io_timeout.seconds = self.io_timeout_secs
    if self.grace_period_secs:
      out.grace_period.seconds = self.grace_period_secs
    out.idempotent = self.idempotent
    if self.outputs:
      out.outputs.extend(self.outputs)

  def _pre_put_hook(self):
    super(TaskProperties, self)._pre_put_hook()
    if self.is_terminate:
      # Most values are not valid with a terminate task. self.is_terminate
      # already check those. Terminate task can only use 'id'.
      return

    if u'pool' not in self.dimensions_data:
      # Only terminate task may not use 'pool'. Others must specify one.
      raise datastore_errors.BadValueError(
          u'\'pool\' must be used as dimensions')

    # Isolated input and commands.
    isolated_input = self.inputs_ref and self.inputs_ref.isolated
    if not self.command and not isolated_input:
      raise datastore_errors.BadValueError(
          'use at least one of command or inputs_ref.isolated')
    if self.command and self.extra_args:
      raise datastore_errors.BadValueError(
          'can\'t use both command and extra_args')
    if self.extra_args and not isolated_input:
      raise datastore_errors.BadValueError(
          'extra_args require inputs_ref.isolated')
    if self.inputs_ref:
      # _pre_put_hook() doesn't recurse correctly into
      # ndb.LocalStructuredProperty. Call the function manually.
      self.inputs_ref._pre_put_hook()
    if len(self.command) > 128:
      raise datastore_errors.BadValueError(
          'command can have up to 128 arguments')
    if len(self.extra_args) > 128:
      raise datastore_errors.BadValueError(
          'extra_args can have up to 128 arguments')

    # Validate caches.
    if len(self.caches) > 32:
      raise datastore_errors.BadValueError(
          'Up to 64 caches can be listed for a task')
    cache_names = set()
    cache_paths = set()
    for c in self.caches:
      # _pre_put_hook() doesn't recurse correctly into
      # ndb.LocalStructuredProperty. Call the function manually.
      c._pre_put_hook()
      if c.name in cache_names:
        raise datastore_errors.BadValueError(
            'Cache name %s is used more than once' % c.name)
      if c.path in cache_paths:
        raise datastore_errors.BadValueError(
            'Cache path "%s" is mapped more than once' % c.path)
      cache_names.add(c.name)
      cache_paths.add(c.path)
    self.caches.sort(key=lambda c: c.name)

    # Validate CIPD Input.
    if self.cipd_input:
      # _pre_put_hook() doesn't recurse correctly into
      # ndb.LocalStructuredProperty. Call the function manually.
      self.cipd_input._pre_put_hook()
      for p in self.cipd_input.packages:
        if p.path in cache_paths:
          raise datastore_errors.BadValueError(
              'Path "%s" is mapped to a named cache and cannot be a target '
              'of CIPD installation' % p.path)
      if self.idempotent:
        pinned = lambda p: cipd.is_pinned_version(p.version)
        assert self.cipd_input.packages  # checked by cipd_input._pre_put_hook
        if any(not pinned(p) for p in self.cipd_input.packages):
          raise datastore_errors.BadValueError(
              'an idempotent task cannot have unpinned packages; '
              'use tags or instance IDs as package versions')

    if len(self.outputs) > 4096:
      raise datastore_errors.BadValueError(
          'Up to 4096 outputs can be listed for a task')


class TaskSlice(ndb.Model):
  """Defines all the various possible sets of properties that a task request
  will use; the task will fallback from one slice to the next until it finds a
  matching bot.

  This entity is not saved in the DB as a standalone entity, instead it is
  embedded in a TaskRequest.

  This model is immutable.
  """
  # Hashing algorithm used to hash TaskProperties to create its key.
  HASHING_ALGO = hashlib.sha256

  # The actual properties are embedded in this model.
  properties = ndb.LocalStructuredProperty(TaskProperties, required=True)
  # If this task request slice is not scheduled by this moment, the next one
  # will be processed.
  expiration_secs = ndb.IntegerProperty(
      validator=_validate_expiration_secs, required=True)

  # When a task is scheduled and there are currently no bots available to run
  # the task, the TaskSlice can either be PENDING, or be denied immediately.
  # When denied, the next TaskSlice is enqueued, and if there's no following
  # TaskSlice, the task state is set to NO_RESOURCE. This should normally be
  # set to False to avoid unnecessary waiting.
  wait_for_capacity = ndb.BooleanProperty(default=False)

  def properties_hash(self, request):
    """Calculates the properties_hash for this request, if applicable.

    Note: if the property has secret bytes, this function call causes a DB GET.
    """
    if not self.properties.idempotent:
      return None
    return self._properties_hash_raw(request).digest()

  def _properties_hash_raw(self, request):
    """Calculates the properties_hash for this request."""
    props = self.properties.to_dict()
    if self.properties.has_secret_bytes:
      # When called from task_scheduler.schedule_task(), this function is called
      # in the same context that stored the SecretBytes entity, so the entity is
      # still in the in process cache.
      #
      # When called in the context of an idempotent TaskRunResult that is
      # COMPLETED with success, this is much more costly since this happens
      # inside a transaction.
      s = task_pack.request_key_to_secret_bytes_key(request.key).get()
      if s:
        props['secret_bytes'] = s.secret_bytes.encode('hex')
      else:
        # A TaskRequest is broken if the corresponding SecretBytes is not
        # present. Tolerate it here but log a warning.
        logging.warning('%s is broken; SecretBytes is missing', request.task_id)
    return self.HASHING_ALGO(utils.encode_to_json(props))

  def to_dict(self):
    # to_dict() doesn't recurse correctly into ndb.LocalStructuredProperty! It
    # will call the default method and not the overridden one. :(
    out = super(TaskSlice, self).to_dict(exclude=['properties'])
    out['properties'] = self.properties.to_dict()
    return out

  def to_proto(self, out, request):
    """Converts self to a swarming_pb2.TaskSlice."""
    if self.properties:
      self.properties.to_proto(out.properties)
      out.properties_hash = self._properties_hash_raw(request).hexdigest()
    out.wait_for_capacity = self.wait_for_capacity
    if self.expiration_secs:
      out.expiration.seconds = self.expiration_secs

  def _pre_put_hook(self):
    # _pre_put_hook() doesn't recurse correctly into
    # ndb.LocalStructuredProperty. Call the function manually.
    super(TaskSlice, self)._pre_put_hook()
    self.properties._pre_put_hook()
    if self.wait_for_capacity is None:
      raise datastore_errors.BadValueError('wait_for_capacity is required')


class TaskRequest(ndb.Model):
  """Contains a user request.

  Key id is a decreasing integer based on time since utils.EPOCH plus some
  randomness on lower order bits. See new_request_key() for the complete gory
  details.

  This model is immutable.
  """
  # Time this request was registered. It is set manually instead of using
  # auto_now_add=True so that expiration_ts can be set very precisely relative
  # to this property.
  created_ts = ndb.DateTimeProperty(required=True)

  ## What

  # The TaskSlice describes what to run. When the list has more than one item,
  # this is to enable task fallback.
  task_slices = ndb.LocalStructuredProperty(
      TaskSlice, compressed=True, repeated=True)
  # Old way of specifying task properties. Only one of properties or
  # task_slices can be set.
  properties_old = ndb.LocalStructuredProperty(
      TaskProperties, compressed=True, name='properties')

  # If the task request is not scheduled by this moment, it will be aborted by a
  # cron job. It is saved instead of scheduling_expiration_secs so finding
  # expired jobs is a simple query.
  #
  # When task_slices is used, this value is the same as
  # self.task_slices[-1].expiration_ts.
  expiration_ts = ndb.DateTimeProperty(indexed=True, required=True)

  ## Why and other contexts

  # The name for this task request. It's only for description.
  name = ndb.StringProperty(required=True)

  # Authenticated client that triggered this task.
  authenticated = auth.IdentityProperty()

  # Which user to blame for this task. Can be arbitrary, not asserted by any
  # credentials.
  user = ndb.StringProperty(default='')

  # Indicates what OAuth2 credentials the task uses when calling other services.
  #
  # Possible values are: 'none', 'bot' or <email>. For more information see
  # swarming_rpcs.NewTaskRequest.
  #
  # This property exists only for informational purposes and for indexing. When
  # actually getting an OAuth credentials, the properly signed OAuth grant token
  # (stored in hidden 'service_account_token' field) is used.
  service_account = ndb.StringProperty(validator=_validate_service_account)

  # The "OAuth token grant" generated when the task was posted.
  #
  # This is an opaque token generated by the Token Server at the time the task
  # was posted (when the end-user is still present). It can be exchanged
  # for an OAuth token of some service account at a later time (when the task is
  # actually running on some bot).
  #
  # This property never shows up in UI or API responses.
  service_account_token = ndb.BlobProperty()

  # Priority of the task to be run. A lower number is higher priority, thus will
  # preempt requests with lower priority (higher numbers).
  priority = ndb.IntegerProperty(
      indexed=False, validator=_validate_priority, required=True)

  # Tags that specify the category of the task. This property contains both the
  # tags specified by the user and the tags for every TaskSlice.
  tags = ndb.StringProperty(repeated=True, validator=_validate_tags)
  # Tags that are provided by the user. This is used to regenerate the list of
  # tags for TaskResultSummary based on the actual TaskSlice used.
  manual_tags = ndb.StringProperty(
      repeated=True, validator=_validate_tags, indexed=False)

  # Set when a task (the parent) reentrantly create swarming tasks. Must be set
  # to a valid task_id pointing to a TaskRunResult or be None.
  parent_task_id = ndb.StringProperty(validator=_validate_task_run_id)

  # PubSub topic to send task completion notification to.
  pubsub_topic = ndb.StringProperty(
      indexed=False, validator=_validate_pubsub_topic)

  # Secret token to send as 'auth_token' attribute with PubSub messages.
  pubsub_auth_token = ndb.StringProperty(indexed=False)

  # Data to send in 'userdata' field of PubSub messages.
  pubsub_userdata = ndb.StringProperty(
      indexed=False, validator=_get_validate_length(1024))

  @property
  def num_task_slices(self):
    """Returns the number of TaskSlice, supports old entities."""
    if self.properties_old:
      return 1
    return len(self.task_slices)

  def task_slice(self, index):
    """Returns the TaskSlice at this index, supports old entities."""
    if self.properties_old:
      assert index == 0, index
      t = TaskSlice(
          properties=self.properties_old, expiration_secs=self.expiration_secs)
    else:
      t = self.task_slices[index]
    return t

  @property
  def pool(self):
    # all task_slices must have the same pool, and we must have at least one
    # task slice, so just return the 0th's pool.
    return self.task_slice(0).properties.pool

  @property
  def secret_bytes_key(self):
    if self.properties_old:
      if self.properties_old.has_secret_bytes:
        return task_pack.request_key_to_secret_bytes_key(self.key)
    else:
      for t in self.task_slices:
        if t.properties.has_secret_bytes:
          return task_pack.request_key_to_secret_bytes_key(self.key)

  @property
  def task_id(self):
    """Returns the TaskResultSummary packed id, not the task request key."""
    return task_pack.pack_result_summary_key(
        task_pack.request_key_to_result_summary_key(self.key))

  @property
  def expiration_secs(self):
    """Reconstructs this value from expiration_ts and created_ts. Integer."""
    return int((self.expiration_ts - self.created_ts).total_seconds())

  @property
  def max_lifetime_secs(self):
    """Calculates the maximum latency at which the task may still be running
    user code.
    """
    max_lifetime_secs = 0
    offset = 0
    for i in xrange(self.num_task_slices):
      t = self.task_slice(i)
      offset += t.expiration_secs
      props = t.properties
      mls = offset + props.execution_timeout_secs + props.grace_period_secs
      if mls > max_lifetime_secs:
        max_lifetime_secs = mls
    return max_lifetime_secs

  def to_dict(self):
    """Supports both old and new format."""
    # to_dict() doesn't recurse correctly into ndb.LocalStructuredProperty! It
    # will call the default method and not the overiden one. :(
    out = super(TaskRequest, self).to_dict(
        exclude=['manual_tags', 'properties_old', 'pubsub_auth_token',
                 'service_account_token', 'task_slice'])
    if self.properties_old:
      out['properties'] = self.properties_old.to_dict()
    if self.task_slices:
      out['task_slices'] = [t.to_dict() for t in self.task_slices]
    return out

  def to_proto(self, out):
    """Converts self to a swarming_pb2.TaskRequest."""
    # Scheduling.
    for task_slice in self.task_slices:
      t = out.task_slices.add()
      task_slice.to_proto(t, self)
    if self.priority:
      out.priority = self.priority
    if self.service_account:
      out.service_account = self.service_account

    # Information.
    if self.created_ts:
      out.create_time.FromDatetime(self.created_ts)
    if self.name:
      out.name = self.name
    out.tags.extend(self.tags)
    if self.user:
      out.user = self.user

    # Hierarchy and notifications.
    if self.key:
      # The task_id can only be set if the TaskRequest entity has a valid key.
      out.task_id = self.task_id
    if self.parent_task_id:
      out.parent_run_id = self.parent_task_id
      out.parent_task_id = self.parent_task_id[:-1] + '0'
    if self.pubsub_topic:
      out.pubsub_notification.topic = self.pubsub_topic
    # self.pubsub_auth_token cannot be retrieved.
    if self.pubsub_userdata:
      out.pubsub_notification.userdata = self.pubsub_userdata

  def _pre_put_hook(self):
    super(TaskRequest, self)._pre_put_hook()
    if self.properties_old:
      raise datastore_errors.BadValueError(
          'old style TaskRequest.properties is not supported anymore')
    if not self.task_slices:
      raise datastore_errors.BadValueError('task_slices is missing')
    if len(self.task_slices) > 8:
      # The objects are large so use a low limit to start, and increase if
      # there's user request.
      raise datastore_errors.BadValueError(
          'A maximum of 8 task_slices is supported')
    for tslice in self.task_slices:
      # _pre_put_hook() doesn't recurse correctly into
      # ndb.LocalStructuredProperty. Call the function manually.
      tslice._pre_put_hook()

    terminate_count = sum(
        1 for t in self.task_slices if t.properties.is_terminate)
    if terminate_count > 1 or (terminate_count and len(self.task_slices) > 1):
      # Revisit this if this becomes a use case, e.g. "try to run this,
      # otherwise terminate the bot". In any case, terminate must be last.
      raise datastore_errors.BadValueError(
          'terminate request must be used alone')
    if terminate_count:
      if not self.priority == 0:
        raise datastore_errors.BadValueError(
            'terminate request must be priority 0')
    else:
      if self.priority == 0:
        raise datastore_errors.BadValueError(
            'priority 0 can only be used for terminate request')

    if len(self.task_slices) > 1:
      # Make sure there is no duplicate task. It is likely an error from the
      # user. Compare dictionary so it works even if idempotent is False.
      num_unique = len(set(
          utils.encode_to_json(t.properties.to_dict())
          for t in self.task_slices))
      if len(self.task_slices) != num_unique:
        raise datastore_errors.BadValueError(
            'cannot request duplicate task slice')

    # All task slices in a single TaskRequest must use the exact same 'pool'.
    # It is fine to use different 'id' values; one bot is targetting in the
    # first TaskSlice, a second bot is targetted as a fallback on the second
    # TaskSlice.
    v = self.task_slice(0).properties.dimensions.get(u'pool')
    for i in xrange(1, self.num_task_slices):
      t = self.task_slice(i)
      w = t.properties.dimensions.get(u'pool')
      if v != w:
        raise datastore_errors.BadValueError(
            u'each task slice must use the same pool dimensions; %s != %s' %
            (v, w))

    if len(self.manual_tags) > 256:
      raise datastore_errors.BadValueError(
          'up to 256 tags can be specified for a task request')

    if (self.pubsub_topic and
        not pubsub.validate_full_name(self.pubsub_topic, 'topics')):
      raise datastore_errors.BadValueError(
          'bad pubsub topic name - %s' % self.pubsub_topic)
    if self.pubsub_auth_token and not self.pubsub_topic:
      raise datastore_errors.BadValueError(
          'pubsub_auth_token requires pubsub_topic')
    if self.pubsub_userdata and not self.pubsub_topic:
      raise datastore_errors.BadValueError(
          'pubsub_userdata requires pubsub_topic')


### Private stuff.


def _get_automatic_tags(request):
  """Returns tags that should automatically be added to the TaskRequest.

  This includes geneated tags from all TaskSlice.
  """
  tags = set((
    u'priority:%s' % request.priority,
    u'service_account:%s' % (request.service_account or u'None'),
    u'user:%s' % (request.user or u'None'),
  ))
  for i in xrange(request.num_task_slices):
    for key, values in request.task_slice(i).properties.dimensions.iteritems():
      for value in values:
        tags.add(u'%s:%s' % (key, value))
  return tags


### Public API.


def get_automatic_tags(request, index):
  """Returns tags that should automatically be added to the TaskRequest for one
  specific TaskSlice.
  """
  tags = set((
    u'priority:%s' % request.priority,
    u'service_account:%s' % (request.service_account or u'None'),
    u'user:%s' % (request.user or u'None'),
  ))
  for key, values in request.task_slice(
      index).properties.dimensions.iteritems():
    for value in values:
      tags.add(u'%s:%s' % (key, value))
  return tags


def create_termination_task(bot_id, wait_for_capacity):
  """Returns a task to terminate the given bot.

  ACL check must have been done before.

  Returns:
    TaskRequest for priority 0 (highest) termination task.
  """
  properties = TaskProperties(
      dimensions_data={u'id': [unicode(bot_id)]},
      execution_timeout_secs=0,
      grace_period_secs=0,
      io_timeout_secs=0)
  now = utils.utcnow()
  request = TaskRequest(
      created_ts=now,
      expiration_ts=now + datetime.timedelta(days=1),
      name=u'Terminate %s' % bot_id,
      priority=0,
      task_slices=[
        TaskSlice(
            expiration_secs=24*60*60,
            properties=properties,
            wait_for_capacity=wait_for_capacity),
      ],
      manual_tags=[u'terminate:1'])
  assert request.task_slice(0).properties.is_terminate
  init_new_request(request, True, TEMPLATE_SKIP)
  return request


def new_request_key():
  """Returns a valid ndb.Key for this entity.

  Task id is a 64 bits integer represented as a string to the user:
  - 1 highest order bits set to 0 to keep value positive.
  - 43 bits is time since _BEGINING_OF_THE_WORLD at 1ms resolution.
    It is good for 2**43 / 365.3 / 24 / 60 / 60 / 1000 = 278 years or 2010+278 =
    2288. The author will be dead at that time.
  - 16 bits set to a random value or a server instance specific value. Assuming
    an instance is internally consistent with itself, it can ensure to not reuse
    the same 16 bits in two consecutive requests and/or throttle itself to one
    request per millisecond.
    Using random value reduces to 2**-15 the probability of collision on exact
    same timestamp at 1ms resolution, so a maximum theoretical rate of 65536000
    requests/sec but an effective rate in the range of ~64k requests/sec without
    much transaction conflicts. We should be fine.
  - 4 bits set to 0x1. This is to represent the 'version' of the entity schema.
    Previous version had 0. Note that this value is XOR'ed in the DB so it's
    stored as 0xE. When the TaskRequest entity tree is modified in a breaking
    way that affects the packing and unpacking of task ids, this value should be
    bumped.

  The key id is this value XORed with task_pack.TASK_REQUEST_KEY_ID_MASK. The
  reason is that increasing key id values are in decreasing timestamp order.
  """
  # TODO(maruel): Use real randomness.
  suffix = random.getrandbits(16)
  return convert_to_request_key(utils.utcnow(), suffix)


def request_key_to_datetime(request_key):
  """Converts a TaskRequest.key to datetime.

  See new_request_key() for more details.
  """
  if request_key.kind() != 'TaskRequest':
    raise ValueError('Expected key to TaskRequest, got %s' % request_key.kind())
  # Ignore lowest 20 bits.
  xored = request_key.integer_id() ^ task_pack.TASK_REQUEST_KEY_ID_MASK
  offset_ms = (xored >> 20) / 1000.
  return _BEGINING_OF_THE_WORLD + datetime.timedelta(seconds=offset_ms)


def datetime_to_request_base_id(now):
  """Converts a datetime into a TaskRequest key base value.

  Used for query order().
  """
  if now < _BEGINING_OF_THE_WORLD:
    raise ValueError(
        'Time %s is set to before %s' % (now, _BEGINING_OF_THE_WORLD))
  delta = now - _BEGINING_OF_THE_WORLD
  return int(round(delta.total_seconds() * 1000.)) << 20


def convert_to_request_key(date, suffix=0):
  assert 0 <= suffix <= 0xffff
  request_id_base = datetime_to_request_base_id(date)
  return request_id_to_key(int(request_id_base | suffix << 4 | 0x1))


def request_id_to_key(request_id):
  """Converts a request id into a TaskRequest key.

  Note that this function does NOT accept a task id. This functions is primarily
  meant for limiting queries to a task creation range.
  """
  return ndb.Key(TaskRequest, request_id ^ task_pack.TASK_REQUEST_KEY_ID_MASK)


def validate_request_key(request_key):
  if request_key.kind() != 'TaskRequest':
    raise ValueError('Expected key to TaskRequest, got %s' % request_key.kind())
  task_id = request_key.integer_id()
  if not task_id:
    raise ValueError('Invalid null TaskRequest key')
  if (task_id & 0xF) == 0xE:
    # New style key.
    return

  # Check the shard.
  # TODO(maruel): Remove support 2015-02-01.
  request_shard_key = request_key.parent()
  if not request_shard_key:
    raise ValueError('Expected parent key for TaskRequest, got nothing')
  if request_shard_key.kind() != 'TaskRequestShard':
    raise ValueError(
        'Expected key to TaskRequestShard, got %s' % request_shard_key.kind())
  root_entity_shard_id = request_shard_key.string_id()
  if (not root_entity_shard_id or
      len(root_entity_shard_id) != task_pack.DEPRECATED_SHARDING_LEVEL):
    raise ValueError(
        'Expected root entity key (used for sharding) to be of length %d but '
        'length was only %d (key value %r)' % (
            task_pack.DEPRECATED_SHARDING_LEVEL,
            len(root_entity_shard_id or ''),
            root_entity_shard_id))


def _select_task_template(pool, template_apply):
  """Selects the task template to apply from the given pool config.

  Args:
    pool (str) - The name of the pool to select a task template from.
    template_apply (swarming_rpcs.PoolTaskTemplate) - The PoolTaskTemplate
      enum controlling application of the deployment.

  Returns:
    (TaskTemplate, extra_tags) if there's a template to apply or else
    (None, extra_tags).
  """
  if not pool:
    # It shouldn't actually be possible, but here for consistency so that tasks
    # always get a swarming.pool.template tag with SOME value.
    return None, ('swarming.pool.template:no_pool',)

  assert isinstance(template_apply, TemplateApplyEnum)

  pool_cfg = pools_config.get_pool_config(pool)
  if not pool_cfg:
    return None, ('swarming.pool.template:no_config',)

  tags = ('swarming.pool.version:%s' % (pool_cfg.rev,),)

  if template_apply == TEMPLATE_SKIP:
    tags += ('swarming.pool.template:skip',)
    return None, tags

  deployment = pool_cfg.task_template_deployment
  if not deployment:
    tags += ('swarming.pool.template:none',)
    return None, tags

  canary = deployment.canary and (
    (template_apply == TEMPLATE_CANARY_PREFER)
    or (template_apply == TEMPLATE_AUTO and
        random.randint(1, 9999) < deployment.canary_chance)
  )
  to_apply = deployment.canary if canary else deployment.prod

  tags += ('swarming.pool.template:%s' % ('canary' if canary else 'prod'),)
  return to_apply, tags


def _apply_task_template(task_template, props):
  """Applies this template to the indicated properties.

  Modifies `props` in-place.

  Args:
    task_template (pools_config.TaskTemplate|None) - The template to apply. If
      None, then this function returns without modifying props.
    props (TaskProperties) - The task properties to modify.
  """
  if task_template is None:
    return

  assert isinstance(task_template, pools_config.TaskTemplate)
  assert isinstance(props, TaskProperties)

  for envvar in task_template.env:
    var_name = envvar.var

    if not envvar.soft:
      if var_name in (props.env or {}):
        raise ValueError(
            'request.env[%r] conflicts with pool\'s template' % var_name)
      if var_name in (props.env_prefixes or {}):
        raise ValueError(
            'request.env_prefixes[%r] conflicts with pool\'s template'
            % var_name)

    if envvar.value:
      props.env = props.env or {}
      props.env[var_name] = props.env.get(var_name, '') or envvar.value

    if envvar.prefix:
      props.env_prefixes = props.env_prefixes or {}
      props.env_prefixes[var_name] = (
          list(envvar.prefix) + props.env_prefixes.get(var_name, []))

  reserved_cache_names = set()
  occlude_checker = directory_occlusion.Checker()
  # Add all task template paths.
  for cache in task_template.cache:
    reserved_cache_names.add(cache.name)
    occlude_checker.add(cache.path, 'task template cache %r' % cache.name, '')
  for cp in task_template.cipd_package:
    occlude_checker.add(
        cp.path, 'task template cipd', '%s:%s' % (cp.pkg, cp.version))

  # Add all task paths, avoiding spurious initializations in the underlying
  # TaskProperties (repeated fields auto-initialize to [] when looped over).
  for cache in (props.caches or ()):
    if cache.name in reserved_cache_names:
      raise ValueError(
          'request.cache[%r] conflicts with pool\'s template' % cache.name)
    occlude_checker.add(cache.path, 'task cache %r' % cache.name, '')
  for cp in (props.cipd_input.packages or () if props.cipd_input else ()):
    occlude_checker.add(
        cp.path, 'task cipd', '%s:%s' % (cp.package_name, cp.version))

  ctx = validation.Context()
  if occlude_checker.conflicts(ctx):
    raise ValueError('\n'.join(m.text for m in ctx.result().messages))

  for cache in task_template.cache:
    props.caches.append(CacheEntry(name=cache.name, path=cache.path))

  if task_template.cipd_package:
    # Only initialize TaskProperties.cipd_input if we have something to add
    props.cipd_input = props.cipd_input or CipdInput()
    for cp in task_template.cipd_package:
      props.cipd_input.packages.append(CipdPackage(
          package_name=cp.pkg, path=cp.path, version=cp.version))


def init_new_request(request, allow_high_priority, template_apply):
  """Initializes a new TaskRequest but doesn't store it.

  ACL check must have been done before, except for high priority task.

  Fills up some values and does minimal checks.

  If parent_task_id is set, properties for the parent are used:
  - priority: defaults to parent.priority - 1
  - user: overridden by parent.user

  template_apply must be one of the TEMPLATE_* singleton values above.
  """
  assert request.__class__ is TaskRequest, request
  if not request.num_task_slices:
    raise ValueError('Either properties or task_slices must be provided')

  if request.parent_task_id:
    # Note that if the child task is _likely_ blocking the parent task, unless
    # the parent task did a fire-and-forget. Make sure to setup the priority and
    # pool allocation accordingly.
    run_result_key = task_pack.unpack_run_result_key(request.parent_task_id)
    result_summary_key = task_pack.run_result_key_to_result_summary_key(
        run_result_key)
    request_key = task_pack.result_summary_key_to_request_key(
        result_summary_key)
    parent = request_key.get()
    # Terminate request can only be requested as a single TaskProperties.
    if not parent or parent.task_slice(0).properties.is_terminate:
      raise ValueError('parent_task_id is not a valid task')
    # Drop the previous user.
    request.user = parent.user

  # If the priority is below 20, make sure the user has right to do so.
  if request.priority < 20 and not allow_high_priority:
    # Special case for terminate request.
    # Terminate request can only be requested as a single TaskProperties.
    if not request.task_slice(0).properties.is_terminate:
      # Silently drop the priority of normal users.
      request.priority = 20

  request.authenticated = auth.get_current_identity()

  # Convert None to 'none', to make it indexable. Here request.service_account
  # can be 'none', 'bot' or an <email>. When using <email>, callers of
  # 'init_new_request' are expected to generate new service account token
  # (by making an RPC to the token server) and put it into service_account_token
  # before storing it.
  request.service_account = request.service_account or u'none'
  request.service_account_token = None

  task_template, extra_tags = _select_task_template(
      request.pool, template_apply)

  if request.task_slices:
    exp = 0
    for t in request.task_slices:
      if not t.expiration_secs:
        raise ValueError('missing expiration_secs')
      exp += t.expiration_secs
      _apply_task_template(task_template, t.properties)
    # Always clobber the overall value.
    # message_conversion.new_task_request_from_rpc() ensures both task_slices
    # and expiration_secs cannot be used simultaneously.
    request.expiration_ts = request.created_ts + datetime.timedelta(seconds=exp)

  # This is useful to categorize the task.
  all_tags = set(request.manual_tags).union(_get_automatic_tags(request))
  all_tags.update(extra_tags)
  request.tags = sorted(all_tags)


def validate_priority(priority):
  """Throws ValueError if priority is not a valid value."""
  if 0 > priority or MAXIMUM_PRIORITY < priority:
    raise datastore_errors.BadValueError(
        'priority (%d) must be between 0 and %d (inclusive)' %
        (priority, MAXIMUM_PRIORITY))


def cron_delete_old_task_requests():
  """Deletes very old TaskRequest entities and their children entities.

  This function doesn't really delete the entities, instead of collect the
  task_id for each of them, and trigger batches of deletion to a task queue.

  This is needed because the rate of deletion is slower than the incoming rate,
  so we need to delete batches of old TaskRequest in parallel on instances with
  high utilization.
  """
  start = utils.utcnow()
  # Run for 4.5 minutes and schedule the cron job every 5 minutes. Running for
  # 9.5 minutes (out of 10 allowed for a cron job) results in 'Exceeded soft
  # private memory limit of 512 MB with 512 MB' even if this loop should be
  # fairly light on memory usage.
  time_to_stop = start + datetime.timedelta(seconds=int(4.5*60))
  # Total TaskRequest entities processed
  total = 0
  # GAE tasks queues that were created to do the actual deletion.
  tasks_succeeded = 0
  tasks_failed = 0
  end_ts = start - _OLD_TASK_REQUEST_CUT_OFF
  first = None
  last = None
  try:
    # Key ordering is by most recent first. We want the reverse, delete the
    # oldest first. That would require ordering by -TaskRequest.key, which would
    # require a new composite index. We don't want that. So instead use
    # .created_ts directly, which is ordered by what is needed.
    opt = ndb.QueryOptions(use_cache=False, use_memcache=False, keys_only=True)
    # Using a keys_only request is eventually consistent, which is normally
    # risky. Here this is fine because this is year+ old entities, so the index
    # should be consistent. :)
    q = TaskRequest.query(default_options=opt).filter(
        TaskRequest.created_ts <= end_ts)
    cursor = None
    while utils.utcnow() <= time_to_stop:
      keys, cursor, more = q.fetch_page(
          _TASKS_DELETE_CHUNK_SIZE, start_cursor=cursor)
      if not keys:
        break
      total += len(keys)
      data = {u'task_ids': [task_pack.pack_request_key(k) for k in keys]}
      if not first:
        first = keys[0]
      last = keys[-1]
      ok = utils.enqueue_task(
          '/internal/taskqueue/cleanup/tasks/delete',
          'delete-tasks',
          payload=utils.encode_to_json(data))
      if not ok:
        logging.info('Failed to enqueue %d tasks for deletion', len(keys))
        tasks_failed += 1
      else:
        tasks_succeeded += 1
      if not more:
        break
  finally:
    first_ts = request_key_to_datetime(first) if first else None
    last_ts = request_key_to_datetime(last) if last else None

    def _format_ts(t):
      # datetime.datetime
      return t.strftime(u'%Y-%m-%d %H:%M') if t else 'N/A'

    def _format_delta(e, s):
      # datetime.timedelta
      return str(e-s).rsplit('.', 1)[0] if e and s else 'N/A'

    logging.info(
        'Found %d TaskRequest entities to delete. %d tasks triggered;'
            ' %d failed\n'
        'From %s to %s (%s)\n'
        'Cut off was %s; trailing by %s',
        total, tasks_succeeded, tasks_failed,
        _format_ts(first_ts), _format_ts(last_ts),
        _format_delta(last_ts, first_ts),
        _format_ts(end_ts), _format_delta(end_ts, last_ts))
  return total


def task_delete_tasks(task_ids):
  """Deletes the specified tasks, a list of string encoded task ids."""
  total = 0
  count = 0
  opt = ndb.QueryOptions(use_cache=False, use_memcache=False, keys_only=True)
  try:
    for task_id in task_ids:
      request_key = task_pack.unpack_request_key(task_id)
      # Delete the whole group. An ancestor query will retrieve the entity
      # itself too, so no need to explicitly delete it.
      keys = ndb.Query(default_options=opt, ancestor=request_key).fetch()
      if not keys:
        # Can happen if it is a retry.
        continue
      ndb.delete_multi(keys)
      total += len(keys)
      count += 1
    return count
  finally:
    logging.info(
        'Deleted %d TaskRequest groups; %d entities in total', count, total)


def task_bq(start, end):
  """Sends TaskRequest to BigQuery swarming.task_requests table."""
  def _convert(e):
    """Returns a tuple(bq_key, row)."""
    out = swarming_pb2.TaskRequest()
    e.to_proto(out)
    return (e.task_id, out)

  total = 0
  failed = 0

  q = TaskRequest.query(
      TaskRequest.created_ts >= start, TaskRequest.created_ts <= end)
  cursor = None
  more = True
  while more:
    entities, cursor, more = q.fetch_page(500, start_cursor=cursor)
    total += len(entities)
    failed += bq_state.send_to_bq(
        'task_requests', [_convert(e) for e in entities])
  return total, failed
