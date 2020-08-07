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

import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_INSTANCE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_PHYSICAL_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.CpuTiming.CPU_TIMING_QUEUE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.MemoryTracking.MEMORY_TRACKING_DEVICE;
import static com.google.gapid.proto.SettingsProto.Perfetto.Vulkan.MemoryTracking.MEMORY_TRACKING_DRIVER;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createCheckboxTableViewer;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createTextarea;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static java.util.concurrent.TimeUnit.MICROSECONDS;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static java.util.concurrent.TimeUnit.NANOSECONDS;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.joining;
import static java.util.stream.Collectors.toList;

import com.google.common.base.Predicate;
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
import com.google.protobuf.TextFormat;
import com.google.protobuf.TextFormat.ParseException;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.CheckStateChangedEvent;
import org.eclipse.jface.viewers.CheckboxTableViewer;
import org.eclipse.jface.viewers.ICheckStateListener;
import org.eclipse.jface.viewers.Viewer;
import org.eclipse.jface.viewers.ViewerFilter;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.Text;

import java.util.Arrays;
import java.util.List;
import java.util.Objects;
import java.util.Set;
import java.util.function.Consumer;
import java.util.logging.Logger;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.util.stream.Collectors;

import perfetto.protos.PerfettoConfig;
import perfetto.protos.PerfettoConfig.TraceConfig.BufferConfig.FillPolicy;

public class TraceConfigDialog extends DialogBase {
  protected static final Logger LOG = Logger.getLogger(TraceConfigDialog.class.getName());

  private static final int MAIN_BUFFER_SIZE = 131072;
  private static final int PROC_BUFFER_SIZE = 4096;
  private static final int PROC_BUFFER = 1;
  // Kernel ftrace buffer size per CPU.
  private static final int FTRACE_BUFFER_SIZE = 8192;

  private static final int PROC_SCAN_PERIOD = 2000;
  private static final int FTRACE_DRAIN_PERIOD = 250;

  private static final int MAX_IN_MEM_DURATION = 15 * 1000;
  private static final int FLUSH_PERIOD = 5000;
  private static final int WRITE_PERIOD = 2000;
  private static final long MAX_FILE_SIZE = 2l * 1024 * 1024 * 1024;

