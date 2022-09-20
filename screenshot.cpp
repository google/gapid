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
#include <chrono>
#include <string>
#include <thread>

#include "layer.h"
#define STB_IMAGE_WRITE_IMPLEMENTATION
#include <atomic>
#include <vector>

#include "externals/stb/stb_image_write.h"
#include "json.hpp"

using PFN_vkSetSwapchainCallback = void(VKAPI_PTR*)(VkSwapchainKHR swapchain,
                                                    void callback(void*,
                                                                  uint8_t*,
                                                                  size_t),
                                                    void* user_data);
std::atomic<uint32_t> waiting_images;

void send_diagnostic(const std::string& str) {
  auto obj = nlohmann::json::object();
  obj["message"] = str;
  SendJson(obj.dump());
}

struct foo {
  ~foo() {
    if (last_data.empty()) {
      return;
    }
    while (waiting_images)
      ;
  }

  void output_image() {
    stbi_write_png_compression_level = 0;
    stbi_write_force_png_filter = 0;
    send_diagnostic("Dumping Image");
    if (format == VK_FORMAT_B8G8R8A8_UNORM) {
      for (size_t i = 0; i < last_data.size() / 4; ++i) {
        const size_t offs = i * 4;
        const uint8_t dat = last_data[offs + 2];
        last_data[offs + 2] = last_data[offs];
        last_data[offs] = dat;
      }
    }
    std::string image_name = "Screenshot";
    image_name += std::to_string(i);
    image_name += ".png";
    stbi_write_png(image_name.data(), width, height, 4, last_data.data(), 0);
    waiting_images--;
  }

  size_t i = 0;
  uint32_t width = 0;
  uint32_t height = 0;
  VkFormat format = VK_FORMAT_UNDEFINED;
  std::vector<uint8_t> last_data;
};

auto begin = std::chrono::high_resolution_clock::now();
VKAPI_ATTR VkResult VKAPI_CALL
override_vkCreateInstance(const VkInstanceCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkInstance* instance) {
  auto begin = std::chrono::high_resolution_clock::now();
  auto ret = vkCreateInstance(pCreateInfo, pAllocator, instance);
  auto end = std::chrono::high_resolution_clock::now();
  send_diagnostic(("Create Instance time:: " +
                   std::to_string(std::chrono::duration<float>(end - begin).count()))
                      .c_str());
  return ret;
}

VKAPI_ATTR VkResult VKAPI_CALL
override_vkCreateSwapchainKHR(VkDevice device,
                              const VkSwapchainCreateInfoKHR* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkSwapchainKHR* pSwapchain) {
  auto res = vkCreateSwapchainKHR(device, pCreateInfo, pAllocator, pSwapchain);
  if (res != VK_SUCCESS) {
    return res;
  }
  if (pCreateInfo->imageFormat != VK_FORMAT_R8G8B8A8_UNORM &&
      pCreateInfo->imageFormat != VK_FORMAT_B8G8R8A8_UNORM &&
      pCreateInfo->imageFormat != VK_FORMAT_R8G8B8A8_SRGB) {
    return res;
  }
  static foo f;
  f.width = pCreateInfo->imageExtent.width;
  f.height = pCreateInfo->imageExtent.height;
  f.format = pCreateInfo->imageFormat;
  OutputDebugString("Setting callback swapchain");

  auto set_callback = reinterpret_cast<PFN_vkSetSwapchainCallback>(
      vkGetDeviceProcAddr(device, "vkSetSwapchainCallback"));
  if (set_callback) {
    set_callback(
        get_raw_handle(pSwapchain[0]),
        [](void* userdata, uint8_t* data, size_t size) {
          f.i++;
          f.last_data.resize(size);
          memcpy(f.last_data.data(), data, size);
          f.output_image();
        },
        nullptr);
  }
  return res;
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR* pPresentInfo) {
  waiting_images++;
  auto ret = vkQueuePresentKHR(queue, pPresentInfo);
  std::this_thread::sleep_for(std::chrono::milliseconds(1000));
  return ret;
}