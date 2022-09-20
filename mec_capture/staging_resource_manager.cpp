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

#include "staging_resource_manager.h"

#include "device.h"
#include "queue.h"
#include "shader_manager.h"
#include "state_block.h"

namespace gapid2 {

namespace {
const uint32_t kInvalidMemoryTypeIndex = 0xFFFFFFFF;
void annotate(gapid2::command_serializer* serializer, std::string str) {
  if (!str.empty()) {
    auto enc = serializer->get_encoder(0);
    enc->encode<uint64_t>(1);
    enc->encode<uint64_t>(1);
    enc->encode<uint64_t>(str.size() + 1);
    enc->encode_primitive_array(str.c_str(), str.size() + 1);
  }
}
}

staging_resource_manager::staging_resource_manager(transform_base* callee,
                                                   command_serializer* serializer,
                                                   const VkPhysicalDeviceWrapper* physical_device,
                                                   const VkDeviceWrapper* device,
                                                   VkDeviceSize maximum_size,
                                                   shader_manager* s_manager) : callee(callee),
                                                                                serializer(serializer),
                                                                                device(device),
                                                                                maximum_size(maximum_size),
                                                                                s_manager(s_manager) {
  VkPhysicalDeviceMemoryProperties properties;
  callee->vkGetPhysicalDeviceMemoryProperties(physical_device->_handle, &properties);

  VkBufferCreateInfo create_info = {
      .sType = VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .size = maximum_size,
      .usage = VK_BUFFER_USAGE_TRANSFER_DST_BIT | VK_BUFFER_USAGE_TRANSFER_SRC_BIT,
      .sharingMode = VK_SHARING_MODE_EXCLUSIVE,
      .queueFamilyIndexCount = 0,
      .pQueueFamilyIndices = 0};

  GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateBuffer(device->_handle, &create_info, nullptr, &dest_buffer),
                "Could not create staging buffer for resource");

  serializer->vkCreateBuffer(device->_handle, &create_info, nullptr, &dest_buffer);

  VkMemoryRequirements requirements;
  callee->vkGetBufferMemoryRequirements(device->_handle, dest_buffer, &requirements);
  serializer->vkGetBufferMemoryRequirements(device->_handle, dest_buffer, &requirements);

  uint32_t idx = get_memory_type_index_for_staging_resource(properties, requirements.memoryTypeBits);

  VkMemoryAllocateInfo allocate_info = {
      .sType = VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
      .pNext = nullptr,
      .allocationSize = requirements.size,
      .memoryTypeIndex = idx};

  GAPID2_ASSERT(VK_SUCCESS == callee->vkAllocateMemory(device->_handle, &allocate_info, nullptr, &device_memory),
                "Could not allocate staging memory");
  GAPID2_ASSERT(VK_SUCCESS == callee->vkBindBufferMemory(device->_handle, dest_buffer, device_memory, 0),
                "Could not bind staging buffer");
  GAPID2_ASSERT(VK_SUCCESS == callee->vkMapMemory(device->_handle, device_memory, 0, VK_WHOLE_SIZE, 0, reinterpret_cast<void**>(&device_memory_ptr)),
                "Could not map staging memory");

  GAPID2_ASSERT(VK_SUCCESS == serializer->vkAllocateMemory(device->_handle, &allocate_info, nullptr, &device_memory),
                "Could not allocate staging memory");
  GAPID2_ASSERT(VK_SUCCESS == serializer->vkBindBufferMemory(device->_handle, dest_buffer, device_memory, 0),
                "Could not bind staging buffer");
  GAPID2_ASSERT(VK_SUCCESS == serializer->vkMapMemory(device->_handle, device_memory, 0, VK_WHOLE_SIZE, 0, reinterpret_cast<void**>(&device_memory_ptr)),
                "Could not map staging memory");
}

uint32_t staging_resource_manager::get_memory_type_index_for_staging_resource(
    const VkPhysicalDeviceMemoryProperties& phy_dev_prop,
    uint32_t requirement_type_bits) {
  uint32_t index = 0;
  uint32_t backup_index = kInvalidMemoryTypeIndex;
  while (requirement_type_bits && index < phy_dev_prop.memoryTypeCount) {
    if (requirement_type_bits & 0x1) {
      VkMemoryPropertyFlags prop_flags =
          phy_dev_prop.memoryTypes[index].propertyFlags;
      if (prop_flags &
          VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_VISIBLE_BIT) {
        if (backup_index == kInvalidMemoryTypeIndex) {
          backup_index = index;
        }
        if ((prop_flags &
             VkMemoryPropertyFlagBits::VK_MEMORY_PROPERTY_HOST_COHERENT_BIT) ==
            0) {
          break;
        }
      }
    }
    requirement_type_bits >>= 1;
    index++;
  }
  if (requirement_type_bits != 0) {
    return index;
  }
  GAPID2_ASSERT(backup_index != kInvalidMemoryTypeIndex,
                "Unknown type index for staging resource");
  return backup_index;
}

