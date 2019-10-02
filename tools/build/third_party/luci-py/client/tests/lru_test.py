#!/usr/bin/env python
# Copyright 2013 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import logging
import os
import sys
import tempfile
import unittest

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(
    __file__.decode(sys.getfilesystemencoding()))))
sys.path.insert(0, ROOT_DIR)

from utils import lru


def _load_from_raw(state_text):
  """Makes a LRUDict by loading the given JSON from a file."""
  handle, tmp_name = tempfile.mkstemp(prefix=u'lru_test')
  os.close(handle)
  try:
    with open(tmp_name, 'w') as f:
      f.write(state_text)
    return lru.LRUDict.load(tmp_name)
  finally:
    try:
      os.unlink(tmp_name)
    except OSError:
      pass


def _save_and_load(lru_dict):
  """Saves then reloads a LRUDict instance."""
  handle, tmp_name = tempfile.mkstemp(prefix=u'lru_test')
  os.close(handle)
  try:
    lru_dict.save(tmp_name)
    return lru.LRUDict.load(tmp_name)
  finally:
    try:
      os.unlink(tmp_name)
    except OSError:
      pass


def _prepare_lru_dict(data):
  """Returns new LRUDict with given |keys| added one by one."""
  lru_dict = lru.LRUDict()
  for key, val in data:
    lru_dict.add(key, val)
  return lru_dict


