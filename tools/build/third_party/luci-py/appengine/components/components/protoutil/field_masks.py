# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utility functions for google.protobuf.field_mask_pb2.FieldMask.

Supports advanced field mask semantics:
- Refer to fields and map keys using . literals:
  - Supported map key types: string, integer types, bool.
  - Floating point (including double and float), enum, and bytes keys are not
    supported by protobuf or this implementation.
  - Fields: 'publisher.name' means field name of field publisher
  - string map keys: 'metadata.year' means string key 'year' of map field
    metadata
  - integer map keys (e.g. int32): 'year_ratings.0' means integer key 0 of a map
    field year_ratings
  - bool map keys: 'access_text.true' means boolean key true of a map field
    access_text
- String map keys that cannot be represented as an unquoted string literal,
  must be quoted using backticks: metadata.`year.published`, metadata.`17`,
  metadata.``. Backtick can be escaped with ``: a.`b``c` means map key "b`c"
  of map field a.
- Refer to all map keys using a * literal: "topics.*.archived" means field
  "archived" of all map values of map field "topic".
- Refer to all elements of a repeated field using a * literal: authors.*.name
- Refer to all fields of a message using * literal: publisher.*.
- Prohibit addressing a single element in repeated fields: authors.0.name

FieldMask.paths string grammar:
  path = segment {'.' segment}
  segment = literal | '*' | quoted_string;
  literal = string | integer | bool
  string = (letter | '_') {letter | '_' | digit}
  integer = ['-'] digit {digit};
  bool = 'true' | 'false';
  quoted_string = '`' { utf8-no-backtick | '``' } '`'

