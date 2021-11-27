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
#include "device.h"
#include "null_cloner.h"

namespace gapid2 {
template <typename HandleUpdater>
struct VkQueueWrapper : handle_base<VkQueue, void> {
  VkQueueWrapper(HandleUpdater* updater_, VkDevice device, VkQueue queue)
      : handle_base<VkQueue, void>(queue) {
    auto dev = updater_->cast_from_vk(device);
    if (HandleUpdater::has_dispatch) {
      dev->vkSetDeviceLoaderData(device, this);
    }
    _functions = dev->_functions.get();
  }

  void set_create_info(uint32_t queueFamilyIndex, uint32_t queueIndex) {
    queue_family_index = queue_family_index;
    queue_index = queue_index;
  }

  void set_create_info2(const VkDeviceQueueInfo2* pQueueInfo) {
    create_info2 = mem.get_typed_memory<VkDeviceQueueInfo2>(1);
    clone<NullCloner>(&cloner, pQueueInfo[0], create_info2[0], &mem);
  }

  uint32_t queue_family_index = 0xFFFFFFFF;
  uint32_t queue_index = 0xFFFFFFFF;

  VkDeviceQueueInfo2* create_info2 = nullptr;
  NullCloner cloner;
  temporary_allocator mem;

  DeviceFunctions* _functions;
};
}  // namespace gapid2
