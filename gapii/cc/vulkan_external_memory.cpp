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

#include "gapii/cc/vulkan_external_memory.h"
#include "gapii/cc/vulkan_layer_extras.h"
#include "gapii/cc/vulkan_spy.h"
#include "gapis/api/vulkan/vulkan_pb/extras.pb.h"

namespace gapii {

void VulkanSpy::recordExternalBarriers(
    VkCommandBuffer commandBuffer, uint32_t bufferMemoryBarrierCount,
    const VkBufferMemoryBarrier* pBufferMemoryBarriers,
    uint32_t imageMemoryBarrierCount,
    const VkImageMemoryBarrier* pImageMemoryBarriers) {
  static const uint32_t VK_QUEUE_FAMILY_EXTERNAL = ~0U - 1;

  size_t externalBufferBarrierCount = 0;
  for (uint32_t i = 0; i < bufferMemoryBarrierCount; ++i) {
    if (pBufferMemoryBarriers[i].msrcQueueFamilyIndex ==
        VK_QUEUE_FAMILY_EXTERNAL) {
      ++externalBufferBarrierCount;
    }
  }

  size_t externalImageBarrierCount = 0;
  for (uint32_t i = 0; i < imageMemoryBarrierCount; ++i) {
    if (pImageMemoryBarriers[i].msrcQueueFamilyIndex ==
        VK_QUEUE_FAMILY_EXTERNAL) {
      ++externalImageBarrierCount;
    }
  }

  if (externalBufferBarrierCount == 0 && externalImageBarrierCount == 0) {
    return;
  }

  std::vector<VkBufferMemoryBarrier>& bufBarriers =
      mExternalBufferBarriers[commandBuffer];
  bufBarriers.reserve(bufBarriers.size() + externalBufferBarrierCount);
  for (uint32_t i = 0; i < bufferMemoryBarrierCount; ++i) {
    if (pBufferMemoryBarriers[i].msrcQueueFamilyIndex ==
        VK_QUEUE_FAMILY_EXTERNAL) {
      bufBarriers.push_back(pBufferMemoryBarriers[i]);
    }
  }
}

ExternalMemoryStaging::ExternalMemoryStaging(
    VulkanSpy* spy, CallObserver* observer, VkQueue queue, uint32_t submitCount,
    const VkSubmitInfo* pSubmits, VkFence fence)
    : spy(spy), observer(observer), queue(queue), origFence(fence) {
  QueueObject& queueObj = *spy->mState.Queues[queue];
  queueFamilyIndex = queueObj.mFamily;
  device = queueObj.mDevice;
  fn = &spy->mImports.mVkDeviceFunctions[device];

  stagingSize = 0;
  submits.resize(submitCount);
  for (uint32_t i = 0; i < submitCount; ++i) {
    submits[i].submitInfo = &pSubmits[i];
    uint32_t commandBufferCount = pSubmits[i].mcommandBufferCount;
    submits[i].commandBuffers.resize(commandBufferCount);
    for (uint32_t j = 0; j < commandBufferCount; ++j) {
      ExternalMemoryCommandBuffer& cmdBuf = submits[i].commandBuffers[j];
      cmdBuf.commandBuffer = pSubmits[i].mpCommandBuffers[j];
      auto bufIt = spy->mExternalBufferBarriers.find(cmdBuf.commandBuffer);
      if (bufIt != spy->mExternalBufferBarriers.end()) {
        for (auto barrierIt = bufIt->second.begin();
             barrierIt != bufIt->second.end(); ++barrierIt) {
          cmdBuf.buffers.push_back(
              ExternalBufferMemoryStaging(*barrierIt, stagingSize));
        }
      }
    }
  }
}

uint32_t ExternalMemoryStaging::CreateResources() {
  VkCommandPoolCreateInfo commandPoolCreateInfo{
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,  // sType
      nullptr,                                                      // pNext
      VkCommandPoolCreateFlagBits::
          VK_COMMAND_POOL_CREATE_TRANSIENT_BIT,  // flags
      queueFamilyIndex,                          // queueFamilyIndex
  };
  uint32_t res = fn->vkCreateCommandPool(device, &commandPoolCreateInfo,
                                         nullptr, &stagingCommandPool);
  if (res != VkResult::VK_SUCCESS) {
    stagingCommandPool = 0;
    GAPID_ERROR("Error creating command pool for external memory observations");
    return res;
  }

  VkFenceCreateInfo fenceCreateInfo{
      VkStructureType::VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,  // sType
      nullptr,                                               // pNext
      0,                                                     // flags
  };
  res = fn->vkCreateFence(device, &fenceCreateInfo, nullptr, &stagingFence);
  if (res != VkResult::VK_SUCCESS) {
    stagingFence = 0;
    GAPID_ERROR("Error creating fence for external memory observations");
    return res;
  }

  size_t commandBufferCount = 1;
  for (auto submitIt = submits.begin(); submitIt != submits.end(); ++submitIt) {
    for (auto cmdBufIt = submitIt->commandBuffers.begin();
         cmdBufIt != submitIt->commandBuffers.end(); ++cmdBufIt) {
      if (!cmdBufIt->empty()) {
        ++commandBufferCount;
      }
    }
  }
  std::vector<VkCommandBuffer> commandBuffers(commandBufferCount);
  VkCommandBufferAllocateInfo commandBufferAllocInfo{
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,  // sType
      nullptr,                                                          // pNext
      stagingCommandPool,                                     // commandPool
      VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY,  // level
      (uint32_t)commandBuffers.size() + 1,  // commandBufferCount
  };
  res = fn->vkAllocateCommandBuffers(device, &commandBufferAllocInfo,
                                     commandBuffers.data());
  for (auto cmdBuf : commandBuffers) {
    set_dispatch_from_parent((void*)cmdBuf, (void*)device);
  }
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR(
        "Error allocating command buffer for external memory observations");
    return res;
  }
  stagingCommandBuffer = commandBuffers.back();
  commandBuffers.pop_back();
  for (auto submitIt = submits.begin(); submitIt != submits.end(); ++submitIt) {
    for (auto cmdBufIt = submitIt->commandBuffers.begin();
         cmdBufIt != submitIt->commandBuffers.end(); ++cmdBufIt) {
      if (!cmdBufIt->empty()) {
        cmdBufIt->stagingCommandBuffer = commandBuffers.back();
        commandBuffers.pop_back();
      }
    }
  }

