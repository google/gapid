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
from components.auth import model
from components.auth import replication
from test_support import test_case


def entity_to_dict(e):
  """Same as e.to_dict() but also adds entity key to the dict."""
  d = e.to_dict()
  d['__id__'] = e.key.id()
  d['__parent__'] = e.key.parent()
  return d


def snapshot_to_dict(snapshot):
  """AuthDBSnapshot -> dict (for comparisons)."""
  result = {
    'global_config': entity_to_dict(snapshot.global_config),
    'groups': [entity_to_dict(g) for g in snapshot.groups],
    'ip_whitelists': [entity_to_dict(l) for l in snapshot.ip_whitelists],
    'ip_whitelist_assignments':
        entity_to_dict(snapshot.ip_whitelist_assignments),
  }
  # Ensure no new keys are forgotten.
  assert len(snapshot) == len(result)
  return result


def make_snapshot_obj(
    global_config=None, groups=None,
    ip_whitelists=None, ip_whitelist_assignments=None):
  """Returns AuthDBSnapshot with empty list of groups and whitelists."""
  return replication.AuthDBSnapshot(
      global_config=global_config or model.AuthGlobalConfig(
          key=model.root_key(),
          oauth_client_id='oauth client id',
          oauth_client_secret='oauth client secret',
          token_server_url='token server',
          security_config='security config blob'),
      groups=groups or [],
      ip_whitelists=ip_whitelists or [],
      ip_whitelist_assignments=(
          ip_whitelist_assignments or
          model.AuthIPWhitelistAssignments(
              key=model.ip_whitelist_assignments_key())),
  )


class NewAuthDBSnapshotTest(test_case.TestCase):
  """Tests for new_auth_db_snapshot function."""

  def test_empty(self):
    state, snapshot = replication.new_auth_db_snapshot()
    self.assertIsNone(state)
    expected_snapshot = {
      'global_config': {
        '__id__': 'root',
        '__parent__': None,
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'modified_by': None,
        'modified_ts': None,
        'oauth_additional_client_ids': [],
        'oauth_client_id': u'',
        'oauth_client_secret': u'',
        'security_config': None,
        'token_server_url': u'',
      },
      'groups': [],
      'ip_whitelists': [],
      'ip_whitelist_assignments': {
        '__id__': 'default',
        '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
        'assignments': [],
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'modified_by': None,
        'modified_ts': None,
      },
    }
    self.assertEqual(expected_snapshot, snapshot_to_dict(snapshot))

  def test_non_empty(self):
    self.mock_now(datetime.datetime(2014, 1, 1, 1, 1, 1))

    state = model.AuthReplicationState(
        key=model.replication_state_key(),
        primary_id='blah',
        primary_url='https://blah',
        auth_db_rev=123)
    state.put()

    global_config = model.AuthGlobalConfig(
        key=model.root_key(),
        modified_ts=utils.utcnow(),
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
        oauth_client_id='oauth_client_id',
        oauth_client_secret='oauth_client_secret',
        oauth_additional_client_ids=['a', 'b'],
        token_server_url='https://token-server',
        security_config='security config blob')
    global_config.put()

    group = model.AuthGroup(
        key=model.group_key('Some group'),
        members=[model.Identity.from_bytes('user:a@example.com')],
        globs=[model.IdentityGlob.from_bytes('user:*@example.com')],
        nested=[],
        description='Some description',
        owners='owning-group',
        created_ts=utils.utcnow(),
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        modified_ts=utils.utcnow(),
        modified_by=model.Identity.from_bytes('user:modifier@example.com'))
    group.put()

    another = model.AuthGroup(
        key=model.group_key('Another group'),
        nested=['Some group'])
    another.put()

    ip_whitelist = model.AuthIPWhitelist(
        key=model.ip_whitelist_key('bots'),
        subnets=['127.0.0.1/32'],
        description='Some description',
        created_ts=utils.utcnow(),
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        modified_ts=utils.utcnow(),
        modified_by=model.Identity.from_bytes('user:modifier@example.com'))
    ip_whitelist.put()

    ip_whitelist_assignments = model.AuthIPWhitelistAssignments(
        key=model.ip_whitelist_assignments_key(),
        modified_ts=utils.utcnow(),
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
        assignments=[
          model.AuthIPWhitelistAssignments.Assignment(
            identity=model.Identity.from_bytes('user:bot_account@example.com'),
            ip_whitelist='bots',
            comment='some comment',
            created_ts=utils.utcnow(),
            created_by=model.Identity.from_bytes('user:creator@example.com')),
        ])
    ip_whitelist_assignments.put()

    captured_state, snapshot = replication.new_auth_db_snapshot()

    expected_state =  {
      'auth_db_rev': 123,
      'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
      'primary_id': u'blah',
      'primary_url': u'https://blah',
    }
    self.assertEqual(expected_state, captured_state.to_dict())

    expected_snapshot = {
      'global_config': {
        '__id__': 'root',
        '__parent__': None,
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'modified_by': model.Identity(kind='user', name='modifier@example.com'),
        'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
        'oauth_additional_client_ids': [u'a', u'b'],
        'oauth_client_id': u'oauth_client_id',
        'oauth_client_secret': u'oauth_client_secret',
        'security_config': 'security config blob',
        'token_server_url': u'https://token-server',
      },
      'groups': [
        {
          '__id__': 'Another group',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': None,
          'description': u'',
          'globs': [],
          'members': [],
          'modified_by': None,
          'modified_ts': None,
          'nested': [u'Some group'],
          'owners': u'administrators',
        },
        {
          '__id__': 'Some group',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': model.Identity(kind='user', name='creator@example.com'),
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'Some description',
          'globs': [model.IdentityGlob(kind='user', pattern='*@example.com')],
          'members': [model.Identity(kind='user', name='a@example.com')],
          'modified_by': model.Identity(
              kind='user', name='modifier@example.com'),
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'nested': [],
          'owners': u'owning-group',
        },
      ],
      'ip_whitelists': [
        {
          '__id__': 'bots',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': model.Identity(kind='user', name='creator@example.com'),
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'Some description',
          'modified_by': model.Identity(
              kind='user', name='modifier@example.com'),
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'subnets': [u'127.0.0.1/32'],
        },
      ],
      'ip_whitelist_assignments': {
        '__id__': 'default',
        '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
        'assignments': [
          {
            'comment': u'some comment',
            'created_by': model.Identity(
                kind='user', name='creator@example.com'),
            'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
            'identity': model.Identity(
                kind='user', name='bot_account@example.com'),
            'ip_whitelist': u'bots',
          },
        ],
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'modified_by': model.Identity(kind='user', name='modifier@example.com'),
        'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
      },
    }
    self.assertEqual(expected_snapshot, snapshot_to_dict(snapshot))


