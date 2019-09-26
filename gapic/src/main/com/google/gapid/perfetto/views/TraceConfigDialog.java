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
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static java.util.stream.Collectors.toSet;

import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.GpuProfiling;
import com.google.gapid.util.Messages;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

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
import java.util.Set;

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
    StringBuilder sb = new StringBuilder();
    if (settings.perfettoCpu) {
      sb.append("CPU");
    }
    Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
    if (gpuCaps.getHasRenderStage() || gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0) {
      if (settings.perfettoGpu) {
        if (sb.length() > 0) {
          sb.append(", ");
        }
        sb.append("GPU");
      }
    }
    if (settings.perfettoMem) {
      if (sb.length() > 0) {
        sb.append(", ");
      }
      sb.append("Memory");
    }
    return sb.toString();
  }

  public static PerfettoConfig.TraceConfig.Builder getConfig(
      Settings settings, Device.PerfettoCapability caps) {
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

    Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
    if (gpuCaps.getHasRenderStage() || gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0) {
      if (settings.perfettoGpu) {
        if (settings.perfettoGpuSlices) {
          config.addDataSourcesBuilder()
              .getConfigBuilder()
                  .setName("gpu.timing");
        }
        if (settings.perfettoGpuCounters && settings.perfettoGpuCounterIds.length > 0) {
          PerfettoConfig.GpuCounterConfig.Builder counters = config.addDataSourcesBuilder()
              .getConfigBuilder()
                  .setName("gpu.counters")
                  .getGpuCounterConfigBuilder()
                      .setCounterPeriodNs(MILLISECONDS.toNanos(settings.perfettoGpuCounterRate));
          for (int counter : settings.perfettoGpuCounterIds) {
            counters.addCounterIds(counter);
          }
        }
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

    return config;
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

    private final Button mem;
    private final Label[] memLabels;
    private final Spinner memRate;

    public InputArea(
        Composite parent, Settings settings, Theme theme, Device.PerfettoCapability caps) {
      super(parent, SWT.NONE);
      setLayout(new GridLayout(1, false));

      cpu = createCheckbox(this, "CPU", settings.perfettoCpu, e -> updateCpu());
      Composite cpuGroup = withLayoutData(
          createComposite(this, withMargin(new GridLayout(1, false), 5, 0)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      cpuFreq = createCheckbox(cpuGroup, "Frequency and idle states", settings.perfettoCpuFreq);
      cpuChain = createCheckbox(cpuGroup, "Scheduling chains / latency", settings.perfettoCpuChain);
      cpuSlices = createCheckbox(cpuGroup, "Thread slices", settings.perfettoCpuSlices);
      addSeparator();

      Device.GPUProfiling gpuCaps = caps.getGpuProfiling();
      if (gpuCaps.getHasRenderStage() || gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0) {
        gpu = createCheckbox(this, "GPU", settings.perfettoGpu, e -> updateGpu());
        Composite gpuGroup = withLayoutData(
            createComposite(this, withMargin(new GridLayout(1, false), 5, 0)),
            withIndents(new GridData(), GROUP_INDENT, 0));
        if (gpuCaps.getHasRenderStage()) {
          gpuSlices = createCheckbox(gpuGroup, "Renderstage slices", settings.perfettoGpuSlices);
        } else {
          gpuSlices = null;
        }

        if (gpuCaps.getGpuCounterDescriptor().getSpecsCount() > 0) {
          gpuCounters = createCheckbox(
              gpuGroup, "Counters", settings.perfettoGpuCounters, e -> updateGpu());
          Composite counterGroup = withLayoutData(
              createComposite(gpuGroup, withMargin(new GridLayout(3, false), 5, 0)),
              withIndents(new GridData(), GROUP_INDENT, 0));
          gpuCountersLabels = new Label[3];
          gpuCountersLabels[1] = createLabel(counterGroup, "Poll Rate:");
          gpuCountersRate = createSpinner(counterGroup, settings.perfettoGpuCounterRate, 1, 1000);
          gpuCountersLabels[2] = createLabel(counterGroup, "ms");

          gpuCountersLabels[0] = createLabel(
              counterGroup, settings.perfettoGpuCounterIds.length + " selected");
          gpuCountersSelect = Widgets.createButton(counterGroup, "Select", e -> {
            GpuCountersDialog dialog =
                new GpuCountersDialog(getShell(), theme, caps, settings.perfettoGpuCounterIds);
            if (dialog.open() == Window.OK) {
              settings.perfettoGpuCounterIds = dialog.getSelectedIds();
              gpuCounters.setSelection(settings.perfettoGpuCounterIds.length != 0);
              gpuCountersLabels[0].setText(settings.perfettoGpuCounterIds.length + " selected");
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
        addSeparator();
      } else {
        gpu = null;
        gpuSlices = null;
        gpuCounters = null;
        gpuCountersLabels = null;
        gpuCountersRate = null;
        gpuCountersSelect = null;
      }

      mem = createCheckbox(this, "Memory", settings.perfettoMem, e -> updateMem());
      memLabels = new Label[2];
      Composite memGroup = withLayoutData(
          createComposite(this, withMargin(new GridLayout(3, false), 5, 0)),
          withIndents(new GridData(), GROUP_INDENT, 0));
      memLabels[0] = createLabel(memGroup, "Poll Rate:");
      memRate = createSpinner(memGroup, settings.perfettoMemRate, 1, 1000);
      memLabels[1] = createLabel(memGroup, "ms");

      updateCpu();
      updateGpu();
      updateMem();
    }

    public void update(Settings settings) {
      settings.perfettoCpu = cpu.getSelection();
      settings.perfettoCpuChain = cpuChain.getSelection();
      settings.perfettoCpuFreq = cpuFreq.getSelection();
      settings.perfettoCpuSlices = cpuSlices.getSelection();

      if (gpu != null) {
        settings.perfettoGpu = gpu.getSelection();
      }
      if (gpuSlices != null) {
        settings.perfettoGpuSlices = gpuSlices.getSelection();
      }
      if (gpuCounters != null) {
        settings.perfettoGpuCounters = gpuCounters.getSelection();
        settings.perfettoGpuCounterRate = gpuCountersRate.getSelection();
      }

      settings.perfettoMem = mem.getSelection();
      settings.perfettoMemRate = memRate.getSelection();
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
    }

    private void updateMem() {
      boolean enabled = mem.getSelection();
      memRate.setEnabled(enabled);
      for (Label label : memLabels) {
        label.setEnabled(enabled);
      }
    }

    private static class GpuCountersDialog extends DialogBase {
      private final Device.PerfettoCapability caps;
      private final Set<Integer> currentIds;

      private Table table;
      private int[] selectedIds;

      public GpuCountersDialog(
          Shell shell, Theme theme, Device.PerfettoCapability caps, int[] currentIds) {
        super(shell, theme);
        this.caps = caps;
        this.currentIds = Arrays.stream(currentIds).boxed().collect(toSet());
      }

      public int[] getSelectedIds() {
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
            .toArray();
        super.okPressed();
      }
    }
  }
}
