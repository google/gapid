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

#include "vulkan_helper.h"

#include <android/log.h>
#include <unistd.h>

#define LOG_TAG "VulkanHelper"
#define ALOGD(...) __android_log_print(ANDROID_LOG_DEBUG, LOG_TAG, __VA_ARGS__)
#define ASSERT(cond)                                                                           \
    if (!(cond)) {                                                                             \
        __android_log_assert(#cond, LOG_TAG, "Error: " #cond " at " __FILE__ ":%d", __LINE__); \
    }

#define GET_PROC(F) \
    vk##F = reinterpret_cast<PFN_vk##F>(vkGetInstanceProcAddr(VK_NULL_HANDLE, "vk" #F))

void VulkanHelper::Init() {
  GET_PROC(CreateInstance);
  GET_PROC(EnumerateInstanceExtensionProperties);
  GET_PROC(EnumerateInstanceVersion);
}
