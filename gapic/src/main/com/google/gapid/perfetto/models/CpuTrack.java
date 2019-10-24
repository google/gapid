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

import static com.google.common.collect.ImmutableList.toImmutableList;
import static com.google.gapid.perfetto.models.QueryEngine.createSpan;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.perfetto.views.TrackContainer.single;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;
import static java.util.function.Function.identity;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.CpuFrequencyPanel;
import com.google.gapid.perfetto.views.CpuPanel;
import com.google.gapid.perfetto.views.CpuSliceSelectionView;
import com.google.gapid.perfetto.views.CpuSlicesSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Collections;
import java.util.List;
import java.util.Map;

/**
 * {@link Track} containing CPU slices for a single core.
 */
public class CpuTrack extends Track<CpuTrack.Data> {
  private static final String FREQ_IDLE_QUERY = "with " +
      "freq as (select ref, counter_id freq_id, max(value) freq " +
        "from counter_definitions cd left join counter_values cv using(counter_id) " +
        "where name = 'cpufreq' group by counter_id), " +
      "idle as (select ref, counter_id idle_id from counter_definitions where name = 'cpuidle') " +
      "select ref, freq_id, freq, idle_id from idle innter join freq using (ref)";
  private static final String SUMMARY_SQL =
      "select quantum_ts, sum(dur)/cast(%d as float) " +
      "from %s where cpu = %d and utid != 0 " +
      "group by quantum_ts";
  private static final String SLICES_SQL =
      "select ts, dur, utid, row_id from %s where cpu = %d and utid != 0";
  private static final String SLICE_SQL =
      "select row_id, ts, dur, cpu, utid, end_state, priority from sched where row_id = %d";
  private static final String SLICE_RANGE_SQL =
      "select row_id, ts, dur, cpu, utid, end_state, priority from sched " +
      "where cpu = %d and utid != 0 and ts < %d and ts_end >= %d";

  private final int cpu;

  public CpuTrack(int cpu) {
    super("cpu_" + cpu);
    this.cpu = cpu;
  }

  public int getCpu() {
    return cpu;
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
    Window window = Window.compute(req, 10);
    return transformAsync(window.update(qe, tableName("window")),
        $ -> window.quantized ? computeSummary(qe, req, window) : computeSlices(qe, req));
  }

  private ListenableFuture<Data> computeSummary(QueryEngine qe, DataRequest req, Window w) {
    return transform(qe.query(summarySql(w.bucketSize)), result -> {
      Data data = new Data(req, w.bucketSize, new double[w.getNumberOfBuckets()]);
      result.forEachRow(($, r) -> data.utilizations[r.getInt(0)] = r.getDouble(1));
      return data;
    });
  }

  private String summarySql(long ns) {
    return format(SUMMARY_SQL, ns, tableName("span"), cpu);
  }

  private ListenableFuture<Data> computeSlices(QueryEngine qe, DataRequest req) {
    return transform(qe.query(slicesSql()), result -> {
      int rows = result.getNumRows();
      Data data = new Data(req, new long[rows], new long[rows], new long[rows], new long[rows]);
      result.forEachRow((i, r) -> {
        long start = r.getLong(0);
        data.starts[i] = start;
        data.ends[i] = start + r.getLong(1);
        data.utids[i] = r.getInt(2);
        data.ids[i] = r.getLong(3);
      });
      return data;
    });
  }

  private String slicesSql() {
    return format(SLICES_SQL, tableName("span"), cpu);
  }

