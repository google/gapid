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
#include <vulkan.h>
#include <cassert>
#include "handles.h"
#include "temporary_allocator.h"
#define _QUOTE(x) #x
#define QUOTE(x) _QUOTE(x)
#define __FILE__LINE__ __FILE__ ":" QUOTE(__LINE__) " : "

#define FIXME(user, ...)                                        \
  message(__FILE__LINE__                                        \
          "\n"                                                  \
          " ------------------------------------------------\n" \
          "|  FIXME(" #user ") :  " #__VA_ARGS__                \
          "\n"                                                  \
          " -------------------------------------------------\n")

#include "buffer.h"
#include "buffer_view.h"
#include "command_buffer.h"
#include "command_pool.h"
#include "common.h"
#include "descriptor_pool.h"
#include "descriptor_set.h"
#include "descriptor_set_layout.h"
#include "descriptor_update_template.h"
#include "device.h"
#include "device_memory.h"
#include "event.h"
#include "fence.h"
#include "framebuffer.h"
#include "handles.h"
#include "image.h"
#include "image_view.h"
#include "instance.h"
#include "physical_device.h"
#include "pipeline.h"
#include "pipeline_cache.h"
#include "pipeline_layout.h"
#include "query_pool.h"
#include "queue.h"
#include "render_pass.h"
#include "sampler.h"
#include "sampler_ycbcr_conversion.h"
#include "semaphore.h"
#include "shader_module.h"
#include "surface.h"
#include "swapchain.h"

