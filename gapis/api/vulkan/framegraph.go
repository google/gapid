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
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/service/path"
)

func newFramegraphBuffer(buf *BufferObjectʳ) *api.FramegraphBuffer {
	return &api.FramegraphBuffer{
		Handle:       uint64(buf.VulkanHandle()),
		Size:         uint64(buf.Info().Size()),
		Usage:        uint32(buf.Info().Usage()),
		TransferSrc:  buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_SRC_BIT) != 0,
		TransferDst:  buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_TRANSFER_DST_BIT) != 0,
		UniformTexel: buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_TEXEL_BUFFER_BIT) != 0,
		StorageTexel: buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_STORAGE_TEXEL_BUFFER_BIT) != 0,
		Uniform:      buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_UNIFORM_BUFFER_BIT) != 0,
		Storage:      buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_STORAGE_BUFFER_BIT) != 0,
		Index:        buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_INDEX_BUFFER_BIT) != 0,
		Vertex:       buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_VERTEX_BUFFER_BIT) != 0,
		Indirect:     buf.Info().Usage()&VkBufferUsageFlags(VkBufferUsageFlagBits_VK_BUFFER_USAGE_INDIRECT_BUFFER_BIT) != 0,
	}
}

func newFramegraphImage(img *ImageObjectʳ) *api.FramegraphImage {
	format, err := getImageFormatFromVulkanFormat(img.Info().Fmt())
	if err != nil {
		panic("Unrecognized Vulkan image format")
	}
	nature := api.FramegraphImageNature_NONE
	if img.IsSwapchainImage() {
		nature = api.FramegraphImageNature_SWAPCHAIN
	} else if img.Info().Usage()&VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT) != 0 {
		nature = api.FramegraphImageNature_TRANSIENT
	}
	return &api.FramegraphImage{
		Handle:    uint64(img.VulkanHandle()),
		Usage:     uint32(img.Info().Usage()),
		ImageType: strings.TrimPrefix(fmt.Sprintf("%v", img.Info().ImageType()), "VK_IMAGE_TYPE_"),
		Nature:    nature,
		Info: &image.Info{
			Format: format,
			Width:  img.Info().Extent().Width(),
			Height: img.Info().Extent().Height(),
			Depth:  img.Info().Extent().Depth(),
		},
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

func newFramegraphAttachment(desc VkAttachmentDescription, imgView ImageViewObjectʳ, isDepthStencil bool) *api.FramegraphAttachment {
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
		Image:           newFramegraphImage(&imgObj),
	}
}

