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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service/path"
)

type CustomState struct {
	SubcommandIndex   sync.SubcommandIndex
	CurrentSubmission *api.Cmd
	HandleSubcommand  func(interface{}) `nobox:"true"`
}

func getStateObject(s *api.State) *State {
	return GetState(s)
}

type VulkanContext struct{}

func (VulkanContext) Name() string {
	return "Vulkan Context"
}

func (VulkanContext) ID() api.ContextID {
	// ID returns the context's unique identifier
	return api.ContextID{1}
}

func (API) Context(s *api.State, thread uint64) api.Context {
	return VulkanContext{}
}

func (API) GetFramebufferAttachmentInfo(state *api.State, thread uint64, attachment api.FramebufferAttachment) (w, h uint32, a uint32, f *image.Format, err error) {
	w, h, form, i, err := GetState(state).getFramebufferAttachmentInfo(attachment)
	switch attachment {
	case api.FramebufferAttachment_Stencil:
		return 0, 0, 0, nil, fmt.Errorf("Unsupported Stencil")
	case api.FramebufferAttachment_Depth:
		format, err := getDepthImageFormatFromVulkanFormat(form)
		if err != nil {
			return 0, 0, 0, nil, fmt.Errorf("Unknown format for Depth attachment")
		}
		return w, h, i, format, err
	default:
		format, err := getImageFormatFromVulkanFormat(form)
		if err != nil {
			return 0, 0, 0, nil, fmt.Errorf("Unknown format for Color attachment")
		}
		return w, h, i, format, err
	}
}

// Mesh implements the api.MeshProvider interface
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh) (*api.Mesh, error) {
	switch dc := o.(type) {
	case *VkQueueSubmit:
		return drawCallMesh(ctx, dc, p)
	}
	return nil, fmt.Errorf("Cannot get the mesh data from %v", o)
}

func (API) ResolveSynchronization(ctx context.Context, d *sync.Data, c *path.Capture) error {
	ctx = capture.Put(ctx, c)
	st, err := capture.NewState(ctx)
	if err != nil {
		return err
	}
	cmds, err := resolve.Cmds(ctx, c)
	if err != nil {
		return err
	}
	s := GetState(st)
	i := api.CmdID(0)
	submissionMap := make(map[*api.Cmd]api.CmdID)

	s.HandleSubcommand = func(a interface{}) {
		rootIdx := api.CmdID(i)
		if k, ok := submissionMap[s.CurrentSubmission]; ok {
			rootIdx = api.CmdID(k)
		} else {
			submissionMap[s.CurrentSubmission] = i
		}

		if rng, ok := d.CommandRanges[rootIdx]; ok {
			rng.LastIndex = append(sync.SubcommandIndex(nil), s.SubcommandIndex...)
			rng.Ranges[i] = rng.LastIndex
		} else {
			er := sync.ExecutionRanges{
				LastIndex: append(sync.SubcommandIndex(nil), s.SubcommandIndex...),
				Ranges:    make(map[api.CmdID]sync.SubcommandIndex),
			}
			er.Ranges[i] = append(sync.SubcommandIndex(nil), s.SubcommandIndex...)
			d.CommandRanges[rootIdx] = er
		}
	}

	for idx, cmd := range cmds {
		i = api.CmdID(idx)
		if err := cmd.Mutate(ctx, st, nil); err != nil {
			return err
		}
	}
	return nil
}

// Interface check
var _ sync.SynchronizedAPI = &API{}

func (API) GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error) {
	return NewVulkanTerminator(ctx, c)
}

// GetDependencyGraphBehaviourProvider implements dependencygraph.DependencyGraphBehaviourProvider interface
func (API) GetDependencyGraphBehaviourProvider(ctx context.Context) dependencygraph.BehaviourProvider {
	return newVulkanDependencyGraphBehaviourProvider()
}

func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd, s *api.State, callback func(*api.State, sync.SubcommandIndex, api.Cmd)) error {
	c := GetState(s)
	c.HandleSubcommand = func(_ interface{}) {
		callback(s, append(sync.SubcommandIndex{uint64(id)}, c.SubcommandIndex...), cmd)
	}
	if err := cmd.Mutate(ctx, s, nil); err != nil && err == context.Canceled {
		return err
	}
	return nil
}
