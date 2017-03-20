/*
 * Copyright (C) 2017 Google Inc.
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

// Note this file is included in context in vulkan_spy.h:
//
// namespace gapii {
//
// class VulkanSpy {
// protected:

PFN_vkVoidFunction SpyOverride_vkGetInstanceProcAddr(VkInstance instance, const char* pName);
PFN_vkVoidFunction SpyOverride_vkGetDeviceProcAddr(VkDevice device, const char* pName);
uint32_t SpyOverride_vkEnumeratePhysicalDevices(VkInstance instance, uint32_t *pPhysicalDeviceCount, VkPhysicalDevice *pPhysicalDevices);
uint32_t SpyOverride_vkEnumerateInstanceExtensionProperties(const char *pLayerName, uint32_t *pCount, VkExtensionProperties *pProperties);
uint32_t SpyOverride_vkEnumerateInstanceLayerProperties(uint32_t* pPropertyCount, VkLayerProperties* pProperties);
uint32_t SpyOverride_vkCreateInstance(VkInstanceCreateInfo *pCreateInfo, VkAllocationCallbacks *pAllocator, VkInstance *pInstance);
void SpyOverride_vkDestroyInstance(VkInstance instance, VkAllocationCallbacks* pAllocator);
uint32_t SpyOverride_vkCreateDevice(VkPhysicalDevice physicalDevice, VkDeviceCreateInfo* pCreateInfo, VkAllocationCallbacks* pAllocator, VkDevice* pDevice);
void SpyOverride_vkDestroyDevice(VkDevice device, VkAllocationCallbacks* pAllocator);
uint32_t SpyOverride_vkEnumerateDeviceLayerProperties(VkPhysicalDevice dev, uint32_t *pCount, VkLayerProperties *pProperties);
uint32_t SpyOverride_vkEnumerateDeviceExtensionProperties(VkPhysicalDevice dev, const char *pLayerName, uint32_t *pCount, VkExtensionProperties *pProperties);
void SpyOverride_vkGetDeviceQueue(VkDevice device, uint32_t queueFamilyIndex, uint32_t queueIndex, VkQueue* pQueue);
uint32_t SpyOverride_vkAllocateCommandBuffers(VkDevice device, VkCommandBufferAllocateInfo* pAllocateInfo, VkCommandBuffer* pCommandBuffers);
uint32_t SpyOverride_vkCreateBuffer(VkDevice device, VkBufferCreateInfo* pCreateInfo, VkAllocationCallbacks* pAllocator, VkBuffer* pBuffer);
uint32_t SpyOverride_vkCreateImage(VkDevice device, VkImageCreateInfo* pCreateInfo, VkAllocationCallbacks* pAllocator, VkImage* pBuffer);

void SpyOverride_RecreateInstance(const VkInstanceCreateInfo*, VkInstance*) {}
void SpyOverride_RecreatePhysicalDevices(VkInstance, uint32_t*, VkPhysicalDevice*) {}
void SpyOverride_RecreateDevice(VkPhysicalDevice, const VkDeviceCreateInfo*, VkDevice*) {}
void SpyOverride_RecreateDeviceMemory(VkDevice,VkMemoryAllocateInfo*,
    VkDeviceSize, VkDeviceSize, void**, VkDeviceMemory*) {}
void SpyOverride_RecreateQueue(VkDevice, uint32_t, uint32_t, VkQueue*) {}
void SpyOverride_RecreateVkCmdBindPipeline(VkCommandBuffer, uint32_t, VkPipeline) {}
void SpyOverride_RecreateVkCommandBuffer(VkDevice, const VkCommandBufferAllocateInfo*, const VkCommandBufferBeginInfo*, VkCommandBuffer*) {}
void SpyOverride_RecreateVkEndCommandBuffer(VkCommandBuffer) {}
void SpyOverride_RecreateSemaphore(VkDevice, const VkSemaphoreCreateInfo*, uint32_t, VkSemaphore*) {}
void SpyOverride_RecreateFence(VkDevice, const VkFenceCreateInfo*, VkFence*) {}
void SpyOverride_RecreateCommandPool(VkDevice, const VkCommandPoolCreateInfo*, VkCommandPool*) {}
void SpyOverride_RecreatePipelineCache(VkDevice, const VkPipelineCacheCreateInfo*, VkPipelineCache*) {}
void SpyOverride_RecreateDescriptorSetLayout(VkDevice, const VkDescriptorSetLayoutCreateInfo*, VkPipelineCache*) {}
void SpyOverride_RecreatePipelineLayout(VkDevice, const VkPipelineLayoutCreateInfo*, VkPipelineLayout*) {}
void SpyOverride_RecreateRenderPass(VkDevice, const VkRenderPassCreateInfo*, VkRenderPass*) {}
void SpyOverride_RecreateShaderModule(VkDevice, const VkShaderModuleCreateInfo*, VkShaderModule*) {}
void SpyOverride_RecreateDescriptorPool(VkDevice, const VkDescriptorPoolCreateInfo*, VkDescriptorPool*) {}
void SpyOverride_RecreateSwapchain(VkDevice, const VkSwapchainCreateInfoKHR*, VkImage*, const uint32_t*, const VkQueue*, VkSwapchainKHR*) {}
void SpyOverride_RecreateImage(VkDevice, const VkImageCreateInfo*, VkImage*) {}
void SpyOverride_RecreateBindAndFillImageMemory(VkDevice, VkImage, uint32_t /*VkImageLayout*/, VkDeviceMemory, uint32_t hostMemoryIndex, VkQueue, VkDeviceSize offset, VkDeviceSize dataSize, void* data) {}
void SpyOverride_RecreateImageView(VkDevice, const VkImageViewCreateInfo*, VkImageView*) {}
void SpyOverride_RecreateSampler(VkDevice, const VkSamplerCreateInfo*, VkSampler*) {}
void SpyOverride_RecreateFramebuffer(VkDevice, const VkFramebufferCreateInfo*, VkFramebuffer*) {}
void SpyOverride_RecreateDescriptorSet(VkDevice, const VkDescriptorSetAllocateInfo*, uint32_t, const VkWriteDescriptorSet*, VkDescriptorSet*) {}
void SpyOverride_RecreateGraphicsPipeline(VkDevice, VkPipelineCache, const VkGraphicsPipelineCreateInfo*, VkPipeline*) {}
void SpyOverride_RecreateComputePipeline(VkDevice, VkPipelineCache, const VkComputePipelineCreateInfo*, VkPipeline*) {}
void SpyOverride_RecreateBuffer(VkDevice, VkBufferCreateInfo*, VkBuffer*) {}
void SpyOverride_RecreateBindAndFillBufferMemory(VkDevice, VkBuffer, VkDeviceMemory, uint32_t hostBufferMemoryIndex, VkQueue, VkDeviceSize offset, void* data) {}
void SpyOverride_RecreateBufferView(VkDevice, const VkBufferViewCreateInfo*, VkBufferView*) {}
void SpyOverride_RecreatePhysicalDeviceProperties(VkPhysicalDevice, uint32_t*, VkQueueFamilyProperties*, VkPhysicalDeviceMemoryProperties*) {}
void SpyOverride_RecreateQueryPool(VkDevice, const VkQueryPoolCreateInfo*, uint32_t*, VkQueryPool*) {}

