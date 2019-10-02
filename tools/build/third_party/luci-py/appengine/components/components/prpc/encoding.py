# Copyright 2017 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from google.protobuf import json_format, text_format


class Encoding(object):
  BINARY = (0, 'application/prpc; encoding=binary')
  JSON   = (1, 'application/json')
  TEXT   = (2, 'application/prpc; encoding=text')

  @staticmethod
  def media_type(encoding):
    return encoding[1]


def get_decoder(encoding):
  """Returns the appropriate decoder for content type.

  Args:
    encoding: A value from the Encoding enum.

  Returns:
    a callable which takes an encoded string and an empty protobuf message, and
        populates the given protobuf with data from the string. Each decoder
        may raise exceptions of its own based on incorrectly formatted data.
  """
  if encoding == Encoding.BINARY:
    return lambda string, proto: proto.ParseFromString(string)
  elif encoding == Encoding.JSON:
    return json_format.Parse
  elif encoding == Encoding.TEXT:
    return text_format.Merge
  else:
    assert False, 'Argument |encoding| was not a value of the Encoding enum.'


def get_encoder(encoding):
  """Returns the appropriate encoder for the Accept content type.

  Args:
    encoding: A value from the Encoding enum.

  Returns:
    a callable which takes an initialized protobuf message, and returns a string
        representing its data. Each encoder may raise exceptions of its own.
  """
  if encoding == Encoding.BINARY:
    return lambda proto: proto.SerializeToString()
  elif encoding == Encoding.JSON:
    return lambda proto: ')]}\'\n' + json_format.MessageToJson(proto)
  elif encoding == Encoding.TEXT:
    return lambda proto: text_format.MessageToString(proto, as_utf8=True)
  else:
    assert False, 'Argument |encoding| was not a value of the Encoding enum.'
