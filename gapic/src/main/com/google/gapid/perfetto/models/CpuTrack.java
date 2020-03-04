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
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.ImmutableSet;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.CpuSliceSelectionView;
import com.google.gapid.perfetto.views.CpuSlicesSelectionView;
import com.google.gapid.perfetto.views.State;

import java.util.Arrays;
import org.eclipse.swt.widgets.Composite;

import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Consumer;

/**
 * {@link Track} containing CPU slices for a single core.
 */
public class CpuTrack extends Track.WithQueryEngine<CpuTrack.Data> {
  private static final String SUMMARY_SQL =
      "select quantum_ts, group_concat(id) ids, sum(dur)/cast(%d as float) util " +
      "from %s where cpu = %d and utid != 0 " +
      "group by quantum_ts";
  private static final String SLICES_SQL =
      "select ts, dur, utid, id from %s where cpu = %d and utid != 0";
  private static final String SLICE_SQL =
      "select sched.id, ts, dur, cpu, utid, upid, end_state, priority " +
      "from sched left join thread using(utid) where sched.id = %d";
  private static final String SLICE_RANGE_SQL =
      "select sched.id, ts, dur, cpu, utid, upid, end_state, priority " +
      "from sched left join thread using(utid) " +
      "where cpu = %d and utid != 0 and ts < %d and ts_end >= %d";
  private static final String SLICE_RANGE_FOR_IDS_SQL =
      "select sched.id, ts, dur, cpu, utid, upid, end_state, priority " +
      "from sched left join thread using(utid) " +
      "where cpu = %d and sched.id in (%s)";
  private static final String SLICE_RANGE_FOR_THREAD_SQL =
      "select sched.id, ts, dur, cpu, utid, upid, end_state, priority " +
      "from sched left join thread using(utid) " +
      "where utid = %d and ts < %d and ts_end >= %d";

  private final CpuInfo.Cpu cpu;

  public CpuTrack(QueryEngine qe, CpuInfo.Cpu cpu) {
    super(qe, "cpu_" + cpu.id);
    this.cpu = cpu;
  }

  public CpuInfo.Cpu getCpu() {
    return cpu;
  }

  @Override
  protected ListenableFuture<?> initialize() {
    String span = tableName("span"), window = tableName("window");
    return qe.queries(
        dropTable(span),
        dropTable(window),
        createWindow(window),
        createSpan(span, "sched PARTITIONED cpu, " + window));
  }