void SpyOverride_RecreateUpdateBuffer(VkCommandBuffer, VkBuffer, VkDeviceSize, VkDeviceSize, const void*) {}
void SpyOverride_RecreateCmdPipelineBarrier(VkCommandBuffer, VkPipelineStageFlags, VkPipelineStageFlags,
                                            VkDependencyFlags, uint32_t, const VkMemoryBarrier*, uint32_t,
                                            const VkBufferMemoryBarrier*, uint32_t, const VkImageMemoryBarrier*) {}
void SpyOverride_RecreateCmdCopyBuffer(VkCommandBuffer, VkBuffer, VkBuffer, uint32_t, const VkBufferCopy*) {}
void SpyOverride_RecreateCmdResolveImage(VkCommandBuffer, VkImage, uint32_t, VkImage, uint32_t, uint32_t, const VkImageResolve*) {}
void SpyOverride_RecreateCmdBeginRenderPass(VkCommandBuffer, const VkRenderPassBeginInfo*, uint32_t) {}
void SpyOverride_RecreateCmdBindPipeline(VkCommandBuffer, uint32_t, VkPipeline) {}
void SpyOverride_RecreateCmdBindDescriptorSets(VkCommandBuffer, uint32_t,
                                               VkPipelineLayout, uint32_t,
                                               uint32_t, const VkDescriptorSet*,
                                               uint32_t, const uint32_t*) {}
