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
from components.auth import ipaddr
from components.auth import model
from test_support import test_case


class IdentityTest(test_case.TestCase):
  """Tests for Identity class."""

  def test_immutable(self):
    # Note that it's still possible to add new attributes to |ident|. To fix
    # this we'd have to add __slots__ = () to Identity and to BytesSerializable
    # (it inherits from). Since adding extra attributes to an instance doesn't
    # harm any expected behavior of Identity (like equality operator or
    # serialization) we ignore this hole in immutability.
    ident = model.Identity(model.IDENTITY_USER, 'joe@example.com')
    self.assertTrue(isinstance(ident, tuple))
    with self.assertRaises(AttributeError):
      ident.kind = model.IDENTITY_USER
    with self.assertRaises(AttributeError):
      ident.name = 'bob@example.com'

  def test_equality(self):
    # Identities are compared by values, not by reference.
    ident1 = model.Identity(model.IDENTITY_USER, 'joe@example.com')
    ident2 = model.Identity(model.IDENTITY_USER, 'joe@example.com')
    ident3 = model.Identity(model.IDENTITY_USER, 'bob@example.com')
    self.assertEqual(ident1, ident2)
    self.assertNotEqual(ident1, ident3)
    # Verify that adding extra attribute doesn't change equality relation.
    ident1.extra = 1
    ident2.extra = 2
    self.assertEqual(ident1, ident2)

  def test_validation(self):
    # Unicode with ASCII data is ok.
    ok_identities = (
      (unicode(model.IDENTITY_USER), 'joe@example.com'),
      (model.IDENTITY_USER, u'joe@example.com'),
      (model.IDENTITY_USER, r'abc%def.com@zzz.com'),
      (model.IDENTITY_USER, r'ABC_DEF@ABC_DEF.com'),
      (model.IDENTITY_SERVICE, 'domain.com:app-id'),
      (model.IDENTITY_PROJECT, 'project-123_name'),
    )
    for kind, name in ok_identities:
      ident = model.Identity(kind, name)
      # Should be 'str', not 'unicode'
      self.assertEqual(type(ident.kind), str)
      self.assertEqual(type(ident.name), str)
      # And data should match.
      self.assertEqual(kind, ident.kind)
      self.assertEqual(name, ident.name)

    # Nasty stuff.
    bad_identities = (
      ('unknown-kind', 'joe@example.com'),
      (model.IDENTITY_ANONYMOUS, 'not-anonymous'),
      (model.IDENTITY_BOT, 'bad bot name - spaces'),
      (model.IDENTITY_SERVICE, 'spaces everywhere'),
      (model.IDENTITY_USER, 'even here'),
      (model.IDENTITY_USER, u'\u043f\u0440\u0438\u0432\u0435\u0442'),
      (model.IDENTITY_PROJECT, 'UPPER_not_allowed'),
    )
    for kind, name in bad_identities:
      with self.assertRaises(ValueError):
        model.Identity(kind, name)

  def test_serialization(self):
    # Identity object goes through serialize-deserialize process unchanged.
    good_cases = (
      model.Identity(model.IDENTITY_USER, 'joe@example.com'),
      model.Anonymous,
    )
    for case in good_cases:
      self.assertEqual(case, model.Identity.from_bytes(case.to_bytes()))

    # Malformed data causes ValueError.
    bad_cases = (
      '',
      'userjoe@example.com',
      'user:',
      ':joe@example.com',
      'user::joe@example.com',
    )
    for case in bad_cases:
      with self.assertRaises(ValueError):
        model.Identity.from_bytes(case)


