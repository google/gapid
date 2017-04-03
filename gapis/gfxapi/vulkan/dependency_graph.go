package vulkan

import (
	"context"
	"fmt"

	"github.com/google/gapid/core/app/benchmark"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/atom"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/database"
	"github.com/google/gapid/gapis/gfxapi"
)

var dependencyGraphBuildCounter = benchmark.GlobalCounters.Duration("dependencyGraph.build")

type StateAddress uint32

///////////// To conform with the dce interface of GLES ////////////
type stateKey interface {
	Parent() stateKey
}

type vulkanStateKey uint64

func (h vulkanStateKey) Parent() stateKey {
	return nil
}

// Device memory hierarchy
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

type vulkanDeviceMemoryData struct {
	binding *vulkanDeviceMemoryBinding
}

func (m *vulkanDeviceMemory) Parent() stateKey {
	return nil
}

func (h *vulkanDeviceMemoryHandle) Parent() stateKey {
	return h.memory
}

func (b *vulkanDeviceMemoryBinding) Parent() stateKey {
	return b.memory
}

func (d *vulkanDeviceMemoryData) Parent() stateKey {
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

// Command buffer hierachy:
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
	Commands      []func(b *AtomBehaviour)
}

func newVulkanCommandBuffer(handle VkCommandBuffer) *vulkanCommandBuffer {
	cb := &vulkanCommandBuffer{handle: nil, records: nil}
	cb.handle = &vulkanCommandBufferHandle{CommandBuffer: cb, vkCommandBuffer: handle}
	cb.records = &vulkanRecordedCommands{CommandBuffer: cb, Commands: []func(b *AtomBehaviour){}}
	return cb
}

func (cb *vulkanCommandBuffer) Parent() stateKey {
	return nil
}

func (h *vulkanCommandBufferHandle) Parent() stateKey {
	return h.CommandBuffer
}

func (c *vulkanRecordedCommands) Parent() stateKey {
	return c.CommandBuffer
}

func (c *vulkanRecordedCommands) appendCommand(f func(b *AtomBehaviour)) *vulkanRecordedCommands {
	c.Commands = append(c.Commands, f)
	return c
}

///////////// End of interface conformance part ////////////////////

//////////////////// Unchanged part from GLES //////////////////////
const nullStateAddress = StateAddress(0)

type DependencyGraph struct {
	atoms          []atom.Atom           // Atom list which this graph was build for.
	behaviours     []AtomBehaviour       // State reads/writes for each atom (graph edges).
	roots          map[StateAddress]bool // State to mark live at requested atoms.
	addressMap     addressMapping        // Remap state keys to integers for performance.
	deviceMemories map[VkDeviceMemory]*vulkanDeviceMemory
	commandBuffers map[VkCommandBuffer]*vulkanCommandBuffer
}

type AtomBehaviour struct {
	Read      []StateAddress // State read by an atom.
	Modify    []StateAddress // State read and written by an atom.
	Write     []StateAddress // State written by an atom.
	KeepAlive bool           // Force the atom to be live.
	Aborted   bool           // Mutation of this command aborts.
}

type addressMapping struct {
	address map[stateKey]StateAddress
	key     map[StateAddress]stateKey
	parent  map[StateAddress]StateAddress
}

func (g *DependencyGraph) Print(ctx context.Context, b *AtomBehaviour) {
	for _, read := range b.Read {
		key := g.addressMap.key[read]
		log.I(ctx, " - read [%v]%T%+v", read, key, key)
	}
	for _, modify := range b.Modify {
		key := g.addressMap.key[modify]
		log.I(ctx, " - modify [%v]%T%+v", modify, key, key)
	}
	for _, write := range b.Write {
		key := g.addressMap.key[write]
		log.I(ctx, " - write [%v]%T%+v", write, key, key)
	}
	if b.Aborted {
		log.I(ctx, " - aborted")
	}
}

func (g *DependencyGraph) getOrCreateDeviceMemory(handle VkDeviceMemory) *vulkanDeviceMemory {
	if m, ok := g.deviceMemories[handle]; ok {
		return m
	}
	newM := newVulkanDeviceMemory(handle)
	g.deviceMemories[handle] = newM
	return newM
}

func (g *DependencyGraph) getOrCreateCommandBuffer(handle VkCommandBuffer) *vulkanCommandBuffer {
	if cb, ok := g.commandBuffers[handle]; ok {
		return cb
	}
	newCb := newVulkanCommandBuffer(handle)
	g.commandBuffers[handle] = newCb
	return newCb
}

// The public accessible entrance of building a dep graph from atom list
func GetDependencyGraph(ctx context.Context) (*DependencyGraph, error) {
	r, err := database.Build(ctx, &DependencyGraphResolvable{Capture: capture.Get(ctx)})
	if err != nil {
		return nil, fmt.Errorf("Could not calculate dependency graph: %v", err)
	}
	return r.(*DependencyGraph), nil
}

// The real entrance of dep graph building
func (r *DependencyGraphResolvable) Resolve(ctx context.Context) (interface{}, error) {
	c, err := capture.ResolveFromPath(ctx, r.Capture)
	if err != nil {
		return nil, err
	}
	atoms, err := c.Atoms(ctx)
	if err != nil {
		return nil, err
	}

	g := &DependencyGraph{
		atoms:      atoms.Atoms,
		behaviours: make([]AtomBehaviour, len(atoms.Atoms)),
		roots:      map[StateAddress]bool{},
		addressMap: addressMapping{
			address: map[stateKey]StateAddress{nil: nullStateAddress},
			key:     map[StateAddress]stateKey{nullStateAddress: nil},
			parent:  map[StateAddress]StateAddress{nullStateAddress: nullStateAddress},
		},
		deviceMemories: map[VkDeviceMemory]*vulkanDeviceMemory{},
		commandBuffers: map[VkCommandBuffer]*vulkanCommandBuffer{},
	}

	s := c.NewState()
	t0 := dependencyGraphBuildCounter.Start()
	for i, a := range g.atoms {
		g.behaviours[i] = g.getBehaviour(ctx, s, atom.ID(i), a)
	}
	dependencyGraphBuildCounter.Stop(t0)
	return g, nil
}

