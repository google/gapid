#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import hashlib
import json
import logging
import os
import sys
import tempfile
import unittest

# net_utils adjusts sys.path.
import net_utils

import isolated_format
from depot_tools import auto_stub
from depot_tools import fix_encoding
from utils import file_path
from utils import fs
from utils import tools

import isolateserver_fake


ALGO = hashlib.sha1


class SymlinkTest(unittest.TestCase):
  def setUp(self):
    super(SymlinkTest, self).setUp()
    self.old_cwd = unicode(os.getcwd())
    self.cwd = tempfile.mkdtemp(prefix=u'isolate_')
    # Everything should work even from another directory.
    fs.chdir(self.cwd)

  def tearDown(self):
    try:
      fs.chdir(self.old_cwd)
      file_path.rmtree(self.cwd)
    finally:
      super(SymlinkTest, self).tearDown()

  if sys.platform == 'darwin':
    def test_expand_symlinks_path_case(self):
      # Ensures that the resulting path case is fixed on case insensitive file
      # system.
      fs.symlink('dest', os.path.join(self.cwd, u'link'))
      fs.mkdir(os.path.join(self.cwd, u'Dest'))
      fs.open(os.path.join(self.cwd, u'Dest', u'file.txt'), 'w').close()

      relfile, symlinks = isolated_format._expand_symlinks(self.cwd, u'.')
      self.assertEqual((u'.', []), (relfile, symlinks))

      relfile, symlinks = isolated_format._expand_symlinks(self.cwd, u'link')
      self.assertEqual((u'Dest', [u'link']), (relfile, symlinks))

      relfile, symlinks = isolated_format._expand_symlinks(
          self.cwd, u'link/File.txt')
      self.assertEqual((u'Dest/file.txt', [u'link']), (relfile, symlinks))

    def test_file_to_metadata_path_case_simple(self):
      # Ensure the symlink dest is saved in the right path case.
      subdir = os.path.join(self.cwd, u'subdir')
      fs.mkdir(subdir)
      linkdir = os.path.join(self.cwd, u'linkdir')
      fs.symlink('subDir', linkdir)
      actual = isolated_format.file_to_metadata(linkdir.upper(), True, False)
      self.assertEqual({'l': u'subdir'}, actual)

    def test_file_to_metadata_path_case_complex(self):
      # Ensure the symlink dest is saved in the right path case. This includes 2
      # layers of symlinks.
      basedir = os.path.join(self.cwd, u'basebir')
      fs.mkdir(basedir)

      linkeddir2 = os.path.join(self.cwd, u'linkeddir2')
      fs.mkdir(linkeddir2)

      linkeddir1 = os.path.join(basedir, u'linkeddir1')
      fs.symlink('../linkedDir2', linkeddir1)

      subsymlinkdir = os.path.join(basedir, u'symlinkdir')
      fs.symlink('linkedDir1', subsymlinkdir)

      actual = isolated_format.file_to_metadata(
          subsymlinkdir.upper(), True, False)
      self.assertEqual({'l': u'linkeddir1'}, actual)

      actual = isolated_format.file_to_metadata(
          linkeddir1.upper(), True, False)
      self.assertEqual({'l': u'../linkeddir2'}, actual)

  if sys.platform != 'win32':
    def test_symlink_input_absolute_path(self):
      # A symlink is outside of the checkout, it should be treated as a normal
      # directory.
      # .../src
      # .../src/out -> .../tmp/foo
      # .../tmp
      # .../tmp/foo
      src = os.path.join(self.cwd, u'src')
      src_out = os.path.join(src, u'out')
      tmp = os.path.join(self.cwd, u'tmp')
      tmp_foo = os.path.join(tmp, u'foo')
      fs.mkdir(src)
      fs.mkdir(tmp)
      fs.mkdir(tmp_foo)
      # The problem was that it's an absolute path, so it must be considered a
      # normal directory.
      fs.symlink(tmp, src_out)
      fs.open(os.path.join(tmp_foo, u'bar.txt'), 'w').close()
      relfile, symlinks = isolated_format._expand_symlinks(
          src, u'out/foo/bar.txt')
      self.assertEqual((u'out/foo/bar.txt', []), (relfile, symlinks))

    def test_file_to_metadata_path_case_collapse(self):
      # Ensure setting the collapse_symlink option doesn't include the symlinks
      basedir = os.path.join(self.cwd, u'basedir')
      fs.mkdir(basedir)
      subdir = os.path.join(basedir, u'subdir')
      fs.mkdir(subdir)
      linkdir = os.path.join(basedir, u'linkdir')
      fs.mkdir(linkdir)

      foo_file = os.path.join(subdir, u'Foo.txt')
      fs.open(foo_file, 'w').close()
      sym_file = os.path.join(basedir, u'linkdir', u'Sym.txt')
      fs.symlink('../subdir/Foo.txt', sym_file)

      actual = isolated_format.file_to_metadata(sym_file, True, True)
      actual['h'] = isolated_format.hash_file(sym_file, ALGO)
      expected = {
        # SHA-1 of empty string
        'h': 'da39a3ee5e6b4b0d3255bfef95601890afd80709',
        'm': 288,
        's': 0,
      }
      self.assertEqual(expected, actual)


