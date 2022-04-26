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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include "forwards.h"
#include "struct_clone.h"

namespace gapid2 {

void VkPipelineWrapper::set_create_info(state_block* state_block_, VkPipelineCache pipelineCache,
                                        const VkGraphicsPipelineCreateInfo* info) {
  cache = pipelineCache;
  bind = VK_PIPELINE_BIND_POINT_GRAPHICS;
  graphics_info = mem.get_typed_memory<VkGraphicsPipelineCreateInfo>(1);
  clone(
      state_block_, info[0], graphics_info[0], &mem,
      _VkGraphicsPipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_VkSpecializationMapEntry_size_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_clone,
      _VkGraphicsPipelineCreateInfo_pVertexInputState_valid,
      _VkGraphicsPipelineCreateInfo_pInputAssemblyState_valid,
      _VkGraphicsPipelineCreateInfo_pTessellationState_valid,
      _VkGraphicsPipelineCreateInfo_pViewportState_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_pViewports_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_x_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_y_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_width_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_height_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_pScissors_valid,
      _VkGraphicsPipelineCreateInfo_pMultisampleState_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineMultisampleStateCreateInfo_VkPipelineSampleLocationsStateCreateInfoEXT_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineMultisampleStateCreateInfo_pSampleMask_length,
      _VkGraphicsPipelineCreateInfo_pDepthStencilState_valid,
      _VkGraphicsPipelineCreateInfo_pColorBlendState_valid,
      _VkGraphicsPipelineCreateInfo_VkPipelineColorBlendStateCreateInfo_logicOp_valid,
      _VkGraphicsPipelineCreateInfo_basePipelineHandle_valid);
}

void VkPipelineWrapper::set_create_info(state_block* state_block_, VkPipelineCache pipelineCache,
                                        const VkComputePipelineCreateInfo* info) {
  cache = pipelineCache;
  compute_info = mem.get_typed_memory<VkComputePipelineCreateInfo>(1);
  bind = VK_PIPELINE_BIND_POINT_COMPUTE;
  clone(
      state_block_, info[0], compute_info[0], &mem,
      _VkComputePipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_VkSpecializationMapEntry_size_valid,
      _VkComputePipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_clone,
      _VkComputePipelineCreateInfo_basePipelineHandle_valid);
}

}  // namespace gapid2
