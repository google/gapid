// Copyright (C) 2020 Google Inc.
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

	"github.com/google/gapid/core/memory/arena"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/replay/builder"
)

// InsertionCommand is a temporary command
// that is expected to be replaced by a down-stream transform.
type InsertionCommand struct {
	cmdBuffer             VkCommandBuffer
	pendingCommandBuffers []VkCommandBuffer
	idx                   api.SubCmdIdx
	callee                api.Cmd
}

// Interface check
var _ api.Cmd = &InsertionCommand{}

func (*InsertionCommand) Mutate(ctx context.Context, cmd api.CmdID, g *api.GlobalState, b *builder.Builder, w api.StateWatcher) error {
	if b != nil {
		return fmt.Errorf("This command should have been replaced before it got to the builder")
	}
	return nil
}

func (s *InsertionCommand) Thread() uint64 {
	return s.callee.Thread()
}

func (s *InsertionCommand) SetThread(c uint64) {
	s.callee.SetThread(c)
}

// CmdName returns the name of the command.
func (s *InsertionCommand) CmdName() string {
	return "CommandBufferInsertion"
}

func (s *InsertionCommand) CmdParams() api.Properties {
	return api.Properties{}
}

func (s *InsertionCommand) CmdResult() *api.Property {
	return nil
}

func (s *InsertionCommand) CmdFlags() api.CmdFlags {
	return 0
}

func (s *InsertionCommand) Extras() *api.CmdExtras {
	return nil
}

func (s *InsertionCommand) Clone(a arena.Arena) api.Cmd {
	return &InsertionCommand{
		s.cmdBuffer,
		append([]VkCommandBuffer{}, s.pendingCommandBuffers...),
		s.idx,
		s.callee.Clone(a),
	}
}

func (s *InsertionCommand) Alive() bool {
	return true
}

func (s *InsertionCommand) Terminated() bool {
	return true
}

func (s *InsertionCommand) SetTerminated(bool) {
}

func (s *InsertionCommand) API() api.API {
	return s.callee.API()
}

