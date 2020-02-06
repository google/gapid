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

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/messages"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service"
	"github.com/google/gapid/gapis/service/path"
)

type customState struct {
	SubCmdIdx         api.SubCmdIdx
	CurrentSubmission api.Cmd
	PreSubcommand     func(interface{})
	PostSubcommand    func(interface{})
	AddCommand        func(interface{})
	IsRebuilding      bool
	pushMarkerGroup   func(name string, next bool, ty MarkerType)
	popMarkerGroup    func(ty MarkerType)
	queuedCommands    map[CommandReferenceʳ]QueuedCommand
	initialCommands   map[VkCommandBuffer][]api.Cmd
}

func (c *customState) init(s *State) {
	c.queuedCommands = make(map[CommandReferenceʳ]QueuedCommand)
	c.initialCommands = make(map[VkCommandBuffer][]api.Cmd)

	for b, cb := range s.CommandBuffers().All() {
		existingCommands := cb.CommandReferences().Len()
		c.initialCommands[b] = make([]api.Cmd, existingCommands)
	}
}

func getStateObject(s *api.GlobalState) *State {
	return GetState(s)
}

type VulkanContext struct{}

// Name returns the display-name of the context.
func (VulkanContext) Name() string {
	return "Vulkan Context"
}

// ID returns the context's unique identifier.
func (VulkanContext) ID() api.ContextID {
	// ID returns the context's unique identifier
	return api.ContextID{1}
}

// API returns the vulkan API.
func (VulkanContext) API() api.API {
	return API{}
}

func (API) Context(ctx context.Context, s *api.GlobalState, thread uint64) api.Context {
	return VulkanContext{}
}

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (*State) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	return p, nil
}

// SetupInitialState recreates the command lamdas from the state block.
// These are not encoded so we have to set them up here.
func (s *State) SetupInitialState(ctx context.Context) {
	s.customState.init(s)
}

func (State) preMutate(ctx context.Context, s *api.GlobalState, cmd api.Cmd) error {
	return nil
}

func (API) GetFramebufferAttachmentInfo(
	ctx context.Context,
	after []uint64,
	state *api.GlobalState,
	thread uint64,
	attachment api.FramebufferAttachment) (info api.FramebufferAttachmentInfo, err error) {

	w, h, form, i, r, err := GetState(state).getFramebufferAttachmentInfo(attachment)
	if err != nil {
		return api.FramebufferAttachmentInfo{}, err
	}

	switch attachment {
	case api.FramebufferAttachment_Stencil:
		return api.FramebufferAttachmentInfo{}, fmt.Errorf("Unsupported Stencil")
	case api.FramebufferAttachment_Depth:
		format, err := getDepthImageFormatFromVulkanFormat(form)
		if err != nil {
			return api.FramebufferAttachmentInfo{}, fmt.Errorf("Unknown format for Depth attachment: %v", form)
		}
		return api.FramebufferAttachmentInfo{w, h, i, format, r}, err
	default:
		format, err := getImageFormatFromVulkanFormat(form)
		if err != nil {
			return api.FramebufferAttachmentInfo{}, fmt.Errorf("Unknown format for Color attachment: %v", form)
		}
		return api.FramebufferAttachmentInfo{w, h, i, format, r}, err
	}
}

// Mesh implements the api.MeshProvider interface
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	switch dc := o.(type) {
	case *VkQueueSubmit:
		return drawCallMesh(ctx, dc, p, r)
	}
	return nil, &service.ErrDataUnavailable{Reason: messages.ErrMeshNotAvailable()}
}

type MarkerType int

const (
	DebugMarker = iota
	RenderPassMarker
)

type markerInfo struct {
	name   string
	ty     MarkerType
	start  uint64
	end    uint64
	parent api.SubCmdIdx
}

