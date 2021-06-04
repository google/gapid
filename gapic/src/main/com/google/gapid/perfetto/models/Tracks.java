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

import static com.google.gapid.perfetto.views.StyleConstants.DEFAULT_COUNTER_TRACK_HEIGHT;
import static com.google.gapid.perfetto.views.StyleConstants.PROCESS_COUNTER_TRACK_HEIGHT;
import static com.google.gapid.perfetto.views.TrackContainer.group;
import static com.google.gapid.perfetto.views.TrackContainer.single;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toList;
import static java.util.stream.Collectors.toMap;

import com.google.common.collect.ImmutableMap;
import com.google.common.collect.ImmutableSet;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;
import com.google.common.collect.Multimap;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Perfetto;
import com.google.gapid.perfetto.canvas.Panel;
import com.google.gapid.perfetto.models.TrackConfig.Group;
import com.google.gapid.perfetto.views.CounterPanel;
import com.google.gapid.perfetto.views.CpuFrequencyPanel;
import com.google.gapid.perfetto.views.CpuPanel;
import com.google.gapid.perfetto.views.CpuSummaryPanel;
import com.google.gapid.perfetto.views.FrameEventsPanel;
import com.google.gapid.perfetto.views.GpuQueuePanel;
import com.google.gapid.perfetto.views.ProcessSummaryPanel;
import com.google.gapid.perfetto.views.ThreadPanel;
import com.google.gapid.perfetto.views.TitlePanel;
import com.google.gapid.perfetto.views.VulkanCounterPanel;
import com.google.gapid.perfetto.views.VulkanEventPanel;
import com.google.gapid.proto.device.GpuProfiling.GpuCounterDescriptor.GpuCounterGroup;
import com.google.gapid.util.Scheduler;

import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.logging.Logger;
import java.util.stream.Collectors;

/**
 * Determines what tracks to show for a trace.
 */
public class Tracks {
  private static final Logger LOG = Logger.getLogger(Tracks.class.getName());

  // CPU usage percentage at which a process/thread is considered idle.
  private static final double IDLE_PERCENT_CUTOFF = 0.001; // 0.1%.
  // Polled counters from the process_stats data source.
  private static final ImmutableSet<String> PROC_STATS_COUNTER = ImmutableSet.of(
      "mem.virt", "mem.rss", "mem.locked", "oom_score_adj"
  );

  private static final ImmutableMap<GpuCounterGroup, String> GPU_COUNTER_GROUP_NAMES =
      new ImmutableMap.Builder<GpuCounterGroup, String>()
          .put(GpuCounterGroup.UNCLASSIFIED, "General Counters")
          .put(GpuCounterGroup.SYSTEM, "System Counters")
          .put(GpuCounterGroup.VERTICES, "Vertex Counters")
          .put(GpuCounterGroup.FRAGMENTS, "Fragment Counters")
          .put(GpuCounterGroup.PRIMITIVES, "Primitive Counters")
          .put(GpuCounterGroup.MEMORY, "Memory Counters")
          .put(GpuCounterGroup.COMPUTE, "Compute Counters")
          .build();

  private Tracks() {
  }

  public static ListenableFuture<Perfetto.Data.Builder> enumerate(Perfetto.Data.Builder data) {
    return Scheduler.EXECUTOR.submit(() -> {
      enumerateCpu(data);
      enumerateCounters(data);
      enumerateGpu(data);
      enumerateFrame(data);
      enumerateProcesses(data);
      enumerateVSync(data);
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
          single(state -> new CpuPanel(state, track), false, false));
      if (cpu.hasFrequency()) {
        CpuFrequencyTrack freqTrack = new CpuFrequencyTrack(data.qe, cpu);
        data.tracks.addTrack(summary.getId(), freqTrack.getId(), "CPU " + cpu.id + " Frequency",
            single(state -> new CpuFrequencyPanel(state, freqTrack), false, false));
        hasAnyFrequency = true;
      }
    }

    Group.UiFactory ui = hasAnyFrequency ?
        group(state -> new CpuSummaryPanel(state, summary), true, (group, showDetails) -> {
          for (int cpu = 0, track = 0; cpu < data.getCpu().count(); cpu++, track++) {
            if (data.getCpu().get(cpu).hasFrequency()) {
              track++;
              group.setVisible(track, showDetails);
            }
          }
        }, false) :
        group(state -> new CpuSummaryPanel(state, summary), true);
    data.tracks.addLabelGroup(null, summary.getId(), "CPU Usage", ui);
    return data;
  }

