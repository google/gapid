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

import static com.google.gapid.util.GapidVersion.GAPID_VERSION;

import com.google.common.base.Preconditions;
import com.google.common.collect.Lists;
import com.google.common.collect.Maps;

import java.io.PrintStream;
import java.util.List;
import java.util.Map;
import java.util.regex.Matcher;
import java.util.regex.Pattern;

/**
 * Command line flag definition and parsing utilities.
 */
public class Flags {
  private static final Pattern FLAG_PATTERN =
      Pattern.compile("-+([^=\\s]+)(?:=(.*))?", Pattern.MULTILINE | Pattern.DOTALL);
  private static final int HELP_PRINT_MARGIN = 20;
  private static final int HELP_LINE_LENGTH = 80;

  public static final Flag<Boolean> help = value("help", false, "Print help information.");
  public static final Flag<Boolean> fullHelp =
      value("fullhelp", false, "Print help information incuding hidden flags.", true);
  public static final Flag<Boolean> version = value("version", false, "Print AGI version.");

  private static boolean initialized = false;

  private Flags() {
  }

  /**
   * Parses the given command line arguments and initializes the given flags.
   */
  public static synchronized String[] initFlags(Flag<?>[] flags, String[] args) {
    if (initialized) {
      throw new RuntimeException("Flags already initalized");
    }
    initialized = true;

    Map<String, Flag<?>> flagMap = Maps.newHashMap();
    for (Flag<?> flag : flags) {
      if (flagMap.put(flag.getName(), flag) != null) {
        throw new RuntimeException("Duplicate flag: " + flag.getName());
      }
      if (isBooleanFlag(flag) && flagMap.put("no" + flag.getName(), flag) != null) {
        throw new RuntimeException("Duplicate flag: no" + flag.getName());
      }
    }

    List<String> result = Lists.newArrayList();
    for (int i = 0; i < args.length; i++) {
      if (!args[i].startsWith("-")) {
        result.add(args[i]);
        continue;
      } else if ("--".equals(args[i])) {
        for (int j = i + 1; j < args.length; j++) {
          result.add(args[j]);
        }
        break;
      } else if (OS.isMac && args[i].startsWith("-psn_")) {
        // Ignore the process number auto-flag.
        continue;
      }

      Matcher m = FLAG_PATTERN.matcher(args[i]);
      if (!m.matches()) {
        throw new InvalidFlagException(args[i]);
      }
      String name = m.group(1);
      String value = m.group(2);

      Flag<?> flag = flagMap.get(name);
      if (flag == null) {
        throw new InvalidFlagException(name);
      }

      if (value == null) {
        if (isBooleanFlag(flag)) {
          if (i + 1 < args.length && !args[i + 1].startsWith("-")) {
            value = args[++i];
          } else {
            value = String.valueOf(!name.startsWith("no"));
          }
        } else if (++i >= args.length) {
          throw new InvalidFlagException(name + " is missing its argument");
        } else {
          value = args[i];
        }
      } else if (isBooleanFlag(flag) && name.startsWith("no")) {
        throw new InvalidFlagException(args[i]);
      }

      flag.setValue(value);
    }

    if (help.get()) {
      printHelp(System.out, flags, false);
      System.exit(0);
    } else if (fullHelp.get()) {
      printHelp(System.out, flags, true);
      System.exit(0);
    } else if (version.get()) {
      printVersion(System.out);
      System.exit(0);
    }

    return result.toArray(new String[result.size()]);
  }

  private static boolean isBooleanFlag(Flag<?> flag) {
    return flag.get() instanceof Boolean;
  }

  private static void printVersion(PrintStream out) {
    out.println("AGI version " + GAPID_VERSION);
  }

  private static void printHelp(PrintStream out, Flag<?>[] flags, boolean full) {
    printVersion(out);
    out.println("Usage:");
    StringBuilder line = new StringBuilder();
    for (Flag<?> flag : flags) {
      if (!full && flag.isHidden()) {
        continue;
      }

      line.setLength(0);
      line.append(" --").append(flag.getName());
      boolean first = true;
      if (line.length() >= HELP_PRINT_MARGIN) {
        out.println(line);
        line.setLength(0);
        first = false;
      }
      String description = flag.getDescription() + " (Default: " + flag.getDefault() + ")";
      while (description.length() > 0) {
        while (line.length() < HELP_PRINT_MARGIN) {
          line.append(' ');
        }
        out.print(line);

        if (description.length() <= HELP_LINE_LENGTH - HELP_PRINT_MARGIN) {
          out.println(description);
          break;
        }

        int p = description.lastIndexOf(' ', HELP_LINE_LENGTH - HELP_PRINT_MARGIN);
        if (p == -1) {
          p = HELP_LINE_LENGTH - HELP_PRINT_MARGIN;
        }
        out.println(description.substring(0, p));
        description = description.substring(p + 1);

        if (first) {
          line.setLength(0);
        }
      }
    }
  }

