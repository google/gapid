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

/**
 * Color handling utilities.
 */
public class Colors {
  public static final double DARK_LUMINANCE_THRESHOLD = 0.65;
  public static final int DARK_LUMINANCE8_THRESHOLD = 165;
  public static final RGB BLACK = new RGB(0, 0, 0);
  public static final RGB WHITE = new RGB(255, 255, 255);
  // https://en.wikipedia.org/wiki/Golden_angle
  private static final double GOLDEN_ANGLE = 2.39996322972865332;

  private Colors() {
  }

  public static RGBA fromARGB(int argb) {
    return new RGBA((argb >> 16) & 0xFF, (argb >> 8) & 0xFF, argb & 0xFF, (argb >> 24) & 0xFF);
  }

  public static RGB fromRGB(int rgb) {
    return new RGB((rgb >> 16) & 0xFF, (rgb >> 8) & 0xFF, rgb & 0xFF);
  }

  public static RGB rgb(double r, double g, double b) {
    return new RGB((int)(r * 255), (int)(g * 255), (int)(b * 255));
  }

  public static RGBA rgb(int r, int g, int b) {
    return new RGBA(r, g, b, 255);
  }

  public static RGBA rgba(int r, int g, int b, int a) {
    return new RGBA(r, g, b, a);
  }

  public static RGBA rgba(int r, int g, int b, float a) {
    return new RGBA(r, g, b, Math.round(255 * a));
  }

  public static RGBA hsl(float h, float s, float l) {
    return hsla(h, s, l, 255);
  }

  public static RGBA hsla(float h, float s, float l, float a) {
    l *= 2;
    s *= (l <= 1) ? l : 2 - l;
    return new RGBA(h, (2 * s) / (l + s), (l + s) / 2, a);
  }


  public static RGB getRandomColor(int index) {
    float hue = (float)Math.toDegrees(index * GOLDEN_ANGLE) % 360;
    return new RGB(hue, 0.8f, 0.8f);
  }

  public static RGB lerp(RGB c1, RGB c2, float a) {
    return new RGB(
        clamp((int)(c2.red   + (c1.red   - c2.red  ) * a)),
        clamp((int)(c2.green + (c1.green - c2.green) * a)),
        clamp((int)(c2.blue  + (c1.blue  - c2.blue ) * a)));
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

  public static double getLuminance(double r, double g, double b) {
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

  public static int clamp(int v) {
    return Math.max(Math.min(v, 255), 0);
  }
}
