// Copyright (C) 2019 Google Inc.
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
	"fmt"
	"strings"

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/memory"
)

type queueCommandHandlerState int

const (
	recordingState queueCommandHandlerState = iota
	submittedState
)

type queueCommandHandler struct {
	state                    queueCommandHandlerState
	queue                    VkQueue
	commandBuffer            VkCommandBuffer
	postExecuted             []func()
	dependentFlushablePieces []flushablePiece
}

func newQueueCommandHandler(sb *stateBuilder, queue VkQueue, commandBuffer VkCommandBuffer) (*queueCommandHandler, error) {
	handler := &queueCommandHandler{
		state:         recordingState,
		queue:         queue,
		commandBuffer: commandBuffer,
		postExecuted:  []func(){},
	}
	cmdBufObj := GetState(sb.newState).CommandBuffers().Get(commandBuffer)
	if cmdBufObj.IsNil() {
		return nil, fmt.Errorf("Command buffer: %v not found in the new state of stateBuilder", commandBuffer)
	}
	cmdPoolObj := GetState(sb.newState).CommandPools().Get(cmdBufObj.Pool())
	if cmdPoolObj.QueueFamilyIndex() != handler.QueueFamily(sb) {
		return nil, fmt.Errorf("Command buffer's queue family index: %v does not match with queue's family index: %v",
			cmdPoolObj.QueueFamilyIndex(), handler.QueueFamily(sb))
	}
	return handler, nil
}

func (h *queueCommandHandler) RecordCommands(sb *stateBuilder, name debugMarkerName, f ...func(VkCommandBuffer)) error {
	if h.state != recordingState {
		return fmt.Errorf("queue command handler is not in recording state")
	}
	cmdBufObj := GetState(sb.newState).CommandBuffers().Get(h.commandBuffer)
	if cmdBufObj.Recording() != RecordingState_RECORDING {
		sb.write(sb.cb.VkBeginCommandBuffer(
			h.commandBuffer,
			sb.MustAllocReadData(NewVkCommandBufferBeginInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
				0, // pNext
				VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
				0, // pInheritanceInfo
			)).Ptr(),
			VkResult_VK_SUCCESS,
		))
	}

	calls := f
	if len(name.String()) > 0 {
		calls = make([]func(VkCommandBuffer), 0, len(f)+2)
		calls = append(calls, func(cb VkCommandBuffer) {
			sb.write(sb.cb.VkCmdDebugMarkerBeginEXT(
				cb,
				sb.MustAllocReadData(NewVkDebugMarkerMarkerInfoEXT(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_MARKER_INFO_EXT, // sType
					0, // pNext
					NewCharᶜᵖ(sb.MustAllocReadData(name.String()).Ptr()), // pMarkerName
					NewF32ː4ᵃ(sb.ta), // color[4]
				)).Ptr(),
			))
		})
		calls = append(calls, f...)
		calls = append(calls, func(cb VkCommandBuffer) {
			sb.write(sb.cb.VkCmdDebugMarkerEndEXT(cb))
		})
	}

	for _, c := range calls {
		c(h.commandBuffer)
	}
	return nil
}

func (h *queueCommandHandler) Submit(sb *stateBuilder) error {
	for _, p := range h.dependentFlushablePieces {
		if !p.IsValid() {
			return fmt.Errorf("dependent piece: %v not valid to use", p)
		}
	}
	if h.state != recordingState {
		return fmt.Errorf("queue command handler is not in recording state")
	}
	cmdBufObj := GetState(sb.newState).CommandBuffers().Get(h.commandBuffer)
	if cmdBufObj.Recording() == RecordingState_RECORDING {
		sb.write(sb.cb.VkEndCommandBuffer(h.commandBuffer, VkResult_VK_SUCCESS))
		sb.write(sb.cb.VkQueueSubmit(
			h.queue,
			1,
			sb.MustAllocReadData(NewVkSubmitInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
				0, // pNext
				0, // waitSemaphoreCount
				0, // pWaitSemaphores
				0, // pWaitDstStageMask
				1, // commandBufferCount
				NewVkCommandBufferᶜᵖ(sb.MustAllocReadData(h.commandBuffer).Ptr()), // pCommandBuffers
				0, // signalSemaphoreCount
				0, // pSignalSemaphores
			)).Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		))
	}
	h.state = submittedState
	return nil
}

