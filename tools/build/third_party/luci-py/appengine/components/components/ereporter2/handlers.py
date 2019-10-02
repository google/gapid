# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""HTTP Handlers."""

import datetime
import itertools
import json
import time

import webapp2

from google.appengine.api import app_identity
from google.appengine.datastore import datastore_query
from google.appengine.ext import ndb

from components import auth
from components import decorators
from components import template
from components import utils

from . import acl
from . import logscraper
from . import models
from . import on_error
from . import ui


# Access to a protected member XXX of a client class - pylint: disable=W0212


### Admin pages.


class RestrictedEreporter2Report(auth.AuthenticatingHandler):
  """Returns all the recent errors as a web page."""

  @auth.autologin
  @auth.require(acl.is_ereporter2_viewer)
  def get(self):
    """Reports the errors logged and ignored.

    Arguments:
      start: epoch time to start looking at. Defaults to the messages since the
             last email.
      end: epoch time to stop looking at. Defaults to now.
      modules: comma separated modules to look at.
      tainted: 0 or 1, specifying if desiring tainted versions. Defaults to 1.
    """
    # TODO(maruel): Be consistent about using either epoch or human readable
    # formatted datetime.
    end = int(float(self.request.get('end', 0)) or time.time())
    start = int(
        float(self.request.get('start', 0)) or
        ui._get_default_start_time() or 0)
    modules = self.request.get('modules')
    if modules:
      modules = modules.split(',')
    tainted = bool(int(self.request.get('tainted', '1')))
    module_versions = utils.get_module_version_list(modules, tainted)
    errors, ignored, _end_time = logscraper.scrape_logs_for_errors(
        start, end, module_versions)

    params = {
      'errors': errors,
      'errors_count': sum(len(e.events) for e in errors),
      'errors_version_count':
          len(set(itertools.chain.from_iterable(e.versions for e in errors))),
      'ignored': ignored,
      'ignored_count': sum(len(i.events) for i in ignored),
      'ignored_version_count':
          len(set(itertools.chain.from_iterable(i.versions for i in ignored))),
      'xsrf_token': self.generate_xsrf_token(),
    }
    params.update(ui._get_template_env(start, end, module_versions))
    self.response.write(template.render('ereporter2/requests.html', params))


class RestrictedEreporter2Request(auth.AuthenticatingHandler):
  """Dumps information about single logged request."""

  @auth.autologin
  @auth.require(acl.is_ereporter2_viewer)
  def get(self, request_id):
    data = logscraper._log_request_id(request_id)
    if not data:
      self.abort(404, detail='Request id was not found.')
    self.response.write(
        template.render('ereporter2/request.html', {'request': data}))


class RestrictedEreporter2ErrorsList(auth.AuthenticatingHandler):
  """Dumps information about reported client side errors."""

  @auth.autologin
  @auth.require(acl.is_ereporter2_viewer)
  def get(self):
    limit = int(self.request.get('limit', 100))
    cursor = datastore_query.Cursor(urlsafe=self.request.get('cursor'))
    errors_found, cursor, more = models.Error.query().order(
        -models.Error.created_ts).fetch_page(limit, start_cursor=cursor)
    params = {
      'cursor': cursor.urlsafe() if cursor and more else None,
      'errors': errors_found,
      'limit': limit,
      'now': utils.utcnow(),
    }
    self.response.out.write(template.render('ereporter2/errors.html', params))


class RestrictedEreporter2Error(auth.AuthenticatingHandler):
  """Dumps information about reported client side errors."""

  @auth.autologin
  @auth.require(acl.is_ereporter2_viewer)
  def get(self, error_id):
    error = models.Error.get_by_id(int(error_id))
    if not error:
      self.abort(404, 'Error not found')
    params = {
      'error': error,
      'now': utils.utcnow(),
    }
    self.response.out.write(template.render('ereporter2/error.html', params))


class RestrictedEreporter2Silence(auth.AuthenticatingHandler):
  @auth.autologin
  @auth.require(acl.is_ereporter2_viewer)
  def get(self):
    # Due to historical reasons where created_ts had indexed=False,, do not use
    # .order(models.ErrorReportingMonitoring.created_ts) yet. Fix this once all
    # objects have been updated.
    items = models.ErrorReportingMonitoring.query().fetch()
    items.sort(key=lambda x: x.created_ts)
    params = {
      'silenced': items,
      'xsrf_token': self.generate_xsrf_token(),
    }
    self.response.out.write(template.render('ereporter2/silence.html', params))

  @auth.require(acl.is_ereporter2_editor)
  def post(self):
    to_delete = self.request.get('to_delete')
    if to_delete:
      ndb.Key(models.ErrorReportingMonitoring, to_delete).delete()
    else:
      mute_type = self.request.get('mute_type')
      error = None
      if mute_type in ('exception_type', 'signature'):
        error = self.request.get(mute_type)
      if not error:
        self.abort(400)
      silenced = self.request.get('silenced')
      silenced_until = self.request.get('silenced_until')
      if silenced_until == 'T':
        silenced_until = ''
      threshold = self.request.get('threshold')
      key = models.ErrorReportingMonitoring.error_to_key(error)
      if not silenced and not silenced_until and not threshold:
        key.delete()
      else:
        item = models.ErrorReportingMonitoring(key=key, error=error)
        if silenced:
          item.silenced = True
        if silenced_until:
          item.silenced_until = datetime.datetime.strptime(
              silenced_until, '%Y-%m-%dT%H:%M')
        if threshold:
          item.threshold = int(threshold)
        item.put()

    self.get()


