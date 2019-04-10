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

import static com.google.common.base.Predicates.not;
import static com.google.common.collect.ImmutableList.toImmutableList;
import static com.google.gapid.perfetto.models.QueryEngine.createSpan;
import static com.google.gapid.perfetto.models.QueryEngine.createSpanLeftJoin;
import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;
import static java.util.Collections.emptyList;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.ThreadState;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.ThreadSliceSelectionView;
import com.google.gapid.perfetto.views.ThreadSlicesSelectionView;
import com.google.gapid.perfetto.views.ThreadStateSliceSelectionView;
import com.google.gapid.perfetto.views.ThreadStateSlicesSelectionView;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Map;
import java.util.Set;

/**
 * {@link Track} containing thread state and slices of a thread.
 */
public class ThreadTrack extends Track<ThreadTrack.Data> {
  private static final String INSTANT_VIEW =
      "select ts, lead(ts) over (order by ts) - ts dur from instants " +
      "where name = 'sched_wakeup' and ref = %d";
  private static final String SCHED_VIEW =
      "select ts, dur, end_state, row_id from sched where utid = %d";
  private static final String SLICES_VIEW =
      "select ts, dur, cat, name, depth, stack_id, parent_stack_id from slices where utid = %d";
  private static final String STATE_SPAN_VIEW =
      "select ts, dur, case " +
      "  when end_state is not null then 'r'" +
      "  when lag(end_state) over ts_win is not null then lag(end_state) over ts_win" +
      "  else 'R'" +
      "end as state, row_id " +
      "from %s window ts_win as (order by ts)";

  private static final String SCHED_SQL =
      "select ts, dur, state, row_id from %s where state != 'S'";
  private static final String SCHED_RANGE_SQL =
      "select ts, dur, state from %s where ts < %d and ts + dur >= %d";
  private static final String SLICES_SQL =
      "select ts, dur, depth, cat, name, stack_id from %s " +
      "where ts >= %d - dur and ts <= %d order by ts";
  private static final String SLICES_QUANT_SQL =
      "select min(start_ts), max(end_ts), depth, label, max(cnt) from (" +
      "  select quantum_ts, start_ts, end_ts, depth, label, count(1) cnt, " +
      "      quantum_ts-row_number() over (partition by depth, label order by quantum_ts) i from (" +
      "    select quantum_ts, min(ts) over win1 start_ts, max(ts + dur) over win1 end_ts, depth, " +
      "        substr(group_concat(name) over win1, 0, 101) label" +
      "    from %s" +
      "    window win1 as (partition by quantum_ts, depth order by dur desc" +
      "        range between unbounded preceding and unbounded following))" +
      "  group by quantum_ts, depth) " +
      "group by depth, label, i";
  private static final String SLICE_SQL =
      "select stack_id, ts, dur, utid, cat, name, parent_stack_id from slices " +
      "where stack_id = %d and ts = %d";
  private static final String SLICE_RANGE_SQL =
      "select stack_id, ts, dur, utid, cat, name, parent_stack_id from slices " +
      "where utid = %d and ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";

  private final ThreadInfo thread;

  public ThreadTrack(ThreadInfo thread) {
    super("thread_" + thread.utid);
    this.thread = thread;
  }

  public ThreadInfo getThread() {
    return thread;
  }

  @Override
  protected ListenableFuture<?> initialize(QueryEngine qe) {
    String wakeup = tableName("wakeup");
    String sched = tableName("sched");
    String slices = tableName("slices");
    String stateSpanJoin = tableName("state_span_join");
    String stateSpanView = tableName("state_span_view");
    String stateSpan = tableName("state_span");
    String sliceSpan = tableName("slice_span");
    String stateWindow = tableName("state_window");
    String sliceWindow = tableName("slice_window");
    return qe.queries(
        dropTable(stateSpan),
        dropView(stateSpanView),
        dropTable(stateSpanJoin),
        dropView(slices),
        dropView(sched),
        dropView(wakeup),
        dropTable(sliceWindow),
        dropTable(stateWindow),
        createWindow(stateWindow),
        createWindow(sliceWindow),
        createView(wakeup, format(INSTANT_VIEW, thread.utid)),
        createView(sched, format(SCHED_VIEW, thread.utid)),
        createView(slices, format(SLICES_VIEW, thread.utid)),
        createSpanLeftJoin(stateSpanJoin, wakeup + ", " + sched),
        createView(stateSpanView, format(STATE_SPAN_VIEW, stateSpanJoin)),
        createSpan(stateSpan, stateWindow + ", " + stateSpanView),
        createSpan(sliceSpan, sliceWindow + ", " + slices + " PARTITIONED depth"));
  }

