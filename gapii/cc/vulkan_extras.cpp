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
#include "gapii/cc/vulkan_layer_extras.h"

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

// Extern functions
void VulkanSpy::trackMappedCoherentMemory(CallObserver*, uint64_t start, size_val size) {
  // If the tracing not started yet, do not track the coherent memory
  if (is_suspended()) {
      return;
  }
#if COHERENT_TRACKING_ENABLED
    if (m_coherent_memory_tracking_enabled) {
        void* start_addr = reinterpret_cast<void*>(start);
        mMemoryTracker.AddTrackingRange(start_addr, size);
    }
#endif // COHERENT_TRACKING_ENABLED
}

void VulkanSpy::readMappedCoherentMemory(CallObserver *observer, VkDeviceMemory memory, uint64_t offset_in_mapped, size_val readSize) {
    auto &memory_object = mState.DeviceMemories[memory];
    const auto mapped_location = (uint64_t)(memory_object->mMappedLocation);
    void *offset_addr = (void *)(offset_in_mapped + mapped_location);
#if COHERENT_TRACKING_ENABLED
    if (m_coherent_memory_tracking_enabled) {
        const size_val page_size = mMemoryTracker.page_size();
        // Get the valid mapped range
        const auto dirty_pages = mMemoryTracker.GetAndResetDirtyPagesInRange(offset_addr, readSize);
        for (const void *p : dirty_pages) {
            uint64_t page_start = (uint64_t)(p);
            observer->read(slice((uint8_t *)page_start, 0ULL, page_size));
        }
        return;
    }
#endif // COHERENT_TRACKING_ENABLED
    observer->read(slice((uint8_t *)offset_addr, 0ULL, readSize));
}

void VulkanSpy::untrackMappedCoherentMemory(CallObserver*, uint64_t start, size_val size) {
#if COHERENT_TRACKING_ENABLED
    if (m_coherent_memory_tracking_enabled) {
        void* start_addr = reinterpret_cast<void*>(start);
        mMemoryTracker.RemoveTrackingRange(start_addr, size);
    }
#endif // COHERENT_TRACKING_ENABLED
}

void VulkanSpy::mapMemory(CallObserver*, void**, gapil::Slice<uint8_t>) {}
void VulkanSpy::unmapMemory(CallObserver*, gapil::Slice<uint8_t>) {}

bool VulkanSpy::hasDynamicProperty(
        CallObserver* observer,
        const VkPipelineDynamicStateCreateInfo* info,
        uint32_t state) {

    if (!info) { return false; }
    for (size_t i = 0; i < info->mdynamicStateCount; ++i) {
        if (info->mpDynamicStates[i] == state) {
            return true;
        }
    }
    return false;
}

void VulkanSpy::resetCmd(CallObserver* observer, VkCommandBuffer cmdBuf) {}
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
void VulkanSpy::pushDebugMarker(CallObserver*, std::string) {}
void VulkanSpy::popDebugMarker(CallObserver*) {}
void VulkanSpy::pushRenderPassMarker(CallObserver*, VkRenderPass) {}
void VulkanSpy::popRenderPassMarker(CallObserver*) {}
void VulkanSpy::popAndPushMarkerForNextSubpass(CallObserver*, uint32_t) {}

// Override API functions
// SpyOverride_vkGetInstanceProcAddr(), SpyOverride_vkGetDeviceProcAddr(),
// SpyOverride_vkCreateInstance() and SpyOverride_vkCreateDevice() require
// the their function table to be created through the template system, so they
// won't be defined here, but vk_spy_helpers.cpp.tmpl
uint32_t VulkanSpy::SpyOverride_vkEnumerateInstanceLayerProperties(uint32_t *pCount, VkLayerProperties *pProperties) {
    if (pProperties == NULL) {
        *pCount = 1;
        return VkResult::VK_SUCCESS;
    }
    if (pCount == 0) {
        return VkResult::VK_INCOMPLETE;
    }
    *pCount = 1;
    memset(pProperties, 0x00, sizeof(*pProperties));
    strcpy((char*)pProperties->mlayerName, "VkGraphicsSpy");
    pProperties->mspecVersion = VK_VERSION_MAJOR(1) | VK_VERSION_MINOR(0) | 5;
    pProperties->mimplementationVersion = 1;
    strcpy((char*)pProperties->mdescription, "vulkan_trace");
    return VkResult::VK_SUCCESS;
}

