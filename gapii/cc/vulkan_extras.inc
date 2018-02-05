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
// public:

PFN_vkVoidFunction SpyOverride_vkGetInstanceProcAddr(VkInstance instance,
                                                     const char* pName);
PFN_vkVoidFunction SpyOverride_vkGetDeviceProcAddr(VkDevice device,
                                                   const char* pName);
uint32_t SpyOverride_vkEnumerateInstanceExtensionProperties(
    const char* pLayerName, uint32_t* pCount,
    VkExtensionProperties* pProperties);
uint32_t SpyOverride_vkEnumerateInstanceLayerProperties(
    uint32_t* pPropertyCount, VkLayerProperties* pProperties);
uint32_t SpyOverride_vkCreateInstance(VkInstanceCreateInfo* pCreateInfo,
                                      VkAllocationCallbacks* pAllocator,
                                      VkInstance* pInstance);
void SpyOverride_vkDestroyInstance(VkInstance instance,
                                   VkAllocationCallbacks* pAllocator);
uint32_t SpyOverride_vkCreateDevice(VkPhysicalDevice physicalDevice,
                                    VkDeviceCreateInfo* pCreateInfo,
                                    VkAllocationCallbacks* pAllocator,
                                    VkDevice* pDevice);
void SpyOverride_vkDestroyDevice(VkDevice device,
                                 VkAllocationCallbacks* pAllocator);
uint32_t SpyOverride_vkEnumerateDeviceLayerProperties(
    VkPhysicalDevice dev, uint32_t* pCount, VkLayerProperties* pProperties);
uint32_t SpyOverride_vkEnumerateDeviceExtensionProperties(
    VkPhysicalDevice dev, const char* pLayerName, uint32_t* pCount,
    VkExtensionProperties* pProperties);
void SpyOverride_vkGetDeviceQueue(VkDevice device, uint32_t queueFamilyIndex,
                                  uint32_t queueIndex, VkQueue* pQueue);
uint32_t SpyOverride_vkAllocateCommandBuffers(
    VkDevice device, VkCommandBufferAllocateInfo* pAllocateInfo,
    VkCommandBuffer* pCommandBuffers);
uint32_t SpyOverride_vkCreateBuffer(VkDevice device,
                                    VkBufferCreateInfo* pCreateInfo,
                                    VkAllocationCallbacks* pAllocator,
                                    VkBuffer* pBuffer);
uint32_t SpyOverride_vkCreateImage(VkDevice device,
                                   VkImageCreateInfo* pCreateInfo,
                                   VkAllocationCallbacks* pAllocator,
                                   VkImage* pBuffer);
uint32_t SpyOverride_vkAllocateMemory(VkDevice device,
                                      VkMemoryAllocateInfo* pAllocateInfo,
                                      VkAllocationCallbacks* pAllocator,
                                      VkDeviceMemory* pMemory);
uint32_t SpyOverride_vkCreateSwapchainKHR(VkDevice device,
                                          VkSwapchainCreateInfoKHR* pCreateInfo,
                                          VkAllocationCallbacks* pAllocator,
                                          VkSwapchainKHR* pImage);
uint32_t SpyOverride_vkDebugMarkerSetObjectTagEXT(
    VkDevice device, VkDebugMarkerObjectTagInfoEXT* pTagInfo) {
  return VkResult::VK_SUCCESS;
}
uint32_t SpyOverride_RecreateDebugMarkerSetObjectTagEXT(
    VkDevice device, VkDebugMarkerObjectTagInfoEXT* pTagInfo) {
  return VkResult::VK_SUCCESS;
}

uint32_t SpyOverride_vkDebugMarkerSetObjectNameEXT(
    VkDevice device, VkDebugMarkerObjectNameInfoEXT* pNameInfo) {
  return VkResult::VK_SUCCESS;
}
uint32_t SpyOverride_RecreateDebugMarkerSetObjectNameEXT(
    VkDevice device, VkDebugMarkerObjectNameInfoEXT* pNameInfo) {
  return VkResult::VK_SUCCESS;
}

void SpyOverride_vkCmdDebugMarkerBeginEXT(
    VkCommandBuffer commandBuffer, VkDebugMarkerMarkerInfoEXT* pMarkerInfo) {}
void SpyOverride_vkCmdDebugMarkerEndEXT(VkCommandBuffer commandBuffer) {}
void SpyOverride_vkCmdDebugMarkerInsertEXT(
    VkCommandBuffer commandBuffer, VkDebugMarkerMarkerInfoEXT* pMarkerInfo) {}

