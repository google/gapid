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
	"sort"
	"strings"

	"github.com/google/gapid/core/image"
	"github.com/google/gapid/core/log"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/service/path"
)

// backedByCoherentMemory returns true if the device memory object is backed by
// memory where the VK_MEMORY_PROPERTY_HOST_COHERENT_BIT is set.
// Note that there exists a generated subIsMemoryCoherent(), but it needs all
// the arguments of generated functions, which are not easily retrievable here.
func backedByCoherentMemory(state *State, mem DeviceMemoryObjectʳ) bool {
	phyDevHandle := state.Devices().Get(mem.Device()).PhysicalDevice()
	phyDevObj := state.PhysicalDevices().Get(phyDevHandle)
	propFlags := phyDevObj.MemoryProperties().MemoryTypes().Get(int(mem.MemoryTypeIndex())).PropertyFlags()
	return uint32(propFlags)&uint32(VkMemoryPropertyFlagBits_VK_MEMORY_PROPERTY_HOST_COHERENT_BIT) != 0
}

func newFramegraphBuffer(state *State, buf *BufferObjectʳ) *api.FramegraphBuffer {
	usage := uint32(buf.Info().Usage())
	coherentMemory := backedByCoherentMemory(state, buf.Memory())
	memoryMapped := !(buf.Memory().MappedLocation().IsNullptr())
	// No need to scan sparse memory if both coherent/mapped are already true
	if !coherentMemory || !memoryMapped {
		for _, sparseMemBinding := range buf.SparseMemoryBindings().All() {
			mem := state.DeviceMemories().Get(sparseMemBinding.Memory())
			coherentMemory = coherentMemory || backedByCoherentMemory(state, mem)
			memoryMapped = memoryMapped || !(mem.MappedLocation().IsNullptr())
		}
	}

	return &api.FramegraphBuffer{
		Handle:         uint64(buf.VulkanHandle()),
		Size:           uint64(buf.Info().Size()),
		Usage:          usage,
		TransferSrc:    usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT) != 0,
		TransferDst:    usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT) != 0,
		UniformTexel:   usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_TEXEL_BUFFER_BIT) != 0,
		StorageTexel:   usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_STORAGE_TEXEL_BUFFER_BIT) != 0,
		Uniform:        usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT) != 0,
		Storage:        usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_STORAGE_BUFFER_BIT) != 0,
		Index:          usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_INDEX_BUFFER_BIT) != 0,
		Vertex:         usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_VERTEX_BUFFER_BIT) != 0,
		Indirect:       usage&uint32(VkBufferUsageFlagBits_VK_BUFFER_USAGE_INDIRECT_BUFFER_BIT) != 0,
		CoherentMemory: coherentMemory,
		MemoryMapped:   memoryMapped,
	}
}

