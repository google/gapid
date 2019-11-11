/*
 * Copyright (c) 2015-2016 The Khronos Group Inc.
 * Copyright (c) 2015-2016 Valve Corporation
 * Copyright (c) 2015-2016 LunarG, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Author: Chia-I Wu <olv@lunarg.com>
 * Author: Courtney Goeltzenleuchter <courtney@LunarG.com>
 * Author: Ian Elliott <ian@LunarG.com>
 * Author: Ian Elliott <ianelliott@google.com>
 * Author: Jon Ashburn <jon@lunarg.com>
 * Author: Gwan-gyeong Mun <elongbug@gmail.com>
 * Author: Tony Barbour <tony@LunarG.com>
 * Author: Bill Hollings <bill.hollings@brenwill.com>
 */

#include <stdio.h>
#include <stdarg.h>
#include <stdlib.h>
#include <string.h>
#include <signal.h>
#include <vector>

#include <android/log.h>
#include <vulkan/vk_platform.h>
#include <unistd.h>

#include "cube.h"
#include "gettime.h"
#include "inttypes.h"

#define LOG_TAG "VKCube"
#define APP_NAME "Vulkan Cube"
#define ALOGD(...) __android_log_print(ANDROID_LOG_DEBUG, LOG_TAG, __VA_ARGS__)
#define ALOGE(...) __android_log_print(ANDROID_LOG_ERROR, LOG_TAG, __VA_ARGS__)

#define ASSERT(cond)                                                                           \
    if (!(cond)) {                                                                             \
        __android_log_assert(#cond, LOG_TAG, "Error: " #cond " at " __FILE__ ":%d", __LINE__); \
    }

#define ERR_EXIT(err_msg, err_class)                                    \
    do {                                                                \
        ((void)__android_log_print(ANDROID_LOG_INFO, "Cube", err_msg)); \
        exit(1);                                                        \
    } while (false)

#define GET_INSTANCE_PROC_ADDR(entrypoint)                                                              \
    do {                                                                                                         \
        mVkHelper.vk##entrypoint = reinterpret_cast<PFN_vk##entrypoint>(vkGetInstanceProcAddr(mInstance, "vk" #entrypoint));             \
        if (mVkHelper.vk##entrypoint == nullptr) {                                                                   \
            ERR_EXIT("vkGetInstanceProcAddr failed to find vk" #entrypoint, "vkGetInstanceProcAddr Failure"); \
        }                                                                                                     \
    } while (false)

#define GET_DEVICE_PROC_ADDR(entrypoint)                                                                    \
    do {                                                                                                            \
        mVkHelper.vk##entrypoint = (PFN_vk##entrypoint)mVkHelper.vkGetDeviceProcAddr(mDevice, "vk" #entrypoint);                                \
        if (mVkHelper.vk##entrypoint == nullptr) {                                                                      \
            ERR_EXIT("vkGetDeviceProcAddr failed to find vk" #entrypoint, "vkGetDeviceProcAddr Failure");        \
        }                                                                                                        \
    } while (false)

const char *texFile = "gapid.ppm";

// Mesh and VertexFormat Data
// clang-format off
static const float gVertexBufferData[] = {
    -1.0f,-1.0f,-1.0f,  // -X side
    -1.0f,-1.0f, 1.0f,
    -1.0f, 1.0f, 1.0f,
    -1.0f, 1.0f, 1.0f,
    -1.0f, 1.0f,-1.0f,
    -1.0f,-1.0f,-1.0f,

    -1.0f,-1.0f,-1.0f,  // -Z side
    1.0f, 1.0f,-1.0f,
    1.0f,-1.0f,-1.0f,
    -1.0f,-1.0f,-1.0f,
    -1.0f, 1.0f,-1.0f,
    1.0f, 1.0f,-1.0f,

    -1.0f,-1.0f,-1.0f,  // -Y side
    1.0f,-1.0f,-1.0f,
    1.0f,-1.0f, 1.0f,
    -1.0f,-1.0f,-1.0f,
    1.0f,-1.0f, 1.0f,
    -1.0f,-1.0f, 1.0f,

    -1.0f, 1.0f,-1.0f,  // +Y side
    -1.0f, 1.0f, 1.0f,
    1.0f, 1.0f, 1.0f,
    -1.0f, 1.0f,-1.0f,
    1.0f, 1.0f, 1.0f,
    1.0f, 1.0f,-1.0f,

    1.0f, 1.0f,-1.0f,  // +X side
    1.0f, 1.0f, 1.0f,
    1.0f,-1.0f, 1.0f,
    1.0f,-1.0f, 1.0f,
    1.0f,-1.0f,-1.0f,
    1.0f, 1.0f,-1.0f,

    -1.0f, 1.0f, 1.0f,  // +Z side
    -1.0f,-1.0f, 1.0f,
    1.0f, 1.0f, 1.0f,
    -1.0f,-1.0f, 1.0f,
    1.0f,-1.0f, 1.0f,
    1.0f, 1.0f, 1.0f,
};

static const float gUvBufferData[] = {
    0.0f, 1.0f,  // -X side
    1.0f, 1.0f,
    1.0f, 0.0f,
    1.0f, 0.0f,
    0.0f, 0.0f,
    0.0f, 1.0f,

    1.0f, 1.0f,  // -Z side
    0.0f, 0.0f,
    0.0f, 1.0f,
    1.0f, 1.0f,
    1.0f, 0.0f,
    0.0f, 0.0f,

    1.0f, 0.0f,  // -Y side
    1.0f, 1.0f,
    0.0f, 1.0f,
    1.0f, 0.0f,
    0.0f, 1.0f,
    0.0f, 0.0f,

    1.0f, 0.0f,  // +Y side
    0.0f, 0.0f,
    0.0f, 1.0f,
    1.0f, 0.0f,
    0.0f, 1.0f,
    1.0f, 1.0f,

    1.0f, 0.0f,  // +X side
    0.0f, 0.0f,
    0.0f, 1.0f,
    0.0f, 1.0f,
    1.0f, 1.0f,
    1.0f, 0.0f,

    0.0f, 0.0f,  // +Z side
    0.0f, 1.0f,
    1.0f, 0.0f,
    0.0f, 1.0f,
    1.0f, 1.0f,
    1.0f, 0.0f,
};
// clang-format on

bool Cube::memoryTypeFromProperties(uint32_t typeBits,
                                    VkFlags requirementsMask,
                                    uint32_t* typeIndex) {
  for (uint32_t i = 0; i < VK_MAX_MEMORY_TYPES; i++) {
    if ((typeBits & 1) == 1) {
      if ((mPhysicalDevicememoryProperties.memoryTypes[i].propertyFlags & requirementsMask) == requirementsMask) {
        *typeIndex = i;
        return true;
      }
    }
    typeBits >>= 1;
  }
  return false;
}

// Read ppm file and convert into RGBA texture image
bool Cube::loadTextureFromPPM(const char* fileName, void* data, VkSubresourceLayout* layout, int32_t* width, int32_t* height) {
  AAsset* file = AAssetManager_open(mAppState->assetManager, fileName, AASSET_MODE_BUFFER);
  size_t fileLength = AAsset_getLength(file);
  auto fileContent = new unsigned char[fileLength];
  AAsset_read(file, fileContent, fileLength);
  AAsset_close(file);

  uint8_t* rgbaData = static_cast<uint8_t*>(data);
  char *cPtr = (char *)fileContent;
  if ((unsigned char *)cPtr >= (fileContent + fileLength) || strncmp(cPtr, "P6\n", 3)) {
    return false;
  }
  while (strncmp(cPtr++, "\n", 1)) {}
  sscanf(cPtr, "%u %u", width, height);
  if (rgbaData == nullptr) {
    return true;
  }
  while (strncmp(cPtr++, "\n", 1)) {}
  if ((unsigned char *)cPtr >= (fileContent + fileLength) || strncmp(cPtr, "255\n", 4)) {
    return false;
  }
  while (strncmp(cPtr++, "\n", 1)) {}
  for (int y = 0; y < *height; y++) {
    uint8_t *rowPtr = rgbaData;
    for (int x = 0; x < *width; x++) {
      memcpy(rowPtr, cPtr, 3);
      rowPtr[3] = 255; // Alpha of 1
      rowPtr += 4;
      cPtr += 3;
    }
    rgbaData += layout->rowPitch;
  }
  return true;
}

void Cube::loadShaderFromFile(const char* filePath, VkShaderModule* outShader) {
  AAssetManager* assetManager = mAppState->assetManager;
  AAsset* file = AAssetManager_open(assetManager, filePath, AASSET_MODE_BUFFER);
  auto fileLength = (size_t)AAsset_getLength(file);
  std::vector<char> fileContent(fileLength);
  AAsset_read(file, fileContent.data(), fileLength);
  AAsset_close(file);
  const VkShaderModuleCreateInfo shaderModuleCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .codeSize = fileLength,
      .pCode = (const uint32_t*)(fileContent.data()),
  };
  mVkHelper.vkCreateShaderModule(mDevice, &shaderModuleCreateInfo, nullptr, outShader);
}

void Cube::flushInitCommands() {
  // This function could get called twice if the texture uses a staging buffer
  // In that case the second call should be ignored
  if (mCommandBuffer == VK_NULL_HANDLE) {
    return;
  }
  ASSERT(VK_SUCCESS == mVkHelper.vkEndCommandBuffer(mCommandBuffer));
  ASSERT(VK_SUCCESS == mVkHelper.vkEndCommandBuffer(mCommandBuffer));
  VkFence fence;
  VkFenceCreateInfo fenceCreateInfo = {.sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO, .pNext = nullptr, .flags = 0};
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateFence(mDevice, &fenceCreateInfo, nullptr, &fence));
  const VkCommandBuffer commandBuffers[] = {mCommandBuffer};
  VkSubmitInfo submitInfo = {
      .sType = VK_STRUCTURE_TYPE_SUBMIT_INFO,
      .pNext = nullptr,
      .waitSemaphoreCount = 0,
      .pWaitSemaphores = nullptr,
      .pWaitDstStageMask = nullptr,
      .commandBufferCount = 1,
      .pCommandBuffers = commandBuffers,
      .signalSemaphoreCount = 0,
      .pSignalSemaphores = nullptr
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkQueueSubmit(mGraphicsQueue, 1, &submitInfo, fence));
  ASSERT(VK_SUCCESS == mVkHelper.vkWaitForFences(mDevice, 1, &fence, VK_TRUE, UINT64_MAX));
  mVkHelper.vkFreeCommandBuffers(mDevice, mCommandPool, 1, commandBuffers);
  mVkHelper.vkDestroyFence(mDevice, fence, nullptr);
  mCommandBuffer = VK_NULL_HANDLE;
}

void Cube::destroyTexture(TextureObject* textureObject) {
  // clean up staging resources
  mVkHelper.vkFreeMemory(mDevice, textureObject->deviceMemory, nullptr);
  if (textureObject->image) mVkHelper.vkDestroyImage(mDevice, textureObject->image, nullptr);
  if (textureObject->buffer) mVkHelper.vkDestroyBuffer(mDevice, textureObject->buffer, nullptr);
}

void Cube::buildImageOwnershipCommand(int index) {
  const VkCommandBufferBeginInfo commandBufferBeginInfo = {
      .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
      .pNext = nullptr,
      .flags = VK_COMMAND_BUFFER_USAGE_SIMULTANEOUS_USE_BIT,
      .pInheritanceInfo = nullptr,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkBeginCommandBuffer(mSwapchainImageResources[index].graphicsToPresentCommandBuffer, &commandBufferBeginInfo));
  VkImageMemoryBarrier imageMemoryBarrier = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
      .pNext = nullptr,
      .srcAccessMask = 0,
      .dstAccessMask = 0,
      .oldLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
      .newLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
      .srcQueueFamilyIndex = mGraphicsQueueFamilyIndex,
      .dstQueueFamilyIndex = mPresentQueueFamilyIndex,
      .image = mSwapchainImageResources[index].image,
      .subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1}
  };
  mVkHelper.vkCmdPipelineBarrier(mSwapchainImageResources[index].graphicsToPresentCommandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
                       VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, 0, 0, nullptr, 0, nullptr, 1, &imageMemoryBarrier);
  ASSERT(VK_SUCCESS == mVkHelper.vkEndCommandBuffer(mSwapchainImageResources[index].graphicsToPresentCommandBuffer));
}

