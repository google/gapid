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

#include "spy.h"

#include "command_buffer.h"
#include "device_memory.h"
#include "encoder.h"
#include "state_block.h"

namespace gapid2 {

void* spy::get_allocation(size_t i) {
  i = (i + 4095) & ~4096;
  void* v = VirtualAlloc(nullptr, i, MEM_RESERVE | MEM_COMMIT | MEM_WRITE_WATCH, PAGE_READWRITE);
  return v;
}
void spy::foreach_write(void*) {
}

void spy::reset_memory_watch() {
  std::unique_lock<std::mutex> l(memory_mutex);
  std::shared_lock<std::shared_mutex> l2(memory_alloc_info_mutex);
  for (auto m : mapped_coherent_memories) {
    auto new_mem = state_block_->get(m);
    auto& nn = memory_infos[m];
    ULONG_PTR l = nn.dirty_page_cache.size();
    DWORD ps = 0;
    GetWriteWatch(WRITE_WATCH_FLAG_RESET, nn.v1, nn.size, nn.dirty_page_cache.data(), &l, &ps);
  }
}

VkResult spy::vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkInstance* pInstance) {
  bool has_physical_device_properties2 = false;
  std::vector<const char*> exts(pCreateInfo->enabledExtensionCount);
  for (size_t i = 0; i < exts.size(); ++i) {
    exts[i] = pCreateInfo->ppEnabledExtensionNames[i];
    if (!strcmp(exts[i], "VK_KHR_external_memory_capabilities")) {
      has_external_memory_capabilities = true;
    }
    if (!strcmp(exts[i], "VK_KHR_get_physical_device_properties2")) {
      has_physical_device_properties2 = true;
    }
  }
  VkInstanceCreateInfo nci = *pCreateInfo;
  const VkInstanceCreateInfo* ci = pCreateInfo;
  if (!has_external_memory_capabilities) {
#pragma TODO(awoloszyn, We should really enable the following code but the loader is broken here)
    /*do {
      uint32_t property_count;
      if (VK_SUCCESS != bypass_caller->vkEnumerateInstanceExtensionProperties("", &property_count, nullptr)) {
        break;
      }
      std::vector<VkExtensionProperties> props(property_count);
      if (VK_SUCCESS != bypass_caller->vkEnumerateInstanceExtensionProperties(nullptr, &property_count, props.data())) {
        break;
      }

      for (auto& i : props) {
        if (!strcmp(i.extensionName, "VK_KHR_external_memory_capabilities")) {
          exts.push_back("VK_KHR_external_memory_capabilities");
          nci = *pCreateInfo;
          nci.enabledExtensionCount += 1;
          nci.ppEnabledExtensionNames = exts.data();
          ci = &nci;
          has_external_memory_capabilities = true;
          continue;
        }
      }
    } while (false);*/
    // We can't actually check for the extension :(
    exts.push_back("VK_KHR_external_memory_capabilities");
    nci.enabledExtensionCount += 1;
    nci.ppEnabledExtensionNames = exts.data();
    ci = &nci;
    has_external_memory_capabilities = true;
  }
  if (!has_physical_device_properties2) {
    has_physical_device_properties2 = true;
    exts.push_back("VK_KHR_get_physical_device_properties2");
    nci.enabledExtensionCount += 1;
    nci.ppEnabledExtensionNames = exts.data();
    ci = &nci;
  }

  if (!has_external_memory_capabilities) {
    OutputDebugStringA("Cannot use VK_KHR_external_memory_capabilities so memory tracking will be less efficient\n");
  } else {
    OutputDebugStringA("Using VK_KHR_external_memory_capabilities. This will cause slight performance inaccuracies, but increase trace performance\n");
  }
  return super::vkCreateInstance(ci, pAllocator, pInstance);
}

