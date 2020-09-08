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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/transform"
	"github.com/google/gapid/gapis/memory"
)

func (splitTransform *commandSplitter) MustAllocReadDataForSubmit(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := splitTransform.allocations.AllocDataOrPanic(ctx, v...)
	splitTransform.readMemoriesForSubmit = append(splitTransform.readMemoriesForSubmit, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (splitTransform *commandSplitter) MustAllocReadDataForCmd(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := splitTransform.allocations.AllocDataOrPanic(ctx, v...)
	splitTransform.readMemoriesForCmd = append(splitTransform.readMemoriesForCmd, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (splitTransform *commandSplitter) MustAllocWriteDataForCmd(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := splitTransform.allocations.AllocDataOrPanic(ctx, v...)
	splitTransform.writeMemoriesForCmd = append(splitTransform.writeMemoriesForCmd, &allocateResult)
	return allocateResult
}

func (splitter *commandSplitter) writeCommand(cmd api.Cmd) error {
	return splitter.stateMutator([]api.Cmd{cmd})
}

func (splitter *commandSplitter) writeCommands(cmds []api.Cmd) error {
	for _, cmd := range cmds {
		if err := splitter.writeCommand(cmd); err != nil {
			return err
		}
	}

	return nil
}

func (s *commandSplitter) observeAndWriteCommand(cmd api.Cmd) error {
	for i := range s.readMemoriesForCmd {
		cmd.Extras().GetOrAppendObservations().AddRead(s.readMemoriesForCmd[i].Data())
	}
	for i := range s.writeMemoriesForCmd {
		cmd.Extras().GetOrAppendObservations().AddWrite(s.writeMemoriesForCmd[i].Data())
	}
	s.readMemoriesForCmd = []*api.AllocResult{}
	s.writeMemoriesForCmd = []*api.AllocResult{}

	return s.writeCommand(cmd)
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
	stateMutator          transform.StateMutator
	allocations           *allocationTracker
}

func NewCommandSplitter(ctx context.Context) *commandSplitter {
	return &commandSplitter{
		lastRequest:            api.SubCmdIdx{},
		requestsSubIndex:       make([]api.SubCmdIdx, 0),
		readMemoriesForSubmit:  make([]*api.AllocResult, 0),
		readMemoriesForCmd:     make([]*api.AllocResult, 0),
		writeMemoriesForCmd:    make([]*api.AllocResult, 0),
		pool:                   0,
		thisRenderPass:         NilVkCmdBeginRenderPassArgsʳ,
		currentRenderPass:      make([][3]VkRenderPass, 0),
		thisSubpass:            0,
		splitRenderPasses:      make(map[VkRenderPass][][3]VkRenderPass),
		fixedGraphicsPipelines: make(map[VkPipeline]VkPipeline),
		pendingCommandBuffers:  make([]VkCommandBuffer, 0),
		stateMutator:           nil,
	}
}

func (splitTransform *commandSplitter) RequiresAccurateState() bool {
	return false
}

func (splitTransform *commandSplitter) RequiresInnerStateMutation() bool {
	return true
}

func (splitTransform *commandSplitter) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	splitTransform.stateMutator = mutator
}

func (splitTransform *commandSplitter) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	splitTransform.allocations = NewAllocationTracker(inputState)
	return nil
}

func (splitTransform *commandSplitter) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	return nil, nil
}

func (splitTransform *commandSplitter) ClearTransformResources(ctx context.Context) {
	splitTransform.allocations.FreeAllocations()
}

func (splitTransform *commandSplitter) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if len(inputCommands) == 0 {
		return inputCommands, nil
	}

	inRange := false
	var topCut api.SubCmdIdx
	cuts := []api.SubCmdIdx{}
	thisID := api.SubCmdIdx{uint64(id.GetID())}
	for _, r := range splitTransform.requestsSubIndex {
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
		return inputCommands, nil
	}

	if len(cuts) == 0 {
		if err := splitTransform.appendInsertionCommand(ctx, inputCommands, topCut); err != nil {
			log.E(ctx, "Failed during appending insertion command : %v", err)
			return []api.Cmd{}, err
		}

		return []api.Cmd{}, nil
	}

	queueSubmitProcessed := false
	for _, cmd := range inputCommands {
		if queueSubmitCmd, ok := cmd.(*VkQueueSubmit); ok {
			if queueSubmitProcessed {
				panic("We should not have more than one vkQueueSubmit for a single command")
			}

			queueSubmitProcessed = true
			if err := splitTransform.modifyVkQueueSubmit(ctx, id.GetID(), queueSubmitCmd, inputState, topCut, cuts); err != nil {
				log.E(ctx, "Failed during modifying VkQueueSubmit : %v", err)
				return nil, err
			}
		} else {
			if err := splitTransform.writeCommand(cmd); err != nil {
				log.E(ctx, "Failed during processing input commands : %v", err)
				return nil, err
			}
		}
	}

	return nil, nil
}

