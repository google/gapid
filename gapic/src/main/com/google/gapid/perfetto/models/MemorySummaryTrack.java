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

import static com.google.gapid.perfetto.models.QueryEngine.createSpan;
import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;

/**
 * {@link Track} containing the total system memory usage data.
 */
public class MemorySummaryTrack extends Track<MemorySummaryTrack.Data> {
  private static final String DEF_TABLE = "counter_definitions";
  private static final String DEF_SQL = "select " +
      "(select counter_id from " + DEF_TABLE + " where name = 'MemTotal' and ref_type is null), " +
      "(select counter_id from " + DEF_TABLE + " where name = 'MemFree' and ref_type is null), " +
      "(select counter_id from " + DEF_TABLE + " where name = 'Buffers' and ref_type is null), " +
      "(select counter_id from " + DEF_TABLE + " where name = 'Cached' and ref_type is null), " +
      "(select counter_id from " + DEF_TABLE + " where name = 'SwapCached' and ref_type is null)";
  private static final String MAX_SQL =
      "select cast(max(value) as int) from counter_values where counter_id = %d";
  private static final String VIEW_SQL =
      "select ts, lead(ts) over (order by ts) - ts dur, max(a) total, max(b) unused," +
      "   max(c) + max(d) + max(e) buffCache " +
      "from (select ts," +
      "  case when counter_id = %d then cast(value as int) end a," +
      "  case when counter_id = %d then cast(value as int) end b," +
      "  case when counter_id = %d then cast(value as int) end c," +
      "  case when counter_id = %d then cast(value as int) end d," +
      "  case when counter_id = %d then cast(value as int) end e " +
      "  from counter_values where counter_id in (%d, %d, %d, %d, %d))" +
      "group by ts";
  private static final String SUMMARY_SQL =
      "select min(ts), max(ts + dur), cast(avg(total) as int), cast(avg(unused) as int)," +
      "  cast(avg(buffCache) as int) " +
      "from %s group by quantum_ts";
  private static final String COUNTER_SQL =
      "select ts, ts + dur, total, unused, buffCache from %s";

  private final long maxTotal;
  private final int totalId;
  private final int unusedId;
  private final int buffersId;
  private final int cachedId;
  private final int swapCachedId;

  public MemorySummaryTrack(
      long maxTotal, int totalId, int unusedId, int buffersId, int cachedId, int swapCachedId) {
    super("mem_sum");
    this.maxTotal = maxTotal;
    this.totalId = totalId;
    this.unusedId = unusedId;
    this.buffersId = buffersId;
    this.cachedId = cachedId;
    this.swapCachedId = swapCachedId;
  }

  public long getMaxTotal() {
    return maxTotal;
  }

  @Override
  protected ListenableFuture<?> initialize(QueryEngine qe) {
    String vals = tableName("vals");
    String span = tableName("span");
    String window = tableName("window");
    return qe.queries(
        dropTable(span),
        dropTable(window),
        dropView(vals),
        createView(vals, viewSql()),
        createWindow(window),
        createSpan(span, vals + ", " + window));
  }

  private String viewSql() {
    return format(VIEW_SQL, totalId, unusedId, buffersId, cachedId, swapCachedId,
        totalId, unusedId, buffersId, cachedId, swapCachedId);
  }

  @Override
  protected ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req) {
    Window win = Window.compute(req, 5);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(qe, req, win));
  }

  private ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req, Window win) {
    return transform(qe.query(win.quantized ? summarySql() : counterSQL()), res -> {
      int rows = res.getNumRows();
      Data data = new Data(
          req, new long[rows + 1], new long[rows + 1], new long[rows + 1], new long[rows + 1]);
      res.forEachRow((i, r) -> {
        data.ts[i] = r.getLong(0);
        data.total[i] = r.getLong(2);
        data.unused[i] = r.getLong(3);
        data.buffCache[i] = r.getLong(4);
      });
      data.ts[rows] = res.getLong(rows - 1, 1, 0);
      data.total[rows] = data.total[rows - 1];
      data.unused[rows] = data.unused[rows - 1];
      data.buffCache[rows] = data.buffCache[rows - 1];
      return data;
    });
  }

  private String summarySql() {
    return format(SUMMARY_SQL, tableName("span"));
  }

  private String counterSQL() {
    return format(COUNTER_SQL, tableName("span"));
  }

  public static ListenableFuture<MemorySummaryTrack> enumerate(QueryEngine qe) {
    return transformAsync(expectOneRow(qe.query(DEF_SQL)), res -> {
      int total = res.getInt(0, -1);
      int unusued = res.getInt(1, -1);
      int buffers = res.getInt(2, -1);
      int cached = res.getInt(3, -1);
      int swapCached = res.getInt(4, -1);
      if ((total < 0) || (unusued < 0) || (buffers < 0) || (cached < 0) || (swapCached < 0)) {
        return Futures.immediateFuture(null);
      }
      return transform(expectOneRow(qe.query(maxTotalSql(total))), max ->
        new MemorySummaryTrack(max.getLong(0), total, unusued, buffers, cached, swapCached));
    });
  }

  private static String maxTotalSql(long id) {
    return format(MAX_SQL, id);
  }

  public static class Data extends Track.Data {
    public final long[] ts;
    public final long[] total;
    public final long[] unused;
    public final long[] buffCache;

    public Data(DataRequest request, long[] ts, long[] total, long[] unusued, long[] buffCache) {
      super(request);
      this.ts = ts;
      this.total = total;
      this.unused = unusued;
      this.buffCache = buffCache;
    }
  }
}
