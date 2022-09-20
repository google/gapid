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

#include "buffer.h"

#include "mid_execution_generator.h"
#include "staging_resource_manager.h"
#include "state_block.h"
#include "utils.h"

namespace gapid2 {

void mid_execution_generator::capture_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecBufferCreation");
  for (auto& it : state_block->VkBuffers) {
    auto buff = it.second.second;
    VkBuffer buffer = it.first;
    serializer->vkCreateBuffer(buff->device,
                               buff->get_create_info(), nullptr, &buffer);
  }
}

void mid_execution_generator::capture_bind_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const {
  serializer->insert_annotation("MecBufferBinds");
  for (auto& it : state_block->VkBuffers) {
    auto buff = it.second.second;
    GAPID2_ASSERT(0 == buff->get_create_info()->flags & VK_BUFFER_CREATE_SPARSE_BINDING_BIT, "We do not support sparse images yet");
    GAPID2_ASSERT(buff->bindings.size() <= 1, "Invalid number of binds");

#pragma TODO(awoloszyn, Handle the different special bind flags)
    if (buff->bindings.empty()) {
      continue;
    }
    serializer->vkBindBufferMemory(buff->device, it.first, buff->bindings[0].memory, buff->bindings[0].offset);
  }
}

void capture_host_mapped_buffer_data(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller, const VkDeviceWrapper* device,
                                     const memory_binding& binding, const VkPhysicalDeviceMemoryProperties& mem_props) {
  auto mem = state_block->get(binding.memory);
  // Map/unmap/remap the memory correctly.
  auto old_map = mem->_mapped_location ? mem->_mapped_offset : 0;
  auto old_size = mem->_mapped_location ? mem->_mapped_size : 0;

  if (mem->_mapped_location) {
    bypass_caller->vkUnmapMemory(device->_handle, mem->_handle);
  }
  void* p;
  bypass_caller->vkMapMemory(device->_handle, mem->_handle, binding.offset, binding.size, 0, &p);

  serializer->vkMapMemory(device->_handle, mem->_handle, binding.offset, binding.size, 0, &p);
  {
    auto enc = serializer->get_encoder(0);
    enc->encode<uint64_t>(0);
    enc->encode<uint64_t>(serializer->get_flags());
    enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(mem->_handle));
    enc->encode<uint64_t>(0);  // offset
    enc->encode<uint64_t>(binding.size);
    enc->encode_primitive_array<char>(reinterpret_cast<const char*>(p), binding.size);
  }

  if (!(mem_props.memoryTypes[mem->allocate_info->memoryTypeIndex].propertyFlags & VK_MEMORY_PROPERTY_HOST_COHERENT_BIT)) {
    VkMappedMemoryRange rng{
        .sType = VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE,
        .pNext = nullptr,
        .memory = mem->_handle,
        .offset = binding.offset,
        .size = binding.size};
    serializer->vkFlushMappedMemoryRanges(device->_handle, 1, &rng);
  }
  serializer->vkUnmapMemory(device->_handle, mem->_handle);
  bypass_caller->vkUnmapMemory(device->_handle, mem->_handle);
  if (mem->_mapped_location) {
#pragma TODO(awoloszyn, "This might cause memory to be mapped to a different location, however as long as we use external_memory_host this doesnt matter")
#pragma TODO(awoloszyn, "If we switch to something else we will have to be more careful")
    void* v;
    bypass_caller->vkMapMemory(device->_handle, mem->_handle, old_map, old_size, 0, &v);
  }
}