func (splitTransform *commandSplitter) appendInsertionCommand(ctx context.Context, inputCommands []api.Cmd, topCut api.SubCmdIdx) error {
	isEndOfFrame := false
	endOfFrameCmdID := 0
	for i, cmd := range inputCommands {
		if cmd.CmdFlags().IsEndOfFrame() {
			isEndOfFrame = true
			endOfFrameCmdID = i
			break
		}
	}

	insertionCommand := &InsertionCommand{
		VkCommandBuffer(0),
		append([]VkCommandBuffer{}, splitTransform.pendingCommandBuffers...),
		topCut,
		inputCommands[endOfFrameCmdID],
	}

	if isEndOfFrame {
		// We want to add insertion command before the vkQueuePresentKHR so that
		// the images are still valid.
		if err := splitTransform.writeCommand(insertionCommand); err != nil {
			log.E(ctx, "Failed during writing insertion command : %v", err)
			return err
		}

		if err := splitTransform.writeCommands(inputCommands); err != nil {
			log.E(ctx, "Failed during processing input commands : %v", err)
			return err
		}
	} else {
		if err := splitTransform.writeCommands(inputCommands); err != nil {
			log.E(ctx, "Failed during processing input commands : %v", err)
			return err
		}
		if err := splitTransform.writeCommand(insertionCommand); err != nil {
			log.E(ctx, "Failed during writing insertion command : %v", err)
			return err
		}
	}

	return nil
}

func (splitTransform *commandSplitter) modifyVkQueueSubmit(
	ctx context.Context,
	id api.CmdID,
	cmd *VkQueueSubmit,
	inputState *api.GlobalState,
	topCut api.SubCmdIdx,
	cuts []api.SubCmdIdx) error {
	newSubmit, err := splitTransform.rewriteQueueSubmit(ctx, id, cuts, cmd, inputState)
	if err != nil {
		log.E(ctx, "Failed during rewriting VkQueueSubmit : %v", err)
		return err
	}

	if err = splitTransform.writeCommand(newSubmit); err != nil {
		log.E(ctx, "Failed during writing VkQueueSubmit : %v", err)
		return err
	}

	if len(topCut) == 0 {
		return nil
	}

	if err := splitTransform.writeCommand(&InsertionCommand{
		VkCommandBuffer(0),
		append([]VkCommandBuffer{}, splitTransform.pendingCommandBuffers...),
		topCut,
		cmd,
	}); err != nil {
		log.E(ctx, "Failed during inserting and Insertion Command after VkQueueSubmit : %v", err)
		return err
	}

	splitTransform.pendingCommandBuffers = []VkCommandBuffer{}
	return nil
}

