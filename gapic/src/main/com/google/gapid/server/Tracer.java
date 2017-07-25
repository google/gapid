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
import com.google.gapid.models.Settings;
import com.google.gapid.proto.device.Device;

import org.eclipse.swt.widgets.Display;

import java.io.File;
import java.util.List;

/**
 * Handles capturing an API trace.
 */
public class Tracer {
  public static Trace trace(
      Display display, Settings settings, TraceRequest request, Listener listener) {
    GapitTraceProcess process = new GapitTraceProcess(settings, request, message ->
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
      public void start() {
        process.startTracing();
      }

      @Override
      public void stop() {
        process.stopTracing();
      }
    };
  }

  @SuppressWarnings("unused")
  public static interface Listener {
    /**
     * Event indicating output from the tracing process.
     */
    public default void onProgress(String message) { /* empty */ }

    /**
     * Event indicating that tracing has failed.
     */
    public default void onFailure(Throwable error) { /* empty */ }
  }

  /**
   * Trace callback interface.
   */
  public static interface Trace {
    /**
     * Requests the current trace to start capturing. Only valid for mid-execution traces.
     */
    public void start();

    /**
     * Requests the current trace to be stopped.
     */
    public void stop();
  }

  /**
   * Contains information about how and what application to trace.
   */
  public static interface TraceRequest {
    public List<String> appendCommandLine(List<String> cmd);
    public File getOutput();
    public String getProgressDialogTitle();
    public boolean usesMidExecutionCapture();
  }

  /**
   * Contains information about how and what android application to trace.
   */
  public static class AndroidTraceRequest implements TraceRequest {
    public final String api;
    public final Device.Instance device;
    public final String pkg;
    public final String activity;
    public final String action;
    public final File output;
    public final boolean midExecution;
    public final boolean clearCache;
    public final boolean disablePcs;

    public AndroidTraceRequest(String api, Device.Instance device, String action, File output,
        boolean midExecution, boolean clearCache, boolean disablePcs) {
      this(api, device, null, null, action, output, midExecution, clearCache, disablePcs);
    }

    public AndroidTraceRequest(String api, Device.Instance device, String pkg, String activity,
        String action, File output, boolean midExecution, boolean clearCache, boolean disablePcs) {
      this.api = api;
      this.device = device;
      this.pkg = pkg;
      this.activity = activity;
      this.action = action;
      this.output = output;
      this.midExecution = midExecution;
      this.clearCache = clearCache;
      this.disablePcs = disablePcs;
    }

    @Override
    public List<String> appendCommandLine(List<String> cmd) {
      if (api != null) {
        cmd.add("-api");
        cmd.add(api);
      }
      if (!device.getSerial().isEmpty()) {
        cmd.add("-gapii-device");
        cmd.add(device.getSerial());
      }
      cmd.add("-out");
      cmd.add(output.getAbsolutePath());
      if (midExecution) {
        cmd.add("-start-defer");
      }
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
    public File getOutput() {
      return output;
    }

    @Override
    public String getProgressDialogTitle() {
      return "Capturing " + ((pkg != null) ? pkg : action) + " to " + output.getName();
    }

    @Override
    public boolean usesMidExecutionCapture() {
      return midExecution;
    }

    @Override
    public String toString() {
      return appendCommandLine(Lists.newArrayList()).toString();
    }
  }

  public static class DesktopTraceRequest implements TraceRequest {
    public final File executable;
    public final String args;
    public final File output;
    public final boolean midExecution;

    public DesktopTraceRequest(File executable, String args, File output, boolean midExecution) {
      this.executable = executable;
      this.args = args;
      this.output = output;
      this.midExecution = midExecution;
    }

    @Override
    public List<String> appendCommandLine(List<String> cmd) {
      cmd.add("-api");
      cmd.add("vulkan");

      cmd.add("-local-app");
      cmd.add(executable.getAbsolutePath());

      if (!args.isEmpty()) {
        cmd.add("-local-args");
        cmd.add(args);
      }

      if (midExecution) {
        cmd.add("-start-defer");
      }

      cmd.add("-out");
      cmd.add(output.getAbsolutePath());

      return cmd;
    }

    @Override
    public File getOutput() {
      return output;
    }

    @Override
    public String getProgressDialogTitle() {
      return "Capturing " + executable.getName() + " to " + output.getName();
    }

    @Override
    public boolean usesMidExecutionCapture() {
      return midExecution;
    }

    @Override
    public String toString() {
      return appendCommandLine(Lists.newArrayList()).toString();
    }
  }
}
