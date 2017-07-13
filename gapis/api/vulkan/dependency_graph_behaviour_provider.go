// Copyright (C) 2017 Google Inc.
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
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

type vulkanStateKey uint64

func (h vulkanStateKey) Parent() dependencygraph.StateKey {
	return nil
}

// Device memory composition hierarchy (parent -> child)
// vulkanDeviceMemory -> vulkanDeviceMemoryHandle
//                   \-> vulkanDeviceMemoryBinding -> vulkanDeviceMemoryData
type vulkanDeviceMemory struct {
	handle   *vulkanDeviceMemoryHandle
	bindings map[uint64][]*vulkanDeviceMemoryBinding // map from offsets to a list of memory bindings
}

type vulkanDeviceMemoryHandle struct {
	memory         *vulkanDeviceMemory
	vkDeviceMemory VkDeviceMemory
}

type vulkanDeviceMemoryBinding struct {
	memory *vulkanDeviceMemory
	start  uint64
	end    uint64
	data   *vulkanDeviceMemoryData
}

var emptyMemoryBindings = []*vulkanDeviceMemoryBinding{}

type vulkanDeviceMemoryData struct {
	binding *vulkanDeviceMemoryBinding
}

func (m *vulkanDeviceMemory) Parent() dependencygraph.StateKey {
	return nil
}

func (h *vulkanDeviceMemoryHandle) Parent() dependencygraph.StateKey {
	return h.memory
}

func (b *vulkanDeviceMemoryBinding) Parent() dependencygraph.StateKey {
	return b.memory
}

func (d *vulkanDeviceMemoryData) Parent() dependencygraph.StateKey {
	return d.binding
}

func newVulkanDeviceMemory(handle VkDeviceMemory) *vulkanDeviceMemory {
	m := &vulkanDeviceMemory{handle: nil, bindings: map[uint64][]*vulkanDeviceMemoryBinding{}}
	m.handle = &vulkanDeviceMemoryHandle{memory: m, vkDeviceMemory: handle}
	return m
}

func (m *vulkanDeviceMemory) addBinding(offset, size uint64) *vulkanDeviceMemoryBinding {
	newBinding := &vulkanDeviceMemoryBinding{
		memory: m,
		start:  offset,
		end:    offset + size,
		data:   nil}
	newBinding.data = &vulkanDeviceMemoryData{binding: newBinding}
	m.bindings[offset] = append(m.bindings[offset], newBinding)
	return newBinding
}

func (m *vulkanDeviceMemory) getOverlappedBindings(offset, size uint64) []*vulkanDeviceMemoryBinding {
	overlappedBindings := []*vulkanDeviceMemoryBinding{}
	for _, bl := range m.bindings {
		for _, b := range bl {
			if overlap(b.start, b.end, offset, offset+size) {
				overlappedBindings = append(overlappedBindings, b)
			}
		}
	}
	return overlappedBindings
}

func overlap(startA, endA, startB, endB uint64) bool {
	if (startA < endB && startA >= startB) ||
		(endA < endB && endA >= startB) ||
		(startB < startA && startB >= startA) ||
		(endB < endA && endB >= startA) {
		return true
	}
	return false
}

// Command buffer composition hierachy (parent -> child):
// vulkanCommandBuffer -> vulkanCommandBufferHandle
//                    \-> vulkanRecordedCommands
type vulkanCommandBuffer struct {
	handle  *vulkanCommandBufferHandle
	records *vulkanRecordedCommands
}

type vulkanCommandBufferHandle struct {
	CommandBuffer   *vulkanCommandBuffer
	vkCommandBuffer VkCommandBuffer
}

type vulkanRecordedCommands struct {
	CommandBuffer *vulkanCommandBuffer
	Commands      []*recordedCommand
}

type recordedCommand struct {
	recordedBehaviours      []func(b *dependencygraph.AtomBehaviour)
	secondaryCommandBuffers []*vulkanCommandBuffer
}

func newVulkanCommandBuffer(handle VkCommandBuffer) *vulkanCommandBuffer {
	cb := &vulkanCommandBuffer{handle: nil, records: nil}
	cb.handle = &vulkanCommandBufferHandle{CommandBuffer: cb, vkCommandBuffer: handle}
	cb.records = &vulkanRecordedCommands{CommandBuffer: cb,
		Commands: []*recordedCommand{}}
	return cb
}

func (cb *vulkanCommandBuffer) Parent() dependencygraph.StateKey {
	return nil
}

func (h *vulkanCommandBufferHandle) Parent() dependencygraph.StateKey {
	return h.CommandBuffer
}

func (c *vulkanRecordedCommands) Parent() dependencygraph.StateKey {
	return c.CommandBuffer
}

func (c *vulkanRecordedCommands) appendCommand(cmd *recordedCommand) {
	c.Commands = append(c.Commands, cmd)
}

func (r *recordedCommand) appendBehaviour(f func(b *dependencygraph.AtomBehaviour)) {
	r.recordedBehaviours = append(r.recordedBehaviours, f)
}

type submitInfo struct {
	commandBuffers []*vulkanCommandBuffer
}

type VulkanDependencyGraphBehaviourProvider struct {
	deviceMemories map[VkDeviceMemory]*vulkanDeviceMemory
	commandBuffers map[VkCommandBuffer]*vulkanCommandBuffer

	submissions map[*VkQueueSubmit][]submitInfo
}

func newVulkanDependencyGraphBehaviourProvider() *VulkanDependencyGraphBehaviourProvider {
	return &VulkanDependencyGraphBehaviourProvider{
		deviceMemories: map[VkDeviceMemory]*vulkanDeviceMemory{},
		commandBuffers: map[VkCommandBuffer]*vulkanCommandBuffer{},
		submissions:    map[*VkQueueSubmit][]submitInfo{},
	}
}

// For a given Vulkan handle of device memory, returns the corresponding
// stateKey of the device memory if it has been created and added to the graph
// before. Otherwise, creates and adds the stateKey for the handle and returns
// the new created stateKey
func (p *VulkanDependencyGraphBehaviourProvider) getOrCreateDeviceMemory(
	handle VkDeviceMemory) *vulkanDeviceMemory {
	if m, ok := p.deviceMemories[handle]; ok {
		return m
	}
	newM := newVulkanDeviceMemory(handle)
	p.deviceMemories[handle] = newM
	return newM
}

type executedCommandIndex struct {
	submit  *VkQueueSubmit
	Indices api.SubCmdIdx
}

// For a given Vulkan handle of command buffer, returns the corresponding
// stateKey of the command buffer if it has been created and added to the graph
// before. Otherwise, creates and adds the stateKey for the handle and returns
// the new created stateKey
func (p *VulkanDependencyGraphBehaviourProvider) getOrCreateCommandBuffer(
	handle VkCommandBuffer) *vulkanCommandBuffer {
	if cb, ok := p.commandBuffers[handle]; ok {
		return cb
	}
	newCb := newVulkanCommandBuffer(handle)
	p.commandBuffers[handle] = newCb
	return newCb
}

