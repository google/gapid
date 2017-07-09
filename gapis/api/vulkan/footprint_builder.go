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

	"github.com/google/gapid/core/log"
	"github.com/google/gapid/core/math/interval"
	"github.com/google/gapid/gapis/api"
	"github.com/google/gapid/gapis/config"
	"github.com/google/gapid/gapis/resolve/dependencygraph"
)

var emptyAtoms = []dependencygraph.DefUseAtom{}

const vkWholeSize = uint64(0xFFFFFFFFFFFFFFFF)
const vkAttachmentUnused = uint32(0xFFFFFFFF)
const vkNullHandle = vkHandle(0x0)

// Assume the value of a Vulkan handle is always unique
type vkHandle uint64

func (vkHandle) DefUseAtom() {}

// label
type label struct {
	uint64
}

var nextLabelVal uint64 = 1

func (label) DefUseAtom() {}

func newLabel() label { i := nextLabelVal; nextLabelVal++; return label{i} }

// Forward-paired label
type forwardPairedLabel struct {
	labelReadBehaviours []*dependencygraph.Behaviour
}

func (*forwardPairedLabel) DefUseAtom() {}
func newForwardPairedLabel(ctx context.Context,
	bh *dependencygraph.Behaviour) *forwardPairedLabel {
	fpl := &forwardPairedLabel{labelReadBehaviours: []*dependencygraph.Behaviour{}}
	write(ctx, bh, fpl)
	return fpl
}

// memory
type memorySpan struct {
	span   interval.U64Span
	memory VkDeviceMemory
}

func (memorySpan) DefUseAtom() {}

// commands
type commandBufferCommand struct {
	isCmdExecuteCommands    bool
	secondaryCommandBuffers []VkCommandBuffer
	behave                  func(submittedCommand, *queueExecutionInfo)
}

func (cbc *commandBufferCommand) newBehaviour(ctx context.Context,
	sc submittedCommand, m *vulkanMachine,
	qei *queueExecutionInfo) *dependencygraph.Behaviour {
	bh := dependencygraph.NewBehaviour(sc.id, m)
	read(ctx, bh, cbc)
	read(ctx, bh, qei.currentSubmitInfo.executionBegin)
	if sc.parentCmd != nil {
		read(ctx, bh, sc.parentCmd)
	}
	return bh
}

func (*commandBufferCommand) DefUseAtom() {}

func (cbc *commandBufferCommand) recordSecondaryCommandBuffer(vkCb VkCommandBuffer) {
	cbc.secondaryCommandBuffers = append(cbc.secondaryCommandBuffers, vkCb)
}

// vulkanMachine is the back-propagation machine for Vulkan API commands.
type vulkanMachine struct {
	handles               map[vkHandle]struct{}
	labels                map[label]struct{}
	memories              map[VkDeviceMemory]*interval.U64SpanList
	commandBufferCommands map[*commandBufferCommand]struct{}
	subpassIndices        map[*subpassIndex]struct{}
	boundDataPieces       map[*boundData]struct{}
	descriptors           map[*descriptor]struct{}
	boundDescriptorSets   map[*boundDescriptorSet]struct{}
	forwardPairedLabels   map[*forwardPairedLabel]struct{}
}

func newVulkanMachine() *vulkanMachine {
	return &vulkanMachine{
		handles:               map[vkHandle]struct{}{},
		labels:                map[label]struct{}{},
		memories:              map[VkDeviceMemory]*interval.U64SpanList{},
		commandBufferCommands: map[*commandBufferCommand]struct{}{},
		subpassIndices:        map[*subpassIndex]struct{}{},
		boundDataPieces:       map[*boundData]struct{}{},
		descriptors:           map[*descriptor]struct{}{},
		boundDescriptorSets:   map[*boundDescriptorSet]struct{}{},
		forwardPairedLabels:   map[*forwardPairedLabel]struct{}{},
	}
}

func (m *vulkanMachine) Clear() {
	m.handles = map[vkHandle]struct{}{}
	m.labels = map[label]struct{}{}
	m.memories = map[VkDeviceMemory]*interval.U64SpanList{}
	m.commandBufferCommands = map[*commandBufferCommand]struct{}{}
	m.subpassIndices = map[*subpassIndex]struct{}{}
	m.boundDataPieces = map[*boundData]struct{}{}
	m.descriptors = map[*descriptor]struct{}{}
	m.boundDescriptorSets = map[*boundDescriptorSet]struct{}{}
	m.forwardPairedLabels = map[*forwardPairedLabel]struct{}{}
}

func (m *vulkanMachine) IsAlive(behaviourIndex uint64,
	ft *dependencygraph.Footprint) bool {
	bh := ft.Behaviours[behaviourIndex]
	for _, w := range bh.Writes {
		if m.checkDef(w) {
			return true
		}
	}
	return false
}

func (m *vulkanMachine) RecordBehaviourEffects(behaviourIndex uint64,
	ft *dependencygraph.Footprint) []uint64 {
	bh := ft.Behaviours[behaviourIndex]
	aliveIndices := []uint64{}
	aliveIndices = append(aliveIndices, behaviourIndex)
	for _, w := range bh.Writes {
		extraAliveBehaviours := m.def(w)
		for _, eb := range extraAliveBehaviours {
			aliveIndices = append(aliveIndices, ft.BehaviourIndices[eb])
		}
	}
	for _, r := range bh.Reads {
		m.use(r)
	}
	return aliveIndices
}

func (m *vulkanMachine) checkDef(c dependencygraph.DefUseAtom) bool {
	switch c := c.(type) {
	case vkHandle:
		if _, ok := m.handles[c]; ok {
			return true
		}
	case label:
		if _, ok := m.labels[c]; ok {
			return true
		}
	case memorySpan:
		if _, exist := m.memories[c.memory]; exist {
			if _, count := interval.Intersect(m.memories[c.memory], c.span); count > 0 {
				return true
			}
		}
	case *commandBufferCommand:
		if _, ok := m.commandBufferCommands[c]; ok {
			return true
		}
	case *subpassIndex:
		if _, ok := m.subpassIndices[c]; ok {
			return true
		}
	case *boundData:
		if _, ok := m.boundDataPieces[c]; ok {
			return true
		}
	case *descriptor:
		if _, ok := m.descriptors[c]; ok {
			return true
		}
	case *boundDescriptorSet:
		if _, ok := m.boundDescriptorSets[c]; ok {
			return true
		}
	case *forwardPairedLabel:
		if _, ok := m.forwardPairedLabels[c]; ok {
			return true
		}
	default:
		return false
	}
	return false
}

func (m *vulkanMachine) use(c dependencygraph.DefUseAtom) {
	switch c := c.(type) {
	case vkHandle:
		m.handles[c] = struct{}{}
	case label:
		m.labels[c] = struct{}{}
	case memorySpan:
		if _, exist := m.memories[c.memory]; !exist {
			m.memories[c.memory] = &interval.U64SpanList{}
		}
		interval.Merge(m.memories[c.memory], c.span, true)
		m.handles[vkHandle(c.memory)] = struct{}{}
	case *commandBufferCommand:
		m.commandBufferCommands[c] = struct{}{}
	case *subpassIndex:
		m.subpassIndices[c] = struct{}{}
	case *boundData:
		m.boundDataPieces[c] = struct{}{}
	case *descriptor:
		m.descriptors[c] = struct{}{}
	case *boundDescriptorSet:
		m.boundDescriptorSets[c] = struct{}{}
	case *forwardPairedLabel:
		m.forwardPairedLabels[c] = struct{}{}
	default:
	}
}

func (m *vulkanMachine) def(c dependencygraph.DefUseAtom) []*dependencygraph.Behaviour {
	switch c := c.(type) {
	case vkHandle:
		delete(m.handles, c)
	case label:
		delete(m.labels, c)
	case memorySpan:
		if _, exist := m.memories[c.memory]; exist {
			interval.Remove(m.memories[c.memory], c.span)
		}
	case *commandBufferCommand:
		delete(m.commandBufferCommands, c)
	case *subpassIndex:
		delete(m.subpassIndices, c)
	case *boundData:
		delete(m.boundDataPieces, c)
	case *descriptor:
		delete(m.descriptors, c)
	case *boundDescriptorSet:
		delete(m.boundDescriptorSets, c)
	case *forwardPairedLabel:
		// For forward paired labels, if a label is defined, the reading
		// behaviours of the label must also be kept alive. This is used for
		// begin-end pairs, like vkCmdBeginRenderPass and vkCmdEndRenderPass.
		// E.g.: If any rendering output of a render pass is used in future, the
		// vkCmdBeginRenderPass should be kept alive, then the paird
		// vkCmdEndRenderPass must also be kept alive, no matter whether the
		// rendering output of the last subpass is used or not.
		alivePairedBehaviours := []*dependencygraph.Behaviour{}
		alivePairedBehaviours = append(alivePairedBehaviours, c.labelReadBehaviours...)
		delete(m.forwardPairedLabels, c)
		return alivePairedBehaviours
	default:
	}
	return []*dependencygraph.Behaviour{}
}

// submittedCommand represents Subcommands. When a submidttedCommand/Subcommand
// is executed, it reads the original commands, and if it is secondary command,
// its parent primary command.
type submittedCommand struct {
	id        api.SubCmdIdx
	cmd       *commandBufferCommand
	parentCmd *commandBufferCommand
}

func newSubmittedCommand(fullCmdIndex api.SubCmdIdx,
	c *commandBufferCommand, pc *commandBufferCommand) *submittedCommand {
	return &submittedCommand{id: fullCmdIndex, cmd: c, parentCmd: pc}
}

func (sc *submittedCommand) runCommand(ctx context.Context,
	ft *dependencygraph.Footprint, m *vulkanMachine,
	execInfo *queueExecutionInfo) {
	sc.cmd.behave(*sc, execInfo)
}

type queueSubmitInfo struct {
	queue            VkQueue
	executionBegin   label
	executionEnd     label
	waitSemaphores   []VkSemaphore
	signalSemaphores []VkSemaphore
	signalFence      VkFence
	pendingCommands  []*submittedCommand
}

type event struct {
	signal   label
	unsignal label
}

type fence struct {
	signal   label
	unsignal label
}

type query struct {
	reset  label
	begin  label
	end    label
	result label
}

func newQuery() *query {
	return &query{
		reset:  newLabel(),
		begin:  newLabel(),
		end:    newLabel(),
		result: newLabel(),
	}
}

type queryPool struct {
	queries []*query
}

type subpassAttachmentInfo struct {
	fullImageData bool
	data          dependencygraph.DefUseAtom
	desc          VkAttachmentDescription
}

type subpassInfo struct {
	colorAttachments       []*subpassAttachmentInfo
	resolveAttachments     []*subpassAttachmentInfo
	inputAttachments       []*subpassAttachmentInfo
	depthStencilAttachment *subpassAttachmentInfo
	modifiedDescriptorData []dependencygraph.DefUseAtom
}

type subpassIndex struct {
	val uint64
}

func (*subpassIndex) DefUseAtom() {}

type commandBufferExecutionState struct {
	vertexBuffers  map[uint32]*boundData
	indexBuffer    *boundData
	indexType      VkIndexType
	descriptorSets map[uint32]*boundDescriptorSet
	pipeline       label
	dynamicState   label
}

func newCommandBufferExecutionState() *commandBufferExecutionState {
	return &commandBufferExecutionState{
		vertexBuffers:  map[uint32]*boundData{},
		descriptorSets: map[uint32]*boundDescriptorSet{},
		pipeline:       newLabel(),
		dynamicState:   newLabel(),
	}
}

type queueExecutionInfo struct {
	currentCmdBufState   *commandBufferExecutionState
	primaryCmdBufState   *commandBufferExecutionState
	secondaryCmdBufState *commandBufferExecutionState

	subpasses       []subpassInfo
	subpass         *subpassIndex
	renderPassBegin *forwardPairedLabel

	currentCommand api.SubCmdIdx

	framebuffer *FramebufferObject

	lastSubmitID      api.CmdID
	currentSubmitInfo *queueSubmitInfo
}

func newQueueExecutionInfo(id api.CmdID) *queueExecutionInfo {
	return &queueExecutionInfo{
		subpasses:      []subpassInfo{},
		lastSubmitID:   id,
		currentCommand: api.SubCmdIdx([]uint64{0, 0, 0, 0}),
	}
}

func (qei *queueExecutionInfo) updateCurrentCommand(ctx context.Context,
	fci api.SubCmdIdx) {
	switch len(fci) {
	case 4:
		current := api.SubCmdIdx(qei.currentCommand[0:3])
		comming := api.SubCmdIdx(fci[0:3])
		if current.LessThan(comming) {
			// primary command buffer changed
			qei.primaryCmdBufState = newCommandBufferExecutionState()
		}
		qei.currentCmdBufState = qei.primaryCmdBufState
	case 6:
		if len(qei.currentCommand) != 6 {
			// Transit from primary command buffer to secondary command buffer
			qei.secondaryCmdBufState = newCommandBufferExecutionState()
		} else {
			current := api.SubCmdIdx(qei.currentCommand[0:5])
			comming := api.SubCmdIdx(fci[0:5])
			if current.LessThan(comming) {
				// secondary command buffer changed
				qei.secondaryCmdBufState = newCommandBufferExecutionState()
			}
		}
		qei.currentCmdBufState = qei.secondaryCmdBufState
	default:
		log.E(ctx, "Invalid length of full command index")
	}
	qei.currentCommand = fci
}

func (qei *queueExecutionInfo) startSubpass(ctx context.Context,
	bh *dependencygraph.Behaviour) {
	write(ctx, bh, qei.subpass)
	subpassI := qei.subpass.val
	handleLoadOp := func(ctx context.Context, bh *dependencygraph.Behaviour,
		attachment *subpassAttachmentInfo) {
		switch attachment.desc.LoadOp {
		case VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR,
			VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE:
			if attachment.fullImageData {
				write(ctx, bh, attachment.data)
			} else {
				modify(ctx, bh, attachment.data)
			}
		default:
			read(ctx, bh, attachment.data)
		}
	}
	for _, c := range qei.subpasses[subpassI].colorAttachments {
		handleLoadOp(ctx, bh, c)
	}
	for _, r := range qei.subpasses[subpassI].resolveAttachments {
		handleLoadOp(ctx, bh, r)
	}
	for _, i := range qei.subpasses[subpassI].inputAttachments {
		handleLoadOp(ctx, bh, i)
	}
	if qei.subpasses[subpassI].depthStencilAttachment != nil {
		dsAtt := qei.subpasses[subpassI].depthStencilAttachment
		if (dsAtt.desc.LoadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR ||
			dsAtt.desc.LoadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE) &&
			(dsAtt.desc.StencilLoadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_CLEAR ||
				dsAtt.desc.StencilLoadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_DONT_CARE) {
			if dsAtt.fullImageData {
				write(ctx, bh, dsAtt.data)
			} else {
				modify(ctx, bh, dsAtt.data)
			}
		} else if dsAtt.desc.LoadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD &&
			dsAtt.desc.StencilLoadOp == VkAttachmentLoadOp_VK_ATTACHMENT_LOAD_OP_LOAD {
			read(ctx, bh, dsAtt.data)
		} else {
			modify(ctx, bh, dsAtt.data)
		}
	}
}

