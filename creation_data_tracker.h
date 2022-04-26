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
class creation_data_tracker : public transform_base {
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
      auto pl = state_block_->get(pInstance[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pDevice[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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

      auto pl = state_block_->get(pQueue[0]);
      pl->set_create_info(state_block_, queueFamilyIndex, queueIndex);
    } else {
      super::vkGetDeviceQueue(device, queueFamilyIndex, queueIndex, pQueue);
    }
  }

  void vkGetDeviceQueue2(VkDevice device,
                         const VkDeviceQueueInfo2* pQueueInfo,
                         VkQueue* pQueue) override {
    if constexpr (args_contain<VkQueue, Args...>()) {
      super::vkGetDeviceQueue2(device, pQueueInfo, pQueue);

      auto pl = state_block_->get(pQueue[0]);
      pl->set_create_info2(pQueueInfo);
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
      auto pl = state_block_->get(pMemory[0]);
      pl->set_allocate_info(state_block_, pAllocateInfo);
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
      auto pl = state_block_->get(pFence[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pSemaphore[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pEvent[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pQueryPool[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pBuffer[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pView[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pImage[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pView[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pShaderModule[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pPipelineCache[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
        auto pl = state_block_->get(pPipelines[i]);
        pl->set_create_info(state_block_, pipelineCache, &pCreateInfos[i]);
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
        auto pl = state_block_->get(pPipelines[i]);
        pl->set_create_info(state_block_, pipelineCache, &pCreateInfos[i]);
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
      auto pl = state_block_->get(pPipelineLayout[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pSampler[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pSetLayout[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pDescriptorPool[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
        auto pl = state_block_->get(pDescriptorSets[i]);
        pl->set_allocate_info(state_block_, mpAllocateInfo, i);
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
      auto pl = state_block_->get(pFramebuffer[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pRenderPass[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pRenderPass[0]);
      pl->set_create_info2(pCreateInfo);
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
      auto pl = state_block_->get(pCommandPool[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
        auto pl = state_block_->get(pCommandBuffers[i]);
        pl->set_allocate_info(state_block_, mpAllocateInfo, i);
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
      auto pl = state_block_->get(pYcbcrConversion[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pDescriptorUpdateTemplate[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pSurface[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
      auto pl = state_block_->get(pSwapchain[0]);
      pl->set_create_info(state_block_, pCreateInfo);
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
          auto pl = state_block_->get(pSwapchainImages[i]);
          pl->set_swapchain_info(swapchain, i);
        }
      }
      return res;
    } else {
      return super::vkGetSwapchainImagesKHR(
          device, swapchain, pSwapchainImageCount, pSwapchainImages);
    }
  }

 protected:
};
}  // namespace gapid2