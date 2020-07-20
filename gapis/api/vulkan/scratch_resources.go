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
	"context"
	"fmt"
	"sort"

	"github.com/google/gapid/core/data/id"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/memory"
)

const (
	scratchMemorySize      = 64 * 1024 * 1204
	scratchBufferAlignment = 256
)

// scratchResources holds a pool of flushing memory and command pool which can
// be used to allocate temporary memory and command buffers. It also contains
// a queue command handler for each queue.
type scratchResources struct {
	memories             map[VkDevice]*flushingMemory
	commandPools         map[VkDevice]map[uint32]VkCommandPool
	queueCommandHandlers map[VkQueue]*queueCommandHandler
}

func newScratchResources() *scratchResources {
	return &scratchResources{
		memories:             map[VkDevice]*flushingMemory{},
		commandPools:         map[VkDevice]map[uint32]VkCommandPool{},
		queueCommandHandlers: map[VkQueue]*queueCommandHandler{},
	}
}

// Free frees first submit all the pending commands held by all the queue
// command handlers, then free all the memories and command pools.
func (res *scratchResources) Free(sb *stateBuilder) {
	{
		keys := make([]VkQueue, 0, len(res.queueCommandHandlers))
		for k := range res.queueCommandHandlers {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, q := range keys {
			h := res.queueCommandHandlers[q]
			err := h.Submit(sb)
			if err != nil {
				panic(err)
			}
			err = h.WaitUntilFinish(sb)
			if err != nil {
				panic(err)
			}
			delete(res.queueCommandHandlers, q)
		}
	}
	{
		keys := make([]VkDevice, 0, len(res.memories))
		for k := range res.memories {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		for _, dev := range keys {
			mem := res.memories[dev]
			mem.Free(sb)
			delete(res.memories, dev)
		}
	}
	{
		keys := make([]VkDevice, 0, len(res.commandPools))
		for k := range res.commandPools {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		// Declare uKeys slice (used in inner loop) here so it is allocated only once
		uKeys := []uint32{}
		for _, dev := range keys {
			families := res.commandPools[dev]
			uKeys = []uint32{}
			for k := range families {
				uKeys = append(uKeys, k)
			}
			sort.Slice(uKeys, func(i, j int) bool { return uKeys[i] < uKeys[j] })
			for _, k := range uKeys {
				pool := families[k]
				sb.write(sb.cb.VkDestroyCommandPool(dev, pool, memory.Nullptr))
			}
			delete(res.commandPools, dev)
		}
	}
}

// GetCommandPool returns a command pool for the given device and queue family
// index, if such a pool has been created before in this scratch resources, the
// existing one will be returned, otherwise a new one will be created.
func (res *scratchResources) GetCommandPool(sb *stateBuilder, dev VkDevice, queueFamilyIndex uint32) VkCommandPool {
	if _, ok := res.commandPools[dev]; !ok {
		res.commandPools[dev] = map[uint32]VkCommandPool{}
	}
	if _, ok := res.commandPools[dev][queueFamilyIndex]; !ok {
		// create new command pool
		commandPool := VkCommandPool(newUnusedID(true, func(x uint64) bool {
			return sb.s.CommandPools().Contains(VkCommandPool(x)) || GetState(sb.newState).CommandPools().Contains(VkCommandPool(x))
		}))
		sb.write(sb.cb.VkCreateCommandPool(
			dev,
			sb.MustAllocReadData(NewVkCommandPoolCreateInfo(sb.ta,
				VkStructureType_VK_STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO, // sType
				0, // pNext
				VkCommandPoolCreateFlags(VkCommandPoolCreateFlagBits_VK_COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT), // flags
				queueFamilyIndex, // queueFamilyIndex
			)).Ptr(),
			memory.Nullptr,
			sb.MustAllocWriteData(commandPool).Ptr(),
			VkResult_VK_SUCCESS,
		))
		res.commandPools[dev][queueFamilyIndex] = commandPool
	}
	return res.commandPools[dev][queueFamilyIndex]
}

// AllocateCommandBuffer returns a new allocated command buffer from a command
// pool which is created with the given device and queue family index.
func (res *scratchResources) AllocateCommandBuffer(sb *stateBuilder, dev VkDevice, queueFamilyIndex uint32) VkCommandBuffer {
	commandBuffer := VkCommandBuffer(newUnusedID(true, func(x uint64) bool {
		return sb.s.CommandBuffers().Contains(VkCommandBuffer(x)) || GetState(sb.newState).CommandBuffers().Contains(VkCommandBuffer(x))
	}))
	sb.write(sb.cb.VkAllocateCommandBuffers(
		dev,
		sb.MustAllocReadData(NewVkCommandBufferAllocateInfo(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO, // sType
			0, // pNext
			res.GetCommandPool(sb, dev, queueFamilyIndex),        // commandPool
			VkCommandBufferLevel_VK_COMMAND_BUFFER_LEVEL_PRIMARY, // level
			uint32(1), // commandBufferCount
		)).Ptr(),
		sb.MustAllocWriteData(commandBuffer).Ptr(),
		VkResult_VK_SUCCESS,
	))
	scratchResName := debugMarkerName("scratch resource")
	attachDebugMarkerName(sb, scratchResName, dev, commandBuffer)
	return commandBuffer
}

// GetMemory returns a flushing memory for temporary memory allocation created
// from the given device. If such a flushing memory has been created before, the
// existing memory will be returned, otherwise a new one will be created.
func (res *scratchResources) GetMemory(sb *stateBuilder, dev VkDevice) *flushingMemory {
	if _, ok := res.memories[dev]; ok {
		return res.memories[dev]
	}
	mem := newFlushingMemory(sb, dev, scratchMemorySize, scratchBufferAlignment,
		debugMarkerName(fmt.Sprintf("scratchMemory dev: %v", dev)))
	res.memories[dev] = mem
	return mem
}

// GetQueueCommandHandler returns a queue command handler for the given queue,
// which means the commands recorded or committed to that command handler will
// be submitted to the given queue. If such a queue has been created before, that
// one will be returned, otherwise a new one will be returned.
func (res *scratchResources) GetQueueCommandHandler(sb *stateBuilder, queue VkQueue) *queueCommandHandler {
	if _, ok := res.queueCommandHandlers[queue]; ok {
		return res.queueCommandHandlers[queue]
	}
	queueObj := GetState(sb.newState).Queues().Get(queue)
	commandBuffer := res.AllocateCommandBuffer(sb, queueObj.Device(), queueObj.Family())
	handler, err := newQueueCommandHandler(sb, queue, commandBuffer)
	if err != nil {
		panic(err)
	}
	res.queueCommandHandlers[queue] = handler
	return handler
}

type bufferFlushInfo struct {
	buffer     VkBuffer
	dataSlices []hashedDataAndOffset
}

func flushDataToBuffers(sb *stateBuilder, alignment uint64, info ...bufferFlushInfo) error {
	memoryFlushes := map[VkDeviceMemory][]hashedDataAndOffset{}

	for _, bfi := range info {
		if !GetState(sb.newState).Buffers().Contains(bfi.buffer) {
			return log.Errf(sb.ctx, nil, "Buffer: %v not found in the new state of stateBuilder", bfi.buffer)
		}
		buf := GetState(sb.newState).Buffers().Get(bfi.buffer)
		if buf.Memory().IsNil() {
			return log.Errf(sb.ctx, nil, "Buffer: %v not bound with memory or is sparsely bound", bfi.buffer)
		}
		mem := buf.Memory().VulkanHandle()
		for _, s := range bfi.dataSlices {
			memoryFlushes[mem] = append(memoryFlushes[mem], hashedDataAndOffset{
				offset: s.offset + uint64(buf.MemoryOffset()),
				data:   s.data,
			})
		}
	}

	for m, f := range memoryFlushes {
		err := flushDataToMemory(sb, m, alignment, f...)
		if err != nil {
			return log.Errf(sb.ctx, err, "flush data to buffer's bound memory")
		}
	}
	return nil
}

// hashedData is a pair of hashed data ID and its size.
type hashedData struct {
	hash id.ID
	size uint64
}

// newHashedDataFromeBytes creates a new hashedData from raw bytes
func newHashedDataFromBytes(ctx context.Context, b []byte) hashedData {
	hash, err := database.Store(ctx, b)
	if err != nil {
		panic(err)
	}
	return hashedData{
		hash: hash,
		size: uint64(len(b)),
	}
}

// newHashedDataFromSlice creates a new hashedData from U8ˢ
func newHashedDataFromSlice(ctx context.Context, sliceSrcState *api.GlobalState, slice U8ˢ) hashedData {
	return hashedData{
		hash: slice.ResourceID(ctx, sliceSrcState),
		size: slice.Size(),
	}
}

// hashedDataAndOffset is a pair of offset and hashed data
type hashedDataAndOffset struct {
	offset uint64
	data   hashedData
}

func newHashedDataAndOffset(data hashedData, offset uint64) hashedDataAndOffset {
	return hashedDataAndOffset{
		offset: offset,
		data:   data,
	}
}

// flushDataToMemory takes a list of hashed data with offsets in device memory
// space and, flush the data to the given device memory based on the
// corresponding offsets.
func flushDataToMemory(sb *stateBuilder, deviceMemory VkDeviceMemory, alignment uint64, dataSlices ...hashedDataAndOffset) error {
	if len(dataSlices) == 0 {
		return nil
	}
	if !GetState(sb.newState).DeviceMemories().Contains(deviceMemory) {
		return fmt.Errorf("DeviceMemory: %v not found in the new state of stateBuilder", deviceMemory)
	}
	dev := GetState(sb.newState).DeviceMemories().Get(deviceMemory).Device()
	sort.Slice(dataSlices, func(i, j int) bool { return dataSlices[i].offset < dataSlices[j].offset })
	begin := dataSlices[0].offset / alignment * alignment
	end := nextMultipleOf(dataSlices[len(dataSlices)-1].offset+dataSlices[len(dataSlices)-1].data.size, alignment)
	atData := sb.MustReserve(end - begin)
	ptrAtData := sb.newState.AllocDataOrPanic(sb.ctx, NewVoidᵖ(atData.Ptr()))

	sb.write(sb.cb.VkMapMemory(
		dev, deviceMemory, VkDeviceSize(begin), VkDeviceSize(end-begin),
		VkMemoryMapFlags(0), ptrAtData.Ptr(), VkResult_VK_SUCCESS,
	).AddRead(ptrAtData.Data()).AddWrite(ptrAtData.Data()))
	ptrAtData.Free()

	for _, f := range dataSlices {
		sb.ReadDataAt(f.data.hash, atData.Address()+f.offset-begin, f.data.size)
	}
	sb.write(sb.cb.VkFlushMappedMemoryRanges(
		dev, 1,
		sb.MustAllocReadData(NewVkMappedMemoryRange(sb.ta,
			VkStructureType_VK_STRUCTURE_TYPE_MAPPED_MEMORY_RANGE, // sType
			0,                       // pNext
			deviceMemory,            // memory
			VkDeviceSize(begin),     // offset
			VkDeviceSize(end-begin), // size
		)).Ptr(),
		VkResult_VK_SUCCESS,
	))
	sb.write(sb.cb.VkUnmapMemory(dev, deviceMemory))
	atData.Free()
	return nil
}

// flushableResource is an interface for resources providers that controlls
// the life time of the resources offered by those providers. Users of the
// resource reserved by a flushableResource should register themselves with
// AddUser method to the flushableResource, and when they are done with the
// reserved piece of resource, the users should use DropUser to indicate the
// piece of resource can be recycled without notifying the user. When a flush
// is triggered (either explicitly by an entity out of the flushableResource, or
// implicitly by an internal logic of the flushableResource), all the users will
// be called with OnResourceFlush method, then all the previously reserved
// pieces of resources will be recycled and become invalid to access.
type flushableResource interface {
	flush(*stateBuilder)
	AddUser(flushableResourceUser)
	DropUser(flushableResourceUser)
}

// flushableResourceUser is an interface for types that can use the resources
// provided by flushableResource interface. When flush method is called on
// a flushableResource, the OnResourceFlush method will be called on the
// flushableResourceUser to process the pieces of resources this user uses.
type flushableResourceUser interface {
	OnResourceFlush(*stateBuilder, flushableResource)
}

// flushablePiece is an interface for resources provided by flushableResource
// interfaces, which can be used to query the provider of this piece of
// resource, and check if this piece of resource is still valid to use.
type flushablePiece interface {
	IsValid() bool
	Owner() flushableResource
}

// flushingMemory only guarantees the validity of the last allocated space, each
// incoming allocation request can cause a flush of pre-allocated data. Users of
// flushingMemory should register themself with AddUser() methods, and their
// OnResourceFlush() method will be call before a flush of allocated spaces is
// to occur.
type flushingMemory struct {
	size        uint64
	allocated   uint64
	alignment   uint64
	mem         VkDeviceMemory
	users       map[flushableResourceUser]struct{}
	newMem      func(*stateBuilder, uint64, debugMarkerName) VkDeviceMemory
	freeMem     func(*stateBuilder, VkDeviceMemory)
	name        debugMarkerName
	validPieces []*flushingMemoryAllocationResult
}

func newFlushingMemory(sb *stateBuilder, dev VkDevice, initialSize uint64, alignment uint64, name debugMarkerName) *flushingMemory {
	newMem := func(sb *stateBuilder, size uint64, nm debugMarkerName) VkDeviceMemory {
		deviceMemory := VkDeviceMemory(newUnusedID(true, func(x uint64) bool {
			return GetState(sb.oldState).DeviceMemories().Contains(VkDeviceMemory(x)) || GetState(sb.newState).DeviceMemories().Contains(VkDeviceMemory(x))
		}))
		memoryTypeIndex := sb.GetScratchBufferMemoryIndex(GetState(sb.newState).Devices().Get(dev))
		size = nextMultipleOf(size, alignment)
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
		if len(nm) > 0 {
			attachDebugMarkerName(sb, nm, dev, deviceMemory)
		}
		return deviceMemory
	}
	freeMem := func(sb *stateBuilder, mem VkDeviceMemory) {
		sb.write(sb.cb.VkFreeMemory(dev, mem, memory.Nullptr))
	}

	initialSize = nextMultipleOf(initialSize, alignment)
	return &flushingMemory{
		size:        initialSize,
		allocated:   0,
		alignment:   alignment,
		mem:         newMem(sb, initialSize, name),
		users:       map[flushableResourceUser]struct{}{},
		newMem:      newMem,
		freeMem:     freeMem,
		name:        name,
		validPieces: []*flushingMemoryAllocationResult{},
	}
}

// flushingMemoryAllocationResult contains the allocated device memory and
// offset for a memory reservation request, and also implements flushablePiece
// interface.
type flushingMemoryAllocationResult struct {
	valid  bool
	mem    VkDeviceMemory
	offset uint64
	owner  flushableResource
}

// IsValid impelements the flushablePiece interface
func (r *flushingMemoryAllocationResult) IsValid() bool {
	return r.valid
}

// Owner impelements the flushablePiece interface
func (r *flushingMemoryAllocationResult) Owner() flushableResource {
	return r.owner
}

// Memory returns the backing device memory of an allocation result.
func (r *flushingMemoryAllocationResult) Memory() VkDeviceMemory {
	return r.mem
}

// Offset returns the offset in the backing device memory for an allocation.
func (r *flushingMemoryAllocationResult) Offset() uint64 {
	return r.offset
}

// Allocate issues an request of memory allocation with the given size, and
// returns an allocation results with device memory and offset to tell the
// valid range to use for the caller. However, this may trigger a flush for the
// previously allocated memory ranges.
func (m *flushingMemory) Allocate(sb *stateBuilder, size uint64) (*flushingMemoryAllocationResult, error) {
	size = nextMultipleOf(size, m.alignment)
	if size > m.size {
		// Need expand the size of this memory
		size = nextMultipleOf(size, m.alignment)
		m.expand(sb, size)
		return m.Allocate(sb, size)
	} else if size+m.allocated > m.size {
		// Need scratch
		m.flush(sb)
		return m.Allocate(sb, size)
	}
	offset := m.allocated
	m.allocated += size
	res := &flushingMemoryAllocationResult{
		valid:  true,
		mem:    m.mem,
		offset: offset,
		owner:  m,
	}
	m.validPieces = append(m.validPieces, res)
	return res, nil
}

// flush implements the flushableResource interface.
func (m *flushingMemory) flush(sb *stateBuilder) {
	for u := range m.users {
		u.OnResourceFlush(sb, m)
	}
	for _, p := range m.validPieces {
		p.valid = false
	}
	m.validPieces = []*flushingMemoryAllocationResult{}
	m.allocated = 0
}

// Flush trigger a flush of previous allocated memory ranges.
func (m *flushingMemory) Flush(sb *stateBuilder) {
	m.flush(sb)
}

// expand replace the backing Vulkan device memory with a larger one. It will
// trigger a flush, destroy the existing device memory and create one with the
// given size.
func (m *flushingMemory) expand(sb *stateBuilder, size uint64) {
	// flush then reallocate memory
	m.flush(sb)
	m.freeMem(sb, m.mem)
	m.mem = m.newMem(sb, size, m.name)
	m.size = size
}

// Free flushes all the memory ranges allocated by this flushing memory and
// destroy the backing device memory handle.
func (m *flushingMemory) Free(sb *stateBuilder) {
	m.flush(sb)
	if m.mem != VkDeviceMemory(0) {
		m.freeMem(sb, m.mem)
		m.mem = VkDeviceMemory(0)
	}
	m.size = 0
	m.users = nil
}

// AddUser registers a user of the memory ranges allocated from this flushing memory
func (m *flushingMemory) AddUser(user flushableResourceUser) {
	m.users[user] = struct{}{}
}

// DropUSer removes a user from the user list of this flushing memory
func (m *flushingMemory) DropUser(user flushableResourceUser) {
	if _, ok := m.users[user]; ok {
		delete(m.users, user)
	}
}

// bufferAllocationSize returns the memory allocation size for the given buffer
// size.
// Since we cannot guess how much the driver will actually request of us,
// overallocate by a factor of 2. This should be enough.
// Align to 0x100 to make validation layers happy. Assuming the buffer memory
// requirement has an alignment value compatible with 0x100.
func bufferAllocationSize(bufferSize uint64, alignment uint64) uint64 {
	return nextMultipleOf(bufferSize*2, alignment)
}