  public static Perfetto.Data.Builder enumerateCounters(Perfetto.Data.Builder data) {
    MemorySummaryTrack.enumerate(data);
    BatterySummaryTrack.enumerate(data);
    return data;
  }

  public static Perfetto.Data.Builder enumerateGpu(Perfetto.Data.Builder data) {
    Map<Long, CounterInfo> counters = data.getCounters(CounterInfo.Type.Gpu).values().stream()
        .filter(c -> c.count > 0)
        .collect(toMap(c -> c.id, c -> c));

    List<CounterInfo> gpuMemGlobalCounter =
        data.getCounters().values().stream()
            .filter(c -> c.type == CounterInfo.Type.Global &&
                c.ref == 0 /* pid - 0 */ &&
                c.count > 0 &&
                c.name.equals("GPU Memory"))
            .collect(toList());

    if (counters.isEmpty() && data.getGpu().isEmpty() && gpuMemGlobalCounter.isEmpty()) {
      // No GPU data available.
      return data;
    }

    data.tracks.addLabelGroup(null, "gpu", "GPU", group(state -> new TitlePanel("GPU"), true));

    if (!gpuMemGlobalCounter.isEmpty()) {
      if (gpuMemGlobalCounter.size() > 1) {
        LOG.log(WARNING, "Expected 1 global gpu memory counter. Found " + gpuMemGlobalCounter.size());
      }
      CounterInfo counter = gpuMemGlobalCounter.get(0);
      CounterTrack track = new CounterTrack(data.qe, counter);
      data.tracks.addTrack("gpu", track.getId(), counter.name,
          single(state -> new CounterPanel(state, track, DEFAULT_COUNTER_TRACK_HEIGHT), true,
              /*right truncate*/ true));
    }

    if (data.getGpu().queueCount() > 0) {
      String parent = "gpu";
      if (data.getGpu().queueCount() > 1) {
        data.tracks.addLabelGroup(
            "gpu", "gpu_queues", "GPU Queues", group(state -> new TitlePanel("GPU Queues"), true));
        parent = "gpu_queues";
      }
      for (GpuInfo.Queue queue : data.getGpu().queues()) {
        SliceTrack track = SliceTrack.forGpuQueue(data.qe, queue);
        data.tracks.addTrack(parent, track.getId(), queue.getDisplay(),
            single(state -> new GpuQueuePanel(state, queue, track), true, false));
      }
    }

    if (data.getGpu().vkApiEventCount() > 0) {
      String parent = "gpu";
      if (data.getGpu().vkApiEventCount() > 1) {
        data.tracks.addLabelGroup(
            "gpu", "vk_api_events", "Vulkan API Events", group(state -> new TitlePanel("Vulkan API Events"), true));
        parent = "vk_api_events";
      }
      for (GpuInfo.VkApiEvent vkApiEvent : data.getGpu().vkApiEvents()) {
        VulkanEventTrack track = new VulkanEventTrack(data.qe, vkApiEvent);
        data.tracks.addTrack(parent, track.getId(), vkApiEvent.getDisplay(),
            single(state -> new VulkanEventPanel(state, vkApiEvent, track), true, false));
      }
    }

    if (!counters.isEmpty()) {
      String parent = "gpu";
      if (counters.size() > 1) {
        data.tracks.addLabelGroup("gpu", "gpu_counters", "GPU Counters",
            group(state -> new TitlePanel("GPU Counters"), true));
        parent = "gpu_counters";
      }
      Map<CounterInfo, CounterTrack> addedTracks = Maps.newHashMap();
      Multimap<Long, Long> groups = data.getGpuCounterGroups();
      if (groups.keySet().size() > 1) {
        for (GpuCounterGroup group : GpuCounterGroup.values()) {
          if (group == GpuCounterGroup.UNRECOGNIZED) {
            continue;
          }
          Collection<Long> grouped_counters = groups.get(Long.valueOf(group.getNumber()));
          if (grouped_counters.isEmpty()) {
            continue;
          }
          parent = "gpu_counters_group_" + group.name();
          String name = GPU_COUNTER_GROUP_NAMES.get(group);
          data.tracks.addLabelGroup("gpu_counters", parent, name,
              group(state -> new TitlePanel(name), true));
          for (Long counter_id : grouped_counters) {
            CounterInfo counter = counters.get(counter_id);
            if (counter == null) {
              continue;
            }
            CounterTrack track = addedTracks.computeIfAbsent(counter, res -> new CounterTrack(data.qe, counter));
            data.tracks.addTrack(parent, group.name() + track.getId(), counter.name,
                single(state -> new CounterPanel(state, track, DEFAULT_COUNTER_TRACK_HEIGHT), true,
                  /*right truncate*/ true));
          }
        }
      } else {
        for (CounterInfo counter : counters.values()) {
          CounterTrack track = new CounterTrack(data.qe, counter);
          data.tracks.addTrack(parent, track.getId(), counter.name,
              single(state -> new CounterPanel(state, track, DEFAULT_COUNTER_TRACK_HEIGHT), true,
                  /*right truncate*/ true));
        }
      }
    }
    return data;
  }