  VkBufferCreateInfo bufferCreateInfo = {
      VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,    // sType
      nullptr,                                                  // pNext
      0,                                                        // flags
      stagingSize,                                              // size
      VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT,  // usage
      VkSharingMode::VK_SHARING_MODE_EXCLUSIVE,                 // sharingMode
      0,        // queueFamilyIndexCount
      nullptr,  // pQueueFamilyIndices
  };
  res = fn->vkCreateBuffer(device, &bufferCreateInfo, nullptr, &stagingBuffer);
  if (res != VkResult::VK_SUCCESS) {
    stagingBuffer = 0;
    GAPID_ERROR("Failed at creating staging buffer to read external memory");
    return res;
  }

  VkMemoryRequirements memReqs(spy->arena());
  fn->vkGetBufferMemoryRequirements(device, stagingBuffer, &memReqs);

  VkPhysicalDevice physDevice = spy->mState.Devices[device]->mPhysicalDevice;
  VkPhysicalDeviceMemoryProperties memProps =
      spy->mState.PhysicalDevices[physDevice]->mMemoryProperties;
  uint32_t memoryTypeIndex =
      GetMemoryTypeIndexForStagingResources(memProps, memReqs.mmemoryTypeBits);
  if (memoryTypeIndex == kInvalidMemoryTypeIndex) {
    GAPID_ERROR(
        "Failed at finding memory type index for staging buffer memory to read "
        "external memory");
    return res;
  }

