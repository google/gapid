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
import com.google.common.collect.ImmutableSet;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.QueryEngine.Row;
import com.google.gapid.perfetto.views.SliceSelectionView;
import com.google.gapid.perfetto.views.SlicesSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Consumer;

/**
 * {@link Track} containing slices.
 */
public abstract class SliceTrack extends Track<SliceTrack.Data> {/*extends Track.WithQueryEngine<SliceTrack.Data>*/
  protected SliceTrack(long trackId) {
    super("slices_" + trackId);
  }

  public static SliceTrack forThread(QueryEngine qe, ThreadInfo thread) {
    return new WithQueryEngine(qe, "slice", thread.trackId) {
      @Override
      protected Slice buildSlice(Row row, ArgSet args) {
        return new Slice.ThreadSlice(row, args, thread);
      }
    };
  }

  public static SliceTrack forGpuQueue(QueryEngine qe, GpuInfo.Queue queue) {
    return new WithQueryEngine(qe, "gpu_slice", queue.trackId) {
      // TODO(b/148540258): Remove the copy pasted SliceTrack code and clean up
      private final String GPU_COLUMNS = "render_target, render_target_name, render_pass, render_pass_name, command_buffer, command_buffer_name, submission_id";
      private final String GPU_SLICES_QUANT_SQL =
          "select min(start_ts), max(end_ts), depth, label, max(cnt), " +
          "    first_value(submission_id) over (partition by depth, label, i)  from (" +
          "  select quantum_ts, start_ts, end_ts, depth, label, count(1) cnt, " +
          "      quantum_ts-row_number() over (partition by depth, label order by quantum_ts) i, " +
          "      submission_id from (" +
          "    select quantum_ts, min(ts) over win1 start_ts, max(ts + dur) over win1 end_ts, depth, " +
          "        substr(group_concat(name) over win1, 0, 101) label, " +
          "        first_value(submission_id) over win1 submission_id" +
          "    from %s" +
          "    window win1 as (partition by quantum_ts, depth order by dur desc" +
          "    range between unbounded preceding and unbounded following))" +
          "  group by quantum_ts, depth)" +
          "group by depth, label, i";

      @Override
      protected String baseColumns() {
        return BASE_COLUMNS + ", " + GPU_COLUMNS;
      }

      @Override
      protected String slicesQuantSql() {
        return format(GPU_SLICES_QUANT_SQL, tableName("span"));
      }

      @Override
      protected void appendForQuant(Data data, QueryEngine.Result res) {
        data.putExtraLongs("submissionIds", res.stream().mapToLong(r -> r.getLong(5)).toArray());
      }

      @Override
      protected ListenableFuture<Data> computeData(DataRequest req) {
        Window window = Window.compute(req, 5);
        return transformAsync(window.update(qe, tableName("window")), $ ->
            window.quantized ? computeQuantSlices(req) : computeSlices(req));
      }

      private ListenableFuture<Data> computeSlices(DataRequest req) {
        return transformAsync(qe.query(slicesSql(req)), res ->
          transform(qe.getAllArgs(res.stream().mapToLong(r -> r.getLong(8))), args -> {
            int rows = res.getNumRows();
            Data data = new Data(req, new long[rows], new long[rows], new long[rows], new int[rows],
                new String[rows], new String[rows], new ArgSet[rows]);
            long[] submissionIds = new long[rows];
            res.forEachRow((i, row) -> {
              long start = row.getLong(1);
              data.ids[i] = row.getLong(0);
              data.starts[i] = start;
              data.ends[i] = start + row.getLong(2);
              data.categories[i] = row.getString(3);
              data.titles[i] = row.getString(4);
              data.depths[i] = row.getInt(5);
              // Add debug marker to title if it exists
              if (data.depths[i] == 0) {
                String debugMarker = row.getString(10);
                if (!debugMarker.isEmpty()) {
                  data.titles[i] += "[" + debugMarker + "]";
                }
              }
              data.args[i] = args.getOrDefault(row.getLong(8), ArgSet.EMPTY);
              submissionIds[i] = row.getLong(15);
            });
            data.putExtraLongs("submissionIds", submissionIds);
            return data;
          }));
      }

      @Override
      protected Slice buildSlice(Row row, ArgSet args) {
        return new Slice.GpuSlice(row, args);
      }
    };
  }

