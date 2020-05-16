/*
 * Copyright (C) 2020 Google Inc.
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

import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.createWindow;
import static com.google.gapid.perfetto.models.QueryEngine.dropTable;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.VulkanEventsSelectionView;

import org.eclipse.swt.widgets.Composite;

import java.util.List;
import java.util.Set;
import java.util.function.Consumer;

public class VulkanEventTrack extends Track.WithQueryEngine<VulkanEventTrack.Data> {
  private static final String BASE_COLUMNS =
      "id, ts, dur, name, depth, command_buffer, submission_id, arg_set_id";
  private static final String SLICES_VIEW =
      "select " + BASE_COLUMNS + " from gpu_slice where track_id = %d";
  private static final String SLICE_SQL =
      "select " + BASE_COLUMNS + " from gpu_slice where id = %d";
  private static final String SLICES_SQL =
      "select " + BASE_COLUMNS + " from %s " +
          "where ts >= %d - dur and ts <= %d order by ts";
  private static final String QUEUE_GROUP_START_SQL =
      "select submission_id, min(ts) start from gpu_track t left join gpu_slice s " +
          "on (t.id = s.track_id) where t.scope = 'gpu_render_stage' group by submission_id";
  private static final String SLICES_WITH_DIST_SQL =
      "with basics as (" + SLICES_SQL + ")," +
      "queue_starts as ("+ QUEUE_GROUP_START_SQL + ") " +
      "select basics.*, queue_starts.start - ts dist from basics left join queue_starts " +
          "on (basics.submission_id = queue_starts.submission_id)";
  private static final String SLICE_RANGE_SQL =
      "select " + BASE_COLUMNS + " from %s " +
          "where ts < %d and ts + dur >= %d and depth >= %d and depth <= %d";

  private final long trackId;

  public VulkanEventTrack(QueryEngine qe, GpuInfo.VkApiEvent vkApiEvent) {
    super(qe, "vk_api_events_" + vkApiEvent.trackId);
    this.trackId = vkApiEvent.trackId;
  }

  @Override
  protected ListenableFuture<?> initialize() {
    String slices = tableName("slices");
    String window = tableName("window");
    return qe.queries(
        dropView(slices),
        dropTable(window),
        createWindow(window),
        createView(slices, format(SLICES_VIEW, trackId)));
  }

  @Override
  protected ListenableFuture<Data> computeData(DataRequest req) {
    Window window = Window.compute(req);
    return transformAsync(window.update(qe, tableName("window")), $ -> computeSlices(req));
  }

  private ListenableFuture<Data> computeSlices(DataRequest req) {
    return transformAsync(qe.query(slicesSql(req)), res ->
        transform(qe.getAllArgs(res.stream().mapToLong(r -> r.getLong(7))), args -> {
          int rows = res.getNumRows();
          Data data = new Data(req, new long[rows], new long[rows], new long[rows],
              new String[rows], new int[rows], new long[rows], new long[rows], new ArgSet[rows],
              new long[rows]);
          res.forEachRow((i, row) -> {
            long start = row.getLong(1);
            data.ids[i] = row.getLong(0);
            data.starts[i] = start;
            data.ends[i] = start + row.getLong(2);
            data.names[i] = row.getString(3);
            data.depths[i] = row.getInt(4);
            data.commandBuffers[i] = row.getLong(5);
            data.submissionIds[i] = row.getLong(6);
            data.args[i] = args.getOrDefault(row.getLong(7), ArgSet.EMPTY);
            data.dists[i] = row.getLong(8);
          });
          return data;
        }));
  }

  private String slicesSql(DataRequest req) {
    return format(SLICES_WITH_DIST_SQL, tableName("slices"), req.range.start, req.range.end);
  }

  public ListenableFuture<Slices> getSlice(long id) {
    return transformAsync(expectOneRow(qe.query(sliceSql(id))), r ->
        transform(qe.getArgs(r.getLong(7)), args -> new Slices(r, args)));
  }

  private String sliceRangeSql(TimeSpan ts, int minDepth, int maxDepth) {
    return format(SLICE_RANGE_SQL, tableName("slices"), ts.end, ts.start, minDepth, maxDepth);
  }

  public ListenableFuture<Slices> getSlices(TimeSpan ts, int minDepth, int maxDepth) {
    return transform(qe.query(sliceRangeSql(ts, minDepth, maxDepth)), Slices::new);
  }

  private static String sliceSql(long id) {
    return format(SLICE_SQL, id);
  }

  public static class Data extends Track.Data {
    public final long[] ids;
    public final long[] starts;
    public final long[] ends;
    public final String[] names;
    public final int[] depths;
    public final long[] commandBuffers;
    public final long[] submissionIds;
    public final ArgSet[] args;
    public final long[] dists; // Distance between a vulkan event and the linked GPU queue events.

    public Data(DataRequest request, long[] ids, long[] starts, long[] ends, String[] names,
        int[] depths, long[] commandBuffers, long[] submissionIds, ArgSet[] args, long[] dists) {
      super(request);
      this.ids = ids;
      this.starts = starts;
      this.ends = ends;
      this.depths = depths;
      this.names = names;
      this.commandBuffers = commandBuffers;
      this.submissionIds = submissionIds;
      this.args = args;
      this.dists = dists;
    }
  }

  public static class Slices implements Selection<Slices> {
    private int count = 0;
    public final List<Long> ids = Lists.newArrayList();
    public final List<Long> times = Lists.newArrayList();
    public final List<Long> durs = Lists.newArrayList();
    public final List<String> names = Lists.newArrayList();
    public final List<Integer> depths = Lists.newArrayList();
    public final List<Long> commandBuffers = Lists.newArrayList();
    public final List<Long> submissionIds = Lists.newArrayList();
    public final List<ArgSet> argSets = Lists.newArrayList();
    private final Set<Long> submissionIdSet = Sets.newHashSet();
    public final Set<Long> sliceKeys = Sets.newHashSet();

    public Slices(QueryEngine.Row row, ArgSet argset) {
      this.add(row, argset);
    }

    public Slices(QueryEngine.Result result) {
      result.forEachRow((i, row) -> this.add(row, ArgSet.EMPTY));
    }

    private void add(QueryEngine.Row row, ArgSet argset) {
      this.add(row.getLong(0), row.getLong(1), row.getLong(2), row.getString(3), row.getInt(4),
          row.getLong(5), row.getLong(6), argset);
    }

    private void add(long id, long time, long dur, String name, int depth, long commandBuffer,
        long submissionId, ArgSet argSet) {
      this.count++;
      this.ids.add(id);
      this.times.add(time);
      this.durs.add(dur);
      this.names.add(name);
      this.depths.add(depth);
      this.commandBuffers.add(commandBuffer);
      this.submissionIds.add(submissionId);
      this.argSets.add(argSet);
      this.submissionIdSet.add(submissionId);
      this.sliceKeys.add(id);
    }

    public Set<Long> getSubmissionIds() {
      return submissionIdSet;
    }

    @Override
    public String getTitle() {
      return "Vulkan API Events";
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
        return new VulkanEventsSelectionView(parent, state, this);
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
          add(other.ids.get(i), other.times.get(i), other.durs.get(i), other.names.get(i),
              other.depths.get(i), other.commandBuffers.get(i), other.submissionIds.get(i),
              other.argSets.get(i));
        }
      }
      return this;
    }

    public int getCount() {
      return count;
    }
  }
}