// Using Vulkan handle as the GLES state key.
// State address is assigned in the function addressOf() and
// used as the identity of handle (vulkan object) in the dep graph

func (m *addressMapping) addressOf(state stateKey) StateAddress {
	if a, ok := m.address[state]; ok {
		return a
	}
	address := StateAddress(len(m.address))
	m.address[state] = address
	m.key[address] = state
	m.parent[address] = m.addressOf(state.Parent())
	return address
}

func (b *AtomBehaviour) read(g *DependencyGraph, state stateKey) {
	if state != nil {
		b.Read = append(b.Read, g.addressMap.addressOf(state))
	}
}

func (b *AtomBehaviour) modify(g *DependencyGraph, state stateKey) {
	if state != nil {
		b.Modify = append(b.Modify, g.addressMap.addressOf(state))
	}
}

func (b *AtomBehaviour) write(g *DependencyGraph, state stateKey) {
	if state != nil {
		b.Write = append(b.Write, g.addressMap.addressOf(state))
	}
}

///////////////////// End of unchanged part //////////////////////////

// Build the corresponding dep graph node for a given atom
// Note this function is called on a new graphics state
func (g *DependencyGraph) getBehaviour(ctx context.Context, s *gfxapi.State, id atom.ID, a atom.Atom) AtomBehaviour {
	b := AtomBehaviour{}

	// Helper function that gets overlapped memory bindings with a given offset and size
	getOverlappingMemoryBindings := func(memory VkDeviceMemory,
		offset, size uint64) []*vulkanDeviceMemoryBinding {
		return g.getOrCreateDeviceMemory(memory).getOverlappedBindings(offset, size)
	}

	// Helper function that gets the overlapped memory bindings for a given image
	getOverlappedBindingsForImage := func(image VkImage) []*vulkanDeviceMemoryBinding {
		if GetState(s).Images.Get(image).IsSwapchainImage {
			return []*vulkanDeviceMemoryBinding{}
		} else if GetState(s).Images.Get(image).BoundMemory != nil {
			boundMemory := GetState(s).Images.Get(image).BoundMemory.VulkanHandle
			offset := uint64(GetState(s).Images.Get(image).BoundMemoryOffset)
			size := uint64(uint64(GetState(s).Images.Get(image).Size))
			return getOverlappingMemoryBindings(boundMemory, offset, size)
		} else {
			log.E(ctx, "Error Image: %v: Cannot get the bound memory for an image which has not been bound yet", image)
			return []*vulkanDeviceMemoryBinding{}
		}
	}

	// Helper function that gets the overlapped memory bindings for a given buffer
	getOverlappedBindingsForBuffer := func(buffer VkBuffer) []*vulkanDeviceMemoryBinding {
		if GetState(s).Buffers.Get(buffer).Memory != nil {
			boundMemory := GetState(s).Buffers.Get(buffer).Memory.VulkanHandle
			offset := uint64(GetState(s).Buffers.Get(buffer).MemoryOffset)
			size := uint64(uint64(GetState(s).Buffers.Get(buffer).Info.Size))
			return getOverlappingMemoryBindings(boundMemory, offset, size)
		} else {
			log.E(ctx, "Error Buffer: %v: Cannot get the bound memory for a buffer which has not been bound yet", buffer)
			return []*vulkanDeviceMemoryBinding{}
		}
	}

	// Helper function that 'read' the given memory bindings
	readMemoryBindingsData := func(pb *AtomBehaviour, bindings []*vulkanDeviceMemoryBinding) {
		for _, binding := range bindings {
			if config.DebugDeadCodeElimination {
				log.I(ctx, "Read binding data: %v <-  binding: %v <- memory: %v", g.addressMap.addressOf(binding.data), g.addressMap.addressOf(binding), g.addressMap.addressOf(binding.Parent()))
			}
			pb.read(g, binding.data)
		}
	}

	// Helper function that 'write' the given memory bindings
	writeMemoryBindingsData := func(pb *AtomBehaviour, bindings []*vulkanDeviceMemoryBinding) {
		for _, binding := range bindings {
			if config.DebugDeadCodeElimination {
				log.I(ctx, "Writing binding data: %v <- binding: %v <- memory: %v", g.addressMap.addressOf(binding.data), g.addressMap.addressOf(binding), g.addressMap.addressOf(binding.Parent()))
			}
			pb.write(g, binding.data)
		}
	}

	// Helper function that 'modify' the given memory bindings
	modifyMemoryBindingsData := func(pb *AtomBehaviour, bindings []*vulkanDeviceMemoryBinding) {
		for _, binding := range bindings {
			if config.DebugDeadCodeElimination {
				log.I(ctx, "Modifying binding data: %v <- binding: %v <- memory: %v", binding.data, g.addressMap.addressOf(binding.data), g.addressMap.addressOf(binding), g.addressMap.addressOf(binding.Parent()))
			}
			pb.modify(g, binding.data)
		}
	}

	// Helper function that records a given behavior, that to be carried out
	// when being submitted, to the command record list of a given command buffer
	recordCommand := func(handle VkCommandBuffer, c func(b *AtomBehaviour)) {
		cmdBuf := g.getOrCreateCommandBuffer(handle)
		cmdBuf.records.appendCommand(c)
	}

	// Mutate the state with the atom.
	if err := a.Mutate(ctx, s, nil); err != nil {
		log.E(ctx, "Atom %v %v: %v", id, a, err)
		return AtomBehaviour{Aborted: true}
	}

	// Add behaviors for the atom according to its type
	switch typedAtom := a.(type) {
	case *VkCreateImage:
		image := typedAtom.PImage.Read(ctx, a, s, nil)
		b.write(g, vulkanStateKey(image))

	case *VkCreateBuffer:
		buffer := typedAtom.PBuffer.Read(ctx, a, s, nil)
		b.write(g, vulkanStateKey(buffer))

	case *RecreateImage:
		image := typedAtom.PImage.Read(ctx, a, s, nil)
		b.write(g, vulkanStateKey(image))

	case *RecreateBuffer:
		buffer := typedAtom.PBuffer.Read(ctx, a, s, nil)
		b.write(g, vulkanStateKey(buffer))

	case *VkAllocateMemory:
		allocateInfo := typedAtom.PAllocateInfo.Read(ctx, a, s, nil)
		memory := typedAtom.PMemory.Read(ctx, a, s, nil)
		b.write(g, g.getOrCreateDeviceMemory(memory))

		// handle dedicated memory allocation
		if allocateInfo.PNext != (Voidᶜᵖ{}) {
			pNext := Voidᵖ(allocateInfo.PNext)
			for pNext != (Voidᵖ{}) {
				sType := (VkStructureTypeᶜᵖ(pNext)).Read(ctx, a, s, nil)
				switch sType {
				case VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV:
					ext := VkDedicatedAllocationMemoryAllocateInfoNVᵖ(pNext).Read(ctx, a, s, nil)
					image := ext.Image
					buffer := ext.Buffer
					if uint64(image) != 0 {
						b.read(g, vulkanStateKey(image))
					}
					if uint64(buffer) != 0 {
						b.read(g, vulkanStateKey(buffer))
					}
				}
				pNext = (VulkanStructHeaderᵖ(pNext)).Read(ctx, a, s, nil).PNext
			}
		}

	case *RecreateDeviceMemory:
		allocateInfo := typedAtom.PAllocateInfo.Read(ctx, a, s, nil)
		memory := typedAtom.PMemory.Read(ctx, a, s, nil)
		b.write(g, g.getOrCreateDeviceMemory(memory))

		// handle dedicated memory allocation
		if allocateInfo.PNext != (Voidᶜᵖ{}) {
			pNext := Voidᵖ(allocateInfo.PNext)
			for pNext != (Voidᵖ{}) {
				sType := (VkStructureTypeᶜᵖ(pNext)).Read(ctx, a, s, nil)
				switch sType {
				case VkStructureType_VK_STRUCTURE_TYPE_DEDICATED_ALLOCATION_MEMORY_ALLOCATE_INFO_NV:
					ext := VkDedicatedAllocationMemoryAllocateInfoNVᵖ(pNext).Read(ctx, a, s, nil)
					image := ext.Image
					buffer := ext.Buffer
					if uint64(image) != 0 {
						b.read(g, vulkanStateKey(image))
					}
					if uint64(buffer) != 0 {
						b.read(g, vulkanStateKey(buffer))
					}
				}
				pNext = (VulkanStructHeaderᵖ(pNext)).Read(ctx, a, s, nil).PNext
			}
		}

	case *VkBindImageMemory:
		image := typedAtom.Image
		memory := typedAtom.Memory
		b.modify(g, vulkanStateKey(image))
		b.read(g, g.getOrCreateDeviceMemory(memory).handle)
		offset := uint64(GetState(s).Images.Get(image).BoundMemoryOffset)
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
		size := uint64(GetState(s).Images.Get(image).Size)
		binding := g.getOrCreateDeviceMemory(memory).addBinding(offset, size)
		b.write(g, binding)

	case *VkBindBufferMemory:
		buffer := typedAtom.Buffer
		memory := typedAtom.Memory
		b.modify(g, vulkanStateKey(buffer))
		b.read(g, g.getOrCreateDeviceMemory(memory).handle)
		offset := uint64(GetState(s).Buffers.Get(buffer).MemoryOffset)
		size := uint64(GetState(s).Buffers.Get(buffer).Info.Size)
		binding := g.getOrCreateDeviceMemory(memory).addBinding(offset, size)
		b.write(g, binding)

	case *RecreateBindImageMemory:
		image := typedAtom.Image
		memory := typedAtom.Memory
		b.modify(g, vulkanStateKey(image))
		b.read(g, g.getOrCreateDeviceMemory(memory).handle)
		offset := uint64(GetState(s).Images.Get(image).BoundMemoryOffset)
		size := uint64(GetState(s).Images.Get(image).Size)
		binding := g.getOrCreateDeviceMemory(memory).addBinding(offset, size)
		log.W(ctx, "RecreateBindImageMemory memory binding: %v, %v", g.addressMap.addressOf(binding), *binding)
		b.write(g, binding)

	case *RecreateBindBufferMemory:
		buffer := typedAtom.Buffer
		memory := typedAtom.Memory
		b.modify(g, vulkanStateKey(buffer))
		b.read(g, g.getOrCreateDeviceMemory(memory).handle)
		offset := uint64(GetState(s).Buffers.Get(buffer).MemoryOffset)
		size := uint64(GetState(s).Buffers.Get(buffer).Info.Size)
		binding := g.getOrCreateDeviceMemory(memory).addBinding(offset, size)
		b.write(g, binding)

	case *RecreateImageData:
		image := typedAtom.Image
		b.modify(g, vulkanStateKey(image))
		overlappingBindings := getOverlappedBindingsForImage(image)
		writeMemoryBindingsData(&b, overlappingBindings)

	case *RecreateBufferData:
		buffer := typedAtom.Buffer
		b.modify(g, vulkanStateKey(buffer))
		overlappingBindings := getOverlappedBindingsForBuffer(buffer)
		writeMemoryBindingsData(&b, overlappingBindings)

	case *VkDestroyImage:
		image := typedAtom.Image
		b.modify(g, vulkanStateKey(image))
		b.KeepAlive = true

	case *VkDestroyBuffer:
		buffer := typedAtom.Buffer
		b.modify(g, vulkanStateKey(buffer))
		b.KeepAlive = true

	case *VkFreeMemory:
		memory := typedAtom.Memory
		// Free/deletion atoms are kept alive so the creation atom of the
		// corresponding handle will also be kept alive, even though the handle
		// may not be used anywhere else.
		b.read(g, vulkanStateKey(memory))
		b.KeepAlive = true

	case *VkMapMemory:
		memory := typedAtom.Memory
		b.modify(g, g.getOrCreateDeviceMemory(memory))

	case *VkUnmapMemory:
		memory := typedAtom.Memory
		b.modify(g, g.getOrCreateDeviceMemory(memory))

	case *VkFlushMappedMemoryRanges:
		ranges := typedAtom.PMemoryRanges.Slice(0, uint64(typedAtom.MemoryRangeCount), s)
		// TODO: Link the contiguous ranges into one so that we don't miss
		// potential overwrites
		for i := uint64(0); i < uint64(typedAtom.MemoryRangeCount); i++ {
			mappedRange := ranges.Index(i, s).Read(ctx, a, s, nil)
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
						b.modify(g, binding.data)
					} else {
						b.write(g, binding.data)
					}
				} else {
					b.modify(g, binding.data)
				}
			}
		}

	case *VkInvalidateMappedMemoryRanges:
		ranges := typedAtom.PMemoryRanges.Slice(0, uint64(typedAtom.MemoryRangeCount), s)
		// TODO: Link the contiguous ranges
		for i := uint64(0); i < uint64(typedAtom.MemoryRangeCount); i++ {
			mappedRange := ranges.Index(i, s).Read(ctx, a, s, nil)
			memory := mappedRange.Memory
			offset := uint64(mappedRange.Offset)
			size := uint64(mappedRange.Size)
			bindings := getOverlappingMemoryBindings(memory, offset, size)
			readMemoryBindingsData(&b, bindings)
		}

	case *VkCreateImageView:
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		image := createInfo.Image
		view := typedAtom.PView.Read(ctx, a, s, nil)
		b.read(g, vulkanStateKey(image))
		b.write(g, vulkanStateKey(view))

	case *RecreateImageView:
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		image := createInfo.Image
		view := typedAtom.PImageView.Read(ctx, a, s, nil)
		b.read(g, vulkanStateKey(image))
		b.write(g, vulkanStateKey(view))

	case *VkCreateBufferView:
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		buffer := createInfo.Buffer
		view := typedAtom.PView.Read(ctx, a, s, nil)
		b.read(g, vulkanStateKey(buffer))
		b.write(g, vulkanStateKey(view))

	case *RecreateBufferView:
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		buffer := createInfo.Buffer
		view := typedAtom.PBufferView.Read(ctx, a, s, nil)
		b.read(g, vulkanStateKey(buffer))
		b.write(g, vulkanStateKey(view))

	case *VkUpdateDescriptorSets:
		// handle descriptor writes
		writeCount := typedAtom.DescriptorWriteCount
		if writeCount > 0 {
			writes := typedAtom.PDescriptorWrites.Slice(0, uint64(writeCount), s)
			if err := processDescriptorWrites(writes, &b, g, ctx, a, s); err != nil {
				log.E(ctx, "Atom %v %v: %v", id, a, err)
				return AtomBehaviour{Aborted: true}
			}
		}
		// handle descriptor copies
		copyCount := typedAtom.DescriptorCopyCount
		if copyCount > 0 {
			copies := typedAtom.PDescriptorCopies.Slice(0, uint64(copyCount), s)
			for i := uint32(0); i < copyCount; i++ {
				copy := copies.Index(uint64(i), s).Read(ctx, a, s, nil)
				srcDescriptor := copy.SrcSet
				dstDescriptor := copy.DstSet
				b.read(g, vulkanStateKey(srcDescriptor))
				b.modify(g, vulkanStateKey(dstDescriptor))
			}
		}

	case *RecreateDescriptorSet:
		// handle descriptor writes
		writeCount := typedAtom.DescriptorWriteCount
		if writeCount > 0 {
			writes := typedAtom.PDescriptorWrites.Slice(0, uint64(writeCount), s)
			if err := processDescriptorWrites(writes, &b, g, ctx, a, s); err != nil {
				log.E(ctx, "Atom %v %v: %v", id, a, err)
				return AtomBehaviour{Aborted: true}
			}
		}

	case *VkCreateFramebuffer:
		b.write(g, vulkanStateKey(typedAtom.PFramebuffer.Read(ctx, a, s, nil)))
		b.read(g, vulkanStateKey(typedAtom.PCreateInfo.Read(ctx, a, s, nil).RenderPass))
		// process the attachments
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		attachmentCount := createInfo.AttachmentCount
		attachments := createInfo.PAttachments.Slice(0, uint64(attachmentCount), s)
		for i := uint32(0); i < attachmentCount; i++ {
			attachedViews := attachments.Index(uint64(i), s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(attachedViews))
		}

	case *RecreateFramebuffer:
		b.write(g, vulkanStateKey(typedAtom.PFramebuffer.Read(ctx, a, s, nil)))
		b.read(g, vulkanStateKey(typedAtom.PCreateInfo.Read(ctx, a, s, nil).RenderPass))
		// process the attachments
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		attachmentCount := createInfo.AttachmentCount
		attachments := createInfo.PAttachments.Slice(0, uint64(attachmentCount), s)
		for i := uint32(0); i < attachmentCount; i++ {
			attachedViews := attachments.Index(uint64(i), s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(attachedViews))
		}

	case *VkCreateRenderPass:
		b.write(g, vulkanStateKey(typedAtom.PRenderPass.Read(ctx, a, s, nil)))

	case *RecreateRenderPass:
		b.write(g, vulkanStateKey(typedAtom.PRenderPass.Read(ctx, a, s, nil)))

	case *VkCreateGraphicsPipelines:
		pipelineCount := uint64(typedAtom.CreateInfoCount)
		createInfos := typedAtom.PCreateInfos.Slice(0, pipelineCount, s)
		pipelines := typedAtom.PPipelines.Slice(0, pipelineCount, s)
		for i := uint64(0); i < pipelineCount; i++ {
			// read shaders
			stageCount := uint64(createInfos.Index(i, s).Read(ctx, a, s, nil).StageCount)
			shaderStages := createInfos.Index(i, s).Read(ctx, a, s, nil).PStages.Slice(0, stageCount, s)
			for j := uint64(0); j < stageCount; j++ {
				shaderStage := shaderStages.Index(j, s).Read(ctx, a, s, nil)
				module := shaderStage.Module
				b.read(g, vulkanStateKey(module))
			}
			// read renderpass
			renderPass := createInfos.Index(i, s).Read(ctx, a, s, nil).RenderPass
			b.read(g, vulkanStateKey(renderPass))
			// Create pipeline
			pipeline := pipelines.Index(i, s).Read(ctx, a, s, nil)
			b.write(g, vulkanStateKey(pipeline))
		}

	case *RecreateGraphicsPipeline:
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		stageCount := uint64(createInfo.StageCount)
		shaderStages := createInfo.PStages.Slice(0, stageCount, s)
		for i := uint64(0); i < stageCount; i++ {
			shaderStage := shaderStages.Index(i, s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(shaderStage.Module))
		}
		b.read(g, vulkanStateKey(createInfo.RenderPass))
		b.write(g, vulkanStateKey(typedAtom.PPipeline.Read(ctx, a, s, nil)))

	case *VkCreateComputePipelines:
		pipelineCount := uint64(typedAtom.CreateInfoCount)
		createInfos := typedAtom.PCreateInfos.Slice(0, pipelineCount, s)
		pipelines := typedAtom.PPipelines.Slice(0, pipelineCount, s)
		for i := uint64(0); i < pipelineCount; i++ {
			// read shader
			shaderStage := createInfos.Index(i, s).Read(ctx, a, s, nil).Stage
			module := shaderStage.Module
			b.read(g, vulkanStateKey(module))
			// Create pipeline
			pipeline := pipelines.Index(i, s).Read(ctx, a, s, nil)
			b.write(g, vulkanStateKey(pipeline))
		}

	case *RecreateComputePipeline:
		createInfo := typedAtom.PCreateInfo.Read(ctx, a, s, nil)
		module := createInfo.Stage.Module
		b.read(g, vulkanStateKey(module))
		b.write(g, vulkanStateKey(typedAtom.PPipeline.Read(ctx, a, s, nil)))

	case *VkCreateShaderModule:
		b.write(g, vulkanStateKey(typedAtom.PShaderModule.Read(ctx, a, s, nil)))

	case *RecreateShaderModule:
		b.write(g, vulkanStateKey(typedAtom.PShaderModule.Read(ctx, a, s, nil)))

	case *VkCmdCopyImage:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdCopyImage:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdCopyImageToBuffer:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdCopyImageToBuffer:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdCopyBufferToImage:
		b.read(g, vulkanStateKey(typedAtom.SrcBuffer))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForBuffer(typedAtom.SrcBuffer)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdCopyBufferToImage:
		b.read(g, vulkanStateKey(typedAtom.SrcBuffer))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForBuffer(typedAtom.SrcBuffer)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdCopyBuffer:
		b.read(g, vulkanStateKey(typedAtom.SrcBuffer))
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		srcBindings := getOverlappedBindingsForBuffer(typedAtom.SrcBuffer)
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdCopyBuffer:
		b.read(g, vulkanStateKey(typedAtom.SrcBuffer))
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		srcBindings := getOverlappedBindingsForBuffer(typedAtom.SrcBuffer)
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdBlitImage:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdBlitImage:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdResolveImage:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdResolveImage:
		b.read(g, vulkanStateKey(typedAtom.SrcImage))
		b.read(g, vulkanStateKey(typedAtom.DstImage))
		srcBindings := getOverlappedBindingsForImage(typedAtom.SrcImage)
		dstBindings := getOverlappedBindingsForImage(typedAtom.DstImage)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, srcBindings)
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdFillBuffer:
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdFillBuffer:
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdUpdateBuffer:
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateUpdateBuffer:
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdCopyQueryPoolResults:
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *RecreateCmdCopyQueryPoolResults:
		b.read(g, vulkanStateKey(typedAtom.DstBuffer))
		dstBindings := getOverlappedBindingsForBuffer(typedAtom.DstBuffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// Be conservative here. Without tracking all the memory ranges and
			// calculating the memory according to the copy region, we cannot assume
			// this command overwrites the data. So it is labelled as 'modify' to
			// kept the previous writes
			modifyMemoryBindingsData(b, dstBindings)
		})

	case *VkCmdBindVertexBuffers:
		count := typedAtom.BindingCount
		buffers := typedAtom.PBuffers.Slice(0, uint64(count), s)
		for i := uint64(0); i < uint64(count); i++ {
			buffer := buffers.Index(i, s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(buffer))
			b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
			b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
			recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
				// as the LastBoundQueue of the buffer object has will change
				b.modify(g, vulkanStateKey(buffer))
			})
		}

	case *RecreateCmdBindVertexBuffers:
		count := typedAtom.BindingCount
		buffers := typedAtom.PBuffers.Slice(0, uint64(count), s)
		for i := uint64(0); i < uint64(count); i++ {
			buffer := buffers.Index(i, s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(buffer))
			b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
			b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
			recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
				// as the LastBoundQueue of the buffer object will change
				b.modify(g, vulkanStateKey(buffer))
			})
		}

	case *VkCmdBindIndexBuffer:
		buffer := typedAtom.Buffer
		b.read(g, vulkanStateKey(buffer))
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// as the LastBoundQueue of the buffer object will change
			b.modify(g, vulkanStateKey(buffer))
		})

	case *RecreateCmdBindIndexBuffer:
		buffer := typedAtom.Buffer
		b.read(g, vulkanStateKey(buffer))
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// as the LastBoundQueue of the buffer object will change
			b.modify(g, vulkanStateKey(buffer))
		})

	case *VkCmdDraw:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
		})
		// TODO: add 'write' or 'modify' behaviour to the attachments

	case *RecreateCmdDraw:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
		})
		// TODO: add 'write' or 'modify' behaviour to the attachments

	case *VkCmdDrawIndexed:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// vertex buffers
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
			// index buffer
			indexBuffer := GetState(s).CurrentIndexBuffer.BoundBuffer.Buffer.VulkanHandle
			memoryBindings := getOverlappedBindingsForBuffer(indexBuffer)
			readMemoryBindingsData(b, memoryBindings)
			// TODO: add 'write' or 'modify' behaviour to the attachments
		})

	case *RecreateCmdDrawIndexed:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// vertex buffers
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
			// index buffer
			indexBuffer := GetState(s).CurrentIndexBuffer.BoundBuffer.Buffer.VulkanHandle
			memoryBindings := getOverlappedBindingsForBuffer(indexBuffer)
			readMemoryBindingsData(b, memoryBindings)
			// TODO: add 'write' or 'modify' behaviour to the attachments
		})

	case *VkCmdDrawIndirect:
		indirectBuf := typedAtom.Buffer
		b.read(g, vulkanStateKey(indirectBuf))
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// vertex buffers
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
			// indirect buffer memory
			memoryBindings := getOverlappedBindingsForBuffer(indirectBuf)
			readMemoryBindingsData(b, memoryBindings)
			// TODO: add 'write' or 'modify' behaviour to the attachments
		})

	case *RecreateCmdDrawIndirect:
		indirectBuf := typedAtom.Buffer
		b.read(g, vulkanStateKey(indirectBuf))
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// vertex buffers
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
			// indirect buffer memory
			memoryBindings := getOverlappedBindingsForBuffer(indirectBuf)
			readMemoryBindingsData(b, memoryBindings)
			// TODO: add 'write' or 'modify' behaviour to the attachments
		})

	case *VkCmdDrawIndexedIndirect:
		indirectBuf := typedAtom.Buffer
		b.read(g, vulkanStateKey(indirectBuf))
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// vertex buffers
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
			// index buffer
			indexBuffer := GetState(s).CurrentIndexBuffer.BoundBuffer.Buffer.VulkanHandle
			indexBufMemoryBindings := getOverlappedBindingsForBuffer(indexBuffer)
			readMemoryBindingsData(b, indexBufMemoryBindings)
			// indirect buffer memory
			indirectBufMemoryBindings := getOverlappedBindingsForBuffer(indirectBuf)
			readMemoryBindingsData(b, indirectBufMemoryBindings)
			// TODO: add 'write' or 'modify' behaviour to the attachments
		})

	case *RecreateCmdDrawIndexedIndirect:
		indirectBuf := typedAtom.Buffer
		b.read(g, vulkanStateKey(indirectBuf))
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			// vertex buffers
			for _, bd := range GetState(s).CurrentGraphicsPipeline.VertexInputState.BindingDescriptions {
				if GetState(s).CurrentBoundVertexBuffers.Contains(bd.Binding) {
					boundVertexBuffer := GetState(s).CurrentBoundVertexBuffers.Get(bd.Binding)
					backingBuffer := boundVertexBuffer.Buffer.VulkanHandle
					memoryBindings := getOverlappedBindingsForBuffer(backingBuffer)
					readMemoryBindingsData(b, memoryBindings)
				}
			}
			// index buffer
			indexBuffer := GetState(s).CurrentIndexBuffer.BoundBuffer.Buffer.VulkanHandle
			indexBufMemoryBindings := getOverlappedBindingsForBuffer(indexBuffer)
			readMemoryBindingsData(b, indexBufMemoryBindings)
			// indirect buffer memory
			indirectBufMemoryBindings := getOverlappedBindingsForBuffer(indirectBuf)
			readMemoryBindingsData(b, indirectBufMemoryBindings)
			// TODO: add 'write' or 'modify' behaviour to the attachments
		})

	case *VkCmdDispatchIndirect:
		buffer := typedAtom.Buffer
		b.read(g, vulkanStateKey(buffer))
		memoryBindings := getOverlappedBindingsForBuffer(buffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, memoryBindings)
		})
		// TODO: add 'write' or 'modify' behaviour to the descriptors

	case *RecreateCmdDispatchIndirect:
		buffer := typedAtom.Buffer
		b.read(g, vulkanStateKey(buffer))
		memoryBindings := getOverlappedBindingsForBuffer(buffer)
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			readMemoryBindingsData(b, memoryBindings)
		})
		// TODO: add 'write' or 'modify' behaviour to the descriptors

	case *VkCmdBeginRenderPass:
		beginInfo := typedAtom.PRenderPassBegin.Read(ctx, a, s, nil)
		framebuffer := beginInfo.Framebuffer
		b.read(g, vulkanStateKey(framebuffer))
		renderpass := beginInfo.RenderPass
		b.read(g, vulkanStateKey(renderpass))

		atts := GetState(s).Framebuffers.Get(framebuffer).ImageAttachments
		attDescs := GetState(s).RenderPasses.Get(renderpass).AttachmentDescriptions
		for i := uint32(0); i < uint32(len(atts)); i++ {
			img := atts.Get(i).Image.VulkanHandle
			loadOp := attDescs.Get(i).LoadOp
			if loadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR {
				// This can be wrong as this is clearing all the memory bindings
				// that overlap with the attachment image, so extra memories might be
				// cleared. However in practical, image should be bound to only one
				// memory binding as a whole. So here should be a problem.
				// TODO: Add intersection operation to get the memory ranges to clear
				bindingsToClear := getOverlappedBindingsForImage(img)
				b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
				b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
				recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
					writeMemoryBindingsData(b, bindingsToClear)
				})
			} else {
				bindingsToRead := getOverlappedBindingsForImage(img)
				b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
				b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
				recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
					readMemoryBindingsData(b, bindingsToRead)
				})
			}
		}

	case *RecreateCmdBeginRenderPass:
		beginInfo := typedAtom.PRenderPassBegin.Read(ctx, a, s, nil)
		framebuffer := beginInfo.Framebuffer
		b.read(g, vulkanStateKey(framebuffer))
		renderpass := beginInfo.RenderPass
		b.read(g, vulkanStateKey(renderpass))

		atts := GetState(s).Framebuffers.Get(framebuffer).ImageAttachments
		attDescs := GetState(s).RenderPasses.Get(renderpass).AttachmentDescriptions
		for i := uint32(0); i < uint32(len(atts)); i++ {
			img := atts.Get(i).Image.VulkanHandle
			loadOp := attDescs.Get(i).LoadOp
			if loadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR {
				// This can be wrong as this is clearing all the memory bindings
				// that overlap with the attachment image, so extra memorise might be
				// cleared. However in real case, image should be bound to only one
				// memory binding as a whole. So here should be cause problem.
				// TODO: Add intersection operation to get the memory ranges to clear
				bindingsToClear := getOverlappedBindingsForImage(img)
				b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
				b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
				recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
					writeMemoryBindingsData(b, bindingsToClear)
				})
			} else {
				bindingsToRead := getOverlappedBindingsForImage(img)
				b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
				b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
				recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
					readMemoryBindingsData(b, bindingsToRead)
				})
			}
		}

	case *VkCmdEndRenderPass:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdEndRenderPass:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdNextSubpass:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdNextSubpass:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdPushConstants:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdPushConstants:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdSetLineWidth:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdSetLineWidth:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdSetScissor:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdSetScissor:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdSetViewport:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdSetViewport:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdBindDescriptorSets:
		descriptorSetCount := typedAtom.DescriptorSetCount
		descriptorSets := typedAtom.PDescriptorSets.Slice(0, uint64(descriptorSetCount), s)
		for i := uint32(0); i < descriptorSetCount; i++ {
			descriptorSet := descriptorSets.Index(uint64(i), s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(descriptorSet))
			for _, descBinding := range GetState(s).DescriptorSets.Get(descriptorSet).Bindings {
				for _, bufferInfo := range descBinding.BufferBinding {
					buf := bufferInfo.Buffer
					b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
					b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
					recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
						// Descriptors might be modified
						b.modify(g, vulkanStateKey(buf))
					})
				}
				for _, imageInfo := range descBinding.ImageBinding {
					view := imageInfo.ImageView
					b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
					b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
					recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
						b.read(g, vulkanStateKey(view))
					})
				}
				for _, bufferView := range descBinding.BufferViewBindings {
					b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
					b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
					recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
						b.read(g, vulkanStateKey(bufferView))
					})
				}
			}
		}

	case *RecreateCmdBindDescriptorSets:
		descriptorSetCount := typedAtom.DescriptorSetCount
		descriptorSets := typedAtom.PDescriptorSets.Slice(0, uint64(descriptorSetCount), s)
		for i := uint32(0); i < descriptorSetCount; i++ {
			descriptorSet := descriptorSets.Index(uint64(i), s).Read(ctx, a, s, nil)
			b.read(g, vulkanStateKey(descriptorSet))
			for _, descBinding := range GetState(s).DescriptorSets.Get(descriptorSet).Bindings {
				for _, bufferInfo := range descBinding.BufferBinding {
					buf := bufferInfo.Buffer
					b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
					b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
					recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
						// Descriptors might be modified
						b.modify(g, vulkanStateKey(buf))
					})
				}
				for _, imageInfo := range descBinding.ImageBinding {
					view := imageInfo.ImageView
					b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
					b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
					recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
						b.read(g, vulkanStateKey(view))
					})
				}
				for _, bufferView := range descBinding.BufferViewBindings {
					b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
					b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
					recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
						b.read(g, vulkanStateKey(bufferView))
					})
				}
			}
		}

	case *VkBeginCommandBuffer:
		cmdbuf := g.getOrCreateCommandBuffer(typedAtom.CommandBuffer)
		b.read(g, cmdbuf.handle)
		b.write(g, cmdbuf.records)

	case *VkEndCommandBuffer:
		cmdbuf := g.getOrCreateCommandBuffer(typedAtom.CommandBuffer)
		b.modify(g, cmdbuf)

	case *RecreateAndBeginCommandBuffer:
		cmdbuf := g.getOrCreateCommandBuffer(typedAtom.PCommandBuffer.Read(ctx, a, s, nil))
		b.write(g, cmdbuf)

	case *RecreateEndCommandBuffer:
		cmdbuf := g.getOrCreateCommandBuffer(typedAtom.CommandBuffer)
		b.modify(g, cmdbuf)

	case *VkCmdPipelineBarrier:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the image and buffer memory barriers?

	case *RecreateCmdPipelineBarrier:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the image and buffer memory barriers?

	case *VkCmdBindPipeline:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			b.read(g, vulkanStateKey(typedAtom.Pipeline))
		})
		b.read(g, vulkanStateKey(typedAtom.Pipeline))

	case *RecreateCmdBindPipeline:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		recordCommand(typedAtom.CommandBuffer, func(b *AtomBehaviour) {
			b.read(g, vulkanStateKey(typedAtom.Pipeline))
		})
		b.read(g, vulkanStateKey(typedAtom.Pipeline))

	case *VkCmdBeginQuery:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdBeginQuery:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdEndQuery:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdEndQuery:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdResetQueryPool:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdResetQueryPool:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdClearAttachments:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdClearAttachments:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the case that the attachment is fully cleared.

	case *VkCmdClearColorImage:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the color image

	case *RecreateCmdClearColorImage:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the color image

	case *VkCmdClearDepthStencilImage:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the depth/stencil image

	case *RecreateCmdClearDepthStencilImage:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)
		//TODO: handle the depth/stencil image

	case *VkCmdSetDepthBias:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdSetDepthBias:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkCmdSetBlendConstants:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *RecreateCmdSetBlendConstants:
		b.read(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).handle)
		b.modify(g, g.getOrCreateCommandBuffer(typedAtom.CommandBuffer).records)

	case *VkQueueSubmit:
		// Queue submit atom should always be alive
		b.KeepAlive = true

		// handle queue
		b.modify(g, vulkanStateKey(typedAtom.Queue))

		// handle command buffers
		submitCount := typedAtom.SubmitCount
		submits := typedAtom.PSubmits.Slice(0, uint64(submitCount), s)
		for i := uint32(0); i < submitCount; i++ {
			submit := submits.Index(uint64(i), s).Read(ctx, a, s, nil)
			commandBufferCount := submit.CommandBufferCount
			commandBuffers := submit.PCommandBuffers.Slice(0, uint64(commandBufferCount), s)
			for j := uint32(0); j < submit.CommandBufferCount; j++ {
				vkCmdBuf := commandBuffers.Index(uint64(j), s).Read(ctx, a, s, nil)
				cb := g.getOrCreateCommandBuffer(vkCmdBuf)
				// All the commands that are submitted will not be dropped.
				b.read(g, cb.handle)
				b.read(g, cb.records)

				// Carry out the behaviors in the recorded commands.
				for _, c := range cb.records.Commands {
					c(&b)
				}
			}
		}

	case *VkQueuePresentKHR:
		b.read(g, vulkanStateKey(typedAtom.Queue))
		g.roots[g.addressMap.addressOf(vulkanStateKey(typedAtom.Queue))] = true
		b.KeepAlive = true

	default:
		// TODO: handle vkGetDeviceMemoryCommitment, VkSparseMemoryBind and other
		// commands
		b.KeepAlive = true
	}
	return b
}

