# Copyright 2012 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""This module defines Isolate Server frontend url handlers."""

import collections
import datetime
import json
import logging

import webapp2

import cloudstorage
from google.appengine.api import modules

import acl
import config
import gcs
import handlers_endpoints_v1
import mapreduce_jobs
import model
import stats
import template
from components import auth
from components import stats_framework
from components import utils


# GViz data description.
_GVIZ_DESCRIPTION = {
  'failures': ('number', 'Failures'),
  'requests': ('number', 'Total'),
  'other_requests': ('number', 'Other'),
  'uploads': ('number', 'Uploads'),
  'uploads_bytes': ('number', 'Uploaded'),
  'downloads': ('number', 'Downloads'),
  'downloads_bytes': ('number', 'Downloaded'),
  'contains_requests': ('number', 'Lookups'),
  'contains_lookups': ('number', 'Items looked up'),
}

# Warning: modifying the order here requires updating templates/stats.html.
_GVIZ_COLUMNS_ORDER = (
  'key',
  'requests',
  'other_requests',
  'failures',
  'uploads',
  'downloads',
  'contains_requests',
  'uploads_bytes',
  'downloads_bytes',
  'contains_lookups',
)

_ISOLATED_ROOT_MEMBERS = (
  'algo',
  'command',
  'files',
  'includes',
  'read_only',
  'relative_cwd',
  'version',
)


### Restricted handlers


class RestrictedConfigHandler(auth.AuthenticatingHandler):
  @auth.autologin
  @auth.require(auth.is_admin)
  def get(self):
    self.common(None)

  @staticmethod
  def cast_to_type(param_name, value):
    def to_bool(value):
      # pylint: disable=unidiomatic-typecheck
      if type(value) is bool:
        return value
      return {'True': True, 'False': False}.get(value, False)

    cast = {
        'enable_ts_monitoring': to_bool,
    }.get(param_name, str)
    return cast(value)

  @auth.require(auth.is_admin)
  def post(self):
    # Convert MultiDict into a dict.
    params = {
      k: self.cast_to_type(k, self.request.params.getone(k))
      for k in self.request.params
      if k not in ('keyid', 'xsrf_token')
    }
    cfg = config.settings(fresh=True)
    keyid = int(self.request.get('keyid', '0'))
    if cfg.key.integer_id() != keyid:
      self.common('Update conflict %s != %s' % (cfg.key.integer_id(), keyid))
      return
    cfg.populate(**params)
    try:
      # Ensure key is correct, it's easy to make a mistake when creating it.
      gcs.URLSigner.load_private_key(cfg.gs_private_key)
    except Exception as exc:
      # TODO(maruel): Handling Exception is too generic. And add self.abort(400)
      self.response.write('Bad private key: %s' % exc)
      return
    cfg.store(updated_by=auth.get_current_identity().to_bytes())
    self.common('Settings updated')

  def common(self, note):
    params = config.settings_info()
    params.update({
        'note': note,
        'path': self.request.path,
        'xsrf_token': self.generate_xsrf_token(),
    })
    self.response.write(
        template.render('isolate/restricted_config.html', params))


class RestrictedPurgeHandler(auth.AuthenticatingHandler):
  @auth.autologin
  @auth.require(auth.is_admin)
  def get(self):
    params = {
      'digest': '',
      'message': '',
      'namespace': '',
      'xsrf_token': self.generate_xsrf_token(),
    }
    self.response.write(
        template.render('isolate/restricted_purge.html', params))

  @auth.require(auth.is_admin)
  def post(self):
    namespace = self.request.get('namespace')
    digest = self.request.get('digest')
    params = {
      'digest': digest,
      'message': '',
      'namespace': namespace,
      'xsrf_token': self.generate_xsrf_token(),
    }
    try:
      key = model.get_entry_key(namespace, digest)
    except ValueError as e:
      params['message'] = 'Invalid entry: %s' % e
      key = None
    if key:
      model.delete_entry_and_gs_entry([key])
      params['message'] = 'Done'
    self.response.write(
        template.render('isolate/restricted_purge.html', params))


### Mapreduce related handlers


