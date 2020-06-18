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

import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.FrameEventsSelectionView;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Set;
import java.util.TreeMap;
import java.util.function.Consumer;

/**
 * Data about a Surface Flinger Frame Events in the trace.
 */
// TODO: dedupe code with SliceTrack.
public class FrameEventsTrack extends Track.WithQueryEngine<FrameEventsTrack.Data>{
  private static final String BASE_COLUMNS =
      "id, ts, dur, name, depth, stack_id, parent_stack_id, arg_set_id, frame_number";
  private static final String STAT_COLUMNS =
      "queue_to_acquire_time, acquire_to_latch_time, latch_to_present_time";
  private static final String SLICE_SQL =
      "select * from " +
      "(select " + BASE_COLUMNS + " from frame_slice where id = %d) join " +
      "(select " + STAT_COLUMNS + " from frame_slice " +
        "where name = 'PresentFenceSignaled' and frame_number = " +
          "(select frame_number from frame_slice where id = %d))";
  private static final String SLICES_SQL =
       "select " + BASE_COLUMNS + " from %s " +
       "where ts >= %d - dur and ts <= %d order by ts";
  private static final String RANGE_SQL =
      "select " + BASE_COLUMNS + ", " + STAT_COLUMNS + " from " +
      "(select " + BASE_COLUMNS + " from %s) " +
      "inner join " +
      "(select frame_number, " + STAT_COLUMNS + " from frame_slice " +
          "where layer_name = '%s' and name = 'PresentFenceSignaled') " +
      "using(frame_number) " +
      "where ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";

  private static final long SIGNAL_MARGIN_NS = 10000;

  private final String layerName;
  private final String viewName;
  private final String eventName;

  public FrameEventsTrack(QueryEngine qe, String layerName, String viewName, String eventName) {
    super(qe, "sfevents_" + viewName);
    this.layerName = layerName;
    this.viewName = viewName;
    this.eventName = eventName;
  }

  public static FrameEventsTrack forFrameEvent(QueryEngine qe, String layerName,
      FrameInfo.Event event) {
    return new FrameEventsTrack(qe, layerName, event.viewName, event.name);
  }

  @Override
  protected ListenableFuture<?> initialize() {
    String window = tableName("window");
    return qe.queries(
        dropTable(window),
        createWindow(window));
  }

  @Override
  public ListenableFuture<Data> computeData(DataRequest req) {
    Window window = Window.compute(req, 5);
    return transformAsync(window.update(qe, tableName("window")), $ -> computeSlices(req));
  }

  private ListenableFuture<Data> computeSlices(DataRequest req) {
    return transformAsync(qe.query(slicesSql(req)), res ->
    transform(qe.getAllArgs(res.stream().mapToLong(r -> r.getLong(7))), args -> {
      int rows = res.getNumRows();
      Data data = new Data(req, new long[rows], new long[rows], new long[rows],
          new int[rows], new String[rows], new long[rows], new String[rows],
          new ArgSet[rows]);
      res.forEachRow((i, row) -> {
        long start = row.getLong(1);
        data.ids[i] = row.getLong(0);
        data.starts[i] = start;
        data.ends[i] = start + row.getLong(2);
        data.depths[i] = row.getInt(4);
        data.titles[i] = row.getString(3);
        data.args[i] = args.getOrDefault(row.getLong(7), ArgSet.EMPTY);
        data.frameNumbers[i] = row.getLong(8);
        data.layerNames[i] = layerName;
      });
      return data;
    }));
  }

  private String slicesSql(DataRequest req) {
    return format(SLICES_SQL, viewName, req.range.start, req.range.end);
  }

