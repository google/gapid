#!/usr/bin/env python
# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import hashlib
import json
import logging
import os
import random
import string
import sys
import tempfile
import time
import unittest

TEST_DIR = os.path.dirname(os.path.abspath(
    __file__.decode(sys.getfilesystemencoding())))
ROOT_DIR = os.path.dirname(TEST_DIR)
sys.path.insert(0, ROOT_DIR)
sys.path.insert(0, os.path.join(ROOT_DIR, 'third_party'))

from depot_tools import auto_stub
from depot_tools import fix_encoding

from utils import file_path
from utils import fs
from utils import lru

import local_caching


def write_file(path, contents):
  with fs.open(path, 'wb') as f:
    f.write(contents)


def read_file(path):
  with fs.open(path, 'rb') as f:
    return f.read()


def read_tree(path):
  """Returns a dict with {filepath: content}."""
  if not fs.isdir(path):
    return None
  out = {}
  for root, _, filenames in fs.walk(path):
    for filename in filenames:
      p = os.path.join(root, filename)
      out[os.path.relpath(p, path)] = read_file(p)
  return out


def _gen_data(size):
  return (string.digits*((size+9)/10))[:size]


class TestCase(auto_stub.TestCase):
  def setUp(self):
    super(TestCase, self).setUp()
    self.tempdir = tempfile.mkdtemp(prefix=u'local_caching')
    self._algo = hashlib.sha1
    # Time mocking.
    self._now = 1000
    self.mock(lru.LRUDict, 'time_fn', lambda _: self._now)

    # Free disk space mocking.
    self._free_disk = 1000
    self.mock(file_path, 'get_free_space', lambda _: self._free_disk)

    # Named cache works with directories.
    def rmtree(p):
      self._free_disk += local_caching._get_recursive_size(p)
      return old_rmtree(p)
    old_rmtree = self.mock(file_path, 'rmtree', rmtree)

    # Isolated cache works with files.
    def try_remove(p):
      try:
        self._free_disk += fs.stat(p).st_size
      except OSError:
        pass
      return old_try_remove(p)
    old_try_remove = self.mock(file_path, 'try_remove', try_remove)

  def tearDown(self):
    try:
      file_path.rmtree(self.tempdir)
    finally:
      super(TestCase, self).tearDown()

  def _add_one_item(self, cache, size):
    """Adds one item of |size| bytes in the cache and returns the created name.
    """
    # Don't allow 0 byte items here. This doesn't work for named cache.
    self.assertTrue(size)
    data = _gen_data(size)
    if isinstance(cache, local_caching.ContentAddressedCache):
      # This covers both MemoryContentAddressedCache and
      # DiskContentAddressedCache.
      return cache.write(self._algo(data).hexdigest(), [data])
    elif isinstance(cache, local_caching.NamedCache):
      # In this case, map a named cache, add a file, unmap it.
      dest_dir = os.path.join(self.tempdir, 'dest')
      self.assertFalse(fs.exists(dest_dir))
      name = unicode(size)
      cache.install(dest_dir, name)
      # Put a file in there named 'hello', otherwise it'll stay empty.
      with fs.open(os.path.join(dest_dir, 'hello'), 'wb') as f:
        f.write(data)
      cache.uninstall(dest_dir, name)
      self.assertFalse(fs.exists(dest_dir))
      return name
    else:
      self.fail('Unexpected cache type %r' % cache)


def _get_policies(
    max_cache_size=0, min_free_space=0, max_items=0, max_age_secs=0):
  """Returns a CachePolicies with only the policy we want to enforce."""
  return local_caching.CachePolicies(
      max_cache_size=max_cache_size,
      min_free_space=min_free_space,
      max_items=max_items,
      max_age_secs=max_age_secs)


