#!/usr/bin/env python
# coding=utf-8
# Copyright 2019 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import logging
import os
import tempfile
import unittest
import sys

FILE_PATH = os.path.abspath(__file__.decode(sys.getfilesystemencoding()))
ROOT_DIR = os.path.dirname(os.path.dirname(FILE_PATH))
sys.path.insert(0, ROOT_DIR)
sys.path.insert(0, os.path.join(ROOT_DIR, 'third_party'))


from depot_tools import fix_encoding
from utils import file_path
from utils import fs


def write_content(path, content):
  with fs.open(path, 'wb') as f:
    f.write(content)


class FSTest(unittest.TestCase):
  @classmethod
  def setUpClass(cls):
    if not file_path.enable_symlink():
      raise Exception('Failed to enable symlink support')

  def setUp(self):
    super(FSTest, self).setUp()
    self._tempdir = None

  def tearDown(self):
    try:
      if self._tempdir:
        file_path.rmtree(self._tempdir)
    finally:
      super(FSTest, self).tearDown()

  @property
  def tempdir(self):
    if not self._tempdir:
      self._tempdir = tempfile.mkdtemp(prefix=u'fs_test')
    return self._tempdir

  def test_symlink_relative(self):
    # A symlink to a relative path is valid.
    # /dir
    # /dir/file
    # /ld -> /dir
    # /lf -> /ld/file
    dirpath = os.path.join(self.tempdir, 'dir')
    filepath = os.path.join(dirpath, 'file')
    fs.mkdir(dirpath)
    write_content(filepath, 'hello')

    linkfile = os.path.join(self.tempdir, 'lf')
    linkdir = os.path.join(self.tempdir, 'ld')
    dstfile = os.path.join('ld', 'file')
    fs.symlink(dstfile, linkfile)
    fs.symlink('dir', linkdir)

    self.assertEqual(True, fs.islink(linkfile))
    self.assertEqual(True, fs.islink(linkdir))
    self.assertEqual(dstfile, fs.readlink(linkfile))
    self.assertEqual('dir', fs.readlink(linkdir))
    self.assertEqual(['file'], fs.listdir(linkdir))
    # /lf resolves to /dir/file.
    self.assertEqual('hello', fs.open(linkfile).read())

    # Ensures that followlinks is respected in walk().
    expected = [
      (self.tempdir, ['dir', 'ld'], ['lf']),
      (dirpath, [], ['file']),
    ]
    actual = [
      (r, sorted(d), sorted(f))
      for r, d, f in sorted(fs.walk(self.tempdir, followlinks=False))
    ]
    self.assertEqual(expected, actual)
    expected = [
      (self.tempdir, ['dir', 'ld'], ['lf']),
      (dirpath, [], ['file']),
      (linkdir, [], ['file']),
    ]
    actual = [
      (r, sorted(d), sorted(f))
      for r, d, f in sorted(fs.walk(self.tempdir, followlinks=True))
    ]
    self.assertEqual(expected, actual)

  def test_symlink_absolute(self):
    # A symlink to an absolute path is valid.
    # /dir
    # /dir/file
    # /ld -> /dir
    # /lf -> /ld/file
    dirpath = os.path.join(self.tempdir, 'dir')
    filepath = os.path.join(dirpath, 'file')
    fs.mkdir(dirpath)
    write_content(filepath, 'hello')

    linkfile = os.path.join(self.tempdir, 'lf')
    linkdir = os.path.join(self.tempdir, 'ld')
    dstfile = os.path.join(linkdir, 'file')
    fs.symlink(dstfile, linkfile)
    fs.symlink(dirpath, linkdir)

    self.assertEqual(True, fs.islink(linkfile))
    self.assertEqual(True, fs.islink(linkdir))
    self.assertEqual(dstfile, fs.readlink(linkfile))
    self.assertEqual(dirpath, fs.readlink(linkdir))
    self.assertEqual(['file'], fs.listdir(linkdir))
    # /lf resolves to /dir/file.
    self.assertEqual('hello', fs.open(linkfile).read())

    # Ensures that followlinks is respected in walk().
    expected = [
      (self.tempdir, ['dir', 'ld'], ['lf']),
      (dirpath, [], ['file']),
    ]
    actual = [
      (r, sorted(d), sorted(f))
      for r, d, f in sorted(fs.walk(self.tempdir, followlinks=False))
    ]
    self.assertEqual(expected, actual)
    expected = [
      (self.tempdir, ['dir', 'ld'], ['lf']),
      (dirpath, [], ['file']),
      (linkdir, [], ['file']),
    ]
    actual = [
      (r, sorted(d), sorted(f))
      for r, d, f in sorted(fs.walk(self.tempdir, followlinks=True))
    ]
    self.assertEqual(expected, actual)

  def test_symlink_missing_destination_rel(self):
    # A symlink to a missing destination is valid and can be read back.
    filepath = 'file'
    linkfile = os.path.join(self.tempdir, 'lf')
    fs.symlink(filepath, linkfile)

    self.assertEqual(True, fs.islink(linkfile))
    self.assertEqual(filepath, fs.readlink(linkfile))

  def test_symlink_missing_destination_abs(self):
    # A symlink to a missing destination is valid and can be read back.
    filepath = os.path.join(self.tempdir, 'file')
    linkfile = os.path.join(self.tempdir, 'lf')
    fs.symlink(filepath, linkfile)

    self.assertEqual(True, fs.islink(linkfile))
    self.assertEqual(filepath, fs.readlink(linkfile))

  def test_symlink_existing(self):
    # Creating a symlink that overrides a file fails.
    filepath = os.path.join(self.tempdir, 'file')
    linkfile = os.path.join(self.tempdir, 'lf')
    write_content(linkfile, 'hello')
    with self.assertRaises(OSError):
      fs.symlink(filepath, linkfile)

  def test_readlink_fail(self):
    # Reading a non-existing symlink fails. Obvious but it's to make sure the
    # Windows part acts the same.
    with self.assertRaises(OSError):
      fs.readlink(os.path.join(self.tempdir, 'not_there'))


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  logging.basicConfig(
      level=logging.DEBUG if '-v' in sys.argv else logging.ERROR)
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
