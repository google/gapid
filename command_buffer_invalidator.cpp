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
#include <externals/SPIRV-Reflect/spirv_reflect.h>
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "buffer.h"
#include "command_buffer.h"
#include "command_buffer_invalidator.h"
#include "descriptor_set.h"
#include "event.h"
#include "framebuffer.h"
#include "image.h"
#include "image_view.h"
#include "pipeline.h"
#include "pipeline_layout.h"
#include "query_pool.h"
#include "render_pass.h"
#include "state_block.h"

namespace gapid2 {

VkResult command_buffer_invalidator::vkBeginCommandBuffer(VkCommandBuffer commandBuffer, const VkCommandBufferBeginInfo* pBeginInfo) {
  auto cb = state_block_->get(commandBuffer);
  cb->invalidated = false;
  return super::vkBeginCommandBuffer(commandBuffer, pBeginInfo);
}

void command_buffer_invalidator::vkCmdBindDescriptorSets(VkCommandBuffer commandBuffer, VkPipelineBindPoint pipelineBindPoint, VkPipelineLayout layout, uint32_t firstSet, uint32_t descriptorSetCount, const VkDescriptorSet* pDescriptorSets, uint32_t dynamicOffsetCount, const uint32_t* pDynamicOffsets) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(layout)->invalidates(cb.get());
  for (uint32_t i = 0; i < descriptorSetCount; ++i) {
    auto ds = state_block_->get(pDescriptorSets[i]);
    ds->invalidates(cb.get());
  }
  return super::vkCmdBindDescriptorSets(commandBuffer, pipelineBindPoint, layout, firstSet, descriptorSetCount, pDescriptorSets, dynamicOffsetCount, pDynamicOffsets);
}

void command_buffer_invalidator::vkCmdBindIndexBuffer(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkIndexType indexType) {
  auto cb = state_block_->get(commandBuffer);
  auto b = state_block_->get(buffer);
  b->invalidates(cb.get());
  return super::vkCmdBindIndexBuffer(commandBuffer, buffer, offset, indexType);
}

void command_buffer_invalidator::vkCmdBindVertexBuffers(VkCommandBuffer commandBuffer, uint32_t firstBinding, uint32_t bindingCount, const VkBuffer* pBuffers, const VkDeviceSize* pOffsets) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < bindingCount; ++i) {
    if (pBuffers[i]) {
      auto b = state_block_->get(pBuffers[i]);
      b->invalidates(cb.get());
    }
  }
  return super::vkCmdBindVertexBuffers(commandBuffer, firstBinding, bindingCount, pBuffers, pOffsets);
}

void command_buffer_invalidator::vkCmdBindPipeline(VkCommandBuffer commandBuffer, VkPipelineBindPoint pipelineBindPoint, VkPipeline pipeline) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(pipeline)->invalidates(cb.get());
  return super::vkCmdBindPipeline(commandBuffer, pipelineBindPoint, pipeline);
}

void command_buffer_invalidator::vkCmdCopyBuffer(VkCommandBuffer commandBuffer, VkBuffer srcBuffer, VkBuffer dstBuffer, uint32_t regionCount, const VkBufferCopy* pRegions) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(srcBuffer)->invalidates(cb.get());
  state_block_->get(dstBuffer)->invalidates(cb.get());
  return super::vkCmdCopyBuffer(commandBuffer, srcBuffer, dstBuffer, regionCount, pRegions);
}

void command_buffer_invalidator::vkCmdCopyImage(VkCommandBuffer commandBuffer, VkImage srcImage, VkImageLayout srcImageLayout, VkImage dstImage, VkImageLayout dstImageLayout, uint32_t regionCount, const VkImageCopy* pRegions) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(srcImage)->invalidates(cb.get());
  state_block_->get(dstImage)->invalidates(cb.get());
  return super::vkCmdCopyImage(commandBuffer, srcImage, srcImageLayout, dstImage, dstImageLayout, regionCount, pRegions);
}