VkResult spy::vkCreateDevice(VkPhysicalDevice physicalDevice, const VkDeviceCreateInfo* pCreateInfo, const VkAllocationCallbacks* pAllocator, VkDevice* pDevice) {
  std::vector<const char*> exts(pCreateInfo->enabledExtensionCount);
  bool has_external_memory_host = false;
  for (size_t i = 0; i < exts.size(); ++i) {
    exts[i] = pCreateInfo->ppEnabledExtensionNames[i];
    if (!strcmp(exts[i], "VK_EXT_external_memory_host")) {
      has_external_memory_host = true;
    }
    if (!strcmp(exts[i], "VK_KHR_external_memory")) {
      has_external_memory = true;
    }
  }
  VkDeviceCreateInfo nci = *pCreateInfo;
  const VkDeviceCreateInfo* ci = pCreateInfo;
  if (!has_external_memory_host || !has_external_memory) {
    do {
      uint32_t property_count;
      if (VK_SUCCESS != bypass_caller->vkEnumerateDeviceExtensionProperties(physicalDevice, nullptr, &property_count, nullptr)) {
        break;
      }
      std::vector<VkExtensionProperties> props(property_count);
      if (VK_SUCCESS != bypass_caller->vkEnumerateDeviceExtensionProperties(physicalDevice, nullptr, &property_count, props.data())) {
        break;
      }

      for (auto& i : props) {
        if (!strcmp(i.extensionName, "VK_EXT_external_memory_host") && !has_external_memory_host) {
          exts.push_back("VK_EXT_external_memory_host");
          nci.enabledExtensionCount += 1;
          nci.ppEnabledExtensionNames = exts.data();
          ci = &nci;
          has_external_memory_host = true;
          continue;
        }
        if (!strcmp(i.extensionName, "VK_KHR_external_memory")) {
          exts.push_back("VK_KHR_external_memory");
          nci.enabledExtensionCount += 1;
          nci.ppEnabledExtensionNames = exts.data();
          ci = &nci;
          has_external_memory = true;
          continue;
        }
      }
    } while (false);
  }

  if (!has_external_memory_host) {
    OutputDebugStringA("Cannot use VK_EXT_external_memory_host so memory tracking will be less efficient\n");
  } else {
    OutputDebugStringA("Using VK_EXT_external_memory_host. This will cause slight performance inaccuracies, but increase trace performance\n");
  }
  if (!has_external_memory) {
    OutputDebugStringA("Cannot use VK_KHR_external_memory so memory tracking will be less efficient\n");
  } else {
    OutputDebugStringA("Using VK_KHR_external_memory. This will cause slight performance inaccuracies, but increase trace performance\n");
  }
  VkResult ret;

  if (ci != pCreateInfo) {
    ret = bypass_caller->vkCreateDevice(physicalDevice, ci, pAllocator, pDevice);
    noop_serializer.vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
  } else {
    ret = super::vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
  }
  if (ret != VK_SUCCESS) {
    return ret;
  }
  has_external_memory_host_[*pDevice] = has_external_memory_host;

  if (has_external_memory_host) {
    auto t = get_allocation(4096);

    VkMemoryHostPointerPropertiesEXT host_pointer_properties;
    host_pointer_properties.sType = VK_STRUCTURE_TYPE_MEMORY_HOST_POINTER_PROPERTIES_EXT;
    host_pointer_properties.pNext = nullptr;
    // Try to allocate a host pointer the same way we would in the memory tracker.
    auto ret = bypass_caller->vkGetMemoryHostPointerPropertiesEXT(pDevice[0], VK_EXTERNAL_MEMORY_HANDLE_TYPE_HOST_ALLOCATION_BIT_EXT, t, &host_pointer_properties);
    VirtualFree(t, 0, MEM_RELEASE);

    if (ret != VK_SUCCESS) {
      GAPID2_ERROR("Could not determine pointer properties");
    }
    VkPhysicalDeviceMemoryProperties dev_mem_props;
    bypass_caller->vkGetPhysicalDeviceMemoryProperties(physicalDevice, &dev_mem_props);

    uint32_t memory_type = 0;
    uint32_t memory_type_bits = host_pointer_properties.memoryTypeBits;
    bool has_host_coherent = false;
    bool has_host_visible = false;
    uint32_t valid_memory = 0;
    while (memory_type_bits) {
      // If this is host_visible memory then make sure we can acutally use a host pointer for it
      if (memory_type_bits & 0x1) {
        bool hc = (dev_mem_props.memoryTypes[memory_type].propertyFlags & VK_MEMORY_PROPERTY_HOST_COHERENT_BIT) != 0;
        bool hv = (dev_mem_props.memoryTypes[memory_type].propertyFlags & VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) != 0;
        bool hcc = (dev_mem_props.memoryTypes[memory_type].propertyFlags & VK_MEMORY_PROPERTY_HOST_CACHED_BIT) != 0;
        has_host_coherent |= hc;
        has_host_visible |= hv;
        // Remove all non host_cached memory for efficiency;
        valid_memory |= ((hc | hv) & !hcc) << memory_type;
      } else {
        // If this is NOT host_visible memory then we are good
        if (!(dev_mem_props.memoryTypes[memory_type].propertyFlags & VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT)) {
          valid_memory |= 1 << memory_type;
        }
      }
      ++memory_type;
      memory_type_bits >>= 1;
    }
    if (!has_host_coherent) {
      OutputDebugStringA("Not VK_EXT_external_memory_host in the end, could not find requisite HOST_COHERENT heap");
    } else {
      std::unique_lock<std::shared_mutex>(dev_info_mutex);
      dev_infos.insert(
          std::make_pair(
              pDevice[0], dev_info{
                              .valid_memory_types = valid_memory,
                              .dev_mem_props = dev_mem_props}));
    }
  }

  return ret;
}

