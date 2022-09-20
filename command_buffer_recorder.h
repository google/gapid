#pragma once

#include <vulkan/vulkan.h>

#include <shared_mutex>
#include <unordered_map>
#include <unordered_set>

#include "command_buffer_deserializer.h"
#include "decoder.h"
#include "encoder.h"
#include "indirect_functions.h"
#include "transform_base.h"

namespace gapid2 {
class command_buffer_recorder : public transform_base {
 public:
  void RerecordCommandBuffer(VkCommandBuffer cb, transform_base* next,
                             std::function<void(uint64_t)> notify_pre_command = nullptr) {
    std::shared_lock<std::shared_mutex> l(command_buffers_mutex);
    std::unordered_map<VkCommandBuffer,
                       std::unique_ptr<command_buffer_recording>>::iterator it;
    it = cbrs.find(reinterpret_cast<VkCommandBuffer>(cb));
    if (it == cbrs.end()) {
      GAPID2_ERROR("Trying to rerecord an untracked command buffer");
      return;
    }
    // Clone the contents in case we want to re-record AGAIN later.
    std::vector v = clone_blocks(it->second->enc.data_);
    decoder dec(std::move(v));

    command_buffer_deserializer deserializer;
    deserializer.next = next;
    deserializer.notify_pre_command_fn = notify_pre_command;
    deserializer.DeserializeStream(&dec, true);
  }

  VkResult vkAllocateCommandBuffers(
      VkDevice device,
      const VkCommandBufferAllocateInfo* pAllocateInfo,
      VkCommandBuffer* pCommandBuffers) {
    // Bypass serialization for vkAllocate*
    auto ret = transform_base::vkAllocateCommandBuffers(device, pAllocateInfo,
                                                        pCommandBuffers);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    std::unique_lock<std::shared_mutex> l(command_buffers_mutex);
    if (command_buffers_to_track.empty()) {
      for (size_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
        cbrs.insert(std::make_pair(
            pCommandBuffers[i], std::make_unique<command_buffer_recording>(
                                    pAllocateInfo->commandPool)));
      }
    } else {
      for (size_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
        if (command_buffers_to_track.count(pCommandBuffers[i])) {
          cbrs.insert(
              std::make_pair(pCommandBuffers[i],
                             std::make_unique<command_buffer_recording>(
                                 pAllocateInfo->commandPool)));
        }
      }
    }
    return ret;
  }

  void vkFreeCommandBuffers(VkDevice device,
                            VkCommandPool commandPool,
                            uint32_t commandBufferCount,
                            const VkCommandBuffer* pCommandBuffers) {
    std::unique_lock<std::shared_mutex> l(command_buffers_mutex);
    for (size_t i = 0; i < commandBufferCount; ++i) {
      auto it = cbrs.find(pCommandBuffers[i]);
      if (it != cbrs.end()) {
        cbrs.erase(it);
      }
    }

    // Bypass serialization for vkFree*
    transform_base::vkFreeCommandBuffers(device, commandPool, commandBufferCount,
                                         pCommandBuffers);
  }

  VkResult vkResetCommandPool(VkDevice device, VkCommandPool commandPool, VkCommandPoolResetFlags flags) {
    auto ret = transform_base::vkResetCommandPool(device, commandPool, flags);
    command_buffers_mutex.lock();
    for (auto& it : cbrs) {
      if (it.second->pool == commandPool) {
        it.second->enc.reset();
      }
    }
    command_buffers_mutex.unlock();
    return ret;
  }

  void do_begin_command_buffer(VkCommandBuffer commandBuffer) {
    std::unordered_map<VkCommandBuffer,
                       std::unique_ptr<command_buffer_recording>>::iterator it;
    {
      std::shared_lock<std::shared_mutex> l(command_buffers_mutex);
      it = cbrs.find(reinterpret_cast<VkCommandBuffer>(commandBuffer));
      if (it != cbrs.end()) {
        it->second->enc.reset();
      }
    }
  }

  struct command_buffer_recording {
    explicit command_buffer_recording(VkCommandPool _pool) : pool(_pool){};
    VkCommandPool pool;
    encoder enc;
  };

  std::shared_mutex command_buffers_mutex;
  std::unordered_set<VkCommandBuffer> command_buffers_to_track;
  std::unordered_map<VkCommandBuffer, std::unique_ptr<command_buffer_recording>>
      cbrs;

  virtual encoder_handle get_encoder(uintptr_t key) {
    std::unordered_map<VkCommandBuffer,
                       std::unique_ptr<command_buffer_recording>>::iterator it;
    {
      std::shared_lock<std::shared_mutex> l(command_buffers_mutex);
      it = cbrs.find(reinterpret_cast<VkCommandBuffer>(key));
      if (it == cbrs.end()) {
        return encoder_handle(nullptr);
      }
    }

    return encoder_handle(&it->second->enc);
  }

  virtual encoder_handle get_locked_encoder(uintptr_t key) {
    // We don't need a locked encoder for any command buffers.
    return get_encoder(key);
  }

  uint64_t get_flags() const {
    return 0;
  }

#include "command_buffer_recorder.inl"
};

}  // namespace gapid2
