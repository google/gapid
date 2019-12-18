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

/**
 * {@link Track} containing the CPU frequency and idle data.
 */
public class CpuFrequencyTrack extends Track.WithQueryEngine<CpuFrequencyTrack.Data> {
  private static final String FREQ_VIEW_SQL =
      "select %d cpu, ts, lead(ts) over (order by ts) - ts dur, value freq_value " +
      "from counter where track_id = %d";
  private static final String IDLE_VIEW_SQL =
      "select %d cpu, ts, lead(ts) over (order by ts) - ts dur, value idle_value " +
      "from counter where track_id = %d";
  private static final String ACT_VIEW_SQL =
      "select ts, dur, quantum_ts, cpu, freq_value freq, " +
        "case idle_value when 4294967295 then -1 else idle_value end idle " +
      "from %s";
  private static final String DATA_SQL = "select ts, dur, cast(idle as DOUBLE), freq from %s";
  private static final String DATA_QUANTIZED_SQL =
      "select min(ts), sum(dur), " +
      "case when min(idle) = -1 then cast(-1 as DOUBLE) else cast(0 as DOUBLE) end, " +
      "sum(weighted_freq) / sum(dur), quantum_ts " +
      "from (select ts, dur, quantum_ts, freq * dur as weighted_freq, idle from %s) " +
      "group by quantum_ts";

  private final CpuInfo.Cpu cpu;

  public CpuFrequencyTrack(QueryEngine qe, CpuInfo.Cpu cpu) {
    super(qe, "cpu_freq_" + cpu.id);
    this.cpu = cpu;
  }

  public CpuInfo.Cpu getCpu() {
    return cpu;
  }

  @Override
  protected ListenableFuture<?> initialize() {
    String activity = tableName("activity");
    String span = tableName("span");
    String idle = tableName("idle");
    String freq = tableName("freq");
    String freqIdle = tableName("freq_idle");
    String window = tableName("window");
    return qe.queries(
        dropView(activity),
        dropTable(span),
        dropView(idle),
        dropView(freq),
        dropTable(freqIdle),
        dropTable(window),
        createWindow(window),
        createView(freq, format(FREQ_VIEW_SQL, cpu.id, cpu.freqId)),
        createView(idle, format(IDLE_VIEW_SQL, cpu.id, cpu.idleId)),
        createSpan(freqIdle, freq + " PARTITIONED cpu, " + idle + " PARTITIONED cpu"),
        createSpan(span, freqIdle + " PARTITIONED cpu, " + window),
        createView(activity, format(ACT_VIEW_SQL, span))
    );
  }

  @Override
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window window = Window.compute(req, 10);
    return transformAsync(window.update(qe, tableName("window")),
        $ -> compute(req, window.quantized));
  }

  private ListenableFuture<Data> compute(DataRequest req, boolean quantized) {
    return transform(qe.query(
        format(quantized ? DATA_QUANTIZED_SQL : DATA_SQL, tableName("activity"))), result -> {
      int rows = result.getNumRows();
      Data data = new Data(
          req, quantized, new long[rows], new long[rows], new byte[rows], new int[rows]);
      result.forEachRow((i, r) -> {
        long start = r.getLong(0);
        data.tsStarts[i] = start;
        data.tsEnds[i] = start + r.getLong(1);
        data.idles[i] = (byte)r.getDouble(2);
        data.freqKHz[i] = (int)r.getDouble(3);
      });
      return data;
    });
  }

  public static class Data extends Track.Data {
    public final boolean quantized;
    public final long[] tsStarts;
    public final long[] tsEnds;
    public final byte[] idles;
    public final int[] freqKHz;

    public Data(DataRequest request, boolean quantized, long[] tsStarts, long[] tsEnds,
        byte[] idles, int[] freqKHz) {
      super(request);
      this.quantized = quantized;
      this.tsStarts = tsStarts;
      this.tsEnds = tsEnds;
      this.idles = idles;
      this.freqKHz = freqKHz;
    }
  }
}
