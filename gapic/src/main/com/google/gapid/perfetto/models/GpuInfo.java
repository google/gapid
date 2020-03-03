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

import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.collect.Iterables;
import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;

import java.util.Collections;
import java.util.List;

/**
 * Data about a GPU in the trace.
 */
public class GpuInfo {
  public static final GpuInfo NONE = new GpuInfo(Collections.emptyList(), Collections.emptyList(),
      Collections.emptyList());

  private static final String MAX_DEPTH_QUERY =
      "select t.id, t.name, t.scope, max(depth) + 1 " +
      "from gpu_track t left join gpu_slice s on (t.id = s.track_id) " +
      "group by t.id " +
      "order by t.id";

  private final List<Queue> queues;
  private final List<VkApiEvent> vkApiEvents;
  private final List<Buffer> buffers;

  private GpuInfo(List<Queue> queues, List<VkApiEvent> vkApiEvents, List<Buffer> buffers) {
    this.queues = queues;
    this.vkApiEvents = vkApiEvents;
    this.buffers = buffers;
  }

  public boolean isEmpty() {
    return queues.isEmpty() && buffers.isEmpty();
  }

  public int queueCount() {
    return queues.size();
  }

  public Iterable<Queue> queues() {
    return Iterables.unmodifiableIterable(queues);
  }

  public int vkApiEventCount() {
    return vkApiEvents.size();
  }

  public Iterable<VkApiEvent> vkApiEvents() {
    return Iterables.unmodifiableIterable(vkApiEvents);
  }

  public int bufferCount() {
    return buffers.size();
  }

  public Iterable<Buffer> buffers() {
    return Iterables.unmodifiableIterable(buffers);
  }

  public static ListenableFuture<Perfetto.Data.Builder> listGpus(Perfetto.Data.Builder data) {
    return transform(info(data.qe), gpu -> data.setGpu(gpu));
  }

  private static ListenableFuture<GpuInfo> info(QueryEngine qe) {
    return transform(qe.queries(MAX_DEPTH_QUERY), res -> {
      List<Queue> queues = Lists.newArrayList();
      List<VkApiEvent> vkApiEvents = Lists.newArrayList();
      List<Buffer> buffers = Lists.newArrayList();
      res.forEachRow(($, r) -> {
        switch (r.getString(2)) {
          case "gpu_render_stage":
            queues.add(new Queue(r));
            break;
          case "vulkan_events":
            vkApiEvents.add(new VkApiEvent(r));
            break;
          case "graphics_frame_event":
            buffers.add(new Buffer(r));
            break;
        }
      });

      // Sort buffers by name, the query is sorted by track id for the queues.
      buffers.sort((b1, b2) -> b1.name.compareTo(b2.name));
      return new GpuInfo(queues, vkApiEvents, buffers);
    });
  }

  public static class Queue {
    public final String name;
    public final long trackId;
    public final int maxDepth;

    public Queue(long trackId, String name, int maxDepth) {
      this.trackId = trackId;
      this.name = name;
      this.maxDepth = maxDepth;
    }

    public Queue(QueryEngine.Row row) {
      this(row.getLong(0), row.getString(1), row.getInt(3));
    }

    public String getDisplay() {
      return name;
    }
  }

  public static class VkApiEvent {
    public final long trackId;
    public final String name;
    public final int maxDepth;

    public VkApiEvent(long trackId, String name, int maxDepth) {
      this.trackId = trackId;
      this.name = name;
      this.maxDepth = maxDepth;
    }

    public VkApiEvent(QueryEngine.Row row) {
      this(row.getLong(0), row.getString(1), row.getInt(3));
    }

    public String getDisplay() {
      return name;
    }
  }

  public static class Buffer {
    public final long trackId;
    public final String name;
    public final int maxDepth;

    public Buffer(long trackId, String name, int maxDepth) {
      this.trackId = trackId;
      this.name = name;
      this.maxDepth = maxDepth;
    }

    public Buffer(QueryEngine.Row row) {
      this(row.getLong(0), row.getString(1), row.getInt(3));
    }

    public String getDisplay() {
      return name;
    }
  }
}