class RestrictedLaunchMapReduceJob(auth.AuthenticatingHandler):
  """Enqueues a task to start a map reduce job on the backend module.

  A tree of map reduce jobs inherits module and version of a handler that
  launched it. All UI handlers are executes by 'default' module. So to run a
  map reduce on a backend module one needs to pass a request to a task running
  on backend module.
  """

  @auth.require(auth.is_admin)
  def post(self):
    job_id = self.request.get('job_id')
    assert job_id in mapreduce_jobs.MAPREDUCE_JOBS
    # Do not use 'backend' module when running from dev appserver. Mapreduce
    # generates URLs that are incompatible with dev appserver URL routing when
    # using custom modules.
    success = utils.enqueue_task(
        url='/internal/taskqueue/mapreduce/launch/%s' % job_id,
        queue_name=mapreduce_jobs.MAPREDUCE_TASK_QUEUE,
        use_dedicated_module=not utils.is_local_dev_server())
    # New tasks should show up on the status page.
    if success:
      self.redirect('/mapreduce/status')
    else:
      self.abort(500, 'Failed to launch the job')


### Non-restricted handlers


class BrowseHandler(auth.AuthenticatingHandler):
  @auth.autologin
  @auth.require(acl.isolate_readable)
  def get(self):
    namespace = self.request.get('namespace', 'default-gzip')
    # Support 'hash' for compatibility with old links. To remove eventually.
    digest = self.request.get('digest', '') or self.request.get('hash', '')
    save_as = self.request.get('as', '')
    params = {
      u'as': unicode(save_as),
      u'digest': unicode(digest),
      u'namespace': unicode(namespace),
    }
    # Check for existence of element, so we can 400/404
    if digest and namespace:
      try:
        model.get_content(namespace, digest)
      except ValueError:
        self.abort(400, 'Invalid key')
      except LookupError:
        self.abort(404, 'Unable to retrieve the entry')
    self.response.write(template.render('isolate/browse.html', params))

  def get_content_security_policy(self):
    csp = super(BrowseHandler, self).get_content_security_policy()
    csp.setdefault('frame-src', []).append("'self'")
    return csp


class ContentHandler(auth.AuthenticatingHandler):
  @auth.autologin
  @auth.require(acl.isolate_readable)
  def get(self):
    namespace = self.request.get('namespace', 'default-gzip')
    digest = self.request.get('digest', '')
    content = None
    if not digest:
      self.abort(400, 'Missing digest')
    if not namespace:
      self.abort(400, 'Missing namespace')

    try:
      raw_data, entity = model.get_content(namespace, digest)
    except ValueError:
      self.abort(400, 'Invalid key')
    except LookupError:
      self.abort(404, 'Unable to retrieve the entry')

    logging.info('%s', entity)
    if not raw_data:
      try:
        stream = gcs.read_file(config.settings().gs_bucket, entity.key.id())
        content = ''.join(model.expand_content(namespace, stream))
      except cloudstorage.NotFoundError:
        logging.error('Entity in DB but not in GCS: deleting entity in DB')
        entity.key.delete()
        self.abort(404, 'Unable to retrieve the file from GCS')
    else:
      content = ''.join(model.expand_content(namespace, [raw_data]))

    self.response.headers['X-Frame-Options'] = 'SAMEORIGIN'
    # We delete Content-Type before storing to it to avoid having two (yes,
    # two) Content-Type headers.
    del self.response.headers['Content-Type']

    # Apparently, setting the content type to text/plain encourages the
    # browser (Chrome, at least) to sniff the mime type and display
    # things like images.  Images are autowrapped in <img> and text is
    # wrapped in <pre>.
    self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'

    # App Engine puts a limit of 33554432 bytes on a request, which includes
    # headers. Headers are ~150 bytes.  If the content + headers might
    # exceed that limit, we give the user an option to workround getting
    # their file.
    if len(content) > 33554000:
      host = modules.get_hostname(module='default', version='default')
      # host is something like default.default.myisolateserver.appspot.com
      host = host.replace('default.default.','')
      sizeInMib = len(content) / (1024.0 * 1024.0)
      content = ('Sorry, your file is %1.1f MiB big, which exceeds the 32 MiB'
      ' App Engine limit.\nTo work around this, run the following command:\n'
      '    python isolateserver.py download -I %s --namespace %s -f %s %s'
      % (sizeInMib, host, namespace, digest, digest))
    else:
      self.response.headers['Content-Disposition'] = str(
        'filename=%s' % self.request.get('as') or digest)
      try:
        json_data = json.loads(content)
        if self._is_isolated_format(json_data):
          self.response.headers['Content-Type'] = 'text/html; charset=utf-8'
          if 'files' in json_data:
            json_data['files'] = collections.OrderedDict(
              sorted(
                json_data['files'].items(),
                key=lambda (filepath, data): filepath))
          params = {
            'namespace': namespace,
            'isolated': json_data,
          }
          content = template.render('isolate/isolated.html', params)
      except ValueError:
        pass

    self.response.write(content)

  @staticmethod
  def _is_isolated_format(json_data):
    """Checks if json_data is a valid .isolated format."""
    if not isinstance(json_data, dict):
      return False
    actual = set(json_data)
    return actual.issubset(_ISOLATED_ROOT_MEMBERS) and (
        'files' in actual or 'includes' in actual
    )


