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

#include "custom.h"

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <functional>
#include <utility>

#include "decoder.h"
#include "descriptor_update_template.h"
#include "encoder.h"
#include "handle_fixer.h"
#include "state_block.h"
#include "temporary_allocator.h"
#include "utils.h"

namespace gapid2 {

void _custom_clone_VkClearValue(
    state_block*,
    const VkClearValue& src,
    VkClearValue& dst,
    temporary_allocator* mem,
    std::function<bool(const VkClearValue& self)> _VkClearValue_color_valid) {
#pragma FIXME(awoloszyn, Do something with the passed function)
  memcpy(&dst, &src, sizeof(src));
}

void _custom_deserialize_vkCmdPushConstants_pValues(
    state_block*,
    VkCommandBuffer commandBuffer,
    VkPipelineLayout layout,
    VkShaderStageFlags stageFlags,
    uint32_t offset,
    uint32_t size,
    void*& pValues,
    decoder* dec) {
  char* dat = dec->get_typed_memory<char>(size);
  dec->decode_primitive_array(dat, size);
  pValues = reinterpret_cast<void*>(dat);
}

void _custom_clone_VkClearColorValue(state_block*,
                                     const VkClearColorValue& src,
                                     VkClearColorValue& dst,
                                     temporary_allocator* mem) {
  memcpy(&dst, &src, sizeof(src));
}

void _custom_serialize_VkClearColorValue(state_block*,
                                         const VkClearColorValue& value,
                                         encoder* enc) {
  enc->encode<uint32_t>(value.int32[0]);
  enc->encode<uint32_t>(value.int32[1]);
  enc->encode<uint32_t>(value.int32[2]);
  enc->encode<uint32_t>(value.int32[3]);
}

void _custom_serialize_VkClearValue(
    state_block*,
    const VkClearValue& value,
    encoder* enc,
    std::function<bool(const VkClearValue& self)> _VkClearValue_color_valid) {
#pragma FIXME(awoloszyn, Do something with the passed function)
  enc->encode<uint32_t>(value.color.int32[0]);
  enc->encode<uint32_t>(value.color.int32[1]);
  enc->encode<uint32_t>(value.color.int32[2]);
  enc->encode<uint32_t>(value.color.int32[3]);
}

void _custom_deserialize_VkClearColorValue(state_block*,
                                           VkClearColorValue& value,
                                           decoder* dec) {
  dec->decode<uint32_t>(&value.int32[0]);
  dec->decode<uint32_t>(&value.int32[1]);
  dec->decode<uint32_t>(&value.int32[2]);
  dec->decode<uint32_t>(&value.int32[3]);
}

void _custom_deserialize_VkClearValue(state_block*,
                                      VkClearValue& value,
                                      decoder* dec) {
  dec->decode<uint32_t>(&value.color.int32[0]);
  dec->decode<uint32_t>(&value.color.int32[1]);
  dec->decode<uint32_t>(&value.color.int32[2]);
  dec->decode<uint32_t>(&value.color.int32[3]);
}

uint64_t get_VkDescriptorUpdateTemplate_size(
    state_block* state_block_,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate) {
  auto dut = state_block_->get(descriptorUpdateTemplate);
  uint64_t last = 0;
  for (size_t i = 0; i < dut->create_info->descriptorUpdateEntryCount; ++i) {
    auto& j = dut->create_info->pDescriptorUpdateEntries[i];
    if (!j.descriptorCount) {
      continue;
    }
    uint64_t elem_end = 0;
    switch (j.descriptorType) {
      case VK_DESCRIPTOR_TYPE_SAMPLER:
      case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
      case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
      case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
      case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
        elem_end = sizeof(VkDescriptorImageInfo);
        break;
      case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
      case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
      case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
      case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
        elem_end = sizeof(VkDescriptorBufferInfo);
        break;
      case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
      case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
        elem_end = sizeof(VkBufferView);
        break;
      default:
        GAPID2_ERROR("Not implemented yet");
    }
    elem_end += j.offset + (j.descriptorCount - 1) * j.stride;
    if (last < elem_end) {
      last = elem_end;
    }
  }
  return last;
}

void _custom_serialize_vkUpdateDescriptorSetWithTemplate_pData(
    state_block* state_block_,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    const void* pData,
    encoder* enc) {
  uint64_t sz =
      get_VkDescriptorUpdateTemplate_size(state_block_, descriptorUpdateTemplate);
  enc->encode<uint64_t>(sz);
  enc->encode_primitive_array<const char>(reinterpret_cast<const char*>(pData),
                                          sz);
}

const void* _custom_unwrap_vkUpdateDescriptorSetWithTemplate_pData(
    state_block* state_block_,
    temporary_allocator* _allocator,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    const void* pData) {
  auto dut = state_block_->get(descriptorUpdateTemplate);
  uint64_t sz =
      get_VkDescriptorUpdateTemplate_size(state_block_, descriptorUpdateTemplate);
  uint8_t* dst = _allocator->get_typed_memory<uint8_t>(sz);
  memcpy(dst, pData, sz);
  for (size_t i = 0; i < dut->create_info->descriptorUpdateEntryCount; ++i) {
    auto& j = dut->create_info->pDescriptorUpdateEntries[i];
    uint8_t* start = dst + j.offset;
    for (size_t i = 0; i < j.descriptorCount; ++i) {
      switch (j.descriptorType) {
        case VK_DESCRIPTOR_TYPE_SAMPLER:
        case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
        case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
        case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
        case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT: {
          VkDescriptorImageInfo* info =
              reinterpret_cast<VkDescriptorImageInfo*>(start);
          if (info->imageView != nullptr) {
            info->imageView = info->imageView;
          }
          if (info->sampler != nullptr) {
            info->sampler = info->sampler;
          }
          break;
        }
        case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
        case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
        case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
        case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC: {
          VkDescriptorBufferInfo* info =
              reinterpret_cast<VkDescriptorBufferInfo*>(start);
          if (info->buffer != nullptr) {
            info->buffer = info->buffer;
          }
        } break;
        case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
        case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
          VkBufferView* info = reinterpret_cast<VkBufferView*>(start);
          if (*info) {
            *info = *info;
          }
        } break;
        default:
          GAPID2_ERROR("Not implemented yet");
      }
      start += j.stride;
    }
  }
  return dst;
}