func (splitTransform *commandSplitter) rewriteQueueSubmit(ctx context.Context, id api.CmdID, cuts []api.SubCmdIdx, queueSubmit *VkQueueSubmit, inputState *api.GlobalState) (*VkQueueSubmit, error) {
	layout := inputState.MemoryLayout
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: inputState.Arena}
	queueSubmit.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	submitInfos := queueSubmit.PSubmits().Slice(0, uint64(queueSubmit.SubmitCount()), layout).MustRead(ctx, queueSubmit, inputState, nil)
	newSubmitInfos := []VkSubmitInfo{}

	newSubmit := cb.VkQueueSubmit(queueSubmit.Queue(), queueSubmit.SubmitCount(), queueSubmit.PSubmits(), queueSubmit.Fence(), queueSubmit.Result())
	newSubmit.Extras().MustClone(queueSubmit.Extras().All()...)

	var err error
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
			newSubmitInfo, err = splitTransform.splitSubmit(ctx, submitInfos[i], subIdx, newCuts, queueSubmit, inputState)
			if err != nil {
				log.E(ctx, "Failed during splitting submit : %v", err)
				return nil, err
			}
		} else {
			commandBuffers := submitInfos[i].PCommandBuffers().Slice(0, uint64(submitInfos[i].CommandBufferCount()), layout).MustRead(ctx, queueSubmit, inputState, nil)
			splitTransform.pendingCommandBuffers = append(splitTransform.pendingCommandBuffers, commandBuffers...)
		}
		newSubmitInfos = append(newSubmitInfos, newSubmitInfo)
		if addAfterSubmit {
			submitInfo, err := splitTransform.splitAfterSubmit(ctx, subIdx, queueSubmit, inputState)
			if err != nil {
				log.E(ctx, "Failed during splitting after Submit : %v", err)
				return nil, err
			}
			newSubmitInfos = append(newSubmitInfos, submitInfo)
		}
	}
	newSubmit.SetSubmitCount(uint32(len(newSubmitInfos)))
	newSubmit.SetPSubmits(NewVkSubmitInfoᶜᵖ(splitTransform.MustAllocReadDataForSubmit(ctx, inputState, newSubmitInfos).Ptr()))

	for x := range splitTransform.readMemoriesForSubmit {
		newSubmit.AddRead(splitTransform.readMemoriesForSubmit[x].Data())
	}
	splitTransform.readMemoriesForSubmit = []*api.AllocResult{}
	return newSubmit, nil
}

func (splitTransform *commandSplitter) splitSubmit(ctx context.Context, submit VkSubmitInfo, idx api.SubCmdIdx, cuts []api.SubCmdIdx, queueSubmit *VkQueueSubmit, inputState *api.GlobalState) (VkSubmitInfo, error) {
	newSubmitInfo := MakeVkSubmitInfo(inputState.Arena)
	newSubmitInfo.SetSType(submit.SType())
	newSubmitInfo.SetPNext(submit.PNext())
	newSubmitInfo.SetWaitSemaphoreCount(submit.WaitSemaphoreCount())
	newSubmitInfo.SetPWaitSemaphores(submit.PWaitSemaphores())
	newSubmitInfo.SetPWaitDstStageMask(submit.PWaitDstStageMask())
	newSubmitInfo.SetCommandBufferCount(submit.CommandBufferCount())

	layout := inputState.MemoryLayout
	// pCommandBuffers
	commandBuffers := submit.PCommandBuffers().Slice(0, uint64(submit.CommandBufferCount()), layout).MustRead(ctx, queueSubmit, inputState, nil)
	newCommandBuffers := make([]VkCommandBuffer, 0)

	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: inputState.Arena}

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
			cbo := GetState(inputState).CommandBuffers().Get(cbuff)
			commandBuffer, err := splitTransform.getStartedCommandBuffer(ctx, queueSubmit, inputState)
			if err != nil {
				log.E(ctx, "Failed during getting started command buffer : %v", err)
				return VkSubmitInfo{}, err
			}

			splitCommandBuffers, err := splitTransform.splitCommandBuffer(ctx, commandBuffer, cbo, queueSubmit, append(idx, uint64(i)), newCuts, inputState)
			if err != nil {
				log.E(ctx, "Failed during splitting command buffer : %v", err)
				return VkSubmitInfo{}, err
			}
			newCommandBuffers = append(newCommandBuffers, splitCommandBuffers)
			if err := splitTransform.observeAndWriteCommand(cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS)); err != nil {
				log.E(ctx, "Failed during writing EndCommandBuffer : %v", err)
				return VkSubmitInfo{}, err
			}
		} else {
			newCommandBuffers = append(newCommandBuffers, commandBuffers[i])
		}
		if splitAfterCommandBuffer {
			commandBuffer, err := splitTransform.getStartedCommandBuffer(ctx, queueSubmit, inputState)
			if err != nil {
				log.E(ctx, "Failed during getting started command buffer : %v", err)
				return VkSubmitInfo{}, err
			}

			if err := splitTransform.writeCommand(&InsertionCommand{
				commandBuffer,
				append([]VkCommandBuffer{}, splitTransform.pendingCommandBuffers...),
				append(idx, uint64(i)),
				queueSubmit,
			}); err != nil {
				log.E(ctx, "Failed during insterting and instertion command : %v", err)
				return VkSubmitInfo{}, err
			}

			if err := splitTransform.observeAndWriteCommand(cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS)); err != nil {
				log.E(ctx, "Failed during writing EndCommandBuffer : %v", err)
				return VkSubmitInfo{}, err
			}
			newCommandBuffers = append(newCommandBuffers, commandBuffer)
		}
	}
	splitTransform.pendingCommandBuffers = append(splitTransform.pendingCommandBuffers, newCommandBuffers...)
	newCbs := splitTransform.MustAllocReadDataForSubmit(ctx, inputState, newCommandBuffers)
	newSubmitInfo.SetPCommandBuffers(NewVkCommandBufferᶜᵖ(newCbs.Ptr()))
	newSubmitInfo.SetCommandBufferCount(uint32(len(newCommandBuffers)))
	newSubmitInfo.SetSignalSemaphoreCount(submit.SignalSemaphoreCount())
	newSubmitInfo.SetPSignalSemaphores(submit.PSignalSemaphores())

	return newSubmitInfo, nil
}

