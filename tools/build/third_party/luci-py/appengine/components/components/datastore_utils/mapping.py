# Copyright 2014 The LUCI Authors. All rights reserved.
# Use of this source code is governed under the Apache License, Version 2.0
# that can be found in the LICENSE file.

"""Queries for incremental mapping."""

from google.appengine.ext import ndb


__all__ = [
  'incremental_map',
  'page_queries',
  'pop_future_done',
]


### Private stuff.


def _process_chunk_of_items(
    map_fn, action_futures, items_to_process, max_inflight, map_page_size):
  """Maps as many items as possible and throttles down to 'max_inflight'.

  |action_futures| is modified in-place.
  Remaining items_to_process is returned.
  """
  # First, throttle.
  pop_future_done(action_futures)
  while len(action_futures) > max_inflight:
    ndb.Future.wait_any(action_futures)
    pop_future_done(action_futures)

  # Then, map. map_fn() may return None so "or []" to not throw an exception. It
  # just means there no async operation to wait on.
  action_futures.extend(map_fn(items_to_process[:map_page_size]) or [])
  return items_to_process[map_page_size:]


### Public API.


def pop_future_done(futures):
  """Removes the currently done futures."""
  for i in xrange(len(futures) - 1, -1, -1):
    if futures[i].done():
      futures.pop(i)


def page_queries(queries, fetch_page_size=20):
  """Yields all the items returned by the queries, page by page.

  It makes heavy use of fetch_page_async() for maximum efficiency.
  """
  queries = queries[:]
  futures = [q.fetch_page_async(fetch_page_size) for q in queries]
  while queries:
    i = futures.index(ndb.Future.wait_any(futures))
    results, cursor, more = futures[i].get_result()
    if not more:
      # Remove completed queries.
      queries.pop(i)
      futures.pop(i)
    else:
      futures[i] = queries[i].fetch_page_async(
          fetch_page_size, start_cursor=cursor)
    yield results


def incremental_map(
    queries, map_fn, filter_fn=None, max_inflight=100, map_page_size=20,
    fetch_page_size=20):
  """Applies |map_fn| to objects in a list of queries asynchrously.

  This function is itself synchronous.

  It's a mapper without a reducer.

  Arguments:
    queries: list of iterators of items to process.
    map_fn: callback that accepts a list of objects to map and optionally
            returns a list of ndb.Future.
    filter_fn: optional callback that can filter out items from |query| from
               deletion when returning False.
    max_inflight: maximum limit of number of outstanding futures returned by
                  |map_fn|.
    map_page_size: number of items to pass to |map_fn| at a time.
    fetch_page_size: number of items to retrieve from |queries| at a time.
  """
  items_to_process = []
  action_futures = []

  for items in page_queries(queries, fetch_page_size=fetch_page_size):
    items_to_process.extend(i for i in items if not filter_fn or filter_fn(i))
    while len(items_to_process) >= map_page_size:
      items_to_process = _process_chunk_of_items(
          map_fn, action_futures, items_to_process, max_inflight, map_page_size)

  while items_to_process:
    items_to_process = _process_chunk_of_items(
        map_fn, action_futures, items_to_process, max_inflight, map_page_size)

  ndb.Future.wait_all(action_futures)

