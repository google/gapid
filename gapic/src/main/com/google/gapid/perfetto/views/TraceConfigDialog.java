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

import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;

import com.google.gapid.models.Devices.DeviceCaptureInfo;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;

import java.util.Arrays;
import java.util.ArrayList;

import perfetto.protos.PerfettoConfig;

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
      "core_services", "dalvik", "database", "disk", "freq", "gfx", "hal", "i2c", "idle", "input",
      "ion", "irq", "memory", "memreclaim", "network", "nnapi", "pagecache", "pdx", "pm", "power",
      "regulators", "res", "rro", "rs", "sched", "sm", "ss", "sync", "vibrator", "video", "view",
      "webview", "wm", "workq"
  };
  private static final PerfettoConfig.MeminfoCounters[] MEM_COUNTERS = {
      PerfettoConfig.MeminfoCounters.MEMINFO_MEM_TOTAL,
      PerfettoConfig.MeminfoCounters.MEMINFO_MEM_FREE,
      PerfettoConfig.MeminfoCounters.MEMINFO_BUFFERS,
      PerfettoConfig.MeminfoCounters.MEMINFO_CACHED,
      PerfettoConfig.MeminfoCounters.MEMINFO_SWAP_CACHED,
  };

  private final Settings settings;
  private final DeviceCaptureInfo device;
  private InputArea input;

  public TraceConfigDialog(DeviceCaptureInfo device, Shell shell, Settings settings, Theme theme) {
    super(shell, theme);
    this.settings = settings;
    this.device = device;
  }

  public static void showPerfettoConfigDialog(DeviceCaptureInfo device, Shell shell, Models models, Widgets widgets) {
    new TraceConfigDialog(device, shell, models.settings, widgets.theme).open();
  }

  public static String getConfigSummary(Settings settings) {
    StringBuilder sb = new StringBuilder();
    if (settings.perfettoCpu) {
      sb.append("CPU");
    }
    if (settings.perfettoMem) {
      if (sb.length() > 0) {
        sb.append(", ");
      }
      sb.append("Memory");
    }
    return sb.toString();
  }

  public static PerfettoConfig.TraceConfig.Builder getConfig(Settings settings) {
    PerfettoConfig.TraceConfig.Builder config = PerfettoConfig.TraceConfig.newBuilder()
        .addBuffers(PerfettoConfig.TraceConfig.BufferConfig.newBuilder()
            .setSizeKb(BUFFER_SIZE));

    if (settings.perfettoCpu) {
      // Record process names.
      config.addDataSourcesBuilder()
          .getConfigBuilder()
              .setName("linux.process_stats")
              .getProcessStatsConfigBuilder()
                  .setScanAllProcessesOnStart(true);

      PerfettoConfig.FtraceConfig.Builder ftrace = config.addDataSourcesBuilder()
          .getConfigBuilder()
              .setName("linux.ftrace")
              .getFtraceConfigBuilder()
                  .setBufferSizeKb(FTRACE_BUFFER_SIZE)
                  .addAllFtraceEvents(Arrays.asList(CPU_BASE_FTRACE));
      if (settings.perfettoCpuFreq) {
        ftrace.addAllFtraceEvents(Arrays.asList(CPU_FREQ_FTRACE));
      }
      if (settings.perfettoCpuChain) {
        ftrace.addAllFtraceEvents(Arrays.asList(CPU_CHAIN_FTRACE));
      }
      if (settings.perfettoCpuSlices) {
        ftrace.addAllAtraceCategories(Arrays.asList(CPU_SLICES_ATRACE));
      }
    }

    if (settings.perfettoMem) {
      config.addDataSourcesBuilder()
          .getConfigBuilder()
              .setName("linux.sys_stats")
              .getSysStatsConfigBuilder()
                  .setMeminfoPeriodMs(settings.perfettoMemRate)
                  .addAllMeminfoCounters(Arrays.asList(MEM_COUNTERS));
    }

    for (int i = 0; i < settings.perfettoVulkanLayers.length; i++) {
      config.addDataSourcesBuilder()
      .getConfigBuilder()
          .setName(settings.perfettoVulkanLayers[i]);
    }

    return config;
  }

  @Override
  public String getTitle() {
    return Messages.CAPTURE_TRACE_PERFETTO;
  }

  @Override
  protected Control createDialogArea(Composite parent) {
    Composite area = (Composite)super.createDialogArea(parent);
    input = withLayoutData(new InputArea(device, area, settings), new GridData(GridData.FILL_BOTH));
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

    private final Button mem;
    private final Label[] memLabels;
    private final Spinner memRate;
    private final Button[] vulkanChecks;
    public InputArea(DeviceCaptureInfo device, Composite parent, Settings settings) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(1, false));

      cpu = createCheckbox(this, "CPU", settings.perfettoCpu, e -> updateCpu());
      Composite cpuGroup = withLayoutData(createComposite(this, new GridLayout(1, false)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      cpuFreq = createCheckbox(cpuGroup, "Frequency and idle states", settings.perfettoCpuFreq);
      cpuChain = createCheckbox(cpuGroup, "Scheduling chains / latency", settings.perfettoCpuChain);
      cpuSlices = createCheckbox(cpuGroup, "Thread slices", settings.perfettoCpuSlices);
      addSeparator();

      mem = createCheckbox(this, "Memory", settings.perfettoMem, e -> updateMem());
      memLabels = new Label[2];
      Composite memGroup = withLayoutData(createComposite(this, new GridLayout(3, false)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      memLabels[0] = createLabel(memGroup, "Poll Rate:");
      memRate = createSpinner(memGroup, settings.perfettoMemRate, 1, 1000);
      memLabels[1] = createLabel(memGroup, "ms");

      addSeparator();
      Composite vulkanGroup = withLayoutData(createComposite(this, new GridLayout(1, false)),
      withIndents(new GridData(), GROUP_INDENT, 0));
      int numLayers = device.device.getConfiguration().getPerfettoCapability().getVulkanProfileLayersCount();
      vulkanChecks = new Button[numLayers];
      for (int i = 0; i < numLayers; i++) {
        String name = device.device.getConfiguration().getPerfettoCapability().getVulkanProfileLayers(i).getProbeName();
        boolean enabled = Arrays.stream(settings.perfettoVulkanLayers).anyMatch(name::equals);
        vulkanChecks[i] = createCheckbox(this, name, enabled, e -> updateVulkan());
      }
      updateCpu();
      updateMem();
      updateVulkan();
    }

    public void update(Settings settings) {
      settings.perfettoCpu = cpu.getSelection();
      settings.perfettoCpuChain = cpuChain.getSelection();
      settings.perfettoCpuFreq = cpuFreq.getSelection();
      settings.perfettoCpuSlices = cpuSlices.getSelection();

      settings.perfettoMem = mem.getSelection();
      settings.perfettoMemRate = memRate.getSelection();
      ArrayList<String> layers = new ArrayList<String>();
      for (int i = 0; i < vulkanChecks.length; i++) {
        if (vulkanChecks[i].getSelection()) {
          layers.add(vulkanChecks[i].getText());
        }
      }
      settings.perfettoVulkanLayers = new String[layers.size()];
      layers.toArray(settings.perfettoVulkanLayers);
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

    private void updateVulkan() {
    }

    private void updateMem() {
      boolean enabled = mem.getSelection();
      memRate.setEnabled(enabled);
      for (Label label : memLabels) {
        label.setEnabled(enabled);
      }
    }
  }
}
