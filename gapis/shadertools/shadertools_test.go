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

func TestConvertGlsl(t *testing.T) {
	for _, test := range []struct {
		desc     string
		opts     shadertools.ConvertOptions
		src      string
		expected string
	}{
		{
			"Test shadertools propagates early_fragment_tests",
			shadertools.ConvertOptions{
				ShaderType:        shadertools.TypeFragment,
				CheckAfterChanges: true,
			},
			`#version 310 es
layout(early_fragment_tests) in;
out highp vec4 color;
void main() { color = vec4(1., 1., 1., 1.); }`,
			`#version 330
#ifdef GL_ARB_shading_language_420pack
#extension GL_ARB_shading_language_420pack : require
#endif
#extension GL_ARB_shader_image_load_store : require
layout(early_fragment_tests) in;

out vec4 color;

void main()
{
    color = vec4(1.0);
}

`}, {
			"Test shadertools strips early_fragment_tests",
			shadertools.ConvertOptions{
				ShaderType:         shadertools.TypeFragment,
				CheckAfterChanges:  true,
				StripOptimizations: true,
			},
			`#version 310 es
layout(early_fragment_tests) in;
out highp vec4 color;
void main() { color = vec4(1., 1., 1., 1.); }`,
			`#version 330
#ifdef GL_ARB_shading_language_420pack
#extension GL_ARB_shading_language_420pack : require
#endif

out vec4 color;

void main()
{
    color = vec4(1.0);
}

`},
	} {
		t.Run(test.desc, func(t *testing.T) {
			ctx := log.Testing(t)
			out, err := shadertools.ConvertGlsl(test.src, &test.opts)
			if assert.For(ctx, "err").ThatError(err).Succeeded() {
				assert.For(ctx, "src").ThatString(out.SourceCode).Equals(test.expected)
			}
		})
	}
}

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
				0x00000012, 0x4b7ffff0, 0x00050036, 0x00000005,
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
