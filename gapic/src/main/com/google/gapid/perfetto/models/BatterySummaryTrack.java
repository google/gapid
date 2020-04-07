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
import com.google.gapid.perfetto.views.BatterySelectionView;
import com.google.gapid.perfetto.views.BatterySummaryPanel;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.Set;
import java.util.function.Consumer;
import java.util.stream.IntStream;

public class BatterySummaryTrack extends Track.WithQueryEngine<BatterySummaryTrack.Data>{
  private static final String VIEW_SQL =
      "select min(id) id, ts, lead(ts) over (order by ts) - ts dur, max(a) capacity, max(b) charge," +
      "   max(c) current " +
      "from (select id, ts," +
      "  case when track_id = %d then cast(value as int) end a," +
      "  case when track_id = %d then cast(value as int) end b," +
      "  case when track_id = %d then cast(value as int) end c " +
      "  from counter where track_id in (%d, %d, %d))" +
      "group by ts";
  private static final String SUMMARY_SQL =
      "select id, min(start), max(end), cast(avg(capacity) as int), cast(avg(charge) as int), " +
      "    cast(avg(current) as int) from ( " +
      "  select min(id) id, min(ts) start, max(ts + dur) end, cast(avg(capacity) as int) capacity, " +
      "      cast(avg(charge) as int) charge, cast(avg(current) as int) current " +
      "  from %s group by quantum_ts)" +
      "group by id";
  private static final String COUNTER_SQL =
      "select id, ts, ts + dur, capacity, charge, current from %s";
  private static final String VALUE_SQL =
      "select id, ts, dur, capacity, charge, current from %s where id = %d";
  private static final String RANGE_SQL =
      "select id, ts, dur, capacity, charge, current from %s " +
          "where ts + dur >= %d and ts <= %d order by ts";

  private final long capacityId;
  private final long chargeId;
  private final long currentId;
  private final double maxAbsCurrent;
  private final boolean needQuantize;

  public BatterySummaryTrack(
      QueryEngine qe, CounterInfo capacity, CounterInfo charge, CounterInfo current) {
    super(qe, "bat_sum");
    this.capacityId = capacity.id;
    this.chargeId = charge.id;
    this.currentId = current.id;
    long maxCount = Math.max(capacity.count, Math.max(charge.count, current.count));
    this.maxAbsCurrent = Math.max(Math.abs(current.min), Math.abs(current.max));
    this.needQuantize = maxCount > Track.QUANTIZE_CUT_OFF;
  }

  public double getMaxAbsCurrent() {
    return maxAbsCurrent;
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
    return format(VIEW_SQL, capacityId, chargeId, currentId, capacityId, chargeId, currentId);
  }

