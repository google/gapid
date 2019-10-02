# Copyright 2019 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import re

import httpserver


class FakeSwarmingServerHandler(httpserver.Handler):
  def do_GET(self):
    logging.info('S GET %s', self.path)
    if self.path == '/auth/api/v1/server/oauth_config':
      self.send_json({
          'client_id': 'c',
          'client_not_so_secret': 's',
          'primary_url': self.server.url})
    elif self.path == '/auth/api/v1/accounts/self':
      self.send_json({'identity': 'user:joe', 'xsrf_token': 'foo'})
    else:
      m = re.match(r'/_ah/api/swarming/v1/task/(\d+)/request', self.path)
      if m:
        logging.info('%s', m.group(1))
        self.send_json(self.server.tasks[int(m.group(1))])
      else:
        self.send_json( {'a': 'b'})
        #raise NotImplementedError(self.path)

  def do_POST(self):
    logging.info('POST %s', self.path)
    raise NotImplementedError(self.path)


class FakeSwarmingServer(httpserver.Server):
  """An extremely minimal implementation of the swarming server client API v1.0.
  """
  _HANDLER_CLS = FakeSwarmingServerHandler

  def __init__(self):
    super(FakeSwarmingServer, self).__init__()
    self._server.tasks = {}
