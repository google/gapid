# Copyright 2017 The Chromium Authors. All rights reserved.
# Use of this source code is governed by a BSD-style license that can be
# found in the LICENSE file.

import logging
import os
import threading
import time

from google.protobuf import duration_pb2
from google.protobuf import json_format
from google.protobuf import message as message_pb
from google.protobuf import struct_pb2
from google.protobuf import timestamp_pb2
# This module is used as a symlink in buildbucket GAE app.
# Do not add import packages not available on GAE.

_BATCH_DEFAULT = 500
_BATCH_LIMIT = 10000

# OAuth 2.0 scope to insert rows.
INSERT_ROWS_SCOPE = 'https://www.googleapis.com/auth/bigquery.insertdata'


def message_to_dict(msg):
  """Converts a protobuf message to a dict, with field names as keys.

  The conversion follows the rules described in
  https://godoc.org/go.chromium.org/luci/tools/cmd/bqschemaupdater
  Also omits Nones and empty lists values.

  Args:
    msg: an instance of google.protobuf.message.Message.

  Returns:
    A dict with BQ-compatible fields. If there are no BQ-compatible fields,
    returns None.
  """
  row = {}
  for f in msg.DESCRIPTOR.fields:
    if f.message_type:
      if _is_empty_message_type(f.message_type):
        # Omit message fields that would result in RECORD fields with no fields.
        continue
      if f.label != f.LABEL_REPEATED and not msg.HasField(f.name):
        # Omit non-repeated message fields that we don't have.
        continue

    val = getattr(msg, f.name)
    if f.label == f.LABEL_REPEATED:
      if val:  # Omit empty arrays.
        row[f.name] = [_to_bq_value(elem, f) for elem in val]
    else:
      bq_value = _to_bq_value(val, f)
      if bq_value is not None:  # Omit NULL values.
        row[f.name] = bq_value
  return row or None  # return None if there are no fields.


def _to_bq_value(value, field_desc):
  if field_desc.enum_type:
    # Enums are stored as strings.
    enum_val = field_desc.enum_type.values_by_number.get(value)
    if not enum_val:
      raise ValueError('Invalid value %r for enum type %s' % (
          value, field_desc.enum_type.full_name))
    return enum_val.name
  elif isinstance(value, duration_pb2.Duration):
    return value.ToTimedelta().total_seconds()
  elif isinstance(value, struct_pb2.Struct):
    # Structs are stored as JSONPB strings,
    # see https://bit.ly/chromium-bq-struct
    return json_format.MessageToJson(value)
  elif isinstance(value, timestamp_pb2.Timestamp):
    return value.ToDatetime().isoformat()
  elif isinstance(value, message_pb.Message):
    return message_to_dict(value)
  else:
    return value


def send_rows(bq_client, dataset_id, table_id, rows, batch_size=_BATCH_DEFAULT):
  """Sends rows to BigQuery.

  Args:
    rows: a list of any of the following
      * tuples: each tuple should contain data of the correct type for each
      schema field on the current table and in the same order as the schema
      fields.
      * google.protobuf.message.Message instance
    bq_client: an instance of google.cloud.bigquery.client.Client
    dataset_id, table_id (str): identifiers for the table to which the rows will
      be inserted
    batch_size (int): the max number of rows to send to BigQuery in a single
      request. Values exceeding the limit will use the limit. Values less than 1
      will use _BATCH_DEFAULT.

  Please use google.protobuf.message.Message instances moving forward.
  Tuples are deprecated.
  """
  if batch_size > _BATCH_LIMIT:
    batch_size = _BATCH_LIMIT
  elif batch_size <= 0:
    batch_size = _BATCH_DEFAULT
  for i, row in enumerate(rows):
    if isinstance(row, tuple):
      continue
    elif isinstance(row, message_pb.Message):
      rows[i] = message_to_dict(row)
    else:
      raise UnsupportedTypeError(type(row).__name__)
  table = bq_client.get_table(bq_client.dataset(dataset_id).table(table_id))
  for row_set in _batch(rows, batch_size):
    insert_errors = bq_client.create_rows(table, row_set)
    if insert_errors:
      logging.error('Failed to send event to bigquery: %s', insert_errors)
      raise BigQueryInsertError(insert_errors)


def _batch(rows, batch_size):
  for i in xrange(0, len(rows), batch_size):
    yield rows[i:i + batch_size]


class UnsupportedTypeError(Exception):
  """BigQueryHelper only supports row representations described by send_rows.

  row_type: string representation of type.
  """
  def __init__(self, row_type):
    msg = 'Unsupported row type for BigQueryHelper.send_rows: %s' % row_type
    super(UnsupportedTypeError, self).__init__(msg)


class BigQueryInsertError(Exception):
  """Error from create_rows() call on BigQuery client.

  insert_errors is in the form of a list of mappings, where each mapping
  contains an "index" key corresponding to a row and an "errors" key.
  """
  def __init__(self, insert_errors):
    message = self._construct_message(insert_errors)
    super(BigQueryInsertError, self).__init__(message)

  @staticmethod
  def _construct_message(insert_errors):
    message = ''
    for row_mapping in insert_errors:
      index = row_mapping.get('index')
      for err in row_mapping.get('errors') or []:
        message += "Error inserting row %d: %s\n" % (index, err)
    return message


def _is_empty_message_type(desc):
  """Returns true if the message type results in an empty RECORD BQ field.

  Note: hangs on recursive messages.
  """
  for f in desc.fields:
    f_empty = f.message_type and _is_empty_message_type(f.message_type)
    if not f_empty:  # pragma: no branch
      return False
  return True