func newFramegraphImage(state *State, img *ImageObjectʳ) *api.FramegraphImage {
	format, err := getImageFormatFromVulkanFormat(img.Info().Fmt())
	if err != nil {
		panic("Unrecognized Vulkan image format")
	}

	coherentMemory := false
	memoryMapped := false
	for _, planeMemInfo := range img.PlaneMemoryInfo().All() {
		mem := planeMemInfo.BoundMemory()
		coherentMemory = coherentMemory || backedByCoherentMemory(state, mem)
		memoryMapped = memoryMapped || !(mem.MappedLocation().IsNullptr())
	}
	// No need to scan sparse memory if both coherent/mapped are already true
	if !coherentMemory || !memoryMapped {
		for _, sparseMemBinding := range img.OpaqueSparseMemoryBindings().All() {
			mem := state.DeviceMemories().Get(sparseMemBinding.Memory())
			coherentMemory = coherentMemory || backedByCoherentMemory(state, mem)
			memoryMapped = memoryMapped || !(mem.MappedLocation().IsNullptr())
		}
	}
	if !coherentMemory || !memoryMapped {
		for _, sparseImgMem := range img.SparseImageMemoryBindings().All() {
			for _, layers := range sparseImgMem.Layers().All() {
				for _, level := range layers.Levels().All() {
					for _, blocks := range level.Blocks().All() {
						mem := state.DeviceMemories().Get(blocks.Memory())
						coherentMemory = coherentMemory || backedByCoherentMemory(state, mem)
						memoryMapped = memoryMapped || !(mem.MappedLocation().IsNullptr())
					}
				}
			}
		}
	}

	usage := uint32(img.Info().Usage())
	return &api.FramegraphImage{
		Handle:    uint64(img.VulkanHandle()),
		Usage:     usage,
		ImageType: strings.TrimPrefix(fmt.Sprintf("%v", img.Info().ImageType()), "VK_IMAGE_TYPE_"),
		Info: &image.Info{
			Format: format,
			Width:  img.Info().Extent().Width(),
			Height: img.Info().Extent().Height(),
			Depth:  img.Info().Extent().Depth(),
		},
		TransferSrc:            usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_SRC_BIT) != 0,
		TransferDst:            usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSFER_DST_BIT) != 0,
		Sampled:                usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_SAMPLED_BIT) != 0,
		Storage:                usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_STORAGE_BIT) != 0,
		ColorAttachment:        usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_COLOR_ATTACHMENT_BIT) != 0,
		DepthStencilAttachment: usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_DEPTH_STENCIL_ATTACHMENT_BIT) != 0,
		TransientAttachment:    usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT) != 0,
		InputAttachment:        usage&uint32(VkImageUsageFlagBits_VK_IMAGE_USAGE_INPUT_ATTACHMENT_BIT) != 0,
		Swapchain:              img.IsSwapchainImage(),
		CoherentMemory:         coherentMemory,
		MemoryMapped:           memoryMapped,
	}
}

func loadOp2LoadStoreOp(loadOp VkAttachmentLoadOp) api.LoadStoreOp {
	switch loadOp {
	case VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD:
		return api.LoadStoreOp_LOAD
	case VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR:
		return api.LoadStoreOp_CLEAR
	case VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE:
		return api.LoadStoreOp_DISCARD
	}
	panic("Unknown loadOp")
}

func storeOp2LoadStoreOp(storeOp VkAttachmentStoreOp) api.LoadStoreOp {
	switch storeOp {
	case VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE:
		return api.LoadStoreOp_STORE
	case VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE:
		return api.LoadStoreOp_DISCARD
	}
	panic("Unknown storeOp")
}

func newFramegraphAttachment(desc VkAttachmentDescription, state *State, imgView ImageViewObjectʳ, isDepthStencil bool) *api.FramegraphAttachment {
	var loadOp VkAttachmentLoadOp
	var storeOp VkAttachmentStoreOp
	if isDepthStencil {
		loadOp = desc.StencilLoadOp()
		storeOp = desc.StencilStoreOp()
	} else {
		loadOp = desc.LoadOp()
		storeOp = desc.StoreOp()
	}
	imgObj := imgView.Image()
	return &api.FramegraphAttachment{
		LoadOp:          loadOp2LoadStoreOp(loadOp),
		StoreOp:         storeOp2LoadStoreOp(storeOp),
		ImageViewHandle: uint64(imgView.VulkanHandle()),
		Image:           newFramegraphImage(state, &imgObj),
	}
}

