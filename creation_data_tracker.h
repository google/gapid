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
#define NOMINMAX
#include <vulkan.h>

namespace gapid2 {
template <typename T>
class CreationDataTracker : public T {
 protected:
  using super = T;

 public:
  VkResult vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkInstance* pInstance) override {
    auto res = super::vkCreateInstance(pCreateInfo, pAllocator, pInstance);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pInstance[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateDevice(VkPhysicalDevice physicalDevice,
                          const VkDeviceCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkDevice* pDevice) override {
    auto res =
        super::vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pDevice[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  void vkGetDeviceQueue(VkDevice device,
                        uint32_t queueFamilyIndex,
                        uint32_t queueIndex,
                        VkQueue* pQueue) override {
    super::vkGetDeviceQueue(device, queueFamilyIndex, queueIndex, pQueue);

    auto pl = this->updater_.cast_from_vk(pQueue[0]);
    pl->set_create_info(queueFamilyIndex, queueIndex);
  }

  void vkGetDeviceQueue2(VkDevice device,
                         const VkDeviceQueueInfo2* pQueueInfo,
                         VkQueue* pQueue) override {
    super::vkGetDeviceQueue2(device, pQueueInfo, pQueue);

    auto pl = this->updater_.cast_from_vk(pQueue[0]);
    pl->set_create_info2(pQueueInfo);
  }

  VkResult vkAllocateMemory(VkDevice device,
                            const VkMemoryAllocateInfo* pAllocateInfo,
                            const VkAllocationCallbacks* pAllocator,
                            VkDeviceMemory* pMemory) override {
    auto res =
        super::vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pMemory[0]);
    pl->set_allocate_info(pAllocateInfo);
    return res;
  }

  VkResult vkCreateFence(VkDevice device,
                         const VkFenceCreateInfo* pCreateInfo,
                         const VkAllocationCallbacks* pAllocator,
                         VkFence* pFence) override {
    auto res = super::vkCreateFence(device, pCreateInfo, pAllocator, pFence);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pFence[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateSemaphore(VkDevice device,
                             const VkSemaphoreCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkSemaphore* pSemaphore) override {
    auto res =
        super::vkCreateSemaphore(device, pCreateInfo, pAllocator, pSemaphore);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pSemaphore[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateEvent(VkDevice device,
                         const VkEventCreateInfo* pCreateInfo,
                         const VkAllocationCallbacks* pAllocator,
                         VkEvent* pEvent) override {
    auto res = super::vkCreateEvent(device, pCreateInfo, pAllocator, pEvent);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pEvent[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateQueryPool(VkDevice device,
                             const VkQueryPoolCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkQueryPool* pQueryPool) override {
    auto res =
        super::vkCreateQueryPool(device, pCreateInfo, pAllocator, pQueryPool);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pQueryPool[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateBuffer(VkDevice device,
                          const VkBufferCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkBuffer* pBuffer) override {
    auto res = super::vkCreateBuffer(device, pCreateInfo, pAllocator, pBuffer);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pBuffer[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateBufferView(VkDevice device,
                              const VkBufferViewCreateInfo* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkBufferView* pView) override {
    auto res =
        super::vkCreateBufferView(device, pCreateInfo, pAllocator, pView);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pView[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateImage(VkDevice device,
                         const VkImageCreateInfo* pCreateInfo,
                         const VkAllocationCallbacks* pAllocator,
                         VkImage* pImage) override {
    auto res = super::vkCreateImage(device, pCreateInfo, pAllocator, pImage);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pImage[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateImageView(VkDevice device,
                             const VkImageViewCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkImageView* pView) override {
    auto res = super::vkCreateImageView(device, pCreateInfo, pAllocator, pView);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pView[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateShaderModule(VkDevice device,
                                const VkShaderModuleCreateInfo* pCreateInfo,
                                const VkAllocationCallbacks* pAllocator,
                                VkShaderModule* pShaderModule) override {
    auto res = super::vkCreateShaderModule(device, pCreateInfo, pAllocator,
                                           pShaderModule);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pShaderModule[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }
  VkResult vkCreatePipelineCache(VkDevice device,
                                 const VkPipelineCacheCreateInfo* pCreateInfo,
                                 const VkAllocationCallbacks* pAllocator,
                                 VkPipelineCache* pPipelineCache) override {
    auto res = super::vkCreatePipelineCache(device, pCreateInfo, pAllocator,
                                            pPipelineCache);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pPipelineCache[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateGraphicsPipelines(
      VkDevice device,
      VkPipelineCache pipelineCache,
      uint32_t createInfoCount,
      const VkGraphicsPipelineCreateInfo* pCreateInfos,
      const VkAllocationCallbacks* pAllocator,
      VkPipeline* pPipelines) override {
    auto res =
        super::vkCreateGraphicsPipelines(device, pipelineCache, createInfoCount,
                                         pCreateInfos, pAllocator, pPipelines);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (size_t i = 0; i < createInfoCount; ++i) {
      auto pl = this->updater_.cast_from_vk(pPipelines[i]);
      pl->set_create_info(pipelineCache, &pCreateInfos[i]);
    }
    return res;
  }

  VkResult vkCreateComputePipelines(
      VkDevice device,
      VkPipelineCache pipelineCache,
      uint32_t createInfoCount,
      const VkComputePipelineCreateInfo* pCreateInfos,
      const VkAllocationCallbacks* pAllocator,
      VkPipeline* pPipelines) override {
    auto res =
        super::vkCreateComputePipelines(device, pipelineCache, createInfoCount,
                                        pCreateInfos, pAllocator, pPipelines);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (size_t i = 0; i < createInfoCount; ++i) {
      auto pl = this->updater_.cast_from_vk(pPipelines[i]);
      pl->set_create_info(pipelineCache, &pCreateInfos[i]);
    }
    return res;
  }
  VkResult vkCreatePipelineLayout(VkDevice device,
                                  const VkPipelineLayoutCreateInfo* pCreateInfo,
                                  const VkAllocationCallbacks* pAllocator,
                                  VkPipelineLayout* pPipelineLayout) override {
    auto res = super::vkCreatePipelineLayout(device, pCreateInfo, pAllocator,
                                             pPipelineLayout);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pPipelineLayout[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }
  VkResult vkCreateSampler(VkDevice device,
                           const VkSamplerCreateInfo* pCreateInfo,
                           const VkAllocationCallbacks* pAllocator,
                           VkSampler* pSampler) override {
    auto res =
        super::vkCreateSampler(device, pCreateInfo, pAllocator, pSampler);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pSampler[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }
  VkResult vkCreateDescriptorSetLayout(
      VkDevice device,
      const VkDescriptorSetLayoutCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorSetLayout* pSetLayout) override {
    auto res = super::vkCreateDescriptorSetLayout(device, pCreateInfo,
                                                  pAllocator, pSetLayout);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pSetLayout[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateDescriptorPool(VkDevice device,
                                  const VkDescriptorPoolCreateInfo* pCreateInfo,
                                  const VkAllocationCallbacks* pAllocator,
                                  VkDescriptorPool* pDescriptorPool) override {
    auto res = super::vkCreateDescriptorPool(device, pCreateInfo, pAllocator,
                                             pDescriptorPool);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pDescriptorPool[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkAllocateDescriptorSets(
      VkDevice device,
      const VkDescriptorSetAllocateInfo* pAllocateInfo,
      VkDescriptorSet* pDescriptorSets) override {
    auto res =
        super::vkAllocateDescriptorSets(device, pAllocateInfo, pDescriptorSets);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (uint32_t i = 0; i < pAllocateInfo->descriptorSetCount; ++i) {
      auto pl = this->updater_.cast_from_vk(pDescriptorSets[i]);
      pl->set_allocate_info(pAllocateInfo, i);
    }
    return res;
  }
  VkResult vkCreateFramebuffer(VkDevice device,
                               const VkFramebufferCreateInfo* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkFramebuffer* pFramebuffer) override {
    auto res = super::vkCreateFramebuffer(device, pCreateInfo, pAllocator,
                                          pFramebuffer);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pFramebuffer[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }
  VkResult vkCreateRenderPass(VkDevice device,
                              const VkRenderPassCreateInfo* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkRenderPass* pRenderPass) override {
    auto res =
        super::vkCreateRenderPass(device, pCreateInfo, pAllocator, pRenderPass);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pRenderPass[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateRenderPass2(VkDevice device,
                               const VkRenderPassCreateInfo2* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkRenderPass* pRenderPass) override {
    auto res = super::vkCreateRenderPass2(device, pCreateInfo, pAllocator,
                                          pRenderPass);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pRenderPass[0]);
    pl->set_create_info2(pCreateInfo);
    return res;
  }

  VkResult vkCreateCommandPool(VkDevice device,
                               const VkCommandPoolCreateInfo* pCreateInfo,
                               const VkAllocationCallbacks* pAllocator,
                               VkCommandPool* pCommandPool) override {
    auto res = super::vkCreateCommandPool(device, pCreateInfo, pAllocator,
                                          pCommandPool);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pCommandPool[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }
  VkResult vkAllocateCommandBuffers(
      VkDevice device,
      const VkCommandBufferAllocateInfo* pAllocateInfo,
      VkCommandBuffer* pCommandBuffers) override {
    auto res =
        super::vkAllocateCommandBuffers(device, pAllocateInfo, pCommandBuffers);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (uint32_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
      auto pl = this->updater_.cast_from_vk(pCommandBuffers[i]);
      pl->set_allocate_info(pAllocateInfo, i);
    }
    return res;
  }
  VkResult vkCreateSamplerYcbcrConversion(
      VkDevice device,
      const VkSamplerYcbcrConversionCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkSamplerYcbcrConversion* pYcbcrConversion) override {
    auto res = super::vkCreateSamplerYcbcrConversion(
        device, pCreateInfo, pAllocator, pYcbcrConversion);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pYcbcrConversion[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateDescriptorUpdateTemplate(
      VkDevice device,
      const VkDescriptorUpdateTemplateCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorUpdateTemplate* pDescriptorUpdateTemplate) override {
    auto res = super::vkCreateDescriptorUpdateTemplate(
        device, pCreateInfo, pAllocator, pDescriptorUpdateTemplate);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pDescriptorUpdateTemplate[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }
  VkResult vkCreateWin32SurfaceKHR(
      VkInstance instance,
      const VkWin32SurfaceCreateInfoKHR* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkSurfaceKHR* pSurface) override {
    auto res = super::vkCreateWin32SurfaceKHR(instance, pCreateInfo, pAllocator,
                                              pSurface);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pSurface[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateSwapchainKHR(VkDevice device,
                                const VkSwapchainCreateInfoKHR* pCreateInfo,
                                const VkAllocationCallbacks* pAllocator,
                                VkSwapchainKHR* pSwapchain) override {
    auto res = super::vkCreateSwapchainKHR(device, pCreateInfo, pAllocator,
                                           pSwapchain);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pSwapchain[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkGetSwapchainImagesKHR(VkDevice device,
                                   VkSwapchainKHR swapchain,
                                   uint32_t* pSwapchainImageCount,
                                   VkImage* pSwapchainImages) override {
    auto res = super::vkGetSwapchainImagesKHR(
        device, swapchain, pSwapchainImageCount, pSwapchainImages);
    if (res != VK_SUCCESS) {
      return res;
    }
    if (pSwapchainImages != nullptr) {
      for (uint32_t i = 0; i < *pSwapchainImageCount; ++i) {
        auto pl = this->updater_.cast_from_vk(pSwapchainImages[i]);
        pl->set_swapchain_info(swapchain, i);
      }
    }
    return res;
  }

 protected:
};
}  // namespace gapid2