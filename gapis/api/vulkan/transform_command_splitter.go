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

// To split a renderpass and insert insertion command in it,
// for every subpass in a renderpass, we create 3 renderpasses
type subpassSplitters struct {
	enter        VkRenderPass
	intermediate VkRenderPass
	exit         VkRenderPass
}

type renderPassSplitters []subpassSplitters

type commandSplitter struct {
	requestedCmds   []api.SubCmdIdx
	cmdsOffset      uint64
	mutationStarted bool
	allocations     *allocationTracker
	stateMutator    transform.StateMutator

	readMemoriesForSubmit []*api.AllocResult

	splitRenderPasses          map[VkRenderPass]renderPassSplitters
	rewrittenGraphicsPipelines map[VkPipeline]VkPipeline
	pendingCommandBuffers      []VkCommandBuffer

	pool VkCommandPool
}

func NewCommandSplitter(ctx context.Context, cmdsOffset uint64) *commandSplitter {
	return &commandSplitter{
		requestedCmds:   make([]api.SubCmdIdx, 0, 0),
		cmdsOffset:      cmdsOffset,
		mutationStarted: false,

		splitRenderPasses:          make(map[VkRenderPass]renderPassSplitters),
		rewrittenGraphicsPipelines: make(map[VkPipeline]VkPipeline),
		pendingCommandBuffers:      make([]VkCommandBuffer, 0),

		allocations:  nil,
		stateMutator: nil,
	}
}

// Add adds the command with identifier id to the set of commands that will be split.
// The requested command ids will be sorted and handled in one pass.
func (splitter *commandSplitter) Split(ctx context.Context, id api.SubCmdIdx) error {
	if splitter.mutationStarted {
		return log.Err(ctx, nil, "Commands cannot be requested to split after mutation started")
	}

	if len(id) == 0 {
		return log.Err(ctx, nil, "Requested id is empty")
	}

	id[0] = id[0] + splitter.cmdsOffset

	splitter.requestedCmds = append(splitter.requestedCmds, append(api.SubCmdIdx{}, id...))
	return nil
}

func (splitter *commandSplitter) RequiresAccurateState() bool {
	return false
}

func (splitter *commandSplitter) RequiresInnerStateMutation() bool {
	return true
}

func (splitter *commandSplitter) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	splitter.stateMutator = mutator
}

func (splitter *commandSplitter) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	splitter.allocations = NewAllocationTracker(inputState)
	api.SortSubCmdIDs(splitter.requestedCmds)
	splitter.mutationStarted = true
	return nil
}

func (splitter *commandSplitter) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	if len(splitter.requestedCmds) == 0 {
		return nil, nil
	}

	err := fmt.Errorf("The requested commands to be splitted could not found: ")

	for _, cmdID := range splitter.requestedCmds {
		cmdID[0] = cmdID[0] - splitter.cmdsOffset
		err = fmt.Errorf("%v %v ", err, cmdID)
	}

	log.E(ctx, "Command Splitter Transform Error: %v", err)
	return nil, err
}

func (splitter *commandSplitter) ClearTransformResources(ctx context.Context) {
	splitter.allocations.FreeAllocations()
}

func (splitter *commandSplitter) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if len(inputCommands) == 0 {
		return inputCommands, nil
	}

	if id.GetCommandType() != transform.TransformCommand {
		// We are not interested in the artificial commands from endTransform.
		return inputCommands, nil
	}

	currentSubCmdID := api.SubCmdIdx{uint64(id.GetID())}

	queueSubmitProcessed := false
	for i := range inputCommands {
		queueSubmitCmd, ok := inputCommands[i].(*VkQueueSubmit)
		if ok && splitter.isSubCmdRequestedNext(currentSubCmdID) {
			if queueSubmitProcessed {
				panic("We should not have more than one vkQueueSubmit for a single command")
			}
			queueSubmitProcessed = true

			newCmd, err := splitter.rewriteQueueSubmit(ctx, id.GetID(), queueSubmitCmd, inputState)
			if err != nil {
				log.E(ctx, "Failed during rewriting VkQueueSubmit : %v", err)
				return nil, err
			}
			inputCommands[i] = newCmd
		}

		// It means that this is a VkQueuePresentKHR and
		// we need to get image before QueuePresent
		if splitter.isCmdRequestedNext(currentSubCmdID) {
			insertionCmd := splitter.createInsertionCommand(ctx, VkCommandBuffer(0), currentSubCmdID, inputCommands[i])
			if err := splitter.writeCommand(insertionCmd); err != nil {
				log.E(ctx, "Failed during processing insertion command : %v", err)
				return nil, err
			}
		}

		if err := splitter.writeCommand(inputCommands[i]); err != nil {
			log.E(ctx, "Failed during processing input commands : %v", err)
			return nil, err
		}
	}

	splitter.pendingCommandBuffers = []VkCommandBuffer{}

	return nil, nil
}

