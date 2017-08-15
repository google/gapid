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
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve"
	"github.com/google/gapid/gapis/service/path"
)

// VulkanTerminator is very similar to EarlyTerminator.
// It has 2 additional properties.
//   1) If a VkQueueSubmit is found, and it contains an event that will be
//      signaled after the final request, we remove the event from the
//      command-list, and remove any subsequent events
//   2) If a request is made to replay until the MIDDLE of a vkQueueSubmit,
//      then it will patch that command-list to remove any commands after
//      the command in question.
//      Furthermore it will continue the replay until that command can be run
//      i.e. it will make sure to continue to mutate the trace until
//      all pending events have been successfully completed.
//      TODO(awoloszyn): Handle #2
// This takes advantage of the fact that all atoms will be in order.
type VulkanTerminator struct {
	lastRequest     api.CmdID
	requestSubIndex []uint64
	stopped         bool
	syncData        *sync.Data
}

var _ transform.Terminator = &VulkanTerminator{}

func NewVulkanTerminator(ctx context.Context, capture *path.Capture) (*VulkanTerminator, error) {
	syncData, err := database.Build(ctx, &resolve.SynchronizationResolvable{capture})
	if err != nil {
		return nil, err
	}
	s, ok := syncData.(*sync.Data)
	if !ok {
		return nil, log.Errf(ctx, nil, "Could not get synchronization data")
	}

	return &VulkanTerminator{api.CmdID(0), make([]uint64, 0), false, s}, nil
}

// Add adds the atom with identifier id to the set of atoms that must be seen
// before the VulkanTerminator will consume all atoms (excluding the EOS atom).
func (t *VulkanTerminator) Add(ctx context.Context, id api.CmdID, subcommand []uint64) error {
	if len(t.requestSubIndex) != 0 {
		return log.Errf(ctx, nil, "Cannot handle multiple requests when requesting a subcommand")
	}

	if id > t.lastRequest {
		t.lastRequest = id
	}

	// If we are not trying to index a subcommand, then just continue on our way.
	if len(subcommand) == 0 {
		return nil
	}

	t.requestSubIndex = append([]uint64{uint64(id)}, subcommand...)
	sc := api.SubCmdIdx(t.requestSubIndex[1:])
	handled := false
	if rng, ok := t.syncData.CommandRanges[id]; ok {
		for _, k := range rng.SortedKeys() {
			if !rng.Ranges[k].LessThan(sc) {
				t.lastRequest = api.CmdID(k)
				handled = true
				break
			}
		}
	} else {
		return log.Errf(ctx, nil, "The given atom does not have a subcommands")
	}

	// If we cannot find the subindex, backtrack to the main command
	if !handled {
		t.lastRequest = id
		t.requestSubIndex = []uint64{uint64(id)}
	}

	return nil
}

func walkCommands(s *State,
	commands CommandBufferCommands,
	callback func(*CommandBufferCommand)) {
	for _, c := range commands {
		callback(&c)
		if execSub, ok := c.recreateData.(*RecreateCmdExecuteCommandsData); ok {
			for _, k := range execSub.CommandBuffers.KeysSorted() {
				cbc := s.CommandBuffers[execSub.CommandBuffers[k]]
				walkCommands(s, cbc.Commands, callback)
			}
		}
	}
}

func getExtra(idx api.SubCmdIdx, loopLevel int) int {
	if len(idx) == loopLevel+1 {
		return 1
	}
	return 0
}

func incrementLoopLevel(idx api.SubCmdIdx, loopLevel *int) bool {
	if len(idx) == *loopLevel+1 {
		return false
	}
	*loopLevel += 1
	return true
}

