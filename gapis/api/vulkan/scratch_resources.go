// Copyright (C) 2018 Google Inc.
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
	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

const (
	scratchBufferSize = uint64(64 * 1024 * 1024)
)

// queueFamilyScratchResources holds the scratch resources for a queue family.
// It manages the creation/destroy of a command pool, a fixed-size memory,
// command buffers for each queue of this family, the usage of the fixed-size
// memory and the submission of the commands buffers.
type queueFamilyScratchResources struct {
	sb             *stateBuilder
	device         VkDevice
	queueFamily    uint32
	commandPool    VkCommandPool
	commandBuffers map[VkQueue]VkCommandBuffer
	memory         VkDeviceMemory
	memorySize     uint64
	allocated      uint64
	postExecuted   map[VkQueue][]func()
}

// getQueueFamilyScratchResources returns the scratch resources for the family
// of the given queue. If such a queeuFamilyScratchResources does not exist,
// it will create one and return it.
func (sb *stateBuilder) getQueueFamilyScratchResources(queue VkQueue) *queueFamilyScratchResources {
	dev := sb.s.Queues().Get(queue).Device()
	family := sb.s.Queues().Get(queue).Family()
	if _, ok := sb.scratchResources[dev]; !ok {
		sb.scratchResources[dev] = map[uint32]*queueFamilyScratchResources{}
	}
	if _, ok := sb.scratchResources[dev][family]; !ok {
		sb.scratchResources[dev][family] = &queueFamilyScratchResources{
			sb:             sb,
			device:         dev,
			queueFamily:    family,
			commandPool:    VkCommandPool(0),
			commandBuffers: map[VkQueue]VkCommandBuffer{},
			memory:         VkDeviceMemory(0),
			memorySize:     bufferAllocationSize(scratchBufferSize),
			allocated:      uint64(0),
			postExecuted:   map[VkQueue][]func(){},
		}
	}
	return sb.scratchResources[dev][family]
}

// flushAllScratchResources submits all the comamnd buffers of all the queue
// family scratch resources, and calls all the after-executed callbacks.
func (sb *stateBuilder) flushAllScratchResources() {
	for _, familyInfo := range sb.scratchResources {
		for _, qr := range familyInfo {
			qr.flush()
		}
	}
}

// freeAllScratchResources frees all the command pool, memory, etc of all the
// queue family scratch resources.
func (sb *stateBuilder) freeAllScratchResources() {
	for _, familyInfo := range sb.scratchResources {
		for _, qr := range familyInfo {
			qr.free()
		}
	}
}

// flushQueueFamilyScratchResources submits all the command buffers of the
// scratch resources of the given queue's family, and also calls all the
// after-executed callbacks registered on that queue family.
func (sb *stateBuilder) flushQueueFamilyScratchResources(queue VkQueue) {
	qr := sb.getQueueFamilyScratchResources(queue)
	qr.flush()
}

// getCommandPool returns the scratch command pool of this queue family
// scratch resource, creates one if it does not exist before.
func (qr *queueFamilyScratchResources) getCommandPool() VkCommandPool {
	if qr.commandPool != VkCommandPool(0) {
		return qr.commandPool
	}
	sb := qr.sb
	commandPoolID := VkCommandPool(newUnusedID(true, func(x uint64) bool {
		return sb.s.CommandPools().Contains(VkCommandPool(x)) || GetState(sb.newState).CommandPools().Contains(VkCommandPool(x))
	}))
	sb.write(sb.cb.VkCreateCommandPool(
		qr.device,
		sb.MustAllocReadData(NewVkCommandPoolCreateInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, // sType
			0, // pNext
			VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT), // flags
			qr.queueFamily, // queueFamilyIndex
		)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(commandPoolID).Ptr(),
		VkResult_VK_SUCCESS,
	))
	qr.commandPool = commandPoolID
	return qr.commandPool
}

