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
package com.google.gapid.perfetto.models;

import static com.google.gapid.util.MoreFutures.logFailure;
import static com.google.gapid.util.MoreFutures.transform;
import static com.google.gapid.widgets.Widgets.scheduleIfNotDisposed;
import static java.util.concurrent.TimeUnit.MILLISECONDS;

import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.proto.perfetto.Perfetto;
import com.google.gapid.proto.service.path.Path;
import com.google.gapid.server.Client;
import com.google.gapid.util.Scheduler;
import com.google.gapid.views.StatusBar;

import java.util.concurrent.atomic.AtomicBoolean;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.logging.Logger;

/**
 * Interface to the trace processor query executor.
 */
public class QueryEngine {
  private static final Logger LOG = Logger.getLogger(QueryEngine.class.getName());

  private final Client client;
  private final Path.Capture capture;
  private final StatusBar status;
  private final AtomicInteger scheduled = new AtomicInteger(0);
  private final AtomicInteger done = new AtomicInteger(0);
  private final AtomicBoolean updating = new AtomicBoolean(false);

  public QueryEngine(Client client, Path.Capture capture, StatusBar status) {
    this.client = client;
    this.capture = capture;
    this.status = status;
  }

  public ListenableFuture<Perfetto.QueryResult> query(String sql) {
    scheduled.incrementAndGet();
    updateStatus();
    return transform(client.perfettoQuery(capture, sql), r -> {
      done.incrementAndGet();
      updateStatus();
      return r;
    });
  }

  private void updateStatus() {
    if (updating.compareAndSet(false, true)) {
      scheduleIfNotDisposed(status, () -> {
        updating.set(false);
        int d = done.get(), s = scheduled.get();
        if (s == 0) {
          status.setServerStatusPrefix("");
        } else {
          status.setServerStatusPrefix("Queries: " + d + "/" + s);
        }

        if (s != 0 && d == s) {
          logFailure(LOG, Scheduler.EXECUTOR.schedule(() -> {
            int dd = done.get();
            if (scheduled.compareAndSet(dd, 0)) {
              done.updateAndGet(x -> x - dd);
              updateStatus();
            }
          }, 250, MILLISECONDS));
        }
      });
    }
  }
}
