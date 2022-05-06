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

#include <map>
#include <vector>

#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
class VkDescriptorSetLayoutWrapper;
struct VkDescriptorSetWrapper : handle_base<VkDescriptorSet> {
  VkDescriptorSetWrapper(VkDescriptorSet descriptor_set)
      : handle_base<VkDescriptorSet>(descriptor_set) {}

  void set_layout(VkDescriptorSetLayoutWrapper* layout);

  void set_allocate_info(VkDevice device_,
                         state_block* state_block_, const VkDescriptorSetAllocateInfo* pAllocateInfo,
                         uint32_t index);
  const VkDescriptorSetAllocateInfo* get_allocate_info() const {
    return allocate_info;
  }

  VkDevice device;
  VkDescriptorSetAllocateInfo* allocate_info = nullptr;
  uint32_t idx = 0;
  temporary_allocator mem;

  VkDescriptorSetLayoutWrapper* _layout;
  union binding_type {
    VkDescriptorImageInfo image_info;
    VkDescriptorBufferInfo buffer_info;
    VkBufferView buffer_view_info;
  };
  struct binding {
    VkDescriptorType type;
    std::vector<binding_type> descriptors;
  };
  std::map<uint32_t, binding> bindings;
};
}  // namespace gapid2
