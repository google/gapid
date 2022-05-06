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

#include "queue.h"

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "forwards.h"
#include "struct_clone.h"

namespace gapid2 {

void VkQueueWrapper::set_create_info(VkDevice device_, uint32_t queueFamilyIndex, uint32_t queueIndex) {
  queue_family_index = queue_family_index;
  queue_index = queue_index;
  device = device_;
}

void VkQueueWrapper::set_create_info2(VkDevice device_, state_block* state_block_, const VkDeviceQueueInfo2* pQueueInfo) {
  set_create_info(device_, pQueueInfo->queueFamilyIndex, pQueueInfo->queueIndex);
  create_info2 = mem.get_typed_memory<VkDeviceQueueInfo2>(1);
  clone(state_block_, pQueueInfo[0], create_info2[0], &mem);
}

}  // namespace gapid2
