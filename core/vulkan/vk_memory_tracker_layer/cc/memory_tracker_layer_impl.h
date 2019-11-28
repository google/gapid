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

#ifndef MEMORY_TRACKER_LAYER_IMPL_H
#define MEMORY_TRACKER_LAYER_IMPL_H

#include <city.h>

#include <chrono>
#include <deque>
#include <functional>
#include <iostream>
#include <map>
#include <memory>
#include <sstream>
#include <string>
#include <unordered_map>
#include <unordered_set>

#include "core/vulkan/layer_helpers/threading.h"
#include "core/vulkan/perfetto_producer/perfetto_proto_structs.h"
#include "core/vulkan/vk_memory_tracker_layer/cc/layer.h"

namespace memory_tracker {

// ---------------------- Events bookkeeping data structs ----------------------

// In the event of freeing the memory calling vkFreeMemory, the Vulkan spec does
// not enforce freeing the bound images and buffers. Therefore the handles may
// remain dangling around.  To honor the spec, we don't cascade deletion from
// device memories to bound images and buffers, and always keep record of the
// previous device memory objects in case of any bound image or buffer still
// dangling around. Therefore, to correctly map those dangling bound resources
// to the destroyed device memory, we need to keep the old record around. To
// make sure that we're safe on the handle side, we use a unique handle
// generator as the key to the image, buffer and device memory
// not-properly-destroyed objects.

typedef uint64_t UniqueHandle;
class UniqueHandleGenerator {
 private:
  static uint64_t Hash64(uint64_t, uint64_t);

 public:
  static UniqueHandle GetImageHandle(VkImage);
  static UniqueHandle GetBufferHandle(VkBuffer);
  static UniqueHandle GetDeviceMemoryHandle(VkDeviceMemory);
};

using mutex = layer_helpers::threading::mutex;
using rwlock = layer_helpers::threading::rwlock;
using VulkanMemoryEvent = core::VulkanMemoryEvent;
using VulkanMemoryEventAnnotation = core::VulkanMemoryEventAnnotation;

using VulkanMemoryEventPtr = std::unique_ptr<VulkanMemoryEvent>;
using VulkanMemoryEventContainer = std::deque<VulkanMemoryEventPtr>;
using VulkanMemoryEventContainerPtr =
    std::unique_ptr<VulkanMemoryEventContainer>;
using VulkanMemoryEventContainerSet = std::deque<VulkanMemoryEventContainerPtr>;
using VulkanMemoryEventContainerSetPtr =
    std::unique_ptr<VulkanMemoryEventContainerSet>;

// Used for CreateImageInfo and
class BindMemoryInfo {
 public:
  BindMemoryInfo(VkDeviceMemory, UniqueHandle, VkDeviceSize,
                 uint32_t /*memory_type*/);
  VulkanMemoryEventPtr GetVulkanMemoryEvent();
  VkDeviceMemory GetDeviceMemory() { return device_memory_; }

 private:
  uint64_t timestamp_;
  VkDeviceMemory device_memory_;
  UniqueHandle device_memory_handle_;
  VkDeviceSize memory_offset_;
  uint32_t memory_type_;
};
using BindMemoryInfoPtr = std::unique_ptr<BindMemoryInfo>;

class CreateBufferInfo {
 public:
  CreateBufferInfo(VkBufferCreateInfo const*, VkDevice);
  VulkanMemoryEventPtr GetVulkanMemoryEvent();
  VkDevice GetVkDevice() { return device; }

 private:
  uint64_t timestamp;
  VkDevice device;
  VkBufferCreateFlags flags;
  VkDeviceSize size;
  VkBufferUsageFlags usage;
  VkSharingMode sharing_mode;
  std::deque<uint32_t> queue_family_indices;
};
using CreateBufferInfoPtr = std::unique_ptr<CreateBufferInfo>;

// Parent class for Buffer and Image
class MemoryObject {
 public:
  UniqueHandle GetUniqueHandle() { return unique_handle; }
  void SetBound() { is_bound = true; }
  bool Bound() { return is_bound; }

