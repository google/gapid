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

#include "minimal_state_tracker.h"

#include "command_buffer.h"
#include "descriptor_update_template.h"
#include "device_memory.h"
#include "struct_clone.h"

namespace gapid2 {
class state_block;
void minimal_state_tracker::vkGetPhysicalDeviceMemoryProperties(
    VkPhysicalDevice physicalDevice,
    VkPhysicalDeviceMemoryProperties* pMemoryProperties) {
  super::vkGetPhysicalDeviceMemoryProperties(physicalDevice,
                                             pMemoryProperties);
  clone(state_block_, pMemoryProperties[0], memory_properties, &mem);
}
void minimal_state_tracker::vkGetPhysicalDeviceMemoryProperties2(
    VkPhysicalDevice physicalDevice,
    VkPhysicalDeviceMemoryProperties2* pMemoryProperties) {
  super::vkGetPhysicalDeviceMemoryProperties2(physicalDevice,
                                              pMemoryProperties);
  clone(state_block_, pMemoryProperties->memoryProperties,
        memory_properties, &mem);
}

VkResult minimal_state_tracker::vkAllocateMemory(VkDevice device,
                                                 const VkMemoryAllocateInfo* pAllocateInfo,
                                                 const VkAllocationCallbacks* pAllocator,
                                                 VkDeviceMemory* pMemory) {
  auto res =
      super::vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
  if (res != VK_SUCCESS) {
    return res;
  }
  auto new_mem = state_block_->get(*pMemory);
  auto type = memory_properties.memoryTypes[pAllocateInfo->memoryTypeIndex];
  new_mem->_is_coherent =
      (type.propertyFlags & VK_MEMORY_PROPERTY_HOST_COHERENT_BIT) != 0;
  new_mem->_size = pAllocateInfo->allocationSize;
  return res;
}

VkResult minimal_state_tracker::vkMapMemory(VkDevice device,
                                            VkDeviceMemory memory,
                                            VkDeviceSize offset,
                                            VkDeviceSize size,
                                            VkMemoryMapFlags flags,
                                            void** ppData) {
  auto res = super::vkMapMemory(device, memory, offset, size, flags, ppData);
  if (res != VK_SUCCESS) {
    return res;
  }
  auto new_mem = state_block_->get(memory);
  if (size == VK_WHOLE_SIZE) {
    size = new_mem->_size - offset;
  }
  size = size > new_mem->_size - offset ? new_mem->_size - offset : size;
  new_mem->_mapped_location = reinterpret_cast<char*>(ppData[0]);
  new_mem->_mapped_offset = offset;
  new_mem->_mapped_size = size;
  return res;
}

void minimal_state_tracker::vkUnmapMemory(VkDevice device, VkDeviceMemory memory) {
  auto new_mem = state_block_->get(memory);
  new_mem->_mapped_location = nullptr;
  super::vkUnmapMemory(device, memory);
}

VkResult minimal_state_tracker::vkCreateDescriptorUpdateTemplate(
    VkDevice device,
    const VkDescriptorUpdateTemplateCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkDescriptorUpdateTemplate* pDescriptorUpdateTemplate) {
  auto res = super::vkCreateDescriptorUpdateTemplate(
      device, pCreateInfo, pAllocator, pDescriptorUpdateTemplate);
  if (res != VK_SUCCESS) {
    return res;
  }
  auto pl = state_block_->get(pDescriptorUpdateTemplate[0]);
  pl->set_create_info(state_block_, pCreateInfo);
  return res;
}

VkResult minimal_state_tracker::vkBeginCommandBuffer(
    VkCommandBuffer commandBuffer,
    const VkCommandBufferBeginInfo* pBeginInfo) {
  auto res = super::vkBeginCommandBuffer(commandBuffer, pBeginInfo);
  if (res != VK_SUCCESS) {
    return res;
  }
  auto cb = state_block_->get(commandBuffer);
  cb->_pre_run_functions.clear();
  cb->_post_run_functions.clear();
  return res;
}

VkResult minimal_state_tracker::vkQueueSubmit(VkQueue queue,
                                              uint32_t submitCount,
                                              const VkSubmitInfo* pSubmits,
                                              VkFence fence) {
  std::vector<VkCommandBuffer> cbs;
  for (size_t i = 0; i < submitCount; ++i) {
    for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
      cbs.push_back(pSubmits[i].pCommandBuffers[j]);
      auto cb = state_block_->get(pSubmits[i].pCommandBuffers[j]);
      for (auto& pf : cb->_pre_run_functions) {
        pf();
      }
    }
  }

  auto res = super::vkQueueSubmit(queue, submitCount, pSubmits, fence);
  if (res != VK_SUCCESS) {
    return res;
  }
  for (size_t i = 0; i < submitCount; ++i) {
    for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
      cbs.push_back(pSubmits[i].pCommandBuffers[j]);
      auto cb = state_block_->get(pSubmits[i].pCommandBuffers[j]);
      for (auto& pf : cb->_post_run_functions) {
        pf();
      }
    }
  }

  return res;
}

}  // namespace gapid2