class IdentityGlobTest(test_case.TestCase):
  """Tests for IdentityGlob class."""

  def test_immutable(self):
    # See comment in IdentityTest.test_immutable regarding existing hole in
    # immutability.
    glob = model.IdentityGlob(model.IDENTITY_USER, '*@example.com')
    self.assertTrue(isinstance(glob, tuple))
    with self.assertRaises(AttributeError):
      glob.kind = model.IDENTITY_USER
    with self.assertRaises(AttributeError):
      glob.pattern = '*@example.com'

  def test_equality(self):
    # IdentityGlobs are compared by values, not by reference.
    glob1 = model.IdentityGlob(model.IDENTITY_USER, '*@example.com')
    glob2 = model.IdentityGlob(model.IDENTITY_USER, '*@example.com')
    glob3 = model.IdentityGlob(model.IDENTITY_USER, '*-sub@example.com')
    self.assertEqual(glob1, glob2)
    self.assertNotEqual(glob1, glob3)
    # Verify that adding extra attribute doesn't change equality relation.
    glob1.extra = 1
    glob2.extra = 2
    self.assertEqual(glob1, glob2)

  def test_validation(self):
    # Unicode with ASCII data is ok.
    ok_globs = (
      (unicode(model.IDENTITY_USER), '*@example.com'),
      (model.IDENTITY_USER, u'*@example.com'),
    )
    for kind, pattern in ok_globs:
      glob = model.IdentityGlob(kind, pattern)
      # Should be 'str', not 'unicode'
      self.assertEqual(type(glob.kind), str)
      self.assertEqual(type(glob.pattern), str)
      # And data should match.
      self.assertEqual(kind, glob.kind)
      self.assertEqual(pattern, glob.pattern)

    # Nasty stuff.
    bad_globs = (
      ('unknown-kind', '*@example.com'),
      (model.IDENTITY_USER, ''),
      (model.IDENTITY_USER, u'\u043f\u0440\u0438\u0432\u0435\u0442')
    )
    for kind, pattern in bad_globs:
      with self.assertRaises(ValueError):
        model.IdentityGlob(kind, pattern)

  def test_serialization(self):
    # IdentityGlob object goes through serialize-deserialize process unchanged.
    glob = model.IdentityGlob(model.IDENTITY_USER, '*@example.com')
    self.assertEqual(glob, model.IdentityGlob.from_bytes(glob.to_bytes()))

    # Malformed data causes ValueError.
    bad_cases = (
      '',
      'user*@example.com',
      'user:',
      ':*@example.com',
    )
    for case in bad_cases:
      with self.assertRaises(ValueError):
        model.IdentityGlob.from_bytes(case)

  def test_match(self):
    glob = model.IdentityGlob(model.IDENTITY_USER, '*@example.com')
    self.assertTrue(
        glob.match(model.Identity(model.IDENTITY_USER, 'a@example.com')))
    self.assertFalse(
        glob.match(model.Identity(model.IDENTITY_BOT, 'a@example.com')))
    self.assertFalse(
        glob.match(model.Identity(model.IDENTITY_USER, 'a@test.com')))


class AuthSecretTest(test_case.TestCase):
  """Tests for AuthSecret class."""

  def setUp(self):
    super(AuthSecretTest, self).setUp()
    self.mock(model.logging, 'warning', lambda *_args: None)

  def test_bootstrap_works(self):
    # Creating it for a first time.
    ent1 = model.AuthSecret.bootstrap('test_secret', length=127)
    self.assertTrue(ent1)
    self.assertEqual(ent1.key.string_id(), 'test_secret')
    self.assertEqual(ent1.key.parent().string_id(), 'local')
    self.assertEqual(1, len(ent1.values))
    self.assertEqual(127, len(ent1.values[0]))
    # Getting same one.
    ent2 = model.AuthSecret.bootstrap('test_secret')
    self.assertEqual(ent1, ent2)


def make_group(group_id, nested=(), owners=model.ADMIN_GROUP, store=True):
  """Makes a new AuthGroup to use in test, puts it in datastore."""
  entity = model.AuthGroup(
      key=model.group_key(group_id), nested=nested, owners=owners)
  if store:
    entity.put()
  return entity


class GroupBootstrapTest(test_case.TestCase):
  """Test for bootstrap_group function."""

  def test_group_bootstrap_empty(self):
    mocked_now = datetime.datetime(2014, 01, 01)
    self.mock_now(mocked_now)

    added = model.bootstrap_group('some-group', [], 'Blah description')
    self.assertTrue(added)

    ent = model.group_key('some-group').get()
    self.assertEqual(
        {
          'auth_db_rev': 1,
          'auth_db_prev_rev': None,
          'created_by': model.get_service_self_identity(),
          'created_ts': mocked_now,
          'description': 'Blah description',
          'globs': [],
          'members': [],
          'modified_by': model.get_service_self_identity(),
          'modified_ts': mocked_now,
          'nested': [],
          'owners': u'administrators',
        },
        ent.to_dict())

  def test_group_bootstrap_non_empty(self):
    ident1 = model.Identity(model.IDENTITY_USER, 'joe@example.com')
    ident2 = model.Identity(model.IDENTITY_USER, 'sam@example.com')

    mocked_now = datetime.datetime(2014, 01, 01)
    self.mock_now(mocked_now)

    added = model.bootstrap_group(
        'some-group', [ident1, ident2], 'Blah description')
    self.assertTrue(added)

    ent = model.group_key('some-group').get()
    self.assertEqual(
        {
          'auth_db_rev': 1,
          'auth_db_prev_rev': None,
          'created_by': model.get_service_self_identity(),
          'created_ts': mocked_now,
          'description': 'Blah description',
          'globs': [],
          'members': [ident1, ident2],
          'modified_by': model.get_service_self_identity(),
          'modified_ts': mocked_now,
          'nested': [],
          'owners': u'administrators',
        },
        ent.to_dict())


