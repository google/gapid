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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/memory"
)

type commandBuilderUtil struct {
	allocations   *allocationTracker
	readMemories  []*api.AllocResult
	writeMemories []*api.AllocResult
}

func newCommandBuilderUtil(allocations *allocationTracker) *commandBuilderUtil {
	return &commandBuilderUtil{
		allocations:   allocations,
		readMemories:  make([]*api.AllocResult, 0),
		writeMemories: make([]*api.AllocResult, 0),
	}
}

func (util *commandBuilderUtil) allocReadData(ctx context.Context, g *api.GlobalState, value ...interface{}) api.AllocResult {
	allocateResult := util.allocations.AllocDataOrPanic(ctx, value...)
	util.readMemories = append(util.readMemories, &allocateResult)
	rng, id := allocateResult.Data()
	g.Memory.ApplicationPool().Write(rng.Base, memory.Resource(id, rng.Size))
	return allocateResult
}

func (util *commandBuilderUtil) allocWriteData(ctx context.Context, v ...interface{}) api.AllocResult {
	allocateResult := util.allocations.AllocDataOrPanic(ctx, v...)
	util.writeMemories = append(util.writeMemories, &allocateResult)
	return allocateResult
}

func (util *commandBuilderUtil) observeNewCommand(cmd api.Cmd) {
	for i := range util.readMemories {
		cmd.Extras().GetOrAppendObservations().AddRead(util.readMemories[i].Data())
	}
	for i := range util.writeMemories {
		cmd.Extras().GetOrAppendObservations().AddWrite(util.writeMemories[i].Data())
	}

	util.readMemories = []*api.AllocResult{}
	util.writeMemories = []*api.AllocResult{}
}

func createNewCommandPool(
	ctx context.Context,
	allocations *allocationTracker,
	inputState *api.GlobalState,
	thread uint64,
	device VkDevice,
	family uint32) (VkCommandPool, api.Cmd) {
	commandPoolID := VkCommandPool(newUnusedID(false, func(x uint64) bool {
		return GetState(inputState).CommandPools().Contains(VkCommandPool(x))
	}))

	poolCreateInfo := NewVkCommandPoolCreateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,                                 // sType
		NewVoidᶜᵖ(memory.Nullptr),                                                                  // pNext
		VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_TRANSIENT_BIT), // flags
		family, // queueFamilyIndex
	)

	util := newCommandBuilderUtil(allocations)
	cb := CommandBuilder{Thread: thread}
	newCmd := cb.VkCreateCommandPool(
		device,
		util.allocReadData(ctx, inputState, poolCreateInfo).Ptr(),
		memory.Nullptr,
		util.allocWriteData(ctx, commandPoolID).Ptr(),
		VkResult_VK_SUCCESS,
	)
	util.observeNewCommand(newCmd)

	return commandPoolID, newCmd
}

func createNewCommandBuffer(
	ctx context.Context,
	allocations *allocationTracker,
	inputState *api.GlobalState,
	thread uint64,
	device VkDevice,
	commandPool VkCommandPool,
	level VkCommandBufferLevel) (VkCommandBuffer, api.Cmd) {
	commandBuffer := VkCommandBuffer(newUnusedID(true, func(x uint64) bool {
		return GetState(inputState).CommandBuffers().Contains(VkCommandBuffer(x))
	}))

	commandBufferAllocateInfo := NewVkCommandBufferAllocateInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                      // pNext
		commandPool,                                                    // commandPool
		level,                                                          // level
		1,                                                              // commandBufferCount
	)

	util := newCommandBuilderUtil(allocations)
	cb := CommandBuilder{Thread: thread}

	newCmd := cb.VkAllocateCommandBuffers(
		device,
		util.allocReadData(ctx, inputState, commandBufferAllocateInfo).Ptr(),
		util.allocWriteData(ctx, commandBuffer).Ptr(),
		VkResult_VK_SUCCESS,
	)
	util.observeNewCommand(newCmd)

	return commandBuffer, newCmd
}

func beginCommandBuffer(
	ctx context.Context,
	allocations *allocationTracker,
	inputState *api.GlobalState,
	thread uint64,
	commandBuffer VkCommandBuffer) api.Cmd {

	util := newCommandBuilderUtil(allocations)

	beginInfo := NewVkCommandBufferBeginInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		NewVoidᶜᵖ(memory.Nullptr),                                   // pNext
		0,                                                           // flags
		NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr), // pInheritanceInfo
	)

	cb := CommandBuilder{Thread: thread}
	newCmd := cb.VkBeginCommandBuffer(
		commandBuffer,
		util.allocReadData(ctx, inputState, beginInfo).Ptr(),
		VkResult_VK_SUCCESS,
	)
	util.observeNewCommand(newCmd)

	return newCmd
}

func beginCommandBufferFromExistingCommandBuffer(
	ctx context.Context,
	allocations *allocationTracker,
	inputState *api.GlobalState,
	thread uint64,
	commandBufferToBegin VkCommandBuffer,
	referenceCommandBufferObject CommandBufferObjectʳ) api.Cmd {

	util := newCommandBuilderUtil(allocations)

	pNext := NewVoidᶜᵖ(memory.Nullptr)
	if !referenceCommandBufferObject.BeginInfo().DeviceGroupBegin().IsNil() {
		beginInfo := NewVkDeviceGroupCommandBufferBeginInfo(
			VkStructureType_VK_STRUCTURE_TYPE_DEVICE_GROUP_COMMAND_BUFFER_BEGIN_INFO, // sType
			pNext, // pNext
			referenceCommandBufferObject.BeginInfo().DeviceGroupBegin().DeviceMask(), // deviceMask
		)
		beginInfoData := util.allocReadData(ctx, inputState, beginInfo)
		pNext = NewVoidᶜᵖ(beginInfoData.Ptr())
	}

	pInheritenceInfo := NewVkCommandBufferInheritanceInfoᶜᵖ(memory.Nullptr)
	if existingBeginInfo := referenceCommandBufferObject.BeginInfo(); existingBeginInfo.Inherited() {
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

		inheritanceInfoData := util.allocReadData(ctx, inputState, inheritenceInfo)
		pInheritenceInfo = NewVkCommandBufferInheritanceInfoᶜᵖ(inheritanceInfoData.Ptr())
	}

	beginInfo := NewVkCommandBufferBeginInfo(
		VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
		pNext, // pNext
		referenceCommandBufferObject.BeginInfo().Flags(), // flags
		pInheritenceInfo, // pInheritanceInfo
	)

	cb := CommandBuilder{Thread: thread}
	newCmd := cb.VkBeginCommandBuffer(
		commandBufferToBegin,
		util.allocReadData(ctx, inputState, beginInfo).Ptr(),
		VkResult_VK_SUCCESS,
	)
	util.observeNewCommand(newCmd)

	return newCmd
}

func endCommandBuffer(thread uint64, commandBuffer VkCommandBuffer) api.Cmd {
	cb := CommandBuilder{Thread: thread}
	newCmd := cb.VkEndCommandBuffer(commandBuffer, VkResult_VK_SUCCESS)
	return newCmd
}
