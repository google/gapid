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

#define VK_NO_PROTOTYPES
#include "mec_controller.h"

#include <vulkan/vulkan.h>

#include "mec_capture/mid_execution_generator.h"
#include "state_block.h"

namespace gapid2 {
void mec_controller::start_capture() {
    for (auto i : state_block_->VkDevices) {
    passthrough_caller_->vkDeviceWaitIdle(i.first);
  }

  spy_serializer_->enable_with_mec();
  spy_->reset_memory_watch();
  mid_execution_generator gen;
  gen.begin_mid_execution_capture(state_block_, &noop_serializer, passthrough_caller_, cbr_);
  spy_serializer_->enable();
}

void mec_controller::end_capture() {
  spy_serializer_->disable();
}

}  // namespace gapid2