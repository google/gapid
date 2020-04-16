/*
 * Copyright (C) 2019 Google Inc.
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

#include <math.h>
#include <chrono>
#include <cstdint>
#include <cstdlib>
#include <cstring>
#include <iostream>
#include <string>
#include <unordered_set>

#include <vector>
#if _WIN32
#include <windows.h>
#define VK_USE_PLATFORM_WIN32_KHR
#elif defined(__ANDROID__)
#include <android/log.h>
#include <dlfcn.h>
#include "android_native_app_glue.h"
#define VK_USE_PLATFORM_ANDROID_KHR
#elif defined(__linux__)
#include <dlfcn.h>
#include <xcb/xcb.h>
#include <iostream>
#define VK_USE_PLATFORM_XCB_KHR
#endif

#include "vulkan/vulkan.h"

const uint32_t kBufferingCount = 2;

namespace cube {
#include "cube.h"
}

namespace icon {
#include "tools/logo/logo_256.h"
}
const uint32_t vertex_shader[] =
#include "vert.h"
    ;

const uint32_t fragment_shader[] =
#include "frag.h"
    ;

const static VkFormat kDepthFormat = VK_FORMAT_D16_UNORM;

#if _WIN32
HWND kNativeWindowHandle;
HINSTANCE kNativeHInstance;
HANDLE kOutHandle;
HMODULE kVulkanLibraryHandle;
const char* kRequiredInstanceExtensions[] = {
    VK_KHR_SURFACE_EXTENSION_NAME, VK_KHR_WIN32_SURFACE_EXTENSION_NAME};

void write_error(HANDLE handle, const char* message) {
  DWORD written = 0;
  WriteConsole(handle, message, static_cast<DWORD>(strlen(message)), &written,
               nullptr);
}

HMODULE LoadVulkan() { return LoadLibrary("vulkan-1.dll"); }

void* GetLibraryProcAddr(HMODULE library, const char* function_name) {
  return reinterpret_cast<void*>(GetProcAddress(library, function_name));
}

// Create Win32 window
bool CreateNativeWindow(int width, int height) {
  kOutHandle = GetStdHandle(STD_OUTPUT_HANDLE);
  if (kOutHandle == INVALID_HANDLE_VALUE) {
    AllocConsole();
    kOutHandle = GetStdHandle(STD_OUTPUT_HANDLE);
    if (kOutHandle == INVALID_HANDLE_VALUE) {
      return false;
    }
  }

  WNDCLASSEX window_class;
  window_class.cbSize = sizeof(WNDCLASSEX);
  window_class.style = CS_HREDRAW | CS_VREDRAW;
  window_class.lpfnWndProc = &DefWindowProc;
  window_class.cbClsExtra = 0;
  window_class.cbWndExtra = 0;
  window_class.hInstance = GetModuleHandle(NULL);
  window_class.hIcon = NULL;
  window_class.hCursor = NULL;
  window_class.hbrBackground = NULL;
  window_class.lpszMenuName = NULL;
  window_class.lpszClassName = "Sample application";
  window_class.hIconSm = NULL;
  if (!RegisterClassEx(&window_class)) {
    DWORD num_written = 0;
    write_error(kOutHandle, "Could not register class");
    return false;
  }
  RECT rect = {0, 0, LONG(width), LONG(height)};

  AdjustWindowRect(&rect, WS_OVERLAPPEDWINDOW, FALSE);

  kNativeWindowHandle =
      CreateWindowEx(0, "Sample application", "", WS_OVERLAPPEDWINDOW,
                     CW_USEDEFAULT, CW_USEDEFAULT, rect.right - rect.left,
                     rect.bottom - rect.top, 0, 0, GetModuleHandle(NULL), NULL);

  if (!kNativeWindowHandle) {
    write_error(kOutHandle, "Could not create window");
    return false;
  }

  kNativeHInstance = reinterpret_cast<HINSTANCE>(
      GetWindowLongPtr(kNativeWindowHandle, GWLP_HINSTANCE));
  ShowWindow(kNativeWindowHandle, SW_SHOW);
  return true;
}

void ProcessNativeWindowEvents() {
  MSG msg;
  while (PeekMessage(&msg, kNativeWindowHandle, 0, 0, PM_REMOVE)) {
    TranslateMessage(&msg);
    DispatchMessage(&msg);
  }
}

#elif defined(__ANDROID__)

const int kOutHandle = 0;
void write_error(int, const char* message) {
  __android_log_print(ANDROID_LOG_ERROR, "GAPIDVKSAMPLE", "%s", message);
}

const char* kRequiredInstanceExtensions[] = {
    VK_KHR_SURFACE_EXTENSION_NAME, VK_KHR_ANDROID_SURFACE_EXTENSION_NAME};
struct ANativeWindow* kANativeWindowHandle = nullptr;
void* kVulkanLibraryHandle = nullptr;

void* LoadVulkan() {
  void* v = dlopen("libvulkan.so", RTLD_NOW);
  if (v == nullptr) {
    write_error(kOutHandle, "Failed to open libvulkan");
    std::terminate();
  }
  return v;
}

void processAppCmd(struct android_app* app, int32_t cmd) {
  switch (cmd) {
    case APP_CMD_INIT_WINDOW:
      kANativeWindowHandle = app->window;
      break;
    case APP_CMD_PAUSE:
    case APP_CMD_STOP:
    case APP_CMD_DESTROY:
      ANativeActivity_finish(app->activity);
      break;
  }
  return;
}

bool CreateNativeWindow(int width, int height) {
  // Window dimensions are ignored on Android
  // At that point, we should already have a window in kANativeWindowHandle
  return kANativeWindowHandle != nullptr;
}

// Forward declaration, must use an other name than "main"
int main_impl();

struct android_app* kAndroidApp = nullptr;

void ProcessNativeWindowEvents() {
  int events;
  int timeoutMillis = 0;
  struct android_poll_source* source;
  while ((ALooper_pollOnce(timeoutMillis, nullptr, &events, (void**)&source)) >=
         0) {
    if (source != nullptr) {
      source->process(kAndroidApp, source);
    }

    if (kAndroidApp->destroyRequested != 0) {
      // Abrupt termination
      std::terminate();
    }
  }
}

void android_main(struct android_app* app) {
  kAndroidApp = app;
  kAndroidApp->onAppCmd = processAppCmd;

  bool waiting_for_window = true;
  while (waiting_for_window) {
    int events;
    int timeoutMillis = 100;
    struct android_poll_source* source;
    while ((ALooper_pollOnce(timeoutMillis, nullptr, &events,
                             (void**)&source)) >= 0) {
      if (source != nullptr) {
        source->process(kAndroidApp, source);
      }
      if (waiting_for_window && kANativeWindowHandle != nullptr) {
        waiting_for_window = false;
      }
      if (kAndroidApp->destroyRequested != 0) {
        return;
      }
    }
  }

  main_impl();
};

#elif defined(__linux__)
typedef xcb_connection_t* (*pfn_xcb_connect)(const char*, int*);
typedef xcb_screen_iterator_t (*pfn_xcb_setup_roots_iterator)(
    const xcb_setup_t*);
typedef const struct xcb_setup_t* (*pfn_xcb_get_setup)(xcb_connection_t*);
typedef uint32_t (*pfn_xcb_generate_id)(xcb_connection_t*);
typedef xcb_void_cookie_t (*pfn_xcb_create_window)(xcb_connection_t*, uint8_t,
                                                   xcb_window_t, xcb_window_t,
                                                   int16_t, int16_t, uint16_t,
                                                   uint16_t, uint16_t, uint16_t,
                                                   xcb_visualid_t, uint32_t,
                                                   const uint32_t*);
typedef xcb_void_cookie_t (*pfn_xcb_map_window)(xcb_connection_t*,
                                                xcb_window_t);
typedef int (*pfn_xcb_flush)(xcb_connection_t* c);

const char* kRequiredInstanceExtensions[] = {VK_KHR_SURFACE_EXTENSION_NAME,
                                             VK_KHR_XCB_SURFACE_EXTENSION_NAME};
xcb_connection_t* k_native_connection_handle_;
xcb_window_t k_native_window_handle_;
void* kOutHandle;

void write_error(void* handle, const char* message) {
  std::cerr << message << std::endl;
}

void* load_xcb() {
  void* v = dlopen("libxcb.so.1", RTLD_NOW);
  if (!v) {
    v = dlopen("libxcb.so", RTLD_NOW);
  }
  if (!v) {
    write_error(kOutHandle, "Error opening libxcb.so");
  }
  return v;
}

void* kVulkanLibraryHandle;
void* LoadVulkan() {
  void* v = dlopen("libvulkan.so.1", RTLD_NOW);
  if (v == nullptr) {
    write_error(kOutHandle, "Failed to open libvulkan");
    std::terminate();
  }
  return v;
}

void* get_xcb() {
  static void* xcb = load_xcb();
  return xcb;
}

void* get_xcb_function(void* v, const char* fn) { return dlsym(v, fn); }

bool CreateNativeWindow(int width, int height) {
  void* xcb = get_xcb();
  pfn_xcb_connect _connect =
      (pfn_xcb_connect)get_xcb_function(xcb, "xcb_connect");

  k_native_connection_handle_ = _connect(nullptr, nullptr);
  if (!k_native_connection_handle_) {
    return false;
  }

  pfn_xcb_setup_roots_iterator _setup_roots_iterator =
      (pfn_xcb_setup_roots_iterator)get_xcb_function(
          xcb, "xcb_setup_roots_iterator");
  pfn_xcb_get_setup _get_setup =
      (pfn_xcb_get_setup)get_xcb_function(xcb, "xcb_get_setup");

  xcb_screen_t* screen =
      _setup_roots_iterator(_get_setup(k_native_connection_handle_)).data;
  if (!screen) {
    return false;
  }

  pfn_xcb_generate_id _generate_id =
      (pfn_xcb_generate_id)get_xcb_function(xcb, "xcb_generate_id");
  k_native_window_handle_ = _generate_id(k_native_connection_handle_);

  pfn_xcb_create_window _create_window =
      (pfn_xcb_create_window)get_xcb_function(xcb, "xcb_create_window");
  _create_window(k_native_connection_handle_, XCB_COPY_FROM_PARENT,
                 k_native_window_handle_, screen->root, 0, 0, width, height, 1,
                 XCB_WINDOW_CLASS_INPUT_OUTPUT, screen->root_visual, 0,
                 nullptr);

  pfn_xcb_map_window _map_window =
      (pfn_xcb_map_window)get_xcb_function(xcb, "xcb_map_window");
  _map_window(k_native_connection_handle_, k_native_window_handle_);
  pfn_xcb_flush _flush = (pfn_xcb_flush)get_xcb_function(xcb, "xcb_flush");
  _flush(k_native_connection_handle_);

  return true;
}

typedef xcb_intern_atom_cookie_t (*pfn_xcb_intern_atom)(xcb_connection_t*,
                                                        uint8_t, uint16_t,
                                                        const char*);
typedef xcb_intern_atom_reply_t* (*pfn_xcb_intern_atom_reply)(
    xcb_connection_t*, xcb_intern_atom_cookie_t, xcb_generic_error_t**);
typedef xcb_generic_event_t* (*pfn_xcb_poll_for_event)(xcb_connection_t*);

void ProcessNativeWindowEvents() {
  void* xcb = get_xcb();
  static pfn_xcb_intern_atom _intern_atom =
      (pfn_xcb_intern_atom)get_xcb_function(xcb, "xcb_intern_atom");
  static xcb_intern_atom_cookie_t delete_cookie =
      _intern_atom(k_native_connection_handle_, 0, 16, "WM_DELETE_WINDOW");
  static pfn_xcb_intern_atom_reply _intern_atom_reply =
      (pfn_xcb_intern_atom_reply)get_xcb_function(xcb, "xcb_intern_atom_reply");
  static xcb_intern_atom_reply_t* delete_reply =
      _intern_atom_reply(k_native_connection_handle_, delete_cookie, 0);

  static pfn_xcb_poll_for_event _poll_for_event =
      (pfn_xcb_poll_for_event)get_xcb_function(xcb, "xcb_poll_for_event");
  xcb_generic_event_t* event;
  while ((event = _poll_for_event(k_native_connection_handle_))) {
    if ((event->response_type & 0x7f) == XCB_CLIENT_MESSAGE) {
      auto message = (xcb_client_message_event_t*)event;
      if (message->data.data32[0] == delete_reply->atom) {
        break;
      }
    }
  }
}

#endif

uint32_t inline GetMemoryIndex(
    const VkPhysicalDeviceMemoryProperties& properties,
    uint32_t required_index_bits,
    VkMemoryPropertyFlags required_property_flags) {
  uint32_t memory_index = 0;
  for (; memory_index < properties.memoryTypeCount; ++memory_index) {
    if (!(required_index_bits & (1 << memory_index))) {
      continue;
    }

    if ((properties.memoryTypes[memory_index].propertyFlags &
         required_property_flags) != required_property_flags) {
      continue;
    }
    break;
  }
  if (memory_index == properties.memoryTypeCount) {
    return static_cast<uint32_t>(-1);
  }
  return memory_index;
}

void usage() {
  std::cout << "Options: \n";
  std::cout << "-h=<height> Set desktop window height (default: 768)\n";
  std::cout << "-w=<width>  Set desktop window width (default: 1024)\n";
}

#ifdef __ANDROID__
int main_impl() {
  int argc = 0;
  char** argv = nullptr;
#else
int main(int argc, const char** argv) {
#endif
  int width = 1024;
  int height = 768;

  for (int i = 1; i < argc; i++) {
    std::string arg(argv[i]);
    if (arg.compare(0, 3, "-h=") == 0) {
      height = std::stoi(&(arg[3]));
    } else if (arg.compare(0, 3, "-w=") == 0) {
      width = std::stoi(&(arg[3]));
    } else {
      std::cout << "Unrecognized argument: " << arg << '\n';
      usage();
      return -1;
    }
  }

  if (!CreateNativeWindow(width, height)) {
    write_error(kOutHandle, "Exiting due to no available window");
    return -1;
  }

  kVulkanLibraryHandle = LoadVulkan();

#ifdef _WIN32
  PFN_vkGetInstanceProcAddr vkGetInstanceProcAddr =
      reinterpret_cast<PFN_vkGetInstanceProcAddr>(
          GetLibraryProcAddr(kVulkanLibraryHandle, "vkGetInstanceProcAddr"));
#elif defined(__linux__)  // Similar for Android
  PFN_vkGetInstanceProcAddr vkGetInstanceProcAddr =
      reinterpret_cast<PFN_vkGetInstanceProcAddr>(
          dlsym(kVulkanLibraryHandle, "vkGetInstanceProcAddr"));
#endif

#define REQUIRE_SUCCESS(fn)                          \
  do {                                               \
    if (VK_SUCCESS != fn) {                          \
      write_error(kOutHandle, "Vulkan Error: " #fn); \
      return -1;                                     \
    }                                                \
  } while (0)

#define LOAD_INSTANCE_FUNCTION(name, instance) \
  PFN_##name name =                            \
      reinterpret_cast<PFN_##name>(vkGetInstanceProcAddr(instance, #name));

  LOAD_INSTANCE_FUNCTION(vkCreateInstance, VK_NULL_HANDLE);
  LOAD_INSTANCE_FUNCTION(vkEnumerateInstanceExtensionProperties,
                         VK_NULL_HANDLE);

  uint32_t nExtensions = 0;
  REQUIRE_SUCCESS(
      vkEnumerateInstanceExtensionProperties(nullptr, &nExtensions, nullptr));
  uint32_t nRequiredExtensions = sizeof(kRequiredInstanceExtensions) /
                                 sizeof(kRequiredInstanceExtensions[0]);

  {
    std::vector<VkExtensionProperties> extension_properties(nExtensions);
    REQUIRE_SUCCESS(vkEnumerateInstanceExtensionProperties(
        nullptr, &nExtensions, extension_properties.data()));
    for (uint32_t i = 0; i < nRequiredExtensions; ++i) {
      bool found = false;
      for (auto& prop : extension_properties) {
        if (std::string(prop.extensionName) == kRequiredInstanceExtensions[i]) {
          found = true;
        }
      }
      if (!found) {
        write_error(kOutHandle, "Could not find all instance extensions");
      }
    }
  }

  VkInstance instance;
  {
    VkApplicationInfo app_info{VK_STRUCTURE_TYPE_APPLICATION_INFO,
                               nullptr,
                               "sample_app",
                               0,
                               "sample_engine",
                               0,
                               VK_MAKE_VERSION(1, 0, 0)};
    VkInstanceCreateInfo create_info{VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
                                     nullptr,
                                     0,
                                     &app_info,
                                     0,
                                     nullptr,
                                     nRequiredExtensions,
                                     kRequiredInstanceExtensions};
    REQUIRE_SUCCESS(vkCreateInstance(&create_info, nullptr, &instance));
  }

  VkSurfaceKHR surface;
#if _WIN32
  {
    LOAD_INSTANCE_FUNCTION(vkCreateWin32SurfaceKHR, instance);
    VkWin32SurfaceCreateInfoKHR create_info{
        VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR, nullptr, 0,
        kNativeHInstance, kNativeWindowHandle};
    REQUIRE_SUCCESS(
        vkCreateWin32SurfaceKHR(instance, &create_info, nullptr, &surface));
  }
#elif defined(__ANDROID__)
  {
    LOAD_INSTANCE_FUNCTION(vkCreateAndroidSurfaceKHR, instance);
    VkAndroidSurfaceCreateInfoKHR create_info{
        VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR, nullptr, 0,
        kANativeWindowHandle};
    REQUIRE_SUCCESS(
        vkCreateAndroidSurfaceKHR(instance, &create_info, nullptr, &surface));
  }
#elif defined(__linux__)
{
  LOAD_INSTANCE_FUNCTION(vkCreateXcbSurfaceKHR, instance);
  VkXcbSurfaceCreateInfoKHR create_info{
      VK_STRUCTURE_TYPE_XCB_SURFACE_CREATE_INFO_KHR, nullptr, 0,
      k_native_connection_handle_, k_native_window_handle_};
  REQUIRE_SUCCESS(
      vkCreateXcbSurfaceKHR(instance, &create_info, nullptr, &surface));
}
#endif
  VkSurfaceCapabilitiesKHR surface_capabilities = {};
  VkSurfaceFormatKHR surface_format = {};

  VkPhysicalDevice physical_device = {};
  uint32_t queue_family_index = static_cast<uint32_t>(-1);

  {
    LOAD_INSTANCE_FUNCTION(vkEnumeratePhysicalDevices, instance);
    uint32_t nPhysicalDevices;
    REQUIRE_SUCCESS(
        vkEnumeratePhysicalDevices(instance, &nPhysicalDevices, nullptr));
    std::vector<VkPhysicalDevice> physical_devices(nPhysicalDevices);
    REQUIRE_SUCCESS(vkEnumeratePhysicalDevices(instance, &nPhysicalDevices,
                                               physical_devices.data()));

    LOAD_INSTANCE_FUNCTION(vkGetPhysicalDeviceSurfaceSupportKHR, instance);
    LOAD_INSTANCE_FUNCTION(vkGetPhysicalDeviceQueueFamilyProperties, instance);
    LOAD_INSTANCE_FUNCTION(vkGetPhysicalDeviceSurfaceCapabilitiesKHR, instance);
    LOAD_INSTANCE_FUNCTION(vkGetPhysicalDeviceSurfaceFormatsKHR, instance);

    uint32_t i;
    for (i = 0; i < nPhysicalDevices; ++i) {
      uint32_t nQueueProperties = 0;
      vkGetPhysicalDeviceQueueFamilyProperties(physical_devices[i],
                                               &nQueueProperties, nullptr);
      std::vector<VkQueueFamilyProperties> queue_properties(nQueueProperties);
      vkGetPhysicalDeviceQueueFamilyProperties(
          physical_devices[i], &nQueueProperties, queue_properties.data());
      for (uint32_t j = 0; j < nQueueProperties; ++j) {
        VkBool32 present_supported;
        REQUIRE_SUCCESS(vkGetPhysicalDeviceSurfaceSupportKHR(
            physical_devices[i], j, surface, &present_supported));
        if (!present_supported) {
          continue;
        }
        uint32_t nSurfaceFormats;
        REQUIRE_SUCCESS(vkGetPhysicalDeviceSurfaceFormatsKHR(
            physical_devices[i], surface, &nSurfaceFormats, nullptr));
        std::vector<VkSurfaceFormatKHR> surface_formats(nSurfaceFormats);
        REQUIRE_SUCCESS(vkGetPhysicalDeviceSurfaceFormatsKHR(
            physical_devices[i], surface, &nSurfaceFormats,
            surface_formats.data()));
        if (nSurfaceFormats < 1) {
          continue;
        }
        surface_format = surface_formats[0];
        REQUIRE_SUCCESS(vkGetPhysicalDeviceSurfaceCapabilitiesKHR(
            physical_devices[i], surface, &surface_capabilities));
        if (surface_capabilities.maxImageCount ==
            1) {  // If it is 0, then infinite, if > 1 then we are ok.
          continue;
        }
        if (queue_properties[j].queueFlags & VK_QUEUE_GRAPHICS_BIT) {
          queue_family_index = j;
          break;
        }
      }
      if (queue_family_index != static_cast<uint32_t>(-1)) {
        break;
      }
    }
    if (i == nPhysicalDevices) {
      write_error(kOutHandle,
                  "Could not find physical devices that could present on the "
                  "graphics queue");
    }
    physical_device = physical_devices[i];
  }

  VkPhysicalDeviceMemoryProperties memory_properties;
  {
    LOAD_INSTANCE_FUNCTION(vkGetPhysicalDeviceMemoryProperties, instance);
    vkGetPhysicalDeviceMemoryProperties(physical_device, &memory_properties);
  }

  LOAD_INSTANCE_FUNCTION(vkCreateDevice, instance);
  VkDevice device;
  {
    float priority = 1.0f;
    VkDeviceQueueCreateInfo queue_create_info{
        VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
        nullptr,
        0,
        queue_family_index,
        1,
        &priority,
    };

    const char* device_extensions[] = {VK_KHR_SWAPCHAIN_EXTENSION_NAME};

    VkDeviceCreateInfo create_info{
        VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
        nullptr,
        0,
        1,
        &queue_create_info,
        0,
        nullptr,
        1,
        device_extensions,
        nullptr,
    };
    REQUIRE_SUCCESS(
        vkCreateDevice(physical_device, &create_info, nullptr, &device));
  }

  LOAD_INSTANCE_FUNCTION(vkGetDeviceProcAddr, instance)
#undef LOAD_INSTANCE_FUNCTION

#define LOAD_DEVICE_FUNCTION(name) \
  PFN_##name name =                \
      reinterpret_cast<PFN_##name>(vkGetDeviceProcAddr(device, #name));

  VkQueue queue;
  LOAD_DEVICE_FUNCTION(vkGetDeviceQueue)
  vkGetDeviceQueue(device, queue_family_index, 0, &queue);

#ifdef VK_USE_PLATFORM_ANDROID_KHR
  constexpr uint32_t kDesiredImageCount = 3;
#else
  constexpr uint32_t kDesiredImageCount = 2;
#endif

  VkSwapchainKHR swapchain;
  {
    LOAD_DEVICE_FUNCTION(vkCreateSwapchainKHR);
    VkSwapchainCreateInfoKHR create_info{
        VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
        nullptr,
        0,
        surface,
        std::max(surface_capabilities.minImageCount, kDesiredImageCount),
        surface_format.format,
        surface_format.colorSpace,
        surface_capabilities.currentExtent,
        1,
        VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
        VK_SHARING_MODE_EXCLUSIVE,
        0,
        nullptr,
        VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR,
        VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR,
        VK_PRESENT_MODE_FIFO_KHR,
        VK_FALSE,
        VK_NULL_HANDLE};

    REQUIRE_SUCCESS(
        vkCreateSwapchainKHR(device, &create_info, nullptr, &swapchain));
  }
  LOAD_DEVICE_FUNCTION(vkGetSwapchainImagesKHR);
  std::vector<VkImage> swapchain_images;
  {
    uint32_t nSwapchainImages;
    REQUIRE_SUCCESS(
        vkGetSwapchainImagesKHR(device, swapchain, &nSwapchainImages, nullptr));
    swapchain_images.resize(nSwapchainImages);
    REQUIRE_SUCCESS(vkGetSwapchainImagesKHR(
        device, swapchain, &nSwapchainImages, swapchain_images.data()));
  }

  // Setup Functions
  LOAD_DEVICE_FUNCTION(vkCreateBuffer);
  LOAD_DEVICE_FUNCTION(vkCreateImage);
  LOAD_DEVICE_FUNCTION(vkAllocateMemory);
  LOAD_DEVICE_FUNCTION(vkGetBufferMemoryRequirements);
  LOAD_DEVICE_FUNCTION(vkGetImageMemoryRequirements);
  LOAD_DEVICE_FUNCTION(vkBindBufferMemory);
  LOAD_DEVICE_FUNCTION(vkBindImageMemory);
  LOAD_DEVICE_FUNCTION(vkCreateSampler);
  LOAD_DEVICE_FUNCTION(vkCreateImageView);
  LOAD_DEVICE_FUNCTION(vkCreateDescriptorSetLayout);
  LOAD_DEVICE_FUNCTION(vkCreatePipelineLayout);
  LOAD_DEVICE_FUNCTION(vkCreateRenderPass);
  LOAD_DEVICE_FUNCTION(vkCreateShaderModule);
  LOAD_DEVICE_FUNCTION(vkCreateGraphicsPipelines);
  LOAD_DEVICE_FUNCTION(vkCreateDescriptorPool);
  LOAD_DEVICE_FUNCTION(vkAllocateDescriptorSets);
  LOAD_DEVICE_FUNCTION(vkUpdateDescriptorSets);
  LOAD_DEVICE_FUNCTION(vkCreateSemaphore);
  LOAD_DEVICE_FUNCTION(vkCreateFence);
  LOAD_DEVICE_FUNCTION(vkWaitForFences);
  LOAD_DEVICE_FUNCTION(vkResetFences);
  LOAD_DEVICE_FUNCTION(vkQueueWaitIdle);
  LOAD_DEVICE_FUNCTION(vkFlushMappedMemoryRanges);
  LOAD_DEVICE_FUNCTION(vkBeginCommandBuffer);
  LOAD_DEVICE_FUNCTION(vkEndCommandBuffer);
  LOAD_DEVICE_FUNCTION(vkResetCommandPool);
  LOAD_DEVICE_FUNCTION(vkQueueSubmit);
  LOAD_DEVICE_FUNCTION(vkCmdCopyBuffer);
  LOAD_DEVICE_FUNCTION(vkCmdCopyBufferToImage);
  LOAD_DEVICE_FUNCTION(vkCmdDrawIndexed);
  LOAD_DEVICE_FUNCTION(vkCmdPipelineBarrier);
  LOAD_DEVICE_FUNCTION(vkCmdBindPipeline);
  LOAD_DEVICE_FUNCTION(vkCmdBindVertexBuffers);
  LOAD_DEVICE_FUNCTION(vkCmdBindIndexBuffer);
  LOAD_DEVICE_FUNCTION(vkCmdBindDescriptorSets);
  LOAD_DEVICE_FUNCTION(vkCmdBeginRenderPass);
  LOAD_DEVICE_FUNCTION(vkCmdEndRenderPass);
  LOAD_DEVICE_FUNCTION(vkMapMemory);
  LOAD_DEVICE_FUNCTION(vkUnmapMemory);
  LOAD_DEVICE_FUNCTION(vkCreateCommandPool);
  LOAD_DEVICE_FUNCTION(vkAllocateCommandBuffers);
  LOAD_DEVICE_FUNCTION(vkCreateFramebuffer);
  LOAD_DEVICE_FUNCTION(vkDestroyFramebuffer);
  LOAD_DEVICE_FUNCTION(vkAcquireNextImageKHR);
  LOAD_DEVICE_FUNCTION(vkQueuePresentKHR);
#undef LOAD_DEVICE_FUNCTION
  // Immutable Data
  VkBuffer vertex_buffer;
  VkDeviceMemory vertex_buffer_memory;
  VkBuffer index_buffer;
  VkDeviceMemory index_buffer_memory;
  VkImage texture;
  VkDeviceMemory texture_memory;
  VkSampler sampler;
  VkImageView image_view;

  VkDescriptorSetLayout descriptor_set_layout;
  VkPipelineLayout pipeline_layout;
  VkRenderPass render_pass;

  VkShaderModule vertex_shader_module;
  VkShaderModule fragment_shader_module;
  VkPipeline graphics_pipeline;

  VkDescriptorPool descriptor_pool;

  // Per-buffer mutable Data
  VkBuffer uniform_buffers[kBufferingCount];
  VkDeviceMemory uniform_buffer_memories[kBufferingCount];
  VkImage depth_buffers[kBufferingCount];
  VkDeviceMemory depth_buffer_memories[kBufferingCount];
  VkImageView depth_buffer_views[kBufferingCount];
  VkDescriptorSet descriptor_sets[kBufferingCount];
  VkCommandPool command_pools[kBufferingCount];
  VkCommandBuffer render_command_buffers[kBufferingCount];
  VkFence ready_fences[kBufferingCount];
  VkSemaphore swapchain_image_ready_semaphores[kBufferingCount];
  VkSemaphore render_done_semaphores[kBufferingCount];

  // Transient things: Framebuffer for now.
  VkFramebuffer framebuffers[kBufferingCount] = {};

  // Per swapchain image mutableData;
  std::vector<VkImageView> swapchain_views;

  // Create the vertex buffer && back it with memory
  {
    VkBufferCreateInfo create_info{
        VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
        nullptr,
        0,
        cube::model.num_vertices * (3 + 2 + 3) * sizeof(float),
        VK_BUFFER_USAGE_TRANSFER_DST_BIT | VK_BUFFER_USAGE_VERTEX_BUFFER_BIT,
        VK_SHARING_MODE_EXCLUSIVE,
        0,
        nullptr};
    REQUIRE_SUCCESS(
        vkCreateBuffer(device, &create_info, nullptr, &vertex_buffer));

    VkMemoryRequirements memory_requirements;
    vkGetBufferMemoryRequirements(device, vertex_buffer, &memory_requirements);

    uint32_t memory_index =
        GetMemoryIndex(memory_properties, memory_requirements.memoryTypeBits,
                       VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    if (memory_index == static_cast<uint32_t>(-1)) {
      write_error(kOutHandle, "Could not find memory index for Vertex Buffer");
      return -1;
    }

    VkMemoryAllocateInfo allocate_info{VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                                       nullptr, memory_requirements.size,
                                       memory_index};

    REQUIRE_SUCCESS(vkAllocateMemory(device, &allocate_info, nullptr,
                                     &vertex_buffer_memory));
    REQUIRE_SUCCESS(
        vkBindBufferMemory(device, vertex_buffer, vertex_buffer_memory, 0));
  }

  // Create the index buffer
  {
    VkBufferCreateInfo create_info{
        VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
        nullptr,
        0,
        cube::model.num_indices * sizeof(uint32_t),
        VK_BUFFER_USAGE_TRANSFER_DST_BIT | VK_BUFFER_USAGE_INDEX_BUFFER_BIT,
        VK_SHARING_MODE_EXCLUSIVE,
        0,
        nullptr};
    REQUIRE_SUCCESS(
        vkCreateBuffer(device, &create_info, nullptr, &index_buffer));

    VkMemoryRequirements memory_requirements;
    vkGetBufferMemoryRequirements(device, index_buffer, &memory_requirements);

    uint32_t memory_index =
        GetMemoryIndex(memory_properties, memory_requirements.memoryTypeBits,
                       VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    if (memory_index == static_cast<uint32_t>(-1)) {
      write_error(kOutHandle, "Could not find memory index for Index Buffer");
      return -1;
    }

    VkMemoryAllocateInfo allocate_info{VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                                       nullptr, memory_requirements.size,
                                       memory_index};

    REQUIRE_SUCCESS(vkAllocateMemory(device, &allocate_info, nullptr,
                                     &index_buffer_memory));
    REQUIRE_SUCCESS(
        vkBindBufferMemory(device, index_buffer, index_buffer_memory, 0));
  }

  {
    VkImageCreateInfo create_info{
        VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
        nullptr,
        0,
        VK_IMAGE_TYPE_2D,
        icon::texture.format,
        VkExtent3D{static_cast<uint32_t>(icon::texture.width),
                   static_cast<uint32_t>(icon::texture.height), 1},
        1,
        1,
        VK_SAMPLE_COUNT_1_BIT,
        VK_IMAGE_TILING_OPTIMAL,
        VK_IMAGE_USAGE_TRANSFER_DST_BIT | VK_IMAGE_USAGE_SAMPLED_BIT,
        VK_SHARING_MODE_EXCLUSIVE,
        0,
        nullptr,
        VK_IMAGE_LAYOUT_UNDEFINED};

    REQUIRE_SUCCESS(vkCreateImage(device, &create_info, nullptr, &texture));

    VkMemoryRequirements memory_requirements;
    vkGetImageMemoryRequirements(device, texture, &memory_requirements);

    uint32_t memory_index =
        GetMemoryIndex(memory_properties, memory_requirements.memoryTypeBits,
                       VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    if (memory_index == static_cast<uint32_t>(-1)) {
      write_error(kOutHandle, "Could not find memory index for Uniform buffer");
      return -1;
    }

    VkMemoryAllocateInfo allocate_info{VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                                       nullptr, memory_requirements.size,
                                       memory_index};

    REQUIRE_SUCCESS(
        vkAllocateMemory(device, &allocate_info, nullptr, &texture_memory));
    REQUIRE_SUCCESS(vkBindImageMemory(device, texture, texture_memory, 0));
  }

  {
    VkSamplerCreateInfo create_info{VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
                                    nullptr,
                                    0,
                                    VK_FILTER_LINEAR,
                                    VK_FILTER_LINEAR,
                                    VK_SAMPLER_MIPMAP_MODE_NEAREST,
                                    VK_SAMPLER_ADDRESS_MODE_REPEAT,
                                    VK_SAMPLER_ADDRESS_MODE_REPEAT,
                                    VK_SAMPLER_ADDRESS_MODE_REPEAT,
                                    0,
                                    false,
                                    0,
                                    false,
                                    VK_COMPARE_OP_ALWAYS,
                                    0,
                                    0,
                                    VK_BORDER_COLOR_FLOAT_TRANSPARENT_BLACK,
                                    false};
    REQUIRE_SUCCESS(vkCreateSampler(device, &create_info, nullptr, &sampler));
  }

  {
    VkImageViewCreateInfo create_info{
        VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
        nullptr,
        0,
        texture,
        VK_IMAGE_VIEW_TYPE_2D,
        icon::texture.format,
        VkComponentMapping{
            VK_COMPONENT_SWIZZLE_IDENTITY, VK_COMPONENT_SWIZZLE_IDENTITY,
            VK_COMPONENT_SWIZZLE_IDENTITY, VK_COMPONENT_SWIZZLE_IDENTITY},
        VkImageSubresourceRange{VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1}};
    REQUIRE_SUCCESS(
        vkCreateImageView(device, &create_info, nullptr, &image_view));
  }

  {
    VkDescriptorSetLayoutBinding bindings[3] = {
        VkDescriptorSetLayoutBinding{0, VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, 1,
                                     VK_SHADER_STAGE_VERTEX_BIT, nullptr},
        VkDescriptorSetLayoutBinding{1, VK_DESCRIPTOR_TYPE_SAMPLER, 1,
                                     VK_SHADER_STAGE_FRAGMENT_BIT, nullptr},
        VkDescriptorSetLayoutBinding{2, VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE, 1,
                                     VK_SHADER_STAGE_FRAGMENT_BIT, nullptr}};
    VkDescriptorSetLayoutCreateInfo create_info{
        VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO, nullptr, 0, 3,
        bindings};

    REQUIRE_SUCCESS(vkCreateDescriptorSetLayout(device, &create_info, nullptr,
                                                &descriptor_set_layout));
  }

  {
    VkPipelineLayoutCreateInfo create_info{
        VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
        nullptr,
        0,
        1,
        &descriptor_set_layout,
        0,
        nullptr};
    REQUIRE_SUCCESS(vkCreatePipelineLayout(device, &create_info, nullptr,
                                           &pipeline_layout));
  }

  {
    VkAttachmentDescription attachments[2] = {
        VkAttachmentDescription{
            0,
            surface_format.format,
            VK_SAMPLE_COUNT_1_BIT,
            VK_ATTACHMENT_LOAD_OP_CLEAR,
            VK_ATTACHMENT_STORE_OP_STORE,
            VK_ATTACHMENT_LOAD_OP_DONT_CARE,
            VK_ATTACHMENT_STORE_OP_DONT_CARE,
            VK_IMAGE_LAYOUT_UNDEFINED,
            VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
        },
        VkAttachmentDescription{
            0, kDepthFormat, VK_SAMPLE_COUNT_1_BIT, VK_ATTACHMENT_LOAD_OP_CLEAR,
            VK_ATTACHMENT_STORE_OP_DONT_CARE, VK_ATTACHMENT_LOAD_OP_DONT_CARE,
            VK_ATTACHMENT_STORE_OP_DONT_CARE, VK_IMAGE_LAYOUT_UNDEFINED,
            VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL}};

    VkAttachmentReference color_attachment{
        0, VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL};
    VkAttachmentReference depth_attachment{
        1, VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL};

    VkSubpassDescription subpass{0,       VK_PIPELINE_BIND_POINT_GRAPHICS,
                                 0,       nullptr,
                                 1,       &color_attachment,
                                 nullptr, &depth_attachment,
                                 0,       nullptr};

    VkRenderPassCreateInfo create_info{
        VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
        nullptr,
        0,
        2,
        attachments,
        1,
        &subpass,
        0,
        nullptr};

    REQUIRE_SUCCESS(
        vkCreateRenderPass(device, &create_info, nullptr, &render_pass));
  }

  {
    VkShaderModuleCreateInfo create_info{
        VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, nullptr, 0,
        sizeof(vertex_shader), vertex_shader};

    REQUIRE_SUCCESS(vkCreateShaderModule(device, &create_info, nullptr,
                                         &vertex_shader_module));
  }

  {
    VkShaderModuleCreateInfo create_info{
        VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, nullptr, 0,
        sizeof(fragment_shader), fragment_shader};

    REQUIRE_SUCCESS(vkCreateShaderModule(device, &create_info, nullptr,
                                         &fragment_shader_module));
  }

  {
    VkPipelineShaderStageCreateInfo stage_infos[2] = {
        VkPipelineShaderStageCreateInfo{
            VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, nullptr, 0,
            VK_SHADER_STAGE_VERTEX_BIT, vertex_shader_module, "main", nullptr},
        VkPipelineShaderStageCreateInfo{
            VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO, nullptr, 0,
            VK_SHADER_STAGE_FRAGMENT_BIT, fragment_shader_module, "main",
            nullptr}};

    VkVertexInputBindingDescription bindings{0, 8 * sizeof(float),
                                             VK_VERTEX_INPUT_RATE_VERTEX};

    VkVertexInputAttributeDescription attributes[3]{
        VkVertexInputAttributeDescription{0, 0, VK_FORMAT_R32G32B32_SFLOAT, 0},
        VkVertexInputAttributeDescription{1, 0, VK_FORMAT_R32G32_SFLOAT,
                                          3 * sizeof(float)},
        VkVertexInputAttributeDescription{2, 0, VK_FORMAT_R32G32B32_SFLOAT,
                                          (3 + 2) * sizeof(float)}};

    VkPipelineVertexInputStateCreateInfo vertex_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
        nullptr,
        0,
        1,
        &bindings,
        3,
        attributes};

    VkPipelineInputAssemblyStateCreateInfo input_assembly_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
        nullptr,
        0,
        VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
        false,
    };

    VkViewport viewport{
        0.0f,
        0.0f,
        static_cast<float>(surface_capabilities.currentExtent.width),
        static_cast<float>(surface_capabilities.currentExtent.height),
        0.0f,
        1.0f,
    };
    VkRect2D scissor{{0, 0},
                     {
                         surface_capabilities.currentExtent.width,
                         surface_capabilities.currentExtent.height,
                     }};

    VkPipelineViewportStateCreateInfo viewport_state_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
        nullptr,
        0,
        1,
        &viewport,
        1,
        &scissor};

    VkPipelineRasterizationStateCreateInfo rasterization_state_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
        nullptr,
        0,
        false,
        false,
        VK_POLYGON_MODE_FILL,
        VK_CULL_MODE_BACK_BIT,
        VK_FRONT_FACE_CLOCKWISE,
        false,
        0.0f,
        0.0f,
        0.0f,
        1.0f};

    VkPipelineMultisampleStateCreateInfo multisample_state_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
        nullptr,
        0,
        VK_SAMPLE_COUNT_1_BIT,
        false,
        0,
        nullptr,
        false,
        false,
    };

    VkPipelineDepthStencilStateCreateInfo depth_stencil_state_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
        nullptr,
        0,
        true,
        true,
        VK_COMPARE_OP_LESS,
        false,
        false,
        VkStencilOpState{},
        VkStencilOpState{},
        0.0f,
        1.0f};

    VkPipelineColorBlendAttachmentState blend_attachment_state{
        VK_FALSE,
        VK_BLEND_FACTOR_ZERO,
        VK_BLEND_FACTOR_ONE,
        VK_BLEND_OP_ADD,
        VK_BLEND_FACTOR_ZERO,
        VK_BLEND_FACTOR_ONE,
        VK_BLEND_OP_ADD,
        VK_COLOR_COMPONENT_R_BIT | VK_COLOR_COMPONENT_G_BIT |
            VK_COLOR_COMPONENT_B_BIT | VK_COLOR_COMPONENT_A_BIT};

    VkPipelineColorBlendStateCreateInfo color_blend_state_create_info{
        VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
        nullptr,
        0,
        false,
        VK_LOGIC_OP_MAX_ENUM,
        1,
        &blend_attachment_state,
        {0, 0, 0, 0}};

    VkGraphicsPipelineCreateInfo create_info{
        VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
        nullptr,
        0,
        2,
        stage_infos,
        &vertex_create_info,
        &input_assembly_create_info,
        nullptr,
        &viewport_state_create_info,
        &rasterization_state_create_info,
        &multisample_state_create_info,
        &depth_stencil_state_create_info,
        &color_blend_state_create_info,
        nullptr,
        pipeline_layout,
        render_pass,
        0,
        VK_NULL_HANDLE,
        0};

    REQUIRE_SUCCESS(vkCreateGraphicsPipelines(
        device, VK_NULL_HANDLE, 1, &create_info, nullptr, &graphics_pipeline));
  }

  {
    VkDescriptorPoolSize sizes[3] = {VkDescriptorPoolSize{
                                         VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
                                         kBufferingCount,
                                     },
                                     VkDescriptorPoolSize{
                                         VK_DESCRIPTOR_TYPE_SAMPLER,
                                         kBufferingCount,
                                     },
                                     VkDescriptorPoolSize{
                                         VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
                                         kBufferingCount,
                                     }};
    VkDescriptorPoolCreateInfo create_info{
        VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
        nullptr,
        0,
        kBufferingCount,
        3,
        sizes};
    REQUIRE_SUCCESS(vkCreateDescriptorPool(device, &create_info, nullptr,
                                           &descriptor_pool));
  }

  // Create the per-buffer resources
  for (size_t i = 0; i < kBufferingCount; ++i) {
    {  // Create the uniform buffers
      VkBufferCreateInfo create_info{VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
                                     nullptr,
                                     0,
                                     4 * 4 * 2 * sizeof(float),
                                     VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
                                     VK_SHARING_MODE_EXCLUSIVE,
                                     0,
                                     nullptr};
      REQUIRE_SUCCESS(
          vkCreateBuffer(device, &create_info, nullptr, &uniform_buffers[i]));

      VkMemoryRequirements memory_requirements;
      vkGetBufferMemoryRequirements(device, uniform_buffers[i],
                                    &memory_requirements);

      uint32_t memory_index =
          GetMemoryIndex(memory_properties, memory_requirements.memoryTypeBits,
                         VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT);
      if (memory_index == static_cast<uint32_t>(-1)) {
        write_error(kOutHandle,
                    "Could not find memory index for Uniform buffer");
        return -1;
      }

      VkMemoryAllocateInfo allocate_info{VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                                         nullptr, memory_requirements.size,
                                         memory_index};

      REQUIRE_SUCCESS(vkAllocateMemory(device, &allocate_info, nullptr,
                                       &uniform_buffer_memories[i]));
      REQUIRE_SUCCESS(vkBindBufferMemory(device, uniform_buffers[i],
                                         uniform_buffer_memories[i], 0));
    }
    {  // Create the depth buffers
      VkImageCreateInfo create_info{
          VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
          nullptr,
          0,
          VK_IMAGE_TYPE_2D,
          kDepthFormat,
          VkExtent3D{surface_capabilities.currentExtent.width,
                     surface_capabilities.currentExtent.height, 1},
          1,
          1,
          VK_SAMPLE_COUNT_1_BIT,
          VK_IMAGE_TILING_OPTIMAL,
          VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT,
          VK_SHARING_MODE_EXCLUSIVE,
          0,
          nullptr,
          VK_IMAGE_LAYOUT_UNDEFINED};

      REQUIRE_SUCCESS(
          vkCreateImage(device, &create_info, nullptr, &depth_buffers[i]));

      VkMemoryRequirements memory_requirements;
      vkGetImageMemoryRequirements(device, depth_buffers[i],
                                   &memory_requirements);

      uint32_t memory_index =
          GetMemoryIndex(memory_properties, memory_requirements.memoryTypeBits,
                         VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
      if (memory_index == static_cast<uint32_t>(-1)) {
        write_error(kOutHandle,
                    "Could not find memory index for Uniform buffer");
        return -1;
      }

      VkMemoryAllocateInfo allocate_info{VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                                         nullptr, memory_requirements.size,
                                         memory_index};

      REQUIRE_SUCCESS(vkAllocateMemory(device, &allocate_info, nullptr,
                                       &depth_buffer_memories[i]));
      REQUIRE_SUCCESS(vkBindImageMemory(device, depth_buffers[i],
                                        depth_buffer_memories[i], 0));
    }
    {
      VkImageViewCreateInfo create_info{
          VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
          nullptr,
          0,
          depth_buffers[i],
          VK_IMAGE_VIEW_TYPE_2D,
          kDepthFormat,
          VkComponentMapping{
              VK_COMPONENT_SWIZZLE_IDENTITY, VK_COMPONENT_SWIZZLE_IDENTITY,
              VK_COMPONENT_SWIZZLE_IDENTITY, VK_COMPONENT_SWIZZLE_IDENTITY},
          VkImageSubresourceRange{VK_IMAGE_ASPECT_DEPTH_BIT, 0, 1, 0, 1}};
      REQUIRE_SUCCESS(vkCreateImageView(device, &create_info, nullptr,
                                        &depth_buffer_views[i]));
    }
    {
      VkDescriptorSetAllocateInfo allocate_info{
          VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO, nullptr,
          descriptor_pool, 1, &descriptor_set_layout};
      REQUIRE_SUCCESS(vkAllocateDescriptorSets(device, &allocate_info,
                                               &descriptor_sets[i]));

      VkDescriptorBufferInfo buffer_info{uniform_buffers[i], 0, VK_WHOLE_SIZE};

      VkDescriptorImageInfo sampler_info{
          sampler,
          VK_NULL_HANDLE,
          VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
      };

      VkDescriptorImageInfo view_info{
          VK_NULL_HANDLE,
          image_view,
          VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
      };

      VkWriteDescriptorSet writes[3]{
          VkWriteDescriptorSet{VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr,
                               descriptor_sets[i], 0, 0, 1,
                               VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER, nullptr,
                               &buffer_info, nullptr},
          VkWriteDescriptorSet{VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr,
                               descriptor_sets[i], 1, 0, 1,
                               VK_DESCRIPTOR_TYPE_SAMPLER, &sampler_info,
                               nullptr, nullptr},
          VkWriteDescriptorSet{VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET, nullptr,
                               descriptor_sets[i], 2, 0, 1,
                               VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE, &view_info,
                               nullptr, nullptr},
      };

      vkUpdateDescriptorSets(device, 3, writes, 0, nullptr);
    }

    {
      VkCommandPoolCreateInfo create_info{
          VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, nullptr, 0,
          queue_family_index};

      REQUIRE_SUCCESS(vkCreateCommandPool(device, &create_info, nullptr,
                                          &command_pools[i]));
    }

    {
      VkCommandBufferAllocateInfo allocate_info{
          VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, nullptr,
          command_pools[i], VK_COMMAND_BUFFER_LEVEL_PRIMARY, 1};
      REQUIRE_SUCCESS(vkAllocateCommandBuffers(device, &allocate_info,
                                               &render_command_buffers[i]));
    }

    {
      VkFenceCreateInfo create_info{VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
                                    nullptr, 0};
      REQUIRE_SUCCESS(
          vkCreateFence(device, &create_info, nullptr, &ready_fences[i]));
    }

    {
      VkSemaphoreCreateInfo create_info{VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO,
                                        nullptr, 0};
      REQUIRE_SUCCESS(vkCreateSemaphore(device, &create_info, nullptr,
                                        &swapchain_image_ready_semaphores[i]));
      REQUIRE_SUCCESS(vkCreateSemaphore(device, &create_info, nullptr,
                                        &render_done_semaphores[i]));
    }
  }

  {
    // Staging Buffer:
    // Stage all necessary resources
    VkBuffer staging_buffer;
    VkDeviceMemory staging_buffer_memory;
    {
      VkBufferCreateInfo create_info{
          VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
          nullptr,
          0,
          cube::model.num_indices * sizeof(uint32_t) +
              cube::model.num_vertices * (3 + 2 + 3) * sizeof(float) +
              sizeof(icon::texture.data),
          VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
          VK_SHARING_MODE_EXCLUSIVE,
          0,
          nullptr};
      REQUIRE_SUCCESS(
          vkCreateBuffer(device, &create_info, nullptr, &staging_buffer));

      VkMemoryRequirements memory_requirements;
      vkGetBufferMemoryRequirements(device, staging_buffer,
                                    &memory_requirements);

      uint32_t memory_index =
          GetMemoryIndex(memory_properties, memory_requirements.memoryTypeBits,
                         VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT);
      if (memory_index == static_cast<uint32_t>(-1)) {
        write_error(kOutHandle,
                    "Could not find memory index for Staging Buffer");
        return -1;
      }

      VkMemoryAllocateInfo allocate_info{VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                                         nullptr, memory_requirements.size,
                                         memory_index};

      REQUIRE_SUCCESS(vkAllocateMemory(device, &allocate_info, nullptr,
                                       &staging_buffer_memory));
      REQUIRE_SUCCESS(
          vkBindBufferMemory(device, staging_buffer, staging_buffer_memory, 0));
    }

    void* staging_buffer_location_vp;
    REQUIRE_SUCCESS(vkMapMemory(device, staging_buffer_memory, 0, VK_WHOLE_SIZE,
                                0, &staging_buffer_location_vp));
    char* staging_buffer_location =
        static_cast<char*>(staging_buffer_location_vp);
    // Vertex Buffer
    const uint32_t kVertexSize = (3 + 2 + 3) * sizeof(float);
    for (size_t i = 0; i < cube::model.num_vertices; ++i) {
      memcpy(staging_buffer_location + i * kVertexSize,
             cube::model.positions + i * 3, 3 * sizeof(float));
      memcpy(staging_buffer_location + i * kVertexSize + 3 * sizeof(float),
             cube::model.uv + i * 2, 2 * sizeof(float));
      memcpy(
          staging_buffer_location + i * kVertexSize + (3 + 2) * sizeof(float),
          cube::model.normals + i * 3, 3 * sizeof(float));
    }
    const uint32_t kVertexBufferSize = kVertexSize * cube::model.num_vertices;
    size_t index_buffer_offset = kVertexBufferSize;
    memcpy(staging_buffer_location + index_buffer_offset, cube::model.indices,
           cube::model.num_indices * sizeof(uint32_t));

    const uint32_t kIndexBufferSize =
        sizeof(uint32_t) * cube::model.num_indices;
    size_t image_offset = kVertexBufferSize + kIndexBufferSize;
    memcpy(staging_buffer_location + image_offset, icon::texture.data,
           sizeof(icon::texture.data));

    VkMappedMemoryRange range{VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, nullptr,
                              staging_buffer_memory, 0, VK_WHOLE_SIZE};
    REQUIRE_SUCCESS(vkFlushMappedMemoryRanges(device, 1, &range));
    VkCommandPool staging_command_pool;
    VkCommandBuffer staging_command_buffer;
    {
      VkCommandPoolCreateInfo create_info{
          VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, nullptr, 0,
          queue_family_index};

      REQUIRE_SUCCESS(vkCreateCommandPool(device, &create_info, nullptr,
                                          &staging_command_pool));
    }

    {
      VkCommandBufferAllocateInfo allocate_info{
          VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, nullptr,
          staging_command_pool, VK_COMMAND_BUFFER_LEVEL_PRIMARY, 1};
      REQUIRE_SUCCESS(vkAllocateCommandBuffers(device, &allocate_info,
                                               &staging_command_buffer));
    }

    {
      VkCommandBufferBeginInfo begin_info{
          VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, nullptr,
          VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT, nullptr};
      REQUIRE_SUCCESS(
          vkBeginCommandBuffer(staging_command_buffer, &begin_info));

      VkBufferMemoryBarrier buffer_barriers[] = {
          {// Staging image
           VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, nullptr,
           VK_ACCESS_HOST_WRITE_BIT, VK_ACCESS_TRANSFER_READ_BIT,
           queue_family_index, queue_family_index, staging_buffer, 0,
           VK_WHOLE_SIZE},
          {VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, nullptr,
           VK_ACCESS_HOST_WRITE_BIT, VK_ACCESS_TRANSFER_WRITE_BIT,
           queue_family_index, queue_family_index, vertex_buffer, 0,
           VK_WHOLE_SIZE},
          {VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, nullptr,
           VK_ACCESS_HOST_WRITE_BIT, VK_ACCESS_TRANSFER_WRITE_BIT,
           queue_family_index, queue_family_index, index_buffer, 0,
           VK_WHOLE_SIZE}};

      VkImageMemoryBarrier image_barrier{
          VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
          nullptr,
          VK_ACCESS_HOST_WRITE_BIT,
          VK_ACCESS_TRANSFER_WRITE_BIT,
          VK_IMAGE_LAYOUT_UNDEFINED,
          VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
          queue_family_index,
          queue_family_index,
          texture,
          VkImageSubresourceRange{VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1}};

      vkCmdPipelineBarrier(
          staging_command_buffer,
          VK_PIPELINE_STAGE_HOST_BIT | VK_PIPELINE_STAGE_TRANSFER_BIT,
          VK_PIPELINE_STAGE_TRANSFER_BIT, 0, 0, nullptr, 3, buffer_barriers, 1,
          &image_barrier);

      VkBufferCopy vertex_copy{0, 0, kVertexBufferSize};

      vkCmdCopyBuffer(staging_command_buffer, staging_buffer, vertex_buffer, 1,
                      &vertex_copy);

      VkBufferCopy index_copy{kVertexBufferSize, 0, kIndexBufferSize};

      vkCmdCopyBuffer(staging_command_buffer, staging_buffer, index_buffer, 1,
                      &index_copy);

      VkBufferImageCopy texture_copy{
          kVertexBufferSize + kIndexBufferSize,
          0,
          0,
          VkImageSubresourceLayers{VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1},
          VkOffset3D{0, 0, 0},
          VkExtent3D{static_cast<uint32_t>(icon::texture.width),
                     static_cast<uint32_t>(icon::texture.height), 1}};

      vkCmdCopyBufferToImage(staging_command_buffer, staging_buffer, texture,
                             VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1,
                             &texture_copy);

      buffer_barriers[1].srcAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;
      buffer_barriers[2].srcAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;

      buffer_barriers[1].dstAccessMask = VK_ACCESS_VERTEX_ATTRIBUTE_READ_BIT;
      buffer_barriers[2].dstAccessMask = VK_ACCESS_INDEX_READ_BIT;

      image_barrier.srcAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;
      image_barrier.dstAccessMask = VK_ACCESS_SHADER_READ_BIT;
      image_barrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;
      image_barrier.newLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL;

      vkCmdPipelineBarrier(staging_command_buffer,
                           VK_PIPELINE_STAGE_TRANSFER_BIT,
                           VK_PIPELINE_STAGE_ALL_GRAPHICS_BIT, 0, 0, nullptr, 2,
                           &buffer_barriers[1], 1, &image_barrier);

      REQUIRE_SUCCESS(vkEndCommandBuffer(staging_command_buffer));

      VkSubmitInfo submit{
          VK_STRUCTURE_TYPE_SUBMIT_INFO, nullptr, 0,      nullptr, nullptr, 1,
          &staging_command_buffer,       0,       nullptr};
      REQUIRE_SUCCESS(vkQueueSubmit(queue, 1, &submit, VK_NULL_HANDLE));
      REQUIRE_SUCCESS(vkQueueWaitIdle(queue));
    }
  }

  // Swapchain-related-stuff
  for (size_t i = 0; i < swapchain_images.size(); ++i) {
    VkImageViewCreateInfo create_info{
        VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
        nullptr,
        0,
        swapchain_images[i],
        VK_IMAGE_VIEW_TYPE_2D,
        surface_format.format,
        VkComponentMapping{
            VK_COMPONENT_SWIZZLE_IDENTITY, VK_COMPONENT_SWIZZLE_IDENTITY,
            VK_COMPONENT_SWIZZLE_IDENTITY, VK_COMPONENT_SWIZZLE_IDENTITY},
        VkImageSubresourceRange{VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1}};
    VkImageView swapchain_view;
    REQUIRE_SUCCESS(
        vkCreateImageView(device, &create_info, nullptr, &swapchain_view));
    swapchain_views.push_back(swapchain_view);
  }

  std::unordered_set<uint32_t> seen_swapchain_images;
  uint64_t frame_count = 0;
  float total_time = 0;
  auto last_frame_time = std::chrono::high_resolution_clock::now();
  // Actually Start Rendering?
  while (true) {
    auto current_time = std::chrono::high_resolution_clock::now();
    std::chrono::duration<float> elapsed_time = current_time - last_frame_time;
    last_frame_time = current_time;
    total_time += elapsed_time.count();
    ProcessNativeWindowEvents();
    uint64_t frame_parity = frame_count % kBufferingCount;

    uint32_t next_image = 0;

    REQUIRE_SUCCESS(
        vkAcquireNextImageKHR(device, swapchain, static_cast<uint64_t>(-1),
                              swapchain_image_ready_semaphores[frame_parity],
                              VK_NULL_HANDLE, &next_image));

    if (frame_count >= kBufferingCount) {
      REQUIRE_SUCCESS(vkWaitForFences(device, 1, &ready_fences[frame_parity],
                                      false, static_cast<uint64_t>(-1)));
      REQUIRE_SUCCESS(vkResetFences(device, 1, &ready_fences[frame_parity]));
    }

    VkCommandBufferBeginInfo begin_info{
        VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, nullptr,
        VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT, nullptr};
    REQUIRE_SUCCESS(vkResetCommandPool(device, command_pools[frame_parity], 0));

    REQUIRE_SUCCESS(vkBeginCommandBuffer(render_command_buffers[frame_parity],
                                         &begin_info));

    if (framebuffers[frame_parity] != VK_NULL_HANDLE) {
      vkDestroyFramebuffer(device, framebuffers[frame_parity], nullptr);
      // Fill the framebuffers
    }

    {
      VkImageView views[2] = {swapchain_views[next_image],
                              depth_buffer_views[frame_parity]};
      VkFramebufferCreateInfo create_info = {
          VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
          nullptr,
          0,
          render_pass,
          2,
          views,
          surface_capabilities.currentExtent.width,
          surface_capabilities.currentExtent.height,
          1};
      REQUIRE_SUCCESS(vkCreateFramebuffer(device, &create_info, nullptr,
                                          &framebuffers[frame_parity]));
    }

    float angle = 3.14f * total_time;
    float ca = cosf(angle);
    float sa = sinf(angle);

    float mat[16] = {1.0f, 0.0f, 0.0f, 0.0f, 0.0f, ca,   sa,    0.0f,
                     0.0f, -sa,  ca,   0.0f, 0.0f, 0.0f, -3.0f, 1.0f};
    float aspect = float(surface_capabilities.currentExtent.width) /
                   float(surface_capabilities.currentExtent.height);

    float camera[16];
    {
      float fovy = 1.5708f;
      float znear = 0.1f;
      float zfar = 100.0f;

      float y = 1 / std::tan(fovy * 0.5f);
      float x = y / aspect;
      float zdist = znear - zfar;
      float zfozd = zfar / zdist;

      camera[0] = x;
      camera[1] = 0;
      camera[2] = 0;
      camera[3] = 0;

      camera[4] = 0;
      camera[5] = y;
      camera[6] = 0;
      camera[7] = 0;

      camera[8] = 0;
      camera[9] = 0;
      camera[10] = zfozd;
      camera[11] = -1;

      camera[12] = 0;
      camera[13] = 0;
      camera[14] = 2.0f * znear * zfozd;
      camera[15] = 0;
    }

    void* uniform_data;
    REQUIRE_SUCCESS(vkMapMemory(device, uniform_buffer_memories[frame_parity],
                                0, VK_WHOLE_SIZE, 0, &uniform_data));
    memcpy(uniform_data, camera, sizeof(float) * 16);
    memcpy((char*)(uniform_data) + sizeof(float) * 16, mat, sizeof(float) * 16);

    {
      VkMappedMemoryRange range{VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, nullptr,
                                uniform_buffer_memories[frame_parity], 0,
                                VK_WHOLE_SIZE};
      REQUIRE_SUCCESS(vkFlushMappedMemoryRanges(device, 1, &range));
    }
    vkUnmapMemory(device, uniform_buffer_memories[frame_parity]);

    {
      VkBufferMemoryBarrier buffer_barriers[] = {
          {// Staging image
           VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, nullptr,
           VK_ACCESS_HOST_WRITE_BIT, VK_ACCESS_UNIFORM_READ_BIT,
           queue_family_index, queue_family_index,
           uniform_buffers[frame_parity], 0, VK_WHOLE_SIZE}};

      vkCmdPipelineBarrier(render_command_buffers[frame_parity],
                           VK_PIPELINE_STAGE_HOST_BIT,
                           VK_PIPELINE_STAGE_VERTEX_SHADER_BIT, 0, 0, nullptr,
                           1, buffer_barriers, 0, nullptr);
    }

    {
      VkClearValue clears[2] = {};
      clears[1].depthStencil.depth = 1.0f;
      VkRenderPassBeginInfo begin{
          VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
          nullptr,
          render_pass,
          framebuffers[frame_parity],
          VkRect2D{{0, 0},
                   {
                       surface_capabilities.currentExtent.width,
                       surface_capabilities.currentExtent.height,
                   }},
          2,
          clears};

      vkCmdBeginRenderPass(render_command_buffers[frame_parity], &begin,
                           VK_SUBPASS_CONTENTS_INLINE);
    }

    vkCmdBindPipeline(render_command_buffers[frame_parity],
                      VK_PIPELINE_BIND_POINT_GRAPHICS, graphics_pipeline);
    vkCmdBindDescriptorSets(render_command_buffers[frame_parity],
                            VK_PIPELINE_BIND_POINT_GRAPHICS, pipeline_layout, 0,
                            1, &descriptor_sets[frame_parity], 0, nullptr);
    VkDeviceSize offset = 0;
    vkCmdBindVertexBuffers(render_command_buffers[frame_parity], 0, 1,
                           &vertex_buffer, &offset);
    vkCmdBindIndexBuffer(render_command_buffers[frame_parity], index_buffer, 0,
                         VK_INDEX_TYPE_UINT32);
    vkCmdDrawIndexed(render_command_buffers[frame_parity],
                     cube::model.num_indices, 1, 0, 0, 0);
    vkCmdEndRenderPass(render_command_buffers[frame_parity]);

    REQUIRE_SUCCESS(vkEndCommandBuffer(render_command_buffers[frame_parity]));
    VkPipelineStageFlags flags = VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT;

    VkSubmitInfo submit{VK_STRUCTURE_TYPE_SUBMIT_INFO,
                        nullptr,
                        1,
                        &swapchain_image_ready_semaphores[frame_parity],
                        &flags,
                        1,
                        &render_command_buffers[frame_parity],
                        1,
                        &render_done_semaphores[frame_parity]};
    REQUIRE_SUCCESS(
        vkQueueSubmit(queue, 1, &submit, ready_fences[frame_parity]));
    VkResult result;
    VkPresentInfoKHR present_info{VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
                                  nullptr,
                                  1,
                                  &render_done_semaphores[frame_parity],
                                  1,
                                  &swapchain,
                                  &next_image,
                                  &result};

    REQUIRE_SUCCESS(vkQueuePresentKHR(queue, &present_info));
    REQUIRE_SUCCESS(result);
    ++frame_count;
  }

  return 0;
}