func (qei *queueExecutionInfo) emitSubpassOutput(ctx context.Context,
	ft *dependencygraph.Footprint, sc submittedCommand, m *vulkanMachine) {
	subpassI := qei.subpass.val
	handleStoreOp := func(ctx context.Context, ft *dependencygraph.Footprint,
		sc submittedCommand, m *vulkanMachine, att *subpassAttachmentInfo,
		readAtt *subpassAttachmentInfo) {
		bh := sc.cmd.newBehaviour(ctx, sc, m, qei)
		if readAtt != nil {
			read(ctx, bh, readAtt.data)
		}
		switch att.desc.StoreOp {
		case VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE:
			modify(ctx, bh, att.data)
		case VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_DONT_CARE:
			if att.fullImageData {
				write(ctx, bh, att.data)
			} else {
				modify(ctx, bh, att.data)
			}
		}
		read(ctx, bh, qei.subpass)
		ft.AddBehaviour(ctx, bh)
	}
	for i, r := range qei.subpasses[subpassI].resolveAttachments {
		c := qei.subpasses[subpassI].colorAttachments[i]
		handleStoreOp(ctx, ft, sc, m, r, c)
	}
	for _, c := range qei.subpasses[subpassI].colorAttachments {
		handleStoreOp(ctx, ft, sc, m, c, nil)
	}
	for _, i := range qei.subpasses[subpassI].inputAttachments {
		handleStoreOp(ctx, ft, sc, m, i, nil)
	}
	if qei.subpasses[subpassI].depthStencilAttachment != nil {
		bh := sc.cmd.newBehaviour(ctx, sc, m, qei)
		dsAtt := qei.subpasses[subpassI].depthStencilAttachment
		if dsAtt.desc.StoreOp == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE ||
			dsAtt.desc.StencilStoreOp == VkAttachmentStoreOp_VK_ATTACHMENT_STORE_OP_STORE {
			modify(ctx, bh, dsAtt.data)
		} else {
			if dsAtt.fullImageData {
				write(ctx, bh, dsAtt.data)
			} else {
				modify(ctx, bh, dsAtt.data)
			}
		}
		read(ctx, bh, qei.subpass)
		ft.AddBehaviour(ctx, bh)
	}
	for _, modified := range qei.subpasses[subpassI].modifiedDescriptorData {
		bh := sc.cmd.newBehaviour(ctx, sc, m, qei)
		modify(ctx, bh, modified)
		read(ctx, bh, qei.subpass)
		ft.AddBehaviour(ctx, bh)
	}
}

func (qei *queueExecutionInfo) endSubpass(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behaviour,
	sc submittedCommand, m *vulkanMachine) {
	qei.emitSubpassOutput(ctx, ft, sc, m)
	read(ctx, bh, qei.subpass)
}

func (qei *queueExecutionInfo) beginRenderPass(ctx context.Context,
	vb *VulkanFootprintBuilder, bh *dependencygraph.Behaviour,
	rp *RenderPassObject, fb *FramebufferObject) {
	read(ctx, bh, vkHandle(rp.VulkanHandle))
	read(ctx, bh, vkHandle(fb.VulkanHandle))
	qei.framebuffer = fb
	qei.subpasses = []subpassInfo{}
	for _, subpass := range rp.SubpassDescriptions.KeysSorted() {
		desc := rp.SubpassDescriptions.Get(subpass)
		qei.subpasses = append(qei.subpasses, subpassInfo{})
		if subpass != uint32(len(qei.subpasses)-1) {
			log.E(ctx, "Cannot get subpass info, subpass: %v, length of info: %v",
				subpass, uint32(len(qei.subpasses)))
		}
		colorAs := map[uint32]struct{}{}
		resolveAs := map[uint32]struct{}{}
		inputAs := map[uint32]struct{}{}

		for _, ref := range desc.ColorAttachments {
			if ref.Attachment != vkAttachmentUnused {
				colorAs[ref.Attachment] = struct{}{}
			}
		}
		for _, ref := range desc.ResolveAttachments {
			if ref.Attachment != vkAttachmentUnused {
				resolveAs[ref.Attachment] = struct{}{}
			}
		}
		for _, ref := range desc.InputAttachments {
			if ref.Attachment != vkAttachmentUnused {
				inputAs[ref.Attachment] = struct{}{}
			}
		}
		// TODO: handle preserveAttachments

		for _, ai := range rp.AttachmentDescriptions.KeysSorted() {
			viewObj := fb.ImageAttachments.Get(ai)
			read(ctx, bh, vkHandle(viewObj.VulkanHandle))
			imgObj := viewObj.Image
			imgData := vb.getImageData(ctx, bh, imgObj.VulkanHandle)
			attDesc := rp.AttachmentDescriptions.Get(ai)
			fullImageData := false
			switch viewObj.Type {
			case VkImageViewType_VK_IMAGE_VIEW_TYPE_2D,
				VkImageViewType_VK_IMAGE_VIEW_TYPE_2D_ARRAY:
				if viewObj.SubresourceRange.BaseArrayLayer == uint32(0) &&
					imgObj.Info.ArrayLayers == viewObj.SubresourceRange.LayerCount &&
					imgObj.Info.ImageType == VkImageType_VK_IMAGE_TYPE_2D &&
					imgObj.Info.Extent.Width == fb.Width &&
					imgObj.Info.Extent.Height == fb.Height &&
					fb.Layers == imgObj.Info.ArrayLayers {
					fullImageData = true
				}
			}

			if _, ok := colorAs[ai]; ok {
				qei.subpasses[subpass].colorAttachments = append(
					qei.subpasses[subpass].colorAttachments,
					&subpassAttachmentInfo{fullImageData, imgData, attDesc})
			}
			if _, ok := resolveAs[ai]; ok {
				qei.subpasses[subpass].resolveAttachments = append(
					qei.subpasses[subpass].resolveAttachments,
					&subpassAttachmentInfo{fullImageData, imgData, attDesc})
			}
			if _, ok := inputAs[ai]; ok {
				qei.subpasses[subpass].inputAttachments = append(
					qei.subpasses[subpass].inputAttachments,
					&subpassAttachmentInfo{fullImageData, imgData, attDesc})
			}
		}
		if desc.DepthStencilAttachment != nil {
			dsAi := desc.DepthStencilAttachment.Attachment
			if dsAi != vkAttachmentUnused {
				viewObj := fb.ImageAttachments.Get(dsAi)
				read(ctx, bh, vkHandle(viewObj.VulkanHandle))
				imgObj := viewObj.Image
				imgData := vb.getImageData(ctx, bh, imgObj.VulkanHandle)
				attDesc := rp.AttachmentDescriptions.Get(dsAi)
				fullImageData := false
				switch viewObj.Type {
				case VkImageViewType_VK_IMAGE_VIEW_TYPE_2D,
					VkImageViewType_VK_IMAGE_VIEW_TYPE_2D_ARRAY:
					if viewObj.SubresourceRange.BaseArrayLayer == uint32(0) &&
						imgObj.Info.ArrayLayers == viewObj.SubresourceRange.LayerCount &&
						imgObj.Info.ImageType == VkImageType_VK_IMAGE_TYPE_2D &&
						imgObj.Info.Extent.Width == fb.Width &&
						imgObj.Info.Extent.Height == fb.Height &&
						fb.Layers == imgObj.Info.ArrayLayers {
						fullImageData = true
					}
				}
				qei.subpasses[subpass].depthStencilAttachment = &subpassAttachmentInfo{
					fullImageData, imgData, attDesc}
			}
		}
	}
	qei.subpass = &subpassIndex{0}
	qei.startSubpass(ctx, bh)
}

func (qei *queueExecutionInfo) nextSubpass(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behaviour,
	sc submittedCommand, m *vulkanMachine) {
	qei.endSubpass(ctx, ft, bh, sc, m)
	qei.subpass.val += 1
	qei.startSubpass(ctx, bh)
}

func (qei *queueExecutionInfo) endRenderPass(ctx context.Context,
	ft *dependencygraph.Footprint, bh *dependencygraph.Behaviour,
	sc submittedCommand, m *vulkanMachine) {
	qei.endSubpass(ctx, ft, bh, sc, m)
}

type renderpass struct {
	begin label
	end   label
}

type commandBuffer struct {
	begin           label
	end             label
	renderPassBegin label
}

type boundData struct {
	backingData dependencygraph.DefUseAtom
}

func newBoundData(ctx context.Context, bh *dependencygraph.Behaviour,
	res dependencygraph.DefUseAtom) *boundData {
	d := &boundData{backingData: res}
	write(ctx, bh, d)
	return d
}

func (*boundData) DefUseAtom() {}

type descriptor struct {
	ty          VkDescriptorType
	backingData dependencygraph.DefUseAtom
	sampler     vkHandle
}

func (*descriptor) DefUseAtom() {}

type descriptorSet struct {
	descriptors      api.SubCmdIdxTrie
	descriptorCounts map[uint64]uint64 // binding -> descriptor count of that binding
}

func newDescriptorSet() *descriptorSet {
	return &descriptorSet{
		descriptors:      *api.NewSubCmdIdxTrie(),
		descriptorCounts: map[uint64]uint64{},
	}
}

func (ds *descriptorSet) reserveDescriptor(bi, di uint64) {
	ds.descriptors.SetValue([]uint64{bi, di}, &descriptor{})
	if _, ok := ds.descriptorCounts[bi]; !ok {
		ds.descriptorCounts[bi] = uint64(0)
	}
	ds.descriptorCounts[bi] += 1
}

func (ds *descriptorSet) getDescriptor(ctx context.Context,
	bh *dependencygraph.Behaviour, bi, di uint64) *descriptor {
	if v, ok := ds.descriptors.GetValue([]uint64{bi, di}); ok {
		if d, ok := v.(*descriptor); ok {
			read(ctx, bh, d)
			return d
		} else {
			log.E(ctx, "Not *descriptor type in descriptorSet: %v, with "+
				"binding: %v, array index: %v", *ds, bi, di)
			return nil
		}
	} else {
		log.E(ctx, "Read target descriptor does not exists in "+
			"descriptorSet: %v, with binding: %v, array index: %v", *ds, bi, di)
		return nil
	}
	return nil
}

func (ds *descriptorSet) setDescriptor(ctx context.Context,
	bh *dependencygraph.Behaviour, bi, di uint64, ty VkDescriptorType,
	data dependencygraph.DefUseAtom, sampler vkHandle) {
	if v, ok := ds.descriptors.GetValue([]uint64{bi, di}); ok {
		if d, ok := v.(*descriptor); ok {
			write(ctx, bh, d)
			d.backingData = data
			d.sampler = sampler
			d.ty = ty
		} else {
			log.E(ctx, "Not *descriptor type in descriptorSet: %v, with "+
				"binding: %v, array index: %v", *ds, bi, di)
		}
	} else {
		log.E(ctx, "Write target descriptor does not exist in "+
			"descriptorSet: %v, with binding: %v, array index: %v", *ds, bi, di)
	}
}

func (ds *descriptorSet) useDescriptors(ctx context.Context,
	bh *dependencygraph.Behaviour) []dependencygraph.DefUseAtom {
	modified := []dependencygraph.DefUseAtom{}
	for binding, count := range ds.descriptorCounts {
		for di := uint64(0); di < count; di++ {
			d := ds.getDescriptor(ctx, bh, binding, di)
			read(ctx, bh, d.sampler)
			switch d.ty {
			case VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER,
				VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
				modify(ctx, bh, d.backingData)
				modified = append(modified, d.backingData)
			default:
				read(ctx, bh, d.backingData)
			}
		}
	}
	return modified
}

func (ds *descriptorSet) writeDescriptors(ctx context.Context,
	cmd api.Cmd, s *api.State, vb *VulkanFootprintBuilder,
	bh *dependencygraph.Behaviour,
	write VkWriteDescriptorSet) {
	l := s.MemoryLayout
	dstElm := uint64(write.DstArrayElement)
	count := uint64(write.DescriptorCount)
	dstBinding := uint64(write.DstBinding)
	updateDstForOverflow := func() {
		if dstElm >= ds.descriptorCounts[dstBinding] {
			dstBinding += 1
			dstElm = uint64(0)
		}
	}
	switch write.DescriptorType {
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLED_IMAGE,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_IMAGE,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_INPUT_ATTACHMENT:
		imageInfos := write.PImageInfo.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			updateDstForOverflow()
			imageInfo := imageInfos.Index(i, l).Read(ctx, cmd, s, nil)
			backingData := dependencygraph.DefUseAtom(vkNullHandle)
			sampler := vkNullHandle
			if write.DescriptorType != VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER &&
				read(ctx, bh, vkHandle(imageInfo.ImageView)) {
				vkView := imageInfo.ImageView
				vkImg := GetState(s).ImageViews.Get(vkView).Image.VulkanHandle
				backingData = vb.getImageData(ctx, bh, vkImg)
			}
			if (write.DescriptorType == VkDescriptorType_VK_DESCRIPTOR_TYPE_SAMPLER ||
				write.DescriptorType == VkDescriptorType_VK_DESCRIPTOR_TYPE_COMBINED_IMAGE_SAMPLER) &&
				read(ctx, bh, vkHandle(imageInfo.Sampler)) {
				sampler = vkHandle(imageInfo.Sampler)
			}
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType,
				backingData, sampler)
			dstElm += 1
		}
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_BUFFER_DYNAMIC,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_BUFFER_DYNAMIC:
		bufferInfos := write.PBufferInfo.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			updateDstForOverflow()
			bufferInfo := bufferInfos.Index(i, l).Read(ctx, cmd, s, nil)
			vkBuf := bufferInfo.Buffer
			bufData, _ := vb.getBufferData(ctx, bh, vkBuf).(memorySpan)
			dData := getSubBufferData(bufData, uint64(bufferInfo.Offset),
				uint64(bufferInfo.Range))
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType, dData,
				vkNullHandle)
			dstElm += 1
		}
	case VkDescriptorType_VK_DESCRIPTOR_TYPE_UNIFORM_TEXEL_BUFFER,
		VkDescriptorType_VK_DESCRIPTOR_TYPE_STORAGE_TEXEL_BUFFER:
		bufferViews := write.PTexelBufferView.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			updateDstForOverflow()
			vkBufView := bufferViews.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkBufView))
			bufView := GetState(s).BufferViews.Get(vkBufView)
			vkBuf := GetState(s).BufferViews.Get(vkBufView).Buffer.VulkanHandle
			bufData, _ := vb.getBufferData(ctx, bh, vkBuf).(memorySpan)
			dData := getSubBufferData(bufData, uint64(bufView.Offset),
				uint64(bufView.Range))
			ds.setDescriptor(ctx, bh, dstBinding, dstElm, write.DescriptorType, dData,
				vkNullHandle)
			dstElm += 1
		}
	}
}