  @Override
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window window = Window.compute(req, 10);
    return transformAsync(window.update(qe, tableName("window")),
        $ -> window.quantized ? computeSummary(req, window) : computeSlices(req));
  }

  private ListenableFuture<Data> computeSummary(DataRequest req, Window w) {
    return transform(qe.query(summarySql(w.bucketSize)), result -> {
      int len = w.getNumberOfBuckets();
      String[] concatedIds = new String[len];
      Arrays.fill(concatedIds, "");
      Data data = new Data(req, w.bucketSize, concatedIds, new double[len]);
      result.forEachRow(($, r) -> {
        data.concatedIds[r.getInt(0)] = r.getString(1);
        data.utilizations[r.getInt(0)] = r.getDouble(2);
      });
      return data;
    });
  }

  private String summarySql(long ns) {
    return format(SUMMARY_SQL, ns, tableName("span"), cpu.id);
  }

  private ListenableFuture<Data> computeSlices(DataRequest req) {
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
    return format(SLICES_SQL, tableName("span"), cpu.id);
  }

  public ListenableFuture<Slice> getSlice(long id) {
    return getSlice(qe, id);
  }

  public static ListenableFuture<Slice> getSlice(QueryEngine qe, long id) {
    return transform(expectOneRow(qe.query(sliceSql(id))), Slice::new);
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id);
  }

  public ListenableFuture<List<Slice>> getSlices(TimeSpan ts) {
    return transform(qe.query(sliceRangeSql(cpu.id, ts)), result -> {
      List<Slice> slices = Lists.newArrayList();
      result.forEachRow((i, r) -> slices.add(new Slice(r)));
      return slices;
    });
  }

  public ListenableFuture<List<Slice>> getSlices(String ids) {
    return transform(qe.query(sliceRangeForIdsSql(cpu.id, ids)), result -> {
      List<Slice> slices = Lists.newArrayList();
      result.forEachRow((i, r) -> slices.add(new Slice(r)));
      return slices;
    });
  }

  public static ListenableFuture<List<Slice>> getSlices(QueryEngine qe, long utid, TimeSpan ts) {
    return transform(qe.query(sliceRangeForThreadSql(utid, ts)), result -> {
      List<Slice> slices = Lists.newArrayList();
      result.forEachRow((i, r) -> slices.add(new Slice(r)));
      return slices;
    });
  }

  private static String sliceRangeSql(int cpu, TimeSpan ts) {
    return format(SLICE_RANGE_SQL, cpu, ts.end, ts.start);
  }

  private static String sliceRangeForIdsSql(int cpu, String ids) {
    return format(SLICE_RANGE_FOR_IDS_SQL, cpu, ids);
  }

  private static String sliceRangeForThreadSql(long utid, TimeSpan ts) {
    return format(SLICE_RANGE_FOR_THREAD_SQL, utid, ts.end, ts.start);
  }

  public static class Data extends Track.Data {
    public final Kind kind;
    // Summary.
    public final long bucketSize;
    public final String[] concatedIds;    // Concated ids for all cpu slices in a each time bucket.
    public final double[] utilizations;
    // Slice.
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final long[] utids;

    public Data(DataRequest request, long bucketSize, String[] concatedIds, double[] utilizations) {
      super(request);
      this.kind = Kind.summary;
      this.bucketSize = bucketSize;
      this.concatedIds = concatedIds;
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
      this.concatedIds = null;
      this.utilizations = null;
    }

    public static enum Kind {
      summary, slice;
    }
  }

  public static class Slice implements Selection<Long> {
    public final long id;
    public final long time;
    public final long dur;
    public final int cpu;
    public final long utid;
    public final long upid;
    public final ThreadState endState;
    public final int priority;

    public Slice(
        long id, long time, long dur, int cpu, long utid, long upid, ThreadState endState, int priority) {
      this.id = id;
      this.time = time;
      this.dur = dur;
      this.cpu = cpu;
      this.utid = utid;
      this.upid = upid;
      this.endState = endState;
      this.priority = priority;
    }

    public Slice(QueryEngine.Row row) {
      this(row.getLong(0), row.getLong(1), row.getLong(2),
          row.getInt(3), row.getLong(4), row.getLong(5),
          ThreadState.of(row.getString(6)), row.getInt(7));
    }

    @Override
    public String getTitle() {
      return "CPU Slices";
    }

    @Override
    public boolean contains(Long key) {
      return id == key;
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new CpuSliceSelectionView(parent, state, this);
    }

    @Override
    public Selection.Builder<SlicesBuilder> getBuilder() {
      return new SlicesBuilder(Lists.newArrayList(this));
    }

    @Override
    public void getRange(Consumer<TimeSpan> span) {
      if (dur > 0) {
        span.accept(new TimeSpan(time, time + dur));
      }
    }

    @Override
    public String toString() {
      return "Slice{@" + time + " +" + dur + " " + utid + " " + endState + "/" + priority + "}";
    }
  }

  public static class Slices implements Selection<Long> {
    private final List<Slice> slices;
    public final ImmutableList<ByProcess> processes;
    public final ImmutableSet<Long> sliceKeys;

    public Slices(List<Slice> slices, ImmutableList<ByProcess> processes,
        ImmutableSet<Long> sliceKeys) {
      this.slices = slices;
      this.processes = processes;
      this.sliceKeys = sliceKeys;
    }

    @Override
    public String getTitle() {
      return "CPU Slices";
    }

    @Override
    public boolean contains(Long key) {
      return sliceKeys.contains(key);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new CpuSlicesSelectionView(parent, state, this);
    }

    @Override
    public Selection.Builder<SlicesBuilder> getBuilder() {
      return new SlicesBuilder(slices);
    }

    @Override
    public void getRange(Consumer<TimeSpan> span) {
      for (Slice slice : slices) {
        slice.getRange(span);
      }
    }
  }

  public static class SlicesBuilder implements Selection.Builder<SlicesBuilder> {
    private final List<Slice> slices;
    private final Map<Long, ByProcess.Builder> processes = Maps.newHashMap();
    private final Set<Long> sliceKeys = Sets.newHashSet();

    public SlicesBuilder(List<Slice> slices) {
      this.slices = slices;
      for (Slice slice : slices) {
        processes.computeIfAbsent(slice.upid, ByProcess.Builder::new).add(slice);
        sliceKeys.add(slice.id);
      }
    }

    @Override
    public SlicesBuilder combine(SlicesBuilder other) {
      this.slices.addAll(other.slices);
      for (Map.Entry<Long, ByProcess.Builder> e : other.processes.entrySet()) {
        processes.merge(e.getKey(), e.getValue(), ByProcess.Builder::combine);
      }
      sliceKeys.addAll(other.sliceKeys);
      return this;
    }

    @Override
    public Selection<Long> build() {
      return new Slices(slices, processes.values().stream()
          .map(ByProcess.Builder::build)
          .sorted((p1, p2) -> Long.compare(p2.dur, p1.dur))
          .collect(toImmutableList()), ImmutableSet.copyOf(sliceKeys));
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
