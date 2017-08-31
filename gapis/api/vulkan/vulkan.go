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
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
	"github.com/google/gapid/gapis/service/path"
)

type CustomState struct {
	SubCmdIdx            api.SubCmdIdx
	CurrentSubmission    *api.Cmd
	PreSubcommand        func(interface{})
	PostSubcommand       func(interface{})
	AddCommand           func(interface{})
	IsRebuilding         bool
	pushDebugMarkerGroup func(name string)
	popDebugMarkerGroup  func()
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

func (c *State) preMutate(ctx context.Context, s *api.State, cmd api.Cmd) error {
	return nil
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

	// Prepare for collect debug marker groups
	debugMarkerStack := map[VkQueue][]*api.SubCmdMarkerGroup{}
	s.pushDebugMarkerGroup = func(name string) {
		vkQu := (*s.CurrentSubmission).(*VkQueueSubmit).Queue
		stack := debugMarkerStack[vkQu]
		fullCmdIdx := api.SubCmdIdx{uint64(submissionMap[s.CurrentSubmission])}
		fullCmdIdx = append(fullCmdIdx, s.SubCmdIdx...)
		group := d.SubCommandMarkerGroups.NewMarkerGroup(fullCmdIdx[0:len(fullCmdIdx)-1], name)
		group.Start = api.CmdID(fullCmdIdx[len(fullCmdIdx)-1])
		debugMarkerStack[vkQu] = append(stack, group)
	}
	s.popDebugMarkerGroup = func() {
		vkQu := (*s.CurrentSubmission).(*VkQueueSubmit).Queue
		stack := debugMarkerStack[vkQu]
		if len(stack) == 0 {
			log.E(ctx, "Cannot pop debug marker, no open debug marker: VkQueueSubmit ID: %v, SubCmdIdx: %v",
				submissionMap[s.CurrentSubmission], s.SubCmdIdx)
			return
		}
		// Update the End value of the debug marker group.
		stack[len(stack)-1].End = api.CmdID(s.SubCmdIdx[len(s.SubCmdIdx)-1]) + 1
		debugMarkerStack[vkQu] = stack[0 : len(stack)-1]
	}

	s.PreSubcommand = func(interface{}) {
		// Update the submission map before execute subcommand callback and
		// postSubCommand callback.
		if _, ok := submissionMap[s.CurrentSubmission]; !ok {
			submissionMap[s.CurrentSubmission] = i
		}
		// Examine the debug marker stack. For the current submission VkQueue, if
		// the comming subcommand is submitted in a different command buffer or
		// submission batch or VkQueueSubmit call, and there are unclosed debug
		// marker group, we need to 1) check whether the unclosed debug marker
		// groups are opened in secondary command buffers, log error and pop them.
		// 2) Close all the unclosed debug marker group, and begin new groups for
		// the new command buffer.
		vkQu := (*s.CurrentSubmission).(*VkQueueSubmit).Queue
		stack := debugMarkerStack[vkQu]
		fullCmdIdx := api.SubCmdIdx{uint64(submissionMap[s.CurrentSubmission])}
		fullCmdIdx = append(fullCmdIdx, s.SubCmdIdx...)

		for lastCmdIndex != api.CmdID(0) && len(stack) > 0 {
			top := stack[len(stack)-1]
			if len(top.Parent) > len(fullCmdIdx) {
				// The top of the stack is an unclosed debug marker group which is
				// opened in a secondary command buffer. This debug marker group will
				// be closed here, the End value of the group will be the last updated
				// value (which should be one plus the last command index in its
				// secondary command buffer).
				log.E(ctx, "DebugMarker began in secondary command buffer does not close. Close now")
				stack = stack[0 : len(stack)-1]
				continue
			}
			break
		}
		// Close all the unclosed debug marker groups that are opened in previous
		// submissions or command buffers. Those closed groups will have their
		// End value to be the last updated value, and new groups with same name
		// will be opened in the new command buffer.
		if lastCmdIndex != api.CmdID(0) && len(stack) > 0 &&
			!stack[len(stack)-1].Parent.Contains(fullCmdIdx) {
			originalStack := []*api.SubCmdMarkerGroup(stack)
			debugMarkerStack[vkQu] = []*api.SubCmdMarkerGroup{}
			for _, o := range originalStack {
				s.pushDebugMarkerGroup(o.Name)
			}
		}
	}

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
				sync.SubcommandReference{append(api.SubCmdIdx(nil), s.SubCmdIdx...), commandMap[data.initialCall], false})
			d.SubcommandReferences[k] = v
		} else {
			d.SubcommandReferences[k] = []sync.SubcommandReference{
				sync.SubcommandReference{append(api.SubCmdIdx(nil), s.SubCmdIdx...), commandMap[data.initialCall], false}}
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

		// Update the End value for all unclosed debug marker groups
		vkQu := (*s.CurrentSubmission).(*VkQueueSubmit).Queue
		for _, ms := range debugMarkerStack[vkQu] {
			// If the last subcommand is in a secondary command buffer and current
			// recording debug marker groups are opened in a primary command buffer,
			// this will assign a wrong End value to the open marker groups.
			// However, those End values will be overwritten when the secondary
			// command buffer ends and vkCmdExecuteCommands get executed.
			ms.End = api.CmdID(s.SubCmdIdx[len(s.SubCmdIdx)-1] + 1)
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

func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd,
	s *api.State, preSubCmdCb func(*api.State, api.SubCmdIdx, api.Cmd),
	postSubCmdCb func(*api.State, api.SubCmdIdx, api.Cmd)) error {
	c := GetState(s)
	if postSubCmdCb != nil {
		c.PostSubcommand = func(interface{}) {
			postSubCmdCb(s, append(api.SubCmdIdx{uint64(id)}, c.SubCmdIdx...), cmd)
		}
	}
	if preSubCmdCb != nil {
		c.PreSubcommand = func(interface{}) {
			preSubCmdCb(s, append(api.SubCmdIdx{uint64(id)}, c.SubCmdIdx...), cmd)
		}
	}
	if err := cmd.Mutate(ctx, s, nil); err != nil && err == context.Canceled {
		return err
	}
	return nil
}

// FootprintBuilder implements dependencygraph.FootprintBuilderProvider interface
func (API) FootprintBuilder(ctx context.Context) dependencygraph.FootprintBuilder {
	return newFootprintBuilder()
}
