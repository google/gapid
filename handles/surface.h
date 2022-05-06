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
struct VkSurfaceKHRWrapper : handle_base<VkSurfaceKHR> {
  VkSurfaceKHRWrapper(VkSurfaceKHR sampler)
      : handle_base<VkSurfaceKHR>(sampler) {}

  const auto* get_create_info() const { return create_info;  }

#if defined(VK_USE_PLATFORM_WIN32_KHR)
  void set_create_info(VkInstance instance, state_block* state_block_, const VkWin32SurfaceCreateInfoKHR* pCreateInfo);

  VkWin32SurfaceCreateInfoKHR* create_info = nullptr;
#elif defined(VK_USE_PLATFORM_XCB_KHR)
  void set_create_info(VkInstance instance, state_block* state_block_, const VkXcbSurfaceCreateInfoKHR* pCreateInfo);

  VkXcbSurfaceCreateInfoKHR* create_info = nullptr;
#endif
  VkInstance instance;
  temporary_allocator mem;
};
}  // namespace gapid2
