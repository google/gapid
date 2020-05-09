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
	"context"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
)

type profilingLayers struct {
	layerName string
}

func (t *profilingLayers) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	ctx = log.Enter(ctx, "ProfilingLayers")

	s := out.State()
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: out.State().Arena}
	allocated := []api.AllocResult{}
	defer func() {
		for _, d := range allocated {
			d.Free()
		}
	}()
	mustAlloc := func(ctx context.Context, v ...interface{}) api.AllocResult {
		res := s.AllocDataOrPanic(ctx, v...)
		allocated = append(allocated, res)
		return res
	}

	switch cmd := cmd.(type) {
	case *VkCreateInstance:
		cmd.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
		info := cmd.PCreateInfo().MustRead(ctx, cmd, s, nil)
		// Strip all instance layers that were originally present. If the device wants
		// a layer in order to support collecting renderstages, then add that layer only.
		layers := []Charᶜᵖ{}

		renderStagesLayerData := mustAlloc(ctx, t.layerName)
		if t.layerName != "" {
			layers = append(layers, NewCharᶜᵖ(renderStagesLayerData.Ptr()))
		}
		layersData := mustAlloc(ctx, layers)

		info.SetEnabledLayerCount(uint32(len(layers)))
		info.SetPpEnabledLayerNames(NewCharᶜᵖᶜᵖ(layersData.Ptr()))
		infoData := mustAlloc(ctx, info)

		newCmd := cb.VkCreateInstance(infoData.Ptr(), cmd.PAllocator(), cmd.PInstance(), cmd.Result())
		if t.layerName != "" {
			newCmd.AddRead(renderStagesLayerData.Data())
		}
		newCmd.AddRead(
			infoData.Data(),
		).AddRead(
			layersData.Data(),
		)
		// Also add back all the other read/write observations of the original vkCreateInstance
		for _, r := range cmd.Extras().Observations().Reads {
			newCmd.AddRead(r.Range, r.ID)
		}
		for _, w := range cmd.Extras().Observations().Writes {
			newCmd.AddWrite(w.Range, w.ID)
		}
		return out.MutateAndWrite(ctx, id, newCmd)

	default:
		return out.MutateAndWrite(ctx, id, cmd)

	}

	return nil
}

func (t *profilingLayers) PreLoop(ctx context.Context, out transform.Writer) {
	out.NotifyPreLoop(ctx)
}
func (t *profilingLayers) PostLoop(ctx context.Context, out transform.Writer) {
	out.NotifyPostLoop(ctx)
}
func (t *profilingLayers) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *profilingLayers) BuffersCommands() bool {
	return false
}