func (splitter *commandSplitter) rewriteQueueSubmit(ctx context.Context,
	id api.CmdID, cmd *VkQueueSubmit, inputState *api.GlobalState) (api.Cmd, error) {
	layout := inputState.MemoryLayout
	cb := CommandBuilder{Thread: cmd.Thread()}
	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	submitInfos, err := cmd.PSubmits().Slice(0, uint64(cmd.SubmitCount()), layout).Read(ctx, cmd, inputState, nil)
	if err != nil {
		return nil, err
	}
	newSubmitInfos := []VkSubmitInfo{}

	newQueueSubmit := cb.VkQueueSubmit(cmd.Queue(), cmd.SubmitCount(), cmd.PSubmits(), cmd.Fence(), cmd.Result())
	newQueueSubmit.Extras().MustClone(cmd.Extras().All()...)

	for i := range submitInfos {
		currentSubCmdID := append(api.SubCmdIdx{uint64(id)}, uint64(i))

		newSubmitInfo := submitInfos[i]
		if splitter.isSubCmdRequestedNext(currentSubCmdID) {
			newSubmitInfo, err = splitter.rewriteSubmissionBatch(ctx, currentSubCmdID, submitInfos[i], cmd, inputState)
			if err != nil {
				log.E(ctx, "Failed during splitting submit : %v", err)
				return nil, err
			}
		} else {
			commandBuffers, err := newSubmitInfo.PCommandBuffers().Slice(0, uint64(newSubmitInfo.CommandBufferCount()), layout).Read(ctx, cmd, inputState, nil)
			if err != nil {
				log.E(ctx, "Failed during reading existing command buffer: %v", err)
				return nil, err
			}
			splitter.pendingCommandBuffers = append(splitter.pendingCommandBuffers, commandBuffers...)
		}
		newSubmitInfos = append(newSubmitInfos, newSubmitInfo)

		if splitter.isCmdRequestedNext(currentSubCmdID) {
			insertionCommandBuffer, err := splitter.createCommandBufferWithInsertion(ctx, inputState, cmd, currentSubCmdID)
			if err != nil {
				log.E(ctx, "Failed during creating insertion command buffer")
				return nil, err
			}

			insertionSubmitInfo := NewVkSubmitInfo(
				VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
				NewVoidᶜᵖ(memory.Nullptr),                     // pNext
				0,                                             // waitSemaphoreCount,
				NewVkSemaphoreᶜᵖ(memory.Nullptr),              // pWaitSemaphores
				NewVkPipelineStageFlagsᶜᵖ(memory.Nullptr), // pWaitDstStageMask
				1, // commandBufferCount
				NewVkCommandBufferᶜᵖ(splitter.mustAllocReadDataForSubmit(ctx, inputState, insertionCommandBuffer).Ptr()),
				0,                                // signalSemaphoreCount
				NewVkSemaphoreᶜᵖ(memory.Nullptr), // pSignalSemaphores
			)

			newSubmitInfos = append(newSubmitInfos, insertionSubmitInfo)
		}
	}

	newQueueSubmit.SetSubmitCount(uint32(len(newSubmitInfos)))
	newQueueSubmit.SetPSubmits(NewVkSubmitInfoᶜᵖ(splitter.mustAllocReadDataForSubmit(ctx, inputState, newSubmitInfos).Ptr()))

	for x := range splitter.readMemoriesForSubmit {
		newQueueSubmit.AddRead(splitter.readMemoriesForSubmit[x].Data())
	}

	splitter.readMemoriesForSubmit = []*api.AllocResult{}
	return newQueueSubmit, nil
}

func (splitter *commandSplitter) rewriteSubmissionBatch(ctx context.Context, idx api.SubCmdIdx, submitInfo VkSubmitInfo, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkSubmitInfo, error) {
	layout := inputState.MemoryLayout
	commandBuffers, err := submitInfo.PCommandBuffers().Slice(0, uint64(submitInfo.CommandBufferCount()), layout).Read(ctx, cmd, inputState, nil)
	if err != nil {
		return VkSubmitInfo{}, err
	}
	newSubmitInfo := MakeVkSubmitInfo()

	newCommandBuffers := make([]VkCommandBuffer, 0)
	for i := range commandBuffers {
		currentCommandBuffer := commandBuffers[i]

		currentSubCmdID := append(idx, uint64(i))
		if splitter.isSubCmdRequestedNext(currentSubCmdID) {
			newCommandBuffer, err := splitter.createNewCommandBufferAndBegin(ctx, inputState, cmd, currentCommandBuffer)
			if err != nil {
				return VkSubmitInfo{}, err
			}

			err = splitter.rewriteCommandBuffer(ctx, inputState, currentSubCmdID, cmd, currentCommandBuffer, newCommandBuffer)
			if err != nil {
				return VkSubmitInfo{}, log.Err(ctx, err, "Failed during rewriting command buffer")
			}

			endCommandBufferCmd := endCommandBuffer(cmd.Thread(), newCommandBuffer)
			if err := splitter.writeCommand(endCommandBufferCmd); err != nil {
				log.E(ctx, "Failed during writing end command buffer : %v", err)
				return VkSubmitInfo{}, err
			}
			currentCommandBuffer = newCommandBuffer
		}

		newCommandBuffers = append(newCommandBuffers, currentCommandBuffer)

		if splitter.isCmdRequestedNext(currentSubCmdID) {
			insertionCommandBuffer, err := splitter.createCommandBufferWithInsertion(ctx, inputState, cmd, currentSubCmdID)
			if err != nil {
				log.E(ctx, "Failed during creating insertion command buffer")
				return VkSubmitInfo{}, err
			}
			newCommandBuffers = append(newCommandBuffers, insertionCommandBuffer)
		}
	}

	splitter.pendingCommandBuffers = append(splitter.pendingCommandBuffers, newCommandBuffers...)

	newCbs := splitter.mustAllocReadDataForSubmit(ctx, inputState, newCommandBuffers)
	newSubmitInfo.SetSType(submitInfo.SType())
	newSubmitInfo.SetPNext(submitInfo.PNext())
	newSubmitInfo.SetWaitSemaphoreCount(submitInfo.WaitSemaphoreCount())
	newSubmitInfo.SetPWaitSemaphores(submitInfo.PWaitSemaphores())
	newSubmitInfo.SetPWaitDstStageMask(submitInfo.PWaitDstStageMask())
	newSubmitInfo.SetCommandBufferCount(uint32(len(newCommandBuffers)))
	newSubmitInfo.SetPCommandBuffers(NewVkCommandBufferᶜᵖ(newCbs.Ptr()))
	newSubmitInfo.SetCommandBufferCount(uint32(len(newCommandBuffers)))
	newSubmitInfo.SetSignalSemaphoreCount(submitInfo.SignalSemaphoreCount())
	newSubmitInfo.SetPSignalSemaphores(submitInfo.PSignalSemaphores())
	return newSubmitInfo, nil
}

