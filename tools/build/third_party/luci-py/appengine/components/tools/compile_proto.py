#!/usr/bin/env python
# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Compiles all *.proto files it finds into *_pb2.py."""

import logging
import optparse
import os
import re
import shutil
import subprocess
import sys
import tempfile


# Directory with this file.
THIS_DIR = os.path.dirname(os.path.abspath(__file__))


# Minimally required protoc version.
MIN_SUPPORTED_PROTOC_VERSION = (3, 6, 1)
# Maximally supported protoc version.
MAX_SUPPORTED_PROTOC_VERSION = (3, 6, 1)


# Printed if protoc is missing or too old.
PROTOC_INSTALL_HELP = (
    "Could not find working protoc (%s <= ver <= %s) in PATH." %
    (
      '.'.join(map(str, MIN_SUPPORTED_PROTOC_VERSION)),
      '.'.join(map(str, MAX_SUPPORTED_PROTOC_VERSION)),
    ))


# Paths that should not be searched for *.proto.
BLACKLISTED_PATHS = [
  re.compile(r'.*(/|\\)third_party(/|\\)?'),
]


def is_blacklisted(path):
  """True if |path| matches any regexp in |blacklist|."""
  return any(b.match(path) for b in BLACKLISTED_PATHS)


def find_proto_files(path):
  """Recursively searches for *.proto files, yields absolute paths to them."""
  path = os.path.abspath(path)
  for dirpath, dirnames, filenames in os.walk(path, followlinks=True):
    # Skip hidden and blacklisted directories
    skipped = [
      x for x in dirnames
      if x[0] == '.' or is_blacklisted(os.path.join(dirpath, x))
    ]
    for dirname in skipped:
      dirnames.remove(dirname)
    # Yield *.proto files.
    for name in filenames:
      if name.endswith('.proto'):
        yield os.path.join(dirpath, name)


def get_protoc():
  """Returns protoc executable path (maybe relative to PATH)."""
  return 'protoc.exe' if sys.platform == 'win32' else 'protoc'

def compile_proto(proto_file, output_path, proto_path):
  """Invokes 'protoc', compiling single *.proto file into *_pb2.py file.

  Returns:
      The path of the generated _pb2.py file.
  """
  cmd = [get_protoc()]
  proto_path = proto_path or os.path.dirname(proto_file)
  cmd.append('--proto_path=%s' % proto_path)
  cmd.append('--python_out=%s' % output_path)
  cmd.append('--prpc-python_out=%s' % output_path)
  cmd.append(proto_file)
  logging.debug('Running %s', cmd)
  env = os.environ.copy()
  env['PATH'] = os.pathsep.join([THIS_DIR, env.get('PATH', '')])
  # Reuse embedded google protobuf.
  root = os.path.dirname(os.path.dirname(os.path.dirname(THIS_DIR)))
  env['PYTHONPATH'] = os.path.join(root, 'client', 'third_party')
  subprocess.check_call(cmd, env=env)
  return proto_file.replace('.proto', '_pb2.py').replace(proto_path,
                                                         output_path)


def check_proto_compiled(proto_file, proto_path):
  """Return True if *_pb2.py on disk is up to date."""
  # Missing?
  expected_path = proto_file.replace('.proto', '_pb2.py')
  if not os.path.exists(expected_path):
    return False

  # Helper to read contents of a file.
  def read(path):
    with open(path, 'r') as f:
      return f.read()

  # Compile *.proto into temp file to compare the result with existing file.
  tmp_dir = tempfile.mkdtemp()
  try:
    try:
      compiled = compile_proto(proto_file, tmp_dir, proto_path)
    except subprocess.CalledProcessError:
      return False
    return read(compiled) == read(expected_path)
  finally:
    shutil.rmtree(tmp_dir)


def compile_all_files(root_dir, proto_path):
  """Compiles all *.proto files it recursively finds in |root_dir|."""
  root_dir = os.path.abspath(root_dir)
  success = True
  for path in find_proto_files(root_dir):
    try:
      compile_proto(path, os.getcwd(), proto_path)
    except subprocess.CalledProcessError:
      print >> sys.stderr, 'Failed to compile: %s' % path[len(root_dir)+1:]
      success = False
  return success


def check_all_files(root_dir, proto_path):
  """Returns True if all *_pb2.py files on disk are up to date."""
  root_dir = os.path.abspath(root_dir)
  success = True
  for path in find_proto_files(root_dir):
    if not check_proto_compiled(path, proto_path):
      print >> sys.stderr, (
          'Need to recompile file: %s' % path[len(root_dir)+1:])
      success = False
  return success


def get_protoc_version():
  """Returns the version of installed 'protoc', or None if not found."""
  cmd = [get_protoc(), '--version']
  try:
    logging.debug('Running %s', cmd)
    proc = subprocess.Popen(cmd, stdout=subprocess.PIPE)
    out, _ = proc.communicate()
    if proc.returncode:
      logging.debug('protoc --version returned %d', proc.returncode)
      return None
  except OSError as err:
    logging.debug('Failed to run protoc --version: %s', err)
    return None
  match = re.match('libprotoc (.*)', out)
  if not match:
    logging.debug('Unexpected output of protoc --version: %s', out)
    return None
  return tuple(map(int, match.group(1).split('.')))


def main(args, app_dir=None):
  parser = optparse.OptionParser(
      description=sys.modules['__main__'].__doc__,
      usage='%prog [options]' + ('' if app_dir else ' <root dir>'))
  parser.add_option(
      '-c', '--check', action='store_true',
      help='Only check that all *.proto files are up to date')
  parser.add_option('-v', '--verbose', action='store_true')
  parser.add_option(
      '--proto_path',
      help=('Passed through to protoc. If not set, will be set to the directory'
            'containing the proto file being compiled.')
  )

  options, args = parser.parse_args(args)
  logging.basicConfig(level=logging.DEBUG if options.verbose else logging.ERROR)
  root_dir = None
  if not app_dir:
    if len(args) != 1:
      parser.error('Expecting single argument')
    root_dir = args[0]
  else:
    if args:
      parser.error('Unexpected arguments')
    root_dir = app_dir

  # Ensure protoc compiler is up-to-date.
  protoc_version = get_protoc_version()
  if protoc_version is None or protoc_version < MIN_SUPPORTED_PROTOC_VERSION:
    if protoc_version:
      existing = '.'.join(map(str, protoc_version))
      expected = '.'.join(map(str, MIN_SUPPORTED_PROTOC_VERSION))
      print >> sys.stderr, (
          'protoc version is too old (%s), expecting at least %s.\n' %
          (existing, expected))
    sys.stderr.write(PROTOC_INSTALL_HELP)
    return 1
  # Make sure protoc produces code compatible with vendored libprotobuf.
  if protoc_version > MAX_SUPPORTED_PROTOC_VERSION:
    existing = '.'.join(map(str, protoc_version))
    expected = '.'.join(map(str, MAX_SUPPORTED_PROTOC_VERSION))
    print >> sys.stderr, (
        'protoc version is too new (%s), expecting at most %s.\n' %
        (existing, expected))
    sys.stderr.write(PROTOC_INSTALL_HELP)
    return 1

  if options.proto_path:
    options.proto_path = os.path.abspath(options.proto_path)

  if options.check:
    success = check_all_files(root_dir, options.proto_path)
  else:
    success = compile_all_files(root_dir, options.proto_path)

  return int(not success)


if __name__ == '__main__':
  sys.exit(main(sys.argv[1:]))