func (dstDs *descriptorSet) copyDescriptors(ctx context.Context,
	cmd api.Cmd, s *api.State, bh *dependencygraph.Behaviour,
	srcDs *descriptorSet, copy VkCopyDescriptorSet) {
	dstElm := uint64(copy.DstArrayElement)
	srcElm := uint64(copy.SrcArrayElement)
	dstBinding := uint64(copy.DstBinding)
	srcBinding := uint64(copy.SrcBinding)
	updateDstAndSrcForOverflow := func() {
		if dstElm >= dstDs.descriptorCounts[dstBinding] {
			dstBinding += 1
			dstElm = uint64(0)
		}
		if srcElm >= srcDs.descriptorCounts[srcBinding] {
			srcBinding += 1
			srcElm = uint64(0)
		}
	}
	for i := uint64(0); i < uint64(copy.DescriptorCount); i++ {
		updateDstAndSrcForOverflow()
		srcD := srcDs.getDescriptor(ctx, bh, srcBinding, srcElm)
		dstDs.setDescriptor(ctx, bh, dstBinding, dstElm, srcD.ty,
			srcD.backingData, srcD.sampler)
		srcElm += 1
		dstElm += 1
	}
}

type boundDescriptorSet struct {
	descriptorSet *descriptorSet
}

func newBoundDescriptor(ctx context.Context, bh *dependencygraph.Behaviour,
	ds *descriptorSet) *boundDescriptorSet {
	bds := &boundDescriptorSet{descriptorSet: ds}
	write(ctx, bh, bds)
	return bds
}

func (*boundDescriptorSet) DefUseAtom() {}

type VulkanFootprintBuilder struct {
	machine *vulkanMachine

	// commands
	commands map[VkCommandBuffer][]*commandBufferCommand

	// coherent memory mapping
	mappedCoherentMemories map[VkDeviceMemory]*DeviceMemoryObject

	// Vulkan handle states
	semaphoreSignals map[VkSemaphore]label
	fences           map[VkFence]*fence
	events           map[VkEvent]*event
	querypools       map[VkQueryPool]*queryPool
	commandBuffers   map[VkCommandBuffer]*commandBuffer
	images           map[VkImage]*boundData
	buffers          map[VkBuffer]*boundData
	descriptorSets   map[VkDescriptorSet]*descriptorSet

	// execution info
	executionInfos map[VkQueue]*queueExecutionInfo
	submitInfos    map[api.CmdID] /*ID of VkQueueSubmit*/ *queueSubmitInfo
	submitIDs      map[*VkQueueSubmit]api.CmdID

	// presentation info
	swapchainImageAcquired  map[VkSwapchainKHR][]label
	swapchainImagePresented map[VkSwapchainKHR][]label
}

func (vb *VulkanFootprintBuilder) getBoundData(ctx context.Context,
	bh *dependencygraph.Behaviour, bound *boundData) dependencygraph.DefUseAtom {
	read(ctx, bh, bound)
	return bound.backingData
}

func (vb *VulkanFootprintBuilder) getImageData(ctx context.Context,
	bh *dependencygraph.Behaviour, vkImg VkImage) dependencygraph.DefUseAtom {
	read(ctx, bh, vkHandle(vkImg))
	return vb.getBoundData(ctx, bh, vb.images[vkImg])
}

func (vb *VulkanFootprintBuilder) getBufferData(ctx context.Context,
	bh *dependencygraph.Behaviour, vkBuf VkBuffer) dependencygraph.DefUseAtom {
	read(ctx, bh, vkHandle(vkBuf))
	return vb.getBoundData(ctx, bh, vb.buffers[vkBuf])
}

func (vb *VulkanFootprintBuilder) newCommand(ctx context.Context,
	bh *dependencygraph.Behaviour, vkCb VkCommandBuffer) *commandBufferCommand {
	cbc := &commandBufferCommand{}
	read(ctx, bh, vkHandle(vkCb))
	read(ctx, bh, vb.commandBuffers[vkCb].begin)
	write(ctx, bh, cbc)
	vb.commands[vkCb] = append(vb.commands[vkCb], cbc)
	return cbc
}

func newVulkanFootprintBuilder() *VulkanFootprintBuilder {
	return &VulkanFootprintBuilder{
		machine:                 newVulkanMachine(),
		commands:                map[VkCommandBuffer][]*commandBufferCommand{},
		mappedCoherentMemories:  map[VkDeviceMemory]*DeviceMemoryObject{},
		semaphoreSignals:        map[VkSemaphore]label{},
		fences:                  map[VkFence]*fence{},
		events:                  map[VkEvent]*event{},
		querypools:              map[VkQueryPool]*queryPool{},
		commandBuffers:          map[VkCommandBuffer]*commandBuffer{},
		images:                  map[VkImage]*boundData{},
		buffers:                 map[VkBuffer]*boundData{},
		descriptorSets:          map[VkDescriptorSet]*descriptorSet{},
		executionInfos:          map[VkQueue]*queueExecutionInfo{},
		submitInfos:             map[api.CmdID]*queueSubmitInfo{},
		submitIDs:               map[*VkQueueSubmit]api.CmdID{},
		swapchainImageAcquired:  map[VkSwapchainKHR][]label{},
		swapchainImagePresented: map[VkSwapchainKHR][]label{},
	}
}

func (vb *VulkanFootprintBuilder) rollOutExecuted(ctx context.Context,
	ft *dependencygraph.Footprint,
	executedCommands []api.SubCmdIdx) {
	for _, executedFCI := range executedCommands {
		submitID := executedFCI[0]
		submittedCmd := vb.submitInfos[api.CmdID(submitID)].pendingCommands[0]
		if executedFCI.Equals(submittedCmd.id) {
			execInfo := vb.executionInfos[vb.submitInfos[api.CmdID(submitID)].queue]
			execInfo.currentSubmitInfo = vb.submitInfos[api.CmdID(submitID)]
			execInfo.updateCurrentCommand(ctx, executedFCI)
			submittedCmd.runCommand(ctx, ft, vb.machine, execInfo)
		} else {
			log.E(ctx, "Execution order differs from submission order. "+
				"Index of executed command: %v, Index of submitted command: %v",
				executedFCI, submittedCmd.id)
			return
		}
		// Remove the front submitted command.
		vb.submitInfos[api.CmdID(submitID)].pendingCommands =
			vb.submitInfos[api.CmdID(submitID)].pendingCommands[1:]
		// After the last command of the submit, we need to add a behaviour for
		// semaphore and fence signaling.
		if len(vb.submitInfos[api.CmdID(submitID)].pendingCommands) == 0 {
			bh := dependencygraph.NewBehaviour(api.SubCmdIdx{
				executedFCI[0]}, vb.machine)
			// add writes to the semaphores and fences
			submitinfo := vb.submitInfos[api.CmdID(submitID)]
			read(ctx, bh, submitinfo.executionBegin)
			write(ctx, bh, submitinfo.executionEnd)
			for _, sp := range submitinfo.signalSemaphores {
				if read(ctx, bh, vkHandle(sp)) {
					write(ctx, bh, vb.semaphoreSignals[sp])
				}
			}
			if read(ctx, bh, vkHandle(submitinfo.signalFence)) {
				write(ctx, bh, vb.fences[submitinfo.signalFence].signal)
			}
			ft.AddBehaviour(ctx, bh)
		}
	}
}

func (vb *VulkanFootprintBuilder) recordReadsWritesModifies(
	ctx context.Context, ft *dependencygraph.Footprint, bh *dependencygraph.Behaviour,
	vkCb VkCommandBuffer, reads []dependencygraph.DefUseAtom,
	writes []dependencygraph.DefUseAtom, modifies []dependencygraph.DefUseAtom) {
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand, execInfo *queueExecutionInfo) {
		cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
		for _, d := range reads {
			read(ctx, cbh, d)
		}
		for _, d := range writes {
			write(ctx, cbh, d)
		}
		for _, d := range modifies {
			modify(ctx, cbh, d)
		}
		ft.AddBehaviour(ctx, cbh)
	}
}

func (vb *VulkanFootprintBuilder) recordModifingDynamicStates(
	ctx context.Context, ft *dependencygraph.Footprint, bh *dependencygraph.Behaviour,
	vkCb VkCommandBuffer) {
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand, execInfo *queueExecutionInfo) {
		cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
		modify(ctx, cbh, execInfo.currentCmdBufState.dynamicState)
		ft.AddBehaviour(ctx, cbh)
	}
}

func (vb *VulkanFootprintBuilder) useBoundDescriptorSets(ctx context.Context,
	bh *dependencygraph.Behaviour,
	cmdBufState *commandBufferExecutionState) []dependencygraph.DefUseAtom {
	modified := []dependencygraph.DefUseAtom{}
	for _, bds := range cmdBufState.descriptorSets {
		read(ctx, bh, bds)
		ds := bds.descriptorSet
		modified = append(modified, ds.useDescriptors(ctx, bh)...)
	}
	return modified
}

func (vb *VulkanFootprintBuilder) draw(ctx context.Context,
	bh *dependencygraph.Behaviour, execInfo *queueExecutionInfo) {
	read(ctx, bh, execInfo.subpass)
	read(ctx, bh, execInfo.currentCmdBufState.pipeline)
	read(ctx, bh, execInfo.currentCmdBufState.dynamicState)
	subpassI := execInfo.subpass.val
	for _, b := range execInfo.currentCmdBufState.vertexBuffers {
		read(ctx, bh, vb.getBoundData(ctx, bh, b))
	}
	modifiedDs := vb.useBoundDescriptorSets(ctx, bh, execInfo.currentCmdBufState)
	execInfo.subpasses[execInfo.subpass.val].modifiedDescriptorData = append(
		execInfo.subpasses[execInfo.subpass.val].modifiedDescriptorData,
		modifiedDs...)
	if execInfo.currentCmdBufState.indexBuffer != nil {
		read(ctx, bh, vb.getBoundData(ctx, bh, execInfo.currentCmdBufState.indexBuffer))
	}
	for _, input := range execInfo.subpasses[subpassI].inputAttachments {
		read(ctx, bh, input.data)
	}
	for _, color := range execInfo.subpasses[subpassI].colorAttachments {
		modify(ctx, bh, color.data)
	}
	if execInfo.subpasses[subpassI].depthStencilAttachment != nil {
		dsAtt := execInfo.subpasses[subpassI].depthStencilAttachment
		modify(ctx, bh, dsAtt.data)
	}
}

func (vb *VulkanFootprintBuilder) readBoundIndexBuffer(ctx context.Context,
	bh *dependencygraph.Behaviour, execInfo *queueExecutionInfo, cmd api.Cmd) {
	boundIndexBufferData, _ := execInfo.currentCmdBufState.indexBuffer.backingData.(memorySpan)
	indexSize := uint64(2)
	switch execInfo.currentCmdBufState.indexType {
	case VkIndexType_VK_INDEX_TYPE_UINT16:
		indexSize = uint64(2)
	case VkIndexType_VK_INDEX_TYPE_UINT32:
		indexSize = uint64(4)
	}

	offset := boundIndexBufferData.span.Start
	size := boundIndexBufferData.span.End - offset
	switch cmd := cmd.(type) {
	case *VkCmdDrawIndexed:
		size = uint64(cmd.IndexCount) * indexSize
		offset = offset + uint64(cmd.FirstIndex)*indexSize
	case *RecreateCmdDrawIndexed:
		size = uint64(cmd.IndexCount) * indexSize
		offset = offset + uint64(cmd.FirstIndex)*indexSize
	case *VkCmdDrawIndexedIndirect:
	case *RecreateCmdDrawIndexedIndirect:
	}
	dataToRead := getSubBufferData(boundIndexBufferData, offset, size)
	read(ctx, bh, dataToRead)
}

func (vb *VulkanFootprintBuilder) recordBarriers(ctx context.Context,
	s *api.State, ft *dependencygraph.Footprint, cmd api.Cmd,
	bh *dependencygraph.Behaviour, vkCb VkCommandBuffer, memoryBarrierCount uint32,
	bufferBarrierCount uint32, pBufferBarriers VkBufferMemoryBarrierᶜᵖ,
	imageBarrierCount uint32, pImageBarriers VkImageMemoryBarrierᶜᵖ,
	attachedReads []dependencygraph.DefUseAtom) {
	l := s.MemoryLayout
	touchedData := []dependencygraph.DefUseAtom{}
	if memoryBarrierCount > 0 {
		// touch all buffer and image backing data
		for _, d := range vb.images {
			touchedData = append(touchedData, d)
		}
		for _, d := range vb.buffers {
			touchedData = append(touchedData, d)
		}
	} else {
		bufBarriers := pBufferBarriers.Slice(0, uint64(bufferBarrierCount), l)
		for i := uint64(0); i < uint64(bufferBarrierCount); i++ {
			barrier := bufBarriers.Index(i, l).Read(ctx, cmd, s, nil)
			bufData, _ := vb.getBufferData(ctx, bh, barrier.Buffer).(memorySpan)
			touchedData = append(touchedData, getSubBufferData(
				bufData, uint64(barrier.Offset), uint64(barrier.Size)))
		}
		imgBarriers := pImageBarriers.Slice(0, uint64(imageBarrierCount), l)
		for i := uint64(0); i < uint64(imageBarrierCount); i++ {
			barrier := imgBarriers.Index(i, l).Read(ctx, cmd, s, nil)
			touchedData = append(touchedData, vb.getImageData(ctx, bh, barrier.Image))
		}
	}
	cbc := vb.newCommand(ctx, bh, vkCb)
	cbc.behave = func(sc submittedCommand,
		execInfo *queueExecutionInfo) {
		for _, d := range touchedData {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			readMultiple(ctx, cbh, attachedReads)
			modify(ctx, cbh, d)
			ft.AddBehaviour(ctx, cbh)
		}
	}
}

