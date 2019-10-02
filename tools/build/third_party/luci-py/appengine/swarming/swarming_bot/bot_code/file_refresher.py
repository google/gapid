# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

import json
import logging
import Queue
import threading

from utils import file_path


class FileRefresherThread(object):
  """Represents a thread that periodically dumps result of a callback to a file.

  Used by bot_main to send authentication headers to task_runner. task_runner
  reads them from the file when making HTTP calls.

  Uses JSON for serialization. Doesn't delete the file when stopped.

  The instance is not reusable (i.e. once stopped, cannot be started again).
  """

  def __init__(self, path, producer_callback, interval_sec=15):
    self._path = path
    self._producer_callback = producer_callback
    self._interval_sec = interval_sec
    self._thread = None
    self._signal = Queue.Queue()
    self._last_dumped_blob = None

  def start(self):
    """Starts a thread that dumps value to the file."""
    assert self._thread is None
    self._dump() # initial dump
    self._thread = threading.Thread(
        target=self._run, name='FileRefresherThread %s' % self._path)
    self._thread.daemon = True
    self._thread.start()

  def stop(self):
    """Stops the dumping thread (if it is running)."""
    if not self._thread:
      return
    self._signal.put(None)
    self._thread.join(60) # don't wait forever
    if self._thread.is_alive():
      logging.error('FileRefresherThread failed to terminate in time')

  def _dump(self):
    """Attempts to rewrite the file, retrying a bunch of times.

    Returns:
      True to carry on, False to exit the thread.
    """
    try:
      blob = json.dumps(
          self._producer_callback(),
          sort_keys=True,
          indent=2,
          separators=(',', ': '))
    except Exception:
      logging.exception('Unexpected exception in the callback')
      return True
    if blob == self._last_dumped_blob:
      return True # already have it on disk
    logging.info('Updating %s', self._path)

    # On Windows the file may be locked by reading process. Don't freak out,
    # just retry a bit later.
    attempts = 100
    while True:
      try:
        file_path.atomic_replace(self._path, blob)
        self._last_dumped_blob = blob
        return True # success!
      except (IOError, OSError) as e:
        logging.error('Failed to update the file: %s', e)
      if not attempts:
        logging.error(
            'Failed to update the file %s after many attempts, giving up',
            self._path)
        return True
      attempts -= 1
      if not self._wait(0.05):
        return False

  def _wait(self, timeout):
    """Waits for the given duration or until the stop signal.

    Returns:
      True if waited, False if received the stop signal.
    """
    try:
      self._signal.get(timeout=timeout)
      return False
    except Queue.Empty:
      return True

  def _run(self):
    while self._wait(self._interval_sec) and self._dump():
      pass