  public abstract ListenableFuture<Slice> getSlice(long id);
  public abstract ListenableFuture<List<Slice>> getSlices(TimeSpan ts, int minDepth, int maxDepth);

  public static class Data extends Track.Data {
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final int[] depths;
    public final String[] titles;
    public final String[] categories;
    public final ArgSet[] args;
    public Map<String, long[]> extraLongs = Maps.newHashMap();

    public Data(DataRequest request) {
      super(request);
      this.ids = new long[0];
      this.starts = new long[0];
      this.ends = new long[0];
      this.depths = new int[0];
      this.titles = new String[0];
      this.categories = new String[0];
      this.args = new ArgSet[0];
    }

    public Data(DataRequest request, long[] ids, long[] starts, long[] ends, int[] depths,
        String[] titles, String[] categories, ArgSet[] args) {
      super(request);
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.depths = depths;
      this.titles = titles;
      this.categories = categories;
      this.args = args;
    }

    public void putExtraLongs(String s, long[] longs) {
      extraLongs.put(s, longs);
    }

    public long[] getExtraLongs(String s) {
      return extraLongs.getOrDefault(s, new long[0]);
    }
  }

  public static abstract class Slice implements Selection<Slice.Key> {
    public final long time;
    public final long dur;
    public final String category;
    public final String name;
    public final int depth;
    public final long stackId;
    public final long parentId;
    public final ArgSet args;

    public Slice(long time, long dur, String category, String name, int depth, long stackId,
        long parentId, ArgSet args) {
      this.time = time;
      this.dur = dur;
      this.category = category;
      this.name = name;
      this.depth = depth;
      this.stackId = stackId;
      this.parentId = parentId;
      this.args = args;
    }

    public Slice(QueryEngine.Row row, ArgSet args) {
      this(row.getLong(1), row.getLong(2), row.getString(3), row.getString(4), row.getInt(5),
          row.getLong(6), row.getLong(7), args);
    }

    public ThreadInfo getThread() {
      return null;
    }

    public RenderStageInfo getRenderStageInfo() {
      return null;
    }

    @Override
    public boolean contains(Slice.Key key) {
      return key.matches(this);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new SliceSelectionView(parent, state, this);
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

    public static class Key {
      public final long time;
      public final long dur;
      public final int depth;

      public Key(long time, long dur, int depth) {
        this.time = time;
        this.dur = dur;
        this.depth = depth;
      }

      public Key(Slice slice) {
        this(slice.time, slice.dur, slice.depth);
      }

      public boolean matches(Slice slice) {
        return slice.time == time && slice.dur == dur && slice.depth == depth;
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof Key)) {
          return false;
        }
        Key o = (Key)obj;
        return time == o.time && dur == o.dur && depth == o.depth;
      }

      @Override
      public int hashCode() {
        return Long.hashCode(time ^ dur) ^ Integer.hashCode(depth);
      }
    }

    public static class ThreadSlice extends Slice {
      public final ThreadInfo thread;

      public ThreadSlice(Row row, ArgSet args, ThreadInfo thread) {
        super(row, args);
        this.thread = thread;
      }

      @Override
      public String getTitle() {
        return "Thread Slices";
      }

      @Override
      public ThreadInfo getThread() {
        return thread;
      }
    }

    public static class GpuSlice extends Slice {
      private final RenderStageInfo renderStageInfo;

      public GpuSlice(Row row, ArgSet args) {
        super(row, args);
        renderStageInfo = new RenderStageInfo(row.getLong(9), row.getString(10), row.getLong(11),
            row.getString(12), row.getLong(13), row.getString(14), row.getLong(15));
      }