func (s *commandSplitter) MustAllocReadDataForSubmit(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := g.AllocDataOrPanic(ctx, v...)
	s.readMemoriesForSubmit = append(s.readMemoriesForSubmit, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (s *commandSplitter) MustAllocReadDataForCmd(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := g.AllocDataOrPanic(ctx, v...)
	s.readMemoriesForCmd = append(s.readMemoriesForCmd, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (s *commandSplitter) MustAllocWriteDataForCmd(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := g.AllocDataOrPanic(ctx, v...)
	s.writeMemoriesForCmd = append(s.writeMemoriesForCmd, &allocateResult)
	return allocateResult
}

func (s *commandSplitter) WriteCommand(ctx context.Context, cmd api.Cmd, out transform.Writer) error {
	for i := range s.readMemoriesForCmd {
		cmd.Extras().GetOrAppendObservations().AddRead(s.readMemoriesForCmd[i].Data())
	}
	for i := range s.writeMemoriesForCmd {
		cmd.Extras().GetOrAppendObservations().AddWrite(s.writeMemoriesForCmd[i].Data())
	}
	s.readMemoriesForCmd = []*api.AllocResult{}
	s.writeMemoriesForCmd = []*api.AllocResult{}
	return out.MutateAndWrite(ctx, api.CmdNoID, cmd)
}

// commandSplitter is a transform that will re-write command-buffers and insert replacement
// commands at the correct locations in the stream for downstream transforms to replace.
// See: https://www.khronos.org/registry/vulkan/specs/1.1-extensions/html/vkspec.html#renderpass
// and https://www.khronos.org/registry/vulkan/specs/1.1-extensions/html/vkspec.html#pipelines-graphics
// to understand how/why we have to split these.
type commandSplitter struct {
	lastRequest      api.SubCmdIdx
	requestsSubIndex []api.SubCmdIdx

	readMemoriesForSubmit []*api.AllocResult
	readMemoriesForCmd    []*api.AllocResult
	writeMemoriesForCmd   []*api.AllocResult
	pool                  VkCommandPool

	thisRenderPass    VkCmdBeginRenderPassArgsʳ
	currentRenderPass [][3]VkRenderPass
	thisSubpass       int

	splitRenderPasses      map[VkRenderPass][][3]VkRenderPass
	fixedGraphicsPipelines map[VkPipeline]VkPipeline

	pendingCommandBuffers []VkCommandBuffer
}

func NewCommandSplitter(ctx context.Context) *commandSplitter {
	return &commandSplitter{api.SubCmdIdx{}, make([]api.SubCmdIdx, 0),
		make([]*api.AllocResult, 0), make([]*api.AllocResult, 0), make([]*api.AllocResult, 0),
		0, NilVkCmdBeginRenderPassArgsʳ, make([][3]VkRenderPass, 0), 0,
		make(map[VkRenderPass][][3]VkRenderPass),
		make(map[VkPipeline]VkPipeline),
		make([]VkCommandBuffer, 0)}
}

// Add adds the command with identifier id to the set of commands that will be split.
func (t *commandSplitter) Split(ctx context.Context, id api.SubCmdIdx) error {
	t.requestsSubIndex = append(t.requestsSubIndex, append(api.SubCmdIdx{}, id...))
	if t.lastRequest.LessThan(id) {
		t.lastRequest = append(api.SubCmdIdx{}, id...)
	}

	return nil
}

func (t *commandSplitter) getCommandPool(ctx context.Context, queueSubmit *VkQueueSubmit, out transform.Writer) (VkCommandPool, error) {
	if t.pool != 0 {
		return t.pool, nil
	}
	s := out.State()
	a := s.Arena
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: s.Arena}
	st := GetState(s)
	queue := st.Queues().Get(queueSubmit.Queue())
	t.pool = VkCommandPool(newUnusedID(false, func(x uint64) bool { ok := GetState(s).CommandPools().Contains(VkCommandPool(x)); return ok }))

	poolCreateInfo := NewVkCommandPoolCreateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
		VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
		queue.Family(), // queueFamilyIndex
	)

	if err := t.WriteCommand(ctx, cb.VkCreateCommandPool(
		queue.Device(),
		t.MustAllocReadDataForCmd(ctx, s, poolCreateInfo).Ptr(),
		memory.Nullptr,
		t.MustAllocWriteDataForCmd(ctx, s, t.pool).Ptr(),
		VkResult_VK_SUCCESS,
	), out); err != nil {
		return VkCommandPool(0), err
	}
	return t.pool, nil
}

func (t *commandSplitter) getStartedCommandBuffer(ctx context.Context, queueSubmit *VkQueueSubmit, out transform.Writer) (VkCommandBuffer, error) {
	s := out.State()
	a := s.Arena
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: a}
	vs := GetState(s)
	queue := vs.Queues().Get(queueSubmit.Queue())

	commandPoolID, err := t.getCommandPool(ctx, queueSubmit, out)
	if err != nil {
		return VkCommandBuffer(0), err
	}

	commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		commandPoolID,                                                  // commandPool
		VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
		1, // commandBufferCount
	)
	commandBufferID := VkCommandBuffer(newUnusedID(true, func(x uint64) bool { ok := GetState(s).CommandBuffers().Contains(VkCommandBuffer(x)); return ok }))

	if err := t.WriteCommand(ctx,
		cb.VkAllocateCommandBuffers(
			queue.Device(),
			t.MustAllocReadDataForCmd(ctx, s, commandBufferAllocateInfo).Ptr(),
			t.MustAllocWriteDataForCmd(ctx, s, commandBufferID).Ptr(),
			VkResult_VK_SUCCESS,
		), out); err != nil {
		return VkCommandBuffer(0), err
	}

	commandBufferBegin := NewVkCommandBufferBeginInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                   // pNext
		0,                                                           // flags
		NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr), // pInheritanceInfo
	)
	if err := t.WriteCommand(ctx,
		cb.VkBeginCommandBuffer(
			commandBufferID,
			t.MustAllocReadDataForCmd(ctx, s, commandBufferBegin).Ptr(),
			VkResult_VK_SUCCESS,
		), out); err != nil {
		return VkCommandBuffer(0), err
	}

	return commandBufferID, nil
}

const VK_ATTACHMENT_UNUSED = uint32(0xFFFFFFFF)