// resolveCurrentRenderPass walks all of the current and pending commands
// to determine what renderpass we are in after the idx'th subcommand
func resolveCurrentRenderPass(ctx context.Context, s *api.State, submit *VkQueueSubmit,
	idx api.SubCmdIdx, lrp *RenderPassObject, subpass uint32) (*RenderPassObject, uint32) {
	if len(idx) == 0 {
		return lrp, subpass
	}
	a := submit
	c := GetState(s)
	queue := c.Queues[submit.Queue]
	l := s.MemoryLayout

	f := func(o *CommandBufferCommand) {
		switch t := o.recreateData.(type) {
		case *RecreateCmdBeginRenderPassData:
			lrp = c.RenderPasses[t.RenderPass]
			subpass = 0
		case *RecreateCmdNextSubpassData:
			subpass += 1
		case *RecreateCmdEndRenderPassData:
			lrp = nil
			subpass = 0
		}
	}

	walkCommands(c, queue.PendingCommands, f)
	submitInfo := submit.PSubmits.Slice(uint64(0), uint64(submit.SubmitCount), l)
	loopLevel := 0
	for sub := 0; sub < int(idx[0])+getExtra(idx, loopLevel); sub++ {
		info := submitInfo.Index(uint64(sub), l).Read(ctx, a, s, nil)
		buffers := info.PCommandBuffers.Slice(uint64(0), uint64(info.CommandBufferCount), l)
		for cmd := 0; cmd < int(info.CommandBufferCount); cmd++ {
			buffer := buffers.Index(uint64(cmd), l).Read(ctx, a, s, nil)
			bufferObject := c.CommandBuffers[buffer]
			walkCommands(c, bufferObject.Commands, f)
		}
	}
	if !incrementLoopLevel(idx, &loopLevel) {
		return lrp, subpass
	}
	lastInfo := submitInfo.Index(uint64(idx[0]), l).Read(ctx, a, s, nil)
	lastBuffers := lastInfo.PCommandBuffers.Slice(uint64(0), uint64(lastInfo.CommandBufferCount), l)
	for cmdbuffer := 0; cmdbuffer < int(idx[1])+getExtra(idx, loopLevel); cmdbuffer++ {
		buffer := lastBuffers.Index(uint64(cmdbuffer), l).Read(ctx, a, s, nil)
		bufferObject := c.CommandBuffers[buffer]
		walkCommands(c, bufferObject.Commands, f)
	}
	if !incrementLoopLevel(idx, &loopLevel) {
		return lrp, subpass
	}
	lastBuffer := lastBuffers.Index(uint64(idx[1]), l).Read(ctx, a, s, nil)
	lastBufferObject := c.CommandBuffers[lastBuffer]
	for cmd := 0; cmd < int(idx[2])+getExtra(idx, loopLevel); cmd++ {
		f(&lastBufferObject.Commands[cmd])
	}
	if !incrementLoopLevel(idx, &loopLevel) {
		return lrp, subpass
	}
	lastCommand := lastBufferObject.Commands[idx[2]]
	if executeSubcommand, ok := (lastCommand).recreateData.(*RecreateCmdExecuteCommandsData); ok {
		for subcmdidx := 0; subcmdidx < int(idx[3])+getExtra(idx, loopLevel); subcmdidx++ {
			buffer := executeSubcommand.CommandBuffers[uint32(subcmdidx)]
			bufferObject := c.CommandBuffers[buffer]
			walkCommands(c, bufferObject.Commands, f)
		}
		if !incrementLoopLevel(idx, &loopLevel) {
			return lrp, subpass
		}
		lastsubBuffer := executeSubcommand.CommandBuffers[uint32(idx[3])]
		lastSubBufferObject := c.CommandBuffers[lastsubBuffer]
		for subcmd := 0; subcmd < int(idx[4]); subcmd++ {
			f(&lastSubBufferObject.Commands[subcmd])
		}
	}

	return lrp, subpass
}

