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

import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.gapid.models.Settings;
import com.google.gapid.util.Experimental;
import com.google.gapid.util.Flags;
import com.google.gapid.util.Flags.Flag;
import com.google.gapid.util.Logging;
import com.google.gapid.util.MoreFutures;

import java.io.File;
import java.security.SecureRandom;
import java.util.Base64;
import java.util.List;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;
import java.util.logging.Logger;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * {@link ChildProcess} running the Graphics API Server (GAPIS) executable. The result of the
 * future returned by {@link #start()} is the port the GAPIS server listens to.
 */
public class GapisProcess extends ChildProcess<Integer> {
  public static final Flag<Boolean> disableGapisTimeout = Flags.value(
      "disable-gapis-timeout", false, "Disables the GAPIS timeout. Useful for debugging.", true);

  public static final Flag<String> gapisArgs = Flags.value(
      "gapis-args", "", "Additional argument to pass to gapis.");
  public static final Flag<String> gapirArgs = Flags.value(
      "gapir-args", "", "Additional argument to pass to gapir.");

  private static final Logger LOG = Logger.getLogger(GapisProcess.class.getName());

  private static final Pattern PORT_PATTERN = Pattern.compile("^Bound on port '(\\d+)'$", 0);

  /** The length in characters of an auth-token */
  private static final int AUTH_TOKEN_LENGTH = 8;

  private static final int SERVER_LAUNCH_TIMEOUT_MS = 10000;
  private static final int HEARTBEAT_RATE_MS = 1000;
  private static final int IDLE_TIMEOUT_MS = 60000;
  private static final String SERVER_HOST = "localhost";

  private final ListenableFuture<GapisConnection> connection;
  private final String authToken =  generateAuthToken();
  private final PanicDetector panicDetector = new PanicDetector();
  private final Listener listener;

  public GapisProcess(Settings settings, Listener listener) {
    super("gapis", settings);
    this.listener = (listener == null) ? Listener.NULL : listener;
    listener.onStatus("Starting gapis...");
    connection = MoreFutures.transform(start(), port -> {
      LOG.log(INFO, "Established a new client connection to " + port);
      return GapisConnection.create(
          SERVER_HOST + ":" + port, authToken, HEARTBEAT_RATE_MS, con -> shutdown());
    });
  }

  @Override
  protected Exception prepare(ProcessBuilder pb) throws GapiPaths.MissingToolsException {
    List<String> args = Lists.newArrayList();
    args.add(GapiPaths.get().gapis().getAbsolutePath());

    String gapirFlags = "";

    args.add("-enable-local-files");

    if (settings.preferences().getReportCrashes()) {
      args.add("-crashreport");
    }

    if (settings.analyticsEnabled()) {
      args.add("-analytics");
      args.add(settings.preferences().getAnalyticsClientId());
    }

    // Append all experimental flags if any is enabled.
    args.addAll(Experimental.getGapisFlags(settings.preferences().getEnableAllExperimentalFeatures()));

    File logDir = Logging.getLogDir();
    if (logDir != null) {
      args.add("-log-file");
      args.add(new File(logDir, "gapis.log").getAbsolutePath());
      args.add("-log-level");
      args.add(Logging.getGapisLogLevel());

      gapirFlags = "--log " + new File(logDir, "gapir.log").getAbsolutePath().replace("\\", "\\\\") +
          " --log-level " + Logging.getGapirLogLevel();
      if (!gapirArgs.get().isEmpty()) {
        gapirFlags += " " + gapirArgs.get();
      }
    }

    if (!gapirFlags.isEmpty()) {
      args.add("-gapir-args");
      args.add(gapirFlags);
    }

    File strings = GapiPaths.get().strings();
    if (strings != null && strings.exists()) {
      args.add("--strings");
      args.add(strings.getAbsolutePath());
    }

    args.add("--gapis-auth-token");
    args.add(authToken);

    if (!disableGapisTimeout.get()) {
      args.add("--idle-timeout");
      args.add(IDLE_TIMEOUT_MS + "ms");
    }

    String adb = GapiPaths.adb(settings);
    if (!adb.isEmpty()) {
      args.add("--adb");
      args.add(adb);
    }

    if (!gapisArgs.get().isEmpty()) {
      args.add("--args");
      args.add(gapisArgs.get());
    }

    GapiPaths.get().addRunfilesFlag(args);

    pb.command(args);
    return null;
  }

  @Override
  protected OutputHandler<Integer> createStdoutHandler() {
    return new LoggingStringHandler<Integer>(LOG, name, false, line -> {
      panicDetector.processLine(line);
      if (!connection.isDone()) {
        Matcher matcher = PORT_PATTERN.matcher(line);
        if (matcher.matches()) {
          int port = Integer.parseInt(matcher.group(1));
          LOG.log(INFO, "Detected gapis startup on port " + port);
          return port;
        }
      }
      return null;
    });
  }

  @Override
  protected OutputHandler<Integer> createStderrHandler() {
    return new LoggingStringHandler<Integer>(LOG, name, true, line -> {
      panicDetector.processLine(line);
      return null;
    });
  }

  @Override
  protected void onExit(int code) {
    super.onExit(code);
    listener.onServerExit(code, panicDetector.hasFoundPanic() ? panicDetector.getPanic() : null);
  }

  public GapisConnection connect() {
    try {
      return connection.get(SERVER_LAUNCH_TIMEOUT_MS, TimeUnit.MILLISECONDS);
    } catch (InterruptedException e) {
      LOG.log(WARNING, "Interrupted while waiting for gapis", e);
    } catch (ExecutionException e) {
      LOG.log(WARNING, "Failed while waiting for gapis", e);
    } catch (TimeoutException e) {
      LOG.log(WARNING, "Timed out waiting for gapis", e);
    }
    return GapisConnection.NOT_CONNECTED;
  }

  /** @return a randomly generated auth-token string. */
  private static String generateAuthToken() {
    SecureRandom rnd = new SecureRandom();
    byte[] bytes = new byte[AUTH_TOKEN_LENGTH * 3 / 4];
    rnd.nextBytes(bytes);
    return Base64.getEncoder().encodeToString(bytes);
  }

  public static interface Listener {
    public static final Listener NULL = new Listener() {
      @Override
      public void onStatus(String message) {
        // Do nothing.
      }

      @Override
      public void onServerExit(int code, String panic) {
        // Do nothing.
      }
    };

    public void onStatus(String message);
    public void onServerExit(int code, String panic);
  }

  private static class PanicDetector {
    private static final int MAX_PANIC_DETAIL_LINES = 256;

    private final StringBuilder panic = new StringBuilder();
    private boolean foundPanic;
    private int count;

    public PanicDetector() {
    }

    public void processLine(String line) {
      if (foundPanic && count < MAX_PANIC_DETAIL_LINES) {
        panic.append(line).append('\n');
        count++;
      } else if (line.startsWith("panic: ") || line.startsWith("fatal error: ")) {
        panic.delete(0, panic.length());
        foundPanic = true;
        count = 0;
        panic.append(line).append('\n');
      }
    }

    public boolean hasFoundPanic() {
      return foundPanic;
    }

    public String getPanic() {
      return panic.toString();
    }
  }
}
