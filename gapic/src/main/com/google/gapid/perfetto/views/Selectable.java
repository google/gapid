/*
 * Copyright (C) 2019 Google Inc.
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
package com.google.gapid.perfetto.views;

import com.google.gapid.perfetto.TimeSpan;
import com.google.gapid.perfetto.canvas.Area;
import com.google.gapid.perfetto.models.Selection;

/**
 * {@link Panel} that can contribute to the current selection. Use as a {@link Panel.Visitor}.
 */
public interface Selectable {
  public void computeSelection(Selection.CombiningBuilder builder, Area area, TimeSpan ts);

  public static enum Kind {
    // Order as shown in the UI.
    Thread, ThreadState, Cpu;
  }
}
