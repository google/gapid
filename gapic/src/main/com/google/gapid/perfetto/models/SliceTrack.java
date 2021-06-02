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
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.models.QueryEngine.Result;
import com.google.gapid.perfetto.models.QueryEngine.Row;
import com.google.gapid.perfetto.views.SlicesSelectionView;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.proto.service.Service;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.function.Consumer;

/**
 * {@link Track} containing slices.
 */
public abstract class SliceTrack extends Track<SliceTrack.Data> {/*extends Track.WithQueryEngine<SliceTrack.Data>*/
  protected SliceTrack(String id) {
    super(id);
  }

  public static SliceTrack forThread(QueryEngine qe, ThreadInfo thread) {
    return new SingleTrackWithQueryEngine(qe, "slice", thread.trackId) {
      @Override
      protected Slices buildSlices(Row row, ArgSet args) {
        return new ThreadSlices(row, args, thread);
      }

      @Override
      protected Slices buildSlices(Result result) {
        return new ThreadSlices(result);
      }
    };
  }

  public static SliceTrack forGpuQueue(QueryEngine qe, GpuInfo.Queue queue) {
    return new SingleTrackWithQueryEngine(qe, "gpu_slice", queue.trackId) {
      private final QuantizedColumn[] QUANTIZED_COLUMNS = new QuantizedColumn[] {
          QuantizedColumn.firstValue("submission_id"),
      };
      private final String[] DATA_COLUMNS = new String[] {
          "render_target", "render_target_name", "render_pass", "render_pass_name",
          "command_buffer", "command_buffer_name", "submission_id"
      };

      @Override
      protected QuantizedColumn[] getExtraQuantizedColumns() {
        return QUANTIZED_COLUMNS;
      }

      @Override
      protected String[] getExtraDataColumns() {
        return DATA_COLUMNS;
      }

      @Override
      protected void appendForQuant(Data data, QueryEngine.Result res) {
        data.putExtraLongs("submissionIds", res.stream().mapToLong(r -> r.getLong(6)).toArray());
      }

      @Override
      protected void appendForSlices(Data data, Result res) {
        int rows = res.getNumRows();
        long[] submissionIds = new long[rows];
        res.forEachRow((i, row) -> {
          submissionIds[i] = row.getLong(15);
          // Add debug marker to title if it exists
          if (data.depths[i] == 0) {
            String debugMarker = row.getString(10);
            if (!debugMarker.isEmpty()) {
              data.titles[i] += "[" + debugMarker + "]";
            }
          }
        });
        data.putExtraLongs("submissionIds", submissionIds);
      }

      @Override
      protected Slices buildSlices(Row row, ArgSet args) {
        return new GpuSlices(row, args);
      }

      @Override
      protected Slices buildSlices(Result result) {
        return new GpuSlices(result);
      }
    };
  }