func newFramegraphSubpass(subpassDesc SubpassDescription, state *State, framebuffer FramebufferObjectʳ, renderpass RenderPassObjectʳ) *api.FramegraphSubpass {
	inputAtts := subpassDesc.InputAttachments()
	colorAtts := subpassDesc.ColorAttachments()
	resolveAtts := subpassDesc.ResolveAttachments()
	sp := &api.FramegraphSubpass{
		Input:   make([]*api.FramegraphAttachment, inputAtts.Len()),
		Color:   make([]*api.FramegraphAttachment, colorAtts.Len()),
		Resolve: make([]*api.FramegraphAttachment, resolveAtts.Len()),
	}

	// Input attachments
	for i := 0; i < inputAtts.Len(); i++ {
		idx := inputAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		sp.Input[i] = newFramegraphAttachment(desc, state, imgView, false)
	}

	// Color attachments
	for i := 0; i < colorAtts.Len(); i++ {
		idx := colorAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		sp.Color[i] = newFramegraphAttachment(desc, state, imgView, false)
	}

	// Resolve attachments
	for i := 0; i < resolveAtts.Len(); i++ {
		idx := resolveAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		sp.Resolve[i] = newFramegraphAttachment(desc, state, imgView, false)
	}

	// DepthStencil attachment
	depthStencilAtt := subpassDesc.DepthStencilAttachment()
	if !depthStencilAtt.IsNil() {
		idx := depthStencilAtt.Attachment()
		if idx != VK_ATTACHMENT_UNUSED {
			desc := renderpass.AttachmentDescriptions().Get(idx)
			imgView := framebuffer.ImageAttachments().Get(idx)
			sp.DepthStencil = newFramegraphAttachment(desc, state, imgView, true)
		}
	}

	return sp
}

type imageAccessInfo struct {
	read  bool
	write bool
	image *api.FramegraphImage
}

type bufferAccessInfo struct {
	read   bool
	write  bool
	buffer *api.FramegraphBuffer
}

// workloadInfo stores the workload details that are relevant for the framegraph.
type workloadInfo struct {
	// Exactly one of the following pointers should be non-nil.
	// See api/service.proto for the possible values of workload.
	// Note: it might be tempting to use the api.isFramegraphNode_Workload
	// interface here, but down the line functions like GetRenderpass() returns
	// either a value or nil, so you will have to check for nil anyway.
	renderpass *api.FramegraphRenderpass
	compute    *api.FramegraphCompute

	// This workload ID, used in the framegraph edges
	id uint64

	// nodes stores the dependency graph nodes of commands of this workload
	nodes []dependencygraph2.NodeID

	// deps stores the set of IDs of workloads this workload depends on
	deps map[uint64]struct{}

	// imageAccesses is a temporary set that is eventually sorted and stored in
	// workload.ImageAccess list.
	imageAccesses map[uint64]*api.FramegraphImageAccess
	// idem imageAccesses, but for buffers
	bufferAccesses map[uint64]*api.FramegraphBufferAccess
}

// framegraphInfoHelpers contains variables that stores information while
// processing subCommands.
type framegraphInfoHelpers struct {
	workloadInfos  []*workloadInfo
	wlInfo         *workloadInfo
	currWorkloadId uint64
	// imageLookup is a lookup table to quickly find images that match a memory
	// access. Scanning the whole state to look for an image matching a memory
	// access is slow, this lookup table saves seconds of computation. It is
	// updated when a subcommand is processed under a new top-level parent
	// command: images can only be created/destroyed between top-level commands,
	// not during the execution of subcommands.
	imageLookup map[memory.PoolID]map[memory.Range][]*api.FramegraphImage
	// idem for buffers
	bufferLookup map[memory.PoolID]map[memory.Range][]*api.FramegraphBuffer
	parentCmdIdx uint64 // used to know when to update the lookup tables
	// lookupInitialized is true if there has been at least one update of the
	// lookup tables.
	lookupInitialized bool
}

// newWorkloadInfo creates a new workload
func (helpers *framegraphInfoHelpers) newWorkloadInfo(renderpass *api.FramegraphRenderpass, compute *api.FramegraphCompute) {
	if helpers.wlInfo != nil {
		panic("Creating a new workloadInfo while there is already one active")
	}
	helpers.wlInfo = &workloadInfo{
		id:             helpers.currWorkloadId,
		renderpass:     renderpass,
		compute:        compute,
		nodes:          []dependencygraph2.NodeID{},
		deps:           make(map[uint64]struct{}),
		imageAccesses:  make(map[uint64]*api.FramegraphImageAccess),
		bufferAccesses: make(map[uint64]*api.FramegraphBufferAccess),
	}
	helpers.currWorkloadId++
}

// endWorkload terminates a workload
func (helpers *framegraphInfoHelpers) endWorkload() {
	if helpers.wlInfo == nil {
		panic("Ending a workload while none is active")
	}
	helpers.workloadInfos = append(helpers.workloadInfos, helpers.wlInfo)
	helpers.wlInfo = nil
}

