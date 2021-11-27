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

#include "layer.h"

#include <iostream>
#include <sstream>

namespace foo {
VKAPI_ATTR VkResult VKAPI_CALL
override_vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkInstance* pInstance) {
  std::ostringstream oss;

  oss << __FUNCTION__ << std::endl;
  oss << "  "
      << "enabledExtensionCount: " << pCreateInfo->enabledExtensionCount
      << std::endl;
  for (size_t i = 0; i < pCreateInfo->enabledExtensionCount; ++i) {
    oss << "    " << pCreateInfo->ppEnabledExtensionNames[i] << std::endl;
  }
  OutputDebugStringA(oss.str().c_str());

  return vkCreateInstance(pCreateInfo, pAllocator, pInstance);
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkCreateDevice(VkPhysicalDevice phys_dev,
                        const VkDeviceCreateInfo* pCreateInfo,
                        const VkAllocationCallbacks* pAllocator,
                        VkDevice* pDevice) {
  std::ostringstream oss;

  oss << __FUNCTION__ << std::endl;
  oss << "  "
      << "enabledExtensionCount: " << pCreateInfo->enabledExtensionCount
      << std::endl;
  for (size_t i = 0; i < pCreateInfo->enabledExtensionCount; ++i) {
    oss << "    " << pCreateInfo->ppEnabledExtensionNames[i] << std::endl;
  }
  OutputDebugStringA(oss.str().c_str());

  return vkCreateDevice(phys_dev, pCreateInfo, pAllocator, pDevice);
}
}  // namespace foo
