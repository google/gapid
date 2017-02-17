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
 * Inclusive integer interval.
 */
public class IntRange {
  public final int from, to;

  public IntRange(int from, int to) {
    this.from = from;
    this.to = to;
  }

  public boolean isWithin(int x) {
    return x >= from && x <= to;
  }

  @Override
  public int hashCode() {
    return (from * 31) ^ to;
  }

  @Override
  public boolean equals(Object obj) {
    if (obj == this) {
      return true;
    } else if (!(obj instanceof IntRange)) {
      return false;
    }
    IntRange o = (IntRange)obj;
    return from == o.from && to == o.to;
  }
}
