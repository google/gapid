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

#include "descriptor_set.h"

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "descriptor_set_layout.h"
#include "forwards.h"
#include "struct_clone.h"

namespace gapid2 {

void VkDescriptorSetWrapper::set_layout(std::shared_ptr<VkDescriptorSetLayoutWrapper> layout) {
  _layout = layout;
  for (size_t i = 0; i < _layout->create_info->bindingCount; ++i) {
    auto& inf = _layout->create_info->pBindings[i];
    bindings[inf.binding] = binding{
        inf.descriptorType, std::vector<binding_type>(inf.descriptorCount)};
  }
}

void VkDescriptorSetWrapper::set_allocate_info(VkDevice device_,
                                               state_block* state_block_, const VkDescriptorSetAllocateInfo* pAllocateInfo,
                                               uint32_t index) {
  device = device_;
  allocate_info = mem.get_typed_memory<VkDescriptorSetAllocateInfo>(1);
  clone(state_block_, pAllocateInfo[0], allocate_info[0], &mem);
  index = idx;
}

}  // namespace gapid2