func (splitter *commandSplitter) rewriteCommandBuffer(
	ctx context.Context,
	inputState *api.GlobalState,
	idx api.SubCmdIdx,
	queueSubmit *VkQueueSubmit,
	existingCommandBuffer VkCommandBuffer,
	newCommandBuffer VkCommandBuffer) error {

	cb := CommandBuilder{Thread: queueSubmit.Thread()}

	existingCommandBufferObject := GetState(inputState).CommandBuffers().Get(existingCommandBuffer)
	for i := uint32(0); i < uint32(existingCommandBufferObject.CommandReferences().Len()); i++ {
		currentCmd := existingCommandBufferObject.CommandReferences().Get(uint32(i))
		currentCmdArgs := GetCommandArgs(ctx, currentCmd, GetState(inputState))

		// If we are outside of a renderpass just copy the commands without modifying
		if _, ok := currentCmdArgs.(VkCmdBeginRenderPassXArgsʳ); !ok {
			if err := splitter.addCommandToCommandBuffer(ctx, inputState, currentCmdArgs, newCommandBuffer, cb); err != nil {
				return err
			}
			continue
		}

		beginRenderPassIndex := i
		endRenderPassIndex, err := splitter.getNextEndRenderPassIndex(ctx, inputState, existingCommandBufferObject, beginRenderPassIndex)
		if err != nil {
			return err
		}

		// Explicitly copy the slices to avoid the possible "gotcha"
		// https://go.dev/blog/slices-intro
		beginIndex := api.SubCmdIdx{}
		beginIndex = append(beginIndex, idx...)
		beginIndex = append(beginIndex, uint64(beginRenderPassIndex))

		endIndex := api.SubCmdIdx{}
		endIndex = append(endIndex, idx...)
		endIndex = append(endIndex, uint64(endRenderPassIndex))

		// If the current renderPassIndex is not requested copy the commands without modifying
		// and jump to the end of the renderpass and continue
		if !splitter.isCmdRequestedNextInInterval(beginIndex, endIndex) {
			// Melih TODO: Due to the b/200873355 we need to fix VkCmdBindPipeline calls
			// outside of the renderpass
			for j := beginRenderPassIndex; j <= endRenderPassIndex; j++ {
				cmdToAdd := existingCommandBufferObject.CommandReferences().Get(uint32(j))
				cmdArgsToAdd := GetCommandArgs(ctx, cmdToAdd, GetState(inputState))
				if err := splitter.addCommandToCommandBuffer(ctx, inputState, cmdArgsToAdd, newCommandBuffer, cb); err != nil {
					return err
				}
			}
			i = endRenderPassIndex
			continue
		}

		// Rewrite the renderpass if we have a request in it
		if err := splitter.rewriteRenderPass(ctx, inputState, queueSubmit, existingCommandBufferObject, newCommandBuffer,
			idx, beginRenderPassIndex, endRenderPassIndex); err != nil {
			return err
		}

		// Jump to the end of the renderpass
		i = endRenderPassIndex
	}

	return nil
}

