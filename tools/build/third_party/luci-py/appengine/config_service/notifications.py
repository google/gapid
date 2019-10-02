# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Notifies interested parties about rejection of a config set revision."""

import logging
import re
import sys

from components import auth
from components import template
from components import utils
from google.appengine.api import app_identity
from google.appengine.api import mail
from google.appengine.api import mail_errors
from google.appengine.ext import ndb
from google.appengine.ext.webapp import mail_handlers

import storage

CC_GROUP = 'config-validation-cc'
RE_GIT_HASH = re.compile('^[0-9a-f]{40}$')


class FailedToNotify(Exception):
  """Raised when failed to send a notification."""


class Notification(ndb.Model):
  """Entity exists if a notification was sent.

  Entity key:
    Root entity. Entity id is a unique id of notification, for example
    URL of a bad commit.
  """


def get_recipients(commit):
  """Returns a list of recipients for |commit|.

  If committer and author have same email, returns only author.
  """
  for r in (commit.author, commit.committer):
    try:
      mail.CheckEmailValid(r.email, 'to')
    except mail.InvalidEmailError as ex:
      raise FailedToNotify(
          ('Failed to notify %s, invalid email %s: %s' %
           (r.name, r.email, ex)))

  names={
    commit.committer.email: commit.committer.name,
    commit.author.email: commit.author.name,
  }
  return [
    '%s <%s>' % (name or email, email)
    for email, name in names.iteritems()
  ]


def notify_gitiles_rejection(config_set, location, validation_result):
  """Notifies interested parties about an error in a config set revision.

  Sends a notification per location only once.

  Args:
    location (gitiles.Location): an absolute gitiles location of the config set
      that could not be imported.
    validation_result (components.config.validation_context.Result).
  """
  assert RE_GIT_HASH.match(location.treeish), location

  if Notification.get_by_id(str(location)):
    logging.debug('Notification was already sent.')
    return

  log = location.get_log(limit=1)
  if not log or not log.commits:
    logging.error('could not load commit %s', location)
    return
  commit = log.commits[0]
  app_id = app_identity.get_application_id()
  rev = location.treeish[:7]

  try:
    template_params = {
      'author': commit.author.name or commit.author.email,
      'messages': [
        {
          'severity': logging.getLevelName(msg.severity),
          'text': msg.text
        }
        for msg in validation_result.messages
      ],
      'rev_link': location,
      'rev_hash': rev,
      'rev_repo': location.project,
      'cur_rev_hash': None,
      'cur_rev_link': None,
    }

    cs = storage.ConfigSet.get_by_id(config_set)
    if cs and cs.latest_revision:
      template_params.update(
          cur_rev_hash=cs.latest_revision[:7],
          cur_rev_link=cs.latest_revision_url,
      )
    msg = mail.EmailMessage(
        sender=(
            '%s.appspot.com <noreply@%s.appspotmail.com>' % (app_id, app_id)),
        subject='Config revision %s is rejected' % rev,
        to=get_recipients(commit),
        html=template.render(
            'templates/validation_notification.html', template_params))
    cc = get_cc_recipients()
    if cc:
      msg.cc = cc
    logging.info('Emailing %s', ', '.join(msg.to))
    _send(msg)
  except mail_errors.Error as ex:
    raise FailedToNotify(ex.message), None, sys.exc_info()[2]

  Notification(id=str(location)).put()


@utils.cache_with_expiration(10 * 60)
def get_cc_recipients():
  """Returns a set of emails in CC group."""
  recipients = set()
  for ident in auth.list_group(CC_GROUP).members:
    if ident.is_user:
      try:
        mail.CheckEmailValid(ident.name, 'to')
        recipients.add(ident.name)
      except mail.InvalidEmailError:
        logging.error('invalid cc recipient %s', ident.name)
  return recipients


class BounceHandler(mail_handlers.BounceNotificationHandler):
  """Logs bounce notifications."""

  def receive(self, bounce_message):
    def to_text(msg):
      return 'Subject: %s\nTo: %s\n%s\n%s' % (
          msg['subject'],
          msg['to'],
          'CC: %s\n' % msg['cc'] if msg['cc'] else '',
          msg['text'],
      )

    logging.error(
        'Bounce notification\n%s', to_text(bounce_message.notification))
    logging.info(
        'Original message\n%s', to_text(bounce_message.original)
    )


def _send(email_message):
  # Mockable
  email_message.send()