 protected:
  bool is_bound;
  UniqueHandle unique_handle;
};

class Buffer : public MemoryObject {
 public:
  Buffer(VkBuffer, CreateBufferInfoPtr);
  VulkanMemoryEventContainerPtr GetVulkanMemoryEvents();
  VkBuffer GetVkBuffer() { return vk_buffer; }
  VkDeviceMemory GetDeviceMemory() {
    return bind_buffer_info->GetDeviceMemory();
  }
  void SetBindBufferInfo(BindMemoryInfoPtr info) {
    bind_buffer_info = std::move(info);
  }

 private:
  const VkBuffer vk_buffer;
  CreateBufferInfoPtr create_buffer_info;
  BindMemoryInfoPtr bind_buffer_info;
};
using BufferPtr = std::unique_ptr<Buffer>;
using BufferMap = std::unordered_map<VkBuffer, BufferPtr>;
using BufferMapInvalid = std::unordered_map<UniqueHandle, BufferPtr>;

class CreateImageInfo {
 public:
  CreateImageInfo(VkImageCreateInfo const*, VkDevice);
  VulkanMemoryEventPtr GetVulkanMemoryEvent();
  VkDevice GetVkDevice() { return device; }

 private:
  uint64_t timestamp;
  VkDevice device;
  VkImageCreateFlags flags;
  VkImageType image_type;
  VkFormat format;
  VkExtent3D extent;
  uint32_t mip_levels;
  uint32_t array_layers;
  VkSampleCountFlagBits samples;
  VkImageTiling tiling;
  VkImageUsageFlags usage;
  VkSharingMode sharing_mode;
  std::deque<uint32_t> queue_family_indices;
  VkImageLayout initial_layout;
};
using CreateImageInfoPtr = std::unique_ptr<CreateImageInfo>;

class Image : public MemoryObject {
 public:
  Image(VkImage, CreateImageInfoPtr);
  VulkanMemoryEventContainerPtr GetVulkanMemoryEvents();
  VkImage GetVkImage() { return vk_image; }
  VkDeviceMemory GetDeviceMemory() {
    return bind_image_info->GetDeviceMemory();
  }
  void SetBindImageInfo(BindMemoryInfoPtr info) {
    bind_image_info = std::move(info);
  }

 private:
  const VkImage vk_image;
  CreateImageInfoPtr create_image_info;
  BindMemoryInfoPtr bind_image_info;
};
using ImagePtr = std::unique_ptr<Image>;
using ImageMap = std::unordered_map<VkImage, ImagePtr>;
using ImageMapInvalid = std::unordered_map<UniqueHandle, ImagePtr>;

class DeviceMemory {
 public:
  DeviceMemory(VkDeviceMemory, VkMemoryAllocateInfo const*);
  VulkanMemoryEventPtr GetVulkanMemoryEvent();
  VkDeviceMemory GetVkHandle() { return memory; }
  UniqueHandle GetUniqueHandle() { return unique_handle; }
  uint32_t GetMemoryType() { return memory_type; }

  void ClearBoundImages() { bound_images.clear(); }
  void EmplaceBoundImage(VkImage image) { bound_images.emplace(image); }
  void EraseBoundImage(VkImage image) { bound_images.erase(image); }

  void ClearBoundBuffers() { bound_buffers.clear(); }
  void EmplaceBoundBuffer(VkBuffer buffer) { bound_buffers.emplace(buffer); }
  void EraseBoundBuffer(VkBuffer buffer) { bound_buffers.erase(buffer); }

  void EmplaceInvalidImage(UniqueHandle handle) {
    invalid_images.emplace(handle);
  }
  void EmplaceInvalidBuffer(UniqueHandle handle) {
    invalid_buffers.emplace(handle);
  }

  const std::unordered_set<VkImage>& GetBoundImages() { return bound_images; }
  const std::unordered_set<VkBuffer>& GetBoundBuffers() {
    return bound_buffers;
  }

