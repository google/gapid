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

#include <cassert>
#include <sstream>
#include <vector>

#include "virtual_swapchain.h"

namespace swapchain {

// Used to set the value of VkSurfaceCapabilitiesKHR->currentExtent
// returned from vkGetPhysicalDeviceSurfaceCapabilitiesKHR.
// E.g. VIRTUAL_SWAPCHAIN_SURFACE_EXTENT="1960 1080"
// If unset then the current extent will be the "special value"
// {0xFFFFFFFF, 0xFFFFFFFF}, which some apps don't handle well.
// I.e. they will try to create a swapchain with this maximum extent size and we
// will then fail to create a buffer of this size.
const char* kOverrideSurfaceExtentEnv = "VIRTUAL_SWAPCHAIN_SURFACE_EXTENT";
// Android property names must be under 32 characters in Android N and below.
const char* kOverrideSurfaceExtentAndroidProp = "debug.vsc.surface_extent";

namespace {

void OverrideCurrentExtentIfNecessary(VkExtent2D* current_extent) {
  std::string overridden_extent;
  if (GetParameter(kOverrideSurfaceExtentEnv, kOverrideSurfaceExtentAndroidProp,
                   &overridden_extent)) {
    std::istringstream ss(overridden_extent);
    VkExtent2D extent;
    ss >> extent.width;
    if (ss.fail()) {
      write_warning("Failed to parse surface extent parameter: " +
                    overridden_extent);
      return;
    }
    ss >> extent.height;
    if (ss.fail()) {
      write_warning("Failed to parse surface extent parameter: " +
                    overridden_extent);
      return;
    }
    *current_extent = extent;
  }
}

}  // namespace

void RegisterInstance(VkInstance instance, const InstanceData& data) {
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

// The VirtualSurface is the surface we return to the application for all
// vkCreateXXXSurface calls.
struct VirtualSurface {
  bool always_return_given_surface_formats_and_present_modes;
  VkExtent2D current_extent;
};

VKAPI_ATTR VkResult VKAPI_CALL vkCreateVirtualSurface(
    VkInstance instance, const CreateNext* pCreateInfo,
    const VkAllocationCallbacks* pAllocator, VkSurfaceKHR* pSurface) {
  auto* surf = new VirtualSurface();
  surf->always_return_given_surface_formats_and_present_modes = false;
  surf->current_extent = {0xFFFFFFFF, 0xFFFFFFFF};

  if (pCreateInfo != nullptr) {
    for (const CreateNext* pNext =
             static_cast<const CreateNext*>(pCreateInfo->pNext);
         pNext != nullptr;
         pNext = static_cast<const CreateNext*>(pNext->pNext)) {
      if (pNext->sType == VIRTUAL_SWAPCHAIN_CREATE_PNEXT) {
        surf->always_return_given_surface_formats_and_present_modes = true;
      }
    }
  }

  OverrideCurrentExtentIfNecessary(&surf->current_extent);

  *pSurface = reinterpret_cast<VkSurfaceKHR>(surf);
  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetPhysicalDeviceSurfaceSupportKHR(
    VkPhysicalDevice physicalDevice, uint32_t queueFamilyIndex,
    VkSurfaceKHR surface, VkBool32* pSupported) {
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
    VkSurfaceCapabilitiesKHR* pSurfaceCapabilities) {
  // It would be illegal for the program to call VkDestroyInstance here.
  // We do not need to lock the map for the whole time, just
  // long enough to get the data out. unordered_map guarantees that
  // even if re-hashing occurs, references remain valid.
  VkPhysicalDeviceProperties& properties =
      GetGlobalContext()
          .GetPhysicalDeviceData(physicalDevice)
          ->physical_device_properties_;

  VirtualSurface* suf = reinterpret_cast<VirtualSurface*>(surface);

  pSurfaceCapabilities->minImageCount = 1;
  pSurfaceCapabilities->maxImageCount = 0;
  pSurfaceCapabilities->currentExtent = suf->current_extent;
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
      VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT | VK_IMAGE_USAGE_TRANSFER_SRC_BIT;
  // TODO(awoloszyn): Find a good set of formats that we can use
  // for rendering.

  return VK_SUCCESS;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetPhysicalDeviceSurfaceFormatsKHR(
    VkPhysicalDevice physicalDevice, VkSurfaceKHR surface,
    uint32_t* pSurfaceFormatCount, VkSurfaceFormatKHR* pSurfaceFormats) {
  VirtualSurface* suf = reinterpret_cast<VirtualSurface*>(surface);
  if (suf->always_return_given_surface_formats_and_present_modes) {
    return VK_SUCCESS;
  }
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
    uint32_t* pPresentModeCount, VkPresentModeKHR* pPresentModes) {
  VirtualSurface* suf = reinterpret_cast<VirtualSurface*>(surface);
  if (suf->always_return_given_surface_formats_and_present_modes) {
    return VK_SUCCESS;
  }
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
    VkDevice device, const VkSwapchainCreateInfoKHR* pCreateInfo,
    const VkAllocationCallbacks* pAllocator, VkSwapchainKHR* pSwapchain) {
  DeviceData& dev_dat = *GetGlobalContext().GetDeviceData(device);
  PhysicalDeviceData& pdd =
      *GetGlobalContext().GetPhysicalDeviceData(dev_dat.physicalDevice);
  InstanceData& inst_dat = *GetGlobalContext().GetInstanceData(pdd.instance_);

  uint32_t property_count = 0;
  inst_dat.vkGetPhysicalDeviceQueueFamilyProperties(dev_dat.physicalDevice,
                                                    &property_count, nullptr);

  std::vector<VkQueueFamilyProperties> queue_properties(property_count);
  inst_dat.vkGetPhysicalDeviceQueueFamilyProperties(
      dev_dat.physicalDevice, &property_count, queue_properties.data());

  size_t queue = 0;
  for (; queue < queue_properties.size(); ++queue) {
    if (queue_properties[queue].queueFlags & VK_QUEUE_GRAPHICS_BIT) break;
  }

  assert(queue < queue_properties.size());

  auto swp = new VirtualSwapchain(
      device, queue, &pdd.physical_device_properties_, &pdd.memory_properties_,
      &dev_dat, pCreateInfo, pAllocator);

  for (const CreateNext* pNext =
           static_cast<const CreateNext*>(pCreateInfo->pNext);
       pNext != nullptr; pNext = static_cast<const CreateNext*>(pNext->pNext)) {
    if (pNext->sType == VIRTUAL_SWAPCHAIN_CREATE_PNEXT) {
      swp->SetAlwaysGetAcquiredImage(true);
      if (pNext->surfaceCreateInfo) {
        swp->CreateBaseSwapchain(pdd.instance_, &inst_dat, pAllocator,
                                 pNext->surfaceCreateInfo);
      }
      break;
    }
  }

  *pSwapchain = reinterpret_cast<VkSwapchainKHR>(swp);
  return VK_SUCCESS;
}

VKAPI_ATTR void VKAPI_CALL vkSetHdrMetadataEXT(
    VkDevice device, uint32_t swapchainCount, const VkSwapchainKHR* pSwapchains,
    const VkHdrMetadataEXT* pMetadata) {
  // This is a no-op for the virtual swapchain
}

VKAPI_ATTR void VKAPI_CALL
vkDestroySwapchainKHR(VkDevice device, VkSwapchainKHR swapchain,
                      const VkAllocationCallbacks* pAllocator) {
  VirtualSwapchain* swp = reinterpret_cast<VirtualSwapchain*>(swapchain);
  swp->Destroy(pAllocator);
  delete swp;
}

VKAPI_ATTR void VKAPI_CALL
vkDestroySurfaceKHR(VkInstance instance, VkSurfaceKHR surface,
                    const VkAllocationCallbacks* pAllocator) {
  VirtualSurface* suf = reinterpret_cast<VirtualSurface*>(surface);
  delete suf;
}

VKAPI_ATTR VkResult VKAPI_CALL vkGetSwapchainImagesKHR(
    VkDevice device, VkSwapchainKHR swapchain, uint32_t* pSwapchainImageCount,
    VkImage* pSwapchainImages) {
  VirtualSwapchain* swp = reinterpret_cast<VirtualSwapchain*>(swapchain);
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

VKAPI_ATTR void VKAPI_CALL vkSetSwapchainCallback(VkSwapchainKHR swapchain,
                                                  void callback(void*, uint8_t*,
                                                                size_t),
                                                  void* user_data) {
  VirtualSwapchain* swp = reinterpret_cast<VirtualSwapchain*>(swapchain);
  swp->SetCallback(callback, user_data);
}

// We actually have to be able to submit data to the Queue right now.
// The user can supply either a semaphore, or a fence or both to this function.
// Because of this, once the image is available we have to submit
// a command to the queue to signal these.
VKAPI_ATTR VkResult VKAPI_CALL vkAcquireNextImageKHR(
    VkDevice device, VkSwapchainKHR swapchain, uint64_t timeout,
    VkSemaphore semaphore, VkFence fence, uint32_t* pImageIndex) {
  VirtualSwapchain* swp = reinterpret_cast<VirtualSwapchain*>(swapchain);
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
  DeviceData& dat = *GetGlobalContext().GetDeviceData(device);
  VkQueue q;

  dat.vkGetDeviceQueue(device, swp->DeviceQueue(), 0, &q);
  set_dispatch_from_parent(q, device);

  bool has_semaphore = semaphore != VK_NULL_HANDLE;

  VkSemaphore wait_semaphore = swp->GetAcquireWaitSemaphore(*pImageIndex);
  VkPipelineStageFlags wait_stage = VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT;
  bool has_wait_semaphore = wait_semaphore != VK_NULL_HANDLE;

  VkSubmitInfo info{
      VK_STRUCTURE_TYPE_SUBMIT_INFO,                     // sType
      nullptr,                                           // pNext
      (has_wait_semaphore ? 1u : 0u),                    // waitSemaphoreCount
      (has_wait_semaphore ? &wait_semaphore : nullptr),  // pWaitSemaphores
      (has_wait_semaphore ? &wait_stage : nullptr),      // pWaitDstStageMask
      0,                                                 // commandBufferCount
      nullptr,                                           // pCommandBuffers
      (has_semaphore ? 1u : 0u),                         // semaphoreCount
      (has_semaphore ? &semaphore : nullptr)};           // pSemaphores
  return swapchain::vkQueueSubmit(q, 1, &info, fence);
}

// We actually have to be able to submit data to the Queue right now.
// The user can supply either a semaphore, or a fence or both to this function.
// Because of this, once the image is available we have to submit
// a command to the queue to signal these.
VKAPI_ATTR VkResult VKAPI_CALL vkAcquireNextImage2KHR(
    VkDevice device, const VkAcquireNextImageInfoKHR* pAcquireInfo,
    uint32_t* pImageIndex) {
  // TODO(awoloszyn): Implement proper multiGPU here eventually.
  return swapchain::vkAcquireNextImageKHR(
      device, pAcquireInfo->swapchain, pAcquireInfo->timeout,
      pAcquireInfo->semaphore, pAcquireInfo->fence, pImageIndex);
}

VKAPI_ATTR VkResult VKAPI_CALL
vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR* pPresentInfo) {
  // We submit to the queue the commands set up by the virtual swapchain.
  // This will start a copy operation from the image to the swapchain
  // buffers.

  VkResult res = VK_SUCCESS;

  std::vector<VkPipelineStageFlags> pipeline_stages(
      pPresentInfo->waitSemaphoreCount, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT);

  size_t i = 0;
  for (; i < pPresentInfo->swapchainCount; ++i) {
    uint32_t image_index = pPresentInfo->pImageIndices[i];

    VirtualSwapchain* swp =
        reinterpret_cast<VirtualSwapchain*>(pPresentInfo->pSwapchains[i]);

    VkSubmitInfo submitInfo{
        VK_STRUCTURE_TYPE_SUBMIT_INFO,                     // sType
        nullptr,                                           // nullptr
        i == 0 ? pPresentInfo->waitSemaphoreCount : 0,     // waitSemaphoreCount
        i == 0 ? pPresentInfo->pWaitSemaphores : nullptr,  // pWaitSemaphores
        i == 0 ? pipeline_stages.data() : nullptr,         // pWaitDstStageMask
        1,                                                 // commandBufferCount
        &swp->GetCommandBuffer(image_index),               // pCommandBuffers
        0,                                                 // semaphoreCount
        nullptr                                            // pSemaphores
    };

    res = EXPECT_SUCCESS(GetGlobalContext().GetQueueData(queue)->vkQueueSubmit(
        queue, 1, &submitInfo, swp->GetFence(image_index)));

    if (res != VK_SUCCESS) {
      break;
    }

    res = swp->PresentToSurface(queue, image_index);
    if (res != VK_SUCCESS) {
      break;
    }

    swp->NotifySubmitted(image_index);

    if (pPresentInfo->pResults) {
      pPresentInfo->pResults[i] = VK_SUCCESS;
    }
  }

  // If we left the above loop early, then set the remaining results as errors.
  if (pPresentInfo->pResults) {
    for (; i < pPresentInfo->swapchainCount; ++i) {
      pPresentInfo->pResults[i] = res;
    }
  }

  return res;
}

VKAPI_ATTR VkResult VKAPI_CALL vkQueueSubmit(VkQueue queue,
                                             uint32_t submitCount,
                                             const VkSubmitInfo* pSubmits,
                                             VkFence fence) {
  // We actually DO have to lock here, we may share this queue with
  // vkAcquireNextImageKHR, which is not externally synchronized on Queue.
  return GetGlobalContext().GetQueueData(queue)->vkQueueSubmit(
      queue, submitCount, pSubmits, fence);
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
    uint32_t memoryBarrierCount, const VkMemoryBarrier* pMemoryBarriers,
    uint32_t bufferMemoryBarrierCount,
    const VkBufferMemoryBarrier* pBufferMemoryBarriers,
    uint32_t imageMemoryBarrierCount,
    const VkImageMemoryBarrier* pImageMemoryBarriers) {
  std::vector<VkImageMemoryBarrier> imageBarriers(imageMemoryBarrierCount);
  for (size_t i = 0; i < imageMemoryBarrierCount; ++i) {
    imageBarriers[i] = pImageMemoryBarriers[i];
    if (imageBarriers[i].oldLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      imageBarriers[i].oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
      imageBarriers[i].srcAccessMask |= VK_ACCESS_TRANSFER_READ_BIT;
      // Ensure the stage mask supports the transfer read access.
      srcStageMask |= VK_PIPELINE_STAGE_TRANSFER_BIT;
    }
    if (imageBarriers[i].newLayout == VK_IMAGE_LAYOUT_PRESENT_SRC_KHR) {
      imageBarriers[i].newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
      imageBarriers[i].dstAccessMask |= VK_ACCESS_TRANSFER_READ_BIT;
      // Ensure the stage mask supports the transfer read access.
      dstStageMask |= VK_PIPELINE_STAGE_TRANSFER_BIT;
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
    VkCommandBuffer commandBuffer, uint32_t eventCount, const VkEvent* pEvents,
    VkPipelineStageFlags srcStageMask, VkPipelineStageFlags dstStageMask,
    uint32_t memoryBarrierCount, const VkMemoryBarrier* pMemoryBarriers,
    uint32_t bufferMemoryBarrierCount,
    const VkBufferMemoryBarrier* pBufferMemoryBarriers,
    uint32_t imageMemoryBarrierCount,
    const VkImageMemoryBarrier* pImageMemoryBarriers) {
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
    VkDevice device, const VkRenderPassCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator, VkRenderPass* pRenderPass) {
  VkRenderPassCreateInfo intercepted = *pCreateInfo;
  std::vector<VkAttachmentDescription> attachments(
      pCreateInfo->attachmentCount);
  intercepted.pAttachments = attachments.data();

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
}  // namespace swapchain
