#!/usr/bin/env python
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import test_env
test_env.setup_test_env()

import datetime
import mock

from components.config import validation
from components import auth
from components import gitiles
from components import template
from google.appengine.ext import ndb
from test_support import test_case

import notifications
import storage


class NotificationsTestCase(test_case.TestCase):
  def test_notify_gitiles_rejection(self):
    ctx = validation.Context()
    ctx.error('err')
    ctx.warning('warn')

    base = gitiles.Location.parse('https://example.com/x/+/infra/config')
    new_rev = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
    new_loc = base._replace(treeish=new_rev)
    old_rev = 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb'
    old_loc = base._replace(treeish=old_rev)

    self.mock(notifications, '_send', mock.Mock())

    john = gitiles.Contribution(
        'John', 'john@x.com', datetime.datetime(2015, 1, 1))
    commit = gitiles.Commit(
        sha=new_rev,
        tree='badcoffee',
        parents=[],
        author=john,
        committer=john,
        message='New config',
        tree_diff=None)
    self.mock(gitiles, 'get_log_async', mock.Mock(return_value=ndb.Future()))
    gitiles.get_log_async.return_value.set_result(
        gitiles.Log(commits=[commit], next_cursor=None))

    self.mock(template, 'render', mock.Mock())

    self.mock(auth, 'list_group', mock.Mock())
    auth.list_group.return_value = auth.GroupListing([
      auth.Identity('user', 'bill@x.com'),
      auth.Identity('service', 'foo'),
    ], [], [])

    # Notify.

    notifications.notify_gitiles_rejection('projects/x', new_loc, ctx.result())

    self.assertTrue(notifications._send.called)
    email = notifications._send.call_args[0][0]
    self.assertEqual(
        email.sender,
        'sample-app.appspot.com <noreply@sample-app.appspotmail.com>')
    self.assertEqual(email.subject, 'Config revision aaaaaaa is rejected')
    self.assertEqual(email.to, ['John <john@x.com>'])
    self.assertEqual(email.cc, {'bill@x.com'})

    template.render.assert_called_with(
        'templates/validation_notification.html',
        {
          'author': 'John',
          'messages': [
            {'severity': 'ERROR', 'text': 'err'},
            {'severity': 'WARNING', 'text': 'warn'}
          ],
          'rev_link': new_loc,
          'rev_hash': 'aaaaaaa',
          'rev_repo': 'x',
          'cur_rev_hash': None,
          'cur_rev_link': None,
        })

    # Do not send second time.
    notifications._send.reset_mock()
    notifications.notify_gitiles_rejection('projects/x', new_loc, ctx.result())
    self.assertFalse(notifications._send.called)

    # Now with config set.

    ndb.Key(notifications.Notification, str(new_loc)).delete()

    storage.ConfigSet(
      id='projects/x',
      latest_revision=old_rev,
      latest_revision_url=str(old_loc),
      location=str(base)
    ).put()

    template.render.reset_mock()
    notifications.notify_gitiles_rejection('projects/x', new_loc, ctx.result())
    template.render.assert_called_with(
        'templates/validation_notification.html',
        {
          'author': 'John',
          'messages': [
            {'severity': 'ERROR', 'text': 'err'},
            {'severity': 'WARNING', 'text': 'warn'}
          ],
          'rev_link': new_loc,
          'rev_hash': 'aaaaaaa',
          'rev_repo': 'x',
          'cur_rev_hash': 'bbbbbbb',
          'cur_rev_link': old_loc,
        })


if __name__ == '__main__':
  test_env.main()
