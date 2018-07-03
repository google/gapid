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

#include "gapii/cc/state_serializer.h"
#include "gapii/cc/vulkan_exports.h"
#include "gapii/cc/vulkan_spy.h"

#include "gapis/memory/memory_pb/memory.pb.h"

#include <algorithm>
#include <deque>
#include <map>
#include <tuple>
#include <unordered_set>
#include <vector>

namespace gapii {

template <typename T>
gapil::Ref<QueueObject> GetQueue(const VkQueueToQueueObject__R &queues,
                                 VkDevice device, const gapil::Ref<T> &obj) {
  if (obj->mLastBoundQueue) {
    return obj->mLastBoundQueue;
  }
  for (const auto &qi : queues) {
    if (qi.second->mDevice == device) {
      return qi.second;
    }
  }
  return nullptr;
}

// An invalid value of memory type index
const uint32_t kInvalidMemoryTypeIndex = 0xFFFFFFFF;
// The queue family value when it is ignored
const uint32_t kQueueFamilyIgnore = 0xFFFFFFFF;

// Try to find memory type within the types specified in
// |requirement_type_bits| which is host-visible and non-host-coherent. If a
// non-host-coherent type is not found in the given |requirement_type_bits|,
// then fall back to just host-visible type. Returns the index of the memory
// type. If no proper memory type is found, returns kInvalidMemoryTypeIndex.
uint32_t GetMemoryTypeIndexForStagingResources(
    const VkPhysicalDeviceMemoryProperties &phy_dev_prop,
    uint32_t requirement_type_bits) {
  uint32_t index = 0;
  uint32_t backup_index = kInvalidMemoryTypeIndex;
  while (requirement_type_bits) {
    if (requirement_type_bits & 0x1) {
      VkMemoryPropertyFlags prop_flags =
          phy_dev_prop.mmemoryTypes[index].mpropertyFlags;
      if (prop_flags &
          VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
        if (backup_index == kInvalidMemoryTypeIndex) {
          backup_index = index;
        }
        if ((prop_flags &
             VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_COHERENT_BIT) ==
            0) {
          break;
        }
      }
    }
    requirement_type_bits >>= 1;
    index++;
  }
  if (requirement_type_bits != 0) {
    return index;
  }
  return backup_index;
}

// Returns true if the resource range from |offset| with |size| is fully
// covered in the |bindings|.
bool IsFullyBound(VkDeviceSize offset, VkDeviceSize size,
                  const U64ToVkSparseMemoryBind &bindings) {
  std::vector<uint64_t> resource_offsets;
  resource_offsets.reserve(bindings.count());
  for (const auto &bi : bindings) {
    resource_offsets.push_back(bi.first);
  }
  std::sort(resource_offsets.begin(), resource_offsets.end());
  auto one_after_req_range = std::upper_bound(
      resource_offsets.begin(), resource_offsets.end(), offset + size);
  if (one_after_req_range - resource_offsets.begin() == 0) {
    return false;
  }
  uint64_t i = one_after_req_range - resource_offsets.begin() - 1;
  VkDeviceSize end = offset + size;
  while (i > 0 && end > offset) {
    uint64_t res_offset = resource_offsets[i];
    if (res_offset + bindings.find(res_offset)->second.msize >= end) {
      end = res_offset;
      i--;
      continue;
    }
    return false;
  }
  if (end <= offset) {
    return true;
  }
  if (i == 0) {
    uint64_t res_offset = resource_offsets[0];
    if (res_offset <= offset &&
        res_offset + bindings.find(res_offset)->second.msize >= end) {
      return true;
    }
  }
  return false;
}

// A helper class that contains a temporary buffer that is bound to
// hold incomming data from other GPU resources.
class StagingBuffer {
 public:
  StagingBuffer(core::Arena *arena,
                VulkanImports::VkDeviceFunctions &device_functions,
                VkDevice device,
                const VkPhysicalDeviceMemoryProperties &memory_properties,
                uint32_t size)
      : device_functions_(device_functions), device_(device), size_(size) {
    VkBufferCreateInfo staging_buffer_create_info{arena};
    staging_buffer_create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    staging_buffer_create_info.msize = size;
    staging_buffer_create_info.musage =
        VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT;
    staging_buffer_create_info.msharingMode =
        VkSharingMode::VK_SHARING_MODE_EXCLUSIVE;

    device_functions_.vkCreateBuffer(device_, &staging_buffer_create_info,
                                     nullptr, &staging_buffer_);

    VkMemoryRequirements memory_requirements{arena};
    device_functions_.vkGetBufferMemoryRequirements(device_, staging_buffer_,
                                                    &memory_requirements);

    uint32_t memory_type_index = GetMemoryTypeIndexForStagingResources(
        memory_properties, memory_requirements.mmemoryTypeBits);

    VkMemoryAllocateInfo memory_allocation_info{
        VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
        nullptr,
        memory_requirements.msize,
        memory_type_index,
    };

    device_functions_.vkAllocateMemory(device_, &memory_allocation_info,
                                       nullptr, &staging_memory_);

    device_functions_.vkBindBufferMemory(device_, staging_buffer_,
                                         staging_memory_, 0);
  }