  public abstract ListenableFuture<Slices> getSlice(long id);
  public abstract ListenableFuture<Slices> getSlices(String concatedId);
  public abstract ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth);

  public static class Data extends Track.Data {
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final int[] depths;
    public final String[] titles;
    public final String[] categories;
    public final ArgSet[] args;
    public Map<String, long[]> extraLongs = Maps.newHashMap();
    public Map<String, String[]> extraStrings = Maps.newHashMap();

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

    public void putExtraLongs(String name, long[] longs) {
      extraLongs.put(name, longs);
    }

    public long[] getExtraLongs(String name) {
      return extraLongs.getOrDefault(name, new long[0]);
    }

    public void putExtraStrings(String name, String[] strings) {
      extraStrings.put(name, strings);
    }

    public String[] getExtraStrings(String name) {
      return extraStrings.getOrDefault(name, new String[0]);
    }
  }

  public static class RenderStageInfo {
    public static final RenderStageInfo EMPTY = new RenderStageInfo(-1, "", -1, "", -1, "", -1);

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

  public static class Slices implements Selection<Slices> {
    private int count = 0;
    public final List<Long> ids = Lists.newArrayList();
    public final List<Long> times = Lists.newArrayList();
    public final List<Long> durs = Lists.newArrayList();
    public final List<String> categories = Lists.newArrayList();
    public final List<String> names = Lists.newArrayList();
    public final List<Integer> depths = Lists.newArrayList();
    public final List<Long> stackIds = Lists.newArrayList();
    public final List<Long> parentIds = Lists.newArrayList();
    public final List<ArgSet> argsets = Lists.newArrayList();   // So far only store non-empty argset when there's only 1 slice.
    public final Set<Long> sliceKeys = Sets.newHashSet();
    private final String title;

    public Slices(String title) {
      this.title = title;
    }

    public Slices(QueryEngine.Row row, ArgSet argset, String title) {
      this.title = title;
      this.add(row, argset);
    }

    public Slices(QueryEngine.Result result, String title) {
      this.title = title;
      result.forEachRow((i, row) -> this.add(row, ArgSet.EMPTY));
    }

    public Slices(List<Service.ProfilingData.GpuSlices.Slice> serverSlices, String title) {
      this.title = title;
      for (Service.ProfilingData.GpuSlices.Slice s : serverSlices) {
        this.add(s.getId(), s.getTs(), s.getDur(), "", s.getLabel(), s.getDepth(), -1, -1, ArgSet.EMPTY);
      }
    }

    private void add(QueryEngine.Row row, ArgSet argset) {
      this.add(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(3), row.getString(4),
          row.getInt(5), row.getLong(6), row.getLong(7), argset);
    }

    private void add(long id, long time, long dur, String category, String name, int depth,
        long stackId, long parentId, ArgSet argset) {
      this.count++;
      this.ids.add(id);
      this.times.add(time);
      this.durs.add(dur);
      this.categories.add(category);
      this.names.add(name);
      this.depths.add(depth);
      this.stackIds.add(stackId);
      this.parentIds.add(parentId);
      this.argsets.add(argset);
      this.sliceKeys.add(id);
    }

    @Override
    public String getTitle() {
      return title;
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
        return new SlicesSelectionView(parent, state, this);
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
          add(other.ids.get(i), other.times.get(i), other.durs.get(i), other.categories.get(i),
              other.names.get(i), other.depths.get(i), other.stackIds.get(i), other.parentIds.get(i),
              other.argsets.get(i));
        }
      }
      return this;
    }

    public int getCount() {
      return count;
    }
  }

  public static class ThreadSlices extends Slices {
    public final List<ThreadInfo> threads = Lists.newArrayList();

    public ThreadSlices(QueryEngine.Row row, ArgSet args, ThreadInfo thread) {
      super(row, args, "Thread Slices");
      this.threads.add(thread);
    }

    public ThreadSlices(QueryEngine.Result result) {
      super(result, "Thread Slices");
      for (int i = 0; i < result.getNumRows(); i++) {
        threads.add(ThreadInfo.EMPTY);
      }
    }

    public ThreadInfo getThreadAt(int index) {
      return index < threads.size() ? threads.get(index) : ThreadInfo.EMPTY;
    }
  }

  public static class GpuSlices extends Slices {
    private final List<RenderStageInfo> renderStageInfos = Lists.newArrayList();

    public GpuSlices(Row row, ArgSet args) {
      super(row, args, "GPU Queue Events");
      this.renderStageInfos.add(new RenderStageInfo(row.getLong(9), row.getString(10), row.getLong(11),
          row.getString(12), row.getLong(13), row.getString(14), row.getLong(15)));
    }

    public GpuSlices(QueryEngine.Result result) {
      super(result, "GPU Queue Events");
      for (int i = 0; i < result.getNumRows(); i++) {
        renderStageInfos.add(RenderStageInfo.EMPTY);
      }
    }

    public RenderStageInfo getRenderStageInfoAt(int index) {
      return index < renderStageInfos.size() ? renderStageInfos.get(index) : RenderStageInfo.EMPTY;
    }
  }

  public static Node[] organizeSlicesToNodes(Slices slices) {
    Map<Long, Node.Builder> byStack = Maps.newHashMap();
    Map<Long, List<Node.Builder>> byParent = Maps.newHashMap();
    Set<Long> roots = Sets.newHashSet();

    for (int i = 0; i < slices.getCount(); i++) {
      String name = slices.names.get(i);
      long stackId = slices.stackIds.get(i);
      long parentId = slices.parentIds.get(i);
      Node.Builder child = byStack.get(stackId);
      if (child == null) {
        byStack.put(stackId, child = new Node.Builder(name, stackId, parentId));
        byParent.computeIfAbsent(parentId, $ -> Lists.newArrayList()).add(child);
        roots.add(parentId);
      }
      roots.remove(stackId);
      child.add(slices.ids.get(i), slices.durs.get(i));
    }

    return roots.stream()
        .filter(not(byStack::containsKey))
        .flatMap(root -> byParent.get(root).stream())
        .map(b -> b.build(byParent))
        .sorted((n1, n2) -> Long.compare(n2.dur, n1.dur))
        .toArray(Node[]::new);
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
      public final long stackId;
      public final long parentId;
      private final Map<Long, Long> durs = Maps.newHashMap(); // slice_id -> slice_dur.

      public Builder(String name, long stackId, long parentId) {
        this.name = name;
        this.stackId = stackId;
        this.parentId = parentId;
      }

      public void add(long sliceId, long duration) {
        durs.put(sliceId, duration);
      }

      public Node build(Map<Long, List<Builder>> byParent) {
        long dur = durs.values().stream().mapToLong(d -> d).sum();
        int count = durs.size();
        ImmutableList<Node> cs = byParent.getOrDefault(stackId, emptyList()).stream()
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

  public abstract static class WithQueryEngine extends SliceTrack {
    private static final String SLICES_SQL =
        "select %s from %s where ts >= %d - dur and ts <= %d order by ts";
    private static final String SLICE_SQL = "select %s from %s where id = %d";
    private static final String SLICES_BY_ID_SQL = "select %s from %s where id in (%s)";
    private static final String SLICE_RANGE_SQL =
        "select %s from %s where ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";

    protected static final QuantizedColumn[] NO_QUANTIZED_COLUMNS = new QuantizedColumn[0];
    protected static final String[] NO_DATA_COLUMNS = new String[0];

    protected final QueryEngine qe;

    public WithQueryEngine(QueryEngine qe, String id) {
      super(id);
      this.qe = qe;
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
          createView(slices, getViewSql()),
          createSpan(span, window + ", " + slices + " PARTITIONED depth"));
    }

    protected abstract String getViewSql();

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
        String[] concatedIds = new String[rows];
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
          concatedIds[i] = row.getString(5);
        });
        data.putExtraStrings("concatedIds", concatedIds);
        appendForQuant(data, res);
        return data;
      });
    }

    private String slicesQuantSql() {
      QuantizedColumn[] extras = getExtraQuantizedColumns();

      StringBuilder level2 = new StringBuilder().append("select " +
          "quantum_ts, min(ts) over win1 start_ts, max(ts + dur) over win1 end_ts, depth, " +
          "substr(group_concat(name) over win1, 0, 101) label, id");
      for (QuantizedColumn qc : extras) {
        level2.append(", ").append(qc.windowed("win1")).append(" ").append(qc.name);
      }
      level2.append(" from ").append(tableName("span"))
          .append(" window win1 as (partition by quantum_ts, depth order by dur desc " +
              "range between unbounded preceding and unbounded following)");

      StringBuilder level1 = new StringBuilder().append("select " +
          "quantum_ts, start_ts, end_ts, depth, label, count(1) cnt, " +
          "quantum_ts - row_number() over win2 i, group_concat(id) id");
      for (QuantizedColumn qc : extras) {
        level1.append(", ").append(qc.name);
      }
      level1.append(" from (").append(level2).append(")")
          .append(" group by quantum_ts, depth ")
          .append(" window win2 as (partition by depth, label order by quantum_ts)");

      StringBuilder outer = new StringBuilder().append("select " +
          "min(start_ts), max(end_ts), depth, label, max(cnt), group_concat(id) id");
      for (QuantizedColumn qc : extras) {
        outer.append(", ").append(qc.windowed("win3"));
      }
      outer.append(" from (").append(level1).append(")")
          .append(" group by depth, label, i")
          .append(" window win3 as (partition by depth, label, i)");

      return outer.toString();
    }

    protected abstract QuantizedColumn[] getExtraQuantizedColumns();
    protected abstract void appendForQuant(Data data, QueryEngine.Result res);

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
          appendForSlices(data, res);
          return data;
        }));
    }

    private String slicesSql(DataRequest req) {
      return format(SLICES_SQL, columns(), tableName("slices"), req.range.start, req.range.end);
    }

    protected final String columns() {
      StringBuilder sb = new StringBuilder(
          "id, ts, dur, category, name, depth, stack_id, parent_stack_id, arg_set_id");
      for (String dc : getExtraDataColumns()) {
        sb.append(", ").append(dc);
      }
      return sb.toString();
    }

    protected abstract String[] getExtraDataColumns();
    protected abstract void appendForSlices(Data data, QueryEngine.Result res);

    @Override
    public ListenableFuture<Slices> getSlice(long id) {
      return transformAsync(expectOneRow(qe.query(sliceSql(id))), r ->
        transform(qe.getArgs(r.getLong(8)), args -> buildSlices(r, args)));
    }

    private String sliceSql(long id) {
      return format(SLICE_SQL, columns(), tableName("slices"), id);
    }

    @Override
    public ListenableFuture<Slices> getSlices(String concatedId) {
      return transform(qe.query(slicesByIdSql(concatedId)), this::buildSlices);
    }

    private String slicesByIdSql(String concatedId) {
      return format(SLICES_BY_ID_SQL, columns(), tableName("slices"), concatedId);
    }

    @Override
    public ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
      return transform(qe.query(sliceRangeSql(ts, minDepth, maxDepth)), this::buildSlices);
    }

    private String sliceRangeSql(TimeSpan ts, int minDepth, int maxDepth) {
      return format(SLICE_RANGE_SQL,
          columns(), tableName("slices"), ts.end, ts.start, minDepth, maxDepth);
    }

    protected abstract Slices buildSlices(QueryEngine.Row row, ArgSet args);
    protected abstract Slices buildSlices(QueryEngine.Result result);

    protected static abstract class QuantizedColumn {
      public final String name;

      public QuantizedColumn(String name) {
        this.name = name;
      }

      public static QuantizedColumn firstValue(String name) {
        return new QuantizedColumn(name) {
          @Override
          public String windowed(String window) {
            return "first_value(" + name + ") over " + window;
          }
        };
      }

      public abstract String windowed(String window);
    }
  }


  public abstract static class SingleTrackWithQueryEngine extends WithQueryEngine {
    private static final String VIEW_SQL = "select %s from %s where track_id = %d";

    private final String table;
    private final long trackId;

    public SingleTrackWithQueryEngine(QueryEngine qe, String table, long trackId) {
      super(qe, "slices_" + trackId);
      this.table = table;
      this.trackId = trackId;
    }

    @Override
    protected String getViewSql() {
      return format(VIEW_SQL, columns(), table, trackId);
    }

    @Override
    protected QuantizedColumn[] getExtraQuantizedColumns() {
      return NO_QUANTIZED_COLUMNS;
    }

    @Override
    protected void appendForQuant(Data data, Result res) {
      // Do nothing.
    }

    @Override
    protected String[] getExtraDataColumns() {
      return NO_DATA_COLUMNS;
    }

    @Override
    protected void appendForSlices(Data data, Result res) {
      // Do nothing.
    }
  }
}
