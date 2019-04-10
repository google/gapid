/*
 * Copyright (C) 2019 Google Inc.
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
import com.google.gapid.perfetto.models.QueryEngine;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Events;
import com.google.gapid.util.Loadable;
import com.google.gapid.views.StatusBar;

import org.eclipse.swt.widgets.Shell;

import java.util.logging.Logger;

/**
 * Model responsible for querying a Perfetto trace.
 */
public class Perfetto extends ModelBase<Perfetto.Data, Path.Capture, Loadable.Message, Perfetto.Listener> {
  private static final Logger LOG = Logger.getLogger(Perfetto.class.getName());

  private final StatusBar status;

  public Perfetto(
      Shell shell, Analytics analytics, Client client, Capture capture, StatusBar status) {
    super(LOG, shell, analytics, client, Listener.class);
    this.status = status;

    capture.addListener(new Capture.Listener() {
      @Override
      public void onCaptureLoadingStart(boolean maintainState) {
        reset();
      }

      @Override
      public void onCaptureLoaded(Loadable.Message error) {
        if (error == null && capture.isPerfetto()) {
          load(capture.getData().path, false);
        } else {
          reset();
        }
      }
    });
  }

  @Override
  protected ListenableFuture<Data> doLoad(Path.Capture source) {
    return Futures.immediateFuture(new Data(new QueryEngine(client, source, status)));
  }

  @Override
  protected void fireLoadStartEvent() {
    // Don't care about this event.
  }

  @Override
  protected void updateError(Loadable.Message error) {
    listeners.fire().onPerfettoLoaded(error);
  }

  @Override
  protected void fireLoadedEvent() {
    listeners.fire().onPerfettoLoaded(null);
  }

  public ListenableFuture<com.google.gapid.proto.perfetto.Perfetto.QueryResult> query(String sql) {
    if (!isLoaded()) {
      return Futures.immediateFailedFuture(new Exception("Perfetto not loaded"));
    }
    return getData().queries.query(sql);
  }

  public static class Data {
    protected final QueryEngine queries;

    public Data(QueryEngine queries) {
      this.queries = queries;
    }
  }

  public static interface Listener extends Events.Listener {
    /**
     * Event indicating that the Perfetto trace has finished loading.
     *
     * @param error the loading error or {@code null} if loading was successful.
     */
    public default void onPerfettoLoaded(Loadable.Message error) { /* empty */ }
  }
}