void Cube::updateDataBuffer() {
  mat4x4 modelMatrix, viewPortMatrix;
  uint8_t *pData;

  mat4x4_mul(viewPortMatrix, mProjectMatrix, mViewMatrix);

  // Set scale.
  mat4x4_identity(mModelMatrix);
  mModelMatrix[0][0] = mScale;
  mModelMatrix[1][1] = mScale;
  mModelMatrix[2][2] = mScale;
  uint64_t longTime = getTimeInNanoseconds();
  longTime = ((longTime << 16) >> 16) >> 18; // Keep only middle bits.
  mSpinAngle = longTime * mSpinSpeed;

  // Rotate around the Y axis.
  mat4x4_dup(modelMatrix, mModelMatrix);
  mat4x4_rotate(mModelMatrix, modelMatrix, 0.0f, 1.0f, 0.0f, mSpinAngle);
  mat4x4_mul(mUniform.mvp, viewPortMatrix, mModelMatrix);
  ASSERT(VK_SUCCESS == mVkHelper.vkMapMemory(mDevice, mSwapchainImageResources[mCurrentBuffer].uniformDeviceMemory, 0, VK_WHOLE_SIZE, 0,
                    (void **)&pData));
  memcpy(pData, &mUniform, sizeof(mUniform));
  mVkHelper.vkUnmapMemory(mDevice, mSwapchainImageResources[mCurrentBuffer].uniformDeviceMemory);
}

void Cube::buildDrawCommands(VkCommandBuffer commandBuffer) {
  const VkCommandBufferBeginInfo commandBufferBeginInfo = {
      .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
      .pNext = nullptr,
      .flags = VK_COMMAND_BUFFER_USAGE_SIMULTANEOUS_USE_BIT,
      .pInheritanceInfo = nullptr,
  };
  const VkClearValue clearValues[2] = {
      [0] = {
          .color.float32 = {0.2f, 0.2f, 0.2f, 0.2f},
      },
      [1] = {
          .depthStencil = {1.0f, 0},
      },
  };
  const VkRenderPassBeginInfo renderPassBeginInfo = {
      .sType = VK_STRUCTURE_TYPE_RENDER_PASS_BEGIN_INFO,
      .pNext = nullptr,
      .renderPass = mRenderPass,
      .framebuffer = mSwapchainImageResources[mCurrentBuffer].framebuffer,
      .renderArea.offset.x = 0,
      .renderArea.offset.y = 0,
      .renderArea.extent.width = mWidth,
      .renderArea.extent.height = mHeight,
      .clearValueCount = 2,
      .pClearValues = clearValues,
  };

  ASSERT(VK_SUCCESS == mVkHelper.vkBeginCommandBuffer(commandBuffer, &commandBufferBeginInfo));
  mVkHelper.vkCmdBeginRenderPass(commandBuffer, &renderPassBeginInfo, VK_SUBPASS_CONTENTS_INLINE);

  mVkHelper.vkCmdBindPipeline(commandBuffer, VK_PIPELINE_BIND_POINT_GRAPHICS, mPipeline);
  mVkHelper.vkCmdBindDescriptorSets(commandBuffer, VK_PIPELINE_BIND_POINT_GRAPHICS, mPipelineLayout, 0, 1,
                                    &mSwapchainImageResources[mCurrentBuffer].descriptorSet, 0, nullptr);
  VkViewport viewport;
  memset(&viewport, 0, sizeof(viewport));
  viewport.height = (float)mHeight;
  viewport.width = (float)mWidth;
  viewport.minDepth = (float)0.0f;
  viewport.maxDepth = (float)1.0f;
  mVkHelper.vkCmdSetViewport(commandBuffer, 0, 1, &viewport);

  VkRect2D scissor;
  memset(&scissor, 0, sizeof(scissor));
  scissor.extent.width = mWidth;
  scissor.extent.height = mHeight;
  scissor.offset.x = 0;
  scissor.offset.y = 0;
  mVkHelper.vkCmdSetScissor(commandBuffer, 0, 1, &scissor);
  mVkHelper.vkCmdDraw(commandBuffer, 12 * 3, 1, 0, 0);

  // Note that ending the renderpass changes the image's layout from
  // COLOR_ATTACHMENT_OPTIMAL to PRESENT_SRC_KHR
  mVkHelper.vkCmdEndRenderPass(commandBuffer);
  if (mSeparatePresentQueue) {
    // We have to transfer ownership from the graphics queue family to the
    // present queue family to be able to present.  Note that we don't have
    // to transfer from present queue family back to graphics queue family at
    // the start of the next frame because we don't care about the image's
    // contents at that point.
    VkImageMemoryBarrier imageMemoryBarrier = {
        .sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
        .pNext = nullptr,
        .srcAccessMask = 0,
        .dstAccessMask = 0,
        .oldLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
        .newLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
        .srcQueueFamilyIndex = mGraphicsQueueFamilyIndex,
        .dstQueueFamilyIndex = mPresentQueueFamilyIndex,
        .image = mSwapchainImageResources[mCurrentBuffer].image,
        .subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1}
    };
    mVkHelper.vkCmdPipelineBarrier(commandBuffer, VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT,
                                   VK_PIPELINE_STAGE_BOTTOM_OF_PIPE_BIT, 0, 0, nullptr, 0,
                                   nullptr, 1, &imageMemoryBarrier);
  }
  ASSERT(VK_SUCCESS == mVkHelper.vkEndCommandBuffer(commandBuffer));
}

void Cube::updateDrawCommands() {
  // Rerecord draw commands.
  mVkHelper.vkDeviceWaitIdle(mDevice);
  uint32_t currentBuffer = mCurrentBuffer;
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    mCurrentBuffer = i;
    buildDrawCommands(mSwapchainImageResources[i].commandBuffer);
  }
  mCurrentBuffer = currentBuffer;
}

void Cube::draw() {
  // Ensure no more than mSwapchainImageCount renderings are outstanding
  mVkHelper.vkWaitForFences(mDevice, 1, &mFences[mFrameIndex], VK_TRUE, UINT64_MAX);
  mVkHelper.vkResetFences(mDevice, 1, &mFences[mFrameIndex]);

  // TODO(lpy) Only draw when dirty once command buffers are instrumented correctly.
  updateDrawCommands();
  VkResult err;
  do {
    // Get the index of the next available swapchain image:
    err = mVkHelper.vkAcquireNextImageKHR(mDevice, mSwapchain, UINT64_MAX,
                                      mImageAcquiredSemaphores[mFrameIndex], VK_NULL_HANDLE, &mCurrentBuffer);

    if (err == VK_ERROR_OUT_OF_DATE_KHR) {
      // mSwapchain is out of date (e.g. the window was resized) and
      // must be recreated:
      resize();
    } else if (err == VK_SUBOPTIMAL_KHR) {
      // mSwapchain is not as optimal as it could be, but the platform's
      // presentation engine will still present the image correctly.
      break;
    } else {
      ASSERT(VK_SUCCESS == err);
    }
  } while (err != VK_SUCCESS);
  updateDataBuffer();

  // Wait for the image acquired semaphore to be signaled to ensure
  // that the image won't be rendered to until the presentation
  // engine has fully released ownership to the application, and it is
  // okay to render to the image.
  VkPipelineStageFlags pipelineStageFlags;
  VkSubmitInfo submitInfo;
  submitInfo.sType = VK_STRUCTURE_TYPE_SUBMIT_INFO;
  submitInfo.pNext = nullptr;
  submitInfo.pWaitDstStageMask = &pipelineStageFlags;
  pipelineStageFlags = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT;
  submitInfo.waitSemaphoreCount = 1;
  submitInfo.pWaitSemaphores = &mImageAcquiredSemaphores[mFrameIndex];
  submitInfo.commandBufferCount = 1;
  submitInfo.pCommandBuffers = &mSwapchainImageResources[mCurrentBuffer].commandBuffer;
  submitInfo.signalSemaphoreCount = 1;
  submitInfo.pSignalSemaphores = &mDrawCompleteSemaphores[mFrameIndex];
  ASSERT(VK_SUCCESS == mVkHelper.vkQueueSubmit(mGraphicsQueue, 1, &submitInfo, mFences[mFrameIndex]));
  if (mSeparatePresentQueue) {
    // If we are using separate queues, change image ownership to the
    // present queue before presenting, waiting for the draw complete
    // semaphore and signalling the ownership released semaphore when finished
    VkFence nullFence = VK_NULL_HANDLE;
    pipelineStageFlags = VK_PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT;
    submitInfo.waitSemaphoreCount = 1;
    submitInfo.pWaitSemaphores = &mDrawCompleteSemaphores[mFrameIndex];
    submitInfo.commandBufferCount = 1;
    submitInfo.pCommandBuffers = &mSwapchainImageResources[mCurrentBuffer].graphicsToPresentCommandBuffer;
    submitInfo.signalSemaphoreCount = 1;
    submitInfo.pSignalSemaphores = &mImageOwnershipSemaphores[mFrameIndex];
    ASSERT(VK_SUCCESS == mVkHelper.vkQueueSubmit(mPresentQueue, 1, &submitInfo, nullFence));
  }

  // If we are using separate queues we have to wait for image ownership,
  // otherwise wait for draw complete
  VkPresentInfoKHR present = {
      .sType = VK_STRUCTURE_TYPE_PRESENT_INFO_KHR,
      .pNext = nullptr,
      .waitSemaphoreCount = 1,
      .pWaitSemaphores = (mSeparatePresentQueue) ? &mImageOwnershipSemaphores[mFrameIndex]
                                                        : &mDrawCompleteSemaphores[mFrameIndex],
      .swapchainCount = 1,
      .pSwapchains = &mSwapchain,
      .pImageIndices = &mCurrentBuffer,
  };
  err = mVkHelper.vkQueuePresentKHR(mPresentQueue, &present);
  mFrameIndex += 1;
  mFrameIndex %= mSwapchainImageCount;
  if (err == VK_ERROR_OUT_OF_DATE_KHR) {
    // mSwapchain is out of date (e.g. the window was resized) and
    // must be recreated:
    resize();
  } else if (err == VK_SUBOPTIMAL_KHR) {
    // mSwapchain is not as optimal as it could be, but the platform's
    // presentation engine will still present the image correctly.
  } else {
    ASSERT(VK_SUCCESS == err);
  }
}

