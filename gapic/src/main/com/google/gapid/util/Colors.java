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

import org.eclipse.swt.graphics.RGB;
import org.eclipse.swt.graphics.RGBA;

public class Colors {
  public static final double DARK_LUMINANCE_THRESHOLD = 0.65;
  public static final int DARK_LUMINANCE8_THRESHOLD = 165;

  private Colors() {
  }

  public static RGBA fromARGB(int argb) {
    return new RGBA((argb >> 16) & 0xFF, (argb >> 8) & 0xFF, argb & 0xFF, (argb >> 24) & 0xFF);
  }

  public static RGB fromRGB(int rgb) {
    return new RGB((rgb >> 16) & 0xFF, (rgb >> 8) & 0xFF, rgb & 0xFF);
  }

  /**
   * Computes the (relative) luminance of the given color.
   * @see <a href="https://en.wikipedia.org/wiki/Relative_luminance">Relative Luminance</a>
   */
  public static double getLuminance(int rgb) {
    return (
        0.2126 * ((rgb >> 16) & 0xFF) +
        0.7152 * ((rgb >> 8) & 0xFF) +
        0.0722 * (rgb & 0xFF)
    ) / 255.0;
  }

  /**
   * Computes the (relative) luminance of the given color.
   * @see <a href="https://en.wikipedia.org/wiki/Relative_luminance">Relative Luminance</a>
   */
  public static double getLuminance(float r, float g, float b) {
    return 0.2126 * r + 0.7152 * g + 0.0722 * b;
  }

  public static byte clamp(float f) {
    if (f <= 0) {
      return 0;
    } else if (f >= 1) {
      return -1;
    }
    return (byte)(f * 255);
  }
}
