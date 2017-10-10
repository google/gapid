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

#include "virtual_swapchain.h"
#include <cassert>
#include <chrono>
#include <functional>
#include <fstream>
#include <iomanip>
#include <iostream>
#include <sstream>
#include <string>

namespace {

// Determines what heap memory should be allocated from, given
// a set of bits.
int32_t
FindMemoryType(const VkPhysicalDeviceMemoryProperties *memory_properties,
               uint32_t memoryTypeBits, VkMemoryPropertyFlags properties) {
  for (int32_t i = 0; i < memory_properties->memoryTypeCount; ++i) {
    if ((memoryTypeBits & (1 << i)) &&
        ((memory_properties->memoryTypes[i].propertyFlags & properties) ==
         properties))
      return i;
  }
  return -1;
}

void null_callback(void *, uint8_t *, size_t) {}
}

namespace swapchain {
VirtualSwapchain::VirtualSwapchain(
    VkDevice device, uint32_t queue,
    const VkPhysicalDeviceProperties *pProperties,
    const VkPhysicalDeviceMemoryProperties *memory_properties,
    const DeviceData *functions,
    const VkSwapchainCreateInfoKHR *_swapchain_info,
    const VkAllocationCallbacks *pAllocator,
    uint32_t pending_image_timeout_in_milliseconds,
    bool always_get_acquired_image)
    : swapchain_info_(*_swapchain_info),
      num_images_(_swapchain_info->minImageCount == 0
                      ? 1
                      : _swapchain_info->minImageCount),
      image_data_(num_images_), should_close_(false),
      device_(device), queue_(queue),
      functions_(functions), pending_image_timeout_in_milliseconds_(
                                 pending_image_timeout_in_milliseconds),
      always_get_acquired_image_(always_get_acquired_image) {
  callback_ = null_callback;
  width_ = _swapchain_info->imageExtent.width;
  height_ = _swapchain_info->imageExtent.height;
  VkPhysicalDeviceMemoryProperties properties = *memory_properties;
  build_swapchain_image_data_ = [this, properties, pAllocator]() {
      SwapchainImageData image_data;

      static const VkFenceCreateInfo fence_info{
          VK_STRUCTURE_TYPE_FENCE_CREATE_INFO, nullptr, 0};
      const VkImageCreateInfo image_create_info{
          VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO, // sType
          nullptr,                             // pNext
          0,                                   // flags
          VK_IMAGE_TYPE_2D,                    // imageType
          swapchain_info_.imageFormat,         // format
          VkExtent3D{swapchain_info_.imageExtent.width,
                     swapchain_info_.imageExtent.height, 1}, // extent
          1,                                                 // mipLevels
          swapchain_info_.imageArrayLayers,                  // arrayLayers
          VK_SAMPLE_COUNT_1_BIT,                             // samples
          VK_IMAGE_TILING_OPTIMAL,                           // tiling
          swapchain_info_.imageUsage | VK_IMAGE_USAGE_TRANSFER_SRC_BIT, // usage
          swapchain_info_.imageSharingMode,      // sharingmode
          swapchain_info_.queueFamilyIndexCount, // queueFamilyIndexCount
          swapchain_info_.pQueueFamilyIndices,   // queueFamilyIndices
          VK_IMAGE_LAYOUT_UNDEFINED,             // initialLayout
      };
      // The size of the buffer that we need is surprisingly easy.
      // Pixel-width * width * height. The GPU will copy into the
      // buffer with the stride we provide.
      // All we want to do here is create a buffer that we can copy
      // the image into.
      // TODO(awolosyn): Currently we know the format is VK_FORMAT_R8G8B8A8_UNORM
      // Handle more formats later if we have other swapchain formats we care
      // about.

      // maximum non-coherent-command-size is 128 bytes
      // This means we can write subsequent layers on 128-byte
      // boundaries
      size_t buffer_memory_size =
          ((ImageByteSize() + 127) & ~127) * swapchain_info_.imageArrayLayers;

      const VkBufferCreateInfo buffer_create_info{
          VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
          nullptr,                              // pNext
          0,                                    // flags
          buffer_memory_size,                   // size
          VK_BUFFER_USAGE_TRANSFER_DST_BIT,     // usage
          VK_SHARING_MODE_EXCLUSIVE,            // sharingMode
          0,                                    // queueFamilyIndexCount
          nullptr                               // pQueueFamilyIndices
        };

      VkCommandPoolCreateInfo command_pool_info{
          VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,      // sType
          nullptr,                                         // pNext
          VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT, // flags
          queue_                                           // queueFamilyIndex
      };

      functions_->vkCreateCommandPool(device_, &command_pool_info, pAllocator,
                                      &command_pool_);

      VkCommandBufferAllocateInfo command_buffer_info{
          VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
          nullptr,                                        // pNext
          command_pool_,                                  // commandPool
          VK_COMMAND_BUFFER_LEVEL_PRIMARY,                // level
          1                                               // count
      };

      // Create the command buffer
      functions_->vkAllocateCommandBuffers(device_, &command_buffer_info,
                                           &image_data.command_buffer_);

      // Create the fence
      {

          functions_->vkCreateFence(device_, &fence_info, pAllocator,
                                    &image_data.fence_);
          functions_->vkResetFences(device_, 1, &image_data.fence_);
      }

      // Create the buffer
      {
          functions_->vkCreateBuffer(device_, &buffer_create_info, pAllocator,
                                     &image_data.buffer_);
          // Create device-memory for the buffer
          {
              VkMemoryRequirements reqs;
              functions_->vkGetBufferMemoryRequirements(
                  device_, image_data.buffer_, &reqs);

              uint32_t memory_type =
                  FindMemoryType(&properties, reqs.memoryTypeBits,
                                 VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT);
              VkMemoryAllocateInfo buffer_memory_info{
                  VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
                  nullptr,                                // pNext
                  reqs.size,                              // allocationSize
                  memory_type                             // memoryTypeIndex
              };

              functions_->vkAllocateMemory(device_, &buffer_memory_info,
                                           pAllocator,
                                           &image_data.buffer_memory_);
              functions_->vkBindBufferMemory(device_, image_data.buffer_,
                                             image_data.buffer_memory_, 0);
          }
      }

      // Create the image
      {

          functions_->vkCreateImage(device_, &image_create_info, pAllocator,
                                    &image_data.image_);
          // Create device-memory for the image
          {
              VkMemoryRequirements reqs;
              functions_->vkGetImageMemoryRequirements(
                  device_, image_data.image_, &reqs);
              uint32_t memory_type =
                  FindMemoryType(&properties, reqs.memoryTypeBits, 0);

              VkMemoryAllocateInfo image_memory_info{
                  VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
                  nullptr,                                // pNext
                  reqs.size,                              // allocationSize
                  memory_type                             // memoryTypeIndex
              };

              functions_->vkAllocateMemory(device_, &image_memory_info, pAllocator,
                                           &image_data.image_memory_);
              functions_->vkBindImageMemory(device_, image_data.image_,
                                            image_data.image_memory_, 0);
          }
      }

      VkBufferMemoryBarrier dest_barrier{
          VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER, // sType
          nullptr,                                 // pNext
          VK_ACCESS_TRANSFER_WRITE_BIT,            // srcAccessMask
          VK_ACCESS_HOST_READ_BIT,                 // dstAccessMask
          VK_QUEUE_FAMILY_IGNORED,                 // srcQueueFamilyIndex
          VK_QUEUE_FAMILY_IGNORED,                 // dstQueueFamilyIndex
          image_data.buffer_,                      // buffer
          0,                                       // offset
          VK_WHOLE_SIZE                            // size
      };

      VkBufferImageCopy region{
          0, // Start of the buffer
          0, // bufferRowLength Tightly packed buffer
          0, // bufferImageHeight same
          VkImageSubresourceLayers{
              VK_IMAGE_ASPECT_COLOR_BIT,          // aspectMask
              0,                                  // mipLevel
              0,                                  // baseArrayLayer
              swapchain_info_.imageArrayLayers},  // imageSubresourceLayers
          VkOffset3D{0, 0, 0},
          VkExtent3D{swapchain_info_.imageExtent.width,
                     swapchain_info_.imageExtent.height, 1}
      };

      VkCommandBufferBeginInfo cbegin{
          VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
          nullptr,                                     // pNext
          0,                                           // flags
          nullptr                                      // pInheritanceInfo
      };

      functions_->vkBeginCommandBuffer(image_data.command_buffer_, &cbegin);
      functions_->vkCmdCopyImageToBuffer(image_data.command_buffer_,
                                         image_data.image_,
                                         VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
                                         image_data.buffer_, 1, &region);
      functions_->vkCmdPipelineBarrier(
          image_data.command_buffer_, VK_PIPELINE_STAGE_TRANSFER_BIT,
          VK_PIPELINE_STAGE_HOST_BIT, 0, 0, nullptr, 1, &dest_barrier, 0, 0);
      functions_->vkEndCommandBuffer(image_data.command_buffer_);

      return image_data;
  };

  // Populate the swapchain image data vector
  for (uint32_t i = 0; i < num_images_; i++) {
    image_data_[i] = build_swapchain_image_data_();
    free_images_.push_back(i);
  }

#ifdef _WIN32
  thread_ = CreateThread(NULL, 0,
                         [](void *data) -> DWORD {
                           ((VirtualSwapchain *)data)->CopyThreadFunc();
                           return 0;
                         },
                         this, 0, nullptr);
#else
  pthread_create(&thread_, nullptr,
                 +[](void *data) -> void * {
                   ((VirtualSwapchain *)data)->CopyThreadFunc();
                   return nullptr;
                 },
                 this);
#endif
}

void VirtualSwapchain::Destroy(const VkAllocationCallbacks *pAllocator) {
  should_close_.store(true);
#ifdef _WIN32
  WaitForSingleObject(thread_, INFINITE);
  CloseHandle(thread_);
#else
  pthread_join(thread_, nullptr);
#endif

  for (size_t i = 0; i < num_images_; ++i) {
    functions_->vkFreeMemory(device_, image_data_[i].image_memory_, pAllocator);
    functions_->vkDestroyImage(device_, image_data_[i].image_, pAllocator);
    functions_->vkFreeMemory(device_, image_data_[i].buffer_memory_,
                             pAllocator);
    functions_->vkDestroyBuffer(device_, image_data_[i].buffer_, pAllocator);
    functions_->vkDestroyFence(device_, image_data_[i].fence_, pAllocator);
  }

  functions_->vkDestroyCommandPool(device_, command_pool_, pAllocator);
}

void VirtualSwapchain::CopyThreadFunc() {
  while (true) {
    uint32_t pending_image = 0;
    // We have to wait until there is a pending image.
    {
      // Wait 10ms for our next image.
      std::unique_lock<threading::mutex> pl(pending_images_lock_);
      while (pending_images_.empty()) {
        if (threading::cv_status::timeout ==
            pending_images_condition_.wait_for(
                pl, std::chrono::milliseconds(
                        pending_image_timeout_in_milliseconds_))) {
          if (should_close_.load()) {
            // One last check to see if there are any more pending images.
            // If not we can return.
            if (!pending_images_.empty())
              break;
            return;
          }
        }
      }
      pending_image = pending_images_.front();
      pending_images_.pop_front();
    }

    VkResult ret = functions_->vkWaitForFences(
        device_, 1, &image_data_[pending_image].fence_, false, UINT64_MAX);
    functions_->vkResetFences(device_, 1, &image_data_[pending_image].fence_);

    void *mapped_value;
    functions_->vkMapMemory(device_, image_data_[pending_image].buffer_memory_,
                            0, VK_WHOLE_SIZE, 0, &mapped_value);
    VkMappedMemoryRange range{
        VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,     // sType
        nullptr,                                   // pNext
        image_data_[pending_image].buffer_memory_, // memory
        0,                                         // offset
        VK_WHOLE_SIZE,                             // size
    };
    functions_->vkInvalidateMappedMemoryRanges(device_, 1, &range);

    uint32_t length = ImageByteSize();
    { callback_(callback_user_data_, (uint8_t *)mapped_value, length); }
    functions_->vkUnmapMemory(device_,
                              image_data_[pending_image].buffer_memory_);
    {
      std::unique_lock<threading::mutex> l(free_images_lock_);
      free_images_.push_back(pending_image);
    }
    free_images_condition_.notify_all();
  }
}

bool VirtualSwapchain::GetImage(uint64_t timeout, uint32_t* image) {
  // A helper function that tries to get a free image.
  auto try_get_image_index = [&](uint32_t* index) {
    uint32_t i = 0;
    if (always_get_acquired_image_) {
      for (auto iter = free_images_.begin(); iter != free_images_.end();
           iter++, i++) {
        if (*iter == *image) {
          *index = *iter;
          free_images_.erase(iter);
          return true;
        }
      }
      return false;
    } else {
      if (free_images_.empty())
        return false;
      *index = free_images_[0];
      free_images_.pop_front();
      return true;
    }
  };

  auto wakeup = std::chrono::nanoseconds(timeout);
  while (true) {
    std::unique_lock<threading::mutex> sl(free_images_lock_);
    if (try_get_image_index(image))
      return true;
    if (timeout == UINT64_MAX) {
      free_images_condition_.wait(sl);
    } else {
      if (free_images_condition_.wait_for(sl, wakeup) ==
          threading::cv_status::timeout) {
        return false;
      }
    }
  }
}

void VirtualSwapchain::SetCallback(void callback(void *, uint8_t *, size_t),
                                   void *user_data) {
  callback_ = callback;
  callback_user_data_ = user_data;
}

uint32_t VirtualSwapchain::ImageByteSize() const {
  // TODO(awoloszyn): Once we support more than RGBA8, have this be
  // more dynamic.
  return width_ * height_ * 4;
}
} // swapchain
