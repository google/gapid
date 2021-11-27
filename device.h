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

#include <handles.h>
#include <vulkan.h>
#include <memory>
#include <mutex>
#include "device_functions.h"
#include "null_cloner.h"
#include "physical_device.h"

#define REGISTER_CHILD_TYPE(type)                                   \
 public:                                                            \
  type get_and_increment_child(type t) {                            \
    std::unique_lock l(child_mutex);                                \
    auto it = __##type##s.find(t);                                  \
    if (it == __##type##s.end()) {                                  \
      return VK_NULL_HANDLE;                                        \
    }                                                               \
    it->second.second++;                                            \
    return it->second.first;                                        \
  }                                                                 \
  void add_child(type t, void* _t) {                                \
    std::unique_lock l(child_mutex);                                \
    __##type##s[t] = std::make_pair(reinterpret_cast<type>(_t), 1); \
  }                                                                 \
                                                                    \
 private:                                                           \
  std::unordered_map<type, std::pair<type, uint32_t>> __##type##s;

namespace gapid2 {
template <typename HandleUpdater>
struct VkDeviceWrapper : handle_base<VkDevice, void> {
  VkDeviceWrapper(HandleUpdater*, VkPhysicalDevice phys_dev, VkDevice device)
      : handle_base<VkDevice, void>(device) {}
  void set_device_loader_data(PFN_vkSetDeviceLoaderData data) {
    vkSetDeviceLoaderData = data;
    vkSetDeviceLoaderData(_handle, this);
  }

  void set_create_info(const VkDeviceCreateInfo* pCreateInfo) {
    create_info = mem.get_typed_memory<VkDeviceCreateInfo>(1);
    clone<NullCloner>(&cloner, pCreateInfo[0], create_info[0], &mem);
  }

  PFN_vkSetDeviceLoaderData vkSetDeviceLoaderData;
  std::unique_ptr<DeviceFunctions> _functions;
  REGISTER_CHILD_TYPE(VkCommandBuffer);
  REGISTER_CHILD_TYPE(VkCommandPool);
  REGISTER_CHILD_TYPE(VkBufferView);
  REGISTER_CHILD_TYPE(VkImageView);
  REGISTER_CHILD_TYPE(VkImage);
  REGISTER_CHILD_TYPE(VkBuffer);
  REGISTER_CHILD_TYPE(VkDescriptorPool);
  REGISTER_CHILD_TYPE(VkDescriptorSet);
  REGISTER_CHILD_TYPE(VkDescriptorSetLayout);
  REGISTER_CHILD_TYPE(VkDescriptorUpdateTemplate);
  REGISTER_CHILD_TYPE(VkDeviceMemory);
  REGISTER_CHILD_TYPE(VkEvent);
  REGISTER_CHILD_TYPE(VkFence);
  REGISTER_CHILD_TYPE(VkFramebuffer);
  REGISTER_CHILD_TYPE(VkPipeline);
  REGISTER_CHILD_TYPE(VkPipelineCache);
  REGISTER_CHILD_TYPE(VkPipelineLayout);
  REGISTER_CHILD_TYPE(VkQueryPool);
  REGISTER_CHILD_TYPE(VkQueue);
  REGISTER_CHILD_TYPE(VkRenderPass);
  REGISTER_CHILD_TYPE(VkSamplerYcbcrConversion);
  REGISTER_CHILD_TYPE(VkSampler);
  REGISTER_CHILD_TYPE(VkSwapchainKHR);
  REGISTER_CHILD_TYPE(VkSemaphore);
  REGISTER_CHILD_TYPE(VkShaderModule);

 private:
  std::mutex child_mutex;
  VkDeviceCreateInfo* create_info = nullptr;
  NullCloner cloner;
  temporary_allocator mem;
};
}  // namespace gapid2

#undef REGISTER_CHILD_TYPE