/*
 * Copyright (C) 2020 Google Inc.
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
package com.google.gapid.perfetto;

import static java.lang.Math.pow;

import java.util.Arrays;
import java.util.stream.Collectors;

public class Unit {
  private static final String[] BIT_NAMES = { "bits", "Kbit", "Mbit", "Gbit", "Tbit", "Pbit" };
  private static final String[] BYTE_NAMES = { "bytes", "KB", "MB", "GB", "TB", "PB" };
  private static final String[] HERTZ_NAMES = { "Hz", "KHz", "MHz", "GHz", "THz", "PHz" };
  private static final String[] WATT_NAMES = { "mW", "W", "KW" };

  public static final Unit NONE = new Unit("", Formatter.DEFAULT);

  public static final Unit BIT = new Unit("bits", new Formatter.Base2(BYTE_NAMES, 0));
  public static final Unit KILO_BIT = new Unit("Kbit", new Formatter.Base2(BIT_NAMES, 1));
  public static final Unit MEGA_BIT = new Unit("Mbit", new Formatter.Base2(BIT_NAMES, 2));
  public static final Unit GIGA_BIT = new Unit("Gbit", new Formatter.Base2(BIT_NAMES, 3));
  public static final Unit TERA_BIT = new Unit("Tbit", new Formatter.Base2(BIT_NAMES, 4));
  public static final Unit PETA_BIT = new Unit("Pbit", new Formatter.Base2(BIT_NAMES, 5));

  public static final Unit BYTE = new Unit("bytes", new Formatter.Base2(BYTE_NAMES, 0));
  public static final Unit KILO_BYTE = new Unit("KB", new Formatter.Base2(BYTE_NAMES, 1));
  public static final Unit MEGA_BYTE = new Unit("MB", new Formatter.Base2(BYTE_NAMES, 2));
  public static final Unit GIGA_BYTE = new Unit("GB", new Formatter.Base2(BYTE_NAMES, 3));
  public static final Unit TERA_BYTE = new Unit("TB", new Formatter.Base2(BYTE_NAMES, 4));
  public static final Unit PETA_BYTE = new Unit("PB", new Formatter.Base2(BYTE_NAMES, 5));

  public static final Unit HERTZ = new Unit("Hz", new Formatter.Base10(HERTZ_NAMES, 0));
  public static final Unit KILO_HERTZ = new Unit("KHz", new Formatter.Base10(HERTZ_NAMES, 1));
  public static final Unit MEGA_HERTZ = new Unit("MHz", new Formatter.Base10(HERTZ_NAMES, 2));
  public static final Unit GIGA_HERTZ = new Unit("GHz", new Formatter.Base10(HERTZ_NAMES, 3));
  public static final Unit TERA_HERTZ = new Unit("THz", new Formatter.Base10(HERTZ_NAMES, 4));
  public static final Unit PETA_HERTZ = new Unit("PHz", new Formatter.Base10(HERTZ_NAMES, 5));

  public static final Unit NANO_SECOND = new Unit("ns", Formatter.Time.nanoseconds());
  public static final Unit MICRO_SECOND = new Unit("us", Formatter.Time.microseconds());
  public static final Unit MILLI_SECOND = new Unit("ms", Formatter.Time.milliseconds());
  public static final Unit SECOND = new Unit("s", Formatter.Time.seconds());
  public static final Unit MINUTE = new Unit("m", Formatter.Time.minutes());
  public static final Unit HOUR = new Unit("h", Formatter.Time.hours());

  public static final Unit VERTEX = new Unit("verts", new Formatter.Simple("verts"));
  public static final Unit PIXEL = new Unit("px", new Formatter.Simple("px"));
  public static final Unit TRIANGLE = new Unit("tris", new Formatter.Simple("tris"));
  public static final Unit PRIMITIVE = new Unit("prims", new Formatter.Simple("prims"));
  public static final Unit FRAGMENT = new Unit("frags", new Formatter.Simple("frags"));

  public static final Unit MILLI_WATT = new Unit("mW", new Formatter.Base10(WATT_NAMES, 0));
  public static final Unit WATT = new Unit("W", new Formatter.Base10(WATT_NAMES, 1));
  public static final Unit KILO_WATT = new Unit("KW", new Formatter.Base10(WATT_NAMES, 2));

  public static final Unit JOULE = new Unit("J", new Formatter.Simple("J"));
  public static final Unit VOLT = new Unit("V", new Formatter.Simple("V"));
  public static final Unit AMPERE = new Unit("A", new Formatter.Simple("A"));

  public static final Unit CELSIUS = new Unit("C", new Formatter.Simple("C"));
  public static final Unit FAHRENHEIT = new Unit("F", new Formatter.Simple("F"));
  public static final Unit KELVIN = new Unit("K", new Formatter.Simple("K"));

  public static final Unit PERCENT = new Unit("%", new Formatter.Percent());

  public static final Unit INSTRUCTION = new Unit("instr", new Formatter.Simple("instr"));

  private static final double MIN_DOUBLE_AS_LONG = 100_000.0;
  private static final double MAX_DOUBLE_AS_LONG = 9.2233720368547748E18;

  public final String name;
  private final Formatter formatter;

  private Unit(String name, Formatter formatter) {
    this.name = name;
    this.formatter = formatter;
  }

  public static Unit combined(Unit numer, Unit denom) {
    if (numer == NONE && denom == NONE) {
      return NONE;
    } else if (numer == NONE) {
      return new Unit("per " + denom.name, Formatter.DEFAULT);
    } else if (denom == NONE) {
      return numer;
    }

    return new Unit(numer.name + "/" + denom.name, new Formatter() {
      @Override
      public String format(long value) {
        return numer.format(value) + "/" + denom.name;
      }

      @Override
      public String format(double value) {
        return numer.format(value) + "/" + denom.name;
      }

      @Override
      public Formatter withFixedScale(double representativeValue) {
        Unit fixed = numer.withFixedScale(representativeValue);
        return new Formatter() {
          @Override
          public String format(long value) {
            return fixed.format(value) + "/" + denom.name;
          }

          @Override
          public String format(double value) {
            return fixed.format(value) + "/" + denom.name;
          }
        };
      }
    });
  }

  public static Unit combined(Unit[] numer, Unit[] denom) {
    if (numer.length == 0 && denom.length == 0) {
      return NONE;
    } else if (numer.length == 0 && denom.length == 1) {
      return combined(NONE, denom[0]);
    } else if (numer.length == 1 && denom.length == 0) {
      return numer[0];
    } else if (numer.length == 1 && denom.length == 1) {
      return combined(numer[0], denom[0]);
    }

    String n = Arrays.stream(numer)
        .map(u -> u.name)
        .filter(name -> !name.isEmpty())
        .collect(Collectors.joining("⋅"));
    String d = Arrays.stream(denom)
        .map(u -> u.name)
        .filter(name -> !name.isEmpty())
        .collect(Collectors.joining("⋅"));
    return new Unit((n.isEmpty()) ? "per " + d : d.isEmpty() ? n : n + "/" + d, Formatter.DEFAULT);
  }

  public String format(long value) {
    if (value == Long.MIN_VALUE) {
      return "-" + formatter.format(Long.MAX_VALUE);
    } else if (value < 0) {
      return "-" + formatter.format(-value);
    } else {
      return formatter.format(value);
    }
  }

  public String format(double value) {
    if (Double.isNaN(value) || Double.isInfinite(value)) {
      return (Double.toString(value) + " " + name).trim();
    } else if (value == 0) {
      return formatter.format(0);
    }

    double abs = Math.abs(value);
    if (abs >= MIN_DOUBLE_AS_LONG && abs <= MAX_DOUBLE_AS_LONG) {
      return (value < 0 ? "-" : "") + formatter.format(Math.round(abs));
    } else {
      return (value < 0 ? "-" : "") + formatter.format(abs);
    }
  }

  public Unit withFixedScale(double representativeValue) {
    return new Unit(name, formatter.withFixedScale(representativeValue));
  }

  public static String bytesToString(long val) {
    return Unit.BYTE.format(val);
  }

  private static interface Formatter {
    public String format(long value);
    public String format(double value);
    // Return a Formatter that format values with a fixed scale, rather than deciding the scale
    // every time based on the coming value. The fixed scale is picked based on representativeValue.
    public default Formatter withFixedScale(double representativeValue) { return this; }

    public static final Formatter DEFAULT = new Formatter() {
      @Override
      public String format(long value) {
        return String.format("%,d", value);
      }

      @Override
      public String format(double value) {
        return String.format("%,g", value);
      }
    };

    public static class Simple implements Formatter {
      private final String unit;

      public Simple(String unit) {
        this.unit = unit;
      }

      @Override
      public String format(long value) {
        return String.format("%,d%s", value, unit);
      }

      @Override
      public String format(double value) {
        return String.format("%,g%s", value, unit);
      }
    }

    public static class Percent extends Simple {
      private static final double CUT_OFF = 0.05;

      public Percent() {
        super("%");
      }

      @Override
      public String format(double value) {
        if (value == 100) {
          return super.format(100);
        } else {
          return value < CUT_OFF ? String.format("%,g%%", value) : String.format("%,.2f%%", value);
        }
      }
    }

    public static class Base2 implements Formatter {
      private static final int BASE = 1024;
      private static final int CUT_OVER = 1536; // 1.5 * BASE

      private final String[] names;
      private final int offset;

      public Base2(String[] names, int offset) {
        this.names = names;
        this.offset = offset;
      }

      @Override
      public String format(long value) {
        if (value < CUT_OVER) {
          return value + " " + names[offset];
        } else if (offset == names.length - 1) {
          return String.format("%,d%s", value, names[offset]);
        }

        int unit = offset + 1;
        while (unit < names.length && value >= CUT_OVER * BASE) {
          value /= BASE;
          unit++;
        }
        if (unit >= names.length) {
          return String.format(
              "%,d.%03d%s", value / BASE, (value % BASE) * 1000 / BASE, names[names.length - 1]);
        }
        return String.format(
            "%d.%03d%s", value / BASE, (value % BASE) * 1000 / BASE, names[unit]);
      }

      @Override
      public String format(double value) {
        if (value < CUT_OVER) {
          return String.format("%g%s", value, names[offset]);
        } else if (offset == names.length - 1) {
          return String.format("%,g%s", value, names[offset]);
        }

        int unit = offset + 1;
        while (unit < names.length && value >= CUT_OVER * BASE) {
          value /= BASE;
          unit++;
        }

        if (unit >= names.length) {
          double major = value / BASE;
          String name = names[names.length - 1];
          if (major > MAX_DOUBLE_AS_LONG) {
            return String.format("%,g%s", major, name);
          }
          return String.format(
              "%,d.%03d%s", (long)major, Math.round((value % BASE) * 1000 / BASE), name);
        }
        return String.format("%,d.%03d%s",
            (long)(value / BASE), Math.round((value % BASE) * 1000 / BASE), names[unit]);
      }

      @Override
      public Formatter withFixedScale(double representativeValue) {
        double bestConvertedValue = representativeValue;
        int bestOffset = this.offset;
        for (int offset = 0; offset < names.length; offset++) {
          double convertedValue = representativeValue * pow(BASE, this.offset - offset);
          // Wish the converted value will be larger than 1, but as small as possible.
          if (convertedValue > 1 && convertedValue < bestConvertedValue) {
            bestConvertedValue = convertedValue;
            bestOffset = offset;
          }
        }
        int fromOffset = this.offset,toOffset = bestOffset;
        return new Formatter() {
          @Override
          public String format(long value) {
            return String.format("%,.3f %s", value * pow(BASE, fromOffset - toOffset), names[toOffset]);
          }

          @Override
          public String format(double value) {
            return String.format("%,.3f %s", value * pow(BASE, fromOffset - toOffset), names[toOffset]);
          }
        };
      }
    }

    public static class Base10 implements Formatter {
      private static final int BASE = 1000;
      private static final int CUT_OVER = 1500; // 1.5 * BASE

      private final String[] names;
      private final int offset;

      public Base10(String[] names, int offset) {
        this.names = names;
        this.offset = offset;
      }

      @Override
      public String format(long value) {
        if (value < CUT_OVER) {
          return value + " " + names[offset];
        } else if (offset == names.length - 1) {
          return String.format("%,d%s", value, names[offset]);
        }

        int unit = offset + 1;
        while (unit < names.length && value >= CUT_OVER * BASE) {
          value /= BASE;
          unit++;
        }
        if (unit >= names.length) {
          return String.format("%,d.%03d%s", value / BASE, value % BASE, names[names.length - 1]);
        }
        return String.format("%d.%03d%s", value / BASE, value % BASE, names[unit]);
      }

      @Override
      public String format(double value) {
        if (value < CUT_OVER) {
          return String.format("%g%s", value, names[offset]);
        } else if (offset == names.length - 1) {
          return String.format("%,g%s", value, names[offset]);
        }

        int unit = offset + 1;
        while (unit < names.length && value >= CUT_OVER * BASE) {
          value /= BASE;
          unit++;
        }

        if (unit >= names.length) {
          double major = value / BASE;
          String name = names[names.length - 1];
          if (major > MAX_DOUBLE_AS_LONG) {
            return String.format("%,g%s", major, name);
          }
          return String.format("%,d.%03d%s", (long)major, Math.round(value % BASE), name);
        }
        return String.format(
            "%,d.%03d%s", (long)(value / BASE), Math.round(value % BASE), names[unit]);
      }

      @Override
      public Formatter withFixedScale(double representativeValue) {
        double bestConvertedValue = representativeValue;
        int bestOffset = this.offset;
        for (int offset = 0; offset < names.length; offset++) {
          double convertedValue = representativeValue * pow(BASE, this.offset - offset);
          // Wish the converted value will be larger than 1, but as small as possible.
          if (convertedValue > 1 && convertedValue < bestConvertedValue) {
            bestConvertedValue = convertedValue;
            bestOffset = offset;
          }
        }
        int fromOffset = this.offset, toOffset = bestOffset;
        return new Formatter() {
          @Override
          public String format(long value) {
            return String.format("%,.3f %s", value * pow(BASE, fromOffset - toOffset), names[toOffset]);
          }

          @Override
          public String format(double value) {
            return String.format("%,.3f %s", value * pow(BASE, fromOffset - toOffset), names[toOffset]);
          }
        };
      }
    }

    public static class Time implements Formatter {
      private static final int BASE_BELOW_S = 1000; // For values < 1s
      private static final int CUT_OVER_BELOW_S = 1500; // 1.5 * BASE
      private static final int BASE_ABOVE_S = 60;  // For values >= 1s
      private static final int CUT_OVER_ABOVE_S = 90; // 1.5 * BASE
      private static final int SECONDS = 3;

      private static final String[] NAMES = { "ns", "us", "ms", "s", "m", "h" };
      private static final double[][] TIME_CONVERT_TABLE = {
          {1,                      1f / 1000,              1f/pow(1000, 2), 1f/pow(1000, 3), 1f/(pow(1000, 3)* 60), 1f/(pow(1000, 3)* 60 * 60)}, // ns -> {ns, us, ms, s, m, h}
          {1000,                   1,                      1f / 1000,       1f/pow(1000, 2), 1f/(pow(1000, 2)* 60), 1f/(pow(1000, 2)* 60 * 60)}, // us -> {ns, us, ms, s, m, h}
          {pow(1000, 2),           1000,                   1,               1f / 1000,       1f/(1000* 60),         1f/(1000 * 60 * 60)},        // ms -> {ns, us, ms, s, m, h}
          {pow(1000, 3),           pow(1000, 2),           1000,            1,               1f/60,                 1f/(60 * 60)},               // s -> {ns, us, ms, s, m, h}
          {pow(1000, 3) * 60,      pow(1000, 2) * 60,      1000 * 60,       60,              1,                     1f/60},                      // m -> {ns, us, ms, s, m, h}
          {pow(1000, 3) * 60 * 60, pow(1000, 2) * 60 * 60, 1000 * 60 * 60,  60 * 60,         60,                    1},                          // h -> {ns, us, ms, s, m, h}
      };
      private final int offset;

      private Time(int offset) {
        this.offset = offset;
      }

      public static Time nanoseconds() {
        return new Time(0);
      }

      public static Time microseconds() {
        return new Time(1);
      }

      public static Time milliseconds() {
        return new Time(2);
      }

      public static Time seconds() {
        return new Time(3);
      }

      public static Time minutes() {
        return new Time(4);
      }

      public static Time hours() {
        return new Time(5);
      }

      @Override
      public String format(long value) {
        if ((offset < SECONDS && value < CUT_OVER_BELOW_S) ||
            (offset >= SECONDS && value < CUT_OVER_ABOVE_S)) {
          return value + " " + NAMES[offset];
        } else if (offset == NAMES.length - 1) {
          return String.format("%,dh", value);
        }

        int unit = offset;
        while (unit < SECONDS) {
          if (value < CUT_OVER_BELOW_S * BASE_BELOW_S) {
            return String.format(
                "%d.%03d%s", value / BASE_BELOW_S, value % BASE_BELOW_S, NAMES[unit + 1]);
          }
          value /= BASE_BELOW_S;
          unit++;
        }

        switch (unit) {
          case 3: {// seconds
            long s = value % BASE_ABOVE_S;
            long m = (value / BASE_ABOVE_S) % BASE_ABOVE_S;
            long h = (value / (BASE_ABOVE_S * BASE_ABOVE_S));
            if (h > 0) {
              return String.format("%dh %02dm %02ds", h, m, s);
            } else if (m > 0) {
              return String.format("%dm %02ds", m, s);
            } else {
              return s + "s";
            }
          }
          case 4: {// minutes
            long m = value % BASE_ABOVE_S;
            long h = (value / BASE_ABOVE_S);
            if (h > 0) {
              return String.format("%dh %02dm", h, m);
            } else {
              return m + "m";
            }
          }
          default:
            return String.format("%,dh", value);
        }
      }

      @Override
      public String format(double value) {
        if ((offset < SECONDS && value < CUT_OVER_BELOW_S) ||
            (offset >= SECONDS && value < CUT_OVER_ABOVE_S)) {
          return String.format("%g%s", value, NAMES[offset]);
        } else if (offset == NAMES.length - 1) {
          return String.format("%,gh", value);
        }

        int unit = offset;
        while (unit < SECONDS) {
          if (value < CUT_OVER_BELOW_S * BASE_BELOW_S) {
            return String.format("%d.%03d%s",
                (long)(value / BASE_BELOW_S), (long)value % BASE_BELOW_S, NAMES[unit + 1]);
          }
          value /= BASE_BELOW_S;
          unit++;
        }

        switch (unit) {
          case 3: {// seconds
            double s = value % BASE_ABOVE_S;
            long m = ((long)value / BASE_ABOVE_S) % BASE_ABOVE_S;
            long h = ((long)value / (BASE_ABOVE_S * BASE_ABOVE_S));
            if (h > 0) {
              return String.format("%dh %02dm %02gs", h, m, s);
            } else if (m > 0) {
              return String.format("%dm %02gs", m, s);
            } else {
              return String.format("%gs", s);
            }
          }
          case 4: {// minutes
            double m = value % BASE_ABOVE_S;
            long h = ((long)value / BASE_ABOVE_S);
            if (h > 0) {
              return String.format("%dh %02gm", h, m);
            } else {
              return String.format("%gm", m);
            }
          }
          default:
            return String.format("%,gh", value);
        }
      }

      @Override
      public Formatter withFixedScale(double representativeValue) {
        double bestConvertedValue = representativeValue;
        int bestOffset = this.offset;
        for (int offset = 0; offset < NAMES.length; offset++) {
          double convertedValue = representativeValue * TIME_CONVERT_TABLE[this.offset][offset];
          // Wish the converted value will be larger than 1, but as small as possible.
          if (convertedValue > 1 && convertedValue < bestConvertedValue) {
            bestConvertedValue = convertedValue;
            bestOffset = offset;
          }
        }
        int fromOffset = this.offset, toOffset = bestOffset;
        return new Formatter() {
          @Override
          public String format(long value) {
            return String.format("%,.3f %s", value * TIME_CONVERT_TABLE[fromOffset][toOffset],
                NAMES[toOffset]);
          }

          @Override
          public String format(double value) {
            return String.format("%,.3f %s", value * TIME_CONVERT_TABLE[fromOffset][toOffset],
                NAMES[toOffset]);
          }
        };
      }
    }
  }
}
