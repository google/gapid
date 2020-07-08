/*
 * Copyright (C) 2020 Google Inc.
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

import static com.google.gapid.util.Logging.throttleLogRpcError;
import static com.google.gapid.util.MoreFutures.logFailure;
import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static com.google.gapid.widgets.Widgets.withIndents;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static java.util.logging.Level.WARNING;

import com.google.gapid.models.Capture;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Devices.DeviceValidationResult;
import com.google.gapid.models.Models;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.util.Loadable;
import com.google.gapid.util.Messages;
import com.google.gapid.util.Scheduler;
import com.google.gapid.util.URLs;
import com.google.gapid.widgets.DialogBase;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Widgets;

import org.eclipse.jface.dialogs.IDialogConstants;
import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Control;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Shell;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.logging.Logger;

/**
 * View responsible to show a replay device selection dialog when need be.
 */
public class DeviceDialog implements Devices.Listener, Capture.Listener {
  protected static final Logger LOG = Logger.getLogger(DeviceDialog.class.getName());

  private final Models models;
  private final Widgets widgets;
  private final Composite parent;
  protected SelectReplayDeviceDialog dialog = null;

  public DeviceDialog(Composite parent, Models models, Widgets widgets) {
    this.models = models;
    this.widgets = widgets;
    this.parent = parent;

    models.devices.addListener(this);
    models.capture.addListener(this);
    parent.addListener(SWT.Dispose, e -> {
      models.devices.removeListener(this);
      models.capture.removeListener(this);
      if (dialog != null && dialog.getShell() != null) {
        dialog.close();
      }
    });
  }

  @Override
  public void onReplayDevicesLoaded() {
    selectReplayDevice();
  }

  @Override
  public void onCaptureLoaded(Loadable.Message error) {
    selectReplayDevice();
  }

  protected void selectReplayDevice() {

    // If the dialog has been closed, remove the reference to it.
    if (dialog != null && dialog.getShell() == null) {
      dialog = null;
    }

    if (dialog != null) {
      // Dialog is already open, just refresh it
      dialog.refresh();
      return;
    }

    if (models.capture.isGraphics() && models.devices.isReplayDevicesLoaded() && !models.devices.hasReplayDevice()) {
      // Show dialog unless there is a single compatible and validated replay
      // device available, in which case it is auto-selected
      boolean skipDialog = false;
      Device.Instance device = null;
      if (models.devices.getReplayDevices() != null
          && models.devices.getReplayDevices().size() == 1) {
        device = models.devices.getReplayDevices().get(0);
        DeviceValidationResult result = models.devices.getValidationStatus(device);
        skipDialog = result.passed || result.skipped;
      }

      if (skipDialog) {
        models.devices.selectReplayDevice(device);
      } else {
        dialog = new SelectReplayDeviceDialog(parent.getShell(), models, widgets);
        scheduleIfNotDisposed(parent, () -> dialog.open());
      }
    }
  }

  /**
   * Dialog to select a replay device.
   */
  static private class SelectReplayDeviceDialog extends DialogBase {

    private final Models models;
    private final Widgets widgets;

    private Label noCompatibleDeviceFound;
    private ComboViewer deviceCombo;
    private LoadingIndicator.Widget deviceLoader;
    private LoadingIndicator.Widget validationStatusLoader;
    private Link validationStatusText;
    private boolean validationPassed;

    private final SingleInFlight rpcController = new SingleInFlight();

    public SelectReplayDeviceDialog(Shell shell, Models models, Widgets widgets) {
      super(shell, widgets.theme);
      this.models = models;
      this.widgets = widgets;
      validationPassed = false;
    }

    @Override
    public String getTitle() {
      return Messages.SELECT_DEVICE_TITLE;
    }

    @Override
    protected Control createDialogArea(Composite parent) {
      Composite composite = (Composite) super.createDialogArea(parent);

      // Recap capture info
      createLabel(composite, "Capture name: " + models.capture.getName());
      Instance dev = models.capture.getData().capture.getDevice();
      createLabel(composite,
          "Capture device: " + Devices.getLabel(dev) + " (Vulkan driver version: " + Devices.getVulkanDriverVersions(dev) + ")");

      // Warning when no compatible device found
      noCompatibleDeviceFound = createLabel(composite, Messages.SELECT_DEVICE_NO_COMPATIBLE_FOUND);
      noCompatibleDeviceFound.setForeground(theme.deviceNotFound());

      // Mirror the device combo from TracerDialog
      Group mainGroup =
          withLayoutData(createGroup(composite, "Select replay device", new GridLayout(3, false)),
              new GridData(GridData.FILL_HORIZONTAL));
      createLabel(mainGroup, "Device:");
      deviceCombo = createDropDownViewer(mainGroup);
      deviceCombo.setContentProvider(ArrayContentProvider.getInstance());
      deviceCombo.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
      deviceCombo.setLabelProvider(new LabelProvider() {
        @Override
        public String getText(Object element) {
          return Devices.getLabel(((Device.Instance) element));
        }
      });

      deviceLoader = widgets.loading.createWidgetWithRefresh(mainGroup);
      deviceLoader
          .setLayoutData(withIndents(new GridData(SWT.RIGHT, SWT.CENTER, false, false), 5, 0));
      // TODO: Make this a true button to allow keyboard use.
      deviceLoader.addListener(SWT.MouseDown, e -> {
        deviceLoader.startLoading();
        // By waiting a tiny bit, the icon will change to the loading indicator, giving the user
        // feedback that something is happening, in case the refresh is really quick.
        logFailure(LOG,
            Scheduler.EXECUTOR.schedule(
                () -> models.devices.loadReplayDevices(models.capture.getData().path), 300,
                TimeUnit.MILLISECONDS));
      });

      validationStatusLoader = widgets.loading.createWidgetWithImage(mainGroup,
          widgets.theme.check(), widgets.theme.error());
      validationStatusLoader
          .setLayoutData(withIndents(new GridData(SWT.LEFT, SWT.BOTTOM, false, false), 0, 0));
      validationStatusText = createLink(mainGroup, "", e -> {
        Program.launch(URLs.DEVICE_COMPATIBILITY_URL);
      });
      validationStatusText.setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));

