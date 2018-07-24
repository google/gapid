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

import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;

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
import com.google.gapid.proto.service.Service.DeviceAPITraceConfiguration;
import com.google.gapid.proto.service.Service.DeviceTraceConfiguration;
import com.google.gapid.proto.service.Service.StatusResponse;
import com.google.gapid.proto.service.Service.TraceTargetTreeNode;
import com.google.gapid.server.Client;
import com.google.gapid.server.Tracer;
import com.google.gapid.server.Tracer.TraceRequest;
import com.google.gapid.util.Messages;
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
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.DirectoryDialog;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Label;
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

/**
 * Dialogs used for capturing a trace.
 */
public class TracerDialog {
  private TracerDialog() {
  }

  public static void showOpenTraceDialog(Shell shell, Models models) {
    models.analytics.postInteraction(View.Main, ClientAction.Open);
    FileDialog dialog = new FileDialog(shell, SWT.OPEN);
    dialog.setFilterNames(new String[] { "Trace Files (*.gfxtrace)", "All Files" });
    dialog.setFilterExtensions(new String[] { "*.gfxtrace", "*" });
    dialog.setFilterPath(models.settings.lastOpenDir);
    String result = dialog.open();
    if (result != null) {
      models.capture.loadCapture(new File(result));
    }
  }

