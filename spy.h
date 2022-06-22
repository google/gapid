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

#include <fstream>
#include <mutex>
#include <shared_mutex>
#include <unordered_set>

#include "command_serializer.h"
#include "encoder.h"
#include "memory_tracker.h"
#include "noop_serializer.h"
#include "null_caller.h"
#include "temporary_allocator.h"
#include "transform.h"

namespace gapid2 {
class spy : public transform_base {
 public:
  using super = transform_base;
  spy() : null_caller_(&empty_), noop_serializer(&empty_) {
  }

  void initialize(command_serializer* encoder_, transform_base* bypass_caller_) {
    bypass_caller = bypass_caller_;
    noop_serializer.encoder = encoder_;
    noop_serializer.state_block_ = state_block_;
    encoding_serializer_ = encoder_;
  }

  VkResult vkMapMemory(VkDevice device,
                       VkDeviceMemory memory,
                       VkDeviceSize offset,
                       VkDeviceSize size,
                       VkMemoryMapFlags flags,
                       void** ppData) override;
  VkResult vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo, const VkAllocationCallbacks* pAllocator, VkInstance* pInstance) override;
  VkResult vkCreateBuffer(VkDevice device,
                          const VkBufferCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkBuffer* pBuffer) override;
  VkResult vkCreateDevice(VkPhysicalDevice physicalDevice, const VkDeviceCreateInfo* pCreateInfo, const VkAllocationCallbacks* pAllocator, VkDevice* pDevice) override;
  VkResult vkEnumeratePhysicalDevices(
      VkInstance instance,
      uint32_t* pPhysicalDeviceCount,
      VkPhysicalDevice* pPhysicalDevices) override;

  void vkUnmapMemory(VkDevice device, VkDeviceMemory memory) override;

  void vkFreeMemory(VkDevice device,
                    VkDeviceMemory memory,
                    const VkAllocationCallbacks* pAllocator) override;

  VkResult vkFlushMappedMemoryRanges(
      VkDevice device,
      uint32_t memoryRangeCount,
      const VkMappedMemoryRange* pMemoryRanges) override;

  VkResult vkInvalidateMappedMemoryRanges(
      VkDevice device,
      uint32_t memoryRangeCount,
      const VkMappedMemoryRange* pMemoryRanges) override;

  VkResult vkQueueSubmit(VkQueue queue,
                         uint32_t submitCount,
                         const VkSubmitInfo* pSubmits,
                         VkFence fence) override;

  VkResult vkDeviceWaitIdle(VkDevice device) override;

  VkResult vkWaitForFences(VkDevice device,
                           uint32_t fenceCount,
                           const VkFence* pFences,
                           VkBool32 waitAll,
                           uint64_t timeout) override;

  void vkGetImageMemoryRequirements(VkDevice device, VkImage image, VkMemoryRequirements* pMemoryRequirements);
  void vkGetBufferMemoryRequirements(VkDevice device, VkBuffer buffer, VkMemoryRequirements* pMemoryRequirements);
  void vkGetImageMemoryRequirements2(VkDevice device, const VkImageMemoryRequirementsInfo2* pInfo, VkMemoryRequirements2* pMemoryRequirements);
  void vkGetBufferMemoryRequirements2(VkDevice device, const VkBufferMemoryRequirementsInfo2* pInfo, VkMemoryRequirements2* pMemoryRequirements);

  void* get_allocation(size_t i);
  void foreach_write(void*);

  VkResult vkAllocateMemory(VkDevice device, const VkMemoryAllocateInfo* pAllocateInfo, const VkAllocationCallbacks* pAllocator, VkDeviceMemory* pMemory);

  void reset_memory_watch();

 private:
  std::unordered_set<VkInstance> instances;
  std::mutex memory_mutex;
  std::unordered_set<VkDeviceMemory> mapped_coherent_memories;
  std::mutex map_mutex;
  temporary_allocator allocator;
  memory_tracker tracker;
  std::unordered_map<VkDevice, bool> has_external_memory_host_;
  bool has_external_memory = false;
  bool has_external_memory_capabilities = false;
  std::shared_mutex dev_info_mutex;
  struct dev_info {
    uint32_t valid_memory_types;
    VkPhysicalDeviceMemoryProperties dev_mem_props;
  };

  std::unordered_map<VkDevice, dev_info> dev_infos;

  std::shared_mutex memory_alloc_info_mutex;
  struct memory_info {
    HANDLE h;
    void* v1;
    void* v2;
    size_t size;
    std::vector<void*> dirty_page_cache;
  };
  std::unordered_map<VkDeviceMemory, memory_info> memory_infos;
  transform_base empty_;
  transform<null_caller> null_caller_;
  transform<noop_serializer> noop_serializer;
  transform_base* bypass_caller;
  command_serializer* encoding_serializer_;
};
}  // namespace gapid2