func (splitter *commandSplitter) rewriteRenderPass(
	ctx context.Context,
	inputState *api.GlobalState,
	queueSubmit *VkQueueSubmit,
	existingCommandBufferObject CommandBufferObjectʳ,
	newCommandBuffer VkCommandBuffer,
	idx api.SubCmdIdx,
	beginRenderPassIndex uint32,
	endRenderPassIndex uint32) error {

	replaceCommand := false
	stateObject := GetState(inputState)
	cb := CommandBuilder{Thread: queueSubmit.Thread()}

	currentSubpass := 0
	currentRenderPassSplitters := renderPassSplitters{}
	currentRenderPassArgs := NilVkCmdBeginRenderPassXArgsʳ

	for i := beginRenderPassIndex; i <= endRenderPassIndex; i++ {
		currentCmd := existingCommandBufferObject.CommandReferences().Get(i)
		newCommandsArgs := make([]interface{}, 0)
		currentCommandArgs := GetCommandArgs(ctx, currentCmd, stateObject)

		switch arg := currentCommandArgs.(type) {
		case VkCmdBeginRenderPassXArgsʳ:
			currentRenderPassSplitters = splitter.createRenderPassSplitters(
				ctx, inputState, stateObject.RenderPasses().Get(arg.RenderPassBeginInfo().RenderPass()))
			currentSubpass = 0
			currentCommandArgs = NewVkCmdBeginRenderPassXArgsʳ(
				NewRenderPassBeginInfoʳ(
					currentRenderPassSplitters[currentSubpass].enter,
					arg.RenderPassBeginInfo().Framebuffer(),
					arg.RenderPassBeginInfo().RenderArea(),
					arg.RenderPassBeginInfo().ClearValues(),
					arg.RenderPassBeginInfo().DeviceGroupBeginInfo(),
				),
				NewSubpassBeginInfoʳ(
					VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
				),
				arg.Version(),
			)
			currentRenderPassArgs = arg
		case VkCmdNextSubpassXArgsʳ:
			currentCommandArgs = NewVkCmdEndRenderPassXArgsʳ(
				NewSubpassEndInfoʳ(),
				arg.Version(),
			)
			newCommandsArgs = append(newCommandsArgs, NewVkCmdBeginRenderPassXArgsʳ(
				NewRenderPassBeginInfoʳ(
					currentRenderPassSplitters[currentSubpass].exit,
					currentRenderPassArgs.RenderPassBeginInfo().Framebuffer(),
					currentRenderPassArgs.RenderPassBeginInfo().RenderArea(),
					currentRenderPassArgs.RenderPassBeginInfo().ClearValues(),
					currentRenderPassArgs.RenderPassBeginInfo().DeviceGroupBeginInfo(),
				),
				NewSubpassBeginInfoʳ(
					VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
				),
				currentRenderPassArgs.Version(),
			))

			newCommandsArgs = append(newCommandsArgs, NewVkCmdEndRenderPassXArgsʳ(
				NewSubpassEndInfoʳ(),
				arg.Version(),
			))

			currentSubpass++
			newCommandsArgs = append(newCommandsArgs, NewVkCmdBeginRenderPassXArgsʳ(
				NewRenderPassBeginInfoʳ(
					currentRenderPassSplitters[currentSubpass].enter,
					currentRenderPassArgs.RenderPassBeginInfo().Framebuffer(),
					currentRenderPassArgs.RenderPassBeginInfo().RenderArea(),
					currentRenderPassArgs.RenderPassBeginInfo().ClearValues(),
					currentRenderPassArgs.RenderPassBeginInfo().DeviceGroupBeginInfo(),
				),
				NewSubpassBeginInfoʳ(
					VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
				),
				currentRenderPassArgs.Version(),
			))
		case VkCmdEndRenderPassXArgsʳ:
			newCommandsArgs = append(newCommandsArgs, NewVkCmdBeginRenderPassXArgsʳ(
				NewRenderPassBeginInfoʳ(
					currentRenderPassSplitters[currentSubpass].exit,
					currentRenderPassArgs.RenderPassBeginInfo().Framebuffer(),
					currentRenderPassArgs.RenderPassBeginInfo().RenderArea(),
					currentRenderPassArgs.RenderPassBeginInfo().ClearValues(),
					currentRenderPassArgs.RenderPassBeginInfo().DeviceGroupBeginInfo(),
				),
				NewSubpassBeginInfoʳ(
					VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
				),
				currentRenderPassArgs.Version(),
			))
			newCommandsArgs = append(newCommandsArgs, NewVkCmdEndRenderPassXArgsʳ(
				NewSubpassEndInfoʳ(),
				arg.Version(),
			))
			currentRenderPassArgs = NilVkCmdBeginRenderPassXArgsʳ
			currentSubpass = 0
		case VkCmdBindPipelineArgsʳ:
			// Melih TODO: Due to the b/200873355 we need to fix VkCmdBindPipeline calls
			// outside of the renderpass
			if arg.PipelineBindPoint() == VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
				// Graphics pipeline, must be split (maybe)
				if stateObject.RenderPasses().Get(currentRenderPassArgs.RenderPassBeginInfo().RenderPass()).SubpassDescriptions().Len() > 1 {
					// If we have more than one renderpass, then we should replace
					newPipeline, err := splitter.rewriteGraphicsPipeline(ctx, inputState,
						arg.Pipeline(), currentRenderPassSplitters[currentSubpass].enter)
					if err != nil {
						log.E(ctx, "Failed during rewriting graphics pipeline : %v", err)
						return err
					}
					newPipelineArgs := arg.Clone(api.CloneContext{})
					newPipelineArgs.SetPipeline(newPipeline)
					currentCommandArgs = newPipelineArgs
				}
			}
		case VkCmdExecuteCommandsArgsʳ:
			currentSubCmdID := append(idx, uint64(i))
			// If there is any framebuffer is requested in a command buffer executed by a VkCmdExecuteCommands
			// flatten the all command buffers.
			if splitter.isSubCmdRequestedNext(currentSubCmdID) {
				replaceCommand = true
				for j := 0; j < arg.CommandBuffers().Len(); j++ {
					executedCmdBufferSubCmdID := append(currentSubCmdID, uint64(j))
					currentCommandBufferToExecute := arg.CommandBuffers().Get(uint32(j))
					if err := splitter.flattenSecondaryCommandBuffer(
						ctx,
						inputState,
						executedCmdBufferSubCmdID,
						queueSubmit,
						currentCommandBufferToExecute,
						newCommandBuffer,
						currentRenderPassArgs,
						currentRenderPassSplitters[currentSubpass].intermediate); err != nil {
						log.E(ctx, "Failed during splitting command buffer : %v", err)
						return err
					}
				}
			}
			if splitter.isCmdRequestedNext(currentSubCmdID) {
				insertionCmd := splitter.createInsertionCommand(ctx, newCommandBuffer, currentSubCmdID, queueSubmit)
				if err := splitter.writeCommand(insertionCmd); err != nil {
					log.E(ctx, "Failed during processing insertion command : %v", err)
					return err
				}
			}
		}

		if !replaceCommand {
			if err := splitter.addCommandToCommandBuffer(
				ctx, inputState, currentCommandArgs, newCommandBuffer, cb); err != nil {
				return err
			}
		}

		currentSubCmdID := append(idx, uint64(i))
		if splitter.isCmdRequestedNext(currentSubCmdID) {
			insertionCmd := splitter.createInsertionCommand(ctx, newCommandBuffer, currentSubCmdID, queueSubmit)
			insertedCmdsArgs := splitter.insertInsertionCommand(ctx, currentRenderPassArgs,
				currentRenderPassSplitters[currentSubpass].intermediate, insertionCmd)
			newCommandsArgs = append(newCommandsArgs, insertedCmdsArgs...)
		}

		for _, args := range newCommandsArgs {
			insertionCmd, ok := args.(api.Cmd)
			if !ok {
				// If it's not an insertion command, add the command to new command buffer
				splitter.addCommandToCommandBuffer(ctx, inputState, args, newCommandBuffer, cb)
				continue
			}

			// Mutate the insertion command without adding it to new command buffer
			if err := splitter.writeCommand(insertionCmd); err != nil {
				log.E(ctx, "Failed during inserting insertion command : %v", err)
				return err
			}
		}
	}

	return nil
}

