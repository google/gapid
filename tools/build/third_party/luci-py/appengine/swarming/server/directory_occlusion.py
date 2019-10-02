# Copyright 2018 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""directory_occlusion.Checker provides a way to check for conflicts between
multiple competing uses of the filesystem.

Users of the Checker can declare various folders in an in-memory filesystem to
be owned by one agent or another, and then ask the Checker if there are any
conflicting ownerships.

For example, say that folder "a/b" is being used by Alice, but folder
"a/b/c" is being used by Charlie. This is a conflict because Alice may decide
that she wants to use folder "c" for something.

Concretely, this is used to check for conflicts between CIPD and named caches,
but could also (potentially) be extended to declare ownership of other
subdirectories in the task (e.g. in the isolated).
"""


import collections


class Checker(object):
  """A very limited filesystem hierarchy checker.

  This forms a tree, where each node is a directory. Nodes in the tree may have
  a mapping from owner claiming this directory to a series of notes
  (descriptions of /why/ this owner claims this directory).

  Paths may only ever have one owner; After adding all paths to the Checker,
  call .conflicts to populate a validation.Context with any conflicts
  discovered.

  Practically, this is used to ensure that Cache directories do not overlap with
  CIPD package directives; CIPD packages may not be installed as subdirs of
  caches, and caches may not be installed as subdirs of CIPD package
  directories. Similarly, multiple caches cannot be mapped to the same
  directory.
  """
  def __init__(self, full_path=''):
    self._full_path = full_path
    self._owner_notes = collections.defaultdict(set) # owner -> set(notes)
    self._subdirs = {}

  def add(self, path, owner, note):
    """Attaches `note` to `path` with the specified `owner`.

    Args:
      path (str) - a '/'-delimited path to annotate
      owner (str) - The owning entity for this path
      note (str) - A brief description of why `owner` lays claim to `path`.
    """
    tokens = path.split('/')
    node = self
    for i, subdir in enumerate(tokens):
      node = node._subdirs.setdefault(subdir, Checker('/'.join(tokens[:i+1])))
    node._owner_notes[owner].add(note)

  def conflicts(self, ctx):
    """Populates `ctx` with all violations found in this Checker.

    This will walk the Checker depth-first, pruning branches
    at the first conflict.

    Args:
      ctx (validation.Context) - Conflicts found will be reported here.

    Returns a boolean indicating if conflicts were found or not.
    """
    return self._conflicts(ctx, None)

  # internal

  def _conflicts(self, ctx, parent_owned_node):
    """Populates `ctx` with all violations found in this Checker.

    This will walk the Checker depth-first, pruning branches
    at the first conflict.

    Args:
      ctx (validation.Context) - Conflicts found will be reported here.
      parent_owned_node (cls|None) - If set, a node in the Checker tree which
        is some (possibly distant) parent in the filesystem that has an owner
        set.

    Returns a boolean indicating if conflicts were found or not.
    """
    my_owners = self._owner_notes.keys()
    # multiple owners tried to claim this directory. In this case there's no
    # discernable owner for subdirectories, so return True immediately.
    if len(my_owners) > 1:
      ctx.error(
          '%r: directory has conflicting owners: %s',
          self._full_path, ' and '.join(self._descriptions()))
      return True

    # something (singluar) claimed this directory
    if len(my_owners) == 1:
      my_owner = my_owners[0]

      # some directory above us also has an owner set, check for conflicts.
      if parent_owned_node:
        if my_owner != parent_owned_node._owner():
          # We found a conflict; there's no discernible owner for
          # subdirectories, so return True immediately.
          ctx.error(
              '%s uses %r, which conflicts with %s using %r',
              self._describe_one(), self._full_path,
              parent_owned_node._describe_one(), parent_owned_node._full_path)
          return True
      else:
        # we're the first owner down this leg of the tree, so parent_owned_node
        # is now us for all subdirectories.
        parent_owned_node = self

    ret = False
    for _, node in sorted(self._subdirs.iteritems()):
      # call _conflicts() first so that it's not short-circuited.
      ret = node._conflicts(ctx, parent_owned_node) or ret
    return ret

  def _descriptions(self):
    """Formats all the _owner_notes on this node into a list of strings."""
    ret = []
    for owner, notes in sorted(self._owner_notes.iteritems()):
      notes = filter(bool, notes)
      if notes:
        # 'owner[note, note, note]'
        ret.append('%s%r' % (owner, sorted(notes)))
      else:
        ret.append(owner)
    return ret

  def _describe_one(self):
    """Gets the sole description for this node as a string.

    Asserts that there's exactly one owner for this node.
    """
    ret = self._descriptions()
    assert len(ret) == 1
    return ret[0]

  def _owner(self):
    """Gets the sole owner for this node as a string.

    Asserts that there's exactly one owner for this node.
    """
    assert len(self._owner_notes) == 1
    return self._owner_notes.keys()[0]