void SpyOverride_RecreateBindVertexBuffers(VkCommandBuffer, uint32_t, uint32_t, const VkBuffer*, const VkDeviceSize*) {}
void SpyOverride_RecreateCmdBindIndexBuffer(VkCommandBuffer, VkBuffer, uint64_t, uint32_t) {}
void SpyOverride_RecreateCmdBlitImage(VkCommandBuffer, VkImage, uint32_t/*srcImageLayout*/, VkImage, uint32_t/*dstImageLayout*/, uint32_t/*regionCount*/, const VkImageBlit*, uint32_t/*filter*/) {}
void SpyOverride_RecreateEndRenderPass(VkCommandBuffer) {}
void SpyOverride_RecreateCmdDrawIndexed(VkCommandBuffer, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdCopyBufferToImage(VkCommandBuffer, VkBuffer, VkImage, uint32_t, uint32_t, const VkBufferImageCopy*) {}
void SpyOverride_RecreateCmdCopyImageToBuffer(VkCommandBuffer, VkImage, uint32_t/*srcImageLayout*/, VkBuffer, uint32_t, const VkBufferImageCopy*) {}
void SpyOverride_RecreateCmdCopyImage(VkCommandBuffer, VkImage, uint32_t, VkImage, uint32_t, uint32_t, const VkImageCopy*) {}
void SpyOverride_RecreateCmdDraw(VkCommandBuffer, uint32_t, uint32_t, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdDispatch(VkCommandBuffer, uint32_t, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdDispatchIndirect(VkCommandBuffer, VkBuffer, VkDeviceSize) {}
void SpyOverride_RecreateCmdSetScissor(VkCommandBuffer, uint32_t, uint32_t, const VkRect2D*) {}
void SpyOverride_RecreateCmdSetViewport(VkCommandBuffer, uint32_t, uint32_t, const VkViewport*) {}
void SpyOverride_RecreateCmdSetDepthBias(VkCommandBuffer, float, float, float) {}
void SpyOverride_RecreateCmdSetLineWidth(VkCommandBuffer, float) {}
void SpyOverride_RecreateCmdPushConstants(VkCommandBuffer, VkPipelineLayout, uint32_t, uint32_t, uint32_t, const void*) {}
void SpyOverride_RecreateCmdBeginQuery(VkCommandBuffer, VkQueryPool, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdEndQuery(VkCommandBuffer, VkQueryPool, uint32_t) {}
void SpyOverride_RecreateCmdResetQueryPool(VkCommandBuffer, VkQueryPool, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdCopyQueryPoolResults(VkCommandBuffer, VkQueryPool, uint32_t, uint32_t, VkBuffer, VkDeviceSize, VkDeviceSize, VkQueryResultFlags) {}
void SpyOverride_RecreateCmdSetBlendConstants(VkCommandBuffer, float [4]) {}

void SpyOverride_RecreateCmdDrawIndirect(VkCommandBuffer, VkBuffer, VkDeviceSize, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdDrawIndexedIndirect(VkCommandBuffer, VkBuffer, VkDeviceSize, uint32_t, uint32_t) {}
void SpyOverride_RecreateCmdClearAttachments(VkCommandBuffer, uint32_t, const VkClearAttachment*, uint32_t, const VkClearRect*) {}
void SpyOverride_RecreateCmdClearColorImage(VkCommandBuffer, VkImage, uint32_t /*VkImageLayout*/, const VkClearColorValue*, uint32_t, const VkImageSubresourceRange*) {}
void SpyOverride_RecreateCmdClearDepthStencilImage(VkCommandBuffer, VkImage, uint32_t /*VkImageLayout*/, const VkClearDepthStencilValue*, uint32_t, const VkImageSubresourceRange*) {}
void SpyOverride_RecreateCmdExecuteCommands(VkCommandBuffer, uint32_t, const VkCommandBuffer*) {}

void SpyOverride_RecreateXCBSurfaceKHR(VkDevice, const VkXcbSurfaceCreateInfoKHR*, VkSurfaceKHR*) {}
void SpyOverride_RecreateAndroidSurfaceKHR(VkDevice, const VkAndroidSurfaceCreateInfoKHR*, VkSurfaceKHR*) {}

void EnumerateVulkanResources(CallObserver* observer);
