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

#include <alloca.h>
#include <android/log.h>
#include <dlfcn.h>
#include <cstring>

#include <unistd.h>
#include "vulkan/vulkan.h"

extern "C" {

#define VK_LAYER_EXPORT __attribute__((visibility("default")))

#define LOG_DEBUG(fmt, ...) \
  __android_log_print(ANDROID_LOG_DEBUG, "AGI", fmt, ##__VA_ARGS__)

#define PROC(name)                                                     \
  static PFN_##name fn = (PFN_##name)(getProcAddress("gapid_" #name)); \
  if (fn != nullptr) return fn

static void* getLibGapii();

static void* getProcAddress(const char* name) {
  LOG_DEBUG("Looking for function %s", name);
  return dlsym(getLibGapii(), name);
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

static void* getLibGapii() {
  static void* libgapii = nullptr;
  if (libgapii == nullptr) {
    Dl_info me;
    dladdr((void*)GraphicsSpyGetDeviceProcAddr, &me);
    if (me.dli_fname != nullptr) {
      const char* base = strrchr(me.dli_fname, '/');
      if (base != nullptr) {
        int baseLen = base - me.dli_fname + 1;
        char* name = static_cast<char*>(alloca(baseLen + 12 /*"libgapii.so"*/));
        memcpy(name, me.dli_fname, baseLen);
        strncpy(name + baseLen, "libgapii.so", 12);
        LOG_DEBUG("Loading gapii at %s", name);
        libgapii = dlopen(name, RTLD_NOW);
      }
    }
  }
  return libgapii;
}

}  // extern "C"