class SnapshotToProtoConversionTest(test_case.TestCase):
  """Tests for entities <-> proto conversion."""

  def assert_serialization_works(self, snapshot):
    """Ensures AuthDBSnapshot == AuthDBSnapshot -> proto -> AuthDBSnapshot."""
    roundtrip = replication.proto_to_auth_db_snapshot(
        replication.auth_db_snapshot_to_proto(snapshot))
    self.assertEqual(snapshot_to_dict(snapshot), snapshot_to_dict(roundtrip))

  def test_empty(self):
    """Serializing empty snapshot."""
    snapshot = make_snapshot_obj()
    self.assert_serialization_works(snapshot)

  def test_global_config_serialization(self):
    """Serializing snapshot with non-trivial AuthGlobalConfig."""
    snapshot = make_snapshot_obj(
        global_config=model.AuthGlobalConfig(
            key=model.root_key(),
            oauth_client_id=u'some-client-id',
            oauth_client_secret=u'some-client-secret',
            oauth_additional_client_ids=[u'id1', u'id2'],
            token_server_url=u'https://example.com',
            security_config='security config blob'))
    self.assert_serialization_works(snapshot)

  def test_group_serialization(self):
    """Serializing snapshot with non-trivial AuthGroup."""
    group = model.AuthGroup(
        key=model.group_key('some-group'),
        members=[
          model.Identity.from_bytes('user:punch@example.com'),
          model.Identity.from_bytes('user:judy@example.com'),
        ],
        globs=[model.IdentityGlob.from_bytes('user:*@example.com')],
        nested=['Group A', 'Group B'],
        description='Blah blah blah',
        created_ts=utils.utcnow(),
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        modified_ts=utils.utcnow(),
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
    )
    snapshot = make_snapshot_obj(groups=[group])
    self.assert_serialization_works(snapshot)

  def test_ip_whitelists_serialization(self):
    """Serializing snapshot with non-trivial IP whitelist."""
    ip_whitelist = model.AuthIPWhitelist(
        key=model.ip_whitelist_key('bots'),
        subnets=['127.0.0.1/32'],
        description='Blah blah blah',
        created_ts=utils.utcnow(),
        created_by=model.Identity.from_bytes('user:creator@example.com'),
        modified_ts=utils.utcnow(),
        modified_by=model.Identity.from_bytes('user:modifier@example.com'),
    )
    snapshot = make_snapshot_obj(ip_whitelists=[ip_whitelist])
    self.assert_serialization_works(snapshot)

  def test_ip_whitelist_assignments_serialization(self):
    """Serializing snapshot with non-trivial AuthIPWhitelistAssignments."""
    entity = model.AuthIPWhitelistAssignments(
        key=model.ip_whitelist_assignments_key(),
        assignments=[
          model.AuthIPWhitelistAssignments.Assignment(
            identity=model.Identity.from_bytes('user:a@example.com'),
            ip_whitelist='some whitelist',
            comment='some comment',
            created_ts=utils.utcnow(),
            created_by=model.Identity.from_bytes('user:creator@example.com'),
          ),
        ],
    )
    snapshot = make_snapshot_obj(ip_whitelist_assignments=entity)
    self.assert_serialization_works(snapshot)


