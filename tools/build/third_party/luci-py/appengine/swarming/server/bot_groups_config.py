# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Functions to fetch and interpret bots.cfg file with list of bot groups."""

import ast
import collections
import hashlib
import logging
import os
import threading

from google.appengine.ext import ndb

from components import auth
from components import config
from components import machine_provider
from components import utils
from components.config import validation

from proto.config import bots_pb2
from server import config as local_config


BOTS_CFG_FILENAME = 'bots.cfg'


# Validated and "frozen" bots_pb2.BotAuth proto, see its doc for meaning of
# fields.
BotAuth = collections.namedtuple('BotAuth', [
  'log_if_failed',
  'require_luci_machine_token',
  'require_service_account',
  'require_gce_vm_token',  # this is BotAuthGCE
  'ip_whitelist',
])

# Validated and "frozen" bots_pb2.BotAuth.GCE proto.
BotAuthGCE = collections.namedtuple('BotAuthGCE', ['project'])

# Configuration that applies to some group of bots. Derived from BotsCfg and
# BotGroup in bots.proto. See comments there. This tuple contains already
# validated values.
BotGroupConfig = collections.namedtuple('BotGroupConfig', [
  # The hash of the rest of the data in this tuple, see _gen_version.
  #
  # TODO(vadimsh): Should we rename it to bot_config_script_ver and hash only
  # bot_config_script_content? There's little value in restarting bots when e.g.
  # only 'owners' changes.
  'version',

  # Tuple with emails of bot owners.
  'owners',

  # A list of BotAuth tuples with all applicable authentication methods.
  #
  # The bot is considered authenticated if at least one method applies.
  'auth',

  # Dict {key => list of values}. Always contains all the keys specified by
  # 'trusted_dimensions' set in BotsCfg. If BotGroup doesn't define some
  # dimension from that set, the list of value for it will be empty. Key and
  # values are unicode strings.
  'dimensions',

  # Name of the supplemental bot_config.py to inject to the bot during
  # handshake.
  'bot_config_script',

  # Content of the supplemental bot_config.py to inject to the bot during
  # handshake.
  'bot_config_script_content',

  # An email, "bot" or "". See 'system_service_account' in bots.proto.
  'system_service_account',
])


# Represents bots_pb2.BotsCfg after all includes are expanded, along with its
# revision and a digest string.
#
# If two ExpandedBotsCfg have identical digests, then they are semantically same
# (and vice versa). There's no promise for how exactly the digest is built, this
# is an internal implementation detail.
#
# The revision alone must not be used to detect changes to the expanded config,
# since it doesn't capture changes to the included files.
ExpandedBotsCfg = collections.namedtuple('ExpandedBotsCfg', [
  'bots',    # instance of bots_pb2.BotsCfg with expanded config
  'rev',     # revision of bots.cfg file this config was built from, FYI
  'digest',  # str with digest string, derived from 'bots' contents
])


class BadConfigError(Exception):
  """Raised if the current bots.cfg config is broken."""


def get_bot_group_config(bot_id, machine_type):
  """Returns BotGroupConfig for a bot with given ID or machine type.

  Returns:
    BotGroupConfig or None if not found.

  Raises:
    BadConfigError if there's no cached config and the current config at HEAD is
    not passing validation.
  """
  cfg = _fetch_bot_groups()

  if machine_type and cfg.machine_types.get(machine_type):
    return cfg.machine_types[machine_type]

  gr = cfg.direct_matches.get(bot_id)
  if gr is not None:
    return gr

  for prefix, gr in cfg.prefix_matches:
    if bot_id.startswith(prefix):
      return gr

  return cfg.default_group


def fetch_machine_types():
  """Returns a dict of MachineTypes contained in bots.cfg.

  Returns:
    A dict mapping the name of a MachineType to a bots_pb2.MachineType.

  Raises:
    BadConfigError if there's no cached config and the current config at HEAD is
    not passing validation.
  """
  return _fetch_bot_groups().machine_types_raw


def warmup():
  """Optional warm up of the in-process caches."""
  try:
    _fetch_bot_groups()
  except BadConfigError as exc:
    logging.error('Failed to warm up bots.cfg cache: %s', exc)


