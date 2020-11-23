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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

var _ transform.Transform = &profilingLayers{}

type profilingLayers struct {
	allocations *allocationTracker
	layerName   string
}

func newProfilingLayers(layerName string) *profilingLayers {
	return &profilingLayers{
		allocations: nil,
		layerName:   layerName,
	}
}

func (profilingLayerTransform *profilingLayers) RequiresAccurateState() bool {
	return false
}

func (profilingLayerTransform *profilingLayers) RequiresInnerStateMutation() bool {
	return false
}

func (profilingLayerTransform *profilingLayers) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	// This transform do not require inner state mutation
}

func (profilingLayerTransform *profilingLayers) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	profilingLayerTransform.allocations = NewAllocationTracker(inputState)
	return nil
}

func (profilingLayerTransform *profilingLayers) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (profilingLayerTransform *profilingLayers) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	for i, cmd := range inputCommands {
		if createInstanceCommand, ok := cmd.(*VkCreateInstance); ok {
			modifiedCmd := profilingLayerTransform.addProfilingLayersToCreateInstance(ctx, createInstanceCommand, inputState)
			if modifiedCmd != nil {
				inputCommands[i] = modifiedCmd
			}
		}
	}
	return inputCommands, nil
}

func (profilingLayerTransform *profilingLayers) ClearTransformResources(ctx context.Context) {
	profilingLayerTransform.allocations.FreeAllocations()
}

func (profilingLayerTransform *profilingLayers) addProfilingLayersToCreateInstance(ctx context.Context, createInstanceCommand *VkCreateInstance, inputState *api.GlobalState) api.Cmd {
	ctx = log.Enter(ctx, "ProfilingLayers")

	createInstanceCommand.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())
	info := createInstanceCommand.PCreateInfo().MustRead(ctx, createInstanceCommand, inputState, nil)
	// Strip all instance layers that were originally present. If the device wants
	// a layer in order to support collecting renderstages, then add that layer only.

	layers := []Charᶜᵖ{}
	renderStagesLayerData := profilingLayerTransform.allocations.AllocDataOrPanic(ctx, profilingLayerTransform.layerName)
	if profilingLayerTransform.layerName != "" {
		layers = append(layers, NewCharᶜᵖ(renderStagesLayerData.Ptr()))
	}

	layersData := profilingLayerTransform.allocations.AllocDataOrPanic(ctx, layers)

	info.SetEnabledLayerCount(uint32(len(layers)))
	info.SetPpEnabledLayerNames(NewCharᶜᵖᶜᵖ(layersData.Ptr()))
	infoData := profilingLayerTransform.allocations.AllocDataOrPanic(ctx, info)

	cb := CommandBuilder{Thread: createInstanceCommand.Thread()}
	newCmd := cb.VkCreateInstance(infoData.Ptr(), createInstanceCommand.PAllocator(), createInstanceCommand.PInstance(), createInstanceCommand.Result())

	if profilingLayerTransform.layerName != "" {
		newCmd.AddRead(renderStagesLayerData.Data())
	}

	newCmd.AddRead(infoData.Data()).AddRead(layersData.Data())

	// Also add back all the other read/write observations of the original vkCreateInstance
	for _, r := range createInstanceCommand.Extras().Observations().Reads {
		newCmd.AddRead(r.Range, r.ID)
	}
	for _, w := range createInstanceCommand.Extras().Observations().Writes {
		newCmd.AddWrite(w.Range, w.ID)
	}
	return newCmd
}
