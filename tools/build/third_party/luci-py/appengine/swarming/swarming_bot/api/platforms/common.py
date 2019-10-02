# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Common code for platforms."""


def _safe_parse(content, split=': '):
  """Safely parse a 'key: value' list of strings from a command."""
  values = {}
  for l in content.splitlines():
    if not l:
      continue
    parts = l.split(split, 2)
    if len(parts) != 2:
      continue
    values.setdefault(
        parts[0].strip().decode('utf-8'), parts[1].decode('utf-8'))
  return values