func (API) ResolveSynchronization(ctx context.Context, d *sync.Data, c *path.Capture) error {
	ctx = status.Start(ctx, "vulkan.ResolveSynchronization")
	defer status.Finish(ctx)
	ctx = capture.Put(ctx, c)
	s, err := capture.NewState(ctx)
	if err != nil {
		return err
	}
	cmds, err := resolve.Cmds(ctx, c)
	if err != nil {
		return err
	}
	st := GetState(s)
	l := s.MemoryLayout

	i := api.CmdID(0)
	// Prepare for collect marker groups
	// Stacks of open markers for each VkQueue
	markerStack := []*markerInfo{}

	commandMap := make(map[api.Cmd]api.CmdID)
	st.AddCommand = func(a interface{}) {
		data := a.(CommandReferenceʳ)
		if initialCommands, ok := st.initialCommands[data.Buffer()]; ok {
			commandMap[initialCommands[data.CommandIndex()]] = i
		}
	}

	popMarker := func(ty MarkerType, id uint64, nCommands uint64) {
		numMarkers := len(markerStack)
		inStack := false
		for i := numMarkers - 1; i >= 0 && len(markerStack) > 0; i-- {
			marker := markerStack[len(markerStack)-1]
			if marker.ty == ty {
				inStack = true
				break
			}
		}
		if !inStack {
			return
		}

		if id < nCommands-1 {
			id++
		}
		for i := numMarkers - 1; i >= 0 && len(markerStack) > 0; i-- {
			marker := markerStack[len(markerStack)-1]
			d.SubCommandMarkerGroups.NewMarkerGroup(marker.parent, marker.name, marker.start, id)
			markerStack = markerStack[0 : len(markerStack)-1]
			if marker.ty != ty {
				break
			}
		}
	}
	var walkCommandBuffer func(cb CommandBufferObjectʳ, idx api.SubCmdIdx) ([]sync.SubcommandReference, []api.SubCmdIdx)
	walkCommandBuffer = func(cb CommandBufferObjectʳ, idx api.SubCmdIdx) ([]sync.SubcommandReference, []api.SubCmdIdx) {
		refs := make([]sync.SubcommandReference, 0)
		subgroups := make([]api.SubCmdIdx, 0)
		lastSubpass := 0
		nCommands := uint64(cb.CommandReferences().Len())
		for i := 0; i < cb.CommandReferences().Len(); i++ {
			initialCommands, ok := st.initialCommands[cb.VulkanHandle()]
			var ref sync.SubcommandReference
			if !ok {
				continue
			}
			nv := append(api.SubCmdIdx{}, idx...)
			nv = append(nv, uint64(i))
			generatingId := initialCommands[i]
			if generatingId == nil {
				ref = sync.SubcommandReference{
					append(api.SubCmdIdx{}, nv[1:]...),
					api.CmdNoID,
					cb.CommandReferences().Get(uint32(i)),
					false,
				}
			} else {
				ref = sync.SubcommandReference{
					append(api.SubCmdIdx{}, nv[1:]...),
					commandMap[generatingId],
					nil,
					false,
				}
			}
			d.SubcommandLookup.SetValue(nv, ref)
			refs = append(refs, ref)

			switch args := GetCommandArgs(ctx, cb.CommandReferences().Get(uint32(i)), st).(type) {
			case VkCmdExecuteCommandsArgsʳ:
				for j := uint64(0); j < uint64(args.CommandBuffers().Len()); j++ {
					cbo := st.CommandBuffers().Get(args.CommandBuffers().Get(uint32(j)))
					subIdx := append(api.SubCmdIdx{}, idx...)
					subIdx = append(subIdx, uint64(i), j)
					newRefs, newSubgroups := walkCommandBuffer(cbo, subIdx)
					refs = append(refs, newRefs...)
					subgroups = append(subgroups, newSubgroups...)
					if cbo.CommandReferences().Len() > 0 {
						subgroups = append(subgroups, append(idx, uint64(i), uint64(j), uint64(cbo.CommandReferences().Len()-1)))
					}
				}
			case VkCmdBeginRenderPassArgsʳ:
				rp := st.RenderPasses().Get(args.RenderPass())
				name := fmt.Sprintf("RenderPass: %v", rp.VulkanHandle())
				if !rp.DebugInfo().IsNil() && len(rp.DebugInfo().ObjectName()) > 0 {
					name = rp.DebugInfo().ObjectName()
				}

				markerStack = append(markerStack,
					&markerInfo{
						name:   name,
						ty:     RenderPassMarker,
						start:  uint64(i),
						end:    uint64(i),
						parent: append(api.SubCmdIdx{}, idx...),
					})
				lastSubpass = 0
				if rp.SubpassDescriptions().Len() > 1 {
					name = fmt.Sprintf("Subpass: %v", lastSubpass)
					markerStack = append(markerStack,
						&markerInfo{
							name:   name,
							ty:     RenderPassMarker,
							start:  uint64(i),
							end:    uint64(i),
							parent: append(api.SubCmdIdx{}, idx...),
						})
				}
				break
			case VkCmdEndRenderPassArgsʳ:
				popMarker(RenderPassMarker, uint64(i), nCommands)
				break
			case VkCmdNextSubpassArgsʳ:
				lastSubpass++
				popMarker(RenderPassMarker, uint64(i), nCommands)
				name := fmt.Sprintf("Subpass: %v", lastSubpass)
				markerStack = append(markerStack,
					&markerInfo{
						name:   name,
						ty:     RenderPassMarker,
						start:  uint64(i),
						end:    uint64(i),
						parent: append(api.SubCmdIdx{}, idx...),
					})
			case VkCmdDebugMarkerBeginEXTArgsʳ:
				markerStack = append(markerStack,
					&markerInfo{
						name:   args.MarkerName(),
						ty:     DebugMarker,
						start:  uint64(i),
						end:    uint64(i),
						parent: append(api.SubCmdIdx{}, idx...),
					})
			case VkCmdBeginDebugUtilsLabelEXTArgsʳ:
				markerStack = append(markerStack,
					&markerInfo{
						name:   args.LabelName(),
						ty:     DebugMarker,
						start:  uint64(i),
						end:    uint64(i),
						parent: append(api.SubCmdIdx{}, idx...),
					})
			case VkCmdEndDebugUtilsLabelEXTArgsʳ:
				popMarker(DebugMarker, uint64(i), nCommands)
			case VkCmdDebugMarkerEndEXTArgsʳ:
				popMarker(DebugMarker, uint64(i), nCommands)
			}
		}
		for i := len(markerStack) - 1; i >= 0; i-- {
			if len(markerStack[i].parent) < len(idx) {
				break
			}
			marker := markerStack[len(markerStack)-1]
			d.SubCommandMarkerGroups.NewMarkerGroup(marker.parent, marker.name, marker.start, uint64(cb.CommandReferences().Len()))
			markerStack = markerStack[0 : len(markerStack)-1]
		}
		return refs, subgroups
	}

	err = api.ForeachCmd(ctx, cmds, true, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		i = id
		if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
			return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
		}

		switch cmd := cmd.(type) {
		case *VkQueueSubmit:
			refs := []sync.SubcommandReference{}
			d.SubcommandGroups[i] = make([]api.SubCmdIdx, 0)
			submitCount := uint64(cmd.SubmitCount())
			submits := cmd.PSubmits().Slice(uint64(0), submitCount, l).MustRead(ctx, cmd, s, nil)
			for submitIdx, submit := range submits {
				bufferCount := submit.CommandBufferCount()
				buffers := submit.PCommandBuffers().Slice(uint64(0), uint64(bufferCount), l).MustRead(ctx, cmd, s, nil)
				for j, buff := range buffers {
					cmdBuff := st.CommandBuffers().Get(buff)
					// If a submitted command-buffer is empty, we shouldn't show it
					if cmdBuff.CommandReferences().Len() > 0 {
						additionalRefs, additionalSubgroups := walkCommandBuffer(cmdBuff, api.SubCmdIdx{uint64(i), uint64(submitIdx), uint64(j)})
						for _, sg := range additionalSubgroups {
							d.SubcommandGroups[i] = append(d.SubcommandGroups[i], sg[1:])
						}
						d.SubcommandGroups[i] = append(d.SubcommandGroups[i], api.SubCmdIdx{uint64(submitIdx), uint64(j), uint64(cmdBuff.CommandReferences().Len() - 1)})

						refs = append(refs, additionalRefs...)
					}
				}
			}
			d.SubcommandReferences[i] = refs
		}
		return nil
	})
	return err
}