void command_buffer_invalidator::vkCmdBlitImage(VkCommandBuffer commandBuffer, VkImage srcImage, VkImageLayout srcImageLayout, VkImage dstImage, VkImageLayout dstImageLayout, uint32_t regionCount, const VkImageBlit* pRegions, VkFilter filter) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(srcImage)->invalidates(cb.get());
  state_block_->get(dstImage)->invalidates(cb.get());
  return super::vkCmdBlitImage(commandBuffer, srcImage, srcImageLayout, dstImage, dstImageLayout, regionCount, pRegions, filter);
}

void command_buffer_invalidator::vkCmdCopyBufferToImage(VkCommandBuffer commandBuffer, VkBuffer srcBuffer, VkImage dstImage, VkImageLayout dstImageLayout, uint32_t regionCount, const VkBufferImageCopy* pRegions) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(srcBuffer)->invalidates(cb.get());
  state_block_->get(dstImage)->invalidates(cb.get());
  return super::vkCmdCopyBufferToImage(commandBuffer, srcBuffer, dstImage, dstImageLayout, regionCount, pRegions);
}

void command_buffer_invalidator::vkCmdCopyImageToBuffer(VkCommandBuffer commandBuffer, VkImage srcImage, VkImageLayout srcImageLayout, VkBuffer dstBuffer, uint32_t regionCount, const VkBufferImageCopy* pRegions) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(srcImage)->invalidates(cb.get());
  state_block_->get(dstBuffer)->invalidates(cb.get());
  return super::vkCmdCopyImageToBuffer(commandBuffer, srcImage, srcImageLayout, dstBuffer, regionCount, pRegions);
}

void command_buffer_invalidator::vkCmdUpdateBuffer(VkCommandBuffer commandBuffer, VkBuffer dstBuffer, VkDeviceSize dstOffset, VkDeviceSize dataSize, const void* pData) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(dstBuffer)->invalidates(cb.get());
  return super::vkCmdUpdateBuffer(commandBuffer, dstBuffer, dstOffset, dataSize, pData);
}

void command_buffer_invalidator::vkCmdFillBuffer(VkCommandBuffer commandBuffer, VkBuffer dstBuffer, VkDeviceSize dstOffset, VkDeviceSize size, uint32_t data) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(dstBuffer)->invalidates(cb.get());
  return super::vkCmdFillBuffer(commandBuffer, dstBuffer, dstOffset, size, data);
}

void command_buffer_invalidator::vkCmdClearColorImage(VkCommandBuffer commandBuffer, VkImage image, VkImageLayout imageLayout, const VkClearColorValue* pColor, uint32_t rangeCount, const VkImageSubresourceRange* pRanges) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(image)->invalidates(cb.get());
  return super::vkCmdClearColorImage(commandBuffer, image, imageLayout, pColor, rangeCount, pRanges);
}

void command_buffer_invalidator::vkCmdDrawIndirect(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, uint32_t drawCount, uint32_t stride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(buffer)->invalidates(cb.get());
  return super::vkCmdDrawIndirect(commandBuffer, buffer, offset, drawCount, stride);
}

void command_buffer_invalidator::vkCmdDrawIndexedIndirect(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, uint32_t drawCount, uint32_t stride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(buffer)->invalidates(cb.get());
  return super::vkCmdDrawIndexedIndirect(commandBuffer, buffer, offset, drawCount, stride);
}

void command_buffer_invalidator::vkCmdDrawIndirectCount(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkBuffer countBuffer, VkDeviceSize countBufferOffset, uint32_t maxDrawCount, uint32_t stride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(buffer)->invalidates(cb.get());
  state_block_->get(countBuffer)->invalidates(cb.get());
  return super::vkCmdDrawIndirectCount(commandBuffer, buffer, offset, countBuffer, countBufferOffset, maxDrawCount, stride);
}

void command_buffer_invalidator::vkCmdDrawIndexedIndirectCount(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkBuffer countBuffer, VkDeviceSize countBufferOffset, uint32_t maxDrawCount, uint32_t stride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(buffer)->invalidates(cb.get());
  state_block_->get(countBuffer)->invalidates(cb.get());
  return super::vkCmdDrawIndexedIndirectCount(commandBuffer, buffer, offset, countBuffer, countBufferOffset, maxDrawCount, stride);
}

