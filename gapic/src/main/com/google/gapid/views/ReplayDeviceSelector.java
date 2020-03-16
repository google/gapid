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

import static com.google.gapid.widgets.Widgets.createDropDownViewer;
import static com.google.gapid.widgets.Widgets.createLabel;

import com.google.gapid.models.Analytics.View;
import com.google.gapid.models.Devices;
import com.google.gapid.models.Models;
import com.google.gapid.proto.device.Device;
import com.google.gapid.proto.service.Service.ClientAction;

import org.eclipse.jface.viewers.ArrayContentProvider;
import org.eclipse.jface.viewers.ComboViewer;
import org.eclipse.jface.viewers.IStructuredSelection;
import org.eclipse.jface.viewers.LabelProvider;
import org.eclipse.jface.viewers.StructuredSelection;
import org.eclipse.swt.SWT;
import org.eclipse.swt.layout.GridData;
import org.eclipse.swt.layout.GridLayout;
import org.eclipse.swt.widgets.Composite;

/**
 * Shows a dropdown for the replay device selection.
 */
public class ReplayDeviceSelector extends Composite implements Devices.Listener {
  private final Models models;
  private final ComboViewer deviceCombo;

  public ReplayDeviceSelector(Composite parent, Models models) {
    super(parent, SWT.NONE);
    this.models = models;

    setLayout(new GridLayout(2, false));

    createLabel(this, "Replay Device:");
    deviceCombo = createDropDownViewer(this);
    deviceCombo.getCombo().setLayoutData(new GridData(SWT.FILL, SWT.FILL, true, false));
    deviceCombo.setContentProvider(ArrayContentProvider.getInstance());
    deviceCombo.setLabelProvider(new LabelProvider() {
      @Override
      public String getText(Object element) {
        return Devices.getLabel((Device.Instance)element);
      }
    });

    models.devices.addListener(this);
    addListener(SWT.Dispose, e -> {
      models.devices.removeListener(this);
    });

    deviceCombo.getCombo().addListener(SWT.Selection, e -> {
      models.analytics.postInteraction(View.ReplayDeviceSelector, ClientAction.Select);
      IStructuredSelection selection = deviceCombo.getStructuredSelection();
      if (!selection.isEmpty()) {
        models.devices.selectReplayDevice((Device.Instance)selection.getFirstElement());
      }
    });
  }

  @Override
  public void onReplayDevicesLoaded() {
    if (!models.devices.hasReplayDevice()) {
      return;
    }

    deviceCombo.setInput(models.devices.getReplayDevices());
    deviceCombo.refresh();

    onReplayDeviceChanged(models.devices.getSelectedReplayDevice());
  }

  @Override
  public void onReplayDeviceChanged(Device.Instance dev) {
    deviceCombo.setSelection(new StructuredSelection(dev));
  }
}