staging_resource_manager::~staging_resource_manager() {
  flush();
  for (auto& it : queue_data) {
    callee->vkFreeCommandBuffers(device->_handle, it.second.command_pool, 1, &it.second.command_buffer);
    callee->vkDestroyCommandPool(device->_handle, it.second.command_pool, nullptr);

    serializer->vkFreeCommandBuffers(device->_handle, it.second.command_pool, 1, &it.second.command_buffer);
    serializer->vkDestroyCommandPool(device->_handle, it.second.command_pool, nullptr);
  }
  callee->vkDeviceWaitIdle(device->_handle);
  callee->vkDestroyBuffer(device->_handle, dest_buffer, nullptr);
  callee->vkFreeMemory(device->_handle, device_memory, nullptr);

  serializer->vkDeviceWaitIdle(device->_handle);
  serializer->vkDestroyBuffer(device->_handle, dest_buffer, nullptr);
  serializer->vkFreeMemory(device->_handle, device_memory, nullptr);
}

void staging_resource_manager::flush() {
  for (auto& it : queue_data) {
    callee->vkEndCommandBuffer(it.second.command_buffer);
    VkSubmitInfo inf{
        .sType = VK_STRUCTURE_TYPE_SUBMIT_INFO,
        .pNext = nullptr,
        .waitSemaphoreCount = 0,
        .pWaitSemaphores = nullptr,
        .pWaitDstStageMask = nullptr,
        .commandBufferCount = 1,
        .pCommandBuffers = &it.second.command_buffer,
        .signalSemaphoreCount = 0,
        .pSignalSemaphores = nullptr};
    GAPID2_ASSERT(VK_SUCCESS == callee->vkQueueSubmit(it.first, 1, &inf, VK_NULL_HANDLE), "Could not submit staging commands");
    GAPID2_ASSERT(VK_SUCCESS == callee->vkQueueWaitIdle(it.first), "Error in submitted commands, crash on the GPU");
    GAPID2_ASSERT(VK_SUCCESS == callee->vkResetCommandPool(device->_handle, it.second.command_pool, VK_COMMAND_POOL_RESET_RELEASE_RESOURCES_BIT),
                  "Could not reset staging command pool");

    VkCommandBufferBeginInfo cbbi{
        .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
        .pNext = nullptr,
        .flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
        .pInheritanceInfo = nullptr};
    GAPID2_ASSERT(VK_SUCCESS == callee->vkBeginCommandBuffer(it.second.command_buffer, &cbbi),
                  "Could not begin command buffer");
  }
  std::vector<std::function<void()>> cleanups;
  for (auto& c : run_data) {
    c.call(c.offs, c.size, &cleanups);
  }

  for (auto& it : queue_data) {
    serializer->vkEndCommandBuffer(it.second.command_buffer);
    VkSubmitInfo inf{
        .sType = VK_STRUCTURE_TYPE_SUBMIT_INFO,
        .pNext = nullptr,
        .waitSemaphoreCount = 0,
        .pWaitSemaphores = nullptr,
        .pWaitDstStageMask = nullptr,
        .commandBufferCount = 1,
        .pCommandBuffers = &it.second.command_buffer,
        .signalSemaphoreCount = 0,
        .pSignalSemaphores = nullptr};

    serializer->vkQueueSubmit(it.first, 1, &inf, VK_NULL_HANDLE);
    serializer->vkQueueWaitIdle(it.first);
    serializer->vkResetCommandPool(device->_handle, it.second.command_pool, VK_COMMAND_POOL_RESET_RELEASE_RESOURCES_BIT);

    VkCommandBufferBeginInfo cbbi{
        .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
        .pNext = nullptr,
        .flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
        .pInheritanceInfo = nullptr};
    serializer->vkBeginCommandBuffer(it.second.command_buffer, &cbbi);
  }
  for (auto& c : cleanups) {
    c();
  }

  run_data.clear();
  offset = 0;
}

VkCommandBuffer staging_resource_manager::get_command_buffer_for_queue(const VkQueueWrapper* queue) {
  auto it = queue_data.find(queue->_handle);
  if (it == queue_data.end()) {
    VkCommandPool p;
    VkCommandPoolCreateInfo command_pool_create{
        .sType = VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
        .pNext = nullptr,
        .flags = VK_COMMAND_POOL_CREATE_TRANSIENT_BIT,
        .queueFamilyIndex = queue->queue_family_index};
    GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateCommandPool(device->_handle, &command_pool_create, nullptr, &p),
                  "Could not create staging command buffer");
    GAPID2_ASSERT(VK_SUCCESS == serializer->vkCreateCommandPool(device->_handle, &command_pool_create, nullptr, &p),
                  "Could not create staging command buffer");
    VkCommandBuffer cb = VK_NULL_HANDLE;
    VkCommandBufferAllocateInfo allocate_info{
        .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
        .pNext = nullptr,
        .commandPool = p,
        .level = VK_COMMAND_BUFFER_LEVEL_PRIMARY,
        .commandBufferCount = 1};

    GAPID2_ASSERT(VK_SUCCESS == callee->vkAllocateCommandBuffers(device->_handle, &allocate_info, &cb),
                  "Could not allocate staging CB");
    GAPID2_ASSERT(VK_SUCCESS == serializer->vkAllocateCommandBuffers(device->_handle, &allocate_info, &cb),
                  "Could not allocate staging CB");

    VkCommandBufferBeginInfo cbbi{
        .sType = VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
        .pNext = nullptr,
        .flags = VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT,
        .pInheritanceInfo = nullptr};
    reinterpret_cast<void**>(cb)[0] = reinterpret_cast<void**>(device->_handle)[0];
    GAPID2_ASSERT(VK_SUCCESS == callee->vkBeginCommandBuffer(cb, &cbbi),
                  "Could not begin command buffer");
    GAPID2_ASSERT(VK_SUCCESS == serializer->vkBeginCommandBuffer(cb, &cbbi),
                  "Could not begin command buffer");

    std::tie(it, std::ignore) = queue_data.insert(std::make_pair(queue->_handle, queue_specific_data{.device = queue->device, .command_pool = p, .command_buffer = cb}));
  }
  return it->second.command_buffer;
}

