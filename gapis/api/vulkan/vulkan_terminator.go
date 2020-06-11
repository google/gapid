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
// This takes advantage of the fact that all commands will be in order.
type VulkanTerminator struct {
	lastRequest     api.CmdID
	requestSubIndex []uint64
	stopped         bool
	syncData        *sync.Data
}

var _ transform.Terminator = &VulkanTerminator{}

func newVulkanTerminator(ctx context.Context, capture *path.Capture) (*VulkanTerminator, error) {
	s, err := resolve.SyncData(ctx, capture)
	if err != nil {
		return nil, err
	}
	return &VulkanTerminator{api.CmdID(0), make([]uint64, 0), false, s}, nil
}

// Add adds the command with identifier id to the set of commands that must be
// seen before the VulkanTerminator will consume all commands (excluding the EOS
// command).
func (t *VulkanTerminator) Add(ctx context.Context, id api.CmdID, subcommand api.SubCmdIdx) error {
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
	t.lastRequest = id

	return nil
}

func walkCommands(s *State,
	commands U32ːCommandReferenceʳDense_ᵐ,
	callback func(CommandReferenceʳ)) {
	for _, c := range commands.Keys() {
		callback(commands.Get(c))
		if commands.Get(c).Type() == CommandType_cmd_vkCmdExecuteCommands {
			execSub := s.CommandBuffers().Get(commands.Get(c).Buffer()).BufferCommands().VkCmdExecuteCommands().Get(commands.Get(c).MapIndex())
			for _, k := range execSub.CommandBuffers().Keys() {
				cbc := s.CommandBuffers().Get(execSub.CommandBuffers().Get(k))
				walkCommands(s, cbc.CommandReferences(), callback)
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
	*loopLevel++
	return true
}

// resolveCurrentRenderPass walks all of the current and pending commands
// to determine what renderpass we are in after the idx'th subcommand
func resolveCurrentRenderPass(ctx context.Context, s *api.GlobalState, submit *VkQueueSubmit,
	idx api.SubCmdIdx, lrp RenderPassObjectʳ, subpass uint32) (RenderPassObjectʳ, uint32) {
	if len(idx) == 0 {
		return lrp, subpass
	}
	a := submit
	c := GetState(s)
	l := s.MemoryLayout

	f := func(o CommandReferenceʳ) {
		switch o.Type() {
		case CommandType_cmd_vkCmdBeginRenderPass:
			t := c.CommandBuffers().Get(o.Buffer()).BufferCommands().VkCmdBeginRenderPass().Get(o.MapIndex())
			lrp = c.RenderPasses().Get(t.RenderPass())
			subpass = 0
		case CommandType_cmd_vkCmdNextSubpass:
			subpass++
		case CommandType_cmd_vkCmdEndRenderPass:
			lrp = NilRenderPassObjectʳ
			subpass = 0
		}
	}

	submitInfo := submit.PSubmits().Slice(0, uint64(submit.SubmitCount()), l)
	loopLevel := 0
	for sub := 0; sub < int(idx[0])+getExtra(idx, loopLevel); sub++ {
		info := submitInfo.Index(uint64(sub)).MustRead(ctx, a, s, nil)[0]
		buffers := info.PCommandBuffers().Slice(0, uint64(info.CommandBufferCount()), l).MustRead(ctx, a, s, nil)
		for _, buffer := range buffers {
			bufferObject := c.CommandBuffers().Get(buffer)
			walkCommands(c, bufferObject.CommandReferences(), f)
		}
	}
	if !incrementLoopLevel(idx, &loopLevel) {
		return lrp, subpass
	}
	lastInfo := submitInfo.Index(uint64(idx[0])).MustRead(ctx, a, s, nil)[0]
	lastBuffers := lastInfo.PCommandBuffers().Slice(0, uint64(lastInfo.CommandBufferCount()), l)
	for cmdbuffer := 0; cmdbuffer < int(idx[1])+getExtra(idx, loopLevel); cmdbuffer++ {
		buffer := lastBuffers.Index(uint64(cmdbuffer)).MustRead(ctx, a, s, nil)[0]
		bufferObject := c.CommandBuffers().Get(buffer)
		walkCommands(c, bufferObject.CommandReferences(), f)
	}
	if !incrementLoopLevel(idx, &loopLevel) {
		return lrp, subpass
	}
	lastBuffer := lastBuffers.Index(uint64(idx[1])).MustRead(ctx, a, s, nil)[0]
	lastBufferObject := c.CommandBuffers().Get(lastBuffer)
	for cmd := 0; cmd < int(idx[2])+getExtra(idx, loopLevel); cmd++ {
		f(lastBufferObject.CommandReferences().Get(uint32(cmd)))
	}
	if !incrementLoopLevel(idx, &loopLevel) {
		return lrp, subpass
	}
	lastCommand := lastBufferObject.CommandReferences().Get(uint32(idx[2]))

	if lastCommand.Type() == CommandType_cmd_vkCmdExecuteCommands {
		executeSubcommand := c.CommandBuffers().Get(lastCommand.Buffer()).BufferCommands().VkCmdExecuteCommands().Get(lastCommand.MapIndex())
		for subcmdidx := 0; subcmdidx < int(idx[3])+getExtra(idx, loopLevel); subcmdidx++ {
			buffer := executeSubcommand.CommandBuffers().Get(uint32(subcmdidx))
			bufferObject := c.CommandBuffers().Get(buffer)
			walkCommands(c, bufferObject.CommandReferences(), f)
		}
		if !incrementLoopLevel(idx, &loopLevel) {
			return lrp, subpass
		}
		lastsubBuffer := executeSubcommand.CommandBuffers().Get(uint32(idx[3]))
		lastSubBufferObject := c.CommandBuffers().Get(lastsubBuffer)
		for subcmd := 0; subcmd < int(idx[4]); subcmd++ {
			f(lastSubBufferObject.CommandReferences().Get(uint32(subcmd)))
		}
	}

	return lrp, subpass
}

// rebuildCommandBuffer takes the commands from commandBuffer up to, and
// including idx. It then appends any recreate* arguments to the end
// of the command buffer.
func rebuildCommandBuffer(ctx context.Context,
	cb CommandBuilder,
	commandBuffer CommandBufferObjectʳ,
	s *api.GlobalState,
	idx api.SubCmdIdx,
	additionalCommands []interface{}) (VkCommandBuffer, []api.Cmd, []func()) {

	a := s.Arena // TODO: Use a temporary arena?

	// DestroyResourcesAtEndOfFrame will handle this actually removing the
	// command buffer. We have no way to handle WHEN this will be done
	commandBufferID, x, cleanup := allocateNewCmdBufFromExistingOneAndBegin(ctx, cb, commandBuffer.VulkanHandle(), s)

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
		numCommandsToCopy++
	case 2:
		// Ends at a secondary command buffer
		numSecondaryCmdBuffersToCopy = idx[1] + 1
	case 3:
		// Ends at a secondary command, copies including idx
		numSecondaryCmdBuffersToCopy = idx[1]
		numSecondaryCommandsToCopy = idx[2] + 1
	}

	for i := uint32(0); i < uint32(numCommandsToCopy); i++ {
		cmd := commandBuffer.CommandReferences().Get(i)
		c, a, _ := AddCommand(ctx, cb, commandBufferID, s, s, GetCommandArgs(ctx, cmd, GetState(s)))
		x = append(x, a)
		cleanup = append(cleanup, c)
	}

	if numSecondaryCommandsToCopy != 0 || numSecondaryCmdBuffersToCopy != 0 {
		newCmdExecuteCommandsData := NewVkCmdExecuteCommandsArgs(a,
			NewU32ːVkCommandBufferDense_ᵐ(a), // CommandBuffers
		)
		pcmd := commandBuffer.CommandReferences().Get(uint32(idx[0]))
		execCmdData, ok := GetCommandArgs(ctx, pcmd, GetState(s)).(VkCmdExecuteCommandsArgsʳ)
		if !ok {
			panic("Rebuild command buffer including secondary commands at a primary " +
				"command other than VkCmdExecuteCommands")
		}
		for scbi := uint32(0); scbi < uint32(numSecondaryCmdBuffersToCopy); scbi++ {
			newCmdExecuteCommandsData.CommandBuffers().Add(scbi, execCmdData.CommandBuffers().Get(scbi))
		}
		if numSecondaryCommandsToCopy != 0 {
			lastSecCmdBuf := execCmdData.CommandBuffers().Get(uint32(idx[1]))
			newSecCmdBuf, extraCmds, extraCleanup := allocateNewCmdBufFromExistingOneAndBegin(ctx, cb, lastSecCmdBuf, s)
			x = append(x, extraCmds...)
			cleanup = append(cleanup, extraCleanup...)
			for sci := uint32(0); sci < uint32(numSecondaryCommandsToCopy); sci++ {
				secCmd := GetState(s).CommandBuffers().Get(lastSecCmdBuf).CommandReferences().Get(sci)
				newCleanups, newSecCmds, _ := AddCommand(ctx, cb, newSecCmdBuf, s, s, GetCommandArgs(ctx, secCmd, GetState(s)))
				x = append(x, newSecCmds)
				cleanup = append(cleanup, newCleanups)
			}
			x = append(x, cb.VkEndCommandBuffer(newSecCmdBuf, VkResult_VK_SUCCESS))
			newCmdExecuteCommandsData.CommandBuffers().Add(uint32(idx[1]), newSecCmdBuf)
		}

		// If we use AddCommand, it will check for the existence of the command buffer,
		// which wont yet exist (because it hasn't been mutated yet)
		commandBufferData, commandBufferCount := unpackMap(ctx, s, newCmdExecuteCommandsData.CommandBuffers())
		newExecSecCmds := cb.VkCmdExecuteCommands(commandBufferID,
			commandBufferCount,
			commandBufferData.Ptr(),
		).AddRead(commandBufferData.Data())

		cleanup = append(cleanup, func() {
			commandBufferData.Free()
		})
		x = append(x, newExecSecCmds)
	}

	for i := range additionalCommands {
		c, a, _ := AddCommand(ctx, cb, commandBufferID, s, s, additionalCommands[i])
		x = append(x, a)
		cleanup = append(cleanup, c)
	}
	x = append(x,
		cb.VkEndCommandBuffer(commandBufferID, VkResult_VK_SUCCESS))
	return VkCommandBuffer(commandBufferID), x, cleanup
}

// cutCommandBuffer rebuilds the given VkQueueSubmit command.
// It will re-write the submission so that it ends at
// idx. It writes any new commands to transform.Writer.
// It will make sure that if the replay were to stop at the given
// index it would remain valid. This means closing any open
// RenderPasses.
func cutCommandBuffer(ctx context.Context, id api.CmdID,
	a *VkQueueSubmit, idx api.SubCmdIdx, out transform.Writer) {
	s := out.State()
	cb := CommandBuilder{Thread: a.Thread(), Arena: s.Arena}
	c := GetState(s)
	l := s.MemoryLayout
	o := a.Extras().Observations()
	o.ApplyReads(s.Memory.ApplicationPool())
	submitInfo := a.PSubmits().Slice(0, uint64(a.SubmitCount()), l)
	skipAll := len(idx) == 0

	// Notes:
	// - We should walk/finish all unfinished render passes
	// idx[0] is the submission index
	// idx[1] is the primary command-buffer index in the submission
	// idx[2] is the command index in the primary command-buffer
	// idx[3] is the secondary command buffer index inside a vkCmdExecuteCommands
	// idx[4] is the secondary command inside the secondary command-buffer
	submitCopy := cb.VkQueueSubmit(a.Queue(), a.SubmitCount(), a.PSubmits(), a.Fence(), a.Result())
	submitCopy.Extras().MustClone(a.Extras().All()...)

	lastSubmit := uint64(0)
	lastCommandBuffer := uint64(0)
	if !skipAll {
		lastSubmit = idx[0]
		if len(idx) > 1 {
			lastCommandBuffer = idx[1]
		}
	}
	submitCopy.SetSubmitCount(uint32(lastSubmit + 1))
	newSubmits := submitInfo.Slice(0, lastSubmit+1).MustRead(ctx, a, s, nil)
	newSubmits[lastSubmit].SetCommandBufferCount(uint32(lastCommandBuffer + 1))

	newCommandBuffers := newSubmits[lastSubmit].PCommandBuffers().Slice(0, lastCommandBuffer+1, l).MustRead(ctx, a, s, nil)

	var lrp RenderPassObjectʳ
	lsp := uint32(0)
	if lastDrawInfo, ok := c.LastDrawInfos().Lookup(a.Queue()); ok {
		if lastDrawInfo.InRenderPass() {
			lrp = lastDrawInfo.RenderPass()
			lsp = lastDrawInfo.LastSubpass()
		} else {
			lrp = NilRenderPassObjectʳ
			lsp = 0
		}
	}
	lrp, lsp = resolveCurrentRenderPass(ctx, s, a, idx, lrp, lsp)

	extraCommands := make([]interface{}, 0)
	if !lrp.IsNil() {
		numSubpasses := uint32(lrp.SubpassDescriptions().Len())
		for i := 0; uint32(i) < numSubpasses-lsp-1; i++ {
			extraCommands = append(extraCommands,
				NewVkCmdNextSubpassArgsʳ(s.Arena, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE))
		}
		extraCommands = append(extraCommands, NewVkCmdEndRenderPassArgsʳ(s.Arena))
	}
	var cleanup []func()
	cmdBuffer := c.CommandBuffers().Get(newCommandBuffers[lastCommandBuffer])
	subIdx := make(api.SubCmdIdx, 0)
	allocResults := []api.AllocResult{}
	if len(idx) > 1 {
		if !skipAll {
			subIdx = idx[2:]
		}
		var b VkCommandBuffer
		var newCommands []api.Cmd

		b, newCommands, cleanup =
			rebuildCommandBuffer(ctx, cb, cmdBuffer, s, subIdx, extraCommands)
		newCommandBuffers[lastCommandBuffer] = b

		bufferMemory := s.AllocDataOrPanic(ctx, newCommandBuffers)
		newSubmits[lastSubmit].SetPCommandBuffers(NewVkCommandBufferᶜᵖ(bufferMemory.Ptr()))

		newSubmitData := s.AllocDataOrPanic(ctx, newSubmits)
		submitCopy.SetPSubmits(NewVkSubmitInfoᶜᵖ(newSubmitData.Ptr()))
		submitCopy.AddRead(bufferMemory.Data()).AddRead(newSubmitData.Data())
		allocResults = append(allocResults, bufferMemory)
		allocResults = append(allocResults, newSubmitData)

		for _, c := range newCommands {
			out.MutateAndWrite(ctx, api.CmdNoID, c)
		}
	} else {
		submitCopy.SetSubmitCount(uint32(lastSubmit + 1))
	}

	out.MutateAndWrite(ctx, id, submitCopy)

	for _, f := range cleanup {
		f()
	}
	for _, res := range allocResults {
		res.Free()
	}
}

func (t *VulkanTerminator) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	if t.stopped {
		return nil
	}

	doCut := false
	cutIndex := api.SubCmdIdx(nil)
	// If we have been requested to cut at a particular subindex,
	// then do that instead of cutting at the derived cutIndex.
	// It is guaranteed to be safe as long as the requestedSubIndex is
	// less than the calculated one (i.e. we are cutting more)
	if len(t.requestSubIndex) > 1 && t.requestSubIndex[0] == uint64(id) && t.syncData.SubcommandLookup.Value(t.requestSubIndex) != nil {
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
	return nil
}

func (t *VulkanTerminator) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *VulkanTerminator) PreLoop(ctx context.Context, output transform.Writer)  {}
func (t *VulkanTerminator) PostLoop(ctx context.Context, output transform.Writer) {}
func (t *VulkanTerminator) BuffersCommands() bool                                 { return false }
