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

#include <handles.h>
#include <vulkan.h>
#include <vector>
#include "device.h"
#include "device_memory.h"
#include "null_cloner.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkImageWrapper : handle_base<VkImage> {
  VkImageWrapper(HandleUpdater*, VkDevice, VkImage image)
      : handle_base<VkImage>(image) {}

  void set_create_info(const VkImageCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkImageCreateInfo>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem,
                      _VkImageCreateInfo_pQueueFamilyIndices_valid);
  }

  void set_swapchain_info(VkSwapchainKHR swap, uint32_t i) {
    swapchain = swap;
    swapchain_idx = i;
  }

  VkImageCreateInfo* create_info = nullptr;
  VkSwapchainKHR swapchain = VK_NULL_HANDLE;
  uint32_t swapchain_idx = 0xFFFFFFFF;
  NullCloner cloner;
  temporary_allocator mem;

  VkDeviceSize required_size;
  std::vector<memory_binding> bindings;
};
}  // namespace gapid2