func (splitTransform *commandSplitter) splitAfterSubmit(ctx context.Context, id api.SubCmdIdx, queueSubmit *VkQueueSubmit, inputState *api.GlobalState) (VkSubmitInfo, error) {
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: inputState.Arena}
	commandBuffer, err := splitTransform.getStartedCommandBuffer(ctx, queueSubmit, inputState)
	if err != nil {
		log.E(ctx, "Failed during getting started command buffer : %v", err)
		return VkSubmitInfo{}, err
	}

	splitTransform.pendingCommandBuffers = append(splitTransform.pendingCommandBuffers, commandBuffer)

	if err = splitTransform.writeCommand(&InsertionCommand{
		commandBuffer,
		append([]VkCommandBuffer{}, splitTransform.pendingCommandBuffers...),
		id,
		queueSubmit,
	}); err != nil {
		log.E(ctx, "Failed during insterting and instertion command : %v", err)
		return VkSubmitInfo{}, err
	}

	if err = splitTransform.observeAndWriteCommand(cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS)); err != nil {
		log.E(ctx, "Failed during writing end command buffer : %v", err)
		return VkSubmitInfo{}, err
	}

	info := NewVkSubmitInfo(inputState.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                     // pNext
		0,                                             // waitSemaphoreCount,
		NewVkSemaphoreᶜᵖ(memory.Nullptr),              // pWaitSemaphores
		NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr), // pWaitDstStageMask
		1, // commandBufferCount
		NewVkCommandBufferᶜᵖ(splitTransform.MustAllocReadDataForSubmit(ctx, inputState, commandBuffer).Ptr()),
		0,                                // signalSemaphoreCount
		NewVkSemaphoreᶜᵖ(memory.Nullptr), // pSignalSemaphores
	)

	return info, nil
}

const VK_ATTACHMENT_UNUSED = uint32(0xFFFFFFFF)

func (splitTransform *commandSplitter) splitRenderPass(ctx context.Context, rp RenderPassObjectʳ, inputState *api.GlobalState) ([][3]VkRenderPass, error) {
	st := GetState(inputState)

	if rp, ok := splitTransform.splitRenderPasses[rp.VulkanHandle()]; ok {
		return rp, nil
	}

	handles := make([][3]VkRenderPass, 0)
	currentLayouts := make(map[uint32]VkImageLayout)
	for i := uint32(0); i < uint32(rp.AttachmentDescriptions().Len()); i++ {
		currentLayouts[i] = rp.AttachmentDescriptions().Get(i).InitialLayout()
	}

	tempTransformWriter := newCommandsplitTransformWriter(inputState, splitTransform)
	sb := st.newStateBuilder(ctx, newTransformerOutput(tempTransformWriter))

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
			rp1 := rp.Clone(inputState.Arena, api.CloneContext{})
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
			rp2 := rp.Clone(inputState.Arena, api.CloneContext{})
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
			rp3 := rp.Clone(inputState.Arena, api.CloneContext{})
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

	splitTransform.splitRenderPasses[rp.VulkanHandle()] = handles
	return handles, nil
}

