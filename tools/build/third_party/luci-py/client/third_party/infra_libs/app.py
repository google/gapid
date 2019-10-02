# Copyright 2015 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import argparse
import datetime
import logging
import os
import sys

import psutil

from infra_libs import logs
from infra_libs import ts_mon


class BaseApplication(object):
  """Encapsulates common boilerplate for setting up an application.

  Subclasses must implement the main() method, and will usually also implement
  add_argparse_options().

  By default this will initialise logging and timeseries monitoring (ts_mon)
  modules.

  Minimal example::

    from infra_libs import app

    class MyApplication(app.BaseApplication):
      def main(self, opts):
        # Do stuff.

    if __name__ == '__main__':
      MyApplication().run()

  Class variables (override these in your class definition):
    PROG_NAME: The program name to display in the --help message.  Defaults to
               sys.argv[0].  Passed to argparse.ArgumentParser.
    DESCRIPTION: Text to display in the --help message.  Passed to
                 argparse.ArgumentParser.
    USES_STANDARD_LOGGING: Whether to configure the standard logging libraries.
                           Defaults to True.
    USES_TS_MON: Whether to configure timeseries monitoring.  Defaults to True.

  Instance variables (use these in your application):
    opts: The argparse.Namespace containing parsed commandline arguments.
  """

  PROG_NAME = None
  DESCRIPTION = None
  USES_STANDARD_LOGGING = True
  USES_TS_MON = True

  def __init__(self):
    self.opts = None
    self.parser = None

  def add_argparse_options(self, parser):
    """Register any arguments used by this application.

    Override this method and call parser.add_argument().

    Args:
      parser: An argparse.ArgumentParser object.
    """

    if self.USES_STANDARD_LOGGING:
      logs.add_argparse_options(parser)
    if self.USES_TS_MON:
      ts_mon.add_argparse_options(parser)

  def process_argparse_options(self, options):
    """Process any commandline arguments.

    Args:
      options: An argparse.Namespace object.
    """

    if self.USES_STANDARD_LOGGING:
      logs.process_argparse_options(options)
    if self.USES_TS_MON:
      ts_mon.process_argparse_options(options)

  def main(self, opts):
    """Your application's main method.

    Do the work of your application here.  When this method returns the
    application will exit.

    Args:
      opts: An argparse.Namespace containing parsed commandline options.  This
        is passed as an argument for convenience but is also accessible as an
        instance variable (self.opts).

    Return:
      An integer exit status, or None to use an exit status of 0.
    """
    raise NotImplementedError

  def run(self, args=None):
    """Main application entry point."""

    if args is None:  # pragma: no cover
      args = sys.argv

    # Add and parse commandline args.
    self.parser = argparse.ArgumentParser(
        description=self.DESCRIPTION,
        prog=self.PROG_NAME or args[0],
        formatter_class=argparse.RawTextHelpFormatter)

    self.add_argparse_options(self.parser)
    self.opts = self.parser.parse_args(args[1:])
    self.process_argparse_options(self.opts)

    # Print a startup log message.
    logging.info('Process started at %s', datetime.datetime.utcfromtimestamp(
        psutil.Process().create_time()).isoformat())
    logging.info('Command line arguments:')
    for index, arg in enumerate(sys.argv):
      logging.info('argv[%d]: %s', index, arg)
    logging.info('Process id %d', os.getpid())
    logging.info('Current working directory %s', os.getcwd())

    # Run the application's main function.
    try:
      status = self.main(self.opts)
    except Exception:
      logging.exception('Uncaught exception, exiting:')
      if self.USES_TS_MON:
        # Flushing ts_mon to try to report the exception.
        ts_mon.flush()
      status = 1

    sys.exit(status)
