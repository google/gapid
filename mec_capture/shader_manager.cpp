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

#include "shader_manager.h"

#include <format>
#include <string>

#include "Standalone\resource_limits_c.h"
#include "common.h"
#include "glslang\Include\glslang_c_interface.h"

namespace gapid2 {

struct glslang_cleaner {
  glslang_cleaner() {
    glslang_initialize_process();
  }
  ~glslang_cleaner() {
    glslang_finalize_process();
  }
};

shader_manager::shader_manager() {
  static glslang_cleaner cleaner;
}

const char* storage_image_format(VkFormat format) {
  switch (format) {
    // uint formats
    case VK_FORMAT_R8_UINT:
      return "r8ui";
    case VK_FORMAT_R16_UINT:
      return "r16ui";
    case VK_FORMAT_R32_UINT:
      return "r32ui";
    case VK_FORMAT_R8G8_UINT:
      return "rg8ui";
    case VK_FORMAT_R16G16_UINT:
      return "rg16ui";
    case VK_FORMAT_R32G32_UINT:
      return "rg32ui";

    case VK_FORMAT_R8G8B8A8_UINT:
    case VK_FORMAT_B8G8R8A8_UINT:
    case VK_FORMAT_A8B8G8R8_UINT_PACK32:
      return "rgba8ui";
    case VK_FORMAT_R16G16B16A16_UINT:
      return "rgba16ui";
    case VK_FORMAT_R32G32B32A32_UINT:
      return "rgba32ui";

    case VK_FORMAT_A2R10G10B10_UINT_PACK32:
    case VK_FORMAT_A2B10G10R10_UINT_PACK32:
      return "rgb10_a2ui";

    // sint formats
    case VK_FORMAT_R8_SINT:
      return "r8i";
    case VK_FORMAT_R16_SINT:
      return "r16i";
    case VK_FORMAT_R32_SINT:
      return "r32i";

    case VK_FORMAT_R8G8_SINT:
      return "rg8i";
    case VK_FORMAT_R16G16_SINT:
      return "rg16i";
    case VK_FORMAT_R32G32_SINT:
      return "rg32i";

    case VK_FORMAT_R8G8B8A8_SINT:
    case VK_FORMAT_B8G8R8A8_SINT:
    case VK_FORMAT_A8B8G8R8_SINT_PACK32:
      return "rgba8i";
    case VK_FORMAT_R16G16B16A16_SINT:
      return "rgba16i";
    case VK_FORMAT_R32G32B32A32_SINT:
      return "rgba32i";

    // unorm formats
    case VK_FORMAT_R8_UNORM:
    case VK_FORMAT_R8_SRGB:
      return "r8";
    case VK_FORMAT_R16_UNORM:
      return "r16";

    case VK_FORMAT_R8G8_UNORM:
    case VK_FORMAT_R8G8_SRGB:
      return "rg8";
    case VK_FORMAT_R16G16_UNORM:
      return "rg16";

    case VK_FORMAT_R8G8B8A8_UNORM:
    case VK_FORMAT_B8G8R8A8_UNORM:
    case VK_FORMAT_R8G8B8A8_SRGB:
    case VK_FORMAT_B8G8R8A8_SRGB:
      return "rgba8";

    case VK_FORMAT_R16G16B16A16_UNORM:
      return "rgba16";

    case VK_FORMAT_A8B8G8R8_UNORM_PACK32:
    case VK_FORMAT_A8B8G8R8_SRGB_PACK32:
      return "rgba8";

    case VK_FORMAT_A2R10G10B10_UNORM_PACK32:
    case VK_FORMAT_A2B10G10R10_UNORM_PACK32:
      return "rgb10_a2";

    // snorm formats
    case VK_FORMAT_R8_SNORM:
      return "r8_snorm";
    case VK_FORMAT_R16_SNORM:
      return "r16_snorm";

    case VK_FORMAT_R8G8_SNORM:
      return "rg8_snorm";
    case VK_FORMAT_R16G16_SNORM:
      return "rg16_snorm";

    case VK_FORMAT_R8G8B8A8_SNORM:
    case VK_FORMAT_B8G8R8A8_SNORM:
    case VK_FORMAT_A8B8G8R8_SNORM_PACK32:
      return "rgba8_snorm";
    case VK_FORMAT_R16G16B16A16_SNORM:
      return "rgba16_snorm";

    // float formats
    case VK_FORMAT_R16_SFLOAT:
      return "r16f";
    case VK_FORMAT_R32_SFLOAT:
      return "r32f";

    case VK_FORMAT_R16G16_SFLOAT:
      return "rg16f";
    case VK_FORMAT_R32G32_SFLOAT:
      return "rg32f";

    case VK_FORMAT_R16G16B16A16_SFLOAT:
      return "rgba16f";
    case VK_FORMAT_R32G32B32A32_SFLOAT:
      return "rgba32f";

    case VK_FORMAT_B10G11R11_UFLOAT_PACK32:
      return "r11f_g11f_b10f";
  }
  GAPID2_ERROR("Unsupported format");
  return nullptr;
}

const char* storage_image_unit(VkFormat format) {
  switch (format) {
    case VK_FORMAT_R8_UINT:
    case VK_FORMAT_R16_UINT:
    case VK_FORMAT_R32_UINT:
    case VK_FORMAT_R8G8_UINT:
    case VK_FORMAT_R16G16_UINT:
    case VK_FORMAT_R32G32_UINT:
    case VK_FORMAT_R8G8B8A8_UINT:
    case VK_FORMAT_B8G8R8A8_UINT:
    case VK_FORMAT_A8B8G8R8_UINT_PACK32:
    case VK_FORMAT_R16G16B16A16_UINT:
    case VK_FORMAT_R32G32B32A32_UINT:
    case VK_FORMAT_A2R10G10B10_UINT_PACK32:
    case VK_FORMAT_A2B10G10R10_UINT_PACK32:
      return "u";

    case VK_FORMAT_R8_SINT:
    case VK_FORMAT_R16_SINT:
    case VK_FORMAT_R32_SINT:
    case VK_FORMAT_R8G8_SINT:
    case VK_FORMAT_R16G16_SINT:
    case VK_FORMAT_R32G32_SINT:
    case VK_FORMAT_R8G8B8A8_SINT:
    case VK_FORMAT_B8G8R8A8_SINT:
    case VK_FORMAT_A8B8G8R8_SINT_PACK32:
    case VK_FORMAT_R16G16B16A16_SINT:
    case VK_FORMAT_R32G32B32A32_SINT:
      return "i";

    // unorm formats
    case VK_FORMAT_R8_UNORM:
    case VK_FORMAT_R8_SRGB:
    case VK_FORMAT_R16_UNORM:
    case VK_FORMAT_R8G8_UNORM:
    case VK_FORMAT_R8G8_SRGB:
    case VK_FORMAT_R16G16_UNORM:
    case VK_FORMAT_R8G8B8A8_UNORM:
    case VK_FORMAT_B8G8R8A8_UNORM:
    case VK_FORMAT_R8G8B8A8_SRGB:
    case VK_FORMAT_B8G8R8A8_SRGB:
    case VK_FORMAT_R16G16B16A16_UNORM:
    case VK_FORMAT_A8B8G8R8_UNORM_PACK32:
    case VK_FORMAT_A8B8G8R8_SRGB_PACK32:
    case VK_FORMAT_A2R10G10B10_UNORM_PACK32:
    case VK_FORMAT_A2B10G10R10_UNORM_PACK32:
      // snorm formats
    case VK_FORMAT_R8_SNORM:
    case VK_FORMAT_R16_SNORM:
    case VK_FORMAT_R8G8_SNORM:
    case VK_FORMAT_R16G16_SNORM:
    case VK_FORMAT_R8G8B8A8_SNORM:
    case VK_FORMAT_B8G8R8A8_SNORM:
    case VK_FORMAT_A8B8G8R8_SNORM_PACK32:
    case VK_FORMAT_R16G16B16A16_SNORM:
      // float formats
    case VK_FORMAT_R16_SFLOAT:
    case VK_FORMAT_R32_SFLOAT:
    case VK_FORMAT_R16G16_SFLOAT:
    case VK_FORMAT_R32G32_SFLOAT:
    case VK_FORMAT_R16G16B16A16_SFLOAT:
    case VK_FORMAT_R32G32B32A32_SFLOAT:
    case VK_FORMAT_B10G11R11_UFLOAT_PACK32:
      return "";
  }
  GAPID2_ERROR("Unsupported format");
  return nullptr;
}

const char* storage_image_type(VkImageType ty) {
  switch (ty) {
    case VK_IMAGE_TYPE_1D:
      return "image1D";
    case VK_IMAGE_TYPE_2D:
      return "image2D";
    case VK_IMAGE_TYPE_3D:
      return "image3D";
  }
  GAPID2_ERROR("Unsupported image type");
  return nullptr;
}

const char* storage_image_position(VkImageType ty) {
  switch (ty) {
    case VK_IMAGE_TYPE_1D:
      return "int pos = x;";
    case VK_IMAGE_TYPE_2D:
      return "ivec2 pos = ivec2(x, y);";
    case VK_IMAGE_TYPE_3D:
      return "ivec3 pos = ivec3(x, y, z);";
  }
  GAPID2_ERROR("Unsupported image type");
  return nullptr;
}

std::vector<uint32_t> compile_shader_to_spirv(glslang_stage_t stage, const char* fileName, const char* shaderSource) {
  const glslang_input_t input = {
      .language = GLSLANG_SOURCE_GLSL,
      .stage = stage,
      .client = GLSLANG_CLIENT_VULKAN,
      .client_version = GLSLANG_TARGET_VULKAN_1_0,
      .target_language = GLSLANG_TARGET_SPV,
      .target_language_version = GLSLANG_TARGET_SPV_1_0,
      .code = shaderSource,
      .default_version = 100,
      .default_profile = GLSLANG_NO_PROFILE,
      .force_default_version_and_profile = false,
      .forward_compatible = false,
      .messages = GLSLANG_MSG_DEFAULT_BIT,
      .resource = glslang_default_resource(),
  };

  glslang_shader_t* shader = glslang_shader_create(&input);

  if (!glslang_shader_preprocess(shader, &input)) {
    printf("GLSL preprocessing failed %s\n", fileName);
    printf("%s\n", glslang_shader_get_info_log(shader));
    printf("%s\n", glslang_shader_get_info_debug_log(shader));
    printf("%s\n", input.code);
    glslang_shader_delete(shader);
    return std::vector<uint32_t>();
  }

  if (!glslang_shader_parse(shader, &input)) {
    printf("GLSL parsing failed %s\n", fileName);
    printf("%s\n", glslang_shader_get_info_log(shader));
    printf("%s\n", glslang_shader_get_info_debug_log(shader));
    printf("%s\n", glslang_shader_get_preprocessed_code(shader));
    glslang_shader_delete(shader);
    return std::vector<uint32_t>();
  }

  glslang_program_t* program = glslang_program_create();
  glslang_program_add_shader(program, shader);

  if (!glslang_program_link(program, GLSLANG_MSG_SPV_RULES_BIT | GLSLANG_MSG_VULKAN_RULES_BIT)) {
    printf("GLSL linking failed %s\n", fileName);
    printf("%s\n", glslang_program_get_info_log(program));
    printf("%s\n", glslang_program_get_info_debug_log(program));
    glslang_program_delete(program);
    glslang_shader_delete(shader);
    return std::vector<uint32_t>();
  }

  glslang_program_SPIRV_generate(program, stage);

  std::vector<uint32_t> outShaderModule(glslang_program_SPIRV_get_size(program));
  glslang_program_SPIRV_get(program, outShaderModule.data());

  const char* spirv_messages = glslang_program_SPIRV_get_messages(program);
  if (spirv_messages)
    printf("(%s) %s\b", fileName, spirv_messages);

  glslang_program_delete(program);
  glslang_shader_delete(shader);

  return outShaderModule;
}

const std::vector<uint32_t>& shader_manager::get_quad_shader(std::string* created_name) {
  if (quad_vertex_shader_spirv.empty()) {
    const auto shader =
        quad_vertex_shader_spirv = compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_VERTEX,
                                                           "quad_shader",
                                                           std::format(R"(#version 450
vec2 positions[6] = vec2[](
	vec2(1.0, 1.0),
	vec2(-1.0, -1.0),
	vec2(-1.0, 1.0),
	vec2(1.0, 1.0),
	vec2(1.0, -1.0),
	vec2(-1.0, -1.0)
);
void main() {{
	gl_Position = vec4(positions[gl_VertexIndex], 0.0, 1.0);
}})")
                                                               .c_str());
    created_name[0] = "quad_shader";
  }

