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

import static com.google.gapid.perfetto.views.FuchsiaTraceConfigDialog.showFuchsiaConfigDialog;
import static com.google.gapid.perfetto.views.TraceConfigDialog.getConfig;
import static com.google.gapid.perfetto.views.TraceConfigDialog.getConfigSummary;
import static com.google.gapid.perfetto.views.TraceConfigDialog.showPerfettoConfigDialog;
import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.MoreFutures.logFailure;
import static com.google.gapid.widgets.Widgets.createBoldLabel;
import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createProgressBar;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withMarginOnly;
import static com.google.gapid.widgets.Widgets.withSpans;
import static java.util.concurrent.TimeUnit.MILLISECONDS;
import static java.util.concurrent.TimeUnit.MINUTES;
import static java.util.concurrent.TimeUnit.SECONDS;
import static java.util.logging.Level.WARNING;
import static java.util.stream.Collectors.toList;

import com.google.common.collect.Lists;
import com.google.gapid.models.Analytics;
import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Devices.DeviceCaptureInfo;
import com.google.gapid.models.Devices.DeviceValidationResult;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.models.TraceTargets;
import com.google.gapid.perfetto.views.FuchsiaTraceConfigDialog;
import com.google.gapid.proto.SettingsProto;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.ClientAction;
import com.google.gapid.proto.service.Service.DeviceTraceConfiguration;
import com.google.gapid.proto.service.Service.StatusResponse;
import com.google.gapid.proto.service.Service.TraceTypeCapabilities;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.server.Client;
import com.google.gapid.server.Tracer;
import com.google.gapid.server.Tracer.TraceRequest;
import com.google.gapid.util.CenteringStackLayout;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.util.Scheduler;
import com.google.gapid.util.URLs;
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
import org.eclipse.swt.custom.StackLayout;
import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.layout.RowLayout;
import org.eclipse.swt.program.Program;
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
import org.eclipse.swt.widgets.ProgressBar;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.Text;

import java.io.File;
import java.io.FileOutputStream;
import java.net.URL;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.Arrays;
import java.util.Collections;
import java.util.Date;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.logging.Level;
import java.util.logging.Logger;

/**
 * Dialogs used for capturing a trace.
 */
public class TracerDialog {
  protected static final Logger LOG = Logger.getLogger(TracerDialog.class.getName());

  public static final Flag<Integer> maxFrames = Flags.value(
      "max-frames", 1, "The maximum number of frames to allow for graphics captures", true);
  public static final Flag<Integer> maxPerfetto = Flags.value(
      "max-perfetto", 10 * 60, "The maximum amount of time to allow for profile captures", true);
  public static final Flag<Boolean> enableLoadValidationLayer = Flags.value(
      "load-validation-layer", false,
      "Show the option to load the Vulkan validation layer at capture time.");

  private TracerDialog() {
  }

  public static void showOpenTraceDialog(Shell shell, Models models) {
    models.analytics.postInteraction(View.Main, ClientAction.Open);
    FileDialog dialog = new FileDialog(shell, SWT.OPEN);
    dialog.setFilterNames(new String[] {
        "Trace Files (*.gfxtrace, *.perfetto, *.fxt)",
        "Graphics Traces (*.gfxtrace)",
        "System Profile (*.perfetto, *.fxt)",
        "All Files"
    });
    dialog.setFilterExtensions(new String[] {
        "*.gfxtrace;*.perfetto;*.fxt",
        "*.gfxtrace",
        "*.perfetto;*.fxt",
        "*"
    });
    dialog.setFilterPath(models.settings.files().getLastOpenDir());
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
    dialog.setFilterPath(models.settings.files().getLastOpenDir());
    String result = dialog.open();
    if (result != null) {
      models.capture.saveCapture(new File(result));
    }
  }

  public static void showSystemTracingDialog(
      Client client, Shell shell, Models models, Widgets widgets) {
    showTracingDialog(TraceType.System, client, shell, models, widgets);
  }

  public static void showFrameTracingDialog(
      Client client, Shell shell, Models models, Widgets widgets) {
    showTracingDialog(TraceType.Vulkan, client, shell, models, widgets);
  }

  // Shows the tracing dialog of the previously shown type (or Vulkan by default).
  public static void showTracingDialog(
      Client client, Shell shell, Models models, Widgets widgets) {
    showTracingDialog(null, client, shell, models, widgets);
  }

