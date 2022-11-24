#include <unordered_map>

#include "base64.h"
#include "json.hpp"
#include "layer.h"
#include "miniz_tdef.h"
// Since this is designed to only handle screenshots of things
// being rendered (for now) we dont have to track the current
// image layout, as it will either be COLOR_ATTACHMENT_OPTIMAL,
// or DEPTH_STENCIL_ATTACHMENT_OPTIMAL

std::unordered_map<VkFormat, uint64_t> bytes_per_pixel = {
    {VK_FORMAT_R8G8B8A8_UNORM, 4},
    {VK_FORMAT_D16_UNORM, 2}};

struct image_info {
  VkImageType type;
  VkFormat format;
  uint32_t mip_levels;
  uint32_t arrayLayers;
  VkSampleCountFlagBits samples;
  VkExtent3D extent;
};

std::unordered_map<VkDevice, VkPhysicalDeviceMemoryProperties> memory_properties;
std::unordered_map<VkQueue, VkDevice> queuesToDevices;

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateDevice(
    VkPhysicalDevice physicalDevice,
    const VkDeviceCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkDevice* pDevice) {
  auto ret = vkCreateDevice(physicalDevice, pCreateInfo, pAllocator, pDevice);
  if (ret != VK_SUCCESS) {
    return ret;
  }
  VkPhysicalDeviceMemoryProperties props;
  vkGetPhysicalDeviceMemoryProperties(physicalDevice, &props);
  memory_properties[*pDevice] = props;
}

VKAPI_ATTR void VKAPI_CALL override_vkGetDeviceQueue(
    VkDevice device,
    uint32_t queueFamilyIndex,
    uint32_t queueIndex,
    VkQueue* pQueue) {
  vkGetDeviceQueue(device, queueFamilyIndex, queueIndex, pQueue);
  queuesToDevices[*pQueue] = device;
}

std::unordered_map<VkImage, image_info> image_infos;

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateImage(
    VkDevice device,
    const VkImageCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkImage* pImage) {
  auto ret = vkCreateImage(device, pCreateInfo, pAllocator, pImage);

  if (ret == VK_SUCCESS) {
    image_infos[*pImage] = image_info{
        .type = pCreateInfo->imageType,
        .format = pCreateInfo->format,
        .mip_levels = pCreateInfo->mipLevels,
        .arrayLayers = pCreateInfo->arrayLayers,
        .samples = pCreateInfo->samples,
        .extent = pCreateInfo->extent};
  }

  return ret;
}

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateSwapchainKHR(
    VkDevice device,
    const VkSwapchainCreateInfoKHR* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkSwapchainKHR* pSwapchain) {
  auto ret = vkCreateSwapchainKHR(device, pCreateInfo, pAllocator, pSwapchain);
  if (ret == VK_SUCCESS) {
    uint32_t count = 0;
    vkGetSwapchainImagesKHR(device, *pSwapchain, &count, nullptr);
    std::vector<VkImage> images(count);
    vkGetSwapchainImagesKHR(device, *pSwapchain, &count, images.data());
    for (auto& x : images) {
      image_infos[x] = image_info{
          .type = VK_IMAGE_TYPE_2D,
          .format = pCreateInfo->imageFormat,
          .mip_levels = 1,
          .arrayLayers = pCreateInfo->imageArrayLayers,
          .samples = VK_SAMPLE_COUNT_1_BIT,
          .extent = VkExtent3D{
              pCreateInfo->imageExtent.width,
              pCreateInfo->imageExtent.height,
              1}};
    }
  }
  return ret;
}

std::unordered_map<VkImageView, VkImageViewCreateInfo> image_view_infos;
VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateImageView(
    VkDevice device,
    const VkImageViewCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkImageView* pView) {
  auto ret = vkCreateImageView(device, pCreateInfo, pAllocator, pView);
  if (ret == VK_SUCCESS) {
    image_view_infos[*pView] = *pCreateInfo;
  }
  return ret;
}

struct framebuffer_info {
  uint32_t width;
  uint32_t height;
  uint32_t layers;
  std::vector<VkImageView> image_views;
};

std::unordered_map<VkFramebuffer, framebuffer_info> framebuffer_infos;

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateFramebuffer(
    VkDevice device,
    const VkFramebufferCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkFramebuffer* pFramebuffer) {
  auto ret = vkCreateFramebuffer(device, pCreateInfo, pAllocator, pFramebuffer);
  if (ret == VK_SUCCESS) {
    framebuffer_infos[*pFramebuffer] = framebuffer_info{
        .width = pCreateInfo->width,
        .height = pCreateInfo->height,
        .layers = pCreateInfo->layers,
        .image_views = std::vector<VkImageView>(pCreateInfo->pAttachments, pCreateInfo->pAttachments + pCreateInfo->attachmentCount)};
  }

  return ret;
}

