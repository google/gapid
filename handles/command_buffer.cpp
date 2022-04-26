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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "forwards.h"
#include "struct_clone.h"

namespace gapid2 {

void VkCommandBufferWrapper::set_allocate_info(state_block* state_block_, const VkCommandBufferAllocateInfo* pAllocateInfo,
                                               uint32_t index) {
  allocate_info = mem.get_typed_memory<VkCommandBufferAllocateInfo>(1);
  clone(state_block_, pAllocateInfo[0], allocate_info[0], &mem);
  idx = index;
}
}  // namespace gapid2
