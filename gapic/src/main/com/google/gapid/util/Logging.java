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
package com.google.gapid.util;

import com.google.gapid.proto.log.Log;
import com.google.gapid.rpclib.schema.Method;
import com.google.gapid.util.Flags.Flag;
import com.google.protobuf.MessageOrBuilder;
import com.google.protobuf.Timestamp;

import java.io.File;
import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.LinkedList;
import java.util.logging.*;

/**
 * Logging setup and utilities.
 */
public class Logging {
  /**
   * Possible logMessage level flag values.
   */
  public enum LogLevel {
    OFF(Level.OFF, Log.Severity.Fatal, "Fatal", "F"),
    ERROR(Level.SEVERE, Log.Severity.Error, "Error", "E"),
    WARNING(Level.WARNING, Log.Severity.Warning, "Warning", "W"),
    INFO(Level.INFO, Log.Severity.Info, "Info", "I"),
    DEBUG(Level.FINE, Log.Severity.Debug, "Debug", "D"),
    ALL(Level.ALL, Log.Severity.Debug, "Debug", "V");

    public final Level level;
    public final Log.Severity severity;
    public final String gapisLevel;
    public final String gapirLevel;

    LogLevel(Level level, Log.Severity severity, String gapisLevel, String gapirLevel) {
      this.level = level;
      this.severity = severity;
      this.gapisLevel = gapisLevel;
      this.gapirLevel = gapirLevel;
    }

    static LogLevel fromLevel(Level level) {
      for (LogLevel logLevel : LogLevel.values()) {
        if (logLevel.level == level) {
          return logLevel;
        }
      }
      return INFO;
    }
  }

  public static final Flag<LogLevel> logLevel = Flags.value(
      "logLevel", LogLevel.INFO, "Logging level [OFF, ERROR, WARNING, INFO, DEBUG, ALL].");
  public static final Flag<String> logDir = Flags.value(
      "logDir", System.getProperty("java.io.tmpdir"), "Directory for logMessage files.");

  private static final Buffer buffer = new Buffer(1000) {{
    setFormatter(new LogFormatter());
    setLevel(Level.ALL);
  }};

  /**
   * Initializes the Java logging system.
   */
  public static void init() {
    LogManager.getLogManager().reset();
    Logger rootLogger = Logger.getLogger("");

    rootLogger.addHandler(buffer);

    ConsoleHandler handler = new ConsoleHandler();
    handler.setFormatter(new LogFormatter());
    handler.setLevel(Level.ALL);
    rootLogger.addHandler(handler);

    if (!logDir.get().isEmpty()) {
      try {
        FileHandler fileHandler = new FileHandler(logDir.get() + File.separator + "gapic.logMessage");
        fileHandler.setFormatter(new LogFormatter());
        fileHandler.setLevel(Level.ALL);
        rootLogger.addHandler(fileHandler);
      } catch (IOException e) {
        // Ignore.
      }
    }

    Logger.getLogger("com.google.gapid").setLevel(logLevel.get().level);
  }

  public static File getLogDir() {
    return logDir.get().isEmpty() ? null : new File(logDir.get());
  }

  public static Log.Message[] getLogMessages() {
    return buffer.getMessages();
  }

  public static void setListener(Runnable listener) {
    buffer.listener = (listener == null) ? () -> { /* empty */ } : listener;
  }

  /**
   * Adds a {@link Log.Message} to the buffer, bypassing the Java {@link LogManager}.
   * This is used to display log messages from other processes such as GAPIS.
   */
  public static void logMessage(Log.Message message) {
    buffer.add(message);
  }

  /**
   * {@link Formatter} implementation for our logs. Contains special handling for logging proto
   * messages in a nice compact format.
   */
  private static class LogFormatter extends Formatter {
    private static final SimpleDateFormat format = new SimpleDateFormat("yyyyMMdd-HHmmssSSS");

    public LogFormatter() {
    }

