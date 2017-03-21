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
import com.google.gapid.util.Flags.Flag;
import com.google.protobuf.MessageOrBuilder;
import com.google.protobuf.Timestamp;

import java.io.File;
import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.Iterator;
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
      int intLevel = level.intValue();
      switch (intLevel) {
        case 1000: return ERROR;
        case  900: return WARNING;
        case  800: case  700: return INFO;
        case  500: case  400: case  300: return DEBUG;
        default:
          if (intLevel > 1000) {
            return ERROR;
          } else if (intLevel > 900) {
            return WARNING;
          } else if (intLevel > 700) {
            return INFO;
          } else {
            return DEBUG;
          }
      }
    }
  }

  /**
   * Listener is the interface implemented by types that want to listen to log messages.
   */
  public interface Listener {
    void onNewMessage(Log.Message message);
  }

  public static final Flag<LogLevel> logLevel = Flags.value(
      "logLevel", LogLevel.INFO, "Logging level [OFF, ERROR, WARNING, INFO, DEBUG, ALL].");
  public static final Flag<String> logDir = Flags.value(
      "logDir", System.getProperty("java.io.tmpdir"), "Directory for logMessage files.");

  private static final Buffer buffer = new Buffer(1000) {{
    setFormatter(new LogFormatter());
    setLevel(Level.ALL);
  }};

  private static final Listener NULL_LISTENER = (m) -> {};

  private static Listener listener = NULL_LISTENER;

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

  public static Iterator<Log.Message> getMessageIterator() {
    return buffer.iterator();
  }

  public static void setListener(Listener listener_) {
    listener = (listener_ == null) ? NULL_LISTENER : listener_;
  }

  /**
   * Adds a {@link Log.Message} to the buffer, bypassing the Java {@link LogManager}.
   * This is used to display log messages from other processes such as GAPIS.
   */
  public static void logMessage(Log.Message message) {
    buffer.add(message);
    listener.onNewMessage(message);
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

  protected static String shorten(String className) {
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
   * MessageIterator is an iterator for the {@link Buffer}.
   * As the {@link Buffer} is a fixed size ring buffer, messages can be discarded if the buffer
   * fills more quickly than it is consumed. This iterator has been implemented to read the next
   * most recently available messages. Some messages may be skipped if the iterator lags behind
   * new messages.
   */
  private static class MessageIterator implements Iterator<Log.Message> {
    private final Buffer buffer;
    private int generation;

    MessageIterator(Buffer buffer) {
      this.buffer = buffer;
    }

    @Override
    public boolean hasNext() {
      synchronized (buffer) {
        return buffer.generation > generation;
      }
    }

    @Override
    public Log.Message next() {
      synchronized (buffer) {
        int remaining = Math.min(buffer.generation - generation, buffer.ring.length);
        if (remaining == 0) {
          return null;
        }
        // Progress the generation so that it is at least the oldest message.
        generation = buffer.generation - remaining;
        Log.Message message = buffer.ring[generation % buffer.ring.length];
        generation++;
        return message;
      }
    }
  }

  /**
   * {@link Handler} that handles and buffers {@link LogRecord}s, converting them to {@link Log.Message}s,
   * and then broadcasting them to listeners.
   */
  private static class Buffer extends Handler {
    private final Log.Message[] ring;
    private final SimpleFormatter simpleFormatter = new SimpleFormatter();
    private int generation;

    public Buffer(int maxSize) {
      this.ring = new Log.Message[maxSize];
    }

    public synchronized void add(Log.Message message) {
      ring[generation % ring.length] = message;
      generation++;
    }

    public Iterator iterator() {
      return new MessageIterator(this);
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
      Logging.logMessage(builder.build());
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
