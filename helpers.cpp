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

#include "helpers.h"
#include "decoder.h"
#include "encoder.h"
namespace gapid2 {

bool _VkRenderPassBeginInfo_pClearValues_valid(
    const VkRenderPassBeginInfo& self) {
  (void)self;
  return self.pClearValues;
}

void _VkAllocationCallbacks_pUserData_serialize(
    const VkAllocationCallbacks& self,
    encoder* enc) {
  (void)self;
  (void)enc;
}
void _VkAllocationCallbacks_pUserData_deserialize(VkAllocationCallbacks& self,
                                                  decoder* dec) {
  (void)self;
  (void)dec;
}
bool _VkBufferCreateInfo_pQueueFamilyIndices_valid(
    const VkBufferCreateInfo& self) {
  if (self.sharingMode == VK_SHARING_MODE_CONCURRENT) {
    return true;
  }
  return false;
}
bool _VkImageCreateInfo_pQueueFamilyIndices_valid(
    const VkImageCreateInfo& self) {
  if (self.sharingMode == VK_SHARING_MODE_CONCURRENT) {
    return true;
  }
  return false;
}

uint64_t _VkShaderModuleCreateInfo_pCode_length(
    const VkShaderModuleCreateInfo& self) {
  return self.codeSize / sizeof(uint32_t);
}

void _VkPipelineCacheCreateInfo_pInitialData_serialize(
    const VkPipelineCacheCreateInfo& self,
    encoder* enc) {
  enc->encode_primitive_array(
      static_cast<const unsigned char*>(self.pInitialData),
      self.initialDataSize);
}
void _VkPipelineCacheCreateInfo_pInitialData_deserialize(
    VkPipelineCacheCreateInfo& self,
    decoder* dec) {
  if (self.initialDataSize != 0) {
    unsigned char* c =
        dec->get_typed_memory<unsigned char>(self.initialDataSize);
    dec->decode_primitive_array(c, self.initialDataSize);
    self.pInitialData = c;
  }
}

void _VkGraphicsPipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_serialize(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    const VkSpecializationInfo& __self,
    encoder* __enc) {
  (void)self;
  (void)_self;
  __enc->encode_primitive_array<unsigned char>(
      static_cast<const unsigned char*>(__self.pData), __self.dataSize);
}

void _VkGraphicsPipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_deserialize(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    VkSpecializationInfo& __self,
    decoder* __dec) {
  (void)self;
  (void)_self;
  if (__self.dataSize != 0) {
    unsigned char* c = __dec->get_typed_memory<unsigned char>(__self.dataSize);
    __dec->decode_primitive_array(c, __self.dataSize);
    __self.pData = c;
  }
}

bool _VkGraphicsPipelineCreateInfo_pVertexInputState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  for (size_t i = 0; i < self.stageCount; ++i) {
    if ((self.pStages[i].stage & VK_SHADER_STAGE_MESH_BIT_NV) ==
        VK_SHADER_STAGE_MESH_BIT_NV) {
      return false;
      return false;
    }
  }
  if (self.pDynamicState) {
    for (size_t i = 0; i < self.pDynamicState->dynamicStateCount; ++i) {
      if (self.pDynamicState->pDynamicStates[i] ==
          VK_DYNAMIC_STATE_VERTEX_INPUT_EXT) {
        return false;
      }
    }
  }
  return true;
}

bool _VkGraphicsPipelineCreateInfo_pInputAssemblyState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  for (size_t i = 0; i < self.stageCount; ++i) {
    if ((self.pStages[i].stage & VK_SHADER_STAGE_MESH_BIT_NV) ==
        VK_SHADER_STAGE_MESH_BIT_NV) {
      return false;
    }
  }
  return true;
}
bool _VkGraphicsPipelineCreateInfo_pTessellationState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  {
    for (size_t i = 0; i < self.stageCount; ++i) {
      if ((self.pStages[i].stage & VK_SHADER_STAGE_TESSELLATION_CONTROL_BIT) ==
          VK_SHADER_STAGE_TESSELLATION_CONTROL_BIT) {
        return true;
      }
      if ((self.pStages[i].stage &
           VK_SHADER_STAGE_TESSELLATION_EVALUATION_BIT) ==
          VK_SHADER_STAGE_TESSELLATION_EVALUATION_BIT) {
        return true;
      }
    }
  }
  return false;
}
bool _VkGraphicsPipelineCreateInfo_pViewportState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  for (size_t i = 0; i < self.stageCount; ++i) {
    if ((self.pStages[i].stage & VK_SHADER_STAGE_FRAGMENT_BIT) ==
        VK_SHADER_STAGE_FRAGMENT_BIT) {
      return true;
    }
  }
  return false;
}
bool _VkGraphicsPipelineCreateInfo_pMultisampleState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  for (size_t i = 0; i < self.stageCount; ++i) {
    if ((self.pStages[i].stage & VK_SHADER_STAGE_FRAGMENT_BIT) ==
        VK_SHADER_STAGE_FRAGMENT_BIT) {
      return true;
    }
  }
  return false;
}
bool _VkGraphicsPipelineCreateInfo_pDepthStencilState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  // Not quite valid, we also have to ignore if the subpass (used in renderpass
  // creation) did not have a depth buffer
  if (!self.pDepthStencilState) {
    return false;
  }
  for (size_t i = 0; i < self.stageCount; ++i) {
    if ((self.pStages[i].stage & VK_SHADER_STAGE_FRAGMENT_BIT) ==
        VK_SHADER_STAGE_FRAGMENT_BIT) {
      return true;
    }
  }
  return false;
}
bool _VkGraphicsPipelineCreateInfo_pColorBlendState_valid(
    const VkGraphicsPipelineCreateInfo& self) {
  // Not quite valid, we also have to ignore if the subpass (used in renderpass
  // creation) did not have a color attachments
  for (size_t i = 0; i < self.stageCount; ++i) {
    if ((self.pStages[i].stage & VK_SHADER_STAGE_FRAGMENT_BIT) ==
        VK_SHADER_STAGE_FRAGMENT_BIT) {
      return true;
    }
  }
  return false;
}

