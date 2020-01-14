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
package com.google.gapid.perfetto.views;

import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_COMMAND_BUFFER;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_INSTANCE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_PHYSICAL_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_QUEUE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.MemoryTracking.MEMORY_TRACKING_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.MemoryTracking.MEMORY_TRACKING_DRIVER;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static java.util.concurrent.TimeUnit.SECONDS;
import static java.util.stream.Collectors.joining;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.ImmutableMap;
import com.google.common.collect.Lists;
import com.google.common.collect.Sets;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.GpuProfiling;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;
import com.google.protobuf.ProtocolMessageEnum;

import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.Table;
import org.eclipse.swt.widgets.TableColumn;
import org.eclipse.swt.widgets.TableItem;

import java.util.Arrays;
import java.util.List;
import java.util.Objects;
import java.util.Set;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

import perfetto.protos.PerfettoConfig;
import perfetto.protos.PerfettoConfig.TraceConfig.BufferConfig.FillPolicy;

public class TraceConfigDialog extends DialogBase {
  private static final int BUFFER_SIZE = 131072;
  private static final int FTRACE_BUFFER_SIZE = 8192;
  private static final String[] CPU_BASE_FTRACE = {
      "sched/sched_switch",
      "sched/sched_process_exit",
      "sched/sched_process_free",
      "task/task_newtask",
      "task/task_rename",
      "power/suspend_resume",
  };
  private static final String[] CPU_FREQ_FTRACE = {
      "power/cpu_frequency",
      "power/cpu_idle"
  };
  private static final String[] CPU_CHAIN_FTRACE = {
      "sched/sched_wakeup",
      "sched/sched_wakeup_new",
      "sched/sched_waking",
  };
  private static final String[] CPU_SLICES_ATRACE = {
      // TODO: this should come from the device.
      "adb", "aidl", "am", "audio", "binder_driver", "binder_lock", "bionic", "camera",
      "core_services", "dalvik", "database", "disk", "freq", "gfx", "hal", "idle", "input",
      "ion", "memory", "memreclaim", "network", "nnapi", "pdx", "pm", "power", "res", "rro", "rs",
      "sched", "sm", "ss", "sync", "vibrator", "video", "view", "webview", "wm",
  };
  private static final String[] GPU_FREQ_FTRACE = {
      "power/gpu_frequency",
  };
  private static final PerfettoConfig.MeminfoCounters[] MEM_COUNTERS = {
      PerfettoConfig.MeminfoCounters.MEMINFO_MEM_TOTAL,
      PerfettoConfig.MeminfoCounters.MEMINFO_MEM_FREE,
      PerfettoConfig.MeminfoCounters.MEMINFO_BUFFERS,
      PerfettoConfig.MeminfoCounters.MEMINFO_CACHED,
      PerfettoConfig.MeminfoCounters.MEMINFO_SWAP_CACHED,
  };
  private static final PerfettoConfig.AndroidPowerConfig.BatteryCounters[] BAT_COUNTERS = {
      PerfettoConfig.AndroidPowerConfig.BatteryCounters.BATTERY_COUNTER_CAPACITY_PERCENT,
      PerfettoConfig.AndroidPowerConfig.BatteryCounters.BATTERY_COUNTER_CHARGE,
      PerfettoConfig.AndroidPowerConfig.BatteryCounters.BATTERY_COUNTER_CURRENT,
  };

  private static final ImmutableMap<ProtocolMessageEnum, String> VK_LABLES =
      ImmutableMap.<ProtocolMessageEnum, String> builder()
        .put(CPU_TIMING_COMMAND_BUFFER, "VkCommandBuffer")
        .put(CPU_TIMING_DEVICE, "VkDevice")
        .put(CPU_TIMING_INSTANCE, "VkInstance")
        .put(CPU_TIMING_PHYSICAL_DEVICE, "VkPhysicalDevice")
        .put(CPU_TIMING_QUEUE, "VkQueue")
        .put(MEMORY_TRACKING_DEVICE, "Device")
        .put(MEMORY_TRACKING_DRIVER, "Driver")
        .build();

  private static final Pattern APP_REGEX = Pattern.compile("(?:[^:]*)?:([^/]+)(?:/[^/]+)");

  private final Settings settings;
  private final Device.PerfettoCapability caps;
  private InputArea input;