    @Override
    public String format(LogRecord rec) {
      final StringBuilder buf = new StringBuilder()
          .append(LogLevel.fromLevel(rec.getLevel()).gapirLevel)
          .append(format.format(new Date(rec.getMillis())))
          .append("[").append(shorten(Thread.currentThread().getName()))
          .append("][").append(shorten(rec.getSourceClassName()))
          .append('.').append(rec.getSourceMethodName()).append("] ");
      final String prefix = buf.toString();
      protosToString(rec);
      buf.append(formatMessage(rec)).append('\n');
      Throwable thrown = rec.getThrown();
      if (thrown != null) {
        StringWriter sw = new StringWriter();
        PrintWriter pw = new PrintWriter(sw) {
          @Override
          public void println(Object x) {
            print(prefix);
            super.println(x);
          }

          @Override
          public void println(String x) {
            print(prefix);
            super.println(x);
          }
        };
        thrown.printStackTrace(pw);
        pw.flush();
        buf.append(sw.toString());
      }
      return buf.toString();
    }

    private static void protosToString(LogRecord rec) {
      Object[] params = rec.getParameters();
      if (params == null) {
        return;
      }

      params = params.clone();
      for (int i = 0; params != null && i < params.length; i++) {
        if (params[i] instanceof MessageOrBuilder) {
          params[i] = ProtoDebugTextFormat.shortDebugString((MessageOrBuilder)params[i]);
        }
      }
      rec.setParameters(params);
    }
  }

  private static final String JAVA_PREFIX = "java.";
  private static final String GAPID_PREFIX = "com.google.gapid.";
  private static final String GOOG_PREFIX = "com.google.";

  private static String shorten(String className) {
    if (className.startsWith(JAVA_PREFIX)) {
      return "j." + className.substring(JAVA_PREFIX.length());
    } else if (className.startsWith(GAPID_PREFIX)) {
      return className.substring(GAPID_PREFIX.length());
    } else if (className.startsWith(GOOG_PREFIX)) {
      return "cg." + className.substring(GOOG_PREFIX.length());
    }
    return className;
  }

  /**
   * Similar to {@link java.util.logging.ConsoleHandler}, except that we logMessage to standard output
   * rather than standard error.
   */
  private static class ConsoleHandler extends StreamHandler {
    public ConsoleHandler() {
      setOutputStream(System.out);
    }

    @Override
    public void publish(LogRecord record) {
      super.publish(record);
      flush();
    }

    @Override
    public void close() {
      flush();
    }
  }

  /**
   * {@link Handler} that keeps a number of messages in memory for later retrieval.
   */
  private static class Buffer extends Handler {
    public Runnable listener;
    private final LinkedList<Log.Message> buffer = new LinkedList<>();
    private final SimpleFormatter simpleFormatter = new SimpleFormatter();

    private int maxSize;

    public Buffer(int maxSize) {
      this.maxSize = maxSize;
      listener = () -> { /* empty */ };
    }

    public synchronized Log.Message[] getMessages() {
      return buffer.toArray(new Log.Message[buffer.size()]);
    }

    public synchronized void add(Log.Message message) {
      buffer.addFirst(message);
      while (buffer.size() > maxSize) {
        buffer.removeLast();
      }
      listener.run();
    }

    @Override
    public void publish(LogRecord record) {
      if (!isLoggable(record)) {
        return;
      }

      String thread = shorten(Thread.currentThread().getName());
      String klass = shorten(record.getSourceClassName());
      String method = record.getSourceMethodName();

      long seconds = record.getMillis() / 1000;
      int millis = (int) (record.getMillis() - seconds * 1000);

      Log.Message.Builder builder = Log.Message.newBuilder()
          .setText(simpleFormatter.formatMessage(record))
          .setProcess("gapic")
          .setSeverity(LogLevel.fromLevel(record.getLevel()).severity)
          .setTag(shorten(record.getLoggerName()))
          .setTime(Timestamp.newBuilder().setSeconds(seconds).setNanos(millis * 1000000));

      builder.addValues(Log.Value.newBuilder().setName("thread").setValue(Pods.pod(thread)));
      builder.addValues(Log.Value.newBuilder().setName("class").setValue(Pods.pod(klass)));
      builder.addValues(Log.Value.newBuilder().setName("method").setValue(Pods.pod(method)));

      Throwable thrown = record.getThrown();
      if (thrown != null) {
        for (StackTraceElement el : thrown.getStackTrace()) {
          builder.addCallstack(Log.SourceLocation.newBuilder()
              .setFile(el.getFileName())
              .setLine(el.getLineNumber())
              .build());
        }
      }
      add(builder.build());
    }

    @Override
    public void flush() {
      // Do nothing.
    }

    @Override
    public void close() {
      // Do nothing.
    }
  }
}
