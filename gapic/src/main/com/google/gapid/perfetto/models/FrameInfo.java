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

import static com.google.gapid.perfetto.models.QueryEngine.createView;
import static com.google.gapid.perfetto.models.QueryEngine.dropView;
import static com.google.gapid.perfetto.models.QueryEngine.expectOneRow;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.lang.String.format;

import com.google.common.collect.Iterables;
import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;

import java.util.Collections;
import java.util.List;

/**
 * Data about a GPU in the trace.
 */
public class FrameInfo {
  public static final FrameInfo NONE = new FrameInfo(Collections.emptyList());

  private static final String LAYER_NAME_QUERY =
      "select distinct layer_name from frame_slice";
  private static final String LAYER_TRACKS_QUERY =
      "select name, track_id from gpu_track inner join " +
      "(select distinct track_id from frame_slice where layer_name = '%s') t" +
      " on gpu_track.id = t.track_id";
  private static final String TRACK_ID_QUERY =
      "select group_concat(track_id) from (%s) where name GLOB '%s'";

  // The following queries use experimental_slice_layout from perfetto. Its a function which takes
  // comma separated string of track_ids and stacks overlapping slices vertically. This is useful
  // for generating ownership phases where there can be more than one slice (not related) at the
  // same time.

  // TODO: (b/158706107)
  // Cast name as int for the phases. This is needed because experimental_slice_layout generates
  // columns only based on slice_table. frame_slice is a child of slice_table and the extra columns
  // such as frame_number and layer_name won't be generated. The alternate is doing a join on
  // frame_slice table with slice_id but that query is very expensive in that it visibly slows down
  // the loading of tracks.
  private static final String PHASE_QUERY =
      "select phase.*, cast(name as INT) as frame_number from " +
      "(select id, ts, dur, name, layout_depth as depth, stack_id, parent_stack_id, arg_set_id " +
      "from experimental_slice_layout where filter_track_ids = (%s)) phase";
  // We don't need layout_depth for display phase because slice1.end = slice2.start in this track
  // and the generator will consider this as overlapping and push slice2 to the next depth.
  private static final String DISPLAY_PHASE_QUERY =
      "select t.*, CAST(name as INT) as frame_number from " +
      "(select id, ts, dur, name, depth, stack_id, parent_stack_id, arg_set_id " +
      "from experimental_slice_layout where filter_track_ids = (%s)) t";

  private static final String BUFFERS_QUERY =
      "select track_id, name from (%s) where name GLOB '*Buffer*'";
  private static final String BUFFERS_VIEW_QUERY =
      "select frame_slice.* from frame_slice join gpu_track " +
      "on frame_slice.track_id = gpu_track.id " +
      "where gpu_track.id = %d";
  private static final String MAX_DEPTH_QUERY =
      "select max(depth) from %s";

  private static final String DISPLAY_TOOLTIP =
      "The time when from was on screen";
  private static final String APP_TOOLTIP =
      "The time from when the buffer was dequeued by the app to when it was enqueued back";
  private static final String GPU_TOOLTIP =
      "Duration for which the buffer was owned by GPU. This is the time from when the buffer " +
      "was sent to GPU to the time when GPU finishes its work on the buffer. " +
      "This *does not* mean the time GPU was working solely on the buffer during this time.";
  private static final String COMPOSITION_TOOLTIP =
      "The time from when SurfaceFlinger latched on to the buffer and sent for composition " +
      "to when it was sent to the display";

  private List<Layer> layers;

  private FrameInfo(List<Layer> layers) {
    this.layers = layers;
  }

  public int layerCount() {
    return layers.size();
  }

  public Iterable<Layer> layers() {
    return Iterables.unmodifiableIterable(layers);
  }

  public static ListenableFuture<Perfetto.Data.Builder> listFrames(Perfetto.Data.Builder data) {
    List<String> layerNames = Lists.newArrayList();
    List<Layer> layers = Lists.newArrayList();
    return transformAsync(data.qe.query(LAYER_NAME_QUERY), r -> {
      r.forEachRow(($, row) -> {
        layerNames.add(row.getString(0));
      });
      if (layerNames.isEmpty()) {
        data.setFrame(FrameInfo.NONE);
        return Futures.immediateFuture(data);
      }
      return transform(getLayers(data.qe, layers, layerNames), resultLayers -> {
        data.setFrame(new FrameInfo(resultLayers));
        return data;
      });
    });
  }

