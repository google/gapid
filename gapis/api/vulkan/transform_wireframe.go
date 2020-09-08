// Copyright (C) 2020 Google Inc.
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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

// wireframeTransform implements a transform that sets each graphics pipeline
// to be created with rasterization polygon mode == VK_POLYGON_MODE_LINE
type wireframeTransform struct {
	allocations *allocationTracker
}

func newWireframeTransform() *wireframeTransform {
	return &wireframeTransform{
		allocations: nil,
	}
}

func (wireframe *wireframeTransform) RequiresAccurateState() bool {
	return false
}

func (wireframe *wireframeTransform) RequiresInnerStateMutation() bool {
	return false
}

func (wireframe *wireframeTransform) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (wireframe *wireframeTransform) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	wireframe.allocations = NewAllocationTracker(inputState)
	return nil
}

func (wireframe *wireframeTransform) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (wireframe *wireframeTransform) ClearTransformResources(ctx context.Context) {
	wireframe.allocations.FreeAllocations()
}

func (wireframe *wireframeTransform) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	for i, cmd := range inputCommands {
		if createGraphicsPipelinesCmd, ok := cmd.(*VkCreateGraphicsPipelines); ok {
			modifiedCmd := wireframe.updateGraphicsPipelines(ctx, createGraphicsPipelinesCmd, inputState)
			if modifiedCmd != nil {
				inputCommands[i] = modifiedCmd
			}
		}
	}

	return inputCommands, nil
}

func (wireframe *wireframeTransform) updateGraphicsPipelines(ctx context.Context, cmd *VkCreateGraphicsPipelines, inputState *api.GlobalState) api.Cmd {
	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	count := uint64(cmd.CreateInfoCount())
	infos := cmd.PCreateInfos().Slice(0, count, inputState.MemoryLayout)
	newInfos := make([]VkGraphicsPipelineCreateInfo, count)

	newRasterStateDatas := make([]api.AllocResult, count)
	for i := uint64(0); i < count; i++ {
		info := infos.Index(i).MustRead(ctx, cmd, inputState, nil)[0]
		rasterState := info.PRasterizationState().MustRead(ctx, cmd, inputState, nil)
		rasterState.SetPolygonMode(VkPolygonMode_VK_POLYGON_MODE_LINE)
		newRasterStateDatas[i] = wireframe.allocations.AllocDataOrPanic(ctx, rasterState)
		info.SetPRasterizationState(NewVkPipelineRasterizationStateCreateInfoᶜᵖ(newRasterStateDatas[i].Ptr()))
		newInfos[i] = info
	}
	newInfosData := wireframe.allocations.AllocDataOrPanic(ctx, newInfos)

	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkCreateGraphicsPipelines(cmd.Device(),
		cmd.PipelineCache(), cmd.CreateInfoCount(), newInfosData.Ptr(),
		cmd.PAllocator(), cmd.PPipelines(), cmd.Result()).AddRead(newInfosData.Data())

	for _, r := range newRasterStateDatas {
		newCmd.AddRead(r.Data())
	}

	for _, w := range cmd.Extras().Observations().Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}
	return newCmd
}
