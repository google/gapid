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
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <functional>

namespace gapid2 {
class decoder;
class encoder;
class handle_fixer;
class state_block;
class temporary_allocator;

void _custom_clone_VkClearValue(
    state_block*,
    const VkClearValue& src,
    VkClearValue& dst,
    temporary_allocator* mem,
    std::function<bool(const VkClearValue& self)> _VkClearValue_color_valid);

void _custom_deserialize_vkCmdPushConstants_pValues(
    state_block*,
    VkCommandBuffer commandBuffer,
    VkPipelineLayout layout,
    VkShaderStageFlags stageFlags,
    uint32_t offset,
    uint32_t size,
    void*& pValues,
    decoder* dec);

void _custom_clone_VkClearColorValue(state_block*,
                                            const VkClearColorValue& src,
                                            VkClearColorValue& dst,
                                            temporary_allocator* mem);

void _custom_serialize_VkClearColorValue(state_block*,
                                                const VkClearColorValue& value,
                                                encoder* enc);

void _custom_serialize_VkClearValue(
    state_block*,
    const VkClearValue& value,
    encoder* enc,
    std::function<bool(const VkClearValue& self)> _VkClearValue_color_valid);

void _custom_deserialize_VkClearColorValue(state_block*,
                                                  VkClearColorValue& value,
                                                  decoder* dec);

void _custom_deserialize_VkClearValue(state_block*,
                                             VkClearValue& value,
                                             decoder* dec);

uint64_t get_VkDescriptorUpdateTemplate_size(
    state_block* state_block_,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate);

void _custom_serialize_vkUpdateDescriptorSetWithTemplate_pData(
    state_block* state_block_,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    const void* pData,
    encoder* enc);

const void* _custom_unwrap_vkUpdateDescriptorSetWithTemplate_pData(
    state_block* state_block_,
    temporary_allocator* _allocator,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    const void* pData);

void _custom_serialize_vkMapMemory_ppData(state_block*,
                                                 VkDevice device,
                                                 VkDeviceMemory memory,
                                                 VkDeviceSize offset,
                                                 VkDeviceSize size,
                                                 VkMemoryMapFlags flags,
                                                 void** ppData,
                                                 encoder* enc);

void _custom_serialize_vkGetMemoryHostPointerPropertiesEXT_pHostPointer(
    state_block* state_block_, VkDevice device, VkExternalMemoryHandleTypeFlagBits handleType, const void* pHostPointer, VkMemoryHostPointerPropertiesEXT* pMemoryHostPointerProperties, encoder* enc);
void _custom_deserialize_vkGetMemoryHostPointerPropertiesEXT_pHostPointer(
    state_block* state_block_, VkDevice device, VkExternalMemoryHandleTypeFlagBits handleType, const void* pHostPointer, VkMemoryHostPointerPropertiesEXT* pMemoryHostPointerProperties, decoder* dec);

void _custom_serialize_vkGetQueryPoolResults_pData(state_block*,
                                                          VkDevice device,
                                                          VkQueryPool queryPool,
                                                          uint32_t firstQuery,
                                                          uint32_t queryCount,
                                                          size_t dataSize,
                                                          void* pData,
                                                          VkDeviceSize stride,
                                                          VkQueryResultFlags flags,
                                                          encoder* enc);

void _custom_serialize_vkGetPipelineCacheData_pData(
    state_block*,
    VkDevice device,
    VkPipelineCache pipelineCache,
    size_t* pDataSize,
    void* pData,
    encoder* enc);

void _custom_serialize_vkCmdUpdateBuffer_pData(state_block*,
                                                      VkCommandBuffer commandBuffer,
                                                      VkBuffer dstBuffer,
                                                      VkDeviceSize dstOffset,
                                                      VkDeviceSize dataSize,
                                                      const void* pData,
                                                      encoder* enc);

void _custom_serialize_vkCmdPushConstants_pValues(state_block*,
                                                         VkCommandBuffer commandBuffer,
                                                         VkPipelineLayout layout,
                                                         VkShaderStageFlags stageFlags,
                                                         uint32_t offset,
                                                         uint32_t size,
                                                         const void* pValues,
                                                         encoder* enc);

void _custom_deserialize_vkUpdateDescriptorSetWithTemplate_pData(
    state_block*,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    void*& pData,
    decoder* dec);

void _custom_deserialize_vkMapMemory_ppData(state_block*,
                                                   VkDevice device,
                                                   VkDeviceMemory memory,
                                                   VkDeviceSize offset,
                                                   VkDeviceSize size,
                                                   VkMemoryMapFlags flags,
                                                   void**& ppData,
                                                   decoder* dec);

void _custom_deserialize_vkGetQueryPoolResults_pData(state_block*,
                                                            VkDevice device,
                                                            VkQueryPool queryPool,
                                                            uint32_t firstQuery,
                                                            uint32_t queryCount,
                                                            size_t dataSize,
                                                            void*& pData,
                                                            VkDeviceSize stride,
                                                            VkQueryResultFlags flags,
                                                            decoder* dec);

void _custom_deserialize_vkGetPipelineCacheData_pData(
    state_block*,
    VkDevice device,
    VkPipelineCache pipelineCache,
    size_t* pDataSize,
    void*& pData,
    decoder* dec);

void _custom_deserialize_vkCmdUpdateBuffer_pData(state_block*,
                                                        VkCommandBuffer commandBuffer,
                                                        VkBuffer dstBuffer,
                                                        VkDeviceSize dstOffset,
                                                        VkDeviceSize dataSize,
                                                        void*& pData,
                                                        decoder* dec);

void custom_register_pPhysicalDeviceGroupProperties(VkPhysicalDeviceGroupProperties* props,
                                                           handle_fixer& fix_);

void custom_process_pPhysicalDeviceGroupProperties(VkPhysicalDeviceGroupProperties* props,
                                                          handle_fixer& fix_);

void custom_fix_vkGetQueryPoolResults_pData(state_block* state_block, handle_fixer& fix_, VkDevice device, VkQueryPool queryPool, uint32_t firstQuery, uint32_t queryCount, size_t dataSize, void* pData, VkDeviceSize stride, VkQueryResultFlags flags);
void custom_fix_vkGetPipelineCacheData_pData(state_block* state_block, handle_fixer& fix_, VkDevice device, VkPipelineCache pipelineCache, size_t* pDataSize, void* pData);
void custom_fix_vkCmdUpdateBuffer_pData(state_block* state_block, handle_fixer& fix_, VkCommandBuffer commandBuffer, VkBuffer dstBuffer, VkDeviceSize dstOffset, VkDeviceSize dataSize, const void* pData);
void custom_fix_vkCmdPushConstants_pValues(state_block* state_block, handle_fixer& fix_, VkCommandBuffer commandBuffer, VkPipelineLayout layout, VkShaderStageFlags stageFlags, uint32_t offset, uint32_t size, const void* pValues);
void custom_fix_vkUpdateDescriptorSetWithTemplate_pData(state_block* state_block, handle_fixer& fix_, VkDevice device, VkDescriptorSet descriptorSet, VkDescriptorUpdateTemplate descriptorUpdateTemplate, const void* pData);
void custom_fix_vkGetMemoryHostPointerPropertiesEXT_pHostPointer(state_block* state_block, handle_fixer& fix_, VkDevice device, VkExternalMemoryHandleTypeFlagBits handleType, const void* pHostPointer, VkMemoryHostPointerPropertiesEXT* pMemoryHostPointerProperties);



}  // namespace gapid2