func (t *commandSplitter) splitRenderPass(ctx context.Context, rp RenderPassObjectʳ, out transform.Writer) [][3]VkRenderPass {
	s := out.State()
	st := GetState(s)
	arena := s.Arena

	if rp, ok := t.splitRenderPasses[rp.VulkanHandle()]; ok {
		return rp
	}

	handles := make([][3]VkRenderPass, 0)
	currentLayouts := make(map[uint32]VkImageLayout)
	for i := uint32(0); i < uint32(rp.AttachmentDescriptions().Len()); i++ {
		currentLayouts[i] = rp.AttachmentDescriptions().Get(i).InitialLayout()
	}
	sb := st.newStateBuilder(ctx, newTransformerOutput(out))

	for i := uint32(0); i < uint32(rp.SubpassDescriptions().Len()); i++ {
		subpassHandles := [3]VkRenderPass{}
		lastSubpass := (i == uint32(rp.SubpassDescriptions().Len()-1))
		firstSubpass := (i == 0)

		patchFinalLayout := func(rpo RenderPassObjectʳ, ar U32ːVkAttachmentReferenceᵐ) {
			for k := 0; k < len(ar.All()); k++ {
				ia := ar.Get(uint32(k))
				if ia.Attachment() != VK_ATTACHMENT_UNUSED {
					currentLayouts[ia.Attachment()] = ia.Layout()
					ad := rpo.AttachmentDescriptions().Get(ia.Attachment())
					ad.SetFinalLayout(ia.Layout())
					rpo.AttachmentDescriptions().Add(ia.Attachment(), ad)
				}
			}
		}

		const (
			PATCH_LOAD uint32 = 1 << iota
			PATCH_STORE
			PATCH_FINAL_LAYOUT
		)

		patchAllDescriptions := func(rpo RenderPassObjectʳ, patch uint32) {
			for j := uint32(0); j < uint32(len(currentLayouts)); j++ {
				ad := rpo.AttachmentDescriptions().Get(j)
				ad.SetInitialLayout(currentLayouts[j])
				if 0 != (patch & PATCH_FINAL_LAYOUT) {
					ad.SetFinalLayout(currentLayouts[j])
				}
				if 0 != (patch & PATCH_LOAD) {
					ad.SetLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD)
					ad.SetStencilLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD)
				}
				if 0 != (patch & PATCH_STORE) {
					ad.SetStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
					ad.SetStencilStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
				}
				rpo.AttachmentDescriptions().Add(j, ad)
			}
		}

		{
			rp1 := rp.Clone(arena, api.CloneContext{})
			rp1.SetVulkanHandle(
				VkRenderPass(newUnusedID(true, func(x uint64) bool {
					return st.RenderPasses().Contains(VkRenderPass(x))
				})))

			spd := rp1.SubpassDescriptions().Get(i)
			patch := uint32(0)
			if !firstSubpass {
				patch = PATCH_LOAD
			}
			patchAllDescriptions(rp1, patch|PATCH_STORE|PATCH_FINAL_LAYOUT)
			patchFinalLayout(rp1, spd.InputAttachments())
			patchFinalLayout(rp1, spd.ColorAttachments())
			spd.ResolveAttachments().Clear()
			if !spd.DepthStencilAttachment().IsNil() {
				ia := spd.DepthStencilAttachment()
				if ia.Attachment() != VK_ATTACHMENT_UNUSED {
					currentLayouts[ia.Attachment()] = ia.Layout()
					ad := rp1.AttachmentDescriptions().Get(ia.Attachment())
					ad.SetFinalLayout(ia.Layout())
					rp1.AttachmentDescriptions().Add(ia.Attachment(), ad)
				}
			}
			spd.PreserveAttachments().Clear()

			rp1.SubpassDescriptions().Clear()
			rp1.SubpassDescriptions().Add(0, spd)
			rp1.SubpassDependencies().Clear()
			sb.createRenderPass(rp1)
			subpassHandles[0] = rp1.VulkanHandle()
		}

		{
			rp2 := rp.Clone(arena, api.CloneContext{})
			rp2.SetVulkanHandle(
				VkRenderPass(newUnusedID(true, func(x uint64) bool {
					return st.RenderPasses().Contains(VkRenderPass(x))
				})))
			spd := rp2.SubpassDescriptions().Get(i)
			patchAllDescriptions(rp2, PATCH_LOAD|PATCH_STORE|PATCH_FINAL_LAYOUT)
			patchFinalLayout(rp2, spd.InputAttachments())
			patchFinalLayout(rp2, spd.ColorAttachments())
			spd.ResolveAttachments().Clear()
			if !spd.DepthStencilAttachment().IsNil() {
				ia := spd.DepthStencilAttachment()
				if ia.Attachment() != VK_ATTACHMENT_UNUSED {
					currentLayouts[ia.Attachment()] = ia.Layout()
					ad := rp2.AttachmentDescriptions().Get(ia.Attachment())
					ad.SetFinalLayout(ia.Layout())
					rp2.AttachmentDescriptions().Add(ia.Attachment(), ad)
				}
			}
			spd.PreserveAttachments().Clear()
			rp2.SubpassDescriptions().Clear()
			rp2.SubpassDescriptions().Add(0, spd)
			rp2.SubpassDependencies().Clear()
			sb.createRenderPass(rp2)
			subpassHandles[1] = rp2.VulkanHandle()
		}

		{
			rp3 := rp.Clone(arena, api.CloneContext{})
			rp3.SetVulkanHandle(
				VkRenderPass(newUnusedID(true, func(x uint64) bool {
					return st.RenderPasses().Contains(VkRenderPass(x))
				})))
			spd := rp3.SubpassDescriptions().Get(i)
			patch := PATCH_LOAD
			if !lastSubpass {
				patch |= PATCH_STORE | PATCH_FINAL_LAYOUT
			}
			patchAllDescriptions(rp3, patch)
			if !lastSubpass {
				patchFinalLayout(rp3, spd.InputAttachments())
				patchFinalLayout(rp3, spd.ColorAttachments())
				if !spd.DepthStencilAttachment().IsNil() {
					ia := spd.DepthStencilAttachment()
					if ia.Attachment() != VK_ATTACHMENT_UNUSED {
						currentLayouts[ia.Attachment()] = ia.Layout()
						ad := rp3.AttachmentDescriptions().Get(ia.Attachment())
						ad.SetFinalLayout(ia.Layout())
						rp3.AttachmentDescriptions().Add(ia.Attachment(), ad)
					}
				}
			}
			spd.PreserveAttachments().Clear()
			rp3.SubpassDescriptions().Clear()
			rp3.SubpassDescriptions().Add(0, spd)
			rp3.SubpassDependencies().Clear()
			sb.createRenderPass(rp3)
			subpassHandles[2] = rp3.VulkanHandle()
		}

		handles = append(handles, subpassHandles)
	}
	t.splitRenderPasses[rp.VulkanHandle()] = handles
	return handles
}

