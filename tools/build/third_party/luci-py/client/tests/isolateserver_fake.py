# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import base64
import hashlib
import json
import logging
import re
import zlib

import httpserver

ALGO = hashlib.sha1


def hash_content(content):
  return ALGO(content).hexdigest()


class FakeSigner(object):
  @classmethod
  def generate(cls, message, embedded):
    return '%s_<<<%s>>>' % (repr(message), json.dumps(embedded))

  @classmethod
  def validate(cls, ticket, message):
    a = re.match(r'^' + repr(message) + r'_<<<(.*)>>>$', ticket, re.DOTALL)
    if not a:
      raise ValueError('Message %s cannot validate ticket %s' % (
          repr(message), ticket))
    return json.loads(a.groups()[0])


class FakeIsolateServerHandler(httpserver.Handler):
  """An extremely minimal implementation of the isolate server API v1.0."""

  def _should_push_to_gs(self, isolated, size):
    max_memcache = 500 * 1024
    min_direct_gs = 501
    if isolated and size <= max_memcache:
      return False
    return size >= min_direct_gs

  def _generate_signed_url(self, digest, namespace='default'):
    return '%s/FAKE_GCS/%s/%s' % (self.server.url, namespace, digest)

  def _generate_ticket(self, entry_dict):
    """Generates an 'upload_ticket' to reply as PreuploadStatus."""
    embedded = dict(
        entry_dict,
        **{
            'c': 'flate',
            'h': 'SHA-1',
        })
    message = (
        'gs' if self._should_push_to_gs(embedded['i'], embedded['s'])
        else 'datastore')
    return FakeSigner.generate(message, embedded)

  def _storage_helper(self, body, finalize_gs):
    """Processes handlers_endpoints_v1.StorageRequest."""
    request = json.loads(body)
    message = 'gs' if finalize_gs else 'datastore'
    content = request['content'] if not finalize_gs else None
    embedded = FakeSigner.validate(request['upload_ticket'], message)
    # Embedded is an internal format used by
    # handlers_endpoints_v1.IsolateService.generate_ticket.
    namespace = embedded['n']
    d = embedded['d']
    if self.server.store_hash_instead:
      if not finalize_gs:
        self.server.contents.setdefault(namespace, {})[d] = hash_content(
            content)
    else:
      self.server.contents.setdefault(namespace, {})[d] = content
    self.send_json({'ok': True})

  ### Faked HTTP Methods

  def do_GET(self):
    logging.info('GET %s', self.path)
    if self.path == '/auth/api/v1/server/oauth_config':
      self.send_json({
          'client_id': 'c',
          'client_not_so_secret': 's',
          'primary_url': self.server.url})
    elif self.path == '/auth/api/v1/accounts/self':
      self.send_json({'identity': 'user:joe', 'xsrf_token': 'foo'})
    else:
      raise NotImplementedError(self.path)

  def do_POST(self):
    logging.info('POST %s', self.path)
    body = self.read_body()
    if self.path.startswith('/_ah/api/isolateservice/v1/preupload'):
      response = {'items': []}
      def append_entry(entry, index, li):
        """Converts a {'h', 's', 'i'} to ["<upload url>", "<finalize url>"] or
        None.
        """
        if entry['d'] not in self.server.contents.get(entry['n'], {}):
          # handlers_endpoints_v1.PreuploadStatus
          status = {
              'digest': entry['d'],
              'index': str(index),
              'upload_ticket': self._generate_ticket(entry),
          }
          if self._should_push_to_gs(entry['i'], entry['s']):
            status['gs_upload_url'] = self._generate_signed_url(entry['d'])
          li.append(status)
        # Don't use finalize url for the fake.

      request = json.loads(body)
      namespace = request['namespace']['namespace']
      for index, i in enumerate(request['items']):
        append_entry({
            'd': i['digest'],
            'i': i['is_isolated'],
            'n': namespace,
            's': i['size'],
        }, index, response['items'])
      logging.info('Returning %s' % response)
      self.send_json(response)
    elif self.path.startswith('/_ah/api/isolateservice/v1/store_inline'):
      self._storage_helper(body, False)
    elif self.path.startswith('/_ah/api/isolateservice/v1/finalize_gs_upload'):
      self._storage_helper(body, True)
    elif self.path.startswith('/_ah/api/isolateservice/v1/retrieve'):
      request = json.loads(body)
      namespace = request['namespace']['namespace']
      data = self.server.contents[namespace].get(request['digest'])
      if data is None:
        logging.error(
            'Failed to retrieve %s / %s', namespace, request['digest'])
      self.send_json({'content': data})
    elif self.path.startswith('/_ah/api/isolateservice/v1/server_details'):
      self.send_json({'server_version': 'such a good version'})
    else:
      raise NotImplementedError(self.path)

  def do_PUT(self):
    logging.info('PUT %s', self.path)
    if self.path.startswith('/FAKE_GCS/'):
      namespace, h = self.path[len('/FAKE_GCS/'):].split('/', 1)
      if self.server.store_hash_instead:
        a = ALGO()
        for c in self.yield_body():
          a.update(c)
        self.server.contents.setdefault(namespace, {})[h] = a.hexdigest()
      else:
        self.server.contents.setdefault(namespace, {})[h] = self.read_body()
      self.send_octet_stream('')
    else:
      raise NotImplementedError(self.path)


class FakeIsolateServer(httpserver.Server):
  _HANDLER_CLS = FakeIsolateServerHandler

  def __init__(self):
    super(FakeIsolateServer, self).__init__()
    self._server.contents = {}
    self._server.store_hash_instead = False

  def store_hash_instead(self):
    """Stops saving content in memory. Used to test large files."""
    self._server.store_hash_instead = True

  @property
  def contents(self):
    return self._server.contents

  def add_content_compressed(self, namespace, content):
    assert not self._server.store_hash_instead
    h = hash_content(content)
    logging.info('add_content_compressed(%s, %s)', namespace, h)
    self._server.contents.setdefault(namespace, {})[h] = base64.b64encode(
        zlib.compress(content))
    return h

  def add_content(self, namespace, content):
    assert not self._server.store_hash_instead
    h = hash_content(content)
    logging.info('add_content(%s, %s)', namespace, h)
    self._server.contents.setdefault(namespace, {})[h] = base64.b64encode(
        content)
    return h
