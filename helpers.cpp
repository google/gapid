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
#include "utils.h"

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

void _VkMemoryAllocateInfo_VkImportMemoryHostPointerInfoEXT_pHostPointer_clone(const VkMemoryAllocateInfo& self, const VkImportMemoryHostPointerInfoEXT& _src, VkImportMemoryHostPointerInfoEXT& _dst, temporary_allocator* _mem) {
  _dst.pHostPointer = _src.pHostPointer;
}

void _VkMemoryAllocateInfo_VkImportMemoryHostPointerInfoEXT_pHostPointer_serialize(const VkMemoryAllocateInfo& self, const VkImportMemoryHostPointerInfoEXT& _self, encoder* _enc) {
  GAPID2_ERROR("Unimplemented: _VkMemoryAllocateInfo_VkImportMemoryHostPointerInfoEXT_pHostPointer_serialize");
}
void _VkMemoryAllocateInfo_VkImportMemoryHostPointerInfoEXT_pHostPointer_deserialize(const VkMemoryAllocateInfo& self, VkImportMemoryHostPointerInfoEXT& _self, decoder* _dec) {
  GAPID2_ERROR("Unimplemented: _VkMemoryAllocateInfo_VkImportMemoryHostPointerInfoEXT_pHostPointer_deserialize");
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

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceSubgroupProperties_supportedStages_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceSubgroupProperties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceSubgroupProperties_supportedOperations_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceSubgroupProperties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceSubgroupProperties_quadOperationsInAllStages_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceSubgroupProperties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceVulkan11Properties_subgroupSize_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceVulkan11Properties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceVulkan11Properties_subgroupSupportedStages_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceVulkan11Properties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceVulkan11Properties_subgroupSupportedOperations_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceVulkan11Properties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceVulkan11Properties_subgroupQuadOperationsInAllStages_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceVulkan11Properties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkPhysicalDeviceProperties2_VkPhysicalDeviceSubgroupProperties_subgroupSize_valid(
    const VkPhysicalDeviceProperties2& self,
    const VkPhysicalDeviceSubgroupProperties& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkDescriptorUpdateTemplateCreateInfo_descriptorSetLayout_valid(
    const VkDescriptorUpdateTemplateCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkDescriptorUpdateTemplateCreateInfo_pipelineBindPoint_valid(
    const VkDescriptorUpdateTemplateCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkDescriptorUpdateTemplateCreateInfo_pipelineLayout_valid(
    const VkDescriptorUpdateTemplateCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkDescriptorUpdateTemplateCreateInfo_set_valid(
    const VkDescriptorUpdateTemplateCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassCreateInfo2_VkSubpassDescription2_VkSubpassDescriptionDepthStencilResolve_depthResolveMode_valid(
    const VkRenderPassCreateInfo2& self,
    const VkSubpassDescription2& _self,
    const VkSubpassDescriptionDepthStencilResolve& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassCreateInfo2_VkSubpassDescription2_VkSubpassDescriptionDepthStencilResolve_stencilResolveMode_valid(
    const VkRenderPassCreateInfo2& self,
    const VkSubpassDescription2& _self,
    const VkSubpassDescriptionDepthStencilResolve& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassCreateInfo2_VkSubpassDescription2_VkSubpassDescriptionDepthStencilResolve_VkAttachmentReference2_aspectMask_valid(
    const VkRenderPassCreateInfo2& self,
    const VkSubpassDescription2& _self,
    const VkSubpassDescriptionDepthStencilResolve& __self,
    const VkAttachmentReference2& ___self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassCreateInfo2_VkSubpassDescription2_VkAttachmentReference2_aspectMask_valid(
    const VkRenderPassCreateInfo2& self,
    const VkSubpassDescription2& _self,
    const VkAttachmentReference2& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid(
    const VkSampleLocationsInfoEXT& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkQueryPoolCreateInfo_pipelineStatistics_valid(
    const VkQueryPoolCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkGraphicsPipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_VkSpecializationMapEntry_size_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    const VkSpecializationInfo& __self,
    const VkSpecializationMapEntry& ___self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_x_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineViewportStateCreateInfo& _self,
    const VkViewport& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_y_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineViewportStateCreateInfo& _self,
    const VkViewport& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_width_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineViewportStateCreateInfo& _self,
    const VkViewport& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkGraphicsPipelineCreateInfo_VkPipelineViewportStateCreateInfo_VkViewport_height_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineViewportStateCreateInfo& _self,
    const VkViewport& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkGraphicsPipelineCreateInfo_VkPipelineMultisampleStateCreateInfo_VkPipelineSampleLocationsStateCreateInfoEXT_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineMultisampleStateCreateInfo& _self,
    const VkPipelineSampleLocationsStateCreateInfoEXT& __self,
    const VkSampleLocationsInfoEXT& ___self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkGraphicsPipelineCreateInfo_VkPipelineColorBlendStateCreateInfo_logicOp_valid(
    const VkGraphicsPipelineCreateInfo& self,
    const VkPipelineColorBlendStateCreateInfo& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkGraphicsPipelineCreateInfo_basePipelineHandle_valid(
    const VkGraphicsPipelineCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkComputePipelineCreateInfo_VkPipelineShaderStageCreateInfo_VkSpecializationInfo_VkSpecializationMapEntry_size_valid(
    const VkComputePipelineCreateInfo& self,
    const VkPipelineShaderStageCreateInfo& _self,
    const VkSpecializationInfo& __self,
    const VkSpecializationMapEntry& ___self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkComputePipelineCreateInfo_basePipelineHandle_valid(
    const VkComputePipelineCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkSamplerCreateInfo_VkSamplerCustomBorderColorCreateInfoEXT_customBorderColor_valid(
    const VkSamplerCreateInfo& self,
    const VkSamplerCustomBorderColorCreateInfoEXT& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkWriteDescriptorSet_dstSet_valid(const VkWriteDescriptorSet& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkWriteDescriptorSet_VkDescriptorImageInfo_sampler_valid(
    const VkWriteDescriptorSet& self,
    const VkDescriptorImageInfo& _self) {
  return self.descriptorType == VK_DESCRIPTOR_TYPE_SAMPLER ||
         self.descriptorType == VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER;
}
bool _VkWriteDescriptorSet_VkDescriptorImageInfo_imageView_valid(
    const VkWriteDescriptorSet& self,
    const VkDescriptorImageInfo& _self) {
  return self.descriptorType == VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE ||
         self.descriptorType == VK_DESCRIPTOR_TYPE_STORAGE_IMAGE ||
         self.descriptorType == VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER ||
         self.descriptorType == VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT;
}
bool _VkWriteDescriptorSet_VkDescriptorImageInfo_imageLayout_valid(
    const VkWriteDescriptorSet& self,
    const VkDescriptorImageInfo& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkCommandBufferBeginInfo_VkCommandBufferInheritanceInfo_renderPass_valid(
    const VkCommandBufferBeginInfo& self,
    const VkCommandBufferInheritanceInfo& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkCommandBufferBeginInfo_VkCommandBufferInheritanceInfo_framebuffer_valid(
    const VkCommandBufferBeginInfo& self,
    const VkCommandBufferInheritanceInfo& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkCommandBufferBeginInfo_VkCommandBufferInheritanceInfo_queryFlags_valid(
    const VkCommandBufferBeginInfo& self,
    const VkCommandBufferInheritanceInfo& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkCommandBufferBeginInfo_VkCommandBufferInheritanceInfo_pipelineStatistics_valid(
    const VkCommandBufferBeginInfo& self,
    const VkCommandBufferInheritanceInfo& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkViewport_x_valid(const VkViewport& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkViewport_y_valid(const VkViewport& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkViewport_width_valid(const VkViewport& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkViewport_height_valid(const VkViewport& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkBufferCopy_size_valid(const VkBufferCopy& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkClearAttachment_clearValue_valid(const VkClearAttachment& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkClearAttachment_VkClearValue_color_valid(const VkClearAttachment& self,
                                                 const VkClearValue& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkBufferMemoryBarrier_srcAccessMask_valid(
    const VkBufferMemoryBarrier& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkBufferMemoryBarrier_dstAccessMask_valid(
    const VkBufferMemoryBarrier& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkImageMemoryBarrier_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid(
    const VkImageMemoryBarrier& self,
    const VkSampleLocationsInfoEXT& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkImageMemoryBarrier_srcAccessMask_valid(
    const VkImageMemoryBarrier& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkImageMemoryBarrier_dstAccessMask_valid(
    const VkImageMemoryBarrier& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassBeginInfo_VkRenderPassSampleLocationsBeginInfoEXT_VkAttachmentSampleLocationsEXT_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid(
    const VkRenderPassBeginInfo& self,
    const VkRenderPassSampleLocationsBeginInfoEXT& _self,
    const VkAttachmentSampleLocationsEXT& __self,
    const VkSampleLocationsInfoEXT& ___self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassBeginInfo_VkRenderPassSampleLocationsBeginInfoEXT_VkSubpassSampleLocationsEXT_VkSampleLocationsInfoEXT_sampleLocationsPerPixel_valid(
    const VkRenderPassBeginInfo& self,
    const VkRenderPassSampleLocationsBeginInfoEXT& _self,
    const VkSubpassSampleLocationsEXT& __self,
    const VkSampleLocationsInfoEXT& ___self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkRenderPassBeginInfo_VkClearValue_color_valid(
    const VkRenderPassBeginInfo& self,
    const VkClearValue& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkBindImageMemoryInfo_memory_valid(const VkBindImageMemoryInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

bool _VkDeviceCreateInfo_VkPhysicalDeviceFeatures2_VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures_shaderSubgroupExtendedTypes_valid(
    const VkDeviceCreateInfo& self,
    const VkPhysicalDeviceFeatures2& _self,
    const VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures& __self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkDeviceCreateInfo_VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures_shaderSubgroupExtendedTypes_valid(
    const VkDeviceCreateInfo& self,
    const VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkSamplerCreateInfo_compareOp_valid(const VkSamplerCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkSamplerCreateInfo_borderColor_valid(const VkSamplerCreateInfo& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkDescriptorSetLayoutCreateInfo_VkDescriptorSetLayoutBinding_stageFlags_valid(
    const VkDescriptorSetLayoutCreateInfo& self,
    const VkDescriptorSetLayoutBinding& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
bool _VkPhysicalDeviceFeatures2_VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures_shaderSubgroupExtendedTypes_valid(
    const VkPhysicalDeviceFeatures2& self,
    const VkPhysicalDeviceShaderSubgroupExtendedTypesFeatures& _self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}

#ifdef VK_USE_PLATFORM_XCB_KHR
bool _VkXcbSurfaceCreateInfoKHR_connection_valid(const VkXcbSurfaceCreateInfoKHR& self) {
#pragma FIXME(awoloszyn, fill this in)
  return true;
}
#endif

}  // namespace gapid2