  /**
   * Command line flag definition.
   */
  public static class Flag<T> {
    private final String name;
    private final Parser<T> parser;
    private final T deflt;
    private final String description;
    private final boolean hidden;
    private T value;
    private boolean specified;

    Flag(String name, Parser<T> parser, T deflt, String description, boolean hidden) {
      Preconditions.checkNotNull(deflt);
      this.name = name;
      this.parser = parser;
      this.value = this.deflt = deflt;
      this.description = description;
      this.hidden = hidden;
      this.specified = false;
    }

    public String getName() {
      return name;
    }

    public T get() {
      return value;
    }

    public T getDefault() {
      return deflt;
    }

    public String getDescription() {
      return description;
    }

    public boolean isHidden() {
      return hidden;
    }

    public boolean isSpecified() {
      return specified;
    }

    void setValue(String value) {
      this.value = parser.parse(value);
      this.specified = true;
    }
  }

  public static Flag<String> value(String name, String dflt, String description) {
    return value(name, dflt, description, false);
  }

  public static Flag<String> value(String name, String dflt, String description, boolean hidden) {
    return new Flag<String>(name, new Parser<String>() {
      @Override
      public String parse(String value) {
        return value;
      }
    }, dflt, description, hidden);
  }

  public static Flag<Integer> value(String name, int dflt, String description) {
    return value(name, dflt, description, false);
  }

  public static Flag<Integer> value(String name, int dflt, String description, boolean hidden) {
    return new Flag<Integer>(name, new Parser<Integer>() {
      @Override
      public Integer parse(String value) {
        try {
          return Integer.parseInt(value);
        } catch (NumberFormatException e) {
          throw new InvalidFlagException(value, e);
        }
      }
    }, dflt, description, hidden);
  }

  public static Flag<Double> value(String name, double dflt, String description) {
    return value(name, dflt, description, false);
  }

  public static Flag<Double> value(String name, double dflt, String description, boolean hidden) {
    return new Flag<Double>(name, new Parser<Double>() {
      @Override
      public Double parse(String value) {
        try {
          return Double.parseDouble(value);
        } catch (NumberFormatException e) {
          throw new InvalidFlagException(value, e);
        }
      }
    }, dflt, description, hidden);
  }

  public static Flag<Boolean> value(String name, boolean dflt, String description) {
    return value(name, dflt, description, false);
  }

  public static Flag<Boolean> value(String name, boolean dflt, String description, boolean hidden) {
    return new Flag<Boolean>(name, new Parser<Boolean>() {
      @Override
      public Boolean parse(String value) {
        return "true".equalsIgnoreCase(value);
      }
    }, dflt, description, hidden);
  }

  public static <T extends Enum<T>> Flag<T> value(String name, final T dflt, String description) {
    return value(name, dflt, description, false);
  }

  public static <T extends Enum<T>> Flag<T> value(
      String name, final T dflt, String description, boolean hidden) {
    return new Flag<T>(name, new Parser<T>() {
      @Override
      public T parse(String value) {
        try {
          return Enum.valueOf(dflt.getDeclaringClass(), value.toUpperCase());
        } catch (IllegalArgumentException e) {
          throw new InvalidFlagException(value, e);
        }
      }
    }, dflt, description, hidden);
  }

  /**
   * {@link RuntimeException} thrown when flag parsing failed.
   */
  public static class InvalidFlagException extends RuntimeException {
    public InvalidFlagException(String flag) {
      super("Invalid flag: " + flag);
    }

    public InvalidFlagException(String flag, Throwable cause) {
      super("Invalid flag: " + flag, cause);
    }
  }

  /**
   * Parses flag values.
   */
  private static interface Parser<T> {
    public T parse(String value);
  }
}
