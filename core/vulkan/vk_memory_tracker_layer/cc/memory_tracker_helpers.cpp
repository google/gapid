/*
 * Copyright (C) 2019 Google Inc.
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

#include "core/vulkan/vk_memory_tracker_layer/cc/layer.h"

namespace memory_tracker {

VkResult vkCreateDevice(PFN_vkCreateDevice fn, VkPhysicalDevice physicalDevice,
                        const VkDeviceCreateInfo* pCreateInfo,
                        AllocationCallbacks pAllocator, VkDevice* pDevice) {
  return fn(physicalDevice, pCreateInfo, pAllocator, pDevice);
}

void vkCmdDraw(PFN_vkCmdDraw fn, VkCommandBuffer commandBuffer,
               uint32_t vertexCount, uint32_t instanceCount,
               uint32_t firstVertex, uint32_t firstInstance) {
  return fn(commandBuffer, vertexCount, instanceCount, firstVertex,
            firstInstance);
}

VkResult vkQueueSubmit(PFN_vkQueueSubmit fn, VkQueue queue,
                       uint32_t submitCount, const VkSubmitInfo* pSubmits,
                       VkFence fence) {
  return fn(queue, submitCount, pSubmits, fence);
}

VkResult vkCreateInstance(PFN_vkCreateInstance fn,
                          const VkInstanceCreateInfo* pCreateInfo,
                          AllocationCallbacks pAllocator,
                          VkInstance* pInstance) {
  return fn(pCreateInfo, pAllocator, pInstance);
}

}  // namespace memory_tracker