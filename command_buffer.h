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

#include <device.h>
#include <handles.h>
#include <vulkan.h>
#include <deque>
#include <functional>
#include "null_cloner.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkCommandBufferWrapper : handle_base<VkCommandBuffer, void> {
  VkCommandBufferWrapper(HandleUpdater* updater_,
                         const VkDevice device,
                         VkCommandBuffer command_buffer)
      : handle_base<VkCommandBuffer, void>(command_buffer) {
    auto dev = updater_->cast_from_vk(device);
    if (HandleUpdater::has_dispatch) {
      dev->vkSetDeviceLoaderData(device, this);
    }
    _functions = dev->_functions.get();
  }

  void set_allocate_info(const VkCommandBufferAllocateInfo* pAllocateInfo,
                         uint32_t index) {
    allocate_info = mem.get_typed_memory<VkCommandBufferAllocateInfo>(1);
    clone<NullCloner>(&cloner, pAllocateInfo[0], allocate_info[0], &mem);
    idx = index;
  }

  VkCommandBufferAllocateInfo* allocate_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;
  uint32_t idx = 0xFFFFFFFF;
  std::deque<std::function<void()>> _pre_run_functions;
  std::deque<std::function<void()>> _post_run_functions;
  DeviceFunctions* _functions;
};
}  // namespace gapid2
