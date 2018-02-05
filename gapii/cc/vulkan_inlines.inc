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
inline void AppendCommand(VkCommandBuffer, VulkanSpy*, gapil::Ref<T>&) {
    GAPID_FATAL("Not implemented");
}

template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBindPipelineArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindPipeline;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindPipeline, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetViewportArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetViewport;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetViewport, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetScissorArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetScissor;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetScissor, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetLineWidthArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetLineWidth;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetLineWidth, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetDepthBiasArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetDepthBias;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetDepthBias, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetBlendConstantsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetBlendConstants;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetBlendConstants, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetDepthBoundsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetDepthBounds;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetDepthBounds, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetStencilCompareMaskArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetStencilCompareMask;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetStencilCompareMask, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetStencilWriteMaskArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetStencilWriteMask;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetStencilWriteMask, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetStencilReferenceArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetStencilReference;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetStencilReference, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBindDescriptorSetsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindDescriptorSets;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindDescriptorSets, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBindIndexBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindIndexBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindIndexBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBindVertexBuffersArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBindVertexBuffers;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBindVertexBuffers, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDrawArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDraw;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDraw, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDrawIndexedArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDrawIndexed;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDrawIndexed, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDrawIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDrawIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDrawIndirect, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDrawIndexedIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDrawIndexedIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDrawIndexedIndirect, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDispatchArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDispatch;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDispatch, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDispatchIndirectArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDispatchIndirect;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDispatchIndirect, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdCopyBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdCopyImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBlitImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBlitImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBlitImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdCopyBufferToImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyBufferToImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyBufferToImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdCopyImageToBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyImageToBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyImageToBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdUpdateBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdUpdateBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdUpdateBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdFillBufferArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdFillBuffer;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdFillBuffer, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdClearColorImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdClearColorImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdClearColorImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdClearDepthStencilImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdClearDepthStencilImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdClearDepthStencilImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdClearAttachmentsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdClearAttachments;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdClearAttachments, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdResolveImageArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdResolveImage;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdResolveImage, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdSetEventArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdSetEvent;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdSetEvent, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdResetEventArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdResetEvent;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdResetEvent, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdWaitEventsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdWaitEvents;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdWaitEvents, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdPipelineBarrierArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdPipelineBarrier;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdPipelineBarrier, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBeginQueryArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBeginQuery;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBeginQuery, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdEndQueryArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdEndQuery;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdEndQuery, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdResetQueryPoolArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdResetQueryPool;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdResetQueryPool, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdWriteTimestampArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdWriteTimestamp;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdWriteTimestamp, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdCopyQueryPoolResultsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdCopyQueryPoolResults;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdCopyQueryPoolResults, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdPushConstantsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdPushConstants;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdPushConstants, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdBeginRenderPassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdBeginRenderPass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdBeginRenderPass, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdNextSubpassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdNextSubpass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdNextSubpass, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdEndRenderPassArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdEndRenderPass;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdEndRenderPass, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdExecuteCommandsArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdExecuteCommands;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdExecuteCommands, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDebugMarkerBeginEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDebugMarkerBeginEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDebugMarkerBeginEXT, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDebugMarkerEndEXTArgs>& args) {
    auto& map = spy->CommandBuffers[buffer]->mBufferCommands.mvkCmdDebugMarkerEndEXT;
    map[map.size()] = args;
    auto& references = spy->CommandBuffers[buffer]->mCommandReferences;
    const uint32_t reference_idx = references.size();
    references[reference_idx] = CommandReference(
        buffer, reference_idx, CommandType::cmd_vkCmdDebugMarkerEndEXT, map.size() - 1, 0, 0, nullptr
    );
}
template<>
inline void AppendCommand(VkCommandBuffer buffer, VulkanSpy* spy, gapil::Ref<vkCmdDebugMarkerInsertEXTArgs>& args) {
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
inline void VulkanSpy::onVkError(CallObserver*, gapil::Ref<ERR_INVALID_HANDLE> err) {
  GAPID_WARNING("Error: Invalid %s: %llu", err->mhandleType.c_str(), err->mhandle)
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, gapil::Ref<ERR_NULL_POINTER> err) {
  GAPID_WARNING("Error: Null pointer of %s", err->mpointerType.c_str())
}

template<>
inline void VulkanSpy::onVkError(CallObserver*, gapil::Ref<ERR_UNRECOGNIZED_EXTENSION> err) {
  GAPID_WARNING("Error: Unrecognized extension: %s", err->mname.c_str())
}
