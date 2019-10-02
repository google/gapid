# coding: utf-8
# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Packing and unpacking of ndb.Key.

End users are only given packed keys, which permits to not expose internal
schema details to the user.
"""

from google.appengine.ext import ndb


# Mask to TaskRequest key ids so they become decreasing numbers.
TASK_REQUEST_KEY_ID_MASK = int(2L**63-1)


### Entities relationships.


def request_key_to_result_summary_key(request_key):
  """Returns the TaskResultSummary ndb.Key for this TaskRequest.key."""
  assert request_key.kind() == 'TaskRequest', request_key
  assert request_key.integer_id(), request_key
  return ndb.Key('TaskResultSummary', 1, parent=request_key)


def request_key_to_secret_bytes_key(request_key):
  """Returns the SecretBytes ndb.Key for this TaskRequest.key."""
  assert request_key.kind() == 'TaskRequest', request_key
  assert request_key.integer_id(), request_key
  return ndb.Key('SecretBytes', 1, parent=request_key)


def result_summary_key_to_request_key(result_summary_key):
  """Returns the TaskRequest ndb.Key for this TaskResultSummmary key."""
  assert result_summary_key.kind() == 'TaskResultSummary', result_summary_key
  return result_summary_key.parent()


def result_summary_key_to_run_result_key(result_summary_key, try_number):
  """Returns the TaskRunResult ndb.Key for this TaskResultSummary.key.

  Arguments:
    result_summary_key: ndb.Key for a TaskResultSummary entity.
    try_number: the try on which TaskRunResult was created for. The first try
        is 1, the second is 2, etc.

  Returns:
    ndb.Key for the corresponding TaskRunResult entity.
  """
  assert result_summary_key.kind() == 'TaskResultSummary', result_summary_key
  if try_number < 1:
    raise ValueError('Try number(%d) must be above 0' % try_number)
  if try_number > 2:
    raise ValueError('Try number(%d) > 2 is not supported' % try_number)
  return ndb.Key('TaskRunResult', try_number, parent=result_summary_key)


def run_result_key_to_result_summary_key(run_result_key):
  """Returns the TaskResultSummary ndb.Key for this TaskRunResult.key.
  """
  assert run_result_key.kind() == 'TaskRunResult', run_result_key
  return run_result_key.parent()


def run_result_key_to_performance_stats_key(run_result_key):
  """Returns the PerformanceStats ndb.Key for this TaskRunResult.key.
  """
  assert run_result_key.kind() == 'TaskRunResult', run_result_key
  return ndb.Key('PerformanceStats', 1, parent=run_result_key)


### Packing and unpacking.


def get_request_and_result_keys(task_id):
  """Provides the key and TaskRequest corresponding to a task ID.

  Returns:
    tuple(request_key, result_key): ndb.Key that yield TaskRequest and either
                                    (TaskRunResult or TaskResultSummay).
  """
  try:
    key = unpack_result_summary_key(task_id)
    request_key = result_summary_key_to_request_key(key)
  except ValueError:
    key = unpack_run_result_key(task_id)
    request_key = result_summary_key_to_request_key(
        run_result_key_to_result_summary_key(key))
  return request_key, key


def pack_request_key(request_key):
  """Returns a request_id as a string from a TaskRequest ndb.Key."""
  key_id = request_key.integer_id()
  # It's 0xE instead of 0x1 in the DB because of the XOR.
  if (key_id & 0xF) != 0xE:
    raise ValueError('Invalid request key')
  return '%x' % (key_id ^ TASK_REQUEST_KEY_ID_MASK)


def pack_result_summary_key(result_summary_key):
  """Returns TaskResultSummary ndb.Key encoded, safe to use in HTTP requests.
  """
  assert result_summary_key.kind() == 'TaskResultSummary'
  request_key = result_summary_key_to_request_key(result_summary_key)
  return pack_request_key(request_key) + '0'


def pack_run_result_key(run_result_key):
  """Returns TaskRunResult ndb.Key encoded, safe to use in HTTP requests.
  """
  assert run_result_key.kind() == 'TaskRunResult'
  request_key = result_summary_key_to_request_key(
      run_result_key_to_result_summary_key(run_result_key))
  try_id = run_result_key.integer_id()
  assert 1 <= try_id <= 15, try_id
  return pack_request_key(request_key) + '%x' % try_id


def unpack_request_key(request_id):
  """Returns the ndb.Key for a TaskRequest id with the try number stripped.

  If you find yourself the need to unpack a task id as a ndb Key to use the
  datastore web UI, run the following in a python shell:
    task = 0x<task id>
    print((task/16)^int(2L**63-1))

  Then create a GQL query in the web UI that retrieves all the entities stored
  for this request, replace 1234 with the number printed by the commands above:
      SELECT * WHERE __key__ HAS ANCESTOR KEY(TaskRequest, 1234)
  """
  assert isinstance(request_id, basestring)
  if not request_id:
    raise ValueError('Invalid null key')
  c = request_id[-1]
  if c == '1':
    # The key id is the reverse of the value.
    task_id_int = int(request_id, 16)
    if task_id_int < 0:
      raise ValueError('Invalid task id (overflowed)')
    return ndb.Key('TaskRequest', task_id_int ^ TASK_REQUEST_KEY_ID_MASK)
  else:
    raise ValueError('Invalid key %r' % request_id)


def unpack_result_summary_key(packed_key):
  """Returns the TaskResultSummary ndb.Key from a packed key.

  The expected format of |packed_key| is %x.
  """
  request_key = unpack_request_key(packed_key[:-1])
  run_id = int(packed_key[-1], 16)
  if run_id & 0xff:
    raise ValueError('Can\'t reference to a specific try result.')
  return request_key_to_result_summary_key(request_key)


def unpack_run_result_key(packed_key):
  """Returns the TaskRunResult ndb.Key from a packed key.

  The expected format of |packed_key| is %x.
  """
  request_key = unpack_request_key(packed_key[:-1])
  run_id = int(packed_key[-1], 16)
  if not run_id:
    raise ValueError('Can\'t reference to the overall task result.')
  result_summary_key = request_key_to_result_summary_key(request_key)
  return result_summary_key_to_run_result_key(result_summary_key, run_id)
