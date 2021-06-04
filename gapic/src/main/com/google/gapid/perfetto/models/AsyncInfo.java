/*
 * Copyright (C) 2021 Google Inc.
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

import com.google.common.collect.ImmutableListMultimap;
import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;

import java.util.List;

public class AsyncInfo {
  private static final String LIST_SQL =
      "select t.upid, t.name, t.id, max(s.depth) + 1 " +
      "from process_track t inner join slice s on s.track_id = t.id " +
      "group by t.upid, t.name, t.id " +
      "order by t.upid, t.name, t.id";

  public final long upid;
  public final String name;
  public final int maxDepth;
  public final Track[] tracks;

  public AsyncInfo(long upid, String name, int maxDepth, Track[] tracks) {
    this.upid = upid;
    this.name = name;
    this.maxDepth = maxDepth;
    this.tracks = tracks;
  }

  public static ListenableFuture<Perfetto.Data.Builder> listAsync(Perfetto.Data.Builder data) {
    return transform(data.qe.query(LIST_SQL), res -> {
      long groupPid = -1;
      String groupName = null;
      int maxDepth = 0;
      List<Track> tracks = Lists.newArrayList();

      ImmutableListMultimap.Builder<Long, AsyncInfo> asyncs = ImmutableListMultimap.builder();
      for (int i = 0; i < res.getNumRows(); i++) {
        long upid = res.getLong(i, 0, 0);
        String name = res.getString(i, 1, "");
        long id = res.getLong(i, 2, 0);
        int depth = (int)res.getLong(i, 3, 1);

        if (upid != groupPid || !name.equals(groupName)) {
          if (!tracks.isEmpty()) {
            asyncs.put(groupPid, new AsyncInfo(
                groupPid, groupName, maxDepth, tracks.toArray(new Track[tracks.size()])));
          }
          groupPid = upid;
          groupName = name;
          maxDepth = 0;
          tracks.clear();
        }

        tracks.add(new Track(id, maxDepth, depth));
        maxDepth += depth;
      }

      if (!tracks.isEmpty()) {
        asyncs.put(groupPid, new AsyncInfo(
            groupPid, groupName, maxDepth, tracks.toArray(new Track[tracks.size()])));
      }
      return data.setAsyncs(asyncs.build());
    });
  }

  public static class Track {
    public final long trackId;
    public final int depthOffset;
    public final int depth;

    public Track(long trackId, int depthOffset, int depth) {
      this.trackId = trackId;
      this.depthOffset = depthOffset;
      this.depth = depth;
    }
  }
}
