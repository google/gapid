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
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;

import java.util.Collections;
import java.util.List;

/**
 * Data about the tracks for the Frame Events.
 */
public class FrameEvents {
  public static final FrameEvents NONE = new FrameEvents(Collections.emptyList());

  private static final String MAX_DEPTH_QUERY =
      "select t.id, max(depth) + 1, t.name " +
      "from gpu_track t, gpu_slice s " +
      "where t.id = s.track_id and t.scope = 'graphics_frame_event' " +
      "group by t.id " +
      "order by t.name";

  private final List<Buffer> buffers;

  private FrameEvents(List<Buffer> buffers) {
    this.buffers = buffers;
  }

  public Iterable<Buffer> buffers() {
    return Iterables.unmodifiableIterable(buffers);
  }

  public int bufferCount() {
    return buffers.size();
  }

  public static ListenableFuture<Perfetto.Data.Builder> listFrameTracks(Perfetto.Data.Builder data) {
    return transform(frameTracks(data.qe), buffers -> {
      return data.setFrameEvents(new FrameEvents(buffers));
    });
  }

  private static ListenableFuture<List<Buffer>> frameTracks(QueryEngine qe) {
    return transform(qe.queries(MAX_DEPTH_QUERY), res -> res.list(Buffer::new));
  }

  public static class Buffer {
    public final int id;
    public final long trackId;
    public final int maxDepth;
    public final String name;

    public Buffer(int id, long trackId, int maxDepth, String name) {
      this.id = id;
      this.trackId = trackId;
      this.maxDepth = maxDepth;
      this.name = name;
    }

    public Buffer(int id, QueryEngine.Row row) {
      this(id, row.getLong(0), row.getInt(1), row.getString(2));
    }

    public String getDisplay() {
      return name;
    }
  }
}
