#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Tools to control application running via dev_appserver.py.

Useful for smoke and integration tests.
"""

import collections
import cookielib
import ctypes
import json
import logging
import os
import shutil
import socket
import subprocess
import sys
import time
import urllib2

from . import gae_sdk_utils

try:
  from utils import file_path
  rmtree = file_path.rmtree
except ImportError:
  rmtree = shutil.rmtree


def terminate_with_parent():
  """Sets up current process to receive SIGTERM when its parent dies.

  Works on Linux only. On Win and Mac it's noop.
  """
  try:
    libc = ctypes.CDLL('libc.so.6')
  except OSError:
    return
  PR_SET_PDEATHSIG = 1
  SIGTERM = 15
  try:
    libc.prctl(PR_SET_PDEATHSIG, SIGTERM)
  except AttributeError:
    return


def is_port_free(host, port):
  """Returns True if the listening port number is available."""
  s = socket.socket()
  try:
    # connect_ex returns 0 on success (i.e. port is being listened to).
    return bool(s.connect_ex((host, port)))
  finally:
    s.close()


def find_free_ports(host, base_port, count):
  """Finds several consecutive listening ports free to listen to."""
  while base_port < (2<<16):
    candidates = range(base_port, base_port + count)
    if all(is_port_free(host, port) for port in candidates):
      return candidates
    base_port += len(candidates)
  assert False, (
      'Failed to find %d available ports starting at %d' % (count, base_port))


class LocalApplication(object):
  """GAE application running locally via dev_appserver.py."""

  def __init__(self, app_dir, base_port, listen_all, root, app_id=None):
    self._app = gae_sdk_utils.Application(app_dir, app_id)
    self._base_port = base_port
    self._client = None
    self._log = None
    self._port = None
    self._proc = None
    self._serving = False
    self._root = os.path.join(root, self.app_id)
    self._listen_all = listen_all

  @property
  def app_id(self):
    """Application ID as specified in app.yaml."""
    return self._app.app_id

  @property
  def port(self):
    """Main HTTP port that serves requests to 'default' module.

    Valid only after app has started.
    """
    return self._port

  @property
  def url(self):
    """Host URL."""
    return 'http://localhost:%d' % self._port

  @property
  def client(self):
    """HttpClient that can be used to make requests to the instance."""
    return self._client

  @property
  def log(self):
    """Returns the log output. Only set after calling stop()."""
    return self._log

  def start(self):
    """Starts dev_appserver process."""
    assert not self._proc, 'Already running'

    # Clear state.
    self._client = None
    self._log = None
    self._serving = False

    # Find available ports, one per module + one for app admin.
    free_ports = find_free_ports(
        'localhost', self._base_port, len(self._app.modules) + 1)
    self._port = free_ports[0]

    os.makedirs(os.path.join(self._root, 'storage'))

    # Launch the process.
    log_file = os.path.join(self._root, 'dev_appserver.log')
    logging.info(
        'Launching %s at %s, log is %s', self.app_id, self.url, log_file)
    cmd = [
      '--port', str(self._port),
      '--admin_port', str(free_ports[-1]),
      '--storage_path', os.path.join(self._root, 'storage'),
      '--automatic_restart', 'no',
      '--log_level', 'debug',
      # Note: The random policy will provide the same consistency every
      # time the test is run because the random generator is always given
      # the same seed.
      '--datastore_consistency_policy', 'random',
    ]
    if self._listen_all:
      cmd.extend(('--host', '0.0.0.0'))
      cmd.extend(('--admin_host', '0.0.0.0'))
      cmd.extend(('--api_host', '0.0.0.0'))
      cmd.extend(('--enable_host_checking', 'false'))
    else:
      # The default is 'localhost' EXCEPT if environment variable
      # 'DEVSHELL_CLIENT_PORT' is set, then the default is '0.0.0.0'. Take no
      # chance and always bind to localhost.
      cmd.extend(('--host', 'localhost'))
      cmd.extend(('--admin_host', 'localhost'))
      cmd.extend(('--api_host', 'localhost'))

    kwargs = {}
    if sys.platform != 'win32':
      kwargs['preexec_fn'] = terminate_with_parent
    with open(log_file, 'wb') as f:
      self._proc = self._app.spawn_dev_appserver(
          cmd,
          stdout=f,
          stderr=subprocess.STDOUT,
          **kwargs)

    # Create a client that can talk to the service.
    self._client = HttpClient(self.url)

  def ensure_serving(self, timeout=10):
    """Waits for the service to start responding."""
    if self._serving:
      return
    if not self._proc:
      self.start()
    logging.info('Waiting for %s to become ready...', self.app_id)
    deadline = time.time() + timeout
    alive = False
    while self._proc.poll() is None and time.time() < deadline:
      try:
        urllib2.urlopen(self.url + '/_ah/warmup')
        alive = True
        break
      except urllib2.URLError as exc:
        if isinstance(exc, urllib2.HTTPError):
          alive = True
          break
      time.sleep(0.05)
    if not alive:
      logging.error('Service %s did\'t come online', self.app_id)
      self.stop()
      self.dump_log()
      raise Exception('Failed to start %s' % self.app_id)
    logging.info('Service %s is ready.', self.app_id)
    self._serving = True

  def stop(self):
    """Stops dev_appserver, collects its log.

    Returns the process error code if applicable.
    """
    if not self._proc:
      return None
    exit_code = self._proc.poll()
    try:
      logging.info('Stopping %s', self.app_id)
      if self._proc.poll() is None:
        try:
          # Send SIGTERM.
          self._proc.terminate()
        except OSError:
          pass
        deadline = time.time() + 5
        while self._proc.poll() is None and time.time() < deadline:
          time.sleep(0.05)
        exit_code = self._proc.poll()
        if exit_code is None:
          logging.error('Leaking PID %d', self._proc.pid)
    finally:
      with open(os.path.join(self._root, 'dev_appserver.log'), 'r') as f:
        self._log = f.read()
      self._client = None
      self._port = None
      self._proc = None
      self._serving = False
    return exit_code

  def wait(self):
    """Waits for the process to exit."""
    self._proc.wait()

  def dump_log(self):
    """Prints dev_appserver log to stderr, works only if app is stopped."""
    print >> sys.stderr, '-' * 60
    print >> sys.stderr, 'dev_appserver.py log for %s' % self.app_id
    print >> sys.stderr, '-' * 60
    for l in (self._log or '').strip('\n').splitlines():
      sys.stderr.write('  %s\n' % l)
    print >> sys.stderr, '-' * 60


class CustomHTTPErrorHandler(urllib2.HTTPDefaultErrorHandler):
  """Swallows exceptions that would be thrown on >30x HTTP status."""
  def http_error_default(self, _request, response, _code, _msg, _hdrs):
    return response


class HttpClient(object):
  """Makes HTTP requests to some instance of dev_appserver."""

  # Return value of request(...) and json_request.
  HttpResponse = collections.namedtuple(
      'HttpResponse', ['http_code', 'body', 'headers'])

  def __init__(self, url):
    self._url = url
    self._opener = urllib2.build_opener(
        CustomHTTPErrorHandler(),
        urllib2.HTTPCookieProcessor(cookielib.CookieJar()))
    self._xsrf_token = None

  def login_as_admin(self, user='test@example.com'):
    """Performs dev_appserver login as admin, modifies cookies."""
    self.request('/_ah/login?email=%s&admin=True&action=Login' % user)
    self._xsrf_token = None

  def request(self, resource, body=None, headers=None, method=None):
    """Sends HTTP request."""
    if not resource.startswith(self._url):
      assert resource.startswith('/')
      resource = self._url + resource
    req = urllib2.Request(resource, body, headers=(headers or {}))
    if method:
      req.get_method = lambda: method
    resp = self._opener.open(req)
    return self.HttpResponse(resp.getcode(), resp.read(), resp.info())

  def json_request(self, resource, body=None, headers=None, method=None):
    """Sends HTTP request and returns deserialized JSON."""
    if body is not None:
      body = json.dumps(body)
      headers = (headers or {}).copy()
      headers['Content-Type'] = 'application/json; charset=UTF-8'
    resp = self.request(resource, body, headers=headers, method=method)
    try:
      value = json.loads(resp.body)
    except ValueError:
      raise ValueError('Invalid JSON: %r' % resp.body)
    return self.HttpResponse(resp.http_code, value, resp.headers)

  @property
  def url_opener(self):
    """Instance of urllib2 opener used by this class."""
    return self._opener

  @property
  def xsrf_token(self):
    """Returns XSRF token for the service, fetching it if necessary.

    It only works with apps that use 'auth' component.
    """
    if self._xsrf_token is None:
      resp = self.json_request(
          '/auth/api/v1/accounts/self/xsrf_token',
          body={},
          headers={'X-XSRF-Token-Request': '1'})
      self._xsrf_token = resp.body['xsrf_token'].encode('ascii')
    return self._xsrf_token