func (vb *VulkanFootprintBuilder) BuildFootprint(ctx context.Context,
	s *api.State, ft *dependencygraph.Footprint, id api.CmdID, cmd api.Cmd) {

	l := s.MemoryLayout

	// Records the mapping from queue submit to command ID, so the
	// HandleSubcommand callback can use it.
	if qs, isSubmit := cmd.(*VkQueueSubmit); isSubmit {
		vb.submitIDs[qs] = id
	}
	// Register callback function to record only the truly executed
	// commandbuffer commands.
	executedCommands := []api.SubCmdIdx{}
	GetState(s).HandleSubcommand = func(a interface{}) {
		queueSubmit, isQs := (*GetState(s).CurrentSubmission).(*VkQueueSubmit)
		if !isQs {
			log.E(ctx, "CurrentSubmission command in State is not a VkQueueSubmit")
		}
		fci := api.SubCmdIdx{uint64(vb.submitIDs[queueSubmit])}
		fci = append(fci, GetState(s).SubCmdIdx...)
		executedCommands = append(executedCommands, fci)
	}

	// Mutate
	if err := cmd.Mutate(ctx, s, nil); err != nil {
		log.E(ctx, "Command %v %v: %v", id, cmd, err)
		return
	}

	bh := dependencygraph.NewBehaviour(
		api.SubCmdIdx{uint64(id)}, vb.machine)

	// The main switch
	switch cmd := cmd.(type) {
	// device memory
	case *VkAllocateMemory:
		vkMem := cmd.PMemory.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkMem))
	case *RecreateDeviceMemory:
		vkMem := cmd.PMemory.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkMem))
	case *VkFreeMemory:
		vkMem := cmd.Memory
		read(ctx, bh, vkHandle(vkMem))
		bh.Alive = true
	case *VkMapMemory:
		modify(ctx, bh, vkHandle(cmd.Memory))
		memObj := GetState(s).DeviceMemories.Get(cmd.Memory)
		isCoherent, _ := subIsMemoryCoherent(ctx, cmd, nil, s, GetState(s),
			cmd.thread, nil, memObj)
		if isCoherent {
			vb.mappedCoherentMemories[cmd.Memory] = memObj
		}
		bh.Alive = true
	case *VkUnmapMemory:
		modify(ctx, bh, vkHandle(cmd.Memory))
		vb.writeCoherentMemoryData(ctx, cmd, bh)
		if _, mappedCoherent := vb.mappedCoherentMemories[cmd.Memory]; mappedCoherent {
			delete(vb.mappedCoherentMemories, cmd.Memory)
		}
	case *VkFlushMappedMemoryRanges:
		count := uint64(cmd.MemoryRangeCount)
		rngs := cmd.PMemoryRanges.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			rng := rngs.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(rng.Memory))
			offset := uint64(rng.Offset)
			size := uint64(rng.Size)
			ms := memorySpan{
				span:   interval.U64Span{Start: offset, End: offset + size},
				memory: rng.Memory,
			}
			write(ctx, bh, ms)
		}
	case *VkInvalidateMappedMemoryRanges:
		count := uint64(cmd.MemoryRangeCount)
		rngs := cmd.PMemoryRanges.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			rng := rngs.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(rng.Memory))
			offset := uint64(rng.Offset)
			size := uint64(rng.Size)
			ms := memorySpan{
				span:   interval.U64Span{Start: offset, End: offset + size},
				memory: rng.Memory,
			}
			read(ctx, bh, ms)
		}

	// image
	case *VkCreateImage:
		vkImg := cmd.PImage.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkImg))
	case *RecreateImage:
		vkImg := cmd.PImage.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkImg))
	case *VkDestroyImage:
		vkImg := cmd.Image
		read(ctx, bh, vkHandle(vkImg))
		bh.Alive = true

	case *VkBindImageMemory:
		read(ctx, bh, vkHandle(cmd.Image))
		read(ctx, bh, vkHandle(cmd.Memory))
		offset := uint64(cmd.MemoryOffset)
		inferred_size, err := subInferImageSize(ctx, cmd, nil, s, nil, cmd.thread,
			nil, GetState(s).Images.Get(cmd.Image))
		if err != nil {
			log.E(ctx, "Cannot get inferred size of image: %v", cmd.Image)
			log.E(ctx, "Command %v %v: %v", id, cmd, err)
			bh.Aborted = true
		}
		size := uint64(inferred_size)
		vb.images[cmd.Image] = newBoundData(ctx, bh, memorySpan{
			span:   interval.U64Span{Start: offset, End: offset + size},
			memory: cmd.Memory,
		})
	case *RecreateBindImageMemory:
		read(ctx, bh, vkHandle(cmd.Image))
		read(ctx, bh, vkHandle(cmd.Memory))
		offset := uint64(cmd.Offset)
		inferred_size, err := subInferImageSize(ctx, cmd, nil, s, nil, cmd.thread,
			nil, GetState(s).Images.Get(cmd.Image))
		if err != nil {
			log.E(ctx, "Cannot get inferred size of image: %v", cmd.Image)
			log.E(ctx, "Command %v %v: %v", id, cmd, err)
			bh.Aborted = true
		}
		size := uint64(inferred_size)
		vb.images[cmd.Image] = newBoundData(ctx, bh, memorySpan{
			span:   interval.U64Span{Start: offset, End: offset + size},
			memory: cmd.Memory,
		})

	case *RecreateImageData:
		write(ctx, bh, vb.getImageData(ctx, bh, cmd.Image))

	case *VkCreateImageView:
		write(ctx, bh, vkHandle(cmd.PView.Read(ctx, cmd, s, nil)))
	case *RecreateImageView:
		write(ctx, bh, vkHandle(cmd.PImageView.Read(ctx, cmd, s, nil)))
	case *VkDestroyImageView:
		read(ctx, bh, vkHandle(cmd.ImageView))
		bh.Alive = true

	// buffer
	case *VkCreateBuffer:
		vkBuf := cmd.PBuffer.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkBuf))
	case *RecreateBuffer:
		vkBuf := cmd.PBuffer.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkBuf))
	case *VkDestroyBuffer:
		vkBuf := cmd.Buffer
		read(ctx, bh, vkHandle(vkBuf))
		bh.Alive = true

	case *VkBindBufferMemory:
		read(ctx, bh, vkHandle(cmd.Buffer))
		read(ctx, bh, vkHandle(cmd.Memory))
		offset := uint64(cmd.MemoryOffset)
		size := uint64(GetState(s).Buffers.Get(cmd.Buffer).Info.Size)
		vb.buffers[cmd.Buffer] = newBoundData(ctx, bh,
			memorySpan{
				span:   interval.U64Span{Start: offset, End: offset + size},
				memory: cmd.Memory,
			})
	case *RecreateBindBufferMemory:
		read(ctx, bh, vkHandle(cmd.Buffer))
		read(ctx, bh, vkHandle(cmd.Memory))
		offset := uint64(cmd.Offset)
		size := uint64(GetState(s).Buffers.Get(cmd.Buffer).Info.Size)
		vb.buffers[cmd.Buffer] = newBoundData(ctx, bh,
			memorySpan{
				span:   interval.U64Span{Start: offset, End: offset + size},
				memory: cmd.Memory,
			})

	case *RecreateBufferData:
		write(ctx, bh, vb.getBufferData(ctx, bh, cmd.Buffer))

	case *VkCreateBufferView:
		write(ctx, bh, vkHandle(cmd.PView.Read(ctx, cmd, s, nil)))
	case *RecreateBufferView:
		write(ctx, bh, vkHandle(cmd.PBufferView.Read(ctx, cmd, s, nil)))
	case *VkDestroyBufferView:
		read(ctx, bh, vkHandle(cmd.BufferView))
		bh.Alive = true

	// swapchain
	case *VkCreateSwapchainKHR:
		vkSw := cmd.PSwapchain.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkSw))
	case *RecreateSwapchain:
		vkSw := cmd.PSwapchain.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkSw))
		imageCount := uint64(cmd.PCreateInfo.Read(ctx, cmd, s, nil).MinImageCount)
		vkImgs := cmd.PSwapchainImages.Slice(0, imageCount, l)
		for i := uint64(0); i < imageCount; i++ {
			vkImg := vkImgs.Index(i, l).Read(ctx, cmd, s, nil)
			write(ctx, bh, vkHandle(vkImg))
			vb.images[vkImg] = newBoundData(ctx, bh, newLabel())
			vb.swapchainImageAcquired[vkSw] = append(
				vb.swapchainImageAcquired[vkSw], newLabel())
			vb.swapchainImagePresented[vkSw] = append(
				vb.swapchainImagePresented[vkSw], newLabel())
		}
	case *VkGetSwapchainImagesKHR:
		read(ctx, bh, vkHandle(cmd.Swapchain))
		if (cmd.PSwapchainImages == VkImageᵖ{}) {
			modify(ctx, bh, vkHandle(cmd.Swapchain))
		} else {
			count := uint64(cmd.PSwapchainImageCount.Read(ctx, cmd, s, nil))
			vkImgs := cmd.PSwapchainImages.Slice(0, count, l)
			for i := uint64(0); i < count; i++ {
				vkImg := vkImgs.Index(i, l).Read(ctx, cmd, s, nil)
				write(ctx, bh, vkHandle(vkImg))
				vb.images[vkImg] = newBoundData(ctx, bh, newLabel())
				vb.swapchainImageAcquired[cmd.Swapchain] = append(
					vb.swapchainImageAcquired[cmd.Swapchain], newLabel())
				vb.swapchainImagePresented[cmd.Swapchain] = append(
					vb.swapchainImagePresented[cmd.Swapchain], newLabel())
			}
		}

	// presentation engine
	case *VkAcquireNextImageKHR:
		if read(ctx, bh, vkHandle(cmd.Semaphore)) {
			write(ctx, bh, vb.semaphoreSignals[cmd.Semaphore])
		}
		if read(ctx, bh, vkHandle(cmd.Fence)) {
			write(ctx, bh, vb.fences[cmd.Fence].signal)
		}
		read(ctx, bh, vkHandle(cmd.Swapchain))
		// The value of this imgId should have been written by the driver.
		imgId := cmd.PImageIndex.Read(ctx, cmd, s, nil)
		vkImg := GetState(s).Swapchains.Get(cmd.Swapchain).SwapchainImages.Get(imgId).VulkanHandle
		if read(ctx, bh, vkHandle(vkImg)) {
			write(ctx, bh, vb.getImageData(ctx, bh, vkImg))
		}
		write(ctx, bh, vb.swapchainImageAcquired[cmd.Swapchain][imgId])
		read(ctx, bh, vb.swapchainImagePresented[cmd.Swapchain][imgId])

	case *VkQueuePresentKHR:
		read(ctx, bh, vkHandle(cmd.Queue))
		info := cmd.PPresentInfo.Read(ctx, cmd, s, nil)
		spCount := uint64(info.WaitSemaphoreCount)
		vkSps := info.PWaitSemaphores.Slice(0, spCount, l)
		for i := uint64(0); i < spCount; i++ {
			vkSp := vkSps.Index(i, l).Read(ctx, cmd, s, nil)
			if read(ctx, bh, vkHandle(vkSp)) {
				read(ctx, bh, vb.semaphoreSignals[vkSp])
			}
		}
		swCount := uint64(info.SwapchainCount)
		vkSws := info.PSwapchains.Slice(0, swCount, l)
		imgIds := info.PImageIndices.Slice(0, swCount, l)
		for i := uint64(0); i < swCount; i++ {
			vkSw := vkSws.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkSw))
			imgId := imgIds.Index(i, l).Read(ctx, cmd, s, nil)
			{
				nbh := dependencygraph.NewBehaviour(
					api.SubCmdIdx{uint64(id)}, vb.machine)
				for i := uint64(0); i < spCount; i++ {
					vkSp := vkSps.Index(i, l).Read(ctx, cmd, s, nil)
					if read(ctx, nbh, vkHandle(vkSp)) {
						read(ctx, nbh, vb.semaphoreSignals[vkSp])
					}
				}
				read(ctx, nbh, vkHandle(cmd.Queue))
				read(ctx, nbh, vb.swapchainImageAcquired[vkSw][imgId])
				write(ctx, nbh, vb.swapchainImagePresented[vkSw][imgId])
				nbh.Alive = true
				ft.AddBehaviour(ctx, nbh)
			}
			vkImg := GetState(s).Swapchains.Get(vkSw).SwapchainImages.Get(imgId).VulkanHandle
			read(ctx, bh, vb.getImageData(ctx, bh, vkImg))
		}

	// sampler
	case *VkCreateSampler:
		write(ctx, bh, vkHandle(cmd.PSampler.Read(ctx, cmd, s, nil)))
	case *RecreateSampler:
		write(ctx, bh, vkHandle(cmd.PSampler.Read(ctx, cmd, s, nil)))
	case *VkDestroySampler:
		read(ctx, bh, vkHandle(cmd.Sampler))
		bh.Alive = true

	// query pool
	case *VkCreateQueryPool:
		vkQp := cmd.PQueryPool.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkQp))
		vb.querypools[vkQp] = &queryPool{}
		count := uint64(cmd.PCreateInfo.Read(ctx, cmd, s, nil).QueryCount)
		for i := uint64(0); i < count; i++ {
			vb.querypools[vkQp].queries = append(vb.querypools[vkQp].queries, newQuery())
		}
	case *RecreateQueryPool:
		vkQp := cmd.PPool.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkQp))
		vb.querypools[vkQp] = &queryPool{}
		count := uint64(cmd.PCreateInfo.Read(ctx, cmd, s, nil).QueryCount)
		for i := uint64(0); i < count; i++ {
			vb.querypools[vkQp].queries = append(vb.querypools[vkQp].queries, newQuery())
		}
	case *VkDestroyQueryPool:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		delete(vb.querypools, cmd.QueryPool)
		bh.Alive = true
	case *VkGetQueryPoolResults:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		count := uint64(cmd.QueryCount)
		first := uint64(cmd.FirstQuery)
		for i := uint64(0); i < count; i++ {
			read(ctx, bh, vb.querypools[cmd.QueryPool].queries[i+first].result)
		}

	// descriptor set
	case *VkCreateDescriptorSetLayout:
		write(ctx, bh, vkHandle(cmd.PSetLayout.Read(ctx, cmd, s, nil)))
	case *RecreateDescriptorSetLayout:
		write(ctx, bh, vkHandle(cmd.PSetLayout.Read(ctx, cmd, s, nil)))
	case *VkDestroyDescriptorSetLayout:
		read(ctx, bh, vkHandle(cmd.DescriptorSetLayout))
		bh.Alive = true
	case *VkAllocateDescriptorSets:
		info := cmd.PAllocateInfo.Read(ctx, cmd, s, nil)
		setCount := uint64(info.DescriptorSetCount)
		vkLayouts := info.PSetLayouts.Slice(0, setCount, l)
		vkSets := cmd.PDescriptorSets.Slice(0, setCount, l)
		for i := uint64(0); i < setCount; i++ {
			vkLayout := vkLayouts.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkLayout))
			layoutObj := GetState(s).DescriptorSetLayouts.Get(vkLayout)
			vkSet := vkSets.Index(i, l).Read(ctx, cmd, s, nil)
			write(ctx, bh, vkHandle(vkSet))
			vb.descriptorSets[vkSet] = newDescriptorSet()
			for bi, bindingInfo := range layoutObj.Bindings {
				for di := uint32(0); di < bindingInfo.Count; di++ {
					vb.descriptorSets[vkSet].reserveDescriptor(uint64(bi), uint64(di))
				}
			}
		}
	case *VkUpdateDescriptorSets:
		writeCount := cmd.DescriptorWriteCount
		if writeCount > 0 {
			writes := cmd.PDescriptorWrites.Slice(0, uint64(writeCount), l)
			for i := uint64(0); i < uint64(writeCount); i++ {
				write := writes.Index(i, l).Read(ctx, cmd, s, nil)
				read(ctx, bh, vkHandle(write.DstSet))
				ds := vb.descriptorSets[write.DstSet]
				ds.writeDescriptors(ctx, cmd, s, vb, bh, write)
			}
		}
		copyCount := cmd.DescriptorCopyCount
		if copyCount > 0 {
			copies := cmd.PDescriptorCopies.Slice(0, uint64(copyCount), l)
			for i := uint64(0); i < uint64(copyCount); i++ {
				copy := copies.Index(i, l).Read(ctx, cmd, s, nil)
				read(ctx, bh, vkHandle(copy.SrcSet))
				read(ctx, bh, vkHandle(copy.DstSet))
				vb.descriptorSets[copy.DstSet].copyDescriptors(ctx, cmd, s, bh,
					vb.descriptorSets[copy.SrcSet], copy)
			}
		}
	case *RecreateDescriptorSet:
		info := cmd.PAllocateInfo.Read(ctx, cmd, s, nil)
		vkLayout := info.PSetLayouts.Slice(0, 1, l).Index(0, l).Read(ctx, cmd, s, nil)
		vkSet := cmd.PDescriptorSet.Read(ctx, cmd, s, nil)
		read(ctx, bh, vkHandle(vkLayout))
		write(ctx, bh, vkHandle(vkSet))
		layoutObj := GetState(s).DescriptorSetLayouts.Get(vkLayout)
		vb.descriptorSets[vkSet] = newDescriptorSet()
		for bi, bindingInfo := range layoutObj.Bindings {
			for di := uint32(0); di < bindingInfo.Count; di++ {
				vb.descriptorSets[vkSet].reserveDescriptor(uint64(bi), uint64(di))
			}
		}
		writeCount := cmd.DescriptorWriteCount
		if writeCount > 0 {
			writes := cmd.PDescriptorWrites.Slice(0, uint64(writeCount), l)
			for i := uint64(0); i < uint64(writeCount); i++ {
				write := writes.Index(i, l).Read(ctx, cmd, s, nil)
				vb.descriptorSets[vkSet].writeDescriptors(ctx, cmd, s, vb, bh, write)
			}
		}

	case *VkFreeDescriptorSets:
		count := uint64(cmd.DescriptorSetCount)
		vkSets := cmd.PDescriptorSets.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			vkSet := vkSets.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkSet))
			delete(vb.descriptorSets, vkSet)
		}
		bh.Alive = true

	// pipelines
	case *VkCreatePipelineLayout:
		info := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(cmd.PPipelineLayout.Read(ctx, cmd, s, nil)))
		setCount := uint64(info.SetLayoutCount)
		setLayouts := info.PSetLayouts.Slice(0, setCount, l)
		for i := uint64(0); i < setCount; i++ {
			setLayout := setLayouts.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(setLayout))
		}
	case *RecreatePipelineLayout:
		info := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(cmd.PPipelineLayout.Read(ctx, cmd, s, nil)))
		setCount := uint64(info.SetLayoutCount)
		setLayouts := info.PSetLayouts.Slice(0, setCount, l)
		for i := uint64(0); i < setCount; i++ {
			setLayout := setLayouts.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(setLayout))
		}
	case *VkDestroyPipelineLayout:
		read(ctx, bh, vkHandle(cmd.PipelineLayout))
		bh.Alive = true
	case *VkCreateGraphicsPipelines:
		read(ctx, bh, vkHandle(cmd.PipelineCache))
		infoCount := uint64(cmd.CreateInfoCount)
		infos := cmd.PCreateInfos.Slice(0, infoCount, l)
		pipelines := cmd.PPipelines.Slice(0, infoCount, l)
		for i := uint64(0); i < infoCount; i++ {
			info := infos.Index(i, l).Read(ctx, cmd, s, nil)
			stageCount := uint64(info.StageCount)
			stages := info.PStages.Slice(0, stageCount, l)
			for j := uint64(0); j < stageCount; j++ {
				stage := stages.Index(j, l).Read(ctx, cmd, s, nil)
				module := stage.Module
				read(ctx, bh, vkHandle(module))
			}
			read(ctx, bh, vkHandle(info.Layout))
			read(ctx, bh, vkHandle(info.RenderPass))
			write(ctx, bh, vkHandle(pipelines.Index(i, l).Read(ctx, cmd, s, nil)))
		}
	case *RecreateGraphicsPipeline:
		read(ctx, bh, vkHandle(cmd.PipelineCache))
		info := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		stageCount := uint64(info.StageCount)
		stages := info.PStages.Slice(0, stageCount, l)
		for i := uint64(0); i < stageCount; i++ {
			stage := stages.Index(i, l).Read(ctx, cmd, s, nil)
			module := stage.Module
			read(ctx, bh, vkHandle(module))
		}
		read(ctx, bh, vkHandle(info.Layout))
		read(ctx, bh, vkHandle(info.RenderPass))
		write(ctx, bh, vkHandle(cmd.PPipeline.Read(ctx, cmd, s, nil)))
	case *VkCreateComputePipelines:
		read(ctx, bh, vkHandle(cmd.PipelineCache))
		infoCount := uint64(cmd.CreateInfoCount)
		infos := cmd.PCreateInfos.Slice(0, infoCount, l)
		pipelines := cmd.PPipelines.Slice(0, infoCount, l)
		for i := uint64(0); i < infoCount; i++ {
			info := infos.Index(i, l).Read(ctx, cmd, s, nil)
			stage := info.Stage
			module := stage.Module
			read(ctx, bh, vkHandle(module))
			read(ctx, bh, vkHandle(info.Layout))
			write(ctx, bh, vkHandle(pipelines.Index(i, l).Read(ctx, cmd, s, nil)))
		}
	case *RecreateComputePipeline:
		read(ctx, bh, vkHandle(cmd.PipelineCache))
		info := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		stage := info.Stage
		module := stage.Module
		read(ctx, bh, vkHandle(module))
		read(ctx, bh, vkHandle(info.Layout))
		write(ctx, bh, vkHandle(cmd.PPipeline.Read(ctx, cmd, s, nil)))
	case *VkDestroyPipeline:
		read(ctx, bh, vkHandle(cmd.Pipeline))
		bh.Alive = true

	case *VkCreatePipelineCache:
		write(ctx, bh, vkHandle(cmd.PPipelineCache.Read(ctx, cmd, s, nil)))
	case *RecreatePipelineCache:
		write(ctx, bh, vkHandle(cmd.PPipelineCache.Read(ctx, cmd, s, nil)))
	case *VkDestroyPipelineCache:
		read(ctx, bh, vkHandle(cmd.PipelineCache))
		bh.Alive = true

	// Shader module
	case *VkCreateShaderModule:
		write(ctx, bh, vkHandle(cmd.PShaderModule.Read(ctx, cmd, s, nil)))
	case *RecreateShaderModule:
		write(ctx, bh, vkHandle(cmd.PShaderModule.Read(ctx, cmd, s, nil)))
	case *VkDestroyShaderModule:
		read(ctx, bh, vkHandle(cmd.ShaderModule))
		bh.Alive = true

	// create/destroy renderpass
	case *VkCreateRenderPass:
		write(ctx, bh, vkHandle(cmd.PRenderPass.Read(ctx, cmd, s, nil)))
	case *RecreateRenderPass:
		write(ctx, bh, vkHandle(cmd.PRenderPass.Read(ctx, cmd, s, nil)))
	case *VkDestroyRenderPass:
		read(ctx, bh, vkHandle(cmd.RenderPass))
		bh.Alive = true

	// create/destroy framebuffer
	case *VkCreateFramebuffer:
		info := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		read(ctx, bh, vkHandle(info.RenderPass))
		attCount := uint64(info.AttachmentCount)
		atts := info.PAttachments.Slice(0, attCount, l)
		for i := uint64(0); i < attCount; i++ {
			att := atts.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(att))
		}
		write(ctx, bh, vkHandle(cmd.PFramebuffer.Read(ctx, cmd, s, nil)))
	case *RecreateFramebuffer:
		info := cmd.PCreateInfo.Read(ctx, cmd, s, nil)
		read(ctx, bh, vkHandle(info.RenderPass))
		attCount := uint64(info.AttachmentCount)
		atts := info.PAttachments.Slice(0, attCount, l)
		for i := uint64(0); i < attCount; i++ {
			att := atts.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(att))
		}
		write(ctx, bh, vkHandle(cmd.PFramebuffer.Read(ctx, cmd, s, nil)))
	case *VkDestroyFramebuffer:
		read(ctx, bh, vkHandle(cmd.Framebuffer))
		bh.Alive = true

	// commandbuffer
	case *VkAllocateCommandBuffers:
		count := uint64(cmd.PAllocateInfo.Read(ctx, cmd, s, nil).CommandBufferCount)
		vkCbs := cmd.PCommandBuffers.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			vkCb := vkCbs.Index(i, l).Read(ctx, cmd, s, nil)
			write(ctx, bh, vkHandle(vkCb))
			vb.commandBuffers[vkCb] = &commandBuffer{begin: newLabel(),
				end: newLabel(), renderPassBegin: newLabel()}
		}

	case *VkResetCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].begin)
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].end)
		vb.commands[cmd.CommandBuffer] = []*commandBufferCommand{}

	case *VkFreeCommandBuffers:
		count := uint64(cmd.CommandBufferCount)
		vkCbs := cmd.PCommandBuffers.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			vkCb := vkCbs.Index(i, l).Read(ctx, cmd, s, nil)
			if read(ctx, bh, vkHandle(vkCb)) {
				write(ctx, bh, vb.commandBuffers[vkCb].begin)
				write(ctx, bh, vb.commandBuffers[vkCb].end)
				delete(vb.commandBuffers, vkCb)
				delete(vb.commands, vkCb)
			}
		}
		bh.Alive = true

	case *VkBeginCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].begin)
		vb.commands[cmd.CommandBuffer] = []*commandBufferCommand{}
	case *RecreateAndBeginCommandBuffer:
		vkCb := cmd.PCommandBuffer.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkCb))
		vb.commandBuffers[vkCb] = &commandBuffer{begin: newLabel(), end: newLabel()}
		write(ctx, bh, vb.commandBuffers[vkCb].begin)
		vb.commands[vkCb] = []*commandBufferCommand{}

	case *VkEndCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer))
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].begin)
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].end)
	case *RecreateEndCommandBuffer:
		read(ctx, bh, vkHandle(cmd.CommandBuffer))
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].begin)
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].end)

	// copy, blit, resolve, clear, fill, update image and buffer
	case *VkCmdCopyImage:
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		// TODO: check dst image coverage correctly
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.DstSubresource, region.DstOffset, region.Extent)
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}
	case *RecreateCmdCopyImage:
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.DstSubresource, region.DstOffset, region.Extent)
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}

	case *VkCmdCopyBuffer:
		srcBufferData, _ := vb.getBufferData(ctx, bh, cmd.SrcBuffer).(memorySpan)
		dstBufferData, _ := vb.getBufferData(ctx, bh, cmd.DstBuffer).(memorySpan)
		src := []dependencygraph.DefUseAtom{}
		dst := []dependencygraph.DefUseAtom{}
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			srcOffset := srcBufferData.span.Start + uint64(region.SrcOffset)
			dstOffset := dstBufferData.span.Start + uint64(region.DstOffset)
			src = append(src, memorySpan{
				span:   interval.U64Span{Start: srcOffset, End: srcOffset + uint64(region.Size)},
				memory: srcBufferData.memory})
			dst = append(dst, memorySpan{
				span:   interval.U64Span{Start: dstOffset, End: dstOffset + uint64(region.Size)},
				memory: dstBufferData.memory})
		}
		vb.recordReadsWritesModifies(
			ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
	case *RecreateCmdCopyBuffer:
		srcBufferData, _ := vb.getBufferData(ctx, bh, cmd.SrcBuffer).(memorySpan)
		dstBufferData, _ := vb.getBufferData(ctx, bh, cmd.DstBuffer).(memorySpan)
		src := []dependencygraph.DefUseAtom{}
		dst := []dependencygraph.DefUseAtom{}
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			srcOffset := srcBufferData.span.Start + uint64(region.SrcOffset)
			dstOffset := dstBufferData.span.Start + uint64(region.DstOffset)
			src = append(src, memorySpan{
				span:   interval.U64Span{Start: srcOffset, End: srcOffset + uint64(region.Size)},
				memory: srcBufferData.memory})
			dst = append(dst, memorySpan{
				span:   interval.U64Span{Start: dstOffset, End: dstOffset + uint64(region.Size)},
				memory: dstBufferData.memory})
		}
		vb.recordReadsWritesModifies(
			ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)

	case *VkCmdCopyImageToBuffer:
		// TODO: calculate the ranges for the overwritten data
		dst := []dependencygraph.DefUseAtom{vb.getBufferData(ctx, bh, cmd.DstBuffer)}
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		vb.recordReadsWritesModifies(
			ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
	case *RecreateCmdCopyImageToBuffer:
		dst := []dependencygraph.DefUseAtom{vb.getBufferData(ctx, bh, cmd.DstBuffer)}
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		vb.recordReadsWritesModifies(
			ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)

	case *VkCmdCopyBufferToImage:
		// TODO: calculate the ranges for the source data
		src := []dependencygraph.DefUseAtom{vb.getBufferData(ctx, bh, cmd.SrcBuffer)}
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.ImageSubresource, region.ImageOffset, region.ImageExtent)
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}
	case *RecreateCmdCopyBufferToImage:
		src := []dependencygraph.DefUseAtom{vb.getBufferData(ctx, bh, cmd.SrcBuffer)}
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.ImageSubresource, region.ImageOffset, region.ImageExtent)
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}

	case *VkCmdBlitImage:
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || blitFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.DstSubresource,
				region.DstOffsets[0], region.DstOffsets[1])
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}
	case *RecreateCmdBlitImage:
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || blitFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.DstSubresource,
				region.DstOffsets[0], region.DstOffsets[1])
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}

	case *VkCmdResolveImage:
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.DstSubresource, region.DstOffset, region.Extent)
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}
	case *RecreateCmdResolveImage:
		src := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.SrcImage)}
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.DstImage)}
		overwritten := false
		count := uint64(cmd.RegionCount)
		regions := cmd.PRegions.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			region := regions.Index(i, l).Read(ctx, cmd, s, nil)
			overwritten = overwritten || subresourceLayersFullyCoverImage(
				GetState(s).Images.Get(cmd.DstImage),
				region.DstSubresource, region.DstOffset, region.Extent)
		}
		if overwritten {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(
				ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
		}

	case *VkCmdFillBuffer:
		dstBufferData, _ := vb.getBufferData(ctx, bh, cmd.DstBuffer).(memorySpan)
		dst := []dependencygraph.DefUseAtom{
			getSubBufferData(dstBufferData, uint64(cmd.DstOffset), uint64(cmd.Size))}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
			emptyAtoms, dst, emptyAtoms)
	case *RecreateCmdFillBuffer:
		dstBufferData, _ := vb.getBufferData(ctx, bh, cmd.DstBuffer).(memorySpan)
		dst := []dependencygraph.DefUseAtom{
			getSubBufferData(dstBufferData, uint64(cmd.DstOffset), uint64(cmd.Size))}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
			emptyAtoms, dst, emptyAtoms)

	case *VkCmdUpdateBuffer:
		dstBufferData, _ := vb.getBufferData(ctx, bh, cmd.DstBuffer).(memorySpan)
		dstOffset := dstBufferData.span.Start + uint64(cmd.DstOffset)
		dstEnd := dstBufferData.span.End + uint64(cmd.DataSize)
		dst := []dependencygraph.DefUseAtom{memorySpan{
			span:   interval.U64Span{Start: dstOffset, End: dstEnd},
			memory: dstBufferData.memory}}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
			emptyAtoms, dst, emptyAtoms)
	case *RecreateCmdUpdateBuffer:
		dstBufferData, _ := vb.getBufferData(ctx, bh, cmd.DstBuffer).(memorySpan)
		dstOffset := dstBufferData.span.Start + uint64(cmd.DstOffset)
		dstEnd := dstBufferData.span.End + uint64(cmd.DataSize)
		dst := []dependencygraph.DefUseAtom{memorySpan{
			span:   interval.U64Span{Start: dstOffset, End: dstEnd},
			memory: dstBufferData.memory}}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
			emptyAtoms, dst, emptyAtoms)

	case *VkCmdClearColorImage:
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.Image)}
		count := uint64(cmd.RangeCount)
		rngs := cmd.PRanges.Slice(0, count, l)
		overwritten := false
		for i := uint64(0); i < count; i++ {
			rng := rngs.Index(i, l).Read(ctx, cmd, s, nil)
			if subresourceRangeFullyCoverImage(GetState(s).Images.Get(cmd.Image), rng) {
				overwritten = true
			}
		}
		if overwritten {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, emptyAtoms, dst)
		}
	case *RecreateCmdClearColorImage:
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.Image)}
		count := uint64(cmd.RangeCount)
		rngs := cmd.PRanges.Slice(0, count, l)
		overwritten := false
		for i := uint64(0); i < count; i++ {
			rng := rngs.Index(i, l).Read(ctx, cmd, s, nil)
			if subresourceRangeFullyCoverImage(GetState(s).Images.Get(cmd.Image), rng) {
				overwritten = true
			}
		}
		if overwritten {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, emptyAtoms, dst)
		}

	case *VkCmdClearDepthStencilImage:
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.Image)}
		count := uint64(cmd.RangeCount)
		rngs := cmd.PRanges.Slice(0, count, l)
		overwritten := false
		for i := uint64(0); i < count; i++ {
			rng := rngs.Index(i, l).Read(ctx, cmd, s, nil)
			if subresourceRangeFullyCoverImage(GetState(s).Images.Get(cmd.Image), rng) {
				overwritten = true
			}
		}
		if overwritten {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, emptyAtoms, dst)
		}
	case *RecreateCmdClearDepthStencilImage:
		dst := []dependencygraph.DefUseAtom{vb.getImageData(ctx, bh, cmd.Image)}
		count := uint64(cmd.RangeCount)
		rngs := cmd.PRanges.Slice(0, count, l)
		overwritten := false
		for i := uint64(0); i < count; i++ {
			rng := rngs.Index(i, l).Read(ctx, cmd, s, nil)
			if subresourceRangeFullyCoverImage(GetState(s).Images.Get(cmd.Image), rng) {
				overwritten = true
			}
		}
		if overwritten {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, dst, emptyAtoms)
		} else {
			vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer,
				emptyAtoms, emptyAtoms, dst)
		}

	// renderpass and subpass
	case *VkCmdBeginRenderPass:
		vkRp := cmd.PRenderPassBegin.Read(ctx, cmd, s, nil).RenderPass
		read(ctx, bh, vkHandle(vkRp))
		vkFb := cmd.PRenderPassBegin.Read(ctx, cmd, s, nil).Framebuffer
		read(ctx, bh, vkHandle(vkFb))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		rp := GetState(s).RenderPasses.Get(vkRp)
		fb := GetState(s).Framebuffers.Get(vkFb)
		read(ctx, bh, vkHandle(fb.RenderPass.VulkanHandle))
		for _, ia := range fb.ImageAttachments.Range() {
			read(ctx, bh, vkHandle(ia.VulkanHandle))
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.beginRenderPass(ctx, vb, cbh, rp, fb)
			execInfo.renderPassBegin = newForwardPairedLabel(ctx, cbh)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdBeginRenderPass:
		vkRp := cmd.PRenderPassBegin.Read(ctx, cmd, s, nil).RenderPass
		read(ctx, bh, vkHandle(vkRp))
		vkFb := cmd.PRenderPassBegin.Read(ctx, cmd, s, nil).Framebuffer
		read(ctx, bh, vkHandle(vkFb))
		write(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		rp := GetState(s).RenderPasses.Get(vkRp)
		fb := GetState(s).Framebuffers.Get(vkFb)
		read(ctx, bh, vkHandle(fb.RenderPass.VulkanHandle))
		for _, ia := range fb.ImageAttachments.Range() {
			read(ctx, bh, vkHandle(ia.VulkanHandle))
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.beginRenderPass(ctx, vb, cbh, rp, fb)
			execInfo.renderPassBegin = newForwardPairedLabel(ctx, cbh)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdNextSubpass:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.nextSubpass(ctx, ft, cbh, sc, vb.machine)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdNextSubpass:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.nextSubpass(ctx, ft, cbh, sc, vb.machine)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdEndRenderPass:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.endRenderPass(ctx, ft, cbh, sc, vb.machine)
			read(ctx, cbh, execInfo.renderPassBegin)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdEndRenderPass:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.endRenderPass(ctx, ft, cbh, sc, vb.machine)
			read(ctx, cbh, execInfo.renderPassBegin)
			ft.AddBehaviour(ctx, cbh)
		}

	// bind vertex buffers, index buffer, pipeline and descriptors
	case *VkCmdBindVertexBuffers:
		count := uint64(cmd.BindingCount)
		vkBufs := cmd.PBuffers.Slice(0, count, l)
		res := []dependencygraph.DefUseAtom{}
		for i := uint64(0); i < count; i++ {
			vkBuf := vkBufs.Index(i, l).Read(ctx, cmd, s, nil)
			res = append(res, vb.getBufferData(ctx, bh, vkBuf))
		}
		firstBinding := cmd.FirstBinding
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			for i, r := range res {
				binding := firstBinding + uint32(i)
				// TODO: handle offsets specified in pOffsets
				execInfo.currentCmdBufState.vertexBuffers[binding] = newBoundData(ctx, cbh, r)
			}
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdBindVertexBuffers:
		count := uint64(cmd.BindingCount)
		vkBufs := cmd.PBuffers.Slice(0, count, l)
		res := []dependencygraph.DefUseAtom{}
		for i := uint64(0); i < count; i++ {
			vkBuf := vkBufs.Index(i, l).Read(ctx, cmd, s, nil)
			res = append(res, vb.getBufferData(ctx, bh, vkBuf))
		}
		firstBinding := cmd.FirstBinding
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			for i, r := range res {
				binding := firstBinding + uint32(i)
				// TODO: handle offsets specified in pOffsets
				execInfo.currentCmdBufState.vertexBuffers[binding] = newBoundData(ctx, cbh, r)
			}
			ft.AddBehaviour(ctx, cbh)
		}
	case *VkCmdBindIndexBuffer:
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		res := getSubBufferData(bufData, uint64(cmd.Offset), vkWholeSize)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.currentCmdBufState.indexBuffer = newBoundData(ctx, cbh, res)
			execInfo.currentCmdBufState.indexType = cmd.IndexType
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdBindIndexBuffer:
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		res := getSubBufferData(bufData, uint64(cmd.Offset), vkWholeSize)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			execInfo.currentCmdBufState.indexBuffer = newBoundData(ctx, cbh, res)
			execInfo.currentCmdBufState.indexType = cmd.IndexType
			ft.AddBehaviour(ctx, cbh)
		}
	case *VkCmdBindPipeline:
		bh.Alive = true
		vkPi := cmd.Pipeline
		read(ctx, bh, vkHandle(vkPi))
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, vkHandle(vkPi))
			write(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdBindPipeline:
		vkPi := cmd.Pipeline
		read(ctx, bh, vkHandle(vkPi))
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, vkHandle(vkPi))
			write(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			ft.AddBehaviour(ctx, cbh)
		}
	case *VkCmdBindDescriptorSets:
		read(ctx, bh, vkHandle(cmd.Layout))
		count := uint64(cmd.DescriptorSetCount)
		vkSets := cmd.PDescriptorSets.Slice(0, count, l)
		dss := []*descriptorSet{}
		for i := uint64(0); i < count; i++ {
			vkSet := vkSets.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkSet))
			dss = append(dss, vb.descriptorSets[vkSet])
		}
		firstSet := cmd.FirstSet
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			for i, ds := range dss {
				set := firstSet + uint32(i)
				execInfo.currentCmdBufState.descriptorSets[set] = newBoundDescriptor(ctx, cbh, ds)
			}
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdBindDescriptorSets:
		read(ctx, bh, vkHandle(cmd.Layout))
		count := uint64(cmd.DescriptorSetCount)
		vkSets := cmd.PDescriptorSets.Slice(0, count, l)
		dss := []*descriptorSet{}
		for i := uint64(0); i < count; i++ {
			vkSet := vkSets.Index(i, l).Read(ctx, cmd, s, nil)
			dss = append(dss, vb.descriptorSets[vkSet])
		}
		firstSet := cmd.FirstSet
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			for i, ds := range dss {
				set := firstSet + uint32(i)
				execInfo.currentCmdBufState.descriptorSets[set] = newBoundDescriptor(ctx, cbh, ds)
			}
			ft.AddBehaviour(ctx, cbh)
		}

	// draw and dispatch
	case *VkCmdDraw:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.draw(ctx, cbh, execInfo)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdDraw:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.draw(ctx, cbh, execInfo)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdDrawIndexed:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.readBoundIndexBuffer(ctx, cbh, execInfo, cmd)
			vb.draw(ctx, cbh, execInfo)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdDrawIndexed:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.readBoundIndexBuffer(ctx, cbh, execInfo, cmd)
			vb.draw(ctx, cbh, execInfo)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdDrawIndirect:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		count := uint64(cmd.DrawCount)
		sizeOfDrawIndirectdCommand := uint64(4 * 4)
		offset := uint64(cmd.Offset)
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		src := []dependencygraph.DefUseAtom{}
		for i := uint64(0); i < count; i++ {
			src = append(src, memorySpan{span: interval.U64Span{
				Start: offset, End: offset + sizeOfDrawIndirectdCommand},
				memory: bufData.memory})
			offset += uint64(cmd.Stride)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.draw(ctx, cbh, execInfo)
			readMultiple(ctx, cbh, src)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdDrawIndirect:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		count := uint64(cmd.DrawCount)
		sizeOfDrawIndirectdCommand := uint64(4 * 4)
		offset := uint64(cmd.Offset)
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		src := []dependencygraph.DefUseAtom{}
		for i := uint64(0); i < count; i++ {
			src = append(src, memorySpan{span: interval.U64Span{
				Start: offset, End: offset + sizeOfDrawIndirectdCommand},
				memory: bufData.memory})
			offset += uint64(cmd.Stride)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.draw(ctx, cbh, execInfo)
			readMultiple(ctx, cbh, src)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdDrawIndexedIndirect:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		count := uint64(cmd.DrawCount)
		sizeOfDrawIndexedIndirectCommand := uint64(5 * 4)
		offset := uint64(cmd.Offset)
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		src := []dependencygraph.DefUseAtom{}
		for i := uint64(0); i < count; i++ {
			src = append(src, memorySpan{span: interval.U64Span{
				Start: offset, End: offset + sizeOfDrawIndexedIndirectCommand},
				memory: bufData.memory})
			offset += uint64(cmd.Stride)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.readBoundIndexBuffer(ctx, cbh, execInfo, cmd)
			vb.draw(ctx, cbh, execInfo)
			readMultiple(ctx, cbh, src)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdDrawIndexedIndirect:
		read(ctx, bh, vb.commandBuffers[cmd.CommandBuffer].renderPassBegin)
		count := uint64(cmd.DrawCount)
		sizeOfDrawIndexedIndirectCommand := uint64(5 * 4)
		offset := uint64(cmd.Offset)
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		src := []dependencygraph.DefUseAtom{}
		for i := uint64(0); i < count; i++ {
			src = append(src, memorySpan{span: interval.U64Span{
				Start: offset, End: offset + sizeOfDrawIndexedIndirectCommand},
				memory: bufData.memory})
			offset += uint64(cmd.Stride)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			vb.readBoundIndexBuffer(ctx, cbh, execInfo, cmd)
			vb.draw(ctx, cbh, execInfo)
			readMultiple(ctx, cbh, src)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdDispatch:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			modified := vb.useBoundDescriptorSets(ctx, cbh, execInfo.currentCmdBufState)
			modifyMultiple(ctx, cbh, modified)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdDispatch:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			modified := vb.useBoundDescriptorSets(ctx, cbh, execInfo.currentCmdBufState)
			modifyMultiple(ctx, cbh, modified)
			ft.AddBehaviour(ctx, cbh)
		}

	case *VkCmdDispatchIndirect:
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		sizeOfDispatchIndirectCommand := uint64(3 * 4)
		src := memorySpan{span: interval.U64Span{
			Start: bufData.span.Start + uint64(cmd.Offset),
			End:   bufData.span.Start + uint64(cmd.Offset) + sizeOfDispatchIndirectCommand,
		}, memory: bufData.memory}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			modified := vb.useBoundDescriptorSets(ctx, cbh, execInfo.currentCmdBufState)
			modifyMultiple(ctx, cbh, modified)
			read(ctx, cbh, src)
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdDispatchIndirect:
		bufData, _ := vb.getBufferData(ctx, bh, cmd.Buffer).(memorySpan)
		sizeOfDispatchIndirectCommand := uint64(3 * 4)
		src := memorySpan{span: interval.U64Span{
			Start: bufData.span.Start + uint64(cmd.Offset),
			End:   bufData.span.Start + uint64(cmd.Offset) + sizeOfDispatchIndirectCommand,
		}, memory: bufData.memory}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			read(ctx, cbh, execInfo.currentCmdBufState.pipeline)
			modified := vb.useBoundDescriptorSets(ctx, cbh, execInfo.currentCmdBufState)
			modifyMultiple(ctx, cbh, modified)
			read(ctx, cbh, src)
			ft.AddBehaviour(ctx, cbh)
		}

	// pipeline settings
	case *VkCmdPushConstants:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdPushConstants:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetLineWidth:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetLineWidth:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetScissor:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetScissor:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetViewport:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetViewport:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetDepthBias:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetDepthBias:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetDepthBounds:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetDepthBounds:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetBlendConstants:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetBlendConstants:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetStencilCompareMask:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetStencilCompareMask:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetStencilWriteMask:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetStencilWriteMask:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *VkCmdSetStencilReference:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)
	case *RecreateCmdSetStencilReference:
		vb.recordModifingDynamicStates(ctx, ft, bh, cmd.CommandBuffer)

	// clear attachments
	case *VkCmdClearAttachments:
		attCount := uint64(cmd.AttachmentCount)
		atts := []VkClearAttachment{}
		rectCount := uint64(cmd.RectCount)
		rects := []VkClearRect{}
		for i := uint64(0); i < attCount; i++ {
			att := cmd.PAttachments.Slice(0, attCount, l).Index(i, l).Read(ctx, cmd, s, nil)
			atts = append(atts, att)
		}
		for i := uint64(0); i < rectCount; i++ {
			rect := cmd.PRects.Slice(0, rectCount, l).Index(i, l).Read(ctx, cmd, s, nil)
			rects = append(rects, rect)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			for _, a := range atts {
				clearAttachmentData(ctx, cbh, execInfo, a, rects)
			}
			ft.AddBehaviour(ctx, cbh)
		}
	case *RecreateCmdClearAttachments:
		attCount := uint64(cmd.AttachmentCount)
		atts := []VkClearAttachment{}
		rectCount := uint64(cmd.RectCount)
		rects := []VkClearRect{}
		for i := uint64(0); i < attCount; i++ {
			att := cmd.PAttachments.Slice(0, attCount, l).Index(i, l).Read(ctx, cmd, s, nil)
			atts = append(atts, att)
		}
		for i := uint64(0); i < rectCount; i++ {
			rect := cmd.PRects.Slice(0, rectCount, l).Index(i, l).Read(ctx, cmd, s, nil)
			rects = append(rects, rect)
		}
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.behave = func(sc submittedCommand,
			execInfo *queueExecutionInfo) {
			cbh := sc.cmd.newBehaviour(ctx, sc, vb.machine, execInfo)
			for _, a := range atts {
				clearAttachmentData(ctx, cbh, execInfo, a, rects)
			}
			ft.AddBehaviour(ctx, cbh)
		}

	// query pool commands
	case *VkCmdResetQueryPool:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		resetLabels := []dependencygraph.DefUseAtom{}
		count := uint64(cmd.QueryCount)
		first := uint64(cmd.FirstQuery)
		for i := uint64(0); i < count; i++ {
			resetLabels = append(resetLabels,
				vb.querypools[cmd.QueryPool].queries[first+i].reset)
		}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, emptyAtoms,
			resetLabels, emptyAtoms)
	case *RecreateCmdResetQueryPool:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		resetLabels := []dependencygraph.DefUseAtom{}
		count := uint64(cmd.QueryCount)
		first := uint64(cmd.FirstQuery)
		for i := uint64(0); i < count; i++ {
			resetLabels = append(resetLabels,
				vb.querypools[cmd.QueryPool].queries[first+i].reset)
		}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, emptyAtoms,
			resetLabels, emptyAtoms)
	case *VkCmdBeginQuery:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		resetLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].reset}
		beginLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].begin}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, resetLabels,
			beginLabels, emptyAtoms)
	case *RecreateCmdBeginQuery:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		resetLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].reset}
		beginLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].begin}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, resetLabels,
			beginLabels, emptyAtoms)
	case *VkCmdEndQuery:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		endAndResultLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].end,
			vb.querypools[cmd.QueryPool].queries[cmd.Query].result,
		}
		beginLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].begin}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, beginLabels,
			endAndResultLabels, emptyAtoms)
	case *RecreateCmdEndQuery:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		endAndResultLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].end,
			vb.querypools[cmd.QueryPool].queries[cmd.Query].result,
		}
		beginLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].begin}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, beginLabels,
			endAndResultLabels, emptyAtoms)
	case *VkCmdWriteTimestamp:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		resetLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].reset}
		resultLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].result}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, resetLabels,
			resultLabels, emptyAtoms)
	case *RecreateCmdWriteTimestamp:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		resetLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].reset}
		resultLabels := []dependencygraph.DefUseAtom{
			vb.querypools[cmd.QueryPool].queries[cmd.Query].result}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, resetLabels,
			resultLabels, emptyAtoms)
	case *VkCmdCopyQueryPoolResults:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		// TODO: calculate the range
		src := []dependencygraph.DefUseAtom{}
		dst := []dependencygraph.DefUseAtom{vb.getBufferData(ctx, bh, cmd.DstBuffer)}
		count := uint64(cmd.QueryCount)
		first := uint64(cmd.FirstQuery)
		for i := uint64(0); i < count; i++ {
			src = append(src, vb.querypools[cmd.QueryPool].queries[first+i].result)
		}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)
	case *RecreateCmdCopyQueryPoolResults:
		read(ctx, bh, vkHandle(cmd.QueryPool))
		src := []dependencygraph.DefUseAtom{}
		dst := []dependencygraph.DefUseAtom{vb.getBufferData(ctx, bh, cmd.DstBuffer)}
		count := uint64(cmd.QueryCount)
		first := uint64(cmd.FirstQuery)
		for i := uint64(0); i < count; i++ {
			src = append(src, vb.querypools[cmd.QueryPool].queries[first+i].result)
		}
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, src, emptyAtoms, dst)

	// event commandbuffer commands
	case *VkCmdSetEvent:
		read(ctx, bh, vkHandle(cmd.Event))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, emptyAtoms,
			[]dependencygraph.DefUseAtom{vb.events[cmd.Event].signal}, emptyAtoms)
	case *RecreateCmdSetEvent:
		read(ctx, bh, vkHandle(cmd.Event))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, emptyAtoms,
			[]dependencygraph.DefUseAtom{vb.events[cmd.Event].signal}, emptyAtoms)
	case *VkCmdResetEvent:
		read(ctx, bh, vkHandle(cmd.Event))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, emptyAtoms,
			[]dependencygraph.DefUseAtom{vb.events[cmd.Event].unsignal}, emptyAtoms)
	case *RecreateCmdResetEvent:
		read(ctx, bh, vkHandle(cmd.Event))
		vb.recordReadsWritesModifies(ctx, ft, bh, cmd.CommandBuffer, emptyAtoms,
			[]dependencygraph.DefUseAtom{vb.events[cmd.Event].unsignal}, emptyAtoms)
	case *VkCmdWaitEvents:
		eventLabels := []dependencygraph.DefUseAtom{}
		evCount := uint64(cmd.EventCount)
		vkEvs := cmd.PEvents.Slice(0, evCount, l)
		for i := uint64(0); i < evCount; i++ {
			vkEv := vkEvs.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkEv))
			eventLabels = append(eventLabels, vb.events[vkEv].signal,
				vb.events[vkEv].unsignal)
		}
		vb.recordBarriers(ctx, s, ft, cmd, bh, cmd.CommandBuffer, cmd.MemoryBarrierCount,
			cmd.BufferMemoryBarrierCount, cmd.PBufferMemoryBarriers,
			cmd.ImageMemoryBarrierCount, cmd.PImageMemoryBarriers, eventLabels)
	case *RecreateCmdWaitEvents:
		eventLabels := []dependencygraph.DefUseAtom{}
		evCount := uint64(cmd.EventCount)
		vkEvs := cmd.PEvents.Slice(0, evCount, l)
		for i := uint64(0); i < evCount; i++ {
			vkEv := vkEvs.Index(i, l).Read(ctx, cmd, s, nil)
			read(ctx, bh, vkHandle(vkEv))
			eventLabels = append(eventLabels, vb.events[vkEv].signal,
				vb.events[vkEv].unsignal)
		}
		vb.recordBarriers(ctx, s, ft, cmd, bh, cmd.CommandBuffer, cmd.MemoryBarrierCount,
			cmd.BufferMemoryBarrierCount, cmd.PBufferMemoryBarriers,
			cmd.ImageMemoryBarrierCount, cmd.PImageMemoryBarriers, eventLabels)

	// pipeline barrier
	case *VkCmdPipelineBarrier:
		vb.recordBarriers(ctx, s, ft, cmd, bh, cmd.CommandBuffer, cmd.MemoryBarrierCount,
			cmd.BufferMemoryBarrierCount, cmd.PBufferMemoryBarriers,
			cmd.ImageMemoryBarrierCount, cmd.PImageMemoryBarriers, emptyAtoms)
	case *RecreateCmdPipelineBarrier:
		vb.recordBarriers(ctx, s, ft, cmd, bh, cmd.CommandBuffer, cmd.MemoryBarrierCount,
			cmd.BufferMemoryBarrierCount, cmd.PBufferMemoryBarriers,
			cmd.ImageMemoryBarrierCount, cmd.PImageMemoryBarriers, emptyAtoms)

	// secondary command buffers
	case *VkCmdExecuteCommands:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.isCmdExecuteCommands = true
		count := uint64(cmd.CommandBufferCount)
		scbs := cmd.PCommandBuffers.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			vkScb := scbs.Index(i, l).Read(ctx, cmd, s, nil)
			cbc.recordSecondaryCommandBuffer(vkScb)
			read(ctx, bh, vkHandle(vkScb))
		}
		cbc.behave = func(sc submittedCommand, execInfo *queueExecutionInfo) {}
	case *RecreateCmdExecuteCommands:
		cbc := vb.newCommand(ctx, bh, cmd.CommandBuffer)
		cbc.isCmdExecuteCommands = true
		count := uint64(cmd.CommandBufferCount)
		scbs := cmd.PCommandBuffers.Slice(0, count, l)
		for i := uint64(0); i < count; i++ {
			vkScb := scbs.Index(i, l).Read(ctx, cmd, s, nil)
			cbc.recordSecondaryCommandBuffer(vkScb)
			read(ctx, bh, vkHandle(vkScb))
		}
		cbc.behave = func(sc submittedCommand, execInfo *queueExecutionInfo) {}

	// execution triggering
	case *VkQueueSubmit:
		read(ctx, bh, vkHandle(cmd.Queue))
		if _, ok := vb.executionInfos[cmd.Queue]; !ok {
			vb.executionInfos[cmd.Queue] = newQueueExecutionInfo(id)
		}
		vb.executionInfos[cmd.Queue].lastSubmitID = id
		// collect submission info and submitted commands
		vb.submitInfos[id] = &queueSubmitInfo{
			executionBegin: newLabel(),
			executionEnd:   newLabel(),
			queue:          cmd.Queue,
		}
		if cmd.Extras() != nil && cmd.Extras().Observations() != nil {
			for _, r := range cmd.Extras().Observations().Reads {
				if r.Range.Size == uint64(4096) {
					bh.Alive = true
					break
				}
			}
		}
		submitCount := uint64(cmd.SubmitCount)
		submits := cmd.PSubmits.Slice(0, submitCount, l)
		for i := uint64(0); i < submitCount; i++ {
			submit := submits.Index(i, l).Read(ctx, cmd, s, nil)
			commandBufferCount := uint64(submit.CommandBufferCount)
			commandBuffers := submit.PCommandBuffers.Slice(0, commandBufferCount, l)
			for j := uint64(0); j < commandBufferCount; j++ {
				vkCb := commandBuffers.Index(j, l).Read(ctx, cmd, s, nil)
				read(ctx, bh, vkHandle(vkCb))
				read(ctx, bh, vb.commandBuffers[vkCb].end)
				for k, cbc := range vb.commands[vkCb] {
					if cbc.isCmdExecuteCommands {
						for scbi, scb := range cbc.secondaryCommandBuffers {
							read(ctx, bh, vb.commandBuffers[scb].end)
							for sci, scbc := range vb.commands[scb] {
								fci := api.SubCmdIdx{uint64(id), i, j, uint64(k), uint64(scbi), uint64(sci)}
								submittedCmd := newSubmittedCommand(fci, scbc, cbc)
								vb.submitInfos[id].pendingCommands = append(vb.submitInfos[id].pendingCommands, submittedCmd)
							}
						}
					}
					fci := api.SubCmdIdx{uint64(id), i, j, uint64(k)}
					submittedCmd := newSubmittedCommand(fci, cbc, nil)
					vb.submitInfos[id].pendingCommands = append(vb.submitInfos[id].pendingCommands, submittedCmd)
				}
			}
			waitSemaphoreCount := uint64(submit.WaitSemaphoreCount)
			waitSemaphores := submit.PWaitSemaphores.Slice(0, waitSemaphoreCount, l)
			for i := uint64(0); i < waitSemaphoreCount; i++ {
				vkSp := waitSemaphores.Index(i, l).Read(ctx, cmd, s, nil)
				vb.submitInfos[id].waitSemaphores = append(
					vb.submitInfos[id].waitSemaphores, vkSp)
			}
			signalSemaphoreCount := uint64(submit.SignalSemaphoreCount)
			signalSemaphores := submit.PSignalSemaphores.Slice(0, signalSemaphoreCount, l)
			for i := uint64(0); i < signalSemaphoreCount; i++ {
				vkSp := signalSemaphores.Index(i, l).Read(ctx, cmd, s, nil)
				vb.submitInfos[id].signalSemaphores = append(
					vb.submitInfos[id].signalSemaphores, vkSp)
			}
		}
		vb.submitInfos[id].signalFence = cmd.Fence

		// queue execution begin
		write(ctx, bh, vb.submitInfos[id].executionBegin)
		vb.writeCoherentMemoryData(ctx, cmd, bh)
		if read(ctx, bh, vkHandle(cmd.Fence)) {
			read(ctx, bh, vb.fences[cmd.Fence].unsignal)
		}
		for _, sp := range vb.submitInfos[id].waitSemaphores {
			if read(ctx, bh, vkHandle(sp)) {
				read(ctx, bh, vb.semaphoreSignals[sp])
			}
		}
		for _, sp := range vb.submitInfos[id].signalSemaphores {
			read(ctx, bh, vkHandle(sp))
		}
		read(ctx, bh, vkHandle(vb.submitInfos[id].signalFence))

	case *VkSetEvent:
		if read(ctx, bh, vkHandle(cmd.Event)) {
			write(ctx, bh, vb.events[cmd.Event].signal)
			vb.writeCoherentMemoryData(ctx, cmd, bh)
			bh.Alive = true
		}

	// synchronization primitives
	case *VkResetEvent:
		if read(ctx, bh, vkHandle(cmd.Event)) {
			write(ctx, bh, vb.events[cmd.Event].unsignal)
			bh.Alive = true
		}

	case *VkCreateSemaphore:
		vkSp := cmd.PSemaphore.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkSp))
		vb.semaphoreSignals[vkSp] = newLabel()
	case *RecreateSemaphore:
		vkSp := cmd.PSemaphore.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkSp))
		vb.semaphoreSignals[vkSp] = newLabel()
		write(ctx, bh, vb.semaphoreSignals[vkSp])
	case *VkDestroySemaphore:
		vkSp := cmd.Semaphore
		if read(ctx, bh, vkHandle(vkSp)) {
			delete(vb.semaphoreSignals, vkSp)
			bh.Alive = true
		}

	case *VkCreateEvent:
		vkEv := cmd.PEvent.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkEv))
		vb.events[vkEv] = &event{signal: newLabel(), unsignal: newLabel()}
	case *RecreateEvent:
		vkEv := cmd.PEvent.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkEv))
		vb.events[vkEv] = &event{signal: newLabel(), unsignal: newLabel()}
		if cmd.Signaled != VkBool32(0) {
			write(ctx, bh, vb.events[vkEv].signal)
		}
	case *VkGetEventStatus:
		vkEv := cmd.Event
		if read(ctx, bh, vkHandle(vkEv)) {
			read(ctx, bh, vb.events[vkEv].signal)
			read(ctx, bh, vb.events[vkEv].unsignal)
			bh.Alive = true
		}
	case *VkDestroyEvent:
		vkEv := cmd.Event
		if read(ctx, bh, vkHandle(vkEv)) {
			delete(vb.events, vkEv)
			bh.Alive = true
		}

	case *VkCreateFence:
		vkFe := cmd.PFence.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkFe))
		vb.fences[vkFe] = &fence{signal: newLabel(), unsignal: newLabel()}
	case *RecreateFence:
		vkFe := cmd.PFence.Read(ctx, cmd, s, nil)
		write(ctx, bh, vkHandle(vkFe))
		vb.fences[vkFe] = &fence{signal: newLabel(), unsignal: newLabel()}
	case *VkGetFenceStatus:
		vkFe := cmd.Fence
		if read(ctx, bh, vkHandle(vkFe)) {
			read(ctx, bh, vb.fences[vkFe].signal)
			read(ctx, bh, vb.fences[vkFe].unsignal)
			bh.Alive = true
		}
	case *VkWaitForFences:
		fenceCount := uint64(cmd.FenceCount)
		vkFes := cmd.PFences.Slice(0, fenceCount, l)
		for i := uint64(0); i < fenceCount; i++ {
			vkFe := vkFes.Index(i, l).Read(ctx, cmd, s, nil)
			if read(ctx, bh, vkHandle(vkFe)) {
				read(ctx, bh, vb.fences[vkFe].signal)
				read(ctx, bh, vb.fences[vkFe].unsignal)
				bh.Alive = true
			}
		}
	case *VkResetFences:
		fenceCount := uint64(cmd.FenceCount)
		vkFes := cmd.PFences.Slice(0, fenceCount, l)
		for i := uint64(0); i < fenceCount; i++ {
			vkFe := vkFes.Index(i, l).Read(ctx, cmd, s, nil)
			if read(ctx, bh, vkHandle(vkFe)) {
				write(ctx, bh, vb.fences[vkFe].unsignal)
				bh.Alive = true
			}
		}
	case *VkDestroyFence:
		vkFe := cmd.Fence
		if read(ctx, bh, vkHandle(vkFe)) {
			delete(vb.fences, vkFe)
			bh.Alive = true
		}

	case *VkQueueWaitIdle:
		vkQu := cmd.Queue
		if read(ctx, bh, vkHandle(vkQu)) {
			if _, ok := vb.executionInfos[vkQu]; ok {
				bh.Alive = true
			}
		}

	case *VkDeviceWaitIdle:
		for _, qei := range vb.executionInfos {
			lastSubmitInfo := vb.submitInfos[qei.lastSubmitID]
			read(ctx, bh, lastSubmitInfo.executionEnd)
			bh.Alive = true
		}

	// Property queries, can be dropped if they are not the requested command.
	case *VkGetDeviceMemoryCommitment:
		read(ctx, bh, vkHandle(cmd.Memory))
	case *VkGetImageMemoryRequirements:
		read(ctx, bh, vkHandle(cmd.Image))
	case *VkGetImageSparseMemoryRequirements:
		read(ctx, bh, vkHandle(cmd.Image))
	case *VkGetImageSubresourceLayout:
		read(ctx, bh, vkHandle(cmd.Image))
	case *VkGetBufferMemoryRequirements:
		read(ctx, bh, vkHandle(cmd.Buffer))
	case *VkGetRenderAreaGranularity:
		read(ctx, bh, vkHandle(cmd.RenderPass))

	// Kept alive
	case *VkGetDeviceProcAddr,
		*VkGetInstanceProcAddr:
		bh.Alive = true
	case *VkCreateInstance,
		*RecreateInstance:
		bh.Alive = true
	case *VkCreateDevice,
		*RecreateDevice,
		*RecreatePhysicalDevices,
		*RecreatePhysicalDeviceProperties:
		bh.Alive = true
	case *VkGetDeviceQueue,
		*RecreateQueue:
		bh.Alive = true
	case *VkCreateDescriptorPool,
		*RecreateDescriptorPool:
		bh.Alive = true
	case *VkCreateAndroidSurfaceKHR,
		*RecreateAndroidSurfaceKHR,
		*VkCreateXlibSurfaceKHR,
		*RecreateXlibSurfaceKHR,
		*VkCreateXcbSurfaceKHR,
		*RecreateXCBSurfaceKHR,
		*VkCreateWaylandSurfaceKHR,
		*RecreateWaylandSurfaceKHR,
		*VkCreateMirSurfaceKHR,
		*RecreateMirSurfaceKHR:
		bh.Alive = true
	case *VkCreateCommandPool,
		*RecreateCommandPool:
		bh.Alive = true
	case *VkGetPhysicalDeviceXlibPresentationSupportKHR,
		*VkGetPhysicalDeviceXcbPresentationSupportKHR,
		*VkGetPhysicalDeviceWaylandPresentationSupportKHR,
		*VkGetPhysicalDeviceMirPresentationSupportKHR:
		bh.Alive = true
	case *VkGetPhysicalDeviceProperties,
		*VkGetPhysicalDeviceMemoryProperties,
		*VkGetPhysicalDeviceQueueFamilyProperties,
		*VkGetPhysicalDeviceDisplayPropertiesKHR,
		*VkGetPhysicalDeviceDisplayPlanePropertiesKHR,
		*VkGetPhysicalDeviceFeatures,
		*VkGetPhysicalDeviceFormatProperties,
		*VkGetPhysicalDeviceImageFormatProperties,
		*VkGetPhysicalDeviceSparseImageFormatProperties:
		bh.Alive = true
	case *VkGetPhysicalDeviceSurfaceSupportKHR,
		*VkGetPhysicalDeviceSurfaceCapabilitiesKHR,
		*VkGetPhysicalDeviceSurfaceFormatsKHR,
		*VkGetPhysicalDeviceSurfacePresentModesKHR:
		bh.Alive = true

	// Unhandled, always keep alive
	default:
		log.W(ctx, "Command: %v is not handled in VulkanFootprintBuilder", cmd)
		bh.Alive = true
	}

	ft.AddBehaviour(ctx, bh)
	// roll out the recorded reads and writes for queue submit and set event
	switch cmd.(type) {
	case *VkQueueSubmit:
		vb.rollOutExecuted(ctx, ft, executedCommands)
	case *VkSetEvent:
		vb.rollOutExecuted(ctx, ft, executedCommands)
	}
}

