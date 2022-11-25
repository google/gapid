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

#include "device.h"
#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
struct VkQueueWrapper : handle_base<VkQueue, void> {
  VkQueueWrapper(VkQueue queue)
      : handle_base<VkQueue, void>(queue) {
  }

  void set_create_info(VkDevice device, uint32_t queueFamilyIndex, uint32_t queueIndex);
  void set_create_info2(VkDevice device , state_block* state_block_, const VkDeviceQueueInfo2* pQueueInfo);

  const VkDeviceQueueInfo2* get_info_2() const {
    return create_info2;
  }

  uint32_t queue_family_index = 0xFFFFFFFF;
  uint32_t queue_index = 0xFFFFFFFF;
  VkDevice device = VK_NULL_HANDLE;
  VkDeviceQueueInfo2* create_info2 = nullptr;
  temporary_allocator mem;
};
}  // namespace gapid2