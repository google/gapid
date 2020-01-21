/*
 * Copyright (C) 2019 Google Inc.
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

import static com.google.common.base.Preconditions.checkArgument;
import static java.util.concurrent.TimeUnit.NANOSECONDS;

import java.util.function.Consumer;

/**
 * Represents a span of time in nanoseconds.
 */
public class TimeSpan {
  public static final TimeSpan ZERO = new TimeSpan(0, 0) {
    @Override
    public boolean isEmpty() {
      return true;
    }

    @Override
    public boolean contains(long time) {
      return false;
    }

    @Override
    public boolean contains(TimeSpan other) {
      return false;
    }

    @Override
    public TimeSpan expand(TimeSpan other) {
      return other;
    }
  };

  public final long start;
  public final long end;

  public TimeSpan(long startNs, long endNs) {
    checkArgument(startNs <= endNs, "%s > %s", startNs, endNs);
    this.start = startNs;
    this.end = endNs;
  }

  public long getDuration() {
    return end - start;
  }

  public boolean isEmpty() {
    return end == start;
  }

  public void ifNotEmpty(Consumer<TimeSpan> run) {
    if (!isEmpty()) {
      run.accept(this);
    }
  }

  public boolean contains(long time) {
    return (time >= start && time <= end);
  }

  public boolean contains(TimeSpan other) {
    return start <= other.start && end >= other.end;
  }

  public boolean overlaps(long oStart, long oEnd) {
    return start < oEnd && oStart < end;
  }

  public TimeSpan expand(long deltaNs) {
    return expand(deltaNs, deltaNs);
  }

  public TimeSpan expand(long deltaStartNs, long deltaEndNs) {
    return new TimeSpan(start - deltaStartNs, end + deltaEndNs);
  }

  public TimeSpan expand(TimeSpan other) {
    if (other.isEmpty() || contains(other)) {
      return this;
    }
    return new TimeSpan(Math.min(start, other.start), Math.max(end, other.end));
  }

  public TimeSpan align(long page) {
    long s = (start / page) * page;
    long e = ((end + page - 1) / page) * page;
    return new TimeSpan(s, e);
  }

  public TimeSpan move(long deltaNs) {
    return new TimeSpan(start + deltaNs, end + deltaNs);
  }

  public TimeSpan moveTo(long newStart) {
    return new TimeSpan(newStart, newStart + getDuration());
  }

  public TimeSpan boundedBy(TimeSpan bounds) {
    long s = Math.max(Math.min(start, bounds.end), bounds.start);
    long e = Math.min(Math.max(end, bounds.start), bounds.end);
    return (s == e) ? ZERO : new TimeSpan(s, e);
  }

  public TimeSpan boundedByPreservingDuration(TimeSpan bounds) {
    if (start < bounds.start) {
      return new TimeSpan(bounds.start, Math.min(bounds.end, bounds.start + getDuration()));
    } else if (end > bounds.end) {
      return new TimeSpan(Math.max(bounds.start, bounds.end - getDuration()), bounds.end);
    }
    return this;
  }

  @Override
  public String toString() {
    return "TimeSpan{start: " + start / 1e9 + ", end: " + end / 1e9 + "}";
  }

  @Override
  public boolean equals(Object obj) {
    if (obj == this) {
      return true;
    } else if (!(obj instanceof TimeSpan)) {
      return false;
    }
    TimeSpan o = (TimeSpan)obj;
    return start == o.start && end == o.end;
  }

  @Override
  public int hashCode() {
    return Long.hashCode(start) ^ Long.hashCode(end);
  }

  public static String timeToString(long ns) {
    ns = Math.abs(ns);
    long u = NANOSECONDS.toMicros(ns) % 1000;
    long m = NANOSECONDS.toMillis(ns) % 1000;
    long s = NANOSECONDS.toSeconds(ns);

    if (s > 0) {
      if (u == 0) {
        return (m == 0) ? s + "s" : String.format("%ds%03dms", s, m);
      } else {
        return String.format("%ds%03d.%03dms", s, m, u);
      }
    } else if (m > 0) {
      return (u == 0) ? m + "ms" : String.format("%d.%03dms", m, u);
    } else if (u > 0) {
      long n = ns % 1000;
      return (n == 0) ? u + "us" : String.format("%d.%03dus", u, n);
    } else {
      return ns + "ns";
    }
  }

  public static String timeToShortString(long ns) {
    ns = Math.abs(ns);
    long u = NANOSECONDS.toMicros(ns) % 1000;
    long m = NANOSECONDS.toMillis(ns) % 1000;
    long s = NANOSECONDS.toSeconds(ns);

    if (s > 0) {
      return (m == 0) ? s + "s" : String.format("%ds%03dms", s, m);
    } else if (m > 0) {
      return (u == 0) ? m + "ms" : String.format("%d.%03dms", m, u);
    } else if (u > 0) {
      long n = ns % 1000;
      return (n == 0) ? u + "us" : String.format("%d.%03dus", u, n);
    } else {
      return ns + "ns";
    }
  }
}