namespace gapid2 {

#define PROCESS_HANDLE(Type)                   \
  template <typename HandleUpdater>            \
  struct handle_type<HandleUpdater, Type> {    \
    using type = Type##Wrapper<HandleUpdater>; \
    const bool is_handle = true;               \
  };
#include "handle_defines.inl"
#undef PROCESS_HANDLE

template <typename F, typename T>
auto bind_first(F&& f, T&& t) {
  return [f = std::forward<F>(f), t = std::forward<T>(t)](auto&&... args) {
    return f(t, std::forward<decltype(args)>(args)...);
  };
}

template <typename HandleUpdater, typename T, typename... Args>
T* clone_struct(HandleUpdater* _updater,
                const T* t,
                const size_t _num,
                temporary_allocator* mem,
                Args... args) {
  if (!t || !_num) {
    return nullptr;
  }
  T* nt = mem->get_typed_memory<T>(_num);
  for (size_t i = 0; i < _num; ++i) {
    clone(_updater, t[i], nt[i], mem, std::forward<Args>(args)...);
  }
  return nt;
}
template <typename HandleUpdater, typename T, typename... Args>
T* clone_struct(HandleUpdater* _updater,
                const T* t,
                uint32_t* _num,
                temporary_allocator* mem,
                Args... args) {
  return clone_struct(_updater, t, static_cast<size_t>(*_num), mem,
                      std::forward<Args>(args)...);
}
template <typename HandleUpdater, typename T, typename... Args>
T* clone_handle(HandleUpdater* _updater,
                const T* t,
                const size_t _num,
                temporary_allocator* mem) {
  T* nt = mem->get_typed_memory<T>(_num);

  for (size_t i = 0; i < _num; ++i) {
    nt[i] = _updater->cast_in(t[i]);
  }
  return nt;
}

template <typename HandleUpdater, typename T, typename... Args>
T* clone_handle(HandleUpdater* _updater,
                const T* t,
                const uint32_t* _num,
                temporary_allocator* mem) {
  return clone_handle(_updater, t, static_cast<size_t>(*_num), mem);
}

template <typename HandleUpdater, typename P, typename T, typename RT>
void create_handle(HandleUpdater* _updater, P p, T* t, size_t num) {
  if (!t) {
    return;
  }
  auto p_ptr = _updater->cast_from_vk(p);

  for (size_t i = 0; i < num; ++i) {
    _updater->fixup_dispatch<P, T>(p, t[i]);
    auto ti = reinterpret_cast<typename handle_type<HandleUpdater, T>::type*>(
        p_ptr->get_and_increment_child(t[i]));
    if (ti) {
      t[i] = _updater->template cast_out<T>(ti);
      continue;
    }
    auto ni = new RT(_updater, p, t[i]);
    p_ptr->add_child(t[i], ni);
    t[i] = _updater->template cast_out<T>(ni);
  }
}

template <typename HandleUpdater, typename P, typename T, typename RT>
void create_handle(HandleUpdater* _updater, P p, T* t, uint32_t* num) {
  if (!t) {
    return;
  }
  return create_handle<HandleUpdater, P, T, RT>(_updater, p, t,
                                                static_cast<size_t>(*num));
}

template <typename HandleUpdater, typename RT>
void create_instance(HandleUpdater* _updater, VkInstance* i) {
  auto ni = new RT(*i);
  *i = _updater->template cast_out<VkInstance>(ni);
}

template <typename HandleUpdater>
inline void create_handle_from_struct(
    HandleUpdater* _updater,
    VkInstance instance,
    VkPhysicalDeviceGroupProperties* pPhysicalDeviceGroupProperties,
    uint32_t* pPhysicalDeviceGroupCount) {
  if (pPhysicalDeviceGroupProperties) {
    for (uint32_t i = 0; i < *pPhysicalDeviceGroupCount; ++i) {
      create_handle<HandleUpdater, VkInstance, VkPhysicalDevice,
                    VkPhysicalDeviceWrapper<HandleUpdater>>(
          _updater, instance, pPhysicalDeviceGroupProperties[i].physicalDevices,
          pPhysicalDeviceGroupProperties[i].physicalDeviceCount);
    }
  }
}
class encoder;
class decoder;
template <typename HandleUpdater>
void _custom_serialize_VkClearColorValue(HandleUpdater*,
                                         const VkClearColorValue& value,
                                         encoder* enc) {
  enc->encode<uint32_t>(value.int32[0]);
  enc->encode<uint32_t>(value.int32[1]);
  enc->encode<uint32_t>(value.int32[2]);
  enc->encode<uint32_t>(value.int32[3]);
}
template <typename HandleUpdater>
void _custom_serialize_VkClearValue(HandleUpdater*,
                                    const VkClearValue& value,
                                    encoder* enc) {
  enc->encode<uint32_t>(value.color.int32[0]);
  enc->encode<uint32_t>(value.color.int32[1]);
  enc->encode<uint32_t>(value.color.int32[2]);
  enc->encode<uint32_t>(value.color.int32[3]);
}

template <typename HandleUpdater>
void _custom_deserialize_VkClearColorValue(HandleUpdater*,
                                           VkClearColorValue& value,
                                           decoder* dec) {
  dec->decode<uint32_t>(&value.int32[0]);
  dec->decode<uint32_t>(&value.int32[1]);
  dec->decode<uint32_t>(&value.int32[2]);
  dec->decode<uint32_t>(&value.int32[3]);
}

template <typename HandleUpdater>
void _custom_deserialize_VkClearValue(HandleUpdater*,
                                      VkClearValue& value,
                                      decoder* dec) {
  dec->decode<uint32_t>(&value.color.int32[0]);
  dec->decode<uint32_t>(&value.color.int32[1]);
  dec->decode<uint32_t>(&value.color.int32[2]);
  dec->decode<uint32_t>(&value.color.int32[3]);
}

template <typename HandleUpdater>
uint64_t get_VkDescriptorUpdateTemplate_size(
    HandleUpdater* updater,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate) {
  auto dut = updater->cast_from_vk(descriptorUpdateTemplate);
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

template <typename HandleUpdater>
void _custom_serialize_vkUpdateDescriptorSetWithTemplate_pData(
    HandleUpdater* updater,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    const void* pData,
    encoder* enc) {
  uint64_t sz =
      get_VkDescriptorUpdateTemplate_size(updater, descriptorUpdateTemplate);
  enc->encode<uint64_t>(sz);
  enc->encode_primitive_array<const char>(reinterpret_cast<const char*>(pData),
                                          sz);
}

template <typename HandleUpdater>
const void* _custom_unwrap_vkUpdateDescriptorSetWithTemplate_pData(
    HandleUpdater* updater,
    temporary_allocator* _allocator,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    const void* pData) {
  auto dut = updater->cast_from_vk(descriptorUpdateTemplate);
  uint64_t sz =
      get_VkDescriptorUpdateTemplate_size(updater, descriptorUpdateTemplate);
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
            info->imageView = updater->cast_in(info->imageView);
          }
          if (info->sampler != nullptr) {
            info->sampler = updater->cast_in(info->sampler);
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
            info->buffer = updater->cast_in(info->buffer);
          }
        } break;
        case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
        case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
          VkBufferView* info = reinterpret_cast<VkBufferView*>(start);
          if (*info) {
            *info = updater->cast_in(*info);
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

template <typename HandleUpdater>
void _custom_serialize_vkMapMemory_ppData(HandleUpdater*,
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

template <typename HandleUpdater>
void _custom_serialize_vkGetQueryPoolResults_pData(HandleUpdater*,
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

template <typename HandleUpdater>
void _custom_serialize_vkGetPipelineCacheData_pData(
    HandleUpdater*,
    VkDevice device,
    VkPipelineCache pipelineCache,
    size_t* pDataSize,
    void* pData,
    encoder* enc) {
  GAPID2_ERROR("Unimplemented: _custom_serialize_vkGetPipelineCacheData_pData");
}

template <typename HandleUpdater>
void _custom_serialize_vkCmdUpdateBuffer_pData(HandleUpdater*,
                                               VkCommandBuffer commandBuffer,
                                               VkBuffer dstBuffer,
                                               VkDeviceSize dstOffset,
                                               VkDeviceSize dataSize,
                                               const void* pData,
                                               encoder* enc) {
  enc->encode_primitive_array<const char>(reinterpret_cast<const char*>(pData),
                                          dataSize);
}

template <typename HandleUpdater>
void _custom_serialize_vkCmdPushConstants_pValues(HandleUpdater*,
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

template <typename HandleUpdater>
void _custom_deserialize_vkUpdateDescriptorSetWithTemplate_pData(
    HandleUpdater*,
    VkDevice device,
    VkDescriptorSet descriptorSet,
    VkDescriptorUpdateTemplate descriptorUpdateTemplate,
    void*& pData,
    decoder* dec) {
  uint64_t data_size = dec->decode<uint64_t>();
  pData = dec->get_typed_memory<char>(data_size);
  dec->decode_primitive_array<char>(reinterpret_cast<char*>(pData), data_size);
}

template <typename HandleUpdater>
void _custom_deserialize_vkMapMemory_ppData(HandleUpdater*,
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

template <typename HandleUpdater>
void _custom_deserialize_vkGetQueryPoolResults_pData(HandleUpdater*,
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
template <typename HandleUpdater>
void _custom_deserialize_vkGetPipelineCacheData_pData(
    HandleUpdater*,
    VkDevice device,
    VkPipelineCache pipelineCache,
    size_t* pDataSize,
    void*& pData,
    decoder* dec) {
  GAPID2_ERROR(
      "Unimplemented: _custom_deserialize_vkGetPipelineCacheData_pData");
}

template <typename HandleUpdater>
void _custom_deserialize_vkCmdUpdateBuffer_pData(HandleUpdater*,
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

template <typename HandleUpdater>
void _custom_deserialize_vkCmdPushConstants_pValues(
    HandleUpdater*,
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

template <typename HandleUpdater>
void _custom_clone_VkClearValue(HandleUpdater*,
                                const VkClearValue& src,
                                VkClearValue& dst,
                                temporary_allocator* mem) {
  memcpy(&dst, &src, sizeof(src));
}

template <typename HandleUpdater>
void _custom_clone_VkClearColorValue(HandleUpdater*,
                                     const VkClearColorValue& src,
                                     VkClearColorValue& dst,
                                     temporary_allocator* mem) {
  memcpy(&dst, &src, sizeof(src));
}

}  // namespace gapid2