def refetch_from_config_service(ctx=None):
  """Updates the cached expanded copy of bots.cfg in the datastore.

  Fetches the bots config from the config service and expands all includes,
  validates the expanded config, and on success rewrites singleton entity with
  the last expanded config, which is later used from serving RPCs.

  Logs errors internally.

  Args:
    ctx: validation.Context to use for config validation or None default.

  Returns:
    ExpandedBotsCfg if the config was successfully fetched.
    None if the config is missing.

  Raises:
    BadConfigError if the config is present, but not valid.
  """
  ctx = ctx or validation.Context.logging()
  cfg = _fetch_and_expand_bots_cfg(ctx)
  if ctx.result().has_errors:
    logging.error('Refusing the invalid config')
    raise BadConfigError('Invalid bots.cfg config, see logs')

  # Fast path to skip the transaction if everything is up-to-date. Mostly
  # important when 'refetch_from_config_service' is called directly from
  # '_get_expanded_bots_cfg', since there may be large stampede of such calls.
  cur = _bots_cfg_head_key().get()
  if cur:
    # Either 'is empty?' flag or the current digest are already set.
    if (cur.empty and not cfg) or cur.digest == cfg.digest:
      logging.info(
          'Config is up-to-date at rev "%s" (digest "%s"), updated %s ago',
          cfg.rev if cfg else 'none',
          cfg.digest if cfg else 'none',
          utils.utcnow()-cur.last_update_ts)
      return cfg

  bots_cfg_pb = cfg.bots.SerializeToString() if cfg else ''

  # pylint: disable=no-value-for-parameter
  @ndb.transactional(propagation=ndb.TransactionOptions.INDEPENDENT)
  def update():
    now = utils.utcnow()
    cur = _bots_cfg_head_key().get()

    # If the config file is missing, we need to let consumers know, otherwise
    # they can't distinguish between "missing the config" and "the fetch cron
    # hasn't run yet".
    if not cfg:
      if not cur or not cur.empty:
        ndb.put_multi([
          BotsCfgHead(
              key=_bots_cfg_head_key(),
              empty=True,
              last_update_ts=now),
          BotsCfgBody(
              key=_bots_cfg_body_key(),
              empty=True,
              last_update_ts=now),
        ])
      return

    # This digest check exists mostly to avoid clobbering memcache if nothing
    # has actually changed.
    if cur and cur.digest == cfg.digest:
      logging.info(
          'Config is up-to-date at rev "%s" (digest "%s"), updated %s ago',
          cfg.rev, cfg.digest, now-cur.last_update_ts)
      return

    logging.info(
        'Storing expanded bots.cfg, its size before compression is %d bytes',
        len(bots_cfg_pb))
    ndb.put_multi([
      BotsCfgHead(
          key=_bots_cfg_head_key(),
          bots_cfg_rev=cfg.rev,
          digest=cfg.digest,
          last_update_ts=now),
      BotsCfgBody(
          key=_bots_cfg_body_key(),
          bots_cfg=bots_cfg_pb,
          bots_cfg_rev=cfg.rev,
          digest=cfg.digest,
          last_update_ts=now),
    ])

  update()
  return cfg


# pylint: disable=no-value-for-parameter
@ndb.transactional(propagation=ndb.TransactionOptions.INDEPENDENT)
def clear_cache():
  """Removes cached bot config from the datastore and the local memory.

  Intended to be used only from tests.
  """
  _bots_cfg_head_key().delete()
  _bots_cfg_body_key().delete()
  _cache.reset()


### Private stuff.


# Bump this to force trigger bots.cfg cache refresh, even if the config itself
# didn't change.
#
# Changing this value is equivalent to removing the entities that hold
# the cache. Note that we intentionally keep older version of the config to
# allow GAE instances that still run the old code to use them.
_BOT_CFG_CACHE_VER = 2


# How often to synchronize in-process bots.cfg cache with what's in the
# datastore.
_IN_PROCESS_CACHE_EXP_SEC = 1


class BotsCfgHead(ndb.Model):
  """Contains digest of the latest expanded bots.cfg, but not the config itself.

  Root singleton entity with ID == _BOT_CFG_CACHE_VER.

  The config body is stored in a separate BotsCfgBody entity. We do it this way
  to avoid fetching a huge entity just to discover we already have the latest
  version cached in the local memory.
  """
  # True if there's no bots.cfg in the config repository.
  empty = ndb.BooleanProperty(indexed=False)
  # The revision of root bots.cfg file used to construct this config, FYI.
  bots_cfg_rev = ndb.StringProperty(indexed=False)
  # Identifies the content of the expanded config and how we got it.
  digest = ndb.StringProperty(indexed=False)
  # When this entity was updated the last time.
  last_update_ts = ndb.DateTimeProperty(indexed=False)


class BotsCfgBody(BotsCfgHead):
  """Contains prefetched and expanded bots.cfg file (with all includes).

  It is updated from a cron via 'refetch_from_config_service' and used for
  serving configs to RPCs via 'get_bot_group_config' and 'fetch_machine_types'.

  There's only one entity of this kind. Its ID is 1 and the parent key is
  corresponding BotsCfgHead.
  """
  # Disable useless in-process per-request cache to save some RAM.
  _use_cache = False

  # Serialized bots_pb2.BotsCfg proto that has all includes expanded.
  bots_cfg = ndb.BlobProperty(compressed=True)


def _bots_cfg_head_key():
  """ndb.Key of BotsCfgHead singleton entity."""
  return ndb.Key(BotsCfgHead, _BOT_CFG_CACHE_VER)


