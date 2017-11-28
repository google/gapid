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

#include <algorithm>
#include <deque>
#include <map>
#include <tuple>
#include <vector>
#include <unordered_set>
#include "gapii/cc/vulkan_exports.h"
#include "gapii/cc/vulkan_spy.h"

#ifdef _WIN32
#define alloca _alloca
#else
#include <alloca.h>
#endif

namespace gapii {

template <typename T>
std::shared_ptr<QueueObject> GetQueue(const VkQueueToQueueObject__R& queues,
                                      const std::shared_ptr<T>& obj) {
  if (obj->mLastBoundQueue) {
    return obj->mLastBoundQueue;
  }
  for (const auto& qi : queues) {
    if (qi.second->mDevice == obj->mDevice) {
      return qi.second;
    }
  }
  return nullptr;
}

// An invalid value of memory type index
const uint32_t kInvalidMemoryTypeIndex = 0xFFFFFFFF;

// Try to find memory type within the types specified in
// |requirement_type_bits| which is host-visible and non-host-coherent. If a
// non-host-coherent type is not found in the given |requirement_type_bits|,
// then fall back to just host-visible type. Returns the index of the memory
// type. If no proper memory type is found, returns kInvalidMemoryTypeIndex.
uint32_t GetMemoryTypeIndexForStagingResources(
    const VkPhysicalDeviceMemoryProperties& phy_dev_prop,
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
                  const U64ToVkSparseMemoryBind& bindings) {
  std::vector<uint64_t> resource_offsets;
  resource_offsets.reserve(bindings.size());
  for (const auto& bi : bindings) {
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
  StagingBuffer(VulkanImports::VkDeviceFunctions& device_functions,
                VkDevice device,
                const VkPhysicalDeviceMemoryProperties& memory_properties,
                uint32_t size):
                  device_functions_(device_functions),
                  device_(device),
                  size_(size) {

      VkBufferCreateInfo staging_buffer_create_info{};
      staging_buffer_create_info.msType =
          VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
      staging_buffer_create_info.msize = size;
      staging_buffer_create_info.musage =
          VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT;
      staging_buffer_create_info.msharingMode =
          VkSharingMode::VK_SHARING_MODE_EXCLUSIVE;

      device_functions_.vkCreateBuffer(device_, &staging_buffer_create_info, nullptr,
                                &staging_buffer_);

      VkMemoryRequirements memory_requirements{};
      device_functions_.vkGetBufferMemoryRequirements(device_, staging_buffer_,
                                               &memory_requirements);

      uint32_t memory_type_index = GetMemoryTypeIndexForStagingResources(
        memory_properties, memory_requirements.mmemoryTypeBits);

      VkMemoryAllocateInfo memory_allocation_info{
          VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, nullptr,
          memory_requirements.msize, memory_type_index,
      };

      device_functions_.vkAllocateMemory(device_, &memory_allocation_info, nullptr, &staging_memory_);

      device_functions_.vkBindBufferMemory(device_, staging_buffer_, staging_memory_, 0);
  }

  void* GetMappedMemory() {
    if (!bound_memory_) {
      device_functions_.vkMapMemory(device_, staging_memory_, 0, size_, 0, &bound_memory_);
    }
    VkMappedMemoryRange range {
      VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
      nullptr,         // pNext
      staging_memory_, // memory
      0,               // offset
      size_            // size
    };
    device_functions_.vkInvalidateMappedMemoryRanges(device_, 1, &range);
    return bound_memory_;
  }

  VkBuffer GetBuffer() {
    return staging_buffer_;
  }

  ~StagingBuffer() {
    if (staging_buffer_) {
      device_functions_.vkDestroyBuffer(device_, staging_buffer_, nullptr);
    }
    if (staging_memory_) {
      device_functions_.vkFreeMemory(device_, staging_memory_, nullptr);
    }
  }
private:
  VulkanImports::VkDeviceFunctions& device_functions_;
  VkDevice device_;
  VkBuffer staging_buffer_ = VkBuffer(0);
  VkDeviceMemory staging_memory_ = VkDeviceMemory(0);
  size_t size_;
  void* bound_memory_ = nullptr;
};

class StagingCommandBuffer {
public:
  StagingCommandBuffer(VulkanImports::VkDeviceFunctions& device_functions,
    VkDevice device,
    uint32_t queueFamilyIndex):
      device_functions_(device_functions),
      device_(device) {
    VkCommandPoolCreateInfo pool_create_info = {
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, // sType
      nullptr, // pNext
      0, // flags
      queueFamilyIndex, // queueFamilyIndex
    };
    device_functions_.vkCreateCommandPool(device_, &pool_create_info, nullptr,
        &command_pool_);

    VkCommandBufferAllocateInfo allocate_info = {
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
      nullptr, // pNext
      command_pool_, // commandLoop
      VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY, // level
      1, // count
    };

    device_functions_.vkAllocateCommandBuffers(device, &allocate_info, &command_buffer_);

    VkCommandBufferBeginInfo begin_info = {
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
      nullptr, // pNext
      VkCommandBufferUsageFlagBits::VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT, // usage
      nullptr // pInheritanceInfo
    };

    device_functions_.vkBeginCommandBuffer(command_buffer_, &begin_info);
  }

  VkCommandBuffer GetBuffer() {
    return command_buffer_;
  }

  void FinishAndSubmit(VkQueue queue) {
    device_functions_.vkEndCommandBuffer(command_buffer_);

    VkSubmitInfo submit_info = {
      VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
      nullptr, // pNext
      0, // waitSemaphoreCount
      nullptr, // pWaitSemaphores
      nullptr, // pWaitDstStageMask
      1, // commandBufferCount
      &command_buffer_, // pCommandBuffers
      0, // signalSemaphoreCount
      nullptr // pSignalSemaphores
    };

    device_functions_.vkQueueSubmit(queue, 1, &submit_info, VkFence(0));
  }

  ~StagingCommandBuffer() {
    device_functions_.vkDestroyCommandPool(device_, command_pool_, nullptr);
  }
private:

  VulkanImports::VkDeviceFunctions& device_functions_;
  VkDevice device_;
  VkCommandPool command_pool_;
  VkCommandBuffer command_buffer_;
};

void VulkanSpy::prepareGPUBuffers(PackEncoder* group, std::unordered_set<uint32_t>* gpu_pools) {
    for (auto& device: Devices) {
      auto& device_functions = mImports.mVkDeviceFunctions[device.second->mVulkanHandle];
      device_functions.vkDeviceWaitIdle(device.second->mVulkanHandle);

      VkBuffer buffer;
      VkBufferCreateInfo create_info {
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
        nullptr,
        0,
        1,
        VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
        VkSharingMode::VK_SHARING_MODE_EXCLUSIVE,
        0,
        nullptr
      };
      mImports.mVkDeviceFunctions[device.second->mVulkanHandle].vkCreateBuffer(
        device.second->mVulkanHandle,
        &create_info,
        nullptr,
        &buffer
      );

      TransferBufferMemoryRequirements[device.second->mVulkanHandle] = VkMemoryRequirements{};
      mImports.mVkDeviceFunctions[device.second->mVulkanHandle].vkGetBufferMemoryRequirements(
        device.second->mVulkanHandle,
        buffer,
        &TransferBufferMemoryRequirements[device.second->mVulkanHandle]);
      mImports.mVkDeviceFunctions[device.second->mVulkanHandle].vkDestroyBuffer(
        device.second->mVulkanHandle,
        buffer,
        nullptr);
    }

    for (auto& mem: DeviceMemories) {
        auto& memory = mem.second;
        memory->mData = Slice<uint8_t>(nullptr, memory->mAllocationSize,
            Pool::create_virtual(getPoolID(), memory->mAllocationSize));
        gpu_pools->insert(memory->mData.poolID());
    }

    for (auto& buffer: Buffers) {
      VkBuffer buf_handle = buffer.first;
      auto buf = buffer.second;
      auto& device = Devices[buf->mDevice];

      auto& device_functions = mImports.mVkDeviceFunctions[buf->mDevice];
      device_functions.vkDeviceWaitIdle(device->mVulkanHandle);

      const BufferInfo& buf_info = buf->mInfo;
      bool denseBound = buf->mMemory != nullptr;
      bool sparseBound = (buf->mSparseMemoryBindings.size() > 0);
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
          0, // resourceOffset
          buf_info.mSize, // size
          buf->mMemory->mVulkanHandle, // memory
          buf->mMemoryOffset, // memoryOffset
          0, // flags
        });
      } else {
        if (!sparseResidency) {
          // It is invalid to read from a partially bound buffer that
          // is not created with SPARSE_RESIDENCY.
          if (!IsFullyBound(0, buf_info.mSize, buf->mSparseMemoryBindings)) {
            continue;
          }
        }
        for (auto& binds: buf->mSparseMemoryBindings) {
          allBindings.push_back(binds.second);
        }
      }

