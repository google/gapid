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

import com.google.gapid.proto.service.Service.MemoryRange;

/**
 * {@link CommandRange} utilities.
 */
public class Ranges {
  private Ranges() {
  }

  /*
  public static CommandRange command(long index) {
    return CommandRange.newBuilder()
        .setFirst(index)
        .setCount(1)
        .build();
  }

  public static CommandRange commands(long from, long count) {
    return CommandRange.newBuilder()
        .setFirst(from)
        .setCount(count)
        .build();
  }
  */

  public static MemoryRange memory(long base, long size) {
    return MemoryRange.newBuilder()
        .setBase(base)
        .setSize(size)
        .build();
  }

  /*
  public static long first(CommandRangeOrBuilder range) {
    return range.getFirst();
  }

  public static long last(CommandRangeOrBuilder range) {
    return range.getFirst() + range.getCount() - 1;
  }

  public static long end(CommandRangeOrBuilder range) {
    return range.getFirst() + range.getCount();
  }

  public static long count(CommandRangeOrBuilder range) {
    return range.getCount();
  }

  public static boolean contains(CommandRangeOrBuilder range, long atomImdex) {
    return atomImdex >= first(range) && atomImdex <= last(range);
  }
  */

  /** @return whether a completely contains b. */
  /*
  public static boolean contains(CommandRangeOrBuilder a, CommandRangeOrBuilder b) {
    return first(a) <= first(b) && end(a) >= end(b);
  }
  */

  /** @return whether a has any overlap with b */
  /*
  public static boolean overlaps(CommandRangeOrBuilder a, CommandRangeOrBuilder b) {
    return first(a) < end(b) && first(b) < end(a);
  }

  public static boolean overlaps(
      List<? extends CommandRangeOrBuilder> list, CommandRangeOrBuilder range) {
    int rangeIndex = Collections.binarySearch(list, null, (x, ignored) ->
      (end(range) <= first(x)) ? 1 :
      (first(range) >= end(x)) ? -1 : 0);
    return rangeIndex >= 0;
  }

  public static int contains(List<? extends CommandRangeOrBuilder> list, long atomIndex) {
    return Collections.binarySearch(list, null, (x, ignored) ->
      (atomIndex < first(x)) ? 1 :
      (atomIndex >= end(x)) ? -1 : 0);
  }

  public static <T extends CommandRangeOrBuilder> int contains(T[] list, long atomIndex) {
    return Arrays.binarySearch(list, null, (x, ignored) ->
      (atomIndex < first(x)) ? 1 :
      (atomIndex >= end(x)) ? -1 : 0);
  }

  public static List<CommandRange> intersection(
      List<? extends CommandRangeOrBuilder> list, CommandRangeOrBuilder range) {
    assert range.getCount() > 0;

    // find something that matches in the list
    int rangeIndex = Collections.binarySearch(list, null, (x, ignored) ->
      (end(range) <= first(x)) ? 1 :
      (first(range) >= end(x)) ? -1 : 0);

    if (rangeIndex < 0) {
      return Collections.emptyList();
    }

    LinkedList<CommandRange> intersections = Lists.newLinkedList();
    intersections.add(intersection(range, list.get(rangeIndex)));

    int back = rangeIndex - 1;
    while (back > 0 && overlaps(list.get(back), range)) {
      intersections.addFirst(intersection(range, list.get(back)));
      back--;
    }

    int forward = rangeIndex + 1;
    while (forward < list.size() && overlaps(list.get(forward), range)) {
      intersections.addLast(intersection(range, list.get(forward)));
      forward++;
    }

    return intersections;
  }

  private static CommandRange intersection(
      CommandRangeOrBuilder range, CommandRangeOrBuilder foundRange) {
    long start = Math.max(first(range), first(foundRange));
    long end = Math.min(end(range), end(foundRange));
    return commands(start, end - start);
  }
  */
}
