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
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.createSpinner;
import static com.google.gapid.widgets.Widgets.createStandardTabFolder;
import static com.google.gapid.widgets.Widgets.createStandardTabItem;
import static com.google.gapid.widgets.Widgets.createTextarea;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.ifNotDisposed;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withMargin;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.base.Throwables;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.device.Device;
import com.google.gapid.server.Tracer;
import com.google.gapid.server.Tracer.AndroidTraceRequest;
import com.google.gapid.server.Tracer.DesktopTraceRequest;
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
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.jface.window.Window;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.DirectoryDialog;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Spinner;
import org.eclipse.swt.widgets.TabFolder;
import org.eclipse.swt.widgets.Text;

import java.io.File;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.List;
import java.util.Optional;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Dialogs used for capturing a trace.
 */
public class TracerDialog {
  private TracerDialog() {
  }

  public static void showOpenTraceDialog(Shell shell, Models models) {
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
    FileDialog dialog = new FileDialog(shell, SWT.SAVE);
    dialog.setFilterNames(new String[] { "Trace Files (*.gfxtrace)", "All Files" });
    dialog.setFilterExtensions(new String[] { "*.gfxtrace", "*" });
    dialog.setFilterPath(models.settings.lastOpenDir);
    String result = dialog.open();
    if (result != null) {
      models.capture.saveCapture(new File(result));
    }
  }

