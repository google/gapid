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

type commandDisabler struct {
	disabledCommands []api.SubCmdIdx
	cmdsOffset       uint64

	stateMutator transform.StateMutator
	allocations  *allocationTracker

	readMemoriesForSubmit []*api.AllocResult
	readMemoriesForCmd    []*api.AllocResult
	writeMemoriesForCmd   []*api.AllocResult

	pool VkCommandPool
}

func newCommandDisabler(ctx context.Context, cmdsOffset uint64) *commandDisabler {
	return &commandDisabler{
		disabledCommands:      make([]api.SubCmdIdx, 0, 0),
		readMemoriesForSubmit: make([]*api.AllocResult, 0),
		stateMutator:          nil,
		allocations:           nil,
		cmdsOffset:            cmdsOffset,
	}
}

// Remove removes a draw call command from a command buffer.
func (disablerTransform *commandDisabler) remove(ctx context.Context, id api.SubCmdIdx) error {
	if len(disablerTransform.disabledCommands) > 0 {
		return fmt.Errorf("Multiple Drawcall removal not implemented")
	}

	if len(id) == 0 {
		return fmt.Errorf("Requested id is empty")
	}

	if len(id) > 4 {
		return fmt.Errorf("Drawcall removal from secondary command buffer is not implemented")
	}

	id[0] = id[0] + disablerTransform.cmdsOffset

	disablerTransform.disabledCommands = append(disablerTransform.disabledCommands,
		append(api.SubCmdIdx{}, id...))
	return nil
}

func (disablerTransform *commandDisabler) RequiresAccurateState() bool {
	return false
}

func (disablerTransform *commandDisabler) RequiresInnerStateMutation() bool {
	return true
}

func (disablerTransform *commandDisabler) SetInnerStateMutationFunction(mutator transform.StateMutator) {
	disablerTransform.stateMutator = mutator
}

func (disablerTransform *commandDisabler) BeginTransform(ctx context.Context, inputState *api.GlobalState) error {
	disablerTransform.allocations = NewAllocationTracker(inputState)
	return nil
}

func (disablerTransform *commandDisabler) EndTransform(ctx context.Context, inputState *api.GlobalState) ([]api.Cmd, error) {
	if len(disablerTransform.disabledCommands) == 0 {
		return nil, nil
	}

	err := fmt.Errorf("The requested commands to be disabled could not found: ")

	for _, cmdID := range disablerTransform.disabledCommands {
		cmdID[0] = cmdID[0] - disablerTransform.cmdsOffset
		err = fmt.Errorf("%v %v ", err, cmdID)
	}

	return nil, err
}

func (disablerTransform *commandDisabler) ClearTransformResources(ctx context.Context) {
	disablerTransform.allocations.FreeAllocations()
}

func (disablerTransform *commandDisabler) TransformCommand(ctx context.Context, id transform.CommandID, inputCommands []api.Cmd, inputState *api.GlobalState) ([]api.Cmd, error) {
	if len(inputCommands) == 0 {
		return inputCommands, nil
	}

	if id.GetCommandType() != transform.TransformCommand {
		// We are not interested in the artificial commands from endTransform.
		return inputCommands, nil
	}

	currentSubCmdID := api.SubCmdIdx{uint64(id.GetID())}
	if !disablerTransform.doesContainDisabledCmd(currentSubCmdID) {
		return inputCommands, nil
	}

	queueSubmitProcessed := false
	for _, cmd := range inputCommands {
		if queueSubmitCmd, ok := cmd.(*VkQueueSubmit); ok {
			if queueSubmitProcessed {
				panic("We should not have more than one vkQueueSubmit for a single command")
			}

			queueSubmitProcessed = true
			if err := disablerTransform.removeCommandFromVkQueueSubmit(ctx, currentSubCmdID, queueSubmitCmd, inputState); err != nil {
				log.E(ctx, "Failed during removing command from VkQueueSubmit : %v", err)
				return nil, err
			}
		} else {
			if err := disablerTransform.writeCommand(cmd); err != nil {
				log.E(ctx, "Failed during processing input commands : %v", err)
				return nil, err
			}
		}
	}

	if !queueSubmitProcessed {
		return nil, fmt.Errorf("No queue submit has found in command path")
	}

	return nil, nil
}

