# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Prints stack trace on SIGUSR1 and starts interactive console on SIGUSR2."""

import StringIO
import logging
import code
import traceback
import signal
import sys
import threading
import traceback


def _dump(_sig, frame):
  """Dumps the stack trace of all threads."""
  buf = StringIO.StringIO()
  buf.write('** SIGUSR1 received **\n')
  for t in sorted(threading.enumerate(), key=lambda x: x.name):
    buf.write('%s:\n' % t.name)
    f = sys._current_frames()[t.ident]
    if t == threading.current_thread():
      # Use 'frame' for the current thread to remove this very function from the
      # stack.
      f = frame
    traceback.print_stack(f, file=buf)
  buf.write('** SIGUSR1 end **')
  # Logging as error so that it'll be printed even if logging.basicConfig() is
  # used. Use logging instead of sys.stderr.write() because stderr could be
  # sink-holed and logging redirected to a file.
  logging.error('\n%s', buf.getvalue())


def _debug(_sig, frame):
  """Starts an interactive prompt in the main thread."""
  d = {'_frame': frame}
  d.update(frame.f_globals)
  d.update(frame.f_locals)
  try:
    # Enables arrows to work normally.
    # pylint: disable=unused-variable
    import readline
  except ImportError:
    pass
  msg = 'Signal received : entering python shell.\nTraceback:\n%s' % (
      ''.join(traceback.format_stack(frame)))
  symbols = set(frame.f_locals.keys() + frame.f_globals.keys())
  msg += 'Symbols:\n%s' % '\n'.join('  ' + x for x in sorted(symbols))
  code.InteractiveConsole(d).interact(msg)


def register():
  """Registers an handler to catch SIGUSR1 and print a stack trace."""
  if sys.platform not in ('cygwin', 'win32'):
    signal.signal(signal.SIGUSR1, _dump)
    signal.signal(signal.SIGUSR2, _debug)
