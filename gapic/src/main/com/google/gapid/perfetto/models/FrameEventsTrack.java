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
import static java.util.concurrent.TimeUnit.MICROSECONDS;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.FrameEventsSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.Set;
import java.util.TreeMap;
import java.util.function.Consumer;

/**
 * Data about a Surface Flinger Frame Events in the trace.
 */
// TODO: dedupe code with SliceTrack.
public class FrameEventsTrack extends Track.WithQueryEngine<FrameEventsTrack.Data>{
  private static final String BASE_COLUMNS =
      "id, ts, dur, name, depth, stack_id, parent_stack_id, arg_set_id, " +
      "frame_numbers, layer_names";
  private static final String SLICES_VIEW =
      "select " + BASE_COLUMNS + " from frame_slice where track_id = %d";
  private static final String SLICE_SQL =
      "select " + BASE_COLUMNS + " from frame_slice where id = %d";
  private static final String SLICES_SQL =
       "select " + BASE_COLUMNS + " from %s " +
       "where ts >= %d - dur and ts <= %d order by ts";
  private static final String SUMMARY_SQL =
      "select group_concat(id) ids, quantum_ts, count(*) from %s " +
      "where name = 'PresentFenceSignaled' or name GLOB '*[0-9]*'" +
      "group by quantum_ts";
  private static final String RANGE_SQL =
      "select " + BASE_COLUMNS + " from %s " +
      "where ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";
  private static final String RANGE_FOR_IDS_SQL =
      "select " + BASE_COLUMNS + " from %s where id in (%s)";
  private static final String STAT_TABLE_SQL =
      "select frame_numbers, layer_names, queue_to_acquire_time, " +
      "acquire_to_latch_time, latch_to_present_time " +
      "from frame_slice left join frame_stats " +
      "on frame_slice.id = frame_stats.slice_id " +
      "where frame_stats.slice_id = %d";

  private static final long SIGNAL_MARGIN_NS = 10000;
  private static final long FRAMELIFECYCLE_QUANTIZE_CUTOFF = MICROSECONDS.toNanos(500);

  private final long trackId;

  public FrameEventsTrack(QueryEngine qe, long trackId) {
    super(qe, "sfevents_" + trackId);
    this.trackId = trackId;
  }

  public static FrameEventsTrack forBuffer(QueryEngine qe, FrameInfo.Buffer buffer) {
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
    Window window = Window.compute(req, 5, FRAMELIFECYCLE_QUANTIZE_CUTOFF);
    return transformAsync(window.update(qe, tableName("window")),
        $ -> window.quantized ? computeSummary(req, window) : computeSlices(req));
  }

  private ListenableFuture<Data> computeSlices(DataRequest req) {
    return transformAsync(qe.query(slicesSql(req)), res ->
    transform(qe.getAllArgs(res.stream().mapToLong(r -> r.getLong(7))), args -> {
      int rows = res.getNumRows();
      Data data = new Data(req, new long[rows], new long[rows], new long[rows],
          new String[rows], new long[rows][], new String[rows][], new ArgSet[rows]);
      res.forEachRow((i, row) -> {
        long start = row.getLong(1);
        data.ids[i] = row.getLong(0);
        data.starts[i] = start;
        data.ends[i] = start + row.getLong(2);
        data.titles[i] = row.getString(3);
        data.args[i] = args.getOrDefault(row.getLong(7), ArgSet.EMPTY);
        data.frameNumbers[i] = Arrays.stream(row.getString(8).split(", "))
            .mapToLong(s -> s.isEmpty() ? 0 : Long.parseLong(s))
            .toArray();
        data.layerNames[i] = row.getString(9).split(", ");
      });
      return data;
    }));
  }

  private String slicesSql(DataRequest req) {
    return format(SLICES_SQL, tableName("slices"), req.range.start, req.range.end);
  }

  private ListenableFuture<Data> computeSummary(DataRequest req, Window w) {
    return transform(qe.query(summarySql()), result -> {
      int len = w.getNumberOfBuckets();
      String[] concatedIds = new String[len];
      Arrays.fill(concatedIds, "");
      Data data = new Data(req, w.bucketSize, concatedIds, new long[len]);
      result.forEachRow(($, r) -> {
        data.concatedIds[r.getInt(1)] = r.getString(0);
        data.numEvents[r.getInt(1)] = r.getLong(2);
      });
      return data;
    });
  }

