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

#pragma once
#include <windows.h>

#include <atomic>
#include <chrono>
#include <string>
#include <thread>
#include <vector>

#include "json.hpp"
#include "layer.h"

std::atomic<HWND> hwnd;
HINSTANCE hInstance;
std::thread thread;
bool quit = false;

LRESULT CALLBACK wnd_proc(
    HWND hwnd, UINT uMsg, WPARAM wParam, LPARAM lParam) {
  switch (uMsg) {
    case WM_USER:
      quit = true;
      PostMessage(hwnd, WM_CLOSE, 0, 0);
      return -1;
    default:
      return DefWindowProc(hwnd, uMsg, wParam, lParam);
  }
}

uint32_t start_idx = 0;
uint32_t indices[3];

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  const char* js = options->GetUserConfig();
  uint32_t width = 1024;
  uint32_t height = 1024;
  if (js) {
    auto setup = nlohmann::json::parse(js, nullptr, false);
    if (setup.contains("start_idx")) {
      start_idx = setup["start_idx"];
    }
    if (setup.contains("width")) {
      width = setup["width"];
    }
    if (setup.contains("height")) {
      height = setup["height"];
    }
  }
  indices[0] = (start_idx + 0) % 3;
  indices[1] = (start_idx + 1) % 3;
  indices[2] = (start_idx + 2) % 3;

  thread = std::thread([width, height]() {
    hInstance = GetModuleHandle(NULL);

    WNDCLASSEX window_class;
    window_class.cbSize = sizeof(WNDCLASSEX);
    window_class.style = CS_HREDRAW | CS_VREDRAW;
    window_class.lpfnWndProc = &wnd_proc;
    window_class.cbClsExtra = 0;
    window_class.cbWndExtra = 0;
    window_class.hInstance = hInstance;
    window_class.hIcon = NULL;
    window_class.hCursor = NULL;
    window_class.hbrBackground = NULL;
    window_class.lpszMenuName = NULL;
    window_class.lpszClassName = "Sample application";
    window_class.hIconSm = NULL;
    RegisterClassExA(&window_class);
    RECT rect = {0, 0, LONG(width), LONG(height)};

    AdjustWindowRect(&rect, WS_OVERLAPPEDWINDOW, FALSE);

    HWND h = CreateWindowExA(
        0, "Sample application", "", WS_OVERLAPPEDWINDOW, CW_USEDEFAULT,
        CW_USEDEFAULT, rect.right - rect.left, rect.bottom - rect.top, 0, 0,
        GetModuleHandle(NULL), NULL);

    hwnd = h;
    ShowWindow(hwnd, SW_NORMAL);
    MSG msg;
    while (GetMessage(&msg, hwnd, 0, 0) && !quit) {
      TranslateMessage(&msg);
      DispatchMessage(&msg);
    }
  });
  LogMessage(debug, std::format("Creating window of size {}x{}", width, height));
  while (hwnd == 0) {
  };
}

VKAPI_ATTR void VKAPI_CALL ShutdownLayer() {
  LogMessage(debug, "Shutting down window");
  SendNotifyMessage(hwnd, WM_USER, 0, 0);
  thread.join();
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateWin32SurfaceKHR(
    VkInstance instance,
    const VkWin32SurfaceCreateInfoKHR* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkSurfaceKHR* pSurface) {
  auto ci = *pCreateInfo;
  ci.hwnd = hwnd;
  ci.hinstance = hInstance;
  return vkCreateWin32SurfaceKHR(instance, &ci, nullptr, pSurface);
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkGetSwapchainImagesKHR(VkDevice device, VkSwapchainKHR swapchain, uint32_t* pSwapchainImageCount, VkImage* pSwapchainImages) {
  VkImage images[3];
  if (pSwapchainImages) {
    images[0] = pSwapchainImages[indices[0]];
    images[1] = pSwapchainImages[indices[1]];
    images[2] = pSwapchainImages[indices[2]];

    pSwapchainImages[0] = images[0];
    pSwapchainImages[1] = images[1];
    pSwapchainImages[2] = images[2];
  }

  auto ret = vkGetSwapchainImagesKHR(device, swapchain, pSwapchainImageCount, pSwapchainImages);
  if (pSwapchainImages == nullptr) {
    return ret;
  }

  return ret;
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR* pPresentInfo) {
  auto present = *pPresentInfo;
  auto index = indices[present.pImageIndices[0]];
  present.pImageIndices = &index;
  std::this_thread::sleep_for(std::chrono::seconds(2));
  return vkQueuePresentKHR(queue, &present);
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkCreateSwapchainKHR(VkDevice device,
                              const VkSwapchainCreateInfoKHR* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkSwapchainKHR* pSwapchain) {
  auto ci = *pCreateInfo;
  ci.pNext = nullptr;
  return vkCreateSwapchainKHR(device, &ci, pAllocator, pSwapchain);
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkAllocateMemory(VkDevice device,
                                                         const VkMemoryAllocateInfo* pAllocateInfo,
                                                         const VkAllocationCallbacks* pAllocator,
                                                         VkDeviceMemory* pMemory) {
  auto ai = *pAllocateInfo;
  ai.pNext = nullptr;
  return vkAllocateMemory(device, &ai, pAllocator, pMemory);
}