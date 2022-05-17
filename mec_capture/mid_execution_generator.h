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
#include "command_serializer.h"
#include "transform_base.h"

namespace gapid2 {

class mid_execution_generator {
 public:
  void begin_mid_execution_capture(
      const state_block* state_block, command_serializer* mid_execution_serializer, transform_base* bypass_caller) {
    capture_instances(state_block, mid_execution_serializer, bypass_caller);
    capture_physical_devices(state_block, mid_execution_serializer, bypass_caller);
    capture_surfaces(state_block, mid_execution_serializer, bypass_caller);

    capture_devices(state_block, mid_execution_serializer, bypass_caller);
    capture_queues(state_block, mid_execution_serializer, bypass_caller);
    capture_swapchains(state_block, mid_execution_serializer, bypass_caller);

    capture_buffers(state_block, mid_execution_serializer, bypass_caller);
    capture_images(state_block, mid_execution_serializer, bypass_caller);
    capture_allocations(state_block, mid_execution_serializer, bypass_caller);

    capture_bind_images(state_block, mid_execution_serializer, bypass_caller);
    capture_bind_buffers(state_block, mid_execution_serializer, bypass_caller);

    // capture_buffer_data
    // capture_image_data

    // capture_image_layouts_and_queues

    capture_sampler_ycbcr_conversions(state_block, mid_execution_serializer, bypass_caller);
    capture_samplers(state_block, mid_execution_serializer, bypass_caller);
    capture_command_pools(state_block, mid_execution_serializer, bypass_caller);
    capture_pipeline_caches(state_block, mid_execution_serializer, bypass_caller);
    capture_descriptor_set_layouts(state_block, mid_execution_serializer, bypass_caller);
    capture_descriptor_update_templates(state_block, mid_execution_serializer, bypass_caller);
    capture_pipeline_layouts(state_block, mid_execution_serializer, bypass_caller);
    capture_render_passes(state_block, mid_execution_serializer, bypass_caller);
    capture_shader_modules(state_block, mid_execution_serializer, bypass_caller);
    capture_pipelines(state_block, mid_execution_serializer, bypass_caller);
    capture_image_views(state_block, mid_execution_serializer, bypass_caller);
    capture_buffer_views(state_block, mid_execution_serializer, bypass_caller);
    capture_descriptor_pools(state_block, mid_execution_serializer, bypass_caller);
    capture_framebuffers(state_block, mid_execution_serializer, bypass_caller);
    capture_descriptor_sets(state_block, mid_execution_serializer, bypass_caller);

    capture_descriptor_set_contents(state_block, mid_execution_serializer, bypass_caller);

    capture_query_pools(state_block, mid_execution_serializer, bypass_caller);
    capture_command_buffers(state_block, mid_execution_serializer, bypass_caller, VK_COMMAND_BUFFER_LEVEL_SECONDARY);
    capture_command_buffers(state_block, mid_execution_serializer, bypass_caller, VK_COMMAND_BUFFER_LEVEL_PRIMARY);
    capture_synchronization(state_block, mid_execution_serializer, bypass_caller);
  }

 private:
  void capture_instances(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_physical_devices(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_surfaces(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_devices(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_queues(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_swapchains(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_images(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_bind_images(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_bind_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_allocations(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_sampler_ycbcr_conversions(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_samplers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_command_pools(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_pipeline_caches(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_descriptor_set_layouts(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_descriptor_update_templates(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_pipeline_layouts(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_render_passes(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_shader_modules(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_pipelines(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_image_views(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_buffer_views(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_descriptor_pools(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_framebuffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_descriptor_sets(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_descriptor_set_contents(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_query_pools(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;
  void capture_command_buffers(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller, VkCommandBufferLevel level) const;
  void capture_synchronization(const state_block* state_block, command_serializer* serializer, transform_base* bypass_caller) const;

  VkQueue get_queue_for_device(const state_block* state_block, VkDevice device) const;
};

}  // namespace gapid2