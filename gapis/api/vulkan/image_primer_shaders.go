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

// ipRenderVertexShaderSpirv returns a vertex shader for priming by rendering
// with hard-coded vertex data, in SPIR-V words.
func ipRenderVertexShaderSpirv() ([]uint32, error) {
	return shadertools.CompileGlsl(
		`#version 450
vec2 positions[6] = vec2[](
	vec2(1.0, 1.0),
	vec2(-1.0, -1.0),
	vec2(-1.0, 1.0),
	vec2(1.0, 1.0),
	vec2(1.0, -1.0),
	vec2(-1.0, -1.0)
);
void main() {
	gl_Position = vec4(positions[gl_VertexIndex], 0.0, 1.0);
}`,
		shadertools.CompileOptions{
			ShaderType: shadertools.TypeVertex,
			ClientType: shadertools.Vulkan,
		})
}

// ipRenderColorShaderSpirv returns a fragment shader for priming by rendering
// for color aspect data, in SPIR-V words.
func ipRenderColorShaderSpirv(vkFmt VkFormat) ([]uint32, error) {
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

	}
	return []uint32{}, fmt.Errorf("%v is not supported", vkFmt)
}

// ipRenderDepthShaderSpirv returns a fragment shader for priming by rendering
// for depth aspect data, in SPIR-V words.
func ipRenderDepthShaderSpirv(vkFmt VkFormat) ([]uint32, error) {
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

	}
	return []uint32{}, fmt.Errorf("%v is not supported", vkFmt)
}