  public static void showTracingDialog(Shell shell, Models models, Widgets widgets) {
    TraceInputDialog input =
        new TraceInputDialog(shell, models.settings, widgets, models.devices::loadDevices);
    if (loadDevicesAndShowDialog(input, models) == Window.OK) {
      TraceProgressDialog progress = new TraceProgressDialog(shell, input.getValue(), widgets.theme);
      AtomicBoolean failed = new AtomicBoolean(false);
      Tracer.Trace trace = Tracer.trace(
          shell.getDisplay(), models.settings, input.getValue(), new Tracer.Listener() {
        @Override
        public void onProgress(String message) {
          progress.append(message);
        }

        @Override
        public void onFailure(Throwable error) {
          progress.append("Tracing failed:");
          progress.append(Throwables.getStackTraceAsString(error));
          failed.set(true);
        }
      });
      progress.setOnStart(trace::start);
      progress.open();
      trace.stop();
      if (!failed.get()) {
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
    private final Settings settings;
    private final Widgets widgets;
    private final Runnable refreshDevices;

    private TabFolder folder;
    private AndroidInput androidInput;
    private DesktopInput desktopInput;
    private List<Device.Instance> devices;

    private Tracer.TraceRequest value;

    public TraceInputDialog(
        Shell shell, Settings settings, Widgets widgets, Runnable refreshDevices) {
      super(shell, widgets.theme);
      this.settings = settings;
      this.widgets = widgets;
      this.refreshDevices = refreshDevices;
    }

    public void setDevices(List<Device.Instance> devices) {
      this.devices = devices;
      if (androidInput != null) {
        androidInput.setDevices(settings, devices);
      }
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

      // Mac has no Vulkan support, so cannot trace desktop apps.
      if (!OS.isMac) {
        folder = createStandardTabFolder(area);
        androidInput = new AndroidInput(folder, settings, widgets, refreshDevices);
        desktopInput = new DesktopInput(folder, settings, widgets);
        createStandardTabItem(folder, "Android", androidInput);
        createStandardTabItem(folder, "Desktop", desktopInput);
        folder.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
      } else {
        androidInput = new AndroidInput(area, settings, widgets, refreshDevices);
        androidInput.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
      }

      if (devices != null) {
        androidInput.setDevices(settings, devices);
      }
      return area;
    }

    private SharedTraceInput getInput() {
      return (folder == null) ? androidInput :
        (SharedTraceInput)folder.getItem(folder.getSelectionIndex()).getControl();
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      Button ok = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);

      Listener modifyListener = e -> {
        ok.setEnabled(getInput().isReady());
      };
      androidInput.addModifyListener(modifyListener);
      if (folder != null) {
        desktopInput.addModifyListener(modifyListener);
        folder.addListener(SWT.Selection, modifyListener);
      }

      modifyListener.handleEvent(null); // Set initial state of widgets.
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (buttonId == IDialogConstants.OK_ID) {
        value = getInput().getTraceRequest(settings);
      }
      super.buttonPressed(buttonId);
    }

    private abstract static class SharedTraceInput extends Composite {
      private static final String DEFAULT_TRACE_FILE = "trace";
      private static final String TRACE_EXTENSION = ".gfxtrace";
      private static final DateFormat TRACE_DATE_FORMAT = new SimpleDateFormat("_yyyyMMdd_HHmm");

      private final String date = TRACE_DATE_FORMAT.format(new Date());
      protected final ComboViewer api;
      protected final FileTextbox.Directory directory;
      protected final Text file;
      protected final Spinner frameCount;
      protected final Button fromBeginning;
      protected boolean userHasChangedOutputFile = false;

      public SharedTraceInput(Composite parent, Settings settings, Widgets widgets) {
        super(parent, SWT.NONE);
        setLayout(new GridLayout(2, false));

        createLabel(this, "API:");
        api = createApiDropDown(this, getDefaultApi(settings));

        buildTargetSelection(settings, widgets);

        createLabel(this, "Output Directory:");
        directory = withLayoutData(new FileTextbox.Directory(this, settings.traceOutDir) {
          @Override
          protected void configureDialog(DirectoryDialog dialog) {
            dialog.setText(Messages.CAPTURE_DIRECTORY);
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "File Name:");
        file = withLayoutData(
            createTextbox(this, ""), new GridData(SWT.FILL, SWT.FILL, true, false));

        file.addListener(SWT.Modify, e -> {
          userHasChangedOutputFile = true;
        });

        createLabel(this, "Stop After:");
        Composite frameCountComposite =
            createComposite(this, withMargin(new GridLayout(2, false), 0, 0));
        frameCount = withLayoutData(
            createSpinner(frameCountComposite, settings.traceFrameCount, 0, 999999),
            new GridData(SWT.LEFT, SWT.FILL, false, false));
        createLabel(frameCountComposite, "Frames (0 for unlimited)");

        createLabel(this, "");
        fromBeginning = withLayoutData(
            createCheckbox(this, "Trace From Beginning", !settings.traceMidExecution),
            new GridData(SWT.FILL, SWT.FILL, true, false));
      }

      protected String formatTraceName(String name) {
        return (name.isEmpty() ? DEFAULT_TRACE_FILE : name) + date + TRACE_EXTENSION;
      }

      protected abstract void buildTargetSelection(Settings settings, Widgets widgets);
      protected abstract Tracer.Api getDefaultApi(Settings settings);

      private static ComboViewer createApiDropDown(Composite parent, Tracer.Api selection) {
        ComboViewer combo = createDropDownViewer(parent);
        combo.setContentProvider(ArrayContentProvider.getInstance());
        combo.setLabelProvider(new LabelProvider() {
          @Override
          public String getText(Object element) {
            return ((Tracer.Api)element).displayName;
          }
        });
        for (Tracer.Api api : Tracer.Api.values()) {
          combo.add(api);
        }
        combo.setSelection(new StructuredSelection(selection));
        return combo;
      }

      public boolean isReady() {
        return api.getCombo().getSelectionIndex() >= 0 &&
            !file.getText().isEmpty() && !directory.getText().isEmpty();
      }

      public void addModifyListener(Listener listener) {
        api.getCombo().addListener(SWT.Selection, listener);
        file.addListener(SWT.Modify, listener);
        directory.addBoxListener(SWT.Modify, listener);
      }

      public TraceRequest getTraceRequest(Settings settings) {
        settings.traceApi = getSelectedApi().name();
        settings.traceOutDir = directory.getText();
        settings.traceFrameCount = frameCount.getSelection();
        settings.traceMidExecution = !fromBeginning.getSelection();

        return getTraceRequest(settings, getSelectedApi(), getOutputFile(),
            frameCount.getSelection(), !fromBeginning.getSelection());
      }

      protected abstract TraceRequest getTraceRequest(
          Settings settings, Tracer.Api traceApi, File output, int frames, boolean midExecution);

      protected Tracer.Api getSelectedApi() {
        return (Tracer.Api)api.getStructuredSelection().getFirstElement();
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

    private static class AndroidInput extends SharedTraceInput {
      private final Runnable refreshDevices;
      private ComboViewer device;
      private LoadingIndicator.Widget deviceLoader;
      private Label pcsWarning;
      private Link adbWarning;
      private ActionTextbox traceTarget;
      private Button clearCache;
      private Button disablePcs;
      private List<Device.Instance> devices;

      public AndroidInput(
          Composite parent, Settings settings, Widgets widgets, Runnable refreshDevices) {
        super(parent, settings, widgets);
        this.refreshDevices = refreshDevices;

        createLabel(this, "");
        clearCache = withLayoutData(
            createCheckbox(this, "Clear package cache", settings.traceClearCache),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "");
        disablePcs = withLayoutData(
            createCheckbox(this, "Disable pre-compiled shaders", settings.traceDisablePcs),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "");
        pcsWarning = withLayoutData(
            createLabel(this, "Warning: Pre-compiled shaders are not supported in the replay."),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        pcsWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_YELLOW));
        pcsWarning.setVisible(!settings.traceDisablePcs);

        withLayoutData(createLabel(this, ""), withSpans(new GridData(), 2, 1));

        createLabel(this, "");
        adbWarning = withLayoutData(
            createLink(this,
                "Path to adb missing. Please specify it in the <a>preferences</a> and restart.",
                e -> SettingsDialog.showSettingsDialog(getShell(), settings, widgets.theme)),
            new GridData(SWT.FILL, SWT.FILL, true, false));
        adbWarning.setForeground(getDisplay().getSystemColor(SWT.COLOR_DARK_RED));
        adbWarning.setVisible(false);

        device.getCombo().addListener(SWT.Selection,
            e -> traceTarget.setActionEnabled(device.getCombo().getSelectionIndex() >= 0));
        updateDevicesDropDown(settings);

        Listener targetListener = e -> {
          if (!userHasChangedOutputFile) {
            String pkg = traceTarget.getText();
            int actionSep = pkg.indexOf(":");
            int pkgSep = pkg.indexOf("/");
            if (actionSep >= 0 && pkgSep > actionSep) {
              pkg = pkg.substring(actionSep + 1, pkgSep);
            }
            file.setText(formatTraceName(pkg.substring(pkg.lastIndexOf('.') + 1)));
            userHasChangedOutputFile = false; // cancel the modify event from set call.
          }
        };
        traceTarget.addBoxListener(SWT.Modify, targetListener);
        targetListener.handleEvent(null);

        Listener apiListener = e -> {
          if (getSelectedApi() == Tracer.Api.Vulkan) {
            fromBeginning.setEnabled(true);
          } else {
            fromBeginning.setEnabled(false);
            fromBeginning.setSelection(true);
          }
        };
        api.getCombo().addListener(SWT.Selection, apiListener);
        apiListener.handleEvent(null);

        disablePcs.addListener(
            SWT.Selection, e -> pcsWarning.setVisible(!disablePcs.getSelection()));
      }

      @Override
      protected void buildTargetSelection(Settings settings, Widgets widgets) {
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

        createLabel(this, "Package / Action:");
        traceTarget = withLayoutData(new ActionTextbox(this, settings.tracePackage) {
          @Override
          protected String createAndShowDialog(String current) {
            ActivityPickerDialog dialog = new ActivityPickerDialog(
                getShell(), settings, widgets, getSelectedDevice());
            dialog.open();
            ActivityPickerDialog.Action action = dialog.getSelected();
            return (action == null) ? null : action.toString();
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        // Space used by the desktop trace tab, helps line things up a bit.
        if (!OS.isMac) {
          createLabel(this, "");
          createLabel(this, "");
        }
      }

      @Override
      protected Tracer.Api getDefaultApi(Settings settings) {
        Tracer.Api result = Tracer.Api.parse(settings.traceApi);
        return (result == null) ? Tracer.Api.GLES : result;
      }

      private static ComboViewer createDeviceDropDown(Composite parent) {
        ComboViewer combo = createDropDownViewer(parent);
        combo.setContentProvider(ArrayContentProvider.getInstance());
        combo.setLabelProvider(new LabelProvider() {
          @Override
          public String getText(Object element) {
            Device.Instance info = (Device.Instance)element;
            StringBuilder sb = new StringBuilder();
            if (!info.getConfiguration().getHardware().getName().isEmpty()) {
              sb.append(info.getConfiguration().getHardware().getName()).append(" - ");
            }
            if (!info.getConfiguration().getOS().getName().isEmpty()) {
              sb.append(info.getConfiguration().getOS().getName()).append(" - ");
            }
            return sb.append(info.getSerial()).toString();
          }
        });
        return combo;
      }

      @Override
      public boolean isReady() {
        return super.isReady() &&
            device.getCombo().getSelectionIndex() >= 0 &&
            !traceTarget.getText().isEmpty();
      }

      @Override
      public void addModifyListener(Listener listener) {
        super.addModifyListener(listener);
        device.getCombo().addListener(SWT.Selection, listener);
        traceTarget.addBoxListener(SWT.Modify, listener);
      }

      public void setDevices(Settings settings, List<Device.Instance> devices) {
        this.devices = devices;
        updateDevicesDropDown(settings);
      }

      private void updateDevicesDropDown(Settings settings) {
        if (device != null && devices != null) {
          deviceLoader.stopLoading();
          device.setInput(devices);
          if (!settings.traceDevice.isEmpty()) {
            Optional<Device.Instance> deflt = devices.stream()
                .filter(dev -> settings.traceDevice.equals(dev.getSerial()))
                .findAny();
            if (deflt.isPresent()) {
              device.setSelection(new StructuredSelection(deflt.get()));
            }
          }
          device.getCombo().notifyListeners(SWT.Selection, new Event());

          adbWarning.setVisible(devices.isEmpty() && settings.adb.isEmpty());
        } else if (deviceLoader != null) {
          deviceLoader.startLoading();
        }
      }

      @Override
      protected TraceRequest getTraceRequest(Settings settings, Tracer.Api traceApi, File output,
          int frames, boolean midExecution) {
        String target = traceTarget.getText();
        int actionSep = target.indexOf(":");
        int pkgSep = target.indexOf("/");

        settings.traceDevice = getSelectedDevice().getSerial();
        settings.tracePackage = traceTarget.getText();
        settings.traceClearCache = clearCache.getSelection();
        settings.traceDisablePcs = disablePcs.getSelection();

        if (actionSep >= 0 && pkgSep > actionSep) {
          String action = target.substring(0, actionSep);
          String pkg = target.substring(actionSep + 1, pkgSep);
          String activity = target.substring(pkgSep + 1);
          return new AndroidTraceRequest(traceApi, getSelectedDevice(), pkg, activity, action,
              output, frames, midExecution, clearCache.getSelection(), disablePcs.getSelection());
        } else {
          return new AndroidTraceRequest(traceApi, getSelectedDevice(), target, output,
              frames, midExecution, clearCache.getSelection(), disablePcs.getSelection());
        }
      }

      protected Device.Instance getSelectedDevice() {
        int index = device.getCombo().getSelectionIndex();
        return (index < 0) ? Device.Instance.getDefaultInstance() :
            (Device.Instance)device.getElementAt(index);
      }
    }

    private static class DesktopInput extends SharedTraceInput {
      private FileTextbox.File executable;
      private Text arguments;
      private FileTextbox.Directory cwd;
      private boolean userHasChangedCwd = false;

      public DesktopInput(Composite parent, Settings settings, Widgets widgets) {
        super(parent, settings, widgets);
        api.getCombo().setEnabled(false);

        Listener exeListener = e -> {
          if (!userHasChangedOutputFile) {
            String exe = executable.getText();
            int fileSep = exe.lastIndexOf(File.separator);
            if (fileSep >= 0) {
              exe = exe.substring(fileSep + 1);
            }
            int extSep = exe.lastIndexOf('.');
            if (extSep > 0) {
              exe = exe.substring(0, extSep);
            }
            file.setText(formatTraceName(exe));
            userHasChangedOutputFile = false; // cancel the modify event from set call.
          }

          if (!userHasChangedCwd) {
            File dir = new File(executable.getText()).getParentFile();
            if (dir != null && dir.exists() && dir.isDirectory()) {
              String path = dir.getAbsolutePath();
              if (path == null) {
                path = dir.getPath();
              }
              if (path != null) {
                cwd.setText(path);
                userHasChangedCwd = false; // cancel the modify event from set call.
              }
            }
          }
        };
        executable.addBoxListener(SWT.Modify, exeListener);
        exeListener.handleEvent(null);

        cwd.addBoxListener(SWT.Modify, e -> {
          userHasChangedCwd = true;
        });
      }

      @Override
      protected void buildTargetSelection(Settings settings, Widgets widgets) {
        createLabel(this, "Executable:");
        executable = withLayoutData(new FileTextbox.File(this, settings.traceExecutable) {
          @Override
          protected void configureDialog(FileDialog dialog) {
            dialog.setText(Messages.CAPTURE_EXECUTABLE);
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "Arguments:");
        arguments = withLayoutData(createTextbox(this, settings.traceArgs),
            new GridData(SWT.FILL, SWT.FILL, true, false));

        createLabel(this, "Working Directory:");
        cwd = withLayoutData(new FileTextbox.Directory(this, settings.traceCwd) {
          @Override
          protected void configureDialog(DirectoryDialog dialog) {
            dialog.setText(Messages.CAPTURE_CWD);
          }
        }, new GridData(SWT.FILL, SWT.FILL, true, false));
      }

      @Override
      protected Tracer.Api getDefaultApi(Settings settings) {
        return Tracer.Api.Vulkan;
      }

      @Override
      public boolean isReady() {
        return super.isReady() &&
            !executable.getText().isEmpty();
      }

      @Override
      public void addModifyListener(Listener listener) {
        super.addModifyListener(listener);
        executable.addBoxListener(SWT.Modify, listener);
      }

      @Override
      protected TraceRequest getTraceRequest(Settings settings, Tracer.Api traceApi, File output,
          int frames, boolean midExecution) {
        settings.traceExecutable = executable.getText();
        settings.traceArgs = arguments.getText();
        settings.traceCwd = cwd.getText();

        return new DesktopTraceRequest(
            new File(executable.getText()), arguments.getText(),
            cwd.getText().isEmpty() ? null : new File(cwd.getText()), output, frames,
            midExecution);
      }
    }
  }

  /**
   * Dialog that shows trace progress to the user and allows the user to stop the capture.
   */
  private static class TraceProgressDialog extends DialogBase {
    private final StringBuilder log = new StringBuilder();
    private final Tracer.TraceRequest request;
    private Text text;
    private Runnable onStart;

    public TraceProgressDialog(Shell shell, Tracer.TraceRequest request, Theme theme) {
      super(shell, theme);
      this.request = request;
    }

    public void setOnStart(Runnable onStart) {
      this.onStart = onStart;
    }

    public void append(String line) {
      ifNotDisposed(text, () -> {
        log.append(line).append(text.getLineDelimiter());
        int selection = text.getCharCount();
        text.setText(log.toString());
        text.setSelection(selection);
      });
    }

    @Override
    public String getTitle() {
      return Messages.CAPTURING_TRACE;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = createComposite(area, new GridLayout(1, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      createBoldLabel(container, request.getProgressDialogTitle())
          .setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

      text = createTextarea(container, log.toString());
      text.setEditable(false);
      text.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      return area;
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      createButton(parent, IDialogConstants.OK_ID, request.midExecution ? "Start" : "Stop", true);
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (IDialogConstants.OK_ID == buttonId && "Start".equals(getButton(buttonId).getText())) {
        getButton(buttonId).setText("Stop");
        onStart.run();
      } else {
        super.buttonPressed(buttonId);
      }
    }
  }
}
