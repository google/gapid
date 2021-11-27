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
#include <string>
#include <unordered_map>
#include <vector>
#include "device.h"
#include "null_cloner.h"

namespace gapid2 {

struct descriptor_usage {
  uint32_t set;
  uint32_t binding;
  uint32_t count;
};

template <typename HandleUpdater>
struct VkShaderModuleWrapper : handle_base<VkShaderModule> {
  VkShaderModuleWrapper(HandleUpdater*, VkDevice, VkShaderModule semaphore)
      : handle_base<VkShaderModule>(semaphore) {}

  void set_create_info(const VkShaderModuleCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkShaderModuleCreateInfo>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem,
                      _VkShaderModuleCreateInfo_pCode_length);
  }

  VkShaderModuleCreateInfo* create_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;

  std::unordered_map<std::string, std::vector<descriptor_usage>> _usage;
};
}  // namespace gapid2
