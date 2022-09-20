/*Copyright(C) 2022 Google Inc.
     *
         *Licensed under the Apache License,
    Version 2.0(the "License");
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

#include "image_copier.h"

#include <numeric>

#include "command_serializer.h"
#include "image.h"
#include "image_helpers.h"
#include "shader_manager.h"
#include "state_block.h"

namespace gapid2 {

bool image_copier::get_image_content(
    const VkImageWrapper* image,
    uint32_t array_layer,
    uint32_t mip_level,
    command_serializer* next_serializer,
    transform_base* bypass_caller,
    VkOffset3D offset,
    VkExtent3D extent,
    VkImageAspectFlagBits aspect) {
  auto* ci = image->get_create_info();
  if (ci->samples != VK_SAMPLE_COUNT_1_BIT) {
    return false;
  }

  if (extent.width == static_cast<uint32_t>(-1)) {
    extent.width = get_mip_size(image->get_create_info()->extent.width, mip_level) - offset.x;
  }

  if (extent.height == static_cast<uint32_t>(-1)) {
    extent.height = get_mip_size(image->get_create_info()->extent.height, mip_level) - offset.y;
  }

  if (extent.depth == static_cast<uint32_t>(-1)) {
    extent.depth = get_mip_size(image->get_create_info()->extent.depth, mip_level) - offset.z;
  }

  element_and_block_size sz = get_element_and_block_size_for_aspect(image->get_create_info()->format, aspect);

  const uint32_t bytes_per_row = ((sz.element_size * extent.width) / sz.texel_block_size.width) * extent.depth;
  const uint32_t rows_per_depth_layer = std::max(extent.height / sz.texel_block_size.height, 1u);

  uint32_t row = 0; 
  uint32_t remaining_rows = rows_per_depth_layer;

  if (!remaining_rows) {
    return false;
  }

  const auto& sd = image->sr_data.find(image->get_subresource_idx(mip_level, array_layer, aspect))->second;
  VkQueue q = get_queue_for_family(m_state_block, image->device, sd.src_queue_idx);

  auto cb = m_resource_manager->get_command_buffer_for_queue(m_state_block->get(q));

  VkImageMemoryBarrier img_memory_barrier = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
      .pNext = nullptr,
      .srcAccessMask = static_cast<VkAccessFlags>(~VK_ACCESS_NONE_KHR),
      .dstAccessMask = VK_ACCESS_TRANSFER_READ_BIT,
      .oldLayout = sd.layout,
      .newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
      .srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
      .dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
      .image = image->_handle,
      .subresourceRange = VkImageSubresourceRange{
          .aspectMask = static_cast<VkImageAspectFlags>(aspect),
          .baseMipLevel = mip_level,
          .levelCount = 1,
          .baseArrayLayer = array_layer,
          .layerCount = 1}};

  bypass_caller->vkCmdPipelineBarrier(
      cb,
      VK_PIPELINE_STAGE_TRANSFER_BIT,
      VK_PIPELINE_STAGE_HOST_BIT,
      0, 0, nullptr, 0, nullptr, 1, &img_memory_barrier);

  while (remaining_rows > 0) {
    const uint32_t size = bytes_per_row * remaining_rows;
    staging_resource_manager::staging_resources* res = new staging_resource_manager::staging_resources;

    *res = m_resource_manager->get_staging_buffer_for_queue(
        m_state_block->get(q),
        size, [this, mip_level, array_layer, aspect, cImage = image, cRes = res, cOffs = offset, cE = extent, cS = next_serializer, cB = bypass_caller, cA = aspect, cBytes_per_row = bytes_per_row](const char* data, VkDeviceSize size, std::vector<std::function<void()>>* cleanups) {
          VkBuffer copy_source = cRes->buffer;
          VkDeviceSize offset = cRes->buffer_offset;
          VkDeviceSize copy_size = size;
          VkOffset3D target_offset = cOffs;
          uint32_t target_mip_level = mip_level;
          uint32_t target_array_layer = array_layer;

          // We MAY have to make a copy of this data if the data has to come from somewhere OTHER than the existing staging buffer.
          // This can happen in 2 cases.
          // 1) The image is preinitialized, in which case we dump the data directly into the buffer. However, we need a host-mapped location
          //     for this data to reside, which cannot (even in theory) overlap wiht any region used on the replay. So the best way to do this
          //     is to just allocate some memory here to hold it.
          // 2) The image data has to be massaged. For example for rendering or compute copies, we can only safely guarantee that everything
          //     will work with RGBA32 images, so we will inline-expand the soure data into RGBA32 data, and then use a virtual buffer to
          //     hold all of the data.
          const auto& sd = cImage->sr_data.find(cImage->get_subresource_idx(mip_level, array_layer, aspect))->second;
          VkImageLayout source_layout = sd.layout;

          std::vector<char> dat;
          VkExtent3D ext = cE;
          const uint32_t num_rows = cRes->returned_size / cBytes_per_row;
          ext.height = num_rows;
          const auto ci = cImage->get_create_info();
          // Deterimine how to prime this image.
          const bool is_depth = ci->usage & VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT;
          const bool has_transfer_dst = ci->usage & VK_IMAGE_USAGE_TRANSFER_DST_BIT;
          const bool is_storage = ci->usage & VK_IMAGE_USAGE_STORAGE_BIT;
          const bool is_attachment = ci->usage & (VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT);

          const bool prime_by_copy = has_transfer_dst && !is_depth;
          const bool prime_by_rendering = !prime_by_copy && is_attachment;
          const bool prime_by_compute_store = !prime_by_copy && !prime_by_rendering && is_storage;
          const bool prime_by_preinitialization = !prime_by_copy && !prime_by_rendering && !prime_by_compute_store &&
                                                  ci->tiling == VK_IMAGE_TILING_LINEAR && ci->initialLayout == VK_IMAGE_LAYOUT_PREINITIALIZED;
          GAPID2_ASSERT(prime_by_copy + prime_by_rendering + prime_by_compute_store + prime_by_preinitialization == 1,
                        "No way to prime this image");
          if (prime_by_preinitialization) {
            GAPID2_ERROR("Not implemented yet: prime by preinitialization");
            // This one is the simplest(ish) :)
            return;
          }

          VkImage copy_target = cImage->_handle;
          if (prime_by_rendering || prime_by_compute_store) {
            convert_data_to_rgba32(data, size, cImage, ext, cA, &dat);
            data = dat.data();

            // First we create a 32-bit uint staging image to put our data into
            // We may have to expand some of the data in here :(
            VkImageUsageFlags flags = VK_IMAGE_USAGE_TRANSFER_DST_BIT;
            flags |= prime_by_rendering ? VK_IMAGE_USAGE_INPUT_ATTACHMENT_BIT | VK_IMAGE_USAGE_SAMPLED_BIT : 0;
            flags |= prime_by_compute_store ? VK_IMAGE_USAGE_STORAGE_BIT : 0;

            VkImageCreateInfo new_create_info = *ci;
            new_create_info.usage = flags;
            new_create_info.arrayLayers = 1;
            new_create_info.mipLevels = 1;
            new_create_info.samples = VK_SAMPLE_COUNT_1_BIT;
            new_create_info.extent = cE;
            new_create_info.format = VK_FORMAT_R32G32B32A32_UINT;

            // Actually create an image here :D
            VkImage image;
            VkResult res = cB->vkCreateImage(cImage->device, &new_create_info, nullptr, &image);
            cS->vkCreateImage(cImage->device, &new_create_info, nullptr, &image);
            GAPID2_ASSERT(res == VK_SUCCESS, "Could not create prototype image for replay");

            // Get host memory requirements (will need this for replay).
            VkMemoryRequirements reqs;
            cB->vkGetImageMemoryRequirements(cImage->device, image, &reqs);
            cS->vkGetImageMemoryRequirements(cImage->device, image, &reqs);

            auto pd = m_state_block->get(m_state_block->get(cImage->device)->get_physical_device());

            VkPhysicalDeviceMemoryProperties memory_properties;
            cB->vkGetPhysicalDeviceMemoryProperties(pd->_handle, &memory_properties);
            cS->vkGetPhysicalDeviceMemoryProperties(pd->_handle, &memory_properties);

            uint32_t memory_index = 0;
            for (; memory_index < memory_properties.memoryTypeCount; ++memory_index) {
              if (!(reqs.memoryTypeBits & (1 << memory_index))) {
                continue;
              }

              if ((memory_properties.memoryTypes[memory_index].propertyFlags &
                   VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT) !=
                  VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT) {
                continue;
              }
              break;
            }

            // Just allocate 128 bytes on host, but we will actually allocate the "right" amount on replay.
            VkDeviceMemory dm;
            VkMemoryAllocateInfo allocate_info{
                .sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                .pNext = nullptr,
                .allocationSize = 128,
                .memoryTypeIndex = memory_index,
            };

            cB->vkAllocateMemory(cImage->device, &allocate_info, nullptr, &dm);
            allocate_info.allocationSize = reqs.size;
            cS->vkAllocateMemory(cImage->device, &allocate_info, nullptr, &dm);

            cS->vkBindImageMemory(cImage->device, image, dm, 0);

            VkBufferCreateInfo buffer_create_info{
                .sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
                .pNext = nullptr,
                .flags = 0,
                .size = dat.size(),
                .usage = VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
                .sharingMode = VK_SHARING_MODE_EXCLUSIVE,
                .queueFamilyIndexCount = 0,
                .pQueueFamilyIndices = nullptr};
            VkBuffer buff;
            cB->vkCreateBuffer(cImage->device, &buffer_create_info, nullptr, &buff);
            cS->vkCreateBuffer(cImage->device, &buffer_create_info, nullptr, &buff);

            cB->vkGetBufferMemoryRequirements(cImage->device, buff, &reqs);
            cS->vkGetBufferMemoryRequirements(cImage->device, buff, &reqs);

            memory_index = 0;
            for (; memory_index < memory_properties.memoryTypeCount; ++memory_index) {
              if (!(reqs.memoryTypeBits & (1 << memory_index))) {
                continue;
              }

              if ((memory_properties.memoryTypes[memory_index].propertyFlags &
                   VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) !=
                  VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
                continue;
              }
              break;
            }

            VkDeviceMemory bufferMem;
            allocate_info = VkMemoryAllocateInfo{
                .sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
                .pNext = nullptr,
                .allocationSize = 128,
                .memoryTypeIndex = memory_index,
            };
            cB->vkAllocateMemory(cImage->device, &allocate_info, nullptr, &bufferMem);
            allocate_info.allocationSize = reqs.size;
            cS->vkAllocateMemory(cImage->device, &allocate_info, nullptr, &bufferMem);

            cS->vkBindBufferMemory(cImage->device, buff, bufferMem, 0);

            void* c = const_cast<char*>(data);
            cS->vkMapMemory(cImage->device, bufferMem, 0, reqs.size, 0, reinterpret_cast<void**>(&c));

            {
              auto enc = cS->get_encoder(0);
              enc->encode<uint64_t>(0);
              enc->encode<uint64_t>(cS->get_flags());
              enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(bufferMem));
              enc->encode<uint64_t>(0);  // offset
              enc->encode<uint64_t>(dat.size());
              enc->encode_primitive_array<char>(reinterpret_cast<const char*>(data), dat.size());
            }

            cleanups->push_back([cS, cB, device = cImage->device, dm, image, buff, bufferMem] {
              cB->vkDestroyImage(device, image, nullptr);
              cS->vkDestroyImage(device, image, nullptr);
              cB->vkDestroyBuffer(device, buff, nullptr);
              cS->vkDestroyBuffer(device, buff, nullptr);
              cB->vkFreeMemory(device, dm, nullptr);
              cS->vkFreeMemory(device, dm, nullptr);
              cB->vkFreeMemory(device, bufferMem, nullptr);
              cS->vkFreeMemory(device, bufferMem, nullptr);
            });
            source_layout = VK_IMAGE_LAYOUT_UNDEFINED;

            copy_target = image;

            copy_source = buff;
            offset = 0;
            copy_size = dat.size();

            target_mip_level = 0;
            target_array_layer = 0;
            target_offset = VkOffset3D{0, 0, 0};
          } else {
            auto enc = cS->get_encoder(0);
            enc->encode<uint64_t>(0);
            enc->encode<uint64_t>(cS->get_flags());
            enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(cRes->memory));
            enc->encode<uint64_t>(cRes->buffer_offset);  // offset
            enc->encode<uint64_t>(size);
            enc->encode_primitive_array<char>(reinterpret_cast<const char*>(data), size);
          }

          VkImageMemoryBarrier imb{
              .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
              .pNext = nullptr,
              .srcAccessMask = VkAccessFlags(~0),
              .dstAccessMask = VkAccessFlags(~0),
              .oldLayout = source_layout,
              .newLayout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
              .srcQueueFamilyIndex = 0,
              .dstQueueFamilyIndex = 0,
              .image = copy_target,
              .subresourceRange = VkImageSubresourceRange{
                  .aspectMask = static_cast<VkImageAspectFlags>(aspect),
                  .baseMipLevel = target_mip_level,
                  .levelCount = 1,
                  .baseArrayLayer = target_array_layer,
                  .layerCount = 1}};

          cS->vkCmdPipelineBarrier(cRes->cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                                   0, 0, nullptr, 0, nullptr, 1, &imb);

          VkBufferImageCopy copy{
              .bufferOffset = offset,
              .bufferRowLength = 0,
              .bufferImageHeight = 0,
              .imageSubresource = VkImageSubresourceLayers{
                  .aspectMask = static_cast<VkImageAspectFlags>(aspect),
                  .mipLevel = target_mip_level,
                  .baseArrayLayer = array_layer,
                  .layerCount = 1},
              .imageOffset = target_offset,
              .imageExtent = ext,
          };

          cS->vkCmdCopyBufferToImage(cRes->cb, copy_source, copy_target, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &copy);

          VkImageLayout old_layout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;

          if (prime_by_rendering) {
            VkImageLayout layout = VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL;
            if (ci->usage & VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT) {
              layout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;
            }

            VkImageMemoryBarrier imb[] = {
                VkImageMemoryBarrier{.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
                                     .pNext = nullptr,
                                     .srcAccessMask = VkAccessFlags(~0),
                                     .dstAccessMask = VkAccessFlags(~0),
                                     .oldLayout = sd.layout,
                                     .newLayout = layout,
                                     .srcQueueFamilyIndex = 0,
                                     .dstQueueFamilyIndex = 0,
                                     .image = cImage->_handle,
                                     .subresourceRange = VkImageSubresourceRange{
                                         .aspectMask = static_cast<VkImageAspectFlags>(aspect),
                                         .baseMipLevel = mip_level,
                                         .levelCount = 1,
                                         .baseArrayLayer = array_layer,
                                         .layerCount = 1}},
                VkImageMemoryBarrier{.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, .pNext = nullptr, .srcAccessMask = VkAccessFlags(~0), .dstAccessMask = VkAccessFlags(~0), .oldLayout = sd.layout, .newLayout = layout, .srcQueueFamilyIndex = 0, .dstQueueFamilyIndex = 0, .image = cImage->_handle, .subresourceRange = VkImageSubresourceRange{.aspectMask = static_cast<VkImageAspectFlags>(aspect), .baseMipLevel = 0, .levelCount = 1, .baseArrayLayer = 0, .layerCount = 1}}};

            cS->vkCmdPipelineBarrier(cRes->cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                                     0, 0, nullptr, 0, nullptr, 2, imb);

            auto res = m_resource_manager->get_pipeline_for_rendering(cImage->device, VK_FORMAT_R32G32B32A32_UINT, cImage->get_create_info()->format, aspect);

            VkImageView image_views[2];

            VkImageViewCreateInfo create_info = {
                .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
                .pNext = nullptr,
                .flags = 0,
                .image = copy_target,
                .viewType = VK_IMAGE_VIEW_TYPE_2D,
                .format = VK_FORMAT_R32G32B32A32_UINT,
                .components = VkComponentMapping{
                    .r = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .g = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .b = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .a = VK_COMPONENT_SWIZZLE_IDENTITY},
                .subresourceRange = VkImageSubresourceRange{.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT, .baseMipLevel = 0, .levelCount = 1, .baseArrayLayer = 0, .layerCount = 1}};

            GAPID2_ASSERT(VK_SUCCESS == cB->vkCreateImageView(cImage->device, &create_info, nullptr, &image_views[0]), "Could not create image view");
            cS->vkCreateImageView(cImage->device, &create_info, nullptr, &image_views[0]);

            VkImageViewCreateInfo image_view_create_info = {
                .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
                .pNext = nullptr,
                .flags = 0,
                .image = cImage->_handle,
                .viewType = VK_IMAGE_VIEW_TYPE_2D,
                .format = cImage->get_create_info()->format,
                .components = VkComponentMapping{
                    .r = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .g = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .b = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .a = VK_COMPONENT_SWIZZLE_IDENTITY},
                .subresourceRange = VkImageSubresourceRange{.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT, .baseMipLevel = mip_level, .levelCount = 1, .baseArrayLayer = array_layer, .layerCount = 1}};

            GAPID2_ASSERT(VK_SUCCESS == cB->vkCreateImageView(cImage->device, &image_view_create_info, nullptr, &image_views[1]), "Could not create image view");
            cS->vkCreateImageView(cImage->device, &image_view_create_info, nullptr, &image_views[1]);

            VkDescriptorImageInfo inf = {
                .sampler = VK_NULL_HANDLE,
                .imageView = image_views[0],
                .imageLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL};

            VkWriteDescriptorSet write = {
                .sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
                .pNext = nullptr,
                .dstSet = res.render_ds,
                .dstBinding = 0,
                .dstArrayElement = 0,
                .descriptorCount = 1,
                .descriptorType = VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
                .pImageInfo = &inf,
                .pBufferInfo = nullptr,
                .pTexelBufferView = nullptr};
            cS->vkUpdateDescriptorSets(cImage->device, 1, &write, 0, nullptr);

            VkFramebufferCreateInfo framebuffer_create_info = {
                .sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
                .pNext = nullptr,
                .flags = 0,
                .renderPass = res.render_pass,
                .attachmentCount = 2,
                .pAttachments = image_views,
                .width = get_mip_size(ci->extent.width, mip_level),
                .height = get_mip_size(ci->extent.height, mip_level),
                .layers = 1};

            VkFramebuffer framebuffer = 0;  // make framebuffer;
            GAPID2_ASSERT(VK_SUCCESS == cB->vkCreateFramebuffer(cImage->device, &framebuffer_create_info, nullptr, &framebuffer),
                          "Could not create framebuffer");
            cS->vkCreateFramebuffer(cImage->device, &framebuffer_create_info, nullptr, &framebuffer);

            VkRenderPassBeginInfo render_pass_begin_info = {
                .sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
                .pNext = nullptr,
                .renderPass = res.render_pass,
                .framebuffer = framebuffer,
                .renderArea = VkRect2D{
                    .offset = VkOffset2D{cOffs.x, cOffs.y},
                    .extent = VkExtent2D{ext.width, ext.height}},
                .clearValueCount = 0,
                .pClearValues = nullptr};
            VkRect2D rect = {
                .offset = VkOffset2D{cOffs.x, cOffs.y},
                .extent = VkExtent2D{ext.width, ext.height}};
            cS->vkCmdBeginRenderPass(cRes->cb, &render_pass_begin_info, VK_SUBPASS_CONTENTS_INLINE);
            if (aspect == VK_IMAGE_ASPECT_STENCIL_BIT) {
              VkClearAttachment clear{
                  .aspectMask = VK_IMAGE_ASPECT_STENCIL_BIT,
                  .colorAttachment = 0,
                  .clearValue = VkClearValue{
                      .depthStencil = VkClearDepthStencilValue{.depth = 0, .stencil = 0}}};

              VkClearRect clear_rect{
                  .rect = rect,
                  .baseArrayLayer = array_layer,
                  .layerCount = 1};

              cS->vkCmdClearAttachments(cRes->cb, 1, &clear, 1, &clear_rect);
            }
            cS->vkCmdBindPipeline(cRes->cb, VK_PIPELINE_BIND_POINT_GRAPHICS, res.pipeline);
            VkViewport viewport = {
                .x = 0,
                .y = 0,
                .width = static_cast<float>(framebuffer_create_info.width),
                .height = static_cast<float>(framebuffer_create_info.height),
                .minDepth = 0,
                .maxDepth = 1};
            cS->vkCmdSetViewport(cRes->cb, 0, 1, &viewport);

            cS->vkCmdSetScissor(cRes->cb, 0, 1, &rect);
            cS->vkCmdBindDescriptorSets(cRes->cb, VK_PIPELINE_BIND_POINT_GRAPHICS, res.pipeline_layout, 0, 1, &res.render_ds, 0, nullptr);
            if (aspect == VK_IMAGE_ASPECT_STENCIL_BIT) {
              for (uint32_t i = 0; i < 8; ++i) {
                cS->vkCmdSetStencilWriteMask(cRes->cb, VK_STENCIL_FRONT_AND_BACK, 1 << i);
                cS->vkCmdSetStencilReference(cRes->cb, VK_STENCIL_FRONT_AND_BACK, 1 << i);
                cS->vkCmdPushConstants(cRes->cb, res.pipeline_layout, VK_SHADER_STAGE_FRAGMENT_BIT, 0, 4, &i);
                cS->vkCmdDraw(cRes->cb, 6, 1, 0, 0);
              }
            } else {
              cS->vkCmdDraw(cRes->cb, 6, 1, 0, 0);
            }
            cS->vkCmdEndRenderPass(cRes->cb);

            cleanups->push_back([this, cS, cB, device = cImage->device, framebuffer, image_views, res] {
              cB->vkDestroyImageView(device, image_views[0], nullptr);
              cS->vkDestroyImageView(device, image_views[0], nullptr);
              cB->vkDestroyImageView(device, image_views[1], nullptr);
              cS->vkDestroyImageView(device, image_views[1], nullptr);
              cB->vkDestroyFramebuffer(device, framebuffer, nullptr);
              cS->vkDestroyFramebuffer(device, framebuffer, nullptr);
              m_resource_manager->cleanup_after_pipeline(res);
            });
          } else if (prime_by_compute_store) {
            VkImageView image_views[2];

            VkImageViewCreateInfo create_info = {
                .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
                .pNext = nullptr,
                .flags = 0,
                .image = copy_target,
                .viewType = VK_IMAGE_VIEW_TYPE_2D,
                .format = VK_FORMAT_R32G32B32A32_UINT,
                .components = VkComponentMapping{
                    .r = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .g = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .b = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .a = VK_COMPONENT_SWIZZLE_IDENTITY},
                .subresourceRange = VkImageSubresourceRange{.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT, .baseMipLevel = 0, .levelCount = 1, .baseArrayLayer = 0, .layerCount = 1}};

            GAPID2_ASSERT(VK_SUCCESS == cB->vkCreateImageView(cImage->device, &create_info, nullptr, &image_views[0]), "Could not create image view");
            cS->vkCreateImageView(cImage->device, &create_info, nullptr, &image_views[0]);

            VkImageViewCreateInfo image_view_create_info = {
                .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
                .pNext = nullptr,
                .flags = 0,
                .image = cImage->_handle,
                .viewType = VK_IMAGE_VIEW_TYPE_2D,
                .format = cImage->get_create_info()->format,
                .components = VkComponentMapping{
                    .r = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .g = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .b = VK_COMPONENT_SWIZZLE_IDENTITY,
                    .a = VK_COMPONENT_SWIZZLE_IDENTITY},
                .subresourceRange = VkImageSubresourceRange{.aspectMask = VK_IMAGE_ASPECT_COLOR_BIT, .baseMipLevel = mip_level, .levelCount = 1, .baseArrayLayer = array_layer, .layerCount = 1}};

            GAPID2_ASSERT(VK_SUCCESS == cB->vkCreateImageView(cImage->device, &image_view_create_info, nullptr, &image_views[1]), "Could not create image view");
            cS->vkCreateImageView(cImage->device, &image_view_create_info, nullptr, &image_views[1]);

            auto res = m_resource_manager->get_pipeline_for_copy(cImage->device, VK_FORMAT_R32G32B32A32_UINT, cImage->get_create_info()->format, VK_IMAGE_ASPECT_COLOR_BIT, aspect, cImage->get_create_info()->imageType);

            VkDescriptorImageInfo inf[2] = {{
                                                .sampler = VK_NULL_HANDLE,
                                                .imageView = image_views[0],
                                                .imageLayout = VK_IMAGE_LAYOUT_GENERAL,

                                            },
                                            {
                                                .sampler = VK_NULL_HANDLE,
                                                .imageView = image_views[0],
                                                .imageLayout = VK_IMAGE_LAYOUT_GENERAL,
                                            }};

            VkWriteDescriptorSet write = {
                .sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET,
                .pNext = nullptr,
                .dstSet = res.copy_ds,
                .dstBinding = 0,
                .dstArrayElement = 0,
                .descriptorCount = 2,
                .descriptorType = VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
                .pImageInfo = inf,
                .pBufferInfo = nullptr,
                .pTexelBufferView = nullptr};
            cS->vkUpdateDescriptorSets(cImage->device, 1, &write, 0, nullptr);

            cS->vkCmdBindDescriptorSets(cRes->cb, VK_PIPELINE_BIND_POINT_COMPUTE, res.pipeline_layout,
                                        0, 1, &res.copy_ds, 0, nullptr);
            int32_t offs[4] = {
                cOffs.x,
                cOffs.y,
                cOffs.z,
                0,  // wordidx; // fix if > 32bbp
            };

            cS->vkCmdPushConstants(cRes->cb, res.pipeline_layout, VK_SHADER_STAGE_COMPUTE_BIT,
                                   0, sizeof(offs), offs);
            cS->vkCmdDispatch(cRes->cb, ext.width, ext.height, ext.depth);

            cleanups->push_back([this, cS, cB, device = cImage->device, image_views, res] {
              cB->vkDestroyImageView(device, image_views[0], nullptr);
              cS->vkDestroyImageView(device, image_views[0], nullptr);
              cB->vkDestroyImageView(device, image_views[1], nullptr);
              cS->vkDestroyImageView(device, image_views[1], nullptr);
              m_resource_manager->cleanup_after_pipeline(res);
            });
          }

          imb.oldLayout = old_layout;
          imb.newLayout = sd.layout;
          imb.image = cImage->_handle;
          imb.subresourceRange.baseMipLevel = mip_level;
          imb.subresourceRange.baseArrayLayer = array_layer;

          cS->vkCmdPipelineBarrier(cRes->cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                                   0, 0, nullptr, 0, nullptr, 1, &imb);
        });

    const uint32_t num_rows = res->returned_size / bytes_per_row;
    remaining_rows -= num_rows;
    // If we can copy this whole layer in a single buffer, do that.
    // Otherwise we have to copy row-by-row layer-by-layer

    VkBufferImageCopy copy{
        .bufferOffset = res->buffer_offset,
        .bufferRowLength = 0,
        .bufferImageHeight = 0,
        .imageSubresource = VkImageSubresourceLayers{
            .aspectMask = static_cast<VkImageAspectFlags>(aspect),
            .mipLevel = mip_level,
            .baseArrayLayer = array_layer,
            .layerCount = 1},
        .imageOffset = offset,
        .imageExtent = VkExtent3D(extent.width, num_rows, extent.depth)};

    bypass_caller->vkCmdCopyImageToBuffer(res->cb, image->_handle, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, res->buffer, 1, &copy);

    extent.height -= num_rows;
    offset.y += num_rows;
  }

  cb = m_resource_manager->get_command_buffer_for_queue(m_state_block->get(q));

  img_memory_barrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
  img_memory_barrier.newLayout = sd.layout;
  img_memory_barrier.srcAccessMask = img_memory_barrier.dstAccessMask;
  img_memory_barrier.dstAccessMask = ~VK_ACCESS_NONE_KHR;

  bypass_caller->vkCmdPipelineBarrier(
      cb,
      VK_PIPELINE_STAGE_TRANSFER_BIT,
      VK_PIPELINE_STAGE_HOST_BIT,
      0, 0, nullptr, 0, nullptr, 1, &img_memory_barrier);

  m_resource_manager->flush();
  return true;
}  // namespace gapid2

uint32_t get_bits(const char* data, uint8_t base, uint8_t count) {
  GAPID2_ASSERT(count <= 32, "We dont yet handle this for 64bbp-bit texture formats");
  GAPID2_ASSERT(base + count <= 32 * 4, "We dont yet handle this for 64bbp-bit texture formats");

  uint32_t dat = 0;
  uint32_t idx = 0;
  for (size_t i = 0; i < 16; ++i) {
    for (size_t j = 0; j < 8; ++j) {
      if (idx >= base) {
        dat <<= 1;
        dat |= (data[i] >> j) & 0x1;
      }
      ++idx;
      if (idx >= base + count) {
        break;
      }
    }
  }
  return dat;
}

uint32_t sign_extend(uint32_t num, uint8_t top_bit) {
  bool tb = (num >> top_bit) & 0x1;
  for (uint8_t tb = top_bit + 1; tb < 32; ++tb) {
    num |= tb << tb;
  }
  return num;
}

void image_copier::convert_data_to_rgba32(const char* data, VkDeviceSize data_size,
                                          const VkImageWrapper* src_image,
                                          VkExtent3D extent,
                                          VkImageAspectFlagBits aspect,
                                          std::vector<char>* out_data) {
  const auto* ci = src_image->get_create_info();
  if (ci->format == VK_FORMAT_R32G32B32A32_UINT ||
      ci->format == VK_FORMAT_R32G32B32A32_SFLOAT ||
      ci->format == VK_FORMAT_R32G32B32A32_UINT) {
    // We can leave RGBA32_* formats alone, they will just turn into bitcasts anyway.
    return;
  }

  auto bl = get_buffer_layout_for_aspect(src_image->get_create_info()->format,
                                         aspect);
  auto nElements = data_size / (bl->stride_bits / 8);
  GAPID2_ASSERT(extent.width * extent.height * extent.depth == nElements, "Weird image size");
  out_data->resize(sizeof(uint32_t) * 4 * extent.width * extent.height * extent.depth);
  uint32_t* d = reinterpret_cast<uint32_t*>(out_data->data());

  uint8_t rgba_elems[] = {0xFF, 0xFF, 0xFF, 0xFF};
  uint8_t offsets[] = {0xFF, 0xFF, 0xFF, 0xFF};
  for (size_t i = 0; i < bl->n_channels; ++i) {
    if (bl->channels[i].name == e_channel_name::r) {
      rgba_elems[0] = i;
    } else if (bl->channels[i].name == e_channel_name::g) {
      rgba_elems[1] = i;
    } else if (bl->channels[i].name == e_channel_name::b) {
      rgba_elems[2] = i;
    } else if (bl->channels[i].name == e_channel_name::a) {
      rgba_elems[3] = i;
    } else if (bl->channels[i].name == e_channel_name::d) {
      if (aspect == VK_IMAGE_ASPECT_DEPTH_BIT) {
        rgba_elems[0] = i;
      } else {
        continue;
      }
    } else if (bl->channels[i].name == e_channel_name::s) {
      if (aspect == VK_IMAGE_ASPECT_STENCIL_BIT) {
        rgba_elems[0] = i;
      } else {
        continue;
      }
    } else {
      GAPID2_ERROR("Unhandled channel type");
    }
  }

  for (size_t i = 0; i < 4; ++i) {
    if (rgba_elems[i] == 0xFF) {
      continue;
    }
    offsets[i] = std::accumulate(&bl->channels[0], &bl->channels[0] + rgba_elems[i], 0, [](uint8_t a, const channel_info& ci) { return a + ci.nbits; });
  }

  for (size_t i = 0; i < nElements; ++i) {
    for (size_t j = 0; j < 4; ++j) {
      if (rgba_elems[j] == 0xFF) {
        continue;
      }
      uint32_t bits = get_bits(data + i, offsets[j], bl->channels[rgba_elems[j]].nbits);
      auto ch = &bl->channels[rgba_elems[j]];
      switch (ch->type) {
        case data_type::sint:
        case data_type::snorm:
        case data_type::sscaled:
          if (ch->nbits < 32) {
            bits = sign_extend(bits, ch->nbits);
          }
          break;
        case data_type::sfloat:
          if (ch->nbits != 32) {
            GAPID2_ERROR("TODO: Handle float16 and float64 types here");
          }
          break;
      }
      d[i * 4 + j] = bits;
    }
  }
}

}  // namespace gapid2
