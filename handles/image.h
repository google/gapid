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
#include <map>
#include <functional>

namespace gapid2 {
class state_block;
struct VkImageWrapper : handle_base<VkImage> {
  VkImageWrapper(VkImage image)
      : handle_base<VkImage>(image) {}

  void set_create_info(VkDevice device, state_block* state_block_, const VkImageCreateInfo* pCreateInfo);
  void set_swapchain_info(VkDevice device, state_block* state_block_, VkSwapchainKHR swap, uint32_t i);

  const VkImageCreateInfo* get_create_info() const {
    return create_info;
  }

  VkSwapchainKHR get_swapchain() const {
    return swapchain;
  }

  uint32_t get_subresource_idx(uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect_flag) const;
  uint32_t get_aspect_index(VkImageAspectFlagBits aspect) const ;

  void for_each_subresource_in(VkImageSubresourceRange range, const std::function<void(uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect)>& fn);

  VkImageCreateInfo* create_info = nullptr;
  VkImageCreateInfo swapchain_create_info;
  VkSwapchainKHR swapchain = VK_NULL_HANDLE;
  VkDevice device = VK_NULL_HANDLE;
  uint32_t swapchain_idx = 0xFFFFFFFF;
  NullCloner cloner;
  temporary_allocator mem;

  VkDeviceSize required_size;
  std::vector<memory_binding> bindings;
  
  struct subresource_data {
    uint32_t src_queue_idx;
    uint32_t dst_queue_idx;
    VkImageLayout layout;
  };

  std::map<uint32_t, subresource_data> sr_data;
};
}  // namespace gapid2
