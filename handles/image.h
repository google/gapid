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

#include <vector>

#include "device.h"
#include "device_memory.h"
#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
struct VkImageWrapper : handle_base<VkImage> {
  VkImageWrapper(VkImage image)
      : handle_base<VkImage>(image) {}

  void set_create_info(VkDevice device, state_block* state_block_, const VkImageCreateInfo* pCreateInfo);
  void set_swapchain_info(VkDevice device, VkSwapchainKHR swap, uint32_t i);

  const VkImageCreateInfo* get_create_info() const {
    return create_info;
  }

  VkSwapchainKHR get_swapchain() const {
    return swapchain;
  }

  VkImageCreateInfo* create_info = nullptr;
  VkSwapchainKHR swapchain = VK_NULL_HANDLE;
  VkDevice device = VK_NULL_HANDLE;
  uint32_t swapchain_idx = 0xFFFFFFFF;
  NullCloner cloner;
  temporary_allocator mem;

  VkDeviceSize required_size;
  std::vector<memory_binding> bindings;
};
}  // namespace gapid2
