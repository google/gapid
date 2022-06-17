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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "forwards.h"
#include "struct_clone.h"
#include "image_helpers.h"
#include "swapchain.h"
#include "state_block.h"
#include <bit>

namespace gapid2 {

void VkImageWrapper::set_create_info(VkDevice device_, state_block* state_block_, const VkImageCreateInfo* pCreateInfo) {
  device = device_;
  create_info = mem.get_typed_memory<VkImageCreateInfo>(1);
  clone(state_block_, pCreateInfo[0], create_info[0], &mem,
        _VkImageCreateInfo_pQueueFamilyIndices_valid);
  uint32_t num_subresources = std::popcount(get_aspects(create_info->format)) *
                              create_info->arrayLayers * create_info->mipLevels;
  for (size_t i = 0; i < num_subresources; ++i) {
    sr_data[i] = subresource_data {
      .src_queue_idx = VK_QUEUE_FAMILY_IGNORED,
      .dst_queue_idx = VK_QUEUE_FAMILY_IGNORED,
      .layout = create_info->initialLayout
    };
  }
}

void VkImageWrapper::set_swapchain_info(VkDevice device_, state_block* state_block_, VkSwapchainKHR swap, uint32_t i) {
  device = device_;
  swapchain = swap;
  swapchain_idx = i;
  auto sci = state_block_->get(swap)->create_info;
  create_info = &swapchain_create_info;
  swapchain_create_info = VkImageCreateInfo {
    .sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
    .pNext = nullptr,
    .flags = 0,
    .imageType = VK_IMAGE_TYPE_2D,
    .format = sci->imageFormat,
    .extent = VkExtent3D{sci->imageExtent.width, sci->imageExtent.height, 1},
    .mipLevels = 1,
    .arrayLayers = sci->imageArrayLayers,
    .samples = VK_SAMPLE_COUNT_1_BIT,
    .tiling = VK_IMAGE_TILING_OPTIMAL,
    .usage = sci->imageUsage,
    .sharingMode = VK_SHARING_MODE_EXCLUSIVE,
#pragma TODO(awoloszyn, "Handle shared queues for presentation images");
    .queueFamilyIndexCount = 0,
    .pQueueFamilyIndices = nullptr,
    .initialLayout = VK_IMAGE_LAYOUT_UNDEFINED
  };
}

uint32_t VkImageWrapper::get_aspect_index(VkImageAspectFlagBits aspect) const {
  VkImageAspectFlags image_aspects = get_aspects(create_info->format);
  if (!(aspect & image_aspects)) {
    return -1;
  }
  if ((aspect & image_aspects) == aspect) {
    return 0;
  }
  uint32_t idx = 0;
  do {
    if (aspect & 0x1) {
      return idx;
    }
    aspect = static_cast<VkImageAspectFlagBits>(aspect >> 1);
    idx += (image_aspects & 0x1);
    image_aspects >>= 1;
  } while (aspect);
  return -1;
}

uint32_t VkImageWrapper::get_subresource_idx(uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect_flag) const {
  uint32_t aspect_index = get_aspect_index(aspect_flag);
  return mip_level + (array_layer * create_info->mipLevels) + (aspect_index * create_info->mipLevels * create_info->arrayLayers);
}

void VkImageWrapper::for_each_subresource_in(VkImageSubresourceRange range, const std::function<void(uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect)>& fn) {
  VkImageAspectFlags all_aspects = get_aspects(create_info->format);
  // Clamp to mip levels
  if (range.baseMipLevel >= create_info->mipLevels || range.baseArrayLayer >= create_info->arrayLayers) {
    return;
  }
  if (range.layerCount - range.baseArrayLayer > create_info->arrayLayers) {
    range.layerCount = create_info->arrayLayers - range.baseArrayLayer;
  }
  if (range.levelCount - range.baseMipLevel > create_info->mipLevels) {
    range.levelCount = create_info->mipLevels - range.baseMipLevel;
  }
  if (is_multi_planar_color(create_info->format) && range.aspectMask & VK_IMAGE_ASPECT_COLOR_BIT) {
    range.aspectMask |= VK_IMAGE_ASPECT_PLANE_0_BIT;
    range.aspectMask |= VK_IMAGE_ASPECT_PLANE_1_BIT;
    range.aspectMask |= VK_IMAGE_ASPECT_PLANE_2_BIT;
  }
  uint32_t aspects = range.aspectMask;
  while (aspects) {
    // Get only the lowest set bit
    uint32_t aspect = aspects & ~(aspects - 1);
    // Then remove this bit for next loop.
    aspects &= ~aspect;

    if (!(aspect & all_aspects)) {
      continue;
    }
    for (uint32_t i = range.baseArrayLayer; i < range.layerCount; ++i) {
      for (uint32_t j = range.baseMipLevel; j < range.levelCount; ++j) {
        fn(j, i, static_cast<VkImageAspectFlagBits>(aspect));
      }
    }
  }
}

}  // namespace gapid2
