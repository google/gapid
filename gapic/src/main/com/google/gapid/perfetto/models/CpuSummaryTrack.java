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
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;

import java.util.List;

/**
 * {@link Track} summarizing the total CPU usage.
 */
public class CpuSummaryTrack extends Track<CpuSummaryTrack.Data> {
  private static final String DATA_SQL =
      "select quantum_ts, sum(dur)/cast(%d * %d as float) " +
      "from %s where utid != 0 group by quantum_ts";
  private static final String SLICE_RANGE_SQL =
      "select row_id, ts, dur, cpu, utid, end_state, priority from sched " +
      "where utid != 0 and ts < %d and ts_end >= %d";

  private final int numCpus;

  public CpuSummaryTrack(int numCpus) {
    super("cpu_sum");
    this.numCpus = numCpus;
  }

  public int getNumCpus() {
    return numCpus;
  }

  @Override
  protected ListenableFuture<?> initialize(QueryEngine qe) {
    String span = tableName("span"), window = tableName("window");
    return qe.queries(
        dropTable(span),
        dropTable(window),
        createWindow(window),
        createSpan(span, "sched PARTITIONED cpu, " + window));
  }

  @Override
  protected ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req) {
    Window window = Window.quantized(req, 5);
    return transformAsync(window.update(qe, tableName("window")), $1 ->
      transform(qe.query(sql(window.bucketSize)), res -> {
        Data data = new Data(req, window.bucketSize, new double[window.getNumberOfBuckets()]);
        res.forEachRow(($2, r) -> data.utilizations[r.getInt(0)] = r.getDouble(1));
        return data;
      }));
  }

  private String sql(long ns) {
    return format(DATA_SQL, numCpus, ns, tableName("span"));
  }

  public static ListenableFuture<List<CpuTrack.Slice>> getSlices(QueryEngine qe, TimeSpan ts) {
    return transform(qe.query(sliceRangeSql(ts)), result -> {
      List<CpuTrack.Slice> slices = Lists.newArrayList();
      result.forEachRow((i, r) -> slices.add(new CpuTrack.Slice(r)));
      return slices;
    });
  }

  private static String sliceRangeSql(TimeSpan ts) {
    return format(SLICE_RANGE_SQL, ts.end, ts.start);
  }

  public static class Data extends Track.Data {
    public final long bucketSize;
    public final double[] utilizations;

    public Data(DataRequest request, long bucketSize, double[] utilizations) {
      super(request);
      this.bucketSize = bucketSize;
      this.utilizations = utilizations;
    }
  }
}