func (splitTransform *commandSplitter) rewriteGraphicsPipeline(ctx context.Context, graphicsPipeline VkPipeline, queueSubmit *VkQueueSubmit, inputState *api.GlobalState) (VkPipeline, error) {
	if gp, ok := splitTransform.fixedGraphicsPipelines[graphicsPipeline]; ok {
		return gp, nil
	}

	st := GetState(inputState)
	tempTransformWriter := newCommandsplitTransformWriter(inputState, splitTransform)
	sb := st.newStateBuilder(ctx, newTransformerOutput(tempTransformWriter))
	newGp := st.GraphicsPipelines().Get(graphicsPipeline).Clone(inputState.Arena, api.CloneContext{})
	newGp.SetRenderPass(st.RenderPasses().Get(splitTransform.currentRenderPass[splitTransform.thisSubpass][0]))
	newGp.SetSubpass(0)
	newGp.SetVulkanHandle(
		VkPipeline(newUnusedID(true, func(x uint64) bool {
			return st.GraphicsPipelines().Contains(VkPipeline(x)) ||
				st.ComputePipelines().Contains(VkPipeline(x))
		})))
	sb.createGraphicsPipeline(newGp)
	splitTransform.fixedGraphicsPipelines[graphicsPipeline] = newGp.VulkanHandle()
	return newGp.VulkanHandle(), nil
}

