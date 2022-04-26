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
#define NOMINMAX
#include <externals/SPIRV-Reflect/spirv_reflect.h>
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <algorithm>
#include <unordered_map>
#include <unordered_set>

#include "creation_data_tracker.h"

namespace gapid2 {
template <typename T>
class StateTracker : public CreationDataTracker<T> {
 protected:
  using super = T;

 public:
  VkResult vkCreateShaderModule(VkDevice device,
                                const VkShaderModuleCreateInfo* pCreateInfo,
                                const VkAllocationCallbacks* pAllocator,
                                VkShaderModule* pShaderModule) override {
    auto res = super::vkCreateShaderModule(device, pCreateInfo, pAllocator,
                                           pShaderModule);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto sm = this->updater_.cast_from_vk(pShaderModule[0]);

    spv_reflect::ShaderModule smod(pCreateInfo->codeSize, pCreateInfo->pCode);
    if (smod.GetResult() != SPV_REFLECT_RESULT_SUCCESS) {
      return res;
    }

    for (uint32_t i = 0; i < smod.GetEntryPointCount(); ++i) {
      const char* epName = smod.GetEntryPointName(i);
      uint32_t count = 0;

      if (SPV_REFLECT_RESULT_SUCCESS !=
          smod.EnumerateEntryPointDescriptorSets(epName, &count, nullptr)) {
        continue;
      }
      if (count == 0) {
        // Correct but zero count. Make sure we note this has at least been
        // parsed correctly.
        sm->_usage.insert(std::make_pair(std::string(epName),
                                         std::vector<descriptor_usage>()));
        continue;
      }
      std::vector<SpvReflectDescriptorSet*> sets;
      sets.resize(count);
      if (SPV_REFLECT_RESULT_SUCCESS !=
          smod.EnumerateEntryPointDescriptorSets(epName, &count, sets.data())) {
        continue;
      }
      auto du = sm->_usage.insert(
          std::make_pair(std::string(epName), std::vector<descriptor_usage>()));
      for (auto& set : sets) {
        for (size_t i = 0; i < set->binding_count; ++i) {
          auto binding = set->bindings[i];
          uint32_t count = 1;
          for (size_t j = 0; j < binding->array.dims_count; ++j) {
            count *= binding->array.dims[j];
          }
          du.first->second.push_back(
              descriptor_usage{binding->set, binding->binding, count});
        }
      }
    }

    return res;
  }

  VkResult vkCreateGraphicsPipelines(
      VkDevice device,
      VkPipelineCache pipelineCache,
      uint32_t createInfoCount,
      const VkGraphicsPipelineCreateInfo* pCreateInfos,
      const VkAllocationCallbacks* pAllocator,
      VkPipeline* pPipelines) override {
    auto res =
        super::vkCreateGraphicsPipelines(device, pipelineCache, createInfoCount,
                                         pCreateInfos, pAllocator, pPipelines);
    if (res != VK_SUCCESS) {
      return res;
    }

    bool use_all = false;
    std::vector<descriptor_usage> usages;

    for (size_t i = 0; i < createInfoCount && !use_all; ++i) {
      auto gp = this->updater_.cast_from_vk(pPipelines[i]);
      for (size_t j = 0; j < pCreateInfos[i].stageCount; ++j) {
        auto& mod = pCreateInfos[i].pStages[j].module;
        auto stage =
            this->updater_.cast_from_vk(pCreateInfos[i].pStages[j].module);
        auto dsd = stage->_usage.find(pCreateInfos[i].pStages[j].pName);
        if (dsd == stage->_usage.end()) {
          use_all = true;
          break;
        }
        for (auto& su : dsd->second) {
          auto f = std::find_if(usages.begin(), usages.end(),
                                [&su](const descriptor_usage& usage) {
                                  return usage.binding == su.binding &&
                                         usage.set == su.set;
                                });
          if (f != usages.end()) {
            f->count = f->count > su.count ? f->count : su.count;
            continue;
          }
          usages.push_back(su);
        }
      }
      if (use_all) {
        usages.clear();
        // If we could not find usages for a particular stage,
        // then we fallback to assuming every descriptor is used.
        auto pl = this->updater_.cast_from_vk(pCreateInfos[i].layout);
        for (uint32_t j = 0; j < pl->create_info->setLayoutCount; ++j) {
          auto dsl =
              this->updater_.cast_from_vk(pl->create_info->pSetLayouts[j]);
          for (uint32_t k = 0; k < dsl->create_info->bindingCount; ++k) {
            usages.push_back(descriptor_usage{
                j, dsl->create_info->pBindings[k].binding,
                dsl->create_info->pBindings[k].descriptorCount});
          }
        }
      }
      gp->usages = std::move(usages);
    }
    return res;
  }

