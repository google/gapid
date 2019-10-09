/*
 * Copyright (C) 2019 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package com.google.gapid.perfetto.models;

import static com.google.gapid.util.MoreFutures.transform;
import static java.lang.String.format;

import com.google.common.collect.ImmutableMap;
import com.google.common.util.concurrent.ListenableFuture;

import java.util.Map;

/**
 * Collection of "extra data" from the args table.
 */
public class ArgSet {
  private static final String ARGS_QUERY =
      "select key, int_value, real_value, string_value from args where arg_set_id = %d";

  public static final ArgSet EMPTY = new ArgSet(ImmutableMap.of());

  private final ImmutableMap<String, Object> values;

  public ArgSet(ImmutableMap<String, Object> values) {
    this.values = values;
  }

  public static ListenableFuture<ArgSet> get(QueryEngine qe, long id) {
    return transform(qe.query(sql(id)), ArgSet::of);
  }

  private static String sql(Long id) {
    return format(ARGS_QUERY, id);
  }

  public static ArgSet of(QueryEngine.Result res) {
    if (res.getNumRows() == 0) {
      return EMPTY;
    }

    ImmutableMap.Builder<String, Object> map = ImmutableMap.builder();
    res.forEachRow(($, r) -> {
      if (!r.isNull(1)) {
        map.put(r.getString(0), r.getInt(1));
      } else if (!r.isNull(2)) {
        map.put(r.getString(0), r.getDouble(2));
      } else {
        map.put(r.getString(0), r.getString(3));
      }
    });
    return new ArgSet(map.build());
  }

  public boolean isEmpty() {
    return values.isEmpty();
  }

  public int size() {
    return values.size();
  }

  public Iterable<String> keys() {
    return values.keySet();
  }

  public Iterable<Map.Entry<String, Object>> entries() {
    return values.entrySet();
  }

  public Object get(String key) {
    return values.get(key);
  }
}