def _bots_cfg_body_key():
  """ndb.Key of BotsCfgBody singleton entity."""
  return ndb.Key(BotsCfgBody, 1, parent=_bots_cfg_head_key())


class _DigestBuilder(object):
  """Tiny helper for building config digest strings."""

  def __init__(self):
    self._h = hashlib.sha256()

  def _write(self, val):
    if val is None:
      val = ''
    elif isinstance(val, unicode):
      val = val.encode('utf-8')
    elif not isinstance(val, str):
      val = str(val)
    self._h.update(str(len(val)))
    self._h.update(val)

  def update(self, key, val):
    self._write(key)
    self._write(val)

  def get(self):
    return 'v%d:%s' % (_BOT_CFG_CACHE_VER, self._h.hexdigest())


def _fetch_and_expand_bots_cfg(ctx):
  """Fetches bots.cfg with all includes from config service, validating it.

  All validation errors are reported through the given validation context.
  Doesn't stop on a first error, parses as much of the config as possible.

  Args:
    ctx: validation.Context to use for accepting validation errors.

  Returns:
    ExpandedBotsCfg if bots.cfg exists.
    None if there's no bots.cfg file, this is not an error.
  """
  # Note: store_last_good=True has a side effect of returning configs that
  # passed @validation.self_rule validators. This is the primary reason we are
  # using it.
  rev, cfg = config.get_self_config(
      BOTS_CFG_FILENAME, bots_pb2.BotsCfg, store_last_good=True)
  if not cfg:
    logging.info('No bots.cfg found')
    return None

  logging.info('Expanding and validating bots.cfg at rev %s', rev)

  digest = _DigestBuilder()
  digest.update('ROOT_REV', rev)

  # Fetch all included bot config scripts.
  _include_bot_config_scripts(cfg, digest, ctx)

  # TODO(vadimsh): Fetch and expand bot lists.
  # TODO(tandrii): Fetch and expand additional bot annotation includes.

  # Revalidate the fully expanded config, it may have new errors not detected
  # when validating each file individually.
  _validate_bots_cfg(cfg, ctx)

  return ExpandedBotsCfg(cfg, rev, digest.get())


def _include_bot_config_scripts(cfg, digest, ctx):
  """Fetches bot_config_script's and substitutes them into bots_pb2.BotsCfg.

  Args:
    cfg: instance of bots_pb2.BotsCfg to mutate.
    digest: instance of _DigestBuilder.
    ctx: instance of validation.Context to emit validation errors.
  """
  # Different bot groups often include same scripts. Deduplicate calls to
  # 'get_self_config'.
  cached = {}  # path -> (rev, content)
  def fetch_script(path):
    if path not in cached:
      rev, content = config.get_self_config(path, store_last_good=True)
      if content:
        logging.info('Using bot config script "%s" at rev %s', path, rev)
      cached[path] = (rev, content)
    return cached[path]

  for idx, gr in enumerate(cfg.bot_group):
    if gr.bot_config_script_content or not gr.bot_config_script:
      continue
    rev, content = fetch_script('scripts/' + gr.bot_config_script)
    if content:
      gr.bot_config_script_content = content
      digest.update('BOT_CONFIG_SCRIPT_REV:%d' % idx, rev)
    else:
      # The entry is invalid. It points to a non existing file. It could be
      # because of a typo in the file name. An empty file is an invalid file.
      ctx.error('missing or empty bot_config_script "%s"', gr.bot_config_script)


def _get_expanded_bots_cfg(known_digest=None):
  """Fetches expanded bots.cfg from the datastore cache.

  If the cache is not there (may happen right after deploying the service or
  after changing _BOT_CFG_CACHE_VER), falls back to fetching the config directly
  right here. This situation is rare.

  Args:
    known_digest: digest of ExpandedBotsCfg already known to the caller, to skip
        fetching it from the cache if nothing has changed.

  Returns:
    (True, ExpandedBotsCfg) if fetched some new version from the cache.
    (True, None) if there's no bots.cfg config at all.
    (False, None) if the cached version has digest matching 'known_digest'.

  Raises:
    BadConfigError if there's no cached config and the current config at HEAD is
    not passing validation.
  """
  head = _bots_cfg_head_key().get()
  if not head:
    # This branch is hit when we deploy the service the first time, before
    # the fetch cron runs, or after changing _BOT_CFG_CACHE_VER. We manually
    # refresh the cache in this case, not waiting for the cron.
    logging.warning(
        'No bots.cfg cached for code v%d, forcing the refresh',
        _BOT_CFG_CACHE_VER)
    expanded = refetch_from_config_service()  # raises BadConfigError on errors
    if expanded and known_digest and expanded.digest == known_digest:
      return False, None
    return True, expanded

  if known_digest and head.digest == known_digest:
    return False, None
  if head.empty:
    return True, None

  # At this point we know there's something newer stored in the cache. Grab it.
  # Since this happens outside of a transaction, we may fetch a version that is
  # ever newer than pointed to by 'head'. This is fine.
  body = _bots_cfg_body_key().get()
  if not body:
    raise AssertionError('BotsCfgBody is missing, this should not be possible')

  if known_digest and body.digest == known_digest:
    return False, None  # the body was sneakily reverted back just now
  if body.empty:
    return True, None

  bots = bots_pb2.BotsCfg()
  bots.ParseFromString(body.bots_cfg)
  return True, ExpandedBotsCfg(bots, body.bots_cfg_rev, body.digest)