struct renderpass_info {
  struct subpass_info {
    std::vector<VkAttachmentReference> input_attachments;
    std::vector<VkAttachmentReference> color_attachments;
    std::unique_ptr<VkAttachmentReference> depth_attachment;
  };

  std::vector<subpass_info> subpasses;
};

std::unordered_map<VkRenderPass, renderpass_info> render_pass_infos;

VKAPI_ATTR VkResult VKAPI_CALL override_vkCreateRenderPass(
    VkDevice device,
    const VkRenderPassCreateInfo* pCreateInfo,
    const VkAllocationCallbacks* pAllocator,
    VkRenderPass* pRenderPass) {
  auto ret = vkCreateRenderPass(device, pCreateInfo, pAllocator, pRenderPass);
  if (ret == VK_SUCCESS) {
    renderpass_info rpi;
    for (uint32_t i = 0; i < pCreateInfo->subpassCount; ++i) {
      const auto& subpassDesc = pCreateInfo->pSubpasses[i];
      rpi.subpasses.push_back({.input_attachments = std::vector<VkAttachmentReference>(
                                   subpassDesc.pInputAttachments, subpassDesc.pInputAttachments + subpassDesc.inputAttachmentCount),
                               .color_attachments = std::vector<VkAttachmentReference>(
                                   subpassDesc.pColorAttachments, subpassDesc.pColorAttachments + subpassDesc.colorAttachmentCount),
                               .depth_attachment = subpassDesc.pDepthStencilAttachment ? std::make_unique<VkAttachmentReference>(*subpassDesc.pDepthStencilAttachment) : nullptr});
    }
    render_pass_infos[*pRenderPass] = std::move(rpi);
  }
  return ret;
}

struct screenshot_locations {
  VkCommandBuffer command_buffer;
  std::vector<uint64_t> cb_indices;
};

std::unordered_map<uint64_t, std::vector<screenshot_locations>> submit_indices;

VKAPI_ATTR void VKAPI_CALL SetupLayer(LayerOptions* options) {
  const char* js = options->GetUserConfig();
  uint32_t width = 1024;
  uint32_t height = 1024;
  if (js) {
    auto setup = nlohmann::json::parse(js, nullptr, false);
    if (setup.contains("screenshot_locations")) {
      for (auto& val : setup["screenshot_locations"]) {
        std::vector<screenshot_locations> locs;
        uint64_t i = val["submit_index"];
        auto& command_buffers = val["command_buffers"];
        for (auto& cb : command_buffers) {
          VkCommandBuffer buff = reinterpret_cast<VkCommandBuffer>(static_cast<uintptr_t>(cb["command_buffer"]));
          std::vector<uint64_t> indices = cb["indices"];
          screenshot_locations sl{
              buff, std::move(indices)};
          locs.push_back(std::move(sl));
        }
        submit_indices[i] = std::move(locs);
        LogMessage(debug, std::format("Adding!! {}", i));
      }
    }
  }
  options->CaptureAllCommands();
}

struct image_copy_data {
  VkImage temporary_image = VK_NULL_HANDLE;
  VkDeviceMemory temporary_image_memory = VK_NULL_HANDLE;
  VkBuffer transfer_buffer = VK_NULL_HANDLE;
  VkDeviceMemory transfer_buffer_memory = VK_NULL_HANDLE;
  VkFormat format = static_cast<VkFormat>(0);
  uint32_t width = 0;
  uint32_t height = 0;
};

VkQueue re_recording_queue = VK_NULL_HANDLE;
std::vector<image_copy_data> images_to_get;
VkFramebuffer current_framebuffer;
VkRenderPass current_renderpass;

