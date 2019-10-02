#!/usr/bin/env python

# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import sys
import unittest

from test_support import test_env
test_env.setup_test_env()

from test_support import test_case
import partial


class ApplyTestCase(test_case.TestCase):
  """Tests for partial._apply."""

  def test_simple(self):
    d = {}
    partial._apply(d, {'a': {}})
    self.assertFalse(d)

    d = {'a': 1}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': 1,
    })

    d = {'a': 1, 'b': 2, 'c': 3}
    partial._apply(d, {'a': {}, 'c': {}})
    self.assertEqual(d, {
      'a': 1,
      'c': 3,
    })

  def test_recursive(self):
    d = {'a': {}}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': {},
    })

    d = {'a': {'b': 1}}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': {
        'b': 1,
      },
    })

    d = {'a': []}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': [],
    })

    d = {'a': [1, 2, 3]}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': [1, 2, 3],
    })

    d = {'a': [{'b': 1}, {'b': 2}, {'b': 3}]}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': [
        {'b': 1},
        {'b': 2},
        {'b': 3},
      ],
    })

    d = {'a': [1, {'b': 2}, 3]}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': [
        1,
        {'b': 2},
        3,
      ],
    })

    d = {'a': [1, {}, 3]}
    partial._apply(d, {'a': {}})
    self.assertEqual(d, {
      'a': [
        1,
        {},
        3,
      ],
    })

    d = {'a': [1, {'b': 2}, 3]}
    partial._apply(d, {'a': {'b': {}}})
    self.assertEqual(d, {
      'a': [
        1,
        {'b': 2},
        3,
      ],
    })

    d = {'a': [{'b': {'c': 1, 'd': 2}}, {'b': {'c': 3, 'd': 4}}]}
    partial._apply(d, {'a': {'b': {'d': {}}}})
    self.assertEqual(d, {
      'a': [
        {
          'b': {'d': 2},
        },
        {
          'b': {'d': 4},
        },
      ],
    })

    d = {'a': [{'b': {'c': 1, 'd': 2}}, {'b': {'c': 3, 'd': 4}}]}
    partial._apply(d, {'a': {'b': {'e': {}}}})
    self.assertFalse(d)

  def test_star(self):
    d = {'a': 1, 'b': 2, 'c': 3}
    partial._apply(d, {'*': {}})
    self.assertEqual(d, {
      'a': 1,
      'b': 2,
      'c': 3,
    })

    d = {'a': {'b': 1}, 'c': {'d': {'e': 3, 'f': 4}}}
    partial._apply(d, {'*': {}})
    self.assertEqual(d, {
      'a': {
        'b': 1,
      },
      'c': {
        'd': {
          'e': 3,
          'f': 4,
        },
      },
    })

    d = {'a': {'b': 1}, 'c': {'d': {'e': 3, 'f': 4}}}
    partial._apply(d, {'*': {'d': {}}})
    self.assertEqual(d, {
      'c': {
        'd': {
          'e': 3,
          'f': 4,
        },
      },
    })

    d = {'a': {'b': 1}, 'c': {'d': {'e': 3, 'f': 4}}}
    partial._apply(d, {'a': {}, '*': {'d': {}}})
    self.assertEqual(d, {
      'a': {
        'b': 1,
      },
      'c': {
        'd': {
          'e': 3,
          'f': 4,
        },
      },
    })

    d = {'a': {'b': {'c': 1, 'd': 2}, 'e': {'c': 3, 'd': 4}}}
    partial._apply(d, {'a': {'b': {'d': {}}, '*': {'d': {}}}})
    self.assertEqual(d, {
      'a': {
        'b': {
          'd': 2,
        },
        'e': {
          'd': 4,
        },
      },
    })

    d = {'a': [1, 2, 3], 'b': [4, 5, 6]}
    partial._apply(d, {'*': {}})
    self.assertEqual(d, {
      'a': [1, 2, 3],
      'b': [4, 5, 6],
    })

    d = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}]}
    partial._apply(d, {'*': {}})
    self.assertEqual(d, {
      'a': [
        {
          'b': 1,
          'c': 2,
        },
        {
          'b': 3,
          'c': 4,
        },
      ],
    })

    d = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}]}
    partial._apply(d, {'a': {'*': {}}})
    self.assertEqual(d, {
      'a': [
        {
          'b': 1,
          'c': 2,
        },
        {
          'b': 3,
          'c': 4,
        },
      ],
    })

    d = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}]}
    partial._apply(d, {'a': {'b': {'*': {}}}})
    self.assertEqual(d, {
      'a': [
        {
          'b': 1,
        },
        {
          'b': 3,
        },
      ],
    })

    d = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}]}
    partial._apply(d, {'*': {'c': {}}})
    self.assertEqual(d, {
      'a': [
        {
          'c': 2,
        },
        {
          'c': 4,
        },
      ],
    })