// FlattenSubcommandIdx, when the |initialCall| is set to true, returns the
// initial command buffer recording command of the specified subcommand,
// according to the given synchronization data. If the |initialCall| is set
// to false, returns zero and indicating the flattening failed.
func (API) FlattenSubcommandIdx(idx api.SubCmdIdx, data *sync.Data, initialCall bool) (api.CmdID, bool) {
	if initialCall {
		subCmdRefVal := data.SubcommandLookup.Value(idx)
		if subCmdRefVal != nil {
			if subCmdRef, ok := subCmdRefVal.(sync.SubcommandReference); ok {
				return subCmdRef.GeneratingCmd, true
			}
		}
	}
	return api.CmdID(0), false
}

// IsTrivialTerminator returns true if the terminator is just stopping at the given index
func (API) IsTrivialTerminator(ctx context.Context, p *path.Capture, after api.SubCmdIdx) (bool, error) {
	if len(after) == 1 {
		return true, nil
	}
	return false, nil
}

// RecoverMidExecutionCommand returns a virtual command, used to describe the
// a subcommand that was created before the start of the trace
func (API) RecoverMidExecutionCommand(ctx context.Context, c *path.Capture, dat interface{}) (api.Cmd, error) {
	cr, ok := dat.(CommandReferenceʳ)
	if !ok {
		return nil, fmt.Errorf("Not a command reference")
	}

	ctx = capture.Put(ctx, c)
	st, err := capture.NewState(ctx)
	if err != nil {
		return nil, err
	}
	s := GetState(st)

	cb := CommandBuilder{Thread: 0, Arena: st.Arena}
	_, a, err := AddCommand(ctx, cb, cr.Buffer(), st, st, GetCommandArgs(ctx, cr, s))
	if err != nil {
		return nil, log.Errf(ctx, err, "Invalid Command")
	}
	return a, nil
}

// Interface check
var _ sync.SynchronizedAPI = &API{}

func (API) GetTerminator(ctx context.Context, c *path.Capture) (transform.Terminator, error) {
	return NewVulkanTerminator(ctx, c)
}

func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd,
	s *api.GlobalState, preSubCmdCb func(*api.GlobalState, api.SubCmdIdx, api.Cmd),
	postSubCmdCb func(*api.GlobalState, api.SubCmdIdx, api.Cmd)) error {
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
	if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
		return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
	}
	return nil
}