  public static ListenableFuture<Slice> getSlice(QueryEngine qe, long id) {
    return transform(expectOneRow(qe.query(sliceSql(id))), Slice::new);
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id);
  }

  public static ListenableFuture<List<Slice>> getSlices(QueryEngine qe, int cpu, TimeSpan ts) {
    return transform(qe.query(sliceRangeSql(cpu, ts)), result -> {
      List<Slice> slices = Lists.newArrayList();
      result.forEachRow((i, r) -> slices.add(new Slice(r)));
      return slices;
    });
  }

  private static String sliceRangeSql(int cpu, TimeSpan ts) {
    return format(SLICE_RANGE_SQL, cpu, ts.end, ts.start);
  }

  public static ListenableFuture<List<CpuConfig>> enumerate(
      String parent, Perfetto.Data.Builder data) {
    return transform(freqMap(data.qe), freqMap -> {
      List<CpuConfig> configs = Lists.newArrayList();
      for (int i = 0; i < data.getNumCpus(); i++) {
        QueryEngine.Row freq = freqMap.get(Long.valueOf(i));
        CpuTrack track = new CpuTrack(i);
        data.tracks.addTrack(parent, track.getId(), "CPU " + (i + 1),
            single(state -> new CpuPanel(state, track), false));
        if (freq != null) {
          CpuFrequencyTrack freqTrack =
              new CpuFrequencyTrack(i, freq.getLong(1), freq.getDouble(2), freq.getLong(3));
          data.tracks.addTrack(
              parent, freqTrack.getId(), "CPU " + (i + 1) + " Frequency",
              single(state -> new CpuFrequencyPanel(state, freqTrack), false));
        }
        configs.add(new CpuConfig(i, freq != null));
      }
      return configs;
    });
  }

  private static ListenableFuture<Map<Long, QueryEngine.Row>> freqMap(QueryEngine qe) {
    return transform(qe.query(FREQ_IDLE_QUERY), res -> res.map(row -> row.getLong(0), identity()));
  }

  public static class CpuConfig {
    public final int id;
    public final boolean hasFrequency;

    public CpuConfig(int id, boolean hasFrequency) {
      this.id = id;
      this.hasFrequency = hasFrequency;
    }
  }

  public static class Data extends Track.Data {
    public final Kind kind;
    // Summary.
    public final long bucketSize;
    public final double[] utilizations;
    // Slice.
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final long[] utids;

    public Data(DataRequest request, long bucketSize, double[] utilizations) {
      super(request);
      this.kind = Kind.summary;
      this.bucketSize = bucketSize;
      this.utilizations = utilizations;
      this.ids = null;
      this.starts = null;
      this.ends = null;
      this.utids = null;
    }

    public Data(DataRequest request, long[] ids, long[] starts, long[] ends, long[] utids) {
      super(request);
      this.kind = Kind.slice;
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.utids = utids;
      this.bucketSize = 0;
      this.utilizations = null;
    }

    public static enum Kind {
      summary, slice;
    }
  }

  public static class Slice implements Selection {
    public final long id;
    public final long time;
    public final long dur;
    public final int cpu;
    public final long utid;
    public final ThreadState endState;
    public final int priority;

    public Slice(
        long id, long time, long dur, int cpu, long utid, ThreadState endState, int priority) {
      this.id = id;
      this.time = time;
      this.dur = dur;
      this.cpu = cpu;
      this.utid = utid;
      this.endState = endState;
      this.priority = priority;
    }

    public Slice(QueryEngine.Row row) {
      this(row.getLong(0), row.getLong(1), row.getLong(2), row.getInt(3), row.getLong(4),
          ThreadState.of(row.getString(5)), row.getInt(6));
    }

    @Override
    public String getTitle() {
      return "CPU Slices";
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new CpuSliceSelectionView(parent, state, this);
    }

    @Override
    public void markTime(State state) {
      if (dur > 0) {
        state.setHighlight(new TimeSpan(time, time + dur));
      }
    }

    @Override
    public void zoom(State state) {
      if (dur > 0) {
        state.setVisibleTime(new TimeSpan(time, time + dur));
      }
    }

    @Override
    public String toString() {
      return "Slice{@" + time + " +" + dur + " " + utid + " " + endState + "/" + priority + "}";
    }
  }

  public static class Slices implements Selection.CombiningBuilder.Combinable<Slices> {
    private final Map<Long, ByProcess.Builder> processes = Maps.newHashMap();

    public Slices(State state, List<Slice> slices) {
      for (Slice slice : slices) {
        ThreadInfo ti = state.getThreadInfo(slice.utid);
        long pid = (ti == null) ? 0 : ti.upid;
        processes.computeIfAbsent(pid, ByProcess.Builder::new).add(slice);
      }
    }

    @Override
    public Slices combine(Slices other) {
      for (Map.Entry<Long, ByProcess.Builder> e : other.processes.entrySet()) {
        processes.merge(e.getKey(), e.getValue(), ByProcess.Builder::combine);
      }
      return this;
    }

    @Override
    public Selection build() {
      return new Selection(processes.values().stream()
          .map(ByProcess.Builder::build)
          .sorted((p1, p2) -> Long.compare(p2.dur, p1.dur))
          .collect(toImmutableList()));
    }

    public static class Selection implements com.google.gapid.perfetto.models.Selection {
      public final ImmutableList<ByProcess> processes;

      public Selection(ImmutableList<ByProcess> processes) {
        this.processes = processes;
      }

      @Override
      public String getTitle() {
        return "CPU Slices";
      }

      @Override
      public Composite buildUi(Composite parent, State state) {
        return new CpuSlicesSelectionView(parent, state, this);
      }
    }

    public static class ByProcess {
      public final long pid;
      public final long dur;
      public final ImmutableList<ByThread> threads;

      public ByProcess(long pid, long dur, ImmutableList<ByThread> threads) {
        this.pid = pid;
        this.dur = dur;
        this.threads = threads;
      }

      public static class Builder {
        public final long pid;
        public long dur = 0;
        public final Map<Long, ByThread.Builder> threads = Maps.newHashMap();

        public Builder(long pid) {
          this.pid = pid;
        }

        public void add(Slice slice) {
          dur += slice.dur;
          threads.computeIfAbsent(slice.utid, ByThread.Builder::new).add(slice);
        }

        public Builder combine(Builder other) {
          dur += other.dur;
          for (Map.Entry<Long, ByThread.Builder> e : other.threads.entrySet()) {
            threads.merge(e.getKey(), e.getValue(), ByThread.Builder::combine);
          }
          return this;
        }

        public ByProcess build() {
          return new ByProcess(pid, dur, threads.values().stream()
              .map(ByThread.Builder::build)
              .sorted((t1, t2) -> Long.compare(t2.dur, t1.dur))
              .collect(toImmutableList()));
        }
      }
    }

    public static class ByThread {
      public final long tid;
      public final long dur;
      public final ImmutableList<Slice> slices;

      public ByThread(long tid, long dur, ImmutableList<Slice> slices) {
        this.tid = tid;
        this.dur = dur;
        this.slices = slices;
      }

      public static class Builder {
        private final long tid;
        private long dur = 0;
        private final List<Slice> slices = Lists.newArrayList();

        public Builder(long tid) {
          this.tid = tid;
        }

        public void add(Slice slice) {
          dur += slice.dur;
          slices.add(slice);
        }

        public Builder combine(Builder other) {
          dur += other.dur;
          slices.addAll(other.slices);
          return this;
        }

        public ByThread build() {
          Collections.sort(slices, (s1, s2) -> Long.compare(s2.dur, s1.dur));
          return new ByThread(tid, dur, slices.stream()
              .sorted((s1, s2) -> Long.compare(s2.dur, s1.dur))
              .collect(toImmutableList()));
        }
      }
    }
  }
}