func (h *queueCommandHandler) WaitUntilFinish(sb *stateBuilder) error {
	for _, p := range h.dependentFlushablePieces {
		if !p.IsValid() {
			return fmt.Errorf("dependent piece: %v not valid to use", p)
		}
	}
	if h.state != submittedState {
		return fmt.Errorf("queue command handler is not in submitted state")
	}
	sb.write(sb.cb.VkQueueWaitIdle(h.queue, VkResult_VK_SUCCESS))
	sb.write(sb.cb.VkResetCommandBuffer(
		h.commandBuffer,
		VkCommandBufferResetFlags(VkCommandBufferResetFlagBits_VK_COMMAND_BUFFER_RESET_RELEASE_RESOURCES_BIT),
		VkResult_VK_SUCCESS,
	))
	for i := len(h.postExecuted) - 1; i >= 0; i-- {
		h.postExecuted[i]()
	}
	h.dependentFlushablePieces = []flushablePiece{}
	h.postExecuted = nil
	h.state = recordingState
	return nil
}

func (h *queueCommandHandler) RecordPostExecuted(f ...func()) error {
	if h.state != recordingState {
		return fmt.Errorf("queue command handler is not in recording state")
	}
	h.postExecuted = append(h.postExecuted, f...)
	return nil
}

func (h *queueCommandHandler) Device(sb *stateBuilder) VkDevice {
	return GetState(sb.newState).Queues().Get(h.queue).Device()
}

func (h *queueCommandHandler) Queue() VkQueue {
	return h.queue
}

func (h *queueCommandHandler) QueueFamily(sb *stateBuilder) uint32 {
	return GetState(sb.newState).Queues().Get(h.queue).Family()
}

// OnMemoryFlush implements the flushable resource user interface
func (h *queueCommandHandler) OnResourceFlush(sb *stateBuilder, res flushableResource) {
	err := h.Submit(sb)
	if err != nil {
		panic(err)
	}
	err = h.WaitUntilFinish(sb)
	if err != nil {
		panic(err)
	}
}

func (h *queueCommandHandler) AddDependentFlushablePieces(p ...flushablePiece) {
	for _, fp := range p {
		fp.Owner().AddUser(h)
		h.RecordPostExecuted(func() { fp.Owner().DropUser(h) })
	}
	h.dependentFlushablePieces = append(h.dependentFlushablePieces, p...)
}

type queueCommandBatch struct {
	name           debugMarkerName
	scratchBuffers map[*flushingMemory][]bufferFlushInfo
	onCommit       []func(*queueCommandHandler)
	records        []func(VkCommandBuffer)
	postExecuted   []func()
}

func newQueueCommandBatch(name string) *queueCommandBatch {
	return &queueCommandBatch{
		name:           debugMarkerName(name),
		scratchBuffers: map[*flushingMemory][]bufferFlushInfo{},
		onCommit:       []func(*queueCommandHandler){},
		records:        []func(VkCommandBuffer){},
		postExecuted:   []func(){},
	}
}

