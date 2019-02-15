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

#include <dlfcn.h>
#include <cstring>

#include "vulkan/vulkan.h"

extern "C" {

#define VK_LAYER_EXPORT __attribute__((visibility("default")))
typedef void* (*eglGetProcAddress)(const char* procname);

#define PROC(name)                                                       \
  static PFN_##name fn = (PFN_##name)(getProcAddress()("gapid_" #name)); \
  if (fn != nullptr) return fn

// On android due to linker namespaces we cannot open libgapii.so,
// but since we already have it loaded, and libEGL.so hooked,
// we can use eglGetProcAddress to find the functions.
eglGetProcAddress getProcAddress() {
  static void* libegl = dlopen("libEGL.so", RTLD_NOW);
  static eglGetProcAddress pa =
      (eglGetProcAddress)dlsym(libegl, "eglGetProcAddress");
  return pa;
}

VK_LAYER_EXPORT VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
GraphicsSpyGetDeviceProcAddr(VkDevice dev, const char* funcName) {
  PROC(vkGetDeviceProcAddr)(dev, funcName);
  return nullptr;
}

VK_LAYER_EXPORT VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
GraphicsSpyGetInstanceProcAddr(VkInstance instance, const char* funcName) {
  PROC(vkGetInstanceProcAddr)(instance, funcName);
  return nullptr;
}

// This should probably match the struct in vulkan_extras.cpp's
// SpyOverride_vkEnumerateInstanceLayerProperties.
static const VkLayerProperties global_layer_properties[] = {{
    "GraphicsSpy",
    VK_VERSION_MAJOR(1) | VK_VERSION_MINOR(0) | 5,
    1,
    "vulkan_trace",
}};

static VkResult get_layer_properties(uint32_t* pCount,
                                     VkLayerProperties* pProperties) {
  if (pProperties == NULL) {
    *pCount = 1;
    return VK_SUCCESS;
  }

  if (pCount == 0) {
    return VK_INCOMPLETE;
  }
  *pCount = 1;
  memcpy(pProperties, global_layer_properties, sizeof(global_layer_properties));
  return VK_SUCCESS;
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceLayerProperties(uint32_t* pCount,
                                   VkLayerProperties* pProperties) {
  return get_layer_properties(pCount, pProperties);
}

// On Android this must also be defined, even if we have 0
// layers to expose.
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceExtensionProperties(const char* pLayerName, uint32_t* pCount,
                                       VkExtensionProperties* pProperties) {
  *pCount = 0;
  return VK_SUCCESS;
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL vkEnumerateDeviceLayerProperties(
    VkPhysicalDevice device, uint32_t* pCount, VkLayerProperties* pProperties) {
  return get_layer_properties(pCount, pProperties);
}

// On android this must also be defined, even if we have 0
// layers to expose.
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateDeviceExtensionProperties(VkPhysicalDevice device,
                                     const char* pLayerName, uint32_t* pCount,
                                     VkExtensionProperties* pProperties) {
  *pCount = 0;
  return VK_SUCCESS;
}
}  // extern "C"