class CacheTestMixin(object):
  """Adds testing for the Cache interface."""
  def get_cache(self, policies):
    raise NotImplementedError()

  def test_contains(self):
    cache = self.get_cache(_get_policies())
    self.assertFalse(u'foo' in cache)
    name = self._add_one_item(cache, 1)
    self.assertFalse(u'foo' in cache)
    self.assertTrue(name in cache)

  def test_iter(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual([], list(cache))
    n10 = self._add_one_item(cache, 10)
    n3 = self._add_one_item(cache, 3)
    self.assertEqual([n10, n3], list(cache))

  def test_len(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual(0, len(cache))
    self._add_one_item(cache, 10)
    self.assertEqual(1, len(cache))
    self._add_one_item(cache, 11)
    self.assertEqual(2, len(cache))

  def test_total_size(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual(0, cache.total_size)
    self._add_one_item(cache, 10)
    self.assertEqual(10, cache.total_size)
    self._add_one_item(cache, 11)
    self.assertEqual(21, cache.total_size)

  def test_added(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual([], cache.added)
    self._add_one_item(cache, 10)
    self.assertEqual([10], cache.added)

  def test_used(self):
    # It depends on the implementation.
    pass

  def test_get_oldest(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual(None, cache.get_oldest())
    ts = self._now
    self._add_one_item(cache, 10)
    self._now += 10
    self._add_one_item(cache, 20)
    self.assertEqual(ts, cache.get_oldest())

  def test_remove_oldest(self):
    cache = self.get_cache(_get_policies())
    self._add_one_item(cache, 10)
    self._add_one_item(cache, 100)
    self.assertTrue(cache.remove_oldest())
    self._add_one_item(cache, 20)
    # added is not yet updated.
    self.assertEqual([10, 100, 20], cache.added)

  def test_save(self):
    cache = self.get_cache(_get_policies())
    self._add_one_item(cache, 100)
    self._add_one_item(cache, 101)
    # Just assert it doesn't crash here.
    cache.save()

  def test_trim(self):
    cache = self.get_cache(_get_policies())
    self._add_one_item(cache, 100)
    self._add_one_item(cache, 101)
    self.assertEqual([], cache.trim())

  def test_true(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual(0, len(cache))
    # Even if empty, it's still "True".
    self.assertTrue(bool(cache))


class ContentAddressedCacheTestMixin(CacheTestMixin):
  """Add testing for the ContentAddressedCache interface."""
  # pylint: disable=abstract-method
  _now = 0

  # write() is indirectly tested with _add_one_item().
  # cleanup() requires specific setup.

  def test_touch(self):
    cache = self.get_cache(_get_policies())
    self._now = 1000
    n1 = self._add_one_item(cache, 1)
    self._now = 1001
    n2 = self._add_one_item(cache, 2)
    self.assertEqual([n1, n2], list(cache))
    self.assertEqual(1000, cache.get_oldest())
    self._now = 1002
    cache.touch(n1, None)
    self.assertEqual([n2, n1], list(cache))
    self.assertEqual(1001, cache.get_oldest())

  def test_getfileobj(self):
    cache = self.get_cache(_get_policies())
    h = self._add_one_item(cache, 1)
    with cache.getfileobj(h) as f:
      self.assertEqual('0', f.read())

  def test_getfileobj_complete_miss(self):
    cache = self.get_cache(_get_policies())
    with self.assertRaises(local_caching.CacheMiss):
      cache.getfileobj('0'*40)

  def test_getfileobj_cache_state_missing(self):
    # Put the file in the cache, but do NOT save cache state.
    cache = self.get_cache(_get_policies())
    h = self._add_one_item(cache, 1)
    # Since we didn't save the state, this should result in CacheMiss.
    cache = self.get_cache(_get_policies())
    with self.assertRaises(local_caching.CacheMiss):
      cache.getfileobj(h)


class MemoryContentAddressedCacheTest(TestCase, ContentAddressedCacheTestMixin):
  def get_cache(self, policies):
    return local_caching.MemoryContentAddressedCache(policies)

  def cleanup(self):
    # Doesn't do anything.
    cache = self.get_cache(_get_policies())
    cache.cleanup()


class DiskContentAddressedCacheTest(TestCase, ContentAddressedCacheTestMixin):
  def setUp(self):
    super(DiskContentAddressedCacheTest, self).setUp()
    # If this fails on Windows, please rerun this tests as an elevated user with
    # administrator access right.
    self.assertEqual(True, file_path.enable_symlink())

  def get_cache(self, policies):
    return local_caching.DiskContentAddressedCache(
        os.path.join(self.tempdir, 'cache'), policies, trim=True)

  def test_write_policies_free_disk(self):
    cache = self.get_cache(_get_policies(min_free_space=1000))
    with self.assertRaises(local_caching.NoMoreSpace):
      self._add_one_item(cache, 1)

  def test_write_policies_fit(self):
    self._free_disk = 1001
    cache = self.get_cache(_get_policies(min_free_space=1000))
    with self.assertRaises(local_caching.NoMoreSpace):
      self._add_one_item(cache, 2)

  def test_write_policies_max_cache_size(self):
    # max_cache_size is ignored while adding items.
    cache = self.get_cache(_get_policies(max_cache_size=1))
    self._add_one_item(cache, 2)
    self._add_one_item(cache, 3)

  def test_write_policies_max_items(self):
    # max_items is ignored while adding items.
    cache = self.get_cache(_get_policies(max_items=1))
    self._add_one_item(cache, 2)
    self._add_one_item(cache, 3)

  def test_write_policies_min_free_space(self):
    # min_free_space is enforced while adding items.
    self._free_disk = 1005
    cache = self.get_cache(_get_policies(min_free_space=1000))
    self._add_one_item(cache, 2)
    self._add_one_item(cache, 3)
    # Mapping more content than the amount of free disk required.
    with self.assertRaises(local_caching.NoMoreSpace) as cm:
      self._add_one_item(cache, 1)
    expected = (
        'Not enough space to fetch the whole isolated tree.\n'
        '  CachePolicies(max_cache_size=0; max_items=0; min_free_space=1000; '
          'max_age_secs=0)\n'
        '  cache=8bytes, 4 items; 999b free_space')
    self.assertEqual(expected, cm.exception.message)

  def test_save_disk(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual(
        sorted([cache.STATE_FILE]), sorted(fs.listdir(cache.cache_dir)))

    h = self._add_one_item(cache, 2)
    self.assertEqual(
        sorted([h, cache.STATE_FILE]), sorted(fs.listdir(cache.cache_dir)))
    items = lru.LRUDict.load(os.path.join(cache.cache_dir, cache.STATE_FILE))
    self.assertEqual(0, len(items))

    cache.save()
    self.assertEqual(
        sorted([h, cache.STATE_FILE]), sorted(fs.listdir(cache.cache_dir)))
    items = lru.LRUDict.load(os.path.join(cache.cache_dir, cache.STATE_FILE))
    self.assertEqual(1, len(items))
    self.assertEqual((h, [2, 1000]), items.get_oldest())

  def test_cleanup_disk(self):
    # Inject an item without a state.json, one is lost. Both will be deleted on
    # cleanup.
    self._free_disk = 1003
    cache = self.get_cache(_get_policies(min_free_space=1000))
    h_foo = self._algo('foo').hexdigest()
    self.assertEqual([], sorted(cache._lru._items.iteritems()))
    cache.write(h_foo, ['foo'])
    self.assertEqual([], cache.trim())
    self.assertEqual([h_foo], [i[0] for i in cache._lru._items.iteritems()])

    h_a = self._algo('a').hexdigest()
    local_caching.file_write(os.path.join(cache.cache_dir, h_a), 'a')

    # file_path.remove() explicitly handle the +R bit on Windows.
    file_path.remove(os.path.join(cache.cache_dir, h_foo))

    # Still hasn't realized that the file is missing.
    self.assertEqual([h_foo], [i[0] for i in cache._lru._items.iteritems()])
    self.assertEqual(
        sorted([h_a, cache.STATE_FILE]), sorted(fs.listdir(cache.cache_dir)))
    cache.cleanup()
    self.assertEqual([cache.STATE_FILE], fs.listdir(cache.cache_dir))

  def test_policies_active_trimming(self):
    # Start with a larger cache, add many object.
    # Reload the cache with smaller policies, the cache should be trimmed on
    # load.
    h_a = self._algo('a').hexdigest()
    h_b = self._algo('b').hexdigest()
    h_c = self._algo('c').hexdigest()
    large = 'b' * 99
    h_large = self._algo(large).hexdigest()

    def assertItems(expected):
      actual = [
        (digest, size) for digest, (size, _) in cache._lru._items.iteritems()]
      self.assertEqual(expected, actual)

    self._free_disk = 1101
    cache = self.get_cache(_get_policies(
        max_cache_size=100,
        max_items=2,
        min_free_space=1000))
    cache.write(h_a, 'a')
    cache.write(h_large, large)
    # Cache (size and # items) is not enforced while adding items. The
    # rationale is that a task may request more data than the size of the
    # cache policies. As long as there is free space, this is fine.
    cache.write(h_b, 'b')
    assertItems([(h_a, 1), (h_large, len(large)), (h_b, 1)])
    self.assertEqual(h_a, cache._protected)
    self.assertEqual(1000, cache._free_disk)
    # Free disk is enforced, because otherwise we assume the task wouldn't
    # be able to start. In this case, it throws an exception since all items
    # are protected. The item is added since it's detected after the fact.
    with self.assertRaises(local_caching.NoMoreSpace):
      cache.write(h_c, 'c')
    self.assertEqual([1, 99], cache.trim())

    # At this point, after the implicit trim in __exit__(), h_a and h_large were
    # evicted.
    self.assertEqual(
        sorted([unicode(h_b), unicode(h_c), cache.STATE_FILE]),
        sorted(fs.listdir(cache.cache_dir)))

    # Allow 3 items and 101 bytes so h_large is kept.
    cache = self.get_cache(_get_policies(
        max_cache_size=101,
        min_free_space=1000,
        max_items=3,
        max_age_secs=0))
    cache.write(h_large, large)
    self.assertEqual(3, len(cache))
    self.assertEqual(101, cache.total_size)
    self.assertEqual([], cache.trim())

    self.assertEqual(
        sorted([h_b, h_c, h_large, cache.STATE_FILE]),
        sorted(fs.listdir(cache.cache_dir)))

    # Assert that trimming is done in constructor too.
    cache = self.get_cache(_get_policies(
        max_cache_size=100,
        min_free_space=1000,
        max_items=2,
        max_age_secs=0))
    assertItems([(h_c, 1), (h_large, len(large))])
    self.assertEqual(None, cache._protected)
    self.assertEqual(1202, cache._free_disk)
    self.assertEqual(2, len(cache))
    self.assertEqual(100, cache.total_size)
    self.assertEqual([], cache.trim())

  def test_trim_policies_trim_old(self):
    # Add two items, one 3 weeks and one minute old, one recent, make sure the
    # old one is trimmed.
    cache = self.get_cache(_get_policies(
        max_cache_size=1000,
        min_free_space=0,
        max_items=1000,
        max_age_secs=21*24*60*60))
    self._now = 100
    # Test the very limit of 3 weeks:
    cache.write(self._algo('old').hexdigest(), 'old')
    self._now += 1
    cache.write(self._algo('recent').hexdigest(), 'recent')
    self._now += 21*24*60*60
    self.assertEqual([3], cache.trim())
    self.assertEqual([self._algo('recent').hexdigest()], list(cache))

  def test_some_file_brutally_deleted(self):
    h_a = self._algo('a').hexdigest()
    self._free_disk = 1100
    cache = self.get_cache(_get_policies())
    cache.write(h_a, 'a')
    self.assertTrue(cache.touch(h_a, local_caching.UNKNOWN_FILE_SIZE))
    self.assertTrue(cache.touch(h_a, 1))
    self.assertEqual([], cache.trim())

    # file_path.remove() explicitly handle the +R bit on Windows.
    file_path.remove(os.path.join(cache.cache_dir, h_a))

    cache = self.get_cache(_get_policies())
    # 'Ghost' entry loaded with state.json is still there.
    self.assertEqual([h_a], list(cache))
    # 'touch' detects the file is missing by returning False.
    self.assertFalse(cache.touch(h_a, local_caching.UNKNOWN_FILE_SIZE))
    self.assertFalse(cache.touch(h_a, 1))
    # 'touch' evicted the entry.
    self.assertEqual([], list(cache))


class NamedCacheTest(TestCase, CacheTestMixin):
  def setUp(self):
    super(NamedCacheTest, self).setUp()
    self.cache_dir = os.path.join(self.tempdir, 'cache')

  def get_cache(self, policies):
    return local_caching.NamedCache(self.cache_dir, policies)

  def test_clean_cache(self):
    dest_dir = os.path.join(self.tempdir, 'dest')
    cache = self.get_cache(_get_policies())
    self.assertEqual([], fs.listdir(cache.cache_dir))

    a_path = os.path.join(dest_dir, u'a')
    b_path = os.path.join(dest_dir, u'b')

    self.assertEqual(0, cache.install(a_path, u'1'))
    self.assertEqual(0, cache.install(b_path, u'2'))
    self.assertEqual(
        False, fs.exists(os.path.join(cache.cache_dir, cache.NAMED_DIR)))

    self.assertEqual({u'a', u'b'}, set(fs.listdir(dest_dir)))
    self.assertFalse(cache.available)
    self.assertEqual([cache.STATE_FILE], fs.listdir(cache.cache_dir))

    write_file(os.path.join(a_path, u'x'), u'x')
    write_file(os.path.join(b_path, u'y'), u'y')

    self.assertEqual(1, cache.uninstall(a_path, u'1'))
    self.assertEqual(1, cache.uninstall(b_path, u'2'))

    self.assertEqual(4, len(fs.listdir(cache.cache_dir)))
    path1 = os.path.join(cache.cache_dir, cache._lru['1'][0])
    self.assertEqual('x', read_file(os.path.join(path1, u'x')))
    path2 = os.path.join(cache.cache_dir, cache._lru['2'][0])
    self.assertEqual('y', read_file(os.path.join(path2, u'y')))
    self.assertEqual(
        os.path.join(u'..', cache._lru['1'][0]),
        fs.readlink(cache._get_named_path('1')))
    self.assertEqual(
        os.path.join(u'..', cache._lru['2'][0]),
        fs.readlink(cache._get_named_path('2')))
    self.assertEqual(
        [u'1', u'2'],
        sorted(fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR))))

  def test_existing_cache(self):
    # Ensures that the code does what is expected under number use.
    dest_dir = os.path.join(self.tempdir, 'dest')
    cache = self.get_cache(_get_policies())
    # Assume test_clean passes.
    a_path = os.path.join(dest_dir, u'a')
    b_path = os.path.join(dest_dir, u'b')

    self.assertEqual(0, cache.install(a_path, u'1'))
    write_file(os.path.join(dest_dir, u'a', u'x'), u'x')
    self.assertEqual(1, cache.uninstall(a_path, u'1'))

    # Test starts here.
    self.assertEqual(1, cache.install(a_path, u'1'))
    self.assertEqual(0, cache.install(b_path, u'2'))
    self.assertEqual({'a', 'b'}, set(fs.listdir(dest_dir)))
    self.assertFalse(cache.available)
    self.assertEqual(
        sorted([cache.NAMED_DIR, cache.STATE_FILE]),
        sorted(fs.listdir(cache.cache_dir)))
    self.assertEqual(
        [], fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR)))

    self.assertEqual('x', read_file(os.path.join(dest_dir, u'a', u'x')))
    write_file(os.path.join(a_path, 'x'), 'x2')
    write_file(os.path.join(b_path, 'y'), 'y')

    self.assertEqual(2, cache.uninstall(a_path, '1'))
    self.assertEqual(1, cache.uninstall(b_path, '2'))

    self.assertEqual(4, len(fs.listdir(cache.cache_dir)))
    path1 = os.path.join(cache.cache_dir, cache._lru['1'][0])
    self.assertEqual('x2', read_file(os.path.join(path1, 'x')))
    path2 = os.path.join(cache.cache_dir, cache._lru['2'][0])
    self.assertEqual('y', read_file(os.path.join(path2, 'y')))
    self.assertEqual(
        os.path.join(u'..', cache._lru['1'][0]),
        fs.readlink(cache._get_named_path('1')))
    self.assertEqual(
        os.path.join(u'..', cache._lru['2'][0]),
        fs.readlink(cache._get_named_path('2')))
    self.assertEqual(
        [u'1', u'2'],
        sorted(fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR))))

  def test_install_throws(self):
    old_isdir = None
    banged = []
    # Crashes, but only on the first call.
    def bang(path):
      if not banged:
        banged.append(True)
        raise IOError('fake')
      return old_isdir(path)

    cache = self.get_cache(_get_policies())
    dest_dir = os.path.join(self.tempdir, 'dest')

    # fs.isdir() happens to be the first function called.
    old_isdir = self.mock(fs, 'isdir', bang)
    with self.assertRaises(local_caching.NamedCacheError):
      cache.install(dest_dir, u'1')

  def test_uninstall_throws(self):
    old_isdir = None
    banged = []
    # Crashes, but only on the first call.
    def bang(path):
      if not banged:
        banged.append(True)
        raise IOError('fake')
      return old_isdir(path)

    cache = self.get_cache(_get_policies())
    dest_dir = os.path.join(self.tempdir, 'dest')
    self.assertEqual(0, cache.install(dest_dir, u'1'))

    # fs.isdir() happens to be the first function called.
    old_isdir = self.mock(fs, 'isdir', bang)
    with self.assertRaises(local_caching.NamedCacheError):
      cache.uninstall(dest_dir, u'1')

  def test_cycle_twice(self):
    # Ensure that named symlink works.
    cache = self.get_cache(_get_policies())
    dest_dir = os.path.join(self.tempdir, 'dest')
    self.assertEqual(0, cache.install(dest_dir, u'1'))
    with fs.open(os.path.join(dest_dir, u'hi'), 'wb') as f:
      f.write('hello')
    self.assertEqual(5, cache.uninstall(dest_dir, u'1'))
    self.assertEqual(
        [u'1'], fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR)))
    self.assertEqual(True, cache.cleanup())
    self.assertEqual(5, cache.install(dest_dir, u'1'))
    self.assertEqual(5, cache.uninstall(dest_dir, u'1'))
    self.assertEqual(
        [u'1'], fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR)))
    self.assertEqual(
        [u'hi'],
        fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR, u'1')))

  def test_save_named(self):
    cache = self.get_cache(_get_policies())
    self.assertEqual([], sorted(fs.listdir(cache.cache_dir)))

    self._add_one_item(cache, 2)
    with fs.open(os.path.join(cache.cache_dir, cache.STATE_FILE)) as f:
      old_content = json.load(f)
    # It's immediately saved.
    items = lru.LRUDict.load(os.path.join(cache.cache_dir, cache.STATE_FILE))
    self.assertEqual(1, len(items))
    _key, (v, _timestamp) = items.get_oldest()
    # This depends on the inner format as generated by NamedCache.
    entry_dir_name = v[0]
    self.assertEqual(
        sorted([entry_dir_name, cache.NAMED_DIR, cache.STATE_FILE]),
        sorted(fs.listdir(cache.cache_dir)))

    cache.save()
    self.assertEqual(
        sorted([entry_dir_name, cache.NAMED_DIR, cache.STATE_FILE]),
        sorted(fs.listdir(cache.cache_dir)))
    with fs.open(os.path.join(cache.cache_dir, cache.STATE_FILE)) as f:
      new_content = json.load(f)
    # That's because uninstall() called from self._add_one_item()
    # causes an implicit save(). See uninstall() comments for more details.
    self.assertEqual(new_content, old_content)

  def test_trim(self):
    cache = self.get_cache(_get_policies(max_items=2))
    item_count = 12
    for i in xrange(item_count):
      self._add_one_item(cache, i+1)
    self.assertEqual(len(cache), item_count)
    self.assertEqual([1, 2, 3, 4, 5, 6, 7, 8, 9, 10], cache.trim())
    self.assertEqual(len(cache), 2)
    self.assertEqual(
        ['11', '12'],
        sorted(fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR))))

  def test_load_corrupted_state(self):
    # cleanup() handles a broken state file.
    fs.mkdir(self.cache_dir)
    c = local_caching.NamedCache
    with fs.open(os.path.join(self.cache_dir, c.STATE_FILE), 'w') as f:
      f.write('}}}}')
    fs.makedirs(os.path.join(self.cache_dir, '1'), 0777)

    cache = self.get_cache(_get_policies())
    self._add_one_item(cache, 1)
    self.assertTrue(
        fs.exists(os.path.join(cache.cache_dir, cache.NAMED_DIR, '1')))
    self.assertTrue(
        fs.islink(os.path.join(cache.cache_dir, cache.NAMED_DIR, '1')))
    self.assertEqual([], cache.trim())
    self.assertTrue(
        fs.exists(os.path.join(cache.cache_dir, cache.NAMED_DIR, '1')))
    self.assertTrue(
        fs.islink(os.path.join(cache.cache_dir, cache.NAMED_DIR, '1')))
    self.assertEqual(True, cache.cleanup())
    self.assertEqual(
        sorted([cache.NAMED_DIR, cache.STATE_FILE, cache._lru[u'1'][0]]),
        sorted(fs.listdir(cache.cache_dir)))

  def test_cleanup_missing(self):
    # cleanup() detects a missing item.
    cache = self.get_cache(_get_policies())
    self._add_one_item(cache, 1)
    file_path.rmtree(os.path.join(cache.cache_dir, cache._lru[u'1'][0]))

    cache = self.get_cache(_get_policies())
    self.assertEqual([u'1'], list(cache))
    self.assertEqual(True, cache.cleanup())
    self.assertEqual([], list(cache))

  def test_cleanup_unexpected(self):
    # cleanup() delete unexpected file in the cache directory.
    fs.mkdir(self.cache_dir)
    with fs.open(os.path.join(self.cache_dir, u'junk'), 'w') as f:
      f.write('random')
    cache = self.get_cache(_get_policies())
    self.assertEqual(['junk'], fs.listdir(cache.cache_dir))
    self.assertEqual(True, cache.cleanup())
    self.assertEqual([cache.STATE_FILE], fs.listdir(cache.cache_dir))

  def test_cleanup_unexpected_named(self):
    # cleanup() deletes unexpected symlink and directory in named/.
    fs.mkdir(self.cache_dir)
    c = local_caching.NamedCache
    fs.mkdir(os.path.join(self.cache_dir, c.NAMED_DIR))
    p = os.path.join(self.cache_dir, c.NAMED_DIR, u'junk_file')
    with fs.open(p, 'w') as f:
      f.write('random')
    fs.mkdir(os.path.join(self.cache_dir, c.NAMED_DIR, u'junk_dir'))
    fs.symlink(
        'invalid_dest',
        os.path.join(self.cache_dir, c.NAMED_DIR, u'junk_link'))

    cache = self.get_cache(_get_policies())
    self.assertEqual([cache.NAMED_DIR], fs.listdir(cache.cache_dir))
    self.assertEqual(
        ['junk_dir', 'junk_file', 'junk_link'],
        sorted(fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR))))
    self.assertEqual(True, cache.cleanup())
    self.assertEqual(
        [cache.NAMED_DIR, cache.STATE_FILE],
        sorted(fs.listdir(cache.cache_dir)))
    self.assertEqual(
        [], fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR)))

  def test_cleanup_incorrect_link(self):
    # cleanup() repairs broken symlink in named/.
    cache = self.get_cache(_get_policies())
    self._add_one_item(cache, 1)
    self._add_one_item(cache, 2)
    fs.remove(os.path.join(self.cache_dir, cache.NAMED_DIR, u'1'))
    fs.remove(os.path.join(self.cache_dir, cache.NAMED_DIR, u'2'))
    fs.symlink(
        'invalid_dest', os.path.join(self.cache_dir, cache.NAMED_DIR, u'1'))
    fs.mkdir(os.path.join(self.cache_dir, cache.NAMED_DIR, u'2'))

    cache = self.get_cache(_get_policies())
    self.assertEqual(
        ['1', '2'],
        sorted(fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR))))
    self.assertEqual(True, cache.cleanup())
    self.assertEqual(
        [], fs.listdir(os.path.join(cache.cache_dir, cache.NAMED_DIR)))

  def test_upgrade(self):
    # Make sure upgrading works. This is temporary as eventually all bots will
    # be updated.
    now = time.time()
    fs.mkdir(self.cache_dir)
    fs.mkdir(os.path.join(self.cache_dir, 'f1'))
    with fs.open(os.path.join(self.cache_dir, 'f1', 'hello'), 'wb') as f:
      f.write('world')
    # v1
    old = {
      'version': 2,
      'items': [
        ['cache1', ['f1', now]],
      ],
    }
    c = local_caching.NamedCache
    with fs.open(os.path.join(self.cache_dir, c.STATE_FILE), 'w') as f:
      json.dump(old, f)
    # It automatically upgrades to v2.
    cache = self.get_cache(_get_policies())
    expected = {u'cache1': ((u'f1', len('world')), now)}
    self.assertEqual(expected, dict(cache._lru._items.iteritems()))
    self.assertEqual(
        [u'f1', cache.STATE_FILE], sorted(fs.listdir(cache.cache_dir)))