  public static void showSaveTraceDialog(Shell shell, Models models) {
    models.analytics.postInteraction(View.Main, ClientAction.Save);
    FileDialog dialog = new FileDialog(shell, SWT.SAVE);
    dialog.setFilterNames(new String[] { "Trace Files (*.gfxtrace)", "All Files" });
    dialog.setFilterExtensions(new String[] { "*.gfxtrace", "*" });
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
      progress.open();

      if (progress.successful()) {
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
      return Messages.CAPTURE_TRACE;
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
      private static final DateFormat TRACE_DATE_FORMAT = new SimpleDateFormat("_yyyyMMdd_HHmm");
      protected static final String MEC_LABEL = "Trace From Beginning";
      private static final String MEC_LABEL_WARNING =
          "Trace From Beginning (mid-execution capture for %s is experimental)";

      private final String date = TRACE_DATE_FORMAT.format(new Date());

      private List<DeviceCaptureInfo> devices;

      private ComboViewer device;
      private LoadingIndicator.Widget deviceLoader;
      private ComboViewer api;
      private ActionTextbox traceTarget;
      private Text arguments;
      private Text cwd;
      private Text envVars;
      private Spinner frameCount;
      private Button fromBeginning;
      private Button withoutBuffering;
      private Button hideUnknownExtensions;
      private Button clearCache;
      private Button disablePcs;
      private final FileTextbox.Directory directory;
      protected final Text file;

      private Label pcsWarning;
      protected String friendlyName = "";
      protected boolean userHasChangedOutputFile = false;

      public TraceInput(Composite parent, Models models, Widgets widgets, Runnable refreshDevices) {
        super(parent, SWT.NONE);
        setLayout(new GridLayout(2, false));

        createLabel(this, "Device:");
        Composite deviceComposite =
            createComposite(this, withMargin(new GridLayout(2, false), 0, 0));
        device = createDeviceDropDown(deviceComposite);
        deviceLoader = widgets.loading.createWidgetWithRefresh(deviceComposite);
        device.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
        deviceLoader.setLayoutData(
            withIndents(new GridData(SWT.RIGHT, SWT.CENTER, false, false), 5, 0));
        // TODO: Make this a true button to allow keyboard use.
        deviceLoader.addListener(SWT.MouseDown, e -> {
          deviceLoader.startLoading();
          // By waiting a tiny bit, the icon will change to the loading indicator, giving the user
          // feedback that something is happening, in case the refresh is really quick.
          Scheduler.EXECUTOR.schedule(refreshDevices, 300, TimeUnit.MILLISECONDS);
        });

        deviceComposite.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "API:");
        api = createApiDropDown(this);
        api.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "Application:");
        traceTarget = withLayoutData(new ActionTextbox(this, models.settings.traceUri) {
          @Override
          protected String createAndShowDialog(String current) {
            DeviceCaptureInfo dev = getSelectedDevice();
            if (dev != null) {
              TraceTargetPickerDialog dialog =
                  new TraceTargetPickerDialog(getShell(), models, dev.targets, widgets);
              if (dialog.open() == Window.OK) {
                TraceTargets.Node node = dialog.getSelected();
                if (node == null) {
                  return null;
                }
                TraceTargetTreeNode data = node.getData();
                if (data != null && !userHasChangedOutputFile) {
                  file.setText(formatTraceName(data.getFriendlyApplication()));
                  friendlyName = data.getFriendlyApplication();
                  userHasChangedOutputFile = false; // cancel the modify event from set call.
                }
                return (data == null) ? null : data.getTraceUri();
              }
            }
            return null;
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "Additional Arguments:");
        arguments = withLayoutData(createTextbox(this, models.settings.traceArguments),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "Working Directory:");
        cwd = withLayoutData(createTextbox(this, models.settings.traceCwd),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        cwd.setEnabled(false);

        createLabel(this, "Environment Variables:");
        envVars = withLayoutData(createTextbox(this, models.settings.traceEnv),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        envVars.setEnabled(false);

        createLabel(this, "Stop After:");
        Composite frameCountComposite =
            createComposite(this, withMargin(new GridLayout(2, false), 0, 0));
        frameCount = withLayoutData(
            createSpinner(frameCountComposite, models.settings.traceFrameCount, 0, 999999),
            new GridData(SWT.LEFT, SWT.FILL, false, false));
        createLabel(frameCountComposite, "Frames (0 for unlimited)");

        createLabel(this, "");
        fromBeginning = withLayoutData(
            createCheckbox(this, MEC_LABEL, !models.settings.traceMidExecution),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        fromBeginning.setEnabled(false);

        createLabel(this, "");
        withoutBuffering = withLayoutData(
            createCheckbox(this, "Disable Buffering", models.settings.traceWithoutBuffering),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "");
        clearCache = withLayoutData(
            createCheckbox(this, "Clear package cache", models.settings.traceClearCache),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        clearCache.setEnabled(false);

        createLabel(this, "");
        hideUnknownExtensions = withLayoutData(
            createCheckbox(this, "Hide Unknown Extensions", models.settings.traceHideUnknownExtensions),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "");
        disablePcs = withLayoutData(
            createCheckbox(this, "Disable pre-compiled shaders", models.settings.traceDisablePcs),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        disablePcs.setEnabled(false);

        createLabel(this, "Output Directory:");
        directory = withLayoutData(new FileTextbox.Directory(this, models.settings.traceOutDir) {
          @Override
          protected void configureDialog(DirectoryDialog dialog) {
            dialog.setText(Messages.CAPTURE_DIRECTORY);
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "File Name:");
        file = withLayoutData(
            createTextbox(this, ""), new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "");
        pcsWarning = withLayoutData(
            createLabel(this, "Warning: Pre-compiled shaders are not supported in the replay."),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        pcsWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_YELLOW));
        pcsWarning.setVisible(!models.settings.traceDisablePcs);

        device.getCombo().addListener(
            SWT.Selection, e -> update(models.settings, getSelectedDevice()));
        api.getCombo().addListener(SWT.Selection, e -> update(models.settings, getSelectedApi()));

        Listener mecListener = e -> {
          DeviceAPITraceConfiguration config = getSelectedApi();
          if (fromBeginning.getSelection() || config == null ||
              config.getMidExecutionCaptureSupport() != Service.FeatureStatus.Experimental) {
            fromBeginning.setText(MEC_LABEL);
          } else {
            fromBeginning.setText(String.format(MEC_LABEL_WARNING, config.getApi()));
          }
        };
        api.getCombo().addListener(SWT.Selection, mecListener);
        fromBeginning.addListener(SWT.Selection, mecListener);

        disablePcs.addListener(
            SWT.Selection, e -> pcsWarning.setVisible(!disablePcs.getSelection()));

        file.addListener(SWT.Modify, e -> {
          userHasChangedOutputFile = true;
          friendlyName = "";
        });
        if (!models.settings.traceFriendlyName.isEmpty()) {
          file.setText(formatTraceName(models.settings.traceFriendlyName));
        }

        updateDevicesDropDown(models.settings);
      }

      private static ComboViewer createDeviceDropDown(Composite parent) {
        ComboViewer combo = createDropDownViewer(parent);
        combo.setContentProvider(ArrayContentProvider.getInstance());
        combo.setLabelProvider(new LabelProvider() {
          @Override
          public String getText(Object element) {
            Device.Instance info = ((DeviceCaptureInfo)element).device;
            StringBuilder sb = new StringBuilder();
            if (!info.getName().isEmpty()) {
              sb.append(info.getName()).append(" - ");
            }
            if (!info.getConfiguration().getOS().getName().isEmpty()) {
              sb.append(info.getConfiguration().getOS().getName()).append(" - ");
            }
            return sb.append(info.getSerial()).toString();
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
            return ((DeviceAPITraceConfiguration)element).getApi();
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
      }

      private void update(Settings settings, DeviceAPITraceConfiguration config) {
        boolean pcs = config != null && config.getCanDisablePcs();
        disablePcs.setEnabled(pcs);
        disablePcs.setSelection(pcs && settings.traceDisablePcs);

        boolean mec = config != null &&
            config.getMidExecutionCaptureSupport() != Service.FeatureStatus.NotSupported;
        fromBeginning.setEnabled(mec);
        fromBeginning.setSelection(!mec || !settings.traceMidExecution);
      }

      private void updateDevicesDropDown(Settings settings) {
        if (device != null && devices != null) {
          deviceLoader.stopLoading();
          device.setInput(devices);
          if (!settings.traceDevice.isEmpty()) {
            Optional<DeviceCaptureInfo> deflt = devices.stream()
                .filter(dev -> settings.traceDevice.equals(dev.device.getSerial()))
                .findAny();
            if (deflt.isPresent()) {
              device.setSelection(new StructuredSelection(deflt.get()));
            }
          }
          device.getCombo().notifyListeners(SWT.Selection, new Event());
        } else if (deviceLoader != null) {
          deviceLoader.startLoading();
        }
      }

      private void updateApiDropdown(DeviceTraceConfiguration config, Settings settings) {
        if (api != null && config != null) {
          api.setInput(config.getApisList());
          DeviceAPITraceConfiguration deflt = config.getApis(0);
          for (DeviceAPITraceConfiguration c : config.getApisList()) {
            if (c.getApi().equals(settings.traceApi)) {
              deflt = c;
              break;
            }
          }
          api.setSelection(new StructuredSelection(deflt));
          api.getCombo().notifyListeners(SWT.Selection, new Event());
        }
      }

      protected String formatTraceName(String name) {
        return (name.isEmpty() ? DEFAULT_TRACE_FILE : name) + date + TRACE_EXTENSION;
      }

      public boolean isReady() {
        return getSelectedDevice() != null && getSelectedApi() != null &&
            !traceTarget.getText().isEmpty() && !directory.getText().isEmpty() &&
            !file.getText().isEmpty();
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
        DeviceAPITraceConfiguration config = getSelectedApi();
        File output = getOutputFile();

        settings.traceDevice = dev.device.getSerial();
        settings.traceApi = config.getApi();
        settings.traceUri = traceTarget.getText();
        settings.traceArguments = arguments.getText();
        settings.traceFrameCount = frameCount.getSelection();
        settings.traceWithoutBuffering = withoutBuffering.getSelection();
        settings.traceHideUnknownExtensions = hideUnknownExtensions.getSelection();
        settings.traceOutDir = directory.getText();
        settings.traceFriendlyName = friendlyName;

        Service.TraceOptions.Builder options = Service.TraceOptions.newBuilder()
            .setDevice(dev.path)
            .addApis(config.getApi())
            .setUri(traceTarget.getText())
            .setAdditionalCommandLineArgs(arguments.getText())
            .setFramesToCapture(frameCount.getSelection())
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
          settings.traceMidExecution = !fromBeginning.getSelection();
          options.setDeferStart(!fromBeginning.getSelection());
        }
        if (dev.config.getHasCache()) {
          settings.traceClearCache = clearCache.getSelection();
          options.setClearCache(clearCache.getSelection());
        }
        if (config.getCanDisablePcs()) {
          settings.traceDisablePcs = disablePcs.getSelection();
          options.setDisablePcs(disablePcs.getSelection());
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

      protected DeviceAPITraceConfiguration getSelectedApi() {
        IStructuredSelection sel = api.getStructuredSelection();
        return sel.isEmpty() ? null : ((DeviceAPITraceConfiguration)sel.getFirstElement());
      }

      private File getOutputFile() {
        String name = file.getText();
        if (name.isEmpty()) {
          name = formatTraceName(DEFAULT_TRACE_FILE);
        }
        String dir = directory.getText();
        return dir.isEmpty() ? new File(name) : new File(dir, name);
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
      return error == null;
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
      // UI not initialized yet.
      if (statusLabel == null) {
        return;
      }

      if (status != null) {
        statusLabel.setText(status.getStatus().name());
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

        if (status.getStatus() == Service.TraceStatus.Done) {
          Button button = getButton(IDialogConstants.OK_ID);
          if (button != null) {
            button.setText("Done");
          }
        }
      }
      if (error != null) {
        statusLabel.setText("Failed");
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
    }

    private String getErrorMessage() {
      // TODO: the server doesn't return nice errors yet.
      return (error == null) ? "" : error.getMessage();
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      createButton(parent, IDialogConstants.OK_ID, request.isMec() ? "Start" : "Stop", true);
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (IDialogConstants.OK_ID == buttonId) {
        Button button = getButton(buttonId);
        switch (button.getText()) {
          case "Start":
            button.setText("Stop");
            trace.start();
            break;
          case "Stop":
            button.setText("Done");
            trace.stop();
            break;
          default:
            super.buttonPressed(buttonId);
        }
      }
    }
  }
}
