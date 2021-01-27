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
	// This should be presorted before mutation starts
	disabledCommands []api.SubCmdIdx
	cmdsOffset       uint64
	mutationStarted  bool

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
		cmdsOffset:            cmdsOffset,
		mutationStarted:       false,
		stateMutator:          nil,
		allocations:           nil,
		readMemoriesForSubmit: make([]*api.AllocResult, 0),
		readMemoriesForCmd:    make([]*api.AllocResult, 0),
		writeMemoriesForCmd:   make([]*api.AllocResult, 0),
		pool:                  VkCommandPool(0),
	}
}

// Remove removes a draw call command from a command buffer.
func (disablerTransform *commandDisabler) remove(ctx context.Context, id api.SubCmdIdx) error {
	if disablerTransform.mutationStarted {
		return log.Err(ctx, nil, "Commands cannot be requested to disable after mutation started")
	}

	if len(id) == 0 {
		return log.Err(ctx, nil, "Requested id is empty")
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
	disablerTransform.sortAndMergeDisabledCmds(ctx)
	disablerTransform.mutationStarted = true
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

	return nil, log.Err(ctx, err, "Command Disabler Transform Error")
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
				return nil, log.Err(ctx, err, "Failed during removing command from VkQueueSubmit")
			}
		} else {
			if err := disablerTransform.writeCommand(cmd); err != nil {
				return nil, log.Err(ctx, err, "Failed during processing input commands")
			}
		}
	}

	if !queueSubmitProcessed {
		return nil, log.Err(ctx, nil, "No queue submit has found in command path")
	}

	return nil, nil
}

func (disablerTransform *commandDisabler) removeCommandFromVkQueueSubmit(ctx context.Context,
	idx api.SubCmdIdx, cmd *VkQueueSubmit, inputState *api.GlobalState) error {
	layout := inputState.MemoryLayout
	cb := CommandBuilder{Thread: cmd.Thread()}
	cmd.Extras().Observations().ApplyReads(inputState.Memory.ApplicationPool())

	submitInfos, err := cmd.PSubmits().Slice(0, uint64(cmd.SubmitCount()), layout).Read(ctx, cmd, inputState, nil)
	if err != nil {
		return err
	}
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
			return log.Err(ctx, err, "Failed during removing command from submit")
		}

		newSubmitInfos = append(newSubmitInfos, newSubmitInfo)
	}

	if len(newSubmitInfos) != len(submitInfos) {
		return log.Err(ctx, nil, "Number of queue submits has changed")
	}

	newSubmit.SetPSubmits(NewVkSubmitInfoᶜᵖ(disablerTransform.mustAllocReadDataForSubmit(ctx, inputState, newSubmitInfos).Ptr()))

	for _, mem := range disablerTransform.readMemoriesForSubmit {
		newSubmit.AddRead(mem.Data())
	}
	disablerTransform.readMemoriesForSubmit = []*api.AllocResult{}

	if err := disablerTransform.writeCommand(newSubmit); err != nil {
		return log.Err(ctx, err, "Failed during writing VkQueueSubmit")
	}

	return nil
}

func (disablerTransform *commandDisabler) removeCommandFromSubmit(ctx context.Context,
	idx api.SubCmdIdx, submitInfo VkSubmitInfo, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkSubmitInfo, error) {
	layout := inputState.MemoryLayout
	// pCommandBuffers
	commandBuffers, err := submitInfo.PCommandBuffers().Slice(0, uint64(submitInfo.CommandBufferCount()), layout).Read(ctx, cmd, inputState, nil)
	if err != nil {
		return VkSubmitInfo{}, err
	}
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
			return VkSubmitInfo{}, log.Err(ctx, err, "Failed during rewriting command buffer")
		}

		newCommandBuffers = append(newCommandBuffers, newCommandBuffer)
	}

	if len(newCommandBuffers) != len(commandBuffers) {
		return VkSubmitInfo{}, log.Err(ctx, nil, "Number of command buffers changed")
	}

	newCbs := disablerTransform.mustAllocReadDataForSubmit(ctx, inputState, newCommandBuffers)

	newSubmitInfo := MakeVkSubmitInfo()
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