  VkMemoryAllocateInfo memoryAllocInfo = {
      VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,  // sType
      nullptr,                                                  // pNext
      memReqs.msize,    // allocationSize
      memoryTypeIndex,  // memoryTypeIndex
  };

  res = VkResult::VK_SUCCESS;
  res = fn->vkAllocateMemory(device, &memoryAllocInfo, nullptr, &stagingMemory);
  if (res != VkResult::VK_SUCCESS) {
    stagingMemory = 0;
    GAPID_ERROR(
        "Failed at allocating staging buffer memory to read external memory");
    return res;
  }

  res = fn->vkBindBufferMemory(device, stagingBuffer, stagingMemory, 0);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Failed at binding staging buffer to read external memory");
    return res;
  }

  return VkResult::VK_SUCCESS;
}

uint32_t ExternalMemoryStaging::RecordCommandBuffers() {
  for (auto submitIt = submits.begin(); submitIt != submits.end(); ++submitIt) {
    for (auto cmdBufIt = submitIt->commandBuffers.begin();
         cmdBufIt != submitIt->commandBuffers.end(); ++cmdBufIt) {
      if (!cmdBufIt->empty()) {
        uint32_t res = RecordStagingCommandBuffer(*cmdBufIt);
        if (res != VkResult::VK_SUCCESS) {
          return res;
        }
      }
    }
  }

  VkCommandBufferBeginInfo beginInfo{
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,  // sType
      nullptr,                                                       // pNext
      VkCommandBufferUsageFlagBits::
          VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,  // flags
      nullptr,                                          // pInheritanceInfo
  };
  uint32_t res = fn->vkBeginCommandBuffer(stagingCommandBuffer, &beginInfo);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Failed at begin command buffer to read external memory");
    return res;
  }

  // Make staging buffer writes visible to the host
  VkBufferMemoryBarrier barrier{
      VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,  // sType
      nullptr,                                                   // pNext
      VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,  // srcAccessMask
      VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,       // dstAccessMask
      queueFamilyIndex,                                // srcQueueFamilyIndex
      queueFamilyIndex,                                // dstQueueFamilyIndex
      stagingBuffer,                                   // buffer
      0,                                               // offset
      stagingSize,                                     // size
  };

  fn->vkCmdPipelineBarrier(
      stagingCommandBuffer,                                     // commandBuffer
      VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,  // srcStageMask
      VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT,      // dstStageMask
      0,         // dependencyFlags
      0,         // memoryBarrierCount
      nullptr,   // pMemoryBarriers
      1,         // bufferMemoryBarrierCount
      &barrier,  // pBufferMemoryBarriers
      0,         // imageMemoryBarrierCount
      nullptr    // pImageMemoryBarriers
  );

  res = fn->vkEndCommandBuffer(stagingCommandBuffer);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Failed at end command buffer to read external memory");
    return res;
  }
  return VkResult::VK_SUCCESS;
}

