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

#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
struct VkCommandPoolWrapper : handle_base<VkCommandPool> {
  VkCommandPoolWrapper(VkCommandPool command_pool)
      : handle_base<VkCommandPool>(command_pool) {}

  void set_create_info(VkDevice device_, state_block* state_block_, const VkCommandPoolCreateInfo* pCreateInfo);
  const VkCommandPoolCreateInfo* get_create_info() const {
    return create_info;
  }
  VkDevice device = VK_NULL_HANDLE;
  VkCommandPoolCreateInfo* create_info = nullptr;
  temporary_allocator mem;
};
}  // namespace gapid2