  VkResult vkCreatePipelineLayout(VkDevice device,
                                  const VkPipelineLayoutCreateInfo* pCreateInfo,
                                  const VkAllocationCallbacks* pAllocator,
                                  VkPipelineLayout* pPipelineLayout) override {
    auto res = super::vkCreatePipelineLayout(device, pCreateInfo, pAllocator,
                                             pPipelineLayout);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto pl = this->updater_.cast_from_vk(pPipelineLayout[0]);
    pl->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateComputePipelines(
      VkDevice device,
      VkPipelineCache pipelineCache,
      uint32_t createInfoCount,
      const VkComputePipelineCreateInfo* pCreateInfos,
      const VkAllocationCallbacks* pAllocator,
      VkPipeline* pPipelines) override {
    auto res =
        super::vkCreateComputePipelines(device, pipelineCache, createInfoCount,
                                        pCreateInfos, pAllocator, pPipelines);
    if (res != VK_SUCCESS) {
      return res;
    }
    bool use_all = false;

    std::vector<descriptor_usage> usages;
    for (size_t i = 0; i < createInfoCount && !use_all; ++i) {
      usages.clear();
      auto gp = this->updater_.cast_from_vk(pPipelines[i]);
      auto& mod = pCreateInfos[i].stage.module;
      auto stage = this->updater_.cast_from_vk(pCreateInfos[i].stage.module);
      auto dsd = stage->_usage.find(pCreateInfos[i].stage.pName);
      if (dsd == stage->_usage.end()) {
        // If we could not find usages for this stage, it means
        // that this shader could not be parsed by spirv-reflect.
        // This is a backup slow-path for such shaders. We
        // assume every descriptor accessible from the pipeline layout
        // is used.
        auto pl = this->updater_.cast_from_vk(pCreateInfos[i].layout);
        for (uint32_t j = 0; j < pl->create_info->setLayoutCount; ++j) {
          auto dsl =
              this->updater_.cast_from_vk(pl->create_info->pSetLayouts[j]);
          for (uint32_t k = 0; k < dsl->create_info->bindingCount; ++k) {
            usages.push_back(descriptor_usage{
                j, dsl->create_info->pBindings[k].binding,
                dsl->create_info->pBindings[k].descriptorCount});
          }
        }
      } else {
        // Fast path, only find descriptors that are actually used
        // by the shader module.
        for (auto& su : dsd->second) {
          auto f = std::find_if(usages.begin(), usages.end(),
                                [&su](const descriptor_usage& usage) {
                                  return usage.binding == su.binding &&
                                         usage.set == su.set;
                                });
          if (f != usages.end()) {
            f->count = f->count > su.count ? f->count : su.count;
            continue;
          }
          usages.push_back(su);
        }
      }
      gp->usages = std::move(usages);
    }
    return res;
  }

  VkResult vkCreateDescriptorSetLayout(
      VkDevice device,
      const VkDescriptorSetLayoutCreateInfo* pCreateInfo,
      const VkAllocationCallbacks* pAllocator,
      VkDescriptorSetLayout* pSetLayout) override {
    auto res = super::vkCreateDescriptorSetLayout(device, pCreateInfo,
                                                  pAllocator, pSetLayout);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_layout = this->updater_.cast_from_vk(pSetLayout[0]);
    new_layout->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateImageView(VkDevice device,
                             const VkImageViewCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkImageView* pView) override {
    auto res = super::vkCreateImageView(device, pCreateInfo, pAllocator, pView);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_view = this->updater_.cast_from_vk(pView[0]);
    new_view->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkCreateBufferView(VkDevice device,
                              const VkBufferViewCreateInfo* pCreateInfo,
                              const VkAllocationCallbacks* pAllocator,
                              VkBufferView* pView) override {
    auto res =
        super::vkCreateBufferView(device, pCreateInfo, pAllocator, pView);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto new_view = this->updater_.cast_from_vk(pView[0]);
    new_view->set_create_info(pCreateInfo);
    return res;
  }

  VkResult vkAllocateDescriptorSets(
      VkDevice device,
      const VkDescriptorSetAllocateInfo* pAllocateInfo,
      VkDescriptorSet* pDescriptorSets) override {
    auto res =
        super::vkAllocateDescriptorSets(device, pAllocateInfo, pDescriptorSets);
    if (res != VK_SUCCESS) {
      return res;
    }

    for (size_t i = 0; i < pAllocateInfo->descriptorSetCount; ++i) {
      auto set = this->updater_.cast_from_vk(pDescriptorSets[i]);
      auto layout = this->updater_.cast_from_vk(pAllocateInfo->pSetLayouts[i]);
      set->set_layout(layout);
    }

    return res;
  }

  void vkUpdateDescriptorSets(
      VkDevice device,
      uint32_t descriptorWriteCount,
      const VkWriteDescriptorSet* pDescriptorWrites,
      uint32_t descriptorCopyCount,
      const VkCopyDescriptorSet* pDescriptorCopies) override {
    for (uint32_t i = 0; i < descriptorWriteCount; ++i) {
      auto& dw = pDescriptorWrites[i];
      auto set = this->updater_.cast_from_vk(dw.dstSet);
      auto it = set->bindings.lower_bound(dw.dstBinding);
      auto elem = dw.dstArrayElement;
      for (size_t j = 0; j < dw.descriptorCount; ++j) {
        while (elem >= it->second.descriptors.size()) {
          ++it;
          elem = 0;
        }
        switch (dw.descriptorType) {
          case VK_DESCRIPTOR_TYPE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
          case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
          case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT: {
            it->second.descriptors[elem++].image_info = dw.pImageInfo[j];
            break;
          }
          case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
            it->second.descriptors[elem++].buffer_view_info =
                dw.pTexelBufferView[j];
            break;
          }
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC: {
            it->second.descriptors[elem++].buffer_info = dw.pBufferInfo[j];
            break;
          }
          default:
            GAPID2_ERROR("Unknown descriptor type");
        }
      }
    }
    super::vkUpdateDescriptorSets(device, descriptorWriteCount,
                                  pDescriptorWrites, descriptorCopyCount,
                                  pDescriptorCopies);
  }

  VkResult vkBeginCommandBuffer(
      VkCommandBuffer commandBuffer,
      const VkCommandBufferBeginInfo* pBeginInfo) override {
    auto res = super::vkBeginCommandBuffer(commandBuffer, pBeginInfo);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.clear();
    cb->_post_run_functions.clear();
    return res;
  }

  VkResult vkQueueSubmit(VkQueue queue,
                         uint32_t submitCount,
                         const VkSubmitInfo* pSubmits,
                         VkFence fence) override {
    for (size_t i = 0; i < submitCount; ++i) {
      for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
        graphics_state.m_bound_descriptors.clear();
        compute_state.m_bound_descriptors.clear();
        auto cb = this->updater_.cast_from_vk(pSubmits[i].pCommandBuffers[j]);
        for (auto& pf : cb->_pre_run_functions) {
          pf();
        }
      }
    }

    auto res = super::vkQueueSubmit(queue, submitCount, pSubmits, fence);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (size_t i = 0; i < submitCount; ++i) {
      for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
        auto cb = this->updater_.cast_from_vk(pSubmits[i].pCommandBuffers[j]);
        for (auto& pf : cb->_post_run_functions) {
          pf();
        }
      }
    }
    if (fence) {
      m_pending_write_fences[fence] = std::move(m_write_bound_device_memories);
    }
    return res;
  }

