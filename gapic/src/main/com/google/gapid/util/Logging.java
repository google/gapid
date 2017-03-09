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

import com.google.gapid.util.Flags.Flag;
import com.google.protobuf.MessageOrBuilder;

import java.io.File;
import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.logging.FileHandler;
import java.util.logging.Formatter;
import java.util.logging.Level;
import java.util.logging.LogManager;
import java.util.logging.LogRecord;
import java.util.logging.Logger;
import java.util.logging.StreamHandler;

/**
 * Logging setup and utilities.
 */
public class Logging {
  /**
   * Possible log level flag values.
   */
  public static enum LogLevel {
    OFF(Level.OFF, "Emergency", "F"), ERROR(Level.SEVERE, "Error", "E"),
    WARNING(Level.WARNING, "Warning", "W"), INFO(Level.INFO, "Info", "I"),
    DEBUG(Level.FINE, "Debug", "D"), ALL(Level.ALL, "Debug", "V");

    public final Level level;
    public final String gapisLevel;
    public final String gapirLevel;

    private LogLevel(Level level, String gapisLevel, String gapirLevel) {
      this.level = level;
      this.gapisLevel = gapisLevel;
      this.gapirLevel = gapirLevel;
    }
  }

  public static final Flag<LogLevel> logLevel = Flags.value(
      "logLevel", LogLevel.INFO, "Logging level [OFF, ERROR, WARNING, INFO, DEBUG, ALL].");
  public static final Flag<String> logDir = Flags.value(
      "logDir", System.getProperty("java.io.tmpdir"), "Directory for log files.");

  /**
   * Initializes the Java logging system.
   */
  public static void init() {
    LogManager.getLogManager().reset();

    ConsoleHandler handler = new ConsoleHandler();
    handler.setFormatter(new LogFormatter());
    handler.setLevel(Level.ALL);
    Logger.getLogger("").addHandler(handler);

    if (!logDir.get().isEmpty()) {
      try {
        FileHandler fileHandler = new FileHandler(logDir.get() + File.separator + "gapic.log");
        fileHandler.setFormatter(new LogFormatter());
        fileHandler.setLevel(Level.ALL);
        Logger.getLogger("").addHandler(fileHandler);
      } catch (IOException e) {
        // Ignore.
      }
    }

    Logger.getLogger("com.google.gapid").setLevel(logLevel.get().level);
  }

  public static File getLogDir() {
    return logDir.get().isEmpty() ? null : new File(logDir.get());
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
        .append(getLogLevel(rec.getLevel().intValue()))
        .append(format.format(new Date(rec.getMillis())))
        .append("[").append(getClassName(Thread.currentThread().getName()))
        .append("][").append(getClassName(rec.getSourceClassName()))
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

    private static final String JAVA_PREFIX = "java.";
    private static final String GAPID_PREFIX = "com.google.gapid.";
    private static final String GOOG_PREFIX = "com.google.";

    private static String getClassName(String className) {
      if (className.startsWith(JAVA_PREFIX)) {
        return "j." + className.substring(JAVA_PREFIX.length());
      } else if (className.startsWith(GAPID_PREFIX)) {
        return className.substring(GAPID_PREFIX.length());
      } else if (className.startsWith(GOOG_PREFIX)) {
        return "cg." + className.substring(GOOG_PREFIX.length());
      }
      return className;
    }

    private static char getLogLevel(int level) {
      switch (level) {
        case 1000: return 'E';
        case  900: return 'W';
        case  800: case  700: return 'I';
        case  500: case  400: case  300: return 'D';
        default:
          if (level > 1000) {
            return 'E';
          } else if (level > 900) {
            return 'W';
          } else if (level > 700) {
            return 'I';
          } else {
            return 'D';
          }
      }
    }
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
}
