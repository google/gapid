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
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.FrameEventsMultiSelectionView;
import com.google.gapid.perfetto.views.FrameEventsSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Set;
import java.util.TreeMap;

/**
 * Data about a Surface Flinger Frame Events in the trace.
 */
// TODO: dedupe code with SliceTrack.
public class FrameEventsTrack extends Track.WithQueryEngine<FrameEventsTrack.Data>{
  private static final String BASE_COLUMNS =
      "id, ts, dur, category, name, depth, stack_id, parent_stack_id, arg_set_id";
  private static final String SLICES_VIEW =
      "select " + BASE_COLUMNS + " from gpu_slice where track_id = %d";
  private static final String SLICE_SQL =
      "select " + BASE_COLUMNS + " from gpu_slice where id = %d";
  private static final String SLICES_SQL =
       "select " + BASE_COLUMNS + " from %s " +
       "where ts >= %d - dur and ts <= %d order by ts";
  private static final String SUMMARY_SQL =
      "select quantum_ts, count(*) from %s " +
      "where name = 'PresentFenceSignaled' or name GLOB '*[0-9]*'" +
      "group by quantum_ts";
  private static final String RANGE_SQL =
      "select " + BASE_COLUMNS + " from %s " +
      "where ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";

  private final long trackId;

  public FrameEventsTrack(QueryEngine qe, long trackId) {
    super(qe, "sfevents_" + trackId);
    this.trackId = trackId;
  }

