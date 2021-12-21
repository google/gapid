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
	"strings"

	"github.com/google/gapid/core/app/status"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/terminator"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

type customState struct {
	SubCmdIdx       api.SubCmdIdx
	PreSubcommand   func(interface{})
	PostSubcommand  func(interface{})
	AddCommand      func(interface{})
	queuedCommands  map[CommandReferenceʳ]QueuedCommand
	initialCommands map[VkCommandBuffer][]api.Cmd
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

// Root returns the path to the root of the state to display. It can vary based
// on filtering mode. Returning nil, nil indicates there is no state to show at
// this point in the capture.
func (*State) Root(ctx context.Context, p *path.State, r *path.ResolveConfig) (path.Node, error) {
	return p, nil
}

// SetupInitialState recreates the command lamdas from the state block.
// These are not encoded so we have to set them up here.
func (s *State) SetupInitialState(ctx context.Context, state *api.GlobalState) {
	s.customState.init(s)

	// Reserve memory for mapped ranges
	for _, dm := range s.DeviceMemories().All() {
		if uint64(dm.MappedLocation()) != uint64(0) {
			state.ReserveMemory([]interval.U64Range{
				interval.U64Range{First: uint64(dm.MappedLocation()), Count: uint64(dm.MappedSize())}})
		}
	}
}

// TrimInitialState scans the capture commands to see which parts of the initial
// state are actually used, and removes some unused parts from it.
//
// Note: the current approach consists in "manually" monitoring which Vulkan
// objects are being used in callbacks passed to sync.MutateWithSubcommands,
// however this basically re-encode some state tracking logic found in the API
// files. A better way would be to use an api.StateWatcher and rely on api.RefID
// to track which objects are accessed: this would avoid to re-encode state
// tracking logic here. There might be some pitfalls though, e.g. when a command
// just reads the handle of an object, the state watcher would not mark an
// access to that object. For instance, when creating a derivate pipeline, a
// VkPipeline handle is used in BasePipelineHandle, the API implementation reads
// this handle, but does not access the corresponding object. So using
// api.StateWatcher might need some wider design considerations.
func (s *State) TrimInitialState(ctx context.Context, capturePath *path.Capture) error {
	// Parts of the state we want to record the usage of.
	descriptorSets := map[VkDescriptorSet]struct{}{}
	pipelines := map[VkPipeline]struct{}{}

	// Record usage in initial state.
	for _, ci := range s.LastComputeInfos().All() {
		pipelines[ci.ComputePipeline().VulkanHandle()] = struct{}{}
	}
	for _, di := range s.LastDrawInfos().All() {
		for _, d := range di.DescriptorSets().All() {
			descriptorSets[d.VulkanHandle()] = struct{}{}
		}
		pipelines[di.GraphicsPipeline().VulkanHandle()] = struct{}{}
	}

	// Record usage in the trace commands
	// top-level commands
	postCmdCb := func(s *api.GlobalState, subCmdIdx api.SubCmdIdx, cmd api.Cmd) {
		switch cmd := cmd.(type) {
		case *VkFreeDescriptorSets:
			ds, err := cmd.PDescriptorSets().Slice(0, (uint64)(cmd.DescriptorSetCount()), s.MemoryLayout).Read(ctx, cmd, s, nil)
			if err != nil {
				panic(err)
			}
			for _, d := range ds {
				descriptorSets[d] = struct{}{}
			}

		case *VkUpdateDescriptorSets:
			// VkWriteDescriptorSet
			writeinfos, err := cmd.PDescriptorWrites().Slice(0, (uint64)(cmd.DescriptorWriteCount()), s.MemoryLayout).Read(ctx, cmd, s, nil)
			if err != nil {
				panic(err)
			}
			for _, wi := range writeinfos {
				descriptorSets[wi.DstSet()] = struct{}{}
			}
			// VkCopyDescriptorSet
			copyinfos, err := cmd.PDescriptorCopies().Slice(0, (uint64)(cmd.DescriptorCopyCount()), s.MemoryLayout).Read(ctx, cmd, s, nil)
			if err != nil {
				panic(err)
			}
			for _, ci := range copyinfos {
				descriptorSets[ci.SrcSet()] = struct{}{}
				descriptorSets[ci.DstSet()] = struct{}{}
			}

		case *VkDestroyPipeline:
			pipelines[cmd.Pipeline()] = struct{}{}
		}

	}
	// sub-commands
	postSubCmdCb := func(state *api.GlobalState, subCmdIdx api.SubCmdIdx, cmd api.Cmd, i interface{}) {
		vkState := GetState(state)
		cmdRef, ok := i.(CommandReferenceʳ)
		if !ok {
			panic("In Vulkan, MutateWithSubcommands' postSubCmdCb 'interface{}' is not a CommandReferenceʳ")
		}
		cmdArgs := GetCommandArgs(ctx, cmdRef, vkState)

		switch args := cmdArgs.(type) {
		case VkCmdBindDescriptorSetsArgsʳ:
			for _, d := range args.DescriptorSets().All() {
				descriptorSets[d] = struct{}{}
			}

		case VkCmdBindPipelineArgsʳ:
			pipelines[args.Pipeline()] = struct{}{}
		}
	}
	if err := sync.MutateWithSubcommands(ctx, capturePath, nil, postCmdCb, nil, postSubCmdCb); err != nil {
		return err
	}

	// Transitive dependencies

	// Each pipeline may be derived from a base pipeline, in which case this
	// base pipeline must be added to the list of used pipelines. Loop on this
	// until we have a stable number of pipelines.
	for numPipelines := 0; numPipelines != len(pipelines); {
		numPipelines = len(pipelines)
		for p := range pipelines {
			// For both graphics and compute derivative pipelines which are
			// created using BasePipelineIndex, our API implementation makes
			// sure that the relevant pipeline handle is set in BasePipeline.
			// Thus, we can safely use the value in BasePipeline. See the
			// post-fence code in vkCreate*Pipelines in
			// gapis/api/vulkan/api/pipeline.api
			g := s.GraphicsPipelines().Get(p)
			if !g.IsNil() && (VkPipelineCreateFlagBits(g.Flags())&VkPipelineCreateFlagBits_VK_PIPELINE_CREATE_DERIVATIVE_BIT) != 0 {
				pipelines[g.BasePipeline()] = struct{}{}
			}
			c := s.ComputePipelines().Get(p)
			if !c.IsNil() && (VkPipelineCreateFlagBits(c.Flags())&VkPipelineCreateFlagBits_VK_PIPELINE_CREATE_DERIVATIVE_BIT) != 0 {
				pipelines[c.BasePipeline()] = struct{}{}
			}
		}
	}

	// Remove unused parts.
	var startSize int

	startSize = s.DescriptorSets().Len()
	for h := range s.DescriptorSets().All() {
		if _, ok := descriptorSets[h]; !ok {
			s.DescriptorSets().Remove(h)
		}
	}
	log.I(ctx, "Trim initial state: DescriptorSets: %v/%v kept", s.DescriptorSets().Len(), startSize)

	startSize = s.GraphicsPipelines().Len()
	for h := range s.GraphicsPipelines().All() {
		if _, ok := pipelines[h]; !ok {
			s.GraphicsPipelines().Remove(h)
		}
	}
	log.I(ctx, "Trim initial state: GraphicsPipelines: %v/%v kept", s.GraphicsPipelines().Len(), startSize)

	startSize = s.ComputePipelines().Len()
	for h := range s.ComputePipelines().All() {
		if _, ok := pipelines[h]; !ok {
			s.ComputePipelines().Remove(h)
		}
	}
	log.I(ctx, "Trim initial state: ComputePipelines: %v/%v kept", s.ComputePipelines().Len(), startSize)

	return nil
}

func (API) GetFramebufferAttachmentInfos(
	ctx context.Context,
	state *api.GlobalState) (info []api.FramebufferAttachmentInfo, err error) {

	count, err := GetState(state).getFramebufferAttachmentCount()
	if err != nil {
		return make([]api.FramebufferAttachmentInfo, 0), err
	}

	infos := make([]api.FramebufferAttachmentInfo, count)

	for attachment := uint32(0); attachment < count; attachment++ {
		w, h, form, i, r, t, err := GetState(state).getFramebufferAttachmentInfo(uint32(attachment))

		info := api.FramebufferAttachmentInfo{
			Width:     w,
			Height:    h,
			Index:     i,
			CanResize: r,
			Type:      t,
		}

		if err != nil {
			info.Err = err
		} else {
			switch t {
			case api.FramebufferAttachmentType_OutputDepth,
				api.FramebufferAttachmentType_InputDepth:
				format, err := getDepthImageFormatFromVulkanFormat(form)
				if err != nil {
					info.Err = fmt.Errorf("Unknown format for Depth attachment: %v", form)
				} else {
					info.Format = format
				}

			case api.FramebufferAttachmentType_OutputColor,
				api.FramebufferAttachmentType_InputColor:
				format, err := getImageFormatFromVulkanFormat(form)
				if err != nil {
					info.Err = fmt.Errorf("Unknown format for Color attachment: %v", form)
				} else {
					info.Format = format
				}

			default:
				info.Err = fmt.Errorf("Unsupported Attachment Type")
			}
		}

		infos[attachment] = info
	}
	return infos, nil
}

// Interface check.
var _ api.MeshProvider = &API{}

// Mesh implements the api.MeshProvider interface
func (API) Mesh(ctx context.Context, o interface{}, p *path.Mesh, r *path.ResolveConfig) (*api.Mesh, error) {
	switch dc := o.(type) {
	case *VkQueueSubmit:
		return drawCallMesh(ctx, dc, p, r)
	}
	return nil, api.ErrMeshNotAvailable
}

// Interface check.
var _ api.PipelineProvider = &API{}

// BoundPipeline implements the api.PipelineProvider interface.
func (API) BoundPipeline(ctx context.Context, o interface{}, p *path.Pipelines, r *path.ResolveConfig) (api.BoundPipeline, error) {
	switch dc := o.(type) {
	case *VkQueueSubmit:
		return drawCallPipeline(ctx, dc, p, r)
	}
	return api.BoundPipeline{}, api.ErrPipelineNotAvailable
}

type markerType int

const (
	debugMarker = iota
	renderPassMarker
	drawGroupMarker
	invalidDrawGroupMarker
)

type markerInfo struct {
	name   string
	ty     markerType
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
	experimentalCmds := []api.SubCmdIdx{}

	commandMap := make(map[api.Cmd]api.CmdID)
	st.AddCommand = func(a interface{}) {
		data := a.(CommandReferenceʳ)
		if initialCommands, ok := st.initialCommands[data.Buffer()]; ok {
			commandMap[initialCommands[data.CommandIndex()]] = i
		}
	}

	getDebugGroupExperimentalCmds := func(executeCmdID api.SubCmdIdx, startID api.SubCmdIdx) []api.SubCmdIdx {
		currentExperimentalCmds := []api.SubCmdIdx{}
		for index := len(experimentalCmds) - 1; index >= 0; index-- {
			if !startID.LessThan(experimentalCmds[index]) {
				break
			}
			currentExperimentalCmds = append(currentExperimentalCmds, experimentalCmds[index])
		}

		return currentExperimentalCmds
	}

	mergeExperimentalCmds := func(executeCmdID api.SubCmdIdx) {
		for len(experimentalCmds) > 0 {
			if !executeCmdID.Contains(experimentalCmds[len(experimentalCmds)-1]) {
				break
			}
			experimentalCmds = experimentalCmds[:len(experimentalCmds)-1]
		}
		experimentalCmds = append(experimentalCmds, executeCmdID)
	}

	pushMarker := func(name string, ty markerType, commandIndex int, parent api.SubCmdIdx) {
		markerStack = append(markerStack,
			&markerInfo{
				name:   name,
				ty:     ty,
				start:  uint64(commandIndex),
				parent: parent,
			})

		if ty == renderPassMarker {
			experimentalCmds = []api.SubCmdIdx{}
		}
	}

	popMarkerWithNewGroupName := func(ty markerType, id uint64, name string) {
		if len(markerStack) == 0 {
			log.W(ctx, "Group Marker stack is empty while trying to pop [%v:%v]:%v", id, name, ty)
			return
		}

		marker := markerStack[len(markerStack)-1]
		if marker.ty != ty {
			if marker.ty != drawGroupMarker && ty == invalidDrawGroupMarker {
				log.D(ctx, "No draw command found after state setting [%v:%v]", id, name)
			} else {
				log.W(ctx, "Mismatch of command groups : [%v:%v]%V != %v", id, name, ty, marker.ty)
			}
		}

		currentExperimentalCmds := []api.SubCmdIdx{}
		if ty == drawGroupMarker {
			currentExperimentalCmds = []api.SubCmdIdx{append(marker.parent, id)}
			experimentalCmds = append(experimentalCmds, currentExperimentalCmds...)
		} else if ty == renderPassMarker {
			currentExperimentalCmds = experimentalCmds
		} else if ty == debugMarker {
			currentExperimentalCmds = getDebugGroupExperimentalCmds(append(marker.parent, id), append(marker.parent, marker.start))
		}

		d.SubCommandMarkerGroups.NewMarkerGroup(marker.parent, name, marker.start, id+1, currentExperimentalCmds)
		markerStack = markerStack[0 : len(markerStack)-1]
	}

	popMarker := func(ty markerType, id uint64) {
		if len(markerStack) > 0 {
			popMarkerWithNewGroupName(ty, id, markerStack[len(markerStack)-1].name)
		}
	}

	var walkCommandBuffer func(cb CommandBufferObjectʳ, idx api.SubCmdIdx, id api.CmdID, order int) ([]sync.SubcommandReference, []api.SubCmdIdx)
	walkCommandBuffer = func(cb CommandBufferObjectʳ, idx api.SubCmdIdx, id api.CmdID, order int) ([]sync.SubcommandReference, []api.SubCmdIdx) {
		refs := make([]sync.SubcommandReference, 0)
		subgroups := make([]api.SubCmdIdx, 0)
		nextSubpass := 0
		canStartDrawGrouping := true

		d.RenderPassLookup.AddCommandBuffer(ctx, order, cb.VulkanHandle().Handle(), idx)
		var renderPassKey sync.RenderPassKey
		var renderPassStart api.SubCmdIdx

		for i := 0; i < cb.CommandReferences().Len(); i++ {
			initialCommands, ok := st.initialCommands[cb.VulkanHandle()]
			var ref sync.SubcommandReference
			if !ok {
				continue
			}

			// Update values in sync data.
			nv := append(api.SubCmdIdx{}, idx...)
			nv = append(nv, uint64(i))
			generatingId := initialCommands[i]
			if generatingId == nil {
				ref = sync.SubcommandReference{
					append(api.SubCmdIdx{}, nv[1:]...),
					api.CmdNoID,
					cb.CommandReferences().Get(uint32(i)),
				}
			} else {
				ref = sync.SubcommandReference{
					append(api.SubCmdIdx{}, nv[1:]...),
					commandMap[generatingId],
					nil,
				}
			}
			d.SubcommandLookup.SetValue(nv, ref)
			refs = append(refs, ref)

			// Handle draw commands grouping.
			cmdName := cb.CommandReferences().Get(uint32(i)).Type().String()
			isDrawGroupEndCmd := strings.HasPrefix(cmdName, "cmd_vkCmdDraw") || strings.HasPrefix(cmdName, "cmd_vkCmdDispatch") ||
				strings.HasPrefix(cmdName, "cmd_vkCmdClearAttachments") || strings.HasPrefix(cmdName, "cmd_vkCmdBeginQuery") ||
				strings.HasPrefix(cmdName, "cmd_vkCmdEndQuery")
			isStateSettingCmd := (strings.HasPrefix(cmdName, "cmd_vkCmdSet") || strings.HasPrefix(cmdName, "cmd_vkCmdPush") ||
				strings.HasPrefix(cmdName, "cmd_vkCmdBind")) && !strings.HasPrefix(cmdName, "cmd_vkCmdSetEvent")
			if isStateSettingCmd && canStartDrawGrouping {
				pushMarker("State Setting Group",
					drawGroupMarker, i, append(api.SubCmdIdx{}, idx...))
				canStartDrawGrouping = false
			} else if isDrawGroupEndCmd && canStartDrawGrouping {
				// When a draw group starts with a draw command it will only contain that single command
				groupName := strings.TrimPrefix(cmdName, "cmd_vkCmd")
				pushMarker(groupName, drawGroupMarker, i, append(api.SubCmdIdx{}, idx...))
				popMarker(drawGroupMarker, uint64(i))
			} else if isDrawGroupEndCmd && !canStartDrawGrouping {
				// When a group is complete with state setting cmds followed by a draw command, override the group name.
				groupName := strings.TrimPrefix(cmdName, "cmd_vkCmd")
				popMarkerWithNewGroupName(drawGroupMarker, uint64(i), groupName)
				canStartDrawGrouping = true
			} else if !isStateSettingCmd && !isDrawGroupEndCmd && !canStartDrawGrouping {
				// Handle an edge case where a group of state setting commands are
				// followed by something other than a drawing command.
				popMarker(invalidDrawGroupMarker, uint64(i-1))
				canStartDrawGrouping = true
			}

			// Handle extra command buffer reference, render pass grouping and debug marker grouping.
			switch args := GetCommandArgs(ctx, cb.CommandReferences().Get(uint32(i)), st).(type) {
			case VkCmdExecuteCommandsArgsʳ:
				d.SubcommandNames.SetValue(nv, "") // Clear the group name so that the original commnd is shown.
				for j := uint64(0); j < uint64(args.CommandBuffers().Len()); j++ {
					cbo := st.CommandBuffers().Get(args.CommandBuffers().Get(uint32(j)))
					subIdx := append(api.SubCmdIdx{}, idx...)
					subIdx = append(subIdx, uint64(i), j)
					newRefs, newSubgroups := walkCommandBuffer(cbo, subIdx, id, order)
					refs = append(refs, newRefs...)
					subgroups = append(subgroups, newSubgroups...)
					if cbo.CommandReferences().Len() > 0 {
						subgroups = append(subgroups, append(idx, uint64(i), uint64(j), uint64(cbo.CommandReferences().Len())))
						name := fmt.Sprintf("Command Buffer: %v", cbo.VulkanHandle())
						if label := cbo.Label(ctx, s); len(label) > 0 {
							name = label
						}
						d.SubcommandNames.SetValue(append(idx, uint64(i), j), name)
					}
					mergeExperimentalCmds(append(idx, uint64(i)))
				}
			case VkCmdBeginRenderPassArgsʳ:
				rp := st.RenderPasses().Get(args.RenderPass())
				name := fmt.Sprintf("RenderPass: %v", rp.VulkanHandle())
				if label := rp.Label(ctx, s); len(label) > 0 {
					name = label
				}
				pushMarker(name, renderPassMarker, i, append(api.SubCmdIdx{}, idx...))

				nextSubpass = 0
				if rp.SubpassDescriptions().Len() > 1 {
					name = fmt.Sprintf("Subpass: %v", nextSubpass)
					pushMarker(name, renderPassMarker, i, append(api.SubCmdIdx{}, idx...))
					nextSubpass++
				}
				break
			case VkCmdEndRenderPassArgsʳ:
				if nextSubpass > 0 { // Pop one more time since there were one extra marker pushed.
					popMarker(renderPassMarker, uint64(i))
				}
				popMarker(renderPassMarker, uint64(i))
				break
			case VkCmdNextSubpassArgsʳ:
				popMarker(renderPassMarker, uint64(i-1))
				name := fmt.Sprintf("Subpass: %v", nextSubpass)
				pushMarker(name, renderPassMarker, i, append(api.SubCmdIdx{}, idx...))
				nextSubpass++
			case VkCmdDebugMarkerBeginEXTArgsʳ:
				pushMarker(args.MarkerName(), debugMarker, i, append(api.SubCmdIdx{}, idx...))
			case VkCmdBeginDebugUtilsLabelEXTArgsʳ:
				pushMarker(args.LabelName(), debugMarker, i, append(api.SubCmdIdx{}, idx...))
			case VkCmdEndDebugUtilsLabelEXTArgsʳ:
				popMarker(debugMarker, uint64(i))
			case VkCmdDebugMarkerEndEXTArgsʳ:
				popMarker(debugMarker, uint64(i))
			}

			if id.IsReal() {
				// Markdown RenderPasses' and SubPasses' command index, for helping
				// connect a command and its correlated GPU slices.
				switch args := GetCommandArgs(ctx, cb.CommandReferences().Get(uint32(i)), st).(type) {
				case VkCmdBeginRenderPassArgsʳ:
					renderPassKey = sync.RenderPassKey{
						Submission:    order,
						CommandBuffer: cb.VulkanHandle().Handle(),
						RenderPass:    args.RenderPass().Handle(),
						Framebuffer:   args.Framebuffer().Handle(),
					}
					renderPassStart = append(api.SubCmdIdx{}, nv...)
				case VkCmdEndRenderPassArgsʳ:
					d.RenderPassLookup.AddRenderPass(ctx, renderPassKey, sync.SubCmdRange{renderPassStart, nv})
				}
			}
		}

		for i := len(markerStack) - 1; i >= 0; i-- {
			if len(markerStack[i].parent) < len(idx) {
				break
			}
			marker := markerStack[len(markerStack)-1]
			d.SubCommandMarkerGroups.NewMarkerGroup(marker.parent, marker.name, marker.start,
				uint64(cb.CommandReferences().Len()), []api.SubCmdIdx{})
			markerStack = markerStack[0 : len(markerStack)-1]
		}
		return refs, subgroups
	}

	order := 0
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
			submits, err := cmd.PSubmits().Slice(uint64(0), submitCount, l).Read(ctx, cmd, s, nil)
			if err != nil {
				return err
			}
			for submitIdx, submit := range submits {
				bufferCount := submit.CommandBufferCount()
				buffers, err := submit.PCommandBuffers().Slice(uint64(0), uint64(bufferCount), l).Read(ctx, cmd, s, nil)
				if err != nil {
					return err
				}
				d.SubcommandNames.SetValue(api.SubCmdIdx{uint64(id), uint64(submitIdx)}, fmt.Sprintf("pSubmits[%v]: ", submitIdx))
				for j, buff := range buffers {
					cmdBuff := st.CommandBuffers().Get(buff)
					name := fmt.Sprintf("Command Buffer: %v", buff)
					if label := cmdBuff.Label(ctx, s); len(label) > 0 {
						name = label
					}
					d.SubcommandNames.SetValue(api.SubCmdIdx{uint64(id), uint64(submitIdx), uint64(j)}, name)
					if cmdBuff.CommandReferences().Len() >= 0 {
						additionalRefs, additionalSubgroups := walkCommandBuffer(cmdBuff, api.SubCmdIdx{uint64(i), uint64(submitIdx), uint64(j)}, i, order)
						for _, sg := range additionalSubgroups {
							d.SubcommandGroups[i] = append(d.SubcommandGroups[i], sg[1:])
						}
						d.SubcommandGroups[i] = append(d.SubcommandGroups[i], api.SubCmdIdx{uint64(submitIdx), uint64(j), uint64(cmdBuff.CommandReferences().Len())})
						refs = append(refs, additionalRefs...)
					}
				}
			}
			order++
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

	cb := CommandBuilder{Thread: 0}
	_, a, err := AddCommand(ctx, cb, cr.Buffer(), st, st, GetCommandArgs(ctx, cr, s))
	if err != nil {
		return nil, log.Errf(ctx, err, "Invalid Command")
	}
	return a, nil
}

// Interface check
var _ sync.SynchronizedAPI = &API{}

func (API) GetTerminator(ctx context.Context, c *path.Capture) (terminator.Terminator, error) {
	return newVulkanTerminator(ctx, c, 0)
}

func (API) MutateSubcommands(ctx context.Context, id api.CmdID, cmd api.Cmd,
	s *api.GlobalState, preSubCmdCb func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{}),
	postSubCmdCb func(s *api.GlobalState, idx api.SubCmdIdx, cmd api.Cmd, subCmdRef interface{})) error {
	c := GetState(s)
	if postSubCmdCb != nil {
		c.PostSubcommand = func(subCmdRef interface{}) {
			postSubCmdCb(s, append(api.SubCmdIdx{uint64(id)}, c.SubCmdIdx...), cmd, subCmdRef)
		}
	}
	if preSubCmdCb != nil {
		c.PreSubcommand = func(subCmdRef interface{}) {
			preSubCmdCb(s, append(api.SubCmdIdx{uint64(id)}, c.SubCmdIdx...), cmd, subCmdRef)
		}
	}
	if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil {
		return fmt.Errorf("Fail to mutate command %v: %v", cmd, err)
	}
	return nil
}
