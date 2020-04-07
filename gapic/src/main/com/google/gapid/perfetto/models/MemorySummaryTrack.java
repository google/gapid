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

import static com.google.gapid.perfetto.models.CounterInfo.needQuantize;
import static com.google.gapid.perfetto.views.TrackContainer.single;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.ImmutableListMultimap;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.models.QueryEngine.Row;
import com.google.gapid.perfetto.views.MemorySelectionView;
import com.google.gapid.perfetto.views.MemorySummaryPanel;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.function.BinaryOperator;

/**
 * {@link Track} containing the total system memory usage data.
 */
public class MemorySummaryTrack
    extends CombinedCountersTrack<MemorySummaryTrack.Data, MemorySummaryTrack.Values> {
  private final long maxTotal;

  public MemorySummaryTrack(QueryEngine qe, CounterInfo total, CounterInfo unused,
      CounterInfo buffers, CounterInfo cached, CounterInfo swapCached) {
    super(qe, "mem_sum", new Column[] {
        new Column(total.id, "int", "0", "avg"),
        new Column(unused.id, "int", "0", "avg"),
        new Column(buffers.id, "int", "0", "avg"),
        new Column(cached.id, "int", "0", "avg"),
        new Column(swapCached.id, "int", "0", "avg"),
    }, needQuantize(total, unused, buffers, cached, swapCached));
    this.maxTotal = (long)total.max;
  }

  public long getMaxTotal() {
    return maxTotal;
  }

  public static Perfetto.Data.Builder enumerate(Perfetto.Data.Builder data) {
    ImmutableListMultimap<String, CounterInfo> counters = data.getCounters(CounterInfo.Type.Global);
    CounterInfo total = onlyOne(counters.get("MemTotal"));
    CounterInfo free = onlyOne(counters.get("MemFree"));
    CounterInfo buffers = onlyOne(counters.get("Buffers"));
    CounterInfo cached = onlyOne(counters.get("Cached"));
    CounterInfo swapCached = onlyOne(counters.get("SwapCached"));
    if ((total == null) || (free  == null) || (buffers  == null) || (cached  == null) ||
        (swapCached  == null)) {
      return data;
    }

    MemorySummaryTrack track =
        new MemorySummaryTrack(data.qe, total, free, buffers, cached, swapCached);
    data.tracks.addTrack(null, track.getId(), "Memory Usage",
        single(state -> new MemorySummaryPanel(state, track), true, false));
    return data;
  }

  private static CounterInfo onlyOne(ImmutableList<CounterInfo> counters) {
    return (counters.size() != 1) ? null : counters.get(0);
  }

  @Override
  protected Data createData(DataRequest request, int numRows) {
    return new Data(request, numRows);
  }

  @Override
  protected Values createValues(int numRows, BinaryOperator<Values> combiner) {
    return new Values(numRows, combiner);
  }

  public static class Data extends CombinedCountersTrack.Data {
    public final long[] total;
    public final long[] unused;
    public final long[] buffCache;

    public Data(DataRequest request, int numRows) {
      super(request, numRows);
      this.total = new long[numRows];
      this.unused = new long[numRows];
      this.buffCache = new long[numRows];
    }

    @Override
    public void set(int idx, QueryEngine.Row row) {
      super.set(idx, row);
      total[idx] = row.getLong(FIRST_DATA_COLUMN + 0);
      unused[idx] = row.getLong(FIRST_DATA_COLUMN + 1);
      buffCache[idx] =
          row.getLong(FIRST_DATA_COLUMN + 2) +
          row.getLong(FIRST_DATA_COLUMN + 3) +
          row.getLong(FIRST_DATA_COLUMN + 4);
    }

    @Override
    public void copyRow(long time, int src, int dst) {
      super.copyRow(time, src, dst);
      total[dst] = total[src];
      unused[dst] = unused[src];
      buffCache[dst] = buffCache[src];
    }
  }

  public static class Values extends CombinedCountersTrack.Values<Values> {
    public final long[] total;
    public final long[] unused;
    public final long[] buffCache;

    public Values(int numRows, BinaryOperator<Values> combiner) {
      super(numRows, combiner);
      this.total = new long[numRows];
      this.unused = new long[numRows];
      this.buffCache = new long[numRows];
    }

    @Override
    public void set(int idx, Row row) {
      super.set(idx, row);
      total[idx] = row.getLong(FIRST_DATA_COLUMN + 0);
      unused[idx] = row.getLong(FIRST_DATA_COLUMN + 1);
      buffCache[idx] =
          row.getLong(FIRST_DATA_COLUMN + 2) +
          row.getLong(FIRST_DATA_COLUMN + 3) +
          row.getLong(FIRST_DATA_COLUMN + 4);
    }

    @Override
    public void copyFrom(Values other, int src, int dst) {
      total[dst] = other.total[src];
      unused[dst] = other.unused[src];
      buffCache[dst] = other.buffCache[src];
    }

    @Override
    public String getTitle() {
      return "Memory Usage";
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new MemorySelectionView(parent, state, this);
    }
  }
}
