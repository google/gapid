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

#include <memory>
#include <string>
#include <unordered_map>
#include <vector>

#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
struct descriptor_usage {
  uint32_t set;
  uint32_t binding;
  uint32_t count;
};

struct VkShaderModuleWrapper : handle_base<VkShaderModule> {
  VkShaderModuleWrapper(VkShaderModule semaphore)
      : handle_base<VkShaderModule>(semaphore) {}

  void set_create_info(VkDevice device_, state_block* state_block_, const VkShaderModuleCreateInfo* pCreateInfo);
  const VkShaderModuleCreateInfo* get_create_info() const {
    return create_info;
  }

  VkDevice device = VK_NULL_HANDLE;
  VkShaderModuleCreateInfo* create_info = nullptr;
  std::vector<uint32_t> words;
  temporary_allocator mem;

  std::unordered_map<std::string, std::vector<descriptor_usage>> _usage;
};
}  // namespace gapid2