func (disablerTransform *commandDisabler) rewriteCommandBuffer(ctx context.Context, idx api.SubCmdIdx,
	existingCommandBuffer CommandBufferObjectʳ, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandBuffer, error) {
	cb := CommandBuilder{Thread: cmd.Thread()}
	st := GetState(inputState)

	newCmdBuffer, err := disablerTransform.getNewCommandBufferAndBegin(ctx, existingCommandBuffer, cmd, inputState)
	if err != nil {
		return VkCommandBuffer(0), log.Err(ctx, err, "Command buffer could not be created")
	}

	for i := 0; i < existingCommandBuffer.CommandReferences().Len(); i++ {
		currentSubCmdID := append(idx, uint64(i))

		currentCmd := existingCommandBuffer.CommandReferences().Get(uint32(i))
		args := GetCommandArgs(ctx, currentCmd, st)

		if disablerTransform.shouldBeDisabled(currentSubCmdID) {
			if !isCmdAllowedToDisable(args) {
				return VkCommandBuffer(0), log.Errf(ctx, nil, "Command type is not allowed to be disabled : %v", args)
			}

			// Skip the disabled command and do not copy it to the new command buffer
			disablerTransform.removeFromDisabledList(currentSubCmdID)
			log.I(ctx, "Command %v disabled", currentSubCmdID)
			continue
		}

		if disablerTransform.doesContainDisabledCmd(currentSubCmdID) {
			// Rewrite the secondary command buffer and create a new VkCmdExecuteCommands
			// to use new command buffer instead the old one
			executeCommandsArgs, ok := args.(VkCmdExecuteCommandsArgsʳ)
			if !ok {
				return VkCommandBuffer(0), log.Errf(ctx, nil, "VkExecuteCommands could not found %v: ", args)
			}

			newCmd, err := disablerTransform.rewriteExecuteSecondaryCommandBuffer(ctx,
				currentSubCmdID, newCmdBuffer, executeCommandsArgs, cmd, inputState)
			if err != nil {
				return VkCommandBuffer(0), log.Err(ctx, err, "Error during rewriting secondary command buffer")
			}
			if err = disablerTransform.observeAndWriteCommand(newCmd); err != nil {
				return VkCommandBuffer(0), log.Errf(ctx, err, "Failed during adding command : [%v]", newCmd)
			}
			log.I(ctx, "Secondary Command Buffer %v updated", currentSubCmdID)
			continue
		}

		// Copy the unaffected commands to the new command buffer
		cleanup, newCmd, err := AddCommand(ctx, cb, newCmdBuffer, inputState, inputState, args)
		if err != nil {
			panic(fmt.Errorf("Cannot create copying command %+v", err))
		}
		if err = disablerTransform.observeAndWriteCommand(newCmd); err != nil {
			return VkCommandBuffer(0), log.Errf(ctx, err, "Failed during adding command : [%v]", newCmd)
		}
		cleanup()
	}

	if err := disablerTransform.observeAndWriteCommand(cb.VkEndCommandBuffer(newCmdBuffer, VkResult_VK_SUCCESS)); err != nil {
		return VkCommandBuffer(0), log.Err(ctx, err, "Failed during writing EndCommandBuffer")
	}

	return newCmdBuffer, nil
}