uint32_t VulkanSpy::SpyOverride_vkEnumerateDeviceLayerProperties(VkPhysicalDevice dev, uint32_t *pCount, VkLayerProperties *pProperties) {
    if (pProperties == NULL) {
       *pCount = 1;
       return VkResult::VK_SUCCESS;
    }
    if (pCount == 0) {
       return VkResult::VK_INCOMPLETE;
    }
    *pCount = 1;
    memset(pProperties, 0x00, sizeof(*pProperties));
    strcpy((char*)pProperties->mlayerName, "VkGraphicsSpy");
    pProperties->mspecVersion = VK_VERSION_MAJOR(1) | VK_VERSION_MINOR(0) | 5;
    pProperties->mimplementationVersion = 1;
    strcpy((char*)pProperties->mdescription, "vulkan_trace");
    return VkResult::VK_SUCCESS;
}
uint32_t VulkanSpy::SpyOverride_vkEnumerateInstanceExtensionProperties(const char *pLayerName, uint32_t *pCount, VkExtensionProperties *pProperties) {
    *pCount = 0;
    return VkResult::VK_SUCCESS;
}

uint32_t VulkanSpy::SpyOverride_vkEnumerateDeviceExtensionProperties(VkPhysicalDevice physicalDevice, const char *pLayerName, uint32_t *pCount, VkExtensionProperties *pProperties) {
    gapii::VulkanImports::PFNVKENUMERATEDEVICEEXTENSIONPROPERTIES next_layer_enumerate_extensions = NULL;
    auto phy_dev_iter = mState.PhysicalDevices.find(physicalDevice);
    if (phy_dev_iter != mState.PhysicalDevices.end()) {
        auto inst_func_iter = mImports.mVkInstanceFunctions.find(phy_dev_iter->second->mInstance);
        if (inst_func_iter != mImports.mVkInstanceFunctions.end()) {
            next_layer_enumerate_extensions = reinterpret_cast<gapii::VulkanImports::PFNVKENUMERATEDEVICEEXTENSIONPROPERTIES>(
                inst_func_iter->second.vkEnumerateDeviceExtensionProperties);
        }
    }

    uint32_t next_layer_count = 0;
    uint32_t next_layer_result;
    if (next_layer_enumerate_extensions) {
        next_layer_result = next_layer_enumerate_extensions(physicalDevice, pLayerName, &next_layer_count, NULL);
        if (next_layer_result != VkResult::VK_SUCCESS) {
            return next_layer_result;
        }
    }
    std::vector<VkExtensionProperties> properties(next_layer_count, VkExtensionProperties{arena()});
    if (next_layer_enumerate_extensions) {
        next_layer_result = next_layer_enumerate_extensions(physicalDevice, pLayerName, &next_layer_count, properties.data());
        if (next_layer_result != VkResult::VK_SUCCESS) {
            return next_layer_result;
        }
    }
    bool has_debug_marker_ext = false;
    for (VkExtensionProperties& ext : properties) {
        // TODO: Check the spec version and emit warning if not match.
        // TODO: refer to VK_EXT_DEBUG_MARKER_EXTENSION_NAME
        if (!strcmp(ext.mextensionName, "VK_EXT_debug_marker")) {
            has_debug_marker_ext = true;
            break;
        }
    }
    if (!has_debug_marker_ext) {
        // TODO: refer to VK_EXT_DEBUG_MARKER_EXTENSION_NAME and VK_EXT_DEBUG_MARKER_SPEC_VERSION
        char debug_marker_extension_name[] = "VK_EXT_debug_marker";
        uint32_t debug_marker_spec_version = 4;
        properties.emplace_back(VkExtensionProperties{debug_marker_extension_name, debug_marker_spec_version});
    }
    if (pProperties == NULL) {
        *pCount = properties.size();
        return VkResult::VK_SUCCESS;
    }
    uint32_t copy_count = properties.size() < *pCount ? properties.size():*pCount;
    memcpy(pProperties, properties.data(), copy_count * sizeof(VkExtensionProperties));
    if (*pCount < properties.size()) {
        return VkResult::VK_INCOMPLETE;
    }
    *pCount = properties.size();
    return VkResult::VK_SUCCESS;
}