func (splitTransform *commandSplitter) splitCommandBuffer(ctx context.Context, embedBuffer VkCommandBuffer, commandBuffer CommandBufferObjectʳ, queueSubmit *VkQueueSubmit, id api.SubCmdIdx, cuts []api.SubCmdIdx, inputState *api.GlobalState) (VkCommandBuffer, error) {
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: inputState.Arena}
	st := GetState(inputState)

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
			var err error
			if splitTransform.currentRenderPass, err = splitTransform.splitRenderPass(ctx, rpo, inputState); err != nil {
				log.E(ctx, "Failed during splitting render pass : %v", err)
				return VkCommandBuffer(0), err
			}
			splitTransform.thisSubpass = 0
			args = NewVkCmdBeginRenderPassArgsʳ(inputState.Arena, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
				splitTransform.currentRenderPass[splitTransform.thisSubpass][0], ar.Framebuffer(), ar.RenderArea(), ar.ClearValues(),
				ar.DeviceGroupBeginInfo())
			splitTransform.thisRenderPass = ar
		case VkCmdNextSubpassArgsʳ:
			args = NewVkCmdEndRenderPassArgsʳ(inputState.Arena)
			extraArgs = append(extraArgs,
				NewVkCmdBeginRenderPassArgsʳ(
					inputState.Arena, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
					splitTransform.currentRenderPass[splitTransform.thisSubpass][2], splitTransform.thisRenderPass.Framebuffer(), splitTransform.thisRenderPass.RenderArea(), splitTransform.thisRenderPass.ClearValues(),
					splitTransform.thisRenderPass.DeviceGroupBeginInfo()))
			extraArgs = append(extraArgs, NewVkCmdEndRenderPassArgsʳ(inputState.Arena))

			splitTransform.thisSubpass++
			extraArgs = append(extraArgs,
				NewVkCmdBeginRenderPassArgsʳ(
					inputState.Arena, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
					splitTransform.currentRenderPass[splitTransform.thisSubpass][0], splitTransform.thisRenderPass.Framebuffer(), splitTransform.thisRenderPass.RenderArea(), splitTransform.thisRenderPass.ClearValues(),
					splitTransform.thisRenderPass.DeviceGroupBeginInfo()))
		case VkCmdEndRenderPassArgsʳ:
			extraArgs = append(extraArgs,
				NewVkCmdBeginRenderPassArgsʳ(
					inputState.Arena, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
					splitTransform.currentRenderPass[splitTransform.thisSubpass][2], splitTransform.thisRenderPass.Framebuffer(), splitTransform.thisRenderPass.RenderArea(), splitTransform.thisRenderPass.ClearValues(),
					splitTransform.thisRenderPass.DeviceGroupBeginInfo()))
			extraArgs = append(extraArgs, NewVkCmdEndRenderPassArgsʳ(inputState.Arena))
			splitTransform.thisRenderPass = NilVkCmdBeginRenderPassArgsʳ
			splitTransform.thisSubpass = 0
		case VkCmdBindPipelineArgsʳ:
			if ar.PipelineBindPoint() == VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
				// Graphics pipeline, must be split (maybe)
				if st.RenderPasses().Get(splitTransform.thisRenderPass.RenderPass()).SubpassDescriptions().Len() > 1 {
					// If we have more than one renderpass, then we should replace
					newPipeline, err := splitTransform.rewriteGraphicsPipeline(ctx, ar.Pipeline(), queueSubmit, inputState)
					if err != nil {
						log.E(ctx, "Failed during rewriting graphics pipeline : %v", err)
						return VkCommandBuffer(0), err
					}
					np := ar.Clone(inputState.Arena, api.CloneContext{})
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

				if _, err := splitTransform.splitCommandBuffer(ctx, embedBuffer, cbo, queueSubmit, append(id, uint64(i), uint64(j)), newSubCuts, inputState); err != nil {
					log.E(ctx, "Failed during splitting command buffer : %v", err)
					return VkCommandBuffer(0), err
				}

				if splitAfterExecute {
					insertionCmd := &InsertionCommand{
						embedBuffer,
						append([]VkCommandBuffer{}, splitTransform.pendingCommandBuffers...),
						append(id, uint64(i), uint64(j)),
						queueSubmit,
					}
					if err := splitTransform.observeAndWriteCommand(insertionCmd); err != nil {
						log.E(ctx, "Failed during inserting insertion command : %v", err)
						return VkCommandBuffer(0), err
					}
				}
			}
		}
		if splitAfterCommand {
			// If we are inside a renderpass, then drop out for this call.
			// If we were not in a renderpass then we do not need to drop out
			// of it.
			if splitTransform.thisRenderPass != NilVkCmdBeginRenderPassArgsʳ {
				extraArgs = append(extraArgs, NewVkCmdEndRenderPassArgsʳ(inputState.Arena))
			}
			extraArgs = append(extraArgs, &InsertionCommand{
				embedBuffer,
				append([]VkCommandBuffer{}, splitTransform.pendingCommandBuffers...),
				append(id, uint64(i)),
				queueSubmit,
			})
			// If we were inside a renderpass, then we have to get back
			// into a renderpass
			if splitTransform.thisRenderPass != NilVkCmdBeginRenderPassArgsʳ {
				extraArgs = append(extraArgs,
					NewVkCmdBeginRenderPassArgsʳ(
						inputState.Arena, VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
						splitTransform.currentRenderPass[splitTransform.thisSubpass][1], splitTransform.thisRenderPass.Framebuffer(), splitTransform.thisRenderPass.RenderArea(), splitTransform.thisRenderPass.ClearValues(),
						splitTransform.thisRenderPass.DeviceGroupBeginInfo()))
			}
		}
		if !replaceCommand {
			cleanup, cmd, err := AddCommand(ctx, cb, embedBuffer, inputState, inputState, args)
			if err != nil {
				panic(fmt.Errorf("Invalid command-buffer detected %+v", err))
			}
			if err := splitTransform.observeAndWriteCommand(cmd); err != nil {
				log.E(ctx, "Failed during adding command : [%v]%v", cmd, err)
				return VkCommandBuffer(0), err
			}
			cleanup()
		}
		for _, ea := range extraArgs {
			if ins, ok := ea.(api.Cmd); ok {
				if err := splitTransform.observeAndWriteCommand(ins); err != nil {
					log.E(ctx, "Failed during inserting insertion command : %v", err)
					return VkCommandBuffer(0), err
				}
			} else {
				cleanup, cmd, err := AddCommand(ctx, cb, embedBuffer, inputState, inputState, ea)
				if err != nil {
					panic(fmt.Errorf("Invalid command-buffer detected %+v", err))
				}
				if err := splitTransform.observeAndWriteCommand(cmd); err != nil {
					log.E(ctx, "Failed during adding command : [%v]%v", cmd, err)
					return VkCommandBuffer(0), err
				}
				cleanup()
			}
		}
	}

	return embedBuffer, nil
}