 private:
  uint64_t timestamp;
  const VkDeviceMemory memory;
  VkDeviceSize allocation_size;
  uint32_t memory_type;
  UniqueHandle unique_handle;
  // We need this to invalidate the bound images and buffers when the device
  // memory is destroyed.
  std::unordered_set<VkImage> bound_images;
  std::unordered_set<VkBuffer> bound_buffers;
  // After invalidation, we still keep the unique handles stored to enable
  // future querying of the bound images and buffers.
  std::unordered_set<UniqueHandle> invalid_images;
  std::unordered_set<UniqueHandle> invalid_buffers;
};
using DeviceMemoryPtr = std::unique_ptr<DeviceMemory>;
using DeviceMemoryMap = std::unordered_map<VkDeviceMemory, DeviceMemoryPtr>;
using DeviceMemoryMapInvalid =
    std::unordered_map<UniqueHandle, DeviceMemoryPtr>;

class Heap {
 public:
  Heap(VkDeviceSize, VkMemoryHeapFlags);
  void AddDeviceMemory(DeviceMemoryPtr);
  void DestroyDeviceMemory(VkDeviceMemory);
  void BindBuffer(BufferPtr, VkDeviceMemory, VkDeviceSize);
  void DestroyBuffer(VkBuffer);
  void BindImage(ImagePtr, VkDeviceMemory, VkDeviceSize);
  void DestroyImage(VkImage);
  VkDeviceSize GetSize() { return size; }
  VkMemoryHeapFlags GetFlags() { return flags; }
  VulkanMemoryEventContainerPtr GetVulkanMemoryEvents(
      VkDevice, uint32_t /* heap_index */);

 private:
  const VkDeviceSize size;
  const VkMemoryHeapFlags flags;

  BufferMap buffers;  // bound buffers
  ImageMap images;    // bound images
  DeviceMemoryMap device_memories;

  rwlock rwl_buffers;
  rwlock rwl_images;
  rwlock rwl_device_memories;

  // Keeping track of images and buffers if the device memory is deleted without
  // correctly destroying the bound images and buffers beforehand.
  BufferMapInvalid invalid_buffers;
  ImageMapInvalid invalid_images;
  DeviceMemoryMapInvalid invalid_device_memories;

  rwlock rwl_invalid_buffers;
  rwlock rwl_invalid_images;
  rwlock rwl_invalid_device_memories;
};
using HeapPtr = std::unique_ptr<Heap>;
using HeapMap = std::unordered_map<uint32_t /* heap_index */, HeapPtr>;

using DeviceMemorySet = std::unique_ptr<std::unordered_set<VkDeviceMemory>>;
using DeviceMemorySetMap = std::unordered_map<VkDevice, DeviceMemorySet>;

class PhysicalDevice {
 public:
  PhysicalDevice(VkPhysicalDevice);
  VkPhysicalDevice GetVkPhysicalDevice() { return physical_device; }
  void AddDeviceMemory(VkDevice, DeviceMemoryPtr);
  void BindBuffer(BufferPtr, VkDeviceMemory, VkDeviceSize);
  void BindImage(ImagePtr, VkDeviceMemory, VkDeviceSize);

  void DestroyDeviceMemory(VkDevice, VkDeviceMemory, bool);
  void DestroyAllDeviceMemories(VkDevice);
  void DestroyBuffer(VkBuffer);
  void DestroyImage(VkImage);
  uint32_t GetHeapIndex(uint32_t memory_type) {
    return memory_type_index_to_heap_index[memory_type];
  }
  std::deque<uint32_t> GetHeapIndexMap() {
    return memory_type_index_to_heap_index;
  }

  VulkanMemoryEventPtr GetVulkanMemoryEvent(VkDevice);
  VulkanMemoryEventContainerSetPtr GetVulkanMemoryEventsForHeaps(VkDevice);

 private:
  uint64_t timestamp;
  const VkPhysicalDevice physical_device;
  VkPhysicalDeviceMemoryProperties memory_properties;
  std::deque<uint32_t> memory_type_index_to_heap_index;
  rwlock rwl_heaps;
  HeapMap heaps;

