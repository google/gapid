/*
 * Copyright (C) 2017 Google Inc.
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
package com.google.gapid.views;

import static com.google.gapid.perfetto.views.TraceConfigDialog.getConfig;
import static com.google.gapid.perfetto.views.TraceConfigDialog.getConfigSummary;
import static com.google.gapid.perfetto.views.TraceConfigDialog.showPerfettoConfigDialog;
import static com.google.gapid.util.MoreFutures.logFailure;
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;
import static java.util.concurrent.TimeUnit.MINUTES;

import com.google.common.collect.Lists;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Devices.DeviceCaptureInfo;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.models.TraceTargets;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.Service.DeviceTraceConfiguration;
import com.google.gapid.proto.service.Service.StatusResponse;
import com.google.gapid.proto.service.Service.TraceType;
import com.google.gapid.proto.service.Service.TraceTypeCapabilities;
import com.google.gapid.server.Client;
import com.google.gapid.server.Tracer;
import com.google.gapid.server.Tracer.TraceRequest;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.util.Scheduler;
import com.google.gapid.widgets.ActionTextbox;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.FileTextbox;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Theme;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Combo;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.DirectoryDialog;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.Text;

import java.io.File;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.Collections;
import java.util.Date;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

/**
 * Dialogs used for capturing a trace.
 */
public class TracerDialog {
  protected static final Logger LOG = Logger.getLogger(TracerDialog.class.getName());

  private TracerDialog() {
  }

  public static void showOpenTraceDialog(Shell shell, Models models) {
    models.analytics.postInteraction(View.Main, ClientAction.Open);
    FileDialog dialog = new FileDialog(shell, SWT.OPEN);
    dialog.setFilterNames(new String[] {
        "Trace Files (*.gfxtrace, *.perfetto)",
        "Graphics Traces (*.gfxtrace)",
        "System Profile (*.perfetto)",
        "All Files"
    });
    dialog.setFilterExtensions(new String[] {
        "*.gfxtrace;*.perfetto",
        "*.gfxtrace",
        "*.perfetto",
        "*"
    });
    dialog.setFilterPath(models.settings.lastOpenDir);
    String result = dialog.open();
    if (result != null) {
      models.capture.loadCapture(new File(result));
    }
  }

  public static void showSaveTraceDialog(Shell shell, Models models) {
    models.analytics.postInteraction(View.Main, ClientAction.Save);
    boolean isPerfetto = models.capture.isPerfetto();
    FileDialog dialog = new FileDialog(shell, SWT.SAVE);
    dialog.setFilterNames(new String[] {
        "Trace Files (" + (isPerfetto ? "*.perfetto" : "*.gfxtrace") + ")", "All Files"
    });
    dialog.setFilterExtensions(new String[] { isPerfetto ? "*.perfetto" : "*.gfxtrace", "*" });
    dialog.setFilterPath(models.settings.lastOpenDir);
    String result = dialog.open();
    if (result != null) {
      models.capture.saveCapture(new File(result));
    }
  }

  public static void showTracingDialog(Client client, Shell shell, Models models, Widgets widgets) {
    models.analytics.postInteraction(View.Trace, ClientAction.Show);
    TraceInputDialog input =
        new TraceInputDialog(shell, models, widgets, models.devices::loadDevices);
    if (loadDevicesAndShowDialog(input, models) == Window.OK) {
      TraceProgressDialog progress = new TraceProgressDialog(
          shell, models.analytics, input.getValue(), widgets.theme);
      Tracer.Trace trace = Tracer.trace(client, shell, input.getValue(), progress);
      progress.setTrace(trace);
      if (progress.open() == Window.OK && progress.successful()) {
        models.capture.loadCapture(input.getValue().output);
      }
    }
  }

  private static int loadDevicesAndShowDialog(TraceInputDialog dialog, Models models) {
    Devices.Listener listener = new Devices.Listener() {
      @Override
      public void onCaptureDevicesLoaded() {
        dialog.setDevices(models.devices.getCaptureDevices());
      }
    };
    models.devices.addListener(listener);
    try {
      models.devices.loadDevices();
      return dialog.open();
    } finally {
      models.devices.removeListener(listener);
    }
  }

  /**
   * Dialog to request the information from the user to start a trace (which app, filename, etc.).
   */
  private static class TraceInputDialog extends DialogBase {
    private final Models models;
    private final Widgets widgets;
    private final Runnable refreshDevices;

