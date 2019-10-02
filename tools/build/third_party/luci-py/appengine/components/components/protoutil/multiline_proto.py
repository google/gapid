# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Allow multiline strings in text protobuf format.

Usage:

  text_proto = protoutil.parse_multiline(multiline_text_proto)
  protobuf.text_format.Merge(text_proto, msg)

`parse_multiline` looks for bash-style heredocs and replaces them with
single-line text-proto-escaped strings.

Example:
  this: <<EOF
    would
      turn \ninto
      a "single"
    line
  EOF

Turns into the following:
  this: "would\nturn \\ninto\n  a \"single\"\nline"

The format must be compatible with
https://github.com/luci/luci-go/blob/master/common/proto/multiline.go

In particular, the inner lines will be treated with `textwrap.dedent`;
any common leading whitespace that occurs on every line will be removed.
Although both tabs and spaces count as whitespace, they are not equivalent (i.e.
only exactly-matching whitespace prefixes count).
"""

import re
import textwrap


_START_RE = re.compile(r'^(.*)<<\s*([_a-zA-Z]+)\s*$')
_SPACE_RE = re.compile('^(\s*)')
_END_RE_FMT = r'^\s*%s\s*$'


class MultilineParseError(Exception):
  pass


def parse_multiline(content):
  """Parses a multiline text proto and turns it into a single-line one.

  Raises an exception is if there's an open heredoc without a
  matching close marker.

  Args:
    content (str): multiline text proto.
  Returns:
    equivalent single-line text proto string.
  """
  terminator = ''
  terminator_re = None
  lines = []
  prefix = ''
  multiline_parts = []

  if isinstance(content, unicode):
    content = content.encode('utf-8')

  for line in content.split('\n'):
    if not terminator:
      m = _START_RE.match(line)
      if not m:
        lines.append(line)
        continue
      prefix = m.group(1)
      terminator = m.group(2)
      terminator_re = re.compile(_END_RE_FMT % re.escape(terminator))
      continue

    if terminator_re.match(line):
      single_line = _escape_line(textwrap.dedent('\n'.join(multiline_parts)))
      lines.append(prefix + '"' + single_line + '"')
      terminator = ''
      terminator_re = None
      multiline_parts = []
      continue

    multiline_parts.append(line)

  if terminator:
    raise MultilineParseError(
        'Unterminated multiline sequence; terminator = %r' % terminator)

  return '\n'.join(lines)


def _escape_char(c):
  subst = {
      '\n': r'\n',
      '\r': r'\r',
      '\t': r'\t',
      '"':  r'\"',
      '\\': r'\\',
  }
  res = subst.get(c)
  if res:
    return res
  # Check if c is directly printable.
  if 0x20 <= ord(c) and ord(c) <= 0x7f:
    return c
  return r'\%03o' % ord(c)


def _escape_line(s):
  return str(''.join(_escape_char(c) for c in s))
