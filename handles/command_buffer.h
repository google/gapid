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

#include <deque>
#include <functional>

#include "device.h"
#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
struct VkCommandBufferWrapper : handle_base<VkCommandBuffer, void> {
  VkCommandBufferWrapper(VkCommandBuffer command_buffer)
      : handle_base<VkCommandBuffer, void>(command_buffer) {
  }

  void set_allocate_info(VkDevice device_, state_block* state_block_, const VkCommandBufferAllocateInfo* pAllocateInfo,
                         uint32_t index);

  const VkCommandBufferAllocateInfo* get_allocate_info() const {
    return allocate_info;
  }
  VkDevice device = VK_NULL_HANDLE;
  VkCommandBufferAllocateInfo* allocate_info = nullptr;
  temporary_allocator mem;
  uint32_t idx = 0xFFFFFFFF;
  std::deque<std::function<void()>> _pre_run_functions;
  std::deque<std::function<void()>> _post_run_functions;
};
}  // namespace gapid2