class FindGroupReferencesTest(test_case.TestCase):
  """Tests for find_referencing_groups function."""

  def test_missing_group(self):
    """Non existent group is not references by anything."""
    self.assertEqual(set(), model.find_referencing_groups('Missing group'))

  def test_not_referenced(self):
    """Existing orphaned groups is not referenced."""
    # Some mix of groups with references.
    make_group('Group 1')
    make_group('Group 2')
    make_group('Group 3', nested=('Group 1', 'Group 2'))
    make_group('Group 4', nested=('Group 3',))

    # And a group that is not referenced by anything.
    make_group('Standalone')

    # Should not be referenced.
    self.assertEqual(set(), model.find_referencing_groups('Standalone'))

  def test_referenced_as_nested_group(self):
    """If group is nested into another group, it's referenced."""
    # Some mix of groups with references, including group to be tested.
    make_group('Referenced')
    make_group('Group 1')
    make_group('Group 2', nested=('Referenced', 'Group 1'))
    make_group('Group 3', nested=('Group 2',))
    make_group('Group 4', nested=('Referenced',))

    # Only direct references are returned.
    self.assertEqual(
        set(['Group 2', 'Group 4']),
        model.find_referencing_groups('Referenced'))

  def test_referenced_as_owner(self):
    """If a group owns another group, it is referenced."""
    make_group('Referenced')
    make_group('Group 1', owners='Referenced')
    make_group('Group 2', owners='Referenced')
    make_group('Group 3', owners='Group 1')
    self.assertEqual(
        set(['Group 1', 'Group 2']),
        model.find_referencing_groups('Referenced'))


class FindDependencyCycleTest(test_case.TestCase):
  """Tests for find_group_dependency_cycle function."""

  def test_empty(self):
    group = make_group('A', store=False)
    self.assertEqual([], model.find_group_dependency_cycle(group))

  def test_no_cycles(self):
    make_group('A')
    make_group('B', nested=('A',))
    group = make_group('C', nested=('B',), store=False)
    self.assertEqual([], model.find_group_dependency_cycle(group))

  def test_self_reference(self):
    group = make_group('A', nested=('A',), store=False)
    self.assertEqual(['A'], model.find_group_dependency_cycle(group))

  def test_simple_cycle(self):
    make_group('A', nested=('B',))
    group = make_group('B', nested=('A',), store=False)
    self.assertEqual(['B', 'A'], model.find_group_dependency_cycle(group))

  def test_long_cycle(self):
    make_group('A', nested=('B',))
    make_group('B', nested=('C',))
    make_group('C', nested=('D',))
    group = make_group('D', nested=('A',), store=False)
    self.assertEqual(
        ['D', 'A', 'B', 'C'], model.find_group_dependency_cycle(group))

  def test_diamond_no_cycles(self):
    make_group('A')
    make_group('B1', nested=('A',))
    make_group('B2', nested=('A',))
    group = make_group('C', nested=('B1', 'B2'), store=False)
    self.assertEqual([], model.find_group_dependency_cycle(group))

  def test_diamond_with_cycles(self):
    make_group('A', nested=('C',))
    make_group('B1', nested=('A',))
    make_group('B2', nested=('A',))
    group = make_group('C', nested=('B1', 'B2'), store=False)
    self.assertEqual(['C', 'B1', 'A'], model.find_group_dependency_cycle(group))


