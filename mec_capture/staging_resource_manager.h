#pragma once
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

#include <functional>
#include <list>

#include "command_serializer.h"
#include "device.h"
#include "queue.h"
#include "transform_base.h"

namespace gapid2 {
class shader_manager;
class staging_resource_manager {
 public:
  staging_resource_manager(transform_base* callee,
                           command_serializer* serializer,
                           const VkPhysicalDeviceWrapper* physical_device,
                           const VkDeviceWrapper* device,
                           VkDeviceSize maximum_size,
                           shader_manager* s_manager);

  ~staging_resource_manager();

  void flush();

  struct staging_resources {
    VkCommandBuffer cb;
    VkDeviceSize buffer_offset;
    VkBuffer buffer;
    VkDeviceSize returned_size;
    VkDeviceMemory memory;
  };

  // Takes in a requested byte size. Returns the number offset
  //   into the buffer that can be copied to.
  // It will flush if the number of bytes cannot be satisfied, but will never
  //   return more than `maximum_size` bytes.
  // It will call `callback` when the data ends up getting flushed out.
  staging_resources get_staging_buffer_for_queue(
      const VkQueueWrapper* queue,
      VkDeviceSize bufferSize,
      std::function<void(const char* data, VkDeviceSize size, std::vector<std::function<void()>>* cleanups)> fn);

  // This provides a command buffer that can be used on the given queue.
  // This command buffer is only valid until the next call to `get_**_for_queue`
  // as any subsequent calls MAY cause a flush. Which would submit this
  // command buffer. get_command_buffer_for_queue must be called again
  // in that case.
  VkCommandBuffer get_command_buffer_for_queue(const VkQueueWrapper* queue);

  struct render_pipeline_data {
    VkDevice device;
    VkRenderPass render_pass;
    VkPipeline pipeline;
    VkDescriptorPool pool;
    VkDescriptorSet render_ds;
    VkPipelineLayout pipeline_layout;
  };
  render_pipeline_data get_pipeline_for_rendering(VkDevice device, VkFormat iaFormat, VkFormat oFormat, VkImageAspectFlagBits aspect);

  void cleanup_after_pipeline(const render_pipeline_data& data);

  struct copy_pipeline_data {
    VkDevice device;
    VkPipeline pipeline;
    VkDescriptorPool pool;
    VkDescriptorSet copy_ds;
    VkPipelineLayout pipeline_layout;
  };
  copy_pipeline_data get_pipeline_for_copy(VkDevice device, VkFormat iaFormat, VkFormat oFormat, VkImageAspectFlagBits input_aspect, VkImageAspectFlagBits output_aspect, VkImageType type);
  void cleanup_after_pipeline(const copy_pipeline_data& data);

 private:
  std::pair<VkDescriptorSet, VkDescriptorPool> get_input_attachment_descriptor_set_for_device(VkDevice device);
  std::pair<VkDescriptorSet, VkDescriptorPool> get_copy_descriptor_set_for_device(VkDevice device);
  uint32_t get_memory_type_index_for_staging_resource(
      const VkPhysicalDeviceMemoryProperties& phy_dev_prop,
      uint32_t requirement_type_bits);

  transform_base* callee;
  command_serializer* serializer;
  const VkDeviceWrapper* device;
  VkBuffer dest_buffer;
  VkDeviceMemory device_memory;
  char* device_memory_ptr;
  VkDeviceSize offset = 0;
  VkDeviceSize maximum_size;
  shader_manager* s_manager;

  struct data_offset {
    std::function<void(char* data, VkDeviceSize size, std::vector<std::function<void()>>*)> call;
    char* offs;
    VkDeviceSize size;
  };

  std::list<data_offset> run_data;

  struct queue_specific_data {
    VkDevice device;
    VkCommandPool command_pool;
    VkCommandBuffer command_buffer;
  };

  struct render_pipeline_key {
    VkFormat input_format;
    VkFormat output_format;
    VkImageAspectFlagBits aspect;
  };

  struct render_pipeline_key_hasher {
    size_t operator()(const render_pipeline_key& key) const {
      return std::hash<uint32_t>()(key.input_format) ^
             (std::hash<uint32_t>()(key.output_format) << 1) ^
             (std::hash<uint32_t>()(key.aspect) << 2);
    }
    bool operator()(const render_pipeline_key& a, const render_pipeline_key& b) const {
      return a.input_format == b.input_format && a.output_format == b.output_format && a.aspect == b.aspect;
    }
  };

  struct copy_pipeline_key {
    VkFormat input_format;
    VkFormat output_format;
    VkImageAspectFlagBits input_aspect;
    VkImageAspectFlagBits output_aspect;
    VkImageType type;
  };

  struct copy_pipeline_key_hasher {
    size_t operator()(const copy_pipeline_key& key) const {
      return std::hash<uint32_t>()(key.input_format) ^
             (std::hash<uint32_t>()(key.output_format) << 1) ^
             (std::hash<uint32_t>()(key.input_aspect) << 2) ^
             (std::hash<uint32_t>()(key.output_aspect) << 3) ^
             (std::hash<uint32_t>()(key.type) << 4);
    }
    bool operator()(const copy_pipeline_key& a, const copy_pipeline_key& b) const {
      return a.input_format == b.input_format &&
             a.output_format == b.output_format &&
             a.input_aspect == b.input_aspect &&
             a.output_aspect == b.output_aspect &&
             a.type == b.type;
    }
  };

  struct descriptor_pool_data {
    VkDescriptorPool pool;
    uint32_t num_ia_descriptors_remaining = 0;
    uint32_t num_copy_descriptors_remaining = 0;
  };

  struct pipeline_data {
    VkPipeline pipeline;
    VkPipelineLayout pipeline_layout;
    VkRenderPass renderpass;
  };

  struct copy_pipeline_dat {
    VkPipeline pipeline;
    VkPipelineLayout pipeline_layout;
  };

  struct device_specific_data {
    std::vector<descriptor_pool_data> descriptor_pools;
    VkDescriptorSetLayout descriptor_set_layout_for_prime_by_render = VK_NULL_HANDLE;
    VkPipelineLayout pipeline_layout_for_prime_by_render = VK_NULL_HANDLE;
    VkDescriptorSetLayout descriptor_set_layout_for_prime_by_copy = VK_NULL_HANDLE;
    VkPipelineLayout pipeline_layout_for_prime_by_copy = VK_NULL_HANDLE;
    std::unordered_map<render_pipeline_key, VkRenderPass, render_pipeline_key_hasher, render_pipeline_key_hasher> renderpasses;
    std::unordered_map<render_pipeline_key, pipeline_data, render_pipeline_key_hasher, render_pipeline_key_hasher> render_pipelines;

    std::unordered_map<copy_pipeline_key, copy_pipeline_dat, copy_pipeline_key_hasher, copy_pipeline_key_hasher> copy_pipelines;
  };

  std::unordered_map<VkQueue, queue_specific_data> queue_data;
  std::unordered_map<VkDevice, device_specific_data> device_data;
};

VkQueue get_queue_for_family(const state_block* sb, VkDevice device, uint32_t queue_family);

}  // namespace gapid2