# Post-processed and validated read-only immutable form of expanded bots.cfg
# config. Its structure is optimized for fast lookup of BotGroupConfig by
# bot_id.
_BotGroups = collections.namedtuple('_BotGroups', [
  'digest',             # a digest of corresponding ExpandedBotsCfg
  'rev',                # a revision of root bots.cfg config file
  'direct_matches',     # dict bot_id => BotGroupConfig
  'prefix_matches',     # list of pairs (bot_id_prefix, BotGroupConfig)
  'machine_types',      # dict machine_type.name => BotGroupConfig
  'machine_types_raw',  # dict machine_type.name => bots_pb2.MachineType
  'default_group',      # fallback BotGroupConfig or None if not defined
])


class _BotGroupsCache(object):
  """State of _BotGroups in-process cache, see _fetch_bot_groups()."""

  def __init__(self):
    self.lock = threading.Lock()
    self.cfg_and_exp = None  # pair (last _BotGroups, its unix expiration time)
    self.fetcher_thread = None  # a thread that fetches the config now

  def get_cfg_if_fresh(self):
    """Returns cached _BotGroups if it is still fresh or None if not."""
    # We allow this to be executed outside the lock. We assume here that when a
    # change to self.cfg_and_exp field is visible to other threads, all changes
    # to the tuple itself are also already visible. This is safe in Python,
    # there's no memory write reordering there.
    tp = self.cfg_and_exp
    if tp and tp[1] > utils.time_time():
      return tp[0]
    return None

  def set_cfg(self, cfg):
    """Updates cfg and bumps the expiration time."""
    self.cfg_and_exp = (cfg, utils.time_time() + _IN_PROCESS_CACHE_EXP_SEC)

  def reset(self):
    """Resets the state of the cache."""
    with self.lock:
      self.cfg_and_exp = None
      self.fetcher_thread = None


# The actual in-process _BotGroups cache.
_cache = _BotGroupsCache()


# Default config to use on unconfigured server.
def _default_bot_groups():
  return _BotGroups(
    digest='none',
    rev='none',
    direct_matches={},
    prefix_matches=[],
    machine_types={},
    machine_types_raw={},
    default_group=BotGroupConfig(
        version='default',
        owners=(),
        auth=(
          BotAuth(
              log_if_failed=False,
              require_luci_machine_token=False,
              require_service_account=None,
              require_gce_vm_token=None,
              ip_whitelist=auth.bots_ip_whitelist()),
        ),
        dimensions={},
        bot_config_script='',
        bot_config_script_content='',
        system_service_account=''))


def _gen_version(fields):
  """Looks at BotGroupConfig fields and derives a digest that summarizes them.

  This digest is going to be sent to the bot in /handshake, and bot would
  include it in its state (and thus send it with each /poll). If server detects
  that the bot is using older version of the config, it would ask the bot
  to restart.

  Args:
    fields: dict with BotGroupConfig fields (without 'version').

  Returns:
    A string that going to be used as 'version' field of BotGroupConfig tuple.
  """
  # Just hash JSON representation (with sorted keys). Assumes it is stable
  # enough. Add a prefix and trim a bit, to clarify that is it not git hash or
  # anything like that, but just a dumb hash of the actual config.
  fields = fields.copy()
  fields['auth'] = [a._asdict() for a in fields['auth']]
  digest = hashlib.sha256(utils.encode_to_json(fields)).hexdigest()
  return 'hash:' + digest[:14]


def _make_bot_group_config(**fields):
  """Instantiates BotGroupConfig properly deriving 'version' field."""
  return BotGroupConfig(version=_gen_version(fields), **fields)


def _bot_group_proto_to_tuple(msg, trusted_dimensions):
  """bots_pb2.BotGroup => BotGroupConfig.

  Assumes body of bots_pb2.BotGroup is already validated (logs inconsistencies,
  but does not fail).
  """
  dimensions = {unicode(k): set() for k in trusted_dimensions}
  for dim_kv_pair in msg.dimensions:
    # In validated config 'dim_kv_pair' is always 'key:value', but be cautious.
    parts = unicode(dim_kv_pair).split(':', 1)
    if len(parts) != 2:
      logging.error('Invalid dimension in bots.cfg - "%s"', dim_kv_pair)
      continue
    k, v = parts[0], parts[1]
    dimensions.setdefault(k, set()).add(v)

  return _make_bot_group_config(
    owners=tuple(msg.owners),
    auth=tuple(
      BotAuth(
        log_if_failed=cfg.log_if_failed,
        require_luci_machine_token=cfg.require_luci_machine_token,
        require_service_account=tuple(cfg.require_service_account),
        require_gce_vm_token=(
            BotAuthGCE(cfg.require_gce_vm_token.project)
            if cfg.HasField('require_gce_vm_token') else None
        ),
        ip_whitelist=cfg.ip_whitelist)
      for cfg in msg.auth
    ),
    dimensions={k: sorted(v) for k, v in dimensions.iteritems()},
    bot_config_script=msg.bot_config_script or '',
    bot_config_script_content=msg.bot_config_script_content or '',
    system_service_account=msg.system_service_account or '')