      // TODO(awoloszyn): Avoid blocking on EVERY buffer read.
      // We can either batch them, or spin up a second thread that
      // simply waits for the reads to be done before continuing.
      for (auto& bind: allBindings) {
        if (DeviceMemories.find(bind.mmemory) == DeviceMemories.end()) {
          continue;
        }
        auto& deviceMemory = DeviceMemories[bind.mmemory];
        StagingBuffer stage(
          device_functions,
          buf->mDevice,
          PhysicalDevices[Devices[buf->mDevice]->mPhysicalDevice]->mMemoryProperties,
          bind.msize
        );
        StagingCommandBuffer commandBuffer(device_functions, buf->mDevice,
          GetQueue(Queues, buf)->mFamily);

        VkBufferCopy region {
          bind.mresourceOffset,
          0, bind.msize
        };

        device_functions.vkCmdCopyBuffer(commandBuffer.GetBuffer(), buf_handle,
          stage.GetBuffer(), 1, &region);

        VkBufferMemoryBarrier barrier {
          VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
            nullptr,
            VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
            VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
            0xFFFFFFFF, 0xFFFFFFFF, stage.GetBuffer(), 0, bind.msize
        };

        device_functions.vkCmdPipelineBarrier(commandBuffer.GetBuffer(),
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT,
            0, 0, nullptr, 1, &barrier, 0, nullptr);

        commandBuffer.FinishAndSubmit(GetQueue(Queues, buf)->mVulkanHandle);
        device_functions.vkQueueWaitIdle(GetQueue(Queues, buf)->mVulkanHandle);

        void* pData = stage.GetMappedMemory();
        auto resIndex = sendResource(VulkanSpy::kApiIndex, pData, bind.msize);

        memory_pb::Observation observation;
        observation.set_base(bind.mmemoryOffset);
        observation.set_size(bind.msize);
        observation.set_resindex(resIndex);
        observation.set_pool(deviceMemory->mData.poolID());
        group->object(&observation);
      }
    }

    for (auto& image: Images) {
      VkImage img_handle = image.first;
      auto img = image.second;
      const ImageInfo& image_info = img->mInfo;
      auto& device_functions = mImports.mVkDeviceFunctions[img->mDevice];

      if (img->mIsSwapchainImage) {
        // Don't bind and fill swapchain images memory here
        continue;
      }
      if (image_info.mSamples != VkSampleCountFlagBits::VK_SAMPLE_COUNT_1_BIT) {
        // TODO(awoloszyn): Handle multisampled images here.
        continue;
      }
      if (img->mImageAspect !=
          VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT) {
        // TODO(awoloszyn): Handle depth stencil images
        continue;
      }
      if (image_info.mLayout == VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED) {
        // Don't capture images with undefined layout. The resulting data
        // itself will be undefined.
        continue;
      }

      bool denseBound = img->mBoundMemory != nullptr;
      bool sparseBound = (img->mOpaqueSparseMemoryBindings.size() >0) ||
        (img->mSparseImageMemoryBindings.size() > 0);
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
          for (const auto& requirements : img->mSparseMemoryRequirements) {
            const auto& prop = requirements.second.mformatProperties;
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

      struct byte_size_and_extent {
        size_t level_size;
        size_t aligned_level_size;
        uint32_t width;
        uint32_t height;
        uint32_t depth;
      };
      auto level_size =
        [this](const VkExtent3D& extent, uint32_t format, uint32_t mip_level) -> byte_size_and_extent {
        auto elementAndTexelBlockSize =
          subGetElementAndTexelBlockSize(nullptr, nullptr, format);

        const uint32_t texel_width = elementAndTexelBlockSize.mTexelBlockSize.mWidth;
        const uint32_t texel_height = elementAndTexelBlockSize.mTexelBlockSize.mHeight;
        const uint32_t texel_depth = 1;
        const uint32_t width = subGetMipSize(nullptr, nullptr, extent.mWidth, mip_level);
        const uint32_t height = subGetMipSize(nullptr, nullptr, extent.mHeight, mip_level);
        const uint32_t depth = subGetMipSize(nullptr, nullptr, extent.mDepth, mip_level);
        const uint32_t width_in_blocks = subRoundUpTo(nullptr, nullptr, width, texel_width);
        const uint32_t height_in_blocks = subRoundUpTo(nullptr, nullptr, height, texel_height);
        const size_t size = width_in_blocks * height_in_blocks * depth * elementAndTexelBlockSize.mElementSize;
        const size_t next_multiple_of_8 = (size + 7) & (~7);

        return byte_size_and_extent{
          size,
          next_multiple_of_8,
          width,
          height,
          depth
        };
      };

      for (auto& l: img->mLayers) {
        auto& layer = l.second;
        int i = 0;
        for (auto& lev: layer->mLevels) {
          auto& level = lev.second;
          byte_size_and_extent e = level_size(image_info.mExtent, image_info.mFormat, i);
          level->mData = Slice<uint8_t>(nullptr, e.level_size,
            Pool::create_virtual(getPoolID(), e.level_size));
          gpu_pools->insert(level->mData.poolID());
        }
        ++i;
      }

      std::vector<VkImageSubresourceRange> opaque_ranges;
      if (denseBound || !sparseResidency) {
        opaque_ranges.push_back(VkImageSubresourceRange{
          img->mImageAspect, // aspectMask
          0, // baseMipLevel
          image_info.mMipLevels, // levelCount
          0, // baseArrayLayer
          image_info.mArrayLayers // layerCount
        });
      } else {
        for (const auto& req : img->mSparseMemoryRequirements) {
          const auto& prop = req.second.mformatProperties;
          if (prop.maspectMask == img->mImageAspect) {
            if (prop.mflags & VkSparseImageFormatFlagBits::
                    VK_SPARSE_IMAGE_FORMAT_SINGLE_MIPTAIL_BIT) {
              if (!IsFullyBound(req.second.mimageMipTailOffset,
                                req.second.mimageMipTailSize,
                                img->mOpaqueSparseMemoryBindings)) {
                continue;
              }
              opaque_ranges.push_back(VkImageSubresourceRange{
                img->mImageAspect, // aspectMask
                req.second.mimageMipTailFirstLod, // baseMipLevel
                image_info.mMipLevels - req.second.mimageMipTailFirstLod, // levelCount
                0, // baseArrayLayer
                image_info.mArrayLayers // layerCount
              });
            } else {
              for (uint32_t i = 0; i < uint32_t(image_info.mArrayLayers); i++) {
                VkDeviceSize offset = req.second.mimageMipTailOffset +
                                      i * req.second.mimageMipTailStride;
                if (!IsFullyBound(offset, req.second.mimageMipTailSize,
                      img->mOpaqueSparseMemoryBindings)) {
                        continue;
                }
                opaque_ranges.push_back(VkImageSubresourceRange{
                    img->mImageAspect,
                    req.second.mimageMipTailFirstLod,
                    image_info.mMipLevels - req.second.mimageMipTailFirstLod,
                    i,
                    1,
                });
              }
            }
          }
        }
      }

      {
        VkDeviceSize offset = 0;
        std::vector<VkBufferImageCopy> copies;
        for (auto& range: opaque_ranges) {
          for (size_t i = 0; i < range.mlevelCount; ++i) {
            uint32_t mip_level = range.mbaseMipLevel + i;
            byte_size_and_extent e = level_size(image_info.mExtent, image_info.mFormat, mip_level);
            copies.push_back(VkBufferImageCopy{
              offset, // bufferOffset,
              0, // bufferRowLength,
              0, // bufferImageHeight,
              {
                img->mImageAspect, // aspectMask
                mip_level,
                range.mbaseArrayLayer, // baseArrayLayer
                range.mlayerCount // layerCount
              },
              {
                0, 0, 0
              },
              {
                e.width,
                e.height,
                e.depth
              }
            });
            offset += (e.aligned_level_size * range.mlayerCount);
          }
        }

        if (sparseResidency) {
          if (img->mSparseImageMemoryBindings.find(img->mImageAspect) !=
                img->mSparseImageMemoryBindings.end()) {
            for (const auto& layer_i : img->mSparseImageMemoryBindings[img->mImageAspect]->mLayers) {
              for (const auto& level_i : layer_i.second->mLevels) {
                for (const auto& block_i : level_i.second->mBlocks) {
                  copies.push_back(VkBufferImageCopy{
                    offset, // bufferOffset,
                    0, // bufferRowLength,
                    0, // bufferImageHeight,
                    VkImageSubresourceLayers{
                      img->mImageAspect, // aspectMask
                      level_i.first,
                      layer_i.first, // baseArrayLayer
                      1 // layerCount
                    },
                    block_i.second->mOffset,
                    block_i.second->mExtent
                  });
                  byte_size_and_extent e = level_size(block_i.second->mExtent, image_info.mFormat, 0);
                  offset += e.aligned_level_size;
                }
              }
            }
          }
        }

        StagingBuffer stage(
          device_functions,
          img->mDevice,
          PhysicalDevices[Devices[img->mDevice]->mPhysicalDevice]->mMemoryProperties,
          offset
        );

        StagingCommandBuffer commandBuffer(device_functions, img->mDevice,
          GetQueue(Queues, img)->mFamily);

        VkImageMemoryBarrier img_barrier{
            VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
            nullptr,
            (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,
            VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT,
            image_info.mLayout,
            VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
            0xFFFFFFFF,
            0xFFFFFFFF,
            img->mVulkanHandle,
            {img->mImageAspect, 0, image_info.mMipLevels,
             0, image_info.mArrayLayers},
        };

        device_functions.vkCmdPipelineBarrier(commandBuffer.GetBuffer(),
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
            0, 0, nullptr, 0, nullptr, 1, &img_barrier);

        device_functions.vkCmdCopyImageToBuffer(commandBuffer.GetBuffer(),
            img->mVulkanHandle, VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
            stage.GetBuffer(), copies.size(), copies.data());

        img_barrier.msrcAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
        img_barrier.mdstAccessMask = (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) -1;
        img_barrier.moldLayout = VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
        img_barrier.mnewLayout = img->mInfo.mLayout;


        VkBufferMemoryBarrier buf_barrier{
          VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
            nullptr,
            VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
            VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
            0xFFFFFFFF, 0xFFFFFFFF, stage.GetBuffer(), 0, offset
        };

        device_functions.vkCmdPipelineBarrier(commandBuffer.GetBuffer(),
          VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
          VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
          0, 0, nullptr, 1, &buf_barrier, 1, &img_barrier);

        commandBuffer.FinishAndSubmit(GetQueue(Queues, img)->mVulkanHandle);
        device_functions.vkQueueWaitIdle(GetQueue(Queues, img)->mVulkanHandle);

        uint8_t* pData = reinterpret_cast<uint8_t*>(stage.GetMappedMemory());
        size_t new_offset = 0;
        for (uint32_t i = 0; i < copies.size(); ++i) {
          auto& copy = copies[i];
          size_t next_offset =
            (i == copies.size() - 1)? offset: copies[i+1].mbufferOffset;

          size_t copy_size = next_offset - new_offset;
          for (size_t j = copy.mimageSubresource.mbaseArrayLayer;
              j < copy.mimageSubresource.mbaseArrayLayer + copy.mimageSubresource.mlayerCount; ++j) {
                byte_size_and_extent e = level_size(copy.mimageExtent, image_info.mFormat, 0);
                byte_size_and_extent offs = level_size(
                    VkExtent3D{
                      static_cast<uint32_t>(copy.mimageOffset.mx),
                      static_cast<uint32_t>(copy.mimageOffset.my),
                        static_cast<uint32_t>(copy.mimageOffset.mz)
                    }, image_info.mFormat, 0);

                auto resIndex = sendResource(VulkanSpy::kApiIndex, pData + new_offset,
                    e.level_size);
                new_offset += e.aligned_level_size;
                const uint32_t mip_level = copy.mimageSubresource.mmipLevel;
                const uint32_t array_layer = j;
                memory_pb::Observation observation;
                observation.set_base(offs.level_size);
                observation.set_size(e.level_size);
                observation.set_resindex(resIndex);
                observation.set_pool(
                  img->mLayers[array_layer]->mLevels[mip_level]->mData.poolID()
                );
                group->object(&observation);
          }
          new_offset = next_offset;
        }
      }
    }
}

} // namesapce gapii