// rebuildCommandBuffer takes the commands from commandBuffer up to, and
// including idx. It then appends any recreate* arguments to the end
// of the command buffer.
func rebuildCommandBuffer(ctx context.Context,
	cb CommandBuilder,
	commandBuffer *CommandBufferObject,
	s *api.State,
	idx api.SubCmdIdx,
	additionalCommands []interface{}) (VkCommandBuffer, []api.Cmd, []func()) {

	// DestroyResourcesAtEndOfFrame will handle this actually removing the
	// command buffer. We have no way to handle WHEN this will be done
	commandBufferId, x, cleanup := allocateNewCmdBufFromExistingOneAndBegin(ctx, cb, commandBuffer.VulkanHandle, s)

	// If we have ANY data, then we need to copy up to that point
	numCommandsToCopy := uint64(0)
	numSecondaryCmdBuffersToCopy := uint64(0)
	numSecondaryCommandsToCopy := uint64(0)
	if len(idx) > 0 {
		numCommandsToCopy = idx[0]
	}
	// If we only have 1 index, then we have to copy the last command entirely,
	// and not re-write. Otherwise the last command is a vkCmdExecuteCommands
	// and it needs to be modified.
	switch len(idx) {
	case 1:
		// Only primary commands, copies including idx
		numCommandsToCopy += 1
	case 2:
		// Ends at a secondary command buffer
		numSecondaryCmdBuffersToCopy = idx[1] + 1
	case 3:
		// Ends at a secondary command, copies including idx
		numSecondaryCmdBuffersToCopy = idx[1]
		numSecondaryCommandsToCopy = idx[2] + 1
	}

	for i := 0; i < int(numCommandsToCopy); i++ {
		cmd := commandBuffer.Commands[i]
		c, a := AddCommand(ctx, cb, commandBufferId, s, cmd.recreateData)
		x = append(x, a)
		cleanup = append(cleanup, c)
	}

	if numSecondaryCommandsToCopy != uint64(0) ||
		numSecondaryCmdBuffersToCopy != uint64(0) {

		newCmdExecuteCommandsData := &RecreateCmdExecuteCommandsData{
			CommandBuffers: U32ːVkCommandBufferᵐ{},
		}
		pcmd := commandBuffer.Commands[idx[0]]
		execCmdData, ok := pcmd.recreateData.(*RecreateCmdExecuteCommandsData)
		if !ok {
			panic("Rebuild command buffer including secondary commands at a primary " +
				"command other than VkCmdExecuteCommands or RecreateCmdExecuteCommands")
		}
		for scbi := uint32(0); scbi < uint32(numSecondaryCmdBuffersToCopy); scbi++ {
			newCmdExecuteCommandsData.CommandBuffers[scbi] = execCmdData.CommandBuffers.Get(scbi)
		}
		if numSecondaryCommandsToCopy != uint64(0) {
			lastSecCmdBuf := execCmdData.CommandBuffers.Get(uint32(idx[1]))
			newSecCmdBuf, extraCmds, extraCleanup := allocateNewCmdBufFromExistingOneAndBegin(ctx, cb, lastSecCmdBuf, s)
			x = append(x, extraCmds...)
			cleanup = append(cleanup, extraCleanup...)
			for sci := 0; sci < int(numSecondaryCommandsToCopy); sci++ {
				secCmd := GetState(s).CommandBuffers.Get(lastSecCmdBuf).Commands[sci]
				newCleanups, newSecCmds := AddCommand(ctx, cb, newSecCmdBuf, s, secCmd.recreateData)
				x = append(x, newSecCmds)
				cleanup = append(cleanup, newCleanups)
			}
			x = append(x, cb.VkEndCommandBuffer(newSecCmdBuf, VkResult_VK_SUCCESS))
			newCmdExecuteCommandsData.CommandBuffers[uint32(idx[1])] = newSecCmdBuf
		}
		cleanupNewExecSecCmds, newExecSecCmds := AddCommand(
			ctx, cb, commandBufferId, s, newCmdExecuteCommandsData)
		cleanup = append(cleanup, cleanupNewExecSecCmds)
		x = append(x, newExecSecCmds)
	}

	for i := range additionalCommands {
		c, a := AddCommand(ctx, cb, commandBufferId, s, additionalCommands[i])
		x = append(x, a)
		cleanup = append(cleanup, c)
	}
	x = append(x,
		cb.VkEndCommandBuffer(commandBufferId, VkResult_VK_SUCCESS))
	return VkCommandBuffer(commandBufferId), x, cleanup
}