void SpyOverride_RecreateInstance(const VkInstanceCreateInfo*, VkInstance*) {}
void SpyOverride_RecreateState() {}
void SpyOverride_RecreatePhysicalDevices(VkInstance, uint32_t*,
                                         VkPhysicalDevice*,
                                         VkPhysicalDeviceProperties*) {}
void SpyOverride_RecreateDevice(VkPhysicalDevice, const VkDeviceCreateInfo*,
                                VkDevice*) {}
void SpyOverride_RecreateDeviceMemory(VkDevice, VkMemoryAllocateInfo*,
                                      VkDeviceSize, VkDeviceSize, void**,
                                      VkDeviceMemory*) {}
void SpyOverride_RecreateQueue(VkDevice, uint32_t, uint32_t, VkQueue*) {}
void SpyOverride_RecreateVkCmdBindPipeline(VkCommandBuffer, uint32_t,
                                           VkPipeline) {}
void SpyOverride_RecreateAndBeginCommandBuffer(
    VkDevice, const VkCommandBufferAllocateInfo*,
    const VkCommandBufferBeginInfo*, VkCommandBuffer*) {}
void SpyOverride_RecreateEndCommandBuffer(VkCommandBuffer) {}
void SpyOverride_RecreateSemaphore(VkDevice, const VkSemaphoreCreateInfo*,
                                   VkBool32, VkSemaphore*) {}
void SpyOverride_RecreateFence(VkDevice, const VkFenceCreateInfo*, VkFence*) {}
void SpyOverride_RecreateEvent(VkDevice, const VkEventCreateInfo*, VkBool32,
                               VkEvent*) {}

void SpyOverride_RecreatePipelineCache(VkDevice,
                                       const VkPipelineCacheCreateInfo*,
                                       VkPipelineCache*) {}
void SpyOverride_RecreateDescriptorSetLayout(
    VkDevice, const VkDescriptorSetLayoutCreateInfo*, VkPipelineCache*) {}
void SpyOverride_RecreatePipelineLayout(VkDevice,
                                        const VkPipelineLayoutCreateInfo*,
                                        VkPipelineLayout*) {}
void SpyOverride_RecreateRenderPass(VkDevice, const VkRenderPassCreateInfo*,
                                    VkRenderPass*) {}
void SpyOverride_RecreateShaderModule(VkDevice, const VkShaderModuleCreateInfo*,
                                      VkShaderModule*) {}
void SpyOverride_RecreateDestroyShaderModule(VkDevice, VkShaderModule) {}
void SpyOverride_RecreateDestroyRenderPass(VkDevice, VkRenderPass) {}
void SpyOverride_RecreateDescriptorPool(VkDevice,
                                        const VkDescriptorPoolCreateInfo*,
                                        VkDescriptorPool*) {}
void SpyOverride_RecreateSwapchain(VkDevice, const VkSwapchainCreateInfoKHR*,
                                   VkImage*, const uint32_t*, const VkQueue*,
                                   VkSwapchainKHR*) {}
void SpyOverride_RecreateImage(
    VkDevice, const VkImageCreateInfo*, VkImage*,
    VkMemoryRequirements* pMemoryRequirements,
    uint32_t sparseMemoryRequirementCount,
    VkSparseImageMemoryRequirements* pSparseMemoryRequirements,
    VkMemoryDedicatedRequirementsKHR* pDedicatedRequirements) {}
void SpyOverride_RecreateImageMemoryBindings(VkDevice, VkImage, VkDeviceMemory,
                                         VkDeviceSize offset,
                                         uint32_t opaqueBindCount,
                                         VkSparseMemoryBind* pOpaqueBinds,
                                         uint32_t imageBindCount,
                                         VkSparseImageMemoryBind* pImageBinds) {}
void SpyOverride_RecreateImageSubrangeData(VkDevice, VkImage,
                                           uint32_t /*VkImageLayout*/,
                                           VkImageSubresourceRange* range,
                                           uint32_t hostMemoryIndex, VkQueue,
                                           VkDeviceSize resourceOffset,
                                           VkDeviceSize dataSize, void* data) {}
void SpyOverride_RecreateSparseImageBindData(
    VkDevice, VkImage, uint32_t /*VkImageLayout*/, VkSparseImageMemoryBind*,
    uint32_t hostMemoryIndex, VkQueue, VkDeviceSize dataSize, void* data) {}
void SpyOverride_RecreateImageView(VkDevice, const VkImageViewCreateInfo*,
                                   VkImageView*) {}
