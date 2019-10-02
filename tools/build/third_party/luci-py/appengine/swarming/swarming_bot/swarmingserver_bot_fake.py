# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import os
import sys
import threading

BOT_DIR = os.path.dirname(os.path.abspath(__file__))

sys.path.insert(
    0,
    os.path.join(os.path.dirname(BOT_DIR), '..', '..', '..', 'client', 'tests'))
import httpserver
sys.path.pop(0)

sys.path.insert(0, os.path.join(os.path.dirname(BOT_DIR), 'server'))
import bot_archive
sys.path.pop(0)


def gen_zip(url):
  """Returns swarming_bot.zip content."""
  with open(os.path.join(BOT_DIR, 'config', 'bot_config.py'), 'rb') as f:
    bot_config_content = f.read()
  return bot_archive.get_swarming_bot_zip(
      BOT_DIR, url, '1', {'config/bot_config.py': bot_config_content}, None)


def flatten_task_updates(updates):
  """Flatten a list of task updates into a single result.

  This is more or less the equivalent of what task_scheduler.bot_update_task()
  would do after all the bot API calls.
  """
  out = {}
  for update in updates:
    if out.get('output') and update.get('output'):
      # Accumulate output.
      update = update.copy()
      out['output'] += update.pop('output')
      update.pop('output_chunk_start')
    out.update(update)
  return out


class Handler(httpserver.Handler):
  """Minimal Swarming bot server fake implementation."""
  def do_GET(self):
    if self.path == '/swarming/api/v1/bot/server_ping':
      self.send_response(200)
      return None
    if self.path == '/auth/api/v1/server/oauth_config':
      return self.send_json({
          'client_id': 'id',
          'client_not_so_secret': 'hunter2',
          'primary_url': self.server.url,
        })
    raise NotImplementedError(self.path)

  def do_POST(self):
    data = json.loads(self.read_body())

    if self.path == '/auth/api/v1/accounts/self/xsrf_token':
      return self.send_json({'xsrf_token': 'a'})

    if self.path == '/swarming/api/v1/bot/event':
      self.server.parent._add_bot_event(data)
      return self.send_json({})

    if self.path == '/swarming/api/v1/bot/handshake':
      return self.send_json({'xsrf_token': 'fine'})

    if self.path == '/swarming/api/v1/bot/poll':
      self.server.parent.has_polled.set()
      return self.send_json({'cmd': 'sleep', 'duration': 60})

    if self.path.startswith('/swarming/api/v1/bot/task_update/'):
      task_id = self.path[len('/swarming/api/v1/bot/task_update/'):]
      must_stop = self.server.parent._on_task_update(task_id, data)
      return self.send_json({'ok': True, 'must_stop': must_stop})

    if self.path.startswith('/swarming/api/v1/bot/task_error'):
      task_id = self.path[len('/swarming/api/v1/bot/task_error/'):]
      self.server.parent._add_task_error(task_id, data)
      return self.send_json({'resp': 1})

    raise NotImplementedError(self.path)

  def do_PUT(self):
    raise NotImplementedError(self.path)


class Server(httpserver.Server):
  """Fake a Swarming bot API server for local testing."""
  _HANDLER_CLS = Handler

  def __init__(self):
    super(Server, self).__init__()
    self._lock = threading.Lock()
    # Accumulated bot events.
    self._bot_events = []
    # Running tasks.
    self._tasks = {}
    # Bot reported task errors.
    self._task_errors = {}
    self.has_polled = threading.Event()
    self.has_updated_task = threading.Event()
    self.must_stop = False

  def get_bot_events(self):
    """Returns the events reported by the bots."""
    with self._lock:
      return self._bot_events[:]

  def get_tasks(self):
    """Returns the tasks run by the bots."""
    with self._lock:
      return self._tasks.copy()

  def get_task_errors(self):
    """Returns the task errors reported by the bots."""
    with self._lock:
      return self._task_errors.copy()

  def _add_bot_event(self, data):
    # Used by the handler.
    with self._lock:
      self._bot_events.append(data)

  def _on_task_update(self, task_id, data):
    with self._lock:
      self._tasks.setdefault(task_id, []).append(data)
      must_stop = self.must_stop
      self.has_updated_task.set()
      return must_stop

  def _add_task_error(self, task_id, data):
    # Used by the handler.
    with self._lock:
      self._task_errors.setdefault(task_id, []).append(data)