func (t *commandSplitter) rewriteGraphicsPipeline(ctx context.Context, graphicsPipeline VkPipeline, queueSubmit *VkQueueSubmit, out transform.Writer) VkPipeline {
	if gp, ok := t.fixedGraphicsPipelines[graphicsPipeline]; ok {
		return gp
	}
	s := out.State()
	st := GetState(s)
	a := s.Arena

	sb := st.newStateBuilder(ctx, newTransformerOutput(out))
	newGp := st.GraphicsPipelines().Get(graphicsPipeline).Clone(a, api.CloneContext{})
	newGp.SetRenderPass(st.RenderPasses().Get(t.currentRenderPass[t.thisSubpass][0]))
	newGp.SetSubpass(0)
	newGp.SetVulkanHandle(
		VkPipeline(newUnusedID(true, func(x uint64) bool {
			return st.GraphicsPipelines().Contains(VkPipeline(x)) ||
				st.ComputePipelines().Contains(VkPipeline(x))
		})))
	sb.createGraphicsPipeline(newGp)
	t.fixedGraphicsPipelines[graphicsPipeline] = newGp.VulkanHandle()
	return newGp.VulkanHandle()
}

func (t *commandSplitter) splitCommandBuffer(ctx context.Context, embedBuffer VkCommandBuffer, commandBuffer CommandBufferObjectʳ, queueSubmit *VkQueueSubmit, id api.SubCmdIdx, cuts []api.SubCmdIdx, out transform.Writer) VkCommandBuffer {
	s := out.State()
	st := GetState(s)
	a := s.Arena
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: a}

	for i := 0; i < commandBuffer.CommandReferences().Len(); i++ {
		splitAfterCommand := false
		replaceCommand := false
		newCuts := []api.SubCmdIdx{}
		for _, s := range cuts {
			if s[0] == uint64(i) {
				if len(s) == 1 {
					splitAfterCommand = true
					continue
				} else {
					newCuts = append(newCuts, s[1:])
				}
			}
		}

		cr := commandBuffer.CommandReferences().Get(uint32(i))
		extraArgs := make([]interface{}, 0)
		args := GetCommandArgs(ctx, cr, st)
		switch ar := args.(type) {
		case VkCmdBeginRenderPassArgsʳ:
			rp := ar.RenderPass()
			rpo := st.RenderPasses().Get(rp)
			t.currentRenderPass = t.splitRenderPass(ctx, rpo, out)
			t.thisSubpass = 0
			args = NewVkCmdBeginRenderPassArgsʳ(a, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
				t.currentRenderPass[t.thisSubpass][0], ar.Framebuffer(), ar.RenderArea(), ar.ClearValues(),
				ar.DeviceGroupBeginInfo())
			t.thisRenderPass = ar
		case VkCmdNextSubpassArgsʳ:
			args = NewVkCmdEndRenderPassArgsʳ(a)
			extraArgs = append(extraArgs,
				NewVkCmdBeginRenderPassArgsʳ(
					a, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
					t.currentRenderPass[t.thisSubpass][2], t.thisRenderPass.Framebuffer(), t.thisRenderPass.RenderArea(), t.thisRenderPass.ClearValues(),
					t.thisRenderPass.DeviceGroupBeginInfo()))
			extraArgs = append(extraArgs, NewVkCmdEndRenderPassArgsʳ(a))

			t.thisSubpass++
			extraArgs = append(extraArgs,
				NewVkCmdBeginRenderPassArgsʳ(
					a, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
					t.currentRenderPass[t.thisSubpass][0], t.thisRenderPass.Framebuffer(), t.thisRenderPass.RenderArea(), t.thisRenderPass.ClearValues(),
					t.thisRenderPass.DeviceGroupBeginInfo()))
		case VkCmdEndRenderPassArgsʳ:
			extraArgs = append(extraArgs,
				NewVkCmdBeginRenderPassArgsʳ(
					a, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
					t.currentRenderPass[t.thisSubpass][2], t.thisRenderPass.Framebuffer(), t.thisRenderPass.RenderArea(), t.thisRenderPass.ClearValues(),
					t.thisRenderPass.DeviceGroupBeginInfo()))
			extraArgs = append(extraArgs, NewVkCmdEndRenderPassArgsʳ(a))
			t.thisRenderPass = NilVkCmdBeginRenderPassArgsʳ
			t.thisSubpass = 0
		case VkCmdBindPipelineArgsʳ:
			if ar.PipelineBindPoint() == VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
				// Graphics pipeline, must be split (maybe)
				if st.RenderPasses().Get(t.thisRenderPass.RenderPass()).SubpassDescriptions().Len() > 1 {
					// If we have more than one renderpass, then we should replace
					newPipeline := t.rewriteGraphicsPipeline(ctx, ar.Pipeline(), queueSubmit, out)
					np := ar.Clone(a, api.CloneContext{})
					np.SetPipeline(newPipeline)
					args = np
				}
			}
		case VkCmdExecuteCommandsArgsʳ:
			replaceCommand = true
			for j := 0; j < ar.CommandBuffers().Len(); j++ {
				splitAfterExecute := false
				newSubCuts := []api.SubCmdIdx{}
				for _, s := range newCuts {
					if s[0] == uint64(j) {
						if len(s) == 1 {
							splitAfterExecute = true
							continue
						} else {
							newSubCuts = append(newSubCuts, s[1:])
						}
					}
				}

				cbo := st.CommandBuffers().Get(ar.CommandBuffers().Get(uint32(j)))
				t.splitCommandBuffer(ctx, embedBuffer, cbo, queueSubmit, append(id, uint64(i), uint64(j)), newSubCuts, out)
				if splitAfterExecute {
					t.WriteCommand(ctx, &InsertionCommand{
						embedBuffer,
						append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
						append(id, uint64(i), uint64(j)),
						queueSubmit,
					}, out)
				}
			}
		}
		if splitAfterCommand {
			// If we are inside a renderpass, then drop out for this call.
			// If we were not in a renderpass then we do not need to drop out
			// of it.
			if t.thisRenderPass != NilVkCmdBeginRenderPassArgsʳ {
				extraArgs = append(extraArgs, NewVkCmdEndRenderPassArgsʳ(a))
			}
			extraArgs = append(extraArgs, &InsertionCommand{
				embedBuffer,
				append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
				append(id, uint64(i)),
				queueSubmit,
			})
			// If we were inside a renderpass, then we have to get back
			// into a renderpass
			if t.thisRenderPass != NilVkCmdBeginRenderPassArgsʳ {
				extraArgs = append(extraArgs,
					NewVkCmdBeginRenderPassArgsʳ(
						a, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
						t.currentRenderPass[t.thisSubpass][1], t.thisRenderPass.Framebuffer(), t.thisRenderPass.RenderArea(), t.thisRenderPass.ClearValues(),
						t.thisRenderPass.DeviceGroupBeginInfo()))
			}
		}
		if !replaceCommand {
			cleanup, cmd, err := AddCommand(ctx, cb, embedBuffer, s, s, args)
			if err != nil {
				panic(fmt.Errorf("Invalid command-buffer detected %+v", err))
			}
			t.WriteCommand(ctx, cmd, out)
			cleanup()
		}
		for _, ea := range extraArgs {
			if ins, ok := ea.(api.Cmd); ok {
				t.WriteCommand(ctx, ins, out)
			} else {
				cleanup, cmd, err := AddCommand(ctx, cb, embedBuffer, s, s, ea)
				if err != nil {
					panic(fmt.Errorf("Invalid command-buffer detected %+v", err))
				}
				t.WriteCommand(ctx, cmd, out)
				cleanup()
			}
		}
	}

	return embedBuffer
}

