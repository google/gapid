# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import os
import sys


# swarming_bot/
BOT_DIR = os.path.dirname(os.path.realpath(os.path.abspath(__file__)))


def init_symlinks(root):
  """Adds support for symlink-as-file on Windows.

  Manually resolves symlinks in path for directory and add them to sys.path.
  """
  if sys.platform != 'win32':
    return
  for i in os.listdir(root):
    if '.' in i:
      continue
    path = os.path.join(root, i)
    if os.path.isfile(path):
      # Found a file instead of a symlink to a directory. Adjust sys.path
      # accordingly to where the symlink points.
      with open(path) as f:
        link = f.read()
      if '\n' in link:
        continue
      dest = os.path.normpath(os.path.join(root, link))
      # This is not exactly right but close enough.
      sys.path.insert(0, os.path.dirname(dest))


def setup_test_env():
  """Sets up the environment for bot tests."""
  init_symlinks(BOT_DIR)
  client = os.path.normpath(os.path.join(BOT_DIR, '..', '..', '..', 'client'))
  client_tests = os.path.join(client, 'tests')
  sys.path.insert(0, client_tests)

  tp = os.path.join(BOT_DIR, 'third_party')
  if sys.platform == 'win32':
    # third_party is a symlink.
    with open(tp, 'rb') as f:
      tp = os.path.join(BOT_DIR, f.read())
  sys.path.insert(0, tp)

  # libusb1 expects to be directly in sys.path.
  sys.path.insert(0, os.path.join(BOT_DIR, 'python_libusb1'))

  # For python-rsa.
  sys.path.insert(0, os.path.join(tp, 'rsa'))
  sys.path.insert(0, os.path.join(tp, 'pyasn1'))

  # Protobuf is now used in the bot itself.
  # See fix_protobuf_package() in appengine/components/components/utils.py
  # but until this code, the version under client is used.
  if 'google' in sys.modules:
    # It may be in lib/python2.7/site-packages/google, take not chance and flush
    # it out.
    del sys.modules['google']
  # This should import client/third_party/google
  import google
  google_pkg = os.path.join(client, 'third_party', 'google')
  if google_pkg not in google.__path__:
    google.__path__.insert(0, google_pkg)
  six_path = os.path.join(client, 'third_party', 'six')
  if six_path not in sys.path:
    sys.path.insert(0, six_path)
