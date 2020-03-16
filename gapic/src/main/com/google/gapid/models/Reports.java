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

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.Report;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.MoreFutures;

import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

/**
 * Model containing the report details of the current capture.
 */
public class Reports extends DeviceDependentModel.ForPath<Reports.Data, Void, Reports.Listener> {
  private static final Logger LOG = Logger.getLogger(Reports.class.getName());

  private final Devices devices;
  private final Capture capture;

  public Reports(
      Shell shell, Analytics analytics, Client client, Capture capture, Devices devices) {
    super(LOG, shell, analytics, client, Listener.class, devices);
    this.devices = devices;
    this.capture = capture;
  }

  protected Path.Any getPath(Path.Capture capturePath) {
    // TODO: the device is now duplicated.
    if (!devices.hasReplayDevice()) {
      return null;
    }
    return Path.Any.newBuilder()
        .setReport(Path.Report.newBuilder()
            .setCapture(capturePath)
            .setDevice(devices.getReplayDevicePath()))
        .build();
  }

  public void reload() {
    load(getPath(capture.getData().path), false);
  }

  @Override
  protected ListenableFuture<Data> doLoad(Path.Any source, Path.Device device) {
    return MoreFutures.transform(client.get(source, device), val -> new Data(device, val.getReport()));
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onReportLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onReportLoaded();
  }

  public static class Data extends DeviceDependentModel.Data {
    public final Service.Report report;

    public Data(Path.Device device, Report report) {
      super(device);
      this.report = report;
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the report items are being loaded.
     */
    public default void onReportLoadingStart() { /* empty */ }

    /**
     * Event indicating that the report items have been loaded.
     */
    public default void onReportLoaded() { /* empty */ }
  }
}
