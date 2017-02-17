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

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.Futures;
import com.google.gapid.proto.device.Device;

import org.eclipse.swt.widgets.Display;

import java.io.File;
import java.util.List;

public class Tracer {
  public static Trace trace(Display display, TraceRequest request, Listener listener) {
    GapitTraceProcess process = new GapitTraceProcess(request, message ->
      display.asyncExec(() -> listener.onProgress(message)));
    Futures.addCallback(process.start(), new FutureCallback<Boolean>() {
      @Override
      public void onFailure(Throwable t) {
        // Give some time for all the output to pump through.
        display.asyncExec(() -> display.timerExec(500, () -> listener.onFailure(t)));
      }

      @Override
      public void onSuccess(Boolean result) {
        // Ignore.
      }
    });

    return new Trace() {
      @Override
      public void stop() {
        process.stopTracing();
      }
    };
  }

  @SuppressWarnings("unused")
  public static interface Listener {
    public default void onProgress(String message) { /* empty */ }
    public default void onFailure(Throwable error) { /* empty */ }
  }

  public static interface Trace {
    public void stop();
  }

  public static class TraceRequest {
    public final Device.Instance device;
    public final String pkg;
    public final String activity;
    public final String action;
    public final File output;
    public final boolean clearCache;
    public final boolean disablePcs;

    public TraceRequest(Device.Instance device, String action, File output,
        boolean clearCache, boolean disablePcs) {
      this.device = device;
      this.pkg = null;
      this.activity = null;
      this.action = action;
      this.output = output;
      this.clearCache = clearCache;
      this.disablePcs = disablePcs;
    }

    public TraceRequest(Device.Instance device, String pkg, String activity, String action, File output,
        boolean clearCache, boolean disablePcs) {
      this.device = device;
      this.pkg = pkg;
      this.activity = activity;
      this.action = action;
      this.output = output;
      this.clearCache = clearCache;
      this.disablePcs = disablePcs;
    }

    public List<String> appendCommandLine(List<String> cmd) {
      if (!device.getSerial().isEmpty()) {
        cmd.add("-gapii-device");
        cmd.add(device.getSerial());
      }
      cmd.add("-out");
      cmd.add(output.getAbsolutePath());
      if (clearCache) {
        cmd.add("-clear-cache");
      }
      if (disablePcs) {
        cmd.add("-disable-pcs");
      }

      if (pkg != null) {
        cmd.add("-android-package");
        cmd.add(pkg);

        cmd.add("-android-activity");
        cmd.add(activity);

        cmd.add("-android-action");
        cmd.add(action);
      } else {
        cmd.add(action);
      }
      return cmd;
    }

    @Override
    public String toString() {
      return appendCommandLine(Lists.newArrayList()).toString();
    }
  }
}
