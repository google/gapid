# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Utilities."""

import logging
import os
import signal
import sys

from utils import subprocess42


def exec_python(args):
  """Executes a python process, replacing the current process if possible.

  On Windows, it returns the child process code. The caller must exit at the
  earliest opportunity.
  """
  cmd = [sys.executable] + args
  if sys.platform not in ('cygwin', 'win32'):
    os.execv(cmd[0], cmd)
    return 1

  try:
    # On Windows, we cannot sanely exec() so shell out the child process
    # instead. But we need to forward any signal received that the bot may care
    # about. This means processes accumulate, sadly.
    # TODO(maruel): If stdin closes, it tells the child process that the parent
    # process died.
    proc = subprocess42.Popen(cmd, detached=True, stdin=subprocess42.PIPE)
    def handler(sig, _):
      logging.info('Got signal %s', sig)
      # Always send SIGTERM, which is properly translated.
      proc.send_signal(signal.SIGTERM)

    sig = signal.SIGBREAK if sys.platform == 'win32' else signal.SIGTERM
    with subprocess42.set_signal_handler([sig], handler):
      proc.wait()
      return proc.returncode
  except Exception as e:
    logging.exception('failed to start: %s', e)
    # Swallow the exception.
    return 1
