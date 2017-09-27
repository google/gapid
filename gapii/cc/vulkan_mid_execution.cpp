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

#include <deque>
#include <map>
#include <vector>
#include <unordered_set>
#include "gapii/cc/vulkan_spy.h"
#include "gapis/api/vulkan/vulkan_pb/api.pb.h"

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
}  // anonymous namespace


struct VulkanStateSerializer: public ReferenceSerializer {
    VulkanStateSerializer(CallObserver* observer): mObserver(observer) {}
    ~VulkanStateSerializer() {
        GAPID_ASSERT(mSerilizationFunctions.size() == 0);
    }
    void finalize() override {
        while(!mSerilizationFunctions.empty()) {
            auto ref = new memory_pb::Reference();
            ref->set_identifier(reinterpret_cast<uint64_t>(mSerilizationFunctions[0].first));
            mObserver->enterAndDelete(ref);
            mObserver->encodeAndDelete(mSerilizationFunctions[0].second());
            mObserver->exit();
            mSerilizationFunctions.pop_front();
        }
    }

    uint64_t process_reference(void* addr, const std::function<::google::protobuf::Message*()>& f) override {
        if (addr == nullptr) {
            return 0;
        }
        bool inserted = false;
        if (mSeenAddresses.count(addr) != 0) {
            return reinterpret_cast<uint64_t>(addr);
        }
        mSeenAddresses.insert(addr);
        mSerilizationFunctions.push_back(std::make_pair(addr, f));
        return reinterpret_cast<uint64_t>(addr);
    }
    uint64_t process_slice(const Pool* pool, void* root, size_t bytes) override {
        auto virtual_pool = mVirtualPools.find(pool);
        if (virtual_pool != mVirtualPools.end()) {
            return virtual_pool->second;
        }
        uint32_t nextPool = mLastSeenPool++;
        uint32_t setPool = mLastSeenPool;
        if (pool == nullptr ) { // Application pool
            nextPool = mLastSeenPool;
            setPool = 0;
        }

        auto static_pool = mRealPoolObservations.insert(std::pair<const Pool*,RealPoolData>(pool, RealPoolData{setPool}));
        if (static_pool.second) {
            mLastSeenPool = nextPool;
        }
        static_pool.first->second.regions.push_back(RealPoolData::MemoryRegion{root, bytes});
        return 0;
    }

    CallObserver* mObserver;
    std::set<void*> mSeenAddresses;
    std::deque<std::pair<void*, std::function<::google::protobuf::Message*()>>> mSerilizationFunctions;

    std::unordered_map<const Pool*, uint64_t> mVirtualPools;
    struct RealPoolData {
        size_t poolIndex;
        struct MemoryRegion {
            void* observationBase;
            size_t size;
        };
        std::vector<MemoryRegion> regions;
    };
    std::unordered_map<const Pool*, RealPoolData> mRealPoolObservations;
    uint64_t mLastSeenPool = 1;
};



