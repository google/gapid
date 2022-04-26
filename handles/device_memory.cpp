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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "forwards.h"
#include "struct_clone.h"

namespace gapid2 {
void VkDeviceMemoryWrapper::set_allocate_info(state_block* state_block_, const VkMemoryAllocateInfo* pAllocateInfo) {
  allocate_info = mem.get_typed_memory<VkMemoryAllocateInfo>(1);
  clone(state_block_, pAllocateInfo[0], allocate_info[0], &mem);
  _size = pAllocateInfo->allocationSize;
}
}  // namespace gapid2
