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

import static com.google.gapid.perfetto.views.TrackContainer.group;
import static com.google.gapid.perfetto.views.TrackContainer.single;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.util.MoreFutures.transformAsync;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.models.TrackConfig.Group;
import com.google.gapid.perfetto.views.CounterPanel;
import com.google.gapid.perfetto.views.CpuFrequencyPanel;
import com.google.gapid.perfetto.views.CpuPanel;
import com.google.gapid.perfetto.views.CpuSummaryPanel;
import com.google.gapid.perfetto.views.GpuQueuePanel;
import com.google.gapid.perfetto.views.ProcessSummaryPanel;
import com.google.gapid.perfetto.views.ThreadPanel;
import com.google.gapid.perfetto.views.TitlePanel;
import com.google.gapid.perfetto.views.VulkanCounterPanel;

import java.util.Collections;
import java.util.List;
import java.util.Objects;
import java.util.stream.Collectors;

/**
 * Determines what tracks to show for a trace.
 */
public class Tracks {
  // CPU usage percentage at which a process/thread is considered idle.
  private static final double IDLE_PERCENT_CUTOFF = 0.001; // 0.1%.

  private Tracks() {
  }

  public static ListenableFuture<Perfetto.Data.Builder> enumerate(Perfetto.Data.Builder data) {
    enumerateCpu(data);
    return transform(enumerateCounters(data), $2 -> {
      enumerateGpu(data);
      enumerateProcesses(data);
      return data;
    });
  }

  private static Perfetto.Data.Builder enumerateCpu(Perfetto.Data.Builder data) {
    if (!data.getCpu().hasCpus()) {
      return data;
    }

    CpuSummaryTrack summary = new CpuSummaryTrack(data.qe, data.getCpu().count());
    boolean hasAnyFrequency = false;
    for (CpuInfo.Cpu cpu : data.getCpu().cpus()) {
      CpuTrack track = new CpuTrack(data.qe, cpu);
      data.tracks.addTrack(summary.getId(), track.getId(), "CPU " + cpu.id,
          single(state -> new CpuPanel(state, track), false));
      if (cpu.hasFrequency()) {
        CpuFrequencyTrack freqTrack = new CpuFrequencyTrack(data.qe, cpu);
        data.tracks.addTrack(summary.getId(), freqTrack.getId(), "CPU " + cpu.id + " Frequency",
            single(state -> new CpuFrequencyPanel(state, freqTrack), false));
        hasAnyFrequency = true;
      }
    }

    Group.UiFactory ui = hasAnyFrequency ?
        group(state -> new CpuSummaryPanel(state, summary), true, (group, filtered) -> {
          for (int cpu = 0, track = 0; cpu < data.getCpu().count(); cpu++, track++) {
            if (data.getCpu().get(cpu).hasFrequency()) {
              track++;
              group.setVisible(track, !filtered);
            }
          }
        }, true) :
        group(state -> new CpuSummaryPanel(state, summary), true);
    data.tracks.addLabelGroup(null, summary.getId(), "CPU Usage", ui);
    return data;
  }

  public static ListenableFuture<Perfetto.Data.Builder> enumerateCounters(
      Perfetto.Data.Builder data) {
    return transformAsync(MemorySummaryTrack.enumerate(data), $1 ->
        BatterySummaryTrack.enumerate(data));
  }

  public static Perfetto.Data.Builder enumerateGpu(Perfetto.Data.Builder data) {
    List<CounterInfo> counters = data.getCounters(CounterInfo.Type.Gpu).values().stream()
        .filter(c -> c.count > 0)
        .collect(toList());

    if (counters.isEmpty() && (data.getGpu().queueCount() == 0)) {
      // No GPU data available.
      return data;
    }

    data.tracks.addLabelGroup(null, "gpu", "GPU", group(state -> new TitlePanel("GPU"), true));

    if (data.getGpu().queueCount() > 0) {
      data.tracks.addLabelGroup(
          "gpu", "gpu_queues", "GPU Queues", group(state -> new TitlePanel("GPU Queues"), true));
      for (GpuInfo.Queue queue : data.getGpu().queues()) {
        SliceTrack track = SliceTrack.forGpuQueue(data.qe, queue);
        data.tracks.addTrack("gpu_queues", track.getId(), queue.getDisplay(),
            single(state -> new GpuQueuePanel(state, queue, track), true));
      }
    }

    if (!counters.isEmpty()) {
      data.tracks.addLabelGroup("gpu", "gpu_counters", "GPU Counters",
          group(state -> new TitlePanel("GPU Counters"), true));
      for (CounterInfo counter : counters) {
        CounterTrack track = new CounterTrack(data.qe, counter);
        data.tracks.addTrack("gpu_counters", track.getId(), counter.name,
            single(state -> new CounterPanel(state, track), true));
      }
    }
    return data;
  }

