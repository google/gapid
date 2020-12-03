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

package shadertools_test

import (
	"testing"

	"github.com/google/gapid/core/assert"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/shadertools"
)

func TestCompileGlsl(t *testing.T) {
	for _, test := range []struct {
		desc     string
		src      string
		shaderTy shadertools.ShaderType
		clientTy shadertools.ClientType
		expected []uint32
	}{
		{
			"Test compile Vulkan vertex shader",
			`#version 450
layout(location=0) in vec3 position;
void main() {
	gl_Position = vec4(position, 1.0);
}`,
			shadertools.TypeVertex,
			shadertools.Vulkan,
			[]uint32{0x07230203, 0x00010000, 0x00070000, 0x0000001b,
				0x00000000, 0x00020011, 0x00000001, 0x0006000b,
				0x00000001, 0x4c534c47, 0x6474732e, 0x3035342e,
				0x00000000, 0x0003000e, 0x00000000, 0x00000001,
				0x0007000f, 0x00000000, 0x00000002, 0x6e69616d,
				0x00000000, 0x00000003, 0x00000004, 0x00030003,
				0x00000002, 0x000001c2, 0x00040005, 0x00000002,
				0x6e69616d, 0x00000000, 0x00060005, 0x00000005,
				0x505f6c67, 0x65567265, 0x78657472, 0x00000000,
				0x00060006, 0x00000005, 0x00000000, 0x505f6c67,
				0x7469736f, 0x006e6f69, 0x00070006, 0x00000005,
				0x00000001, 0x505f6c67, 0x746e696f, 0x657a6953,
				0x00000000, 0x00070006, 0x00000005, 0x00000002,
				0x435f6c67, 0x4470696c, 0x61747369, 0x0065636e,
				0x00070006, 0x00000005, 0x00000003, 0x435f6c67,
				0x446c6c75, 0x61747369, 0x0065636e, 0x00030005,
				0x00000003, 0x00000000, 0x00050005, 0x00000004,
				0x69736f70, 0x6e6f6974, 0x00000000, 0x00050048,
				0x00000005, 0x00000000, 0x0000000b, 0x00000000,
				0x00050048, 0x00000005, 0x00000001, 0x0000000b,
				0x00000001, 0x00050048, 0x00000005, 0x00000002,
				0x0000000b, 0x00000003, 0x00050048, 0x00000005,
				0x00000003, 0x0000000b, 0x00000004, 0x00030047,
				0x00000005, 0x00000002, 0x00040047, 0x00000004,
				0x0000001e, 0x00000000, 0x00020013, 0x00000006,
				0x00030021, 0x00000007, 0x00000006, 0x00030016,
				0x00000008, 0x00000020, 0x00040017, 0x00000009,
				0x00000008, 0x00000004, 0x00040015, 0x0000000a,
				0x00000020, 0x00000000, 0x0004002b, 0x0000000a,
				0x0000000b, 0x00000001, 0x0004001c, 0x0000000c,
				0x00000008, 0x0000000b, 0x0006001e, 0x00000005,
				0x00000009, 0x00000008, 0x0000000c, 0x0000000c,
				0x00040020, 0x0000000d, 0x00000003, 0x00000005,
				0x0004003b, 0x0000000d, 0x00000003, 0x00000003,
				0x00040015, 0x0000000e, 0x00000020, 0x00000001,
				0x0004002b, 0x0000000e, 0x0000000f, 0x00000000,
				0x00040017, 0x00000010, 0x00000008, 0x00000003,
				0x00040020, 0x00000011, 0x00000001, 0x00000010,
				0x0004003b, 0x00000011, 0x00000004, 0x00000001,
				0x0004002b, 0x00000008, 0x00000012, 0x3f800000,
				0x00040020, 0x00000013, 0x00000003, 0x00000009,
				0x00050036, 0x00000006, 0x00000002, 0x00000000,
				0x00000007, 0x000200f8, 0x00000014, 0x0004003d,
				0x00000010, 0x00000015, 0x00000004, 0x00050051,
				0x00000008, 0x00000016, 0x00000015, 0x00000000,
				0x00050051, 0x00000008, 0x00000017, 0x00000015,
				0x00000001, 0x00050051, 0x00000008, 0x00000018,
				0x00000015, 0x00000002, 0x00070050, 0x00000009,
				0x00000019, 0x00000016, 0x00000017, 0x00000018,
				0x00000012, 0x00050041, 0x00000013, 0x0000001a,
				0x00000003, 0x0000000f, 0x0003003e, 0x0000001a,
				0x00000019, 0x000100fd, 0x00010038},
		},
		{
			"Test compile fragment shader",
			`#version 450
precision highp int;
precision highp float;
out float gl_FragDepth;
layout(input_attachment_index = 0, binding = 0, set = 0) uniform usubpassInput in_depth;
void main() {
	gl_FragDepth = subpassLoad(in_depth).r / 16777215.0;
}`,
			shadertools.TypeFragment,
			shadertools.Vulkan,
			[]uint32{0x07230203, 0x00010000, 0x00070000, 0x00000019,
				0x00000000, 0x00020011, 0x00000001, 0x00020011,
				0x00000028, 0x0006000b, 0x00000001, 0x4c534c47,
				0x6474732e, 0x3035342e, 0x00000000, 0x0003000e,
				0x00000000, 0x00000001, 0x0006000f, 0x00000004,
				0x00000002, 0x6e69616d, 0x00000000, 0x00000003,
				0x00030010, 0x00000002, 0x00000007, 0x00030010,
				0x00000002, 0x0000000c, 0x00030003, 0x00000002,
				0x000001c2, 0x00040005, 0x00000002, 0x6e69616d,
				0x00000000, 0x00060005, 0x00000003, 0x465f6c67,
				0x44676172, 0x68747065, 0x00000000, 0x00050005,
				0x00000004, 0x645f6e69, 0x68747065, 0x00000000,
				0x00040047, 0x00000003, 0x0000000b, 0x00000016,
				0x00040047, 0x00000004, 0x00000022, 0x00000000,
				0x00040047, 0x00000004, 0x00000021, 0x00000000,
				0x00040047, 0x00000004, 0x0000002b, 0x00000000,
				0x00020013, 0x00000005, 0x00030021, 0x00000006,
				0x00000005, 0x00030016, 0x00000007, 0x00000020,
				0x00040020, 0x00000008, 0x00000003, 0x00000007,
				0x0004003b, 0x00000008, 0x00000003, 0x00000003,
				0x00040015, 0x00000009, 0x00000020, 0x00000000,
				0x00090019, 0x0000000a, 0x00000009, 0x00000006,
				0x00000000, 0x00000000, 0x00000000, 0x00000002,
				0x00000000, 0x00040020, 0x0000000b, 0x00000000,
				0x0000000a, 0x0004003b, 0x0000000b, 0x00000004,
				0x00000000, 0x00040015, 0x0000000c, 0x00000020,
				0x00000001, 0x0004002b, 0x0000000c, 0x0000000d,
				0x00000000, 0x00040017, 0x0000000e, 0x0000000c,
				0x00000002, 0x0005002c, 0x0000000e, 0x0000000f,
				0x0000000d, 0x0000000d, 0x00040017, 0x00000010,
				0x00000009, 0x00000004, 0x0004002b, 0x00000009,
				0x00000011, 0x00000000, 0x0004002b, 0x00000007,
				0x00000012, 0x4b7fffff, 0x00050036, 0x00000005,
				0x00000002, 0x00000000, 0x00000006, 0x000200f8,
				0x00000013, 0x0004003d, 0x0000000a, 0x00000014,
				0x00000004, 0x00050062, 0x00000010, 0x00000015,
				0x00000014, 0x0000000f, 0x00050051, 0x00000009,
				0x00000016, 0x00000015, 0x00000000, 0x00040070,
				0x00000007, 0x00000017, 0x00000016, 0x00050088,
				0x00000007, 0x00000018, 0x00000017, 0x00000012,
				0x0003003e, 0x00000003, 0x00000018, 0x000100fd,
				0x00010038},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			ctx := log.Testing(t)
			opt := shadertools.CompileOptions{
				ShaderType: test.shaderTy,
				ClientType: test.clientTy,
			}
			out, err := shadertools.CompileGlsl(test.src, opt)
			if assert.For(ctx, "err").ThatError(err).Succeeded() {
				assert.For(ctx, "src").ThatSlice(
					// disassemble and assemble so that the output converge.
					shadertools.AssembleSpirvText(
						shadertools.DisassembleSpirvBinary(out))).Equals(test.expected)
			}
		})
	}
}