void command_buffer_invalidator::vkCmdBindVertexBuffers2EXT(VkCommandBuffer commandBuffer, uint32_t firstBinding, uint32_t bindingCount, const VkBuffer* pBuffers, const VkDeviceSize* pOffsets, const VkDeviceSize* pSizes, const VkDeviceSize* pStrides) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < bindingCount; ++i) {
    if (pBuffers[i]) {
      auto b = state_block_->get(pBuffers[i]);
      b->invalidates(cb.get());
    }
  }
  return super::vkCmdBindVertexBuffers2EXT(commandBuffer, firstBinding, bindingCount, pBuffers, pOffsets, pSizes, pStrides);
}

void command_buffer_invalidator::vkCmdBindTransformFeedbackBuffersEXT(VkCommandBuffer commandBuffer, uint32_t firstBinding, uint32_t bindingCount, const VkBuffer* pBuffers, const VkDeviceSize* pOffsets, const VkDeviceSize* pSizes) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < bindingCount; ++i) {
    auto b = state_block_->get(pBuffers[i]);
    b->invalidates(cb.get());
  }
  return super::vkCmdBindTransformFeedbackBuffersEXT(commandBuffer, firstBinding, bindingCount, pBuffers, pOffsets, pSizes);
}

void command_buffer_invalidator::vkCmdBeginTransformFeedbackEXT(VkCommandBuffer commandBuffer, uint32_t firstCounterBuffer, uint32_t counterBufferCount, const VkBuffer* pCounterBuffers, const VkDeviceSize* pCounterBufferOffsets) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < counterBufferCount; ++i) {
    auto b = state_block_->get(pCounterBuffers[i]);
    b->invalidates(cb.get());
  }
  return super::vkCmdBeginTransformFeedbackEXT(commandBuffer, firstCounterBuffer, counterBufferCount, pCounterBuffers, pCounterBufferOffsets);
}

void command_buffer_invalidator::vkCmdEndTransformFeedbackEXT(VkCommandBuffer commandBuffer, uint32_t firstCounterBuffer, uint32_t counterBufferCount, const VkBuffer* pCounterBuffers, const VkDeviceSize* pCounterBufferOffsets) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < counterBufferCount; ++i) {
    auto b = state_block_->get(pCounterBuffers[i]);
    b->invalidates(cb.get());
  }
  return super::vkCmdEndTransformFeedbackEXT(commandBuffer, firstCounterBuffer, counterBufferCount, pCounterBuffers, pCounterBufferOffsets);
}

void command_buffer_invalidator::vkCmdBeginQueryIndexedEXT(VkCommandBuffer commandBuffer, VkQueryPool queryPool, uint32_t query, VkQueryControlFlags flags, uint32_t index) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  return super::vkCmdBeginQueryIndexedEXT(commandBuffer, queryPool, query, flags, index);
}

void command_buffer_invalidator::vkCmdEndQueryIndexedEXT(VkCommandBuffer commandBuffer, VkQueryPool queryPool, uint32_t query, uint32_t index) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  return super::vkCmdEndQueryIndexedEXT(commandBuffer, queryPool, query, index);
}

void command_buffer_invalidator::vkCmdDrawIndirectByteCountEXT(VkCommandBuffer commandBuffer, uint32_t instanceCount, uint32_t firstInstance, VkBuffer counterBuffer, VkDeviceSize counterBufferOffset, uint32_t counterOffset, uint32_t vertexStride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(counterBuffer)->invalidates(cb.get());
  return super::vkCmdDrawIndirectByteCountEXT(commandBuffer, instanceCount, firstInstance, counterBuffer, counterBufferOffset, counterOffset, vertexStride);
}

void command_buffer_invalidator::vkCmdBeginRenderPass2KHR(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, const VkSubpassBeginInfo* pSubpassBeginInfo) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(pRenderPassBegin->framebuffer)->invalidates(cb.get());
  state_block_->get(pRenderPassBegin->renderPass)->invalidates(cb.get());
  return super::vkCmdBeginRenderPass2KHR(commandBuffer, pRenderPassBegin, pSubpassBeginInfo);
}