  void vkGetImageMemoryRequirements2(
      VkDevice device,
      const VkImageMemoryRequirementsInfo2* pInfo,
      VkMemoryRequirements2* pMemoryRequirements) override {
    super::vkGetImageMemoryRequirements2(device, pInfo, pMemoryRequirements);
    this->updater_.cast_from_vk(pInfo->image)->required_size =
        pMemoryRequirements->memoryRequirements.size;
  }
  void vkGetBufferMemoryRequirements(
      VkDevice device,
      VkBuffer buffer,
      VkMemoryRequirements* pMemoryRequirements) override {
    super::vkGetBufferMemoryRequirements(device, buffer, pMemoryRequirements);
    this->updater_.cast_from_vk(buffer)->required_size =
        pMemoryRequirements->size;
  }
  void vkGetBufferMemoryRequirements2(
      VkDevice device,
      const VkBufferMemoryRequirementsInfo2* pInfo,
      VkMemoryRequirements2* pMemoryRequirements) override {
    super::vkGetBufferMemoryRequirements2(device, pInfo, pMemoryRequirements);
    this->updater_.cast_from_vk(pInfo->buffer)->required_size =
        pMemoryRequirements->memoryRequirements.size;
  }

  void vkGetImageMemoryRequirements(
      VkDevice device,
      VkImage image,
      VkMemoryRequirements* pMemoryRequirements) override {
    super::vkGetImageMemoryRequirements(device, image, pMemoryRequirements);
    this->updater_.cast_from_vk(image)->required_size =
        pMemoryRequirements->size;
  }

