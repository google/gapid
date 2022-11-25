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

#include "command_buffer.h"

#include <format>

#include "command_buffer_recorder.h"
#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_command_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller, VkCommandBufferLevel level,
                                                      command_buffer_recorder* cbr) const {
  if (level == VK_COMMAND_BUFFER_LEVEL_PRIMARY) {
    serializer->insert_annotation("MecPrimaryCommandBuffers");
  } else {
    serializer->insert_annotation("MecSecondaryCommandBuffers");
  }
  for (auto& it : state_block->VkCommandBuffers) {
    auto buff = it.second.second;
    if (buff->get_allocate_info()->level != level) {
      continue;
    }
    VkCommandBuffer command_buffer = it.first;
    VkCommandBufferAllocateInfo inf = *buff->get_allocate_info();
    inf.commandBufferCount = 1;
    serializer->vkAllocateCommandBuffers(buff->device,
                                         &inf, &command_buffer);
    if (buff->invalidated) {
      serializer->insert_annotation(std::format("CommandBuffer - {} - Invalid", reinterpret_cast<uintptr_t>(buff->_handle)).c_str());
      continue;
    }
    cbr->RerecordCommandBuffer(command_buffer, serializer);
  }
}

}  // namespace gapid2