func (vb *VulkanFootprintBuilder) writeCoherentMemoryData(ctx context.Context,
	cmd api.Cmd, bh *dependencygraph.Behaviour) {
	if cmd.Extras() == nil || cmd.Extras().Observations() == nil {
		return
	}
	for _, ro := range cmd.Extras().Observations().Reads {
		roStart := ro.Range.Base
		roEnd := ro.Range.End()
		for vkDm, mm := range vb.mappedCoherentMemories {
			if roStart >= mm.MappedLocation.Address() &&
				roEnd < mm.MappedLocation.Address()+uint64(mm.MappedSize) {
				offset := uint64(mm.MappedOffset) + roStart - mm.MappedLocation.Address()
				size := roEnd - roStart
				ms := memorySpan{
					span:   interval.U64Span{Start: offset, End: offset + size},
					memory: vkDm,
				}
				write(ctx, bh, ms)
			}
		}
	}
}

// helper functions
func debug(ctx context.Context, fmt string, args ...interface{}) {
	if config.DebugDeadCodeElimination {
		log.D(ctx, fmt, args...)
	}
}

func read(ctx context.Context, bh *dependencygraph.Behaviour,
	c dependencygraph.DefUseAtom) bool {
	switch c := c.(type) {
	case vkHandle:
		if c == vkNullHandle {
			debug(ctx, "Read to VK_NULL_HANDLE is ignored")
			return false
		}
		// if uint64(c) == uint64(18446744072147626624) {
		// log.E(ctx, "FCI: %v, read image 6624", bh.BelongTo)
		// }
	case *forwardPairedLabel:
		c.labelReadBehaviours = append(c.labelReadBehaviours, bh)
	}
	bh.Read(c)
	debug(ctx, "<Behaviour: %v, Read: %v>", bh, c)
	return true
}

