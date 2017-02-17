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

import static com.google.gapid.widgets.Widgets.createCheckbox;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSpans;

import com.google.common.base.Throwables;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Models;
import com.google.gapid.models.Settings;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.server.Tracer;
import com.google.gapid.server.Tracer.TraceRequest;
import com.google.gapid.util.Messages;
import com.google.gapid.views.ActivityPickerDialog.Action;
import com.google.gapid.widgets.ActionTextbox;
import com.google.gapid.widgets.FileTextbox;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.dialogs.TitleAreaDialog;
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
import org.eclipse.swt.widgets.FileDialog;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Shell;
import org.eclipse.swt.widgets.Text;

import java.io.File;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.List;
import java.util.Optional;

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

  public static void showTracingDialog(Shell shell, Models models, Widgets widgets) {
    TraceInputDialog input = new TraceInputDialog(shell, models.settings, widgets);
    if (loadDevicesAndShowDialog(input, models) == Window.OK) {
      TraceProgressDialog progress = new TraceProgressDialog(shell, input.getValue());
      Tracer.Trace trace = Tracer.trace(shell.getDisplay(), input.getValue(), new Tracer.Listener() {
        @Override
        public void onProgress(String message) {
          progress.append(message);
        }

        @Override
        public void onFailure(Throwable error) {
          progress.append("Tracing failed:");
          progress.append(Throwables.getStackTraceAsString(error));
        }
      });
      progress.open();
      trace.stop();
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

  private static class TraceInputDialog extends TitleAreaDialog {
    private static final String DEFAULT_TRACE_FILE = "trace.gfxtrace";
    private static final DateFormat TRACE_DATE_FORMAT = new SimpleDateFormat("_yyyyMMdd_HHmm");

    private final Settings settings;
    protected final Widgets widgets;
    private ComboViewer device;
    private ActionTextbox traceTarget;
    private FileTextbox.Directory directory;
    private Text file;
    private Button clearCache;
    private Button disablePcs;
    private boolean userHasChangedOutputFile = false;
    private List<Instance> devices;

    private Tracer.TraceRequest value;

    public TraceInputDialog(Shell shell, Settings settings, Widgets widgets) {
      super(shell);
      this.settings = settings;
      this.widgets = widgets;
    }

    public void setDevices(List<Instance> devices) {
      this.devices = devices;
      updateDevicesDropDown();
    }


    public Tracer.TraceRequest getValue() {
      return value;
    }

    @Override
    public void create() {
      super.create();
      setTitle(Messages.CAPTURE_TRACE);
    }

    @Override
    protected boolean isResizable() {
      return true;
    }

    @Override
    protected void configureShell(Shell newShell) {
      super.configureShell(newShell);
      newShell.setText(Messages.TRACE);
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = createComposite(area, new GridLayout(2, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      createLabel(container, "Device:");
      device = createDeviceDropDown(container);
      device.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      createLabel(container, "Package / Action:");
      traceTarget = withLayoutData(new ActionTextbox(container, settings.tracePackage) {
        @Override
        protected String createAndShowDialog(String current) {
          ActivityPickerDialog dialog = new ActivityPickerDialog(
              getShell(), widgets, getSelectedDevice());
          dialog.open();
          Action action = dialog.getSelected();
          return (action == null) ? null :
            action.action.getName() + ":" + action.pkg.getName() + "/" + action.activity.getName();
        }
      }, new GridData(SWT.FILL, SWT.FILL, true, false));

      createLabel(container, "Output Directory:");
      directory = withLayoutData(new FileTextbox.Directory(container, settings.traceOutDir) {
        @Override
        protected void configureDialog(DirectoryDialog dialog) {
          dialog.setText(Messages.CAPTURE_DIRECTORY);
        }
      }, new GridData(SWT.FILL, SWT.FILL, true, false));

      createLabel(container, "File Name:");
      file = withLayoutData(createTextbox(container, settings.traceOutFile),
          new GridData(SWT.FILL, SWT.FILL, true, false));

      clearCache = withLayoutData(
          createCheckbox(container, "Clear package cache", settings.traceClearCache),
          withSpans(new GridData(SWT.FILL, SWT.FILL, true, false), 2, 1));
      disablePcs = withLayoutData(
          createCheckbox(container, "Disable pre-compiled shaders", settings.traceDisablePcs),
          withSpans(new GridData(SWT.FILL, SWT.FILL, true, false), 2, 1));

      updateDevicesDropDown();

      traceTarget.addListener(SWT.Modify, e -> {
        if (!userHasChangedOutputFile) {
          String pkg = traceTarget.getText();
          int p = pkg.lastIndexOf('.');
          if (p >= pkg.length() - 1) {
            file.setText(DEFAULT_TRACE_FILE);
          } else {
            file.setText(pkg.substring(p + 1) + ".gfxtrace");
          }
          userHasChangedOutputFile = false;
        }
      });
      file.addListener(SWT.Modify, e -> {
        userHasChangedOutputFile = true;
      });

      return area;
    }

    private static ComboViewer createDeviceDropDown(Composite parent) {
      ComboViewer combo = new ComboViewer(parent, SWT.READ_ONLY);
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
      combo.setUseHashlookup(true);
      return combo;
    }

    private void updateDevicesDropDown() {
      if (device != null && devices != null) {
        device.setInput(devices);
        boolean found = false;
        if (!settings.traceDevice.isEmpty()) {
          Optional<Device.Instance> deflt = devices.stream()
              .filter(dev -> settings.traceDevice.equals(dev.getSerial()))
              .findAny();
          if (deflt.isPresent()) {
            device.setSelection(new StructuredSelection(deflt.get()));
            found = true;
          }
        }
        traceTarget.setEnabled(found);
      }
    }


    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      Button ok = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      createButton(parent, IDialogConstants.CANCEL_ID, IDialogConstants.CANCEL_LABEL, false);

      ok.setEnabled(false);
      Listener modifyListener = e -> {
        traceTarget.setActionEnabled(device.getCombo().getSelectionIndex() >= 0);
        ok.setEnabled(
            device.getCombo().getSelectionIndex() >= 0 &&
            !traceTarget.getText().isEmpty() &&
            !file.getText().isEmpty());
      };
      device.getCombo().addListener(SWT.Selection, modifyListener);
      traceTarget.addListener(SWT.Modify, modifyListener);
      file.addListener(SWT.Modify, modifyListener);
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (buttonId == IDialogConstants.OK_ID) {
        String target = traceTarget.getText();
        int actionSep = target.indexOf(":");
        int pkgSep = target.indexOf("/");
        if (actionSep >= 0 && pkgSep > actionSep) {
          String action = target.substring(0, actionSep);
          String pkg = target.substring(actionSep + 1, pkgSep);
          String activity = target.substring(pkgSep + 1);
          value = new TraceRequest(getSelectedDevice(), pkg, activity, action, getOutputFile(),
              clearCache.getSelection(), disablePcs.getSelection());
        } else {
          value = new TraceRequest(getSelectedDevice(), target, getOutputFile(),
              clearCache.getSelection(), disablePcs.getSelection());
        }
        settings.traceDevice = getSelectedDevice().getSerial();
        settings.tracePackage = traceTarget.getText();
        settings.traceOutDir = directory.getText();
        settings.traceOutFile = file.getText();
        settings.traceClearCache = clearCache.getSelection();
        settings.traceDisablePcs = disablePcs.getSelection();
      }
      super.buttonPressed(buttonId);
    }

    protected Device.Instance getSelectedDevice() {
      int index = device.getCombo().getSelectionIndex();
      return (index < 0) ? Device.Instance.getDefaultInstance() :
          (Device.Instance)device.getElementAt(index);
    }

    private File getOutputFile() {
      String name = file.getText();
      if (name.isEmpty()) {
        name = DEFAULT_TRACE_FILE;
      }
      if (!userHasChangedOutputFile) {
        int p = name.lastIndexOf('.');
        if (p < 0) {
          name = name + TRACE_DATE_FORMAT.format(new Date());
        } else {
          name = name.substring(0, p) + TRACE_DATE_FORMAT.format(new Date()) + name.substring(p);
        }
      }

      String dir = directory.getText();
      return dir.isEmpty() ? new File(name) : new File(dir, name);
    }
  }

  private static class TraceProgressDialog extends TitleAreaDialog {
    private final StringBuilder log = new StringBuilder();
    private final Tracer.TraceRequest request;
    private Text text;

    public TraceProgressDialog(Shell shell, Tracer.TraceRequest request) {
      super(shell);
      this.request = request;
    }

    public void append(String line) {
      log.append(line).append(text.getLineDelimiter());
      text.setText(log.toString());
      text.setSelection(text.getCharCount() - 1);
    }

    @Override
    public void create() {
      super.create();
      setTitle(Messages.CAPTURING_TRACE);
      setMessage("Capturing " + request.action + " to " + request.output.getName());
    }

    @Override
    protected void configureShell(Shell newShell) {
      super.configureShell(newShell);
      newShell.setText(Messages.TRACE);
    }

    @Override
    protected boolean isResizable() {
      return true;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite area = (Composite)super.createDialogArea(parent);

      Composite container = createComposite(area, new GridLayout(1, false));
      container.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      text = Widgets.createTextarea(container, log.toString());
      text.setEditable(false);
      text.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, true));

      return area;
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      createButton(parent, IDialogConstants.OK_ID, "Stop", true);
    }
  }
}