      @Override
      public String getTitle() {
        return "GPU Queue Events";
      }

      @Override
      public RenderStageInfo getRenderStageInfo() {
        return renderStageInfo;
      }
    }
  }

  public static class RenderStageInfo {
    public final long frameBufferHandle;
    public final String frameBufferName;
    public final long renderPassHandle;
    public final String renderPassName;
    public final long commandBufferHandle;
    public final String commandBufferName;
    public final long submissionId;

    public RenderStageInfo(long frameBufferHandle, String frameBufferName, long renderPassHandle,
        String renderPassName, long commandBufferHandle, String commandBufferName, long submissionId) {
      this.frameBufferHandle = frameBufferHandle;
      this.frameBufferName = frameBufferName;
      this.commandBufferHandle = commandBufferHandle;
      this.commandBufferName = commandBufferName;
      this.renderPassHandle = renderPassHandle;
      this.renderPassName = renderPassName;
      this.submissionId = submissionId;
    }
  }

  public static class Slices implements Selection<Slice.Key> {
    private final List<Slice> slices;
    private final String title;
    public final ImmutableList<Node> nodes;
    public final ImmutableSet<Slice.Key> sliceKeys;

    public Slices(List<Slice> slices, String title, ImmutableList<Node> nodes,
        ImmutableSet<Slice.Key> sliceKeys) {
      this.slices = slices;
      this.title = title;
      this.nodes = nodes;
      this.sliceKeys = sliceKeys;
    }

    @Override
    public String getTitle() {
      return title;
    }

    @Override
    public boolean contains(Slice.Key key) {
      return sliceKeys.contains(key);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new SlicesSelectionView(parent, this);
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
    private final String title;
    private final Map<Long, Node.Builder> byStack = Maps.newHashMap();
    private final Map<Long, List<Node.Builder>> byParent = Maps.newHashMap();
    private final Set<Long> roots = Sets.newHashSet();
    private final Set<Slice.Key> sliceKeys = Sets.newHashSet();

    public SlicesBuilder(List<Slice> slices) {
      this.slices = slices;
      String ti = "";
      for (Slice slice : slices) {
        ti = slice.getTitle();
        Node.Builder child = byStack.get(slice.stackId);
        if (child == null) {
          byStack.put(slice.stackId, child = new Node.Builder(slice.name, slice.stackId, slice.parentId));
          byParent.computeIfAbsent(slice.parentId, $ -> Lists.newArrayList()).add(child);
          roots.add(slice.parentId);
        }
        roots.remove(slice.stackId);
        child.add(slice.dur);
        sliceKeys.add(new Slice.Key(slice));
      }
      this.title = ti;
    }

    @Override
    public SlicesBuilder combine(SlicesBuilder other) {
      this.slices.addAll(other.slices);
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
      sliceKeys.addAll(other.sliceKeys);
      return this;
    }

    @Override
    public Selection<Slice.Key> build() {
      return new Slices(slices, title, roots.stream()
          .filter(not(byStack::containsKey))
          .flatMap(root -> byParent.get(root).stream())
          .map(b -> b.build(byParent))
          .sorted((n1, n2) -> Long.compare(n2.dur, n1.dur))
          .collect(toImmutableList()), ImmutableSet.copyOf(sliceKeys));
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

  private abstract static class WithQueryEngine extends SliceTrack {
    protected static final String BASE_COLUMNS =
        "id, ts, dur, category, name, depth, stack_id, parent_stack_id, arg_set_id";
    protected final String table;
    protected final long trackId;

    private final String SLICES_VIEW =
        "select " + baseColumns() + " from %s where track_id = %d";
    private final String SLICES_SQL =
        "select " + baseColumns() + " from %s " +
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

    private final String SLICE_SQL =
        "select " + baseColumns() + " from %s where id = %d";
    private final String SLICE_RANGE_SQL =
        "select " + baseColumns() + " from %s " +
        "where ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";
    private final QueryEngine qe;

    protected String baseColumns() {
      return BASE_COLUMNS;
    }

    protected void appendForQuant(Data data, QueryEngine.Result res) { /* Do nothing by default. */}

    protected WithQueryEngine(QueryEngine qe, String table, long trackId) {
      super(trackId);
      this.qe = qe;
      this.table = table;
      this.trackId = trackId;
    }

    @Override
    protected ListenableFuture<?> initialize() {
      String slices = tableName("slices");
      String window = tableName("window");
      String span = tableName("span");
      return qe.queries(
          dropTable(span),
          dropView(slices),
          dropTable(window),
          createWindow(window),
          createView(slices, format(SLICES_VIEW, table, trackId)),
          createSpan(span, window + ", " + slices + " PARTITIONED depth"));
    }

    @Override
    protected ListenableFuture<Data> computeData(DataRequest req) {
      Window window = Window.compute(req, 5);
      return transformAsync(window.update(qe, tableName("window")), $ ->
          window.quantized ? computeQuantSlices(req) : computeSlices(req));
    }

    protected ListenableFuture<Data> computeQuantSlices(DataRequest req) {
      return transform(qe.query(slicesQuantSql()), res -> {
        int rows = res.getNumRows();
        Data data = new Data(req, new long[rows], new long[rows], new long[rows], new int[rows],
            new String[rows], new String[rows], new ArgSet[rows]);
        res.forEachRow((i, row) -> {
          data.ids[i] = -1;
          data.starts[i] = row.getLong(0);
          data.ends[i] = row.getLong(1);
          data.depths[i] = row.getInt(2);
          data.categories[i] = "";
          data.titles[i] = row.getString(3);
          if (data.titles[i].length() >= 100 && row.getInt(4) > 1) {
            data.titles[i] += "...";
          }
          data.args[i] = ArgSet.EMPTY;
        });
        appendForQuant(data, res);
        return data;
      });
    }

    protected String slicesQuantSql() {
      return format(SLICES_QUANT_SQL, tableName("span"));
    }

    private ListenableFuture<Data> computeSlices(DataRequest req) {
      return transformAsync(qe.query(slicesSql(req)), res ->
        transform(qe.getAllArgs(res.stream().mapToLong(r -> r.getLong(8))), args -> {
          int rows = res.getNumRows();
          Data data = new Data(req, new long[rows], new long[rows], new long[rows], new int[rows],
              new String[rows], new String[rows], new ArgSet[rows]);
          res.forEachRow((i, row) -> {
            long start = row.getLong(1);
            data.ids[i] = row.getLong(0);
            data.starts[i] = start;
            data.ends[i] = start + row.getLong(2);
            data.categories[i] = row.getString(3);
            data.titles[i] = row.getString(4);
            data.depths[i] = row.getInt(5);
            data.args[i] = args.getOrDefault(row.getLong(8), ArgSet.EMPTY);
          });
          return data;
        }));
    }

    protected String slicesSql(DataRequest req) {
      return format(SLICES_SQL, tableName("slices"), req.range.start, req.range.end);
    }

    @Override
    public ListenableFuture<Slice> getSlice(long id) {
      return transformAsync(expectOneRow(qe.query(sliceSql(id))), r ->
          transform(qe.getArgs(r.getLong(8)), args -> buildSlice(r, args)));
    }

    private Slice buildSlice(QueryEngine.Row row) {
      return buildSlice(row, ArgSet.EMPTY);
    }

    protected abstract Slice buildSlice(QueryEngine.Row row, ArgSet args);

    private String sliceSql(long id) {
      return format(SLICE_SQL, tableName("slices"), id);
    }

    @Override
    public ListenableFuture<List<Slice>> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
      return transform(qe.query(sliceRangeSql(ts, minDepth, maxDepth)),
          res -> res.list(($, row) -> buildSlice(row)));
    }

    private String sliceRangeSql(TimeSpan ts, int minDepth, int maxDepth) {
      return format(SLICE_RANGE_SQL, tableName("slices"), ts.end, ts.start, minDepth, maxDepth);
    }
  }
}