  std::unordered_map<VkBuffer, uint32_t> buffer_to_heap_index;
  std::unordered_map<VkImage, uint32_t> image_to_heap_index;
  std::unordered_map<VkDeviceMemory, uint32_t> device_memory_to_heap_index;
  DeviceMemorySetMap device_to_device_memory_set;

  rwlock rwl_buffer_to_heap_index;
  rwlock rwl_image_to_heap_index;
  rwlock rwl_device_memory_to_heap_index;
  rwlock rwl_device_to_device_memory_set;
};
using PhysicalDevicePtr = std::shared_ptr<PhysicalDevice>;
using PhysicalDeviceMap =
    std::unordered_map<VkPhysicalDevice, PhysicalDevicePtr>;

class Device {
 public:
  Device(VkDevice, PhysicalDevicePtr);
  void AddDeviceMemory(DeviceMemoryPtr);
  void DestroyDeviceMemory(VkDeviceMemory);
  void DestroyAllDeviceMemories();
  void AddBuffer(BufferPtr);
  void BindBuffer(VkBuffer, VkDeviceMemory, VkDeviceSize);
  void DestroyBuffer(VkBuffer);
  void AddImage(ImagePtr);
  void BindImage(VkImage, VkDeviceMemory, VkDeviceSize);
  void DestroyImage(VkImage);
  uint32_t GetHeapIndex(uint32_t /*memory_type*/);
  VulkanMemoryEventContainerSetPtr GetVulkanMemoryEvents();
  VkPhysicalDevice GetVkPhysicalDevice() {
    return physical_device->GetVkPhysicalDevice();
  }

 private:
  uint64_t timestamp;
  const VkDevice device;
  PhysicalDevicePtr physical_device;
  BufferMap buffers;  // unbound buffers
  ImageMap images;    // unbound images

  rwlock rwl_buffers;
  rwlock rwl_images;

  void SendMemoryEventToTraceDaemon(VulkanMemoryEventPtr);
};
using DevicePtr = std::unique_ptr<Device>;
using DeviceMap = std::unordered_map<VkDevice, DevicePtr>;

// ----------------------- Wrapping allocation callbacks -----------------------

enum AllocatorType { atDefault, atUser };
using AllocationCallbacksHandle = uint64_t;

class AllocationCallbacksTracker {
 private:
  VkAllocationCallbacks tracked_allocator = {};

  // Since function pointers must be static members of the class, we need to
  // keep all the required information in static map objects. For now, we store
  // the following data items:
  // - user provided allocation callbacks
  // - user provided data (pUserData parameter)
  // - name of the Vulkan function that requested the memory
  // - size of the allocations, to be used in non-WIN32 code for realloc

  static rwlock rwl_global_callback_mapping;
  static std::unordered_map<uintptr_t, const VkAllocationCallbacks*>
      global_callback_mapping;

  static rwlock rwl_global_user_data_mapping;
  static std::unordered_map<uintptr_t, uintptr_t> global_user_data_mapping;

  static rwlock rwl_global_caller_api_mapping;
  static std::unordered_map<uintptr_t, std::string> global_caller_api_mapping;

#if !defined(WIN32)
  static rwlock rwl_global_allocation_size_mapping;
  static std::unordered_map<uintptr_t, size_t> global_allocation_size_mapping;
#endif

  // Static tracked allocation functions. We don't care about internal
  // allocation and free notification functions.
  static void* TrackedAllocationFunction(void*, size_t, size_t,
                                         VkSystemAllocationScope) VKAPI_ATTR;
  static void* TrackedReallocationFunction(void*, void*, size_t, size_t,
                                           VkSystemAllocationScope) VKAPI_ATTR;
  static void TrackedFreeFunction(void*, void*) VKAPI_ATTR;