void VulkanSpy::SpyOverride_vkDestroyInstance(
        VkInstance                   instance,
        const VkAllocationCallbacks* pAllocator) {

    // First we have to find the function to chain to, then we have to
    // remove this instance from our list, then we forward the call.
    auto it = mImports.mVkInstanceFunctions.find(instance);
    gapii::VulkanImports::PFNVKDESTROYINSTANCE destroy_instance =
        it == mImports.mVkInstanceFunctions.end() ? nullptr :
        it->second.vkDestroyInstance;
    if (destroy_instance) {
      destroy_instance(instance, pAllocator);
    }
    mImports.mVkInstanceFunctions.erase(mImports.mVkInstanceFunctions.find(instance));
}

uint32_t VulkanSpy::SpyOverride_vkCreateBuffer(
    VkDevice                     device,
    const VkBufferCreateInfo*    pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkBuffer*                    pBuffer) {

  if (is_suspended()) {
    VkBufferCreateInfo override_create_info = *pCreateInfo;
    override_create_info.musage |=
        VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_SRC_BIT;
    return mImports.mVkDeviceFunctions[device].vkCreateBuffer(
        device, &override_create_info, pAllocator, pBuffer);
  } else {
    return mImports.mVkDeviceFunctions[device].vkCreateBuffer(
        device, pCreateInfo, pAllocator, pBuffer);
  }
}

uint32_t VulkanSpy::SpyOverride_vkCreateImage(
    VkDevice                     device,
    const VkImageCreateInfo*     pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkImage*                     pImage) {

  if (is_suspended() || is_observing()) {
    VkImageCreateInfo override_create_info = *pCreateInfo;
    override_create_info.musage |=
        VkImageUsageFlagBits::VK_IMAGE_USAGE_TRANSFER_SRC_BIT;
    return mImports.mVkDeviceFunctions[device].vkCreateImage(
        device, &override_create_info, pAllocator, pImage);
  } else {
    return mImports.mVkDeviceFunctions[device].vkCreateImage(
        device, pCreateInfo, pAllocator, pImage);
  }
}

uint32_t VulkanSpy::SpyOverride_vkCreateSwapchainKHR(
        VkDevice                         device,
        const VkSwapchainCreateInfoKHR*  pCreateInfo,
        const VkAllocationCallbacks*     pAllocator,
        VkSwapchainKHR*                  pImage) {
    if (is_observing() || is_suspended()) {
        VkSwapchainCreateInfoKHR override_create_info = *pCreateInfo;
        override_create_info.mimageUsage |= VkImageUsageFlagBits::VK_IMAGE_USAGE_TRANSFER_SRC_BIT;
        return  mImports.mVkDeviceFunctions[device].vkCreateSwapchainKHR(device, &override_create_info, pAllocator, pImage);
    } else {
        return  mImports.mVkDeviceFunctions[device].vkCreateSwapchainKHR(device, pCreateInfo, pAllocator, pImage);
    }
}

void VulkanSpy::SpyOverride_vkDestroyDevice(VkDevice device, const VkAllocationCallbacks* pAllocator) {
    // First we have to find the function to chain to, then we have to
    // remove this instance from our list, then we forward the call.
    auto it = mImports.mVkDeviceFunctions.find(device);
    gapii::VulkanImports::PFNVKDESTROYDEVICE destroy_device =
        it == mImports.mVkDeviceFunctions.end()
            ? nullptr
            : it->second.vkDestroyDevice;
    if (destroy_device) {
        destroy_device(device, pAllocator);
    }
    mImports.mVkDeviceFunctions.erase(mImports.mVkDeviceFunctions.find(device));
}

