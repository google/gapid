# Copyright 2016 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.


class InitializationError(Exception):
  """Raised by RemoteClient.initialize on fatal errors."""
  def __init__(self, last_error):
    super(InitializationError, self).__init__('Failed to grab auth headers')
    self.last_error = last_error


class BotCodeError(Exception):
  """Raised by RemoteClient.get_bot_code."""
  def __init__(self, new_zip, url, version):
    super(BotCodeError, self).__init__(
        'Unable to download %s from %s; first tried version %s' %
        (new_zip, url, version))


class InternalError(Exception):
  """Raised on unrecoverable errors that abort task with 'internal error'."""


class PollError(Exception):
  """Raised on unrecoverable errors in RemoteClient.poll."""


class MintOAuthTokenError(Exception):
  """Raised on unrecoverable errors in RemoteClient.mint_oauth_token."""