  // These ftrace categories are always enabled to track process creation and ending.
  private static final String[] PROCESS_TRACKING_FTRACE = {
    "sched/sched_process_free",
    "task/task_newtask",
    "task/task_rename",
  };
  // These ftrace categories are used to track CPU slices.
  private static final String[] CPU_BASE_FTRACE = {
      "sched/sched_switch",
      "power/suspend_resume",
  };
  // These ftrace categories provide CPU frequency data.
  private static final String[] CPU_FREQ_FTRACE = {
      "power/cpu_frequency",
      "power/cpu_idle"
  };
  // These ftrace categories provide scheduling dependency data.
  private static final String[] CPU_CHAIN_FTRACE = {
      "sched/sched_wakeup",
      "sched/sched_wakeup_new",
      "sched/sched_waking",
  };
  // These ftrace categories provide memory usage data.
  private static final String[] MEM_FTRACE = {
      "kmem/rss_stat",
  };
  private static final String[] CPU_SLICES_ATRACE = {
      "am", "audio", "gfx", "hal", "input", "pm", "power", "res", "rs", "sm", "video", "view", "wm",
  };
  private static final String[] GPU_FREQ_FTRACE = {
      "power/gpu_frequency",
  };
  private static final String[] GPU_MEM_FTRACE = {
      "gpu_mem/gpu_mem_total",
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
    SettingsProto.PerfettoOrBuilder p = settings.perfetto();
    if (p.getUseCustom()) {
      return "Custom";
    }

    List<String> enabled = Lists.newArrayList();
    if (p.getCpuOrBuilder().getEnabled()) {
      enabled.add("CPU");
    }
    Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
    if (gpuCaps.getHasRenderStage() ||
        gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0 ||
        caps.getHasFrameLifecycle()) {
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
      Settings settings, Device.PerfettoCapability caps, String traceTarget, int duration) {
    SettingsProto.PerfettoOrBuilder p = settings.perfetto();
    if (p.getUseCustom()) {
      return p.getCustomConfig().toBuilder().setDurationMs(duration);
    }

    PerfettoConfig.TraceConfig.Builder config = PerfettoConfig.TraceConfig.newBuilder();
    PerfettoConfig.FtraceConfig.Builder ftrace = config.addDataSourcesBuilder()
        .getConfigBuilder()
            .setName("linux.ftrace")
            .getFtraceConfigBuilder()
            .addAllFtraceEvents(Arrays.asList(PROCESS_TRACKING_FTRACE))
            .setDrainPeriodMs(FTRACE_DRAIN_PERIOD)
            .setBufferSizeKb(FTRACE_BUFFER_SIZE)
            .setCompactSched(PerfettoConfig.FtraceConfig.CompactSchedConfig.newBuilder()
                .setEnabled(true));
    // Record process names at startup into the metadata buffer.
    config.addDataSourcesBuilder()
        .getConfigBuilder()
            .setName("linux.process_stats")
            .setTargetBuffer(PROC_BUFFER)
            .getProcessStatsConfigBuilder()
                .setScanAllProcessesOnStart(true);
    // Periodically record process information into the main buffer.
    config.addDataSourcesBuilder()
        .getConfigBuilder()
            .setName("linux.process_stats")
            .getProcessStatsConfigBuilder()
                .setProcStatsPollMs(PROC_SCAN_PERIOD)
                .setProcStatsCacheTtlMs(10 * PROC_SCAN_PERIOD);

    if (p.getCpuOrBuilder().getEnabled()) {
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
      ftrace.addAllFtraceEvents(Arrays.asList(GPU_MEM_FTRACE));

      Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
      SettingsProto.Perfetto.GPUOrBuilder gpu = p.getGpuOrBuilder();
      if (gpuCaps.getHasRenderStage() && gpu.getSlices()) {
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("gpu.renderstages");
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("VulkanAPI");
      }
      if (gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0 &&
          gpu.getCounters() && gpu.getCounterIdsCount() > 0) {
        PerfettoConfig.GpuCounterConfig.Builder counters = config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("gpu.counters")
                .getGpuCounterConfigBuilder()
                    .setCounterPeriodNs(MICROSECONDS.toNanos(gpu.getCounterRate()));
        counters.addAllCounterIds(gpu.getCounterIdsList());
      }
      if (caps.getHasFrameLifecycle() && gpu.getSurfaceFlinger()) {
        config.addDataSourcesBuilder()
            .getConfigBuilder()
                .setName("android.surfaceflinger.frame");
      }
    }

    if (p.getMemoryOrBuilder().getEnabled()) {
      ftrace.addAllFtraceEvents(Arrays.asList(MEM_FTRACE));
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
                  .addAllBatteryCounters(Arrays.asList(BAT_COUNTERS));
    }

    if (p.getVulkanOrBuilder().getEnabled()) {
      Device.VulkanProfilingLayers vkLayers = caps.getVulkanProfileLayers();
      SettingsProto.Perfetto.VulkanOrBuilder vk = p.getVulkanOrBuilder();
      if (vkLayers.getCpuTiming() && vk.getCpuTiming()) {
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

    // Buffer 0 (default): main buffer.
    config.addBuffers(PerfettoConfig.TraceConfig.BufferConfig.newBuilder()
        .setSizeKb(MAIN_BUFFER_SIZE)
        .setFillPolicy(FillPolicy.DISCARD));
    // Buffer 1: Initial process metadata.
    config.addBuffers(PerfettoConfig.TraceConfig.BufferConfig.newBuilder()
        .setSizeKb(PROC_BUFFER_SIZE)
        .setFillPolicy(FillPolicy.DISCARD));

    config.setFlushPeriodMs(FLUSH_PERIOD);
    config.setDurationMs(duration);
    if (duration > MAX_IN_MEM_DURATION) {
      config.setWriteIntoFile(true);
      config.setFileWritePeriodMs(WRITE_PERIOD);
      config.setMaxFileSizeBytes(MAX_FILE_SIZE);
    }

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
    Composite container = withLayoutData(createComposite(area, new StackLayout()),
        new GridData(GridData.FILL_BOTH));

    InputArea[] areas = new InputArea[2];
    areas[0] = new BasicInputArea(
        container, settings, theme, caps, () -> switchTo(container, areas[1]));
    areas[1] = new AdvancedInputArea(
        container, () -> switchTo(container, areas[0]), this::setOkButtonEnabled);

    input = settings.perfetto().getUseCustom() ? areas[1] : areas[0];
    ((StackLayout)container.getLayout()).topControl = input.asControl();

    // Delay this, so the dialog size is computed only based on the basic dialog.
    scheduleIfNotDisposed(container, () -> input.onSwitchedTo(settings));

    return area;
  }

  private void switchTo(Composite container, InputArea newArea) {
    input = newArea;
    ((StackLayout)container.getLayout()).topControl = input.asControl();
    container.requestLayout();
    input.onSwitchedTo(settings);
    setOkButtonEnabled(true);
  }

  private void setOkButtonEnabled(boolean enabled) {
    Button button = getButton(IDialogConstants.OK_ID);
    if (button != null) {
      button.setEnabled(enabled);
    }
  }

  @Override
  protected void okPressed() {
    input.update(settings);
    super.okPressed();
  }

  private static interface InputArea {
    public default void onSwitchedTo(@SuppressWarnings("unused") Settings settings) {
      // Do nothing.
    }
    public void update(Settings settings);
    public default Control asControl() {
      return (Control)this;
    }
  }

  private static class BasicInputArea extends Composite implements InputArea {
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
    private final Button vulkanCPUTimingDevice;
    private final Button vulkanCPUTimingInstance;
    private final Button vulkanCPUTimingPhysicalDevice;
    private final Button vulkanCPUTimingQueue;

    private final Button vulkanMemoryTracking;
    private final Button vulkanMemoryTrackingDevice;
    private final Button vulkanMemoryTrackingDriver;

    public BasicInputArea(Composite parent, Settings settings, Theme theme,
        Device.PerfettoCapability caps, Runnable toAdvanced) {
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
          caps.getHasFrameLifecycle()) {
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

          long minSamplingPeriod = NANOSECONDS.toMicros(gpuCaps.getGpuCounterDescriptor().getMinSamplingPeriodNs());
          minSamplingPeriod = minSamplingPeriod > 0 ? minSamplingPeriod : MILLISECONDS.toMicros(1);
          long maxSamplingPeriod = NANOSECONDS.toMicros(gpuCaps.getGpuCounterDescriptor().getMaxSamplingPeriodNs());
          maxSamplingPeriod = maxSamplingPeriod > 0 ? maxSamplingPeriod : minSamplingPeriod * 1000;

          long counterRate = Math.max(sGpu.getCounterRate(), minSamplingPeriod);
          gpuCountersRate = createSpinner(counterGroup, (int)counterRate,
                                          (int)minSamplingPeriod, (int)maxSamplingPeriod);

          // If the minimum sampling period is smaller than 1ms, it means GPU
          // counters can be sampled at a higher rate than 1ms. And hence set
          // the incremental steps to 100us. Otherwise, it means the sampling
          // rate can not be faster than 1ms, and hence set the incremental
          // steps to 1ms, which is 1000us.
          if (MILLISECONDS.toMicros(1) > minSamplingPeriod) {
            gpuCountersRate.setIncrement(100);
          } else {
            gpuCountersRate.setIncrement(1000);
          }
          gpuCountersLabels[2] = createLabel(counterGroup, "us");

          long count = caps.getGpuProfiling().getGpuCounterDescriptor().getSpecsList().stream()
              .filter(c -> sGpu.getCounterIdsList().contains(c.getCounterId())).count();
          gpuCountersLabels[0] = createLabel(counterGroup, count + " selected");
          gpuCountersSelect = Widgets.createButton(counterGroup, "Select", e -> {
            List<Integer> currentIds = settings.perfetto().getGpuOrBuilder().getCounterIdsList();
            GpuCountersDialog dialog = new GpuCountersDialog(getShell(), theme, caps, currentIds);
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

        if (caps.getHasFrameLifecycle()) {
          gpuFrame = createCheckbox(
              gpuGroup, "Frame Lifecycle (experimental)", sGpu.getSurfaceFlinger(), e -> updateGpu());
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
        } else {
          vulkanCPUTiming = null;
          vulkanCPUTimingInstance = null;
          vulkanCPUTimingPhysicalDevice = null;
          vulkanCPUTimingDevice = null;
          vulkanCPUTimingQueue = null;
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
        vulkanMemoryTracking = null;
        vulkanMemoryTrackingDevice = null;
        vulkanMemoryTrackingDriver = null;
      }

      withLayoutData(createLink(this, "<a>Switch to advanced mode</a>", e -> {
        // Remember the input thus far and turn it into a proto to be modified by the user.
        update(settings);
        settings.writePerfetto().setCustomConfig(
            // Use a config that writes to file for custom by default.
            getConfig(settings, caps, "", MAX_IN_MEM_DURATION + 1)
                .clearDurationMs());
        toAdvanced.run();
      }), new GridData(SWT.END, SWT.BEGINNING, false, false));

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

    @Override
    public void update(Settings settings) {
      SettingsProto.Perfetto.CPU.Builder sCpu = settings.writePerfetto().getCpuBuilder();
      SettingsProto.Perfetto.GPU.Builder sGpu = settings.writePerfetto().getGpuBuilder();
      SettingsProto.Perfetto.Memory.Builder sMem = settings.writePerfetto().getMemoryBuilder();
      SettingsProto.Perfetto.Battery.Builder sBatt = settings.writePerfetto().getBatteryBuilder();
      SettingsProto.Perfetto.Vulkan.Builder sVk = settings.writePerfetto().getVulkanBuilder();
      settings.writePerfetto().setUseCustom(false);

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
      private static final Predicate<GpuProfiling.GpuCounterDescriptor.GpuCounterSpec>
          SELECT_DEFAULT = GpuProfiling.GpuCounterDescriptor.GpuCounterSpec::getSelectByDefault;

      private final Device.PerfettoCapability caps;
      private final Set<Integer> currentIds;

      private CheckboxTableViewer table;
      private List<Integer> selectedIds;
      private Set<Object> checkedElements;
      private boolean hasFilters;

      public GpuCountersDialog(
          Shell shell, Theme theme, Device.PerfettoCapability caps, List<Integer> currentIds) {
        super(shell, theme);
        this.caps = caps;
        this.currentIds = Sets.newHashSet(currentIds);
        this.checkedElements = Sets.newHashSet();
        this.hasFilters = false;
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
        Text search = new Text(area, SWT.SINGLE | SWT.SEARCH | SWT.ICON_SEARCH | SWT.ICON_CANCEL);
        search.setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

        table = createCheckboxTableViewer(area, SWT.NONE);
        table.getTable().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));
        Widgets.<GpuProfiling.GpuCounterDescriptor.GpuCounterSpec>createTableColumn(
            table, "Name", counter -> counter.getName());
        Widgets.<GpuProfiling.GpuCounterDescriptor.GpuCounterSpec>createTableColumn(
            table, "Description", counter -> counter.getDescription());
        table.setContentProvider(new ArrayContentProvider());
        table.setInput(caps.getGpuProfiling().getGpuCounterDescriptor().getSpecsList());
        table.setCheckedElements(
            caps.getGpuProfiling().getGpuCounterDescriptor().getSpecsList().stream()
                .filter(c -> currentIds.contains(c.getCounterId()))
                .toArray(GpuProfiling.GpuCounterDescriptor.GpuCounterSpec[]::new));
        table.getTable().getColumn(0).pack();
        table.getTable().getColumn(1).pack();
        collectCheckedElements();

        table.addCheckStateListener(new ICheckStateListener() {
          @Override
          public void checkStateChanged(CheckStateChangedEvent event) {
            if (event.getChecked()) {
              checkedElements.add(event.getElement());
            } else {
              checkedElements.remove(event.getElement());
            }
          }
        });

        createLink(area, "Select <a>none</a> | <a>default</a> | <a>all</a>", e -> {
          switch (e.text) {
            case "none":
              checkedElements.removeAll(Arrays.stream(table.getCheckedElements()).collect(Collectors.toSet()));
              table.setAllChecked(false);
              break;
            case "default":
              table.setCheckedElements(
                  caps.getGpuProfiling().getGpuCounterDescriptor().getSpecsList().stream()
                      .filter(SELECT_DEFAULT)
                      .toArray(GpuProfiling.GpuCounterDescriptor.GpuCounterSpec[]::new));
              if (hasFilters) {
                appendCheckedElements();
              } else {
                collectCheckedElements();
              }
              break;
            case "all":
              table.setAllChecked(true);
              appendCheckedElements();
              break;
          }
        });

        search.addListener(SWT.Modify, e -> {
          String query = search.getText().trim().toLowerCase();
          if (query.isEmpty()) {
            table.resetFilters();
            hasFilters = false;
            resumeCheckedElements();
            return;
          }
          table.setFilters(new ViewerFilter() {
            @Override
            public boolean select(Viewer viewer, Object parentElement, Object element) {
              return ((GpuProfiling.GpuCounterDescriptor.GpuCounterSpec)element)
                  .getName()
                  .toLowerCase()
                  .contains(query);
            }
          });
          hasFilters = true;
          resumeCheckedElements();
        });
        return area;
      }

      @Override
      protected Point getInitialSize() {
        return new Point(convertHorizontalDLUsToPixels(450), convertVerticalDLUsToPixels(300));
      }

      @Override
      protected void okPressed() {
        selectedIds = Arrays.stream(checkedElements.toArray())
            .map(item -> (GpuProfiling.GpuCounterDescriptor.GpuCounterSpec)item)
            .mapToInt(GpuProfiling.GpuCounterDescriptor.GpuCounterSpec::getCounterId)
            .boxed()
            .collect(toList());
        super.okPressed();
      }

      private void collectCheckedElements() {
        checkedElements = Arrays.stream(table.getCheckedElements()).collect(Collectors.toSet());
      }

      private void appendCheckedElements() {
        checkedElements.addAll(Arrays.stream(table.getCheckedElements()).collect(Collectors.toSet()));
      }

      private void resumeCheckedElements() {
        table.setCheckedElements(checkedElements.toArray());
      }
    }
  }

  private static class AdvancedInputArea extends Composite implements InputArea {
    private final Text input;

    public AdvancedInputArea(Composite parent, Runnable toBasic, Consumer<Boolean> okEnabled) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(1, false));
      input = withLayoutData(createTextarea(this, ""),
          new GridData(SWT.FILL, SWT.FILL, true, true));
      withLayoutData(createLink(
          this, "<a>Reset and switch back to basic</a>", e -> toBasic.run()),
          new GridData(SWT.END, SWT.BEGINNING, false, false));
      Label error = Widgets.createLabel(this, "");
      error.setVisible(false);
      error.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));

      input.addListener(SWT.Modify, ev -> {
        try {
          TextFormat.merge(input.getText(), PerfettoConfig.TraceConfig.newBuilder());
          error.setVisible(false);
          error.setText("");
          okEnabled.accept(true);
        } catch (ParseException e) {
          error.setVisible(true);
          error.setText("Parse Error: " + e.getMessage());
          okEnabled.accept(false);
        }
        error.requestLayout();
      });
    }

    @Override
    public void onSwitchedTo(Settings settings) {
      input.setText(TextFormat.printToString(settings.perfetto().getCustomConfig()));
    }

    @Override
    public void update(Settings settings) {
      try {
        TextFormat.merge(input.getText(), settings.writePerfetto()
            .getCustomConfigBuilder()
            .clear());
        settings.writePerfetto().setUseCustom(true);
      } catch (ParseException e) {
        // This shouldn't happen as we disable the OK button.
        LOG.log(WARNING, "Unexpected proto parse exception", e);
      }
    }
  }
}