  return quad_vertex_shader_spirv;
}

const std::vector<uint32_t>& shader_manager::get_copy_by_rendering_color_shader(VkFormat format, std::string* created_name) {
  if (!copy_by_rendering_color_shaders.count(format)) {
    auto name = std::format("copy_render_by_color{}", static_cast<uint32_t>(format));

    auto unit = storage_image_unit(format);
    copy_by_rendering_color_shaders[format] =
        compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                name.c_str(),
                                std::format(R"(#version 450
	precision highp int;
	precision highp float;
	layout(location = 0) out {}vec4 out_color;
	layout(input_attachment_index = 0, binding = 0, set = 0) uniform {}subpassInput in_color;
	void main() {{
		out_color = subpassLoad(in_color);
	}})",
                                            unit, unit)
                                    .c_str());
    created_name[0] = name;
  }

  return copy_by_rendering_color_shaders[format];
}

const std::vector<uint32_t>& shader_manager::get_copy_stencil_by_render_shader(VkFormat format, std::string* created_name) {
  if (!copy_stencil_by_render_shaders.count(format)) {
    auto name = std::format("copy_render_by_stencil{}", static_cast<uint32_t>(format));

    copy_stencil_by_render_shaders[format] =
        compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                name.c_str(),
                                R"(#version 450
	precision highp int;
	precision highp float;
	layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_stencil;
	layout (push_constant) uniform mask_data {{ uint current_bit; }};
	void main() {{
		uint stencil_value = subpassLoad(in_stencil).r;
		if ((stencil_value & (0x1 << current_bit)) == 0) {
			discard;
		}
	}})");
    created_name[0] = name;
  }

  return copy_stencil_by_render_shaders[format];
}

