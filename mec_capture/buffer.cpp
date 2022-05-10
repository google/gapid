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

#include "buffer.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  for (auto& it : state_block->VkBuffers) {
    VkBufferWrapper* buff = it.second.second;
    VkBuffer buffer = it.first;
    serializer->vkCreateBuffer(buff->device,
                               buff->get_create_info(), nullptr, &buffer);
  }
}

void mid_execution_generator::capture_bind_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  for (auto& it : state_block->VkBuffers) {
    VkBufferWrapper* buff = it.second.second;
    GAPID2_ASSERT(0 == buff->get_create_info()->flags & VK_BUFFER_CREATE_SPARSE_BINDING_BIT, "We do not support sparse images yet");
    GAPID2_ASSERT(buff->bindings.size() <= 1, "Invalid number of binds");

#pragma TODO(awoloszyn, Handle the different special bind flags)
    if (buff->bindings.empty()) {
      continue;
    }
    serializer->vkBindBufferMemory(buff->device, it.first, buff->bindings[0].memory, buff->bindings[0].offset);
  }
}

}  // namespace gapid2