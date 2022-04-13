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
#include "helpers.h"
#include "null_cloner.h"
#include "struct_clone.h"
#include "temporary_allocator.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkDescriptorSetLayoutWrapper : handle_base<VkDescriptorSetLayout> {
  VkDescriptorSetLayoutWrapper(HandleUpdater*,
                               VkDevice,
                               VkDescriptorSetLayout descriptor_set)
      : handle_base<VkDescriptorSetLayout>(descriptor_set) {}

  void set_create_info(const VkDescriptorSetLayoutCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkDescriptorSetLayoutCreateInfo>(1);
    clone<NullCloner>(
        &cloner, pCreateInfo[0], create_info[0], &mem,
        _VkDescriptorSetLayoutCreateInfo_VkDescriptorSetLayoutBinding_stageFlags_valid,
        _VkDescriptorSetLayoutCreateInfo_VkDescriptorSetLayoutBinding_pImmutableSamplers_valid);
  }

  VkDescriptorSetLayoutCreateInfo* create_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2
