# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Understands .isolated files and can do local operations on them."""

import hashlib
import json
import logging
import os
import re
import stat
import sys

from utils import file_path
from utils import fs
from utils import tools


# Version stored and expected in .isolated files.
ISOLATED_FILE_VERSION = '1.6'


# Chunk size to use when doing disk I/O.
DISK_FILE_CHUNK = 1024 * 1024


# Sadly, hashlib uses 'shaX' instead of the standard 'sha-X' so explicitly
# specify the names here.
SUPPORTED_ALGOS = {
  'sha-1': hashlib.sha1,
  'sha-256': hashlib.sha256,
  'sha-512': hashlib.sha512,
}


# Used for serialization.
SUPPORTED_ALGOS_REVERSE = dict((v, k) for k, v in SUPPORTED_ALGOS.iteritems())


SUPPORTED_FILE_TYPES = ['basic', 'tar']


class IsolatedError(ValueError):
  """Generic failure to load a .isolated file."""
  pass


class MappingError(OSError):
  """Failed to recreate the tree."""
  pass


def is_valid_hash(value, algo):
  """Returns if the value is a valid hash for the corresponding algorithm."""
  size = 2 * algo().digest_size
  return bool(re.match(r'^[a-fA-F0-9]{%d}$' % size, value))


def hash_file(filepath, algo):
  """Calculates the hash of a file without reading it all in memory at once.

  |algo| should be one of hashlib hashing algorithm.
  """
  digest = algo()
  with fs.open(filepath, 'rb') as f:
    while True:
      chunk = f.read(DISK_FILE_CHUNK)
      if not chunk:
        break
      digest.update(chunk)
  return digest.hexdigest()


class IsolatedFile(object):
  """Represents a single parsed .isolated file."""

  def __init__(self, obj_hash, algo):
    """|obj_hash| is really the hash of the file."""
    self.obj_hash = obj_hash
    self.algo = algo

    # Raw data.
    self.data = {}
    # A IsolatedFile instance, one per object in self.includes.
    self.children = []

    # Set once the .isolated file is loaded.
    self._is_loaded = False

  def __repr__(self):
    return 'IsolatedFile(%s, loaded: %s)' % (self.obj_hash, self._is_loaded)

  def load(self, content):
    """Verifies the .isolated file is valid and loads this object with the json
    data.
    """
    logging.debug('IsolatedFile.load(%s)' % self.obj_hash)
    assert not self._is_loaded
    self.data = load_isolated(content, self.algo)
    self.children = [
        IsolatedFile(i, self.algo) for i in self.data.get('includes', [])
    ]
    self._is_loaded = True

  @property
  def is_loaded(self):
    """Returns True if 'load' was already called."""
    return self._is_loaded


def walk_includes(isolated):
  """Walks IsolatedFile include graph and yields IsolatedFile objects.

  Visits root node first, then recursively all children, left to right.
  Not yet loaded nodes are considered childless.
  """
  yield isolated
  for child in isolated.children:
    for x in walk_includes(child):
      yield x


@tools.profile
def _expand_symlinks(indir, relfile):
  """Finds symlinks in relfile.

  Follows symlinks in |relfile|, but treating symlinks that point outside the
  build tree as if they were ordinary directories/files. Returns the final
  symlink-free target and a list of paths to symlinks encountered in the
  process.

  The rule about symlinks outside the build tree is for the benefit of the
  Chromium OS ebuild, which symlinks the output directory to an unrelated path
  in the chroot.

  Fails when a directory loop is detected, although in theory we could support
  that case.

  Arguments:
  - indir: base directory; symlinks in indir are not processed; this is
    the base directory that is considered 'outside of the tree'.
  - relfile: part of the path to expand symlink.

  Returns:
    tuple(relfile, list(symlinks)): relfile is real path of relfile where all
    symlinks were evaluated. symlinks if the chain of symlinks found along the
    way, if any.
  """
  is_directory = relfile.endswith(os.path.sep)
  done = indir
  todo = relfile.strip(os.path.sep)
  symlinks = []

  while todo:
    pre_symlink, symlink, post_symlink = file_path.split_at_symlink(done, todo)
    if not symlink:
      todo = file_path.fix_native_path_case(done, todo)
      done = os.path.join(done, todo)
      break
    symlink_path = os.path.join(done, pre_symlink, symlink)
    post_symlink = post_symlink.lstrip(os.path.sep)
    # readlink doesn't exist on Windows.
    # pylint: disable=E1101
    target = os.path.normpath(os.path.join(done, pre_symlink))
    symlink_target = fs.readlink(symlink_path)
    if os.path.isabs(symlink_target):
      # Absolute path are considered a normal directories. The use case is
      # generally someone who puts the output directory on a separate drive.
      target = symlink_target
    else:
      # The symlink itself could be using the wrong path case.
      target = file_path.fix_native_path_case(target, symlink_target)

    if not fs.exists(target):
      raise MappingError(
          'Symlink target doesn\'t exist: %s -> %s' % (symlink_path, target))
    target = file_path.get_native_path_case(target)
    if not file_path.path_starts_with(indir, target):
      done = symlink_path
      todo = post_symlink
      continue
    if file_path.path_starts_with(target, symlink_path):
      raise MappingError(
          'Can\'t map recursive symlink reference %s -> %s' %
          (symlink_path, target))
    logging.info('Found symlink: %s -> %s', symlink_path, target)
    symlinks.append(os.path.relpath(symlink_path, indir))
    # Treat the common prefix of the old and new paths as done, and start
    # scanning again.
    target = target.split(os.path.sep)
    symlink_path = symlink_path.split(os.path.sep)
    prefix_length = 0
    for target_piece, symlink_path_piece in zip(target, symlink_path):
      if target_piece != symlink_path_piece:
        break
      prefix_length += 1
    done = os.path.sep.join(target[:prefix_length])
    todo = os.path.join(
        os.path.sep.join(target[prefix_length:]), post_symlink)

  relfile = os.path.relpath(done, indir)
  relfile = relfile.rstrip(os.path.sep) + is_directory * os.path.sep
  return relfile, symlinks