      deviceCombo.getCombo().addListener(SWT.Selection, e -> {
        runValidationCheck(getSelectedDevice());
      });

      refresh();
      return composite;
    }

    @Override
    protected void createButtonsForButtonBar(Composite parent) {
      Button openTrace = createButton(parent, IDialogConstants.OK_ID, IDialogConstants.OK_LABEL, true);
      openTrace.setEnabled(validationPassed);
    }

    @Override
    protected void buttonPressed(int buttonId) {
      if (buttonId == IDialogConstants.OK_ID) {
        models.devices.selectReplayDevice(getSelectedDevice());
      }
      super.buttonPressed(buttonId);
    }

    protected void refresh() {
      boolean noReplayDevices =
          models.devices.getReplayDevices() == null || models.devices.getReplayDevices().isEmpty();

      noCompatibleDeviceFound.setVisible(noReplayDevices);
      noCompatibleDeviceFound.requestLayout();

      deviceCombo.setInput(models.devices.getReplayDevices());
      if (noReplayDevices) {
        deviceCombo.setSelection(null);
      } else {
        deviceCombo.setSelection(new StructuredSelection(models.devices.getReplayDevices().get(0)));
      }
      deviceCombo.getCombo().notifyListeners(SWT.Selection, new Event());
      deviceLoader.stopLoading();
    }

    private Device.Instance getSelectedDevice() {
      IStructuredSelection sel = deviceCombo.getStructuredSelection();
      return sel.isEmpty() ? null : (Device.Instance) sel.getFirstElement();
    }

    private void runValidationCheck(Device.Instance dev) {
      if (dev == null) {
        validationStatusLoader.setVisible(false);
        validationStatusText.setVisible(false);
        return;
      }
      validationStatusLoader.setVisible(true);
      validationStatusText.setVisible(true);
      // We need a DeviceCaptureInfo to do validation.
      setValidationStatus(models.devices.getValidationStatus(dev));
      if (!models.devices.getValidationStatus(dev).passed) {
        validationStatusLoader.startLoading();
        validationStatusText.setText("Device is being validated");
        rpcController.start().listen(models.devices.validateDevice(dev),
            new UiErrorCallback<DeviceValidationResult, DeviceValidationResult, DeviceValidationResult>(
                validationStatusLoader, LOG) {
              @Override
              protected ResultOrError<DeviceValidationResult, DeviceValidationResult> onRpcThread(
                  Rpc.Result<DeviceValidationResult> response)
                  throws RpcException, ExecutionException {
                try {
                  return success(response.get());
                } catch (RpcException | ExecutionException e) {
                  throttleLogRpcError(LOG, "LoadData error", e);
                  return error(null);
                }
              }

              @Override
              protected void onUiThreadSuccess(DeviceValidationResult result) {
                setValidationStatus(result);
              }

              @Override
              protected void onUiThreadError(DeviceValidationResult result) {
                LOG.log(WARNING, "UI thread error while validating device");
                setValidationStatus(result);
              }
            });
      }
    }

    protected void setValidationStatus(DeviceValidationResult result) {
      if (result.skipped) {
        validationStatusLoader.updateStatus(true);
        validationStatusLoader.stopLoading();
        validationStatusText.setText("Validation skipped.");
        validationPassed = true;
      } else {
        validationStatusLoader.updateStatus(result.passed);
        validationStatusText.setText("Validation "
            + (result.passed ? "Passed." : "Failed. " + Messages.VALIDATION_FAILED_LANDING_PAGE));
        validationStatusLoader.stopLoading();
        validationPassed = result.passed;
      }
      Button openTrace = getButton(IDialogConstants.OK_ID);
      if (openTrace != null) {
        openTrace.setEnabled(validationPassed);
      }
    }

  }

}