void spy::vkGetImageMemoryRequirements(VkDevice device, VkImage image, VkMemoryRequirements* pMemoryRequirements) {
  super::vkGetImageMemoryRequirements(device, image, pMemoryRequirements);
  std::shared_lock<std::shared_mutex> l(dev_info_mutex);
  auto it = dev_infos.find(device);
  if (it != dev_infos.end()) {
    pMemoryRequirements->memoryTypeBits &= it->second.valid_memory_types;
  }
  GAPID2_ASSERT(pMemoryRequirements->memoryTypeBits != 0, "No valid place to put this now :|");
}

void spy::vkGetBufferMemoryRequirements(VkDevice device, VkBuffer buffer, VkMemoryRequirements* pMemoryRequirements) {
  super::vkGetBufferMemoryRequirements(device, buffer, pMemoryRequirements);
  std::shared_lock<std::shared_mutex> l(dev_info_mutex);
  auto it = dev_infos.find(device);
  if (it != dev_infos.end()) {
    pMemoryRequirements->memoryTypeBits &= it->second.valid_memory_types;
  }
  GAPID2_ASSERT(pMemoryRequirements->memoryTypeBits != 0, "No valid place to put this now :|");
}

void spy::vkGetImageMemoryRequirements2(VkDevice device, const VkImageMemoryRequirementsInfo2* pInfo, VkMemoryRequirements2* pMemoryRequirements) {
  super::vkGetImageMemoryRequirements2(device, pInfo, pMemoryRequirements);
  std::shared_lock<std::shared_mutex> l(dev_info_mutex);
  auto it = dev_infos.find(device);
  if (it != dev_infos.end()) {
    pMemoryRequirements->memoryRequirements.memoryTypeBits &= it->second.valid_memory_types;
  }
  GAPID2_ASSERT(pMemoryRequirements->memoryRequirements.memoryTypeBits != 0, "No valid place to put this now :|");
}

void spy::vkGetBufferMemoryRequirements2(VkDevice device, const VkBufferMemoryRequirementsInfo2* pInfo, VkMemoryRequirements2* pMemoryRequirements) {
  super::vkGetBufferMemoryRequirements2(device, pInfo, pMemoryRequirements);
  std::shared_lock<std::shared_mutex> l(dev_info_mutex);
  auto it = dev_infos.find(device);
  if (it != dev_infos.end()) {
    pMemoryRequirements->memoryRequirements.memoryTypeBits &= it->second.valid_memory_types;
  }
  GAPID2_ASSERT(pMemoryRequirements->memoryRequirements.memoryTypeBits != 0, "No valid place to put this now :|");
}