func (qcb *queueCommandBatch) NewScratchBuffer(sb *stateBuilder, name debugMarkerName, mem *flushingMemory, dev VkDevice, usages VkBufferUsageFlags, data ...hashedDataAndOffset) VkBuffer {
	size := uint64(0)
	for _, d := range data {
		if d.offset+d.data.size > size {
			size = d.offset + d.data.size
		}
	}
	size = nextMultipleOf(size, scratchBufferAlignment)
	buffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
		return sb.s.Buffers().Contains(VkBuffer(x)) || GetState(sb.newState).Buffers().Contains(VkBuffer(x))
	}))
	usages = VkBufferUsageFlags(uint32(usages) | uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT))
	sb.write(sb.cb.VkCreateBuffer(
		dev,
		sb.MustAllocReadData(
			NewVkBufferCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
				0,                                       // pNext
				0,                                       // flags
				VkDeviceSize(size),                      // size
				usages,                                  // usage
				VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
				0,                                       // queueFamilyIndexCount
				0,                                       // pQueueFamilyIndices
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(buffer).Ptr(),
		VkResult_VK_SUCCESS,
	))
	if len(qcb.name.String()) > 0 || len(name.String()) > 0 {
		sb.write(sb.cb.VkDebugMarkerSetObjectNameEXT(
			dev,
			NewVkDebugMarkerObjectNameInfoEXTᵖ(sb.MustAllocReadData(
				NewVkDebugMarkerObjectNameInfoEXT(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_DEBUG_MARKER_OBJECT_NAME_INFO_EXT, // sType
					0, // pNext
					VkDebugReportObjectTypeEXT_VK_DEBUG_REPORT_OBJECT_TYPE_BUFFER_EXT, // objectType
					uint64(buffer), // object
					NewCharᶜᵖ(sb.MustAllocReadData(qcb.name.Child(name.String()).String()).Ptr()), // pObjectName
				)).Ptr(),
			),
			VkResult_VK_SUCCESS,
		))
	}
	flush := bufferFlushInfo{buffer: buffer, dataSlices: data}
	if _, ok := qcb.scratchBuffers[mem]; !ok {
		qcb.scratchBuffers[mem] = []bufferFlushInfo{}
	}
	qcb.scratchBuffers[mem] = append(qcb.scratchBuffers[mem], flush)
	// qcb.DoOnCommit(func(handler *queueCommandHandler) {
	// 	mem.AddUser(handler)
	// 	handler.RecordPostExecuted(func() {
	// 		mem.DropUser(handler)
	// 	})
	// })
	qcb.DeferToPostExecuted(func() {
		sb.write(sb.cb.VkDestroyBuffer(dev, buffer, memory.Nullptr))
	})
	return buffer
}

func (qcb *queueCommandBatch) DoOnCommit(f ...func(handler *queueCommandHandler)) {
	qcb.onCommit = append(qcb.onCommit, f...)
}

func (qcb *queueCommandBatch) DeferToPostExecuted(f ...func()) {
	qcb.postExecuted = append(qcb.postExecuted, f...)
}

func (qcb *queueCommandBatch) RecordCommandsOnCommit(f ...func(VkCommandBuffer)) {
	qcb.records = append(qcb.records, f...)
}

func (qcb *queueCommandBatch) Commit(sb *stateBuilder, handler *queueCommandHandler) error {
	for mem, bufs := range qcb.scratchBuffers {
		relativeOffsets := []uint64{}
		currentOffset := uint64(0)
		for _, info := range bufs {
			bufObj := GetState(sb.newState).Buffers().Get(info.buffer)
			size := uint64(bufObj.Info().Size())
			allocSize := bufferAllocationSize(size, scratchBufferAlignment)
			relativeOffsets = append(relativeOffsets, currentOffset)
			currentOffset += allocSize
		}
		allocationResult, err := mem.Allocate(sb, currentOffset)
		if err != nil {
			return log.Errf(sb.ctx, err, "failed to allocate memory for scratch buffers")
		}
		handler.AddDependentFlushablePieces(allocationResult)
		vulkanMem := allocationResult.Memory()
		globalOffset := allocationResult.Offset()
		for i, info := range bufs {
			bufObj := GetState(sb.newState).Buffers().Get(info.buffer)
			sb.write(sb.cb.VkBindBufferMemory(
				bufObj.Device(),
				info.buffer,
				vulkanMem,
				VkDeviceSize(globalOffset+relativeOffsets[i]),
				VkResult_VK_SUCCESS,
			))
		}
		if err := flushDataToBuffers(sb, scratchBufferAlignment, bufs...); err != nil {
			return log.Errf(sb.ctx, err, "failed at flushing data to scratch buffers")
		}
	}
	for _, f := range qcb.onCommit {
		f(handler)
	}
	var err error
	err = handler.RecordCommands(sb, qcb.name, qcb.records...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed to record commands to queue command handler")
	}
	err = handler.RecordPostExecuted(qcb.postExecuted...)
	if err != nil {
		return log.Errf(sb.ctx, err, "failed to record post executed callbacks to queue command handler")
	}
	return nil
}

type debugMarkerName string

func (n debugMarkerName) Child(s string) debugMarkerName {
	l := []string{n.String(), s}
	return debugMarkerName(strings.Join(l, " "))
}

func (n debugMarkerName) String() string {
	return string(n)
}