 public:
  AllocationCallbacksTracker(const VkAllocationCallbacks*, const std::string&);
  const VkAllocationCallbacks* TrackedAllocator() { return &tracked_allocator; }
  static AllocationCallbacksHandle GetAllocationCallbacksHandle(
      const VkAllocationCallbacks* allocator, const std::string& caller);
};
using AllocationCallbacksTrackerPtr =
    std::unique_ptr<AllocationCallbacksTracker>;
using AllocationCallbacksTrackerMap =
    std::unordered_map<AllocationCallbacksHandle,
                       AllocationCallbacksTrackerPtr>;

// -------------------------- Host allocation classes --------------------------

class HostAllocation {
 public:
  HostAllocation(uint64_t timestamp_, uintptr_t ptr_, size_t size_,
                 size_t alignment_, VkSystemAllocationScope scope_,
                 const std::string& caller_api_, AllocatorType allocator_type_)
      : timestamp(timestamp_),
        ptr(ptr_),
        size(size_),
        alignment(alignment_),
        scope(scope_),
        caller_api(caller_api_),
        allocator_type(allocator_type_){};
  VulkanMemoryEventPtr GetVulkanMemoryEvent();

 private:
  const uint64_t timestamp;
  const uintptr_t ptr;
  const size_t size;
  const size_t alignment;
  const VkSystemAllocationScope scope;
  const std::string caller_api;
  const AllocatorType allocator_type;
};
using HostAllocationPtr = std::unique_ptr<HostAllocation>;
// At the moment, we don't care about the allocation history of an object if
// the memory is reallocated. The allocation history can be easily added by
// using a map of a container of HostAllocation and adding the proper logic to
// respective functions in class MemoryTracker.
using HostAllocationMap = std::unordered_map<uintptr_t, HostAllocationPtr>;

// --------------------------- Memory events tracker ---------------------------

class MemoryTracker {
 public:
  MemoryTracker();

  const VkAllocationCallbacks* GetTrackedAllocator(const VkAllocationCallbacks*,
                                                   const std::string& caller);

  // Process the event. We might store the event in the memory state or send it
  // to the trace daemon if the trace is already started.
  void ProcessCreateDeviceEvent(VkPhysicalDevice physical_device,
                                VkDeviceCreateInfo const* create_info,
                                VkDevice device);
  void ProcessDestoryDeviceEvent(VkDevice device);
  void ProcessAllocateMemoryEvent(VkDevice device, VkDeviceMemory memory,
                                  VkMemoryAllocateInfo const* allocate_info);
  void ProcessFreeMemoryEvent(VkDevice device, VkDeviceMemory memory);
  void ProcessCreateBufferEvent(VkDevice device, VkBuffer buffer,
                                const VkBufferCreateInfo* create_info);
  void ProcessBindBufferEvent(VkDevice device, VkBuffer buffer,
                              VkDeviceMemory memory, size_t offset);
  void ProcessDestroyBufferEvent(VkDevice device, VkBuffer buffer);
  void ProcessCreateImageEvent(VkDevice device, VkImage image,
                               const VkImageCreateInfo* create_info);
  void ProcessBindImageEvent(VkDevice device, VkImage image,
                             VkDeviceMemory memory, size_t offset);
  void ProcessDestroyImageEvent(VkDevice device, VkImage image);

  void ProcessHostMemoryAllocationEvent(uintptr_t ptr, size_t size,
                                        size_t alignment,
                                        VkSystemAllocationScope scope,
                                        const std::string& caller_api,
                                        AllocatorType allocator_type);
  void ProcessHostMemoryReallocationEvent(uintptr_t ptr, uintptr_t original,
                                          size_t size, size_t alignment,
                                          VkSystemAllocationScope scope,
                                          const std::string& caller_api,
                                          AllocatorType allocator_type);
  void ProcessHostMemoryFreeEvent(uintptr_t ptr);

 private:
  rwlock rwl_devices;
  DeviceMap devices;

  rwlock rwl_allocation_trackers;
  AllocationCallbacksTrackerMap m_allocation_callbacks_trackers;

  rwlock rwl_host_allocations;
  HostAllocationMap host_allocations;