class ReplaceAuthDbTest(test_case.TestCase):
  """Tests for replace_auth_db function."""

  @staticmethod
  def configure_as_replica(auth_db_rev=0, modified_ts=None):
    state = model.AuthReplicationState(
         key=model.replication_state_key(),
         primary_id='primary',
         primary_url='https://primary',
         auth_db_rev=auth_db_rev,
         modified_ts=modified_ts)
    state.put()

  def test_works(self):
    self.mock_now(datetime.datetime(2014, 1, 1, 1, 1, 1))
    self.configure_as_replica(0)

    # Prepare auth db state.
    model.AuthGlobalConfig(
        key=model.root_key(),
        modified_ts=utils.utcnow(),
        oauth_client_id='oauth_client_id',
        oauth_client_secret='oauth_client_secret',
        oauth_additional_client_ids=['a', 'b']).put()

    def group(name, **kwargs):
      return model.AuthGroup(
          key=model.group_key(name),
          created_ts=utils.utcnow(),
          modified_ts=utils.utcnow(),
          **kwargs)
    group('Modify').put()
    group('Delete').put()
    group('Keep').put()

    def ip_whitelist(name, **kwargs):
      return model.AuthIPWhitelist(
          key=model.ip_whitelist_key(name),
          created_ts=utils.utcnow(),
          modified_ts=utils.utcnow(),
          **kwargs)
    ip_whitelist('modify').put()
    ip_whitelist('delete').put()
    ip_whitelist('keep').put()

    def assignment(ident, ip_whitelist):
      return model.AuthIPWhitelistAssignments.Assignment(
          identity=model.Identity.from_bytes(ident),
          ip_whitelist=ip_whitelist,
          created_ts=utils.utcnow(),
          comment='comment')
    model.AuthIPWhitelistAssignments(
        key=model.ip_whitelist_assignments_key(),
        modified_ts=utils.utcnow(),
        assignments=[
          assignment('user:1@example.com', 'modify'),
          assignment('user:2@example.com', 'delete'),
          assignment('user:3@example.com', 'keep'),
        ]).put()

    # Prepare snapshot.
    snapshot = replication.AuthDBSnapshot(
        global_config=model.AuthGlobalConfig(
            key=model.root_key(),
            modified_ts=utils.utcnow(),
            oauth_client_id='another_oauth_client_id',
            oauth_client_secret='another_oauth_client_secret',
            oauth_additional_client_ids=[],
            token_server_url='https://token-server',
            security_config='security config blob'),
        groups=[
          group('New'),
          group('Modify', description='blah', owners='some-other-owners'),
          group('Keep'),
        ],
        ip_whitelists=[
          ip_whitelist('new', subnets=['1.1.1.1/32']),
          ip_whitelist('modify', subnets=['127.0.0.1/32', '192.168.0.1/32']),
          ip_whitelist('keep'),
        ],
        ip_whitelist_assignments=model.AuthIPWhitelistAssignments(
            key=model.ip_whitelist_assignments_key(),
            assignments=[
              assignment('user:a@example.com', 'new'),
              assignment('user:b@example.com', 'modify'),
              assignment('user:c@example.com', 'keep'),
            ],
        ),
    )

    # Push it.
    updated, state = replication.replace_auth_db(
        auth_db_rev=1234,
        modified_ts=datetime.datetime(2014, 1, 1, 1, 1, 1),
        snapshot=snapshot)
    self.assertTrue(updated)
    expected_state = {
      'auth_db_rev': 1234,
      'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
      'primary_id': u'primary',
      'primary_url': u'https://primary',
    }
    self.assertEqual(expected_state, state.to_dict())

    # Verify expected Auth db state.
    current_state, current_snapshot = replication.new_auth_db_snapshot()
    self.assertEqual(expected_state, current_state.to_dict())

    expected_auth_db = {
      'global_config': {
        '__id__': 'root',
        '__parent__': None,
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'modified_by': None,
        'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
        'oauth_additional_client_ids': [],
        'oauth_client_id': u'another_oauth_client_id',
        'oauth_client_secret': u'another_oauth_client_secret',
        'security_config': 'security config blob',
        'token_server_url': u'https://token-server',
      },
      'groups': [
        {
          '__id__': 'Keep',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'',
          'globs': [],
          'members': [],
          'modified_by': None,
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'nested': [],
          'owners': u'administrators',
        },
        {
          '__id__': 'Modify',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'blah',
          'globs': [],
          'members': [],
          'modified_by': None,
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'nested': [],
          'owners': u'some-other-owners',
        },
        {
          '__id__': 'New',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'',
          'globs': [],
          'members': [],
          'modified_by': None,
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'nested': [],
          'owners': u'administrators',
        },
      ],
      'ip_whitelists': [
        {
          '__id__': 'keep',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'',
          'modified_by': None,
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'subnets': [],
        },
        {
          '__id__': 'modify',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'',
          'modified_by': None,
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'subnets': [u'127.0.0.1/32', u'192.168.0.1/32'],
        },
        {
          '__id__': 'new',
          '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
          'auth_db_rev': None,
          'auth_db_prev_rev': None,
          'created_by': None,
          'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'description': u'',
          'modified_by': None,
          'modified_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
          'subnets': [u'1.1.1.1/32'],
        },
      ],
      'ip_whitelist_assignments': {
        '__id__': 'default',
        '__parent__': ndb.Key('AuthGlobalConfig', 'root'),
        'assignments': [
          {
            'comment': u'comment',
            'created_by': None,
            'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
            'identity': model.Identity(kind='user', name='a@example.com'),
            'ip_whitelist': u'new',
          },
          {
            'comment': u'comment',
            'created_by': None,
            'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
            'identity': model.Identity(kind='user', name='b@example.com'),
            'ip_whitelist': u'modify',
          },
          {
            'comment': u'comment',
            'created_by': None,
            'created_ts': datetime.datetime(2014, 1, 1, 1, 1, 1),
            'identity': model.Identity(kind='user', name='c@example.com'),
            'ip_whitelist': u'keep',
          },
        ],
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'modified_by': None,
        'modified_ts': None, # not transfered currently in proto
      },
    }
    self.assertEqual(expected_auth_db, snapshot_to_dict(current_snapshot))

  def test_old_rev(self):
    """Refuses to push with old auth_db revision."""
    self.configure_as_replica(123, datetime.datetime(2000, 1, 1, 1, 1, 1))
    updated, state = replication.replace_auth_db(
        auth_db_rev=123,
        modified_ts=datetime.datetime(2014, 1, 1, 1, 1, 1),
        snapshot=make_snapshot_obj())
    self.assertFalse(updated)
    # Old modified_ts, update is not applied.
    expected_state = {
      'auth_db_rev': 123,
      'modified_ts': datetime.datetime(2000, 1, 1, 1, 1, 1),
      'primary_id': u'primary',
      'primary_url': u'https://primary',
    }
    self.assertEqual(expected_state, state.to_dict())


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
