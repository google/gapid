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

#define VK_NO_PROTOTYPES
#include <vulkan/vk_layer.h>
#include <vulkan/vulkan.h>

#include "call_forwards.h"
#include "device.h"
#include "handles.h"
#include "helpers.h"
#include "instance.h"
#include "layer_base.h"
#include "physical_device.h"

namespace gapid2 {
layer_base* get_layer_base();
}

template <typename T>
struct link_info_traits {
  const static bool is_instance =
      std::is_same<T, const VkInstanceCreateInfo>::value;

  using layer_info_type =
      typename std::conditional<is_instance,
                                VkLayerInstanceCreateInfo,
                                VkLayerDeviceCreateInfo>::type;

  const static VkStructureType sType =
      is_instance ? VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO
                  : VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO;
};

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

template <typename T>
typename link_info_traits<T>::layer_info_type* get_layer_fn_info(
    T* pCreateInfo) {
  using layer_info_type = typename link_info_traits<T>::layer_info_type;

  auto layer_info = const_cast<layer_info_type*>(
      static_cast<const layer_info_type*>(pCreateInfo->pNext));

  while (layer_info) {
    if (layer_info->sType == link_info_traits<T>::sType &&
        layer_info->function == VK_LOADER_DATA_CALLBACK) {
      return layer_info;
    }
    layer_info = const_cast<layer_info_type*>(
        static_cast<const layer_info_type*>(layer_info->pNext));
  }
  return layer_info;
}

namespace {
static const VkLayerProperties props[] = {
    {"Gapid2", VK_VERSION_MAJOR(1) | VK_VERSION_MINOR(0) | 5, 1, "GAPID2"}};

VKAPI_ATTR VkResult VKAPI_CALL
get_layer_properties(uint32_t* pPropertyCount, VkLayerProperties* pProperties) {
  if (!pProperties) {
    *pPropertyCount = 1;
    return VK_SUCCESS;
  }

  if (pPropertyCount == 0) {
    return VK_INCOMPLETE;
  }

  *pPropertyCount = 1;
  memcpy(pProperties, props, sizeof(props));
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL
physical_device_layer_properties(VkPhysicalDevice,
                                 uint32_t* pPropertyCount,
                                 VkLayerProperties* pProperties) {
  return get_layer_properties(pPropertyCount, pProperties);
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
gapid2_vkEnumerateInstanceLayerProperties(uint32_t* pPropertyCount,
                                          VkLayerProperties* pProperties) {
  return (VkResult)get_layer_properties(pPropertyCount, pProperties);
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
gapid2_vkEnumerateDeviceLayerProperties(VkPhysicalDevice device,
                                        uint32_t* pPropertyCount,
                                        VkLayerProperties* pProperties) {
  return (VkResult)physical_device_layer_properties(device, pPropertyCount,
                                                    pProperties);
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
enumerate_instance_layer_properties(const char*,
                                    uint32_t* pPropertyCount,
                                    VkExtensionProperties*) {
  *pPropertyCount = 0;
  return VK_SUCCESS;
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
gapid2_vkEnumerateInstanceExtensionProperties(
    const char* pLayerName,
    uint32_t* pPropertyCount,
    VkExtensionProperties* pProperties) {
  return enumerate_instance_layer_properties(pLayerName, pPropertyCount,
                                             pProperties);
}

// Overload vkEnumerateDeviceExtensionProperties
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
gapid2_vkEnumerateDeviceExtensionProperties(
    VkPhysicalDevice device,
    const char* pLayerName,
    uint32_t* pPropertyCount,
    VkExtensionProperties* pProperties) {
  if (!device) {
    return VK_SUCCESS;
  }

  return gapid2::vkEnumerateDeviceExtensionProperties(
      device, pLayerName, pPropertyCount, pProperties);
}

VKAPI_ATTR VkResult VKAPI_CALL
gapid2_vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                        const VkAllocationCallbacks* pAllocator,
                        VkInstance* pInstance) {
  VkLayerInstanceCreateInfo* layer_info = get_layer_link_info(pCreateInfo);
  VkLayerInstanceCreateInfo* set_instance_loader_data =
      get_layer_fn_info(pCreateInfo);

  PFN_vkGetInstanceProcAddr get_instance_proc_addr =
      layer_info->u.pLayerInfo->pfnNextGetInstanceProcAddr;

  PFN_vkCreateInstance create_instance = reinterpret_cast<PFN_vkCreateInstance>(
      get_instance_proc_addr(NULL, "vkCreateInstance"));

  if (!create_instance) {
    return VK_ERROR_INITIALIZATION_FAILED;
  }

  layer_info->u.pLayerInfo = layer_info->u.pLayerInfo->pNext;

  gapid2::get_layer_base()->set_nexts(create_instance, get_instance_proc_addr);

  VkResult result =
      gapid2::get_layer_base()->get_top_level_functions()->vkCreateInstance(pCreateInfo, pAllocator, pInstance);

  return result;
}
}  // namespace

VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
gapid2_vkGetDeviceProcAddr(VkDevice device, const char* pName) {
  if (strcmp(pName, "vkEnumerateDeviceExtensionProperties") == 0) {
    return (PFN_vkVoidFunction)gapid2_vkEnumerateDeviceExtensionProperties;
  }

  if (strcmp(pName, "vkGetDeviceProcAddr") == 0) {
    return (PFN_vkVoidFunction)gapid2_vkGetDeviceProcAddr;
  }
  return gapid2::get_device_function(pName);
}

VKAPI_ATTR VkResult VKAPI_CALL
gapid2_vkCreateDevice(VkPhysicalDevice physicalDevice,
                      const VkDeviceCreateInfo* pCreateInfo,
                      VkAllocationCallbacks* pAllocator,
                      VkDevice* pDevice) {
  VkLayerDeviceCreateInfo* layer_info = get_layer_link_info(pCreateInfo);
  VkLayerDeviceCreateInfo* set_device_loader_data =
      get_layer_fn_info(pCreateInfo);
  PFN_vkGetInstanceProcAddr get_instance_proc_addr =
      layer_info->u.pLayerInfo->pfnNextGetInstanceProcAddr;

  PFN_vkGetDeviceProcAddr get_device_proc_addr =
      layer_info->u.pLayerInfo->pfnNextGetDeviceProcAddr;
  gapid2::get_layer_base()->set_device_nexts(get_device_proc_addr);

  layer_info->u.pLayerInfo = layer_info->u.pLayerInfo->pNext;

  auto ret =
      gapid2::vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
  if (ret != VK_SUCCESS) {
    return ret;
  }

  return ret;
}

VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
gapid2_vkGetInstanceProcAddr(VkInstance instance, const char* pName) {
  if (!strcmp(pName, "vkCreateInstance")) {
    return (PFN_vkVoidFunction)gapid2_vkCreateInstance;
  }
  if (!strcmp(pName, "vkEnumerateInstanceExtensionProperties")) {
    return (PFN_vkVoidFunction)gapid2_vkEnumerateInstanceExtensionProperties;
  }
  if (!strcmp(pName, "vkGetInstanceProcAddr")) {
    return (PFN_vkVoidFunction)gapid2_vkGetInstanceProcAddr;
  }
  if (!strcmp(pName, "vkGetDeviceProcAddr")) {
    return (PFN_vkVoidFunction)gapid2_vkGetDeviceProcAddr;
  }
  if (!strcmp(pName, "vkCreateDevice")) {
    return (PFN_vkVoidFunction)gapid2_vkCreateDevice;
  }
  if (instance == 0) {
    return nullptr;
  }
  return gapid2::get_instance_function(pName);
}
