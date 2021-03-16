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

#include "layer.h"

#include <cstring>
#include <string>
#include <type_traits>
#include <unordered_map>
#include <vector>

#include <vulkan/vk_layer.h>
#include <vulkan/vulkan.h>

#include "swapchain.h"

#if defined(__ANDROID__)
#include <android/log.h>
#include <sys/system_properties.h>
#else
#include <iostream>
#endif

#define LAYER_NAME "VirtualSwapchain"

#define LAYER_NAME_FUNCTION(fn) VirtualSwapchain##fn

namespace swapchain {

namespace {

#if defined(__ANDROID__)
bool GetAndroidProperty(const char* const property, std::string* value) {
  char buff[PROP_VALUE_MAX];
  if (__system_property_get(property, buff) <= 0) {
    return false;
  }
  *value = buff;
  return true;
}
#endif

}  // namespace

#if defined(__ANDROID__)
void write_warning(const char* message) {
  __android_log_print(ANDROID_LOG_WARN, "VirtualSwapchainLayer", "%s", message);
}
#else
void write_warning(const char* message) {
  std::cerr << "VirtualSwapchainLayer: " << message << std::endl;
}
#endif

void write_warning(const std::string& message) {
  write_warning(message.c_str());
}

bool GetParameter(const char* const env_var_name,
                  const char* const android_prop_name,
                  std::string* param_value) {
#if defined(__ANDROID__)
  return GetAndroidProperty(android_prop_name, param_value);
#else
  const char* const env_var_value = std::getenv(env_var_name);
  if (!env_var_value) {
    return false;
  }
  *param_value = env_var_value;
  return true;
#endif
}

Context& GetGlobalContext() {
  // We rely on C++11 static initialization rules here.
  // kContext will get allocated on first use, and freed in the
  // same order (more or less).
  static Context kContext;
  return kContext;
}

namespace {

template <typename T>
struct link_info_traits {
  const static bool is_instance =
      std::is_same<T, const VkInstanceCreateInfo>::value;
  using layer_info_type =
      typename std::conditional<is_instance, VkLayerInstanceCreateInfo,
                                VkLayerDeviceCreateInfo>::type;
  const static VkStructureType sType =
      is_instance ? VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO
                  : VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO;
};

// Get layer_specific data for this layer.
// Will return either VkLayerInstanceCreateInfo or
// VkLayerDeviceCreateInfo depending on the type of the pCreateInfo
// passed in.
template <typename T>
typename link_info_traits<T>::layer_info_type* get_layer_link_info(
    T* pCreateInfo) {
  using layer_info_type = typename link_info_traits<T>::layer_info_type;

  auto layer_info = const_cast<layer_info_type*>(
      static_cast<const layer_info_type*>(pCreateInfo->pNext));

  while (layer_info) {
    if (layer_info->sType == link_info_traits<T>::sType &&
        layer_info->function == VK_LAYER_LINK_INFO) {
      return layer_info;
    }
    layer_info = const_cast<layer_info_type*>(
        static_cast<const layer_info_type*>(layer_info->pNext));
  }
  return layer_info;
}
}  // namespace

// Overload vkCreateInstance. It is all book-keeping
// and passthrough to the next layer (or ICD) in the chain.
VKAPI_ATTR VkResult VKAPI_CALL vkCreateInstance(
    const VkInstanceCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator, VkInstance* pInstance) {
  VkLayerInstanceCreateInfo* layer_info = get_layer_link_info(pCreateInfo);

  // Grab the pointer to the next vkGetInstanceProcAddr in the chain.
  PFN_vkGetInstanceProcAddr get_instance_proc_addr =
      layer_info->u.pLayerInfo->pfnNextGetInstanceProcAddr;

  // From that get the next vkCreateInstance function.
  PFN_vkCreateInstance create_instance = reinterpret_cast<PFN_vkCreateInstance>(
      get_instance_proc_addr(NULL, "vkCreateInstance"));

  if (create_instance == NULL) {
    return VK_ERROR_INITIALIZATION_FAILED;
  }
  // The next layer may read from layer_info,
  // so advance the pointer for it.
  layer_info->u.pLayerInfo = layer_info->u.pLayerInfo->pNext;

  // Actually call vkCreateInstance, and keep track of the result.
  VkResult result = create_instance(pCreateInfo, pAllocator, pInstance);

  // If it failed, then we don't need to track this instance.
  if (result != VK_SUCCESS) return result;

  PFN_vkEnumeratePhysicalDevices enumerate_physical_devices =
      reinterpret_cast<PFN_vkEnumeratePhysicalDevices>(
          get_instance_proc_addr(*pInstance, "vkEnumeratePhysicalDevices"));
  if (!enumerate_physical_devices) {
    return VK_ERROR_INITIALIZATION_FAILED;
  }

  PFN_vkEnumerateDeviceExtensionProperties
      enumerate_device_extension_properties =
          reinterpret_cast<PFN_vkEnumerateDeviceExtensionProperties>(
              get_instance_proc_addr(*pInstance,
                                     "vkEnumerateDeviceExtensionProperties"));
  if (!enumerate_device_extension_properties) {
    return VK_ERROR_INITIALIZATION_FAILED;
  }

  InstanceData data;

#define GET_PROC(name) \
  data.name =          \
      reinterpret_cast<PFN_##name>(get_instance_proc_addr(*pInstance, #name))
  GET_PROC(vkGetInstanceProcAddr);
  GET_PROC(vkDestroyInstance);
  GET_PROC(vkEnumeratePhysicalDevices);
  GET_PROC(vkEnumerateDeviceExtensionProperties);
  GET_PROC(vkCreateDevice);
  GET_PROC(vkGetPhysicalDeviceQueueFamilyProperties);
  GET_PROC(vkGetPhysicalDeviceProperties);
  GET_PROC(vkGetPhysicalDeviceMemoryProperties);

#ifdef VK_USE_PLATFORM_ANDROID_KHR
  GET_PROC(vkCreateAndroidSurfaceKHR);
#endif
#ifdef VK_USE_PLATFORM_XCB_KHR
  GET_PROC(vkCreateXcbSurfaceKHR);
#endif
#ifdef VK_USE_PLATFORM_WIN32_KHR
  GET_PROC(vkCreateWin32SurfaceKHR);
#endif
  GET_PROC(vkDestroySurfaceKHR);

#undef GET_PROC
  // Add this instance, along with the vkGetInstanceProcAddr to our
  // map. This way when someone calls vkGetInstanceProcAddr, we can forward
  // it to the correct "next" vkGetInstanceProcAddr.
  {
    auto instances = GetGlobalContext().GetInstanceMap();
    // The same instance was returned twice, this is a problem.
    if (instances->find(*pInstance) != instances->end()) {
      return VK_ERROR_INITIALIZATION_FAILED;
    }
    (*instances)[*pInstance] = data;
  }

  RegisterInstance(*pInstance, data);
  return result;
}

// On vkDestroyInstance, printf("VkDestroyInstance") and clean up our
// tracking data.
VKAPI_ATTR void vkDestroyInstance(VkInstance instance,
                                  const VkAllocationCallbacks* pAllocator) {
  // First we have to find the function to chain to, then we have to
  // remove this instance from our list, then we forward the call.
  auto instance_map = GetGlobalContext().GetInstanceMap();
  auto it = instance_map->find(instance);
  it->second.vkDestroyInstance(instance, pAllocator);
  instance_map->erase(it);
}

// Overload vkCreateDevice. It is all book-keeping
// and passthrough to the next layer (or ICD) in the chain.
VKAPI_ATTR VkResult VKAPI_CALL
vkCreateDevice(VkPhysicalDevice gpu, const VkDeviceCreateInfo* pCreateInfo,
               const VkAllocationCallbacks* pAllocator, VkDevice* pDevice) {
  VkLayerDeviceCreateInfo* layer_info = get_layer_link_info(pCreateInfo);

  // Grab the fpGetInstanceProcAddr from the layer_info. We will get
  // vkCreateDevice from this.
  // Note: we cannot use our instance_map because we do not have a
  // vkInstance here.
  PFN_vkGetInstanceProcAddr get_instance_proc_addr =
      layer_info->u.pLayerInfo->pfnNextGetInstanceProcAddr;

  PFN_vkCreateDevice create_device = reinterpret_cast<PFN_vkCreateDevice>(
      get_instance_proc_addr(NULL, "vkCreateDevice"));

  if (!create_device) {
    return VK_ERROR_INITIALIZATION_FAILED;
  }

  // We want to store off the next vkGetDeviceProcAddr so keep track of it now
  // before we advance the pointer.
  PFN_vkGetDeviceProcAddr get_device_proc_addr =
      layer_info->u.pLayerInfo->pfnNextGetDeviceProcAddr;

  // The next layer may read from layer_info,
  // so advance the pointer for it.
  layer_info->u.pLayerInfo = layer_info->u.pLayerInfo->pNext;

  // Actually make the call to vkCreateDevice.
  VkResult result = create_device(gpu, pCreateInfo, pAllocator, pDevice);

  // If we failed, then we don't store the associated pointers.
  if (result != VK_SUCCESS) {
    return result;
  }

  DeviceData data{gpu};

#define GET_PROC(name) \
  data.name =          \
      reinterpret_cast<PFN_##name>(get_device_proc_addr(*pDevice, #name));

  GET_PROC(vkGetDeviceProcAddr);
  GET_PROC(vkGetDeviceQueue);

  GET_PROC(vkAllocateMemory);
  GET_PROC(vkFreeMemory);
  GET_PROC(vkMapMemory);
  GET_PROC(vkUnmapMemory);
  GET_PROC(vkInvalidateMappedMemoryRanges);

  GET_PROC(vkCreateSemaphore);
  GET_PROC(vkDestroySemaphore);
  GET_PROC(vkCreateFence);
  GET_PROC(vkGetFenceStatus);
  GET_PROC(vkWaitForFences);
  GET_PROC(vkDestroyFence);
  GET_PROC(vkResetFences);

  GET_PROC(vkCreateImage);
  GET_PROC(vkGetImageMemoryRequirements);
  GET_PROC(vkBindImageMemory);
  GET_PROC(vkDestroyImage);

  GET_PROC(vkCreateBuffer);
  GET_PROC(vkGetBufferMemoryRequirements);
  GET_PROC(vkBindBufferMemory);
  GET_PROC(vkDestroyBuffer);

  GET_PROC(vkCreateCommandPool);
  GET_PROC(vkDestroyCommandPool);
  GET_PROC(vkAllocateCommandBuffers);
  GET_PROC(vkFreeCommandBuffers);

  GET_PROC(vkBeginCommandBuffer);
  GET_PROC(vkEndCommandBuffer);
  GET_PROC(vkResetCommandBuffer);

  GET_PROC(vkCmdCopyImageToBuffer);
  GET_PROC(vkCmdBlitImage);
  GET_PROC(vkCmdPipelineBarrier);
  GET_PROC(vkCmdWaitEvents);
  GET_PROC(vkCreateRenderPass);

  GET_PROC(vkQueueSubmit);
  GET_PROC(vkDestroyDevice);

  GET_PROC(vkCreateSwapchainKHR);
  GET_PROC(vkGetSwapchainImagesKHR);
  GET_PROC(vkAcquireNextImageKHR);
  GET_PROC(vkAcquireNextImage2KHR);
  GET_PROC(vkQueuePresentKHR);
  GET_PROC(vkDestroySwapchainKHR);

#undef GET_PROC

  // Add this device, along with the vkGetDeviceProcAddr to our map.
  // This way when someone calls vkGetDeviceProcAddr, we can forward
  // it to the correct "next" vkGetDeviceProcAddr.
  {
    auto device_map = GetGlobalContext().GetDeviceMap();
    if (device_map->find(*pDevice) != device_map->end()) {
      return VK_ERROR_INITIALIZATION_FAILED;
    }
    (*device_map)[*pDevice] = data;
  }

  {
    auto queue_map = GetGlobalContext().GetQueueMap();
    for (size_t i = 0; i < pCreateInfo->queueCreateInfoCount; ++i) {
      for (size_t j = 0; j < pCreateInfo->pQueueCreateInfos[i].queueCount;
           ++j) {
        VkQueue q;
        data.vkGetDeviceQueue(
            *pDevice, pCreateInfo->pQueueCreateInfos[i].queueFamilyIndex, j,
            &q);
        set_dispatch_from_parent(q, *pDevice);
        (*queue_map)[q] = {*pDevice, data.vkQueueSubmit,
                           data.vkQueuePresentKHR};
      }
    }
  }

  return result;
}

// On vkDestroyDevice, clean up our tracking data.
VKAPI_ATTR void vkDestroyDevice(VkDevice device,
                                const VkAllocationCallbacks* pAllocator) {
  // First we have to find the function to chain to, then we have to
  // remove this instance from our list, then we forward the call.
  auto device_map = GetGlobalContext().GetDeviceMap();
  auto it = device_map->find(device);

  it->second.vkDestroyDevice(device, pAllocator);
  device_map->erase(it);
}

static const VkLayerProperties global_layer_properties[] = {{
    LAYER_NAME,
    VK_VERSION_MAJOR(1) | VK_VERSION_MINOR(0) | 5,
    1,
    "Virtual Swapchain Layer",
}};

VkResult get_layer_properties(uint32_t* pPropertyCount,
                              VkLayerProperties* pProperties) {
  if (pProperties == NULL) {
    *pPropertyCount = 1;
    return VK_SUCCESS;
  }

  if (pPropertyCount == 0) {
    return VK_INCOMPLETE;
  }
  *pPropertyCount = 1;
  memcpy(pProperties, global_layer_properties, sizeof(global_layer_properties));
  return VK_SUCCESS;
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceLayerProperties(uint32_t* pPropertyCount,
                                   VkLayerProperties* pProperties) {
  return get_layer_properties(pPropertyCount, pProperties);
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateDeviceLayerProperties(VkPhysicalDevice, uint32_t* pPropertyCount,
                                 VkLayerProperties* pProperties) {
  return get_layer_properties(pPropertyCount, pProperties);
}

// Overload EnumeratePhysicalDevices, this is entirely for
// book-keeping.
VKAPI_ATTR VkResult VKAPI_CALL
vkEnumeratePhysicalDevices(VkInstance instance, uint32_t* pPhysicalDeviceCount,
                           VkPhysicalDevice* pPhysicalDevices) {
  auto instance_data = GetGlobalContext().GetInstanceData(instance);
  if (instance_data->physical_devices_.empty()) {
    uint32_t count;
    VkResult res =
        instance_data->vkEnumeratePhysicalDevices(instance, &count, nullptr);
    if (res != VK_SUCCESS) {
      return res;
    }
    instance_data->physical_devices_.resize(count);
    if (VK_SUCCESS !=
        (res = instance_data->vkEnumeratePhysicalDevices(
             instance, &count, instance_data->physical_devices_.data()))) {
      instance_data->physical_devices_.clear();
      return res;
    }
  }

  uint32_t count = instance_data->physical_devices_.size();
  if (pPhysicalDevices) {
    if (*pPhysicalDeviceCount > count) *pPhysicalDeviceCount = count;
    memcpy(pPhysicalDevices, instance_data->physical_devices_.data(),
           *pPhysicalDeviceCount * sizeof(VkPhysicalDevice));
  } else {
    *pPhysicalDeviceCount = count;
  }
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkAllocateCommandBuffers(
    VkDevice device, const VkCommandBufferAllocateInfo* pAllocateInfo,
    VkCommandBuffer* pCommandBuffers) {
  auto command_buffer_map = GetGlobalContext().GetCommandBufferMap();
  const auto device_data = GetGlobalContext().GetDeviceData(device);

  VkResult res = device_data->vkAllocateCommandBuffers(device, pAllocateInfo,
                                                       pCommandBuffers);
  if (res == VK_SUCCESS) {
    for (size_t i = 0; i < pAllocateInfo->commandBufferCount; ++i) {
      (*command_buffer_map)[pCommandBuffers[i]] = {
          device, device_data->vkCmdPipelineBarrier,
          device_data->vkCmdWaitEvents};
    }
  }
  return res;
}

// Overload vkEnumerateDeviceExtensionProperties
VKAPI_ATTR VkResult VKAPI_CALL vkEnumerateDeviceExtensionProperties(
    VkPhysicalDevice physicalDevice, const char* pLayerName,
    uint32_t* pPropertyCount, VkExtensionProperties* pProperties) {
  if (!physicalDevice) {
    *pPropertyCount = 0;
    return VK_SUCCESS;
  }

  auto instance_data = GetGlobalContext().GetInstanceData(
      GetGlobalContext().GetPhysicalDeviceData(physicalDevice)->instance_);

  return instance_data->vkEnumerateDeviceExtensionProperties(
      physicalDevice, pLayerName, pPropertyCount, pProperties);
}

// Overload vkEnumerateInstanceExtensionProperties
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceExtensionProperties(const char* /*pLayerName*/,
                                       uint32_t* pPropertyCount,
                                       VkExtensionProperties* /*pProperties*/) {
  *pPropertyCount = 0;
  return VK_SUCCESS;
}

VKAPI_ATTR void VKAPI_CALL vkFreeCommandBuffers(
    VkDevice device, VkCommandPool commandPool, uint32_t commandBufferCount,
    const VkCommandBuffer* pCommandBuffers) {
  auto command_buffer_map = GetGlobalContext().GetCommandBufferMap();
  for (size_t i = 0; i < commandBufferCount; ++i) {
    command_buffer_map->erase(pCommandBuffers[i]);
  }
  GetGlobalContext().GetDeviceData(device)->vkFreeCommandBuffers(
      device, commandPool, commandBufferCount, pCommandBuffers);
}

// Overload GetInstanceProcAddr.
// It also provides the overloaded function for vkCreateDevice. This way we can
// also hook vkGetDeviceProcAddr.
// Lastly it provides vkDestroyInstance for book-keeping purposes.
VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
vkGetInstanceProcAddr(VkInstance instance, const char* funcName) {
#define INTERCEPT(func)         \
  if (!strcmp(funcName, #func)) \
  return reinterpret_cast<PFN_vkVoidFunction>(func)

  INTERCEPT(vkGetInstanceProcAddr);

  INTERCEPT(vkCreateDevice);
  INTERCEPT(vkCreateInstance);
  INTERCEPT(vkDestroyInstance);
  INTERCEPT(vkEnumerateDeviceExtensionProperties);
  INTERCEPT(vkEnumerateDeviceLayerProperties);
  INTERCEPT(vkEnumerateInstanceExtensionProperties);
  INTERCEPT(vkEnumerateInstanceLayerProperties);
  INTERCEPT(vkEnumeratePhysicalDevices);

  // From here on down these are what is needed for
  // swapchain/surface support.
  INTERCEPT(vkDestroySurfaceKHR);

  INTERCEPT(vkGetPhysicalDeviceSurfaceSupportKHR);
  INTERCEPT(vkGetPhysicalDeviceSurfaceFormatsKHR);
  INTERCEPT(vkGetPhysicalDeviceSurfaceCapabilitiesKHR);
  INTERCEPT(vkGetPhysicalDeviceSurfacePresentModesKHR);

  // From here down it is just functions that have to be overriden for swapchain
  INTERCEPT(vkQueuePresentKHR);
  INTERCEPT(vkQueueSubmit);
  INTERCEPT(vkCmdPipelineBarrier);
  INTERCEPT(vkCmdWaitEvents);
  INTERCEPT(vkCreateRenderPass);

  INTERCEPT(vkCreateSwapchainKHR);
  INTERCEPT(vkDestroySwapchainKHR);
  INTERCEPT(vkGetSwapchainImagesKHR);
  INTERCEPT(vkAcquireNextImageKHR);
  INTERCEPT(vkAcquireNextImage2KHR);

  INTERCEPT(vkAllocateCommandBuffers);
  INTERCEPT(vkFreeCommandBuffers);
  INTERCEPT(vkSetSwapchainCallback);
#undef INTERCEPT

#define INTERCEPT_SURFACE(name) \
  if (!strcmp(funcName, #name)) \
  return reinterpret_cast<PFN_vkVoidFunction>(vkCreateVirtualSurface)

  // Since we are faking our swapchains, we also have to fake the surface.
  // Intercept all of the surface creation routines for all platforms.
  INTERCEPT_SURFACE(vkCreateAndroidSurfaceKHR);
  INTERCEPT_SURFACE(vkCreateMirSurfaceKHR);
  INTERCEPT_SURFACE(vkCreateWaylandSurfaceKHR);
  INTERCEPT_SURFACE(vkCreateWin32SurfaceKHR);
  INTERCEPT_SURFACE(vkCreateXcbSurfaceKHR);
  INTERCEPT_SURFACE(vkCreateXlibSurfaceKHR);
  INTERCEPT_SURFACE(vkCreateMacOSSurfaceMVK);

#undef INTERCEPT_SURFACE
  // If we are calling a non-overloaded function then we have to
  // return the "next" in the chain. On vkCreateInstance we stored this in
  // the map so we can call it here.
  PFN_vkGetInstanceProcAddr instance_proc_addr =
      GetGlobalContext().GetInstanceData(instance)->vkGetInstanceProcAddr;

  return instance_proc_addr(instance, funcName);
}

// Overload GetDeviceProcAddr.
// We provide an overload of vkDestroyDevice for book-keeping.
// The rest of the overloads are swapchain-specific.
VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
vkGetDeviceProcAddr(VkDevice dev, const char* funcName) {
#define INTERCEPT(func)         \
  if (!strcmp(funcName, #func)) \
  return reinterpret_cast<PFN_vkVoidFunction>(func)

  INTERCEPT(vkGetDeviceProcAddr);
  INTERCEPT(vkDestroyDevice);

  // From here down it is just functions that have to be overriden for swapchain
  INTERCEPT(vkQueuePresentKHR);
  INTERCEPT(vkQueueSubmit);
  INTERCEPT(vkCmdPipelineBarrier);
  INTERCEPT(vkCmdWaitEvents);
  INTERCEPT(vkCreateRenderPass);

  INTERCEPT(vkCreateSwapchainKHR);
  INTERCEPT(vkDestroySwapchainKHR);
  INTERCEPT(vkGetSwapchainImagesKHR);
  INTERCEPT(vkAcquireNextImageKHR);
  INTERCEPT(vkAcquireNextImage2KHR);

  INTERCEPT(vkAllocateCommandBuffers);
  INTERCEPT(vkFreeCommandBuffers);
  INTERCEPT(vkSetSwapchainCallback);

  INTERCEPT(vkSetHdrMetadataEXT);
#undef INTERCEPT

  // If we are calling a non-overloaded function then we have to
  // return the "next" in the chain. On vkCreateDevice we stored this in the
  // map so we can call it here.

  PFN_vkGetDeviceProcAddr device_proc_addr =
      GetGlobalContext().GetDeviceData(dev)->vkGetDeviceProcAddr;
  return device_proc_addr(dev, funcName);
}
}  // namespace swapchain

extern "C" {

// For this to function on Android the entry-point names for GetDeviceProcAddr
// and GetInstanceProcAddr must be ${layer_name}/Get*ProcAddr.
// This is a bit surprising given that we *MUST* also export
// vkEnumerate*Layers without any prefix.
VK_LAYER_EXPORT VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
LAYER_NAME_FUNCTION(GetDeviceProcAddr)(VkDevice dev, const char* funcName) {
  return swapchain::vkGetDeviceProcAddr(dev, funcName);
}

VK_LAYER_EXPORT VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL LAYER_NAME_FUNCTION(
    GetInstanceProcAddr)(VkInstance instance, const char* funcName) {
  return swapchain::vkGetInstanceProcAddr(instance, funcName);
}

// Documentation is sparse for Android, looking at libvulkan.so
// These 4 functions must be defined in order for this to even
// be considered for loading.
#if defined(__ANDROID__)
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceLayerProperties(uint32_t* pPropertyCount,
                                   VkLayerProperties* pProperties) {
  return swapchain::vkEnumerateInstanceLayerProperties(pPropertyCount,
                                                       pProperties);
}

// On Android this must also be defined, even if we have 0
// layers to expose.
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceExtensionProperties(const char* pLayerName,
                                       uint32_t* pPropertyCount,
                                       VkExtensionProperties* pProperties) {
  return swapchain::vkEnumerateInstanceExtensionProperties(
      pLayerName, pPropertyCount, pProperties);
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL vkEnumerateDeviceLayerProperties(
    VkPhysicalDevice physicalDevice, uint32_t* pPropertyCount,
    VkLayerProperties* pProperties) {
  return swapchain::vkEnumerateDeviceLayerProperties(
      physicalDevice, pPropertyCount, pProperties);
}

// On Android this must also be defined, even if we have 0
// layers to expose.
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateDeviceExtensionProperties(VkPhysicalDevice physicalDevice,
                                     const char* pLayerName,
                                     uint32_t* pPropertyCount,
                                     VkExtensionProperties* pProperties) {
  return swapchain::vkEnumerateDeviceExtensionProperties(
      physicalDevice, pLayerName, pPropertyCount, pProperties);
}
#endif
}