void command_buffer_invalidator::vkCmdDrawIndirectCountKHR(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkBuffer countBuffer, VkDeviceSize countBufferOffset, uint32_t maxDrawCount, uint32_t stride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(countBuffer)->invalidates(cb.get());
  state_block_->get(buffer)->invalidates(cb.get());
  return super::vkCmdDrawIndirectCountKHR(commandBuffer, buffer, offset, countBuffer, countBufferOffset, maxDrawCount, stride);
}

void command_buffer_invalidator::vkCmdDrawIndexedIndirectCountKHR(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset, VkBuffer countBuffer, VkDeviceSize countBufferOffset, uint32_t maxDrawCount, uint32_t stride) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(buffer)->invalidates(cb.get());
  state_block_->get(countBuffer)->invalidates(cb.get());
  return super::vkCmdDrawIndexedIndirectCountKHR(commandBuffer, buffer, offset, countBuffer, countBufferOffset, maxDrawCount, stride);
}

void command_buffer_invalidator::vkCmdPushConstants(VkCommandBuffer commandBuffer, VkPipelineLayout layout, VkShaderStageFlags stageFlags, uint32_t offset, uint32_t size, const void* pValues) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(layout)->invalidates(cb.get());
  return super::vkCmdPushConstants(commandBuffer, layout, stageFlags, offset, size, pValues);
}

void command_buffer_invalidator::vkCmdBeginQuery(VkCommandBuffer commandBuffer, VkQueryPool queryPool, uint32_t query, VkQueryControlFlags flags) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  return super::vkCmdBeginQuery(commandBuffer, queryPool, query, flags);
}

void command_buffer_invalidator::vkCmdEndQuery(VkCommandBuffer commandBuffer, VkQueryPool queryPool, uint32_t query) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  return super::vkCmdEndQuery(commandBuffer, queryPool, query);
}

void command_buffer_invalidator::vkCmdResetQueryPool(VkCommandBuffer commandBuffer, VkQueryPool queryPool, uint32_t firstQuery, uint32_t queryCount) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  return super::vkCmdResetQueryPool(commandBuffer, queryPool, firstQuery, queryCount);
}

void command_buffer_invalidator::vkCmdWriteTimestamp(VkCommandBuffer commandBuffer, VkPipelineStageFlagBits pipelineStage, VkQueryPool queryPool, uint32_t query) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  return super::vkCmdWriteTimestamp(commandBuffer, pipelineStage, queryPool, query);
}

void command_buffer_invalidator::vkCmdCopyQueryPoolResults(VkCommandBuffer commandBuffer, VkQueryPool queryPool, uint32_t firstQuery, uint32_t queryCount, VkBuffer dstBuffer, VkDeviceSize dstOffset, VkDeviceSize stride, VkQueryResultFlags flags) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(queryPool)->invalidates(cb.get());
  state_block_->get(dstBuffer)->invalidates(cb.get());
  return super::vkCmdCopyQueryPoolResults(commandBuffer, queryPool, firstQuery, queryCount, dstBuffer, dstOffset, stride, flags);
}

void command_buffer_invalidator::vkCmdDispatchIndirect(VkCommandBuffer commandBuffer, VkBuffer buffer, VkDeviceSize offset) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(buffer)->invalidates(cb.get());
  return super::vkCmdDispatchIndirect(commandBuffer, buffer, offset);
}

void command_buffer_invalidator::vkCmdClearDepthStencilImage(VkCommandBuffer commandBuffer, VkImage image, VkImageLayout imageLayout, const VkClearDepthStencilValue* pDepthStencil, uint32_t rangeCount, const VkImageSubresourceRange* pRanges) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(image)->invalidates(cb.get());
  return super::vkCmdClearDepthStencilImage(commandBuffer, image, imageLayout, pDepthStencil, rangeCount, pRanges);
}

void command_buffer_invalidator::vkCmdResolveImage(VkCommandBuffer commandBuffer, VkImage srcImage, VkImageLayout srcImageLayout, VkImage dstImage, VkImageLayout dstImageLayout, uint32_t regionCount, const VkImageResolve* pRegions) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(srcImage)->invalidates(cb.get());
  state_block_->get(dstImage)->invalidates(cb.get());
  return super::vkCmdResolveImage(commandBuffer, srcImage, srcImageLayout, dstImage, dstImageLayout, regionCount, pRegions);
}