// updateImageLookup updates the lookup table to quickly find an image matching a memory observation.
func (helpers *framegraphInfoHelpers) updateImageLookup(state *State) {
	helpers.imageLookup = make(map[memory.PoolID]map[memory.Range][]*api.FramegraphImage)
	for _, image := range state.Images().All() {
		for _, aspect := range image.Aspects().All() {
			for _, layer := range aspect.Layers().All() {
				for _, level := range layer.Levels().All() {
					pool := level.Data().Pool()
					memRange := level.Data().Range()
					if _, ok := helpers.imageLookup[pool]; !ok {
						helpers.imageLookup[pool] = make(map[memory.Range][]*api.FramegraphImage)
					}
					helpers.imageLookup[pool][memRange] = append(helpers.imageLookup[pool][memRange], newFramegraphImage(state, &image))
				}
			}
		}
	}
}

// lookupImages returns all the images that contain (pool, memRange).
func (helpers *framegraphInfoHelpers) lookupImages(pool memory.PoolID, memRange memory.Range) []*api.FramegraphImage {
	images := []*api.FramegraphImage{}
	for imgRange := range helpers.imageLookup[pool] {
		if imgRange.Includes(memRange) {
			images = append(images, helpers.imageLookup[pool][imgRange]...)
		}
	}
	return images
}

// updateBufferLookup updates the lookup table to quickly find a buffer matching a memory observation.
func (helpers *framegraphInfoHelpers) updateBufferLookup(state *State) {
	helpers.bufferLookup = make(map[memory.PoolID]map[memory.Range][]*api.FramegraphBuffer)
	for _, buffer := range state.Buffers().All() {
		pool := buffer.Memory().Data().Pool()
		memRange := memory.Range{
			Base: uint64(buffer.MemoryOffset()),
			Size: uint64(buffer.Info().Size()),
		}
		if _, ok := helpers.bufferLookup[pool]; !ok {
			helpers.bufferLookup[pool] = make(map[memory.Range][]*api.FramegraphBuffer)
		}
		helpers.bufferLookup[pool][memRange] = append(helpers.bufferLookup[pool][memRange], newFramegraphBuffer(state, &buffer))
	}
}

// lookupBuffers returns all the buffers that contain (pool, memRange).
func (helpers *framegraphInfoHelpers) lookupBuffers(pool memory.PoolID, memRange memory.Range) []*api.FramegraphBuffer {
	buffers := []*api.FramegraphBuffer{}
	for bufRange := range helpers.bufferLookup[pool] {
		if bufRange.Includes(memRange) {
			buffers = append(buffers, helpers.bufferLookup[pool][bufRange]...)
		}
	}
	return buffers
}