func (disablerTransform *commandDisabler) rewriteExecuteSecondaryCommandBuffer(ctx context.Context,
	idx api.SubCmdIdx, primaryCommandBuffer VkCommandBuffer, args VkCmdExecuteCommandsArgsʳ,
	cmd *VkQueueSubmit, inputState *api.GlobalState) (api.Cmd, error) {

	existingCmdBufferCount := uint32(args.CommandBuffers().Len())
	newCommandBuffers := make([]VkCommandBuffer, 0, existingCmdBufferCount)

	for i := uint32(0); i < existingCmdBufferCount; i++ {
		currentSubCmdID := append(idx, uint64(i))
		existingCommandBuffer := args.CommandBuffers().Get(i)
		if !disablerTransform.doesContainDisabledCmd(currentSubCmdID) {
			newCommandBuffers = append(newCommandBuffers, existingCommandBuffer)
			continue
		}

		existingCommandBufferObject := GetState(inputState).CommandBuffers().Get(existingCommandBuffer)
		newCommandBuffer, err := disablerTransform.rewriteCommandBuffer(ctx, currentSubCmdID, existingCommandBufferObject, cmd, inputState)
		if err != nil {
			return nil, log.Err(ctx, err, "Error during writing secondary command buffer")
		}

		newCommandBuffers = append(newCommandBuffers, newCommandBuffer)
	}

	if uint32(len(newCommandBuffers)) != existingCmdBufferCount {
		return nil, log.Err(ctx, nil, "Number of command buffers changed")
	}

	cb := CommandBuilder{Thread: cmd.Thread()}
	commandBuffersMemory := disablerTransform.mustAllocReadDataForCmd(ctx, inputState, newCommandBuffers)
	newCmd := cb.VkCmdExecuteCommands(primaryCommandBuffer, existingCmdBufferCount, commandBuffersMemory.Ptr())
	return newCmd, nil
}

func (disablerTransform *commandDisabler) getNewCommandBufferAndBegin(ctx context.Context, existingCommandBuffer CommandBufferObjectʳ, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandBuffer, error) {
	cb := CommandBuilder{Thread: cmd.Thread()}
	queue := GetState(inputState).Queues().Get(cmd.Queue())

	commandPoolID, err := disablerTransform.getNewCommandPool(ctx, cmd, inputState)
	if err != nil {
		return VkCommandBuffer(0), log.Err(ctx, err, "Failed during getting command pool")
	}

	commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		commandPoolID,                                                  // commandPool
		existingCommandBuffer.Level(),                                  // level
		1,                                                              // commandBufferCount
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
		return VkCommandBuffer(0), log.Err(ctx, err, "Failed during allocating command buffer")
	}

	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !existingCommandBuffer.BeginInfo().DeviceGroupBegin().IsNil() {
		beginInfo := NewVkDeviceGroupCommandBufferBeginInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DEVICE_GROUP_COMMAND_BUFFER_BEGIN_INFO, // sType
			pNext, // pNext
			existingCommandBuffer.BeginInfo().DeviceGroupBegin().DeviceMask(), // deviceMask
		)
		beginInfoData := disablerTransform.mustAllocReadDataForCmd(ctx, inputState, beginInfo)
		pNext = NewVoidᶜᵖ(beginInfoData.Ptr())
	}

	pInheritenceInfo := NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr)
	if existingBeginInfo := existingCommandBuffer.BeginInfo(); existingBeginInfo.Inherited() {
		inheritenceInfo := NewVkCommandBufferInheritanceInfo(
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_INHERITANCE_INFO,
			NewVoidᶜᵖ(memory.Nullptr),
			existingBeginInfo.InheritedRenderPass(),
			existingBeginInfo.InheritedSubpass(),
			existingBeginInfo.InheritedFramebuffer(),
			existingBeginInfo.InheritedOcclusionQuery(),
			existingBeginInfo.InheritedQueryFlags(),
			existingBeginInfo.InheritedPipelineStatsFlags(),
		)

		inheritanceInfoData := disablerTransform.mustAllocReadDataForCmd(ctx, inputState, inheritenceInfo)
		pInheritenceInfo = NewVkCommandBufferInheritanceInfoᶜᵖ(inheritanceInfoData.Ptr())
	}

	beginInfo := NewVkCommandBufferBeginInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		pNext, // pNext
		existingCommandBuffer.BeginInfo().Flags(), // flags
		pInheritenceInfo, // pInheritanceInfo
	)

	beginCommandBufferCmd := cb.VkBeginCommandBuffer(
		commandBufferID,
		disablerTransform.mustAllocReadDataForCmd(ctx, inputState, beginInfo).Ptr(),
		VkResult_VK_SUCCESS,
	)
	if err = disablerTransform.observeAndWriteCommand(beginCommandBufferCmd); err != nil {
		return VkCommandBuffer(0), log.Err(ctx, err, "Failed during begin command buffer")
	}
	return commandBufferID, nil
}

