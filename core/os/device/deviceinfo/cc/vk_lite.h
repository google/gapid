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

#ifndef GAPID_CORE_OS_DEVICEINFO_VK_LITE
#define GAPID_CORE_OS_DEVICEINFO_VK_LITE

#include "core/cc/static_array.h"
#include "core/cc/vulkan_ptr_types.h"

// Enums
typedef enum VkStructureType {
  VK_STRUCTURE_TYPE_APPLICATION_INFO = 0,
  VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO = 1,
} VkStructureType;

typedef enum VkResult {
  VK_SUCCESS = 0,
  VK_NOT_READY = 1,
  VK_TIMEOUT = 2,
  VK_EVENT_SET = 3,
  VK_EVENT_RESET = 4,
  VK_INCOMPLETE = 5,
  VK_ERROR_OUT_OF_HOST_MEMORY = 4294967295,
  VK_ERROR_OUT_OF_DEVICE_MEMORY = 4294967294,
  VK_ERROR_INITIALIZATION_FAILED = 4294967293,
  VK_ERROR_DEVICE_LOST = 4294967292,
  VK_ERROR_MEMORY_MAP_FAILED = 4294967291,
  VK_ERROR_LAYER_NOT_PRESENT = 4294967290,
  VK_ERROR_EXTENSION_NOT_PRESENT = 4294967289,
  VK_ERROR_FEATURE_NOT_PRESENT = 4294967288,
  VK_ERROR_INCOMPATIBLE_DRIVER = 4294967287,
  VK_ERROR_TOO_MANY_OBJECTS = 4294967286,
  VK_ERROR_FORMAT_NOT_SUPPORTED = 4294967285,
  VK_ERROR_SURFACE_LOST_KHR = 3294967296,
  VK_ERROR_NATIVE_WINDOW_IN_USE_KHR = 3294967295,
  VK_SUBOPTIMAL_KHR = 1000001003,
  VK_ERROR_OUT_OF_DATE_KHR = 3294966292,
  VK_ERROR_INCOMPATIBLE_DISPLAY_KHR = 3294964295,
  VK_ERROR_VALIDATION_FAILED_EXT = 3294956295,
  VK_ERROR_INVALID_SHADER_NV = 1000012000,
} VkResult;

enum class VkPhysicalDeviceType : uint32_t {
  VK_PHYSICAL_DEVICE_TYPE_OTHER = 0,
  VK_PHYSICAL_DEVICE_TYPE_INTEGRATED_GPU = 1,
  VK_PHYSICAL_DEVICE_TYPE_DISCRETE_GPU = 2,
  VK_PHYSICAL_DEVICE_TYPE_VIRTUAL_GPU = 3,
  VK_PHYSICAL_DEVICE_TYPE_CPU = 4,
};

// Type alias
typedef uint32_t VkBool32;
typedef uint32_t VkFlags;
typedef uint64_t VkDeviceSize;
typedef size_t VkInstance;
typedef size_t VkPhysicalDevice;

typedef VkFlags VkInstanceCreateFlags;
typedef VkFlags VkSampleCountFlags;

typedef void* PFN_vkAllocationFunction;
typedef void* PFN_vkReallocationFunction;
typedef void* PFN_vkFreeFunction;
typedef void* PFN_vkInternalAllocationNotification;
typedef void* PFN_vkInternalFreeNotification;
typedef void* PFN_vkVoidFunction;

// Structs
typedef struct {
  core::StaticArray<char, 256> layerName;
  uint32_t specVersion;
  uint32_t implementationVersion;
  core::StaticArray<char, 256> description;
} VkLayerProperties;

typedef struct {
  core::StaticArray<char, 256> extensionName;
  uint32_t specVersion;
} VkExtensionProperties;

typedef struct {
  VkStructureType sType;
  void* pNext;
  char* pApplicationName;
  uint32_t applicationVersion;
  char* pEngineName;
  uint32_t engineVersion;
  uint32_t apiVersion;
} VkApplicationInfo;

typedef struct {
  VkStructureType sType;
  void* pNext;
  VkInstanceCreateFlags flags;
  VkApplicationInfo* pApplicationInfo;
  uint32_t enabledLayerCount;
  char** ppEnabledLayerNames;
  uint32_t enabledExtensionCount;
  char** ppEnabledExtensionNames;
} VkInstanceCreateInfo;