  VkResult vkBindImageMemory(VkDevice device,
                             VkImage image,
                             VkDeviceMemory memory,
                             VkDeviceSize memoryOffset) override {
    auto res = super::vkBindImageMemory(device, image, memory, memoryOffset);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto img = this->updater_.cast_from_vk(image);
    img->bindings.clear();
    img->bindings.push_back(
        memory_binding{memory, memoryOffset, img->required_size});
    return res;
  }

  VkResult vkBindBufferMemory(VkDevice device,
                              VkBuffer buffer,
                              VkDeviceMemory memory,
                              VkDeviceSize memoryOffset) override {
    auto res = super::vkBindBufferMemory(device, buffer, memory, memoryOffset);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto buff = this->updater_.cast_from_vk(buffer);
    buff->bindings.clear();
    buff->bindings.push_back(
        memory_binding{memory, memoryOffset, buff->required_size});
    return res;
  }
  VkResult vkBindBufferMemory2(
      VkDevice device,
      uint32_t bindInfoCount,
      const VkBindBufferMemoryInfo* pBindInfos) override {
    auto res = super::vkBindBufferMemory2(device, bindInfoCount, pBindInfos);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (size_t i = 0; i < bindInfoCount; ++i) {
      auto bi = pBindInfos[i];
      auto buff = this->updater_.cast_from_vk(bi.buffer);
      buff->bindings.clear();
      buff->bindings.push_back(
          memory_binding{bi.memory, bi.memoryOffset, buff->required_size});
    }
    return res;
  }
  VkResult vkBindImageMemory2(
      VkDevice device,
      uint32_t bindInfoCount,
      const VkBindImageMemoryInfo* pBindInfos) override {
    auto res = super::vkBindImageMemory2(device, bindInfoCount, pBindInfos);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (size_t i = 0; i < bindInfoCount; ++i) {
      auto mi = pBindInfos[i];
      auto buff = this->updater_.cast_from_vk(mi.image);
      buff->bindings.clear();
      buff->bindings.push_back(
          memory_binding{mi.memory, mi.memoryOffset, buff->required_size});
    }
    return res;
  }
  void vkCmdBindDescriptorSets(VkCommandBuffer commandBuffer,
                               VkPipelineBindPoint pipelineBindPoint,
                               VkPipelineLayout layout,
                               uint32_t firstSet,
                               uint32_t descriptorSetCount,
                               const VkDescriptorSet* pDescriptorSets,
                               uint32_t dynamicOffsetCount,
                               const uint32_t* pDynamicOffsets) override {
    super::vkCmdBindDescriptorSets(
        commandBuffer, pipelineBindPoint, layout, firstSet, descriptorSetCount,
        pDescriptorSets, dynamicOffsetCount, pDynamicOffsets);
    auto ds = std::vector<VkDescriptorSet>(descriptorSetCount);
    for (size_t i = 0; i < descriptorSetCount; ++i) {
      ds[i] = pDescriptorSets[i];
    }

    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this, ids = std::move(ds), pipelineBindPoint, firstSet]() {
          std::unordered_map<uint32_t, VkDescriptorSet>* sets;
          switch (pipelineBindPoint) {
            case VK_PIPELINE_BIND_POINT_GRAPHICS:
              sets = &graphics_state.m_bound_descriptors;
              break;
            case VK_PIPELINE_BIND_POINT_COMPUTE:
              sets = &compute_state.m_bound_descriptors;
              break;
            default:
              GAPID2_ERROR("Unknown bind point");
          }
          auto dsc = ids.size();
          for (uint32_t i = firstSet, j = 0; i < firstSet + dsc; ++i, ++j) {
            (*sets)[i] = ids[j];
          }
        });
  }

  void vkCmdBindPipeline(VkCommandBuffer commandBuffer,
                         VkPipelineBindPoint pipelineBindPoint,
                         VkPipeline pipeline) override {
    super::vkCmdBindPipeline(commandBuffer, pipelineBindPoint, pipeline);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back([this, pipelineBindPoint, pipeline]() {
      switch (pipelineBindPoint) {
        case VK_PIPELINE_BIND_POINT_GRAPHICS:
          graphics_state.current_pipeline = pipeline;
          break;
        case VK_PIPELINE_BIND_POINT_COMPUTE:
          compute_state.current_pipeline = pipeline;
          break;
        default:
          GAPID2_ERROR("Unknown bind point");
      }
    });
  }

  void vkCmdDraw(VkCommandBuffer commandBuffer,
                 uint32_t vertexCount,
                 uint32_t instanceCount,
                 uint32_t firstVertex,
                 uint32_t firstInstance) override {
    super::vkCmdDraw(commandBuffer, vertexCount, instanceCount, firstVertex,
                     firstInstance);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_GRAPHICS); });
  }

  void vkCmdDrawIndexed(VkCommandBuffer commandBuffer,
                        uint32_t indexCount,
                        uint32_t instanceCount,
                        uint32_t firstIndex,
                        int32_t vertexOffset,
                        uint32_t firstInstance) override {
    super::vkCmdDrawIndexed(commandBuffer, indexCount, instanceCount,
                            firstIndex, vertexOffset, firstInstance);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_GRAPHICS); });
  }
  void vkCmdDrawIndirect(VkCommandBuffer commandBuffer,
                         VkBuffer buffer,
                         VkDeviceSize offset,
                         uint32_t drawCount,
                         uint32_t stride) override {
    super::vkCmdDrawIndirect(commandBuffer, buffer, offset, drawCount, stride);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_GRAPHICS); });
  }
  void vkCmdDrawIndexedIndirect(VkCommandBuffer commandBuffer,
                                VkBuffer buffer,
                                VkDeviceSize offset,
                                uint32_t drawCount,
                                uint32_t stride) override {
    super::vkCmdDrawIndexedIndirect(commandBuffer, buffer, offset, drawCount,
                                    stride);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_GRAPHICS); });
  }
  void vkCmdDrawIndirectCount(VkCommandBuffer commandBuffer,
                              VkBuffer buffer,
                              VkDeviceSize offset,
                              VkBuffer countBuffer,
                              VkDeviceSize countBufferOffset,
                              uint32_t maxDrawCount,
                              uint32_t stride) override {
    super::vkCmdDrawIndirectCount(commandBuffer, buffer, offset, countBuffer,
                                  countBufferOffset, maxDrawCount, stride);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_GRAPHICS); });
  }
  void vkCmdDrawIndexedIndirectCount(VkCommandBuffer commandBuffer,
                                     VkBuffer buffer,
                                     VkDeviceSize offset,
                                     VkBuffer countBuffer,
                                     VkDeviceSize countBufferOffset,
                                     uint32_t maxDrawCount,
                                     uint32_t stride) override {
    super::vkCmdDrawIndexedIndirectCount(commandBuffer, buffer, offset,
                                         countBuffer, countBufferOffset,
                                         maxDrawCount, stride);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_GRAPHICS); });
  }

  void vkCmdDispatch(VkCommandBuffer commandBuffer,
                     uint32_t groupCountX,
                     uint32_t groupCountY,
                     uint32_t groupCountZ) override {
    super::vkCmdDispatch(commandBuffer, groupCountX, groupCountY, groupCountZ);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_COMPUTE); });
  }

  void vkCmdDispatchIndirect(VkCommandBuffer commandBuffer,
                             VkBuffer buffer,
                             VkDeviceSize offset) override {
    super::vkCmdDispatchIndirect(commandBuffer, buffer, offset);
    auto cb = this->updater_.cast_from_vk(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this]() { handle_descriptor_sets(VK_PIPELINE_BIND_POINT_COMPUTE); });
  }

  void handle_descriptor_sets(VkPipelineBindPoint bind_point) {
    decltype(this->updater_.cast_from_vk(VkPipeline(0))) pipeline;
    std::unordered_map<uint32_t, VkDescriptorSet>* sets;
    switch (bind_point) {
      case VK_PIPELINE_BIND_POINT_GRAPHICS:
        pipeline = this->updater_.cast_from_vk(graphics_state.current_pipeline);
        sets = &graphics_state.m_bound_descriptors;
        break;
      case VK_PIPELINE_BIND_POINT_COMPUTE:
        pipeline = this->updater_.cast_from_vk(compute_state.current_pipeline);
        sets = &compute_state.m_bound_descriptors;
        break;
      default:
        GAPID2_ERROR("Unknown bind point");
    }
    for (auto& usage : pipeline->usages) {
      auto ds = this->updater_.cast_from_vk((*sets)[usage.set]);
      auto& bt = ds->bindings[usage.binding].descriptors;
      auto tp = ds->bindings[usage.binding].type;
      for (size_t i = 0; i < usage.count; ++i) {
        switch (tp) {
          case VK_DESCRIPTOR_TYPE_SAMPLER:
            break;
          case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
          case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
          case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT: {
            if (bt[i].image_info.imageView) {
              auto img = this->updater_.cast_from_vk(
                  this->updater_.cast_from_vk(bt[i].image_info.imageView)
                      ->create_info->image);
              for (auto& b : img->bindings) {
                auto mem = this->updater_.cast_from_vk(b.memory);
                if (mem->_is_coherent && mem->_mapped_location) {
                  m_read_bound_device_memories.insert(b.memory);
                }
              }
              if (tp == VK_DESCRIPTOR_TYPE_STORAGE_IMAGE) {
                for (auto& b : img->bindings) {
                  m_write_bound_device_memories.insert(b.memory);
                }
              }
            }
          } break;
          case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
            if (bt[i].buffer_view_info) {
              auto buffer = this->updater_.cast_from_vk(
                  this->updater_.cast_from_vk(bt[i].buffer_view_info)
                      ->create_info->buffer);
              for (auto& b : buffer->bindings) {
                auto mem = this->updater_.cast_from_vk(b.memory);
                if (mem->_is_coherent && mem->_mapped_location) {
                  m_read_bound_device_memories.insert(b.memory);
                }
              }

              if (tp == VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER) {
                for (auto& b : buffer->bindings) {
                  auto mem = this->updater_.cast_from_vk(b.memory);
                  m_write_bound_device_memories.insert(b.memory);
                }
              }
            }
          } break;
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC: {
            if (bt[i].buffer_info.buffer) {
              auto buffer =
                  this->updater_.cast_from_vk(bt[i].buffer_info.buffer);
              for (auto& b : buffer->bindings) {
                auto mem = this->updater_.cast_from_vk(b.memory);
                if (mem->_is_coherent && mem->_mapped_location) {
                  m_read_bound_device_memories.insert(b.memory);
                }
              }
              if (tp == VK_DESCRIPTOR_TYPE_STORAGE_BUFFER ||
                  tp == VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC) {
                for (auto& b : buffer->bindings) {
                  auto mem = this->updater_.cast_from_vk(b.memory);
                  m_write_bound_device_memories.insert(b.memory);
                }
              }
            }
          } break;
          default:
            GAPID2_ERROR("Unknown descriptor type");
        }
      }
    }
  }

 protected:
  std::unordered_set<VkDeviceMemory> m_read_bound_device_memories;
  std::unordered_set<VkDeviceMemory> m_write_bound_device_memories;

  std::unordered_map<VkFence, std::unordered_set<VkDeviceMemory>>
      m_pending_write_fences;

  struct {
    std::unordered_map<uint32_t, VkDescriptorSet> m_bound_descriptors;
    VkPipeline current_pipeline;
  } graphics_state;

  struct {
    std::unordered_map<uint32_t, VkDescriptorSet> m_bound_descriptors;
    VkPipeline current_pipeline;
  } compute_state;
};
}  // namespace gapid2