// getCommandPool returns the scratch command buffer for the given queue,
// creates one if it does not exist before.
func (qr *queueFamilyScratchResources) getCommandBuffer(queue VkQueue) VkCommandBuffer {
	sb := qr.sb
	commandPool := qr.getCommandPool()
	if _, ok := qr.commandBuffers[queue]; !ok {
		commandBufferID := VkCommandBuffer(newUnusedID(true, func(x uint64) bool {
			return sb.s.CommandBuffers().Contains(VkCommandBuffer(x)) || GetState(sb.newState).CommandBuffers().Contains(VkCommandBuffer(x))
		}))
		sb.write(sb.cb.VkAllocateCommandBuffers(
			qr.device,
			sb.MustAllocReadData(NewVkCommandBufferAllocateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
				0,           // pNext
				commandPool, // commandPool
				VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY, // level
				uint32(1), // commandBufferCount
			)).Ptr(),
			sb.MustAllocWriteData(commandBufferID).Ptr(),
			VkResult_VK_SUCCESS,
		))
		qr.commandBuffers[queue] = commandBufferID
	}
	commandBuffer := qr.commandBuffers[queue]
	if GetState(sb.newState).CommandBuffers().Get(commandBuffer).Recording() != RecordingState_RECORDING {
		sb.write(sb.cb.VkBeginCommandBuffer(
			commandBuffer,
			sb.MustAllocReadData(NewVkCommandBufferBeginInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO, // sType
				0, // pNext
				VkCommandBufferUsageFlags(VkCommandBufferUsageFlagBits_VK_COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT), // flags
				0, // pInheritanceInfo
			)).Ptr(),
			VkResult_VK_SUCCESS,
		))
	}
	return commandBuffer
}

// getDeviceMemory returns the fixed-size scratch memory of this scratch
// resource, creates one if it does not exist before.
func (qr *queueFamilyScratchResources) getDeviceMemory() VkDeviceMemory {
	if qr.memory == VkDeviceMemory(0) {
		qr.memory = qr.newDeviceMemory(qr.memorySize)
		qr.allocated = uint64(0)
	}
	return qr.memory
}

// newDeviceMemory creates a device memory with the given size.
func (qr *queueFamilyScratchResources) newDeviceMemory(size uint64) VkDeviceMemory {
	sb := qr.sb
	dev := qr.device
	deviceMemory := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
		return sb.s.DeviceMemories().Contains(VkDeviceMemory(x)) || GetState(sb.newState).DeviceMemories().Contains(VkDeviceMemory(x))
	}))
	memoryTypeIndex := sb.GetScratchBufferMemoryIndex(sb.s.Devices().Get(dev))
	size = nextMultipleOf(size, 256)
	sb.write(sb.cb.VkAllocateMemory(
		dev,
		NewVkMemoryAllocateInfoᶜᵖ(sb.MustAllocReadData(
			NewVkMemoryAllocateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
				0,                  // pNext
				VkDeviceSize(size), // allocationSize
				memoryTypeIndex,    // memoryTypeIndex
			)).Ptr()),
		memory.Nullptr,
		sb.MustAllocWriteData(deviceMemory).Ptr(),
		VkResult_VK_SUCCESS,
	))
	return deviceMemory
}