void Cube::prepareDepth() {
  const VkFormat depthFormat = VK_FORMAT_D16_UNORM;
  const VkImageCreateInfo imageCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
      .pNext = nullptr,
      .imageType = VK_IMAGE_TYPE_2D,
      .format = depthFormat,
      .extent = {mWidth, mHeight, 1},
      .mipLevels = 1,
      .arrayLayers = 1,
      .samples = VK_SAMPLE_COUNT_1_BIT,
      .tiling = VK_IMAGE_TILING_OPTIMAL,
      .usage = VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT,
      .flags = 0,
  };
  mDepth.format = depthFormat;

  ASSERT(VK_SUCCESS == mVkHelper.vkCreateImage(mDevice, &imageCreateInfo, nullptr, &mDepth.image));

  VkMemoryRequirements memoryRequirements;
  mVkHelper.vkGetImageMemoryRequirements(mDevice, mDepth.image, &memoryRequirements);
  mDepth.memoryAllocationInfo.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
  mDepth.memoryAllocationInfo.pNext = nullptr;
  mDepth.memoryAllocationInfo.allocationSize = memoryRequirements.size;
  mDepth.memoryAllocationInfo.memoryTypeIndex = 0;

  ASSERT(memoryTypeFromProperties(memoryRequirements.memoryTypeBits,
                                  VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT,
                                  &mDepth.memoryAllocationInfo.memoryTypeIndex));
  ASSERT(VK_SUCCESS == mVkHelper.vkAllocateMemory(mDevice, &mDepth.memoryAllocationInfo, nullptr, &mDepth.deviceMemory));
  ASSERT(VK_SUCCESS == mVkHelper.vkBindImageMemory(mDevice, mDepth.image, mDepth.deviceMemory, 0));

  const VkImageViewCreateInfo imageViewCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
      .pNext = nullptr,
      .format = depthFormat,
      .subresourceRange =
          {.aspectMask = VK_IMAGE_ASPECT_DEPTH_BIT, .baseMipLevel = 0, .levelCount = 1, .baseArrayLayer = 0, .layerCount = 1},
      .flags = 0,
      .viewType = VK_IMAGE_VIEW_TYPE_2D,
      .image = mDepth.image,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateImageView(mDevice, &imageViewCreateInfo, nullptr, &mDepth.view));
}

void Cube::prepareTextureBuffer(const char* filename, TextureObject* textureObject) {
  int32_t texWidth;
  int32_t texHeight;

  if (!loadTextureFromPPM(filename, nullptr, nullptr, &texWidth, &texHeight)) {
    ERR_EXIT("Failed to load textures", "Load Texture Failure");
  }
  textureObject->width = texWidth;
  textureObject->height = texHeight;
  const VkBufferCreateInfo bufferCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .size = static_cast<VkDeviceSize>(texWidth * texHeight * 4),
      .usage = VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
      .sharingMode = VK_SHARING_MODE_EXCLUSIVE,
      .queueFamilyIndexCount = 0,
      .pQueueFamilyIndices = nullptr
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateBuffer(mDevice, &bufferCreateInfo, nullptr, &textureObject->buffer));

  VkMemoryRequirements memoryRequirements;
  mVkHelper.vkGetBufferMemoryRequirements(mDevice, textureObject->buffer, &memoryRequirements);

  textureObject->memoryAllocationInfo.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
  textureObject->memoryAllocationInfo.pNext = nullptr;
  textureObject->memoryAllocationInfo.allocationSize = memoryRequirements.size;
  textureObject->memoryAllocationInfo.memoryTypeIndex = 0;

  VkFlags requirements = VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT;
  ASSERT(memoryTypeFromProperties(memoryRequirements.memoryTypeBits,
                                  requirements,
                                  &textureObject->memoryAllocationInfo.memoryTypeIndex));

  ASSERT(VK_SUCCESS == mVkHelper.vkAllocateMemory(mDevice, &textureObject->memoryAllocationInfo, nullptr, &(textureObject->deviceMemory)));

  ASSERT(VK_SUCCESS == mVkHelper.vkBindBufferMemory(mDevice, textureObject->buffer, textureObject->deviceMemory, 0));

  void *data;
  VkSubresourceLayout layout = {
      .rowPitch = static_cast<VkDeviceSize>(texWidth * 4),
  };

  ASSERT(VK_SUCCESS == mVkHelper.vkMapMemory(mDevice, textureObject->deviceMemory, 0, textureObject->memoryAllocationInfo.allocationSize, 0, &data));

  if (!loadTextureFromPPM(filename, data, &layout, &texWidth, &texHeight)) {
    fprintf(stderr, "Error loading texture: %s\n", filename);
  }
  mVkHelper.vkUnmapMemory(mDevice, textureObject->deviceMemory);
}

void Cube::prepareTextureImage(const char* filename, TextureObject* textureObject,
                               VkImageTiling tiling, VkImageUsageFlags usage, VkFlags requiredProperties) {
  const VkFormat textureFormat = VK_FORMAT_R8G8B8A8_UNORM;
  int32_t textureWidth;
  int32_t textureHeight;

  if (!loadTextureFromPPM(filename, nullptr, nullptr, &textureWidth, &textureHeight)) {
    ERR_EXIT("Failed to load textures", "Load Texture Failure");
  }
  textureObject->width = textureWidth;
  textureObject->height = textureHeight;
  const VkImageCreateInfo image_create_info = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_CREATE_INFO,
      .pNext = nullptr,
      .imageType = VK_IMAGE_TYPE_2D,
      .format = textureFormat,
      .extent = {static_cast<uint32_t>(textureWidth), static_cast<uint32_t>(textureHeight), 1},
      .mipLevels = 1,
      .arrayLayers = 1,
      .samples = VK_SAMPLE_COUNT_1_BIT,
      .tiling = tiling,
      .usage = usage,
      .flags = 0,
      .initialLayout = VK_IMAGE_LAYOUT_PREINITIALIZED,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateImage(mDevice, &image_create_info, nullptr, &textureObject->image));

  VkMemoryRequirements memoryRequirements;
  mVkHelper.vkGetImageMemoryRequirements(mDevice, textureObject->image, &memoryRequirements);
  textureObject->memoryAllocationInfo.sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO;
  textureObject->memoryAllocationInfo.pNext = nullptr;
  textureObject->memoryAllocationInfo.allocationSize = memoryRequirements.size;
  textureObject->memoryAllocationInfo.memoryTypeIndex = 0;
  ASSERT(memoryTypeFromProperties(memoryRequirements.memoryTypeBits,
                                  requiredProperties,
                                  &textureObject->memoryAllocationInfo.memoryTypeIndex));
  ASSERT(VK_SUCCESS == mVkHelper.vkAllocateMemory(mDevice, &textureObject->memoryAllocationInfo, nullptr, &(textureObject->deviceMemory)));
  ASSERT(VK_SUCCESS == mVkHelper.vkBindImageMemory(mDevice, textureObject->image, textureObject->deviceMemory, 0));
  if (requiredProperties & VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
    const VkImageSubresource imageSubresource = {
        .aspectMask = VK_IMAGE_ASPECT_COLOR_BIT,
        .mipLevel = 0,
        .arrayLayer = 0,
    };
    VkSubresourceLayout subresourceLayout;
    void *data;
    mVkHelper.vkGetImageSubresourceLayout(mDevice, textureObject->image, &imageSubresource, &subresourceLayout);
    ASSERT(VK_SUCCESS == mVkHelper.vkMapMemory(mDevice, textureObject->deviceMemory, 0, textureObject->memoryAllocationInfo.allocationSize, 0, &data));
    if (!loadTextureFromPPM(filename, data, &subresourceLayout, &textureWidth, &textureHeight)) {
      fprintf(stderr, "Error loading texture: %s\n", filename);
    }
    mVkHelper.vkUnmapMemory(mDevice, textureObject->deviceMemory);
  }
  textureObject->imageLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL;
}

void Cube::setImageLayout(VkImage image,
                          VkImageAspectFlags aspectMask,
                          VkImageLayout oldImageLayout,
                          VkImageLayout newImageLayout,
                          VkAccessFlagBits srcAccessMask,
                          VkPipelineStageFlags sourceStages,
                          VkPipelineStageFlags destinationStages) {
  ASSERT(mCommandBuffer);
  VkImageMemoryBarrier imageMemoryBarrier = {.sType = VK_STRUCTURE_TYPE_IMAGE_MEMORY_BARRIER,
      .pNext = nullptr,
      .srcAccessMask = srcAccessMask,
      .dstAccessMask = 0,
      .srcQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
      .dstQueueFamilyIndex = VK_QUEUE_FAMILY_IGNORED,
      .oldLayout = oldImageLayout,
      .newLayout = newImageLayout,
      .image = image,
      .subresourceRange = {aspectMask, 0, 1, 0, 1}};
  switch (newImageLayout) {
    case VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL:
      // Make sure anything that was copying from this image has completed
      imageMemoryBarrier.dstAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT;
      break;
    case VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL:
      imageMemoryBarrier.dstAccessMask = VK_ACCESS_COLOR_ATTACHMENT_WRITE_BIT;
      break;
    case VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL:
      imageMemoryBarrier.dstAccessMask = VK_ACCESS_DEPTH_STENCIL_ATTACHMENT_WRITE_BIT;
      break;
    case VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL:
      imageMemoryBarrier.dstAccessMask = VK_ACCESS_SHADER_READ_BIT | VK_ACCESS_INPUT_ATTACHMENT_READ_BIT;
      break;
    case VK_IMAGE_LAYOUT_TRANSFER_SRC_OPTIMAL:
      imageMemoryBarrier.dstAccessMask = VK_ACCESS_TRANSFER_READ_BIT;
      break;
    case VK_IMAGE_LAYOUT_PRESENT_SRC_KHR:
      imageMemoryBarrier.dstAccessMask = VK_ACCESS_MEMORY_READ_BIT;
      break;
    default:
      imageMemoryBarrier.dstAccessMask = 0;
      break;
  }
  mVkHelper.vkCmdPipelineBarrier(mCommandBuffer, sourceStages, destinationStages, 0, 0, nullptr, 0, nullptr, 1, &imageMemoryBarrier);
}