  private static void showTracingDialog(
      TraceType type, Client client, Shell shell, Models models, Widgets widgets) {
    models.analytics.postInteraction(View.Trace, ClientAction.Show);
    TraceInputDialog input =
        new TraceInputDialog(shell, type, models, widgets, models.devices::loadDevices);
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
    private final TraceType type;
    private final Models models;
    private final Widgets widgets;
    private final Runnable refreshDevices;

    private TraceInput traceInput;
    private List<DeviceCaptureInfo> devices;

    private Tracer.TraceRequest value;

    public TraceInputDialog(Shell shell, TraceType type, Models models, Widgets widgets,
        Runnable refreshDevices) {
      super(shell, widgets.theme);
      this.type = type;
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
      return Messages.CAPTURE_TRACE_DEFAULT;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);
      traceInput = new TraceInput(area, type, models, widgets, refreshDevices);
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
        if (LOG.isLoggable(Level.FINE)) {
          LOG.log(Level.FINE, "Trace request: {0}", value.options);
        }
      }
      super.buttonPressed(buttonId);
    }

    private static class TraceInput extends Composite {
      private static final String DEFAULT_TRACE_FILE = "trace";
      private static final String ANGLE_STRING = "_angle";
      private static final DateFormat TRACE_DATE_FORMAT = new SimpleDateFormat("_yyyyMMdd_HHmm");
      private static final String TARGET_LABEL = "Application";
      private static final String FRAMES_LABEL = "Stop After:";
      private static final String ONE_FRAME_LABEL = "Duration: 1 Frame";
      private static final String DURATION_LABEL = "Duration:";
      private static final int DURATION_FRAMES_MAX = maxFrames.get();
      private static final int DURATION_PERFETTO_MAX = maxPerfetto.get();
      private static final String DURATION_FRAMES_UNIT = "Frames";
      private static final String DURATION_PERFETTO_UNIT = "Seconds";
      private static final String START_AT_TIME_UNIT = "Seconds";
      private static final String PERFETTO_LABEL = "Profile Config: ";
      private static final String FUCHSIA_LABEL = "Fuchsia Trace Config: ";
      // Quote 'GPU activity' in the warning message to match the config panel tickbox title.
      private static final String EMPTY_APP_WITH_RENDER_STAGE =
          "Warning: cannot record 'GPU activity' when no application is selected";

      private final String date = TRACE_DATE_FORMAT.format(new Date());

      private List<DeviceCaptureInfo> devices;
      private final SingleInFlight rpcController = new SingleInFlight();
      private final Models models;

      private final ComboViewer device;
      private final Label deviceLabel;
      private final StackLayout deviceLayout;
      private final Label deviceWarning;
      private final LoadingIndicator.Widget deviceLoader;
      private final ComboViewer api;
      private final Label apiLabel;
      private final DeviceValidationView deviceValidationView;
      private final ActionTextbox traceTarget;
      private final Label targetLabel;
      private final Text arguments;
      private final Text cwd;
      private final Text processName;
      private final Text envVars;
      private final Combo startType;
      private final Spinner startFrame;
      private final Label durationLabel;
      private final Spinner duration;
      private final Label durationUnit;
      private final Label startUnit;
      private final Button withoutBuffering;
      private final Button includeUnsupportedExtensions;
      private final Button loadValidationLayer;
      private final Button clearCache;
      private final Composite systemTracingConfig;
      private final Label systemTracingConfigLabel;
      private final FileTextbox.Directory directory;
      private final Label directoryLabel;
      protected final Text file;
      private final Label fileLabel;
      private final Label requiredFieldMessage;
      private final Label emptyAppWarning;
      private final Composite angleBar;
      private final Label angleMessage;
      private final Button installAngle;

      protected String friendlyName = "";
      protected boolean userHasChangedOutputFile = false;
      protected boolean userHasChangedTarget = false;

      public TraceInput(Composite parent, TraceType type, Models models, Widgets widgets,
          Runnable refreshDevices) {
        super(parent, SWT.NONE);
        this.models = models;
        SettingsProto.TraceOrBuilder trace = models.settings.trace();

        this.friendlyName = trace.getFriendlyName();

        setLayout(new GridLayout(1, false));

        Service.TraceType lastType = Service.TraceType.Graphics;
        try {
          lastType = Service.TraceType.valueOf(trace.getType());
        } catch (IllegalArgumentException e) {
          // The serialized name was invalid, ignore it and use Graphics as the default.
        }

        if (type == null) {
          // Use the last used type if no specific type was requested.
          type = TraceType.from(lastType);
        } else if (type == TraceType.Vulkan && lastType == Service.TraceType.ANGLE) {
          // If frame profiler was requested, use ANGLE if the previous trace was an ANGLE trace.
          type = TraceType.ANGLE;
        }

        Group mainGroup = withLayoutData(
            createGroup(this, "Device and Type", new GridLayout(3, false)),
            new GridData(GridData.FILL_HORIZONTAL));

        apiLabel = createLabel(mainGroup, "Type*:");
        api = createApiDropDown(mainGroup, type);
        api.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
        // Placeholder label to fill the 3rd column
        createLabel(mainGroup, "");

        deviceLabel = createLabel(mainGroup, "Device*:");
        deviceLayout = new CenteringStackLayout();
        Composite deviceContainer = withLayoutData(createComposite(mainGroup, deviceLayout),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        device = createDeviceDropDown(deviceContainer);
        deviceLayout.topControl = device.getControl();
        deviceWarning = createLabel(
            deviceContainer, "No device connected that supports this trace type.");
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

        createLabel(mainGroup, "Validation:");
        deviceValidationView = new DeviceValidationView(mainGroup, this.models, widgets);

        Group appGroup  = withLayoutData(
            createGroup(this, "Application", new GridLayout(2, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        targetLabel = createLabel(appGroup, TARGET_LABEL + ":");
        traceTarget = withLayoutData(new ActionTextbox(appGroup, trace.getUri()) {
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
        arguments = withLayoutData(createTextbox(appGroup, trace.getArguments()),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(appGroup, "Process Name:");
        processName = withLayoutData(createTextbox(appGroup, trace.getProcessName()),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        processName.setEnabled(false);

        createLabel(appGroup, "Working Directory:");
        cwd = withLayoutData(createTextbox(appGroup, trace.getCwd()),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        cwd.setEnabled(false);

        createLabel(appGroup, "Environment Variables:");
        envVars = withLayoutData(createTextbox(appGroup, trace.getEnv()),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        envVars.setEnabled(false);

        Group durGroup = withLayoutData(
            createGroup(this, "Start and Duration", new GridLayout(4, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        createLabel(durGroup, "Start at:");
        startType = Widgets.createDropDown(durGroup);
        startType.setItems(Arrays.stream(StartType.values()).map(Enum::name).toArray(String[]::new));
        startType.select(StartType.Manual.ordinal());
        startFrame = withLayoutData(createSpinner(durGroup, 100, 1, 999999),
            new GridData(SWT.FILL, SWT.TOP, false, false));
        startFrame.setVisible(false);
        startUnit = createLabel(durGroup, START_AT_TIME_UNIT);
        startUnit.setVisible(false);

        durationLabel = createLabel(durGroup, DURATION_FRAMES_MAX == 1 ? ONE_FRAME_LABEL : FRAMES_LABEL);
        duration = withLayoutData(createSpinner(durGroup, 1, 1, DURATION_FRAMES_MAX),
            new GridData(SWT.FILL, SWT.TOP, false, false));
        duration.setVisible(DURATION_FRAMES_MAX > 1);
        durationUnit = createLabel(durGroup, DURATION_FRAMES_UNIT);
        durationUnit.setVisible(DURATION_FRAMES_MAX > 1);

        Group optGroup  = withLayoutData(
            createGroup(this, "Trace Options", new GridLayout(2, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        withoutBuffering = createCheckbox(
            optGroup, "Disable Buffering", trace.getWithoutBuffering());
        withoutBuffering.setEnabled(true);
        clearCache = createCheckbox(
            optGroup, "Clear Package Cache", trace.getClearCache());
        clearCache.setEnabled(false);
        includeUnsupportedExtensions = createCheckbox(optGroup, "Include Unsupported Extensions", false);
        includeUnsupportedExtensions.setEnabled(false);
        loadValidationLayer = createCheckbox(
            optGroup, "Load Vulkan Validation Layer", trace.getLoadValidationLayer());
        loadValidationLayer.setEnabled(false);
        loadValidationLayer.setVisible(enableLoadValidationLayer.get());

        systemTracingConfig = withLayoutData(
            createComposite(optGroup, withMargin(new GridLayout(2, false), 5, 0)),
            withSpans(new GridData(GridData.FILL_HORIZONTAL), 2, 1));
        systemTracingConfigLabel = createLabel(systemTracingConfig, PERFETTO_LABEL);
        Widgets.createButton(systemTracingConfig, "Configure", e -> {
          DeviceCaptureInfo dev = getSelectedDevice();
          if (dev == null) {
            // Ignore.
          } else if (dev.isFuchsia()) {
            showFuchsiaConfigDialog(getShell(), models, widgets);
          } else {
            showPerfettoConfigDialog(getShell(), models, widgets, dev);
          }

          if (!isDisposed()) {
            updateSystemTracingConfigLabel(models.settings, dev);
            updateEmptyAppWithRenderStageWarning(models.settings);
          }
        });
        systemTracingConfig.setVisible(false);

        Group outGroup = withLayoutData(
            createGroup(this, "Output", new GridLayout(2, false)),
            new GridData(GridData.FILL_HORIZONTAL));
        directoryLabel = createLabel(outGroup, "Output Directory*:");
        directory = withLayoutData(new FileTextbox.Directory(outGroup, trace.getOutDir()) {
          @Override
          protected void configureDialog(DirectoryDialog dialog) {
            dialog.setText(Messages.CAPTURE_DIRECTORY);
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        fileLabel = createLabel(outGroup, "File Name*:");
        file = withLayoutData(createTextbox(outGroup, formatTraceName(friendlyName)),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        requiredFieldMessage = withLayoutData(
            createLabel(this, "Please fill out required information (labeled with *)."),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        requiredFieldMessage.setForeground(getDisplay().getSystemColor(SWT.COLOR_RED));
        requiredFieldMessage.setVisible(false);

        if (!models.settings.isAdbValid()) {
          Link adbWarning = withLayoutData(
              createLink(this, "Path to adb invalid/missing. " +
                  "To trace on Android, please fix it in the <a>preferences</a>.",
                  e -> SettingsDialog.showSettingsDialog(getShell(), models, widgets.theme)),
              new GridData(SWT.FILL, SWT.FILL, true, false));
          adbWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));
        }

        emptyAppWarning = withLayoutData(
            createLabel(this, EMPTY_APP_WITH_RENDER_STAGE),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        emptyAppWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_YELLOW));
        emptyAppWarning.setVisible(false);
        device.getCombo().addListener(SWT.Selection, e -> updateEmptyAppWithRenderStageWarning(models.settings));
        api.getCombo().addListener(SWT.Selection, e -> updateEmptyAppWithRenderStageWarning(models.settings));
        traceTarget.addBoxListener(SWT.Modify, e -> updateEmptyAppWithRenderStageWarning(models.settings));

        RowLayout angleBarLayout = new RowLayout();
        angleBarLayout.center = true;
        angleBarLayout.spacing = 10;
        angleBar = withLayoutData(
            createComposite(this, angleBarLayout),
            new GridData(SWT.RIGHT, SWT.FILL, true, false));
        angleMessage = createLabel(angleBar, "");
        installAngle = Widgets.createButton(angleBar, "",
            e -> downloadAndInstallAngle(widgets.theme, models.settings));
        angleBar.setVisible(false);

        device.getCombo().addListener(SWT.Selection, e -> {
          updateOnDeviceChange(models.settings, getSelectedDevice());
          deviceValidationView.ValidateDevice(getSelectedDevice());
        });
        api.getCombo().addListener(SWT.Selection, e -> {
          updateOnApiChange(models.settings, trace, getSelectedType());
        });

        Listener mecListener = e -> {
          StartType start = StartType.values()[startType.getSelectionIndex()];
          startFrame.setVisible(start == StartType.Frame || start == StartType.Time);
          startUnit.setVisible(start == StartType.Time);
        };
        api.getCombo().addListener(SWT.Selection, mecListener);
        startType.addListener(SWT.Selection, mecListener);

        traceTarget.addBoxListener(SWT.Modify, e -> {
          userHasChangedTarget = true;
          if (traceTarget.getText().isEmpty() && !userHasChangedOutputFile) {
            // The user may clear the target name do a system profiling with no app in particular,
            // in this case avoid to keep an old app-based output file name.
            file.setText(formatTraceName(DEFAULT_TRACE_FILE));
            userHasChangedOutputFile = false; // cancel the modify event from set call.
          }
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

        updateDevicesDropDown(trace);
        colorFilledInput(widgets.theme);
      }

      private void colorFilledInput(Theme theme) {
        if (devices == null) {
          // Don't mark anything red, until the devices are loaded.
          return;
        }

        TraceType type = getSelectedType();
        DeviceCaptureInfo dev = getSelectedDevice();

        if (type == null) {
          apiLabel.setForeground(theme.missingInput());
          deviceLabel.setForeground(theme.missingInput());
          targetLabel.setForeground(theme.filledInput());
        } else {
          apiLabel.setForeground(theme.filledInput());

          Service.TraceTypeCapabilities config = (dev == null) ? null : type.getCapabilities(dev);
          if (config == null) {
            deviceLabel.setForeground(theme.missingInput());
            targetLabel.setForeground(theme.filledInput());
          } else {
            deviceLabel.setForeground(theme.filledInput());
            targetLabel.setForeground(
                (config.getRequiresApplication() && traceTarget.getText().isEmpty()) ?
                    theme.missingInput() : theme.filledInput());
          }
        }

        requiredFieldMessage.setVisible(!isInputReady(type, dev));
        directoryLabel.setForeground(
            directory.getText().isEmpty() ? theme.missingInput() : theme.filledInput());
        fileLabel.setForeground(
            file.getText().isEmpty() ? theme.missingInput() : theme.filledInput());
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

      private static ComboViewer createApiDropDown(Composite parent, TraceType deflt) {
        ComboViewer combo = createDropDownViewer(parent);
        combo.setContentProvider(ArrayContentProvider.getInstance());
        combo.setInput(TraceType.values());
        combo.setSelection(new StructuredSelection(deflt));
        return combo;
      }

      private void updateOnApiChange(
          Settings settings, SettingsProto.TraceOrBuilder trace, TraceType type) {
        updateDevicesDropDown(trace);
        updateOnConfigChange(settings, trace, type, getSelectedDevice());
      }

      private void updateOnDeviceChange(Settings settings, DeviceCaptureInfo dev) {
        DeviceTraceConfiguration config = (dev == null) ? null : dev.config;
        traceTarget.setActionEnabled(dev != null);
        cwd.setEnabled(config != null && config.getCanSpecifyCwd());
        envVars.setEnabled(config != null && config.getCanSpecifyEnv());
        clearCache.setEnabled(config != null && config.getHasCache());
        updateSystemTracingConfigLabel(settings, dev);

        if (dev != null) {
          updateOnConfigChange(settings, settings.trace(), getSelectedType(), dev);
        }
      }

      private void updateOnConfigChange(Settings settings, SettingsProto.TraceOrBuilder trace,
          TraceType type, DeviceCaptureInfo dev) {
        if (type == null || dev == null) {
          return;
        }

        Service.TraceTypeCapabilities config = type.getCapabilities(dev);

        boolean ext = config != null && config.getCanEnableUnsupportedExtensions();
        includeUnsupportedExtensions.setEnabled(ext);

        boolean appRequired = config != null && config.getRequiresApplication();
        targetLabel.setText(TARGET_LABEL + (appRequired ? "*:" : ":"));
        targetLabel.requestLayout();

        boolean canSelectProcessName = config != null && config.getCanSelectProcessName();
        processName.setEnabled(canSelectProcessName);

        boolean isSystem = type == TraceType.System;
        SettingsProto.Trace.DurationOrBuilder dur = isSystem ?
            trace.getProfileDurationOrBuilder() : trace.getGfxDurationOrBuilder();
        withoutBuffering.setEnabled(!isSystem);
        withoutBuffering.setSelection(!isSystem && trace.getWithoutBuffering());
        loadValidationLayer.setEnabled(!isSystem && getSelectedDevice().isAndroid());
        loadValidationLayer.setSelection(trace.getLoadValidationLayer());
        if (isSystem && startType.getItemCount() == 4) {
          startType.remove(StartType.Frame.ordinal());
        } else if (!isSystem && startType.getItemCount() == 3) {
          startType.add(StartType.Frame.name(), StartType.Frame.ordinal());
        }
        switch (dur.getType()) {
          case BEGINNING:
            startType.select(StartType.Beginning.ordinal());
            break;
          case FRAME:
            startType.select(StartType.Frame.ordinal());
            startFrame.setSelection(dur.getStartFrame());
            break;
          case TIME:
            startType.select(StartType.Time.ordinal());
            startFrame.setSelection(dur.getStartTime());
            break;
          case MANUAL:
          default:
            startType.select(StartType.Manual.ordinal());
        }

        int maxDuration = isSystem ? DURATION_PERFETTO_MAX : DURATION_FRAMES_MAX;
        durationLabel.setText(isSystem ? DURATION_LABEL : (maxDuration == 1 ? ONE_FRAME_LABEL : FRAMES_LABEL));
        duration.setMaximum(maxDuration);
        duration.setSelection(Math.min(dur.getDuration(), maxDuration));
        duration.setVisible(maxDuration > 1);
        durationUnit.setText(isSystem ? DURATION_PERFETTO_UNIT : DURATION_FRAMES_UNIT);
        durationUnit.setVisible(maxDuration > 1);
        durationUnit.requestLayout();

        systemTracingConfig.setVisible(isSystem);

        if (!userHasChangedOutputFile) {
          file.setText(formatTraceName(friendlyName));
          userHasChangedOutputFile = false; // cancel the modify event from set call.
        }

        boolean showAngleBar = false;
        if (type == TraceType.ANGLE) {
          int version = dev.device.getConfiguration().getAngle().getVersion();
          if (version == 0 || version < settings.preferences().getLatestAngleRelease().getVersion())
          {
            if (version == 0) {
              angleMessage.setText("ANGLE is required for this trace type (AGI restart required):");
              installAngle.setText("Install ANGLE");
            } else {
              angleMessage.setText("A recommended update to ANGLE is available (AGI restart required):");
              installAngle.setText("Update ANGLE");
            }
            angleMessage.requestLayout();
            installAngle.requestLayout();
            showAngleBar = true;
          }
        }
        angleBar.setVisible(showAngleBar);
      }

      private void updateDevicesDropDown(SettingsProto.TraceOrBuilder trace) {
        if (device != null && devices != null) {
          deviceLoader.stopLoading();

          TraceType type = getSelectedType();
          if (type == null) {
            return;
          }

          List<DeviceCaptureInfo> matching = devices.stream()
              .filter(d -> type.getCapabilities(d) != null)
              .collect(toList());
          device.setInput(matching);

          if (matching.isEmpty()) {
            deviceLayout.topControl = deviceWarning;
            deviceWarning.requestLayout();
          } else {
            deviceLayout.topControl = device.getControl();
            device.getControl().requestLayout();
          }

          DeviceCaptureInfo deflt = getPreviouslySelectedDevice(trace).orElseGet(() -> {
            if (matching.size() == 1) {
              return matching.get(0);
            } else if (matching.size() == 2) {
              // If there are exactly two devices and exactly one of them is an Android device,
              // select the Android device. It is a fair assumption that a developer that has an
              // Android device connected wants to trace Android, not desktop.
              boolean firstIsAndroid = matching.get(0).isAndroid();
              boolean secondIsAndroid = matching.get(1).isAndroid();
              if (firstIsAndroid != secondIsAndroid) {
                return firstIsAndroid ? matching.get(0) : matching.get(1);
              }
              // Same for Fuchsia.
              boolean firstIsFuchsia = matching.get(0).isFuchsia();
              boolean secondIsFuchsia = matching.get(1).isFuchsia();
              if (firstIsFuchsia != secondIsFuchsia) {
                return firstIsFuchsia ? matching.get(0) : matching.get(1);
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

      private Optional<DeviceCaptureInfo> getPreviouslySelectedDevice(
          SettingsProto.TraceOrBuilder trace) {
        if (!trace.getDeviceSerial().isEmpty()) {
          return devices.stream()
              .filter(dev -> trace.getDeviceSerial().equals(dev.device.getSerial()))
              .findAny();
        } else if (!trace.getDeviceName().isEmpty()) {
          return devices.stream()
              .filter(dev -> "".equals(dev.device.getSerial()) &&
                  trace.getDeviceName().equals(dev.device.getName()))
              .findAny();
        } else {
          return Optional.empty();
        }
      }

      private void updateSystemTracingConfigLabel(Settings settings, DeviceCaptureInfo dev) {
        if (dev == null) {
          systemTracingConfigLabel.setText("");
        } else if (dev.isFuchsia()) {
          systemTracingConfigLabel.setText(
              FUCHSIA_LABEL + FuchsiaTraceConfigDialog.getConfigSummary(settings));
        } else {
          systemTracingConfigLabel.setText(
              PERFETTO_LABEL + getConfigSummary(settings, dev));
        }
        systemTracingConfigLabel.requestLayout();
      }

      private void updateEmptyAppWithRenderStageWarning(Settings settings) {
        if (getSelectedDevice() == null || !getSelectedDevice().isAndroid() ||
            getSelectedType() != TraceType.System) {
          emptyAppWarning.setVisible(false);
          return;
        }
        SettingsProto.Perfetto.GPUOrBuilder gpu = settings.perfetto().getGpuOrBuilder();
        if (traceTarget.getText().isEmpty() && gpu.getEnabled() && gpu.getSlices()) {
          emptyAppWarning.setVisible(true);
        } else  {
          emptyAppWarning.setVisible(false);
        }
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
        TraceType type = getSelectedType();
        DeviceCaptureInfo dev = getSelectedDevice();
        String ext = type.getFileExtension(dev);
        // For ANGLE captures include "_angle" in trace name
        String angle = (type == TraceType.ANGLE) ? ANGLE_STRING : "";
        return (name.isEmpty() ? DEFAULT_TRACE_FILE : name) + angle + date + ext;
      }

      private void downloadAndInstallAngle(Theme theme, Settings settings) {
        DeviceCaptureInfo dev = getSelectedDevice();
        Service.Releases.ANGLERelease release = settings.preferences().getLatestAngleRelease();
        String url = getAngleAPKUrl(dev, release);
        if (url == null || !url.startsWith(URLs.EXPECTED_ANGLE_PREFIX)) {
          // Something went wrong, bring the user to our download landing page.
          Program.launch(URLs.ANGLE_DOWNLOAD);
          return;
        }

        if (InstallAngleDialog
            .showDialogAndInstallApk(getShell(), models, theme, dev.device, url) == Window.OK) {
          Composite comp = this;
          while (comp.getParent() != null) {
            comp = comp.getParent();
          }
          comp.getShell().close();
        }
      }

      private static String getAngleAPKUrl(
          DeviceCaptureInfo dev, Service.Releases.ANGLERelease release) {
        if (dev == null || dev.device.getConfiguration().getABIsCount() == 0) {
          return null;
        }

        switch (dev.device.getConfiguration().getABIs(0).getArchitecture()) {
          case ARMv8a: return release.getArm64();
          case ARMv7a: return release.getArm32();
          case X86: return release.getX86();
          default: return null;
        }
      }

      public boolean isReady() {
        TraceType type = getSelectedType();
        DeviceCaptureInfo dev = getSelectedDevice();
        return isInputReady(type, dev) && deviceValidationView.PassesValidation() &&
            (type != TraceType.ANGLE || dev.device.getConfiguration().getAngle().getVersion() > 0);
      }

      public boolean isInputReady(TraceType type, DeviceCaptureInfo dev) {
        if (type == null || dev == null) {
          return false;
        }
        Service.TraceTypeCapabilities config = type.getCapabilities(dev);

        return config != null &&
            (!config.getRequiresApplication() || !traceTarget.getText().isEmpty()) &&
            !directory.getText().isEmpty() && !file.getText().isEmpty();
      }

      public void addModifyListener(Listener listener) {
        device.getCombo().addListener(SWT.Selection, listener);
        api.getCombo().addListener(SWT.Selection, listener);
        traceTarget.addBoxListener(SWT.Modify, listener);
        directory.addBoxListener(SWT.Modify, listener);
        file.addListener(SWT.Modify, listener);
        this.addListener(SWT.Modify, listener);
        deviceValidationView.addListener(SWT.Modify, listener);
      }

      public void setDevices(Settings settings, List<DeviceCaptureInfo> devices) {
        this.devices = devices;
        updateDevicesDropDown(settings.trace());
      }

      public TraceRequest getTraceRequest(Settings settings) {
        TraceType type = getSelectedType();
        DeviceCaptureInfo dev = getSelectedDevice();
        TraceTypeCapabilities config = type.getCapabilities(dev);
        File output = getOutputFile();
        SettingsProto.Trace.Builder trace = settings.writeTrace();
        StartType start = StartType.values()[startType.getSelectionIndex()];

        trace.setDeviceSerial(dev.device.getSerial());
        trace.setDeviceName(dev.device.getName());
        trace.setType(config.getType().name());
        trace.setUri(traceTarget.getText());
        trace.setArguments(arguments.getText());
        SettingsProto.Trace.Duration.Builder dur = (type == TraceType.System) ?
            trace.getProfileDurationBuilder() : trace.getGfxDurationBuilder();
        dur.setType(start.proto);
        if (start == StartType.Frame) {
          dur.setStartFrame(startFrame.getSelection());
        } else if (start == StartType.Time) {
          dur.setStartTime(startFrame.getSelection());
        }
        dur.setDuration(duration.getSelection());
        trace.setWithoutBuffering(withoutBuffering.getSelection());
        trace.setOutDir(directory.getText());
        trace.setFriendlyName(friendlyName);
        trace.setProcessName(processName.getText());

        Service.TraceOptions.Builder options = Service.TraceOptions.newBuilder()
            .setDevice(dev.path)
            .setType(config.getType())
            .setUri(traceTarget.getText())
            .setAdditionalCommandLineArgs(arguments.getText())
            .setFramesToCapture(duration.getSelection())
            .setNoBuffer(withoutBuffering.getSelection())
            .setHideUnknownExtensions(!includeUnsupportedExtensions.getSelection())
            .setServerLocalSavePath(output.getAbsolutePath())
            .setProcessName(processName.getText());

        if (enableLoadValidationLayer.get()) {
          trace.setLoadValidationLayer(loadValidationLayer.getSelection());
          options.setLoadValidationLayer(loadValidationLayer.getSelection());
        }

        if (dev.config.getCanSpecifyCwd()) {
          trace.setCwd(cwd.getText());
          options.setCwd(cwd.getText());
        }
        if (dev.config.getCanSpecifyEnv()) {
          trace.setEnv(envVars.getText());
          options.addAllEnvironment(splitEnv(envVars.getText()));
        }
        int delay = 0;
        switch (start) {
          case Time:
            delay = startFrame.getSelection();
            // $FALL-THROUGH$
          case Manual:
            options.setDeferStart(true);
            break;
          case Frame:
            options.setStartFrame(startFrame.getSelection());
            break;
          default:
            // Do nothing.
        }
        if (dev.config.getHasCache()) {
          trace.setClearCache(clearCache.getSelection());
          options.setClearCache(clearCache.getSelection());
        }

        if (type == TraceType.System) {
          options.setDuration(duration.getSelection());
          if (dev.isFuchsia()) {
            options.setFuchsiaTraceConfig(FuchsiaTraceConfigDialog.getConfig(settings));
          } else {
            int durationMs = duration.getSelection() * 1000;
            // TODO: this isn't really unlimited.
            durationMs = (durationMs == 0) ? (int)MINUTES.toMillis(10) : durationMs;
            options.setPerfettoConfig(getConfig(settings, dev, traceTarget.getText(), durationMs));
          }
        }

        return new TraceRequest(output, options.build(), delay);
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

      protected TraceType getSelectedType() {
        IStructuredSelection sel = api.getStructuredSelection();
        return sel.isEmpty() ? null : ((TraceType)sel.getFirstElement());
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

    private static enum StartType {
      Beginning(SettingsProto.Trace.Duration.Type.BEGINNING),
      Manual(SettingsProto.Trace.Duration.Type.MANUAL),
      Time(SettingsProto.Trace.Duration.Type.TIME),
      Frame(SettingsProto.Trace.Duration.Type.FRAME);

      public final SettingsProto.Trace.Duration.Type proto;

      private StartType(SettingsProto.Trace.Duration.Type proto) {
        this.proto = proto;
      }
    }
  }

  public static enum TraceType {
    System("System Profile"),
    Vulkan("Frame Profile - Vulkan"),
    ANGLE("Frame Profile - OpenGL on ANGLE");

    private static final String PERFETTO_EXTENSION = ".perfetto";
    private static final String FUCHSIA_EXTENSION = ".fxt";
    private static final String VULKAN_EXTENSION = ".gfxtrace";

    public final String label;

    private TraceType(String label) {
      this.label = label;
    }

    @Override
    public String toString() {
      return label;
    }

    public static TraceType from(Service.TraceType type) {
      switch (type) {
        case Perfetto: return System;
        case Fuchsia: return System;
        case Graphics: return Vulkan;
        case ANGLE: return ANGLE;
        default: throw new AssertionError();
      }
    }

    public Service.TraceTypeCapabilities getCapabilities(DeviceCaptureInfo dev) {
      switch (this) {
        case System:
          if (dev.isFuchsia()) {
            return dev.getTypeCapabilities(Service.TraceType.Fuchsia);
          } else {
            return dev.getTypeCapabilities(Service.TraceType.Perfetto);
          }
        case Vulkan: return dev.getTypeCapabilities(Service.TraceType.Graphics);
        case ANGLE:  return dev.getTypeCapabilities(Service.TraceType.ANGLE);
        default: throw new AssertionError();
      }
    }

    public String getFileExtension(DeviceCaptureInfo dev) {
      switch (this) {
        case System:
          return (dev != null && dev.isFuchsia()) ? FUCHSIA_EXTENSION : PERFETTO_EXTENSION;
        case Vulkan:
        case ANGLE:
          return VULKAN_EXTENSION;
        default: throw new AssertionError();
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
    private Label autoStartLabel;
    private final String capturingStatusLabel;

    private Tracer.Trace trace;

    private StatusResponse status;
    private Throwable error;
    private long autoStartTime = -1;
    private boolean started = false;

    public TraceProgressDialog(
        Shell shell, Analytics analytics, Tracer.TraceRequest request, Theme theme) {
      super(shell, theme);
      this.analytics = analytics;
      this.request = request;

      if (request.options.getType() == Service.TraceType.Perfetto ||
          request.options.getType() == Service.TraceType.Fuchsia) {
        int duration = (int)request.options.getDuration();
        if (duration <= 0) {
          capturingStatusLabel = "Capturing...";
        } else {
          capturingStatusLabel =
              "Capturing " + duration + " second" + (duration > 1 ? "s" : "") + "...";
        }
      } else {
        int framesToCapture = request.options.getFramesToCapture();
        capturingStatusLabel =
            "Capturing " + framesToCapture + " frame" + (framesToCapture > 1 ? "s" : "") + "...";
      }
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

      // Make sure the stop signal is issued when the dialog is closed.
      getShell().addListener(SWT.Close, e -> {
          if (!successful()) {
              trace.stop();
          }
      });

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

        if (request.delay > 0 && !started &&
            status.getStatus() == Service.TraceStatus.WaitingToStart) {
          if (autoStartTime < 0) {
            autoStartTime = System.currentTimeMillis() + SECONDS.toMillis(request.delay);
          } else if (autoStartTime <= System.currentTimeMillis()) {
            Button button = getButton(IDialogConstants.OK_ID);
            if (button != null) {
              button.setEnabled(false);
            }
            started = true;
            trace.start();
          }

          if (autoStartLabel != null) {
            autoStartLabel.setVisible(true);
            long left = Math.max(0,
                MILLISECONDS.toSeconds(autoStartTime - System.currentTimeMillis() + 999));
            autoStartLabel.setText("Automatically starting in " + left + "s...");
            autoStartLabel.requestLayout();
            Widgets.scheduleIfNotDisposed(autoStartLabel, 100, () -> update());
          }
        } else if (autoStartLabel != null) {
          autoStartLabel.setVisible(false);
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

    private String getStatusLabel(Service.TraceStatus traceStatus) {
      switch (traceStatus) {
        case Uninitialized:
          return "Sending request...";
        case Initializing:
          return "Initializing...";
        case WaitingToStart:
          return "Press 'Start' to begin capture";
        case Capturing:
          return capturingStatusLabel;
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
            if (request.options.getType() == Service.TraceType.Perfetto) {
              okPressed();
            } else {
              button.setText("Open Trace");
            }
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
      if (request.delay > 0) {
        ((GridLayout) parent.getLayout()).numColumns++;
        autoStartLabel = createLabel(parent, "Automatically starting in " + request.delay + "s...");
        autoStartLabel.setVisible(false);
      }
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
              if (!started) {
                started = true;
                trace.start();
              }
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

  /**
   * Dialog that shows progress of downloading and installing an ANGLE apk.
   */
  private static class InstallAngleDialog extends DialogBase {
    private ProgressBar progressBar;
    private Label statusLabel;
    private Status status = Status.DOWNLOADING;

    public InstallAngleDialog(Shell shell, Theme theme) {
      super(shell, theme);
    }

    @SuppressWarnings("CheckReturnValue")
    public static int showDialogAndInstallApk(
        Shell shell, Models models, Theme theme, Device.Instance device, String url) {
      InstallAngleDialog dialog = new InstallAngleDialog(shell, theme);
      Scheduler.EXECUTOR.schedule(
          () -> dialog.downloadAndInstall(shell, models, device, url),
          100, TimeUnit.MILLISECONDS /*give the dialog some time to show*/);
      return dialog.open();
    }

    @Override
    public String getTitle() {
      return Messages.INSTALL_ANGLE_TITLE;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = createComposite(area, new GridLayout(1, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      createLabel(container, Messages.INSTALL_ANGLE_TITLE);
      progressBar = withLayoutData(createProgressBar(container, 100),
          new GridData(SWT.FILL, SWT.TOP, true, false));
      statusLabel = createLabel(container, status.message);

      return area;
    }

    @Override
    public void create() {
      super.create();
      Button ok = getButton(IDialogConstants.OK_ID);
      if (ok != null) {
        ok.setText("Close AGI");
      }
      updateButtons();
    }

    @Override
    protected void handleShellCloseEvent()
    {
      if (status != Status.INSTALLING) {
        super.handleShellCloseEvent();
        setReturnCode(status == Status.DONE ? Window.OK : Window.CANCEL);
      }
    }

    public void scheduleUpdate(Status newStatus, int progress) {
      Widgets.scheduleIfNotDisposed(getShell(), () -> updateStatus(newStatus, progress));
    }

    public void updateStatus(Status newStatus, int progress) {
      if (getShell().isDisposed()) {
        return;
      }

      status = newStatus;
      progressBar.setSelection(progress);
      if (newStatus ==  Status.FAILED) {
        progressBar.setState(SWT.ERROR);
      }
      statusLabel.setText(newStatus.message);
      statusLabel.requestLayout();

      updateButtons();
    }

    private void updateButtons() {
      Button ok = getButton(IDialogConstants.OK_ID);
      if (ok != null) {
        ok.setEnabled(status == Status.DONE);
      }
      Button cancel = getButton(IDialogConstants.CANCEL_ID);
      if (cancel != null) {
        // Can only be cancelled while downloading or when installation failed.
        cancel.setEnabled(status == Status.DOWNLOADING || status == Status.FAILED);
      }
    }

    // This runs on a background thread.
    private void downloadAndInstall(Shell parent, Models models, Device.Instance dev, String url) {
      // ProgressBar: 5% get download size, 80% download, 15% install
      try {
        File tmpFile = File.createTempFile("agi_angle", ".apk");
        try (FileOutputStream out = new FileOutputStream(tmpFile)) {
          URLs.downloadWithProgressUpdates(new URL(url), out, new URLs.DownloadProgressListener() {
            @Override
            public boolean onProgress(long done, long total) {
              scheduleUpdate(Status.DOWNLOADING, (int)(5 + 80 * done / total));
              return !getShell().isDisposed();
            }
          });
        }

        scheduleUpdate(Status.INSTALLING, 85);
        if (getShell().isDisposed()) {
          return;
        }
        models.devices.installApp(dev, tmpFile).get();
        tmpFile.delete();
        scheduleUpdate(Status.DONE, 100);
      } catch (Exception e) {
        scheduleUpdate(Status.FAILED, 100);
        Widgets.scheduleIfNotDisposed(parent, () -> {
          ErrorDialog.showErrorDialog(
              parent, models.analytics, "Error downloading or installing ANGLE!", e);
          close();
        });
      }
    }

    private static enum Status {
      DOWNLOADING("Downloading APK..."),
      INSTALLING("Installing APK..."),
      DONE("All done. AGI restart required."),
      FAILED("Download/Install failed.");

      public final String message;

      private Status(String message) {
        this.message = message;
      }
    }
  }
}