// processSubCommand records framegraph information upon each subcommand.
func (helpers *framegraphInfoHelpers) processSubCommand(ctx context.Context, dependencyGraph dependencygraph2.DependencyGraph, state *api.GlobalState, subCmdIdx api.SubCmdIdx, cmd api.Cmd, i interface{}) {
	vkState := GetState(state)
	cmdRef, ok := i.(CommandReferenceʳ)
	if !ok {
		panic("In Vulkan, MutateWithSubCommands' postSubCmdCb 'interface{}' is not a CommandReferenceʳ")
	}
	cmdArgs := GetCommandArgs(ctx, cmdRef, vkState)

	// Detect beginning of workload
	switch args := cmdArgs.(type) {

	// Beginning of renderpass
	case VkCmdBeginRenderPassArgsʳ:
		if helpers.wlInfo != nil {
			panic("Renderpass starts within another workload")
		}

		framebuffer := vkState.Framebuffers().Get(args.Framebuffer())
		renderpassObj := vkState.RenderPasses().Get(args.RenderPass())
		subpassesDesc := renderpassObj.SubpassDescriptions()
		subpasses := make([]*api.FramegraphSubpass, subpassesDesc.Len())
		for i := 0; i < len(subpasses); i++ {
			subpassDesc := subpassesDesc.Get(uint32(i))
			subpasses[i] = newFramegraphSubpass(subpassDesc, vkState, framebuffer, renderpassObj)
		}

		renderpass := &api.FramegraphRenderpass{
			Handle:            uint64(renderpassObj.VulkanHandle()),
			BeginSubCmdIdx:    []uint64(subCmdIdx),
			FramebufferWidth:  framebuffer.Width(),
			FramebufferHeight: framebuffer.Height(),
			FramebufferLayers: framebuffer.Layers(),
			Subpass:           subpasses,
		}
		helpers.newWorkloadInfo(renderpass, nil)

	// Begin of compute: vkCmdDispatch
	case VkCmdDispatchArgsʳ:
		compute := &api.FramegraphCompute{
			SubCmdIdx:   []uint64(subCmdIdx),
			BaseGroupX:  0,
			BaseGroupY:  0,
			BaseGroupZ:  0,
			GroupCountX: args.GroupCountX(),
			GroupCountY: args.GroupCountY(),
			GroupCountZ: args.GroupCountZ(),
			Indirect:    false,
		}
		helpers.newWorkloadInfo(nil, compute)

	// Begin of compute: vkCmdDispatchBase
	case VkCmdDispatchBaseArgsʳ:
		compute := &api.FramegraphCompute{
			SubCmdIdx:   []uint64(subCmdIdx),
			BaseGroupX:  args.BaseGroupX(),
			BaseGroupY:  args.BaseGroupY(),
			BaseGroupZ:  args.BaseGroupZ(),
			GroupCountX: args.GroupCountX(),
			GroupCountY: args.GroupCountY(),
			GroupCountZ: args.GroupCountZ(),
			Indirect:    false,
		}
		helpers.newWorkloadInfo(nil, compute)

		// Begin of compute: vkCmdDispatchBaseKHR (duplicate from vkCmdDispatchBase
		// as we cannot 'fallthrough' in a type switch)
	case VkCmdDispatchBaseKHRArgsʳ:
		compute := &api.FramegraphCompute{
			SubCmdIdx:   []uint64(subCmdIdx),
			BaseGroupX:  args.BaseGroupX(),
			BaseGroupY:  args.BaseGroupY(),
			BaseGroupZ:  args.BaseGroupZ(),
			GroupCountX: args.GroupCountX(),
			GroupCountY: args.GroupCountY(),
			GroupCountZ: args.GroupCountZ(),
			Indirect:    false,
		}
		helpers.newWorkloadInfo(nil, compute)

	// Begin of compute: vkCmdDispatchIndirect
	case VkCmdDispatchIndirectArgsʳ:
		compute := &api.FramegraphCompute{
			SubCmdIdx: []uint64(subCmdIdx),
			Indirect:  true,
		}
		helpers.newWorkloadInfo(nil, compute)

	}

	// Process commands that are inside a workload
	if helpers.wlInfo != nil {

		// Update image/buffer lookup tables if we change of top-level parent
		// command index: these cannot be created/destroyed during subcommands.
		parentCmdIdx := subCmdIdx[0]
		if !helpers.lookupInitialized || parentCmdIdx > helpers.parentCmdIdx {
			helpers.parentCmdIdx = parentCmdIdx
			helpers.updateImageLookup(vkState)
			helpers.updateBufferLookup(vkState)
			helpers.lookupInitialized = true
		}

		nodeID := dependencyGraph.GetCmdNodeID(api.CmdID(subCmdIdx[0]), subCmdIdx[1:])
		helpers.wlInfo.nodes = append(helpers.wlInfo.nodes, nodeID)

		for _, memAccess := range dependencyGraph.GetNodeAccesses(nodeID).MemoryAccesses {
			memRange := memory.Range{
				Base: memAccess.Span.Start,
				Size: memAccess.Span.End - memAccess.Span.Start,
			}

			for _, image := range helpers.lookupImages(memAccess.Pool, memRange) {
				imgAcc, ok := helpers.wlInfo.imageAccesses[image.Handle]
				if !ok {
					imgAcc = &api.FramegraphImageAccess{
						Image: image,
					}
					helpers.wlInfo.imageAccesses[image.Handle] = imgAcc
				}
				if memAccess.Mode&dependencygraph2.ACCESS_PLAIN_READ != 0 {
					imgAcc.Read = true
				}
				if memAccess.Mode&dependencygraph2.ACCESS_PLAIN_WRITE != 0 {
					imgAcc.Write = true
				}
			}

			for _, buffer := range helpers.lookupBuffers(memAccess.Pool, memRange) {
				bufAcc, ok := helpers.wlInfo.bufferAccesses[buffer.Handle]
				if !ok {
					bufAcc = &api.FramegraphBufferAccess{
						Buffer: buffer,
					}
					helpers.wlInfo.bufferAccesses[buffer.Handle] = bufAcc
				}
				if memAccess.Mode&dependencygraph2.ACCESS_PLAIN_READ != 0 {
					bufAcc.Read = true
				}
				if memAccess.Mode&dependencygraph2.ACCESS_PLAIN_WRITE != 0 {
					bufAcc.Write = true
				}
			}
		}
	}

	// Ending of a workload
	switch cmdArgs.(type) {

	// End of renderpass
	case VkCmdEndRenderPassArgsʳ:
		if helpers.wlInfo == nil || helpers.wlInfo.renderpass == nil {
			panic("Renderpass ends without having started")
		}
		helpers.wlInfo.renderpass.EndSubCmdIdx = []uint64(subCmdIdx)
		helpers.endWorkload()

	// End of compute (duplicate code since we cannot 'fallthrough' in a type switch)
	case VkCmdDispatchArgsʳ:
		if helpers.wlInfo == nil || helpers.wlInfo.compute == nil {
			panic("Compute ends without having started")
		}
		helpers.endWorkload()
	case VkCmdDispatchBaseArgsʳ:
		if helpers.wlInfo == nil || helpers.wlInfo.compute == nil {
			panic("Compute ends without having started")
		}
		helpers.endWorkload()
	case VkCmdDispatchBaseKHRArgsʳ:
		if helpers.wlInfo == nil || helpers.wlInfo.compute == nil {
			panic("Compute ends without having started")
		}
		helpers.endWorkload()
	case VkCmdDispatchIndirectArgsʳ:
		if helpers.wlInfo == nil || helpers.wlInfo.compute == nil {
			panic("Compute ends without having started")
		}
		helpers.endWorkload()

	}
}

