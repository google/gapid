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

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/memory"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/service/path"
)

type imageInfo struct {
	handle           VkImage
	width            uint32
	height           uint32
	depth            uint32
	usage            VkImageUsageFlags
	imgType          VkImageType
	format           VkFormat
	isSwapchainImage bool
}

func (img *imageInfo) String() string {
	swapchain := ""
	if img.isSwapchainImage {
		swapchain = " swapchain"
	}
	transient := ""
	if (img.usage & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT)) != 0 {
		transient = " transient"
	}
	imgType := strings.TrimPrefix(fmt.Sprintf("%v", img.imgType), "VK_IMAGE_TYPE_")
	imgFormat := strings.TrimPrefix(fmt.Sprintf("%v", img.format), "VK_FORMAT_")
	return fmt.Sprintf("[Img %v%s%s %s %s %vx%vx%v]", img.handle, swapchain, transient, imgType, imgFormat, img.width, img.height, img.depth)
}

func newImageInfo(image *ImageObjectʳ) *imageInfo {
	return &imageInfo{
		handle:           image.VulkanHandle(),
		width:            image.Info().Extent().Width(),
		height:           image.Info().Extent().Height(),
		depth:            image.Info().Extent().Depth(),
		usage:            image.Info().Usage(),
		imgType:          image.Info().ImageType(),
		format:           image.Info().Fmt(),
		isSwapchainImage: image.IsSwapchainImage(),
	}
}

type attachmentInfo struct {
	loadOp        VkAttachmentLoadOp
	storeOp       VkAttachmentStoreOp
	imgViewHandle VkImageView
	imgViewType   VkImageViewType
	imgViewFormat VkFormat
	img           *imageInfo
}

func (a *attachmentInfo) String() string {
	if a == nil {
		return "unused"
	}

	load := strings.TrimPrefix(fmt.Sprintf("%v", a.loadOp), "VK_ATTACHMENT_LOAD_OP_")
	store := strings.TrimPrefix(fmt.Sprintf("%v", a.storeOp), "VK_ATTACHMENT_STORE_OP_")
	imgViewType := strings.TrimPrefix(fmt.Sprintf("%v", a.imgViewType), "VK_IMAGE_VIEW_TYPE_")
	imgViewFormat := strings.TrimPrefix(fmt.Sprintf("%v", a.imgViewFormat), "VK_FORMAT_")
	att := fmt.Sprintf("load:%s store:%s [View %v %s %s]", load, store, a.imgViewHandle, imgViewType, imgViewFormat)

	return fmt.Sprintf("%s %v", att, a.img)
}

func newAttachmentInfo(desc VkAttachmentDescription, imgView ImageViewObjectʳ, isDepthStencil bool) *attachmentInfo {
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
	return &attachmentInfo{
		loadOp:        loadOp,
		storeOp:       storeOp,
		imgViewHandle: imgView.VulkanHandle(),
		imgViewType:   imgView.Type(),
		imgViewFormat: imgView.Fmt(),
		img:           newImageInfo(&imgObj),
	}
}

type subpassInfo struct {
	inputAttachments       []*attachmentInfo
	colorAttachments       []*attachmentInfo
	resolveAttachments     []*attachmentInfo
	depthStencilAttachment *attachmentInfo
}

func newSubpassInfo(subpassDesc SubpassDescription, framebuffer FramebufferObjectʳ, renderpass RenderPassObjectʳ) *subpassInfo {
	inputAtts := subpassDesc.InputAttachments()
	colorAtts := subpassDesc.ColorAttachments()
	resolveAtts := subpassDesc.ResolveAttachments()
	spInfo := &subpassInfo{
		inputAttachments:   make([]*attachmentInfo, inputAtts.Len()),
		colorAttachments:   make([]*attachmentInfo, colorAtts.Len()),
		resolveAttachments: make([]*attachmentInfo, resolveAtts.Len()),
	}

	// Input attachments
	for i := 0; i < inputAtts.Len(); i++ {
		idx := inputAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		spInfo.inputAttachments[i] = newAttachmentInfo(desc, imgView, false)
	}

	// Color attachments
	for i := 0; i < colorAtts.Len(); i++ {
		idx := colorAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		spInfo.colorAttachments[i] = newAttachmentInfo(desc, imgView, false)
	}

	// Resolve attachments
	for i := 0; i < resolveAtts.Len(); i++ {
		idx := resolveAtts.Get(uint32(i)).Attachment()
		if idx == VK_ATTACHMENT_UNUSED {
			continue
		}
		desc := renderpass.AttachmentDescriptions().Get(idx)
		imgView := framebuffer.ImageAttachments().Get(idx)
		spInfo.resolveAttachments[i] = newAttachmentInfo(desc, imgView, false)
	}

	// DepthStencil attachment
	depthStencilAtt := subpassDesc.DepthStencilAttachment()
	if !depthStencilAtt.IsNil() {
		idx := depthStencilAtt.Attachment()
		if idx != VK_ATTACHMENT_UNUSED {
			desc := renderpass.AttachmentDescriptions().Get(idx)
			imgView := framebuffer.ImageAttachments().Get(idx)
			spInfo.depthStencilAttachment = newAttachmentInfo(desc, imgView, true)
		}
	}

	return spInfo
}

