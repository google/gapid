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

type queueFamilyScratchResources struct {
	sb             *stateBuilder
	device         VkDevice
	queueFamily    uint32
	commandPool    VkCommandPool
	commandBuffers map[VkQueue]VkCommandBuffer
	memory         VkDeviceMemory
	memorySize     VkDeviceSize
	allocated      uint64
	postExecuted   map[VkQueue][]func()
}

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
			memorySize:     VkDeviceSize(bufferAllocationSize(scratchBufferSize)),
			allocated:      uint64(0),
			postExecuted:   map[VkQueue][]func(){},
		}
	}
	return sb.scratchResources[dev][family]
}

func (sb *stateBuilder) flushAllScratchResources() {
	for _, familyInfo := range sb.scratchResources {
		for _, qr := range familyInfo {
			qr.flush()
		}
	}
}

func (sb *stateBuilder) freeAllScratchResources() {
	for _, familyInfo := range sb.scratchResources {
		for _, qr := range familyInfo {
			qr.free()
		}
	}
}

func (sb *stateBuilder) flushQueueFamilyScratchResources(queue VkQueue) {
	qr := sb.getQueueFamilyScratchResources(queue)
	qr.flush()
}

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

func (qr *queueFamilyScratchResources) getDeviceMemory() VkDeviceMemory {
	sb := qr.sb
	dev := qr.device
	if qr.memory == VkDeviceMemory(0) {
		deviceMemory := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
			return sb.s.DeviceMemories().Contains(VkDeviceMemory(x)) || GetState(sb.newState).DeviceMemories().Contains(VkDeviceMemory(x))
		}))
		memoryTypeIndex := sb.GetScratchBufferMemoryIndex(sb.s.Devices().Get(dev))
		memorySize := VkDeviceSize(bufferAllocationSize(scratchBufferSize))
		sb.write(sb.cb.VkAllocateMemory(
			dev,
			NewVkMemoryAllocateInfoᶜᵖ(sb.MustAllocReadData(
				NewVkMemoryAllocateInfo(sb.ta,
					VkStructureType_VK_STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO, // sType
					0, // pNext
					VkDeviceSize(memorySize), // allocationSize
					memoryTypeIndex,          // memoryTypeIndex
				)).Ptr()),
			memory.Nullptr,
			sb.MustAllocWriteData(deviceMemory).Ptr(),
			VkResult_VK_SUCCESS,
		))
		qr.memory = deviceMemory
		qr.memorySize = memorySize
		qr.allocated = uint64(0)
	}
	return qr.memory
}

func (qr *queueFamilyScratchResources) bindAndFillBuffers(totalAllocationSize uint64, buffers map[VkBuffer]scratchBufferInfo) error {
	sb := qr.sb
	dev := qr.device
	if totalAllocationSize > uint64(qr.memorySize) {
		return log.Errf(sb.ctx, nil, "cannot allocated scratch memory of size: %v, maximum allowed size is: %v", totalAllocationSize, qr.memorySize)
	}
	if totalAllocationSize+qr.allocated > uint64(qr.memorySize) {
		qr.flush()
		return qr.bindAndFillBuffers(totalAllocationSize, buffers)
	}
	deviceMemory := qr.getDeviceMemory()
	for buf, info := range buffers {
		sb.write(sb.cb.VkBindBufferMemory(
			dev, buf, deviceMemory, VkDeviceSize(qr.allocated), VkResult_VK_SUCCESS))

		atData := sb.MustReserve(info.allocationSize)
		ptrAtData := sb.newState.AllocDataOrPanic(sb.ctx, NewVoidᵖ(atData.Ptr()))
		sb.write(sb.cb.VkMapMemory(
			dev, deviceMemory, VkDeviceSize(qr.allocated), VkDeviceSize(info.allocationSize),
			VkMemoryMapFlags(0), ptrAtData.Ptr(), VkResult_VK_SUCCESS,
		).AddRead(ptrAtData.Data()).AddWrite(ptrAtData.Data()))
		ptrAtData.Free()

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
			sb.ReadDataAt(hash, atData.Address()+r.rng.First, r.rng.Count)
		}
		sb.write(sb.cb.VkFlushMappedMemoryRanges(
			dev,
			1,
			sb.MustAllocReadData(NewVkMappedMemoryRange(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
				0,                                 // pNext
				deviceMemory,                      // memory
				VkDeviceSize(qr.allocated),        // offset
				VkDeviceSize(info.allocationSize), // size
			)).Ptr(),
			VkResult_VK_SUCCESS,
		))

		sb.write(sb.cb.VkUnmapMemory(dev, deviceMemory))
		atData.Free()
		qr.allocated += info.allocationSize
	}
	return nil
}

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

func (qr *queueFamilyScratchResources) free() {
	sb := qr.sb
	sb.write(sb.cb.VkDestroyCommandPool(qr.device, qr.commandPool, memory.Nullptr))
	qr.commandPool = VkCommandPool(0)
	qr.commandBuffers = map[VkQueue]VkCommandBuffer{}
	sb.write(sb.cb.VkFreeMemory(qr.device, qr.memory, memory.Nullptr))
	qr.memory = VkDeviceMemory(0)
	qr.allocated = uint64(0)
}

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

func (t *scratchTask) commit() error {
	sb := t.sb
	if t.totalAllocationSize == uint64(0) {
		return log.Err(sb.ctx, nil, "Nil or empty scratch buffer session")
	}
	res := sb.getQueueFamilyScratchResources(t.queue)
	if err := res.bindAndFillBuffers(t.totalAllocationSize, t.buffers); err != nil {
		return log.Errf(sb.ctx, err, "commiting scratch buffers")
	}
	for _, f := range t.onCommit {
		f()
	}
	cb := res.getCommandBuffer(t.queue)
	for _, f := range t.cmdBufRecorded {
		f(cb)
	}
	for i := len(t.defered) - 1; i >= 0; i-- {
		res.postExecuted[t.queue] = append(res.postExecuted[t.queue], t.defered[i])
	}
	return nil
}

func (t *scratchTask) deferUntilCommitted(f ...func()) *scratchTask {
	t.onCommit = append(t.onCommit, f...)
	return t
}

func (t *scratchTask) recordCmdBufCommand(f ...func(cb VkCommandBuffer)) *scratchTask {
	t.cmdBufRecorded = append(t.cmdBufRecorded, f...)
	return t
}

func (t *scratchTask) deferUntilExecuted(f ...func()) *scratchTask {
	t.defered = append(t.defered, f...)
	return t
}

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
