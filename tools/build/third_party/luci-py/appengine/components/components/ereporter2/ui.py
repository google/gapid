# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""UI code to generate email and HTML reports."""

import itertools
import logging
import os
import re
from xml.sax import saxutils

from google.appengine.api import app_identity
from google.appengine.api import mail
from google.appengine.api import mail_errors

from components import template
from components import utils

from . import logscraper
from . import models
from . import on_error


ROOT_DIR = os.path.dirname(os.path.abspath(__file__))


### Private stuff.


def _get_default_start_time():
  """Calculates default value for start_time."""
  info = models.ErrorReportingInfo.primary_key().get()
  return info.timestamp if info else None


def _get_end_time_for_email():
  """Exists so it can be mocked when testing _generate_and_email_report().

  Do not read by default anything more recent than 5 minutes to cope with mild
  level of logservice inconsistency. High levels of logservice inconsistencies
  will result in lost messages.
  """
  return int((utils.utcnow() - utils.EPOCH).total_seconds() - 5 * 60)


def _records_to_params(categories, ignored_count, request_id_url, report_url):
  """Generates and returns a dict to generate an error report.

  Arguments:
    categories: list of _ErrorCategory reports.
    ignored_count: number of errors not reported because ignored.
    request_id_url: base url to link to a specific request_id.
    report_url: base url to use to recreate this report.
  """
  categories = sorted(
      categories, key=lambda e: (e.signature, -e.events.total_count))
  return {
    'error_count': len(categories),
    'errors': categories,
    'ignored_count': ignored_count,
    'occurrence_count': sum(e.events.total_count for e in categories),
    'report_url': report_url,
    'request_id_url': request_id_url,
    'version_count':
        len(set(itertools.chain.from_iterable(e.versions for e in categories))),
  }


def _email_html(to, subject, body):
  """Sends an email including a textual representation of the HTML body.

  The body must not contain <html> or <body> tags.
  """
  mail_args = {
    'body': saxutils.unescape(re.sub(r'<[^>]+>', r'', body)),
    'html': '<html><body>%s</body></html>' % body,
    'sender': 'no_reply@%s.appspotmail.com' % app_identity.get_application_id(),
    'subject': subject,
  }
  try:
    if to:
      mail_args['to'] = to
      mail.send_mail(**mail_args)
    else:
      mail.send_mail_to_admins(**mail_args)
    return True
  except mail_errors.BadRequestError:
    return False


def _get_template_env(start_time, end_time, module_versions):
  """Generates commonly used jinja2 template variables."""
  return {
    'end': end_time,
    'module_versions': module_versions or [],
    'start': start_time or 0,
  }


def _generate_and_email_report(
    module_versions, recipients, request_id_url, report_url, extras):
  """Generates and emails an exception report.

  To be called from a cron_job.

  Arguments:
    module_versions: list of tuple of module-version to gather info about.
    recipients: str containing comma separated email addresses.
    request_id_url: base url to use to link to a specific request_id.
    report_url: base url to use to recreate this report.
    extras: extra dict to use to render the template.

  Returns:
    True if the email was sent successfully.
  """
  start_time = _get_default_start_time()
  end_time = _get_end_time_for_email()
  logging.info(
      '_generate_and_email_report(%s, %s, %s, ..., %s)',
      start_time, end_time, module_versions, recipients)
  categories, ignored, end_time = logscraper.scrape_logs_for_errors(
      start_time, end_time, module_versions)
  if categories:
    params = _get_template_env(start_time, end_time, module_versions)
    params.update(extras or {})
    params.update(
        _records_to_params(
            categories, sum(c.events.total_count for c in ignored),
            request_id_url, report_url))
    body = template.render('ereporter2/email_report_content.html', params)
    subject_line = template.render(
        'ereporter2/email_report_title.html', params)
    if not _email_html(recipients, subject_line, body):
      on_error.log(
          source='server',
          category='email',
          message='Failed to email ereporter2 report')
  logging.info('New timestamp %s', end_time)
  models.ErrorReportingInfo(
      key=models.ErrorReportingInfo.primary_key(),
      timestamp=end_time).put()
  logging.info(
      'Processed %d items, ignored %d, reduced to %d categories, sent to %s.',
      sum(c.events.total_count for c in categories),
      sum(c.events.total_count for c in ignored),
      len(categories),
      recipients)
  return True


### Public API.


def configure():
  template.bootstrap({'ereporter2': os.path.join(ROOT_DIR, 'templates')})