void Cube::prepareTextures() {
  const VkFormat textureFormat = VK_FORMAT_R8G8B8A8_UNORM;
  VkFormatProperties formatProperties;
  mVkHelper.vkGetPhysicalDeviceFormatProperties(mGpu, textureFormat, &formatProperties);
  if (formatProperties.optimalTilingFeatures & VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT) {
    // Must use staging buffer to copy linear texture to optimized
    memset(&mStagingTexture, 0, sizeof(mStagingTexture));
    prepareTextureBuffer(texFile, &mStagingTexture);
    prepareTextureImage(texFile, &mTexture, VK_IMAGE_TILING_OPTIMAL,
                        (VK_IMAGE_USAGE_TRANSFER_DST_BIT | VK_IMAGE_USAGE_SAMPLED_BIT),
                        VK_MEMORY_PROPERTY_DEVICE_LOCAL_BIT);
    setImageLayout(mTexture.image, VK_IMAGE_ASPECT_COLOR_BIT,
                   VK_IMAGE_LAYOUT_PREINITIALIZED, VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
                   static_cast<VkAccessFlagBits>(0), VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT,
                   VK_PIPELINE_STAGE_TRANSFER_BIT);
    const VkBufferImageCopy bufferImageCopy = {
        .bufferOffset = 0,
        .bufferRowLength = mStagingTexture.width,
        .bufferImageHeight = mStagingTexture.height,
        .imageSubresource = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 0, 1},
        .imageOffset = {0, 0, 0},
        .imageExtent = {mStagingTexture.width, mStagingTexture.height, 1},
    };
    mVkHelper.vkCmdCopyBufferToImage(mCommandBuffer, mStagingTexture.buffer, mTexture.image,
                                     VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL, 1, &bufferImageCopy);
    setImageLayout(mTexture.image,
                   VK_IMAGE_ASPECT_COLOR_BIT,
                   VK_IMAGE_LAYOUT_TRANSFER_DST_OPTIMAL,
                   mTexture.imageLayout,
                   VK_ACCESS_TRANSFER_WRITE_BIT,
                   VK_PIPELINE_STAGE_TRANSFER_BIT,
                   VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT);
  } else if (formatProperties.linearTilingFeatures & VK_FORMAT_FEATURE_SAMPLED_IMAGE_BIT) {
    // Device can texture using linear textures
    prepareTextureImage(texFile, &mTexture, VK_IMAGE_TILING_LINEAR, VK_IMAGE_USAGE_SAMPLED_BIT,
                        VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT);
    // Nothing in the pipeline needs to be complete to start, and don't allow fragment
    // shader to run until layout transition completes
    setImageLayout(mTexture.image, VK_IMAGE_ASPECT_COLOR_BIT, VK_IMAGE_LAYOUT_PREINITIALIZED,
                   mTexture.imageLayout, static_cast<VkAccessFlagBits>(0),
                   VK_PIPELINE_STAGE_TOP_OF_PIPE_BIT, VK_PIPELINE_STAGE_FRAGMENT_SHADER_BIT);
    mStagingTexture.image = VK_NULL_HANDLE;
  } else {
    // This should never happen.
    ASSERT(!"No support for R8G8B8A8_UNORM as texture image format");
  }
  const VkSamplerCreateInfo samplerCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_SAMPLER_CREATE_INFO,
      .pNext = nullptr,
      .magFilter = VK_FILTER_NEAREST,
      .minFilter = VK_FILTER_NEAREST,
      .mipmapMode = VK_SAMPLER_MIPMAP_MODE_NEAREST,
      .addressModeU = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
      .addressModeV = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
      .addressModeW = VK_SAMPLER_ADDRESS_MODE_CLAMP_TO_EDGE,
      .mipLodBias = 0.0f,
      .anisotropyEnable = VK_FALSE,
      .maxAnisotropy = 1,
      .compareOp = VK_COMPARE_OP_NEVER,
      .minLod = 0.0f,
      .maxLod = 0.0f,
      .borderColor = VK_BORDER_COLOR_FLOAT_OPAQUE_WHITE,
      .unnormalizedCoordinates = VK_FALSE,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateSampler(mDevice, &samplerCreateInfo, nullptr, &mTexture.sampler));

  VkImageViewCreateInfo imageViewCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
      .pNext = nullptr,
      .image = mTexture.image,
      .viewType = VK_IMAGE_VIEW_TYPE_2D,
      .format = textureFormat,
      .components = {
          VK_COMPONENT_SWIZZLE_R,
          VK_COMPONENT_SWIZZLE_G,
          VK_COMPONENT_SWIZZLE_B,
          VK_COMPONENT_SWIZZLE_A,
      },
      .subresourceRange = {VK_IMAGE_ASPECT_COLOR_BIT, 0, 1, 0, 1},
      .flags = 0,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateImageView(mDevice, &imageViewCreateInfo, nullptr, &mTexture.view));
}

void Cube::prepareDataBuffers() {
  uint8_t *pData;
  mat4x4 viewportMatrix;

  mat4x4_mul(viewportMatrix, mProjectMatrix, mViewMatrix);
  mat4x4_mul(mUniform.mvp, viewportMatrix, mModelMatrix);
  for (uint32_t i = 0; i < 12 * 3; i++) {
    mUniform.position[i][0] = gVertexBufferData[i * 3];
    mUniform.position[i][1] = gVertexBufferData[i * 3 + 1];
    mUniform.position[i][2] = gVertexBufferData[i * 3 + 2];
    mUniform.position[i][3] = 1.0f;
    mUniform.attr[i][0] = gUvBufferData[2 * i];
    mUniform.attr[i][1] = gUvBufferData[2 * i + 1];
    mUniform.attr[i][2] = 0;
    mUniform.attr[i][3] = 0;
  }
  const VkBufferCreateInfo bufferCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
      .usage = VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT,
      .size = sizeof(mUniform),
  };
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateBuffer(mDevice, &bufferCreateInfo, nullptr, &mSwapchainImageResources[i].uniformBuffer));
    VkMemoryRequirements memoryRequirements;
    mVkHelper.vkGetBufferMemoryRequirements(mDevice, mSwapchainImageResources[i].uniformBuffer, &memoryRequirements);
    VkMemoryAllocateInfo memoryAllocationInfo = {
        .sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
        .pNext = nullptr,
        .allocationSize = memoryRequirements.size,
        .memoryTypeIndex = 0,
    };
    ASSERT(memoryTypeFromProperties(memoryRequirements.memoryTypeBits,
                                    VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT | VK_MEMORY_PROPERTY_HOST_COHERENT_BIT,
                                    &memoryAllocationInfo.memoryTypeIndex));
    ASSERT(VK_SUCCESS == mVkHelper.vkAllocateMemory(mDevice, &memoryAllocationInfo, nullptr, &mSwapchainImageResources[i].uniformDeviceMemory));
    ASSERT(VK_SUCCESS == mVkHelper.vkMapMemory(mDevice, mSwapchainImageResources[i].uniformDeviceMemory, 0, VK_WHOLE_SIZE, 0, (void **)&pData));
    memcpy(pData, &mUniform, sizeof(mUniform));
    mVkHelper.vkUnmapMemory(mDevice, mSwapchainImageResources[i].uniformDeviceMemory);
    ASSERT(VK_SUCCESS == mVkHelper.vkBindBufferMemory(mDevice, mSwapchainImageResources[i].uniformBuffer,
                             mSwapchainImageResources[i].uniformDeviceMemory, 0));
  }
}

void Cube::prepareDescriptorLayout() {
  const VkDescriptorSetLayoutBinding descriptorSetLayoutBindings[2] = {
      [0] =
          {
              .binding = 0,
              .descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
              .descriptorCount = 1,
              .stageFlags = VK_SHADER_STAGE_VERTEX_BIT | VK_SHADER_STAGE_FRAGMENT_BIT,
              .pImmutableSamplers = nullptr,
          },
      [1] =
          {
              .binding = 1,
              .descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
              .descriptorCount = 1,
              .stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT,
              .pImmutableSamplers = nullptr,
          },
  };
  const VkDescriptorSetLayoutCreateInfo descriptorSetLayoutCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
      .pNext = nullptr,
      .bindingCount = 2,
      .pBindings = descriptorSetLayoutBindings,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateDescriptorSetLayout(mDevice, &descriptorSetLayoutCreateInfo, nullptr, &mDescriptorSetLayout));

  const VkPipelineLayoutCreateInfo pipelineLayoutCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
      .pNext = nullptr,
      .setLayoutCount = 1,
      .pSetLayouts = &mDescriptorSetLayout,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreatePipelineLayout(mDevice, &pipelineLayoutCreateInfo, nullptr, &mPipelineLayout));
}

void Cube::prepareRenderPass() {
  // The initial layout for the color and depth attachments will be LAYOUT_UNDEFINED
  // because at the start of the render pass, we don't care about their contents.
  // At the start of the subpass, the color attachment's layout will be transitioned
  // to LAYOUT_COLOR_ATTACHMENT_OPTIMAL and the depth stencil attachment's layout
  // will be transitioned to LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL.  At the end of
  // the render pass, the color attachment's layout will be transitioned to
  // LAYOUT_PRESENT_SRC_KHR to be ready to present. This is all done as part of
  // the render pass, no barriers are necessary.
  const VkAttachmentDescription attachmentDescriptions[2] = {
      [0] = {
          .format = mFormat,
          .flags = 0,
          .samples = VK_SAMPLE_COUNT_1_BIT,
          .loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR,
          .storeOp = VK_ATTACHMENT_STORE_OP_STORE,
          .stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE,
          .stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE,
          .initialLayout = VK_IMAGE_LAYOUT_UNDEFINED,
          .finalLayout = VK_IMAGE_LAYOUT_PRESENT_SRC_KHR,
      },
      [1] = {
          .format = mDepth.format,
          .flags = 0,
          .samples = VK_SAMPLE_COUNT_1_BIT,
          .loadOp = VK_ATTACHMENT_LOAD_OP_CLEAR,
          .storeOp = VK_ATTACHMENT_STORE_OP_DONT_CARE,
          .stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE,
          .stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE,
          .initialLayout = VK_IMAGE_LAYOUT_UNDEFINED,
          .finalLayout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
      },
  };
  const VkAttachmentReference colorAttachmentReference = {
      .attachment = 0,
      .layout = VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
  };
  const VkAttachmentReference depthAttachmentReference = {
      .attachment = 1,
      .layout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL,
  };
  const VkSubpassDescription subpassDescription = {
      .pipelineBindPoint = VK_PIPELINE_BIND_POINT_GRAPHICS,
      .flags = 0,
      .inputAttachmentCount = 0,
      .pInputAttachments = nullptr,
      .colorAttachmentCount = 1,
      .pColorAttachments = &colorAttachmentReference,
      .pResolveAttachments = nullptr,
      .pDepthStencilAttachment = &depthAttachmentReference,
      .preserveAttachmentCount = 0,
      .pPreserveAttachments = nullptr,
  };
  const VkRenderPassCreateInfo renderPassCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .attachmentCount = 2,
      .pAttachments = attachmentDescriptions,
      .subpassCount = 1,
      .pSubpasses = &subpassDescription,
      .dependencyCount = 0,
      .pDependencies = nullptr,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateRenderPass(mDevice, &renderPassCreateInfo, nullptr, &mRenderPass));
}

