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

namespace {
class TemporaryShaderModule {
 public:
  TemporaryShaderModule(CallObserver* observer, VulkanSpy* spy)
      : observer_(observer), spy_(spy), temporary_shader_modules_() {}

  VkShaderModule CreateShaderModule(
      std::shared_ptr<ShaderModuleObject> module_obj) {
    if (!module_obj) {
      return VkShaderModule(0);
    }
    VkShaderModuleCreateInfo create_info{
        VkStructureType::VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,  // sType
        nullptr,                                                       // pNext
        0,                                                             // flags
        static_cast<size_t>(module_obj->mWords.size()),               // codeSize
        module_obj->mWords.begin(),               // pCode
    };
    spy_->RecreateShaderModule(observer_, module_obj->mDevice, &create_info,
                               &module_obj->mVulkanHandle);
    temporary_shader_modules_.push_back(module_obj);
    return module_obj->mVulkanHandle;
  }

  ~TemporaryShaderModule() {
    for (auto m : temporary_shader_modules_) {
      spy_->RecreateDestroyShaderModule(observer_, m->mDevice,
                                        m->mVulkanHandle);
    }
  }

 private:
  CallObserver* observer_;
  VulkanSpy* spy_;
  std::vector<std::shared_ptr<ShaderModuleObject>> temporary_shader_modules_;
};

VkRenderPass RebuildRenderPass(
    CallObserver* observer, VulkanSpy* spy,
    std::shared_ptr<RenderPassObject>& render_pass_object) {
  if (!render_pass_object) {
    return VkRenderPass(0);
  }
  VkRenderPassCreateInfo create_info = {
      VkStructureType::VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
      nullptr,
      0,
      0,
      nullptr,
      0,
      nullptr,
      0,
      nullptr};
  RenderPassObject& render_pass = *render_pass_object;
  std::vector<VkAttachmentDescription> attachment_descriptions;
  for (size_t i = 0; i < render_pass.mAttachmentDescriptions.size(); ++i) {
    attachment_descriptions.push_back(render_pass.mAttachmentDescriptions[i]);
  }
  struct SubpassDescriptionData {
    std::vector<VkAttachmentReference> inputAttachments;
    std::vector<VkAttachmentReference> colorAttachments;
    std::vector<VkAttachmentReference> resolveAttachments;
    std::vector<uint32_t> preserveAttachments;
  };
  std::vector<std::unique_ptr<SubpassDescriptionData>> descriptionData;
  std::vector<VkSubpassDescription> subpassDescriptions;
  for (size_t i = 0; i < render_pass.mSubpassDescriptions.size(); ++i) {
    auto& s = render_pass.mSubpassDescriptions[i];
    auto dat =
        std::unique_ptr<SubpassDescriptionData>(new SubpassDescriptionData());
    for (size_t j = 0; j < s.mInputAttachments.size(); ++j) {
      dat->inputAttachments.push_back(s.mInputAttachments[j]);
    }
    for (size_t j = 0; j < s.mColorAttachments.size(); ++j) {
      dat->colorAttachments.push_back(s.mColorAttachments[j]);
    }
    for (size_t j = 0; j < s.mResolveAttachments.size(); ++j) {
      dat->resolveAttachments.push_back(s.mResolveAttachments[j]);
    }
    for (size_t j = 0; j < s.mPreserveAttachments.size(); ++j) {
      dat->preserveAttachments.push_back(s.mPreserveAttachments[j]);
    }
    subpassDescriptions.push_back({});
    auto& d = subpassDescriptions.back();
    d.mflags = s.mFlags;
    d.mpipelineBindPoint = s.mPipelineBindPoint;
    d.minputAttachmentCount = dat->inputAttachments.size();
    d.mpInputAttachments = dat->inputAttachments.data();
    d.mcolorAttachmentCount = dat->colorAttachments.size();
    d.mpColorAttachments = dat->colorAttachments.data();
    d.mpResolveAttachments = dat->resolveAttachments.size() > 0
                                 ? dat->resolveAttachments.data()
                                 : nullptr;
    d.mpreserveAttachmentCount = dat->preserveAttachments.size();
    d.mpPreserveAttachments = dat->preserveAttachments.data();

    if (s.mDepthStencilAttachment) {
      d.mpDepthStencilAttachment = s.mDepthStencilAttachment.get();
    }
    descriptionData.push_back(std::move(dat));
  }
  std::vector<VkSubpassDependency> subpassDependencies;
  for (size_t i = 0; i < render_pass.mSubpassDependencies.size(); ++i) {
    subpassDependencies.push_back(render_pass.mSubpassDependencies[i]);
  }
  create_info.mattachmentCount = attachment_descriptions.size();
  create_info.mpAttachments = attachment_descriptions.data();
  create_info.msubpassCount = subpassDescriptions.size();
  create_info.mpSubpasses = subpassDescriptions.data();
  create_info.mdependencyCount = subpassDependencies.size();
  create_info.mpDependencies = subpassDependencies.data();
  spy->RecreateRenderPass(observer, render_pass.mDevice, &create_info,
                          &render_pass.mVulkanHandle);
  return render_pass.mVulkanHandle;
}

class TemporaryRenderPass {
 public:
  TemporaryRenderPass(CallObserver* observer, VulkanSpy* spy)
      : observer_(observer), spy_(spy), temporary_render_passes_() {}

  VkRenderPass CreateRenderPass(
      std::shared_ptr<RenderPassObject>& render_pass_object) {
    RebuildRenderPass(observer_, spy_, render_pass_object);
    temporary_render_passes_.push_back(render_pass_object);
    return render_pass_object->mVulkanHandle;
  }

  ~TemporaryRenderPass() {
    for (auto m : temporary_render_passes_) {
      spy_->RecreateDestroyRenderPass(observer_, m->mDevice, m->mVulkanHandle);
    }
  }

  bool has(VkRenderPass renderpass) {
    return std::find_if(temporary_render_passes_.begin(),
                        temporary_render_passes_.end(),
                        [renderpass](std::shared_ptr<RenderPassObject>& p) {
                          return p->mVulkanHandle == renderpass;
                        }) != temporary_render_passes_.end();
  }