VkResult spy::vkCreateBuffer(VkDevice device,
                             const VkBufferCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkBuffer* pBuffer) {
  VkExternalMemoryBufferCreateInfo embci = {
      .sType = VK_STRUCTURE_TYPE_EXTERNAL_MEMORY_BUFFER_CREATE_INFO,
      .pNext = pCreateInfo->pNext,
      .handleTypes = VK_EXTERNAL_MEMORY_HANDLE_TYPE_HOST_ALLOCATION_BIT_EXT};

  auto ci = *pCreateInfo;
  ci.pNext = &embci;
  return super::vkCreateBuffer(device, &ci, pAllocator, pBuffer);
}

VkResult spy::vkAllocateMemory(VkDevice device, const VkMemoryAllocateInfo* pAllocateInfo, const VkAllocationCallbacks* pAllocator, VkDeviceMemory* pMemory) {
  bool special = false;
  std::shared_lock<std::shared_mutex>(dev_info_mutex);
  auto it = dev_infos.find(device);
  if (it != dev_infos.end()) {
    GAPID2_ASSERT((it->second.valid_memory_types & (1 << pAllocateInfo->memoryTypeIndex)),
                  "Application is allocating a piece of memory that can never be used");
    if (it->second.dev_mem_props.memoryTypes[pAllocateInfo->memoryTypeIndex].propertyFlags & VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
      special = true;
    }
  }

  if (!special) {
    return super::vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
  }

  auto allocationSize = (pAllocateInfo->allocationSize + 4095) & ~4095;
  void* t = get_allocation(allocationSize);
  VkImportMemoryHostPointerInfoEXT import{
      VK_STRUCTURE_TYPE_IMPORT_MEMORY_HOST_POINTER_INFO_EXT,
      pAllocateInfo->pNext,
      VK_EXTERNAL_MEMORY_HANDLE_TYPE_HOST_ALLOCATION_BIT_EXT,
      t};

  VkMemoryAllocateInfo inf = *pAllocateInfo;
  inf.pNext = &import;
  inf.allocationSize = allocationSize;

  auto ret = bypass_caller->vkAllocateMemory(device, &inf, pAllocator, pMemory);
  noop_serializer.vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
  if (ret == VK_SUCCESS) {
    std::unique_lock<std::shared_mutex> l(memory_alloc_info_mutex);
    memory_infos.insert(std::make_pair(pMemory[0], memory_info{
                                                       .v1 = t,
                                                       .size = allocationSize,
                                                       .dirty_page_cache = std::vector<void*>(allocationSize / 4096)}));
  }
  return ret;
}

VkResult spy::vkMapMemory(VkDevice device,
                          VkDeviceMemory memory,
                          VkDeviceSize offset,
                          VkDeviceSize size,
                          VkMemoryMapFlags flags,
                          void** ppData) {
  memory_info mi;
  bool special = false;
  {
    std::shared_lock<std::shared_mutex> l(memory_alloc_info_mutex);
    auto it = memory_infos.find(memory);
    if (it != memory_infos.end()) {
      special = true;
      mi = it->second;
    }
  }

  if (special) {
    auto res = super::vkMapMemory(device, memory, offset, size, flags, ppData);
    if (res != VK_SUCCESS) {
      return res;
    }
    // Take over the memory mapping :D
    ppData[0] = reinterpret_cast<char*>(mi.v1) + offset;

    noop_serializer.vkMapMemory(device, memory, offset, size, flags, ppData);

    std::unique_lock<std::mutex> l(memory_mutex);
    auto new_mem = state_block_->get(memory);
    new_mem->_mapped_location = reinterpret_cast<char*>(ppData[0]);
    if (size == VK_WHOLE_SIZE) {
      size = new_mem->_size - offset;
    }
    // tracker.AddTrackedRange(memory, reinterpret_cast<char*>(mi.v1) + offset, offset, size, reinterpret_cast<char*>(mi.v2) + offset);
    if (new_mem->_is_coherent) {
      mapped_coherent_memories.insert(memory);
    }
    return VK_SUCCESS;
  } else {
    auto res = super::vkMapMemory(device, memory, offset, size, flags, ppData);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_mem = state_block_->get(memory);
    if (size == VK_WHOLE_SIZE) {
      size = new_mem->_size - offset;
    }
    ppData[0] = tracker.AddTrackedRange(memory, ppData[0], offset, size);
    std::unique_lock<std::mutex> l(memory_mutex);
    if (new_mem->_is_coherent) {
      mapped_coherent_memories.insert(memory);
    }
    return res;
  }
}