staging_resource_manager::staging_resources staging_resource_manager::get_staging_buffer_for_queue(
    const VkQueueWrapper* queue,
    VkDeviceSize bufferSize,
    std::function<void(const char* data, VkDeviceSize size, std::vector<std::function<void()>>* cleanups)> fn) {
  VkDeviceSize available = maximum_size - offset;
  // If we dont have enough space, try and flush first,
  // that way we will get the next resource in as few
  // flushes as possible.
  if (available < bufferSize) {
    if (offset == 0) {
      available = maximum_size;
    } else {
      flush();
      if (bufferSize > maximum_size) {
        available = maximum_size;
      } else {
        available = bufferSize;
      }
    }
  }

  const VkDeviceSize used = bufferSize < available ? bufferSize : available;
  VkDeviceSize offs = offset;

  const VkDeviceSize kNonCoherentAtomSize = 256;
  offset = (offset + used + (kNonCoherentAtomSize - 1)) & ~(kNonCoherentAtomSize - 1);
  if (offset >= maximum_size) {
    flush();
  }

  run_data.push_back(data_offset{
      .call = fn,
      .offs = device_memory_ptr + offs,
      .size = used,
  });

  return staging_resource_manager::staging_resources{
      .cb = get_command_buffer_for_queue(queue),
      .buffer_offset = offs,
      .buffer = dest_buffer,
      .returned_size = used,
      .memory = device_memory};
}

VkQueue get_queue_for_family(const state_block* sb, VkDevice device, uint32_t queue_family) {
  for (auto& q : sb->VkQueues) {
    if (q.second.second->device == device &&
        (q.second.second->queue_family_index == queue_family ||
         queue_family == VK_QUEUE_FAMILY_IGNORED)) {
      return q.first;
    }
  }
  return VK_NULL_HANDLE;
}

std::pair<VkDescriptorSet, VkDescriptorPool> staging_resource_manager::get_input_attachment_descriptor_set_for_device(VkDevice device) {
  if (device_data.find(device) == device_data.end()) {
    device_data[device] = device_specific_data{};
  }

  auto& dd = device_data[device];

  if (VK_NULL_HANDLE == dd.descriptor_set_layout_for_prime_by_render) {
    VkDescriptorSetLayoutBinding binding{
        .binding = k_render_input_attachment_index,
        .descriptorType = VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
        .descriptorCount = 1,
        .stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT,
        .pImmutableSamplers = nullptr};

    VkDescriptorSetLayoutCreateInfo dsci{
        .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .bindingCount = 1,
        .pBindings = &binding,
    };

    GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_render),
                  "Could not create descriptor set layout");
    serializer->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_render);
  }

  for (auto& i : dd.descriptor_pools) {
    if (i.num_ia_descriptors_remaining > 1) {
      VkDescriptorSetAllocateInfo alloc_info = {
          .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
          .pNext = nullptr,
          .descriptorPool = i.pool,
          .descriptorSetCount = 1,
          .pSetLayouts = &dd.descriptor_set_layout_for_prime_by_render};
      VkDescriptorSet ds;
      GAPID2_ASSERT(VK_SUCCESS == callee->vkAllocateDescriptorSets(device, &alloc_info, &ds),
                    "Could not allocate descriptor sets");
      serializer->vkAllocateDescriptorSets(device, &alloc_info, &ds);
      i.num_ia_descriptors_remaining -= 1;
      return std::make_pair(ds, i.pool);
    }
  }
  VkDescriptorPoolSize sz = {
      .type = VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
      .descriptorCount = 100};
  VkDescriptorPoolCreateInfo create_info = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .maxSets = 100,
      .poolSizeCount = 1,
      .pPoolSizes = &sz};

  VkDescriptorPool pool;
  GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateDescriptorPool(device, &create_info, nullptr, &pool),
                "Failed to create descriptor pool");
  serializer->vkCreateDescriptorPool(device, &create_info, nullptr, &pool);

  dd.descriptor_pools.push_back(descriptor_pool_data{
      .pool = pool,
      .num_ia_descriptors_remaining = 99,
      .num_copy_descriptors_remaining = 0});

  VkDescriptorSetAllocateInfo alloc_info = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
      .pNext = nullptr,
      .descriptorPool = pool,
      .descriptorSetCount = 1,
      .pSetLayouts = &dd.descriptor_set_layout_for_prime_by_render};
  VkDescriptorSet ds;
  GAPID2_ASSERT(VK_SUCCESS == callee->vkAllocateDescriptorSets(device, &alloc_info, &ds),
                "Could not allocate descriptor sets");
  serializer->vkAllocateDescriptorSets(device, &alloc_info, &ds);

  return std::make_pair(ds, pool);
}

