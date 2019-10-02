# coding: utf-8
# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import datetime
import hashlib
import logging
import os
import threading
import time
import traceback
import urllib

from utils import net

from remote_client_errors import BotCodeError
from remote_client_errors import InitializationError
from remote_client_errors import InternalError
from remote_client_errors import MintOAuthTokenError
from remote_client_errors import PollError


# RemoteClient will attempt to refresh the authentication headers once they are
# this close to the expiration.
#
# The total possible delay between the headers are checked and used is the sum:
#  1) FileRefresherThread update interval (15 sec).
#  2) FileReaderThread update interval (15 sec).
#  3) NET_CONNECTION_TIMEOUT_SEC, when resending requests on errors (3 min).
#
# AUTH_HEADERS_EXPIRATION_SEC must be larger than this sum.
#
# Additionally, there's an upper limit: AUTH_HEADERS_EXPIRATION_SEC must be less
# than the minimum expiration time of headers produced by bot_config's
# get_authentication_headers hook (otherwise we'll be calling this hook all the
# time). On GCE machines it is usually 5 min.
AUTH_HEADERS_EXPIRATION_SEC = 4*60+30


# How long to wait for a response from the server. Must not be greater than
# AUTH_HEADERS_EXPIRATION_SEC, since otherwise there's a chance auth headers
# will expire while we wait for connection.
NET_CONNECTION_TIMEOUT_SEC = 3*60


def createRemoteClient(server, auth, hostname, work_dir, grpc_proxy):
  grpc_proxy = os.environ.get('SWARMING_GRPC_PROXY', grpc_proxy)
  if grpc_proxy:
    import remote_client_grpc
    return remote_client_grpc.RemoteClientGrpc(grpc_proxy)
  return RemoteClientNative(server, auth, hostname, work_dir)


def utcnow():
  return datetime.datetime.utcnow()


def make_appengine_id(hostname, work_dir):
  """Generate a value to use in the GOOGAPPUID cookie for AppEngine.

  AppEngine looks for this cookie: if it contains a value in the range 0-999,
  it is used to split traffic. For more details, see:
  https://cloud.google.com/appengine/docs/flexible/python/splitting-traffic

  The bot code will send requests with a value generated locally:
    GOOGAPPUID = sha1('YYYY-MM-DD-hostname:work_dir') % 1000
  (from go/swarming-release-canaries)

  This scheme should result in the values being roughly uniformly distributed.
  The date is included in the hash to ensure that across different rollouts,
  it's not the same set of bots being used as the canary (otherwise we might
  be unlucky and get a unrepresentative sample).

  Args:
    hostname: The short hostname of the bot.
    work_dir: The working directory used by the bot.

  Returns:
    An integer in the range [0, 999].
  """
  s = '%s-%s:%s' % (utcnow().strftime('%Y-%m-%d'), hostname, work_dir)
  googappuid = int(hashlib.sha1(s).hexdigest(), 16) % 1000
  logging.debug('GOOGAPPUID = sha1(%s) %% 1000 = %d', s, googappuid)
  return googappuid


