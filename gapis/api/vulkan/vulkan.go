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
	SubCmdIdx         api.SubCmdIdx
	CurrentSubmission *api.Cmd
	PreSubcommand     func(interface{})
	PostSubcommand    func(interface{})
	AddCommand        func(interface{})
	IsRebuilding      bool
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

func (s *State) SetThread(thread uint64) {
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
	commandMap := make(map[*api.Cmd]api.CmdID)
	lastSubcommand := api.SubCmdIdx{}
	lastCmdIndex := api.CmdID(0)

	s.PostSubcommand = func(a interface{}) {
		// We do not record/handle any subcommands inside any of our
		// rebuild commands
		if s.IsRebuilding {
			return
		}

		data := a.(CommandBufferCommand)
		rootIdx := api.CmdID(i)
		if k, ok := submissionMap[s.CurrentSubmission]; ok {
			rootIdx = api.CmdID(k)
		} else {
			submissionMap[s.CurrentSubmission] = i
		}
		// No way for this to not exist, we put it in up there
		k := submissionMap[s.CurrentSubmission]
		if v, ok := d.SubcommandReferences[k]; ok {
			v = append(v,
				sync.SubcommandReference{append(api.SubCmdIdx(nil), s.SubCmdIdx...), commandMap[data.initialCall]})
			d.SubcommandReferences[k] = v
		} else {
			d.SubcommandReferences[k] = []sync.SubcommandReference{
				sync.SubcommandReference{append(api.SubCmdIdx(nil), s.SubCmdIdx...), commandMap[data.initialCall]}}
		}

		previousIndex := append(api.SubCmdIdx(nil), s.SubCmdIdx...)
		previousIndex.Decrement()
		if !previousIndex.Equals(lastSubcommand) && lastCmdIndex != api.CmdID(0) {
			if v, ok := d.SubcommandGroups[lastCmdIndex]; ok {
				v = append(v, append(api.SubCmdIdx(nil), lastSubcommand...))
				d.SubcommandGroups[lastCmdIndex] = v
			} else {
				d.SubcommandGroups[lastCmdIndex] = []api.SubCmdIdx{append(api.SubCmdIdx(nil), lastSubcommand...)}
			}
			lastSubcommand = append(api.SubCmdIdx(nil), s.SubCmdIdx...)
			lastCmdIndex = k
		} else {
			lastSubcommand = append(api.SubCmdIdx(nil), s.SubCmdIdx...)
			lastCmdIndex = k
		}

		if rng, ok := d.CommandRanges[rootIdx]; ok {
			rng.LastIndex = append(api.SubCmdIdx(nil), s.SubCmdIdx...)
			rng.Ranges[i] = rng.LastIndex
			d.CommandRanges[rootIdx] = rng
		} else {
			er := sync.ExecutionRanges{
				LastIndex: append(api.SubCmdIdx(nil), s.SubCmdIdx...),
				Ranges:    make(map[api.CmdID]api.SubCmdIdx),
			}
			er.Ranges[i] = append(api.SubCmdIdx(nil), s.SubCmdIdx...)
			d.CommandRanges[rootIdx] = er
		}
	}

	s.AddCommand = func(a interface{}) {
		data := a.(CommandBufferCommand)
		commandMap[data.initialCall] = i
	}

	err = api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		i = id
		cmd.Mutate(ctx, st, nil)
		return nil
	})
	if err != nil {
		return err
	}

	if lastCmdIndex != api.CmdID(0) {
		if v, ok := d.SubcommandGroups[lastCmdIndex]; ok {
			v = append(v, append(api.SubCmdIdx(nil), lastSubcommand...))
			d.SubcommandGroups[lastCmdIndex] = v
		} else {
			d.SubcommandGroups[lastCmdIndex] = []api.SubCmdIdx{
				append(api.SubCmdIdx(nil), lastSubcommand...)}
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

func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd,
	s *api.State, preSubCmdCb func(*api.State, api.SubCmdIdx, api.Cmd),
	postSubCmdCb func(*api.State, api.SubCmdIdx, api.Cmd)) error {
	c := GetState(s)
	if postSubCmdCb != nil {
		c.PostSubcommand = func(_ interface{}) {
			postSubCmdCb(s, append(api.SubCmdIdx{uint64(id)}, c.SubCmdIdx...), cmd)
		}
	}
	if preSubCmdCb != nil {
		c.PreSubcommand = func(_ interface{}) {
			preSubCmdCb(s, append(api.SubCmdIdx{uint64(id)}, c.SubCmdIdx...), cmd)
		}
	}
	if err := cmd.Mutate(ctx, s, nil); err != nil && err == context.Canceled {
		return err
	}
	return nil
}