  @Override
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window win = needQuantize ? Window.compute(req, 5) : Window.compute(req);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(req, win));
  }

  private ListenableFuture<Data> computeData(DataRequest req, Window win) {
    return transform(qe.query(win.quantized ? summarySql() : counterSQL()), res -> {
      int rows = res.getNumRows();
      if (rows == 0) {
        return Data.empty(req);
      }

      Data data = new Data(req, new long[rows + 1], new long[rows + 1], new long[rows + 1],
          new long[rows + 1], new long[rows + 1]);
      res.forEachRow((i, r) -> {
        data.id[i] = r.getLong(0);
        data.ts[i] = r.getLong(1);
        data.capacity[i] = r.getLong(3);
        data.charge[i] = r.getLong(4);
        data.current[i] = r.getLong(5);
      });
      data.id[rows] = data.id[rows - 1];
      data.ts[rows] = res.getLong(rows - 1, 2, 0);
      data.capacity[rows] = data.capacity[rows - 1];
      data.charge[rows] = data.charge[rows - 1];
      data.current[rows] = data.current[rows - 1];
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
    CounterInfo battCap = onlyOne(counters.get("batt.capacity_pct"));
    CounterInfo battCharge = onlyOne(counters.get("batt.charge_uah"));
    CounterInfo battCurrent = onlyOne(counters.get("batt.current_ua"));
    if ((battCap == null) || (battCharge  == null) || (battCurrent  == null)) {
      return data;
    }

    BatterySummaryTrack track = new BatterySummaryTrack(data.qe, battCap, battCharge, battCurrent);
    data.tracks.addTrack(null, track.getId(), "Battery Usage",
        single(state -> new BatterySummaryPanel(state, track), true, false));
    return data;
  }

  public ListenableFuture<Values> getValue(long id) {
    return transform(expectOneRow(qe.query(valueSql(id))), row -> {
      Values v = new Values(new long[1], new long[1], new long[1], new long[1], new long[1],
          Sets.newHashSet());
      v.valueKeys.add(row.getLong(0));
      v.ts[0] = row.getLong(1);
      v.dur[0] = row.getLong(2);
      v.capacity[0] = row.getLong(3);
      v.charge[0] = row.getLong(4);
      v.current[0] = row.getLong(5);
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
        v.capacity[i] = r.getLong(3);
        v.charge[i] = r.getLong(4);
        v.current[i] = r.getLong(5);
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
    public final long[] capacity;
    public final long[] charge;
    public final long[] current;

    public Data(DataRequest request, long[] id, long[] ts, long[] capacity, long[] charge,
        long[] current) {
      super(request);
      this.id = id;
      this.ts = ts;
      this.capacity = capacity;
      this.charge = charge;
      this.current = current;
    }

    public static Data empty(DataRequest req) {
      return new Data(req, new long[0], new long[0], new long[0], new long[0], new long[0]);
    }
  }

  public static class Values implements Selection, Selection.Builder<Values> {
    public final long[] ts;
    public final long[] dur;
    public final long[] capacity;
    public final long[] charge;
    public final long[] current;
    private final Set<Long> valueKeys;

    public Values(long[] ts, long[] dur, long[] capacity, long[] charge, long[] current,
        Set<Long> valueKeys) {
      this.ts = ts;
      this.dur = dur;
      this.capacity = capacity;
      this.charge = charge;
      this.current = current;
      this.valueKeys = valueKeys;
    }

    @Override
    public String getTitle() {
      return "Battery Usage";
    }

    @Override
    public boolean contains(Long key) {
      return valueKeys.contains(key);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new BatterySelectionView(parent, state, this);
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
      long[] newCapacity = new long[this.ts.length + that.ts.length];
      long[] newCharge = new long[this.ts.length + that.ts.length];
      long[] newCurrent = new long[this.ts.length + that.ts.length];

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
          newCapacity[ri] = this.capacity[ai - 1];
          newCharge[ri] = this.charge[ai - 1];
          newCurrent[ri] = this.current[ai - 1];
        } else {
          newTs[ri] = bt;
          newDur[ri] = that.dur[bi - 1];
          newCapacity[ri] = that.capacity[bi - 1];
          newCharge[ri] = that.charge[bi - 1];
          newCurrent[ri] = that.current[bi - 1];
        }
      }

      // Copy the rest trailing parts from the unfinished arrays.
      for (; ai < this.ts.length; ri++, ai++) {
        newTs[ri] = this.ts[ai];
        newDur[ri] = this.dur[ai];
        newCapacity[ri] = this.capacity[ai];
        newCharge[ri] = this.charge[ai];
        newCurrent[ri] = this.current[ai];
      }
      for (; bi < that.ts.length; ri++, bi++) {
        newTs[ri] = that.ts[bi];
        newDur[ri] = that.dur[bi];
        newCapacity[ri] = that.capacity[bi];
        newCharge[ri] = that.charge[bi];
        newCurrent[ri] = that.current[bi];
      }

      Set<Long> newValueKeys = Sets.newHashSet(this.valueKeys);
      newValueKeys.addAll(that.valueKeys);

      // Truncate.
      int newL = ri;
      return new Values(Arrays.copyOf(newTs, newL), Arrays.copyOf(newDur, newL),
          Arrays.copyOf(newCapacity, newL), Arrays.copyOf(newCharge, newL),
          Arrays.copyOf(newCurrent, newL), newValueKeys);
    }

    @Override
    public Selection build() {
      return this;
    }
  }
}