class IpWhitelistTest(test_case.TestCase):
  """Tests for AuthIPWhitelist related functions."""

  def test_bootstrap_ip_whitelist_empty(self):
    self.assertIsNone(model.ip_whitelist_key('list').get())

    mocked_now = datetime.datetime(2014, 01, 01)
    self.mock_now(mocked_now)

    ret = model.bootstrap_ip_whitelist('list', [], 'comment')
    self.assertTrue(ret)

    ent = model.ip_whitelist_key('list').get()
    self.assertTrue(ent)
    self.assertEqual({
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'created_by': model.get_service_self_identity(),
      'created_ts': mocked_now,
      'description': u'comment',
      'modified_by': model.get_service_self_identity(),
      'modified_ts': mocked_now,
      'subnets': [],
    }, ent.to_dict())

  def test_bootstrap_ip_whitelist(self):
    self.assertIsNone(model.ip_whitelist_key('list').get())

    mocked_now = datetime.datetime(2014, 01, 01)
    self.mock_now(mocked_now)

    ret = model.bootstrap_ip_whitelist(
        'list', ['192.168.0.0/24', '127.0.0.1/32'], 'comment')
    self.assertTrue(ret)

    ent = model.ip_whitelist_key('list').get()
    self.assertTrue(ent)
    self.assertEqual({
      'auth_db_rev': 1,
      'auth_db_prev_rev': None,
      'created_by': model.get_service_self_identity(),
      'created_ts': mocked_now,
      'description': u'comment',
      'modified_by': model.get_service_self_identity(),
      'modified_ts': mocked_now,
      'subnets': [u'192.168.0.0/24', u'127.0.0.1/32'],
    }, ent.to_dict())

  def test_bootstrap_ip_whitelist_bad_subnet(self):
    self.assertFalse(model.bootstrap_ip_whitelist('list', ['not a subnet']))

  def test_bootstrap_ip_whitelist_assignment_new(self):
    self.mock_now(datetime.datetime(2014, 01, 01))

    ret = model.bootstrap_ip_whitelist_assignment(
        model.Identity(model.IDENTITY_USER, 'a@example.com'),
        'some ip whitelist', 'some comment')
    self.assertTrue(ret)

    self.assertEqual(
      {
        'assignments': [
          {
            'comment': 'some comment',
            'created_by': model.get_service_self_identity(),
            'created_ts': datetime.datetime(2014, 1, 1),
            'identity': model.Identity(model.IDENTITY_USER, 'a@example.com'),
            'ip_whitelist': 'some ip whitelist',
          },
        ],
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2014, 1, 1),
      }, model.ip_whitelist_assignments_key().get().to_dict())

  def test_bootstrap_ip_whitelist_assignment_modify(self):
    self.mock_now(datetime.datetime(2014, 01, 01))

    ret = model.bootstrap_ip_whitelist_assignment(
        model.Identity(model.IDENTITY_USER, 'a@example.com'),
        'some ip whitelist', 'some comment')
    self.assertTrue(ret)

    ret = model.bootstrap_ip_whitelist_assignment(
        model.Identity(model.IDENTITY_USER, 'a@example.com'),
        'another ip whitelist', 'another comment')
    self.assertTrue(ret)

    self.assertEqual(
      {
        'assignments': [
          {
            'comment': 'another comment',
            'created_by': model.get_service_self_identity(),
            'created_ts': datetime.datetime(2014, 1, 1),
            'identity': model.Identity(model.IDENTITY_USER, 'a@example.com'),
            'ip_whitelist': 'another ip whitelist',
          },
        ],
        'auth_db_rev': 2,
        'auth_db_prev_rev': 1,
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2014, 1, 1),
      }, model.ip_whitelist_assignments_key().get().to_dict())

  def test_is_ip_whitelisted(self):
    ent = model.AuthIPWhitelist(subnets=['127.0.0.1', '192.168.0.0/24'])
    test = lambda ip: ent.is_ip_whitelisted(ipaddr.ip_from_string(ip))
    self.assertTrue(test('127.0.0.1'))
    self.assertTrue(test('192.168.0.0'))
    self.assertTrue(test('192.168.0.9'))
    self.assertTrue(test('192.168.0.255'))
    self.assertFalse(test('192.168.1.0'))
    self.assertFalse(test('192.1.0.0'))

  def test_fetch_ip_whitelists_empty(self):
    assignments, whitelists = model.fetch_ip_whitelists()
    self.assertEqual(model.ip_whitelist_assignments_key(), assignments.key)
    self.assertEqual(0, len(assignments.assignments))
    self.assertEqual([], whitelists)

  def test_fetch_ip_whitelists_non_empty(self):
    ent = model.AuthIPWhitelistAssignments(
        key=model.ip_whitelist_assignments_key())

    def add(identity, **kwargs):
      kwargs['identity'] = model.Identity.from_bytes(identity)
      ent.assignments.append(
          model.AuthIPWhitelistAssignments.Assignment(**kwargs))
    add('user:a1@example.com', ip_whitelist='A')
    add('user:a2@example.com', ip_whitelist='A')
    add('user:b@example.com', ip_whitelist='B')
    add('user:c@example.com', ip_whitelist='missing')
    ent.put()

    def store_whitelist(name):
      model.AuthIPWhitelist(key=model.ip_whitelist_key(name)).put()
    store_whitelist('A')
    store_whitelist('B')
    store_whitelist('bots')

    assignments, whitelists = model.fetch_ip_whitelists()
    self.assertEqual(ent.to_dict(), assignments.to_dict())
    self.assertEqual(['A', 'B', 'bots'], [e.key.id() for e in whitelists])


