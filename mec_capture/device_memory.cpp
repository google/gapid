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

#include "device_memory.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_allocations(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecAllocations");
  for (auto& it : state_block->VkDeviceMemorys) {
    auto dev_mem = it.second.second;
    VkDeviceMemory device_memory = it.first;

    serializer->vkAllocateMemory(
        dev_mem->device,
        dev_mem->allocate_info,
        nullptr,
        &device_memory);
    if (dev_mem->_mapped_location != nullptr && dev_mem->_mapped_size != 0) {
      void* ml = dev_mem->_mapped_location;
      serializer->vkMapMemory(dev_mem->device, device_memory, dev_mem->_mapped_offset,
                              dev_mem->_mapped_size, dev_mem->_mapped_flags, &ml);
    }
  }
}

}  // namespace gapid2