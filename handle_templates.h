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

template <typename T>
struct needs_dispatch_fixup {};

#define NO_HANDLE_FIXUP(x)         \
  template <>                      \
  struct needs_dispatch_fixup<x> { \
    static const bool val = false; \
  }

NO_HANDLE_FIXUP(VkInstance);
NO_HANDLE_FIXUP(VkDevice);
NO_HANDLE_FIXUP(VkDeviceMemory);
NO_HANDLE_FIXUP(VkFence);
NO_HANDLE_FIXUP(VkSemaphore);
NO_HANDLE_FIXUP(VkEvent);
NO_HANDLE_FIXUP(VkQueryPool);
NO_HANDLE_FIXUP(VkBuffer);
NO_HANDLE_FIXUP(VkBufferView);
NO_HANDLE_FIXUP(VkImage);
NO_HANDLE_FIXUP(VkImageView);
NO_HANDLE_FIXUP(VkShaderModule);
NO_HANDLE_FIXUP(VkPipelineCache);
NO_HANDLE_FIXUP(VkPipeline);
NO_HANDLE_FIXUP(VkPipelineLayout);
NO_HANDLE_FIXUP(VkSampler);
NO_HANDLE_FIXUP(VkDescriptorPool);
NO_HANDLE_FIXUP(VkDescriptorSet);
NO_HANDLE_FIXUP(VkDescriptorSetLayout);
NO_HANDLE_FIXUP(VkFramebuffer);
NO_HANDLE_FIXUP(VkRenderPass);
NO_HANDLE_FIXUP(VkCommandPool);
NO_HANDLE_FIXUP(VkSamplerYcbcrConversion);
NO_HANDLE_FIXUP(VkDescriptorUpdateTemplate);
NO_HANDLE_FIXUP(VkSurfaceKHR);
NO_HANDLE_FIXUP(VkSwapchainKHR);

template <>
struct needs_dispatch_fixup<VkPhysicalDevice> {
  static const bool val = true;
};

template <>
struct needs_dispatch_fixup<VkQueue> {
  static const bool val = true;
};

template <>
struct needs_dispatch_fixup<VkCommandBuffer> {
  static const bool val = true;
};