// Traverse through the given VkWriteDescriptorSet slice, add behaviors to
// |b| according to the descriptor type.
func processDescriptorWrites(writes VkWriteDescriptorSetˢ, b *AtomBehaviour, g *DependencyGraph, ctx context.Context, a atom.Atom, s *gfxapi.State) error {
	writeCount := writes.Info().Count
	for i := uint64(0); i < writeCount; i++ {
		write := writes.Index(uint64(i), s).Read(ctx, a, s, nil)
		if write.DescriptorCount > 0 {
			// handle the target descriptor set
			b.modify(g, vulkanStateKey(write.DstSet))
			switch write.DescriptorType {
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
				imageInfos := write.PImageInfo.Slice(0, uint64(write.DescriptorCount), s)
				for j := uint64(0); j < imageInfos.Info().Count; j++ {
					imageInfo := imageInfos.Index(uint64(j), s).Read(ctx, a, s, nil)
					sampler := imageInfo.Sampler
					imageView := imageInfo.ImageView
					b.read(g, vulkanStateKey(sampler))
					b.read(g, vulkanStateKey(imageView))
				}
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
				bufferInfos := write.PBufferInfo.Slice(0, uint64(write.DescriptorCount), s)
				for j := uint64(0); j < bufferInfos.Info().Count; j++ {
					bufferInfo := bufferInfos.Index(uint64(j), s).Read(ctx, a, s, nil)
					buffer := bufferInfo.Buffer
					b.read(g, vulkanStateKey(buffer))
				}
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
				bufferViews := write.PTexelBufferView.Slice(0, uint64(write.DescriptorCount), s)
				for j := uint64(0); j < bufferViews.Info().Count; j++ {
					bufferView := bufferViews.Index(uint64(j), s).Read(ctx, a, s, nil)
					b.read(g, vulkanStateKey(bufferView))
				}
			default:
				return fmt.Errorf("Unhandled DescriptorType: %v", write.DescriptorType)
			}
		}
	}
	return nil
}