class AuditLogTest(test_case.TestCase):
  """Tests to verify replicate_auth_db() keeps historical copies of entities."""

  def grab_log(self, original_cls):
    copies = original_cls.get_historical_copy_class().query(
        ancestor=model.root_key()).fetch()
    # All keys under correct historical_revision_key().
    for c in copies:
      self.assertEqual(
          ndb.Key('Rev', c.auth_db_rev, parent=model.root_key()),
          c.key.parent())
    return {x.key: x.to_dict() for x in copies}

  def setUp(self):
    super(AuditLogTest, self).setUp()
    self.mock_now(datetime.datetime(2015, 1, 1, 1, 1))

  def test_global_config_log(self):
    @ndb.transactional
    def modify(**kwargs):
      e = model.root_key().get() or model.AuthGlobalConfig(key=model.root_key())
      e.populate(**kwargs)
      e.record_revision(
          modified_by=model.Identity.from_bytes('user:a@example.com'),
          modified_ts=utils.utcnow(),
          comment='Comment')
      e.put()
      model.replicate_auth_db()

    # Global config is never deleted, so test only modifications.
    modify(oauth_client_id='1', oauth_additional_client_ids=[])
    modify(oauth_client_id='2', oauth_additional_client_ids=['a'])
    modify(oauth_client_id='3', oauth_additional_client_ids=['a', 'b'])
    modify(oauth_client_id='4', oauth_additional_client_ids=[])
    modify(oauth_client_id='4', security_config='zzz')

    # Final state.
    self.assertEqual({
      'auth_db_rev': 5,
      'auth_db_prev_rev': 4,
      'oauth_additional_client_ids': [],
      'oauth_client_id': u'4',
      'oauth_client_secret': u'',
      'security_config': 'zzz',
      'token_server_url': u'',
      'modified_by': model.Identity.from_bytes('user:a@example.com'),
      'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
    }, model.root_key().get().to_dict())

    # Copies in the history.
    cpy = lambda rev: ndb.Key(
        'Rev', rev, 'AuthGlobalConfigHistory', 'root', parent=model.root_key())
    self.assertEqual({
      cpy(1): {
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'oauth_additional_client_ids': [],
        'oauth_client_id': u'1',
        'oauth_client_secret': u'',
        'security_config': None,
        'token_server_url': u'',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
      cpy(2): {
        'auth_db_rev': 2,
        'auth_db_prev_rev': 1,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'oauth_additional_client_ids': [u'a'],
        'oauth_client_id': u'2',
        'oauth_client_secret': u'',
        'security_config': None,
        'token_server_url': u'',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
      cpy(3): {
        'auth_db_rev': 3,
        'auth_db_prev_rev': 2,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'oauth_additional_client_ids': [u'a', u'b'],
        'oauth_client_id': u'3',
        'oauth_client_secret': u'',
        'security_config': None,
        'token_server_url': u'',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
      cpy(4): {
        'auth_db_rev': 4,
        'auth_db_prev_rev': 3,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'oauth_additional_client_ids': [],
        'oauth_client_id': u'4',
        'oauth_client_secret': u'',
        'security_config': None,
        'token_server_url': u'',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
      cpy(5): {
        'auth_db_rev': 5,
        'auth_db_prev_rev': 4,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'oauth_additional_client_ids': [],
        'oauth_client_id': u'4',
        'oauth_client_secret': u'',
        'security_config': 'zzz',
        'token_server_url': u'',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
    }, self.grab_log(model.AuthGlobalConfig))

  def test_groups_log(self):
    ident_a = model.Identity.from_bytes('user:a@example.com')
    ident_b = model.Identity.from_bytes('user:b@example.com')

    glob_a = model.IdentityGlob.from_bytes('user:*@a.com')
    glob_b = model.IdentityGlob.from_bytes('user:*@b.com')

    @ndb.transactional
    def modify(name, commit=True, **kwargs):
      k = model.group_key(name)
      e = k.get()
      if not e:
        e = model.AuthGroup(
            key=k,
            created_by=ident_a,
            created_ts=utils.utcnow())
      e.record_revision(
          modified_by=ident_a,
          modified_ts=utils.utcnow(),
          comment='Comment')
      e.populate(**kwargs)
      e.put()
      if commit:
        model.replicate_auth_db()

    @ndb.transactional
    def remove(name, commit=True):
      e = model.group_key(name).get()
      if e:
        e.record_deletion(
            modified_by=model.Identity.from_bytes('user:a@example.com'),
            modified_ts=utils.utcnow(),
            comment='Comment')
        e.key.delete()
      if commit:
        model.replicate_auth_db()

    modify('A', members=[])
    modify('A', members=[ident_a], globs=[glob_a])
    modify('B', members=[ident_b], globs=[glob_b])
    modify('A', nested=['B'])
    @ndb.transactional
    def batch():
      modify('B', commit=False, description='Blah')
      remove('A', commit=True)
    batch()
    modify('B', members=[ident_a, ident_b], globs=[glob_a, glob_b])

    # Final state.
    self.assertIsNone(model.group_key('A').get())
    self.assertEqual({
      'auth_db_rev': 6,
      'auth_db_prev_rev': 5,
      'created_by': model.Identity(kind='user', name='a@example.com'),
      'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
      'description': u'Blah',
      'globs': [
        model.IdentityGlob(kind='user', pattern='*@a.com'),
        model.IdentityGlob(kind='user', pattern='*@b.com'),
      ],
      'members': [
        model.Identity(kind='user', name='a@example.com'),
        model.Identity(kind='user', name='b@example.com'),
      ],
      'modified_by': model.Identity(kind='user', name='a@example.com'),
      'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      'nested': [],
      'owners': u'administrators',
    }, model.group_key('B').get().to_dict())

    # Copies in the history.
    cpy = lambda name, rev: ndb.Key(
        'Rev', rev, 'AuthGroupHistory', name, parent=model.root_key())
    self.assertEqual({
      cpy('A', 1): {
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'',
        'globs': [],
        'members': [],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [],
        'owners': u'administrators',
      },
      cpy('A', 2): {
        'auth_db_rev': 2,
        'auth_db_prev_rev': 1,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'',
        'globs': [glob_a],
        'members': [ident_a],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [],
        'owners': u'administrators',
      },
      cpy('B', 3): {
        'auth_db_rev': 3,
        'auth_db_prev_rev': None,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'',
        'globs': [glob_b],
        'members': [ident_b],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [],
        'owners': u'administrators',
      },
      cpy('A', 4): {
        'auth_db_rev': 4,
        'auth_db_prev_rev': 2,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'',
        'globs': [glob_a],
        'members': [ident_a],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [u'B'],
        'owners': u'administrators',
      },
      # Batch revision.
      cpy('A', 5): {
        'auth_db_rev': 5,
        'auth_db_prev_rev': 4,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': True,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'',
        'globs': [glob_a],
        'members': [ident_a],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [u'B'],
        'owners': u'administrators',
      },
      cpy('B', 5): {
        'auth_db_rev': 5,
        'auth_db_prev_rev': 3,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'Blah',
        'globs': [glob_b],
        'members': [ident_b],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [],
        'owners': u'administrators',
      },
      # /end of batch revision
      cpy('B', 6): {
        'auth_db_rev': 6,
        'auth_db_prev_rev': 5,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'Blah',
        'globs': [glob_a, glob_b],
        'members': [ident_a, ident_b],
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'nested': [],
        'owners': u'administrators',
      },
    }, self.grab_log(model.AuthGroup))

  def test_ip_whitelist_log(self):
    @ndb.transactional
    def modify(name, **kwargs):
      k = model.ip_whitelist_key(name)
      e = k.get()
      if not e:
        e = model.AuthIPWhitelist(
            key=k,
            created_by=model.Identity.from_bytes('user:a@example.com'),
            created_ts=utils.utcnow())
      e.record_revision(
          modified_by=model.Identity.from_bytes('user:a@example.com'),
          modified_ts=utils.utcnow(),
          comment='Comment')
      e.populate(**kwargs)
      e.put()
      model.replicate_auth_db()

    @ndb.transactional
    def remove(name):
      e = model.ip_whitelist_key(name).get()
      if e:
        e.record_deletion(
            modified_by=model.Identity.from_bytes('user:a@example.com'),
            modified_ts=utils.utcnow(),
            comment='Comment')
        e.key.delete()
      model.replicate_auth_db()

    # Very similar to test_groups_log, so do less test cases.
    modify('A', subnets=['127.0.0.1/32'])
    modify('A', description='Blah')
    modify('A', subnets=['1.0.0.0/32'])
    remove('A')

    # Copies in the history.
    cpy = lambda name, rev: ndb.Key(
        'Rev', rev, 'AuthIPWhitelistHistory', name, parent=model.root_key())
    self.assertEqual({
      cpy('A', 1): {
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'',
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'subnets': [u'127.0.0.1/32'],
      },
      cpy('A', 2): {
        'auth_db_rev': 2,
        'auth_db_prev_rev': 1,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'Blah',
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'subnets': [u'127.0.0.1/32'],
      },
      cpy('A', 3): {
        'auth_db_rev': 3,
        'auth_db_prev_rev': 2,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'Blah',
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'subnets': [u'1.0.0.0/32'],
      },
      cpy('A', 4): {
        'auth_db_rev': 4,
        'auth_db_prev_rev': 3,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': True,
        'auth_db_change_comment': u'Comment',
        'created_by': model.Identity(kind='user', name='a@example.com'),
        'created_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'description': u'Blah',
        'modified_by': model.Identity(kind='user', name='a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
        'subnets': [u'1.0.0.0/32'],
      },
    }, self.grab_log(model.AuthIPWhitelist))

  def test_ip_whitelist_assignment_log(self):
    # AuthIPWhitelistAssignments is special, it has LocalStructuredProperty.

    @ndb.transactional
    def modify(assignments):
      key = model.ip_whitelist_assignments_key()
      e = key.get() or model.AuthIPWhitelistAssignments(key=key)
      e.record_revision(
          modified_by=model.Identity.from_bytes('user:a@example.com'),
          modified_ts=datetime.datetime(2015, 1, 1, 1, 1),
          comment='Comment')
      e.assignments = assignments
      e.put()
      model.replicate_auth_db()

    Assignment = model.AuthIPWhitelistAssignments.Assignment
    modify([])
    modify([
      Assignment(
          identity=model.Identity.from_bytes('user:a@example.com'),
          ip_whitelist='bots',
          comment='Blah'),
    ])
    modify([])

    cpy = lambda rev: ndb.Key(
        'Rev', rev, 'AuthIPWhitelistAssignmentsHistory', 'default',
        parent=model.root_key())
    self.assertEqual({
      cpy(1): {
        'assignments': [],
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
      cpy(2): {
        'assignments': [{
          'comment': u'Blah',
          'created_by': None,
          'created_ts': None,
          'identity': model.Identity(kind='user', name='a@example.com'),
          'ip_whitelist': u'bots',
        }],
        'auth_db_rev': 2,
        'auth_db_prev_rev': 1,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
      cpy(3): {
        'assignments': [],
        'auth_db_rev': 3,
        'auth_db_prev_rev': 2,
        'auth_db_app_version': u'v1a',
        'auth_db_deleted': False,
        'auth_db_change_comment': u'Comment',
        'modified_by': model.Identity.from_bytes('user:a@example.com'),
        'modified_ts': datetime.datetime(2015, 1, 1, 1, 1),
      },
    }, self.grab_log(model.AuthIPWhitelistAssignments))


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