@tools.profile
def expand_directory_and_symlink(indir, relfile, blacklist, follow_symlinks):
  """Expands a single input. It can result in multiple outputs.

  This function is recursive when relfile is a directory.

  Note: this code doesn't properly handle recursive symlink like one created
  with:
    ln -s .. foo

  Yields:
    tuple(Relative path, bool is_symlink) to files and symlinks inside |indir|.
  """
  if os.path.isabs(relfile):
    raise MappingError(u'Can\'t map absolute path %s' % relfile)

  infile = file_path.normpath(os.path.join(indir, relfile))
  if not infile.startswith(indir):
    raise MappingError(u'Can\'t map file %s outside %s' % (infile, indir))

  filepath = os.path.join(indir, relfile)
  native_filepath = file_path.get_native_path_case(filepath)
  if filepath != native_filepath:
    # Special case './'.
    if filepath != native_filepath + u'.' + os.path.sep:
      # While it'd be nice to enforce path casing on Windows, it's impractical.
      # Also give up enforcing strict path case on OSX. Really, it's that sad.
      # The case where it happens is very specific and hard to reproduce:
      # get_native_path_case(
      #    u'Foo.framework/Versions/A/Resources/Something.nib') will return
      # u'Foo.framework/Versions/A/resources/Something.nib', e.g. lowercase 'r'.
      #
      # Note that this is really something deep in OSX because running
      # ls Foo.framework/Versions/A
      # will print out 'Resources', while file_path.get_native_path_case()
      # returns a lower case 'r'.
      #
      # So *something* is happening under the hood resulting in the command 'ls'
      # and Carbon.File.FSPathMakeRef('path').FSRefMakePath() to disagree.  We
      # have no idea why.
      if sys.platform not in ('darwin', 'win32'):
        raise MappingError(
            u'File path doesn\'t equal native file path\n%s != %s' %
            (filepath, native_filepath))

  symlinks = []
  if follow_symlinks:
    try:
      relfile, symlinks = _expand_symlinks(indir, relfile)
    except OSError:
      # The file doesn't exist, it will throw below.
      pass

  # The symlinks need to be mapped in.
  for s in symlinks:
    yield s, True

  if relfile.endswith(os.path.sep):
    if not fs.isdir(infile):
      raise MappingError(
          u'%s is not a directory but ends with "%s"' % (infile, os.path.sep))

    # Special case './'.
    if relfile.startswith(u'.' + os.path.sep):
      relfile = relfile[2:]
    try:
      for filename in fs.listdir(infile):
        inner_relfile = os.path.join(relfile, filename)
        if blacklist and blacklist(inner_relfile):
          continue
        if fs.isdir(os.path.join(indir, inner_relfile)):
          inner_relfile += os.path.sep
        # Apply recursively.
        for i, is_symlink in expand_directory_and_symlink(
            indir, inner_relfile, blacklist, follow_symlinks):
          yield i, is_symlink
    except OSError as e:
      raise MappingError(
          u'Unable to iterate over directory %s.\n%s' % (infile, e))
  else:
    # Always add individual files even if they were blacklisted.
    if fs.isdir(infile):
      raise MappingError(
          u'Input directory %s must have a trailing slash' % infile)

    if not fs.isfile(infile):
      raise MappingError(u'Input file %s doesn\'t exist' % infile)

    yield relfile, False


