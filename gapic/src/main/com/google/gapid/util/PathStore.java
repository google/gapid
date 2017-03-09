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

import com.google.gapid.proto.service.path.Path;

/**
 * Contains a mutable path reference.
 */
public class PathStore {
  private Path.Any path;

  public boolean update(Path.Any newPath) {
    if (is(newPath)) {
      return false;
    }
    path = newPath;
    return true;
  }

  /**
   * Same as {@link #update(Path.Any)}, but ignores passed null values.
   */
  public boolean updateIfNotNull(Path.Any newPath) {
    return newPath != null && update(newPath);
  }

  public Path.Any getPath() {
    return path;
  }

  public boolean is(Path.Any otherPath) {
    return (path == null) ? otherPath == null : path.equals(otherPath);
  }
}
