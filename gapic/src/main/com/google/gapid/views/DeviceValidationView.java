/*
 * Copyright (C) 2022 Google Inc.
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
import static com.google.gapid.widgets.Widgets.createLink;
import static com.google.gapid.widgets.Widgets.withLayoutData;
import static com.google.gapid.widgets.Widgets.withSpans;
import static com.google.gapid.widgets.Widgets.createButton;
import static com.google.gapid.widgets.Widgets.createComposite;
import static com.google.gapid.widgets.Widgets.createGroup;
import static com.google.gapid.widgets.Widgets.createLabel;
import static com.google.gapid.widgets.Widgets.createTextbox;
import static java.util.logging.Level.WARNING;
import static java.util.logging.Level.SEVERE;

import com.google.gapid.models.Devices.DeviceValidationResult;
import com.google.gapid.models.Devices.DeviceCaptureInfo;
import com.google.gapid.models.Models;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.device.Device.Instance;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.SingleInFlight;
import com.google.gapid.rpc.UiErrorCallback;
import com.google.gapid.widgets.LoadingIndicator;
import com.google.gapid.widgets.Widgets;
import com.google.gapid.util.Messages;
import com.google.gapid.util.OS;
import com.google.gapid.util.URLs;

import org.eclipse.swt.graphics.Point;
import org.eclipse.swt.graphics.Rectangle;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.program.Program;
import org.eclipse.swt.widgets.Button;
import org.eclipse.swt.widgets.Composite;
import org.eclipse.swt.widgets.Event;
import org.eclipse.swt.widgets.Group;
import org.eclipse.swt.widgets.Label;
import org.eclipse.swt.widgets.Link;
import org.eclipse.swt.widgets.Listener;
import org.eclipse.swt.widgets.Text;
import org.eclipse.swt.SWT;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class DeviceValidationView extends Composite {
  protected static final Logger LOG = Logger.getLogger(DeviceValidationView.class.getName());

  private final Widgets widgets;
  private final Models models;
  private final SingleInFlight rpcController = new SingleInFlight();
    
  private boolean validationPassed;
  private LoadingIndicator.Widget statusLoader;
  private Label statusText;
  private Button retryButton;

  // Separate error text from help textas a workaround to enable proper text wrapping.
  private Group errorMessageGroup;
  private Composite helpComposite;

  public DeviceValidationView(Composite parent, Models models, Widgets widgets) {
    super(parent, SWT.NONE);
    this.widgets = widgets;
    this.models = models;

    validationPassed = false;

    setLayout(new GridLayout(/* numColumns= */ 3, /* makeColumnsEqualWidth= */ false));
    setLayoutData(new GridData(SWT.FILL, SWT.TOP, true, false));

    // Status icon (loader) & accompanying text
    statusLoader = widgets.loading.createWidgetWithImage(this, 
        widgets.theme.check(), widgets.theme.error());
    statusLoader.setLayoutData(new GridData(SWT.LEFT, SWT.CENTER, false, false));
    statusText = withLayoutData(createLabel(this, ""),
      new GridData(SWT.FILL, SWT.CENTER, true, false));
    retryButton = withLayoutData(createButton(this, "Retry", e -> {
      // Intentionally empty.
    }), new GridData(SWT.FILL, SWT.CENTER, false, true));

    statusLoader.setVisible(false);
    statusText.setVisible(false);
    retryButton.setVisible(false);
  }

  public void ValidateDevice(DeviceCaptureInfo deviceInfo) {
    if (deviceInfo == null) {
      statusLoader.setVisible(false);
      statusText.setVisible(false);
      retryButton.setVisible(false);
      return;
    }

    ValidateDevice(deviceInfo.device);
  }

  public void ValidateDevice(Device.Instance dev) {
    statusLoader.setVisible(true);
    statusText.setVisible(true);
    retryButton.setVisible(false);
    if (errorMessageGroup != null) {
      errorMessageGroup.dispose();
      errorMessageGroup = null;
      helpComposite.dispose();
      helpComposite = null;
      requestLayout();
    }

    DeviceValidationResult cachedResult = models.devices.getCachedValidationStatus(dev);
    if (cachedResult != null) {
      if (cachedResult.passed || cachedResult.skipped) {
        setValidationStatus(cachedResult);
        return;
      }
    }

    statusLoader.startLoading();
    statusText.setText("Device support is being validated");

    // Assign appropriate callback for retry button.
    for (Listener listener : retryButton.getListeners(SWT.Selection)) {
      retryButton.removeListener(SWT.Selection, listener);
    }
    retryButton.addListener(SWT.Selection, e -> {
      ValidateDevice(dev);
    });

    models.devices.validateDevice(dev, this::setValidationStatus);
  }

  private void setValidationStatus(DeviceValidationResult result) {
    boolean passedOrSkipped = result.passed || result.skipped;
    statusLoader.stopLoading();
    statusLoader.updateStatus(passedOrSkipped);
    validationPassed = passedOrSkipped;
    statusText.setText("Device support validation " + result.toString() + ".");
    notifyListeners(SWT.Modify, new Event());

    if (passedOrSkipped) {
      return;
    }

    // Extra details (i.e. error message & help text)
    retryButton.setVisible(true);

    errorMessageGroup = withLayoutData(createGroup(this, ""),
      withSpans(new GridData(SWT.FILL, SWT.TOP, true, false), 3, 1));
    Text errText = withLayoutData(
      createTextbox(errorMessageGroup, SWT.READ_ONLY | SWT.WRAP , result.validationFailureMsg),
      new GridData(SWT.LEFT, SWT.TOP, false, false));

    helpComposite = createComposite(this, new GridLayout(1, false));
    helpComposite.setLayoutData(withSpans(new GridData(), 3, 1));
    Link helpLink = withLayoutData(createLink(helpComposite, Messages.VALIDATION_FAILED_LANDING_PAGE, e -> {
      Program.launch(URLs.DEVICE_COMPATIBILITY_URL);
    }), new GridData(SWT.LEFT, SWT.TOP, false, false));
    if (result.tracePath.length() > 0) {
      Link traceLink = withLayoutData(createLink(helpComposite, "View <a>trace file</a>", e -> {
        try {
          OS.openFileInSystemExplorer(new File(result.tracePath));
        } catch (IOException exception) {
          LOG.log(SEVERE, "Failed to open log directory in system explorer", exception);
        }
      }), new GridData(SWT.LEFT, SWT.TOP, false, false));
    }

    // Resize the rest of the window if needed.
    Point curr = getShell().getSize();
    Point want = getShell().computeSize(curr.x, SWT.DEFAULT);
    if (want.y > curr.y) {
      getShell().setSize(new Point(curr.x, want.y));
    }

    requestLayout();
  }

  public boolean PassesValidation() {
    return validationPassed;
  }
}
