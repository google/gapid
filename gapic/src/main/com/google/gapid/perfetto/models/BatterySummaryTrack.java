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
import com.google.gapid.perfetto.views.BatterySelectionView;
import com.google.gapid.perfetto.views.BatterySummaryPanel;
import com.google.gapid.perfetto.views.State;

import org.eclipse.swt.widgets.Composite;

import java.util.function.BinaryOperator;

public class BatterySummaryTrack
    extends CombinedCountersTrack<BatterySummaryTrack.Data, BatterySummaryTrack.Values> {
  private final double maxAbsCurrent;

  public BatterySummaryTrack(
      QueryEngine qe, CounterInfo capacity, CounterInfo charge, CounterInfo current) {
    super(qe, "bat_sum", new Column[] {
        new Column(capacity.id, "int", "0", "avg"),
        new Column(charge.id, "int", "0", "avg"),
        new Column(current.id, "int", "0", "avg"),
    }, needQuantize(capacity, charge, current));
    this.maxAbsCurrent = Math.max(Math.abs(current.min), Math.abs(current.max));
  }

  public double getMaxAbsCurrent() {
    return maxAbsCurrent;
  }

  public static Perfetto.Data.Builder enumerate(Perfetto.Data.Builder data) {
    ImmutableListMultimap<String, CounterInfo> counters = data.getCounters(CounterInfo.Type.Global);
    CounterInfo battCap = onlyOne(counters.get("batt.capacity_pct"));
    CounterInfo battCharge = onlyOne(counters.get("batt.charge_uah"));
    CounterInfo battCurrent = onlyOne(counters.get("batt.current_ua"));
    if ((battCap == null) || (battCharge  == null) || (battCurrent  == null)) {
      return data;
    }

    BatterySummaryTrack track = new BatterySummaryTrack(data.qe, battCap, battCharge, battCurrent);
    data.tracks.addTrack(null, track.getId(), "Battery Usage",
        single(state -> new BatterySummaryPanel(state, track), true, false));
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
    public final long[] capacity;
    public final long[] charge;
    public final long[] current;

    public Data(DataRequest request, int numRows) {
      super(request, numRows);
      this.capacity = new long[numRows];
      this.charge = new long[numRows];
      this.current = new long[numRows];
    }

    @Override
    public void set(int idx, Row row) {
      super.set(idx, row);
      capacity[idx] = row.getLong(FIRST_DATA_COLUMN + 0);
      charge[idx] = row.getLong(FIRST_DATA_COLUMN + 1);
      current[idx] = row.getLong(FIRST_DATA_COLUMN + 2);
    }

    @Override
    public void copyRow(long time, int src, int dst) {
      super.copyRow(time, src, dst);
      capacity[dst] = capacity[src];
      charge[dst] = charge[src];
      current[dst] = current[src];
    }
  }

  public static class Values extends CombinedCountersTrack.Values<Values> {
    public final long[] capacity;
    public final long[] charge;
    public final long[] current;

    public Values(int numRows, BinaryOperator<Values> combiner) {
      super(numRows, combiner);
      this.capacity = new long[numRows];
      this.charge = new long[numRows];
      this.current = new long[numRows];
    }

    @Override
    public void set(int idx, Row row) {
      super.set(idx, row);
      capacity[idx] = row.getLong(FIRST_DATA_COLUMN + 0);
      charge[idx] = row.getLong(FIRST_DATA_COLUMN + 1);
      current[idx] = row.getLong(FIRST_DATA_COLUMN + 2);
    }

    @Override
    public void copyFrom(Values other, int src, int dst) {
      capacity[dst] = other.capacity[src];
      charge[dst] = other.charge[src];
      current[dst] = other.current[src];
    }

    @Override
    public String getTitle() {
      return "Battery Usage";
    }

    @Override
    public Composite buildUi(Composite parent, State state) {
      return new BatterySelectionView(parent, state, this);
    }
  }
}
