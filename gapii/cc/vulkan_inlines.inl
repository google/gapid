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
inline void AppendCommand(VkCommandBuffer, VulkanSpy*, std::shared_ptr<T>&) {
    GAPID_FATAL("Not implemented");
}

template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindPipelineArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBindPipeline;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBindPipeline, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetViewportArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetViewport;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetViewport, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetScissorArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetScissor;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetScissor, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetLineWidthArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetLineWidth;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetLineWidth, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetDepthBiasArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetDepthBias;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetDepthBias, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetBlendConstantsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetBlendConstants;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetBlendConstants, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetDepthBoundsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetDepthBounds;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetDepthBounds, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetStencilCompareMaskArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetStencilCompareMask;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetStencilCompareMask, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetStencilWriteMaskArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetStencilWriteMask;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetStencilWriteMask, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetStencilReferenceArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetStencilReference;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetStencilReference, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindDescriptorSetsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBindDescriptorSets;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBindDescriptorSets, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindIndexBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBindIndexBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBindIndexBuffer, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBindVertexBuffersArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBindVertexBuffers;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBindVertexBuffers, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDraw;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDraw, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawIndexedArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDrawIndexed;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDrawIndexed, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDrawIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDrawIndirect, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDrawIndexedIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDrawIndexedIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDrawIndexedIndirect, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDispatchArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDispatch;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDispatch, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDispatchIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDispatchIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDispatchIndirect, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdCopyBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdCopyBuffer, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdCopyImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdCopyImage, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBlitImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBlitImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBlitImage, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyBufferToImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdCopyBufferToImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdCopyBufferToImage, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyImageToBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdCopyImageToBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdCopyImageToBuffer, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdUpdateBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdUpdateBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdUpdateBuffer, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdFillBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdFillBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdFillBuffer, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdClearColorImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdClearColorImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdClearColorImage, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdClearDepthStencilImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdClearDepthStencilImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdClearDepthStencilImage, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdClearAttachmentsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdClearAttachments;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdClearAttachments, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdResolveImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdResolveImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdResolveImage, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdSetEventArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdSetEvent;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdSetEvent, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdResetEventArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdResetEvent;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdResetEvent, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdWaitEventsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdWaitEvents;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdWaitEvents, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdPipelineBarrierArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdPipelineBarrier;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdPipelineBarrier, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBeginQueryArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBeginQuery;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBeginQuery, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdEndQueryArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdEndQuery;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdEndQuery, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdResetQueryPoolArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdResetQueryPool;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdResetQueryPool, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdWriteTimestampArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdWriteTimestamp;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdWriteTimestamp, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdCopyQueryPoolResultsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdCopyQueryPoolResults;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdCopyQueryPoolResults, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdPushConstantsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdPushConstants;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdPushConstants, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdBeginRenderPassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdBeginRenderPass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdBeginRenderPass, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdNextSubpassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdNextSubpass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdNextSubpass, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdEndRenderPassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdEndRenderPass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdEndRenderPass, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdExecuteCommandsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdExecuteCommands;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdExecuteCommands, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDebugMarkerBeginEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDebugMarkerBeginEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDebugMarkerBeginEXT, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDebugMarkerEndEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDebugMarkerEndEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDebugMarkerEndEXT, map.size() - 1, 0, 0
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, std::shared_ptr<vkCmdDebugMarkerInsertEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mVkCmdDebugMarkerInsertEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    references[references.size()] = CommandReference(
        buffer, references.size(), CommandType::cmd_vkCmdDebugMarkerInsertEXT, map.size() - 1, 0, 0
    );
}
//////////////// Command Buffer Insertion

template<typename Payload, typename Func>
void VulkanSpy::addCmd(CallObserver* observer, VkCommandBuffer cmdBuf, Payload payload, Func func) {
    AppendCommand(cmdBuf, this, payload);
    auto buffer = CommandBuffers[cmdBuf];
    buffer->commands.push_back([this, payload, func](CallObserver* observer) {
        ((*this).*func)(observer, nullptr, payload);
    });
}

inline void VulkanSpy::notifyPendingCommandAdded(CallObserver*, VkQueue) {}