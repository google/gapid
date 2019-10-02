# Copyright 2014 Google Inc. All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Common exceptions for ADB and Fastboot."""


class CommonUsbError(IOError):
  """Base class for usb communication errors.

  Inherits from IOError since they are all related to I/O.
  """


class FormatMessageWithArgumentsException(CommonUsbError):
  """Exception that both looks good and is functional.

  Okay, not that kind of functional, it's still a class.

  This interpolates the message with the given arguments to make it
  human-readable, but keeps the arguments in case other code try-excepts it.
  """

  def __init__(self, message, *args):
    message %= args
    super(FormatMessageWithArgumentsException, self).__init__(message, *args)


class DeviceNotFoundError(FormatMessageWithArgumentsException):
  """Device isn't on USB."""


class DeviceAuthError(FormatMessageWithArgumentsException):
  """Device authentication failed."""


class LibusbWrappingError(CommonUsbError):
  """Wraps libusb1 errors while keeping its original usefulness.

  Attributes:
    usb_error: Instance of libusb1.USBError
  """

  def __init__(self, msg, usb_error):
    super(LibusbWrappingError, self).__init__(msg)
    self.usb_error = usb_error

  def __str__(self):  # pragma: no cover
    return '%s: %s' % (
        super(LibusbWrappingError, self).__str__(), repr(self.usb_error))


class WriteFailedError(LibusbWrappingError):
  """Raised when the device doesn't accept our command."""


class ReadFailedError(LibusbWrappingError):
  """Raised when the device doesn't respond to our commands."""


class AdbCommandFailureException(Exception):
  """ADB Command returned a FAIL.

  This is an high level application error."""