uint32_t ExternalMemoryStaging::RecordStagingCommandBuffer(
    const ExternalMemoryCommandBuffer& cmdBuf) {
  VkCommandBufferBeginInfo beginInfo{
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,  // sType
      nullptr,                                                       // pNext
      VkCommandBufferUsageFlagBits::
          VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,  // flags
      nullptr,                                          // pInheritanceInfo
  };
  uint32_t res =
      fn->vkBeginCommandBuffer(cmdBuf.stagingCommandBuffer, &beginInfo);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Failed at begin command buffer to read external memory");
    return res;
  }

  std::vector<VkBufferMemoryBarrier> acquireBufferBarriers;
  acquireBufferBarriers.reserve(cmdBuf.buffers.size());
  std::vector<VkBufferMemoryBarrier> releaseBufferBarriers;
  releaseBufferBarriers.reserve(cmdBuf.buffers.size());

  for (auto bufIt = cmdBuf.buffers.begin(); bufIt != cmdBuf.buffers.end();
       ++bufIt) {
    VkBufferMemoryBarrier barrier = bufIt->barrier;
    barrier.msrcAccessMask = 0;
    barrier.mdstAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
    acquireBufferBarriers.push_back(barrier);
    std::swap(barrier.msrcAccessMask, barrier.mdstAccessMask);
    std::swap(barrier.msrcQueueFamilyIndex, barrier.mdstQueueFamilyIndex);
    releaseBufferBarriers.push_back(barrier);
  }

  // acquire from external queue family
  fn->vkCmdPipelineBarrier(
      cmdBuf.stagingCommandBuffer,  // commandBuffer
      VkPipelineStageFlagBits::
          VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,                   // srcStageMask
      VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,  // dstStageMask
      0,                                       // dependencyFlags
      0,                                       // memoryBarrierCount
      nullptr,                                 // pMemoryBarriers
      (uint32_t)acquireBufferBarriers.size(),  // bufferMemoryBarrierCount
      acquireBufferBarriers.data(),            // pBufferMemoryBarriers
      (uint32_t)0,                             // imageMemoryBarrierCount
      nullptr                                  // pImageMemoryBarriers
  );

  // copy external buffer barrier regions to staging buffer
  for (auto bufIt = cmdBuf.buffers.begin(); bufIt != cmdBuf.buffers.end();
       ++bufIt) {
    fn->vkCmdCopyBuffer(cmdBuf.stagingCommandBuffer,  // commandBuffer
                        bufIt->buffer,                // srcBuffer
                        stagingBuffer,                // dstBuffer
                        1,                            // regionCount
                        &bufIt->copy                  // pRegions
    );
  }

  // release external barrier regions back to external queue family
  // (so that the original barriers run correctly when they execute later)
  fn->vkCmdPipelineBarrier(
      cmdBuf.stagingCommandBuffer,                              // commandBuffer
      VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,  // srcStageMask
      VkPipelineStageFlagBits::
          VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,  // dstStageMask
      0,                                       // dependencyFlags
      0,                                       // memoryBarrierCount
      nullptr,                                 // pMemoryBarriers
      (uint32_t)releaseBufferBarriers.size(),  // bufferMemoryBarrierCount
      releaseBufferBarriers.data(),            // pBufferMemoryBarriers
      (uint32_t)0,                             // imageMemoryBarrierCount
      nullptr                                  // pImageMemoryBarriers
  );

  res = fn->vkEndCommandBuffer(cmdBuf.stagingCommandBuffer);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Failed at end command buffer to read external memory");
    return res;
  }

  return VkResult::VK_SUCCESS;
}

uint32_t ExternalMemoryStaging::Submit() {
  std::vector<std::vector<VkCommandBuffer>> commandBuffers;
  std::vector<VkSubmitInfo> submitInfos;
  for (auto submitIt = submits.begin(); submitIt != submits.end(); ++submitIt) {
    commandBuffers.push_back({});
    std::vector<VkCommandBuffer>& submitCmds = commandBuffers.back();
    for (auto cmdBufIt = submitIt->commandBuffers.begin();
         cmdBufIt != submitIt->commandBuffers.end(); ++cmdBufIt) {
      if (!cmdBufIt->empty()) {
        submitCmds.push_back(cmdBufIt->stagingCommandBuffer);
      }
      submitCmds.push_back(cmdBufIt->commandBuffer);
    }
    submitInfos.push_back(*submitIt->submitInfo);
    submitInfos.back().mcommandBufferCount = (uint32_t)submitCmds.size();
    submitInfos.back().mpCommandBuffers = submitCmds.data();
  }
  submitInfos.push_back({
      VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO,  // sType
      nullptr,                                         // pNext
      0,                                               // waitSemaphoreCount
      nullptr,                                         // pWaitSemaphores
      nullptr,                                         // pWaitDstStageMask
      1,                                               // commandBufferCount
      &stagingCommandBuffer,                           // pCommandBuffers
      0,                                               // signalSemaphoreCount
      nullptr,                                         // pSignalSemaphores
  });
  uint32_t res = fn->vkQueueSubmit(queue, submitInfos.size(),
                                   submitInfos.data(), stagingFence);
  if (res != VkResult::VK_SUCCESS) {
    return res;
  }
  if (origFence != 0) {
    res = fn->vkQueueSubmit(queue, 0, nullptr, origFence);
    if (res != VkResult::VK_SUCCESS) {
      GAPID_ERROR(
          "Error submitting original fence after external memory observations");
      return res;
    }
  }
  return VkResult::VK_SUCCESS;
}