void command_buffer_invalidator::vkCmdSetEvent(VkCommandBuffer commandBuffer, VkEvent event, VkPipelineStageFlags stageMask) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(event)->invalidates(cb.get());
  return super::vkCmdSetEvent(commandBuffer, event, stageMask);
}

void command_buffer_invalidator::vkCmdResetEvent(VkCommandBuffer commandBuffer, VkEvent event, VkPipelineStageFlags stageMask) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(event)->invalidates(cb.get());
  return super::vkCmdResetEvent(commandBuffer, event, stageMask);
}

void command_buffer_invalidator::vkCmdWaitEvents(VkCommandBuffer commandBuffer, uint32_t eventCount, const VkEvent* pEvents, VkPipelineStageFlags srcStageMask, VkPipelineStageFlags dstStageMask, uint32_t memoryBarrierCount, const VkMemoryBarrier* pMemoryBarriers, uint32_t bufferMemoryBarrierCount, const VkBufferMemoryBarrier* pBufferMemoryBarriers, uint32_t imageMemoryBarrierCount, const VkImageMemoryBarrier* pImageMemoryBarriers) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < eventCount; ++i) {
    state_block_->get(pEvents[i])->invalidates(cb.get());
  }
  return super::vkCmdWaitEvents(commandBuffer, eventCount, pEvents, srcStageMask, dstStageMask, memoryBarrierCount, pMemoryBarriers, bufferMemoryBarrierCount, pBufferMemoryBarriers, imageMemoryBarrierCount, pImageMemoryBarriers);
}

void command_buffer_invalidator::vkCmdPipelineBarrier(VkCommandBuffer commandBuffer, VkPipelineStageFlags srcStageMask, VkPipelineStageFlags dstStageMask, VkDependencyFlags dependencyFlags, uint32_t memoryBarrierCount, const VkMemoryBarrier* pMemoryBarriers, uint32_t bufferMemoryBarrierCount, const VkBufferMemoryBarrier* pBufferMemoryBarriers, uint32_t imageMemoryBarrierCount, const VkImageMemoryBarrier* pImageMemoryBarriers) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < bufferMemoryBarrierCount; ++i) {
    state_block_->get(pBufferMemoryBarriers[i].buffer)->invalidates(cb.get());
  }
  for (uint32_t i = 0; i < imageMemoryBarrierCount; ++i) {
    state_block_->get(pImageMemoryBarriers[i].image)->invalidates(cb.get());
  }
  return super::vkCmdPipelineBarrier(commandBuffer, srcStageMask, dstStageMask, dependencyFlags, memoryBarrierCount, pMemoryBarriers, bufferMemoryBarrierCount, pBufferMemoryBarriers, imageMemoryBarrierCount, pImageMemoryBarriers);
}

void command_buffer_invalidator::vkCmdBeginRenderPass(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, VkSubpassContents contents) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(pRenderPassBegin->renderPass)->invalidates(cb.get());
  state_block_->get(pRenderPassBegin->framebuffer)->invalidates(cb.get());
  return super::vkCmdBeginRenderPass(commandBuffer, pRenderPassBegin, contents);
}

void command_buffer_invalidator::vkCmdExecuteCommands(VkCommandBuffer commandBuffer, uint32_t commandBufferCount, const VkCommandBuffer* pCommandBuffers) {
  auto cb = state_block_->get(commandBuffer);
  for (uint32_t i = 0; i < commandBufferCount; ++i) {
    state_block_->get(pCommandBuffers[i])->invalidates(cb.get());
  }
  return super::vkCmdExecuteCommands(commandBuffer, commandBufferCount, pCommandBuffers);
}

void command_buffer_invalidator::vkCmdBeginRenderPass2(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, const VkSubpassBeginInfo* pSubpassBeginInfo) {
  auto cb = state_block_->get(commandBuffer);
  state_block_->get(pRenderPassBegin->framebuffer)->invalidates(cb.get());
  state_block_->get(pRenderPassBegin->renderPass)->invalidates(cb.get());
  return super::vkCmdBeginRenderPass2(commandBuffer, pRenderPassBegin, pSubpassBeginInfo);
}

}  // namespace gapid2