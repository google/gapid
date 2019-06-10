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

import com.google.common.collect.ImmutableList;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.ThreadStateSliceSelectionView;
import com.google.gapid.perfetto.views.ThreadStateSlicesSelectionView;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Map;

/**
 * {@link Track} containing thread state and slices of a thread.
 */
public class ThreadTrack extends Track<ThreadTrack.Data> {
  private static final String SCHED_VIEW =
      "select ts, dur, end_state, row_id from sched where utid = %d";
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
      "select ts, dur, case " +
      "  when end_state is not null then 'r'" +
      "  when lag(end_state) over ts_win is not null then lag(end_state) over ts_win" +
      "  else 'R'" +
      "end as state, row_id " +
      "from %s window ts_win as (order by ts)";

  private static final String SCHED_SQL =
      "select ts, dur, state, row_id from %s where state != 'S' and state != 'x'";
  private static final String SCHED_RANGE_SQL =
      "select ts, dur, state from %s where ts < %d and ts + dur >= %d";


  private final ThreadInfo thread;
  private final SliceTrack sliceTrack;

  public ThreadTrack(ThreadInfo thread) {
    super("thread_" + thread.utid);
    this.thread = thread;
    this.sliceTrack = SliceTrack.forThread(thread.utid);
  }

  public ThreadInfo getThread() {
    return thread;
  }

  @Override
  protected ListenableFuture<?> initialize(QueryEngine qe) {
    String wakeup = tableName("wakeup");
    String sched = tableName("sched");
    String spanJoin = tableName("span_join");
    String spanView = tableName("span_view");
    String span = tableName("span");
    String window = tableName("window");
    return transformAsync(sliceTrack.initialize(qe), $ -> qe.queries(
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
        createView(spanView, format(STATE_SPAN_VIEW, spanJoin)),
        createSpan(span, window + ", " + spanView)));
  }

  @Override
  protected ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req) {
    Window window = Window.compute(req);
    return transformAsync(sliceTrack.computeData(qe, req), slices ->
        transformAsync(window.update(qe, tableName("window")), $ ->
            computeSched(qe, req, slices)));
  }

  private ListenableFuture<Data> computeSched(
      QueryEngine qe, DataRequest req, SliceTrack.Data slices) {
    return transform(qe.query(schedSql()), res -> {
      int rows = res.getNumRows();
      Data data = new Data(req, new long[rows], new long[rows], new long[rows],
          new ThreadState[rows], slices);
      res.forEachRow((i, row) -> {
        long start = row.getLong(0);
        data.schedStarts[i] = start;
        data.schedEnds[i] = start + row.getLong(1);
        data.schedStates[i] = ThreadState.of(row.getString(2));
        data.schedIds[i] = row.getLong(3);
      });
      return data;
    });
  }

  private String schedSql() {
    return format(SCHED_SQL, tableName("span"));
  }

  public ListenableFuture<List<StateSlice>> getStates(QueryEngine qe, TimeSpan ts) {
    return transform(qe.query(stateRangeSql(ts)), res -> {
      List<StateSlice> slices = Lists.newArrayList();
      res.forEachRow((i, r) -> slices.add(new StateSlice(r, thread.utid)));
      return slices;
    });
  }

  private String stateRangeSql(TimeSpan ts) {
    return format(SCHED_RANGE_SQL, tableName("span_view"), ts.end, ts.start);
  }

  public static class Data extends Track.Data {
    // sched
    public final long[] schedIds;
    public final long[] schedStarts;
    public final long[] schedEnds;
    public final ThreadState[] schedStates;
    // slices
    public final SliceTrack.Data slices;

    public Data(DataRequest request, long[] schedIds, long[] schedStarts, long[] schedEnds,
        ThreadState[] schedStates, SliceTrack.Data slices) {
      super(request);
      this.schedIds = schedIds;
      this.schedStarts = schedStarts;
      this.schedEnds = schedEnds;
      this.schedStates = schedStates;
      this.slices = slices;
    }
  }

  public static class StateSlice implements Selection {
    public final long time;
    public final long dur;
    public final long utid;
    public final ThreadState state;

    public StateSlice(long time, long dur, long utid, ThreadState state) {
      this.time = time;
      this.dur = dur;
      this.utid = utid;
      this.state = state;
    }

    public StateSlice(QueryEngine.Row row, long utid) {
      this.time = row.getLong(0);
      this.dur = row.getLong(1);
      this.utid = utid;
      this.state = ThreadState.of(row.getString(2));
    }

    @Override
    public String getTitle() {
      return "Thread State";
    }

    @Override
    public Composite buildUi(Composite parent, State uiState) {
      return new ThreadStateSliceSelectionView(parent, uiState, this);
    }
  }

  public static class StateSlices implements Selection.CombiningBuilder.Combinable<StateSlices> {
    private final Map<ThreadState, Long> byState = Maps.newHashMap();

    public StateSlices(List<StateSlice> slices) {
      for (StateSlice slice : slices) {
        byState.compute(slice.state, (state, old) -> (old == null) ? slice.dur : old + slice.dur);
      }
    }

    @Override
    public StateSlices combine(StateSlices other) {
      for (Map.Entry<ThreadState, Long> e : other.byState.entrySet()) {
        byState.merge(e.getKey(), e.getValue(), Long::sum);
      }
      return this;
    }

    @Override
    public Selection build() {
      return new Selection(byState.entrySet().stream()
          .map(e -> new Selection.Entry(e.getKey(), e.getValue()))
          .sorted((e1, e2) -> Long.compare(e2.totalDur, e1.totalDur))
          .collect(ImmutableList.toImmutableList()));
    }

    public static class Selection implements com.google.gapid.perfetto.models.Selection {
      public final ImmutableList<Entry> entries;

      public Selection(ImmutableList<Entry> entries) {
        this.entries = entries;
      }

      @Override
      public String getTitle() {
        return "Thread States";
      }

      @Override
      public Composite buildUi(Composite parent, State state) {
        return new ThreadStateSlicesSelectionView(parent, this);
      }

      public static class Entry {
        public final ThreadState state;
        public final long totalDur;

        public Entry(ThreadState state, long totalDur) {
          this.state = state;
          this.totalDur = totalDur;
        }
      }
    }
  }
}