class LRUDictTest(unittest.TestCase):
  def assert_same_data(self, expected, lru_dict):
    """Asserts that given |lru_dict| contains same data as |expected|.

    Tests iteritems(), itervalues(), get(), __iter__.
    """
    self.assertEqual(list(lru_dict.iteritems()), expected)
    self.assertEqual(set(lru_dict), set(k for k, v in expected))
    self.assertEqual(list(lru_dict.itervalues()), [v for k, v in expected])
    for k, v in expected:
      self.assertEqual(lru_dict.get(k), v)

  def test_empty(self):
    self.assert_same_data([], lru.LRUDict())

  def test_magic_methods_empty(self):
    """Tests __nonzero__, __iter, __len__, __getitem__ and __contains__."""
    # Check for empty dict.
    lru_dict = lru.LRUDict()
    self.assertFalse(lru_dict)
    self.assertEqual(len(lru_dict), 0)
    self.assertFalse(1 in lru_dict)
    self.assertFalse([i for i in lru_dict])
    with self.assertRaises(KeyError):
      _ = lru_dict[1]

  def test_magic_methods_nonempty(self):
    """Tests __nonzero__, __iter, __len__, __getitem__ and __contains__."""
    # Dict with one item.
    lru_dict = lru.LRUDict()
    lru_dict.add(1, 'one')
    self.assertTrue(lru_dict)
    self.assertEqual(len(lru_dict), 1)
    self.assertTrue(1 in lru_dict)
    self.assertFalse(2 in lru_dict)
    self.assertTrue([i for i in lru_dict])
    self.assertEqual('one', lru_dict[1])

  def test_add(self):
    lru_dict = _prepare_lru_dict([(1, 'one'), (2, 'two'), (3, 'three')])
    lru_dict.add(1, 'one!!!')
    expected = [(2, 'two'), (3, 'three'), (1, 'one!!!')]
    self.assert_same_data(expected, lru_dict)
    lru_dict.add(0, 'zero')
    expected = [(2, 'two'), (3, 'three'), (1, 'one!!!'), (0, 'zero')]
    self.assert_same_data(expected, lru_dict)

  def test_pop_first(self):
    lru_dict = _prepare_lru_dict([(1, 'one'), (2, 'two'), (3, 'three')])
    lru_dict.pop(1)
    self.assert_same_data([(2, 'two'), (3, 'three')], lru_dict)

  def test_pop_middle(self):
    lru_dict = _prepare_lru_dict([(1, 'one'), (2, 'two'), (3, 'three')])
    lru_dict.pop(2)
    self.assert_same_data([(1, 'one'), (3, 'three')], lru_dict)

  def test_pop_last(self):
    lru_dict = _prepare_lru_dict([(1, 'one'), (2, 'two'), (3, 'three')])
    lru_dict.pop(3)
    self.assert_same_data([(1, 'one'), (2, 'two')], lru_dict)

  def test_pop_missing(self):
    lru_dict = _prepare_lru_dict([(1, 'one'), (2, 'two'), (3, 'three')])
    with self.assertRaises(KeyError):
      lru_dict.pop(4)

  def test_touch(self):
    lru_dict = _prepare_lru_dict([(1, 'one'), (2, 'two'), (3, 'three')])
    lru_dict.touch(2)
    self.assert_same_data([(1, 'one'), (3, 'three'), (2, 'two')], lru_dict)
    with self.assertRaises(KeyError):
      lru_dict.touch(4)

  def test_timestamp(self):
    """Tests get_oldest, pop_oldest."""
    lru_dict = lru.LRUDict()

    now = 0
    lru_dict.time_fn = lambda: now

    lru_dict.add('ka', 'va')
    now += 1

    lru_dict.add('kb', 'vb')
    now += 1

    self.assertEqual(lru_dict.get_oldest(), ('ka', ('va', 0)))
    self.assertEqual(lru_dict.pop_oldest(), ('ka', ('va', 0)))
    self.assertEqual(lru_dict.get_oldest(), ('kb', ('vb', 1)))
    self.assertEqual(lru_dict.pop_oldest(), ('kb', ('vb', 1)))

  def test_transform(self):
    lru_dict = lru.LRUDict()
    lru_dict.add('ka', 'va')
    lru_dict.add('kb', 'vb')
    lru_dict.transform(lambda k, v: v + '*')
    self.assert_same_data([('ka', 'va*'), ('kb', 'vb*')], lru_dict)

  def test_load_save_empty(self):
    self.assertFalse(_save_and_load(lru.LRUDict()))

  def test_load_save(self):
    data = [(1, None), (2, None), (3, None)]
    # Normal flow.
    lru_dict = _prepare_lru_dict(data)
    expected = [(1, None), (2, None), (3, None)]
    self.assert_same_data(expected, _save_and_load(lru_dict))

    # After touches.
    lru_dict = _prepare_lru_dict(data)
    lru_dict.touch(2)
    expected = [(1, None), (3, None), (2, None)]
    self.assert_same_data(expected, _save_and_load(lru_dict))

    # After pop.
    lru_dict = _prepare_lru_dict(data)
    lru_dict.pop(2)
    expected = [(1, None), (3, None)]
    self.assert_same_data(expected, _save_and_load(lru_dict))

    # After add.
    lru_dict = _prepare_lru_dict(data)
    lru_dict.add(4, 4)
    expected = [(1, None), (2, None), (3, None), (4, 4)]
    self.assert_same_data(expected, _save_and_load(lru_dict))

  def test_corrupted_state_file(self):
    # Loads correct state just fine.
    s = _load_from_raw(json.dumps(
      {
        'version': 2,
        'items': [
          ['key1', ['value1', 1]],
          ['key2', ['value2', 2]],
        ],
      }))
    self.assertIsNotNone(s)
    self.assertEqual(2, len(s))

    # Not a json.
    with self.assertRaises(ValueError):
      _load_from_raw('garbage, not a state')

    # Not a list.
    with self.assertRaises(ValueError):
      _load_from_raw('{}')

    # Not a list of pairs.
    with self.assertRaises(ValueError):
      _load_from_raw(json.dumps([
          ['key', 'value', 'and whats this?'],
      ]))

    # Duplicate keys.
    with self.assertRaises(ValueError):
      _load_from_raw(json.dumps([
          ['key', 'value'],
          ['key', 'another_value'],
      ]))


if __name__ == '__main__':
  VERBOSE = '-v' in sys.argv
  logging.basicConfig(level=logging.DEBUG if VERBOSE else logging.ERROR)
  unittest.main()
