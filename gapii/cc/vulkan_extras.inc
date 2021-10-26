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

PFN_vkVoidFunction SpyOverride_vkGetInstanceProcAddr(CallObserver*, VkInstance instance,
                                                     const char* pName);
PFN_vkVoidFunction SpyOverride_vkGetDeviceProcAddr(CallObserver*, VkDevice device,
                                                   const char* pName);
uint32_t SpyOverride_vkEnumerateInstanceExtensionProperties(
    CallObserver*, const char* pLayerName, uint32_t* pCount,
    VkExtensionProperties* pProperties);
uint32_t SpyOverride_vkEnumerateInstanceLayerProperties(
    CallObserver*, uint32_t* pPropertyCount, VkLayerProperties* pProperties);
uint32_t SpyOverride_vkCreateInstance(CallObserver*, const VkInstanceCreateInfo* pCreateInfo,
                                      const VkAllocationCallbacks* pAllocator,
                                      VkInstance* pInstance);
void SpyOverride_vkDestroyInstance(CallObserver*, VkInstance instance,
                                   const VkAllocationCallbacks* pAllocator);
uint32_t SpyOverride_vkCreateDevice(CallObserver*, VkPhysicalDevice physicalDevice,
                                    const VkDeviceCreateInfo* pCreateInfo,
                                    const VkAllocationCallbacks* pAllocator,
                                    VkDevice* pDevice);
void SpyOverride_vkDestroyDevice(CallObserver*, VkDevice device,
                                 const VkAllocationCallbacks* pAllocator);
uint32_t SpyOverride_vkEnumerateDeviceLayerProperties(
    CallObserver*, VkPhysicalDevice dev, uint32_t* pCount, VkLayerProperties* pProperties);
uint32_t SpyOverride_vkEnumerateDeviceExtensionProperties(
    CallObserver*, VkPhysicalDevice dev, const char* pLayerName, uint32_t* pCount,
    VkExtensionProperties* pProperties);
void SpyOverride_vkGetPhysicalDeviceFeatures2(
    CallObserver*, VkPhysicalDevice physicalDevice, VkPhysicalDeviceFeatures2* pFeatures);
void SpyOverride_vkGetPhysicalDeviceFeatures2KHR(
    CallObserver*, VkPhysicalDevice physicalDevice, VkPhysicalDeviceFeatures2KHR* pFeatures);
void SpyOverride_vkGetPhysicalDeviceQueueFamilyProperties(
    CallObserver*, VkPhysicalDevice physicalDevice,
    uint32_t* pQueueFamilyPropertyCount, VkQueueFamilyProperties* pQueueFamilyProperties);
void SpyOverride_vkGetPhysicalDeviceQueueFamilyProperties2(
    CallObserver*, VkPhysicalDevice physicalDevice,
    uint32_t* pQueueFamilyPropertyCount, VkQueueFamilyProperties2* pQueueFamilyProperties);
void SpyOverride_vkGetPhysicalDeviceQueueFamilyProperties2KHR(
    CallObserver*, VkPhysicalDevice physicalDevice,
    uint32_t* pQueueFamilyPropertyCount, VkQueueFamilyProperties2KHR* pQueueFamilyProperties);
uint32_t SpyOverride_vkEnumeratePhysicalDeviceGroups(
    CallObserver*, VkInstance instance, uint32_t* pPhysicalDeviceGroupCount,
    VkPhysicalDeviceGroupProperties* pPhysicalDeviceGroupProperties);
uint32_t SpyOverride_vkEnumeratePhysicalDeviceGroupsKHR(
    CallObserver*, VkInstance instance, uint32_t* pPhysicalDeviceGroupCount,
    VkPhysicalDeviceGroupProperties* pPhysicalDeviceGroupProperties);

void SpyOverride_vkGetDeviceQueue(CallObserver*, VkDevice device, uint32_t queueFamilyIndex,
                                  uint32_t queueIndex, VkQueue* pQueue);
uint32_t SpyOverride_vkAllocateCommandBuffers(
    CallObserver*, VkDevice device, VkCommandBufferAllocateInfo* pAllocateInfo,
    VkCommandBuffer* pCommandBuffers);
uint32_t SpyOverride_vkCreateBuffer(CallObserver*, VkDevice device,
                                    const VkBufferCreateInfo* pCreateInfo,
                                    const VkAllocationCallbacks* pAllocator,
                                    VkBuffer* pBuffer);
uint32_t SpyOverride_vkCreateImage(CallObserver*, VkDevice device,
                                   const VkImageCreateInfo* pCreateInfo,
                                   const VkAllocationCallbacks* pAllocator,
                                   VkImage* pBuffer);
uint32_t SpyOverride_vkAllocateMemory(CallObserver*, VkDevice device,
                                      const VkMemoryAllocateInfo* pAllocateInfo,
                                      const VkAllocationCallbacks* pAllocator,
                                      VkDeviceMemory* pMemory);
uint32_t SpyOverride_vkCreateSwapchainKHR(CallObserver*, VkDevice device,
                                          const VkSwapchainCreateInfoKHR* pCreateInfo,
                                          const VkAllocationCallbacks* pAllocator,
                                          VkSwapchainKHR* pImage);