func (disablerTransform *commandDisabler) getNewCommandPool(ctx context.Context, cmd *VkQueueSubmit, inputState *api.GlobalState) (VkCommandPool, error) {
	if disablerTransform.pool != 0 {
		return disablerTransform.pool, nil
	}

	cb := CommandBuilder{Thread: cmd.Thread()}
	queue := GetState(inputState).Queues().Get(cmd.Queue())

	disablerTransform.pool = VkCommandPool(newUnusedID(false, func(x uint64) bool {
		return GetState(inputState).CommandPools().Contains(VkCommandPool(x))
	}))

	poolCreateInfo := NewVkCommandPoolCreateInfo(
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
		return VkCommandPool(0), log.Err(ctx, err, "Failed during creating command pool")
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

func (disablerTransform *commandDisabler) sortAndMergeDisabledCmds(ctx context.Context) {
	/*
		This function ensures these conditions:
			- To guarantee the command indices to be valid, all operations has to be done to the
			original command buffers in order and one pass. In that way, we can guarantee
			that the command order in command buffers are untouched and as it is in the trace.

			- For this input set: "[3893.0.6.3.0.13], [3893.0.6.3.0.17], [3893.0.6.3.0.22], [3893.0.6.3]"
			All the subcommands of [3893.0.6.3] are eliminated as they are not necessary.

			- Removes all duplicates.
	*/

	api.SortSubCmdIDs(disablerTransform.disabledCommands)

	size := len(disablerTransform.disabledCommands)
	ids := make([]api.SubCmdIdx, 0, size)
	ids = append(ids, disablerTransform.disabledCommands[size-1])

	// Remove the duplicates and clean the subcommands of deleted subcommands
	for i := int64(size - 2); i >= 0; i-- {
		if ids[len(ids)-1].Contains(disablerTransform.disabledCommands[i]) {
			continue
		}
		ids = append(ids, disablerTransform.disabledCommands[i])
	}

	api.ReverseSubCmdIDs(ids)
	disablerTransform.disabledCommands = ids
}

func (disablerTransform *commandDisabler) doesContainDisabledCmd(id api.SubCmdIdx) bool {
	if len(disablerTransform.disabledCommands) == 0 {
		return false
	}

	// Disabled commands are sorted and as we traverse in our command tree in order,
	// if we cannot find it in first element, it will not be there after
	return id.Contains(disablerTransform.disabledCommands[0])
}

func (disablerTransform *commandDisabler) shouldBeDisabled(id api.SubCmdIdx) bool {
	if len(disablerTransform.disabledCommands) == 0 {
		return false
	}

	// Disabled commands are sorted and as we traverse in our command tree in order,
	// if we cannot find it in first element, it will not be there after
	return id.Equals(disablerTransform.disabledCommands[0])
}

func (disablerTransform *commandDisabler) removeFromDisabledList(id api.SubCmdIdx) {
	if len(disablerTransform.disabledCommands) == 0 {
		panic("Disabled Command list is empty, this should not happen")
	}

	// Disabled commands are sorted and as we traverse in our command tree in order,
	// if we cannot find it in first element, it will not be there after
	if !id.Equals(disablerTransform.disabledCommands[0]) {
		panic("Disabled command is not the first element, this should not happen")
	}

	disablerTransform.disabledCommands = disablerTransform.disabledCommands[1:len(disablerTransform.disabledCommands)]
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
	case VkCmdExecuteCommandsArgsʳ:
		return true
	default:
		return false
	}
}
