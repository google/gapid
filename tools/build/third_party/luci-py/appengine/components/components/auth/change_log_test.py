#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import utils
from components.auth import change_log
from components.auth import model
from components.auth.proto import security_config_pb2

from test_support import test_case


class MakeInitialSnapshotTest(test_case.TestCase):
  """Tests for ensure_initial_snapshot function."""

  def test_works(self):
    # Initial entities. Don't call 'record_revision' to imitate "old"
    # application without history related code.
    @ndb.transactional
    def make_auth_db():
      model.AuthGlobalConfig(key=model.root_key()).put()
      model.AuthIPWhitelistAssignments(
          key=model.ip_whitelist_assignments_key()).put()
      model.AuthGroup(key=model.group_key('A group')).put()
      model.AuthIPWhitelist(key=model.ip_whitelist_key('A whitelist')).put()
      model.replicate_auth_db()
    make_auth_db()

    # Bump auth_db once more to avoid hitting trivial case of "processing first
    # revision ever".
    auth_db_rev = ndb.transaction(model.replicate_auth_db)
    self.assertEqual(2, auth_db_rev)

    # Now do the work.
    change_log.ensure_initial_snapshot(auth_db_rev)

    # Generated new AuthDB rev with updated entities.
    self.assertEqual(3, model.get_auth_db_revision())

    # Check all *History entitites exist now.
    p = model.historical_revision_key(3)
    self.assertIsNotNone(
        ndb.Key('AuthGlobalConfigHistory', 'root', parent=p).get())
    self.assertIsNotNone(
        ndb.Key(
            'AuthIPWhitelistAssignmentsHistory', 'default', parent=p).get())
    self.assertIsNotNone(ndb.Key('AuthGroupHistory', 'A group', parent=p).get())
    self.assertIsNotNone(
        ndb.Key('AuthIPWhitelistHistory', 'A whitelist', parent=p).get())

    # Call again, should be noop (marker is set).
    change_log.ensure_initial_snapshot(3)
    self.assertEqual(3, model.get_auth_db_revision())


ident = lambda x: model.Identity.from_bytes('user:' + x)
glob = lambda x: model.IdentityGlob.from_bytes('user:' + x)


def make_group(name, comment, **kwargs):
  group = model.AuthGroup(key=model.group_key(name), **kwargs)
  group.record_revision(
      modified_by=ident('me@example.com'),
      modified_ts=utils.utcnow(),
      comment=comment)
  group.put()


def make_ip_whitelist(name, comment, **kwargs):
  wl = model.AuthIPWhitelist(key=model.ip_whitelist_key(name), **kwargs)
  wl.record_revision(
      modified_by=ident('me@example.com'),
      modified_ts=utils.utcnow(),
      comment=comment)
  wl.put()


def security_config(regexps):
  msg = security_config_pb2.SecurityConfig(internal_service_regexp=regexps)
  return msg.SerializeToString()


