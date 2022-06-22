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
  serializer->insert_annotation("MecPipelines");
  for (auto& it : state_block->VkPipelines) {
    VkPipelineWrapper* pipe = it.second.second;
    VkPipeline pipeline = it.first;
    if (pipe->bind == VK_PIPELINE_BIND_POINT_COMPUTE) {
      auto create_info = *pipe->get_compute_create_info();
#pragma TODO(awoloszyn, Figure out if we want to handle pipeline inheritance.In theory it only matters for performance)
      create_info.basePipelineHandle = VK_NULL_HANDLE;
      create_info.basePipelineIndex = -1;

      if (state_block->VkShaderModules.count(create_info.stage.module) == 0) {
        // The shader module was removed, create a temporary one.
        VkShaderModuleCreateInfo inf{
            .sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
            .pNext = nullptr,
            .flags = 0,
            .codeSize = pipe->shader_code[0]->size() * sizeof(uint32_t),
            .pCode = pipe->shader_code[0]->data()};

        GAPID2_ASSERT(VK_SUCCESS == bypass_caller->vkCreateShaderModule(pipe->device, &inf, nullptr, &create_info.stage.module),
                      "Could not create shader module");
        serializer->vkCreateShaderModule(pipe->device, &inf, nullptr, &create_info.stage.module);
      }

      serializer->vkCreateComputePipelines(pipe->device, pipe->cache, 1, &create_info, nullptr, &pipeline);

      if (create_info.stage.module != pipe->get_compute_create_info()->stage.module) {
        bypass_caller->vkDestroyShaderModule(pipe->device, create_info.stage.module, nullptr);
        serializer->vkDestroyShaderModule(pipe->device, create_info.stage.module, nullptr);
      }
    } else {
      GAPID2_ASSERT(pipe->bind == VK_PIPELINE_BIND_POINT_GRAPHICS, "Unknown pipeline type")
      auto create_info = *pipe->get_graphics_create_info();
#pragma TODO(awoloszyn, Figure out if we want to handle pipeline inheritance.In theory it only matters for performance)
      create_info.basePipelineHandle = VK_NULL_HANDLE;
      create_info.basePipelineIndex = -1;

      std::vector<VkPipelineShaderStageCreateInfo> stages(
          create_info.pStages, create_info.pStages + create_info.stageCount);
      create_info.pStages = stages.data();
      for (uint32_t i = 0; i < create_info.stageCount; ++i) {
        if (0 == state_block->VkShaderModules.count(stages[i].module)) {
          // The shader module was removed, create a temporary one.
          VkShaderModuleCreateInfo inf{
              .sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
              .pNext = nullptr,
              .flags = 0,
              .codeSize = pipe->shader_code[i]->size() * sizeof(uint32_t),
              .pCode = pipe->shader_code[i]->data()};

          GAPID2_ASSERT(VK_SUCCESS == bypass_caller->vkCreateShaderModule(pipe->device, &inf, nullptr, &stages[i].module),
                        "Could not create shader module");
          serializer->vkCreateShaderModule(pipe->device, &inf, nullptr, &stages[i].module);
        }
      }
      serializer->vkCreateGraphicsPipelines(pipe->device, pipe->cache, 1, &create_info, nullptr, &pipeline);

      for (uint32_t i = 0; i < create_info.stageCount; ++i) {
        if (create_info.pStages[i].module != pipe->get_graphics_create_info()->pStages[i].module) {
          bypass_caller->vkDestroyShaderModule(pipe->device, create_info.pStages[i].module, nullptr);
          serializer->vkDestroyShaderModule(pipe->device, create_info.pStages[i].module, nullptr);
        }
      }
    }
  }
}

}  // namespace gapid2