###  Public pages.


class RootHandler(auth.AuthenticatingHandler):
  """Tells the user to RTM."""

  @auth.public
  def get(self):
    params = {
      'is_admin': auth.is_admin(),
      'is_user': acl.isolate_readable(),
      'mapreduce_jobs': [],
      'user_type': acl.get_user_type(),
    }
    if auth.is_admin():
      params['mapreduce_jobs'] = [
        {'id': job_id, 'name': job_def['job_name']}
        for job_id, job_def in mapreduce_jobs.MAPREDUCE_JOBS.iteritems()
      ]
      params['xsrf_token'] = self.generate_xsrf_token()
    self.response.write(template.render('isolate/root.html', params))


class UIHandler(auth.AuthenticatingHandler):
  """Serves the landing page for the new UI of the requested page.

  This landing page is stamped with the OAuth 2.0 client id from the
  configuration.
  """
  @auth.public
  def get(self):
    params = {
      'client_id': config.settings().ui_client_id,
    }
    # Can cache for 1 week, because the only thing that would change in this
    # template is the oauth client id, which changes very infrequently.
    self.response.cache_control.no_cache = None
    self.response.cache_control.public = True
    self.response.cache_control.max_age = 604800
    try:
      self.response.write(template.render(
        'isolate/public_isolate_index.html', params))
    except template.TemplateNotFound:
      self.abort(404, 'Page not found.')


class WarmupHandler(webapp2.RequestHandler):
  def get(self):
    config.warmup()
    auth.warmup()
    self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
    self.response.write('ok')


class EmailHandler(webapp2.RequestHandler):
  """Blackhole any email sent."""
  def post(self, to):
    pass


def get_routes():
  routes = [
      # AppEngine-specific urls:
      webapp2.Route(r'/_ah/mail/<to:.+>', EmailHandler),
      webapp2.Route(r'/_ah/warmup', WarmupHandler),
  ]
  if not utils.should_disable_ui_routes():
    routes.extend([
      # Administrative urls.
      webapp2.Route(r'/restricted/config', RestrictedConfigHandler),
      webapp2.Route(r'/restricted/purge', RestrictedPurgeHandler),

      # Mapreduce related urls.
      webapp2.Route(
          r'/restricted/launch_mapreduce',
          RestrictedLaunchMapReduceJob),

      # User web pages.
      webapp2.Route(r'/browse', BrowseHandler),
      webapp2.Route(r'/content', ContentHandler),
      webapp2.Route(r'/', RootHandler),
      webapp2.Route(r'/newui', UIHandler),
    ])
  routes.extend(handlers_endpoints_v1.get_routes())
  return routes


def create_application(debug):
  """Creates the url router.

  The basic layouts is as follow:
  - /restricted/.* requires being an instance administrator.
  - /stats/.* has statistics.
  """
  acl.bootstrap()
  template.bootstrap()
  return webapp2.WSGIApplication(get_routes(), debug=debug)