func TestParseDescriptorSets(t *testing.T) {
	for _, test := range []struct {
		desc     string
		spirvDis string
		expected map[string]shadertools.DescriptorSets
	}{
		{
			"Test shadertools parses simple uniform",
			`
; SPIR-V
; Version: 1.3
; Generator: Khronos SPIR-V Tools Assembler; 0
; Bound: 50
; Schema: 0
               OpCapability Shader
          %1 = OpExtInstImport "GLSL.std.450"
               OpMemoryModel Logical GLSL450
               OpEntryPoint Vertex %2 "main" %3 %4 %5 %6 %7 %8
               OpSource GLSL 450
               OpSourceExtension "GL_ARB_separate_shader_objects"
               OpName %2 "main"
               OpName %9 "gl_PerVertex"
               OpMemberName %9 0 "gl_Position"
               OpName %3 ""
               OpName %10 "UniformBufferObject"
               OpMemberName %10 0 "model"
               OpMemberName %10 1 "view"
               OpMemberName %10 2 "proj"
               OpName %11 "ubo"
               OpName %4 "inPosition"
               OpName %5 "fragColor"
               OpName %6 "inColor"
               OpName %7 "fragTexCoord"
               OpName %8 "inTexCoord"
               OpMemberDecorate %9 0 BuiltIn Position
               OpDecorate %9 Block
               OpMemberDecorate %10 0 ColMajor
               OpMemberDecorate %10 0 Offset 0
               OpMemberDecorate %10 0 MatrixStride 16
               OpMemberDecorate %10 1 ColMajor
               OpMemberDecorate %10 1 Offset 64
               OpMemberDecorate %10 1 MatrixStride 16
               OpMemberDecorate %10 2 ColMajor
               OpMemberDecorate %10 2 Offset 128
               OpMemberDecorate %10 2 MatrixStride 16
               OpDecorate %10 Block
               OpDecorate %11 DescriptorSet 0
               OpDecorate %11 Binding 0
               OpDecorate %4 Location 0
               OpDecorate %5 Location 0
               OpDecorate %6 Location 1
               OpDecorate %7 Location 1
               OpDecorate %8 Location 2
         %12 = OpTypeVoid
         %13 = OpTypeFunction %12
         %14 = OpTypeFloat 32
         %15 = OpTypeVector %14 4
          %9 = OpTypeStruct %15
         %16 = OpTypePointer Output %9
          %3 = OpVariable %16 Output
         %17 = OpTypeInt 32 1
         %18 = OpConstant %17 0
         %19 = OpTypeMatrix %15 4
         %10 = OpTypeStruct %19 %19 %19
         %20 = OpTypePointer Uniform %10
         %11 = OpVariable %20 Uniform
         %21 = OpConstant %17 2
         %22 = OpTypePointer Uniform %19
         %23 = OpConstant %17 1
         %24 = OpTypeVector %14 3
         %25 = OpTypePointer Input %24
          %4 = OpVariable %25 Input
         %26 = OpConstant %14 1
         %27 = OpTypePointer Output %15
         %28 = OpTypePointer Output %24
          %5 = OpVariable %28 Output
          %6 = OpVariable %25 Input
         %29 = OpTypeVector %14 2
         %30 = OpTypePointer Output %29
          %7 = OpVariable %30 Output
         %31 = OpTypePointer Input %29
          %8 = OpVariable %31 Input
          %2 = OpFunction %12 None %13
         %32 = OpLabel
         %33 = OpAccessChain %22 %11 %21
         %34 = OpLoad %19 %33
         %35 = OpAccessChain %22 %11 %23
         %36 = OpLoad %19 %35
         %37 = OpMatrixTimesMatrix %19 %34 %36
         %38 = OpAccessChain %22 %11 %18
         %39 = OpLoad %19 %38
         %40 = OpMatrixTimesMatrix %19 %37 %39
         %41 = OpLoad %24 %4
         %42 = OpCompositeExtract %14 %41 0
         %43 = OpCompositeExtract %14 %41 1
         %44 = OpCompositeExtract %14 %41 2
         %45 = OpCompositeConstruct %15 %42 %43 %44 %26
         %46 = OpMatrixTimesVector %15 %40 %45
         %47 = OpAccessChain %27 %3 %18
               OpStore %47 %46
         %48 = OpLoad %24 %6
               OpStore %5 %48
         %49 = OpLoad %29 %8
               OpStore %7 %49
               OpReturn
               OpFunctionEnd
			`,
			map[string]shadertools.DescriptorSets{
				"main": shadertools.DescriptorSets{
					0: shadertools.DescriptorSet{
						shadertools.DescriptorBinding{
							Set:             0,
							Binding:         0,
							SpirvId:         11,
							DescriptorType:  6, // VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
							DescriptorCount: 1,
							ShaderStage:     1, // VK_SHADER_STAGE_VERTEX_BIT
						},
					},
				},
			},
		},
		{
			"Test shadertools handles multiple entrypoints",
			multientrypoint_spv,
			map[string]shadertools.DescriptorSets{
				"entry_vert": shadertools.DescriptorSets{
					0: shadertools.DescriptorSet{
						shadertools.DescriptorBinding{
							Set:             0,
							Binding:         1,
							SpirvId:         11,
							DescriptorType:  6, // VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
							DescriptorCount: 1,
							ShaderStage:     1, // VK_SHADER_STAGE_VERTEX_BIT
						},
					},
				},
				"entry_frag": shadertools.DescriptorSets{
					0: shadertools.DescriptorSet{
						shadertools.DescriptorBinding{
							Set:             0,
							Binding:         0,
							SpirvId:         17,
							DescriptorType:  1, // VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER
							DescriptorCount: 1,
							ShaderStage:     16, // VK_SHADER_STAGE_FRAGMENT_BIT
						},
						shadertools.DescriptorBinding{
							Set:             0,
							Binding:         1,
							SpirvId:         11,
							DescriptorType:  6, // VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER
							DescriptorCount: 1,
							ShaderStage:     16, // VK_SHADER_STAGE_FRAGMENT_BIT
						},
					},
				},
			},
		},
		{
			"Test shadertools parses array descriptor count",
			`
; SPIR-V
; Version: 1.3
; Generator: Khronos SPIR-V Tools Assembler; 0
; Bound: 53
; Schema: 0
               OpCapability Shader
          %1 = OpExtInstImport "GLSL.std.450"
               OpMemoryModel Logical GLSL450
               OpEntryPoint Fragment %2 "main" %3 %4
               OpExecutionMode %2 OriginUpperLeft
               OpSource GLSL 450
               OpName %2 "main"
               OpName %3 "colour"
               OpName %5 "samps"
               OpName %4 "uv"
               OpDecorate %3 Location 0
               OpDecorate %5 DescriptorSet 0
               OpDecorate %5 Binding 0
               OpDecorate %4 Location 0
          %6 = OpTypeVoid
          %7 = OpTypeFunction %6
          %8 = OpTypeFloat 32
          %9 = OpTypeVector %8 4
         %10 = OpTypePointer Output %9
          %3 = OpVariable %10 Output
         %11 = OpTypeImage %8 2D 0 0 0 1 Unknown
         %12 = OpTypeSampledImage %11
         %13 = OpTypeInt 32 0
         %14 = OpConstant %13 5
         %15 = OpTypeArray %12 %14
         %16 = OpConstant %13 4
         %17 = OpTypeArray %15 %16
         %18 = OpTypePointer UniformConstant %17
          %5 = OpVariable %18 UniformConstant
         %19 = OpTypeInt 32 1
         %20 = OpConstant %19 0
         %21 = OpTypePointer UniformConstant %12
         %22 = OpTypeVector %8 2
         %23 = OpTypePointer Input %22
          %4 = OpVariable %23 Input
         %24 = OpConstant %13 0
         %25 = OpConstant %19 1
         %26 = OpConstant %13 1
         %27 = OpConstant %19 2
         %28 = OpConstant %13 2
         %29 = OpConstant %19 3
         %30 = OpConstant %13 3
          %2 = OpFunction %6 None %7
         %31 = OpLabel
         %32 = OpAccessChain %21 %5 %20 %20
         %33 = OpLoad %12 %32
         %34 = OpLoad %22 %4
         %35 = OpImageSampleImplicitLod %9 %33 %34
         %36 = OpCompositeExtract %8 %35 0
         %37 = OpAccessChain %21 %5 %25 %25
         %38 = OpLoad %12 %37
         %39 = OpLoad %22 %4
         %40 = OpImageSampleImplicitLod %9 %38 %39
         %41 = OpCompositeExtract %8 %40 1
         %42 = OpAccessChain %21 %5 %27 %27
         %43 = OpLoad %12 %42
         %44 = OpLoad %22 %4
         %45 = OpImageSampleImplicitLod %9 %43 %44
         %46 = OpCompositeExtract %8 %45 2
         %47 = OpAccessChain %21 %5 %29 %29
         %48 = OpLoad %12 %47
         %49 = OpLoad %22 %4
         %50 = OpImageSampleImplicitLod %9 %48 %49
         %51 = OpCompositeExtract %8 %50 3
         %52 = OpCompositeConstruct %9 %36 %41 %46 %51
               OpStore %3 %52
               OpReturn
               OpFunctionEnd
			`,
			map[string]shadertools.DescriptorSets{
				"main": shadertools.DescriptorSets{
					0: shadertools.DescriptorSet{
						shadertools.DescriptorBinding{
							Set:             0,
							Binding:         0,
							SpirvId:         5,
							DescriptorType:  1, // VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER
							DescriptorCount: 20,
							ShaderStage:     16, // VK_SHADER_STAGE_FRAGMENT_BIT
						},
					},
				},
			},
		},
		{
			"Test shadertools works with no descriptor sets",
			`
; SPIR-V
; Version: 1.3
; Generator: Khronos SPIR-V Tools Assembler; 0
; Bound: 24
; Schema: 0
               OpCapability Shader
          %1 = OpExtInstImport "GLSL.std.450"
               OpMemoryModel Logical GLSL450
               OpEntryPoint Vertex %2 "main" %3 %4
               OpSource GLSL 450
               OpName %2 "main"
               OpName %5 "gl_PerVertex"
               OpMemberName %5 0 "gl_Position"
               OpName %3 ""
               OpName %4 "pos"
               OpMemberDecorate %5 0 BuiltIn Position
               OpDecorate %5 Block
               OpDecorate %4 Location 0
          %6 = OpTypeVoid
          %7 = OpTypeFunction %6
          %8 = OpTypeFloat 32
          %9 = OpTypeVector %8 4
          %5 = OpTypeStruct %9
         %10 = OpTypePointer Output %5
          %3 = OpVariable %10 Output
         %11 = OpTypeInt 32 1
         %12 = OpConstant %11 0
         %13 = OpTypeVector %8 3
         %14 = OpTypePointer Input %13
          %4 = OpVariable %14 Input
         %15 = OpConstant %8 1
         %16 = OpTypePointer Output %9
          %2 = OpFunction %6 None %7
         %17 = OpLabel
         %18 = OpLoad %13 %4
         %19 = OpCompositeExtract %8 %18 0
         %20 = OpCompositeExtract %8 %18 1
         %21 = OpCompositeExtract %8 %18 2
         %22 = OpCompositeConstruct %9 %19 %20 %21 %15
         %23 = OpAccessChain %16 %3 %12
               OpStore %23 %22
               OpReturn
               OpFunctionEnd
		`,
			map[string]shadertools.DescriptorSets{
				"main": shadertools.DescriptorSets{},
			},
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			ctx := log.Testing(t)
			spv := shadertools.AssembleSpirvText(test.spirvDis)
			sets, err := shadertools.ParseAllDescriptorSets(spv)
			assert.For(ctx, "err").ThatError(err).Succeeded()
			assert.For(ctx, "sets").ThatMap(sets).DeepEquals(test.expected)
		})
	}
}

