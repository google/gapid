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
	"github.com/google/gapid/gapis/memory"
)

// afDisablerTransform implements the Transform interface to disable anisotropic
// during every vkCreateSampler call
type afDisablerTransform struct {
	allocations *allocationTracker
}

func newAfDisablerTransform() *afDisablerTransform {
	return &afDisablerTransform{
		allocations: nil,
	}
}

func (afDisabler *afDisablerTransform) RequiresAccurateState() bool {
	return false
}

func (afDisabler *afDisablerTransform) RequiresInnerStateMutation() bool {
	return false
}

func (afDisabler *afDisablerTransform) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform does not require inner state mutation
}

func (afDisabler *afDisablerTransform) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	afDisabler.allocations = NewAllocationTracker(inputState)
	return nil
}

func (afDisabler *afDisablerTransform) ClearTransformResources(ctx context.Context) {
	afDisabler.allocations.FreeAllocations()
}

func (afDisabler *afDisablerTransform) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	for i, cmd := range inputCommands {
		if cmd, ok := cmd.(*VkCreateSampler); ok {
			inputCommands[i] = afDisabler.disableAFInSamplerCreation(ctx, cmd, inputState)
		}
	}

	return inputCommands, nil
}

func (afDisabler *afDisablerTransform) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (afDisabler *afDisablerTransform) disableAFInSamplerCreation(ctx context.Context, cmd *VkCreateSampler, inputState *api.GlobalState) api.Cmd {
	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	pAlloc := memory.Pointer(cmd.PAllocator())
	pSampler := memory.Pointer(cmd.PSampler())

	pInfo := cmd.PCreateInfo()
	info := pInfo.MustRead(ctx, cmd, inputState, nil)
	info.SetAnisotropyEnable(VkBool32(0))
	newInfo := afDisabler.allocations.AllocDataOrPanic(ctx, info)

	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	newCmd := cb.VkCreateSampler(cmd.Device(), newInfo.Ptr(), pAlloc, pSampler, cmd.Result())
	newCmd.AddRead(newInfo.Data())
	for _, w := range cmd.Extras().Observations().Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}

	return newCmd
}
