#pragma once
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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "creation_tracker.h"
#include "null_cloner.h"
#include "temporary_allocator.h"

namespace gapid2 {
class minimal_state_tracker : public creation_tracker<VkDeviceMemory, VkDescriptorUpdateTemplate, VkCommandBuffer> {
 protected:
  using super = creation_tracker;

 public:
  void vkGetPhysicalDeviceMemoryProperties(
      VkPhysicalDevice physicalDevice,
      VkPhysicalDeviceMemoryProperties* pMemoryProperties) override;
  void vkGetPhysicalDeviceMemoryProperties2(
      VkPhysicalDevice physicalDevice,
      VkPhysicalDeviceMemoryProperties2* pMemoryProperties) override;

  VkResult vkAllocateMemory(VkDevice device,
                            const VkMemoryAllocateInfo* pAllocateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkDeviceMemory* pMemory) override;

  VkResult vkMapMemory(VkDevice device,
                       VkDeviceMemory memory,
                       VkDeviceSize offset,
                       VkDeviceSize size,
                       VkMemoryMapFlags flags,
                       void** ppData) override;

  void vkUnmapMemory(VkDevice device, VkDeviceMemory memory) override;

  VkResult vkBeginCommandBuffer(
      VkCommandBuffer commandBuffer,
      const VkCommandBufferBeginInfo* pBeginInfo) override;
  VkResult vkQueueSubmit(VkQueue queue,
                         uint32_t submitCount,
                         const VkSubmitInfo* pSubmits,
                         VkFence fence) override;
  VkResult vkCreateDescriptorUpdateTemplate(
      VkDevice device,
      const VkDescriptorUpdateTemplateCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorUpdateTemplate* pDescriptorUpdateTemplate) override;

 protected:
  VkPhysicalDeviceMemoryProperties memory_properties;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2