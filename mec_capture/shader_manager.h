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

#define VK_NO_PROTOTYPES
#include <vulkan/vulkan.h>

#include <unordered_map>

namespace gapid2 {

static const uint32_t k_store_output_image_binding = 0;
static const uint32_t k_store_input_image_binding = 1;
static const uint32_t k_store_max_compute_group_count_x = 65536;
static const uint32_t k_store_max_compute_group_count_y = 65536;
static const uint32_t k_store_max_compute_group_count_z = 65536;
static const uint32_t k_store_initial_descriptor_set_size = 16;
static const VkImageLayout k_store_image_layout = VK_IMAGE_LAYOUT_GENERAL;
static const uint32_t k_render_input_attachment_index = 0;
static const uint32_t k_render_output_attachment_index = 1;

class shader_manager {
 public:
  shader_manager();
  const std::vector<uint32_t>& get_quad_shader(std::string* created_name);
  const std::vector<uint32_t>& get_prime_by_rendering_color_shader(VkFormat format, std::string* created_name);
  const std::vector<uint32_t>& get_prime_by_rendering_depth_shader(VkFormat format, std::string* created_name);
  const std::vector<uint32_t>& get_copy_by_rendering_color_shader(VkFormat format, std::string* created_name);
  const std::vector<uint32_t>& get_copy_stencil_by_render_shader(VkFormat format, std::string* created_name);
  const std::vector<uint32_t>& get_prime_by_rendering_stencil_shader(std::string* created_name);

  const std::vector<uint32_t>& get_prime_by_compute_copy_shader(VkFormat format,
                                                                VkImageAspectFlagBits aspect, VkImageType type, std::string* created_name);
  const std::vector<uint32_t>& get_prime_by_compute_store_shader(VkFormat output_format, VkImageAspectFlagBits output_aspect, VkFormat input_format, VkImageAspectFlagBits input_aspect, VkImageType type, std::string* created_name);

 private:
  std::vector<uint32_t> quad_vertex_shader_spirv;
  std::unordered_map<VkFormat, std::vector<uint32_t>> prime_by_rendering_color_shaders;
  std::unordered_map<VkFormat, std::vector<uint32_t>> prime_by_rendering_depth_shaders;
  std::vector<uint32_t> prime_by_rendering_stencil_shader_spirv;
  struct compute_copy_key {
    VkFormat input_format;
    VkFormat output_format;
    VkImageAspectFlagBits input_aspect;
    VkImageAspectFlagBits output_aspect;
    VkImageType type;
  };

  struct compute_copy_key_hasher {
    size_t operator()(const compute_copy_key& key) const {
      return std::hash<uint32_t>()(key.input_format) ^
             (std::hash<uint32_t>()(key.output_format) << 1) ^
             (std::hash<uint32_t>()(key.input_aspect) << 2) ^
             (std::hash<uint32_t>()(key.type) << 3) ^
             (std::hash<uint32_t>()(key.output_aspect) << 4);
    }
  };

  struct compute_key_equals {
    bool operator()(const compute_copy_key& a, const compute_copy_key& b) const {
      return a.input_aspect == b.input_aspect &&
             a.input_format == b.input_format &&
             a.output_aspect == b.output_aspect &&
             a.output_format == b.output_format &&
             a.type == b.type;
    };
  };

  std::unordered_map<compute_copy_key, std::vector<uint32_t>, compute_copy_key_hasher, compute_key_equals>
      prime_by_compute_copy_shaders;
  std::unordered_map<compute_copy_key, std::vector<uint32_t>, compute_copy_key_hasher, compute_key_equals>
      prime_by_compute_store_shaders;

  std::unordered_map<VkFormat, std::vector<uint32_t>> copy_by_rendering_color_shaders;
  std::unordered_map<VkFormat, std::vector<uint32_t>> copy_stencil_by_render_shaders;
};

}  // namespace gapid2