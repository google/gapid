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

import static com.google.gapid.util.Logging.logLevel;
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.gapid.models.Settings;
import com.google.gapid.server.Tracer.TraceRequest;

import java.io.File;
import java.io.IOException;
import java.io.OutputStream;
import java.util.List;
import java.util.function.Consumer;
import java.util.logging.Logger;

/**
 * {@link ChildProcess} for the "gapit trace" command to handle capturing an API trace.
 */
public class GapitTraceProcess extends ChildProcess<Boolean> {
  private static final Logger LOG = Logger.getLogger(GapitTraceProcess.class.getName());

  private static final String TRACING_MESSAGE = "Press enter to stop capturing...";

  private final TraceRequest request;
  private final Consumer<String> onOutput;

  public GapitTraceProcess(Settings settings, TraceRequest request, Consumer<String> onOutput) {
    super("gapit", settings);
    this.request = request;
    this.onOutput = onOutput;
  }

  @Override
  protected Exception prepare(ProcessBuilder pb) {
    File gapit = GapiPaths.gapit();
    if (gapit == null || !gapit.exists()) {
      LOG.log(WARNING, "Could not find gapit for tracing.");
      return new Exception("Could not find the gapit executable.");
    }

    List<String> args = Lists.newArrayList();
    args.add(gapit.getAbsolutePath());

    args.add("-log-level");
    args.add(logLevel.get().gapisLevel);

    args.add("trace");

    String adb = GapiPaths.adb(settings);
    if (!adb.isEmpty()) {
      args.add("--adb");
      args.add(adb);
    }

    request.appendCommandLine(args);

    pb.command(args);
    return null;
  }

  @Override
  protected OutputHandler<Boolean> createStdoutHandler() {
    return new LoggingStringHandler<Boolean>(LOG, name, false, line -> {
      if (TRACING_MESSAGE.equals(line)) {
        onOutput.accept("Tracing...");
        return Boolean.TRUE;
      }

      onOutput.accept(line);
      return null;
    });
  }

  @Override
  protected OutputHandler<Boolean> createStderrHandler() {
    return new LoggingStringHandler<Boolean>(LOG, name, true, line -> {
      onOutput.accept(line);
      return null;
    });
  }

  /**
   * Only required for mid execution capture. Since the (current) mechanism to start the trace
   * is the same as to end it, this function should never be called for non mid execution captures.
   */
  public void startTracing() {
    if (isRunning()) {
      LOG.log(INFO, "Attempting to start the trace.");
      sendEnter();
    }
  }

  public void stopTracing() {
    if (isRunning()) {
      LOG.log(INFO, "Attempting to end the trace.");
      sendEnter();
    }
  }

  private void sendEnter() {
    try {
      OutputStream out = process.getOutputStream();
      out.write('\n');
      out.flush();
    } catch (IOException e) {
      LOG.log(WARNING, "Failed to send the 'enter' command to the trace", e);
      shutdown();
    }
  }
}