func (splitTransform *commandSplitter) getStartedCommandBuffer(ctx context.Context, queueSubmit *VkQueueSubmit, inputState *api.GlobalState) (VkCommandBuffer, error) {
	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: inputState.Arena}
	queue := GetState(inputState).Queues().Get(queueSubmit.Queue())

	commandPoolID, err := splitTransform.getCommandPool(ctx, queueSubmit, inputState)
	if err != nil {
		log.E(ctx, "Failed during getting command pool : %v", err)
		return VkCommandBuffer(0), err
	}

	commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(inputState.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		commandPoolID,                                                  // commandPool
		VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY,           // level
		1, // commandBufferCount
	)
	commandBufferID := VkCommandBuffer(newUnusedID(true, func(x uint64) bool {
		return GetState(inputState).CommandBuffers().Contains(VkCommandBuffer(x))
	}))

	allocateCmd := cb.VkAllocateCommandBuffers(
		queue.Device(),
		splitTransform.MustAllocReadDataForCmd(ctx, inputState, commandBufferAllocateInfo).Ptr(),
		splitTransform.MustAllocWriteDataForCmd(ctx, inputState, commandBufferID).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if err = splitTransform.observeAndWriteCommand(allocateCmd); err != nil {
		log.E(ctx, "Failed during allocating command buffer : %v", err)
		return VkCommandBuffer(0), err
	}

	beginInfo := NewVkCommandBufferBeginInfo(inputState.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                   // pNext
		0,                                                           // flags
		NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr), // pInheritanceInfo
	)
	beginCommandBufferCmd := cb.VkBeginCommandBuffer(
		commandBufferID,
		splitTransform.MustAllocReadDataForCmd(ctx, inputState, beginInfo).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if err = splitTransform.observeAndWriteCommand(beginCommandBufferCmd); err != nil {
		log.E(ctx, "Failed during begin command buffer : %v", err)
		return VkCommandBuffer(0), err
	}
	return commandBufferID, nil
}

func (splitTransform *commandSplitter) getCommandPool(ctx context.Context, queueSubmit *VkQueueSubmit, inputState *api.GlobalState) (VkCommandPool, error) {
	if splitTransform.pool != 0 {
		return splitTransform.pool, nil
	}

	cb := CommandBuilder{Thread: queueSubmit.Thread(), Arena: inputState.Arena}
	queue := GetState(inputState).Queues().Get(queueSubmit.Queue())

	splitTransform.pool = VkCommandPool(newUnusedID(false, func(x uint64) bool {
		return GetState(inputState).CommandPools().Contains(VkCommandPool(x))
	}))

	poolCreateInfo := NewVkCommandPoolCreateInfo(inputState.Arena,
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
		VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
		queue.Family(), // queueFamilyIndex
	)

	newCmd := cb.VkCreateCommandPool(
		queue.Device(),
		splitTransform.MustAllocReadDataForCmd(ctx, inputState, poolCreateInfo).Ptr(),
		memory.Nullptr,
		splitTransform.MustAllocWriteDataForCmd(ctx, inputState, splitTransform.pool).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if err := splitTransform.observeAndWriteCommand(newCmd); err != nil {
		log.E(ctx, "Failed during creating command pool : %v", err)
		return VkCommandPool(0), err
	}
	return splitTransform.pool, nil
}

// Add adds the command with identifier id to the set of commands that will be split.
func (splitTransform *commandSplitter) Split(ctx context.Context, id api.SubCmdIdx) error {
	splitTransform.requestsSubIndex = append(splitTransform.requestsSubIndex, append(api.SubCmdIdx{}, id...))
	if splitTransform.lastRequest.LessThan(id) {
		splitTransform.lastRequest = append(api.SubCmdIdx{}, id...)
	}

	return nil
}

type commandsplitTransformWriter struct {
	state    *api.GlobalState
	splitter *commandSplitter
}

func newCommandsplitTransformWriter(state *api.GlobalState, splitter *commandSplitter) *commandsplitTransformWriter {
	return &commandsplitTransformWriter{
		state:    state,
		splitter: splitter,
	}
}

func (writer *commandsplitTransformWriter) State() *api.GlobalState {
	return writer.state
}

func (writer *commandsplitTransformWriter) MutateAndWrite(ctx context.Context, id api.CmdID, cmd api.Cmd) error {
	if err := writer.splitter.writeCommand(cmd); err != nil {
		log.E(ctx, "Failed during state rebuilding in command splitter : %v", err)
		return err
	}
	return nil
}