func (t *commandSplitter) splitSubmit(ctx context.Context, submit VkSubmitInfo, idx api.SubCmdIdx, cuts []api.SubCmdIdx, queueSubmit *VkQueueSubmit, out transform.Writer) (VkSubmitInfo, error) {
	s := out.State()
	l := s.MemoryLayout
	st := GetState(s)
	a := s.Arena
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: a}

	newSubmit := MakeVkSubmitInfo(a)
	newSubmit.SetSType(submit.SType())
	newSubmit.SetPNext(submit.PNext())
	newSubmit.SetWaitSemaphoreCount(submit.WaitSemaphoreCount())
	newSubmit.SetPWaitSemaphores(submit.PWaitSemaphores())
	newSubmit.SetPWaitDstStageMask(submit.PWaitDstStageMask())
	newSubmit.SetCommandBufferCount(submit.CommandBufferCount())
	// pCommandBuffers
	commandBuffers := submit.PCommandBuffers().Slice(0, uint64(submit.CommandBufferCount()), l).MustRead(ctx, queueSubmit, s, nil)
	newCommandBuffers := make([]VkCommandBuffer, 0)
	for i := range commandBuffers {
		splitAfterCommandBuffer := false
		newCuts := []api.SubCmdIdx{}
		for _, s := range cuts {
			if s[0] == uint64(i) {
				if len(s) == 1 {
					splitAfterCommandBuffer = true
					continue
				} else {
					newCuts = append(newCuts, s[1:])
				}
			}
		}
		if len(newCuts) > 0 {
			cbuff := commandBuffers[i]
			cbo := st.CommandBuffers().Get(cbuff)
			commandBuffer, err := t.getStartedCommandBuffer(ctx, queueSubmit, out)
			if err != nil {
				return VkSubmitInfo{}, err
			}
			newCommandBuffers = append(newCommandBuffers, t.splitCommandBuffer(ctx, commandBuffer, cbo, queueSubmit, append(idx, uint64(i)), newCuts, out))
			t.WriteCommand(ctx,
				cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS), out)
		} else {
			newCommandBuffers = append(newCommandBuffers, commandBuffers[i])
		}
		if splitAfterCommandBuffer {
			commandBuffer, err := t.getStartedCommandBuffer(ctx, queueSubmit, out)
			if err != nil {
				return VkSubmitInfo{}, err
			}
			if err := out.MutateAndWrite(ctx, api.CmdNoID, &InsertionCommand{
				commandBuffer,
				append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
				append(idx, uint64(i)),
				queueSubmit,
			}); err != nil {
				return VkSubmitInfo{}, err
			}
			if err := t.WriteCommand(ctx,
				cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS), out); err != nil {
				return VkSubmitInfo{}, err
			}
			newCommandBuffers = append(newCommandBuffers, commandBuffer)
		}
	}
	t.pendingCommandBuffers = append(t.pendingCommandBuffers, newCommandBuffers...)
	newCbs := t.MustAllocReadDataForSubmit(ctx, s, newCommandBuffers)
	newSubmit.SetPCommandBuffers(NewVkCommandBufferᶜᵖ(newCbs.Ptr()))
	newSubmit.SetCommandBufferCount(uint32(len(newCommandBuffers)))
	newSubmit.SetSignalSemaphoreCount(submit.SignalSemaphoreCount())
	newSubmit.SetPSignalSemaphores(submit.PSignalSemaphores())
	return newSubmit, nil
}

