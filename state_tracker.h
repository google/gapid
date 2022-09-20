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
#include <externals/SPIRV-Reflect/spirv_reflect.h>
#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <algorithm>
#include <unordered_map>
#include <unordered_set>

#include "creation_data_tracker.h"

namespace gapid2 {
class state_tracker : public creation_data_tracker<> {
 protected:
  using super = creation_data_tracker<>;

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
    auto sm = state_block_->get(pShaderModule[0]);

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
      auto gp = state_block_->get(pPipelines[i]);
      for (size_t j = 0; j < pCreateInfos[i].stageCount; ++j) {
        auto& mod = pCreateInfos[i].pStages[j].module;
        auto stage =
            state_block_->get(pCreateInfos[i].pStages[j].module);
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
        auto pl = state_block_->get(pCreateInfos[i].layout);
        for (uint32_t j = 0; j < pl->create_info->setLayoutCount; ++j) {
          auto dsl =
              state_block_->get(pl->create_info->pSetLayouts[j]);
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
      auto gp = state_block_->get(pPipelines[i]);
      auto& mod = pCreateInfos[i].stage.module;
      auto stage = state_block_->get(pCreateInfos[i].stage.module);
      auto dsd = stage->_usage.find(pCreateInfos[i].stage.pName);
      if (dsd == stage->_usage.end()) {
        // If we could not find usages for this stage, it means
        // that this shader could not be parsed by spirv-reflect.
        // This is a backup slow-path for such shaders. We
        // assume every descriptor accessible from the pipeline layout
        // is used.
        auto pl = state_block_->get(pCreateInfos[i].layout);
        for (uint32_t j = 0; j < pl->create_info->setLayoutCount; ++j) {
          auto dsl =
              state_block_->get(pl->create_info->pSetLayouts[j]);
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
      auto set = state_block_->get(pDescriptorSets[i]);
      auto layout = state_block_->get(pAllocateInfo->pSetLayouts[i]);
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
      auto set = state_block_->get(dw.dstSet);
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
    for (uint32_t i = 0; i < descriptorCopyCount; ++i) {
      auto& dw = pDescriptorCopies[i];
      auto dst = state_block_->get(dw.dstSet);
      auto src = state_block_->get(dw.srcSet);
      auto src_it = src->bindings.lower_bound(dw.srcBinding);
      auto dst_it = dst->bindings.lower_bound(dw.dstBinding);

      auto src_elem = dw.srcArrayElement;
      auto dst_elem = dw.dstArrayElement;
      for (size_t j = 0; j < dw.descriptorCount; ++j) {
        while (src_elem >= src_it->second.descriptors.size()) {
          ++src_it;
          src_elem = 0;
        }
        while (dst_elem >= dst_it->second.descriptors.size()) {
          ++dst_it;
          dst_elem = 0;
        }
        switch (src_it->second.type) {
          case VK_DESCRIPTOR_TYPE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
          case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
          case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT: {
            dst_it->second.descriptors[dst_elem++].image_info = src_it->second.descriptors[src_elem++].image_info;
            break;
          }
          case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
            dst_it->second.descriptors[dst_elem++].buffer_view_info = src_it->second.descriptors[src_elem++].buffer_view_info;
            break;
          }
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC: {
            dst_it->second.descriptors[dst_elem++].buffer_info = src_it->second.descriptors[src_elem++].buffer_info;
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

  void vkUpdateDescriptorSetWithTemplate(VkDevice device, VkDescriptorSet descriptorSet, VkDescriptorUpdateTemplate descriptorUpdateTemplate, const void* pData) override {
    auto set = state_block_->get(descriptorSet);
    auto templ = state_block_->get(descriptorUpdateTemplate);
    auto ci = templ->get_create_info();
    for (uint32_t i = 0; i < ci->descriptorUpdateEntryCount; ++i) {
      auto& entry = ci->pDescriptorUpdateEntries[i];
      auto elem = entry.dstArrayElement;
      auto it = set->bindings.lower_bound(entry.dstBinding);
      for (size_t j = 0; j < entry.descriptorCount; ++j) {
        while (elem >= it->second.descriptors.size()) {
          ++it;
          elem = 0;
        }
        switch (entry.descriptorType) {
          case VK_DESCRIPTOR_TYPE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
          case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
          case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
          case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT: {
            it->second.descriptors[elem++].image_info = *reinterpret_cast<const VkDescriptorImageInfo*>(reinterpret_cast<const uint8_t*>(pData) + entry.offset + j * entry.stride);
            break;
          }
          case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER: {
            it->second.descriptors[elem++].buffer_view_info = *reinterpret_cast<const VkBufferView*>(reinterpret_cast<const uint8_t*>(pData) + entry.offset + j * entry.stride);
            break;
          }
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC: {
            it->second.descriptors[elem++].buffer_info = *reinterpret_cast<const VkDescriptorBufferInfo*>(reinterpret_cast<const uint8_t*>(pData) + entry.offset + j * entry.stride);
            break;
          }
          default:
            GAPID2_ERROR("Unknown descriptor type");
        }
      }
    }
    super::vkUpdateDescriptorSetWithTemplate(device, descriptorSet,
                                             descriptorUpdateTemplate, pData);
  }

  VkResult vkQueueSubmit(VkQueue queue,
                         uint32_t submitCount,
                         const VkSubmitInfo* pSubmits,
                         VkFence fence) override {
#pragma TODO(awoloszyn, "Handle timeline semaphores")

    for (size_t i = 0; i < submitCount; ++i) {
      auto& sub = pSubmits[i];
      for (uint32_t j = 0; j < sub.waitSemaphoreCount; ++j) {
        auto sem = state_block_->get(sub.pWaitSemaphores[j]);
        sem->value = 0;
      }
      for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
        graphics_state.m_bound_descriptors.clear();
        compute_state.m_bound_descriptors.clear();
        auto cb = state_block_->get(pSubmits[i].pCommandBuffers[j]);
        for (auto& pf : cb->_pre_run_functions) {
          pf(queue);
        }
      }
    }

    auto res = super::vkQueueSubmit(queue, submitCount, pSubmits, fence);
    if (res != VK_SUCCESS) {
      return res;
    }
    for (size_t i = 0; i < submitCount; ++i) {
      auto& sub = pSubmits[i];
      for (uint32_t j = 0; j < sub.signalSemaphoreCount; ++j) {
        auto sem = state_block_->get(sub.pSignalSemaphores[j]);
        sem->value = 1;
      }
      for (size_t j = 0; j < pSubmits[i].commandBufferCount; ++j) {
        auto cb = state_block_->get(pSubmits[i].pCommandBuffers[j]);
        for (auto& pf : cb->_post_run_functions) {
          pf(queue);
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
    state_block_->get(pInfo->image)->required_size =
        pMemoryRequirements->memoryRequirements.size;
  }
  void vkGetBufferMemoryRequirements(
      VkDevice device,
      VkBuffer buffer,
      VkMemoryRequirements* pMemoryRequirements) override {
    super::vkGetBufferMemoryRequirements(device, buffer, pMemoryRequirements);
    state_block_->get(buffer)->required_size =
        pMemoryRequirements->size;
  }
  void vkGetBufferMemoryRequirements2(
      VkDevice device,
      const VkBufferMemoryRequirementsInfo2* pInfo,
      VkMemoryRequirements2* pMemoryRequirements) override {
    super::vkGetBufferMemoryRequirements2(device, pInfo, pMemoryRequirements);
    state_block_->get(pInfo->buffer)->required_size =
        pMemoryRequirements->memoryRequirements.size;
  }

  void vkGetImageMemoryRequirements(
      VkDevice device,
      VkImage image,
      VkMemoryRequirements* pMemoryRequirements) override {
    super::vkGetImageMemoryRequirements(device, image, pMemoryRequirements);
    state_block_->get(image)->required_size =
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
    auto img = state_block_->get(image);
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
    auto buff = state_block_->get(buffer);
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
      auto buff = state_block_->get(bi.buffer);
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
      auto buff = state_block_->get(mi.image);
      buff->bindings.clear();
      buff->bindings.push_back(
          memory_binding{mi.memory, mi.memoryOffset, buff->required_size});
    }
    return res;
  }

  VkResult vkCreateBuffer(VkDevice device,
                          const VkBufferCreateInfo* pCreateInfo,
                          const VkAllocationCallbacks* pAllocator,
                          VkBuffer* pBuffer) override {
    VkBufferCreateInfo bci = *pCreateInfo;
    bci.usage |= VK_BUFFER_USAGE_TRANSFER_SRC_BIT;
    return super::vkCreateBuffer(device, &bci, nullptr, pBuffer);
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

    auto cb = state_block_->get(commandBuffer);
    cb->_pre_run_functions.push_back(
        [this, ids = std::move(ds), pipelineBindPoint, firstSet](VkQueue) {
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

  void handle_renderpass_begin(VkCommandBuffer commandBuffer, VkFramebuffer framebuffer) {
    auto cb = state_block_->get(commandBuffer);

    cb->_pre_run_functions.push_back(
        [this, cFb = std::move(framebuffer)](VkQueue queue) {
          auto fb = state_block_->get(cFb);
          auto q = state_block_->get(queue);
          for (uint32_t i = 0; i < fb->get_create_info()->attachmentCount; ++i) {
            auto iview = state_block_->get(fb->get_create_info()->pAttachments[i]);
            auto ivci = iview->get_create_info();
            auto img = state_block_->get(ivci->image);

            img->for_each_subresource_in(ivci->subresourceRange, [cQ = q, cImg = img](uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect) {
              auto subresource_idx = cImg->get_subresource_idx(mip_level, array_layer, aspect);
              cImg->sr_data[subresource_idx].src_queue_idx = cQ->queue_family_index;
              cImg->sr_data[subresource_idx].dst_queue_idx = cQ->queue_family_index;
            });
          }
        });
  }

  void vkCmdBeginRenderPass2(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, const VkSubpassBeginInfo* pSubpassBeginInfo) {
    handle_renderpass_begin(commandBuffer, pRenderPassBegin->framebuffer);
    return super::vkCmdBeginRenderPass2(commandBuffer, pRenderPassBegin, pSubpassBeginInfo);
  }

  void vkCmdBeginRenderPass(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, VkSubpassContents contents) {
    handle_renderpass_begin(commandBuffer, pRenderPassBegin->framebuffer);
    return super::vkCmdBeginRenderPass(commandBuffer, pRenderPassBegin, contents);
  }

  void vkCmdBeginRenderPass2KHR(VkCommandBuffer commandBuffer, const VkRenderPassBeginInfo* pRenderPassBegin, const VkSubpassBeginInfo* pSubpassBeginInfo) {
    handle_renderpass_begin(commandBuffer, pRenderPassBegin->framebuffer);
    return super::vkCmdBeginRenderPass2KHR(commandBuffer, pRenderPassBegin, pSubpassBeginInfo);
  }

  void vkCmdPipelineBarrier(VkCommandBuffer commandBuffer,
                            VkPipelineStageFlags srcStageMask,
                            VkPipelineStageFlags dstStageMask,
                            VkDependencyFlags dependencyFlags,
                            uint32_t memoryBarrierCount,
                            const VkMemoryBarrier* pMemoryBarriers,
                            uint32_t bufferMemoryBarrierCount,
                            const VkBufferMemoryBarrier* pBufferMemoryBarriers,
                            uint32_t imageMemoryBarrierCount,
                            const VkImageMemoryBarrier* pImageMemoryBarriers) override {
    super::vkCmdPipelineBarrier(commandBuffer,
                                srcStageMask,
                                dstStageMask,
                                dependencyFlags,
                                memoryBarrierCount,
                                pMemoryBarriers,
                                bufferMemoryBarrierCount,
                                pBufferMemoryBarriers,
                                imageMemoryBarrierCount,
                                pImageMemoryBarriers);

    struct buffer_barrier {
      uint32_t src_queue_index;
      uint32_t dst_queue_index;
      VkBuffer buffer;
    };

    std::vector<buffer_barrier> buffers;
    for (uint32_t i = 0; i < bufferMemoryBarrierCount; ++i) {
      // If we have not assigned a queue family index OR
      //   if that queue family index is changing, then we should record this.
      if (state_block_->get(pBufferMemoryBarriers[i].buffer)->src_queue ==
              VK_QUEUE_FAMILY_IGNORED ||
          pBufferMemoryBarriers[i].srcQueueFamilyIndex !=
              pBufferMemoryBarriers[i].dstQueueFamilyIndex) {
        buffers.push_back(
            buffer_barrier{
                .src_queue_index = pBufferMemoryBarriers[i].srcQueueFamilyIndex,
                .dst_queue_index = pBufferMemoryBarriers[i].dstQueueFamilyIndex,
                .buffer = pBufferMemoryBarriers[i].buffer});
      }
    }

    struct image_barrier {
      uint32_t src_queue_index;
      uint32_t dst_queue_index;
      VkImage image;
      VkImageLayout layout;
      VkImageSubresourceRange range;
    };
    std::vector<image_barrier> images;
    for (uint32_t i = 0; i < imageMemoryBarrierCount; ++i) {
      // If we have not assigned a queue family index OR
      //   if that queue family index is changing, then we should record this.
      auto img = state_block_->get(pImageMemoryBarriers[i].image);
      bool add = pImageMemoryBarriers[i].srcQueueFamilyIndex != pImageMemoryBarriers[i].dstQueueFamilyIndex ||
                 pImageMemoryBarriers[i].oldLayout != pImageMemoryBarriers[i].newLayout;
      if (!add) {
        img->for_each_subresource_in(pImageMemoryBarriers[i].subresourceRange,
                                     [&cAdd = add, cImg = img](uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect) {
                                       if (cAdd) {
                                         return;
                                       }
                                       auto subresource_idx = cImg->get_subresource_idx(mip_level, array_layer, aspect);
                                       if (cImg->sr_data[subresource_idx].src_queue_idx == VK_QUEUE_FAMILY_IGNORED) {
                                         cAdd = true;
                                       }
                                     });
      }
      if (add) {
        images.push_back(image_barrier{
            .src_queue_index = pImageMemoryBarriers[i].srcQueueFamilyIndex,
            .dst_queue_index = pImageMemoryBarriers[i].dstQueueFamilyIndex,
            .image = pImageMemoryBarriers[i].image,
            .layout = pImageMemoryBarriers[i].newLayout,
            .range = pImageMemoryBarriers[i].subresourceRange});
      }
    }

    auto cb = state_block_->get(commandBuffer);
    if (!images.empty() || !buffers.empty()) {
      cb->_post_run_functions.push_back([this, cB = std::move(buffers), cI = std::move(images)](VkQueue queue) {
        auto q = state_block_->get(queue);
        for (const auto b : cB) {
          auto buff = state_block_->get(b.buffer);
          if (b.src_queue_index == b.dst_queue_index) {
            buff->src_queue = q->queue_family_index;
            buff->dst_queue = q->queue_family_index;
          } else {
            // This is a release operation, so record it as such
            if (b.src_queue_index == q->queue_family_index) {
              buff->src_queue = b.src_queue_index;
              buff->dst_queue = b.dst_queue_index;
            } else {  // This is an acquire operation, so record it as such (meaning we are done and now on the new queue)
              buff->src_queue = b.dst_queue_index;
              buff->dst_queue = b.dst_queue_index;
            }
          }
        }
        for (const auto i : cI) {
          auto img = state_block_->get(i.image);
          img->for_each_subresource_in(i.range, [&cI = i, cImg = img, cQ = q](uint32_t mip_level, uint32_t array_layer, VkImageAspectFlagBits aspect) {
            auto sr = cImg->get_subresource_idx(mip_level, array_layer, aspect);
            auto& srdata = cImg->sr_data[sr];
            srdata.layout = cI.layout;
            if (cI.src_queue_index == cI.dst_queue_index) {
              srdata.src_queue_idx = cQ->queue_family_index;
              srdata.dst_queue_idx = cQ->queue_family_index;
            } else {
              if (cI.src_queue_index == cQ->queue_family_index) {
                srdata.src_queue_idx = cI.src_queue_index;
                srdata.dst_queue_idx = cI.dst_queue_index;
              } else {  // This is an acquire operation, so record it as such (meaning we are done and now on the new queue)
                srdata.src_queue_idx = cI.dst_queue_index;
                srdata.dst_queue_idx = cI.dst_queue_index;
              }
            }
          });
        }
      });
    }
  }

  VkResult vkAcquireNextImageKHR(VkDevice device, VkSwapchainKHR swapchain, uint64_t timeout, VkSemaphore semaphore, VkFence fence, uint32_t* pImageIndex) {
    auto ret = super::vkAcquireNextImageKHR(device, swapchain, timeout, semaphore, fence, pImageIndex);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    if (semaphore) {
      auto sem = state_block_->get(semaphore);
      sem->value = 1;
    }
    return ret;
  }
  VkResult vkAcquireNextImage2KHR(VkDevice device, const VkAcquireNextImageInfoKHR* pAcquireInfo, uint32_t* pImageIndex) {
    auto ret = super::vkAcquireNextImage2KHR(device, pAcquireInfo, pImageIndex);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    if (pAcquireInfo->semaphore) {
      auto sem = state_block_->get(pAcquireInfo->semaphore);
      sem->value = 1;
    }
    return ret;
  }

  VkResult vkQueuePresentKHR(VkQueue queue, const VkPresentInfoKHR* pPresentInfo) {
    for (size_t i = 0; i < pPresentInfo->waitSemaphoreCount; ++i) {
      auto sem = state_block_->get(pPresentInfo->pWaitSemaphores[i]);
      sem->value = 0;
    }
    auto ret = super::vkQueuePresentKHR(queue, pPresentInfo);
    return ret;
  }

  VkResult vkQueueBindSparse(VkQueue queue, uint32_t bindInfoCount, const VkBindSparseInfo* pBindInfo, VkFence fence) {
    for (size_t i = 0; i < bindInfoCount; ++i) {
      auto& binds = pBindInfo[i];
      for (uint32_t j = 0; j < binds.waitSemaphoreCount; ++j) {
        auto sem = state_block_->get(binds.pWaitSemaphores[j]);
        sem->value = 0;
      }
    }

    auto ret = vkQueueBindSparse(queue, bindInfoCount, pBindInfo, fence);
    if (ret != VK_SUCCESS) {
      return ret;
    }
    for (size_t i = 0; i < bindInfoCount; ++i) {
      auto& binds = pBindInfo[i];
      for (uint32_t j = 0; j < binds.signalSemaphoreCount; ++j) {
        auto sem = state_block_->get(binds.pSignalSemaphores[j]);
        sem->value = 1;
      }
    }
    return ret;
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
    auto view = state_block_->get(pView[0]);
    auto buffer = state_block_->get(pCreateInfo->buffer);
    buffer->invalidates(view.get());

    return res;
  }

  VkResult vkCreateImageView(VkDevice device,
                             const VkImageViewCreateInfo* pCreateInfo,
                             const VkAllocationCallbacks* pAllocator,
                             VkImageView* pView) override {
    auto res =
        super::vkCreateImageView(device, pCreateInfo, pAllocator, pView);
    if (res != VK_SUCCESS) {
      return res;
    }
    auto view = state_block_->get(pView[0]);
    auto image = state_block_->get(pCreateInfo->image);
    image->invalidates(view.get());

    return res;
  }

  VkResult vkCreateFramebuffer(VkDevice device, const VkFramebufferCreateInfo* pCreateInfo, const VkAllocationCallbacks* pAllocator, VkFramebuffer* pFramebuffer) {
    auto ret = super::vkCreateFramebuffer(device, pCreateInfo, pAllocator, pFramebuffer);
    for (uint32_t i = 0; i < pCreateInfo->attachmentCount; ++i) {
      state_block_->get(pCreateInfo->pAttachments[i])->invalidates(state_block_->get(*pFramebuffer).get());
    }
    return ret;
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