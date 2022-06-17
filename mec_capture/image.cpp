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

#include "image_copier.h"
#include "image_helpers.h"
#include "mid_execution_generator.h"
#include "staging_resource_manager.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_images(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecImageCreation");
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

void mid_execution_generator::capture_bind_images(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller, shader_manager* shader_manager) const {
  serializer->insert_annotation("MecImageBinds");
  for (auto& dev : state_block->VkDevices) {
    auto device = dev.second.second;
    staging_resource_manager staging(bypass_caller, serializer, state_block->get(device->get_physical_device()), device, max_copy_overhead_bytes_, shader_manager);
    image_copier copier(&staging, state_block);
    for (auto& it : state_block->VkImages) {
      if (it.second.second->device != dev.first) {
        continue;
      }
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

      img->for_each_subresource_in(VkImageSubresourceRange{
                                       .aspectMask = 0xFFFFFFFF,
                                       .baseMipLevel = 0,
                                       .levelCount = VK_REMAINING_MIP_LEVELS,
                                       .baseArrayLayer = 0,
                                       .layerCount = VK_REMAINING_ARRAY_LAYERS,
                                   },
                                   [&staging, device, img, state_block, serializer, bypass_caller, &copier](uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect) {
                                     const auto& dat = img->sr_data[img->get_subresource_idx(mip_level, array_layer, aspect)];
        // This has never been used on a queue, so we can ignore this for now.
#pragma TODO(awoloszyn, "Some images might end up without a queue but with real data, specifically preinitialized, special-case that")
                                     if (dat.src_queue_idx == VK_QUEUE_FAMILY_IGNORED) {
                                       return;
                                     }
                                     VkCommandBuffer cb = staging.get_command_buffer_for_queue(state_block->get(get_queue_for_family(state_block, device->_handle, dat.src_queue_idx)));

                                     VkImageMemoryBarrier img_memory_barrier = {
                                         .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
                                         .pNext = nullptr,
                                         .srcAccessMask = static_cast<VkAccessFlags>(~VK_ACCESS_NONE_KHR),
                                         .dstAccessMask = VK_ACCESS_TRANSFER_READ_BIT,
                                         .oldLayout = img->get_create_info()->initialLayout,
                                         .newLayout = dat.layout,
                                         .srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
                                         .dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
                                         .image = img->_handle,
                                         .subresourceRange = VkImageSubresourceRange{
                                             .aspectMask = static_cast<VkImageAspectFlags>(aspect),
                                             .baseMipLevel = mip_level,
                                             .levelCount = 1,
                                             .baseArrayLayer = array_layer,
                                             .layerCount = 1}};

                                     serializer->vkCmdPipelineBarrier(
                                         cb,
                                         VK_PIPELINE_STAGE_TRANSFER_BIT,
                                         VK_PIPELINE_STAGE_HOST_BIT,
                                         0, 0, nullptr, 0, nullptr, 1, &img_memory_barrier);
                                     VkExtent3D xe{
                                         .width = get_mip_size(img->get_create_info()->extent.width, mip_level),
                                         .height = get_mip_size(img->get_create_info()->extent.height, mip_level),
                                         .depth = get_mip_size(img->get_create_info()->extent.depth, mip_level),
                                     };
                                     VkOffset3D offs{0, 0, 0};
                                     copier.get_image_content(img, array_layer, mip_level, serializer, bypass_caller, offs, xe, aspect);
                                   });
    }
  }
}

}  // namespace gapid2