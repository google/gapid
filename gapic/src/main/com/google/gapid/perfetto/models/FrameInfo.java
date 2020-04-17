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
public class FrameInfo {
  public static final FrameInfo NONE = new FrameInfo(Collections.emptyList());

  private static final String FRAME_SLICE_QUERY =
      "select t.id, t.name, t.scope, max(depth) + 1 " +
      "from gpu_track t join frame_slice s on (t.id = s.track_id) " +
      "group by t.id " +
      "order by t.id";
  private static final String DISPLAYED_FRAME_TRACK_NAME = "Displayed Frame";

  private final List<Buffer> buffers;

  private FrameInfo(List<Buffer> buffers) {
    this.buffers = buffers;
  }

  public boolean isEmpty() {
    return buffers.isEmpty();
  }

  public int bufferCount() {
    return buffers.size();
  }

  public Iterable<Buffer> buffers() {
    return Iterables.unmodifiableIterable(buffers);
  }

  public static ListenableFuture<Perfetto.Data.Builder> listFrames(Perfetto.Data.Builder data) {
    return transform(info(data.qe), frame -> data.setFrame(frame));
  }

  private static ListenableFuture<FrameInfo> info(QueryEngine qe) {
    return
        transform(qe.queries(FRAME_SLICE_QUERY), frame -> {
            List<Buffer> buffers = Lists.newArrayList();
            frame.forEachRow(($, r) -> {
              buffers.add(new Buffer(r));
            });

            // Sort buffers by name, the query is sorted by track id.
            buffers.sort((b1, b2) -> {
              // Displayed Frame track should always be at top
              if (b1.name.equals(DISPLAYED_FRAME_TRACK_NAME)) {
                return -1;
              } else if(b2.name.equals(DISPLAYED_FRAME_TRACK_NAME)) {
                return 1;
              }
              return b1.name.compareTo(b2.name);
            });
            return new FrameInfo(buffers);
        });
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
