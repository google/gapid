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

import com.google.gapid.models.ApiContext.FilteringContext;
import com.google.gapid.proto.service.Service;
import com.google.gapid.rpclib.futures.FutureController;
import com.google.gapid.rpclib.futures.SingleInFlight;
import com.google.gapid.rpclib.rpccore.Rpc;
import com.google.gapid.rpclib.rpccore.Rpc.Result;
import com.google.gapid.rpclib.rpccore.RpcException;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Paths;
import com.google.gapid.util.UiCallback;

import org.eclipse.swt.widgets.Shell;

import java.util.Iterator;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.logging.Logger;

public class Timeline implements ApiContext.Listener {
  private static final Logger LOG = Logger.getLogger(Timeline.class.getName());

  private final Shell shell;
  private final Client client;
  private final Capture capture;
  private final ApiContext context;
  private final FutureController rpcController = new SingleInFlight();
  private final Events.ListenerCollection<Listener> listeners = Events.listeners(Listener.class);
  private List<Service.Event> events;

  public Timeline(Shell shell, Client client, Capture capture, ApiContext context) {
    this.shell = shell;
    this.client = client;
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
    events = null;
    listeners.fire().onTimeLineLoadingStart();
    Rpc.listen(client.get(Paths.events(capture.getCapture(), ctx)), rpcController,
        new UiCallback<Service.Value, List<Service.Event>>(shell, LOG) {
      @Override
      protected List<Service.Event> onRpcThread(Result<Service.Value> result)
          throws RpcException, ExecutionException {
        return result.get().getEvents().getListList();
      }

      @Override
      protected void onUiThread(List<Service.Event> result) {
        update(result);
      }
    });
  }

  protected void update(List<Service.Event> newEvents) {
    events = newEvents;
    listeners.fire().onTimeLineLoaded();
  }

  public boolean isLoaded() {
    return events != null;
  }

  public Iterator<Service.Event> getEndOfFrames() {
    return events.stream()
        .filter(e -> e.getKind() == Service.EventKind.LastInFrame)
        .iterator();
  }

  public void addListener(Listener listener) {
    listeners.addListener(listener);
  }

  public void removeListener(Listener listener) {
    listeners.removeListener(listener);
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