void VulkanSpy::CaptureState(CallObserver* observer) {
    // TODO: set up our virtual pools.
    VulkanStateSerializer serializer(observer);

    std::deque<std::shared_ptr<Pool>> virtualPools;
    for (auto& deviceMemory: DeviceMemories) {
        DeviceMemoryObject* obj = deviceMemory.second.get();
        std::shared_ptr<Pool> pool = Pool::create(1); // We need to have at least one byte in the pool
        obj->mData = Slice<uint8_t>(reinterpret_cast<uint8_t*>(pool->base()), obj->mAllocationSize, pool);
        virtualPools.push_back(pool);
        serializer.mVirtualPools[pool.get()] = serializer.mLastSeenPool++;
    }
    for (auto& image: Images) {
        auto& info = image.second->mInfo;
        for (auto& layer: image.second->mLayers) {
            for (auto& level: layer.second->mLevels) {
                auto obj = level.second.get();
                const uint32_t i = level.first;
                auto elementAndTexelBlockSize = subGetElementAndTexelBlockSize(nullptr, nullptr, info.mFormat);
                uint32_t width = subGetMipSize(nullptr, nullptr, info.mExtent.mWidth, i);
                uint32_t height = subGetMipSize(nullptr, nullptr, info.mExtent.mHeight, i);
                uint32_t depth = subGetMipSize(nullptr, nullptr, info.mExtent.mDepth, i);
                uint32_t widthInBlocks = subRoundUpTo(nullptr, nullptr, width, elementAndTexelBlockSize.mTexelBlockSize.mWidth);
                uint32_t heightInBlocks = subRoundUpTo(nullptr, nullptr, height, elementAndTexelBlockSize.mTexelBlockSize.mHeight);
                size_t size = widthInBlocks * heightInBlocks * depth * elementAndTexelBlockSize.mElementSize;

                std::shared_ptr<Pool> pool = Pool::create(1); // We need to have at least one byte in the pool
                obj->mData = Slice<uint8_t>(reinterpret_cast<uint8_t*>(pool->base()), size, pool);
                virtualPools.push_back(pool);
                serializer.mVirtualPools[pool.get()] = serializer.mLastSeenPool++;
            }
        }
    }

    observer->encodeAndDelete(VulkanSpy::serializeState(observer, &serializer));

    for (auto& pool: serializer.mRealPoolObservations) {
        const char* observation_base =
            pool.first ? reinterpret_cast<const char*>(pool.first->base()) : nullptr;
        uint32_t pool_idx = pool.second.poolIndex;

        for (auto& region: pool.second.regions) {
            auto memory_write = new memory_pb::MemoryWrite();
            const char* offset = reinterpret_cast<const char*>(region.observationBase);
            size_t offset_base = offset - observation_base;
            memory_write->set_poolid(pool_idx);
            memory_write->set_offset(offset_base);
            memory_write->set_size(region.size);
            memory_write->mutable_data()->assign(offset, region.size);
            observer->encodeAndDelete(memory_write);
        }
    }

    VkBufferCreateInfo create_info = {};
    create_info.msType = VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    for (auto& buffer : Buffers) {
        if (!buffer.second->mMemory) continue;
        BufferInfo& info = buffer.second->mInfo;

        std::shared_ptr<QueueObject> submit_queue;
        if (buffer.second->mLastBoundQueue) {
          submit_queue = buffer.second->mLastBoundQueue;
        } else {
          for (auto& queue : Queues) {
            if (queue.second->mDevice == buffer.second->mDevice) {
              submit_queue = queue.second;
              break;
            }
          }
        }

        void* data = nullptr;
        size_t data_size = 0;
        uint32_t host_buffer_memory_index = 0;

        VkDevice device = buffer.second->mDevice;
        std::shared_ptr<DeviceObject> device_object =
            Devices[buffer.second->mDevice];
        VkPhysicalDevice& physical_device = device_object->mPhysicalDevice;
        auto& device_functions =
            mImports.mVkDeviceFunctions[buffer.second->mDevice];
        VkInstance& instance = PhysicalDevices[physical_device]->mInstance;

        VkBuffer copy_buffer;
        VkDeviceMemory copy_memory;


        VkPhysicalDeviceMemoryProperties properties;
        mImports.mVkInstanceFunctions[instance]
            .vkGetPhysicalDeviceMemoryProperties(physical_device, &properties);
        create_info.msize = info.mSize;
        create_info.musage =
            VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT;
        create_info.msharingMode = VkSharingMode::VK_SHARING_MODE_EXCLUSIVE;
        device_functions.vkCreateBuffer(device, &create_info, nullptr,
                                        &copy_buffer);
        VkMemoryRequirements buffer_memory_requirements;
        device_functions.vkGetBufferMemoryRequirements(
            device, copy_buffer, &buffer_memory_requirements);
        uint32_t index = 0;
        uint32_t backup_index = 0xFFFFFFFF;

        while (buffer_memory_requirements.mmemoryTypeBits) {
        if (buffer_memory_requirements.mmemoryTypeBits & 0x1) {
            if (properties.mmemoryTypes[index].mpropertyFlags &
                VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
            if (backup_index == 0xFFFFFFFF) {
                backup_index = index;
            }
            if (0 == (properties.mmemoryTypes[index].mpropertyFlags &
                        VkMemoryPropertyFlagBits::
                            VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)) {
                break;
            }
            }
        }
        buffer_memory_requirements.mmemoryTypeBits >>= 1;
        index++;
        }

        // If we could not find a non-coherent memory, then use
        // the only one we found.
        if (buffer_memory_requirements.mmemoryTypeBits != 0) {
        host_buffer_memory_index = index;
        } else {
        host_buffer_memory_index = backup_index;
        }

        VkMemoryAllocateInfo create_copy_memory{
            VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, nullptr,
            info.mSize, host_buffer_memory_index};

        device_functions.vkAllocateMemory(device, &create_copy_memory, nullptr,
                                        &copy_memory);

        device_functions.vkBindBufferMemory(device, copy_buffer, copy_memory,
                                            0);

        VkCommandPool pool;
        VkCommandPoolCreateInfo command_pool_create{
            VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
            nullptr,
            VkCommandPoolCreateFlagBits::VK_COMMAND_POOL_CREATE_TRANSIENT_BIT,
            submit_queue->mFamily};
        device_functions.vkCreateCommandPool(device, &command_pool_create,
                                            nullptr, &pool);

        VkCommandBuffer copy_commands;
        VkCommandBufferAllocateInfo copy_command_create_info{
            VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
            nullptr, pool,
            VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY, 1};
        device_functions.vkAllocateCommandBuffers(
            device, &copy_command_create_info, &copy_commands);

        VkCommandBufferBeginInfo begin_info = {
            VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
            nullptr, VkCommandBufferUsageFlagBits::
                        VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
            nullptr};

        device_functions.vkBeginCommandBuffer(copy_commands, &begin_info);

        VkBufferCopy region{0, 0, info.mSize};

        device_functions.vkCmdCopyBuffer(copy_commands,
                                        buffer.second->mVulkanHandle,
                                        copy_buffer, 1, &region);

        VkBufferMemoryBarrier buffer_barrier = {
            VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
            nullptr,
            VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
            VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
            0xFFFFFFFF,
            0xFFFFFFFF,
            copy_buffer,
            0,
            info.mSize};

        device_functions.vkCmdPipelineBarrier(
            copy_commands,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT, 0, 0, nullptr,
            1, &buffer_barrier, 0, nullptr);

        device_functions.vkEndCommandBuffer(copy_commands);

        VkSubmitInfo submit_info = {
            VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO,
            nullptr,
            0,
            nullptr,
            nullptr,
            1,
            &copy_commands,
            0,
            nullptr};
        device_functions.vkQueueSubmit(submit_queue->mVulkanHandle, 1,
                                        &submit_info, 0);

        device_functions.vkQueueWaitIdle(submit_queue->mVulkanHandle);
        device_functions.vkMapMemory(device, copy_memory, 0, info.mSize, 0,
                                    &data);
        VkMappedMemoryRange range{
            VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, nullptr,
            copy_memory, 0, info.mSize};

        device_functions.vkInvalidateMappedMemoryRanges(device, 1, &range);

        device_functions.vkDestroyCommandPool(device, pool, nullptr);

        DeviceMemoryObject* mem = buffer.second->mMemory.get();
        const Slice<uint8_t>& contents = mem->mData;
        const Pool* p = contents.getPool();

        GAPID_ASSERT(serializer.mVirtualPools.count(p) == 1);
        uint32_t pool_idx = serializer.mVirtualPools[p];

        auto memory_write = new memory_pb::MemoryWrite();
        memory_write->set_poolid(pool_idx);
        memory_write->set_offset(buffer.second->mMemoryOffset);
        memory_write->set_size(buffer.second->mInfo.mSize);
        memory_write->mutable_data()->assign(reinterpret_cast<const char*>(data),
            buffer.second->mInfo.mSize);
        observer->encodeAndDelete(memory_write);

        device_functions.vkDestroyBuffer(device, copy_buffer, nullptr);
        device_functions.vkFreeMemory(device, copy_memory, nullptr);
    }

    VkBufferCreateInfo buffer_create_info = {};
    buffer_create_info.msType =
        VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;
    buffer_create_info.msType = VkStructureType::VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO;

    for (auto& image : Images) {
      if (image.second->mIsSwapchainImage) {
        // Don't bind and fill swapchain images memory here
        continue;
      }

      ImageInfo& info = image.second->mInfo;
      VkQueue lastQueue = 0;
      uint32_t lastQueueFamily = 0;
      if (image.second->mLastBoundQueue) {
        lastQueue = image.second->mLastBoundQueue->mVulkanHandle;
        lastQueueFamily = image.second->mLastBoundQueue->mFamily;
      }

      void* data = nullptr;
      uint64_t data_size = 0;
      uint32_t host_buffer_memory_index = 0;
      bool need_to_clean_up_temps = false;

      VkDevice device = image.second->mDevice;
      std::shared_ptr<DeviceObject> device_object =
          Devices[image.second->mDevice];
      VkPhysicalDevice& physical_device = device_object->mPhysicalDevice;
      auto& device_functions =
          mImports.mVkDeviceFunctions[image.second->mDevice];
      VkInstance& instance = PhysicalDevices[physical_device]->mInstance;

      VkBuffer copy_buffer;
      VkDeviceMemory copy_memory = 0;

      uint32_t imageLayout = info.mLayout;

      if (image.second->mBoundMemory &&
          info.mSamples == VkSampleCountFlagBits::VK_SAMPLE_COUNT_1_BIT &&
          // Don't capture images with undefined layout. The resulting data
          // itself will be undefined.
          imageLayout != VkImageLayout::VK_IMAGE_LAYOUT_UNDEFINED &&
          !image.second->mIsSwapchainImage &&
          image.second->mImageAspect ==
              VkImageAspectFlagBits::VK_IMAGE_ASPECT_COLOR_BIT) {
        // TODO(awoloszyn): Handle multisampled images here.
        //                  Figure out how we are supposed to get the data BACK
        //                  into a MS image (shader?)
        // TODO(awoloszyn): Handle depth stencil images
        data_size = subInferImageSize(nullptr, nullptr, image.second);

        need_to_clean_up_temps = true;

        VkPhysicalDeviceMemoryProperties properties;
        mImports.mVkInstanceFunctions[instance]
            .vkGetPhysicalDeviceMemoryProperties(physical_device, &properties);
        buffer_create_info.msize = data_size;
        buffer_create_info.musage =
            VkBufferUsageFlagBits::VK_BUFFER_USAGE_TRANSFER_DST_BIT;
        buffer_create_info.msharingMode = VkSharingMode::VK_SHARING_MODE_EXCLUSIVE;
        device_functions.vkCreateBuffer(device, &buffer_create_info, nullptr,
                                        &copy_buffer);
        VkMemoryRequirements buffer_memory_requirements;

        device_functions.vkGetBufferMemoryRequirements(
            device, copy_buffer, &buffer_memory_requirements);
        uint32_t index = 0;
        uint32_t backup_index = 0xFFFFFFFF;

        while (buffer_memory_requirements.mmemoryTypeBits) {
          if (buffer_memory_requirements.mmemoryTypeBits & 0x1) {
            if (properties.mmemoryTypes[index].mpropertyFlags &
                VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
              if (backup_index == 0xFFFFFFFF) {
                backup_index = index;
              }
              if (0 == (properties.mmemoryTypes[index].mpropertyFlags &
                        VkMemoryPropertyFlagBits::
                            VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)) {
                break;
              }
            }
          }
          buffer_memory_requirements.mmemoryTypeBits >>= 1;
          index++;
        }

        // If we could not find a non-coherent memory, then use
        // the only one we found.
        if (buffer_memory_requirements.mmemoryTypeBits != 0) {
          host_buffer_memory_index = index;
        } else {
          host_buffer_memory_index = backup_index;
        }

        VkMemoryAllocateInfo create_copy_memory{
            VkStructureType::VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, nullptr,
            buffer_memory_requirements.msize, host_buffer_memory_index};

        uint32_t res = device_functions.vkAllocateMemory(
            device, &create_copy_memory, nullptr, &copy_memory);

        device_functions.vkBindBufferMemory(device, copy_buffer, copy_memory,
                                            0);

        VkCommandPool pool;
        VkCommandPoolCreateInfo command_pool_create{
            VkStructureType::VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
            nullptr,
            VkCommandPoolCreateFlagBits::VK_COMMAND_POOL_CREATE_TRANSIENT_BIT,
            lastQueueFamily};
        res = device_functions.vkCreateCommandPool(device, &command_pool_create,
                                                   nullptr, &pool);

        VkCommandBuffer copy_commands;
        VkCommandBufferAllocateInfo copy_command_create_info{
            VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
            nullptr, pool,
            VkCommandBufferLevel::VK_COMMAND_BUFFER_LEVEL_PRIMARY, 1};

        res = device_functions.vkAllocateCommandBuffers(
            device, &copy_command_create_info, &copy_commands);
        VkCommandBufferBeginInfo begin_info = {
            VkStructureType::VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
            nullptr, VkCommandBufferUsageFlagBits::
                         VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
            nullptr};

        device_functions.vkBeginCommandBuffer(copy_commands, &begin_info);

        std::vector<VkBufferImageCopy> image_copies(info.mMipLevels);
        size_t buffer_offset = 0;
        for (size_t i = 0; i < info.mMipLevels; ++i) {
          const size_t width =
              subGetMipSize(nullptr, nullptr, info.mExtent.mWidth, i);
          const size_t height =
              subGetMipSize(nullptr, nullptr, info.mExtent.mHeight, i);
          const size_t depth =
              subGetMipSize(nullptr, nullptr, info.mExtent.mDepth, i);
          image_copies[i] = {
              static_cast<uint64_t>(buffer_offset),
              0,  // bufferRowLength << tightly packed
              0,  // bufferImageHeight << tightly packed
              {
                  image.second->mImageAspect, static_cast<uint32_t>(i), 0,
                  info.mArrayLayers,
              },  /// subresource
              {0, 0, 0},
              {static_cast<uint32_t>(width), static_cast<uint32_t>(height),
               static_cast<uint32_t>(depth)}};

          buffer_offset +=
              subInferImageLevelSize(nullptr, nullptr, image.second, i);
        }

        VkImageMemoryBarrier memory_barrier = {
            VkStructureType::VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
            nullptr,
            (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1,
            VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT,
            imageLayout,
            VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
            0xFFFFFFFF,
            0xFFFFFFFF,
            image.second->mVulkanHandle,
            {image.second->mImageAspect, 0, info.mMipLevels, 0,
             info.mArrayLayers}};

        device_functions.vkCmdPipelineBarrier(
            copy_commands,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT, 0, 0, nullptr,
            0, nullptr, 1, &memory_barrier);

        device_functions.vkCmdCopyImageToBuffer(
            copy_commands, image.second->mVulkanHandle,
            VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, copy_buffer,
            image_copies.size(), image_copies.data());

        memory_barrier.msrcAccessMask =
            VkAccessFlagBits::VK_ACCESS_TRANSFER_READ_BIT;
        memory_barrier.mdstAccessMask =
            (VkAccessFlagBits::VK_ACCESS_MEMORY_WRITE_BIT << 1) - 1;
        memory_barrier.moldLayout =
            VkImageLayout::VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
        memory_barrier.mnewLayout = imageLayout;

        VkBufferMemoryBarrier buffer_barrier = {
            VkStructureType::VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
            nullptr,
            VkAccessFlagBits::VK_ACCESS_TRANSFER_WRITE_BIT,
            VkAccessFlagBits::VK_ACCESS_HOST_READ_BIT,
            0xFFFFFFFF,
            0xFFFFFFFF,
            copy_buffer,
            0,
            data_size};

        device_functions.vkCmdPipelineBarrier(
            copy_commands,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_TRANSFER_BIT,
            VkPipelineStageFlagBits::VK_PIPELINE_STAGE_HOST_BIT, 0, 0, nullptr,
            1, &buffer_barrier, 1, &memory_barrier);

        device_functions.vkEndCommandBuffer(copy_commands);

        VkSubmitInfo submit_info = {
            VkStructureType::VK_STRUCTURE_TYPE_SUBMIT_INFO,
            nullptr,
            0,
            nullptr,
            nullptr,
            1,
            &copy_commands,
            0,
            nullptr};

        res = device_functions.vkQueueSubmit(lastQueue, 1, &submit_info, 0);
        res = device_functions.vkQueueWaitIdle(lastQueue);
        device_functions.vkMapMemory(device, copy_memory, 0, data_size, 0,
                                     &data);
        VkMappedMemoryRange range{
            VkStructureType::VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, nullptr,
            copy_memory, 0, data_size};

        device_functions.vkInvalidateMappedMemoryRanges(device, 1, &range);

        device_functions.vkDestroyCommandPool(device, pool, nullptr);


      buffer_offset = 0;
      for (size_t i = 0; i < info.mMipLevels; ++i) {
        const size_t width =
            subGetMipSize(nullptr, nullptr, info.mExtent.mWidth, i);
        const size_t height =
            subGetMipSize(nullptr, nullptr, info.mExtent.mHeight, i);
        const size_t depth =
            subGetMipSize(nullptr, nullptr, info.mExtent.mDepth, i);
        VkDeviceSize levelSize = subInferImageLevelSize(
            nullptr, nullptr, image.second, i);
        VkDeviceSize layerSize = levelSize / info.mArrayLayers;
        for (size_t j = 0; j < info.mArrayLayers; ++j) {
            auto& layer = image.second->mLayers[j];
            const Slice<uint8_t>& contents = layer->mLevels[i]->mData;
            const Pool* p = contents.getPool();

            GAPID_ASSERT(serializer.mVirtualPools.count(p) == 1);
            uint32_t pool_idx = serializer.mVirtualPools[p];

            auto memory_write = new memory_pb::MemoryWrite();
            memory_write->set_poolid(pool_idx);
            memory_write->set_offset(0);
            memory_write->set_size(layerSize);
            memory_write->mutable_data()->assign(reinterpret_cast<const char*>(data) + buffer_offset,
            layerSize);
            observer->encodeAndDelete(memory_write);

            buffer_offset += layerSize;
        }
      }


        device_functions.vkDestroyBuffer(device, copy_buffer, nullptr);
        device_functions.vkFreeMemory(device, copy_memory, nullptr);
      }
    }
    serializer.finalize();
}
}