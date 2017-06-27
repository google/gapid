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

template<typename T>
struct CommandListRecreator {
    void inline operator()(VkCommandBuffer commandBuf, CallObserver* observer,
        VulkanSpy* spy, const T& t) {
        GAPID_FATAL("Not implemented");
    }
private:
};


////////// Command Buffer Commands
template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdUpdateBufferData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdUpdateBufferData>& t) {
    if (!spy->Buffers.count(t->mdstBuffer)) {
        return;
    }
    spy->RecreateCmdUpdateBuffer(observer, commandBuf,
        t->mdstBuffer, t->mdstOffset, t->mdataSize, t->bufferData.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdEndRenderPassData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdEndRenderPassData>&) {
    spy->RecreateCmdEndRenderPass(observer, commandBuf);
}


template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdPipelineBarrierData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdPipelineBarrierData>& t) {
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
            return;
        }
    }
    std::vector<VkImageMemoryBarrier> image_memory_barriers;
    memory_barriers.reserve(t->mImageMemoryBarriers.size());
    for (size_t i = 0; i < t->mImageMemoryBarriers.size(); ++i) {
        image_memory_barriers.push_back(t->mImageMemoryBarriers[i]);
        if (!spy->Images.count(t->mImageMemoryBarriers[i].mimage)) {
            return;
        }
    }
    spy->RecreateCmdPipelineBarrier(observer, commandBuf,
        t->mSrcStageMask, t->mDstStageMask, t->mDependencyFlags,
        memory_barriers.size(), memory_barriers.data(),
        buffer_memory_barriers.size(), buffer_memory_barriers.data(),
        image_memory_barriers.size(), image_memory_barriers.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdCopyBufferData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdCopyBufferData>& t) {
    if (!spy->Buffers.count(t->mSrcBuffer) ||
        !spy->Buffers.count(t->mDstBuffer)) {
        return;
    }
    std::vector<VkBufferCopy> buffer_copies;
    buffer_copies.reserve(t->mCopyRegions.size());
    for (size_t i = 0; i < t->mCopyRegions.size(); ++i) {
        buffer_copies.push_back(t->mCopyRegions[i]);
    }
    spy->RecreateCmdCopyBuffer(observer, commandBuf,
        t->mSrcBuffer, t->mDstBuffer,
        buffer_copies.size(), buffer_copies.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdResolveImageData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdResolveImageData>& t) {
    std::vector<VkImageResolve> image_resolves;
    image_resolves.reserve(t->mResolveRegions.size());
    if (!spy->Images.count(t->mSrcImage) ||
        !spy->Images.count(t->mDstImage)) {
        return;
    }
    for (size_t i = 0; i < t->mResolveRegions.size(); ++i) {
        image_resolves.push_back(t->mResolveRegions[i]);
    }
    spy->RecreateCmdResolveImage(observer, commandBuf,
        t->mSrcImage, t->mSrcImageLayout, t->mDstImage, t->mDstImageLayout,
        image_resolves.size(), image_resolves.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBeginRenderPassData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBeginRenderPassData>& t) {
    if (!spy->RenderPasses.count(t->mRenderPass) ||
        !spy->Framebuffers.count(t->mFramebuffer)) {
        return;
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
    spy->RecreateCmdBeginRenderPass(observer, commandBuf,
        &begin_info, t->mContents);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBindDescriptorSetsData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBindDescriptorSetsData>& t) {
    std::vector<uint32_t> dynamic_offsets;
    dynamic_offsets.reserve(t->mDynamicOffsets.size());
    for (size_t i = 0; i < t->mDynamicOffsets.size(); ++i) {
        dynamic_offsets.push_back(t->mDynamicOffsets[i]);
    }
    std::vector<VkDescriptorSet> descriptor_sets;
    descriptor_sets.reserve(t->mDescriptorSets.size());
    for (size_t i = 0; i < t->mDescriptorSets.size(); ++i) {
        if (!spy->DescriptorSets.count(t->mDescriptorSets[i])) {
            return;
        }
        descriptor_sets.push_back(t->mDescriptorSets[i]);
    }

    spy->RecreateCmdBindDescriptorSets(observer, commandBuf,
        t->mPipelineBindPoint, t->mLayout, t->mFirstSet,
        descriptor_sets.size(), descriptor_sets.data(),
        dynamic_offsets.size(), dynamic_offsets.data());
}


template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBindVertexBuffersData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBindVertexBuffersData>& t) {
    std::vector<VkBuffer> buffers;
    buffers.reserve(t->mBuffers.size());
    for (size_t i = 0; i < t->mBuffers.size(); ++i) {
        if (!spy->Buffers.count(t->mBuffers[i])) {
            return;
        }
        buffers.push_back(t->mBuffers[i]);
    }
    std::vector<uint64_t> offsets;
    offsets.reserve(t->mOffsets.size());
    for (size_t i = 0; i < t->mOffsets.size(); ++i) {
        offsets.push_back(t->mOffsets[i]);
    }

    spy->RecreateCmdBindVertexBuffers(observer, commandBuf,
        t->mFirstBinding, t->mBindingCount, buffers.data(), offsets.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBindIndexBufferData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBindIndexBufferData>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return;
    }
    spy->RecreateCmdBindIndexBuffer(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mIndexType);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdDrawIndirectData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdDrawIndirectData>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return;
    }
    spy->RecreateCmdDrawIndirect(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mDrawCount, t->mStride);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdDrawIndexedIndirectData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdDrawIndexedIndirectData>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return;
    }
    spy->RecreateCmdDrawIndexedIndirect(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mDrawCount, t->mStride);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetDepthBiasData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetDepthBiasData>& t) {
    spy->RecreateCmdSetDepthBias(observer, commandBuf,
        t->mDepthBiasConstantFactor, t->mDepthBiasClamp, t->mDepthBiasSlopeFactor);
}