class TestIsolated(auto_stub.TestCase):
  def test_load_isolated_empty(self):
    m = isolated_format.load_isolated('{}', isolateserver_fake.ALGO)
    self.assertEqual({}, m)

  def test_load_isolated_good(self):
    data = {
      u'command': [u'foo', u'bar'],
      u'files': {
        u'a': {
          u'l': u'somewhere',
        },
        u'b': {
          u'm': 123,
          u'h': u'0123456789abcdef0123456789abcdef01234567',
          u's': 3,
        }
      },
      u'includes': [u'0123456789abcdef0123456789abcdef01234567'],
      u'read_only': 1,
      u'relative_cwd': u'somewhere_else',
      u'version': isolated_format.ISOLATED_FILE_VERSION,
    }
    m = isolated_format.load_isolated(json.dumps(data), isolateserver_fake.ALGO)
    self.assertEqual(data, m)

  def test_load_isolated_bad(self):
    data = {
      u'files': {
        u'a': {
          u'l': u'somewhere',
          u'h': u'0123456789abcdef0123456789abcdef01234567'
        }
      },
      u'version': isolated_format.ISOLATED_FILE_VERSION,
    }
    with self.assertRaises(isolated_format.IsolatedError):
      isolated_format.load_isolated(json.dumps(data), isolateserver_fake.ALGO)

  def test_load_isolated_bad_abs(self):
    for i in ('/a', 'a/..', 'a/', '\\\\a'):
      data = {
        u'files': {i: {u'l': u'somewhere'}},
        u'version': isolated_format.ISOLATED_FILE_VERSION,
      }
      with self.assertRaises(isolated_format.IsolatedError):
        isolated_format.load_isolated(json.dumps(data), isolateserver_fake.ALGO)

  def test_load_isolated_os_only(self):
    # Tolerate 'os' on older version.
    data = {
      u'os': 'HP/UX',
      u'version': '1.3',
    }
    m = isolated_format.load_isolated(json.dumps(data), isolateserver_fake.ALGO)
    self.assertEqual(data, m)

  def test_load_isolated_os_only_bad(self):
    data = {
      u'os': 'HP/UX',
      u'version': isolated_format.ISOLATED_FILE_VERSION,
    }
    with self.assertRaises(isolated_format.IsolatedError):
      isolated_format.load_isolated(json.dumps(data), isolateserver_fake.ALGO)

  def test_load_isolated_path(self):
    # Automatically convert the path case.
    wrong_path_sep = u'\\' if os.path.sep == '/' else u'/'
    def gen_data(path_sep):
      return {
        u'command': [u'foo', u'bar'],
        u'files': {
          path_sep.join(('a', 'b')): {
            u'l': path_sep.join(('..', 'somewhere')),
          },
        },
        u'relative_cwd': path_sep.join(('somewhere', 'else')),
        u'version': isolated_format.ISOLATED_FILE_VERSION,
      }

    data = gen_data(wrong_path_sep)
    actual = isolated_format.load_isolated(
        json.dumps(data), isolateserver_fake.ALGO)
    expected = gen_data(os.path.sep)
    self.assertEqual(expected, actual)

  def test_save_isolated_good_long_size(self):
    calls = []
    self.mock(tools, 'write_json', lambda *x: calls.append(x))
    data = {
      u'algo': 'sha-1',
      u'files': {
        u'b': {
          u'm': 123,
          u'h': u'0123456789abcdef0123456789abcdef01234567',
          u's': 2181582786L,
        }
      },
    }
    isolated_format.save_isolated('foo', data)
    self.assertEqual([('foo', data, True)], calls)


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=(logging.DEBUG if '-v' in sys.argv else logging.ERROR))
  unittest.main()