// bindAndFillBuffers takes a list of buffer info, bind them with memory and
// fill them. If the total allocation size of the buffers can be fit in the
// fixed-size memory, bind with the fixed-size memory, returns the memory handle
// and false. A flush on this scratch resource may be triggered if the remaining
// space of the fixed-size memory is not large enough. If the total allocation
// size is greater than the fixed-size memory size, a temporary device memory
// will be created and returned with boolean value: true to indicate a temporary
// device memory is created.
func (qr *queueFamilyScratchResources) bindAndFillBuffers(totalAllocationSize uint64, buffers map[VkBuffer]scratchBufferInfo) (VkDeviceMemory, bool) {
	sb := qr.sb
	dev := qr.device
	var deviceMemory VkDeviceMemory
	var allocated uint64
	var usingTempMem bool
	if totalAllocationSize > qr.memorySize {
		deviceMemory = qr.newDeviceMemory(totalAllocationSize)
		allocated = uint64(0)
		usingTempMem = true
	} else {
		// Use the fixed-size scratch memory
		if totalAllocationSize+qr.allocated > qr.memorySize {
			qr.flush()
			return qr.bindAndFillBuffers(totalAllocationSize, buffers)
		}
		deviceMemory = qr.getDeviceMemory()
		allocated = qr.allocated
		usingTempMem = false
	}
	atData := sb.MustReserve(totalAllocationSize)
	ptrAtData := sb.newState.AllocDataOrPanic(sb.ctx, NewVoidᵖ(atData.Ptr()))
	sb.write(sb.cb.VkMapMemory(
		dev, deviceMemory, VkDeviceSize(allocated), VkDeviceSize(totalAllocationSize),
		VkMemoryMapFlags(0), ptrAtData.Ptr(), VkResult_VK_SUCCESS,
	).AddRead(ptrAtData.Data()).AddWrite(ptrAtData.Data()))
	ptrAtData.Free()

	bufBindingOffset := allocated
	for buf, info := range buffers {
		sb.write(sb.cb.VkBindBufferMemory(
			dev, buf, deviceMemory, VkDeviceSize(bufBindingOffset), VkResult_VK_SUCCESS))
		for _, r := range info.data {
			var hash id.ID
			var err error
			if r.hasNewData {
				hash, err = database.Store(sb.ctx, r.data)
				if err != nil {
					panic(err)
				}
			} else {
				hash = r.hash
			}
			sb.ReadDataAt(hash, atData.Address()+bufBindingOffset-allocated+r.rng.First, r.rng.Count)
		}
		bufBindingOffset += info.allocationSize
	}
	sb.write(sb.cb.VkFlushMappedMemoryRanges(
		dev,
		1,
		sb.MustAllocReadData(NewVkMappedMemoryRange(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
			0,                                 // pNext
			deviceMemory,                      // memory
			VkDeviceSize(allocated),           // offset
			VkDeviceSize(totalAllocationSize), // size
		)).Ptr(),
		VkResult_VK_SUCCESS,
	))
	sb.write(sb.cb.VkUnmapMemory(dev, deviceMemory))
	atData.Free()

	if !usingTempMem {
		qr.allocated += totalAllocationSize
	}
	return deviceMemory, usingTempMem
}

// flush submits all the command buffers of this scratch resource, waits until
// all the submitted commands finish, resets the command buffer, clear the
// usage of the fixed-size memory, and carry out the after-executed callbacks
// registered on this queue family scratch resource.
func (qr *queueFamilyScratchResources) flush() {
	sb := qr.sb
	for q, cb := range qr.commandBuffers {
		// Do not submit executed commandbuffer, state rebuilding does not reuse
		// recorded commands in command buffers.
		if GetState(sb.newState).CommandBuffers().Get(cb).Recording() != RecordingState_RECORDING {
			continue
		}
		sb.write(sb.cb.VkEndCommandBuffer(
			cb,
			VkResult_VK_SUCCESS,
		))

		sb.write(sb.cb.VkQueueSubmit(
			q,
			1,
			sb.MustAllocReadData(NewVkSubmitInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_SUBMIT_INFO, // sType
				0, // pNext
				0, // waitSemaphoreCount
				0, // pWaitSemaphores
				0, // pWaitDstStageMask
				1, // commandBufferCount
				NewVkCommandBufferᶜᵖ(sb.MustAllocReadData(cb).Ptr()), // pCommandBuffers
				0, // signalSemaphoreCount
				0, // pSignalSemaphores
			)).Ptr(),
			VkFence(0),
			VkResult_VK_SUCCESS,
		))
	}
	for q, cb := range qr.commandBuffers {
		sb.write(sb.cb.VkQueueWaitIdle(q, VkResult_VK_SUCCESS))
		sb.write(sb.cb.VkResetCommandBuffer(
			cb, VkCommandBufferResetFlags(VkCommandBufferResetFlagBits_VK_COMMAND_BUFFER_RESET_RELEASE_RESOURCES_BIT),
			VkResult_VK_SUCCESS,
		))
	}
	qr.allocated = 0
	for q, fs := range qr.postExecuted {
		for _, f := range fs {
			f()
		}
		qr.postExecuted[q] = []func(){}
	}
}