  public TraceConfigDialog(
      Shell shell, Settings settings, Theme theme, Device.PerfettoCapability caps) {
    super(shell, theme);
    this.settings = settings;
    this.caps = caps;
  }

  public static void showPerfettoConfigDialog(
      Shell shell, Models models, Widgets widgets, Device.PerfettoCapability caps) {
    new TraceConfigDialog(shell, models.settings, widgets.theme, caps).open();
  }

  public static String getConfigSummary(Settings settings, Device.PerfettoCapability caps) {
    List<String> enabled = Lists.newArrayList();
    SettingsProto.PerfettoOrBuilder p = settings.perfetto();
    if (p.getCpuOrBuilder().getEnabled()) {
      enabled.add("CPU");
    }
    Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
    if (gpuCaps.getHasRenderStage() ||
        gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0 ||
        gpuCaps.getHasFrameLifecycle()) {
      if (p.getGpuOrBuilder().getEnabled()) {
        enabled.add("GPU");
      }
    }
    if (p.getMemoryOrBuilder().getEnabled()) {
      enabled.add("Memory");
    }
    if (p.getBatteryOrBuilder().getEnabled()) {
      enabled.add("Battery");
    }
    if (p.getVulkanOrBuilder().getEnabled()) {
      Device.VulkanProfilingLayers vkLayers = caps.getVulkanProfileLayers();
      SettingsProto.Perfetto.VulkanOrBuilder vk = p.getVulkanOrBuilder();
      if ((vk.getCpuTiming() && vkLayers.getCpuTiming()) ||
          (vk.getMemoryTracking() && vkLayers.getMemoryTracker())) {
        enabled.add("Vulkan");
      }
    }
    return enabled.stream().collect(joining(", "));
  }

  public static PerfettoConfig.TraceConfig.Builder getConfig(
      Settings settings, Device.PerfettoCapability caps, String traceTarget) {
    PerfettoConfig.TraceConfig.Builder config = PerfettoConfig.TraceConfig.newBuilder();
    SettingsProto.PerfettoOrBuilder p = settings.perfetto();

    PerfettoConfig.FtraceConfig.Builder ftrace = null;
    if (p.getCpuOrBuilder().getEnabled() || (p.getGpuOrBuilder().getEnabled())) {
      ftrace = config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("linux.ftrace")
                .getFtraceConfigBuilder()
                    .setBufferSizeKb(FTRACE_BUFFER_SIZE);
    }

    if (p.getCpuOrBuilder().getEnabled()) {
      // Record process names.
      config.addDataSourcesBuilder()
          .getConfigBuilder()
              .setName("linux.process_stats")
              .getProcessStatsConfigBuilder()
                  .setScanAllProcessesOnStart(true);

      ftrace.addAllFtraceEvents(Arrays.asList(CPU_BASE_FTRACE));
      if (p.getCpuOrBuilder().getFrequency()) {
        ftrace.addAllFtraceEvents(Arrays.asList(CPU_FREQ_FTRACE));
      }
      if (p.getCpuOrBuilder().getChain()) {
        ftrace.addAllFtraceEvents(Arrays.asList(CPU_CHAIN_FTRACE));
      }
      if (p.getCpuOrBuilder().getSlices() && caps.getCanSpecifyAtraceApps()) {
        ftrace.addAllAtraceCategories(Arrays.asList(CPU_SLICES_ATRACE));
        if (!traceTarget.isEmpty()) {
          Matcher m = APP_REGEX.matcher(traceTarget);
          ftrace.addAtraceApps(m.matches() ? m.group(1) : traceTarget);
        }
      }
    }

    if (p.getGpuOrBuilder().getEnabled()) {
      ftrace.addAllFtraceEvents(Arrays.asList(GPU_FREQ_FTRACE));

      Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
      SettingsProto.Perfetto.GPUOrBuilder gpu = p.getGpuOrBuilder();
      if (gpuCaps.getHasRenderStage() && gpu.getSlices()) {
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("gpu.renderstages");
      }
      if (gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0 &&
          gpu.getCounters() && gpu.getCounterIdsCount() > 0) {
        PerfettoConfig.GpuCounterConfig.Builder counters = config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("gpu.counters")
                .getGpuCounterConfigBuilder()
                    .setCounterPeriodNs(MILLISECONDS.toNanos(gpu.getCounterRate()));
        counters.addAllCounterIds(gpu.getCounterIdsList());

        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("VulkanAPI");
      }
      if (gpuCaps.getHasFrameLifecycle() && gpu.getSurfaceFlinger()) {
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("android.surfaceflinger.frame");
      }
    }

    if (p.getMemoryOrBuilder().getEnabled()) {
      config.addDataSourcesBuilder()
          .getConfigBuilder()
              .setName("linux.sys_stats")
              .getSysStatsConfigBuilder()
                  .setMeminfoPeriodMs(p.getMemoryOrBuilder().getRate())
                  .addAllMeminfoCounters(Arrays.asList(MEM_COUNTERS));
    }

    if (p.getBatteryOrBuilder().getEnabled()) {
      config.addDataSourcesBuilder()
          .getConfigBuilder()
              .setName("android.power")
              .getAndroidPowerConfigBuilder()
                  .setBatteryPollMs(p.getBatteryOrBuilder().getRate())
                  .setCollectPowerRails(true)
                  .addAllBatteryCounters(Arrays.asList(BAT_COUNTERS));
    }

    boolean largeBuffer = false;
    if (p.getVulkanOrBuilder().getEnabled()) {
      Device.VulkanProfilingLayers vkLayers = caps.getVulkanProfileLayers();
      SettingsProto.Perfetto.VulkanOrBuilder vk = p.getVulkanOrBuilder();
      if (vkLayers.getCpuTiming() && vk.getCpuTiming()) {
        largeBuffer = true;
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("VulkanCPUTiming")
                .setLegacyConfig(vkLabels(vk.getCpuTimingCategoriesList()));
      }
      if (vkLayers.getMemoryTracker() && vk.getMemoryTracking()) {
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("VulkanMemoryTracker")
                .setLegacyConfig(vkLabels(vk.getMemoryTrackingCategoriesList()));
      }
    }

    config.addBuffers(PerfettoConfig.TraceConfig.BufferConfig.newBuilder()
        .setSizeKb((largeBuffer ? 8 : 1) * BUFFER_SIZE)
        .setFillPolicy(FillPolicy.DISCARD));
    config.setFlushPeriodMs((int)SECONDS.toMillis(5));

    return config;
  }

