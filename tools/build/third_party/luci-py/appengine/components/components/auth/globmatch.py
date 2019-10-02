# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Implementation of a limited subset of glob language used in AuthDB."""

import re


def match(s, pat):
  """Returns True if string 's' matches glob-like pattern 'pat'.

  The only supported glob syntax is '*', which matches zero or more characters.

  Case sensitive. Doesn't support multi-line strings or patterns. There's no way
  to match '*' itself.
  """
  if '\n' in s or '\n' in pat:
    raise ValueError('Multiline strings are not supported')
  return bool(re.match(_translate(pat), s))


def _translate(pat):
  """Given a pattern, returns a regexp string for it."""
  out = '^'
  for c in pat:
    if c == '*':
      out += '.*'
    else:
      out += re.escape(c)
  out += '$'
  return out
