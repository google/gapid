# Copyright 2015 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

from google.appengine.datastore import datastore_query
from google.appengine.ext import ndb


def fetch_page(query, batch_size, cursor_str, **kwargs):
  """Fetches a page from a query.

  Arguments:
    query: ndb.Query.
    batch_size: Maximum number of items to return.
    cursor_str: query-dependent string encoded cursor to continue a previous
        search.

  Returns:
  - items
  - str encoded cursor if relevant or None.
  """
  assert isinstance(query, ndb.Query), query
  if not 0 < batch_size <= 1000 or not isinstance(batch_size, int):
    raise ValueError(
        'batch_size must be between 1 and 1000, got %r', batch_size)
  if cursor_str:
    if not isinstance(cursor_str, basestring):
      raise ValueError(
          'cursor must be between valid string, got %r', cursor_str)
    cursor = datastore_query.Cursor(urlsafe=cursor_str)
  else:
    cursor = None
  items, cursor, more = query.fetch_page(
      batch_size, start_cursor=cursor, **kwargs)
  if not more:
    return items, None
  return items, cursor.urlsafe()
