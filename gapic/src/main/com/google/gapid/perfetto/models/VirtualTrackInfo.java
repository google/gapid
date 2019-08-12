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
  // TODO: When the perfetto tracks table is exposed, query that directly for id and name instead of
  // doing this reduction on the slices table.
  private static final String TRACKS_QUERY = "select ref, name from "
      + "(select ref, arg_set_id from slices where ref_type = 'track' group by ref) as slices "
      + "left join "
      + "(select flat_key, string_value as name, arg_set_id from args "
      + "where string_value is not null and flat_key = 'layer_name') as args "
      + "on slices.arg_set_id = args.arg_set_id "
      + "order by name desc";

  private static final String MAX_DEPTH_QUERY =
      "select ref, max(depth) + 1 from slices where ref_type = 'track' group by ref";

  public final long trackId;
  public final String name;
  public final int maxDepth;

  public VirtualTrackInfo(long trackId, String name, int maxDepth) {
    this.trackId = trackId;
    this.name = name;
    this.maxDepth = maxDepth;
  }

  public static ListenableFuture<Perfetto.Data.Builder> listVirtualTracks(
      Perfetto.Data.Builder data) {
    return transformAsync(
        maxDepth(data.qe), maxDepth -> transform(data.qe.query(TRACKS_QUERY), res -> {
          ImmutableMap.Builder<Long, VirtualTrackInfo> tracks = ImmutableMap.builder();
          res.forEachRow(($1, row) -> {
            long ref = row.getLong(0);
            String name = row.getString(1);
            if (name == null) {
              name = "Virtual track " + ref;
            }

            tracks.put(ref, new VirtualTrackInfo(ref, name, maxDepth.getOrDefault(ref, 0)));
          });
          data.setVirtualTracks(tracks.build());

          return data;
        }));
  }

  private static ListenableFuture<SortedMap<Long, Integer>> maxDepth(QueryEngine qe) {
    return transform(qe.queries(MAX_DEPTH_QUERY),
        res -> res.sortedMap(row -> row.getLong(0), row -> row.getInt(1)));
  }
}