def _expand_bot_id_expr(expr):
  """Expands string with bash-like sets (if they are there).

  E.g. takes "vm{1..3}-m1" and yields "vm1-m1", "vm2-m1", "vm3-m1". Also
  supports list syntax ({1,2,3}). Either one should be used, but not both, e.g.
  following WILL NOT work: {1..3,4,5}.

  Yields original string if it doesn't have '{...}' section.

  Raises ValueError if expression has invalid format.
  """
  if not expr:
    raise ValueError('empty bot_id is not allowed')

  left = expr.find('{')
  right = expr.rfind('}')

  if left == -1 and right == -1:
    yield expr
    return

  if expr.count('{') > 1 or expr.count('}') > 1 or left > right:
    raise ValueError('bad bot_id set expression')

  prefix, body, suffix = expr[:left], expr[left+1:right], expr[right+1:]

  # An explicit list?
  if ',' in body:
    # '..' is probably a mistake then.
    if '..' in body:
      raise ValueError(
          '".." is appearing alongside "," in "%s", probably a mistake' % body)
    for itm in body.split(','):
      yield prefix + itm + suffix
    return

  # A range then ('<start>..<end>').
  start, sep, end = body.partition('..')
  if sep != '..':
    raise ValueError('Invalid set "%s", not a list and not a range' % body)
  try:
    start = int(start)
  except ValueError:
    raise ValueError('Not a valid range start "%s"' % start)
  try:
    end = int(end)
  except ValueError:
    raise ValueError('Not a valid range end "%s"' % end)
  for i in xrange(start, end+1):
    yield prefix + str(i) + suffix


def _fetch_bot_groups():
  """Loads bots.cfg and parses it into _BotGroups struct.

  If bots.cfg doesn't exist, returns default config that allows any caller from
  'bots' IP whitelist to act as a bot.

  Caches the loaded bot config internally.

  Returns:
    _BotGroups with pre-processed bots.cfg ready for serving.

  Raises:
    BadConfigError if there's no cached config and the current config at HEAD is
    not passing validation.
  """
  cfg = _cache.get_cfg_if_fresh()
  if cfg:
    logging.info('Using cached bots.cfg at rev %s', cfg.rev)
    return cfg

  with _cache.lock:
    # Maybe someone refreshed it already?
    cfg = _cache.get_cfg_if_fresh()
    if cfg:
      logging.info('Using cached bots.cfg at rev %s', cfg.rev)
      return cfg

    # Nothing is known yet? Block everyone (by holding the lock) until we get
    # a result, there's no other choice.
    known_cfg, exp = _cache.cfg_and_exp or (None, None)
    if not known_cfg:
      cfg = _do_fetch_bot_groups(None)
      _cache.set_cfg(cfg)
      return cfg

    # Someone is already refreshing the cache? Let them finish.
    if _cache.fetcher_thread is not None:
      delta = utils.time_time() - exp
      msg = (
          'Using stale cached bots.cfg at rev %s while another thread is '
          'refreshing it. Cache expired %.1f sec ago.')
      if delta > 5:
        # Only warn if it's more than 5 seconds.
        logging.warning(msg, known_cfg.rev, delta)
      else:
        logging.info(msg, known_cfg.rev, delta)
      return known_cfg

    # Ok, we'll do it, outside the lock.
    tid = threading.current_thread()
    _cache.fetcher_thread = tid

  cfg = None
  try:
    cfg = _do_fetch_bot_groups(known_cfg)
    return cfg
  finally:
    with _cache.lock:
      # 'fetcher_thread' may be different if _cache.reset() was used while we
      # were fetching. Ignore the result in this case.
      if _cache.fetcher_thread is tid:
        _cache.fetcher_thread = None
        if cfg:  # may be None on exceptions
          _cache.set_cfg(cfg)