func write(ctx context.Context, bh *dependencygraph.Behaviour,
	c dependencygraph.DefUseAtom) bool {
	switch c := c.(type) {
	case vkHandle:
		if c == vkNullHandle {
			debug(ctx, "Write to VK_NULL_HANDLE is ignored")
			return false
		}
	}
	bh.Write(c)
	debug(ctx, "<Behaviour: %v, Write: %v>", bh, c)
	return true
}

func modify(ctx context.Context, bh *dependencygraph.Behaviour,
	c dependencygraph.DefUseAtom) bool {
	switch c := c.(type) {
	case vkHandle:
		if c == vkNullHandle {
			debug(ctx, "Write to VK_NULL_HANDLE is ignored")
			return false
		}
	}
	bh.Modify(c)
	debug(ctx, "<Behaviour: %v, Modify: %v>", bh, c)
	return true
}

func readMultiple(ctx context.Context, bh *dependencygraph.Behaviour,
	cs []dependencygraph.DefUseAtom) {
	for _, c := range cs {
		read(ctx, bh, c)
	}
}

func modifyMultiple(ctx context.Context, bh *dependencygraph.Behaviour,
	cs []dependencygraph.DefUseAtom) {
	for _, c := range cs {
		modify(ctx, bh, c)
	}
}