// ipRenderStencilShaderSpirv returns a fragment shader for priming by rendering
// for stencil aspect data, in SPIR-V words.
func ipRenderStencilShaderSpirv() ([]uint32, error) {

	return shadertools.CompileGlsl(
		`#version 450
precision highp int;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_stencil;
layout (push_constant) uniform mask_data { uint current_bit; };
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
}

// ipComputeShaderSpirv returns the compute shader to be used for priming image
// data through imageStore operation.
func ipComputeShaderSpirv(
	outputFormat VkFormat, outputAspect VkImageAspectFlagBits, inputFormat VkFormat,
	inputAspect VkImageAspectFlagBits, imageType VkImageType) ([]uint32, error) {

	if outputAspect != VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT ||
		inputAspect != VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT {
		return []uint32{}, fmt.Errorf("Aspect other than COLOR is not supported")
	}

	fmtStr := func(format VkFormat) (string, error) {
		switch format {
		// uint formats
		case VkFormat_VK_FORMAT_R8_UINT:
			return "r8ui", nil
		case VkFormat_VK_FORMAT_R16_UINT:
			return "r16ui", nil
		case VkFormat_VK_FORMAT_R32_UINT:
			return "r32ui", nil

		case VkFormat_VK_FORMAT_R8G8_UINT:
			return "rg8ui", nil
		case VkFormat_VK_FORMAT_R16G16_UINT:
			return "rg16ui", nil
		case VkFormat_VK_FORMAT_R32G32_UINT:
			return "rg32ui", nil

		case VkFormat_VK_FORMAT_R8G8B8A8_UINT,
			VkFormat_VK_FORMAT_B8G8R8A8_UINT,
			VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32:
			return "rgba8ui", nil
		case VkFormat_VK_FORMAT_R16G16B16A16_UINT:
			return "rgba16ui", nil
		case VkFormat_VK_FORMAT_R32G32B32A32_UINT:
			return "rgba32ui", nil

		case VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
			return "rgb10_a2ui", nil

		// sint formats
		case VkFormat_VK_FORMAT_R8_SINT:
			return "r8i", nil
		case VkFormat_VK_FORMAT_R16_SINT:
			return "r16i", nil
		case VkFormat_VK_FORMAT_R32_SINT:
			return "r32i", nil

		case VkFormat_VK_FORMAT_R8G8_SINT:
			return "rg8i", nil
		case VkFormat_VK_FORMAT_R16G16_SINT:
			return "rg16i", nil
		case VkFormat_VK_FORMAT_R32G32_SINT:
			return "rg32i", nil

		case VkFormat_VK_FORMAT_R8G8B8A8_SINT,
			VkFormat_VK_FORMAT_B8G8R8A8_SINT,
			VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32:
			return "rgba8i", nil
		case VkFormat_VK_FORMAT_R16G16B16A16_SINT:
			return "rgba16i", nil
		case VkFormat_VK_FORMAT_R32G32B32A32_SINT:
			return "rgba32i", nil

		// unorm formats
		case VkFormat_VK_FORMAT_R8_UNORM,
			VkFormat_VK_FORMAT_R8_SRGB:
			return "r8", nil
		case VkFormat_VK_FORMAT_R16_UNORM:
			return "r16", nil

		case VkFormat_VK_FORMAT_R8G8_UNORM,
			VkFormat_VK_FORMAT_R8G8_SRGB:
			return "rg8", nil
		case VkFormat_VK_FORMAT_R16G16_UNORM:
			return "rg16", nil

		case VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
			VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
			VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
			VkFormat_VK_FORMAT_B8G8R8A8_SRGB:
			return "rgba8", nil

		case VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
			return "rgba16", nil

		case VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
			VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
			return "rgba8", nil

		case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
			VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
			return "rgb10_a2", nil

		// snorm formats
		case VkFormat_VK_FORMAT_R8_SNORM:
			return "r8_snorm", nil
		case VkFormat_VK_FORMAT_R16_SNORM:
			return "r16_snorm", nil

		case VkFormat_VK_FORMAT_R8G8_SNORM:
			return "rg8_snorm", nil
		case VkFormat_VK_FORMAT_R16G16_SNORM:
			return "rg16_snorm", nil

		case VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
			VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
			VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
			return "rgba8_snorm", nil
		case VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
			return "rgba16_snorm", nil

		// float formats
		case VkFormat_VK_FORMAT_R16_SFLOAT:
			return "r16f", nil
		case VkFormat_VK_FORMAT_R32_SFLOAT:
			return "r32f", nil

		case VkFormat_VK_FORMAT_R16G16_SFLOAT:
			return "rg16f", nil
		case VkFormat_VK_FORMAT_R32G32_SFLOAT:
			return "rg32f", nil

		case VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT:
			return "rgba16f", nil
		case VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT:
			return "rgba32f", nil

		case VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32:
			return "r11f_g11f_b10f", nil
		}
		return "", fmt.Errorf("Unsupported format: %v", format)
	}

	fmtG := func(format VkFormat) (string, error) {
		switch format {
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
			return "u", nil

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
			return "i", nil

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
			return "", nil
		}
		return "", fmt.Errorf("Not supported format: %v", format)
	}

	typeStr := func(ty VkImageType) (string, error) {
		switch ty {
		case VkImageType_VK_IMAGE_TYPE_1D:
			return "image1D", nil
		case VkImageType_VK_IMAGE_TYPE_2D:
			return "image2D", nil
		case VkImageType_VK_IMAGE_TYPE_3D:
			return "image3D", nil
		}
		return "", fmt.Errorf("Not supported image type: %v", ty)
	}

	posStr := func(ty VkImageType) (string, error) {
		switch ty {
		case VkImageType_VK_IMAGE_TYPE_1D:
			return `int pos = x;`, nil
		case VkImageType_VK_IMAGE_TYPE_2D:
			return `ivec2 pos = ivec2(x, y);`, nil
		case VkImageType_VK_IMAGE_TYPE_3D:
			return `ivec3 pos = ivec3(x, y, z);`, nil
		}
		return "", fmt.Errorf("Not supported image type: %v", ty)
	}

	colorStr := func(inputFmt, outputFmt VkFormat) (string, error) {
		inputG, err := fmtG(inputFmt)
		if err != nil {
			return "", err
		}
		if inputFmt == outputFmt {
			return inputG + `vec4 color = imageLoad(input_img, pos);`, nil
		}

		// Only support R32G32B32A32_UINT as the input format, when input and
		// output format are different.
		if inputFmt == VkFormat_VK_FORMAT_R32G32B32A32_UINT {
			switch outputFmt {
			case VkFormat_VK_FORMAT_R8_UINT,
				VkFormat_VK_FORMAT_R8G8_UINT,
				// VkFormat_VK_FORMAT_R8G8B8_UINT,
				VkFormat_VK_FORMAT_R8G8B8A8_UINT,
				// VkFormat_VK_FORMAT_B8G8R8_UINT,
				VkFormat_VK_FORMAT_B8G8R8A8_UINT,
				VkFormat_VK_FORMAT_R16_UINT,
				VkFormat_VK_FORMAT_R16G16_UINT,
				// VkFormat_VK_FORMAT_R16G16B16_UINT,
				VkFormat_VK_FORMAT_R16G16B16A16_UINT,
				VkFormat_VK_FORMAT_R32_UINT,
				VkFormat_VK_FORMAT_R32G32_UINT,
				// VkFormat_VK_FORMAT_R32G32B32_UINT,
				VkFormat_VK_FORMAT_R32G32B32A32_UINT,
				VkFormat_VK_FORMAT_A8B8G8R8_UINT_PACK32,
				VkFormat_VK_FORMAT_A2R10G10B10_UINT_PACK32,
				VkFormat_VK_FORMAT_A2B10G10R10_UINT_PACK32:
				return `uvec4 color = imageLoad(input_img, pos);`, nil

			case VkFormat_VK_FORMAT_R8_SINT,
				VkFormat_VK_FORMAT_R8G8_SINT,
				// VkFormat_VK_FORMAT_R8G8B8_SINT,
				VkFormat_VK_FORMAT_R8G8B8A8_SINT,
				// VkFormat_VK_FORMAT_B8G8R8_SINT,
				VkFormat_VK_FORMAT_B8G8R8A8_SINT,
				VkFormat_VK_FORMAT_R16_SINT,
				VkFormat_VK_FORMAT_R16G16_SINT,
				// VkFormat_VK_FORMAT_R16G16B16_SINT,
				VkFormat_VK_FORMAT_R16G16B16A16_SINT,
				VkFormat_VK_FORMAT_R32_SINT,
				VkFormat_VK_FORMAT_R32G32_SINT,
				// VkFormat_VK_FORMAT_R32G32B32_SINT,
				VkFormat_VK_FORMAT_R32G32B32A32_SINT,
				VkFormat_VK_FORMAT_A8B8G8R8_SINT_PACK32,
				VkFormat_VK_FORMAT_A2R10G10B10_SINT_PACK32,
				VkFormat_VK_FORMAT_A2B10G10R10_SINT_PACK32:
				return `uvec4 input_color = imageLoad(input_img, pos);
				int r = int(input_color.r);
				int g = int(input_color.g);
				int b = int(input_color.b);
				int a = int(input_color.a);
				ivec4 color = ivec4(r, g, b, a);
				`, nil

			case VkFormat_VK_FORMAT_R8_UNORM,
				VkFormat_VK_FORMAT_R8G8_UNORM,
				// VkFormat_VK_FORMAT_R8G8B8_UNORM,
				VkFormat_VK_FORMAT_R8G8B8A8_UNORM,
				VkFormat_VK_FORMAT_B8G8R8_UNORM,
				VkFormat_VK_FORMAT_B8G8R8A8_UNORM,
				VkFormat_VK_FORMAT_R8_SRGB,
				VkFormat_VK_FORMAT_R8G8_SRGB,
				// VkFormat_VK_FORMAT_R8G8B8_SRGB,
				VkFormat_VK_FORMAT_R8G8B8A8_SRGB,
				// VkFormat_VK_FORMAT_B8G8R8_SRGB,
				VkFormat_VK_FORMAT_B8G8R8A8_SRGB,
				VkFormat_VK_FORMAT_A8B8G8R8_UNORM_PACK32,
				VkFormat_VK_FORMAT_A8B8G8R8_SRGB_PACK32:
				return `vec4 color = imageLoad(input_img, pos).rgba/vec4(255.0, 255.0, 255.0, 255.0);`, nil

			case VkFormat_VK_FORMAT_R16_UNORM,
				VkFormat_VK_FORMAT_R16G16_UNORM,
				// VkFormat_VK_FORMAT_R16G16B16_UNORM,
				VkFormat_VK_FORMAT_R16G16B16A16_UNORM:
				return `vec4 color = imageLoad(input_img, pos).rgba/vec4(65535.0, 65535.0, 65535.0, 65535.0);`, nil

			case VkFormat_VK_FORMAT_A2R10G10B10_UNORM_PACK32,
				VkFormat_VK_FORMAT_A2B10G10R10_UNORM_PACK32:
				return `vec4 color = imageLoad(input_img, pos).rgba/vec4(1023.0, 1023.0, 1023.0, 3.0);`, nil

			case VkFormat_VK_FORMAT_R8_SNORM,
				VkFormat_VK_FORMAT_R8G8_SNORM,
				// VkFormat_VK_FORMAT_R8G8B8_SNORM,
				VkFormat_VK_FORMAT_R8G8B8A8_SNORM,
				// VkFormat_VK_FORMAT_B8G8R8_SNORM,
				VkFormat_VK_FORMAT_B8G8R8A8_SNORM,
				VkFormat_VK_FORMAT_A8B8G8R8_SNORM_PACK32:
				return `float r = (int(imageLoad(input_img, pos).r) * 2.0 + 1.0) / 255.0;
					float g = (int(imageLoad(input_img, pos).g) * 2.0 + 1.0) / 255.0;
					float b = (int(imageLoad(input_img, pos).b) * 2.0 + 1.0) / 255.0;
					float a = (int(imageLoad(input_img, pos).a) * 2.0 + 1.0) / 255.0;
					vec4 color = vec4(r, g, b, a);`, nil

			case VkFormat_VK_FORMAT_R16_SNORM,
				VkFormat_VK_FORMAT_R16G16_SNORM,
				// VkFormat_VK_FORMAT_R16G16B16_SNORM,
				VkFormat_VK_FORMAT_R16G16B16A16_SNORM:
				return `float r = (int(imageLoad(input_img, pos).r) * 2.0 + 1.0) / 65535.0;
					float g = (int(imageLoad(input_img, pos).g) * 2.0 + 1.0) / 65535.0;
					float b = (int(imageLoad(input_img, pos).b) * 2.0 + 1.0) / 65535.0;
					float a = (int(imageLoad(input_img, pos).a) * 2.0 + 1.0) / 65535.0;
					vec4 color = vec4(r, g, b, a);`, nil

			// case VkFormat_VK_FORMAT_A2R10G10B10_SNORM_PACK32,
			// 	VkFormat_VK_FORMAT_A2B10G10R10_SNORM_PACK32:

			case VkFormat_VK_FORMAT_R16_SFLOAT,
				VkFormat_VK_FORMAT_R16G16_SFLOAT,
				// VkFormat_VK_FORMAT_R16G16B16_SFLOAT,
				VkFormat_VK_FORMAT_R16G16B16A16_SFLOAT,
				VkFormat_VK_FORMAT_R32_SFLOAT,
				VkFormat_VK_FORMAT_R32G32_SFLOAT,
				// VkFormat_VK_FORMAT_R32G32B32_SFLOAT,
				VkFormat_VK_FORMAT_R32G32B32A32_SFLOAT,
				// VkFormat_VK_FORMAT_E5B9G9R9_UFLOAT_PACK32,
				VkFormat_VK_FORMAT_B10G11R11_UFLOAT_PACK32:
				return `float r = uintBitsToFloat(imageLoad(input_img, pos).r);
					float g = uintBitsToFloat(imageLoad(input_img, pos).g);
					float b = uintBitsToFloat(imageLoad(input_img, pos).b);
					float a = uintBitsToFloat(imageLoad(input_img, pos).a);
					vec4 color = vec4(r, g, b, a);`, nil
			}
		}

		return "", fmt.Errorf("Unsupported format, input fomrat: %v, output format: %v", inputFmt, outputFmt)
	}

	outputFmtStr, err := fmtStr(outputFormat)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating output image format string, err: %v", err)
	}
	inputFmtStr, err := fmtStr(inputFormat)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating input image format string, err: %v", err)
	}
	outputG, err := fmtG(outputFormat)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating output image unit format string, err: %v", err)
	}
	inputG, err := fmtG(inputFormat)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating input image unit format string, err: %v", err)
	}
	imgTypeStr, err := typeStr(imageType)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating image type string, err: %v", err)
	}
	pos, err := posStr(imageType)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating position, err: %v", err)
	}
	color, err := colorStr(inputFormat, outputFormat)
	if err != nil {
		return []uint32{}, fmt.Errorf("Generating color, err: %v", err)
	}

	// Generate source code
	source := fmt.Sprintf(
		`#version 450
	precision highp int;
	layout (local_size_x = 1, local_size_y = 1, local_size_z = 1) in;
	layout (%s, set = 0, binding = %d) uniform %s%s output_img;
	layout (%s, set = 0, binding = %d) uniform %s%s input_img;
	layout (push_constant) uniform metadata {
		uint offset_x;
		uint offset_y;
		uint offset_z;
		// Reserved for handling image formats wider than 32 bit per channel
		uint input_img_index;
	};
	void main() {
		int x = int(gl_GlobalInvocationID.x + offset_x);
		int y = int(gl_GlobalInvocationID.y + offset_y);
		int z = int(gl_GlobalInvocationID.z + offset_z);
		%s
		%s
		imageStore(output_img, pos, color);
	}
	`, outputFmtStr, ipStoreOutputImageBinding, outputG, imgTypeStr,
		inputFmtStr, ipStoreInputImageBinding, inputG, imgTypeStr,
		pos, color)

	opt := shadertools.CompileOptions{
		ShaderType: shadertools.TypeCompute,
		ClientType: shadertools.Vulkan,
	}
	return shadertools.CompileGlsl(source, opt)
}