uint32_t VulkanSpy::SpyOverride_vkAllocateMemory(
        VkDevice                     device,
        const VkMemoryAllocateInfo*  pAllocateInfo,
        const VkAllocationCallbacks* pAllocator,
        VkDeviceMemory*              pMemory) {

    uint32_t r = mImports.mVkDeviceFunctions[device].vkAllocateMemory(device, pAllocateInfo, pAllocator, pMemory);
    auto l_physical_device = mState.PhysicalDevices[mState.Devices[device]->mPhysicalDevice];
    if (0 != (l_physical_device->mMemoryProperties.mmemoryTypes[pAllocateInfo->mmemoryTypeIndex].mpropertyFlags &
        ((uint32_t)(VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)))) {
        // This is host-coherent memory. Some drivers actually allocate these pages on-demand.
        // This forces all of the pages to be created.
        // This is needed as our coherent memory tracker relies on page-faults which interferes with the
        // on-demand allocation.
        char* memory;
        mImports.mVkDeviceFunctions[device].vkMapMemory(device, *pMemory, 0, pAllocateInfo->mallocationSize, 0, reinterpret_cast<void**>(&memory));
        memset(memory, 0x00, pAllocateInfo->mallocationSize);
        mImports.mVkDeviceFunctions[device].vkUnmapMemory(device, *pMemory);
    }
    return r;
}

// Synthetic functions at tracing time
uint32_t VulkanSpy::CreateImageAndGetMemoryRequirements(
    VkDevice                     device,
    const VkImageCreateInfo*     pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkImage*                     pImage) {

  core::Arena arena;
  uint32_t result =
      gapii::vkCreateImage(device, pCreateInfo, pAllocator, pImage);
  if (result == gapii::VkResult::VK_SUCCESS) {
    gapii::VkMemoryRequirements mem_req{&arena};
    gapii::vkGetImageMemoryRequirements(device, *pImage, &mem_req);
    if ((pCreateInfo->mflags &
         VkImageCreateFlagBits::VK_IMAGE_CREATE_SPARSE_BINDING_BIT) != 0) {
      uint32_t sparse_mem_req_count = 0;
      gapii::vkGetImageSparseMemoryRequirements(device, *pImage,
                                                &sparse_mem_req_count, nullptr);
      std::vector<VkSparseImageMemoryRequirements> sparse_mem_reqs(
          sparse_mem_req_count, VkSparseImageMemoryRequirements{&arena});
      gapii::vkGetImageSparseMemoryRequirements(
          device, *pImage, &sparse_mem_req_count, sparse_mem_reqs.data());
    }
  }
  return result;
}

uint32_t VulkanSpy::CreateBufferAndGetMemoryRequirements(
    VkDevice                     device,
    const VkBufferCreateInfo*    pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkBuffer*                    pBuffer) {

  core::Arena arena;
  uint32_t result =
      gapii::vkCreateBuffer(device, pCreateInfo, pAllocator, pBuffer);
  if (result == gapii::VkResult::VK_SUCCESS) {
    gapii::VkMemoryRequirements mem_req{&arena};
    gapii::vkGetBufferMemoryRequirements(device, *pBuffer, &mem_req);
  }
  return result;
}

uint32_t VulkanSpy::EnumeratePhysicalDevicesAndCacheProperties(
    VkInstance instance, uint32_t* pPhysicalDeviceCount,
    VkPhysicalDevice* pPhysicalDevices) {

  core::Arena arena;
  uint32_t result = gapii::vkEnumeratePhysicalDevices(
      instance, pPhysicalDeviceCount, pPhysicalDevices);
  if ((result == gapii::VkResult::VK_SUCCESS) &&
      (pPhysicalDevices != nullptr)) {
    uint32_t dev_count = *pPhysicalDeviceCount;
    std::vector<VkPhysicalDevice> devs(pPhysicalDevices,
                                       pPhysicalDevices + dev_count);
    for (VkPhysicalDevice dev : devs) {
      gapii::VkPhysicalDeviceProperties dev_prop{&arena};
      gapii::vkGetPhysicalDeviceProperties(dev, &dev_prop);
      uint32_t queue_family_count = 0;
      gapii::vkGetPhysicalDeviceQueueFamilyProperties(dev, &queue_family_count,
                                                      nullptr);
      std::vector<VkQueueFamilyProperties> queue_family_props(
          queue_family_count, VkQueueFamilyProperties{&arena});
      gapii::vkGetPhysicalDeviceQueueFamilyProperties(
          dev, &queue_family_count, queue_family_props.data());
    }
  }
  return result;
}

// Utility functions
uint32_t VulkanSpy::numberOfPNext(CallObserver* observer, const void* pNext) {
  uint32_t counter = 0;
  while (pNext) {
    counter++;
    pNext = reinterpret_cast<const void*>(reinterpret_cast<const uintptr_t*>(pNext)[1]);
  }
  return counter;
}

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