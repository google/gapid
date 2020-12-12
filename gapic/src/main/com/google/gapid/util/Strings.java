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

import com.google.gapid.proto.service.path.Path;

/**
 * String utilities.
 */
public class Strings {
  private Strings() {
  }

  public static String stripQuotes(String s) {
    return (s == null || s.length() < 2 || s.charAt(0) != '"' || s.charAt(s.length() - 1) != '"') ?
      s : s.substring(1, s.length() - 1);
  }

  public static String toString(Path.ID id) {
    return new String(ProtoDebugTextFormat.escapeBytes(id.getData()));
  }
}