template <>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetDepthBoundsData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetDepthBoundsData>& t) {
  spy->RecreateCmdSetDepthBounds(observer, commandBuf, t->mMinDepthBounds, t->mMaxDepthBounds);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetLineWidthData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetLineWidthData>& t) {
    spy->RecreateCmdSetLineWidth(observer, commandBuf, t->mLineWidth);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdFillBufferData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdFillBufferData>& t) {
    if (!spy->Buffers.count(t->mBuffer)) {
        return;
    }
    spy->RecreateCmdFillBuffer(observer, commandBuf, t->mBuffer, t->mDstBuffer, t->mSize, t->mData);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetBlendConstantsData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetBlendConstantsData>& t) {
    float constants[4] = {
        t->mR,
        t->mG,
        t->mB,
        t->mA
    };
    spy->RecreateCmdSetBlendConstants(observer, commandBuf, constants);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBindPipelineData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBindPipelineData>& t) {
    if (!spy->GraphicsPipelines.count(t->mPipeline) &&
        !spy->ComputePipelines.count(t->mPipeline)) {
        return;
    }
    spy->RecreateCmdBindPipeline(observer, commandBuf,
        t->mPipelineBindPoint, t->mPipeline);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBeginQueryData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBeginQueryData>& t) {
    if (!spy->QueryPools.count(t->mQueryPool)) {
        return;
    }
    spy->RecreateCmdBeginQuery(observer, commandBuf,
        t->mQueryPool, t->mQuery, t->mFlags);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdEndQueryData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdEndQueryData>& t) {
    if (!spy->QueryPools.count(t->mQueryPool)) {
        return;
    }
    spy->RecreateCmdEndQuery(observer, commandBuf,
        t->mQueryPool, t->mQuery);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdResetQueryPoolData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdResetQueryPoolData>& t) {
    if (!spy->QueryPools.count(t->mQueryPool)) {
        return;
    }
    spy->RecreateCmdResetQueryPool(observer, commandBuf,
        t->mQueryPool, t->mFirstQuery, t->mQueryCount);
}

