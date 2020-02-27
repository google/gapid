/*
 * Copyright (C) 2020 Google Inc.
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

#include "gapii/cc/vulkan_spy.h"

#ifndef GAPII_VULKAN_EXTERNAL_MEMORY_H_
#define GAPII_VULKAN_EXTERNAL_MEMORY_H_

namespace gapii {

struct ExternalBufferMemoryStaging {
  VkBuffer buffer;
  VkBufferMemoryBarrier barrier;
  VkBufferCopy copy;
  inline ExternalBufferMemoryStaging(const VkBufferMemoryBarrier& barrier,
                                     VkDeviceSize& stagingOffset)
      : buffer(barrier.mbuffer),
        barrier(barrier),
        copy(barrier.moffset, stagingOffset, barrier.msize) {
    stagingOffset += barrier.msize;
  }
};

struct ExternalImageMemoryStaging {
  VkImage image;
  VkImageMemoryBarrier barrier;
  std::vector<VkBufferImageCopy> copies;
  inline ExternalImageMemoryStaging(const VkImageMemoryBarrier& barrier)
      : image(barrier.mimage), barrier(barrier) {}
};

struct ExternalMemoryCommandBuffer {
  std::vector<ExternalBufferMemoryStaging> buffers;
  std::vector<ExternalImageMemoryStaging> images;
  VkCommandBuffer commandBuffer = 0;
  VkCommandBuffer stagingCommandBuffer = 0;
  inline bool empty() const { return buffers.empty() && images.empty(); }
};

struct ExternalMemorySubmitInfo {
  const VkSubmitInfo* submitInfo;
  std::vector<ExternalMemoryCommandBuffer> commandBuffers;
};

struct ExternalMemoryStaging {
  VulkanSpy* spy = nullptr;
  CallObserver* observer = nullptr;
  VkQueue queue = 0;
  uint32_t queueFamilyIndex = 0;
  VkFence origFence = 0;
  VkFence stagingFence = 0;
  VkDevice device = 0;
  const VulkanImports::VkDeviceFunctions* fn = nullptr;

  std::vector<ExternalMemorySubmitInfo> submits;

  VkBuffer stagingBuffer = 0;
  VkDeviceMemory stagingMemory = 0;
  VkDeviceSize stagingSize = 0;
  VkCommandPool stagingCommandPool = 0;
  VkCommandBuffer stagingCommandBuffer = 0;

  ExternalMemoryStaging(VulkanSpy* spy, CallObserver* observer, VkQueue queue,
                        uint32_t submitCount, const VkSubmitInfo* pSubmits,
                        VkFence fence);
  inline ~ExternalMemoryStaging() { Cleanup(); }
  uint32_t CreateResources();
  uint32_t RecordCommandBuffers();
  uint32_t RecordStagingCommandBuffer(
      const ExternalMemoryCommandBuffer& cmdBuf);
  uint32_t Submit();
  void SendData();
  void Cleanup();
};

}  // namespace gapii

#endif  // GAPII_VULKAN_EXTERNAL_MEMORY_H_