TODO(nodir): replace spec above with a link to a spec when it is available.
"""

from google import protobuf
from google.protobuf import descriptor

__all__ = [
    'EXCLUDE',
    'INCLUDE_ENTIRELY',
    'INCLUDE_PARTIALLY',
    'Mask',
    'STAR',
]


# Used in a parsed path to represent a star segment.
# See Mask docstring.
STAR = object()

EXCLUDE = 0
INCLUDE_PARTIALLY = 1
INCLUDE_ENTIRELY = 2


class Mask(object):
  """A tree representation of a field mask. Serves as a tree node too.

  Each node represents a segment of a paths string, e.g. 'bar' in 'foo.bar.qux'.
  A Field mask with paths ['a', 'b.c'] is parsed as
    <root>
      a
      b
        c

  Attrs:
    desc: a descriptor of the message of the field this node represents.
      If the field type is not a message, then desc is None and the node
      must be a leaf.
    repeated: True means that the segment represents a repeated field, and not
      one of the elements. Children of the node are the field elements.
    children: a dict that maps a segment to its node, e.g. children of the root
      of the example above has keys 'a' and 'b', and values are Mask objects. A
      segment can be of type str, int, bool or it can be the value of
      field_masks.STAR for '*' segments.
  """

  def __init__(self, desc=None, repeated=False, children=None):
    """Initializes the mask.

    The arguments initialize attributes of the same names, see Mask docstring.
    """
    self.desc = desc
    self.repeated = repeated
    self.children = children or {}

  def trim(self, message):
    """Clears message fields that are not in the mask.

    The message must be a google.protobuf.message.Message.
    Uses self.includes to decide what to trim, see its docstring.
    If self is a leaf, this is a noop.
    """
    for f, v in message.ListFields():
      incl = self._includes((f.name,))
      if incl == INCLUDE_ENTIRELY:
        continue

      if incl == EXCLUDE:
        message.ClearField(f.name)
        continue

      assert incl == INCLUDE_PARTIALLY
      # Child for this field must exist because INCLUDE_PARTIALLY.
      child = self.children[f.name]

      if not f.message_type:
        # The field is scalar, but the field mask does not specify to
        # include it entirely. Skip it because scalars do not have
        # subfields. Note that from_field_mask would fail on such a mask
        # because a scalar field cannot be followed by other fields.
        message.ClearField(f.name)
        continue

      # Trim the field value.
      if f.message_type.GetOptions().map_entry:
        for mk, mv in v.items():
          incl = self._includes((f.name, mk))
          if incl == INCLUDE_ENTIRELY:
            pass
          elif incl == EXCLUDE:
            v.pop(mk)
          elif isinstance(mv, protobuf.message.Message):
            assert incl == INCLUDE_PARTIALLY
            # Child for mk must exist because INCLUDE_PARTIALLY.
            child.children[mk].trim(mv)
          else:
            # The field is scalar, see the comment above.
            v.pop(mk)
      elif f.label == descriptor.FieldDescriptor.LABEL_REPEATED:
        star_child = child.children[STAR]
        for rv in v:
          star_child.trim(rv)
      else:
        child.trim(v)

  def includes(self, path):
    """Tells if a field value at the given path must be included.

    Args:
      path: a path string. Must use canonical field names, i.e. not json names.

    Returns:
      EXCLUDE if the field value must be excluded.
      INCLUDE_PARTIALLY if some subfields of the field value must be included.
      INCLUDE_ENTIRELY if the field value must be included entirely.

    Raises:
      ValueError: path is a string and it is invalid according to
        self.desc and self.repeated.
    """
    assert path
    return self._includes(_parse_path(path, self.desc, repeated=self.repeated))

  def _includes(self, path, start_at=0):
    """Implements includes()."""

    if not self.children:
      return INCLUDE_ENTIRELY

    if start_at == len(path):
      # This node is intermediate and we've exhausted the path.
      # Some of the value's subfields are included, so includes this value
      # partially.
      return INCLUDE_PARTIALLY

    # Find children that match current segment.
    seg = path[start_at]
    children = [self.children.get(seg)]
    if seg != STAR:
      # self might have a star child
      # e.g. self is {'a': {'b': {}}, STAR: {'c': {}}}
      # If seg is 'x', we should check the star child.
      children.append(self.children.get(STAR))
    children = [c for c in children if c is not None]
    if not children:
      # Nothing matched.
      return EXCLUDE
    return max(c._includes(path, start_at + 1) for c in children)

  def merge(self, src, dest):
    """Merges masked fields from src to dest.

    Merges even empty/unset fields, as long as they are present in the mask.

    Overwrites repeated/map fields entirely. Does not support partial updates of
    such fields.
    """
    assert isinstance(src, protobuf.message.Message)
    assert type(src) == type(dest)  # pylint: disable=unidiomatic-typecheck

    for f_name, submask in self.children.iteritems():
      include_partially = bool(submask.children)

      dest_value = getattr(dest, f_name)
      src_value = getattr(src, f_name)

      f_desc = dest.DESCRIPTOR.fields_by_name[f_name]
      is_repeated = f_desc.label == descriptor.FieldDescriptor.LABEL_REPEATED
      is_message = f_desc.type == descriptor.FieldDescriptor.TYPE_MESSAGE

      # Only non-repeated submessages can be merged partially.
      if include_partially and is_message and not is_repeated:
        submask.merge(src_value, dest_value)
      # Otherwise overwrite entirely.
      elif is_repeated:
        dest.ClearField(f_name)
        dest_value = getattr(dest, f_name)  # restore after ClearField.
        dest_value.extend(src_value)
      elif is_message:
        dest_value.CopyFrom(src_value)
      else:
        # Scalar value.
        setattr(dest, f_name, src_value)


  def submask(self, path):
    """Returns a sub-mask given a path from self to it.

    For example, for a mask ["a.b.c"], mask.get('a.b') will return a mask with
    c.

    Args:
      path: a path string. Must use canonical field names, i.e. not json names.

    Returns:
      A Mask or None.

    Raises:
      ValueError: path is a string and it is invalid according to
        self.desc and self.repeated.
    """
    assert path
    return self._submask(_parse_path(path, self.desc, repeated=self.repeated))

  def _submask(self, path, start_at=0):
    """Implements submask()."""
    if start_at == len(path):
      return self
    child = self.children.get(path[start_at])
    return child and child._submask(path, start_at + 1)

  @classmethod
  def from_field_mask(
      cls, field_mask, desc, json_names=False, update_mask=False):
    """Parses a field mask to a Mask.

    Removes trailing stars, e.g. parses ['a.*'] as ['a'].
    Removes redundant paths, e.g. parses ['a', 'a.b'] as ['a'].

    Args:
      field_mask: a google.protobuf.field_mask_pb2.FieldMask instance.
      desc: a google.protobuf.descriptor.Descriptor for the target message.
      json_names: True if field_mask uses json field names for field names,
        e.g. "fooBar" instead of "foo_bar".
        Field names will be parsed in the canonical form.
      update_mask: if True, the field_mask is treated as an update mask.
        In an update mask, a repeated field is allowed only as the last
        field in a paths string.

    Raises:
      ValueError if a field path is invalid.
    """
    parsed_paths = []
    for p in field_mask.paths:
      try:
        parsed_paths.append(_parse_path(p, desc, json_names=json_names))
      except ValueError as ex:
        raise ValueError('invalid path "%s": %s' % (p, ex))

    parsed_paths = _normalize_paths(parsed_paths)

    root = cls(desc)
    for i, p in enumerate(parsed_paths):
      node = root
      node_name = ''
      for seg in p:
        if node.repeated and update_mask:
          raise ValueError(
              ('update mask allows a repeated field only at the last '
               'position; field "%s" in "%s" is not last')
              % (node_name, field_mask.paths[i]))
        if seg not in node.children:
          if node.desc.GetOptions().map_entry:
            child = cls(node.desc.fields_by_name['value'].message_type)
          elif node.repeated:
            child = cls(node.desc)
          else:
            field = node.desc.fields_by_name[seg]
            repeated = field.label == descriptor.FieldDescriptor.LABEL_REPEATED
            child = cls(field.message_type, repeated=repeated)
          node.children[seg] = child
        node = node.children[seg]
        node_name = seg
    return root

  def __eq__(self, other):
    """Returns True if other is equivalent to self."""
    return (
        self.desc == other.desc and
        self.repeated == other.repeated and
        self.children == other.children)

  def __ne__(self, other):
    """Returns False if other is equivalent to self."""
    return not (self == other)

  def __repr__(self):
    """Returns a string representation of the Mask."""
    return 'Mask(%r, %r, %r)' % (self.desc, self.repeated, self.children)


def _normalize_paths(paths):
  """Normalizes field paths. Returns a new set of paths.

  paths must be parsed, see _parse_path.

  Removes trailing stars, e.g. convertes ('a', STAR) to ('a',).

  Removes paths that have a segment prefix already present in paths,
  e.g. removes ('a', 'b') from [('a', 'b'), ('a',)].
  """
  paths = _remove_trailing_stars(paths)
  return {
      p for p in paths
      if not any(p[:i] in paths for i in xrange(len(p)))
  }


def _remove_trailing_stars(paths):
  ret = set()
  for p in paths:
    assert isinstance(p, tuple), p
    if p[-1] == STAR:
      p = p[:-1]
    ret.add(p)
  return ret


# Token types.
_STAR, _PERIOD, _LITERAL, _STRING, _INTEGER, _UNKNOWN, _EOF = xrange(7)


_INTEGER_FIELD_TYPES = {
    descriptor.FieldDescriptor.TYPE_INT64,
    descriptor.FieldDescriptor.TYPE_INT32,
    descriptor.FieldDescriptor.TYPE_UINT32,
    descriptor.FieldDescriptor.TYPE_UINT64,
    descriptor.FieldDescriptor.TYPE_FIXED64,
    descriptor.FieldDescriptor.TYPE_FIXED32,
    descriptor.FieldDescriptor.TYPE_SFIXED64,
    descriptor.FieldDescriptor.TYPE_SFIXED32,
}
_SUPPORTED_MAP_KEY_TYPES = _INTEGER_FIELD_TYPES | {
    descriptor.FieldDescriptor.TYPE_STRING,
    descriptor.FieldDescriptor.TYPE_BOOL,
}


def _parse_path(path, desc, repeated=False, json_names=False):
  """Parses a field path to a tuple of segments.

  See grammar in the module docstring.

  Args:
    path: a field path.
    desc: a google.protobuf.descriptor.Descriptor of the target message.
    repeated: True means that desc is a repeated field. For example,
      the target field is a repeated message field and path starts with an
      index.
    json_names: True if path uses json field names for field names,
      e.g. "fooBar" instead of "foo_bar".
      Field names will be parsed in the canonical form.

  Returns:
    A tuple of segments. A star is returned as STAR object.

  Raises:
    ValueError if path is invalid.
  """
  tokens = list(_tokenize(path))
  ctx = _ParseContext(desc, repeated)
  peek = lambda: tokens[ctx.i]

  def read():
    tok = peek()
    ctx.i += 1
    return tok

  def read_path():
    segs = []
    while True:
      seg, must_be_last = read_segment()
      segs.append(seg)

      tok_type, tok = read()
      if tok_type == _EOF:
        break
      if must_be_last:
        raise ValueError('unexpected token "%s"; expected end of string' % tok)
      if tok_type != _PERIOD:
        raise ValueError('unexpected token "%s"; expected a period' % tok)
    return tuple(segs)

  def read_segment():
    """Returns (segment, must_be_last) tuple."""
    tok_type, tok = peek()
    assert tok
    if tok_type == _PERIOD:
      raise ValueError('a segment cannot start with a period')
    if tok_type == _EOF:
      raise ValueError('unexpected end')

    is_map_key = ctx.desc and ctx.desc.GetOptions().map_entry
    if ctx.repeated and not is_map_key:
      if tok_type != _STAR:
        raise ValueError('unexpected token "%s", expected a star' % tok)
      read()  # Swallow star.
      ctx.repeated = False
      return STAR, False

    if ctx.desc is None:
      raise ValueError(
          'scalar field "%s" cannot have subfields' % ctx.field_path)

    if is_map_key:
      key_type = ctx.desc.fields_by_name['key'].type
      if key_type not in _SUPPORTED_MAP_KEY_TYPES:
        raise ValueError(
            'unsupported key type of field "%s"' % ctx.field_path)
      if tok_type == _STAR:
        read()  # Swallow star.
        seg = STAR
      elif key_type == descriptor.FieldDescriptor.TYPE_BOOL:
        seg = read_bool()
      elif key_type in _INTEGER_FIELD_TYPES:
        seg = read_integer()
      else:
        assert key_type == descriptor.FieldDescriptor.TYPE_STRING
        seg = read_string()

      ctx.advance_to_field(ctx.desc.fields_by_name['value'])
      return seg, False

    if tok_type == _STAR:
      # Include all fields.
      read()  # Swallow star.
       # A STAR field cannot be followed by subfields.
      return STAR, True

    if tok_type != _LITERAL:
      raise ValueError(
          'unexpected token "%s"; expected a field name' % tok)
    read()  # Swallow field name.

    field = _find_field(ctx.desc, tok, json_names)
    if field is None:
      raise ValueError(
          'field "%s" does not exist in message %s' % (
              tok, ctx.desc.full_name))
    ctx.advance_to_field(field)
    return field.name, False

  def read_bool():
    tok_type, tok = read()
    if tok_type != _LITERAL or tok not in ('true', 'false'):
      raise ValueError(
          'unexpected token "%s", expected true or false' % tok)
    return tok == 'true'

  def read_integer():
    tok_type, tok = read()
    if tok_type != _INTEGER:
      raise ValueError('unexpected token "%s"; expected an integer' % tok)
    return int(tok)

  def read_string():
    tok_type, tok = read()
    if tok_type not in (_LITERAL, _STRING):
      raise ValueError('unexpected token "%s"; expected a string' % tok)
    return tok

  return read_path()


def _find_field(desc, name, json_name):
  if not json_name:
    return desc.fields_by_name.get(name)
  for f in desc.fields:
    if f.json_name == name:
      return f
  return None


class _ParseContext(object):
  """Context of parsing in _parse_path."""

  def __init__(self, desc, repeated):
    self.i = 0
    self.desc = desc
    self.repeated = repeated
    self._field_path = []  # full path of the current field

  def advance_to_field(self, field):
    """Advances the context to the next message field.

    Args:
      field: a google.protobuf.descriptor.FieldDescriptor to move to.
    """
    self.desc = field.message_type
    self.repeated = field.label == descriptor.FieldDescriptor.LABEL_REPEATED
    self._field_path.append(field.name)

  @property
  def field_path(self):
    return '.'.join(self._field_path)


def _tokenize(path):
  """Transforms path to an iterator of (token_type, string) tuples.

  Raises:
    ValueError if a quoted string is not closed.
  """
  assert isinstance(path, basestring), path

  i = 0

  while i < len(path):
    start = i
    c = path[i]
    i += 1
    if c == '`':
      quoted_string = []  # Parsed quoted string as list of string parts.
      while True:
        next_backtick = path.find('`', i)
        if next_backtick == -1:
          raise ValueError('a quoted string is not closed')

        quoted_string.append(path[i:next_backtick])
        i = next_backtick + 1  # Swallow the discovered backtick.

        escaped_backtick = i < len(path) and path[i] == '`'
        if not escaped_backtick:
          break
        quoted_string.append('`')
        i += 1  # Swallow second backtick.

      yield (_STRING, ''.join(quoted_string))
    elif c == '*':
      yield (_STAR, c)
    elif c == '.':
      yield (_PERIOD, c)
    elif c == '-' or c.isdigit():
      while i < len(path) and path[i].isdigit():
        i += 1
      yield (_INTEGER, path[start:i])
    elif c == '_' or c.isalpha():
      while i < len(path) and (path[i].isalnum() or path[i] == '_'):
        i += 1
      yield (_LITERAL, path[start:i])
    else:
      yield (_UNKNOWN, c)
  yield (_EOF, '<eof>')
