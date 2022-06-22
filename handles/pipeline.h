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

#include "device.h"
#include "handles.h"
#include "shader_module.h"
#include "temporary_allocator.h"

namespace gapid2 {
class state_block;
struct VkPipelineWrapper : handle_base<VkPipeline> {
  VkPipelineWrapper(VkPipeline pipeline)
      : handle_base<VkPipeline>(pipeline) {}

  void set_create_info(VkDevice device_, state_block* state_block_, VkPipelineCache pipelineCache,
                       const VkGraphicsPipelineCreateInfo* info);

  void set_create_info(VkDevice device_, state_block* state_block_, VkPipelineCache pipelineCache,
                       const VkComputePipelineCreateInfo* info);

  const VkGraphicsPipelineCreateInfo* get_graphics_create_info() const {
    return graphics_info;
  }
  const VkComputePipelineCreateInfo* get_compute_create_info() const {
    return compute_info;
  }

  VkDevice device;
  VkPipelineCache cache = VK_NULL_HANDLE;
  VkPipelineBindPoint bind;
  VkGraphicsPipelineCreateInfo* graphics_info = nullptr;
  VkComputePipelineCreateInfo* compute_info = nullptr;
  std::vector<std::shared_ptr<std::vector<uint32_t>>> shader_code;
  std::vector<descriptor_usage> usages;
  temporary_allocator mem;
};
}  // namespace gapid2
