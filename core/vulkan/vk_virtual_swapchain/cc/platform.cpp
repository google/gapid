/*
 * Copyright (C) 2018 Google Inc.
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


#include "platform.h"

namespace swapchain {

void CreateSurface(const InstanceData* functions, VkInstance instance, const void* data, const VkAllocationCallbacks* pAllocator, VkSurfaceKHR* pSurface) {
#ifdef VK_USE_PLATFORM_ANDROID_KHR
  {
    auto pCreateInfo = static_cast<const VkAndroidSurfaceCreateInfoKHR*>(data);
    if (pCreateInfo->sType == VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR) {
      // Attempt to create android surface
      if (functions->vkCreateAndroidSurfaceKHR(instance, pCreateInfo, pAllocator, pSurface) != VK_SUCCESS) {
        *pSurface = 0;
      }
    }
  }
#else
  *pSurface = 0;
#endif
}

}
