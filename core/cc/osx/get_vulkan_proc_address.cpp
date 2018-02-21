/*
 * Copyright (C) 2017 Google Inc.
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

#include "../get_vulkan_proc_address.h"
#include "../log.h"

namespace {

void* getVulkanInstanceProcAddress(size_t instance, const char* name) {
  GAPID_FATAL("No Vulkan support on macOS");
  return nullptr;
}

void* getVulkanDeviceProcAddress(size_t instance, size_t device,
                                 const char* name) {
  GAPID_FATAL("No Vulkan support on macOS");
  return nullptr;
}

void* getVulkanProcAddress(const char* name) {
  return getVulkanInstanceProcAddress(0u, name);
}

}  // anonymous namespace

namespace core {

GetVulkanInstanceProcAddressFunc* GetVulkanInstanceProcAddress =
    getVulkanInstanceProcAddress;
GetVulkanDeviceProcAddressFunc* GetVulkanDeviceProcAddress =
    getVulkanDeviceProcAddress;
GetVulkanProcAddressFunc* GetVulkanProcAddress = getVulkanProcAddress;
bool HasVulkanLoader() { return false; }
}  // namespace core