// GetFramegraph creates the framegraph of the given capture.
func (API) GetFramegraph(ctx context.Context, p *path.Capture) (*api.Framegraph, error) {
	config := dependencygraph2.DependencyGraphConfig{
		SaveNodeAccesses:    true,
		ReverseDependencies: true,
	}
	dependencyGraph, err := dependencygraph2.GetDependencyGraph(ctx, p, config)
	if err != nil {
		return nil, err
	}

	// postSubCmdCb effectively processes each subcommand to extract renderpass
	// info, while recording information into the helpers.
	helpers := &framegraphInfoHelpers{
		workloadInfos:     []*workloadInfo{},
		wlInfo:            nil,
		currWorkloadId:    uint64(0),
		parentCmdIdx:      uint64(0),
		lookupInitialized: false,
	}
	postSubCmdCb := func(state *api.GlobalState, subCmdIdx api.SubCmdIdx, cmd api.Cmd, i interface{}) {
		helpers.processSubCommand(ctx, dependencyGraph, state, subCmdIdx, cmd, i)
	}

	// Iterate on the capture commands to collect information
	c, err := capture.ResolveGraphicsFromPath(ctx, p)
	if err != nil {
		return nil, err
	}
	if err := sync.MutateWithSubcommands(ctx, p, c.Commands, nil, nil, postSubCmdCb); err != nil {
		return nil, err
	}

	updateDependencies(helpers.workloadInfos, dependencyGraph)

	// Build the framegraph nodes and edges from collected data.
	nodes := make([]*api.FramegraphNode, len(helpers.workloadInfos))
	for i, wlInfo := range helpers.workloadInfos {
		imgAcc := make([]*api.FramegraphImageAccess, len(wlInfo.imageAccesses))
		imgHandles := make([]uint64, 0, len(wlInfo.imageAccesses))
		for h := range wlInfo.imageAccesses {
			imgHandles = append(imgHandles, h)
		}
		sort.Slice(imgHandles, func(i, j int) bool { return imgHandles[i] < imgHandles[j] })
		for j, h := range imgHandles {
			imgAcc[j] = wlInfo.imageAccesses[h]
		}

		bufAcc := make([]*api.FramegraphBufferAccess, len(wlInfo.bufferAccesses))
		bufHandles := make([]uint64, 0, len(wlInfo.bufferAccesses))
		for h := range wlInfo.bufferAccesses {
			bufHandles = append(bufHandles, h)
		}
		sort.Slice(bufHandles, func(i, j int) bool { return bufHandles[i] < bufHandles[j] })
		for j, h := range bufHandles {
			bufAcc[j] = wlInfo.bufferAccesses[h]
		}

		nodes[i] = &api.FramegraphNode{
			Id: wlInfo.id,
		}
		switch {
		case wlInfo.renderpass != nil:
			wlInfo.renderpass.ImageAccess = imgAcc
			wlInfo.renderpass.BufferAccess = bufAcc
			nodes[i].Workload = &api.FramegraphNode_Renderpass{Renderpass: wlInfo.renderpass}
		case wlInfo.compute != nil:
			wlInfo.compute.ImageAccess = imgAcc
			wlInfo.compute.BufferAccess = bufAcc
			nodes[i].Workload = &api.FramegraphNode_Compute{Compute: wlInfo.compute}
		default:
			return nil, log.Errf(ctx, nil, "Invalid framegraph workload")
		}

	}

	edges := []*api.FramegraphEdge{}
	for _, wlInfo := range helpers.workloadInfos {
		for deps := range wlInfo.deps {
			edges = append(edges, &api.FramegraphEdge{
				// We want the graph to show the flow of how the frame is
				// created (rather than the flow of dependencies), so use the
				// dependency as the edge origin and wlInfo as the destination.
				Origin:      deps,
				Destination: wlInfo.id,
			})
		}
	}

	return &api.Framegraph{Nodes: nodes, Edges: edges}, nil
}

