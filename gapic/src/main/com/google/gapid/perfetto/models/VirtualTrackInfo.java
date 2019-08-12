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
import static com.google.gapid.util.MoreFutures.transformAsync;

import com.google.common.collect.ImmutableMap;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import java.util.SortedMap;

/**
 * Data about a virtual track in the trace.
 */
public class VirtualTrackInfo {
  private static final String TRACKS_QUERY = "select id, name from track order by name desc";

  public final long trackId;
  public final String name;

  public VirtualTrackInfo(long trackId, String name) {
    this.trackId = trackId;
    this.name = name;
  }

  public static ListenableFuture<Perfetto.Data.Builder> listVirtualTracks(
      Perfetto.Data.Builder data) {
    return transform(data.qe.queries(TRACKS_QUERY), res -> {
          ImmutableMap.Builder<Long, VirtualTrackInfo> tracks = ImmutableMap.builder();
          res.forEachRow(($1, row) -> {
            long ref = row.getLong(0);
            String name = row.getString(1);
            if (name == null) {
              name = "Virtual track " + ref;
            }

            tracks.put(ref, new VirtualTrackInfo(ref, name));
          });
          data.setVirtualTracks(tracks.build());

          return data;
        });
  }
}