void ExternalMemoryStaging::SendData() {
  uint32_t res = fn->vkWaitForFences(device, 1, &stagingFence, 0, UINT64_MAX);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Error waiting for fence to save external memory observations");
    return;
  }

  static const VkDeviceSize VK_WHOLE_SIZE = ~0ULL;

  uint8_t* data = nullptr;
  res = fn->vkMapMemory(device, stagingMemory, 0, VK_WHOLE_SIZE, 0,
                        (void**)&data);
  if (res != VkResult::VK_SUCCESS) {
    GAPID_ERROR("Failed at mapping staging memory to save external memory");
    return;
  }

  VkMappedMemoryRange range{
      VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,  // sType
      nullptr,                                                 // pNext
      stagingMemory,                                           // memory
      0,                                                       // offset
      VK_WHOLE_SIZE,                                           // size
  };
  if (VkResult::VK_SUCCESS !=
      fn->vkInvalidateMappedMemoryRanges(device, 1, &range)) {
    GAPID_ERROR("Failed at invalidating mapped memory to save external memory");
  } else {
    auto resIndex = spy->sendResource(VulkanSpy::kApiIndex, data, stagingSize);

    auto extra = new vulkan_pb::ExternalMemoryData();
    extra->set_res_index(resIndex);
    extra->set_res_size(stagingSize);
    for (uint32_t submitIndex = 0; submitIndex < submits.size();
         ++submitIndex) {
      const ExternalMemorySubmitInfo& submit = submits[submitIndex];
      for (uint32_t commandBufferIndex = 0;
           commandBufferIndex < submit.commandBuffers.size();
           ++commandBufferIndex) {
        const ExternalMemoryCommandBuffer& cmdBuf =
            submit.commandBuffers[commandBufferIndex];
        for (auto bufIt = cmdBuf.buffers.begin(); bufIt != cmdBuf.buffers.end();
             ++bufIt) {
          auto bufMsg = extra->add_buffers();
          bufMsg->set_buffer(bufIt->buffer);
          bufMsg->set_buffer_offset(bufIt->copy.msrcOffset);
          bufMsg->set_data_offset(bufIt->copy.mdstOffset);
          bufMsg->set_size(bufIt->copy.msize);
          bufMsg->set_submit_index(submitIndex);
          bufMsg->set_command_buffer_index(commandBufferIndex);
        }
      }
    }
    observer->encodeAndDelete(extra);
  }

  fn->vkUnmapMemory(device, stagingMemory);
}

void ExternalMemoryStaging::Cleanup() {
  if (stagingCommandPool != 0) {
    fn->vkDestroyCommandPool(device, stagingCommandPool, nullptr);
    stagingCommandPool = 0;
  }

  if (stagingFence != 0) {
    fn->vkDestroyFence(device, stagingFence, nullptr);
    stagingFence = 0;
  }

  if (stagingBuffer != 0) {
    fn->vkDestroyBuffer(device, stagingBuffer, nullptr);
    stagingBuffer = 0;
  }

  if (stagingMemory != 0) {
    fn->vkFreeMemory(device, stagingMemory, nullptr);
    stagingMemory = 0;
  }
}

}  // namespace gapii