func (p *VulkanDependencyGraphBehaviourProvider) GetBehaviourForAtom(
	ctx context.Context, s *api.State, id api.CmdID, cmd api.Cmd, g *dependencygraph.DependencyGraph) dependencygraph.AtomBehaviour {
	// The behaviour going to be populated and returned.
	b := dependencygraph.AtomBehaviour{}
	// The recordedCommand going to be filled with behaviours which will be
	// carried out later when the command is executed.
	rc := &recordedCommand{
		recordedBehaviours:      []func(b *dependencygraph.AtomBehaviour){},
		secondaryCommandBuffers: []*vulkanCommandBuffer{},
	}

	l := s.MemoryLayout

	// Records all the truely executed commandbuffer commands. The behaviours
	// carried by those commands should only be rolled out when they are executed.
	// All the behaviours added by the executed commands will be rolled out when
	// processing vkQueueSubmit or vkSetEvent atoms.
	executedCommands := []executedCommandIndex{}
	GetState(s).HandleSubcommand = func(a interface{}) {
		subcommandIndex := append(api.SubCmdIdx(nil), GetState(s).SubCmdIdx...)
		submitAtom, ok := (*GetState(s).CurrentSubmission).(*VkQueueSubmit)
		if !ok {
			return
		}
		executedCommands = append(executedCommands,
			executedCommandIndex{submitAtom, subcommandIndex})
	}

	// Helper function for debug info logging when debug info dumpping is turned on
	debug := func(fmt string, args ...interface{}) {
		if config.DebugDeadCodeElimination {
			log.D(ctx, fmt, args...)
		}
	}

	// Wraps dependencygraph.AtomBehaviour's read/write/modify to add debug info.
	addRead := func(b *dependencygraph.AtomBehaviour, g *dependencygraph.DependencyGraph, state dependencygraph.StateKey) {
		b.Read(g, state)
		debug("\tread: stateKey: %v, stateAddress: %v", state, g.GetStateAddressOf(state))
	}
	addWrite := func(b *dependencygraph.AtomBehaviour, g *dependencygraph.DependencyGraph, state dependencygraph.StateKey) {
		b.Write(g, state)
		debug("\twrite: stateKey: %v, stateAddress: %v", state, g.GetStateAddressOf(state))
	}
	addModify := func(b *dependencygraph.AtomBehaviour, g *dependencygraph.DependencyGraph, state dependencygraph.StateKey) {
		b.Modify(g, state)
		debug("\tmodify: stateKey: %v, stateAddress: %v", state, g.GetStateAddressOf(state))
	}

	// Helper function that gets overlapped memory bindings with a given offset and size
	getOverlappingMemoryBindings := func(memory VkDeviceMemory,
		offset, size uint64) []*vulkanDeviceMemoryBinding {
		return p.getOrCreateDeviceMemory(memory).getOverlappedBindings(offset, size)
	}

	// Helper function that gets the overlapped memory bindings for a given image
	getOverlappedBindingsForImage := func(image VkImage) []*vulkanDeviceMemoryBinding {
		if !GetState(s).Images.Contains(image) {
			log.E(ctx, "Error Image: %v: does not exist in state", image)
			return []*vulkanDeviceMemoryBinding{}
		}
		imageObj := GetState(s).Images.Get(image)
		if imageObj.IsSwapchainImage {
			return []*vulkanDeviceMemoryBinding{}
		} else if imageObj.BoundMemory != nil {
			boundMemory := imageObj.BoundMemory.VulkanHandle
			offset := uint64(imageObj.BoundMemoryOffset)
			infer_size, err := subInferImageSize(ctx, cmd, nil, s, nil, cmd.Thread(), nil, imageObj)
			if err != nil {
				log.E(ctx, "Error Image: %v: Cannot infer the size of the image", image)
			}
			size := uint64(infer_size)
			return getOverlappingMemoryBindings(boundMemory, offset, size)
		} else {
			log.E(ctx, "Error Image: %v: Cannot get the bound memory for an image which has not been bound yet", image)
			return []*vulkanDeviceMemoryBinding{}
		}
	}

	// Helper function that gets the overlapped memory bindings for a given buffer
	getOverlappedBindingsForBuffer := func(buffer VkBuffer) []*vulkanDeviceMemoryBinding {
		if !GetState(s).Buffers.Contains(buffer) {
			log.E(ctx, "Error Buffer: %v: does not exist in state", buffer)
			return []*vulkanDeviceMemoryBinding{}
		}
		bufferObj := GetState(s).Buffers.Get(buffer)
		if bufferObj.Memory != nil {
			boundMemory := bufferObj.Memory.VulkanHandle
			offset := uint64(bufferObj.MemoryOffset)
			size := uint64(uint64(bufferObj.Info.Size))
			return getOverlappingMemoryBindings(boundMemory, offset, size)
		} else {
			log.E(ctx, "Error Buffer: %v: Cannot get the bound memory for a buffer which has not been bound yet", buffer)
			return []*vulkanDeviceMemoryBinding{}
		}
	}

	// Helper function that reads the given image handle, and returns the memory
	// bindings of the image
	readImageHandleAndGetBindings := func(b *dependencygraph.AtomBehaviour, image VkImage) []*vulkanDeviceMemoryBinding {
		b.Read(g, vulkanStateKey(image))
		return getOverlappedBindingsForImage(image)
	}

	// Helper function that reads the given buffer handle, and returns the memory
	// bindings of the buffer
	readBufferHandleAndGetBindings := func(b *dependencygraph.AtomBehaviour, buffer VkBuffer) []*vulkanDeviceMemoryBinding {
		b.Read(g, vulkanStateKey(buffer))
		return getOverlappedBindingsForBuffer(buffer)
	}

	// Helper function that 'read' the given memory bindings
	readMemoryBindingsData := func(pb *dependencygraph.AtomBehaviour, bindings []*vulkanDeviceMemoryBinding) {
		for _, binding := range bindings {
			pb.Read(g, binding.data)
			debug("\tread binding data: %v <-  binding: %v <- memory: %v",
				g.GetStateAddressOf(binding.data),
				g.GetStateAddressOf(binding),
				g.GetStateAddressOf(binding.Parent()))
		}
	}

	// Helper function that 'write' the given memory bindings
	writeMemoryBindingsData := func(pb *dependencygraph.AtomBehaviour, bindings []*vulkanDeviceMemoryBinding) {
		for _, binding := range bindings {
			pb.Write(g, binding.data)
			debug("\twrite binding data: %v <- binding: %v <- memory: %v",
				g.GetStateAddressOf(binding.data),
				g.GetStateAddressOf(binding),
				g.GetStateAddressOf(binding.Parent()))
		}
	}

	// Helper function that 'modify' the given memory bindings
	modifyMemoryBindingsData := func(pb *dependencygraph.AtomBehaviour, bindings []*vulkanDeviceMemoryBinding) {
		for _, binding := range bindings {
			pb.Modify(g, binding.data)
			debug("\tmodify binding data: %v <- binding: %v <- memory: %v", binding.data, g.GetStateAddressOf(binding.data), g.GetStateAddressOf(binding), g.GetStateAddressOf(binding.Parent()))
		}
	}

	// Helper function that adds 'read' to the given command buffer handle and
	// 'modify' to the given comamnd buffer records to the current behavior if
	// such behaviours have not been added before. And records a callback to
	// carry out other behaviours later when the command buffer is submitted.
	recordCommand := func(currentBehaviour *dependencygraph.AtomBehaviour,
		handle VkCommandBuffer,
		c func(futureBehaviour *dependencygraph.AtomBehaviour)) {
		cmdBuf := p.getOrCreateCommandBuffer(handle)
		if len(currentBehaviour.Reads) == 0 ||
			currentBehaviour.Reads[len(currentBehaviour.Reads)-1] !=
				g.GetStateAddressOf(cmdBuf.handle) {
			currentBehaviour.Read(g, cmdBuf.handle)
		}
		if len(currentBehaviour.Modifies) == 0 ||
			currentBehaviour.Modifies[len(currentBehaviour.Modifies)-1] !=
				g.GetStateAddressOf(cmdBuf.records) {
			currentBehaviour.Modify(g, cmdBuf.records)
		}
		rc.appendBehaviour(c)
		// If current recordedCommand is not same as the last one in the command
		// buffer, this must be a new command, and it should be appended to the
		// list of recorded commands
		if len(cmdBuf.records.Commands) == 0 ||
			rc != cmdBuf.records.Commands[len(cmdBuf.records.Commands)-1] {
			cmdBuf.records.Commands = append(cmdBuf.records.Commands, rc)
		}
	}

	// Helper function that adds 'read' to the given command buffer handle and
	// 'modify' to the given command buffer records to the current behaviour if
	// such behaviours have not been added before. And adds a secondary command
	// buffer to the current command, so that later we can roll out the behaviours
	// registered in the secondary command buffer.
	recordSecondaryCommandBuffer := func(
		currentBehaviour *dependencygraph.AtomBehaviour,
		handle VkCommandBuffer,
		scb *vulkanCommandBuffer) {
		cmdBuf := p.getOrCreateCommandBuffer(handle)
		if len(currentBehaviour.Reads) == 0 ||
			currentBehaviour.Reads[len(currentBehaviour.Reads)-1] !=
				g.GetStateAddressOf(cmdBuf.handle) {
			currentBehaviour.Read(g, cmdBuf.handle)
		}
		if len(currentBehaviour.Modifies) == 0 ||
			currentBehaviour.Modifies[len(currentBehaviour.Modifies)-1] !=
				g.GetStateAddressOf(cmdBuf.records) {
			currentBehaviour.Modify(g, cmdBuf.records)
		}
		rc.secondaryCommandBuffers = append(rc.secondaryCommandBuffers, scb)
		// If current recordedCommand is not same as the last one in the command
		// buffer, this must be a new command, and it should be appended to the
		// list of recorded commands
		if len(cmdBuf.records.Commands) == 0 ||
			rc != cmdBuf.records.Commands[len(cmdBuf.records.Commands)-1] {
			cmdBuf.records.Commands = append(cmdBuf.records.Commands, rc)
		}
	}

	// Helper function that adds 'read' to the given command buffer handle and
	// 'modify' to the given comamnd buffer records to the current behavior, if
	// such behaviours have not been added before. And records 'read' of the
	// given read memory bindings, 'modify' of the given modify memory bindings
	// and 'write' of the given write memory bindings, to be carried out later
	// when the command buffer is submitted.
	recordTouchingMemoryBindingsData := func(currentBehaviour *dependencygraph.AtomBehaviour,
		handle VkCommandBuffer,
		readBindings, modifyBindings, writeBindings []*vulkanDeviceMemoryBinding) {
		cmdBuf := p.getOrCreateCommandBuffer(handle)
		if len(currentBehaviour.Reads) == 0 || currentBehaviour.Reads[len(currentBehaviour.Reads)-1] !=
			g.GetStateAddressOf(cmdBuf.handle) {
			currentBehaviour.Read(g, cmdBuf.handle)
		}
		if len(currentBehaviour.Modifies) == 0 || currentBehaviour.Modifies[len(currentBehaviour.Modifies)-1] !=
			g.GetStateAddressOf(cmdBuf.records) {
			currentBehaviour.Modify(g, cmdBuf.records)
		}

		rc.appendBehaviour(func(b *dependencygraph.AtomBehaviour) {
			readMemoryBindingsData(b, readBindings)
			modifyMemoryBindingsData(b, modifyBindings)
			writeMemoryBindingsData(b, writeBindings)
		})
		// If current recordedCommand is not same as the last one in the command
		// buffer, this must be a new command, and it should be appended to the
		// list of recorded commands
		if len(cmdBuf.records.Commands) == 0 ||
			rc != cmdBuf.records.Commands[len(cmdBuf.records.Commands)-1] {
			cmdBuf.records.Commands = append(cmdBuf.records.Commands, rc)
		}
	}

	// Mutate the state with the atom.
	if err := cmd.Mutate(ctx, s, nil); err != nil {
		log.E(ctx, "Command %v %v: %v", id, cmd, err)
		return dependencygraph.AtomBehaviour{Aborted: true}
	}

	debug("DCE::DependencyGraph::getBehaviour: %v, %T", id, cmd)

	// Add behaviors for the atom according to its type.
	// Note that there are a few cases in which the behaviour is NOT added to the
	// place that the behaviour is carried out in real execution of the API
	// commands:
	// Draw commands (vkCmdDraw, RecreateCmdDraw, vkCmdDrawIndexed, etc):
	// The 'read' behaviour of the currently bound vertex buffer and index
	// buffers are recorded to the command buffer records by binding commands,
	// like: vkCmdBindVertexBuffers etc, not by the draw commands. This is
	// because after the call to vkQueueSubmit's Mutate(), when we process the
	// recorded draw command, only the last set of bound vertex buffers and
	// bound index buffer will be kept in the global's state
	// CurrentBoundVertexBuffers. So we cannot obtain previous bound vertex
	// buffers from it and so we cannot add 'read' behaviours to the buffers
	// data. To solve the problem, we read the buffer memory data here. This may
	// result into a dummy read behavior of the buffer data, as the buffer may
	// never be used later. But this ensures the correctness of the trace and the
	// state.
	// 'Read' and 'modify' behaviours to descriptors, like textures, uniform
	// buffers, etc, have similar problem, as we cannot application is allowed
	// to call vkCmdBindDescriptorSets multiple times and we only get the last
	// bound one after VkQueueSubmit's Mutate() is called. So we records the
	// behaviours in VkCmdBindDescriptorSets and RecreateCmdBindDescriptorSets,
	// instead of the draw calls.
	switch cmd := cmd.(type) {
	case *VkCreateImage:
		image := cmd.PImage.Read(ctx, cmd, s, nil)
		addWrite(&b, g, vulkanStateKey(image))

	case *VkCreateBuffer:
		buffer := cmd.PBuffer.Read(ctx, cmd, s, nil)
		addWrite(&b, g, vulkanStateKey(buffer))

	case *RecreateImage:
		image := cmd.PImage.Read(ctx, cmd, s, nil)
		addWrite(&b, g, vulkanStateKey(image))

	case *RecreateBuffer:
		buffer := cmd.PBuffer.Read(ctx, cmd, s, nil)
		addWrite(&b, g, vulkanStateKey(buffer))

	case *VkAllocateMemory:
		allocateInfo := cmd.PAllocateInfo.Read(ctx, cmd, s, nil)
		memory := cmd.PMemory.Read(ctx, cmd, s, nil)
		addWrite(&b, g, p.getOrCreateDeviceMemory(memory))

		// handle dedicated memory allocation
		if allocateInfo.PNext != (Voidᶜᵖ{}) {
			pNext := Voidᵖ(allocateInfo.PNext)
			for pNext != (Voidᵖ{}) {
				sType := (VkStructureTypeᶜᵖ(pNext)).Read(ctx, cmd, s, nil)
				switch sType {
				case VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV:
					ext := VkDedicatedAllocationMemoryAllocateInfoNVᵖ(pNext).Read(ctx, cmd, s, nil)
					image := ext.Image
					buffer := ext.Buffer
					if uint64(image) != 0 {
						addRead(&b, g, vulkanStateKey(image))
					}
					if uint64(buffer) != 0 {
						addRead(&b, g, vulkanStateKey(buffer))
					}
				}
				pNext = (VulkanStructHeaderᵖ(pNext)).Read(ctx, cmd, s, nil).PNext
			}
		}

	case *RecreateDeviceMemory:
		allocateInfo := cmd.PAllocateInfo.Read(ctx, cmd, s, nil)
		memory := cmd.PMemory.Read(ctx, cmd, s, nil)
		addWrite(&b, g, p.getOrCreateDeviceMemory(memory))

		// handle dedicated memory allocation
		if allocateInfo.PNext != (Voidᶜᵖ{}) {
			pNext := Voidᵖ(allocateInfo.PNext)
			for pNext != (Voidᵖ{}) {
				sType := (VkStructureTypeᶜᵖ(pNext)).Read(ctx, cmd, s, nil)
				switch sType {
				case VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV:
					ext := VkDedicatedAllocationMemoryAllocateInfoNVᵖ(pNext).Read(ctx, cmd, s, nil)
					image := ext.Image
					buffer := ext.Buffer
					if uint64(image) != 0 {
						addRead(&b, g, vulkanStateKey(image))
					}
					if uint64(buffer) != 0 {
						addRead(&b, g, vulkanStateKey(buffer))
					}
				}
				pNext = (VulkanStructHeaderᵖ(pNext)).Read(ctx, cmd, s, nil).PNext
			}
		}

	case *VkGetDeviceMemoryCommitment:
		memory := cmd.Memory
		addRead(&b, g, p.getOrCreateDeviceMemory(memory).handle)

	case *VkGetImageMemoryRequirements:
		image := cmd.Image
		addRead(&b, g, vulkanStateKey(image))

	case *VkGetImageSparseMemoryRequirements:
		image := cmd.Image
		addRead(&b, g, vulkanStateKey(image))

	case *VkGetImageSubresourceLayout:
		image := cmd.Image
		addRead(&b, g, vulkanStateKey(image))

	case *VkGetBufferMemoryRequirements:
		buffer := cmd.Buffer
		addModify(&b, g, vulkanStateKey(buffer))

	case *VkBindImageMemory:
		image := cmd.Image
		memory := cmd.Memory
		offset := cmd.MemoryOffset
		addModify(&b, g, vulkanStateKey(image))
		addRead(&b, g, p.getOrCreateDeviceMemory(memory).handle)
		if GetState(s).Images.Contains(image) {
			// In some applications, `vkGetImageMemoryRequirements` is not called so we
			// don't have the image size. However, a memory binding for a zero-sized
			// memory range will also be created here and used later to check
			// overlapping. The problem is that this memory range will always be
			// considered as fully covered by any range that starts at the same offset
			// or across the offset.
			// So to ensure correctness, overwriting of zero sized memory binding is
			// not allowed, execept for the vkCmdBeginRenderPass, whose target is
			// always an image as a whole.
			// TODO(qining) Fix this
			infer_size, err := subInferImageSize(ctx, cmd, nil, s, nil, cmd.thread, nil, GetState(s).Images.Get(image))
			if err != nil {
				log.E(ctx, "Command %v %v: %v", id, cmd, err)
				return dependencygraph.AtomBehaviour{Aborted: true}
			}
			size := uint64(infer_size)
			binding := p.getOrCreateDeviceMemory(memory).addBinding(uint64(offset), size)
			addWrite(&b, g, binding)
		}

	case *VkBindBufferMemory:
		buffer := cmd.Buffer
		memory := cmd.Memory
		offset := cmd.MemoryOffset
		addModify(&b, g, vulkanStateKey(buffer))
		addRead(&b, g, p.getOrCreateDeviceMemory(memory).handle)
		if GetState(s).Buffers.Contains(buffer) {
			size := uint64(GetState(s).Buffers.Get(buffer).Info.Size)
			binding := p.getOrCreateDeviceMemory(memory).addBinding(uint64(offset), size)
			addWrite(&b, g, binding)
		}

	case *RecreateBindImageMemory:
		image := cmd.Image
		memory := cmd.Memory
		offset := cmd.Offset
		addModify(&b, g, vulkanStateKey(image))
		addRead(&b, g, p.getOrCreateDeviceMemory(memory).handle)
		if GetState(s).Images.Contains(image) {
			infer_size, err := subInferImageSize(ctx, cmd, nil, s, nil, cmd.thread, nil, GetState(s).Images.Get(image))
			if err != nil {
				log.E(ctx, "Command %v %v: %v", id, cmd, err)
				return dependencygraph.AtomBehaviour{Aborted: true}
			}
			size := uint64(infer_size)
			binding := p.getOrCreateDeviceMemory(memory).addBinding(uint64(offset), size)
			addWrite(&b, g, binding)
		}

	case *RecreateBindBufferMemory:
		buffer := cmd.Buffer
		memory := cmd.Memory
		offset := cmd.Offset
		addModify(&b, g, vulkanStateKey(buffer))
		addRead(&b, g, p.getOrCreateDeviceMemory(memory).handle)
		if GetState(s).Buffers.Contains(buffer) {
			size := uint64(GetState(s).Buffers.Get(buffer).Info.Size)
			binding := p.getOrCreateDeviceMemory(memory).addBinding(uint64(offset), size)
			addWrite(&b, g, binding)
		}

	case *RecreateImageData:
		image := cmd.Image
		addModify(&b, g, vulkanStateKey(image))
		overlappingBindings := getOverlappedBindingsForImage(image)
		writeMemoryBindingsData(&b, overlappingBindings)

	case *RecreateBufferData:
		buffer := cmd.Buffer
		addModify(&b, g, vulkanStateKey(buffer))
		overlappingBindings := getOverlappedBindingsForBuffer(buffer)
		writeMemoryBindingsData(&b, overlappingBindings)

	case *VkDestroyImage:
		image := cmd.Image
		addModify(&b, g, vulkanStateKey(image))
		b.KeepAlive = true

	case *VkDestroyBuffer:
		buffer := cmd.Buffer
		addModify(&b, g, vulkanStateKey(buffer))
		b.KeepAlive = true

	case *VkFreeMemory:
		memory := cmd.Memory
		// Free/deletion atoms are kept alive so the creation atom of the
		// corresponding handle will also be kept alive, even though the handle
		// may not be used anywhere else.
		addRead(&b, g, vulkanStateKey(memory))
		b.KeepAlive = true

	case *VkMapMemory:
		memory := cmd.Memory
		addModify(&b, g, p.getOrCreateDeviceMemory(memory))

	case *VkUnmapMemory:
		memory := cmd.Memory
		addModify(&b, g, p.getOrCreateDeviceMemory(memory))

	case *VkFlushMappedMemoryRanges:
		ranges := cmd.PMemoryRanges.Slice(0, uint64(cmd.MemoryRangeCount), l)
		// TODO: Link the contiguous ranges into one so that we don't miss
		// potential overwrites
		for i := uint64(0); i < uint64(cmd.MemoryRangeCount); i++ {
			mappedRange := ranges.Index(i, l).Read(ctx, cmd, s, nil)
			memory := mappedRange.Memory
			offset := uint64(mappedRange.Offset)
			size := uint64(mappedRange.Size)
			// For the overlapping bindings in the memory, if the flush range covers
			// the whole binding range, the data in that binding will be overwritten,
			// otherwise the data is modified.
			bindings := getOverlappingMemoryBindings(memory, offset, size)
			for _, binding := range bindings {
				if offset <= binding.start && offset+size >= binding.end {
					// If the memory binding size is zero, the binding is for an image
					// whose size is unknown at binding time. As we don't know whether
					// this flush overwrites the whole image, we conservatively label the
					// flushing always as 'modify'
					if binding.start == binding.end {
						addModify(&b, g, binding.data)
					} else {
						addWrite(&b, g, binding.data)
					}
				} else {
					addModify(&b, g, binding.data)
				}
			}
		}

	case *VkInvalidateMappedMemoryRanges:
		ranges := cmd.PMemoryRanges.Slice(0, uint64(cmd.MemoryRangeCount), l)
		// TODO: Link the contiguous ranges
		for i := uint64(0); i < uint64(cmd.MemoryRangeCount); i++ {
			mappedRange := ranges.Index(i, l).Read(ctx, cmd, s, nil)
			memory := mappedRange.Memory
			offset := uint64(mappedRange.Offset)
			size := uint64(mappedRange.Size)
			bindings := getOverlappingMemoryBindings(memory, offset, size)
			readMemoryBindingsData(&b, bindings)
		}

	case *VkCreateImageView:
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		image := createInfo.Image
		view := cmd.PView.Read(ctx, cmd, s, nil)
		addRead(&b, g, vulkanStateKey(image))
		addWrite(&b, g, vulkanStateKey(view))

	case *RecreateImageView:
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		image := createInfo.Image
		view := cmd.PImageView.Read(ctx, cmd, s, nil)
		addRead(&b, g, vulkanStateKey(image))
		addWrite(&b, g, vulkanStateKey(view))

	case *VkCreateBufferView:
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		buffer := createInfo.Buffer
		view := cmd.PView.Read(ctx, cmd, s, nil)
		addRead(&b, g, vulkanStateKey(buffer))
		addWrite(&b, g, vulkanStateKey(view))

	case *RecreateBufferView:
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		buffer := createInfo.Buffer
		view := cmd.PBufferView.Read(ctx, cmd, s, nil)
		addRead(&b, g, vulkanStateKey(buffer))
		addWrite(&b, g, vulkanStateKey(view))

	case *VkUpdateDescriptorSets:
		// handle descriptor writes
		writeCount := cmd.DescriptorWriteCount
		if writeCount > 0 {
			writes := cmd.PDescriptorWrites.Slice(0, uint64(writeCount), l)
			if err := processDescriptorWrites(writes, &b, g, ctx, cmd, s); err != nil {
				log.E(ctx, "Command %v %v: %v", id, cmd, err)
				return dependencygraph.AtomBehaviour{Aborted: true}
			}
		}
		// handle descriptor copies
		copyCount := cmd.DescriptorCopyCount
		if copyCount > 0 {
			copies := cmd.PDescriptorCopies.Slice(0, uint64(copyCount), l)
			for i := uint32(0); i < copyCount; i++ {
				copy := copies.Index(uint64(i), l).Read(ctx, cmd, s, nil)
				srcDescriptor := copy.SrcSet
				dstDescriptor := copy.DstSet
				addRead(&b, g, vulkanStateKey(srcDescriptor))
				addModify(&b, g, vulkanStateKey(dstDescriptor))
			}
		}

	case *RecreateDescriptorSet:
		// handle descriptor writes
		writeCount := cmd.DescriptorWriteCount
		if writeCount > 0 {
			writes := cmd.PDescriptorWrites.Slice(0, uint64(writeCount), l)
			if err := processDescriptorWrites(writes, &b, g, ctx, cmd, s); err != nil {
				log.E(ctx, "Command %v %v: %v", id, cmd, err)
				return dependencygraph.AtomBehaviour{Aborted: true}
			}
		}

	case *VkCreateFramebuffer:
		addWrite(&b, g, vulkanStateKey(cmd.PFramebuffer.Read(ctx, cmd, s, nil)))
		addRead(&b, g, vulkanStateKey(cmd.PCreateInfo.Read(ctx, cmd, s, nil).RenderPass))
		// process the attachments
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		attachmentCount := createInfo.AttachmentCount
		attachments := createInfo.PAttachments.Slice(0, uint64(attachmentCount), l)
		for i := uint32(0); i < attachmentCount; i++ {
			attachedViews := attachments.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			addRead(&b, g, vulkanStateKey(attachedViews))
		}

	case *RecreateFramebuffer:
		addWrite(&b, g, vulkanStateKey(cmd.PFramebuffer.Read(ctx, cmd, s, nil)))
		addRead(&b, g, vulkanStateKey(cmd.PCreateInfo.Read(ctx, cmd, s, nil).RenderPass))
		// process the attachments
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		attachmentCount := createInfo.AttachmentCount
		attachments := createInfo.PAttachments.Slice(0, uint64(attachmentCount), l)
		for i := uint32(0); i < attachmentCount; i++ {
			attachedViews := attachments.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			addRead(&b, g, vulkanStateKey(attachedViews))
		}

	case *VkDestroyFramebuffer:
		addModify(&b, g, vulkanStateKey(cmd.Framebuffer))
		b.KeepAlive = true

	case *VkCreateRenderPass:
		addWrite(&b, g, vulkanStateKey(cmd.PRenderPass.Read(ctx, cmd, s, nil)))

	case *RecreateRenderPass:
		addWrite(&b, g, vulkanStateKey(cmd.PRenderPass.Read(ctx, cmd, s, nil)))

	case *VkDestroyRenderPass:
		addModify(&b, g, vulkanStateKey(cmd.RenderPass))
		b.KeepAlive = true

	case *VkGetRenderAreaGranularity:
		addRead(&b, g, vulkanStateKey(cmd.RenderPass))

	case *VkCreateGraphicsPipelines:
		pipelineCount := uint64(cmd.CreateInfoCount)
		createInfos := cmd.PCreateInfos.Slice(0, pipelineCount, l)
		pipelines := cmd.PPipelines.Slice(0, pipelineCount, l)
		for i := uint64(0); i < pipelineCount; i++ {
			// read shaders
			stageCount := uint64(createInfos.Index(i, l).Read(ctx, cmd, s, nil).StageCount)
			shaderStages := createInfos.Index(i, l).Read(ctx, cmd, s, nil).PStages.Slice(0, stageCount, l)
			for j := uint64(0); j < stageCount; j++ {
				shaderStage := shaderStages.Index(j, l).Read(ctx, cmd, s, nil)
				module := shaderStage.Module
				addRead(&b, g, vulkanStateKey(module))
			}
			// read renderpass
			renderPass := createInfos.Index(i, l).Read(ctx, cmd, s, nil).RenderPass
			addRead(&b, g, vulkanStateKey(renderPass))
			// Create pipeline
			pipeline := pipelines.Index(i, l).Read(ctx, cmd, s, nil)
			addWrite(&b, g, vulkanStateKey(pipeline))
		}

	case *RecreateGraphicsPipeline:
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		stageCount := uint64(createInfo.StageCount)
		shaderStages := createInfo.PStages.Slice(0, stageCount, l)
		for i := uint64(0); i < stageCount; i++ {
			shaderStage := shaderStages.Index(i, l).Read(ctx, cmd, s, nil)
			addRead(&b, g, vulkanStateKey(shaderStage.Module))
		}
		addRead(&b, g, vulkanStateKey(createInfo.RenderPass))
		addWrite(&b, g, vulkanStateKey(cmd.PPipeline.Read(ctx, cmd, s, nil)))

	case *VkCreateComputePipelines:
		pipelineCount := uint64(cmd.CreateInfoCount)
		createInfos := cmd.PCreateInfos.Slice(0, pipelineCount, l)
		pipelines := cmd.PPipelines.Slice(0, pipelineCount, l)
		for i := uint64(0); i < pipelineCount; i++ {
			// read shader
			shaderStage := createInfos.Index(i, l).Read(ctx, cmd, s, nil).Stage
			module := shaderStage.Module
			addRead(&b, g, vulkanStateKey(module))
			// Create pipeline
			pipeline := pipelines.Index(i, l).Read(ctx, cmd, s, nil)
			addWrite(&b, g, vulkanStateKey(pipeline))
		}

	case *RecreateComputePipeline:
		createInfo := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		module := createInfo.Stage.Module
		addRead(&b, g, vulkanStateKey(module))
		addWrite(&b, g, vulkanStateKey(cmd.PPipeline.Read(ctx, cmd, s, nil)))

	case *VkCreateShaderModule:
		addWrite(&b, g, vulkanStateKey(cmd.PShaderModule.Read(ctx, cmd, s, nil)))

	case *RecreateShaderModule:
		addWrite(&b, g, vulkanStateKey(cmd.PShaderModule.Read(ctx, cmd, s, nil)))

	case *VkDestroyShaderModule:
		addModify(&b, g, vulkanStateKey(cmd.ShaderModule))
		b.KeepAlive = true

	case *VkAllocateCommandBuffers:
		count := uint64(cmd.PAllocateInfo.Read(ctx, cmd, s, nil).CommandBufferCount)
		cmdBufs := cmd.PCommandBuffers.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			cmdBuf := p.getOrCreateCommandBuffer(cmdBufs.Index(i, l).Read(ctx, cmd, s, nil))
			addWrite(&b, g, cmdBuf)
		}

	case *VkResetCommandBuffer:
		cmdBuf := p.getOrCreateCommandBuffer(cmd.CommandBuffer)
		addRead(&b, g, cmdBuf.handle)
		addWrite(&b, g, cmdBuf.records)
		cmdBuf.records.Commands = []*recordedCommand{}

	case *VkFreeCommandBuffers:
		count := uint64(cmd.CommandBufferCount)
		cmdBufs := cmd.PCommandBuffers.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			cmdBuf := p.getOrCreateCommandBuffer(cmdBufs.Index(i, l).Read(ctx, cmd, s, nil))
			addModify(&b, g, cmdBuf)
		}
		b.KeepAlive = true

	// CommandBuffer Commands:
	case *VkCmdCopyImage:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		// TODO(qining): Track all the memory ranges
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, srcBindings,
			dstBindings, emptyMemoryBindings)

	case *RecreateCmdCopyImage:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		// TODO(qining): Track all the memory ranges
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, srcBindings,
			dstBindings, emptyMemoryBindings)

	case *VkCmdCopyImageToBuffer:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			srcBindings, dstBindings, emptyMemoryBindings)

	case *RecreateCmdCopyImageToBuffer:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			srcBindings, dstBindings, emptyMemoryBindings)

	case *VkCmdCopyBufferToImage:
		srcBindings := readBufferHandleAndGetBindings(&b, cmd.SrcBuffer)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			srcBindings, dstBindings, emptyMemoryBindings)

	case *RecreateCmdCopyBufferToImage:
		srcBindings := readBufferHandleAndGetBindings(&b, cmd.SrcBuffer)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			srcBindings, dstBindings, emptyMemoryBindings)

	case *VkCmdCopyBuffer:
		srcBindings := readBufferHandleAndGetBindings(&b, cmd.SrcBuffer)
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			srcBindings, dstBindings, emptyMemoryBindings)

	case *RecreateCmdCopyBuffer:
		srcBindings := readBufferHandleAndGetBindings(&b, cmd.SrcBuffer)
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			srcBindings, dstBindings, emptyMemoryBindings)

	case *VkCmdBlitImage:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		// TODO(qining): Track all the memory ranges
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, srcBindings,
			dstBindings, emptyMemoryBindings)

	case *RecreateCmdBlitImage:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		// TODO(qining): Track all the memory ranges
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, srcBindings,
			dstBindings, emptyMemoryBindings)

	case *VkCmdResolveImage:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		// TODO(qining): Track all the memory ranges
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, srcBindings,
			dstBindings, emptyMemoryBindings)

	case *RecreateCmdResolveImage:
		srcBindings := readImageHandleAndGetBindings(&b, cmd.SrcImage)
		dstBindings := readImageHandleAndGetBindings(&b, cmd.DstImage)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		// TODO(qining): Track all the memory ranges
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, srcBindings,
			dstBindings, emptyMemoryBindings)

	case *VkCmdFillBuffer:
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
			dstBindings, emptyMemoryBindings)

	case *RecreateCmdFillBuffer:
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
			dstBindings, emptyMemoryBindings)

	case *VkCmdUpdateBuffer:
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
			dstBindings, emptyMemoryBindings)

	case *RecreateCmdUpdateBuffer:
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
			dstBindings, emptyMemoryBindings)

	case *VkCmdCopyQueryPoolResults:
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
			dstBindings, emptyMemoryBindings)

	case *RecreateCmdCopyQueryPoolResults:
		dstBindings := readBufferHandleAndGetBindings(&b, cmd.DstBuffer)
		// Be conservative here. Without tracking all the memory ranges and
		// calculating the memory according to the copy region, we cannot assume
		// this command overwrites the data. So it is labelled as 'modify' to
		// kept the previous writes
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
			dstBindings, emptyMemoryBindings)

	case *VkCmdBindVertexBuffers:
		count := cmd.BindingCount
		buffers := cmd.PBuffers.Slice(0, uint64(count), l)
		for i := uint64(0); i < uint64(count); i++ {
			buffer := buffers.Index(i, l).Read(ctx, cmd, s, nil)
			bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
			recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
				// As the LastBoundQueue of the buffer object has will change, so it is
				// a 'modify' instead of a 'read'
				addModify(b, g, vulkanStateKey(buffer))
				// Read the vertex buffer memory data here.
				readMemoryBindingsData(b, bufferBindings)
			})
		}

	case *RecreateCmdBindVertexBuffers:
		count := cmd.BindingCount
		buffers := cmd.PBuffers.Slice(0, uint64(count), l)
		for i := uint64(0); i < uint64(count); i++ {
			buffer := buffers.Index(i, l).Read(ctx, cmd, s, nil)
			bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
			recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
				// As the LastBoundQueue of the buffer object has will change, so it is
				// a 'modify' instead of a 'read'
				addModify(b, g, vulkanStateKey(buffer))
				// Read the vertex buffer memory data here.
				readMemoryBindingsData(b, bufferBindings)
			})
		}

	case *VkCmdBindIndexBuffer:
		buffer := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
			// As the LastBoundQueue of the buffer object has will change, so it is
			// a 'modify' instead of a 'read'
			addModify(b, g, vulkanStateKey(buffer))
			// Read the index buffer memory data here.
			readMemoryBindingsData(b, bufferBindings)
		})

	case *RecreateCmdBindIndexBuffer:
		buffer := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
			// As the LastBoundQueue of the buffer object has will change, so it is
			// a 'modify' instead of a 'read'
			addModify(b, g, vulkanStateKey(buffer))
			// Read the index buffer memory data here.
			readMemoryBindingsData(b, bufferBindings)
		})

	case *VkCmdDraw:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdDraw:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdDrawIndexed:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdDrawIndexed:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdDrawIndirect:
		indirectBuf := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, indirectBuf)
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			bufferBindings, emptyMemoryBindings, emptyMemoryBindings)

	case *RecreateCmdDrawIndirect:
		indirectBuf := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, indirectBuf)
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			bufferBindings, emptyMemoryBindings, emptyMemoryBindings)

	case *VkCmdDrawIndexedIndirect:
		indirectBuf := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, indirectBuf)
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			bufferBindings, emptyMemoryBindings, emptyMemoryBindings)

	case *RecreateCmdDrawIndexedIndirect:
		indirectBuf := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, indirectBuf)
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			bufferBindings, emptyMemoryBindings, emptyMemoryBindings)

	case *VkCmdDispatch:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdDispatch:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdDispatchIndirect:
		buffer := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			bufferBindings, emptyMemoryBindings, emptyMemoryBindings)

	case *RecreateCmdDispatchIndirect:
		buffer := cmd.Buffer
		bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
		recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
			bufferBindings, emptyMemoryBindings, emptyMemoryBindings)

	case *VkCmdBeginRenderPass:
		beginInfo := cmd.PRenderPassBegin.Read(ctx, cmd, s, nil)
		framebuffer := beginInfo.Framebuffer
		addRead(&b, g, vulkanStateKey(framebuffer))
		renderpass := beginInfo.RenderPass
		addRead(&b, g, vulkanStateKey(renderpass))

		if GetState(s).Framebuffers.Contains(framebuffer) {
			atts := GetState(s).Framebuffers.Get(framebuffer).ImageAttachments
			if GetState(s).RenderPasses.Contains(renderpass) {
				attDescs := GetState(s).RenderPasses.Get(renderpass).AttachmentDescriptions
				for i := uint32(0); i < uint32(len(atts)); i++ {
					img := atts.Get(i).Image.VulkanHandle
					// This can be wrong as this is getting all the memory bindings
					// that OVERLAP with the attachment image, so extra memories might be
					// covered. However in practical, image should be bound to only one
					// memory binding as a whole. So here should be a problem.
					// TODO: Use intersection operation to get the memory ranges
					imgBindings := getOverlappedBindingsForImage(img)
					loadOp := attDescs.Get(i).LoadOp
					storeOp := attDescs.Get(i).StoreOp

					if (loadOp != VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD) &&
						(storeOp != VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE) {
						// If the loadOp is not LOAD, and the storeOp is not DONT_CARE, the
						// render target attachment's data should be overwritten later.
						recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
							emptyMemoryBindings, emptyMemoryBindings, imgBindings)
					} else if (loadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD) &&
						(storeOp != VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE) {
						// If the loadOp is LOAD, and the storeOp is not DONT_CARE, the
						// render target attachment should be 'modified'.
						recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
							emptyMemoryBindings, imgBindings, emptyMemoryBindings)
					} else if (loadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD) &&
						(storeOp == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE) {
						// If the storeOp is DONT_CARE, and the loadOp is LOAD, the render target
						// attachment should be 'read'.
						recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, imgBindings,
							emptyMemoryBindings, emptyMemoryBindings)
					}
					// If the LoadOp is not LOAD and the storeOp is DONT_CARE, no operation
					// must be done to the attahcment then.
					// TODO(qining): Actually we should disable all the 'write', 'modify'
					// behaviour in this render pass.
				}
			}
		}

	case *RecreateCmdBeginRenderPass:

		beginInfo := cmd.PRenderPassBegin.Read(ctx, cmd, s, nil)
		framebuffer := beginInfo.Framebuffer
		addRead(&b, g, vulkanStateKey(framebuffer))
		renderpass := beginInfo.RenderPass
		addRead(&b, g, vulkanStateKey(renderpass))

		if GetState(s).Framebuffers.Contains(framebuffer) {
			atts := GetState(s).Framebuffers.Get(framebuffer).ImageAttachments
			if GetState(s).RenderPasses.Contains(renderpass) {
				attDescs := GetState(s).RenderPasses.Get(renderpass).AttachmentDescriptions
				for i := uint32(0); i < uint32(len(atts)); i++ {
					img := atts.Get(i).Image.VulkanHandle
					// This can be wrong as this is getting all the memory bindings
					// that OVERLAP with the attachment image, so extra memories might be
					// covered. However in practical, image should be bound to only one
					// memory binding as a whole. So here should be a problem.
					// TODO: Use intersection operation to get the memory ranges
					imgBindings := getOverlappedBindingsForImage(img)
					loadOp := attDescs.Get(i).LoadOp
					storeOp := attDescs.Get(i).StoreOp

					if (loadOp != VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD) &&
						(storeOp != VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE) {
						// If the loadOp is not LOAD, and the storeOp is not DONT_CARE, the
						// render target attachment's data should be overwritten later.
						recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
							emptyMemoryBindings, emptyMemoryBindings, imgBindings)
					} else if (loadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD) &&
						(storeOp != VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE) {
						// If the loadOp is LOAD, and the storeOp is not DONT_CARE, the
						// render target attachment should be 'modified'.
						recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer,
							emptyMemoryBindings, imgBindings, emptyMemoryBindings)
					} else if (loadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD) &&
						(storeOp == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE) {
						// If the storeOp is DONT_CARE, and the loadOp is LOAD, the render target
						// attachment should be 'read'.
						recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, imgBindings,
							emptyMemoryBindings, emptyMemoryBindings)
					}
					// If the LoadOp is not LOAD and the storeOp is DONT_CARE, no operation
					// must be done to the attahcment then.
					// TODO(qining): Actually we should disable all the 'write', 'modify'
					// behaviour in this render pass.
				}
			}
		}

	case *VkCmdEndRenderPass:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdEndRenderPass:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdNextSubpass:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdNextSubpass:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdPushConstants:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdPushConstants:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetLineWidth:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetLineWidth:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetScissor:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetScissor:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetViewport:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetViewport:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdBindDescriptorSets:
		descriptorSetCount := cmd.DescriptorSetCount
		descriptorSets := cmd.PDescriptorSets.Slice(0, uint64(descriptorSetCount), l)
		for i := uint32(0); i < descriptorSetCount; i++ {
			descriptorSet := descriptorSets.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			addRead(&b, g, vulkanStateKey(descriptorSet))
			if GetState(s).DescriptorSets.Contains(descriptorSet) {
				for _, descBinding := range GetState(s).DescriptorSets.Get(descriptorSet).Bindings {
					for _, bufferInfo := range descBinding.BufferBinding {
						buf := bufferInfo.Buffer

						recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
							// Descriptors might be modified
							addModify(b, g, vulkanStateKey(buf))
							// Advance the read/modify behavior of the descriptors from
							// draw and dispatch calls to here. Details in the handling
							// of vkCmdDispatch and vkCmdDraw.
							modifyMemoryBindingsData(b, getOverlappedBindingsForBuffer(buf))
						})
					}
					for _, imageInfo := range descBinding.ImageBinding {
						view := imageInfo.ImageView

						recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
							addRead(b, g, vulkanStateKey(view))
							if GetState(s).ImageViews.Contains(view) {
								img := GetState(s).ImageViews.Get(view).Image.VulkanHandle
								// Advance the read/modify behavior of the descriptors from
								// draw and dispatch calls to here. Details in the handling
								// of vkCmdDispatch and vkCmdDraw.
								readMemoryBindingsData(b, getOverlappedBindingsForImage(img))
							}
						})
					}
					for _, bufferView := range descBinding.BufferViewBindings {

						recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
							addRead(b, g, vulkanStateKey(bufferView))
							if GetState(s).BufferViews.Contains(bufferView) {
								buf := GetState(s).BufferViews.Get(bufferView).Buffer.VulkanHandle
								// Advance the read/modify behavior of the descriptors from
								// draw and dispatch calls to here. Details in the handling
								// of vkCmdDispatch and vkCmdDraw.
								readMemoryBindingsData(b, getOverlappedBindingsForBuffer(buf))
							}
						})
					}
				}
			}
		}

	case *RecreateCmdBindDescriptorSets:
		descriptorSetCount := cmd.DescriptorSetCount
		descriptorSets := cmd.PDescriptorSets.Slice(0, uint64(descriptorSetCount), l)
		for i := uint32(0); i < descriptorSetCount; i++ {
			descriptorSet := descriptorSets.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			addRead(&b, g, vulkanStateKey(descriptorSet))
			if GetState(s).DescriptorSets.Contains(descriptorSet) {
				for _, descBinding := range GetState(s).DescriptorSets.Get(descriptorSet).Bindings {
					for _, bufferInfo := range descBinding.BufferBinding {
						buf := bufferInfo.Buffer

						recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
							// Descriptors might be modified
							addModify(b, g, vulkanStateKey(buf))
						})
					}
					for _, imageInfo := range descBinding.ImageBinding {
						view := imageInfo.ImageView

						recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
							addRead(b, g, vulkanStateKey(view))
						})
					}
					for _, bufferView := range descBinding.BufferViewBindings {

						recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
							addRead(b, g, vulkanStateKey(bufferView))
						})
					}
				}
			}
		}

	case *VkBeginCommandBuffer:
		cmdbuf := p.getOrCreateCommandBuffer(cmd.CommandBuffer)
		addRead(&b, g, cmdbuf.handle)
		addWrite(&b, g, cmdbuf.records)

	case *VkEndCommandBuffer:
		cmdbuf := p.getOrCreateCommandBuffer(cmd.CommandBuffer)
		addModify(&b, g, cmdbuf)

	case *RecreateAndBeginCommandBuffer:
		cmdbuf := p.getOrCreateCommandBuffer(cmd.PCommandBuffer.Read(ctx, cmd, s, nil))
		addWrite(&b, g, cmdbuf)

	case *RecreateEndCommandBuffer:
		cmdbuf := p.getOrCreateCommandBuffer(cmd.CommandBuffer)
		addModify(&b, g, cmdbuf)

	case *VkCmdPipelineBarrier:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		bufferBarrierCount := cmd.BufferMemoryBarrierCount
		bufferBarriers := cmd.PBufferMemoryBarriers.Slice(0, uint64(bufferBarrierCount), l)
		imageBarrierCount := cmd.ImageMemoryBarrierCount
		imageBarriers := cmd.PImageMemoryBarriers.Slice(0, uint64(imageBarrierCount), l)
		for i := uint64(0); i < uint64(bufferBarrierCount); i++ {
			bufferBarrier := bufferBarriers.Index(i, l).Read(ctx, cmd, s, nil)
			buffer := bufferBarrier.Buffer
			// Getting the bindings for the whole buffer is conservative, as the
			// barrier may only affect a region of the buffer specified by offset
			// and size.
			bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
			recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
				bufferBindings, emptyMemoryBindings)
		}
		for i := uint64(0); i < uint64(imageBarrierCount); i++ {
			imageBarrier := imageBarriers.Index(i, l).Read(ctx, cmd, s, nil)
			image := imageBarrier.Image
			// Getting the bindings for the whole image is conservative, as the
			// barrier may only affect a region of the image specified by
			// subresourceRange.
			imageBindings := readImageHandleAndGetBindings(&b, image)
			recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
				imageBindings, emptyMemoryBindings)
		}

	case *RecreateCmdPipelineBarrier:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		bufferBarrierCount := cmd.BufferMemoryBarrierCount
		bufferBarriers := cmd.PBufferMemoryBarriers.Slice(0, uint64(bufferBarrierCount), l)
		imageBarrierCount := cmd.ImageMemoryBarrierCount
		imageBarriers := cmd.PImageMemoryBarriers.Slice(0, uint64(imageBarrierCount), l)
		for i := uint64(0); i < uint64(bufferBarrierCount); i++ {
			bufferBarrier := bufferBarriers.Index(i, l).Read(ctx, cmd, s, nil)
			buffer := bufferBarrier.Buffer
			// Getting the bindings for the whole buffer is conservative, as the
			// barrier may only affect a region of the buffer specified by offset
			// and size.
			bufferBindings := readBufferHandleAndGetBindings(&b, buffer)
			recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
				bufferBindings, emptyMemoryBindings)
		}
		for i := uint64(0); i < uint64(imageBarrierCount); i++ {
			imageBarrier := imageBarriers.Index(i, l).Read(ctx, cmd, s, nil)
			image := imageBarrier.Image
			// Getting the bindings for the whole image is conservative, as the
			// barrier may only affect a region of the image specified by
			// subresourceRange.
			imageBindings := readImageHandleAndGetBindings(&b, image)
			recordTouchingMemoryBindingsData(&b, cmd.CommandBuffer, emptyMemoryBindings,
				imageBindings, emptyMemoryBindings)
		}

	case *VkCmdBindPipeline:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
			addRead(b, g, vulkanStateKey(cmd.Pipeline))
		})
		addRead(&b, g, vulkanStateKey(cmd.Pipeline))

	case *RecreateCmdBindPipeline:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {
			addRead(b, g, vulkanStateKey(cmd.Pipeline))
		})
		addRead(&b, g, vulkanStateKey(cmd.Pipeline))

	case *VkCmdBeginQuery:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdBeginQuery:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdEndQuery:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdEndQuery:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdResetQueryPool:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdResetQueryPool:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdWriteTimestamp:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdWriteTimestamp:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdClearAttachments:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdClearAttachments:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the case that the attachment is fully cleared.

	case *VkCmdClearColorImage:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the color image

	case *RecreateCmdClearColorImage:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the color image

	case *VkCmdClearDepthStencilImage:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the depth/stencil image

	case *RecreateCmdClearDepthStencilImage:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the depth/stencil image

	case *VkCmdSetDepthBias:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetDepthBias:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetDepthBounds:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetDepthBounds:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetBlendConstants:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetBlendConstants:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetEvent:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetEvent:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdResetEvent:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdResetEvent:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdWaitEvents:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the image and buffer memory barriers?

	case *RecreateCmdWaitEvents:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})
		//TODO: handle the image and buffer memory barriers?

	case *VkCmdSetStencilCompareMask:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetStencilCompareMask:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetStencilWriteMask:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetStencilWriteMask:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdSetStencilReference:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *RecreateCmdSetStencilReference:
		recordCommand(&b, cmd.CommandBuffer, func(b *dependencygraph.AtomBehaviour) {})

	case *VkCmdExecuteCommands:
		secondaryCmdBufs := cmd.PCommandBuffers.Slice(0, uint64(cmd.CommandBufferCount), l)
		for i := uint32(0); i < cmd.CommandBufferCount; i++ {
			secondaryCmdBuf := secondaryCmdBufs.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			scb := p.getOrCreateCommandBuffer(secondaryCmdBuf)
			addRead(&b, g, scb)
			recordSecondaryCommandBuffer(&b, cmd.CommandBuffer, scb)
		}

	case *RecreateCmdExecuteCommands:
		secondaryCmdBufs := cmd.PCommandBuffers.Slice(0, uint64(cmd.CommandBufferCount), l)
		for i := uint32(0); i < cmd.CommandBufferCount; i++ {
			secondaryCmdBuf := secondaryCmdBufs.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			scb := p.getOrCreateCommandBuffer(secondaryCmdBuf)
			addRead(&b, g, scb)
			recordSecondaryCommandBuffer(&b, cmd.CommandBuffer, scb)
		}

	// CommandBuffer commands execution triggering commands:
	case *VkQueueSubmit:
		// Queue submit atom should always be alive
		b.KeepAlive = true

		// handle queue
		addModify(&b, g, vulkanStateKey(cmd.Queue))

		// handle command buffers
		submitCount := cmd.SubmitCount
		submits := cmd.PSubmits.Slice(0, uint64(submitCount), l)
		p.submissions[cmd] = []submitInfo{}
		for i := uint32(0); i < submitCount; i++ {
			p.submissions[cmd] = append(p.submissions[cmd], submitInfo{})
			submit := submits.Index(uint64(i), l).Read(ctx, cmd, s, nil)
			commandBufferCount := submit.CommandBufferCount
			commandBuffers := submit.PCommandBuffers.Slice(0, uint64(commandBufferCount), l)
			for j := uint32(0); j < submit.CommandBufferCount; j++ {
				vkCmdBuf := commandBuffers.Index(uint64(j), l).Read(ctx, cmd, s, nil)
				cb := p.getOrCreateCommandBuffer(vkCmdBuf)
				p.submissions[cmd][i].commandBuffers = append(p.submissions[cmd][i].commandBuffers, cb)
				// All the commands that are submitted will not be dropped.
				addRead(&b, g, cb)
			}
		}
		debug("\tvkQueueSubmit: Executed Commands: %v", executedCommands)
		p.rollOutBehavioursForExecutedCommands(&b, executedCommands)

	case *VkSetEvent:
		b.KeepAlive = true
		debug("\tvkSetEvent Executed Commands: %v", executedCommands)
		p.rollOutBehavioursForExecutedCommands(&b, executedCommands)

	// Keep-alive commands:
	case *VkQueuePresentKHR:
		addRead(&b, g, vulkanStateKey(cmd.Queue))
		g.SetRoot(vulkanStateKey(cmd.Queue))
		b.KeepAlive = true

	case *VkCreateInstance,
		*RecreateInstance,
		*VkDestroyInstance,
		*VkCreateDevice,
		*VkDestroyDevice,
		*RecreateDevice,
		*VkEnumerateDeviceExtensionProperties,
		*VkEnumerateDeviceLayerProperties,
		*VkEnumerateInstanceExtensionProperties,
		*VkEnumerateInstanceLayerProperties,
		*VkEnumeratePhysicalDevices,
		*VkGetDeviceProcAddr,
		*VkGetInstanceProcAddr,
		*VkGetDeviceQueue,
		*RecreateQueue,
		*VkGetPhysicalDeviceSparseImageFormatProperties,
		*VkGetPhysicalDeviceFeatures,
		*VkGetPhysicalDeviceFormatProperties,
		*VkGetPhysicalDeviceImageFormatProperties,
		*VkGetPhysicalDeviceMemoryProperties,
		*VkGetPhysicalDeviceProperties,
		*VkGetPhysicalDeviceQueueFamilyProperties,
		*RecreatePhysicalDevices,
		*RecreatePhysicalDeviceProperties,
		*VkAcquireNextImageKHR,
		*VkCreateCommandPool,
		*RecreateCommandPool,
		*VkDestroyCommandPool,
		*VkResetCommandPool,
		*VkCreateDescriptorPool,
		*RecreateDescriptorPool,
		*VkDestroyDescriptorPool,
		*VkResetDescriptorPool,
		*VkCreateDescriptorSetLayout,
		*RecreateDescriptorSetLayout,
		*VkDestroyDescriptorSetLayout,
		*VkAllocateDescriptorSets,
		*VkCreateSampler,
		*RecreateSampler,
		*VkDestroySampler,
		*VkCreateSwapchainKHR,
		*RecreateSwapchain,
		*VkDestroySwapchainKHR,
		*VkGetSwapchainImagesKHR,
		*VkCreatePipelineLayout,
		*RecreatePipelineLayout,
		*VkDestroyPipelineLayout,
		*VkCreatePipelineCache,
		*RecreatePipelineCache,
		*VkDestroyPipelineCache,
		*VkGetPipelineCacheData,
		*VkMergePipelineCaches,
		*VkCreateQueryPool,
		*RecreateQueryPool,
		*VkDestroyQueryPool,
		*VkGetQueryPoolResults,
		*VkQueueBindSparse,
		// Synchronizations
		*VkCreateEvent,
		*RecreateEvent,
		*VkResetEvent,
		*VkDestroyEvent,
		*VkGetEventStatus,
		*VkCreateSemaphore,
		*RecreateSemaphore,
		*VkDestroySemaphore,
		*VkCreateFence,
		*VkDestroyFence,
		*RecreateFence,
		*VkGetFenceStatus,
		*VkResetFences,
		*VkWaitForFences,
		*VkDeviceWaitIdle,
		*VkQueueWaitIdle,
		// Surfaces creation
		*VkCreateAndroidSurfaceKHR,
		*RecreateAndroidSurfaceKHR,
		*VkCreateWaylandSurfaceKHR,
		*RecreateWaylandSurfaceKHR,
		*VkCreateWin32SurfaceKHR,
		*RecreateWin32SurfaceKHR,
		*VkCreateXcbSurfaceKHR,
		*RecreateXCBSurfaceKHR,
		*VkCreateXlibSurfaceKHR,
		*RecreateXlibSurfaceKHR,
		*VkDestroySurfaceKHR,
		// VK_KHR_display extension related
		*VkCreateDisplayModeKHR,
		*VkCreateDisplayPlaneSurfaceKHR,
		*VkCreateSharedSwapchainsKHR,
		*VkGetDisplayModePropertiesKHR,
		*VkGetDisplayPlaneCapabilitiesKHR,
		*VkGetDisplayPlaneSupportedDisplaysKHR,
		*VkGetPhysicalDeviceDisplayPlanePropertiesKHR,
		*VkGetPhysicalDeviceDisplayPropertiesKHR,
		*VkGetPhysicalDeviceMirPresentationSupportKHR,
		*VkGetPhysicalDeviceSurfaceCapabilitiesKHR,
		*VkGetPhysicalDeviceSurfaceFormatsKHR,
		*VkGetPhysicalDeviceSurfacePresentModesKHR,
		*VkGetPhysicalDeviceSurfaceSupportKHR,
		*VkGetPhysicalDeviceWaylandPresentationSupportKHR,
		*VkGetPhysicalDeviceWin32PresentationSupportKHR,
		*VkGetPhysicalDeviceXcbPresentationSupportKHR,
		*VkGetPhysicalDeviceXlibPresentationSupportKHR:
		b.KeepAlive = true

	default:
		log.E(ctx, "Command not handled: %v", cmd)
		b.Aborted = true
	}
	return b
}