void _custom_serialize_vkMapMemory_ppData(state_block*,
                                          VkDevice device,
                                          VkDeviceMemory memory,
                                          VkDeviceSize offset,
                                          VkDeviceSize size,
                                          VkMemoryMapFlags flags,
                                          void** ppData,
                                          encoder* enc) {
  enc->encode<uint64_t>(
      static_cast<uint64_t>(reinterpret_cast<uintptr_t>(ppData[0])));
}

void _custom_serialize_vkGetMemoryHostPointerPropertiesEXT_pHostPointer(
    state_block* state_block_, VkDevice device, VkExternalMemoryHandleTypeFlagBits handleType, const void* pHostPointer, VkMemoryHostPointerPropertiesEXT* pMemoryHostPointerProperties, encoder* enc) {
  GAPID2_ERROR("Unimplemented: _custom_serialize_vkGetMemoryHostPointerPropertiesEXT_pHostPointer");
}

void _custom_deserialize_vkGetMemoryHostPointerPropertiesEXT_pHostPointer(
    state_block* state_block_, VkDevice device, VkExternalMemoryHandleTypeFlagBits handleType, const void* pHostPointer, VkMemoryHostPointerPropertiesEXT* pMemoryHostPointerProperties, decoder* dec) {
  GAPID2_ERROR("Unimplemented: _custom_deserialize_vkGetMemoryHostPointerPropertiesEXT_pHostPointer");
}

