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

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;

import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

/**
 * Model containing the report details of the current capture.
 */
public class Reports extends ModelBase.ForPath<Service.Report, Void, Reports.Listener> {
  private static final Logger LOG = Logger.getLogger(Reports.class.getName());

  private final Devices devices;

  public Reports(Shell shell, Client client, Capture capture, Devices devices, ApiContext context) {
    super(LOG, shell, client, Listener.class);
    this.devices = devices;

    devices.addListener(new Devices.Listener() {
      @Override
      public void onReplayDeviceChanged() {
        if (context.isLoaded()) {
          load(getPath(capture.getData(), context.getSelectedContext()), false);
        }
      }
    });

    context.addListener(new ApiContext.Listener() {
      @Override
      public void onContextsLoaded() {
        onContextSelected(context.getSelectedContext());
      }

      @Override
      public void onContextSelected(FilteringContext ctx) {
        load(getPath(capture.getData(), ctx), false);
      }
    });
  }

  protected Path.Any getPath(Path.Capture capturePath, FilteringContext context) {
    if (!devices.hasReplayDevice()) {
      return null;
    }
    return Path.Any.newBuilder()
        .setReport(context.report(Path.Report.newBuilder())
            .setCapture(capturePath)
            .setDevice(devices.getReplayDevice()))
        .build();
  }

  @Override
  protected ListenableFuture<Service.Report> doLoad(Path.Any source) {
    return Futures.transform(client.get(source), Service.Value::getReport);
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onReportLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onReportLoaded();
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