type imageAccessInfo struct {
	read  bool
	write bool
	img   *imageInfo
}

func (i *imageAccessInfo) String() string {
	r := "-"
	if i.read {
		r = "r"
	}
	w := "-"
	if i.write {
		w = "w"
	}
	return fmt.Sprintf("%s%s %v", r, w, i.img)
}

// renderpassInfo stores a renderpass' info relevant for the framegraph.
type renderpassInfo struct {
	id             uint64
	beginIdx       api.SubCmdIdx
	endIdx         api.SubCmdIdx
	nodes          []dependencygraph2.NodeID
	deps           map[uint64]struct{} // set of renderpasses this renderpass depends on
	subpasses      []*subpassInfo
	framebufWidth  uint32
	framebufHeight uint32
	framebufLayers uint32
	imageAccesses  map[VkImage]*imageAccessInfo
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
	imageLookup  map[memory.PoolID]map[memory.Range][]*ImageObjectʳ
	parentCmdIdx uint64 // used to know when to update imageLookup
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
		// Update image lookup table if we change of top-level parent command
		// index: images can only be created/destroyed between submits.
		parentCmdIdx := subCmdIdx[0]
		if parentCmdIdx == 0 || parentCmdIdx > helpers.parentCmdIdx {
			helpers.parentCmdIdx = parentCmdIdx
			helpers.updateImageLookup(vkState)
		}

		framebuffer := vkState.Framebuffers().Get(args.Framebuffer())
		renderpass := vkState.RenderPasses().Get(args.RenderPass())
		subpassesDesc := renderpass.SubpassDescriptions()
		subpasses := make([]*subpassInfo, subpassesDesc.Len())
		for i := 0; i < len(subpasses); i++ {
			subpassDesc := subpassesDesc.Get(uint32(i))
			subpasses[i] = newSubpassInfo(subpassDesc, framebuffer, renderpass)
		}

		helpers.rpInfo = &renderpassInfo{
			id:             helpers.currRpId,
			beginIdx:       subCmdIdx,
			nodes:          []dependencygraph2.NodeID{},
			deps:           make(map[uint64]struct{}),
			subpasses:      subpasses,
			framebufWidth:  framebuffer.Width(),
			framebufHeight: framebuffer.Height(),
			framebufLayers: framebuffer.Layers(),
			imageAccesses:  make(map[VkImage]*imageAccessInfo),
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
					imgAcc = &imageAccessInfo{
						img: newImageInfo(image),
					}
					helpers.rpInfo.imageAccesses[image.VulkanHandle()] = imgAcc
				}
				switch memAccess.Mode {
				case dependencygraph2.ACCESS_READ:
					imgAcc.read = true
				case dependencygraph2.ACCESS_WRITE:
					imgAcc.write = true
				}
			}
		}
	}

	// Ending of renderpass
	if _, ok := cmdArgs.(VkCmdEndRenderPassArgsʳ); ok {
		if helpers.rpInfo == nil {
			panic("Renderpass ends without having started")
		}
		helpers.rpInfo.endIdx = subCmdIdx
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
		// Graphviz DOT: use "\l" as a newline to obtain left-aligned text.
		text := fmt.Sprintf("Renderpass %v\\lbegin:%v\\lend:  %v\\lFramebuffer: %vx%vx%v\\l", rpInfo.id, rpInfo.beginIdx, rpInfo.endIdx, rpInfo.framebufWidth, rpInfo.framebufHeight, rpInfo.framebufLayers)
		for i, subpass := range rpInfo.subpasses {
			text += fmt.Sprintf("\\lSubpass %v\\l", i)
			for j, a := range subpass.inputAttachments {
				text += fmt.Sprintf("input(%v): %v\\l", j, a)
			}
			for j, a := range subpass.colorAttachments {
				text += fmt.Sprintf("color(%v): %v\\l", j, a)
			}
			for j, a := range subpass.resolveAttachments {
				text += fmt.Sprintf("resolve(%v): %v\\l", j, a)
			}
			text += fmt.Sprintf("depth/stencil: %v\\l", subpass.depthStencilAttachment)
		}

		if len(rpInfo.imageAccesses) > 0 {
			text += "\\lImage accesses:\\l"
			handles := make([]VkImage, 0, len(rpInfo.imageAccesses))
			for h := range rpInfo.imageAccesses {
				handles = append(handles, h)
			}
			sort.Slice(handles, func(i, j int) bool { return handles[i] < handles[j] })
			for _, h := range handles {
				text += fmt.Sprintf("%v\\l", rpInfo.imageAccesses[h])
			}
		}

		nodes[i] = &api.FramegraphNode{
			Id:   rpInfo.id,
			Text: text,
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
