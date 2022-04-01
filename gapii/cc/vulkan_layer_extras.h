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

// Functionality in this file was generated from vulkan_layer.h with the
// following license.
/*
 * Copyright (c) 2015-2016 The Khronos Group Inc.
 * Copyright (c) 2015-2016 Valve Corporation
 * Copyright (c) 2015-2016 LunarG, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

#ifndef GAPII_VULKAN_LAYER_EXTRAS_H
#define GAPII_VULKAN_LAYER_EXTRAS_H

#include "gapii/cc/vulkan_exports.h"
#include "gapii/cc/vulkan_extras.h"
#include "gapii/cc/vulkan_imports.h"
#include "gapii/cc/vulkan_spy.h"
#include "gapii/cc/vulkan_types.h"

namespace gapii {
namespace {

// ------------------------------------------------------------------------------------------------
// CreateInstance and CreateDevice support structures

typedef enum VkLayerFunction_ {
  VK_LAYER_LINK_INFO = 0,
  VK_LAYER_DEVICE_INFO = 1,
  VK_LAYER_INSTANCE_INFO = 2
} VkLayerFunction;

/*
 * When creating the device chain the loader needs to pass
 * down information about it's device structure needed at
 * the end of the chain. Passing the data via the
 * VkLayerInstanceInfo avoids issues with finding the
 * exact instance being used.
 */
typedef struct VkLayerInstanceInfo_ {
  void* instance_info;
  gapii::VulkanImports::PFNVKGETINSTANCEPROCADDR pfnNextGetInstanceProcAddr;
} VkLayerInstanceInfo;

typedef struct VkLayerInstanceLink_ {
  struct VkLayerInstanceLink_* pNext;
  gapii::VulkanImports::PFNVKGETINSTANCEPROCADDR pfnNextGetInstanceProcAddr;
} VkLayerInstanceLink;

/*
 * When creating the device chain the loader needs to pass
 * down information about it's device structure needed at
 * the end of the chain. Passing the data via the
 * VkLayerDeviceInfo avoids issues with finding the
 * exact instance being used.
 */
typedef struct VkLayerDeviceInfo_ {
  void* device_info;
  gapii::VulkanImports::PFNVKGETINSTANCEPROCADDR pfnNextGetInstanceProcAddr;
} VkLayerDeviceInfo;

typedef struct {
  uint32_t sType;  // VK_STRUCTURE_TYPE_LAYER_INSTANCE_CREATE_INFO
  const void* pNext;
  VkLayerFunction function;
  union {
    VkLayerInstanceLink* pLayerInfo;
    VkLayerInstanceInfo instanceInfo;
  } u;
} VkLayerInstanceCreateInfo;

typedef struct VkLayerDeviceLink_ {
  struct VkLayerDeviceLink_* pNext;
  gapii::VulkanImports::PFNVKGETINSTANCEPROCADDR pfnNextGetInstanceProcAddr;
  gapii::VulkanImports::PFNVKGETDEVICEPROCADDR pfnNextGetDeviceProcAddr;
} VkLayerDeviceLink;

typedef struct {
  uint32_t sType;  // VK_STRUCTURE_TYPE_LAYER_DEVICE_CREATE_INFO
  const void* pNext;
  VkLayerFunction function;
  union {
    VkLayerDeviceLink* pLayerInfo;
    VkLayerDeviceInfo deviceInfo;
  } u;
} VkLayerDeviceCreateInfo;

// Vulkan 1.0 version number
#define VK_API_VERSION_1_0 VK_MAKE_VERSION(1, 0, 0)

#define VK_VERSION_MAJOR(version) ((uint32_t)(version) >> 22)
#define VK_VERSION_MINOR(version) (((uint32_t)(version) >> 12) & 0x3ff)
#define VK_VERSION_PATCH(version) ((uint32_t)(version)&0xfff)

#if defined(_WIN32)
#define VK_LAYER_EXPORT __declspec(dllexport)
#elif defined(__GNUC__) && __GNUC__ >= 4
#define VK_LAYER_EXPORT __attribute__((visibility("default")))
#elif defined(__SUNPRO_C) && (__SUNPRO_C >= 0x590)
#define VK_LAYER_EXPORT __attribute__((visibility("default")))
#else
#define VK_LAYER_EXPORT
#endif

// ------------------------------------------------------------------------------------------------
// API functions

template <typename T>
struct link_info_traits {
  const static bool is_instance = std::is_same<T, VkInstanceCreateInfo>::value;
  using layer_info_type =
      typename std::conditional<is_instance, VkLayerInstanceCreateInfo,
                                VkLayerDeviceCreateInfo>::type;
  const static uint32_t sType =
      is_instance
          ? VkStructureType::VK_STRUCTURE_TYPE_LOADER_INSTANCE_CREATE_INFO
          : VkStructureType::VK_STRUCTURE_TYPE_LOADER_DEVICE_CREATE_INFO;
};

// Get layer_specific data for this layer.
// Will return either VkLayerInstanceCreateInfo or
// VkLayerDeviceCreateInfo depending on the type of the pCreateInfo
// passed in.
template <typename T>
typename link_info_traits<T>::layer_info_type* get_layer_link_info(
    const T* pCreateInfo) {
  using layer_info_type = typename link_info_traits<T>::layer_info_type;

  auto layer_info = const_cast<layer_info_type*>(
      static_cast<const layer_info_type*>(pCreateInfo->mpNext));

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
}  // namespace gapii

#endif
