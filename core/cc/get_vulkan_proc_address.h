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

#ifndef CORE_GET_VULKAN_PROC_ADDRESS_H
#define CORE_GET_VULKAN_PROC_ADDRESS_H

#include <stddef.h>

namespace core {

typedef void*(GetVulkanProcAddressFunc)(const char* name);
typedef void*(GetVulkanInstanceProcAddressFunc)(size_t instance,
                                                const char* name);
typedef void*(GetVulkanDeviceProcAddressFunc)(size_t instance, size_t device,
                                              const char* name);

// GetVulkanProcAddress returns the Vulkan function pointer to the function with
// the given name, or nullptr if the function was not found.
extern GetVulkanProcAddressFunc* GetVulkanProcAddress;
// Additional functions for fetching the function pointer with the given name
// from the given scope (instance or device).
extern GetVulkanInstanceProcAddressFunc* GetVulkanInstanceProcAddress;
extern GetVulkanDeviceProcAddressFunc* GetVulkanDeviceProcAddress;

// HasVulkanLoader returns true if Vulkan loader is found, otherwise returns
// false.
bool HasVulkanLoader();

}  // namespace core

#endif  // CORE_GET_VULKAN_PROC_ADDRESS_H