class RemoteClientNative(object):
  """RemoteClientNative knows how to make authenticated calls to the backend.

  It also holds in-memory cache of authentication headers and periodically
  refreshes them (by calling supplied callback, that usually is implemented in
  terms of bot_config.get_authentication_headers() function).

  If the callback is None, skips authentication (this is used during initial
  stages of the bot bootstrap).

  If the callback returns (*, None), disables authentication. This allows
  bot_config.py to disable strong authentication on machines that don't have any
  credentials (the server uses only IP whitelist check in this case).

  If the callback returns (*, 0), effectively disables the caching of headers:
  the callback will be called for each request.
  """

  def __init__(self, server, auth_headers_callback, hostname, work_dir):
    self._server = server
    self._auth_headers_callback = auth_headers_callback
    self._lock = threading.Lock()
    self._headers = None
    self._exp_ts = None
    self._disabled = not auth_headers_callback
    self._bot_hostname = hostname
    self._bot_work_dir = work_dir

  @property
  def server(self):
    return self._server

  @property
  def is_grpc(self):
    return False

  def initialize(self, quit_bit=None):
    """Grabs initial auth headers, retrying on errors a bunch of times.

    Disabled authentication (when auth_headers_callback returns None) is not
    an error. Retries only real exceptions raised by the callback.

    Raises InitializationError if all attempts fail. Aborts attempts and returns
    if quit_bit is signaled. If quit_bit is None, retries until success or until
    all attempts fail.
    """
    attempts = 30
    while not quit_bit or not quit_bit.is_set():
      try:
        logging.info('Fetching initial auth headers')
        headers = self._get_headers_or_throw()
        logging.info('Got auth headers: %s', headers.keys() or 'none')
        return
      except Exception as e:
        last_error = '%s\n%s' % (e, traceback.format_exc()[-2048:])
        logging.exception('Failed to grab initial auth headers')
      attempts -= 1
      if not attempts:
        raise InitializationError(last_error)
      time.sleep(2)

  @property
  def uses_auth(self):
    """Returns True if get_authentication_headers() returns some headers.

    If bot_config.get_authentication_headers() is not implement it will return
    False.
    """
    return bool(self.get_authentication_headers())

  def get_headers(self, include_auth=False):
    """Returns the headers to use to send a request.

    Args:
      include_auth: Whether or not to include authentication headers.

    Returns:
      A dict of HTTP headers.
    """
    googappuid = make_appengine_id(self._bot_hostname, self._bot_work_dir)
    headers = {'Cookie': 'GOOGAPPUID=%d' % googappuid}
    if include_auth:
      headers.update(self.get_authentication_headers())
    return headers

  def get_authentication_headers(self):
    """Returns a dict with the headers, refreshing them if necessary.

    Will always return a dict (perhaps empty if no auth headers are provided by
    the callback or it has failed).
    """
    try:
      return self._get_headers_or_throw()
    except Exception:
      logging.exception('Failed to refresh auth headers, using cached ones')
      return self._headers or {}

  @property
  def authentication_headers_expiration(self):
    """Returns int unix timestamp of when current cached auth headers expire.

    Returns 0 if unknown or None if not using auth at all.
    """
    return int(self._exp_ts) if not self._disabled else None

  def _get_headers_or_throw(self):
    if self._disabled:
      return {}
    with self._lock:
      if (not self._exp_ts or
          self._exp_ts - time.time() < AUTH_HEADERS_EXPIRATION_SEC):
        self._headers, self._exp_ts = self._auth_headers_callback()
        if self._exp_ts is None:
          logging.info('Headers callback returned None, disabling auth')
          self._disabled = True
          self._headers = {}
        elif self._exp_ts:
          next_check = max(
              0, self._exp_ts - AUTH_HEADERS_EXPIRATION_SEC - time.time())
          if self._headers:
            logging.info(
                'Fetched auth headers (%s), they expire in %d sec. '
                'Next check in %d sec.',
                self._headers.keys(),
                self._exp_ts - time.time(),
                next_check)
          else:
            logging.info(
                'No headers available yet, next check in %d sec.', next_check)
        else:
          logging.info('Using auth headers (%s).', self._headers.keys())
      return self._headers or {}

  def _url_read_json(self, url_path, data=None):
    """Does POST (if data is not None) or GET request to a JSON endpoint."""
    return net.url_read_json(
        self._server + url_path,
        data=data,
        headers=self.get_headers(include_auth=True),
        timeout=NET_CONNECTION_TIMEOUT_SEC,
        follow_redirects=False)

  def _url_retrieve(self, filepath, url_path):
    """Fetches the file from the given URL path on the server."""
    return net.url_retrieve(
        filepath,
        self._server + url_path,
        headers=self.get_headers(include_auth=True),
        timeout=NET_CONNECTION_TIMEOUT_SEC)

  def post_bot_event(self, event_type, message, attributes):
    """Logs bot-specific info to the server"""
    data = attributes.copy()
    data['event'] = event_type
    data['message'] = message
    self._url_read_json('/swarming/api/v1/bot/event', data=data)

  def post_task_update(self, task_id, bot_id, params,
                       stdout_and_chunk=None, exit_code=None):
    """Posts task update to task_update.

    Arguments:
      stdout: Incremental output since last call, if any.
      stdout_chunk_start: Total number of stdout previously sent, for coherency
          with the server.
      params: Default JSON parameters for the POST.
      exit_code: if None, this is an intermediate update. If non-None, this is
          the final update.

    Returns:
      False if the task should stop.

    Raises:
      InternalError if can't contact the server after many attempts or the
      server replies with an error.
    """
    data = {
        'id': bot_id,
        'task_id': task_id,
    }
    data.update(params)
    # Preserving prior behaviour: empty stdout is not transmitted
    if stdout_and_chunk and stdout_and_chunk[0]:
      data['output'] = base64.b64encode(stdout_and_chunk[0])
      data['output_chunk_start'] = stdout_and_chunk[1]
    if exit_code != None:
      data['exit_code'] = exit_code

    resp = self._url_read_json(
        '/swarming/api/v1/bot/task_update/%s' % task_id, data)
    logging.debug('post_task_update() = %s', resp)
    if not resp or resp.get('error'):
      raise InternalError(
          resp.get('error') if resp else 'Failed to contact server')
    return not resp.get('must_stop', False)

  def post_task_error(self, task_id, bot_id, message):
    """Logs task-specific info to the server"""
    data = {
        'id': bot_id,
        'message': message,
        'task_id': task_id,
    }
    resp = self._url_read_json(
        '/swarming/api/v1/bot/task_error/%s' % task_id,
        data=data)
    return resp and resp['resp'] == 1

  def do_handshake(self, attributes):
    """Performs the initial handshake. Returns a dict (contents TBD)"""
    return self._url_read_json(
        '/swarming/api/v1/bot/handshake',
        data=attributes)

  def poll(self, attributes):
    """Polls for new work or other commands; returns a (cmd, value) pair as
    shown below.

    Raises:
      PollError if can't contact the server after many attempts, the server
      replies with an error or the returned dict does not have the correct
      values set.
    """
    resp = self._url_read_json('/swarming/api/v1/bot/poll', data=attributes)
    if not resp or resp.get('error'):
      raise PollError(
          resp.get('error') if resp else 'Failed to contact server')

    cmd = resp['cmd']
    if cmd == 'sleep':
      return (cmd, resp['duration'])
    if cmd == 'terminate':
      return (cmd, resp['task_id'])
    if cmd == 'run':
      return (cmd, resp['manifest'])
    if cmd == 'update':
      return (cmd, resp['version'])
    if cmd in ('restart', 'host_reboot'):
      return (cmd, resp['message'])
    if cmd == 'bot_restart':
      return (cmd, resp['message'])
    raise PollError('Unexpected command: %s\n%s' % (cmd, resp))

  def get_bot_code(self, new_zip_path, bot_version, bot_id):
    """Downloads code into the file specified by new_zip_fn (a string).

    Throws BotCodeError on error.
    """
    url_path = '/swarming/api/v1/bot/bot_code/%s?bot_id=%s' % (
        bot_version, urllib.quote_plus(bot_id))
    if not self._url_retrieve(new_zip_path, url_path):
      raise BotCodeError(new_zip_path, self._server + url_path, bot_version)

  def ping(self):
    """Unlike all other methods, this one isn't authenticated."""
    resp = net.url_read(self._server + '/swarming/api/v1/bot/server_ping')
    if resp is None:
      logging.error('No response from server_ping')

  def mint_oauth_token(self, task_id, bot_id, account_id, scopes):
    """Asks the server to generate an access token for a service account.

    Each task has two service accounts associated with it: 'system' and 'task'.
    Swarming server is capable of generating oauth tokens for them (if the bot
    is currently authorized to have access to them).

    Args:
      task_id: identifier of currently executing task.
      bot_id: name of the bot.
      account_id: logical identifier of the account (e.g 'system' or 'task').
      scopes: list of OAuth scopes the new token should have.

    Returns:
      {
        'service_account': <str>,      # account email or 'bot', or 'none'
        'access_token': <str> or None, # actual token, if using real account
        'expiry': <int>,               # unix timestamp in seconds
      }

    Raises:
      InternalError if can't contact the server after many attempts or the
      server consistently replies with HTTP 5** errors.

      MintOAuthTokenError on fatal errors.
    """
    resp = self._url_read_json('/swarming/api/v1/bot/oauth_token', data={
        'account_id': account_id,
        'id': bot_id,
        'scopes': scopes,
        'task_id': task_id,
    })
    if not resp:
      raise InternalError('Error when minting the token')
    if resp.get('error'):
      raise MintOAuthTokenError(resp['error'])
    return resp