VKAPI_ATTR VkResult VKAPI_CALL override_vkQueueSubmit(
    VkQueue queue,
    uint32_t submitCount,
    const VkSubmitInfo* pSubmits,
    VkFence fence) {
  auto current_command_index = GetCommandIndex();
  if (submit_indices.count(current_command_index)) {
    current_framebuffer = VK_NULL_HANDLE;
    re_recording_queue = queue;
    const auto& screenshot_locations = submit_indices[current_command_index];
    for (auto& loc : screenshot_locations) {
      std::string positions = "[";
      bool first = true;
      for (auto& x : loc.cb_indices) {
        if (!first) {
          positions += ", ";
        }
        positions += int32_t(x);
      }
      positions += "]";
      Split_CommandBuffer(loc.command_buffer, loc.cb_indices.data(), static_cast<uint32_t>(loc.cb_indices.size()));
    }
  }

  auto ret = vkQueueSubmit(queue, submitCount, pSubmits, fence);

  if (re_recording_queue != VK_NULL_HANDLE) {
    re_recording_queue = VK_NULL_HANDLE;
    vkQueueWaitIdle(queue);

    VkDevice device = queuesToDevices[queue];

    for (auto& img : images_to_get) {
      size_t sz = bytes_per_pixel[img.format] * img.width * img.height;
      void* image_data;
      if (VK_SUCCESS != vkMapMemory(device, img.transfer_buffer_memory, 0, VK_WHOLE_SIZE, 0, &image_data)) {
        LogMessage(error, std::format("Could not map memory for image"));
        continue;
      }
#if 0
      image_data = tdefl_compress_mem_to_heap(image_data, sz, &sz, TDEFL_HUFFMAN_ONLY);
#endif
      std::string encoded_buffer;
      encoded_buffer.resize(sz * 2);
      size_t d = fast_avx2_base64_encode(encoded_buffer.data(), static_cast<const char*>(image_data), sz);
      encoded_buffer.resize(d);
      auto data = nlohmann::json();
      data["data"] = std::move(encoded_buffer);
      data["width"] = img.width;
      data["height"] = img.height;
      data["format"] = img.format;

      auto str = data.dump();
      SendJson(str.c_str(), str.size());

      vkDestroyImage(device, img.temporary_image, nullptr);
      vkFreeMemory(device, img.temporary_image_memory, nullptr);
      vkDestroyBuffer(device, img.transfer_buffer, nullptr);
      vkFreeMemory(device, img.transfer_buffer_memory, nullptr);
    }
    images_to_get.clear();
  }
  return ret;
}

VKAPI_ATTR void VKAPI_CALL override_vkCmdBeginRenderPass(
    VkCommandBuffer commandBuffer,
    const VkRenderPassBeginInfo* pRenderPassBegin,
    VkSubpassContents contents) {
  vkCmdBeginRenderPass(commandBuffer, pRenderPassBegin, contents);
  if (re_recording_queue != VK_NULL_HANDLE) {
    current_framebuffer = pRenderPassBegin->framebuffer;
    current_renderpass = pRenderPassBegin->renderPass;
  }
}