  public static FrameEventsTrack forBuffer(QueryEngine qe, GpuInfo.Buffer buffer) {
    return new FrameEventsTrack(qe, buffer.trackId);
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
        createView(slices, format(SLICES_VIEW, trackId)),
        createSpan(span, window + ", " + slices + " PARTITIONED depth"));
  }

  @Override
  public ListenableFuture<Data> computeData(DataRequest req) {
    Window window = Window.compute(req, 5);
    return transformAsync(window.update(qe, tableName("window")),
        $ -> window.quantized ? computeSummary(req, window) : computeSlices(req));
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

  private String slicesSql(DataRequest req) {
    return format(SLICES_SQL, tableName("slices"), req.range.start, req.range.end);
  }

  private ListenableFuture<Data> computeSummary(DataRequest req, Window w) {
    return transform(qe.query(summarySql()), result -> {
      Data data = new Data(req, w.bucketSize, new long[w.getNumberOfBuckets()]);
      result.forEachRow(($, r) -> data.numEvents[r.getInt(0)] = r.getLong(1));
      return data;
    });
  }

  private String summarySql() {
    return format(SUMMARY_SQL, tableName("span"));
  }

  public ListenableFuture<Slice> getSlice(long id) {
    return transformAsync(expectOneRow(qe.query(sliceSql(id))), r ->
        transform(qe.getArgs(r.getLong(8)), args -> buildSlice(r, args)));
  }

  protected Slice buildSlice(QueryEngine.Row row, ArgSet args) {
    return new Slice(row, args, trackId);
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id);
  }

  public ListenableFuture<List<Slice>> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
    return transform(qe.query(sliceRangeSql(ts, minDepth, maxDepth)),
        res -> res.list(($, row) -> buildSlice(row, ArgSet.EMPTY)));
  }

  private String sliceRangeSql(TimeSpan ts, int minDepth, int maxDepth) {
    return format(RANGE_SQL, tableName("slices"), ts.end, ts.start, minDepth, maxDepth);
  }

  public static class Data extends Track.Data {
    public final Kind kind;
    // Summary.
    public final long bucketSize;
    public final long[] numEvents;
    // slices
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final int[] depths;
    public final String[] titles;
    public final String[] categories;
    public final ArgSet[] args;

    public static enum Kind {
      slices,
      summary,
    }

    public Data(DataRequest request, long bucketSize, long[] numEvents) {
      super(request);
      this.kind = Kind.summary;
      this.bucketSize = bucketSize;
      this.numEvents = numEvents;
      this.ids = null;
      this.starts = null;
      this.ends = null;
      this.depths = null;
      this.titles = null;
      this.categories = null;
      this.args = null;
    }

    public Data(DataRequest request, long[] ids, long[] starts, long[] ends, int[] depths,
        String[] titles, String[] categories, ArgSet[] args) {
      super(request);
      this.kind = Kind.slices;
      this.bucketSize = 0;
      this.numEvents = null;
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.depths = depths;
      this.titles = titles;
      this.categories = categories;
      this.args = args;
    }
  }

  public static class Slice implements Selection<Slice.Key> {
    public final long id;
    public final long time;
    public final long dur;
    public final String name;
    public final ArgSet args;
    public final long trackId;

    public Slice(long id, long time, long dur, String name, long trackId) {
      this.id = id;
      this.time = time;
      this.dur = dur;
      this.name = name;
      this.args = ArgSet.EMPTY;
      this.trackId = trackId;
    }

    public Slice(long id, long time, long dur, String name, ArgSet args, long trackId) {
      this.id = id;
      this.time = time;
      this.dur = dur;
      this.name = name;
      this.args = args;
      this.trackId = trackId;
    }

    public Slice(QueryEngine.Row row, ArgSet args, long trackId) {
      this(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(4), args, trackId);
    }

    public Slice(QueryEngine.Row row, long trackId) {
      this(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(4), trackId);
    }

    @Override
    public String getTitle() {
      return "Frame Events";
    }

    @Override
    public boolean contains(Slice.Key key) {
      return key.matches(this);
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new FrameEventsSelectionView(parent, state, this);
    }

    @Override
    public Selection.Builder<SlicesBuilder> getBuilder() {
      return new SlicesBuilder(Lists.newArrayList(this));
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

    public static class Key {
      public final long time;
      public final long dur;

      public Key(long time, long dur) {
        this.time = time;
        this.dur = dur;
      }

      public Key(Slice slice) {
        this(slice.time, slice.dur);
      }

      public boolean matches(Slice slice) {
        return slice.time == time && slice.dur == dur;
      }

      @Override
      public boolean equals(Object obj) {
        if (obj == this) {
          return true;
        } else if (!(obj instanceof Key)) {
          return false;
        }
        Key o = (Key)obj;
        return time == o.time && dur == o.dur;
      }

      @Override
      public int hashCode() {
        return Long.hashCode(time ^ dur);
      }
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
      return new FrameEventsMultiSelectionView(parent, this);
    }

    @Override
    public Selection.Builder<SlicesBuilder> getBuilder() {
      return new SlicesBuilder(slices);
    }
  }

  public static class SlicesBuilder implements Selection.Builder<SlicesBuilder> {
    private final List<Slice> slices;
    private final String title;
    private final TreeMap<Long, Node> roots = Maps.newTreeMap();
    private final Set<Slice.Key> sliceKeys = Sets.newHashSet();

    public SlicesBuilder(List<Slice> slices) {
      this.slices = slices;
      String ti = "";
      for (Slice slice : slices) {
        ti = slice.getTitle();
        roots.put(slice.id, new Node(slice.name, slice.dur, slice.dur, slice.trackId));
        sliceKeys.add(new Slice.Key(slice));
      }
      this.title = ti;
    }

    @Override
    public SlicesBuilder combine(SlicesBuilder other) {
      this.slices.addAll(other.slices);
      roots.putAll(other.roots);
      sliceKeys.addAll(other.sliceKeys);
      return this;
    }

    @Override
    public Selection<Slice.Key> build() {
      return new Slices(slices, title, ImmutableList.copyOf(roots.values()),
          ImmutableSet.copyOf(sliceKeys));
    }
  }

  public static class Node {
    public final String name;
    public final long dur;
    public final long self;
    public final long trackId;

    public Node(String name, long dur, long self, long id) {
      this.name = name;
      this.dur = dur;
      this.self = self;
      this.trackId = id;
    }
  }
}