  private static ListenableFuture<List<Layer>> getLayers(QueryEngine qe, List<Layer> layers,
      List<String> layerNames) {
    List<List<Event>> phases = Lists.newArrayList();
    List<List<Event>> buffers = Lists.newArrayList();
    // The group hierarchy looks like:
    // > Layer 1
    //   - DISPLAY
    //   - APP
    //   - Wait for GPU
    //   - COMPOSITION
    //   - > Buffers
    //   -   - Buffer 1
    //   -   - ...
    //   -   - Buffer n
    // ...
    // > Layer m

    // For every layer, add the four phases first, then add the buffers.
    return transformAsync(createPhases(qe, phases, layerNames, 0), $1 -> {
      return transform(createBuffers(qe, buffers, layerNames, 0), $2 -> {
        for (int i = 0; i < layerNames.size(); i++) {
          layers.add(new Layer(layerNames.get(i), buffers.get(i), phases.get(i)));
        }
        return layers;
      });
    });
  }

  private static ListenableFuture<List<List<Event>>> createPhases(QueryEngine qe,
      List<List<Event>> phases, List<String> layerNames, int idx) {
    String layerName = layerNames.get(idx);
    // SQL tables cannot start with a numeric or contain special characters.
    // Some toast messages do not set the layer name properly.
    // For example, "#0" is a valid layer name but cannot be used directly for SQL
    String baseName = ("Layer" + layerName).replaceAll("[^A-Za-z0-9]", "");
    List<Event> currentLayerPhases = Lists.newArrayList();

    // Example:
    // create view comxxLayer1_APP as
    // select phase.*, cast(NAME as INT) as frame_number from
    //  (select id, ts, dur, name, layout_depth as depth, stack_id, parent_stack_id, arg_set_id
    //  from experimental_slice_layout where filter_track_ids =
    //      (select group_concat(track_id) from
    //          (select name, track_id from gpu_track inner join
    //              (select distinct track_id from frame_slice where layer_name = 'com.xx.Layer1') t
    //          on gpu_track.id = t.track_id);
    //      where name GLOB 'APP_*')) phase
    String displayQuery =
        displayPhaseQuery(trackIdQuery(layerTracksQuery(layerName),"Display_*"));
    String appQuery = phaseQuery(trackIdQuery(layerTracksQuery(layerName), "APP_*"));
    String gpuQuery = phaseQuery(trackIdQuery(layerTracksQuery(layerName), "GPU_*"));
    String compositionQuery = phaseQuery(trackIdQuery(layerTracksQuery(layerName), "SF_*"));
    String displayViewName = baseName + "_DISPLAY";
    String appViewName = baseName + "_APP";
    String gpuViewName = baseName + "_GPU";
    String compositionViewName = baseName + "_COMPOSITION";

    // we need to determine the max depth of these views here, so that panel creation
    // can happen appropriately.
    return transformAsync(qe.queries(
        dropView(appViewName),
        dropView(gpuViewName),
        dropView(compositionViewName),
        dropView(displayViewName),
        createView(appViewName, appQuery),
        createView(gpuViewName, gpuQuery),
        createView(compositionViewName, compositionQuery),
        createView(displayViewName, displayQuery)), $ -> {
          return transformAsync(Futures.allAsList(
              getMaxDepth(qe, displayViewName),
              getMaxDepth(qe, appViewName),
              getMaxDepth(qe, gpuViewName),
              getMaxDepth(qe, compositionViewName)), depthList -> {
                currentLayerPhases.add(new Event("On Display",baseName + "_DISPLAY", depthList.get(0) + 1,
                    DISPLAY_TOOLTIP));
                currentLayerPhases.add(new Event("Application", baseName + "_APP", depthList.get(1) + 1,
                    APP_TOOLTIP));
                currentLayerPhases.add(new Event("Wait for GPU", baseName + "_GPU", depthList.get(2) + 1,
                    GPU_TOOLTIP));
                currentLayerPhases.add(new Event("Composition", baseName + "_COMPOSITION",
                    depthList.get(3) + 1, COMPOSITION_TOOLTIP));
                phases.add(currentLayerPhases);
                if (idx + 1 >= layerNames.size()) {
                  return Futures.immediateFuture(phases);
                }
                return createPhases(qe, phases, layerNames, idx + 1);
              });
    });
  }