func (splitter *commandSplitter) flattenSecondaryCommandBuffer(
	ctx context.Context,
	inputState *api.GlobalState,
	idx api.SubCmdIdx,
	queueSubmit *VkQueueSubmit,
	secondaryCommandBuffer VkCommandBuffer,
	newCommandBuffer VkCommandBuffer,
	currentRenderPassArgs VkCmdBeginRenderPassXArgsʳ,
	newRenderpass VkRenderPass) error {

	cb := CommandBuilder{Thread: queueSubmit.Thread()}
	stateObject := GetState(inputState)
	secondaryCommandBufferObject := stateObject.CommandBuffers().Get(secondaryCommandBuffer)
	for i := uint32(0); i < uint32(secondaryCommandBufferObject.CommandReferences().Len()); i++ {
		currentCmd := secondaryCommandBufferObject.CommandReferences().Get(i)
		currentCommandArgs := GetCommandArgs(ctx, currentCmd, stateObject)

		var err error
		switch cmdArgs := currentCommandArgs.(type) {
		case VkCmdBindPipelineArgsʳ:
			currentCommandArgs, err = splitter.rewritePipelineBind(ctx, inputState, cmdArgs, currentRenderPassArgs, newRenderpass)
			if err != nil {
				return err
			}
		}

		if err := splitter.addCommandToCommandBuffer(
			ctx, inputState, currentCommandArgs, newCommandBuffer, cb); err != nil {
			return err
		}

		currentSubCmdID := append(idx, uint64(i))
		if splitter.isCmdRequestedNext(currentSubCmdID) {
			insertionCmd := splitter.createInsertionCommand(ctx, newCommandBuffer, currentSubCmdID, queueSubmit)
			insertedCmdsArgs := splitter.insertInsertionCommand(ctx, currentRenderPassArgs, newRenderpass, insertionCmd)

			for _, cmdArgs := range insertedCmdsArgs {
				insertionCmd, ok := cmdArgs.(api.Cmd)
				if !ok {
					// If it's not an insertion command, add the command to new command buffer
					splitter.addCommandToCommandBuffer(ctx, inputState, cmdArgs, newCommandBuffer, cb)
					continue
				}

				// Mutate the insertion command without adding it to new command buffer
				if err := splitter.writeCommand(insertionCmd); err != nil {
					log.E(ctx, "Failed during inserting insertion command : %v", err)
					return err
				}
			}
		}
	}

	return nil
}

func (splitter *commandSplitter) rewritePipelineBind(
	ctx context.Context,
	inputState *api.GlobalState,
	bindPipelineArgs VkCmdBindPipelineArgsʳ,
	currentRenderPassArgs VkCmdBeginRenderPassXArgsʳ,
	newRenderPass VkRenderPass) (VkCmdBindPipelineArgsʳ, error) {

	// We are only interested in graphics pipeline
	if bindPipelineArgs.PipelineBindPoint() != VkPipelineBindPoint_VK_PIPELINE_BIND_POINT_GRAPHICS {
		return bindPipelineArgs, nil
	}

	stateObject := GetState(inputState)

	// If we have only one renderpass we don't modify the renderpass, therefore no change is required.
	if stateObject.RenderPasses().Get(currentRenderPassArgs.RenderPassBeginInfo().RenderPass()).SubpassDescriptions().Len() <= 1 {
		return bindPipelineArgs, nil
	}

	newPipeline, err := splitter.rewriteGraphicsPipeline(ctx, inputState,
		bindPipelineArgs.Pipeline(), newRenderPass)
	if err != nil {
		log.E(ctx, "Failed during rewriting graphics pipeline : %v", err)
		return VkCmdBindPipelineArgsʳ{}, err
	}

	newPipelineArgs := bindPipelineArgs.Clone(api.CloneContext{})
	newPipelineArgs.SetPipeline(newPipeline)
	return newPipelineArgs, nil
}

type renderPassRequirements struct {
	loadAttachment                 bool
	storeAttachment                bool
	updateFinalLayout              bool
	updateSubpassLayoutTransitions bool
	preventAttachmentResolve       bool
}