  public ListenableFuture<Slices> getSlice(long id) {
    return transformAsync(expectOneRow(qe.query(sliceSql(id))), r ->
      transform(qe.getArgs(r.getLong(7)), args -> new Slices(r, args, layerName, eventName)));
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id, id);
  }

  public ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
    // Stats are available only on PresentFenceSignaled events in frame_slice, so we need
    // to do a join to get stats.

    // Example
    // select  id, ts, dur, name, depth, stack_id, parent_stack_id, arg_set_id, frame_number,
    // queue_to_acquire_time, acquire_to_latch_time, latch_to_present_time from
    // (select id, ts, dur, name, depth, stack_id, parent_stack_id, arg_set_id, frame_number,"
    //      from NavigationBar00_APP)
    // inner join
    // (select frame_number, queue_to_acquire_time, acquire_to_latch_time, latch_to_present_time
    //      from frame_slice where layer_name = '%s' and name = 'PresentFenceSignaled')
    // using(frame_number)
    // where ts < 123 and ts + dur >= 234 and depth >= 0 and depth <= 1
    return transform(qe.query(sliceRangeSql(ts, minDepth, maxDepth)),
        r -> new Slices(r, layerName, eventName));
  }

  private String sliceRangeSql(TimeSpan ts, int minDepth, int maxDepth) {
    return format(RANGE_SQL, viewName, layerName, ts.end, ts.start, minDepth, maxDepth);
  }

  public static class Data extends Track.Data {
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final int[] depths;
    public final String[] titles;
    public final long[] frameNumbers;
    public final String[] layerNames;
    public final ArgSet[] args;

    public Data(DataRequest request, long[] ids, long[] starts, long[] ends, int[] depths,
        String[] titles, long[] frameNumbers, String[] layerNames, ArgSet[] args) {
      super(request);
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.depths = depths;
      this.titles = titles;
      this.frameNumbers = frameNumbers;
      this.layerNames = layerNames;
      this.args = args;
    }
  }

  public static class Slices implements Selection<Slices> {
    public int count = 0;
    public final List<Long> ids = Lists.newArrayList();
    public final List<Long> times = Lists.newArrayList();
    public final List<Long> durs = Lists.newArrayList();
    public final List<String> names = Lists.newArrayList();
    public final List<ArgSet> argsets = Lists.newArrayList();
    public final List<Long> frameNumbers = Lists.newArrayList();
    public final List<String> layerNames = Lists.newArrayList();
    public final List<String> eventNames = Lists.newArrayList();
    public final List<Long> queueToAcquireTimes = Lists.newArrayList();
    public final List<Long> acquireToLatchTimes = Lists.newArrayList();
    public final List<Long> latchToPresentTimes = Lists.newArrayList();
    public final Set<Long> sliceKeys = Sets.newHashSet();
    public final Set<Long> selectedFrameNumbers = Sets.newHashSet();

    public Slices(QueryEngine.Row row, ArgSet args, String layerName, String eventName) {
      add(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(3), args,
          row.getLong(8), layerName, eventName, row.getLong(9), row.getLong(10), row.getLong(11));
    }

    public Slices(QueryEngine.Result result, String layerName, String eventName) {
      result.forEachRow((i, row) -> this.add(row.getLong(0), row.getLong(1), row.getLong(2),
          row.getString(3), ArgSet.EMPTY, row.getLong(8), layerName, eventName,
          row.getLong(9), row.getLong(10), row.getLong(11)));
    }

    private void add(long id, long time, long dur, String name, ArgSet args, long frameNumber,
        String layerName, String eventName, long qaTime, long alTime, long lpTime) {
      count++;
      this.ids.add(id);
      this.times.add(time);
      this.durs.add(dur);
      this.names.add(name);
      this.argsets.add(args);
      this.frameNumbers.add(frameNumber);
      this.layerNames.add(layerName);
      this.eventNames.add(eventName);
      this.queueToAcquireTimes.add(qaTime);
      this.acquireToLatchTimes.add(alTime);
      this.latchToPresentTimes.add(lpTime);
      this.sliceKeys.add(id);
      this.selectedFrameNumbers.add(frameNumber);
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
          span.accept(new TimeSpan(times.get(i), times.get(i) +
              durs.get(i)).expand(SIGNAL_MARGIN_NS));
        }
      }
    }

    @Override
    public Slices combine(Slices other) {
      for (int i = 0; i < other.count; i++) {
        if (!this.sliceKeys.contains(other.ids.get(i))) {
          add(other.ids.get(i), other.times.get(i), other.durs.get(i), other.names.get(i),
              other.argsets.get(i), other.frameNumbers.get(i), other.layerNames.get(i),
              other.eventNames.get(i), other.queueToAcquireTimes.get(i),
              other.acquireToLatchTimes.get(i), other.latchToPresentTimes.get(i));
        }
      }
      return this;
    }

    public int getCount() {
      return count;
    }

    public Set<Long> getSelectedFrameNumbers() {
      return selectedFrameNumbers;
    }
  }

  public static Node[] organizeSlicesToNodes(Slices slices) {
    TreeMap<Long, Node> roots = Maps.newTreeMap();
    for (int i = 0; i < slices.count; i++) {
      roots.put(slices.ids.get(i), new Node(slices.names.get(i), slices.frameNumbers.get(i),
          slices.durs.get(i), slices.durs.get(i), slices.eventNames.get(i),
          slices.layerNames.get(i), slices.queueToAcquireTimes.get(i),
          slices.acquireToLatchTimes.get(i), slices.latchToPresentTimes.get(i)));
    }
    return roots.values().stream().toArray(Node[]::new);
  }

  public static class Node {
    public final String name;
    public final long frameNumber;
    public final long dur;
    public final long self;
    public final String eventName;
    public final String layerName;
    public final long queueToAcquireTime;
    public final long acquireToLatchTime;
    public final long latchToPresentTime;

    public Node(String name, long frameNumber, long dur, long self, String eventName,
        String layerName, long qaTime, long alTime, long lpTime) {
      this.name = name;
      this.frameNumber = frameNumber;
      this.dur = dur;
      this.self = self;
      this.eventName = eventName;
      this.layerName = layerName;
      this.queueToAcquireTime = qaTime;
      this.acquireToLatchTime = alTime;
      this.latchToPresentTime = lpTime;
    }
  }
}