 private:
  CallObserver* observer_;
  VulkanSpy* spy_;
  std::vector<std::shared_ptr<RenderPassObject>> temporary_render_passes_;
};

template <typename ObjectClass>
VkDevice getObjectCreatingDevice(const std::shared_ptr<ObjectClass>& obj) {
  return obj->mDevice;
}

template <>
VkDevice getObjectCreatingDevice<InstanceObject>(
    const std::shared_ptr<InstanceObject>& obj) {
  return VkDevice(0);
}


template <>
VkDevice getObjectCreatingDevice<PhysicalDeviceObject>(
    const std::shared_ptr<PhysicalDeviceObject>& obj) {
  return VkDevice(0);
}

template <>
VkDevice getObjectCreatingDevice<DeviceObject>(
    const std::shared_ptr<DeviceObject>& obj) {
  return obj->mVulkanHandle;
}

template <>
VkDevice getObjectCreatingDevice<SurfaceObject>(
    const std::shared_ptr<SurfaceObject>& obj) {
  return VkDevice(0);
}

template <typename ObjectClass>
void recreateDebugInfo(VulkanSpy* spy, CallObserver* observer,
                       uint32_t objectType,
                       const std::shared_ptr<ObjectClass>& obj) {
  const std::shared_ptr<VulkanDebugMarkerInfo>& info = obj->mDebugInfo;
  if (!info) {
    return;
  }
  uint64_t object = static_cast<uint64_t>(obj->mVulkanHandle);
  if (info->mObjectName.length() > 0) {
    VkDebugMarkerObjectNameInfoEXT name_info{
        VkStructureType::
            VK_STRUCTURE_TYPE_DEBUG_MARKER_OBJECT_NAME_INFO_EXT,  // sType
        nullptr,                                                  // pNext
        objectType,                                               // objectType
        object,                                                   // object
        const_cast<char*>(info->mObjectName.c_str()),             // pObjectName
        // type of pObjectName is const char* in the Spec, but in GAPID header
        // its type is char*
    };
    spy->RecreateDebugMarkerSetObjectNameEXT(
        observer, getObjectCreatingDevice(obj), &name_info);
  }
  VkDebugMarkerObjectTagInfoEXT tag_info{
      VkStructureType::
          VK_STRUCTURE_TYPE_DEBUG_MARKER_OBJECT_TAG_INFO_EXT,  // sType
      nullptr,                                                 // pNext
      objectType,                                              // objectType
      object,                                                  // object
      info->mTagName,                                          // tagName
      static_cast<size_val>(info->mTag.size()),            // tagSize
      reinterpret_cast<void*>(info->mTag.begin()),          // pTag
  };
  spy->RecreateDebugMarkerSetObjectTagEXT(
      observer, getObjectCreatingDevice(obj), &tag_info);
}

// Get a proper queue to handle operations on |obj|. For image and buffer
// objects, if there is a last bound queue, returns the object of that last
// bound queue, Otherwise, and for other objects, returns a queue object whose
// handle is created with the same device of the object. If such a queue object
// is not found, returns nullptr.
template <typename ObjectClass>
std::shared_ptr<QueueObject> GetQueue(const VkQueueToQueueObject__R& queues,
                                      const std::shared_ptr<ObjectClass>& obj) {
  for (const auto& qi : queues) {
    if (qi.second->mDevice == getObjectCreatingDevice(obj)) {
      return qi.second;
    }
  }
  return nullptr;
}

template <>
std::shared_ptr<QueueObject> GetQueue(const VkQueueToQueueObject__R& queues,
                                      const std::shared_ptr<ImageObject>& obj) {
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

template <>
std::shared_ptr<QueueObject> GetQueue(const VkQueueToQueueObject__R& queues,
                                      const std::shared_ptr<BufferObject>& obj) {
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
  return true;
}

// A helper class to copy the data from a given buffer range or subresources of
// an image. It handles the creation of extra Vulkan handles to do the job and
// cleaning of those helper handles.
class CopyDataHelper {
  public:
   CopyDataHelper(VulkanSpy* spy,
       VulkanImports::VkDeviceFunctions& device_functions,
                  VkDevice dev, std::shared_ptr<QueueObject> queue,
                  const VkPhysicalDeviceMemoryProperties& mem_props)
       : spy_(spy),
         dev_funcs_(device_functions),
         dev_(dev),
         queue_obj_(queue),
         cmd_pool_(0),
         cmd_buf_(0),
         mem_(0),
         phy_dev_mem_props_(mem_props),
         staging_buf_(0) {
   }

   // not movable, not copyable
   CopyDataHelper(const CopyDataHelper&) = delete;
   CopyDataHelper(CopyDataHelper&&) = delete;
   CopyDataHelper& operator=(const CopyDataHelper&) = delete;
   CopyDataHelper& operator=(CopyDataHelper&&) = delete;

  ~CopyDataHelper() {
    Clean();
  }

  std::tuple<uint32_t, std::vector<uint8_t>> GetBufferData(VkBuffer src_buf, VkDeviceSize offset, VkDeviceSize data_size) {
    uint32_t mem_type_index = Prepare(data_size);
    VkBufferCopy region {offset, 0, data_size};
    dev_funcs_.vkCmdCopyBuffer(cmd_buf_, src_buf, staging_buf_, 1, &region);
    VkBufferMemoryBarrier barrier {
      VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
        nullptr,
        VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
        VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
        0xFFFFFFFF, 0xFFFFFFFF, staging_buf_, 0, data_size
    };
    dev_funcs_.vkCmdPipelineBarrier(cmd_buf_,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT,
        0, 0, nullptr, 1, &barrier, 0, nullptr);
    std::vector<uint8_t> data = SubmitAndGetData(data_size);
    auto return_tuple = std::make_tuple(mem_type_index, data);
    Clean();
    return return_tuple;
  }


  std::tuple<uint32_t, std::vector<uint8_t>> GetImageSparseSubresourceData(
      std::shared_ptr<ImageObject> img, const VkImageSubresource& subres, const VkOffset3D& offset,
      const VkExtent3D& extent) {
    VkExtent3D gran{};
    bool found = false;
    for (auto& ri : img->mSparseMemoryRequirements) {
      if (ri.second.mformatProperties.maspectMask == subres.maspectMask) {
        gran = ri.second.mformatProperties.mimageGranularity;
        found = true;
      }
    }
    if (!found) {
      GAPID_ERROR("Cannot find sparse memory requirement for apsect: %u", subres.maspectMask);
      return std::make_tuple(0, std::vector<uint8_t>());
    }
    VkDeviceSize dx = (extent.mWidth + gran.mWidth - 1) / gran.mWidth;
    VkDeviceSize dy = (extent.mHeight + gran.mHeight - 1) / gran.mHeight;
    VkDeviceSize dz = (extent.mDepth + gran.mDepth - 1) / gran.mDepth;
    VkDeviceSize data_size = dx * dy * dz * img->mMemoryRequirements.malignment;
    uint32_t mem_type_index = Prepare(data_size);
    VkBufferImageCopy copy{
      0, 0, 0, {subres.maspectMask, subres.mmipLevel, subres.marrayLayer, 1},
      offset, extent,
    };
    VkImageMemoryBarrier img_barrier{
        VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
        nullptr,
        (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,
        VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT,
        img->mInfo.mLayout,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        0xFFFFFFFF,
        0xFFFFFFFF,
        img->mVulkanHandle,
        {subres.maspectMask, subres.mmipLevel, 1,
         subres.marrayLayer, 1},
    };
    dev_funcs_.vkCmdPipelineBarrier(cmd_buf_,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        0, 0, nullptr, 0, nullptr, 1, &img_barrier);
    dev_funcs_.vkCmdCopyImageToBuffer(cmd_buf_, img->mVulkanHandle,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        staging_buf_, 1, &copy);
    img_barrier.msrcAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
    img_barrier.mdstAccessMask = (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) -1;
    img_barrier.moldLayout = VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
    img_barrier.mnewLayout = img->mInfo.mLayout;

    VkBufferMemoryBarrier buf_barrier{
      VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
        nullptr,
        VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
        VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
        0xFFFFFFFF, 0xFFFFFFFF, staging_buf_, 0, data_size
    };
    dev_funcs_.vkCmdPipelineBarrier(cmd_buf_,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT,
        0, 0, nullptr, 1, &buf_barrier, 1, &img_barrier);
    std::vector<uint8_t> data = SubmitAndGetData(data_size);
    auto return_tuple = std::make_tuple(mem_type_index, data);
    Clean();
    return return_tuple;
  }

  std::tuple<uint32_t, std::vector<uint8_t>> GetImageSubrangeData(
      std::shared_ptr<ImageObject> img, const VkImageSubresourceRange& range, VkDeviceSize data_size) {
    uint32_t mem_type_index = Prepare(data_size);
    std::vector<VkBufferImageCopy> copies(range.mlevelCount);
    uint64_t buffer_offset = 0;
    for (uint32_t mip = range.mbaseMipLevel; mip < range.mbaseMipLevel + range.mlevelCount; mip++) {
      const uint32_t width = static_cast<uint32_t>(spy_->subGetMipSize(
          nullptr, nullptr, img->mInfo.mExtent.mWidth, mip));
      const uint32_t height = static_cast<uint32_t>(spy_->subGetMipSize(
          nullptr, nullptr, img->mInfo.mExtent.mHeight, mip));
      const uint32_t depth = static_cast<uint32_t>(spy_->subGetMipSize(
          nullptr, nullptr, img->mInfo.mExtent.mDepth, mip));
      copies[mip - range.mbaseMipLevel] = {
          buffer_offset,
          0, // tight pack
          0, // tight pack
          {range.maspectMask, mip, range.mbaseArrayLayer,
           range.mlayerCount},
          {0, 0, 0},
          {width, height, depth},
      };
      buffer_offset += spy_->subInferImageLevelSize(nullptr, nullptr, img, mip);
    }

    VkImageMemoryBarrier img_barrier{
        VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
        nullptr,
        (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,
        VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT,
        img->mInfo.mLayout,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        0xFFFFFFFF,
        0xFFFFFFFF,
        img->mVulkanHandle,
        {range.maspectMask, range.mbaseMipLevel, range.mlevelCount,
         range.mbaseArrayLayer, range.mlayerCount},
    };

    dev_funcs_.vkCmdPipelineBarrier(cmd_buf_,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        0, 0, nullptr, 0, nullptr, 1, &img_barrier);
    dev_funcs_.vkCmdCopyImageToBuffer(cmd_buf_, img->mVulkanHandle,
        VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        staging_buf_, copies.size(), copies.data());
    img_barrier.msrcAccessMask = VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
    img_barrier.mdstAccessMask = (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) -1;
    img_barrier.moldLayout = VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
    img_barrier.mnewLayout = img->mInfo.mLayout;

    VkBufferMemoryBarrier buf_barrier{
      VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
        nullptr,
        VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
        VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
        0xFFFFFFFF, 0xFFFFFFFF, staging_buf_, 0, data_size
    };
    dev_funcs_.vkCmdPipelineBarrier(cmd_buf_,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
        VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT,
        0, 0, nullptr, 1, &buf_barrier, 1, &img_barrier);
    std::vector<uint8_t> data = SubmitAndGetData(data_size);
    auto return_tuple = std::make_tuple(mem_type_index, data);
    Clean();
    return return_tuple;
  }

  private:
  uint32_t Prepare(VkDeviceSize buf_size) {
     VkBufferCreateInfo staging_buf_create_info{};
     staging_buf_create_info.msType =
         VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
     staging_buf_create_info.msize = buf_size;
     staging_buf_create_info.musage =
         VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT;
     staging_buf_create_info.msharingMode =
         VkSharingMode::VK_SHARING_MODE_EXCLUSIVE;
     dev_funcs_.vkCreateBuffer(dev_, &staging_buf_create_info, nullptr,
                               &staging_buf_);
     VkMemoryRequirements staging_buf_mem_req{};
     dev_funcs_.vkGetBufferMemoryRequirements(dev_, staging_buf_,
                                              &staging_buf_mem_req);
     uint32_t mem_type_index = GetMemoryTypeIndexForStagingResources(
         phy_dev_mem_props_, staging_buf_mem_req.mmemoryTypeBits);
     VkMemoryAllocateInfo mem_alloc_info{
         VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, nullptr,
         buf_size, mem_type_index,
     };
     dev_funcs_.vkAllocateMemory(dev_, &mem_alloc_info, nullptr, &mem_);
     dev_funcs_.vkBindBufferMemory(dev_, staging_buf_, mem_, 0);
     VkCommandPoolCreateInfo cmd_pool_create_info{
         VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, nullptr,
         VkCommandPoolCreateFlagBits::VK_COMMAND_POOL_CREATE_TRANSIENT_BIT,
         queue_obj_->mFamily};
     dev_funcs_.vkCreateCommandPool(dev_, &cmd_pool_create_info, nullptr,
                                    &cmd_pool_);
     VkCommandBufferAllocateInfo cmd_buf_alloc_info{
         VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
         nullptr,
         cmd_pool_,
         VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY,
         1,
     };
     dev_funcs_.vkAllocateCommandBuffers(dev_, &cmd_buf_alloc_info, &cmd_buf_);
    VkCommandBufferBeginInfo begin_info{
      VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
        nullptr, VkCommandBufferUsageFlagBits::VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT, nullptr};
    dev_funcs_.vkBeginCommandBuffer(cmd_buf_, &begin_info);
     return mem_type_index;
  }

  std::vector<uint8_t> SubmitAndGetData(VkDeviceSize data_size) {
    dev_funcs_.vkEndCommandBuffer(cmd_buf_);
    VkSubmitInfo submit_info {
      VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO, nullptr, 0, nullptr, nullptr,
        1, &cmd_buf_, 0, nullptr,
    };
    dev_funcs_.vkQueueSubmit(queue_obj_->mVulkanHandle, 1, &submit_info, VkFence(0));
    dev_funcs_.vkQueueWaitIdle(queue_obj_->mVulkanHandle);
    uint8_t* data = nullptr;
    dev_funcs_.vkMapMemory(dev_, mem_, 0, data_size, 0, (void**)(&data));
    VkMappedMemoryRange rng {
      VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, nullptr,
        mem_, 0, data_size};
    dev_funcs_.vkInvalidateMappedMemoryRanges(dev_, 1, &rng);
    return std::vector<uint8_t>(data, data+size_t(data_size));
  }

  void Clean() {
    if (cmd_pool_ != VkCommandPool(0)) {
      dev_funcs_.vkDestroyCommandPool(dev_, cmd_pool_, nullptr);
      cmd_pool_ = VkCommandPool(0);
      cmd_buf_ = VkCommandBuffer(0);
    }
    if (staging_buf_ != VkBuffer(0)) {
      dev_funcs_.vkDestroyBuffer(dev_, staging_buf_, nullptr);
      staging_buf_ = VkBuffer(0);
    }
    if (mem_ != VkDeviceMemory(0)) {
      dev_funcs_.vkFreeMemory(dev_, mem_, nullptr);
      mem_ = VkDeviceMemory(0);
    }
  }

   VulkanSpy* spy_;
   VulkanImports::VkDeviceFunctions& dev_funcs_;
   VkDevice dev_;
   std::shared_ptr<QueueObject> queue_obj_;
   VkCommandPool cmd_pool_;
   VkCommandBuffer cmd_buf_;
   VkDeviceMemory mem_;
   VkPhysicalDeviceMemoryProperties phy_dev_mem_props_;
   VkBuffer staging_buf_;
};
}  // anonymous namespace

void VulkanSpy::EnumerateVulkanResources(CallObserver* observer) {
  for (auto& instance : Instances) {
    VkInstanceCreateInfo create_info = {};
    // TODO(awoloszyn): Add ApplicationInfo here if we
    //                   choose
    create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO;
    create_info.menabledLayerCount = instance.second->mEnabledLayers.size();
    create_info.menabledExtensionCount =
        instance.second->mEnabledExtensions.size();

    const char** enabled_layers =
        (const char**)alloca(sizeof(char*) * create_info.menabledLayerCount);
    const char** enabled_extensions = (const char**)alloca(
        sizeof(char*) * create_info.menabledExtensionCount);

    for (size_t i = 0; i < create_info.menabledLayerCount; ++i) {
      enabled_layers[i] = instance.second->mEnabledLayers[i].c_str();
    }
    for (size_t i = 0; i < create_info.menabledExtensionCount; ++i) {
      enabled_extensions[i] = instance.second->mEnabledExtensions[i].c_str();
    }
    create_info.mppEnabledLayerNames =
        create_info.menabledLayerCount > 0 ? (char**)enabled_layers : nullptr;
    create_info.mppEnabledExtensionNames =
        create_info.menabledExtensionCount > 0 ? (char**)enabled_extensions
                                               : nullptr;
    VkInstance i = instance.second->mVulkanHandle;
    RecreateInstance(observer, &create_info, &i);
    recreateDebugInfo(
        this, observer,
        VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_INSTANCE_EXT,
        instance.second);
  }
  {
    for (auto& surface : Surfaces) {
      switch (surface.second->mType) {
        case SurfaceType::SURFACE_TYPE_XCB: {
          VkXcbSurfaceCreateInfoKHR create_info = {
              VkStructureType::VK_STRUCTURE_TYPE_XCB_SURFACE_CREATE_INFO_KHR,
              nullptr, 0, nullptr,
              0};  // We don't actually have to plug this in, our replay
                   // handles this without any arguments just fine.
          RecreateXCBSurfaceKHR(observer, surface.second->mInstance,
                                &create_info, &surface.second->mVulkanHandle);
        } break;
        case SurfaceType::SURFACE_TYPE_ANDROID: {
          VkAndroidSurfaceCreateInfoKHR create_info = {
              VkStructureType::
                  VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR,
              nullptr, 0,
              nullptr};  // We don't actually have to plug this in, our replay
                         // handles this without any arguments just fine.
          RecreateAndroidSurfaceKHR(observer, surface.second->mInstance,
                                    &create_info,
                                    &surface.second->mVulkanHandle);
        } break;
        case SurfaceType::SURFACE_TYPE_WIN32: {
          VkWin32SurfaceCreateInfoKHR create_info = {
              VkStructureType::VK_STRUCTURE_TYPE_WIN32_SURFACE_CREATE_INFO_KHR,
              nullptr, 0, 0,
              0};  // We don't actually have to plug this in, our replay
                   // handles this without any arguments just fine.
          RecreateWin32SurfaceKHR(observer, surface.second->mInstance,
                                  &create_info, &surface.second->mVulkanHandle);
        } break;
        case SurfaceType::SURFACE_TYPE_WAYLAND: {
          VkWaylandSurfaceCreateInfoKHR create_info = {
              VkStructureType::
                  VK_STRUCTURE_TYPE_WAYLAND_SURFACE_CREATE_INFO_KHR,
              nullptr, 0, nullptr,
              nullptr};  // We don't actually have to plug this in, our replay
                         // handles this without any arguments just fine.
          RecreateWaylandSurfaceKHR(observer, surface.second->mInstance,
                                    &create_info,
                                    &surface.second->mVulkanHandle);
        } break;
        case SurfaceType::SURFACE_TYPE_XLIB: {
          VkXlibSurfaceCreateInfoKHR create_info = {
              VkStructureType::VK_STRUCTURE_TYPE_XLIB_SURFACE_CREATE_INFO_KHR,
              nullptr, 0, nullptr,
              0};  // We don't actually have to plug this in, our replay
                   // handles this without any arguments just fine.
          RecreateXlibSurfaceKHR(observer, surface.second->mInstance,
                                 &create_info, &surface.second->mVulkanHandle);
        } break;
        case SurfaceType::SURFACE_TYPE_MIR: {
          VkMirSurfaceCreateInfoKHR create_info = {
              VkStructureType::VK_STRUCTURE_TYPE_MIR_SURFACE_CREATE_INFO_KHR,
              nullptr, 0, nullptr,
              nullptr};  // We don't actually have to plug this in, our replay
                         // handles this without any arguments just fine.
          RecreateMirSurfaceKHR(observer, surface.second->mInstance,
                                &create_info, &surface.second->mVulkanHandle);
        } break;
        default:
          GAPID_FATAL("Unhandled surface type");
      }
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_SURFACE_KHR_EXT,
                        surface.second);
    }
  }
  {
    std::map<VkInstance, std::vector<VkPhysicalDevice>> devices;
    for (auto& physical_device : PhysicalDevices) {
      auto it = devices.find(physical_device.second->mInstance);
      if (it == devices.end()) {
        it = devices
                 .insert(std::make_pair(physical_device.second->mInstance,
                                        std::vector<VkPhysicalDevice>()))
                 .first;
      }
      it->second.push_back(physical_device.second->mVulkanHandle);
    }
    for (auto& instance_devices : devices) {
      // Enumerate the physical devices for one instance
      uint32_t count = instance_devices.second.size();
      // The physical device properties must be in the same order as the
      // physical devices
      std::vector<VkPhysicalDeviceProperties> props_in_order;
      props_in_order.reserve(count);
      for (size_t i = 0; i < count; ++i) {
        props_in_order.push_back(PhysicalDevices[instance_devices.second[i]]
                                     ->mPhysicalDeviceProperties);
      }
      RecreatePhysicalDevices(observer, instance_devices.first, &count,
                              instance_devices.second.data(),
                              props_in_order.data());
    }

    for (auto& physical_device : PhysicalDevices) {
      uint32_t queueFamilyPropertyCount = 0;
      std::vector<VkQueueFamilyProperties> queueFamilyProperties;
      mImports.mVkInstanceFunctions[physical_device.second->mInstance]
          .vkGetPhysicalDeviceQueueFamilyProperties(
              physical_device.second->mVulkanHandle, &queueFamilyPropertyCount,
              nullptr);
      queueFamilyProperties.resize(queueFamilyPropertyCount);
      mImports.mVkInstanceFunctions[physical_device.second->mInstance]
          .vkGetPhysicalDeviceQueueFamilyProperties(
              physical_device.second->mVulkanHandle, &queueFamilyPropertyCount,
              queueFamilyProperties.data());

      VkPhysicalDeviceMemoryProperties memory_properties;
      mImports.mVkInstanceFunctions[physical_device.second->mInstance]
          .vkGetPhysicalDeviceMemoryProperties(
              physical_device.second->mVulkanHandle, &memory_properties);

      RecreatePhysicalDeviceProperties(
          observer, physical_device.second->mVulkanHandle,
          &queueFamilyPropertyCount, queueFamilyProperties.data(),
          &memory_properties);
    }
    for (auto& physical_device : PhysicalDevices) {
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_PHYSICAL_DEVICE_EXT,
                        physical_device.second);
    }
  }
  for (auto& device : Devices) {
    mImports.mVkDeviceFunctions[device.second->mVulkanHandle].vkDeviceWaitIdle(
        device.second->mVulkanHandle);
    VkDeviceCreateInfo create_info = {};
    create_info.msType = VkStructureType::VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO;
    create_info.mflags = 0;
    create_info.menabledLayerCount = device.second->mEnabledLayers.size();
    create_info.menabledExtensionCount =
        device.second->mEnabledExtensions.size();

    const char** enabled_layers =
        (const char**)alloca(sizeof(char*) * create_info.menabledLayerCount);
    const char** enabled_extensions = (const char**)alloca(
        sizeof(char*) * create_info.menabledExtensionCount);

    for (size_t i = 0; i < create_info.menabledLayerCount; ++i) {
      enabled_layers[i] = device.second->mEnabledLayers[i].c_str();
    }
    for (size_t i = 0; i < create_info.menabledExtensionCount; ++i) {
      enabled_extensions[i] = device.second->mEnabledExtensions[i].c_str();
    }
    create_info.mppEnabledLayerNames =
        create_info.menabledLayerCount > 0 ? (char**)enabled_layers : nullptr;
    create_info.mppEnabledExtensionNames =
        create_info.menabledExtensionCount > 0 ? (char**)enabled_extensions
                                               : nullptr;

    std::vector<VkDeviceQueueCreateInfo> queue_create_infos;
    std::vector<std::vector<float>> queue_priorities;
    for (auto queue : device.second->mQueues) {
      uint32_t family_index = queue.second.mQueueFamilyIndex;
      uint32_t queue_index = queue.second.mQueueIndex;
      float queue_priority = queue.second.mPriority;
      auto a =
          std::find_if(queue_create_infos.begin(), queue_create_infos.end(),
                       [&](VkDeviceQueueCreateInfo& a) {
                         return family_index == a.mqueueFamilyIndex;
                       });
      if (a == queue_create_infos.end()) {
        queue_create_infos.push_back({});
        queue_priorities.push_back({});
        a = queue_create_infos.end() - 1;
        a->msType = VkStructureType::VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO;
        a->mpNext = nullptr;
        a->mflags = 0;
        a->mqueueFamilyIndex = family_index;
        a->mqueueCount = 0;
      }
      uint32_t num_queues = queue_index + 1;
      if (a->mqueueCount < num_queues) {
        a->mqueueCount = num_queues;
      }
      queue_priorities[a - queue_create_infos.begin()].resize(num_queues);
      queue_priorities[a - queue_create_infos.begin()][queue_index] =
          queue_priority;
    }

    for (size_t i = 0; i < queue_create_infos.size(); ++i) {
      auto& v = queue_create_infos[i];
      v.mpQueuePriorities = queue_priorities[i].data();
    }
    create_info.mqueueCreateInfoCount = queue_create_infos.size();
    create_info.mpQueueCreateInfos = queue_create_infos.data();
    create_info.mpEnabledFeatures = &device.second->mEnabledFeatures;

    VkDevice d = device.second->mVulkanHandle;
    VkPhysicalDevice pd = device.second->mPhysicalDevice;
    RecreateDevice(observer, pd, &create_info, &d);
    recreateDebugInfo(
        this, observer,
        VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_DEVICE_EXT,
        device.second);
  }
  for (auto& queue : Queues) {
    auto& queue_object = queue.second;
    VkQueue q = queue_object->mVulkanHandle;
    uint32_t family = queue_object->mFamily;
    uint32_t index = queue_object->mIndex;
    VkDevice device = queue_object->mDevice;
    RecreateQueue(observer, device, family, index, &q);
    recreateDebugInfo(
        this, observer,
        VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_QUEUE_EXT,
        queue.second);
  }
  {
    VkSwapchainCreateInfoKHR create_info = {};
    create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR;
    for (auto& swapchain : Swapchains) {
      ImageInfo& info = swapchain.second->mInfo;
      std::vector<uint32_t> queues(info.mQueueFamilyIndices.size());
      create_info.msurface = swapchain.second->mSurface->mVulkanHandle;
      create_info.mimageFormat = info.mFormat;
      create_info.mimageColorSpace = swapchain.second->mColorSpace;
      create_info.mimageExtent.mWidth = info.mExtent.mWidth;
      create_info.mimageExtent.mHeight = info.mExtent.mHeight;
      create_info.mimageArrayLayers = info.mArrayLayers;
      create_info.mimageUsage = info.mUsage;
      create_info.mqueueFamilyIndexCount = info.mQueueFamilyIndices.size();
      create_info.mpreTransform = swapchain.second->mPreTransform;
      create_info.mcompositeAlpha = swapchain.second->mCompositeAlpha;
      create_info.mpresentMode = swapchain.second->mPresentMode;
      create_info.mclipped = swapchain.second->mClipped;
      for (size_t i = 0; i < info.mQueueFamilyIndices.size(); ++i) {
        queues[i] = info.mQueueFamilyIndices[i];
      }
      create_info.mpQueueFamilyIndices = queues.data();
      uint32_t swapchainImages;
      VkDevice device = swapchain.second->mDevice;
      mImports.mVkDeviceFunctions[device].vkGetSwapchainImagesKHR(
          device, swapchain.second->mVulkanHandle, &swapchainImages, nullptr);
      std::vector<VkImage> images(swapchainImages);
      std::vector<uint32_t> imageLayouts(
          swapchainImages, VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED);
      std::vector<VkQueue> lastQueues(swapchainImages, 0);
      mImports.mVkDeviceFunctions[device].vkGetSwapchainImagesKHR(
          device, swapchain.second->mVulkanHandle, &swapchainImages,
          images.data());
      for (size_t i = 0; i < swapchainImages; ++i) {
        auto imageIt = Images.find(images[i]);
        if (imageIt != Images.end()) {
          imageLayouts[i] = imageIt->second->mInfo.mLayout;
          lastQueues[i] = imageIt->second->mLastBoundQueue->mVulkanHandle;
        }
      }
      create_info.mminImageCount = images.size();
      RecreateSwapchain(observer, device, &create_info, images.data(),
                        imageLayouts.data(), lastQueues.data(),
                        &swapchain.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_SWAPCHAIN_KHR_EXT,
                        swapchain.second);
    }
  }
  // Recreate CreateBuffers
  {
    VkBufferCreateInfo buffer_create_info{};
    buffer_create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    for (auto& buffer : Buffers) {
      BufferInfo& info = buffer.second->mInfo;
      std::vector<uint32_t> queues(info.mQueueFamilyIndices.size());
      for (size_t i = 0; i < info.mQueueFamilyIndices.size(); ++i) {
        queues[i] = info.mQueueFamilyIndices[i];
      }
      buffer_create_info.mflags = info.mCreateFlags;
      buffer_create_info.msize = info.mSize;
      buffer_create_info.musage = info.mUsage;
      buffer_create_info.msharingMode = info.mSharingMode;
      buffer_create_info.mqueueFamilyIndexCount = queues.size();
      buffer_create_info.mpQueueFamilyIndices = queues.data();

      // Empty NV dedicated allocation struct
      VkDedicatedAllocationBufferCreateInfoNV dedicated_allocation_create_info{
          VkStructureType::
              VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_BUFFER_CREATE_INFO_NV,  // sType
          nullptr,          // pNext
          VkBool32(false),  // dedicatedAllocation
      };
      // If the buffer is created with Dedicated Allocation NV extension,
      // we need to populate the pNext pointer here.
      if (buffer.second->mInfo.mDedicatedAllocationNV) {
        dedicated_allocation_create_info.mdedicatedAllocation =
            buffer.second->mInfo.mDedicatedAllocationNV->mDedicatedAllocation;
        buffer_create_info.mpNext = &dedicated_allocation_create_info;
      }
      RecreateBuffer(observer, buffer.second->mDevice, &buffer_create_info,
                     &buffer.second->mVulkanHandle, &buffer.second->mMemoryRequirements);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_BUFFER_EXT,
          buffer.second);
    }
  }