template <>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdWriteTimestampData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdWriteTimestampData>& t) {
    spy->RecreateCmdWriteTimestamp(observer, commandBuf, t->mPipelineStage,
                                  t->mQueryPool, t->mQuery);
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCmdCopyQueryPoolResultsData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdCopyQueryPoolResultsData>& t) {
  if (!spy->QueryPools.count(t->mQueryPool)) {
    return;
  }
  spy->RecreateCmdCopyQueryPoolResults(
      observer, commandBuf, t->mQueryPool, t->mFirstQuery, t->mQueryCount,
      t->mDstBuffer, t->mDstOffset, t->mStride, t->mFlags);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCopyBufferToImageData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCopyBufferToImageData>& t) {
    if (!spy->Buffers.count(t->mSrcBuffer) ||
        !spy->Images.count(t->mDstImage)) {
        return;
    }
    std::vector<VkBufferImageCopy> regions;
    regions.reserve(t->mRegions.size());
    for (size_t i = 0; i < t->mRegions.size(); ++i) {
        regions.push_back(t->mRegions[i]);
    }
    spy->RecreateCmdCopyBufferToImage(observer, commandBuf,
        t->mSrcBuffer, t->mDstImage, t->mLayout, regions.size(), regions.data());
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCopyImageToBufferData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCopyImageToBufferData>& t) {
  if (!spy->Images.count(t->mSrcImage) || !spy->Buffers.count(t->mDstBuffer)) {
    return;
  }
  std::vector<VkBufferImageCopy> regions;
  regions.reserve(t->mRegions.size());
  for (size_t i = 0; i < t->mRegions.size(); ++i) {
    regions.push_back(t->mRegions[i]);
  }
  spy->RecreateCmdCopyImageToBuffer(observer, commandBuf, t->mSrcImage,
                                    t->mSrcImageLayout, t->mDstBuffer,
                                    regions.size(), regions.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBlitImageData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBlitImageData>& t) {
    if (!spy->Images.count(t->mSrcImage) ||
        !spy->Images.count(t->mDstImage)) {
        return;
    }
    std::vector<VkImageBlit> regions;
    regions.reserve(t->mRegions.size());
    for (size_t i = 0; i < t->mRegions.size(); ++i) {
        regions.push_back(t->mRegions[i]);
    }
    spy->RecreateCmdBlitImage(
        observer, commandBuf, t->mSrcImage, t->mSrcImageLayout, t->mDstImage,
        t->mDstImageLayout, regions.size(), regions.data(), t->mFilter);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdCopyImageData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdCopyImageData>& t) {
    if (!spy->Images.count(t->mSrcImage) ||
        !spy->Images.count(t->mDstImage)) {
        return;
    }
    std::vector<VkImageCopy> regions;
    regions.reserve(t->mRegions.size());
    for (size_t i = 0; i < t->mRegions.size(); ++i) {
        regions.push_back(t->mRegions[i]);
    }
    spy->RecreateCmdCopyImage(observer, commandBuf,
        t->mSrcImage, t->mSrcImageLayout,
        t->mDstImage, t->mDstImageLayout,
        regions.size(), regions.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdPushConstantsData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdPushConstantsData>& t) {
    if (!spy->PipelineLayouts.count(t->mLayout)) {
        return;
    }
    spy->RecreateCmdPushConstants(observer, commandBuf,
        t->mLayout, t->mStageFlags,
        t->mOffset, t->mSize,
        t->pushConstantData.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetScissorData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetScissorData>& t) {
    std::vector<VkRect2D> rects;
    rects.reserve(t->mScissors.size());
    for (size_t i = 0; i < t->mScissors.size(); ++i) {
        rects.push_back(t->mScissors[i]);
    }
    spy->RecreateCmdSetScissor(observer, commandBuf,
        t->mFirstScissor, rects.size(), rects.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetViewportData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetViewportData>& t) {
    std::vector<VkViewport> viewports;
    viewports.reserve(t->mViewports.size());
    for (size_t i = 0; i < t->mViewports.size(); ++i) {
        viewports.push_back(t->mViewports[i]);
    }
    spy->RecreateCmdSetViewport(observer, commandBuf,
        t->mFirstViewport, viewports.size(), viewports.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdDrawData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdDrawData>& t) {
    spy->RecreateCmdDraw(observer, commandBuf,
        t->mVertexCount, t->mInstanceCount, t->mFirstVertex, t->mFirstInstance);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdDispatchData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdDispatchData>& t) {
    spy->RecreateCmdDispatch(observer, commandBuf,
        t->mGroupCountX, t->mGroupCountY, t->mGroupCountZ);
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCmdDispatchIndirectData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdDispatchIndirectData>& t) {
  spy->RecreateCmdDispatchIndirect(observer, commandBuf, t->mBuffer,
                                   t->mOffset);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdDrawIndexedData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdDrawIndexedData>& t) {
    spy->RecreateCmdDrawIndexed(observer, commandBuf,
        t->mIndexCount, t->mInstanceCount, t->mFirstIndex, t->mVertexOffset, t->mFirstInstance);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdClearAttachmentsData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdClearAttachmentsData>& t) {
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
    spy->RecreateCmdClearAttachments(observer, commandBuf, attachments.size(),
                                     attachments.data(), rects.size(),
                                     rects.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdClearColorImageData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdClearColorImageData>& t) {
    VkClearColorValue& color = t->mColor;
    std::vector<VkImageSubresourceRange> clear_ranges;
    clear_ranges.reserve(t->mRanges.size());
    for (size_t i = 0; i < t->mRanges.size(); ++i) {
        clear_ranges.push_back(t->mRanges[i]);
    }
    spy->RecreateCmdClearColorImage(observer, commandBuf, t->mImage,
                                    t->mImageLayout, &color,
                                    clear_ranges.size(), clear_ranges.data());
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCmdClearDepthStencilImageData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdClearDepthStencilImageData>& t) {
  VkClearDepthStencilValue& depthStencil = t->mDepthStencil;
  std::vector<VkImageSubresourceRange> clear_ranges;
  clear_ranges.reserve(t->mRanges.size());
  for (size_t i = 0; i < t->mRanges.size(); ++i) {
    clear_ranges.push_back(t->mRanges[i]);
  }
  spy->RecreateCmdClearDepthStencilImage(
      observer, commandBuf, t->mImage, t->mImageLayout, &depthStencil,
      clear_ranges.size(), clear_ranges.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdExecuteCommandsData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdExecuteCommandsData>& t) {
    std::vector<VkCommandBuffer> command_buffers;
    command_buffers.reserve(t->mCommandBuffers.size());
    for (size_t i = 0; i < t->mCommandBuffers.size(); ++i) {
        command_buffers.push_back(t->mCommandBuffers[i]);
    }
    spy->RecreateCmdExecuteCommands(
        observer, commandBuf, command_buffers.size(), command_buffers.data());
}

template <>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdNextSubpassData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdNextSubpassData>& t) {
  spy->RecreateCmdNextSubpass(observer, commandBuf, t->mContents);
}

template <>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdSetEventData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdSetEventData>& t) {
  spy->RecreateCmdSetEvent(observer, commandBuf, t->mEvent, t->mStageMask);
}

template <>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdResetEventData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdResetEventData>& t) {
  spy->RecreateCmdResetEvent(observer, commandBuf, t->mEvent, t->mStageMask);
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCmdSetStencilCompareMaskData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdSetStencilCompareMaskData>& t) {
  spy->RecreateCmdSetStencilCompareMask(observer, commandBuf, t->mFaceMask,
                                        t->mCompareMask);
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCmdSetStencilWriteMaskData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdSetStencilWriteMaskData>& t) {
  spy->RecreateCmdSetStencilWriteMask(observer, commandBuf, t->mFaceMask,
                                      t->mWriteMask);
}

template <>
void inline CommandListRecreator<
    std::shared_ptr<RecreateCmdSetStencilReferenceData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdSetStencilReferenceData>& t) {
  spy->RecreateCmdSetStencilReference(observer, commandBuf, t->mFaceMask,
                                      t->mReference);
}

template <>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdWaitEventsData>>::
operator()(VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
           const std::shared_ptr<RecreateCmdWaitEventsData>& t) {
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
  spy->RecreateCmdWaitEvents(
      observer, commandBuf, events.size(), events.data(), t->mSrcStageMask,
      t->mDstStageMask, memory_barriers.size(), memory_barriers.data(),
      buffer_memory_barriers.size(), buffer_memory_barriers.data(),
      image_memory_barriers.size(), image_memory_barriers.data());
}
///////////////// End CommandBuffer Commands

template<typename RecreatePayload, typename Payload, typename Func>
void VulkanSpy::addCmd(CallObserver* observer, VkCommandBuffer cmdBuf, RecreatePayload recreate, Payload payload, Func func) {
    auto buffer = CommandBuffers[cmdBuf];
    buffer->commands.push_back([this, payload, func](CallObserver* observer) {
        ((*this).*func)(observer, nullptr, payload);
    });
    buffer->recreateCommands.push_back([this, recreate, cmdBuf](CallObserver* observer) {
        CommandListRecreator<RecreatePayload>()(cmdBuf, observer, this, recreate);
    });
}