  private static String vkLabels(List<?> list) {
    return list.stream()
      .map(VK_LABLES::get)
      .filter(Objects::nonNull)
      .distinct()
      .collect(joining(":"));
  }

  @Override
  public String getTitle() {
    return Messages.CAPTURE_TRACE_PERFETTO;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);
    input = withLayoutData(
        new InputArea(area, settings, theme, caps), new GridData(GridData.FILL_BOTH));
    return area;
  }

  @Override
  protected void okPressed() {
    input.update(settings);
    super.okPressed();
  }

  private static class InputArea extends Composite {
    private static final int GROUP_INDENT = 20;

    private final Button cpu;
    private final Button cpuFreq;
    private final Button cpuChain;
    private final Button cpuSlices;

    private final Button gpu;
    private final Button gpuSlices;
    private final Button gpuCounters;
    private final Label[] gpuCountersLabels;
    private final Button gpuCountersSelect;
    private final Spinner gpuCountersRate;
    private final Button gpuFrame;

    private final Button mem;
    private final Label[] memLabels;
    private final Spinner memRate;

    private final Button bat;
    private final Label[] batLabels;
    private final Spinner batRate;
    private final Button vulkan;
    private final Button vulkanCPUTiming;
    private final Button vulkanCPUTimingCommandBuffer;
    private final Button vulkanCPUTimingDevice;
    private final Button vulkanCPUTimingInstance;
    private final Button vulkanCPUTimingPhysicalDevice;
    private final Button vulkanCPUTimingQueue;

    private final Button vulkanMemoryTracking;
    private final Button vulkanMemoryTrackingDevice;
    private final Button vulkanMemoryTrackingDriver;

    public InputArea(
        Composite parent, Settings settings, Theme theme, Device.PerfettoCapability caps) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(1, false));

      SettingsProto.Perfetto.CPUOrBuilder sCpu = settings.perfetto().getCpuOrBuilder();
      SettingsProto.Perfetto.GPUOrBuilder sGpu = settings.perfetto().getGpuOrBuilder();
      SettingsProto.Perfetto.MemoryOrBuilder sMem = settings.perfetto().getMemoryOrBuilder();
      SettingsProto.Perfetto.BatteryOrBuilder sBatt = settings.perfetto().getBatteryOrBuilder();
      SettingsProto.Perfetto.VulkanOrBuilder sVk = settings.perfetto().getVulkanOrBuilder();

      cpu = createCheckbox(this, "CPU", sCpu.getEnabled(), e -> updateCpu());
      Composite cpuGroup = withLayoutData(
          createComposite(this, withMargin(new GridLayout(1, false), 5, 0)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      cpuFreq = createCheckbox(cpuGroup, "Frequency and idle states", sCpu.getFrequency());
      cpuChain = createCheckbox(cpuGroup, "Scheduling chains / latency", sCpu.getChain());
      cpuSlices = createCheckbox(cpuGroup, "Thread slices", sCpu.getSlices());
      addSeparator();

      Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
      if (gpuCaps.getHasRenderStage() ||
          gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0 ||
          gpuCaps.getHasFrameLifecycle()) {
        gpu = createCheckbox(this, "GPU", sGpu.getEnabled(), e -> updateGpu());
        Composite gpuGroup = withLayoutData(
            createComposite(this, withMargin(new GridLayout(1, false), 5, 0)),
            withIndents(new GridData(), GROUP_INDENT, 0));
        if (gpuCaps.getHasRenderStage()) {
          gpuSlices = createCheckbox(gpuGroup, "Renderstage slices", sGpu.getSlices());
        } else {
          gpuSlices = null;
        }

        if (gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0) {
          gpuCounters = createCheckbox(
              gpuGroup, "Counters", sGpu.getCounters(), e -> updateGpu());
          Composite counterGroup = withLayoutData(
              createComposite(gpuGroup, withMargin(new GridLayout(3, false), 5, 0)),
              withIndents(new GridData(), GROUP_INDENT, 0));
          gpuCountersLabels = new Label[3];
          gpuCountersLabels[1] = createLabel(counterGroup, "Poll Rate:");
          gpuCountersRate = createSpinner(counterGroup, sGpu.getCounterRate(), 1, 1000);
          gpuCountersLabels[2] = createLabel(counterGroup, "ms");

          gpuCountersLabels[0] = createLabel(counterGroup, sGpu.getCounterIdsCount() + " selected");
          gpuCountersSelect = Widgets.createButton(counterGroup, "Select", e -> {
            GpuCountersDialog dialog =
                new GpuCountersDialog(getShell(), theme, caps, sGpu.getCounterIdsList());
            if (dialog.open() == Window.OK) {
              List<Integer> newIds = dialog.getSelectedIds();
              settings.writePerfetto().getGpuBuilder()
                  .clearCounterIds()
                  .addAllCounterIds(newIds)
                  .setCounters(!newIds.isEmpty());
              gpuCountersLabels[0].setText(newIds.size() + " selected");
              gpuCountersLabels[0].requestLayout();
              updateGpu();
            }
          });
        } else {
          gpuCounters = null;
          gpuCountersLabels = null;
          gpuCountersRate = null;
          gpuCountersSelect = null;
        }

        if (gpuCaps.getHasFrameLifecycle()) {
          gpuFrame = createCheckbox(
              gpuGroup, "Frame Lifecycle", sGpu.getSurfaceFlinger(), e -> updateGpu());
        } else {
          gpuFrame = null;
        }
        addSeparator();
      } else {
        gpu = null;
        gpuSlices = null;
        gpuCounters = null;
        gpuCountersLabels = null;
        gpuCountersRate = null;
        gpuCountersSelect = null;
        gpuFrame = null;
      }

      mem = createCheckbox(this, "Memory", sMem.getEnabled(), e -> updateMem());
      memLabels = new Label[2];
      Composite memGroup = withLayoutData(
          createComposite(this, withMargin(new GridLayout(3, false), 5, 0)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      memLabels[0] = createLabel(memGroup, "Poll Rate:");
      memRate = createSpinner(memGroup, sMem.getRate(), 1, 1000);
      memLabels[1] = createLabel(memGroup, "ms");
      addSeparator();

      bat = createCheckbox(this, "Battery", sBatt.getEnabled(), e -> updateBat());
      batLabels = new Label[2];
      Composite batGroup = withLayoutData(
          createComposite(this, withMargin(new GridLayout(3, false), 5, 0)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      batLabels[0] = createLabel(batGroup, "Poll Rate:");
      batRate = createSpinner(batGroup, sBatt.getRate(), 250, 60000);
      batLabels[1] = createLabel(batGroup, "ms");

      Device.VulkanProfilingLayers vkLayers = caps.getVulkanProfileLayers();
      if (vkLayers.getCpuTiming() || vkLayers.getMemoryTracker()) {
        addSeparator();
        vulkan = createCheckbox(this, "Vulkan", sVk.getEnabled(), e -> updateVulkan());
        Composite vkGroup = withLayoutData(
            createComposite(this, new GridLayout(1, false)),
            withIndents(new GridData(), GROUP_INDENT, 0));

        if (vkLayers.getCpuTiming()) {
          vulkanCPUTiming = createCheckbox(
              vkGroup, "CPU Timing", sVk.getCpuTiming(), e -> updateVulkan());
          Composite cpuTimingGroup = withLayoutData(
              createComposite(vkGroup, withMargin(new GridLayout(1, false), 5, 0)),
              withIndents(new GridData(), GROUP_INDENT, 0));

          vulkanCPUTimingInstance =
              createCheckbox(cpuTimingGroup, "Instance", hasCategory(sVk, CPU_TIMING_INSTANCE));
          vulkanCPUTimingPhysicalDevice = createCheckbox(
              cpuTimingGroup, "Physical Device", hasCategory(sVk, CPU_TIMING_PHYSICAL_DEVICE));
          vulkanCPUTimingDevice =
              createCheckbox(cpuTimingGroup, "Device", hasCategory(sVk, CPU_TIMING_DEVICE));
          vulkanCPUTimingQueue =
              createCheckbox(cpuTimingGroup, "Queue", hasCategory(sVk, CPU_TIMING_QUEUE));
          vulkanCPUTimingCommandBuffer = createCheckbox(
              cpuTimingGroup, "CommandBuffer", hasCategory(sVk, CPU_TIMING_COMMAND_BUFFER));
        } else {
          vulkanCPUTiming = null;
          vulkanCPUTimingInstance = null;
          vulkanCPUTimingPhysicalDevice = null;
          vulkanCPUTimingDevice = null;
          vulkanCPUTimingQueue = null;
          vulkanCPUTimingCommandBuffer = null;
        }
        if (caps.getVulkanProfileLayers().getMemoryTracker()) {
          vulkanMemoryTracking = createCheckbox(
              vkGroup, "Memory Tracking", sVk.getMemoryTracking(), e -> updateVulkan());
          Composite memoryTrackingGroup = withLayoutData(
              createComposite(vkGroup, withMargin(new GridLayout(1, false), 5, 0)),
              withIndents(new GridData(), GROUP_INDENT, 0));

          vulkanMemoryTrackingDevice = createCheckbox(
              memoryTrackingGroup, "Device", hasCategory(sVk, MEMORY_TRACKING_DEVICE));
          vulkanMemoryTrackingDriver = createCheckbox(
              memoryTrackingGroup, "Driver", hasCategory(sVk, MEMORY_TRACKING_DRIVER));
        } else {
          vulkanMemoryTracking = null;
          vulkanMemoryTrackingDevice = null;
          vulkanMemoryTrackingDriver = null;
        }
      } else {
        vulkan = null;
        vulkanCPUTiming = null;
        vulkanCPUTimingInstance = null;
        vulkanCPUTimingPhysicalDevice = null;
        vulkanCPUTimingDevice = null;
        vulkanCPUTimingQueue = null;
        vulkanCPUTimingCommandBuffer = null;
        vulkanMemoryTracking = null;
        vulkanMemoryTrackingDevice = null;
        vulkanMemoryTrackingDriver = null;
      }

      updateCpu();
      updateGpu();
      updateMem();
      updateBat();
      updateVulkan();
    }

    private static boolean hasCategory(
        SettingsProto.Perfetto.VulkanOrBuilder vk, SettingsProto.Perfetto.Vulkan.CpuTiming cat) {
      return vk.getCpuTimingCategoriesList().contains(cat);
    }

    private static boolean hasCategory(SettingsProto.Perfetto.VulkanOrBuilder vk,
        SettingsProto.Perfetto.Vulkan.MemoryTracking cat) {
      return vk.getMemoryTrackingCategoriesList().contains(cat);
    }

    public void update(Settings settings) {
      SettingsProto.Perfetto.CPU.Builder sCpu = settings.writePerfetto().getCpuBuilder();
      SettingsProto.Perfetto.GPU.Builder sGpu = settings.writePerfetto().getGpuBuilder();
      SettingsProto.Perfetto.Memory.Builder sMem = settings.writePerfetto().getMemoryBuilder();
      SettingsProto.Perfetto.Battery.Builder sBatt = settings.writePerfetto().getBatteryBuilder();
      SettingsProto.Perfetto.Vulkan.Builder sVk = settings.writePerfetto().getVulkanBuilder();

      sCpu.setEnabled(cpu.getSelection());
      sCpu.setChain(cpuChain.getSelection());
      sCpu.setFrequency(cpuFreq.getSelection());
      sCpu.setSlices(cpuSlices.getSelection());

      if (gpu != null) {
        sGpu.setEnabled(gpu.getSelection());
      }
      if (gpuSlices != null) {
        sGpu.setSlices(gpuSlices.getSelection());
      }
      if (gpuCounters != null) {
        sGpu.setCounters(gpuCounters.getSelection());
        sGpu.setCounterRate(gpuCountersRate.getSelection());
      }
      if (gpuFrame != null) {
        sGpu.setSurfaceFlinger(gpuFrame.getSelection());
      }

      sMem.setEnabled(mem.getSelection());
      sMem.setRate(memRate.getSelection());
      sBatt.setEnabled(bat.getSelection());
      sBatt.setRate(batRate.getSelection());

      if (vulkan != null) {
        sVk.setEnabled(vulkan.getSelection());
      }
      if (vulkanCPUTiming != null) {
        sVk.setCpuTiming(vulkanCPUTiming.getSelection());
        sVk.clearCpuTimingCategories();
        addCategory(vulkanCPUTimingCommandBuffer, sVk, CPU_TIMING_COMMAND_BUFFER);
        addCategory(vulkanCPUTimingDevice, sVk, CPU_TIMING_DEVICE);
        addCategory(vulkanCPUTimingPhysicalDevice, sVk, CPU_TIMING_PHYSICAL_DEVICE);
        addCategory(vulkanCPUTimingInstance, sVk, CPU_TIMING_INSTANCE);
        addCategory(vulkanCPUTimingQueue, sVk, CPU_TIMING_QUEUE);
      }
      if (vulkanMemoryTracking != null) {
        sVk.setMemoryTracking(vulkanMemoryTracking.getSelection());
        sVk.clearMemoryTrackingCategories();
        addCategory(vulkanMemoryTrackingDevice, sVk, MEMORY_TRACKING_DEVICE);
        addCategory(vulkanMemoryTrackingDriver, sVk, MEMORY_TRACKING_DRIVER);
      }
    }

    private static void addCategory(Button checkbox, SettingsProto.Perfetto.Vulkan.Builder vk,
        SettingsProto.Perfetto.Vulkan.CpuTiming cat) {
      if (checkbox.getSelection()) {
        vk.addCpuTimingCategories(cat);
      }
    }

    private static void addCategory(Button checkbox, SettingsProto.Perfetto.Vulkan.Builder vk,
        SettingsProto.Perfetto.Vulkan.MemoryTracking cat) {
      if (checkbox.getSelection()) {
        vk.addMemoryTrackingCategories(cat);
      }
    }

    private void addSeparator() {
      withLayoutData(new Label(this, SWT.SEPARATOR | SWT.HORIZONTAL),
          new GridData(GridData.FILL_HORIZONTAL));
    }

    private void updateCpu() {
      boolean enabled = cpu.getSelection();
      cpuFreq.setEnabled(enabled);
      cpuChain.setEnabled(enabled);
      cpuSlices.setEnabled(enabled);
    }

    private void updateGpu() {
      if (gpu == null) {
        return;
      }

      boolean enabled = gpu.getSelection();
      if (gpuSlices != null) {
        gpuSlices.setEnabled(enabled);
      }
      if (gpuCounters != null) {
        gpuCounters.setEnabled(enabled);
        boolean countersEnabled = enabled && gpuCounters.getSelection();
        gpuCountersRate.setEnabled(countersEnabled);
        gpuCountersSelect.setEnabled(countersEnabled);
        for (Label label : gpuCountersLabels) {
          label.setEnabled(countersEnabled);
        }
      }
      if (gpuFrame != null) {
        gpuFrame.setEnabled(enabled);
      }
    }

    private void updateVulkan() {
      if (vulkan == null) {
        return;
      }

      boolean vkEnabled = vulkan.getSelection();
      if (vulkanCPUTiming != null) {
        vulkanCPUTiming.setEnabled(vkEnabled);
        boolean enabled = vkEnabled && vulkanCPUTiming.getSelection();
        vulkanCPUTimingInstance.setEnabled(enabled);
        vulkanCPUTimingPhysicalDevice.setEnabled(enabled);
        vulkanCPUTimingDevice.setEnabled(enabled);
        vulkanCPUTimingQueue.setEnabled(enabled);
        vulkanCPUTimingCommandBuffer.setEnabled(enabled);
      }
      if (vulkanMemoryTracking != null) {
        vulkanMemoryTracking.setEnabled(vkEnabled);
        boolean enabled = vkEnabled && vulkanMemoryTracking.getSelection();
        vulkanMemoryTrackingDevice.setEnabled(enabled);
        vulkanMemoryTrackingDriver.setEnabled(enabled);
      }
    }

    private void updateMem() {
      boolean enabled = mem.getSelection();
      memRate.setEnabled(enabled);
      for (Label label : memLabels) {
        label.setEnabled(enabled);
      }
    }

    private void updateBat() {
      boolean enabled = bat.getSelection();
      batRate.setEnabled(enabled);
      for (Label label : batLabels) {
        label.setEnabled(enabled);
      }
    }

    private static class GpuCountersDialog extends DialogBase {
      private final Device.PerfettoCapability caps;
      private final Set<Integer> currentIds;

      private Table table;
      private List<Integer> selectedIds;

      public GpuCountersDialog(
          Shell shell, Theme theme, Device.PerfettoCapability caps, List<Integer> currentIds) {
        super(shell, theme);
        this.caps = caps;
        this.currentIds = Sets.newHashSet(currentIds);
      }

      public List<Integer> getSelectedIds() {
        return selectedIds;
      }

      @Override
      public String getTitle() {
        return Messages.CAPTURE_TRACE_PERFETTO;
      }

      @Override
      protected Control createDialogArea(Composite parent) {
        Composite area = (Composite)super.createDialogArea(parent);
        table = withLayoutData(new Table(area, SWT.CHECK), new GridData(GridData.FILL_BOTH));
        table.setHeaderVisible(true);
        table.setLinesVisible(true);
        new TableColumn(table, SWT.NONE).setText("Name");
        new TableColumn(table, SWT.NONE).setText("Description");
        for (GpuProfiling.GpuCounterDescriptor.GpuCounterSpec counter :
            caps.getGpuProfiling().getGpuCounterDescriptor().getSpecsList()) {
          TableItem item = new TableItem(table, SWT.NONE);
          item.setText(new String[] { counter.getName(), counter.getDescription() });
          item.setData(counter);
          if (currentIds.contains(counter.getCounterId())) {
            item.setChecked(true);
          }
        }
        table.getColumn(0).pack();
        table.getColumn(1).pack();
        createLink(area, "Select <a>none</a> | <a>all</a>", e -> {
          boolean checked = "all".equals(e.text);
          for (TableItem item : table.getItems()) {
            item.setChecked(checked);
          }
        });
        return area;
      }

      @Override
      protected Point getInitialSize() {
        return new Point(convertHorizontalDLUsToPixels(450), convertVerticalDLUsToPixels(300));
      }

      @Override
      protected void okPressed() {
        selectedIds = Arrays.stream(table.getItems())
            .filter(item -> item.getChecked())
            .map(item -> (GpuProfiling.GpuCounterDescriptor.GpuCounterSpec)item.getData())
            .mapToInt(GpuProfiling.GpuCounterDescriptor.GpuCounterSpec::getCounterId)
            .boxed()
            .collect(toList());
        super.okPressed();
      }
    }
  }
}
