// Copyright (C) 2019 Google Inc.
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
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/os/device"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/shadertools"
)

const opEntryPoint uint32 = 15
const opFragmentExecutionMode uint32 = 4
const constantColorShader string = `
#version 330
layout (location = 0) out vec4 fragColor;
void main() {
   fragColor = vec4(0, 0, 1, 1);
}`

func isFragmentShader(ctx context.Context, info VkShaderModuleCreateInfo, l *device.MemoryLayout, s *api.GlobalState) bool {
	codeSize := uint64(info.CodeSize()) / 4 // num bytes to num uint32
	code := info.PCode().Slice(0, codeSize, l).MustRead(ctx, nil, s, nil)

	i := uint64(5) // Instructions start at word 5
	for i < codeSize {
		instruction := code[i]                 // uint32
		wordCount := uint64(instruction >> 16) // Upper 16 bits
		opCode := (instruction << 16) >> 16    // Lower 16 bits

		if opCode == opEntryPoint {
			opExecutionMode := code[i+1]
			return opExecutionMode == opFragmentExecutionMode
		}

		i += wordCount
	}

	panic("No shader entry point found.")
}

// simplifyFragmentShader returns a transform that replaces fragment shaders with a constant color shader
func simplifyFragmentShader(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "simplifyFragmentShader")
	return transform.Transform("simplifyFragmentShader", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) {

		s := out.State()
		l := s.MemoryLayout
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}
		switch cmd := cmd.(type) {
		case *VkCreateShaderModule:
			oldCreateInfo := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
			if isFragmentShader(ctx, oldCreateInfo, l, s) {
				shaderSource, _ := shadertools.CompileGlsl(
					constantColorShader,
					shadertools.CompileOptions{
						ShaderType: shadertools.TypeFragment,
						ClientType: shadertools.Vulkan,
					},
				)
				shaderData := s.AllocDataOrPanic(ctx, shaderSource)
				defer shaderData.Free()

				createInfo := NewVkShaderModuleCreateInfo(s.Arena,
					oldCreateInfo.SType(),            // sType
					oldCreateInfo.PNext(),            // pNext
					oldCreateInfo.Flags(),            // flags
					memory.Size(len(shaderSource)*4), // codeSize
					NewU32ᶜᵖ(shaderData.Ptr()),       // pCode
				)
				createInfoData := s.AllocDataOrPanic(ctx, createInfo)
				defer createInfoData.Free()

				newCmd := cb.VkCreateShaderModule(
					cmd.Device(),
					createInfoData.Ptr(),
					cmd.PAllocator(),
					cmd.PShaderModule(),
					VkResult_VK_SUCCESS,
				).AddRead(
					createInfoData.Data(),
				).AddRead(
					shaderData.Data(),
				)

				for _, w := range cmd.extras.Observations().Writes {
					newCmd.AddWrite(w.Range, w.ID)
				}
				out.MutateAndWrite(ctx, id, newCmd)
			} else {
				out.MutateAndWrite(ctx, id, cmd)
			}
		default:
			out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