func (disablerTransform *commandDisabler) removeCommandFromVkQueueSubmit(ctx context.Context, idx api.SubCmdIdx, cmd *VkQueueSubmit, inputState *api.GlobalState) error {
	layout := inputState.MemoryLayout
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	submitInfos := cmd.PSubmits().Slice(0, uint64(cmd.SubmitCount()), layout).MustRead(ctx, cmd, inputState, nil)
	newSubmitInfos := []VkSubmitInfo{}

	newSubmit := cb.VkQueueSubmit(cmd.Queue(), cmd.SubmitCount(), cmd.PSubmits(), cmd.Fence(), cmd.Result())
	newSubmit.Extras().MustClone(cmd.Extras().All()...)

	for i, submitInfo := range submitInfos {
		currentSubCmdID := append(idx, uint64(i))
		if !disablerTransform.doesContainDisabledCmd(currentSubCmdID) {
			newSubmitInfos = append(newSubmitInfos, submitInfo)
			continue
		}

		newSubmitInfo, err := disablerTransform.removeCommandFromSubmit(ctx, currentSubCmdID, submitInfo, cmd, inputState)
		if err != nil {
			log.E(ctx, "Failed during removing command from submit : %v", err)
			return err
		}

		newSubmitInfos = append(newSubmitInfos, newSubmitInfo)
	}

	if len(newSubmitInfos) != len(submitInfos) {
		return fmt.Errorf("Number of queue submits has changed")
	}

	newSubmit.SetPSubmits(NewVkSubmitInfoᶜᵖ(disablerTransform.mustAllocReadDataForSubmit(ctx, inputState, newSubmitInfos).Ptr()))

	for _, mem := range disablerTransform.readMemoriesForSubmit {
		newSubmit.AddRead(mem.Data())
	}
	disablerTransform.readMemoriesForSubmit = []*api.AllocResult{}

	if err := disablerTransform.writeCommand(newSubmit); err != nil {
		log.E(ctx, "Failed during writing VkQueueSubmit : %v", err)
		return err
	}

	return nil
}

func (disablerTransform *commandDisabler) removeCommandFromSubmit(ctx context.Context, idx api.SubCmdIdx, submitInfo VkSubmitInfo, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkSubmitInfo, error) {
	layout := inputState.MemoryLayout
	// pCommandBuffers
	commandBuffers := submitInfo.PCommandBuffers().Slice(0, uint64(submitInfo.CommandBufferCount()), layout).MustRead(ctx, cmd, inputState, nil)
	newCommandBuffers := make([]VkCommandBuffer, 0, len(commandBuffers))

	for i, commandBuffer := range commandBuffers {
		currentSubCmdID := append(idx, uint64(i))
		if !disablerTransform.doesContainDisabledCmd(currentSubCmdID) {
			newCommandBuffers = append(newCommandBuffers, commandBuffer)
			continue
		}

		existingCommandBufferObject := GetState(inputState).CommandBuffers().Get(commandBuffer)
		newCommandBuffer, err := disablerTransform.rewriteCommandBuffer(ctx, currentSubCmdID, existingCommandBufferObject, cmd, inputState)
		if err != nil {
			log.E(ctx, "Failed during rewriting command buffer : %v", err)
			return VkSubmitInfo{}, err
		}

		newCommandBuffers = append(newCommandBuffers, newCommandBuffer)
	}

	if len(newCommandBuffers) != len(commandBuffers) {
		return VkSubmitInfo{}, fmt.Errorf("Number of command buffers changed")
	}

	newCbs := disablerTransform.mustAllocReadDataForSubmit(ctx, inputState, newCommandBuffers)

	newSubmitInfo := MakeVkSubmitInfo(inputState.Arena)
	newSubmitInfo.SetSType(submitInfo.SType())
	newSubmitInfo.SetPNext(submitInfo.PNext())
	newSubmitInfo.SetWaitSemaphoreCount(submitInfo.WaitSemaphoreCount())
	newSubmitInfo.SetPWaitSemaphores(submitInfo.PWaitSemaphores())
	newSubmitInfo.SetPWaitDstStageMask(submitInfo.PWaitDstStageMask())
	newSubmitInfo.SetCommandBufferCount(submitInfo.CommandBufferCount())
	newSubmitInfo.SetPCommandBuffers(NewVkCommandBufferᶜᵖ(newCbs.Ptr()))
	newSubmitInfo.SetSignalSemaphoreCount(submitInfo.SignalSemaphoreCount())
	newSubmitInfo.SetPSignalSemaphores(submitInfo.PSignalSemaphores())
	return newSubmitInfo, nil
}