### Cron jobs.


class CronEreporter2Mail(webapp2.RequestHandler):
  """Generate and emails an exception report."""
  @decorators.require_cronjob
  def get(self):
    """Sends email(s) containing the errors logged."""
    # Do not use self.request.host_url because it will be http:// and will point
    # to the backend, with an host format that breaks the SSL certificate.
    # TODO(maruel): On the other hand, Google Apps instances are not hosted on
    # appspot.com.
    host_url = 'https://%s.appspot.com' % app_identity.get_application_id()
    request_id_url = host_url + '/restricted/ereporter2/request/'
    report_url = host_url + '/restricted/ereporter2/report'
    recipients = self.request.get('recipients', acl.get_ereporter2_recipients())
    result = ui._generate_and_email_report(
        utils.get_module_version_list(None, False),
        recipients,
        request_id_url,
        report_url,
        {})
    self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
    if result:
      self.response.write('Success.')
    else:
      # Do not HTTP 500 since we do not want it to be retried.
      self.response.write('Failed.')


class CronEreporter2Cleanup(webapp2.RequestHandler):
  """Deletes old error reports."""
  @decorators.require_cronjob
  def get(self):
    old_cutoff = utils.utcnow() - on_error.ERROR_TIME_TO_LIVE
    items = models.Error.query(
        models.Error.created_ts < old_cutoff,
        default_options=ndb.QueryOptions(keys_only=True))
    out = len(ndb.delete_multi(items))
    self.response.headers['Content-Type'] = 'text/plain; charset=utf-8'
    self.response.write(str(out))


### Public API.


class OnErrorHandler(auth.AuthenticatingHandler):
  """Adds an error report.

  This one is open so errors like authentication reports are logged in too.
  This means we could get spammed a lot about it. Implement DDoS protection by
  rate limiting once a kid figures out.
  """
  xsrf_token_enforce_on = ()

  # TODO(maruel): This was copied from ../../auth/ui/rest_api.py and needs to be
  # factored out.
  def parse_body(self):
    """Parse JSON body and verifies it's a dict."""
    expected = ('application/json', 'application/json; charset=utf-8')
    if self.request.headers.get('Content-Type').lower() not in expected:
      msg = 'Expecting JSON body with content type \'application/json\''
      self.abort(400, msg)
    try:
      body = json.loads(self.request.body)
      if not isinstance(body, dict):
        raise ValueError()
    except ValueError:
      self.abort(400, 'Not a valid json dict body')
    return body

  @auth.public
  def post(self):
    body = self.parse_body()
    version = body.get('v')
    # Do not enforce version for now, just assert it is present.
    if not version:
      self.abort(400, 'Missing version')

    report = body.get('r')
    if not report:
      self.abort(400, 'Missing report')

    kwargs = dict(
        (k, report[k]) for k in on_error.VALID_ERROR_KEYS if report.get(k))
    report_id = on_error.log_request(self.request, add_params=False, **kwargs)
    self.response.headers['Content-Type'] = 'application/json; charset=utf-8'
    body = {
      'id': report_id,
      'url':
          '%s/restricted/ereporter2/errors/%d' %
          (self.request.host_url, report_id),
    }
    self.response.write(utils.encode_to_json(body))


def get_frontend_routes():
  routes = [
    # Public API.
    webapp2.Route(
      '/ereporter2/api/v1/on_error', OnErrorHandler),
  ]
  if not utils.should_disable_ui_routes():
    routes.extend([
      webapp2.Route(
        r'/restricted/ereporter2/errors',
        RestrictedEreporter2ErrorsList),
      webapp2.Route(
        r'/restricted/ereporter2/errors/<error_id:\d+>',
        RestrictedEreporter2Error),
      webapp2.Route(
        r'/restricted/ereporter2/report',
        RestrictedEreporter2Report),
      webapp2.Route(
        r'/restricted/ereporter2/request/<request_id:[0-9a-fA-F]+>',
        RestrictedEreporter2Request),
      webapp2.Route(
        r'/restricted/ereporter2/silence',
        RestrictedEreporter2Silence),
    ])

  return routes


def get_backend_routes():
  # This requires a cron job to this URL.
  return [
    webapp2.Route(
        r'/internal/cron/ereporter2/cleanup', CronEreporter2Cleanup),
    webapp2.Route(
        r'/internal/cron/ereporter2/mail', CronEreporter2Mail),
  ]
