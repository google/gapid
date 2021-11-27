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
#include <fstream>
#include <set>
#include "command_caller.h"
#include "commands.h"
#include "encoder.h"
#include "layerer.h"
#include "memory_tracker.h"
#include "minimal_state_tracker.h"
#include "state_tracker.h"
namespace gapid2 {
class Spy : public StateTracker<gapid2::Layerer<MinimalStateTracker<
                CommandSerializer<CommandCaller<HandleWrapperUpdater>>>>> {
  using super = StateTracker<gapid2::Layerer<MinimalStateTracker<
      CommandSerializer<CommandCaller<HandleWrapperUpdater>>>>>;
  using caller = CommandCaller<HandleWrapperUpdater>;

 public:
  Spy() : out_file("file.trace", std::ios::out | std::ios::binary) {
    encoder_tls_key = TlsAlloc();
    std::vector<std::string> layers = {{"D:\\src\\gapid2\\test3.cpp"}};
    initializeLayers(layers);
  }
  void add_instance(VkInstance instance) {
    std::unique_lock l(map_mutex);
    instances.insert(instance);
  }

  VkResult vkMapMemory(VkDevice device,
                       VkDeviceMemory memory,
                       VkDeviceSize offset,
                       VkDeviceSize size,
                       VkMemoryMapFlags flags,
                       void** ppData) override {
    auto res = super::vkMapMemory(device, memory, offset, size, flags, ppData);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_mem = updater_.cast_from_vk(memory);
    if (size == VK_WHOLE_SIZE) {
      size = new_mem->_size - offset;
    }
    ppData[0] = tracker.AddTrackedRange(memory, ppData[0], offset, size);
    std::unique_lock<std::mutex> l(memory_mutex);
    if (new_mem->_is_coherent) {
      mapped_coherent_memories.insert(memory);
    }
    return res;
  }

  VkResult vkEnumeratePhysicalDevices(
      VkInstance instance,
      uint32_t* pPhysicalDeviceCount,
      VkPhysicalDevice* pPhysicalDevices) override {
    auto ret = super::vkEnumeratePhysicalDevices(instance, pPhysicalDeviceCount,
                                                 pPhysicalDevices);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    if (pPhysicalDevices) {
      auto enc = get_encoder();
      for (size_t i = 0; i < *pPhysicalDeviceCount; ++i) {
        VkPhysicalDeviceProperties properties;
        caller::vkGetPhysicalDeviceProperties(pPhysicalDevices[i], &properties);
        enc->encode<uint32_t>(properties.deviceID);
        enc->encode<uint32_t>(properties.vendorID);
        enc->encode<uint32_t>(properties.driverVersion);
      }
    }
    return ret;
  }

  void vkUnmapMemory(VkDevice device, VkDeviceMemory memory) override {
    tracker.RemoveTrackedRange(memory);
    std::unique_lock<std::mutex> l(memory_mutex);
    mapped_coherent_memories.erase(memory);
    super::vkUnmapMemory(device, memory);
  }

  void vkFreeMemory(VkDevice device,
                    VkDeviceMemory memory,
                    const VkAllocationCallbacks* pAllocator) override {
    auto new_mem = updater_.cast_from_vk(memory);
    if (new_mem->_mapped_location) {
      tracker.RemoveTrackedRange(memory);
    }
    std::unique_lock<std::mutex> l(memory_mutex);
    mapped_coherent_memories.erase(memory);
    super::vkFreeMemory(device, memory, pAllocator);
  }

  VkResult vkFlushMappedMemoryRanges(
      VkDevice device,
      uint32_t memoryRangeCount,
      const VkMappedMemoryRange* pMemoryRanges) override {
    auto res = super::vkFlushMappedMemoryRanges(device, memoryRangeCount,
                                                pMemoryRanges);
    for (uint32_t i = 0; i < memoryRangeCount; ++i) {
      auto& mr = pMemoryRanges[i];
      auto new_mem = updater_.cast_from_vk(mr.memory);
      tracker.for_dirty_in_mem(mr.memory, [this, mr, new_mem](
                                              void* ptr, VkDeviceSize size) {
        auto enc = get_encoder();
        auto offset = reinterpret_cast<char*>(ptr) - new_mem->_mapped_location;
        enc->encode<uint64_t>(0);
        enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(mr.memory));
        enc->encode<uint64_t>(offset);
        enc->encode<uint64_t>(size);
        enc->encode_primitive_array<char>(reinterpret_cast<const char*>(ptr),
                                          size);
      });
    }
    return res;
  }

  VkResult vkInvalidateMappedMemoryRanges(
      VkDevice device,
      uint32_t memoryRangeCount,
      const VkMappedMemoryRange* pMemoryRanges) override {
    auto res = super::vkInvalidateMappedMemoryRanges(device, memoryRangeCount,
                                                     pMemoryRanges);
    for (uint32_t i = 0; i < memoryRangeCount; ++i) {
      auto& mr = pMemoryRanges[i];
      auto new_mem = updater_.cast_from_vk(mr.memory);
      auto sz = mr.size;
      if (sz == VK_WHOLE_SIZE) {
        sz = new_mem->allocate_info->allocationSize - mr.offset;
      }
      tracker.InvalidateMappedRange(mr.memory, mr.offset, sz);
    }
    return res;
  }