func framebufferPortCoveredByClearRect(fb *FramebufferObject, r VkClearRect) bool {
	if r.BaseArrayLayer == uint32(0) &&
		r.LayerCount == fb.Layers &&
		r.Rect.Offset.X == 0 && r.Rect.Offset.Y == 0 &&
		r.Rect.Extent.Width == fb.Width &&
		r.Rect.Extent.Height == fb.Height {
		return true
	}
	return false
}

func clearAttachmentData(ctx context.Context, bh *dependencygraph.Behaviour,
	execInfo *queueExecutionInfo, a VkClearAttachment, rects []VkClearRect) {
	subpass := &execInfo.subpasses[execInfo.subpass.val]
	if a.AspectMask == VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT) ||
		a.AspectMask == VkImageAspectFlags(VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) {
		if subpass.depthStencilAttachment != nil {
			modify(ctx, bh, subpass.depthStencilAttachment.data)
			return
		}
	} else if a.AspectMask == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT|
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) {
		if subpass.depthStencilAttachment != nil {
			overwritten := false
			for _, r := range rects {
				if framebufferPortCoveredByClearRect(execInfo.framebuffer, r) {
					overwritten = true
				}
			}
			if overwritten && subpass.depthStencilAttachment.fullImageData {
				write(ctx, bh, subpass.depthStencilAttachment.data)
				return
			} else {
				modify(ctx, bh, subpass.depthStencilAttachment.data)
				return
			}
		}
	} else {
		if a.ColorAttachment != vkAttachmentUnused {
			overwritten := false
			for _, r := range rects {
				if framebufferPortCoveredByClearRect(execInfo.framebuffer, r) {
					overwritten = true
				}
			}
			att := subpass.colorAttachments[a.ColorAttachment]
			if overwritten && att.fullImageData {
				write(ctx, bh, att.data)
				return
			} else {
				modify(ctx, bh, att.data)
				return
			}
		}
	}
}

