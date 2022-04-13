#pragma once

#include <vulkan/vulkan.h>
#include <shared_mutex>
#include <unordered_map>
#include <unordered_set>
#include "command_deserializer.h"
#include "commands.h"
#include "decoder.h"
#include "encoder.h"
#include "fn_caller.h"
#include "layerer.h"

namespace gapid2 {
template <typename T>
class CommandBufferRecorder : public CommandSerializer<true, FnCaller<T>> {
 public:
  void SetupCommandBufferRecorderFunctions(fns& f);
  CommandBufferRecorder(
      fns& f_) {  // Make a copy of the function calls at this point.
                  // Once we have a copy, we can modify the calls to call us for
                  // any functions we might care about.
    f = f_;
    SetupCommandBufferRecorderFunctions(f_);
  }

  void RerecordCommandBuffer(VkCommandBuffer cb) {
    std::shared_lock<std::shared_mutex> l(command_buffers_mutex);
    std::unordered_map<VkCommandBuffer,
                       std::unique_ptr<command_buffer_recording>>::iterator it;
    it = cbrs.find(reinterpret_cast<VkCommandBuffer>(cb));
    if (it == cbrs.end()) {
      OutputDebugStringA("Trying to rerecord an untracked command buffer");
      return;
    }
    // Clone the contents in case we want to re-record AGAIN later.
    std::vector v = it->second->enc.data_;
    decoder dec(std::move(v));
    CommandDeserializer<FnCaller<T>> deserializer;
    deserializer.f = f;
    deserializer.updater = updater;
    deserializer.DeserializeStream(&dec, true);
  }

  VkResult vkAllocateCommandBuffers(
      VkDevice device,
      const VkCommandBufferAllocateInfo* pAllocateInfo,
      VkCommandBuffer* pCommandBuffers) {
    // Bypass serialization for vkAllocate*
    auto ret = FnCaller::vkAllocateCommandBuffers(device, pAllocateInfo,
                                                  pCommandBuffers);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    std::unique_lock<std::shared_mutex> l(command_buffers_mutex);
    if (command_buffers_to_track.empty()) {
      for (size_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
        cbrs.insert(std::make_pair(
            pCommandBuffers[i], std::make_unique<command_buffer_recording>()));
      }
    } else {
      for (size_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
        if (command_buffers_to_track.count(pCommandBuffers[i])) {
          cbrs.insert(
              std::make_pair(pCommandBuffers[i],
                             std::make_unique<command_buffer_recording>()));
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
    FnCaller::vkFreeCommandBuffers(device, commandPool, commandBufferCount,
                                   pCommandBuffers);
  }

  VkResult vkBeginCommandBuffer(
      VkCommandBuffer commandBuffer,
      const VkCommandBufferBeginInfo* pBeginInfo) override {
    std::unordered_map<VkCommandBuffer,
                       std::unique_ptr<command_buffer_recording>>::iterator it;
    {
      std::shared_lock<std::shared_mutex> l(command_buffers_mutex);
      it = cbrs.find(reinterpret_cast<VkCommandBuffer>(commandBuffer));
      if (it != cbrs.end()) {
        it->second->enc.reset();
      }
    }

    return CommandSerializer<true, FnCaller<T>>::vkBeginCommandBuffer(
        commandBuffer, pBeginInfo);
  }

  struct command_buffer_recording {
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
};

}  // namespace gapid2

#include "command_buffer_recorder.inl"