func (splitter *commandSplitter) createRenderPassSplitters(ctx context.Context, inputState *api.GlobalState, renderPassObject RenderPassObjectʳ) renderPassSplitters {
	st := GetState(inputState)

	// Check if we already split this renderpass
	if splitRenderPasses, ok := splitter.splitRenderPasses[renderPassObject.VulkanHandle()]; ok {
		return splitRenderPasses
	}

	handles := make(renderPassSplitters, 0)
	currentLayouts := make(map[uint32]VkImageLayout)

	for i := uint32(0); i < uint32(renderPassObject.AttachmentDescriptions().Len()); i++ {
		currentLayouts[i] = renderPassObject.AttachmentDescriptions().Get(i).InitialLayout()
	}

	tempTransformWriter := newCommandSplitTransformWriter(inputState, splitter)
	sb := st.newStateBuilder(ctx, newTransformerOutput(tempTransformWriter))

	// For every subpass we have, we create 3 kind of renderpass. How they will be used at the end
	// is based on what commands requested.
	//
	// For example for a renderpass that has 3 subpass with 3 draw in each:
	// {
	//     SP1[draw, draw, draw], SP2[draw, draw, draw], SP3[draw, draw, draw]
	// }
	//
	// Let's assume we request drawcalls with *
	// {
	//     SP1[draw*, draw, draw], SP2[draw, draw*, draw], SP3[draw, draw*, draw*]
	// }
	//
	// The new renderpasses at the end of this transform will be
	// {
	//     Enter1[draw*], [InsertionCmd], Intermediate1[draw, draw], Exit1[],
	//     Enter2[draw, draw*], [InsertionCmd], Intermedite2[draw], Exit2[],
	//     Enter3[draw, draw*], [InsertionCmd], Intermediate2[draw*], [InsertionCmd], Intermediate2[], Exit2[]
	// }

	for i := uint32(0); i < uint32(renderPassObject.SubpassDescriptions().Len()); i++ {
		// This is the first renderpass for the subpass
		// If it's also the first subpass then we should behave exactly like the original renderpass
		// in terms of loading the attachments.
		// If it's not the first subpass, then we should always load attachments as the original application
		// would be able to read it between subpasses.
		//
		// We do always store and patch the final layouts of attachments for the next renderpass.
		// This allows the final layout of this renderpass will be the same with inital layout,
		// so that, the intermediate renderpass gets the same initial layout with the original renderpass.
		//
		// Also we do update subpass final layouts for the intermediate renderpass.
		//
		// We prevent resolving the attachments because those may be written by the intermediate renderpasses
		// on the way.
		isFirstSubpass := (i == 0)
		patchRequirements := renderPassRequirements{
			loadAttachment:                 !isFirstSubpass,
			storeAttachment:                true,
			updateFinalLayout:              true,
			updateSubpassLayoutTransitions: true,
			preventAttachmentResolve:       true,
		}
		enterRenderPass := createSplitRenderPassObject(inputState, currentLayouts,
			renderPassObject, i, patchRequirements)
		sb.createRenderPass(enterRenderPass)

		// This is the intermediate renderpass for the subpass.
		// After every insertion command we will begin this renderpass to continue.
		// If we are requesting the last drawcall, this renderpass may be empty.
		//
		// We always load and store every attachment and we always update the final layout as well.
		// Because this renderpass means being in the middle of a subpass.
		//
		// We prevent resolving the attachments because those may be written by the intermediate renderpasses
		// on the way.
		patchRequirements = renderPassRequirements{
			loadAttachment:                 true,
			storeAttachment:                true,
			updateFinalLayout:              true,
			updateSubpassLayoutTransitions: true,
			preventAttachmentResolve:       true,
		}
		intermediateRenderPass := createSplitRenderPassObject(inputState, currentLayouts,
			renderPassObject, i, patchRequirements)
		sb.createRenderPass(intermediateRenderPass)

		// This is the exit renderpass for the subpass.
		// This renderpass is always empty.
		//
		// If this is the last subpass we use this to represent ending of the original renderpass.
		// Therefore we do not change the store operation or final layout
		//
		// Otherwise, this represent a subpass transition(e.g. VkCmdNextSubpass).
		// Thereforewe always load, store and update the final layout.
		//
		// We keep resolve the attachments as the original subpass. So if there is resolve
		// between subpasses, it can happen here.
		lastSubpass := (i == uint32(renderPassObject.SubpassDescriptions().Len()-1))
		patchRequirements = renderPassRequirements{
			loadAttachment:                 true,
			storeAttachment:                !lastSubpass,
			updateFinalLayout:              !lastSubpass,
			updateSubpassLayoutTransitions: !lastSubpass,
			preventAttachmentResolve:       false,
		}
		exitRenderPass := createSplitRenderPassObject(
			inputState, currentLayouts, renderPassObject, i,
			patchRequirements)
		sb.createRenderPass(exitRenderPass)

		newRenderPasses := subpassSplitters{
			enter:        enterRenderPass.VulkanHandle(),
			intermediate: intermediateRenderPass.VulkanHandle(),
			exit:         exitRenderPass.VulkanHandle(),
		}

		handles = append(handles, newRenderPasses)
	}

	splitter.splitRenderPasses[renderPassObject.VulkanHandle()] = handles
	return handles
}

func createSplitRenderPassObject(
	inputState *api.GlobalState,
	currentLayouts map[uint32]VkImageLayout,
	renderPassObject RenderPassObjectʳ,
	subpassIndex uint32,
	patchRequirements renderPassRequirements) RenderPassObjectʳ {

	newRenderPass := renderPassObject.Clone(api.CloneContext{})
	newRenderPass.SetVulkanHandle(
		VkRenderPass(newUnusedID(true, func(x uint64) bool {
			return GetState(inputState).RenderPasses().Contains(VkRenderPass(x))
		})))

	updateAllAttachmentDescriptions(currentLayouts, newRenderPass, patchRequirements)

	subpassDescription := newRenderPass.SubpassDescriptions().Get(subpassIndex)

	if patchRequirements.updateSubpassLayoutTransitions {
		updateAttachmentFinalLayouts(currentLayouts, newRenderPass, subpassDescription.InputAttachments())
		updateAttachmentFinalLayouts(currentLayouts, newRenderPass, subpassDescription.ColorAttachments())
		if !subpassDescription.DepthStencilAttachment().IsNil() {
			updateAttachmentFinalLayout(currentLayouts, newRenderPass, subpassDescription.DepthStencilAttachment().Get())
		}
	}

	if patchRequirements.preventAttachmentResolve {
		subpassDescription.ResolveAttachments().Clear()
	}

	subpassDescription.PreserveAttachments().Clear()

	newRenderPass.SubpassDescriptions().Clear()
	newRenderPass.SubpassDescriptions().Add(0, subpassDescription)
	newRenderPass.SubpassDependencies().Clear()
	return newRenderPass
}

func updateAttachmentFinalLayouts(currentLayouts map[uint32]VkImageLayout, renderPassObject RenderPassObjectʳ, attachmentReferences U32ːAttachmentReferenceᵐ) {
	for i := 0; i < len(attachmentReferences.All()); i++ {
		updateAttachmentFinalLayout(currentLayouts, renderPassObject, attachmentReferences.Get(uint32(i)))
	}
}

