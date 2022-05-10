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

#include "pipeline.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_pipelines(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  for (auto& it : state_block->VkPipelines) {
    VkPipelineWrapper* pipe = it.second.second;
    VkPipeline pipeline = it.first;
    if (pipe->bind == VK_PIPELINE_BIND_POINT_COMPUTE) {
      auto create_info = *pipe->get_compute_create_info();
#pragma TODO(awoloszyn, Figure out if we want to handle pipeline inheritance.In theory it only matters for performance)
      create_info.basePipelineHandle = VK_NULL_HANDLE;
      create_info.basePipelineIndex = -1;
      serializer->vkCreateComputePipelines(pipe->device, pipe->cache, 1, &create_info, nullptr, &pipeline);
    } else {
      GAPID2_ASSERT(pipe->bind == VK_PIPELINE_BIND_POINT_GRAPHICS, "Unknown pipeline type")
      auto create_info = *pipe->get_graphics_create_info();
#pragma TODO(awoloszyn, Figure out if we want to handle pipeline inheritance.In theory it only matters for performance)
      create_info.basePipelineHandle = VK_NULL_HANDLE;
      create_info.basePipelineIndex = -1;
      serializer->vkCreateGraphicsPipelines(pipe->device, pipe->cache, 1, &create_info, nullptr, &pipeline);
    }
  }
}

}  // namespace gapid2