def _do_fetch_bot_groups(known_cfg=None):
  """Does the actual job of loading and parsing the expanded bots.cfg config.

  Args:
    known_cfg: a currently cached _BotGroups instance to skip refetching it if
        nothing has changed.

  Returns:
    _BotGroups instances (perhaps same as 'known_cfg' if nothing has changed).

  Raises:
    BadConfigError if there's no cached config and the current config at HEAD is
    not passing validation.
  """
  refreshed, expanded_cfg = _get_expanded_bots_cfg(
      known_digest=known_cfg.digest if known_cfg else None)
  if not refreshed:
    logging.info('Cached bots.cfg at rev %s is still fresh', known_cfg.rev)
    return known_cfg
  if not expanded_cfg:
    logging.info('Didn\'t find bots.cfg, using default')
    return _default_bot_groups()

  logging.info('Fetched cached bots.cfg at rev %s', expanded_cfg.rev)
  cfg = expanded_cfg.bots

  direct_matches = {}
  prefix_matches = []
  machine_types = {}
  machine_types_raw = {}
  default_group = None

  known_prefixes = set()

  for entry in cfg.bot_group:
    group_cfg = _bot_group_proto_to_tuple(entry, cfg.trusted_dimensions or [])

    for bot_id_expr in entry.bot_id:
      try:
        for bot_id in _expand_bot_id_expr(bot_id_expr):
          # This should not happen in validated config. If it does, log the
          # error, but carry on, since dying here will bring service offline.
          if bot_id in direct_matches:
            logging.error(
                'Bot "%s" is specified in two different bot groups', bot_id)
            continue
          if bot_id in known_prefixes:
            # TODO(tandrii): change to error and skip this prefix
            # https://crbug.com/781087.
            logging.warn(
                'bot_id "%s" is equal to existing bot_id_prefix of other group',
                bot_id)
          direct_matches[bot_id] = group_cfg
      except ValueError as exc:
        logging.error('Invalid bot_id expression "%s": %s', bot_id_expr, exc)

    for bot_id_prefix in entry.bot_id_prefix:
      if not bot_id_prefix:
        logging.error('Skipping empty bot_id_prefix')
        continue
      if bot_id_prefix in direct_matches:
        # TODO(tandrii): change to error and skip this prefix
        # https://crbug.com/781087.
        logging.warn(
            'bot_id_prefix "%s" is equal to existing bot of %s', bot_id_prefix,
            'the same group ' if group_cfg == direct_matches[bot_id_prefix] else
            'another group')
      prefix_matches.append((bot_id_prefix, group_cfg))
      known_prefixes.add(bot_id_prefix)

    for machine_type in entry.machine_type:
      machine_types[machine_type.name] = group_cfg
      machine_types_raw[machine_type.name] = machine_type

    # Default group?
    if not entry.bot_id and not entry.bot_id_prefix and not entry.machine_type:
      if default_group is not None:
        logging.error('Default bot group is specified twice')
      else:
        default_group = group_cfg

  return _BotGroups(
      expanded_cfg.digest,
      expanded_cfg.rev,
      direct_matches,
      prefix_matches,
      machine_types,
      machine_types_raw,
      default_group)


### Config validation.


def _validate_email(ctx, email, designation):
  try:
    auth.Identity(auth.IDENTITY_USER, email)
  except ValueError:
    ctx.error('invalid %s email "%s"', designation, email)


def _validate_machine_type(ctx, machine_type, known_machine_type_names):
  """Validates machine_type section and updates known_machine_type_names set."""
  if not machine_type.name:
    ctx.error('name is required')
    return
  if machine_type.name in known_machine_type_names:
    ctx.error('reusing name "%s"', machine_type.name)
    return
  known_machine_type_names.add(machine_type.name)
  if not (machine_type.lease_duration_secs or machine_type.lease_indefinitely):
    ctx.error('lease_duration_secs or lease_indefinitely must be specified')
    return
  if machine_type.lease_indefinitely and machine_type.early_release_secs:
    ctx.error('early_release_secs cannot be specified with lease_indefinitely')
    return
  if machine_type.lease_duration_secs < 0:
    ctx.error('lease_duration_secs must be positive')
    return
  if machine_type.early_release_secs < 0:
    ctx.error('early_release_secs must be positive')
    return
  required = {'disk_type', 'num_cpus', 'project'}
  seen = set()
  for j, dim in enumerate(machine_type.mp_dimensions):
    with ctx.prefix('mp_dimensions #%d: ', j):
      if ':' not in dim:
        ctx.error('bad dimension "%s", not a key:value pair', dim)
        return
      key = dim.split(':', 1)[0]
      try:
        field = machine_provider.Dimensions.field_by_name(key)
      except KeyError:
        ctx.error('unknown dimension "%s"', key)
        return
      if key in seen and not field.repeated:
        ctx.error('duplicate value for non-repeated dimension "%s"', key)
        return
      seen.add(key)
      required.discard(key)
  if required:
    ctx.error('missing required mp_dimensions: %s', ', '.join(sorted(required)))
    return
  if machine_type.target_size < 0:
    ctx.error('target_size must be positive')
    return
  _validate_machine_type_schedule(ctx, machine_type.schedule)


