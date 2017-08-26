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

#include "swapchain.h"
#include "virtual_swapchain.h"
#include <cassert>
#include <vector>

namespace swapchain {

void RegisterInstance(VkInstance instance, const InstanceData &data) {
  uint32_t num_devices = 0;
  data.vkEnumeratePhysicalDevices(instance, &num_devices, nullptr);

  std::vector<VkPhysicalDevice> physical_devices(num_devices);
  data.vkEnumeratePhysicalDevices(instance, &num_devices,
                                  physical_devices.data());

  auto physical_device_map = GetGlobalContext().GetPhysicalDeviceMap();

  for (VkPhysicalDevice physical_device : physical_devices) {
    PhysicalDeviceData dat{instance};
    data.vkGetPhysicalDeviceMemoryProperties(physical_device,
                                             &dat.memory_properties_);
    data.vkGetPhysicalDeviceProperties(physical_device,
                                       &dat.physical_device_properties_);
    (*physical_device_map)[physical_device] = dat;
  }
}

// For now VirtualSurface is empty. Once we start tracking more
// information from the host, then we can start expanding
// what we are able to expose here.
struct VirtualSurface {};

VKAPI_ATTR VkResult VKAPI_CALL vkCreateVirtualSurface(
    VkInstance instance, const void * /*pCreateInfo*/,
    const VkAllocationCallbacks *pAllocator, VkSurfaceKHR *pSurface) {
  *pSurface = reinterpret_cast<VkSurfaceKHR>(new VirtualSurface());
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetPhysicalDeviceSurfaceSupportKHR(
    VkPhysicalDevice physicalDevice, uint32_t queueFamilyIndex,
    VkSurfaceKHR surface, VkBool32 *pSupported) {
  const auto instance_dat = *GetGlobalContext().GetInstanceData(
      GetGlobalContext().GetPhysicalDeviceData(physicalDevice)->instance_);

  for (uint32_t i = 0; i <= queueFamilyIndex; ++i) {
    uint32_t property_count = 0;
    instance_dat.vkGetPhysicalDeviceQueueFamilyProperties(
        physicalDevice, &property_count, nullptr);
    assert(property_count > queueFamilyIndex);

    std::vector<VkQueueFamilyProperties> properties(property_count);
    instance_dat.vkGetPhysicalDeviceQueueFamilyProperties(
        physicalDevice, &property_count, properties.data());

    if (properties[queueFamilyIndex].queueFlags & VK_QUEUE_GRAPHICS_BIT) {
      *pSupported = (i == queueFamilyIndex);
      return VK_SUCCESS;
    }
  }

  // For now only support the FIRST graphics queue. It looks like all of
  // the commands we will have to run are transfer commands, so
  // we can probably get away with ANY queue (other than
  // SPARSE_BINDING).
  *pSupported = false;
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetPhysicalDeviceSurfaceCapabilitiesKHR(
    VkPhysicalDevice physicalDevice, VkSurfaceKHR surface,
    VkSurfaceCapabilitiesKHR *pSurfaceCapabilities) {

  // It would be illegal for the program to call VkDestroyInstance here.
  // We do not need to lock the map for the whole time, just
  // long enough to get the data out. unordered_map guarantees that
  // even if re-hashing occurs, references remain valid.
  VkPhysicalDeviceProperties &properties =
      GetGlobalContext()
          .GetPhysicalDeviceData(physicalDevice)
          ->physical_device_properties_;

  pSurfaceCapabilities->minImageCount = 1;
  pSurfaceCapabilities->maxImageCount = 0;
  pSurfaceCapabilities->currentExtent = {0xFFFFFFFF, 0xFFFFFFFF};
  pSurfaceCapabilities->minImageExtent = {1, 1};
  pSurfaceCapabilities->maxImageExtent = {
      properties.limits.maxImageDimension2D,
      properties.limits.maxImageDimension2D};
  pSurfaceCapabilities->maxImageArrayLayers =
      properties.limits.maxImageArrayLayers;
  pSurfaceCapabilities->supportedTransforms =
      VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR;
  // TODO(awoloszyn): Handle all of the transforms eventually
  pSurfaceCapabilities->currentTransform =
      VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR;
  pSurfaceCapabilities->supportedCompositeAlpha =
      VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR;
  // TODO(awoloszyn): Handle all of the composite types.

  pSurfaceCapabilities->supportedUsageFlags =
      VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT;
  // TODO(awoloszyn): Find a good set of formats that we can use
  // for rendering.

  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetPhysicalDeviceSurfaceFormatsKHR(
    VkPhysicalDevice physicalDevice, VkSurfaceKHR surface,
    uint32_t *pSurfaceFormatCount, VkSurfaceFormatKHR *pSurfaceFormats) {

  if (!pSurfaceFormats) {
    *pSurfaceFormatCount = 1;
    return VK_SUCCESS;
  }
  if (*pSurfaceFormatCount < 1) {
    return VK_INCOMPLETE;
  }
  *pSurfaceFormatCount = 1;

  // TODO(awoloszyn): Handle more different formats.
  pSurfaceFormats->format = VK_FORMAT_R8G8B8A8_UNORM;
  pSurfaceFormats->colorSpace = VK_COLORSPACE_SRGB_NONLINEAR_KHR;
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetPhysicalDeviceSurfacePresentModesKHR(
    VkPhysicalDevice physicalDevice, VkSurfaceKHR surface,
    uint32_t *pPresentModeCount, VkPresentModeKHR *pPresentModes) {

  if (!pPresentModes) {
    *pPresentModeCount = 1;
    return VK_SUCCESS;
  }
  if (*pPresentModeCount < 1) {
    return VK_INCOMPLETE;
  }
  // TODO(awoloszyn): Add more present modes. we MUST support
  // VK_PRESENT_MODE_FIFO_KHR.
  *pPresentModes = VK_PRESENT_MODE_FIFO_KHR;
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkCreateSwapchainKHR(
    VkDevice device, const VkSwapchainCreateInfoKHR *pCreateInfo,
    const VkAllocationCallbacks *pAllocator, VkSwapchainKHR *pSwapchain) {

  DeviceData &dev_dat = *GetGlobalContext().GetDeviceData(device);
  PhysicalDeviceData &pdd =
      *GetGlobalContext().GetPhysicalDeviceData(dev_dat.physicalDevice);
  InstanceData &inst_dat = *GetGlobalContext().GetInstanceData(pdd.instance_);

  uint32_t property_count = 0;
  inst_dat.vkGetPhysicalDeviceQueueFamilyProperties(dev_dat.physicalDevice,
                                                    &property_count, nullptr);

  std::vector<VkQueueFamilyProperties> queue_properties(property_count);
  inst_dat.vkGetPhysicalDeviceQueueFamilyProperties(
      dev_dat.physicalDevice, &property_count, queue_properties.data());

  size_t queue = 0;
  for (; queue < queue_properties.size(); ++queue) {
    if (queue_properties[queue].queueFlags & VK_QUEUE_GRAPHICS_BIT)
      break;
  }

  assert(queue < queue_properties.size());

  *pSwapchain = reinterpret_cast<VkSwapchainKHR>(new VirtualSwapchain(
      device, queue, &pdd.physical_device_properties_, &pdd.memory_properties_,
      &dev_dat, pCreateInfo, pAllocator));
  return VK_SUCCESS;
}

VKAPI_ATTR void VKAPI_CALL
vkDestroySwapchainKHR(VkDevice device, VkSwapchainKHR swapchain,
                      const VkAllocationCallbacks *pAllocator) {
  VirtualSwapchain *swp = reinterpret_cast<VirtualSwapchain *>(swapchain);
  swp->Destroy(pAllocator);
  delete swp;
}

VKAPI_ATTR void VKAPI_CALL
vkDestroySurfaceKHR(VkInstance instance, VkSurfaceKHR surface,
                    const VkAllocationCallbacks *pAllocator) {}

VKAPI_ATTR VkResult VKAPI_CALL vkGetSwapchainImagesKHR(
    VkDevice device, VkSwapchainKHR swapchain, uint32_t *pSwapchainImageCount,
    VkImage *pSwapchainImages) {
  VirtualSwapchain *swp = reinterpret_cast<VirtualSwapchain *>(swapchain);
  const auto images =
      swp->GetImages(*pSwapchainImageCount, pSwapchainImages != nullptr);
  if (!pSwapchainImages) {
    *pSwapchainImageCount = images.size();
    return VK_SUCCESS;
  }

  VkResult res = VK_INCOMPLETE;
  if (*pSwapchainImageCount >= images.size()) {
    *pSwapchainImageCount = images.size();
    res = VK_SUCCESS;
  }

  for (size_t i = 0; i < *pSwapchainImageCount; ++i) {
    pSwapchainImages[i] = images[i];
  }

  return res;
}

VKAPI_ATTR void VKAPI_CALL vkSetSwapchainCallback(
    VkSwapchainKHR swapchain, void callback(void *, uint8_t *, size_t),
    void *user_data) {
  VirtualSwapchain *swp = reinterpret_cast<VirtualSwapchain *>(swapchain);
  swp->SetCallback(callback, user_data);
}

// We actually have to be able to submit data to the Queue right now.
// The user can supply either a semaphore, or a fence or both to this function.
// Because of this, once the image is available we have to submit
// a command to the queue to signal these.
VKAPI_ATTR VkResult VKAPI_CALL vkAcquireNextImageKHR(
    VkDevice device, VkSwapchainKHR swapchain, uint64_t timeout,
    VkSemaphore semaphore, VkFence fence, uint32_t *pImageIndex) {
  VirtualSwapchain *swp = reinterpret_cast<VirtualSwapchain *>(swapchain);
  if (!swp->GetImage(timeout, pImageIndex)) {
    return timeout == 0 ? VK_NOT_READY : VK_TIMEOUT;
  }

  // It is important that we do not keep the lock here.
  // *GetGlobalContext().GetDeviceData() only holds the lock
  // for the duration of the call, if we instead do something like
  // auto dat = GetGlobalContext().GetDeviceData(device),
  // then the lock will be let go when dat is destroyed, which is
  // AFTER swapchain::vkQueueSubmit, this would be a priority
  // inversion on the locks.
  DeviceData &dat = *GetGlobalContext().GetDeviceData(device);
  VkQueue q;

  dat.vkGetDeviceQueue(device, swp->DeviceQueue(), 0, &q);

  bool has_semaphore = semaphore != VK_NULL_HANDLE;

  VkSubmitInfo info{VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
                    nullptr,                       // pNext
                    0,                             // waitSemaphoreCount
                    nullptr,                       // waitSemaphores
                    nullptr,                       // waitDstStageMask
                    0,                             // commandBufferCount
                    nullptr,                       // pCommandBuffers
                    (has_semaphore ? 1u : 0u),     // semaphoreCount
                    (has_semaphore ? &semaphore : nullptr)}; // pSemaphores
  return swapchain::vkQueueSubmit(q, 1, &info, fence);

}

VKAPI_ATTR VkResult VKAPI_CALL
vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR *pPresentInfo) {
  // We submit to the queue the commands set up by the virtual swapchain.
  // This will start a copy operation from the image to the swapchain
  // buffers.
  uint32_t res = VK_SUCCESS;
  std::vector<VkPipelineStageFlags> pipeline_stages(
      pPresentInfo->waitSemaphoreCount, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT);
  for (size_t i = 0; i < pPresentInfo->swapchainCount; ++i) {
    uint32_t image_index = pPresentInfo->pImageIndices[i];
    VirtualSwapchain *swp =
        reinterpret_cast<VirtualSwapchain *>(pPresentInfo->pSwapchains[i]);

    VkSubmitInfo submitInfo{
        VK_STRUCTURE_TYPE_SUBMIT_INFO,                    // sType
        nullptr,                                          // nullptr
        i == 0 ? pPresentInfo->waitSemaphoreCount : 0,    // waitSemaphoreCount
        i == 0 ? pPresentInfo->pWaitSemaphores : nullptr, // pWaitSemaphores
        i == 0 ? pipeline_stages.data() : nullptr,        // pWaitDstStageMask
        1,                                                // commandBufferCount
        &swp->GetCommandBuffer(image_index),              // pCommandBuffers
        0,                                                // semaphoreCount
        nullptr                                           // pSemaphores
    };

    res |= GetGlobalContext().GetQueueData(queue)->vkQueueSubmit(
        queue, 1, &submitInfo, swp->GetFence(image_index));
    swp->NotifySubmitted(image_index);
  }

  return VkResult(res);
}

VKAPI_ATTR VkResult VKAPI_CALL vkQueueSubmit(VkQueue queue,
                                             uint32_t submitCount,
                                             const VkSubmitInfo *pSubmits,
                                             VkFence fence) {
  // We actually DO have to lock here, we may share this queue with
  // vkAcquireNextImageKHR, which is not externally synchronized on Queue.
  return GetGlobalContext().GetQueueData(queue)->vkQueueSubmit(queue, submitCount,
                                                        pSubmits, fence);
}

// The following 3 functions are special. We would normally not have to
// handle them, but since we cannot rely on there being an internal swapchain
// mechanism, we cannot allow VK_IMAGE_LAYOUT_PRESENT_SRC_KHR to be passed
// to the driver. In this case any time a user uses a layout that is
// VK_IMAGE_LAYOUT_PRESENT_SRC_KHR we replace that with
// VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, which is what we need an image to be
// set up as when we have to copy anyway.
VKAPI_ATTR void VKAPI_CALL vkCmdPipelineBarrier(
    VkCommandBuffer commandBuffer, VkPipelineStageFlags srcStageMask,
    VkPipelineStageFlags dstStageMask, VkDependencyFlags dependencyFlags,
    uint32_t memoryBarrierCount, const VkMemoryBarrier *pMemoryBarriers,
    uint32_t bufferMemoryBarrierCount,
    const VkBufferMemoryBarrier *pBufferMemoryBarriers,
    uint32_t imageMemoryBarrierCount,
    const VkImageMemoryBarrier *pImageMemoryBarriers) {

  std::vector<VkImageMemoryBarrier> imageBarriers(imageMemoryBarrierCount);
  for (size_t i = 0; i < imageMemoryBarrierCount; ++i) {
    imageBarriers[i] = pImageMemoryBarriers[i];
    if (imageBarriers[i].oldLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      imageBarriers[i].oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
      imageBarriers[i].srcAccessMask |= VK_ACCESS_TRANSFER_READ_BIT;
    }
    if (imageBarriers[i].newLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      imageBarriers[i].newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
      imageBarriers[i].dstAccessMask |= VK_ACCESS_TRANSFER_READ_BIT;
    }
  }
  PFN_vkCmdPipelineBarrier func = GetGlobalContext()
                                      .GetCommandBufferData(commandBuffer)
                                      ->vkCmdPipelineBarrier;

  return func(commandBuffer, srcStageMask, dstStageMask, dependencyFlags,
              memoryBarrierCount, pMemoryBarriers, bufferMemoryBarrierCount,
              pBufferMemoryBarriers, imageMemoryBarrierCount,
              imageBarriers.data());
}

VKAPI_ATTR void VKAPI_CALL vkCmdWaitEvents(
    VkCommandBuffer commandBuffer, uint32_t eventCount, const VkEvent *pEvents,
    VkPipelineStageFlags srcStageMask, VkPipelineStageFlags dstStageMask,
    uint32_t memoryBarrierCount, const VkMemoryBarrier *pMemoryBarriers,
    uint32_t bufferMemoryBarrierCount,
    const VkBufferMemoryBarrier *pBufferMemoryBarriers,
    uint32_t imageMemoryBarrierCount,
    const VkImageMemoryBarrier *pImageMemoryBarriers) {

  std::vector<VkImageMemoryBarrier> imageBarriers(imageMemoryBarrierCount);
  for (size_t i = 0; i < imageMemoryBarrierCount; ++i) {
    imageBarriers[i] = pImageMemoryBarriers[i];
    if (imageBarriers[i].oldLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      imageBarriers[i].oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
      imageBarriers[i].srcAccessMask |= VK_ACCESS_TRANSFER_READ_BIT;
    }
    if (imageBarriers[i].newLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      imageBarriers[i].newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
      imageBarriers[i].dstAccessMask |= VK_ACCESS_TRANSFER_READ_BIT;
    }
  }
  PFN_vkCmdWaitEvents func =
      GetGlobalContext().GetCommandBufferData(commandBuffer)->vkCmdWaitEvents;

  func(commandBuffer, eventCount, pEvents, srcStageMask, dstStageMask,
       memoryBarrierCount, pMemoryBarriers, bufferMemoryBarrierCount,
       pBufferMemoryBarriers, imageMemoryBarrierCount, imageBarriers.data());
}

VKAPI_ATTR VkResult VKAPI_CALL vkCreateRenderPass(
    VkDevice device, const VkRenderPassCreateInfo *pCreateInfo,
    const VkAllocationCallbacks *pAllocator, VkRenderPass *pRenderPass) {
  VkRenderPassCreateInfo intercepted = *pCreateInfo;
  std::vector<VkAttachmentDescription> attachments(
      pCreateInfo->attachmentCount);

  for (size_t i = 0; i < pCreateInfo->attachmentCount; ++i) {
    attachments[i] = pCreateInfo->pAttachments[i];
    if (attachments[i].initialLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      attachments[i].initialLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
    }
    if (attachments[i].finalLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      attachments[i].finalLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
    }
  }
  PFN_vkCreateRenderPass func =
      GetGlobalContext().GetDeviceData(device)->vkCreateRenderPass;

  return func(device, &intercepted, pAllocator, pRenderPass);
}
} // swapchain
