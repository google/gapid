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

#ifndef VK_VIRTUAL_SWAPCHAIN_LAYER_H_
#define VK_VIRTUAL_SWAPCHAIN_LAYER_H_

#include <unordered_map>
#include <vector>

#include "vulkan/vulkan.h"

#include "threading.h"

#define EXPECT_SUCCESS(fn)                                                  \
  [&]() {                                                                   \
    auto r = fn;                                                            \
    if (VK_SUCCESS != r) {                                                  \
      swapchain::write_warning(__FILE__ ":" + std::to_string(__LINE__) +    \
                               ": " #fn " RETURNED: " + std::to_string(r)); \
    }                                                                       \
    return r;                                                               \
  }()

namespace swapchain {

void write_warning(const char* message);

void write_warning(const std::string& message);

// Sets the key of the dispatch tables used in lower layers of the parent
// dispatchable handle to the new child dispatchable handle. This is necessary
// as lower layers may use that key to find the dispatch table, and a child
// handle should share the same dispatch table key. E.g. VkCommandBuffer is a
// child dispatchable handle of VkDevice, all the VkCommandBuffer dispatching
// functions are actually device functions (resolved by VkGetDeviceProcAddress).
// Ref:
// https://github.com/KhronosGroup/Vulkan-LoaderAndValidationLayers/blob/master/loader/LoaderAndLayerInterface.md#creating-new-dispatchable-objects,
static inline void set_dispatch_from_parent(void* child, void* parent) {
  *((const void**)child) = *((const void**)parent);
}

// All of the instance data that is needed for book-keeping in a layer.
struct InstanceData {
  PFN_vkGetInstanceProcAddr vkGetInstanceProcAddr;
  PFN_vkDestroyInstance vkDestroyInstance;
  PFN_vkEnumeratePhysicalDevices vkEnumeratePhysicalDevices;
  PFN_vkEnumerateDeviceExtensionProperties vkEnumerateDeviceExtensionProperties;
  PFN_vkCreateDevice vkCreateDevice;
  PFN_vkGetPhysicalDeviceQueueFamilyProperties
      vkGetPhysicalDeviceQueueFamilyProperties;
  PFN_vkGetPhysicalDeviceProperties vkGetPhysicalDeviceProperties;
  PFN_vkGetPhysicalDeviceMemoryProperties vkGetPhysicalDeviceMemoryProperties;

#ifdef VK_USE_PLATFORM_ANDROID_KHR
  PFN_vkCreateAndroidSurfaceKHR vkCreateAndroidSurfaceKHR;
#endif
#ifdef VK_USE_PLATFORM_XCB_KHR
  PFN_vkCreateXcbSurfaceKHR vkCreateXcbSurfaceKHR;
#endif
#ifdef VK_USE_PLATFORM_WIN32_KHR
  PFN_vkCreateWin32SurfaceKHR vkCreateWin32SurfaceKHR;
#endif
  PFN_vkDestroySurfaceKHR vkDestroySurfaceKHR;

  // All of the physical devices associated with this instance.
  std::vector<VkPhysicalDevice> physical_devices_;
};

// All of the command buffer data that is needed for book-keeping in our layer.
struct CommandBufferData {
  VkDevice device_;
  PFN_vkCmdPipelineBarrier vkCmdPipelineBarrier;
  PFN_vkCmdWaitEvents vkCmdWaitEvents;
};

// All of the physical device data needed for book-keeping in our layer.
struct PhysicalDeviceData {
  // The instance that this physical device belongs to.
  VkInstance instance_;
  VkPhysicalDeviceMemoryProperties memory_properties_;
  VkPhysicalDeviceProperties physical_device_properties_;
};

// All of the device data we need for book-keeping.
struct DeviceData {
  VkPhysicalDevice physicalDevice;

  PFN_vkGetDeviceProcAddr vkGetDeviceProcAddr;
  PFN_vkGetDeviceQueue vkGetDeviceQueue;

  PFN_vkAllocateMemory vkAllocateMemory;
  PFN_vkFreeMemory vkFreeMemory;
  PFN_vkMapMemory vkMapMemory;
  PFN_vkUnmapMemory vkUnmapMemory;
  PFN_vkInvalidateMappedMemoryRanges vkInvalidateMappedMemoryRanges;

  PFN_vkCreateSemaphore vkCreateSemaphore;
  PFN_vkDestroySemaphore vkDestroySemaphore;
  PFN_vkCreateFence vkCreateFence;
  PFN_vkGetFenceStatus vkGetFenceStatus;
  PFN_vkWaitForFences vkWaitForFences;
  PFN_vkDestroyFence vkDestroyFence;
  PFN_vkResetFences vkResetFences;

  PFN_vkCreateImage vkCreateImage;
  PFN_vkGetImageMemoryRequirements vkGetImageMemoryRequirements;
  PFN_vkBindImageMemory vkBindImageMemory;
  PFN_vkDestroyImage vkDestroyImage;

  PFN_vkCreateBuffer vkCreateBuffer;
  PFN_vkGetBufferMemoryRequirements vkGetBufferMemoryRequirements;
  PFN_vkBindBufferMemory vkBindBufferMemory;
  PFN_vkDestroyBuffer vkDestroyBuffer;

  PFN_vkCreateCommandPool vkCreateCommandPool;
  PFN_vkDestroyCommandPool vkDestroyCommandPool;
  PFN_vkAllocateCommandBuffers vkAllocateCommandBuffers;
  PFN_vkFreeCommandBuffers vkFreeCommandBuffers;

  PFN_vkBeginCommandBuffer vkBeginCommandBuffer;
  PFN_vkEndCommandBuffer vkEndCommandBuffer;
  PFN_vkResetCommandBuffer vkResetCommandBuffer;

  PFN_vkCmdCopyImageToBuffer vkCmdCopyImageToBuffer;
  PFN_vkCmdBlitImage vkCmdBlitImage;
  PFN_vkCmdPipelineBarrier vkCmdPipelineBarrier;
  PFN_vkCmdWaitEvents vkCmdWaitEvents;
  PFN_vkCreateRenderPass vkCreateRenderPass;

  PFN_vkQueueSubmit vkQueueSubmit;
  PFN_vkQueuePresentKHR vkQueuePresentKHR;
  PFN_vkDestroyDevice vkDestroyDevice;

  PFN_vkCreateSwapchainKHR vkCreateSwapchainKHR;
  PFN_vkGetSwapchainImagesKHR vkGetSwapchainImagesKHR;
  PFN_vkAcquireNextImageKHR vkAcquireNextImageKHR;
  PFN_vkAcquireNextImage2KHR vkAcquireNextImage2KHR;
  PFN_vkDestroySwapchainKHR vkDestroySwapchainKHR;
};

struct QueueData {
  VkDevice device_;
  PFN_vkQueueSubmit vkQueueSubmit;
  PFN_vkQueuePresentKHR vkQueuePresentKHR;
};

// All context functions return a context token.
// Any data within a ContextToken is only valid
// for the lifetime of the ContextToken.
template <typename T>
struct ContextToken {
  ContextToken(T& object, threading::mutex& locker)
      : object_(object), context_lock_(locker) {}

  ContextToken(T& object, std::unique_lock<threading::mutex>&& locker)
      : object_(object), context_lock_(std::move(locker)) {}

  ContextToken(ContextToken&& _other)
      : object_(_other.object_),
        context_lock_(std::move(_other.context_lock_)) {}

  ContextToken(const ContextToken& _other) = delete;
  ContextToken& operator=(const ContextToken& _other) = delete;

  const T* operator->() const { return &object_; }
  const T& operator*() const { return object_; }
  T* operator->() { return &object_; }
  T& operator*() { return object_; }

 private:
  T& object_;
  std::unique_lock<threading::mutex> context_lock_;
};

// In order to prevent dead-locks in the presence of
// needing multiple maps/objects, All of these
// should always be acquired in order from the most
// specific to the least specific.
// CommandBuffer->Queue->Device->PhysicalDevice->Instance
// Is is valid to acquire only a subset (Queue->PhysicalDevice),
// but never valid to acquire them in the reverse order.
struct Context {
  using InstanceMap = std::unordered_map<VkInstance, InstanceData>;
  using CommandBufferMap =
      std::unordered_map<VkCommandBuffer, CommandBufferData>;
  using PhysicalDeviceMap =
      std::unordered_map<VkPhysicalDevice, PhysicalDeviceData>;
  using QueueMap = std::unordered_map<VkQueue, QueueData>;
  using DeviceMap = std::unordered_map<VkDevice, DeviceData>;

  ContextToken<InstanceMap> GetInstanceMap() {
    return ContextToken<InstanceMap>(instance_data_map_, instance_lock_);
  }

  ContextToken<CommandBufferMap> GetCommandBufferMap() {
    return ContextToken<CommandBufferMap>(command_buffer_data_map_,
                                          command_buffer_lock_);
  }

  ContextToken<QueueMap> GetQueueMap() {
    return ContextToken<QueueMap>(queue_data_map_, queue_lock_);
  }

  ContextToken<PhysicalDeviceMap> GetPhysicalDeviceMap() {
    return ContextToken<PhysicalDeviceMap>(physical_device_data_map_,
                                           physical_device_lock_);
  }

  ContextToken<DeviceMap> GetDeviceMap() {
    return ContextToken<DeviceMap>(device_data_map_, device_lock_);
  }

  ContextToken<InstanceData> GetInstanceData(VkInstance instance) {
    std::unique_lock<threading::mutex> locker(instance_lock_);
    return ContextToken<InstanceData>(instance_data_map_.at(instance),
                                      std::move(locker));
  }

  ContextToken<CommandBufferData> GetCommandBufferData(VkCommandBuffer buffer) {
    std::unique_lock<threading::mutex> locker(command_buffer_lock_);
    return ContextToken<CommandBufferData>(command_buffer_data_map_.at(buffer),
                                           std::move(locker));
  }

  ContextToken<QueueData> GetQueueData(VkQueue queue) {
    std::unique_lock<threading::mutex> locker(queue_lock_);
    return ContextToken<QueueData>(queue_data_map_.at(queue),
                                   std::move(locker));
  }

  ContextToken<PhysicalDeviceData> GetPhysicalDeviceData(
      VkPhysicalDevice physical_device) {
    std::unique_lock<threading::mutex> locker(physical_device_lock_);
    return ContextToken<PhysicalDeviceData>(
        physical_device_data_map_.at(physical_device), std::move(locker));
  }

  ContextToken<DeviceData> GetDeviceData(VkDevice device) {
    std::unique_lock<threading::mutex> locker(device_lock_);
    return ContextToken<DeviceData>(device_data_map_.at(device),
                                    std::move(locker));
  }

 private:
  // Map of instances to their data. Our other option would be
  // to wrap the instance object, but then we have to handle every possible
  // instance function.
  InstanceMap instance_data_map_;
  // Lock for use when reading/writing from kInstanceDataMap.
  threading::mutex instance_lock_;

  // The global map of command buffers to their data.
  CommandBufferMap command_buffer_data_map_;
  // Lock for use when reading/writing from kCommandBufferDataMap.
  threading::mutex command_buffer_lock_;

  // The global map of physical devices to their data.
  // This should be locked along with the related instance.

  PhysicalDeviceMap physical_device_data_map_;
  threading::mutex physical_device_lock_;

  // A map from queues to their devices.
  QueueMap queue_data_map_;
  // A lock around queue operations. This is needed in the virtual swapchain
  // because we have to insert into a queue, but cannot guarantee that
  // another application operation is not submitting to the queue. In this case
  // we lock around all queue operations.
  threading::mutex queue_lock_;

  // The global map of devices to their data.
  DeviceMap device_data_map_;

  // Lock for use when reading/writing from kDeviceDataMap.
  threading::mutex device_lock_;
};

Context& GetGlobalContext();

bool GetParameter(const char* env_var_name, const char* android_prop_name,
                  std::string* param_value);

}  // namespace swapchain

#endif  // VK_VIRTUAL_SWAPCHAIN_LAYER_H