typedef struct {
  uint32_t maxImageDimension1D;
  uint32_t maxImageDimension2D;
  uint32_t maxImageDimension3D;
  uint32_t maxImageDimensionCube;
  uint32_t maxImageArrayLayers;
  uint32_t maxTexelBufferElements;
  uint32_t maxUniformBufferRange;
  uint32_t maxStorageBufferRange;
  uint32_t maxPushConstantsSize;
  uint32_t maxMemoryAllocationCount;
  uint32_t maxSamplerAllocationCount;
  VkDeviceSize bufferImageGranularity;
  VkDeviceSize sparseAddressSpaceSize;
  uint32_t maxBoundDescriptorSets;
  uint32_t maxPerStageDescriptorSamplers;
  uint32_t maxPerStageDescriptorUniformBuffers;
  uint32_t maxPerStageDescriptorStorageBuffers;
  uint32_t maxPerStageDescriptorSampledImages;
  uint32_t maxPerStageDescriptorStorageImages;
  uint32_t maxPerStageDescriptorInputAttachments;
  uint32_t maxPerStageResources;
  uint32_t maxDescriptorSetSamplers;
  uint32_t maxDescriptorSetUniformBuffers;
  uint32_t maxDescriptorSetUniformBuffersDynamic;
  uint32_t maxDescriptorSetStorageBuffers;
  uint32_t maxDescriptorSetStorageBuffersDynamic;
  uint32_t maxDescriptorSetSampledImages;
  uint32_t maxDescriptorSetStorageImages;
  uint32_t maxDescriptorSetInputAttachments;
  uint32_t maxVertexInputAttributes;
  uint32_t maxVertexInputBindings;
  uint32_t maxVertexInputAttributeOffset;
  uint32_t maxVertexInputBindingStride;
  uint32_t maxVertexOutputComponents;
  uint32_t maxTessellationGenerationLevel;
  uint32_t maxTessellationPatchSize;
  uint32_t maxTessellationControlPerVertexInputComponents;
  uint32_t maxTessellationControlPerVertexOutputComponents;
  uint32_t maxTessellationControlPerPatchOutputComponents;
  uint32_t maxTessellationControlTotalOutputComponents;
  uint32_t maxTessellationEvaluationInputComponents;
  uint32_t maxTessellationEvaluationOutputComponents;
  uint32_t maxGeometryShaderInvocations;
  uint32_t maxGeometryInputComponents;
  uint32_t maxGeometryOutputComponents;
  uint32_t maxGeometryOutputVertices;
  uint32_t maxGeometryTotalOutputComponents;
  uint32_t maxFragmentInputComponents;
  uint32_t maxFragmentOutputAttachments;
  uint32_t maxFragmentDualSrcAttachments;
  uint32_t maxFragmentCombinedOutputResources;
  uint32_t maxComputeSharedMemorySize;
  core::StaticArray<uint32_t, 3> maxComputeWorkGroupCount;
  uint32_t maxComputeWorkGroupInvocations;
  core::StaticArray<uint32_t, 3> maxComputeWorkGroupSize;
  uint32_t subPixelPrecisionBits;
  uint32_t subTexelPrecisionBits;
  uint32_t mipmapPrecisionBits;
  uint32_t maxDrawIndexedIndexValue;
  uint32_t maxDrawIndirectCount;
  float maxSamplerLodBias;
  float maxSamplerAnisotropy;
  uint32_t maxViewports;
  core::StaticArray<uint32_t, 2> maxViewportDimensions;
  core::StaticArray<float, 2> viewportBoundsRange;
  uint32_t viewportSubPixelBits;
  size_t minMemoryMapAlignment;
  VkDeviceSize minTexelBufferOffsetAlignment;
  VkDeviceSize minUniformBufferOffsetAlignment;
  VkDeviceSize minStorageBufferOffsetAlignment;
  int32_t minTexelOffset;
  uint32_t maxTexelOffset;
  int32_t minTexelGatherOffset;
  uint32_t maxTexelGatherOffset;
  float minInterpolationOffset;
  float maxInterpolationOffset;
  uint32_t subPixelInterpolationOffsetBits;
  uint32_t maxFramebufferWidth;
  uint32_t maxFramebufferHeight;
  uint32_t maxFramebufferLayers;
  VkSampleCountFlags framebufferColorSampleCounts;
  VkSampleCountFlags framebufferDepthSampleCounts;
  VkSampleCountFlags framebufferStencilSampleCounts;
  VkSampleCountFlags framebufferNoAttachmentSampleCounts;
  uint32_t maxColorAttachments;
  VkSampleCountFlags sampledImageColorSampleCounts;
  VkSampleCountFlags sampledImageIntegerSampleCounts;
  VkSampleCountFlags sampledImageDepthSampleCounts;
  VkSampleCountFlags sampledImageStencilSampleCounts;
  VkSampleCountFlags storageImageSampleCounts;
  uint32_t maxSampleMaskWords;
  VkBool32 timestampComputeAndGraphics;
  float timestampPeriod;
  uint32_t maxClipDistances;
  uint32_t maxCullDistances;
  uint32_t maxCombinedClipAndCullDistances;
  uint32_t discreteQueuePriorities;
  core::StaticArray<float, 2> pointSizeRange;
  core::StaticArray<float, 2> lineWidthRange;
  float pointSizeGranularity;
  float lineWidthGranularity;
  VkBool32 strictLines;
  VkBool32 standardSampleLocations;
  VkDeviceSize optimalBufferCopyOffsetAlignment;
  VkDeviceSize optimalBufferCopyRowPitchAlignment;
  VkDeviceSize nonCoherentAtomSize;
} VkPhysicalDeviceLimits;