std::pair<VkDescriptorSet, VkDescriptorPool> staging_resource_manager::get_copy_descriptor_set_for_device(VkDevice device) {
  if (device_data.find(device) == device_data.end()) {
    device_data[device] = device_specific_data{};
  }

  auto& dd = device_data[device];

  if (VK_NULL_HANDLE == dd.descriptor_set_layout_for_prime_by_copy) {
    VkDescriptorSetLayoutBinding binding[2] = {{.binding = 0,
                                                .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
                                                .descriptorCount = 1,
                                                .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT,
                                                .pImmutableSamplers = nullptr},
                                               {.binding = 1,
                                                .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
                                                .descriptorCount = 1,
                                                .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT,
                                                .pImmutableSamplers = nullptr}};

    VkDescriptorSetLayoutCreateInfo dsci{
        .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .bindingCount = 2,
        .pBindings = binding,
    };

    GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_copy),
                  "Could not create descriptor set layout");
    serializer->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_copy);
  }

  for (auto& i : dd.descriptor_pools) {
    if (i.num_copy_descriptors_remaining > 2) {
      VkDescriptorSetAllocateInfo alloc_info = {
          .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
          .pNext = nullptr,
          .descriptorPool = i.pool,
          .descriptorSetCount = 1,
          .pSetLayouts = &dd.descriptor_set_layout_for_prime_by_copy};
      VkDescriptorSet ds;
      GAPID2_ASSERT(VK_SUCCESS == callee->vkAllocateDescriptorSets(device, &alloc_info, &ds),
                    "Could not allocate descriptor sets");
      serializer->vkAllocateDescriptorSets(device, &alloc_info, &ds);
      i.num_copy_descriptors_remaining -= 2;
      return std::make_pair(ds, i.pool);
    }
  }
  VkDescriptorPoolSize sz = {
      .type = VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
      .descriptorCount = 200};
  VkDescriptorPoolCreateInfo create_info = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
      .pNext = nullptr,
      .flags = 0,
      .maxSets = 200,
      .poolSizeCount = 1,
      .pPoolSizes = &sz};

  VkDescriptorPool pool;
  GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateDescriptorPool(device, &create_info, nullptr, &pool),
                "Failed to create descriptor pool");
  serializer->vkCreateDescriptorPool(device, &create_info, nullptr, &pool);

  dd.descriptor_pools.push_back(descriptor_pool_data{
      .pool = pool,
      .num_ia_descriptors_remaining = 0,
      .num_copy_descriptors_remaining = 198});

  VkDescriptorSetAllocateInfo alloc_info = {
      .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
      .pNext = nullptr,
      .descriptorPool = pool,
      .descriptorSetCount = 1,
      .pSetLayouts = &dd.descriptor_set_layout_for_prime_by_copy};
  VkDescriptorSet ds;
  GAPID2_ASSERT(VK_SUCCESS == callee->vkAllocateDescriptorSets(device, &alloc_info, &ds),
                "Could not allocate descriptor sets");
  serializer->vkAllocateDescriptorSets(device, &alloc_info, &ds);

  return std::make_pair(ds, pool);
}

