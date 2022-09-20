/*
 * Copyright (C) 2022 Google Inc.
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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "handles.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
struct VkRenderPassWrapper : handle_base<VkRenderPass> {
  VkRenderPassWrapper(VkRenderPass render_pass)
      : handle_base<VkRenderPass>(render_pass) {}

  void set_create_info(VkDevice device_, state_block* state_block_,
                       const VkRenderPassCreateInfo* pCreateInfo);
  void set_create_info2(VkDevice device_, state_block* state_block_, const VkRenderPassCreateInfo2* pCreateInfo);
  void set_create_info2_khr(VkDevice device_, state_block* state_block_, const VkRenderPassCreateInfo2* pCreateInfo);

  const VkRenderPassCreateInfo* get_create_info() const {
    return create_info;
  }

  const VkRenderPassCreateInfo2* get_create_info2() const {
    return create_info2;
  }

  const VkRenderPassCreateInfo2* get_create_info2_khr() const {
    return create_info2_khr;
  }

  VkDevice device = VK_NULL_HANDLE;
  VkRenderPassCreateInfo* create_info = nullptr;
  VkRenderPassCreateInfo2* create_info2 = nullptr;
  VkRenderPassCreateInfo2* create_info2_khr = nullptr;
  temporary_allocator mem;
};
}  // namespace gapid2
