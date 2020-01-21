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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/shadertools"
)

const constantColorShader string = `
#version 330
layout (location = 0) out vec4 fragColor;
void main() {
   fragColor = vec4(0, 0, 1, 1);
}`

const nameStr = "main"

func createSimpleFragmentShaderModule(ctx context.Context,
	cb CommandBuilder,
	device VkDevice,
	out transform.Writer,
) VkShaderModule {
	s := out.State()

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
		VkStructureType_VK_STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                   // pNext
		0,                                                           // flags
		memory.Size(len(shaderSource)*4),                            // codeSize
		NewU32ᶜᵖ(shaderData.Ptr()),                                  // pCode
	)
	createInfoData := s.AllocDataOrPanic(ctx, createInfo)
	defer createInfoData.Free()

	shaderModuleHandle := VkShaderModule(newUnusedID(false, func(id uint64) bool {
		return GetState(s).ShaderModules().Contains(VkShaderModule(id))
	}))
	shaderModuleData := s.AllocDataOrPanic(ctx, shaderModuleHandle)
	defer shaderModuleData.Free()

	createShaderModuleCmd := cb.VkCreateShaderModule(
		device,
		createInfoData.Ptr(),
		memory.Nullptr,
		shaderModuleData.Ptr(),
		VkResult_VK_SUCCESS,
	).AddRead(
		createInfoData.Data(),
	).AddRead(
		shaderData.Data(),
	).AddWrite(
		shaderModuleData.Data(),
	)

	out.MutateAndWrite(ctx, api.CmdNoID, createShaderModuleCmd)

	return shaderModuleHandle
}

// simplifyFragmentShader returns a transform that replaces all
// fragment shaders with a constant color shader
func simplifyFragmentShader(ctx context.Context) transform.Transformer {
	ctx = log.Enter(ctx, "simplifyFragmentShader")

	return transform.Transform("simplifyFragmentShader", func(ctx context.Context,
		id api.CmdID, cmd api.Cmd, out transform.Writer) error {

		s := out.State()
		l := s.MemoryLayout
		cb := CommandBuilder{Thread: cmd.Thread(), Arena: s.Arena}

		switch cmd := cmd.(type) {
		case *VkCreateGraphicsPipelines:
			cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())

			shaderModuleHandle := createSimpleFragmentShaderModule(ctx, cb, cmd.Device(), out)
			nameData := s.AllocDataOrPanic(ctx, nameStr)

			createInfoCount := uint64(cmd.CreateInfoCount())
			createInfos := cmd.PCreateInfos().Slice(0, createInfoCount, l).MustRead(ctx, cmd, s, nil)
			shaderStageStateCreateInfosData := make([]api.AllocResult, createInfoCount)

			for i := uint64(0); i < createInfoCount; i++ {
				stageCount := uint64(createInfos[i].StageCount())
				shaderStageStateCreateInfos := createInfos[i].PStages().Slice(0, stageCount, l).MustRead(ctx, cmd, s, nil)
				for j := uint64(0); j < stageCount; j++ {
					if shaderStageStateCreateInfos[j].Stage() == VkShaderStageFlagBits_VK_SHADER_STAGE_FRAGMENT_BIT {
						shaderStageStateCreateInfos[j].SetPNext(NewVoidᶜᵖ(memory.Nullptr))
						shaderStageStateCreateInfos[j].SetModule(shaderModuleHandle)
						shaderStageStateCreateInfos[j].SetPName(NewCharᶜᵖ(nameData.Ptr()))
						shaderStageStateCreateInfos[j].SetPSpecializationInfo(NewVkSpecializationInfoᶜᵖ(memory.Nullptr))
					}
				}

				shaderStageStateCreateInfosData[i] = s.AllocDataOrPanic(ctx, shaderStageStateCreateInfos)
				defer shaderStageStateCreateInfosData[i].Free()

				createInfos[i].SetPStages(NewVkPipelineShaderStageCreateInfoᶜᵖ(shaderStageStateCreateInfosData[i].Ptr()))
			}

			createInfosData := s.AllocDataOrPanic(ctx, createInfos)
			defer createInfosData.Free()

			newCmd := cb.VkCreateGraphicsPipelines(
				cmd.Device(),
				cmd.PipelineCache(),
				cmd.CreateInfoCount(),
				createInfosData.Ptr(),
				cmd.PAllocator(),
				cmd.PPipelines(),
				cmd.Result(),
			).AddRead(
				createInfosData.Data(),
			)

			for _, r := range cmd.Extras().Observations().Reads {
				newCmd.AddRead(r.Range, r.ID)
			}
			for _, w := range cmd.Extras().Observations().Writes {
				newCmd.AddWrite(w.Range, w.ID)
			}

			for _, ss := range shaderStageStateCreateInfosData {
				newCmd.AddRead(ss.Data())
			}
			newCmd.AddRead(nameData.Data())

			return out.MutateAndWrite(ctx, id, newCmd)
		default:
			return out.MutateAndWrite(ctx, id, cmd)
		}
	})
}