typedef struct {
  VkBool32 residencyStandard2DBlockShape;
  VkBool32 residencyStandard2DMultisampleBlockShape;
  VkBool32 residencyStandard3DBlockShape;
  VkBool32 residencyAlignedMipSize;
  VkBool32 residencyNonResidentStrict;
} VkPhysicalDeviceSparseProperties;

typedef struct {
  uint32_t apiVersion;
  uint32_t driverVersion;
  uint32_t vendorID;
  uint32_t deviceID;
  VkPhysicalDeviceType deviceType;
  core::StaticArray<char, 256> deviceName;
  core::StaticArray<uint8_t, 16> pipelineCacheUUID;
  VkPhysicalDeviceLimits limits;
  VkPhysicalDeviceSparseProperties sparseProperties;
} VkPhysicalDeviceProperties;

typedef struct {
  void* pUserData;
  PFN_vkAllocationFunction pfnAllocation;
  PFN_vkReallocationFunction pfnReallocation;
  PFN_vkFreeFunction pfnFree;
  PFN_vkInternalAllocationNotification pfnInternalAllocation;
  PFN_vkInternalFreeNotification pfnInternalFree;
} VkAllocationCallbacks;

// Function types
typedef PFN_vkVoidFunction(VULKAN_API_PTR* PFNVKGETINSTANCEPROCADDR)(
    VkInstance instance, const char* pName);
typedef VkResult(VULKAN_API_PTR* PFNVKENUMERATEINSTANCELAYERPROPERTIES)(
    uint32_t* pPropertyCount, VkLayerProperties* pProperties);
typedef VkResult(VULKAN_API_PTR* PFNVKENUMERATEINSTANCEEXTENSIONPROPERTIES)(
    const char* pLayerName, uint32_t* pPropertyCount,
    VkExtensionProperties* pProperties);
typedef VkResult(VULKAN_API_PTR* PFNVKCREATEINSTANCE)(
    VkInstanceCreateInfo* pCreateInfo, VkAllocationCallbacks* pAllocator,
    VkInstance* pInstance);
typedef void(VULKAN_API_PTR* PFNVKDESTROYINSTANCE)(
    VkInstance instance, VkAllocationCallbacks* pAllocator);
typedef VkResult(VULKAN_API_PTR* PFNVKENUMERATEPHYSICALDEVICES)(
    VkInstance instance, uint32_t* pPhysicalDeviceCount,
    VkPhysicalDevice* pPhysicalDevices);
typedef void(VULKAN_API_PTR* PFNVKGETPHYSICALDEVICEPROPERTIES)(
    VkPhysicalDevice physicalDevice, VkPhysicalDeviceProperties* pProperties);

#endif  // GAPID_CORE_OS_DEVICEINFO_VK_LITE