func (p *VulkanDependencyGraphBehaviourProvider) rollOutBehavioursForExecutedCommands(
	b *dependencygraph.AtomBehaviour, executedCommands []executedCommandIndex) {
	for _, e := range executedCommands {
		submit := e.submit
		si := e.Indices[0]
		cbi := e.Indices[1]
		ci := e.Indices[2]
		command := p.submissions[submit][si].commandBuffers[cbi].records.Commands[ci]
		behaviours := command.recordedBehaviours
		// Handle secondary command buffers
		if len(e.Indices) == 5 {
			scbi := e.Indices[3]
			sci := e.Indices[4]
			behaviours = command.secondaryCommandBuffers[scbi].records.Commands[sci].recordedBehaviours
		}
		for _, rb := range behaviours {
			rb(b)
		}
	}
}

// Traverse through the given VkWriteDescriptorSet slice, add behaviors to
// |b| according to the descriptor type.
func processDescriptorWrites(writes VkWriteDescriptorSetˢ, b *dependencygraph.AtomBehaviour, g *dependencygraph.DependencyGraph, ctx context.Context, cmd api.Cmd, s *api.State) error {
	l := s.MemoryLayout
	writeCount := writes.count
	for i := uint64(0); i < writeCount; i++ {
		write := writes.Index(uint64(i), l).Read(ctx, cmd, s, nil)
		if write.DescriptorCount > 0 {
			// handle the target descriptor set
			b.Modify(g, vulkanStateKey(write.DstSet))
			switch write.DescriptorType {
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
				imageInfos := write.PImageInfo.Slice(0, uint64(write.DescriptorCount), l)
				for j := uint64(0); j < imageInfos.count; j++ {
					imageInfo := imageInfos.Index(uint64(j), l).Read(ctx, cmd, s, nil)
					sampler := imageInfo.Sampler
					imageView := imageInfo.ImageView
					b.Read(g, vulkanStateKey(sampler))
					b.Read(g, vulkanStateKey(imageView))
				}
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
				bufferInfos := write.PBufferInfo.Slice(0, uint64(write.DescriptorCount), l)
				for j := uint64(0); j < bufferInfos.count; j++ {
					bufferInfo := bufferInfos.Index(uint64(j), l).Read(ctx, cmd, s, nil)
					buffer := bufferInfo.Buffer
					b.Read(g, vulkanStateKey(buffer))
				}
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
				bufferViews := write.PTexelBufferView.Slice(0, uint64(write.DescriptorCount), l)
				for j := uint64(0); j < bufferViews.count; j++ {
					bufferView := bufferViews.Index(uint64(j), l).Read(ctx, cmd, s, nil)
					b.Read(g, vulkanStateKey(bufferView))
				}
			default:
				return fmt.Errorf("Unhandled DescriptorType: %v", write.DescriptorType)
			}
		}
	}
	return nil
}