// cutCommandBuffer rebuilds the given VkQueueSubmit atom.
// It will re-write the submission so that it ends at
// idx. It writes any new atoms to transform.Writer.
// It will make sure that if the replay were to stop at the given
// index it would remain valid. This means closing any open
// RenderPasses.
func cutCommandBuffer(ctx context.Context, id api.CmdID,
	a *VkQueueSubmit, idx api.SubCmdIdx, out transform.Writer) {
	cb := CommandBuilder{Thread: a.Thread()}
	s := out.State()
	c := GetState(s)
	l := s.MemoryLayout
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	submitInfo := a.PSubmits.Slice(uint64(0), uint64(a.SubmitCount), l)
	skipAll := len(idx) == 0

	// Notes:
	// - We should walk/finish all unfinished render passes
	// idx[0] is the submission index
	// idx[1] is the primary command-buffer index in the submission
	// idx[2] is the command index in the primary command-buffer
	// idx[3] is the secondary command buffer index inside a vkCmdExecuteCommands
	// idx[4] is the secondary command inside the secondary command-buffer
	submitCopy := cb.VkQueueSubmit(a.Queue, a.SubmitCount, a.PSubmits, a.Fence, a.Result)
	submitCopy.Extras().MustClone(a.Extras().All()...)

	newCommandBuffers := make([]VkCommandBuffer, 1)
	lastSubmit := uint64(0)
	lastCommandBuffer := uint64(0)
	if !skipAll {
		lastSubmit = idx[0]
		if len(idx) > 1 {
			lastCommandBuffer = idx[1]
		}
	}
	submitCopy.SubmitCount = uint32(lastSubmit + 1)
	newSubmits := make([]VkSubmitInfo, lastSubmit+1)
	for i := 0; i < int(lastSubmit)+1; i++ {
		newSubmits[i] = submitInfo.Index(uint64(i), l).Read(ctx, a, s, nil)
	}
	newSubmits[lastSubmit].CommandBufferCount = uint32(lastCommandBuffer + 1)

	newCommandBuffers = make([]VkCommandBuffer, lastCommandBuffer+1)
	buffers := newSubmits[lastSubmit].PCommandBuffers.Slice(uint64(0), uint64(newSubmits[lastSubmit].CommandBufferCount), l)
	for i := 0; i < int(lastCommandBuffer)+1; i++ {
		newCommandBuffers[i] = buffers.Index(uint64(i), l).Read(ctx, a, s, nil)
	}

	var lrp *RenderPassObject
	lsp := uint32(0)
	if lastDrawInfo, ok := c.LastDrawInfos[a.Queue]; ok {
		if lastDrawInfo.InRenderPass {
			lrp = lastDrawInfo.RenderPass
			lsp = lastDrawInfo.LastSubpass
		} else {
			lrp = nil
			lsp = 0
		}
	}
	lrp, lsp = resolveCurrentRenderPass(ctx, s, a, idx, lrp, lsp)

	extraCommands := make([]interface{}, 0)
	if lrp != nil {
		numSubpasses := uint32(len(lrp.SubpassDescriptions))
		for i := 0; uint32(i) < numSubpasses-lsp-1; i++ {
			extraCommands = append(extraCommands, &RecreateCmdNextSubpassData{})
		}
		extraCommands = append(extraCommands, &RecreateCmdEndRenderPassData{})
	}

	cmdBuffer := c.CommandBuffers[newCommandBuffers[lastCommandBuffer]]
	subIdx := make(api.SubCmdIdx, 0)
	if !skipAll {
		subIdx = idx[2:]
	}
	b, newCommands, cleanup :=
		rebuildCommandBuffer(ctx, cb, cmdBuffer, s, subIdx, extraCommands)
	newCommandBuffers[lastCommandBuffer] = b

	bufferMemory := s.AllocDataOrPanic(ctx, newCommandBuffers)
	newSubmits[lastSubmit].PCommandBuffers = NewVkCommandBufferᶜᵖ(bufferMemory.Ptr())

	newSubmitData := s.AllocDataOrPanic(ctx, newSubmits)
	submitCopy.PSubmits = NewVkSubmitInfoᶜᵖ(newSubmitData.Ptr())
	submitCopy.AddRead(bufferMemory.Data()).AddRead(newSubmitData.Data())

	for _, c := range newCommands {
		out.MutateAndWrite(ctx, api.CmdNoID, c)
	}

	out.MutateAndWrite(ctx, id, submitCopy)

	for _, f := range cleanup {
		f()
	}

	bufferMemory.Free()
	newSubmitData.Free()
}

func (t *VulkanTerminator) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) {
	if t.stopped {
		return
	}

	doCut := false
	cutIndex := api.SubCmdIdx(nil)
	if rng, ok := t.syncData.CommandRanges[id]; ok {
		for k, v := range rng.Ranges {
			if api.CmdID(k) > t.lastRequest {
				doCut = true
			} else {
				if len(cutIndex) == 0 || cutIndex.LessThan(v) {
					// Make a copy of v, we do not want to modify the original.
					cutIndex = append(api.SubCmdIdx(nil), v...)
					cutIndex.Decrement()
				}
			}
		}
	}

	// If we have been requested to cut at a particular subindex,
	// then do that instead of cutting at the derived cutIndex.
	// It is guaranteed to be safe as long as the requestedSubIndex is
	// less than the calculated one (i.e. we are cutting more)
	if len(t.requestSubIndex) > 1 && t.requestSubIndex[0] == uint64(id) {
		if len(cutIndex) == 0 || !cutIndex.LessThan(t.requestSubIndex[1:]) {
			cutIndex = t.requestSubIndex[1:]
			doCut = true
		}
	}

	// We have to cut somewhere
	if doCut {
		cutCommandBuffer(ctx, id, cmd.(*VkQueueSubmit), cutIndex, out)
	} else {
		out.MutateAndWrite(ctx, id, cmd)
	}

	if id == t.lastRequest {
		t.stopped = true
	}
}

func (t *VulkanTerminator) Flush(ctx context.Context, out transform.Writer) {}
