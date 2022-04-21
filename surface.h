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
struct VkSurfaceKHRWrapper : handle_base<VkSurfaceKHR> {
  VkSurfaceKHRWrapper(HandleUpdater*, VkInstance, VkSurfaceKHR sampler)
      : handle_base<VkSurfaceKHR>(sampler) {}

#if defined(VK_USE_PLATFORM_WIN32_KHR)
  void set_create_info(const VkWin32SurfaceCreateInfoKHR* pCreateInfo) {
    create_info = mem.get_typed_memory<VkWin32SurfaceCreateInfoKHR>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem);
  }

  VkWin32SurfaceCreateInfoKHR* create_info = nullptr;
#elif defined(VK_USE_PLATFORM_XCB_KHR)
void set_create_info(const VkXcbSurfaceCreateInfoKHR* pCreateInfo) {
    create_info = mem.get_typed_memory<VkXcbSurfaceCreateInfoKHR>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem, _VkXcbSurfaceCreateInfoKHR_connection_valid);
  }

  VkXcbSurfaceCreateInfoKHR* create_info = nullptr;
#endif
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2