void _custom_serialize_vkGetQueryPoolResults_pData(state_block*,
                                                   VkDevice device,
                                                   VkQueryPool queryPool,
                                                   uint32_t firstQuery,
                                                   uint32_t queryCount,
                                                   size_t dataSize,
                                                   void* pData,
                                                   VkDeviceSize stride,
                                                   VkQueryResultFlags flags,
                                                   encoder* enc) {
  GAPID2_ERROR("Unimplemented: _custom_serialize_vkGetQueryPoolResults_pData");
}

void _custom_serialize_vkGetPipelineCacheData_pData(
    state_block*,
    VkDevice device,
    VkPipelineCache pipelineCache,
    size_t* pDataSize,
    void* pData,
    encoder* enc) {
  GAPID2_ERROR("Unimplemented: _custom_serialize_vkGetPipelineCacheData_pData");
}

void _custom_serialize_vkCmdUpdateBuffer_pData(state_block*,
                                               VkCommandBuffer commandBuffer,
                                               VkBuffer dstBuffer,
                                               VkDeviceSize dstOffset,
                                               VkDeviceSize dataSize,
                                               const void* pData,
                                               encoder* enc) {
  enc->encode_primitive_array<const char>(reinterpret_cast<const char*>(pData),
                                          dataSize);
}

void _custom_serialize_vkCmdPushConstants_pValues(state_block*,
                                                  VkCommandBuffer commandBuffer,
                                                  VkPipelineLayout layout,
                                                  VkShaderStageFlags stageFlags,
                                                  uint32_t offset,
                                                  uint32_t size,
                                                  const void* pValues,
                                                  encoder* enc) {
  enc->encode_primitive_array<const char>(
      reinterpret_cast<const char*>(pValues), size);
}

void _custom_deserialize_vkUpdateDescriptorSetWithTemplate_pData(
    state_block*,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    void*& pData,
    decoder* dec) {
  uint64_t data_size = dec->decode<uint64_t>();
  pData = dec->get_typed_memory<char>(data_size);
  dec->decode_primitive_array<char>(reinterpret_cast<char*>(pData), data_size);
}

void _custom_deserialize_vkMapMemory_ppData(state_block*,
                                            VkDevice device,
                                            VkDeviceMemory memory,
                                            VkDeviceSize offset,
                                            VkDeviceSize size,
                                            VkMemoryMapFlags flags,
                                            void**& ppData,
                                            decoder* dec) {
  ppData = dec->get_typed_memory<void*>(1);
  ppData[0] = reinterpret_cast<void*>(
      static_cast<uintptr_t>(static_cast<uint64_t>(dec->decode<uint64_t>())));
}

void _custom_deserialize_vkGetQueryPoolResults_pData(state_block*,
                                                     VkDevice device,
                                                     VkQueryPool queryPool,
                                                     uint32_t firstQuery,
                                                     uint32_t queryCount,
                                                     size_t dataSize,
                                                     void*& pData,
                                                     VkDeviceSize stride,
                                                     VkQueryResultFlags flags,
                                                     decoder* dec) {
  GAPID2_ERROR(
      "Unimplemented: _custom_deserialize_vkGetQueryPoolResults_pData");
}

void _custom_deserialize_vkGetPipelineCacheData_pData(
    state_block*,
    VkDevice device,
    VkPipelineCache pipelineCache,
    size_t* pDataSize,
    void*& pData,
    decoder* dec) {
  GAPID2_ERROR(
      "Unimplemented: _custom_deserialize_vkGetPipelineCacheData_pData");
}

void _custom_deserialize_vkCmdUpdateBuffer_pData(state_block*,
                                                 VkCommandBuffer commandBuffer,
                                                 VkBuffer dstBuffer,
                                                 VkDeviceSize dstOffset,
                                                 VkDeviceSize dataSize,
                                                 void*& pData,
                                                 decoder* dec) {
  char* dat = dec->get_typed_memory<char>(dataSize);
  dec->decode_primitive_array(dat, dataSize);
  pData = reinterpret_cast<void*>(dat);
}

