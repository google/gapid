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

import static com.google.gapid.perfetto.views.StyleConstants.DEFAULT_COUNTER_TRACK_HEIGHT;

import com.google.gapid.perfetto.models.CounterTrack;

public class VulkanCounterPanel extends CounterPanel implements Selectable {
  public VulkanCounterPanel(State state, CounterTrack track) {
    super(state, track, DEFAULT_COUNTER_TRACK_HEIGHT);
  }

  @Override
  public VulkanCounterPanel copy() {
    return new VulkanCounterPanel(state, track);
  }

  @Override
  public String getTitle() {
    // Perfetto convention for Vulkan counter names:
    // Device: vulkan.mem.device.memory.type.{}.{allocation|bind}
    // Driver: vulkan.mem.driver.scope.{COMMAND|OBJECT|CACHE|DEVICE|INSTANCE}
    String name = track.getCounter().name
        .replace("vulkan.mem.", "")
        .replace("driver.", "Driver, ")
        .replace("device.", "GPU, ")
        .replace("scope.", "Scope: ")
        .replace("memory.type.", "MemoryType: ")
        .replace(".allocation", ", Allocated")
        .replace(".bind", ", Bound");
    return name;
  }
}
