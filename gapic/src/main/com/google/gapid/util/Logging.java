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

import com.google.common.collect.Sets;
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
import java.util.Set;
import java.util.function.Consumer;
import java.util.logging.FileHandler;
import java.util.logging.Formatter;
import java.util.logging.Handler;
import java.util.logging.Level;
import java.util.logging.LogManager;
import java.util.logging.LogRecord;
import java.util.logging.Logger;
import java.util.logging.SimpleFormatter;
import java.util.logging.StreamHandler;

import io.grpc.Status;

/**
 * Logging setup and utilities.
 */
public class Logging {
  /**
   * Possible log level flag values.
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
      "log-level", LogLevel.INFO, "Logging level [OFF, ERROR, WARNING, INFO, DEBUG, ALL].");
  public static final Flag<String> logDir = Flags.value(
      "log-dir", System.getProperty("java.io.tmpdir"), "Directory for log files.");
  // Actual default is to use logLevel's value.
  public static final Flag<LogLevel> gapisLogLevel = Flags.value(
      "gapis-log-level", LogLevel.INFO, "Gapis log level [OFF, ERROR, WARNING, INFO, DEBUG, ALL].");
  // Actual default is to use logLevel's value.
  public static final Flag<LogLevel> gapirLogLevel = Flags.value(
      "gapir-log-level", LogLevel.INFO, "Gapir log level [OFF, ERROR, WARNING, INFO, DEBUG, ALL].");

  private static final int BUFFER_SIZE = 1000;
  private static final long THROTTLE_TIME_MS = 5000;

  private static final RingBuffer buffer = new RingBuffer(BUFFER_SIZE);

  private static final Listener NULL_LISTENER = (m) -> { /* do nothing */ };

  private static Listener listener = NULL_LISTENER;
  private static long timeOfLastThrottledMessage = 0;

  /**
   * Initializes the Java logging system.
   */
  public static void init() {
    LogManager.getLogManager().reset();
    Logger rootLogger = Logger.getLogger("");

    rootLogger.addHandler(new LogToMessageHandler() {{
      setFormatter(new LogFormatter());
      setLevel(Level.ALL);
    }});

    ConsoleHandler handler = new ConsoleHandler();
    handler.setFormatter(new LogFormatter());
    handler.setLevel(Level.ALL);
    rootLogger.addHandler(handler);

    initFileHandler(rootLogger);

    Logger.getLogger("com.google.gapid").setLevel(logLevel.get().level);
  }

  private static void initFileHandler(Logger rootLogger) {
    if (!logDir.get().isEmpty()) {
      File dir = new File(logDir.get());
      dir.mkdirs();

      File file = new File(dir, "gapic.log");
      for (int i = 0; i < 10; i++) {
        if (!file.exists() || file.canWrite()) {
          try {
            FileHandler fileHandler = new FileHandler(file.getAbsolutePath());
            fileHandler.setFormatter(new LogFormatter());
            fileHandler.setLevel(Level.ALL);
            rootLogger.addHandler(fileHandler);
            return;
          } catch (IOException e) {
            System.err.println("Failed to create log file " + file + ":");
            e.printStackTrace(System.err);
          }
        }

        // Try a different name next.
        file = new File(dir, "gapic-" + i + ".log");
      }

      // Give up.
      System.err.println("Failed to create log file in " + logDir.get());
    }
  }

  public static File getLogDir() {
    return logDir.get().isEmpty() ? null : new File(logDir.get());
  }

  public static String getGapisLogLevel() {
    return (gapisLogLevel.isSpecified() ? gapisLogLevel.get() : logLevel.get()).gapisLevel;
  }

  public static String getGapirLogLevel() {
    return (gapirLogLevel.isSpecified() ? gapirLogLevel.get() : logLevel.get()).gapirLevel;
  }

  public static MessageIterator getMessageIterator() {
    return buffer.iterator();
  }

  public static void setListener(Listener newListener) {
    listener = (newListener == null) ? NULL_LISTENER : newListener;
  }

  /**
   * Adds a {@link com.google.gapid.proto.log.Log.Message} to the buffer, bypassing the Java
   * {@link LogManager}.
   * This is used to display log messages from other processes such as GAPIS.
   */
  public static void logMessage(Log.Message message) {
    buffer.add(message);
    listener.onNewMessage(message);
  }

  /**
   * Throttles logging of certain RPC errors. This is primarily used to prevent the logs from
   * exploding when GAPIS dies, or the connection goes away.
   */
  public static void throttleLogRpcError(Logger log, String message, Throwable exception) {
    Status status = Status.fromThrowable(exception);
    if (status.getCode() == Status.Code.UNAVAILABLE) {
      synchronized (Logging.class) {
        long now = System.currentTimeMillis();
        if ((now - timeOfLastThrottledMessage) < THROTTLE_TIME_MS) {
          return;
        }
        timeOfLastThrottledMessage = now;
      }
    }
    log.log(Level.WARNING, message, exception);
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
          .append(getTag(rec)).append(' ');
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
  }

  protected static String getTag(LogRecord record) {
    return "[" + shorten(Thread.currentThread().getName()) + "][" +
      shorten(record.getSourceClassName()) + "." +
      record.getSourceMethodName() + "]";
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

  protected static void protosToString(LogRecord rec) {
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

  /**
   * Similar to {@link java.util.logging.ConsoleHandler}, except that we log to standard output
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
   * MessageIterator is an iterator for the {@link RingBuffer}.
   * As the {@link RingBuffer} is a fixed size ring buffer, messages can be discarded if the buffer
   * fills more quickly than it is consumed. This iterator has been implemented to read the next
   * most recently available messages. Some messages may be skipped if the iterator lags behind
   * new messages.
   */
  public static class MessageIterator {
    private final RingBuffer target;
    private int generation;

    MessageIterator(RingBuffer target) {
      this.target = target;
    }

    public Log.Message next() {
      synchronized (target) {
        int remaining = Math.min(target.generation - generation, target.ring.length);
        if (remaining == 0) {
          return null;
        }
        // Progress the generation so that it is at least the oldest message.
        generation = target.generation - remaining;
        Log.Message message = target.ring[generation % target.ring.length];
        generation++;
        return message;
      }
    }
  }

  /**
   * A ring buffer of {@link LogRecord}s.
   */
  protected static class RingBuffer {
    protected final Log.Message[] ring;
    protected int generation;

    public RingBuffer(int maxSize) {
      this.ring = new Log.Message[maxSize];
    }

    public synchronized void add(Log.Message message) {
      ring[generation % ring.length] = message;
      generation++;
    }

    public MessageIterator iterator() {
      return new MessageIterator(this);
    }
  }

  /**
   * {@link Handler} that handles {@link LogRecord}s converting them to
   * {@link com.google.gapid.proto.log.Log.Message}s and then broadcasting them to listeners.
   */
  protected static class LogToMessageHandler extends Handler {
    private final SimpleFormatter simpleFormatter = new SimpleFormatter();

    @Override
    public void publish(LogRecord record) {
      if (!isLoggable(record)) {
        return;
      }

      protosToString(record);

      long seconds = record.getMillis() / 1000;
      int millis = (int) (record.getMillis() - seconds * 1000);

      Log.Message.Builder builder = Log.Message.newBuilder()
          .setText(simpleFormatter.formatMessage(record))
          .setProcess("gapic")
          .setSeverity(LogLevel.fromLevel(record.getLevel()).severity)
          .setTag(getTag(record))
          .setTime(Timestamp.newBuilder().setSeconds(seconds).setNanos(millis * 1000000));

      Throwable thrown = record.getThrown();
      if (thrown != null) {
        exceptionToCause(thrown, cause -> builder.addCause(cause));
      }
      Logging.logMessage(builder.build());
    }

    private static void exceptionToCause(Throwable thrown, Consumer<Log.Cause.Builder> addCause) {
      Log.Cause.Builder result = Log.Cause.newBuilder()
          .setMessage(thrown.toString());
      StackTraceElement[] trace = thrown.getStackTrace();
      for (StackTraceElement el : trace) {
        result.addCallstack(toSourceLocation(el));
      }
      addCause.accept(result);

      Set<Throwable> seen = Sets.newHashSet(thrown);
      for (Throwable suppressed : thrown.getSuppressed()) {
        enclosedExceptionToCause(suppressed, trace, "Suppressed: ", seen, addCause);
      }
      Throwable cause = thrown.getCause();
      if (cause != null) {
        enclosedExceptionToCause(cause, trace, "Caused by: ", seen, addCause);
      }
    }

    private static void enclosedExceptionToCause(Throwable thrown,
        StackTraceElement[] enclosingTrace, String title, Set<Throwable>seen,
        Consumer<Log.Cause.Builder> addCause) {
      if (!seen.add(thrown)) {
        addCause.accept(Log.Cause.newBuilder().setMessage(title + "Circular reference: " + thrown));
        return;
      }

      Log.Cause.Builder result = Log.Cause.newBuilder()
          .setMessage(title + thrown.toString());

      StackTraceElement[] trace = thrown.getStackTrace();
      int ours = trace.length - 1;
      int outer = enclosingTrace.length - 1;
      while (ours >= 0 && outer >=0 && trace[ours].equals(enclosingTrace[outer])) {
          ours--;
          outer--;
      }
      for (int i = 0; i <= ours; i++) {
        result.addCallstack(toSourceLocation(trace[i]));
      }
      addCause.accept(result);

      for (Throwable suppressed : thrown.getSuppressed()) {
        enclosedExceptionToCause(suppressed, trace, "Suppressed: ", seen, addCause);
      }
      Throwable cause = thrown.getCause();
      if (cause != null) {
        enclosedExceptionToCause(cause, trace, "Caused by: ", seen, addCause);
      }
    }

    private static Log.SourceLocation.Builder toSourceLocation(StackTraceElement el) {
      String file = el.getFileName();
      String method = el.getClassName() + "." + el.getMethodName() +
          (el.isNativeMethod() ? "(Native method)" : "");
      return Log.SourceLocation.newBuilder()
          .setFile(file == null ? "" : file)
          .setLine(Math.max(0, el.getLineNumber()))
          .setMethod(method);
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
