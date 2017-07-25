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

import static java.nio.charset.StandardCharsets.UTF_8;
import static java.util.logging.Level.INFO;
import static java.util.logging.Level.WARNING;

import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.SettableFuture;
import com.google.gapid.models.Settings;

import java.io.BufferedReader;
import java.io.Closeable;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.logging.Logger;

/**
 * Manages the invocation and bookkeeping of a child process.
 */
public abstract class ChildProcess<T> {
  private static final Logger LOG = Logger.getLogger(ChildProcess.class.getName());

  protected final String name;
  protected final Settings settings;
  private Thread serverThread;
  protected Process process;

  ChildProcess(String name, Settings settings) {
    this.name = name;
    this.settings = settings;
  }

  protected abstract Exception prepare(ProcessBuilder pb);

  public boolean isRunning() {
    return serverThread != null && serverThread.isAlive();
  }

  public ListenableFuture<T> start() {
    final ProcessBuilder pb = new ProcessBuilder();
    // Use the base directory as the working directory for the server.
    pb.directory(GapiPaths.base());
    Exception prepareError = prepare(pb);
    if (prepareError != null) {
      return Futures.immediateFailedFuture(prepareError);
    }

    final SettableFuture<T> result = SettableFuture.create();
    serverThread = new Thread(ChildProcess.class.getName() + "-" + name) {
      @Override
      public void run() {
        runProcess(result, pb);
      }
    };
    serverThread.start();
    return result;
  }

  protected void runProcess(final SettableFuture<T> result, final ProcessBuilder pb) {
    try {
      // This will throw IOException if the executable is not found.
      LOG.log(INFO, "Starting " + name + " as " + pb.command());
      process = pb.start();
    } catch (IOException e) {
      LOG.log(WARNING, "IO Error running process", e);
      result.setException(e);
      return;
    }

    int exitCode = -1;
    try (OutputHandler<T> stdout = createStdoutHandler();
        OutputHandler<T> stderr = createStderrHandler()) {
      stdout.start(process.getInputStream(), result);
      stderr.start(process.getErrorStream(), result);
      exitCode = process.waitFor();
      stderr.join();
      stdout.join();
      stderr.finish(result);
      stdout.finish(result);
      if (!result.isDone()) {
        result.setException(new EarlyExitException(name, exitCode));
      }
    } catch (InterruptedException e) {
      LOG.log(INFO, "Killing " + name);
      result.setException(e);
      process.destroy();
    } finally {
      onExit(exitCode);
    }
  }

  protected void onExit(int code) {
    if (code != 0) {
      LOG.log(WARNING, "The " + name + " process exited with a non-zero exit value: " + code);
    } else {
      LOG.log(INFO, name + " exited cleanly");
    }
    shutdown();
  }

  protected OutputHandler<T> createStdoutHandler() {
    return new LoggingStringHandler<T>(LOG, name, false, null);
  }

  protected OutputHandler<T> createStderrHandler() {
    return new LoggingStringHandler<T>(LOG, name, true, null);
  }

  public void shutdown() {
    LOG.log(INFO, "Shutting down " + name);
    serverThread.interrupt();
  }

  /**
   * Exception used to signal that the process has exited before a result has been detected
   * from the process output.
   */
  public static class EarlyExitException extends Exception {
    public final int exitCode;

    public EarlyExitException(String name, int exitCode) {
      super(name + " has exited with exit code " + exitCode);
      this.exitCode = exitCode;
    }
  }

  /**
   * Handler for the child process' standard and error output. The handler is responsible for
   * producing the process result returned as a future from {@link ChildProcess#start()}. This is
   * typically done by parsing the output produces by the process.
   */
  protected static abstract class OutputHandler<T> implements Closeable {
    private Thread thread;

    protected abstract void run(InputStream in, SettableFuture<T> result);

    public void start(InputStream in, SettableFuture<T> result) {
      close();
      thread = new Thread(() -> run(in, result), getClass().getName());
      thread.start();
    }

    @SuppressWarnings("unused")
    public void finish(SettableFuture<T> result) throws InterruptedException {
      // Do nothing by default.
    }

    public void join() throws InterruptedException {
      if (thread != null) {
        thread.join(5000);
      }
    }

    @Override
    public void close() {
      if (thread != null) {
        thread.interrupt();
        thread = null;
      }
    }
  }

  /**
   * String line based {@link ChildProcess.OutputHandler}.
   */
  protected static class StringHandler<T> extends OutputHandler<T> {
    private final Parser<T> parser;

    public static interface Parser<T> {
      public T parse(String line) throws IOException;
    }

    public StringHandler(Parser<T> parser) {
      this.parser = parser;
    }

    @Override
    public void run(InputStream in, SettableFuture<T> result) {
      try (BufferedReader reader = new BufferedReader(new InputStreamReader(in, UTF_8))) {
        for (String line; (line = reader.readLine()) != null; ) {
          T object = parser.parse(line);
          if (object != null) {
            result.set(object);
          }
        }
      } catch (IOException e) {
        result.setException(e);
      }
    }
  }

  /**
   * {@link ChildProcess.OutputHandler} that forward the output to the log.
   */
  protected static class LoggingStringHandler<T> extends StringHandler<T> {
    public LoggingStringHandler(Logger logger, String name, boolean warn, Parser<T> parser) {
      super(line -> {
        if (warn) {
          logger.log(WARNING, name + ": " + line);
        } else {
          logger.log(INFO, name + ": " + line);
        }
        if (parser != null) {
          return parser.parse(line);
        }
        return null;
      });
    }
  }

  /**
   * {@link ChildProcess.OutputHandler} for binary (non text) output.
   */
  protected static class BinaryHandler<T> extends OutputHandler<T> {
    private final Parser<T> parser;

    public static interface Parser<T> {
      public T parse(InputStream in) throws IOException;
    }

    public BinaryHandler(Parser<T> parser) {
      this.parser = parser;
    }

    @Override
    public void run(InputStream in, SettableFuture<T> result) {
      try {
        T object = parser.parse(in);
        if (object != null) {
          result.set(object);
        }
      } catch (IOException e) {
        result.setException(e);
      }
    }
  }
}