  public static Perfetto.Data.Builder enumerateProcesses(Perfetto.Data.Builder data) {
    List<ProcessInfo> processes = Lists.newArrayList(data.getProcesses().values());
    Collections.sort(processes, (p1, p2) -> Long.compare(p2.totalDur, p1.totalDur));
    final int count = processes.size();
    if (count == 0) {
      return data;
    }

    final long idleCutoffProc = Math.round(IDLE_PERCENT_CUTOFF * data.getTraceTime().getDuration());

    data.tracks.addLabelGroup(null, "procs", "Processes",
        group(state -> new TitlePanel("Processes (" + count + ")"), true));
    // Whether we have at least two idle processes.
    boolean hasIdles = count > 1 && processes.get(processes.size() - 2).totalDur < idleCutoffProc;
    processes.forEach(process -> {
      ProcessSummaryTrack summary =
          new ProcessSummaryTrack(data.qe, data.getCpu().count(), process);
      String parent = (process.totalDur >= idleCutoffProc || !hasIdles) ? "procs" : "procs_idle";
      data.tracks.addGroup(parent, summary.getId(), process.getDisplay(),
          group(state -> new ProcessSummaryPanel(state, summary), false));

      List<ThreadTrack> threads = process.utids.stream()
          .map(tid -> data.getThreads().get(tid))
          .filter(Objects::nonNull)
          .sorted((t1, t2) -> Long.compare(t2.totalDur, t1.totalDur))
          .map(t -> new ThreadTrack(data.qe, t))
          .collect(Collectors.toList());
      final long idleCutoffThread =
          Math.min(idleCutoffProc, Math.round(IDLE_PERCENT_CUTOFF * process.totalDur));
      // Whether we have at least two idle threads.
      boolean hasIdleThreads = threads.size() > 1 &&
          threads.get(threads.size() - 2).getThread().totalDur < idleCutoffThread;
      threads.forEach(track -> {
        TrackConfig.Track.UiFactory<Panel> ui;
        if (track.getThread().maxDepth == 0) {
          ui = single(state -> new ThreadPanel(state, track), false);
        } else {
          ui = single(
              state -> new ThreadPanel(state, track), false, ThreadPanel::setCollapsed, true);
        }
        String threadParent = (track.getThread().totalDur >= idleCutoffThread || !hasIdleThreads) ?
            summary.getId() : summary.getId() + "_idle";
        data.tracks.addTrack(threadParent, track.getId(), track.getThread().getDisplay(), ui);
      });
      if (hasIdleThreads) {
        int firstIdle = Collections.binarySearch(
            threads, null, (t1, $) -> Long.compare(idleCutoffThread, t1.getThread().totalDur));
        if (firstIdle < 0) {
          firstIdle = -firstIdle - 1;
        } else {
          while (firstIdle > 1 && threads.get(firstIdle - 1).getThread().totalDur == 0) {
            firstIdle--;
          }
        }
        final int idleCount = threads.size() - firstIdle;
        data.tracks.addLabelGroup(summary.getId(), summary.getId() + "_idle", "Idle Threads",
            group(state -> new TitlePanel(idleCount + " Idle Threads (< 0.1%)"), false));
      }

      // For each process, add Vulkan memory usage counters if any exist
      List<CounterInfo> counters = data.getCounters().values().stream()
      .filter(c -> (c.count > 0) && (c.name.startsWith("vulkan")))
      .collect(toList());

      if (!counters.isEmpty()) {
        data.tracks.addLabelGroup(summary.getId(), "vulkan_counters", "Vulkan Counters",
            group(state -> new TitlePanel("Vulkan Memory Usage Counters"), true));
        for (CounterInfo counter : counters) {
          CounterTrack track = new CounterTrack(data.qe, counter);
          data.tracks.addTrack("vulkan_counters", track.getId(), counter.name,
              single(state -> new VulkanCounterPanel(state, track), true));
        }
      }
    });

    if (hasIdles) {
      int firstIdle = Collections.binarySearch(
          processes, null, (p1, $) -> Long.compare(idleCutoffProc, p1.totalDur));
      if (firstIdle < 0) {
        firstIdle = -firstIdle - 1;
      } else {
        while (firstIdle > 1 && processes.get(firstIdle - 1).totalDur < idleCutoffProc) {
          firstIdle--;
        }
      }
      final int idleCount = processes.size() - firstIdle;
      data.tracks.addLabelGroup("procs", "procs_idle", "Idle Processes",
          group(state -> new TitlePanel(idleCount + " Idle Processes (< 0.1%)"), false));
    }
    return data;
  }
}
