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

#include "vulkan/vulkan.h"

#if defined(_WIN32)
#define VK_LAYER_EXPORT __declspec(dllexport)
#elif defined(__GNUC__)
#define VK_LAYER_EXPORT __attribute__((visibility("default")))
#else
#define VK_LAYER_EXPORT
#endif

extern "C" {

typedef void *(*eglGetProcAddress)(char const *procname);

#ifdef _WIN32
#include <windows.h>
#include <cstdio>
// On windows we do not have linker namespaces, we also don't have a
// convenient way to have already loaded libgapii.dll. In this case
// we can just LoadModule(libgapii.dll) and get the pointers from there.
#define PROC(name)                                          \
  static PFN_##name fn = (PFN_##name)getProcAddress(#name); \
  if (fn != nullptr) return fn

HMODULE LoadGAPIIDLL() {
  char path[MAX_PATH] = {'\0'};
  HMODULE this_module = GetModuleHandle("libVkLayer_GraphicsSpy.dll");
  const char libgapii[] = "libgapii.dll";
  if (this_module == NULL) {
    fprintf(stderr, "Could not find libVkLayer_GraphicsSpy.dll\n");
    return 0;
  }
  SetLastError(0);
  DWORD num_characters = GetModuleFileName(this_module, path, MAX_PATH);
  if (GetLastError() != 0) {
    fprintf(stderr, "Could not the path to libVkLayer_GraphicsSpy.dll\n");
    return 0;
  }
  for (DWORD i = num_characters - 1; i >= 0; --i) {
    // Wipe out the file-name but keep the directory name if we can.
    if (path[i] == '\\') {
      path[i] = '\0';
    }
    num_characters = i + 1;
  }

  if (num_characters + strlen(libgapii) + 1 > MAX_PATH) {
    fprintf(stderr, "Path too long\n");
    return 0;
  }
  // Append "libgapii.dll" to the full path of libVKLayer_GraphicsSpy
  memcpy(path + num_characters, libgapii, strlen(libgapii) + 1);
  return LoadLibrary(path);
}

FARPROC getProcAddress(const char *name) {
  static HMODULE libgapii = LoadGAPIIDLL();
  if (libgapii != NULL) {
    return GetProcAddress(libgapii, name);
  }
  return NULL;
}

#else

#include <dlfcn.h>

#define PROC(name)                                                       \
  static PFN_##name fn = (PFN_##name)(getProcAddress()("gapid_" #name)); \
  if (fn != nullptr) return fn

// On android due to linker namespaces we cannot open libgapii.so,
// but since we already have it loaded, and libEGL.so hooked,
// we can use eglGetProcAddress to find the functions.
eglGetProcAddress getProcAddress() {
  static void *libegl = dlopen("libEGL.so", RTLD_NOW);
  static eglGetProcAddress pa =
      (eglGetProcAddress)dlsym(libegl, "eglGetProcAddress");
  return pa;
}
#endif

VK_LAYER_EXPORT VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
VkGraphicsSpyGetDeviceProcAddr(VkDevice dev, const char *funcName) {
  PROC(vkGetDeviceProcAddr)(dev, funcName);
  return nullptr;
}

VK_LAYER_EXPORT VKAPI_ATTR PFN_vkVoidFunction VKAPI_CALL
VkGraphicsSpyGetInstanceProcAddr(VkInstance instance, const char *funcName) {
  PROC(vkGetInstanceProcAddr)(instance, funcName);
  return nullptr;
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceLayerProperties(uint32_t *pCount,
                                   VkLayerProperties *pProperties) {
  PROC(vkEnumerateInstanceLayerProperties)(pCount, pProperties);
  *pCount = 0;
  return VK_SUCCESS;
}

// On Android this must also be defined, even if we have 0
// layers to expose.
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateInstanceExtensionProperties(const char *pLayerName, uint32_t *pCount,
                                       VkExtensionProperties *pProperties) {
  PROC(vkEnumerateInstanceExtensionProperties)(pLayerName, pCount, pProperties);
  *pCount = 0;
  return VK_SUCCESS;
}

VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL vkEnumerateDeviceLayerProperties(
    VkPhysicalDevice device, uint32_t *pCount, VkLayerProperties *pProperties) {
  PROC(vkEnumerateDeviceLayerProperties)(device, pCount, pProperties);
  *pCount = 0;
  return VK_SUCCESS;
}

// On android this must also be defined, even if we have 0
// layers to expose.
VK_LAYER_EXPORT VKAPI_ATTR VkResult VKAPI_CALL
vkEnumerateDeviceExtensionProperties(VkPhysicalDevice device,
                                     const char *pLayerName, uint32_t *pCount,
                                     VkExtensionProperties *pProperties) {
  PROC(vkEnumerateDeviceExtensionProperties)
  (device, pLayerName, pCount, pProperties);
  *pCount = 0;
  return VK_SUCCESS;
}
}  // extern "C"
