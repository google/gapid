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
package com.google.gapid.models;

import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.Report;
import com.google.gapid.proto.service.Service.Value;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;

import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

/**
 * Model containing the report details of the current capture.
 */
public class Reports extends CaptureDependentModel.ForValue<Service.Report, Reports.Listener> {
  private static final Logger LOG = Logger.getLogger(Reports.class.getName());

  private final Devices devices;

  public Reports(Shell shell, Client client, Devices devices, Capture capture) {
    super(LOG, shell, client, Listener.class, capture);
    this.devices = devices;

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDeviceChanged() {
        load(getPath(capture.getData()), false);
      }
    });
  }

  @Override
  protected Path.Any getPath(Path.Capture capturePath) {
    if (!devices.hasReplayDevice()) {
      return null;
    }
    return Path.Any.newBuilder()
        .setReport(Path.Report.newBuilder()
            .setCapture(capturePath)
            .setDevice(devices.getReplayDevice()))
        .build();
  }

  @Override
  protected Report unbox(Value value) {
    return value.getReport();
  }

  @Override
  protected void fireLoadStartEvent() {
    // Do nothing.
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onReportLoaded();
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the report items have been loaded.
     */
    public default void onReportLoaded() { /* empty */ }
  }
}
