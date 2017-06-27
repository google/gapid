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
	"fmt"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/atom/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/gfxapi"
	"github.com/google/gapid/gapis/gfxapi/synchronization"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

type CustomState struct {
	SubcommandIndex   synchronization.SubcommandIndex
	CurrentSubmission *atom.Atom
	HandleSubcommand  func(interface{}) `nobox:"true"`
}

func getStateObject(s *gfxapi.State) *State {
	return GetState(s)
}

type VulkanContext struct{}

func (VulkanContext) Name() string {
	return "Vulkan Context"
}

func (VulkanContext) ID() gfxapi.ContextID {
	// ID returns the context's unique identifier
	return gfxapi.ContextID{1}
}

func (api) Context(s *gfxapi.State) gfxapi.Context {
	return VulkanContext{}
}

func (api) GetFramebufferAttachmentInfo(state *gfxapi.State, attachment gfxapi.FramebufferAttachment) (w, h uint32, f *image.Format, err error) {
	w, h, form, _, err := GetState(state).getFramebufferAttachmentInfo(attachment)
	switch attachment {
	case gfxapi.FramebufferAttachment_Stencil:
		return 0, 0, nil, fmt.Errorf("Unsupported Stencil")
	case gfxapi.FramebufferAttachment_Depth:
		format, err := getDepthImageFormatFromVulkanFormat(form)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("Unknown format for Depth attachment")
		}
		return w, h, format, err
	default:
		format, err := getImageFormatFromVulkanFormat(form)
		if err != nil {
			return 0, 0, nil, fmt.Errorf("Unknown format for Color attachment")
		}
		return w, h, format, err
	}
}

// Mesh implements the gfxapi.MeshProvider interface
func (api) Mesh(ctx context.Context, o interface{}, p *path.Mesh) (*gfxapi.Mesh, error) {
	switch dc := o.(type) {
	case *VkQueueSubmit:
		return drawCallMesh(ctx, dc, p)
	}
	return nil, fmt.Errorf("Cannot get the mesh data from %v", o)
}

func (api) ResolveSynchronization(ctx context.Context, d *synchronization.SynchronizationData, c *path.Capture) error {
	ctx = capture.Put(ctx, c)
	st, err := capture.NewState(ctx)
	if err != nil {
		return err
	}
	a, err := resolve.Atoms(ctx, c)
	if err != nil {
		return err
	}
	s := GetState(st)
	i := synchronization.SynchronizationIndex(0)
	submissionMap := make(map[*atom.Atom]synchronization.SynchronizationIndex)

	s.HandleSubcommand = func(a interface{}) {
		rootIdx := synchronization.SynchronizationIndex(i)
		if k, ok := submissionMap[s.CurrentSubmission]; ok {
			rootIdx = synchronization.SynchronizationIndex(k)
		} else {
			submissionMap[s.CurrentSubmission] = i
		}

		if rng, ok := d.CommandRanges[rootIdx]; ok {
			rng.LastIndex = append(synchronization.SubcommandIndex(nil), s.SubcommandIndex...)
			rng.Ranges[i] = rng.LastIndex
		} else {
			er := synchronization.ExecutionRanges{
				LastIndex: append(synchronization.SubcommandIndex(nil), s.SubcommandIndex...),
				Ranges:    make(map[synchronization.SynchronizationIndex]synchronization.SubcommandIndex),
			}
			er.Ranges[i] = append(synchronization.SubcommandIndex(nil), s.SubcommandIndex...)
			d.CommandRanges[rootIdx] = er
		}
	}

	for idx, a := range a.Atoms {
		i = synchronization.SynchronizationIndex(idx)
		if err := a.Mutate(ctx, st, nil); err != nil {
			return err
		}
	}
	return nil
}

// Interface check
var _ synchronization.SynchronizedApi = &api{}

func (api) GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error) {
	return NewVulkanTerminator(ctx, c)
}
