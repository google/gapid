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

#include "descriptor_set.h"

#include "mid_execution_generator.h"
#include "state_block.h"
#include "utils.h"
#include "sampler.h"
#include "image_view.h"
#include "image.h"
#include "buffer.h"
#include "buffer_view.h"

namespace gapid2 {

void mid_execution_generator::capture_descriptor_sets(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecDescriptorSets");
  for (auto& it : state_block->VkDescriptorSets) {
    VkDescriptorSetWrapper* ds = it.second.second;
    VkDescriptorSet descriptor_set = it.first;
    serializer->vkAllocateDescriptorSets(ds->device,
                                         ds->get_allocate_info(), &descriptor_set);
  }
}

void mid_execution_generator::capture_descriptor_set_contents(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecDescriptorSetContents");
  for (auto& it : state_block->VkDescriptorSets) {
    VkDescriptorSetWrapper* ds = it.second.second;
    VkDescriptorSet descriptor_set = it.first;
    std::vector<VkWriteDescriptorSet> writes;
    std::list<VkDescriptorImageInfo> image_infos;
    std::list<VkDescriptorBufferInfo> buffer_infos;
    std::list<VkBufferView> buffer_views;
    
    for (auto& b : ds->bindings) {
      for (uint32_t i = 0; i < b.second.descriptors.size(); ++i) {
        VkWriteDescriptorSet dws;
        dws.sType = VK_STRUCTURE_TYPE_WRITE_DESCRIPTOR_SET;
        dws.pNext = nullptr;
        dws.dstSet = descriptor_set;
        dws.dstBinding = b.first;
        dws.dstArrayElement = i;
        dws.descriptorType = b.second.type;
        dws.descriptorCount = 1;
        switch (dws.descriptorType) {
          case VK_DESCRIPTOR_TYPE_SAMPLER:
            if (state_block->get(b.second.descriptors[i].image_info.sampler) &&
                state_block->get(b.second.descriptors[i].image_info.sampler)->invalidated) {
              continue;
            }
            image_infos.push_back(b.second.descriptors[i].image_info);
            dws.pImageInfo = &image_infos.back();
            break;
          case VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER:
            if (state_block->get(b.second.descriptors[i].image_info.sampler) &&
                state_block->get(b.second.descriptors[i].image_info.sampler)->invalidated) {
              continue;
            }
            if (state_block->get(b.second.descriptors[i].image_info.imageView) &&
                state_block->get(b.second.descriptors[i].image_info.imageView)->invalidated) {
              continue;
            }
            image_infos.push_back(b.second.descriptors[i].image_info);
            dws.pImageInfo = &image_infos.back();
            break;
          case VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE:
          case VK_DESCRIPTOR_TYPE_STORAGE_IMAGE:
          case VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
            if (state_block->get(b.second.descriptors[i].image_info.imageView) &&
                state_block->get(b.second.descriptors[i].image_info.imageView)->invalidated) {
              continue;
            }
            image_infos.push_back(b.second.descriptors[i].image_info);
            dws.pImageInfo = &image_infos.back();
            break;
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER:
          case VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC:
          case VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
            if (state_block->get(b.second.descriptors[i].buffer_info.buffer) &&
                state_block->get(b.second.descriptors[i].buffer_info.buffer)->invalidated) {
              continue;
            }

            buffer_infos.push_back(b.second.descriptors[i].buffer_info);
            dws.pBufferInfo = &buffer_infos.back();
            break;
          case VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER:
          case VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
            if (state_block->get(b.second.descriptors[i].buffer_view_info) &&
                state_block->get(b.second.descriptors[i].buffer_view_info)->invalidated) {
              continue;
            }
            buffer_views.push_back(b.second.descriptors[i].buffer_view_info);
            dws.pTexelBufferView = &buffer_views.back();
            break;
        }
        writes.push_back(dws);
      }
    }
    serializer->vkUpdateDescriptorSets(ds->device, writes.size(),
                                       writes.data(), 0, nullptr);
  }
}

}  // namespace gapid2