/*
 * Copyright (C) 2022 Google Inc.
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

#pragma once
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <fstream>
#include <set>

#include "command_caller.h"
#include "commands.h"
#include "encoder.h"
#include "layer_helper.h"
#include "layerer.h"
#include "minimal_state_tracker.h"

namespace gapid2 {
class Spy : public gapid2::Layerer<
                MinimalStateTracker<CommandCaller<HandleWrapperUpdater>>,
                HandleWrapperUpdater> {
  using super =
      gapid2::Layerer<MinimalStateTracker<CommandCaller<HandleWrapperUpdater>>,
                      HandleWrapperUpdater>;
  using caller = CommandCaller<HandleWrapperUpdater>;

 public:
  Spy() {
    auto layers = gapid2::get_layers();
    auto user_config = gapid2::get_user_config();
    // dont inline these as they are order dependent
    initializeLayers(layers, user_config);
  }
  void add_instance(VkInstance instance) {
    std::unique_lock l(map_mutex);
    instances.insert(instance);
  }
  std::mutex map_mutex;
  std::set<VkInstance> instances;
  temporary_allocator allocator;
};
}  // namespace gapid2