void custom_register_pPhysicalDeviceGroupProperties(VkPhysicalDeviceGroupProperties* props,
                                                    handle_fixer& fix_) {
  for (size_t i = 0; i < VK_MAX_DEVICE_GROUP_SIZE; ++i) {
    fix_.register_handle(&props->physicalDevices[i]);
  }
}

void custom_process_pPhysicalDeviceGroupProperties(VkPhysicalDeviceGroupProperties* props,
                                                   handle_fixer& fix_) {
  for (size_t i = 0; i < props->physicalDeviceCount; ++i) {
    fix_.process_handle(&props->physicalDevices[i]);
  }
  for (size_t i = props->physicalDeviceCount; i < VK_MAX_DEVICE_GROUP_SIZE; ++i) {
    fix_.VkPhysicalDevice_registered_handles_.erase(&props->physicalDevices[i]);
  }
}

void custom_generate_pPhysicalDeviceGroupProperties(state_block* state_block, VkPhysicalDeviceGroupProperties* props) {
  for (size_t i = 0; i < VK_MAX_DEVICE_GROUP_SIZE; ++i) {
    if (props->physicalDevices[i] == VK_NULL_HANDLE) {
      props->physicalDevices[i] = state_block->get_unused_VkPhysicalDevice();
    }
  }
}

void custom_fix_vkGetQueryPoolResults_pData(state_block*, handle_fixer&, VkDevice, VkQueryPool, uint32_t, uint32_t, size_t, void*, VkDeviceSize, VkQueryResultFlags) {
}
void custom_fix_vkGetPipelineCacheData_pData(state_block*, handle_fixer&, VkDevice, VkPipelineCache, size_t*, void*) {
}
void custom_fix_vkCmdUpdateBuffer_pData(state_block*, handle_fixer&, VkCommandBuffer, VkBuffer, VkDeviceSize, VkDeviceSize, const void*) {
}
void custom_fix_vkCmdPushConstants_pValues(state_block*, handle_fixer&, VkCommandBuffer, VkPipelineLayout, VkShaderStageFlags, uint32_t, uint32_t, const void*) {
}
void custom_fix_vkGetMemoryHostPointerPropertiesEXT_pHostPointer(state_block*, handle_fixer&, VkDevice, VkExternalMemoryHandleTypeFlagBits, const void*, VkMemoryHostPointerPropertiesEXT*) {
}

void custom_fix_vkUpdateDescriptorSetWithTemplate_pData(state_block* state_block, handle_fixer& fix_, VkDevice device, VkDescriptorSet descriptorSet, VkDescriptorUpdateTemplate descriptorUpdateTemplate, const void* pData) {
  auto dut = state_block->get(descriptorUpdateTemplate);
  const uint8_t* dst = reinterpret_cast<const uint8_t*>(pData);
  for (size_t i = 0; i < dut->create_info->descriptorUpdateEntryCount; ++i) {
    auto& j = dut->create_info->pDescriptorUpdateEntries[i];
    const uint8_t* start = dst + j.offset;
    for (size_t i = 0; i < j.descriptorCount; ++i) {
      switch (j.descriptorType) {
        case VK_DESCRIPTOR_TYPE_SAMPLER:
        case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
        case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
        case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
        case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT: {
          const VkDescriptorImageInfo* info =
              reinterpret_cast<const VkDescriptorImageInfo*>(start);
          if (info->imageView != nullptr) {
            fix_.fix_handle(&info->imageView);
          }
          if (info->sampler != nullptr) {
            fix_.fix_handle(&info->sampler);
          }
          break;
        }
        case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
        case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
        case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
        case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC: {
          const VkDescriptorBufferInfo* info =
              reinterpret_cast<const VkDescriptorBufferInfo*>(start);
          if (info->buffer != nullptr) {
            fix_.fix_handle(&info->buffer);
          }
        } break;
        case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
        case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
          const VkBufferView* info = reinterpret_cast<const VkBufferView*>(start);
          fix_.fix_handle(info);
        } break;
        default:
          GAPID2_ERROR("Not implemented yet");
      }
      start += j.stride;
    }
  }
}
}  // namespace gapid2