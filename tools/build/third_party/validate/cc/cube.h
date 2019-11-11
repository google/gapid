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

#pragma once

#include <android/log.h>
#include <android/looper.h>
#include <android/native_activity.h>

#include "linmath.h"
#include "vulkan_helper.h"

// Struct for passing state from java app to native app.
struct AndroidAppState {
  ANativeWindow* window;
  AAssetManager* assetManager;
  JavaVM* vm;
  bool running;
  bool destroyRequested;
};

// structure to track all objects related to a texture.
struct TextureObject {
  VkSampler sampler;
  VkImage image;
  VkBuffer buffer;
  VkImageLayout imageLayout;
  VkMemoryAllocateInfo memoryAllocationInfo;
  VkDeviceMemory deviceMemory;
  VkImageView view;
  uint32_t width, height;
};

struct SwapchainImageResources {
  VkImage image;
  VkCommandBuffer commandBuffer;
  VkCommandBuffer graphicsToPresentCommandBuffer;
  VkImageView imageView;
  VkBuffer uniformBuffer;
  VkDeviceMemory uniformDeviceMemory;
  VkFramebuffer framebuffer;
  VkDescriptorSet descriptorSet;
};

struct VkTexCubeVsUniform {
  // Must start with MVP
  float mvp[4][4];
  float position[12 * 3][4];
  float attr[12 * 3][4];
};

class Cube {
 public:
  // Start the cube application's render loop.
  void Run(AndroidAppState* state);

 private:
  void init();
  void draw();
  void createInstance();
  void createDevice();
  void createSwapchain();
  void createSemaphores();
  void prepare();
  void prepareDepth();
  void prepareTextures();
  void prepareTextureBuffer(const char* filename, TextureObject* textureObject);
  void prepareTextureImage(const char* filename, TextureObject* textureObject,
                           VkImageTiling tiling, VkImageUsageFlags usage, VkFlags requiredProperties);
  void setImageLayout(VkImage image,
                      VkImageAspectFlags aspectMask,
                      VkImageLayout oldImageLayout,
                      VkImageLayout newImageLayout,
                      VkAccessFlagBits srcAccessMask,
                      VkPipelineStageFlags sourceStages,
                      VkPipelineStageFlags destinationStages);
  void prepareDataBuffers();
  void prepareDescriptorLayout();
  void prepareRenderPass();
  void preparePipeline();
  void buildImageOwnershipCommand(int index);
  void prepareDescriptorPool();
  void prepareDescriptorSet();
  void prepareFramebuffers();
  void buildDrawCommands(VkCommandBuffer commandBuffer);
  void flushInitCommands();
  void updateDataBuffer();
  void updateDrawCommands();
  void destroyTexture(TextureObject* textureObject);
  void resize();
  void cleanup();
  bool loadTextureFromPPM(const char *fileName, void* data, VkSubresourceLayout* layout, int32_t* width, int32_t* height);
  void loadShaderFromFile(const char* filePath, VkShaderModule* outShader);
  bool memoryTypeFromProperties(uint32_t typeBits,
                                VkFlags requirementsMask,
                                uint32_t* typeIndex);

  VulkanHelper mVkHelper;
  AndroidAppState* mAppState;
  struct ANativeWindow *mNativeWindow;
  VkInstance mInstance;
  VkPhysicalDevice mGpu;
  VkDevice mDevice;
  VkSurfaceKHR mSurface;
  VkSwapchainKHR mSwapchain;
  VkQueue mGraphicsQueue;
  VkQueue mPresentQueue;
  VkCommandPool mCommandPool;
  VkCommandPool mPresentCommandPool;
  VkCommandBuffer mCommandBuffer;  // Buffer for initialization commands
  VkPipelineLayout mPipelineLayout;
  VkDescriptorSetLayout mDescriptorSetLayout;
  VkPipelineCache mPipelineCache;
  VkRenderPass mRenderPass;
  VkPipeline mPipeline;
  VkPhysicalDeviceProperties mGpuProperties;
  std::vector<VkQueueFamilyProperties> mQueueFamilyProperties;
  VkPhysicalDeviceMemoryProperties mPhysicalDevicememoryProperties;
  std::vector<SwapchainImageResources> mSwapchainImageResources;
  VkFormat mFormat;
  VkColorSpaceKHR mColorSpace;
  struct {
    VkFormat format;
    VkImage image;
    VkMemoryAllocateInfo memoryAllocationInfo;
    VkDeviceMemory deviceMemory;
    VkImageView view;
  } mDepth;
  TextureObject mTexture;
  TextureObject mStagingTexture;
  VkTexCubeVsUniform mUniform;
  VkShaderModule mVertexShaderModule;
  VkShaderModule mFragmentShaderModule;
  VkDescriptorPool mDescriptorPool;
  std::vector<VkSemaphore> mImageAcquiredSemaphores;
  std::vector<VkSemaphore> mDrawCompleteSemaphores;
  std::vector<VkSemaphore> mImageOwnershipSemaphores;
  std::vector<VkFence> mFences;
  uint32_t mGraphicsQueueFamilyIndex;
  uint32_t mPresentQueueFamilyIndex;
  uint32_t mSwapchainImageCount;
  uint32_t mWidth, mHeight;
  int mFrameIndex;
  mat4x4 mProjectMatrix;
  mat4x4 mViewMatrix;
  mat4x4 mModelMatrix;
  float mScale;
  float mSpinAngle;
  float mSpinSpeed;
  uint32_t mCurrentBuffer;
  uint32_t mQueueFamilyCount;
  bool mPrepared;
  bool mSeparatePresentQueue;
  bool mMinimized;
};