func newFramegraphSubpass(subpassDesc SubpassDescription, framebuffer FramebufferObjectʳ, renderpass RenderPassObjectʳ) *api.FramegraphSubpass {
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
		sp.Input[i] = newFramegraphAttachment(desc, imgView, false)
	}

	// Color attachments
	for i := 0; i < colorAtts.Len(); i++ {
		idx := colorAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		sp.Color[i] = newFramegraphAttachment(desc, imgView, false)
	}

	// Resolve attachments
	for i := 0; i < resolveAtts.Len(); i++ {
		idx := resolveAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		sp.Resolve[i] = newFramegraphAttachment(desc, imgView, false)
	}

	// DepthStencil attachment
	depthStencilAtt := subpassDesc.DepthStencilAttachment()
	if !depthStencilAtt.IsNil() {
		idx := depthStencilAtt.Attachment()
		if idx != VK_ATTACHMENT_UNUSED {
			desc := renderpass.AttachmentDescriptions().Get(idx)
			imgView := framebuffer.ImageAttachments().Get(idx)
			sp.DepthStencil = newFramegraphAttachment(desc, imgView, true)
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

// renderpassInfo stores a renderpass' info relevant for the framegraph.
type renderpassInfo struct {
	renderpass *api.FramegraphRenderpass
	id         uint64
	nodes      []dependencygraph2.NodeID

	// deps stores the set of renderpasses this renderpass depends on
	deps map[uint64]struct{}

	// imageAccesses is a temporary set that is eventually sorted and stored in
	// renderpass.ImageAccess list.
	imageAccesses map[VkImage]*api.FramegraphImageAccess
	// idem imageAccesses, but for buffers
	bufferAccesses map[VkBuffer]*api.FramegraphBufferAccess
}

// framegraphInfoHelpers contains variables that stores information while
// processing subCommands.
type framegraphInfoHelpers struct {
	rpInfos  []*renderpassInfo
	rpInfo   *renderpassInfo
	currRpId uint64
	// imageLookup is a lookup table to quickly find images that match a memory
	// access. Scanning the whole state to look for an image matching a memory
	// access is slow, this lookup table saves seconds of computation. It is
	// updated when a new renderpass is started under a new top-level parent
	// command: images can only be created/destroyed between top-level commands,
	// not during the execution of subcommands.
	imageLookup map[memory.PoolID]map[memory.Range][]*ImageObjectʳ
	// idem for buffers
	bufferLookup map[memory.PoolID]map[memory.Range][]*BufferObjectʳ
	parentCmdIdx uint64 // used to know when to update the lookup tables
}

// updateImageLookup updates the lookup table to quickly find an image matching a memory observation.
func (helpers *framegraphInfoHelpers) updateImageLookup(state *State) {
	helpers.imageLookup = make(map[memory.PoolID]map[memory.Range][]*ImageObjectʳ)
	for _, image := range state.Images().All() {
		for _, aspect := range image.Aspects().All() {
			for _, layer := range aspect.Layers().All() {
				for _, level := range layer.Levels().All() {
					pool := level.Data().Pool()
					memRange := level.Data().Range()
					if _, ok := helpers.imageLookup[pool]; !ok {
						helpers.imageLookup[pool] = make(map[memory.Range][]*ImageObjectʳ)
					}
					imgObj := image
					helpers.imageLookup[pool][memRange] = append(helpers.imageLookup[pool][memRange], &imgObj)
				}
			}
		}
	}
}

// lookupImages returns all the images that contain (pool, memRange).
func (helpers *framegraphInfoHelpers) lookupImages(pool memory.PoolID, memRange memory.Range) []*ImageObjectʳ {
	images := []*ImageObjectʳ{}
	for imgRange := range helpers.imageLookup[pool] {
		if imgRange.Includes(memRange) {
			images = append(images, helpers.imageLookup[pool][imgRange]...)
		}
	}
	return images
}

// updateBufferLookup updates the lookup table to quickly find a buffer matching a memory observation.
func (helpers *framegraphInfoHelpers) updateBufferLookup(state *State) {
	helpers.bufferLookup = make(map[memory.PoolID]map[memory.Range][]*BufferObjectʳ)
	for _, buffer := range state.Buffers().All() {
		pool := buffer.Memory().Data().Pool()
		memRange := memory.Range{
			Base: uint64(buffer.MemoryOffset()),
			Size: uint64(buffer.Info().Size()),
		}
		if _, ok := helpers.bufferLookup[pool]; !ok {
			helpers.bufferLookup[pool] = make(map[memory.Range][]*BufferObjectʳ)
		}
		bufObj := buffer
		helpers.bufferLookup[pool][memRange] = append(helpers.bufferLookup[pool][memRange], &bufObj)
	}
}

// lookupBuffers returns all the buffers that contain (pool, memRange).
func (helpers *framegraphInfoHelpers) lookupBuffers(pool memory.PoolID, memRange memory.Range) []*BufferObjectʳ {
	buffers := []*BufferObjectʳ{}
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

	// Beginning of renderpass
	if args, ok := cmdArgs.(VkCmdBeginRenderPassArgsʳ); ok {
		if helpers.rpInfo != nil {
			panic("Renderpass starts without having ended")
		}
		// Update image/buffer lookup tables if we change of top-level parent
		// command index: these cannot be created/destroyed during subcommands.
		parentCmdIdx := subCmdIdx[0]
		if parentCmdIdx == 0 || parentCmdIdx > helpers.parentCmdIdx {
			helpers.parentCmdIdx = parentCmdIdx
			helpers.updateImageLookup(vkState)
			helpers.updateBufferLookup(vkState)
		}

		framebuffer := vkState.Framebuffers().Get(args.Framebuffer())
		renderpass := vkState.RenderPasses().Get(args.RenderPass())
		subpassesDesc := renderpass.SubpassDescriptions()
		subpasses := make([]*api.FramegraphSubpass, subpassesDesc.Len())
		for i := 0; i < len(subpasses); i++ {
			subpassDesc := subpassesDesc.Get(uint32(i))
			subpasses[i] = newFramegraphSubpass(subpassDesc, framebuffer, renderpass)
		}

		helpers.rpInfo = &renderpassInfo{
			id:    helpers.currRpId,
			nodes: []dependencygraph2.NodeID{},
			deps:  make(map[uint64]struct{}),
			renderpass: &api.FramegraphRenderpass{
				Handle:            uint64(renderpass.VulkanHandle()),
				BeginSubCmdIdx:    []uint64(subCmdIdx),
				FramebufferWidth:  framebuffer.Width(),
				FramebufferHeight: framebuffer.Height(),
				FramebufferLayers: framebuffer.Layers(),
				Subpass:           subpasses,
			},
			imageAccesses:  make(map[VkImage]*api.FramegraphImageAccess),
			bufferAccesses: make(map[VkBuffer]*api.FramegraphBufferAccess),
		}
		helpers.currRpId++
	}

	// Process commands that are inside a renderpass
	if helpers.rpInfo != nil {
		nodeID := dependencyGraph.GetCmdNodeID(api.CmdID(subCmdIdx[0]), subCmdIdx[1:])
		helpers.rpInfo.nodes = append(helpers.rpInfo.nodes, nodeID)

		for _, memAccess := range dependencyGraph.GetNodeAccesses(nodeID).MemoryAccesses {
			memRange := memory.Range{
				Base: memAccess.Span.Start,
				Size: memAccess.Span.End - memAccess.Span.Start,
			}
			for _, image := range helpers.lookupImages(memAccess.Pool, memRange) {
				imgAcc, ok := helpers.rpInfo.imageAccesses[image.VulkanHandle()]
				if !ok {
					imgAcc = &api.FramegraphImageAccess{
						Image: newFramegraphImage(image),
					}
					helpers.rpInfo.imageAccesses[image.VulkanHandle()] = imgAcc
				}
				switch memAccess.Mode {
				case dependencygraph2.ACCESS_READ:
					imgAcc.Read = true
				case dependencygraph2.ACCESS_WRITE:
					imgAcc.Write = true
				}
			}
			buffers := helpers.lookupBuffers(memAccess.Pool, memRange)
			for _, buffer := range buffers {
				bufAcc, ok := helpers.rpInfo.bufferAccesses[buffer.VulkanHandle()]
				if !ok {
					bufAcc = &api.FramegraphBufferAccess{
						Buffer: newFramegraphBuffer(buffer),
					}
					helpers.rpInfo.bufferAccesses[buffer.VulkanHandle()] = bufAcc
				}
				switch memAccess.Mode {
				case dependencygraph2.ACCESS_READ:
					bufAcc.Read = true
				case dependencygraph2.ACCESS_WRITE:
					bufAcc.Write = true
				}
			}
		}
	}

	// Ending of renderpass
	if _, ok := cmdArgs.(VkCmdEndRenderPassArgsʳ); ok {
		if helpers.rpInfo == nil {
			panic("Renderpass ends without having started")
		}
		helpers.rpInfo.renderpass.EndSubCmdIdx = []uint64(subCmdIdx)
		helpers.rpInfos = append(helpers.rpInfos, helpers.rpInfo)
		helpers.rpInfo = nil
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
		rpInfos:      []*renderpassInfo{},
		rpInfo:       nil,
		currRpId:     uint64(0),
		parentCmdIdx: uint64(0),
		imageLookup:  make(map[memory.PoolID]map[memory.Range][]*ImageObjectʳ),
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

	updateDependencies(helpers.rpInfos, dependencyGraph)

	// Build the framegraph nodes and edges from collected data.
	nodes := make([]*api.FramegraphNode, len(helpers.rpInfos))
	for i, rpInfo := range helpers.rpInfos {
		rpInfo.renderpass.ImageAccess = make([]*api.FramegraphImageAccess, len(rpInfo.imageAccesses))
		imgHandles := make([]VkImage, 0, len(rpInfo.imageAccesses))
		for h := range rpInfo.imageAccesses {
			imgHandles = append(imgHandles, h)
		}
		sort.Slice(imgHandles, func(i, j int) bool { return imgHandles[i] < imgHandles[j] })
		for j, h := range imgHandles {
			rpInfo.renderpass.ImageAccess[j] = rpInfo.imageAccesses[h]
		}

		rpInfo.renderpass.BufferAccess = make([]*api.FramegraphBufferAccess, len(rpInfo.bufferAccesses))
		bufHandles := make([]VkBuffer, 0, len(rpInfo.bufferAccesses))
		for h := range rpInfo.bufferAccesses {
			bufHandles = append(bufHandles, h)
		}
		sort.Slice(bufHandles, func(i, j int) bool { return bufHandles[i] < bufHandles[j] })
		for j, h := range bufHandles {
			rpInfo.renderpass.BufferAccess[j] = rpInfo.bufferAccesses[h]
		}

		nodes[i] = &api.FramegraphNode{
			Id:       rpInfo.id,
			Workload: &api.FramegraphNode_Renderpass{Renderpass: rpInfo.renderpass},
		}
	}

	edges := []*api.FramegraphEdge{}
	for _, rpInfo := range helpers.rpInfos {
		for deps := range rpInfo.deps {
			edges = append(edges, &api.FramegraphEdge{
				// We want the graph to show the flow of how the frame is
				// created (rather than the flow of dependencies), so use the
				// dependency as the edge origin and rpInfo as the destination.
				Origin:      deps,
				Destination: rpInfo.id,
			})
		}
	}

	return &api.Framegraph{Nodes: nodes, Edges: edges}, nil
}

// updateDependencies establishes dependencies between renderpasses.
func updateDependencies(rpInfos []*renderpassInfo, dependencyGraph dependencygraph2.DependencyGraph) {
	// isInsideRenderpass: node -> renderpass it belongs to.
	isInsideRenderpass := map[dependencygraph2.NodeID]uint64{}
	for _, rpInfo := range rpInfos {
		for _, n := range rpInfo.nodes {
			isInsideRenderpass[n] = rpInfo.id
		}
	}
	// node2renderpasses: node -> set of renderpasses it depends on.
	node2renderpasses := map[dependencygraph2.NodeID]map[uint64]struct{}{}

	// For a given renderpass RP, for each of its node, explore the dependency
	// graph in reverse order to mark all the nodes dependending on RP until we
	// hit the node of another renderpass, which then depends on RP.
	for _, rpInfo := range rpInfos {
		// markNode is recursive, so declare it before initializing it.
		var markNode func(dependencygraph2.NodeID) error
		markNode = func(node dependencygraph2.NodeID) error {
			if id, ok := isInsideRenderpass[node]; ok {
				if id != rpInfo.id {
					// Reached a node that is inside another renderpass, so this
					// renderpass depends on rpInfo.
					rpInfos[id].deps[rpInfo.id] = struct{}{}
				}
				return nil
			}
			if _, ok := node2renderpasses[node]; !ok {
				node2renderpasses[node] = map[uint64]struct{}{}
			}
			if _, ok := node2renderpasses[node][rpInfo.id]; ok {
				// Node already visited, stop recursion
				return nil
			}
			node2renderpasses[node][rpInfo.id] = struct{}{}
			return dependencyGraph.ForeachDependencyTo(node, markNode)
		}
		for _, node := range rpInfo.nodes {
			dependencyGraph.ForeachDependencyTo(node, markNode)
		}
	}
}