func subresourceLayersFullyCoverImage(img *ImageObject, layers VkImageSubresourceLayers,
	offset VkOffset3D, extent VkExtent3D) bool {
	if offset.X != 0 || offset.Y != 0 || offset.Z != 0 {
		return false
	}
	if extent.Width != img.Info.Extent.Width ||
		extent.Height != img.Info.Extent.Height ||
		extent.Depth != img.Info.Extent.Depth {
		return false
	}
	if layers.BaseArrayLayer != uint32(0) {
		return false
	}
	if layers.LayerCount != img.Info.ArrayLayers {
		return false
	}
	// Be conservative, only returns true if both the depth and the stencil
	// bits are set.
	if layers.AspectMask == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT|
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) {
		return true
	}
	// For color images, returns true
	if layers.AspectMask == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT) {
		return true
	}
	return false
}

func subresourceRangeFullyCoverImage(img *ImageObject, rng VkImageSubresourceRange) bool {
	if rng.BaseArrayLayer != 0 || rng.BaseMipLevel != 0 {
		return false
	}
	if rng.LayerCount != img.Info.ArrayLayers || rng.LevelCount != img.Info.MipLevels {
		return false
	}
	// Be conservative, only returns true if both the depth and the stencil bits
	// are set.
	if rng.AspectMask == VkImageAspectFlags(
		VkImageAspectFlagBits_VK_IMAGE_ASPECT_DEPTH_BIT|
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_STENCIL_BIT) ||
		rng.AspectMask == VkImageAspectFlags(
			VkImageAspectFlagBits_VK_IMAGE_ASPECT_COLOR_BIT) {
		return true
	}
	return false
}

func blitFullyCoverImage(img *ImageObject, layers VkImageSubresourceLayers,
	offset1 VkOffset3D, offset2 VkOffset3D) bool {
	if offset1.X == 0 && offset1.Y == 0 && offset1.Z == 0 {
		offset := offset1
		extent := VkExtent3D{
			Width:  uint32(offset2.X - offset1.X),
			Height: uint32(offset2.Y - offset1.Y),
			Depth:  uint32(offset2.Z - offset1.Z),
		}
		return subresourceLayersFullyCoverImage(img, layers, offset, extent)
	} else if offset2.X == 0 && offset2.Y == 0 && offset2.Z == 0 {
		offset := offset2
		extent := VkExtent3D{
			Width:  uint32(offset1.X - offset2.X),
			Height: uint32(offset1.Y - offset2.Y),
			Depth:  uint32(offset1.Z - offset2.Z),
		}
		return subresourceLayersFullyCoverImage(img, layers, offset, extent)
	} else {
		return false
	}
}

func getSubBufferData(bufData memorySpan, offset, size uint64) memorySpan {
	start := offset + bufData.span.Start
	end := bufData.span.End
	if size != vkWholeSize {
		end = start + size
	}
	return memorySpan{
		span:   interval.U64Span{Start: start, End: end},
		memory: bufData.memory,
	}
}
