# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Swarming bot code. Includes bootstrap and swarming_bot.zip.

It includes everything that is AppEngine specific. The non-GAE code is in
bot_archive.py.
"""

import ast
import collections
import hashlib
import logging
import os.path
import urllib

from google.appengine.api import memcache
from google.appengine.ext import ndb

from components import auth
from components import config
from components import utils
from server import bot_archive
from server import config as local_config


ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# In theory, a memcache entry can be 1MB in size, and this sometimes works, but
# in practice we found that it's flaky at 500kb or above. 250kb seems to be safe
# though and doesn't appear to have any runtime impact.
#    - aludwin@, June 2017
MAX_MEMCACHED_SIZE_BYTES = 250000


### Models.


File = collections.namedtuple('File', ('content', 'who', 'when', 'version'))


### Public APIs.


def get_bootstrap(host_url, bootstrap_token=None):
  """Returns the mangled version of the utility script bootstrap.py.

  Try to find the content in the following order:
  - get the file from luci-config
  - return the default version

  Returns:
    File instance.
  """
  # Calculate the header to inject at the top of the file.
  if bootstrap_token:
    quoted = urllib.quote_plus(bootstrap_token)
    assert bootstrap_token == quoted, bootstrap_token
  header = (
      '#!/usr/bin/env python\n'
      '# coding: utf-8\n'
      'host_url = %r\n'
      'bootstrap_token = %r\n') % (host_url or '', bootstrap_token or '')

  # Check in luci-config imported file if present.
  rev, cfg = config.get_self_config(
      'scripts/bootstrap.py', store_last_good=True)
  if cfg:
    return File(header + cfg, config.config_service_hostname(), None, rev)

  # Fallback to the one embedded in the tree.
  path = os.path.join(ROOT_DIR, 'swarming_bot', 'config', 'bootstrap.py')
  with open(path, 'rb') as f:
    return File(header + f.read(), None, None, None)


def get_bot_config():
  """Returns the current version of bot_config.py.

  Try to find the content in the following order:
  - get the file from luci-config
  - return the default version

  Returns:
    File instance.
  """
  # Check in luci-config imported file if present.
  rev, cfg = config.get_self_config(
      'scripts/bot_config.py', store_last_good=True)
  if cfg:
    return File(cfg, config.config_service_hostname(), None, rev)

  # Fallback to the one embedded in the tree.
  path = os.path.join(ROOT_DIR, 'swarming_bot', 'config', 'bot_config.py')
  with open(path, 'rb') as f:
    return File(f.read(), None, None, None)


def get_bot_version(host):
  """Retrieves the current bot version (SHA256) loaded on this server.

  The memcache is first checked for the version, otherwise the value
  is generated and then stored in the memcache.

  Returns:
    tuple(hash of the current bot version, dict of additional files).
  """
  signature = _get_signature(host)
  version = memcache.get('version-' + signature, namespace='bot_code')
  if version:
    return version, None

  # Need to calculate it.
  additionals = {'config/bot_config.py': get_bot_config().content}
  bot_dir = os.path.join(ROOT_DIR, 'swarming_bot')
  version = bot_archive.get_swarming_bot_version(
      bot_dir, host, utils.get_app_version(), additionals,
      local_config.settings())
  memcache.set('version-' + signature, version, namespace='bot_code', time=60)
  return version, additionals


def get_swarming_bot_zip(host):
  """Returns a zipped file of all the files a bot needs to run.

  Returns:
    A string representing the zipped file's contents.
  """
  version, additionals = get_bot_version(host)
  content = get_cached_swarming_bot_zip(version)
  if content:
    logging.debug('memcached bot code %s; %d bytes', version, len(content))
    return content

  # Get the start bot script from the database, if present. Pass an empty
  # file if the files isn't present.
  additionals = additionals or {
    'config/bot_config.py': get_bot_config().content,
  }
  bot_dir = os.path.join(ROOT_DIR, 'swarming_bot')
  content, version = bot_archive.get_swarming_bot_zip(
      bot_dir, host, utils.get_app_version(), additionals,
      local_config.settings())
  logging.info('generated bot code %s; %d bytes', version, len(content))
  cache_swarming_bot_zip(version, content)
  return content


def get_cached_swarming_bot_zip(version):
  """Returns the bot contents if its been cached, or None if missing."""
  # see cache_swarming_bot_zip for how the "meta" entry is set
  meta = bot_memcache_get(version, 'meta').get_result()
  if meta is None:
    logging.info('memcache did not include metadata for version %s', version)
    return None
  num_parts, true_sig = meta.split(':')

  # Get everything asynchronously. If something's missing, the hash will be
  # wrong so no need to check that we got something from each call.
  futures = [bot_memcache_get(version, 'content', p)
             for p in range(int(num_parts))]
  content = ''
  missing = 0
  for idx, f in enumerate(futures):
    chunk = f.get_result()
    if chunk is None:
      logging.debug(
          'bot code %s was missing chunk %d/%d', version, idx, len(futures))
      missing += 1
    else:
      content += chunk
  if missing:
    logging.warning(
        'bot code %s was missing %d/%d chunks', version, missing, len(futures))
    return None
  h = hashlib.sha256()
  h.update(content)
  if h.hexdigest() != true_sig:
    logging.error('bot code %s had signature %s instead of expected %s',
                  version, h.hexdigest(), true_sig)
    return None
  return content


def cache_swarming_bot_zip(version, content):
  """Caches the bot code to memcache."""
  h = hashlib.sha256()
  h.update(content)
  p = 0
  futures = []
  while len(content) > 0:
    chunk_size = min(MAX_MEMCACHED_SIZE_BYTES, len(content))
    futures.append(bot_memcache_set(content[:chunk_size],
                                    version, 'content', p))
    content = content[chunk_size:]
    p += 1
  for f in futures:
    f.check_success()
  meta = "%s:%s" % (p, h.hexdigest())
  bot_memcache_set(meta, version, 'meta').check_success()
  logging.info('bot %s with sig %s saved in memcached in %d chunks',
               version, h.hexdigest(), p)


def bot_memcache_get(version, desc, part=None):
  """Mockable async memcache getter."""
  return ndb.get_context().memcache_get(bot_key(version, desc, part),
                                        namespace='bot_code')


def bot_memcache_set(value, version, desc, part=None):
  """Mockable async memcache setter."""
  return ndb.get_context().memcache_set(bot_key(version, desc, part),
                                        value, namespace='bot_code')


def bot_key(version, desc, part=None):
  """Returns a memcache key for bot entries."""
  key = 'code-%s-%s' % (version, desc)
  if part is not None:
    key = '%s-%d' % (key, part)
  return key


### Bootstrap token.


class BootstrapToken(auth.TokenKind):
  expiration_sec = 3600
  secret_key = auth.SecretKey('bot_bootstrap_token')
  version = 1


def generate_bootstrap_token():
  """Returns a token that authenticates calls to bot bootstrap endpoints.

  The authenticated bootstrap workflow looks like this:
    1. An admin visit Swarming server root page and copy-pastes URL to
       bootstrap.py that has a '?tok=...' parameter with the bootstrap token,
       generated by this function.
    2. /bootstrap verifies the token and serves bootstrap.py, with same token
       embedded into it.
    3. The modified bootstrap.py is executed on the bot. It fetches bot code
       from /bot_code, passing it the bootstrap token again.
    4. /bot_code verifies the token and serves the bot code zip archive.

  This function assumes the caller is already authorized.
  """
  # The embedded payload is mostly FYI. The important expiration time is added
  # by BootstrapToken already.
  return BootstrapToken.generate(message=None, embedded={
    'for': auth.get_current_identity().to_bytes(),
  })


def validate_bootstrap_token(tok):
  """Returns a token payload if the token is valid or None if not.

  The token is valid if its HMAC signature is correct and it hasn't expired yet.

  Doesn't recheck ACLs. Logs errors.
  """
  try:
    return BootstrapToken.validate(tok, message=None)
  except auth.InvalidTokenError as exc:
    logging.warning('Failed to validate bootstrap token: %s', exc)
    return None


### Private code


def _validate_python(content):
  """Returns True if content is valid python script."""
  try:
    ast.parse(content)
  except (SyntaxError, TypeError):
    return False
  return True


def _get_signature(host):
  # CURRENT_VERSION_ID is unique per appcfg.py upload so it can be trusted.
  return hashlib.sha256(host + os.environ['CURRENT_VERSION_ID']).hexdigest()


## Config validators


@config.validation.self_rule('regex:scripts/.+\\.py')
def _validate_scripts(content, ctx):
  try:
    ast.parse(content)
  except (SyntaxError, TypeError) as e:
    ctx.error('invalid %s: %s' % (ctx.path, e))
