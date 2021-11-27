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
#include <string>
#include "layer.h"
#define STB_IMAGE_WRITE_IMPLEMENTATION
#include "externals/stb/stb_image_write.h"

using PFN_vkSetSwapchainCallback = void(VKAPI_PTR*)(VkSwapchainKHR swapchain,
                                                    void callback(void*,
                                                                  uint8_t*,
                                                                  size_t),
                                                    void* user_data);

struct foo {
  size_t i = 0;
  uint32_t width = 0;
  uint32_t height = 0;
  VkFormat format = VK_FORMAT_UNDEFINED;
};

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
      pCreateInfo->imageFormat != VK_FORMAT_B8G8R8A8_UNORM) {
    return res;
  }
  foo* f = new foo();
  f->width = pCreateInfo->imageExtent.width;
  f->height = pCreateInfo->imageExtent.height;
  f->format = pCreateInfo->imageFormat;
  OutputDebugString("Setting callback swapchain\n");

  auto set_callback = reinterpret_cast<PFN_vkSetSwapchainCallback>(
      vkGetDeviceProcAddr(device, "vkSetSwapchainCallback"));
  if (set_callback) {
    set_callback(
        get_raw_handle(pSwapchain[0]),
        [](void* userdata, uint8_t* data, size_t size) {
          stbi_write_png_compression_level = 0;
          stbi_write_force_png_filter = 0;
          OutputDebugString("Outputting image\n");
          foo* f = reinterpret_cast<foo*>(userdata);
          if (false && f->format == VK_FORMAT_B8G8R8A8_UNORM) {
            for (size_t i = 0; i < size / 4; ++i) {
              const size_t offs = i * 4;
              const uint8_t dat = data[offs + 2];
              data[offs + 2] = data[offs];
              data[offs] = dat;
            }
          }
          std::string image_name = "Screenshot";
          image_name += std::to_string(f->i++);
          image_name += ".png";
          stbi_write_png(image_name.data(), f->width, f->height, 4, data, 0);
          OutputDebugString("Image has been output\n");
        },
        f);
  }
  return res;
}