  private static ListenableFuture<Long> getMaxDepth(QueryEngine qe, String viewName) {
    return transform(expectOneRow(qe.query(maxDepthQuery(viewName))), r -> {
      return r.getLong(0);
    });
  }

  private static String trackIdQuery(String from, String filter) {
    return format(TRACK_ID_QUERY, from, filter);
  }

  private static String phaseQuery(String filter) {
    return format(PHASE_QUERY, filter);
  }

  private static String displayPhaseQuery(String filter) {
    return format(DISPLAY_PHASE_QUERY, filter);
  }

  private static String maxDepthQuery(String viewName) {
    return format(MAX_DEPTH_QUERY, viewName);
  }

  private static ListenableFuture<List<List<Event>>> createBuffers(QueryEngine qe,
      List<List<Event>> buffers, List<String> layerNames, int idx) {
    String l = layerNames.get(idx);
    return transformAsync(qe.query(buffersQuery(layerTracksQuery(l))), res -> {
      List<Event> currentLayerBuffers = Lists.newArrayList();
      List<Long> trackIds = Lists.newArrayList();
      List<String> trackNames = Lists.newArrayList();
      res.forEachRow(($, row) -> {
        trackIds.add(row.getLong(0));
        trackNames.add(row.getString(1));
      });
      if (trackIds.isEmpty()) {
        return Futures.immediateFuture(buffers);
      }
      return transformAsync(createBufferViews(qe, trackIds, trackNames,currentLayerBuffers, 0), $ -> {
        buffers.add(currentLayerBuffers);
        if (idx + 1 >= layerNames.size()) {
          return Futures.immediateFuture(buffers);
        }
        // Create buffers for the next layer.
        return createBuffers(qe, buffers, layerNames, idx + 1);
      });
    });
  }

  private static ListenableFuture<List<Event>> createBufferViews(QueryEngine qe, List<Long> trackIds,
      List<String> names, List<Event> buffers, int idx) {
    long trackId = trackIds.get(idx);
    return transformAsync(qe.queries(
        dropView("buffer_" + trackId),
        createView("buffer_" + trackId, buffersViewQuery(trackId))), $ -> {
          // Depth of instant events is always 1, the depth query can be avoided here.
          buffers.add(new Event(names.get(idx), "buffer_" + trackId, 1));
          if (idx + 1 >= trackIds.size()) {
            return Futures.immediateFuture(buffers);
          }
          // Create view for the next buffer.
          return createBufferViews(qe, trackIds, names, buffers, idx + 1);
    });
  }

  private static String layerTracksQuery(String layerName) {
    return format(LAYER_TRACKS_QUERY, layerName);
  }

  private static String buffersQuery(String from) {
    return format(BUFFERS_QUERY, from);
  }

  private static String buffersViewQuery(long filter) {
    return format(BUFFERS_VIEW_QUERY, filter);
  }

  public static class Layer {
    public final String layerName;
    private final List<Event> bufferEvents;
    private final List<Event> phaseEvents;

    public Layer(String layerName, List<Event> bufferEvents, List<Event> phaseEvents) {
      this.layerName = layerName;
      this.bufferEvents = bufferEvents;
      this.phaseEvents = phaseEvents;
    }

    public boolean isBufferEventsEmpty() {
      return bufferEvents.isEmpty();
    }

    public int bufferEventsCount() {
      return bufferEvents.size();
    }

    public Iterable<Event> bufferEvents() {
      return Iterables.unmodifiableIterable(bufferEvents);
    }

    public boolean isPhaseEventsEmpty() {
      return phaseEvents.isEmpty();
    }

    public int phaseEventsCount() {
      return phaseEvents.size();
    }

    public Iterable<Event> phaseEvents() {
      return Iterables.unmodifiableIterable(phaseEvents);
    }
  }

  public static class Event {
    public final String name;
    public final String viewName;
    public final long maxDepth;
    public final String tooltip;

    public Event(String name, String viewName, long maxDepth, String tooltip) {
      this.name = name;
      this.viewName = viewName;
      this.maxDepth = maxDepth;
      this.tooltip = tooltip;
    }

    public Event(String name, String viewName, long maxDepth) {
      this.name = name;
      this.viewName = viewName;
      this.maxDepth = maxDepth;
      this.tooltip = name;
    }

    public String getDisplay() {
      return name;
    }
  }
}
