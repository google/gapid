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
import static com.google.gapid.perfetto.models.QueryEngine.createSpanLeftJoin;
import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.SliceTrack.Slices;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.ThreadStateSlicesSelectionView;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Consumer;

/**
 * {@link Track} containing thread state and slices of a thread.
 */
public class ThreadTrack extends Track.WithQueryEngine<ThreadTrack.Data> {
  private static final String SCHED_VIEW =
      "select ts, dur, end_state, id from sched where utid = %d";
  private static final String INSTANT_VIEW =
      "with" +
      "  events as (select ts from instants where name = 'sched_wakeup' and ref = %d)," +
      "  wakeup as (select " +
      "    min(" +
      "      coalesce((select min(ts) from events), (select end_ts from trace_bounds))," +
      "      (select min(ts) from %s)" +
      "    ) ts union select ts from events)" +
      "select ts, lead(ts, 1, (select end_ts from trace_bounds)) over (order by ts) - ts dur " +
      "from wakeup";
  private static final String STATE_SPAN_VIEW =
      "select ts, dur, case when end_state is null then false else true end is_sched, case " +
      "  when end_state is not null then 'r'" +
      "  when lag(end_state) over ts_win is not null then lag(end_state) over ts_win" +
      "  else 'R'" +
      "end as state, case " +
      "  when id is not null then id " +                       // Sched id, for slices of 'Running'.
      "  else rank() over ts_win + (select max(id) from %s)" + // Assigned unique id, for other state slices like 'Waking', 'Runnable', etc.
      "end as id " +
      "from %s window ts_win as (order by ts)";

  private static final String SCHED_SQL =
      "select ts, dur, is_sched, state, id from %s where state != 'S' and state != 'x'";
  private static final String SCHED_RANGE_SQL =
      "select ts, dur, is_sched, state, id from %s where ts < %d and ts + dur >= %d";


  private final ThreadInfo thread;
  private final SliceFetcher sliceTrack;

  public ThreadTrack(QueryEngine qe, ThreadInfo thread) {
    super(qe, "thread_" + thread.utid);
    this.thread = thread;
    this.sliceTrack = SliceFetcher.forThread(qe, thread);
  }

  public ThreadInfo getThread() {
    return thread;
  }

  @Override
  protected ListenableFuture<?> initialize() {
    String wakeup = tableName("wakeup");
    String sched = tableName("sched");
    String spanJoin = tableName("span_join");
    String spanView = tableName("span_view");
    String span = tableName("span");
    String window = tableName("window");
    return transformAsync(sliceTrack.initialize(), $ -> qe.queries(
        dropTable(span),
        dropView(spanView),
        dropTable(spanJoin),
        dropView(sched),
        dropView(wakeup),
        dropTable(window),
        createWindow(window),
        createView(sched, format(SCHED_VIEW, thread.utid)),
        createView(wakeup, format(INSTANT_VIEW, thread.utid, sched)),
        createSpanLeftJoin(spanJoin, wakeup + ", " + sched),
        createView(spanView, format(STATE_SPAN_VIEW, spanJoin, spanJoin)),
        createSpan(span, window + ", " + spanView)));
  }