void SpyOverride_RecreateSampler(VkDevice, const VkSamplerCreateInfo*,
                                 VkSampler*) {}
void SpyOverride_RecreateFramebuffer(VkDevice, const VkFramebufferCreateInfo*,
                                     VkFramebuffer*) {}
void SpyOverride_RecreateDescriptorSet(VkDevice,
                                       const VkDescriptorSetAllocateInfo*,
                                       uint32_t, const VkWriteDescriptorSet*,
                                       VkDescriptorSet*) {}
void SpyOverride_RecreateGraphicsPipeline(VkDevice, VkPipelineCache,
                                          const VkGraphicsPipelineCreateInfo*,
                                          VkPipeline*) {}
void SpyOverride_RecreateComputePipeline(VkDevice, VkPipelineCache,
                                         const VkComputePipelineCreateInfo*,
                                         VkPipeline*) {}
void SpyOverride_RecreateBuffer(VkDevice, VkBufferCreateInfo*, VkBuffer*,
                                VkMemoryRequirements*,
                                VkMemoryDedicatedRequirementsKHR*) {}
void SpyOverride_RecreateBufferMemoryBindings(VkDevice, VkBuffer,
                                              VkDeviceMemory,
                                              VkDeviceSize offset,
                                              uint32_t sparseBindCount,
                                              VkSparseMemoryBind* binds) {}
void SpyOverride_RecreateBufferData(VkDevice, VkBuffer, VkDeviceSize,
                                    VkDeviceSize,
                                    uint32_t hostBufferMemoryIndex, VkQueue,
                                    void* data) {}
void SpyOverride_RecreateSparseBufferData(VkDevice, VkBuffer, VkDeviceSize,
                                          VkDeviceSize, void*) {}
void SpyOverride_RecreateBufferView(VkDevice, const VkBufferViewCreateInfo*,
                                    VkBufferView*) {}
void SpyOverride_RecreatePhysicalDeviceProperties(
    VkPhysicalDevice, uint32_t*, VkQueueFamilyProperties*,
    VkPhysicalDeviceMemoryProperties*) {}
void SpyOverride_RecreateQueryPool(VkDevice, const VkQueryPoolCreateInfo*,
                                   uint32_t*, VkQueryPool*) {}

void SpyOverride_RecreateXCBSurfaceKHR(VkDevice,
                                       const VkXcbSurfaceCreateInfoKHR*,
                                       VkSurfaceKHR*) {}
void SpyOverride_RecreateAndroidSurfaceKHR(VkDevice,
                                           const VkAndroidSurfaceCreateInfoKHR*,
                                           VkSurfaceKHR*) {}
void SpyOverride_RecreateWin32SurfaceKHR(VkDevice,
                                         const VkWin32SurfaceCreateInfoKHR*,
                                         VkSurfaceKHR*) {}
void SpyOverride_RecreateXlibSurfaceKHR(VkDevice,
                                        const VkXlibSurfaceCreateInfoKHR*,
                                        VkSurfaceKHR*) {}
void SpyOverride_RecreateWaylandSurfaceKHR(VkDevice,
                                           const VkWaylandSurfaceCreateInfoKHR*,
                                           VkSurfaceKHR*) {}
void SpyOverride_RecreateMirSurfaceKHR(VkDevice,
                                       const VkMirSurfaceCreateInfoKHR*,
                                       VkSurfaceKHR*) {}

void EnumerateVulkanResources(CallObserver* observer);

static uint32_t CreateImageAndGetMemoryRequirements(VkDevice,
                                                    VkImageCreateInfo*,
                                                    VkAllocationCallbacks*,
                                                    VkImage*);
static uint32_t CreateBufferAndGetMemoryRequirements(VkDevice,
                                                     VkBufferCreateInfo*,
                                                     VkAllocationCallbacks*,
                                                     VkBuffer*);

static uint32_t EnumeratePhysicalDevicesAndCacheProperties(
    VkInstance, uint32_t* pPhysicalDeviceCount,
    VkPhysicalDevice* pPhysicalDevices);

bool m_coherent_memory_tracking_enabled = false;

#if TARGET_OS == GAPID_OS_ANDROID
bool m_should_unset_debug_vulkan_layers = true;
#endif // TARGET_OS == GAPID_OS_ANDROID

void SpyOverride_cacheImageSparseMemoryRequirements(
    VkDevice device, VkImage image, uint32_t count,
    VkSparseImageMemoryRequirements* pSparseMemoryRequirements);

void prepareGPUBuffers(PackEncoder* group, std::unordered_set<uint32_t>* gpu_pools);
