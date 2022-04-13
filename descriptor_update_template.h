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
#include "device.h"
#include "device_functions.h"
#include "handles.h"
#include "null_cloner.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkDescriptorUpdateTemplateWrapper
    : handle_base<VkDescriptorUpdateTemplate> {
  VkDescriptorUpdateTemplateWrapper(HandleUpdater*,
                                    VkDevice,
                                    VkDescriptorUpdateTemplate descriptor_set)
      : handle_base<VkDescriptorUpdateTemplate>(descriptor_set) {}

  void set_create_info(
      const VkDescriptorUpdateTemplateCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkDescriptorUpdateTemplateCreateInfo>(1);
    clone<NullCloner>(
        &cloner, pCreateInfo[0], create_info[0], &mem,
        _VkDescriptorUpdateTemplateCreateInfo_descriptorSetLayout_valid,
        _VkDescriptorUpdateTemplateCreateInfo_pipelineBindPoint_valid,
        _VkDescriptorUpdateTemplateCreateInfo_pipelineLayout_valid,
        _VkDescriptorUpdateTemplateCreateInfo_set_valid);
  }

  VkDescriptorUpdateTemplateCreateInfo* create_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2
