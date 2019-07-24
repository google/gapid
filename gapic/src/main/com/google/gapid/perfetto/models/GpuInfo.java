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

import static com.google.common.collect.Iterators.transform;
import static com.google.common.collect.Iterators.unmodifiableIterator;
import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.models.GpuInfo.Gpu;

import java.util.Collections;
import java.util.Iterator;
import java.util.SortedMap;

/**
 * Data about a GPU in the trace.
 */
public class GpuInfo implements Iterable<Gpu> {
  public static final GpuInfo NONE = new GpuInfo(Collections.emptySortedMap());

  private static final String MAX_DEPTH_QUERY =
      "select ref, max(depth) + 1 from slices where ref_type = 'gpu' group by ref";

  private final SortedMap<Long, Integer> maxDepth;

  private GpuInfo(SortedMap<Long, Integer> maxDepth) {
    this.maxDepth = maxDepth;
  }

  @Override
  public Iterator<Gpu> iterator() {
    return unmodifiableIterator(transform(
        maxDepth.entrySet().iterator(), e -> new Gpu(e.getKey(), e.getValue())));
  }

  public static ListenableFuture<Perfetto.Data.Builder> listGpus(Perfetto.Data.Builder data) {
    return transform(maxDepth(data.qe), maxDepth -> {
      return data.setGpu(new GpuInfo(maxDepth));
    });
  }

  private static ListenableFuture<SortedMap<Long, Integer>> maxDepth(QueryEngine qe) {
    return transform(qe.queries(MAX_DEPTH_QUERY),
        res -> res.sortedMap(row -> row.getLong(0), row -> row.getInt(1)));
  }

  public static class Gpu {
    public final long id;
    public final int maxDepth;

    public Gpu(long id, int maxDepth) {
      this.id = id;
      this.maxDepth = maxDepth;
    }

    public String getDisplay() {
      return "GPU Queue " + id;
    }
  }
}
