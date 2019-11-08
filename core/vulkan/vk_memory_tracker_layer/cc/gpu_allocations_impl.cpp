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

#include "core/vulkan/vk_memory_tracker_layer/cc/memory_tracker_layer_impl.h"

namespace memory_tracker {

extern MemoryTracker memory_tracker_instance;

VkResult vkCreateDevice(PFN_vkCreateDevice fn, VkPhysicalDevice physicalDevice,
                        VkDeviceCreateInfo const* pCreateInfo,
                        AllocationCallbacks pAllocator, VkDevice* pDevice) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator, "vkCreateDevice");
  VkResult result = fn(physicalDevice, pCreateInfo, trackedAllocator, pDevice);
  if (result == VK_SUCCESS) {
    memory_tracker_instance.ProcessCreateDeviceEvent(physicalDevice,
                                                     pCreateInfo, *pDevice);
  }
  return result;
}

void vkDestroyDevice(PFN_vkDestroyDevice fn, VkDevice device,
                     AllocationCallbacks pAllocator) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator,
                                                  "vkDestroyDevice");
  memory_tracker_instance.ProcessDestoryDeviceEvent(device);
  return fn(device, trackedAllocator);
}

VkResult vkAllocateMemory(PFN_vkAllocateMemory fn, VkDevice device,
                          VkMemoryAllocateInfo const* pAllocateInfo,
                          AllocationCallbacks pAllocator,
                          VkDeviceMemory* pMemory) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator,
                                                  "vkAllocateMemory");
  VkResult result = fn(device, pAllocateInfo, trackedAllocator, pMemory);
  if (result == VK_SUCCESS) {
    memory_tracker_instance.ProcessAllocateMemoryEvent(device, *pMemory,
                                                       pAllocateInfo);
  }
  return result;
}

void vkFreeMemory(PFN_vkFreeMemory fn, VkDevice device, VkDeviceMemory memory,
                  AllocationCallbacks pAllocator) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator, "vkFreeMemory");
  memory_tracker_instance.ProcessFreeMemoryEvent(device, memory);
  return fn(device, memory, trackedAllocator);
}

VkResult vkCreateBuffer(PFN_vkCreateBuffer fn, VkDevice device,
                        VkBufferCreateInfo const* pCreateInfo,
                        AllocationCallbacks pAllocator, VkBuffer* pBuffer) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator, "vkCreateBuffer");
  VkResult result = fn(device, pCreateInfo, trackedAllocator, pBuffer);
  if (result == VK_SUCCESS) {
    memory_tracker_instance.ProcessCreateBufferEvent(device, *pBuffer,
                                                     pCreateInfo);
  }
  return result;
}

VkResult vkBindBufferMemory(PFN_vkBindBufferMemory fn, VkDevice device,
                            VkBuffer buffer, VkDeviceMemory memory,
                            VkDeviceSize memoryOffset) {
  VkResult result = fn(device, buffer, memory, memoryOffset);
  if (result == VK_SUCCESS) {
    memory_tracker_instance.ProcessBindBufferEvent(device, buffer, memory,
                                                   memoryOffset);
  }
  return result;
}

VkResult vkBindBufferMemory2(PFN_vkBindBufferMemory2 fn, VkDevice device,
                             uint32_t bindInfoCount,
                             VkBindBufferMemoryInfo const* pBindInfos) {
  VkResult result = fn(device, bindInfoCount, pBindInfos);
  if (result == VK_SUCCESS) {
    for (uint32_t i = 0; i < bindInfoCount; i++) {
      memory_tracker_instance.ProcessBindBufferEvent(
          device, pBindInfos[i].buffer, pBindInfos[i].memory,
          pBindInfos[i].memoryOffset);
    }
  }
  return result;
}

void vkDestroyBuffer(PFN_vkDestroyBuffer fn, VkDevice device, VkBuffer buffer,
                     AllocationCallbacks pAllocator) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator,
                                                  "vkDestroyBuffer");
  memory_tracker_instance.ProcessDestroyBufferEvent(device, buffer);
  return fn(device, buffer, trackedAllocator);
}

VkResult vkCreateImage(PFN_vkCreateImage fn, VkDevice device,
                       VkImageCreateInfo const* pCreateInfo,
                       AllocationCallbacks pAllocator, VkImage* pImage) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator, "vkCreateImage");
  VkResult result = fn(device, pCreateInfo, trackedAllocator, pImage);
  if (result == VK_SUCCESS) {
    memory_tracker_instance.ProcessCreateImageEvent(device, *pImage,
                                                    pCreateInfo);
  }
  return result;
}

VkResult vkBindImageMemory(PFN_vkBindImageMemory fn, VkDevice device,
                           VkImage image, VkDeviceMemory memory,
                           VkDeviceSize memoryOffset) {
  VkResult result = fn(device, image, memory, memoryOffset);
  if (result == VK_SUCCESS) {
    memory_tracker_instance.ProcessBindImageEvent(device, image, memory,
                                                  memoryOffset);
  }
  return result;
}

VkResult vkBindImageMemory2(PFN_vkBindImageMemory2 fn, VkDevice device,
                            uint32_t bindInfoCount,
                            VkBindImageMemoryInfo const* pBindInfos) {
  VkResult result = fn(device, bindInfoCount, pBindInfos);
  if (result == VK_SUCCESS) {
    for (uint32_t i = 0; i < bindInfoCount; i++) {
      memory_tracker_instance.ProcessBindImageEvent(device, pBindInfos[i].image,
                                                    pBindInfos[i].memory,
                                                    pBindInfos[i].memoryOffset);
    }
  }
  return result;
}

void vkDestroyImage(PFN_vkDestroyImage fn, VkDevice device, VkImage image,
                    AllocationCallbacks pAllocator) {
  AllocationCallbacks trackedAllocator =
      memory_tracker_instance.GetTrackedAllocator(pAllocator, "vkDestroyImage");
  memory_tracker_instance.ProcessDestroyImageEvent(device, image);
  return fn(device, image, trackedAllocator);
}

}  // namespace memory_tracker