void mid_execution_generator::capture_buffer_data(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller, shader_manager* shader_manager) const {
  serializer->insert_annotation("MecBufferData");
  for (auto& dev : state_block->VkDevices) {
    auto device = dev.second.second;
    staging_resource_manager staging(bypass_caller, serializer, state_block->get(device->get_physical_device()), device.get(), max_copy_overhead_bytes_, shader_manager);
    auto phys_dev = state_block->get(device->get_physical_device());
    VkPhysicalDeviceMemoryProperties mem_props;
    bypass_caller->vkGetPhysicalDeviceMemoryProperties(phys_dev->_handle, &mem_props);
    for (auto& it : state_block->VkBuffers) {
      if (it.second.second->device != dev.first) {
        continue;
      }
      auto buff = it.second.second;
      GAPID2_ASSERT(0 == buff->get_create_info()->flags & VK_BUFFER_CREATE_SPARSE_BINDING_BIT, "We do not support sparse images yet");
      GAPID2_ASSERT(buff->bindings.size() <= 1, "Invalid number of binds");

#pragma TODO(awoloszyn, Handle the different special bind flags)
      if (buff->bindings.empty()) {
        continue;
      }

      const auto& binding = buff->bindings[0];

      auto mem = state_block->get(binding.memory);
      // If the memory is host-visible, we can just map it here.
      if (false && mem_props.memoryTypes[mem->allocate_info->memoryTypeIndex].propertyFlags &
                       VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
        capture_host_mapped_buffer_data(state_block, serializer, bypass_caller, device.get(), binding, mem_props);
        continue;
      }

      // If this is not host-visible AND this has not been used on a queue, then
      // this cannot have any useful data in it :)
#pragma TODO(awoloszyyn, "Handle aliased memory here")
      //if (buff->src_queue == VK_QUEUE_FAMILY_IGNORED) {
      //  break;
      //}

      for (VkDeviceSize offset = 0; offset < binding.size;) {
        staging_resource_manager::staging_resources* res = new staging_resource_manager::staging_resources;
        auto q = get_queue_for_family(state_block, buff->device, buff->src_queue);

        *res = staging.get_staging_buffer_for_queue(
            state_block->get(q),
            binding.size, [cRes = res, cOffs = offset, cS = serializer, cM = mem, cB = buff](const char* data, VkDeviceSize size, std::vector<std::function<void()>>*) {
              auto enc = cS->get_encoder(0);
              enc->encode<uint64_t>(0);
              enc->encode<uint64_t>(cS->get_flags());
              enc->encode<uint64_t>(reinterpret_cast<uintptr_t>(cRes->memory));
              enc->encode<uint64_t>(cRes->buffer_offset);  // offset
              enc->encode<uint64_t>(size);
              enc->encode_primitive_array<char>(reinterpret_cast<const char*>(data), size);

              VkBufferCopy cp{
                  .srcOffset = cRes->buffer_offset,
                  .dstOffset = cOffs,
                  .size = cRes->returned_size};

              VkBufferMemoryBarrier barrier{
                  .sType = VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
                  .pNext = nullptr,
                  .srcAccessMask = (VK_ACCESS_MEMORY_WRITE_BIT - 1) | VK_ACCESS_MEMORY_WRITE_BIT,
                  .dstAccessMask = (VK_ACCESS_MEMORY_WRITE_BIT - 1) | VK_ACCESS_MEMORY_WRITE_BIT,
                  .srcQueueFamilyIndex = 0xFFFFFFFF,
                  .dstQueueFamilyIndex = 0xFFFFFFFF,
                  .buffer = cB->_handle,
                  .offset = cOffs,
                  .size = cRes->returned_size};

              cS->vkCmdPipelineBarrier(
                  cRes->cb,
                  VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                  VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                  0, 0, nullptr, 1, &barrier, 0, nullptr);

              cS->vkCmdCopyBuffer(cRes->cb, cRes->buffer, cB->_handle, 1, &cp);

              barrier.srcQueueFamilyIndex = cB->src_queue;
              barrier.dstQueueFamilyIndex = cB->dst_queue;
              // Get it ready and move it onto the right queue
              cS->vkCmdPipelineBarrier(
                  cRes->cb,
                  VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                  VK_PIPELINE_STAGE_ALL_COMMANDS_BIT,
                  0, 0, nullptr, 1, &barrier, 0, nullptr);
              delete cRes;
            });

        VkBufferCopy cp{
            .srcOffset = offset,
            .dstOffset = res->buffer_offset,
            .size = res->returned_size};
        bypass_caller->vkCmdCopyBuffer(res->cb, buff->_handle, res->buffer, 1, &cp);
        VkBufferMemoryBarrier barrier{
            .sType = VK_STRUCTURE_TYPE_BUFFER_MEMORY_BARRIER,
            .pNext = nullptr,
            .srcAccessMask = VK_ACCESS_TRANSFER_WRITE_BIT,
            .dstAccessMask = VK_ACCESS_HOST_READ_BIT,
            .srcQueueFamilyIndex = 0xFFFFFFFF,
            .dstQueueFamilyIndex = 0xFFFFFFFF,
            .buffer = res->buffer,
            .offset = offset,
            .size = res->returned_size};
        bypass_caller->vkCmdPipelineBarrier(
            res->cb,
            VK_PIPELINE_STAGE_TRANSFER_BIT,
            VK_PIPELINE_STAGE_HOST_BIT,
            0, 0, nullptr, 1, &barrier, 0, nullptr);
        offset = offset + res->returned_size;
      }
    }
  }
}

}  // namespace gapid2