  VkResult vkQueueSubmit(VkQueue queue,
                         uint32_t submitCount,
                         const VkSubmitInfo* pSubmits,
                         VkFence fence) override {
    for (size_t i = 0; i < submitCount; ++i) {
      for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
        auto cb = updater_.cast_from_vk(pSubmits[i].pCommandBuffers[j]);
        cb->_pre_run_functions.push_back([this]() {
          std::unique_lock<std::mutex> l(memory_mutex);
          for (auto m : mapped_coherent_memories) {
            auto new_mem = updater_.cast_from_vk(m);
            tracker.for_dirty_in_mem(
                m, [this, m, new_mem](void* ptr, VkDeviceSize size) {
                  auto enc = get_encoder();
                  auto offset =
                      reinterpret_cast<char*>(ptr) - new_mem->_mapped_location;
                  enc->encode<uint64_t>(0);
                  enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(m));
                  enc->encode<uint64_t>(offset);
                  enc->encode<uint64_t>(size);
                  enc->encode_primitive_array<char>(
                      reinterpret_cast<const char*>(ptr), size);
                });
          }
        });
      }
    }
    auto res = super::vkQueueSubmit(queue, submitCount, pSubmits, fence);
    if (res != VK_SUCCESS) {
      return res;
    }

    return res;
  }

  encoder_handle get_encoder() override {
    encoder* enc = reinterpret_cast<encoder*>(TlsGetValue(encoder_tls_key));
    if (!enc) {
      enc = new encoder();
      TlsSetValue(encoder_tls_key, enc);
    }

    call_mutex.lock();
    return encoder_handle(enc, [this, enc]() {
      uint64_t data_size = 0;
      for (size_t i = 0; i <= enc->data_offset; ++i) {
        data_size += enc->data_[i].size - enc->data_[i].left;
      }
      char dat[sizeof(data_size)];
      memcpy(dat, &data_size, sizeof(data_size));
      out_file.write(dat, sizeof(dat));

      for (size_t i = 0; i <= enc->data_offset; ++i) {
        out_file.write(enc->data_[i].data,
                       enc->data_[i].size - enc->data_[i].left);
        enc->data_[i].left = enc->data_[i].size;
      }
      enc->data_offset = 0;
      call_mutex.unlock();
    });
  }

  encoder_handle get_locked_encoder() override {
    encoder* enc = reinterpret_cast<encoder*>(TlsGetValue(encoder_tls_key));
    if (!enc) {
      enc = new encoder();
      TlsSetValue(encoder_tls_key, enc);
    }

    call_mutex.lock();
    return encoder_handle(enc, [this, enc]() {
      uint64_t data_size = 0;
      for (size_t i = 0; i <= enc->data_offset; ++i) {
        data_size += enc->data_[i].size - enc->data_[i].left;
      }
      char dat[sizeof(data_size)];
      memcpy(dat, &data_size, sizeof(data_size));
      out_file.write(dat, sizeof(dat));

      for (size_t i = 0; i <= enc->data_offset; ++i) {
        out_file.write(enc->data_[i].data,
                       enc->data_[i].size - enc->data_[i].left);
        enc->data_[i].left = enc->data_[i].size;
      }
      enc->data_offset = 0;
      call_mutex.unlock();
    });
  }

  VkResult vkDeviceWaitIdle(VkDevice device) override {
    auto res = super::vkDeviceWaitIdle(device);
    if (res == VK_SUCCESS) {
      return res;
    }
    for (auto x : m_pending_write_fences) {
      for (auto d : x.second) {
        auto device_mem = updater_.cast_from_vk(d);
        if (device_mem->_mapped_location && device_mem->_is_coherent) {
          tracker.AddGPUWrite(d, 0, device_mem->_mapped_size);
        }
      }
    }
    return res;
  }

  VkResult vkWaitForFences(VkDevice device,
                           uint32_t fenceCount,
                           const VkFence* pFences,
                           VkBool32 waitAll,
                           uint64_t timeout) override {
    auto res =
        super::vkWaitForFences(device, fenceCount, pFences, waitAll, timeout);
    if (res == VK_TIMEOUT) {
      return res;
    }
    if (fenceCount == 1) {
      return res;
    }
    auto enc = get_encoder();
    for (uint32_t i = 0; i < fenceCount; ++i) {
      if (caller::vkGetFenceStatus(device, pFences[i]) == VK_SUCCESS) {
        enc->encode<char>(1);
        auto it = m_pending_write_fences.find(pFences[i]);
        if (it == m_pending_write_fences.end()) {
          continue;
        }
        for (auto d : it->second) {
          auto device_mem = updater_.cast_from_vk(d);
          if (device_mem->_mapped_location && device_mem->_is_coherent) {
            tracker.AddGPUWrite(d, 0, device_mem->_mapped_size);
          }
        }
        m_pending_write_fences.erase(it);
      } else {
        enc->encode<char>(0);
      }
    }
    return res;
  }

  std::set<VkInstance> instances;
  std::mutex memory_mutex;
  std::set<VkDeviceMemory> mapped_coherent_memories;
  std::mutex map_mutex;
  std::mutex call_mutex;
  temporary_allocator allocator;
  DWORD encoder_tls_key;
  std::fstream out_file;
  memory_tracker tracker;
};  // namespace gapid2
}  // namespace gapid2