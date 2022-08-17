/*
 * Copyright (C) 2019 Google Inc.
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

#include <stdlib.h>
#include <algorithm>
#include <sstream>
#include <thread>

#if defined(WIN32)
#include <direct.h>
#include <malloc.h>
#endif

#include "core/cc/make_unique.h"
#include "core/vulkan/vk_memory_tracker_layer/cc/tracing_helpers.h"
#include "perfetto/base/time.h"

#include "core/vulkan/vk_memory_tracker_layer/cc/memory_tracker_layer_impl.h"

using namespace std::chrono;

namespace memory_tracker {

using scoped_read_lock = layer_helpers::threading::scoped_read_lock;
using scoped_write_lock = layer_helpers::threading::scoped_write_lock;

MemoryTracker memory_tracker_instance;
rwlock rwl_global_unique_handles;
std::unordered_map<uint64_t, uint64_t> global_unique_handles;

// --------------------------- UniqueHandleGenerator ---------------------------

uint64_t UniqueHandleGenerator::Hash64(uint64_t handle, uint64_t counter) {
  std::vector<uint64_t> concat;
  concat.push_back(handle);
  concat.push_back(counter);
  auto unique_handle = CityHash64((const char*)(concat.data()), 16);
  {
    scoped_write_lock wlock(&rwl_global_unique_handles);
    global_unique_handles[handle] = unique_handle;
  }
  return unique_handle;
}

UniqueHandle UniqueHandleGenerator::GetImageHandle(VkImage image) {
  static uint64_t internal_image_counter = 0;
  return Hash64((uint64_t)(image), ++internal_image_counter);
}

UniqueHandle UniqueHandleGenerator::GetBufferHandle(VkBuffer buffer) {
  static uint64_t internal_buffer_counter = 0;
  return Hash64((uint64_t)(buffer), ++internal_buffer_counter);
}

UniqueHandle UniqueHandleGenerator::GetDeviceMemoryHandle(
    VkDeviceMemory device_memory) {
  static uint64_t internal_device_memory_counter = 0;
  return Hash64((uint64_t)(device_memory), ++internal_device_memory_counter);
}

// ----------------------- Wrapping allocation callbacks -----------------------

rwlock AllocationCallbacksTracker::rwl_global_callback_mapping;
std::unordered_map<uintptr_t, const VkAllocationCallbacks*>
    AllocationCallbacksTracker::global_callback_mapping;

rwlock AllocationCallbacksTracker::rwl_global_user_data_mapping;
std::unordered_map<uintptr_t, uintptr_t>
    AllocationCallbacksTracker::global_user_data_mapping;

rwlock AllocationCallbacksTracker::rwl_global_caller_api_mapping;
std::unordered_map<uintptr_t, std::string>
    AllocationCallbacksTracker::global_caller_api_mapping;

#if !defined(WIN32)
rwlock AllocationCallbacksTracker::rwl_global_allocation_size_mapping;
std::unordered_map<uintptr_t, size_t>
    AllocationCallbacksTracker::global_allocation_size_mapping;
#endif

AllocationCallbacksTracker::AllocationCallbacksTracker(
    const VkAllocationCallbacks* user_allocator,
    const std::string& caller_api) {
  if (user_allocator) {
    {
      scoped_write_lock wlock(&rwl_global_user_data_mapping);
      global_user_data_mapping[(uintptr_t)this] =
          (uintptr_t)(user_allocator->pUserData);
    }
    {
      scoped_write_lock wlock(&rwl_global_callback_mapping);
      global_callback_mapping[(uintptr_t)this] = user_allocator;
    }
  }
  {
    scoped_write_lock wlock(&rwl_global_caller_api_mapping);
    global_caller_api_mapping[(uintptr_t)this] = caller_api;
  }
  tracked_allocator.pUserData = this;
  tracked_allocator.pfnAllocation = &TrackedAllocationFunction;
  tracked_allocator.pfnReallocation = &TrackedReallocationFunction;
  tracked_allocator.pfnFree = &TrackedFreeFunction;
  if (user_allocator) {
    tracked_allocator.pfnInternalAllocation =
        user_allocator->pfnInternalAllocation;
    tracked_allocator.pfnInternalFree = user_allocator->pfnInternalFree;
  }
}

AllocationCallbacksHandle
AllocationCallbacksTracker::GetAllocationCallbacksHandle(
    const VkAllocationCallbacks* allocator, const std::string& caller) {
  std::stringstream stream;
  if (allocator) {
    stream << (uint64_t)(allocator->pUserData);
    stream << (uint64_t)(allocator->pfnAllocation);
    stream << (uint64_t)(allocator->pfnReallocation);
    stream << (uint64_t)(allocator->pfnFree);
    stream << (uint64_t)(allocator->pfnInternalAllocation);
    stream << (uint64_t)(allocator->pfnInternalFree);
  }
  stream << caller;
  std::string str = stream.str();
  return CityHash64((const char*)(str.c_str()), str.length());
}

void* AllocationCallbacksTracker::TrackedAllocationFunction(
    void* pUserData, size_t size, size_t alignment,
    VkSystemAllocationScope allocationScope) {
  void* ptr = nullptr;
  void* user_data = nullptr;
  const VkAllocationCallbacks* user_allocator = nullptr;
  AllocatorType allocator_type = atDefault;

  {
    scoped_read_lock rlock(&rwl_global_user_data_mapping);
    if (global_user_data_mapping.find((uintptr_t)pUserData) !=
        global_user_data_mapping.end()) {
      user_data = (void*)(global_user_data_mapping[(uintptr_t)pUserData]);
    }
  }

  {
    scoped_read_lock rlock(&rwl_global_callback_mapping);
    if (global_callback_mapping.find((uintptr_t)pUserData) !=
        global_callback_mapping.end()) {
      user_allocator = global_callback_mapping[(uintptr_t)pUserData];
    }
  }

  if (user_allocator && user_allocator->pfnAllocation) {
    allocator_type = atUser;
    ptr = user_allocator->pfnAllocation(user_data, size, alignment,
                                        allocationScope);
  } else {
#if defined(WIN32)
    ptr = _aligned_malloc(size, alignment);
#else
    auto corrected_alignment = std::max(alignment, sizeof(void*));
    if (posix_memalign(&ptr, corrected_alignment, size) != 0) {
      return ptr;
    }
#endif
  }
  if (ptr) {
    {
      scoped_write_lock wlock(&rwl_global_allocation_size_mapping);
      global_allocation_size_mapping[(uintptr_t)ptr] = size;
    }
    std::string caller_api = "Unknown";
    {
      scoped_read_lock rlock(&rwl_global_caller_api_mapping);
      if (global_caller_api_mapping.find((uintptr_t)pUserData) !=
          global_caller_api_mapping.end()) {
        caller_api = global_caller_api_mapping[(uintptr_t)pUserData];
      }
    }
    memory_tracker::memory_tracker_instance.ProcessHostMemoryAllocationEvent(
        (uintptr_t)ptr, size, alignment, allocationScope, caller_api,
        allocator_type);
  }
  return ptr;
}

void* AllocationCallbacksTracker::TrackedReallocationFunction(
    void* pUserData, void* pOriginal, size_t size, size_t alignment,
    VkSystemAllocationScope allocationScope) {
  if (!pOriginal) {
    return TrackedAllocationFunction(pUserData, size, alignment,
                                     allocationScope);
  }
  if (size == 0) {
    TrackedFreeFunction(pUserData, pOriginal);
    return nullptr;
  }
  size_t osize = 0;
  {
    scoped_read_lock rlock(&rwl_global_allocation_size_mapping);
    osize = (global_allocation_size_mapping.find((uintptr_t)pOriginal) ==
             global_allocation_size_mapping.end())
                ? 0
                : global_allocation_size_mapping[(uintptr_t)pOriginal];
  }
  if (osize == 0) {
    return TrackedAllocationFunction(pUserData, size, alignment,
                                     allocationScope);
  }
  if (osize == size) return pOriginal;

  void* ptr = nullptr;
  void* user_data = nullptr;
  const VkAllocationCallbacks* user_allocator = nullptr;
  AllocatorType allocator_type = atDefault;

  {
    scoped_read_lock rlock(&rwl_global_user_data_mapping);
    if (global_user_data_mapping.find((uintptr_t)pUserData) !=
        global_user_data_mapping.end()) {
      user_data = (void*)(global_user_data_mapping[(uintptr_t)pUserData]);
    }
  }

  {
    scoped_read_lock rlock(&rwl_global_callback_mapping);
    if (global_callback_mapping.find((uintptr_t)pUserData) !=
        global_callback_mapping.end()) {
      user_allocator = global_callback_mapping[(uintptr_t)pUserData];
    }
  }

  if (user_allocator && user_allocator->pfnReallocation) {
    allocator_type = atUser;
    ptr = user_allocator->pfnReallocation(user_data, pOriginal, size, alignment,
                                          allocationScope);
  } else {
#if defined(WIN32)
    ptr = _aligned_realloc(pOriginal, size, alignment);
#else
    auto corrected_alignment = std::max(alignment, sizeof(void*));
    if (posix_memalign(&ptr, corrected_alignment, size) != 0) {
      return ptr;
    }
#endif
    if (ptr) {
      size_t cpsize = osize < size ? osize : size;
      memcpy(ptr, pOriginal, cpsize);
#if defined(WIN32)
      _aligned_free(pOriginal);
#else
      free(pOriginal);
#endif
    }
  }
  if (ptr) {
    {
      scoped_write_lock wlock(&rwl_global_allocation_size_mapping);
      global_allocation_size_mapping[(uintptr_t)ptr] = size;
    }
    std::string caller_api = "Unknown";
    {
      scoped_read_lock rlock(&rwl_global_caller_api_mapping);
      if (global_caller_api_mapping.find((uintptr_t)pUserData) !=
          global_caller_api_mapping.end()) {
        caller_api = global_caller_api_mapping[(uintptr_t)pUserData];
      }
    }
    memory_tracker::memory_tracker_instance.ProcessHostMemoryReallocationEvent(
        (uintptr_t)ptr, (uintptr_t)pOriginal, size, alignment, allocationScope,
        caller_api, allocator_type);
  }
  return ptr;
}

void AllocationCallbacksTracker::TrackedFreeFunction(void* pUserData,
                                                     void* pMemory) {
  const VkAllocationCallbacks* user_allocator = nullptr;
  {
    scoped_read_lock rlock(&rwl_global_callback_mapping);
    if (global_callback_mapping.find((uintptr_t)pUserData) !=
        global_callback_mapping.end()) {
      user_allocator = global_callback_mapping[(uintptr_t)pUserData];
    }
  }

  memory_tracker::memory_tracker_instance.ProcessHostMemoryFreeEvent(
      (uintptr_t)pMemory);

  if (user_allocator && user_allocator->pfnFree) {
    void* user_data = nullptr;
    {
      scoped_read_lock rlock(&rwl_global_user_data_mapping);
      if (global_user_data_mapping.find((uintptr_t)pUserData) !=
          global_user_data_mapping.end()) {
        user_data = (void*)(global_user_data_mapping[(uintptr_t)pUserData]);
      }
    }
    user_allocator->pfnFree(user_data, pMemory);
  } else {
#if defined(WIN32)
    _aligned_free(pMemory);
#else
    free(pMemory);
#endif
  }
}

// ------------------------ Bookkeeping memory events --------------------------

BindMemoryInfo::BindMemoryInfo(VkDeviceMemory device_memory,
                               UniqueHandle device_memory_handle,
                               VkDeviceSize memory_offset,
                               uint32_t memory_type) {
  timestamp_ = perfetto::base::GetBootTimeNs().count();
  device_memory_ = device_memory;
  device_memory_handle_ = device_memory_handle;
  memory_offset_ = memory_offset;
  memory_type_ = memory_type;
}

VulkanMemoryEventPtr BindMemoryInfo::GetVulkanMemoryEvent() {
  auto event = make_unique<VulkanMemoryEvent>();
  // event->source will be set by the object that this bind memory info is
  // attached to, which can be a buffer or an image.
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_BIND;
  event->timestamp = timestamp_;
  event->has_device_memory = true;
  event->device_memory = device_memory_handle_;
  event->has_memory_address = true;
  event->memory_address = memory_offset_;
  event->has_memory_type = true;
  event->memory_type = memory_type_;
  return event;
}

//                     ------------------------------------

CreateBufferInfo::CreateBufferInfo(VkBufferCreateInfo const* create_buffer_info,
                                   VkDevice device) {
  timestamp = perfetto::base::GetBootTimeNs().count();
  this->device = device;
  flags = create_buffer_info->flags;
  size = create_buffer_info->size;
  usage = create_buffer_info->usage;
  sharing_mode = create_buffer_info->sharingMode;
  if (sharing_mode == VK_SHARING_MODE_CONCURRENT) {
    for (uint32_t i = 0; i < create_buffer_info->queueFamilyIndexCount; i++)
      queue_family_indices.push_back(
          create_buffer_info->pQueueFamilyIndices[i]);
  }
}

VulkanMemoryEventPtr CreateBufferInfo::GetVulkanMemoryEvent() {
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_BUFFER;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_CREATE;
  event->timestamp = timestamp;
  event->has_device = true;
  event->device = (uint64_t)(device);

  event->annotations.push_back(
      VulkanMemoryEventAnnotation("flags", (int)(flags)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("usage", (int)(usage)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("sharing_mode", (int)(sharing_mode)));
  for (auto index : queue_family_indices) {
    event->annotations.push_back(
        VulkanMemoryEventAnnotation("queue_family_index", (int)(index)));
  }
  return event;
}

Buffer::Buffer(VkBuffer buffer_, CreateBufferInfoPtr create_buffer_info_)
    : vk_buffer(buffer_) {
  is_bound = false;
  unique_handle = UniqueHandleGenerator::GetBufferHandle(buffer_);
  create_buffer_info = std::move(create_buffer_info_);
}

VulkanMemoryEventContainerPtr Buffer::GetVulkanMemoryEvents() {
  auto events = make_unique<VulkanMemoryEventContainer>();
  auto create_event = create_buffer_info->GetVulkanMemoryEvent();
  create_event->has_object_handle = true;
  create_event->object_handle = unique_handle;
  VkMemoryRequirements memory_requirements;
  GetGlobalContext()
      .GetVkDeviceData(create_buffer_info->GetVkDevice())
      ->functions->vkGetBufferMemoryRequirements(
          create_buffer_info->GetVkDevice(), vk_buffer, &memory_requirements);
  auto memory_size = memory_requirements.size;
  create_event->has_memory_size = true;
  create_event->memory_size = memory_size;
  create_event->annotations.push_back(
      VulkanMemoryEventAnnotation("vk_handle", (uint64_t)(vk_buffer)));
  events->push_back(std::move(create_event));

  if (is_bound) {
    auto bind_event = bind_buffer_info->GetVulkanMemoryEvent();
    bind_event->source =
        perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_BUFFER;
    bind_event->has_memory_size = true;
    bind_event->memory_size = memory_size;
    bind_event->has_object_handle = true;
    bind_event->object_handle = unique_handle;
    events->push_back(std::move(bind_event));
  }
  return events;
}

//                     ------------------------------------

CreateImageInfo::CreateImageInfo(VkImageCreateInfo const* create_image_info,
                                 VkDevice device) {
  timestamp = perfetto::base::GetBootTimeNs().count();
  this->device = device;
  flags = create_image_info->flags;
  image_type = create_image_info->imageType;
  format = create_image_info->format;
  extent = create_image_info->extent;
  mip_levels = create_image_info->mipLevels;
  array_layers = create_image_info->arrayLayers;
  samples = create_image_info->samples;
  tiling = create_image_info->tiling;
  usage = create_image_info->usage;
  sharing_mode = create_image_info->sharingMode;
  if (sharing_mode == VK_SHARING_MODE_CONCURRENT) {
    for (uint32_t i = 0; i < create_image_info->queueFamilyIndexCount; i++)
      queue_family_indices.push_back(create_image_info->pQueueFamilyIndices[i]);
  }
  initial_layout = create_image_info->initialLayout;
}

VulkanMemoryEventPtr CreateImageInfo::GetVulkanMemoryEvent() {
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_IMAGE;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_CREATE;
  event->timestamp = timestamp;
  event->has_device = true;
  event->device = (uint64_t)(device);

  event->annotations.push_back(
      VulkanMemoryEventAnnotation("flags", (int)(flags)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("image_type", (int)(image_type)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("format", (int)(format)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("extent.width", (int)(extent.width)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("extent.height", (int)(extent.height)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("extent.depth", (int)(extent.depth)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("mip_levels", (int)(mip_levels)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("array_layers", (int)(array_layers)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("samples", (int)(samples)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("tiling", (int)(tiling)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("usage", (int)(usage)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("sharing_mode", (int)(sharing_mode)));
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("initial_layout", (int)(initial_layout)));
  for (auto index : queue_family_indices) {
    event->annotations.push_back(
        VulkanMemoryEventAnnotation("queue_family_index", (int)(index)));
  }
  return event;
}

Image::Image(VkImage image_, CreateImageInfoPtr create_image_info_)
    : vk_image(image_) {
  is_bound = false;
  unique_handle = UniqueHandleGenerator::GetImageHandle(image_);
  create_image_info = std::move(create_image_info_);
}

VulkanMemoryEventContainerPtr Image::GetVulkanMemoryEvents() {
  auto events = make_unique<VulkanMemoryEventContainer>();
  auto create_event = create_image_info->GetVulkanMemoryEvent();
  create_event->has_object_handle = true;
  create_event->object_handle = unique_handle;
  VkMemoryRequirements memory_requirements;
  GetGlobalContext()
      .GetVkDeviceData(create_image_info->GetVkDevice())
      ->functions->vkGetImageMemoryRequirements(
          create_image_info->GetVkDevice(), vk_image, &memory_requirements);
  auto memory_size = memory_requirements.size;
  create_event->has_memory_size = true;
  create_event->memory_size = memory_size;

  create_event->annotations.push_back(
      VulkanMemoryEventAnnotation("vk_handle", (uint64_t)(vk_image)));
  events->push_back(std::move(create_event));

  if (is_bound) {
    auto bind_event = bind_image_info->GetVulkanMemoryEvent();
    bind_event->source =
        perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_IMAGE;
    bind_event->has_memory_size = true;
    bind_event->memory_size = memory_size;
    events->push_back(std::move(bind_event));
  }
  return events;
}

//                     ------------------------------------

DeviceMemory::DeviceMemory(VkDeviceMemory memory_,
                           VkMemoryAllocateInfo const* allocate_info)
    : memory(memory_) {
  timestamp = perfetto::base::GetBootTimeNs().count();
  allocation_size = allocate_info->allocationSize;
  memory_type = allocate_info->memoryTypeIndex;
  unique_handle = UniqueHandleGenerator::GetDeviceMemoryHandle(memory_);
}

VulkanMemoryEventPtr DeviceMemory::GetVulkanMemoryEvent() {
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DEVICE_MEMORY;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_CREATE;
  event->timestamp = timestamp;
  event->has_object_handle = true;
  event->object_handle = unique_handle;
  event->has_memory_size = true;
  event->memory_size = allocation_size;
  event->has_memory_type = true;
  event->memory_type = memory_type;
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("vk_handle", (uint64_t)(memory)));
  return event;
}

//                     ------------------------------------

Heap::Heap(VkDeviceSize size_, VkMemoryHeapFlags flags_)
    : size(size_), flags(flags_) {}

void Heap::AddDeviceMemory(DeviceMemoryPtr device_memory) {
  scoped_write_lock wlock(&rwl_device_memories);
  device_memories[device_memory->GetVkHandle()] = std::move(device_memory);
}

void Heap::DestroyDeviceMemory(VkDeviceMemory vk_device_memory) {
  DeviceMemoryPtr device_memory = nullptr;
  {
    scoped_read_lock rlock(&rwl_device_memories);
    if (device_memories.find(vk_device_memory) == device_memories.end()) return;
  }

  {
    scoped_write_lock wlock(&rwl_device_memories);
    device_memory = std::move(device_memories[vk_device_memory]);
    device_memories.erase(vk_device_memory);
  }

  {
    scoped_write_lock wlockb(&rwl_buffers);
    scoped_write_lock wlocki(&rwl_invalid_buffers);
    // Move all non-destroyed bound buffers to invalid container
    for (const auto& vk_buffer : device_memory->GetBoundBuffers()) {
      auto unique_handle = buffers[vk_buffer]->GetUniqueHandle();
      invalid_buffers[unique_handle] = std::move(buffers[vk_buffer]);
      buffers.erase(vk_buffer);
      device_memory->EmplaceInvalidBuffer(unique_handle);
    }
  }

  {
    scoped_write_lock wlockim(&rwl_images);
    scoped_write_lock wlockin(&rwl_invalid_images);
    // Move all non-destroyed bound images to invalid container
    for (const auto& vk_image : device_memory->GetBoundImages()) {
      auto unique_handle = images[vk_image]->GetUniqueHandle();
      invalid_images[unique_handle] = std::move(images[vk_image]);
      images.erase(vk_image);
      device_memory->EmplaceInvalidImage(unique_handle);
    }
  }

  device_memory->ClearBoundBuffers();
  device_memory->ClearBoundImages();
  {
    scoped_write_lock wlock(&rwl_invalid_device_memories);
    invalid_device_memories[device_memory->GetUniqueHandle()] =
        std::move(device_memory);
  }
}

void Heap::BindBuffer(BufferPtr buffer, VkDeviceMemory device_memory,
                      VkDeviceSize memory_offset) {
  uint32_t memory_type = UINT32_MAX;
  UniqueHandle device_memory_handle = UINT64_MAX;
  {
    scoped_read_lock rlock(&rwl_device_memories);
    if (device_memories.find(device_memory) == device_memories.end()) return;
    device_memory_handle = device_memories[device_memory]->GetUniqueHandle();
    memory_type = device_memories[device_memory]->GetMemoryType();
  }

  buffer->SetBound();
  buffer->SetBindBufferInfo(make_unique<BindMemoryInfo>(
      device_memory, device_memory_handle, memory_offset, memory_type));
  // Add to the device memory list of bound buffers
  {
    scoped_write_lock wlock(&rwl_device_memories);
    device_memories[device_memory]->EmplaceBoundBuffer(buffer->GetVkBuffer());
  }
  {
    scoped_write_lock wlock(&rwl_buffers);
    buffers[buffer->GetVkBuffer()] = std::move(buffer);
  }
}

void Heap::DestroyBuffer(VkBuffer vk_buffer) {
  {
    scoped_read_lock rlock(&rwl_buffers);
    if (buffers.find(vk_buffer) == buffers.end()) return;
  }

  BufferPtr buffer = nullptr;
  {
    scoped_write_lock wlock(&rwl_buffers);
    buffer = std::move(buffers[vk_buffer]);
    buffers.erase(vk_buffer);
  }

  // Remove buffer from the device memory list of bound buffers
  if (buffer && buffer->Bound()) {
    scoped_write_lock wlock(&rwl_device_memories);
    device_memories[buffer->GetDeviceMemory()]->EraseBoundBuffer(vk_buffer);
  }
}

void Heap::BindImage(ImagePtr image, VkDeviceMemory device_memory,
                     VkDeviceSize memory_offset) {
  uint32_t memory_type = UINT32_MAX;
  UniqueHandle device_memory_handle = UINT64_MAX;
  {
    scoped_read_lock rlock(&rwl_device_memories);
    if (device_memories.find(device_memory) == device_memories.end()) return;
    device_memory_handle = device_memories[device_memory]->GetUniqueHandle();
    memory_type = device_memories[device_memory]->GetMemoryType();
  }
  image->SetBound();
  image->SetBindImageInfo(make_unique<BindMemoryInfo>(
      device_memory, device_memory_handle, memory_offset, memory_type));
  // Add to the device memory list of bound images
  {
    scoped_write_lock wlock(&rwl_device_memories);
    device_memories[device_memory]->EmplaceBoundImage(image->GetVkImage());
  }
  {
    scoped_write_lock wlock(&rwl_images);
    images[image->GetVkImage()] = std::move(image);
  }
}

void Heap::DestroyImage(VkImage vk_image) {
  {
    scoped_read_lock rlock(&rwl_images);
    if (images.find(vk_image) == images.end()) return;
  }
  ImagePtr image = nullptr;
  {
    scoped_write_lock wlock(&rwl_images);
    image = std::move(images[vk_image]);
    images.erase(vk_image);
  }
  // Remove image from the device memory list of bound images
  if (image && image->Bound()) {
    scoped_write_lock wlock(&rwl_device_memories);
    device_memories[image->GetDeviceMemory()]->EraseBoundImage(vk_image);
  }
}

VulkanMemoryEventContainerPtr Heap::GetVulkanMemoryEvents(VkDevice device,
                                                          uint32_t heap_index) {
  auto events = make_unique<VulkanMemoryEventContainer>();
  {
    // Device memories
    scoped_read_lock rlock(&rwl_device_memories);
    for (auto it = device_memories.begin(); it != device_memories.end(); it++) {
      events->push_back(it->second->GetVulkanMemoryEvent());
      events->back()->has_device = true;
      events->back()->device = (uint64_t)(device);
      events->back()->has_heap = true;
      events->back()->heap = heap_index;
    }
  }

  {
    // Bound buffers
    scoped_read_lock rlock(&rwl_buffers);
    for (auto it = buffers.begin(); it != buffers.end(); it++) {
      auto buffer_events = it->second->GetVulkanMemoryEvents();
      for (auto itt = buffer_events->begin(); itt != buffer_events->end();) {
        events->push_back(std::move(*itt));
        events->back()->has_device = true;
        events->back()->device = (uint64_t)(device);
        events->back()->has_heap = true;
        events->back()->heap = heap_index;
        itt = buffer_events->erase(itt);
      }
    }
  }

  {
    // Bound images
    scoped_read_lock rlock(&rwl_images);
    for (auto it = images.begin(); it != images.end(); it++) {
      auto image_events = it->second->GetVulkanMemoryEvents();
      for (auto itt = image_events->begin(); itt != image_events->end();) {
        events->push_back(std::move(*itt));
        events->back()->has_device = true;
        events->back()->device = (uint64_t)(device);
        events->back()->has_heap = true;
        events->back()->heap = heap_index;
        itt = image_events->erase(itt);
      }
    }
  }
  return events;
}

//                     ------------------------------------

PhysicalDevice::PhysicalDevice(VkPhysicalDevice physical_device_)
    : physical_device(physical_device_) {
  timestamp = perfetto::base::GetBootTimeNs().count();
  GetGlobalContext()
      .GetVkPhysicalDeviceData(physical_device)
      ->functions->vkGetPhysicalDeviceMemoryProperties(physical_device,
                                                       &memory_properties);
  for (uint32_t i = 0; i < memory_properties.memoryTypeCount; ++i) {
    memory_type_index_to_heap_index.push_back(
        memory_properties.memoryTypes[i].heapIndex);
  }
  for (uint32_t i = 0; i < memory_properties.memoryHeapCount; ++i) {
    heaps[i] = make_unique<Heap>(memory_properties.memoryHeaps[i].size,
                                 memory_properties.memoryHeaps[i].flags);
  }
}

void PhysicalDevice::AddDeviceMemory(VkDevice device,
                                     DeviceMemoryPtr device_memory) {
  auto heap_index =
      memory_type_index_to_heap_index[device_memory->GetMemoryType()];
  auto vk_handle = device_memory->GetVkHandle();
  {
    scoped_write_lock wlock(&rwl_heaps);
    heaps[heap_index]->AddDeviceMemory(std::move(device_memory));
  }
  {
    scoped_write_lock wlock(&rwl_device_to_device_memory_set);
    if (device_to_device_memory_set.find(device) ==
        device_to_device_memory_set.end()) {
      DeviceMemorySet device_memory_set =
          make_unique<std::unordered_set<VkDeviceMemory>>();
      device_to_device_memory_set[device] = std::move(device_memory_set);
    }
    device_to_device_memory_set[device]->emplace(vk_handle);
  }
};

void PhysicalDevice::DestroyDeviceMemory(
    VkDevice device, VkDeviceMemory device_memory,
    bool erase_from_device_memory_set = true) {
  auto heap_index = -1;
  {
    scoped_write_lock wlock(&rwl_device_memory_to_heap_index);
    if (device_memory_to_heap_index.find(device_memory) !=
        device_memory_to_heap_index.end()) {
      heap_index = device_memory_to_heap_index[device_memory];
      device_memory_to_heap_index.erase(device_memory);
    }
  }
  if (heap_index < 0) return;
  {
    scoped_read_lock rlock(&rwl_heaps);
    heaps[heap_index]->DestroyDeviceMemory(device_memory);
  }
  if (erase_from_device_memory_set) {
    scoped_write_lock wlock(&rwl_device_to_device_memory_set);
    if (device_to_device_memory_set.find(device) !=
        device_to_device_memory_set.end()) {
      device_to_device_memory_set[device]->erase(device_memory);
    }
  }
}

void PhysicalDevice::DestroyAllDeviceMemories(VkDevice device) {
  DeviceMemorySet device_memory_set = nullptr;
  {
    scoped_write_lock wlock(&rwl_device_to_device_memory_set);
    if (device_to_device_memory_set.find(device) !=
        device_to_device_memory_set.end()) {
      device_memory_set = std::move(device_to_device_memory_set[device]);
      device_to_device_memory_set.erase(device);
    }
  }
  if (device_memory_set)
    for (auto& it : *device_memory_set) DestroyDeviceMemory(device, it, false);
}

void PhysicalDevice::BindBuffer(BufferPtr buffer, VkDeviceMemory device_memory,
                                VkDeviceSize size) {
  auto vk_buffer = buffer->GetVkBuffer();
  uint32_t heap_index = 0;
  {
    scoped_read_lock rlock(&rwl_buffer_to_heap_index);
    if (buffer_to_heap_index.find(vk_buffer) == buffer_to_heap_index.end())
      return;
    heap_index = buffer_to_heap_index[vk_buffer];
  }
  {
    scoped_write_lock wlock(&rwl_heaps);
    heaps[heap_index]->BindBuffer(std::move(buffer), device_memory, size);
  }
}

void PhysicalDevice::DestroyBuffer(VkBuffer vk_buffer) {
  uint32_t heap_index = 0;
  {
    scoped_read_lock rlock(&rwl_buffer_to_heap_index);
    if (buffer_to_heap_index.find(vk_buffer) == buffer_to_heap_index.end())
      return;
    heap_index = buffer_to_heap_index[vk_buffer];
    buffer_to_heap_index.erase(vk_buffer);
  }
  {
    scoped_read_lock rlock(&rwl_heaps);
    heaps[heap_index]->DestroyBuffer(vk_buffer);
  }
}

void PhysicalDevice::BindImage(ImagePtr image, VkDeviceMemory device_memory,
                               VkDeviceSize size) {
  auto vk_image = image->GetVkImage();
  uint32_t heap_index = 0;
  {
    scoped_read_lock rlock(&rwl_image_to_heap_index);
    if (image_to_heap_index.find(vk_image) == image_to_heap_index.end()) return;
    heap_index = image_to_heap_index[vk_image];
  }
  {
    scoped_read_lock rlock(&rwl_heaps);
    heaps[heap_index]->BindImage(std::move(image), device_memory, size);
  }
}

void PhysicalDevice::DestroyImage(VkImage vk_image) {
  uint32_t heap_index = 0;
  {
    scoped_read_lock rlock(&rwl_image_to_heap_index);
    if (image_to_heap_index.find(vk_image) == image_to_heap_index.end()) return;
    heap_index = image_to_heap_index[vk_image];
    image_to_heap_index.erase(vk_image);
  }
  {
    scoped_read_lock rlock(&rwl_heaps);
    heaps[heap_index]->DestroyImage(vk_image);
  }
}

VulkanMemoryEventPtr PhysicalDevice::GetVulkanMemoryEvent(VkDevice device) {
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DEVICE;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_ANNOTATIONS;
  event->timestamp = timestamp;
  event->has_object_handle = true;
  event->object_handle = (uint64_t)(device);

  {
    scoped_read_lock rlock(&rwl_heaps);
    for (auto it = heaps.begin(); it != heaps.end(); it++) {
      event->annotations.push_back(VulkanMemoryEventAnnotation(
          "heap_" + std::to_string(it->first) + "_size",
          (int64_t)(it->second->GetSize())));
      event->annotations.push_back(VulkanMemoryEventAnnotation(
          "heap_" + std::to_string(it->first) + "_flags",
          (int64_t)(it->second->GetFlags())));
    }
  }
  return event;
}

VulkanMemoryEventContainerSetPtr PhysicalDevice::GetVulkanMemoryEventsForHeaps(
    VkDevice device) {
  auto events = make_unique<VulkanMemoryEventContainerSet>();
  {
    scoped_read_lock rlock(&rwl_heaps);
    for (auto it = heaps.begin(); it != heaps.end(); it++)
      events->push_back(it->second->GetVulkanMemoryEvents(device, it->first));
  }
  return events;
}

//                     ------------------------------------

Device::Device(VkDevice device_, PhysicalDevicePtr physical_device_)
    : device(device_), physical_device(physical_device_) {
  timestamp = perfetto::base::GetBootTimeNs().count();
}

void Device::AddDeviceMemory(DeviceMemoryPtr device_memory) {
  physical_device->AddDeviceMemory(device, std::move(device_memory));
};

void Device::DestroyDeviceMemory(VkDeviceMemory device_memory) {
  physical_device->DestroyDeviceMemory(device, device_memory);
}

void Device::DestroyAllDeviceMemories() {
  physical_device->DestroyAllDeviceMemories(device);
}

void Device::AddBuffer(BufferPtr buffer_) {
  scoped_write_lock wlock(&rwl_buffers);
  buffers[buffer_->GetVkBuffer()] = std::move(buffer_);
}

void Device::BindBuffer(VkBuffer vk_buffer, VkDeviceMemory device_memory,
                        VkDeviceSize size) {
  {
    scoped_read_lock rlock(&rwl_buffers);
    if (buffers.find(vk_buffer) == buffers.end()) return;
  }
  {
    scoped_write_lock wlock(&rwl_buffers);
    BufferPtr buffer = std::move(buffers[vk_buffer]);
    buffers.erase(vk_buffer);
    physical_device->BindBuffer(std::move(buffer), device_memory, size);
  }
}

void Device::DestroyBuffer(VkBuffer vk_buffer) {
  bool buffer_destroyed = false;
  {
    scoped_write_lock wlock(&rwl_buffers);
    if (buffers.find(vk_buffer) != buffers.end()) {
      buffers.erase(vk_buffer);
      buffer_destroyed = true;
    }
  }
  if (!buffer_destroyed) physical_device->DestroyBuffer(vk_buffer);
}

void Device::AddImage(ImagePtr image_) {
  scoped_write_lock wlock(&rwl_images);
  images[image_->GetVkImage()] = std::move(image_);
}

void Device::BindImage(VkImage vk_image, VkDeviceMemory device_memory,
                       VkDeviceSize size) {
  {
    scoped_read_lock rlock(&rwl_images);
    if (images.find(vk_image) == images.end()) return;
  }
  {
    scoped_write_lock wlock(&rwl_images);
    ImagePtr image = std::move(images[vk_image]);
    images.erase(vk_image);
    physical_device->BindImage(std::move(image), device_memory, size);
  }
}

void Device::DestroyImage(VkImage vk_image) {
  bool image_destroyed = false;
  {
    scoped_write_lock wlock(&rwl_images);
    if (images.find(vk_image) != images.end()) {
      images.erase(vk_image);
      image_destroyed = true;
    }
  }
  if (!image_destroyed) physical_device->DestroyImage(vk_image);
}

uint32_t Device::GetHeapIndex(uint32_t memory_type) {
  if (!physical_device) return UINT32_MAX - 1;
  return physical_device->GetHeapIndex(memory_type);
}

// This is called when emitting the stored state of the memory usage,
// hence we can free the objects after we generated the trace packets
// for them. We keep the device and physical device info since they
// will be referred later in bind events.
VulkanMemoryEventContainerSetPtr Device::GetVulkanMemoryEvents() {
  auto events = make_unique<VulkanMemoryEventContainer>();
  // Add an event for the device itself
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DEVICE;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_CREATE;
  event->timestamp = timestamp;
  event->has_object_handle = true;
  event->object_handle = (uint64_t)(device);
  event->annotations.push_back(VulkanMemoryEventAnnotation(
      "physical_device", (uint64_t)(physical_device->GetVkPhysicalDevice())));
  events->push_back(std::move(event));

  // Add physical device info
  events->push_back(physical_device->GetVulkanMemoryEvent(device));

  // Add unbound images and buffers
  {
    scoped_read_lock rlock(&rwl_buffers);
    for (auto it = buffers.begin(); it != buffers.end(); it++) {
      auto buffer_events = it->second->GetVulkanMemoryEvents();
      for (auto itt = buffer_events->begin(); itt != buffer_events->end();) {
        events->push_back(std::move(*itt));
        events->back()->has_device = true;
        events->back()->device = (uint64_t)(device);
        itt = buffer_events->erase(itt);
      }
    }
  }

  {
    scoped_read_lock rlock(&rwl_images);
    for (auto it = images.begin(); it != images.end(); it++) {
      auto image_events = it->second->GetVulkanMemoryEvents();
      for (auto itt = image_events->begin(); itt != image_events->end();) {
        events->push_back(std::move(*itt));
        events->back()->has_device = true;
        events->back()->device = (uint64_t)(device);
        itt = image_events->erase(itt);
      }
    }
  }

  // Get events for  device memories, bound buffers and bound images
  auto events_set = physical_device->GetVulkanMemoryEventsForHeaps(device);
  events_set->insert(events_set->begin(), std::move(events));
  return events_set;
}

//                     ------------------------------------

VulkanMemoryEventPtr HostAllocation::GetVulkanMemoryEvent() {
  auto event = make_unique<core::VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DRIVER;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_CREATE;
  event->timestamp = timestamp;
  event->has_memory_address = true;
  event->memory_address = ptr;
  event->has_memory_size = true;
  event->memory_size = size;
  event->function_name = caller_api;
  event->has_allocation_scope = true;
  event->allocation_scope =
      static_cast<perfetto::protos::pbzero::VulkanMemoryEvent::AllocationScope>(
          scope + 1);

  event->annotations.push_back(
      VulkanMemoryEventAnnotation("alignment", (int)(alignment)));
  event->annotations.push_back(VulkanMemoryEventAnnotation(
      "allocator", (allocator_type == atDefault) ? "dafault" : "user"));
  return event;
}

// --------------------------- Memory events tracker ---------------------------
MemoryTracker::MemoryTracker() : initial_state_is_sent_(false) {}

const VkAllocationCallbacks* MemoryTracker::GetTrackedAllocator(
    const VkAllocationCallbacks* pUserAllocator, const std::string& caller) {
  if (!Emit().CategoryEnabled("Driver")) return pUserAllocator;

  auto cb_handle = AllocationCallbacksTracker::GetAllocationCallbacksHandle(
      pUserAllocator, caller);
  scoped_write_lock wlock(&rwl_allocation_trackers);
  auto it = m_allocation_callbacks_trackers.find(cb_handle);
  if (it != m_allocation_callbacks_trackers.end())
    return it->second->TrackedAllocator();
  auto allocation_cb_tracker =
      make_unique<AllocationCallbacksTracker>(pUserAllocator, caller);
  m_allocation_callbacks_trackers[cb_handle] = std::move(allocation_cb_tracker);
  return m_allocation_callbacks_trackers[cb_handle]->TrackedAllocator();
}

// --------------- Storing the events in the state of the memory ---------------

void MemoryTracker::StoreCreateDeviceEvent(
    VkPhysicalDevice physical_device, VkDeviceCreateInfo const* create_info,
    VkDevice device) {
  {
    scoped_write_lock wlock(&rwl_physical_devices);
    if (physical_devices.find(physical_device) == physical_devices.end()) {
      physical_devices[physical_device] =
          make_unique<PhysicalDevice>(physical_device);
    }
  }

  {
    scoped_write_lock wlock(&rwl_devices);
    devices[device] =
        make_unique<Device>(device, physical_devices[physical_device]);
  }
}

void MemoryTracker::StoreDestoryDeviceEvent(VkDevice vk_device) {
  DevicePtr device = nullptr;
  {
    scoped_write_lock wlock(&rwl_devices);
    if (devices.find(vk_device) != devices.end()) {
      device = std::move(devices[vk_device]);
      devices.erase(vk_device);
    }
  }
  if (device) device->DestroyAllDeviceMemories();
}

void MemoryTracker::StoreAllocateMemoryEvent(
    VkDevice device, VkDeviceMemory memory,
    VkMemoryAllocateInfo const* allocate_info) {
  auto device_memory = make_unique<DeviceMemory>(memory, allocate_info);
  {
    scoped_read_lock rlock(&rwl_devices);
    if (devices.find(device) != devices.end())
      devices[device]->AddDeviceMemory(std::move(device_memory));
  }
  {
    scoped_write_lock wlock(&rwl_device_memory_type_map);
    device_memory_type_map[memory] = allocate_info->memoryTypeIndex;
  }
}

void MemoryTracker::StoreFreeMemoryEvent(VkDevice device,
                                         VkDeviceMemory device_memory) {
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->DestroyDeviceMemory(device_memory);
}

void MemoryTracker::StoreCreateBufferEvent(
    VkDevice device, VkBuffer buffer, VkBufferCreateInfo const* create_info) {
  CreateBufferInfoPtr create_info_ptr =
      make_unique<CreateBufferInfo>(create_info, device);
  BufferPtr buffer_ptr =
      make_unique<Buffer>(buffer, std::move(create_info_ptr));
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->AddBuffer(std::move(buffer_ptr));
}

void MemoryTracker::StoreBindBufferEvent(VkDevice device, VkBuffer buffer,
                                         VkDeviceMemory memory, size_t offset) {
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->BindBuffer(buffer, memory, offset);
}

void MemoryTracker::StoreDestroyBufferEvent(VkDevice device, VkBuffer buffer) {
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->DestroyBuffer(buffer);
}

void MemoryTracker::StoreCreateImageEvent(
    VkDevice device, VkImage image, const VkImageCreateInfo* create_info) {
  CreateImageInfoPtr create_info_ptr =
      make_unique<CreateImageInfo>(create_info, device);
  ImagePtr image_ptr = make_unique<Image>(image, std::move(create_info_ptr));
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->AddImage(std::move(image_ptr));
}

void MemoryTracker::StoreBindImageEvent(VkDevice device, VkImage image,
                                        VkDeviceMemory memory, size_t offset) {
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->BindImage(image, memory, offset);
}

void MemoryTracker::StoreDestroyImageEvent(VkDevice device, VkImage image) {
  scoped_read_lock rlock(&rwl_devices);
  if (devices.find(device) != devices.end())
    devices[device]->DestroyImage(image);
}

void MemoryTracker::StoreHostMemoryAllocationEvent(
    uintptr_t ptr, size_t size, size_t alignment, VkSystemAllocationScope scope,
    const std::string& caller_api, AllocatorType allocator_type) {
  auto timestamp = perfetto::base::GetBootTimeNs().count();
  HostAllocationPtr allocation = make_unique<HostAllocation>(
      timestamp, ptr, size, alignment, scope, caller_api, allocator_type);
  scoped_write_lock wlock(&rwl_host_allocations);
  host_allocations[ptr] = std::move(allocation);
}

void MemoryTracker::StoreHostMemoryReallocationEvent(
    uintptr_t ptr, uintptr_t original, size_t size, size_t alignment,
    VkSystemAllocationScope scope, const std::string& caller_api,
    AllocatorType allocator_type) {
  auto timestamp = perfetto::base::GetBootTimeNs().count();
  HostAllocationPtr allocation = make_unique<HostAllocation>(
      timestamp, ptr, size, alignment, scope, caller_api, allocator_type);
  scoped_write_lock wlock(&rwl_host_allocations);
  host_allocations[ptr] = std::move(allocation);
  if (original != ptr &&
      host_allocations.find(original) != host_allocations.end()) {
    host_allocations[original].reset(nullptr);
    host_allocations.erase(original);
  }
}

void MemoryTracker::StoreHostMemoryFreeEvent(uintptr_t ptr) {
  scoped_write_lock wlock(&rwl_host_allocations);
  if (host_allocations.find(ptr) != host_allocations.end()) {
    host_allocations[ptr].reset(nullptr);
    host_allocations.erase(ptr);
  }
}

void MemoryTracker::EmitAndClearAllStoredEvents() {
  // Emit and clear GPU memory events. Device information also includes physical
  // device and heaps information.
  if (Emit().CategoryEnabled("Device")) {
    scoped_read_lock rlock(&rwl_devices);
    for (auto it = devices.begin(); it != devices.end(); ++it) {
      auto event_container_set = it->second->GetVulkanMemoryEvents();
      for (auto& events : *event_container_set) {
        for (auto& event : *events)
          Emit().EmitVulkanMemoryUsageEvent(event.get());
        events->clear();
      }
      event_container_set->clear();
    }
  }

  // Emit and clear host memory events
  if (Emit().CategoryEnabled("Driver")) {
    scoped_read_lock rlock(&rwl_host_allocations);
    for (auto it = host_allocations.begin(); it != host_allocations.end();
         it++) {
      Emit().EmitVulkanMemoryUsageEvent(
          it->second->GetVulkanMemoryEvent().get());
    }
  }
}

void MemoryTracker::EmitAllStoredEventsIfNecessary() {
  if (initial_state_is_sent_) return;
  initial_state_is_sent_ = true;
  // While generating the memory usage events, we do not care about the thread
  // info. Therefore, we can safely delegate sending the stored events to
  // another thread.
  std::thread emitter(&MemoryTracker::EmitAndClearAllStoredEvents, this);
  emitter.join();
}

// ----------------- Send the events directly to trace daemon -----------------

void MemoryTracker::EmitCreateDeviceEvent(VkPhysicalDevice physical_device,
                                          VkDeviceCreateInfo const* create_info,
                                          VkDevice device) {
  EmitAllStoredEventsIfNecessary();

  VulkanMemoryEventPtr pdevice_event = nullptr;
  {
    scoped_write_lock wlock(&rwl_physical_devices);
    if (physical_devices.find(physical_device) == physical_devices.end()) {
      physical_devices[physical_device] =
          make_unique<PhysicalDevice>(physical_device);
      pdevice_event =
          physical_devices[physical_device]->GetVulkanMemoryEvent(device);
    }
    if (pdevice_event) Emit().EmitVulkanMemoryUsageEvent(pdevice_event.get());
  }

  {
    scoped_write_lock wlock(&rwl_devices);
    devices[device] =
        make_unique<Device>(device, physical_devices[physical_device]);
  }
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DEVICE;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_CREATE;
  event->timestamp = perfetto::base::GetBootTimeNs().count();
  event->has_object_handle = true;
  event->object_handle = (uint64_t)(device);
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitDestoryDeviceEvent(VkDevice device) {
  EmitAllStoredEventsIfNecessary();
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DEVICE;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_DESTROY;
  event->timestamp = perfetto::base::GetBootTimeNs().count();
  event->has_object_handle = true;
  event->object_handle = (uint64_t)(device);
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitAllocateMemoryEvent(
    VkDevice device, VkDeviceMemory memory,
    VkMemoryAllocateInfo const* allocate_info) {
  EmitAllStoredEventsIfNecessary();
  auto device_memory = make_unique<DeviceMemory>(memory, allocate_info);
  auto event = device_memory->GetVulkanMemoryEvent();
  event->has_device = true;
  event->device = (uint64_t)(device);
  auto memory_type = allocate_info->memoryTypeIndex;
  event->has_heap = true;
  if (!devices[device])
    event->heap = UINT32_MAX;
  else
    event->heap = devices[device]->GetHeapIndex(memory_type);
  event->has_memory_type = true;
  event->memory_type = memory_type;
  {
    scoped_write_lock wlock(&rwl_device_memory_type_map);
    device_memory_type_map[memory] = memory_type;
  }
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitFreeMemoryEvent(VkDevice device,
                                        VkDeviceMemory device_memory) {
  EmitAllStoredEventsIfNecessary();
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DEVICE_MEMORY;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_DESTROY;
  event->timestamp = perfetto::base::GetBootTimeNs().count();
  event->has_device = true;
  event->device = (uint64_t)(device);
  event->has_object_handle = true;
  event->object_handle = (uint64_t)(device_memory);
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitCreateBufferEvent(
    VkDevice device, VkBuffer buffer, VkBufferCreateInfo const* create_info) {
  EmitAllStoredEventsIfNecessary();
  auto crinfo = CreateBufferInfo(create_info, device);
  auto event = crinfo.GetVulkanMemoryEvent();
  event->has_device = true;
  event->device = (uint64_t)(device);
  VkMemoryRequirements memory_requirements;
  GetGlobalContext()
      .GetVkDeviceData(device)
      ->functions->vkGetBufferMemoryRequirements(device, buffer,
                                                 &memory_requirements);
  event->has_memory_size = true;
  event->memory_size = memory_requirements.size;
  event->has_object_handle = true;
  auto buffer_handle = UniqueHandleGenerator::GetBufferHandle(buffer);
  event->object_handle = buffer_handle;
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("vk_handle", (uint64_t)(buffer)));
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitBindBufferEvent(VkDevice device, VkBuffer buffer,
                                        VkDeviceMemory device_memory,
                                        size_t offset) {
  EmitAllStoredEventsIfNecessary();
  uint32_t memory_type = UINT32_MAX;
  {
    scoped_read_lock rlock(&rwl_device_memory_type_map);
    memory_type = device_memory_type_map[device_memory];
  }
  auto bindinfo = BindMemoryInfo(
      device_memory, global_unique_handles[(uint64_t)(device_memory)], offset,
      memory_type);
  auto event = bindinfo.GetVulkanMemoryEvent();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_BUFFER;
  event->has_device = true;
  event->device = (uint64_t)(device);
  event->has_heap = true;
  {
    scoped_read_lock rlock(&rwl_devices);
    event->heap = devices[device]->GetHeapIndex(memory_type);
  }
  event->has_object_handle = true;
  event->object_handle = global_unique_handles[(uint64_t)(buffer)];
  VkMemoryRequirements memory_requirements;
  GetGlobalContext()
      .GetVkDeviceData(device)
      ->functions->vkGetBufferMemoryRequirements(device, buffer,
                                                 &memory_requirements);
  event->has_memory_size = true;
  event->memory_size = memory_requirements.size;
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitDestroyBufferEvent(VkDevice device, VkBuffer buffer) {
  EmitAllStoredEventsIfNecessary();
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_BUFFER;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_DESTROY;
  event->timestamp = perfetto::base::GetBootTimeNs().count();
  event->has_device = true;
  event->device = (uint64_t)(device);
  event->has_object_handle = true;
  event->object_handle = global_unique_handles[(uint64_t)(buffer)];
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitCreateImageEvent(VkDevice device, VkImage image,
                                         VkImageCreateInfo const* create_info) {
  EmitAllStoredEventsIfNecessary();
  auto crinfo = CreateImageInfo(create_info, device);
  auto event = crinfo.GetVulkanMemoryEvent();
  event->has_device = true;
  event->device = (uint64_t)(device);
  VkMemoryRequirements memory_requirements;
  GetGlobalContext()
      .GetVkDeviceData(device)
      ->functions->vkGetImageMemoryRequirements(device, image,
                                                &memory_requirements);
  event->has_memory_size = true;
  event->memory_size = memory_requirements.size;
  event->has_object_handle = true;
  auto image_handle = UniqueHandleGenerator::GetImageHandle(image);
  event->object_handle = image_handle;
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("vk_handle", (uint64_t)(image)));
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitBindImageEvent(VkDevice device, VkImage image,
                                       VkDeviceMemory device_memory,
                                       size_t offset) {
  EmitAllStoredEventsIfNecessary();
  uint32_t memory_type = UINT32_MAX;
  {
    scoped_read_lock rlock(&rwl_device_memory_type_map);
    memory_type = device_memory_type_map[device_memory];
  }
  auto bindinfo = BindMemoryInfo(
      device_memory, global_unique_handles[(uint64_t)(device_memory)], offset,
      memory_type);
  auto event = bindinfo.GetVulkanMemoryEvent();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_IMAGE;
  event->has_device = true;
  event->device = (uint64_t)(device);
  event->has_heap = true;
  {
    scoped_read_lock rlock(&rwl_devices);
    event->heap = devices[device]->GetHeapIndex(memory_type);
  }
  event->has_object_handle = true;
  event->object_handle = global_unique_handles[(uint64_t)(image)];
  VkMemoryRequirements memory_requirements;
  GetGlobalContext()
      .GetVkDeviceData(device)
      ->functions->vkGetImageMemoryRequirements(device, image,
                                                &memory_requirements);
  event->has_memory_size = true;
  event->memory_size = memory_requirements.size;
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitDestroyImageEvent(VkDevice device, VkImage image) {
  EmitAllStoredEventsIfNecessary();
  auto event = make_unique<VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_IMAGE;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_DESTROY;
  event->timestamp = perfetto::base::GetBootTimeNs().count();
  event->has_device = true;
  event->device = (uint64_t)(device);
  event->has_object_handle = true;
  event->object_handle = global_unique_handles[(uint64_t)(image)];
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitHostMemoryAllocationEvent(
    uintptr_t ptr, size_t size, size_t alignment, VkSystemAllocationScope scope,
    const std::string& caller_api, AllocatorType allocator_type) {
  EmitAllStoredEventsIfNecessary();
  auto timestamp = perfetto::base::GetBootTimeNs().count();
  HostAllocationPtr allocation = make_unique<HostAllocation>(
      timestamp, ptr, size, alignment, scope, caller_api, allocator_type);
  auto event = allocation->GetVulkanMemoryEvent();
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitHostMemoryReallocationEvent(
    uintptr_t ptr, uintptr_t original, size_t size, size_t alignment,
    VkSystemAllocationScope scope, const std::string& caller_api,
    AllocatorType allocator_type) {
  EmitAllStoredEventsIfNecessary();
  auto timestamp = perfetto::base::GetBootTimeNs().count();
  HostAllocationPtr allocation = make_unique<HostAllocation>(
      timestamp, ptr, size, alignment, scope, caller_api, allocator_type);
  auto event = allocation->GetVulkanMemoryEvent();
  event->annotations.push_back(
      VulkanMemoryEventAnnotation("original_ptr", (int)(original)));
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

void MemoryTracker::EmitHostMemoryFreeEvent(uintptr_t ptr) {
  EmitAllStoredEventsIfNecessary();
  auto event = make_unique<core::VulkanMemoryEvent>();
  event->source =
      perfetto::protos::pbzero::VulkanMemoryEvent_Source::SOURCE_DRIVER;
  event->operation =
      perfetto::protos::pbzero::VulkanMemoryEvent_Operation::OP_DESTROY;
  event->timestamp = perfetto::base::GetBootTimeNs().count();
  event->has_memory_address = true;
  event->memory_address = ptr;
  Emit().EmitVulkanMemoryUsageEvent(event.get());
}

// ---------------------------- Process the events ----------------------------

void MemoryTracker::ProcessCreateDeviceEvent(
    VkPhysicalDevice physical_device, VkDeviceCreateInfo const* create_info,
    VkDevice device) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled())
    return EmitCreateDeviceEvent(physical_device, create_info, device);
  return StoreCreateDeviceEvent(physical_device, create_info, device);
}

void MemoryTracker::ProcessDestoryDeviceEvent(VkDevice vk_device) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled()) return EmitDestoryDeviceEvent(vk_device);
  return StoreDestoryDeviceEvent(vk_device);
}

void MemoryTracker::ProcessAllocateMemoryEvent(
    VkDevice device, VkDeviceMemory memory,
    VkMemoryAllocateInfo const* allocate_info) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled())
    return EmitAllocateMemoryEvent(device, memory, allocate_info);
  return StoreAllocateMemoryEvent(device, memory, allocate_info);
}

void MemoryTracker::ProcessFreeMemoryEvent(VkDevice vk_device,
                                           VkDeviceMemory vk_device_memory) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled()) return EmitFreeMemoryEvent(vk_device, vk_device_memory);
  return StoreFreeMemoryEvent(vk_device, vk_device_memory);
}

void MemoryTracker::ProcessCreateBufferEvent(
    VkDevice device, VkBuffer buffer, VkBufferCreateInfo const* create_info) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled())
    return EmitCreateBufferEvent(device, buffer, create_info);
  return StoreCreateBufferEvent(device, buffer, create_info);
}

void MemoryTracker::ProcessBindBufferEvent(VkDevice device, VkBuffer buffer,
                                           VkDeviceMemory memory,
                                           size_t offset) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled())
    return EmitBindBufferEvent(device, buffer, memory, offset);
  return StoreBindBufferEvent(device, buffer, memory, offset);
}

void MemoryTracker::ProcessDestroyBufferEvent(VkDevice device,
                                              VkBuffer buffer) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled()) return EmitDestroyBufferEvent(device, buffer);
  return StoreDestroyBufferEvent(device, buffer);
}

void MemoryTracker::ProcessCreateImageEvent(
    VkDevice device, VkImage image, const VkImageCreateInfo* create_info) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled()) return EmitCreateImageEvent(device, image, create_info);
  return StoreCreateImageEvent(device, image, create_info);
}

void MemoryTracker::ProcessBindImageEvent(VkDevice device, VkImage image,
                                          VkDeviceMemory memory,
                                          size_t offset) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled())
    return EmitBindImageEvent(device, image, memory, offset);
  return StoreBindImageEvent(device, image, memory, offset);
}

void MemoryTracker::ProcessDestroyImageEvent(VkDevice device, VkImage image) {
  if (!Emit().CategoryEnabled("Device")) return;
  if (Emit().Enabled()) return EmitDestroyImageEvent(device, image);
  return StoreDestroyImageEvent(device, image);
}

void MemoryTracker::ProcessHostMemoryAllocationEvent(
    uintptr_t ptr, size_t size, size_t alignment, VkSystemAllocationScope scope,
    const std::string& caller_api, AllocatorType allocator_type) {
  if (Emit().Enabled())
    return EmitHostMemoryAllocationEvent(ptr, size, alignment, scope,
                                         caller_api, allocator_type);
  return StoreHostMemoryAllocationEvent(ptr, size, alignment, scope, caller_api,
                                        allocator_type);
}

void MemoryTracker::ProcessHostMemoryReallocationEvent(
    uintptr_t ptr, uintptr_t original, size_t size, size_t alignment,
    VkSystemAllocationScope scope, const std::string& caller_api,
    AllocatorType allocator_type) {
  if (Emit().Enabled())
    return EmitHostMemoryReallocationEvent(ptr, original, size, alignment,
                                           scope, caller_api, allocator_type);
  return StoreHostMemoryReallocationEvent(ptr, original, size, alignment, scope,
                                          caller_api, allocator_type);
}

void MemoryTracker::ProcessHostMemoryFreeEvent(uintptr_t ptr) {
  if (Emit().Enabled()) return EmitHostMemoryFreeEvent(ptr);
  return StoreHostMemoryFreeEvent(ptr);
}

}  // namespace memory_tracker