VkResult spy::vkEnumeratePhysicalDevices(
    VkInstance instance,
    uint32_t* pPhysicalDeviceCount,
    VkPhysicalDevice* pPhysicalDevices) {
  auto ret = super::vkEnumeratePhysicalDevices(instance, pPhysicalDeviceCount,
                                               pPhysicalDevices);
  if (ret != VK_SUCCESS) {
    return ret;
  }
  if (pPhysicalDevices) {
    auto enc = encoding_serializer_->get_encoder(reinterpret_cast<uintptr_t>(instance));
    if (enc) {
      for (size_t i = 0; i < *pPhysicalDeviceCount; ++i) {
        VkPhysicalDeviceProperties properties;
        // Bypass serializing the call to GPDP
        bypass_caller->vkGetPhysicalDeviceProperties(pPhysicalDevices[i], &properties);
        enc->encode<uint32_t>(properties.deviceID);
        enc->encode<uint32_t>(properties.vendorID);
        enc->encode<uint32_t>(properties.driverVersion);
      }
    }
  }
  return ret;
}

void spy::vkUnmapMemory(VkDevice device, VkDeviceMemory memory) {
  tracker.RemoveTrackedRange(memory);
  std::unique_lock<std::mutex> l(memory_mutex);
  mapped_coherent_memories.erase(memory);
  super::vkUnmapMemory(device, memory);
}

void spy::vkFreeMemory(VkDevice device,
                       VkDeviceMemory memory,
                       const VkAllocationCallbacks* pAllocator) {
  {
    std::unique_lock<std::shared_mutex> l(memory_alloc_info_mutex);
    memory_infos.erase(memory);
  }

  auto new_mem = state_block_->get(memory);
  if (new_mem->_mapped_location) {
    tracker.RemoveTrackedRange(memory);
  }
  {
    std::unique_lock<std::mutex> l(memory_mutex);
    mapped_coherent_memories.erase(memory);
  }
  super::vkFreeMemory(device, memory, pAllocator);
}

VkResult spy::vkFlushMappedMemoryRanges(
    VkDevice device,
    uint32_t memoryRangeCount,
    const VkMappedMemoryRange* pMemoryRanges) {
  auto res = super::vkFlushMappedMemoryRanges(device, memoryRangeCount,
                                              pMemoryRanges);
  std::unique_lock<std::mutex> l(memory_mutex);
  std::shared_lock<std::shared_mutex> l2(memory_alloc_info_mutex);
  auto enc = encoding_serializer_->get_encoder(0);
  if (enc) {
    for (uint32_t i = 0; i < memoryRangeCount; ++i) {
      auto& mr = pMemoryRanges[i];
      auto new_mem = state_block_->get(mr.memory);
      auto& nn = memory_infos[mr.memory];
      ULONG_PTR l = nn.dirty_page_cache.size();
      DWORD ps = 0;
      GetWriteWatch(WRITE_WATCH_FLAG_RESET, nn.v1, nn.size, nn.dirty_page_cache.data(), &l, &ps);
      for (size_t i = 0; i < l; ++i) {
        auto offset =
            reinterpret_cast<char*>(nn.dirty_page_cache[i]) - new_mem->_mapped_location;
        enc->encode<uint64_t>(0);
        enc->encode<uint64_t>(encoding_serializer_->get_flags());
        enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(mr.memory));
        enc->encode<uint64_t>(offset);
        enc->encode<uint64_t>(4096);
        enc->encode_primitive_array<char>(
            reinterpret_cast<const char*>(nn.dirty_page_cache[i]), 4096);
        // reset the encoder to flush the write
        enc = encoding_serializer_->get_encoder(0);
      }
    }
  }
  return res;
}