func (t *commandSplitter) splitAfterSubmit(ctx context.Context, id api.SubCmdIdx, queueSubmit *VkQueueSubmit, out transform.Writer) (VkSubmitInfo, error) {
	s := out.State()
	a := s.Arena
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: a}

	commandBuffer, err := t.getStartedCommandBuffer(ctx, queueSubmit, out)
	if err != nil {
		return VkSubmitInfo{}, err
	}
	t.pendingCommandBuffers = append(t.pendingCommandBuffers, commandBuffer)
	if err := out.MutateAndWrite(ctx, api.CmdNoID, &InsertionCommand{
		commandBuffer,
		append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
		id,
		queueSubmit,
	}); err != nil {
		return VkSubmitInfo{}, err
	}
	if err := t.WriteCommand(ctx,
		cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS), out); err != nil {
		return VkSubmitInfo{}, err
	}

	info := NewVkSubmitInfo(a,
		VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                     // pNext
		0,                                             // waitSemaphoreCount,
		NewVkSemaphoreᶜᵖ(memory.Nullptr),              // pWaitSemaphores
		NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr), // pWaitDstStageMask
		1, // commandBufferCount
		NewVkCommandBufferᶜᵖ(t.MustAllocReadDataForSubmit(ctx, s, commandBuffer).Ptr()),
		0,                                // signalSemaphoreCount
		NewVkSemaphoreᶜᵖ(memory.Nullptr), // pSignalSemaphores
	)

	return info, nil
}