void Cube::preparePipeline() {
  const VkPipelineVertexInputStateCreateInfo pipelineVertexInputStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
  };

  const VkPipelineInputAssemblyStateCreateInfo pipelineInputAssemblyStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
      .topology = VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
  };

  const VkPipelineRasterizationStateCreateInfo pipelineRasterizationStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
      .polygonMode = VK_POLYGON_MODE_FILL,
      .cullMode = VK_CULL_MODE_BACK_BIT,
      .frontFace = VK_FRONT_FACE_COUNTER_CLOCKWISE,
      .depthClampEnable = VK_FALSE,
      .rasterizerDiscardEnable = VK_FALSE,
      .depthBiasEnable = VK_FALSE,
      .lineWidth = 1.0f,
  };

  const VkPipelineColorBlendAttachmentState pipelineColorBlendAttachmentState = {
      .colorWriteMask = 0xf,
      .blendEnable = VK_FALSE,
  };
  const VkPipelineColorBlendStateCreateInfo pipelineColorBlendStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
      .attachmentCount = 1,
      .pAttachments = &pipelineColorBlendAttachmentState,
  };

  const VkPipelineViewportStateCreateInfo pipelineViewportStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
      .viewportCount = 1,
      .scissorCount = 1
  };

  const VkDynamicState dynamicStateEnables[2] = { VK_DYNAMIC_STATE_VIEWPORT, VK_DYNAMIC_STATE_SCISSOR };
  const VkPipelineDynamicStateCreateInfo pipelineDynamicStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
      .dynamicStateCount = 2,
      .pDynamicStates = dynamicStateEnables,
  };

  VkPipelineDepthStencilStateCreateInfo pipelineDepthStencilStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
      .depthTestEnable = VK_TRUE,
      .depthWriteEnable = VK_TRUE,
      .depthCompareOp = VK_COMPARE_OP_LESS_OR_EQUAL,
      .depthBoundsTestEnable = VK_FALSE,
      .back.failOp = VK_STENCIL_OP_KEEP,
      .back.passOp = VK_STENCIL_OP_KEEP,
      .back.compareOp = VK_COMPARE_OP_ALWAYS,
      .stencilTestEnable = VK_FALSE,
  };
  pipelineDepthStencilStateCreateInfo.front = pipelineDepthStencilStateCreateInfo.back;

  const VkPipelineMultisampleStateCreateInfo pipelineMultisampleStateCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
      .pSampleMask = nullptr,
      .rasterizationSamples = VK_SAMPLE_COUNT_1_BIT,
  };
  loadShaderFromFile("cube.vert.spv", &mVertexShaderModule);
  loadShaderFromFile("cube.frag.spv", &mFragmentShaderModule);

  // Two stages: vs and fs
  const VkPipelineShaderStageCreateInfo shaderStages[2] = {
      [0] = {
          .sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
          .stage = VK_SHADER_STAGE_VERTEX_BIT,
          .module = mVertexShaderModule,
          .pName = "main",
      },
      [1] = {
          .sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
          .stage = VK_SHADER_STAGE_FRAGMENT_BIT,
          .module = mFragmentShaderModule,
          .pName = "main",
      },
  };

  const VkPipelineCacheCreateInfo pipelineCacheCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_PIPELINE_CACHE_CREATE_INFO,
  };

  ASSERT(VK_SUCCESS == mVkHelper.vkCreatePipelineCache(mDevice, &pipelineCacheCreateInfo, nullptr, &mPipelineCache));

  const VkGraphicsPipelineCreateInfo pipelineCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
      .layout = mPipelineLayout,
      .pVertexInputState = &pipelineVertexInputStateCreateInfo,
      .pInputAssemblyState = &pipelineInputAssemblyStateCreateInfo,
      .pRasterizationState = &pipelineRasterizationStateCreateInfo,
      .pColorBlendState = &pipelineColorBlendStateCreateInfo,
      .pMultisampleState = &pipelineMultisampleStateCreateInfo,
      .pViewportState = &pipelineViewportStateCreateInfo,
      .pDepthStencilState = &pipelineDepthStencilStateCreateInfo,
      .stageCount = 2,
      .pStages = shaderStages,
      .renderPass = mRenderPass,
      .pDynamicState = &pipelineDynamicStateCreateInfo,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateGraphicsPipelines(mDevice, mPipelineCache, 1, &pipelineCreateInfo, nullptr, &mPipeline));
  mVkHelper.vkDestroyShaderModule(mDevice, mFragmentShaderModule, nullptr);
  mVkHelper.vkDestroyShaderModule(mDevice, mVertexShaderModule, nullptr);
}

void Cube::prepareDescriptorPool() {
  const VkDescriptorPoolSize descriptorPoolSize[2] = {
      [0] = {
          .type = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
          .descriptorCount = mSwapchainImageCount,
      },
      [1] = {
          .type = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
          .descriptorCount = mSwapchainImageCount,
      },
  };
  const VkDescriptorPoolCreateInfo descriptorPoolCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
      .pNext = nullptr,
      .maxSets = mSwapchainImageCount,
      .poolSizeCount = 2,
      .pPoolSizes = descriptorPoolSize,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateDescriptorPool(mDevice, &descriptorPoolCreateInfo, nullptr, &mDescriptorPool));
}

void Cube::prepareDescriptorSet() {
  const VkDescriptorImageInfo descriptorImageInfo = {
      .sampler = mTexture.sampler,
      .imageView = mTexture.view,
      .imageLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
  };

  const VkDescriptorSetAllocateInfo descriptorSetAllocateInfo = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
      .pNext = nullptr,
      .descriptorPool = mDescriptorPool,
      .descriptorSetCount = 1,
      .pSetLayouts = &mDescriptorSetLayout
  };

  VkDescriptorBufferInfo descriptorBufferInfo = {
      .offset = 0,
      .range = sizeof(VkTexCubeVsUniform),
  };

  VkWriteDescriptorSet writeDescriptorSet[2];
  memset(&writeDescriptorSet, 0, sizeof(writeDescriptorSet));
  writeDescriptorSet[0].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
  writeDescriptorSet[0].descriptorCount = 1;
  writeDescriptorSet[0].descriptorType = VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER;
  writeDescriptorSet[0].pBufferInfo = &descriptorBufferInfo;

  writeDescriptorSet[1].sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
  writeDescriptorSet[1].dstBinding = 1;
  writeDescriptorSet[1].descriptorCount = 1;
  writeDescriptorSet[1].descriptorType = VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
  writeDescriptorSet[1].pImageInfo = &descriptorImageInfo;

  for (unsigned int i = 0; i < mSwapchainImageCount; i++) {
    ASSERT(VK_SUCCESS == mVkHelper.vkAllocateDescriptorSets(mDevice, &descriptorSetAllocateInfo, &mSwapchainImageResources[i].descriptorSet));
    descriptorBufferInfo.buffer = mSwapchainImageResources[i].uniformBuffer;
    writeDescriptorSet[0].dstSet = mSwapchainImageResources[i].descriptorSet;
    writeDescriptorSet[1].dstSet = mSwapchainImageResources[i].descriptorSet;
    mVkHelper.vkUpdateDescriptorSets(mDevice, 2, writeDescriptorSet, 0, nullptr);
  }
}

void Cube::prepareFramebuffers() {
  VkImageView imageViews[2];
  imageViews[1] = mDepth.view;

  const VkFramebufferCreateInfo framebufferCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_FRAMEBUFFER_CREATE_INFO,
      .pNext = nullptr,
      .renderPass = mRenderPass,
      .attachmentCount = 2,
      .pAttachments = imageViews,
      .width = mWidth,
      .height = mHeight,
      .layers = 1,
  };
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    imageViews[0] = mSwapchainImageResources[i].imageView;
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateFramebuffer(mDevice, &framebufferCreateInfo, nullptr, &mSwapchainImageResources[i].framebuffer));
  }
}

void Cube::cleanup() {
  mPrepared = false;
  mVkHelper.vkDeviceWaitIdle(mDevice);

  // Wait for fences from present operations
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    mVkHelper.vkWaitForFences(mDevice, 1, &mFences[i], VK_TRUE, UINT64_MAX);
    mVkHelper.vkDestroyFence(mDevice, mFences[i], nullptr);
    mVkHelper.vkDestroySemaphore(mDevice, mImageAcquiredSemaphores[i], nullptr);
    mVkHelper.vkDestroySemaphore(mDevice, mDrawCompleteSemaphores[i], nullptr);
    if (mSeparatePresentQueue) {
      mVkHelper.vkDestroySemaphore(mDevice, mImageOwnershipSemaphores[i], nullptr);
    }
  }
  mFences.clear();
  mImageAcquiredSemaphores.clear();
  mDrawCompleteSemaphores.clear();
  mImageOwnershipSemaphores.clear();

  // If the window is currently minimized, resize() has already done some cleanup for us.
  if (!mMinimized) {
    for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
      mVkHelper.vkDestroyFramebuffer(mDevice, mSwapchainImageResources[i].framebuffer, nullptr);
    }
    mVkHelper.vkDestroyDescriptorPool(mDevice, mDescriptorPool, nullptr);
    mVkHelper.vkDestroyPipeline(mDevice, mPipeline, nullptr);
    mVkHelper.vkDestroyPipelineCache(mDevice, mPipelineCache, nullptr);
    mVkHelper.vkDestroyRenderPass(mDevice, mRenderPass, nullptr);
    mVkHelper.vkDestroyPipelineLayout(mDevice, mPipelineLayout, nullptr);
    mVkHelper.vkDestroyDescriptorSetLayout(mDevice, mDescriptorSetLayout, nullptr);
    mVkHelper.vkDestroyImageView(mDevice, mTexture.view, nullptr);
    mVkHelper.vkDestroyImage(mDevice, mTexture.image, nullptr);
    mVkHelper.vkFreeMemory(mDevice, mTexture.deviceMemory, nullptr);
    mVkHelper.vkDestroySampler(mDevice, mTexture.sampler, nullptr);
    mVkHelper.vkDestroySwapchainKHR(mDevice, mSwapchain, nullptr);
    mSwapchain = VK_NULL_HANDLE;
    mVkHelper.vkDestroyImageView(mDevice, mDepth.view, nullptr);
    mVkHelper.vkDestroyImage(mDevice, mDepth.image, nullptr);
    mVkHelper.vkFreeMemory(mDevice, mDepth.deviceMemory, nullptr);
    for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
      mVkHelper.vkDestroyImageView(mDevice, mSwapchainImageResources[i].imageView, nullptr);
      mVkHelper.vkFreeCommandBuffers(mDevice, mCommandPool, 1, &mSwapchainImageResources[i].commandBuffer);
      mVkHelper.vkDestroyBuffer(mDevice, mSwapchainImageResources[i].uniformBuffer, nullptr);
      mVkHelper.vkFreeMemory(mDevice, mSwapchainImageResources[i].uniformDeviceMemory, nullptr);
    }
    mSwapchainImageResources.clear();
    mQueueFamilyProperties.clear();
    mVkHelper.vkDestroyCommandPool(mDevice, mCommandPool, nullptr);
    if (mSeparatePresentQueue) {
      mVkHelper.vkDestroyCommandPool(mDevice, mPresentCommandPool, nullptr);
    }
  }
  mVkHelper.vkDeviceWaitIdle(mDevice);
  mVkHelper.vkDestroyDevice(mDevice, nullptr);
  mVkHelper.vkDestroySurfaceKHR(mInstance, mSurface, nullptr);
  mVkHelper.vkDestroyInstance(mInstance, nullptr);
}

