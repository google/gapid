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
#include "gapii/cc/vulkan_spy.h"

namespace gapii {

struct destroyer {
    destroyer(const std::function<void(void)>& f) {
        destroy = f;
    }
    ~destroyer() {
        destroy();
    }
    std::function<void(void)> destroy;
};

// Declared in api_spy.h.tmpl
bool VulkanSpy::observeFramebuffer(CallObserver* observer,
        uint32_t* w, uint32_t* h, std::vector<uint8_t>* data) {
    gapil::Ref<ImageObject> image;
    uint32_t frame_buffer_img_level = 0;
    uint32_t frame_buffer_img_layer = 0;
    if (mState.LastSubmission == LastSubmissionType::SUBMIT) {
        if (!mState.LastBoundQueue) {
            return false;
        }
        if (!mState.LastDrawInfos.contains(mState.LastBoundQueue->mVulkanHandle)) {
            return false;
        }
        auto& lastDrawInfo = *mState.LastDrawInfos[mState.LastBoundQueue->mVulkanHandle];
        if (!lastDrawInfo.mRenderPass) {
            return false;
        }
        if (!lastDrawInfo.mFramebuffer) {
            return false;
        }
        if (lastDrawInfo.mLastSubpass >=
            lastDrawInfo.mRenderPass->mSubpassDescriptions.count()) {
            return false;
        }
        if (lastDrawInfo.mRenderPass->mSubpassDescriptions[lastDrawInfo.mLastSubpass].mColorAttachments.empty()) {
            return false;
        }

        uint32_t color_attachment_index = lastDrawInfo.mRenderPass->mSubpassDescriptions[lastDrawInfo.mLastSubpass].mColorAttachments[0].mAttachment;
        if (!lastDrawInfo.mFramebuffer->mImageAttachments.contains(color_attachment_index)) {
            return false;
        }

        auto& imageView = lastDrawInfo.mFramebuffer->mImageAttachments[color_attachment_index];
        image = imageView->mImage;
        *w = lastDrawInfo.mFramebuffer->mWidth;
        *h = lastDrawInfo.mFramebuffer->mHeight;
        // If the image view is to be used as framebuffer attachment, it must
        // contains only one level.
        frame_buffer_img_level = imageView->mSubresourceRange.mbaseMipLevel;
        // There might be more layers, but we only show the first layer.
        // TODO: support multi-layer rendering.
        frame_buffer_img_layer = imageView->mSubresourceRange.mbaseArrayLayer;
    } else {
        if (mState.LastPresentInfo.mPresentImageCount == 0) {
            return false;
        }
        image = mState.LastPresentInfo.mPresentImages[0];
        *w = image->mInfo.mExtent.mWidth;
        *h = image->mInfo.mExtent.mHeight;
        // Swapchain images have only one miplevel.
        frame_buffer_img_level = 0;
        // There might be more than one array layers for swapchain images,
        // currently we only show the data at layer 0
        // TODO: support multi-layer swapchain images.
        frame_buffer_img_layer = 0;
    }

    // TODO: Handle multisampled images. This is only a concern for
    // draw-level observations.

    VkDevice device = image->mDevice;
    VkPhysicalDevice physical_device = mState.Devices[device]->mPhysicalDevice;
    VkInstance instance = mState.PhysicalDevices[physical_device]->mInstance;
    VkQueue queue = image->mLastBoundQueue->mVulkanHandle;
    uint32_t queue_family = image->mLastBoundQueue->mFamily;
    auto& instance_fn = mImports.mVkInstanceFunctions[instance];

    VkPhysicalDeviceMemoryProperties memory_properties(arena());
    instance_fn.vkGetPhysicalDeviceMemoryProperties(physical_device,
        &memory_properties);


    uint32_t format = image->mInfo.mFormat;
    auto& fn = mImports.mVkDeviceFunctions[device];

    VkImageCreateInfo create_info {
        VkStructureType::VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO, // sType
        nullptr,                                              // pNext
        0,                                                    // flags
        VkImageType::VK_IMAGE_TYPE_2D,                        // imageType
        VkFormat::VK_FORMAT_R8G8B8A8_UNORM,                   // format
        VkExtent3D{*w, *h, 1},                                // extent
        1,                                                    // mipLevels
        1,                                                    // arrayLayers
        VkSampleCountFlagBits::VK_SAMPLE_COUNT_1_BIT,         // samples
        VkImageTiling::VK_IMAGE_TILING_OPTIMAL,               // tiling
        VkImageUsageFlagBits::VK_IMAGE_USAGE_TRANSFER_SRC_BIT |
        VkImageUsageFlagBits::VK_IMAGE_USAGE_TRANSFER_DST_BIT, // usage
        VkSharingMode::VK_SHARING_MODE_EXCLUSIVE,             // sharingMode
        0,                                                    // queueFamilyIndexCount
        nullptr,                                              // queueFamilyIndices
        VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED              // layout
    };

    VkImage resolve_image;
    VkDeviceMemory image_memory;

    if (VkResult::VK_SUCCESS != fn.vkCreateImage(device, &create_info, nullptr, &resolve_image)) {
        return false;
    }
    destroyer image_destroyer([&]() {
        fn.vkDestroyImage(device, resolve_image, nullptr);
    });


    VkMemoryRequirements image_reqs(arena());
    fn.vkGetImageMemoryRequirements(device, resolve_image, &image_reqs);

    uint32_t image_memory_req = 0xFFFFFFFF;
    for (size_t i = 0; i < 32; ++i) {
        if (image_reqs.mmemoryTypeBits & (1 << i)) {
            image_memory_req = i;
            break;
        }
    }

    if (image_memory_req == 0xFFFFFFFF) {
        return false;
    }

    VkMemoryAllocateInfo allocate {
        VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
        nullptr,                                                 // pNext
        image_reqs.msize,                                        // allocationSize
        image_memory_req                                         // memoryTypeIndex
    };
    if (VkResult::VK_SUCCESS != fn.vkAllocateMemory(device, &allocate, nullptr, &image_memory)) {
        return false;
    }
    destroyer image_memory_destroyer([&]() {
        fn.vkFreeMemory(device, image_memory, nullptr);
    });

    fn.vkBindImageMemory(device, resolve_image, image_memory, 0);

    VkBuffer buffer;
    VkDeviceMemory buffer_memory;
    VkBufferCreateInfo buffer_info = {
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,   // sType
        nullptr,                                                 // pNext
        0,                                                       // flags
        *w * *h * 4,                                             // size
        VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT, // usage
        VkSharingMode::VK_SHARING_MODE_EXCLUSIVE,                // sharingMode
        0,                                                       // queueFamilyIndexCountg
        nullptr                                                  // queueFamilyIndices
    };

    if (VkResult::VK_SUCCESS != fn.vkCreateBuffer(device, &buffer_info, nullptr, &buffer)) {
        return false;
    }
    destroyer buffer_destroyer([&]() {
        fn.vkDestroyBuffer(device, buffer, nullptr);
    });

    VkMemoryRequirements buffer_reqs(arena());
    fn.vkGetBufferMemoryRequirements(device, buffer, &buffer_reqs);

    uint32_t buffer_memory_req = 0;
    while(buffer_reqs.mmemoryTypeBits) {
        if (buffer_reqs.mmemoryTypeBits & 0x1) {
            if (memory_properties.mmemoryTypes[buffer_memory_req].mpropertyFlags &
                    VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
                break;
            }
        }
        buffer_reqs.mmemoryTypeBits >>= 1;
        ++buffer_memory_req;
    }
    if (!buffer_reqs.mmemoryTypeBits) {
        return false;
    }
    allocate.mallocationSize = buffer_reqs.msize;
    allocate.mmemoryTypeIndex = buffer_memory_req;
    if (VkResult::VK_SUCCESS != fn.vkAllocateMemory(device, &allocate, nullptr, &buffer_memory)) {
        return false;
    }
    destroyer buffer_memory_destroyer([&]() {
        fn.vkFreeMemory(device, buffer_memory, nullptr);
    });

    fn.vkBindBufferMemory(device, buffer, buffer_memory, 0);

    VkCommandPoolCreateInfo command_pool_info = {
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, // sType
        nullptr,                                                     // pNext
        0,                                                           // flags
        queue_family                                                 // queueFamilyIndex
    };

    VkCommandPool command_pool;
    if (VkResult::VK_SUCCESS != fn.vkCreateCommandPool(device, &command_pool_info, nullptr, &command_pool)){
        return false;
    }
    destroyer command_pool_destroyer([&]() {
        fn.vkDestroyCommandPool(device, command_pool, nullptr);
    });

    VkCommandBufferAllocateInfo command_buffer_info = {
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
        nullptr,                                                         // pNext
        command_pool,                                                    // pool
        VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
        1                                                                // commandBufferCount
    };

    VkCommandBuffer command_buffer;
    if (VkResult::VK_SUCCESS != fn.vkAllocateCommandBuffers(device, &command_buffer_info, &command_buffer)) {
        return false;
    }

    VkImageMemoryBarrier barriers[2] = {{
         VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,  // sType
         nullptr,                                                  // pNext
         (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,  // srcAccessMask
         VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT,            // dstAccessMask
         image->mAspects[VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT]
             ->mLayers[frame_buffer_img_layer]
             ->mLevels[frame_buffer_img_level]
             ->mLayout,                                            // srcLayout
         VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,      // dstLayout
         0xFFFFFFFF,                                               // srcQueueFamily
         0xFFFFFFFF,                                               // dstQueueFamily
         image->mVulkanHandle,                                     // image
         {
             // subresourcerange
             VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT,     // aspectMask
             0,                                                    // baseMipLevel
             1,                                                    // mipLevelCount
             0,                                                    // baseArrayLayer
             1,                                                    // layerCount
         }
    }, {
         VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,  // sType
         nullptr,                                                  // pNext
         (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,  // srcAccessMask
         VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,           // dstAccessMask
         VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED,                 // srcLayout
         VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,      // dstLayout
         0xFFFFFFFF,                                               // srcQueueFamily
         0xFFFFFFFF,                                               // dstQueueFamily
         resolve_image,                                            // image
         {
             // subresourcerange
             VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT,     // aspectMask
             0,                                                    // baseMipLevel
             1,                                                    // mipLevelCount
             0,  // baseArrayLayer
             1,  // layerCount
         }
    },
    };

    fn.vkCmdPipelineBarrier(command_buffer,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_GRAPHICS_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT, 0, 0,
        nullptr, 0, nullptr, 2, barriers);
    VkImageBlit blit = {
        {VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1},
        core::StaticArray<VkOffset3D,2>::create({{0, 0, 0}, {static_cast<int32_t>(*w), static_cast<int32_t>(*h), 1}}),
        {VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1},
        core::StaticArray<VkOffset3D,2>::create({{0, 0, 0}, {static_cast<int32_t>(*w), static_cast<int32_t>(*h), 1}})
    };
    fn.vkCmdBlitImage(command_buffer, image->mVulkanHandle,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        resolve_image,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        1,
        &blit, VkFilter::VK_FILTER_NEAREST);

    barriers[0].msrcAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
    barriers[0].mdstAccessMask = (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1;
    barriers[0].moldLayout = VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
    barriers[0].mnewLayout =
        image->mAspects[VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT]
            ->mLayers[frame_buffer_img_layer]
            ->mLevels[frame_buffer_img_level]
            ->mLayout;
    barriers[1].msrcAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT;
    barriers[1].mdstAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
    barriers[1].moldLayout = VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;
    barriers[1].mnewLayout = VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;

    fn.vkCmdPipelineBarrier(command_buffer,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_GRAPHICS_BIT,
        0, 0, nullptr, 0, nullptr, 2, barriers);

    VkBufferImageCopy copy_region = {
        0, // bufferOffset
        0, // bufferRowLength
        0, // bufferImageHeight
        {VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1},
        {0, 0, 0},
        {*w, *h, 1}
    }
    ;
    fn.vkCmdCopyImageToBuffer(command_buffer, resolve_image,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        buffer, 1, &copy_region);

    VkBufferMemoryBarrier buffer_barrier = {
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
        nullptr,                                                  // pNext
        VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,           // srcAccessMask
        VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,                // dstAccessMask
        0xFFFFFFFF,                                               // srcqueueFamily
        0xFFFFFFFF,                                               // dstQueueFamily
        buffer,                                                   // buffer
        0,                                                        // offset
        0xFFFFFFFFFFFFFFFF                                        // size
    };
    fn.vkCmdPipelineBarrier(command_buffer,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_GRAPHICS_BIT,
        0, 0, nullptr, 1, &buffer_barrier, 0, nullptr);

    fn.vkEndCommandBuffer(command_buffer);

    VkSubmitInfo submit_info = {
        VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO,
        nullptr,
        0, nullptr, nullptr, 1, &command_buffer, 0, nullptr
    };
    if (VkResult::VK_SUCCESS != fn.vkQueueSubmit(queue, 1, &submit_info, 0)) {
        return false;
    }
    fn.vkQueueWaitIdle(queue);
    char* image_data;
    if (VkResult::VK_SUCCESS != fn.vkMapMemory(device, buffer_memory, 0,
        0xFFFFFFFFFFFFFFFF, 0, reinterpret_cast<void**>(&image_data))) {
        return false;
    }
    VkMappedMemoryRange range = {
        VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
        nullptr,                                                // pNext
        buffer_memory,                                          // memory
        0,                                                      // offset
        0xFFFFFFFFFFFFFFFF                                      // size
    };
    fn.vkInvalidateMappedMemoryRanges(device, 1, &range);
    data->resize(*w * *h * 4);
    // Flip the image because vulkan renders upside-down.
    for (size_t i = 0 ; i < *h; ++i) {
        memcpy(data->data() + i * (*w * 4),
            image_data + ((*h - i - 1) * (*w * 4)),
                *w * 4);
    }

    return true;
}

// API extern functions
void VulkanSpy::enterSubcontext(CallObserver*) {}
void VulkanSpy::leaveSubcontext(CallObserver*) {}
void VulkanSpy::nextSubcontext(CallObserver*) {}
void VulkanSpy::resetSubcontext(CallObserver*) {}
void VulkanSpy::onPreSubcommand(CallObserver*, gapil::Ref<CommandReference>) {}
void VulkanSpy::onPreProcessCommand(CallObserver*, gapil::Ref<CommandReference>) {}
void VulkanSpy::onPostSubcommand(CallObserver*, gapil::Ref<CommandReference>) {}
void VulkanSpy::onDeferSubcommand(CallObserver*, gapil::Ref<CommandReference>) {}
void VulkanSpy::onCommandAdded(CallObserver*, VkCommandBuffer) {}
void VulkanSpy::postBindSparse(CallObserver*, gapil::Ref<QueuedSparseBinds>) {}

// Utility functions
void VulkanSpy::walkImageSubRng(
    gapil::Ref<ImageObject> img, VkImageSubresourceRange rng,
    std::function<void(uint32_t aspect_bit, uint32_t layer, uint32_t level)>
        f) {
  auto aspect_map =
      subUnpackImageAspectFlags(nullptr, nullptr, rng.maspectMask);
  for (auto b : aspect_map->mBits) {
    auto ai = img->mAspects.find(b.second);
    if (ai == img->mAspects.end()) {
      continue;
    }
    for (uint32_t layer = rng.mbaseArrayLayer;
         layer < rng.mbaseArrayLayer + rng.mlayerCount; layer++) {
      auto layi = ai->second->mLayers.find(layer);
      if (layi == ai->second->mLayers.end()) {
        continue;
      }
      for (uint32_t level = rng.mbaseMipLevel;
           level < rng.mbaseMipLevel + rng.mlevelCount; level++) {
        auto levi = layi->second->mLevels.find(level);
        if (levi == layi->second->mLevels.end()) {
          continue;
        }
        f(b.second, layer, level);
      }
    }
  }
}
}