  // Recreate CreateImages
  {
    VkImageCreateInfo image_create_info{};
    image_create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO;
    for (auto& image : Images) {
      if (image.second->mIsSwapchainImage) {
        // Don't recreate the swapchain images
        continue;
      }
      ImageInfo& info = image.second->mInfo;
      std::vector<uint32_t> queues(info.mQueueFamilyIndices.size());
      for (size_t i = 0; i < info.mQueueFamilyIndices.size(); ++i) {
        queues[i] = info.mQueueFamilyIndices[i];
      }
      image_create_info.mflags = info.mFlags;
      image_create_info.mimageType = info.mImageType;
      image_create_info.mformat = info.mFormat;
      image_create_info.mextent = info.mExtent;
      image_create_info.mmipLevels = info.mMipLevels;
      image_create_info.marrayLayers = info.mArrayLayers;
      image_create_info.msamples = info.mSamples;
      image_create_info.mtiling = info.mTiling;
      image_create_info.musage = info.mUsage;
      image_create_info.msharingMode = info.mSharingMode;
      image_create_info.mqueueFamilyIndexCount = queues.size();
      image_create_info.mpQueueFamilyIndices = queues.data();
      image_create_info.minitialLayout =
          VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED;

      // Empty NV dedicated allocation struct
      VkDedicatedAllocationImageCreateInfoNV dedicated_allocation_create_info{
          VkStructureType::
              VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_IMAGE_CREATE_INFO_NV,  // sType
          nullptr,          // pNext
          VkBool32(false),  // dedicatedAllocation
      };
      // If the buffer is created with Dedicated Allocation NV extension,
      // we need to populate the pNext pointer here.
      if (info.mDedicatedAllocationNV) {
        dedicated_allocation_create_info.mdedicatedAllocation =
            info.mDedicatedAllocationNV->mDedicatedAllocation;
        image_create_info.mpNext = &dedicated_allocation_create_info;
      }
      std::vector<VkSparseImageMemoryRequirements> sparse_img_mem_reqs;
      for (auto& req : image.second->mSparseMemoryRequirements) {
        sparse_img_mem_reqs.emplace_back(req.second);
      }
      RecreateImage(observer, image.second->mDevice, &image_create_info,
                    &image.second->mVulkanHandle,
                    &image.second->mMemoryRequirements,
                    sparse_img_mem_reqs.size(), sparse_img_mem_reqs.data());
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_IMAGE_EXT,
          image.second);
    }
  }

  // Recreate AllocateMemory
  for (auto& memory : DeviceMemories) {
    // Empty NV dedicated allocation struct
    VkDedicatedAllocationMemoryAllocateInfoNV dedicated_allocation_info{
        VkStructureType::
            VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV,  // sType
        nullptr,      // pNext
        VkImage(0),   // image
        VkBuffer(0),  // buffer
    };

    auto& dm = memory.second;
    VkDeviceMemory mem = memory.second->mVulkanHandle;
    VkMemoryAllocateInfo allocate_info{
        VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, nullptr,
        dm->mAllocationSize, dm->mMemoryTypeIndex};

    if (memory.second->mDedicatedAllocationNV) {
      dedicated_allocation_info.mimage =
          memory.second->mDedicatedAllocationNV->mImage;
      dedicated_allocation_info.mbuffer =
          memory.second->mDedicatedAllocationNV->mBuffer;
      allocate_info.mpNext = &dedicated_allocation_info;
    }

    RecreateDeviceMemory(observer, dm->mDevice, &allocate_info,
                         dm->mMappedOffset, dm->mMappedSize,
                         &dm->mMappedLocation, &mem);
    recreateDebugInfo(this, observer,
                      VkDebugReportObjectTypeEXT::
                          VK_DEBUG_REPORT_OBJECT_TYPE_DEVICE_MEMORY_EXT,
                      memory.second);
  }

  // Bind and Fill the "recreated" buffer memory
  {
    for (auto& bi: Buffers) {
      VkBuffer buf_handle = bi.first;
      auto buf = bi.second;
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

      if (sparseBound || denseBound) {
        // recover buffer memory bindings
        std::vector<VkSparseMemoryBind> sparseBinds;
        if (sparseBound) {
          for (const auto& b: buf->mSparseMemoryBindings) {
            sparseBinds.emplace_back(b.second);
          }
        }
        RecreateBufferMemoryBindings(
            observer, buf->mDevice, buf_handle,
            denseBound ? buf->mMemory->mVulkanHandle : VkDeviceMemory(0),
            denseBound ? buf->mMemoryOffset : VkDeviceSize(0),
            sparseBound ? sparseBinds.size()
                        : 0,
            sparseBound ? sparseBinds.data() : nullptr);

        // recover the buffer data
        VkDevice dev_handle = getObjectCreatingDevice(buf);
        if (Devices.count(dev_handle) == 0) {
          subVkErrorInvalidDevice(observer, [](){}, dev_handle);
        }
        auto& device_functions = mImports.mVkDeviceFunctions[dev_handle];
        VkPhysicalDeviceMemoryProperties mem_props =
            PhysicalDevices[Devices[dev_handle]->mPhysicalDevice]
                ->mMemoryProperties;
        std::shared_ptr<QueueObject> queue = GetQueue(Queues, buf);

        auto recreate_data = [&](VkDeviceSize res_offset, VkDeviceSize size) {
          CopyDataHelper helper(this, device_functions, dev_handle, queue,
                                mem_props);
          uint32_t host_visible_mem_index = 0;
          std::vector<uint8_t> data;
          std::tie(host_visible_mem_index, data) =
              helper.GetBufferData(buf_handle, res_offset, size);
          RecreateBufferData(observer, dev_handle, buf_handle,
                            res_offset, size, host_visible_mem_index,
                            queue->mVulkanHandle, data.data());
        };
        if (sparseResidency) {
          for (const auto& b : sparseBinds) {
            recreate_data(b.mresourceOffset, b.msize);
          }
        } else {
          if (sparseBinding) {
            // If the buffer is not fully bound, do not try to access its data.
            if (!IsFullyBound(0, buf_info.mSize, buf->mSparseMemoryBindings)) {
              continue;
            }
          }
          recreate_data(0, buf_info.mSize);
        }
      }
    }
  }
  {
    VkSamplerCreateInfo sampler_create_info = {};
    sampler_create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO;
    for (auto& sampler : Samplers) {
      auto& samplerObject = *sampler.second;
      sampler_create_info.mmagFilter = samplerObject.mMagFilter;
      sampler_create_info.mminFilter = samplerObject.mMinFilter;
      sampler_create_info.mmipmapMode = samplerObject.mMipMapMode;
      sampler_create_info.maddressModeU = samplerObject.mAddressModeU;
      sampler_create_info.maddressModeV = samplerObject.mAddressModeV;
      sampler_create_info.maddressModeW = samplerObject.mAddressModeW;
      sampler_create_info.mmipLodBias = samplerObject.mMipLodBias;
      sampler_create_info.manisotropyEnable = samplerObject.mAnisotropyEnable;
      sampler_create_info.mmaxAnisotropy = samplerObject.mMaxAnisotropy;
      sampler_create_info.mcompareEnable = samplerObject.mCompareEnable;
      sampler_create_info.mcompareOp = samplerObject.mCompareOp;
      sampler_create_info.mminLod = samplerObject.mMinLod;
      sampler_create_info.mmaxLod = samplerObject.mMaxLod;
      sampler_create_info.mborderColor = samplerObject.mBorderColor;
      sampler_create_info.munnormalizedCoordinates =
          samplerObject.mUnnormalizedCoordinates;

      RecreateSampler(observer, samplerObject.mDevice, &sampler_create_info,
                      &samplerObject.mVulkanHandle);
    }
  }

  // Bind and Fill the "recreated" image memory
  {
    for (auto& ii : Images) {
      VkImage img_handle = ii.first;
      auto img = ii.second;
      if (img->mIsSwapchainImage) {
        // Don't bind and fill swapchain images memory here
        continue;
      }
      const ImageInfo& img_info = ii.second->mInfo;
      bool denseBound = img->mBoundMemory != nullptr;
      bool sparseBound = (img->mOpaqueSparseMemoryBindings.size() >0) ||
        (img->mSparseImageMemoryBindings.size() > 0);
      bool sparseBinding =
          (img_info.mFlags &
           VkImageCreateFlagBits::VK_IMAGE_CREATE_SPARSE_BINDING_BIT) != 0;
      bool sparseResidency =
          sparseBinding &&
          (img_info.mFlags &
           VkImageCreateFlagBits::VK_IMAGE_CREATE_SPARSE_RESIDENCY_BIT) != 0;

      if (denseBound || sparseBound) {
        std::vector<VkSparseMemoryBind> opaqueBinds;
        std::vector<VkSparseImageMemoryBind> imageBinds;
        if (sparseBound) {
          for (const auto& b : img->mOpaqueSparseMemoryBindings) {
            opaqueBinds.emplace_back(b.second);
          }
          for (const auto& aspect_i : img->mSparseImageMemoryBindings) {
            for (const auto& layer_i : aspect_i.second->mLayers) {
              for (const auto& level_i : layer_i.second->mLevels) {
                for (const auto& block_i : level_i.second->mBlocks) {
                  imageBinds.emplace_back(VkSparseImageMemoryBind{
                      VkImageSubresource{
                          VkImageAspectFlags(aspect_i.first), level_i.first,
                          layer_i.first,
                      },
                      block_i.second->mOffset, block_i.second->mExtent,
                      block_i.second->mMemory, block_i.second->mMemoryOffset,
                      block_i.second->mFlags,
                  });
                }
              }
            }
          }
        }
        // recover image memory bindings
        RecreateImageMemoryBindings(
            observer, img->mDevice, img_handle,
            denseBound ? img->mBoundMemory->mVulkanHandle : VkDeviceMemory(0),
            denseBound ? img->mBoundMemoryOffset : VkDeviceSize(0),
            sparseBound ? opaqueBinds.size() : 0,
            sparseBound ? opaqueBinds.data() : nullptr,
            sparseBound && sparseResidency ? imageBinds.size() : 0,
            sparseBound && sparseResidency ? imageBinds.data() : nullptr);

        // recover image data
        if (img_info.mSamples != VkSampleCountFlagBits::VK_SAMPLE_COUNT_1_BIT) {
          // TODO(awoloszyn): Handle multisampled images here.
          //                  Figure out how we are supposed to get the data
          //                  BACK
          //                  into a MS image (shader?)
          continue;
        }
        if (img->mImageAspect !=
            VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT) {
          // TODO(awoloszyn): Handle depth stencil images
          continue;
        }
        if (img_info.mLayout == VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED) {
          // Don't capture images with undefined layout. The resulting data
          // itself will be undefined.
          continue;
        }
        if (img->mIsSwapchainImage) {
          // Don't capture swapchain images
          continue;
        }

        VkDevice dev_handle = getObjectCreatingDevice(img);
        if (Devices.count(dev_handle) == 0) {
          subVkErrorInvalidDevice(observer, []() {}, dev_handle);
        }
        auto& device_functions = mImports.mVkDeviceFunctions[dev_handle];
        VkPhysicalDeviceMemoryProperties mem_props =
            PhysicalDevices[Devices[dev_handle]->mPhysicalDevice]
                ->mMemoryProperties;
        std::shared_ptr<QueueObject> queue = GetQueue(Queues, img);

        auto recreate_sparse_bind_data = [&](VkSparseImageMemoryBind& bind) {
          CopyDataHelper helper(this, device_functions, dev_handle, queue,
                                mem_props);
          uint32_t host_visible_mem_index = 0;
          std::vector<uint8_t> data;
          std::tie(host_visible_mem_index, data) =
              helper.GetImageSparseSubresourceData(img, bind.msubresource,
                                                   bind.moffset, bind.mextent);
          RecreateSparseImageBindData(
              observer, dev_handle, img_handle, img_info.mLayout, &bind,
              host_visible_mem_index, queue->mVulkanHandle, data.size(),
              data.data());
        };

        auto recreate_subrng_data = [&](VkImageSubresourceRange& range,
                                        VkDeviceSize res_offset,
                                        VkDeviceSize data_size) {
          CopyDataHelper helper(this, device_functions, dev_handle, queue,
                                mem_props);
          uint32_t host_visible_mem_index = 0;
          std::vector<uint8_t> data;
          std::tie(host_visible_mem_index, data) =
              helper.GetImageSubrangeData(img, range, data_size);
          RecreateImageSubrangeData(
              observer, dev_handle, img_handle, img_info.mLayout, &range,
              host_visible_mem_index, img->mLastBoundQueue->mVulkanHandle,
              res_offset, data.size(), data.data());
        };

        // Returns true if the metadata is fully bound.
        auto is_metadata_bound = [&]() {
          for (const auto& ri : img->mSparseMemoryRequirements) {
            const auto& prop = ri.second.mformatProperties;
            if (prop.maspectMask ==
                VkImageAspectFlagBits::VK_IMAGE_ASPECT_METADATA_BIT) {
              // All Metadata must be bound before use. And metdata aspect only
              // has a miptail region for all the data.
              if (!IsFullyBound(ri.second.mimageMipTailOffset,
                                ri.second.mimageMipTailSize,
                                img->mOpaqueSparseMemoryBindings)) {
                return false;
              }
            }
          }
          return true;
        };

        // Helper function to recreate miptail region data.
        auto recreate_miptail = [&]() {
          for (const auto& ri : img->mSparseMemoryRequirements) {
            const auto& prop = ri.second.mformatProperties;
            if (prop.maspectMask ==
                VkImageAspectFlagBits::VK_IMAGE_ASPECT_METADATA_BIT) {
              VkImageSubresourceRange rng{
                  VkImageAspectFlagBits::VK_IMAGE_ASPECT_METADATA_BIT,
                  ri.second.mimageMipTailFirstLod,
                  img_info.mMipLevels - ri.second.mimageMipTailFirstLod,
                  0,
                  img_info.mArrayLayers,
              };
              recreate_subrng_data(rng, ri.second.mimageMipTailOffset,
                                   ri.second.mimageMipTailSize);
            } else {
              // Other aspects than Metadata.
              if ((prop.mflags &
                   VkSparseImageFormatFlagBits::
                       VK_SPARSE_IMAGE_FORMAT_SINGLE_MIPTAIL_BIT) != 0) {
                // One miptail region for all layers' miptails.
                if (IsFullyBound(ri.second.mimageMipTailOffset,
                                 ri.second.mimageMipTailSize,
                                 img->mOpaqueSparseMemoryBindings)) {
                  VkImageSubresourceRange rng{
                      // TODO: To add support of depth/stencil images, need to
                      // handle the case that there are multiple bits in
                      // AspectMask here. vkCmdCopyBufferToImage or
                      // ImageToBuffer does not support multiple bits in the
                      // aspectMask.
                      ri.second.mformatProperties.maspectMask,
                      ri.second.mimageMipTailFirstLod,
                      img_info.mMipLevels - ri.second.mimageMipTailFirstLod,
                      0,
                      img_info.mArrayLayers,
                  };
                  recreate_subrng_data(rng, ri.second.mimageMipTailOffset,
                                       ri.second.mimageMipTailSize);
                }
              } else {
                // Each layer has its own miptail region
                for (uint32_t i = 0; i < uint32_t(img_info.mArrayLayers); i++) {
                  VkDeviceSize offset = ri.second.mimageMipTailOffset +
                                        i * ri.second.mimageMipTailStride;
                  if (IsFullyBound(offset, ri.second.mimageMipTailSize,
                        img->mOpaqueSparseMemoryBindings)) {
                    VkImageSubresourceRange rng{
                        ri.second.mformatProperties.maspectMask,
                        ri.second.mimageMipTailFirstLod,
                        img_info.mMipLevels - ri.second.mimageMipTailFirstLod,
                        i,
                        1,
                    };
                    recreate_subrng_data(rng, offset,
                                         ri.second.mimageMipTailSize);
                  }
                }
              }
            }
          }
        };

        if (sparseResidency) {
          // Handle MipTails first, all metadata (mip tail) regions must be
          // bound before the image can be accessed.
          if(!is_metadata_bound()) {
            continue;
          }
          recreate_miptail();
          for (auto& b : imageBinds) {
            recreate_sparse_bind_data(b);
          }
        } else {
          // If the image has opaque sparse bindings, check if all the regions
          // are bound before accessing the image. If it is not fully bound,
          // do not access its data.
          if (sparseBound) {
            if (!IsFullyBound(0, img->mMemoryRequirements.msize,
                              img->mOpaqueSparseMemoryBindings)) {
              continue;
            }
          }
          // The image must have been bound entirely, copy the whole data
          VkImageSubresourceRange rng {
            img->mImageAspect,
            0, img_info.mMipLevels,
            0, img_info.mArrayLayers,
          };
          recreate_subrng_data(rng, 0, img->mMemoryRequirements.msize);
        }
      }
    }


  }

  {
    VkFenceCreateInfo create_info{
        VkStructureType::VK_STRUCTURE_TYPE_FENCE_CREATE_INFO, nullptr, 0};
    for (auto& fence_object : Fences) {
      VkDevice device = fence_object.second->mDevice;
      VkFence fence = fence_object.second->mVulkanHandle;

      uint32_t status =
          mImports.mVkDeviceFunctions[device].vkGetFenceStatus(device, fence);
      if (status == VkResult::VK_SUCCESS) {
        create_info.mflags =
            VkFenceCreateFlagBits::VK_FENCE_CREATE_SIGNALED_BIT;
      } else {
        create_info.mflags = 0;
      }

      RecreateFence(observer, device, &create_info, &fence);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_FENCE_EXT,
          fence_object.second);
    }
  }
  {
    VkSemaphoreCreateInfo create_info{
        VkStructureType::VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO, nullptr, 0};
    for (auto& semaphore : Semaphores) {
      RecreateSemaphore(observer, semaphore.second->mDevice, &create_info,
                        semaphore.second->mSignaled,
                        &semaphore.second->mVulkanHandle);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_SEMAPHORE_EXT,
          semaphore.second);
    }
  }
  {
    VkEventCreateInfo create_info{
        VkStructureType::VK_STRUCTURE_TYPE_EVENT_CREATE_INFO, nullptr, 0};
    for (auto& event : Events) {
      RecreateEvent(observer, event.second->mDevice, &create_info,
                    event.second->mSignaled, &event.second->mVulkanHandle);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_EVENT_EXT,
          event.second);
    }
  }
  {
    VkCommandPoolCreateInfo create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, nullptr, 0,
        0};
    for (auto& commandPool : CommandPools) {
      create_info.mflags = commandPool.second->mFlags;
      create_info.mqueueFamilyIndex = commandPool.second->mQueueFamilyIndex;
      RecreateCommandPool(observer, commandPool.second->mDevice, &create_info,
                          &commandPool.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_COMMAND_POOL_EXT,
                        commandPool.second);
    }
  }

  // Samplers go here
  {
    VkPipelineCacheCreateInfo create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_PIPELINE_CACHE_CREATE_INFO, nullptr,
        0, 0, nullptr};
    for (auto& pipelineCache : PipelineCaches) {
      RecreatePipelineCache(observer, pipelineCache.second->mDevice,
                            &create_info, &pipelineCache.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_CACHE_EXT,
                        pipelineCache.second);
    }
  }

  {
    VkDescriptorSetLayoutCreateInfo create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
        nullptr, 0, 0, nullptr};
    for (auto& descriptorSetLayout : DescriptorSetLayouts) {
      std::vector<VkDescriptorSetLayoutBinding> bindings;
      std::vector<std::vector<VkSampler>> immutableSamplers;
      for (auto& binding : descriptorSetLayout.second->mBindings) {
        bindings.push_back({binding.first, binding.second.mType,
                            binding.second.mCount, binding.second.mStages,
                            nullptr});
        immutableSamplers.push_back({});
        if (binding.second.mImmutableSamplers.size()) {
          for (size_t i = 0; i < binding.second.mImmutableSamplers.size();
               ++i) {
            immutableSamplers.back().push_back(
                binding.second.mImmutableSamplers[i]->mVulkanHandle);
          }
        }
      }

      for (size_t i = 0; i < bindings.size(); ++i) {
        if (!immutableSamplers[i].empty()) {
          bindings[i].mpImmutableSamplers = immutableSamplers[i].data();
        }
      }

      create_info.mbindingCount = bindings.size();
      create_info.mpBindings = bindings.data();
      RecreateDescriptorSetLayout(observer, descriptorSetLayout.second->mDevice,
                                  &create_info,
                                  &descriptorSetLayout.second->mVulkanHandle);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::
              VK_DEBUG_REPORT_OBJECT_TYPE_DESCRIPTOR_SET_LAYOUT_EXT,
          descriptorSetLayout.second);
    }
  }

  {
    VkPipelineLayoutCreateInfo create_info{
        VkStructureType::VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
        nullptr,
        0,
        0,
        nullptr,
        0,
        nullptr};
    for (auto& pipelineLayout : PipelineLayouts) {
      create_info.msetLayoutCount = pipelineLayout.second->mSetLayouts.size();
      create_info.mpushConstantRangeCount =
          pipelineLayout.second->mPushConstantRanges.size();
      std::vector<VkDescriptorSetLayout> layouts;
      for (size_t i = 0; i < create_info.msetLayoutCount; ++i) {
        layouts.push_back(pipelineLayout.second->mSetLayouts[i]->mVulkanHandle);
      }
      std::vector<VkPushConstantRange> ranges;
      for (size_t i = 0; i < create_info.mpushConstantRangeCount; ++i) {
        ranges.push_back(pipelineLayout.second->mPushConstantRanges[i]);
      }
      create_info.mpPushConstantRanges = ranges.data();
      create_info.mpSetLayouts = layouts.data();

      RecreatePipelineLayout(observer, pipelineLayout.second->mDevice,
                             &create_info,
                             &pipelineLayout.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_LAYOUT_EXT,
                        pipelineLayout.second);
    }
  }
  {
    for (auto& rp : RenderPasses) {
      RebuildRenderPass(observer, this, rp.second);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_RENDER_PASS_EXT,
                        rp.second);
    }
  }
  {
    VkShaderModuleCreateInfo create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, nullptr,
        0, 0, nullptr};
    for (auto& shaderModule : ShaderModules) {
      create_info.mcodeSize = static_cast<size_t>(shaderModule.second->mWords.size());
      create_info.mpCode = shaderModule.second->mWords.begin();
      RecreateShaderModule(observer, shaderModule.second->mDevice, &create_info,
                           &shaderModule.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_SHADER_MODULE_EXT,
                        shaderModule.second);
    }
  }

  // Scope for creating and deleting temporary shader modules. Pipelines are
  // allowed to use destroyed shader modules. Such shader module may not be
  // alive when we enumerate Vulkan resources, so we need to create them,
  // use them in the recreated pipelines, then delete them.
  {
    TemporaryShaderModule temporary_shader_modules(observer, this);
    TemporaryRenderPass temporary_render_passes(observer, this);

    for (auto& compute_pipeline : ComputePipelines) {
      auto& pipeline = *compute_pipeline.second;
      VkComputePipelineCreateInfo create_info = {};
      create_info.msType =
          VkStructureType::VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO;
      create_info.mflags = pipeline.mFlags;
      create_info.mlayout = pipeline.mPipelineLayout->mVulkanHandle;
      create_info.mbasePipelineHandle = pipeline.mBasePipeline;

      VkSpecializationInfo specialization_info;
      std::vector<VkSpecializationMapEntry> specialization_entries;
      create_info.mstage.msType =
          VkStructureType::VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
      create_info.mstage.mstage = pipeline.mStage.mStage;
      // Create temporary shader module if the shader module has been
      // destroyed
      if (ShaderModules.find(pipeline.mStage.mModule->mVulkanHandle) ==
          ShaderModules.end()) {
        create_info.mstage.mmodule =
            temporary_shader_modules.CreateShaderModule(
                pipeline.mStage.mModule);
      } else {
        create_info.mstage.mmodule = pipeline.mStage.mModule->mVulkanHandle;
      }

      create_info.mstage.mpName =
          const_cast<char*>(pipeline.mStage.mEntryPoint.c_str());
      if (pipeline.mStage.mSpecialization) {
        specialization_info.mmapEntryCount =
            pipeline.mStage.mSpecialization->mSpecializations.size();
        for (size_t j = 0; j < specialization_info.mmapEntryCount; ++j) {
          specialization_entries.push_back(
              pipeline.mStage.mSpecialization->mSpecializations[j]);
        }
        specialization_info.mpMapEntries = specialization_entries.data();
        specialization_info.mdataSize =
            pipeline.mStage.mSpecialization->mData.size();
        specialization_info.mpData =
            pipeline.mStage.mSpecialization->mData.begin();
        create_info.mstage.mpSpecializationInfo = &specialization_info;
      }
      RecreateComputePipeline(
          observer, pipeline.mDevice,
          pipeline.mPipelineCache ? pipeline.mPipelineCache->mVulkanHandle : 0,
          &create_info, &pipeline.mVulkanHandle);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_EXT,
          compute_pipeline.second);
    }

    std::set<std::string> entrypoint_names;
    auto last_insert = entrypoint_names.begin();

    for (auto& graphics_pipeline : GraphicsPipelines) {
      auto& pipeline = *graphics_pipeline.second;
      VkGraphicsPipelineCreateInfo create_info = {};
      create_info.msType =
          VkStructureType::VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO;

      std::vector<VkPipelineShaderStageCreateInfo> stages;
      std::deque<VkSpecializationInfo> specialization_infos;
      std::deque<std::vector<VkSpecializationMapEntry>> specialization_entries;
      std::vector<VkVertexInputBindingDescription> vertex_binding_descriptions;
      std::vector<VkVertexInputAttributeDescription>
          vertex_attribute_descriptions;
      std::vector<VkViewport> viewports;
      std::vector<VkRect2D> scissors;
      std::vector<uint32_t> sample_mask;
      std::vector<VkPipelineColorBlendAttachmentState>
          color_blend_attachment_states;
      std::vector<uint32_t> dynamic_states;

      VkPipelineVertexInputStateCreateInfo vertex_input_state = {};
      vertex_input_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO;

      VkPipelineInputAssemblyStateCreateInfo input_assembly_state = {};
      input_assembly_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO;

      VkPipelineTessellationStateCreateInfo tessellation_state = {};
      tessellation_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_TESSELLATION_STATE_CREATE_INFO;

      VkPipelineViewportStateCreateInfo viewport_state = {};
      viewport_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO;

      VkPipelineRasterizationStateCreateInfo rasterization_state = {};
      rasterization_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO;

      VkPipelineMultisampleStateCreateInfo multisample_state = {};
      multisample_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO;

      VkPipelineDepthStencilStateCreateInfo depth_stencil_state = {};
      depth_stencil_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO;

      VkPipelineColorBlendStateCreateInfo color_blend_state = {};
      color_blend_state.msType = VkStructureType::
          VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO;

      VkPipelineDynamicStateCreateInfo dynamic_state = {};
      dynamic_state.msType =
          VkStructureType::VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO;

      create_info.mflags =
          pipeline.mFlags &
          ~(VkPipelineCreateFlagBits::VK_PIPELINE_CREATE_DERIVATIVE_BIT);
      create_info.mstageCount = pipeline.mStages.size();
      create_info.mlayout = pipeline.mLayout->mVulkanHandle;
      bool render_pass_exists =
          RenderPasses.find(pipeline.mRenderPass->mVulkanHandle) !=
              RenderPasses.end() ||
          temporary_render_passes.has(pipeline.mRenderPass->mVulkanHandle);
      create_info.mrenderPass =
          render_pass_exists
              ? pipeline.mRenderPass->mVulkanHandle
              : temporary_render_passes.CreateRenderPass(pipeline.mRenderPass);
      create_info.msubpass = pipeline.mSubpass;
      // Turn off derived pipelines in MEC.
      // Dervied pipelines are a performance improvement, but have no semantic
      // impact.
      // TODO(awoloszy): Re-enable derived pipelines, i.e. sort pipeline
      // creation.
      create_info.mbasePipelineHandle = 0;

      for (size_t i = 0; i < pipeline.mStages.size(); ++i) {
        auto& stage = pipeline.mStages[i];
        stages.push_back({});
        stages.back().msType = VkStructureType::
            VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO;
        stages.back().mstage = stage.mStage;
        // Create temporary shader module if the shader module has been
        // destroyed
        if (ShaderModules.find(stage.mModule->mVulkanHandle) ==
            ShaderModules.end()) {
          stages.back().mmodule =
              temporary_shader_modules.CreateShaderModule(stage.mModule);
        } else {
          stages.back().mmodule = stage.mModule->mVulkanHandle;
        }
        // In reality most entry_point names are probably the same,
        // so we can always insert to the same place.
        last_insert =
            entrypoint_names.insert(last_insert, stage.mEntryPoint);
        stages.back().mpName = const_cast<char*>(last_insert->c_str());
        if (stage.mSpecialization) {
          specialization_infos.push_back({});
          specialization_entries.push_back({});
          specialization_infos.back().mmapEntryCount =
              stage.mSpecialization->mSpecializations.size();
          for (size_t j = 0; j < specialization_infos.back().mmapEntryCount;
               ++j) {
            specialization_entries.back().push_back(
                stage.mSpecialization->mSpecializations[j]);
          }
          specialization_infos.back().mpMapEntries =
              specialization_entries.back().data();
          specialization_infos.back().mdataSize =
              stage.mSpecialization->mData.size();
          specialization_infos.back().mpData =
              stage.mSpecialization->mData.begin();
          stages.back().mpSpecializationInfo = &specialization_infos.back();
        }
      }
      create_info.mpStages = stages.data();
      for (size_t i = 0;
           i < pipeline.mVertexInputState.mBindingDescriptions.size(); ++i) {
        vertex_binding_descriptions.push_back(
            pipeline.mVertexInputState.mBindingDescriptions[i]);
      }
      for (size_t i = 0;
           i < pipeline.mVertexInputState.mAttributeDescriptions.size(); ++i) {
        vertex_attribute_descriptions.push_back(
            pipeline.mVertexInputState.mAttributeDescriptions[i]);
      }
      vertex_input_state.mvertexBindingDescriptionCount =
          vertex_binding_descriptions.size();
      vertex_input_state.mpVertexBindingDescriptions =
          vertex_binding_descriptions.data();
      vertex_input_state.mvertexAttributeDescriptionCount =
          vertex_attribute_descriptions.size();
      vertex_input_state.mpVertexAttributeDescriptions =
          vertex_attribute_descriptions.data();
      create_info.mpVertexInputState = &vertex_input_state;

      input_assembly_state.mtopology = pipeline.mInputAssemblyState.mTopology;
      input_assembly_state.mprimitiveRestartEnable =
          pipeline.mInputAssemblyState.mPrimitiveRestartEnable;
      create_info.mpInputAssemblyState = &input_assembly_state;

      if (pipeline.mTessellationState) {
        tessellation_state.mpatchControlPoints =
            pipeline.mTessellationState->mPatchControlPoints;
        create_info.mpTessellationState = &tessellation_state;
      }

      if (pipeline.mViewportState) {
        for (size_t i = 0; i < pipeline.mViewportState->mViewports.size();
             ++i) {
          viewports.push_back(pipeline.mViewportState->mViewports[i]);
        }
        for (size_t i = 0; i < pipeline.mViewportState->mScissors.size(); ++i) {
          scissors.push_back(pipeline.mViewportState->mScissors[i]);
        }
        viewport_state.mviewportCount = viewports.size();
        viewport_state.mpViewports = viewports.data();
        viewport_state.mscissorCount = scissors.size();
        viewport_state.mpScissors = scissors.data();
        create_info.mpViewportState = &viewport_state;
      }

      rasterization_state.mdepthClampEnable =
          pipeline.mRasterizationState.mDepthClampEnable;
      rasterization_state.mrasterizerDiscardEnable =
          pipeline.mRasterizationState.mRasterizerDiscardEnable;
      rasterization_state.mpolygonMode =
          pipeline.mRasterizationState.mPolygonMode;
      rasterization_state.mcullMode = pipeline.mRasterizationState.mCullMode;
      rasterization_state.mfrontFace = pipeline.mRasterizationState.mFrontFace;
      rasterization_state.mdepthBiasEnable =
          pipeline.mRasterizationState.mDepthBiasEnable;
      rasterization_state.mdepthBiasConstantFactor =
          pipeline.mRasterizationState.mDepthBiasConstantFactor;
      rasterization_state.mdepthBiasClamp =
          pipeline.mRasterizationState.mDepthBiasClamp;
      rasterization_state.mdepthBiasSlopeFactor =
          pipeline.mRasterizationState.mDepthBiasSlopeFactor;
      rasterization_state.mlineWidth = pipeline.mRasterizationState.mLineWidth;
      create_info.mpRasterizationState = &rasterization_state;

      if (pipeline.mMultisampleState) {
        multisample_state.mrasterizationSamples =
            pipeline.mMultisampleState->mRasterizationSamples;
        multisample_state.msampleShadingEnable =
            pipeline.mMultisampleState->mSampleShadingEnable;
        multisample_state.mminSampleShading =
            pipeline.mMultisampleState->mMinSampleShading;
        multisample_state.malphaToCoverageEnable =
            pipeline.mMultisampleState->mAlphaToCoverageEnable;
        multisample_state.malphaToOneEnable =
            pipeline.mMultisampleState->mAlphaToOneEnable;
        for (size_t i = 0; i < pipeline.mMultisampleState->mSampleMask.size();
             ++i) {
          sample_mask.push_back(pipeline.mMultisampleState->mSampleMask[i]);
        }
        if (sample_mask.size() > 0) {
          multisample_state.mpSampleMask = sample_mask.data();
        }
        create_info.mpMultisampleState = &multisample_state;
      }

      if (pipeline.mDepthState) {
        depth_stencil_state.mdepthTestEnable =
            pipeline.mDepthState->mDepthTestEnable;
        depth_stencil_state.mdepthWriteEnable =
            pipeline.mDepthState->mDepthWriteEnable;
        depth_stencil_state.mdepthCompareOp =
            pipeline.mDepthState->mDepthCompareOp;
        depth_stencil_state.mdepthBoundsTestEnable =
            pipeline.mDepthState->mDepthBoundsTestEnable;
        depth_stencil_state.mstencilTestEnable =
            pipeline.mDepthState->mStencilTestEnable;
        depth_stencil_state.mfront = pipeline.mDepthState->mFront;
        depth_stencil_state.mback = pipeline.mDepthState->mBack;
        depth_stencil_state.mminDepthBounds =
            pipeline.mDepthState->mMinDepthBounds;
        depth_stencil_state.mmaxDepthBounds =
            pipeline.mDepthState->mMaxDepthBounds;
        create_info.mpDepthStencilState = &depth_stencil_state;
      }

      if (pipeline.mColorBlendState) {
        color_blend_state.mlogicOpEnable =
            pipeline.mColorBlendState->mLogicOpEnable;
        color_blend_state.mlogicOp = pipeline.mColorBlendState->mLogicOp;
        color_blend_state.mblendConstants[0] =
            pipeline.mColorBlendState->mLogicOpEnable;
        color_blend_state.mblendConstants[1] =
            pipeline.mColorBlendState->mLogicOpEnable;
        color_blend_state.mblendConstants[2] =
            pipeline.mColorBlendState->mLogicOpEnable;
        color_blend_state.mblendConstants[3] =
            pipeline.mColorBlendState->mLogicOpEnable;
        for (size_t i = 0; i < pipeline.mColorBlendState->mAttachments.size();
             ++i) {
          color_blend_attachment_states.push_back(
              pipeline.mColorBlendState->mAttachments[i]);
        }
        color_blend_state.mattachmentCount =
            color_blend_attachment_states.size();
        color_blend_state.mpAttachments = color_blend_attachment_states.data();
        create_info.mpColorBlendState = &color_blend_state;
      }

      if (pipeline.mDynamicState) {
        for (size_t i = 0; i < pipeline.mDynamicState->mDynamicStates.size();
             ++i) {
          dynamic_states.push_back(pipeline.mDynamicState->mDynamicStates[i]);
        }
        dynamic_state.mdynamicStateCount = dynamic_states.size();
        dynamic_state.mpDynamicStates = dynamic_states.data();
        create_info.mpDynamicState = &dynamic_state;
      }
      // The pipeline cache is allowed to be VK_NULL_HANDLE.
      RecreateGraphicsPipeline(
          observer, pipeline.mDevice,
          pipeline.mPipelineCache ? pipeline.mPipelineCache->mVulkanHandle : 0,
          &create_info, &pipeline.mVulkanHandle);
      recreateDebugInfo(
          this, observer,
          VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_PIPELINE_EXT,
          graphics_pipeline.second);
    }
  }

  {
    VkImageViewCreateInfo create_info = {};
    create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO;
    for (auto& image_view : ImageViews) {
      create_info.mimage = image_view.second->mImage->mVulkanHandle;
      create_info.mviewType = image_view.second->mType;
      create_info.mformat = image_view.second->mFormat;
      create_info.mcomponents = image_view.second->mComponents;
      create_info.msubresourceRange = image_view.second->mSubresourceRange;

      RecreateImageView(observer, image_view.second->mDevice, &create_info,
                        &image_view.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_IMAGE_VIEW_EXT,
                        image_view.second);
    }
  }
  // Recreate buffer views.
  {
    VkBufferViewCreateInfo create_info = {};
    create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_VIEW_CREATE_INFO;
    for (auto& buffer_view : BufferViews) {
      create_info.mbuffer = buffer_view.second->mBuffer->mVulkanHandle;
      create_info.mformat = buffer_view.second->mFormat;
      create_info.moffset = buffer_view.second->mOffset;
      create_info.mrange = buffer_view.second->mRange;

      RecreateBufferView(observer, buffer_view.second->mDevice, &create_info,
                         &buffer_view.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_BUFFER_VIEW_EXT,
                        buffer_view.second);
    }
  }
  {
    VkDescriptorPoolCreateInfo create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
        nullptr,
        0,
        0,
        0,
        nullptr};
    for (auto& pool : DescriptorPools) {
      create_info.mflags = pool.second->mFlags;
      create_info.mmaxSets = pool.second->mMaxSets;
      create_info.mpoolSizeCount = pool.second->mSizes.size();
      std::vector<VkDescriptorPoolSize> sizes(create_info.mpoolSizeCount);
      for (size_t i = 0; i < create_info.mpoolSizeCount; ++i) {
        sizes[i] = pool.second->mSizes[i];
      }
      create_info.mpPoolSizes = sizes.data();
      RecreateDescriptorPool(observer, pool.second->mDevice, &create_info,
                             &pool.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_DESCRIPTOR_POOL_EXT,
                        pool.second);
    }
  }
  {
    VkDescriptorSetAllocateInfo allocate_info = {};
    allocate_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO;
    allocate_info.mdescriptorSetCount = 1;
    for (auto& descriptorSet : DescriptorSets) {
      allocate_info.mdescriptorPool = descriptorSet.second->mDescriptorPool;
      allocate_info.mpSetLayouts =
          &descriptorSet.second->mLayout->mVulkanHandle;

      std::deque<VkDescriptorImageInfo> image_infos;
      std::deque<VkDescriptorBufferInfo> buffer_infos;
      std::deque<VkBufferView> buffer_views;
      std::vector<VkWriteDescriptorSet> writes;
      for (size_t i = 0; i < descriptorSet.second->mBindings.size(); ++i) {
        auto& binding = descriptorSet.second->mBindings[i];
        VkWriteDescriptorSet write_template = {};

        write_template.msType =
            VkStructureType::VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
        write_template.mdstSet = descriptorSet.second->mVulkanHandle;
        write_template.mdstBinding = i;
        write_template.mdescriptorCount = 1;
        write_template.mdescriptorType = binding.mBindingType;

        switch (binding.mBindingType) {
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_SAMPLER:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
            for (size_t j = 0; j < binding.mImageBinding.size(); ++j) {
              if (!binding.mImageBinding[j]->mSampler &&
                  !binding.mImageBinding[j]->mImageView) {
                continue;
              }
              if (binding.mBindingType ==
                  VkDescriptorType::VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER) {
                // If this a combined image/sampler, then we have to make sure
                // that
                // both are valid.
                if (!binding.mImageBinding[j]->mSampler ||
                    !binding.mImageBinding[j]->mImageView) {
                  continue;
                }
              }
              if (binding.mImageBinding[j]->mSampler &&
                  Samplers.find(binding.mImageBinding[j]->mSampler) ==
                      Samplers.end()) {
                continue;
              }
              if (binding.mImageBinding[j]->mImageView &&
                  ImageViews.find(binding.mImageBinding[j]->mImageView) ==
                      ImageViews.end()) {
                continue;
              }
              image_infos.push_back(*binding.mImageBinding[j]);
              VkWriteDescriptorSet write = write_template;
              write.mdstArrayElement = j;
              write.mpImageInfo = &image_infos.back();
              writes.push_back(write);
            }
            break;
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:

            for (size_t j = 0; j < binding.mBufferViewBindings.size(); ++j) {
              if (!binding.mBufferViewBindings[j] ||
                  BufferViews.find(binding.mBufferViewBindings[j]) ==
                      BufferViews.end()) {
                continue;
              }
              buffer_views.push_back(binding.mBufferViewBindings[j]);
              VkWriteDescriptorSet write = write_template;
              write.mdstArrayElement = j;
              write.mpTexelBufferView = &buffer_views.back();
              writes.push_back(write);
            }
            break;
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
          case VkDescriptorType::VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
            for (size_t j = 0; j < binding.mBufferBinding.size(); ++j) {
              if (!binding.mBufferBinding[j]->mBuffer ||
                  Buffers.find(binding.mBufferBinding[j]->mBuffer) ==
                      Buffers.end()) {
                continue;
              }
              buffer_infos.push_back(*binding.mBufferBinding[j]);
              VkWriteDescriptorSet write = write_template;
              write.mdstArrayElement = j;
              write.mpBufferInfo = &buffer_infos.back();
              writes.push_back(write);
            }
            break;
        }
      }
      RecreateDescriptorSet(observer, descriptorSet.second->mDevice,
                            &allocate_info, writes.size(), writes.data(),
                            &descriptorSet.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_DESCRIPTOR_SET_EXT,
                        descriptorSet.second);
    }
  }
  {
    VkFramebufferCreateInfo create_info = {};
    create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO;
    for (auto& framebuffer : Framebuffers) {
      std::vector<VkImageView> imageAttachments(
          framebuffer.second->mImageAttachments.size());
      for (size_t i = 0; i < imageAttachments.size(); ++i) {
        imageAttachments[i] =
            framebuffer.second->mImageAttachments[i]->mVulkanHandle;
      }
      create_info.mrenderPass = framebuffer.second->mRenderPass->mVulkanHandle;
      create_info.mattachmentCount =
          framebuffer.second->mImageAttachments.size();
      create_info.mpAttachments = imageAttachments.data();
      create_info.mwidth = framebuffer.second->mWidth;
      create_info.mheight = framebuffer.second->mHeight;
      create_info.mlayers = framebuffer.second->mLayers;

      RecreateFramebuffer(observer, framebuffer.second->mDevice, &create_info,
                          &framebuffer.second->mVulkanHandle);
      recreateDebugInfo(this, observer,
                        VkDebugReportObjectTypeEXT::
                            VK_DEBUG_REPORT_OBJECT_TYPE_FRAMEBUFFER_EXT,
                        framebuffer.second);
    }
  }

  for (auto& queryPool : QueryPools) {
    auto& pool = *queryPool.second;
    VkQueryPoolCreateInfo create_info = {
        VkStructureType::VK_STRUCTURE_TYPE_QUERY_POOL_CREATE_INFO,
        nullptr,
        0,
        pool.mQueryType,
        pool.mQueryCount,
        pool.mPipelineStatistics};

    std::vector<uint32_t> queries(pool.mQueryCount);
    for (size_t i = 0; i < pool.mStatus.size(); ++i) {
      queries[i] = pool.mStatus[i];
    }
    RecreateQueryPool(observer, pool.mDevice, &create_info, queries.data(),
                      &pool.mVulkanHandle);
    recreateDebugInfo(
        this, observer,
        VkDebugReportObjectTypeEXT::VK_DEBUG_REPORT_OBJECT_TYPE_QUERY_POOL_EXT,
        queryPool.second);
  }

  // Helper function to recreate and begin a given command buffer object.
  auto recreate_and_begin_cmd_buf = [this](
      CallObserver* observer, std::shared_ptr<CommandBufferObject> cmdBuff) {
    VkCommandBufferAllocateInfo allocate_info{
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
        nullptr, cmdBuff->mPool, cmdBuff->mLevel, 1};
    VkCommandBufferBeginInfo begin_info{
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, nullptr,
        0, nullptr};
    VkCommandBufferInheritanceInfo inheritance_info{
        VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_INHERITANCE_INFO,
        nullptr,
        0,
        0,
        0,
        0,
        0,
        0};
    if (cmdBuff->mBeginInfo) {
      begin_info.mflags = cmdBuff->mBeginInfo->mFlags;
      if (cmdBuff->mBeginInfo->mInherited) {
        inheritance_info.mrenderPass =
            cmdBuff->mBeginInfo->mInheritedRenderPass;
        inheritance_info.msubpass = cmdBuff->mBeginInfo->mInheritedSubpass;
        inheritance_info.mframebuffer =
            cmdBuff->mBeginInfo->mInheritedFramebuffer;
        inheritance_info.mocclusionQueryEnable =
            cmdBuff->mBeginInfo->mInheritedOcclusionQuery;
        inheritance_info.mqueryFlags =
            cmdBuff->mBeginInfo->mInheritedQueryFlags;
        inheritance_info.mpipelineStatistics =
            cmdBuff->mBeginInfo->mInheritedPipelineStatsFlags;
        begin_info.mpInheritanceInfo = &inheritance_info;
      }
    }
    RecreateAndBeginCommandBuffer(observer, cmdBuff->mDevice, &allocate_info,
                                  cmdBuff->mBeginInfo ? &begin_info : nullptr,
                                  &cmdBuff->mVulkanHandle);
  };

  // Helper function to fill and end a given command buffer object.
  auto fill_and_end_cmd_buf = [this](
      CallObserver* observer, std::shared_ptr<CommandBufferObject> cmdBuff) {
    // We have to reset the state of this command buffer after we record,
    // since we might be modifying it.
    bool failure = false;
    for (uint32_t i = 0;
        i < static_cast<uint32_t>(cmdBuff->mCommandReferences.size()); ++i) {
        auto& ref = cmdBuff->mCommandReferences[i];
        if (!RecreateCommand(observer, cmdBuff->mVulkanHandle,
                                this, ref)) {
            failure = true;
            break;
        }
    }
    if (cmdBuff->mRecording == RecordingState::COMPLETED && !failure) {
      RecreateEndCommandBuffer(observer, cmdBuff->mVulkanHandle);
    }
  };

  // Recreate and begin all the secondary command buffers
  for (auto& commandBuffer : CommandBuffers) {
    auto cmdBuff = commandBuffer.second;
    if (cmdBuff->mLevel ==
        VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_SECONDARY) {
      recreate_and_begin_cmd_buf(observer, cmdBuff);
    }
  }

  // Re-record commands and end for all the secondary command buffers
  for (auto& commandBuffer : CommandBuffers) {
    auto cmdBuff = commandBuffer.second;
    if (cmdBuff->mLevel ==
        VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_SECONDARY) {
      fill_and_end_cmd_buf(observer, cmdBuff);
    }
  }

  // Recreate and begin primary command buffers
  for (auto& commandBuffer : CommandBuffers) {
    auto cmdBuff = commandBuffer.second;
    if (cmdBuff->mLevel ==
        VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY) {
      recreate_and_begin_cmd_buf(observer, cmdBuff);
    }
  }

  // Re-record commands and end for all the primary command buffers
  for (auto& commandBuffer : CommandBuffers) {
    auto cmdBuff = commandBuffer.second;
    if (cmdBuff->mLevel ==
        VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY) {
      fill_and_end_cmd_buf(observer, cmdBuff);
    }
  }

  for (auto& commandBuffer : CommandBuffers) {
    recreateDebugInfo(this, observer,
                      VkDebugReportObjectTypeEXT::
                          VK_DEBUG_REPORT_OBJECT_TYPE_COMMAND_BUFFER_EXT,
                      commandBuffer.second);
  }
}
}