void Cube::resize() {
  // Don't react to resize until after first initialization.
  if (!mPrepared) {
    if (mMinimized) {
      prepare();
    }
    return;
  }

  // In order to properly resize the window, we must re-create the swapchain
  // AND redo the command buffers, etc.
  //
  // First, perform part of the cleanup() function.
  mPrepared = false;
  mVkHelper.vkDeviceWaitIdle(mDevice);
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    mVkHelper.vkDestroyFramebuffer(mDevice, mSwapchainImageResources[i].framebuffer, nullptr);
  }
  mVkHelper.vkDestroyDescriptorPool(mDevice, mDescriptorPool, nullptr);
  mVkHelper.vkDestroyPipeline(mDevice, mPipeline, nullptr);
  mVkHelper.vkDestroyPipelineCache(mDevice, mPipelineCache, nullptr);
  mVkHelper.vkDestroyRenderPass(mDevice, mRenderPass, nullptr);
  mVkHelper.vkDestroyPipelineLayout(mDevice, mPipelineLayout, nullptr);
  mVkHelper.vkDestroyDescriptorSetLayout(mDevice, mDescriptorSetLayout, nullptr);
  mVkHelper.vkDestroyImageView(mDevice, mTexture.view, nullptr);
  mVkHelper.vkDestroyImage(mDevice, mTexture.image, nullptr);
  mVkHelper.vkFreeMemory(mDevice, mTexture.deviceMemory, nullptr);
  mVkHelper.vkDestroySampler(mDevice, mTexture.sampler, nullptr);
  mVkHelper.vkDestroyImageView(mDevice, mDepth.view, nullptr);
  mVkHelper.vkDestroyImage(mDevice, mDepth.image, nullptr);
  mVkHelper.vkFreeMemory(mDevice, mDepth.deviceMemory, nullptr);
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    mVkHelper.vkDestroyImageView(mDevice, mSwapchainImageResources[i].imageView, nullptr);
    mVkHelper.vkFreeCommandBuffers(mDevice, mCommandPool, 1, &mSwapchainImageResources[i].commandBuffer);
    mVkHelper.vkDestroyBuffer(mDevice, mSwapchainImageResources[i].uniformBuffer, nullptr);
    mVkHelper.vkFreeMemory(mDevice, mSwapchainImageResources[i].uniformDeviceMemory, nullptr);
  }
  mVkHelper.vkDestroyCommandPool(mDevice, mCommandPool, nullptr);
  mCommandPool = VK_NULL_HANDLE;
  if (mSeparatePresentQueue) {
    mVkHelper.vkDestroyCommandPool(mDevice, mPresentCommandPool, nullptr);
  }
  mSwapchainImageResources.clear();

  // Perform the prepare() again, which will re-create the swapchain.
  prepare();
}

void Cube::createInstance() {
  mMinimized = false;
  mCommandPool = VK_NULL_HANDLE;

  uint32_t extensionCount = 0;
  ASSERT(VK_SUCCESS == mVkHelper.vkEnumerateInstanceExtensionProperties(nullptr, &extensionCount, nullptr));
  std::vector<const char*> enabledExtensions;
  if (extensionCount > 0) {
    std::vector<VkExtensionProperties> supportedExtensions(extensionCount);
    ASSERT(VK_SUCCESS == mVkHelper.vkEnumerateInstanceExtensionProperties(nullptr, &extensionCount, supportedExtensions.data()));
    for (auto extension : kRequiredInstanceExtensions) {
      ASSERT(hasExtension(extension, supportedExtensions));
      enabledExtensions.push_back(extension);
    }
  }
  const VkApplicationInfo applicationInfo = {
      .sType = VK_STRUCTURE_TYPE_APPLICATION_INFO,
      .pNext = nullptr,
      .pApplicationName = APP_NAME,
      .applicationVersion = 0,
      .pEngineName = APP_NAME,
      .engineVersion = 0,
      .apiVersion = VK_MAKE_VERSION(1, 0, 0),
  };
  const VkInstanceCreateInfo instanceCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
      .pNext = nullptr,
      .pApplicationInfo = &applicationInfo,
      .enabledLayerCount = 0,
      .ppEnabledLayerNames = nullptr,
      .enabledExtensionCount = static_cast<uint32_t>(enabledExtensions.size()),
      .ppEnabledExtensionNames = enabledExtensions.data(),
  };

  VkResult err = mVkHelper.vkCreateInstance(&instanceCreateInfo, nullptr, &mInstance);
  if (err == VK_ERROR_INCOMPATIBLE_DRIVER) {
    ERR_EXIT(
        "Cannot find a compatible Vulkan installable client driver (ICD).\n\n"
        "Please look at the Getting Started guide for additional information.\n",
        "vkCreateInstance Failure");
  } else if (err == VK_ERROR_EXTENSION_NOT_PRESENT) {
    ERR_EXIT(
        "Cannot find a specified extension library.\n"
        "Make sure your layers path is set appropriately.\n",
        "vkCreateInstance Failure");
  } else if (err) {
    ERR_EXIT(
        "vkCreateInstance failed.\n\n"
        "Do you have a compatible Vulkan installable client driver (ICD) installed?\n"
        "Please look at the Getting Started guide for additional information.\n",
        "vkCreateInstance Failure");
  }

  GET_INSTANCE_PROC_ADDR(CreateAndroidSurfaceKHR);
  GET_INSTANCE_PROC_ADDR(CreateDevice);
  GET_INSTANCE_PROC_ADDR(DestroyInstance);
  GET_INSTANCE_PROC_ADDR(DestroySurfaceKHR);
  GET_INSTANCE_PROC_ADDR(EnumerateDeviceExtensionProperties);
  GET_INSTANCE_PROC_ADDR(EnumeratePhysicalDevices);
  GET_INSTANCE_PROC_ADDR(GetDeviceProcAddr);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceMemoryProperties);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceQueueFamilyProperties);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceSurfaceFormatsKHR);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceSurfaceSupportKHR);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceSurfaceCapabilitiesKHR);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceSurfacePresentModesKHR);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceFormatProperties);
  GET_INSTANCE_PROC_ADDR(GetPhysicalDeviceProperties);
}