func updateAttachmentFinalLayout(currentLayouts map[uint32]VkImageLayout, renderPassObject RenderPassObjectʳ, attachmentReference AttachmentReference) {
	// We are not interested with unusued attachment
	if attachmentReference.Attachment() == VK_ATTACHMENT_UNUSED {
		return
	}

	// set the current layout for the attachment reference
	currentLayouts[attachmentReference.Attachment()] = attachmentReference.Layout()

	// Update the layout of the corresponding attachment description in renderpass object
	attachmentDescription := renderPassObject.AttachmentDescriptions().Get(attachmentReference.Attachment())
	attachmentDescription.SetFinalLayout(attachmentReference.Layout())
	renderPassObject.AttachmentDescriptions().Add(attachmentReference.Attachment(), attachmentDescription)
}

func updateAllAttachmentDescriptions(currentLayouts map[uint32]VkImageLayout, renderPassObject RenderPassObjectʳ, patchRequirements renderPassRequirements) {
	for i := uint32(0); i < uint32(len(currentLayouts)); i++ {
		attachmentDescription := renderPassObject.AttachmentDescriptions().Get(i)

		attachmentDescription.SetInitialLayout(currentLayouts[i])

		if patchRequirements.updateFinalLayout {
			attachmentDescription.SetFinalLayout(currentLayouts[i])
		}

		if patchRequirements.loadAttachment {
			attachmentDescription.SetLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD)
			attachmentDescription.SetStencilLoadOp(VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD)
		}

		if patchRequirements.storeAttachment {
			attachmentDescription.SetStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
			attachmentDescription.SetStencilStoreOp(VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE)
		}

		renderPassObject.AttachmentDescriptions().Add(i, attachmentDescription)
	}
}

func (splitter *commandSplitter) addCommandToCommandBuffer(ctx context.Context, inputState *api.GlobalState, commandArgs interface{}, commandBufer VkCommandBuffer, cb CommandBuilder) error {
	cleanup, newCmd, err := AddCommand(ctx, cb, commandBufer, inputState, inputState, commandArgs)
	if err != nil {
		log.E(ctx, "Failed during copying the command : %v", err)
		return err
	}

	if err = splitter.writeCommand(newCmd); err != nil {
		log.E(ctx, "Failed during writing new command : %v", err)
		return err
	}
	cleanup()
	return nil
}

func (splitter *commandSplitter) getNextEndRenderPassIndex(ctx context.Context, inputState *api.GlobalState, existingCommandBuffer CommandBufferObjectʳ, beginRenderPassId uint32) (uint32, error) {
	for i := beginRenderPassId; i < uint32(existingCommandBuffer.CommandReferences().Len()); i++ {
		currentCmd := existingCommandBuffer.CommandReferences().Get(uint32(i))
		cmdArgs := GetCommandArgs(ctx, currentCmd, GetState(inputState))
		if _, ok := cmdArgs.(VkCmdEndRenderPassXArgsʳ); ok {
			return i, nil
		}
	}

	return 0, fmt.Errorf("No end renderpass found!")
}

func (splitter *commandSplitter) createCommandBufferWithInsertion(ctx context.Context, inputState *api.GlobalState, cmd *VkQueueSubmit, subCmdID api.SubCmdIdx) (VkCommandBuffer, error) {
	newCommandBuffer, err := splitter.createNewCommandBufferAndBegin(ctx, inputState, cmd, VkCommandBuffer(0))
	if err != nil {
		log.E(ctx, "Failed during creating insertion command buffer")
		return VkCommandBuffer(0), err
	}

	splitter.pendingCommandBuffers = append(splitter.pendingCommandBuffers, newCommandBuffer)

	newCmd := splitter.createInsertionCommand(ctx, newCommandBuffer, subCmdID, cmd)
	if err := splitter.writeCommand(newCmd); err != nil {
		log.E(ctx, "Failed during writing insertion command")
		return VkCommandBuffer(0), err
	}

	endCommandBufferCmd := endCommandBuffer(cmd.Thread(), newCommandBuffer)
	if err := splitter.writeCommand(endCommandBufferCmd); err != nil {
		log.E(ctx, "Failed during writing end command buffer for insertion: %v", err)
		return VkCommandBuffer(0), err
	}

	return newCommandBuffer, nil
}

func (splitter *commandSplitter) insertInsertionCommand(
	ctx context.Context,
	currentRenderPassArgs VkCmdBeginRenderPassXArgsʳ,
	splittingRenderPass VkRenderPass,
	insertionCmd api.Cmd) []interface{} {
	if currentRenderPassArgs == NilVkCmdBeginRenderPassXArgsʳ {
		return []interface{}{insertionCmd}
	}

	return []interface{}{
		// If we are inside a renderpass, we need to split it
		NewVkCmdEndRenderPassXArgsʳ(
			NewSubpassEndInfoʳ(),
			currentRenderPassArgs.Version(),
		),
		insertionCmd,
		// Begin new renderpass for the remaining commands in the original renderpass
		NewVkCmdBeginRenderPassXArgsʳ(
			NewRenderPassBeginInfoʳ(
				splittingRenderPass,
				currentRenderPassArgs.RenderPassBeginInfo().Framebuffer(),
				currentRenderPassArgs.RenderPassBeginInfo().RenderArea(),
				currentRenderPassArgs.RenderPassBeginInfo().ClearValues(),
				currentRenderPassArgs.RenderPassBeginInfo().DeviceGroupBeginInfo(),
			),
			NewSubpassBeginInfoʳ(
				VkSubpassContents_VK_SUBPASS_CONTENTS_INLINE,
			),
			currentRenderPassArgs.Version(),
		),
	}
}

