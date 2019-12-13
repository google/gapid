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

import static com.google.gapid.util.Paths.events;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.MoreFutures;

import org.eclipse.swt.widgets.Shell;

import java.util.Iterator;
import java.util.List;
import java.util.logging.Logger;

public class Timeline extends CaptureDependentModel.ForPath<Timeline.Data, Timeline.Listener>
    implements ApiContext.Listener {
  private static final Logger LOG = Logger.getLogger(Timeline.class.getName());

  private final Capture capture;
  private final ApiContext context;

  public Timeline(Shell shell, Analytics analytics, Client client, Capture capture,
      Devices devices, ApiContext context) {
    super(LOG, shell, analytics, client, Listener.class, capture, devices);
    this.capture = capture;
    this.context = context;

    context.addListener(this);
  }

  @Override
  public void onContextsLoaded() {
    onContextSelected(context.getSelectedContext());
  }

  @Override
  public void onContextSelected(FilteringContext ctx) {
    load(getSource(capture.getData()), false);
  }

  @Override
  protected Path.Any getSource(Capture.Data data) {
    FilteringContext ctx = context.isLoaded() ? context.getSelectedContext() : null;
    return (ctx == null) ? null : events(data.path, ctx);
  }

  @Override
  protected boolean shouldLoad(Capture.Data data) {
    return data.isGraphics();
  }

  @Override
  protected ListenableFuture<Data> doLoad(Path.Any path, Path.Device device) {
    return MoreFutures.transform(
        client.get(path, device), v -> new Data(device, v.getEvents().getListList()));
  }

  @Override
  protected void fireLoadStartEvent() {
    listeners.fire().onTimeLineLoadingStart();
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onTimeLineLoaded();
  }

  public Iterator<Service.Event> getEndOfFrames() {
    return getData().events.stream()
        .filter(e -> e.getKind() == Service.EventKind.LastInFrame)
        .iterator();
  }

  public static class Data extends DeviceDependentModel.Data {
    public final List<Service.Event> events;

    public Data(Path.Device device, List<Service.Event> events) {
      super(device);
      this.events = events;
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the time line is being loaded.
     */
    public default void onTimeLineLoadingStart() { /* empty */ }

    /**
     * Event indicating that the time line has been loaded.
     */
    public default void onTimeLineLoaded() { /* empty */ }
  }
}
