/*
 * Copyright (C) 2018 Google Inc.
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

#ifndef VK_BASE_SWAPCHAIN_VIRTUAL_SWAPCHAIN_H_
#define VK_BASE_SWAPCHAIN_VIRTUAL_SWAPCHAIN_H_

#include <vulkan/vulkan.h>
#include <mutex>
#include <vector>
#include "layer.h"

namespace swapchain {
// The BaseSwapchain handles blitting presenting images to the original surface
class BaseSwapchain {
 public:
  BaseSwapchain(VkInstance instance, VkDevice device, uint32_t queue,
                VkCommandPool command_pool, uint32_t num_images,
                const InstanceData* instance_functions,
                const DeviceData* device_functions,
                const VkSwapchainCreateInfoKHR* swapchain_info,
                const VkAllocationCallbacks* pAllocator,
                const void* platform_info);
  void Destroy(const VkAllocationCallbacks* pAllocator);
  bool Valid() const;

  VkResult PresentFrom(VkQueue queue, size_t index, VkImage image);
  VkSemaphore BlitWaitSemaphore(size_t index);

 private:
  VkInstance instance_;
  VkDevice device_;
  const InstanceData* instance_functions_;
  const DeviceData* device_functions_;
  VkSwapchainCreateInfoKHR swapchain_info_;

  threading::mutex present_lock_;

  // The real surface to present to.
  VkSurfaceKHR surface_;
  // The real swapchain to present to.
  VkSwapchainKHR swapchain_;
  // The swapchain images to blit to.
  std::vector<VkImage> images_;
  // The semaphores to signal when the acquire is completed.
  VkSemaphore acquire_semaphore_;
  // The semaphores to signal when the blit is completed.
  std::vector<VkSemaphore> blit_semaphores_;
  // The semaphores that will be signaled when the blit is complete and will be
  // waited on by the queue present
  std::vector<VkSemaphore> present_semaphores_;
  // Whether the semaphores at a given index are currently pending and should
  // have their blit semaphore waited on before acquiring.
  std::vector<bool> is_pending_;
  // The command buffers to use to blit.  We need several in case someone
  // submits while a previous one is still pending.
  std::vector<VkCommandBuffer> command_buffers_;
  // Whether we completed construction successfully
  bool valid_;
};

}  // namespace swapchain

#endif  // VK_BASE_SWAPCHAIN_VIRTUAL_SWAPCHAIN_H_
