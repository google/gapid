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
	"sort"

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
	postBindSparse    func(binds QueuedSparseBindsʳ)
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
	submissionMap := make(map[api.Cmd]api.CmdID)
	commandMap := make(map[api.Cmd]api.CmdID)
	lastCmdIndex := api.CmdID(0)
	lastSubCmdsInSubmittedCmdBufs := &api.SubCmdIdxTrie{}

	// Prepare for collect marker groups
	// Stacks of open markers for each VkQueue
	markerStack := map[VkQueue][]*markerInfo{}
	// Stacks of markers to be opened in the next subcommand for each VkQueue
	markersToOpen := map[VkQueue][]*markerInfo{}
	s.pushMarkerGroup = func(name string, next bool, ty MarkerType) {
		vkQu := (s.CurrentSubmission).(*VkQueueSubmit).Queue()
		if next {
			// Add to the to-open marker stack, marker will be opened in the next
			// subcommand
			stack := markersToOpen[vkQu]
			markersToOpen[vkQu] = append(stack, &markerInfo{name: name, ty: ty})
		} else {
			// Add to the marker stack
			stack := markerStack[vkQu]
			fullCmdIdx := api.SubCmdIdx{uint64(submissionMap[s.CurrentSubmission])}
			fullCmdIdx = append(fullCmdIdx, s.SubCmdIdx...)
			marker := &markerInfo{name: name,
				ty:     ty,
				start:  fullCmdIdx[len(fullCmdIdx)-1],
				end:    uint64(0),
				parent: fullCmdIdx[0 : len(fullCmdIdx)-1]}
			markerStack[vkQu] = append(stack, marker)
		}
	}
	s.popMarkerGroup = func(ty MarkerType) {
		vkQu := (s.CurrentSubmission).(*VkQueueSubmit).Queue()
		stack := markerStack[vkQu]
		if len(stack) == 0 {
			log.D(ctx, "Cannot pop marker with type: %v, no open marker with same type at: VkQueueSubmit ID: %v, SubCmdIdx: %v",
				ty, submissionMap[s.CurrentSubmission], s.SubCmdIdx)
			return
		}
		// If the type of the top marker in the stack does not match with the
		// request type, pop until a matching marker is found and pop it. The
		// spilled markers are processed in the following way: if it is a debug
		// marker, resurrect it in the next subcommand, if it is a renderpass
		// marker, discard it.
		top := len(stack) - 1
		for top >= 0 && stack[top].ty != ty {
			log.D(ctx, "Type of the top marker does not match with the pop request")
			end := s.SubCmdIdx[len(s.SubCmdIdx)-1] + 1
			d.SubCommandMarkerGroups.NewMarkerGroup(stack[top].parent, stack[top].name, stack[top].start, end)
			switch stack[top].ty {
			case DebugMarker:
				markersToOpen[vkQu] = append(markersToOpen[vkQu], stack[top])
				log.D(ctx, "Debug marker popped due to popping renderpass marker, new debug marker group will be opened again in the next subcommand")
			default:
				log.D(ctx, "Renderpass marker popped due to popping debug marker, renderpass marker group will be closed here")
			}
			top--
		}
		// Update the End value of the debug marker and create new group.
		if top >= 0 {
			end := s.SubCmdIdx[len(s.SubCmdIdx)-1] + 1
			d.SubCommandMarkerGroups.NewMarkerGroup(stack[top].parent, stack[top].name, stack[top].start, end)
			markerStack[vkQu] = stack[0:top]
		} else {
			markerStack[vkQu] = []*markerInfo{}
		}
	}

	s.PreSubcommand = func(interface{}) {
		// Update the submission map before execute subcommand callback and
		// postSubCommand callback.
		if _, ok := submissionMap[s.CurrentSubmission]; !ok {
			submissionMap[s.CurrentSubmission] = i
		}
		// Examine the marker stack. If the comming subcommand is submitted in a
		// different command buffer or submission batch or VkQueueSubmit call, and
		// there are unclosed marker group, we need to 1) check whether the
		// unclosed marker groups are opened in secondary command buffers, log
		// error and pop them.  2) Close all the unclosed "debug marker" group, and
		// begin new groups for the new command buffer. Note that only "debug
		// marker" groups are resurrected in this step, all unclosed "renderpass
		// markers" are assumed closed.
		// Finally, no matter whether the comming subcommand is in a different
		// command buffer or submission batch, If there are pending markers in the
		// to-open stack, begin new groups for those pending markers.
		vkQu := (s.CurrentSubmission).(*VkQueueSubmit).Queue()
		stack := markerStack[vkQu]
		fullCmdIdx := api.SubCmdIdx{uint64(submissionMap[s.CurrentSubmission])}
		fullCmdIdx = append(fullCmdIdx, s.SubCmdIdx...)

		for lastCmdIndex != api.CmdID(0) && len(stack) > 0 {
			top := stack[len(stack)-1]
			if len(top.parent) > len(fullCmdIdx) {
				// The top of the stack is an unclosed debug marker group which is
				// opened in a secondary command buffer. This debug marker group will
				// be closed here, the End value of the group will be the last updated
				// value (which should be one plus the last command index in its
				// secondary command buffer).
				log.E(ctx, "DebugMarker began in secondary command buffer does not close. Close now")
				d.SubCommandMarkerGroups.NewMarkerGroup(top.parent, top.name, top.start, top.end)
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
			!stack[len(stack)-1].parent.Contains(fullCmdIdx) {
			originalStack := []*markerInfo(stack)
			markerStack[vkQu] = []*markerInfo{}
			for _, o := range originalStack {
				s.pushMarkerGroup(o.name, false, DebugMarker)
			}
		}
		// Open new groups for the pending markers in the to-open stack
		toOpenStack := markersToOpen[vkQu]
		i := len(toOpenStack) - 1
		for i >= 0 {
			s.pushMarkerGroup(toOpenStack[i].name, false, toOpenStack[i].ty)
			i--
		}
		markersToOpen[vkQu] = []*markerInfo{}
	}

	s.PostSubcommand = func(a interface{}) {
		data := a.(CommandReferenceʳ)
		rootIdx := api.CmdID(i)
		if k, ok := submissionMap[s.CurrentSubmission]; ok {
			rootIdx = api.CmdID(k)
		} else {
			submissionMap[s.CurrentSubmission] = i
		}

		// No way for this to not exist, we put it in up there
		k := submissionMap[s.CurrentSubmission]
		id := api.CmdNoID

		if initialCommands, ok := s.initialCommands[data.Buffer()]; ok {
			if initialCommands[data.CommandIndex()] != nil {
				id = commandMap[initialCommands[data.CommandIndex()]]
			}
		}
		subCmdIdx := append(api.SubCmdIdx(nil), s.SubCmdIdx...)
		subCmdRef := sync.SubcommandReference{subCmdIdx, id, data, false}
		if v, ok := d.SubcommandReferences[k]; ok {
			v = append(v, subCmdRef)
			d.SubcommandReferences[k] = v
		} else {
			d.SubcommandReferences[k] = []sync.SubcommandReference{subCmdRef}
		}
		fullSubCmdIdx := api.SubCmdIdx(append([]uint64{uint64(k)}, s.SubCmdIdx...))
		d.SubcommandLookup.SetValue(fullSubCmdIdx, subCmdRef)
		lastSubCmdsInSubmittedCmdBufs.SetValue(fullSubCmdIdx[0:len(fullSubCmdIdx)-1], fullSubCmdIdx[len(fullSubCmdIdx)-1])
		lastCmdIndex = k

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
		vkQu := (s.CurrentSubmission).(*VkQueueSubmit).Queue()
		for _, ms := range markerStack[vkQu] {
			// If the last subcommand is in a secondary command buffer and current
			// recording debug marker groups are opened in a primary command buffer,
			// this prevents assigning a wrong End value to the open marker groups.
			if len(s.SubCmdIdx) == len(ms.parent) {
				ms.end = s.SubCmdIdx[len(s.SubCmdIdx)-1] + 1
			}
		}
	}

	s.AddCommand = func(a interface{}) {
		data := a.(CommandReferenceʳ)
		if initialCommands, ok := s.initialCommands[data.Buffer()]; ok {
			commandMap[initialCommands[data.CommandIndex()]] = i
		}
	}

	err = api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		i = id
		if err := cmd.Mutate(ctx, id, st, nil, nil); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	submittedCmdBufs := lastSubCmdsInSubmittedCmdBufs.PostOrderSortedKeys()
	for _, submittedCmdBufIdx := range submittedCmdBufs {
		if len(submittedCmdBufIdx) == 0 {
			panic(fmt.Sprintf("%+v", submittedCmdBufs))
		}
		submissionId := api.CmdID(submittedCmdBufIdx[0])
		lastSubCmdIdInCmdBuf := lastSubCmdsInSubmittedCmdBufs.Value(submittedCmdBufIdx).(uint64)
		if v, ok := d.SubcommandGroups[submissionId]; ok {
			v = append(v, append(submittedCmdBufIdx[1:], lastSubCmdIdInCmdBuf))
			d.SubcommandGroups[submissionId] = v
		} else {
			d.SubcommandGroups[submissionId] = []api.SubCmdIdx{
				append(submittedCmdBufIdx[1:], lastSubCmdIdInCmdBuf),
			}
		}
	}

	return dependencySync(ctx, d, c)
}

func dependencySync(ctx context.Context, d *sync.Data, c *path.Capture) error {
	st, err := capture.NewState(ctx)
	if err != nil {
		return err
	}
	cmds, err := resolve.Cmds(ctx, c)
	if err != nil {
		return err
	}

	l := st.MemoryLayout

	addNode := func(pt sync.SyncNode) sync.SyncNodeIdx {
		d.SyncNodes = append(d.SyncNodes, pt)
		return sync.SyncNodeIdx(len(d.SyncNodes) - 1)
	}
	addDep := func(depender, dependee sync.SyncNodeIdx) {
		v, _ := d.SyncDependencies[depender]
		// append to nil is ok
		d.SyncDependencies[depender] = append(v, dependee)
	}
	lastHostBarrier := addNode(sync.AbstractNode{})

	getCmdNode := func(id api.CmdID) sync.SyncNodeIdx {
		if pt, ok := d.CmdSyncNodes[id]; ok {
			return pt
		}
		d.CmdSyncNodes[id] = addNode(sync.CmdNode{[]uint64{uint64(id)}})
		return d.CmdSyncNodes[id]
	}

	semSignaler := map[VkSemaphore]sync.SyncNodeIdx{}
	semDepend := func(idx sync.SyncNodeIdx, sem VkSemaphore) {
		signaler, ok := semSignaler[sem]
		if ok {
			addDep(idx, signaler)
			// Waiting on a semaphore clears the semaphore
			delete(semSignaler, sem)
		}
	}

	fenceSignaler := map[VkFence]sync.SyncNodeIdx{}
	fenceDepend := func(idx sync.SyncNodeIdx, fence VkFence) {
		signaler, ok := fenceSignaler[fence]
		if ok {
			addDep(idx, signaler)
		}
	}

	queueSubmits := map[VkQueue][]sync.SyncNodeIdx{}
	deviceQueues := map[VkDevice]map[VkQueue]struct{}{}
	addSubmit := func(idx sync.SyncNodeIdx, queue VkQueue) {
		qs, _ := queueSubmits[queue]
		queueSubmits[queue] = append(qs, idx)
	}
	clearQueue := func(idx sync.SyncNodeIdx, queue VkQueue) {
		qs, ok := queueSubmits[queue]
		if ok {
			delete(queueSubmits, queue)
			val, _ := d.SyncDependencies[idx]
			d.SyncDependencies[idx] = append(val, qs...)
		}
	}

	api.ForeachCmd(ctx, cmds, func(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
		// only track the commands that do anything
		switch c := cmd.(type) {
		case *VkQueueSubmit:
		case *VkQueuePresentKHR:
		case *VkCreateFence:
		case *VkGetFenceStatus:
		case *VkWaitForFences:
		case *VkResetFences:
		case *VkQueueWaitIdle:
		case *VkDeviceWaitIdle:
		case *VkGetDeviceQueue:
			// Not part of the graph, just need to associate the queues with the device
			c.Extras().Observations().ApplyReads(st.Memory.ApplicationPool())
			queue := c.PQueue().MustRead(ctx, c, st, nil)
			if _, ok := deviceQueues[c.Device()]; !ok {
				deviceQueues[c.Device()] = map[VkQueue]struct{}{}
			}
			deviceQueues[c.Device()][queue] = struct{}{}
			return nil
		default:
			return nil
		}
		cmd.Extras().Observations().ApplyReads(st.Memory.ApplicationPool())
		cmdPt := getCmdNode(id)
		addDep(cmdPt, lastHostBarrier)
		switch c := cmd.(type) {
		case *VkQueueSubmit:
			submitCount := uint64(c.SubmitCount())
			submits := c.PSubmits().Slice(uint64(0), submitCount, l).MustRead(ctx, cmd, st, nil)

			submitSrcs := make([]sync.SyncNodeIdx, len(submits))
			submitDsts := make([]sync.SyncNodeIdx, len(submits))
			// For each submit, create an abstract node at each
			// end.  The start will depend on the semaphore
			// signalers, and the end will be the sources for the
			// next semaphores.
			for i, s := range submits {
				src := addNode(sync.AbstractNode{})
				dst := addNode(sync.AbstractNode{})

				waitSems := s.PWaitSemaphores().Slice(
					uint64(0),
					uint64(s.WaitSemaphoreCount()), l).
					MustRead(ctx, cmd, st, nil)
				for _, s := range waitSems {
					semDepend(src, s)
				}

				signalSems := s.PSignalSemaphores().Slice(
					uint64(0),
					uint64(s.SignalSemaphoreCount()), l).
					MustRead(ctx, cmd, st, nil)
				for _, s := range signalSems {
					semSignaler[s] = dst
				}

				addDep(src, cmdPt)

				submitSrcs[i] = src
				submitDsts[i] = dst
			}

			type commandExecutor struct {
				lastIdx  api.SubCmdIdx
				executor api.CmdID
			}
			// CommandRanges doesn't have a defined iteration
			// order, so we need to sort it.
			executors := make([]commandExecutor, 0,
				len(d.CommandRanges[id].Ranges))
			for executor, lastIdx := range d.CommandRanges[id].Ranges {
				executors = append(executors, commandExecutor{
					lastIdx:  lastIdx,
					executor: executor,
				})
			}
			sort.Slice(executors, func(i, j int) bool {
				return executors[i].lastIdx.LessThan(executors[j].lastIdx)
			})

			excIdx := 0
			for _, subcmd := range d.SubcommandReferences[id] {
				for excIdx < len(executors)-1 && executors[excIdx].lastIdx.LessThan(subcmd.Index) {
					excIdx++
				}

				sp := addNode(sync.CmdNode{
					append(api.SubCmdIdx{uint64(id)}, subcmd.Index...),
				})
				sub := subcmd.Index[0]
				// Make each subcommand depend on the source of
				// the submit it came from, and be a dependency
				// of the dst in the submit it came from.
				addDep(sp, submitSrcs[sub])
				addDep(submitDsts[sub], sp)
				addDep(sp, getCmdNode(executors[excIdx].executor))
			}

			donePt := addNode(sync.AbstractNode{})
			for _, dst := range submitDsts {
				addDep(donePt, dst)
			}

			fence := c.Fence()
			if fence != VkFence(0) {
				fenceSignaler[fence] = donePt
			}
			queue := c.Queue()
			// If we do vkWaitQueueIdle then we depend on all previous submits.
			addSubmit(donePt, queue)
		case *VkQueuePresentKHR:
			info := c.PPresentInfo().Slice(uint64(0), uint64(1), l).MustRead(ctx, cmd, st, nil)[0]
			waitSems := info.PWaitSemaphores().Slice(
				uint64(0),
				uint64(info.WaitSemaphoreCount()),
				l).MustRead(ctx, cmd, st, nil)
			for _, s := range waitSems {
				semDepend(cmdPt, s)
			}
		case *VkCreateFence:
			signaledBit := VkFenceCreateFlags(VkFenceCreateFlagBits_VK_FENCE_CREATE_SIGNALED_BIT)
			signaled := ((c.PCreateInfo().
				Slice(uint64(0), uint64(1), l).
				MustRead(ctx, cmd, st, nil)[0].
				Flags()) & signaledBit) == signaledBit
			if signaled {
				fence := c.PFence().Slice(uint64(0), uint64(1), l).MustRead(ctx, cmd, st, nil)[0]
				fenceSignaler[fence] = cmdPt
			}
		case *VkGetFenceStatus:
			if c.Result() == VkResult_VK_SUCCESS {
				fenceDepend(cmdPt, c.Fence())
				lastHostBarrier = cmdPt
			}
		case *VkWaitForFences:
			fenceCount := uint64(c.FenceCount())
			if fenceCount == 1 || c.WaitAll() != VkBool32(0) {
				// We can be sure all the fences were signaled
				fences := c.PFences().Slice(uint64(0), fenceCount, l).MustRead(ctx, cmd, st, nil)
				for _, f := range fences {
					fenceDepend(cmdPt, f)
				}
				lastHostBarrier = cmdPt
			}
		case *VkResetFences:
			fences := c.PFences().Slice(uint64(0), uint64(c.FenceCount()), l).MustRead(ctx, cmd, st, nil)
			for _, f := range fences {
				delete(fenceSignaler, f)
			}
		case *VkQueueWaitIdle:
			clearQueue(cmdPt, c.Queue())
			lastHostBarrier = cmdPt
		case *VkDeviceWaitIdle:
			queues, ok := deviceQueues[c.Device()]
			if ok {
				for q := range queues {
					clearQueue(cmdPt, q)
				}
			}
			lastHostBarrier = cmdPt
		}
		return nil
	})
	return nil
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
	s, err := resolve.SyncData(ctx, p)
	if err != nil {
		return false, err
	}

	if len(after) == 1 {
		a := api.CmdID(after[0])
		// If we are not running subcommands we can probably batch
		for _, v := range s.SortedKeys() {
			if v > a {
				return true, nil
			}
			for _, k := range s.CommandRanges[v].SortedKeys() {
				if k > a {
					return false, nil
				}
			}
		}
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
	if err := cmd.Mutate(ctx, id, s, nil, nil); err != nil && err == context.Canceled {
		return err
	}
	return nil
}