class MaskTestCase(test_case.TestCase):
  """Tests for partial.mask."""

  def test_simple(self):
    r = {'a': 1, 'b': 2}
    partial.mask(r, 'a')
    self.assertEqual(r, {
      'a': 1,
    })

    r = {'a': 1, 'b': 2}
    partial.mask(r, 'b')
    self.assertEqual(r, {
      'b': 2,
    })

    r = {'a': 1, 'b': 2}
    partial.mask(r, 'a,b')
    self.assertEqual(r, {
      'a': 1,
      'b': 2,
    })

  def test_recursive(self):
    r = {'a': {'b': 1, 'c': 2}, 'd': {'e': 3, 'f': 4}}
    partial.mask(r, 'a')
    self.assertEqual(r, {
      'a': {
        'b': 1,
        'c': 2,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'e': 3, 'f': 4}}
    partial.mask(r, 'd')
    self.assertEqual(r, {
      'd': {
        'e': 3,
        'f': 4,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'e': 3, 'f': 4}}
    partial.mask(r, 'a,d')
    self.assertEqual(r, {
      'a': {
        'b': 1,
        'c': 2,
      },
      'd': {
        'e': 3,
        'f': 4,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'e': 3, 'f': 4}}
    partial.mask(r, 'a/b')
    self.assertEqual(r, {
      'a': {
        'b': 1,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'e': 3, 'f': 4}}
    partial.mask(r, 'a/b,d/f')
    self.assertEqual(r, {
      'a': {
        'b': 1,
      },
      'd': {
        'f': 4,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'b': 3, 'c': 4}}
    partial.mask(r, 'a/x,d/c')
    self.assertEqual(r, {
      'd': {
        'c': 4,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'b': 3, 'c': 4}}
    partial.mask(r, 'a/b,d/x')
    self.assertEqual(r, {
      'a': {
        'b': 1,
      },
    })

    r = {'a': {'b': 1, 'c': 2}, 'd': {'b': 3, 'c': 4}}
    partial.mask(r, 'a/x,d/y')
    self.assertFalse(r)

    r = {'a': {'b': 1, 'c': 2, 'd': 3}, 'e': {'b': 4, 'c': 5, 'd': 6}}
    partial.mask(r, 'a(b,d),e/c')
    self.assertEqual(r, {
      'a': {
        'b': 1,
        'd': 3,
      },
      'e': {
        'c': 5,
      },
    })

    r = {'a': {'b': 1, 'c': 2, 'd': 3}, 'e': {'b': 4, 'c': 5, 'd': 6}}
    partial.mask(r, '*(b,d),e/c')
    self.assertEqual(r, {
      'a': {
        'b': 1,
        'd': 3,
      },
      'e': {
        'b': 4,
        'c': 5,
        'd': 6,
      },
    })

    r = {'a': {'b': 1, 'c': 2, 'd': 3}, 'e': {'b': 4, 'c': 5, 'd': 6}}
    partial.mask(r, 'a(*),e/c')
    self.assertEqual(r, {
      'a': {
        'b': 1,
        'c': 2,
        'd': 3,
      },
      'e': {
        'c': 5,
      },
    })

    r = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}], 'd': [{'b': 5, 'c': 6}]}
    partial.mask(r, 'a(b)')
    self.assertEqual(r, {
      'a': [
        {'b': 1},
        {'b': 3},
      ],
    })

    r = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}], 'd': [{'b': 5, 'c': 6}]}
    partial.mask(r, 'd/b')
    self.assertEqual(r, {
      'd': [
        {'b': 5},
      ],
    })
    r = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}], 'd': [{'b': 5, 'c': 6}]}
    partial.mask(r, 'a(b),d/b')
    self.assertEqual(r, {
      'a': [
        {'b': 1},
        {'b': 3},
      ],
      'd': [
        {'b': 5},
      ],
    })

    r = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}], 'd': [{'b': 5, 'c': 6}]}
    partial.mask(r, '*(b)')
    self.assertEqual(r, {
      'a': [
        {'b': 1},
        {'b': 3},
      ],
      'd': [
        {'b': 5},
      ],
    })

    r = {'a': [{'b': 1, 'c': 2}, {'b': 3, 'c': 4}], 'd': [{'b': 5, 'c': 6}]}
    partial.mask(r, '*(x)')
    self.assertFalse(r)


