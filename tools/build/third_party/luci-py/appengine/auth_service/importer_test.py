#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import datetime
import logging
import os
import StringIO
import sys
import tarfile
import tempfile
import unittest

APP_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

import test_env
test_env.setup_test_env()

from google.appengine.ext import ndb

from components import auth
from components import auth_testing
from components.auth import model
from test_support import test_case

from proto import config_pb2
import importer


def build_tar_gz(content):
  """Returns bytes of tar.gz archive build from {filename -> body} dict."""
  out = StringIO.StringIO()
  with tarfile.open(mode='w|gz', fileobj=out) as tar:
    for name, value in content.iteritems():
      # tarfile module doesn't support in-memory files (it tries to os.stat
      # them), so dump to disk first to keep the code simple.
      path = None
      try:
        fd, path = tempfile.mkstemp(prefix='importer_test')
        with os.fdopen(fd, 'w') as f:
          f.write(value)
        tar.add(path, arcname=name)
      finally:
        if path:
          os.remove(path)
  return out.getvalue()


def ident(name):
  if '@' not in name:
    return auth.Identity(auth.IDENTITY_USER, '%s@example.com' % name)
  else:
    return auth.Identity(auth.IDENTITY_USER, name)


def group(name, members, nested=None):
  return model.AuthGroup(
      key=model.group_key(name),
      created_by=ident('admin'),
      created_ts=datetime.datetime(1999, 1, 2, 3, 4, 5, 6),
      modified_by=ident('admin'),
      modified_ts=datetime.datetime(1999, 1, 2, 3, 4, 5, 6),
      members=[ident(x) for x in members],
      nested=nested or [])


def fetch_groups():
  return {x.key.id(): x.to_dict() for x in model.AuthGroup.query()}


def put_config(config_proto):
  importer.GroupImporterConfig(
      key=importer.config_key(),
      config_proto=config_proto).put()


def put_and_load_config_err(config_proto):
  try:
    put_config(config_proto)
    importer.load_config()
    return None
  except importer.BundleImportError as exc:
    return str(exc)