class GenerateChangesTest(test_case.TestCase):
  """Tests for generate_changes function."""

  def setUp(self):
    super(GenerateChangesTest, self).setUp()
    self.mock(change_log, 'enqueue_process_change_task', lambda _: None)
    self.mock_now(datetime.datetime(2015, 1, 2, 3, 4, 5))

  def auth_db_transaction(self, callback):
    """Imitates AuthDB change and subsequent 'process-change' task.

    Returns parent entity of entity subgroup with all generated changes.
    """
    @ndb.transactional
    def run():
      callback()
      return model.replicate_auth_db()
    auth_db_rev = run()
    change_log.process_change(auth_db_rev)
    return change_log.change_log_revision_key(auth_db_rev)

  def grab_all(self, ancestor):
    """Returns dicts with all entities under given ancestor."""
    entities = {}
    def cb(key):
      # Skip AuthDBLogRev itself, it's not interesting.
      if key == ancestor:
        return
      as_str = []
      k = key
      while k and k != ancestor:
        as_str.append('%s:%s' % (k.kind(), k.id()))
        k = k.parent()
      entities['/'.join(as_str)] = {
        prop: val for prop, val in key.get().to_dict().iteritems() if val
      }
    ndb.Query(ancestor=ancestor).map(cb, keys_only=True)
    return entities

  def test_works(self):
    # Touch all kinds of entities at once. More thorough tests for per-entity
    # changes are below.
    def touch_all():
      make_group(
          name='A group',
          members=[ident('a@example.com'), ident('b@example.com')],
          description='Blah',
          comment='New group')
      make_ip_whitelist(
          name='An IP whitelist',
          subnets=['127.0.0.1/32'],
          description='Bluh',
          comment='New IP whitelist')
      a = model.AuthIPWhitelistAssignments(
          key=model.ip_whitelist_assignments_key(),
          assignments=[
            model.AuthIPWhitelistAssignments.Assignment(
              identity=ident('a@example.com'),
              ip_whitelist='An IP whitelist')
          ])
      a.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='New assignment')
      a.put()
      c = model.AuthGlobalConfig(
          key=model.root_key(),
          oauth_client_id='client_id',
          oauth_client_secret='client_secret',
          oauth_additional_client_ids=['1', '2'])
      c.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Config change')
      c.put()

    changes = self.grab_all(self.auth_db_transaction(touch_all))
    self.assertEqual({
      'AuthDBChange:AuthGlobalConfig$root!7000': {
      'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_CONF_OAUTH_CLIENT_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'oauth_client_id': u'client_id',
        'oauth_client_secret': u'client_secret',
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGlobalConfig$root!7100': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_CONF_CLIENT_IDS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'oauth_additional_client_ids': [u'1', u'2'],
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_CREATED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'New group',
        'description': u'Blah',
        'owners': u'administrators',
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1200': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'New group',
        'members': [
          model.Identity(kind='user', name='a@example.com'),
          model.Identity(kind='user', name='b@example.com'),
        ],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelist$An IP whitelist!3000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_CREATED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'New IP whitelist',
        'description': u'Bluh',
        'target': u'AuthIPWhitelist$An IP whitelist',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelist$An IP whitelist!3200': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_SUBNETS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'New IP whitelist',
        'subnets': [u'127.0.0.1/32'],
        'target': u'AuthIPWhitelist$An IP whitelist',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelistAssignments'
          '$default$user:a@example.com!5000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWLASSIGN_SET,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistAssignmentChange'],
        'comment': u'New assignment',
        'identity': model.Identity(kind='user', name='a@example.com'),
        'ip_whitelist': u'An IP whitelist',
        'target': u'AuthIPWhitelistAssignments$default$user:a@example.com',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com')
      },
    }, changes)

  def test_groups_diff(self):
    def create():
      make_group(
          name='A group',
          members=[ident('a@example.com'), ident('b@example.com')],
          globs=[glob('*@example.com'), glob('*@other.com')],
          nested=['A', 'B'],
          description='Blah',
          comment='New group')
    changes = self.grab_all(self.auth_db_transaction(create))
    self.assertEqual({
      'AuthDBChange:AuthGroup$A group!1000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_CREATED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'New group',
        'description': u'Blah',
        'owners': u'administrators',
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1200': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'New group',
        'members': [
          model.Identity(kind='user', name='a@example.com'),
          model.Identity(kind='user', name='b@example.com'),
        ],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1400': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_GLOBS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'New group',
        'globs': [
          model.IdentityGlob(kind='user', pattern='*@example.com'),
          model.IdentityGlob(kind='user', pattern='*@other.com'),
        ],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1600': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_NESTED_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'New group',
        'nested': [u'A', u'B'],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

    def modify():
      g = model.group_key('A group').get()
      g.members = [ident('a@example.com'), ident('c@example.com')]
      g.globs = [glob('*@example.com'), glob('*@blah.com')]
      g.nested = ['A', 'C']
      g.description = 'Another blah'
      g.owners = 'another-owners'
      g.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Changed')
      g.put()
    changes = self.grab_all(self.auth_db_transaction(modify))
    self.assertEqual({
      'AuthDBChange:AuthGroup$A group!1100': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_DESCRIPTION_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'description': u'Another blah',
        'old_description': u'Blah',
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1150': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_OWNERS_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'old_owners': u'administrators',
        'owners': u'another-owners',
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1200': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'members': [model.Identity(kind='user', name='c@example.com')],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1300': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'members': [model.Identity(kind='user', name='b@example.com')],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1400': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_GLOBS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'globs': [model.IdentityGlob(kind='user', pattern='*@blah.com')],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1500': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_GLOBS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'globs': [model.IdentityGlob(kind='user', pattern='*@other.com')],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1600': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_NESTED_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'nested': [u'C'],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1700': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_NESTED_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Changed',
        'nested': [u'B'],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

    def delete():
      g = model.group_key('A group').get()
      g.record_deletion(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Deleted')
      g.key.delete()
    changes = self.grab_all(self.auth_db_transaction(delete))
    self.assertEqual({
      'AuthDBChange:AuthGroup$A group!1300': {
        'app_version': u'v1a',
        'auth_db_rev': 3,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Deleted',
        'members': [
          model.Identity(kind='user', name='a@example.com'),
          model.Identity(kind='user', name='c@example.com'),
        ],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1500': {
        'app_version': u'v1a',
        'auth_db_rev': 3,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_GLOBS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Deleted',
        'globs': [
          model.IdentityGlob(kind='user', pattern='*@example.com'),
          model.IdentityGlob(kind='user', pattern='*@blah.com'),
        ],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1700': {
        'app_version': u'v1a',
        'auth_db_rev': 3,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_NESTED_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Deleted',
        'nested': [u'A', u'C'],
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGroup$A group!1800': {
        'app_version': u'v1a',
        'auth_db_rev': 3,
        'change_type': change_log.AuthDBChange.CHANGE_GROUP_DELETED,
        'class_': [u'AuthDBChange', u'AuthDBGroupChange'],
        'comment': u'Deleted',
        'old_description': u'Another blah',
        'old_owners': u'another-owners',
        'target': u'AuthGroup$A group',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

  def test_ip_whitelists_diff(self):
    def create():
      make_ip_whitelist(
          name='A list',
          subnets=['127.0.0.1/32', '127.0.0.2/32'],
          description='Blah',
          comment='New list')
    changes = self.grab_all(self.auth_db_transaction(create))
    self.assertEqual({
      'AuthDBChange:AuthIPWhitelist$A list!3000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_CREATED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'New list',
        'description': u'Blah',
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelist$A list!3200': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_SUBNETS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'New list',
        'subnets': [u'127.0.0.1/32', u'127.0.0.2/32'],
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

    def modify():
      l = model.ip_whitelist_key('A list').get()
      l.subnets = ['127.0.0.1/32', '127.0.0.3/32']
      l.description = 'Another blah'
      l.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Changed')
      l.put()
    changes = self.grab_all(self.auth_db_transaction(modify))
    self.assertEqual({
      'AuthDBChange:AuthIPWhitelist$A list!3100': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_DESCRIPTION_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'Changed',
        'description': u'Another blah',
        'old_description': u'Blah',
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelist$A list!3200': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_SUBNETS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'Changed',
        'subnets': [u'127.0.0.3/32'],
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelist$A list!3300': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_SUBNETS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'Changed',
        'subnets': [u'127.0.0.2/32'],
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

    def delete():
      l = model.ip_whitelist_key('A list').get()
      l.record_deletion(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Deleted')
      l.key.delete()
    changes = self.grab_all(self.auth_db_transaction(delete))
    self.assertEqual({
      'AuthDBChange:AuthIPWhitelist$A list!3300': {
        'app_version': u'v1a',
        'auth_db_rev': 3,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_SUBNETS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'Deleted',
        'subnets': [u'127.0.0.1/32', u'127.0.0.3/32'],
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelist$A list!3400': {
        'app_version': u'v1a',
        'auth_db_rev': 3,
        'change_type': change_log.AuthDBChange.CHANGE_IPWL_DELETED,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistChange'],
        'comment': u'Deleted',
        'old_description': u'Another blah',
        'target': u'AuthIPWhitelist$A list',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

  def test_ip_wl_assignments_diff(self):
    def create():
      a = model.AuthIPWhitelistAssignments(
          key=model.ip_whitelist_assignments_key(),
          assignments=[
            model.AuthIPWhitelistAssignments.Assignment(
              identity=ident('a@example.com'),
              ip_whitelist='An IP whitelist'),
            model.AuthIPWhitelistAssignments.Assignment(
              identity=ident('b@example.com'),
              ip_whitelist='Another IP whitelist'),
          ])
      a.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='New assignment')
      a.put()
    changes = self.grab_all(self.auth_db_transaction(create))
    self.assertEqual({
      'AuthDBChange:AuthIPWhitelistAssignments$'
          'default$user:a@example.com!5000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWLASSIGN_SET,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistAssignmentChange'],
        'comment': u'New assignment',
        'identity': model.Identity(kind='user', name='a@example.com'),
        'ip_whitelist': u'An IP whitelist',
        'target': u'AuthIPWhitelistAssignments$default$user:a@example.com',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelistAssignments$'
          'default$user:b@example.com!5000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_IPWLASSIGN_SET,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistAssignmentChange'],
        'comment': u'New assignment',
        'identity': model.Identity(kind='user', name='b@example.com'),
        'ip_whitelist': u'Another IP whitelist',
        'target': u'AuthIPWhitelistAssignments$default$user:b@example.com',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

    def change():
      a = model.ip_whitelist_assignments_key().get()
      a.assignments=[
        model.AuthIPWhitelistAssignments.Assignment(
          identity=ident('a@example.com'),
          ip_whitelist='Another IP whitelist'),
        model.AuthIPWhitelistAssignments.Assignment(
          identity=ident('c@example.com'),
          ip_whitelist='IP whitelist'),
      ]
      a.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='change')
      a.put()
    changes = self.grab_all(self.auth_db_transaction(change))
    self.assertEqual({
      'AuthDBChange:AuthIPWhitelistAssignments$'
          'default$user:a@example.com!5000': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_IPWLASSIGN_SET,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistAssignmentChange'],
        'comment': u'change',
        'identity': model.Identity(kind='user', name='a@example.com'),
        'ip_whitelist': u'Another IP whitelist',
        'target': u'AuthIPWhitelistAssignments$default$user:a@example.com',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelistAssignments$'
          'default$user:b@example.com!5100': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_IPWLASSIGN_UNSET,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistAssignmentChange'],
        'comment': u'change',
        'identity': model.Identity(kind='user', name='b@example.com'),
        'ip_whitelist': u'Another IP whitelist',
        'target': u'AuthIPWhitelistAssignments$default$user:b@example.com',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthIPWhitelistAssignments$'
          'default$user:c@example.com!5000': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_IPWLASSIGN_SET,
        'class_': [u'AuthDBChange', u'AuthDBIPWhitelistAssignmentChange'],
        'comment': u'change',
        'identity': model.Identity(kind='user', name='c@example.com'),
        'ip_whitelist': u'IP whitelist',
        'target': u'AuthIPWhitelistAssignments$default$user:c@example.com',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

  def test_global_config_diff(self):
    def create():
      c = model.AuthGlobalConfig(
          key=model.root_key(),
          oauth_client_id='client_id',
          oauth_client_secret='client_secret',
          oauth_additional_client_ids=['1', '2'])
      c.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Config change')
      c.put()
    changes = self.grab_all(self.auth_db_transaction(create))
    self.assertEqual({
      'AuthDBChange:AuthGlobalConfig$root!7000': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_CONF_OAUTH_CLIENT_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'oauth_client_id': u'client_id',
        'oauth_client_secret': u'client_secret',
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGlobalConfig$root!7100': {
        'app_version': u'v1a',
        'auth_db_rev': 1,
        'change_type': change_log.AuthDBChange.CHANGE_CONF_CLIENT_IDS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'oauth_additional_client_ids': [u'1', u'2'],
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)

    def modify():
      c = model.root_key().get()
      c.oauth_additional_client_ids = ['1', '3']
      c.token_server_url = 'https://token-server'
      c.security_config = security_config(['hi'])
      c.record_revision(
          modified_by=ident('me@example.com'),
          modified_ts=utils.utcnow(),
          comment='Config change')
      c.put()
    changes = self.grab_all(self.auth_db_transaction(modify))
    self.assertEqual({
      'AuthDBChange:AuthGlobalConfig$root!7100': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_CONF_CLIENT_IDS_ADDED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'oauth_additional_client_ids': [u'3'],
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGlobalConfig$root!7200': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type': change_log.AuthDBChange.CHANGE_CONF_CLIENT_IDS_REMOVED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'oauth_additional_client_ids': [u'2'],
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGlobalConfig$root!7300': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type':
            change_log.AuthDBChange.CHANGE_CONF_TOKEN_SERVER_URL_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'target': u'AuthGlobalConfig$root',
        'token_server_url_new': u'https://token-server',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
      'AuthDBChange:AuthGlobalConfig$root!7400': {
        'app_version': u'v1a',
        'auth_db_rev': 2,
        'change_type':
            change_log.AuthDBChange.CHANGE_CONF_SECURITY_CONFIG_CHANGED,
        'class_': [u'AuthDBChange', u'AuthDBConfigChange'],
        'comment': u'Config change',
        'security_config_new': security_config(['hi']),
        'target': u'AuthGlobalConfig$root',
        'when': datetime.datetime(2015, 1, 2, 3, 4, 5),
        'who': model.Identity(kind='user', name='me@example.com'),
      },
    }, changes)


class AuthDBChangeTest(test_case.TestCase):
  # Test to_jsonins for AuthDBGroupChange and AuthDBIPWhitelistAssignmentChange,
  # the rest are trivial.

  def test_group_change_to_jsonish(self):
    c = change_log.AuthDBGroupChange(
        change_type=change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_ADDED,
        target='AuthGroup$abc',
        auth_db_rev=123,
        who=ident('a@example.com'),
        when=datetime.datetime(2015, 1, 2, 3, 4, 5),
        comment='A comment',
        app_version='v123',
        description='abc',
        members=[ident('a@a.com')],
        globs=[glob('*@a.com')],
        nested=['A'],
        owners='abc',
        old_owners='def')
    self.assertEqual({
      'app_version': 'v123',
      'auth_db_rev': 123,
      'change_type': 'GROUP_MEMBERS_ADDED',
      'comment': 'A comment',
      'description': 'abc',
      'globs': ['user:*@a.com'],
      'members': ['user:a@a.com'],
      'nested': ['A'],
      'old_description': None,
      'old_owners': 'def',
      'owners': 'abc',
      'target': 'AuthGroup$abc',
      'when': 1420167845000000,
      'who': 'user:a@example.com',
    }, c.to_jsonish())

  def test_wl_assignment_to_jsonish(self):
    c = change_log.AuthDBIPWhitelistAssignmentChange(
        change_type=change_log.AuthDBChange.CHANGE_GROUP_MEMBERS_ADDED,
        target='AuthIPWhitelistAssignments$default',
        auth_db_rev=123,
        who=ident('a@example.com'),
        when=datetime.datetime(2015, 1, 2, 3, 4, 5),
        comment='A comment',
        app_version='v123',
        identity=ident('b@example.com'),
        ip_whitelist='whitelist')
    self.assertEqual({
      'app_version': 'v123',
      'auth_db_rev': 123,
      'change_type': 'GROUP_MEMBERS_ADDED',
      'comment': 'A comment',
      'identity': 'user:b@example.com',
      'ip_whitelist': 'whitelist',
      'target': 'AuthIPWhitelistAssignments$default',
      'when': 1420167845000000,
      'who': 'user:a@example.com',
    }, c.to_jsonish())

  def test_security_config_change_to_jsonish(self):
    c = change_log.AuthDBConfigChange(
        change_type=change_log.AuthDBChange.CHANGE_CONF_SECURITY_CONFIG_CHANGED,
        target='AuthGlobalConfig$default',
        auth_db_rev=123,
        who=ident('a@example.com'),
        when=datetime.datetime(2015, 1, 2, 3, 4, 5),
        comment='A comment',
        app_version='v123',
        security_config_old=None,
        security_config_new=security_config(['hi']))
    self.assertEqual({
      'app_version': 'v123',
      'auth_db_rev': 123,
      'change_type': 'CONF_SECURITY_CONFIG_CHANGED',
      'comment': 'A comment',
      'oauth_additional_client_ids': [],
      'oauth_client_id': None,
      'oauth_client_secret': None,
      'security_config_new': {'internal_service_regexp': [u'hi']},
      'security_config_old': None,
      'target': 'AuthGlobalConfig$default',
      'token_server_url_new': None,
      'token_server_url_old': None,
      'when': 1420167845000000,
      'who': 'user:a@example.com',
    }, c.to_jsonish())


class ChangeLogQueryTest(test_case.TestCase):
  # We know that some indexes are required. But component can't declare them,
  # so don't check them.
  SKIP_INDEX_YAML_CHECK = True

  def test_is_changle_log_indexed(self):
    self.assertTrue(change_log.is_changle_log_indexed())

  def test_make_change_log_query(self):
    def mk_ch(tp, rev, target):
      ch = change_log.AuthDBChange(
          change_type=getattr(change_log.AuthDBChange, 'CHANGE_%s' % tp),
          auth_db_rev=rev,
          target=target)
      ch.key = change_log.make_change_key(ch)
      ch.put()

    def key(c):
      return '%s/%s' % (c.key.parent().id(), c.key.id())

    mk_ch('GROUP_CREATED', 1, 'AuthGroup$abc')
    mk_ch('GROUP_MEMBERS_ADDED', 1, 'AuthGroup$abc')
    mk_ch('GROUP_CREATED', 1, 'AuthGroup$another')
    mk_ch('GROUP_DELETED', 2, 'AuthGroup$abc')
    mk_ch('GROUP_MEMBERS_ADDED', 2, 'AuthGroup$another')

    # All. Most recent first. Largest even types first.
    q = change_log.make_change_log_query()
    self.assertEqual([
      '2/AuthGroup$another!1200',
      '2/AuthGroup$abc!1800',
      '1/AuthGroup$another!1000',
      '1/AuthGroup$abc!1200',
      '1/AuthGroup$abc!1000',
    ], map(key, q.fetch()))

    # Single revision only.
    q = change_log.make_change_log_query(auth_db_rev=1)
    self.assertEqual([
      '1/AuthGroup$another!1000',
      '1/AuthGroup$abc!1200',
      '1/AuthGroup$abc!1000',
    ], map(key, q.fetch()))

    # Single target only.
    q = change_log.make_change_log_query(target='AuthGroup$another')
    self.assertEqual([
      '2/AuthGroup$another!1200',
      '1/AuthGroup$another!1000',
    ], map(key, q.fetch()))

    # Single revision and single target.
    q = change_log.make_change_log_query(
        auth_db_rev=1, target='AuthGroup$another')
    self.assertEqual(['1/AuthGroup$another!1000'], map(key, q.fetch()))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
