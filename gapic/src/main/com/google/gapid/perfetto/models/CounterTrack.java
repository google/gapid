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
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.util.concurrent.ListenableFuture;

public class CounterTrack extends Track<CounterTrack.Data> {
  private static final String VIEW_SQL =
      "select ts, lead(ts) over (order by ts) - ts dur, value " +
      "from counter_values where counter_id = %d";
  private static final String SUMMARY_SQL =
      "select min(ts), max(ts + dur), avg(value) from %s group by quantum_ts";
  private static final String COUNTER_SQL = "select ts, ts + dur, value from %s";

  private final long id;
  private final double min;
  private final double max;

  public CounterTrack(long id, double min, double max) {
    super("counter_" + id);
    this.id = id;
    this.min = min;
    this.max = max;
  }

  public double getMin() {
    return min;
  }

  public double getMax() {
    return max;
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
    return format(VIEW_SQL, id);
  }

  @Override
  protected ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req) {
    Window win = Window.compute(req, 5);
    return transformAsync(win.update(qe, tableName("window")), $ -> computeData(qe, req, win));
  }

  private ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req, Window win) {
    return transform(qe.query(win.quantized ? summarySql() : counterSQL()), res -> {
      int rows = res.getNumRows();
      Data data = new Data(req, new long[rows + 1], new double[rows + 1]);
      res.forEachRow((i, r) -> {
        data.ts[i] = r.getLong(0);
        data.values[i] = r.getDouble(2);
      });
      data.ts[rows] = res.getLong(rows - 1, 1, 0);
      data.values[rows] = data.values[rows - 1];
      return data;
    });
  }

  private String summarySql() {
    return format(SUMMARY_SQL, tableName("span"));
  }

  private String counterSQL() {
    return format(COUNTER_SQL, tableName("span"));
  }

  public static class Data extends Track.Data {
    public final long[] ts;
    public final double[] values;

    public Data(DataRequest request, long[] ts, double[] values) {
      super(request);
      this.ts = ts;
      this.values = values;
    }
  }
}