uint32_t SpyOverride_vkDebugMarkerSetObjectTagEXT(
    CallObserver*, VkDevice device, const VkDebugMarkerObjectTagInfoEXT* pTagInfo) {
  return VkResult::VK_SUCCESS;
}
uint32_t SpyOverride_RecreateDebugMarkerSetObjectTagEXT(
    CallObserver*, VkDevice device, const VkDebugMarkerObjectTagInfoEXT* pTagInfo) {
  return VkResult::VK_SUCCESS;
}

uint32_t SpyOverride_vkDebugMarkerSetObjectNameEXT(
    CallObserver*, VkDevice device, const VkDebugMarkerObjectNameInfoEXT* pNameInfo) {
  return VkResult::VK_SUCCESS;
}

void SpyOverride_vkCmdDebugMarkerBeginEXT(
    CallObserver*, VkCommandBuffer commandBuffer, const VkDebugMarkerMarkerInfoEXT* pMarkerInfo) {}
void SpyOverride_vkCmdDebugMarkerEndEXT(CallObserver*, VkCommandBuffer commandBuffer) {}
void SpyOverride_vkCmdDebugMarkerInsertEXT(
    CallObserver*, VkCommandBuffer commandBuffer, const VkDebugMarkerMarkerInfoEXT* pMarkerInfo) {}


void SpyOverride_vkCmdBeginDebugUtilsLabelEXT(
    CallObserver*, VkCommandBuffer commandBuffer, const VkDebugUtilsLabelEXT* pMarkerInfo) {}
void SpyOverride_vkCmdEndDebugUtilsLabelEXT(CallObserver*, VkCommandBuffer commandBuffer) {}
void SpyOverride_vkCmdInsertDebugUtilsLabelEXT(
    CallObserver*, VkCommandBuffer commandBuffer, const VkDebugUtilsLabelEXT* pMarkerInfo) {}


void SpyOverride_vkQueueBeginDebugUtilsLabelEXT(
    CallObserver*, VkQueue queue, const VkDebugUtilsLabelEXT* pMarkerInfo) {}
void SpyOverride_vkQueueEndDebugUtilsLabelEXT(CallObserver*, VkQueue queue) {}
void SpyOverride_vkQueueInsertDebugUtilsLabelEXT(
    CallObserver*, VkQueue queue, const VkDebugUtilsLabelEXT* pMarkerInfo) {}


uint32_t SpyOverride_vkSetDebugUtilsObjectNameEXT(
    CallObserver*, VkDevice device, const VkDebugUtilsObjectNameInfoEXT* pNameInfo) {
  return VkResult::VK_SUCCESS;
}
uint32_t SpyOverride_vkSetDebugUtilsObjectTagEXT(
    CallObserver*, VkDevice device, const VkDebugUtilsObjectTagInfoEXT* pTagInfo) {
  return VkResult::VK_SUCCESS;
}

uint32_t SpyOverride_vkCreateDebugUtilsMessengerEXT(
    CallObserver*,
    VkInstance                                instance,
    const VkDebugUtilsMessengerCreateInfoEXT* pCreateInfo,
    const VkAllocationCallbacks*              pAllocator,
    VkDebugUtilsMessengerEXT*                 pMessenger) {
        return  VkResult::VK_SUCCESS;
}
void SpyOverride_vkDestroyDebugUtilsMessengerEXT(
    CallObserver*,
    VkInstance                   instance,
    VkDebugUtilsMessengerEXT     messenger,
    const VkAllocationCallbacks* pAllocator) {
}
void SpyOverride_vkSubmitDebugUtilsMessageEXT(
    CallObserver*,
    VkInstance                                  instance,
    uint32_t                                    messageSeverity,
    VkDebugUtilsMessageTypeFlagsEXT             messageTypes,
    const VkDebugUtilsMessengerCallbackDataEXT* pCallbackData) {
}

void SpyOverride_vkCmdWriteBufferMarkerAMD(
  CallObserver*,
  VkCommandBuffer commandBuffer, uint32_t pipelineStage,
  VkBuffer dstBuffer, VkDeviceSize dstOffset, uint32_t marker) {}

bool m_coherent_memory_tracking_enabled = false;

void SpyOverride_cacheImageSparseMemoryRequirements(
    CallObserver*,
    VkDevice device, VkImage image, uint32_t count,
    VkSparseImageMemoryRequirements* pSparseMemoryRequirements);

void serializeGPUBuffers(StateSerializer* serializer);

void walkImageSubRng(
    gapil::Ref<ImageObject> img, VkImageSubresourceRange rng,
    std::function<void(uint32_t aspect_bit, uint32_t layer, uint32_t level)> f);

void parseShaderModule(
    StageData* stage,
    gapil::Ref<DescriptorInfo>& descriptors);

// AGI implements VK_ANDROID_frame_boundary itself, the underlying ICD
// is not expected to provide it. Hence, override the spy to avoid calling into
// the next layer/ICD implementation of that function.
void SpyOverride_vkFrameBoundaryANDROID(CallObserver*,
                                        VkDevice device,
                                        VkSemaphore semaphore,
                                        VkImage image) {}