const std::vector<uint32_t>& shader_manager::get_prime_by_rendering_color_shader(VkFormat format, std::string* created_name) {
  if (!prime_by_rendering_color_shaders.count(format)) {
    auto name = std::format("render_by_color{}", static_cast<uint32_t>(format));
    switch (format) {
      case VK_FORMAT_R8_UINT:
      case VK_FORMAT_R8G8_UINT:
      case VK_FORMAT_R8G8B8_UINT:
      case VK_FORMAT_R8G8B8A8_UINT:
      case VK_FORMAT_B8G8R8_UINT:
      case VK_FORMAT_B8G8R8A8_UINT:
      case VK_FORMAT_R16_UINT:
      case VK_FORMAT_R16G16_UINT:
      case VK_FORMAT_R16G16B16_UINT:
      case VK_FORMAT_R16G16B16A16_UINT:
      case VK_FORMAT_R32_UINT:
      case VK_FORMAT_R32G32_UINT:
      case VK_FORMAT_R32G32B32_UINT:
      case VK_FORMAT_R32G32B32A32_UINT:
      case VK_FORMAT_A8B8G8R8_UINT_PACK32:
      case VK_FORMAT_A2R10G10B10_UINT_PACK32:
      case VK_FORMAT_A2B10G10R10_UINT_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
layout(location = 0) out uvec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r;
	out_color.g = subpassLoad(in_color).g;
	out_color.b = subpassLoad(in_color).b;
	out_color.a = subpassLoad(in_color).a;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R8_SINT:
      case VK_FORMAT_R8G8_SINT:
      case VK_FORMAT_R8G8B8_SINT:
      case VK_FORMAT_R8G8B8A8_SINT:
      case VK_FORMAT_B8G8R8_SINT:
      case VK_FORMAT_B8G8R8A8_SINT:
      case VK_FORMAT_R16_SINT:
      case VK_FORMAT_R16G16_SINT:
      case VK_FORMAT_R16G16B16_SINT:
      case VK_FORMAT_R16G16B16A16_SINT:
      case VK_FORMAT_R32_SINT:
      case VK_FORMAT_R32G32_SINT:
      case VK_FORMAT_R32G32B32_SINT:
      case VK_FORMAT_R32G32B32A32_SINT:
      case VK_FORMAT_A8B8G8R8_SINT_PACK32:
      case VK_FORMAT_A2R10G10B10_SINT_PACK32:
      case VK_FORMAT_A2B10G10R10_SINT_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
layout(location = 0) out ivec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = int(subpassLoad(in_color).r);
	out_color.g = int(subpassLoad(in_color).g);
	out_color.b = int(subpassLoad(in_color).b);
	out_color.a = int(subpassLoad(in_color).a);
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R8_UNORM:
      case VK_FORMAT_R8G8_UNORM:
      case VK_FORMAT_R8G8B8_UNORM:
      case VK_FORMAT_R8G8B8A8_UNORM:
      case VK_FORMAT_B8G8R8_UNORM:
      case VK_FORMAT_B8G8R8A8_UNORM:
      case VK_FORMAT_R8_SRGB:
      case VK_FORMAT_R8G8_SRGB:
      case VK_FORMAT_R8G8B8_SRGB:
      case VK_FORMAT_R8G8B8A8_SRGB:
      case VK_FORMAT_B8G8R8_SRGB:
      case VK_FORMAT_B8G8R8A8_SRGB:
      case VK_FORMAT_A8B8G8R8_UNORM_PACK32:
      case VK_FORMAT_A8B8G8R8_SRGB_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r/255.0;
	out_color.g = subpassLoad(in_color).g/255.0;
	out_color.b = subpassLoad(in_color).b/255.0;
	out_color.a = subpassLoad(in_color).a/255.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R16_UNORM:
      case VK_FORMAT_R16G16_UNORM:
      case VK_FORMAT_R16G16B16_UNORM:
      case VK_FORMAT_R16G16B16A16_UNORM:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r/65535.0;
	out_color.g = subpassLoad(in_color).g/65535.0;
	out_color.b = subpassLoad(in_color).b/65535.0;
	out_color.a = subpassLoad(in_color).a/65535.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R4G4_UNORM_PACK8:
      case VK_FORMAT_R4G4B4A4_UNORM_PACK16:
      case VK_FORMAT_B4G4R4A4_UNORM_PACK16:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r/15.0;
	out_color.g = subpassLoad(in_color).g/15.0;
	out_color.b = subpassLoad(in_color).b/15.0;
	out_color.a = subpassLoad(in_color).a/15.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R5G6B5_UNORM_PACK16:
      case VK_FORMAT_B5G6R5_UNORM_PACK16:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r/31.0;
	out_color.g = subpassLoad(in_color).g/63.0;
	out_color.b = subpassLoad(in_color).b/31.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R5G5B5A1_UNORM_PACK16:
      case VK_FORMAT_B5G5R5A1_UNORM_PACK16:
      case VK_FORMAT_A1R5G5B5_UNORM_PACK16:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r/31.0;
	out_color.g = subpassLoad(in_color).g/31.0;
	out_color.b = subpassLoad(in_color).b/31.0;
	out_color.a = subpassLoad(in_color).a/1.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_A2R10G10B10_UNORM_PACK32:
      case VK_FORMAT_A2B10G10R10_UNORM_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = subpassLoad(in_color).r/1023.0;
	out_color.g = subpassLoad(in_color).g/1023.0;
	out_color.b = subpassLoad(in_color).b/1023.0;
	out_color.a = subpassLoad(in_color).a/3.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R8_SNORM:
      case VK_FORMAT_R8G8_SNORM:
      case VK_FORMAT_R8G8B8_SNORM:
      case VK_FORMAT_R8G8B8A8_SNORM:
      case VK_FORMAT_B8G8R8_SNORM:
      case VK_FORMAT_B8G8R8A8_SNORM:
      case VK_FORMAT_A8B8G8R8_SNORM_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
float snorm(in uint u, in float d) {{
	return (int(u) * 2.0  + 1.0) / d;
}}
void main() {{
	out_color.r = snorm(subpassLoad(in_color).r, 255.0);
	out_color.g = snorm(subpassLoad(in_color).g, 255.0);
	out_color.b = snorm(subpassLoad(in_color).b, 255.0);
	out_color.a = snorm(subpassLoad(in_color).a, 255.0);
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R16_SNORM:
      case VK_FORMAT_R16G16_SNORM:
      case VK_FORMAT_R16G16B16_SNORM:
      case VK_FORMAT_R16G16B16A16_SNORM:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
float snorm(in uint u, in float d) {{
	return (int(u) * 2.0  + 1.0) / d;
}}
void main() {{
	out_color.r = snorm(subpassLoad(in_color).r, 65535.0);
	out_color.g = snorm(subpassLoad(in_color).g, 65535.0);
	out_color.b = snorm(subpassLoad(in_color).b, 65535.0);
	out_color.a = snorm(subpassLoad(in_color).a, 65535.0);
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_A2R10G10B10_SNORM_PACK32:
      case VK_FORMAT_A2B10G10R10_SNORM_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
float snorm(in uint u, in float d) {{
	return (int(u) * 2.0  + 1.0) / d;
}}
void main() {{
	out_color.r = snorm(subpassLoad(in_color).r, 1023.0);
	out_color.g = snorm(subpassLoad(in_color).g, 1023.0);
	out_color.b = snorm(subpassLoad(in_color).b, 1023.0);
	out_color.a = snorm(subpassLoad(in_color).a, 1.0);
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_R16_SFLOAT:
      case VK_FORMAT_R16G16_SFLOAT:
      case VK_FORMAT_R16G16B16_SFLOAT:
      case VK_FORMAT_R16G16B16A16_SFLOAT:
      case VK_FORMAT_R32_SFLOAT:
      case VK_FORMAT_R32G32_SFLOAT:
      case VK_FORMAT_R32G32B32_SFLOAT:
      case VK_FORMAT_R32G32B32A32_SFLOAT:
      case VK_FORMAT_B10G11R11_UFLOAT_PACK32:
      case VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
        prime_by_rendering_color_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {{
	out_color.r = uintBitsToFloat(subpassLoad(in_color).r);
	out_color.g = uintBitsToFloat(subpassLoad(in_color).g);
	out_color.b = uintBitsToFloat(subpassLoad(in_color).b);
	out_color.a = uintBitsToFloat(subpassLoad(in_color).a);
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      default:
        GAPID2_ERROR("Unsupported format for prime_by_rendering");
    }
  }

  return prime_by_rendering_color_shaders[format];
}

const std::vector<uint32_t>& shader_manager::get_prime_by_rendering_depth_shader(VkFormat format, std::string* created_name) {
  if (!prime_by_rendering_depth_shaders.count(format)) {
    auto name = std::format("render_by_depth{}", static_cast<uint32_t>(format));
    switch (format) {
      case VK_FORMAT_D16_UNORM:
      case VK_FORMAT_D16_UNORM_S8_UINT:
        prime_by_rendering_depth_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {{
	gl_FragDepth = subpassLoad(in_depth).r / 65535.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_D24_UNORM_S8_UINT:
      case VK_FORMAT_X8_D24_UNORM_PACK32:
        prime_by_rendering_depth_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {{
	gl_FragDepth = (subpassLoad(in_depth).r & 0x00FFFFFF) / 16777215.0;
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      case VK_FORMAT_D32_SFLOAT:
      case VK_FORMAT_D32_SFLOAT_S8_UINT:
        prime_by_rendering_depth_shaders[format] =
            compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                    name.c_str(),
                                    std::format(R"(#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {{
	gl_FragDepth = uintBitsToFloat(subpassLoad(in_depth).r);
}})")
                                        .c_str());
        created_name[0] = name;
        break;
      default:
        GAPID2_ERROR("Unsupported format for prime_by_rendering_depth");
    }
  }

  return prime_by_rendering_depth_shaders[format];
}

const std::vector<uint32_t>& shader_manager::get_prime_by_rendering_stencil_shader(std::string* created_name) {
  if (prime_by_rendering_stencil_shader_spirv.empty()) {
    prime_by_rendering_stencil_shader_spirv =
        compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_FRAGMENT,
                                "prime_by_render_stencil",
                                std::format(R"(#version 450
precision highp int;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_stencil;
layout (push_constant) uniform mask_data {{ uint current_bit; }};
void main() {{
  uint stencil_value = subpassLoad(in_stencil).r;
  if ((stencil_value & (0x1 << current_bit)) == 0) {{
    discard;
  }}
}})")
                                    .c_str());
    created_name[0] = "prime_by_render_stencil";
  }
  return prime_by_rendering_stencil_shader_spirv;
}

const std::vector<uint32_t>& shader_manager::get_prime_by_compute_copy_shader(VkFormat format, VkImageAspectFlagBits aspect, VkImageType type, std::string* created_name) {
  GAPID2_ASSERT(aspect == VK_IMAGE_ASPECT_COLOR_BIT, "Invalid aspect for compute copy");
  compute_copy_key cck{format, format, aspect, aspect, type};

  if (!prime_by_compute_copy_shaders.count(cck)) {
    auto name = std::format("render_by_compute_copy{}", static_cast<uint32_t>(format));
    auto fmtStr = storage_image_format(format);
    GAPID2_ASSERT(fmtStr, "Unable to get format str");
    auto pos = storage_image_position(type);
    GAPID2_ASSERT(pos, "Unable to get position string");
    auto unit = storage_image_unit(format);
    GAPID2_ASSERT(unit, "Unable to determine unit");
    auto imgTypeStr = storage_image_type(type);
    prime_by_compute_copy_shaders[cck] = compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_COMPUTE,
                                                                 name.c_str(),
                                                                 std::format(R"(#version 450
	precision highp int;
	layout (local_size_x = 1, local_size_y = 1, local_size_z = 1) in;
	layout ({}, set = 0, binding = {}) uniform {}{} output_img;
	layout ({}, set = 0, binding = {}) uniform {}{} input_img;
	layout (push_constant) uniform metadata {{
		uint offset_x;
		uint offset_y;
		uint offset_z;
		// Reserved for handling image formats wider than 32 bit per channel
		uint input_img_index;
	}};
	void main() {{
		int x = int(gl_GlobalInvocationID.x + offset_x);
		int y = int(gl_GlobalInvocationID.y + offset_y);
		int z = int(gl_GlobalInvocationID.z + offset_z);
		{}
		imageStore(output_img, pos, imageLoad(input_img, pos));
	}})",
                                                                             fmtStr, k_store_output_image_binding, unit, imgTypeStr,
                                                                             fmtStr, k_store_input_image_binding, unit, imgTypeStr,
                                                                             pos)
                                                                     .c_str());
    created_name[0] = name;
  }

  return prime_by_compute_copy_shaders[cck];
}

const std::vector<uint32_t>& shader_manager::get_prime_by_compute_store_shader(VkFormat output_format, VkImageAspectFlagBits output_aspect, VkFormat input_format, VkImageAspectFlagBits input_aspect, VkImageType type, std::string* created_name) {
  GAPID2_ASSERT(input_aspect == VK_IMAGE_ASPECT_COLOR_BIT, "Invalid aspect for compute copy");
  GAPID2_ASSERT(output_aspect == VK_IMAGE_ASPECT_COLOR_BIT, "Invalid aspect for compute copy");

  compute_copy_key cck{input_format, output_format, input_aspect, output_aspect, type};
  if (!prime_by_compute_store_shaders.count(cck)) {
    auto name = std::format("render_by_compute_store{}-{}", static_cast<uint32_t>(input_format), static_cast<uint32_t>(output_format));
    std::string color = [&]() -> std::string {
      const char* inputG = storage_image_unit(input_format);
      if (input_format == output_format) {
        return std::string(inputG) + "vec4 color = imageLoad(input_img, pos);";
      }
      if (input_format == VK_FORMAT_R32G32B32A32_UINT) {
        switch (output_format) {
          case VK_FORMAT_R8_UINT:
          case VK_FORMAT_R8G8_UINT:
          // VK_FORMAT_R8G8B8_UINT:
          case VK_FORMAT_R8G8B8A8_UINT:
          // VK_FORMAT_B8G8R8_UINT:
          case VK_FORMAT_B8G8R8A8_UINT:
          case VK_FORMAT_R16_UINT:
          case VK_FORMAT_R16G16_UINT:
          // VK_FORMAT_R16G16B16_UINT:
          case VK_FORMAT_R16G16B16A16_UINT:
          case VK_FORMAT_R32_UINT:
          case VK_FORMAT_R32G32_UINT:
          // VK_FORMAT_R32G32B32_UINT:
          case VK_FORMAT_R32G32B32A32_UINT:
          case VK_FORMAT_A8B8G8R8_UINT_PACK32:
          case VK_FORMAT_A2R10G10B10_UINT_PACK32:
          case VK_FORMAT_A2B10G10R10_UINT_PACK32:
            return "uvec4 color = imageLoad(input_img, pos);";
          case VK_FORMAT_R8_SINT:
          case VK_FORMAT_R8G8_SINT:
          // VK_FORMAT_R8G8B8_SINT:
          case VK_FORMAT_R8G8B8A8_SINT:
          // VK_FORMAT_B8G8R8_SINT:
          case VK_FORMAT_B8G8R8A8_SINT:
          case VK_FORMAT_R16_SINT:
          case VK_FORMAT_R16G16_SINT:
          // VK_FORMAT_R16G16B16_SINT:
          case VK_FORMAT_R16G16B16A16_SINT:
          case VK_FORMAT_R32_SINT:
          case VK_FORMAT_R32G32_SINT:
          // VK_FORMAT_R32G32B32_SINT:
          case VK_FORMAT_R32G32B32A32_SINT:
          case VK_FORMAT_A8B8G8R8_SINT_PACK32:
          case VK_FORMAT_A2R10G10B10_SINT_PACK32:
          case VK_FORMAT_A2B10G10R10_SINT_PACK32:
            return R"(uvec4 input_color = imageLoad(input_img, pos);
				int r = int(input_color.r);
				int g = int(input_color.g);
				int b = int(input_color.b);
				int a = int(input_color.a);
				ivec4 color = ivec4(r, g, b, a);
				)";
          case VK_FORMAT_R8_UNORM:
          case VK_FORMAT_R8G8_UNORM:
          // case VK_FORMAT_R8G8B8_UNORM:
          case VK_FORMAT_R8G8B8A8_UNORM:
          case VK_FORMAT_B8G8R8_UNORM:
          case VK_FORMAT_B8G8R8A8_UNORM:
          case VK_FORMAT_R8_SRGB:
          case VK_FORMAT_R8G8_SRGB:
          // case VK_FORMAT_R8G8B8_SRGB:
          case VK_FORMAT_R8G8B8A8_SRGB:
          // case VK_FORMAT_B8G8R8_SRGB:
          case VK_FORMAT_B8G8R8A8_SRGB:
          case VK_FORMAT_A8B8G8R8_UNORM_PACK32:
          case VK_FORMAT_A8B8G8R8_SRGB_PACK32:
            return "vec4 color = imageLoad(input_img, pos).rgba/vec4(255.0, 255.0, 255.0, 255.0);";
          case VK_FORMAT_R16_UNORM:
          case VK_FORMAT_R16G16_UNORM:
          // case VK_FORMAT_R16G16B16_UNORM:
          case VK_FORMAT_R16G16B16A16_UNORM:
            return "vec4 color = imageLoad(input_img, pos).rgba/vec4(65535.0, 65535.0, 65535.0, 65535.0);";
          case VK_FORMAT_A2R10G10B10_UNORM_PACK32:
          case VK_FORMAT_A2B10G10R10_UNORM_PACK32:
            return "vec4 color = imageLoad(input_img, pos).rgba/vec4(1023.0, 1023.0, 1023.0, 3.0);";
          case VK_FORMAT_R8_SNORM:
          case VK_FORMAT_R8G8_SNORM:
          // case VK_FORMAT_R8G8B8_SNORM:
          case VK_FORMAT_R8G8B8A8_SNORM:
          // case VK_FORMAT_B8G8R8_SNORM:
          case VK_FORMAT_B8G8R8A8_SNORM:
          case VK_FORMAT_A8B8G8R8_SNORM_PACK32:
            return R"(float r = (int(imageLoad(input_img, pos).r) * 2.0 + 1.0) / 255.0;
					float g = (int(imageLoad(input_img, pos).g) * 2.0 + 1.0) / 255.0;
					float b = (int(imageLoad(input_img, pos).b) * 2.0 + 1.0) / 255.0;
					float a = (int(imageLoad(input_img, pos).a) * 2.0 + 1.0) / 255.0;
					vec4 color = vec4(r, g, b, a);)";

          case VK_FORMAT_R16_SNORM:
          case VK_FORMAT_R16G16_SNORM:
          // case VK_FORMAT_R16G16B16_SNORM:
          case VK_FORMAT_R16G16B16A16_SNORM:
            return R"(float r = (int(imageLoad(input_img, pos).r) * 2.0 + 1.0) / 65535.0;
					float g = (int(imageLoad(input_img, pos).g) * 2.0 + 1.0) / 65535.0;
					float b = (int(imageLoad(input_img, pos).b) * 2.0 + 1.0) / 65535.0;
					float a = (int(imageLoad(input_img, pos).a) * 2.0 + 1.0) / 65535.0;
					vec4 color = vec4(r, g, b, a);)";
            // case VK_FORMAT_A2R10G10B10_SNORM_PACK32,
            // 	VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32:
          case VK_FORMAT_R16_SFLOAT:
          case VK_FORMAT_R16G16_SFLOAT:
          // case VK_FORMAT_R16G16B16_SFLOAT:
          case VK_FORMAT_R16G16B16A16_SFLOAT:
          case VK_FORMAT_R32_SFLOAT:
          case VK_FORMAT_R32G32_SFLOAT:
          // case VK_FORMAT_R32G32B32_SFLOAT:
          case VK_FORMAT_R32G32B32A32_SFLOAT:
          // case VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
          case VK_FORMAT_B10G11R11_UFLOAT_PACK32:
            return R"(float r = uintBitsToFloat(imageLoad(input_img, pos).r);
					float g = uintBitsToFloat(imageLoad(input_img, pos).g);
					float b = uintBitsToFloat(imageLoad(input_img, pos).b);
					float a = uintBitsToFloat(imageLoad(input_img, pos).a);
					vec4 color = vec4(r, g, b, a);)";
        }
      }
      GAPID2_ERROR("Unsupported format");
      return "";
    }();

    const char* output_fmt_str = storage_image_format(output_format);
    const char* input_fmt_str = storage_image_format(input_format);
    const char* output_g = storage_image_unit(output_format);
    const char* input_g = storage_image_unit(input_format);
    const char* image_type_str = storage_image_type(type);
    const char* pos = storage_image_position(type);

    prime_by_compute_copy_shaders[cck] = compile_shader_to_spirv(glslang_stage_t::GLSLANG_STAGE_COMPUTE,
                                                                 name.c_str(),
                                                                 std::format(R"(#version 450
	precision highp int;
	layout (local_size_x = 1, local_size_y = 1, local_size_z = 1) in;
	layout ({}, set = 0, binding = {}) uniform {}{} output_img;
	layout ({}, set = 0, binding = {}) uniform {}{} input_img;
	layout (push_constant) uniform metadata {{
		uint offset_x;
		uint offset_y;
		uint offset_z;
		// Reserved for handling image formats wider than 32 bit per channel
		uint input_img_index;
	}};
	void main() {{
		int x = int(gl_GlobalInvocationID.x + offset_x);
		int y = int(gl_GlobalInvocationID.y + offset_y);
		int z = int(gl_GlobalInvocationID.z + offset_z);
		{}
		{}
		imageStore(output_img, pos, color);
	}})",
                                                                             output_fmt_str, k_store_output_image_binding, output_g, image_type_str,
                                                                             input_fmt_str, k_store_input_image_binding, input_g, image_type_str,
                                                                             pos, color)
                                                                     .c_str());
    created_name[0] = name;
  }

  return prime_by_compute_store_shaders[cck];
}
}  // namespace gapid2