var (
	multientrypoint_spv = `
; SPIR-V
; Version: 1.3
; Generator: Khronos SPIR-V Tools Assembler; 0
; Bound: 67
; Schema: 0
               OpCapability Shader
          %1 = OpExtInstImport "GLSL.std.450"
               OpMemoryModel Logical GLSL450
               OpEntryPoint Vertex %2 "entry_vert" %3 %4 %5 %6
               OpEntryPoint Fragment %7 "entry_frag" %4 %8
               OpSource GLSL 450
               OpSourceExtension "GL_GOOGLE_cpp_style_line_directive"
               OpSourceExtension "GL_GOOGLE_include_directive"
               OpName %9 "getData("
               OpName %2 "entry_vert("
               OpName %7 "entry_frag("
               OpName %10 "Ubo"
               OpMemberName %10 0 "data"
               OpName %11 "ubo"
               OpName %5 "oUV"
               OpName %4 "iUV"
               OpName %12 "gl_PerVertex"
               OpMemberName %12 0 "gl_Position"
               OpName %3 ""
               OpName %6 "pos"
               OpName %13 "PushConstantVert"
               OpMemberName %13 0 "val"
               OpName %14 "push_constant_vert"
               OpName %15 "PushConstantFrag"
               OpMemberName %15 0 "val"
               OpName %16 "push_constant_frag"
               OpName %8 "colour"
               OpName %17 "tex"
               OpMemberDecorate %10 0 Offset 0
               OpDecorate %10 Block
               OpDecorate %11 DescriptorSet 0
               OpDecorate %11 Binding 1
               OpDecorate %5 Location 0
               OpDecorate %4 Location 0
               OpMemberDecorate %12 0 BuiltIn Position
               OpDecorate %12 Block
               OpDecorate %6 Location 1
               OpMemberDecorate %13 0 Offset 0
               OpDecorate %13 Block
               OpDecorate %8 Location 0
               OpDecorate %17 DescriptorSet 0
               OpDecorate %17 Binding 0
         %18 = OpTypeVoid
         %19 = OpTypeFunction %18
         %20 = OpTypeFloat 32
         %21 = OpTypeVector %20 4
         %22 = OpTypeFunction %21
         %10 = OpTypeStruct %21
         %23 = OpTypePointer Uniform %10
         %11 = OpVariable %23 Uniform
         %24 = OpTypeInt 32 1
         %25 = OpConstant %24 0
         %26 = OpTypePointer Uniform %21
         %27 = OpTypeVector %20 2
         %28 = OpTypePointer Output %27
          %5 = OpVariable %28 Output
         %29 = OpTypePointer Input %27
          %4 = OpVariable %29 Input
         %12 = OpTypeStruct %21
         %30 = OpTypePointer Output %12
          %3 = OpVariable %30 Output
         %31 = OpTypeVector %20 3
         %32 = OpTypePointer Input %31
          %6 = OpVariable %32 Input
         %33 = OpConstant %20 1
         %13 = OpTypeStruct %20
         %34 = OpTypePointer PushConstant %13
         %14 = OpVariable %34 PushConstant
         %15 = OpTypeStruct %20
         %35 = OpTypePointer PushConstant %15
         %16 = OpVariable %35 PushConstant
         %36 = OpTypePointer PushConstant %20
         %37 = OpTypePointer Output %21
          %8 = OpVariable %37 Output
         %38 = OpTypeImage %20 2D 0 0 0 1 Unknown
         %39 = OpTypeSampledImage %38
         %40 = OpTypePointer UniformConstant %39
         %17 = OpVariable %40 UniformConstant
         %41 = OpConstant %20 0
          %9 = OpFunction %21 None %22
         %42 = OpLabel
         %43 = OpAccessChain %26 %11 %25
         %44 = OpLoad %21 %43
               OpReturnValue %44
               OpFunctionEnd
          %2 = OpFunction %18 None %19
         %45 = OpLabel
         %46 = OpLoad %27 %4
               OpStore %5 %46
         %47 = OpLoad %31 %6
         %48 = OpCompositeExtract %20 %47 0
         %49 = OpCompositeExtract %20 %47 1
         %50 = OpCompositeExtract %20 %47 2
         %51 = OpCompositeConstruct %21 %48 %49 %50 %33
         %52 = OpAccessChain %36 %14 %25
         %53 = OpLoad %20 %52
         %54 = OpVectorTimesScalar %21 %51 %53
         %55 = OpFunctionCall %21 %9
         %56 = OpFMul %21 %54 %55
         %57 = OpAccessChain %37 %3 %25
               OpStore %57 %56
               OpReturn
               OpFunctionEnd
          %7 = OpFunction %18 None %19
         %58 = OpLabel
         %59 = OpLoad %39 %17
         %60 = OpLoad %27 %4
         %61 = OpImageSampleExplicitLod %21 %59 %60 Lod %41
         %62 = OpAccessChain %36 %16 %25
         %63 = OpLoad %20 %62
         %64 = OpVectorTimesScalar %21 %61 %63
         %65 = OpFunctionCall %21 %9
         %66 = OpFMul %21 %64 %65
               OpStore %8 %66
               OpReturn
               OpFunctionEnd`
)