class ImporterTest(test_case.TestCase):
  def setUp(self):
    super(ImporterTest, self).setUp()
    auth_testing.mock_is_admin(self, True)
    auth_testing.mock_get_current_identity(self)

  def mock_urlfetch(self, urls):
    @ndb.tasklet
    def mock_get_access_token_async(*_args):
      raise ndb.Return(('token', 0))
    self.mock(auth, 'get_access_token_async', mock_get_access_token_async)

    @ndb.tasklet
    def mock_fetch(**kwargs):
      self.assertIn(kwargs['url'], urls)
      self.assertEqual({'Authorization': 'Bearer token'}, kwargs['headers'])
      class ReturnValue(object):
        status_code = 200
        content = urls[kwargs['url']]
      raise ndb.Return(ReturnValue())
    self.mock(ndb.get_context(), 'urlfetch', mock_fetch)

  def test_extract_tar_archive(self):
    expected = {
      '0': '0',
      'a/1': '1',
      'a/2': '2',
      'b/1': '3',
      'b/c/d': '4',
    }
    out = {
      name: fileobj.read()
      for name, fileobj in importer.extract_tar_archive(build_tar_gz(expected))
    }
    self.assertEqual(expected, out)

  def test_load_group_file_ok(self):
    body = '\n'.join(['', 'b', 'a', 'a', ''])
    expected = [
      auth.Identity.from_bytes('user:a@example.com'),
      auth.Identity.from_bytes('user:b@example.com'),
    ]
    self.assertEqual(expected, importer.load_group_file(body, 'example.com'))

  def test_load_group_file_gtempaccount(self):
    self.assertEqual(
        [auth.Identity.from_bytes('user:blah@domain.org')],
        importer.load_group_file(r'blah%domain.org@gtempaccount.com', None))

  def test_load_group_file_bad_id(self):
    body = 'bad id'
    with self.assertRaises(importer.BundleBadFormatError):
      importer.load_group_file(body, 'example.com')

  def test_prepare_import(self):
    existing_groups = [
      group('normal-group', [], ['ldap/cleared']),
      group('not-ldap/some', []),
      group('ldap/updated', ['a']),
      group('ldap/unchanged', ['a']),
      group('ldap/deleted', ['a']),
      group('ldap/cleared', ['a']),
    ]
    imported_groups = {
      'ldap/new': [ident('a')],
      'ldap/updated': [ident('a'), ident('b')],
      'ldap/unchanged': [ident('a')],
    }
    to_put, to_delete = importer.prepare_import(
        'ldap',
        existing_groups,
        imported_groups,
        datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        model.get_service_self_identity())

    expected_to_put = {
      'ldap/cleared': {
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'created_by': ident('admin'),
        'created_ts': datetime.datetime(1999, 1, 2, 3, 4, 5, 6),
        'description': '',
        'globs': [],
        'members': [],
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
      'ldap/new': {
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'created_by': model.get_service_self_identity(),
        'created_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'description': '',
        'globs': [],
        'members': [ident('a')],
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
      'ldap/updated': {
        'auth_db_rev': None,
        'auth_db_prev_rev': None,
        'created_by': ident('admin'),
        'created_ts': datetime.datetime(1999, 1, 2, 3, 4, 5, 6),
        'description': '',
        'globs': [],
        'members': [ident('a'), ident('b')],
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
    }
    self.assertEqual(expected_to_put, {x.key.id(): x.to_dict() for x in to_put})
    self.assertEqual(
        [model.group_key('ldap/deleted')], [x.key for x in to_delete])

  def test_load_tarball(self):
    bundle = build_tar_gz({
      'at_root': 'a\nb',
      'ldap/ bad name': 'a\nb',
      'ldap/group-a': 'a\nb',
      'ldap/group-b': 'a\nb',
      'ldap/group-c': 'a\nb',
      'ldap/deeper/group-a': 'a\nb',
      'not-ldap/group-a': 'a\nb',
    })
    result = importer.load_tarball(
        content=bundle,
        systems=['ldap'],
        groups=['ldap/group-a', 'ldap/group-b'],
        domain='example.com')

    expected = {
      'ldap': {
        'ldap/group-a': [
          auth.Identity.from_bytes('user:a@example.com'),
          auth.Identity.from_bytes('user:b@example.com')
        ],
        'ldap/group-b': [
          auth.Identity.from_bytes('user:a@example.com'),
          auth.Identity.from_bytes('user:b@example.com')
        ],
      }
    }
    self.assertEqual(expected, result)

  def test_load_tarball_bad_group(self):
    bundle = build_tar_gz({
      'at_root': 'a\nb',
      'ldap/group-a': 'a\n!!!!!',
    })
    with self.assertRaises(importer.BundleBadFormatError):
      importer.load_tarball(
        content=bundle,
        systems=['ldap'],
        groups=['ldap/group-a', 'ldap/group-b'],
        domain='example.com')

  def test_import_external_groups(self):
    self.mock_now(datetime.datetime(2010, 1, 2, 3, 4, 5, 6))

    importer.write_config("""
      tarball {
        domain: "example.com"
        groups: "ldap/new"
        oauth_scopes: "scope"
        systems: "ldap"
        url: "https://fake_tarball"
      }
      tarball_upload {
        name: "should be ignored"
        authorized_uploader: "zzz@example.com"
        systems: "zzz"
      }
      plainlist {
        group: "external_1"
        oauth_scopes: "scope"
        url: "https://fake_external_1"
      }
      plainlist {
        domain: "example.com"
        group: "external_2"
        oauth_scopes: "scope"
        url: "https://fake_external_2"
      }
    """)

    self.mock_urlfetch({
      'https://fake_tarball': build_tar_gz({
        'ldap/new': 'a\nb',
      }),
      'https://fake_external_1': 'abc@test.com\ndef@test.com\nabc@test.com',
      'https://fake_external_2': '123\n456',
    })

    # Should be deleted during import, since not in a imported bundle.
    group('ldap/deleted', []).put()
    # Should be updated.
    group('external/external_1', ['x', 'y']).put()
    # Should be removed, since not in list of external groups.
    group('external/deleted', []).put()

    # Run the import.
    initial_auth_db_rev = model.get_auth_db_revision()
    importer.import_external_groups()
    self.assertEqual(initial_auth_db_rev + 1, model.get_auth_db_revision())

    # Verify final state.
    expected_groups = {
      'ldap/new': {
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'created_by': model.get_service_self_identity(),
        'created_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'description': u'',
        'globs': [],
        'members': [ident('a'), ident('b')],
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
      'external/external_1': {
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'created_by': ident('admin'),
        'created_ts': datetime.datetime(1999, 1, 2, 3, 4, 5, 6),
        'description': u'',
        'globs': [],
        'members': [ident('abc@test.com'), ident('def@test.com')],
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
      'external/external_2': {
        'auth_db_rev': 1,
        'auth_db_prev_rev': None,
        'created_by': model.get_service_self_identity(),
        'created_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'description': u'',
        'globs': [],
        'members': [ident('123'), ident('456')],
        'modified_by': model.get_service_self_identity(),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
    }
    self.assertEqual(expected_groups, fetch_groups())

  def test_read_config(self):
    # Empty.
    put_config('')
    self.assertEqual('', importer.read_config())
    # Good.
    put_config('tarball{}')
    self.assertEqual('tarball{}', importer.read_config())

  def test_write_config(self):
    put_config('')
    importer.write_config('tarball{url:"12"\nsystems:"12"}')
    e = importer.config_key().get()
    self.assertEqual('tarball{url:"12"\nsystems:"12"}', e.config_proto)

  def test_load_config_happy(self):
    self.assertIsNone(importer.load_config())

    put_config("""
      tarball {
        url: "http://example.com/tarball"
        oauth_scopes: "scope1"
        oauth_scopes: "scope2"
        domain: "zzz1.example.com"
        systems: "s1"
        groups: "s1/g1"
        groups: "s1/g2"
      }

      tarball_upload {
        name: "tarball upload"
        authorized_uploader: "abc@example.com"
        authorized_uploader: "def@example.com"
        domain: "zzz2.example.com"
        systems: "s2"
        groups: "s2/g1"
        groups: "s2/g2"
      }

      plainlist {
        url: "http://example.com/plainlist"
        oauth_scopes: "scope1"
        oauth_scopes: "scope2"
        domain: "zzz3.example.com"
        group: "g3"
      }
    """)

    cfg = importer.load_config()
    self.assertEqual(config_pb2.GroupImporterConfig(
      tarball=[config_pb2.GroupImporterConfig.TarballEntry(
        url='http://example.com/tarball',
        oauth_scopes=['scope1', 'scope2'],
        domain='zzz1.example.com',
        systems=['s1'],
        groups=['s1/g1', 's1/g2'],
      )],
      tarball_upload=[config_pb2.GroupImporterConfig.TarballUploadEntry(
        name='tarball upload',
        authorized_uploader=['abc@example.com', 'def@example.com'],
        domain='zzz2.example.com',
        systems=['s2'],
        groups=['s2/g1', 's2/g2'],
      )],
      plainlist=[config_pb2.GroupImporterConfig.PlainlistEntry(
        url='http://example.com/plainlist',
        oauth_scopes=['scope1', 'scope2'],
        domain='zzz3.example.com',
        group='g3',
      )]
    ), cfg)

  def test_load_config_no_urls(self):
    self.assertEqual(
        'Bad config structure: "url" field is required in TarballEntry',
        put_and_load_config_err("""
        tarball {
          systems: "s1"
        }
        """))

    self.assertEqual(
        'Bad config structure: "url" field is required in PlainlistEntry',
        put_and_load_config_err("""
        plainlist {
          group: "g3"
        }
        """))

  def test_load_config_dup_upload_name(self):
    self.assertEqual(
        'Bad config structure: tarball_upload entry "ball" is specified twice',
        put_and_load_config_err("""
        tarball_upload {
          name: "ball"
          authorized_uploader: "abc@example.com"
          systems: "s"
        }
        tarball_upload {
          name: "ball"
          authorized_uploader: "abc@example.com"
          systems: "s"
        }
        """))

  def test_load_config_bad_authorized_uploader(self):
    self.assertEqual(
        'Bad config structure: authorized_uploader is required in '
            'tarball_upload entry "ball"',
        put_and_load_config_err("""
        tarball_upload {
          name: "ball"
          systems: "s"
        }
        """))

    self.assertEqual(
        'Bad config structure: invalid email "not an email" in '
            'tarball_upload entry "ball"',
        put_and_load_config_err("""
        tarball_upload {
          name: "ball"
          authorized_uploader: "not an email"
          systems: "s"
        }
        """))

  def test_load_config_bad_systems(self):
    self.assertEqual(
        'Bad config structure: "tarball" entry with URL '
            '"http://example.com/tarball" needs "systems" field',
        put_and_load_config_err("""
        tarball {
          url: "http://example.com/tarball"
        }
        """))

    self.assertEqual(
        'Bad config structure: "tarball_upload" entry with name "ball" '
            'needs "systems" field',
        put_and_load_config_err("""
        tarball_upload {
          name: "ball"
          authorized_uploader: "abc@example.com"
        }
        """))

    self.assertEqual(
        'Bad config structure: "tarball_upload" entry with name "conflicting" '
            'is specifying a duplicate system(s): '
            '[u\'external\', u\'s1\', u\'s3\', u\'s5\']',
        put_and_load_config_err("""
        tarball {
          url: "http://example.com/tarball1"
          systems: "s1"
          systems: "s2"
        }

        tarball {
          url: "http://example.com/tarball2"
          systems: "s3"
          systems: "s4"
        }

        tarball_upload {
          name: "tarball3"
          authorized_uploader: "abc@example.com"
          systems: "s5"
          systems: "s6"
        }

        tarball_upload {
          name: "conflicting"
          authorized_uploader: "abc@example.com"
          systems: "external" # this one is predefined
          systems: "s1"
          systems: "s3"
          systems: "s5"
          systems: "ok"
        }
        """))

  def test_load_config_bad_plainlists(self):
    self.assertEqual(
        'Bad config structure: "plainlist" entry "http://example.com/plainlist"'
            ' needs "group" field',
        put_and_load_config_err("""
        plainlist {
          url: "http://example.com/plainlist"
        }
        """))

    self.assertEqual(
        'Bad config structure: The group "gr" imported twice',
        put_and_load_config_err("""
        plainlist {
          url: "http://example.com/plainlist1"
          group: "gr"
        }
        plainlist {
          url: "http://example.com/plainlist2"
          group: "gr"
        }
        """))

  def test_ingest_tarball_happy(self):
    self.mock_now(datetime.datetime(2010, 1, 2, 3, 4, 5, 6))

    put_config("""
      tarball_upload {
        name: "tarball.tar.gz"
        authorized_uploader: "mocked@example.com"
        domain: "zzz.example.com"
        systems: "ldap"
        groups: "ldap/ok1"
        groups: "ldap/ok2"
      }
    """)

    tarball = build_tar_gz({
      'ldap/ok1': 'a',
      'ldap/ok2': 'b',
      'ldap/ignored': '1',
      'ignored/zzz': '2',
    })

    groups, rev = importer.ingest_tarball('tarball.tar.gz', tarball)
    self.assertEqual(['ldap/ok1', 'ldap/ok2'], groups)
    self.assertEqual(1, rev)

    expected_groups = {
      'ldap/ok1': {
        'auth_db_prev_rev': None,
        'auth_db_rev': 1,
        'created_by': auth.Identity(kind='user', name='mocked@example.com'),
        'created_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'description': u'',
        'globs': [],
        'members': [auth.Identity(kind='user', name='a@zzz.example.com')],
        'modified_by': auth.Identity(kind='user', name='mocked@example.com'),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
      'ldap/ok2': {
        'auth_db_prev_rev': None,
        'auth_db_rev': 1,
        'created_by': auth.Identity(kind='user', name='mocked@example.com'),
        'created_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'description': u'',
        'globs': [],
        'members': [auth.Identity(kind='user', name='b@zzz.example.com')],
        'modified_by': auth.Identity(kind='user', name='mocked@example.com'),
        'modified_ts': datetime.datetime(2010, 1, 2, 3, 4, 5, 6),
        'nested': [],
        'owners': u'administrators',
      },
    }
    self.assertEqual(expected_groups, fetch_groups())

    # Same tarball again => noop.
    groups, rev = importer.ingest_tarball('tarball.tar.gz', tarball)
    self.assertEqual([], groups)
    self.assertEqual(0, rev)

    # Empty tarball => removes the groups, they are no longer exported.
    tarball  = build_tar_gz({})
    groups, rev = importer.ingest_tarball('tarball.tar.gz', tarball)
    self.assertEqual(['ldap/ok1', 'ldap/ok2'], groups)
    self.assertEqual(2, rev)

    # All are deleted.
    self.assertEqual({}, fetch_groups())

  def test_ingest_tarball_not_configured(self):
    with self.assertRaises(auth.AuthorizationError):
      importer.ingest_tarball('zzz', '')

  def test_ingest_tarball_unknown_tarball(self):
    put_config("""
      tarball_upload {
        name: "tarball.tar.gz"
        authorized_uploader: "mocked@example.com"
        systems: "ldap"
      }
    """)
    with self.assertRaises(auth.AuthorizationError):
      importer.ingest_tarball('unknown.tar.gz', '')

  def test_ingest_tarball_unauthorized(self):
    put_config("""
      tarball_upload {
        name: "tarball.tar.gz"
        authorized_uploader: "someone-else@example.com"
        systems: "ldap"
      }
    """)
    with self.assertRaises(auth.AuthorizationError):
      importer.ingest_tarball('tarball.tar.gz', '')

  def test_ingest_tarball_bad_tarball(self):
    put_config("""
      tarball_upload {
        name: "tarball.tar.gz"
        authorized_uploader: "mocked@example.com"
        systems: "ldap"
      }
    """)
    with self.assertRaises(importer.BundleImportError):
      importer.ingest_tarball('tarball.tar.gz', 'zzzzzzz')


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
    logging.basicConfig(level=logging.DEBUG)
  else:
    logging.basicConfig(level=logging.FATAL)
  unittest.main()
