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
import static com.google.gapid.perfetto.views.TrackContainer.single;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.ImmutableListMultimap;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.views.MemorySummaryPanel;

/**
 * {@link Track} containing the total system memory usage data.
 */
public class MemorySummaryTrack extends Track.WithQueryEngine<MemorySummaryTrack.Data> {
  private static final String VIEW_SQL =
      "select ts, lead(ts) over (order by ts) - ts dur, max(a) total, max(b) unused," +
      "   max(c) + max(d) + max(e) buffCache " +
      "from (select ts," +
      "  case when track_id = %d then cast(value as int) end a," +
      "  case when track_id = %d then cast(value as int) end b," +
      "  case when track_id = %d then cast(value as int) end c," +
      "  case when track_id = %d then cast(value as int) end d," +
      "  case when track_id = %d then cast(value as int) end e " +
      "  from counter where track_id in (%d, %d, %d, %d, %d))" +
      "group by ts";
  private static final String SUMMARY_SQL =
      "select min(ts), max(ts + dur), cast(avg(total) as int), cast(avg(unused) as int)," +
      "  cast(avg(buffCache) as int) " +
      "from %s group by quantum_ts";
  private static final String COUNTER_SQL =
      "select ts, ts + dur, total, unused, buffCache from %s";

  private final long maxTotal;
  private final long totalId;
  private final long unusedId;
  private final long buffersId;
  private final long cachedId;
  private final long swapCachedId;

  public MemorySummaryTrack(QueryEngine qe, long maxTotal, long totalId, long unusedId,
      long buffersId, long cachedId, long swapCachedId) {
    super(qe, "mem_sum");
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
  protected ListenableFuture<?> initialize() {
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
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window win = Window.compute(req, 5);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(req, win));
  }

  private ListenableFuture<Data> computeData(DataRequest req, Window win) {
    return transform(qe.query(win.quantized ? summarySql() : counterSQL()), res -> {
      int rows = res.getNumRows();
      if (rows == 0) {
        return Data.empty(req);
      }

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

  public static Perfetto.Data.Builder enumerate(Perfetto.Data.Builder data) {
    ImmutableListMultimap<String, CounterInfo> counters = data.getCounters(CounterInfo.Type.Global);
    CounterInfo total = onlyOne(counters.get("MemTotal"));
    CounterInfo free = onlyOne(counters.get("MemFree"));
    CounterInfo buffers = onlyOne(counters.get("Buffers"));
    CounterInfo cached = onlyOne(counters.get("Cached"));
    CounterInfo swapCached = onlyOne(counters.get("SwapCached"));
    if ((total == null) || (free  == null) || (buffers  == null) || (cached  == null) ||
        (swapCached  == null)) {
      return data;
    }

    MemorySummaryTrack track = new MemorySummaryTrack(
        data.qe, (long)total.max, total.id, free.id, buffers.id, cached.id, swapCached.id);
    data.tracks.addTrack(null, track.getId(), "Memory Usage",
        single(state -> new MemorySummaryPanel(state, track), true, false));
    return data;
  }

  private static CounterInfo onlyOne(ImmutableList<CounterInfo> counters) {
    return (counters.size() != 1) ? null : counters.get(0);
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

    public static Data empty(DataRequest req) {
      return new Data(req, new long[0], new long[0], new long[0], new long[0]);
    }
  }
}
