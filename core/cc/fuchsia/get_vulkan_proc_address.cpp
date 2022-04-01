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

#include <iostream>

#include "../dl_loader.h"
#include "../get_vulkan_proc_address.h"
#include "../log.h"

namespace {

using namespace core;

// Definitions from the Vulkan API. Should keep the same with those in
// vulkan_types.h.
typedef void* PFN_vkVoidFunction;
typedef size_t VkDevice;
typedef size_t VkInstance;

void* getVulkanInstanceProcAddress(size_t instance, const char* name) {
  typedef PFN_vkVoidFunction (*VPAPROC)(VkInstance instance, const char* name);

  static DlLoader dylib("libvulkan.so", "libvulkan.so.1");

  if (VPAPROC vpa =
          reinterpret_cast<VPAPROC>(dylib.lookup("vkGetInstanceProcAddr"))) {
    if (void* proc = vpa(instance, name)) {
      GAPID_DEBUG("GetVulkanInstanceProcAddress(0x%x, %s) -> 0x%x", instance,
                  name, proc);
      return proc;
    }
  }

  GAPID_DEBUG("GetVulkanInstanceProcAddress(0x%x, %s) -> not found", instance,
              name);
  return nullptr;
}

void* getVulkanDeviceProcAddress(size_t instance, size_t device,
                                 const char* name) {
  typedef PFN_vkVoidFunction (*VPAPROC)(VkDevice device, const char* name);

  if (auto vpa = reinterpret_cast<VPAPROC>(
          getVulkanInstanceProcAddress(instance, "vkGetDeviceProcAddr"))) {
    if (void* proc = vpa(device, name)) {
      GAPID_DEBUG("GetVulkanDeviceProcAddress(0x%x, 0x%x, %s) -> 0x%x",
                  instance, device, name, proc);
      return proc;
    }
  }
  GAPID_DEBUG("GetVulkanDeviceProcAddress(0x%x, 0x%x, %s) -> not found",
              instance, device, name);
  return nullptr;
}

void* getVulkanProcAddress(const char* name) {
  auto ret = getVulkanInstanceProcAddress(0u, name);
  return ret;
}

}  // anonymous namespace

namespace core {

GetVulkanInstanceProcAddressFunc* GetVulkanInstanceProcAddress =
    getVulkanInstanceProcAddress;
GetVulkanDeviceProcAddressFunc* GetVulkanDeviceProcAddress =
    getVulkanDeviceProcAddress;
GetVulkanProcAddressFunc* GetVulkanProcAddress = getVulkanProcAddress;
bool HasVulkanLoader() {
  auto ret = DlLoader::can_load("libvulkan.dylib") ||
             DlLoader::can_load("libvulkan.1.dylib");
  return ret;
}
}  // namespace core
