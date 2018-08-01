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

import com.google.common.primitives.UnsignedLongs;
import com.google.gapid.proto.service.Service;

import java.util.ArrayList;
import java.util.List;

/**
 * Range utilities.
 */
public class Ranges {
  private Ranges() {
  }

  public static Service.MemoryRange memory(long base, long size) {
    return Service.MemoryRange.newBuilder()
        .setBase(base)
        .setSize(size)
        .build();
  }

  public static boolean contains(Service.MemoryRange range, long value) {
    return contains(range.getBase(), range.getSize(), value);
  }

  public static boolean contains(long base, long size, long value) {
    return UnsignedLongs.compare(base, value) <= 0 &&
        UnsignedLongs.compare(base + size, value) > 0;
  }

  public static boolean overlap(Service.MemoryRange range, long base, long size) {
    return overlap(range.getBase(), range.getSize(), base, size);
  }

  public static boolean overlap(long base1, long size1, long base2, long size2) {
    return base1 < base2 + size2 && base2 < base1 + size1;
  }

  /**
   * Returns the intersection of the ranges, relative to {@code base}. The ranges must overlap.
   */
  public static Service.MemoryRange relative(long base, long size, Service.MemoryRange range) {
    long b = Math.max(base, range.getBase());
    long s = Math.min(base + size, range.getBase() + range.getSize()) - b;
    return memory(b - base, s);
  }

  // r1 must contain r2.base.
  public static Service.MemoryRange merge(Service.MemoryRange r1, Service.MemoryRange r2) {
    long base = r1.getBase(), size = r2.getBase() + r2.getSize() - base;
    return (size <= r1.getSize()) ? r1 : memory(base, size);
  }

  /**
   * Merges overlapping and adjacent ranges. Notes: sorts the list.
   */
  public static List<Service.MemoryRange> merge(List<Service.MemoryRange> ranges) {
    if (ranges.size() < 2) {
      return ranges;
    }

    return ranges.stream()
      .sorted((r1, r2) -> {
        int r = Long.compare(r1.getBase(), r2.getBase());
        if (r == 0) {
          // sort by size descending, so that small ones are easily absorbed.
          r = Long.compare(r2.getSize(), r1.getSize());
        }
        return r;
      })
      .sequential()
      .collect(ArrayList::new, (list, range) -> {
        if (list.isEmpty()) {
          list.add(range);
        } else {
          Service.MemoryRange last = list.get(list.size() - 1);
          if (range.getBase() <= last.getBase() + last.getSize()) {
            list.set(list.size() - 1, merge(last, range));
          } else {
            list.add(range);
          }
        }
      }, (x, y) -> {
        // Sequential streams don't need a combiner.
        throw new UnsupportedOperationException();
      });
  }
}
