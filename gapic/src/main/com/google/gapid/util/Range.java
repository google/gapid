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

/**
 * Range defines an immutable min-max interval of doubles.
 */
public class Range {
  public static final Range IDENTITY = new Range(0.0, 1.0);

  public final double min;
  public final double max;

  public Range(double min, double max) {
    this.min = min;
    this.max = max;
  }

  /**
   * @return the value limited to the min and max values of this range.
   */
  public double clamp(double value) {
    return Math.max(Math.min(value, max), min);
  }

  /**
   * @return the linear interpolated value between min and max by frac.
   */
  public double lerp(double frac) {
    return min + (max - min) * frac;
  }

  /**
   * @return the inverse of {@link #lerp}, where X = frac(lerp(X)).
   */
  public double frac(double value) {
    return (value - min) / (max - min);
  }

  /**
   * @return the size of the range interval.
   */
  public double range() {
    return max - min;
  }
}