  void *GetMappedMemory() {
    if (!bound_memory_) {
      device_functions_.vkMapMemory(device_, staging_memory_, 0, size_, 0,
                                    &bound_memory_);
    }
    VkMappedMemoryRange range{
        VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,  // sType
        nullptr,                                                 // pNext
        staging_memory_,                                         // memory
        0,                                                       // offset
        size_                                                    // size
    };
    device_functions_.vkInvalidateMappedMemoryRanges(device_, 1, &range);
    return bound_memory_;
  }

  VkBuffer GetBuffer() { return staging_buffer_; }

  ~StagingBuffer() {
    if (staging_buffer_) {
      device_functions_.vkDestroyBuffer(device_, staging_buffer_, nullptr);
    }
    if (staging_memory_) {
      device_functions_.vkFreeMemory(device_, staging_memory_, nullptr);
    }
  }

 private:
  VulkanImports::VkDeviceFunctions &device_functions_;
  VkDevice device_;
  VkBuffer staging_buffer_ = VkBuffer(0);
  VkDeviceMemory staging_memory_ = VkDeviceMemory(0);
  size_t size_;
  void *bound_memory_ = nullptr;
};

class StagingCommandBuffer {
 public:
  StagingCommandBuffer(VulkanImports::VkDeviceFunctions &device_functions,
                       VkDevice device, uint32_t queueFamilyIndex)
      : device_functions_(device_functions), device_(device) {
    VkCommandPoolCreateInfo pool_create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,  // sType
        nullptr,                                                      // pNext
        0,                                                            // flags
        queueFamilyIndex,  // queueFamilyIndex
    };
    device_functions_.vkCreateCommandPool(device_, &pool_create_info, nullptr,
                                          &command_pool_);

    VkCommandBufferAllocateInfo allocate_info = {
        VkStructureType::
            VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,     // sType
        nullptr,                                                // pNext
        command_pool_,                                          // commandLoop
        VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY,  // level
        1,                                                      // count
    };

    device_functions_.vkAllocateCommandBuffers(device, &allocate_info,
                                               &command_buffer_);
    // Set the key of the dispatch tables used in lower layers of the parent
    // dispatchable handle to the new child dispatchable handle. This is
    // necessary as lower layers may use that key to find the dispatch table,
    // and a child handle should share the same dispatch table key.
    // Ref:
    // https://github.com/KhronosGroup/Vulkan-LoaderAndValidationLayers/blob/master/loader/LoaderAndLayerInterface.md#creating-new-dispatchable-objects
    *((const void **)command_buffer_) = *((const void **)device_);

    VkCommandBufferBeginInfo begin_info = {
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,  // sType
        nullptr,                                                       // pNext
        VkCommandBufferUsageFlagBits::
            VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,  // usage
        nullptr                                           // pInheritanceInfo
    };

    device_functions_.vkBeginCommandBuffer(command_buffer_, &begin_info);
  }

  VkCommandBuffer GetBuffer() { return command_buffer_; }

  void FinishAndSubmit(VkQueue queue) {
    device_functions_.vkEndCommandBuffer(command_buffer_);

    VkSubmitInfo submit_info = {
        VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO,  // sType
        nullptr,                                         // pNext
        0,                                               // waitSemaphoreCount
        nullptr,                                         // pWaitSemaphores
        nullptr,                                         // pWaitDstStageMask
        1,                                               // commandBufferCount
        &command_buffer_,                                // pCommandBuffers
        0,                                               // signalSemaphoreCount
        nullptr                                          // pSignalSemaphores
    };

    device_functions_.vkQueueSubmit(queue, 1, &submit_info, VkFence(0));
  }