func (t *commandSplitter) rewriteQueueSubmit(ctx context.Context, id api.CmdID, cuts []api.SubCmdIdx, queueSubmit *VkQueueSubmit, out transform.Writer) (*VkQueueSubmit, error) {
	s := out.State()
	l := s.MemoryLayout
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: s.Arena}
	queueSubmit.Extras().Observations().ApplyReads(s.Memory.ApplicationPool())
	var err error
	submitInfos := queueSubmit.PSubmits().Slice(0, uint64(queueSubmit.SubmitCount()), l).MustRead(ctx, queueSubmit, s, nil)
	newSubmitInfos := []VkSubmitInfo{}

	newSubmit := cb.VkQueueSubmit(queueSubmit.Queue(), queueSubmit.SubmitCount(), queueSubmit.PSubmits(), queueSubmit.Fence(), queueSubmit.Result())
	newSubmit.Extras().MustClone(queueSubmit.Extras().All()...)

	for i := 0; i < len(submitInfos); i++ {
		subIdx := api.SubCmdIdx{uint64(id), uint64(i)}
		newCuts := []api.SubCmdIdx{}
		addAfterSubmit := false
		for _, s := range cuts {
			if s[0] == uint64(i) {
				if len(s) == 1 {
					addAfterSubmit = true
				} else {
					newCuts = append(newCuts, s[1:])
				}
			}
		}
		newSubmitInfo := submitInfos[i]
		if len(newCuts) != 0 {
			newSubmitInfo, err = t.splitSubmit(ctx, submitInfos[i], subIdx, newCuts, queueSubmit, out)
			if err != nil {
				return nil, err
			}
		} else {
			commandBuffers := submitInfos[i].PCommandBuffers().Slice(0, uint64(submitInfos[i].CommandBufferCount()), l).MustRead(ctx, queueSubmit, s, nil)
			t.pendingCommandBuffers = append(t.pendingCommandBuffers, commandBuffers...)
		}
		newSubmitInfos = append(newSubmitInfos, newSubmitInfo)
		if addAfterSubmit {
			s, err := t.splitAfterSubmit(ctx, subIdx, queueSubmit, out)
			if err != nil {
				return nil, err
			}
			newSubmitInfos = append(newSubmitInfos, s)
		}
	}
	newSubmit.SetSubmitCount(uint32(len(newSubmitInfos)))
	newSubmit.SetPSubmits(NewVkSubmitInfoᶜᵖ(t.MustAllocReadDataForSubmit(ctx, s, newSubmitInfos).Ptr()))

	for x := range t.readMemoriesForSubmit {
		newSubmit.AddRead(t.readMemoriesForSubmit[x].Data())
	}
	t.readMemoriesForSubmit = []*api.AllocResult{}
	return newSubmit, nil
}

