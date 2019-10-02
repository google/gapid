# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Implements a singleton."""

import contextlib
import logging
import os
import sys


if sys.platform == 'win32':
  from ctypes import wintypes
else:
  import fcntl


class Singleton(object):
  """Creates an global singleton that can be held by only one process on the
  host.

  On Windows, uses a global Mutex. On others, use a flock'ed file.
  """
  def __init__(self, rootdir):
    rootdir = os.path.realpath(rootdir)
    self.handle = None
    if sys.platform == 'win32':
      # Use the directory name without '\\'. Enforce lowercase.
      self.key = u'Global\\' + unicode(rootdir.replace('\\', '_')).lower()
    else:
      self.key = os.path.join(rootdir, 'swarming.lck')

  def acquire(self):
    """Tries to acquire the singleton.

    Returns:
      True if there was no previous process, False if this process is a
      duplicate and should exit.
    """
    if sys.platform == 'win32':
      # Create a global mutex. Make the mutex so that it disapear automatically
      # when the process dies. The handle is not inherited so task_runner
      # doesn't get to keep it alive.
      # pylint: disable=undefined-variable
      self.handle = wintypes.windll.kernel32.CreateMutexW(
          wintypes.c_int(0),
          wintypes.c_int(-1),
          wintypes.create_unicode_buffer(self.key))
      last_error = wintypes.GetLastError()
      logging.info('%s = %s ; %s', self.key, self.handle, last_error)
      if not self.handle:
        return False
      # ERROR_ALREADY_EXISTS
      if last_error == 183:
        self.release()
      return bool(self.handle)
    else:
      self.handle = open(self.key, 'a+b')
      try:
        fcntl.flock(self.handle, fcntl.LOCK_EX | fcntl.LOCK_NB)
      except IOError:
        # There's a small race conditon where it could report a previous pid.
        logging.error('Singleton held by %s', self.handle.read())
        self.handle.close()
        self.handle = None
        return False
      self.handle.seek(0, os.SEEK_SET)
      self.handle.truncate(0)
      self.handle.write(str(os.getpid()))
      return True

  def release(self):
    """Release the singleton."""
    if not self.handle:
      return
    if sys.platform == 'win32':
      # pylint: disable=undefined-variable
      wintypes.windll.kernel32.CloseHandle(self.handle)
    else:
      self.handle.close()
      try:
        os.remove(self.key)
      except (IOError, OSError):
        pass
    self.handle = None


@contextlib.contextmanager
def singleton(rootdir):
  s = Singleton(rootdir)
  acquired = s.acquire()
  try:
      yield acquired
  finally:
    if acquired:
      s.release()
