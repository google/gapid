/*
 * Copyright (C) 2018 Google Inc.
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
 * Array utilities.
 */
public class Arrays {
  private Arrays() {
  }

  public static <T> T last(T[] array) {
    return (array == null || array.length == 0) ? null : array[array.length - 1];
  }

  public static <T> T getOrDefault(T[] array, int idx, T dflt) {
    return array == null || idx >= array.length ? dflt : array[idx];
  }
}