// free frees the command pool, command buffers and device memory of this
// queue family scratch resource.
func (qr *queueFamilyScratchResources) free() {
	sb := qr.sb
	sb.write(sb.cb.VkDestroyCommandPool(qr.device, qr.commandPool, memory.Nullptr))
	qr.commandPool = VkCommandPool(0)
	qr.commandBuffers = map[VkQueue]VkCommandBuffer{}
	sb.write(sb.cb.VkFreeMemory(qr.device, qr.memory, memory.Nullptr))
	qr.memory = VkDeviceMemory(0)
	qr.allocated = uint64(0)
}

// scratchTask wraps a set of buffers and command buffer commands which will be
// used together for host and GPU execution. Buffers in a scratchTask will be
// be allocated altogether, and command buffer commands will be submitted
// altogether after all the buffers are properly allocated. It is guaranteed
// that at the submission time of the command buffer commands, the buffers are
// allocated and available to be accessed by the commands. ScratchTask also
// holds callbacks for the host side commands to be carried out before the
// submission of the comamnd buffer commands, and after the execution of the
// commands.
type scratchTask struct {
	sb                  *stateBuilder
	buffers             map[VkBuffer]scratchBufferInfo
	totalAllocationSize uint64
	queue               VkQueue
	onCommit            []func()
	cmdBufRecorded      []func(VkCommandBuffer)
	defered             []func()
}

type scratchBufferInfo struct {
	data           []bufferSubRangeFillInfo
	size           uint64
	allocationSize uint64
}

// newScratchTaskOnQueue creates a new scratchTask for the given queue, all the
// commands to be recorded in the returned task will be submitted to the given
// queue, and all the scratch resources, e.g. scratch memory, will be provided
// by the queue family scratch resource of the given queue.
func (sb *stateBuilder) newScratchTaskOnQueue(queue VkQueue) *scratchTask {
	return &scratchTask{
		sb:                  sb,
		buffers:             map[VkBuffer]scratchBufferInfo{},
		totalAllocationSize: uint64(0),
		queue:               queue,
		onCommit:            []func(){},
		cmdBufRecorded:      []func(VkCommandBuffer){},
		defered:             []func(){},
	}
}

// commit closes a scratchTask, tries to allocate memory for its buffers,
// carries out the callbacks before the command buffer comamnds submission, add
// the command buffer commands to the command, and pass the after-execution
// callbacks to the after-execution callback queue.
func (t *scratchTask) commit() error {
	sb := t.sb
	if t.totalAllocationSize == uint64(0) {
		return log.Err(sb.ctx, nil, "Nil or empty scratch buffer session")
	}
	res := sb.getQueueFamilyScratchResources(t.queue)
	if mem, isTemp := res.bindAndFillBuffers(t.totalAllocationSize, t.buffers); isTemp {
		// The fixed size scratch buffer is not large enough for the allocation,
		// temporary device memory is created for this task, need to free the
		// memory after the task is done.
		t.deferUntilExecuted(func() {
			sb.write(sb.cb.VkFreeMemory(res.device, mem, memory.Nullptr))
		})
		defer res.flush()
	}
	for _, f := range t.onCommit {
		f()
	}
	cb := res.getCommandBuffer(t.queue)
	for _, f := range t.cmdBufRecorded {
		f(cb)
	}
	// pass the after-execution callbacks in the reverse order.
	for i := len(t.defered) - 1; i >= 0; i-- {
		res.postExecuted[t.queue] = append(res.postExecuted[t.queue], t.defered[i])
	}
	return nil
}