func (splitter *commandSplitter) createInsertionCommand(ctx context.Context, cmdBuffer VkCommandBuffer, subCmdId api.SubCmdIdx, callee api.Cmd) api.Cmd {
	splitter.removeFromRequestedList(subCmdId)
	return &InsertionCommand{
		cmdBuffer,
		append([]VkCommandBuffer{}, splitter.pendingCommandBuffers...),
		subCmdId,
		callee,
	}
}

func (splitter *commandSplitter) createNewCommandBufferAndBegin(
	ctx context.Context, inputState *api.GlobalState,
	cmd *VkQueueSubmit, referenceCommandBuffer VkCommandBuffer) (VkCommandBuffer, error) {

	queue := GetState(inputState).Queues().Get(cmd.Queue())

	if splitter.pool == VkCommandPool(0) {
		commandPoolID, poolCmd := createNewCommandPool(
			ctx, splitter.allocations, inputState,
			cmd.Thread(), queue.Device(), queue.Family())

		if err := splitter.writeCommand(poolCmd); err != nil {
			return VkCommandBuffer(0), err
		}

		splitter.pool = commandPoolID
	}

	commandBuffer, createCmd := createNewCommandBuffer(
		ctx, splitter.allocations, inputState,
		cmd.Thread(), queue.Device(), splitter.pool,
		VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY)

	if err := splitter.writeCommand(createCmd); err != nil {
		return VkCommandBuffer(0), err
	}

	beginCmd := api.Cmd(nil)

	if referenceCommandBuffer != VkCommandBuffer(0) {
		referenceCommandBufferObject := GetState(inputState).CommandBuffers().Get(referenceCommandBuffer)
		beginCmd = beginCommandBufferFromExistingCommandBuffer(ctx, splitter.allocations, inputState,
			cmd.Thread(), commandBuffer, referenceCommandBufferObject)
	} else {
		beginCmd = beginCommandBuffer(ctx, splitter.allocations, inputState,
			cmd.Thread(), commandBuffer)
	}

	if err := splitter.writeCommand(beginCmd); err != nil {
		return VkCommandBuffer(0), err
	}

	return commandBuffer, nil
}

func (splitter *commandSplitter) rewriteGraphicsPipeline(
	ctx context.Context,
	inputState *api.GlobalState,
	graphicsPipeline VkPipeline,
	renderPass VkRenderPass) (VkPipeline, error) {
	if gp, ok := splitter.rewrittenGraphicsPipelines[graphicsPipeline]; ok {
		return gp, nil
	}

	stateObject := GetState(inputState)
	tempTransformWriter := newCommandSplitTransformWriter(inputState, splitter)
	sb := stateObject.newStateBuilder(ctx, newTransformerOutput(tempTransformWriter))

	newGp := stateObject.GraphicsPipelines().Get(graphicsPipeline).Clone(api.CloneContext{})
	newGp.SetVulkanHandle(
		VkPipeline(newUnusedID(true, func(x uint64) bool {
			return stateObject.GraphicsPipelines().Contains(VkPipeline(x)) ||
				stateObject.ComputePipelines().Contains(VkPipeline(x))
		})))

	newGp.SetRenderPass(stateObject.RenderPasses().Get(renderPass))
	newGp.SetSubpass(0)
	sb.createGraphicsPipeline(newGp)

	splitter.rewrittenGraphicsPipelines[graphicsPipeline] = newGp.VulkanHandle()
	return newGp.VulkanHandle(), nil
}

func (splitter *commandSplitter) writeCommand(cmd api.Cmd) error {
	return splitter.stateMutator([]api.Cmd{cmd})
}

// This function should return only if next requested command is a sub command of id
// but not itself.
func (splitter *commandSplitter) isSubCmdRequestedNext(id api.SubCmdIdx) bool {
	if len(splitter.requestedCmds) == 0 {
		return false
	}

	// Requested commands are sorted and as we traverse in our command tree in order,
	// if we cannot find it in first element, it will not be there after
	if id.Equals(splitter.requestedCmds[0]) {
		return false
	}

	return id.Contains(splitter.requestedCmds[0])
}

func (splitter *commandSplitter) isCmdRequestedNext(id api.SubCmdIdx) bool {
	if len(splitter.requestedCmds) == 0 {
		return false
	}

	// Requested commands are sorted and as we traverse in our command tree in order,
	// if we cannot find it in first element, it will not be there after unless it's a parent
	return id.Equals(splitter.requestedCmds[0])
}

// Returns true if the next requested cmd is in an interval of cmds. e.g renderpass
func (splitter *commandSplitter) isCmdRequestedNextInInterval(begin api.SubCmdIdx, end api.SubCmdIdx) bool {
	if len(splitter.requestedCmds) == 0 {
		return false
	}

	return splitter.requestedCmds[0].InRange(begin, end)
}

func (splitter *commandSplitter) removeFromRequestedList(id api.SubCmdIdx) {
	if len(splitter.requestedCmds) == 0 {
		panic("Requested Command list is empty, this should not happen")
	}

	// Requested commands are sorted and as we traverse in our command tree in order,
	// if we cannot find it in first element, it will not be there after
	if !id.Equals(splitter.requestedCmds[0]) {
		panic("Requested command is not the first element, this should not happen")
	}

	splitter.requestedCmds = splitter.requestedCmds[1:len(splitter.requestedCmds)]
}

func (splitter *commandSplitter) mustAllocReadDataForSubmit(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := splitter.allocations.AllocDataOrPanic(ctx, v...)
	splitter.readMemoriesForSubmit = append(splitter.readMemoriesForSubmit, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

type commandsplitTransformWriter struct {
	state    *api.GlobalState
	splitter *commandSplitter
}

func newCommandSplitTransformWriter(state *api.GlobalState, splitter *commandSplitter) *commandsplitTransformWriter {
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