  @Override
  protected ListenableFuture<Data> computeData(QueryEngine qe, DataRequest req) {
    Window stateWin = Window.compute(req);
    Window sliceWin = Window.compute(req, 5);
    return transformAsync(stateWin.update(qe, tableName("state_window")), $1 ->
      transformAsync(sliceWin.update(qe, tableName("slice_window")), $2 ->
        transformAsync(computeSched(qe), sched ->
          sliceWin.quantized ?
              computeQuantSlices(qe, req, sched) : computeSlices(qe, req, sched))));
  }

  private ListenableFuture<Sched> computeSched(QueryEngine qe) {
    return transform(qe.query(schedSql()), res -> {
      int rows = res.getNumRows();
      Sched sched =
          new Sched(new long[rows], new long[rows], new long[rows], new ThreadState[rows]);
      res.forEachRow((i, row) -> {
        long start = row.getLong(0);
        sched.starts[i] = start;
        sched.ends[i] = start + row.getLong(1);
        sched.states[i] = ThreadState.of(row.getString(2));
        sched.ids[i] = row.getLong(3);
      });
      return sched;
    });
  }

  private String schedSql() {
    return format(SCHED_SQL, tableName("state_span"));
  }

  private ListenableFuture<Data> computeQuantSlices(QueryEngine qe, DataRequest req, Sched sched) {
    return transform(qe.query(slicesQuantSql()), res -> {
      int rows = res.getNumRows();
      Data data = new Data(req, sched.ids, sched.starts, sched.ends, sched.states, new long[rows],
          new long[rows], new long[rows], new int[rows], new String[rows], new String[rows]);
      res.forEachRow((i, row) -> {
        data.starts[i] = row.getLong(0);
        data.ends[i] = row.getLong(1);
        data.depths[i] = row.getInt(2);
        data.categories[i] = "";
        data.titles[i] = row.getString(3);
        if (data.titles[i].length() >= 100 && row.getInt(4) > 1) {
          data.titles[i] += "...";
        }
      });
      return data;
    });
  }

  private String slicesQuantSql() {
    return format(SLICES_QUANT_SQL, tableName("slice_span"));
  }

  private ListenableFuture<Data> computeSlices(QueryEngine qe, DataRequest req, Sched sched) {
    return transform(qe.query(slicesSql(req)), res -> {
      int rows = res.getNumRows();
      Data data = new Data(req, sched.ids, sched.starts, sched.ends, sched.states, new long[rows],
          new long[rows], new long[rows], new int[rows], new String[rows], new String[rows]);
      res.forEachRow((i, row) -> {
        long start = row.getLong(0);
        data.starts[i] = start;
        data.ends[i] = start + row.getLong(1);
        data.depths[i] = row.getInt(2);
        data.categories[i] = row.getString(3);
        data.titles[i] = row.getString(4);
        data.ids[i] = row.getLong(5);
      });
      return data;
    });
  }

  private String slicesSql(DataRequest req) {
    return format(SLICES_SQL, tableName("slices"), req.range.start, req.range.end);
  }

  public ListenableFuture<List<StateSlice>> getStates(QueryEngine qe, TimeSpan ts) {
    return transform(qe.query(stateRangeSql(ts)), res -> {
      List<StateSlice> slices = Lists.newArrayList();
      res.forEachRow((i, r) -> slices.add(new StateSlice(r, thread.utid)));
      return slices;
    });
  }

  private String stateRangeSql(TimeSpan ts) {
    return format(SCHED_RANGE_SQL, tableName("state_span_view"), ts.end, ts.start);
  }

  public static ListenableFuture<Slice> getSlice(QueryEngine qe, long id, long ts) {
    return transform(expectOneRow(qe.query(sliceSql(id, ts))), Slice::new);
  }

  private static String sliceSql(long id, long ts) {
    return format(SLICE_SQL, id, ts);
  }

  public static ListenableFuture<List<Slice>> getSlices(
      QueryEngine qe, long utid, TimeSpan ts, int minDepth, int maxDepth) {
    return transform(qe.queries(sliceRangeSql(utid, ts, minDepth, maxDepth)), res -> {
      List<Slice> slices = Lists.newArrayList();
      res.forEachRow((i, r) -> slices.add(new Slice(r)));
      return slices;
    });
  }

  private static String sliceRangeSql(long utid, TimeSpan ts, int minDepth, int maxDepth) {
    return format(SLICE_RANGE_SQL, utid, ts.end, ts.start, minDepth, maxDepth);
  }

  public static class Data extends Track.Data {
    // sched
    public final long[] schedIds;
    public final long[] schedStarts;
    public final long[] schedEnds;
    public final ThreadState[] schedStates;
    // slices
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final int[] depths;
    public final String[] titles;
    public final String[] categories;

