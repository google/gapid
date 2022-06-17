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
#include <mutex>

#include "handles.h"
#include "null_cloner.h"
#include "physical_device.h"
#include "temporary_allocator.h"

namespace gapid2 {
struct VkDeviceWrapper : handle_base<VkDevice, void> {
  VkDeviceWrapper(VkDevice device)
      : handle_base<VkDevice, void>(device) {}
  void set_device_loader_data(PFN_vkSetDeviceLoaderData data) {
    vkSetDeviceLoaderData = data;
    vkSetDeviceLoaderData(_handle, this);
  }

  void set_create_info(VkPhysicalDevice physical_device,
    state_block* state_block_,
    const VkDeviceCreateInfo* pCreateInfo);

  const VkDeviceCreateInfo* get_create_info() const {
    return create_info;
  }
  const VkPhysicalDevice get_physical_device() const {
    return physical_device;
  }

  PFN_vkSetDeviceLoaderData vkSetDeviceLoaderData;
 private:
  std::mutex child_mutex;
  VkDeviceCreateInfo* create_info = nullptr;
  VkPhysicalDevice physical_device = VK_NULL_HANDLE;
  temporary_allocator mem;
};
}  // namespace gapid2
