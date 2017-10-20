/*
 * Copyright (C) 2017 Google Inc.
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

// This file is intended to be included by vulkan_spy.h inside
// of the gapid namespace.

// Note: Almost this entire file can be removed when we
// serialize the intial state instead of passing recreate commands.
template<typename T>
struct CommandListRecreator {
    bool inline operator()(VkCommandBuffer commandBuf, CallObserver* observer,
        VulkanSpy* spy, const T& t) {
        GAPID_FATAL("Not implemented");
    }
private:
};


////////// Command Buffer Commands
template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdUpdateBufferArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdUpdateBufferArgs>& t) {
    if (!spy->Buffers.count(t->mDstBuffer)) {
        return false;
    }
    spy->vkCmdUpdateBuffer(observer, commandBuf,
        t->mDstBuffer, t->mDstOffset, t->mDataSize, &t->mData[0]);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdEndRenderPassArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdEndRenderPassArgs>&) {
    spy->vkCmdEndRenderPass(observer, commandBuf);
    return true;
}


template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdPipelineBarrierArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdPipelineBarrierArgs>& t) {
    std::vector<VkMemoryBarrier> memory_barriers;
    memory_barriers.reserve(t->mMemoryBarriers.size());
    for (size_t i = 0; i < t->mMemoryBarriers.size(); ++i) {
        memory_barriers.push_back(t->mMemoryBarriers[i]);
    }
    std::vector<VkBufferMemoryBarrier> buffer_memory_barriers;
    memory_barriers.reserve(t->mBufferMemoryBarriers.size());
    for (size_t i = 0; i < t->mBufferMemoryBarriers.size(); ++i) {
        buffer_memory_barriers.push_back(t->mBufferMemoryBarriers[i]);
        if (!spy->Buffers.count(t->mBufferMemoryBarriers[i].mbuffer)) {
            return false;
        }
    }
    std::vector<VkImageMemoryBarrier> image_memory_barriers;
    memory_barriers.reserve(t->mImageMemoryBarriers.size());
    for (size_t i = 0; i < t->mImageMemoryBarriers.size(); ++i) {
        image_memory_barriers.push_back(t->mImageMemoryBarriers[i]);
        if (!spy->Images.count(t->mImageMemoryBarriers[i].mimage)) {
            return false;
        }
    }
    spy->vkCmdPipelineBarrier(observer, commandBuf,
        t->mSrcStageMask, t->mDstStageMask, t->mDependencyFlags,
        memory_barriers.size(), memory_barriers.data(),
        buffer_memory_barriers.size(), buffer_memory_barriers.data(),
        image_memory_barriers.size(), image_memory_barriers.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdCopyBufferArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdCopyBufferArgs>& t) {
    if (!spy->Buffers.count(t->mSrcBuffer) ||
        !spy->Buffers.count(t->mDstBuffer)) {
        return false;
    }
    std::vector<VkBufferCopy> buffer_copies;
    buffer_copies.reserve(t->mCopyRegions.size());
    for (size_t i = 0; i < t->mCopyRegions.size(); ++i) {
        buffer_copies.push_back(t->mCopyRegions[i]);
    }
    spy->vkCmdCopyBuffer(observer, commandBuf,
        t->mSrcBuffer, t->mDstBuffer,
        buffer_copies.size(), buffer_copies.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdResolveImageArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdResolveImageArgs>& t) {
    std::vector<VkImageResolve> image_resolves;
    image_resolves.reserve(t->mResolveRegions.size());
    if (!spy->Images.count(t->mSrcImage) ||
        !spy->Images.count(t->mDstImage)) {
        return false;
    }
    for (size_t i = 0; i < t->mResolveRegions.size(); ++i) {
        image_resolves.push_back(t->mResolveRegions[i]);
    }
    spy->vkCmdResolveImage(observer, commandBuf,
        t->mSrcImage, t->mSrcImageLayout, t->mDstImage, t->mDstImageLayout,
        image_resolves.size(), image_resolves.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBeginRenderPassArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBeginRenderPassArgs>& t) {
    if (!spy->RenderPasses.count(t->mRenderPass) ||
        !spy->Framebuffers.count(t->mFramebuffer)) {
        return false;
    }
    std::vector<VkClearValue> clear_values;
    clear_values.reserve(t->mClearValues.size());
    for (size_t i = 0; i < t->mClearValues.size(); ++i) {
        clear_values.push_back(t->mClearValues[i]);
    }
    VkRenderPassBeginInfo begin_info {
        VkStructureType::VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
        nullptr,
        t->mRenderPass,
        t->mFramebuffer,
        t->mRenderArea,
        static_cast<uint32_t>(clear_values.size()),
        clear_values.data()
    };
    spy->vkCmdBeginRenderPass(observer, commandBuf,
        &begin_info, t->mContents);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBindDescriptorSetsArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBindDescriptorSetsArgs>& t) {
    std::vector<uint32_t> dynamic_offsets;
    dynamic_offsets.reserve(t->mDynamicOffsets.size());
    for (size_t i = 0; i < t->mDynamicOffsets.size(); ++i) {
        dynamic_offsets.push_back(t->mDynamicOffsets[i]);
    }
    std::vector<VkDescriptorSet> descriptor_sets;
    descriptor_sets.reserve(t->mDescriptorSets.size());
    for (size_t i = 0; i < t->mDescriptorSets.size(); ++i) {
        if (!spy->DescriptorSets.count(t->mDescriptorSets[i])) {
            return false;
        }
        descriptor_sets.push_back(t->mDescriptorSets[i]);
    }

    spy->vkCmdBindDescriptorSets(observer, commandBuf,
        t->mPipelineBindPoint, t->mLayout, t->mFirstSet,
        descriptor_sets.size(), descriptor_sets.data(),
        dynamic_offsets.size(), dynamic_offsets.data());
    return true;
}


template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBindVertexBuffersArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBindVertexBuffersArgs>& t) {
    std::vector<VkBuffer> buffers;
    buffers.reserve(t->mBuffers.size());
    for (size_t i = 0; i < t->mBuffers.size(); ++i) {
        if (!spy->Buffers.count(t->mBuffers[i])) {
            return false;
        }
        buffers.push_back(t->mBuffers[i]);
    }
    std::vector<uint64_t> offsets;
    offsets.reserve(t->mOffsets.size());
    for (size_t i = 0; i < t->mOffsets.size(); ++i) {
        offsets.push_back(t->mOffsets[i]);
    }

    spy->vkCmdBindVertexBuffers(observer, commandBuf,
        t->mFirstBinding, t->mBindingCount, buffers.data(), offsets.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBindIndexBufferArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBindIndexBufferArgs>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return false;
    }
    spy->vkCmdBindIndexBuffer(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mIndexType);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdDrawIndirectArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdDrawIndirectArgs>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return false;
    }
    spy->vkCmdDrawIndirect(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mDrawCount, t->mStride);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdDrawIndexedIndirectArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdDrawIndexedIndirectArgs>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return false;
    }
    spy->vkCmdDrawIndexedIndirect(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mDrawCount, t->mStride);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetDepthBiasArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetDepthBiasArgs>& t) {
    spy->vkCmdSetDepthBias(observer, commandBuf,
        t->mDepthBiasConstantFactor, t->mDepthBiasClamp, t->mDepthBiasSlopeFactor);
    return true;
}

template <>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetDepthBoundsArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetDepthBoundsArgs>& t) {
  spy->vkCmdSetDepthBounds(observer, commandBuf, t->mMinDepthBounds, t->mMaxDepthBounds);
  return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetLineWidthArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetLineWidthArgs>& t) {
    spy->vkCmdSetLineWidth(observer, commandBuf, t->mLineWidth);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdFillBufferArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdFillBufferArgs>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return false;
    }
    spy->vkCmdFillBuffer(observer, commandBuf, t->mBuffer, t->mDstOffset, t->mSize, t->mData);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetBlendConstantsArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetBlendConstantsArgs>& t) {
    float constants[4] = {
        t->mR,
        t->mG,
        t->mB,
        t->mA
    };
    spy->vkCmdSetBlendConstants(observer, commandBuf, constants);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBindPipelineArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBindPipelineArgs>& t) {
    if (!spy->GraphicsPipelines.count(t->mPipeline) &&
        !spy->ComputePipelines.count(t->mPipeline)) {
        return false;
    }
    spy->vkCmdBindPipeline(observer, commandBuf,
        t->mPipelineBindPoint, t->mPipeline);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBeginQueryArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBeginQueryArgs>& t) {
    if (!spy->QueryPools.count(t->mQueryPool)) {
        return false;
    }
    spy->vkCmdBeginQuery(observer, commandBuf,
        t->mQueryPool, t->mQuery, t->mFlags);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdEndQueryArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdEndQueryArgs>& t) {
    if (!spy->QueryPools.count(t->mQueryPool)) {
        return false;
    }
    spy->vkCmdEndQuery(observer, commandBuf,
        t->mQueryPool, t->mQuery);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdResetQueryPoolArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdResetQueryPoolArgs>& t) {
    if (!spy->QueryPools.count(t->mQueryPool)) {
        return false;
    }
    spy->vkCmdResetQueryPool(observer, commandBuf,
        t->mQueryPool, t->mFirstQuery, t->mQueryCount);
    return true;
}

template <>
bool inline CommandListRecreator<std::shared_ptr<vkCmdWriteTimestampArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdWriteTimestampArgs>& t) {
    spy->vkCmdWriteTimestamp(observer, commandBuf, t->mPipelineStage,
                                  t->mQueryPool, t->mQuery);
    return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdCopyQueryPoolResultsArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdCopyQueryPoolResultsArgs>& t) {
  if (!spy->QueryPools.count(t->mQueryPool)) {
    return false;
  }
  spy->vkCmdCopyQueryPoolResults(
      observer, commandBuf, t->mQueryPool, t->mFirstQuery, t->mQueryCount,
      t->mDstBuffer, t->mDstOffset, t->mStride, t->mFlags);
  return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdCopyBufferToImageArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdCopyBufferToImageArgs>& t) {
    if (!spy->Buffers.count(t->mSrcBuffer) ||
        !spy->Images.count(t->mDstImage)) {
        return false;
    }
    std::vector<VkBufferImageCopy> regions;
    regions.reserve(t->mRegions.size());
    for (size_t i = 0; i < t->mRegions.size(); ++i) {
        regions.push_back(t->mRegions[i]);
    }
    spy->vkCmdCopyBufferToImage(observer, commandBuf,
        t->mSrcBuffer, t->mDstImage, t->mLayout, regions.size(), regions.data());
    return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdCopyImageToBufferArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdCopyImageToBufferArgs>& t) {
  if (!spy->Images.count(t->mSrcImage) || !spy->Buffers.count(t->mDstBuffer)) {
    return false;
  }
  std::vector<VkBufferImageCopy> regions;
  regions.reserve(t->mRegions.size());
  for (size_t i = 0; i < t->mRegions.size(); ++i) {
    regions.push_back(t->mRegions[i]);
  }
  spy->vkCmdCopyImageToBuffer(observer, commandBuf, t->mSrcImage,
                                    t->mSrcImageLayout, t->mDstBuffer,
                                    regions.size(), regions.data());
  return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdBlitImageArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdBlitImageArgs>& t) {
    if (!spy->Images.count(t->mSrcImage) ||
        !spy->Images.count(t->mDstImage)) {
        return false;
    }
    std::vector<VkImageBlit> regions;
    regions.reserve(t->mRegions.size());
    for (size_t i = 0; i < t->mRegions.size(); ++i) {
        regions.push_back(t->mRegions[i]);
    }
    spy->vkCmdBlitImage(
        observer, commandBuf, t->mSrcImage, t->mSrcImageLayout, t->mDstImage,
        t->mDstImageLayout, regions.size(), regions.data(), t->mFilter);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdCopyImageArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdCopyImageArgs>& t) {
    if (!spy->Images.count(t->mSrcImage) ||
        !spy->Images.count(t->mDstImage)) {
        return false;
    }
    std::vector<VkImageCopy> regions;
    regions.reserve(t->mRegions.size());
    for (size_t i = 0; i < t->mRegions.size(); ++i) {
        regions.push_back(t->mRegions[i]);
    }
    spy->vkCmdCopyImage(observer, commandBuf,
        t->mSrcImage, t->mSrcImageLayout,
        t->mDstImage, t->mDstImageLayout,
        regions.size(), regions.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdPushConstantsArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdPushConstantsArgs>& t) {
    if (!spy->PipelineLayouts.count(t->mLayout)) {
        return false;
    }
    spy->vkCmdPushConstants(observer, commandBuf,
        t->mLayout, t->mStageFlags,
        t->mOffset, t->mSize,
        &t->mData[0]);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetScissorArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetScissorArgs>& t) {
    std::vector<VkRect2D> rects;
    rects.reserve(t->mScissors.size());
    for (size_t i = 0; i < t->mScissors.size(); ++i) {
        rects.push_back(t->mScissors[i]);
    }
    spy->vkCmdSetScissor(observer, commandBuf,
        t->mFirstScissor, rects.size(), rects.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetViewportArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetViewportArgs>& t) {
    std::vector<VkViewport> viewports;
    viewports.reserve(t->mViewports.size());
    for (size_t i = 0; i < t->mViewports.size(); ++i) {
        viewports.push_back(t->mViewports[i]);
    }
    spy->vkCmdSetViewport(observer, commandBuf,
        t->mFirstViewport, viewports.size(), viewports.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdDrawArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdDrawArgs>& t) {
    spy->vkCmdDraw(observer, commandBuf,
        t->mVertexCount, t->mInstanceCount, t->mFirstVertex, t->mFirstInstance);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdDispatchArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdDispatchArgs>& t) {
    spy->vkCmdDispatch(observer, commandBuf,
        t->mGroupCountX, t->mGroupCountY, t->mGroupCountZ);
    return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdDispatchIndirectArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdDispatchIndirectArgs>& t) {
  spy->vkCmdDispatchIndirect(observer, commandBuf, t->mBuffer,
                                   t->mOffset);
  return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdDrawIndexedArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdDrawIndexedArgs>& t) {
    spy->vkCmdDrawIndexed(observer, commandBuf,
        t->mIndexCount, t->mInstanceCount, t->mFirstIndex, t->mVertexOffset, t->mFirstInstance);
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdClearAttachmentsArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdClearAttachmentsArgs>& t) {
    std::vector<VkClearAttachment> attachments;
    attachments.reserve(t->mAttachments.size());
    for (size_t i = 0; i < t->mAttachments.size(); ++i) {
        attachments.push_back(t->mAttachments[i]);
    }
    std::vector<VkClearRect> rects;
    rects.reserve(t->mRects.size());
    for (size_t i = 0; i < t->mRects.size(); ++i) {
        rects.push_back(t->mRects[i]);
    }
    spy->vkCmdClearAttachments(observer, commandBuf, attachments.size(),
                                     attachments.data(), rects.size(),
                                     rects.data());
    return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdClearColorImageArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdClearColorImageArgs>& t) {
    VkClearColorValue& color = t->mColor;
    std::vector<VkImageSubresourceRange> clear_ranges;
    clear_ranges.reserve(t->mRanges.size());
    for (size_t i = 0; i < t->mRanges.size(); ++i) {
        clear_ranges.push_back(t->mRanges[i]);
    }
    spy->vkCmdClearColorImage(observer, commandBuf, t->mImage,
                                    t->mImageLayout, &color,
                                    clear_ranges.size(), clear_ranges.data());
    return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdClearDepthStencilImageArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdClearDepthStencilImageArgs>& t) {
  VkClearDepthStencilValue& depthStencil = t->mDepthStencil;
  std::vector<VkImageSubresourceRange> clear_ranges;
  clear_ranges.reserve(t->mRanges.size());
  for (size_t i = 0; i < t->mRanges.size(); ++i) {
    clear_ranges.push_back(t->mRanges[i]);
  }
  spy->vkCmdClearDepthStencilImage(
      observer, commandBuf, t->mImage, t->mImageLayout, &depthStencil,
      clear_ranges.size(), clear_ranges.data());
  return true;
}

template<>
bool inline CommandListRecreator<std::shared_ptr<vkCmdExecuteCommandsArgs>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdExecuteCommandsArgs>& t) {
    std::vector<VkCommandBuffer> command_buffers;
    command_buffers.reserve(t->mCommandBuffers.size());
    for (size_t i = 0; i < t->mCommandBuffers.size(); ++i) {
        command_buffers.push_back(t->mCommandBuffers[i]);
    }
    spy->vkCmdExecuteCommands(
        observer, commandBuf, command_buffers.size(), command_buffers.data());
    return true;
}

template <>
bool inline CommandListRecreator<std::shared_ptr<vkCmdNextSubpassArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdNextSubpassArgs>& t) {
  spy->vkCmdNextSubpass(observer, commandBuf, t->mContents);
  return true;
}

template <>
bool inline CommandListRecreator<std::shared_ptr<vkCmdSetEventArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdSetEventArgs>& t) {
  spy->vkCmdSetEvent(observer, commandBuf, t->mEvent, t->mStageMask);
  return true;
}

template <>
bool inline CommandListRecreator<std::shared_ptr<vkCmdResetEventArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<vkCmdResetEventArgs>& t) {
  spy->vkCmdResetEvent(observer, commandBuf, t->mEvent, t->mStageMask);
  return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdSetStencilCompareMaskArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdSetStencilCompareMaskArgs>& t) {
  spy->vkCmdSetStencilCompareMask(observer, commandBuf, t->mFaceMask,
                                        t->mCompareMask);
  return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdSetStencilWriteMaskArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdSetStencilWriteMaskArgs>& t) {
  spy->vkCmdSetStencilWriteMask(observer, commandBuf, t->mFaceMask,
                                      t->mWriteMask);
  return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdSetStencilReferenceArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdSetStencilReferenceArgs>& t) {
  spy->vkCmdSetStencilReference(observer, commandBuf, t->mFaceMask,
                                      t->mReference);
  return true;
}

template <>
bool inline CommandListRecreator<std::shared_ptr<vkCmdWaitEventsArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdWaitEventsArgs>& t) {
  std::vector<VkEvent> events;
  events.reserve(t->mEvents.size());
  for (size_t i = 0; i < t->mEvents.size(); ++i) {
    events.push_back(t->mEvents[i]);
  }
  std::vector<VkMemoryBarrier> memory_barriers;
  memory_barriers.reserve(t->mMemoryBarriers.size());
  for (size_t i = 0; i < t->mMemoryBarriers.size(); ++i) {
    memory_barriers.push_back(t->mMemoryBarriers[i]);
  }
  std::vector<VkBufferMemoryBarrier> buffer_memory_barriers;
  buffer_memory_barriers.reserve(t->mBufferMemoryBarriers.size());
  for (size_t i = 0; i < t->mBufferMemoryBarriers.size(); ++i) {
    buffer_memory_barriers.push_back(t->mBufferMemoryBarriers[i]);
  }
  std::vector<VkImageMemoryBarrier> image_memory_barriers;
  image_memory_barriers.reserve(t->mImageMemoryBarriers.size());
  for (size_t i = 0; i < t->mImageMemoryBarriers.size(); ++i) {
    image_memory_barriers.push_back(t->mImageMemoryBarriers[i]);
  }
  spy->vkCmdWaitEvents(
      observer, commandBuf, events.size(), events.data(), t->mSrcStageMask,
      t->mDstStageMask, memory_barriers.size(), memory_barriers.data(),
      buffer_memory_barriers.size(), buffer_memory_barriers.data(),
      image_memory_barriers.size(), image_memory_barriers.data());
   return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdDebugMarkerBeginEXTArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdDebugMarkerBeginEXTArgs>& t) {
  VkDebugMarkerMarkerInfoEXT info{
    VkStructureType::VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT,
    nullptr,
    const_cast<char*>(t->mMarkerName.c_str()),
    t->mColor,
  };
  spy->vkCmdDebugMarkerBeginEXT(observer, commandBuf, &info);
  return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdDebugMarkerEndEXTArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdDebugMarkerEndEXTArgs>& t) {
  spy->vkCmdDebugMarkerEndEXT(observer, commandBuf);
  return true;
}

template <>
bool inline CommandListRecreator<
    std::shared_ptr<vkCmdDebugMarkerInsertEXTArgs>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<vkCmdDebugMarkerInsertEXTArgs>& t) {
  VkDebugMarkerMarkerInfoEXT info{
    VkStructureType::VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT,
    nullptr,
    const_cast<char*>(t->mMarkerName.c_str()),
    t->mColor,
  };
  spy->vkCmdDebugMarkerInsertEXT(observer, commandBuf, &info);
  return true;
}
///////////////// End CommandBuffer Commands

template<typename T>
inline void AppendCommand(VkCommandBuffer, VulkanSpy*, std::shared_ptr<T>&) {
    GAPID_FATAL("Not implemented");
}

template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindPipelineArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindPipeline;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindPipeline, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetViewportArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetViewport;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetViewport, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetScissorArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetScissor;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetScissor, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetLineWidthArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetLineWidth;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetLineWidth, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetDepthBiasArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetDepthBias;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetDepthBias, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetBlendConstantsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetBlendConstants;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetBlendConstants, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetDepthBoundsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetDepthBounds;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetDepthBounds, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetStencilCompareMaskArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetStencilCompareMask;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetStencilCompareMask, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetStencilWriteMaskArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetStencilWriteMask;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetStencilWriteMask, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetStencilReferenceArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetStencilReference;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetStencilReference, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindDescriptorSetsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindDescriptorSets;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindDescriptorSets, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindIndexBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindIndexBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindIndexBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindVertexBuffersArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindVertexBuffers;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindVertexBuffers, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDraw;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDraw, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawIndexedArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDrawIndexed;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDrawIndexed, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDrawIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDrawIndirect, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawIndexedIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDrawIndexedIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDrawIndexedIndirect, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDispatchArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDispatch;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDispatch, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDispatchIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDispatchIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDispatchIndirect, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBlitImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBlitImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBlitImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyBufferToImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyBufferToImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyBufferToImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyImageToBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyImageToBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyImageToBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdUpdateBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdUpdateBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdUpdateBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdFillBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdFillBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdFillBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdClearColorImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdClearColorImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdClearColorImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdClearDepthStencilImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdClearDepthStencilImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdClearDepthStencilImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdClearAttachmentsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdClearAttachments;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdClearAttachments, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdResolveImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdResolveImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdResolveImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetEventArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetEvent;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetEvent, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdResetEventArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdResetEvent;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdResetEvent, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdWaitEventsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdWaitEvents;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdWaitEvents, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdPipelineBarrierArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdPipelineBarrier;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdPipelineBarrier, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBeginQueryArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBeginQuery;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBeginQuery, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdEndQueryArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdEndQuery;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdEndQuery, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdResetQueryPoolArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdResetQueryPool;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdResetQueryPool, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdWriteTimestampArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdWriteTimestamp;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdWriteTimestamp, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyQueryPoolResultsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyQueryPoolResults;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyQueryPoolResults, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdPushConstantsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdPushConstants;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdPushConstants, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBeginRenderPassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBeginRenderPass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBeginRenderPass, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdNextSubpassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdNextSubpass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdNextSubpass, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdEndRenderPassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdEndRenderPass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdEndRenderPass, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdExecuteCommandsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdExecuteCommands;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdExecuteCommands, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDebugMarkerBeginEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDebugMarkerBeginEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDebugMarkerBeginEXT, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDebugMarkerEndEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDebugMarkerEndEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDebugMarkerEndEXT, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDebugMarkerInsertEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDebugMarkerInsertEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDebugMarkerInsertEXT, map.size() - 1, 0, 0, nullptr
    );
}
//////////////// Command Buffer Insertion

template<typename Payload, typename Func>
void VulkanSpy::addCmd(CallObserver* observer, VkCommandBuffer cmdBuf, Payload payload, Func func) {
    if (is_recording_state()) return;
    AppendCommand(cmdBuf, this, payload);
    auto buffer = CommandBuffers[cmdBuf];
    buffer->commands.push_back([this, payload, func](CallObserver* observer) {
        ((*this).*func)(observer, nullptr, payload);
    });
}

inline bool RecreateCommand(CallObserver* observer,
        VkCommandBuffer buffer,
        VulkanSpy* spy, const CommandReference& reference) {
    switch(reference.mType) {
        case CommandType::cmd_vkCmdBindPipeline: {
            return CommandListRecreator<std::shared_ptr<vkCmdBindPipelineArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBindPipeline[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetViewport: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetViewportArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetViewport[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetScissor: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetScissorArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetScissor[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetLineWidth: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetLineWidthArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetLineWidth[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetDepthBias: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetDepthBiasArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetDepthBias[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetBlendConstants: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetBlendConstantsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetBlendConstants[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetDepthBounds: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetDepthBoundsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetDepthBounds[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetStencilCompareMask: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetStencilCompareMaskArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetStencilCompareMask[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetStencilWriteMask: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetStencilWriteMaskArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetStencilWriteMask[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetStencilReference: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetStencilReferenceArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetStencilReference[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdBindDescriptorSets: {
            return CommandListRecreator<std::shared_ptr<vkCmdBindDescriptorSetsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBindDescriptorSets[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdBindIndexBuffer: {
            return CommandListRecreator<std::shared_ptr<vkCmdBindIndexBufferArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBindIndexBuffer[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdBindVertexBuffers: {
            return CommandListRecreator<std::shared_ptr<vkCmdBindVertexBuffersArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBindVertexBuffers[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDraw: {
            return CommandListRecreator<std::shared_ptr<vkCmdDrawArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDraw[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDrawIndexed: {
            return CommandListRecreator<std::shared_ptr<vkCmdDrawIndexedArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDrawIndexed[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDrawIndirect: {
            return CommandListRecreator<std::shared_ptr<vkCmdDrawIndirectArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDrawIndirect[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDrawIndexedIndirect: {
            return CommandListRecreator<std::shared_ptr<vkCmdDrawIndexedIndirectArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDrawIndexedIndirect[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDispatch: {
            return CommandListRecreator<std::shared_ptr<vkCmdDispatchArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDispatch[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDispatchIndirect: {
            return CommandListRecreator<std::shared_ptr<vkCmdDispatchIndirectArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDispatchIndirect[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdCopyBuffer: {
            return CommandListRecreator<std::shared_ptr<vkCmdCopyBufferArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdCopyBuffer[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdCopyImage: {
            return CommandListRecreator<std::shared_ptr<vkCmdCopyImageArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdCopyImage[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdBlitImage: {
            return CommandListRecreator<std::shared_ptr<vkCmdBlitImageArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBlitImage[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdCopyBufferToImage: {
            return CommandListRecreator<std::shared_ptr<vkCmdCopyBufferToImageArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdCopyBufferToImage[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdCopyImageToBuffer: {
            return CommandListRecreator<std::shared_ptr<vkCmdCopyImageToBufferArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdCopyImageToBuffer[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdUpdateBuffer: {
            return CommandListRecreator<std::shared_ptr<vkCmdUpdateBufferArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdUpdateBuffer[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdFillBuffer: {
            return CommandListRecreator<std::shared_ptr<vkCmdFillBufferArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdFillBuffer[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdClearColorImage: {
            return CommandListRecreator<std::shared_ptr<vkCmdClearColorImageArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdClearColorImage[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdClearDepthStencilImage: {
            return CommandListRecreator<std::shared_ptr<vkCmdClearDepthStencilImageArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdClearDepthStencilImage[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdClearAttachments: {
            return CommandListRecreator<std::shared_ptr<vkCmdClearAttachmentsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdClearAttachments[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdResolveImage: {
            return CommandListRecreator<std::shared_ptr<vkCmdResolveImageArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdResolveImage[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdSetEvent: {
            return CommandListRecreator<std::shared_ptr<vkCmdSetEventArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdSetEvent[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdResetEvent: {
            return CommandListRecreator<std::shared_ptr<vkCmdResetEventArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdResetEvent[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdWaitEvents: {
            return CommandListRecreator<std::shared_ptr<vkCmdWaitEventsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdWaitEvents[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdPipelineBarrier: {
            return CommandListRecreator<std::shared_ptr<vkCmdPipelineBarrierArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdPipelineBarrier[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdBeginQuery: {
            return CommandListRecreator<std::shared_ptr<vkCmdBeginQueryArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBeginQuery[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdEndQuery: {
            return CommandListRecreator<std::shared_ptr<vkCmdEndQueryArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdEndQuery[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdResetQueryPool: {
            return CommandListRecreator<std::shared_ptr<vkCmdResetQueryPoolArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdResetQueryPool[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdWriteTimestamp: {
            return CommandListRecreator<std::shared_ptr<vkCmdWriteTimestampArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdWriteTimestamp[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdCopyQueryPoolResults: {
            return CommandListRecreator<std::shared_ptr<vkCmdCopyQueryPoolResultsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdCopyQueryPoolResults[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdPushConstants: {
            return CommandListRecreator<std::shared_ptr<vkCmdPushConstantsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdPushConstants[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdBeginRenderPass: {
            return CommandListRecreator<std::shared_ptr<vkCmdBeginRenderPassArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdBeginRenderPass[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdNextSubpass: {
            return CommandListRecreator<std::shared_ptr<vkCmdNextSubpassArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdNextSubpass[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdEndRenderPass: {
            return CommandListRecreator<std::shared_ptr<vkCmdEndRenderPassArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdEndRenderPass[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdExecuteCommands: {
            return CommandListRecreator<std::shared_ptr<vkCmdExecuteCommandsArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdExecuteCommands[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDebugMarkerBeginEXT: {
            return CommandListRecreator<std::shared_ptr<vkCmdDebugMarkerBeginEXTArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDebugMarkerBeginEXT[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDebugMarkerEndEXT: {
            return CommandListRecreator<std::shared_ptr<vkCmdDebugMarkerEndEXTArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDebugMarkerEndEXT[reference.mMapIndex]
            );
        }
        case CommandType::cmd_vkCmdDebugMarkerInsertEXT: {
            return CommandListRecreator<std::shared_ptr<vkCmdDebugMarkerInsertEXTArgs>>()(buffer, observer, spy,
                spy->CommandBuffers[reference.mBuffer]->mBufferCommands.mvkCmdDebugMarkerInsertEXT[reference.mMapIndex]
            );
        }
        default:
            return false;
    }
}

inline void VulkanSpy::notifyPendingCommandAdded(CallObserver*, VkQueue) {}
