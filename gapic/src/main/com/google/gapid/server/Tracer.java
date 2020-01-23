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
package com.google.gapid.server;

import com.google.gapid.proto.service.Service;
import com.google.gapid.proto.service.Service.TraceOptions;
import com.google.gapid.rpc.Rpc;
import com.google.gapid.rpc.Rpc.Result;
import com.google.gapid.rpc.RpcException;
import com.google.gapid.rpc.UiCallback;
import com.google.gapid.widgets.Widgets;

import org.eclipse.swt.widgets.Shell;

import java.io.File;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.atomic.AtomicBoolean;
import java.util.logging.Logger;

/**
 * Handles capturing an API trace.
 */
public class Tracer {
  private static final Logger LOG = Logger.getLogger(Tracer.class.getName());

  public static Trace trace(Client client, Shell shell, TraceRequest request, Listener listener) {
    AtomicBoolean done = new AtomicBoolean();
    GapidClient.StreamSender<Service.TraceRequest> sender = client.streamTrace(message -> {
      Widgets.scheduleIfNotDisposed(shell, () -> listener.onProgress(message));
      if (message.getStatus() == Service.TraceStatus.Done && done.compareAndSet(false, true)) {
        return GapidClient.Result.DONE;
      }
      return GapidClient.Result.CONTINUE;
    });

    Rpc.listen(sender.getFuture(), new UiCallback<Void, Throwable>(shell, LOG) {
      @Override
      protected Throwable onRpcThread(Result<Void> result) {
        done.set(true);
        try {
          result.get();
          return null;
        } catch (RpcException | ExecutionException e) {
          return e;
        }
      }

      @Override
      protected void onUiThread(Throwable result) {
        // Give some time for all the output to pump through.
        Widgets.scheduleIfNotDisposed(shell, 500, () -> {
          if (result == null) {
            listener.onFinished();
          } else {
            listener.onFailure(result);
          }
        });
      }
    });

    sender.send(Service.TraceRequest.newBuilder()
        .setInitialize(request.options)
        .build());

    return new Trace() {
      @Override
      public boolean start() {
        return sendEvent(Service.TraceEvent.Begin);
      }

      @Override
      public boolean getStatus() {
        return sendEvent(Service.TraceEvent.Status);
      }

      @Override
      public boolean stop() {
        return sendEvent(Service.TraceEvent.Stop);
      }

      private boolean sendEvent(Service.TraceEvent event) {
        if (done.get()) {
          return false;
        }

        sender.send(Service.TraceRequest.newBuilder()
            .setQueryEvent(event)
            .build());
        return true;
      }
    };
  }

  @SuppressWarnings("unused")
  public static interface Listener {
    /**
     * Event indicating output from the tracing process.
     */
    public default void onProgress(Service.StatusResponse status) { /* empty */ }

    /**
     * Event indicating that tracing has failed.
     */
    public default void onFailure(Throwable error) { /* empty */ }

    /**
     * Event indicating that tracing has completed successfully.
     */
    public default void onFinished() { /* empty */ }
  }

  /**
   * Trace callback interface.
   */
  public static interface Trace {
    /**
     * Requests the current trace to start capturing. Only valid for mid-execution traces.
     * @returns whether the start request was sent.
     */
    public boolean start();

    /**
     * Queries for trace status. The status is communicated via
     * {@link Listener#onProgress(com.google.gapid.proto.service.Service.StatusResponse)}.
     * @returns whether the status request was sent.
     */
    public boolean getStatus();

    /**
     * Requests the current trace to be stopped.
     * @returns whether the stop request was sent.
     */
    public boolean stop();
  }

  /**
   * Contains information about how and what application to trace.
   */
  public static class TraceRequest {
    public final File output;
    public final Service.TraceOptions options;
    public final int delay;

    public TraceRequest(File output, TraceOptions options, int delay) {
      this.output = output;
      this.options = options;
      this.delay = delay;
    }

    public boolean isMec() {
      return options.getDeferStart();
    }

    public String getProgressDialogTitle() {
      return "Capturing " + output.getName();
    }
  }
}
