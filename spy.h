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
#include <unordered_set>

#include "command_serializer.h"
#include "encoder.h"
#include "memory_tracker.h"
#include "temporary_allocator.h"

namespace gapid2 {
class spy : public command_serializer {
 public:
  using super = command_serializer;
  spy() : out_file("file.trace", std::ios::out | std::ios::binary) {
    encoder_tls_key = TlsAlloc();
  }

  VkResult vkMapMemory(VkDevice device,
                       VkDeviceMemory memory,
                       VkDeviceSize offset,
                       VkDeviceSize size,
                       VkMemoryMapFlags flags,
                       void** ppData) override;
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

  encoder_handle get_locked_encoder(uintptr_t key) override;
  encoder_handle get_encoder(uintptr_t key) override;

 private:
  std::unordered_set<VkInstance> instances;
  std::mutex memory_mutex;
  std::unordered_set<VkDeviceMemory> mapped_coherent_memories;
  std::mutex map_mutex;
  std::mutex call_mutex;
  temporary_allocator allocator;
  DWORD encoder_tls_key;
  std::fstream out_file;
  memory_tracker tracker;
};
}  // namespace gapid2