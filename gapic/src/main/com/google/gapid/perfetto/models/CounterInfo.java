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

import com.google.common.collect.ImmutableMap;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;

public class CounterInfo {
  private static final String LIST_SQL =
      "select counter_id, name, ref, ref_type, description, count(1)," +
      " min(value), max(value), avg(value) " +
      "from counter_definitions left join counter_values using (counter_id) " +
      "group by counter_id";

  public final long id;
  public final String name;
  public final long ref;
  public final String refType;
  public final String description;
  public final long count;
  public final double min;
  public final double max;
  public final double avg;

  public CounterInfo(long id, String name, long ref, String refType, String description, long count,
      double min, double max, double avg) {
    this.id = id;
    this.name = name;
    this.ref = ref;
    this.refType = refType;
    this.description = description;
    this.count = count;
    this.min = min;
    this.max = max;
    this.avg = avg;
  }

  private CounterInfo(QueryEngine.Row row) {
    this(row.getLong(0), row.getString(1), row.getLong(2), row.getString(3), row.getString(4),
        row.getLong(5), row.getDouble(6), row.getDouble(7), row.getDouble(8));
  }

  public static ListenableFuture<Perfetto.Data.Builder> listCounters(Perfetto.Data.Builder data) {
    return transform(data.qe.query(LIST_SQL), res -> {
      ImmutableMap.Builder<Long, CounterInfo> counters = ImmutableMap.builder();
      res.forEachRow((i, r) -> counters.put(r.getLong(0), new CounterInfo(r)));
      return data.setCounters(counters.build());
    });
  }
}