func (disablerTransform *commandDisabler) rewriteCommandBuffer(ctx context.Context, idx api.SubCmdIdx, existingCommandBuffer CommandBufferObjectʳ, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandBuffer, error) {
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	st := GetState(inputState)

	newCmdBuffer, err := disablerTransform.getNewCommandBufferAndBegin(ctx, cmd, inputState)
	if err != nil {
		log.E(ctx, "Command buffer could not be created : %v", err)
		return VkCommandBuffer(0), err
	}

	for i := 0; i < existingCommandBuffer.CommandReferences().Len(); i++ {
		currentSubCmdID := append(idx, uint64(i))

		currentCmd := existingCommandBuffer.CommandReferences().Get(uint32(i))
		args := GetCommandArgs(ctx, currentCmd, st)

		if !disablerTransform.shouldBeDisabled(currentSubCmdID) {
			cleanup, newCmd, err := AddCommand(ctx, cb, newCmdBuffer, inputState, inputState, args)
			if err != nil {
				panic(fmt.Errorf("Invalid command-buffer detected %+v", err))
			}
			if err := disablerTransform.observeAndWriteCommand(newCmd); err != nil {
				log.E(ctx, "Failed during adding command : [%v]%v", newCmd, err)
				return VkCommandBuffer(0), err
			}
			cleanup()
			continue
		}

		if !isCmdAllowedToDisable(args) {
			return VkCommandBuffer(0), fmt.Errorf("Command type is not allowed to be disabled : %v", args)
		}

		disablerTransform.removeFromDisabledList(currentSubCmdID)
		log.I(ctx, "Command %v disabled", currentSubCmdID)
	}

	if err := disablerTransform.observeAndWriteCommand(cb.VkEndCommandBuffer(newCmdBuffer, VkResult_VK_SUCCESS)); err != nil {
		log.E(ctx, "Failed during writing EndCommandBuffer : %v", err)
		return VkCommandBuffer(0), err
	}

	return newCmdBuffer, nil
}