def _validate_machine_type_schedule(ctx, schedule):
  if not schedule:
    # No schedule is allowed.
    return
  # Maps day of the week to a list of 2-tuples (start time in minutes,
  # end time in minutes). Used to ensure intervals do not intersect.
  daily_schedules = {day: [] for day in xrange(7)}

  for daily_schedule in schedule.daily:
    if daily_schedule.target_size < 0:
      ctx.error('target size must be non-negative')
    if not daily_schedule.start or not daily_schedule.end:
      ctx.error('daily schedule must have a start and end time')
      continue
    try:
      h1, m1 = map(int, daily_schedule.start.split(':'))
      h2, m2 = map(int, daily_schedule.end.split(':'))
    except ValueError:
      ctx.error('start and end times must be formatted as %%H:%%M')
      continue
    if m1 < 0 or m1 > 59 or m2 < 0 or m2 > 59:
      ctx.error('start and end times must be formatted as %%H:%%M')
      continue
    if h1 < 0 or h1 > 23 or h2 < 0 or h2 > 23:
      ctx.error('start and end times must be formatted as %%H:%%M')
      continue
    start = h1 * 60 + m1
    end = h2 * 60 + m2
    if daily_schedule.days_of_the_week:
      for day in daily_schedule.days_of_the_week:
        if day < 0 or day > 6:
          ctx.error(
              'days of the week must be between 0 (Mon) and 6 (Sun)')
        else:
          daily_schedules[day].append((start, end))
    else:
      # Unspecified means all days.
      for day in xrange(7):
        daily_schedules[day].append((start, end))
    if start >= end:
      ctx.error(
          'end time "%s" must be later than start time "%s"',
          daily_schedule.end,
          daily_schedule.start,
      )
      continue

  # Detect intersections. For each day of the week, sort by start time
  # and ensure that the end of each interval is earlier than the start
  # of the next interval.
  for intervals in daily_schedules.itervalues():
    intervals.sort(key=lambda i: i[0])
    for i in xrange(len(intervals) - 1):
      current_end = intervals[i][1]
      next_start = intervals[i + 1][0]
      if current_end >= next_start:
        ctx.error('intervals must be disjoint')
        continue

  for load_based in schedule.load_based:
    if load_based.maximum_size < load_based.minimum_size:
      ctx.error('maximum size cannot be less than minimum size')
    if load_based.minimum_size < 1:
      ctx.error('minimum size must be positive')


def _validate_group_bot_ids(
    ctx, group_bot_ids, group_idx, known_bot_ids, known_bot_id_prefixes):
  """Validates bot_id sections of a group and updates known_bot_ids."""
  for bot_id_expr in group_bot_ids:
    try:
      for bot_id in _expand_bot_id_expr(bot_id_expr):
        if bot_id in known_bot_ids:
          ctx.error(
              'bot_id "%s" was already mentioned in group #%d',
              bot_id, known_bot_ids[bot_id])
          continue
        if bot_id in known_bot_id_prefixes:
          ctx.error(
              'bot_id "%s" was already mentioned as bot_id_prefix in group #%d',
              bot_id, known_bot_id_prefixes[bot_id])
          continue
        known_bot_ids[bot_id] = group_idx
    except ValueError as exc:
      ctx.error('bad bot_id expression "%s" - %s', bot_id_expr, exc)


def _validate_group_bot_id_prefixes(
    ctx, group_bot_id_prefixes, group_idx, known_bot_id_prefixes,
    known_bot_ids):
  """Validates bot_id_prefixes and updates known_bot_id_prefixes."""
  for bot_id_prefix in group_bot_id_prefixes:
    if not bot_id_prefix:
      ctx.error('empty bot_id_prefix is not allowed')
      continue
    if bot_id_prefix in known_bot_id_prefixes:
      ctx.error(
          'bot_id_prefix "%s" is already specified in group #%d',
          bot_id_prefix, known_bot_id_prefixes[bot_id_prefix])
      continue
    if bot_id_prefix in known_bot_ids:
      ctx.error(
          'bot_id_prefix "%s" is already specified as bot_id in group #%d',
          bot_id_prefix, known_bot_ids[bot_id_prefix])
      continue

    for p, idx in known_bot_id_prefixes.iteritems():
      # Inefficient, but robust code wrt variable char length.
      if p.startswith(bot_id_prefix):
        msg = 'bot_id_prefix "%s" is subprefix of "%s"'
      elif bot_id_prefix.startswith(p):
        msg = 'bot_id_prefix "%s" contains prefix "%s"'
      else:
        continue
      ctx.error(
          msg + ', defined in group #%d, making group assigned for bots '
          'with prefix "%s" ambigious',
          bot_id_prefix, p, idx, min(p, bot_id_prefix))
    known_bot_id_prefixes[bot_id_prefix] = group_idx


