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


inline void VulkanSpy::notifyPendingCommandAdded(CallObserver*, VkQueue) {}

template<typename VkErrorType>
inline void VulkanSpy::onVkError(CallObserver* observer, VkErrorType err) {
  GAPID_WARNING("Unhandled Vulkan error");
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, std::shared_ptr<ERR_INVALID_HANDLE> err) {
  GAPID_WARNING("Error: Invalid %s: %llu", err->mhandleType.c_str(), err->mhandle)
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, std::shared_ptr<ERR_NULL_POINTER> err) {
  GAPID_WARNING("Error: Null pointer of %s", err->mpointerType.c_str())
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, std::shared_ptr<ERR_UNRECOGNIZED_EXTENSION> err) {
  GAPID_WARNING("Error: Unrecognized extension: %s", err->mname.c_str())
}