func (disablerTransform *commandDisabler) getNewCommandBufferAndBegin(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandBuffer, error) {
	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	queue := GetState(inputState).Queues().Get(cmd.Queue())

	commandPoolID, err := disablerTransform.getNewCommandPool(ctx, cmd, inputState)
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
		disablerTransform.mustAllocReadDataForCmd(ctx, inputState, commandBufferAllocateInfo).Ptr(),
		disablerTransform.mustAllocWriteDataForCmd(ctx, inputState, commandBufferID).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if err = disablerTransform.observeAndWriteCommand(allocateCmd); err != nil {
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
		disablerTransform.mustAllocReadDataForCmd(ctx, inputState, beginInfo).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if err = disablerTransform.observeAndWriteCommand(beginCommandBufferCmd); err != nil {
		log.E(ctx, "Failed during begin command buffer : %v", err)
		return VkCommandBuffer(0), err
	}
	return commandBufferID, nil
}

func (disablerTransform *commandDisabler) getNewCommandPool(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandPool, error) {
	if disablerTransform.pool != 0 {
		return disablerTransform.pool, nil
	}

	cb := CommandBuilder{Thread: cmd.Thread(), Arena: inputState.Arena}
	queue := GetState(inputState).Queues().Get(cmd.Queue())

	disablerTransform.pool = VkCommandPool(newUnusedID(false, func(x uint64) bool {
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
		disablerTransform.mustAllocReadDataForCmd(ctx, inputState, poolCreateInfo).Ptr(),
		memory.Nullptr,
		disablerTransform.mustAllocWriteDataForCmd(ctx, inputState, disablerTransform.pool).Ptr(),
		VkResult_VK_SUCCESS,
	)

	if err := disablerTransform.observeAndWriteCommand(newCmd); err != nil {
		log.E(ctx, "Failed during creating command pool : %v", err)
		return VkCommandPool(0), err
	}
	return disablerTransform.pool, nil
}

func (disablerTransform *commandDisabler) mustAllocReadDataForSubmit(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := disablerTransform.allocations.AllocDataOrPanic(ctx, v...)
	disablerTransform.readMemoriesForSubmit = append(disablerTransform.readMemoriesForSubmit, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (disablerTransform *commandDisabler) mustAllocReadDataForCmd(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := disablerTransform.allocations.AllocDataOrPanic(ctx, v...)
	disablerTransform.readMemoriesForCmd = append(disablerTransform.readMemoriesForCmd, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (disablerTransform *commandDisabler) mustAllocWriteDataForCmd(ctx context.Context, g *api.GlobalState, v ...interface{}) api.AllocResult {
	allocateResult := disablerTransform.allocations.AllocDataOrPanic(ctx, v...)
	disablerTransform.writeMemoriesForCmd = append(disablerTransform.writeMemoriesForCmd, &allocateResult)
	return allocateResult
}

func (disablerTransform *commandDisabler) writeCommand(cmd api.Cmd) error {
	return disablerTransform.stateMutator([]api.Cmd{cmd})
}

func (disablerTransform *commandDisabler) observeAndWriteCommand(cmd api.Cmd) error {
	for _, mem := range disablerTransform.readMemoriesForCmd {
		cmd.Extras().GetOrAppendObservations().AddRead(mem.Data())
	}
	for _, mem := range disablerTransform.writeMemoriesForCmd {
		cmd.Extras().GetOrAppendObservations().AddWrite(mem.Data())
	}
	disablerTransform.readMemoriesForCmd = []*api.AllocResult{}
	disablerTransform.writeMemoriesForCmd = []*api.AllocResult{}

	return disablerTransform.writeCommand(cmd)
}

func (disablerTransform *commandDisabler) doesContainDisabledCmd(id api.SubCmdIdx) bool {
	for _, r := range disablerTransform.disabledCommands {
		if id.Contains(r) {
			return true
		}
	}

	return false
}

func (disablerTransform *commandDisabler) shouldBeDisabled(id api.SubCmdIdx) bool {
	for _, r := range disablerTransform.disabledCommands {
		if id.Equals(r) {
			return true
		}
	}

	return false
}

func (disablerTransform *commandDisabler) removeFromDisabledList(id api.SubCmdIdx) {
	for i, r := range disablerTransform.disabledCommands {
		if id.Equals(r) {
			disablerTransform.disabledCommands[i] =
				disablerTransform.disabledCommands[len(disablerTransform.disabledCommands)-1]
			disablerTransform.disabledCommands =
				disablerTransform.disabledCommands[:len(disablerTransform.disabledCommands)-1]
		}
	}
}

func isCmdAllowedToDisable(commandArgs interface{}) bool {
	switch commandArgs.(type) {
	case VkCmdDrawArgsʳ:
		return true
	case VkCmdDrawIndexedArgsʳ:
		return true
	case VkCmdDrawIndexedIndirectArgsʳ:
		return true
	case VkCmdDrawIndirectArgsʳ:
		return true
	default:
		return false
	}
}
