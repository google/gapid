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

import java.util.Arrays;

/**
 * Half-float (16bit float) utilities.
 */
public class Float16 {
  private static final int[] MANTISSA_LOOKUP;
  private static final int[] EXPONENT_LOOKUP;
  private static final int[] OFFSET_LOOKUP;
  static {
    MANTISSA_LOOKUP = new int[2048];
    MANTISSA_LOOKUP[0] = 0;
    for (int i = 1; i < 1024; i++) {
      int m = i << 13, e = 0;
      while ((m & 0x00800000) == 0) {
        e -= 0x00800000;
        m <<= 1;
      }
      MANTISSA_LOOKUP[i] = (m &~0x00800000) | (e + 0x38800000);
    }
    for (int i = 1024; i < 2048; i++) {
      MANTISSA_LOOKUP[i] = 0x38000000 | ((i - 1024) << 13);
    }

    EXPONENT_LOOKUP = new int[64];
    EXPONENT_LOOKUP[0] = 0;
    for (int i = 1; i < 31; i++) {
      EXPONENT_LOOKUP[i] = i << 23;
    }
    EXPONENT_LOOKUP[31] = 0x47800000;
    EXPONENT_LOOKUP[32] = 0x80000000;
    for (int i = 33; i < 63; i++) {
      EXPONENT_LOOKUP[i] =  0x80000000 | (i - 32) << 23;
    }
    EXPONENT_LOOKUP[63] = 0xC7800000;

    OFFSET_LOOKUP = new int[64];
    Arrays.fill(OFFSET_LOOKUP, 1024);
    OFFSET_LOOKUP[0] = OFFSET_LOOKUP[32] = 0;
  }

  private Float16() {
  }

  public static float shortBitsToFloat(int bits) {
    int upper = (bits >> 10) & 0x3f, lower = bits & 0x3ff;
    return Float.intBitsToFloat(
        MANTISSA_LOOKUP[OFFSET_LOOKUP[upper] + lower] | EXPONENT_LOOKUP[upper]);
  }
}
