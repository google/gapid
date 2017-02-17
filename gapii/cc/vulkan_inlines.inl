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
void inline CommandListRecreator<std::shared_ptr<RecreateUpdateBufferData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateUpdateBufferData>& t) {
    spy->RecreateUpdateBuffer(observer, commandBuf,
        t->mdstBuffer, t->mdstOffset, t->mdataSize, t->bufferData.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateEndRenderPassData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateEndRenderPassData>&) {
    spy->RecreateEndRenderPass(observer, commandBuf);
}


template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdPipelineBarrierData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdPipelineBarrierData>& t) {
    std::vector<VkMemoryBarrier> memory_barriers;
    for (size_t i = 0; i < t->mMemoryBarriers.size(); ++i) {
        memory_barriers.push_back(t->mMemoryBarriers[i]);
    }
    std::vector<VkBufferMemoryBarrier> buffer_memory_barriers;
    for (size_t i = 0; i < t->mBufferMemoryBarriers.size(); ++i) {
        buffer_memory_barriers.push_back(t->mBufferMemoryBarriers[i]);
    }
    std::vector<VkImageMemoryBarrier> image_memory_barriers;
    for (size_t i = 0; i < t->mImageMemoryBarriers.size(); ++i) {
        image_memory_barriers.push_back(t->mImageMemoryBarriers[i]);
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
    std::vector<VkBufferCopy> buffer_copies;
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
    std::vector<VkClearValue> clear_values;
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
    for (size_t i = 0; i < t->mDynamicOffsets.size(); ++i) {
        dynamic_offsets.push_back(t->mDynamicOffsets[i]);
    }
    std::vector<VkDescriptorSet> descriptor_sets;
    for (size_t i = 0; i < t->mDescriptorSets.size(); ++i) {
        descriptor_sets.push_back(t->mDescriptorSets[i]);
    }

    spy->RecreateCmdBindDescriptorSets(observer, commandBuf,
        t->mPipelineBindPoint, t->mLayout, t->mFirstSet,
        descriptor_sets.size(), descriptor_sets.data(),
        dynamic_offsets.size(), dynamic_offsets.data());
}


template<>
void inline CommandListRecreator<std::shared_ptr<RecreateBindVertexBuffersData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateBindVertexBuffersData>& t) {
    std::vector<VkBuffer> buffers;
    for (size_t i = 0; i < t->mBuffers.size(); ++i) {
        buffers.push_back(t->mBuffers[i]);
    }
    std::vector<uint64_t> device_sizes;
    for (size_t i = 0; i < t->mOffsets.size(); ++i) {
        device_sizes.push_back(t->mOffsets[i]);
    }

    spy->RecreateBindVertexBuffers(observer, commandBuf,
        t->mFirstBinding, t->mBindingCount, buffers.data(), device_sizes.data());
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBindIndexBufferData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBindIndexBufferData>& t) {
    spy->RecreateCmdBindIndexBuffer(observer, commandBuf,
        t->mBuffer, t->mOffset, t->mIndexType);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdBindPipelineData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdBindPipelineData>& t) {
    spy->RecreateCmdBindPipeline(observer, commandBuf,
        t->mPipelineBindPoint, t->mPipeline);
}

template<>
void inline CommandListRecreator<std::shared_ptr<RecreateCmdDrawIndexedData>>::operator()(
    VkCommandBuffer commandBuf, CallObserver* observer, VulkanSpy* spy,
    const std::shared_ptr<RecreateCmdDrawIndexedData>& t) {
    spy->RecreateCmdDrawIndexed(observer, commandBuf,
        t->mIndexCount, t->mInstanceCount, t->mFirstIndex, t->mVertexOffset, t->mFirstInstance);
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