staging_resource_manager::render_pipeline_data staging_resource_manager::get_pipeline_for_rendering(VkDevice device, VkFormat iaFormat, VkFormat oFormat, VkImageAspectFlagBits aspect) {
  if (device_data.find(device) == device_data.end()) {
    device_data[device] = device_specific_data{};
  }
  render_pipeline_key key{.input_format = iaFormat, .output_format = oFormat, .aspect = aspect};
  auto& dd = device_data[device];
  if (dd.render_pipelines.find(key) == dd.render_pipelines.end()) {
    // We have to create this pipeline because it doesnt exist yet.

    if (dd.renderpasses.find(key) == dd.renderpasses.end()) {
      VkAttachmentReference input_ref{
          .attachment = k_render_input_attachment_index,
          .layout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL};
      VkAttachmentReference output_ref{
          .attachment = k_render_output_attachment_index,
          .layout = VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL};

      VkAttachmentDescription descs[2] = {{.flags = 0,
                                           .format = iaFormat,
                                           .samples = VK_SAMPLE_COUNT_1_BIT,
                                           .loadOp = VK_ATTACHMENT_LOAD_OP_LOAD,
                                           .storeOp = VK_ATTACHMENT_STORE_OP_DONT_CARE,
                                           .stencilLoadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE,
                                           .stencilStoreOp = VK_ATTACHMENT_STORE_OP_DONT_CARE,
                                           .initialLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL,
                                           .finalLayout = VK_IMAGE_LAYOUT_SHADER_READ_ONLY_OPTIMAL},
                                          {.flags = 0,
                                           .format = oFormat,
                                           .samples = VK_SAMPLE_COUNT_1_BIT,
                                           .loadOp = VK_ATTACHMENT_LOAD_OP_DONT_CARE,
                                           .storeOp = VK_ATTACHMENT_STORE_OP_STORE,
                                           // Keep the stencil aspect data. When rendering color or depth aspect,
                                           // stencil test will be disabled so stencil data won't be modified.
                                           .stencilLoadOp = VK_ATTACHMENT_LOAD_OP_LOAD,
                                           .stencilStoreOp = VK_ATTACHMENT_STORE_OP_STORE,
                                           .initialLayout = VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL,
                                           .finalLayout = VK_IMAGE_LAYOUT_COLOR_ATTACHMENT_OPTIMAL}};

      VkSubpassDescription subpass_desc{
          .flags = 0,
          .pipelineBindPoint = VK_PIPELINE_BIND_POINT_GRAPHICS,
          .inputAttachmentCount = 1,
          .pInputAttachments = &input_ref,
          // Color and depth attachments to be set below
          .colorAttachmentCount = 0,
          .pColorAttachments = nullptr,
          .pResolveAttachments = 0,
          .pDepthStencilAttachment = nullptr,
          .preserveAttachmentCount = 0,
          .pPreserveAttachments = nullptr};

      if (aspect == VK_IMAGE_ASPECT_DEPTH_BIT || aspect == VK_IMAGE_ASPECT_STENCIL_BIT) {
        output_ref.layout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;
        descs[1].initialLayout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;
        descs[1].finalLayout = VK_IMAGE_LAYOUT_DEPTH_STENCIL_ATTACHMENT_OPTIMAL;
        subpass_desc.pDepthStencilAttachment = &output_ref;
      } else {
        subpass_desc.colorAttachmentCount = 1;
        subpass_desc.pColorAttachments = &output_ref;
      }

      VkRenderPassCreateInfo create_info{
          .sType = VK_STRUCTURE_TYPE_RENDER_PASS_CREATE_INFO,
          .pNext = nullptr,
          .attachmentCount = 2,
          .pAttachments = descs,
          .subpassCount = 1,
          .pSubpasses = &subpass_desc,
          .dependencyCount = 0,
          .pDependencies = 0};
      VkRenderPass rp;
      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateRenderPass(device, &create_info, nullptr, &rp),
                    "Could not create render pass");
      // Create the real one just to 1) make sure it works, and 2) get the handle.
      serializer->vkCreateRenderPass(device, &create_info, nullptr, &rp);
      dd.renderpasses[key] = rp;
    }

    if (VK_NULL_HANDLE == dd.descriptor_set_layout_for_prime_by_render) {
      VkDescriptorSetLayoutBinding binding{
          .binding = k_render_input_attachment_index,
          .descriptorType = VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT,
          .descriptorCount = 1,
          .stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT,
          .pImmutableSamplers = nullptr};

      VkDescriptorSetLayoutCreateInfo dsci{
          .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
          .pNext = nullptr,
          .flags = 0,
          .bindingCount = 1,
          .pBindings = &binding,
      };

      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_render),
                    "Could not create descriptor set layout");
      serializer->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_render);
    }

    if (VK_NULL_HANDLE == dd.pipeline_layout_for_prime_by_render) {
      // Used only for stencil, but no harm in adding
      VkPushConstantRange range{
          .stageFlags = VK_SHADER_STAGE_FRAGMENT_BIT,
          .offset = 0,
          .size = 4};

      VkPipelineLayoutCreateInfo create_info{
          .sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
          .pNext = nullptr,
          .flags = 0,
          .setLayoutCount = 1,
          .pSetLayouts = &dd.descriptor_set_layout_for_prime_by_render,
          .pushConstantRangeCount = 1,
          .pPushConstantRanges = &range,
      };

      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreatePipelineLayout(device, &create_info, nullptr, &dd.pipeline_layout_for_prime_by_render),
                    "Could not create pipeline layout");
      serializer->vkCreatePipelineLayout(device, &create_info, nullptr, &dd.pipeline_layout_for_prime_by_render);
    }

    VkShaderModule vertex_module;
    VkShaderModule fragment_module;

    static const std::vector<uint32_t> empty;
    std::string created_name;
    const auto& vertex_shader_data = s_manager->get_quad_shader(&created_name);
    annotate(serializer, created_name);
    created_name.clear();

    const auto& fragment_shader = (aspect == VK_IMAGE_ASPECT_COLOR_BIT) ? s_manager->get_prime_by_rendering_color_shader(oFormat, &created_name) : (aspect == VK_IMAGE_ASPECT_DEPTH_BIT) ? s_manager->get_prime_by_rendering_depth_shader(oFormat, &created_name)
                                                                                                                                               : (aspect == VK_IMAGE_ASPECT_STENCIL_BIT) ? s_manager->get_prime_by_rendering_stencil_shader(&created_name)
                                                                                                                                                                                         : empty;
    annotate(serializer, created_name);
    GAPID2_ASSERT(!fragment_shader.empty(), "Could not get proper shader for rendering");

    {
      VkShaderModuleCreateInfo create_info{
          .sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
          .pNext = nullptr,
          .flags = 0,
          .codeSize = vertex_shader_data.size() * sizeof(uint32_t),
          .pCode = vertex_shader_data.data()};
      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateShaderModule(device, &create_info, nullptr, &vertex_module),
                    "Could not create vertex shader module");
      serializer->vkCreateShaderModule(device, &create_info, nullptr, &vertex_module);

      create_info.codeSize = fragment_shader.size() * sizeof(uint32_t);
      create_info.pCode = fragment_shader.data();
      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateShaderModule(device, &create_info, nullptr, &fragment_module),
                    "Could not create vertex shader module");
      serializer->vkCreateShaderModule(device, &create_info, nullptr, &fragment_module);
    }

    VkRenderPass render_pass = dd.renderpasses[key];
    VkDescriptorSetLayout set_layout = dd.descriptor_set_layout_for_prime_by_render;
    VkPipelineLayout pipeline_layout = dd.pipeline_layout_for_prime_by_render;

    uint32_t num_color_attachments = 1;
    VkBool32 depth_test_enabled = false;
    VkBool32 depth_write_enabled = false;
    VkBool32 stencil_test_enabled = false;
    std::vector<VkDynamicState> states{
        VK_DYNAMIC_STATE_VIEWPORT,
        VK_DYNAMIC_STATE_SCISSOR};

    if (aspect == VK_IMAGE_ASPECT_DEPTH_BIT) {
      depth_test_enabled = true;
      depth_write_enabled = true;
      num_color_attachments = 0;
    }
    if (aspect == VK_IMAGE_ASPECT_STENCIL_BIT) {
      stencil_test_enabled = true;
      num_color_attachments = 0;
      states.push_back(VK_DYNAMIC_STATE_STENCIL_WRITE_MASK);
      states.push_back(VK_DYNAMIC_STATE_STENCIL_REFERENCE);
    }

    VkPipelineShaderStageCreateInfo shader_create_info[2] = {{.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
                                                              .pNext = nullptr,
                                                              .flags = 0,
                                                              .stage = VK_SHADER_STAGE_VERTEX_BIT,
                                                              .module = vertex_module,
                                                              .pName = "main",
                                                              .pSpecializationInfo = nullptr},
                                                             {.sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
                                                              .pNext = nullptr,
                                                              .flags = 0,
                                                              .stage = VK_SHADER_STAGE_FRAGMENT_BIT,
                                                              .module = fragment_module,
                                                              .pName = "main",
                                                              .pSpecializationInfo = nullptr}};

    VkPipelineVertexInputStateCreateInfo pipeline_vertex_input_stage_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_VERTEX_INPUT_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .vertexBindingDescriptionCount = 0,
        .pVertexBindingDescriptions = nullptr,
        .vertexAttributeDescriptionCount = 0,
        .pVertexAttributeDescriptions = nullptr};

    VkPipelineInputAssemblyStateCreateInfo pipeline_input_assembly_state_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_INPUT_ASSEMBLY_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .topology = VK_PRIMITIVE_TOPOLOGY_TRIANGLE_LIST,
        .primitiveRestartEnable = false};

    VkPipelineViewportStateCreateInfo pipeline_viewport_state_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_VIEWPORT_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .viewportCount = 1,
        .pViewports = nullptr,  // set dynamically
        .scissorCount = 1,
        .pScissors = nullptr  // set dynamically
    };

    VkPipelineRasterizationStateCreateInfo pipeline_rasterization_state_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_RASTERIZATION_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .depthClampEnable = false,
        .rasterizerDiscardEnable = false,
        .polygonMode = VK_POLYGON_MODE_FILL,
        .cullMode = VK_CULL_MODE_BACK_BIT,
        .frontFace = VK_FRONT_FACE_COUNTER_CLOCKWISE,
        .depthBiasEnable = false,
        .depthBiasConstantFactor = 0,
        .depthBiasClamp = 0,
        .depthBiasSlopeFactor = 0,
        .lineWidth = 1,
    };

    VkPipelineMultisampleStateCreateInfo pipeline_multisample_state_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_MULTISAMPLE_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .rasterizationSamples = VK_SAMPLE_COUNT_1_BIT,
        .sampleShadingEnable = false,
        .minSampleShading = 0,
        .pSampleMask = 0,
        .alphaToCoverageEnable = false,
        .alphaToOneEnable = false};

    VkPipelineDepthStencilStateCreateInfo pipeline_depth_stencil_state_create_info{
        .sType = VK_STRUCTURE_TYPE_PIPELINE_DEPTH_STENCIL_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .depthTestEnable = depth_test_enabled,
        .depthWriteEnable = depth_write_enabled,
        .depthCompareOp = VK_COMPARE_OP_ALWAYS,
        .depthBoundsTestEnable = false,
        .stencilTestEnable = stencil_test_enabled,
        .front = VkStencilOpState{
            .failOp = VK_STENCIL_OP_KEEP,
            .passOp = VK_STENCIL_OP_REPLACE,
            .depthFailOp = VK_STENCIL_OP_REPLACE,
            .compareOp = VK_COMPARE_OP_ALWAYS,
            .compareMask = 0,
            .writeMask = 0,
            .reference = 0},
        .back = VkStencilOpState{.failOp = VK_STENCIL_OP_KEEP, .passOp = VK_STENCIL_OP_KEEP, .depthFailOp = VK_STENCIL_OP_KEEP, .compareOp = VK_COMPARE_OP_ALWAYS, .compareMask = 0, .writeMask = 0, .reference = 0},
        .minDepthBounds = 0.0,
        .maxDepthBounds = 0.0};

    VkPipelineColorBlendAttachmentState pipeline_color_blend_attachment_state = {
        .blendEnable = false,
        .srcColorBlendFactor = VK_BLEND_FACTOR_ZERO,
        .dstColorBlendFactor = VK_BLEND_FACTOR_ONE,
        .colorBlendOp = VK_BLEND_OP_ADD,
        .srcAlphaBlendFactor = VK_BLEND_FACTOR_ZERO,
        .dstAlphaBlendFactor = VK_BLEND_FACTOR_ONE,
        .alphaBlendOp = VK_BLEND_OP_ADD,
        .colorWriteMask = 0xf};

    VkPipelineColorBlendStateCreateInfo pipeline_color_blend_state_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_COLOR_BLEND_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .logicOpEnable = false,
        .logicOp = VK_LOGIC_OP_CLEAR,
        .attachmentCount = num_color_attachments,
        .pAttachments = &pipeline_color_blend_attachment_state,
        .blendConstants = {0.0f, 0.0f, 0.0f, 0.0f}};

    VkPipelineDynamicStateCreateInfo pipeline_dynamic_state_create_info = {
        .sType = VK_STRUCTURE_TYPE_PIPELINE_DYNAMIC_STATE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .dynamicStateCount = static_cast<uint32_t>(states.size()),
        .pDynamicStates = states.data()};

    VkPipeline pipeline;

    VkGraphicsPipelineCreateInfo graphics_pipeline_create_info = {
        .sType = VK_STRUCTURE_TYPE_GRAPHICS_PIPELINE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .stageCount = 2,
        .pStages = shader_create_info,
        .pVertexInputState = &pipeline_vertex_input_stage_create_info,
        .pInputAssemblyState = &pipeline_input_assembly_state_create_info,
        .pTessellationState = nullptr,
        .pViewportState = &pipeline_viewport_state_create_info,
        .pRasterizationState = &pipeline_rasterization_state_create_info,
        .pMultisampleState = &pipeline_multisample_state_create_info,
        .pDepthStencilState = &pipeline_depth_stencil_state_create_info,
        .pColorBlendState = &pipeline_color_blend_state_create_info,
        .pDynamicState = &pipeline_dynamic_state_create_info,
        .layout = pipeline_layout,
        .renderPass = render_pass,
        .subpass = 0,
        .basePipelineHandle = 0,
        .basePipelineIndex = 0};

    GAPID2_ASSERT(VK_SUCCESS ==
                      callee->vkCreateGraphicsPipelines(device,
                                                        VK_NULL_HANDLE, 1, &graphics_pipeline_create_info,
                                                        nullptr, &pipeline),
                  "Could not create graphics pipeline");
    serializer->vkCreateGraphicsPipelines(device,
                                          VK_NULL_HANDLE, 1, &graphics_pipeline_create_info,
                                          nullptr, &pipeline);
    dd.render_pipelines[key] = pipeline_data{
        .pipeline = pipeline,
        .pipeline_layout = pipeline_layout,
        .renderpass = render_pass};

    // Post creation cleanup (shader modules are temporary)
    {
      callee->vkDestroyShaderModule(device, vertex_module, nullptr);
      serializer->vkDestroyShaderModule(device, vertex_module, nullptr);
      callee->vkDestroyShaderModule(device, fragment_module, nullptr);
      serializer->vkDestroyShaderModule(device, fragment_module, nullptr);
    }
  }

  // Create descriptor set.

  const auto& pl = dd.render_pipelines[key];
  auto desc = get_input_attachment_descriptor_set_for_device(device);
  return render_pipeline_data{
      .device = device,
      .render_pass = pl.renderpass,
      .pipeline = pl.pipeline,
      .pool = desc.second,
      .render_ds = desc.first,
      .pipeline_layout = pl.pipeline_layout};
}

