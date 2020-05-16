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
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.CpuSlicesSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
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

  public ListenableFuture<Slices> getSlice(long id) {
    return getSlice(qe, id);
  }

  public static ListenableFuture<Slices> getSlice(QueryEngine qe, long id) {
    return transform(expectOneRow(qe.query(sliceSql(id))), Slices::new);
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id);
  }

  public ListenableFuture<Slices> getSlices(TimeSpan ts) {
    return transform(qe.query(sliceRangeSql(cpu.id, ts)), Slices::new);
  }

  public ListenableFuture<Slices> getSlices(String ids) {
    return transform(qe.query(sliceRangeForIdsSql(cpu.id, ids)), Slices::new);
  }

  public static ListenableFuture<Slices> getSlices(QueryEngine qe, long utid, TimeSpan ts) {
    return transform(qe.query(sliceRangeForThreadSql(utid, ts)), Slices::new);
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

  public static class Slices implements Selection<Slices> {
    private int count = 0;
    public final List<Long> ids = Lists.newArrayList();
    public final List<Long> times = Lists.newArrayList();
    public final List<Long> durs = Lists.newArrayList();
    public final List<Integer> cpus = Lists.newArrayList();
    public final List<Long> utids = Lists.newArrayList();
    public final List<Long> upids = Lists.newArrayList();
    public final List<ThreadState> endStates = Lists.newArrayList();
    public final List<Integer> priorities = Lists.newArrayList();
    public final Set<Long> sliceKeys = Sets.newHashSet();

    public Slices(QueryEngine.Row row) {
      this.add(row);
    }

    public Slices(QueryEngine.Result result) {
      result.forEachRow((i, row) -> this.add(row));
    }

    private void add(QueryEngine.Row row) {
      this.count++;
      this.ids.add(row.getLong(0));
      this.times.add(row.getLong(1));
      this.durs.add(row.getLong(2));
      this.cpus.add(row.getInt(3));
      this.utids.add(row.getLong(4));
      this.upids.add(row.getLong(5));
      this.endStates.add(ThreadState.of(row.getString(6)));
      this.priorities.add(row.getInt(7));
      this.sliceKeys.add(row.getLong(0));
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
      if (count <= 0) {
        return null;
      } else {
        return new CpuSlicesSelectionView(parent, state, this);
      }
    }

    @Override
    public void getRange(Consumer<TimeSpan> span) {
      for (int i = 0; i < count; i++) {
        if (durs.get(i) > 0) {
          span.accept(new TimeSpan(times.get(i), times.get(i) + durs.get(i)));
        }
      }
    }

    @Override
    public Slices combine(Slices other) {
      for (int i = 0; i < other.count; i++) {
        if (!this.sliceKeys.contains(other.ids.get(i))) {
          this.count++;
          this.ids.add(other.ids.get(i));
          this.times.add(other.times.get(i));
          this.durs.add(other.durs.get(i));
          this.cpus.add(other.cpus.get(i));
          this.utids.add(other.utids.get(i));
          this.upids.add(other.upids.get(i));
          this.endStates.add(other.endStates.get(i));
          this.priorities.add(other.priorities.get(i));
          this.sliceKeys.add(other.ids.get(i));
        }
      }
      return this;
    }

    public int getCount() {
      return count;
    }
  }

  public static ByProcess[] organizeSlicesByProcess(Slices slices) {
    Map<Long, ByProcess.Builder> processes = Maps.newHashMap();
    for (int i = 0; i < slices.count; i++) {
      processes.computeIfAbsent(slices.upids.get(i), upid -> new ByProcess.Builder(upid, slices)).add(i);
    }
    return processes.values().stream()
        .map(ByProcess.Builder::build)
        .sorted((p1, p2) -> Long.compare(p2.dur, p1.dur))
        .toArray(ByProcess[]::new);
  }

  public static class ByProcess {
    public final long pid;
    public final long dur;
    public final ImmutableList<ByThread> threads;

    public ByProcess(long pid, ImmutableList<ByThread> threads) {
      this.pid = pid;
      this.dur = threads.stream().mapToLong(t -> t.dur).sum();
      this.threads = threads;
    }

    public static class Builder {
      public final long pid;
      private final Slices slices;
      public final Map<Long, ByThread.Builder> threads = Maps.newHashMap();

      public Builder(long pid, Slices slices) {
        this.pid = pid;
        this.slices = slices;
      }

      public void add(int index) {
        threads.computeIfAbsent(slices.utids.get(index), utid -> new ByThread.Builder(utid, slices)).add(index);
      }

      public ByProcess build() {
        return new ByProcess(pid, threads.values().stream()
            .map(ByThread.Builder::build)
            .sorted((t1, t2) -> Long.compare(t2.dur, t1.dur))
            .collect(toImmutableList()));
      }
    }
  }

  public static class ByThread {
    public final long tid;
    public final long dur;
    public final Slices slices;
    public final ImmutableList<Integer> sliceIndexes;

    public ByThread(long tid, long dur, Slices slices, ImmutableList<Integer> sliceIndexes) {
      this.tid = tid;
      this.dur = dur;
      this.slices = slices;
      this.sliceIndexes = sliceIndexes;
    }

    public static class Builder {
      private final long tid;
      private long dur = 0;
      private final Slices slices;
      private final List<Integer> sliceIndexes = Lists.newArrayList();
      private final Set<Long> sliceKeys = Sets.newHashSet();

      public Builder(long tid, Slices slices) {
        this.tid = tid;
        this.slices = slices;
      }

      public void add(int index) {
        if (!sliceKeys.contains(slices.ids.get(index))) {
          dur += slices.durs.get(index);
          sliceIndexes.add(index);
          sliceKeys.add(slices.ids.get(index));
        }
      }

      public ByThread build() {
        sliceIndexes.sort((i1, i2) -> Long.compare(slices.durs.get(i2), slices.durs.get(i1)));
        return new ByThread(tid, dur, slices, sliceIndexes.stream().collect(toImmutableList()));
      }
    }
  }
}
