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
	"strings"

	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/api/sync"
	"github.com/google/gapid/gapis/capture"
	"github.com/google/gapid/gapis/resolve/dependencygraph2"
	"github.com/google/gapid/gapis/service/path"
)

type attachmentInfo struct {
	isSwapchainImage bool
	format           VkFormat
	imgType          VkImageViewType
	loadOp           VkAttachmentLoadOp
	storeOp          VkAttachmentStoreOp
	usage            VkImageUsageFlags
	handle           VkImageView
	width            uint32
	height           uint32
	depth            uint32
}

func (a *attachmentInfo) String() string {
	if a == nil {
		return "unused"
	}
	isSwapChain := ""
	if a.isSwapchainImage {
		isSwapChain = " [swapchain]"
	}
	transient := ""
	if (a.usage & VkImageUsageFlags(VkImageUsageFlagBits_VK_IMAGE_USAGE_TRANSIENT_ATTACHMENT_BIT)) != 0 {
		transient = " [transient]"
	}
	load := strings.TrimPrefix(fmt.Sprintf("%v", a.loadOp), "VK_ATTACHMENT_LOAD_OP_")
	store := strings.TrimPrefix(fmt.Sprintf("%v", a.storeOp), "VK_ATTACHMENT_STORE_OP_")
	imgType := strings.TrimPrefix(fmt.Sprintf("%v", a.imgType), "VK_IMAGE_VIEW_TYPE_")
	format := strings.TrimPrefix(fmt.Sprintf("%v", a.format), "VK_FORMAT_")
	return fmt.Sprintf("0x%X%s%s %vx%vx%v load:%s store:%s %v %v", a.handle, isSwapChain, transient, a.width, a.height, a.depth, load, store, imgType, format)
}

func newAttachmentInfo(desc VkAttachmentDescription, imgView ImageViewObjectʳ, isDepthStencil bool) *attachmentInfo {
	imgInfo := imgView.Image().Info()
	var loadOp VkAttachmentLoadOp
	var storeOp VkAttachmentStoreOp
	if isDepthStencil {
		loadOp = desc.StencilLoadOp()
		storeOp = desc.StencilStoreOp()
	} else {
		loadOp = desc.LoadOp()
		storeOp = desc.StoreOp()
	}
	return &attachmentInfo{
		isSwapchainImage: imgView.Image().IsSwapchainImage(),
		format:           desc.Fmt(),
		imgType:          imgView.Type(),
		loadOp:           loadOp,
		storeOp:          storeOp,
		usage:            imgInfo.Usage(),
		handle:           imgView.VulkanHandle(),
		width:            imgInfo.Extent().Width(),
		height:           imgInfo.Extent().Height(),
		depth:            imgInfo.Extent().Depth(),
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
}

// framegraphInfoHelpers contains variables that stores information while
// processing subCommands.
type framegraphInfoHelpers struct {
	rpInfos  []*renderpassInfo
	rpInfo   *renderpassInfo
	currRpId uint64
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
		}
		helpers.currRpId++
	}

	// Process commands that are inside a renderpass
	if helpers.rpInfo != nil {
		nodeID := dependencyGraph.GetCmdNodeID(api.CmdID(subCmdIdx[0]), subCmdIdx[1:])
		helpers.rpInfo.nodes = append(helpers.rpInfo.nodes, nodeID)
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
		rpInfos:  []*renderpassInfo{},
		rpInfo:   nil,
		currRpId: uint64(0),
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
