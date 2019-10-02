# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Defines `Context` class, convenient for customizable validation.

Usage:

    def validate(value, ctx=None):
      ctx = ctx or Context.raise_on_error()
      if value['foo'] < 0:
        ctx.error('foo is too low: %d', value['foo'])
      if len(value['bar']) == 0:
        ctx.warning('bar is empty')

    validate({'foo': -1, 'bar': []})  # Will raise ValueError with message.

    ctx = validation.Context.logging()
    validate({'foo': -1, 'bar': []}, ctx)  # Does not raise, but logs messages.

    ctx = validation.Context()  # Generic context
    validate({'foo': -1, 'bar': []}, ctx)
    result = ctx.result()
    # result is an instance of Result. It contains messages with text and
    # severity level.
"""

import collections
import contextlib
import logging

__all__ = [
  'Context',
  'Message',
  'Result',
]

Message = collections.namedtuple('Message', ['text', 'severity'])
ResultBase = collections.namedtuple('ResultBase', ['messages'])


class Result(ResultBase):
  def _has_level(self, severity):
    return any(m.severity >= severity for m in self.messages)

  @property
  def has_errors(self):
    return self._has_level(logging.ERROR)


class Context(object):
  """Collects validation messages.

  A validation message has:
    text (str): text for humans.
    severity (int): severity level. Use standard logging levels.
  """

  def __init__(self, on_message=None):
    """Initializes a new Context.

    Args:
      on_message: a function that is called whenever a message is emitted.
    """
    assert on_message is None or hasattr(on_message, '__call__'), on_message
    self.messages = []
    self.on_message = on_message
    self.prefixes = ['']

  @contextlib.contextmanager
  def prefix(self, prefix, *args):
    """Adds a prefix to prepend to error messages.

    If called multiple times, prefixes appear in messages in the order this
    method is called.

    Usage:
      with ctx.prefix('foo: '):
        if foo < 0:
          ctx.error('less than zero')  # 'foo: less than zero'
        if foo % 2 == 1:
          ctx.error('must be an even number')  # 'foo: must be an even number'
    """
    new_prefix = '%s%s' % (self.prefixes[-1], prefix % args)
    self.prefixes.append(new_prefix)
    try:
      yield
    finally:
      assert self.prefixes.pop() == new_prefix

  def msg(self, severity, text, *args):
    """Emits a validation message.

    Args:
      severity (int): severity level. Use standard logging levels.
      text (basestring): message text for humans.
      *args: format args for text, like in logging.info.
    """
    assert isinstance(severity, int), severity
    assert isinstance(text, basestring), text
    assert severity >= 0, severity
    msg = Message(
        severity=severity,
        text='%s%s' % (self.prefixes[-1], text % args),
    )
    self.messages.append(msg)
    if self.on_message:
      self.on_message(msg)

  def debug(self, *args, **kwargs):
    """Emits a debug message."""
    self.msg(logging.DEBUG, *args, **kwargs)

  def info(self, *args, **kwargs):
    """Emits an info message."""
    self.msg(logging.INFO, *args, **kwargs)

  def warning(self, *args, **kwargs):
    """Emits a warning message."""
    self.msg(logging.WARNING, *args, **kwargs)

  def error(self, *args, **kwargs):
    """Emits an error message."""
    self.msg(logging.ERROR, *args, **kwargs)

  def critical(self, *args, **kwargs):
    """Emits an critical message."""
    self.msg(logging.CRITICAL, *args, **kwargs)

  def result(self):
    """Returns an instance of Result with a copy of the context state."""
    return Result(self.messages[:])

  @classmethod
  def raise_on_error(cls, prefix=None, exc_type=None):
    """Returns a context that raises an exception on a first error message.

    Args:
      exc_type (type): exception type to raise. Defaults to ValueError.
    """
    exc_type = exc_type or ValueError
    def on_message(msg):
      if msg.severity >= logging.ERROR:
        text = msg.text
        if prefix:
          text = '%s%s' % (prefix, text)
        raise exc_type(text)
    return cls(on_message=on_message)

  @classmethod
  def logging(cls, logger=None):
    """Returns a context that logs all messages."""
    logger = logger or logging.getLogger()
    on_message = lambda msg: logger.log(msg.severity, msg.text)
    return cls(on_message=on_message)
