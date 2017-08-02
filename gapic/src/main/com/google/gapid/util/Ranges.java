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
    return UnsignedLongs.compare(range.getBase(), value) <= 0 &&
        UnsignedLongs.compare(range.getBase() + range.getSize(), value) > 0;
  }
}