def _gen_state(items):
  state = {'items': items, 'version': 2}
  return json.dumps(state, sort_keys=True, separators=(',', ':'))


class FnTest(TestCase):
  """Test functions that leverage both DiskContentAddressedCache and
  NamedCache.
  """
  def setUp(self):
    super(FnTest, self).setUp()
    # Simulate that the memory cache used disk space.
    def remove_oldest(c):
      s = old_remove_oldest(c)
      self._free_disk += s
      return s
    old_remove_oldest = self.mock(
        local_caching.MemoryContentAddressedCache, 'remove_oldest',
        remove_oldest)

  def put_to_named_cache(self, manager, cache_name, file_name, contents):
    """Puts files into named cache."""
    cache_dir = os.path.join(self.tempdir, 'put_to_named_cache')
    manager.install(cache_dir, cache_name)
    with fs.open(os.path.join(cache_dir, file_name), 'wb') as f:
      f.write(contents)
    manager.uninstall(cache_dir, cache_name)

  def _prepare_cache(self, cache):
    now = self._now
    for i in xrange(1, 11):
      self._add_one_item(cache, i)
      self._now += 1
    self._now = now
    self.assertEqual([], cache.trim())

  def _prepare_isolated_cache(self, cache):
    self._prepare_cache(cache)
    self._verify_isolated_cache(cache, range(1, 11))

  def _verify_isolated_cache(self, cache, items):
    # Isolated cache verification.
    expected = {
      unicode(self._algo(_gen_data(n)).hexdigest()): _gen_data(n) for n in items
    }
    expected[cache.STATE_FILE] = _gen_state(
        [
          [unicode(self._algo(_gen_data(n)).hexdigest()), [n, self._now+n-1]]
          for n in items
        ])
    self.assertEqual(expected, read_tree(cache.cache_dir))

  def _prepare_named_cache(self, cache):
    self._prepare_cache(cache)
    # Figure out the short names via the symlinks.
    items = range(1, 11)
    short_names = {
      n: os.path.basename(fs.readlink(
          os.path.join(cache.cache_dir, cache.NAMED_DIR, unicode(n))))
      for n in items
    }
    self._verify_named_cache(cache, short_names, items)
    return short_names

  def _verify_named_cache(self, cache, short_names, items):
    # Named cache verification. Ensures the cache contain the expected data.
    actual = read_tree(cache.cache_dir)
    # There's assumption about json encoding format but here it's good enough.
    expected = {
        os.path.join(short_names[n], u'hello'): _gen_data(n) for n in items
    }
    expected[cache.STATE_FILE] = _gen_state(
        [
          [unicode(n), [[short_names[n], n], self._now+n-1]]
          for n in items
        ])
    self.assertEqual(expected, actual)

  def test_clean_caches_disk(self):
    # Create an isolated cache and a named cache each with 2 items. Ensure that
    # one item from each is removed.
    now = self._now
    self._free_disk = 100000

    # Setup caches.
    policies = _get_policies(min_free_space=1000)
    named_cache = local_caching.NamedCache(
        tempfile.mkdtemp(dir=self.tempdir, prefix='nc'), policies)
    short_names = self._prepare_named_cache(named_cache)

    isolated_cache = local_caching.DiskContentAddressedCache(
        tempfile.mkdtemp(dir=self.tempdir, prefix='ic'), policies, trim=False)
    self._prepare_isolated_cache(isolated_cache)
    self.assertEqual(now, self._now)

    # Request triming.
    self._free_disk = 950
    trimmed = local_caching.trim_caches(
        [isolated_cache, named_cache],
        self.tempdir,
        min_free_space=policies.min_free_space,
        max_age_secs=policies.max_age_secs)
    # Enough to free 50 bytes. The following sums to 56.
    expected = [1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7]
    self.assertEqual(expected, trimmed)

    # Cache verification.
    self._verify_named_cache(named_cache, short_names, range(8, 11))
    self._verify_isolated_cache(isolated_cache, range(8, 11))

  def _get_5_caches(self):
    # Add items from size 1 to 101 randomly into 5 caches.
    caches = [
      local_caching.MemoryContentAddressedCache(),
      local_caching.MemoryContentAddressedCache(),
      local_caching.MemoryContentAddressedCache(),
      local_caching.MemoryContentAddressedCache(),
      local_caching.MemoryContentAddressedCache(),
    ]
    for i in xrange(100):
      self._add_one_item(caches[random.randint(0, len(caches)-1)], i+1)
      self._now += 1
    return caches

  def test_clean_caches_memory_size(self):
    # Test that cleaning is correctly distributed independent of the cache
    # location.
    caches = self._get_5_caches()
    # 100 bytes must be freed.
    self._free_disk = 900
    trimmed = local_caching.trim_caches(
        caches,
        self.tempdir,
        min_free_space=1000,
        max_age_secs=0)
    # sum(range(1, 15)) == 105, the first value after 100.
    self.assertEqual(range(1, 15), trimmed)

  def test_clean_caches_memory_time(self):
    # Test that cleaning is correctly distributed independent of the cache
    # location.
    caches = self._get_5_caches()
    self.mock(time, 'time', lambda: self._now)
    trimmed = local_caching.trim_caches(
        caches,
        self.tempdir,
        min_free_space=0,
        max_age_secs=10)
    # Only the last 10 items are kept. The first 90 items were trimmed.
    self.assertEqual(range(1, 91), trimmed)


if __name__ == '__main__':
  fix_encoding.fix_encoding()
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  logging.basicConfig(
      level=(logging.DEBUG if '-v' in sys.argv else logging.CRITICAL))
  unittest.main()