@tools.profile
def file_to_metadata(filepath, read_only, collapse_symlinks):
  """Processes an input file, a dependency, and return meta data about it.

  Behaviors:
  - Retrieves the file mode, file size, file timestamp, file link
    destination if it is a file link and calcultate the SHA-1 of the file's
    content if the path points to a file and not a symlink.

  Arguments:
    filepath: File to act on.
    read_only: If 1 or 2, the file mode is manipulated. In practice, only save
               one of 4 modes: 0755 (rwx), 0644 (rw), 0555 (rx), 0444 (r). On
               windows, mode is not set since all files are 'executable' by
               default.
    collapse_symlinks: True if symlinked files should be treated like they were
                       the normal underlying file.

  Returns:
    The necessary dict to create a entry in the 'files' section of an .isolated
    file *except* 'h' for files.
  """
  # TODO(maruel): None is not a valid value.
  assert read_only in (None, 0, 1, 2), read_only
  out = {}
  # Always check the file stat and check if it is a link.
  try:
    if collapse_symlinks:
      # os.stat follows symbolic links
      filestats = fs.stat(filepath)
    else:
      # os.lstat does not follow symbolic links, and thus preserves them.
      filestats = fs.lstat(filepath)
  except OSError:
    # The file is not present.
    raise MappingError('%s is missing' % filepath)
  is_link = stat.S_ISLNK(filestats.st_mode)

  if sys.platform != 'win32':
    # Ignore file mode on Windows since it's not really useful there.
    filemode = stat.S_IMODE(filestats.st_mode)
    # Remove write access for group and all access to 'others'.
    filemode &= ~(stat.S_IWGRP | stat.S_IRWXO)
    if read_only:
      filemode &= ~stat.S_IWUSR
    if filemode & (stat.S_IXUSR|stat.S_IRGRP) == (stat.S_IXUSR|stat.S_IRGRP):
      # Only keep x group bit if both x user bit and group read bit are set.
      filemode |= stat.S_IXGRP
    else:
      filemode &= ~stat.S_IXGRP
    if not is_link:
      out['m'] = filemode

  if not is_link:
    out['s'] = filestats.st_size
  else:
    # The link could be in an incorrect path case. In practice, this only
    # happens on macOS on case insensitive HFS.
    # TODO(maruel): It'd be better if it was only done once, in
    # expand_directory_and_symlink(), so it would not be necessary to do again
    # here.
    symlink_value = fs.readlink(filepath)  # pylint: disable=no-member
    filedir = file_path.get_native_path_case(os.path.dirname(filepath))
    native_dest = file_path.fix_native_path_case(filedir, symlink_value)
    out['l'] = os.path.relpath(native_dest, filedir)
  return out


def save_isolated(isolated, data):
  """Writes one or multiple .isolated files.

  Note: this reference implementation does not create child .isolated file so it
  always returns an empty list.
  """
  # Make sure the data is valid .isolated data by 'reloading' it.
  algo = SUPPORTED_ALGOS[data['algo']]
  load_isolated(json.dumps(data), algo)
  tools.write_json(isolated, data, True)


def split_path(path):
  """Splits a path and return a list with each element."""
  out = []
  while path:
    path, rest = os.path.split(path)
    if rest:
      out.append(rest)
  return out


