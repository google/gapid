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
import com.google.gapid.perfetto.views.BatterySummaryPanel;

public class BatterySummaryTrack extends Track.WithQueryEngine<BatterySummaryTrack.Data>{
  private static final String VIEW_SQL =
      "select ts, lead(ts) over (order by ts) - ts dur, max(a) capacity, max(b) charge," +
      "   max(c) current " +
      "from (select ts," +
      "  case when track_id = %d then cast(value as int) end a," +
      "  case when track_id = %d then cast(value as int) end b," +
      "  case when track_id = %d then cast(value as int) end c " +
      "  from counter where track_id in (%d, %d, %d))" +
      "group by ts";
  private static final String SUMMARY_SQL =
      "select min(ts), max(ts + dur), cast(avg(capacity) as int), cast(avg(charge) as int)," +
      "  cast(avg(current) as int) " +
      "from %s group by quantum_ts";
  private static final String COUNTER_SQL =
      "select ts, ts + dur, capacity, charge, current from %s";

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

      Data data = new Data(
          req, new long[rows + 1], new long[rows + 1], new long[rows + 1], new long[rows + 1]);
      res.forEachRow((i, r) -> {
        data.ts[i] = r.getLong(0);
        data.capacity[i] = r.getLong(2);
        data.charge[i] = r.getLong(3);
        data.current[i] = r.getLong(4);
      });
      data.ts[rows] = res.getLong(rows - 1, 1, 0);
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

  private static CounterInfo onlyOne(ImmutableList<CounterInfo> counters) {
    return (counters.size() != 1) ? null : counters.get(0);
  }

  public static class Data extends Track.Data {
    public final long[] ts;
    public final long[] capacity;
    public final long[] charge;
    public final long[] current;

    public Data(DataRequest request, long[] ts, long[] capacity, long[] charge, long[] current) {
      super(request);
      this.ts = ts;
      this.capacity = capacity;
      this.charge = charge;
      this.current = current;
    }

    public static Data empty(DataRequest req) {
      return new Data(req, new long[0], new long[0], new long[0], new long[0]);
    }
  }
}