void dump_image_view(VkDevice device, VkCommandBuffer cb, VkImageView image_view, VkImageLayout layout, uint32_t width, uint32_t height) {
  // Step 1 create new image to match the old one (ish);
  const auto& image_view_create_info = image_view_infos[image_view];
  if (image_view_create_info.viewType > VK_IMAGE_VIEW_TYPE_3D) {
    LogMessage(error, std::format("We do not currently handle cube or array views for dumping view: {}", reinterpret_cast<uintptr_t>(image_view)));
    return;
  }
  if (bytes_per_pixel.count(image_view_create_info.format) == 0) {
    LogMessage(error, std::format("We do not handle the format: {} yet", static_cast<uint32_t>(image_view_create_info.format)));
    return;
  }
  auto subresource_range = image_view_create_info.subresourceRange;

  VkImage image_copy_source = image_view_create_info.image;
  const auto& image_create_info = image_infos[image_view_create_info.image];
  image_copy_data dat;
  dat.width = width;
  dat.height = height;
  dat.format = image_view_create_info.format;

  const auto& memory_props = memory_properties[device];

  // Create the copy destination resources here so that we know if we will fail from OOM
  VkDeviceSize buffer_size = bytes_per_pixel[image_view_create_info.format] * width * height;
  VkBufferCreateInfo buffer_create_info = {
      .sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .size = buffer_size,
      .usage = VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
      .sharingMode = VK_SHARING_MODE_EXCLUSIVE,
      .queueFamilyIndexCount = 0,
      .pQueueFamilyIndices = nullptr};

  if (VK_SUCCESS != vkCreateBuffer(device, &buffer_create_info, nullptr, &dat.transfer_buffer)) {
    LogMessage(error, std::format("Could not allocate buffer for image copy {}", reinterpret_cast<uintptr_t>(image_view)));
    return;
  }

  VkMemoryRequirements buffer_reqs;
  vkGetBufferMemoryRequirements(device, dat.transfer_buffer, &buffer_reqs);

  uint32_t memory_index = 0;
  for (; memory_index < memory_props.memoryTypeCount; ++memory_index) {
    if (!(buffer_reqs.memoryTypeBits & (1 << memory_index))) {
      continue;
    }
    if ((memory_props.memoryTypes[memory_index].propertyFlags &
         VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) !=
        VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
      continue;
    }
    break;
  }
  VkMemoryAllocateInfo buffer_mem_info = {
      .sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
      .pNext = nullptr,
      .allocationSize = buffer_reqs.size,
      .memoryTypeIndex = memory_index};
  if (VK_SUCCESS != vkAllocateMemory(device, &buffer_mem_info, nullptr, &dat.transfer_buffer_memory)) {
    LogMessage(error, std::format("Could not allocate memory for copy buffer: {}", reinterpret_cast<uintptr_t>(image_view)));
    vkDestroyBuffer(device, dat.transfer_buffer, nullptr);
    return;
  }

  vkBindBufferMemory(device, dat.transfer_buffer, dat.transfer_buffer_memory, 0);

  if (image_create_info.samples != VK_SAMPLE_COUNT_1_BIT) {
    VkImageCreateInfo ci = {
        .sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .imageType = static_cast<VkImageType>(image_view_create_info.viewType),
        .format = image_view_create_info.format,
        .extent = VkExtent3D{width, height, 1},
        .mipLevels = 1,
        .arrayLayers = 1,
        .samples = VK_SAMPLE_COUNT_1_BIT,
        .tiling = VK_IMAGE_TILING_OPTIMAL,
        .usage = VK_IMAGE_USAGE_TRANSFER_DST_BIT,
        .sharingMode = VK_SHARING_MODE_EXCLUSIVE,
        .queueFamilyIndexCount = 0,
        .pQueueFamilyIndices = nullptr,
        .initialLayout = VK_IMAGE_LAYOUT_UNDEFINED};
    if (VK_SUCCESS != vkCreateImage(device, &ci, nullptr, &dat.temporary_image)) {
      vkDestroyBuffer(device, dat.transfer_buffer, nullptr);
      vkFreeMemory(device, dat.transfer_buffer_memory, nullptr);
      LogMessage(error, std::format("Error creating resolve image for view: {}", reinterpret_cast<uintptr_t>(image_view)));
      return;
    }

    VkMemoryRequirements reqs;
    vkGetImageMemoryRequirements(device, dat.temporary_image, &reqs);

    memory_index = 0;
    for (; memory_index < memory_props.memoryTypeCount; ++memory_index) {
      if (!(reqs.memoryTypeBits & (1 << memory_index))) {
        continue;
      }
      // This one can be device local
      //  if ((memory_props.memoryTypes[memory_index].propertyFlags &
      //    VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT) !=
      //    VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT) {
      //    continue;
      //  }
      break;
    }
    VkMemoryAllocateInfo info = {
        .sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
        .pNext = nullptr,
        .allocationSize = reqs.size,
        .memoryTypeIndex = memory_index};
    if (VK_SUCCESS != vkAllocateMemory(device, &info, nullptr, &dat.temporary_image_memory)) {
      LogMessage(error, std::format("Could not allocate memory for resove image: {}", reinterpret_cast<uintptr_t>(image_view)));
      vkDestroyImage(device, dat.temporary_image, nullptr);
      vkDestroyBuffer(device, dat.transfer_buffer, nullptr);
      vkFreeMemory(device, dat.transfer_buffer_memory, nullptr);
      return;
    }

    vkBindImageMemory(device, dat.temporary_image, dat.temporary_image_memory, 0);

    VkImageMemoryBarrier barrier = {
        .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
        .pNext = nullptr,
        .srcAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT | VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT,
        .dstAccessMask = VK_ACCESS_TRANSFER_READ_BIT,
        .oldLayout = layout,
        .newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
        .srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
        .dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
        .image = image_view_create_info.image,
        .subresourceRange = image_view_create_info.subresourceRange};

    vkCmdPipelineBarrier(cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, 0, 0, nullptr, 0, nullptr, 1, &barrier);

    VkImageMemoryBarrier barrier2 = {
        .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
        .pNext = nullptr,
        .srcAccessMask = 0,
        .dstAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT,
        .oldLayout = VK_IMAGE_LAYOUT_UNDEFINED,
        .newLayout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
        .srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
        .dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
        .image = dat.temporary_image,
        .subresourceRange = image_view_create_info.subresourceRange};

    vkCmdPipelineBarrier(cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, 0, 0, nullptr, 0, nullptr, 1, &barrier);
    layout = VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL;

    VkImageResolve resolve{
        .srcSubresource = VkImageSubresourceLayers{
            .aspectMask = image_view_create_info.subresourceRange.aspectMask,
            .mipLevel = image_view_create_info.subresourceRange.baseMipLevel,
            .baseArrayLayer = image_view_create_info.subresourceRange.baseArrayLayer,
            .layerCount = 1},
        .srcOffset = VkOffset3D(0, 0, 0),
        .dstSubresource = VkImageSubresourceLayers{.aspectMask = image_view_create_info.subresourceRange.aspectMask, .mipLevel = image_view_create_info.subresourceRange.baseMipLevel, .baseArrayLayer = 0, .layerCount = 1},
        .dstOffset = VkOffset3D(0, 0, 0),
        .extent = VkExtent3D(width, height, 1)};
    vkCmdResolveImage(cb, image_view_create_info.image, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, dat.temporary_image, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &resolve);

    // Put the original image back
    barrier.srcAccessMask = VK_ACCESS_TRANSFER_READ_BIT;
    barrier.dstAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT | VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;
    barrier.newLayout = layout;
    barrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;

    vkCmdPipelineBarrier(cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, 0, 0, nullptr, 0, nullptr, 1, &barrier);
    image_copy_source = dat.temporary_image;
    subresource_range.baseMipLevel = 0;
    subresource_range.baseArrayLayer = 0;
    subresource_range.layerCount = 1;
    subresource_range.levelCount = 1;
  }

  VkImageMemoryBarrier barrier = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
      .pNext = nullptr,
      .srcAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT | VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT | VK_ACCESS_TRANSFER_WRITE_BIT,
      .dstAccessMask = VK_ACCESS_TRANSFER_READ_BIT,
      .oldLayout = layout,
      .newLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL,
      .srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
      .dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
      .image = image_copy_source,
      .subresourceRange = subresource_range};

  vkCmdPipelineBarrier(cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, 0, 0, nullptr, 0, nullptr, 1, &barrier);

  VkBufferImageCopy region = {
      .bufferOffset = 0,
      .bufferRowLength = 0,
      .bufferImageHeight = 0,
      .imageSubresource = VkImageSubresourceLayers{
          subresource_range.aspectMask,
          subresource_range.baseMipLevel,
          subresource_range.baseArrayLayer,
          1,
      },
      .imageOffset = VkOffset3D{0, 0, 0},
      .imageExtent = VkExtent3D{width, height, 1}};

  vkCmdCopyImageToBuffer(cb, image_copy_source, VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL, dat.transfer_buffer, 1, &region);

  barrier.newLayout = layout;
  barrier.oldLayout = VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL;
  barrier.dstAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT | VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT | VK_ACCESS_TRANSFER_WRITE_BIT,
  barrier.srcAccessMask = VK_ACCESS_TRANSFER_READ_BIT,
  vkCmdPipelineBarrier(cb, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, VK_PIPELINE_STAGE_ALL_COMMANDS_BIT, 0, 0, nullptr, 0, nullptr, 1, &barrier);
  images_to_get.push_back(dat);
}

VKAPI_ATTR void VKAPI_CALL OnCommandBufferSplit(VkCommandBuffer cb) {
  LogMessage(debug, std::format("Inserting into cb {}", reinterpret_cast<uintptr_t>(cb)));
  const auto& rp_data = render_pass_infos[current_renderpass];
  const auto& fb_data = framebuffer_infos[current_framebuffer];
  VkDevice device = queuesToDevices[re_recording_queue];
  for (auto& x : rp_data.subpasses[0].color_attachments) {
    if (x.attachment == VK_ATTACHMENT_UNUSED) {
      continue;
    }
    dump_image_view(device, cb, fb_data.image_views[x.attachment], x.layout, fb_data.width, fb_data.height);
  }

  if (rp_data.subpasses[0].depth_attachment && rp_data.subpasses[0].depth_attachment->attachment != VK_ATTACHMENT_UNUSED) {
    uint32_t attachment = rp_data.subpasses[0].depth_attachment->attachment;
    dump_image_view(device, cb, fb_data.image_views[attachment], rp_data.subpasses[0].depth_attachment->layout, fb_data.width, fb_data.height);
  }
}