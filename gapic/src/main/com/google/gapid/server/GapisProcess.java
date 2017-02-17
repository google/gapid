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

import static com.google.gapid.util.Logging.logDir;
import static com.google.gapid.util.Logging.logLevel;
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.common.collect.Lists;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;

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

public class GapisProcess extends ChildProcess<Integer> {
  private static final Logger LOG = Logger.getLogger(GapisProcess.class.getName());

  private static final Pattern PORT_PATTERN = Pattern.compile("^Bound on port '(\\d+)'$", 0);

  /** The length in characters of an auth-token */
  private static final int AUTH_TOKEN_LENGTH = 8;

  private static final int SERVER_LAUNCH_TIMEOUT_MS = 10000;
  private static final String SERVER_HOST = "localhost";

  private final ListenableFuture<GapisConnection> connection;
  private final String authToken =  generateAuthToken();

  public GapisProcess() {
    super("gapis");
    connection = Futures.transform(start(), port -> {
      LOG.log(INFO, "Established a new client connection to " + port);
      return GapisConnection.create(SERVER_HOST + ":" + port, authToken, con -> {
        shutdown();
      });
    });
  }

  @Override
  protected Exception prepare(ProcessBuilder pb) {
    if (!GapiPaths.isValid()) {
      LOG.log(WARNING, "Could not find gapis, but needed to start the server.");
      return new Exception("Could not find the gapis executable.");
    }

    List<String> args = Lists.newArrayList();
    args.add(GapiPaths.gapis().getAbsolutePath());

    if (!logDir.get().isEmpty()) {
      args.add("-log-file");
      args.add(logDir.get() + File.separator + "gapis.log");
      args.add("-log-level");
      args.add(logLevel.get().gapisLevel);
      args.add("-gapir-args");
      args.add("--log " + logDir.get() + File.separator + "gapir.log");
    }

    File strings = GapiPaths.strings();
    if (strings.exists()) {
      args.add("--strings");
      args.add(strings.getAbsolutePath());
    }

    args.add("--gapis-auth-token");
    args.add(authToken);

    pb.command(args);
    return null;
  }

  @Override
  protected OutputHandler<Integer> createStdoutHandler() {
    return new LoggingStringHandler<Integer>(LOG, name, false, line -> {
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
}