void Cube::createDevice() {
  uint32_t gpuCount;
  ASSERT(VK_SUCCESS == mVkHelper.vkEnumeratePhysicalDevices(mInstance, &gpuCount, nullptr));
  ASSERT(gpuCount > 0);
  std::vector<VkPhysicalDevice> physicalDevices(gpuCount);
  ASSERT(VK_SUCCESS == mVkHelper.vkEnumeratePhysicalDevices(mInstance, &gpuCount, physicalDevices.data()));
  mGpu = physicalDevices[0];

  uint32_t extensionCount = 0;
  ASSERT(VK_SUCCESS == mVkHelper.vkEnumerateDeviceExtensionProperties(mGpu, nullptr, &extensionCount, nullptr));
  std::vector<const char*> enabledExtensions;
  if (extensionCount > 0) {
    std::vector<VkExtensionProperties> supportedExtensions(extensionCount);
    ASSERT(VK_SUCCESS == mVkHelper.vkEnumerateDeviceExtensionProperties(mGpu, nullptr, &extensionCount, supportedExtensions.data()));
    for (auto extension : kRequiredDeviceExtensions) {
      ASSERT(hasExtension(extension, supportedExtensions));
      enabledExtensions.push_back(extension);
    }
  }

  mVkHelper.vkGetPhysicalDeviceProperties(mGpu, &mGpuProperties);
  mVkHelper.vkGetPhysicalDeviceQueueFamilyProperties(mGpu, &mQueueFamilyCount, nullptr);
  ASSERT(mQueueFamilyCount >= 1);
  mQueueFamilyProperties.resize(mQueueFamilyCount);
  mVkHelper.vkGetPhysicalDeviceQueueFamilyProperties(mGpu, &mQueueFamilyCount, mQueueFamilyProperties.data());

  // Create surface first in order to determine present queue.
  const VkAndroidSurfaceCreateInfoKHR androidSurfaceCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_ANDROID_SURFACE_CREATE_INFO_KHR,
      .pNext = nullptr,
      .flags = 0,
      .window = mNativeWindow,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateAndroidSurfaceKHR(mInstance, &androidSurfaceCreateInfo, nullptr, &mSurface));

  // Iterate over each queue to learn whether it supports presenting.
  std::vector<VkBool32> supportsPresent(mQueueFamilyCount);
  for (uint32_t i = 0; i < mQueueFamilyCount; i++) {
    mVkHelper.vkGetPhysicalDeviceSurfaceSupportKHR(mGpu, i, mSurface, &supportsPresent[i]);
  }

  // Search for a graphics and a present queue in the array of queue
  // families, try to find one that supports both.
  uint32_t graphicsQueueFamilyIndex = UINT32_MAX;
  uint32_t presentQueueFamilyIndex = UINT32_MAX;
  for (uint32_t i = 0; i < mQueueFamilyCount; i++) {
    if ((mQueueFamilyProperties[i].queueFlags & VK_QUEUE_GRAPHICS_BIT) != 0) {
      if (graphicsQueueFamilyIndex == UINT32_MAX) {
        graphicsQueueFamilyIndex = i;
      }
      if (supportsPresent[i] == VK_TRUE) {
        graphicsQueueFamilyIndex = i;
        presentQueueFamilyIndex = i;
        break;
      }
    }
  }

  if (presentQueueFamilyIndex == UINT32_MAX) {
    // If didn't find a queue that supports both graphics and present, then
    // find a separate present queue.
    for (uint32_t i = 0; i < mQueueFamilyCount; ++i) {
      if (supportsPresent[i] == VK_TRUE) {
        presentQueueFamilyIndex = i;
        break;
      }
    }
  }

  // Generate error if could not find both a graphics and a present queue
  if (graphicsQueueFamilyIndex == UINT32_MAX || presentQueueFamilyIndex == UINT32_MAX) {
    ERR_EXIT("Could not find both graphics and present queues\n", "Swapchain Initialization Failure");
  }

  mGraphicsQueueFamilyIndex = graphicsQueueFamilyIndex;
  mPresentQueueFamilyIndex = presentQueueFamilyIndex;
  mSeparatePresentQueue = (mGraphicsQueueFamilyIndex != mPresentQueueFamilyIndex);

  //
  const float queuePriority = 1.0f;
  VkDeviceQueueCreateInfo deviceQueueCreateInfos[2];
  deviceQueueCreateInfos[0].sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO;
  deviceQueueCreateInfos[0].pNext = nullptr;
  deviceQueueCreateInfos[0].queueFamilyIndex = mGraphicsQueueFamilyIndex;
  deviceQueueCreateInfos[0].queueCount = 1;
  deviceQueueCreateInfos[0].pQueuePriorities = &queuePriority;
  deviceQueueCreateInfos[0].flags = 0;

  VkDeviceCreateInfo deviceCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_DEVICE_CREATE_INFO,
      .pNext = nullptr,
      .queueCreateInfoCount = 1,
      .pQueueCreateInfos = deviceQueueCreateInfos,
      .enabledLayerCount = 0,
      .ppEnabledLayerNames = nullptr,
      .enabledExtensionCount = static_cast<uint32_t>(enabledExtensions.size()),
      .ppEnabledExtensionNames = enabledExtensions.data(),
      .pEnabledFeatures = nullptr,  // If specific features are required, pass them in here
  };
  if (mSeparatePresentQueue) {
    deviceQueueCreateInfos[1].sType = VK_STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO;
    deviceQueueCreateInfos[1].pNext = nullptr;
    deviceQueueCreateInfos[1].queueFamilyIndex = mPresentQueueFamilyIndex;
    deviceQueueCreateInfos[1].queueCount = 1;
    deviceQueueCreateInfos[1].pQueuePriorities = &queuePriority;
    deviceQueueCreateInfos[1].flags = 0;
    deviceCreateInfo.queueCreateInfoCount = 2;
  }
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateDevice(mGpu, &deviceCreateInfo, nullptr, &mDevice));

  GET_DEVICE_PROC_ADDR(AcquireNextImageKHR);
  GET_DEVICE_PROC_ADDR(AllocateCommandBuffers);
  GET_DEVICE_PROC_ADDR(FreeCommandBuffers);
  GET_DEVICE_PROC_ADDR(AllocateMemory);
  GET_DEVICE_PROC_ADDR(FreeMemory);
  GET_DEVICE_PROC_ADDR(BeginCommandBuffer);
  GET_DEVICE_PROC_ADDR(EndCommandBuffer);
  GET_DEVICE_PROC_ADDR(CmdBeginRenderPass);
  GET_DEVICE_PROC_ADDR(CmdBindPipeline);
  GET_DEVICE_PROC_ADDR(CmdBindVertexBuffers);
  GET_DEVICE_PROC_ADDR(CmdDraw);
  GET_DEVICE_PROC_ADDR(CmdEndRenderPass);
  GET_DEVICE_PROC_ADDR(CmdPushConstants);
  GET_DEVICE_PROC_ADDR(CreateBuffer);
  GET_DEVICE_PROC_ADDR(DestroyBuffer);
  GET_DEVICE_PROC_ADDR(CreateCommandPool);
  GET_DEVICE_PROC_ADDR(DestroyCommandPool);
  GET_DEVICE_PROC_ADDR(CreateFramebuffer);
  GET_DEVICE_PROC_ADDR(DestroyFramebuffer);
  GET_DEVICE_PROC_ADDR(CreateGraphicsPipelines);
  GET_DEVICE_PROC_ADDR(DestroyPipeline);
  GET_DEVICE_PROC_ADDR(CreateImageView);
  GET_DEVICE_PROC_ADDR(DestroyImageView);
  GET_DEVICE_PROC_ADDR(CreateImage);
  GET_DEVICE_PROC_ADDR(DestroyImage);
  GET_DEVICE_PROC_ADDR(CreatePipelineLayout);
  GET_DEVICE_PROC_ADDR(DestroyPipelineLayout);
  GET_DEVICE_PROC_ADDR(CreateRenderPass);
  GET_DEVICE_PROC_ADDR(DestroyRenderPass);
  GET_DEVICE_PROC_ADDR(CreateSampler);
  GET_DEVICE_PROC_ADDR(DestroySampler);
  GET_DEVICE_PROC_ADDR(CreateSemaphore);
  GET_DEVICE_PROC_ADDR(DestroySemaphore);
  GET_DEVICE_PROC_ADDR(CreateShaderModule);
  GET_DEVICE_PROC_ADDR(DestroyShaderModule);
  GET_DEVICE_PROC_ADDR(CreateSwapchainKHR);
  GET_DEVICE_PROC_ADDR(DestroySwapchainKHR);
  GET_DEVICE_PROC_ADDR(DestroyDevice);
  GET_DEVICE_PROC_ADDR(DeviceWaitIdle);
  GET_DEVICE_PROC_ADDR(GetBufferMemoryRequirements);
  GET_DEVICE_PROC_ADDR(GetDeviceQueue);
  GET_DEVICE_PROC_ADDR(GetSwapchainImagesKHR);
  GET_DEVICE_PROC_ADDR(QueuePresentKHR);
  GET_DEVICE_PROC_ADDR(QueueSubmit);
  GET_DEVICE_PROC_ADDR(DestroyDescriptorSetLayout);
  GET_DEVICE_PROC_ADDR(CreatePipelineCache);
  GET_DEVICE_PROC_ADDR(DestroyPipelineCache);
  GET_DEVICE_PROC_ADDR(DestroyDescriptorPool);
  GET_DEVICE_PROC_ADDR(ResetFences);
  GET_DEVICE_PROC_ADDR(WaitForFences);
  GET_DEVICE_PROC_ADDR(CreateFence);
  GET_DEVICE_PROC_ADDR(DestroyFence);
  GET_DEVICE_PROC_ADDR(BindBufferMemory);
  GET_DEVICE_PROC_ADDR(CmdPipelineBarrier);
  GET_DEVICE_PROC_ADDR(MapMemory);
  GET_DEVICE_PROC_ADDR(UnmapMemory);
  GET_DEVICE_PROC_ADDR(GetImageSubresourceLayout);
  GET_DEVICE_PROC_ADDR(GetImageMemoryRequirements);
  GET_DEVICE_PROC_ADDR(BindImageMemory);
  GET_DEVICE_PROC_ADDR(CmdBindDescriptorSets);
  GET_DEVICE_PROC_ADDR(CmdSetViewport);
  GET_DEVICE_PROC_ADDR(CmdSetScissor);
  GET_DEVICE_PROC_ADDR(AllocateDescriptorSets);
  GET_DEVICE_PROC_ADDR(UpdateDescriptorSets);
  GET_DEVICE_PROC_ADDR(CreateDescriptorPool);
  GET_DEVICE_PROC_ADDR(CreateDescriptorSetLayout);
  GET_DEVICE_PROC_ADDR(CmdCopyBufferToImage);

  mVkHelper.vkGetDeviceQueue(mDevice, mGraphicsQueueFamilyIndex, 0, &mGraphicsQueue);
  if (!mSeparatePresentQueue) {
    mPresentQueue = mGraphicsQueue;
  } else {
    mVkHelper.vkGetDeviceQueue(mDevice, mPresentQueueFamilyIndex, 0, &mPresentQueue);
  }

  // Get the list of VkFormat's that are supported.
  uint32_t formatCount;
  ASSERT(VK_SUCCESS == mVkHelper.vkGetPhysicalDeviceSurfaceFormatsKHR(mGpu, mSurface, &formatCount, nullptr));
  // VkSurfaceFormatKHR *surfFormats = (VkSurfaceFormatKHR *)malloc(formatCount * sizeof(VkSurfaceFormatKHR));
  std::vector<VkSurfaceFormatKHR> surfaceFormats(formatCount);
  ASSERT(VK_SUCCESS == mVkHelper.vkGetPhysicalDeviceSurfaceFormatsKHR(mGpu, mSurface, &formatCount, surfaceFormats.data()));
  // If the format list includes just one entry of VK_FORMAT_UNDEFINED,
  // the surface has no preferred format.  Otherwise, at least one
  // supported format will be returned.
  uint32_t formatIndex;
  for (formatIndex = 0; formatIndex < formatCount; ++formatIndex) {
    if (surfaceFormats[formatIndex].format == VK_FORMAT_R8G8B8A8_UNORM) {
      break;
    }
  }
  ASSERT(formatIndex < formatCount);
  mFormat = surfaceFormats[formatIndex].format;
  mColorSpace = surfaceFormats[formatIndex].colorSpace;

  // Get Memory information and properties
  mVkHelper.vkGetPhysicalDeviceMemoryProperties(mGpu, &mPhysicalDevicememoryProperties);
}

