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

import static com.google.gapid.perfetto.models.CounterInfo.needQuantize;

import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.models.QueryEngine.Row;
import com.google.gapid.perfetto.views.MemorySelectionView;
import com.google.gapid.perfetto.views.ProcessMemoryPanel;
import com.google.gapid.perfetto.views.State;
import com.google.gapid.perfetto.views.TrackContainer;

import org.eclipse.swt.widgets.Composite;

import java.util.Map;
import java.util.function.BinaryOperator;
import java.util.function.Function;
import java.util.stream.Collectors;

public class ProcessMemoryTrack
    extends CombinedCountersTrack<ProcessMemoryTrack.Data, ProcessMemoryTrack.Values> {
  private final long maxUsage;
  private final long maxSwap;

  public ProcessMemoryTrack(QueryEngine qe, long upid,
      CounterInfo file, CounterInfo anon, CounterInfo shared, CounterInfo swap) {
    super(qe, "mem_" + upid, new Column[] {
        file == null ? Column.nil("0") : new Column(file.id, "int", "0", "avg"),
        anon == null ? Column.nil("0") : new Column(anon.id, "int", "0", "avg"),
        shared == null ? Column.nil("0") : new Column(shared.id, "int", "0", "avg"),
        swap == null ? Column.nil("0") : new Column(swap.id, "int", "0", "avg"),
    }, needQuantize(file, anon, shared, swap));
    this.maxUsage = (long)(
        (file == null ? 0 : file.max) +
        (anon == null ? 0 : anon.max) +
        (shared == null ? 0 : shared.max));
    this.maxSwap = swap == null ? 0 : (long)swap.max;
  }

  public long getMaxUsage() {
    return maxUsage;
  }

  public long getMaxSwap() {
    return maxSwap;
  }

  public static Perfetto.Data.Builder enumerate(Perfetto.Data.Builder data, String parent, long upid) {
    Map<String, CounterInfo> counters = data.getCounters().values().stream()
        .filter(c -> c.type == CounterInfo.Type.Process && c.ref == upid &&
            c.count > 0 && isMemoryCounter(c))
        .collect(Collectors.toMap(c -> c.name, Function.identity()));
    if (!counters.isEmpty()) {
      ProcessMemoryTrack track = new ProcessMemoryTrack(data.qe, upid,
          counters.get("mem.rss.file"),
          counters.get("mem.rss.anon"),
          counters.get("mem.rss.shmem"),
          counters.get("mem.swap"));
      data.tracks.addTrack(parent, track.getId(), "Memory Usage",
          TrackContainer.single(state -> new ProcessMemoryPanel(state, track), true, true));
    }

    return data;
  }

  private static boolean isMemoryCounter(CounterInfo c) {
    return "mem.rss.file".equals(c.name) ||
        "mem.rss.anon".equals(c.name) ||
        "mem.rss.shmem".equals(c.name) ||
        "mem.swap".equals(c.name);
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
    public final long[] file;
    public final long[] anon;
    public final long[] shared;
    public final long[] swap;

    public Data(DataRequest request, int numRows) {
      super(request, numRows);
      this.file = new long[numRows];
      this.anon = new long[numRows];
      this.shared = new long[numRows];
      this.swap = new long[numRows];
    }

    @Override
    public void set(int idx, Row row) {
      super.set(idx, row);
      file[idx] = row.getLong(FIRST_DATA_COLUMN + 0);
      anon[idx] = row.getLong(FIRST_DATA_COLUMN + 1);
      shared[idx] = row.getLong(FIRST_DATA_COLUMN + 2);
      swap[idx] = row.getLong(FIRST_DATA_COLUMN + 3);
    }

    @Override
    public void copyRow(long time, int src, int dst) {
      super.copyRow(time, src, dst);
      file[dst] = file[src];
      anon[dst] = anon[src];
      shared[dst] = shared[src];
      swap[dst] = swap[src];
    }
  }

  public static class Values extends CombinedCountersTrack.Values<Values> {
    public final long[] file;
    public final long[] anon;
    public final long[] shared;
    public final long[] swap;

    public Values(int numRows, BinaryOperator<Values> combiner) {
      super(numRows, combiner);
      this.file = new long[numRows];
      this.anon = new long[numRows];
      this.shared = new long[numRows];
      this.swap = new long[numRows];
    }

    @Override
    public void set(int idx, Row row) {
      super.set(idx, row);
      file[idx] = row.getLong(FIRST_DATA_COLUMN + 0);
      anon[idx] = row.getLong(FIRST_DATA_COLUMN + 1);
      shared[idx] = row.getLong(FIRST_DATA_COLUMN + 2);
      swap[idx] = row.getLong(FIRST_DATA_COLUMN + 3);
    }

    @Override
    public void copyFrom(Values other, int src, int dst) {
      file[dst] = other.file[src];
      anon[dst] = other.anon[src];
      shared[dst] = other.shared[src];
      swap[dst] = other.swap[src];
    }

    @Override
    public String getTitle() {
      return "Process Memory Usage";
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new MemorySelectionView(parent, state, this);
    }
  }
}
