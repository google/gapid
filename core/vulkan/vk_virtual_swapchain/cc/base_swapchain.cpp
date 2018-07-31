/*
 * Copyright (C) 2018 Google Inc.
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

#include "platform.h"

#include "base_swapchain.h"

namespace swapchain {

BaseSwapchain::BaseSwapchain(VkInstance instance, VkDevice device, uint32_t queue,
                             VkCommandPool command_pool, uint32_t num_images,
                             const InstanceData *instance_functions,
                             const DeviceData *device_functions,
                             const VkSwapchainCreateInfoKHR *swapchain_info,
                             const VkAllocationCallbacks *pAllocator,
                             const void *platform_info)
    : instance_(instance),
      device_(device),
      instance_functions_(instance_functions),
      device_functions_(device_functions),
      swapchain_info_(*swapchain_info)
{

  if (platform_info == nullptr) {
    return;
  }

  CreateSurface(instance_functions, instance, platform_info, pAllocator, &surface_);
  if (surface_ == 0) {
    // Surface creation failed
    return;
  }

  {
    // Create the swapchain
    VkSwapchainCreateInfoKHR createInfo = {
      VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR, // sType
      nullptr, // pNext
      0, // flags
      surface_, // surface
      num_images, // minImageCount
      swapchain_info_.imageFormat,  // imageFormat
      swapchain_info_.imageColorSpace, // imageColorSpace
      swapchain_info_.imageExtent, // imageExtent
      swapchain_info_.imageArrayLayers, // arrayLayers
      VK_IMAGE_USAGE_TRANSFER_DST_BIT, // imageUsage
      VK_SHARING_MODE_EXCLUSIVE, // imageSharingMode,
      0, // queueFamilyIndexCount
      nullptr, // pQueueFamilyIndices
      swapchain_info_.preTransform, // preTransform
      swapchain_info_.compositeAlpha, // compositeAlpha
      VK_PRESENT_MODE_FIFO_KHR, // presentMode
      VK_TRUE, // clipped
      0, // oldSwapchain
    };
    if (device_functions_->vkCreateSwapchainKHR(device_, &createInfo, pAllocator, &swapchain_) != VK_SUCCESS) {
      // Creating swapchain failed
      swapchain_ = 0;
      return;
    }
  }

  uint32_t numImages = 0;
  device_functions_->vkGetSwapchainImagesKHR(device, swapchain_, &numImages, nullptr);
  images_.resize(numImages);
  device_functions_->vkGetSwapchainImagesKHR(device, swapchain_, &numImages, images_.data());

  VkCommandBufferAllocateInfo command_buffer_info = {
    VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
    nullptr, // pNext
    command_pool, // commandPool
    VK_COMMAND_BUFFER_LEVEL_PRIMARY, // level
    1, // commandBufferCount
  };
  device_functions_->vkAllocateCommandBuffers(device, &command_buffer_info, &command_buffer_);
  set_dispatch_from_parent(command_buffer_, device);

  {
    VkSemaphoreCreateInfo createInfo = {
      VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO, // sType
      nullptr, // pNext
      0, // flags
    };
    if (device_functions_->vkCreateSemaphore(device_, &createInfo, pAllocator,
                                             &acquire_semaphore_) != VK_SUCCESS) {
      return;
    }
    if (device_functions_->vkCreateSemaphore(device_, &createInfo, pAllocator,
                                             &blit_semaphore_) != VK_SUCCESS) {
      return;
    }
  }
  {
    VkFenceCreateInfo createInfo = {
      VK_STRUCTURE_TYPE_FENCE_CREATE_INFO, // sType
      nullptr, // pNext
      0, // flags
    };
    if (device_functions_->vkCreateFence(device_, &createInfo, pAllocator,
                                        &blit_fence_) != VK_SUCCESS) {
      return;
    }
  }
}

void BaseSwapchain::Destroy(const VkAllocationCallbacks *pAllocator) {
  device_functions_->vkDestroySemaphore(device_, acquire_semaphore_, pAllocator);
  acquire_semaphore_ = VK_NULL_HANDLE;

  device_functions_->vkDestroySemaphore(device_, blit_semaphore_, pAllocator);
  blit_semaphore_ = VK_NULL_HANDLE;

  device_functions_->vkDestroyFence(device_, blit_fence_, pAllocator);
  blit_fence_ = VK_NULL_HANDLE;

  device_functions_->vkDestroySwapchainKHR(device_, swapchain_, pAllocator);
  swapchain_ = VK_NULL_HANDLE;

  instance_functions_->vkDestroySurfaceKHR(instance_, surface_, pAllocator);
  surface_ = VK_NULL_HANDLE;
}

VkResult BaseSwapchain::PresentFrom(VkQueue queue, VkImage image) {
  std::unique_lock<threading::mutex> guard(present_lock_);
  VkResult res;
  uint32_t index = 0;
  // TODO: the error return values here aren't necessarily valid return values
  // for VkQueueSubmit
  if ((res = device_functions_->vkAcquireNextImageKHR(device_, swapchain_, 0, acquire_semaphore_,
                                   VK_NULL_HANDLE, &index)) != VK_SUCCESS) {
    return res;
  }

  if ((res = device_functions_->vkResetCommandBuffer(command_buffer_, 0)) != VK_SUCCESS) {
    return res;
  }
  VkCommandBufferBeginInfo beginInfo = {
    VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
    0, // pNext
    VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT, // flags
    nullptr, // pInheritanceInfo
  };
  if ((res = device_functions_->vkBeginCommandBuffer(command_buffer_, &beginInfo)) != VK_SUCCESS) {
    return res;
  }

  // The source image is already in VK_IMAGE_LAYOUT_TRANSFER_SRC, we need to
  // transition our image between VK_IMAGE_LAYOUT_TRANSFER_DST and
  // VK_IMAGE_LAYOUT_SHADER_PRESENT_KHR
  VkImageMemoryBarrier initialBarrier = {
    VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER, // sType
    nullptr, // pNext
    0, // srcAccessFlags
    VK_ACCESS_TRANSFER_WRITE_BIT, // dstAccessFlags
    VK_IMAGE_LAYOUT_PRESENT_SRC_KHR, // oldLayout
    VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, // newLayout
    VK_QUEUE_FAMILY_IGNORED, // srcQueueFamilyIndex
    VK_QUEUE_FAMILY_IGNORED, // dstQueueFamilyIndex
    images_[index], // image
    VkImageSubresourceRange {
      VK_IMAGE_ASPECT_COLOR_BIT, // aspectMask
      0, // baseMipLevel
      1, // levelCount
      0, // baseArrayLayer
      swapchain_info_.imageArrayLayers, // layerCount
    }, // subresourceRange
  };

  device_functions_->vkCmdPipelineBarrier(
      command_buffer_,
      0, // srcStageMask
      VK_PIPELINE_STAGE_TRANSFER_BIT, // dstStageMask
      0, // dependencyFlags
      0, // memoryBarrierCount
      nullptr, // pMemoryBarriers
      0, // bufferMemoryBarrierCount
      nullptr, // pBufferMemoryBarriers
      1, // imageMemoryBarrierCount
      &initialBarrier // pImageMemoryBarriers
  );

  VkImageSubresourceLayers subresource = {
    VK_IMAGE_ASPECT_COLOR_BIT, // aspectMask
    0, // mipLevel
    0, // baseArrayLayer
    swapchain_info_.imageArrayLayers, // layerCount
  };
  VkOffset3D offsets[2] = {
    { 0, 0, 0, },
    { (int32_t)swapchain_info_.imageExtent.width,
      (int32_t)swapchain_info_.imageExtent.height,
      1,
    },
  };
  VkImageBlit blit = {
    subresource,
    { offsets[0], offsets[1] },
    subresource,
    { offsets[0], offsets[1] },
  };
  device_functions_->vkCmdBlitImage(
      command_buffer_,
      image, // srcImage
      VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, // srcImageLayout
      images_[index], // dstImage
      VK_IMAGE_LAYOUT_PRESENT_SRC_KHR, // dstImageLayout
      1, // regionCount
      &blit, // pRegions
      VK_FILTER_NEAREST // filter
  );

  VkImageMemoryBarrier finalBarrier = initialBarrier;
  finalBarrier.srcAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;
  finalBarrier.dstAccessMask = VK_ACCESS_MEMORY_READ_BIT;
  finalBarrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;
  finalBarrier.newLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR;

  device_functions_->vkCmdPipelineBarrier(
      command_buffer_,
      VK_PIPELINE_STAGE_TRANSFER_BIT, // srcStageMask
      0, // dstStageMask
      0, // dependencyFlags
      0, // memoryBarrierCount
      nullptr, // pMemoryBarriers
      0, // bufferMemoryBarrierCount
      nullptr, // pBufferMemoryBarriers
      1, // imageMemoryBarrierCount
      &finalBarrier // pImageMemoryBarriers
  );

  if ((res = device_functions_->vkEndCommandBuffer(command_buffer_)) !=
      VK_SUCCESS) {
    return res;
  }

  VkPipelineStageFlags waitStage = VK_PIPELINE_STAGE_TRANSFER_BIT;
  VkSubmitInfo submitInfo = {
    VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
    0, // pNext
    1, // waitSemaphoreCount
    &acquire_semaphore_, // pWaitSemaphores
    &waitStage, // pWaitDstStageMask
    1, // commandBufferCount
    &command_buffer_, // pCommandBuffers
    1, // signalSemaphoreCount
    &blit_semaphore_, // pSignalSemaphores
  };
  auto queue_functions = GetGlobalContext().GetQueueData(queue);
  if ((res = queue_functions->vkQueueSubmit(queue, 1, &submitInfo,
                                            blit_fence_)) != VK_SUCCESS) {
    return res;
  }

  VkPresentInfoKHR presentInfo = {
    VK_STRUCTURE_TYPE_PRESENT_INFO_KHR, // sType
    0, // pNext
    1, // waitSemaphoreCount
    &blit_semaphore_, // waitSemaphores
    1, // swapchainCount
    &swapchain_, // pSwapchains,
    &index, // pImageIndices
    nullptr, // pResults
  };
  if ((res = queue_functions->vkQueuePresentKHR(queue, &presentInfo)) !=
      VK_SUCCESS) {
    return res;
  }

  device_functions_->vkWaitForFences(device_, 1, &blit_fence_, VK_TRUE,
                                     0xFFFFFFFFFFFFFFFFULL);
  device_functions_->vkResetFences(device_, 1, &blit_fence_);
  return VK_SUCCESS;
}
}