  rwlock rwl_physical_devices;
  PhysicalDeviceMap physical_devices;

  bool initial_state_is_sent_;

  rwlock rwl_device_memory_type_map;
  std::unordered_map<VkDeviceMemory, uint32_t> device_memory_type_map;

  // Add the event to the current state of the memory usage.
  void StoreCreateDeviceEvent(VkPhysicalDevice physical_device,
                              VkDeviceCreateInfo const* create_info,
                              VkDevice device);
  void StoreDestoryDeviceEvent(VkDevice device);
  void StoreAllocateMemoryEvent(VkDevice device, VkDeviceMemory memory,
                                VkMemoryAllocateInfo const* allocate_info);
  void StoreFreeMemoryEvent(VkDevice device, VkDeviceMemory memory);
  void StoreCreateBufferEvent(VkDevice device, VkBuffer buffer,
                              const VkBufferCreateInfo* create_info);
  void StoreBindBufferEvent(VkDevice device, VkBuffer buffer,
                            VkDeviceMemory memory, size_t offset);
  void StoreDestroyBufferEvent(VkDevice device, VkBuffer buffer);
  void StoreCreateImageEvent(VkDevice device, VkImage image,
                             const VkImageCreateInfo* create_info);
  void StoreBindImageEvent(VkDevice device, VkImage image,
                           VkDeviceMemory memory, size_t offset);
  void StoreDestroyImageEvent(VkDevice device, VkImage image);

  void StoreHostMemoryAllocationEvent(uintptr_t ptr, size_t size,
                                      size_t alignment,
                                      VkSystemAllocationScope scope,
                                      const std::string& caller_api,
                                      AllocatorType allocator_type);
  void StoreHostMemoryReallocationEvent(uintptr_t ptr, uintptr_t original,
                                        size_t size, size_t alignment,
                                        VkSystemAllocationScope scope,
                                        const std::string& caller_api,
                                        AllocatorType allocator_type);
  void StoreHostMemoryFreeEvent(uintptr_t ptr);

  void EmitAndClearAllStoredEvents();
  void EmitAllStoredEventsIfNecessary();

  // Send the event directly to the trace daemon.
  void EmitCreateDeviceEvent(VkPhysicalDevice physical_device,
                             VkDeviceCreateInfo const* create_info,
                             VkDevice device);
  void EmitDestoryDeviceEvent(VkDevice device);
  void EmitAllocateMemoryEvent(VkDevice device, VkDeviceMemory memory,
                               VkMemoryAllocateInfo const* allocate_info);
  void EmitFreeMemoryEvent(VkDevice device, VkDeviceMemory memory);
  void EmitCreateBufferEvent(VkDevice device, VkBuffer buffer,
                             const VkBufferCreateInfo* create_info);
  void EmitBindBufferEvent(VkDevice device, VkBuffer buffer,
                           VkDeviceMemory memory, size_t offset);
  void EmitDestroyBufferEvent(VkDevice device, VkBuffer buffer);
  void EmitCreateImageEvent(VkDevice device, VkImage image,
                            const VkImageCreateInfo* create_info);
  void EmitBindImageEvent(VkDevice device, VkImage image, VkDeviceMemory memory,
                          size_t offset);
  void EmitDestroyImageEvent(VkDevice device, VkImage image);

  void EmitHostMemoryAllocationEvent(uintptr_t ptr, size_t size,
                                     size_t alignment,
                                     VkSystemAllocationScope scope,
                                     const std::string& caller_api,
                                     AllocatorType allocator_type);
  void EmitHostMemoryReallocationEvent(uintptr_t ptr, uintptr_t original,
                                       size_t size, size_t alignment,
                                       VkSystemAllocationScope scope,
                                       const std::string& caller_api,
                                       AllocatorType allocator_type);
  void EmitHostMemoryFreeEvent(uintptr_t ptr);
};

// -----------------------------------------------------------------------------

extern MemoryTracker memory_tracker_instance;

}  // namespace memory_tracker

#endif  // MEMORY_TRACKER_LAYER_IMPL_H