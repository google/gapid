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

namespace gapid2 {
template <typename T>
class MinimalStateTracker : public T {
 protected:
  using super = T;

 public:
  void vkGetPhysicalDeviceMemoryProperties(
      VkPhysicalDevice physicalDevice,
      VkPhysicalDeviceMemoryProperties* pMemoryProperties) override {
    super::vkGetPhysicalDeviceMemoryProperties(physicalDevice,
                                               pMemoryProperties);
    clone<NullCloner>(&cloner, pMemoryProperties[0], memory_properties, &mem);
  }
  void vkGetPhysicalDeviceMemoryProperties2(
      VkPhysicalDevice physicalDevice,
      VkPhysicalDeviceMemoryProperties2* pMemoryProperties) override {
    super::vkGetPhysicalDeviceMemoryProperties2(physicalDevice,
                                                pMemoryProperties);
    clone<NullCloner>(&cloner, pMemoryProperties->memoryProperties,
                      memory_properties, &mem);
  }

  VkResult vkAllocateMemory(VkDevice device,
                            const VkMemoryAllocateInfo* pAllocateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkDeviceMemory* pMemory) override {
    auto res =
        super::vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_mem = this->updater_.cast_from_vk(*pMemory);
    auto type = memory_properties.memoryTypes[pAllocateInfo->memoryTypeIndex];
    new_mem->_is_coherent =
        (type.propertyFlags & VK_MEMORY_PROPERTY_HOST_COHERENT_BIT) != 0;
    new_mem->_size = pAllocateInfo->allocationSize;
    return res;
  }

  VkResult vkMapMemory(VkDevice device,
                       VkDeviceMemory memory,
                       VkDeviceSize offset,
                       VkDeviceSize size,
                       VkMemoryMapFlags flags,
                       void** ppData) override {
    auto res = super::vkMapMemory(device, memory, offset, size, flags, ppData);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_mem = this->updater_.cast_from_vk(memory);
    if (size == VK_WHOLE_SIZE) {
      size = new_mem->_size - offset;
    }
    size = size > new_mem->_size - offset ? new_mem->_size - offset : size;
    new_mem->_mapped_location = reinterpret_cast<char*>(ppData[0]);
    new_mem->_mapped_offset = offset;
    new_mem->_mapped_size = size;
    return res;
  }

  void vkUnmapMemory(VkDevice device, VkDeviceMemory memory) override {
    auto new_mem = this->updater_.cast_from_vk(memory);
    new_mem->_mapped_location = nullptr;
    super::vkUnmapMemory(device, memory);
  }

  VkResult vkCreateDescriptorUpdateTemplate(
      VkDevice device,
      const VkDescriptorUpdateTemplateCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorUpdateTemplate* pDescriptorUpdateTemplate) override {
    auto res = super::vkCreateDescriptorUpdateTemplate(
        device, pCreateInfo, pAllocator, pDescriptorUpdateTemplate);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pDescriptorUpdateTemplate[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

 protected:
  VkPhysicalDeviceMemoryProperties memory_properties;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2