  public static Perfetto.Data.Builder enumerateFrame(Perfetto.Data.Builder data) {
    if (data.getFrame().layerCount() > 0) {
      String parent = "sf_events";
      data.tracks.addLabelGroup(null, parent, "Surface Flinger Events",
          group(state -> new TitlePanel("Surface Flinger Events"), true));

      // We assume here that the target application's layer will generally have more
      // events than unintended layers from the likes of StatusBar, NavBar etc.
      long maxEvents = 0;
      for (FrameInfo.Layer layer : data.getFrame().layers()) {
        if (layer.numEvents > maxEvents) {
          maxEvents = layer.numEvents;
        }
      }

      for (FrameInfo.Layer layer : data.getFrame().layers()) {
        boolean expanded = (layer.numEvents == maxEvents);
        data.tracks.addLabelGroup(parent, layer.layerName, layer.layerName,
            group(state -> new TitlePanel("Layer - " + layer.layerName), expanded));
        for (FrameInfo.Event phase : layer.phaseEvents()) {
          FrameEventsTrack track = FrameEventsTrack.forFrameEvent(data.qe, layer.layerName, phase);
          data.tracks.addTrack(layer.layerName, track.getId(), phase.getDisplay(),
              single(state -> new FrameEventsPanel(state, phase, track), true, false));
        }
        String buffersGroup = "buffers_" + layer.layerName;
        data.tracks.addLabelGroup(layer.layerName, buffersGroup, "Buffers",
            group(state -> new TitlePanel("Buffers"), false));
        for (FrameInfo.Event buffer : layer.bufferEvents()) {
          FrameEventsTrack track = FrameEventsTrack.forFrameEvent(data.qe, layer.layerName, buffer);
          data.tracks.addTrack(buffersGroup, track.getId(), buffer.getDisplay(),
              single(state -> new FrameEventsPanel(state, buffer, track), true, true));
        }
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
      boolean isIdleProcess = hasIdles && (process.totalDur < idleCutoffProc);
      String parent =  isIdleProcess ? "procs_idle" : "procs";
      data.tracks.addGroup(parent, summary.getId(), process.getDisplay(),
          group(state -> new ProcessSummaryPanel(state, summary), false));

      // For each process, add Vulkan memory usage counters if any exist.
      List<CounterInfo> counters = data.getCounters().values().stream()
          .filter(c -> c.type == CounterInfo.Type.Process && c.ref == process.upid &&
              c.count > 0 && c.name.startsWith("vulkan"))
          .collect(toList());
      if (!counters.isEmpty()) {
        String groupId = "vulkan_counters_" + process.upid;
        data.tracks.addLabelGroup(summary.getId(), groupId, "Vulkan Memory Usage",
            group(state -> new TitlePanel("Vulkan Memory Usage"), true));
        for (int i = 0; i < counters.size(); i++) {
          CounterInfo counter = counters.get(i);
          boolean last = i == counters.size() - 1;
          CounterTrack track = new CounterTrack(data.qe, counter);
          data.tracks.addTrack(groupId, track.getId(), counter.name,
              single(state -> new VulkanCounterPanel(state, track), last, false));
        }
      }

      // For each process, add the memory usage panel if it exists.
      ProcessMemoryTrack.enumerate(data, summary.getId(), process.upid);

      // For each process, add any other process counters if any exist.
      counters = data.getCounters().values().stream()
          .filter(c -> c.type == CounterInfo.Type.Process && c.ref == process.upid &&
              c.count > 0 && shouldShowProcessCounter(c))
          .collect(toList());
      if (!counters.isEmpty()) {
        String parentId = summary.getId();
        if (counters.size() > 3) {
          parentId = "proc_counters_" + process.upid;
          data.tracks.addLabelGroup(summary.getId(), parentId, "Process Counters",
              group(state -> new TitlePanel("Process Counters"), false));
        }
        for (int i = 0; i < counters.size(); i++) {
          CounterInfo counter = counters.get(i);
          boolean last = i == counters.size() - 1;
          CounterTrack track = new CounterTrack(data.qe, counter);
          data.tracks.addTrack(parentId, track.getId(), counter.name,
              single(state -> new CounterPanel(
                  state, track, PROCESS_COUNTER_TRACK_HEIGHT), last, false));
        }
      }

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
        boolean isIdleThread = hasIdleThreads && track.getThread().totalDur < idleCutoffThread;
        TrackConfig.Track.UiFactory<Panel> ui;
        if (track.getThread().maxDepth == 0) {
          ui = single(state -> new ThreadPanel(state, track, false), false, false);
        } else {
          boolean expanded = !isIdleProcess && !isIdleThread;
          ui = single(state -> new ThreadPanel(state, track, expanded), false,
              ThreadPanel::setExpanded, expanded, false);
        }
        String threadParent = isIdleThread ? summary.getId() + "_idle" : summary.getId();
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

  private static boolean shouldShowProcessCounter(CounterInfo counter) {
    if (counter.name.startsWith("vulkan")) {
      // Shown in the vulkan memory usage group.
      return false;
    }

    if (counter.name.startsWith("mem.rss.") || counter.name.startsWith("mem.ion.") ||
        "mem.swap".equals(counter.name) ||  "Heap size (KB)".equals(counter.name)) {
      // Memory counters get their own UI.
      return false;
    }

    if (PROC_STATS_COUNTER.contains(counter.name)) {
      // The process_stat counters are too infrequent to be helpful.
      return false;
    }

    if ("VSYNC-app".equals(counter.name)) {
      // VSync has a custom UI.
      return false;
    }

    return true;
  }

  public static Perfetto.Data.Builder enumerateVSync(Perfetto.Data.Builder data) {
    List<CounterInfo> counters = data.getCounters(CounterInfo.Type.Process).get("VSYNC-app");
    if (counters.size() != 1) {
      return data;
    }

    return data.setVSync(new VSync.FromSurfaceFlingerAppCounter(data.qe, counters.get(0)));
  }
}