  private String summarySql() {
    return format(SUMMARY_SQL, tableName("span"));
  }

  public ListenableFuture<Slices> getSlice(long id) {
    return transformAsync(expectOneRow(qe.query(sliceSql(id))), r ->
        transformAsync(qe.getArgs(r.getLong(7)), args ->
        transform(getStats(id, r.getLong(2)), stats -> new Slices(r, args, stats))));
  }

  private ListenableFuture<Map<String, FrameStats>> getStats(long id, long dur) {
    if (dur == 0) { // No stats for instant events
      return Futures.immediateFuture(null);
    }
    return transform(qe.query(statSql(id)), result -> {
      Map<String, FrameStats> stats = Maps.newHashMap();
      result.forEachRow((i, row) -> {
        stats.put(row.getString(1).split(", ")[i],
            new FrameStats(Long.parseLong(row.getString(0).split(", ")[i]),
                row.getLong(2), row.getLong(3), row.getLong(4)));
      });
      return stats;
    });
  }

  private static String statSql(long sliceId) {
    return format(STAT_TABLE_SQL, sliceId);
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id);
  }

  public ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
    return transform(qe.query(sliceRangeSql(ts, minDepth, maxDepth)), Slices::new);
  }

  private String sliceRangeSql(TimeSpan ts, int minDepth, int maxDepth) {
    return format(RANGE_SQL, tableName("slices"), ts.end, ts.start, minDepth, maxDepth);
  }

  public ListenableFuture<Slices> getSlices(String ids) {
    return transform(qe.query(sliceRangeForIdsSql(ids)), Slices::new);
  }

  private String sliceRangeForIdsSql(String ids) {
    return format(RANGE_FOR_IDS_SQL, tableName("slices"), ids);
  }

  public static class Data extends Track.Data {
    public final Kind kind;
    // Summary.
    public final long bucketSize;
    public final String[] concatedIds;
    public final long[] numEvents;
    // slices
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final String[] titles;
    public final long[][] frameNumbers;
    public final String[][] layerNames;
    public final ArgSet[] args;

    public static enum Kind {
      slices,
      summary,
    }

    public Data(DataRequest request, long bucketSize, String[] concatedIds, long[] numEvents) {
      super(request);
      this.kind = Kind.summary;
      this.bucketSize = bucketSize;
      this.concatedIds = concatedIds;
      this.numEvents = numEvents;
      this.ids = null;
      this.starts = null;
      this.ends = null;
      this.titles = null;
      this.frameNumbers = null;
      this.layerNames = null;
      this.args = null;
    }

    public Data(DataRequest request, long[] ids, long[] starts, long[] ends,
        String[] titles, long[][] frameNumbers, String[][] layerNames, ArgSet[] args) {
      super(request);
      this.kind = Kind.slices;
      this.bucketSize = 0;
      this.concatedIds = null;
      this.numEvents = null;
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.titles = titles;
      this.frameNumbers = frameNumbers;
      this.layerNames = layerNames;
      this.args = args;
    }
  }

  public static class Slices implements Selection<Slices> {
    private int count = 0;
    public final List<Long> ids = Lists.newArrayList();
    public final List<Long> times = Lists.newArrayList();
    public final List<Long> durs = Lists.newArrayList();
    public final List<String> names = Lists.newArrayList();
    public final List<ArgSet> argsets = Lists.newArrayList();
    public final List<Long[]> frameNumbers = Lists.newArrayList();
    public final List<String[]> layerNames = Lists.newArrayList();
    // Map of each buffer(layerName) that contributed to the displayed frame, to
    // its corresponding FrameStats
    public final List<Map<String, FrameStats>> frameStats = Lists.newArrayList();
    public final FrameSelection frameSelection = new FrameSelection();
    public final Set<Long> sliceKeys = Sets.newHashSet();

    public Slices(QueryEngine.Row row, ArgSet args, Map<String, FrameStats> frameStats) {
      add(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(3), args,
          row.getString(8), row.getString(9), frameStats);
    }

    public Slices(QueryEngine.Row row, Map<String, FrameStats> frameStats) {
      add(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(3), ArgSet.EMPTY,
          row.getString(8), row.getString(9), frameStats);
    }

    public Slices(QueryEngine.Result result) {
      result.forEachRow((i, row) -> this.add(row.getLong(0), row.getLong(1), row.getLong(2),
          row.getString(3), ArgSet.EMPTY, row.getString(8), row.getString(9), null));
    }

    private void add(long id, long time, long dur, String name, ArgSet args, String frameNumbers,
        String layerNames, Map<String, FrameStats> frameStats) {
      Long[] parsedFrameNumbers = Arrays.stream(frameNumbers.split(", "))
          .mapToLong(Long::parseLong).boxed().toArray(Long[]::new);
      String[] parsedLayerNames = layerNames.split(", ");
      add(id, time, dur, name, args, parsedFrameNumbers, parsedLayerNames, frameStats);
    }

    private void add(long id, long time, long dur, String name, ArgSet args, Long[] frameNumbers,
        String[] layerNames, Map<String, FrameStats> frameStats) {
      count++;
      this.ids.add(id);
      this.times.add(time);
      this.durs.add(dur);
      this.names.add(name);
      this.argsets.add(args);
      this.frameNumbers.add(frameNumbers);
      this.layerNames.add(layerNames);
      this.frameStats.add(frameStats);
      this.frameSelection.combine(new FrameSelection(frameNumbers, layerNames));
      this.sliceKeys.add(id);
    }

    public FrameSelection getSelection() {
      return frameSelection;
    }

    @Override
    public String getTitle() {
      return "Frame Events";
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
        return new FrameEventsSelectionView(parent, state, this);
      }
    }

    @Override
    public void getRange(Consumer<TimeSpan> span) {
      for (int i = 0; i < count; i++) {
        if (durs.get(i) > 0) {
          span.accept(new TimeSpan(times.get(i), times.get(i) + durs.get(i)));
        } else { // Expand the zoom/highlight time range for signal selections whose dur is 0.
          span.accept(new TimeSpan(times.get(i), times.get(i) + durs.get(i)).expand(SIGNAL_MARGIN_NS));
        }
      }
    }

    @Override
    public Slices combine(Slices other) {
      for (int i = 0; i < other.count; i++) {
        if (!this.sliceKeys.contains(other.ids.get(i))) {
          add(other.ids.get(i), other.times.get(i), other.durs.get(i), other.names.get(i),
              other.argsets.get(i), other.frameNumbers.get(i), other.layerNames.get(i),
              other.frameStats.get(i));
        }
      }
      return this;
    }

    public int getCount() {
      return count;
    }
  }

  public static class FrameSelection {
    public static FrameSelection EMPTY = new FrameSelection();
    // key = concat(layerName, '_', frameNumber)
    private Set<String> keys;

    public FrameSelection() {
      keys = Sets.newHashSet();
    }

    public FrameSelection(Long[] f, String[] l) {
      keys = Sets.newHashSet();
      for (int i = 0; i < l.length; i++) {
        keys.add(l[i] + "_" + f[i]);
      }
    }

    public boolean contains(long[] f, String[] l) {
      for (int i = 0; i < f.length; i++) {
        if (keys.contains(l[i] + "_" + f[i])) {
          return true;
        }
      }
      return false;
    }

    public void combine(FrameSelection other) {
      keys.addAll(other.keys);
    }

    public boolean isEmpty() {
      return keys.isEmpty();
    }
  }

  public static class FrameStats {
    public final long frameNumber;
    public final long queueToAcquireTime;
    public final long acquireToLatchTime;
    public final long latchToPresentTime;

    public FrameStats(long frameNumber, long queueToAcquireTime, long acquireToLatchTime,
        long latchToPresentTime) {
      this.frameNumber = frameNumber;
      this.queueToAcquireTime = queueToAcquireTime;
      this.acquireToLatchTime = acquireToLatchTime;
      this.latchToPresentTime = latchToPresentTime;
    }
  }

  public static Node[] organizeSlicesToNodes(Slices slices) {
    TreeMap<Long, Node> roots = Maps.newTreeMap();
    for (int i = 0; i < slices.count; i++) {
      roots.put(slices.ids.get(i), new Node(slices.names.get(i), slices.durs.get(i),
          slices.durs.get(i), slices.layerNames.get(i)));
    }
    return roots.values().stream().toArray(Node[]::new);
  }

  public static class Node {
    public final String name;
    public final long dur;
    public final long self;
    public final String[] layers;

    public Node(String name, long dur, long self, String[] layers) {
      this.name = name;
      this.dur = dur;
      this.self = self;
      this.layers = layers;
    }
  }
}