void staging_resource_manager::cleanup_after_pipeline(const render_pipeline_data& data) {
  //serializer->vkDestroyRenderPass(data.device, data.render_pass, nullptr);
  //callee->vkDestroyRenderPass(data.device, data.render_pass, nullptr);

  serializer->vkFreeDescriptorSets(data.device, data.pool, 1, &data.render_ds);
  callee->vkFreeDescriptorSets(data.device, data.pool, 1, &data.render_ds);

  auto dd = device_data[data.device];
  for (auto& d : dd.descriptor_pools) {
    if (d.pool = data.pool) {
      d.num_ia_descriptors_remaining += 1;
    }
  }
}

staging_resource_manager::copy_pipeline_data staging_resource_manager::get_pipeline_for_copy(VkDevice device, VkFormat iaFormat, VkFormat oFormat, VkImageAspectFlagBits input_aspect, VkImageAspectFlagBits output_aspect, VkImageType type) {
  if (device_data.find(device) == device_data.end()) {
    device_data[device] = device_specific_data{};
  }
  auto& dd = device_data[device];

  copy_pipeline_key key{
      .input_format = iaFormat,
      .output_format = oFormat,
      .input_aspect = input_aspect,
      .output_aspect = output_aspect,
      .type = type};

  if (dd.copy_pipelines.find(key) == dd.copy_pipelines.end()) {
    if (VK_NULL_HANDLE == dd.descriptor_set_layout_for_prime_by_copy) {
      VkDescriptorSetLayoutBinding binding[2] = {{.binding = 0,
                                                  .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
                                                  .descriptorCount = 1,
                                                  .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT,
                                                  .pImmutableSamplers = nullptr},
                                                 {.binding = 1,
                                                  .descriptorType = VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
                                                  .descriptorCount = 1,
                                                  .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT,
                                                  .pImmutableSamplers = nullptr}};

      VkDescriptorSetLayoutCreateInfo dsci{
          .sType = VK_STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
          .pNext = nullptr,
          .flags = 0,
          .bindingCount = 2,
          .pBindings = binding,
      };

      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_copy),
                    "Could not create descriptor set layout");
      serializer->vkCreateDescriptorSetLayout(device, &dsci, nullptr, &dd.descriptor_set_layout_for_prime_by_copy);
    }

    if (VK_NULL_HANDLE == dd.pipeline_layout_for_prime_by_copy) {
      // Used only for stencil, but no harm in adding
      VkPushConstantRange range{
          .stageFlags = VK_SHADER_STAGE_COMPUTE_BIT,
          .offset = 0,
          .size = 4 * 4};

      VkPipelineLayoutCreateInfo create_info{
          .sType = VK_STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
          .pNext = nullptr,
          .flags = 0,
          .setLayoutCount = 1,
          .pSetLayouts = &dd.descriptor_set_layout_for_prime_by_copy,
          .pushConstantRangeCount = 1,
          .pPushConstantRanges = &range,
      };

      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreatePipelineLayout(device, &create_info, nullptr, &dd.pipeline_layout_for_prime_by_copy),
                    "Could not create pipeline layout");
      serializer->vkCreatePipelineLayout(device, &create_info, nullptr, &dd.pipeline_layout_for_prime_by_copy);
    }

    VkShaderModule compute_shader;

    static const std::vector<uint32_t> empty;
    std::string created_name;
    const auto& shader_data = s_manager->get_prime_by_compute_store_shader(oFormat, output_aspect, iaFormat, input_aspect, type, &created_name);
    annotate(serializer, created_name);
    GAPID2_ASSERT(!shader_data.empty(), "Could not get proper shader for copying");

    {
      VkShaderModuleCreateInfo create_info{
          .sType = VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
          .pNext = nullptr,
          .flags = 0,
          .codeSize = shader_data.size() * sizeof(uint32_t),
          .pCode = shader_data.data()};
      GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateShaderModule(device, &create_info, nullptr, &compute_shader),
                    "Could not create vertex shader module");
      serializer->vkCreateShaderModule(device, &create_info, nullptr, &compute_shader);
    }

    VkComputePipelineCreateInfo create_info{
        .sType = VK_STRUCTURE_TYPE_COMPUTE_PIPELINE_CREATE_INFO,
        .pNext = nullptr,
        .flags = 0,
        .stage = VkPipelineShaderStageCreateInfo{
            .sType = VK_STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
            .pNext = nullptr,
            .flags = 0,
            .stage = VK_SHADER_STAGE_COMPUTE_BIT,
            .module = compute_shader,
            .pName = "main",
            .pSpecializationInfo = nullptr},
        .layout = dd.pipeline_layout_for_prime_by_copy,
        .basePipelineHandle = 0,
        .basePipelineIndex = 0};

    VkPipeline compute_pipeline;
    GAPID2_ASSERT(VK_SUCCESS == callee->vkCreateComputePipelines(device, VK_NULL_HANDLE,
                                                                 1, &create_info, nullptr, &compute_pipeline),
                  "Could not create compute copy pipeline");
    serializer->vkCreateComputePipelines(device, VK_NULL_HANDLE,
                                         1, &create_info, nullptr, &compute_pipeline);

    dd.copy_pipelines[key] = copy_pipeline_dat{
        .pipeline = compute_pipeline,
        .pipeline_layout = dd.pipeline_layout_for_prime_by_copy};
  }

  auto copy_pipe = dd.copy_pipelines[key];
  auto copy_ds = get_copy_descriptor_set_for_device(device);

  return copy_pipeline_data{
      .device = device,
      .pipeline = copy_pipe.pipeline,
      .pool = copy_ds.second,
      .copy_ds = copy_ds.first,
      .pipeline_layout = copy_pipe.pipeline_layout};
}

void staging_resource_manager::cleanup_after_pipeline(const copy_pipeline_data& data) {
  serializer->vkFreeDescriptorSets(data.device, data.pool, 1, &data.copy_ds);
  callee->vkFreeDescriptorSets(data.device, data.pool, 1, &data.copy_ds);

  auto dd = device_data[data.device];
  for (auto& d : dd.descriptor_pools) {
    if (d.pool = data.pool) {
      d.num_copy_descriptors_remaining += 2;
    }
  }
}

}  // namespace gapid2
