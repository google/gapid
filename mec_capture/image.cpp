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

#include "image.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_images(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  for (auto& it : state_block->VkImages) {
    VkImageWrapper* img = it.second.second;
    // Swapchain images are already created for us during swapchain creation.
    if (img->get_swapchain() != VK_NULL_HANDLE) {
      continue;
    }
    VkImage image = it.first;
    serializer->vkCreateImage(img->device,
                              img->get_create_info(), nullptr, &image);
  }
}

void mid_execution_generator::capture_bind_images(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  for (auto& it : state_block->VkImages) {
    VkImageWrapper* img = it.second.second;
    if (img->get_swapchain() != VK_NULL_HANDLE) {
      continue;
    }
    GAPID2_ASSERT(0 == img->get_create_info()->flags & VK_IMAGE_CREATE_SPARSE_BINDING_BIT, "We do not support sparse images yet");
    GAPID2_ASSERT(img->bindings.size() <= 1, "Invalid number of binds");

#pragma TODO(awoloszyn, Handle the different special bind flags)
    if (img->bindings.empty()) {
      continue;
    }
    serializer->vkBindImageMemory(img->device, it.first, img->bindings[0].memory, img->bindings[0].offset);
  }
}

}  // namespace gapid2