def _validate_auth(ctx, a):
  fields = []
  if a.require_luci_machine_token:
    fields.append('require_luci_machine_token')
  if a.require_service_account:
    fields.append('require_service_account')
  if a.HasField('require_gce_vm_token'):
    fields.append('require_gce_vm_token')

  if len(fields) > 1:
    ctx.error('%s can\'t be used at the same time', ' and '.join(fields))
  if not fields and not a.ip_whitelist:
    ctx.error('if all auth requirements are unset, ip_whitelist must be set')

  if a.require_service_account:
    for email in a.require_service_account:
      _validate_email(ctx, email, 'service account')

  if a.HasField('require_gce_vm_token') and not a.require_gce_vm_token.project:
    ctx.error('missing project in require_gce_vm_token')

  if a.ip_whitelist and not auth.is_valid_ip_whitelist_name(a.ip_whitelist):
    ctx.error('invalid ip_whitelist name "%s"', a.ip_whitelist)


def _validate_system_service_account(ctx, bot_group):
  if bot_group.system_service_account == 'bot':
    # If it is 'bot', the bot auth must be configured to use OAuth, since we
    # need to get a bot token somewhere.
    if not any(a.require_service_account for a in bot_group.auth):
      ctx.error(
          'system_service_account "bot" requires '
          'auth.require_service_account to be used')
  elif bot_group.system_service_account:
    # TODO(vadimsh): Strictly speaking we can try to grab a token right
    # here and thus check that IAM policies are configured. But it's not
    # clear what happens if they are not. Will config-service reject the
    # config forever? Will it attempt to revalidate it later?
    _validate_email(
        ctx, bot_group.system_service_account, 'system service account')


@validation.self_rule(BOTS_CFG_FILENAME, bots_pb2.BotsCfg)
def _validate_bots_cfg(cfg, ctx):
  """Validates bots.cfg file."""
  with ctx.prefix('trusted_dimensions: '):
    for dim_key in cfg.trusted_dimensions:
      if not local_config.validate_dimension_key(dim_key):
        ctx.error('invalid dimension key %r', dim_key)

  # Explicitly mentioned bot_id => index of a group where it was mentioned.
  bot_ids = {}
  # bot_id_prefix => index of a group where it was defined.
  bot_id_prefixes = {}
  # Index of a group to use as default fallback (there can be only one).
  default_group_idx = None
  # machine_type names.
  machine_type_names = set()

  for i, entry in enumerate(cfg.bot_group):
    with ctx.prefix('bot_group #%d: ', i):
      # Validate bot_id field and make sure bot_id groups do not intersect.
      _validate_group_bot_ids(ctx, entry.bot_id, i, bot_ids, bot_id_prefixes)

      # Validate bot_id_prefix and make sure bot_id_prefix groups do not
      # intersect.
      _validate_group_bot_id_prefixes(
          ctx, entry.bot_id_prefix, i, bot_id_prefixes, bot_ids)

      # A group without bot_id, bot_id_prefix and machine_type is applied to
      # bots that don't fit any other groups. There should be at most one such
      # group.
      if (not entry.bot_id and
          not entry.bot_id_prefix and
          not entry.machine_type):
        if default_group_idx is not None:
          ctx.error('group #%d is already set as default', default_group_idx)
        else:
          default_group_idx = i

      # Validate machine_type.
      for i, machine_type in enumerate(entry.machine_type):
        with ctx.prefix('machine_type #%d: ', i):
          _validate_machine_type(ctx, machine_type, machine_type_names)

      # Validate 'auth' and 'system_service_account' fields.
      if not entry.auth:
        ctx.error('an "auth" entry is required')
      for a in entry.auth:
        _validate_auth(ctx, a)
      _validate_system_service_account(ctx, entry)

      # Validate 'owners'. Just check they are emails.
      for own in entry.owners:
        _validate_email(ctx, own, 'owner')

      # Validate 'dimensions'.
      for dim in entry.dimensions:
        if not local_config.validate_flat_dimension(dim):
          ctx.error('bad dimension %r', dim)

      # Validate 'bot_config_script': the supplemental bot_config.py.
      if entry.bot_config_script:
        # Another check in bot_code.py confirms that the script itself is valid
        # python before it is accepted by the config service. See
        # _validate_scripts validator there. We later recheck this (see below)
        # when assembling the final expanded bots.cfg.
        if not entry.bot_config_script.endswith('.py'):
          ctx.error('invalid bot_config_script name: must end with .py')
        if os.path.basename(entry.bot_config_script) != entry.bot_config_script:
          ctx.error(
              'invalid bot_config_script name: must not contain path entry')
        # We can't validate that the file exists here. We'll do it later in
        # _fetch_and_expand_bots_cfg when assembling the final config from
        # individual files.

      # Validate 'bot_config_script_content': the content must be valid python.
      # This validation is hit when validating the expanded bot config.
      if entry.bot_config_script_content:
        try:
          ast.parse(entry.bot_config_script_content)
        except (SyntaxError, TypeError) as e:
          ctx.error(
              'invalid bot config script "%s": %s' %
              (entry.bot_config_script, e))
