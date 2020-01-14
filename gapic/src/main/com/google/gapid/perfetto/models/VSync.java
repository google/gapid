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

import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static java.lang.String.format;

import com.google.common.util.concurrent.ListenableFuture;

/**
 * A source of V-Sync data.
 */
public interface VSync {
  public static final VSync EMPTY = new VSync() {
    @Override
    public boolean hasData() {
      return false;
    }

    @Override
    public Data getData(Track.DataRequest req, Track.OnUiThread<Data> onUiThread) {
      return new Data(req, new long[0], false);
    }
  };

  public default boolean hasData() {
    return true;
  }

  public Data getData(Track.DataRequest req, Track.OnUiThread<Data> onUiThread);

  public static class Data extends Track.Data {
    public final long[] ts;
    public boolean fillFirst;

    public Data(Track.DataRequest request, long[] ts, boolean fillFirst) {
      super(request);
      this.ts = ts;
      this.fillFirst = fillFirst;
    }
  }

  public static class FromSurfaceFlingerAppCounter extends Track.WithQueryEngine<VSync.Data>
      implements VSync {
    private static final String FILL_FIRST_SQL =
        "select cast(value as int) from counter " +
        "where track_id = %d "  +
        "order by ts limit 1";

    private static final String COUNTER_SQL =
        "select ts, v from (" +
          "select row_number() over (order by ts) rn, ts, cast(value as int) v " +
          "from counter where track_id = %d and ts > %d " +
          "order by ts) " +
        "where rn <= 2 or ts < %d";

    private final CounterInfo counter;
    private boolean fillZeroValue = false;

    public FromSurfaceFlingerAppCounter(QueryEngine qe, CounterInfo counter) {
      super(qe, "vsync");
      this.counter = counter;
    }

    @Override
    protected ListenableFuture<?> initialize() {
      return transform(expectOneRow(qe.query(fillFirstSql())), r -> {
        fillZeroValue = r.getInt(0) != 0;
        return null;
      });
    }

    private String fillFirstSql() {
      return format(FILL_FIRST_SQL, counter.id);
    }

    @Override
    protected ListenableFuture<VSync.Data> computeData(DataRequest req) {
      return transform(qe.query(counterSql(req)), res -> {
        int rows = res.getNumRows();
        boolean fillFirst = rows > 0 && (fillZeroValue == (res.getLong(0, 1, 0) == 0));
        VSync.Data data = new VSync.Data(req, new long[rows], fillFirst);
        res.forEachRow((i, r) -> {
          data.ts[i] = r.getLong(0);
        });
        return data;
      });
    }

    private String counterSql(DataRequest req) {
      return format(COUNTER_SQL, counter.id, req.range.start, req.range.end);
    }
  }
}