// updateDependencies establishes dependencies between workloads.
func updateDependencies(workloadInfos []*workloadInfo, dependencyGraph dependencygraph2.DependencyGraph) {
	// isInWorkload: node -> workload it belongs to.
	isInWorkload := map[dependencygraph2.NodeID]uint64{}
	for _, wlInfo := range workloadInfos {
		for _, n := range wlInfo.nodes {
			isInWorkload[n] = wlInfo.id
		}
	}
	// node2workloads: node -> set of workloads it depends on.
	node2workloads := map[dependencygraph2.NodeID]map[uint64]struct{}{}

	// For a given workload WL, for each of its node, explore the dependency
	// graph in reverse order to mark all the nodes dependending on WL until we
	// hit the node of another workload, which then depends on WL.
	for _, wlInfo := range workloadInfos {
		// markNode is recursive, so declare it before initializing it.
		var markNode func(dependencygraph2.NodeID) error
		markNode = func(node dependencygraph2.NodeID) error {
			if id, ok := isInWorkload[node]; ok {
				if id != wlInfo.id {
					// Reached a node that is inside another workload, so this
					// workload depends on wlInfo.
					workloadInfos[id].deps[wlInfo.id] = struct{}{}
				}
				return nil
			}
			if _, ok := node2workloads[node]; !ok {
				node2workloads[node] = map[uint64]struct{}{}
			}
			if _, ok := node2workloads[node][wlInfo.id]; ok {
				// Node already visited, stop recursion
				return nil
			}
			node2workloads[node][wlInfo.id] = struct{}{}
			return dependencyGraph.ForeachDependencyTo(node, markNode)
		}
		for _, node := range wlInfo.nodes {
			dependencyGraph.ForeachDependencyTo(node, markNode)
		}
	}
}