def load_isolated(content, algo):
  """Verifies the .isolated file is valid and loads this object with the json
  data.

  Arguments:
  - content: raw serialized content to load.
  - algo: hashlib algorithm class. Used to confirm the algorithm matches the
          algorithm used on the Isolate Server.
  """
  if not algo:
    raise IsolatedError('\'algo\' is required')
  try:
    data = json.loads(content)
  except ValueError as v:
    logging.error('Failed to parse .isolated file:\n%s', content)
    raise IsolatedError('Failed to parse (%s): %s...' % (v, content[:100]))

  if not isinstance(data, dict):
    raise IsolatedError('Expected dict, got %r' % data)

  # Check 'version' first, since it could modify the parsing after.
  value = data.get('version', '1.0')
  if not isinstance(value, basestring):
    raise IsolatedError('Expected string, got %r' % value)
  try:
    version = tuple(map(int, value.split('.')))
  except ValueError:
    raise IsolatedError('Expected valid version, got %r' % value)

  expected_version = tuple(map(int, ISOLATED_FILE_VERSION.split('.')))
  # Major version must match.
  if version[0] != expected_version[0]:
    raise IsolatedError(
        'Expected compatible \'%s\' version, got %r' %
        (ISOLATED_FILE_VERSION, value))

  algo_name = SUPPORTED_ALGOS_REVERSE[algo]

  for key, value in data.iteritems():
    if key == 'algo':
      if not isinstance(value, basestring):
        raise IsolatedError('Expected string, got %r' % value)
      if value not in SUPPORTED_ALGOS:
        raise IsolatedError(
            'Expected one of \'%s\', got %r' %
            (', '.join(sorted(SUPPORTED_ALGOS)), value))
      if value != SUPPORTED_ALGOS_REVERSE[algo]:
        raise IsolatedError(
            'Expected \'%s\', got %r' % (SUPPORTED_ALGOS_REVERSE[algo], value))

    elif key == 'command':
      if not isinstance(value, list):
        raise IsolatedError('Expected list, got %r' % value)
      if not value:
        raise IsolatedError('Expected non-empty command')
      for subvalue in value:
        if not isinstance(subvalue, basestring):
          raise IsolatedError('Expected string, got %r' % subvalue)

    elif key == 'files':
      if not isinstance(value, dict):
        raise IsolatedError('Expected dict, got %r' % value)
      for subkey, subvalue in value.iteritems():
        if not isinstance(subkey, basestring):
          raise IsolatedError('Expected string, got %r' % subkey)
        if os.path.isabs(subkey) or subkey.startswith('\\\\'):
          # Disallow '\\\\', it could UNC on Windows but disallow this
          # everywhere.
          raise IsolatedError('File path can\'t be absolute: %r' % subkey)
        if subkey.endswith(('/', '\\')):
          raise IsolatedError(
              'File path can\'t end with \'%s\': %r' % (subkey[-1], subkey))
        if '..' in split_path(subkey):
          raise IsolatedError('File path can\'t reference parent: %r' % subkey)
        if not isinstance(subvalue, dict):
          raise IsolatedError('Expected dict, got %r' % subvalue)
        for subsubkey, subsubvalue in subvalue.iteritems():
          if subsubkey == 'l':
            if not isinstance(subsubvalue, basestring):
              raise IsolatedError('Expected string, got %r' % subsubvalue)
          elif subsubkey == 'm':
            if not isinstance(subsubvalue, int):
              raise IsolatedError('Expected int, got %r' % subsubvalue)
          elif subsubkey == 'h':
            if not is_valid_hash(subsubvalue, algo):
              raise IsolatedError('Expected %s, got %r' %
                                  (algo_name, subsubvalue))
          elif subsubkey == 's':
            if not isinstance(subsubvalue, (int, long)):
              raise IsolatedError('Expected int or long, got %r' % subsubvalue)
          elif subsubkey == 't':
            if subsubvalue not in SUPPORTED_FILE_TYPES:
              raise IsolatedError('Expected one of \'%s\', got %r' % (
                  ', '.join(sorted(SUPPORTED_FILE_TYPES)), subsubvalue))
          else:
            raise IsolatedError('Unknown subsubkey %s' % subsubkey)
        if bool('h' in subvalue) == bool('l' in subvalue):
          raise IsolatedError(
              'Need only one of \'h\' (%s) or \'l\' (link), got: %r' %
              (algo_name, subvalue))
        if bool('h' in subvalue) != bool('s' in subvalue):
          raise IsolatedError(
              'Both \'h\' (%s) and \'s\' (size) should be set, got: %r' %
              (algo_name, subvalue))
        if bool('s' in subvalue) == bool('l' in subvalue):
          raise IsolatedError(
              'Need only one of \'s\' (size) or \'l\' (link), got: %r' %
              subvalue)
        if bool('l' in subvalue) and bool('m' in subvalue):
          raise IsolatedError(
              'Cannot use \'m\' (mode) and \'l\' (link), got: %r' %
              subvalue)

    elif key == 'includes':
      if not isinstance(value, list):
        raise IsolatedError('Expected list, got %r' % value)
      if not value:
        raise IsolatedError('Expected non-empty includes list')
      for subvalue in value:
        if not is_valid_hash(subvalue, algo):
          raise IsolatedError('Expected %s, got %r' % (algo_name, subvalue))

    elif key == 'os':
      if version >= (1, 4):
        raise IsolatedError('Key \'os\' is not allowed starting version 1.4')

    elif key == 'read_only':
      if not value in (0, 1, 2):
        raise IsolatedError('Expected 0, 1 or 2, got %r' % value)

    elif key == 'relative_cwd':
      if not isinstance(value, basestring):
        raise IsolatedError('Expected string, got %r' % value)

    elif key == 'version':
      # Already checked above.
      pass

    else:
      raise IsolatedError('Unknown key %r' % key)

  # Automatically fix os.path.sep if necessary. While .isolated files are always
  # in the native path format, someone could want to download an .isolated tree
  # from another OS.
  wrong_path_sep = '/' if os.path.sep == '\\' else '\\'
  if 'files' in data:
    data['files'] = dict(
        (k.replace(wrong_path_sep, os.path.sep), v)
        for k, v in data['files'].iteritems())
    for v in data['files'].itervalues():
      if 'l' in v:
        v['l'] = v['l'].replace(wrong_path_sep, os.path.sep)
  if 'relative_cwd' in data:
    data['relative_cwd'] = data['relative_cwd'].replace(
        wrong_path_sep, os.path.sep)
  return data