func (t *commandSplitter) Transform(ctx context.Context, id api.CmdID, cmd api.Cmd, out transform.Writer) error {
	inRange := false
	var topCut api.SubCmdIdx
	cuts := []api.SubCmdIdx{}
	thisID := api.SubCmdIdx{uint64(id)}
	for _, r := range t.requestsSubIndex {
		if thisID.Contains(r) {
			inRange = true
			if thisID.Equals(r) {
				topCut = r
			} else {
				cuts = append(cuts, r[1:])
			}
		}
	}

	if !inRange {
		return out.MutateAndWrite(ctx, id, cmd)
	}

	if len(cuts) == 0 {
		if cmd.CmdFlags().IsEndOfFrame() {
			if err := out.MutateAndWrite(ctx, id, &InsertionCommand{
				VkCommandBuffer(0),
				append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
				topCut,
				cmd,
			}); err != nil {
				return err
			}
			if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
				return err
			}
		} else {
			if err := out.MutateAndWrite(ctx, id, cmd); err != nil {
				return err
			}
			if err := out.MutateAndWrite(ctx, id, &InsertionCommand{
				VkCommandBuffer(0),
				append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
				topCut,
				cmd,
			}); err != nil {
				return err
			}
		}

		return nil
	}

	// Actually do the cutting here:
	queueSubmit, ok := cmd.(*VkQueueSubmit)
	// If this is not a queue submit it has no business having
	// subcommands.
	if !ok {
		return out.MutateAndWrite(ctx, id, cmd)
	}
	thisCmd, err := t.rewriteQueueSubmit(ctx, id, cuts, queueSubmit, out)
	if err != nil {
		return err
	}
	if err := out.MutateAndWrite(ctx, id, thisCmd); err != nil {
		return err
	}
	if len(topCut) == 0 {
		return nil
	}
	if err := out.MutateAndWrite(ctx, id, &InsertionCommand{
		VkCommandBuffer(0),
		append([]VkCommandBuffer{}, t.pendingCommandBuffers...),
		topCut,
		cmd,
	}); err != nil {
		return err
	}
	t.pendingCommandBuffers = []VkCommandBuffer{}
	return nil
}

func (t *commandSplitter) Flush(ctx context.Context, out transform.Writer) error { return nil }
func (t *commandSplitter) PreLoop(ctx context.Context, output transform.Writer)  {}
func (t *commandSplitter) PostLoop(ctx context.Context, output transform.Writer) {}
func (t *commandSplitter) BuffersCommands() bool                                 { return false }
