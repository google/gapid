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
#include "instance.h"
#include "temporary_allocator.h"

namespace gapid2 {
struct VkPhysicalDeviceWrapper : handle_base<VkPhysicalDevice, void> {
  VkPhysicalDeviceWrapper(
      VkPhysicalDevice physical_device)
      : handle_base<VkPhysicalDevice, void>(physical_device) {
  }

  void set_create_info(VkInstance instance_, uint32_t idx) {
    instance = instance_;
    physical_device_idx = idx;
  }

  VkInstance instance = VK_NULL_HANDLE;
  uint32_t physical_device_idx = static_cast<uint32_t>(-1);
  std::mutex child_mutex;
};
}  // namespace gapid2