VkResult spy::vkInvalidateMappedMemoryRanges(
    VkDevice device,
    uint32_t memoryRangeCount,
    const VkMappedMemoryRange* pMemoryRanges) {
  auto res = super::vkInvalidateMappedMemoryRanges(device, memoryRangeCount,
                                                   pMemoryRanges);
  for (uint32_t i = 0; i < memoryRangeCount; ++i) {
    auto& mr = pMemoryRanges[i];
    auto new_mem = state_block_->get(mr.memory);
    auto sz = mr.size;
    if (sz == VK_WHOLE_SIZE) {
      sz = new_mem->allocate_info->allocationSize - mr.offset;
    }
    tracker.InvalidateMappedRange(mr.memory, mr.offset, sz);
  }
  return res;
}

VkResult spy::vkQueueSubmit(VkQueue queue,
                            uint32_t submitCount,
                            const VkSubmitInfo* pSubmits,
                            VkFence fence) {
  for (size_t i = 0; i < submitCount; ++i) {
    for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
      auto cb = state_block_->get(pSubmits[i].pCommandBuffers[j]);
      cb->_pre_run_functions.push_back([this](VkQueue) {
        auto enc = encoding_serializer_->get_encoder(0);
        if (!enc) {
          return;
        }
        std::unique_lock<std::mutex> l(memory_mutex);
        std::shared_lock<std::shared_mutex> l2(memory_alloc_info_mutex);
        for (auto m : mapped_coherent_memories) {
          auto new_mem = state_block_->get(m);
          auto& nn = memory_infos[m];
          ULONG_PTR l = nn.dirty_page_cache.size();
          DWORD ps = 0;
          GetWriteWatch(WRITE_WATCH_FLAG_RESET, nn.v1, nn.size, nn.dirty_page_cache.data(), &l, &ps);
          for (size_t i = 0; i < l; ++i) {
            auto offset =
                reinterpret_cast<char*>(nn.dirty_page_cache[i]) - new_mem->_mapped_location;
            enc->encode<uint64_t>(0);
            enc->encode<uint64_t>(encoding_serializer_->get_flags());
            enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(m));
            enc->encode<uint64_t>(offset);
            enc->encode<uint64_t>(4096);
            enc->encode_primitive_array<char>(
                reinterpret_cast<const char*>(nn.dirty_page_cache[i]), 4096);
            // reset the encoder to flush
            enc = encoding_serializer_->get_encoder(0);
          }
        }
      });
    }
  }
  auto res = super::vkQueueSubmit(queue, submitCount, pSubmits, fence);
  if (res != VK_SUCCESS) {
    return res;
  }

  return res;
}

VkResult spy::vkDeviceWaitIdle(VkDevice device) {
  auto res = super::vkDeviceWaitIdle(device);
  if (res == VK_SUCCESS) {
    return res;
  }
  return res;
}

VkResult spy::vkWaitForFences(VkDevice device,
                              uint32_t fenceCount,
                              const VkFence* pFences,
                              VkBool32 waitAll,
                              uint64_t timeout) {
  auto res =
      super::vkWaitForFences(device, fenceCount, pFences, waitAll, timeout);
  if (res == VK_TIMEOUT) {
    return res;
  }
  if (fenceCount == 1) {
    return res;
  }
  auto enc = encoding_serializer_->get_encoder(reinterpret_cast<uintptr_t>(device));
  if (enc) {
    for (uint32_t i = 0; i < fenceCount; ++i) {
      // Bypass serializing the call to GPDP
      if (bypass_caller->vkGetFenceStatus(device, pFences[i]) == VK_SUCCESS) {
        enc->encode<char>(1);
      } else {
        enc->encode<char>(0);
      }
    }
  }
  return res;
}

}  // namespace gapid2