    public Data(DataRequest request, long[] schedIds, long[] schedStarts, long[] schedEnds,
        ThreadState[] schedStates, long[] ids, long[] starts, long[] ends, int[] depths,
        String[] titles, String[] categories) {
      super(request);
      this.schedIds = schedIds;
      this.schedStarts = schedStarts;
      this.schedEnds = schedEnds;
      this.schedStates = schedStates;
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.depths = depths;
      this.titles = titles;
      this.categories = categories;
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

  public static class Slice implements Selection {
    public final long id;
    public final long time;
    public final long dur;
    public final long utid;
    public final String category;
    public final String name;
    public final long parentId;

    public Slice(
        long id, long time, long dur, long utid, String category, String name, long parentId) {
      this.id = id;
      this.time = time;
      this.dur = dur;
      this.utid = utid;
      this.category = category;
      this.name = name;
      this.parentId = parentId;
    }

    public Slice(QueryEngine.Row row) {
      this(row.getLong(0), row.getLong(1), row.getLong(2), row.getLong(3), row.getString(4),
          row.getString(5), row.getLong(6));
    }

    @Override
    public String getTitle() {
      return "Thread Slices";
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new ThreadSliceSelectionView(parent, state, this);
    }
  }

  public static class Slices implements Selection.CombiningBuilder.Combinable<Slices> {
    private final Map<Long, Node.Builder> byStack = Maps.newHashMap();
    private final Map<Long, List<Node.Builder>> byParent = Maps.newHashMap();
    private final Set<Long> roots = Sets.newHashSet();

    public Slices(List<Slice> slices) {
      for (Slice slice : slices) {
        Node.Builder child = byStack.get(slice.id);
        if (child == null) {
          byStack.put(slice.id, child = new Node.Builder(slice.name, slice.id, slice.parentId));
          byParent.computeIfAbsent(slice.parentId, $ -> Lists.newArrayList()).add(child);
          roots.add(slice.parentId);
        }
        roots.remove(slice.id);
        child.add(slice.dur);
      }
    }

    @Override
    public Slices combine(Slices other) {
      for (Map.Entry<Long, Node.Builder> e : other.byStack.entrySet()) {
        Node.Builder mine = byStack.get(e.getKey());
        if (mine == null) {
          byStack.put(e.getKey(), mine = new Node.Builder(e.getValue()));
          byParent.computeIfAbsent(mine.parent, $ -> Lists.newArrayList()).add(mine);
        } else {
          mine.add(e.getValue());
        }
      }
      roots.addAll(other.roots);
      return this;
    }

    @Override
    public Selection build() {
      return new Selection(roots.stream()
          .filter(not(byStack::containsKey))
          .flatMap(root -> byParent.get(root).stream())
          .map(b -> b.build(byParent))
          .sorted((n1, n2) -> Long.compare(n2.dur, n1.dur))
          .collect(toImmutableList()));
    }

    public static class Selection implements com.google.gapid.perfetto.models.Selection {
      public final ImmutableList<Node> nodes;

      public Selection(ImmutableList<Node> nodes) {
        this.nodes = nodes;
      }

      @Override
      public String getTitle() {
        return "Thread Slices";
      }

      @Override
      public Composite buildUi(Composite parent, State state) {
        return new ThreadSlicesSelectionView(parent, this);
      }
    }

    public static class Node {
      public final String name;
      public final long dur;
      public final long self;
      public final int count;
      public final ImmutableList<Node> children;

      public Node(String name, long dur, long self, int count, ImmutableList<Node> children) {
        this.name = name;
        this.dur = dur;
        this.self = self;
        this.count = count;
        this.children = children;
      }

      public static class Builder {
        public final String name;
        public final long id;
        public final long parent;
        private long dur = 0;
        private int count = 0;

        public Builder(String name, long id, long parent) {
          this.name = name;
          this.id = id;
          this.parent = parent;
        }

        public Builder(Builder other) {
          this.name = other.name;
          this.id = other.id;
          this.parent = other.parent;
          this.dur = other.dur;
          this.count = other.count;
        }

        public long getParent() {
          return parent;
        }

        public void add(long duration) {
          dur += duration;
          count++;
        }

        public void add(Builder other) {
          dur += other.dur;
          count += other.count;
        }

        public Node build(Map<Long, List<Builder>> byParent) {
          ImmutableList<Node> cs = byParent.getOrDefault(id, emptyList()).stream()
              .map(b -> b.build(byParent))
              .sorted((n1, n2) -> Long.compare(n2.dur, n1.dur))
              .collect(toImmutableList());
          long cDur = cs.stream()
              .mapToLong(n -> n.dur)
              .sum();
          return new Node(name, dur, dur - cDur, count, cs);
        }
      }
    }
  }

  private static class Sched {
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final ThreadState[] states;

    public Sched(long[] ids, long[] starts, long[] ends, ThreadState[] states) {
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.states = states;
    }
  }
}
