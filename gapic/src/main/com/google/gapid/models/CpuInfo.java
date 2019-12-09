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
package com.google.gapid.models;

import static com.google.gapid.util.MoreFutures.transform;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.Iterables;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.perfetto.models.QueryEngine;

import java.util.Collections;
import java.util.List;

public class CpuInfo {
  public static final CpuInfo NONE = new CpuInfo(Collections.emptyList());

  private static final String CPU_FREQ_IDLE_QUERY =
      "with freq as (" +
        "select cpu, cd.id freq_id, max(value) freq " +
        "from cpu_counter_track cd left join counter_values cv on cd.id = cv.counter_id " +
        "where name = 'cpufreq' group by cd.id), " +
      "idle as (select cpu, id idle_id from cpu_counter_track where name = 'cpuidle'), " +
      "cpus as (select distinct(cpu) from sched union select distinct(cpu) from idle) " +
      "select cpu, freq_id, freq, idle_id " +
      "from cpus left join idle using (cpu) left join freq using (cpu) " +
      "order by cpu";

  private final List<Cpu> cpus;

  private CpuInfo(List<Cpu> cpus) {
    this.cpus = cpus;
  }

  public int count() {
    return cpus.size();
  }

  public boolean hasCpus() {
    return !cpus.isEmpty();
  }

  public Cpu get(int idx) {
    return cpus.get(idx);
  }

  public Iterable<Cpu> cpus() {
    return Iterables.unmodifiableIterable(cpus);
  }

  public static ListenableFuture<Perfetto.Data.Builder> listCpus(Perfetto.Data.Builder data) {
    return transform(data.qe.query(CPU_FREQ_IDLE_QUERY), res -> {
      ImmutableList.Builder<Cpu> cpus = ImmutableList.builderWithExpectedSize(res.getNumRows());
      res.forEachRow(($, r) -> {
        cpus.add(Cpu.of(r));
      });
      return data.setCpu(new CpuInfo(cpus.build()));
    });
  }

  public static class Cpu {
    public final int id;
    public final long freqId;
    public final double maxFreq;
    public final long idleId;

    private Cpu(int id) {
      this.id = id;
      this.freqId = -1;
      this.maxFreq = Double.NaN;
      this.idleId = -1;
    }

    private Cpu(int id, long freqId, double maxFreq, long idleId) {
      this.id = id;
      this.freqId = freqId;
      this.maxFreq = maxFreq;
      this.idleId = idleId;
    }

    public static Cpu of(QueryEngine.Row r) {
      int cpu = r.getInt(0);
      return (r.isNull(1) || r.isNull(3)) ? new Cpu(cpu) :
          new Cpu(cpu, r.getLong(1), r.getDouble(2), r.getLong(3));
    }

    public boolean hasFrequency() {
      return freqId >= 0;
    }
  }
}