    private TraceInput traceInput;
    private List<DeviceCaptureInfo> devices;

    private Tracer.TraceRequest value;

    public TraceInputDialog(Shell shell, Models models, Widgets widgets, Runnable refreshDevices) {
      super(shell, widgets.theme);
      this.models = models;
      this.widgets = widgets;
      this.refreshDevices = refreshDevices;
    }

    public void setDevices(List<DeviceCaptureInfo> devices) {
      this.devices = devices;
      traceInput.setDevices(models.settings, devices);
    }

    public Tracer.TraceRequest getValue() {
      return value;
    }

    @Override
    public String getTitle() {
      return Messages.CAPTURE_TRACE_GRAPHICS;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);
      traceInput = new TraceInput(area, models, widgets, refreshDevices);
      traceInput.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      if (devices != null) {
        traceInput.setDevices(models.settings, devices);
      }
      return area;
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      Button ok = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);

      Listener modifyListener = e -> {
        ok.setEnabled(traceInput.isReady());
      };
      traceInput.addModifyListener(modifyListener);

      modifyListener.handleEvent(null); // Set initial state of widgets.
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (buttonId == IDialogConstants.OK_ID) {
        value = traceInput.getTraceRequest(models.settings);
      }
      super.buttonPressed(buttonId);
    }

    private static class TraceInput extends Composite {
      private static final String DEFAULT_TRACE_FILE = "trace";
      private static final String TRACE_EXTENSION = ".gfxtrace";
      private static final String PERFETTO_EXTENSION = ".perfetto";
      private static final DateFormat TRACE_DATE_FORMAT = new SimpleDateFormat("_yyyyMMdd_HHmm");
      private static final String TARGET_LABEL = "Application";
      private static final String FRAMES_LABEL = "Stop After:";
      private static final String DURATION_LABEL = "Duration:";
      private static final String FRAMES_UNIT = "Frames (0 for manual)";
      private static final String DURATION_UNIT = "Seconds (0 for manual)";
      private static final String MEC_LABEL_WARNING =
          "NOTE: Mid-Execution capture for %s is experimental";
      private static final int DEFAULT_START_FRAME = 100;
      private static final String PERFETTO_LABEL = "Profile Config: ";

      private final String date = TRACE_DATE_FORMAT.format(new Date());

      private List<DeviceCaptureInfo> devices;

      private final ComboViewer device;
      private final Label deviceLabel;
      private final LoadingIndicator.Widget deviceLoader;
      private final ComboViewer api;
      private final Label apiLabel;
      private final ActionTextbox traceTarget;
      private final Label targetLabel;
      private final Text arguments;
      private final Text cwd;
      private final Text envVars;
      private final Combo startType;
      private final Spinner startFrame;
      private final Label durationLabel;
      private final Spinner duration;
      private final Label durationUnit;
      private Label mecWarningLabel;
      private final Button withoutBuffering;
      private final Button hideUnknownExtensions;
      private final Button clearCache;
      private final Button disablePcs;
      private final Composite perfettoConfig;
      private final Label perfettoConfigLabel;
      private final FileTextbox.Directory directory;
      private final Label directoryLabel;
      protected final Text file;
      private final Label fileLabel;
      private final Label pcsWarning;
      private final Label requiredFieldMessage;

      protected String friendlyName = "";
      protected boolean userHasChangedOutputFile = false;
      protected boolean userHasChangedTarget = false;

      public TraceInput(Composite parent, Models models, Widgets widgets, Runnable refreshDevices) {
        super(parent, SWT.NONE);
        this.friendlyName = models.settings.traceFriendlyName;

        setLayout(new GridLayout(1, false));

        Group mainGroup = withLayoutData(
            createGroup(this, "Device and Type", new GridLayout(3, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        deviceLabel = createLabel(mainGroup, "Device*:");
        device = createDeviceDropDown(mainGroup);
        deviceLoader = widgets.loading.createWidgetWithRefresh(mainGroup);
        device.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
        deviceLoader.setLayoutData(
            withIndents(new GridData(SWT.RIGHT, SWT.CENTER, false, false), 5, 0));
        // TODO: Make this a true button to allow keyboard use.
        deviceLoader.addListener(SWT.MouseDown, e -> {
          deviceLoader.startLoading();
          // By waiting a tiny bit, the icon will change to the loading indicator, giving the user
          // feedback that something is happening, in case the refresh is really quick.
          logFailure(LOG, Scheduler.EXECUTOR.schedule(refreshDevices, 300, TimeUnit.MILLISECONDS));
        });

        apiLabel = createLabel(mainGroup, "Type*:");
        api = createApiDropDown(mainGroup);
        api.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

        Group appGroup  = withLayoutData(
            createGroup(this, "Application", new GridLayout(2, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        targetLabel = createLabel(appGroup, TARGET_LABEL + ":");
        traceTarget = withLayoutData(new ActionTextbox(appGroup, models.settings.traceUri) {
          @Override
          protected String createAndShowDialog(String current) {
            DeviceCaptureInfo dev = getSelectedDevice();
            if (dev != null) {
              TraceTargets.Target target =
                  showTraceTargetPicker(dev, current, getShell(), models, widgets);
              if (target != null) {
                friendlyName = target.friendlyName;
                if (!userHasChangedOutputFile) {
                  file.setText(formatTraceName(friendlyName));
                  userHasChangedOutputFile = false; // cancel the modify event from set call.
                }

                // Setting the text ourselves so we cancel the event.
                setText(target.url);
                userHasChangedTarget = false; // cancel the modify event from set call.
              }
            }
            return null;
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(appGroup, "Additional Arguments:");
        arguments = withLayoutData(createTextbox(appGroup, models.settings.traceArguments),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(appGroup, "Working Directory:");
        cwd = withLayoutData(createTextbox(appGroup, models.settings.traceCwd),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        cwd.setEnabled(false);

        createLabel(appGroup, "Environment Variables:");
        envVars = withLayoutData(createTextbox(appGroup, models.settings.traceEnv),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        envVars.setEnabled(false);

        Group durGroup = withLayoutData(
            createGroup(this, "Start and Duration", new GridLayout(4, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        createLabel(durGroup, "Start at:");
        startType = Widgets.createDropDown(durGroup);
        startType.setItems("Beginning", "Manual", "Frame");
        startFrame = withLayoutData(
            createSpinner(durGroup, Math.max(1, models.settings.traceStartAt), 1, 999999),
            new GridData(SWT.FILL, SWT.TOP, false, false));
        mecWarningLabel = createLabel(durGroup, "");

        if (models.settings.traceStartAt < 0) {
          startType.select(1);
          startFrame.setSelection(DEFAULT_START_FRAME);
        } else if (models.settings.traceStartAt > 0) {
          startType.select(2);
        } else {
          startType.select(0);
          startFrame.setSelection(DEFAULT_START_FRAME);
        }

        durationLabel = createLabel(durGroup, FRAMES_LABEL);
        duration = withLayoutData(
            createSpinner(durGroup, models.settings.traceDuration, 0, 999999),
            new GridData(SWT.FILL, SWT.TOP, false, false));
        durationUnit = createLabel(durGroup, FRAMES_UNIT);

        Group optGroup  = withLayoutData(
            createGroup(this, "Trace Options", new GridLayout(2, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        withoutBuffering = createCheckbox(
            optGroup, "Disable Buffering", models.settings.traceWithoutBuffering);
        withoutBuffering.setEnabled(true);
        disablePcs = createCheckbox(
            optGroup, "Disable pre-compiled shaders", models.settings.traceDisablePcs);
        disablePcs.setEnabled(false);
        clearCache = createCheckbox(
            optGroup, "Clear package cache", models.settings.traceClearCache);
        clearCache.setEnabled(false);
        hideUnknownExtensions = createCheckbox(
            optGroup, "Hide Unknown Extensions", models.settings.traceHideUnknownExtensions);
        hideUnknownExtensions.setEnabled(false);

        perfettoConfig = withLayoutData(
            createComposite(optGroup, withMargin(new GridLayout(2, false), 5, 0)),
            withSpans(new GridData(GridData.FILL_HORIZONTAL), 2, 1));
        perfettoConfigLabel = createLabel(
            perfettoConfig, PERFETTO_LABEL + getConfigSummary(models.settings, getPerfettoCaps()));
        Widgets.createButton(perfettoConfig, "Configure", e -> {
          showPerfettoConfigDialog(getShell(), models, widgets, getPerfettoCaps());
          updatePerfettoConfigLabel(models.settings);
        });
        perfettoConfig.setVisible(false);

        Group outGroup = withLayoutData(
            createGroup(this, "Output", new GridLayout(2, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        directoryLabel = createLabel(outGroup, "Output Directory*:");
        directory = withLayoutData(new FileTextbox.Directory(outGroup, models.settings.traceOutDir) {
          @Override
          protected void configureDialog(DirectoryDialog dialog) {
            dialog.setText(Messages.CAPTURE_DIRECTORY);
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        fileLabel = createLabel(outGroup, "File Name*:");
        file = withLayoutData(createTextbox(outGroup, formatTraceName(friendlyName)),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        pcsWarning = withLayoutData(
            createLabel(this, "Warning: Pre-compiled shaders are not supported in the replay."),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        pcsWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_YELLOW));
        pcsWarning.setVisible(!models.settings.traceDisablePcs);

        requiredFieldMessage = withLayoutData(
            createLabel(this, "Please fill out required information (labeled with *)."),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        requiredFieldMessage.setForeground(getDisplay().getSystemColor(SWT.COLOR_RED));
        requiredFieldMessage.setVisible(false);

        Link adbWarning = withLayoutData(
            createLink(this, "Path to adb invalid/missing. " +
                "To trace on Android, please fix it in the <a>preferences</a>.",
                e -> SettingsDialog.showSettingsDialog(getShell(), models, widgets.theme)),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        adbWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));
        adbWarning.setVisible(!models.settings.isAdbValid());

        device.getCombo().addListener(
            SWT.Selection, e -> update(models.settings, getSelectedDevice()));
        api.getCombo().addListener(SWT.Selection, e -> update(models.settings, getSelectedApi()));

        Listener mecListener = e -> {
          TraceTypeCapabilities config = getSelectedApi();
          boolean beginning = startType.getSelectionIndex() == 0;
          if (!beginning && config != null &&
              config.getMidExecutionCaptureSupport() == Service.FeatureStatus.Experimental) {
            mecWarningLabel.setText(String.format(MEC_LABEL_WARNING, config.getApi()));
          } else {
            mecWarningLabel.setText("");
          }
          mecWarningLabel.requestLayout();

          boolean startAtFrame = startType.getSelectionIndex() == 2;
          startFrame.setVisible(startAtFrame);
        };
        api.getCombo().addListener(SWT.Selection, mecListener);
        startType.addListener(SWT.Selection, mecListener);

        disablePcs.addListener(
            SWT.Selection, e -> pcsWarning.setVisible(!disablePcs.getSelection()));

        traceTarget.addBoxListener(SWT.Modify, e -> {
          userHasChangedTarget = true;
        });
        file.addListener(SWT.Modify, e -> {
          userHasChangedOutputFile = true;
          if (userHasChangedTarget) {
            // The user has both manually changed the trace target and the output file.
            // Clearly the suggested friendly name is no good.
            friendlyName = "";
          }
        });

        addModifyListener(e -> colorFilledInput(widgets.theme));

        updateDevicesDropDown(models.settings);
        colorFilledInput(widgets.theme);
      }

      private void colorFilledInput(Theme theme) {
        if (devices == null) {
          // Don't mark anything red, until the devices are loaded.
          return;
        }

        requiredFieldMessage.setVisible(!this.isReady());
        deviceLabel.setForeground(getSelectedDevice() == null ? theme.missingInput() : theme.filledInput());
        directoryLabel.setForeground(directory.getText().isEmpty() ? theme.missingInput() : theme.filledInput());
        fileLabel.setForeground(file.getText().isEmpty() ? theme.missingInput() : theme.filledInput());

        TraceTypeCapabilities config = getSelectedApi();
        if (config != null) {
          apiLabel.setForeground(theme.filledInput());
          targetLabel.setForeground(
              (config.getRequiresApplication() && traceTarget.getText().isEmpty()) ?
                theme.missingInput() : theme.filledInput());
        } else {
          apiLabel.setForeground(theme.missingInput());
          targetLabel.setForeground(theme.filledInput());
        }
      }

      private static ComboViewer createDeviceDropDown(Composite parent) {
        ComboViewer combo = createDropDownViewer(parent);
        combo.setContentProvider(ArrayContentProvider.getInstance());
        combo.setLabelProvider(new LabelProvider() {
          @Override
          public String getText(Object element) {
            return Devices.getLabel(((DeviceCaptureInfo)element).device);
          }
        });
        return combo;
      }

      private static ComboViewer createApiDropDown(Composite parent) {
        ComboViewer combo = createDropDownViewer(parent);
        combo.setContentProvider(ArrayContentProvider.getInstance());
        combo.setLabelProvider(new LabelProvider() {
          @Override
          public String getText(Object element) {
            TraceTypeCapabilities ttc = (TraceTypeCapabilities)element;
            switch (ttc.getType()) {
              case Graphics: return ttc.getApi();
              case Perfetto: return "System Profile";
              default: throw new AssertionError();
            }
          }
        });
        return combo;
      }

      private void update(Settings settings, DeviceCaptureInfo dev) {
        DeviceTraceConfiguration config = (dev == null) ? null : dev.config;
        traceTarget.setActionEnabled(dev != null);
        cwd.setEnabled(config != null && config.getCanSpecifyCwd());
        envVars.setEnabled(config != null && config.getCanSpecifyEnv());
        clearCache.setEnabled(config != null && config.getHasCache());
        updateApiDropdown(config, settings);
        updatePerfettoConfigLabel(settings);
      }

      private void update(Settings settings, TraceTypeCapabilities config) {
        boolean pcs = config != null && config.getCanDisablePcs();
        disablePcs.setEnabled(pcs);
        disablePcs.setSelection(pcs && settings.traceDisablePcs);
        pcsWarning.setVisible(pcs && !settings.traceDisablePcs);

        boolean ext = config != null && config.getCanEnableUnsupportedExtensions();
        hideUnknownExtensions.setEnabled(ext);
        hideUnknownExtensions.setSelection(!ext || settings.traceHideUnknownExtensions);

        boolean appRequired = config != null && config.getRequiresApplication();
        targetLabel.setText(TARGET_LABEL + (appRequired ? "*:" : ":"));
        targetLabel.requestLayout();

        boolean isPerfetto = isPerfetto(config);
        getShell().setText(
            isPerfetto ? Messages.CAPTURE_TRACE_PERFETTO : Messages.CAPTURE_TRACE_GRAPHICS);
        withoutBuffering.setEnabled(!isPerfetto);
        withoutBuffering.setSelection(!isPerfetto && settings.traceWithoutBuffering);
        if (isPerfetto && startType.getItemCount() == 3) {
          if (startType.getSelectionIndex() == 2) {
            // Switch to manual if it was "start at frame x".
            startType.select(1);
          }
          startType.remove(2);
        } else if (!isPerfetto && startType.getItemCount() == 2) {
          startType.add("Frame");
        }
        durationLabel.setText(isPerfetto ? DURATION_LABEL : FRAMES_LABEL);
        durationUnit.setText(isPerfetto ? DURATION_UNIT : FRAMES_UNIT);
        durationUnit.requestLayout();
        perfettoConfig.setVisible(isPerfetto);

        if (!userHasChangedOutputFile) {
          file.setText(formatTraceName(friendlyName));
          userHasChangedOutputFile = false; // cancel the modify event from set call.
        }
      }

      private void updateDevicesDropDown(Settings settings) {
        if (device != null && devices != null) {
          deviceLoader.stopLoading();
          device.setInput(devices);
          DeviceCaptureInfo deflt = getPreviouslySelectedDevice(settings).orElseGet(() -> {
            if (devices.size() == 1) {
              return devices.get(0);
            } else if (devices.size() == 2) {
              // If there are exactly two devices and exactly one of them is an Android device,
              // select the Android device. It is a fair assumption that a developer that has an
              // Android device connected wants to trace Android, not desktop.
              boolean firstIsAndroid = devices.get(0).isAndroid();
              boolean secondIsAndroid = devices.get(1).isAndroid();
              if (firstIsAndroid != secondIsAndroid) {
                return firstIsAndroid ? devices.get(0) : devices.get(1);
              }
            }
            return null;
          });
          if (deflt != null) {
            device.setSelection(new StructuredSelection(deflt));
          }
          device.getCombo().notifyListeners(SWT.Selection, new Event());
        } else if (deviceLoader != null) {
          deviceLoader.startLoading();
        }
      }

      private Optional<DeviceCaptureInfo> getPreviouslySelectedDevice(Settings settings) {
        return (settings.traceDevice.isEmpty()) ? Optional.empty() : devices.stream()
            .filter(dev -> settings.traceDevice.equals(dev.device.getSerial()))
            .findAny();
      }

      private void updateApiDropdown(DeviceTraceConfiguration config, Settings settings) {
        if (api != null && config != null) {
          List<TraceTypeCapabilities> caps = config.getApisList();
          api.setInput(config.getApisList());
          if (!caps.isEmpty()) {
            TraceTypeCapabilities deflt = caps.get(0);
            for (TraceTypeCapabilities c : caps) {
              if (c.getType().name().equals(settings.traceType) &&
                  (c.getApi().isEmpty() || c.getApi().equals(settings.traceApi))) {
                deflt = c;
                break;
              }
            }
            api.setSelection(new StructuredSelection(deflt));
            api.getCombo().notifyListeners(SWT.Selection, new Event());
          }
        }
      }

      private void updatePerfettoConfigLabel(Settings settings) {
        perfettoConfigLabel.setText(
            PERFETTO_LABEL + getConfigSummary(settings, getPerfettoCaps()));
        perfettoConfigLabel.requestLayout();
      }

      protected static TraceTargets.Target showTraceTargetPicker(
          DeviceCaptureInfo dev, String current, Shell shell, Models models, Widgets widgets) {
        if (dev.config.getServerLocalPath()) {
          // We are local, show a system file browser dialog.
          FileDialog dialog = new FileDialog(shell);
          if (!setDialogPath(dialog, current) &&
              !setDialogPath(dialog, dev.config.getPreferredRootUri()) &&
              !setDialogPath(dialog, OS.userHomeDir)) {
            dialog.setFilterPath(OS.cwd);
          }
          dialog.setText(Messages.CAPTURE_EXECUTABLE);
          String exe = dialog.open();
          if (exe != null) {
            return new TraceTargets.Target(exe, new File(exe).getName());
          }
        } else {
          // Use the server to query the trace target tree.
          TraceTargetPickerDialog dialog =
              new TraceTargetPickerDialog(shell, models, dev.targets, widgets);
          if (dialog.open() == Window.OK) {
            TraceTargets.Node node = dialog.getSelected();
            return (node == null) ? null : node.getTraceTarget();
          }
        }
        return null;
      }

      private static boolean setDialogPath(FileDialog dialog, String path) {
        if (path == null || path.isEmpty()) {
          return false;
        }

        File file = new File(path).getAbsoluteFile();
        if (!file.exists()) {
          return false;
        }
        if (file.isDirectory()) {
          dialog.setFilterPath(file.getPath());
        } else {
          dialog.setFilterPath(file.getParent());
          dialog.setFileName(file.getName());
        }
        return true;
      }

      protected String formatTraceName(String name) {
        TraceTypeCapabilities config = getSelectedApi();
        String ext = config != null && isPerfetto(config) ? PERFETTO_EXTENSION : TRACE_EXTENSION;
        return (name.isEmpty() ? DEFAULT_TRACE_FILE : name) + date + ext;
      }

      public boolean isReady() {
        TraceTypeCapabilities config = getSelectedApi();
        return getSelectedDevice() != null && config != null &&
            (!config.getRequiresApplication() || !traceTarget.getText().isEmpty()) &&
            !directory.getText().isEmpty() && !file.getText().isEmpty();
      }

      public void addModifyListener(Listener listener) {
        device.getCombo().addListener(SWT.Selection, listener);
        api.getCombo().addListener(SWT.Selection, listener);
        traceTarget.addBoxListener(SWT.Modify, listener);
        directory.addBoxListener(SWT.Modify, listener);
        file.addListener(SWT.Modify, listener);
      }

      public void setDevices(Settings settings, List<DeviceCaptureInfo> devices) {
        this.devices = devices;
        updateDevicesDropDown(settings);
      }

      public TraceRequest getTraceRequest(Settings settings) {
        DeviceCaptureInfo dev = devices.get(device.getCombo().getSelectionIndex());
        TraceTypeCapabilities config = getSelectedApi();
        File output = getOutputFile();

        settings.traceDevice = dev.device.getSerial();
        settings.traceType = config.getType().name();
        settings.traceApi = config.getApi();
        settings.traceUri = traceTarget.getText();
        settings.traceArguments = arguments.getText();
        switch (startType.getSelectionIndex()) {
          case 0: // Beginning
            settings.traceStartAt = 0;
            break;
          case 1: // Manaul
            settings.traceStartAt = -1;
            break;
          default: // Frame
            settings.traceStartAt = startFrame.getSelection();
        }
        settings.traceDuration = duration.getSelection();
        settings.traceWithoutBuffering = withoutBuffering.getSelection();
        settings.traceHideUnknownExtensions = hideUnknownExtensions.getSelection();
        settings.traceOutDir = directory.getText();
        settings.traceFriendlyName = friendlyName;

        Service.TraceOptions.Builder options = Service.TraceOptions.newBuilder()
            .setDevice(dev.path)
            .setType(config.getType())
            .addApis(config.getApi())
            .setUri(traceTarget.getText())
            .setAdditionalCommandLineArgs(arguments.getText())
            .setFramesToCapture(duration.getSelection())
            .setNoBuffer(withoutBuffering.getSelection())
            .setHideUnknownExtensions(hideUnknownExtensions.getSelection())
            .setServerLocalSavePath(output.getAbsolutePath());

        if (dev.config.getCanSpecifyCwd()) {
          settings.traceCwd = cwd.getText();
          options.setCwd(cwd.getText());
        }
        if (dev.config.getCanSpecifyEnv()) {
          settings.traceEnv = envVars.getText();
          options.addAllEnvironment(splitEnv(envVars.getText()));
        }
        if (config.getMidExecutionCaptureSupport() != Service.FeatureStatus.NotSupported) {
          options.setDeferStart(settings.traceStartAt != 0);
          if (settings.traceStartAt > 0) {
            options.setStartFrame(settings.traceStartAt);
          }
        }
        if (dev.config.getHasCache()) {
          settings.traceClearCache = clearCache.getSelection();
          options.setClearCache(clearCache.getSelection());
        }
        if (config.getCanDisablePcs()) {
          settings.traceDisablePcs = disablePcs.getSelection();
          options.setDisablePcs(disablePcs.getSelection());
        }

        if (isPerfetto(config)) {
          int durationMs = duration.getSelection() * 1000;
          // TODO: this isn't really unlimitted.
          durationMs = (durationMs == 0) ? (int)MINUTES.toMillis(10) : durationMs;
          options.setPerfettoConfig(getConfig(settings, getPerfettoCaps())
              .setDurationMs(durationMs));
        }

        return new TraceRequest(output, options.build());
      }

      private static List<String> splitEnv(String env) {
        if ((env = env.trim()).isEmpty()) {
          return Collections.emptyList();
        }

        List<String> result = Lists.newArrayList();
        boolean inQuote = false, foundEq = false;
        int start = 0;
        for (int i = 0; i < env.length(); i++) {
          switch (env.charAt(i)) {
            case ' ':
              if (!inQuote && foundEq) {
                result.add(env.substring(start, i));
                start = i;
                foundEq = false;
              }
              break;
            case '"':
              inQuote = !inQuote;
              break;
            case '=':
              foundEq = true;
              break;
          }
        }
        result.add(env.substring(start));
        return result;
      }

      protected DeviceCaptureInfo getSelectedDevice() {
        IStructuredSelection sel = device.getStructuredSelection();
        return sel.isEmpty() ? null : (DeviceCaptureInfo)sel.getFirstElement();
      }

      protected TraceTypeCapabilities getSelectedApi() {
        IStructuredSelection sel = api.getStructuredSelection();
        return sel.isEmpty() ? null : ((TraceTypeCapabilities)sel.getFirstElement());
      }

      protected Device.PerfettoCapability getPerfettoCaps() {
        DeviceCaptureInfo dev = getSelectedDevice();
        return (dev == null) ? Device.PerfettoCapability.getDefaultInstance() :
            dev.device.getConfiguration().getPerfettoCapability();
      }

      private File getOutputFile() {
        String name = file.getText();
        if (name.isEmpty()) {
          name = formatTraceName(DEFAULT_TRACE_FILE);
        }
        String dir = directory.getText();
        return dir.isEmpty() ? new File(name) : new File(dir, name);
      }

      private static boolean isPerfetto(TraceTypeCapabilities config) {
        return config.getType() == TraceType.Perfetto;
      }
    }
  }

  /**
   * Dialog that shows trace progress to the user and allows the user to stop the capture.
   */
  private static class TraceProgressDialog extends DialogBase implements Tracer.Listener {
    private static final int STATUS_INTERVAL_MS = 1000;

    private final Analytics analytics;
    private final Tracer.TraceRequest request;
    private Label statusLabel;
    private Label bytesLabel;
    private Text errorText;
    private Button errorButton;

    private Tracer.Trace trace;

    private StatusResponse status;
    private Throwable error;

    public TraceProgressDialog(
        Shell shell, Analytics analytics, Tracer.TraceRequest request, Theme theme) {
      super(shell, theme);
      this.analytics = analytics;
      this.request = request;
    }

    public void setTrace(Tracer.Trace trace) {
      this.trace = trace;
    }

    public boolean successful() {
      return error == null && status != null && status.getStatus() == Service.TraceStatus.Done;
    }

    @Override
    public void onProgress(StatusResponse progress) {
      status = progress;
      update();
    }

    @Override
    public void onFailure(Throwable e) {
      error = e;
      update();
    }

    @Override
    public String getTitle() {
      return Messages.CAPTURING_TRACE;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = createComposite(area, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      createBoldLabel(container, request.getProgressDialogTitle())
          .setLayoutData(withSpans(new GridData(SWT.FILL, SWT.TOP, true, false), 2, 1));

      createLabel(container, "Status:");
      statusLabel = withLayoutData(createLabel(container, "Submitting request"),
          new GridData(SWT.FILL, SWT.TOP, true, false));

      createLabel(container, "Captured so far:");
      bytesLabel = withLayoutData(createLabel(container, "0 Bytes"),
          new GridData(SWT.FILL, SWT.TOP, true, false));

      errorText = withLayoutData(createTextbox(container, SWT.WRAP | SWT.READ_ONLY, ""),
          withIndents(withSpans(new GridData(SWT.FILL, SWT.TOP, true, false), 2, 1), 0, 10));
      errorText.setBackground(container.getBackground());
      errorText.setVisible(false);

      errorButton = Widgets.createButton(container, "Details", e ->
          ErrorDialog.showErrorDialog(getShell(), analytics, getErrorMessage(), error));
      errorButton.setVisible(false);

      update();

      Widgets.scheduleUntilDisposed(getShell(), STATUS_INTERVAL_MS, trace::getStatus);
      return area;
    }

    private void update() {
      // UI not initialized yet or already disposed.
      if (statusLabel == null || statusLabel.isDisposed()) {
        return;
      }

      if (status != null) {
        statusLabel.setText(getStatusLabel(status.getStatus()));
        long bytes = status.getBytesCaptured();
        if (bytes < 1024 + 512) {
          bytesLabel.setText(bytes + " Bytes");
        } else if (bytes < (1024 + 512) * 1024) {
          bytesLabel.setText(String.format("%.1f KBytes", bytes / 1024.0));
        } else if (bytes < (1024 + 512) * 1024 * 1024) {
          bytesLabel.setText(String.format("%.2f MBytes", bytes / 1024.0 / 1024.0));
        } else {
          bytesLabel.setText(String.format("%.2f GBytes", bytes / 1024.0 / 1024.0 / 1024.0));
        }
      }
      if (error != null) {
        statusLabel.setText("Failed.");
        errorText.setText(getErrorMessage());
        errorText.setVisible(true);
        errorButton.setVisible(true);

        errorText.requestLayout();
        Point curr = getShell().getSize();
        Point want = getShell().computeSize(SWT.DEFAULT, SWT.DEFAULT);
        if (want.y > curr.y) {
          getShell().setSize(new Point(curr.x, want.y));
        }
      }

      updateButton();
    }

    private static String getStatusLabel(Service.TraceStatus status) {
      switch (status) {
        case Uninitialized:
          return "Sending request...";
        case Initializing:
          return "Initializing...";
        case WaitingToStart:
          return "Press 'Start' to begin capture";
        case Capturing:
          return "Capturing...";
        case Done:
          return "Done.";
        default:
          return "Unknown.";
      }
    }

    private void updateButton() {
      Button button = getButton(IDialogConstants.OK_ID);
      if (button == null) {
        return;
      }
      button.setEnabled(true);

      if (error != null) {
        button.setText("Close");
      } else if (status == null) {
        button.setText("Cancel");
      } else {
        switch (status.getStatus()) {
          case WaitingToStart:
            button.setText("Start");
            break;
          case Capturing:
            button.setText("Stop");
            break;
          case Done:
            button.setText("Open Trace");
            break;
          default:
            button.setText("Cancel");
            break;
        }
      }
    }

    private String getErrorMessage() {
      // TODO: the server doesn't return nice errors yet.
      return (error == null) ? "" : error.getMessage();
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      createButton(parent, IDialogConstants.OK_ID, "Cancel", true);
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (IDialogConstants.OK_ID == buttonId) {
        Button button = getButton(buttonId);
        if (error != null || status == null) {
          cancelPressed();
        } else {
          switch (status.getStatus()) {
            case WaitingToStart:
              button.setEnabled(false);
              trace.start();
              break;
            case Capturing:
              button.setEnabled(false);
              trace.stop();
              break;
            case Done:
              okPressed();
              break;
            default:
              cancelPressed();
          }
        }
      }
    }
  }
}