// doOnCommitted register callbacks to be called when this scratchTask is
// closed i.e. when onCommit() is called. Callbacks will be called in the order
// in the argument list, and the calling order of doOnCommited.
func (t *scratchTask) doOnCommitted(f ...func()) *scratchTask {
	t.onCommit = append(t.onCommit, f...)
	return t
}

// recordCmdBufCommand register callbacks to be called when this scratchTask is
// committed, and after all buffers are allocated properly. A command buffer
// will be given to each callback. Callbacks will be called in the same order
// of recordCmdBufCommand being called and the argument list.
func (t *scratchTask) recordCmdBufCommand(f ...func(cb VkCommandBuffer)) *scratchTask {
	t.cmdBufRecorded = append(t.cmdBufRecorded, f...)
	return t
}

// deferUntilExecuted register callbacks to be called when the execution of the
// command buffer commands of this scratchTask is fully finished. Note that the
// callbacks will be called in the reverse order of deferUntilExecuted being
// called and the argument list.
func (t *scratchTask) deferUntilExecuted(f ...func()) *scratchTask {
	t.defered = append(t.defered, f...)
	return t
}

// newBuffer creates a new VkBuffer with the given content and usage bits. The
// content will NOT be filled to the buffer until this scratchTask is committed,
// i.e. onCommit() being called. A VkBuffer will always be returned.
func (t *scratchTask) newBuffer(subRngs []bufferSubRangeFillInfo, usages ...VkBufferUsageFlagBits) VkBuffer {
	sb := t.sb
	size := uint64(0)
	for _, r := range subRngs {
		if r.rng.Span().End > size {
			size = r.rng.Span().End
		}
	}
	size = nextMultipleOf(size, 256)
	buffer := VkBuffer(newUnusedID(true, func(x uint64) bool {
		return sb.s.Buffers().Contains(VkBuffer(x)) || GetState(sb.newState).Buffers().Contains(VkBuffer(x))
	}))
	usageFlags := VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT)
	for _, u := range usages {
		usageFlags |= VkBufferUsageFlags(u)
	}
	dev := sb.s.Queues().Get(t.queue).Device()
	sb.write(sb.cb.VkCreateBuffer(
		dev,
		sb.MustAllocReadData(
			NewVkBufferCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_BUFFER_CREATE_INFO, // sType
				0,                                       // pNext
				0,                                       // flags
				VkDeviceSize(size),                      // size
				usageFlags,                              // usage
				VkSharingMode_VK_SHARING_MODE_EXCLUSIVE, // sharingMode
				0, // queueFamilyIndexCount
				0, // pQueueFamilyIndices
			)).Ptr(),
		memory.Nullptr,
		sb.MustAllocWriteData(buffer).Ptr(),
		VkResult_VK_SUCCESS,
	))
	allocSize := bufferAllocationSize(size)

	sb.write(sb.cb.VkGetBufferMemoryRequirements(
		dev,
		buffer,
		sb.MustAllocWriteData(NewVkMemoryRequirements(sb.ta,
			VkDeviceSize(allocSize), VkDeviceSize(256), 0xFFFFFFFF)).Ptr(),
	))

	t.buffers[buffer] = scratchBufferInfo{data: subRngs, size: size, allocationSize: allocSize}
	t.totalAllocationSize += allocSize
	return buffer
}

// bufferAllocationSize returns the memory allocation size for the given buffer
// size.
// Since we cannot guess how much the driver will actually request of us,
// overallocate by a factor of 2. This should be enough.
// Align to 0x100 to make validation layers happy. Assuming the buffer memory
// requirement has an alignment value compatible with 0x100.
func bufferAllocationSize(bufferSize uint64) uint64 {
	return nextMultipleOf(bufferSize*2, 256)
}
