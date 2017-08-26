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

#ifndef VK_VIRTUAL_SWAPCHAIN_VIRTUAL_SWAPCHAIN_H_
#define VK_VIRTUAL_SWAPCHAIN_VIRTUAL_SWAPCHAIN_H_

#include "layer.h"
#include <atomic>
#include <deque>
#include <mutex>
#include <functional>
#include <vulkan/vulkan.h>

namespace swapchain {

// The VirtualSwapchain is the bulk of the data for handling
// all of the images/synchronization/buffers for our swapchain.
class VirtualSwapchain {
public:
  // pending_image_timeout_in_milliseconds_ can be configured based on your
  // application. By default it is 10ms. This tells the secondary thread
  // how long it should wait if no image has been submitted to see if
  // it should shut down. Increasing this number will mean that the
  // secondary thread will wake up less frequently un-necessarily, at the
  // expense of a longer stall on shutdown.
  VirtualSwapchain(VkDevice device, uint32_t queue,
                   const VkPhysicalDeviceProperties *pProperties,
                   const VkPhysicalDeviceMemoryProperties *memory_properties,
                   const DeviceData *functions,
                   const VkSwapchainCreateInfoKHR *_swapchain_info,
                   const VkAllocationCallbacks *pAllocator,
                   uint32_t pending_image_timeout_in_milliseconds = 10,
                   bool always_get_acquired_image = false);
  // Call this to release all of the resources associated with this object.
  void Destroy(const VkAllocationCallbacks *pAllocator);
  // Sets the function to be called when a frame has completed, along with
  // a piece of user-data to be passed.
  void SetCallback(void callback(void *, uint8_t *, size_t), void *);
  // Returns in *image the index of the next free image. Returns false
  // if timeout nanoseconds have passed and no image could be returned.
  // If timeout is UINT64_MAX, then this function will wait forever.
  bool GetImage(uint64_t timeout, uint32_t *image);
  // Returns a vector of all of the images contained in this swapchain.
  std::vector<VkImage> GetImages(uint32_t num_images, bool create_new_images) {
    std::unique_lock<threading::mutex> sl(free_images_lock_);
    while (num_images > num_images_ && create_new_images) {
      image_data_.push_back(build_swapchain_image_data_());
      free_images_.push_back(num_images_);
      free_images_condition_.notify_all();
      num_images_++;
    }
    std::vector<VkImage> image_vec;
    image_vec.reserve(num_images_);
    for (const auto &data : image_data_) {
      image_vec.push_back(data.image_);
    }
    return image_vec;
  }

  // Returns the queue index that this swapchain was created with.
  uint32_t DeviceQueue() { return queue_; }

  // Returns the VkFence associated with the i'th image.
  VkFence GetFence(size_t i) { return image_data_[i].fence_; }
  // Returns the VkCommandBuffer with the i'th image.
  VkCommandBuffer &GetCommandBuffer(size_t i) {
    return image_data_[i].command_buffer_;
  }
  // When the commands associated with an image have been submitted to
  // a VkQueue, NotifySubmitted must be called to inform the swapchain
  // that the image in question is no longer needed.
  void NotifySubmitted(size_t i) {
    {
      std::lock_guard<threading::mutex> lock(pending_images_lock_);
      pending_images_.push_back(static_cast<uint32_t>(i));
    }
    pending_images_condition_.notify_one();
  }

  // Sets the flag to control the behavior of GetImage(). When true, the
  // virtual swapchain will always wait for the acquired image and always get
  // the acquired image. When false, the virtual swapchain will act like a
  // normal swapchain, which will randomly get a free image and write its index
  // to the given index pointer.
  void SetAlwaysGetAcquiredImage(bool always_get_acquired_image) {
    always_get_acquired_image_ = always_get_acquired_image;
  }

private:
  const VkSwapchainCreateInfoKHR swapchain_info_;
  // This is the entry-point to our secondary thread.
  // It is responsible for keeping track of copies, and calling the
  // callback when a copy has completed.
  void CopyThreadFunc();
  // Returns the size of the image in bytes.
  uint32_t ImageByteSize() const;
  // All of the data associated with a single swapchain VkImage.
  struct SwapchainImageData {
    VkImage image_;               // The image itself.
    VkDeviceMemory image_memory_; // The device memory allocated to this image.

    VkBuffer buffer_; // The buffer to copy the image contents into.
    VkDeviceMemory buffer_memory_; // The memory for the buffer.

    VkFence fence_; // The fence to signal when the copy is complete.
    VkCommandBuffer
        command_buffer_; // The command_buffer that contains the copy commands.
  };

  // In our constructor we rely on num_images_ being
  // initialized first, so don't move anything above it.
  uint32_t num_images_; // The number of images requested.

  uint32_t width_;  // The width of our swapchain.
  uint32_t height_; // The height of our swapchain.
  std::deque<SwapchainImageData>
      image_data_; // All of the data for each requested swapchain image.
  std::deque<uint32_t>
      pending_images_; // Indices into image_data_ for all images that
                       // have been submitted but not processed yet.
  std::deque<uint32_t>
      free_images_; // Indices into image_data_ for all images that
                    // are not currently in use.
  VkDevice device_; // The device that this swapchain belongs to.
  VkCommandPool
      command_pool_; // The command_pool that we are allocating buffers from.

  // If should_close_ == true then the next time we wake up we should
  // terminate our thread.
  std::atomic<bool> should_close_;

// Some versions of the STL do not handle std::thread correctly,
// use pthread/win thread instead.
#ifdef _WIN32
  HANDLE thread_;
#else
  pthread_t thread_;
#endif

  threading::condition_variable pending_images_condition_; // Condition variable
                                                           // to wait for
                                                           // pending_images_ to
                                                           // contain an image.
  threading::mutex
      pending_images_lock_; // The lock for modifying our pending images list.

  threading::condition_variable free_images_condition_; // The condition
                                                        // variable to wait on
                                                        // for free_images_ to
                                                        // contain an image.

  threading::mutex
      free_images_lock_; // The lock for modifying our free images list.

  void (*callback_)(void *, uint8_t *, size_t); // The user-supplied callback.
  void *callback_user_data_; // The user-data to pass to this callback.

  const uint32_t queue_; // the queue that we need to use to signal things
  const DeviceData *
      functions_; // All of the resolved function pointers that we need to call.

  // This is how many milliseconds we should wait for an image before waking up
  // and seeing if we should shut down.
  const uint32_t pending_image_timeout_in_milliseconds_;
  // A flag to indicate whether GetImage() always wasit for the acquired image
  // specified with the value pointed by the index pointer.  When set to true,
  // GetImage() will wait until the acquired image is ready to use. When set to
  // false, GetImage() will write the index of a randomly free image to the
  // given index pointer.
  bool always_get_acquired_image_;
  // Function to build swapchain images
  std::function<SwapchainImageData()> build_swapchain_image_data_;
};
} // swapchain

#endif //  VK_VIRTUAL_SWAPCHAIN_VIRTUAL_SWAPCHAIN_H_
