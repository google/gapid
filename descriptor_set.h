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

#include <vulkan.h>
#include <map>
#include <vector>
#include "descriptor_set_layout.h"
#include "device.h"
#include "device_functions.h"
#include "handles.h"
#include "null_cloner.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkDescriptorSetWrapper : handle_base<VkDescriptorSet> {
  VkDescriptorSetWrapper(HandleUpdater*,
                         VkDevice,
                         VkDescriptorSet descriptor_set)
      : handle_base<VkDescriptorSet>(descriptor_set) {}

  void set_layout(VkDescriptorSetLayoutWrapper<HandleUpdater>* layout) {
    _layout = layout;
    for (size_t i = 0; i < _layout->create_info->bindingCount; ++i) {
      auto& inf = _layout->create_info->pBindings[i];
      bindings[inf.binding] = binding{
          inf.descriptorType, std::vector<binding_type>(inf.descriptorCount)};
    }
  }

  void set_allocate_info(const VkDescriptorSetAllocateInfo* pAllocateInfo,
                         uint32_t index) {
    allocate_info = mem.get_typed_memory<VkDescriptorSetAllocateInfo>(1);
    clone<NullCloner>(&cloner, pAllocateInfo[0], allocate_info[0], &mem);
    index = idx;
  }

  VkDescriptorSetAllocateInfo* allocate_info = nullptr;
  uint32_t idx = 0;
  NullCloner cloner;
  temporary_allocator mem;

  VkDescriptorSetLayoutWrapper<HandleUpdater>* _layout;
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
