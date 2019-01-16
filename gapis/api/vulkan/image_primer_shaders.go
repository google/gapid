// Copyright (C) 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vulkan

import (
	"fmt"

	"github.com/google/gapid/gapis/shadertools"
)

// ipVertexShaderSpirv returns a pass-through vertex shader in SPIR-V words.
func ipVertexShaderSpirv() ([]uint32, error) {
	return shadertools.CompileGlsl(
		`#version 450
layout(location = 0) in vec3 position;
void main() {
	gl_Position = vec4(position, 1.0);
}`,
		shadertools.CompileOptions{
			ShaderType: shadertools.TypeVertex,
			ClientType: shadertools.Vulkan,
		})
}

// ipFragmentShaderSpirv returns the fragment shader to be used for priming
// image data through rendering in SPIR-V words.
func ipFragmentShaderSpirv(vkFmt VkFormat, aspect VkImageAspectFlagBits) ([]uint32, error) {
	switch aspect {
	// Render color data
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		switch vkFmt {
		case VkFormat_VK_FORMAT_R8_UINT,
			VkFormat_VK_FORMAT_R8G8_UINT,
			VkFormat_VK_FORMAT_R8G8B8_UINT,
			VkFormat_VK_FORMAT_R8G8B8A8_UINT,
			VkFormat_VK_FORMAT_B8G8R8_UINT,
			VkFormat_VK_FORMAT_B8G8R8A8_UINT,
			VkFormat_VK_FORMAT_R16_UINT,
			VkFormat_VK_FORMAT_R16G16_UINT,
			VkFormat_VK_FORMAT_R16G16B16_UINT,
			VkFormat_VK_FORMAT_R16G16B16A16_UINT,
			VkFormat_VK_FORMAT_R32_UINT,
			VkFormat_VK_FORMAT_R32G32_UINT,
			VkFormat_VK_FORMAT_R32G32B32_UINT,
			VkFormat_VK_FORMAT_R32G32B32A32_UINT,
			VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
			VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
layout(location = 0) out uvec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r;
	out_color.g = subpassLoad(in_color).g;
	out_color.b = subpassLoad(in_color).b;
	out_color.a = subpassLoad(in_color).a;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R8_SINT,
			VkFormat_VK_FORMAT_R8G8_SINT,
			VkFormat_VK_FORMAT_R8G8B8_SINT,
			VkFormat_VK_FORMAT_R8G8B8A8_SINT,
			VkFormat_VK_FORMAT_B8G8R8_SINT,
			VkFormat_VK_FORMAT_B8G8R8A8_SINT,
			VkFormat_VK_FORMAT_R16_SINT,
			VkFormat_VK_FORMAT_R16G16_SINT,
			VkFormat_VK_FORMAT_R16G16B16_SINT,
			VkFormat_VK_FORMAT_R16G16B16A16_SINT,
			VkFormat_VK_FORMAT_R32_SINT,
			VkFormat_VK_FORMAT_R32G32_SINT,
			VkFormat_VK_FORMAT_R32G32B32_SINT,
			VkFormat_VK_FORMAT_R32G32B32A32_SINT,
			VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
			VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
layout(location = 0) out ivec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = int(subpassLoad(in_color).r);
	out_color.g = int(subpassLoad(in_color).g);
	out_color.b = int(subpassLoad(in_color).b);
	out_color.a = int(subpassLoad(in_color).a);
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R8_UNORM,
			VkFormat_VK_FORMAT_R8G8_UNORM,
			VkFormat_VK_FORMAT_R8G8B8_UNORM,
			VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
			VkFormat_VK_FORMAT_B8G8R8_UNORM,
			VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
			VkFormat_VK_FORMAT_R8_SRGB,
			VkFormat_VK_FORMAT_R8G8_SRGB,
			VkFormat_VK_FORMAT_R8G8B8_SRGB,
			VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
			VkFormat_VK_FORMAT_B8G8R8_SRGB,
			VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
			VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
			VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r/255.0;
	out_color.g = subpassLoad(in_color).g/255.0;
	out_color.b = subpassLoad(in_color).b/255.0;
	out_color.a = subpassLoad(in_color).a/255.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R16_UNORM,
			VkFormat_VK_FORMAT_R16G16_UNORM,
			VkFormat_VK_FORMAT_R16G16B16_UNORM,
			VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r/65535.0;
	out_color.g = subpassLoad(in_color).g/65535.0;
	out_color.b = subpassLoad(in_color).b/65535.0;
	out_color.a = subpassLoad(in_color).a/65535.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R4G4_UNORM_PACK8,
			VkFormat_VK_FORMAT_R4G4B4A4_UNORM_PACK16,
			VkFormat_VK_FORMAT_B4G4R4A4_UNORM_PACK16:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r/15.0;
	out_color.g = subpassLoad(in_color).g/15.0;
	out_color.b = subpassLoad(in_color).b/15.0;
	out_color.a = subpassLoad(in_color).a/15.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R5G6B5_UNORM_PACK16,
			VkFormat_VK_FORMAT_B5G6R5_UNORM_PACK16:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r/31.0;
	out_color.g = subpassLoad(in_color).g/63.0;
	out_color.b = subpassLoad(in_color).b/31.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R5G5B5A1_UNORM_PACK16,
			VkFormat_VK_FORMAT_B5G5R5A1_UNORM_PACK16,
			VkFormat_VK_FORMAT_A1R5G5B5_UNORM_PACK16:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r/31.0;
	out_color.g = subpassLoad(in_color).g/31.0;
	out_color.b = subpassLoad(in_color).b/31.0;
	out_color.a = subpassLoad(in_color).a/1.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = subpassLoad(in_color).r/1023.0;
	out_color.g = subpassLoad(in_color).g/1023.0;
	out_color.b = subpassLoad(in_color).b/1023.0;
	out_color.a = subpassLoad(in_color).a/3.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R8_SNORM,
			VkFormat_VK_FORMAT_R8G8_SNORM,
			VkFormat_VK_FORMAT_R8G8B8_SNORM,
			VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
			VkFormat_VK_FORMAT_B8G8R8_SNORM,
			VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
			VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
float snorm(in uint u, in float d) {
	return (int(u) * 2.0  + 1.0) / d;
}
void main() {
	out_color.r = snorm(subpassLoad(in_color).r, 255.0);
	out_color.g = snorm(subpassLoad(in_color).g, 255.0);
	out_color.b = snorm(subpassLoad(in_color).b, 255.0);
	out_color.a = snorm(subpassLoad(in_color).a, 255.0);
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R16_SNORM,
			VkFormat_VK_FORMAT_R16G16_SNORM,
			VkFormat_VK_FORMAT_R16G16B16_SNORM,
			VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
float snorm(in uint u, in float d) {
	return (int(u) * 2.0  + 1.0) / d;
}
void main() {
	out_color.r = snorm(subpassLoad(in_color).r, 65535.0);
	out_color.g = snorm(subpassLoad(in_color).g, 65535.0);
	out_color.b = snorm(subpassLoad(in_color).b, 65535.0);
	out_color.a = snorm(subpassLoad(in_color).a, 65535.0);
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
float snorm(in uint u, in float d) {
	return (int(u) * 2.0  + 1.0) / d;
}
void main() {
	out_color.r = snorm(subpassLoad(in_color).r, 1023.0);
	out_color.g = snorm(subpassLoad(in_color).g, 1023.0);
	out_color.b = snorm(subpassLoad(in_color).b, 1023.0);
	out_color.a = snorm(subpassLoad(in_color).a, 1.0);
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_R16_SFLOAT,
			VkFormat_VK_FORMAT_R16G16_SFLOAT,
			VkFormat_VK_FORMAT_R16G16B16_SFLOAT,
			VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT,
			VkFormat_VK_FORMAT_R32_SFLOAT,
			VkFormat_VK_FORMAT_R32G32_SFLOAT,
			VkFormat_VK_FORMAT_R32G32B32_SFLOAT,
			VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT,
			VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32,
			VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
layout(location = 0) out vec4 out_color;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_color;
void main() {
	out_color.r = uintBitsToFloat(subpassLoad(in_color).r);
	out_color.g = uintBitsToFloat(subpassLoad(in_color).g);
	out_color.b = uintBitsToFloat(subpassLoad(in_color).b);
	out_color.a = uintBitsToFloat(subpassLoad(in_color).a);
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		default:
			return []uint32{}, fmt.Errorf("%v is not supported", vkFmt)
		}

	// Render depth data
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT:
		switch vkFmt {
		case VkFormat_VK_FORMAT_D16_UNORM,
			VkFormat_VK_FORMAT_D16_UNORM_S8_UINT:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {
	gl_FragDepth = subpassLoad(in_depth).r / 65535.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_D24_UNORM_S8_UINT,
			VkFormat_VK_FORMAT_X8_D24_UNORM_PACK32:
			return shadertools.CompileGlsl(
				// When doing a buffer-image copy for these
				// formats, the 8 MSBs of the 32 bits are
				// undefined, so in case those values came from
				// such a source, mask them out.
				`#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {
	gl_FragDepth = (subpassLoad(in_depth).r & 0x00FFFFFF) / 16777215.0;
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		case VkFormat_VK_FORMAT_D32_SFLOAT,
			VkFormat_VK_FORMAT_D32_SFLOAT_S8_UINT:
			return shadertools.CompileGlsl(
				`#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {
	gl_FragDepth = uintBitsToFloat(subpassLoad(in_depth).r);
}`,
				shadertools.CompileOptions{
					ShaderType: shadertools.TypeFragment,
					ClientType: shadertools.Vulkan,
				})

		default:
			return []uint32{}, fmt.Errorf("%v is not supported", vkFmt)
		}

	// Render stencil data
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		return shadertools.CompileGlsl(
			`#version 450
precision highp int;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_stencil;
layout(binding = 1, set = 0) uniform mask_data { uint current_bit;};
void main() {
  uint stencil_value = subpassLoad(in_stencil).r;
  if ((stencil_value & (0x1 << current_bit)) == 0) {
    discard;
  }
}`,
			shadertools.CompileOptions{
				ShaderType: shadertools.TypeFragment,
				ClientType: shadertools.Vulkan,
			})

	// other aspect data
	default:
		return []uint32{}, fmt.Errorf("%v is not supported", aspect)
	}
}

// ipComputeShaderSpirv returns the compute shader to be used for priming image
// data through imageStore operation.
func ipComputeShaderSpirv(vkFmt VkFormat, aspect VkImageAspectFlagBits, imageType VkImageType) ([]uint32, error) {
	// Determine the image format token in shader
	// Ref: https://www.khronos.org/opengl/wiki/Layout_Qualifier_(GLSL)#Image_formats
	var imgFmtStr string
	switch vkFmt {
	// uint formats
	case VkFormat_VK_FORMAT_R8_UINT:
		imgFmtStr = "r8ui"
	case VkFormat_VK_FORMAT_R16_UINT:
		imgFmtStr = "r16ui"
	case VkFormat_VK_FORMAT_R32_UINT:
		imgFmtStr = "r32ui"

	case VkFormat_VK_FORMAT_R8G8_UINT:
		imgFmtStr = "rg8ui"
	case VkFormat_VK_FORMAT_R16G16_UINT:
		imgFmtStr = "rg16ui"
	case VkFormat_VK_FORMAT_R32G32_UINT:
		imgFmtStr = "rg32ui"

	case VkFormat_VK_FORMAT_R8G8B8A8_UINT,
		VkFormat_VK_FORMAT_B8G8R8A8_UINT,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32:
		imgFmtStr = "rgba8ui"
	case VkFormat_VK_FORMAT_R16G16B16A16_UINT:
		imgFmtStr = "rgba16ui"
	case VkFormat_VK_FORMAT_R32G32B32A32_UINT:
		imgFmtStr = "rgba32ui"

	case VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
		imgFmtStr = "rgb10_a2ui"

	// sint formats
	case VkFormat_VK_FORMAT_R8_SINT:
		imgFmtStr = "r8i"
	case VkFormat_VK_FORMAT_R16_SINT:
		imgFmtStr = "r16i"
	case VkFormat_VK_FORMAT_R32_SINT:
		imgFmtStr = "r32i"

	case VkFormat_VK_FORMAT_R8G8_SINT:
		imgFmtStr = "rg8i"
	case VkFormat_VK_FORMAT_R16G16_SINT:
		imgFmtStr = "rg16i"
	case VkFormat_VK_FORMAT_R32G32_SINT:
		imgFmtStr = "rg32i"

	case VkFormat_VK_FORMAT_R8G8B8A8_SINT,
		VkFormat_VK_FORMAT_B8G8R8A8_SINT,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32:
		imgFmtStr = "rgba8i"
	case VkFormat_VK_FORMAT_R16G16B16A16_SINT:
		imgFmtStr = "rgba16i"
	case VkFormat_VK_FORMAT_R32G32B32A32_SINT:
		imgFmtStr = "rgba32i"

	// unorm formats
	case VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R8_SRGB:
		imgFmtStr = "r8"
	case VkFormat_VK_FORMAT_R16_UNORM:
		imgFmtStr = "r16"

	case VkFormat_VK_FORMAT_R8G8_UNORM,
		VkFormat_VK_FORMAT_R8G8_SRGB:
		imgFmtStr = "rg8"
	case VkFormat_VK_FORMAT_R16G16_UNORM:
		imgFmtStr = "rg16"

	case VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
		VkFormat_VK_FORMAT_B8G8R8A8_SRGB:
		imgFmtStr = "rgba8"

	case VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
		imgFmtStr = "rgba16"

	case VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
		imgFmtStr = "rgba8"

	case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
		imgFmtStr = "rgb10_a2"

	// snorm formats
	case VkFormat_VK_FORMAT_R8_SNORM:
		imgFmtStr = "r8_snorm"
	case VkFormat_VK_FORMAT_R16_SNORM:
		imgFmtStr = "r16_snorm"

	case VkFormat_VK_FORMAT_R8G8_SNORM:
		imgFmtStr = "rg8_snorm"
	case VkFormat_VK_FORMAT_R16G16_SNORM:
		imgFmtStr = "rg16_snorm"

	case VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
		imgFmtStr = "rgba8_snorm"
	case VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
		imgFmtStr = "rgba16_snorm"

	// float formats
	case VkFormat_VK_FORMAT_R16_SFLOAT:
		imgFmtStr = "r16f"
	case VkFormat_VK_FORMAT_R32_SFLOAT:
		imgFmtStr = "r32f"

	case VkFormat_VK_FORMAT_R16G16_SFLOAT:
		imgFmtStr = "rg16f"
	case VkFormat_VK_FORMAT_R32G32_SFLOAT:
		imgFmtStr = "rg32f"

	case VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT:
		imgFmtStr = "rgba16f"
	case VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT:
		imgFmtStr = "rgba32f"

	case VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32:
		imgFmtStr = "r11f_g11f_b10f"

	default:
		return []uint32{}, fmt.Errorf("%v does not support imageStore", vkFmt)
	}

	var imgTypeG string
	switch vkFmt {
	// uint formats
	case VkFormat_VK_FORMAT_R8_UINT,
		VkFormat_VK_FORMAT_R16_UINT,
		VkFormat_VK_FORMAT_R32_UINT,
		VkFormat_VK_FORMAT_R8G8_UINT,
		VkFormat_VK_FORMAT_R16G16_UINT,
		VkFormat_VK_FORMAT_R32G32_UINT,
		VkFormat_VK_FORMAT_R8G8B8A8_UINT,
		VkFormat_VK_FORMAT_B8G8R8A8_UINT,
		VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_UINT,
		VkFormat_VK_FORMAT_R32G32B32A32_UINT,
		VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
		imgTypeG = "u"

	// sint formats
	case VkFormat_VK_FORMAT_R8_SINT,
		VkFormat_VK_FORMAT_R16_SINT,
		VkFormat_VK_FORMAT_R32_SINT,
		VkFormat_VK_FORMAT_R8G8_SINT,
		VkFormat_VK_FORMAT_R16G16_SINT,
		VkFormat_VK_FORMAT_R32G32_SINT,
		VkFormat_VK_FORMAT_R8G8B8A8_SINT,
		VkFormat_VK_FORMAT_B8G8R8A8_SINT,
		VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SINT,
		VkFormat_VK_FORMAT_R32G32B32A32_SINT:
		imgTypeG = "i"

	// unorm formats
	case VkFormat_VK_FORMAT_R8_UNORM,
		VkFormat_VK_FORMAT_R8_SRGB,
		VkFormat_VK_FORMAT_R16_UNORM,
		VkFormat_VK_FORMAT_R8G8_UNORM,
		VkFormat_VK_FORMAT_R8G8_SRGB,
		VkFormat_VK_FORMAT_R16G16_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
		VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
		VkFormat_VK_FORMAT_R16G16B16A16_UNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
		VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32,
		VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
		VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32,
		// snorm formats
		VkFormat_VK_FORMAT_R8_SNORM,
		VkFormat_VK_FORMAT_R16_SNORM,
		VkFormat_VK_FORMAT_R8G8_SNORM,
		VkFormat_VK_FORMAT_R16G16_SNORM,
		VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
		VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
		VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32,
		VkFormat_VK_FORMAT_R16G16B16A16_SNORM,
		// float formats
		VkFormat_VK_FORMAT_R16_SFLOAT,
		VkFormat_VK_FORMAT_R32_SFLOAT,
		VkFormat_VK_FORMAT_R16G16_SFLOAT,
		VkFormat_VK_FORMAT_R32G32_SFLOAT,
		VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT,
		VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT,
		VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32:
		imgTypeG = ""
	}

	// image1D/image2D/image3D
	var imgTypeStr string
	switch imageType {
	case VkImageType_VK_IMAGE_TYPE_1D:
		imgTypeStr = imgTypeG + "image1D"
	case VkImageType_VK_IMAGE_TYPE_2D:
		imgTypeStr = imgTypeG + "image2D"
	case VkImageType_VK_IMAGE_TYPE_3D:
		imgTypeStr = imgTypeG + "image3D"
	default:
		return []uint32{}, fmt.Errorf("unknown image type: %v", imageType)
	}

	// Determine the declaration of block offsets
	var blockOffsetStr string
	switch imageType {
	case VkImageType_VK_IMAGE_TYPE_1D:
		blockOffsetStr = `uint block_offset_x;`
	case VkImageType_VK_IMAGE_TYPE_2D:
		blockOffsetStr = `uint block_offset_x;
		uint block_offset_y;`
	case VkImageType_VK_IMAGE_TYPE_3D:
		blockOffsetStr = `uint block_offset_x;
		uint block_offset_y;
		uint block_offset_z;`
	}

	// Determine the declaration of block extent
	var blockExtentStr string
	switch imageType {
	case VkImageType_VK_IMAGE_TYPE_1D:
		blockExtentStr = `uint block_width;`
	case VkImageType_VK_IMAGE_TYPE_2D:
		blockExtentStr = `uint block_width;
		uint block_height;`
	case VkImageType_VK_IMAGE_TYPE_3D:
		blockExtentStr = `uint block_width;
		uint block_height;
		uint block_depth;`
	}

	// Calculate the pixel position in the block
	var posStr string
	switch imageType {
	case VkImageType_VK_IMAGE_TYPE_1D:
		posStr = `int pos = int(linear_pos + block_offset_x);`
	case VkImageType_VK_IMAGE_TYPE_2D:
		posStr = `ivec2 pos = ivec2(int(linear_pos%block_width), int(linear_pos/block_width));
		pos = pos + ivec2(int(block_offset_x), int(block_offset_y));`
	case VkImageType_VK_IMAGE_TYPE_3D:
		posStr = `uint depth_pitch = block_width * block_height;
		int z = int(linear_pos / depth_pitch);
		int y = int(linear_pos % depth_pitch) / int(block_width);
		int x = int(linear_pos % depth_pitch) % int(block_width);
		ivec3 pos = ivec3(x, y, z) + ivec3(int(block_offset_x), int(block_offset_y), int(block_offset_z));`
	}

	// Convert the data
	var valueStr string
	switch aspect {
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT:
		switch vkFmt {
		case VkFormat_VK_FORMAT_R8_UINT,
			VkFormat_VK_FORMAT_R16_UINT,
			VkFormat_VK_FORMAT_R32_UINT:
			valueStr = `uint r = data[buf_texel_index * 4];
			uvec4 value = uvec4(r, 0, 0, 0);`
		case VkFormat_VK_FORMAT_R8G8_UINT,
			VkFormat_VK_FORMAT_R16G16_UINT,
			VkFormat_VK_FORMAT_R32G32_UINT:
			valueStr = `uint r = data[buf_texel_index * 4];
			uint g = data[buf_texel_index * 4 + 1];
			uvec4 value = uvec4(r, g, 0, 0);`

		case VkFormat_VK_FORMAT_R8G8B8A8_UINT,
			VkFormat_VK_FORMAT_B8G8R8A8_UINT,
			VkFormat_VK_FORMAT_R16G16B16A16_UINT,
			VkFormat_VK_FORMAT_R32G32B32A32_UINT,
			VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
			VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
			valueStr = `uint r = data[buf_texel_index * 4];
			uint g = data[buf_texel_index * 4 + 1];
			uint b = data[buf_texel_index * 4 + 2];
			uint a = data[buf_texel_index * 4 + 3];
			uvec4 value = uvec4(r, g, b, a);`

		case VkFormat_VK_FORMAT_R8_SINT,
			VkFormat_VK_FORMAT_R16_SINT,
			VkFormat_VK_FORMAT_R32_SINT:
			valueStr = `int r = int(data[buf_texel_index * 4]);
			ivec4 value = ivec4(r, 0, 0, 0);`

		case VkFormat_VK_FORMAT_R8G8_SINT,
			VkFormat_VK_FORMAT_R16G16_SINT,
			VkFormat_VK_FORMAT_R32G32_SINT:
			valueStr = `int r = int(data[buf_texel_index * 4]);
			int g = int(data[buf_texel_index * 4 + 1]);
			ivec4 value = ivec4(r, g, 0, 0);`

		case VkFormat_VK_FORMAT_R8G8B8A8_SINT,
			VkFormat_VK_FORMAT_B8G8R8A8_SINT,
			VkFormat_VK_FORMAT_R16G16B16A16_SINT,
			VkFormat_VK_FORMAT_R32G32B32A32_SINT,
			VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
			VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32:
			valueStr = `int r = int(data[buf_texel_index * 4]);
			int g = int(data[buf_texel_index * 4 + 1]);
			int b = int(data[buf_texel_index * 4 + 2]);
			int a = int(data[buf_texel_index * 4 + 3]);
			ivec4 value = ivec4(r, g, b, a);`

		case VkFormat_VK_FORMAT_R8_UNORM,
			VkFormat_VK_FORMAT_R8_SRGB:
			valueStr = `float r = data[buf_texel_index * 4] / 255.0;
			vec4 value = vec4(r, 0.0, 0.0, 0.0);`

		case VkFormat_VK_FORMAT_R8G8_UNORM,
			VkFormat_VK_FORMAT_R8G8_SRGB:
			valueStr = `float r = data[buf_texel_index * 4] / 255.0;
			float g = data[buf_texel_index * 4 + 1] / 255.0;
			vec4 value = vec4(r, g, 0.0, 0.0);`

		case VkFormat_VK_FORMAT_B8G8R8_UNORM:
			valueStr = `float r = data[buf_texel_index * 4] / 255.0;
			float g = data[buf_texel_index * 4 + 1] / 255.0;
			float b = data[buf_texel_index * 4 + 2] / 255.0;
			vec4 value = vec4(r, g, b, 0.0);`

		case VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
			VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
			VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
			VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
			VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
			VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
			valueStr = `float r = data[buf_texel_index * 4] / 255.0;
			float g = data[buf_texel_index * 4 + 1] / 255.0;
			float b = data[buf_texel_index * 4 + 2] / 255.0;
			float a = data[buf_texel_index * 4 + 3] / 255.0;
			vec4 value = vec4(r, g, b, a);`

		case VkFormat_VK_FORMAT_R16_UNORM:
			valueStr = `float r = data[buf_texel_index * 4] / 65535.0;
			vec4 value = vec4(r, 0.0, 0.0, 0.0);`
		case VkFormat_VK_FORMAT_R16G16_UNORM:
			valueStr = `float r = data[buf_texel_index * 4] / 65535.0;
			float g = data[buf_texel_index * 4 + 1] / 65535.0;
			vec4 value = vec4(r, g, 0.0, 0.0);`
		case VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
			valueStr = `float r = data[buf_texel_index * 4] / 65535.0;
			float g = data[buf_texel_index * 4 + 1] / 65535.0;
			float b = data[buf_texel_index * 4 + 2] / 65535.0;
			float a = data[buf_texel_index * 4 + 3] / 65535.0;
			vec4 value = vec4(r, g, b, a);`

		case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
			valueStr = `float r = data[buf_texel_index * 4] / 1023.0;
			float g = data[buf_texel_index * 4 + 1] / 1023.0;
			float b = data[buf_texel_index * 4 + 2] / 1023.0;
			float a = data[buf_texel_index * 4 + 3] / 3.0;
			vec4 value = vec4(r, g, b, a);`

		case VkFormat_VK_FORMAT_R8_SNORM:
			valueStr = `float r = (int(data[buf_texel_index * 4]) * 2.0 + 1.0) / 255.0;
			vec4 value = vec4(r, 0.0, 0.0, 0.0);`

		case VkFormat_VK_FORMAT_R8G8_SNORM:
			valueStr = `float r = (int(data[buf_texel_index * 4]) * 2.0 + 1.0) / 255.0;
			float g = (int(data[buf_texel_index * 4 + 1]) * 2.0 + 1.0) / 255.0;
			vec4 value = vec4(r, g, 0.0, 0.0);`
			// VkFormat_VK_FORMAT_R8G8B8_SNORM,
		case VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
			VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
			VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
			valueStr = `float r = (int(data[buf_texel_index * 4]) * 2.0 + 1.0) / 255.0;
			float g = (int(data[buf_texel_index * 4 + 1]) * 2.0 + 1.0) / 255.0;
			float b = (int(data[buf_texel_index * 4 + 2]) * 2.0 + 1.0) / 255.0;
			float a = (int(data[buf_texel_index * 4 + 3]) * 2.0 + 1.0) / 255.0;
			vec4 value = vec4(r, g, b, a);`

		case VkFormat_VK_FORMAT_R16_SNORM:
			valueStr = `float r = (int(data[buf_texel_index * 4]) * 2.0 + 1.0) / 65535.0;
			vec4 value = vec4(r, 0.0, 0.0, 0.0);`
		case VkFormat_VK_FORMAT_R16G16_SNORM:
			valueStr = `float r = (int(data[buf_texel_index * 4]) * 2.0 + 1.0) / 65535.0;
			float g = (int(data[buf_texel_index * 4 + 1]) * 2.0 + 1.0) / 65535.0;
			vec4 value = vec4(r, g, 0.0, 0.0);`
		case VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
			valueStr = `float r = (int(data[buf_texel_index * 4]) * 2.0 + 1.0) / 65535.0;
			float g = (int(data[buf_texel_index * 4 + 1]) * 2.0 + 1.0) / 65535.0;
			float b = (int(data[buf_texel_index * 4 + 2]) * 2.0 + 1.0) / 65535.0;
			float a = (int(data[buf_texel_index * 4 + 3]) * 2.0 + 1.0) / 65535.0;
			vec4 value = vec4(r, g, b, a);`

		// case VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32,
		// 	VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32:

		case VkFormat_VK_FORMAT_R16_SFLOAT,
			VkFormat_VK_FORMAT_R32_SFLOAT:
			valueStr = `float r = uintBitsToFloat(data[buf_texel_index * 4]);
			vec4 value = vec4(r, 0.0, 0.0, 0.0);`
		case VkFormat_VK_FORMAT_R16G16_SFLOAT,
			VkFormat_VK_FORMAT_R32G32_SFLOAT:
			// VkFormat_VK_FORMAT_R16G16B16_SFLOAT,
			valueStr = `float r = uintBitsToFloat(data[buf_texel_index * 4]);
			float g = uintBitsToFloat(data[buf_texel_index * 4 + 1]);
			vec4 value = vec4(r, g, 0.0, 0.0);`
		case VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32:
			// VkFormat_VK_FORMAT_R32G32B32_SFLOAT,
			// VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32,
			valueStr = `float r = uintBitsToFloat(data[buf_texel_index * 4]);
			float g = uintBitsToFloat(data[buf_texel_index * 4 + 1]);
			float b = uintBitsToFloat(data[buf_texel_index * 4 + 2]);
			vec4 value = vec4(r, g, b, 0.0);`
		case VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT,
			VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT:
			valueStr = `float r = uintBitsToFloat(data[buf_texel_index * 4]);
			float g = uintBitsToFloat(data[buf_texel_index * 4 + 1]);
			float b = uintBitsToFloat(data[buf_texel_index * 4 + 2]);
			float a = uintBitsToFloat(data[buf_texel_index * 4 + 3]);
			vec4 value = vec4(r, g, b, a);`
		default:
			return []uint32{}, fmt.Errorf("unsupported format: %v", vkFmt)
		}
	case VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT,
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT:
		// should never reach here
		return []uint32{}, fmt.Errorf("imageStore to depth/stencil image is not yet supported")
	}

	// Generate source code
	source := fmt.Sprintf(
		`#version 450
	precision highp int;
	layout (local_size_x = 128, local_size_y = 1, local_size_z = 1) in;
	layout (%s, set = 0, binding = 0) uniform %s img_output;
	layout (set = 0, binding = 1) buffer SourceData {
		uint data[];
	};
	layout (set = 0, binding = 2) uniform metadata {
		%s
		%s
		uint texel_offset;
		uint count;
	};
	void main() {
		uint buf_texel_index = gl_GlobalInvocationID.x;
		if (buf_texel_index >= count) {
			return;
		}
		uint linear_pos = buf_texel_index + texel_offset;
		%s
		%s
		imageStore(img_output, pos, value);
	}
	`, imgFmtStr, imgTypeStr, blockOffsetStr, blockExtentStr, posStr, valueStr)

	opt := shadertools.CompileOptions{
		ShaderType: shadertools.TypeCompute,
		ClientType: shadertools.Vulkan,
	}
	return shadertools.CompileGlsl(source, opt)
}