  @Override
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window window = Window.compute(req);
    return transformAsync(sliceTrack.computeData(req), slices ->
        transformAsync(window.update(qe, tableName("window")), $ ->
            computeSched(req, slices)));
  }

  private ListenableFuture<Data> computeSched(DataRequest req, SliceTrack.Data slices) {
    return transform(qe.query(schedSql()), res -> {
      int rows = res.getNumRows();
      Data data = new Data(req, new boolean[rows], new long[rows], new long[rows], new long[rows],
          new ThreadState[rows], slices);
      res.forEachRow((i, row) -> {
        long start = row.getLong(0);
        data.schedStarts[i] = start;
        data.schedEnds[i] = start + row.getLong(1);
        data.isSched[i] = row.getInt(2) != 0;
        data.schedStates[i] = ThreadState.of(row.getString(3));
        data.ids[i] = row.getLong(4);
      });
      return data;
    });
  }

  private String schedSql() {
    return format(SCHED_SQL, tableName("span"));
  }

  public ListenableFuture<Slices> getSlice(long id) {
    return sliceTrack.getSlice(id);
  }

  public ListenableFuture<CpuTrack.Slices> getCpuSlice(long id) {
    return CpuTrack.getSlice(qe, id);
  }

  public ListenableFuture<Slices> getSlices(String concatedId) {
    return sliceTrack.getSlices(concatedId);
  }

  public ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
    return sliceTrack.getSlices(ts, minDepth, maxDepth);
  }

  public ListenableFuture<CpuTrack.Slices> getCpuSlices(TimeSpan ts) {
    return CpuTrack.getSlices(qe, thread.utid, ts);
  }

  public ListenableFuture<StateSlices> getStates(TimeSpan ts) {
    return transform(qe.query(stateRangeSql(ts)), res -> new StateSlices(res, thread.utid));
  }

  private String stateRangeSql(TimeSpan ts) {
    return format(SCHED_RANGE_SQL, tableName("span_view"), ts.end, ts.start);
  }

  public static class Data extends Track.Data {
    // sched
    public final boolean[] isSched;
    public final long[] ids; // Sched id for sched slice, generated unique id for other state slice.
    public final long[] schedStarts;
    public final long[] schedEnds;
    public final ThreadState[] schedStates;
    // slices
    public final SliceTrack.Data slices;

    public Data(DataRequest request, boolean[] isSched, long[] ids, long[] schedStarts, long[] schedEnds,
        ThreadState[] schedStates, SliceTrack.Data slices) {
      super(request);
      this.isSched = isSched;
      this.ids = ids;
      this.schedStarts = schedStarts;
      this.schedEnds = schedEnds;
      this.schedStates = schedStates;
      this.slices = slices;
    }
  }

  public static class StateSlices implements Selection<StateSlices> {
    private int count = 0;
    public final List<Long> ids = Lists.newArrayList();
    public final List<Long> times = Lists.newArrayList();
    public final List<Long> durs = Lists.newArrayList();
    public final List<Long> utids = Lists.newArrayList();
    public final List<Boolean> isScheds = Lists.newArrayList();
    public final List<ThreadState> states = Lists.newArrayList();
    public final Set<Long> sliceKeys = Sets.newHashSet();

    public StateSlices(long time, long dur, long utid, boolean isSched, ThreadState state, long id) {
      this.add(id, time, dur, utid, isSched, state);
    }

    public StateSlices(QueryEngine.Row row, long utid) {
      this.add(row.getLong(4), row.getLong(0), row.getLong(1), utid, row.getInt(2) != 0,
          ThreadState.of(row.getString(3)));
    }

    public StateSlices(QueryEngine.Result result, long utid) {
      result.forEachRow((i, row) -> this.add(row.getLong(4), row.getLong(0), row.getLong(1), utid, row.getInt(2) != 0,
          ThreadState.of(row.getString(3))));
    }

    private void add(long id, long time, long dur, long utid, boolean isSched, ThreadState state) {
      this.count++;
      this.ids.add(id);
      this.times.add(time);
      this.durs.add(dur);
      this.utids.add(utid);
      this.isScheds.add(isSched);
      this.states.add(state);
      this.sliceKeys.add(id);
    }

    @Override
    public String getTitle() {
      return "Thread States";
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
        return new ThreadStateSlicesSelectionView(parent, state, this);
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
    public StateSlices combine(StateSlices other) {
      for (int i = 0; i < other.count; i++) {
        if (!this.sliceKeys.contains(other.ids.get(i))) {
          add(other.ids.get(i), other.times.get(i), other.durs.get(i), other.utids.get(i),
              other.isScheds.get(i), other.states.get(i));
        }
      }
      return this;
    }

    public int getCount() {
      return count;
    }
  }

  public static Entry[] organizeSlicesToEntry(StateSlices slices) {
    Map<ThreadState, Map<Long, Long>> byState = Maps.newHashMap(); // state -> (slice_id -> dur)
    for (int i = 0; i < slices.count; i++) {
      byState.putIfAbsent(slices.states.get(i), Maps.newHashMap());
      byState.get(slices.states.get(i)).put(slices.ids.get(i), slices.durs.get(i));
    }
    return byState.entrySet().stream()
        .map(e -> new Entry(e.getKey(), e.getValue().values().stream().mapToLong(v -> v).sum()))
        .sorted((e1, e2) -> Long.compare(e2.totalDur, e1.totalDur))
        .toArray(Entry[]::new);
  }

  public static class Entry {
    public final ThreadState state;
    public final long totalDur;

    public Entry(ThreadState state, long totalDur) {
      this.state = state;
      this.totalDur = totalDur;
    }
  }

  private static interface SliceFetcher {
    public static final SliceFetcher NONE = new SliceFetcher() { /* empty */ };

    @SuppressWarnings("unused")
    public default ListenableFuture<?> initialize() {
      return Futures.immediateFuture(null);
    }

    @SuppressWarnings("unused")
    public default ListenableFuture<SliceTrack.Data> computeData(DataRequest req) {
      return Futures.immediateFuture(new SliceTrack.Data(req));
    }

    @SuppressWarnings("unused")
    public default ListenableFuture<Slices> getSlice(long id) {
      throw new UnsupportedOperationException();
    }

    @SuppressWarnings("unused")
    public default ListenableFuture<Slices> getSlices(String concatedId) {
      return Futures.immediateFuture(null);
    }

    @SuppressWarnings("unused")
    public default ListenableFuture<Slices> getSlices(
        TimeSpan ts, int minDepth, int maxDepth) {
      return Futures.immediateFuture(null);
    }

    public static SliceFetcher forThread(QueryEngine q, ThreadInfo thread) {
      if (thread.trackId < 0) {
        return SliceFetcher.NONE;
      }

      SliceTrack track = SliceTrack.forThread(q, thread);
      return new SliceFetcher() {
        @Override
        public ListenableFuture<?> initialize() {
          return track.initialize();
        }

        @Override
        public ListenableFuture<SliceTrack.Data> computeData(DataRequest req) {
          return track.computeData(req);
        }

        @Override
        public ListenableFuture<Slices> getSlice(long id) {
          return track.getSlice(id);
        }

        @Override
        public ListenableFuture<Slices> getSlices(String concatedId) {
          return track.getSlices(concatedId);
        }

        @Override
        public ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
          return track.getSlices(ts, minDepth, maxDepth);
        }
      };
    }
  }
}