void Cube::createSwapchain() {
  VkSwapchainKHR oldSwapchain = mSwapchain;

  // Check the surface capabilities and formats
  VkSurfaceCapabilitiesKHR surfaceCapabilities;
  ASSERT(VK_SUCCESS == mVkHelper.vkGetPhysicalDeviceSurfaceCapabilitiesKHR(mGpu, mSurface, &surfaceCapabilities));

  VkExtent2D swapchainExtent;
  // width and height are either both 0xFFFFFFFF, or both not 0xFFFFFFFF.
  if (surfaceCapabilities.currentExtent.width == 0xFFFFFFFF) {
    // If the surface size is undefined, the size is set to the size
    // of the images requested, which must fit within the minimum and
    // maximum values.
    swapchainExtent.width = mWidth;
    swapchainExtent.height = mHeight;
    if (swapchainExtent.width < surfaceCapabilities.minImageExtent.width) {
      swapchainExtent.width = surfaceCapabilities.minImageExtent.width;
    } else if (swapchainExtent.width > surfaceCapabilities.maxImageExtent.width) {
      swapchainExtent.width = surfaceCapabilities.maxImageExtent.width;
    }

    if (swapchainExtent.height < surfaceCapabilities.minImageExtent.height) {
      swapchainExtent.height = surfaceCapabilities.minImageExtent.height;
    } else if (swapchainExtent.height > surfaceCapabilities.maxImageExtent.height) {
      swapchainExtent.height = surfaceCapabilities.maxImageExtent.height;
    }
  } else {
    // If the surface size is defined, the swap chain size must match
    swapchainExtent = surfaceCapabilities.currentExtent;
    mWidth = surfaceCapabilities.currentExtent.width;
    mHeight = surfaceCapabilities.currentExtent.height;
  }

  if (mWidth == 0 || mHeight == 0) {
    mMinimized = true;
    return;
  } else {
    mMinimized = false;
  }

  // Determine the number of VkImages to use in the swap chain.
  // Application desires to acquire 3 images at a time for triple
  // buffering
  uint32_t desiredNumOfSwapchainImages = 3;
  if (desiredNumOfSwapchainImages < surfaceCapabilities.minImageCount) {
    desiredNumOfSwapchainImages = surfaceCapabilities.minImageCount;
  }
  // If maxImageCount is 0, we can ask for as many images as we want;
  // otherwise we're limited to maxImageCount
  if ((surfaceCapabilities.maxImageCount > 0) && (desiredNumOfSwapchainImages > surfaceCapabilities.maxImageCount)) {
    // Application must settle for fewer images than desired.
    desiredNumOfSwapchainImages = surfaceCapabilities.maxImageCount;
  }

  VkSurfaceTransformFlagBitsKHR preTransform;
  if (surfaceCapabilities.supportedTransforms & VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR) {
    preTransform = VK_SURFACE_TRANSFORM_IDENTITY_BIT_KHR;
  } else {
    preTransform = surfaceCapabilities.currentTransform;
  }

  // Find a supported composite alpha mode - one of these is guaranteed to be set
  VkCompositeAlphaFlagBitsKHR compositeAlpha = VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR;
  VkCompositeAlphaFlagBitsKHR compositeAlphaFlags[4] = {
      VK_COMPOSITE_ALPHA_OPAQUE_BIT_KHR,
      VK_COMPOSITE_ALPHA_PRE_MULTIPLIED_BIT_KHR,
      VK_COMPOSITE_ALPHA_POST_MULTIPLIED_BIT_KHR,
      VK_COMPOSITE_ALPHA_INHERIT_BIT_KHR,
  };
  for (uint32_t i = 0; i < 4; i++) {
    if (surfaceCapabilities.supportedCompositeAlpha & compositeAlphaFlags[i]) {
      compositeAlpha = compositeAlphaFlags[i];
      break;
    }
  }

  VkSwapchainCreateInfoKHR swapchainCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_SWAPCHAIN_CREATE_INFO_KHR,
      .pNext = nullptr,
      .surface = mSurface,
      .minImageCount = desiredNumOfSwapchainImages,
      .imageFormat = mFormat,
      .imageColorSpace = mColorSpace,
      .imageExtent = {
          .width = swapchainExtent.width,
          .height = swapchainExtent.height,
      },
      .imageUsage = VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT,
      .preTransform = preTransform,
      .compositeAlpha = compositeAlpha,
      .imageArrayLayers = 1,
      .imageSharingMode = VK_SHARING_MODE_EXCLUSIVE,
      .queueFamilyIndexCount = 0,
      .pQueueFamilyIndices = nullptr,
      .presentMode = VK_PRESENT_MODE_FIFO_KHR,
      .oldSwapchain = oldSwapchain,
      .clipped = true,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkCreateSwapchainKHR(mDevice, &swapchainCreateInfo, nullptr, &mSwapchain));

  // If we just re-created an existing swapchain, we should destroy the old
  // swapchain at this point.
  // Note: destroying the swapchain also cleans up all its associated
  // presentable images once the platform is done with them.
  if (oldSwapchain != VK_NULL_HANDLE) {
    mVkHelper.vkDestroySwapchainKHR(mDevice, oldSwapchain, nullptr);
  }

  // Query VKImage from swapchain and create VKImageView.
  ASSERT(VK_SUCCESS == mVkHelper.vkGetSwapchainImagesKHR(mDevice, mSwapchain, &mSwapchainImageCount, nullptr));
  std::vector<VkImage> swapchainImages(mSwapchainImageCount);
  ASSERT(VK_SUCCESS == mVkHelper.vkGetSwapchainImagesKHR(mDevice, mSwapchain, &mSwapchainImageCount, swapchainImages.data()));
  mSwapchainImageResources.resize(mSwapchainImageCount);
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    const VkImageViewCreateInfo imageViewCreateInfo = {
        .sType = VK_STRUCTURE_TYPE_IMAGE_VIEW_CREATE_INFO,
        .pNext = nullptr,
        .format = mFormat,
        .components = {
            .r = VK_COMPONENT_SWIZZLE_R,
            .g = VK_COMPONENT_SWIZZLE_G,
            .b = VK_COMPONENT_SWIZZLE_B,
            .a = VK_COMPONENT_SWIZZLE_A,
        },
        .subresourceRange = {
            .aspectMask = VK_IMAGE_ASPECT_COLOR_BIT,
            .baseMipLevel = 0,
            .levelCount = 1,
            .baseArrayLayer = 0,
            .layerCount = 1
        },
        .viewType = VK_IMAGE_VIEW_TYPE_2D,
        .flags = 0,
        .image = swapchainImages[i],
    };
    mSwapchainImageResources[i].image = swapchainImages[i];
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateImageView(mDevice, &imageViewCreateInfo, nullptr, &mSwapchainImageResources[i].imageView));
  }
}

void Cube::prepare() {
  if (mCommandPool == VK_NULL_HANDLE) {
    const VkCommandPoolCreateInfo commandPoolCreateInfo = {
        .sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
        .pNext = nullptr,
        .queueFamilyIndex = mGraphicsQueueFamilyIndex,
        .flags = 0,
    };
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateCommandPool(mDevice, &commandPoolCreateInfo, nullptr, &mCommandPool));
  }
  const VkCommandBufferAllocateInfo commandBufferAllocateInfo = {
      .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
      .pNext = nullptr,
      .commandPool = mCommandPool,
      .level = VK_COMMAND_BUFFER_LEVEL_PRIMARY,
      .commandBufferCount = 1,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkAllocateCommandBuffers(mDevice, &commandBufferAllocateInfo, &mCommandBuffer));
  const VkCommandBufferBeginInfo commandBufferBeginInfo = {
      .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
      .pNext = nullptr,
      .flags = 0,
      .pInheritanceInfo = nullptr,
  };
  ASSERT(VK_SUCCESS == mVkHelper.vkBeginCommandBuffer(mCommandBuffer, &commandBufferBeginInfo));
  createSwapchain();
  if (mMinimized) {
    mPrepared = false;
    return;
  }
  prepareDepth();
  prepareTextures();
  prepareDataBuffers();
  prepareDescriptorLayout();
  prepareRenderPass();
  preparePipeline();
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    ASSERT(VK_SUCCESS == mVkHelper.vkAllocateCommandBuffers(mDevice, &commandBufferAllocateInfo, &mSwapchainImageResources[i].commandBuffer));
  }
  if (mSeparatePresentQueue) {
    const VkCommandPoolCreateInfo presentCommandPoolCreateInfo = {
        .sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
        .pNext = nullptr,
        .queueFamilyIndex = mPresentQueueFamilyIndex,
        .flags = 0,
    };
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateCommandPool(mDevice, &presentCommandPoolCreateInfo, nullptr, &mPresentCommandPool));

    const VkCommandBufferAllocateInfo presentCommandBufferAllocateInfo = {
        .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
        .pNext = nullptr,
        .commandPool = mPresentCommandPool,
        .level = VK_COMMAND_BUFFER_LEVEL_PRIMARY,
        .commandBufferCount = 1,
    };
    for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
      ASSERT(VK_SUCCESS == mVkHelper.vkAllocateCommandBuffers(mDevice, &presentCommandBufferAllocateInfo, &mSwapchainImageResources[i].graphicsToPresentCommandBuffer));
      buildImageOwnershipCommand(i);
    }
  }
  prepareDescriptorPool();
  prepareDescriptorSet();
  prepareFramebuffers();
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    mCurrentBuffer = i;
    buildDrawCommands(mSwapchainImageResources[i].commandBuffer);
  }

  // Prepare functions above may generate pipeline commands
  // that need to be flushed before beginning the render loop.
  flushInitCommands();
  if (mStagingTexture.buffer) {
    destroyTexture(&mStagingTexture);
  }
  mCurrentBuffer = 0;
  mPrepared = true;
}

void Cube::createSemaphores() {
  // Create semaphores to synchronize acquiring presentable buffers before
  // rendering and waiting for drawing to be complete before presenting
  const VkSemaphoreCreateInfo semaphoreCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
  };

  // Create fences that we can use to throttle if we get too far
  // ahead of the image presents
  const VkFenceCreateInfo fenceCreateInfo = {
      .sType = VK_STRUCTURE_TYPE_FENCE_CREATE_INFO,
      .pNext = nullptr,
      .flags = VK_FENCE_CREATE_SIGNALED_BIT
  };
  mFences.resize(mSwapchainImageCount, VK_NULL_HANDLE);
  mImageAcquiredSemaphores.resize(mSwapchainImageCount, VK_NULL_HANDLE);
  mDrawCompleteSemaphores.resize(mSwapchainImageCount, VK_NULL_HANDLE);
  mImageOwnershipSemaphores.resize(mSwapchainImageCount, VK_NULL_HANDLE);
  for (uint32_t i = 0; i < mSwapchainImageCount; i++) {
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateFence(mDevice, &fenceCreateInfo, nullptr, &mFences[i]));
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateSemaphore(mDevice, &semaphoreCreateInfo, nullptr, &mImageAcquiredSemaphores[i]));
    ASSERT(VK_SUCCESS == mVkHelper.vkCreateSemaphore(mDevice, &semaphoreCreateInfo, nullptr, &mDrawCompleteSemaphores[i]));
    if (mSeparatePresentQueue) {
      ASSERT(VK_SUCCESS == mVkHelper.vkCreateSemaphore(mDevice, &semaphoreCreateInfo, nullptr, &mImageOwnershipSemaphores[i]));
    }
  }
  mFrameIndex = 0;
}

void Cube::init() {
  vec3 eye = {0.0f, 3.0f, 5.0f};
  vec3 origin = {0, 0, 0};
  vec3 up = {0.0f, 1.0f, 0.0};
  mWidth = 500;
  mHeight = 500;
  mScale = 1.0f;
  mSpinAngle = 4.0f;
  mSpinSpeed = 0.0005f;
  mat4x4_perspective(mProjectMatrix, (float)degreesToRadians(45.0f), 1.0f, 0.1f, 100.0f);
  mat4x4_look_at(mViewMatrix, eye, origin, up);
  mat4x4_identity(mModelMatrix);
  mProjectMatrix[1][1] *= -1;  // Flip projection matrix from GL to Vulkan orientation.
}

static bool initialized = false;

void Cube::Run(AndroidAppState *app) {
  mVkHelper.Init();
  mPrepared = false;
  mAppState = app;
  while(true) {
    if (!initialized) {
      mNativeWindow = app->window;
      init();
      createInstance();
      createDevice();
      prepare();
      createSemaphores();
      initialized = true;
    }

    if (app->destroyRequested != 0) {
      JavaVM* vm = app->vm;
      vm->DetachCurrentThread();
      cleanup();
      initialized = false;
      return;
    }

    if (initialized && mPrepared) {
      draw();
    }
  }
}
