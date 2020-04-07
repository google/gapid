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
import static com.google.gapid.perfetto.views.TrackContainer.single;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.ImmutableListMultimap;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.MemorySelectionView;
import com.google.gapid.perfetto.views.MemorySummaryPanel;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.Set;
import java.util.function.Consumer;
import java.util.stream.IntStream;

/**
 * {@link Track} containing the total system memory usage data.
 */
public class MemorySummaryTrack extends Track.WithQueryEngine<MemorySummaryTrack.Data> {
  private static final String VIEW_SQL =
      "select min(id) id, ts, lead(ts) over (order by ts) - ts dur, max(a) total, max(b) unused," +
      "   max(c) + max(d) + max(e) buffCache " +
      "from (select id, ts," +
      "  case when track_id = %d then cast(value as int) end a," +
      "  case when track_id = %d then cast(value as int) end b," +
      "  case when track_id = %d then cast(value as int) end c," +
      "  case when track_id = %d then cast(value as int) end d," +
      "  case when track_id = %d then cast(value as int) end e " +
      "  from counter where track_id in (%d, %d, %d, %d, %d))" +
      "group by ts";
  private static final String SUMMARY_SQL =
      "select id, min(start), max(end), cast(avg(total) as int), cast(avg(unused) as int), " +
      "    cast(avg(buffCache) as int) from ( " +
      "  select min(id) id, min(ts) start, max(ts + dur) end, cast(avg(total) as int) total, " +
      "      cast(avg(unused) as int) unused, cast(avg(buffCache) as int) buffCache " +
      "  from %s group by quantum_ts)" +
      "group by id";
  private static final String COUNTER_SQL =
      "select id, ts, ts + dur, total, unused, buffCache from %s";
  private static final String VALUE_SQL =
      "select id, ts, dur, total, unused, buffCache from %s where id = %d";
  private static final String RANGE_SQL =
      "select id, ts, dur, total, unused, buffCache from %s " +
      "where ts + dur >= %d and ts <= %d order by ts";

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

      Data data = new Data(req, new long [rows + 1], new long[rows + 1], new long[rows + 1],
          new long[rows + 1], new long[rows + 1]);
      res.forEachRow((i, r) -> {
        data.id[i] = r.getLong(0);
        data.ts[i] = r.getLong(1);
        data.total[i] = r.getLong(3);
        data.unused[i] = r.getLong(4);
        data.buffCache[i] = r.getLong(5);
      });
      data.id[rows] = data.id[rows - 1];
      data.ts[rows] = res.getLong(rows - 1, 2, 0);
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

  public ListenableFuture<Values> getValue(long id) {
    return transform(expectOneRow(qe.query(valueSql(id))), row -> {
      Values v = new Values(new long[1], new long[1], new long[1], new long[1], new long[1],
          Sets.newHashSet());
      v.valueKeys.add(row.getLong(0));
      v.ts[0] = row.getLong(1);
      v.dur[0] = row.getLong(2);
      v.total[0] = row.getLong(3);
      v.unused[0] = row.getLong(4);
      v.buffCache[0] = row.getLong(5);
      return v;
    });
  }

  private String valueSql(long id) {
    return format(VALUE_SQL, tableName("vals"), id);
  }

  public ListenableFuture<Values> getValues(TimeSpan ts) {
    return transform(qe.query(rangeSql(ts)), res -> {
      int rows = res.getNumRows();
      Values v = new Values(new long[rows], new long[rows], new long[rows], new long[rows],
          new long[rows], Sets.newHashSet());
      res.forEachRow((i, r) -> {
        v.valueKeys.add(r.getLong(0));
        v.ts[i] = r.getLong(1);
        v.dur[i] = r.getLong(2);
        v.total[i] = r.getLong(3);
        v.unused[i] = r.getLong(4);
        v.buffCache[i] = r.getLong(5);
      });
      return v;
    });
  }

  private String rangeSql(TimeSpan ts) {
    return format(RANGE_SQL, tableName("vals"), ts.start, ts.end);
  }

  private static CounterInfo onlyOne(ImmutableList<CounterInfo> counters) {
    return (counters.size() != 1) ? null : counters.get(0);
  }

  public static class Data extends Track.Data {
    public final long[] id;
    public final long[] ts;
    public final long[] total;
    public final long[] unused;
    public final long[] buffCache;