class MergeTestCase(test_case.TestCase):
  """Tests for partial._merge."""

  def test_empty(self):
    d = {}
    partial._merge({}, d)
    self.assertFalse(d)

    d = {'a': {}}
    partial._merge({}, d)
    self.assertEqual(d, {
      'a': {},
    })

  def test_simple(self):
    d = {}
    partial._merge({'a': {}}, d)
    self.assertEqual(d, {
      'a': {},
    })

    d = {'a': {}}
    partial._merge({'a': {}}, d)
    self.assertEqual(d, {
      'a': {},
    })

    d = {'b': {}}
    partial._merge({'a': {}}, d)
    self.assertEqual(d, {
      'a': {},
      'b': {},
    })

  def test_recursive(self):
    d = {}
    partial._merge({'a': {'b': {}}}, d)
    self.assertEqual(d, {
      'a': {
        'b': {},
      },
    })

    d = {'a': {'c': {}}}
    partial._merge({'a': {'b': {}}}, d)
    self.assertEqual(d, {
      'a': {
        'b': {},
        'c': {},
      },
    })

    d = {'a': {'d': {'e': {}}}}
    partial._merge({'a': {'b': {'c': {}}}}, d)
    self.assertEqual(d, {
      'a': {
        'b': {
          'c': {},
        },
        'd': {
          'e': {},
        }
      },
    })

    d = {'a': {'b': {'c': {'f': {}}, 'g': {}}}}
    partial._merge({'a': {'b': {'c': {'d': {}, 'e': {}}}}}, d)
    self.assertEqual(d, {
      'a': {
        'b': {
          'c': {
            'd': {},
            'e': {},
            'f': {},
          },
          'g': {},
        },
      },
    })


