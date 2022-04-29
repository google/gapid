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

#include <type_traits>

#include "common.h"
#include "state_block.h"
#include "transform_base.h"

namespace gapid2 {

template <typename T, typename... Ts>
constexpr bool args_contain() { return std::disjunction_v<std::is_same<T, Ts>...>; }

template <typename... Args>
class creation_tracker : public transform_base {
 protected:
  using super = transform_base;

 public:
  VkResult vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkInstance* pInstance) override {
    if constexpr (args_contain<VkInstance, Args...>()) {
      auto res = super::vkCreateInstance(pCreateInfo, pAllocator, pInstance);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pInstance), "Instance already exists");
      return res;
    } else {
      return super::vkCreateInstance(pCreateInfo, pAllocator, pInstance);
    }
  }

  VkResult vkCreateDevice(VkPhysicalDevice physicalDevice,
                          const VkDeviceCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkDevice* pDevice) override {
    if constexpr (args_contain<VkDevice, Args...>()) {
      auto res =
          super::vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pDevice), "Device already exists");
      return res;
    } else {
      return super::vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
    }
  }

  void vkGetDeviceQueue(VkDevice device,
                        uint32_t queueFamilyIndex,
                        uint32_t queueIndex,
                        VkQueue* pQueue) override {
    if constexpr (args_contain<VkQueue, Args...>()) {
      super::vkGetDeviceQueue(device, queueFamilyIndex, queueIndex, pQueue);
      GAPID2_ASSERT(state_block_->get_or_create(*pQueue), "Queue already exists");
    } else {
      super::vkGetDeviceQueue(device, queueFamilyIndex, queueIndex, pQueue);
    }
  }

  void vkGetDeviceQueue2(VkDevice device,
                         const VkDeviceQueueInfo2* pQueueInfo,
                         VkQueue* pQueue) override {
    if constexpr (args_contain<VkQueue, Args...>()) {
      super::vkGetDeviceQueue2(device, pQueueInfo, pQueue);
      GAPID2_ASSERT(state_block_->get_or_create(*pQueue), "Queue already exists");
    } else {
      super::vkGetDeviceQueue2(device, pQueueInfo, pQueue);
    }
  }

  VkResult vkAllocateMemory(VkDevice device,
                            const VkMemoryAllocateInfo* pAllocateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkDeviceMemory* pMemory) override {
    if constexpr (args_contain<VkDeviceMemory, Args...>()) {
      auto res =
          super::vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pMemory), "Memory already exists");
      return res;
    } else {
      return super::vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
    }
  }

  VkResult vkCreateFence(VkDevice device,
                         const VkFenceCreateInfo* pCreateInfo,
                         const VkAllocationCallbacks* pAllocator,
                         VkFence* pFence) override {
    if constexpr (args_contain<VkFence, Args...>()) {
      auto res = super::vkCreateFence(device, pCreateInfo, pAllocator, pFence);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pFence), "Memory already exists");
      return res;
    } else {
      return super::vkCreateFence(device, pCreateInfo, pAllocator, pFence);
    }
  }

  VkResult vkCreateSemaphore(VkDevice device,
                             const VkSemaphoreCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkSemaphore* pSemaphore) override {
    if constexpr (args_contain<VkSemaphore, Args...>()) {
      auto res =
          super::vkCreateSemaphore(device, pCreateInfo, pAllocator, pSemaphore);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pSemaphore), "Semaphore already exists");
      return res;
    } else {
      return super::vkCreateSemaphore(device, pCreateInfo, pAllocator, pSemaphore);
    }
  }

  VkResult vkCreateEvent(VkDevice device,
                         const VkEventCreateInfo* pCreateInfo,
                         const VkAllocationCallbacks* pAllocator,
                         VkEvent* pEvent) override {
    if constexpr (args_contain<VkEvent, Args...>()) {
      auto res = super::vkCreateEvent(device, pCreateInfo, pAllocator, pEvent);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pEvent), "Event already exists");
      return res;
    } else {
      return super::vkCreateEvent(device, pCreateInfo, pAllocator, pEvent);
    }
  }

  VkResult vkCreateQueryPool(VkDevice device,
                             const VkQueryPoolCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkQueryPool* pQueryPool) override {
    if constexpr (args_contain<VkQueryPool, Args...>()) {
      auto res =
          super::vkCreateQueryPool(device, pCreateInfo, pAllocator, pQueryPool);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pQueryPool), "QueryPool already exists");
      return res;
    } else {
      return super::vkCreateQueryPool(device, pCreateInfo, pAllocator, pQueryPool);
    }
  }

  VkResult vkCreateBuffer(VkDevice device,
                          const VkBufferCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkBuffer* pBuffer) override {
    if constexpr (args_contain<VkBuffer, Args...>()) {
      auto res = super::vkCreateBuffer(device, pCreateInfo, pAllocator, pBuffer);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pBuffer), "Buffer already exists");
      return res;
    } else {
      return super::vkCreateBuffer(device, pCreateInfo, pAllocator, pBuffer);
    }
  }

  VkResult vkCreateBufferView(VkDevice device,
                              const VkBufferViewCreateInfo* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkBufferView* pView) override {
    if constexpr (args_contain<VkBufferView, Args...>()) {
      auto res =
          super::vkCreateBufferView(device, pCreateInfo, pAllocator, pView);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pView), "BufferView already exists");
      return res;
    } else {
      return super::vkCreateBufferView(device, pCreateInfo, pAllocator, pView);
    }
  }

  VkResult vkCreateImage(VkDevice device,
                         const VkImageCreateInfo* pCreateInfo,
                         const VkAllocationCallbacks* pAllocator,
                         VkImage* pImage) override {
    if constexpr (args_contain<VkImage, Args...>()) {
      auto res = super::vkCreateImage(device, pCreateInfo, pAllocator, pImage);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pImage), "Image already exists");
      return res;
    } else {
      return super::vkCreateImage(device, pCreateInfo, pAllocator, pImage);
    }
  }

  VkResult vkCreateImageView(VkDevice device,
                             const VkImageViewCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkImageView* pView) override {
    if constexpr (args_contain<VkImageView, Args...>()) {
      auto res = super::vkCreateImageView(device, pCreateInfo, pAllocator, pView);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pView), "ImageView already exists");
      return res;
    } else {
      return super::vkCreateImageView(device, pCreateInfo, pAllocator, pView);
    }
  }

  VkResult vkCreateShaderModule(VkDevice device,
                                const VkShaderModuleCreateInfo* pCreateInfo,
                                const VkAllocationCallbacks* pAllocator,
                                VkShaderModule* pShaderModule) override {
    if constexpr (args_contain<VkShaderModule, Args...>()) {
      auto res = super::vkCreateShaderModule(device, pCreateInfo, pAllocator,
                                             pShaderModule);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pShaderModule), "ShaderModule already exists");
      return res;
    } else {
      return super::vkCreateShaderModule(device, pCreateInfo, pAllocator,
                                         pShaderModule);
    }
  }
  VkResult vkCreatePipelineCache(VkDevice device,
                                 const VkPipelineCacheCreateInfo* pCreateInfo,
                                 const VkAllocationCallbacks* pAllocator,
                                 VkPipelineCache* pPipelineCache) override {
    if constexpr (args_contain<VkPipelineCache, Args...>()) {
      auto res = super::vkCreatePipelineCache(device, pCreateInfo, pAllocator,
                                              pPipelineCache);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pPipelineCache), "PipelineCache already exists");
      return res;
    } else {
      return super::vkCreatePipelineCache(device, pCreateInfo, pAllocator,
                                          pPipelineCache);
    }
  }

  VkResult vkCreateGraphicsPipelines(
      VkDevice device,
      VkPipelineCache pipelineCache,
      uint32_t createInfoCount,
      const VkGraphicsPipelineCreateInfo* pCreateInfos,
      const VkAllocationCallbacks* pAllocator,
      VkPipeline* pPipelines) override {
    if constexpr (args_contain<VkPipeline, Args...>()) {
      auto res =
          super::vkCreateGraphicsPipelines(device, pipelineCache, createInfoCount,
                                           pCreateInfos, pAllocator, pPipelines);
      if (res != VK_SUCCESS) {
        return res;
      }
      for (size_t i = 0; i < createInfoCount; ++i) {
        GAPID2_ASSERT(state_block_->create(pPipelines[i]), "Pipeline already exists");
      }
      return res;
    } else {
      return super::vkCreateGraphicsPipelines(device, pipelineCache, createInfoCount,
                                              pCreateInfos, pAllocator, pPipelines);
    }
  }

  VkResult vkCreateComputePipelines(
      VkDevice device,
      VkPipelineCache pipelineCache,
      uint32_t createInfoCount,
      const VkComputePipelineCreateInfo* pCreateInfos,
      const VkAllocationCallbacks* pAllocator,
      VkPipeline* pPipelines) override {
    if constexpr (args_contain<VkPipeline, Args...>()) {
      auto res =
          super::vkCreateComputePipelines(device, pipelineCache, createInfoCount,
                                          pCreateInfos, pAllocator, pPipelines);
      if (res != VK_SUCCESS) {
        return res;
      }
      for (size_t i = 0; i < createInfoCount; ++i) {
        GAPID2_ASSERT(state_block_->create(pPipelines[i]), "Pipeline already exists");
      }
      return res;
    } else {
      return super::vkCreateComputePipelines(device, pipelineCache, createInfoCount,
                                             pCreateInfos, pAllocator, pPipelines);
    }
  }
  VkResult vkCreatePipelineLayout(VkDevice device,
                                  const VkPipelineLayoutCreateInfo* pCreateInfo,
                                  const VkAllocationCallbacks* pAllocator,
                                  VkPipelineLayout* pPipelineLayout) override {
    if constexpr (args_contain<VkPipelineLayout, Args...>()) {
      auto res = super::vkCreatePipelineLayout(device, pCreateInfo, pAllocator,
                                               pPipelineLayout);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pPipelineLayout), "PipelineLayout already exists");

      return res;
    } else {
      return super::vkCreatePipelineLayout(device, pCreateInfo, pAllocator,
                                           pPipelineLayout);
    }
  }
  VkResult vkCreateSampler(VkDevice device,
                           const VkSamplerCreateInfo* pCreateInfo,
                           const VkAllocationCallbacks* pAllocator,
                           VkSampler* pSampler) override {
    if constexpr (args_contain<VkSampler, Args...>()) {
      auto res =
          super::vkCreateSampler(device, pCreateInfo, pAllocator, pSampler);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pSampler), "Sampler already exists");
      return res;
    } else {
      return super::vkCreateSampler(device, pCreateInfo, pAllocator, pSampler);
    }
  }

  VkResult vkCreateDescriptorSetLayout(
      VkDevice device,
      const VkDescriptorSetLayoutCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorSetLayout* pSetLayout) override {
    if constexpr (args_contain<VkDescriptorSetLayout, Args...>()) {
      auto res = super::vkCreateDescriptorSetLayout(device, pCreateInfo,
                                                    pAllocator, pSetLayout);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pSetLayout), "DescriptorSetLayout already exists");
      return res;
    } else {
      return super::vkCreateDescriptorSetLayout(device, pCreateInfo,
                                                pAllocator, pSetLayout);
    }
  }

  VkResult vkCreateDescriptorPool(VkDevice device,
                                  const VkDescriptorPoolCreateInfo* pCreateInfo,
                                  const VkAllocationCallbacks* pAllocator,
                                  VkDescriptorPool* pDescriptorPool) override {
    if constexpr (args_contain<VkDescriptorPool, Args...>()) {
      auto res = super::vkCreateDescriptorPool(device, pCreateInfo, pAllocator,
                                               pDescriptorPool);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pDescriptorPool), "DescriptorPool already exists");
      return res;
    } else {
      return super::vkCreateDescriptorPool(device, pCreateInfo, pAllocator,
                                           pDescriptorPool);
    }
  }

  VkResult vkAllocateDescriptorSets(
      VkDevice device,
      const VkDescriptorSetAllocateInfo* pAllocateInfo,
      VkDescriptorSet* pDescriptorSets) override {
    if constexpr (args_contain<VkDescriptorSet, Args...>()) {
      auto res =
          super::vkAllocateDescriptorSets(device, pAllocateInfo, pDescriptorSets);
      if (res != VK_SUCCESS) {
        return res;
      }
      for (uint32_t i = 0; i < pAllocateInfo->descriptorSetCount; ++i) {
        GAPID2_ASSERT(state_block_->create(pDescriptorSets[i]), "DescriptorSet already exists");
      }
      return res;
    } else {
      return super::vkAllocateDescriptorSets(device, pAllocateInfo, pDescriptorSets);
    }
  }
  VkResult vkCreateFramebuffer(VkDevice device,
                               const VkFramebufferCreateInfo* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkFramebuffer* pFramebuffer) override {
    if constexpr (args_contain<VkFramebuffer, Args...>()) {
      auto res = super::vkCreateFramebuffer(device, pCreateInfo, pAllocator,
                                            pFramebuffer);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pFramebuffer), "Framebuffer already exists");
      return res;
    } else {
      return super::vkCreateFramebuffer(device, pCreateInfo, pAllocator,
                                        pFramebuffer);
    }
  }

  VkResult vkCreateRenderPass(VkDevice device,
                              const VkRenderPassCreateInfo* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkRenderPass* pRenderPass) override {
    if constexpr (args_contain<VkRenderPass, Args...>()) {
      auto res =
          super::vkCreateRenderPass(device, pCreateInfo, pAllocator, pRenderPass);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pRenderPass), "RenderPass already exists");
      return res;
    } else {
      return super::vkCreateRenderPass(device, pCreateInfo, pAllocator, pRenderPass);
    }
  }

  VkResult vkCreateRenderPass2(VkDevice device,
                               const VkRenderPassCreateInfo2* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkRenderPass* pRenderPass) override {
    if constexpr (args_contain<VkRenderPass, Args...>()) {
      auto res = super::vkCreateRenderPass2(device, pCreateInfo, pAllocator,
                                            pRenderPass);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pRenderPass), "RenderPass already exists");
      return res;
    } else {
      return super::vkCreateRenderPass2(device, pCreateInfo, pAllocator,
                                        pRenderPass);
    }
  }

  VkResult vkCreateCommandPool(VkDevice device,
                               const VkCommandPoolCreateInfo* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkCommandPool* pCommandPool) override {
    if constexpr (args_contain<VkCommandPool, Args...>()) {
      auto res = super::vkCreateCommandPool(device, pCreateInfo, pAllocator,
                                            pCommandPool);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pCommandPool), "CommandPool already exists");
      return res;
    } else {
      return super::vkCreateCommandPool(device, pCreateInfo, pAllocator,
                                        pCommandPool);
    }
  }
  VkResult vkAllocateCommandBuffers(
      VkDevice device,
      const VkCommandBufferAllocateInfo* pAllocateInfo,
      VkCommandBuffer* pCommandBuffers) override {
    if constexpr (args_contain<VkCommandBuffer, Args...>()) {
      auto res =
          super::vkAllocateCommandBuffers(device, pAllocateInfo, pCommandBuffers);
      if (res != VK_SUCCESS) {
        return res;
      }
      for (uint32_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
        GAPID2_ASSERT(state_block_->create(pCommandBuffers[i]), "CommandBuffer already exists");
      }
      return res;
    } else {
      return super::vkAllocateCommandBuffers(device, pAllocateInfo, pCommandBuffers);
    }
  }

  VkResult vkCreateSamplerYcbcrConversion(
      VkDevice device,
      const VkSamplerYcbcrConversionCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkSamplerYcbcrConversion* pYcbcrConversion) override {
    if constexpr (args_contain<VkSamplerYcbcrConversion, Args...>()) {
      auto res = super::vkCreateSamplerYcbcrConversion(
          device, pCreateInfo, pAllocator, pYcbcrConversion);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pYcbcrConversion), "YcbcrConversion already exists");
      return res;
    } else {
      return super::vkCreateSamplerYcbcrConversion(
          device, pCreateInfo, pAllocator, pYcbcrConversion);
    }
  }

  VkResult vkCreateDescriptorUpdateTemplate(
      VkDevice device,
      const VkDescriptorUpdateTemplateCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorUpdateTemplate* pDescriptorUpdateTemplate) override {
    if constexpr (args_contain<VkDescriptorUpdateTemplate, Args...>()) {
      auto res = super::vkCreateDescriptorUpdateTemplate(
          device, pCreateInfo, pAllocator, pDescriptorUpdateTemplate);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pDescriptorUpdateTemplate), "DescriptorUpdateTemplate already exists");
      return res;
    } else {
      return super::vkCreateDescriptorUpdateTemplate(
          device, pCreateInfo, pAllocator, pDescriptorUpdateTemplate);
    }
  }
  VkResult vkCreateWin32SurfaceKHR(
      VkInstance instance,
      const VkWin32SurfaceCreateInfoKHR* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkSurfaceKHR* pSurface) override {
    if constexpr (args_contain<VkSurfaceKHR, Args...>()) {
      auto res = super::vkCreateWin32SurfaceKHR(instance, pCreateInfo, pAllocator,
                                                pSurface);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pSurface), "Surface already exists");
      return res;
    } else {
      return super::vkCreateWin32SurfaceKHR(instance, pCreateInfo, pAllocator,
                                            pSurface);
    }
  }

  VkResult vkCreateSwapchainKHR(VkDevice device,
                                const VkSwapchainCreateInfoKHR* pCreateInfo,
                                const VkAllocationCallbacks* pAllocator,
                                VkSwapchainKHR* pSwapchain) override {
    if constexpr (args_contain<VkSwapchainKHR, Args...>()) {
      auto res = super::vkCreateSwapchainKHR(device, pCreateInfo, pAllocator,
                                             pSwapchain);
      if (res != VK_SUCCESS) {
        return res;
      }
      GAPID2_ASSERT(state_block_->create(*pSwapchain), "Swapchain already exists");
      return res;
    } else {
      return super::vkCreateSwapchainKHR(device, pCreateInfo, pAllocator,
                                         pSwapchain);
    }
  }

  VkResult vkGetSwapchainImagesKHR(VkDevice device,
                                   VkSwapchainKHR swapchain,
                                   uint32_t* pSwapchainImageCount,
                                   VkImage* pSwapchainImages) override {
    if constexpr (args_contain<VkSwapchainKHR, Args...>()) {
      auto res = super::vkGetSwapchainImagesKHR(
          device, swapchain, pSwapchainImageCount, pSwapchainImages);
      if (res != VK_SUCCESS) {
        return res;
      }
      if (pSwapchainImages != nullptr) {
        for (uint32_t i = 0; i < *pSwapchainImageCount; ++i) {
          GAPID2_ASSERT(state_block_->create(pSwapchainImages[i]), "Swapchain Image already exists");
        }
      }
      return res;
    } else {
      return super::vkGetSwapchainImagesKHR(
          device, swapchain, pSwapchainImageCount, pSwapchainImages);
    }
  }

  void vkDestroyInstance(VkInstance instance, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkInstance, Args...>()) {
      if (instance) {
        GAPID2_ASSERT(state_block_->erase(instance), "Could not find instance to erase");
      }
    }

    return super::vkDestroyInstance(instance, pAllocator);
  }

  void vkDestroyDevice(VkDevice device, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkDevice, Args...>()) {
      if (device) {
        GAPID2_ASSERT(state_block_->erase(device), "Could not find device to erase");
      }
    }
    return super::vkDestroyDevice(device, pAllocator);
  }

  void vkFreeCommandBuffers(VkDevice device, VkCommandPool commandPool, uint32_t commandBufferCount, const VkCommandBuffer* pCommandBuffers) override {
    if constexpr (args_contain<VkCommandBuffer, Args...>()) {
      for (size_t i = 0; i < commandBufferCount; ++i) {
        if (pCommandBuffers[i]) {
          GAPID2_ASSERT(state_block_->erase(pCommandBuffers[i]), "Could not find pCommandBuffers to erase");
        }
      }
    }
    return super::vkFreeCommandBuffers(device, commandPool, commandBufferCount, pCommandBuffers);
  }

  void vkFreeMemory(VkDevice device, VkDeviceMemory memory, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkDeviceMemory, Args...>()) {
      if (memory) {
        GAPID2_ASSERT(state_block_->erase(memory), "Could not find memory to erase");
      }
    }
    return super::vkFreeMemory(device, memory, pAllocator);
  }
  VkResult vkFreeDescriptorSets(VkDevice device, VkDescriptorPool descriptorPool, uint32_t descriptorSetCount, const VkDescriptorSet* pDescriptorSets) override {
    if constexpr (args_contain<VkDescriptorSet, Args...>()) {
      for (size_t i = 0; i < descriptorSetCount; ++i) {
        if (pDescriptorSets[0]) {
          GAPID2_ASSERT(state_block_->erase(pDescriptorSets[0]), "Could not find pDescriptorSets to erase");
        }
      }
    }
    return super::vkFreeDescriptorSets(device, descriptorPool, descriptorSetCount, pDescriptorSets);
  }

  void vkDestroyFence(VkDevice device, VkFence fence, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkFence, Args...>()) {
      if (fence) {
        GAPID2_ASSERT(state_block_->erase(fence), "Could not find fence to erase");
      }
    }
    return super::vkDestroyFence(device, fence, pAllocator);
  }

  void vkDestroySemaphore(VkDevice device, VkSemaphore semaphore, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkSemaphore, Args...>()) {
      if (semaphore) {
        GAPID2_ASSERT(state_block_->erase(semaphore), "Could not find semaphore to erase");
      }
    }
    return super::vkDestroySemaphore(device, semaphore, pAllocator);
  }

  void vkDestroyEvent(VkDevice device, VkEvent event, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkEvent, Args...>()) {
      if (event) {
        GAPID2_ASSERT(state_block_->erase(event), "Could not find event to erase");
      }
    }
    return super::vkDestroyEvent(device, event, pAllocator);
  }

  void vkDestroyQueryPool(VkDevice device, VkQueryPool queryPool, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkQueryPool, Args...>()) {
      if (queryPool) {
        GAPID2_ASSERT(state_block_->erase(queryPool), "Could not find queryPool to erase");
      }
    }
    return super::vkDestroyQueryPool(device, queryPool, pAllocator);
  }

  void vkDestroyBuffer(VkDevice device, VkBuffer buffer, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkBuffer, Args...>()) {
      if (buffer) {
        GAPID2_ASSERT(state_block_->erase(buffer), "Could not find buffer to erase");
      }
    }
    return super::vkDestroyBuffer(device, buffer, pAllocator);
  }
  void vkDestroyBufferView(VkDevice device, VkBufferView bufferView, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkBufferView, Args...>()) {
      if (bufferView) {
        GAPID2_ASSERT(state_block_->erase(bufferView), "Could not find bufferView to erase");
      }
    }
    return super::vkDestroyBufferView(device, bufferView, pAllocator);
  }

  void vkDestroyImage(VkDevice device, VkImage image, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkImage, Args...>()) {
      if (image) {
        GAPID2_ASSERT(state_block_->erase(image), "Could not find image to erase");
      }
    }
    return super::vkDestroyImage(device, image, pAllocator);
  }

  void vkDestroyImageView(VkDevice device, VkImageView imageView, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkImageView, Args...>()) {
      if (imageView) {
        GAPID2_ASSERT(state_block_->erase(imageView), "Could not find imageView to erase");
      }
    }
    return super::vkDestroyImageView(device, imageView, pAllocator);
  }

  void vkDestroyShaderModule(VkDevice device, VkShaderModule shaderModule, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkShaderModule, Args...>()) {
      if (shaderModule) {
        GAPID2_ASSERT(state_block_->erase(shaderModule), "Could not find shaderModule to erase");
      }
    }
    return super::vkDestroyShaderModule(device, shaderModule, pAllocator);
  }

  void vkDestroyPipelineCache(VkDevice device, VkPipelineCache pipelineCache, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkPipelineCache, Args...>()) {
      if (pipelineCache) {
        GAPID2_ASSERT(state_block_->erase(pipelineCache), "Could not find pipelineCache to erase");
      }
    }
    return super::vkDestroyPipelineCache(device, pipelineCache, pAllocator);
  }

  void vkDestroyPipeline(VkDevice device, VkPipeline pipeline, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkPipeline, Args...>()) {
      if (pipeline) {
        GAPID2_ASSERT(state_block_->erase(pipeline), "Could not find pipeline to erase");
      }
    }
    return super::vkDestroyPipeline(device, pipeline, pAllocator);
  }

  void vkDestroyPipelineLayout(VkDevice device, VkPipelineLayout pipelineLayout, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkPipelineLayout, Args...>()) {
      if (pipelineLayout) {
        GAPID2_ASSERT(state_block_->erase(pipelineLayout), "Could not find pipelineLayout to erase");
      }
    }
    return super::vkDestroyPipelineLayout(device, pipelineLayout, pAllocator);
  }

  void vkDestroySampler(VkDevice device, VkSampler sampler, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkSampler, Args...>()) {
      if (sampler) {
        GAPID2_ASSERT(state_block_->erase(sampler), "Could not find sampler to erase");
      }
    }
    return super::vkDestroySampler(device, sampler, pAllocator);
  }

  void vkDestroyDescriptorSetLayout(VkDevice device, VkDescriptorSetLayout descriptorSetLayout, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkDescriptorSetLayout, Args...>()) {
      if (descriptorSetLayout) {
        GAPID2_ASSERT(state_block_->erase(descriptorSetLayout), "Could not find descriptorSetLayout to erase");
      }
    }
    return super::vkDestroyDescriptorSetLayout(device, descriptorSetLayout, pAllocator);
  }

  void vkDestroyDescriptorPool(VkDevice device, VkDescriptorPool descriptorPool, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkDescriptorPool, Args...>()) {
      if (descriptorPool) {
        GAPID2_ASSERT(state_block_->erase(descriptorPool), "Could not find descriptorPool to erase");
      }
    }
    return super::vkDestroyDescriptorPool(device, descriptorPool, pAllocator);
  }

  void vkDestroyFramebuffer(VkDevice device, VkFramebuffer framebuffer, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkFramebuffer, Args...>()) {
      if (framebuffer) {
        GAPID2_ASSERT(state_block_->erase(framebuffer), "Could not find framebuffer to erase");
      }
    }
    return super::vkDestroyFramebuffer(device, framebuffer, pAllocator);
  }

  void vkDestroyRenderPass(VkDevice device, VkRenderPass renderPass, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkRenderPass, Args...>()) {
      if (renderPass) {
        GAPID2_ASSERT(state_block_->erase(renderPass), "Could not find renderPass to erase");
      }
    }
    return super::vkDestroyRenderPass(device, renderPass, pAllocator);
  }

  void vkDestroyCommandPool(VkDevice device, VkCommandPool commandPool, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkCommandPool, Args...>()) {
      if (commandPool) {
        GAPID2_ASSERT(state_block_->erase(commandPool), "Could not find commandPool to erase");
      }
    }
    return super::vkDestroyCommandPool(device, commandPool, pAllocator);
  }

  void vkDestroySamplerYcbcrConversion(VkDevice device, VkSamplerYcbcrConversion ycbcrConversion, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkSamplerYcbcrConversion, Args...>()) {
      if (ycbcrConversion) {
        GAPID2_ASSERT(state_block_->erase(ycbcrConversion), "Could not find ycbcrConversion to erase");
      }
    }
    return super::vkDestroySamplerYcbcrConversion(device, ycbcrConversion, pAllocator);
  }

  void vkDestroyDescriptorUpdateTemplate(VkDevice device, VkDescriptorUpdateTemplate descriptorUpdateTemplate, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkDescriptorUpdateTemplate, Args...>()) {
      if (descriptorUpdateTemplate) {
        GAPID2_ASSERT(state_block_->erase(descriptorUpdateTemplate), "Could not find descriptorUpdateTemplate to erase");
      }
    }
    return super::vkDestroyDescriptorUpdateTemplate(device, descriptorUpdateTemplate, pAllocator);
  }

  void vkDestroySurfaceKHR(VkInstance instance, VkSurfaceKHR surface, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkSurfaceKHR, Args...>()) {
      if (surface) {
        GAPID2_ASSERT(state_block_->erase(surface), "Could not find surface to erase");
      }
    }
    return super::vkDestroySurfaceKHR(instance, surface, pAllocator);
  }

  void vkDestroySwapchainKHR(VkDevice device, VkSwapchainKHR swapchain, const VkAllocationCallbacks* pAllocator) override {
    if constexpr (args_contain<VkSwapchainKHR, Args...>()) {
      if (swapchain) {
        GAPID2_ASSERT(state_block_->erase(swapchain), "Could not find swapchain to erase");
      }
    }
    return super::vkDestroySwapchainKHR(device, swapchain, pAllocator);
  }

 protected:
};
}  // namespace gapid2