  ~StagingCommandBuffer() {
    device_functions_.vkDestroyCommandPool(device_, command_pool_, nullptr);
  }

 private:
  VulkanImports::VkDeviceFunctions &device_functions_;
  VkDevice device_;
  VkCommandPool command_pool_;
  VkCommandBuffer command_buffer_;
};

void VulkanSpy::serializeGPUBuffers(StateSerializer *serializer) {
  for (auto &device : mState.Devices) {
    auto &device_functions =
        mImports.mVkDeviceFunctions[device.second->mVulkanHandle];
    device_functions.vkDeviceWaitIdle(device.second->mVulkanHandle);

    // Prep fences
    for (auto &fence : mState.Fences) {
      if (fence.second->mDevice == device.second->mVulkanHandle) {
        ;
        fence.second->mSignaled =
            (device_functions.vkGetFenceStatus(device.second->mVulkanHandle,
                                               fence.second->mVulkanHandle) ==
             VkResult::VK_SUCCESS);
      }
    }

    VkBuffer buffer;
    VkBufferCreateInfo create_info{
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
        nullptr,
        0,
        1,
        VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
        VkSharingMode::VK_SHARING_MODE_EXCLUSIVE,
        0,
        nullptr};
    mImports.mVkDeviceFunctions[device.second->mVulkanHandle].vkCreateBuffer(
        device.second->mVulkanHandle, &create_info, nullptr, &buffer);

    mState.TransferBufferMemoryRequirements[device.second->mVulkanHandle] =
        VkMemoryRequirements{arena()};
    mImports.mVkDeviceFunctions[device.second->mVulkanHandle]
        .vkGetBufferMemoryRequirements(
            device.second->mVulkanHandle, buffer,
            &mState.TransferBufferMemoryRequirements[device.second
                                                         ->mVulkanHandle]);
    mImports.mVkDeviceFunctions[device.second->mVulkanHandle].vkDestroyBuffer(
        device.second->mVulkanHandle, buffer, nullptr);
  }

  for (auto &mem : mState.DeviceMemories) {
    auto &memory = mem.second;
    memory->mData =
        serializer->encodeBuffer<uint8_t>(memory->mAllocationSize, nullptr);
    if (memory->mMappedLocation != nullptr) {
      if (subIsMemoryCoherent(nullptr, nullptr, memory)) {
        trackMappedCoherentMemory(
            nullptr, reinterpret_cast<uint64_t>(memory->mMappedLocation),
            memory->mMappedSize);
      }
    }
  }

  for (auto &buffer : mState.Buffers) {
    VkBuffer buf_handle = buffer.first;
    auto buf = buffer.second;
    auto &device = mState.Devices[buf->mDevice];

    auto &device_functions = mImports.mVkDeviceFunctions[buf->mDevice];
    device_functions.vkDeviceWaitIdle(device->mVulkanHandle);

    const BufferInfo &buf_info = buf->mInfo;
    bool denseBound = buf->mMemory != nullptr;
    bool sparseBound = (buf->mSparseMemoryBindings.count() > 0);
    bool sparseBinding =
        (buf_info.mCreateFlags &
         VkBufferCreateFlagBits::VK_BUFFER_CREATE_SPARSE_BINDING_BIT) != 0;
    bool sparseResidency =
        sparseBinding &&
        (buf_info.mCreateFlags &
         VkBufferCreateFlagBits::VK_BUFFER_CREATE_SPARSE_RESIDENCY_BIT) != 0;
    if (!denseBound && !sparseBound) {
      continue;
    }

    // Note: We treat the dense bind, as a single sparse bind of the entire
    //       resource.
    std::vector<VkSparseMemoryBind> allBindings;
    if (denseBound) {
      allBindings.push_back(VkSparseMemoryBind{
          0,                            // resourceOffset
          buf_info.mSize,               // size
          buf->mMemory->mVulkanHandle,  // memory
          buf->mMemoryOffset,           // memoryOffset
          0,                            // flags
      });
    } else {
      if (!sparseResidency) {
        // It is invalid to read from a partially bound buffer that
        // is not created with SPARSE_RESIDENCY.
        if (!IsFullyBound(0, buf_info.mSize, buf->mSparseMemoryBindings)) {
          continue;
        }
      }
      for (auto &binds : buf->mSparseMemoryBindings) {
        allBindings.push_back(binds.second);
      }
    }

    // TODO(awoloszyn): Avoid blocking on EVERY buffer read.
    // We can either batch them, or spin up a second thread that
    // simply waits for the reads to be done before continuing.
    for (auto &bind : allBindings) {
      if (mState.DeviceMemories.find(bind.mmemory) ==
          mState.DeviceMemories.end()) {
        continue;
      }
      auto &deviceMemory = mState.DeviceMemories[bind.mmemory];
      StagingBuffer stage(
          arena(), device_functions, buf->mDevice,
          mState.PhysicalDevices[mState.Devices[buf->mDevice]->mPhysicalDevice]
              ->mMemoryProperties,
          bind.msize);
      StagingCommandBuffer commandBuffer(
          device_functions, buf->mDevice,
          GetQueue(mState.Queues, buf->mDevice, buf)->mFamily);

      VkBufferCopy region{bind.mresourceOffset, 0, bind.msize};

      device_functions.vkCmdCopyBuffer(commandBuffer.GetBuffer(), buf_handle,
                                       stage.GetBuffer(), 1, &region);

      VkBufferMemoryBarrier barrier{
          VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
          nullptr,
          VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
          VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
          0xFFFFFFFF,
          0xFFFFFFFF,
          stage.GetBuffer(),
          0,
          bind.msize};

      device_functions.vkCmdPipelineBarrier(
          commandBuffer.GetBuffer(),
          VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
          VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT, 0, 0, nullptr, 1,
          &barrier, 0, nullptr);

      commandBuffer.FinishAndSubmit(
          GetQueue(mState.Queues, buf->mDevice, buf)->mVulkanHandle);
      device_functions.vkQueueWaitIdle(
          GetQueue(mState.Queues, buf->mDevice, buf)->mVulkanHandle);

      void *pData = stage.GetMappedMemory();
      memory::Observation observation;
      observation.set_base(bind.mmemoryOffset);
      observation.set_pool(deviceMemory->mData.pool_id());
      serializer->sendData(&observation, true, pData, bind.msize);
    }
  }

  for (auto &image : mState.Images) {
    auto img = image.second;
    const ImageInfo &image_info = img->mInfo;
    auto &device_functions = mImports.mVkDeviceFunctions[img->mDevice];

    auto get_element_size = [this](uint32_t format, uint32_t aspect_bit,
                                   bool in_buffer) -> uint32_t {
      switch (aspect_bit) {
        case VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT:
          return subGetElementAndTexelBlockSize(nullptr, nullptr, format)
              .mElementSize;
        case VkImageAspectFlagBits::VK_IMAGE_ASPECT_DEPTH_BIT:
          return subGetDepthElementSize(nullptr, nullptr, format, in_buffer);
        case VkImageAspectFlagBits::VK_IMAGE_ASPECT_STENCIL_BIT:
          return 1;
      }
      return 0;
    };

    auto next_multiple_of_8 = [](size_t value) -> size_t {
      return (value + 7) & (~7);
    };

    struct pitch {
      uint32_t height_pitch;
      uint32_t depth_pitch;
      uint32_t texel_width;
      uint32_t texel_height;
      uint32_t element_size;
    };

    // block pitch is calculated with the in-image element size.
    auto block_pitch = [this, &get_element_size](
                           const VkExtent3D &extent, uint32_t format,
                           uint32_t mip_level, uint32_t aspect_bit) -> pitch {
      auto elementAndTexelBlockSize =
          subGetElementAndTexelBlockSize(nullptr, nullptr, format);
      const uint32_t texel_width =
          elementAndTexelBlockSize.mTexelBlockSize.mWidth;
      const uint32_t texel_height =
          elementAndTexelBlockSize.mTexelBlockSize.mHeight;

      const uint32_t width =
          subGetMipSize(nullptr, nullptr, extent.mWidth, mip_level);
      const uint32_t height =
          subGetMipSize(nullptr, nullptr, extent.mHeight, mip_level);
      const uint32_t width_in_blocks =
          subRoundUpTo(nullptr, nullptr, width, texel_width);
      const uint32_t height_in_blocks =
          subRoundUpTo(nullptr, nullptr, height, texel_height);
      const uint32_t element_size = get_element_size(format, aspect_bit, false);
      const size_t size = width_in_blocks * height_in_blocks * element_size;

      return pitch{
          uint32_t(width_in_blocks * element_size),
          uint32_t(size),
          uint32_t(elementAndTexelBlockSize.mTexelBlockSize.mWidth),
          uint32_t(elementAndTexelBlockSize.mTexelBlockSize.mHeight),
          uint32_t(element_size),
      };
    };

    struct byte_size_and_extent {
      size_t level_size;
      size_t aligned_level_size;
      size_t level_size_in_buf;
      size_t aligned_level_size_in_buf;
      uint32_t width;
      uint32_t height;
      uint32_t depth;
    };

    auto level_size = [this, &get_element_size, &next_multiple_of_8](
                          const VkExtent3D &extent, uint32_t format,
                          uint32_t mip_level,
                          uint32_t aspect_bit) -> byte_size_and_extent {
      auto elementAndTexelBlockSize =
          subGetElementAndTexelBlockSize(nullptr, nullptr, format);

      const uint32_t texel_width =
          elementAndTexelBlockSize.mTexelBlockSize.mWidth;
      const uint32_t texel_height =
          elementAndTexelBlockSize.mTexelBlockSize.mHeight;
      const uint32_t width =
          subGetMipSize(nullptr, nullptr, extent.mWidth, mip_level);
      const uint32_t height =
          subGetMipSize(nullptr, nullptr, extent.mHeight, mip_level);
      const uint32_t depth =
          subGetMipSize(nullptr, nullptr, extent.mDepth, mip_level);
      const uint32_t width_in_blocks =
          subRoundUpTo(nullptr, nullptr, width, texel_width);
      const uint32_t height_in_blocks =
          subRoundUpTo(nullptr, nullptr, height, texel_height);
      const uint32_t element_size = get_element_size(format, aspect_bit, false);
      const uint32_t element_size_in_buf =
          get_element_size(format, aspect_bit, true);
      const size_t size =
          width_in_blocks * height_in_blocks * depth * element_size;
      const size_t size_in_buf =
          width_in_blocks * height_in_blocks * depth * element_size_in_buf;

      return byte_size_and_extent{size,        next_multiple_of_8(size),
                                  size_in_buf, next_multiple_of_8(size_in_buf),
                                  width,       height,
                                  depth};
    };

    VkImageSubresourceRange img_whole_rng = VkImageSubresourceRange{
        img->mImageAspect,       // aspectMask
        0,                       // baseMipLevel
        img->mInfo.mMipLevels,   // levelCount
        0,                       // baseArrayLayer
        img->mInfo.mArrayLayers  // layerCount
    };

    std::unordered_map<ImageLevel *, byte_size_and_extent> level_sizes;
    walkImageSubRng(img, img_whole_rng,
                    [&serializer, &level_size, &img, &level_sizes](
                        uint32_t aspect, uint32_t layer, uint32_t level) {
                      auto img_level =
                          img->mAspects[aspect]->mLayers[layer]->mLevels[level];
                      level_sizes[img_level.get()] =
                          level_size(img->mInfo.mExtent, img->mInfo.mFormat,
                                     level, aspect);
                      img_level->mData = serializer->encodeBuffer<uint8_t>(
                          level_sizes[img_level.get()].level_size, nullptr);
                    });

    if (img->mIsSwapchainImage) {
      // Don't bind and fill swapchain images memory here
      continue;
    }
    if (image_info.mSamples != VkSampleCountFlagBits::VK_SAMPLE_COUNT_1_BIT) {
      // TODO(awoloszyn): Handle multisampled images here.
      continue;
    }

    // Since we add TRANSFER_SRC_BIT to all the created images (except the
    // swapchain ones), we can copy directly from all such images. Note that
    // later this fact soon will be changed.

    bool denseBound = img->mBoundMemory != nullptr;
    bool sparseBound = (img->mOpaqueSparseMemoryBindings.count() > 0) ||
                       (img->mSparseImageMemoryBindings.count() > 0);
    bool sparseBinding =
        (image_info.mFlags &
         VkImageCreateFlagBits::VK_IMAGE_CREATE_SPARSE_BINDING_BIT) != 0;
    bool sparseResidency =
        sparseBinding &&
        (image_info.mFlags &
         VkImageCreateFlagBits::VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT) != 0;
    if (!denseBound && !sparseBound) {
      continue;
    }
    // First check for validity before we go any further.
    if (sparseBound) {
      if (sparseResidency) {
        bool is_valid = true;
        // If this is a sparsely resident image, then at least ALL metadata
        // must be bound.
        for (const auto &requirements : img->mSparseMemoryRequirements) {
          const auto &prop = requirements.second.mformatProperties;
          if (prop.maspectMask ==
              VkImageAspectFlagBits::VK_IMAGE_ASPECT_METADATA_BIT) {
            if (!IsFullyBound(requirements.second.mimageMipTailOffset,
                              requirements.second.mimageMipTailSize,
                              img->mOpaqueSparseMemoryBindings)) {
              is_valid = false;
              break;
            }
          }
        }
        if (!is_valid) {
          continue;
        }
      } else {
        // If we are not sparsely-resident, then all memory must
        // be bound before we are used.
        if (!IsFullyBound(0, img->mMemoryRequirements.msize,
                          img->mOpaqueSparseMemoryBindings)) {
          continue;
        }
      }
    }

    struct opaque_piece {
      uint32_t aspect_bit;
      uint32_t layer;
      uint32_t level;
    };
    std::vector<opaque_piece> opaque_pieces;
    auto append_image_level_to_opaque_pieces =
        [&img, &opaque_pieces](uint32_t aspect_bit, uint32_t layer,
                               uint32_t level) {
          auto &img_level =
              img->mAspects[aspect_bit]->mLayers[layer]->mLevels[level];
          if (img_level->mLayout == VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED) {
            return;
          }
          opaque_pieces.push_back(opaque_piece{aspect_bit, layer, level});
        };
    if (denseBound || !sparseResidency) {
      walkImageSubRng(img, img_whole_rng, append_image_level_to_opaque_pieces);
    } else {
      for (const auto &req : img->mSparseMemoryRequirements) {
        const auto &prop = req.second.mformatProperties;
        if (prop.maspectMask == img->mImageAspect) {
          if (prop.mflags & VkSparseImageFormatFlagBits::
                                VK_SPARSE_IMAGE_FORMAT_SINGLE_MIPTAIL_BIT) {
            if (!IsFullyBound(req.second.mimageMipTailOffset,
                              req.second.mimageMipTailSize,
                              img->mOpaqueSparseMemoryBindings)) {
              continue;
            }
            VkImageSubresourceRange bound_rng = VkImageSubresourceRange{
                img->mImageAspect,                 // aspectMask
                req.second.mimageMipTailFirstLod,  // baseMipLevel
                image_info.mMipLevels -
                    req.second.mimageMipTailFirstLod,  // levelCount
                0,                                     // baseArrayLayer
                image_info.mArrayLayers,               // layerCount
            };
            walkImageSubRng(img, bound_rng,
                            append_image_level_to_opaque_pieces);
          } else {
            for (uint32_t i = 0; i < uint32_t(image_info.mArrayLayers); i++) {
              VkDeviceSize offset = req.second.mimageMipTailOffset +
                                    i * req.second.mimageMipTailStride;
              if (!IsFullyBound(offset, req.second.mimageMipTailSize,
                                img->mOpaqueSparseMemoryBindings)) {
                continue;
              }
              VkImageSubresourceRange bound_rng = VkImageSubresourceRange{
                  img->mImageAspect,
                  req.second.mimageMipTailFirstLod,
                  image_info.mMipLevels - req.second.mimageMipTailFirstLod,
                  i,
                  1,
              };
              walkImageSubRng(img, bound_rng,
                              append_image_level_to_opaque_pieces);
            }
          }
        }
      }
    }

    // Don't capture images with undefined layout for all its subresources.
    // The resulting data itself will be undefined.
    if (opaque_pieces.size() == 0) {
      continue;
    }

    {
      VkDeviceSize offset = 0;
      std::vector<VkBufferImageCopy> copies_in_order;
      // queue families to corresponding buffer image copies
      std::unordered_map<uint32_t, std::vector<VkBufferImageCopy> > copies;
      // queue families to queues
      std::unordered_map<uint32_t, gapil::Ref<QueueObject>> queues;
      for (auto &piece : opaque_pieces) {
        auto img_level = img->mAspects[piece.aspect_bit]
                             ->mLayers[piece.layer]
                             ->mLevels[piece.level];
        auto queue = GetQueue(mState.Queues, img->mDevice, img_level);
        uint32_t queue_family = queue->mFamily;
        if (copies.find(queue_family) == copies.end()) {
          copies[queue_family] = std::vector<VkBufferImageCopy>();
        }
        if (queues.find(queue_family) == queues.end()) {
            queues[queue_family] = queue;
        }
        auto copy = VkBufferImageCopy{
            offset,  // bufferOffset
            0,       // bufferRowLength
            0,       // bufferImageHeight,
            {
                VkImageAspectFlags(piece.aspect_bit),  // aspectMask
                piece.level,                           // level
                piece.layer,                           // layer
                1,                                     // layerCount
            },
            {0, 0, 0},
            {level_sizes[img_level.get()].width,
             level_sizes[img_level.get()].height,
             level_sizes[img_level.get()].depth}};
        copies[queue_family].push_back(copy);
        copies_in_order.push_back(copy);
        offset += level_sizes[img_level.get()].aligned_level_size_in_buf;
      }

      if (sparseResidency) {
        for (auto &aspect_i :
             subUnpackImageAspectFlags(nullptr, nullptr, img->mImageAspect)
                 ->mBits) {
          uint32_t aspect_bit = aspect_i.second;
          if (img->mSparseImageMemoryBindings.find(aspect_bit) !=
              img->mSparseImageMemoryBindings.end()) {
            for (const auto &layer_i :
                 img->mSparseImageMemoryBindings[aspect_bit]->mLayers) {
              for (const auto &level_i : layer_i.second->mLevels) {
                auto img_level = img->mAspects[aspect_bit]->mLayers[layer_i.first]->mLevels[level_i.first];
                auto queue = GetQueue(mState.Queues, img->mDevice, img_level);
                uint32_t queue_family = queue->mFamily;
                if (copies.find(queue_family) == copies.end()) {
                    copies[queue_family] = std::vector<VkBufferImageCopy>();
                }
                if (queues.find(queue_family) == queues.end()) {
                    queues[queue_family] = queue;
                }
                for (const auto &block_i : level_i.second->mBlocks) {
                  auto copy =
                      VkBufferImageCopy{offset,  // bufferOffset,
                                        0,       // bufferRowLength,
                                        0,       // bufferImageHeight,
                                        VkImageSubresourceLayers{
                                            aspect_bit,  // aspectMask
                                            level_i.first,
                                            layer_i.first,  // baseArrayLayer
                                            1               // layerCount
                                        },
                                        block_i.second->mOffset,
                                        block_i.second->mExtent};

                  copies[queue_family].push_back(copy);
                  copies_in_order.push_back(copy);
                  byte_size_and_extent e =
                      level_size(block_i.second->mExtent, image_info.mFormat, 0,
                                 aspect_bit);
                  offset += e.aligned_level_size_in_buf;
                }
              }
            }
          }
        }
      }

      StagingBuffer stage(
          arena(), device_functions, img->mDevice,
          mState.PhysicalDevices[mState.Devices[img->mDevice]->mPhysicalDevice]
              ->mMemoryProperties,
          offset);

      auto copyImageToBuffer = [&img, &img_whole_rng, &stage, &device_functions,
                                this](
                                   const std::vector<VkBufferImageCopy> &copies,
                                   gapil::Ref<QueueObject> queue) {
        StagingCommandBuffer commandBuffer(device_functions, img->mDevice,
                                           queue->mFamily);
        std::vector<VkImageMemoryBarrier> img_barriers;
        std::vector<uint32_t> old_layouts;
        walkImageSubRng(
            img, img_whole_rng,
            [&img, &img_barriers, &old_layouts](
                uint32_t aspect_bit, uint32_t layer, uint32_t level) {
              auto &img_level =
                  img->mAspects[aspect_bit]->mLayers[layer]->mLevels[level];
              img_barriers.push_back(VkImageMemoryBarrier{
                  VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
                  nullptr,
                  (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,
                  VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT,
                  img_level->mLayout,
                  VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
                  kQueueFamilyIgnore,
                  kQueueFamilyIgnore,
                  img->mVulkanHandle,
                  {VkImageAspectFlags(aspect_bit), level, 1, layer, 1},
              });
              old_layouts.push_back(img_level->mLayout);
            });
        device_functions.vkCmdPipelineBarrier(
            commandBuffer.GetBuffer(),
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT, 0, 0,
            nullptr, 0, nullptr, img_barriers.size(), img_barriers.data());

        device_functions.vkCmdCopyImageToBuffer(
            commandBuffer.GetBuffer(), img->mVulkanHandle,
            VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
            stage.GetBuffer(), copies.size(), copies.data());

        for (size_t i = 0; i < img_barriers.size(); i++) {
          img_barriers[i].msrcAccessMask =
              VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
          img_barriers[i].mdstAccessMask =
              (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1;
          img_barriers[i].moldLayout =
              VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
          img_barriers[i].mnewLayout = old_layouts[i];
        }
        commandBuffer.FinishAndSubmit(queue->mVulkanHandle);
        device_functions.vkQueueWaitIdle(queue->mVulkanHandle);
      };

      for (auto &family_copies : copies) {
        copyImageToBuffer(family_copies.second, queues[family_copies.first]);
      }

      uint8_t *pData = reinterpret_cast<uint8_t *>(stage.GetMappedMemory());
      size_t new_offset = 0;
      for (uint32_t i = 0; i < copies_in_order.size(); ++i) {
        auto &copy = copies_in_order[i];
        size_t next_offset =
            (i == copies_in_order.size() - 1) ? offset : copies_in_order[i + 1].mbufferOffset;
        const uint32_t aspect_bit =
            (uint32_t)copy.mimageSubresource.maspectMask;
        byte_size_and_extent e =
            level_size(copy.mimageExtent, image_info.mFormat, 0, aspect_bit);
        auto bp = block_pitch(copy.mimageExtent, image_info.mFormat,
                              copy.mimageSubresource.mmipLevel, aspect_bit);

        if ((copy.mimageOffset.mx % bp.texel_width != 0) ||
            (copy.mimageOffset.my % bp.texel_height != 0)) {
          // We cannot place partial blocks
          return;
        }
        uint32_t x = (copy.mimageOffset.mx / bp.texel_width) * bp.element_size;
        uint32_t y = (copy.mimageOffset.my / bp.texel_height) * bp.height_pitch;
        uint32_t z = copy.mimageOffset.mz * bp.depth_pitch;

        if ((image_info.mFormat == VkFormat::VK_FORMAT_X8_D24_UNORM_PACK32 ||
             image_info.mFormat == VkFormat::VK_FORMAT_D24_UNORM_S8_UINT) &&
            (aspect_bit == VkImageAspectFlagBits::VK_IMAGE_ASPECT_DEPTH_BIT)) {
          // The width of the depth channel are different for img buf copy.
          size_t element_size_in_img = 3;
          size_t element_size_in_buf = 4;
          // It is always the MSB byte to be stripped.
          uint8_t *buf = pData + new_offset;
          for (size_t i = 0;
               i < e.aligned_level_size_in_buf / element_size_in_buf; i++) {
            if (i < 3) {
              memmove(&buf[i * element_size_in_img],
                      &buf[i * element_size_in_buf], element_size_in_img);
            } else {
              memcpy(&buf[i * element_size_in_img],
                     &buf[i * element_size_in_buf], element_size_in_img);
            }
          }
        } else {
          if (e.level_size_in_buf != e.level_size) {
            // Unhandled case where the element size is different in buffer and
            // image. Should never reach here.
            GAPID_ERROR(
                "[Recovering data for image: %" PRIu64 ", format: %" PRIu32
                "] unhandled case: element size different in buffer and image",
                img->mVulkanHandle, img->mInfo.mFormat);
          }
        }

        memory::Observation observation;
        const uint32_t mip_level = copy.mimageSubresource.mmipLevel;
        const uint32_t array_layer = copy.mimageSubresource.mbaseArrayLayer;
        observation.set_base(x + y + z);
        observation.set_pool(img->mAspects[aspect_bit]
                                 ->mLayers[array_layer]
                                 ->mLevels[mip_level]
                                 ->mData.pool_id());
        serializer->sendData(&observation, true, pData + new_offset,
                             e.level_size);
        new_offset = next_offset;
      }
    }
  }
}

}  // namespace gapii
