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

import static com.google.common.collect.Streams.stream;
import static com.google.gapid.util.MoreFutures.transform;
import static java.util.logging.Level.WARNING;

import com.google.common.base.Splitter;
import com.google.common.collect.ImmutableMap;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.Unit;
import com.google.gapid.proto.device.GpuProfiling;

import java.util.Objects;
import java.util.logging.Logger;

public class CounterInfo {
  private static final Logger LOG = Logger.getLogger(CounterInfo.class.getName());

  private static final String LIST_SQL =
      "select ct.id, ct.type, coalesce(cpu, gpu_id, upid, utid), ct.name, ct.description, " +
      "  ct.unit, count(value), min(value), max(value), avg(value) " +
      "from counter_track ct " +
        "left join cpu_counter_track using (id) " +
        "left join gpu_counter_track using (id) " +
        "left join process_counter_track using (id) " +
        "left join thread_counter_track using (id) " +
        "left join counter on (track_id = ct.id) " +
      "where (value + 1 > value or value - 1 < value) " +
      "group by ct.id";

  private static final ImmutableMap<Integer, Unit> UNITS = ImmutableMap.<Integer, Unit> builder()
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.NONE_VALUE, Unit.NONE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.BIT_VALUE, Unit.BIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.KILOBIT_VALUE, Unit.KILO_BIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MEGABIT_VALUE, Unit.MEGA_BIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.GIGABIT_VALUE, Unit.GIGA_BIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.TERABIT_VALUE, Unit.TERA_BIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.PETABIT_VALUE, Unit.PETA_BIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.BYTE_VALUE, Unit.BYTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.KILOBYTE_VALUE, Unit.KILO_BYTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MEGABYTE_VALUE, Unit.MEGA_BYTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.GIGABYTE_VALUE, Unit.GIGA_BYTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.TERABYTE_VALUE, Unit.TERA_BYTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.PETABYTE_VALUE, Unit.PETA_BYTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.HERTZ_VALUE, Unit.HERTZ)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.KILOHERTZ_VALUE, Unit.KILO_HERTZ)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MEGAHERTZ_VALUE, Unit.MEGA_HERTZ)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.GIGAHERTZ_VALUE, Unit.GIGA_HERTZ)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.TERAHERTZ_VALUE, Unit.TERA_HERTZ)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.PETAHERTZ_VALUE, Unit.PETA_HERTZ)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.NANOSECOND_VALUE, Unit.NANO_SECOND)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MICROSECOND_VALUE, Unit.MICRO_SECOND)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MILLISECOND_VALUE, Unit.MILLI_SECOND)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.SECOND_VALUE, Unit.SECOND)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MINUTE_VALUE, Unit.MINUTE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.HOUR_VALUE, Unit.HOUR)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.VERTEX_VALUE, Unit.VERTEX)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.PIXEL_VALUE, Unit.PIXEL)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.TRIANGLE_VALUE, Unit.TRIANGLE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.PRIMITIVE_VALUE, Unit.PRIMITIVE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.FRAGMENT_VALUE, Unit.FRAGMENT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.MILLIWATT_VALUE, Unit.MILLI_WATT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.WATT_VALUE, Unit.WATT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.KILOWATT_VALUE, Unit.KILO_WATT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.JOULE_VALUE, Unit.JOULE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.VOLT_VALUE, Unit.VOLT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.AMPERE_VALUE, Unit.AMPERE)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.CELSIUS_VALUE, Unit.CELSIUS)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.FAHRENHEIT_VALUE, Unit.FAHRENHEIT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.KELVIN_VALUE, Unit.KELVIN)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.PERCENT_VALUE, Unit.PERCENT)
      .put(GpuProfiling.GpuCounterDescriptor.MeasureUnit.INSTRUCTION_VALUE, Unit.INSTRUCTION)
      .build();

  public final long id;
  public final Type type;
  public final String name;
  public final String description;
  public final Unit unit;
  public final long ref;
  public final long count;
  public final double min;
  public final double max;
  public final double avg;

  public CounterInfo(long id, Type type, long ref, String name, String description, Unit unit,
      long count, double min, double max, double avg) {
    this.id = id;
    this.type = type;
    this.ref = ref;
    this.name = name;
    this.description = description;
    this.unit = unit;
    this.count = count;
    this.min = min;
    this.max = (max == min && max == 0) ? 1 : max;
    this.avg = avg;
  }

  private CounterInfo(QueryEngine.Row row) {
    this(row.getLong(0), Type.of(row.getString(1)), row.getLong(2), row.getString(3),
        row.getString(4), unitFromString(row.getString(5)), row.getLong(6), row.getDouble(7),
        row.getDouble(8), row.getDouble(9));
  }

  public static Unit unitFromString(String unit) {
    unit = unit.trim();
    if (unit.isEmpty()) {
      return Unit.NONE;
    }

    int p = unit.indexOf('/');
    String numer = (p < 0) ? unit : unit.substring(0, p);
    String denom = (p < 0) ? "" : unit.substring(p + 1);
    try {
      return Unit.combined(parseUnits(numer), parseUnits(denom));
    } catch (NumberFormatException e) {
      LOG.log(WARNING, "Failed to parse counter unit: " + unit, e);
      return Unit.NONE;
    }
  }

  private static Unit[] parseUnits(String units) throws NumberFormatException {
    units = units.trim();
    if (units.isEmpty()) {
      return new Unit[0];
    }
    return stream(Splitter.on(":").omitEmptyStrings().trimResults().split(units))
        .mapToInt(Integer::parseInt)
        .mapToObj(UNITS::get)
        .filter(Objects::nonNull)
        .toArray(Unit[]::new);

  }

  public static ListenableFuture<Perfetto.Data.Builder> listCounters(Perfetto.Data.Builder data) {
    return transform(data.qe.query(LIST_SQL), res -> {
      ImmutableMap.Builder<Long, CounterInfo> counters = ImmutableMap.builder();
      res.forEachRow((i, r) -> counters.put(r.getLong(0), new CounterInfo(r)));
      return data.setCounters(counters.build());
    });
  }

  public static enum Type {
    Global, Cpu, Gpu, Process, Thread;

    public static Type of(String string) {
      switch (string) {
        case "cpu_counter_track": return Cpu;
        case "gpu_counter_track": return Gpu;
        case "process_counter_track": return Process;
        case "thread_counter_track": return Thread;
        default:
          return Global; // Treat unknowns as global counters.
      }
    }
  }
}