bool _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_pViewports_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineViewportStateCreateInfo& _self) {
  if (self.pDynamicState) {
    for (size_t i = 0; i < self.pDynamicState->dynamicStateCount; ++i) {
      if (self.pDynamicState->pDynamicStates[i] == VK_DYNAMIC_STATE_VIEWPORT) {
        return false;
      }
    }
  }
  return true;
}
bool _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_pScissors_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineViewportStateCreateInfo& _self) {
  if (self.pDynamicState) {
    for (size_t i = 0; i < self.pDynamicState->dynamicStateCount; ++i) {
      if (self.pDynamicState->pDynamicStates[i] == VK_DYNAMIC_STATE_SCISSOR) {
        return false;
      }
    }
  }
  return true;
}

uint64_t
_VkGraphicsPipelineCreateInfo_VkPipelineMultisampleStateCreateInfo_pSampleMask_length(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineMultisampleStateCreateInfo& _self) {
  (void)self;
  return (_self.rasterizationSamples + 31) / 32;
}

void _VkComputePipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_serialize(
    const VkComputePipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    const VkSpecializationInfo& __self,
    encoder* __enc) {
  (void)self;
  (void)_self;
  __enc->encode_primitive_array<unsigned char>(
      static_cast<const unsigned char*>(__self.pData), __self.dataSize);
}

void _VkComputePipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_deserialize(
    const VkComputePipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    VkSpecializationInfo& __self,
    decoder* __dec) {
  (void)self;
  (void)_self;
  if (__self.dataSize != 0) {
    unsigned char* c = __dec->get_typed_memory<unsigned char>(__self.dataSize);
    __dec->decode_primitive_array(c, __self.dataSize);
    __self.pData = c;
  }
}

void _VkGraphicsPipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_clone(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    const VkSpecializationInfo& __src,
    VkSpecializationInfo& __dst,
    temporary_allocator* __mem) {
  (void)self;
  (void)_self;
  if (__src.dataSize != 0) {
    void* pd = __mem->get_typed_memory<char>(__src.dataSize);
    memcpy(pd, __src.pData, __src.dataSize);
    __dst.pData = pd;
  }
}

void _VkPipelineCacheCreateInfo_pInitialData_clone(
    const VkPipelineCacheCreateInfo& src,
    VkPipelineCacheCreateInfo& dst,
    temporary_allocator* __mem) {
  dst.pInitialData = __mem->get_memory(src.initialDataSize);
  memcpy(const_cast<void*>(dst.pInitialData), src.pInitialData,
         src.initialDataSize);
}

void _VkComputePipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_pData_clone(
    const VkComputePipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    const VkSpecializationInfo& __src,
    VkSpecializationInfo& __dst,
    temporary_allocator* __mem) {
  (void)self;
  (void)_self;
  if (__src.dataSize != 0) {
    void* pd = __mem->get_typed_memory<char>(__src.dataSize);
    memcpy(pd, __src.pData, __src.dataSize);
    __dst.pData = pd;
  }
}

bool _VkWriteDescriptorSet_pImageInfo_valid(const VkWriteDescriptorSet& self) {
  if (self.descriptorType == VK_DESCRIPTOR_TYPE_SAMPLER ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_STORAGE_IMAGE ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT) {
    return true;
  }
  return false;
}

bool _VkWriteDescriptorSet_pBufferInfo_valid(const VkWriteDescriptorSet& self) {
  if (self.descriptorType == VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_STORAGE_BUFFER ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC) {
    return true;
  }
  return false;
}

bool _VkWriteDescriptorSet_pTexelBufferView_valid(
    const VkWriteDescriptorSet& self) {
  if (self.descriptorType == VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER ||
      self.descriptorType == VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER) {
    return true;
  }
  return false;
}

bool _VkFramebufferCreateInfo_pAttachments_valid(
    const VkFramebufferCreateInfo& self) {
  if (self.flags & VK_FRAMEBUFFER_CREATE_IMAGELESS_BIT) {
    return false;
  }
  return true;
}

bool _VkCommandBufferBeginInfo_pInheritanceInfo_valid(
    const VkCommandBufferBeginInfo& self) {
#pragma FIXME(awoloszyn, This is slightly wrong, \
              we need CommandPool information)
  return self.pInheritanceInfo != nullptr;
}

bool _VkSwapchainCreateInfoKHR_pQueueFamilyIndices_valid(
    const VkSwapchainCreateInfoKHR& self) {
  if (self.imageSharingMode == VK_SHARING_MODE_CONCURRENT) {
    return true;
  }
  return false;
}

bool _VkDescriptorSetLayoutCreateInfo_VkDescriptorSetLayoutBinding_pImmutableSamplers_valid(
    const VkDescriptorSetLayoutCreateInfo& self,
    const VkDescriptorSetLayoutBinding& _self) {
  (void)self;
  if ((_self.descriptorType == VK_DESCRIPTOR_TYPE_SAMPLER ||
       _self.descriptorType == VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER) &&
      _self.pImmutableSamplers != nullptr) {
    return true;
  }
  return false;
}

}  // namespace gapid2