class ParseTestCase(test_case.TestCase):
  """Tests for partial._parse."""

  def test_simple(self):
    self.assertEqual(partial._parse('a'), {
      'a': {},
    })
    self.assertEqual(partial._parse('a,b'), {
      'a': {},
      'b': {},
    })
    self.assertEqual(partial._parse('a,b,c'), {
      'a': {},
      'b': {},
      'c': {},
    })

  def test_components(self):
    self.assertEqual(partial._parse('a/b'), {
      'a': {
        'b': {},
      },
    })
    self.assertEqual(partial._parse('a/b,c'), {
      'a': {
        'b': {},
      },
      'c': {},
    })
    self.assertEqual(partial._parse('a,b/c'), {
      'a': {},
      'b': {
        'c': {},
      },
    })
    self.assertEqual(partial._parse('a/b,c/d'), {
      'a': {
        'b': {},
      },
      'c': {
        'd': {},
      },
    })
    self.assertEqual(partial._parse('a/b/c,d/e/f'), {
      'a': {
        'b': {
          'c': {},
        },
      },
      'd': {
        'e': {
          'f': {},
        },
      },
    })

  def test_subfields(self):
    self.assertEqual(partial._parse('a(b)'), {
      'a': {
        'b': {},
      },
    })
    self.assertEqual(partial._parse('a/b(c)'), {
      'a': {
        'b': {
          'c': {},
        },
      },
    })
    self.assertEqual(partial._parse('a(b/c)'), {
      'a': {
        'b': {
          'c': {},
        },
      },
    })
    self.assertEqual(partial._parse('a(b,c)'), {
      'a': {
        'b': {},
        'c': {},
      },
    })
    self.assertEqual(partial._parse('a(b, c)'), {
      'a': {
        'b': {},
        'c': {},
      },
    })
    self.assertEqual(partial._parse('a (b,c)'), {
      'a': {
        'b': {},
        'c': {},
      },
    })
    self.assertEqual(partial._parse('a/b(c,d)'), {
      'a': {
        'b': {
          'c': {},
          'd': {},
        },
      },
    })
    self.assertEqual(partial._parse('a(b/c,d)'), {
      'a': {
        'b': {
          'c': {},
        },
        'd': {},
      },
    })
    self.assertEqual(partial._parse('a(b,c/d)'), {
      'a': {
        'b': {
        },
        'c': {
          'd': {},
        },
      },
    })
    self.assertEqual(partial._parse('a(b,c(d))'), {
      'a': {
        'b': {},
        'c': {
          'd': {},
         }
      },
    })
    self.assertEqual(partial._parse('a(b),c(d)'), {
      'a': {
        'b': {},
      },
      'c': {
        'd': {},
      },
    })
    self.assertEqual(partial._parse('a(b(c,d),e),f'), {
      'a': {
        'b': {
          'c': {},
          'd': {},
        },
        'e': {},
      },
      'f': {},
    })

  def test_duplicates(self):
    self.assertEqual(partial._parse('a,a'), {
      'a': {},
    })
    self.assertEqual(partial._parse('a,b,a'), {
      'a': {},
      'b': {},
    })
    self.assertEqual(partial._parse('a,b,b,a'), {
      'a': {},
      'b': {},
    })
    self.assertEqual(partial._parse('a/b,a/b'), {
      'a': {
        'b': {},
      },
    })
    self.assertEqual(partial._parse('a/b,a/b,c,d,d'), {
      'a': {
        'b': {},
      },
      'c': {},
      'd': {},
    })
    self.assertEqual(partial._parse('a/b,a(b)'), {
      'a': {
        'b': {},
      },
    })
    self.assertEqual(partial._parse('a(b),a/b,c,d,d'), {
      'a': {
        'b': {},
      },
      'c': {},
      'd': {},
    })
    self.assertEqual(partial._parse('a(b,c),a/b,a/c,a(b),a(c)'), {
      'a': {
        'b': {},
        'c': {},
      },
    })
    self.assertEqual(partial._parse('a/b/c,a(b/d,b/e)'), {
      'a': {
        'b': {
          'c': {},
          'd': {},
          'e': {},
        },
      },
    })

  def test_raises(self):
    fields = [
      '',
      '/',
      'a/',
      '/b',
      'a//b',
      ','
      'a,',
      ',b',
      '/,',
      ',/',
      'a/b,',
      ',a/b',
      '()',
      '(a)',
      '(,)',
      '(a,)',
      '(,b)',
      '(a,b)',
      '(a,b)c',
      'a(())',
      'a((b))',
      'a((b,))',
      'a((,c))',
      'a((b,c))',
      '(',
      'a(',
      'a(b',
      ')',
      'a)',
      'a(b))',
      'a(b),',
      'a(b)c',
      'a(/)',
      'a(b/)',
      'a(/b)',
    ]
    for f in fields:
      with self.assertRaises(partial.ParsingError):
        partial._parse(f)


if __name__ == '__main__':
  if '-v' in sys.argv:
    unittest.TestCase.maxDiff = None
  unittest.main()