    public Data(DataRequest request, long[] id, long[] ts, long[] total, long[] unusued,
        long[] buffCache) {
      super(request);
      this.id = id;
      this.ts = ts;
      this.total = total;
      this.unused = unusued;
      this.buffCache = buffCache;
    }

    public static Data empty(DataRequest req) {
      return new Data(req, new long[0], new long[0], new long[0], new long[0], new long[0]);
    }
  }

  public static class Values implements Selection, Selection.Builder<Values> {
    public final long[] ts;
    public final long[] dur;
    public final long[] total;
    public final long[] unused;
    public final long[] buffCache;
    private final Set<Long> valueKeys;

    public Values(long[] ts, long[] dur, long[] total, long[] unused, long[] buffCache,
        Set<Long> valueKeys) {
      this.ts = ts;
      this.dur = dur;
      this.total = total;
      this.unused = unused;
      this.buffCache = buffCache;
      this.valueKeys = valueKeys;
    }

    @Override
    public String getTitle() {
      return "Memory Usage";
    }

    @Override
    public boolean contains(Long key) {
      return valueKeys.contains(key);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new MemorySelectionView(parent, state, this);
    }

    @Override
    public Selection.Builder<Values> getBuilder() {
      return this;
    }

    @Override
    public void getRange(Consumer<TimeSpan> span) {
      long min = Arrays.stream(ts).min().getAsLong();
      long max = IntStream.range(0, ts.length).mapToLong(i -> ts[i] + dur[i]).max().getAsLong();
      span.accept(new TimeSpan(min, max));
    }

    @Override
    public Values combine(Values that) {
      if (ts.length == 0) {
        return that;
      } else if (that.ts.length == 0) {
        return this;
      }

      long[] newTs = new long[this.ts.length + that.ts.length];
      long[] newDur = new long[this.ts.length + that.ts.length];
      long[] newTotal = new long[this.ts.length + that.ts.length];
      long[] newUnused = new long[this.ts.length + that.ts.length];
      long[] newBuffCache = new long[this.ts.length + that.ts.length];

      int ai = 0, bi = 0, ri = 0;
      for (; ai < this.ts.length && bi < that.ts.length; ri++) {
        long at = this.ts[ai], bt = that.ts[bi];
        boolean selectThis = true;
        if (at == bt) {
          ai++;
          bi++;
        } else if (at < bt) {
          ai++;
        } else {
          bi++;
          selectThis = false;
        }
        if (selectThis) {
          newTs[ri] = at;
          newDur[ri] = this.dur[ai - 1];
          newTotal[ri] = this.total[ai - 1];
          newUnused[ri] = this.unused[ai - 1];
          newBuffCache[ri] = this.buffCache[ai - 1];
        } else {
          newTs[ri] = bt;
          newDur[ri] = that.dur[bi - 1];
          newTotal[ri] = that.total[bi - 1];
          newUnused[ri] = that.unused[bi - 1];
          newBuffCache[ri] = that.buffCache[bi - 1];
        }
      }

      // Copy the rest trailing parts from the unfinished arrays.
      for (; ai < this.ts.length; ri++, ai++) {
        newTs[ri] = this.ts[ai];
        newDur[ri] = this.dur[ai];
        newTotal[ri] = this.total[ai];
        newUnused[ri] = this.unused[ai];
        newBuffCache[ri] = this.buffCache[ai];
      }
      for (; bi < that.ts.length; ri++, bi++) {
        newTs[ri] = that.ts[bi];
        newDur[ri] = that.dur[bi];
        newTotal[ri] = that.total[bi];
        newUnused[ri] = that.unused[bi];
        newBuffCache[ri] = that.buffCache[bi];
      }

      Set<Long> newValueKeys = Sets.newHashSet(this.valueKeys);
      newValueKeys.addAll(that.valueKeys);

      // Truncate.
      int newL = ri;
      return new Values(Arrays.copyOf(newTs, newL), Arrays.copyOf(newDur, newL),
          Arrays.copyOf(newTotal, newL), Arrays.copyOf(newUnused, newL),
          Arrays.copyOf(newBuffCache, newL), newValueKeys);
    }

    @Override
    public Selection build() {
      return this;
    }
  }
}
